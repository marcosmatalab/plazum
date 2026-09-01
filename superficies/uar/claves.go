package uar

import "sort"

// Las claves de catalogo que pide esta superficie.
//
// POR QUE EXISTE ESTA LISTA, y es lo mismo que en superficies/pantallas:
// `Faltantes(idioma)` responde "¿al ingles le falta lo que el espanol tiene?",
// que es otra pregunta. La que rompe la pantalla es "¿cubre el catalogo lo que
// la interfaz pide?", y a esa responde esto. Un catalogo completo en los dos
// idiomas y sin "uar.decidir.boton" da un boton que pone literalmente
// "uar.decidir.boton", y Faltantes da verde.
//
// Un test compara esta lista contra las claves que la plantilla pide de verdad,
// EN LAS DOS DIRECCIONES: no puede quedarse corta ni sobrarle nada.
//
// Aqui NO va contenido del censo del cliente. El rotulo de una cuenta y el
// nombre de un permiso salen del fichero de su IdP y viajan tal cual: traducir
// el nombre de un permiso ajeno es inventarselo.
func ClavesDeCatalogo() []string {
	out := append([]string(nil), claves...)
	sort.Strings(out)
	return out
}

var claves = []string{
	// Marco, compartidas con las demas superficies.
	"ui.marca",
	"ui.saltar",
	"ui.pie.no_asesoramiento",

	"uar.titulo",
	"uar.error.render",
	"uar.sin_token",
	"uar.aviso.sin_autor",

	"uar.sin_sesion.titulo",
	"uar.sin_sesion.por_que",

	// El estado vacio, con su siguiente paso (puerta D11-b).
	"uar.sin_campana.titulo",
	"uar.sin_campana.que_es",
	"uar.sin_campana.paso1",
	"uar.sin_campana.paso2",
	"uar.sin_campana.por_que_el_fichero",

	"uar.identidad.titulo",
	"uar.identidad.campana",
	"uar.identidad.sello",
	"uar.identidad.hash",
	"uar.identidad.reproducible",

	"uar.cubos.titulo",
	"uar.cubos.quitar_filtro",
	"uar.cubo.aprobada",
	"uar.cubo.revocada",
	"uar.cubo.delegada",
	"uar.cubo.sin_revisar",

	// El descargo. Su valor en espanol tiene que ser LETRA POR LETRA el de
	// accesos.LaFraseDeLoNoRevisado, y hay test: una frase que vive en dos
	// sitios se corrige en uno.
	"uar.no_consta",
	"uar.no_revisado.cuantos",
	"uar.sin_revisor",

	"uar.ilegibles.titulo",
	"uar.ilegibles.bloquean",
	"uar.ilegibles.linea",
	"uar.excusar.desde",
	"uar.excusar.hasta",
	"uar.excusar.motivo",
	"uar.excusar.boton",
	"uar.excusar.no_es_numero",
	"uar.excusar.deja_rastro",
	"uar.excusas.titulo",
	"uar.excusas.una",
	"uar.duplicadas",

	"uar.tabla.titulo",
	"uar.tabla.pie",
	"uar.tabla.cuenta",
	"uar.tabla.permiso",
	"uar.tabla.rotulo",
	"uar.tabla.estado",
	"uar.tabla.revisor",
	"uar.tabla.decision",
	"uar.tabla.sin_revisor",
	"uar.tabla.decidio",

	"uar.decidir.veredicto",
	"uar.decidir.aprobar",
	"uar.decidir.revocar",
	"uar.decidir.delegar",
	"uar.decidir.motivo",
	"uar.decidir.a",
	"uar.decidir.boton",

	"uar.cierre.titulo",
	"uar.cierre.que_es",
	"uar.cierre.boton",
	"uar.cierre.hecho",
}
