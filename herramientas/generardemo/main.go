// Comando generardemo: construye el expediente de demostracion y su contexto.
//
// Por que existe, que es una tension de diseno y no una limpieza.
//
// El expediente de demostracion es un artefacto de PRODUCTO: es lo que verifica
// `dutiq verify` recien instalado, y su valor esta justo en que ensena normas
// REALES, el ENS y el RGPD y el CRA, con sus articulos. Un demo que ensena
// urn:demo:agregada no le demuestra nada a quien lo abre.
//
// El escenario de nucleo/expediente/expediente_test.go es un artefacto de
// PRUEBA, y vive en nucleo/, que NO PUEDE cablear normas: lo vigila
// TestNingunaNormaCableada, que desde el 25-08-2026 mira tambien los _test.go de
// nucleo/ y adaptadores/. Las dos cosas compartian un solo constructor, asi que
// al purgar el escenario a identificadores sinteticos la regeneracion del demo
// quedo cerrada y el demo publicado solo se podia editar a mano.
//
// El arreglo es este directorio: el ESCENARIO vive en escenario.json, que si
// puede nombrar normas reales porque un fichero de datos no lo mira el detector,
// y el CODIGO que lo monta esta aqui, sin un solo identificador de norma. La
// frontera entre los dos:
//
//	datos    lo que el emisor AFIRMA: su corpus (paquetes, reglas de
//	         aplicabilidad en el dialecto, obligaciones), sus hechos, sus
//	         relojes con su calendario, sus pruebas y observaciones, y lo que
//	         declara haber calculado (aplicables, reclamaciones, estados,
//	         denominadores).
//	codigo   como se sella y se hace reproducible el artefacto: la clave del
//	         operador del demo, la derivacion de clave y nonce de cada entrada,
//	         el checkpoint Merkle y el sello RFC 3161, que se LEE y no se
//	         regenera.
//
// El sello no se regenera aqui a proposito: sale a la red y lo repone
// herramientas/sellardemo, a mano y nunca en CI. Esta herramienta lo lee de
// testdata y se niega a publicar nada si el expediente que sale no verifica con
// el verificador de verdad contra las raices embebidas.
//
// Uso:
//
//	go run ./herramientas/generardemo             # compara y no escribe nada
//	go run ./herramientas/generardemo -escribir   # dice que cambia y lo escribe
//
// Sin -escribir es una PUERTA: sale con codigo 1 y con el diff si lo publicado
// no es lo que sale del escenario. Nunca reescribe en silencio.
//
// Se ejecuta a mano y muy de vez en cuando. Lo que si corre en CI es
// main_test.go, que hace la comparacion en memoria: si alguien edita
// expediente-demo.json a mano, la puerta se pone roja y dice en que linea.
package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dutiq/adaptadores/tsa"
	"dutiq/nucleo/aplicabilidad"
	"dutiq/nucleo/corpus"
	"dutiq/nucleo/estado"
	"dutiq/nucleo/expediente"
	"dutiq/nucleo/ledger"
)

// Rutas, todas relativas a la raiz del repositorio, que se localiza sola.
const (
	rutaEscenario = "herramientas/generardemo/escenario.json"
	rutaSello     = "nucleo/expediente/testdata/sello-demo.bin"
	rutaDemo      = "expediente-demo.json"
	rutaContexto  = "contexto-demo.json"
)

// anclajeDeclarado es lo que el checkpoint DICE de su anclaje temporal. No
// decide nada: quien verifica comprueba el token RFC 3161 contra sus raices.
const anclajeDeclarado = "tsa:rfc3161://tsa.example + testigo publico diario"

// semillaDelOperador genera la clave con la que el demo firma sus checkpoints.
//
// Es una clave de DEMOSTRACION y su privada es publica a proposito: sin ella el
// demo no seria reproducible y cada regeneracion daria un fichero distinto. No
// es un secreto filtrado, es parte del artefacto. La misma cadena aparece en los
// tests que verifican el demo, y tiene que seguir coincidiendo.
const semillaDelOperador = "dutiq-demo-semilla-determinista"

