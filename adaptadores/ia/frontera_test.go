package ia

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/internal/modulo"
)

// fuentesDelPaquete son los .go de este directorio que NO son tests.
//
// Se recorre a mano y no con parser.ParseDir porque ParseDir esta obsoleto
// desde Go 1.22, y staticcheck es puerta bloqueante de CI (SA1019). Una
// supresion aqui seria una directiva dirigida a una herramienta que si corre,
// o sea que valdria, pero taparia un aviso correcto en vez de arreglarlo.
func fuentesDelPaquete(t *testing.T) map[string]*ast.File {
	t.Helper()
	entradas, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	out := map[string]*ast.File{}
	for _, e := range entradas {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		a, err := parser.ParseFile(fset, n, nil, 0)
		if err != nil {
			t.Fatalf("no puedo parsear %s: %v", n, err)
		}
		out[n] = a
	}
	return out
}

// LAS PUERTAS DE FORMA DE ESTE PAQUETE. Vigilan propiedades que no se pueden
// comprobar ejecutando codigo, porque son propiedades de COMO ESTA ESCRITO.

// importsPermitidos es la lista CERRADA de lo que puede importar el arnes.
//
// ES UNA LISTA BLANCA Y NO UNA NEGRA, y esa es la decision: una lista negra
// ("no importes el ledger") deja pasar todo lo que a nadie se le ocurrio
// prohibir, que es donde entran las cosas nuevas. Una lista blanca obliga a
// venir aqui y escribir el import con su motivo, y ese gesto es la
// conversacion que hay que tener.
//
// `nucleo/corpus` esta y es el unico del nucleo: se lee para construir las
// fuentes citables, y leer el corpus no muta nada. Lo que NO puede estar, y es
// el invariante 4 dicho como codigo, es cualquier cosa que ESCRIBA estado o
// ledger: la IA no muta el expediente, propone.
var deLaBiblioteca = map[string]bool{
	"crypto/sha256": true,
	"encoding/hex":  true,
	"errors":        true,
	"fmt":           true,
	"os":            true,
	"strings":       true,
	"unicode":       true,
}

// deCasa va SIN el prefijo del modulo, y no por comodidad: la ruta del modulo
// vive en go.mod y en ningun otro sitio (TestNadieCableaLaRutaDelModulo en la
// raiz). Escrita aqui seria una segunda copia, y una segunda copia no se rompe
// cuando la primera cambia: se queda vieja dando verde. Ya paso con el
// renombrado del 28-08-2026, y dejo dos puertas vigilando el vacio.
var deCasa = map[string]bool{
	"adaptadores/busqueda": true,
	"nucleo/corpus":        true,
	"puertos":              true,
}

func permitido(imp, mod string) bool {
	if !modulo.EsDeCasa(imp, mod) {
		return deLaBiblioteca[imp]
	}
	return deCasa[modulo.Interno(imp, mod)]
}

func TestElArnesNoImportaNadaQueEscribaEstado(t *testing.T) {
	mod, err := modulo.Ruta()
	if err != nil {
		t.Fatal(err)
	}
	ficheros := 0
	for ruta, a := range fuentesDelPaquete(t) {
		ficheros++
		for _, imp := range a.Imports {
			v := strings.Trim(imp.Path.Value, `"`)
			if permitido(v, mod) {
				continue
			}
			t.Errorf(`%s importa %q, que no esta en la lista blanca del arnes.

  Invariante 4: los adaptadores de IA JAMAS importan escritura de estado ni
  ledger. La lista de imports de este paquete es cerrada, y se amplia viniendo
  a importsPermitidos y escribiendo por que, no anadiendo el import y ya.

  Si lo que hace falta es leer algo del nucleo, mira primero si puede entrar
  como DATO: es la misma forma que el instante (invariante 1).`, ruta, v)
		}
	}
	// Suelo: si el recorrido deja de encontrar el paquete, "ningun import
	// prohibido" se leeria exactamente igual que "todo en orden".
	if ficheros < 3 {
		t.Fatalf("solo se han mirado %d ficheros de este paquete: el recorrido esta roto y "+
			"esta puerta daria verde sin mirar nada", ficheros)
	}
}

