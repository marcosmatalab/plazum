package latido

import (
	"context"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// La frontera de lo que sale de la maquina del operador.
//
// SE COMPRUEBA CONTRA UNA LISTA BLANCA, no contra una lista negra, y esa es la
// unica decision de este fichero. Una lista negra responde "¿va aqui algo
// prohibido?", que es una pregunta que solo se puede contestar sobre lo que ya
// se te ha ocurrido; una lista blanca responde "¿es esto exactamente lo unico
// que va?", que es la pregunta que hay que contestar. El dia que alguien anada
// un campo "util" al pulso, esto se pone rojo sin que nadie haya tenido que
// preverlo.
//
// Y se comprueba en los DOS sitios por los que se filtra algo: el cuerpo y la
// peticion. Un cuerpo impecable con un identificador en la parte de consulta de
// la direccion, o en una cabecera, filtra igual.

// camposDelPulso es la lista blanca del cuerpo. Dos.
var camposDelPulso = []string{"instancia", "instante"}

// cabecerasDelPulso es la lista blanca de la peticion.
var cabecerasDelPulso = []string{"Content-Length", "Content-Type", "User-Agent"}

// clavesDe devuelve las claves de un objeto JSON, ordenadas.
func clavesDeJSON(t *testing.T, b []byte) []string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("el pulso no es un objeto JSON: %v", err)
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// El cuerpo del pulso lleva exactamente dos campos.
//
// Se mira lo que de verdad viaja (los bytes serializados) y ademas la forma del
// tipo. Lo primero caza un campo anadido; lo segundo caza un campo que hoy no
// se serializa pero manana si, porque alguien le quite el `json:"-"`.
func TestElPulsoLlevaExactamenteDosCamposYNiUnoMas(t *testing.T) {
	b, err := json.Marshal(Pulso{Instancia: "abc", Instante: ahora})
	if err != nil {
		t.Fatal(err)
	}
	if got := clavesDeJSON(t, b); !reflect.DeepEqual(got, camposDelPulso) {
		t.Errorf("el pulso manda %v y lo declarado es %v.\n"+
			"  Lo que sale de la maquina del operador tiene que caber en las dos lineas de\n"+
			"  latido.QueSeManda. Si de verdad hace falta un campo mas, se anade AQUI y se\n"+
			"  reescribe esa declaracion, que es lo que el operador acepto.\n"+
			"  Cuerpo: %s", got, camposDelPulso, b)
	}

	tipo := reflect.TypeOf(Pulso{})
	if tipo.NumField() != len(camposDelPulso) {
		t.Errorf("el tipo Pulso tiene %d campos y la lista blanca declara %d",
			tipo.NumField(), len(camposDelPulso))
	}
	for i := 0; i < tipo.NumField(); i++ {
		f := tipo.Field(i)
		nombre := strings.Split(f.Tag.Get("json"), ",")[0]
		if !contiene(camposDelPulso, nombre) {
			t.Errorf("el campo %s del pulso sale como %q, que no esta en la lista blanca",
				f.Name, nombre)
		}
	}
}

// CONTROL NEGATIVO de la comprobacion de arriba. Sin esto, un verde no prueba
// que se este mirando nada: un extractor que devolviera siempre la lista buena
// daria exactamente el mismo verde.
func TestLaListaBlancaCazaUnCampoDeMas(t *testing.T) {
	// Exactamente la forma que tiene el "campo util" que alguien anadiria:
	// el nombre de la organizacion, para poder cruzar los pulsos con los
	// clientes.
	conDeMas := struct {
		Instancia    string    `json:"instancia"`
		Instante     time.Time `json:"instante"`
		Organizacion string    `json:"organizacion"`
	}{"abc", ahora, "Acme SL"}
	b, err := json.Marshal(conDeMas)
	if err != nil {
		t.Fatal(err)
	}
	if got := clavesDeJSON(t, b); reflect.DeepEqual(got, camposDelPulso) {
		t.Fatalf("la comprobacion da por buena una carga con un campo de mas: %v", got)
	}
}

func contiene(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}

// La peticion que llega al otro lado lleva solo lo declarado.
//
// Esto no sale a la red: httptest levanta el receptor en la propia maquina, asi
// que el test es hermetico y ademas prueba el canal de verdad, con su cliente
// HTTP de verdad. Un smoke test que use un camino distinto del real prueba el
// camino distinto.
func TestLaPeticionQueSaleLlevaSoloLoDeclarado(t *testing.T) {
	// El candado no es adorno: lo que apunta el manejador corre en la goruta
	// del servidor y se lee en la del test. Sin el, la suite entera con
	// -race, que es una puerta de CI, se pondria roja por el arnes y no por
	// el producto.
	var mu sync.Mutex
	var visto *http.Request
	var cuerpo []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		visto = r.Clone(context.Background())
		cuerpo, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	dir := t.TempDir()
	if _, err := Activar(dir, srv.URL+"/latido", ahora, &secretosDePrueba{}); err != nil {
		t.Fatal(err)
	}
	e, err := Cargar(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Pulsar(context.Background(), e, CanalHTTP{}, ahora); err != nil {
		t.Fatalf("el pulso contra un receptor que contesta 204 falla: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if visto == nil {
		t.Fatal("no ha llegado ninguna peticion")
	}

	if visto.Method != http.MethodPost {
		t.Errorf("el pulso va por %s", visto.Method)
	}
	if visto.URL.Path != "/latido" {
		t.Errorf("el pulso va a %q", visto.URL.Path)
	}
	if visto.URL.RawQuery != "" {
		t.Errorf("la direccion del pulso lleva parte de consulta: %q. Ahi es donde se cuela "+
			"un identificador y ademas acaba en los logs de cada intermediario",
			visto.URL.RawQuery)
	}
	if len(visto.Cookies()) != 0 {
		t.Errorf("el pulso lleva cookies: %v", visto.Cookies())
	}

	var cabeceras []string
	for k := range visto.Header {
		cabeceras = append(cabeceras, k)
	}
	sort.Strings(cabeceras)
	if !reflect.DeepEqual(cabeceras, cabecerasDelPulso) {
		t.Errorf("la peticion lleva las cabeceras %v y lo declarado es %v.\n"+
			"  Una cabecera de mas filtra igual que un campo de mas: el nombre de la\n"+
			"  maquina, una version, una marca de tiempo local o un token son todos\n"+
			"  identificadores", cabeceras, cabecerasDelPulso)
	}
	if ua := visto.Header.Get("User-Agent"); ua != AgenteDeUsuario {
		t.Errorf("el agente de usuario es %q y esperaba %q. Meter la version del producto "+
			"ahi es mandar un dato mas sin decirlo en QueSeManda", ua, AgenteDeUsuario)
	}

	if got := clavesDeJSON(t, cuerpo); !reflect.DeepEqual(got, camposDelPulso) {
		t.Errorf("el cuerpo que llega al receptor lleva %v: %s", got, cuerpo)
	}
	var p Pulso
	if err := json.Unmarshal(cuerpo, &p); err != nil {
		t.Fatal(err)
	}
	if p.Instancia != e.Instancia {
		t.Errorf("el pulso manda la instancia %q y la de esta instalacion es %q",
			p.Instancia, e.Instancia)
	}
	if !p.Instante.Equal(ahora) {
		t.Errorf("el pulso manda el instante %v y esperaba %v", p.Instante, ahora)
	}
}

// El canal no sigue redirecciones.
//
// Una redireccion mueve el pulso a otra maquina, o le anade una parte de
// consulta, sin que el operador se entere: el destino que el escribio deja de
// ser el destino. Es la unica forma de que un receptor "de confianza" reenvie a
// donde quiera.
func TestElCanalNoSigueRedirecciones(t *testing.T) {
	var mu sync.Mutex
	llegoAlSegundo := false
	segundo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		llegoAlSegundo = true
		mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer segundo.Close()
	primero := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, segundo.URL+"/latido?org=acme", http.StatusTemporaryRedirect)
	}))
	defer primero.Close()

	err := CanalHTTP{}.Entregar(context.Background(), primero.URL+"/latido", []byte(`{}`))
	if err == nil {
		t.Fatal("el canal ha seguido la redireccion sin protestar")
	}
	mu.Lock()
	defer mu.Unlock()
	if llegoAlSegundo {
		t.Error("el pulso ha acabado en una maquina que el operador no escribio, y con un " +
			"identificador en la parte de consulta puesto por el intermediario")
	}
}

