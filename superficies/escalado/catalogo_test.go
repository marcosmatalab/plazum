package escalado

// LOS OCHO CUBOS, ATADOS AL NUCLEO POR UNA PUERTA Y YA NO POR UNA CONVENCION.
//
// # El hueco que cierra, con su cardinal
//
// El rotulo castellano de cada cubo es LETRA POR LETRA la constante de
// `nucleo/escalado`, y eso no era casualidad: es lo que conserva la promesa que
// defendia el godoc anterior de `ClavesDeCatalogo`, que la pantalla y la
// terminal no den dos nombres al mismo cubo alli donde de verdad se comparan,
// que es en castellano. Lo que no habia era nada que lo exigiera. Quedo escrito
// en docs/hallazgos-d11.md el 04-09-2026 como P2, con su numero: «8 cadenas
// atadas por convencion y no por puerta».
//
// El acta si tenia la suya desde antes (`nucleo/acta.CadenasDelActa()`,
// contrastada en `adaptadores/catalogo/acta_test.go`), asi que hay molde y no
// hay nada que inventar. Esta es la misma puerta para la otra familia.
//
// # POR QUE VIVE AQUI Y NO EN adaptadores/catalogo
//
// El hallazgo la ponia alli, y aqui esta mejor por dos motivos, no por uno:
//
//  1. EL MAPA VIVE AQUI. `cubos` es de esta superficie a proposito (una clave de
//     catalogo es vocabulario de interfaz y el nucleo no tiene interfaz), asi
//     que la puerta que lo vigila esta pegada a lo que vigila y se lee con ello.
//  2. `adaptadores/catalogo/` no es de esta columna en el tramo 3; `cadenas/`
//     si. Escribirla alli seria salirse de la frontera por comodidad, y ese es
//     el sitio por donde una campana en paralelo se destruye a si misma.
//
// La direccion contraria (una clave `escalado.cubo.*` que el catalogo publique y
// nadie pida) YA la cierra el inventario de `pantalla_test.go`, que cruza en los
// dos sentidos: aqui no se repite.
//
// # QUE SE ATA, EXACTAMENTE
//
// El CASTELLANO al valor de la constante. El ingles NO se puede atar a nada, y
// eso se dice en vez de disimularse: no hay una segunda fuente de verdad en
// ingles contra la que contrastar. Lo que si se exige del ingles es lo unico
// comprobable sin ella: que exista, y que no sea el castellano copiado.

import (
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/catalogo"
	nescalado "github.com/marcosmatalab/plazum/nucleo/escalado"
)

// sinTildesNiEspacios normaliza para comparar dos frases que se escribieron en
// dos ficheros distintos.
//
// SOLO TILDES Y ESPACIOS DE SOBRA. No baja a minusculas ni quita puntuacion: si
// una de las dos escribiera «Sin destinatario» con mayuscula, eso SI es una
// discrepancia que hay que ver, porque es la que se nota al comparar una
// captura con un log.
func sinTildesNiEspacios(s string) string {
	r := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
		"Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U", "Ñ", "N")
	return strings.Join(strings.Fields(r.Replace(s)), " ")
}

func TestElCatalogoDiceDeCadaCuboLoMismoQueElNucleo(t *testing.T) {
	c, err := catalogo.Nuevo()
	if err != nil {
		t.Fatalf("el catalogo embebido no carga, asi que el producto no arranca: %v", err)
	}
	estados := nescalado.EstadosPosibles()
	// SUELO: sin el, un `EstadosPosibles` vaciado dejaria esta puerta verde
	// habiendo recorrido la nada, que es el verde vacio de siempre.
	if len(estados) != 8 {
		t.Fatalf("el nucleo declara %d estados y hoy son 8: o la particion ha cambiado y esta "+
			"puerta no se ha enterado, o esta mirando otra cosa", len(estados))
	}

	for _, e := range estados {
		clave, hay := ClaveDelCubo(e)
		if !hay {
			t.Errorf("el estado %q no tiene rotulo emparejado", e)
			continue
		}
		// EL CASTELLANO, LETRA POR LETRA.
		es := c.Traducir("es", clave)
		if es == clave {
			t.Errorf("el catalogo no tiene %q, asi que la pantalla la pintaria en crudo", clave)
			continue
		}
		if sinTildesNiEspacios(es) != sinTildesNiEspacios(string(e)) {
			t.Errorf("la clave %q dice una cosa en la pantalla y otra en la terminal.\n"+
				"  nucleo:   %q\n  catalogo: %q\n"+
				"  El rotulo castellano de un cubo es la constante del nucleo, y no por "+
				"pereza: es lo que hace que una captura de pantalla y un log se puedan "+
				"comparar. Arreglo: copiar el texto del nucleo a es.json. Y si lo que cambio "+
				"fue la constante, MIRAR TAMBIEN EL INGLES: esta puerta no lo puede "+
				"comprobar, y su commit tiene que decir si se reviso.", clave, string(e), es)
		}
		// EL INGLES EXISTE Y NO ES EL CASTELLANO.
		//
		// No se puede atar a ninguna constante (no hay una segunda fuente de
		// verdad en ingles), asi que se exige lo unico comprobable sin ella. Un
		// rotulo ingles que sea el espanol copiado es media traduccion, y deja
		// al lector ingles viendo «suprimido por una ventana de silencio», que
		// es de donde se venia.
		en := c.Traducir("en", clave)
		if en == clave {
			t.Errorf("la clave %q no tiene ingles", clave)
			continue
		}
		if sinTildesNiEspacios(en) == sinTildesNiEspacios(string(e)) {
			t.Errorf("la clave %q tiene en ingles el mismo texto que en castellano (%q): o no "+
				"se tradujo o se copio", clave, en)
		}
	}
}

// CONTROL NEGATIVO: la comparacion muerde.
//
// Su fallo probable es el de siempre en una comparacion normalizada: normalizar
// de mas hasta que todo se parece a todo. Se prueba con las cuatro formas de
// discrepancia que de verdad pueden aparecer al editar es.json, y con las dos
// que NO son discrepancia y tienen que pasar.
func TestLaComparacionDeCubosDistingueUnaFraseDeOtra(t *testing.T) {
	const constante = "colapsado en un escalon anterior"
	iguales := []string{
		constante,
		"colapsado en un escalón anterior", // la tilde no es discrepancia
		"colapsado  en un   escalon anterior",
	}
	for _, s := range iguales {
		if sinTildesNiEspacios(s) != sinTildesNiEspacios(constante) {
			t.Errorf("%q y %q se dan por distintos y son la misma frase", s, constante)
		}
	}
	distintos := []string{
		"colapsado en un escalon previo",    // sinonimo
		"Colapsado en un escalon anterior",  // mayuscula
		"colapsado en un escalon",           // recortado
		"colapsado en un escalon anterior.", // puntuacion anadida
	}
	for _, s := range distintos {
		if sinTildesNiEspacios(s) == sinTildesNiEspacios(constante) {
			t.Errorf("%q pasa por %q: la comparacion normaliza de mas y no vigila nada",
				s, constante)
		}
	}
}
