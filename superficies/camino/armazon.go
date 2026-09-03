package camino

// EL ARMAZON COMPARTIDO: una sola barra lateral para las cuatro superficies.
//
// # Que problema resuelve, y no es de estetica
//
// La barra lateral con el camino guiado la pintaban SOLO las seis pantallas de
// superficies/pantallas. El camino, el acta y la revision de accesos armaban
// cada una su propio <body> y su unica salida era el enlace de vuelta. El
// sintoma es concreto y se mide en una sesion: quien entra en el acta desde el
// consejo o en la revision de accesos desde un correo pierde la orientacion
// entera, porque la pantalla no dice en que paso del camino esta ni cual es el
// siguiente. Eran tres superficies de cuatro, contadas en docs/pendientes.md.
//
// # Por que el armazon vive AQUI y no en superficies/pantallas
//
// Porque pantallas ya importa este paquete (lee el camino canonico), asi que la
// direccion contraria seria un ciclo. Y porque el dueno natural del armazon es
// quien declara el camino: la barra lateral ES el camino, y el resto es marco.
//
// # La regla que sostiene todo esto
//
// La plantilla compartida (armazon/armazon.html) NO escribe ni un paso. Los
// pasos llegan como dato desde Canonico(), igual que antes, y el emparejamiento
// entre un paso y su enlace se hace POR IDENTIFICADOR y nunca por posicion
// (invariante 7). Hay dos puertas: una compara la barra pintada contra
// Canonico() y otra lee los ficheros de plantilla buscando un rotulo de paso
// escrito a mano.

import (
	"embed"
	"io/fs"
	"strings"
)

// PasoTira es un paso del camino LISTO PARA PINTAR en la barra lateral.
//
// Es Enlace (numero, rotulo, direccion y si es el actual) mas UNA cosa que
// aquel no distingue: si el paso se recorre sin salir del navegador. Hace falta
// porque los dos pasos que todavia no son pantalla SI llevan direccion, la de la
// pantalla del camino, que es donde esta la orden que los hace hoy. Sin este
// campo la plantilla los pintaria como pantallas normales y quien pulsara
// esperaria el calendario.
type PasoTira struct {
	Enlace
	// EsPantalla dice si el destino es el paso mismo (true) o la pantalla del
	// camino, que solo cuenta como se hace (false).
	EsPantalla bool
}

// TiraDe construye la barra lateral del camino guiado, marcando el paso actual.
//
// Es EL UNICO sitio donde se decide como se comporta la tira, y por eso lo usan
// las cuatro superficies. Antes esta logica vivia entera dentro de
// superficies/pantallas y las otras tres no tenian barra: el dia que se la
// dieran, la copia habria empezado ahi.
//
// Los argumentos, y por que cada uno:
//
//	pasos     el camino. Vacio devuelve nil, que es el valor cero restrictivo:
//	          quien no pasa pasos no pinta barra, en vez de pintar una barra
//	          inventada con el canonico (invariante 8).
//	raiz      el prefijo del sitio del que cuelgan las rutas de los pasos.
//	caminoURL la pantalla del camino, a donde van los pasos que todavia no son
//	          pantalla. Vacia deja esos pasos sin enlace, que se pintan apagados.
//	actual    el ID del paso en el que esta quien pinta. Una cadena que no sea
//	          ningun paso NO es un error: no se marca ninguno, que es lo que le
//	          corresponde a una pantalla que no esta en el camino.
//	consulta  la parte de consulta ya codificada. Se cuelga SOLO de los pasos
//	          que declaran LlevaAlcance: las respuestas de la entrevista viajan
//	          en la direccion y no se guardan, asi que un enlace pelado se come
//	          el trabajo de quien lo pulse. Y colgarsela a los que no la leen
//	          (acta, revision de accesos) seria arrastrar un dato hasta una
//	          pantalla que no lo entiende.
func TiraDe(pasos []Paso, raiz, caminoURL, actual, consulta string) []PasoTira {
	if len(pasos) == 0 {
		return nil
	}
	// EL EMPAREJAMIENTO ES POR IDENTIFICADOR (invariante 7). Tira() y esta
	// funcion recorren la misma lista, pero atarlos por indice escribiria aqui
	// una dependencia del orden que nadie declara y que se rompe en silencio el
	// dia que una de las dos filtre un paso.
	porID := make(map[string]Paso, len(pasos))
	for _, p := range pasos {
		porID[p.ID] = p
	}
	conConsulta := func(u string) string {
		if u == "" || consulta == "" {
			return u
		}
		return u + "?" + consulta
	}

	enlaces := Tira(pasos, raiz, actual)
	out := make([]PasoTira, 0, len(enlaces))
	for _, e := range enlaces {
		p := porID[e.ID]
		v := PasoTira{Enlace: e, EsPantalla: e.URL != ""}
		switch {
		case v.EsPantalla && p.LlevaAlcance:
			v.URL = conConsulta(e.URL)
		case !v.EsPantalla:
			// EL PASO QUE TODAVIA NO ES PANTALLA NO ES UN CALLEJON. Se pinta
			// apagado y con su rotulo escrito ("por terminal"), y lleva a la
			// pantalla del camino, que es donde esta la orden exacta que lo
			// hace hoy. Un paso que no lleva a ningun sitio ensena a ignorar la
			// barra lateral entera.
			v.URL = conConsulta(caminoURL)
		}
		out = append(out, v)
	}
	return out
}

