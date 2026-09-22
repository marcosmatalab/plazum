// LO QUE SE DERIVA DE UN PAQUETE sin escribir una linea por norma: los
// formularios, la entrevista de alcance, la trazabilidad hasta la plantilla, los
// conectores que hacen falta y la cobertura honesta de lo escrito.
//
// Es la propiedad que sostiene el invariante 2, y vive aparte del esquema
// porque la mueve la SUPERFICIE y no el formato.

package corpus

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Peticion es UNA norma pidiendo un dato, con la cita y la ayuda que da ELLA.
//
// Es la respuesta a "por que me piden este dato", que es la primera pregunta de
// quien rellena un formulario de cumplimiento y la unica que convierte "rellena
// esto" en trabajo que se entiende.
type Peticion struct {
	Paquete string `json:"paquete"`
	Cita    string `json:"cita"`
	Ayuda   string `json:"ayuda,omitempty"`
}

// CampoUI es un campo de formulario generado desde el modelo.
type CampoUI struct {
	Entidad  string
	Atributo string
	Etiqueta string
	Tipo     string
	Valores  []string
	Obligado bool
	Ayuda    string
	Cita     string
	Paquetes []string // que paquetes necesitan este dato
	// Peticiones dice POR QUE lo pide cada uno, una entrada por paquete DISTINTO
	// y en orden de URN. Ayuda y Cita de arriba son las de la primera, que es lo
	// que habia antes de esto y se mantiene para no romper a quien ya las lee.
	//
	// OJO: no se emparejan por indice con Paquetes. Paquetes cuenta una vez por
	// declaracion, asi que un paquete que declare la misma entidad dos veces
	// sale dos veces ahi y una sola aqui. Esa cuenta inflada es anterior a
	// Peticiones y se deja como esta a proposito, para no cambiarle la forma a
	// quien ya la lee; queda apuntada.
	Peticiones []Peticion
}

// EsquemaUI deriva los formularios de la interfaz de los paquetes instalados.
// Un atributo pedido por tres normas se pregunta una vez y se dice quien lo pide.
//
// HALLAZGO (nucleo/pantalla, caso dorado): cuando dos paquetes declaran el mismo
// atributo, el PRIMERO que se recorre fija la etiqueta, el tipo, los valores, la
// ayuda y la cita, y los demas solo suman su URN a Paquetes. Como el cargador
// recorre un directorio, ese "primero" no estaba garantizado: el mismo corpus
// daba formularios distintos entre ejecuciones, con otra ayuda y otra cita. Se
// recorre en orden de URN para que el resultado sea estable. Lo vigila
// TestElModeloNoDependeDelOrdenDeLosPaquetes, comprobado por mutacion.
//
// LA PERDIDA DE INFORMACION, ya cerrada. De las tres normas que piden el dato
// solo sobrevivia la ayuda y la cita de UNA, la de URN menor. Paquetes decia
// quienes eran, pero no por que lo pedia cada una, asi que al comprador que
// pregunta "por que me piden este dato" se le respondia con el articulo de una
// de tres, elegido por orden alfabetico. Ahora cada campo lleva Peticiones, una
// entrada por paquete con SU cita y SU ayuda.
//
// El arreglo es ADITIVO a proposito: Ayuda, Cita y Paquetes siguen exactamente
// donde estaban y significando lo mismo. Quitarlos habria roto a quien ya
// compilaba contra esta forma, y un frente no le cambia el suelo a otro.
func EsquemaUI(ps []*Paquete) []CampoUI {
	idx := map[string]*CampoUI{}
	var orden []string
	enOrden := append([]*Paquete(nil), ps...)
	sort.SliceStable(enOrden, func(i, j int) bool { return enOrden[i].URN < enOrden[j].URN })
	for _, p := range enOrden {
		for _, te := range p.Entidades {
			for _, a := range te.Atributos {
				k := te.Nombre + "." + a.Nombre
				c, ok := idx[k]
				if !ok {
					c = &CampoUI{
						Entidad: te.Nombre, Atributo: a.Nombre,
						Etiqueta: a.Nombre, Tipo: a.Tipo.String(),
						Valores: a.Valores, Obligado: a.Obligado,
						Ayuda: a.Ayuda, Cita: a.Cita,
					}
					idx[k] = c
					orden = append(orden, k)
				}
				c.Obligado = c.Obligado || a.Obligado
				c.Paquetes = append(c.Paquetes, p.URN)
				// Una peticion por PAQUETE, no por declaracion. Un paquete que
				// declare dos veces la misma entidad no puede aparecer dos
				// veces diciendo dos cosas distintas sobre el mismo dato.
				yaPide := false
				for _, x := range c.Peticiones {
					if x.Paquete == p.URN {
						yaPide = true
						break
					}
				}
				if !yaPide {
					c.Peticiones = append(c.Peticiones, Peticion{
						Paquete: p.URN, Cita: a.Cita, Ayuda: a.Ayuda,
					})
				}
			}
		}
	}
	sort.Strings(orden)
	out := make([]CampoUI, 0, len(orden))
	for _, k := range orden {
		out = append(out, *idx[k])
	}
	return out
}

