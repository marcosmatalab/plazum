package pantallas

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/marcosmatalab/plazum/nucleo/pantalla"
	"github.com/marcosmatalab/plazum/superficies/camino"
)

// El modelo de vista: lo unico que ven las plantillas.
//
// Regla de este fichero, y es la que mantiene honesta a la superficie entera:
// aqui NO se decide que se ensena ni en que orden. Eso ya lo decidio
// nucleo/pantalla, con casos dorados comparados byte a byte. Lo de aqui es
// exclusivamente presentacion: que enlace lleva a donde, que columna cabe en la
// tabla y que clave de catalogo rotula cada cosa.
//
// Y la otra regla: los rotulos son CLAVES de catalogo, el contenido del corpus
// es texto. Un campo de estos tipos es una cosa o la otra, nunca las dos, y el
// nombre lo dice: Titulo y Clave son claves; Texto, Ayuda, Cita y Etiqueta
// vienen del paquete y viajan tal cual, en el idioma del paquete.

// Marco es lo que comparten todas las pantallas: la navegacion y el pie.
type Marco struct {
	// Idioma es el que de verdad se va a renderizar, no el que pidio el
	// navegador. Va a <html lang>, y una pagina que declara un idioma que no
	// lleva dentro rompe a los lectores de pantalla.
	Idioma string
	// Base es el prefijo bajo el que esta montada la superficie.
	Base string
	// Estatico es la ruta de los ficheros servidos por nosotros.
	Estatico string
	// Cuerpo dice que sub-plantilla pinta el contenido.
	Cuerpo string
	// Titulo es la CLAVE de catalogo del titulo de la pantalla.
	Titulo string
	// Menu son las seis pantallas, siempre las seis. Una pantalla que
	// desaparece del menu porque no tiene datos deja al operador sin saber
	// que existia.
	Menu []Entrada
	// Tira es EL CAMINO GUIADO en la barra lateral: los pasos en su orden, con
	// su numero y con el actual marcado. Vacia cuando quien monta no ha pasado
	// pasos, que es el valor cero y el restrictivo.
	//
	// Los pasos NO se escriben en la plantilla. Salen de camino.Canonico(), que
	// es donde vive el orden, y llegan hasta aqui por Opciones.Pasos: una
	// segunda lista escrita al lado se desincroniza el dia que el camino gane
	// un paso, que es el dia en que la barra lateral tiene que ensenarlo.
	Tira []VistaPaso
	// Fuentes son los paquetes instalados con su aviso de derechos, tal como
	// los deriva nucleo/pantalla. Se pintan en el pie de TODAS las paginas,
	// al lado del descargo de asesoramiento juridico y por la misma razon:
	// la Decision 2011/833/UE autoriza reutilizar el DOUE con atribucion, y
	// un aviso que solo sale en la portada es un aviso que no se lee.
	//
	// El texto es CONTENIDO del corpus y no clave de catalogo: viaja tal
	// cual, en el idioma del paquete, y no se traduce.
	Fuentes []pantalla.Fuente
}

// VistaPaso es un paso del camino guiado listo para pintar en la barra lateral.
//
// Es camino.Enlace (numero, rotulo, direccion y si es el actual) mas UNA cosa
// que la tira de aquella no distingue: si el paso se recorre sin salir del
// navegador. Hace falta porque aqui los dos pasos que todavia no son pantalla SI
// llevan direccion, la de la pantalla del camino, que es donde esta la orden que
// los hace hoy. Sin este campo, la plantilla los pintaria como pantallas
// normales y el operador pulsaria esperando el calendario.
type VistaPaso struct {
	camino.Enlace
	// EsPantalla dice si el destino es el paso mismo (true) o la pantalla del
	// camino, que solo cuenta como se hace (false).
	EsPantalla bool
}

// Entrada es una pantalla en el menu.
type Entrada struct {
	ID     pantalla.ID
	Titulo string // clave de catalogo
	URL    string
	Activa bool
	// Vacia marca en el menu las que no tienen contenido todavia, para que
	// el operador no las visite esperando algo.
	Vacia bool
	// N y Marcador ensenan un contador junto a la entrada cuando aporta
	// algo (cuantas obligaciones aplican). Marcador es clave de catalogo.
	N        int
	Marcador string
}

