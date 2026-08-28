package pantalla

import (
	"testing"

	"plazum/nucleo/corpus"
	"plazum/nucleo/ventana"
)

// LA CONTABILIDAD DEL CALENDARIO TIENE QUE CUADRAR, Y ESO SE COMPRUEBA SUMANDO.
//
// POR QUE ESTE TEST ES LA PUERTA DE LA FAMILIA ENTERA. El fallo original (`if
// !vigente { continue }`, un continue mudo que se llevaba del calendario lo que
// empieza a obligar manana) no lo cazaba ningun test porque TODOS preguntaban
// por lo que sale y NINGUNO por lo que se cae. Contar filas no lo habria visto:
// las filas que quedaban eran correctas.
//
// Lo que si lo ve es una LEY DE CONSERVACION: si cada hito instalado tiene que
// aparecer en exactamente un cubo, un descarte mudo rompe la suma en el acto,
// diga lo que diga la lista. Es la unica forma de test que crece sola: el dia
// que alguien anada una rama nueva a la derivacion y se olvide de contarla,
// esto se pone rojo sin que nadie tenga que acordarse de escribir un caso.
//
// SON DOS PARTICIONES Y NO UNA, y mezclarlas seria el error. La del TIEMPO
// (en vigor, estrena, ya ceso, empieza despues, ilegible) son hechos del
// calendario y cubren el corpus entero. La del ALCANCE (te alcanza o no) es una
// respuesta sobre TI y solo tiene sentido sobre lo que hoy esta en vigor.

// corpusDeLosCincoCubos monta un corpus sintetico con una obligacion de cada
// caso que la derivacion sabe distinguir. La ventana de ahoraDePrueba
// (2026-09-01) llega hasta 2027-09-01.
func corpusDeLosCincoCubos() []*corpus.Paquete {
	derogada := plazoDe("c.ya_ceso", "1", "x", "P10D", "2020-01-01")
	derogada.Vigencia.Hasta = "2021-01-01"

	cesaDentro := plazoDe("c.cesa_dentro", "2", "x", "P10D", "2020-01-01")
	cesaDentro.Vigencia.Hasta = "2027-03-15"

	ilegible := plazoDe("c.ilegible", "3", "x", "P10D", "no-es-una-fecha")

	return []*corpus.Paquete{paqueteConRelojes("urn:demo:cubos",
		plazoDe("c.en_vigor", "0", "x", "P10D", "2020-01-01"),
		derogada,
		cesaDentro,
		plazoDe("c.estrena_dentro", "4", "x", "P10D", "2026-12-01"),
		plazoDe("c.empieza_despues", "5", "x", "P10D", "2030-01-01"),
		ilegible,
	)}
}

func TestLaContabilidadDelCalendarioCuadraPorTiempo(t *testing.T) {
	ps := corpusDeLosCincoCubos()
	cal := Derivar12Meses(ps, TodoAplica, ventana.Hechos{"x": ahoraDePrueba}, ahoraDePrueba)

	suma := cal.HitosEnVigor + cal.HitosQueEstrenan + cal.HitosYaCesados +
		cal.HitosQueEmpiezanDespues + cal.HitosConVigenciaIlegible
	if suma != cal.HitosDelCorpus {
		t.Fatalf("la particion por tiempo NO cuadra: %d instalados y %d repartidos "+
			"(en vigor %d, estrenan %d, ya cesados %d, empiezan despues %d, ilegibles %d). "+
			"Un hito que no cae en ningun cubo es un descarte mudo, que es el fallo que "+
			"esta ley existe para cazar",
			cal.HitosDelCorpus, suma, cal.HitosEnVigor, cal.HitosQueEstrenan,
			cal.HitosYaCesados, cal.HitosQueEmpiezanDespues, cal.HitosConVigenciaIlegible)
	}
	// Y que cada cubo tenga a su inquilino: sin esto, la ley se cumple sola con
	// todo en el mismo cubo y no demuestra que la derivacion distinga nada.
	for nombre, n := range map[string]int{
		"en vigor":          cal.HitosEnVigor,
		"estrenan":          cal.HitosQueEstrenan,
		"ya cesados":        cal.HitosYaCesados,
		"empiezan despues":  cal.HitosQueEmpiezanDespues,
		"vigencia ilegible": cal.HitosConVigenciaIlegible,
	} {
		if n == 0 {
			t.Errorf("el cubo %q esta vacio, asi que la suma cuadra sin haber distinguido "+
				"ese caso: el corpus de prueba trae uno de cada", nombre)
		}
	}
}

func TestLaContabilidadDelCalendarioCuadraPorAlcance(t *testing.T) {
	ps := corpusDeLosCincoCubos()
	// Solo una de las que estan en vigor te alcanza.
	soloUna := func(id string) (bool, bool) { return id == "c.en_vigor", false }
	cal := Derivar12Meses(ps, soloUna, ventana.Hechos{"x": ahoraDePrueba}, ahoraDePrueba)

	if got := cal.HitosAplicables + cal.HitosNoAlcanzados; got != cal.HitosEnVigor {
		t.Fatalf("la particion por alcance NO cuadra: %d en vigor y %d repartidos "+
			"(alcanzados %d, no alcanzados %d)",
			cal.HitosEnVigor, got, cal.HitosAplicables, cal.HitosNoAlcanzados)
	}
	if cal.HitosNoAlcanzados == 0 {
		t.Error("con una sola obligacion alcanzada, el resto de lo vigente tiene que contarse " +
			"como no alcanzado: callarlo es el descarte mudo mas grande de la derivacion")
	}
}

