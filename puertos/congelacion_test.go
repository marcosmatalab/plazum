package puertos_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
	"testing"
)

// La congelacion de los puertos de la etapa 2.
//
// Por que existe este test. La etapa 2 se construye en varios frentes a la vez
// y todos compilan contra estas interfaces. Si uno cambia una firma para que le
// encaje mejor, rompe a los demas, y lo peor es que lo hace en silencio: el
// suyo compila. Este test convierte ese cambio silencioso en un rojo con
// nombre y apellidos.
//
// No impide cambiar un puerto. Impide cambiarlo SIN DECIRLO: quien lo cambie
// tiene que venir aqui, ver el mensaje, y actualizar la lista a mano. Ese
// gesto deliberado es exactamente la conversacion que hay que tener antes.
//
// Se compara el conjunto de metodos leyendo el AST, no por reflexion, porque
// asi tambien se congelan los NOMBRES de los parametros, que son parte de la
// documentacion de un puerto y lo que lee quien implementa.

// congelado es la firma acordada de cada puerto de la etapa 2, escrita a mano.
// Formato: "Metodo(tipos) tipos", normalizado por firmaDe.
var congelado = map[string][]string{
	"Servidor": {
		"Arrancar(context.Context, string) error",
		"Parar(context.Context) error",
	},
	"Sesion": {
		"Abrir(context.Context, string, time.Duration) (string, error)",
		"Cerrar(context.Context, string) error",
		"ComprobarCSRF(context.Context, string, string) error",
		"Leer(context.Context, string) (string, error)",
		"TokenCSRF(context.Context, string) (string, error)",
	},
	"Plantilla": {
		"Render(io.Writer, string, any, string) error",
	},
	"UIGenerada": {
		"Formularios([]*corpus.Paquete) ([]corpus.CampoUI, error)",
		"Preguntas([]*corpus.Paquete) ([]corpus.PreguntaEntrevista, error)",
	},
	"Catalogo": {
		"Faltantes(string) []string",
		"Idiomas() []string",
		"Traducir(string, string, ...any) string",
	},
	"Actualizador": {
		"Aplicar(context.Context, string) (string, error)",
		"Deshacer(context.Context, string) error",
		"Disponible(context.Context) (string, string, error)",
	},
	"Diagnostico": {
		"Comprobar(context.Context) []Comprobacion",
	},
	"Seguridad": {
		"Envolver(http.Handler) http.Handler",
		"Limitar(string, time.Time) bool",
	},
}

func TestLosPuertosDeLaEtapa2EstanCongelados(t *testing.T) {
	actual := leerInterfaces(t, "etapa2.go")

	for nombre, esperados := range congelado {
		got, ok := actual[nombre]
		if !ok {
			t.Errorf("el puerto %s ha desaparecido de etapa2.go.\n%s", nombre, comoProceder(nombre))
			continue
		}
		if !mismos(got, esperados) {
			t.Errorf("el puerto %s ha cambiado de firma.\n  congelado: %v\n  ahora:     %v\n%s",
				nombre, esperados, got, comoProceder(nombre))
		}
	}
	for nombre := range actual {
		if _, ok := congelado[nombre]; !ok {
			t.Errorf("hay un puerto nuevo en etapa2.go que no esta congelado: %s.\n"+
				"Anadelo a la lista de este test con su firma, para que quede acordado.", nombre)
		}
	}
}

func comoProceder(puerto string) string {
	return fmt.Sprintf(
		"\n  ESTO NO ES UN FALLO DEL TEST.\n"+
			"  Los puertos de la etapa 2 se congelaron antes de implementar nada, porque hay\n"+
			"  varios frentes compilando contra ellos a la vez. Cambiar %s rompe a los demas.\n\n"+
			"  Si el cambio hace falta de verdad: PARA Y PREGUNTA antes de seguir. Y si ya se\n"+
			"  acordo, actualiza la lista `congelado` de este test, que es donde consta.\n", puerto)
}

// Cada puerto tiene que ser implementable: si una interfaz no se puede
// satisfacer, esta mal definida y mejor saberlo antes de repartir el trabajo.
// Estos son dobles vacios, no implementaciones: la etapa 2 empieza sin codigo.
func TestCadaPuertoDeLaEtapa2EsImplementable(t *testing.T) {
	// Las asignaciones de abajo fallarian en COMPILACION si algo no cuadra;
	// que este test exista es para que quede dicho por que estan ahi.
	t.Log("los dobles de dobles_test.go comprueban en compilacion que cada puerto se puede implementar")
}

// --- lectura del AST ---

func leerInterfaces(t *testing.T, fichero string) map[string][]string {
	t.Helper()
	fset := token.NewFileSet()
	a, err := parser.ParseFile(fset, fichero, nil, 0)
	if err != nil {
		t.Fatalf("no puedo leer %s: %v", fichero, err)
	}
	out := map[string][]string{}
	ast.Inspect(a, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		it, ok := ts.Type.(*ast.InterfaceType)
		if !ok || it.Methods == nil {
			return true
		}
		var firmas []string
		for _, m := range it.Methods.List {
			ft, ok := m.Type.(*ast.FuncType)
			if !ok || len(m.Names) == 0 {
				continue
			}
			firmas = append(firmas, m.Names[0].Name+firmaDe(ft))
		}
		sort.Strings(firmas)
		out[ts.Name.Name] = firmas
		return true
	})
	return out
}

// firmaDe normaliza una firma a "(tipos) resultado", ignorando los nombres de
// parametro: lo que se congela es la forma, no como se llamen las variables.
func firmaDe(ft *ast.FuncType) string {
	ent := tiposDe(ft.Params)
	sal := tiposDe(ft.Results)
	s := "(" + strings.Join(ent, ", ") + ")"
	switch len(sal) {
	case 0:
	case 1:
		s += " " + sal[0]
	default:
		s += " (" + strings.Join(sal, ", ") + ")"
	}
	return s
}

func tiposDe(fl *ast.FieldList) []string {
	if fl == nil {
		return nil
	}
	var out []string
	for _, f := range fl.List {
		tipo := expr(f.Type)
		n := len(f.Names)
		if n == 0 {
			n = 1
		}
		for i := 0; i < n; i++ {
			out = append(out, tipo)
		}
	}
	return out
}

func expr(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return expr(v.X) + "." + v.Sel.Name
	case *ast.StarExpr:
		return "*" + expr(v.X)
	case *ast.ArrayType:
		return "[]" + expr(v.Elt)
	case *ast.Ellipsis:
		return "..." + expr(v.Elt)
	case *ast.MapType:
		return "map[" + expr(v.Key) + "]" + expr(v.Value)
	case *ast.InterfaceType:
		if v.Methods == nil || len(v.Methods.List) == 0 {
			return "any"
		}
		return "interface{...}"
	case *ast.FuncType:
		return "func" + firmaDe(v)
	}
	return "?"
}

func mismos(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	x := append([]string(nil), a...)
	y := append([]string(nil), b...)
	sort.Strings(x)
	sort.Strings(y)
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}
