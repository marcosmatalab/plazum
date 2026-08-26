package ventana

import (
	"testing"
	"time"
)

// Las dos primitivas que encontro el censo, con sus dorados.
//
// LOS VALORES ESPERADOS NO SALEN DE ESTE CODIGO. Se calcularon con una
// implementacion de referencia independiente en Python (zoneinfo + datetime),
// que usa otra base de datos de husos y otra aritmetica, igual que la tabla
// dorada de dorado_test.go. Si Go y Python discrepan, discrepan por un motivo,
// y el motivo es un fallo.
//
// Y no hay ni un identificador de norma en este fichero, a proposito: las dos
// primitivas son del MOTOR, no de ninguna norma concreta. Quien las use es un
// paquete de corpus.

// ---------------------------------------------------------------------------
// Restar: la aritmetica hacia atras
// ---------------------------------------------------------------------------

func TestDoradoRestar(t *testing.T) {
	cal := calES(t)
	natSin := Regimen{Comp: Naturales, Cal: cal, Trasl: TrasladoNinguno}
	natCon := Regimen{Comp: Naturales, Cal: cal, Trasl: TrasladoSiguienteHabil, Fuente: "Rgto. 1182/71 art. 3.4"}
	habSin := Regimen{Comp: Habiles, Cal: cal, Trasl: TrasladoNinguno}

	casos := []struct {
		nombre   string
		efecto   string
		d        string
		reg      Regimen
		esperado string
		porque   string
	}{
		{
			"dos meses de antelacion, naturales",
			"2026-06-01T00:00:00+02:00", "P2M", natSin,
			"2026-04-01T00:00:00+02:00",
			"de fecha a fecha hacia atras; conserva la hora de pared",
		},
		{
			"un mes hacia atras desde un 31, con recorte al ultimo dia",
			"2026-03-31T12:00:00+02:00", "P1M", natSin,
			"2026-02-28T12:00:00+01:00",
			"febrero de 2026 no tiene 31: se recorta al 28. Y CRUZA EL CAMBIO DE HORA, " +
				"asi que el desplazamiento pasa de +02:00 a +01:00: la hora de pared se " +
				"conserva y el instante absoluto no, que es lo correcto para un plazo civil",
		},
		{
			"cinco dias habiles hacia atras cruzando dos festivos",
			"2026-01-12T10:00:00+01:00", "P5D", habSin,
			"2026-01-02T10:00:00+01:00",
			"desde el lunes 12 hacia atras: 9, 8 y 7 (habiles), el 6 es festivo, el 5 " +
				"(habil), el 3 y el 4 son fin de semana, y el quinto habil es el viernes 2",
		},
		{
			"el ultimo dia de aviso cae en inhabil y se ADELANTA, no se retrasa",
			"2026-05-04T09:00:00+02:00", "P1D", natCon,
			"2026-04-30T09:00:00+02:00",
			"un dia natural atras da el domingo 3; hacia atras el 2 y el 1 tambien son " +
				"festivos, asi que el ultimo dia util de aviso es el jueves 30. Retrasarlo al " +
				"siguiente habil habria dado el 4, que es el dia del efecto: cero antelacion",
		},
		{
			"noventa dias naturales de antelacion",
			"2026-09-01T00:00:00+02:00", "P90D", natSin,
			"2026-06-03T00:00:00+02:00",
			"noventa dias naturales exactos, sin mirar el calendario",
		},
		{
			"treinta y ocho dias naturales de espera previa",
			"2026-07-01T00:00:00+02:00", "P38D", natSin,
			"2026-05-24T00:00:00+02:00",
			"la forma de la espera obligatoria antes de aplicar una modificacion",
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			d, err := ParseDuracion(c.d)
			if err != nil {
				t.Fatal(err)
			}
			got, regla := Restar(ts(t, c.efecto), d, c.reg)
			quiero := ts(t, c.esperado)
			if !got.Equal(quiero) {
				t.Fatalf("Restar(%s, %s)\n  esperado %s\n  obtenido %s\n  porque: %s\n  derivacion: %s",
					c.efecto, c.d, quiero.Format(time.RFC3339), got.Format(time.RFC3339), c.porque, regla)
			}
			if regla == "" {
				t.Error("sin derivacion no hay `plazum explain` que valga: el operador no " +
					"puede comprobar de donde sale la fecha")
			}
		})
	}
}

