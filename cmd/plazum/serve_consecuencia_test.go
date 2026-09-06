package main

import (
	"net/url"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/adaptadores/instalacion"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/pantalla"
	"github.com/marcosmatalab/plazum/superficies/pantallas"
)

// LA PROPIEDAD QUE SOSTIENE LA PIEZA 2 ENTERA: lo que se ensena ANTES de
// contestar es lo que ocurre DESPUES de contestar.
//
// # Por que es esta y no otra
//
// Esta pantalla le dice a alguien «si contestas que si, se te activan nueve
// obligaciones» justo antes de que lo conteste. Si el numero no fuera el que
// ocurre de verdad, el fallo NO daria error en ningun sitio: daria una pantalla
// plausible que cambia la respuesta de alguien sobre su propia organizacion, y
// se descubriria semanas despues, cuando el calendario no cuadre con lo que le
// prometieron. Es la clase de error mas cara que este producto puede cometer.
//
// # Como se comprueba, y por que asi
//
// No se compara contra un numero escrito aqui: se CONTESTA la pregunta de verdad
// y se vuelve a derivar. La afirmacion es una igualdad entre dos derivaciones
// del mismo motor, no entre una derivacion y una expectativa, asi que el dia que
// el corpus cambie este test sigue diciendo lo mismo sin tocarlo.
//
// EL CONTROL POSITIVO VA DENTRO: se exige que AL MENOS UNA pregunta active algo.
// Sin eso, un calculo que devolviera cero siempre pasaria esta puerta con nota,
// que es exactamente el verde vacio.
func TestLaConsecuenciaQueSeEnsenaEsLaQueOcurre(t *testing.T) {
	ps := corpusDePrueba(t)
	c := &consecuenciasDeLaEntrevista{paquetes: ps, quienEs: identidadDePrueba}

	preguntas := preguntasBooleanas(t, ps)
	if len(preguntas) < 5 {
		t.Fatalf("solo hay %d preguntas de si/no en el corpus: este test recorreria casi nada",
			len(preguntas))
	}

	base := url.Values{}
	conAlguna := 0
	comprobadas := 0
	for _, id := range preguntas {
		prometida, err := c.De(t.Context(), base, id)
		if err != nil {
			t.Fatalf("consecuencia de %q: %v", id, err)
		}
		// Y AHORA SE CONTESTA DE VERDAD. La cuenta que sale de aqui es la que
		// el cliente va a tener en su calendario.
		antes, _, err := c.derivar(base)
		if err != nil {
			t.Fatal(err)
		}
		despues, _, err := c.derivar(copiaCon(base, id))
		if err != nil {
			t.Fatal(err)
		}
		nuevas := 0
		for ob := range despues {
			if !antes[ob] {
				nuevas++
			}
		}
		if prometida.N != nuevas {
			t.Errorf(`la pantalla promete %d obligaciones al contestar que si a %q y ocurren %d.

  Este numero se le ensena a alguien JUSTO ANTES de que cambie lo que declara
  sobre su organizacion. Un numero que no es el que ocurre no da error en ningun
  sitio: da una pantalla plausible, y el cliente lo descubre semanas despues
  cuando su calendario no cuadra con lo que le prometieron.`,
				prometida.N, id, nuevas)
		}
		comprobadas++
		if prometida.N > 0 {
			conAlguna++
		}
	}
	if comprobadas != len(preguntas) {
		t.Errorf("se han comprobado %d de %d preguntas", comprobadas, len(preguntas))
	}
	// EL CONTROL POSITIVO. Sin esta linea, un calculo que devolviera cero
	// siempre pasaria todo lo de arriba.
	if conAlguna == 0 {
		t.Error("ninguna pregunta activa ninguna obligacion, asi que la igualdad de arriba " +
			"se ha comprobado entre dos ceros")
	}
	t.Logf("MEDIDO: %d preguntas de si/no, %d activan al menos una obligacion",
		comprobadas, conAlguna)
}

