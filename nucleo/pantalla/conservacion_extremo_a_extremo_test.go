package pantalla

import (
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// UNA CADENA, UNA LEY: de corpus.Cargar hasta el cubo impreso.
//
// POR QUE HAY UNA TERCERA LEY DE CONSERVACION teniendo ya dos que cuadraban. El
// 29-08-2026 se perdieron 46 relojes en la derivacion y las dos que existian
// dieron verde todo el rato. No fallaron: es que el hueco vivia EXACTAMENTE
// ENTRE LAS DOS.
//
//	particion por TIEMPO    en vigor + estrenan + ya cesados + empiezan despues
//	                        + ilegibles = instalados.        Cuadraba.
//	particion por ALCANCE   alcanzados + no alcanzados = en vigor.  Cuadraba.
//	                        ... y entre «te alcanza» y «sale en pantalla»,
//	                        nada.
//
// Dos particiones que se equilibran cada una por su lado no componen una ley:
// componen dos leyes con una junta sin vigilar, y la junta es donde se rompe.
//
// ESTA MIRA LA CADENA ENTERA y hace DOS preguntas, no una:
//
//  1. ¿tiene todo reloj instalado exactamente un destino?
//  2. ¿el destino que promete una fila TIENE esa fila?
//
// La segunda es la que cierra la junta. Etiquetar es barato: sin ella, una
// etiqueta puesta sin fila detras dejaria la ley en verde y el reloj sin verse,
// que es literalmente el fallo que esto persigue.
func TestTodoRelojInstaladoAcabaEnExactamenteUnDestino(t *testing.T) {
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatalf("no puedo cargar el corpus publicado: %v", err)
	}
	ahora := time.Date(2026, 8, 27, 9, 0, 0, 0, time.UTC)

	// SIN NI UN HECHO, que es el dia uno de un cliente y el peor caso: ninguna
	// cadencia tiene todavia su fecha de arranque. Es el escenario exacto en el
	// que el fallo se comia 46 relojes.
	cal := Derivar12Meses(ps, TodoAplica, ventana.Hechos{}, ahora)

	if sin := RelojesSinDestino(ps, cal); len(sin) > 0 {
		muestra := sin
		if len(muestra) > 8 {
			muestra = muestra[:8]
		}
		t.Errorf(`%d relojes instalados no acaban en ningun destino.

  Ni fila, ni motivo, ni cubo con nombre: desaparecen. Para quien lee el
  calendario es indistinguible de que no existieran.

  Las primeras: %v`, len(sin), muestra)
	}

	// EL VOCABULARIO ES CERRADO: un destino nuevo que nadie declare rompe esto,
	// que es lo que se quiere el dia que alguien anada una rama a la derivacion.
	for id, d := range cal.Destinos {
		if !DestinosConocidos[d] {
			t.Errorf("%s acaba en el destino %q, que no esta en el vocabulario cerrado", id, d)
		}
	}

	// LA SEGUNDA PREGUNTA, la que cierra la junta.
	if rotas := DestinosQuePrometenFilaSinTenerla(cal); len(rotas) > 0 {
		t.Errorf(`%d destinos prometen una fila en pantalla y no la tienen: %v

  Una etiqueta sin fila detras deja esta ley en verde y el reloj sin verse, que
  es exactamente el fallo que persigue.`, len(rotas), rotas)
	}

	// Suelo: sin el, un corpus que dejara de cargar relojes daria verde sin
	// haber comprobado ni uno.
	if len(cal.Destinos) < 50 {
		t.Fatalf("solo %d relojes etiquetados: el recorrido no esta viendo el corpus y esta "+
			"ley estaria comprobando el vacio", len(cal.Destinos))
	}
	if !t.Failed() {
		t.Logf("%d relojes instalados, todos con destino", len(cal.Destinos))
	}
}

