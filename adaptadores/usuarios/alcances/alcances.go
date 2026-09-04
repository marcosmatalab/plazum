// Package alcances guarda las respuestas de la entrevista de cada cuenta: es
// la PRIMERA ESCRITURA DE ESTADO del producto que no es un ledger.
//
// # Que faltaba, y por que el hueco era el que era
//
// Hasta hoy las respuestas de /alcance viajaban en la direccion de la pagina y
// se perdian al cerrar el navegador. La pantalla lo decia con esas palabras
// («todavia no se guardan en ningun sitio»), y decirlo era lo correcto mientras
// no hubiera donde guardarlas: `docs/instantanea.md` dejo escrito que «un boton
// que no guarda seria la peor mentira de esa pantalla». Con el almacen de
// cuentas construido, el alcance pasa a ser ESTADO DE LA CUENTA, y esto es
// donde vive.
//
// # Lo que este paquete promete, y lo que no
//
// Promete: que lo que una cuenta guarda no lo lee ni lo pisa otra; que un
// fichero que no se entiende NO se lee como «esta cuenta no ha contestado
// nada»; que dos pestanas contestando preguntas distintas a la vez conservan
// las dos respuestas; y que la escritura es atomica, asi que un corte de
// corriente no deja el fichero a medias.
//
// NO promete ser prueba de nada. Esto no es el ledger ni el expediente: no va
// firmado, no va encadenado y no es evidencia. Es lo que la persona contesto
// sobre su propia organizacion, guardado para que no tenga que volver a
// contestarlo manana. El dia que una respuesta tenga que ser oponible a un
// auditor, el sitio es el expediente y no este fichero, y se dice aqui para que
// nadie lo confunda leyendo el codigo.
//
// # LAS TRES FORMAS DE LA NADA, otra vez, porque son tres
//
// Es el invariante 8, y aqui aparece dos veces: al abrir el fichero y al leer
// CADA respuesta de dentro.
//
//	fichero AUSENTE           nadie ha guardado nada todavia. Almacen vacio y
//	                          sin error: es el unico caso en que la nada es de
//	                          verdad.
//	fichero PRESENTE Y VACIO  error. Un fichero de cero bytes donde deberia
//	                          haber respuestas es una escritura cortada o un
//	                          `> alcances.json` de alguien, y leerlo como «esta
//	                          cuenta no ha contestado» borra el trabajo de una
//	                          persona sin decirselo.
//	PRESENTE Y NO
//	INTERPRETABLE             error, siempre. Una version que no conocemos, una
//	                          respuesta que no es `si` ni `no`, un instante que
//	                          no es RFC3339: son datos que HAY y no se
//	                          entienden, y tomarlos por la nada es inventarse un
//	                          valor.
//
// La tercera es la que de verdad muerde en este fichero. Una respuesta vale
// `si` o `no` y NADA MAS: el valor cero de Respuesta es `Ninguna`, que en disco
// esta PROHIBIDO. Un `respuesta: ""` no es «sin responder», porque «sin
// responder» se escribe no teniendo fila; es un dato roto, y se dice.
package alcances

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/marcosmatalab/plazum/adaptadores/usuarios"
)

// VersionDelAlmacen es la unica que este binario lee y escribe.
//
// SE EXIGE Y NO SE SUPONE, igual que `usuarios.VersionDelAlmacen` y que
// `incidente.VersionDelRegistro`. Un fichero sin version se leeria hoy con las
// reglas de hoy y dentro de un ano con las de entonces, en silencio y sobre el
// mismo contenido.
const VersionDelAlmacen = 1

// NombreDelFichero es como se llama el almacen dentro del directorio de datos.
//
// SE LLAMA `respuestas.json` Y NO `alcances.json`, aunque el paquete se llame
// alcances, y no es un descuido. El producto YA tiene un fichero que se llama
// `alcance.json`: el de `plazum serve --alcance`, que son los HECHOS derivados
// de una organizacion y lo lee el calendario. Dos ficheros llamados casi igual,
// con contenidos distintos y en el mismo directorio, es una llamada de soporte
// esperando a ocurrir; y el operador que restaure el equivocado desde su copia
// de seguridad no se va a enterar de que ha restaurado el otro.
//
// El nombre de dentro del codigo puede ser el del concepto; el de fuera tiene
// que ser el que no se confunde.
const NombreDelFichero = "respuestas.json"

