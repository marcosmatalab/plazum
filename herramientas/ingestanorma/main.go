// Comando ingestanorma: la tuberia de ingesta legal, reejecutable.
//
// QUE HACE, y por que son dos cosas y no una.
//
//  1. INGESTA. Convierte una norma publicada (un ELI del BOE, un CELEX del
//     DOUE) en articulado estructurado con su cita y su URL de fuente, listo
//     para que una persona escriba con ello un paquete.json. NO escribe en
//     paquetes/. Un paquete lo autoriza un humano, siempre, y por eso el
//     borrador que emite -borrador esta deliberadamente incompleto: le faltan
//     el id y la clase e2e de cada obligacion, asi que el linter lo rechaza
//     hasta que alguien lo lea y lo complete. Que no se pueda commitear por
//     descuido es una propiedad, no una molestia.
//
//  2. VIGILANCIA. Al volver a ejecutarla sobre una norma ya ingerida dice QUE
//     HA CAMBIADO: articulos nuevos, modificados y derogados, contra la
//     instantanea anterior. Eso es el mecanismo de vigilancia normativa del
//     producto, y el historial que va dejando es el track record publico (la
//     tabla fecha de la fuente hacia fecha del paquete). Por eso el almacen se
//     disena para eso desde el principio: instantanea para comparar, historial
//     append-only para publicar.
//
// LA FRONTERA LEGAL, que aqui es permisiva y hay que aprovecharla bien. BOE se
// transcribe por el art. 13 TRLPI; DOUE se transcribe por la Decision
// 2011/833/UE, CON ATRIBUCION. Las dos condiciones salen en cada extraccion, en
// los campos licencia_fuente y atribucion, porque una atribucion que hay que
// acordarse de poner mas tarde es una atribucion que se pierde. Solo se ingiere
// de fuente primaria: un espejo de GitHub con licencia MIT no vale, porque la
// licencia de un repositorio no alcanza al texto que quien lo subio no poseia.
// ISO, PCI DSS, SOC 2, TISAX y CIS no se tocan: no tienen ELI y su texto no se
// puede redistribuir.
//
// Esto NO es asesoramiento juridico y la extraccion NO es texto oficial. Lo dice
// tambien la salida.
//
// Uso:
//
//	go run ./herramientas/ingestanorma -eli https://www.boe.es/eli/es/RANGO/AAAA/MM/DD/NUM
//	go run ./herramientas/ingestanorma -id BOE-A-AAAA-NNNNN -articulos 31,33
//	go run ./herramientas/ingestanorma -celex 3AAAARNNNN -json
//	go run ./herramientas/ingestanorma -celex 3AAAARNNNN -borrador > borrador.json
//	go run ./herramientas/ingestanorma -historial
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// Limite de peticiones deliberadamente conservador: son APIs publicas y
	// gratuitas que no nos deben nada. Una peticion cada segundo y medio.
	esperaEntrePeticiones = 1500 * time.Millisecond

	// Agente honesto: quien mire sus logs sabe quien es y para que.
	agente = "dutiq-ingestanorma/1 (ingesta de corpus normativo; contacto en el repositorio del proyecto)"

	// Techo de descarga. El texto consolidado mas largo del BOE anda por los
	// 20 MB; 96 deja sitio de sobra y sigue siendo un techo contra una
	// respuesta hinchada.
	maxDescarga = 96 << 20
)

// anfitrionesPrimarios es la frontera legal escrita como lista de anfitriones.
// Solo fuente primaria: BOE y las dos caras de EUR-Lex (el portal y Cellar, el
// servicio de datos de la Oficina de Publicaciones).
var anfitrionesPrimarios = []string{anfitrionBOE, "boe.es", anfitrionCellar, anfitrionEURLex}

func anfitrionAutorizado(host string) bool {
	h := strings.ToLower(host)
	if i := strings.LastIndex(h, ":"); i > 0 && !strings.Contains(h[i:], "]") {
		h = h[:i]
	}
	for _, a := range anfitrionesPrimarios {
		if h == a {
			return true
		}
	}
	return false
}