// CALCULAR LA CONSECUENCIA DE UNA PREGUNTA NO CONTESTA NINGUNA OTRA.
//
// # El fallo que esto impide, y por que no se ve mirando la pantalla
//
// Si el calculo mutara el url.Values que recibe en vez de copiarlo, la
// consecuencia de la pregunta 3 dejaria la 3 contestada para la 4, la 4 para la
// 5, y la ultima diria lo que pasa si contestas que SI A TODO. Cada numero
// seria mayor que el anterior, la pantalla PARECERIA funcionar (una entrevista
// donde las consecuencias crecen se lee como razonable) y nadie lo notaria sin
// recontar a mano.
//
// Es la familia de siempre: el fallo que no rompe nada y produce un artefacto
// plausible.
func TestCalcularUnaConsecuenciaNoContestaLasDemas(t *testing.T) {
	ps := corpusDePrueba(t)
	c := &consecuenciasDeLaEntrevista{paquetes: ps, quienEs: identidadDePrueba}
	preguntas := preguntasBooleanas(t, ps)

	base := url.Values{}
	antes := base.Encode()
	for _, id := range preguntas {
		if _, err := c.De(t.Context(), base, id); err != nil {
			t.Fatal(err)
		}
	}
	if base.Encode() != antes {
		t.Fatalf(`calcular las consecuencias ha modificado las respuestas que recibio.

  antes:   %q
  despues: %q

  Con esto, la consecuencia de cada pregunta incluiria las anteriores y la
  ultima diria lo que pasa si contestas que si a todo. Los numeros crecerian, la
  pantalla pareceria razonable, y nadie lo veria sin recontar.`, antes, base.Encode())
	}

	// Y LA MITAD QUE LO DEMUESTRA DE VERDAD: dos consecuencias calculadas en
	// orden distinto dan lo mismo. Sin esto, una copia mal hecha que restaurara
	// el valor al final pasaria la comprobacion de arriba.
	primera, err := c.De(t.Context(), base, preguntas[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range preguntas[1:] {
		if _, err := c.De(t.Context(), base, id); err != nil {
			t.Fatal(err)
		}
	}
	otraVez, err := c.De(t.Context(), base, preguntas[0])
	if err != nil {
		t.Fatal(err)
	}
	if primera.N != otraVez.N {
		t.Errorf("la consecuencia de %q vale %d la primera vez y %d despues de calcular las "+
			"demas: el calculo arrastra estado", preguntas[0], primera.N, otraVez.N)
	}
}

// EL PRESUPUESTO, PORQUE ES UN COSTE POR PREGUNTA Y LA PANTALLA PINTA DIECINUEVE.
//
// Un calculo de 50 ms por pregunta son casi mil milisegundos de pagina, y
// entonces la pieza que existe para que la entrevista se TERMINE seria la razon
// de que se abandone. Se mide sobre el corpus real y no sobre un dato de
// juguete: lo que decide el coste es cuantas reglas hay que evaluar.
//
// LA MEDIDA SE TOMA DEL MINIMO de varias pasadas, no de la media: el ruido de la
// maquina siempre SUMA, asi que el minimo es el que mas se acerca al coste real.
// Es la misma leccion del canal lateral de tiempo.
func TestLaConsecuenciaCabeEnSuPresupuesto(t *testing.T) {
	ps := corpusDePrueba(t)
	c := &consecuenciasDeLaEntrevista{paquetes: ps, quienEs: identidadDePrueba}
	preguntas := preguntasBooleanas(t, ps)
	base := url.Values{}

	// Una pasada en frio para llenar la memoria de la base: lo que se mide es
	// el coste de UNA pregunta, no el del arranque.
	if _, err := c.De(t.Context(), base, preguntas[0]); err != nil {
		t.Fatal(err)
	}

	mejor := time.Duration(1<<62 - 1)
	for i := 0; i < 5; i++ {
		inicio := time.Now()
		if _, err := c.De(t.Context(), base, preguntas[i%len(preguntas)]); err != nil {
			t.Fatal(err)
		}
		if d := time.Since(inicio); d < mejor {
			mejor = d
		}
	}
	if mejor > PresupuestoDeLaConsecuencia {
		t.Errorf(`calcular UNA consecuencia cuesta %s y el presupuesto es %s.

  La pantalla pinta hasta 19 preguntas, asi que esto se multiplica por 19 en
  cada carga. La pieza que existe para que la entrevista se termine no puede ser
  la razon de que se abandone.`, mejor.Round(time.Microsecond), PresupuestoDeLaConsecuencia)
	}
	t.Logf("MEDIDO: %s por consecuencia sobre el corpus real (presupuesto %s), %d preguntas",
		mejor.Round(time.Microsecond), PresupuestoDeLaConsecuencia, len(preguntas))
}

// SIN IDENTIDAD NO SE INVENTA UN CERO.
//
// Un conjunto vacio diria que contestar que si no activa nada, que es una
// afirmacion sobre el cumplimiento de alguien salida de una instalacion a medio
// configurar. Es la tercera forma de la nada (invariante 8) en la pantalla donde
// mas cara sale.
func TestSinIdentidadLaConsecuenciaEsUnErrorYNoUnCero(t *testing.T) {
	ps := corpusDePrueba(t)
	c := &consecuenciasDeLaEntrevista{
		paquetes: ps,
		quienEs:  func() instalacion.Identidad { return instalacion.Identidad{} },
	}
	preguntas := preguntasBooleanas(t, ps)
	got, err := c.De(t.Context(), url.Values{}, preguntas[0])
	if err == nil {
		t.Fatalf("sin identidad se ha devuelto una consecuencia de %d obligaciones", got.N)
	}
	// CONTROL POSITIVO: con identidad si contesta. Sin esto, un calculo que
	// fallara siempre pasaria la mitad de arriba.
	c.quienEs = identidadDePrueba
	if _, err := c.De(t.Context(), url.Values{}, preguntas[0]); err != nil {
		t.Fatalf("con identidad tenia que calcular: %v", err)
	}
}

func identidadDePrueba() instalacion.Identidad {
	return instalacion.Identidad{Organizacion: "Ejemplo SL", Sujeto: "ejemplo-sl"}
}

// preguntasBooleanas son las de si/no del corpus, que son las que tienen
// consecuencia. Salen del corpus y no de una lista escrita aqui: una lista se
// queda vieja el dia que entre el marco trece.
func preguntasBooleanas(t *testing.T, ps []*corpus.Paquete) []string {
	t.Helper()
	var qs []pantalla.Pregunta
	for _, p := range pantalla.Derivar(ps) {
		qs = append(qs, p.Preguntas...)
	}
	voc := pantallas.VocabularioDe(ps, qs)
	var out []string
	for _, q := range qs {
		// LAS QUE PIDEN UN VALOR NO TIENEN CONSECUENCIA DE «SI», y por eso se
		// quedan fuera: no se contestan que si. Es la misma linea que decide
		// en la pantalla cual de las dos mitades se pinta.
		if voc.Tipo(q.ID).PideValor() {
			continue
		}
		out = append(out, q.ID)
	}
	return out
}
