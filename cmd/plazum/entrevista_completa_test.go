package main

import (
	"net/url"
	"sort"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/superficies/pantallas"
)

// CUANTAS PREGUNTAS DE LA ENTREVISTA LLEGAN AL MOTOR, MEDIDO DE PUNTA A PUNTA.
//
// # Que mide esto que no midiera ya nada
//
// `entrevista_alcanza_al_motor_test.go` (en la raiz) cuenta lo que EL CORPUS
// declara: cuantas preguntas traen bloque `hecho` y de que forma. Es la mitad de
// la respuesta, y la que no depende de la pantalla.
//
// La otra mitad es la que esta rebanada mueve: de las que el corpus sabe
// traducir, CUANTAS PUEDE MANDAR LA PANTALLA. Hasta el 04-09-2026 solo podia
// mandar si/no, asi que las de forma `con_valor` se quedaban por el camino y el
// exportador las apartaba a un cubo. Este test lo mide contestando la entrevista
// ENTERA como la contestaria un operador desde la pantalla, y pasandola por el
// exportador de verdad.
//
// # Por que las respuestas se derivan del corpus y no se escriben aqui
//
// Escritas a mano, este test mediria la lista que yo escribi y no la entrevista:
// una pregunta nueva no entraria, y el cardinal se quedaria quieto mientras el
// corpus crece. Se contesta cada pregunta con lo que su propio atributo declara
// (un booleano con un si, un enumerado con el primero de SUS valores, un texto,
// una fecha y un entero con algo del tipo que piden), que es exactamente el
// rango de lo que la pantalla ofrece.

// PreguntasQueLaPantallaMandaAlMotor es el cardinal, y es un trinquete en los
// dos sentidos.
//
// Es cuantas respuestas de la entrevista completa producen un HECHO. Si sube,
// alguien ha ensanchado lo que llega y hay que subirlo aqui en el mismo commit;
// si baja, algo ha dejado de llegar y hay que enterarse el mismo dia, porque una
// respuesta que deja de llegar son obligaciones que dejan de aparecer.
//
// Antes de esta rebanada era 27: las de forma `afirma_si` y `afirma_si_valor`,
// que son las que un si/no basta para afirmar. Las 25 de forma `con_valor` no
// tenian por donde llegar.
const PreguntasQueLaPantallaMandaAlMotor = 52

// TotalDePreguntasQueSeContestan es el denominador, congelado por lo mismo: sin
// el, el numerador se podria "mejorar" borrando preguntas.
const TotalDePreguntasQueSeContestan = 68

// entrevistaEntera compone la consulta que dejaria un operador que contesta
// TODAS las preguntas desde la pantalla, con lo que cada atributo declara.
func entrevistaEntera(t *testing.T, ps []*corpus.Paquete) (url.Values, int) {
	t.Helper()
	type clave struct{ urn, entidad, atributo string }
	attr := map[clave]corpus.Atributo{}
	for _, p := range ps {
		for _, e := range p.Entidades {
			for _, a := range e.Atributos {
				attr[clave{p.URN, e.Nombre, a.Nombre}] = a
			}
		}
	}
	v := url.Values{}
	n := 0
	for _, q := range corpus.Entrevista(ps) {
		a, hay := attr[clave{q.Paquete, q.Entidad, q.Atributo}]
		if !hay {
			t.Fatalf("la pregunta %q nombra un atributo que su paquete no declara: este "+
				"test estaria contestando al vacio", q.ID)
		}
		n++
		switch a.Tipo {
		case corpus.Booleano:
			v.Add(pantallas.ParamSi, q.ID)
		case corpus.Enumerado:
			if len(a.Valores) == 0 {
				t.Fatalf("el enumerado de %q no declara ni un valor", q.ID)
			}
			v.Set(pantallas.ClaveValor(q.ID), a.Valores[0])
		case corpus.Fecha:
			v.Set(pantallas.ClaveValor(q.ID), "2026-01-15")
		case corpus.Entero:
			v.Set(pantallas.ClaveValor(q.ID), "7")
		default: // texto
			v.Set(pantallas.ClaveValor(q.ID), "lo que el operador escriba")
		}
	}
	return v, n
}

