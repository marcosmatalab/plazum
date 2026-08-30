// Package escalado decide A QUIEN se avisa, DE QUE y EN QUE ORDEN.
//
// LA FRONTERA, que es la primera decision y la que ordena el resto: aqui se
// decide; los adaptadores ENTREGAN. Email y Teams no saben nada de
// obligaciones, de figuras ni de colapso de niveles: reciben un aviso ya hecho
// y lo llevan. Al reves — un adaptador que resolviera a quien toca — el orden
// del escalado dependeria del canal, y dos canales darian dos ordenes.
//
// EL MENSAJE LLEVA LO MINIMO, Y ES TESIS DE PRODUCTO, NO ESTILO. Un aviso dice
// QUE obligacion, CUANDO vence y un ENLACE A LA INSTANCIA LOCAL. Nunca el
// contenido del incidente. Un correo por un SMTP ajeno, o un mensaje de Teams,
// con los detalles de una brecha es dato de cliente saliendo por un canal de
// terceros, que es exactamente lo que la promesa del producto dice que no pasa.
// Y no se resuelve con cuidado al redactar: se resuelve con la FORMA del tipo.
// `Aviso` no tiene ningun campo donde quepa el contenido de un incidente, asi
// que no hay redaccion descuidada que pueda meterlo. La clasificacion tampoco
// viaja: "incidente_fallecimiento" en un chat de equipo cuenta algo que nadie
// autorizo a contar.
//
// Lo unico del incidente que sale es su IDENTIFICADOR, dentro del enlace,
// porque sin el, el enlace no lleva a ningun sitio. Es la linea, y esta puesta
// a proposito donde esta: identificador si, contenido no.
//
// EL ESCALADO ES UN FLUJO DE HECHOS, como el incidente. Un escalon no tiene un
// campo "enviado" que se pisa: tiene sucesos que se anaden, y su estado es el
// ultimo. "Se notifico" y "alguien lo atendio" son DOS HECHOS DISTINTOS, y el
// acta del 9.3 va a necesitar el segundo: un envio entregado del que nadie se
// hizo cargo no es un escalon atendido.
//
// Y CADA ESCALON TERMINA EN EXACTAMENTE UN CUBO. Es la ley de conservacion otra
// vez, y aqui muerde mas que en el calendario: un envio fallido que solo se
// loguea es el `continue` mudo en forma de notificacion. Un escalado que muere
// en silencio es PEOR que no tenerlo, porque el operador cree que esta cubierto.
package escalado

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// ---------------------------------------------------------------------------
// Quien ocupa cada figura
// ---------------------------------------------------------------------------

// Asignacion dice que persona ocupa cada figura declarada por un paquete. Es
// DATO DEL CLIENTE: lo trae el alcance, de SCIM o a mano.
//
// La clave es el id de figura con su prefijo de paquete (D-18 y roles.go), asi
// que dos normas que definan figuras parecidas siguen siendo dos preguntas
// distintas. Que una misma persona ocupe las dos es normal y es cosa suya.
type Asignacion map[string]string

// Falta es una figura que un paquete declara y que nadie ocupa.
type Falta struct {
	Figura    string
	Titulo    string
	Paquete   string
	Origen    string // de la norma o propuesta por plazum: se dicen distinto
	Escalones int    // cuantos avisos se quedarian sin destinatario
}

// FigurasSinPersona es LA COMPROBACION DEL DIA UNO.
//
// La otra mitad de la alcanzabilidad. El linter ya garantiza que todo escalon
// nombra una figura que el paquete declara; esto comprueba lo que falta: que
// esa figura tenga a alguien detras EN ESTA organizacion. Sin ella, el aviso se
// descubre roto el dia del incidente, que es el unico dia en que no se puede
// arreglar.
//
// Devuelve las faltas ordenadas por numero de escalones afectados y luego por
// id, para que la lista sea estable y la mas cara salga arriba.
func FigurasSinPersona(paquetes []*corpus.Paquete, a Asignacion) []Falta {
	var out []Falta
	for _, p := range paquetes {
		usos := map[string]int{}
		for _, o := range p.Obligaciones {
			for _, e := range o.Escalado {
				usos[e.A]++
			}
		}
		for _, r := range p.Roles {
			if persona := a[r.ID]; strings.TrimSpace(persona) != "" {
				continue
			}
			out = append(out, Falta{Figura: r.ID, Titulo: r.Titulo, Paquete: p.URN,
				Origen: r.Origen, Escalones: usos[r.ID]})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Escalones != out[j].Escalones {
			return out[i].Escalones > out[j].Escalones
		}
		return out[i].Figura < out[j].Figura
	})
	return out
}

