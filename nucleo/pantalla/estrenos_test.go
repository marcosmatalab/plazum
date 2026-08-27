package pantalla

import (
	"testing"

	"plazum/nucleo/corpus"
	"plazum/nucleo/ventana"
)

// LO QUE EMPIEZA A OBLIGAR DENTRO DE LA VENTANA.
//
// EL AGUJERO. La derivacion hacia `if !vigente { continue }`, un continue mudo,
// y con el desaparecia del calendario cualquier obligacion cuya vigencia empieza
// manana. El fichero de la derivacion declara en su cabecera que "lo que no
// produce fecha NO desaparece: sale en SinFecha con el motivo", y esta era la
// unica rama que lo incumplia.
//
// MEDIDO SOBRE TRAFICO REAL, no sobre una mutacion: el perfil de arranque
// es-fabricante-software no ensenaba NI UNA de las dos notificaciones del art.
// 14 del Reglamento (UE) 2024/2847, que empiezan a aplicarse el 11-09-2026,
// quince dias despues del dia en que se midio. El producto entero existe para
// decir esa fecha y era justo la que se callaba.

func TestLoQueEmpiezaAObligarDentroDeLaVentanaNoDesaparece(t *testing.T) {
	// ahoraDePrueba es 2026-09-01, asi que la ventana llega a 2027-09-01.
	ps := []*corpus.Paquete{paqueteConRelojes("urn:demo:e",
		plazoDe("e.estrena_dentro", "14", "x", "P10D", "2026-09-11"),
		plazoDe("e.estrena_fuera", "15", "x", "P10D", "2030-01-01"),
	)}
	cal := Derivar12Meses(ps, TodoAplica, ventana.Hechos{"x": ahoraDePrueba}, ahoraDePrueba)

	if len(cal.Estrenos) != 1 {
		t.Fatalf("%d estrenos, esperaba 1: la de 2026-09-11 entra y la de 2030 no", len(cal.Estrenos))
	}
	e := cal.Estrenos[0]
	if e.Obligacion != "e.estrena_dentro" {
		t.Errorf("el estreno es de %s y tenia que ser de e.estrena_dentro", e.Obligacion)
	}
	if got := e.Desde.Format("2006-01-02"); got != "2026-09-11" {
		t.Errorf("el estreno dice %s y la vigencia empieza el 2026-09-11", got)
	}
	if cal.HitosQueEstrenan != 1 {
		t.Errorf("HitosQueEstrenan %d, esperaba 1", cal.HitosQueEstrenan)
	}
	// Y NO se cuela en "en vigor": en el instante del calculo no lo estaba, y
	// si se sumara ahi la contabilidad dejaria de decir lo que su nombre dice.
	if cal.HitosEnVigor != 0 {
		t.Errorf("relojes en vigor %d, esperaba 0: ninguna de las dos obliga hoy", cal.HitosEnVigor)
	}
	// Ni produce una FECHA. Un estreno no es un vencimiento: pintarlo como tal
	// le diria al operador que entregue algo ese dia, que es falso.
	if cal.Total() != 0 {
		t.Errorf("%d fechas, esperaba 0: un estreno no es un vencimiento", cal.Total())
	}
}

// CONTROL NEGATIVO: la aplicabilidad manda tambien en los estrenos. El estreno
// de algo que no te alcanza no es noticia tuya, y ensenarlo seria el mismo fallo
// que el calendario evita en todas las demas ramas.
func TestUnEstrenoDeAlgoQueNoTeAlcanzaNoSePinta(t *testing.T) {
	ps := []*corpus.Paquete{paqueteConRelojes("urn:demo:e",
		plazoDe("e.estrena_dentro", "14", "x", "P10D", "2026-09-11"),
	)}
	nada := func(string) (bool, bool) { return false, false }
	cal := Derivar12Meses(ps, nada, ventana.Hechos{"x": ahoraDePrueba}, ahoraDePrueba)
	if len(cal.Estrenos) != 0 {
		t.Fatalf("%d estrenos, esperaba 0: la aplicabilidad dijo que no", len(cal.Estrenos))
	}
}

// CONTROL NEGATIVO: una obligacion DEROGADA no estrena. Su vigencia empezo en el
// pasado, asi que el `desde` no cae dentro de la ventana y no puede confundirse
// con una que arranca. Es la otra mitad de !vigente, y la que un `if !vigente`
// convertido en estreno a lo bruto habria pintado como novedad.
func TestUnaObligacionDerogadaNoEstrena(t *testing.T) {
	o := plazoDe("e.derogada", "9", "x", "P10D", "2020-01-01")
	o.Vigencia.Hasta = "2021-01-01"
	ps := []*corpus.Paquete{paqueteConRelojes("urn:demo:e", o)}
	cal := Derivar12Meses(ps, TodoAplica, ventana.Hechos{"x": ahoraDePrueba}, ahoraDePrueba)
	if len(cal.Estrenos) != 0 {
		t.Fatalf("%d estrenos: una obligacion derogada no empieza nada", len(cal.Estrenos))
	}
}

// El orden es por fecha de estreno y luego por identidades del dato (marco y
// obligacion). Sin desempate, dos estrenos del mismo dia salen en el orden en
// que se recorra el corpus, que no es un orden: es el mismo fallo que ya se
// midio para las fechas.
func TestElOrdenDeLosEstrenosEsTotal(t *testing.T) {
	ps := []*corpus.Paquete{
		paqueteConRelojes("urn:demo:z", plazoDe("z.uno", "1", "x", "P10D", "2026-09-11")),
		paqueteConRelojes("urn:demo:a", plazoDe("a.uno", "1", "x", "P10D", "2026-09-11")),
		paqueteConRelojes("urn:demo:m", plazoDe("m.uno", "1", "x", "P10D", "2026-09-10")),
	}
	quiero := []string{"m.uno", "a.uno", "z.uno"}
	for vuelta := 0; vuelta < 20; vuelta++ {
		cal := Derivar12Meses(ps, TodoAplica, ventana.Hechos{"x": ahoraDePrueba}, ahoraDePrueba)
		if len(cal.Estrenos) != 3 {
			t.Fatalf("%d estrenos, esperaba 3", len(cal.Estrenos))
		}
		for i, q := range quiero {
			if cal.Estrenos[i].Obligacion != q {
				t.Fatalf("vuelta %d: en la posicion %d sale %s y esperaba %s",
					vuelta, i, cal.Estrenos[i].Obligacion, q)
			}
		}
	}
}
