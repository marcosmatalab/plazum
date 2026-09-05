package pantallas

import (
	"context"
	"net/url"
)

// LA PUBLICACION DEL ALCANCE DE LA INSTALACION, y por que es un puerto aparte
// del almacen de la cuenta.
//
// # HAY DOS ALCANCES Y NO SON EL MISMO (invariante 12)
//
// El de la CUENTA es de quien contesta la entrevista y vive en `Alcances`: se
// guarda pregunta a pregunta, sirve para que al volver mañana no haya que
// contestar otra vez, y no lo ve nadie mas.
//
// El de la INSTALACION es de quien monto plazum, y es el que alimenta al
// CALENDARIO y al PLAN DE AVISOS. Esas dos pantallas se sirven SIN SESION:
// `PasosDelCamino` las declara alcanzables y su fuente no recibe la peticion,
// asi que no pueden sacar su dato de la cuenta que mira. Darles el de una
// cuenta cualquiera -- la primera, la ultima, la del que lo publico -- seria
// publicar las respuestas de una persona a cualquiera que abra la pagina.
//
// Por eso publicar es un ACTO EXPLICITO y no un efecto de guardar: se hace al
// ADOPTAR, con su rotulo diciendo que lo que se adopta lo va a ver todo el que
// entre en esta instalacion. Un dato que cruza de un ambito privado a uno
// publico no puede cruzar en silencio.
//
// # EL LIMITE, DICHO
//
// Cualquier cuenta con sesion puede publicar, y publicar PISA lo que hubiera.
// Es una escritura, no una lectura, asi que no filtra nada de nadie; lo que
// puede pasar es que dos personas de la misma organizacion se cambien el
// calendario la una a la otra. Con una sola cuenta administradora, que es la
// instalacion de la v1, no ocurre. El dia que haya roles, esto pide uno.
//
// # SU VALOR CERO ES NO PUBLICAR
//
// Nil significa que esta instalacion no sabe publicar, y entonces la pantalla no
// pinta el rotulo que lo promete. Prometer una publicacion que no va a ocurrir
// es peor que no ofrecerla: el calendario se quedaria vacio y quien adopto
// creeria que ya esta hecho.
type Publicaciones interface {
	// Publicar toma las respuestas TAL COMO LLEGAN DEL FORMULARIO y compone con
	// ellas el alcance de la instalacion.
	//
	// Recibe la forma cruda a proposito: quien lo implementa es el mismo codigo
	// que ya sabe convertir una entrevista en alcance (el que hoy corre por
	// `plazum alcance`), y ese trabajo -- que respuesta produce que hecho, cual
	// no tiene puente, cual pide valor -- es del adaptador y no de la pantalla.
	// Traducirlo aqui a un vocabulario propio seria una segunda copia de esa
	// decision, y la que se quedaria vieja es siempre la de la pantalla.
	//
	// LO QUE ESTA SUPERFICIE SI GARANTIZA es que lo que llega ya ha pasado por
	// `De` contra el corpus instalado en la misma peticion, o sea que se ha
	// comprobado que las preguntas existen antes de guardarlas en la cuenta.
	Publicar(ctx context.Context, respuestas url.Values) error
}