// PreguntaEntrevista es una pregunta del cuestionario de alcance ya ordenada.
type PreguntaEntrevista struct {
	Pregunta
	Paquete     string
	NDesbloquea int
	// LlegaAlMotor dice si contestar esta pregunta produce algun hecho.
	//
	// SU VALOR CERO ES EL RESTRICTIVO Y ESO ES LA MITAD DE POR QUE EXISTE
	// (invariante 8). `false` significa «la respuesta no llega al motor», y de
	// eso la superficie deduce que NO PUEDE decir que contestar que si no active
	// nada. Si algun dia este campo dejara de rellenarse, todas las preguntas
	// caerian del lado que se calla, que es el inocuo. Al reves (un `true` por
	// defecto) la pantalla absolveria a todo el mundo por olvido.
	//
	// Es `false` exactamente cuando el atributo de la pregunta declara
	// PuenteNoLlegaAlMotor, que es la forma en la que un paquete dice «esta
	// respuesta no alimenta ninguna regla» (ver puente.go). Tambien es `false`
	// si el atributo no aparece en la entidad, que el linter no deja pasar pero
	// que aqui se trata como la nada peligrosa y no como un si.
	LlegaAlMotor bool
}

// llegaAlMotor contesta si la respuesta a esta pregunta produce algun hecho.
//
// Se resuelve por (entidad, atributo), que son los dos campos que la pregunta
// declara y que el linter ya obliga a que existan. NO se empareja por posicion
// ni por orden (invariante 7): el mismo nombre de atributo puede aparecer en
// entidades distintas del mismo paquete, y casar por indice haria que insertar
// una entidad cambiara la respuesta de otra pregunta sin tocarla.
//
// La forma de la nada, dicha: si la entidad o el atributo no aparecen, se
// devuelve `false`, que es «no llega». Es un corpus roto que el linter rechaza
// antes (ErrPreguntaSinAtributo), pero si alguna vez llegara aqui, la respuesta
// segura es callarse y no la de absolver.
func llegaAlMotor(p *Paquete, q Pregunta) bool {
	for _, e := range p.Entidades {
		if e.Nombre != q.Entidad {
			continue
		}
		for _, a := range e.Atributos {
			if a.Nombre != q.Atributo {
				continue
			}
			return a.Hecho != nil && a.Hecho.Forma != PuenteNoLlegaAlMotor
		}
	}
	return false
}

