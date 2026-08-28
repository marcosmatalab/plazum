// Package historia es el registro bitemporal de cambios de estado.
//
// Dos ejes de tiempo, y la diferencia es la que un auditor pregunta:
//
//	InstanteHecho     cuando paso en el mundo
//	InstanteRegistro  cuando lo supo el sistema
//
// El estado actual es un pliegue de los eventos; el estado en cualquier
// instante pasado es reproducible; corregir el pasado DEJA RASTRO (un evento
// nuevo con InstanteHecho antiguo e InstanteRegistro de hoy), no lo reescribe.
// De aqui salen la ventana de observacion de SOC 2, el cronometro desde el
// primer conocimiento (RGPD art. 33) y el MTTR.
package historia

import (
	"sort"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/estado"
)

type CambioEstado struct {
	Prueba           string
	De, A            estado.Estado
	InstanteHecho    time.Time
	InstanteRegistro time.Time
	Causa            string // observacion | ritual | excepcion | correccion
}

// Historia es append-only. No hay borrado ni edicion: una correccion es un
// evento mas.
type Historia struct {
	eventos []CambioEstado
}

func (h *Historia) Registrar(c CambioEstado) { h.eventos = append(h.eventos, c) }

func (h *Historia) porHecho(prueba string) []CambioEstado {
	var out []CambioEstado
	for _, e := range h.eventos {
		if e.Prueba == prueba {
			out = append(out, e)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].InstanteHecho.Before(out[j].InstanteHecho) })
	return out
}

// EstadoEn reconstruye el estado de una prueba en un instante del MUNDO
// (eje hecho). Reproducible: mismos eventos, mismo resultado.
func (h *Historia) EstadoEn(prueba string, instante time.Time) (estado.Estado, bool) {
	var ultimo estado.Estado
	visto := false
	for _, e := range h.porHecho(prueba) {
		if e.InstanteHecho.After(instante) {
			break
		}
		ultimo, visto = e.A, true
	}
	return ultimo, visto
}

// Ventana devuelve los cambios de una prueba dentro de una ventana de
// observacion (la de SOC 2 tipo II): que estados tuvo y cuando.
func (h *Historia) Ventana(prueba string, desde, hasta time.Time) []CambioEstado {
	var out []CambioEstado
	for _, e := range h.porHecho(prueba) {
		if !e.InstanteHecho.Before(desde) && !e.InstanteHecho.After(hasta) {
			out = append(out, e)
		}
	}
	return out
}

// PrimerConocimiento es el instante de REGISTRO del primer evento que puso la
// prueba en un estado dado: el "tener constancia" del RGPD art. 33, donde
// arranca el reloj de 72 horas.
func (h *Historia) PrimerConocimiento(prueba string, en estado.Estado) (time.Time, bool) {
	reg := append([]CambioEstado(nil), h.porHecho(prueba)...)
	sort.SliceStable(reg, func(i, j int) bool { return reg[i].InstanteRegistro.Before(reg[j].InstanteRegistro) })
	for _, e := range reg {
		if e.A == en {
			return e.InstanteRegistro, true
		}
	}
	return time.Time{}, false
}

// MTTR: media de tiempo entre entrar en un estado que escala y volver a Pass,
// sobre el eje hecho. Cero pares completos = (0, false).
func (h *Historia) MTTR(prueba string) (time.Duration, bool) {
	var total time.Duration
	var pares int
	var abierta *time.Time
	for _, e := range h.porHecho(prueba) {
		ev := e
		if ev.A.EscalaAlAuditor() && abierta == nil {
			abierta = &ev.InstanteHecho
		}
		if ev.A == estado.Pass && abierta != nil {
			total += ev.InstanteHecho.Sub(*abierta)
			pares++
			abierta = nil
		}
	}
	if pares == 0 {
		return 0, false
	}
	return total / time.Duration(pares), true
}