// TestLaEntrevistaEnteraLlegaAlMotorYSeCuenta es la puerta.
func TestLaEntrevistaEnteraLlegaAlMotorYSeCuenta(t *testing.T) {
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatalf("cargar el corpus: %v", err)
	}
	consulta, total := entrevistaEntera(t, ps)
	if total != TotalDePreguntasQueSeContestan {
		t.Errorf("el corpus trae %d preguntas y la constante dice %d.\n"+
			"  Si han crecido, sube el numero; si han menguado, comprueba que el numerador "+
			"no se esta mejorando borrando preguntas, que es la forma barata de mover una "+
			"cifra sin arreglar nada", total, TotalDePreguntasQueSeContestan)
	}

	doc, cuenta, err := exportarAlcance(ps, consulta, "sis", "Acme SaaS SL")
	if err != nil {
		t.Fatalf("exportando la entrevista entera: %v", err)
	}

	// NI UNA RESPUESTA SE PIERDE POR FALTA DE VALOR. Es la casilla entera de
	// esta rebanada: el cubo `ConValor` es el de «tu norma pregunta CUAL y la
	// pantalla solo sabe preguntar si o no», y contestando desde la pantalla de
	// hoy no puede caer nada ahi.
	if len(cuenta.ConValor) > 0 {
		t.Errorf("%d respuestas se pierden por falta de valor: %v.\n"+
			"  La pantalla ya sabe preguntarlas, asi que si alguna cae en este cubo es que "+
			"no las esta mandando", len(cuenta.ConValor), cuenta.ConValor)
	}
	// NI UNA LLEGA SIN ENTENDERSE. Contestando con lo que el propio atributo
	// declara, todo tiene que ser interpretable; si no lo es, la lectura de
	// entrada y el corpus estan diciendo cosas distintas del mismo dato.
	if len(cuenta.NoInterpretables) > 0 {
		t.Errorf("%d respuestas contestadas CON LO QUE DECLARA SU PROPIO ATRIBUTO llegan sin "+
			"entenderse: %v.\n"+
			"  Entonces la pantalla acepta valores que el corpus no admite, o al reves, y las "+
			"dos mitades no estan mirando la misma lista",
			len(cuenta.NoInterpretables), cuenta.NoInterpretables)
	}
	if len(cuenta.Desconocidas) > 0 {
		t.Errorf("hay ids desconocidos y salen del propio corpus: %v", cuenta.Desconocidas)
	}
	if cuenta.Suma() != cuenta.Leidas {
		t.Errorf("los cubos suman %d y se leyeron %d respuestas", cuenta.Suma(), cuenta.Leidas)
	}

	// EL CARDINAL. Se cuentan los HECHOS y no las traducidas: una respuesta cuyo
	// atributo declara `no_llega_al_motor` se manda al puente igual (y cuenta
	// como traducida) y no afirma nada a proposito, asi que contar traducidas
	// diria que llegan al motor 68 de 68, que es falso.
	if len(doc.Hechos) != PreguntasQueLaPantallaMandaAlMotor {
		direccion := "HA CRECIDO: llega mas de lo que decia la constante, y hay que subirla"
		if len(doc.Hechos) < PreguntasQueLaPantallaMandaAlMotor {
			direccion = "HA MENGUADO: algo ha dejado de llegar al motor, y eso son " +
				"obligaciones que dejan de aparecer"
		}
		t.Errorf("la entrevista entera produce %d hechos y la constante dice %d. %s.\n"+
			"  leidas=%d traducidas=%d negativas=%d",
			len(doc.Hechos), PreguntasQueLaPantallaMandaAlMotor, direccion,
			cuenta.Leidas, cuenta.Traducidas, cuenta.Negativas)
	}

	// CONTROL POSITIVO DE LA MITAD QUE ESTA REBANADA ANADE. Sin esto, el
	// cardinal de arriba lo cumpliria un exportador que solo tradujera booleanos
	// si alguien bajara la constante, y no habria forma de verlo.
	conValor := 0
	for _, h := range doc.Hechos {
		if len(h.Args) == 2 {
			conValor++
		}
	}
	if conValor == 0 {
		t.Error("ni un hecho tiene dos argumentos, asi que ninguna respuesta CON VALOR ha " +
			"llegado y este test estaria contando solo los booleanos de siempre")
	}
	t.Logf("entrevista entera: %d preguntas contestadas, %d hechos, %d de ellos con valor",
		total, len(doc.Hechos), conValor)
}

// TestSinLaMitadConValorLlegaMuchoMenos es el CONTROL NEGATIVO del cardinal.
//
// Contesta la misma entrevista SIN mandar ni un valor (que es lo que la pantalla
// sabia hacer hasta esta rebanada) y comprueba que llega estrictamente menos. Sin
// esta comparacion, el numero de arriba seria un numero solo: no diria si lo que
// se ha construido mueve algo.
func TestSinLaMitadConValorLlegaMuchoMenos(t *testing.T) {
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatalf("cargar el corpus: %v", err)
	}
	entera, _ := entrevistaEntera(t, ps)

	// La misma entrevista como la mandaba la pantalla de antes: todo con un si.
	antes := url.Values{}
	for _, q := range corpus.Entrevista(ps) {
		antes.Add(pantallas.ParamSi, q.ID)
	}

	docEntera, _, err := exportarAlcance(ps, entera, "sis", "Acme")
	if err != nil {
		t.Fatalf("exportando la entera: %v", err)
	}
	docAntes, cuentaAntes, err := exportarAlcance(ps, antes, "sis", "Acme")
	if err != nil {
		t.Fatalf("exportando la de antes: %v", err)
	}
	if len(docAntes.Hechos) >= len(docEntera.Hechos) {
		t.Errorf("mandando solo si/no llegan %d hechos y mandando los valores %d. "+
			"La mitad con valor no esta anadiendo nada",
			len(docAntes.Hechos), len(docEntera.Hechos))
	}
	if len(cuentaAntes.ConValor) == 0 {
		t.Error("mandando solo si/no no cae ni una respuesta en el cubo de «tu norma " +
			"pregunta CUAL»: entonces esta comparacion no esta midiendo lo que dice")
	}
	perdidas := append([]string(nil), cuentaAntes.ConValor...)
	sort.Strings(perdidas)
	t.Logf("solo si/no: %d hechos. Con valores: %d. Preguntas que la pantalla de antes no "+
		"podia mandar: %d", len(docAntes.Hechos), len(docEntera.Hechos), len(perdidas))
}