const (
	// MaxRespuestasPorCuenta acota lo que una cuenta puede guardar. No lo
	// alcanza nadie desde la interfaz, que solo admite ids de pregunta que el
	// corpus instalado declara; existe porque el fichero se puede editar a
	// mano, y un fichero de cien megabytes se lee entero en cada peticion.
	MaxRespuestasPorCuenta = 5000
	// MaxLongitudDeID acota el identificador de pregunta. Los del corpus miden
	// decenas de caracteres.
	MaxLongitudDeID = 200
)

// Los centinelas. Cada uno se comprueba con errors.Is desde quien monta.
var (
	// ErrAlmacenIlegible: el fichero existe y no es lo que dice ser.
	ErrAlmacenIlegible = errors.New("el almacen de alcances no se puede leer")
	// ErrAlmacenVacio: el fichero existe y no tiene ni un byte. ES UN ERROR y
	// no «aqui no hay respuestas»: ver el encabezado del paquete.
	ErrAlmacenVacio = errors.New("el almacen de alcances existe y esta vacio")
	// ErrAlmacenSinVersion: falta la version del formato.
	ErrAlmacenSinVersion = errors.New("el almacen de alcances no dice su version")
	// ErrVersionDesconocida: una version que este binario no sabe leer.
	ErrVersionDesconocida = errors.New("version del almacen de alcances desconocida")
	// ErrRespuestaNoInterpretable: hay un valor y no se entiende.
	ErrRespuestaNoInterpretable = errors.New("respuesta no interpretable")
	// ErrSinRuta: se pidio abrir un almacen sin decir donde vive.
	ErrSinRuta = errors.New("almacen de alcances sin ruta")
	// ErrSinUsuario: se pidio leer o escribir el alcance de nadie.
	//
	// ES UN ERROR Y NO UNA CUENTA VACIA, y es la guarda que impide que una
	// peticion sin sesion escriba en un cajon comun que despues leeria
	// cualquiera. Un alcance sin dueno no es de nadie: es de todos.
	ErrSinUsuario = errors.New("alcance sin usuario")
	// ErrPreguntaNoValida: el identificador de pregunta no sirve.
	ErrPreguntaNoValida = errors.New("identificador de pregunta no valido")
	// ErrDemasiadasRespuestas: una cuenta pasa del tope.
	ErrDemasiadasRespuestas = errors.New("demasiadas respuestas para una cuenta")
)

// Respuesta es lo que consta contestado a una pregunta de la entrevista.
//
// EL VALOR CERO ES INVALIDO A PROPOSITO. En este almacen no existe «sin
// responder»: no responder se escribe NO TENIENDO FILA. Si el cero significara
// «sin responder», un campo ausente, uno vacio y uno con basura dentro darian
// los tres el mismo resultado, que es exactamente el fallo del `Atoi` con el
// error descartado.
type Respuesta uint8

const (
	// Ninguna es el cero y NO SE ESCRIBE NUNCA en disco.
	Ninguna Respuesta = iota
	// Si y No son las dos unicas respuestas que la entrevista admite.
	Si
	No
)

// String da el valor que viaja al fichero. El cero devuelve cadena vacia, que
// es lo que hace que escribirlo se note en vez de colarse.
func (r Respuesta) String() string {
	switch r {
	case Si:
		return "si"
	case No:
		return "no"
	}
	return ""
}

// LeerRespuesta interpreta lo que trae el fichero o el formulario.
//
// LOS TRES CASOS, y el tercero es un error y no un valor por defecto: `""`,
// `"quizas"` y `"SI "` son datos que hay y que no se entienden. El unico que
// no llega aqui es la ausencia de fila, que es la nada de verdad.
func LeerRespuesta(v string) (Respuesta, error) {
	switch v {
	case "si":
		return Si, nil
	case "no":
		return No, nil
	}
	return Ninguna, fmt.Errorf("%w: %q no es ni %q ni %q ni %q. En este almacen no existe "+
		"«sin responder» con un valor: sin responder es no tener fila",
		ErrRespuestaNoInterpretable, recortar(v), "si", "no", EtiquetaDeValorEnDisco)
}

// ErrContestacionNoValida: una contestacion que no es exactamente una de las
// dos formas.
var ErrContestacionNoValida = errors.New("contestacion no valida")

// MaxLongitudDeValor acota el valor de una respuesta. Los del corpus son
// etiquetas cortas (ALTA, MEDIA, un numero), no prosa.
const MaxLongitudDeValor = 200

