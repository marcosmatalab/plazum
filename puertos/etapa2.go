package puertos

import (
	"context"
	"io"
	"time"
)

// Los puertos de la etapa 2: serve, la UI generada y el autoservicio.
//
// ESTAS INTERFACES ESTAN CONGELADAS. Se definieron antes de escribir una sola
// linea de implementacion, a proposito, porque la etapa 2 se construye en
// varios frentes a la vez y todos compilan contra esto. Un cambio aqui rompe a
// los demas en silencio.
//
// Si al implementar hace falta cambiar una firma: PARAR Y PREGUNTAR. No se
// toca por cuenta propia. Lo vigila congelacion_test.go, que compara los
// conjuntos de metodos contra una lista escrita a mano: si alguien cambia una
// firma, ese test se pone rojo y dice exactamente esto.
//
// Por que tan pequenas: la regla del paquete es que un adaptador debe poder
// escribirse en una tarde y sustituirse sin tocar nada mas. Si una interfaz de
// aqui crece a mas de cuatro metodos, casi siempre es que son dos interfaces.
//
// Y por que tan POCAS. Un puerto existe para poder sustituir la implementacion,
// no para organizar el codigo. Dos candidatos se propusieron y se retiraron, y
// consta en docs/puertos-propuestas.md:
//
//	UIGenerada  la derivacion del corpus a modelo de pantalla no es una
//	            frontera de sustitucion: es una funcion pura del corpus, y no
//	            queremos dos derivaciones distintas del mismo corpus sino una
//	            comprobable. Vive en nucleo/pantalla, con casos dorados.
//	Seguridad   nada de capa de seguridad enchufable en un producto cuya tesis
//	            es que el receptor no se fia. Cabeceras y rate limit son
//	            middleware con puerta de CI contra peticiones reales; el CSRF se
//	            emite en Sesion y se aplica en middleware, con un test que
//	            enumera las rutas del router y falla si alguna ruta mutante no
//	            pasa por el.

// Servidor es la superficie HTTP. Levanta, sirve y para; nada mas.
//
// Deliberadamente NO expone enrutado: las rutas las declara quien construye el
// servidor, no quien lo arranca. Asi la superficie web se puede sustituir
// entera (html/template hoy, otra cosa manana) sin tocar a quien la arranca.
type Servidor interface {
	// Arrancar sirve hasta que el contexto se cancela. Bloquea.
	// Devuelve nil en cierre ordenado, error si no pudo ni empezar.
	Arrancar(ctx context.Context, direccion string) error
	// Parar cierra en orden, esperando como mucho lo que diga el contexto.
	Parar(ctx context.Context) error
}

// Sesion gestiona la sesion del operador y el token CSRF que la acompana.
//
// Van juntos a proposito: un CSRF que no esta atado a la sesion no protege de
// nada, y separarlos en dos puertos invita a implementarlos desacoplados. El
// diseno de la etapa 2 dice "CSRF en todo POST", y esto es lo que lo hace
// exigible.
//
// Este puerto EMITE y COMPRUEBA el token. Aplicarlo a cada peticion es trabajo
// del middleware, y su puerta es un test que enumera las rutas del router y
// falla si alguna ruta mutante no pasa por el. Emitir bien un token que nadie
// exige no protege de nada, y esa mitad no se puede vigilar desde aqui.
type Sesion interface {
	// Abrir crea sesion para un sujeto y devuelve su identificador.
	Abrir(ctx context.Context, sujeto string, duracion time.Duration) (id string, err error)
	// Leer devuelve el sujeto de una sesion viva. Error si no existe,
	// caduco o fue cerrada: el llamante no distingue, y es intencionado.
	Leer(ctx context.Context, id string) (sujeto string, err error)
	// Cerrar invalida la sesion. Idempotente.
	Cerrar(ctx context.Context, id string) error
	// TokenCSRF emite un token atado a ESTA sesion.
	TokenCSRF(ctx context.Context, id string) (token string, err error)
	// ComprobarCSRF valida el token contra la sesion. Un token de otra
	// sesion, o de una sesion ya cerrada, no vale.
	ComprobarCSRF(ctx context.Context, id, token string) error
}

// Plantilla renderiza la UI. La implementacion de referencia es html/template
// con los ficheros embebidos por go:embed y htmx vendorizado.
//
// Datos va como any y no como un tipo cerrado porque la UI de la etapa 2 se
// genera desde corpus.EsquemaUI, y el esquema no se conoce en tiempo de
// compilacion. El precio se paga en el adaptador, no aqui.
type Plantilla interface {
	// Render escribe la plantilla nombre con datos. El idioma decide el
	// catalogo de cadenas; vacio significa el idioma por defecto.
	Render(w io.Writer, nombre string, datos any, idioma string) error
}