func main() {
	var (
		eli         = flag.String("eli", "", "identificador ELI del BOE (https://www.boe.es/eli/es/...)")
		id          = flag.String("id", "", "identificador del BOE (BOE-A-AAAA-NNNNN), si ya lo tienes")
		celex       = flag.String("celex", "", "numero CELEX de EUR-Lex (3AAAARNNNN, o el consolidado con guion)")
		aFecha      = flag.String("fecha", "", "texto vigente a esa fecha (AAAA-MM-DD); solo BOE. Vacio = ultima version")
		articulos   = flag.String("articulos", "", "solo estos, separados por comas: 31,33 o \"Disposicion adicional segunda\"")
		todo        = flag.Bool("todo", false, "incluir tambien preambulo, firma y encabezados de estructura")
		salidaJSON  = flag.Bool("json", false, "la extraccion completa en JSON")
		borrador    = flag.Bool("borrador", false, "un borrador de paquete.json en la salida estandar (incompleto a proposito)")
		historial   = flag.Bool("historial", false, "lo observado hasta hoy, para la pagina de vigilancia")
		dirAlmacen  = flag.String("almacen", "corpus-vigilancia", "directorio del almacen de vigilancia")
		dirCache    = flag.String("cache", ".cache/ingestanorma", "directorio de cache de descargas")
		sinCache    = flag.Bool("sin-cache", false, "ignorar la cache y volver a descargar")
		sinRegistro = flag.Bool("sin-registrar", false, "no tocar el almacen de vigilancia")
	)
	// Un -h que solo lista banderas obliga a leer el codigo para saber que hace
	// la herramienta. Quien la abre por primera vez es quien autora el corpus,
	// no quien la escribio.
	flag.Usage = func() {
		w := flag.CommandLine.Output()
		fmt.Fprintln(w, "ingestanorma: de una norma publicada a articulado con su cita, y luego")
		fmt.Fprintln(w, "a que ha cambiado desde la ultima vez.")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "  ingestanorma -eli https://"+anfitrionBOE+"/eli/es/RANGO/AAAA/MM/DD/NUM")
		fmt.Fprintln(w, "        lista el articulado de una norma del BOE y guarda la instantanea")
		fmt.Fprintln(w, "  ingestanorma -celex 3AAAARNNNN")
		fmt.Fprintln(w, "        lo mismo para una norma del DOUE")
		fmt.Fprintln(w, "  ingestanorma -eli ... -borrador > paquete.json")
		fmt.Fprintln(w, "        un borrador de paquete de corpus, incompleto a proposito:")
		fmt.Fprintln(w, "        le faltan el id y la clase e2e de cada obligacion, que las")
		fmt.Fprintln(w, "        escribe una persona. Hasta entonces el linter lo rechaza")
		fmt.Fprintln(w, "  ingestanorma -historial")
		fmt.Fprintln(w, "        lo observado hasta hoy, para la pagina de vigilancia")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Solo se descarga de fuente primaria (BOE y EUR-Lex). No es asesoramiento")
		fmt.Fprintln(w, "juridico ni texto oficial. Detalle en herramientas/ingestanorma/LEEME.md.")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Banderas:")
		flag.PrintDefaults()
	}
	flag.Parse()

	if err := ejecutar(opciones{
		ELI: *eli, ID: *id, CELEX: *celex, AFecha: *aFecha, Articulos: *articulos,
		Todo: *todo, JSON: *salidaJSON, Borrador: *borrador, Historial: *historial,
		Almacen: *dirAlmacen, Cache: *dirCache, SinCache: *sinCache, SinRegistro: *sinRegistro,
	}, os.Stdout, time.Now); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

type opciones struct {
	ELI, ID, CELEX     string
	AFecha, Articulos  string
	Todo, JSON         bool
	Borrador           bool
	Historial          bool
	Almacen, Cache     string
	SinCache           bool
	SinRegistro        bool
	ClienteParaPruebas *cliente // solo en tests: evita salir a la red
}