// Un receptor que contesta mal se informa como fallo del canal, no como si el
// pulso hubiera llegado.
func TestUnReceptorQueContestaMalNoCuentaComoPulsoEntregado(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	dir := t.TempDir()
	if _, err := Activar(dir, srv.URL+"/latido", ahora, &secretosDePrueba{}); err != nil {
		t.Fatal(err)
	}
	e, err := Probar(context.Background(), dir, CanalHTTP{}, ahora)
	if err == nil {
		t.Fatal("un 500 del receptor se ha dado por bueno")
	}
	if !e.UltimoPulso.IsZero() {
		t.Errorf("un 500 del receptor ha quedado apuntado como pulso entregado: %v",
			e.UltimoPulso)
	}
	if !e.FalloElUltimoIntento {
		t.Error("un 500 del receptor no ha quedado apuntado como fallo")
	}
}

// ---------------------------------------------------------------------------
// Lo que este paquete no puede ni mirar
// ---------------------------------------------------------------------------

// importesYLlamadasProhibidas devuelve lo que un fuente de este paquete no
// puede hacer, con el motivo.
//
// Vive fuera del test para que el control negativo pase por este mismo codigo:
// un detector que solo se ha ejecutado contra el caso bueno no ha demostrado
// que sepa decir que no.
func importesYLlamadasProhibidas(a *ast.File) []string {
	var out []string
	for _, imp := range a.Imports {
		v := strings.Trim(imp.Path.Value, `"`)
		switch {
		case strings.HasPrefix(v, "plazum/nucleo/") && v != "plazum/nucleo/pantalla":
			out = append(out, v+": el latido no puede leer el corpus ni el estado de "+
				"cumplimiento. Un adaptador de telemetria que puede leerlo acaba mandandolo")
		case strings.HasPrefix(v, "plazum/superficies/"):
			out = append(out, v+": el latido no depende de la interfaz web")
		case v == "os/user":
			out = append(out, v+": el usuario del sistema es un identificador de una persona")
		}
	}
	// Llamadas que producen un identificador de la maquina o del entorno.
	prohibidas := map[string]string{
		"os.Hostname":        "el nombre de la maquina es un identificador de la organizacion",
		"os.Getenv":          "el entorno lleva nombres, rutas y secretos",
		"os.Environ":         "el entorno lleva nombres, rutas y secretos",
		"net.Interfaces":     "una MAC es un identificador permanente del equipo",
		"net.InterfaceAddrs": "una direccion de red es un identificador de la red del cliente",
		"user.Current":       "el usuario del sistema es un identificador de una persona",
	}
	ast.Inspect(a, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if motivo, mal := prohibidas[id.Name+"."+sel.Sel.Name]; mal {
			out = append(out, id.Name+"."+sel.Sel.Name+": "+motivo)
		}
		return true
	})
	return out
}

