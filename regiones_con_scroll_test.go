package plazum

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TODA REGION CON SCROLL SE PUEDE ENFOCAR CON EL TECLADO.
//
// # De donde sale, y es la puerta de accesibilidad cobrandose lo que cerro
//
// El 04-09-2026 se cablearon las cuentas de usuario, y con ellas /acta/ y /uar/
// dejaron de contestar 401. Esas dos pantallas llevaban desde que existen sin
// auditarse, con su motivo escrito («exigen sesion y no hay forma de entrar»),
// y en cuanto entraron en la auditoria salieron DOS VIOLACIONES SERIAS reales,
// en los dos idiomas: `scrollable-region-focusable` sobre un `<pre>`.
//
// El CSS de la casa da `overflow-x: auto` a todo `<pre>`, asi que cualquiera
// que se pase de ancho se convierte en una region con scroll. Con raton se
// arrastra; con teclado no se llega, porque un `<pre>` no es enfocable. Quien
// navega sin raton no puede leer el final de la orden que la pantalla le pide
// teclear, que es justo el contenido que hay que copiar entero.
//
// # Por que una puerta y no cuatro atributos
//
// Porque el fallo no estaba en esas dos pantallas: estaba en que NADIE LOS
// CONTABA. Axe solo marca los `<pre>` que de verdad se desbordan hoy, asi que
// alargar una orden en un template convierte un `<pre>` inocente en una
// violacion sin tocar nada mas. Es la familia de este tramo: la pieza pasa su
// puerta, la junta no la mira nadie.
//
// Se enumera desde el ARBOL de plantillas, no desde una lista escrita al lado.
//
// # El alcance, dicho
//
// Esto vigila `<pre>`, que es el unico elemento al que la hoja de la casa da
// scroll hoy. El dia que se le de a un `<div>` o a una tabla, esta puerta NO lo
// vera, y por eso el comentario lo dice: se amplia la lista, no se confia en
// que no pase.
var (
	rePre        = regexp.MustCompile(`(?i)<pre\b[^>]*>`)
	reTieneFoco  = regexp.MustCompile(`(?i)\btabindex\s*=\s*"0"`)
	// EL SUELO BAJA DE 4 A 3, y se dice por que en vez de tocarlo callando: la
	// pantalla de revision de accesos tenia DOS bloques preformateados con las
	// ordenes de terminal de su estado vacio, y ese estado vacio ya no manda al
	// terminal, asi que los dos se fueron con las ordenes. Bajar un suelo es
	// sospechoso por definicion (es como se afloja una puerta para que pase
	// algo), y aqui es legitimo justo porque lo que desaparecio es lo que este
	// numero contaba. Medido: `grep -c` sobre las plantillas da 3.
	minimoDePres = 3
)

func TestTodaRegionConScrollSeAlcanzaConElTeclado(t *testing.T) {
	vistos := 0
	err := filepath.WalkDir("superficies", func(ruta string, d fs.DirEntry, err error) error {
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
		for _, etiqueta := range rePre.FindAllString(string(b), -1) {
			vistos++
			if reTieneFoco.MatchString(etiqueta) {
				continue
			}
			t.Errorf("%s: %s no se puede enfocar con el teclado.\n"+
				"  La hoja de la casa da `overflow-x: auto` a todo <pre>, asi que en cuanto "+
				"su contenido se pasa de ancho es una REGION CON SCROLL, y a una region con "+
				"scroll que no es enfocable no se llega sin raton: quien navegue con teclado "+
				"no puede leer el final de la orden que esta pantalla le pide teclear.\n"+
				"  Arreglo: `<pre tabindex=\"0\">`. Es lo que axe pide en "+
				"scrollable-region-focusable, y salio en /acta/ y /uar/ el dia que dejaron "+
				"de contestar 401 y pudieron auditarse por primera vez.", ruta, etiqueta)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("recorriendo las plantillas: %v", err)
	}
	// EL SUELO. Si un dia el recorrido deja de encontrar plantillas (se mueven,
	// se empaquetan, cambia la extension), este test seguiria verde sin mirar
	// nada, que es el verde vacio de siempre.
	if vistos < minimoDePres {
		t.Fatalf("solo se han encontrado %d elementos <pre> en las plantillas y hoy son al "+
			"menos %d: este recorrido esta midiendo el vacio", vistos, minimoDePres)
	}
	t.Logf("MEDIDO: %d regiones con scroll en las plantillas, todas alcanzables con el teclado",
		vistos)
}
