package main

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// La puerta de que el binario CUENTA lo que sabe hacer.
//
// POR QUE EXISTE. `plazum serve` estuvo implementada, arrancaba, servia las seis
// pantallas y tenia su suite en verde, y la lista que imprime `plazum` a secas no
// la nombraba. Un comprador que descarga el binario a las nueve de la manana
// tiene entonces exactamente dos formas de enterarse de que existe la interfaz
// web: leer el codigo fuente o adivinarla. Las dos son la misma: no se entera.
//
// Y no era solo un problema de descubrimiento. La puerta de accesibilidad de CI
// preguntaba a esa lista para saber si habia pantallas que auditar, no las
// encontraba, y se quedaba en rojo diciendo que el producto no sabia servirlas
// cuando llevaba semanas sabiendo. Una lista escrita a mano al lado de un
// despachador se desincroniza sola; lo unico que decide si eso se nota es que
// haya algo mirando.
//
// COMO SE COMPRUEBA. Las ordenes NO se escriben aqui: se leen del AST de
// main.go, del sitio donde de verdad se despachan (los `case` de los switch
// sobre os.Args[1] y las comparaciones sueltas contra os.Args[1]). Una lista
// escrita en este test seria la tercera copia del mismo dato y se desincronizaria
// igual que las otras dos.

// ordenesDeMain lee un fuente de main y devuelve, por separado, las ordenes que
// DESPACHA y las que IMPRIME en su lista de uso.
//
// Vive fuera del test para que el control negativo de abajo pase por este mismo
// codigo: un checkeo que solo se ha ejecutado contra el caso bueno no ha
// demostrado que sepa decir que no.
func ordenesDeMain(t *testing.T, fuente string) (despacha, imprime []string) {
	t.Helper()
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "main.go", fuente, 0)
	if err != nil {
		t.Fatalf("no se puede parsear el fuente de main: %v", err)
	}

	vistoD := map[string]bool{}
	vistoI := map[string]bool{}

	// esArgs1 dice si una expresion es os.Args[1], que es el sitio por donde
	// entra la orden.
	esArgs1 := func(e ast.Expr) bool {
		ix, ok := e.(*ast.IndexExpr)
		if !ok {
			return false
		}
		if lit, ok := ix.Index.(*ast.BasicLit); !ok || lit.Value != "1" {
			return false
		}
		sel, ok := ix.X.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Args" {
			return false
		}
		id, ok := sel.X.(*ast.Ident)
		return ok && id.Name == "os"
	}
	literal := func(e ast.Expr) (string, bool) {
		lit, ok := e.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return "", false
		}
		s, err := strconv.Unquote(lit.Value)
		return s, err == nil
	}

	ast.Inspect(f, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.SwitchStmt:
			// switch os.Args[1] { case "demo": ... }
			if v.Tag == nil || !esArgs1(v.Tag) {
				return true
			}
			for _, s := range v.Body.List {
				cc, ok := s.(*ast.CaseClause)
				if !ok {
					continue // default: no nombra ninguna orden
				}
				for _, e := range cc.List {
					if s, ok := literal(e); ok {
						vistoD[s] = true
					}
				}
			}
		case *ast.BinaryExpr:
			// if os.Args[1] == "cobertura" { ... }
			if v.Op != token.EQL {
				return true
			}
			if esArgs1(v.X) {
				if s, ok := literal(v.Y); ok {
					vistoD[s] = true
				}
			}
			if esArgs1(v.Y) {
				if s, ok := literal(v.X); ok {
					vistoD[s] = true
				}
			}
		case *ast.BasicLit:
			// La lista de uso son literales que empiezan por "plazum " tras
			// recortar. Se lee de los literales y no de una llamada concreta
			// para no atarse a que se imprima con Fprintln, Printf o lo que
			// venga: lo que le llega al operador es el texto.
			s, ok := literal(v)
			if !ok {
				return true
			}
			campos := strings.Fields(s)
			if len(campos) >= 2 && campos[0] == "plazum" {
				vistoI[campos[1]] = true
			}
		}
		return true
	})

	for k := range vistoD {
		despacha = append(despacha, k)
	}
	for k := range vistoI {
		imprime = append(imprime, k)
	}
	sort.Strings(despacha)
	sort.Strings(imprime)
	return despacha, imprime
}