// Contestacion es lo que consta contestado a una pregunta, en cualquiera de las
// DOS formas que la entrevista admite.
//
// # Por que hizo falta, con su cardinal
//
// Hasta el 04-09-2026 aqui solo cabia un si o un no, y la entrevista pregunta
// valores desde ese mismo dia. Medido sobre el corpus real: **35 de las 68
// preguntas se contestan con un valor**. O sea que la mitad larga de la
// entrevista no cabia en la cuenta, y quien la respondiera entera en el
// navegador se llevaba guardada poco mas de la mitad.
//
// Eso no era solo una perdida: era la UNICA forma que tenia este producto de
// producir un alcance corto SIN QUE NINGUN CARDINAL LO DIJERA. Las respuestas
// que no caben no salen como descarte ni como cubo, salen como AUSENTES, y
// ausente es una respuesta legitima. Un alcance al que le faltan 35 preguntas
// son obligaciones que no aparecen en el calendario de un cliente sin que nada
// avise.
//
// # EL VALOR CERO ES INVALIDO, y «las dos a la vez» tambien
//
// Exactamente una de las dos formas. Ni ninguna (que es el cero de la
// estructura, y es el que sale por olvido) ni las dos, que seria una fila que
// dice dos cosas y en la que mandaria el orden en que alguien la lea.
//
// Las dos comprobaciones viven en Valida() y NO se repiten en cada llamante: una
// segunda implementacion de la misma regla es como se consigue que dos esten de
// acuerdo y la que mande sea la otra.
type Contestacion struct {
	// Booleana es Si o No. Ninguna (el cero) significa «esta no es la forma».
	Booleana Respuesta
	// Valor es la respuesta con valor. La cadena vacia significa «esta no es la
	// forma», y por eso un valor vacio NO es un valor: es un dato que hay y no
	// se entiende, y sale por error. Ver el tercer caso del encabezado.
	Valor string
}

// Booleana construye la contestacion de un si o un no.
func Booleana(r Respuesta) Contestacion { return Contestacion{Booleana: r} }

// ConValor construye la contestacion de un valor.
func ConValor(v string) Contestacion { return Contestacion{Valor: v} }

// EsValor dice si esta contestacion es de la forma con valor.
func (c Contestacion) EsValor() bool { return c.Valor != "" }

// Valida comprueba que es exactamente una de las dos formas.
//
// LAS TRES FORMAS DE NO SERLO, y ninguna se degrada a un valor por defecto:
//
//	{Ninguna, ""}      el valor cero. Es el que sale por olvido, y por eso es el
//	                   que mas importa que no pase.
//	{Si, "ALTA"}       dice dos cosas. Cual manda dependeria de quien la lea.
//	{Ninguna, "  "}    un valor que solo tiene espacios no es un valor.
func (c Contestacion) Valida() error {
	tieneBool := c.Booleana == Si || c.Booleana == No
	tieneValor := strings.TrimSpace(c.Valor) != ""
	switch {
	case tieneBool && tieneValor:
		return fmt.Errorf("%w: llega con la respuesta %q Y con el valor %q a la vez. "+
			"Cual manda dependeria de quien la lea", ErrContestacionNoValida,
			c.Booleana, recortar(c.Valor))
	case tieneBool:
		return nil
	case tieneValor:
		if len(c.Valor) > MaxLongitudDeValor {
			return fmt.Errorf("%w: el valor mide %d caracteres y el tope son %d",
				ErrContestacionNoValida, len(c.Valor), MaxLongitudDeValor)
		}
		return nil
	case c.Booleana != Ninguna:
		return fmt.Errorf("%w: la respuesta %d no es ni «si» ni «no»",
			ErrContestacionNoValida, c.Booleana)
	case c.Valor != "":
		// Presente y no interpretable: hay algo escrito y no significa nada.
		return fmt.Errorf("%w: el valor %q solo tiene espacios. Un valor en blanco no es "+
			"«sin responder»: sin responder es no tener fila",
			ErrContestacionNoValida, recortar(c.Valor))
	}
	return fmt.Errorf("%w: no trae ni respuesta ni valor. En este almacen no existe «sin "+
		"responder»: sin responder es no tener fila", ErrContestacionNoValida)
}

// String da una forma legible para los mensajes de error. NO es lo que viaja al
// fichero: eso lo compone guardar(), que escribe los dos campos por separado.
func (c Contestacion) String() string {
	if c.EsValor() {
		return "valor " + recortar(c.Valor)
	}
	return c.Booleana.String()
}

// EtiquetaDeValorEnDisco es lo que va en el campo `respuesta` de una fila con
// valor. No puede ser «si» ni «no», y no se deja vacio: un campo vacio es
// exactamente lo que LeerRespuesta rechaza, y aqui hace falta que la fila diga
// de que forma es ANTES de mirar el otro campo.
const EtiquetaDeValorEnDisco = "valor"