// Errores como centinelas: un test que compruebe que sin sello no se publica
// tiene que poder hacerlo con errors.Is y no buscando una subcadena.
var (
	ErrSinSello       = errors.New("el demo no lleva sello RFC 3161")
	ErrNoVerifica     = errors.New("el expediente generado no verifica")
	ErrEscenario      = errors.New("el escenario no se puede leer")
	ErrDesincronizado = errors.New("lo publicado no es lo que sale del escenario")
)

func main() {
	escribir := flag.Bool("escribir", false,
		"escribe expediente-demo.json y contexto-demo.json; sin esto solo compara")
	flag.Parse()
	if err := ejecutar(*escribir, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func ejecutar(escribir bool, out io.Writer) error {
	raiz, err := raizDelRepo()
	if err != nil {
		return err
	}
	g, err := generar(raiz)
	if err != nil {
		return err
	}

	cambios := 0
	for _, f := range []struct {
		ruta   string
		quiero []byte
	}{
		{rutaDemo, g.Demo},
		{rutaContexto, g.Contexto},
	} {
		completa := filepath.Join(raiz, f.ruta)
		publicado, err := os.ReadFile(completa) // #nosec G304 -- ruta fija del repo
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("no puedo leer %s: %w", f.ruta, err)
		}
		difs := diferencias(publicado, f.quiero)
		if len(difs) == 0 {
			fmt.Fprintf(out, "  igual  %s (%d bytes)\n", f.ruta, len(f.quiero))
			continue
		}
		cambios++
		fmt.Fprintf(out, "  CAMBIA %s (%d diferencias)\n", f.ruta, len(difs))
		for _, d := range difs {
			fmt.Fprintln(out, "    "+d)
		}
	}

	if cambios == 0 {
		fmt.Fprintln(out, "lo publicado es exactamente lo que sale del escenario")
		return nil
	}
	if !escribir {
		return fmt.Errorf("%w. Arriba esta en que difiere. Si el cambio es el que "+
			"buscabas, repitelo con -escribir; si no, el escenario o el fichero publicado "+
			"se han tocado a mano y hay que decidir cual gana", ErrDesincronizado)
	}
	if err := os.WriteFile(filepath.Join(raiz, rutaDemo), g.Demo, 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(raiz, rutaContexto), g.Contexto, 0o600); err != nil {
		return err
	}
	fmt.Fprintf(out, "escritos %s y %s\n", rutaDemo, rutaContexto)
	fmt.Fprintln(out, "el sello NO se ha tocado: si has cambiado las observaciones que se")
	fmt.Fprintln(out, "encadenan, la generacion habria fallado antes de llegar aqui.")
	return nil
}

// Generado es lo que produce la herramienta: el expediente ya verificado y los
// bytes exactos de los dos ficheros publicados.
type Generado struct {
	Expediente *expediente.Expediente
	Demo       []byte
	Contexto   []byte
}

func generar(raiz string) (*Generado, error) {
	esc, err := leerEscenario(filepath.Join(raiz, rutaEscenario))
	if err != nil {
		return nil, err
	}
	sello, err := leerSello(filepath.Join(raiz, rutaSello))
	if err != nil {
		return nil, err
	}
	return Montar(esc, sello)
}

// Montar construye el expediente desde el escenario ya leido y lo verifica.
// Sin tocar disco, para que un test pueda mutar el escenario y ver que pasa.
func Montar(esc *Escenario, sello []byte) (*Generado, error) {
	e, err := construir(esc, sello)
	if err != nil {
		return nil, err
	}
	anclas := map[string]string{}
	for _, p := range e.Paquetes {
		anclas[p.URN] = p.Digest
	}
	if err := verificarDeVerdad(e, anclas); err != nil {
		return nil, err
	}
	demo, err := e.Guardar()
	if err != nil {
		return nil, err
	}
	ctx := contextoPublicado{
		Anclas:           anclas,
		ClavesConfiables: e.Cadena.ClavesDeclaradas,
		ClaveOperador:    e.Cadena.ClavesDeclaradas[0],
	}
	bctx, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return nil, err
	}
	// El expediente se publica tal cual lo serializa Guardar, sin salto final:
	// asi lo lleva el fichero commiteado y un byte de mas seria un cambio del
	// artefacto publicado sin ninguna razon. El contexto si lo lleva, tambien
	// como esta publicado.
	return &Generado{Expediente: e, Demo: demo, Contexto: append(bctx, '\n')}, nil
}

