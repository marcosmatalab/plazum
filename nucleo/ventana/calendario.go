package ventana

import (
	"fmt"
	"sort"
	"time"
)

// Computo distingue como se cuentan los dias.
type Computo uint8

const (
	Naturales Computo = iota
	Habiles
)

func (c Computo) String() string {
	if c == Habiles {
		return "habiles"
	}
	return "naturales"
}

// Traslado modela la regla de cierre. Es un eje INDEPENDIENTE del computo, y
// esa separacion es un hallazgo de implementacion, no una decision de diseno
// previa: el articulo 3.4 del Reglamento 1182/71 y el articulo 30.5 de la Ley
// 39/2015 trasladan al siguiente dia habil el vencimiento que cae en inhabil,
// y lo hacen para plazos expresados en dias o meses, no para los expresados en
// horas. Si el traslado se deduce del computo, como estaba en el primer boceto,
// nunca se dispara y el motor da fechas mal.
type Traslado uint8

const (
	TrasladoNinguno Traslado = iota
	TrasladoSiguienteHabil
)

// Cierre decide si el plazo vence en un instante exacto o al final del dia.
//
// HALLAZGO DE IMPLEMENTACION. El test de monotonia lo destapo en la primera
// hora: si un plazo en dias conserva la hora de la base, una base POSTERIOR
// puede vencer ANTES que una anterior (sabado 14:03 y domingo 09:50 caen los
// dos en el mismo lunes, pero a distinta hora). No es un fallo de codigo, es
// que la semantica estaba mal: el articulo 3.2.b del Reglamento 1182/71 y el
// articulo 30 de la Ley 39/2015 dicen que un plazo por dias o meses termina
// "con la expiracion de la ultima hora del ultimo dia". Solo los plazos en
// horas vencen en un instante exacto.
type Cierre uint8

const (
	CierreAuto   Cierre = iota // dias o meses -> fin del dia; solo horas -> instante exacto
	CierreExacto               // fuerza instante exacto (SLA internos, plazos contractuales)
	CierreFinDia               // fuerza fin del dia
)

// Regimen es el cuarteto (computo, calendario, regla de cierre, fin de dia)
// con su fuente normativa.
type Regimen struct {
	Comp   Computo
	Cal    *Calendario
	Trasl  Traslado
	Cierre Cierre
	Fuente string
}

// Calendario es un dato con procedencia y ambito, no una constante del codigo.
// El articulo 30.6 de la Ley 39/2015 hace inhabil el dia que lo sea en la
// localidad del interesado O en la sede del organo, por eso existe Combinar.
type Calendario struct {
	ID       string
	Zona     *time.Location
	Fuente   string
	Ambito   string // "nacional", "autonomico:MD", "local:28079", "ue"
	festivos map[string]struct{}
}

func NuevoCalendario(id, ambito, fuente string, zona *time.Location, festivos ...string) *Calendario {
	c := &Calendario{ID: id, Ambito: ambito, Fuente: fuente, Zona: zona, festivos: map[string]struct{}{}}
	for _, f := range festivos {
		c.festivos[f] = struct{}{}
	}
	return c
}

func Combinar(id string, cals ...*Calendario) *Calendario {
	if len(cals) == 0 {
		panic("Combinar sin calendarios")
	}
	out := &Calendario{ID: id, Zona: cals[0].Zona, Ambito: "combinado", festivos: map[string]struct{}{}}
	amb := []string{}
	for _, c := range cals {
		amb = append(amb, c.Ambito)
		for f := range c.festivos {
			out.festivos[f] = struct{}{}
		}
	}
	sort.Strings(amb)
	out.Fuente = fmt.Sprintf("union de %v", amb)
	return out
}

func (c *Calendario) EsFestivo(t time.Time) bool {
	_, ok := c.festivos[t.In(c.Zona).Format("2006-01-02")]
	return ok
}

func (c *Calendario) EsHabil(t time.Time) bool {
	t = t.In(c.Zona)
	switch t.Weekday() {
	case time.Saturday, time.Sunday:
		return false
	}
	return !c.EsFestivo(t)
}

// Sumar aplica una duracion a un instante bajo un regimen y devuelve la
// derivacion completa, que es literalmente lo que imprime `plazum explain`.
func Sumar(base time.Time, d Duracion, reg Regimen) (time.Time, string) {
	if d.Indeterminado {
		return time.Time{}, "limite indeterminado: la norma no fija plazo"
	}
	cal := reg.Cal
	loc := cal.Zona
	t := base.In(loc)
	regla := fmt.Sprintf("base %s", t.Format(time.RFC3339))

	if d.Meses != 0 {
		t = sumarMeses(t, d.Meses, loc)
		regla += fmt.Sprintf(" ; +%d mes(es) de fecha a fecha con recorte al ultimo dia -> %s", d.Meses, t.Format(time.RFC3339))
	}
	if d.Dias != 0 {
		switch reg.Comp {
		case Naturales:
			t = t.AddDate(0, 0, d.Dias).In(loc)
			regla += fmt.Sprintf(" ; +%d dia(s) naturales a la misma hora de pared -> %s", d.Dias, t.Format(time.RFC3339))
		case Habiles:
			restantes := d.Dias
			for restantes > 0 {
				t = t.AddDate(0, 0, 1).In(loc)
				if cal.EsHabil(t) {
					restantes--
				}
			}
			regla += fmt.Sprintf(" ; +%d dia(s) habiles segun calendario %s -> %s", d.Dias, cal.ID, t.Format(time.RFC3339))
		}
	}
	if d.Horas != 0 || d.Mins != 0 {
		t = t.Add(time.Duration(d.Horas)*time.Hour + time.Duration(d.Mins)*time.Minute)
		regla += fmt.Sprintf(" ; +%dh%dm de tiempo absoluto -> %s", d.Horas, d.Mins, t.In(loc).Format(time.RFC3339))
	}
	finDia := reg.Cierre == CierreFinDia ||
		(reg.Cierre == CierreAuto && (d.Meses != 0 || d.Dias != 0))

	if finDia {
		t = finDelDia(t, loc)
		regla += fmt.Sprintf(" ; plazo por dias o meses: vence al final del ultimo dia (Rgto. 1182/71 art. 3.2.b) -> %s",
			t.Format(time.RFC3339))
	}
	if reg.Trasl == TrasladoSiguienteHabil && !cal.EsHabil(t) {
		orig := t
		for !cal.EsHabil(t) {
			t = t.AddDate(0, 0, 1).In(loc)
		}
		t = finDelDia(t, loc)
		regla += fmt.Sprintf(" ; vencia en inhabil (%s), traslado al fin del siguiente habil por %s -> %s",
			orig.Format("2006-01-02"), reg.Fuente, t.Format(time.RFC3339))
	}
	return t.In(loc), regla
}

