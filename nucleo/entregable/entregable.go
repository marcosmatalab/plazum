// Package entregable rellena una plantilla del corpus a partir de las fuentes
// que la organizacion tiene, y dice CAMPO A CAMPO de donde salio cada valor y
// que falta.
//
// POR QUE ES UN PAQUETE Y NO UN METODO DEL INCIDENTE. Una plantilla se rellena
// de varios sitios a la vez — atributos del alcance, cumplimiento de
// obligaciones, el incidente que la dispara — y ninguno de ellos es el dueno del
// entregable. Meter el recorrido dentro del incidente le daria al incidente
// opinion sobre plantillas que no le tocan.
//
// LA REGLA QUE MANDA AQUI ES LA DOCTRINA DEL FALSO POSITIVO. Un campo sin dato
// se dice de dos maneras distintas y nunca de una tercera:
//
//	NoConsta            plazum sabria derivarlo y el dato no esta registrado
//	LoAportaUnaPersona  por diseno lo escribe quien responde
//
// Ninguna de las dos es "incumplido", y ninguna se rellena con el valor cero. Un
// entregable que pusiera una fecha cero o una cadena vacia estaria afirmando
// algo que nadie ha dicho, y este es el documento que sale de la organizacion
// hacia una autoridad: un dato inventado ahi no es un fallo de pantalla.
package entregable

import (
	"fmt"
	"sort"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// Fuente es quien sabe contestar a una clase de origen. El nombre llega sin
// prefijo: la gramatica la lee corpus.ParseOrigen y las fuentes no la conocen.
type Fuente interface {
	Campo(nombre string) (string, bool)
}

// Estado es lo que se sabe de un campo. Vocabulario cerrado.
type Estado uint8

const (
	// Derivado: plazum lo ha rellenado con un dato que consta.
	Derivado Estado = iota
	// NoConsta: es derivable y el dato no esta registrado. NO es un
	// incumplimiento y no se presenta como tal.
	NoConsta
	// LoAportaUnaPersona: por diseno lo escribe quien responde. Tampoco es un
	// hueco del producto: es la mitad del trabajo que ninguna maquina hace.
	LoAportaUnaPersona
	// SinFuente: el origen es derivable y NADIE ha aportado la fuente que sabe
	// contestarlo. Es un fallo de quien llama, no del dato, y se distingue de
	// NoConsta a proposito: "no tengo el dato" y "no me has dado de donde
	// sacarlo" se arreglan de maneras distintas.
	SinFuente
)

var nombresDeEstado = [...]string{"derivado", "no consta", "lo aporta una persona", "sin fuente"}

func (e Estado) String() string {
	if int(e) >= len(nombresDeEstado) {
		return fmt.Sprintf("estado desconocido(%d)", uint8(e))
	}
	return nombresDeEstado[e]
}

// Campo es una fila del entregable ya resuelta.
type Campo struct {
	Nombre string
	Valor  string // solo cuando Estado == Derivado
	Estado Estado
	// Que describe el origen en una frase, para que la pantalla que pida el
	// dato que falta tenga algo que decir sin volver al corpus.
	Que string
	// Origen es la cadena tal cual la escribio el paquete, para poder
	// rastrearla desde el documento.
	Origen string
}

// Relleno es la plantilla resuelta entera.
type Relleno struct {
	Plantilla string
	Campos    []Campo
}

// Cuenta devuelve cuantos campos hay en cada estado.
//
// ES LA LEY DE CONSERVACION DE ESTE ENTREGABLE: todo campo de la plantilla cae
// en exactamente un estado, y la suma tiene que dar el numero de campos. No es
// celo: la unica forma de que un campo desapareciera sin que nadie lo notara es
// que el recorrido lo saltara, y un documento que va a una autoridad de control
// con un apartado de menos es el peor sitio para un descarte silencioso.
func (r Relleno) Cuenta() map[Estado]int {
	out := map[Estado]int{}
	for _, c := range r.Campos {
		out[c.Estado]++
	}
	return out
}

// Rellenar recorre la plantilla y resuelve cada campo con la fuente de su clase
// de origen.
//
// Las fuentes se pasan por clase y pueden faltar: rellenar el payload de una
// brecha con el incidente delante y sin el alcance es un caso normal, y lo que
// no puede pasar es que el campo del alcance salga vacio COMO SI el dato no
// existiera. Por eso hay un estado para cada cosa.
func Rellenar(p corpus.Plantilla, fuentes map[corpus.ClaseOrigen]Fuente) (Relleno, error) {
	r := Relleno{Plantilla: p.ID}
	for _, c := range p.Campos {
		o, err := corpus.ParseOrigen(c.Origen)
		if err != nil {
			// No se sigue con los demas: una plantilla con un origen ilegible no
			// se ha validado con el linter, y rellenar media plantilla y
			// entregarla es peor que no entregarla.
			return Relleno{}, fmt.Errorf("plantilla %s, campo %s: %w", p.ID, c.Nombre, err)
		}
		campo := Campo{Nombre: c.Nombre, Que: o.Que, Origen: c.Origen}
		switch {
		case !o.Derivable:
			campo.Estado = LoAportaUnaPersona
		default:
			f, hay := fuentes[o.Clase]
			if !hay || f == nil {
				campo.Estado = SinFuente
				break
			}
			// El nombre que se le pide a la fuente es el atributo en una
			// entidad y el campo en un incidente; en una obligacion es el
			// sufijo derivado sobre su patron, asi que se le pasa entero.
			v, consta := f.Campo(nombreParaLaFuente(o))
			if !consta {
				campo.Estado = NoConsta
				break
			}
			campo.Valor, campo.Estado = v, Derivado
		}
		r.Campos = append(r.Campos, campo)
	}
	return r, nil
}

// nombreParaLaFuente traduce el origen leido a la clave que la fuente entiende.
func nombreParaLaFuente(o corpus.Origen) string {
	switch o.Clase {
	case corpus.DeIncidente:
		return o.Patron
	case corpus.DeEntidad:
		return o.Sufijo
	default:
		return o.Patron + "." + o.Sufijo
	}
}

// Pendientes lista los campos que todavia no tienen valor, ordenados por
// nombre, con la frase que hay que ensenarle a quien los tiene que aportar.
//
// LA REDACCION ES PARTE DE LA FUNCION. Dice que falta y de quien es, y no dice
// ni una vez que alguien haya incumplido nada: un campo sin rellenar de una
// notificacion que todavia no se ha enviado no es un incumplimiento, es un
// documento a medias.
func (r Relleno) Pendientes() []string {
	var out []string
	for _, c := range r.Campos {
		switch c.Estado {
		case LoAportaUnaPersona:
			out = append(out, fmt.Sprintf("%s: lo escribe quien responde (%s)", c.Nombre, c.Que))
		case NoConsta:
			out = append(out, fmt.Sprintf("%s: no consta el dato (%s). Esto NO dice que no "+
				"exista: dice que en tus respuestas no aparece", c.Nombre, c.Que))
		case SinFuente:
			out = append(out, fmt.Sprintf("%s: nadie ha aportado la fuente que sabe "+
				"contestarlo (%s)", c.Nombre, c.Que))
		}
	}
	sort.Strings(out)
	return out
}
