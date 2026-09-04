package busqueda

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// Los documentos de prueba NO se llaman como normas reales: `adaptadores/` esta
// vigilado por extensibilidad_test.go tambien en sus tests, y ademas lo que se
// prueba aqui es el ranking, que no sabe de normas.

func doc(id, texto string) Documento {
	suma := sha256.Sum256([]byte(texto))
	return Documento{
		ID:       id,
		Hash:     hex.EncodeToString(suma[:]),
		Marco:    "marco-de-prueba",
		Articulo: id,
		Texto:    texto,
	}
}

func indiceDePrueba(t *testing.T) *Indice {
	t.Helper()
	i, err := Nuevo([]Documento{
		doc("a", "El responsable notificara la violacion de seguridad a la autoridad de control"),
		doc("b", "La notificacion se hara sin dilacion indebida y a mas tardar en 72 horas"),
		doc("c", "El encargado del tratamiento llevara un registro de las actividades"),
		doc("d", "Se designara un delegado de proteccion de datos cuando proceda"),
		doc("e", "La autoridad de control publicara la lista de tratamientos"),
	})
	if err != nil {
		t.Fatal(err)
	}
	return i
}

func TestBuscarPonePrimeroLoQueMasCasa(t *testing.T) {
	i := indiceDePrueba(t)
	res, err := i.Buscar("autoridad de control", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) == 0 {
		t.Fatal("cero resultados para una consulta que casa con dos documentos")
	}
	ids := map[string]bool{}
	for _, r := range res {
		ids[r.ID] = true
	}
	if !ids["a"] || !ids["e"] {
		t.Errorf("los dos documentos que hablan de la autoridad de control no salen: %v", ids)
	}
	// El primero tiene que traer los tres terminos, no uno repetido.
	if res[0].Aciertos < 2 {
		t.Errorf("el primer resultado solo casa con %d terminos", res[0].Aciertos)
	}
	// Y la puntuacion tiene que ir de mas a menos.
	for n := 1; n < len(res); n++ {
		if res[n].Puntuacion > res[n-1].Puntuacion {
			t.Fatalf("los resultados no vienen ordenados: %v", res)
		}
	}
}

// EL TERMINO RARO PUNTUA MAS QUE EL COMUN. Es lo que hace BM25 y no un
// contador de apariciones, y si esto se rompiera el buscador pondria primero
// siempre al documento mas largo.
func TestUnTerminoRaroPesaMasQueUnoQueEstaEnTodos(t *testing.T) {
	i, err := Nuevo([]Documento{
		doc("a", "el plazo el plazo el plazo el plazo"),
		doc("b", "el plazo de conservacion documental"),
	})
	if err != nil {
		t.Fatal(err)
	}
	// "el" esta en los dos; "conservacion" solo en uno.
	res, err := i.Buscar("el conservacion", 2)
	if err != nil {
		t.Fatal(err)
	}
	if res[0].ID != "b" {
		t.Fatalf("gana %q; tenia que ganar el que trae el termino raro. Si aqui gana el "+
			"que repite el termino comun, esto no es BM25, es un contador", res[0].ID)
	}
}

// LA REPETICION EN LA CONSULTA NO INFLA NADA. Sin esto, quien escribe la
// consulta (que en el camino de la IA puede ser el propio modelo) sube un
// resultado escribiendo la misma palabra cinco veces.
func TestRepetirUnTerminoEnLaConsultaNoCambiaElOrden(t *testing.T) {
	i := indiceDePrueba(t)
	una, err := i.Buscar("autoridad de control", 5)
	if err != nil {
		t.Fatal(err)
	}
	cinco, err := i.Buscar("autoridad autoridad autoridad de control autoridad", 5)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(una, cinco) {
		t.Fatalf("repetir un termino cambia el resultado.\n  una:   %v\n  cinco: %v", una, cinco)
	}
}