// contextoPublicado es contexto-demo.json: lo que aporta el RECEPTOR. Se publica
// al lado del expediente porque sin anclas del registro la verificacion seria
// circular, y quien acaba de instalar el binario no tiene registro todavia.
//
// Se genera aqui, y no a mano, justo para que las anclas no puedan dejar de
// cuadrar con el expediente: salen del mismo calculo.
type contextoPublicado struct {
	Anclas           map[string]string `json:"anclas"`
	ClavesConfiables []string          `json:"claves_confiables"`
	ClaveOperador    string            `json:"clave_operador"`
}

// ---------------------------------------------------------------------------
// El escenario, como datos
// ---------------------------------------------------------------------------

// Escenario es escenario.json. Los tipos que ya viajan con etiquetas en
// castellano (obligaciones, relojes, calendario, reclamaciones, estados) se
// reutilizan tal cual; los que no las tienen se declaran aqui para que el
// fichero se lea, en vez de pedir nanosegundos y nombres de campo de Go.
type Escenario struct {
	Leeme string `json:"_leeme"`

	Organizacion string    `json:"organizacion"`
	Alcance      string    `json:"alcance"`
	Emitido      time.Time `json:"emitido"`
	ComoEstaba   time.Time `json:"como_estaba"`

	Paquetes     []PaqueteEscenario      `json:"paquetes"`
	Hechos       []HechoEscenario        `json:"hechos"`
	Obligaciones []expediente.Obligacion `json:"obligaciones"`

	// Calendario se declara UNA vez y se le pega a cada reloj al montar: en el
	// expediente viaja repetido dentro de cada uno, y repetirlo en el fichero
	// de datos es pedir que dos copias se separen.
	Calendario expediente.CalendarioDeclarado `json:"calendario"`
	Relojes    []expediente.RelojDeclarado    `json:"relojes"`

	Pruebas       []PruebaEscenario      `json:"pruebas"`
	Observaciones []ObservacionEscenario `json:"observaciones"`

	Declarado Declarado `json:"declarado"`
}

// PaqueteEscenario es un paquete del corpus del demo. Las reglas van en el
// MISMO formato que las declara un paquete de verdad bajo paquetes/, y las
// carga el mismo codigo, para que el demo no pueda ensenar una forma de
// declarar reglas que el producto no acepta.
type PaqueteEscenario struct {
	URN      string               `json:"urn"`
	Version  string               `json:"version"`
	Clase    string               `json:"clase"`
	Vigencia expediente.Intervalo `json:"vigencia"`
	Reglas   []corpus.ReglaSpec   `json:"reglas"`
}

type HechoEscenario struct {
	Pred string   `json:"pred"`
	Args []string `json:"args"`
}

// PruebaEscenario declara TTL y SLA como duraciones de Go (24h, 2160h) y no
// como los nanosegundos con los que viajan serializados.
type PruebaEscenario struct {
	ID          string `json:"id"`
	Control     string `json:"control"`
	TTL         string `json:"ttl"`
	SLA         string `json:"sla"`
	Descripcion string `json:"descripcion"`
}

type ObservacionEscenario struct {
	Prueba      string    `json:"prueba"`
	Recurso     string    `json:"recurso"`
	Satisfecho  bool      `json:"satisfecho"`
	Recolectada time.Time `json:"recolectada"`
	Recolector  string    `json:"recolector"`
	Version     string    `json:"version"`
	HashCarga   string    `json:"hash_carga"`
}

// Declarado es lo que el emisor dice haber calculado. Va como dato y no se
// calcula aqui a proposito: el demo tiene que ensenar la propiedad del formato,
// que es que un tercero RECALCULA esto y lo contrasta. Si se generara desde el
// mismo motor que luego verifica, el demo no probaria nada, y una declaracion
// equivocada en el escenario no romperia nada. Ahora rompe: Montar verifica
// antes de devolver.
type Declarado struct {
	Aplicables    []string                   `json:"aplicables"`
	Reclamaciones []expediente.Reclamacion   `json:"reclamaciones"`
	Estados       []expediente.EstadoControl `json:"estados"`
	Denominadores DenominadoresEscenario     `json:"denominadores"`
}

