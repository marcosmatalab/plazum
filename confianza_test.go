package plazum_test

import (
	"go/ast"
	"go/parser"
	"go/token"
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
// Excepcion explicita: los que terminan en "Declaradas"/"Declarado" SI se
// serializan, porque son justo lo contrario: lo que el emisor DICE haber
// usado, que se contrasta contra lo que trae el receptor y no decide nada.

// nombresDeConfianza son las raices que marcan un campo como confianza del
// receptor. Ampliar esta lista al anadir un concepto nuevo de confianza.
var nombresDeConfianza = []string{
	"Ancla", "Anclas",
	"Confiable", "Confiables", "Confianza",
	"ClaveOperador", "ClavesConfiables",
	"VerificarSello",
}

// esConfianzaDelReceptor decide si el nombre de un campo dice que es confianza.
func esConfianzaDelReceptor(nombre string) bool {
	if strings.HasSuffix(nombre, "Declaradas") || strings.HasSuffix(nombre, "Declarados") ||
		strings.HasSuffix(nombre, "Declarado") || strings.HasSuffix(nombre, "Declarada") {
		return false
	}
	for _, r := range nombresDeConfianza {
		if strings.Contains(nombre, r) {
			return true
		}
	}
	return false
}

// nombreDeCampo devuelve los nombres de un campo de struct. Un campo embebido no
// tiene Names, y su nombre a efectos de json es el del tipo: el barrido de
// mutacion embebio el tipo AnclasDeConfianza con etiqueta json y el detector no
// lo veia, porque solo miraba campo.Names.
func nombreDeCampo(campo *ast.Field) []string {
	if len(campo.Names) > 0 {
		var out []string
		for _, n := range campo.Names {
			out = append(out, n.Name)
		}
		return out
	}
	tipo := campo.Type
	if e, ok := tipo.(*ast.StarExpr); ok {
		tipo = e.X
	}
	switch x := tipo.(type) {
	case *ast.Ident:
		return []string{x.Name}
	case *ast.SelectorExpr:
		return []string{x.Sel.Name}
	}
	return nil
}

// hogaresDeLaConfianza son los DOS tipos que existen precisamente para guardar
// lo que aporta el receptor, cualificados por paquete. No viajan: se pasan como
// argumento a Verificar, viven en la memoria de quien recibe el expediente y
// ninguno de sus campos lleva etiqueta json. Son el destino que el mensaje de
// error de este test recomienda, asi que no pueden ser tambien su hallazgo.
//
// La exencion no es gratis: un campo de estos tipos CON etiqueta json si es un
// hallazgo, porque una etiqueta json solo se pone para serializar, y serializar
// el contexto del receptor es volver al bloqueante de la etapa 1.
var hogaresDeLaConfianza = map[string]bool{
	"expediente.ContextoReceptor": true,
	"ledger.Confianza":            true,
}

// confianzaSerializada devuelve los campos de confianza del receptor que
// viajarian dentro del fichero.
//
// Un campo viaja si es exportado y NO lleva `json:"-"`. Esa es la regla de
// encoding/json y es la que se comprueba aqui.
//
// BARRIDO DE MUTACION, el hallazgo grande de este fichero: la version anterior
// hacia `if campo.Tag == nil { continue }` con el comentario "sin etiqueta no se
// serializa por nombre json explicito". Es falso. Un campo exportado sin
// etiqueta se serializa igual, con el nombre del campo. Se comprobo:
// `AnclasDeConfianza map[string]string` sin etiqueta salia en el JSON del
// expediente como "AnclasDeConfianza":{...} y el test daba verde. O sea que
// para meter la confianza dentro del fichero bastaba con borrar la etiqueta.
//
// Los campos no exportados quedan fuera a proposito: encoding/json no los toca,
// asi que no pueden viajar.
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
		hogar := hogaresDeLaConfianza[a.Name.Name+"."+ts.Name.Name]
		for _, campo := range st.Fields.List {
			etiqueta := ""
			conEtiquetaJSON := false
			if campo.Tag != nil {
				if valor, err := strconv.Unquote(campo.Tag.Value); err == nil {
					etiqueta = reflect.StructTag(valor).Get("json")
					_, conEtiquetaJSON = reflect.StructTag(valor).Lookup("json")
				}
			}
			// El hogar de la confianza no viaja... mientras nadie le ponga una
			// etiqueta json, que es la senal de que alguien quiere serializarlo.
			if hogar && !conEtiquetaJSON {
				continue
			}
			// La unica forma de que un campo exportado no viaje.
			if etiqueta == "-" || strings.HasPrefix(etiqueta, "-,") {
				continue
			}
			comoViaja := "sin etiqueta json, viaja con el nombre del campo"
			if etiqueta != "" {
				comoViaja = "json:\"" + etiqueta + "\""
			}
			for _, nom := range nombreDeCampo(campo) {
				if !ast.IsExported(nom) {
					continue // encoding/json no serializa lo no exportado
				}
				if esConfianzaDelReceptor(nom) {
					hallazgos = append(hallazgos,
						ts.Name.Name+"."+nom+" ("+comoViaja+")")
				}
			}
		}
		return true
	})
	return hallazgos
}

