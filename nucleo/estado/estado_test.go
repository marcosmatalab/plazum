package estado

import (
	"testing"
	"time"
)

func t0(s string) time.Time {
	v, _ := time.Parse(time.RFC3339, s)
	return v
}

func TestLosOchoEstados(t *testing.T) {
	ahora := t0("2026-08-23T12:00:00+02:00")
	p := Prueba{ID: "mfa.todos", Control: "op.acc.5", TTL: 24 * time.Hour, SLA: 72 * time.Hour}

	casos := []struct {
		nombre string
		obs    []Observacion
		ctx    Contexto
		quiero Estado
		por    string
	}{
		{"todo conforme",
			[]Observacion{{Prueba: "mfa.todos", Recurso: "u1", Satisfecho: true, Recolectada: ahora.Add(-time.Hour)}},
			Contexto{Ahora: ahora, Aplicable: true}, Pass, ""},
		{"falla pero dentro del plazo de remediacion",
			[]Observacion{{Prueba: "mfa.todos", Recurso: "u2", Satisfecho: false, Recolectada: ahora.Add(-time.Hour)}},
			Contexto{Ahora: ahora, Aplicable: true}, FailEnPlazo,
			"un fallo de una hora no puede tener el mismo tratamiento que uno de una semana"},
		// El caso que la revision destapo: con TTL 24 h y SLA 72 h, un fallo de
		// 100 h antes devolvia "obsoleto" y fail_vencido era inalcanzable.
		// Ahora el fallo con el plazo agotado manda sobre la obsolescencia.
		{"falla y el plazo se agoto, con la observacion ya caducada",
			[]Observacion{{Prueba: "mfa.todos", Recurso: "u2", Satisfecho: false, Recolectada: ahora.Add(-100 * time.Hour)}},
			Contexto{Ahora: ahora, Aplicable: true}, FailVencido, "esto si escala y esto si ve el auditor"},
		{"la observacion caduco",
			[]Observacion{{Prueba: "mfa.todos", Recurso: "u1", Satisfecho: true, Recolectada: ahora.Add(-48 * time.Hour)}},
			Contexto{Ahora: ahora, Aplicable: true}, Obsoleto,
			"no haberlo mirado hoy no es incumplir: confundirlo destruye la confianza en la herramienta"},
		{"el recolector no pudo leer",
			[]Observacion{{Prueba: "mfa.todos", Recurso: "u1", ErrorRecol: "429 rate limit", Recolectada: ahora}},
			Contexto{Ahora: ahora, Aplicable: true}, Error, "un 429 del proveedor no es un incumplimiento del cliente"},
		{"fuera de la declaracion de aplicabilidad",
			nil, Contexto{Ahora: ahora, Aplicable: false}, NoAplica,
			"lo decide el motor de aplicabilidad, no un boton de la interfaz"},
		{"excepcion sin aprobador NO se aplica",
			[]Observacion{{Prueba: "mfa.todos", Recurso: "u2", Satisfecho: false, Recolectada: ahora.Add(-time.Hour)}},
			Contexto{Ahora: ahora, Aplicable: true, Excepciones: []Excepcion{{
				Control: "op.acc.5", Motivo: "porque si", Desde: ahora.Add(-time.Hour), Hasta: ahora.Add(240 * time.Hour)}}},
			FailEnPlazo, "una excepcion invalida se ignora, no se aplica"},
		{"excepcion aprobada y vigente",
			[]Observacion{{Prueba: "mfa.todos", Recurso: "u2", Satisfecho: false, Recolectada: ahora}},
			Contexto{Ahora: ahora, Aplicable: true, Excepciones: []Excepcion{{
				Control: "op.acc.5", Motivo: "migracion de IdP", Aprobador: "CISO",
				Desde: ahora.Add(-24 * time.Hour), Hasta: ahora.Add(240 * time.Hour)}}}, Exceptuado, ""},
		{"prueba aun en despliegue",
			nil, Contexto{Ahora: ahora, Aplicable: true}, Manual,
			"una prueba nueva no puede generar hallazgos retroactivos"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			pp := p
			if c.nombre == "prueba aun en despliegue" {
				pp.Activa = ahora.Add(48 * time.Hour)
			}
			got := Calcular(pp, c.obs, c.ctx)
			if got.Estado != c.quiero {
				t.Fatalf("esperaba %s, obtuve %s (%s). %s", c.quiero, got.Estado, got.Motivo, c.por)
			}
		})
	}
}

