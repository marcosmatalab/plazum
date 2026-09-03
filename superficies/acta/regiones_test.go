package acta

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TODA ZONA DESPLAZABLE DE ESTAS TRES PANTALLAS SE ALCANZA CON EL TECLADO.
//
// # El hueco que cierra, y es una junta entre dos puertas que existian
//
// La hoja de la casa da `overflow: auto` y `max-height: 78vh` a `.marco-tabla`:
// cualquier tabla dentro de uno es una REGION CON SCROLL. Con raton se arrastra,
// con teclado no se llega si el marco no es enfocable, y eso es la violacion
// `scrollable-region-focusable` de axe. Habia DOS puertas y entre las dos
// dejaban esta pantalla fuera:
//
//	superficies/pantallas/armazon_test.go   mira `.marco-tabla`, y solo en las
//	                                        cuatro rutas de esa superficie
//	regiones_con_scroll_test.go (raiz)      recorre TODAS las plantillas y solo
//	                                        mira `<pre>`, que era lo que se
//	                                        habia cobrado el dia anterior
//
// El acta tenia su tabla en un `<div class="marco-tabla">` pelado. No la miraba
// ninguna de las dos, y es la tabla en la que se abre cada cifra del acta, o sea
// la razon de ser de la pantalla.
//
// # Por que se estrena sobre el arbol y no sobre una mutacion
//
// Porque nacio ROJA sobre las plantillas de verdad, sin que nadie le metiera
// nada: la del acta no tenia `tabindex` ni nombre. Eso es lo que va a tener que
// hacer el resto de su vida.
//
// # Lo que NO cubre, dicho
//
// Solo recorre las tres superficies de esta columna (acta, calendario,
// escalado). `superficies/uar/plantillas/uar.html` tiene el MISMO fallo, sin
// arreglar, y no es de esta columna este tramo: va como hallazgo en
// docs/hallazgos-d11.md. Y vigila `.marco-tabla`, que es la unica clase con
// scroll propio de la hoja: el dia que se le de a otra, esta puerta no la vera,
// y por eso se dice en vez de suponerse.
var reMarcoTabla = regexp.MustCompile(`(?i)<div[^>]*class="[^"]*marco-tabla[^"]*"[^>]*>`)

func TestTodaZonaDesplazableDeEstasPantallasSeAlcanzaConElTeclado(t *testing.T) {
	// La hoja de estilo es quien decide que se desplaza. Se lee de ahi, no de
	// una lista escrita al lado: si `.marco-tabla` deja de tener overflow, esta
	// puerta esta vigilando un fantasma y hay que enterarse.
	hoja, err := os.ReadFile("../pantallas/estatico/plazum.css")
	if err != nil {
		t.Fatalf("no puedo leer la hoja de la casa: %v", err)
	}
	if !strings.Contains(string(hoja), ".marco-tabla { overflow: auto;") {
		t.Fatal("la hoja ya no da overflow a .marco-tabla: esta puerta no vigila nada, o " +
			"la clase con scroll es otra y hay que anadirla")
	}

	vistos := 0
	for _, dir := range []string{"plantillas", "../calendario/plantillas", "../escalado/plantillas"} {
		err := filepath.WalkDir(dir, func(ruta string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".html") {
				return nil
			}
			b, err := os.ReadFile(ruta) // #nosec G304 -- recorre el arbol del repositorio
			if err != nil {
				return err
			}
			for _, m := range reMarcoTabla.FindAllString(string(b), -1) {
				vistos++
				if !strings.Contains(m, `tabindex="0"`) {
					t.Errorf(`%s: una zona desplazable sin tabindex.

  %s

  La hoja le da overflow:auto y max-height, asi que se desplaza. Quien navega con
  el tabulador no llega a su contenido, y aqui el contenido es la derivacion
  entera de una cifra. Arreglo: tabindex="0" role="region" aria-label=...`, ruta, m)
				}
				if !strings.Contains(m, "aria-label=") && !strings.Contains(m, "aria-labelledby=") {
					t.Errorf("%s: una zona desplazable enfocable y sin nombre. Un lector de "+
						"pantalla anuncia 'grupo' y ya.\n  %s", ruta, m)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("recorriendo %s: %v", dir, err)
		}
	}
	// SUELO: sin el, borrar la tabla del acta dejaria esta puerta verde
	// recorriendo cero elementos, que es el verde por vacio.
	if vistos == 0 {
		t.Fatal("no se ha encontrado ni un .marco-tabla en las tres superficies de esta " +
			"columna: el detector no esta mirando el HTML que cree mirar")
	}
}