// ejecutar es main sin os.Exit, para poder probarlo entero.
func ejecutar(op opciones, salida io.Writer, reloj func() time.Time) error {
	alm := Almacen{Dir: op.Almacen}
	if op.Historial {
		return imprimirHistorial(alm, salida, op.JSON)
	}
	fuentes := 0
	for _, s := range []string{op.ELI, op.ID, op.CELEX} {
		if strings.TrimSpace(s) != "" {
			fuentes++
		}
	}
	if fuentes != 1 {
		return fmt.Errorf("hay que dar exactamente una norma y se han dado %d.\n"+
			"  -eli    https://%s/eli/es/RANGO/AAAA/MM/DD/NUMERO   (lo que sale en la ficha del BOE)\n"+
			"  -id     BOE-A-AAAA-NNNNN                            (si ya tienes el identificador)\n"+
			"  -celex  3AAAARNNNN                                  (el CELEX de la ficha de EUR-Lex)\n"+
			"  -historial                                          (lo observado hasta hoy)",
			fuentes, anfitrionBOE)
	}
	if op.AFecha != "" && op.CELEX != "" {
		return errors.New("-fecha solo vale para el BOE, que sirve todas las versiones de cada " +
			"articulo.\nArreglo para el DOUE: pide el CELEX consolidado a esa fecha, que tiene " +
			"la forma 0AAAARNNNN-AAAAMMDD")
	}
	referencia, err := fechaReferencia(op.AFecha, reloj())
	if err != nil {
		return err
	}

	cli := op.ClienteParaPruebas
	if cli == nil {
		cli = nuevoCliente(op.Cache, op.SinCache)
	}
	ahora := reloj().UTC()

	var ext *Extraccion
	if op.CELEX != "" {
		ext, err = ingerirCELEX(cli, op.CELEX, ahora)
	} else {
		ext, err = ingerirBOE(cli, op.ELI, op.ID, referencia, op.AFecha, op.Todo, ahora)
	}
	if err != nil {
		return err
	}

	// El filtro se aplica DESPUES de calcular la huella de la norma entera: la
	// vigilancia mira la norma, no el trozo que a uno le interesaba hoy.
	parcial := false
	if op.Articulos != "" {
		ext.Articulos, err = filtrar(ext.Articulos, op.Articulos)
		if err != nil {
			return err
		}
		parcial = true
	}

	// Registrar una extraccion PARCIAL o de una fecha pasada envenenaria el
	// almacen: la siguiente ejecucion completa veria media norma nueva, o la
	// veria retroceder en el tiempo. Se avisa y no se registra.
	motivoSinRegistro := ""
	switch {
	case op.SinRegistro:
		motivoSinRegistro = "se pidio -sin-registrar"
	case parcial:
		motivoSinRegistro = "la extraccion es parcial (-articulos): registrarla haria que la " +
			"proxima ejecucion completa viera el resto de la norma como articulos nuevos"
	case op.AFecha != "":
		motivoSinRegistro = "se pidio el texto a una fecha pasada (-fecha): registrarlo haria " +
			"retroceder la vigilancia"
	}

	var cambios Cambios
	if motivoSinRegistro == "" {
		k := clave(ext.Fuente)
		anterior, err := alm.Anterior(k)
		if err != nil {
			return err
		}
		cambios = Comparar(anterior, ext)
		if err := alm.Registrar(k, ext, cambios, ahora); err != nil {
			return err
		}
	}

	switch {
	case op.JSON:
		return escribirJSON(salida, ext)
	case op.Borrador:
		return escribirJSON(salida, borradorDe(ext))
	default:
		imprimirTabla(salida, ext, cambios, motivoSinRegistro)
		return nil
	}
}