func TestSoloLoVencidoEscalaAlAuditor(t *testing.T) {
	if FailEnPlazo.EscalaAlAuditor() {
		t.Fatal("un fallo dentro de SLA es trabajo normal, no un hallazgo de auditoria")
	}
	if !FailVencido.EscalaAlAuditor() || !Obsoleto.EscalaAlAuditor() {
		t.Fatal("lo vencido y lo obsoleto si tienen que verse")
	}
	// Corregido tras la revision: un error de recoleccion SI escala. Un 429 que
	// dura un ano significa que ese control lleva un ano sin comprobarse, y eso
	// el auditor tiene que verlo. Antes quedaba invisible.
	if !Error.EscalaAlAuditor() {
		t.Fatal("un error de recoleccion persistente tiene que verse: es un control sin comprobar")
	}
}

func TestExclusionPorRecursoNoApagaElControl(t *testing.T) {
	ahora := t0("2026-08-23T12:00:00+02:00")
	p := Prueba{ID: "cifrado.discos", Control: "mp.si.2", TTL: 48 * time.Hour, SLA: 72 * time.Hour}
	obs := []Observacion{
		{Prueba: "cifrado.discos", Recurso: "portatil-1", Satisfecho: true, Recolectada: ahora},
		{Prueba: "cifrado.discos", Recurso: "kiosco-recepcion", Satisfecho: false, Recolectada: ahora},
	}
	sin := Calcular(p, obs, Contexto{Ahora: ahora, Aplicable: true})
	if sin.Estado != FailEnPlazo {
		t.Fatalf("sin exclusion deberia fallar, obtuve %s", sin.Estado)
	}
	con := Calcular(p, obs, Contexto{Ahora: ahora, Aplicable: true, Exclusiones: []Exclusion{
		{Prueba: "cifrado.discos", Recurso: "kiosco-recepcion", Motivo: "terminal publica sin datos, aislada en VLAN"}}})
	if con.Estado != Pass {
		t.Fatalf("excluir un recurso no apaga el control entero, obtuve %s", con.Estado)
	}
	if len(con.Excluidos) != 1 {
		t.Fatal("la exclusion se declara al auditor con nombre y motivo")
	}
}

func TestLaExcepcionEternaSeRechaza(t *testing.T) {
	if err := (Excepcion{Control: "x", Motivo: "y", Aprobador: "CISO"}).Valida(); err == nil {
		t.Fatal("una excepcion sin fecha fin tiene que rechazarse en el modelo, no en la revision")
	}
	if err := (Excepcion{Control: "x", Motivo: "y", Desde: t0("2026-01-01T00:00:00Z"),
		Hasta: t0("2026-06-01T00:00:00Z")}).Valida(); err == nil {
		t.Fatal("una excepcion sin aprobador tiene que rechazarse")
	}
}

// Regresion del bug principal: con la configuracion normal (SLA mayor que TTL),
// fail_vencido tiene que ser alcanzable.
func TestFailVencidoEsAlcanzableConSLAMayorQueTTL(t *testing.T) {
	ahora := t0("2026-08-23T12:00:00+02:00")
	p := Prueba{ID: "x", Control: "c", TTL: 24 * time.Hour, SLA: 30 * 24 * time.Hour}
	casos := []struct {
		horas  int
		quiero Estado
	}{
		{1, FailEnPlazo},   // fallo reciente, observacion fresca
		{25, Obsoleto},     // fallo dentro de plazo pero la observacion caduco
		{800, FailVencido}, // plazo de remediacion agotado: esto escala
	}
	for _, c := range casos {
		obs := []Observacion{{Prueba: "x", Recurso: "r", Satisfecho: false,
			Recolectada: ahora.Add(-time.Duration(c.horas) * time.Hour)}}
		got := Calcular(p, obs, Contexto{Ahora: ahora, Aplicable: true})
		if got.Estado != c.quiero {
			t.Fatalf("fallo de %d h: esperaba %s, obtuve %s (%s)", c.horas, c.quiero, got.Estado, got.Motivo)
		}
	}
}

func TestPassPorDefectoNoTapaUnFalloObservado(t *testing.T) {
	ahora := t0("2026-08-23T12:00:00+02:00")
	p := Prueba{ID: "x", Control: "c", PassPorDef: true, TTL: 24 * time.Hour, SLA: 72 * time.Hour}
	obs := []Observacion{{Prueba: "x", Recurso: "r", Satisfecho: false, Recolectada: ahora}}
	if got := Calcular(p, obs, Contexto{Ahora: ahora, Aplicable: true}); got.Estado == Pass {
		t.Fatal("si el recolector ve un fallo, el fallo manda sobre lo que declare el proveedor")
	}
}
