package pantalla

import (
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// LA LISTA DE UN DESCARTE CUADRA CON SU NUMERO, O NO SE PUBLICA.
//
// # Por que existe esta ley, con su cardinal
//
// Diez de las catorce cifras del pie del calendario no se podian abrir, y el
// motivo declarado no era pereza: esta derivacion guardaba los descartes como
// CONTADORES DE HITOS y lo unico que retenia por elemento era `Destinos`, que va
// por obligacion. Enlazar una cifra en hitos a una lista en obligaciones manda a
// una lista mas corta que el numero que la abre, y eso es PEOR que no tener
// enlace: un numero que no cuadra con su lista hace que se deje de leer la
// pantalla entera, y con razon.
//
// Cinco de esas diez ya se pueden abrir porque la derivacion retiene su lista en
// la misma unidad que su contador. Esta ley es lo que impide que se separen: no
// comprueba que la lista exista, comprueba que CUBRE su numero.
//
// # Y se estrena contra el corpus REAL, no contra una mutacion
//
// Una mutacion demuestra que la puerta detecta un fallo que tu le metiste; un
// rojo sobre dato real demuestra que detecta uno que nadie le metio. La mutacion
// viene despues, en el control negativo, para las ramas que el dato real no
// toca.
func TestCadaListaDeDescartesCuadraConSuContador(t *testing.T) {
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatalf("no puedo cargar el corpus publicado: %v", err)
	}
	ahora := time.Date(2026, 9, 3, 9, 0, 0, 0, time.UTC)

	// SIN NI UN HECHO Y CON TODO ALCANZADO: es el recorrido mas ancho que hay,
	// el que mas ramas de descarte toca.
	cal := Derivar12Meses(ps, TodoAplica, ventana.Hechos{}, ahora)
	if err := cal.Cuadra(); err != nil {
		t.Errorf(`una lista de descarte no cubre su contador sobre el corpus publicado:

%v

  La pantalla enlaza la cifra a la lista. Si no cuadran, quien lo lea deja de
  creerse el pie entero.`, err)
	}

	// SUELO: sin el, un corpus que dejara de cargar relojes daria verde con
	// todos los cubos a cero, que es el verde por vacio.
	if cal.HitosDelCorpus < 100 {
		t.Fatalf("solo %d hitos instalados: esta ley estaria comprobando el vacio",
			cal.HitosDelCorpus)
	}
	// Y SE DICE QUE CUBOS TOCA EL CORPUS REAL Y CUALES NO. Un cubo que el dato
	// real no alcanza es un cubo cuya rama solo la recorre el caso sintetico, y
	// eso hay que saberlo, no suponerlo.
	for nombre, n := range map[string]int{
		"alcanzados":           len(cal.RelojesAlcanzados),
		"estrenan":             len(cal.RelojesQueEstrenan),
		"cesan":                len(cal.Ceses),
		"ya cesados":           len(cal.RelojesYaCesados),
		"empiezan despues":     len(cal.RelojesQueEmpiezanDespues),
		"vigencia ilegible":    len(cal.RelojesConVigenciaIlegible),
		"mas alla":             len(cal.VencimientosMasAlla),
		"antes de la vigencia": len(cal.VencimientosAnterioresALaVigencia),
	} {
		t.Logf("el corpus publicado llena el cubo %q con %d filas", nombre, n)
	}
}

// TODOS LOS CUBOS, INCLUIDOS LOS QUE EL CORPUS REAL DEJA VACIOS.
//
// Una rama que ninguna entrada recorre es una rama que no existe, y la mutacion
// la deja verde porque no hay nada que romper (M47). El corpus sintetico de
// `corpusDeLosCincoCubos` trae un inquilino por caso de tiempo; aqui se le anade
// lo que le falta para llenar tambien los dos cubos de OCURRENCIAS.
func TestCadaCuboDeDescarteTieneSuInquilino(t *testing.T) {
	// Una periodica anual anclada en un hecho de hace tres anos, cuya norma
	// entro en vigor hace uno: produce ocurrencias ANTERIORES a la vigencia
	// (que no son incumplimientos) y ocurrencias dentro de la ventana.
	prematura := periodicaDe("d.prematura", "9", "viejo", "P12M", "2025-09-01")
	// Un plazo larguisimo: su unica ocurrencia cae mas alla de la ventana.
	lejana := plazoDe("d.lejana", "10", "x", "P800D", "2020-01-01")

	ps := append(corpusDeLosCincoCubos(),
		paqueteConRelojes("urn:demo:ocurrencias", prematura, lejana))
	cal := Derivar12Meses(ps, TodoAplica, ventana.Hechos{
		"x":     ahoraDePrueba,
		"viejo": ahoraDePrueba.AddDate(-3, 0, 0),
	}, ahoraDePrueba)

	if err := cal.Cuadra(); err != nil {
		t.Fatalf("el corpus sintetico no cuadra: %v", err)
	}
	for nombre, n := range map[string]int{
		"RelojesAlcanzados":                 len(cal.RelojesAlcanzados),
		"RelojesQueEstrenan":                len(cal.RelojesQueEstrenan),
		"Ceses":                             len(cal.Ceses),
		"RelojesYaCesados":                  len(cal.RelojesYaCesados),
		"RelojesQueEmpiezanDespues":         len(cal.RelojesQueEmpiezanDespues),
		"RelojesConVigenciaIlegible":        len(cal.RelojesConVigenciaIlegible),
		"VencimientosMasAlla":               len(cal.VencimientosMasAlla),
		"VencimientosAnterioresALaVigencia": len(cal.VencimientosAnterioresALaVigencia),
	} {
		if n == 0 {
			t.Errorf(`el cubo %s esta vacio, asi que su rama no la recorre nadie.

  Una lista que ninguna entrada llena es una lista que no existe: la mutacion la
  deja verde porque no hay nada que romper.`, nombre)
		}
	}
	// Y CADA FILA DICE DE DONDE SALE. Una fila de descarte sin regla es un
	// "no sale y no te digo por que", que es el silencio con una capa de
	// pintura encima.
	for _, r := range cal.RelojesYaCesados {
		if strings.TrimSpace(r.Regla) == "" {
			t.Errorf("%s se descarta por haber cesado y no dice con que dato", r.Obligacion)
		}
	}
	for _, r := range cal.RelojesQueEmpiezanDespues {
		if !strings.Contains(r.Regla, "empieza a obligar el ") {
			t.Errorf("%s se descarta por empezar tarde y su regla no trae la fecha: %q",
				r.Obligacion, r.Regla)
		}
	}
	// LA RAMA DEL DESCARGO, CON SU CONTROL POSITIVO. Es la unica de las cinco
	// listas que ensena fechas PASADAS, asi que es la unica que se puede leer
	// como una acusacion. La regla tiene que decir que ese dia la norma no
	// obligaba, y no que se incumplio.
	for _, v := range cal.VencimientosAnterioresALaVigencia {
		if !strings.Contains(v.Regla, "ese dia la norma no obligaba") {
			t.Errorf(`la ocurrencia de %s anterior a la vigencia no lleva su descargo: %q

  Es una fecha pasada al lado de una obligacion. Sin la frase se lee como un
  incumplimiento, y lo que pasa es que ese dia la norma no obligaba.`,
				v.Obligacion, v.Regla)
		}
	}
}

