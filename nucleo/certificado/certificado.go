// Package certificado modela el ciclo de vida del certificado como objeto de
// primera clase: emision, vigilancias, recertificacion, ventanas de
// observacion, y las obligaciones internas que cada hito dispara.
//
// Los hitos son primitivas del motor temporal: la vigilancia anual de ISO es
// una periodica, la bienal del ENS otra, la ventana de SOC 2 una continua.
// Nada de fechas sueltas en campos: relojes con regimen y regla citable.
package certificado

import (
	"fmt"
	"sort"
	"time"

	"dutiq/nucleo/ventana"
)

type EstadoCert uint8

const (
	Vigente EstadoCert = iota
	EnVigilancia
	Suspendido
	Retirado
)

func (e EstadoCert) String() string {
	return [...]string{"vigente", "en_vigilancia", "suspendido", "retirado"}[e]
}

// HitoCert es un reloj del ciclo con las obligaciones internas que dispara
// (preparar la vigilancia es una obligacion con reloj como cualquier otra).
type HitoCert struct {
	Tipo      string // vigilancia | recertificacion | ventana_observacion | informe
	Primitiva ventana.Primitiva
	Genera    []string // IDs de obligaciones internas que dispara
	Cita      string
}

type Certificado struct {
	ID, Marco, Alcance, Emisor string
	Emision                    time.Time
	Hitos                      []HitoCert
	Estado                     EstadoCert
}

// VencimientoCert es un vencimiento del ciclo con su procedencia completa.
type VencimientoCert struct {
	Certificado string
	Tipo        string
	ventana.Vencimiento
	Genera []string
	Cita   string
}

// Vencimientos calcula todos los relojes del ciclo hasta un horizonte, con la
// regla de cada fecha. Un certificado suspendido o retirado no genera relojes:
// genera un hallazgo, y eso es de otra capa.
func (c Certificado) Vencimientos(h ventana.Hechos, hasta time.Time) []VencimientoCert {
	if c.Estado == Suspendido || c.Estado == Retirado {
		return nil
	}
	var out []VencimientoCert
	for _, hito := range c.Hitos {
		for _, v := range hito.Primitiva.Vencimientos(h, hasta) {
			out = append(out, VencimientoCert{
				Certificado: c.ID, Tipo: hito.Tipo, Vencimiento: v,
				Genera: hito.Genera, Cita: hito.Cita,
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Vence.Before(out[j].Vence) })
	return out
}

// Proximo devuelve el siguiente vencimiento estrictamente posterior a un
// instante: la pregunta de la pantalla de certificados.
func (c Certificado) Proximo(h ventana.Hechos, desde time.Time, horizonte time.Time) (VencimientoCert, error) {
	for _, v := range c.Vencimientos(h, horizonte) {
		if v.Vence.After(desde) {
			return v, nil
		}
	}
	return VencimientoCert{}, fmt.Errorf("certificado %s: sin vencimientos tras %s en el horizonte dado",
		c.ID, desde.Format(time.RFC3339))
}
