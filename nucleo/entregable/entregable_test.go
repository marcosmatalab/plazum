package entregable_test

import (
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/entregable"
	"github.com/marcosmatalab/plazum/nucleo/incidente"
)

// EL PAYLOAD DE UNA NOTIFICACION DE BRECHA, de punta a punta.
//
// La plantilla se busca POR PROPIEDAD (la que tiene campos que pide al
// incidente) y no por nombre: el invariante 2 prohibe nombrar una norma en el
// codigo, y ademas asi la prueba cubre la plantilla que alguien escriba manana.

func plantillasConIncidente(t *testing.T) []corpus.Plantilla {
	t.Helper()
	paquetes, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatalf("cargando el corpus: %v", err)
	}
	var out []corpus.Plantilla
	for _, p := range paquetes {
		for _, pl := range p.Plantillas {
			for _, c := range pl.Campos {
				if strings.HasPrefix(c.Origen, "incidente:") {
					out = append(out, pl)
					break
				}
			}
		}
	}
	if len(out) == 0 {
		t.Fatal("ninguna plantilla pide nada al incidente: el test no comprueba nada")
	}
	return out
}

func ins(t *testing.T, s string) time.Time {
	t.Helper()
	x, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("instante %q: %v", s, err)
	}
	return x
}

func brecha(t *testing.T) *incidente.Incidente {
	t.Helper()
	i, err := incidente.Abrir("incidente/2026-014",
		ins(t, "2026-03-01T04:00:00Z"), ins(t, "2026-03-02T09:00:00Z"), "soc")
	if err != nil {
		t.Fatal(err)
	}
	return i
}

// ---------------------------------------------------------------------------
// Lo que plazum rellena solo y lo que espera
// ---------------------------------------------------------------------------

func TestElPayloadSaleDelIncidenteYDelArticulo(t *testing.T) {
	i := brecha(t)
	for _, pl := range plantillasConIncidente(t) {
		r, err := entregable.Rellenar(pl,
			map[corpus.ClaseOrigen]entregable.Fuente{corpus.DeIncidente: i})
		if err != nil {
			t.Fatalf("%s: %v", pl.ID, err)
		}
		derivados := 0
		for _, c := range r.Campos {
			if !strings.HasPrefix(c.Origen, "incidente:") {
				continue
			}
			if c.Estado != entregable.Derivado {
				t.Errorf("%s: el campo %s viene del incidente y sale como %q",
					pl.ID, c.Nombre, c.Estado)
				continue
			}
			if c.Valor == "" {
				t.Errorf("%s: el campo %s sale derivado y vacio, que es afirmar un dato que "+
					"nadie ha dado", pl.ID, c.Nombre)
			}
			derivados++
		}
		if derivados == 0 {
			t.Errorf("%s: ni un campo derivado del incidente", pl.ID)
		}

		// EL INSTANTE QUE MANDA. El plazo cuenta desde la constancia, asi que el
		// payload tiene que llevar los dos y distinguirlos: si los dos salieran
		// con la fecha de la ocurrencia, la notificacion diria que se supo cuatro
		// meses antes de saberse.
		valores := map[string]string{}
		for _, c := range r.Campos {
			valores[c.Origen] = c.Valor
		}
		if o, pc := valores["incidente:ocurrio"], valores["incidente:primer_conocimiento"]; o != "" &&
			pc != "" && o == pc {
			t.Errorf("%s: el momento de la violacion y el de la constancia salen iguales (%s)",
				pl.ID, o)
		}
	}
}