// CONTROL NEGATIVO, UNO POR CUBO: Cuadra sabe decir que no.
//
// Cinco mutaciones, una por lista, aplicadas sobre un calendario ya derivado. Sin
// esto, el verde de arriba podria venir de un metodo que devuelve nil siempre, y
// las cinco ramas se comprueban por separado porque un solo caso dejaria las
// otras cuatro sin recorrer.
func TestCuadraCazaCadaListaQueSeSepareDeSuNumero(t *testing.T) {
	base := func() Calendario {
		return Calendario{
			HitosAplicables:    3,
			RelojesAlcanzados:  []RelojDescartado{{Hitos: []string{"a"}}, {Hitos: []string{"b", "c"}}},
			HitosQueEstrenan:   2,
			RelojesQueEstrenan: []RelojDescartado{{Hitos: []string{"a", "b"}}},
			// EL CESE VA CON DOS HITOS EN UNA SOLA OBLIGACION a proposito: es
			// el caso en el que contar OBLIGACIONES y contar HITOS dan numeros
			// distintos, o sea el unico en el que el descuadre se ve.
			HitosQueCesan:  2,
			Ceses:          []Cese{{Hitos: 2, NombresDeHitos: []string{"a", "b"}}},
			HitosYaCesados: 2, RelojesYaCesados: []RelojDescartado{{Hitos: []string{"a", "b"}}},
			HitosQueEmpiezanDespues:           3,
			RelojesQueEmpiezanDespues:         []RelojDescartado{{Hitos: []string{"a"}}, {Hitos: []string{"b", "c"}}},
			HitosConVigenciaIlegible:          1,
			RelojesConVigenciaIlegible:        []RelojDescartado{{Hitos: []string{"a"}}},
			MasAllaDeLaVentana:                2,
			VencimientosMasAlla:               []VencimientoDescartado{{}, {}},
			VencimientosAntesDeLaVigencia:     1,
			VencimientosAnterioresALaVigencia: []VencimientoDescartado{{}},
		}
	}
	// RAMA POSITIVA: el caso bueno cuadra. Sin esta mitad, un Cuadra que
	// devolviera error siempre pasaria las cinco de abajo.
	if err := base().Cuadra(); err != nil {
		t.Fatalf("el calendario de prueba tenia que cuadrar y dijo: %v", err)
	}

	for nombre, romper := range map[string]func(*Calendario){
		"alcanzados": func(c *Calendario) {
			c.RelojesAlcanzados = c.RelojesAlcanzados[:1]
		},
		"estrenan": func(c *Calendario) {
			c.HitosQueEstrenan = 5
		},
		// LA MUTACION DEL CESE ES LA QUE IMPORTA Y ES ESTA: dejar el `Hitos
		// int` intacto y quitarle un NOMBRE. Es exactamente lo que produce una
		// seccion que pinta menos filas de las que su cabecera cuenta.
		"cesan": func(c *Calendario) {
			c.Ceses[0].NombresDeHitos = []string{"a"}
		},
		"ya cesados": func(c *Calendario) {
			c.RelojesYaCesados[0].Hitos = []string{"a"}
		},
		"empiezan despues": func(c *Calendario) {
			c.RelojesQueEmpiezanDespues = c.RelojesQueEmpiezanDespues[:1]
		},
		"vigencia ilegible": func(c *Calendario) {
			c.HitosConVigenciaIlegible = 4
		},
		"mas alla": func(c *Calendario) {
			c.VencimientosMasAlla = append(c.VencimientosMasAlla, VencimientoDescartado{})
		},
		"antes de la vigencia": func(c *Calendario) {
			c.VencimientosAnterioresALaVigencia = nil
		},
	} {
		c := base()
		romper(&c)
		if err := c.Cuadra(); err == nil {
			t.Errorf(`separada la lista de %q de su numero, Cuadra no ha dicho nada.

  Esa lista se podria enlazar desde una cifra que no la cuenta, y quien la abra
  vera una lista mas larga o mas corta que el numero que pulso.`, nombre)
		}
	}
}
