package plazum

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// UNA URL DE CONFIGURACION NO VIAJA ENTERA A UN ERROR NI A UN LOG.
//
// POR QUE ES UNA REGLA Y NO UN CUIDADO. Cuales de las URL que configura el
// operador son credenciales NO ES UNA PROPIEDAD QUE EL CODIGO PUEDA SABER: el
// webhook de Teams basta tenerlo para escribir en ese canal; el destino de un
// "dead man's switch" lleva el identificador secreto en la ruta; una TSA de pago
// lo lleva en la consulta. Las tres se ven igual que una URL publica, asi que la
// regla no puede ser "redacta las secretas" — eso obliga a acertar — sino "no
// sale ninguna entera", que no obliga a nada.
//
// Y DONDE ACABAN: en el log, en la pantalla y sobre todo en el bloque copiable
// de `plazum doctor --issue`, que existe PARA PEGARLO EN UN ISSUE PUBLICO. Un
// token que llegue ahi lo publica quien pide ayuda.
//
// LO QUE ESTA PUERTA NO CAZA, y hay que decirlo porque es la mitad del problema:
// un error de `http.Client` LLEVA LA URL ENTERA DENTRO, asi que envolverlo con
// %w la filtra aunque el mensaje de fuera este impecable. Eso no se ve leyendo
// nombres de variable. Lo caza la otra mitad, que es conductual: los tests de
// centinela de cada adaptador (adaptadores/tsa/credencial_test.go,
// adaptadores/latido/credencial_test.go, adaptadores/canal/frontera_test.go),
// que plantan un valor dentro de la URL configurada y recorren los caminos de
// fallo. Los DOS hacen falta, y se sabe medido: el barrido por nombres encontro
// cuatro sitios y el de centinela encontro otros cuatro que aquel no veia.

// nombresDeURL son los nombres de variable que en este repositorio significan
// "una direccion que configura el operador".
var nombresDeURL = []string{"url", "destino", "webhook", "endpoint", "hook"}

// formasRedactadas son las unicas maneras autorizadas de nombrar una de esas
// direcciones dentro de un mensaje.
//
// `Redacted()` NO esta, y no es un olvido: solo sustituye la contrasena del
// userinfo y deja intactos ruta, consulta y fragmento, que es donde viven los
// secretos de los servicios de webhook y de ping. Estaba usado en latido
// creyendo que redactaba.
var formasRedactadas = []string{
	"redactado.Anfitrion", ".Host", ".Hostname()", ".Scheme", ".Port()",
}

// deLaPeticionEntrante son expresiones que NO son una URL de configuracion sino
// la de una peticion que llega de fuera. Son otra clase de problema y esta
// puerta no las juzga.
//
// La unica de hoy es la ruta que se registra cuando un handler entra en panico,
// y registrarla es lo que hace operable un servidor: sin ella, un panico dice
// que hubo un panico y no donde. SE EXIME CON UNA CONDICION, comprobada el
// 30-08-2026 recorriendo las rutas registradas: NINGUNA ruta de este servidor
// lleva un secreto en su camino (no hay enlaces magicos ni tokens en la ruta;
// la sesion va por cookie y SCIM por cabecera). El dia que alguna lo lleve,
// esta exencion deja de valer y hay que quitarla, no ampliarla.
var deLaPeticionEntrante = []string{"r.URL.Path"}

// dondeSeMira son las capas que hablan con el exterior. `herramientas/` y los
// destinos de fichero de `cmd/` quedan fuera a proposito: una ruta de disco no
// es una credencial, y el informe de `--issue` ya las redacta por su cuenta.
var dondeSeMira = []string{"adaptadores", "superficies"}

