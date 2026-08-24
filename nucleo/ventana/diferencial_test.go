package ventana

import (
	"math/rand"
	"testing"
	"time"
)

// Testing diferencial: la misma pregunta contra una implementacion ingenua y
// obviamente correcta, escrita aparte y a proposito lenta. Semilla fija: el
// fallo se reproduce siempre.

func ingenuoHabiles(base time.Time, n int, cal *Calendario) time.Time {
	t := base.In(cal.Zona)
	for i := 0; i < n; i++ {
		t = t.AddDate(0, 0, 1)
		for !cal.EsHabil(t) {
			t = t.AddDate(0, 0, 1)
		}
	}
	return t
}

func ingenuoMeses(base time.Time, n int, loc *time.Location) time.Time {
	t := base.In(loc)
	y, m, diaOriginal := t.Date()
	hh, mm, ss := t.Clock()
	for i := 0; i < n; i++ {
		if m == time.December {
			y, m = y+1, time.January
		} else {
			m++
		}
	}
	d := diaOriginal
	if last := diasEnMes(y, m); d > last {
		d = last
	}
	return time.Date(y, m, d, hh, mm, ss, 0, loc)
}

func TestDiferencialHabiles(t *testing.T) {
	cal := calES(t)
	reg := Regimen{Comp: Habiles, Cal: cal, Cierre: CierreExacto}
	r := rand.New(rand.NewSource(20260823))
	inicio := ts(t, "2026-01-01T00:00:00+01:00")
	for i := 0; i < 20000; i++ {
		base := inicio.Add(time.Duration(r.Intn(365*24*60)) * time.Minute)
		n := 1 + r.Intn(40)
		got, regla := Sumar(base, Duracion{Dias: n}, reg)
		want := ingenuoHabiles(base, n, cal)
		if !got.Equal(want) {
			t.Fatalf("caso %d: base %s + %d habiles\n  motor   %s\n  ingenuo %s\n  regla   %s",
				i, base.Format(time.RFC3339), n, got.Format(time.RFC3339), want.Format(time.RFC3339), regla)
		}
	}
}

func TestDiferencialMeses(t *testing.T) {
	loc := mad(t)
	reg := Regimen{Comp: Naturales, Cal: calES(t), Cierre: CierreExacto}
	r := rand.New(rand.NewSource(20260823))
	inicio := ts(t, "2024-01-01T00:00:00+01:00")
	for i := 0; i < 20000; i++ {
		base := inicio.Add(time.Duration(r.Intn(4*365*24*60)) * time.Minute)
		n := 1 + r.Intn(36)
		got, regla := Sumar(base, Duracion{Meses: n}, reg)
		want := ingenuoMeses(base, n, loc)
		if !got.Equal(want) {
			t.Fatalf("caso %d: base %s + %d meses\n  motor   %s\n  ingenuo %s\n  regla   %s",
				i, base.Format(time.RFC3339), n, got.Format(time.RFC3339), want.Format(time.RFC3339), regla)
		}
	}
}
