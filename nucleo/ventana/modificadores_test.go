package ventana

import (
	"testing"
	"time"
)

// Suspension del plazo por requerimiento de subsanacion (articulo 68 de la
// Ley 39/2015). El reloj no se reinicia: se para y se reanuda.
func TestSuspensionPorSubsanacion(t *testing.T) {
	cal := calES(t)
	reg := Regimen{Comp: Habiles, Cal: cal, Trasl: TrasladoSiguienteHabil, Fuente: "Ley 39/2015 art. 30.5"}
	p := Plazo{
		Disparador: "notificacion",
		Hitos:      []Hito{{ID: "alegaciones", Limite: Duracion{Dias: 10}, Reg: reg}},
	}
	h := Hechos{"notificacion": ts(t, "2026-12-04T09:00:00+01:00")}

	sin := p.Vencimientos(h, time.Time{})[0]
	if got := sin.Vence.Format("2006-01-02"); got != "2026-12-21" {
		t.Fatalf("sin suspension esperaba 2026-12-21, obtenido %s", got)
	}

	p.Mods = []Modificador{Suspension{
		Desde:  ts(t, "2026-12-09T00:00:00+01:00"),
		Hasta:  ts(t, "2026-12-16T00:00:00+01:00"),
		Motivo: "requerimiento de subsanacion, art. 68",
	}}
	con := p.Vencimientos(h, time.Time{})[0]
	if got := con.Vence.Format("2006-01-02"); got != "2026-12-29" {
		t.Fatalf("con suspension de 5 dias habiles esperaba 2026-12-29, obtenido %s (regla: %s)", got, con.Regla)
	}
}

// Prorroga de dos meses del articulo 12.3 del RGPD sobre el plazo de un mes.
func TestProrrogaRGPD(t *testing.T) {
	reg := Regimen{Comp: Naturales, Cal: calES(t), Cierre: CierreExacto}
	p := Plazo{
		Disparador: "solicitud",
		Hitos:      []Hito{{ID: "respuesta", Limite: Duracion{Meses: 1}, Reg: reg}},
		Mods:       []Modificador{Prorroga{D: Duracion{Meses: 2}, Motivo: "RGPD art. 12.3, complejidad"}},
	}
	h := Hechos{"solicitud": ts(t, "2026-03-31T12:00:00+02:00")}
	v := p.Vencimientos(h, time.Time{})[0]
	if got := v.Vence.Format(time.RFC3339); got != "2026-06-30T12:00:00+02:00" {
		t.Fatalf("esperado 2026-06-30T12:00:00+02:00, obtenido %s", got)
	}
}

// Articulo 30.6 de la Ley 39/2015: es inhabil el dia que lo sea en la localidad
// del interesado O en la sede del organo. Un calendario por ambito no basta.
func TestCalendariosCombinados(t *testing.T) {
	loc := mad(t)
	nacional := NuevoCalendario("es", "nacional", "BOE", loc, "2026-12-25")
	local := NuevoCalendario("sevilla", "local:41091", "BOJA", loc, "2026-12-28")
	comb := Combinar("efectivo", nacional, local)

	if comb.EsHabil(ts(t, "2026-12-28T10:00:00+01:00")) {
		t.Fatal("el 28 es festivo local y el calendario combinado tiene que verlo")
	}
	if nacional.EsHabil(ts(t, "2026-12-28T10:00:00+01:00")) == false {
		t.Fatal("el calendario nacional no deberia conocer el festivo local: el test estaria mal montado")
	}
	if comb.Fuente == "" || comb.Ambito != "combinado" {
		t.Fatal("la procedencia del calendario combinado tiene que quedar registrada")
	}
}

func TestParseDuracion(t *testing.T) {
	ok := map[string]Duracion{
		"PT24H":         {Horas: 24},
		"PT4H":          {Horas: 4},
		"P14D":          {Dias: 14},
		"P1M":           {Meses: 1},
		"P1MT12H":       {Meses: 1, Horas: 12},
		"indeterminado": {Indeterminado: true},
	}
	for s, want := range ok {
		got, err := ParseDuracion(s)
		if err != nil || got != want {
			t.Fatalf("%s: obtenido %+v err=%v", s, got, err)
		}
		if s != "indeterminado" && got.String() != s && s != "P1MT12H" {
			t.Fatalf("ida y vuelta rota: %s -> %s", s, got.String())
		}
	}
	for _, mal := range []string{"", "P", "24H", "1M", "PT", "P1Y"} {
		if _, err := ParseDuracion(mal); err == nil {
			t.Fatalf("%q deberia fallar", mal)
		}
	}
}

func TestIntervaloContiene(t *testing.T) {
	i := Intervalo{Desde: ts(t, "2026-01-01T00:00:00+01:00"), Hasta: ts(t, "2026-02-01T00:00:00+01:00")}
	abierto := Intervalo{Desde: i.Desde}
	if !i.Contiene(ts(t, "2026-01-15T00:00:00+01:00")) || i.Contiene(ts(t, "2026-02-01T00:00:00+01:00")) {
		t.Fatal("intervalo cerrado por la izquierda y abierto por la derecha")
	}
	if !abierto.Contiene(ts(t, "2030-01-01T00:00:00+01:00")) {
		t.Fatal("un intervalo sin fin contiene cualquier instante posterior")
	}
}