// Se mira el codigo de produccion del nucleo ENTERO, raiz y subpaquetes
// anidados incluidos (ver ficherosDelNucleo en arquitectura_test.go). Los
// _test.go quedan fuera porque un struct declarado en un test no se compila en
// el binario y por tanto no puede definir el formato del fichero publicado; y
// porque los tests hostiles necesitan poder declarar structs malos a proposito.
func TestLaConfianzaNoViajaEnElFichero(t *testing.T) {
	fset := token.NewFileSet()
	var todos []string
	for _, ruta := range ficherosDelNucleo(t, false) {
		a, err := parser.ParseFile(fset, ruta, nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		for _, h := range confianzaSerializada(a) {
			todos = append(todos, ruta+": "+h)
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

type AnclasDeConfianza map[string]string

type Malo struct {
	AnclasDeConfianza map[string]string ` + "`json:\"anclas_de_confianza\"`" + `
	ClavesConfiables  []string          ` + "`json:\"claves_confiables,omitempty\"`" + `
	ClaveOperador     []byte            ` + "`json:\"clave_operador\"`" + `
}

// MaloSinEtiqueta es la mutacion que la version anterior del detector no veia:
// un campo exportado sin etiqueta json viaja igual, con el nombre del campo.
type MaloSinEtiqueta struct {
	AnclasDeConfianza map[string]string
	ClavesConfiables  []string ` + "`xml:\"claves\"`" + `
}

// MaloEmbebido: sin nombre de campo propio, el nombre a efectos de json es el
// del tipo, y tambien viaja.
type MaloEmbebido struct {
	AnclasDeConfianza ` + "`json:\"anclas\"`" + `
}

type Bueno struct {
	AnclasDeclaradas  map[string]string ` + "`json:\"anclas_declaradas\"`" + `
	ClavesDeclaradas  []string          ` + "`json:\"claves_declaradas\"`" + `
	Entradas          []string          ` + "`json:\"entradas\"`" + `
	NoSeSerializa     []string          ` + "`json:\"-\"`" + `
	SinNombreNiOpcion []string          ` + "`json:\"-,omitempty\"`" + `
	anclasDeConfianza map[string]string
}
`
	a, err := parser.ParseFile(fset, "sintetico.go", fuente, 0)
	if err != nil {
		t.Fatal(err)
	}
	h := confianzaSerializada(a)
	// 3 de Malo, 2 de MaloSinEtiqueta y 1 de MaloEmbebido.
	if len(h) != 6 {
		t.Fatalf("el detector debia encontrar los 6 campos que viajan y encontro %d: %v", len(h), h)
	}
	for _, x := range h {
		if strings.Contains(x, "Declaradas") || strings.Contains(x, "Entradas") ||
			strings.Contains(x, "NoSeSerializa") || strings.Contains(x, "SinNombreNiOpcion") ||
			strings.Contains(x, "anclasDeConfianza") {
			t.Fatalf("falso positivo: %s. Lo declarado, lo marcado json:\"-\" y lo no "+
				"exportado son legitimos", x)
		}
	}
	// Y las tres formas de viajar tienen que estar cada una en la lista.
	junto := strings.Join(h, " | ")
	for _, q := range []string{
		"Malo.AnclasDeConfianza",
		"MaloSinEtiqueta.AnclasDeConfianza",
		"MaloSinEtiqueta.ClavesConfiables",
		"MaloEmbebido.AnclasDeConfianza",
	} {
		if !strings.Contains(junto, q) {
			t.Errorf("el detector no ve %s: %s", q, junto)
		}
	}
}

// Control negativo de la exencion: el hogar de la confianza esta exento
// mientras no lleve etiqueta json, y deja de estarlo en cuanto la lleva.
func TestElHogarDeLaConfianzaSoloEstaExentoSiNoSeSerializa(t *testing.T) {
	fset := token.NewFileSet()

	limpio := `package expediente

type ContextoReceptor struct {
	Anclas           map[string]string
	ClavesConfiables []string
	ClaveOperador    []byte
	VerificarSello   func(hash, token []byte) error
}
`
	a, err := parser.ParseFile(fset, "limpio.go", limpio, 0)
	if err != nil {
		t.Fatal(err)
	}
	if h := confianzaSerializada(a); len(h) != 0 {
		t.Fatalf("el contexto del receptor no viaja y no debe ser hallazgo: %v", h)
	}

	marcado := `package expediente

type ContextoReceptor struct {
	Anclas           map[string]string ` + "`json:\"anclas\"`" + `
	ClavesConfiables []string
}
`
	b, err := parser.ParseFile(fset, "marcado.go", marcado, 0)
	if err != nil {
		t.Fatal(err)
	}
	h := confianzaSerializada(b)
	if len(h) != 1 || !strings.Contains(h[0], "ContextoReceptor.Anclas") {
		t.Fatalf("una etiqueta json dentro del hogar de la confianza tiene que saltar, "+
			"y salto %d: %v", len(h), h)
	}

	// Y la exencion es por tipo cualificado: un ContextoReceptor de otro
	// paquete no hereda el permiso.
	ajeno := strings.Replace(limpio, "package expediente", "package otro", 1)
	c, err := parser.ParseFile(fset, "ajeno.go", ajeno, 0)
	if err != nil {
		t.Fatal(err)
	}
	if h := confianzaSerializada(c); len(h) != 4 {
		t.Fatalf("la exencion se ha escapado a otro paquete: %v", h)
	}
}