// EL DETERMINISMO NO ES UN LUJO: este buscador construye el contexto de un
// modelo y alimenta un eval de precision. Si dos ejecuciones dan ordenes
// distintos, el eval mide ruido.
func TestElOrdenEsDeterministaTambienEnLosEmpates(t *testing.T) {
	// Tres documentos con EXACTAMENTE el mismo texto: los tres empatan a
	// puntuacion, asi que el unico desempate posible es el ID. Sin desempate,
	// el orden lo decidiria el recorrido de un map, que en Go es aleatorio a
	// proposito.
	i, err := Nuevo([]Documento{
		{ID: "zeta", Hash: strings.Repeat("1", 64), Texto: "conservacion documental por diez anos"},
		{ID: "alfa", Hash: strings.Repeat("2", 64), Texto: "conservacion documental por diez anos"},
		{ID: "mu", Hash: strings.Repeat("3", 64), Texto: "conservacion documental por diez anos"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var primera []Resultado
	for n := 0; n < 30; n++ {
		res, err := i.Buscar("conservacion documental", 3)
		if err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			primera = res
			continue
		}
		if !reflect.DeepEqual(res, primera) {
			t.Fatalf("la busqueda %d da otro orden que la primera.\n  %v\n  %v", n, primera, res)
		}
	}
	if primera[0].ID != "alfa" || primera[1].ID != "mu" || primera[2].ID != "zeta" {
		t.Errorf("el desempate no es por ID: %v", []string{primera[0].ID, primera[1].ID, primera[2].ID})
	}
}

// LAS TRES FORMAS DE LA NADA EN LA ENTRADA DE Buscar.
func TestLosValoresDegeneradosDeBuscarSonError(t *testing.T) {
	i := indiceDePrueba(t)
	casos := []struct {
		nombre    string
		consulta  string
		tope      int
		centinela error
	}{
		{"tope cero NO significa todos", "autoridad", 0, ErrSinTope},
		{"tope negativo", "autoridad", -1, ErrSinTope},
		{"consulta ausente", "", 5, ErrConsultaVacia},
		{"consulta presente y no interpretable", "!!! ??? ---", 5, ErrConsultaIlegible},
		{"consulta de solo espacios", "   \t\n ", 5, ErrConsultaIlegible},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			res, err := i.Buscar(c.consulta, c.tope)
			if !errors.Is(err, c.centinela) {
				t.Fatalf("da (%d resultados, %v), se esperaba %v", len(res), err, c.centinela)
			}
			if len(res) != 0 {
				t.Errorf("con error se devuelven %d resultados", len(res))
			}
		})
	}
	// CONTROL POSITIVO: con los tres bien, sale algo. Sin esto, los cinco de
	// arriba los pasaria igual un Buscar que devolviera error siempre.
	res, err := i.Buscar("autoridad", 5)
	if err != nil || len(res) == 0 {
		t.Fatalf("la consulta buena da (%d, %v)", len(res), err)
	}
}

func TestElTopeSeRespeta(t *testing.T) {
	i := indiceDePrueba(t)
	for _, tope := range []int{1, 2, 3} {
		res, err := i.Buscar("de la", tope)
		if err != nil {
			t.Fatal(err)
		}
		if len(res) > tope {
			t.Errorf("tope %d y devuelve %d", tope, len(res))
		}
	}
}

// LAS DOS FORMAS DE LA NADA AL CONSTRUIR, recorridas las dos.
//
// Aqui la nada es inocua, y por eso NO se prohibe: en un buscador no existe el
// "sin restriccion" que en un almacen de certificados significa "acepto
// cualquier CA". Se recorren las dos igualmente, porque la afirmacion "aqui la
// nada es inocua" solo vale si alguien la comprueba.
func TestUnIndiceVacioContestaCeroPorLasDosFormasDeLaNada(t *testing.T) {
	for _, c := range []struct {
		nombre string
		docs   []Documento
	}{
		{"nil", nil},
		{"vacio presente", []Documento{}},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			i, err := Nuevo(c.docs)
			if err != nil {
				t.Fatalf("no construye: %v", err)
			}
			if i.Documentos() != 0 {
				t.Fatalf("Documentos() = %d", i.Documentos())
			}
			res, err := i.Buscar("cualquier cosa", 5)
			if err != nil {
				t.Fatalf("buscar en un indice vacio da error: %v", err)
			}
			if len(res) != 0 {
				t.Fatalf("un indice vacio devuelve %d resultados", len(res))
			}
		})
	}
}

