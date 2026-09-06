package pantallas

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// LA PREGUNTA CON SU CONSECUENCIA AL LADO (pieza 2 de docs/ia.md).
//
// # Donde esta la friccion, y por que aqui
//
// El abandono de una entrevista de cumplimiento no se produce al principio ni al
// final: se produce en la pregunta cuya respuesta el cliente NO SABE QUE
// SIGNIFICA. «¿Prestas servicios esenciales?» es una pregunta de sí o no cuya
// respuesta afirmativa puede encender veintiocho obligaciones con sus plazos, y
// quien la contesta lo hace a ciegas. Con la consecuencia delante deja de ser una
// pregunta de tramite y pasa a ser una decision informada, que es lo unico que
// esta pantalla puede ofrecer que un formulario no.
//
// # ESTA PIEZA ES DETERMINISTA, Y ESO NO ES UN DETALLE
//
// Vive en el bloque «IA de adopcion» del plan y NO USA IA: la consecuencia la
// calcula el motor de aplicabilidad, que es el mismo que decide de verdad. Se
// dice en voz alta por tres motivos y los tres importan:
//
//   - Es COMPROBABLE. Un numero que sale de una regla Datalog se puede recontar;
//     uno que sale de un modelo, no. En una pantalla que va a cambiar la
//     respuesta de alguien sobre su propia organizacion, esa diferencia es la
//     unica que cuenta.
//   - Funciona con PLAZUM_SIN_IA=1, o sea que refuerza la puerta del invariante
//     9 en vez de gastarla. La mejor pieza de adopcion del bloque es la que no
//     necesita el modelo.
//   - Y marca donde SI hace falta la IA: en leer el PDF del cliente (piezas 1, 3
//     y 7), no en decidir que le aplica. La IA entra donde hay texto libre, no
//     donde hay reglas.
//
// # SU VALOR CERO ES NO PROMETER NADA
//
// Nil significa que esta instalacion no sabe calcular consecuencias, y entonces
// la pantalla NO pinta ninguna: una consecuencia en blanco se leeria como «esta
// respuesta no activa nada», que es una afirmacion sobre el cumplimiento de
// alguien hecha por un campo sin rellenar.
//
// LO VIGILA: TestSinCalculadoraLaPantallaNoAfirmaNadaSobreLaConsecuencia, y no
// lo vigilaba cuando este parrafo se escribio. La mutacion M12 (06-09-2026)
// dejaba el bloque encendido sin calculadora y la suite entera se quedo verde:
// el peligro estaba dicho aqui, con estas palabras, y no habia nadie mirando.
// De ahi sale la regla que ahora vigila godoc_vigilado_test.go.
type Consecuencias interface {
	// De dice que pasa si la pregunta se contesta que SI, partiendo de las
	// respuestas que se estan viendo.
	//
	// Recibe la forma cruda del formulario por lo mismo que Publicaciones: quien
	// lo implementa es el codigo que ya sabe convertir una entrevista en hechos,
	// y traducirlo aqui a un vocabulario propio seria una segunda copia de esa
	// decision. La que se quedaria vieja siempre es la de la pantalla.
	//
	// UN ERROR NO SE CONVIERTE EN CERO. Si la consecuencia no se puede calcular,
	// se devuelve el error y la pantalla NO pinta nada para esa pregunta:
	// «no lo se» y «no activa nada» son dos cosas distintas y solo una es
	// inocua.
	De(ctx context.Context, respuestas url.Values, pregunta string) (Consecuencia, error)
}

// Consecuencia es lo que se activa al contestar que si.
//
// # POR QUE VIAJAN EL NUMERO Y LOS TITULOS, Y NO SOLO EL NUMERO
//
// Porque «se te activan 9 obligaciones» es un numero que hay que creerse, y este
// producto se vende exactamente al reves. Con tres titulos delante, quien lee
// reconoce el trabajo del que le estan hablando y puede decidir; sin ellos, el 9
// es tan opaco como la pregunta que venia a aclarar.
//
// Y NO VIAJAN LAS NUEVE: una lista de nueve titulos dentro de una pregunta la
// convierte en una pagina. Van tres y el resto se cuenta, que es la misma regla
// que el resto del producto usa para no enterrar (D-13).
type Consecuencia struct {
	// N es cuantas obligaciones se activan. Cero es una respuesta valida y se
	// pinta: saber que una pregunta no activa nada es informacion, y ademas
	// evita que la ausencia de aviso se lea como un fallo.
	N int
	// Titulos son hasta TresQueSeEnsenan titulos, para que el numero se pueda
	// reconocer. Salen del corpus y viajan tal cual, sin pasar por el catalogo:
	// son texto de una norma.
	Titulos []string
	// Marcos son los marcos distintos que aportan esas obligaciones, ordenados.
	// Es lo que convierte «nueve obligaciones» en «nueve obligaciones de NIS2 y
	// del ENS», que es la forma en la que un CISO piensa el trabajo.
	Marcos []string
}

// TresQueSeEnsenan es cuantos titulos acompanan al numero.
//
// Son TRES y no cinco ni todos, y el criterio no es estetico: la pregunta tiene
// que seguir cabiendo de un vistazo. El dia que alguien quiera verlas todas, la
// puerta es la pantalla de derivacion, que ya existe y ya las lista.
const TresQueSeEnsenan = 3

// pintarConsecuencia rellena la consecuencia de UNA pregunta.
//
// # Las tres respuestas, y la del medio es la que se olvida
//
//	sin adaptador     no se pinta nada. La instalacion no sabe calcular esto.
//	calculada         se pinta, incluido el cero: «contestar que si no activa
//	                  ninguna obligacion» es informacion, y ademas evita que la
//	                  ausencia de aviso se lea como un fallo.
//	error al calcular no se pinta nada Y NO SE PINTA UN CERO. «No lo se» y «no
//	                  activa nada» son dos cosas distintas, y la segunda es una
//	                  afirmacion sobre el cumplimiento de alguien.
//
// # POR QUE EL ERROR NO SE ENSENA
//
// Porque esto es un ADORNO INFORMATIVO de una pregunta, no el contenido de la
// pantalla: un aviso rojo por cada pregunta cuya consecuencia no se pudo
// calcular convertiria la entrevista en una lista de errores y nadie la
// terminaria. Se calla en la pregunta y se cuenta por el canal de fallos de
// quien monta (Opciones.AlFallar), que es donde se mira.
func (s *Superficie) pintarConsecuencia(r *http.Request, vq *VistaPregunta, id string,
	resp Respuestas) {

	if s.consecuencias == nil {
		return
	}
	c, err := s.consecuencias.De(r.Context(), resp.Consulta(), id)
	if err != nil {
		if s.alFallar != nil {
			s.alFallar(fmt.Errorf("consecuencia de %q: %w", id, err))
		}
		return
	}
	vq.HayConsecuencia = true
	vq.Consecuencia = c
	if len(c.Titulos) > TresQueSeEnsenan {
		c.Titulos = c.Titulos[:TresQueSeEnsenan]
		vq.Consecuencia = c
	}
	// LA RESTA SE HACE AQUI Y NO EN LA PLANTILLA. Una resta en el HTML es una
	// resta que nadie prueba, y ademas html/template no sabe restar.
	if resto := c.N - len(c.Titulos); resto > 0 {
		vq.MasConsecuencias = resto
	}
}