// El latido no puede leer el corpus, ni el estado de cumplimiento, ni el nombre
// de la maquina, ni el entorno, ni las interfaces de red.
//
// Es una puerta ESTRUCTURAL y por eso vale mas que la lista blanca del cuerpo:
// la lista blanca dice que hoy no se manda nada de eso, y esto dice que no se
// puede, porque el codigo ni siquiera lo tiene delante. El identificador de
// instalacion es aleatorio, generado aqui, y no se deriva de nada del operador.
func TestElLatidoNoPuedeLeerElCorpusNiLaMaquina(t *testing.T) {
	fset := token.NewFileSet()
	fuentes, err := filepath.Glob("*.go")
	if err != nil || len(fuentes) == 0 {
		t.Fatalf("no encuentro los fuentes del paquete: %v", err)
	}
	n := 0
	for _, f := range fuentes {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		n++
		a, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, h := range importesYLlamadasProhibidas(a) {
			t.Errorf("%s: %s", f, h)
		}
	}
	if n < 2 {
		t.Fatalf("solo se han mirado %d fuentes de produccion: el detector no esta viendo "+
			"el paquete", n)
	}
}

// CONTROL NEGATIVO del detector. Se le da un fuente con las cuatro cosas
// prohibidas y se exige que las senale, y uno legitimo del que no puede
// protestar.
func TestElDetectorDeFronteraSaltaCuandoDebe(t *testing.T) {
	malo := `package latido

import (
	"os"
	"plazum/nucleo/corpus"
	"os/user"
)

func x() {
	h, _ := os.Hostname()
	_ = h
	_ = os.Getenv("HOME")
	_ = corpus.Cargar
	_, _ = user.Current()
}
`
	bueno := `package latido

import (
	"encoding/json"
	"net/http"
	"os"

	"plazum/nucleo/pantalla"
	"plazum/puertos"
)

func x() {
	_, _ = os.ReadFile("latido.json")
	_ = json.Marshal
	_ = http.MethodPost
	_ = pantalla.Marcas{}
	var _ puertos.Secretos
}
`
	fset := token.NewFileSet()
	a, err := parser.ParseFile(fset, "malo.go", malo, 0)
	if err != nil {
		t.Fatal(err)
	}
	// dos imports (corpus y os/user) y tres llamadas (Hostname, Getenv,
	// user.Current).
	if h := importesYLlamadasProhibidas(a); len(h) != 5 {
		t.Fatalf("el detector tenia que encontrar 5 problemas y encontro %d: %v", len(h), h)
	}
	b, err := parser.ParseFile(fset, "bueno.go", bueno, 0)
	if err != nil {
		t.Fatal(err)
	}
	if h := importesYLlamadasProhibidas(b); len(h) != 0 {
		t.Fatalf("falso positivo sobre un fuente legitimo: %v", h)
	}
}

// La declaracion de lo que se manda cabe en DOS LINEAS.
//
// No es cosmetica: la restriccion de tamano es la puerta. Si la lista de campos
// no cabe en dos lineas de documentacion, es que se esta mandando demasiado, y
// esta comprobacion obliga a que anadir un campo sea una decision que alguien
// tiene que defender por escrito.
func TestLoQueSeMandaCabeEnDosLineas(t *testing.T) {
	lineas := strings.Split(strings.TrimSpace(QueSeManda), "\n")
	if len(lineas) != 2 {
		t.Errorf("la declaracion tiene %d lineas y tiene que caber en 2", len(lineas))
	}
	// Y dice las dos cosas que se mandan, y que no se manda nada mas.
	for _, quiero := range []string{"identificador", "instante", "Nada mas"} {
		if !strings.Contains(QueSeManda, quiero) {
			t.Errorf("la declaracion no dice %q: %s", quiero, QueSeManda)
		}
	}
}
