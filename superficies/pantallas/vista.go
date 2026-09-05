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
	// Inicio es LA RAIZ DEL SITIO, con su barra, para el enlace de la marca.
	//
	// Es un campo aparte de Base y no un calculo de la plantilla porque el
	// armazon compartido lo pide con ese nombre a las cuatro superficies, y en
	// las otras tres Base es el prefijo de SU montaje ("/acta", "/uar"): usar
	// aquel mandaria el logo del producto a la pantalla en la que te has
	// perdido en vez de sacarte de ella.
	Inicio string
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
// ES UN ALIAS, no un tipo nuevo: el tipo lo declara superficies/camino, que es
// quien declara tambien la plantilla compartida que lo pinta y la funcion que lo
// construye. Se conserva el nombre de aqui porque es el que usan las pruebas y
// quien lee esta superficie, y porque un alias no admite que las dos formas se
// separen: si camino le anade un campo, esta superficie lo tiene el mismo dia.
type VistaPaso = camino.PasoTira

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

	// LA REVELACION PROGRESIVA. Visibles son las preguntas que esta pagina
	// pinta y Dormidas las que ha dejado fuera por no decidir nada todavia.
	//
	// LAS DOS VIAJAN SIEMPRE, tambien en el modo largo, y Dormidas cuenta las
	// dormidas AUNQUE SE ESTEN PINTANDO: el cardinal es del corpus, no de la
	// pagina. Un numero que desapareciera al abrir la lista larga dejaria al
	// operador sin saber cuantas de las que esta viendo no sirven de nada.
	//
	// TotalPreguntas sigue siendo el total del corpus instalado, sin restar
	// nada: la barra de progreso no puede mejorar porque la pantalla ensene
	// menos, que seria exactamente la forma barata de aprobar esta puerta.
	Visibles int
	Dormidas int
	// VerTodas dice si esta pagina es la larga. Cambia el rotulo y el enlace
	// de vuelta, no la evaluacion: las dos paginas derivan lo mismo.
	VerTodas bool
	// Los dos enlaces de la revelacion. Llevan dentro las respuestas dadas,
	// como todo en esta superficie: abrir la lista larga no pierde la
	// entrevista.
	URLVerTodas string
	URLVerVivas string

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

	// ValoresQueNoSeEntienden es el cardinal de la TERCERA forma de la nada,
	// que no es la nada: respuestas que llegaron con un dato dentro que no se
	// ha podido interpretar (una fecha que pone "ayer", un valor que no esta en
	// la lista del paquete, dos valores para la misma pregunta).
	//
	// VA A LA VISTA Y NO SE TRAGA. Sin este numero, un valor que no se entiende
	// se comportaria igual que no haber contestado, y quien lo mando veria su
	// pregunta en blanco creyendo que no llego. Es el caso que el invariante 8
	// separa: presente y no interpretable NO es ausente.
	ValoresQueNoSeEntienden int

	// ValoresSinGuardar es cuantas respuestas CON VALOR hay puestas que el
	// almacen de alcances no sabe guardar.
	//
	// # El hueco que este numero declara, y por que no se tapa aqui
	//
	// El almacen guarda una respuesta como Si o como No (`Alcances.Responder`
	// toma una `Respuesta`), y un valor no cabe en ese vocabulario. Ampliarlo
	// toca `adaptadores/usuarios/alcances`, que esta fuera de la columna de esta
	// rebanada, asi que NO se toca: se declara con su cardinal, que es lo que
	// hace que el hueco moleste hasta que se cierre en vez de olvidarse.
	//
	// Lo que no se hace es guardar la mitad en silencio. Quien pulsa «guardar»
	// y vuelve al dia siguiente con la entrevista a medias y sin una linea que
	// lo explique tiene razon en dejar de creerse la pantalla.
	ValoresSinGuardar int

	// EL GUARDADO. Ver persistencia.go.
	//
	// Guarda y DeLaCuenta son DOS AFIRMACIONES DISTINTAS y la pantalla las
	// separa a proposito:
	//
	//	Guarda      esta pagina puede escribir en la cuenta (hay almacen, hay
	//	            sesion y hay token). Cambia los enlaces de respuesta por
	//	            formularios.
	//	DeLaCuenta  lo que se esta viendo SALE de la cuenta. Falso cuando la
	//	            direccion trae respuestas, que es el caso de un enlace
	//	            compartido: alli lo que se ve es lo del enlace, y decir que
	//	            es lo tuyo seria afirmar que has guardado algo que no has
	//	            guardado.
	Guarda     bool
	DeLaCuenta bool
	// Guardado es cuando se escribio por ultima vez lo de esta cuenta, ya
	// formateado. Vacio si nunca se guardo nada.
	Guardado string
	// Huerfanas son las respuestas GUARDADAS cuya pregunta el corpus instalado
	// ya no declara (se desinstalo un paquete, o cambio de version). Se dicen
	// en vez de descartarse en silencio: es la direccion contraria del
	// emparejamiento entre lo guardado y el corpus.
	Huerfanas int
	// CSRF y CampoCSRF son el token de esta peticion y el nombre de su campo.
	// Vacios cuando no se puede guardar, y entonces no se pinta ningun
	// formulario.
	CSRF      string
	CampoCSRF string
	// URLGuardar es la direccion a la que envian los formularios.
	URLGuardar string
	// Publica dice si ADOPTAR va a publicar ademas el alcance de la
	// INSTALACION, o sea el que alimenta al calendario y al plan de avisos
	// (invariante 12).
	//
	// SE PINTA PORQUE UN DATO QUE CRUZA DE LO PRIVADO A LO PUBLICO NO PUEDE
	// CRUZAR EN SILENCIO. Esas dos pantallas se sirven sin sesion, asi que
	// lo que se adopta lo va a ver cualquiera que abra esta instalacion. Y
	// el valor cero es no prometerlo: sin quien publique, adoptar guarda
	// solo en la cuenta y el rotulo no sale.
	Publica bool
	// IDsSi e IDsNo son las respuestas QUE SE ESTAN VIENDO, para que el
	// formulario de «guardar estas en mi cuenta» las lleve dentro.
	//
	// Van por CAMPO OCULTO y no leyendo la direccion en el servidor: un POST no
	// tiene por que conservar la consulta de la pagina desde la que se envio, y
	// apoyarse en que la conserva ataria el guardado a como el navegador
	// componga la accion del formulario. Lo que se guarda es exactamente lo que
	// la pagina ensena, y eso viaja en el propio envio.
	IDsSi []string
	IDsNo []string
}

