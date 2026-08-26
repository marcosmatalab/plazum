// Comando cribamarca: criba de anterioridades de marca contra TMview.
//
// Por que existe, y por que mira en tres direcciones.
//
// El fallo que nos costo la marca no fue no buscar. Fue buscar mal: se
// comprueba si el candidato colisiona con algo, y nadie comprueba si el
// candidato CONTIENE una marca ajena. DUTIQ contiene UTIQ entero, y UTIQ es
// una marca denominativa registrada por la empresa conjunta de cuatro
// operadoras, en las mismas clases 9 y 42 que necesitamos. Una busqueda por
// igualdad no lo habria encontrado nunca.
//
// De ahi las tres lentes, que son tres preguntas distintas:
//
//  1. COLISION: existe una marca que se llame igual que el candidato.
//     Es la que todo el mundo hace.
//  2. CONTENEDORAS: existe una marca registrada que contenga al candidato.
//     "dutiq" dentro de "midutiqpro". Menos comun, pero se mira.
//  3. CONTENIDAS: el candidato contiene una marca registrada.
//     "utiq" dentro de "dutiq". ESTA es la que nadie mira y la que nos mordio.
//
// La tercera es la cara: hay que probar cada subcadena del candidato contra la
// base, no una sola consulta. Por eso hay cache en disco y un rate limit
// conservador: la API es publica y gratuita y no se le martillea.
//
// LO QUE ESTA HERRAMIENTA NO MIRA, y costo descartar un finalista el 26-08-2026:
// TMview es un registro de MARCAS. No sabe nada de empresas en activo que
// operan sin registrar, y el uso anterior no registrado crea derechos en
// varias jurisdicciones. "Deontia" salio limpia en EUIPO y en OEPM, y existe
// Deontic (deontic.ai, Lovaina, 2022), plataforma de IA para cumplimiento
// regulatorio: mismo sector y una letra de diferencia. El registro estaba
// limpio y el mercado no.
//
// Asi que la criba tiene un paso MANUAL obligatorio despues, y la salida lo
// dice: buscar el finalista como nombre de empresa y como dominio. No se
// automatiza porque no hay fuente gratuita y fiable de razones sociales de la
// Union, y una automatizacion a medias aqui daria justo el falso verde que
// esta herramienta existe para no dar.
//
// Esto NO es un dictamen. Una criba automatica reduce la sorpresa, no sustituye
// a un agente de la propiedad industrial. Lo dice tambien la salida.
//
// Uso:
//
//	go run ./herramientas/cribamarca -candidatos dutiq,otronombre
//	go run ./herramientas/cribamarca -candidatos dutiq -clases 9,42 -json
package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	urlTMview = "https://www.tmdn.org/tmview/api/search/results"

	// TMview rechaza por cabecera, no por red. Sin un user-agent de navegador
	// la conexion ni siquiera se establece, y eso me llevo a concluir por error
	// que la base era inalcanzable. Queda escrito para que no se repita.
	agente = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 " +
		"(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	// Rate limit deliberadamente conservador: es una API publica y gratuita
	// que no nos debe nada. Una consulta cada segundo y medio.
	esperaEntreConsultas = 1500 * time.Millisecond

	// Subcadenas mas cortas que esto generan ruido inservible: "ut", "iq".
	minSubcadena = 3

	// Tamano de pagina. NO SUBIR SIN MIRAR EL CANARIO: con 200 la API contesta
	// HTTP 200 con la lista vacia, sin error y sin aviso. 50 y 100 funcionan.
	paginaTamano = 50

	// Tope de paginas por termino. Mil anterioridades para una subcadena no es
	// un resultado que cribar, es un termino que no distingue nada.
	maxPaginas = 20

	// El canario: un termino con cientos de anterioridades vivas en EUIPO y en
	// OEPM, que por eso no puede volver vacio si el transporte funciona. Se
	// eligio despues de comprobar que devuelve pagina llena en las dos oficinas.
	terminoCanario = "tec"
)

