package ventana

import (
	"fmt"
	"sort"
	"time"
)

// EstadoVenc distingue tres situaciones que un checklist funde en una sola.
type EstadoVenc uint8

const (
	Determinado      EstadoVenc = iota // hay fecha y hora exactas
	PendienteDeHecho                   // el reloj cuelga de un hecho que aun no ha ocurrido
	SinPlazoLegal                      // la norma no fija limite ("sin dilacion indebida")
)

func (e EstadoVenc) String() string {
	switch e {
	case PendienteDeHecho:
		return "pendiente de hecho"
	case SinPlazoLegal:
		return "sin plazo legal"
	}
	return "determinado"
}

// Lectura es una interpretacion discrepante del mismo plazo. Existe porque el
// derecho no siempre da una sola respuesta: el EDPB sostiene que las 72 horas
// del articulo 33 del RGPD son 72 horas exactas caiga donde caiga, y hay
// doctrina que aplica el articulo 3 del Reglamento 1182/71 para extenderlas.
// El motor no elige en silencio: calcula las dos, marca la elegida y ensena la
// divergencia con su cita.
type Lectura struct {
	ID     string
	Limite Duracion
	Reg    Regimen
	Cita   string
}

type Divergencia struct {
	Lectura string
	Vence   time.Time
	Cita    string
	Delta   time.Duration
}

type Vencimiento struct {
	Hito   string
	Estado EstadoVenc
	Vence  time.Time
	Regla  string
	Aviso  string
	// NoAntesDe es el SUELO que fija la norma cuando la fecha final todavia no
	// se sabe. Solo lo rellena la primitiva Maximo, y solo cuando el estado es
	// PendienteDeHecho.
	//
	// POR QUE HACE FALTA UN CAMPO Y NO VALE METERLO EN Vence. Un plazo de la
	// forma "diez anos o el periodo de soporte, el que sea mayor" tiene, antes
	// de que el obligado declare el periodo, una fecha final DESCONOCIDA y un
	// suelo CONOCIDO. Ponerlo en Vence con estado Determinado seria mentir: la
	// fecha puede alargarse. Dejarlo fuera seria tirar un dato legal cierto,
	// porque la retencion no puede terminar antes del suelo pase lo que pase.
	// Son dos cosas distintas y se dicen las dos.
	NoAntesDe    time.Time
	Divergencias []Divergencia
}

type Intervalo struct{ Desde, Hasta time.Time }

func (i Intervalo) Contiene(t time.Time) bool {
	return !t.Before(i.Desde) && (i.Hasta.IsZero() || t.Before(i.Hasta))
}

// Hechos son los sucesos registrados: el disparador y el cumplimiento efectivo
// de cada hito, con la clave "<hito>.cumplido". Un plazo encadenado no tiene
// fecha hasta que el hito del que cuelga se cumple DE VERDAD, no cuando deberia
// haberse cumplido.
type Hechos map[string]time.Time

// Modificador altera un plazo ya calculado.
type Modificador interface {
	aplicar(t time.Time, reg Regimen) (time.Time, string)
}

// Suspension para el reloj entre dos instantes (articulos 22 y 68 de la Ley 39/2015).
type Suspension struct {
	Desde, Hasta time.Time
	Motivo       string
}

func (s Suspension) aplicar(t time.Time, reg Regimen) (time.Time, string) {
	if !s.Hasta.After(s.Desde) || !t.After(s.Desde) {
		return t, ""
	}
	if reg.Comp == Habiles {
		n := 0
		for d := s.Desde; d.Before(s.Hasta); d = d.AddDate(0, 0, 1) {
			if reg.Cal.EsHabil(d) {
				n++
			}
		}
		nt, _ := Sumar(t, Duracion{Dias: n}, Regimen{Comp: Habiles, Cal: reg.Cal})
		return nt, fmt.Sprintf("suspension (%s): +%d dias habiles -> %s", s.Motivo, n, nt.Format(time.RFC3339))
	}
	nt := t.Add(s.Hasta.Sub(s.Desde))
	return nt, fmt.Sprintf("suspension (%s): +%s -> %s", s.Motivo, s.Hasta.Sub(s.Desde), nt.Format(time.RFC3339))
}