// Alcance son las respuestas de UNA cuenta, ya validadas.
type Alcance struct {
	// Usuario es el nombre canonico de la cuenta.
	Usuario string
	// Actualizado es cuando se escribio por ultima vez. Cero cuando la cuenta
	// no ha guardado nada nunca, y ese cero SI significa la nada: no viene de
	// un campo que no se entendiera, viene de no haber fila.
	Actualizado time.Time
	// Respuestas van por id de pregunta. Cada una es una Contestacion valida:
	// nunca el valor cero, nunca las dos formas a la vez.
	Respuestas map[string]Contestacion
}

// Copia devuelve un alcance independiente. Se usa al salir del candado: si se
// devolviera el mapa de dentro, quien lo recibe podria escribirlo desde otra
// gorutina y el detector de carreras lo diria tarde, en la peticion de alguien.
func (a Alcance) Copia() Alcance {
	otra := Alcance{Usuario: a.Usuario, Actualizado: a.Actualizado,
		Respuestas: make(map[string]Contestacion, len(a.Respuestas))}
	for k, v := range a.Respuestas {
		otra.Respuestas[k] = v
	}
	return otra
}

// --- la forma en disco ---

type respuestaEnDisco struct {
	Pregunta  string `json:"pregunta"`
	Respuesta string `json:"respuesta"`
	// Valor solo va cuando Respuesta es EtiquetaDeValorEnDisco, y entonces es
	// OBLIGATORIO. Se omite en las filas de si/no para que un fichero escrito
	// por este binario y que no use valores sea byte a byte el de antes.
	Valor string `json:"valor,omitempty"`
}

type alcanceEnDisco struct {
	Usuario     string             `json:"usuario"`
	Actualizado string             `json:"actualizado"`
	Respuestas  []respuestaEnDisco `json:"respuestas"`
}

type almacenEnDisco struct {
	Version  int              `json:"version"`
	Alcances []alcanceEnDisco `json:"alcances"`
}

// Almacen es el conjunto de alcances de una instalacion, respaldado por un
// fichero. Seguro para uso concurrente: es lo que atiende las peticiones HTTP.
type Almacen struct {
	mu    sync.Mutex
	ruta  string
	ahora func() time.Time
	// porUsuario va por el nombre CANONICO de la cuenta, el que devuelve
	// usuarios.NormalizarUsuario. Casar por el nombre crudo dejaria «CISO» y
	// «ciso» con dos cajones distintos, y quien entrara escribiendolo de la
	// otra forma veria su entrevista en blanco.
	porUsuario map[string]Alcance
}

// Opciones para abrir un almacen.
type Opciones struct {
	// Ruta del fichero. Obligatoria: un almacen sin ruta se pierde al parar el
	// proceso, y unas respuestas que hay que volver a teclear en cada arranque
	// no estan guardadas.
	Ruta string
	// Reloj para la marca de actualizacion. Nil es time.Now.
	Reloj func() time.Time
}