// Umbrales del semaforo, y por que hubo que ponerlos.
//
// La primera version pintaba ROJO en cuanto aparecia una marca contenida, de
// la longitud que fuera. Con minSubcadena=3 y millones de marcas de la Union,
// TODO candidato lleva dentro algun acronimo de tres letras registrado en la
// clase 9: VEN, ENC, NCI, CIA, REC, ECE, CEP, EPT. El semaforo decia ROJO
// siempre, y un semaforo que siempre dice lo mismo no dice nada. Es la misma
// familia que "una puerta que nunca se ha visto fallar no es una puerta",
// vista del reves: una que salta siempre tampoco guarda.
//
// Lo que separo a DUTIQ de una casualidad no fue que UTIQ existiera, fue
// CUANTO de DUTIQ era UTIQ: cuatro letras de cinco, el 80%. Un acronimo de
// tres letras dentro de un nombre de nueve es el 33% y no se parece a nada.
// Asi que lo que pesa es la COBERTURA, no la presencia.
const (
	// Por debajo de cuatro letras es acronimo. Se cuenta y se dice, no se
	// pinta. Con -todo se listan igualmente: aqui no se tira nada en silencio.
	minRelevante = 4

	// Lente 3, la marca va DENTRO del candidato: cobertura sobre el candidato.
	// UTIQ en DUTIQ = 0,80. PRECEPT en PRECEPTUM = 0,78.
	cobRojaDentro  = 0.60
	cobAmbarDentro = 0.50

	// Lente 2, el candidato va DENTRO de la marca: cobertura sobre la marca.
	// VENCIA en AVENCIA = 0,86, que a efectos practicos es identidad.
	cobRojaFuera  = 0.70
	cobAmbarFuera = 0.50
)

// Los tres niveles del semaforo, por marca encontrada.
const (
	nivelRojo  = "rojo"
	nivelAmbar = "ambar"
	nivelRuido = "ruido"
)

// marca es lo que interesa de un registro de TMview.
type marca struct {
	Numero      string `json:"numero"`
	Nombre      string `json:"nombre"`
	Tipo        string `json:"tipo"`
	Estado      string `json:"estado"`
	Clases      []int  `json:"clases"`
	Titular     string `json:"titular"`
	Registrada  string `json:"registrada,omitempty"`
	Solicitada  string `json:"solicitada,omitempty"`
	Expira      string `json:"expira,omitempty"`
	Oficina     string `json:"oficina"`
	URLDetalle  string `json:"url_detalle,omitempty"`
	Coincidente string `json:"coincidente,omitempty"` // que subcadena la trajo
	// Cobertura: que fraccion del signo corto ocupa la coincidencia. Es lo que
	// separa "UTIQ dentro de DUTIQ" de "CIA dentro de VENCIA".
	Cobertura float64 `json:"cobertura"`
	Nivel     string  `json:"nivel"` // rojo, ambar o ruido
}

// vigente dice si la marca puede oponerse a algo. Una caducada no.
func (m marca) vigente() bool {
	switch strings.ToLower(m.Estado) {
	case "registered", "filed", "opposition pending", "under examination":
		return true
	}
	return false
}

func (m marca) tocaClases(clases []int) bool {
	if len(clases) == 0 {
		return true
	}
	for _, c := range m.Clases {
		for _, q := range clases {
			if c == q {
				return true
			}
		}
	}
	return false
}

// Hallazgo agrupa lo que se encontro para un candidato, por lente.
type Hallazgo struct {
	Candidato    string  `json:"candidato"`
	Colisiones   []marca `json:"colisiones"`
	Contenedoras []marca `json:"contenedoras"`
	Contenidas   []marca `json:"contenidas"`
	Consultas    int     `json:"consultas_hechas"`
	DesdeCache   int     `json:"consultas_desde_cache"`
}

// nivelPorCobertura traduce cobertura a semaforo. Ruido cuando la coincidencia
// es demasiado corta o demasiado parcial para que nadie confunda los signos.
func nivelPorCobertura(letras int, cob, roja, ambar float64) string {
	switch {
	case letras < minRelevante:
		return nivelRuido
	case cob >= roja:
		return nivelRojo
	case cob >= ambar:
		return nivelAmbar
	}
	return nivelRuido
}