// VistaAlcance es la pantalla de Alcance: la entrevista y su derivacion.
type VistaAlcance struct {
	Marco
	Vacia  bool
	PorQue string // clave de catalogo, la que trae el modelo derivado
	// Origen distingue "no hay corpus instalado" de "todavia no hay estado".
	// Son dos problemas con dos arreglos distintos y confundirlos es una
	// llamada de soporte, asi que se dice en las seis pantallas y no solo en
	// las que salen del estado.
	Origen string

	Preguntas []VistaPregunta
	Campos    []pantalla.Campo

	// Respondidas de TotalPreguntas, y la siguiente sugerida. Las preguntas
	// ya vienen ordenadas por cuantas obligaciones desbloquea cada una, asi
	// que la siguiente sugerida es simplemente la primera sin responder: el
	// operador no tiene que elegir por donde empezar.
	Respondidas     int
	TotalPreguntas  int
	Siguiente       string // ID de la pregunta sugerida
	Contradictorias int

	// Resumen y las listas son la derivacion a un clic.
	Resumen Resumen
	Aplican []Veredicto
	// AplicanMas son las que aplican y no caben en el panel. La lista se
	// acota porque un corpus grande convertiria el panel de la entrevista en
	// un listado de miles de lineas, que es justo el catalogo de controles en
	// frio que este diseno evita. El listado entero esta en Controles.
	AplicanMas int
	Proximas   []Veredicto // pendientes que desbloquea la siguiente pregunta

	HayRespuestas bool
	URLControles  string
	URLLimpiar    string
}

// VistaPregunta es una pregunta de la entrevista con sus tres enlaces.
type VistaPregunta struct {
	pantalla.Pregunta // Texto, Ayuda, Cita y Paquete vienen del corpus

	EsSi             bool
	EsNo             bool
	EsContradictoria bool
	SinResponder     bool
	Sugerida         bool

	URLSi      string
	URLNo      string
	URLLimpiar string
}

// VistaTabla sirve para Controles y para Certificados: la forma es la misma y
// el contenido lo pone quien deriva, igual que en nucleo/pantalla.
type VistaTabla struct {
	Marco
	Vacia  bool
	PorQue string
	Origen string // ver VistaAlcance.Origen

	// Columnas son claves de catalogo con el prefijo columna.; las
	// desconocidas se pintan con su nombre crudo para que se vean.
	Columnas             []string
	ColumnasDesconocidas []string

	Filas   []Veredicto
	Resumen Resumen
	Filtros []VistaFiltro

	// Paginacion. Un corpus grande no puede convertirse en una sola pagina
	// de varios megabytes que ningun navegador termina de pintar.
	Desde        int
	Hasta        int
	Total        int
	URLAnterior  string
	URLSiguiente string

	// EsEntregables cambia como se lee la columna de motivos: en Controles
	// el motivo es una respuesta de la entrevista, en Certificados es la
	// obligacion que pide el documento.
	EsEntregables bool
	URLAlcance    string
	SinResultados bool
}

// VistaFiltro es uno de los cuatro filtros de estado de una tabla.
type VistaFiltro struct {
	Clave  string // clave de catalogo
	URL    string
	Activo bool
	N      int
}

// VistaVacia es una pantalla sin contenido, CON su explicacion.
//
// Existe como tipo propio para que sea imposible pintar una pantalla en blanco
// sin decir por que: PorQue es obligatorio y hay un test que se pone rojo si
// alguna de las seis llega aqui sin el.
type VistaVacia struct {
	Marco
	PorQue string // clave de catalogo
	// Origen es la clave de catalogo que distingue "no hay corpus instalado"
	// de "todavia no hay estado". Son dos problemas con dos arreglos
	// distintos y confundirlos es una llamada de soporte.
	Origen     string
	URLAlcance string
}