// Frase describe una falta para la pantalla, DICIENDO DE QUIEN ES LA FIGURA.
//
// Una figura que nombra la norma y una que propone plazum no se piden igual: la
// primera hay que ocuparla porque lo dice el boletin; la segunda se puede
// cambiar por otra o quitarse. Presentarlas iguales le pide al cliente el mismo
// esfuerzo para dos cosas que no lo valen igual.
func (f Falta) Frase() string {
	quien := "la propone plazum y puedes cambiarla"
	if f.Origen == corpus.FiguraDeLaNorma {
		quien = "la nombra la norma"
	}
	return fmt.Sprintf("%s (%s): nadie la ocupa, y %s. Se quedarian sin destinatario %d "+
		"aviso(s) de escalado", f.Titulo, f.Figura, quien, f.Escalones)
}

// ---------------------------------------------------------------------------
// El estado de un escalon: vocabulario CERRADO
// ---------------------------------------------------------------------------

// Estado es donde acaba un escalon. Cerrado, y con la ley de conservacion
// encima: todo escalon planificado cae en exactamente uno.
type Estado string

const (
	// Pendiente: toca mas adelante y todavia no se ha hecho nada.
	Pendiente Estado = "pendiente"
	// SinDestinatario: la figura no tiene persona en esta organizacion. NO es
	// un fallo de entrega: es un dato que falta, y se dice antes de que haga
	// falta el aviso.
	SinDestinatario Estado = "sin destinatario"
	// Colapsado: otro escalon anterior ya avisa a la misma persona. La misma
	// persona no recibe dos veces por el mismo hito.
	Colapsado Estado = "colapsado en un escalon anterior"
	// EnSilencio: una ventana de silencio lo suprimio. Se cuenta y se audita;
	// callar un aviso sin dejar rastro es el fallo que esta clase persigue.
	EnSilencio Estado = "suprimido por una ventana de silencio"
	// Enviado: entregado al canal, sin confirmacion todavia.
	Enviado Estado = "enviado al canal"
	// Entregado: el canal confirmo la entrega.
	Entregado Estado = "entregado"
	// Fallido: la entrega fallo. Se ENSENA donde el operador mira; un fallo
	// que solo se loguea es un escalado que muere en silencio, y eso es peor
	// que no tener escalado.
	Fallido Estado = "fallido en la entrega"
	// Atendido: una persona se hizo cargo. "Se notifico" y "alguien lo
	// atendio" son dos hechos distintos, y el acta del 9.3 necesita el segundo.
	Atendido Estado = "atendido"
)

// EstadosPosibles es la particion completa. Existe para que la ley de
// conservacion pueda recorrerla en vez de repetir la lista.
func EstadosPosibles() []Estado {
	return []Estado{Pendiente, SinDestinatario, Colapsado, EnSilencio,
		Enviado, Entregado, Fallido, Atendido}
}