// clasificar rellena cobertura y nivel de cada marca encontrada.
//
// Va aparte de la busqueda a proposito: el juicio es una funcion pura y se
// prueba entera sin salir a la red, que es donde estaba el fallo del semaforo.
func clasificar(h *Hallazgo) {
	cand := float64(len(h.Candidato))
	if cand == 0 {
		return
	}
	for i := range h.Colisiones {
		h.Colisiones[i].Cobertura = 1
		h.Colisiones[i].Nivel = nivelRojo
	}
	for i := range h.Contenedoras {
		m := &h.Contenedoras[i]
		largo := len(m.Nombre)
		if largo == 0 {
			largo = len(h.Candidato)
		}
		m.Cobertura = cand / float64(largo)
		m.Nivel = nivelPorCobertura(len(h.Candidato), m.Cobertura, cobRojaFuera, cobAmbarFuera)
	}
	for i := range h.Contenidas {
		m := &h.Contenidas[i]
		m.Cobertura = float64(len(m.Coincidente)) / cand
		m.Nivel = nivelPorCobertura(len(m.Coincidente), m.Cobertura, cobRojaDentro, cobAmbarDentro)
	}
}

// Riesgo resume en una palabra, para la tabla. No es un dictamen.
func (h Hallazgo) Riesgo() string {
	ambar := false
	for _, ms := range [][]marca{h.Colisiones, h.Contenedoras, h.Contenidas} {
		for _, m := range ms {
			switch m.Nivel {
			case nivelRojo:
				return "ROJO"
			case nivelAmbar:
				ambar = true
			}
		}
	}
	if ambar {
		return "AMBAR"
	}
	return "sin hallazgos"
}

// verTodo lo pone la bandera -todo: lista tambien el ruido, para poder
// auditar el umbral en vez de creerselo.
var verTodo bool

func main() {
	os.Exit(ejecutar(os.Args[1:], os.Stdout, os.Stderr))
}

// ejecutar es main sin os.Exit ni globales, para que se pueda probar.
//
// POR QUE SE PARTE ASI, y no es gusto: la unica pieza de este programa que
// decide algo es el ORDEN, y el orden vive aqui. El canario va antes de cribar
// nada; el que salga "sin hallazgos" en vez de un error depende de que esa
// linea siga estando donde esta. Con main() llamando a os.Exit no hay forma de
// escribir un test que lo compruebe, y una herramienta cuyo unico producto es
// una prueba no se puede permitir que su parte no comprobable sea justo esa.
//
// La salida y los errores entran como io.Writer por el mismo motivo. Es la
// misma forma que usa cmd/plazum con cmdDoctor.
func ejecutar(args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("cribamarca", flag.ContinueOnError)
	fs.SetOutput(errores)
	var (
		candidatos = fs.String("candidatos", "", "lista separada por comas de nombres a cribar")
		clasesTxt  = fs.String("clases", "9,42", "clases Niza que importan, separadas por comas; vacio = todas")
		oficina    = fs.String("oficina", "EM", "oficina TMview: EM = EUIPO, ES = OEPM")
		dirCache   = fs.String("cache", ".cache/cribamarca", "directorio de cache en disco")
		salidaJSON = fs.Bool("json", false, "salida en JSON en vez de tabla")
		sinCache   = fs.Bool("sin-cache", false, "ignorar la cache y volver a consultar")
		todo       = fs.Bool("todo", false, "listar tambien lo que queda por debajo del umbral")
		base       = fs.String("extremo", "", "extremo alternativo de TMview; solo para pruebas")
	)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	verTodo = *todo

	if strings.TrimSpace(*candidatos) == "" {
		fmt.Fprintln(errores, "falta -candidatos: los nombres que quieres cribar, separados por comas")
		fmt.Fprintln(errores, "ejemplo: go run ./herramientas/cribamarca -candidatos dutiq,otronombre")
		return 2
	}
	clases, err := parsearClases(*clasesTxt)
	if err != nil {
		fmt.Fprintln(errores, "error:", err)
		return 2
	}

	c := &criba{
		oficina:  *oficina,
		clases:   clases,
		cache:    *dirCache,
		sinCache: *sinCache,
		base:     *base,
		http:     &http.Client{Timeout: 45 * time.Second},
	}
	// El rate limit protege a TMview, que es una API publica y gratuita que no
	// nos debe nada. Un extremo alternativo NO ES TMview por definicion, asi
	// que ahi no hay a quien proteger y esperar segundo y medio por consulta
	// solo hace que una suite tarde un minuto en hablar con localhost.
	if *base != "" {
		c.espera = time.Millisecond
	}

	// El canario, ANTES de cribar nada. Si el transporte no funciona, esta
	// herramienta no imprime una tabla mas floja: no imprime tabla. Una criba
	// que sale "sin hallazgos" porque no llego a preguntar es peor que no
	// cribar, porque se decide un nombre con ella.
	if err := c.comprobarTransporte(); err != nil {
		fmt.Fprintln(errores, "error:", err)
		return 1
	}

	var todos []Hallazgo
	for _, cand := range strings.Split(*candidatos, ",") {
		cand = strings.ToLower(strings.TrimSpace(cand))
		if cand == "" {
			continue
		}
		h, err := c.cribar(cand)
		if err != nil {
			fmt.Fprintf(errores, "error cribando %q: %v\n", cand, err)
			return 1
		}
		todos = append(todos, h)
	}

	if *salidaJSON {
		enc := json.NewEncoder(salida)
		enc.SetIndent("", "  ")
		if err := enc.Encode(todos); err != nil {
			fmt.Fprintln(errores, "error:", err)
			return 1
		}
		return 0
	}
	imprimirTabla(salida, todos, clases, *oficina)
	return 0
}

