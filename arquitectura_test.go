package plazum_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// La regla de dependencias del nucleo, verificada leyendo el AST.
//
// El documento la afirmaba y no existia. Ahora existe: los paquetes del nucleo
// no pueden importar HTTP, base de datos, red, ni nada fuera de la biblioteca
// estandar. Si alguien mete un cliente de LLM en `ventana`, esto falla.
//
// BARRIDO DE MUTACION. El detector anterior daba verde con estas cinco cosas
// delante, y las cinco las escribe un humano sin mala intencion:
//
//	1. un import externo en un _test.go de nucleo/ (los tests estaban excluidos,
//	   que es exactamente el fallo que ya nos mordio una vez). El paquete
//	   compila con la dependencia dentro, y "en nucleo/, cero, para siempre"
//	   deja de ser cierto.
//	2. un fichero .go colgado directamente de nucleo/, sin subpaquete.
//	3. un subpaquete anidado, nucleo/ledger/interno/, a profundidad 2.
//	4. importar plazum/adaptadores/tsa, que es "plazum/" y por tanto pasaba, y
//	   arrastra pkcs7 y timestamp al nucleo por la puerta de atras.
//	5. leer el reloj esquivando la cadena "time.Now()": alias de import,
//	   referencia sin parentesis, o time.Since.

// prohibidosSiempre son paquetes de la biblioteca estandar que el nucleo no
// puede usar ni en produccion ni en sus tests: hablan con el exterior, ejecutan
// procesos o esquivan el sistema de tipos.
var prohibidosSiempre = []string{
	"net", "net/http", "database/sql", "os/exec", "os/signal",
	"log/slog", "plugin", "syscall", "unsafe",
}

// prohibidosEnProduccion es lo que ademas rompe el determinismo. math/rand se
// admite en los tests de propiedades de ventana, que lo siembran a mano; en el
// codigo que se publica no tiene sitio.
var prohibidosEnProduccion = []string{"math/rand"}

// importesProhibidos devuelve los imports de un fichero del nucleo que rompen
// la regla de dependencias, con el motivo. Extraida para poder demostrar con un
// control negativo que el detector salta cuando debe.
func importesProhibidos(a *ast.File, produccion bool) []string {
	lista := prohibidosSiempre
	if produccion {
		lista = append(append([]string{}, prohibidosSiempre...), prohibidosEnProduccion...)
	}
	var hallazgos []string
	for _, imp := range a.Imports {
		v := strings.Trim(imp.Path.Value, `"`)
		// Fuera de la biblioteca estandar: un import con punto en el primer
		// segmento es un dominio, o sea una dependencia externa.
		if strings.Contains(strings.SplitN(v, "/", 2)[0], ".") {
			hallazgos = append(hallazgos, v+": el nucleo no admite dependencias externas")
			continue
		}
		// Dentro del modulo, el nucleo solo puede mirar hacia el nucleo.
		// Importar plazum/adaptadores/... o plazum/superficies/... mete en el
		// nucleo, por transitividad, todo lo que esos si pueden importar.
		if strings.HasPrefix(v, "plazum/") && !strings.HasPrefix(v, "plazum/nucleo/") {
			hallazgos = append(hallazgos, v+": el nucleo solo importa plazum/nucleo/..., "+
				"si no arrastra las dependencias del adaptador por transitividad")
			continue
		}
		for _, p := range lista {
			if v == p || strings.HasPrefix(v, p+"/") {
				hallazgos = append(hallazgos, v+": el nucleo no habla con el exterior")
				break // un import se reporta una vez, aunque case con "net" y "net/http"
			}
		}
	}
	return hallazgos
}

