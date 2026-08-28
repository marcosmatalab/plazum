package pantalla

import (
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// El corpus de estos tests es SINTETICO a proposito. Un test de la derivacion
// que dependa del corpus publicado se pone rojo cada vez que alguien transcribe
// un articulo, y entonces deja de decir si la derivacion esta bien para decir si
// el corpus ha cambiado. Lo segundo ya lo dicen los dorados.

var ahoraDePrueba = time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)

func regimenExacto() corpus.RegimenSpec {
	return corpus.RegimenSpec{Computo: "naturales", Cierre: "exacto", Traslado: "ninguno"}
}

// paqueteConRelojes arma un paquete con las obligaciones que se le pidan.
func paqueteConRelojes(urn string, obs ...corpus.Obligacion) *corpus.Paquete {
	return &corpus.Paquete{
		URN: urn, Version: "1", Clase: corpus.Transcrito,
		Vigencia:     corpus.Vigencia{Desde: "2020-01-01"},
		Obligaciones: obs,
	}
}

func plazoDe(id, articulo, hecho, limite, desde string) corpus.Obligacion {
	return corpus.Obligacion{
		ID: id, Articulo: articulo, ClaseE2E: "notificatoria",
		Vigencia: corpus.Vigencia{Desde: desde},
		Temporalidad: &corpus.Temporalidad{
			Primitiva: "plazo", Hito: "limite", Limite: limite,
			Regimen: regimenExacto(), Disparador: map[string]string{"hecho": hecho},
		},
	}
}

// EL ORDEN TOTAL, y el desempate es lo que se mide.
//
// Quince relojes anuales arrancados el mismo dia vencen en el MISMO instante. Sin
// desempate salen barajados, porque el recorrido de los paquetes viene de un
// mapa, y un dorado byte a byte los caza una vez de cada tres: la peor forma de
// rojo que hay, porque se lee como un fallo intermitente del entorno.
func TestElOrdenDelCalendarioEsTotalYNoDependeDelRecorrido(t *testing.T) {
	// Tres obligaciones que vencen EXACTAMENTE a la vez, declaradas al reves
	// del orden que tienen que salir.
	ps := []*corpus.Paquete{
		paqueteConRelojes("urn:demo:c", plazoDe("c.tres", "3", "x", "P10D", "2020-01-01")),
		paqueteConRelojes("urn:demo:a", plazoDe("a.uno", "1", "x", "P10D", "2020-01-01")),
		paqueteConRelojes("urn:demo:b", plazoDe("b.dos", "2", "x", "P10D", "2020-01-01")),
	}
	hechos := ventana.Hechos{"x": ahoraDePrueba}

	quiero := []string{"a.uno", "b.dos", "c.tres"}
	for vuelta := 0; vuelta < 20; vuelta++ {
		cal := Derivar12Meses(ps, TodoAplica, hechos, ahoraDePrueba)
		if cal.Total() != 3 {
			t.Fatalf("vuelta %d: %d fechas y esperaba 3", vuelta, cal.Total())
		}
		var got []string
		for _, m := range cal.Meses {
			for _, f := range m.Fechas {
				got = append(got, f.Obligacion)
			}
		}
		for i := range quiero {
			if got[i] != quiero[i] {
				t.Fatalf("vuelta %d: orden %v, esperaba %v. Tres fechas iguales sin desempate "+
					"salen en un orden distinto en cada ejecucion", vuelta, got, quiero)
			}
		}
	}
}

// LA VIGENCIA MANDA ANTES QUE EL RELOJ. Una obligacion que todavia no obliga no
// es una fecha del calendario de nadie, por mucho que su reloj sepa calcular.
func TestElCalendarioNoEnsenaObligacionesQueTodaviaNoObligan(t *testing.T) {
	ps := []*corpus.Paquete{paqueteConRelojes("urn:demo:v",
		plazoDe("v.ya", "1", "x", "P10D", "2020-01-01"),
		plazoDe("v.todavia_no", "2", "x", "P10D", "2030-01-01"),
		func() corpus.Obligacion {
			o := plazoDe("v.derogada", "3", "x", "P10D", "2020-01-01")
			o.Vigencia.Hasta = "2021-01-01"
			return o
		}(),
	)}
	cal := Derivar12Meses(ps, TodoAplica, ventana.Hechos{"x": ahoraDePrueba}, ahoraDePrueba)
	if cal.HitosDelCorpus != 3 {
		t.Errorf("relojes del corpus %d, esperaba 3", cal.HitosDelCorpus)
	}
	if cal.HitosEnVigor != 1 {
		t.Errorf("relojes en vigor %d, esperaba 1: la futura y la derogada no cuentan",
			cal.HitosEnVigor)
	}
	if cal.Total() != 1 {
		t.Fatalf("%d fechas, esperaba 1", cal.Total())
	}
	if got := cal.Meses[0].Fechas[0].Obligacion; got != "v.ya" {
		t.Errorf("la fecha es de %s y tenia que ser de v.ya", got)
	}
}