func parsearClases(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	var out []int
	for _, p := range strings.Split(s, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return nil, fmt.Errorf("clase %q no es un numero", p)
		}
		if n < 1 || n > 45 {
			return nil, fmt.Errorf("la clase %d no existe: las de Niza van de 1 a 45", n)
		}
		out = append(out, n)
	}
	return out, nil
}

type criba struct {
	oficina   string
	base      string        // extremo alternativo, solo para pruebas
	espera    time.Duration // rate limit; 0 = el de produccion
	clases    []int
	cache     string
	sinCache  bool
	http      *http.Client
	ultima    time.Time
	consultas int
	deCache   int
}

// cribar pasa un candidato por las tres lentes.
func (c *criba) cribar(cand string) (Hallazgo, error) {
	h := Hallazgo{Candidato: cand}
	c.consultas, c.deCache = 0, 0

	// Lentes 1 y 2 con una sola consulta: se busca el candidato y se separa
	// lo que se llama igual de lo que lo contiene.
	ms, err := c.buscar(cand)
	if err != nil {
		return h, err
	}
	for _, m := range ms {
		if !m.vigente() || !m.tocaClases(c.clases) {
			continue
		}
		n := strings.ToLower(m.Nombre)
		switch {
		case n == cand:
			m.Coincidente = cand
			h.Colisiones = append(h.Colisiones, m)
		case strings.Contains(n, cand):
			m.Coincidente = cand
			h.Contenedoras = append(h.Contenedoras, m)
		}
	}

	// Lente 3, la que nadie mira: cada subcadena del candidato, de la mas
	// larga a la mas corta, buscada como marca propia. Si alguna existe y esta
	// viva en nuestras clases, el candidato la lleva dentro.
	vistas := map[string]bool{}
	for _, sub := range subcadenas(cand) {
		ms, err := c.buscar(sub)
		if err != nil {
			return h, err
		}
		for _, m := range ms {
			if !m.vigente() || !m.tocaClases(c.clases) {
				continue
			}
			if strings.ToLower(m.Nombre) != sub {
				continue // solo coincidencia exacta con la subcadena
			}
			if vistas[m.Numero] {
				continue
			}
			vistas[m.Numero] = true
			m.Coincidente = sub
			h.Contenidas = append(h.Contenidas, m)
		}
	}
	// De la subcadena mas larga a la mas corta: una marca mas larga dentro del
	// candidato preocupa mas que una de tres letras.
	sort.SliceStable(h.Contenidas, func(i, j int) bool {
		return len(h.Contenidas[i].Coincidente) > len(h.Contenidas[j].Coincidente)
	})

	h.Consultas, h.DesdeCache = c.consultas, c.deCache
	clasificar(&h)
	return h, nil
}

// subcadenas devuelve las subcadenas propias del candidato, de larga a corta.
// Propias: se excluye el candidato entero, que ya cubre la lente 1.
func subcadenas(s string) []string {
	var out []string
	for l := len(s) - 1; l >= minSubcadena; l-- {
		for i := 0; i+l <= len(s); i++ {
			out = append(out, s[i:i+l])
		}
	}
	return out
}