// Prorroga amplia el plazo (articulo 32 de la Ley 39/2015, articulo 12.3 del RGPD).
type Prorroga struct {
	D      Duracion
	Motivo string
}

func (p Prorroga) aplicar(t time.Time, reg Regimen) (time.Time, string) {
	nt, r := Sumar(t, p.D, reg)
	return nt, fmt.Sprintf("prorroga (%s): %s", p.Motivo, r)
}

// ---------------------------------------------------------------------------
// Las seis primitivas
// ---------------------------------------------------------------------------

type Primitiva interface {
	Nombre() string
	Vencimientos(h Hechos, hasta time.Time) []Vencimiento
}

// 1. Puntual
type Puntual struct {
	Hito string
	En   time.Time
}

func (Puntual) Nombre() string { return "puntual" }
func (p Puntual) Vencimientos(Hechos, time.Time) []Vencimiento {
	return []Vencimiento{{Hito: p.Hito, Estado: Determinado, Vence: p.En, Regla: "instante fijado por la norma"}}
}

// 2. Periodica
type Periodica struct {
	Hito   string
	Desde  time.Time
	Cada   Duracion
	Gracia Duracion
	Reg    Regimen
}

func (Periodica) Nombre() string { return "periodica" }
func (p Periodica) Vencimientos(_ Hechos, hasta time.Time) []Vencimiento {
	var out []Vencimiento
	for i := 1; i <= 400; i++ {
		nt, r := Sumar(p.Desde, Duracion{Meses: p.Cada.Meses * i, Dias: p.Cada.Dias * i, Horas: p.Cada.Horas * i}, p.Reg)
		if nt.After(hasta) {
			break
		}
		lim, rg := Sumar(nt, p.Gracia, p.Reg)
		out = append(out, Vencimiento{Hito: fmt.Sprintf("%s#%d", p.Hito, i), Estado: Determinado, Vence: lim,
			Regla: fmt.Sprintf("ocurrencia %d: %s ; gracia: %s", i, r, rg)})
	}
	return out
}

// 3. Continua
type Continua struct {
	Hito string
	I    Intervalo
}

func (Continua) Nombre() string { return "continua" }
func (c Continua) Vencimientos(Hechos, time.Time) []Vencimiento {
	if c.I.Hasta.IsZero() {
		return []Vencimiento{{Hito: c.Hito, Estado: SinPlazoLegal, Regla: "vigencia continua sin fecha de fin declarada"}}
	}
	return []Vencimiento{{Hito: c.Hito, Estado: Determinado, Vence: c.I.Hasta, Regla: "fin del intervalo de vigencia"}}
}

// 4. Plazo
type Hito struct {
	ID           string
	Limite       Duracion
	Reg          Regimen
	DesdeHito    string // vacio: desde el disparador. Si no: desde el CUMPLIMIENTO de ese hito
	Alternativas []Lectura
	Nota         string
}

type Plazo struct {
	Disparador string
	Hitos      []Hito
	Mods       []Modificador
}