// ParamVerTodas es el valor del parametro de lista larga, para que la plantilla
// no escriba una segunda copia de la cadena que declara revelacion.go.
func (v *VistaAlcance) ParamVerTodas() string { return VerTodas }

// VistaPregunta es una pregunta de la entrevista con sus tres enlaces.
type VistaPregunta struct {
	pantalla.Pregunta // Texto, Ayuda, Cita y Paquete vienen del corpus

	EsSi             bool
	EsNo             bool
	EsContradictoria bool
	SinResponder     bool
	Sugerida         bool

	// Dormida: esta pregunta no puede cambiar hoy el veredicto de ninguna
	// obligacion, asi que la lista corta la deja fuera. Solo se pinta en la
	// lista larga, y alli SIEMPRE con PorQueDormida al lado: una pregunta
	// marcada como inutil sin decir por que se lee como un fallo del producto.
	Dormida bool
	// PorQueDormida es la clave de catalogo del motivo. Vacia cuando la
	// pregunta esta viva.
	PorQueDormida string

	URLSi      string
	URLNo      string
	URLLimpiar string

	// LA MITAD CON VALOR. Ver valores.go: una pregunta que pide un valor no se
	// contesta con si ni con no, y pintarle esos dos botones era la forma de que
	// la respuesta no llegara nunca al motor.
	//
	// PideValor decide cual de las dos formas se pinta. Cuando es falso, todo lo
	// que sigue esta vacio y la pregunta se pinta como siempre: el valor cero de
	// esta mitad es «esta pregunta es de si/no», que es el restrictivo.
	PideValor bool
	// Opciones son los valores que declara el paquete, uno por enlace. NINGUNO
	// VIENE ELEGIDO DE PARTIDA, y esa es la propiedad entera: no hay estado por
	// defecto que afirme porque no hay estado por defecto.
	Opciones []VistaOpcion
	// EsCampoLibre distingue el enumerado (enlaces) del texto, el entero y la
	// fecha (un campo donde se escribe).
	EsCampoLibre bool
	// CampoValor es el nombre del campo del formulario, `v.<id>`. Sale de
	// ClaveValor y no se compone en la plantilla: componer un nombre de
	// parametro en dos sitios es tenerlo escrito dos veces.
	CampoValor string
	// Ocultos son las respuestas que ya hay, para que el formulario de campo
	// libre no las pierda. Un formulario GET no conserva la consulta de su
	// `action`, asi que si esto faltara, escribir una fecha borraria el resto de
	// la entrevista.
	Ocultos []VistaOculto
	// Formato es la pista de formato, hoy solo la de la fecha. Clave de
	// catalogo, vacia si el tipo no tiene forma que explicar.
	Formato string
	// ValorPuesto es lo contestado, si se entendio. Viene del corpus o del
	// operador y se ensena tal cual, sin pasar por el catalogo.
	ValorPuesto string
	// AvisoValor es la clave de catalogo del aviso cuando llego un dato y no se
	// entendio. Vacia cuando no hay nada que avisar. Ver EstadoDelValor.Clave.
	AvisoValor string
	// URLSinValor deshace la respuesta con valor.
	URLSinValor string
	// URLAccion es adonde apunta el formulario de campo libre. Va sin consulta a
	// proposito: la lleva entera en los ocultos.
	URLAccion string
}

