package main

import (
	"os"
	"strings"
	"testing"
)

// LA PRIMERA PANTALLA SENALA EL PRODUCTO.
//
// # De donde sale esta puerta, y no fue de leer el codigo
//
// De la pasada del comprador sobre el binario PUBLICADO (R4, 06-09-2026).
// Descargada la release, comprobada su firma, instalado el corpus y arrancada la
// web, el recorrido funciona entero: los seis pasos contestan 200 y el alta del
// primer administrador va. Lo que fallaba estaba antes de todo eso.
//
// `plazum serve` vivia en «el resto», debajo de nueve ordenes, mientras arriba
// salian `calendario` y `escalado`, que son dos vistas sueltas de lo mismo. O
// sea: la primera pantalla que ve quien se descarga esto NO SENALABA EL
// PRODUCTO. El camino guiado entero -- la entrevista, el calendario, la
// derivacion, el acta, la revision de accesos y el escalado -- vive detras de esa
// orden, y el unico sitio donde se nombraba era la salida de
// `corpus --instalar`, que solo se lee si ya has acertado con el primer paso.
//
// Es lo que prohibe D11-a en su forma mas barata de cometer y mas cara de notar:
// nada esta roto, todo contesta 200, y el comprador se va habiendo visto la
// mitad del producto por el sitio mas incomodo.
//
// # Que vigila, y por que por POSICION y no por presencia
//
// Que `serve` este en el bloque de «empieza por aqui» y no mas abajo. Comprobar
// que aparece «en algun sitio» no vigila nada: aparecia, y ese era el problema.
func TestLaPrimeraPantallaSenalaElProducto(t *testing.T) {
	ayuda := ayudaDeLaOrdenSuelta(t)

	i := strings.Index(ayuda, "empieza por aqui:")
	j := strings.Index(ayuda, "el resto:")
	if i < 0 || j < 0 || j < i {
		t.Fatalf("la primera pantalla ha dejado de tener sus dos bloques, asi que esta puerta "+
			"no puede mirar la posicion de nada:\n%s", ayuda)
	}
	arriba := ayuda[i:j]

	// LO QUE TIENE QUE ESTAR ARRIBA, con su motivo al lado. Son los dos que
	// convierten una descarga en el producto: el corpus (sin el, el binario no
	// trae los marcos) y la web (que es donde estan los seis pasos).
	for _, orden := range []string{"plazum corpus --instalar", "plazum serve"} {
		if !strings.Contains(arriba, orden) {
			t.Errorf(`«%s» no esta en el bloque de «empieza por aqui».

  La primera pantalla de un producto que se descarga es la unica formacion que
  va a recibir su comprador. Si el camino guiado no se nombra ahi, se llega a el
  por casualidad o no se llega, y eso es exactamente lo que prohibe D11-a.
--- lo que hay arriba ---
%s`, orden, arriba)
		}
	}

	// CONTROL NEGATIVO DEL DETECTOR: una orden que SI esta abajo tiene que
	// salir como ausente de arriba. Sin esto, un recorte mal hecho (que se
	// llevara la pagina entera) daria verde sobre cualquier cosa.
	if strings.Contains(arriba, "plazum doctor") {
		t.Error("el recorte de «empieza por aqui» se ha llevado tambien «el resto»: esta " +
			"puerta estaria dando por bueno cualquier orden este donde este")
	}
}

// ayudaDeLaOrdenSuelta devuelve lo que imprime `plazum` a secas.
//
// SE LEE DEL FUENTE Y NO SE EJECUTA EL BINARIO, y el limite se dice: esta puerta
// vigila el ORDEN DE LOS BLOQUES tal como estan escritos, no lo que saldria si
// alguien metiera un `if` que los reordenara en tiempo de ejecucion. Ejecutar
// exigiria compilar el binario dentro de este paquete (medio segundo por
// ejecucion de la suite) para vigilar una funcion que no tiene ni una rama.
//
// El dia que esa funcion gane una rama, esta puerta deja de valer y hay que
// cambiarla por una que ejecute. Queda dicho aqui para que se note.
func ayudaDeLaOrdenSuelta(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("leyendo main.go: %v", err)
	}
	// Solo las lineas que IMPRIMEN, para que un comentario que nombre una orden
	// no cuente como que la orden sale en pantalla. Es el mismo falso positivo
	// que persigue el detector de supresiones.
	var out []string
	for _, l := range strings.Split(string(b), "\n") {
		if !strings.Contains(l, "fmt.Fprintln(os.Stderr,") {
			continue
		}
		out = append(out, l)
	}
	if len(out) < 20 {
		t.Fatalf("solo se han encontrado %d lineas que impriman en main.go: esta puerta "+
			"estaria mirando el vacio", len(out))
	}
	return strings.Join(out, "\n")
}