// EL CESE, espejo del estreno. Lo que hoy te obliga y dejara de obligarte.
func TestLoQueDejaDeObligarDentroDeLaVentanaSeDice(t *testing.T) {
	ps := corpusDeLosCincoCubos()
	cal := Derivar12Meses(ps, TodoAplica, ventana.Hechos{"x": ahoraDePrueba}, ahoraDePrueba)

	if len(cal.Ceses) != 1 {
		t.Fatalf("%d ceses, esperaba 1: solo c.cesa_dentro deja de obligar dentro de la "+
			"ventana. La derogada en 2021 ceso ANTES y no es una transicion de estos doce "+
			"meses; las abiertas no cesan", len(cal.Ceses))
	}
	c := cal.Ceses[0]
	if c.Obligacion != "c.cesa_dentro" {
		t.Errorf("el cese es de %s y tenia que ser de c.cesa_dentro", c.Obligacion)
	}
	if got := c.Hasta.Format("2006-01-02"); got != "2027-03-15" {
		t.Errorf("el cese dice %s y la vigencia acaba el 2027-03-15", got)
	}
	// UN CESE NO ES UN VENCIMIENTO. Si se colara en las fechas, la agenda del
	// operador tendria un plazo que no existe.
	for _, m := range cal.Meses {
		for _, f := range m.Fechas {
			if f.Vence.Equal(c.Hasta) && f.Obligacion == c.Obligacion {
				t.Error("el cese se ha colado como fecha de vencimiento: ese dia no hay " +
					"nada que entregar")
			}
		}
	}
	// Y sigue contando como EN VIGOR, porque hoy lo esta. Es la diferencia con
	// un estreno y la razon de que se cuente aparte.
	if cal.HitosQueCesan != 1 {
		t.Errorf("HitosQueCesan %d, esperaba 1", cal.HitosQueCesan)
	}
}

// CONTROL NEGATIVO: la aplicabilidad manda tambien en los ceses. Enterarte de
// que deja de obligarte algo que nunca te obligo no es una buena noticia, es
// ruido.
func TestUnCeseDeAlgoQueNoTeAlcanzaNoSePinta(t *testing.T) {
	ps := corpusDeLosCincoCubos()
	nada := func(string) (bool, bool) { return false, false }
	cal := Derivar12Meses(ps, nada, ventana.Hechos{"x": ahoraDePrueba}, ahoraDePrueba)
	if len(cal.Ceses) != 0 {
		t.Fatalf("%d ceses, esperaba 0: la aplicabilidad dijo que no", len(cal.Ceses))
	}
}

// CONTROL NEGATIVO: una vigencia ABIERTA por arriba no cesa nunca.
//
// LO QUE ESTE TEST NO GUARDA, y conviene decirlo porque su primera version
// afirmaba lo contrario. Se escribio diciendo que protegia el booleano de
// FinDeVigencia (invariante 8), y la mutacion lo desmintio: haciendo que
// devolviera `true` con el cero de time.Time, esto SEGUIA VERDE, porque el
// caller comprueba tambien `fin.After(ahora)` y el ano 1 no esta despues de
// hoy. El booleano no es load-bearing aqui.
//
// Lo que si guarda es que esta rama del calendario no invente ceses, que es
// util. El contrato del booleano se comprueba donde se declara, en
// nucleo/corpus (TestFinDeVigenciaDistingueLaAbiertaDeLaQueAcaba), y ese si se
// pone rojo con la mutacion.
func TestUnaVigenciaAbiertaNoCesaNunca(t *testing.T) {
	ps := []*corpus.Paquete{paqueteConRelojes("urn:demo:abierta",
		plazoDe("a.sin_fin", "1", "x", "P10D", "2020-01-01"),
	)}
	cal := Derivar12Meses(ps, TodoAplica, ventana.Hechos{"x": ahoraDePrueba}, ahoraDePrueba)
	if len(cal.Ceses) != 0 || cal.HitosQueCesan != 0 {
		t.Fatalf("una vigencia sin `hasta` no acaba: %d ceses, contador %d",
			len(cal.Ceses), cal.HitosQueCesan)
	}
}

// El orden de los ceses es total, por lo mismo que el de las fechas y el de los
// estrenos: dos ceses del mismo dia salen si no en el orden del recorrido del
// corpus, que no es un orden.
func TestElOrdenDeLosCesesEsTotal(t *testing.T) {
	conFin := func(id, art, hasta string) corpus.Obligacion {
		o := plazoDe(id, art, "x", "P10D", "2020-01-01")
		o.Vigencia.Hasta = hasta
		return o
	}
	ps := []*corpus.Paquete{
		paqueteConRelojes("urn:demo:z", conFin("z.uno", "1", "2027-03-15")),
		paqueteConRelojes("urn:demo:a", conFin("a.uno", "1", "2027-03-15")),
		paqueteConRelojes("urn:demo:m", conFin("m.uno", "1", "2027-01-10")),
	}
	quiero := []string{"m.uno", "a.uno", "z.uno"}
	for vuelta := 0; vuelta < 20; vuelta++ {
		cal := Derivar12Meses(ps, TodoAplica, ventana.Hechos{"x": ahoraDePrueba}, ahoraDePrueba)
		if len(cal.Ceses) != 3 {
			t.Fatalf("%d ceses, esperaba 3", len(cal.Ceses))
		}
		for i, q := range quiero {
			if cal.Ceses[i].Obligacion != q {
				t.Fatalf("vuelta %d: en la posicion %d sale %s y esperaba %s",
					vuelta, i, cal.Ceses[i].Obligacion, q)
			}
		}
	}
}