// LA LEY DE CONSERVACION DEL ENTREGABLE: todo campo de la plantilla cae en
// exactamente un estado, y la suma da el numero de campos.
//
// Un documento que va a una autoridad de control con un apartado de menos es el
// peor sitio posible para un descarte silencioso: nadie lo nota hasta que lo
// nota la autoridad.
func TestTodoCampoDelEntregableCaeEnExactamenteUnEstado(t *testing.T) {
	i := brecha(t)
	for _, pl := range plantillasConIncidente(t) {
		for _, caso := range []struct {
			nombre  string
			fuentes map[corpus.ClaseOrigen]entregable.Fuente
		}{
			{"con el incidente", map[corpus.ClaseOrigen]entregable.Fuente{corpus.DeIncidente: i}},
			{"sin ninguna fuente", nil},
			{"con la fuente a nil",
				map[corpus.ClaseOrigen]entregable.Fuente{corpus.DeIncidente: nil}},
		} {
			r, err := entregable.Rellenar(pl, caso.fuentes)
			if err != nil {
				t.Fatalf("%s (%s): %v", pl.ID, caso.nombre, err)
			}
			if len(r.Campos) != len(pl.Campos) {
				t.Errorf("%s (%s): la plantilla tiene %d campos y el relleno %d",
					pl.ID, caso.nombre, len(pl.Campos), len(r.Campos))
			}
			suma := 0
			for _, n := range r.Cuenta() {
				suma += n
			}
			if suma != len(r.Campos) {
				t.Errorf("%s (%s): los estados suman %d y hay %d campos",
					pl.ID, caso.nombre, suma, len(r.Campos))
			}
		}
	}
}

// LAS DOS FORMAS DE LA NADA, otra vez (invariante 8): el mapa de fuentes a nil y
// el mapa presente con la fuente a nil tienen que dar lo mismo, y no puede ser
// un panico ni un valor cero con cara de dato.
func TestSinFuenteNoEsLoMismoQueSinDatoYNingunaRellenaConElCero(t *testing.T) {
	for _, pl := range plantillasConIncidente(t) {
		for _, fuentes := range []map[corpus.ClaseOrigen]entregable.Fuente{
			nil,
			{corpus.DeIncidente: nil},
			{},
		} {
			r, err := entregable.Rellenar(pl, fuentes)
			if err != nil {
				t.Fatalf("%s: %v", pl.ID, err)
			}
			for _, c := range r.Campos {
				if !strings.HasPrefix(c.Origen, "incidente:") {
					continue
				}
				if c.Estado != entregable.SinFuente {
					t.Errorf("%s: sin fuente, el campo %s sale como %q", pl.ID, c.Nombre, c.Estado)
				}
				if c.Valor != "" {
					t.Errorf("%s: el campo %s sale con valor %q sin fuente que lo diera",
						pl.ID, c.Nombre, c.Valor)
				}
			}
		}
	}

	// Y un incidente que existe pero al que le falta ESE dato da NoConsta, que
	// es otra cosa: "no tengo el dato" y "no me has dado de donde sacarlo" se
	// arreglan de maneras distintas.
	i := brecha(t) // sin clasificar
	if v, consta := i.Campo("clasificacion_vigente"); consta || v != "" {
		t.Fatalf("un incidente sin clasificar dice que su clasificacion consta: %q", v)
	}
}

// ---------------------------------------------------------------------------
// La doctrina del falso positivo, como test
// ---------------------------------------------------------------------------

