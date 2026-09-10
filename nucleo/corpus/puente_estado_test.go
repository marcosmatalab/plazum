package corpus

import (
	"testing"
	"time"
)

// EL PUENTE NO MUEVE UNA FECHA MAS DE LO QUE DECLARA, Y SIEMPRE HACIA EL LADO
// SEGURO.
//
// # De donde sale este test, que es lo primero que hay que saber de el
//
// De un hueco, no de una sospecha. `puente_estado.go:78` llevaba escrito
// `// LO VIGILA: TestElPuenteNoMueveUnaFechaMasDeLoDeclarado` y **ese test no
// existia en el arbol**. O sea: la forma exacta que el propio godoc de
// `godoc_vigilado_test.go` avisa que hay que impedir («un nombre con la forma de
// lo verificable es justo lo que hace que nadie vaya a verificarlo»), escrita en
// el arbol y sin que nadie la cazara.
//
// Y no la cazaba nadie por una razon concreta: la marca esta sobre un bloque
// `var (...)`, que no es un interfaz exportado ni lleva `PELIGRO:`, y esas son
// las dos unicas posiciones que la puerta miraba. La tercera mitad que se anade
// hoy (TestTodaVigilanciaDeclaradaNombraUnTestQueExiste) recorre TODA linea de
// vigilancia del arbol, y por eso este fichero existe.
//
// # Que se afirma, exactamente
//
// El puente cuenta un mes como `DiasPorMes` (30) porque `estado.Prueba.TTL` es
// un `time.Duration` y una duracion no sabe de calendarios. Eso MUEVE fechas, a
// diferencia de otras aproximaciones del arbol que solo comparan. La afirmacion
// del godoc son dos cosas y se comprueban las dos:
//
//	CUANTO   un P6M sobre un semestre real se adelanta entre 1 y 4 dias
//	HACIA DONDE  siempre ADELANTA, nunca retrasa
//
// La segunda es la que de verdad importa y es la que no se puede deducir de la
// primera. Un dato que se declara viejo ANTES de tiempo hace que alguien vuelva
// a recolectarlo de mas; uno que se declara viejo DESPUES deja un expediente
// afirmando que algo consta cuando ya habia caducado. Los dos son errores y solo
// uno es aceptable, asi que el test recorre las dos direcciones en vez de
// medir un valor absoluto.

// TestElPuenteNoMueveUnaFechaMasDeLoDeclarado es la puerta que el godoc del
// puente nombraba y que no existia.
func TestElPuenteNoMueveUnaFechaMasDeLoDeclarado(t *testing.T) {
	// 1. EL VALOR EXACTO. Sin esto, lo de abajo pasaria con cualquier
	//    conversion que no se pasara de largo, incluida una que devolviera cero.
	seis, err := aDuracion("P6M")
	if err != nil {
		t.Fatalf("P6M no cruza: %v", err)
	}
	if quiero := 6 * DiasPorMes * 24 * time.Hour; seis != quiero {
		t.Fatalf("P6M cruza como %v y el puente declara %d dias por mes, o sea %v",
			seis, DiasPorMes, quiero)
	}

	// 2. CONTRA EL CALENDARIO DE VERDAD, mes a mes de un ano entero. Un semestre
	//    real dura entre 181 y 184 dias segun de que dia arranque, asi que el
	//    error depende del punto de partida y medirlo en uno solo no dice nada.
	//
	//    SE RECORREN LOS DOCE ARRANQUES y ademas un bisiesto, porque febrero es
	//    justo donde la aproximacion de 30 dias mas se aleja.
	for _, ano := range []int{2026, 2028 /* bisiesto */} {
		for mes := time.January; mes <= time.December; mes++ {
			desde := time.Date(ano, mes, 1, 0, 0, 0, 0, time.UTC)
			real := desde.AddDate(0, 6, 0)
			delPuente := desde.Add(seis)

			// LA DIRECCION, que es la mitad que importa.
			if delPuente.After(real) {
				t.Errorf("desde %s, el puente caduca el %s y el semestre real acaba el %s: "+
					"RETRASA la caducidad.\n"+
					"  Un expediente que dice que algo consta cuando ya habia caducado es lo "+
					"unico que esta aproximacion no puede hacer. Adelantar es el lado seguro; "+
					"retrasar no lo es.",
					desde.Format("2006-01-02"), delPuente.Format("2006-01-02"),
					real.Format("2006-01-02"))
			}
			// Y CUANTO, contra el numero que el godoc publica.
			dias := int(real.Sub(delPuente).Hours() / 24)
			if dias < 1 || dias > 4 {
				t.Errorf("desde %s, el puente adelanta la caducidad %d dias y el godoc del "+
					"puente declara entre 1 y 4.\n"+
					"  Si el numero real ha cambiado, se corrige el godoc Y este test a la vez: "+
					"una aproximacion cuyo margen publicado no es el suyo es peor que una sin "+
					"margen publicado, porque parece medida.",
					desde.Format("2006-01-02"), dias)
			}
		}
	}
}

// EL CONTROL NEGATIVO, sobre la afirmacion y no sobre el detector.
//
// La de arriba comprueba que el puente adelanta poco. Esta comprueba que el test
// SABRIA verlo si no lo hiciera: con un mes contado a 31 dias, la conversion
// pasa a retrasar la caducidad, que es la direccion prohibida. Sin esta mitad,
// un `aDuracion` que devolviera exactamente el calendario real tambien pasaria
// la de arriba, y entonces el test no estaria afirmando nada sobre la
// aproximacion, que es lo unico que existe para vigilar.
func TestSiUnMesSeContaraDeMasLaCaducidadSeRetrasaria(t *testing.T) {
	const deMas = 31 // lo que NO hace el puente
	desde := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)
	real := desde.AddDate(0, 6, 0)
	malo := desde.Add(6 * deMas * 24 * time.Hour)
	if !malo.After(real) {
		t.Fatalf("con %d dias por mes la caducidad del 1 de febrero saldria el %s y el "+
			"semestre real acaba el %s, o sea que NO retrasaria.\n"+
			"  Este control existe para demostrar que la comprobacion de direccion de "+
			"TestElPuenteNoMueveUnaFechaMasDeLoDeclarado puede ponerse roja. Si esto pasa, "+
			"aquella no esta afirmando nada.",
			deMas, malo.Format("2006-01-02"), real.Format("2006-01-02"))
	}
	// Y la otra direccion: con los 30 que el puente SI usa, no retrasa.
	bueno := desde.Add(6 * DiasPorMes * 24 * time.Hour)
	if bueno.After(real) {
		t.Errorf("con los %d dias por mes del puente, la caducidad del 1 de febrero "+
			"retrasa: %s contra %s", DiasPorMes,
			bueno.Format("2006-01-02"), real.Format("2006-01-02"))
	}
}
