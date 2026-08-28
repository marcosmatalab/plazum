package plazum

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/internal/modulo"
)

// La ruta del modulo se LEE de go.mod. No se escribe en ningun otro sitio.
//
// EL HALLAZGO, y es de los que solo se ven una vez. El 28-08-2026 el modulo
// paso de `plazum` a `github.com/marcosmatalab/plazum`, porque con un modulo
// desnudo nadie puede hacer `go install` y el repositorio acababa de hacerse
// publico. Cinco puertas llevaban la ruta vieja escrita a mano para decidir
// que codigo es de casa:
//
//	arquitectura_test.go      el nucleo solo importa <modulo>/nucleo/...
//	dependencias_test.go      el binario no lleva nada que no sea stdlib o de casa
//	ia_test.go                el nucleo no importa <modulo>/puertos (dos sitios)
//	latido/frontera_test.go   el latido no mira al nucleo ni a las superficies
//
// DOS DE LAS CINCO SE HABRIAN QUEDADO VERDES, y esto no es una hipotesis: se
// midio. Con `esElPuertoDeIA` devuelto a la ruta vieja cableada, el fichero
// compila y:
//
//	TestElNucleoNoConoceLaIA            --- PASS   <- la PUERTA, verde y ciega
//	TestElDetectorDeImportDeIAFunciona  --- FAIL   <- su control negativo
//
// La puerta que vigila el invariante 9 (el nucleo no conoce la IA) pasa
// tranquilamente, porque su detector ya no reconoce ningun import: no hay nada
// que detectar cuando se busca una cadena que ya no existe. Lo mismo le pasa a
// la frontera del latido. **Lo unico que las caza es su control negativo**, y
// esa es la vindicacion mas limpia que ha tenido en este repositorio la regla
// de que toda puerta nace con su fallo demostrado: aqui el control negativo no
// fue una formalidad de la primera escritura, fue el unico testigo.
//
// Las otras tres (nucleo, dependencias, y el control de la de IA) salieron
// rojas y ruidosas.
//
// O sea: la ruta del modulo estaba en go.mod Y en cinco tests, que es la
// segunda lista que este repositorio lleva catorce hallazgos prohibiendo. Esta
// puerta cierra la unica forma de volver a escribirla.
//
// QUE MIRA Y QUE NO. Mira LITERALES DE CADENA que no sean rutas de import. No
// mira los comentarios, y no es un descuido: media docena de ficheros (este
// incluido) explican en prosa por que el nucleo no puede importar tal cosa, y
// un detector que leyera comentarios cazaria la explicacion del invariante
// como si fuera su violacion. Es el falso positivo que ya desactivo una vez el
// detector del coste cuadratico de ber2der.
func TestNadieCableaLaRutaDelModulo(t *testing.T) {
	mod, err := modulo.Ruta()
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	mirados := 0
	for _, ruta := range ficherosGoDe(t, ".") {
		a, err := parser.ParseFile(fset, ruta, nil, 0) // 0 = sin comentarios
		if err != nil {
			t.Fatalf("no puedo parsear %s: %v", ruta, err)
		}
		mirados++
		for _, lit := range literalesConLaRuta(a, mod) {
			t.Errorf(`%s cablea la ruta del modulo en un literal: %s

  La ruta del modulo vive en go.mod y en ningun otro sitio. Escrita aqui, es una
  segunda copia, y una segunda copia no se rompe cuando la primera cambia: se
  queda vieja y sigue dando verde. Paso el 28-08-2026 con el renombrado.

  Arreglo: internal/modulo.Ruta() la lee de go.mod; EsDeCasa e Interno hacen las
  dos preguntas que se suelen querer hacer con ella. Para una fuente sintetica
  de un control negativo, se compone: "package x\n\nimport _ \"" + mod + "/..."`,
				ruta, lit)
		}
	}
	// Suelo: si el recorrido dejara de encontrar ficheros, esto saldria verde
	// sin haber mirado ni uno.
	if mirados < 100 {
		t.Fatalf("solo se han parseado %d ficheros .go: el recorrido no esta viendo el "+
			"repositorio y esta puerta estaria auditando el vacio", mirados)
	}
	// El log tranquilizador SOLO si de verdad no ha saltado nada. Un "ninguno
	// cablea" impreso al lado de una lista de errores es una afirmacion falsa
	// en la salida de la puerta, y este repositorio ya tuvo una (M14: el
	// comentario de un test decia que protegia algo que no protegia).
	if !t.Failed() {
		t.Logf("%d ficheros .go mirados, ninguno cablea %q", mirados, mod)
	}
}

// literalesConLaRuta devuelve los literales de cadena del fichero que contienen
// la ruta del modulo Y NO SON un import.
//
// Vive fuera del test para que el control negativo pase por este mismo codigo:
// un detector que solo se ha ejecutado contra el caso bueno no ha demostrado
// que sepa decir que no.
func literalesConLaRuta(a *ast.File, mod string) []string {
	// Las posiciones de los imports, que son literales legitimos.
	deImport := map[token.Pos]bool{}
	for _, imp := range a.Imports {
		deImport[imp.Path.Pos()] = true
	}
	var out []string
	ast.Inspect(a, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING || deImport[lit.Pos()] {
			return true
		}
		v, err := strconv.Unquote(lit.Value)
		if err != nil {
			v = lit.Value // un literal que no se puede desescapar se mira crudo
		}
		if strings.Contains(v, mod) {
			out = append(out, strconv.Quote(recorta(v, 70)))
		}
		return true
	})
	return out
}

func recorta(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// CONTROL NEGATIVO. Se le da un fuente con la ruta en las tres formas que
// importan y se exige que distinga: el import es legitimo, el literal no, y el
// comentario no se mira.
func TestElDetectorDeRutaCableadaDistingueImportDeLiteral(t *testing.T) {
	mod, err := modulo.Ruta()
	if err != nil {
		t.Fatal(err)
	}
	fuente := "package x\n\n" +
		"import (\n" +
		"\t\"strings\"\n" +
		"\t\"" + mod + "/nucleo/ventana\"\n" + // legitimo: es un import
		")\n\n" +
		"// El nucleo solo importa " + mod + "/nucleo/... y esto es un comentario.\n" +
		"func f(v string) bool {\n" +
		"\treturn strings.HasPrefix(v, \"" + mod + "/nucleo/\")\n" + // cableado
		"}\n"

	fset := token.NewFileSet()
	a, err := parser.ParseFile(fset, "sintetico.go", fuente, 0)
	if err != nil {
		t.Fatal(err)
	}
	h := literalesConLaRuta(a, mod)
	if len(h) != 1 {
		t.Fatalf("el detector tenia que encontrar EXACTAMENTE el literal cableado y "+
			"encontro %d: %v.\n"+
			"  0 = no ve el literal, y entonces la puerta esta dando verde sin mirar.\n"+
			"  2 = tambien cuenta el import, y entonces protesta de todo fichero del\n"+
			"      repositorio y acaba desactivada la primera semana.", len(h), h)
	}

	// Y no protesta de un fichero que solo importa.
	limpio := "package x\n\nimport \"" + mod + "/nucleo/ventana\"\n\nvar _ = ventana.Plazo{}\n"
	b, err := parser.ParseFile(fset, "limpio.go", limpio, 0)
	if err != nil {
		t.Fatal(err)
	}
	if h := literalesConLaRuta(b, mod); len(h) != 0 {
		t.Fatalf("falso positivo sobre un fichero que solo importa: %v", h)
	}
}
