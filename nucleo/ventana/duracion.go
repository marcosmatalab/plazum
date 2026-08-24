// Package ventana implementa la aritmetica temporal de obligaciones normativas.
//
// Reglas semanticas, fijadas aqui porque el derecho no las fija y una
// implementacion ingenua las confunde:
//
//	horas y minutos -> tiempo absoluto transcurrido. 24 h son 24 h reales
//	                   aunque haya cambio de hora por medio.
//	dias naturales  -> dias de calendario a la misma hora de pared, en la
//	                   zona del calendario. Un dia que cruza el cambio de
//	                   hora dura 23 o 25 horas reales, no 24.
//	dias habiles    -> se avanza dia a dia saltando sabados, domingos y
//	                   festivos del calendario aplicable.
//	meses           -> de fecha a fecha, con recorte al ultimo dia del mes
//	                   destino. 31 de enero + 1 mes = 28 o 29 de febrero.
//
// Orden de aplicacion cuando una duracion mezcla campos: meses, dias, horas.
// El orden importa y se fija aqui por escrito.
package ventana

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// Duracion es un subconjunto de ISO 8601 suficiente para plazos normativos.
type Duracion struct {
	Meses int
	Dias  int
	Horas int
	Mins  int
	// Indeterminado marca las obligaciones tipo "sin dilacion indebida":
	// no hay limite legal, solo un objetivo interno que fija la organizacion.
	Indeterminado bool
}

var reISO = regexp.MustCompile(`^P(?:(\d+)M)?(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?)?$`)

// ParseDuracion acepta P1M, P14D, PT24H, PT4H, PT72H, P1MT12H.
func ParseDuracion(s string) (Duracion, error) {
	if s == "indeterminado" {
		return Duracion{Indeterminado: true}, nil
	}
	m := reISO.FindStringSubmatch(s)
	if m == nil {
		return Duracion{}, fmt.Errorf("duracion no reconocida: %q", s)
	}
	// "P" y "PT" casan con el patron pero no son duraciones validas: sin ningun
	// componente no hay plazo. Lo destapo el test de ida y vuelta del parser.
	if m[1] == "" && m[2] == "" && m[3] == "" && m[4] == "" {
		return Duracion{}, fmt.Errorf("duracion sin componentes: %q", s)
	}
	n := func(x string) int {
		if x == "" {
			return 0
		}
		v, _ := strconv.Atoi(x)
		return v
	}
	return Duracion{Meses: n(m[1]), Dias: n(m[2]), Horas: n(m[3]), Mins: n(m[4])}, nil
}

func (d Duracion) String() string {
	if d.Indeterminado {
		return "indeterminado"
	}
	s := "P"
	if d.Meses > 0 {
		s += fmt.Sprintf("%dM", d.Meses)
	}
	if d.Dias > 0 {
		s += fmt.Sprintf("%dD", d.Dias)
	}
	if d.Horas > 0 || d.Mins > 0 {
		s += "T"
		if d.Horas > 0 {
			s += fmt.Sprintf("%dH", d.Horas)
		}
		if d.Mins > 0 {
			s += fmt.Sprintf("%dM", d.Mins)
		}
	}
	if s == "P" {
		return "P0D"
	}
	return s
}

func diasEnMes(y int, m time.Month) int {
	return time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// sumarMeses suma meses de fecha a fecha con recorte al ultimo dia del mes
// destino. Se implementa a mano porque time.AddDate normaliza por desbordamiento
// (31 de enero + 1 mes le da 3 de marzo), que es lo contrario de lo que dice
// el articulo 30.4 de la Ley 39/2015.
func sumarMeses(t time.Time, n int, loc *time.Location) time.Time {
	t = t.In(loc)
	y, m, d := t.Date()
	hh, mm, ss := t.Clock()
	total := (y*12 + int(m) - 1) + n
	ny, nm := total/12, time.Month(total%12+1)
	if last := diasEnMes(ny, nm); d > last {
		d = last
	}
	return time.Date(ny, nm, d, hh, mm, ss, 0, loc)
}