// fechaReferencia decide a que fecha se quiere el texto. Nunca vacia: sin fecha
// de referencia no se puede decidir si un bloque caducado sigue mostrandose.
func fechaReferencia(aFecha string, ahora time.Time) (string, error) {
	if aFecha == "" {
		return ahora.UTC().Format("20060102"), nil
	}
	t, err := time.Parse("2006-01-02", aFecha)
	if err != nil {
		return "", fmt.Errorf("-fecha %q no es una fecha: se espera AAAA-MM-DD", aFecha)
	}
	return t.Format("20060102"), nil
}

// --- ingesta BOE ---

func ingerirBOE(cli *cliente, eli, id, referencia, aFecha string, todo bool, ahora time.Time) (*Extraccion, error) {
	var piezas piezasELI
	if eli != "" {
		p, err := partirELI(eli)
		if err != nil {
			return nil, err
		}
		piezas = p
		cuerpo, err := cli.obtener(urlBusquedaPorFecha(p), cabecerasBOE)
		if err != nil {
			return nil, err
		}
		id, err = resolverELIBOE(cuerpo, p)
		if err != nil {
			return nil, err
		}
	}
	id = strings.ToUpper(strings.TrimSpace(id))
	if err := validarIDBOE(id); err != nil {
		return nil, err
	}

	base := baseAPIBOE + "/id/" + url.PathEscape(id)
	crudoMeta, err := cli.obtener(base+"/metadatos", cabecerasBOE)
	if err != nil {
		return nil, err
	}
	var meta respuestaMetadatosBOE
	if err := decodificarBOE(crudoMeta, &meta); err != nil {
		return nil, err
	}
	if piezas.Base == "" && meta.Meta.URLELI != "" {
		if p, err := partirELI(meta.Meta.URLELI); err == nil {
			piezas = p
		}
	}
	urlDatos := base + "/texto"
	crudoTexto, err := cli.obtener(urlDatos, cabecerasBOE)
	if err != nil {
		return nil, err
	}
	urlDoc := meta.Meta.URLHTML
	if piezas.Base != "" {
		urlDoc = piezas.Base + "/con" // el texto consolidado, que es lo que se cita
	}
	arts, err := parsearTextoBOE(crudoTexto, meta, urlDoc, opcionesBOE{Referencia: referencia, Todo: todo})
	if err != nil {
		return nil, err
	}
	origen := origenDeMetadatos(meta, piezas, urlDoc, aFecha)
	origen.URLDatos = urlDatos
	return armar(origen, LicenciaBOE, AtribucionBOE, arts, ahora), nil
}

// validarIDBOE comprueba la forma antes de meter el identificador en una ruta de
// URL. Viene de fuera, y un identificador con barras dentro sale del camino de la
// API y pide otra cosa.
func validarIDBOE(id string) error {
	if len(id) < 10 || len(id) > 30 || !strings.HasPrefix(id, "BOE-") {
		return fmt.Errorf("%w: %q no tiene forma de identificador del BOE. "+
			"Arreglo: usa la forma BOE-A-AAAA-NNNNN, o pasa el ELI con -eli y se resuelve solo",
			ErrIdentificadorInvalido, id)
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
			return fmt.Errorf("%w: %q lleva el caracter %q, que no cabe en un identificador "+
				"del BOE", ErrIdentificadorInvalido, id, string(c))
		}
	}
	return nil
}

var cabecerasBOE = map[string]string{"Accept": "application/xml"}

// --- ingesta EUR-Lex ---

var (
	cabecerasFicha = map[string]string{
		"Accept": "application/xml;notice=object", "Accept-Language": "spa",
	}
	cabecerasTexto = map[string]string{
		"Accept": "application/xhtml+xml", "Accept-Language": "spa",
	}
)

