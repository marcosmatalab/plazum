package corpus

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// ---------------------------------------------------------------------------
// Cuando toca un escalon
// ---------------------------------------------------------------------------

// Sentido dice si el escalon cae antes o despues del vencimiento.
type Sentido uint8

const (
	// TrasElVencimiento: el plazo ya paso y nadie lo cerro.
	TrasElVencimiento Sentido = iota
	// AntesDelVencimiento: el aviso de cortesia, el sufijo `_antes`.
	AntesDelVencimiento
)

// Momento es un `tras` de un escalon ya leido.
type Momento struct {
	D       ventana.Duracion
	Sentido Sentido
}

var (
	ErrTrasVacio      = errors.New("escalon sin `tras`")
	ErrTrasIlegible   = errors.New("el `tras` de un escalon no es una duracion")
	ErrTrasIndetermin = errors.New("un escalon no puede tener antelacion indeterminada")
)

// ParseTras lee el `tras` de un escalon, con su sufijo `_antes` opcional.
//
// POR QUE ESTO NO EXISTIA HASTA HOY. El campo esta declarado en el formato
// desde el primer dia y NADIE LO HABIA PARSEADO NUNCA: el linter comprobaba que
// no estuviera vacio. Los 53 escalones del corpus resultan estar todos bien
// escritos — se midio antes de escribir esto, no se supuso — pero eso era
// suerte, no una propiedad: el primer `P60D_ants` habria salido el dia del
// incidente, en el unico momento en que nadie quiere descubrir un error de
// formato.
func ParseTras(s string) (Momento, error) {
	if s == "" {
		return Momento{}, ErrTrasVacio
	}
	m := Momento{Sentido: TrasElVencimiento}
	base := s
	if resto, ok := strings.CutSuffix(s, "_antes"); ok {
		m.Sentido, base = AntesDelVencimiento, resto
	}
	d, err := ventana.ParseDuracion(base)
	if err != nil {
		return Momento{}, fmt.Errorf("%w: %q (%v). Se escribe en ISO-8601, con `_antes` "+
			"opcional para el aviso de cortesia: P60D_antes, PT4H", ErrTrasIlegible, s, err)
	}
	if d.Indeterminado {
		return Momento{}, fmt.Errorf("%w: %q. Un escalon sin numero no se puede programar, "+
			"asi que no avisaria nunca", ErrTrasIndetermin, s)
	}
	m.D = d
	return m, nil
}

// Instante calcula cuando toca el escalon respecto de un vencimiento.
//
// Suma o RESTA segun el sentido, y la resta es la de `ventana` y no una suma de
// duracion negada: `Sumar` con dias habiles negativos recorre `for restantes >
// 0`, que con un numero negativo no entra ni una vez y devuelve la base sin
// tocar. O sea, un aviso "60 dias habiles antes" que cae el mismo dia del
// vencimiento, sin error y sin rastro.
func (m Momento) Instante(vence time.Time, reg ventana.Regimen) (time.Time, string) {
	if m.Sentido == AntesDelVencimiento {
		return ventana.Restar(vence, m.D, reg)
	}
	return ventana.Sumar(vence, m.D, reg)
}
