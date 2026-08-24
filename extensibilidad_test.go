package obligo

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// urnsProhibidos son identificadores de norma concretos. Si alguno aparece en el
// codigo, es que la norma esta cableada, y entonces "anadir la norma 31 no toca
// codigo" es falso.
//
// Esta es la propiedad que separa este diseno de un GRC cuyo corpus es un arbol
// plano de requisitos cargado desde una hoja de Excel: alli la temporalidad, las
// preguntas de alcance y los entregables no tienen donde declararse, asi que
// acaban en el codigo o no existen.
var urnsProhibidos = []string{
	"ens@", "rgpd@", "nis2@", "dora@", "cra@", "iso27001@", "iso22301@",
	"iso42001@", "pcidss@", "soc2@", "tisax@", "cis@", "nist80053@", "csf@",
	"lopdgdd@", "eidas@", "mica@", "psd2@", "dga@", "dataact@", "aiact@",
}

func ficherosGo(t *testing.T) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(".", func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == "corpus_datos" || d.Name() == "testdata") {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, "_test.go") {
			out = append(out, p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestNingunaNormaCableada es el test que hace verificable la extensibilidad.
// Falla el build si alguien mete el identificador de una norma en el codigo.
func TestNingunaNormaCableada(t *testing.T) {
	fset := token.NewFileSet()
	for _, f := range ficherosGo(t) {
		a, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		ast.Inspect(a, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			v := strings.ToLower(lit.Value)
			for _, u := range urnsProhibidos {
				if strings.Contains(v, u) {
					t.Errorf("%s:%d: la norma %q esta cableada en el codigo. "+
						"Toda norma vive en su paquete de datos o no vive",
						f, fset.Position(lit.Pos()).Line, u)
				}
			}
			return true
		})
	}
}

// TestNormaNuevaNoTocaCodigo comprueba la propiedad de frente: se anade un
// paquete que el codigo no ha visto nunca, escrito solo como datos, y la
// interfaz, la entrevista, los entregables y la lista de conectores cambian
// solas. Cero ficheros Go tocados.
func TestNormaNuevaNoTocaCodigo(t *testing.T) {
	dir := t.TempDir()
	escribir := func(nombre, json string) {
		d := filepath.Join(dir, nombre)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "paquete.json"), []byte(json), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	escribir("norma-a", paqueteDemo("urn:demo:a", "sistema", "categoria"))

	antes := medirDerivados(t, dir)

	// La norma 31: nunca vista por el codigo, solo datos, y ademas introduce un
	// tipo de entidad y un recurso que no existian.
	escribir("norma-31", paqueteDemo("urn:demo:31", "proveedor", "criticidad"))

	despues := medirDerivados(t, dir)

	if despues.campos <= antes.campos {
		t.Errorf("la interfaz no ha crecido: %d -> %d", antes.campos, despues.campos)
	}
	if despues.preguntas <= antes.preguntas {
		t.Errorf("la entrevista no ha crecido: %d -> %d", antes.preguntas, despues.preguntas)
	}
	if despues.trazas <= antes.trazas {
		t.Errorf("los entregables no han crecido: %d -> %d", antes.trazas, despues.trazas)
	}
	if despues.recursos <= antes.recursos {
		t.Errorf("los conectores necesarios no han crecido: %d -> %d", antes.recursos, despues.recursos)
	}
}