// VistaHoy es la pantalla Hoy: el estado del planificador arriba y, debajo, lo
// que Hoy ensenara cuando haya expediente.
//
// Tiene tipo propio y no reusa VistaVacia por una razon que no es de estilo: en
// Hoy la parte que importa NO esta vacia. El vigilante tiene algo que decir
// desde el primer minuto, tambien (y sobre todo) cuando no hay ni corpus ni
// expediente, porque "tu planificador no ha corrido nunca" es informacion, no
// un hueco.
type VistaHoy struct {
	Marco
	// Panel son las cifras grandes y sus derivaciones. Va embebido para
	// que la plantilla las lea sin un nivel mas de anidamiento.
	Panel
	// PorQue explica que aparecera aqui cuando haya estado. Clave de
	// catalogo.
	PorQue string
	// Origen distingue "no hay corpus" de "no hay estado".
	Origen     string
	URLAlcance string
	// Planificador es el veredicto de nucleo/pantalla.Vigilar, calculado con
	// el instante de ESTA peticion. Clave, Arreglo y Descargo son claves de
	// catalogo; Horas y UmbralHoras son numeros que viajan como argumento de
	// la clave, porque la forma plural la decide el idioma.
	Planificador pantalla.Planificador
}

// VistaError es la pagina de un fallo de la peticion.
type VistaError struct {
	Marco
	// Clave de catalogo del mensaje. Nunca se refleja lo que mando el
	// cliente: un mensaje de error que repite la entrada es la mitad de un
	// XSS reflejado, y esa mitad no hace falta para nada.
	Clave      string
	Codigo     int
	URLAlcance string
}

// claveOrigen traduce el origen del modelo a clave de catalogo.
func claveOrigen(o pantalla.Origen) string {
	if o == pantalla.DelEstado {
		return "origen.estado"
	}
	return "origen.corpus"
}

// menu construye las seis entradas conservando el orden del modelo derivado.
func (s *Superficie) menu(m modelo, actual pantalla.ID, r Respuestas, aplican int) []Entrada {
	q := r.Consulta()
	out := make([]Entrada, 0, len(m.pantallas))
	for _, p := range m.pantallas {
		e := Entrada{
			ID:     p.ID,
			Titulo: p.Titulo,
			URL:    s.enlace(rutaDe(p.ID), q),
			Activa: p.ID == actual,
			Vacia:  p.Vacia,
		}
		if p.ID == pantalla.Controles && !p.Vacia {
			e.N, e.Marcador = aplican, "menu.aplican"
		}
		out = append(out, e)
	}
	// Y LA VUELTA AL CAMINO GUIADO, al final y siempre que este configurada.
	//
	// Va en el menu y no en un pie porque el problema que resuelve es de
	// descubrimiento: hasta hoy, desde estas seis pantallas no habia forma de
	// enterarse de que existian el acta y la revision de accesos, ni de en que
	// orden se recorre esto. Un enlace que solo sale en la portada es un enlace
	// que no se ve, que es lo mismo que decidio que la atribucion fuera al pie
	// de TODAS las paginas.
	//
	// Nunca se marca activa: esta entrada no es una de estas pantallas, asi que
	// ninguna de ellas "esta" en el camino.
	//
	// Y LA ENTRADA LLEVA LAS RESPUESTAS PUESTAS, igual que las otras seis. Las
	// respuestas de la entrevista viajan en la direccion y no se guardan (esta
	// misma superficie lo dice con esas palabras en Alcance), asi que un enlace
	// pelado al camino borra el trabajo de quien lo pulse. El camino que existe
	// para no perder a nadie seria el que te pierde.
	if s.camino.URL != "" {
		e := s.camino
		if len(q) > 0 {
			e.URL += "?" + q.Encode()
		}
		out = append(out, e)
	}
	return out
}