//go:embed armazon
var armazonFS embed.FS

// PlantillasDelArmazon expone la plantilla compartida, por si alguien quiere
// montar otro motor de render.
func PlantillasDelArmazon() fs.FS { return armazonFS }

// PatronDelArmazon es el patron con el que se carga la plantilla compartida.
//
// Se exporta para que las superficies no lo escriban a mano. Una cadena
// "armazon/*.html" repetida en cuatro paquetes es una cadena que se queda vieja
// el dia que el fichero cambie de sitio, y su sintoma seria un error de arranque
// en una superficie y no en las otras tres.
const PatronDelArmazon = "armazon/*.html"

// ConArmazon junta las plantillas propias de una superficie con la del armazon.
//
// POR QUE HACE FALTA UNA UNION Y NO BASTA UN SEGUNDO PATRON: el motor de
// plantillas (adaptadores/plantilla) recibe UN sistema de ficheros y varios
// patrones, y las plantillas de cada superficie viven en su propio embed.FS.
// Sin esto, la unica forma de compartir el armazon seria copiar el marcado en
// las cuatro, que es exactamente la segunda lista que este repositorio no
// admite: cuatro copias de una barra de navegacion se separan el dia que una
// cambie, y el sintoma seria una pantalla que enlaza a un paso que ya no existe.
//
// Los dos arboles son DISJUNTOS por construccion: las superficies embeben
// "plantillas/" y el armazon vive en "armazon/". Por eso la union puede resolver
// "el primero que responda" sin tener que fusionar directorios, y por eso las
// definiciones de plantilla tampoco chocan: el armazon define nombres propios
// ("armazon-marca", "tira-camino") y no "pagina", que es de cada superficie.
// Hay una puerta que comprueba las dos mitades y que los nombres no colisionan.
func ConArmazon(propio fs.FS) fs.FS {
	if propio == nil {
		// Sin plantillas propias no hay pagina que pintar, pero devolver nil
		// aqui convertiria el olvido en un panico en la primera peticion en vez
		// de en el error de arranque que da el motor. Se pasa el armazon solo y
		// que el motor diga que falta "pagina".
		return armazonFS
	}
	return union{propio: propio, armazon: armazonFS}
}

// union sirve dos arboles disjuntos como uno.
//
// Implementa ReadDirFS y ReadFileFS ademas de FS porque es lo que usan de
// verdad fs.Glob y template.ParseFS: sin ReadDir, fs.Glob cae en Open y el
// fichero del armazon no aparece en el patron; sin ReadFile, ParseFS lo abre
// igual pero por el camino lento. Las dos estan, y hay puerta.
type union struct {
	propio  fs.FS
	armazon fs.FS
}

var (
	_ fs.FS         = union{}
	_ fs.ReadDirFS  = union{}
	_ fs.ReadFileFS = union{}
)

func (u union) Open(nombre string) (fs.File, error) {
	if f, err := u.propio.Open(nombre); err == nil {
		return f, nil
	}
	return u.armazon.Open(nombre)
}

func (u union) ReadDir(nombre string) ([]fs.DirEntry, error) {
	if e, err := fs.ReadDir(u.propio, nombre); err == nil {
		return e, nil
	}
	return fs.ReadDir(u.armazon, nombre)
}

func (u union) ReadFile(nombre string) ([]byte, error) {
	if b, err := fs.ReadFile(u.propio, nombre); err == nil {
		return b, nil
	}
	return fs.ReadFile(u.armazon, nombre)
}

// ClavesDelArmazon son las claves de catalogo que pide la plantilla compartida.
//
// Las declara ESTE paquete y no cada superficie por la misma razon por la que la
// plantilla vive aqui: son las palabras del armazon, no las de la pantalla que
// lo usa. Cada superficie las suma a su propia lista, y su test de cobertura de
// catalogo las exige igual que las suyas.
//
// LOS ROTULOS DE LOS PASOS NO ESTAN AQUI a proposito: los declara el camino
// canonico y llegan como dato. Escribirlos seria una segunda copia del camino.
func ClavesDelArmazon() []string {
	return []string{
		"ui.marca",
		"camino.titulo",
		"ui.aqui",
		"ui.paso_por_terminal",
	}
}

// nombresDelArmazon son las plantillas que define el armazon compartido.
//
// Existe para que la puerta de colision pueda compararlas contra las de cada
// superficie sin leer el fichero a ojo.
var nombresDelArmazon = []string{"armazon-marca", "tira-camino", "dibujo-vacio"}

// NombresDelArmazon devuelve las plantillas que define el armazon compartido.
func NombresDelArmazon() []string {
	return append([]string(nil), nombresDelArmazon...)
}

// InicioDe compone el enlace de la marca: la raiz del sitio, con su barra.
//
// Es de una linea y esta aqui igualmente porque el fallo que evita ya se ha
// visto: cada superficie tiene su propio Base (el prefijo donde esta montada) y
// usarlo para la marca manda a "/acta/" en vez de a la raiz, o sea que el logo
// del producto no te saca de la pantalla en la que te has perdido.
func InicioDe(raiz string) string {
	return strings.TrimSuffix(raiz, "/") + "/"
}