func (Plazo) Nombre() string { return "plazo" }
func (p Plazo) Vencimientos(h Hechos, _ time.Time) []Vencimiento {
	var out []Vencimiento
	disp, okDisp := h[p.Disparador]
	for _, hi := range p.Hitos {
		base, ok := disp, okDisp
		origen := "disparador " + p.Disparador
		if hi.DesdeHito != "" {
			base, ok = h[hi.DesdeHito+".cumplido"]
			origen = "cumplimiento efectivo de " + hi.DesdeHito
		}
		if !ok {
			out = append(out, Vencimiento{Hito: hi.ID, Estado: PendienteDeHecho,
				Regla: "el reloj no ha arrancado: falta " + origen, Aviso: hi.Nota})
			continue
		}
		if hi.Limite.Indeterminado {
			out = append(out, Vencimiento{Hito: hi.ID, Estado: SinPlazoLegal,
				Regla: "la norma no fija limite; se mide el tiempo transcurrido desde " + origen, Aviso: hi.Nota})
			continue
		}
		t, r := Sumar(base, hi.Limite, hi.Reg)
		regla := fmt.Sprintf("%s (%s) ; %s ; computo %s", origen, base.In(hi.Reg.Cal.Zona).Format(time.RFC3339), r, hi.Reg.Comp)
		for _, m := range p.Mods {
			nt, rm := m.aplicar(t, hi.Reg)
			if rm != "" {
				t, regla = nt, regla+" ; "+rm
			}
		}
		v := Vencimiento{Hito: hi.ID, Estado: Determinado, Vence: t, Regla: regla, Aviso: hi.Nota}
		for _, alt := range hi.Alternativas {
			at, ar := Sumar(base, alt.Limite, alt.Reg)
			if !at.Equal(t) {
				v.Divergencias = append(v.Divergencias, Divergencia{
					Lectura: alt.ID, Vence: at, Cita: alt.Cita, Delta: at.Sub(t)})
				_ = ar
			}
		}
		out = append(out, v)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Vence.Before(out[j].Vence) })
	return out
}

// 5. Observacion
type Observacion struct {
	Hito     string
	Ventana  Intervalo
	Muestreo Duracion
	Reg      Regimen
}

func (Observacion) Nombre() string { return "observacion" }
func (o Observacion) Vencimientos(Hechos, time.Time) []Vencimiento {
	var out []Vencimiento
	i := 1
	for t := o.Ventana.Desde; t.Before(o.Ventana.Hasta) && i <= 500; i++ {
		nt, r := Sumar(t, o.Muestreo, o.Reg)
		if nt.After(o.Ventana.Hasta) {
			nt, r = o.Ventana.Hasta, r+" ; recortado al fin de la ventana"
		}
		out = append(out, Vencimiento{Hito: fmt.Sprintf("%s#%d", o.Hito, i), Estado: Determinado, Vence: nt,
			Regla: fmt.Sprintf("punto de muestreo %d: %s", i, r)})
		t = nt
	}
	return out
}

// 6. Secuencia
type Fase struct {
	ID        string
	Duracion  Duracion
	DependeDe string
}

type Secuencia struct {
	Inicio time.Time
	Fases  []Fase
	Reg    Regimen
}

func (Secuencia) Nombre() string { return "secuencia" }
func (s Secuencia) Vencimientos(h Hechos, _ time.Time) []Vencimiento {
	fin := map[string]time.Time{}
	var out []Vencimiento
	for _, f := range s.Fases {
		base, origen := s.Inicio, "inicio de la secuencia"
		if f.DependeDe != "" {
			if real, ok := h[f.DependeDe+".cumplido"]; ok {
				base, origen = real, "cumplimiento efectivo de "+f.DependeDe
			} else if prev, ok := fin[f.DependeDe]; ok {
				base, origen = prev, "fin previsto de "+f.DependeDe
			} else {
				out = append(out, Vencimiento{Hito: f.ID, Estado: PendienteDeHecho, Regla: "fase previa no resuelta: " + f.DependeDe})
				continue
			}
		}
		t, r := Sumar(base, f.Duracion, s.Reg)
		fin[f.ID] = t
		out = append(out, Vencimiento{Hito: f.ID, Estado: Determinado, Vence: t, Regla: origen + " ; " + r})
	}
	return out
}