// LAS DOS FORMAS DE LA NADA EN LA FRONTERA DE APLICABILIDAD, que es el
// invariante 8. Una funcion nil NO puede significar "todo aplica": ese es el
// valor cero permisivo, y le ensenaria a un banco las obligaciones de un
// fabricante de productos sanitarios.
func TestSinFuncionDeAplicabilidadNoSeDerivaNada(t *testing.T) {
	ps := []*corpus.Paquete{paqueteConRelojes("urn:demo:n",
		plazoDe("n.una", "1", "x", "P10D", "2020-01-01"))}
	hechos := ventana.Hechos{"x": ahoraDePrueba}

	t.Run("nil: no deriva nada", func(t *testing.T) {
		cal := Derivar12Meses(ps, nil, hechos, ahoraDePrueba)
		if cal.Total() != 0 {
			t.Errorf("%d fechas con la funcion nil. El valor cero tiene que ser el RESTRICTIVO: "+
				"tratar nil como \"todo aplica\" es exactamente el fallo del invariante 8",
				cal.Total())
		}
		if cal.HitosEnVigor != 1 {
			t.Errorf("y aun asi el reloj tiene que contarse como en vigor (%d): el calendario "+
				"dice cuantos no alcanzo, no los esconde", cal.HitosEnVigor)
		}
	})
	t.Run("TodoAplica: se pide en voz alta y entonces si", func(t *testing.T) {
		cal := Derivar12Meses(ps, TodoAplica, hechos, ahoraDePrueba)
		if cal.Total() != 1 {
			t.Errorf("%d fechas con TodoAplica, esperaba 1", cal.Total())
		}
	})
	t.Run("una funcion que dice que no: tampoco", func(t *testing.T) {
		nada := func(string) (bool, bool) { return false, false }
		if cal := Derivar12Meses(ps, nada, hechos, ahoraDePrueba); cal.Total() != 0 {
			t.Errorf("%d fechas", cal.Total())
		}
	})
}

// LO QUE NO DA FECHA NO DESAPARECE. Es la propiedad que separa una agenda
// honesta de una que le hace creer al que la lee que lo que no ve no existe.
func TestLoQueNoProduceFechaSaleConSuMotivo(t *testing.T) {
	sinPlazo := corpus.Obligacion{
		ID: "s.sin_numero", Articulo: "67.1", ClaseE2E: "notificatoria",
		Vigencia: corpus.Vigencia{Desde: "2020-01-01"},
		Temporalidad: &corpus.Temporalidad{
			Primitiva: "plazo", Regimen: regimenExacto(),
			Disparador: map[string]string{"hecho": "x"},
			Hitos: []corpus.HitoSpec{{ID: "notificacion", Limite: "indeterminado",
				Nota: "la norma dice de forma inmediata y no da numero"}},
		},
	}
	pendiente := plazoDe("s.pendiente", "2", "hecho_que_no_consta", "P10D", "2020-01-01")
	sinEjecutor := corpus.Obligacion{
		ID: "s.sin_ejecutor", Articulo: "3", ClaseE2E: "observable",
		Vigencia: corpus.Vigencia{Desde: "2020-01-01"},
		Temporalidad: &corpus.Temporalidad{
			Primitiva: "continua", Regimen: regimenExacto(),
			Disparador: map[string]string{"hecho": "x"},
		},
	}
	ps := []*corpus.Paquete{paqueteConRelojes("urn:demo:s", sinPlazo, pendiente, sinEjecutor)}
	cal := Derivar12Meses(ps, TodoAplica, ventana.Hechos{"x": ahoraDePrueba}, ahoraDePrueba)

	if cal.Total() != 0 {
		t.Fatalf("%d fechas y no tenia que haber ninguna", cal.Total())
	}
	quiero := map[string]string{
		"s.sin_numero":   MotivoSinPlazoLegal,
		"s.pendiente":    MotivoPendienteDeHecho,
		"s.sin_ejecutor": MotivoSinEjecutor,
	}
	visto := map[string]string{}
	for _, s := range cal.SinFecha {
		visto[s.Obligacion] = s.Motivo
		if s.Regla == "" {
			t.Errorf("%s sale sin fecha y sin decir por que. Un hueco sin motivo es peor que "+
				"un hueco", s.Obligacion)
		}
	}
	for id, motivo := range quiero {
		if visto[id] != motivo {
			t.Errorf("%s sale con motivo %q y tenia que ser %q", id, visto[id], motivo)
		}
	}
	if len(cal.SinFecha) != 3 {
		t.Errorf("%d filas sin fecha, esperaba 3: %+v", len(cal.SinFecha), cal.SinFecha)
	}
}