// buscar consulta TMview, con cache en disco, rate limit y PAGINACION.
//
// La paginacion no es una mejora, es una puerta. TMview devuelve como mucho
// pageSize registros por pagina y NO dice cuantos hay en total: no manda
// totalResults ni cabecera de conteo. Con una sola pagina de 50, un termino con
// mas de 50 anterioridades se cribaba contra las 50 primeras y las demas eran
// invisibles.
//
// Para "plazum" los terminos que deciden (plazum, plazu, lazum, plaz, lazu,
// azum) devuelven entre 0 y 4, o sea que aquel veredicto no estaba truncado;
// los que llegaban al tope eran los de tres letras, que son ruido por
// definicion. Pero eso es suerte del candidato, no una propiedad de la
// herramienta, y la proxima criba puede no tenerla.
//
// Se para cuando una pagina viene incompleta (menos de paginaTamano), que es
// como se sabe que era la ultima, y hay tope duro por si la base decide
// devolver paginas llenas para siempre.
func (c *criba) buscar(termino string) ([]marca, error) {
	if !c.sinCache {
		if ms, ok := c.leerCache(termino); ok {
			c.deCache++
			return ms, nil
		}
	}
	var todas []marca
	for pagina := 1; pagina <= maxPaginas; pagina++ {
		ms, err := c.consultarPagina(termino, pagina)
		if err != nil {
			return nil, err
		}
		todas = append(todas, ms...)
		if len(ms) < paginaTamano {
			c.escribirCache(termino, todas)
			return todas, nil
		}
	}
	// Tope alcanzado: se ESCUPE, no se devuelve lo que haya. Devolver mil de un
	// numero desconocido y pintar "sin hallazgos" seria exactamente el falso
	// verde que esta herramienta existe para no dar.
	return nil, fmt.Errorf("TMview sigue devolviendo paginas llenas para %q despues de %d "+
		"paginas de %d, o sea al menos %d anterioridades.\n"+
		"  No se puede cribar un termino asi: lo que se dejaria fuera es de tamano desconocido.\n"+
		"  Arreglo: el termino es demasiado generico para servir de subcadena. Sube minSubcadena\n"+
		"  o descarta el candidato por indistintivo",
		termino, maxPaginas, paginaTamano, maxPaginas*paginaTamano)
}

// consultarPagina es una consulta HTTP y nada mas. Va aparte de buscar para que
// el bucle de paginacion se lea sin ruido de transporte.
func (c *criba) consultarPagina(termino string, pagina int) ([]marca, error) {
	// Rate limit: se espera lo que falte desde la ultima consulta real.
	espera := c.espera
	if espera == 0 {
		espera = esperaEntreConsultas
	}
	if !c.ultima.IsZero() {
		if falta := espera - time.Since(c.ultima); falta > 0 {
			time.Sleep(falta)
		}
	}
	c.ultima = time.Now()
	c.consultas++

	cuerpo, err := json.Marshal(map[string]any{
		"page":        strconv.Itoa(pagina),
		"pageSize":    strconv.Itoa(paginaTamano),
		"criteria":    "E", // busqueda por similitud, ordenada por relevancia
		"basicSearch": termino,
		"fOffices":    []string{c.oficina},
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.url(), bytes.NewReader(cuerpo))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", agente)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("no responde TMview: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMview devolvio HTTP %d para %q pagina %d; "+
			"si es 403 o 405, revisa el user-agent y el metodo", resp.StatusCode, termino, pagina)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	ms, err := parsear(b)
	if err != nil {
		return nil, fmt.Errorf("respuesta de TMview ilegible para %q pagina %d: %w", termino, pagina, err)
	}
	return ms, nil
}

// url es el extremo de TMview, sobreescribible en pruebas.
func (c *criba) url() string {
	if c.base != "" {
		return c.base
	}
	return urlTMview
}

