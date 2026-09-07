package pantallas

import (
	"regexp"
	"testing"
)

// LA TABLA PINTA LAS FILAS QUE DICE PINTAR.
//
// # De donde sale esta puerta: del camino barato de otra
//
// La encontro la mutacion M1 de la pasada 2 del 08-09-2026, y no buscando un
// fallo de esta pantalla sino preguntando **como se aprueba sin arreglar nada**
// la medida nueva del TTFV. Ese modelo cobra desde hoy el PESO del contenido, o
// sea cuantos trozos de prosa pone delante cada paso; y la forma mas barata de
// bajar ese numero no es mejorar el producto, es **pintar menos filas**.
//
// La mutacion dejo la tabla en tres filas de doscientas. El TTFV bajo de 21m7s a
// 16m52s y la marca estructural del paso seguia ahi —el `<div class="marco-tabla">`
// existe con tres filas igual que con doscientas—, asi que la unica que protesto
// fue la mitad de ABAJO del techo del TTFV: «el numero ha bajado y el techo no».
// Eso obliga a venir a bajar el techo, que es donde una persona lo notaria, pero
// **no lo impide**: vaciar la tabla y bajar el techo en el mismo commit pasa.
//
// # Lo que se afirma, que ademas es una promesa de la propia pagina
//
// La tabla escribe «De la 1 a la 200, de 549» con `tabla.mostrando`, o sea que
// AFIRMA cuantas filas ensena. Esta puerta comprueba que las pinte. No es una
// guarda inventada para tapar el camino barato: es la promesa que la pagina ya
// hacia y que no comprobaba nadie, y cerrar el camino barato sale de propina.
//
// Es la disciplina de la casa aplicada a una tabla: **un cardinal publicado se
// ata a lo que lo compone**. Aqui el cardinal es el rotulo y lo que lo compone
// son las filas.
func TestLaTablaPintaLasFilasQueDicePintar(t *testing.T) {
	s, cat := superficie(t, corpusDemo())
	w, cuerpo := pedir(t, s, "/controles")
	if w.Code != 200 {
		t.Fatalf("codigo %d", w.Code)
	}
	desde, hasta, total := rangoQueAfirma(t, cat)
	filas := filasPintadas(cuerpo)

	if quiero := hasta - desde + 1; filas != quiero {
		t.Errorf("la tabla dice ensenar de la %d a la %d (de %d) y pinta %d filas.\n"+
			"  La pagina AFIRMA un cardinal y no lo cumple: quien lo lea contara mal lo que\n"+
			"  tiene delante, y una tabla que ensena menos de lo que dice esconde\n"+
			"  obligaciones sin decirlo.\n"+
			"  Y hay un segundo motivo para que esto este vigilado: el TTFV cobra desde el\n"+
			"  08-09-2026 el peso del contenido, asi que pintar menos filas BAJA esa medida\n"+
			"  sin mejorar el producto. Esta puerta es lo que impide ese atajo.",
			desde, hasta, total, filas)
	}
	if filas == 0 {
		t.Error("la tabla no pinta ni una fila: este recorrido estaria midiendo el vacio")
	}
	if total < filas {
		t.Errorf("la tabla dice que hay %d obligaciones en total y pinta %d filas: el total "+
			"no puede ser menor que la pagina", total, filas)
	}
	t.Logf("la tabla afirma de la %d a la %d de %d, y pinta %d filas", desde, hasta, total, filas)
}

// TestElContadorDeFilasSabePonerseRojo es el control negativo.
//
// El contador de filas es el unico trozo propio de la puerta de arriba: si
// contara siempre lo mismo que el rotulo, o siempre cero, la comparacion casaria
// sola. Se le ponen delante trozos de HTML sinteticos.
func TestElContadorDeFilasSabePonerseRojo(t *testing.T) {
	casos := []struct {
		que   string
		html  string
		filas int
	}{
		{"tres filas", `<tr class="e-aplica"><tr class="e-no_aplica"><tr class="e-pendiente">`, 3},
		{"ninguna", `<table><tbody></tbody></table>`, 0},
		// UNA CLASE MAS NO DESCUENTA UNA FILA. Es el fallo exacto que ya tuvo el
		// contador de preguntas del TTFV: buscaba el atributo con su comilla de
		// cierre y dejo de casar el dia que una clase gano un sufijo, contando
		// una de menos durante semanas y en la direccion comoda.
		{"con una clase de mas", `<tr class="e-aplica destacada">`, 1},
	}
	for _, c := range casos {
		if got := filasPintadas(c.html); got != c.filas {
			t.Errorf("%s: el contador ve %d filas y son %d", c.que, got, c.filas)
		}
	}
}

// filasPintadas cuenta las filas de datos de la tabla.
//
// SIN LA COMILLA DE CIERRE, a proposito y con precedente: el contador de
// preguntas del TTFV buscaba `class="pregunta"` con comilla y dejo de casar el
// dia que la pregunta sugerida gano una clase (`class="pregunta sugerida"`).
// Conto una de menos durante semanas, y nadie lo noto porque el error iba en la
// direccion comoda.
var reFilaDeDatos = regexp.MustCompile(`<tr class="e-`)

func filasPintadas(html string) int {
	// El <thead> no trae `class="e-`, asi que no hace falta recortarlo: la
	// cabecera de esta tabla son `<th scope="col">` sueltos dentro de un `<tr>`
	// sin clase.
	return len(reFilaDeDatos.FindAllString(html, -1))
}

// rangoQueAfirma saca del catalogo los tres numeros que la pagina publica.
//
// SE SACAN DE LO QUE LA PAGINA PIDIO AL CATALOGO y no del HTML pintado, y esa
// es la mitad que hace honesta la comparacion: si se leyeran los dos del mismo
// HTML, se estaria comparando una cadena consigo misma.
func rangoQueAfirma(t *testing.T, cat *catalogo) (desde, hasta, total int) {
	t.Helper()
	for _, a := range cat.argumentosDe("tabla.mostrando") {
		if len(a) != 3 {
			t.Fatalf("tabla.mostrando se pidio con %d argumentos y son tres "+
				"(desde, hasta, total): %v", len(a), a)
		}
		return aEnteroDelCatalogo(t, a[0]), aEnteroDelCatalogo(t, a[1]),
			aEnteroDelCatalogo(t, a[2])
	}
	t.Fatal("la pagina no pidio `tabla.mostrando`, asi que no afirma cuantas filas ensena.\n" +
		"  Sin ese rotulo la tabla deja de decir de cuantas obligaciones ensena un trozo,\n" +
		"  que es la unica pista de que hay mas detras")
	return 0, 0, 0
}

func aEnteroDelCatalogo(t *testing.T, v any) int {
	t.Helper()
	n, ok := v.(int)
	if !ok {
		t.Fatalf("tabla.mostrando recibio %v (%T) donde iba un entero", v, v)
	}
	return n
}
