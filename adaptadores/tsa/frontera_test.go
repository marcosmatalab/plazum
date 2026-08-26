package tsa

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// La frontera con el codigo ajeno de este adaptador, vigilada sobre el AST.
//
// ESTA PUERTA ERA OTRA HASTA EL 26-08-2026, y conviene contarlo. Se llamaba
// TestTimestampSoloConstruyeLaPeticion y vigilaba que
// `github.com/digitorus/timestamp` se usara SOLO para armar la consulta RFC
// 3161, porque cualquier otro uso volvia a meter un segundo parser del mismo
// ASN.1 en el camino del expediente.
//
// **Se murio de exito.** Se escribieron las cuarenta lineas del TimeStampReq
// (rfc3161_peticion.go), la dependencia salio entera de go.mod, y con ella
// salio `github.com/digitorus/pkcs7`, que era timestamp quien lo arrastraba. La
// propia puerta lo dijo al quedarse sin nada que vigilar, que es como tenia que
// terminar:
//
//	ningun fichero de este paquete usa github.com/digitorus/timestamp. O la
//	dependencia se ha ido del todo (entonces borra esta puerta Y su fila de
//	DEPENDENCIAS.md, que es el objetivo declarado)
//
// Lo que vigila AHORA es lo que queda: que nadie importe el pkcs7 de aguas
// arriba en vez de la copia vendorizada. La otra mitad, que el modulo no vuelva
// a go.mod ni al binario, esta en la raiz (dependencias_test.go), porque eso no
// es de un paquete sino del modulo entero.

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
