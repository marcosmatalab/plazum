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
//
// Se mira el AST y no el texto. Buscar la subcadena "time.Now()" fallaba en las
// dos direcciones: se la saltaba cualquier forma que no pegue los parentesis
// (var ahora = time.Now; ahora()) o que renombre el paquete al importarlo
// (import reloj "time"; reloj.Now()), y en cambio saltaba con una mencion en un
// comentario, que no ejecuta nada. Con el AST se afirma sobre el programa.
func TestElNucleoNoLeeElRelojDelSistema(t *testing.T) {
	fset := token.NewFileSet()
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
			f, err := parser.ParseFile(fset, ruta, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, uso := range relojesDelSistema(f) {
				t.Errorf("%s usa %s: el instante de evaluacion entra como dato, "+
					"si no el expediente no es reproducible", ruta, uso)
			}
		}
	}
}

// relojesDelSistema devuelve las referencias a las funciones de "time" que leen
// el reloj del proceso, con el nombre que tenga el paquete en ESE fichero.
//
// No hace falta que la referencia sea una llamada: guardarse time.Now en una
// variable ya es tener el reloj a mano, y esconde la llamada del que lee.
func relojesDelSistema(f *ast.File) []string {
	relojes := map[string]bool{"Now": true, "Since": true, "Until": true}

	nombres := map[string]bool{}
	punto := false
	for _, imp := range f.Imports {
		if strings.Trim(imp.Path.Value, `"`) != "time" {
			continue
		}
		switch {
		case imp.Name == nil:
			nombres["time"] = true
		case imp.Name.Name == ".":
			punto = true
		case imp.Name.Name == "_":
			// import en blanco: no da acceso a nada
		default:
			nombres[imp.Name.Name] = true
		}
	}
	if len(nombres) == 0 && !punto {
		return nil
	}

	// Los identificadores que son el lado derecho de un selector se marcan
	// aparte: sin esto, el "Now" de "otra.Now" contaria como import con punto.
	seleccionados := map[*ast.Ident]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		if s, ok := n.(*ast.SelectorExpr); ok {
			seleccionados[s.Sel] = true
		}
		return true
	})

	var usos []string
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if id, ok := x.X.(*ast.Ident); ok && nombres[id.Name] && relojes[x.Sel.Name] {
				usos = append(usos, id.Name+"."+x.Sel.Name)
			}
		case *ast.Ident:
			if punto && relojes[x.Name] && !seleccionados[x] {
				usos = append(usos, x.Name+" (time importado con punto)")
			}
		}
		return true
	})
	return usos
}

// Control negativo del detector de reloj. Sin esto, el test de arriba podria
// estar pasando por no mirar nada, que es justo como se colaba la version de
// subcadena. Cada caso es una forma que la version vieja NO cazaba, o una que
// cazaba de mas.
func TestElDetectorDeRelojSaltaCuandoDebe(t *testing.T) {
	casos := []struct {
		nombre string
		fuente string
		usos   int
	}{
		{"llamada directa", "package x\nimport \"time\"\nfunc f() time.Time { return time.Now() }\n", 1},
		{"sin parentesis", "package x\nimport \"time\"\nvar ahora = time.Now\n", 1},
		{"con alias", "package x\nimport reloj \"time\"\nfunc f() { _ = reloj.Now() }\n", 1},
		{"con punto", "package x\nimport . \"time\"\nfunc f() { _ = Now() }\n", 1},
		{"Since tambien lee el reloj", "package x\nimport \"time\"\nfunc f(d time.Time) { _ = time.Since(d) }\n", 1},
		{"solo en un comentario", "package x\n// aqui NO se llama a time.Now()\nfunc f() {}\n", 0},
		{"solo en una cadena", "package x\nvar s = \"time.Now()\"\n", 0},
		{"time sin reloj", "package x\nimport \"time\"\nvar d = time.Hour\n", 0},
		{"Now de otro paquete", "package x\nimport \"time\"\nvar _ = time.Hour\nfunc f(o interface{ Now() int }) { _ = o.Now() }\n", 0},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			a, err := parser.ParseFile(token.NewFileSet(), "sintetico.go", c.fuente, 0)
			if err != nil {
				t.Fatal(err)
			}
			if u := relojesDelSistema(a); len(u) != c.usos {
				t.Fatalf("esperaba %d uso(s) del reloj y encontro %d: %v", c.usos, len(u), u)
			}
		})
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