// LA PROPIEDAD QUE JUSTIFICA QUE Restar EXISTA, y no es un detalle.
//
// Si el traslado hacia atras fuera al SIGUIENTE habil (copiando Sumar), la fecha
// limite de aviso se moveria HACIA el dia del efecto y la antelacion quedaria
// por debajo del minimo legal. Aqui se mide exactamente eso: el resultado nunca
// puede estar despues del resultado sin traslado.
func TestElTrasladoHaciaAtrasNuncaAcortaLaAntelacion(t *testing.T) {
	cal := calES(t)
	sin := Regimen{Comp: Naturales, Cal: cal, Trasl: TrasladoNinguno}
	con := Regimen{Comp: Naturales, Cal: cal, Trasl: TrasladoSiguienteHabil, Fuente: "prueba"}
	d, err := ParseDuracion("P1D")
	if err != nil {
		t.Fatal(err)
	}

	// Un ano entero de fechas de efecto, para no elegir la que conviene.
	inicio := ts(t, "2026-01-01T09:00:00+01:00")
	movidos := 0
	for i := 0; i < 365; i++ {
		efecto := inicio.AddDate(0, 0, i)
		liso, _ := Restar(efecto, d, sin)
		trasladado, _ := Restar(efecto, d, con)
		if trasladado.After(liso) {
			t.Fatalf("HALLAZGO: con efecto el %s, el traslado ha movido el ultimo dia de "+
				"aviso HACIA ADELANTE (%s -> %s). Eso acorta la antelacion por debajo del "+
				"minimo legal, que es exactamente lo que un preaviso no puede hacer",
				efecto.Format("2006-01-02"), liso.Format(time.RFC3339), trasladado.Format(time.RFC3339))
		}
		if trasladado.Before(liso) {
			movidos++
		}
		if !cal.EsHabil(trasladado) {
			t.Fatalf("con efecto el %s el resultado con traslado cae en inhabil (%s), o sea "+
				"que el traslado no ha hecho su trabajo",
				efecto.Format("2006-01-02"), trasladado.Format("2006-01-02"))
		}
	}
	// CONTROL NEGATIVO: si NUNCA se moviera ninguno, este test estaria pasando
	// sin ejercitar el traslado ni una vez.
	if movidos == 0 {
		t.Fatal("en 365 fechas de efecto el traslado no se ha disparado ni una vez: este " +
			"test no esta midiendo el traslado, esta midiendo que Restar devuelve algo")
	}
	t.Logf("PROPIEDAD FIJADA: en 365 fechas, %d se adelantaron y ninguna se retraso", movidos)
}

// ---------------------------------------------------------------------------
// Maximo: el mas tardio de dos duraciones
// ---------------------------------------------------------------------------