// Abrir lee el almacen del disco, o devuelve uno vacio si el fichero no existe.
// Ver el encabezado del paquete para las tres formas de la nada.
func Abrir(o Opciones) (*Almacen, error) {
	ruta := strings.TrimSpace(o.Ruta)
	if ruta == "" {
		return nil, fmt.Errorf("%w. Arreglo para quien monta: pasa Opciones.Ruta con el "+
			"fichero donde viven las respuestas de esta instalacion", ErrSinRuta)
	}
	a := &Almacen{ruta: ruta, ahora: o.Reloj, porUsuario: map[string]Alcance{}}
	if a.ahora == nil {
		a.ahora = time.Now
	}

	// #nosec G304 -- la ruta la elige el operador con --alcances en su propia
	// maquina, igual que --usuarios y --datos.
	b, err := os.ReadFile(ruta)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		// LA NADA DE VERDAD: nadie ha guardado nada. Sin error y sin aviso.
		return a, nil
	case err != nil:
		return nil, fmt.Errorf("%w: no puedo leer %s: %w.\n"+
			"Arreglo: comprueba que el fichero es legible por el usuario que ejecuta "+
			"plazum, o elige otro sitio con --alcances", ErrAlmacenIlegible, ruta, err)
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return nil, fmt.Errorf("%w: %s tiene %d bytes.\n"+
			"NO se lee como «nadie ha contestado todavia»: si se leyera asi, plazum "+
			"ensenaria la entrevista en blanco a quien ya la habia respondido y le diria, "+
			"sin decirlo, que su trabajo no existio.\n"+
			"Arreglo: restaura el fichero de tu copia de seguridad. Si de verdad no hay "+
			"nada que conservar, borralo (no lo vacies) y vuelve a arrancar",
			ErrAlmacenVacio, ruta, len(b))
	}

	var doc almacenEnDisco
	// DisallowUnknownFields no se usa aqui a proposito, igual que en el almacen
	// de usuarios: un campo de mas de una version futura tiene que poder
	// ignorarse. Lo que NO se ignora es un campo de menos ni uno que no se
	// entienda, que es lo que comprueba alcanceDeDisco.
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("%w: %s no es un almacen de alcances en JSON: %w.\n"+
			"Arreglo: restauralo de tu copia de seguridad, o borralo si prefieres volver a "+
			"responder la entrevista", ErrAlmacenIlegible, ruta, err)
	}
	if doc.Version == 0 {
		return nil, fmt.Errorf("%w: %s no dice con que version del formato se escribio. Sin "+
			"ella se leeria hoy con las reglas de hoy y manana con las de manana, en "+
			"silencio. Arreglo: si el fichero es de plazum, restauralo de tu copia; si lo "+
			"escribio otra cosa, no es un almacen de plazum", ErrAlmacenSinVersion, ruta)
	}
	if doc.Version != VersionDelAlmacen {
		return nil, fmt.Errorf("%w: %s dice version %d y este plazum lee la %d.\n"+
			"Arreglo: si el fichero es mas nuevo, actualiza plazum; si es mas viejo, usa la "+
			"version de plazum que lo escribio para migrarlo",
			ErrVersionDesconocida, ruta, doc.Version, VersionDelAlmacen)
	}

	for i, c := range doc.Alcances {
		leido, err := alcanceDeDisco(ruta, i, c)
		if err != nil {
			return nil, err
		}
		if _, repetido := a.porUsuario[leido.Usuario]; repetido {
			return nil, fmt.Errorf("%w: %s trae dos alcances de la cuenta %q. Con dos, cual "+
				"manda depende del orden del fichero, y el orden no lo firma nadie",
				ErrAlmacenIlegible, ruta, leido.Usuario)
		}
		a.porUsuario[leido.Usuario] = leido
	}
	return a, nil
}

// alcanceDeDisco valida el alcance de una cuenta. TODO campo que falte, no se
// entienda o se salga de rango es un error, nunca un valor por defecto.
func alcanceDeDisco(ruta string, i int, c alcanceEnDisco) (Alcance, error) {
	donde := fmt.Sprintf("%s, alcance %d", ruta, i+1)
	nombre, err := usuarios.NormalizarUsuario(c.Usuario)
	if err != nil {
		return Alcance{}, fmt.Errorf("%w: %s: %w", ErrAlmacenIlegible, donde, err)
	}
	// EL INSTANTE QUE NO SE ENTIENDE ES UN ERROR, NUNCA EL CERO. El cero de
	// time.Time es el ano 1, y de ahi salen marcas de guardado con cara de dato.
	actualizado, err := time.Parse(time.RFC3339, strings.TrimSpace(c.Actualizado))
	if err != nil {
		return Alcance{}, fmt.Errorf("%w: %s no trae un instante de actualizacion RFC3339 "+
			"(2026-09-04T09:00:00Z)", ErrAlmacenIlegible, donde)
	}
	if len(c.Respuestas) > MaxRespuestasPorCuenta {
		return Alcance{}, fmt.Errorf("%w: %s trae %d respuestas y el tope son %d",
			ErrDemasiadasRespuestas, donde, len(c.Respuestas), MaxRespuestasPorCuenta)
	}
	out := Alcance{Usuario: nombre, Actualizado: actualizado.UTC(),
		Respuestas: make(map[string]Contestacion, len(c.Respuestas))}
	for j, r := range c.Respuestas {
		id, err := NormalizarPregunta(r.Pregunta)
		if err != nil {
			return Alcance{}, fmt.Errorf("%w: %s, respuesta %d: %w",
				ErrAlmacenIlegible, donde, j+1, err)
		}
		valor, err := contestacionDeDisco(r)
		if err != nil {
			return Alcance{}, fmt.Errorf("%w: %s, respuesta %d (%s): %w",
				ErrAlmacenIlegible, donde, j+1, id, err)
		}
		if anterior, repetida := out.Respuestas[id]; repetida && anterior != valor {
			// DOS RESPUESTAS DISTINTAS A LA MISMA PREGUNTA NO SE RESUELVEN
			// ELIGIENDO UNA. Cual gana dependeria del orden del fichero, y el
			// orden no lo firma nadie (invariante 7). La contradiccion de la
			// direccion de la pagina si tiene tratamiento, porque alli la
			// escribe quien navega; aqui la escribiria quien edito el fichero.
			return Alcance{}, fmt.Errorf("%w: %s trae la pregunta %s respondida %q y %q. "+
				"Cual manda dependeria del orden del fichero",
				ErrAlmacenIlegible, donde, id, anterior, valor)
		}
		out.Respuestas[id] = valor
	}
	return out, nil
}