// comprobarTransporte es el canario, y es la puerta mas importante del programa.
//
// POR QUE EXISTE. Con pageSize a 200 la API devuelve HTTP 200 con la lista
// VACIA. No un error, no un 400: doscientos es mas de lo que sirve y contesta
// que no hay nada. Con la version anterior de este fichero, subir esa constante
// habria hecho que TODO candidato saliera "sin hallazgos" y la herramienta
// habria seguido imprimiendo su tabla con la misma cara de siempre.
//
// Un transporte roto y un nombre limpio se leen EXACTAMENTE IGUAL, y esa es la
// definicion de falso verde. Asi que antes de cribar nada se hace una consulta
// cuya respuesta se sabe que no puede estar vacia, y sin pasar por la cache: un
// canario servido de disco no dice nada de la red de hoy.
func (c *criba) comprobarTransporte() error {
	ms, err := c.consultarPagina(terminoCanario, 1)
	if err != nil {
		return fmt.Errorf("el canario de transporte no llego a responder: %w", err)
	}
	if len(ms) == 0 {
		return fmt.Errorf("el canario de transporte volvio VACIO.\n"+
			"  Se ha consultado %q en la oficina %s, que tiene cientos de anterioridades vivas,\n"+
			"  y TMview ha contestado con la lista vacia. Eso no es un nombre limpio: es que la\n"+
			"  consulta no esta llegando como la base la espera.\n"+
			"  Sin esto, cualquier candidato saldria 'sin hallazgos' y la tabla se leeria igual.\n"+
			"  Arreglo: mirar paginaTamano (con 200 la API contesta 200 OK y lista vacia), el\n"+
			"  user-agent, y que el cuerpo siga llevando page, pageSize, criteria y fOffices",
			terminoCanario, c.oficina)
	}
	return nil
}

// respuestaTMview es solo lo que se usa. TMview devuelve mucho mas.
type respuestaTMview struct {
	TradeMarks []struct {
		ApplicationNumber string   `json:"applicationNumber"`
		TmName            string   `json:"tmName"`
		TmOffice          string   `json:"tmOffice"`
		TmOfficeURL       string   `json:"tmOfficeURL"`
		TradeMarkStatus   string   `json:"tradeMarkStatus"`
		TradeMarkType     string   `json:"tradeMarkType"`
		NiceClass         []int    `json:"niceClass"`
		ApplicantName     []string `json:"applicantName"`
		ApplicationDate   string   `json:"applicationDate"`
		RegistrationDate  string   `json:"registrationDate"`
		ExpirationDate    string   `json:"expirationDate"`
	} `json:"tradeMarks"`
}

func parsear(b []byte) ([]marca, error) {
	var r respuestaTMview
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	out := make([]marca, 0, len(r.TradeMarks))
	for _, t := range r.TradeMarks {
		out = append(out, marca{
			Numero: t.ApplicationNumber, Nombre: t.TmName, Tipo: t.TradeMarkType,
			Estado: t.TradeMarkStatus, Clases: t.NiceClass,
			Titular:    strings.Join(t.ApplicantName, "; "),
			Registrada: soloFecha(t.RegistrationDate), Solicitada: soloFecha(t.ApplicationDate),
			Expira:  soloFecha(t.ExpirationDate),
			Oficina: t.TmOffice, URLDetalle: t.TmOfficeURL,
		})
	}
	return out, nil
}

func soloFecha(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}

// --- Cache en disco ---
//
// Por termino, no por candidato: las subcadenas se repiten mucho entre
// candidatos parecidos y asi una segunda ejecucion no vuelve a salir a la red.

func (c *criba) rutaCache(termino string) string {
	// hex del termino: evita cualquier sorpresa con nombres de fichero.
	return filepath.Join(c.cache, c.oficina+"-"+hex.EncodeToString([]byte(termino))+".json")
}

func (c *criba) leerCache(termino string) ([]marca, bool) {
	b, err := os.ReadFile(c.rutaCache(termino)) // #nosec G304 -- ruta derivada del termino, dentro del dir de cache
	if err != nil {
		return nil, false
	}
	var ms []marca
	if err := json.Unmarshal(b, &ms); err != nil {
		return nil, false
	}
	return ms, true
}

func (c *criba) escribirCache(termino string, ms []marca) {
	if err := os.MkdirAll(c.cache, 0o750); err != nil {
		return // la cache es una optimizacion; si falla, se sigue sin ella
	}
	b, err := json.Marshal(ms)
	if err != nil {
		return
	}
	_ = os.WriteFile(c.rutaCache(termino), b, 0o600)
}

