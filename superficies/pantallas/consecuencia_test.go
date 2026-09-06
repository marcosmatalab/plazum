package pantallas

import (
	"strings"
	"testing"
)

// SIN CALCULADORA NO SE AFIRMA NADA SOBRE LA CONSECUENCIA.
//
// # De donde sale esta puerta, y no fue de leer el codigo
//
// De una mutacion que SOBREVIVIO (M12, 06-09-2026). Cambiado `{{if
// .HayConsecuencia}}` por `{{if true}}`, la suite entera de este paquete siguio
// EN VERDE. Con esa mutacion, una instalacion que no sabe calcular consecuencias
// pintaria en LAS DIECINUEVE preguntas la frase «contestar que si aqui no activa
// ninguna obligacion nueva».
//
// O sea: una afirmacion sobre el cumplimiento de alguien, hecha por un campo sin
// rellenar, en la pantalla donde esa persona esta decidiendo que declara sobre su
// organizacion. Es exactamente la familia del 8 de 72 y de M47: el valor cero
// leido como un hecho.
//
// # Las dos direcciones, y la segunda es la que la hace valer
//
// Sin calculadora, NADA. Con calculadora, la frase. Sin la segunda mitad, un
// producto que hubiera perdido la pieza entera pasaria esta puerta con nota,
// porque tampoco pintaria nada.
func TestSinCalculadoraLaPantallaNoAfirmaNadaSobreLaConsecuencia(t *testing.T) {
	// El texto que NO puede aparecer es el de la rama del cero, que es la que
	// afirma. Sale del catalogo por su CLAVE y no escrito aqui: un literal se
	// queda viejo el dia que alguien mejore la redaccion y la puerta pasaria a
	// buscar algo que no existe, o sea a dar verde sobre cualquier cosa.
	s, cat := superficie(t, corpusDemo())
	_, cuerpo := pedir(t, s, "/alcance?"+ParamVer+"="+VerTodas)
	pedidas := cat.vistas()
	if pedidas["alcance.consecuencia.ninguna"] > 0 || pedidas["alcance.consecuencia.activa"] > 0 {
		t.Errorf(`sin quien calcule consecuencias, la pantalla las pide igual.

  Con la rama mal escrita, esto pinta en las diecinueve preguntas que contestar
  que si no activa ninguna obligacion: una afirmacion sobre el cumplimiento de
  alguien hecha por un campo sin rellenar, en la pantalla donde esa persona
  decide que declara sobre su organizacion.`)
	}
	if strings.Contains(cuerpo, `class="consecuencia"`) {
		t.Errorf("sin calculadora se pinta el bloque de la consecuencia:\n%s", cuerpo)
	}

	// LA DIRECCION CONTRARIA. Sin esto, un producto que hubiera perdido la pieza
	// entera aprobaria: tampoco pintaria nada.
	for _, n := range []int{5, 0} {
		s, cat := superficie(t, corpusDemo(), conConsecuencias(consecuenciasFalsas{n: n}))
		_, cuerpo := pedir(t, s, "/alcance?"+ParamVer+"="+VerTodas)
		if !strings.Contains(cuerpo, `class="consecuencia"`) {
			t.Errorf("con calculadora (n=%d) NO se pinta la consecuencia:\n%s", n, cuerpo)
		}
		clave := "alcance.consecuencia.activa"
		if n == 0 {
			clave = "alcance.consecuencia.ninguna"
		}
		if cat.vistas()[clave] == 0 {
			t.Errorf("con n=%d la pantalla no pide %q, asi que la rama que corresponde no se "+
				"esta pintando", n, clave)
		}
	}
}