// tira construye el camino guiado listo para pintar en la barra lateral,
// marcando el paso en el que esta la pantalla que se esta sirviendo.
//
// DOS COSAS QUE NO SON DE ESTILO:
//
// El paso actual se resuelve POR LA RUTA de la pantalla, no por una tabla que
// traduzca identificadores de pantalla a identificadores de paso. La ruta es lo
// que el camino declara (Paso.Ruta) y es lo mismo que lee quien monta las
// superficies (camino.RutaDe), asi que aqui casan por el dato que ya existe en
// vez de por una segunda lista. Una pantalla que no es ningun paso (Hoy,
// Certificados, Personas, Estado) no marca ninguno, que es lo correcto: marcar
// uno "por si acaso" seria decirle al operador que esta donde no esta.
//
// Y LOS ENLACES LLEVAN LAS RESPUESTAS PUESTAS en los pasos que trabajan con el
// alcance. Las respuestas de la entrevista viajan en la direccion y no se
// guardan (la propia pantalla de Alcance lo dice con esas palabras), asi que una
// tira con enlaces pelados se come el trabajo de quien la use, que es el fallo
// que el camino guiado ya tuvo una vez. El emparejamiento entre el paso y su
// enlace se hace POR IDENTIFICADOR, no por posicion: los dos vienen de la misma
// llamada, pero atarlos por indice seria escribir aqui una dependencia del orden
// que nadie declara.
func (s *Superficie) tira(actual pantalla.ID, r Respuestas) []VistaPaso {
	if len(s.pasos) == 0 {
		return nil
	}
	// El paso actual, por su ruta.
	ruta := rutaDe(actual)
	id := ""
	porID := make(map[string]camino.Paso, len(s.pasos))
	for _, p := range s.pasos {
		porID[p.ID] = p
		if p.Ruta == ruta {
			id = p.ID
		}
	}
	consulta := r.Consulta().Encode()
	conConsulta := func(u string) string {
		if u == "" || consulta == "" {
			return u
		}
		return u + "?" + consulta
	}

	enlaces := camino.Tira(s.pasos, s.base, id)
	out := make([]VistaPaso, 0, len(enlaces))
	for _, e := range enlaces {
		p := porID[e.ID]
		v := VistaPaso{Enlace: e, EsPantalla: e.URL != ""}
		switch {
		case v.EsPantalla && p.LlevaAlcance:
			v.URL = conConsulta(e.URL)
		case !v.EsPantalla:
			// EL PASO QUE TODAVIA NO ES PANTALLA NO ES UN CALLEJON. Se pinta
			// apagado y con su rotulo escrito ("por terminal"), y lleva a la
			// pantalla del camino, que es donde esta la orden exacta que lo
			// hace hoy. Un paso que no lleva a ningun sitio ensena a ignorar
			// la barra lateral entera.
			//
			// Y la consulta viaja tambien: la pantalla del camino la reparte a
			// los pasos que la usan, asi que ir alli y volver no borra la
			// entrevista.
			v.URL = conConsulta(s.camino.URL)
		}
		out = append(out, v)
	}
	return out
}

// validarCamino comprueba el enlace de vuelta al camino guiado.
//
// LAS DOS MITADES O NINGUNA. Una direccion sin rotulo pinta una entrada de menu
// sin palabras y un rotulo sin direccion pinta un enlace que no lleva a ningun
// sitio: las dos son peores que no tener la entrada, y las dos salen del mismo
// descuido, que es rellenar un campo y olvidar el otro.
//
// Y la direccion tiene que ser de este sitio. Con dos barras al principio el
// navegador la lee como otro anfitrion, asi que el enlace que existe para no
// perder a nadie sacaria al operador de plazum.
func validarCamino(ruta, clave string) error {
	if ruta == "" && clave == "" {
		return nil // el valor cero: no hay entrada, y la interfaz es la de antes
	}
	if ruta == "" || clave == "" {
		return fmt.Errorf("%w: llega la direccion %q y el rotulo %q, y hacen falta los dos. "+
			"Arreglo: pasar CaminoRuta y CaminoClave juntos, o ninguno de los dos",
			ErrCamino, ruta, clave)
	}
	if !strings.HasPrefix(ruta, "/") || strings.HasPrefix(ruta, "//") {
		return fmt.Errorf("%w: la direccion del camino es %q. Tiene que empezar por una sola "+
			"barra: con dos, el navegador la lee como otro anfitrion y el enlace que existe "+
			"para no perder a nadie saca al operador de plazum", ErrCamino, ruta)
	}
	return nil
}

// enlace compone una direccion de la superficie. Se compone SIEMPRE aqui y
// nunca concatenando a mano en una plantilla: una direccion construida en la
// plantilla se escapa distinto segun el contexto y acaba siendo un fallo raro.
func (s *Superficie) enlace(ruta string, v url.Values) string {
	u := s.base + ruta
	if len(v) == 0 {
		return u
	}
	return u + "?" + v.Encode()
}

// rutaDe da la ruta de una pantalla. El identificador viene de nucleo/pantalla,
// asi que anadir una septima pantalla al modelo le da ruta sola.
func rutaDe(id pantalla.ID) string { return "/" + string(id) }
