package historia

import (
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/estado"
)

func ts(s string) time.Time { t, _ := time.Parse(time.RFC3339, s); return t }

func historiaDePrueba() *Historia {
	h := &Historia{}
	h.Registrar(CambioEstado{Prueba: "mfa", De: estado.Pass, A: estado.FailVencido,
		InstanteHecho: ts("2026-03-01T10:00:00Z"), InstanteRegistro: ts("2026-03-01T10:05:00Z"), Causa: "observacion"})
	h.Registrar(CambioEstado{Prueba: "mfa", De: estado.FailVencido, A: estado.Pass,
		InstanteHecho: ts("2026-03-03T10:00:00Z"), InstanteRegistro: ts("2026-03-03T10:01:00Z"), Causa: "observacion"})
	return h
}

func TestEstadoEnReconstruyeElPasado(t *testing.T) {
	h := historiaDePrueba()
	if e, ok := h.EstadoEn("mfa", ts("2026-03-02T00:00:00Z")); !ok || e != estado.FailVencido {
		t.Fatalf("el 2 de marzo estaba en fail_vencido: %v %v", e, ok)
	}
	if e, _ := h.EstadoEn("mfa", ts("2026-03-04T00:00:00Z")); e != estado.Pass {
		t.Fatalf("el 4 de marzo estaba en pass: %v", e)
	}
	if _, ok := h.EstadoEn("mfa", ts("2026-02-01T00:00:00Z")); ok {
		t.Fatal("antes del primer evento no hay estado")
	}
}

// La correccion del pasado deja rastro: InstanteHecho antiguo, registro nuevo.
// El eje hecho la incorpora; el eje registro conserva cuando se supo.
func TestCorregirElPasadoDejaRastro(t *testing.T) {
	h := historiaDePrueba()
	h.Registrar(CambioEstado{Prueba: "mfa", De: estado.Pass, A: estado.Obsoleto,
		InstanteHecho: ts("2026-03-02T12:00:00Z"), InstanteRegistro: ts("2026-03-10T09:00:00Z"), Causa: "correccion"})
	if e, _ := h.EstadoEn("mfa", ts("2026-03-02T13:00:00Z")); e != estado.Obsoleto {
		t.Fatalf("la correccion entra en el eje hecho: %v", e)
	}
	v := h.Ventana("mfa", ts("2026-03-01T00:00:00Z"), ts("2026-03-04T00:00:00Z"))
	if len(v) != 3 {
		t.Fatalf("la ventana ve los 3 eventos del mundo, vio %d", len(v))
	}
}

func TestPrimerConocimientoEsEjeRegistro(t *testing.T) {
	h := historiaDePrueba()
	got, ok := h.PrimerConocimiento("mfa", estado.FailVencido)
	if !ok || !got.Equal(ts("2026-03-01T10:05:00Z")) {
		t.Fatalf("el reloj del art. 33 arranca cuando se SUPO: %v %v", got, ok)
	}
}

func TestMTTR(t *testing.T) {
	h := historiaDePrueba()
	d, ok := h.MTTR("mfa")
	if !ok || d != 48*time.Hour {
		t.Fatalf("48h de remediacion: %v %v", d, ok)
	}
	if _, ok := h.MTTR("otra"); ok {
		t.Fatal("sin pares completos no hay MTTR")
	}
}