// La ley recorre TODOS los destinos, no solo los dos comodos.
//
// Sin este test, la de arriba podria cumplirse con el corpus entero cayendo en
// un solo cubo, y no demostraria que la derivacion distinga nada. Se monta un
// corpus con un inquilino por destino y se exige que aparezcan todos.
func TestCadaDestinoTieneSuInquilino(t *testing.T) {
	derogada := plazoDe("d.ya_ceso", "1", "x", "P10D", "2020-01-01")
	derogada.Vigencia.Hasta = "2021-01-01"
	ilegible := plazoDe("d.ilegible", "2", "x", "P10D", "no-es-una-fecha")

	ps := []*corpus.Paquete{paqueteConRelojes("urn:demo:destinos",
		plazoDe("d.con_fecha", "3", "x", "P10D", "2020-01-01"),
		plazoDe("d.mas_alla", "4", "x", "P800D", "2020-01-01"),
		plazoDe("d.no_te_alcanza", "5", "x", "P10D", "2020-01-01"),
		plazoDe("d.estrena", "6", "x", "P10D", "2026-12-01"),
		plazoDe("d.empieza_despues", "7", "x", "P10D", "2030-01-01"),
		derogada,
		ilegible,
		periodicaDe("d.sin_fecha", "8", "hecho_que_no_consta", "P12M", "2020-01-01"),
		periodicaDe("d.vencida", "9", "y", "P12M", "2020-01-01"),
	)}
	aplica := func(id string) (bool, bool) { return id != "d.no_te_alcanza", false }
	cal := Derivar12Meses(ps, aplica, ventana.Hechos{
		"x": ahoraDePrueba,
		"y": ahoraDePrueba.AddDate(-3, 0, 0),
	}, ahoraDePrueba)

	if sin := RelojesSinDestino(ps, cal); len(sin) > 0 {
		t.Fatalf("relojes sin destino: %v", sin)
	}
	vistos := map[Destino]string{}
	for id, d := range cal.Destinos {
		vistos[d] = id
	}
	for d := range DestinosConocidos {
		if _, ok := vistos[d]; !ok {
			t.Errorf(`el destino %q no lo ejercita ningun caso del corpus de prueba.

  Una ley que solo recorre los cubos comodos se cumple sola. Si este destino ya
  no puede darse, se quita del vocabulario; si puede, el corpus de prueba tiene
  que traer su inquilino.`, d)
		}
	}
	if rotas := DestinosQuePrometenFilaSinTenerla(cal); len(rotas) > 0 {
		t.Errorf("destinos que prometen fila sin tenerla: %v", rotas)
	}
}

// CONTROL NEGATIVO: la ley sabe decir que no.
//
// Se le quita a mano una etiqueta y se exige que el detector la eche en falta.
// Sin esto, el verde de arriba podria venir de un recorrido que no mira nada.
func TestLaLeyDeDestinosCazaUnRelojSinEtiqueta(t *testing.T) {
	ps := []*corpus.Paquete{paqueteConRelojes("urn:demo:falta",
		plazoDe("f.uno", "1", "x", "P10D", "2020-01-01"),
		plazoDe("f.dos", "2", "x", "P10D", "2020-01-01"),
	)}
	cal := Derivar12Meses(ps, TodoAplica, ventana.Hechos{"x": ahoraDePrueba}, ahoraDePrueba)
	if len(RelojesSinDestino(ps, cal)) != 0 {
		t.Fatal("el corpus de prueba tenia que salir entero etiquetado")
	}
	delete(cal.Destinos, "f.dos")
	sin := RelojesSinDestino(ps, cal)
	if len(sin) != 1 || sin[0] != "f.dos" {
		t.Fatalf("quitada una etiqueta, el detector tenia que echarla en falta y devolvio %v", sin)
	}

	// Y el otro detector: una etiqueta que promete fila sin tenerla.
	cal.Destinos["f.dos"] = DestinoVencida // no hay ninguna vencida en este corpus
	rotas := DestinosQuePrometenFilaSinTenerla(cal)
	if len(rotas) != 1 {
		t.Fatalf("una etiqueta que promete una fila inexistente tenia que saltar, y salieron "+
			"%d: %v", len(rotas), rotas)
	}
}