// TestVerificadaNoSePuedeConstruirDesdeFuera.
//
// LA PUERTA ANTIALUCINACION ESCRITA EN EL SISTEMA DE TIPOS. `Verificada` no
// tiene ni un campo exportado, asi que fuera de este paquete NO HAY FORMA de
// fabricar una: el unico camino es Verificar. Eso convierte "la cita se
// verifica ANTES de ensenarla" de una convencion que alguien tiene que
// recordar en una propiedad que el compilador sostiene.
//
// Y por eso la comprobacion es de AST y no de ejecucion: lo que se vigila no es
// lo que el codigo hace, es lo que el codigo DEJA HACER a quien lo use.
func TestVerificadaNoSePuedeConstruirDesdeFuera(t *testing.T) {
	fset := token.NewFileSet()
	a, err := parser.ParseFile(fset, "verificador.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	encontrada := false
	ast.Inspect(a, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok || ts.Name.Name != "Verificada" {
			return true
		}
		encontrada = true
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			t.Fatalf("Verificada ha dejado de ser una estructura")
		}
		for _, campo := range st.Fields.List {
			for _, nombre := range campo.Names {
				if nombre.IsExported() {
					t.Errorf(`Verificada tiene el campo exportado %s.

  Con un campo exportado, cualquiera puede escribir ia.Verificada{%s: ...} y
  saltarse el verificador entero. La puerta antialucinacion dejaria de ser una
  propiedad del sistema de tipos y volveria a ser "acuerdate de llamar a
  Verificar", que es exactamente lo que no funciona.

  Arreglo: campo sin exportar mas un metodo de lectura.`, nombre.Name, nombre.Name)
				}
			}
		}
		return false
	})
	if !encontrada {
		t.Fatal("no se ha encontrado el tipo Verificada en verificador.go. Si se ha movido, " +
			"esta puerta estaria comprobando el vacio")
	}
}

// TestNingunaSalidaExportadaDevuelveLaPropuestaCruda.
//
// EL AGUJERO QUE ESTO CIERRA, y no es teorico: basta con anadir
//
//	func (v Verificada) Cruda() puertos.Propuesta { return v.prop }
//
// para que toda la opacidad de arriba deje de servir, porque entonces cualquiera
// saca el texto que escribio el modelo y lo pinta. Un campo sin exportar con un
// metodo que lo devuelve es un campo exportado con pasos de mas.
//
// La propuesta cruda la produce `adaptadores/ia/ollama`, que es OTRO paquete y
// que la devuelve porque implementa `puertos.Asistente`, que asi esta
// congelado. Ahi es correcto: es la boca del adaptador. Lo que no puede pasar
// es que salga POR AQUI, que es el sitio por el que se supone que ya paso.
func TestNingunaSalidaExportadaDevuelveLaPropuestaCruda(t *testing.T) {
	funciones := 0
	for ruta, a := range fuentesDelPaquete(t) {
		for _, d := range a.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			funciones++
			if !fd.Name.IsExported() || fd.Type.Results == nil {
				continue
			}
			for _, r := range fd.Type.Results.List {
				if tipoDe(r.Type) != "puertos.Propuesta" {
					continue
				}
				t.Errorf(`%s: %s devuelve una puertos.Propuesta cruda.

  Este paquete es la puerta: lo que sale de aqui ya tiene que estar verificado,
  y eso es lo que significa el tipo Verificada. Una funcion exportada que
  devuelve la propuesta tal cual permite pintar el texto del modelo sin haber
  comprobado su cita, y entonces la puerta esta puesta pero tiene una ventana
  al lado.

  Arreglo: devuelve Verificada, o no lo exportes.`, ruta, fd.Name.Name)
			}
		}
	}
	if funciones < 10 {
		t.Fatalf("solo se han mirado %d funciones: el recorrido esta roto", funciones)
	}
}

