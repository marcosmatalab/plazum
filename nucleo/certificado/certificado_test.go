package certificado

// Los tres dorados del diseno: ISO trienal con vigilancias anuales, ENS con la
// bienal del art. 31 mas INES anual, y SOC 2 con ventanas de observacion
// solapadas. Fechas derivadas de la regla, no de la implementacion.

import (
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

func ts(s string) time.Time { t, _ := time.Parse(time.RFC3339, s); return t }

func regNat(t *testing.T) ventana.Regimen {
	t.Helper()
	cal := ventana.NuevoCalendario("utc", "test", "test", time.UTC)
	return ventana.Regimen{Comp: ventana.Naturales, Cal: cal, Cierre: ventana.CierreFinDia}
}

func TestDoradoISOTrienalConVigilancias(t *testing.T) {
	reg := regNat(t)
	c := Certificado{
		ID: "iso-27001-2025", Marco: "iso27001", Emision: ts("2025-06-15T00:00:00Z"),
		Hitos: []HitoCert{
			{Tipo: "vigilancia", Cita: "ISO/IEC 17021-1: vigilancia al menos anual",
				Primitiva: ventana.Periodica{Hito: "vigilancia", Desde: ts("2025-06-15T00:00:00Z"),
					Cada: ventana.Duracion{Meses: 12}, Reg: reg},
				Genera: []string{"preparar_vigilancia"}},
			{Tipo: "recertificacion", Cita: "ciclo de certificacion de 3 anos",
				Primitiva: ventana.Puntual{Hito: "recertificacion", En: ts("2028-06-14T23:59:59Z")}},
		},
	}
	vs := c.Vencimientos(nil, ts("2028-12-31T00:00:00Z"))
	// 3 vigilancias caben antes de la recertificacion (2026, 2027, 2028) + la puntual
	if len(vs) != 4 {
		t.Fatalf("esperaba 4 vencimientos, hay %d", len(vs))
	}
	if !vs[0].Vence.Equal(ts("2026-06-15T23:59:59Z")) {
		t.Fatalf("la primera vigilancia vence el 15-06-2026 fin de dia, no %s", vs[0].Vence)
	}
	if vs[0].Genera[0] != "preparar_vigilancia" {
		t.Fatalf("el hito dispara su obligacion interna: %v", vs[0].Genera)
	}
	p, err := c.Proximo(nil, ts("2027-01-01T00:00:00Z"), ts("2028-12-31T00:00:00Z"))
	if err != nil || !p.Vence.Equal(ts("2027-06-15T23:59:59Z")) {
		t.Fatalf("proximo desde 2027: %v %s", err, p.Vence)
	}
}

func TestDoradoENSBienalMasINESAnual(t *testing.T) {
	reg := regNat(t)
	c := Certificado{
		ID: "ens-media-2025", Marco: "ens", Emision: ts("2025-03-10T00:00:00Z"),
		Hitos: []HitoCert{
			{Tipo: "vigilancia", Cita: "RD 311/2022 art. 31: auditoria al menos cada dos anos",
				Primitiva: ventana.Periodica{Hito: "auditoria", Desde: ts("2025-03-10T00:00:00Z"),
					Cada: ventana.Duracion{Meses: 24}, Reg: reg}},
			{Tipo: "informe", Cita: "BOE-A-2016-10108: INES al menos con caracter anual",
				Primitiva: ventana.Periodica{Hito: "ines", Desde: ts("2025-01-01T00:00:00Z"),
					Cada: ventana.Duracion{Meses: 12}, Reg: reg}},
		},
	}
	vs := c.Vencimientos(nil, ts("2027-06-30T00:00:00Z"))
	// INES 2026-01-01 y 2027-01-01, auditoria 2027-03-10: tres relojes, ordenados
	if len(vs) != 3 {
		t.Fatalf("esperaba 3, hay %d", len(vs))
	}
	if vs[2].Tipo != "vigilancia" || !vs[2].Vence.Equal(ts("2027-03-10T23:59:59Z")) {
		t.Fatalf("la bienal del art. 31 vence el 10-03-2027 fin de dia: %s %s", vs[2].Tipo, vs[2].Vence)
	}
}

func TestDoradoSOC2VentanasSolapadas(t *testing.T) {
	c := Certificado{
		ID: "soc2-tipo2", Marco: "soc2", Emision: ts("2025-01-01T00:00:00Z"),
		Hitos: []HitoCert{
			{Tipo: "ventana_observacion", Cita: "periodo de observacion 01-01 a 31-12-2026",
				Primitiva: ventana.Continua{Hito: "ventana-2026",
					I: ventana.Intervalo{Desde: ts("2026-01-01T00:00:00Z"), Hasta: ts("2026-12-31T23:59:59Z")}}},
			{Tipo: "ventana_observacion", Cita: "periodo siguiente solapado 01-07-2026 a 30-06-2027",
				Primitiva: ventana.Continua{Hito: "ventana-2027",
					I: ventana.Intervalo{Desde: ts("2026-07-01T00:00:00Z"), Hasta: ts("2027-06-30T23:59:59Z")}}},
		},
	}
	vs := c.Vencimientos(nil, ts("2027-12-31T00:00:00Z"))
	if len(vs) != 2 {
		t.Fatalf("dos ventanas, hay %d", len(vs))
	}
	// el solape: el 1-10-2026 las dos ventanas estan abiertas
	en := ts("2026-10-01T00:00:00Z")
	abiertas := 0
	for _, h := range c.Hitos {
		if cont, ok := h.Primitiva.(ventana.Continua); ok && cont.I.Contiene(en) {
			abiertas++
		}
	}
	if abiertas != 2 {
		t.Fatalf("el 01-10-2026 hay 2 ventanas abiertas, hay %d", abiertas)
	}
}

func TestSuspendidoNoGeneraRelojes(t *testing.T) {
	c := Certificado{ID: "x", Estado: Suspendido,
		Hitos: []HitoCert{{Tipo: "vigilancia", Primitiva: ventana.Puntual{Hito: "v", En: ts("2027-01-01T00:00:00Z")}}}}
	if vs := c.Vencimientos(nil, ts("2028-01-01T00:00:00Z")); vs != nil {
		t.Fatalf("suspendido no genera relojes: %v", vs)
	}
}