func ingerirCELEX(cli *cliente, celex string, ahora time.Time) (*Extraccion, error) {
	c, err := validarCELEX(celex)
	if err != nil {
		return nil, err
	}
	u := baseCellar + url.PathEscape(c)
	ficha, err := cli.obtener(u, cabecerasFicha)
	if err != nil {
		return nil, err
	}
	titulo, eli, actualizada, err := parsearNoticiaCellar(ficha)
	if err != nil {
		return nil, err
	}
	texto, err := cli.obtener(u, cabecerasTexto)
	if err != nil {
		return nil, err
	}
	urlDoc := eli
	if urlDoc == "" {
		urlDoc = "https://" + anfitrionEURLex + "/legal-content/ES/TXT/?uri=CELEX:" + c
	}
	arts, err := parsearXHTMLEurLex(texto, titulo, urlDoc)
	if err != nil {
		return nil, err
	}
	origen := origenDeCELEX(c, titulo, eli, actualizada, urlDoc)
	origen.URLDatos = u
	return armar(origen, LicenciaDOUE, AtribucionDOUE, arts, ahora), nil
}

// --- armado y salidas ---

func armar(o Origen, licencia, atribucion string, arts []Articulo, ahora time.Time) *Extraccion {
	return &Extraccion{
		Esquema:        EsquemaIngesta,
		Fuente:         o,
		LicenciaFuente: licencia,
		Atribucion:     atribucion,
		Obtenido:       ahora.UTC().Format(time.RFC3339),
		Huella:         HuellaDeExtraccion(arts),
		Articulos:      arts,
	}
}

// filtrar deja solo los articulos pedidos. Un termino que no casa con nada es un
// error y no un silencio: pedir el articulo 310 de una norma de 41 y que salga
// una lista vacia es la forma de creerse que un articulo no existe.
func filtrar(as []Articulo, lista string) ([]Articulo, error) {
	quiero := map[string]bool{}
	for _, t := range strings.Split(lista, ",") {
		if t = strings.TrimSpace(t); t != "" {
			quiero[strings.ToLower(t)] = true
		}
	}
	var out []Articulo
	casados := map[string]bool{}
	for _, a := range as {
		for t := range quiero {
			if strings.EqualFold(a.Referencia, t) || strings.EqualFold(a.Numero, t) ||
				strings.EqualFold(a.ID, t) {
				out = append(out, a)
				casados[t] = true
				break
			}
		}
	}
	var faltan []string
	for t := range quiero {
		if !casados[t] {
			faltan = append(faltan, t)
		}
	}
	if len(faltan) > 0 {
		sort.Strings(faltan)
		return nil, fmt.Errorf("no hay ningun articulo que case con %s. "+
			"Arreglo: ejecuta sin -articulos para ver la lista de referencias tal cual las "+
			"publica la fuente, y copia una de ahi", strings.Join(faltan, ", "))
	}
	return out, nil
}

func escribirJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func imprimirHistorial(a Almacen, w io.Writer, comoJSON bool) error {
	entradas, err := a.Historial()
	if err != nil {
		return err
	}
	if comoJSON {
		if entradas == nil {
			entradas = []Entrada{}
		}
		return escribirJSON(w, entradas)
	}
	if len(entradas) == 0 {
		fmt.Fprintf(w, "el almacen %s esta vacio: no se ha ingerido ninguna norma todavia\n", a.Dir)
		return nil
	}
	fmt.Fprintf(w, "vigilancia normativa: %d observaciones en %s\n\n", len(entradas), a.Dir)
	fmt.Fprintf(w, "%-20s %-22s %-12s %s\n", "observado", "norma", "cambios", "fuente actualizada")
	for _, e := range entradas {
		fmt.Fprintf(w, "%-20s %-22s %-12s %s\n",
			recortar(e.Observado, 20), recortar(e.Identificador, 22),
			resumenCambios(e.Cambios), e.Cambios.FuenteAhora)
	}
	fmt.Fprintln(w, "\nCada linea es una observacion, no una publicacion: dice cuando lo vio esta")
	fmt.Fprintln(w, "herramienta, no cuando se actualizo el paquete.")
	return nil
}