type DenominadoresEscenario struct {
	Maquina              int `json:"maquina"`
	Humano               int `json:"humano"`
	Externo              int `json:"externo"`
	Desconocido          int `json:"desconocido"`
	CaducadoOContradicho int `json:"caducado_o_contradicho"`
}

// LeerEscenario lee y valida el fichero de datos.
//
// Rechaza campos desconocidos, y no es celo: un escenario es un fichero que se
// edita a mano, y una errata en el nombre de una seccion la haria desaparecer
// del expediente EN SILENCIO. Con esto la errata sale por pantalla.
func LeerEscenario(b []byte) (*Escenario, error) {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	var esc Escenario
	if err := d.Decode(&esc); err != nil {
		return nil, fmt.Errorf("%w: %w. Si el campo no lo conoce nadie, o sobra o esta mal "+
			"escrito; comparalo con los tipos de herramientas/generardemo/main.go", ErrEscenario, err)
	}
	// Y que no haya nada DETRAS del objeto. Un decodificador de JSON se para en
	// la primera llave que cierra, asi que un fichero con dos escenarios pegados
	// usaria el primero y tiraria el segundo sin decir nada.
	if d.More() {
		return nil, fmt.Errorf("%w: hay contenido despues del objeto del escenario. "+
			"El fichero tiene que ser UN solo objeto JSON; lo que va detras se estaria "+
			"ignorando en silencio", ErrEscenario)
	}
	return &esc, nil
}

func leerEscenario(ruta string) (*Escenario, error) {
	b, err := os.ReadFile(ruta) // #nosec G304 -- ruta fija del repo
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEscenario, err)
	}
	return LeerEscenario(b)
}

// leerSello lee el token RFC 3161 que ya esta commiteado. NO lo regenera: eso
// sale a la red y lo hace herramientas/sellardemo, a mano.
func leerSello(ruta string) ([]byte, error) {
	b, err := os.ReadFile(ruta) // #nosec G304 -- ruta fija del repo
	if err != nil || len(b) == 0 {
		return nil, fmt.Errorf("%w: no hay token en %s (%v).\n"+
			"  El demo publicado no puede llevar un sello de relleno: lo primero que hace\n"+
			"  cualquiera es verificarlo, y con un sello inventado eso falla.\n"+
			"  Reponlo con: go run ./herramientas/sellardemo", ErrSinSello, ruta, err)
	}
	return b, nil
}

// ---------------------------------------------------------------------------
// De datos a expediente
// ---------------------------------------------------------------------------