// contestacionDeDisco interpreta UNA fila del fichero.
//
// LOS DOS DESACUERDOS ENTRE LOS DOS CAMPOS SON ERROR, y ninguno se resuelve
// eligiendo un campo:
//
//	respuesta:"si" + valor:"ALTA"   la fila dice dos cosas
//	respuesta:"valor" + valor:""    la fila se anuncia con valor y no lo trae.
//	                                Es «presente y no interpretable», que es la
//	                                tercera hermana del invariante 8 y la que
//	                                sale por descuido: tomarla por «sin
//	                                responder» seria inventarse una ausencia.
//
// Una fila de si/no escrita por un binario ANTERIOR a los valores no trae el
// campo, asi que llega vacio y pasa por el primer camino sin tocar nada. Es lo
// que hace que los ficheros de ayer se sigan leyendo.
func contestacionDeDisco(r respuestaEnDisco) (Contestacion, error) {
	if r.Respuesta == EtiquetaDeValorEnDisco {
		c := ConValor(r.Valor)
		if err := c.Valida(); err != nil {
			return Contestacion{}, fmt.Errorf("se anuncia como %q y %w",
				EtiquetaDeValorEnDisco, err)
		}
		return c, nil
	}
	v, err := LeerRespuesta(r.Respuesta)
	if err != nil {
		return Contestacion{}, err
	}
	if r.Valor != "" {
		return Contestacion{}, fmt.Errorf("%w: viene respondida %q Y con el valor %q. "+
			"Cual manda dependeria de quien la lea", ErrRespuestaNoInterpretable,
			r.Respuesta, recortar(r.Valor))
	}
	return Booleana(v), nil
}

// NormalizarPregunta valida un identificador de pregunta.
//
// NO se comprueba contra el corpus, y es deliberado: este paquete es un
// adaptador de almacenamiento y no conoce ningun corpus (invariante 2). Quien
// decide si una pregunta EXISTE es la superficie, que la mira contra el corpus
// instalado; lo de aqui es solo que el identificador tenga forma de tal.
func NormalizarPregunta(id string) (string, error) {
	n := strings.TrimSpace(id)
	if n == "" {
		return "", fmt.Errorf("%w: el identificador de pregunta va vacio", ErrPreguntaNoValida)
	}
	if len(n) > MaxLongitudDeID {
		return "", fmt.Errorf("%w: pasa de %d caracteres", ErrPreguntaNoValida, MaxLongitudDeID)
	}
	for _, r := range n {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return "", fmt.Errorf("%w: lleva un espacio o un caracter de control dentro. Un "+
				"identificador con un invisible se ve igual que otro y no es el mismo",
				ErrPreguntaNoValida)
		}
	}
	return n, nil
}

// Ruta dice donde vive este almacen.
func (a *Almacen) Ruta() string { return a.ruta }

// Cuentas dice cuantas cuentas tienen alcance guardado. No devuelve los
// nombres: este tipo no enumera a nadie, por lo mismo que `usuarios.Almacen`.
func (a *Almacen) Cuentas() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.porUsuario)
}