func TestUnDocumentoMalFormadoNoEntraAlIndice(t *testing.T) {
	bueno := doc("a", "un texto cualquiera con palabras dentro")
	casos := []struct {
		nombre    string
		docs      []Documento
		centinela error
	}{
		{"sin ID", []Documento{{Hash: bueno.Hash, Texto: "texto"}}, ErrDocumentoSinID},
		{"sin hash", []Documento{{ID: "a", Texto: "texto"}}, ErrDocumentoSinHash},
		{"ID repetido", []Documento{bueno, doc("a", "otro texto distinto")}, ErrIDRepetido},
		{"texto que no da terminos", []Documento{{ID: "a", Hash: bueno.Hash, Texto: "!!! ---"}}, ErrDocumentoSinTexto},
		{"texto vacio", []Documento{{ID: "a", Hash: bueno.Hash, Texto: ""}}, ErrDocumentoSinTexto},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			i, err := Nuevo(c.docs)
			if !errors.Is(err, c.centinela) {
				t.Fatalf("construye (%v, %v), se esperaba %v", i, err, c.centinela)
			}
			if !strings.Contains(err.Error(), "Arreglo:") {
				t.Errorf("el error no dice como arreglarlo: %v", err)
			}
		})
	}
	// EL ID REPETIDO IMPORTA POR EL INVARIANTE 7: con dos documentos bajo el
	// mismo ID, cual gana lo decide el ORDEN de la lista, y el orden no lo firma
	// nadie. Reordenar el corpus moveria a que articulo apunta un resultado sin
	// romper ninguna firma.
}

// -------------------------------------------------------------------------
// El tokenizador.
// -------------------------------------------------------------------------

func TestElTokenizadorPliegaLoQueDiceQuePliega(t *testing.T) {
	casos := []struct {
		entrada  string
		esperado []string
	}{
		{"Notificacion", []string{"notificacion"}},
		{"Notificación", []string{"notificacion"}},
		{"NOTIFICACIÓN", []string{"notificacion"}},
		{"año", []string{"ano"}},
		{"España, Portugal.", []string{"espana", "portugal"}},
		{"art. 33.1", []string{"art", "33", "1"}},
		{"72 horas", []string{"72", "horas"}},
		{"---!!!---", nil},
		{"", nil},
		{"  \t\n ", nil},
	}
	for _, c := range casos {
		got := Tokenizar(c.entrada)
		if len(got) == 0 && len(c.esperado) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, c.esperado) {
			t.Errorf("Tokenizar(%q) = %v, se esperaba %v", c.entrada, got, c.esperado)
		}
	}
}

func TestBuscarSinTildeEncuentraLoQueLaTraeYAlReves(t *testing.T) {
	i, err := Nuevo([]Documento{doc("a", "La notificación se hará en un plazo de setenta y dos horas")})
	if err != nil {
		t.Fatal(err)
	}
	for _, consulta := range []string{"notificacion", "notificación", "NOTIFICACION"} {
		res, err := i.Buscar(consulta, 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(res) != 1 {
			t.Errorf("la consulta %q no encuentra el documento: quien teclea no siempre "+
				"tiene el teclado de la norma", consulta)
		}
	}
}

// La tabla de plegado se lee de una cadena, asi que una entrada mal escrita se
// saltaria en silencio al arrancar. Esto la recorre entera.
func TestLaTablaDePlegadoEstaBienFormada(t *testing.T) {
	pares := strings.Fields(plegado)
	if len(pares) < 50 {
		t.Fatalf("la tabla tiene %d entradas: se ha encogido y el plegado dejaria de "+
			"cubrir el latin de la UE", len(pares))
	}
	for _, par := range pares {
		rs := []rune(par)
		if len(rs) != 2 {
			t.Errorf("la entrada %q tiene %d runas y tenia que tener 2 (origen y destino). "+
				"construirPliegues la salta EN SILENCIO, asi que sin este test el plegado "+
				"de esa letra desapareceria sin que nada lo dijera", par, len(rs))
			continue
		}
		if rs[1] > 127 {
			t.Errorf("la entrada %q pliega a un caracter que no es ASCII: el destino tiene "+
				"que ser la letra base", par)
		}
		if pliegues[rs[0]] != rs[1] {
			t.Errorf("la entrada %q no ha llegado al mapa", par)
		}
	}
	if len(pliegues) != len(pares) {
		t.Errorf("la tabla tiene %d entradas y el mapa %d: hay origenes repetidos",
			len(pares), len(pliegues))
	}
}