// Catalogo traduce. El mecanismo se disena en la etapa 2 aunque solo carguen
// es y en: la promesa de aleman se recorta por escrito hasta que exista el
// partner DACH que lo revise.
//
// Traducir NUNCA falla: una cadena sin traducir devuelve la clave, para que
// falte texto en pantalla en vez de romperse la pagina. Los huecos los caza
// Faltantes, que es lo que se mira en CI.
//
// REGLA, y no es de estilo: EL CATALOGO NUNCA TRANSPORTA texto_legal. Aqui van
// las cadenas de la interfaz (rotulos, ayudas, mensajes), y nada mas. El idioma
// del corpus va por paquete, en el paquete. Traducir texto transcrito del BOE
// crea obra derivada y se sale de la estratificacion de licencias que sostiene
// todo el corpus: el BOE se puede reproducir (art. 13 TRLPI), pero una version
// nuestra en otro idioma ya no es el BOE, es obra nuestra que se presenta como
// la norma. Un paquete en aleman es un paquete distinto con su propia fuente,
// no una traduccion en tiempo de render.
type Catalogo interface {
	// Traducir devuelve la cadena para clave en idioma. Si no existe,
	// devuelve la clave tal cual.
	Traducir(idioma, clave string, args ...any) string
	// Idiomas lista los cargados, el primero es el de por defecto.
	Idiomas() []string
	// Faltantes devuelve las claves que un idioma no cubre respecto al de
	// por defecto. Vacio significa catalogo completo.
	Faltantes(idioma string) []string
}

// Actualizador actualiza el binario y los paquetes, con vuelta atras.
//
// El rollback no es opcional y por eso esta en la interfaz: una actualizacion
// que no se puede deshacer, en un producto que vigila plazos legales, convierte
// un fallo de actualizacion en un incumplimiento.
type Actualizador interface {
	// Disponible consulta si hay version nueva. Devuelve version vacia si no.
	Disponible(ctx context.Context) (version string, notas string, err error)
	// Aplicar instala una version y devuelve el identificador del punto de
	// retorno que deja preparado.
	Aplicar(ctx context.Context, version string) (puntoRetorno string, err error)
	// Deshacer vuelve a un punto de retorno.
	Deshacer(ctx context.Context, puntoRetorno string) error
}

// Diagnostico es `dutiq doctor`: responde "por que no funciona" sin que el
// operador tenga que leer codigo fuente.
//
// Cada comprobacion dice que se esperaba, que se encontro y COMO SE ARREGLA.
// Un diagnostico que solo dice "fallo" traslada el trabajo al operador, que es
// justo lo que esta pieza existe para evitar.
type Diagnostico interface {
	// Comprobar ejecuta todas las comprobaciones. No falla aunque haya
	// problemas: los problemas van en el resultado.
	Comprobar(ctx context.Context) []Comprobacion
}

// Comprobacion es el resultado de una sola comprobacion de `doctor`.
type Comprobacion struct {
	// Nombre corto e identificable, en minusculas: "keystore", "tsa", "puerto".
	Nombre string
	// Estado del resultado.
	Estado EstadoComprobacion
	// Detalle de lo que se encontro, en una linea.
	Detalle string
	// Arreglo dice que hacer. Obligatorio cuando Estado no es Correcto:
	// un problema sin arreglo escrito es un problema a medias.
	Arreglo string
}

// EstadoComprobacion es el veredicto de una comprobacion de doctor.
type EstadoComprobacion int

const (
	// Correcto: la comprobacion pasa.
	Correcto EstadoComprobacion = iota
	// Aviso: funciona, pero algo se esta perdiendo. No bloquea.
	Aviso
	// Roto: no funciona y hay que arreglarlo.
	Roto
)

func (e EstadoComprobacion) String() string {
	switch e {
	case Correcto:
		return "correcto"
	case Aviso:
		return "aviso"
	case Roto:
		return "roto"
	}
	return "desconocido"
}

// Secretos es la fuente de aleatoriedad. Es el unico puerto anadido sobre la
// lista pedida para la etapa 2, y existe por una razon concreta: en produccion
// tiene que ser crypto/rand y en los tests tiene que ser reproducible.
//
// Sin puerto, esa sustitucion se hace con una variable de paquete que alguien
// deja puesta, y entonces los identificadores de sesion del binario que se
// instala son los mismos que los del test. Con puerto, la fuente determinista
// solo existe donde se inyecta, y un adaptador que la use en produccion es un
// cambio visible en la construccion del servidor.
//
// Nunca devuelve aleatoriedad a medias: un relleno parcial es peor que un
// fallo, porque nadie lo mira y el token sale corto.
type Secretos interface {
	// Token devuelve n bytes de aleatoriedad en hexadecimal. Sirve para
	// identificador de sesion, token CSRF y token de primer admin.
	Token(n int) (string, error)
	// Bytes llena b entero. Error si no puede: nunca a medias.
	Bytes(b []byte) error
}