// La ventana son doce meses, y lo que cae fuera SE CUENTA. Si solo se filtrara,
// un corpus con treinta relojes a dos anos vista se leeria como un corpus vacio.
func TestLoQueCaeFueraDeLaVentanaSeCuentaYNoSePinta(t *testing.T) {
	ps := []*corpus.Paquete{paqueteConRelojes("urn:demo:f",
		plazoDe("f.dentro", "1", "x", "P10D", "2020-01-01"),
		plazoDe("f.fuera", "2", "x", "P800D", "2020-01-01"),
	)}
	cal := Derivar12Meses(ps, TodoAplica, ventana.Hechos{"x": ahoraDePrueba}, ahoraDePrueba)
	if cal.Total() != 1 {
		t.Fatalf("%d fechas, esperaba 1", cal.Total())
	}
	if cal.FueraDeLaVentana != 1 {
		t.Errorf("fuera de la ventana %d, esperaba 1. Filtrar sin contar convierte un corpus "+
			"lleno de relojes lejanos en un calendario vacio sin explicacion",
			cal.FueraDeLaVentana)
	}
}

// La marca de SUPUESTO viaja hasta la fila. Un calendario que no distingue un
// hecho declarado de uno supuesto convierte una conjetura en una obligacion.
func TestLaMarcaDeSupuestoLlegaHastaLaFila(t *testing.T) {
	ps := []*corpus.Paquete{paqueteConRelojes("urn:demo:sup",
		plazoDe("sup.declarada", "1", "x", "P10D", "2020-01-01"),
		plazoDe("sup.supuesta", "2", "x", "P20D", "2020-01-01"),
	)}
	aplica := func(id string) (bool, bool) { return true, id == "sup.supuesta" }
	cal := Derivar12Meses(ps, aplica, ventana.Hechos{"x": ahoraDePrueba}, ahoraDePrueba)
	got := map[string]bool{}
	for _, m := range cal.Meses {
		for _, f := range m.Fechas {
			got[f.Obligacion] = f.Supuesta
		}
	}
	if got["sup.declarada"] {
		t.Error("una obligacion derivada de un hecho declarado sale marcada como supuesta")
	}
	if !got["sup.supuesta"] {
		t.Error("una obligacion derivada de un supuesto NO sale marcada. Es la diferencia entre " +
			"una fecha que el cliente puede defender y una que le hemos adivinado")
	}
}

// Las claves de catalogo que el calendario emite estan todas declaradas. Una
// clave que se emite y no esta en la lista no la traduce nadie, y sale el
// identificador en bruto en la pantalla de un cliente.
func TestTodaClaveQueElCalendarioEmiteEstaDeclarada(t *testing.T) {
	declaradas := map[string]bool{}
	for _, k := range ClavesDelCalendario() {
		declaradas[k] = true
	}
	if len(declaradas) != 15 {
		t.Fatalf("%d claves declaradas, esperaba 15 (tres motivos y doce meses)", len(declaradas))
	}
	// Se emiten de verdad: doce meses de fechas y los tres motivos.
	var obs []corpus.Obligacion
	for mes := 1; mes <= 12; mes++ {
		obs = append(obs, plazoDe("m."+time.Month(mes).String(), "1", "x",
			"P"+itoa(mes*30)+"D", "2020-01-01"))
	}
	ps := []*corpus.Paquete{paqueteConRelojes("urn:demo:k", obs...)}
	cal := Derivar12Meses(ps, TodoAplica, ventana.Hechos{"x": ahoraDePrueba}, ahoraDePrueba)
	for _, m := range cal.Meses {
		if !declaradas[m.Clave] {
			t.Errorf("el calendario emite la clave %q y no esta en ClavesDelCalendario", m.Clave)
		}
	}
	if len(cal.Meses) < 10 {
		t.Errorf("solo %d meses distintos: el caso no esta ejercitando el agrupador",
			len(cal.Meses))
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