func resumenCambios(c Cambios) string {
	if c.PrimeraVez {
		return "primera vez"
	}
	if !c.Hay() {
		return "sin cambios"
	}
	return fmt.Sprintf("+%d ~%d -%d", len(c.Nuevos), len(c.Modificados), len(c.Derogados))
}

func imprimirTabla(w io.Writer, e *Extraccion, c Cambios, sinRegistro string) {
	f := e.Fuente
	fmt.Fprintf(w, "== %s\n", recortar(f.Titulo, 100))
	fmt.Fprintf(w, "   identificador  %s\n", f.Identificador)
	if f.ELI != "" {
		fmt.Fprintf(w, "   eli            %s\n", f.ELI)
	}
	fmt.Fprintf(w, "   texto citable  %s\n", f.URLDocumento)
	fmt.Fprintf(w, "   datos          %s\n", f.URLDatos)
	if f.URNSugerido != "" {
		fmt.Fprintf(w, "   urn sugerido   %s\n", f.URNSugerido)
	}
	if f.FechaVigencia != "" {
		fmt.Fprintf(w, "   en vigor desde %s\n", f.FechaVigencia)
	}
	if f.ActualizadaEn != "" {
		fmt.Fprintf(w, "   la fuente la actualizo por ultima vez el %s\n", f.ActualizadaEn)
	}
	if f.Derogada {
		fmt.Fprintf(w, "   DEROGADA el %s: la norma entera ya no esta en vigor\n", f.FechaDerogacion)
	}
	if f.TextoVigenteEn != "" {
		fmt.Fprintf(w, "   texto vigente a %s (no es la ultima version)\n", f.TextoVigenteEn)
	}
	derogados := 0
	for _, a := range e.Articulos {
		if a.Derogado {
			derogados++
		}
	}
	fmt.Fprintf(w, "   %d articulos, %d derogados\n\n", len(e.Articulos), derogados)

	fmt.Fprintln(w, "   VIGILANCIA")
	switch {
	case sinRegistro != "":
		fmt.Fprintf(w, "      no se compara ni se registra: %s\n", sinRegistro)
	case c.PrimeraVez:
		fmt.Fprintln(w, "      primera vez: no habia instantanea anterior, no hay nada que comparar.")
		fmt.Fprintln(w, "      Vuelve a ejecutar esto dentro de un tiempo y dira que ha cambiado.")
	case !c.Hay():
		fmt.Fprintf(w, "      sin cambios: los %d articulos estan igual que la ultima vez\n", c.SinCambio)
	default:
		fmt.Fprintf(w, "      %d nuevos, %d modificados, %d derogados, %d sin cambio\n",
			len(c.Nuevos), len(c.Modificados), len(c.Derogados), c.SinCambio)
		for _, r := range c.Nuevos {
			fmt.Fprintf(w, "      NUEVO       %s\n", r)
		}
		for _, r := range c.Modificados {
			fmt.Fprintf(w, "      MODIFICADO  %s%s\n", r, porQuien(e, r))
		}
		for _, r := range c.Derogados {
			fmt.Fprintf(w, "      DEROGADO    %s%s\n", r, porQuien(e, r))
		}
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "   ARTICULADO")
	for _, a := range e.Articulos {
		marca := " "
		if a.Derogado {
			marca = "x"
		}
		fmt.Fprintf(w, "   %s %-34s %-11s %s\n", marca, recortar(a.Referencia, 34),
			a.VigenciaDesde, recortar(a.Titulo, 60))
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "   LICENCIA DE LA FUENTE")
	fmt.Fprintf(w, "      %s\n", e.LicenciaFuente)
	fmt.Fprintln(w, "   ATRIBUCION (obligatoria, tiene que viajar con el texto)")
	fmt.Fprintf(w, "      %s\n", e.Atribucion)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Esto no es asesoramiento juridico ni texto oficial. La extraccion es material")
	fmt.Fprintln(w, "de trabajo: un paquete de corpus lo autoriza y lo firma una persona.")
}