// ---------------------------------------------------------------------------
// 7. Maximo: el mas TARDIO de dos duraciones sobre la misma base
// ---------------------------------------------------------------------------
//
// LA ENCONTRO EL CENSO, NO EL PLAN. Midiendo el corpus (docs/censo-relojes.md,
// familia E) salio que la mayor familia de relojes del CRA tiene una forma que
// el motor no sabia calcular: "diez anos o el periodo de soporte, el que sea
// mayor". Son 31 relojes solo en `cra`, mas `mica` art. 68.9 (cinco anos, siete
// si la autoridad lo pide antes de que venzan). Sin esta primitiva, esos
// paquetes se escribirian aproximados y habria que reescribirlos.
//
// LA TRAMPA, Y ES LA RAZON DE SER DE LA PRIMITIVA. La rama declarada NO PUEDE
// ACORTAR el suelo legal. Un fabricante que declara un periodo de soporte de
// dos anos no reduce a dos anos una retencion de diez: la norma dice "el que
// sea mayor", asi que en ese caso gana el suelo y la declaracion no hace nada.
// Implementar esto como "usa la fecha declarada si la hay" es exactamente el
// error, y da una fecha mas corta que la legal en el sentido peligroso.
//
// Y LA SEGUNDA TRAMPA: cuando la rama declarada todavia no existe, la fecha
// final es DESCONOCIDA y el suelo es CONOCIDO. Devolver el suelo como
// Determinado seria un verde mas debil que se lee igual que uno fuerte, que es
// lo que este proyecto no hace. Se devuelve PendienteDeHecho con NoAntesDe
// puesto: no se sabe cuando termina, y se sabe que no termina antes de ahi.
type Maximo struct {
	Hito string
	// Disparador es el hecho desde el que corren LAS DOS ramas.
	Disparador string
	// Suelo es la duracion que fija la norma. Nunca se puede bajar de aqui.
	Suelo Duracion
	Reg   Regimen
	// Ampliacion es el nombre del hecho que trae la FECHA de la segunda rama:
	// el fin del periodo de soporte que declara el fabricante, o el nuevo
	// limite que impone una autoridad. Vacio significa que esta norma no tiene
	// segunda rama y la primitiva se comporta como un plazo normal.
	Ampliacion string
	// Exigible dice si la norma OBLIGA al obligado a declarar la ampliacion.
	// Cuando lo hace, su ausencia no es "no hay ampliacion", es "falta un dato
	// que la norma exige", y eso no se puede presentar como fecha cerrada.
	Exigible bool
	Nota     string
}

func (Maximo) Nombre() string { return "maximo" }

func (m Maximo) Vencimientos(h Hechos, _ time.Time) []Vencimiento {
	base, ok := h[m.Disparador]
	if !ok {
		return []Vencimiento{{Hito: m.Hito, Estado: PendienteDeHecho,
			Regla: "el reloj no ha arrancado: falta el disparador " + m.Disparador, Aviso: m.Nota}}
	}
	if m.Suelo.Indeterminado {
		return []Vencimiento{{Hito: m.Hito, Estado: SinPlazoLegal,
			Regla: "la norma no fija suelo; se mide el tiempo transcurrido desde " + m.Disparador,
			Aviso: m.Nota}}
	}
	suelo, rSuelo := Sumar(base, m.Suelo, m.Reg)
	origen := fmt.Sprintf("disparador %s (%s)", m.Disparador, base.In(m.Reg.Cal.Zona).Format(time.RFC3339))

	if m.Ampliacion == "" {
		return []Vencimiento{{Hito: m.Hito, Estado: Determinado, Vence: suelo,
			Regla: origen + " ; rama unica (la norma no admite ampliacion) ; " + rSuelo, Aviso: m.Nota}}
	}

	amp, hayAmp := h[m.Ampliacion]
	if !hayAmp {
		if m.Exigible {
			return []Vencimiento{{Hito: m.Hito, Estado: PendienteDeHecho, NoAntesDe: suelo,
				Regla: fmt.Sprintf("%s ; suelo legal: %s ; PERO la fecha final depende de %s, "+
					"que la norma obliga a declarar y aun no consta. No termina antes del suelo "+
					"y puede terminar despues", origen, rSuelo, m.Ampliacion),
				Aviso: m.Nota}}
		}
		return []Vencimiento{{Hito: m.Hito, Estado: Determinado, Vence: suelo,
			Regla: fmt.Sprintf("%s ; %s ; la ampliacion (%s) no consta y la norma no obliga a "+
				"declararla, asi que rige el suelo", origen, rSuelo, m.Ampliacion),
			Aviso: m.Nota}}
	}

	amp = amp.In(m.Reg.Cal.Zona)
	// EL MAXIMO, Y SE DICE CUAL GANA. Que la rama declarada sea mas corta no
	// es un error del declarante ni un caso raro: es lo normal cuando el suelo
	// legal es largo, y la norma ya lo previo con "el que sea mayor".
	if amp.After(suelo) {
		return []Vencimiento{{Hito: m.Hito, Estado: Determinado, Vence: amp,
			Regla: fmt.Sprintf("%s ; suelo legal: %s ; ampliacion declarada (%s): %s ; "+
				"GANA LA AMPLIACION por %s", origen, rSuelo, m.Ampliacion,
				amp.Format(time.RFC3339), amp.Sub(suelo).Round(time.Hour)),
			Aviso: m.Nota}}
	}
	return []Vencimiento{{Hito: m.Hito, Estado: Determinado, Vence: suelo,
		Regla: fmt.Sprintf("%s ; suelo legal: %s ; ampliacion declarada (%s): %s ; "+
			"GANA EL SUELO: la fecha declarada es %s mas corta y una declaracion del obligado "+
			"no reduce el minimo legal", origen, rSuelo, m.Ampliacion,
			amp.Format(time.RFC3339), suelo.Sub(amp).Round(time.Hour)),
		Aviso: m.Nota}}
}

