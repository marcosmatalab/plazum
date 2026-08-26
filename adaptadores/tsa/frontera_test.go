package tsa

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// La frontera con github.com/digitorus/timestamp, vigilada sobre el AST.
//
// LA REGLA, y viene de una decision, no de una preferencia: **`timestamp` se
// queda SOLO como constructor de la peticion RFC 3161**. Construir un
// TimeStampReq no es frontera de confianza, porque los bytes los ponemos
// nosotros y quien los lee es la TSA. Todo lo demas (leer la respuesta, sacar el
// TSTInfo, decidir que se sello y cuando) es nuestro, con `encoding/asn1` sobre
// el contenido que nuestro propio pkcs7 ya extrajo.
//
// POR QUE HACE FALTA UNA PUERTA Y NO BASTA CON EL COMENTARIO. La forma en que
// esto se deshace no es una decision: es una linea. Alguien necesita un campo
// del TSTInfo un martes, ve que `timestamp` ya esta importada, escribe
// `timestamp.Parse(token)` y en ese momento vuelve a haber dos parsers sobre
// los mismos bytes y el pendiente 53 resucita sin que nadie lo note. El coste
// de esa linea es cero y el de descubrirla, meses.
//
// Se vigila por AST y no por `grep` a proposito: un `grep` de "timestamp."
// tambien casa con la palabra dentro de un comentario, y este fichero esta
// lleno de comentarios que la nombran.
func TestTimestampSoloConstruyeLaPeticion(t *testing.T) {
	// Lo unico que se admite del paquete. Request es la estructura de la
	// peticion y Marshal su metodo; ParseRequest lo usan los tests para hacer
	// de TSA de mentira, que es el otro lado del cable y no decide nada nuestro.
	permitidoEnProduccion := map[string]bool{"Request": true}
	permitidoEnPruebas := map[string]bool{"Request": true, "ParseRequest": true, "Timestamp": true}

	const modulo = "github.com/digitorus/timestamp"
	fset := token.NewFileSet()
	paquete, err := parser.ParseDir(fset, ".", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("no puedo parsear el paquete: %v", err)
	}

	vistos := 0
	for _, p := range paquete {
		for ruta, fichero := range p.Files {
			alias := aliasDelImport(fichero, modulo)
			if alias == "" {
				continue // este fichero no lo importa
			}
			esPrueba := strings.HasSuffix(ruta, "_test.go")
			permitido := permitidoEnProduccion
			donde := "codigo de produccion"
			if esPrueba {
				permitido = permitidoEnPruebas
				donde = "un test"
			}
			for _, usado := range selectoresDe(fichero, alias) {
				vistos++
				if permitido[usado] {
					continue
				}
				t.Errorf(`%s usa %s.%s, y en %s solo se admite %v.

  timestamp se quedo SOLO como constructor de la peticion RFC 3161 el
  26-08-2026, cuando se quito el segundo parser: el TSTInfo se lee ahora con
  encoding/asn1 sobre el p7.Content de la copia vendorizada
  (adaptadores/tsa/rfc3161.go), asi que hay UN parser y no dos.

  Volver a llamar a %s.%s pone otra vez dos lecturas independientes de los
  mismos bytes del expediente, sin ninguna identidad dentro de lo firmado que
  las ate (invariante 7 de CLAUDE.md). Y timestamp.Parse ademas llama a
  p7.Verify(), que es la funcion que el recorte 1 quito de nuestra copia porque
  desactiva la verificacion de cadena.

  Arreglo: si hace falta un campo del TSTInfo que no esta en el tipo Sello,
  anadirlo a infoSello en rfc3161.go. Son cinco lineas y no traen otro parser.`,
					filepath.Base(ruta), alias, usado, donde, claves(permitido), alias, usado)
			}
		}
	}
	// Suelo: si nadie usa el paquete, o el recorrido ha dejado de encontrarlo,
	// esta puerta no vigila nada y "sin hallazgos" se leeria como "todo en
	// orden". Es la misma trampa que el canario del cribador de marcas.
	if vistos == 0 {
		t.Fatal("ningun fichero de este paquete usa github.com/digitorus/timestamp. O la " +
			"dependencia se ha ido del todo (entonces borra esta puerta Y su fila de " +
			"DEPENDENCIAS.md, que es el objetivo declarado) o el recorrido del AST ha " +
			"dejado de encontrarla, y eso deja la frontera sin vigilancia")
	}
	if !t.Failed() {
		t.Logf("%d usos de timestamp, todos dentro de lo admitido", vistos)
	}
}

// aliasDelImport devuelve con que nombre se usa un modulo en este fichero, o
// vacio si no se importa. Se mira el alias y no el ultimo tramo de la ruta
// porque un import con nombre propio se saltaria la comprobacion entera.
func aliasDelImport(f *ast.File, modulo string) string {
	for _, imp := range f.Imports {
		ruta := strings.Trim(imp.Path.Value, `"`)
		if ruta != modulo {
			continue
		}
		if imp.Name != nil {
			if imp.Name.Name == "_" || imp.Name.Name == "." {
				// Un import en blanco no usa nada; uno con punto mete los
				// nombres en el ambito y esta comprobacion no lo puede seguir.
				return imp.Name.Name
			}
			return imp.Name.Name
		}
		if i := strings.LastIndex(ruta, "/"); i >= 0 {
			return ruta[i+1:]
		}
		return ruta
	}
	return ""
}

// selectoresDe devuelve los identificadores usados como `alias.X`.
func selectoresDe(f *ast.File, alias string) []string {
	visto := map[string]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		id, ok := sel.X.(*ast.Ident)
		if !ok || id.Name != alias {
			return true
		}
		visto[sel.Sel.Name] = true
		return true
	})
	out := make([]string, 0, len(visto))
	for k := range visto {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func claves(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Y la otra mitad, que es la que de verdad se buscaba: ningun fichero de
// produccion de este adaptador importa el pkcs7 de AGUAS ARRIBA.
//
// Es lo que hace que el pendiente 53 este muerto y no vigilado. Mientras
// `timestamp` siga importada, el modulo sigue en el grafo (lo dice
// DEPENDENCIAS.md como objetivo abierto), pero ningun codigo NUESTRO lo llama.
func TestNingunFicheroNuestroImportaElPkcs7DeAguasArriba(t *testing.T) {
	const modulo = "github.com/digitorus/pkcs7"
	entradas, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	mirados := 0
	for _, e := range entradas {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		// LOS TESTS SE EXCLUYEN, Y NO ES UNA GRIETA. dependencias_test.go
		// importa el pkcs7 de aguas arriba A PROPOSITO: es la puerta que vigila
		// que la version transitiva no sea la que revienta, y para vigilarla
		// tiene que tocarla. Una puerta que llega a lo que vigila por un camino
		// prestado esta midiendo el camino, no lo vigilado.
		if strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		mirados++
		f, err := parser.ParseFile(fset, e.Name(), nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("no puedo parsear %s: %v", e.Name(), err)
		}
		if alias := aliasDelImport(f, modulo); alias != "" {
			t.Errorf(`%s importa %s directamente.

  Esa es la copia de aguas arriba, no la vendorizada. Importarla vuelve a meter
  en el binario dos parsers del mismo ASN.1 y deshace el trabajo del 26-08-2026.
  Arreglo: usar plazum/adaptadores/tsa/internal/pkcs7.`, e.Name(), modulo)
		}
	}
	if mirados < 3 {
		t.Fatalf("solo se han mirado %d ficheros .go: si el paquete se ha movido, esta "+
			"puerta estaria auditando el vacio", mirados)
	}
}
