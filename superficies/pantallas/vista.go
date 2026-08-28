package pantallas

import (
	"net/url"

	"github.com/marcosmatalab/plazum/nucleo/pantalla"
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
	return out
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
