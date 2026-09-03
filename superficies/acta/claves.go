package acta

import (
	"sort"

	"github.com/marcosmatalab/plazum/superficies/camino"
)

// Las claves de catalogo que pide ESTA SUPERFICIE, y solo esta.
//
// POR QUE LA LISTA ES CORTA. Casi todas las palabras de esta pantalla no son
// suyas: los rotulos de cubo, los de reparto, los catorce descargos, los titulos
// del documento y los nombres de las secciones los declara nucleo/acta, porque
// el board pack impreso los usa igual y el papel y la pantalla no pueden titular
// distinto el mismo acta. El inventario del catalogo las pide desde alli.
//
// Aqui queda el MARCO: lo que existe porque hay un navegador delante y no
// existiria en un folio. Un enlace que dice "ver de que sale", un boton de
// volver, las cabeceras de una tabla, y los tres estados que no son "aqui esta
// el acta".
//
// Un test compara esta lista contra las claves que la plantilla pide de verdad,
// EN LAS DOS DIRECCIONES: no puede quedarse corta ni sobrarle nada.
func ClavesDeCatalogo() []string {
	out := append([]string(nil), claves...)
	// Y LAS DEL ARMAZON COMPARTIDO. Se piden a quien las declara
	// (superficies/camino) en vez de copiarlas aqui: si la barra lateral gana
	// un rotulo, las cuatro superficies lo exigen a la vez o ninguna. Los
	// ROTULOS DE LOS PASOS no entran, y no es un olvido: llegan como dato en
	// Opciones.Pasos y los declara el camino, que es quien los tiene.
	out = append(out, camino.ClavesDelArmazon()...)
	sort.Strings(out)
	// Se deduplica porque las dos listas se solapan a proposito en "ui.marca":
	// es del marco de esta pantalla (va en el <title>) y tambien del armazon.
	// Devolverla dos veces romperia el test que compara esta lista contra lo
	// que la plantilla pide de verdad.
	n := 0
	for i, k := range out {
		if i == 0 || k != out[i-1] {
			out[n] = k
			n++
		}
	}
	return out[:n]
}

var claves = []string{
	// Marco, compartidas con las demas superficies.
	"ui.marca",
	"ui.saltar",
	"ui.pie.no_asesoramiento",

	"acta.pantalla.error_render",

	// Sin sesion. Esta pantalla lleva una lista de quien hizo que dentro de la
	// organizacion, asi que no se sirve a quien no ha entrado.
	"acta.pantalla.sin_sesion.titulo",
	"acta.pantalla.sin_sesion.por_que",

	// El estado vacio, con su siguiente paso (puerta D11-b).
	"acta.pantalla.sin_acta.titulo",
	"acta.pantalla.sin_acta.que_es",
	"acta.pantalla.sin_acta.paso",

	// La identidad del documento.
	"acta.pantalla.organizacion",
	"acta.pantalla.periodo",
	"acta.pantalla.identificador",

	// La derivacion a un clic (puerta D11-c).
	"acta.pantalla.ver_derivacion",
	"acta.pantalla.volver",
	"acta.pantalla.no_existe",
	"acta.pantalla.abierta",
	"acta.pantalla.columna_clave",
	"acta.pantalla.columna_que",
	"acta.pantalla.columna_nota",
	"acta.pantalla.vacio",

	// Avisos de la ley de conservacion.
	"acta.pantalla.descuadre",
	"acta.pantalla.descuadre_reparto",
}
