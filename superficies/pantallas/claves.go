package pantallas

import (
	"sort"

	"github.com/marcosmatalab/plazum/nucleo/pantalla"
	"github.com/marcosmatalab/plazum/superficies/camino"
)

// Las claves de catalogo que necesita esta superficie.
//
// Por que existe esta lista. puertos.Catalogo tiene Faltantes(idioma), que
// compara un idioma contra el de por defecto: sirve para saber si al ingles le
// falta lo que el espanol tiene, y no sirve para nada mas. Lo que NO responde es
// la pregunta que de verdad rompe la pantalla: ¿cubre el catalogo lo que la
// interfaz pide? Un catalogo completo en los dos idiomas y sin la clave
// "alcance.pregunta.si" da una pagina con botones que ponen
// "alcance.pregunta.si", y Faltantes da verde.
//
// Esta funcion es la otra mitad. Es el contrato entre esta superficie y quien
// escriba el catalogo: estas claves, todas, o hay huecos en pantalla. Un test
// del paquete la compara contra las claves que las plantillas piden de verdad
// durante un barrido de las seis pantallas, en las dos direcciones, asi que no
// puede quedarse corta ni sobrarle nada.
//
// Las claves son de INTERFAZ. Aqui NO va ni una linea de contenido del corpus:
// la etiqueta de un atributo, la ayuda, la cita y el texto de una pregunta
// viajan tal cual, en el idioma del paquete, y no se traducen. Traducir texto
// transcrito del BOE crea obra derivada y se sale de la estratificacion de
// licencias que sostiene el corpus.

// clavesFijas son las que escriben las plantillas y el codigo de la superficie.
var clavesFijas = []string{
	// Marco de la pagina.
	"ui.marca",
	"ui.saltar",
	"ui.navegacion",
	"ui.pie.no_asesoramiento",
	"pantalla.error.titulo",

	// Menu.
	"menu.aplican",
	"menu.vacia",

	// La tira del camino guiado en la barra lateral. Los ROTULOS DE LOS PASOS
	// no estan aqui a proposito: los declara superficies/camino y llegan como
	// dato en Opciones.Pasos, asi que escribirlos aqui seria una segunda copia
	// del camino. Lo unico que pone esta superficie son las tres palabras que
	// son suyas: el rotulo de la seccion, el "estas aqui" del paso actual y el
	// apunte de los pasos que todavia se hacen por terminal.
	"camino.titulo",
	"ui.aqui",
	"ui.paso_por_terminal",

	// Hoy: el estado del planificador y el del canal del latido.
	//
	// Aqui van SOLO los dos rotulos de seccion y el de los plazos. Las claves
	// de cada veredicto las publica nucleo/pantalla en
	// ClavesDelPlanificador() y se anaden abajo: escribirlas aqui seria una
	// tercera copia de la misma lista, y la tercera copia siempre es la que
	// se queda vieja.
	"pantalla.hoy.planificador",
	"pantalla.hoy.canal",
	"pantalla.hoy.plazos",

	// Hoy: EL PANEL DE INICIO. Las cuatro cifras, sus derivaciones y los dos
	// estados en los que una cifra no es un numero.
	//
	// hoy.cifra.sin_dato es la que separa "he contado y salen cero" de "no he
	// podido contar", y por eso es una cadena y no un cero: son dos frases
	// distintas y solo la primera es un numero.
	//
	// hoy.sin_constancia.descargo es EL descargo de esta pantalla, el que dice
	// que una ausencia de registro no es un incumplimiento. Va con el dato, no
	// en un pie, y tiene puerta propia.
	"pantalla.hoy.cifra.vence_semana",
	"pantalla.hoy.cifra.sin_constancia",
	"pantalla.hoy.cifra.te_alcanzan",
	"pantalla.hoy.cifra.marcos",
	"pantalla.hoy.cifra.sin_dato",
	"pantalla.hoy.cifra.sin_corpus",
	"pantalla.hoy.cifra.sin_responder",
	"pantalla.hoy.cifra.derivacion",
	"pantalla.hoy.vence_semana.esperando",
	"pantalla.hoy.vence_semana.vacio",
	"pantalla.hoy.sin_constancia.descargo",
	"pantalla.hoy.sin_constancia.vacio",
	"pantalla.hoy.vencida.desde",
	"pantalla.hoy.marcos.caption",
	"pantalla.hoy.ir_alcance",

	// Pantallas vacias: por que lo estan y que se hace al respecto.
	"origen.corpus",
	"origen.estado",
	"vacia.que_hacer",
	"vacia.sin_explicacion",
	"vacia.volver_alcance",

	// Alcance: la entrevista.
	"alcance.intro",
	"alcance.progreso",
	"alcance.siguiente",
	"alcance.sin_preguntas",
	"alcance.pregunta.si",
	"alcance.pregunta.no",
	"alcance.pregunta.limpiar",
	"alcance.pregunta.desbloquea",
	"alcance.pregunta.la_pide",
	"alcance.pregunta.contradictoria",
	"alcance.pregunta.respondida_si",
	"alcance.pregunta.respondida_no",

	// Alcance: LA REVELACION PROGRESIVA. Las preguntas que hoy no deciden
	// nada se dejan fuera de la lista corta, y estas seis claves son lo que
	// hace que eso no sea esconder: el cardinal, el enlace en las dos
	// direcciones y el motivo de cada una. Ver revelacion.go.
	//
	// titulo y ver reciben un CONTADOR, por lo mismo que aplican y proximas:
	// la forma plural la decide el catalogo, que es quien sabe en que idioma
	// escribe.
	"alcance.dormidas.titulo",
	"alcance.dormidas.porque",
	"alcance.dormidas.ver",
	"alcance.dormidas.volver",
	"alcance.dormidas.nadie_la_pide",
	"alcance.dormidas.ya_decidida",

	// Alcance: la derivacion a un clic.
	"alcance.derivacion.titulo",
	"alcance.derivacion.sin_respuestas",
	"alcance.derivacion.no_es_dictamen",
	"alcance.derivacion.no_guardado",
	// aplican y proximas reciben un CONTADOR como argumento, y no dos claves
	// para singular y plural. La forma plural depende del idioma (el ruso
	// tiene tres, el arabe seis) y esa decision es del catalogo, que es quien
	// sabe en que idioma esta escribiendo. Aqui solo se pasa el numero.
	"alcance.derivacion.aplican",
	"alcance.derivacion.y_mas",
	"alcance.derivacion.proximas",
	"alcance.derivacion.ver_controles",
	"alcance.derivacion.limpiar",

	// Alcance: los datos que hay que reunir, y quien los pide.
	"alcance.campos.titulo",
	"alcance.campos.intro",
	"alcance.campos.obligatorio",
	"alcance.campos.lo_piden",

	// El por que de cada veredicto.
	"derivacion.sin_condiciones",
	"derivacion.respondiste_si",
	"derivacion.respondiste_no",
	"derivacion.sin_responder",
	"derivacion.respuesta_contradictoria",
	"derivacion.pregunta_desconocida",
	"derivacion.entregable_huerfano",
	"derivacion.lo_pide_y_aplica",
	"derivacion.lo_pide_y_no_aplica",
	"derivacion.lo_pide",

	// Tablas de Controles y Certificados.
	"filtro.etiqueta",
	"filtro.todos",
	"tabla.intro.controles",
	"tabla.intro.certificados",
	"tabla.mostrando",
	"tabla.sin_resultados",
	"tabla.anterior",
	"tabla.siguiente",
	"tabla.volver_alcance",
	"columna.id",
	"columna.paquete",
	"columna.estado",
	"columna.porque",

	// Errores de la peticion.
	"error.no_encontrado",
	"error.consulta_larga",
}

