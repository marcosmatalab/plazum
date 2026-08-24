package ventana

import (
	"math/rand"
	"testing"
	"time"
)

// Propiedades metamorficas. Mas potentes que los casos sueltos porque no hay
// que saber la respuesta, solo la relacion entre respuestas.

func TestPropiedadHorasEsAditiva(t *testing.T) {
	reg := Regimen{Comp: Naturales, Cal: calES(t), Cierre: CierreExacto}
	r := rand.New(rand.NewSource(1182))
	inicio := ts(t, "2026-01-01T00:00:00+01:00")
	for i := 0; i < 5000; i++ {
		base := inicio.Add(time.Duration(r.Intn(365*24*60)) * time.Minute)
		a, b := 1+r.Intn(100), 1+r.Intn(100)
		x, _ := Sumar(base, Duracion{Horas: a}, reg)
		x, _ = Sumar(x, Duracion{Horas: b}, reg)
		y, _ := Sumar(base, Duracion{Horas: a + b}, reg)
		if !x.Equal(y) {
			t.Fatalf("horas no aditivas: base %s, %d+%d", base, a, b)
		}
	}
}

func TestPropiedadHabilesEsAditivaYCaeEnHabil(t *testing.T) {
	cal := calES(t)
	reg := Regimen{Comp: Habiles, Cal: cal, Cierre: CierreExacto}
	r := rand.New(rand.NewSource(3915))
	inicio := ts(t, "2026-01-01T00:00:00+01:00")
	for i := 0; i < 5000; i++ {
		base := inicio.Add(time.Duration(r.Intn(365*24*60)) * time.Minute)
		a, b := 1+r.Intn(20), 1+r.Intn(20)
		x, _ := Sumar(base, Duracion{Dias: a}, reg)
		if !cal.EsHabil(x) {
			t.Fatalf("un plazo en dias habiles vencio en inhabil: %s", x)
		}
		x, _ = Sumar(x, Duracion{Dias: b}, reg)
		y, _ := Sumar(base, Duracion{Dias: a + b}, reg)
		if !x.Equal(y) {
			t.Fatalf("habiles no aditivos: base %s, %d+%d", base, a, b)
		}
	}
}

func TestPropiedadMonotonia(t *testing.T) {
	reg := Regimen{Comp: Habiles, Cal: calES(t), Trasl: TrasladoSiguienteHabil}
	r := rand.New(rand.NewSource(30052015))
	inicio := ts(t, "2026-01-01T00:00:00+01:00")
	for i := 0; i < 5000; i++ {
		b1 := inicio.Add(time.Duration(r.Intn(365*24*60)) * time.Minute)
		b2 := b1.Add(time.Duration(1+r.Intn(60*24)) * time.Minute)
		n := 1 + r.Intn(30)
		x, _ := Sumar(b1, Duracion{Dias: n}, reg)
		y, _ := Sumar(b2, Duracion{Dias: n}, reg)
		if y.Before(x) {
			t.Fatalf("monotonia rota: base posterior vence antes\n  %s -> %s\n  %s -> %s", b1, x, b2, y)
		}
		if !x.Equal(y) && y.Sub(x) < 0 {
			t.Fatal("invariante de orden roto")
		}
	}
}

// HALLAZGO. La suma de meses NO es asociativa, y el derecho no lo resuelve.
// 31 de enero + 1 mes + 1 mes = 28 de marzo (recorte a febrero y arrastre).
// 31 de enero + 2 meses      = 31 de marzo.
// Tres dias de diferencia en un plazo de sancion. El motor fija por escrito
// que un plazo se calcula SIEMPRE desde su base original en un solo paso, y
// nunca acumulando tramos, y este test lo deja clavado.
func TestLaSumaDeMesesNoEsAsociativa(t *testing.T) {
	reg := Regimen{Comp: Naturales, Cal: calES(t), Cierre: CierreExacto}
	base := ts(t, "2026-01-31T09:00:00+01:00")
	uno, _ := Sumar(base, Duracion{Meses: 1}, reg)
	dosPasos, _ := Sumar(uno, Duracion{Meses: 1}, reg)
	unPaso, _ := Sumar(base, Duracion{Meses: 2}, reg)

	if dosPasos.Equal(unPaso) {
		t.Fatal("si esto empieza a pasar, alguien ha cambiado la semantica de recorte sin decirlo")
	}
	if got := dosPasos.Format("2006-01-02"); got != "2026-03-28" {
		t.Fatalf("acumulando tramos esperaba 2026-03-28, obtenido %s", got)
	}
	if got := unPaso.Format("2006-01-02"); got != "2026-03-31" {
		t.Fatalf("en un solo paso esperaba 2026-03-31, obtenido %s", got)
	}
	t.Logf("divergencia documentada: %s frente a %s (%v)",
		dosPasos.Format("2006-01-02"), unPaso.Format("2006-01-02"), unPaso.Sub(dosPasos))
}