func TestNingunaURLDeConfiguracionViajaEnteraAUnError(t *testing.T) {
	fset := token.NewFileSet()
	mirados, hallazgos := 0, 0
	for _, raiz := range dondeSeMira {
		err := filepath.WalkDir(raiz, func(ruta string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(ruta, ".go") ||
				strings.HasSuffix(ruta, "_test.go") {
				return err
			}
			mirados++
			a, err := parser.ParseFile(fset, ruta, nil, 0)
			if err != nil {
				t.Fatalf("no puedo parsear %s: %v", ruta, err)
			}
			// LAS CONSTANTES DECLARADAS NO CUENTAN, y no es una excepcion de
			// conveniencia: una constante esta compilada dentro del binario,
			// asi que no puede ser una direccion que configure el operador.
			// `DestinoPorDefecto` es https://plazum.dev/latido y decirlo en un
			// error es justo lo que ayuda. Que no haya un secreto ESCRITO en
			// una constante es otro problema, y lo vigila el escaneo de
			// secretos del CI, no esto.
			constantes := constantesDe(a)
			ast.Inspect(a, func(n ast.Node) bool {
				c, ok := n.(*ast.CallExpr)
				if !ok || !esMensaje(c.Fun) {
					return true
				}
				for _, arg := range c.Args {
					texto := expr(arg)
					if !pareceURL(texto) || estaRedactada(texto) || constantes[texto] ||
						esDeLaPeticionEntrante(texto) {
						continue
					}
					hallazgos++
					t.Errorf(`%s:%d interpola %s en un mensaje.

  Una URL que configura el operador puede ser una credencial, y el codigo no
  puede saber cuales lo son. Este mensaje acaba en el log y en el bloque
  copiable de "plazum doctor --issue", que se pega en issues publicos.

  Arreglo: redactado.Anfitrion(...), o el campo concreto que no puede ser el
  secreto (.Host, .Scheme). Y si el error que envuelves viene de http.Client,
  NO lo envuelvas: lleva la URL entera dentro.`,
						ruta, fset.Position(arg.Pos()).Line, texto)
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	// Suelo: si el recorrido deja de encontrar ficheros, "ninguna fuga" se
	// leeria exactamente igual que "todo en orden".
	if mirados < 20 {
		t.Fatalf("solo se han mirado %d ficheros de %v: se ha roto el recorrido y este test "+
			"dejaria de probar nada", mirados, dondeSeMira)
	}
	if hallazgos == 0 {
		t.Logf("%d ficheros mirados, ninguna URL de configuracion en un mensaje", mirados)
	}
}

// esMensaje dice si la llamada construye un texto que puede acabar en un log.
func esMensaje(fun ast.Expr) bool {
	s, ok := fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := s.X.(*ast.Ident)
	if !ok {
		return false
	}
	switch pkg.Name + "." + s.Sel.Name {
	case "fmt.Errorf", "fmt.Sprintf", "fmt.Fprintf", "fmt.Fprintln", "errors.New",
		"log.Printf", "log.Println", "log.Fatalf":
		return true
	}
	return false
}

// expr devuelve el texto de una expresion, solo para las formas que interesan
// (identificadores y selectores). Una llamada compuesta se aplana a su fuente
// aproximada, que es suficiente para reconocer el nombre.
func expr(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return expr(v.X) + "." + v.Sel.Name
	case *ast.CallExpr:
		return expr(v.Fun) + "()"
	}
	return ""
}

func pareceURL(texto string) bool {
	if texto == "" {
		return false
	}
	// Los centinelas de error (ErrDestinoInseguro) llevan el nombre dentro y no
	// son direcciones. Se reconocen por el prefijo, que es la convencion del
	// repositorio.
	ultimo := texto
	if i := strings.LastIndex(texto, "."); i >= 0 {
		ultimo = texto[i+1:]
	}
	if strings.HasPrefix(ultimo, "Err") {
		return false
	}
	bajo := strings.ToLower(texto)
	for _, n := range nombresDeURL {
		if strings.Contains(bajo, n) {
			return true
		}
	}
	return false
}

func estaRedactada(texto string) bool {
	for _, f := range formasRedactadas {
		if strings.Contains(texto, f) {
			return true
		}
	}
	return false
}

// constantesDe recoge los nombres declarados con `const` en un fichero.
func constantesDe(a *ast.File) map[string]bool {
	out := map[string]bool{}
	for _, d := range a.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || g.Tok != token.CONST {
			continue
		}
		for _, spec := range g.Specs {
			v, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, n := range v.Names {
				out[n.Name] = true
			}
		}
	}
	return out
}

func esDeLaPeticionEntrante(texto string) bool {
	for _, e := range deLaPeticionEntrante {
		if texto == e {
			return true
		}
	}
	return false
}