// VistaOpcion es uno de los valores que un enumerado admite, con su enlace.
type VistaOpcion struct {
	// Valor es el identificador que declara el paquete. Se ensena tal cual: es
	// contenido del corpus y no pasa por el catalogo, igual que el texto de la
	// pregunta y su cita.
	Valor string
	URL   string
	// Elegido dice si es el que esta contestado. Nunca lo es al abrir la pagina.
	Elegido bool
}

// VistaOculto es un campo oculto del formulario de campo libre.
type VistaOculto struct {
	Nombre string
	Valor  string
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
	// El paso actual, POR SU RUTA. La ruta es lo que el camino declara
	// (Paso.Ruta) y es lo mismo que lee quien monta las superficies
	// (camino.RutaDe), asi que casan por un dato que ya existe en vez de por
	// una segunda tabla que traduzca identificadores de pantalla a
	// identificadores de paso.
	ruta := rutaDe(actual)
	id := ""
	for _, p := range s.pasos {
		if p.Ruta == ruta {
			id = p.ID
		}
	}
	// Y EL RESTO LO CONSTRUYE EL CAMINO, que es donde vive esa regla desde que
	// las cuatro superficies pintan la misma barra. Antes estaba entera aqui, y
	// por eso las otras tres no tenian barra: el dia que se la dieran, la
	// segunda copia habria empezado en este fichero.
	return camino.TiraDe(s.pasos, s.base, s.camino.URL, id, r.Consulta().Encode())
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
	if !esRutaDeEsteSitio(ruta) {
		return fmt.Errorf("%w: la direccion del camino es %q. Tiene que empezar por una sola "+
			"barra, y su segundo caracter no puede ser otra barra ni una contrabarra: con "+
			"cualquiera de las dos, el navegador la lee como otro anfitrion y el enlace que "+
			"existe para no perder a nadie saca al operador de plazum", ErrCamino, ruta)
	}
	return nil
}

// esRutaDeEsteSitio dice si una ruta de configuracion apunta a este sitio.
//
// SE MIRA EL SEGUNDO CARACTER, NO EL PREFIJO, y ese es todo el arreglo.
//
// Lo que habia era `!HasPrefix(ruta, "/") || HasPrefix(ruta, "//")`, y el autor
// YA HABIA PENSADO EN ESTE ATAQUE: el comentario de encima explica exactamente
// que con dos barras el navegador lee otro anfitrion. Rechazaba
// «//evil.example» y dejaba pasar «/\evil.example», porque Chrome y Firefox
// normalizan `/\` a `//` ANTES de resolver el destino. Es media guarda: la forma
// que el autor tenia en la cabeza estaba cerrada y su hermana no.
//
// Es la familia del invariante 8 aplicada a una cadena: las formas de la nada
// no son una, y la que se olvida es la que usa el atacante. En este mismo
// fichero ya habia pasado (misma alerta de CodeQL en pantallas.go, marcada como
// arreglada), asi que es la segunda vez.
//
// LAS FORMAS QUE SE RECHAZAN, y por que cada una:
//
//	"//x"    protocolo-relativa, la clasica
//	`/\x`    Chrome y Firefox la normalizan a "//x" antes de resolver
//	"/%2fx"  la barra escrita en porcentaje
//	"/%5cx"  la contrabarra escrita en porcentaje
//
// Las dos ultimas dependen de QUE CAPA decodifica y de CUANDO, y de las capas
// que hay delante de un navegador no mandamos ninguna. Se rechazan las cuatro,
// que sale mas barato que acertar.
//
// Y LA RUTA DE UN SOLO CARACTER SE ACEPTA: "/" es la raiz de este sitio, es un
// destino legitimo y no tiene segundo caracter que mirar. Tratarla como
// sospechosa por ser corta seria decir que no a un caso bueno, que es la otra
// forma de que una guarda deje de servir.
func esRutaDeEsteSitio(ruta string) bool {
	if !strings.HasPrefix(ruta, "/") {
		return false
	}
	if ruta == "/" {
		return true
	}
	if ruta[1] == '/' || ruta[1] == '\\' {
		return false
	}
	enMinusculas := strings.ToLower(ruta)
	return !strings.HasPrefix(enMinusculas, "/%2f") && !strings.HasPrefix(enMinusculas, "/%5c")
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