// Un campo sin rellenar de una notificacion que todavia no se ha enviado no es
// un incumplimiento: es un documento a medias. Acusar en falso es el unico error
// que un producto de cumplimiento no puede cometer ni una vez, y este documento
// sale de la organizacion hacia una autoridad de control.
func TestNingunCampoPendienteAcusaANadie(t *testing.T) {
	prohibidas := []string{"incumpl", "infracc", "sancion", "no lo has", "no se hizo",
		"olvidado", "negligen"}
	i := brecha(t) // abierto y SIN clasificar, para que haya campos que no constan
	visto := 0
	// La plantilla sintetica es imprescindible y se aprendio con una mutacion.
	// Con solo las del corpus, este test daba VERDE al cambiar el descargo por
	// una acusacion: ninguno de sus campos salia como "no consta" (el incidente
	// contesta los tres que piden), asi que la rama del descargo no se ejecutaba
	// nunca. Un test que no recorre la rama que vigila no vigila nada.
	sintetica := corpus.Plantilla{ID: "prueba.todos_los_campos_del_incidente"}
	for nombre := range corpus.CamposDeIncidente() {
		sintetica.Campos = append(sintetica.Campos,
			corpus.CampoPlantilla{Nombre: nombre, Origen: "incidente:" + nombre})
	}
	noConsta := 0
	for _, pl := range append(plantillasConIncidente(t), sintetica) {
		r, err := entregable.Rellenar(pl,
			map[corpus.ClaseOrigen]entregable.Fuente{corpus.DeIncidente: i})
		if err != nil {
			t.Fatal(err)
		}
		ps := r.Pendientes()
		if len(ps) == 0 {
			t.Errorf("%s: no hay ni un campo pendiente, asi que este test no mira nada", pl.ID)
		}
		for _, p := range ps {
			visto++
			bajo := strings.ToLower(p)
			for _, mala := range prohibidas {
				if strings.Contains(bajo, mala) {
					t.Errorf("%s: la linea de un campo pendiente dice %q: %s", pl.ID, mala, p)
				}
			}
		}
		// Y la frase del descargo tiene que estar, con estas palabras, en los
		// campos que plazum sabria derivar y no constan.
		for _, c := range r.Campos {
			if c.Estado != entregable.NoConsta {
				continue
			}
			hay := false
			for _, p := range ps {
				if strings.HasPrefix(p, c.Nombre+":") &&
					strings.Contains(p, "en tus respuestas no aparece") {
					hay = true
				}
			}
			noConsta++
			if !hay {
				t.Errorf("%s: el campo %s no consta y su linea no lleva el descargo",
					pl.ID, c.Nombre)
			}
		}
	}
	if visto == 0 {
		t.Fatal("ninguna linea de pendientes revisada")
	}
	// Y la afirmacion que faltaba: que la rama del descargo se ha recorrido de
	// verdad. Sin esto, cero campos "no consta" se lee igual que cero fallos.
	if noConsta == 0 {
		t.Fatal("ningun campo salio como \"no consta\", asi que la rama del descargo no se " +
			"ha recorrido y este test no ha comprobado la frase que dice comprobar")
	}
}

// ---------------------------------------------------------------------------
// Alcanzabilidad, en las dos direcciones
// ---------------------------------------------------------------------------

// Direccion 1: todo campo que un paquete PUEDE pedir al incidente, un incidente
// lo sabe contestar. Un vocabulario con una entrada sin respuesta es un hueco
// con nombre, que es peor que un hueco.
func TestTodoCampoDeIncidenteDeclaradoLoContestaUnIncidente(t *testing.T) {
	i := brecha(t)
	en := ins(t, "2026-03-02T10:00:00Z")
	if err := i.Registrar(incidente.Suceso{Tipo: incidente.Clasificacion,
		Clase: "una_clase_cualquiera", InstanteHecho: en, InstanteRegistro: en}); err != nil {
		t.Fatal(err)
	}
	campos := corpus.CamposDeIncidente()
	if len(campos) == 0 {
		t.Fatal("el vocabulario esta vacio")
	}
	for nombre, que := range campos {
		v, consta := i.Campo(nombre)
		if !consta {
			t.Errorf("el vocabulario declara %q (%s) y un incidente completo no lo contesta",
				nombre, que)
			continue
		}
		if v == "" {
			t.Errorf("%q contesta con la cadena vacia", nombre)
		}
	}
}

// Direccion 2, la que se olvida: lo que el objeto contesta esta DECLARADO. No es
// mecanizable entera sin leer el AST del switch, asi que se comprueba lo que si
// lo es: nada de fuera del vocabulario se contesta. Un dato que el incidente
// diera y que ningun paquete pudiera pedir seria un camino que no existe.
func TestUnIncidenteNoContestaNadaQueNoEsteDeclarado(t *testing.T) {
	i := brecha(t)
	campos := corpus.CamposDeIncidente()
	for _, fuera := range []string{"", "Id", "id ", "ocurrio_en", "fuente", "sucesos",
		"clasificacion", "primer_conocimiento_utc", "notificado"} {
		if _, ya := campos[fuera]; ya {
			continue
		}
		if v, consta := i.Campo(fuera); consta {
			t.Errorf("contesta %q, que no esta en el vocabulario, con %q. Un dato que ningun "+
				"paquete puede pedir no llega a ningun entregable", fuera, v)
		}
	}
}