// De devuelve el alcance de una cuenta. Si esa cuenta no ha guardado nada,
// devuelve un alcance vacio SIN error: ahi la nada si es la nada.
//
// EL CONTEXTO NO SE USA HOY porque el almacen es local. Esta en la firma por lo
// mismo que en `usuarios.HayAdministrador`: el almacen que venga despues
// (SQLite, un servicio) si lo necesitara, y anadirlo entonces cambiaria la firma
// en todos los sitios a la vez.
func (a *Almacen) De(_ context.Context, usuario string) (Alcance, error) {
	nombre, err := usuarios.NormalizarUsuario(usuario)
	if err != nil {
		return Alcance{}, fmt.Errorf("%w: %w", ErrSinUsuario, err)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	al, hay := a.porUsuario[nombre]
	if !hay {
		return Alcance{Usuario: nombre, Respuestas: map[string]Contestacion{}}, nil
	}
	return al.Copia(), nil
}

// Responder deja constancia de UNA respuesta.
//
// # POR QUE ES UN DELTA Y NO UN VOLCADO
//
// Dos pestanas abiertas contestando preguntas distintas a la vez son el caso
// normal, no el raro: se responde la entrevista mirando papeles en otra
// ventana. Si cada respuesta mandara el mapa entero, la segunda pestana
// escribiria el estado que leyo ANTES de que la primera guardara, y la
// respuesta de la primera desapareceria sin un error en ningun sitio.
//
// Con un delta aplicado DENTRO del candado, las dos sobreviven. La unica
// operacion que sigue siendo un volcado es Reemplazar, y ahi machacar es
// justamente lo que se pide (empezar de cero, importar un fichero).
func (a *Almacen) Responder(ctx context.Context, usuario, pregunta string, c Contestacion) error {
	// EL VALOR CERO NO ENTRA, y tampoco «las dos a la vez». La comprobacion es
	// la MISMA que usa la lectura de disco, a proposito: si fueran dos, el dia
	// que se separen habria contestaciones que se pueden escribir y no se
	// pueden volver a leer.
	if err := c.Valida(); err != nil {
		return fmt.Errorf("%w. Para quitar una respuesta se usa Olvidar", err)
	}
	id, err := NormalizarPregunta(pregunta)
	if err != nil {
		return err
	}
	return a.cambiar(ctx, usuario, func(al *Alcance) error {
		if _, ya := al.Respuestas[id]; !ya && len(al.Respuestas) >= MaxRespuestasPorCuenta {
			return fmt.Errorf("%w: esta cuenta ya tiene %d y el tope son %d",
				ErrDemasiadasRespuestas, len(al.Respuestas), MaxRespuestasPorCuenta)
		}
		al.Respuestas[id] = c
		return nil
	})
}

// Olvidar quita UNA respuesta. Quitar la que no esta no es un error: el
// resultado pedido (que no conste) ya se cumple.
func (a *Almacen) Olvidar(ctx context.Context, usuario, pregunta string) error {
	id, err := NormalizarPregunta(pregunta)
	if err != nil {
		return err
	}
	return a.cambiar(ctx, usuario, func(al *Alcance) error {
		delete(al.Respuestas, id)
		return nil
	})
}

// Reemplazar pone el alcance entero de una cuenta. Es la operacion de «empezar
// de cero» (mapa vacio) y la de importar un alcance.json.
//
// MACHACA A PROPOSITO, y por eso no se usa para responder una pregunta suelta.
func (a *Almacen) Reemplazar(ctx context.Context, usuario string, rs map[string]Contestacion) error {
	limpias := make(map[string]Contestacion, len(rs))
	for k, v := range rs {
		id, err := NormalizarPregunta(k)
		if err != nil {
			return err
		}
		if err := v.Valida(); err != nil {
			return fmt.Errorf("la pregunta %s: %w", id, err)
		}
		limpias[id] = v
	}
	if len(limpias) > MaxRespuestasPorCuenta {
		return fmt.Errorf("%w: se han pedido %d y el tope son %d",
			ErrDemasiadasRespuestas, len(limpias), MaxRespuestasPorCuenta)
	}
	return a.cambiar(ctx, usuario, func(al *Alcance) error {
		al.Respuestas = limpias
		return nil
	})
}

// cambiar es el UNICO camino de escritura: toma el candado, aplica el cambio,
// escribe a disco y solo entonces acepta el cambio en memoria.
//
// EL ORDEN IMPORTA Y ES EL MISMO QUE EN EL ALMACEN DE CUENTAS: al reves, un
// fallo de disco dejaria una respuesta que existe en este proceso y desaparece
// al reiniciar, y quien la escribio la daria por guardada.
func (a *Almacen) cambiar(_ context.Context, usuario string, f func(*Alcance) error) error {
	nombre, err := usuarios.NormalizarUsuario(usuario)
	if err != nil {
		return fmt.Errorf("%w: %w.\n"+
			"  Un alcance sin dueno no es de nadie: seria un cajon comun que despues leeria "+
			"cualquiera que entrase", ErrSinUsuario, err)
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	al, hay := a.porUsuario[nombre]
	if !hay {
		al = Alcance{Usuario: nombre, Respuestas: map[string]Contestacion{}}
	} else {
		al = al.Copia()
	}
	if err := f(&al); err != nil {
		return err
	}
	al.Actualizado = a.ahora().UTC()

	siguiente := make(map[string]Alcance, len(a.porUsuario)+1)
	for k, v := range a.porUsuario {
		siguiente[k] = v
	}
	siguiente[nombre] = al
	if err := a.guardar(siguiente); err != nil {
		return err
	}
	a.porUsuario = siguiente
	return nil
}

// guardar escribe el almacen entero de forma atomica: fichero temporal en el
// MISMO directorio, sincronizado, y rename encima. Es lo mismo que hace el
// almacen de cuentas, y por lo mismo: sin esto, un corte durante la escritura
// deja el fichero de cero bytes que Abrir se niega a leer.
func (a *Almacen) guardar(porUsuario map[string]Alcance) error {
	doc := almacenEnDisco{Version: VersionDelAlmacen}
	// EL ORDEN DEL FICHERO ES ESTABLE (por cuenta y por pregunta) para que dos
	// guardados del mismo estado den el mismo fichero. No es cosmetica: un
	// fichero que cambia sin que cambie el contenido convierte cualquier copia
	// de seguridad incremental en una copia completa, y ademas hace ilegible
	// cualquier diff que alguien mire para entender que paso.
	for _, nombre := range ordenar(porUsuario) {
		al := porUsuario[nombre]
		fila := alcanceEnDisco{Usuario: al.Usuario,
			Actualizado: al.Actualizado.UTC().Format(time.RFC3339)}
		ids := make([]string, 0, len(al.Respuestas))
		for id := range al.Respuestas {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			// LOS DOS CAMPOS SE COMPONEN AQUI Y NO EN UN String(): el de disco
			// tiene que poder volver a leerse, y `Contestacion.String()` es para
			// mensajes de error. Que fueran el mismo metodo es como se acaba
			// escribiendo «valor ALTA» en un campo que espera «valor».
			c := al.Respuestas[id]
			d := respuestaEnDisco{Pregunta: id}
			if c.EsValor() {
				d.Respuesta, d.Valor = EtiquetaDeValorEnDisco, c.Valor
			} else {
				d.Respuesta = c.Booleana.String()
			}
			fila.Respuestas = append(fila.Respuestas, d)
		}
		doc.Alcances = append(doc.Alcances, fila)
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("no puedo serializar el almacen de alcances: %w", err)
	}
	b = append(b, '\n')

	dir := filepath.Dir(a.ruta)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("no puedo crear el directorio %s del almacen de alcances: %w. "+
			"Arreglo: comprueba los permisos, o elige otro sitio con --alcances", dir, err)
	}
	tmp, err := os.CreateTemp(dir, "alcances-*.json.tmp")
	if err != nil {
		return fmt.Errorf("no puedo escribir en %s: %w. Arreglo: comprueba los permisos del "+
			"directorio", dir, err)
	}
	nombreTmp := tmp.Name()
	limpiar := func() { _ = tmp.Close(); _ = os.Remove(nombreTmp) }
	// Este fichero dice lo que una organizacion ha contestado sobre si misma:
	// que sistemas tiene, que datos trata y a que se dedica. No es una
	// credencial y aun asi no es del resto de la maquina.
	if err := os.Chmod(nombreTmp, 0o600); err != nil {
		limpiar()
		return fmt.Errorf("no puedo dejar el almacen de alcances en 0600: %w", err)
	}
	if _, err := tmp.Write(b); err != nil {
		limpiar()
		return fmt.Errorf("no puedo escribir el almacen de alcances: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		limpiar()
		return fmt.Errorf("no puedo sincronizar el almacen de alcances a disco: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(nombreTmp)
		return fmt.Errorf("no puedo cerrar el almacen de alcances: %w", err)
	}
	if err := os.Rename(nombreTmp, a.ruta); err != nil {
		_ = os.Remove(nombreTmp)
		return fmt.Errorf("no puedo dejar el almacen de alcances en %s: %w. Arreglo: "+
			"comprueba los permisos del directorio", a.ruta, err)
	}
	return nil
}

func ordenar(m map[string]Alcance) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// recortar acota lo que un error repite de la entrada. Un mensaje que devuelve
// entero lo que mando el cliente es la mitad de un reflejado, y esa mitad no
// hace falta para saber que el valor no se entiende.
func recortar(v string) string {
	const tope = 24
	if len(v) <= tope {
		return v
	}
	return v[:tope] + "..."
}

// RutaPorDefecto compone donde vive el almacen dentro del directorio de datos.
func RutaPorDefecto(datos string) string {
	d := strings.TrimSpace(datos)
	if d == "" {
		d = "."
	}
	return filepath.Join(d, NombreDelFichero)
}