func TestElUsoNombraTodaOrdenQueElBinarioDespacha(t *testing.T) {
	b, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("no se puede leer main.go: %v", err)
	}
	despacha, imprime := ordenesDeMain(t, string(b))

	// Sin suelo, el dia que el extractor deje de reconocer los `case` este test
	// compararia el vacio contra el vacio y saldria verde. Hoy son ocho.
	if len(despacha) < 8 {
		t.Fatalf("solo se han encontrado %d ordenes despachadas (%v). O main.go ha "+
			"adelgazado a la mitad, o este test ha dejado de ver por donde se despacha, "+
			"y las dos cosas dejan la comprobacion de abajo comparando el vacio contra "+
			"el vacio", len(despacha), despacha)
	}

	dice := map[string]bool{}
	for _, o := range imprime {
		dice[o] = true
	}
	for _, o := range despacha {
		if !dice[o] {
			t.Errorf("`plazum %s` se despacha en main.go y la lista de uso no la nombra.\n"+
				"  Quien descarga el binario y teclea `plazum` no tiene forma de enterarse\n"+
				"  de que existe salvo leyendo el codigo fuente.\n"+
				"  Arreglo: anadir la linea a la lista que imprime main(), con una frase\n"+
				"  que diga para que sirve.\n"+
				"  Se despachan: %v\n  Se imprimen:  %v", o, despacha, imprime)
		}
	}
}

// CONTROL NEGATIVO. Sin esto, el verde de arriba no prueba que se este mirando
// nada: un extractor que devolviera dos listas vacias daria exactamente el mismo
// verde. Se le da un main que despacha una orden que no imprime y se exige que
// la comprobacion la senale.
func TestLaPuertaDelUsoCazaUnaOrdenNoAnunciada(t *testing.T) {
	const fuenteRota = `package main

import "os"

func main() {
	switch os.Args[1] {
	case "demo":
		os.Exit(0)
	case "escondida":
		os.Exit(0)
	}
	println("     plazum demo      lo unico que se anuncia")
}
`
	despacha, imprime := ordenesDeMain(t, fuenteRota)

	dice := map[string]bool{}
	for _, o := range imprime {
		dice[o] = true
	}
	faltan := []string{}
	for _, o := range despacha {
		if !dice[o] {
			faltan = append(faltan, o)
		}
	}
	if len(faltan) != 1 || faltan[0] != "escondida" {
		t.Fatalf("la comprobacion del uso no caza una orden despachada y no anunciada.\n"+
			"  Mientras esto pase, su verde sobre main.go no significa nada.\n"+
			"  despacha=%v imprime=%v faltan=%v", despacha, imprime, faltan)
	}
}

// Las dos primeras ordenes de la lista tienen que encadenar.
//
// POR QUE EXISTE. Recorriendo el producto como lo recorre quien lo acaba de
// descargar: `plazum` a secas, `plazum demo`, y despues lo siguiente que anuncia
// la lista, `plazum serve`. Se estrellaba. El demo deja su corpus en
// plazum-demo/paquetes y serve mira en paquetes, y el error decia "Arreglo: [...]
// ejecuta `plazum demo`", que es exactamente lo que la persona acababa de hacer.
// Un mensaje que manda a repetir el paso anterior no es un error accionable: es
// un callejon con luz.
func TestServeSinCorpusSenalaElDelDemoCuandoEstaDelante(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.MkdirAll(corpusDelDemo, 0o755); err != nil {
		t.Fatal(err)
	}

	var salida, errsal bytes.Buffer
	if rc := cmdServe(nil, &salida, &errsal); rc == 0 {
		t.Fatalf("serve ha arrancado sin corpus. Salida: %s", errsal.String())
	}
	quiero := "plazum serve --corpus " + corpusDelDemo
	if !strings.Contains(errsal.String(), quiero) {
		t.Errorf("con el corpus del demo delante, serve no dice el comando que funciona.\n"+
			"  Esperaba encontrar: %s\n  Dijo:\n%s", quiero, errsal.String())
	}

	// CONTROL NEGATIVO: sin el corpus del demo delante, ese comando NO se
	// sugiere, porque no funcionaria. Sin esto, un mensaje que lo dijera siempre
	// pasaria la comprobacion de arriba y mandaria a la gente a un directorio
	// que no existe.
	otro := t.TempDir()
	t.Chdir(otro)
	if _, err := os.Stat(filepath.Join(otro, "plazum-demo")); err == nil {
		t.Fatal("el directorio de control no esta vacio, asi que no controla nada")
	}
	salida.Reset()
	errsal.Reset()
	if rc := cmdServe(nil, &salida, &errsal); rc == 0 {
		t.Fatalf("serve ha arrancado sin corpus ninguno. Salida: %s", errsal.String())
	}
	if strings.Contains(errsal.String(), quiero) {
		t.Errorf("sin corpus del demo delante, serve sigue sugiriendo %q, que no existe "+
			"aqui. El mensaje se estaria imprimiendo siempre y no diria nada.\n  Dijo:\n%s",
			quiero, errsal.String())
	}
	if !strings.Contains(errsal.String(), "--corpus") {
		t.Errorf("sin corpus del demo delante, serve tampoco dice como se arregla.\n  Dijo:\n%s",
			errsal.String())
	}
}