func TestElNucleoNoImportaElExterior(t *testing.T) {
	fset := token.NewFileSet()
	for _, ruta := range ficherosDelNucleo(t, true) {
		a, err := parser.ParseFile(fset, ruta, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		produccion := !strings.HasSuffix(ruta, "_test.go")
		for _, h := range importesProhibidos(a, produccion) {
			t.Errorf("%s importa %s", ruta, h)
		}
	}
}

// Control negativo del detector de imports: se demuestra que salta con lo que
// debe y que no salta con lo legitimo. Sin esto, un verde no demuestra nada.
func TestElDetectorDeImportsSaltaCuandoDebe(t *testing.T) {
	malo := `package x

import (
	"net/http"
	_ "github.com/digitorus/pkcs7"
	tsa "plazum/adaptadores/tsa"
)
`
	bueno := `package x

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"plazum/nucleo/ledger"
)
`
	fset := token.NewFileSet()
	a, err := parser.ParseFile(fset, "malo.go", malo, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	if h := importesProhibidos(a, true); len(h) != 3 {
		t.Fatalf("el detector debia encontrar los 3 imports malos (net/http, la dependencia "+
			"externa y el adaptador) y encontro %d: %v", len(h), h)
	}
	b, err := parser.ParseFile(fset, "bueno.go", bueno, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	if h := importesProhibidos(b, true); len(h) != 0 {
		t.Fatalf("falso positivo sobre imports legitimos del nucleo: %v", h)
	}
	// Y math/rand: prohibido en produccion, admitido en los tests de propiedades.
	rnd := "package x\n\nimport \"math/rand\"\n"
	c, err := parser.ParseFile(fset, "rand.go", rnd, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	if h := importesProhibidos(c, true); len(h) != 1 {
		t.Fatalf("math/rand debia caer en produccion: %v", h)
	}
	if h := importesProhibidos(c, false); len(h) != 0 {
		t.Fatalf("math/rand no debe caer en un test: %v", h)
	}
}

// ---------------------------------------------------------------------------
// El reloj
// ---------------------------------------------------------------------------

// lectoresDelReloj son las funciones de time que consultan el reloj del
// sistema. Todas convierten el resultado en irreproducible, no solo Now.
var lectoresDelReloj = map[string]bool{
	"Now": true, "Since": true, "Until": true,
	"After": true, "AfterFunc": true,
	"Tick": true, "NewTicker": true, "NewTimer": true,
}

// lecturasDelReloj devuelve las lecturas del reloj del sistema de un fichero ya
// parseado. Mira el AST y no el texto, porque el detector textual anterior
// buscaba la cadena "time.Now()" y daba verde con:
//
//	import crono "time" ... crono.Now()   el alias
//	var reloj = time.Now                  la referencia sin parentesis, que es
//	                                      el patron de "asi lo falseo en tests"
//	time.Since(x)                         medir lo transcurrido es leer el reloj
//
// Y ademas saltaba con un comentario que mencionara time.Now(), que es un rojo
// que nadie sabe arreglar.
func lecturasDelReloj(a *ast.File) []string {
	locales := map[string]bool{}
	var hallazgos []string
	for _, imp := range a.Imports {
		if strings.Trim(imp.Path.Value, `"`) != "time" {
			continue
		}
		switch {
		case imp.Name == nil:
			locales["time"] = true
		case imp.Name.Name == ".":
			// Con dot import no se distingue una lectura del reloj de
			// cualquier otra llamada, asi que el import ya es el hallazgo.
			hallazgos = append(hallazgos, "import . \"time\" (dot import: hace "+
				"indistinguible la lectura del reloj)")
		case imp.Name.Name == "_":
			// Importado por efecto secundario: no se puede llamar a nada.
		default:
			locales[imp.Name.Name] = true
		}
	}
	if len(locales) == 0 {
		return hallazgos
	}
	ast.Inspect(a, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		id, ok := sel.X.(*ast.Ident)
		if ok && locales[id.Name] && lectoresDelReloj[sel.Sel.Name] {
			hallazgos = append(hallazgos, id.Name+"."+sel.Sel.Name)
		}
		return true
	})
	return hallazgos
}

// Y el nucleo no puede leer el reloj del sistema: el instante entra como dato.
//
// Se aplica al codigo de produccion. Los _test.go quedan fuera a proposito: un
// test que lea el reloj es flaky, pero no rompe la reproducibilidad del
// expediente, que es la propiedad que este invariante defiende. La regla de
// dependencias de arriba si los cubre.
func TestElNucleoNoLeeElRelojDelSistema(t *testing.T) {
	fset := token.NewFileSet()
	for _, ruta := range ficherosDelNucleo(t, false) {
		a, err := parser.ParseFile(fset, ruta, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, h := range lecturasDelReloj(a) {
			t.Errorf("%s lee el reloj del sistema (%s): el instante de evaluacion entra "+
				"como dato, si no el expediente no es reproducible", ruta, h)
		}
	}
}

// Control negativo del detector de reloj: las tres evasiones del barrido de
// mutacion tienen que saltar, y lo legitimo de time no puede saltar.
func TestElDetectorDeRelojSaltaCuandoDebe(t *testing.T) {
	fset := token.NewFileSet()

	malo := `package x

import (
	"time"

	crono "time"
)

var reloj = time.Now

func a() time.Time      { return time.Now() }
func b() time.Time      { return crono.Now() }
func c(t time.Time) time.Duration { return time.Since(t) }
`
	a, err := parser.ParseFile(fset, "malo.go", malo, 0)
	if err != nil {
		t.Fatal(err)
	}
	h := lecturasDelReloj(a)
	if len(h) != 4 {
		t.Fatalf("el detector debia encontrar 4 lecturas del reloj (la referencia sin "+
			"parentesis, las dos llamadas y el Since) y encontro %d: %v", len(h), h)
	}

	// Lo legitimo: tipos, constantes, aritmetica y fechas dadas como dato.
	bueno := `package x

import "time"

const Plazo = 72 * time.Hour

func vence(desde time.Time) time.Time {
	return desde.Add(Plazo).In(time.UTC)
}

func fija() time.Time { return time.Date(2022, time.May, 5, 0, 0, 0, 0, time.UTC) }

func lee(s string) (time.Time, error) { return time.Parse(time.RFC3339, s) }
`
	b, err := parser.ParseFile(fset, "bueno.go", bueno, 0)
	if err != nil {
		t.Fatal(err)
	}
	if h := lecturasDelReloj(b); len(h) != 0 {
		t.Fatalf("falso positivo sobre uso legitimo de time: %v", h)
	}

	// Y el comentario que menciona el reloj ya no es un rojo.
	coment := "package x\n\n// aqui NO se llama a time.Now()\nfunc f() {}\n"
	c, err := parser.ParseFile(fset, "coment.go", coment, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if h := lecturasDelReloj(c); len(h) != 0 {
		t.Fatalf("falso positivo sobre un comentario: %v", h)
	}
}

// ---------------------------------------------------------------------------
// El recorrido
// ---------------------------------------------------------------------------

// ficherosGoBajo recorre el arbol ENTERO de una raiz y devuelve sus .go: los de
// la raiz, los de los subpaquetes y los de los subpaquetes de los subpaquetes.
// testdata se salta, que es datos y no codigo.
//
// La version anterior listaba solo los directorios de primer nivel de nucleo/ y
// leia cada uno sin recursion, asi que nucleo/reloj.go (un fichero colgado de la
// raiz) y nucleo/ledger/interno/reloj.go (un subpaquete anidado) quedaban fuera
// de toda vigilancia. Las dos mutaciones daban verde.
func ficherosGoBajo(raiz string, conTests bool) ([]string, error) {
	var out []string
	err := filepath.WalkDir(raiz, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") {
			return nil
		}
		if !conTests && strings.HasSuffix(p, "_test.go") {
			return nil
		}
		out = append(out, filepath.ToSlash(p))
		return nil
	})
	return out, err
}

// ficherosDelNucleo aplica el recorrido a nucleo/ y ademas se niega a pasar en
// blanco: un error de lectura o un arbol sospechosamente pequeno es un fallo,
// no un silencio. Si el directorio se renombra, el test no puede dar verde.
func ficherosDelNucleo(t *testing.T, conTests bool) []string {
	t.Helper()
	out, err := ficherosGoBajo("nucleo", conTests)
	if err != nil {
		t.Fatalf("no puedo recorrer nucleo/: %v", err)
	}
	dirs := map[string]bool{}
	for _, f := range out {
		dirs[filepath.ToSlash(filepath.Dir(f))] = true
	}
	minimo := 12
	if conTests {
		minimo = 30
	}
	if len(out) < minimo {
		t.Fatalf("nucleo/ tiene %d ficheros .go y esperaba al menos %d: renombrado, "+
			"o el recorrido ha dejado de recorrer?", len(out), minimo)
	}
	if len(dirs) < 6 {
		t.Fatalf("nucleo/ tiene codigo en %d directorios, esperaba al menos 6: renombrado?",
			len(dirs))
	}
	return out
}

// Control negativo del recorrido, sobre un arbol sintetico con las tres formas
// que el recorrido anterior no veia: fichero en la raiz, subpaquete anidado y
// _test.go. Sin esto, los detectores de arriba podrian estar pasando por no
// mirar nada.
func TestElRecorridoDelNucleoVeTodoElArbol(t *testing.T) {
	raiz := t.TempDir()
	escribir := func(rel string) {
		p := filepath.Join(raiz, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("package x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	escribir("raiz.go")                  // colgado de la raiz
	escribir("paquete/fichero.go")       // el caso normal
	escribir("paquete/fichero_test.go")  // un test
	escribir("paquete/interno/hondo.go") // subpaquete anidado
	escribir("paquete/testdata/no.go")   // datos: no es codigo
	escribir("paquete/LEEME.md")         // no es .go

	todos, err := ficherosGoBajo(raiz, true)
	if err != nil {
		t.Fatal(err)
	}
	visto := map[string]bool{}
	for _, f := range todos {
		visto[strings.TrimPrefix(f, filepath.ToSlash(raiz)+"/")] = true
	}
	for _, q := range []string{"raiz.go", "paquete/fichero.go",
		"paquete/fichero_test.go", "paquete/interno/hondo.go"} {
		if !visto[q] {
			t.Errorf("el recorrido no ve %s; ve %v", q, todos)
		}
	}
	if visto["paquete/testdata/no.go"] {
		t.Error("el recorrido entra en testdata, que son datos y no codigo")
	}

	sinTests, err := ficherosGoBajo(raiz, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(sinTests) != len(todos)-1 {
		t.Errorf("sin tests esperaba %d ficheros y hay %d: %v", len(todos)-1, len(sinTests), sinTests)
	}

	// Y sobre el arbol de verdad: los _test.go tienen que estar entrando.
	if len(ficherosDelNucleo(t, true)) <= len(ficherosDelNucleo(t, false)) {
		t.Error("ningun _test.go de nucleo/ esta entrando en el recorrido")
	}
}
