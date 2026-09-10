package documentos

import "sort"

// Las claves de catalogo que pide esta superficie.
//
// POR QUE EXISTE ESTA LISTA. `Faltantes(idioma)` contesta «¿al ingles le falta
// lo que el espanol tiene?», que es otra pregunta. La que rompe la pantalla es
// «¿cubre el catalogo lo que la interfaz pide?», y a esa contesta esto. Un
// catalogo completo en los dos idiomas y sin `documentos.titulo` da una pestana
// que pone literalmente «documentos.titulo».
//
// Un test compara esta lista contra las claves que la plantilla pide de verdad,
// EN LAS DOS DIRECCIONES: no puede quedarse corta ni sobrarle nada.
//
// AQUI NO VA CONTENIDO DEL CLIENTE NI DEL CORPUS. El nombre de un fichero, la
// cita de una politica y el titulo de una obligacion viajan tal cual: traducir
// las palabras de una norma crea obra derivada, y traducir las de una persona es
// reescribir lo que dijo.
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

	"documentos.titulo",
	"documentos.error.render",

	// SIN SESION. La pantalla no cuenta si hay documentos: solo dice que hay
	// que entrar y por que.
	"documentos.sin_sesion.titulo",
	"documentos.sin_sesion.por_que",

	// SIN ALMACEN: esta instalacion no sabe guardar documentos. Es el valor
	// cero del puerto y se dice, en vez de pintar un boton que contesta 500.
	"documentos.sin_almacen.titulo",
	"documentos.sin_almacen.por_que",

	// SIN TOKEN: no se pinta el formulario y se dice por que.
	"documentos.sin_token",

	// EL ALMACEN QUE NO SE LEE. La tercera forma de la nada en la pantalla:
	// «no he podido mirar» no es «no has subido nada».
	"documentos.ilegible",

	// EL ESTADO VACIO, con su siguiente paso (puerta D11-b).
	"documentos.vacio.titulo",
	"documentos.vacio.que_hacer",

	// EL FORMULARIO.
	"documentos.subir.titulo",
	"documentos.subir.explica",
	"documentos.subir.etiqueta",
	"documentos.subir.boton",
	"documentos.subir.formatos",

	// LOS RECHAZOS, uno por clase y cada uno con su arreglo dentro. Se piden
	// por clave y nunca imprimiendo el error del nucleo: ese lo escribe
	// adaptadores/ingesta en castellano, y pintarlo tal cual en la pagina
	// inglesa es el defecto que ya se pago una vez.
	"documentos.subir.sin_almacen",
	"documentos.subir.falta_fichero",
	"documentos.subir.vacio",
	"documentos.subir.demasiado_grande",
	"documentos.subir.no_se_lee",
	"documentos.subir.no_se_entiende",
	"documentos.subir.formato",
	"documentos.subir.cifrado",
	"documentos.subir.sin_contenido",
	"documentos.subir.no_se_guarda",

	// LA LISTA DE LO SUBIDO.
	"documentos.lista.titulo",
	"documentos.lista.paginas",
	"documentos.lista.sin_paginas",
	"documentos.lista.fragmentos",
	"documentos.lista.truncado",
	"documentos.lista.nombre_enganoso",

	// LOS HALLAZGOS, y su descargo, que es la frase que sostiene esta pantalla
	// entera.
	"documentos.hallazgos.titulo",
	"documentos.hallazgos.explica",
	"documentos.hallazgos.descargo",
	"documentos.hallazgos.ninguno",
	"documentos.hallazgos.en",
	"documentos.hallazgos.pagina",
	"documentos.hallazgos.sin_pagina",

	// LA FICHA DEL DOCUMENTO (pieza 7): lo propuesto, lo aceptado y sus cuatro
	// campos. El NOMBRE del campo es vocabulario de plazum y se traduce; el
	// VALOR son palabras del documento del cliente y no pasa por aqui.
	"documentos.ficha.titulo",
	"documentos.ficha.explica",
	"documentos.ficha.descargo",
	"documentos.ficha.aceptar",
	"documentos.ficha.acepto",
	"documentos.ficha.ninguno",
	"documentos.ficha.campo.fecha",
	"documentos.ficha.campo.alcance",
	"documentos.ficha.campo.firmante",
	"documentos.ficha.campo.caducidad",

	// LOS RECHAZOS DE LA CONFIRMACION. `no_casa` es la tercera forma de la nada
	// en esta ruta: llega un dato, se entiende, y NO es ninguna de las
	// propuestas de esta cuenta. No es una ausencia y no se trata como tal.
	"documentos.aceptar.no_se_lee",
	"documentos.aceptar.falta_campo",
	"documentos.aceptar.no_casa",
	"documentos.aceptar.no_se_guarda",
}