// tipoDe aplana una expresion de tipo a su texto, para las formas que
// interesan: un identificador, un selector de paquete, un puntero y un slice.
func tipoDe(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return tipoDe(v.X) + "." + v.Sel.Name
	case *ast.StarExpr:
		return tipoDe(v.X)
	case *ast.ArrayType:
		return tipoDe(v.Elt)
	}
	return ""
}

// CONTROL DEL DETECTOR. Las tres puertas de arriba son de AST, o sea que su
// fallo probable no es dejar pasar algo: es NO RECONOCER la forma que buscan y
// dar verde sobre un arbol que no ha mirado. Esto lo demuestra sobre fuentes
// sinteticas, que es donde se puede.
func TestLosDetectoresDeEsteFicheroFuncionan(t *testing.T) {
	fset := token.NewFileSet()

	t.Run("ve un campo exportado en Verificada", func(t *testing.T) {
		a, err := parser.ParseFile(fset, "s.go",
			"package ia\n\ntype Verificada struct {\n\tProp int\n\tfuente int\n}\n", 0)
		if err != nil {
			t.Fatal(err)
		}
		visto := []string{}
		ast.Inspect(a, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok || ts.Name.Name != "Verificada" {
				return true
			}
			for _, c := range ts.Type.(*ast.StructType).Fields.List {
				for _, nom := range c.Names {
					if nom.IsExported() {
						visto = append(visto, nom.Name)
					}
				}
			}
			return false
		})
		if len(visto) != 1 || visto[0] != "Prop" {
			t.Fatalf("el detector de campos exportados ve %v; mientras eso pase, la puerta "+
				"de la opacidad esta dando verde sin mirar nada", visto)
		}
	})

	t.Run("ve una funcion que devuelve la propuesta cruda", func(t *testing.T) {
		fuente := "package ia\n\n" +
			"func (v Verificada) Cruda() puertos.Propuesta { return v.prop }\n" +
			"func (v Verificada) Cita() string { return \"\" }\n" +
			"func Muchas() []puertos.Propuesta { return nil }\n"
		a, err := parser.ParseFile(fset, "s.go", fuente, 0)
		if err != nil {
			t.Fatal(err)
		}
		var visto []string
		for _, d := range a.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || !fd.Name.IsExported() || fd.Type.Results == nil {
				continue
			}
			for _, r := range fd.Type.Results.List {
				if tipoDe(r.Type) == "puertos.Propuesta" {
					visto = append(visto, fd.Name.Name)
				}
			}
		}
		if len(visto) != 2 {
			t.Fatalf("el detector ve %v y tenia que ver las dos (Cruda y Muchas): la forma "+
				"de slice tambien devuelve propuestas crudas", visto)
		}
		// CONTROL NEGATIVO: no grita con lo que si puede salir.
		for _, n := range visto {
			if n == "Cita" {
				t.Error("falso positivo: Cita() devuelve string")
			}
		}
	})

	t.Run("la lista blanca no admite un import del nucleo que muta", func(t *testing.T) {
		mod, err := modulo.Ruta()
		if err != nil {
			t.Fatal(err)
		}
		for _, prohibido := range []string{
			mod + "/nucleo/ledger",
			mod + "/nucleo/estado",
			mod + "/nucleo/expediente",
			"net/http",
		} {
			if permitido(prohibido, mod) {
				t.Errorf("la lista blanca admite %q, que escribe estado o sale a la red",
					prohibido)
			}
		}
		// CONTROL POSITIVO DEL DETECTOR: los que si estan, estan. Sin esto, un
		// `permitido` que devolviera false siempre pasaria este caso y dejaria
		// la puerta de arriba gritando por todo hasta que alguien la apagara.
		for _, bueno := range []string{mod + "/puertos", mod + "/nucleo/corpus", "fmt"} {
			if !permitido(bueno, mod) {
				t.Errorf("la lista blanca no admite %q, que si tiene que estar", bueno)
			}
		}
	})
}