func finDelDia(t time.Time, loc *time.Location) time.Time {
	y, m, d := t.In(loc).Date()
	return time.Date(y, m, d, 23, 59, 59, 0, loc)
}

// Restar lleva un instante HACIA ATRAS bajo un regimen, y no es Sumar con el
// signo cambiado.
//
// POR QUE EXISTE. La familia de preaviso contractual se calcula al reves que
// todas las demas: la fecha limite es un DATO DE ENTRADA (el dia en que el
// obligado quiere que su decision surta efecto) y lo que se calcula es cuando
// hay que empezar. Son siete relojes del censo, en psd2, mica, mdr y data-act.
//
// LO QUE CAMBIA DE SIGNO, Y ES LO QUE HACE FALTA UNA FUNCION PROPIA: el
// traslado. Sumando, un vencimiento que cae en inhabil se traslada al SIGUIENTE
// habil, y eso favorece al obligado porque le da mas tiempo. Restando, hacer lo
// mismo seria un error legal: mover el ultimo dia util de aviso hacia adelante
// ACORTA la antelacion por debajo del minimo que exige la norma, que es
// justamente lo que no se puede hacer. Un preaviso de dos meses que se entrega
// un dia tarde no es un preaviso de dos meses.
//
// Asi que aqui TrasladoSiguienteHabil significa "al habil ANTERIOR". Se
// conserva el nombre de la constante porque el eje es el mismo (¿se admite
// vencer en inhabil?) y lo que cambia es la direccion en la que se resuelve.
//
// LO QUE NO CAMBIA: el cierre. Un preaviso vence en el INSTANTE exacto
// base menos duracion, no al final de aquel dia, y esto es deliberado. "Con al
// menos dos meses de antelacion" es una medida de distancia entre dos
// instantes: avisar EN ese instante deja exactamente dos meses y avisar un
// minuto despues deja menos. Redondear al final del dia regalaria hasta 24
// horas que la norma no da. Redondear al final del dia ANTERIOR quitaria hasta
// 24 horas que la norma si da, y para hacer eso haria falta una fuente que lo
// diga; no hay ninguna, asi que no se inventa.
func Restar(base time.Time, d Duracion, reg Regimen) (time.Time, string) {
	if d.Indeterminado {
		return time.Time{}, "antelacion indeterminada: la norma no fija cuanta"
	}
	cal := reg.Cal
	loc := cal.Zona
	t := base.In(loc)
	regla := fmt.Sprintf("efecto %s", t.Format(time.RFC3339))

	if d.Meses != 0 {
		t = sumarMeses(t, -d.Meses, loc)
		regla += fmt.Sprintf(" ; -%d mes(es) de fecha a fecha con recorte al ultimo dia -> %s",
			d.Meses, t.Format(time.RFC3339))
	}
	if d.Dias != 0 {
		switch reg.Comp {
		case Naturales:
			t = t.AddDate(0, 0, -d.Dias).In(loc)
			regla += fmt.Sprintf(" ; -%d dia(s) naturales a la misma hora de pared -> %s",
				d.Dias, t.Format(time.RFC3339))
		case Habiles:
			restantes := d.Dias
			for restantes > 0 {
				t = t.AddDate(0, 0, -1).In(loc)
				if cal.EsHabil(t) {
					restantes--
				}
			}
			regla += fmt.Sprintf(" ; -%d dia(s) habiles segun calendario %s -> %s",
				d.Dias, cal.ID, t.Format(time.RFC3339))
		}
	}
	if d.Horas != 0 || d.Mins != 0 {
		t = t.Add(-(time.Duration(d.Horas)*time.Hour + time.Duration(d.Mins)*time.Minute))
		regla += fmt.Sprintf(" ; -%dh%dm de tiempo absoluto -> %s",
			d.Horas, d.Mins, t.In(loc).Format(time.RFC3339))
	}
	if reg.Trasl == TrasladoSiguienteHabil && !cal.EsHabil(t) {
		orig := t
		for !cal.EsHabil(t) {
			t = t.AddDate(0, 0, -1).In(loc)
		}
		regla += fmt.Sprintf(" ; el ultimo dia de aviso caia en inhabil (%s) y se ADELANTA al "+
			"habil anterior, no se retrasa: retrasarlo acortaria la antelacion por debajo del "+
			"minimo (%s) -> %s", orig.Format("2006-01-02"), reg.Fuente, t.Format(time.RFC3339))
	}
	return t.In(loc), regla
}
