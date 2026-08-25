package main

// El acumulador de texto que comparten los dos parsers.
//
// Las dos fuentes sirven el cuerpo del articulado como HTML incrustado, y las
// dos hay que recorrerlas POR TOKENS y no deserializando a una estructura: al
// deserializar se pierde el orden entre el texto y los elementos en linea (un
// enlace de referencia, un <strong>, un <span>), y con el orden se pierde el
// sentido de la frase. Un texto legal reordenado es peor que ninguno.
//
// Lo que cambia entre fuentes es DONDE termina un parrafo. Eso lo decide cada
// parser; aqui solo se acumula, se normaliza y se arman las filas de tabla.

import (
	"strings"
)

// trozo es un parrafo ya normalizado, con la clase CSS que traia de la fuente.
// La clase no es decoracion: es lo que distingue el encabezado del articulo de
// su cuerpo, y la nota al pie del texto normativo.
type trozo struct {
	Clase string
	Texto string
	Nota  bool // venia dentro de una nota (blockquote)
	Prof  int  // profundidad del elemento que lo cerro
}

type acumulador struct {
	trozos  []trozo
	buf     strings.Builder
	clase   string
	celdas  []string
	enTabla int
	enNota  int
}

// marcarClase deja apuntada la clase del proximo trozo que se cierre. Se llama
// al ABRIR el elemento porque al cerrarlo el atributo ya no viaja en el token.
func (ac *acumulador) marcarClase(c string) {
	if c != "" {
		ac.clase = c
	}
}

func (ac *acumulador) texto(b []byte) { ac.buf.Write(b) }

// literal mete texto propio de la herramienta en el flujo (el aviso de imagen
// omitida, los separadores). Va marcado entre corchetes para que quien lea la
// extraccion sepa que esa parte no la escribio el legislador.
func (ac *acumulador) literal(s string) { ac.buf.WriteString(s) }

// cerrar vuelca lo acumulado. Dentro de una tabla el volcado es una celda; fuera
// es un parrafo.
func (ac *acumulador) cerrar(prof int) {
	t := normalizarTexto(ac.buf.String())
	ac.buf.Reset()
	clase := ac.clase
	ac.clase = ""
	if t == "" {
		return
	}
	if ac.enTabla > 0 {
		ac.celdas = append(ac.celdas, t)
		return
	}
	ac.trozos = append(ac.trozos, trozo{Clase: clase, Texto: t, Nota: ac.enNota > 0, Prof: prof})
}

// emitirFila cierra una fila de tabla como un solo parrafo con las celdas
// separadas. Una tabla de una norma (la de niveles del anexo, la de plazos) es
// contenido normativo, y perderla por no saber pintarla seria perder obligacion.
func (ac *acumulador) emitirFila(prof int) {
	ac.cerrar(prof) // la celda que quedara abierta
	if len(ac.celdas) == 0 {
		return
	}
	fila := strings.Join(ac.celdas, " | ")
	ac.celdas = ac.celdas[:0]
	if fila == "" {
		return
	}
	ac.trozos = append(ac.trozos, trozo{Clase: "tabla", Texto: fila, Nota: ac.enNota > 0, Prof: prof})
}

// tirar descarta lo acumulado sin emitirlo. Se usa al entrar en una zona cuyo
// texto no es del articulo.
func (ac *acumulador) tirar() {
	ac.buf.Reset()
	ac.clase = ""
}