func construir(esc *Escenario, sello []byte) (*expediente.Expediente, error) {
	programas := make([]aplicabilidad.Programa, 0, len(esc.Paquetes))
	paquetes := make([]expediente.Paquete, 0, len(esc.Paquetes))
	for _, p := range esc.Paquetes {
		// El mismo cargador que usa el corpus publicado: dialecto, agregado,
		// escala y linter incluidos. Asi una regla que el producto no aceptaria
		// tampoco entra en el demo.
		pk := corpus.Paquete{URN: p.URN, Aplicabilidad: corpus.Aplicabilidad{Reglas: p.Reglas}}
		prog, errs := pk.Programa()
		if len(errs) > 0 {
			var msgs []string
			for _, e := range errs {
				msgs = append(msgs, e.Error())
			}
			return nil, fmt.Errorf("%w: las reglas de %s no cargan:\n  %s",
				ErrEscenario, p.URN, strings.Join(msgs, "\n  "))
		}
		programas = append(programas, prog)
		paquetes = append(paquetes, expediente.Paquete{
			URN: p.URN, Version: p.Version, Clase: p.Clase, Vigencia: p.Vigencia,
		})
	}

	hechos := make([]aplicabilidad.Hecho, 0, len(esc.Hechos))
	for _, h := range esc.Hechos {
		hechos = append(hechos, aplicabilidad.H(h.Pred, h.Args...))
	}

	pruebas := make([]estado.Prueba, 0, len(esc.Pruebas))
	for _, p := range esc.Pruebas {
		ttl, err := time.ParseDuration(p.TTL)
		if err != nil {
			return nil, fmt.Errorf("%w: ttl de la prueba %s: %w. Escribelo como duracion "+
				"de Go, por ejemplo 24h o 2160h", ErrEscenario, p.ID, err)
		}
		sla, err := time.ParseDuration(p.SLA)
		if err != nil {
			return nil, fmt.Errorf("%w: sla de la prueba %s: %w. Escribelo como duracion "+
				"de Go, por ejemplo 72h o 720h", ErrEscenario, p.ID, err)
		}
		pruebas = append(pruebas, estado.Prueba{
			ID: p.ID, Control: p.Control, TTL: ttl, SLA: sla, Descripcion: p.Descripcion,
		})
	}

	observaciones := make([]estado.Observacion, 0, len(esc.Observaciones))
	for _, o := range esc.Observaciones {
		observaciones = append(observaciones, estado.Observacion{
			Prueba: o.Prueba, Recurso: o.Recurso, Satisfecho: o.Satisfecho,
			Recolectada: o.Recolectada, Recolector: o.Recolector,
			Version: o.Version, HashCarga: o.HashCarga,
		})
	}

	relojes := make([]expediente.RelojDeclarado, 0, len(esc.Relojes))
	for _, r := range esc.Relojes {
		r.Calendario = esc.Calendario
		relojes = append(relojes, r)
	}

	e := &expediente.Expediente{
		Version: expediente.Version, Emitido: esc.Emitido, ComoEstaba: esc.ComoEstaba,
		Organizacion: esc.Organizacion, Alcance: esc.Alcance,
		Paquetes: paquetes, Programas: programas, Hechos: hechos,
		Obligaciones:  esc.Obligaciones,
		Pruebas:       pruebas,
		Observaciones: observaciones,
		Relojes:       relojes,
		Aplicables:    esc.Declarado.Aplicables,
		Reclamaciones: esc.Declarado.Reclamaciones,
		Estados:       esc.Declarado.Estados,
		Denominadores: estado.Denominadores{
			Maquina:              esc.Declarado.Denominadores.Maquina,
			Humano:               esc.Declarado.Denominadores.Humano,
			Externo:              esc.Declarado.Denominadores.Externo,
			Desconocido:          esc.Declarado.Denominadores.Desconocido,
			CaducadoOContradicho: esc.Declarado.Denominadores.CaducadoOContradicho,
		},
	}

	// Los digests salen del CONTENIDO (reglas y obligaciones de cada paquete),
	// no del fichero de datos: si alguien toca una regla, el ancla cambia sola.
	e.AnclasDeclaradas = map[string]string{}
	for i := range e.Paquetes {
		d := expediente.DigestPaquete(e.Paquetes[i].URN, e.Programas, e.Obligaciones)
		e.Paquetes[i].Digest = d
		e.AnclasDeclaradas[e.Paquetes[i].URN] = d
	}

	if err := encadenar(e, observaciones, sello); err != nil {
		return nil, err
	}
	return e, nil
}

// encadenar monta la cadena v2: una entrada cifrada por observacion, sus claves
// divulgadas, y un checkpoint firmado sobre la raiz Merkle con el sello leido.
//
// Todo lo de aqui es determinista a proposito. La clave y el nonce de cada
// entrada salen de su indice y la del operador de una semilla fija: sin eso,
// dos regeneraciones del demo darian ficheros distintos y no habria forma de
// saber si el cambio es el que se buscaba.
func encadenar(e *expediente.Expediente, obs []estado.Observacion, sello []byte) error {
	semilla := make([]byte, ed25519.SeedSize)
	copy(semilla, []byte(semillaDelOperador))
	k := ed25519.NewKeyFromSeed(semilla)
	e.Cadena.ClavesDeclaradas = []string{hex.EncodeToString(k.Public().(ed25519.PublicKey))}

	ks := ledger.NuevoKeystore()
	e.ClavesEntradas = map[uint64]string{}
	for i, o := range obs {
		carga, err := json.Marshal(o)
		if err != nil {
			return err
		}
		clave, nonce := claveDeEntrada(byte(i+1)), nonceDeEntrada(byte(i+1))
		ent, err := e.Cadena.Anadir(ks, clave, nonce, carga)
		if err != nil {
			return err
		}
		e.ClavesEntradas[ent.Indice] = hex.EncodeToString(clave)
	}
	e.Cadena.Cerrar(k, e.ComoEstaba, anclajeDeclarado, sello)
	return nil
}

