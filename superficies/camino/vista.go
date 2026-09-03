package camino

import (
	"embed"
	"io/fs"
	"sort"
)

//go:embed plantillas
var plantillasFS embed.FS

// Plantillas expone el sistema de ficheros embebido, por si alguien quiere
// montar otro motor de render.
func Plantillas() fs.FS { return plantillasFS }

// Vista es lo unico que ve la plantilla.
//
// La regla de siempre: Titulo, Verbo y los rotulos son CLAVES de catalogo;
// Comando es texto y viaja tal cual, porque una orden de terminal no se
// traduce (traducir "--corpus" daria una orden que no existe).
type Vista struct {
	Idioma   string
	Estatico string
	Titulo   string // clave
	Pasos    []PasoVista

	// Inicio es la raiz del sitio, con su barra. Lo pide el armazon
	// compartido para el enlace de la marca.
	Inicio string
	// Tira es EL CAMINO en la barra lateral. Vacia no pinta barra, que es el
	// valor cero restrictivo.
	//
	// EN ESTA PANTALLA NO SE MARCA NINGUN PASO, y no es un olvido: el camino
	// no es uno de sus propios pasos. Marcar el primero "por si acaso" seria
	// decirle al operador que esta donde no esta.
	Tira []PasoTira
}

// PasoVista es un paso pintable.
type PasoVista struct {
	Numero int
	Total  int
	ID     string
	Titulo string // clave
	Verbo  string // clave
	// URL vacia significa que el paso todavia no es una pantalla. Entonces
	// se pinta el Comando, y nunca un enlace muerto.
	URL     string
	Comando string
}

// EsPantalla dice si este paso se recorre sin salir del servidor.
func (p PasoVista) EsPantalla() bool { return p.URL != "" }

// ClavesDeCatalogo son las claves que pide esta superficie.
//
// POR QUE EXISTE ESTA LISTA, igual que en las demas superficies:
// Faltantes(idioma) responde "¿al ingles le falta lo que el espanol tiene?", y
// la pregunta que rompe la pantalla es otra, "¿cubre el catalogo lo que la
// interfaz pide?". Un catalogo completo en los dos idiomas y sin
// "camino.paso.acta" pinta un paso que se llama literalmente
// "camino.paso.acta", y Faltantes da verde.
//
// La lista NO se escribe entera a mano: los rotulos y verbos de los pasos salen
// del camino canonico, que es quien los declara. Escribirlos aqui otra vez seria
// una segunda copia, y una segunda copia se desincroniza el dia que alguien
// anade un paso.
func ClavesDeCatalogo() []string {
	out := append([]string(nil), clavesDelMarco...)
	// Y LAS DEL ARMAZON COMPARTIDO, que esta pantalla pinta como las demas.
	// Se piden a quien las declara en vez de copiarlas: si el armazon gana un
	// rotulo, las cuatro superficies lo exigen a la vez o ninguna.
	out = append(out, ClavesDelArmazon()...)
	for _, p := range canonico {
		out = append(out, p.Titulo, p.Verbo)
	}
	// Se DEDUPLICA porque las listas se solapan a proposito: ClaveTitulo esta
	// en el marco de esta pantalla y tambien en el armazon, que la usa de
	// rotulo de seccion. Devolver la clave dos veces haria fallar al test que
	// compara esta lista contra lo que la plantilla pide de verdad.
	sort.Strings(out)
	return unicas(out)
}

// unicas quita repetidos de una lista YA ORDENADA.
func unicas(xs []string) []string {
	out := xs[:0]
	for i, x := range xs {
		if i == 0 || x != xs[i-1] {
			out = append(out, x)
		}
	}
	return out
}

// clavesDelMarco son las que pone la pantalla, no los pasos.
var clavesDelMarco = []string{
	// Marco, compartidas con las demas superficies.
	"ui.marca",
	"ui.saltar",
	"ui.pie.no_asesoramiento",

	ClaveTitulo,
	"camino.que_es",
	"camino.paso_n",
	"camino.ir",
	"camino.todavia_no",
	"camino.sin_progreso",
	"camino.error.render",
	"camino.error.consulta_larga",
}
