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

	"plazum/nucleo/ventana"
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
	esperado, err := parseFecha(d.Esperado.Vence)
	if err != nil {
		return fmt.Errorf("dorado %q: esperado.vence ilegible: %w", d.Caso, err)
	}
	// TODOS los hechos del dorado, no solo el disparador.
	//
	// Hasta el 26-08-2026 aqui solo se leia el disparador, asi que un dorado no
	// podia expresar ni el cumplimiento de un hito previo (que es de donde
	// cuenta la notificacion intermedia) ni la clasificacion del incidente (que
	// es lo que decide el limite). Los dos son hechos con su instante, asi que
	// caben en el mismo mapa y no hacia falta un tipo nuevo: lo que faltaba era
	// pasarlos.
	todosLosHechos := func() (ventana.Hechos, error) {
		h := ventana.Hechos{}
		for k, v := range d.Hechos {
			t, err := parseFecha(v)
			if err != nil {
				return nil, fmt.Errorf("el hecho %q no es una fecha legible: %w", k, err)
			}
			h[k] = t
		}
		return h, nil
	}

	// UNA SOLA TRADUCCION, la misma que usa el producto (VencimientosDe). Aqui
	// habia una copia con su propia tabla de regimenes, y por eso un dorado
	// podia pasar contra una deriva que el producto no hacia.
	hechos, err := todosLosHechos()
	if err != nil {
		return fmt.Errorf("dorado %q: %w", d.Caso, err)
	}
	if _, ok := hechos[tmp.Disparador["hecho"]]; !ok && tmp.Primitiva != "puntual" {
		return fmt.Errorf("dorado %q: falta el hecho %q, que es el disparador de la obligacion",
			d.Caso, tmp.Disparador["hecho"])
	}
	vs, err := VencimientosDe(o, hechos, esperado.Add(24*time.Hour))
	if err != nil {
		return fmt.Errorf("dorado %q: %w", d.Caso, err)
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

// hitosDe traduce los hitos declarados por el paquete a los del motor.
//
// El limite vacio o "indeterminado" produce una Duracion indeterminada, que es
// como el motor dice "la norma exige la accion y no fija plazo". Es el caso de
// la notificacion inicial de la tabla 3 del anexo del RD 43/2021, que dice
// "Inmediata": hay obligacion y no hay numero, y las dos cosas son ciertas a la
// vez. Devolver ahi cero horas seria inventarse un plazo que la norma no da.
func hitosDe(specs []HitoSpec, reg ventana.Regimen) ([]ventana.Hito, error) {
	out := make([]ventana.Hito, 0, len(specs))
	vistos := map[string]bool{}
	for i, h := range specs {
		if h.ID == "" {
			return nil, fmt.Errorf("el hito %d no tiene id, asi que no se puede referenciar "+
				"desde desde_hito ni desde el esperado de un dorado", i)
		}
		if vistos[h.ID] {
			return nil, fmt.Errorf("el hito %q esta declarado dos veces: desde_hito no sabria "+
				"a cual de los dos apunta", h.ID)
		}
		vistos[h.ID] = true
		lim, err := limiteDe(h.Limite)
		if err != nil {
			return nil, fmt.Errorf("hito %q: %w", h.ID, err)
		}
		hito := ventana.Hito{ID: h.ID, Limite: lim, Reg: reg,
			DesdeHito: h.DesdeHito, Clase: h.Clase, Nota: h.Nota}
		if h.Tope != nil {
			lt, err := ventana.ParseDuracion(h.Tope.Limite)
			if err != nil {
				return nil, fmt.Errorf("hito %q, tope desde %q: %w", h.ID, h.Tope.Desde, err)
			}
			hito.Tope = &ventana.Tope{Desde: h.Tope.Desde, Limite: lt, Reg: reg,
				Caduca: h.Tope.Caduca, Cita: h.Tope.Cita}
		}
		for _, a := range h.Alternativas {
			al, err := limiteDe(a.Limite)
			if err != nil {
				return nil, fmt.Errorf("hito %q, lectura %q: %w", h.ID, a.ID, err)
			}
			hito.Alternativas = append(hito.Alternativas,
				ventana.Lectura{ID: a.ID, Limite: al, Reg: reg, Cita: a.Cita})
		}
		out = append(out, hito)
	}
	// desde_hito tiene que apuntar a un hito QUE EXISTE. Un encadenamiento a un
	// nombre mal escrito deja ese hito pendiente para siempre, y eso se lee
	// como "el reloj no ha arrancado" en vez de como el error que es.
	for _, h := range specs {
		if h.DesdeHito != "" && !vistos[h.DesdeHito] {
			return nil, fmt.Errorf("el hito %q cuenta desde %q, que no existe en esta "+
				"obligacion. Un encadenamiento a un nombre que no esta deja el hito pendiente "+
				"para siempre y se lee como si el reloj no hubiera arrancado", h.ID, h.DesdeHito)
		}
	}
	return out, nil
}

// limiteDe admite el vacio y la palabra "indeterminado" como "la norma no fija
// plazo".
func limiteDe(s string) (ventana.Duracion, error) {
	if s == "" || s == "indeterminado" {
		return ventana.Duracion{Indeterminado: true}, nil
	}
	return ventana.ParseDuracion(s)
}

// VencimientosDe traduce el reloj declarado por un paquete a vencimientos del
// motor. ES LA UNICA TRADUCCION QUE HAY, y esa es toda la gracia.
//
// POR QUE ESTA AQUI Y NO EN cmd/plazum. Hasta el 26-08-2026 habia DOS: una en
// el ejecutor de dorados y otra en `cmd/plazum/demo.go`, con la misma tabla de
// regimenes copiada en las dos. El comentario de aquella decia:
//
//	El dia que las dos derivas se separen, el test se pone rojo. Anotado como
//	P1 en docs/pendientes.md: el sitio correcto para esto es una funcion
//	exportada de nucleo/corpus.
//
// Ese dia fue el que se escribio el primer plazo ESCALONADO (nis1-es, tabla 3
// del anexo del RD 43/2021): se enseno a una de las dos a leer los hitos
// encadenados y la otra se quedo atras, y el test se puso rojo exactamente como
// estaba escrito que pasaria.
//
// Es la misma familia que los dos parsers de ASN.1 del adaptador de sellado:
// dos lecturas independientes del mismo dato que no ata ninguna identidad. Aqui
// no era explotable, solo caro, pero el arreglo es el mismo: quedarse con una.
//
// `hasta` acota las primitivas que generan ocurrencias (periodica); las demas
// lo ignoran.
func VencimientosDe(o Obligacion, hechos ventana.Hechos, hasta time.Time) ([]ventana.Vencimiento, error) {
	if o.Temporalidad == nil {
		return nil, fmt.Errorf("la obligacion %s no declara reloj", o.ID)
	}
	t := *o.Temporalidad
	reg, err := regimenDe(t.Regimen)
	if err != nil {
		return nil, fmt.Errorf("obligacion %s: %w", o.ID, err)
	}
	disparador := t.Disparador["hecho"]

	switch t.Primitiva {
	case "periodica":
		base, ok := hechos[disparador]
		if !ok {
			return nil, nil
		}
		cada, err := ventana.ParseDuracion(t.Cadencia)
		if err != nil {
			return nil, fmt.Errorf("obligacion %s: cadencia %q: %w", o.ID, t.Cadencia, err)
		}
		hito := t.Hito
		if hito == "" {
			hito = "ocurrencia"
		}
		p := ventana.Periodica{Hito: hito, Desde: base, Cada: cada, Reg: reg}
		return p.Vencimientos(nil, hasta), nil

	case "plazo":
		hitos, err := hitosDelPlazo(o.ID, t, reg)
		if err != nil {
			return nil, err
		}
		p := ventana.Plazo{Disparador: disparador, Hitos: hitos}
		return p.Vencimientos(hechos, hasta), nil

	default:
		return nil, fmt.Errorf("obligacion %s: la primitiva %q todavia no tiene ejecutor "+
			"(llega con su etapa)", o.ID, t.Primitiva)
	}
}

// hitosDelPlazo devuelve los hitos de un plazo, sean escalonados o el simple.
//
// EL ORDEN IMPORTA Y COSTO UN ROJO: cuando el paquete declara `hitos`, el
// `limite` suelto de arriba NI SIQUIERA SE MIRA. La primera version lo parseaba
// antes, asi que un paquete escalonado (que no tiene limite suelto, porque cada
// hito lleva el suyo) moria con "duracion no reconocida" sobre la cadena vacia.
func hitosDelPlazo(id string, t Temporalidad, reg ventana.Regimen) ([]ventana.Hito, error) {
	if len(t.Hitos) > 0 {
		hs, err := hitosDe(t.Hitos, reg)
		if err != nil {
			return nil, fmt.Errorf("obligacion %s: %w", id, err)
		}
		return hs, nil
	}
	lim, err := ventana.ParseDuracion(t.Limite)
	if err != nil {
		return nil, fmt.Errorf("obligacion %s: limite %q: %w", id, t.Limite, err)
	}
	hito := t.Hito
	if hito == "" {
		hito = "limite"
	}
	return []ventana.Hito{{ID: hito, Limite: lim, Reg: reg}}, nil
}
