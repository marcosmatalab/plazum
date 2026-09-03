package pantallas

import (
	"errors"
	"testing"
)

// LA GUARDA DEL ENLACE QUE SACA DE PLAZUM, con su tabla y sus dos direcciones.
//
// # De donde sale
//
// De una alerta abierta de CodeQL (go/bad-redirect-check) en este fichero. Lo
// interesante no es la alerta: es que el autor YA HABIA PENSADO EN EL ATAQUE.
// El comentario de validarCamino explicaba exactamente que con dos barras el
// navegador lee otro anfitrion, y la comprobacion rechazaba "//evil.example".
// Lo que dejaba pasar era `/\evil.example`, que Chrome y Firefox normalizan a
// "//evil.example" ANTES de resolver el destino.
//
// Media guarda: la forma que el autor tenia en la cabeza estaba cerrada y su
// hermana no. Es el invariante 8 aplicado a una cadena, y es la SEGUNDA vez que
// este arbol la pisa (la primera fue pantallas.go, misma regla, ya marcada como
// arreglada).
//
// # Por que hay tabla y no un caso
//
// Porque el fallo de esta familia es exactamente «cerre la forma que se me
// ocurrio». Un caso comprueba la forma que se te ocurrio.
//
// # Y por que hay controles POSITIVOS
//
// Porque una guarda de rechazo sin ellos acaba diciendo que no a todo, y
// entonces el enlace de vuelta al camino no se pinta en ninguna pantalla y nadie
// se entera: el sintoma de una guarda demasiado estricta aqui no es un error, es
// una barra de navegacion que desaparece. Las rutas reales del producto salen de
// camino.Canonico() en la puerta de al lado; aqui van las formas.

func TestElEnlaceDelCaminoNoPuedeSacarDePlazum(t *testing.T) {
	casos := []struct {
		que    string
		ruta   string
		valida bool
	}{
		// LOS RECHAZOS. Cada uno con la razon por la que llega hasta aqui.
		{"protocolo-relativa, la clasica", "//evil.example", false},
		{"protocolo-relativa con camino detras", "//evil.example/acta/", false},
		{"contrabarra, que el navegador normaliza a doble barra", `/\evil.example`, false},
		{"contrabarra con camino detras", `/\evil.example/acta/`, false},
		{"barra y contrabarra", `/\/evil.example`, false},
		{"contrabarra y barra", `/\/`, false},
		{"barra escrita en porcentaje", "/%2fevil.example", false},
		{"contrabarra escrita en porcentaje", "/%5cevil.example", false},
		{"contrabarra en porcentaje, en mayusculas", "/%5Cevil.example", false},
		{"absoluta con esquema", "https://evil.example/", false},
		{"absoluta sin esquema pero con dos puntos", "javascript:alert(1)", false},
		{"relativa sin barra", "camino/", false},
		{"relativa que sube", "../camino/", false},

		// LOS CONTROLES POSITIVOS. Sin ellos, `return false` pasaria la mitad
		// de arriba y esta guarda no dejaria pintar ni un enlace.
		{"la raiz del sitio", "/", true},
		{"la pantalla del camino", "/camino/", true},
		{"el calendario", "/calendario/", true},
		{"la revision de accesos", "/uar/", true},
		{"el acta", "/acta/", true},
		{"una ruta con consulta", "/camino/?ambito=publico", true},
		{"una ruta montada bajo prefijo", "/ui/camino/", true},
		{"una ruta con un guion bajo detras", "/_interno/camino", true},
	}
	rechazos, aceptaciones := 0, 0
	for _, c := range casos {
		t.Run(c.que, func(t *testing.T) {
			err := validarCamino(c.ruta, "camino.titulo")
			if c.valida && err != nil {
				t.Errorf("la ruta %q es de este sitio y la guarda la rechaza: %v.\n"+
					"  Una guarda que dice que no a un destino legitimo no protege, quita el "+
					"enlace de vuelta al camino de una pantalla entera.", c.ruta, err)
			}
			if !c.valida && err == nil {
				t.Errorf("la ruta %q sale de plazum y la guarda la deja pasar.\n"+
					"  Con ella puesta, el enlace que existe para no perder a nadie lleva a "+
					"otro anfitrion.", c.ruta)
			}
			if !c.valida && err != nil && !errors.Is(err, ErrCamino) {
				t.Errorf("la ruta %q se rechaza con un error que no es ErrCamino (%v): quien "+
					"lo compruebe con errors.Is no lo vera", c.ruta, err)
			}
		})
		if c.valida {
			aceptaciones++
		} else {
			rechazos++
		}
	}
	// LAS DOS DIRECCIONES SE HAN RECORRIDO DE VERDAD. Si la tabla se queda sin
	// una de las dos mitades, este test seguiria verde comprobando media
	// propiedad, que es exactamente el fallo que lo trae.
	if rechazos < 10 || aceptaciones < 5 {
		t.Fatalf("la tabla trae %d rechazos y %d aceptaciones: le falta una de las dos "+
			"direcciones y esta comprobando media guarda", rechazos, aceptaciones)
	}
}

// EL VALOR CERO Y LA MITAD, que son las otras dos formas de esta frontera.
//
// Invariante 8: las dos vacias son «no hay entrada de menu», que es lo
// restrictivo; UNA sola vacia es medio enlace, y medio enlace pinta o una
// entrada sin palabras o una palabra que no lleva a ningun sitio.
func TestElEnlaceDelCaminoOEstaEnteroONoEsta(t *testing.T) {
	if err := validarCamino("", ""); err != nil {
		t.Errorf("las dos vacias son el valor cero (no pintar nada) y se rechaza: %v", err)
	}
	for _, c := range []struct{ ruta, clave string }{
		{"/camino/", ""},
		{"", "camino.titulo"},
	} {
		if err := validarCamino(c.ruta, c.clave); err == nil {
			t.Errorf("medio enlace (ruta %q, clave %q) se acepta, y pinta o una entrada de "+
				"menu sin palabras o un rotulo que no lleva a ningun sitio", c.ruta, c.clave)
		}
	}
}