// porQuien anade, si se sabe, la norma que toco el articulo.
func porQuien(e *Extraccion, ref string) string {
	for _, a := range e.Articulos {
		if a.Referencia == ref && a.ModificadoPor != "" {
			return fmt.Sprintf("  (por %s, en vigor %s)", a.ModificadoPor, a.VigenciaDesde)
		}
	}
	return ""
}

func recortar(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "."
}

// --- el cliente HTTP ---

type cliente struct {
	http       *http.Client
	cache      string
	sinCache   bool
	ultima     time.Time
	Peticiones int
	DeCache    int
	// Local, si no es nil, sustituye a la red. Solo lo usan los tests: la
	// herramienta no sale a internet en ninguna prueba.
	Local func(url string, cab map[string]string) ([]byte, error)
}

func nuevoCliente(cache string, sinCache bool) *cliente {
	return &cliente{
		cache:    cache,
		sinCache: sinCache,
		http: &http.Client{
			Timeout: 120 * time.Second,
			// Un redirect a un anfitrion que no es fuente primaria se corta.
			// La frontera legal no vale de nada si se puede saltar contestando
			// con un 302.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return errors.New("demasiados redirects")
				}
				if !anfitrionAutorizado(req.URL.Host) {
					return fmt.Errorf("%w: la fuente redirige a %q", ErrFuenteNoAutorizada, req.URL.Host)
				}
				return nil
			},
		},
	}
}

func (c *cliente) obtener(u string, cab map[string]string) ([]byte, error) {
	if c.Local != nil {
		return c.Local(u, cab)
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("%w: %q no es una URL", ErrIdentificadorInvalido, u)
	}
	if !anfitrionAutorizado(parsed.Host) {
		return nil, fmt.Errorf("%w: %q. Solo se descarga de fuente primaria (%s)",
			ErrFuenteNoAutorizada, parsed.Host, strings.Join(anfitrionesPrimarios, ", "))
	}
	llave := c.rutaCache(u, cab)
	if !c.sinCache {
		if b, err := os.ReadFile(llave); err == nil { // #nosec G304 -- ruta derivada de un hash
			c.DeCache++
			return b, nil
		}
	}
	if !c.ultima.IsZero() {
		if falta := esperaEntrePeticiones - time.Since(c.ultima); falta > 0 {
			time.Sleep(falta)
		}
	}
	c.ultima = time.Now()
	c.Peticiones++

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", agente)
	for k, v := range cab {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("no responde %s: %w. Arreglo: comprueba la conexion y vuelve a "+
			"intentarlo; lo ya descargado sigue en la cache", parsed.Host, err)
	}
	defer func() { _ = resp.Body.Close() }()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, fmt.Errorf("%w: %s responde 404 para %s", ErrNormaNoEncontrada, parsed.Host, u)
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("%s responde HTTP %d para %s. Arreglo: si es 403, la fuente ha "+
			"empezado a filtrar por cabecera; si es 5xx, reintenta mas tarde",
			parsed.Host, resp.StatusCode, u)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxDescarga))
	if err != nil {
		return nil, err
	}
	c.guardarCache(llave, b)
	return b, nil
}

func (c *cliente) rutaCache(u string, cab map[string]string) string {
	h := sha256.New()
	_, _ = h.Write([]byte(u))
	claves := make([]string, 0, len(cab))
	for k := range cab {
		claves = append(claves, k)
	}
	sort.Strings(claves)
	for _, k := range claves {
		_, _ = h.Write([]byte("\x00" + k + "=" + cab[k]))
	}
	return filepath.Join(c.cache, hex.EncodeToString(h.Sum(nil))+".bin")
}

func (c *cliente) guardarCache(ruta string, b []byte) {
	if err := os.MkdirAll(filepath.Dir(ruta), 0o750); err != nil {
		return // la cache es una optimizacion; si falla, se sigue sin ella
	}
	_ = os.WriteFile(ruta, b, 0o600)
}