// EsFinal dice si el estado cierra el escalon. Un escalon en un estado no final
// todavia puede moverse.
func (e Estado) EsFinal() bool {
	switch e {
	case Atendido, Fallido, SinDestinatario, Colapsado, EnSilencio:
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// El aviso: lo minimo, y por la forma del tipo
// ---------------------------------------------------------------------------

// Aviso es lo que sale de la organizacion. NO TIENE NI UN CAMPO donde quepa el
// contenido de un incidente, y eso es la garantia: no depende de que quien
// redacte se acuerde.
//
// Cada campo, con su porque:
//
//	Obligacion  el id, para poder buscarlo. Es dato publico del corpus.
//	Titulo      el titulo de la obligacion, que es texto del boletin.
//	Hito        cual de los plazos de esa obligacion.
//	Vence       cuando. Es la mitad util del aviso.
//	Figura      a quien le toca, por su figura y no por su nombre.
//	Nivel       el numero de escalon, para que se note que ya hubo otros.
//	Enlace      a la instancia LOCAL. Es lo unico que lleva a los detalles, y
//	            los detalles se ven ahi dentro, autenticado, y no en el correo.
type Aviso struct {
	Obligacion string
	Titulo     string
	Hito       string
	Vence      time.Time
	Figura     string
	Nivel      int
	Enlace     string
}

// Asunto y Cuerpo son la UNICA redaccion, y se hacen desde los campos de
// arriba. Un canal que quiera decir mas no tiene de donde sacarlo.
func (a Aviso) Asunto() string {
	return fmt.Sprintf("[plazum] vence %s: %s", a.Vence.Format("2006-01-02"), a.Titulo)
}

func (a Aviso) Cuerpo() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Obligacion: %s\n", a.Obligacion)
	fmt.Fprintf(&b, "Hito: %s\n", a.Hito)
	fmt.Fprintf(&b, "Vence: %s\n", a.Vence.Format(time.RFC3339))
	fmt.Fprintf(&b, "Te toca por la figura: %s (escalon %d)\n", a.Figura, a.Nivel)
	fmt.Fprintf(&b, "\nAbre el expediente aqui: %s\n", a.Enlace)
	b.WriteString("\nEste aviso no lleva detalles a proposito: se ven dentro de plazum.\n")
	return b.String()
}

// ---------------------------------------------------------------------------
// El plan: quien, que y en que orden
// ---------------------------------------------------------------------------

// Paso es un escalon resuelto: cuando toca, a quien, y donde ha acabado.
type Paso struct {
	Nivel   int
	Tras    string
	Cuando  time.Time
	Figura  string
	Persona string
	Estado  Estado
	// Motivo explica el estado cuando no es Pendiente. Obligatorio en todos
	// los estados que cierran el escalon sin enviar: un cubo sin motivo es una
	// etiqueta, no una explicacion.
	Motivo string
	// Aviso solo esta relleno cuando el escalon va a salir de verdad.
	Aviso *Aviso
}

// Silencio es una ventana en la que no se avisa.
//
// TRES REGLAS, Y LAS TRES SON DE PRODUCTO, NO DE FORMA:
//
//	OPT-IN. No hay silencios por defecto. Un producto cuya tesis es que no se
//	te pasa nada no puede callar avisos por su cuenta.
//
//	FIN OBLIGATORIO. Un silencio sin fecha de fin ES EL ROJO PERMANENTE CON
//	OTRO NOMBRE: se pone "hasta que arreglemos esto", nadie lo quita, y seis
//	meses despues el operador cree que no vence nada. Su caducidad DESPIERTA
//	SOLA porque es una comparacion de instantes: nadie tiene que acordarse de
//	levantarlo.
//
//	CON NOMBRE Y MOTIVO. Quien silencia y por que, para que la supresion se
//	pueda auditar. Un aviso callado sin constancia de quien lo callo es
//	indistinguible de un aviso perdido.
//
// Y la supresion NO BORRA: el escalon sigue contado, en el cubo EnSilencio y
// con su motivo. Callar sin dejar rastro es el fallo que esta clase persigue.
type Silencio struct {
	Desde, Hasta time.Time
	Motivo       string
	// Quien lo declaro. No es decoracion: es lo que hace auditable la
	// supresion cuando alguien pregunte por que no se aviso.
	Quien string
}

var (
	ErrSilencioSinFin    = errors.New("ventana de silencio sin fecha de fin")
	ErrSilencioAlReves   = errors.New("ventana de silencio que termina antes de empezar")
	ErrSilencioSinMotivo = errors.New("ventana de silencio sin quien la declara o sin motivo")
)

// Validar rechaza las ventanas que no se pueden auditar ni caducar.
func (s Silencio) Validar() error {
	switch {
	case s.Desde.IsZero() || s.Hasta.IsZero():
		return fmt.Errorf("%w: sin fin, un silencio se pone \"hasta que arreglemos esto\", "+
			"nadie lo quita, y seis meses despues el operador cree que no vence nada. "+
			"Arreglo: ponle fecha de fin, aunque sea la de manana", ErrSilencioSinFin)
	case !s.Hasta.After(s.Desde):
		return fmt.Errorf("%w: de %s a %s. Una ventana asi no cubre nada y da la falsa "+
			"sensacion de estar puesta", ErrSilencioAlReves,
			s.Desde.Format(time.RFC3339), s.Hasta.Format(time.RFC3339))
	case strings.TrimSpace(s.Quien) == "" || strings.TrimSpace(s.Motivo) == "":
		return fmt.Errorf("%w: un aviso callado sin constancia de quien lo callo y por que "+
			"es indistinguible de un aviso perdido", ErrSilencioSinMotivo)
	}
	return nil
}

func (s Silencio) cubre(t time.Time) bool {
	return !t.Before(s.Desde) && t.Before(s.Hasta)
}

// Planificar resuelve los escalones de UNA obligacion para UN vencimiento.
//
// EL ORDEN LO DECIDE EL INSTANTE, no el orden en que el paquete los escribio.
// Un paquete puede declarar `P30D_antes` despues de `P60D_antes` y el aviso de
// sesenta dias sigue siendo el primero. Ordenar por posicion seria emparejar
// por indice, que es lo que el invariante 7 prohibe en todas partes.
//
// EL COLAPSO se hace DESPUES de ordenar y mirando a los escalones anteriores en
// el tiempo: si la persona ya recibio el aviso de este hito, el segundo se
// colapsa. Colapsar por el orden del fichero mandaria dos avisos o ninguno
// segun como estuviera escrito el paquete.
func Planificar(o corpus.Obligacion, hito string, vence time.Time, reg ventana.Regimen,
	a Asignacion, silencios []Silencio, enlace func(obligacion, hito string) string,
) ([]Paso, error) {
	// UN SILENCIO MAL PUESTO NO CALLA NADA. Se rechaza al planificar y no al
	// aplicarlo: si se ignorara, una ventana sin fin se leeria como "no habia
	// silencios" y los avisos saldrian igual, que suena bien y es peor —
	// significa que la configuracion del operador no hace lo que dice.
	for i, s := range silencios {
		if err := s.Validar(); err != nil {
			return nil, fmt.Errorf("obligacion %s, ventana de silencio %d: %w", o.ID, i+1, err)
		}
	}
	var pasos []Paso
	for i, e := range o.Escalado {
		m, err := corpus.ParseTras(e.Tras)
		if err != nil {
			return nil, fmt.Errorf("obligacion %s, escalon %d: %w", o.ID, i+1, err)
		}
		cuando, _ := m.Instante(vence, reg)
		pasos = append(pasos, Paso{Tras: e.Tras, Cuando: cuando, Figura: e.A,
			Persona: a[e.A], Estado: Pendiente})
	}
	// Estable por el instante, y con el `tras` como desempate para que dos
	// escalones del mismo instante no bailen entre ejecuciones.
	sort.SliceStable(pasos, func(i, j int) bool {
		if !pasos[i].Cuando.Equal(pasos[j].Cuando) {
			return pasos[i].Cuando.Before(pasos[j].Cuando)
		}
		return pasos[i].Tras < pasos[j].Tras
	})

	yaAvisadas := map[string]int{} // persona -> nivel que la aviso
	for i := range pasos {
		p := &pasos[i]
		p.Nivel = i + 1
		switch {
		case strings.TrimSpace(p.Persona) == "":
			p.Estado = SinDestinatario
			p.Motivo = fmt.Sprintf("la figura %s no tiene a nadie asignado en este alcance. "+
				"Esto NO dice que el aviso haya fallado: dice que no hay a quien mandarlo, y "+
				"se sabe HOY y no el dia que venza", p.Figura)
		case yaAvisadas[p.Persona] > 0:
			p.Estado = Colapsado
			p.Motivo = fmt.Sprintf("la misma persona ya recibe el aviso en el escalon %d por "+
				"la figura %s, asi que este no se manda: dos avisos iguales del mismo hito no "+
				"informan mas y ensenan a ignorarlos", yaAvisadas[p.Persona], p.Figura)
		default:
			if s, hay := enSilencio(silencios, p.Cuando); hay {
				p.Estado = EnSilencio
				p.Motivo = fmt.Sprintf("cae dentro de una ventana de silencio que declaro %s "+
					"(%s a %s): %s. Caduca sola: pasado el fin, este aviso vuelve a salir "+
					"sin que nadie tenga que levantarla", s.Quien,
					s.Desde.Format(time.RFC3339), s.Hasta.Format(time.RFC3339), s.Motivo)
				break
			}
			yaAvisadas[p.Persona] = p.Nivel
			p.Aviso = &Aviso{
				Obligacion: o.ID, Titulo: o.TituloLegible(), Hito: hito, Vence: vence,
				Figura: p.Figura, Nivel: p.Nivel, Enlace: enlace(o.ID, hito),
			}
		}
	}
	return pasos, nil
}

func enSilencio(ss []Silencio, t time.Time) (Silencio, bool) {
	for _, s := range ss {
		if s.cubre(t) {
			return s, true
		}
	}
	return Silencio{}, false
}

// Cuenta reparte los pasos por estado. Es la ley de conservacion del escalado:
// la suma tiene que dar el numero de pasos, y ningun paso puede quedarse fuera.
func Cuenta(pasos []Paso) map[Estado]int {
	c := map[Estado]int{}
	for _, p := range pasos {
		c[p.Estado]++
	}
	return c
}
