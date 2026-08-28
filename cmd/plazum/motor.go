package main

// El montaje del motor de aplicabilidad sobre un corpus y un alcance.
//
// Se extrajo de demo.go el 27-08-2026 por la misma razon que el alcance: `plazum
// calendario` monta exactamente el mismo motor, y dos montajes del mismo motor
// son dos derivas. Este proyecto ya pago esa leccion una vez, con los dos
// traductores del reloj declarado, y el comentario de aquel decia "el dia que
// las dos derivas se separen, el test se pone rojo". Fue ese dia.

import (
	"fmt"
	"strings"

	"github.com/marcosmatalab/plazum/nucleo/aplicabilidad"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// derivacion es un motor ya evaluado, con lo que hace falta para explicar cada
// conclusion. Se devuelve entero porque enseñar una obligacion sin poder decir
// de que regla sale es lo que hace un checklist, y este producto existe para lo
// contrario.
type derivacion struct {
	Motor  *aplicabilidad.Motor
	Reglas map[string]corpus.ReglaSpec
	// PaquetesConReglas es cuantos paquetes aportaron un programa. Los demas no
	// declaran aplicabilidad y por tanto NO PUEDEN derivar nada: es el numero
	// que separa "no te aplica" de "el corpus no sabe si te aplica", y son
	// cosas muy distintas.
	PaquetesConReglas int
	PaquetesSinReglas int
}

// montarMotor carga los programas de todos los paquetes que declaran reglas,
// afirma los hechos del alcance y evalua.
func montarMotor(ps []*corpus.Paquete, al alcance) (derivacion, error) {
	d := derivacion{Motor: aplicabilidad.NuevoMotor(), Reglas: map[string]corpus.ReglaSpec{}}
	for _, p := range ps {
		for _, r := range p.Aplicabilidad.Reglas {
			d.Reglas[p.URN+":"+r.ID] = r
		}
		if len(p.Aplicabilidad.Reglas) == 0 {
			d.PaquetesSinReglas++
			continue
		}
		d.PaquetesConReglas++
		prog, errs := p.Programa()
		if len(errs) > 0 {
			return d, fmt.Errorf("las reglas de %s no compilan: %v", p.URN, errs)
		}
		if err := d.Motor.Cargar(prog); err != nil {
			return d, fmt.Errorf("el motor rechaza las reglas de %s: %w", p.URN, err)
		}
	}
	for _, h := range al.Hechos {
		hecho := aplicabilidad.H(h.Pred, h.Args...)
		hecho.Procedencia = "declarado en el alcance"
		d.Motor.Afirmar(hecho)
	}
	if _, err := d.Motor.Evaluar(); err != nil {
		return d, fmt.Errorf("evaluando la aplicabilidad: %w", err)
	}
	return d, nil
}

// aplicable es una obligacion derivada con la regla que la derivo.
type aplicable struct {
	Obligacion string
	Regla      corpus.ReglaSpec
	IDRegla    string
}

// aplicablesDe consulta que obligaciones le alcanzan al sujeto y de que regla
// sale cada una.
func aplicablesDe(d derivacion, sujeto string) []aplicable {
	var out []aplicable
	for _, h := range d.Motor.Consultar(aplicabilidad.A("aplica",
		aplicabilidad.V("O"), aplicabilidad.C(sujeto))) {
		idRegla := strings.TrimPrefix(d.Motor.Explicar(h), "regla ")
		idRegla = strings.TrimSuffix(idRegla, " (agregado)")
		out = append(out, aplicable{Obligacion: h.Args[0], Regla: d.Reglas[idRegla], IDRegla: idRegla})
	}
	return out
}