func claveDeEntrada(b byte) []byte {
	c := make([]byte, 32)
	for i := range c {
		c[i] = b
	}
	return c
}

func nonceDeEntrada(b byte) []byte {
	n := make([]byte, 12)
	for i := range n {
		n[i] = b
	}
	return n
}

// ---------------------------------------------------------------------------
// La puerta: nada se publica sin verificar
// ---------------------------------------------------------------------------

// verificarDeVerdad pasa el expediente por el verificador del producto, con el
// verificador de sellos real y las raices que trae el binario. Sin red.
//
// Las anclas que se le dan son las que se van a publicar en contexto-demo.json,
// que salen del propio expediente: aqui eso es correcto porque el demo publica
// su registro al lado. Que un emisor NO pueda fabricarse sus anclas tiene su
// propio test en nucleo/expediente.
func verificarDeVerdad(e *expediente.Expediente, anclas map[string]string) error {
	pool, err := tsa.RaicesPorDefecto()
	if err != nil {
		return err
	}
	cadena := &tsa.Cadena{Anclas: pool}
	inf := expediente.Verificar(e, expediente.ContextoReceptor{
		Anclas:           anclas,
		ClavesConfiables: e.Cadena.ClavesDeclaradas,
		ClaveOperador:    claveOperador(e),
		VerificarSello:   cadena.VerificarOffline,
	})
	if inf.Valido {
		return nil
	}
	var difs []string
	for _, d := range inf.Discrepancias {
		difs = append(difs, fmt.Sprintf("%s: esperado %q, obtenido %q", d.Que, d.Esperado, d.Obtenido))
	}
	return fmt.Errorf("%w, asi que no se publica:\n  %s\n\n"+
		"  Si has cambiado las OBSERVACIONES, la raiz Merkle de la cadena ha cambiado y el\n"+
		"  sello commiteado ya no la cubre. Vuelve a sellar y repite:\n"+
		"    go run ./herramientas/sellardemo\n"+
		"    go run ./herramientas/generardemo -escribir\n"+
		"  Si has cambiado reglas, hechos u obligaciones, revisa lo que declara la seccion\n"+
		"  \"declarado\" del escenario: el verificador lo recalcula y no se fia de ella",
		ErrNoVerifica, strings.Join(difs, "\n  "))
}

func claveOperador(e *expediente.Expediente) ed25519.PublicKey {
	b, err := hex.DecodeString(e.Cadena.ClavesDeclaradas[0])
	if err != nil {
		return nil
	}
	return b
}

// ---------------------------------------------------------------------------
// Comparacion
// ---------------------------------------------------------------------------

// diferencias compara linea a linea y devuelve las primeras que no coinciden.
// Linea a linea y no "son distintos" porque el fichero tiene 900 lineas y un
// diff que no dice donde obliga a buscarlo a mano.
func diferencias(publicado, generado []byte) []string {
	if string(publicado) == string(generado) {
		return nil
	}
	a := strings.Split(string(publicado), "\n")
	b := strings.Split(string(generado), "\n")
	const tope = 12
	var out []string
	n := len(a)
	if len(b) > n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		var la, lb string
		if i < len(a) {
			la = a[i]
		}
		if i < len(b) {
			lb = b[i]
		}
		if la == lb {
			continue
		}
		if len(out) == tope {
			out = append(out, fmt.Sprintf("... y mas diferencias a partir de la linea %d", i+1))
			break
		}
		out = append(out, fmt.Sprintf("linea %d:\n      publicado: %s\n      generado : %s",
			i+1, recortar(la), recortar(lb)))
	}
	return out
}

func recortar(s string) string {
	const tope = 120
	s = strings.TrimRight(s, "\r")
	if len(s) <= tope {
		return s
	}
	return s[:tope] + "..."
}

// raizDelRepo sube hasta encontrar go.mod. Asi la herramienta funciona desde
// donde se lance, y no hace falta acordarse de ejecutarla desde la raiz.
func raizDelRepo() (string, error) {
	d, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d, nil
		}
		padre := filepath.Dir(d)
		if padre == d {
			return "", errors.New("no encuentro la raiz del repositorio (ningun go.mod " +
				"por encima del directorio actual). Ejecuta esto dentro del repositorio")
		}
		d = padre
	}
}
