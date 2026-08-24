package ventana

import (
	"testing"
	"time"
)

// Tabla dorada. Los valores esperados NO salen de este codigo: se calcularon con
// una implementacion de referencia independiente en Python (zoneinfo + datetime),
// que usa otra base de datos de husos y otra aritmetica. Si Go y Python
// discrepan, discrepan por un motivo, y el motivo es un fallo.

func mad(t *testing.T) *time.Location {
	t.Helper()
	l, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func calES(t *testing.T) *Calendario {
	return NuevoCalendario("es-2026", "nacional+MD", "BOE calendario laboral 2026", mad(t),
		"2026-01-01", "2026-01-06", "2026-04-03", "2026-05-01", "2026-08-15",
		"2026-10-12", "2026-11-02", "2026-12-08", "2026-12-25", "2026-05-02", "2026-11-09")
}

func ts(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return v.In(mad(t))
}

func TestTablaDorada(t *testing.T) {
	cal := calES(t)
	natSin := Regimen{Comp: Naturales, Cal: cal, Trasl: TrasladoNinguno}
	natCon := Regimen{Comp: Naturales, Cal: cal, Trasl: TrasladoSiguienteHabil, Fuente: "Rgto. 1182/71 art. 3.4"}
	habCon := Regimen{Comp: Habiles, Cal: cal, Trasl: TrasladoSiguienteHabil, Fuente: "Ley 39/2015 art. 30.5"}
	exacto := Regimen{Comp: Naturales, Cal: cal, Cierre: CierreExacto}

	casos := []struct {
		nombre   string
		base     string
		d        string
		reg      Regimen
		esperado string
		porque   string
	}{
		{"CRA art.14 alerta temprana 24 h", "2026-09-15T22:40:00+02:00", "PT24H", natSin,
			"2026-09-16T22:40:00+02:00", "horas = tiempo absoluto, sin traslado (Rgto. 1182/71 art. 3.4 solo alcanza a plazos no expresados en horas)"},
		{"CRA art.14 notificacion 72 h", "2026-09-15T22:40:00+02:00", "PT72H", natSin,
			"2026-09-18T22:40:00+02:00", "idem"},
		{"CRA art.14 informe final 1 mes, vence en domingo", "2026-09-18T10:00:00+02:00", "P1M", natCon,
			"2026-10-19T23:59:59+02:00", "18-10-2026 es domingo: se traslada al fin del lunes habil siguiente"},
		{"RGPD art.33 72 h, lectura EDPB", "2026-09-17T20:00:00+02:00", "PT72H", natSin,
			"2026-09-20T20:00:00+02:00", "el EDPB sostiene 72 horas exactas caiga donde caiga"},
		{"RGPD art.33 72 h, lectura Rgto. 1182/71", "2026-09-17T20:00:00+02:00", "PT72H", natCon,
			"2026-09-21T23:59:59+02:00", "doctrina que aplica el traslado: 1 dia y 3:59 h de diferencia"},
		{"AI Act art.73, 15 dias naturales", "2026-09-15T22:40:00+02:00", "P15D", natSin,
			"2026-09-30T23:59:59+02:00", "dias naturales: vence al final del dia 15, no a las 22:40 (Rgto. 1182/71 art. 3.2.b)"},
		{"1 mes desde el 31 de enero, sin traslado", "2026-01-31T09:00:00+01:00", "P1M", natSin,
			"2026-02-28T23:59:59+01:00", "recorte al ultimo dia del mes destino (no desbordamiento al 3 de marzo) y cierre al final del dia"},
		{"1 mes desde el 31 de enero, con traslado", "2026-01-31T09:00:00+01:00", "P1M", natCon,
			"2026-03-02T23:59:59+01:00", "28-02-2026 es sabado: traslado al lunes"},
		{"1 mes cruzando el cambio de hora", "2026-09-30T22:40:00+02:00", "P1M", natSin,
			"2026-10-30T23:59:59+01:00", "el dia se calcula conservando la hora de pared (por eso el desfase pasa de +02:00 a +01:00) y luego cierra al final del dia"},
		{"24 horas cruzando el cambio de hora", "2026-10-24T22:40:00+02:00", "PT24H", natSin,
			"2026-10-25T21:40:00+01:00", "24 horas ABSOLUTAS: el reloj de pared marca 21:40, no 22:40"},
		{"1 dia natural cruzando el cambio de hora, cierre exacto", "2026-10-24T22:40:00+02:00", "P1D", exacto,
			"2026-10-25T22:40:00+01:00", "con cierre exacto, 1 dia natural dura 25 horas reales ese dia: una hora mas que las 24 h del caso anterior"},
		{"1 dia natural cruzando el cambio de hora, cierre normal", "2026-10-24T22:40:00+02:00", "P1D", natSin,
			"2026-10-25T23:59:59+01:00", "con la regla legal, ni siquiera es la misma magnitud: vence al final del dia"},
		{"10 dias habiles con festivo de la Inmaculada", "2026-12-04T09:00:00+01:00", "P10D", habCon,
			"2026-12-21T23:59:59+01:00", "saltando sabados, domingos y el 8 de diciembre, con cierre al final del dia 21"},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			d, err := ParseDuracion(c.d)
			if err != nil {
				t.Fatal(err)
			}
			got, regla := Sumar(ts(t, c.base), d, c.reg)
			want := ts(t, c.esperado)
			if !got.Equal(want) {
				t.Fatalf("\n  esperado %s\n  obtenido %s\n  motivo   %s\n  regla    %s",
					want.Format(time.RFC3339), got.Format(time.RFC3339), c.porque, regla)
			}
		})
	}
}

// La diferencia entre "24 horas" y "1 dia" es la trampa que hace que una
// implementacion ingenua de plazos falle dos domingos al ano.
func TestVeinticuatroHorasNoEsUnDia(t *testing.T) {
	reg := Regimen{Comp: Naturales, Cal: calES(t), Cierre: CierreExacto}
	base := ts(t, "2026-10-24T22:40:00+02:00")
	h, _ := Sumar(base, Duracion{Horas: 24}, reg)
	d, _ := Sumar(base, Duracion{Dias: 1}, reg)
	if diff := d.Sub(h); diff != time.Hour {
		t.Fatalf("esperada 1 h de diferencia en el cambio de hora, obtenida %s", diff)
	}
}