// ---------------------------------------------------------------------------
// 8. Preaviso: el plazo que corre HACIA ATRAS
// ---------------------------------------------------------------------------
//
// FAMILIA G DEL CENSO, la septima, y no existia en el plan porque no habia
// ningun reloj de esta forma contado. Ahora hay siete: psd2 arts. 54.1, 55.1 y
// 55.3, mica arts. 65.4 y 67.4.b, mdr art. 75.3 y data-act art. 25.2.d.
//
// EN QUE SE DISTINGUE DE TODAS LAS DEMAS. En las otras seis primitivas ocurre
// un hecho y se calcula cuando vence. Aqui el obligado ELIGE una fecha futura
// (cuando quiere que su decision surta efecto) y lo que se calcula es hasta
// cuando puede seguir callado. La fecha limite es un dato de ENTRADA.
//
// La consecuencia practica, y es la que hace util a la primitiva: el
// vencimiento SE MUEVE cuando se mueve el hito. Adelantar la fecha de efecto
// adelanta la fecha limite de aviso, y puede dejarla en el pasado, que es
// exactamente la situacion que hay que ensenar y no esconder.
type Preaviso struct {
	Hito string
	// Efecto es el nombre del hecho que trae la fecha en la que la decision va
	// a surtir efecto. La pone el obligado, no le ocurre.
	Efecto string
	// Antelacion es cuanto antes hay que avisar.
	Antelacion Duracion
	Reg        Regimen
	Nota       string
}

func (Preaviso) Nombre() string { return "preaviso" }

func (p Preaviso) Vencimientos(h Hechos, _ time.Time) []Vencimiento {
	efecto, ok := h[p.Efecto]
	if !ok {
		return []Vencimiento{{Hito: p.Hito, Estado: PendienteDeHecho,
			Regla: "no hay nada que preavisar todavia: falta la fecha de efecto " + p.Efecto,
			Aviso: p.Nota}}
	}
	if p.Antelacion.Indeterminado {
		return []Vencimiento{{Hito: p.Hito, Estado: SinPlazoLegal,
			Regla: "la norma exige preaviso pero no fija cuanta antelacion", Aviso: p.Nota}}
	}
	t, r := Restar(efecto, p.Antelacion, p.Reg)
	return []Vencimiento{{Hito: p.Hito, Estado: Determinado, Vence: t,
		Regla: fmt.Sprintf("cuenta atras desde la fecha de efecto que fija el obligado (%s) ; %s ; "+
			"avisar despues de ese instante incumple la antelacion", p.Efecto, r),
		Aviso: p.Nota}}
}