func TestDoradoMaximo(t *testing.T) {
	cal := calES(t)
	reg := Regimen{Comp: Naturales, Cal: cal, Trasl: TrasladoNinguno}
	diez, err := ParseDuracion("P120M")
	if err != nil {
		t.Fatal(err)
	}
	const disparo = "puesta.en.mercado"
	const declarado = "periodo.soporte.fin"
	base := ts(t, "2026-09-11T00:00:00+02:00")

	// El suelo, calculado aparte con la referencia de Python: 120 meses de fecha
	// a fecha desde el 11-09-2026, y el cierre de fin de dia porque el plazo se
	// expresa en meses.
	const sueloEsperado = "2036-09-11T23:59:59+02:00"

	m := Maximo{Hito: "retencion", Disparador: disparo, Suelo: diez, Reg: reg,
		Ampliacion: declarado, Exigible: true}

	t.Run("sin disparador el reloj no ha arrancado", func(t *testing.T) {
		v := m.Vencimientos(Hechos{}, time.Time{})
		if len(v) != 1 || v[0].Estado != PendienteDeHecho {
			t.Fatalf("esperaba pendiente de hecho: %+v", v)
		}
	})

	t.Run("declarada mas larga: gana la declarada", func(t *testing.T) {
		fin := ts(t, "2039-01-01T00:00:00+01:00")
		v := m.Vencimientos(Hechos{disparo: base, declarado: fin}, time.Time{})
		if len(v) != 1 || v[0].Estado != Determinado {
			t.Fatalf("esperaba una fecha cerrada: %+v", v)
		}
		if !v[0].Vence.Equal(fin) {
			t.Fatalf("la ampliacion declarada es posterior al suelo y tenia que ganar.\n"+
				"  esperado %s\n  obtenido %s\n  derivacion: %s",
				fin.Format(time.RFC3339), v[0].Vence.Format(time.RFC3339), v[0].Regla)
		}
	})

	// EL CASO QUE JUSTIFICA LA PRIMITIVA. Una declaracion del obligado mas corta
	// que el suelo NO reduce el suelo. Implementar esto como "usa la declarada si
	// la hay" da una fecha mas corta que la legal, en el sentido peligroso.
	t.Run("declarada mas corta: gana el suelo legal", func(t *testing.T) {
		fin := ts(t, "2030-01-01T00:00:00+01:00")
		v := m.Vencimientos(Hechos{disparo: base, declarado: fin}, time.Time{})
		if len(v) != 1 || v[0].Estado != Determinado {
			t.Fatalf("esperaba una fecha cerrada: %+v", v)
		}
		if !v[0].Vence.Equal(ts(t, sueloEsperado)) {
			t.Fatalf("HALLAZGO: una declaracion del obligado ha ACORTADO el minimo legal.\n"+
				"  esperado %s (el suelo)\n  obtenido %s\n  derivacion: %s",
				sueloEsperado, v[0].Vence.Format(time.RFC3339), v[0].Regla)
		}
	})

	// El estado que se olvida: la norma exige declarar y no se ha declarado. La
	// fecha final no se sabe; el suelo si.
	t.Run("exigible y sin declarar: pendiente, pero con suelo", func(t *testing.T) {
		v := m.Vencimientos(Hechos{disparo: base}, time.Time{})
		if len(v) != 1 {
			t.Fatalf("esperaba un vencimiento: %+v", v)
		}
		if v[0].Estado != PendienteDeHecho {
			t.Fatalf("HALLAZGO: la fecha final no se sabe y se presenta como %s. Un verde "+
				"mas debil que se lee igual que uno fuerte es lo que este proyecto no hace.\n"+
				"  derivacion: %s", v[0].Estado, v[0].Regla)
		}
		if !v[0].Vence.IsZero() {
			t.Errorf("con estado pendiente, Vence tiene que quedarse a cero o alguien lo "+
				"pintara como si fuera la respuesta: %s", v[0].Vence.Format(time.RFC3339))
		}
		if !v[0].NoAntesDe.Equal(ts(t, sueloEsperado)) {
			t.Fatalf("el suelo legal es un dato cierto y se ha perdido.\n"+
				"  esperado %s\n  obtenido %s", sueloEsperado, v[0].NoAntesDe.Format(time.RFC3339))
		}
	})

	// Y la otra mitad: si la norma NO obliga a declarar, la ausencia no es una
	// incognita, es que rige el suelo.
	t.Run("no exigible y sin declarar: rige el suelo", func(t *testing.T) {
		n := m
		n.Exigible = false
		v := n.Vencimientos(Hechos{disparo: base}, time.Time{})
		if len(v) != 1 || v[0].Estado != Determinado {
			t.Fatalf("esperaba una fecha cerrada: %+v", v)
		}
		if !v[0].Vence.Equal(ts(t, sueloEsperado)) {
			t.Fatalf("esperado %s, obtenido %s", sueloEsperado, v[0].Vence.Format(time.RFC3339))
		}
	})

	t.Run("sin segunda rama se comporta como un plazo normal", func(t *testing.T) {
		n := m
		n.Ampliacion = ""
		v := n.Vencimientos(Hechos{disparo: base}, time.Time{})
		if len(v) != 1 || v[0].Estado != Determinado || !v[0].Vence.Equal(ts(t, sueloEsperado)) {
			t.Fatalf("esperaba el suelo y salio %+v", v)
		}
	})
}

// La derivacion tiene que decir CUAL de las dos ramas ha ganado. Sin eso, un
// auditor ve una fecha y no puede saber si sale de la ley o de una declaracion
// del propio obligado, que es justo la diferencia que le importa.
func TestElMaximoDiceQueRamaGana(t *testing.T) {
	cal := calES(t)
	reg := Regimen{Comp: Naturales, Cal: cal}
	diez, err := ParseDuracion("P120M")
	if err != nil {
		t.Fatal(err)
	}
	m := Maximo{Hito: "retencion", Disparador: "d", Suelo: diez, Reg: reg,
		Ampliacion: "a", Exigible: true}
	base := ts(t, "2026-09-11T00:00:00+02:00")

	larga := m.Vencimientos(Hechos{"d": base, "a": ts(t, "2039-01-01T00:00:00+01:00")}, time.Time{})
	corta := m.Vencimientos(Hechos{"d": base, "a": ts(t, "2030-01-01T00:00:00+01:00")}, time.Time{})

	if larga[0].Regla == corta[0].Regla {
		t.Fatal("las dos ramas producen la misma derivacion: no hay forma de saber cual gano")
	}
	if !contiene(larga[0].Regla, "GANA LA AMPLIACION") {
		t.Errorf("cuando gana la ampliacion no se dice: %s", larga[0].Regla)
	}
	if !contiene(corta[0].Regla, "GANA EL SUELO") {
		t.Errorf("cuando gana el suelo no se dice: %s", corta[0].Regla)
	}
}

