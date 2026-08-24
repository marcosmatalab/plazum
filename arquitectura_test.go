package dutiq_test

import (
	"go/ast"
	"go/parser"
	"go/token"
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
func TestElNucleoNoImportaElExterior(t *testing.T) {
	nucleo := subdirectoriosDelNucleo(t)
	prohibidos := []string{"net/http", "database/sql", "net", "os/exec", "log/slog"}

	for _, dir := range nucleo {
		t.Run(dir, func(t *testing.T) {
			entradas, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range entradas {
				if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
					continue
				}
				ruta := filepath.Join(dir, e.Name())
				f, err := parser.ParseFile(token.NewFileSet(), ruta, nil, parser.ImportsOnly)
				if err != nil {
					t.Fatal(err)
				}
				for _, imp := range f.Imports {
					v := strings.Trim(imp.Path.Value, `"`)
					for _, p := range prohibidos {
						if v == p || strings.HasPrefix(v, p+"/") {
							t.Errorf("%s importa %q: el nucleo no habla con el exterior", ruta, v)
						}
					}
					if strings.Contains(v, ".") && !strings.HasPrefix(v, "dutiq/") {
						t.Errorf("%s importa %q: el nucleo no admite dependencias externas", ruta, v)
					}
				}
				_ = ast.Inspect
			}
		})
	}
}

// Y el nucleo no puede leer el reloj del sistema: el instante entra como dato.
func TestElNucleoNoLeeElRelojDelSistema(t *testing.T) {
	for _, dir := range subdirectoriosDelNucleo(t) {
		entradas, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("no puedo leer %s: %v", dir, err)
		}
		for _, e := range entradas {
			if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			ruta := filepath.Join(dir, e.Name())
			b, err := os.ReadFile(ruta)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(b), "time.Now()") {
				t.Errorf("%s llama a time.Now(): el instante de evaluacion entra como dato, "+
					"si no el expediente no es reproducible", ruta)
			}
		}
	}
}

// subdirectoriosDelNucleo enumera nucleo/ dinamicamente: todo paquete nuevo
// (certificado, historia, keystore...) nace vigilado sin tocar este test.
// Un error de lectura es un fallo, no un silencio (el arreglo del revisor:
// si el directorio se renombra, el test no puede pasar en blanco).
func subdirectoriosDelNucleo(t *testing.T) []string {
	t.Helper()
	entradas, err := os.ReadDir("nucleo")
	if err != nil {
		t.Fatalf("no puedo leer nucleo/: %v", err)
	}
	var dirs []string
	for _, e := range entradas {
		if e.IsDir() {
			dirs = append(dirs, "nucleo/"+e.Name())
		}
	}
	if len(dirs) < 6 {
		t.Fatalf("nucleo/ tiene %d paquetes, esperaba al menos 6: renombrado?", len(dirs))
	}
	return dirs
}
