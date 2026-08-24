package corpus

// El ejecutor de dorados: convierte la Temporalidad declarada del paquete en
// la primitiva del motor de ventana, calcula, y compara con el esperado que se
// derivo del texto legal. Si discrepan, GANA EL DORADO y se arregla el motor.
//
// Calendario: v1 usa UTC sin festivos (los calendarios por paquete llegan con
// su propia seccion del corpus). Los dorados que dependan de festivos deben
// esperar a esa pieza, y decirlo en su caso.

import (
	"fmt"
	"time"

	"obligo/nucleo/ventana"
)

func regimenDe(r RegimenSpec) (ventana.Regimen, error) {
	reg := ventana.Regimen{Cal: ventana.NuevoCalendario("utc-v1", "dorados", "corpus", time.UTC)}
	switch r.Computo {
	case "naturales", "":
		reg.Comp = ventana.Naturales
	case "habiles":
		reg.Comp = ventana.Habiles
	default:
		return reg, fmt.Errorf("computo %q no reconocido", r.Computo)
	}
	switch r.Cierre {
	case "":
		reg.Cierre = ventana.CierreAuto
	case "exacto":
		reg.Cierre = ventana.CierreExacto
	case "fin_de_dia":
		reg.Cierre = ventana.CierreFinDia
	default:
		return reg, fmt.Errorf("cierre %q no reconocido", r.Cierre)
	}
	switch r.Traslado {
	case "", "ninguno":
		reg.Trasl = ventana.TrasladoNinguno
	case "siguiente_habil":
		reg.Trasl = ventana.TrasladoSiguienteHabil
	default:
		return reg, fmt.Errorf("traslado %q no reconocido", r.Traslado)
	}
	return reg, nil
}

func parseFecha(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}

// EjecutarDorado calcula el reloj de la obligacion con los hechos del caso y
// compara con el esperado. Error = discrepancia motor/texto.
func EjecutarDorado(o Obligacion, d Dorado) error {
	if o.Temporalidad == nil {
		return fmt.Errorf("dorado %q: la obligacion %s no declara temporalidad", d.Caso, o.ID)
	}
	tmp := *o.Temporalidad
	reg, err := regimenDe(tmp.Regimen)
	if err != nil {
		return fmt.Errorf("dorado %q: %w", d.Caso, err)
	}
	esperado, err := parseFecha(d.Esperado.Vence)
	if err != nil {
		return fmt.Errorf("dorado %q: esperado.vence ilegible: %w", d.Caso, err)
	}
	hecho := func() (time.Time, error) {
		clave := tmp.Disparador["hecho"]
		v, ok := d.Hechos[clave]
		if !ok {
			return time.Time{}, fmt.Errorf("falta el hecho %q", clave)
		}
		return parseFecha(v)
	}

	var vs []ventana.Vencimiento
	switch tmp.Primitiva {
	case "periodica":
		desde, err := hecho()
		if err != nil {
			return fmt.Errorf("dorado %q: %w", d.Caso, err)
		}
		cada, err := ventana.ParseDuracion(tmp.Cadencia)
		if err != nil {
			return fmt.Errorf("dorado %q: cadencia: %w", d.Caso, err)
		}
		nombre := tmp.Hito
		if nombre == "" {
			nombre = "ocurrencia"
		}
		p := ventana.Periodica{Hito: nombre, Desde: desde, Cada: cada, Reg: reg}
		vs = p.Vencimientos(nil, esperado.Add(24*time.Hour))
	case "plazo":
		base, err := hecho()
		if err != nil {
			return fmt.Errorf("dorado %q: %w", d.Caso, err)
		}
		lim, err := ventana.ParseDuracion(tmp.Limite)
		if err != nil {
			return fmt.Errorf("dorado %q: limite: %w", d.Caso, err)
		}
		disparador := tmp.Disparador["hecho"]
		nombre := tmp.Hito
		if nombre == "" {
			nombre = "limite"
		}
		p := ventana.Plazo{Disparador: disparador,
			Hitos: []ventana.Hito{{ID: nombre, Limite: lim, Reg: reg}}}
		vs = p.Vencimientos(ventana.Hechos{disparador: base}, esperado.Add(24*time.Hour))
	default:
		return fmt.Errorf("dorado %q: primitiva %q sin ejecutor todavia (llega con su etapa)", d.Caso, tmp.Primitiva)
	}

	for _, v := range vs {
		if d.Esperado.Hito != "" && v.Hito != d.Esperado.Hito {
			continue
		}
		if v.Vence.Equal(esperado) {
			return nil
		}
		if d.Esperado.Hito != "" {
			return fmt.Errorf("dorado %q: el motor dice %s y el texto dice %s (%s). Gana el dorado: arreglar el motor",
				d.Caso, v.Vence.Format(time.RFC3339), esperado.Format(time.RFC3339), d.CitaDelEsperado)
		}
	}
	return fmt.Errorf("dorado %q: ningun vencimiento del motor coincide con %s (%s)",
		d.Caso, esperado.Format(time.RFC3339), d.CitaDelEsperado)
}

// EjecutarDorados corre todos los dorados de un paquete.
func EjecutarDorados(p *Paquete) []error {
	idx := map[string]Obligacion{}
	for _, o := range p.Obligaciones {
		idx[o.ID] = o
	}
	var errs []error
	for _, d := range p.Dorados {
		if err := EjecutarDorado(idx[d.Obligacion], d); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