// Entrevista construye el cuestionario de alcance: la union de las preguntas de
// los paquetes instalados, ordenada por cuantas obligaciones desbloquea cada una.
// Nunca se ensena un catalogo de controles en frio.
func Entrevista(ps []*Paquete) []PreguntaEntrevista {
	var out []PreguntaEntrevista
	for _, p := range ps {
		for _, q := range p.Preguntas {
			out = append(out, PreguntaEntrevista{
				Pregunta: q, Paquete: p.URN, NDesbloquea: len(q.Desbloquea),
				LlegaAlMotor: llegaAlMotor(p, q),
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].NDesbloquea != out[j].NDesbloquea {
			return out[i].NDesbloquea > out[j].NDesbloquea
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// Traza es la trazabilidad obligacion -> entregable -> campo, que es lo que
// convierte un generador de plantillas en una herramienta que implanta.
type Traza struct {
	Obligacion string
	Plantilla  string
	Campo      string
	Origen     string
}

// Trazabilidad devuelve el mapa completo. Si esta vacio para una obligacion que
// declara entregable, el linter ya lo habra rechazado.
func Trazabilidad(ps []*Paquete) []Traza {
	var out []Traza
	for _, p := range ps {
		pl := map[string]Plantilla{}
		for _, t := range p.Plantillas {
			pl[t.ID] = t
		}
		for _, o := range p.Obligaciones {
			if o.Entregable == "" {
				continue
			}
			for _, c := range pl[o.Entregable].Campos {
				out = append(out, Traza{o.ID, o.Entregable, c.Nombre, c.Origen})
			}
		}
	}
	return out
}

// NecesidadRecurso dice cuantas obligaciones dependen de un tipo de recurso.
// Es la respuesta a "que conector construyo primero" con un numero detras, en
// vez de con una intuicion.
type NecesidadRecurso struct {
	Recurso      TipoRecurso
	Obligaciones int
	Normas       []string
}

// Conectores ordena los tipos de recurso por cuantas obligaciones desbloquean.
func Conectores(ps []*Paquete) []NecesidadRecurso {
	n := map[TipoRecurso]*NecesidadRecurso{}
	for _, p := range ps {
		vistos := map[TipoRecurso]bool{}
		for _, o := range p.Obligaciones {
			for _, r := range o.Recursos {
				x, ok := n[r]
				if !ok {
					x = &NecesidadRecurso{Recurso: r}
					n[r] = x
				}
				x.Obligaciones++
				if !vistos[r] {
					x.Normas = append(x.Normas, p.URN)
					vistos[r] = true
				}
			}
		}
	}
	out := make([]NecesidadRecurso, 0, len(n))
	for _, x := range n {
		out = append(out, *x)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Obligaciones != out[j].Obligaciones {
			return out[i].Obligaciones > out[j].Obligaciones
		}
		return out[i].Recurso < out[j].Recurso
	})
	return out
}

// Cobertura es lo que se publica en COBERTURA.md. Un proyecto que publica lo que
// le falta es mas creible que uno que publica un porcentaje.
type Cobertura struct {
	Paquete        string
	Total          int
	ConEntregable  int
	ConRecurso     int
	Delegadas      int
	SinAutomatizar []string
}

// Medir calcula la cobertura sin redondear a favor.
func Medir(p *Paquete) Cobertura {
	c := Cobertura{Paquete: p.URN, Total: len(p.Obligaciones)}
	for _, o := range p.Obligaciones {
		if o.Entregable != "" {
			c.ConEntregable++
		}
		if len(o.Recursos) > 0 {
			c.ConRecurso++
		}
		if o.Delegado != "" {
			c.Delegadas++
		}
		if len(o.Recursos) == 0 && o.Delegado == "" {
			c.SinAutomatizar = append(c.SinAutomatizar, o.ID)
		}
	}
	return c
}

func (c Cobertura) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %d obligaciones\n", c.Paquete, c.Total)
	fmt.Fprintf(&b, "  con entregable documental   %d\n", c.ConEntregable)
	fmt.Fprintf(&b, "  con recurso observable      %d\n", c.ConRecurso)
	fmt.Fprintf(&b, "  delegadas a herramienta     %d\n", c.Delegadas)
	fmt.Fprintf(&b, "  sin automatizar             %d", len(c.SinAutomatizar))
	if len(c.SinAutomatizar) > 0 {
		fmt.Fprintf(&b, "  %s", strings.Join(c.SinAutomatizar, ", "))
	}
	return b.String()
}

// InicioDeVigencia es el instante desde el que la obligacion obliga, ya cruzada
// con la vigencia de su norma. Es el `desde` de la interseccion, el mismo dato
// que EnVigor usa para decir si o no.
//
// EXISTE PARA QUE UNA OBLIGACION QUE TODAVIA NO OBLIGA PUEDA DECIR CUANDO
// EMPEZARA. EnVigor devuelve un booleano, y con un booleano lo unico que puede
// hacer quien pinta una pantalla es esconder la fila. Esconderla es lo que hacia
// el calendario, y es un fallo del mismo tipo que el que documenta VigentesEn
// unas lineas mas abajo: una obligacion que desaparece sin explicacion se lee
// como un fallo del producto. Para una norma que empieza a aplicarse DENTRO de
// la ventana que estas mirando, es peor todavia, porque la fila que falta es
// justo la unica noticia del calendario.
func (p *Paquete) InicioDeVigencia(o Obligacion) (time.Time, error) {
	rp, err := p.Vigencia.interpretar()
	if err != nil {
		return time.Time{}, fmt.Errorf("paquete %s: %w", p.URN, err)
	}
	ro, err := o.Vigencia.interpretar()
	if err != nil {
		return time.Time{}, fmt.Errorf("paquete %s, obligacion %s: %w", p.URN, o.ID, err)
	}
	x := rp.interseccion(ro)
	if x.sinInicio {
		return time.Time{}, fmt.Errorf("%w: paquete %s, obligacion %s", ErrVigenciaSinDesde, p.URN, o.ID)
	}
	return x.desde, nil
}

// FinDeVigencia es el instante en que la obligacion DEJA de obligar, ya cruzado
// con la vigencia de su norma. El booleano dice si hay fin: una vigencia abierta
// por arriba (la mayoria) no lo tiene, y ahi el valor cero de time.Time seria
// "el ano 1", que es justo la lectura peligrosa.
//
// SE DEVUELVE UN BOOLEANO Y NO UN CERO CENTINELA a proposito. Es el invariante 8
// en una frontera pequena: un time.Time cero comparado con "esta antes de hoy?"
// responde que si, asi que una vigencia abierta se leeria como una obligacion
// ya derogada. La forma que sale por olvidarse tiene que ser la que no compila.
func (p *Paquete) FinDeVigencia(o Obligacion) (time.Time, bool, error) {
	rp, err := p.Vigencia.interpretar()
	if err != nil {
		return time.Time{}, false, fmt.Errorf("paquete %s: %w", p.URN, err)
	}
	ro, err := o.Vigencia.interpretar()
	if err != nil {
		return time.Time{}, false, fmt.Errorf("paquete %s, obligacion %s: %w", p.URN, o.ID, err)
	}
	x := rp.interseccion(ro)
	if x.abierta {
		return time.Time{}, false, nil
	}
	return x.hasta, true, nil
}

// periodicidadEnElTexto devuelve la palabra con la que un texto legal declara
// que algo se repite, o "" si no la hay.
//
// Se usa para que una obligacion no pueda declararse `continua` cuando su
// propia transcripcion dice lo contrario. La lista es corta y literal a
// proposito: no pretende clasificar, solo cazar la contradiccion evidente entre
// lo que dice el boletin y lo que dice el paquete.
func periodicidadEnElTexto(texto string) string {
	bajo := sinTildesMinusculas(texto)
	for _, p := range []string{
		"periodicamente", "periodicas", "periodicos", "periodica", "periodico",
		"regularmente", "con regularidad", "anualmente", "trimestral", "semestral",
		"a intervalos planificados", "una vez al ano",
	} {
		if nombraA(bajo, p) {
			return p
		}
	}
	return ""
}
