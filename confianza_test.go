package dutiq_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// La invariante que sale del bloqueante de la revision hostil de la etapa 1.
//
// El bug no era un caso borde, era una clase: TODO lo que el receptor debe
// aportar estaba guardado como campo que escribe el emisor. Pasó dos veces
// (expediente.AnclasDeConfianza y ledger.ClavesConfiables) y las dos con un
// comentario encima diciendo que lo aportaba el receptor. El comentario no
// impide nada; esto si.
//
// Regla: un campo de nucleo/ cuyo nombre dice que es confianza del receptor no
// puede serializarse. Si viaja en el fichero, lo escribe el emisor, y entonces
// la verificacion compara al emisor consigo mismo.
//
// Excepcion explicita: los que terminan en "Declarado" y sus tres variantes de
// genero y numero SI se serializan, porque son justo lo contrario: lo que el
// emisor DICE haber usado, que se contrasta contra lo que trae el receptor y no
// decide nada.

// nombresDeConfianza son las raices que marcan un campo como confianza del
// receptor. Ampliar esta lista al anadir un concepto nuevo de confianza.
var nombresDeConfianza = []string{
	"Ancla", "Anclas",
	"Confiable", "Confiables", "Confianza",
	"ClaveOperador", "ClavesConfiables",
	"VerificarSello",
}

// esConfianzaDelReceptor decide si el nombre de un campo dice que es confianza.
//
// La excepcion se comprueba con las cuatro terminaciones de "declarado", no con
// dos de ellas repetidas: aqui habia "Declaradas" dos veces y "Declarada" y
// "Declarados" en ninguna, asi que un campo llamado AnclaDeclarada se habria
// tomado por confianza del receptor y habria roto el build sin motivo.
func esConfianzaDelReceptor(nombre string) bool {
	for _, d := range []string{"Declarado", "Declarados", "Declarada", "Declaradas"} {
		if strings.HasSuffix(nombre, d) {
			return false
		}
	}
	for _, r := range nombresDeConfianza {
		if strings.Contains(nombre, r) {
			return true
		}
	}
	return false
}

// confianzaSerializada devuelve los campos de confianza que llevan etiqueta
// json distinta de "-", o sea los que viajarian dentro del fichero.
func confianzaSerializada(a *ast.File) []string {
	var hallazgos []string
	ast.Inspect(a, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok || st.Fields == nil {
			return true
		}
		for _, campo := range st.Fields.List {
			if campo.Tag == nil {
				continue // sin etiqueta no se serializa por nombre json explicito
			}
			valor, err := strconv.Unquote(campo.Tag.Value)
			if err != nil {
				continue
			}
			etiqueta := reflect.StructTag(valor).Get("json")
			if etiqueta == "" || etiqueta == "-" || strings.HasPrefix(etiqueta, "-,") {
				continue
			}
			for _, nom := range campo.Names {
				if esConfianzaDelReceptor(nom.Name) {
					hallazgos = append(hallazgos,
						ts.Name.Name+"."+nom.Name+" (json:\""+etiqueta+"\")")
				}
			}
		}
		return true
	})
	return hallazgos
}

func TestLaConfianzaNoViajaEnElFichero(t *testing.T) {
	fset := token.NewFileSet()
	var todos []string
	for _, dir := range subdirectoriosDelNucleo(t) {
		entradas, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entradas {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			ruta := filepath.Join(dir, e.Name())
			a, err := parser.ParseFile(fset, ruta, nil, parser.ParseComments)
			if err != nil {
				t.Fatal(err)
			}
			for _, h := range confianzaSerializada(a) {
				todos = append(todos, ruta+": "+h)
			}
		}
	}
	if len(todos) > 0 {
		t.Fatalf("hay %d campo(s) de confianza del receptor serializados en el fichero:\n  %s\n\n"+
			"Si viaja en el fichero, lo escribe el emisor, y la verificacion acaba comparandolo "+
			"consigo mismo. Es el bloqueante de la etapa 1, que aparecio dos veces. Muevelo a "+
			"ContextoReceptor o a ledger.Confianza, o renombralo a ...Declaradas si de verdad es "+
			"lo que el emisor DICE haber usado.",
			len(todos), strings.Join(todos, "\n  "))
	}
}

// Control negativo: se demuestra que el detector salta cuando debe. Sin esto,
// el test de arriba podria estar pasando por no mirar nada.
func TestElDetectorDeConfianzaSaltaCuandoDebe(t *testing.T) {
	fset := token.NewFileSet()
	fuente := `package x

type Malo struct {
	AnclasDeConfianza map[string]string ` + "`json:\"anclas_de_confianza\"`" + `
	ClavesConfiables  []string          ` + "`json:\"claves_confiables,omitempty\"`" + `
	ClaveOperador     []byte            ` + "`json:\"clave_operador\"`" + `
}

type Bueno struct {
	AnclasDeclaradas map[string]string ` + "`json:\"anclas_declaradas\"`" + `
	ClavesDeclaradas []string          ` + "`json:\"claves_declaradas\"`" + `
	Entradas         []string          ` + "`json:\"entradas\"`" + `
	NoSeSerializa    []string          ` + "`json:\"-\"`" + `
}
`
	a, err := parser.ParseFile(fset, "sintetico.go", fuente, 0)
	if err != nil {
		t.Fatal(err)
	}
	// El conjunto EXACTO, no un recuento mas una lista de subcadenas prohibidas:
	// con eso, un detector que devolviera tres veces el mismo campo de Malo y se
	// dejara ClaveOperador fuera pasaria igual (son 3 y ninguno dice
	// "Declaradas"), que es el hueco que este control negativo existe para
	// cerrar. Las etiquetas van completas porque son parte de lo que se reporta.
	quiero := map[string]bool{
		`Malo.AnclasDeConfianza (json:"anclas_de_confianza")`:        true,
		`Malo.ClavesConfiables (json:"claves_confiables,omitempty")`: true,
		`Malo.ClaveOperador (json:"clave_operador")`:                 true,
	}
	h := confianzaSerializada(a)
	if len(h) != len(quiero) {
		t.Fatalf("el detector debia encontrar los %d campos de Malo y encontro %d: %v",
			len(quiero), len(h), h)
	}
	visto := map[string]bool{}
	for _, x := range h {
		if !quiero[x] {
			t.Fatalf("falso positivo: %s. Lo declarado y lo no serializado son legitimos", x)
		}
		if visto[x] {
			t.Fatalf("el detector repite %s: repetir uno tapa que falta otro", x)
		}
		visto[x] = true
	}
}