// --- Salida ---

// nombreOficina traduce el codigo de TMview a algo que se pueda leer en una
// prueba. Existe porque la cabecera decia "oficina EUIPO" SIEMPRE, mirara
// donde mirara: con -oficina ES la consulta iba de verdad a la OEPM y los
// registros que salian eran espanoles (M1928953), pero el rotulo seguia
// diciendo EUIPO.
//
// En una herramienta cuyo unico producto es la PRUEBA, una cabecera que
// miente sobre que registro se ha consultado invalida la prueba entera: quien
// la lea dentro de un ano no puede saber donde se busco.
func nombreOficina(codigo string) string {
	switch strings.ToUpper(codigo) {
	case "EM":
		return "EUIPO (marca de la Union Europea)"
	case "ES":
		return "OEPM (marca nacional espanola)"
	}
	// Sin inventar: si es una oficina que esta funcion no conoce, se dice el
	// codigo tal cual en vez de adivinar un nombre.
	return "codigo de oficina " + codigo
}

func imprimirTabla(w io.Writer, hs []Hallazgo, clases []int, oficina string) {
	fmt.Fprintf(w, "criba de marca, clases %s, %s\n\n", listaClases(clases), nombreOficina(oficina))
	for _, h := range hs {
		fmt.Fprintf(w, "== %s  [%s]  (%d consultas, %d de cache)\n",
			strings.ToUpper(h.Candidato), h.Riesgo(), h.Consultas, h.DesdeCache)

		seccion(w, "COLISIONES (se llaman igual)", h.Colisiones, "")
		seccion(w, "CONTENEDORAS (una marca contiene el candidato)", h.Contenedoras, "")
		seccion(w, "CONTENIDAS (el candidato contiene una marca)", h.Contenidas,
			"        ^ esta es la lente que casi nadie mira, y la que encontro UTIQ dentro de DUTIQ")
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, "FALTA UN PASO, Y ES MANUAL: esto es un registro de MARCAS, no sabe de")
	fmt.Fprintln(w, "empresas en activo sin registrar. Busca cada finalista como nombre de")
	fmt.Fprintln(w, "empresa y como dominio antes de decidir. Un finalista limpio aqui se cayo")
	fmt.Fprintln(w, "asi el 26-08-2026: competidor directo con una letra de diferencia.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Una criba automatica reduce la sorpresa, no sustituye a un agente de la")
	fmt.Fprintln(w, "propiedad industrial. Nada de esto es un dictamen juridico.")
}

func seccion(w io.Writer, titulo string, ms []marca, nota string) {
	pintar := make([]marca, 0, len(ms))
	ruido := 0
	for _, m := range ms {
		if m.Nivel == nivelRuido && !verTodo {
			ruido++
			continue
		}
		pintar = append(pintar, m)
	}
	ms = pintar

	fmt.Fprintf(w, "   %s: %d\n", titulo, len(ms))
	if ruido > 0 {
		// El ruido se cuenta y se dice. Un umbral que descarta en silencio hace
		// que "sin hallazgos" se lea como "se ha mirado todo".
		fmt.Fprintf(w, "      (+%d por debajo del umbral: acronimos de menos de %d letras o "+
			"coincidencias parciales. Con -todo se listan)\n", ruido, minRelevante)
	}
	if len(ms) == 0 {
		return
	}
	if nota != "" {
		fmt.Fprintln(w, nota)
	}
	for _, m := range ms {
		fmt.Fprintf(w, "      [%-5s %3.0f%%] %-12s %-22s %-12s %-11s clases %-18s %s\n",
			m.Nivel, m.Cobertura*100,
			m.Numero, recortar(m.Nombre, 22), m.Tipo, m.Estado, listaClases(m.Clases), recortar(m.Titular, 30))
		if m.Coincidente != "" && !strings.EqualFold(m.Coincidente, m.Nombre) {
			fmt.Fprintf(w, "                   (encontrada buscando %q)\n", m.Coincidente)
		}
	}
}

func listaClases(cs []int) string {
	if len(cs) == 0 {
		return "todas"
	}
	p := make([]string, 0, len(cs))
	for _, c := range cs {
		p = append(p, strconv.Itoa(c))
	}
	return strings.Join(p, ",")
}

func recortar(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "."
}