// ClavesDeCatalogo devuelve, ordenadas y sin repetidos, TODAS las claves de
// interfaz que esta superficie puede pedirle al catalogo.
//
// Tres familias, y dos de ellas se calculan en vez de escribirse:
//
//	las fijas          las que escriben las plantillas y el codigo de aqui
//	las del modelo     titulo y "por que esta vacia" de cada pantalla, sacadas
//	                   de nucleo/pantalla. Se calculan para que una septima
//	                   pantalla en el nucleo aparezca aqui sola
//	las de columna     una por columna conocida de las tablas
func ClavesDeCatalogo() []string {
	vistas := make(map[string]bool, len(clavesFijas)+32)
	anadir := func(c string) {
		if c != "" {
			vistas[c] = true
		}
	}
	for _, c := range clavesFijas {
		anadir(c)
	}
	for _, e := range []Estado{Aplica, NoAplica, Pendiente} {
		anadir(e.Clave())
	}
	for _, c := range columnasEnOrden {
		anadir("columna." + c)
	}
	// Los rotulos de los pasos del camino guiado, que la barra lateral pinta.
	// SE CALCULAN de camino.Canonico() y no se escriben, por lo mismo que las
	// pantallas se calculan de nucleo/pantalla: el dia que el camino gane un
	// paso, su rotulo entra en este contrato solo y el catalogo se pone rojo
	// hasta que alguien lo traduzca. Escribirlos seria una segunda copia del
	// camino, y la segunda copia es la que se queda vieja.
	//
	// Se declara el camino CANONICO porque es el que monta el producto. Quien
	// construya la superficie con otro camino tiene que cubrir los rotulos de
	// SU camino, y eso ya lo dice Opciones.Pasos.
	for _, p := range camino.Canonico() {
		anadir(p.Titulo)
	}
	// Los veredictos de la vigilancia del planificador. Se piden al nucleo
	// para que un estado nuevo alli aparezca aqui solo, igual que las seis
	// pantallas.
	for _, c := range pantalla.ClavesDelPlanificador() {
		anadir(c)
	}
	// El modelo sin corpus trae las seis pantallas con su titulo y su
	// explicacion de por que estan vacias, que son las claves de la familia
	// "pantalla.". Es la unica forma de derivarlas sin repetirlas a mano.
	for _, p := range pantalla.Derivar(nil) {
		anadir(p.Titulo)
		anadir(p.PorQue)
	}
	out := make([]string, 0, len(vistas))
	for c := range vistas {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}