func contiene(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

// ---------------------------------------------------------------------------
// Preaviso: el plazo que corre al reves
// ---------------------------------------------------------------------------

func TestDoradoPreaviso(t *testing.T) {
	cal := calES(t)
	reg := Regimen{Comp: Naturales, Cal: cal, Trasl: TrasladoNinguno}
	dos, err := ParseDuracion("P2M")
	if err != nil {
		t.Fatal(err)
	}
	p := Preaviso{Hito: "aviso.modificacion", Efecto: "modificacion.efecto", Antelacion: dos, Reg: reg}

	t.Run("sin fecha de efecto no hay nada que preavisar", func(t *testing.T) {
		v := p.Vencimientos(Hechos{}, time.Time{})
		if len(v) != 1 || v[0].Estado != PendienteDeHecho {
			t.Fatalf("esperaba pendiente de hecho: %+v", v)
		}
	})

	t.Run("dos meses de antelacion", func(t *testing.T) {
		v := p.Vencimientos(Hechos{"modificacion.efecto": ts(t, "2026-06-01T00:00:00+02:00")}, time.Time{})
		quiero := ts(t, "2026-04-01T00:00:00+02:00")
		if len(v) != 1 || v[0].Estado != Determinado || !v[0].Vence.Equal(quiero) {
			t.Fatalf("esperado %s y salio %+v", quiero.Format(time.RFC3339), v)
		}
	})

	// LA PROPIEDAD QUE DISTINGUE ESTA FAMILIA DE TODAS LAS DEMAS: el vencimiento
	// SE MUEVE cuando se mueve el hito, porque el hito lo elige el obligado.
	// Adelantar la fecha de efecto adelanta la fecha limite de aviso, y puede
	// dejarla en el pasado, que es la situacion que hay que ensenar y no
	// esconder.
	t.Run("adelantar el efecto adelanta el limite de aviso", func(t *testing.T) {
		tarde := p.Vencimientos(Hechos{"modificacion.efecto": ts(t, "2026-06-01T00:00:00+02:00")}, time.Time{})
		pronto := p.Vencimientos(Hechos{"modificacion.efecto": ts(t, "2026-05-01T00:00:00+02:00")}, time.Time{})
		if !pronto[0].Vence.Before(tarde[0].Vence) {
			t.Fatalf("HALLAZGO: adelantar un mes la fecha de efecto no ha adelantado la fecha "+
				"limite de aviso (%s vs %s). Entonces esto no es un preaviso, es un plazo "+
				"normal con otro nombre",
				pronto[0].Vence.Format(time.RFC3339), tarde[0].Vence.Format(time.RFC3339))
		}
		delta := tarde[0].Vence.Sub(pronto[0].Vence)
		if delta < 28*24*time.Hour || delta > 31*24*time.Hour {
			t.Fatalf("un mes de diferencia en el efecto tenia que dar un mes en el aviso y "+
				"ha dado %v", delta)
		}
	})

	// Y el caso incomodo, que es el util: un efecto tan cercano que el limite de
	// aviso ya ha pasado. El motor NO lo esconde ni lo empuja a hoy.
	t.Run("un efecto demasiado cercano deja el limite en el pasado", func(t *testing.T) {
		efecto := ts(t, "2026-03-01T00:00:00+01:00")
		v := p.Vencimientos(Hechos{"modificacion.efecto": efecto}, time.Time{})
		if v[0].Estado != Determinado {
			t.Fatalf("esperaba una fecha cerrada: %+v", v)
		}
		if !v[0].Vence.Before(efecto) {
			t.Fatal("el limite de aviso tiene que quedar ANTES del efecto")
		}
		// 2026-01-01, que es lo que dio la referencia independiente.
		if !v[0].Vence.Equal(ts(t, "2026-01-01T00:00:00+01:00")) {
			t.Fatalf("esperado 2026-01-01T00:00:00+01:00 y salio %s", v[0].Vence.Format(time.RFC3339))
		}
	})
}

// Las dos primitivas nuevas cumplen el interfaz, que es lo que las hace usables
// desde un paquete de corpus sin tocar codigo.
func TestLasDosPrimitivasNuevasSonPrimitivas(t *testing.T) {
	var _ Primitiva = Maximo{}
	var _ Primitiva = Preaviso{}
	// Los parentesis no son estilo: en la condicion de un `if`, el parser lee
	// `Maximo{` como el principio del bloque.
	if (Maximo{}).Nombre() == (Preaviso{}).Nombre() {
		t.Fatal("dos primitivas con el mismo nombre: un paquete no podria elegir cual quiere")
	}
	for _, n := range []string{(Maximo{}).Nombre(), (Preaviso{}).Nombre()} {
		if n == "" {
			t.Fatal("una primitiva sin nombre no se puede declarar desde un paquete")
		}
	}
}
