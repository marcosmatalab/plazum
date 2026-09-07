package catalogo

import (
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/estado"
)

// EL CATALOGO DICE DEL MOTIVO EXACTAMENTE LO QUE DICE EL MOTOR.
//
// # El defecto que esta puerta cierra, y por que ninguna existente lo veia
//
// `estado.Calcular` escribia el porque de cada estado en castellano y esa cadena
// viajaba CRUDA hasta la celda de la tabla de controles. La pagina inglesa
// imprimia espanol, y las puertas de i18n de este repositorio no podian verlo
// **porque todas vigilan el catalogo y aquello no era una clave**: la forma de
// publicar sin traducir aqui es no pasar por el catalogo.
//
// # Por que este test es hermano del del acta y no de los demas
//
// Las otras cadenas de la interfaz son de la interfaz: si alguien las reescribe,
// cambia un rotulo. Estas no. Son la DERIVACION del motor, que ademas sale por
// dos medios: el expediente la lleva en espanol resuelto (sin navegador delante
// que la traduzca) y la pantalla la pinta por clave. Si las dos se separan, el
// documento que verifica un auditor y la pagina que mira un CISO explican de
// forma distinta el mismo veredicto.
//
// Y HAY UNA EXIGENCIA QUE EL ACTA NO TIENE: estas cadenas llevan HUECOS. Dos
// redacciones que digan lo mismo con los `%s` en distinto orden colocan un
// recurso donde va una fecha. Por eso se cuenta los verbos, en los dos idiomas.
//
// SE COMPARA SIN TILDES, por lo mismo que en el acta: las constantes de Go de
// este repositorio van sin tildes por convencion y el catalogo lleva espanol
// escrito para leerse. Lo que tiene que coincidir son las PALABRAS.
func TestElCatalogoDiceDelEstadoLoMismoQueElNucleo(t *testing.T) {
	c := nuevoParaTest(t)
	cadenas := estado.CadenasDelEstado()
	// El suelo es el numero real de hoy. Puesto en 1 solo protegeria del cero,
	// y lo que hace falta es enterarse cuando el conjunto MENGUA: una rama de
	// Calcular que pierda su clave vuelve a escribir frases.
	if len(cadenas) != 11 {
		t.Fatalf("nucleo/estado declara %d motivos y son once: si ha entrado una rama nueva "+
			"en Calcular, su cadena tiene que pasar por aqui; y si ha salido una, hay que "+
			"borrarla del catalogo", len(cadenas))
	}
	for _, f := range cadenas {
		es := c.Traducir("es", f.Clave)
		if es == f.Clave {
			t.Errorf("el catalogo no tiene %q, asi que la pantalla la pintaria en crudo",
				f.Clave)
			continue
		}
		if sinTildes(es) != sinTildes(f.Texto) {
			t.Errorf("la clave %q dice cosas distintas en el expediente y en la pantalla.\n"+
				"  nucleo:   %q\n  catalogo: %q\n"+
				"  Arreglo: copiar el texto de nucleo/estado a es.json. Y si lo que cambio "+
				"fue la constante de nucleo, MIRAR TAMBIEN EL INGLES: este test no lo puede "+
				"comprobar y su commit tiene que decir si se reviso y por que",
				f.Clave, f.Texto, es)
		}
		// EL INGLES EXISTE Y NO ES EL ESPANOL. No se puede comprobar que diga lo
		// mismo (para eso haria falta otra constante, y entonces habria dos
		// fuentes), pero si que alguien lo escribio: una clave que cae al idioma
		// por defecto deja la pagina inglesa con el motivo en espanol, que es
		// exactamente el defecto que esta puerta existe para impedir.
		en := c.Traducir("en", f.Clave)
		if en == f.Clave {
			t.Errorf("la clave %q no tiene ingles, asi que la pagina inglesa imprimiria "+
				"espanol: es el defecto entero, otra vez", f.Clave)
			continue
		}
		if sinTildes(en) == sinTildes(f.Texto) && !esCortaYComun(f.Texto) {
			t.Errorf("la clave %q tiene el mismo texto en ingles que en espanol (%q), asi que "+
				"o no se tradujo o se copio", f.Clave, en)
		}
		// LOS HUECOS, QUE ES LA MITAD QUE EL ACTA NO NECESITA COMPROBAR.
		//
		// `Traducir` no falla cuando los verbos no casan: devuelve la plantilla
		// SIN FORMATEAR, para no escupir el %!s(MISSING) de fmt en una pagina.
		// O sea que una traduccion a la que le falte un %s no rompe nada: pinta
		// «%s recurso(s) no pasan» con el %s literal delante de un CISO. Es un
		// verde silencioso, y por eso se cuenta aqui.
		//
		// SE REUSA `verbos` DEL PROPIO PAQUETE en vez de contar los % aqui, y no
		// es comodidad: aquella sabe de banderas, de anchos y de indices
		// explicitos (%[1]s), y ya trae su test. Una segunda implementacion de
		// la misma cuenta es como se consigue que dos esten de acuerdo y la que
		// mande sea la tercera.
		//
		// Y se comparan EN ORDEN y no en numero: dos plantillas con los mismos
		// huecos cambiados de sitio colocan un recurso donde va una fecha, y un
		// recuento no lo ve.
		nucleo := strings.Join(verbos(f.Texto), ",")
		ingles := strings.Join(verbos(en), ",")
		if nucleo != ingles {
			t.Errorf("la clave %q tiene los huecos [%s] en el nucleo y [%s] en el ingles.\n"+
				"  nucleo: %q\n  ingles: %q\n"+
				"  Un hueco de menos pinta el %%s literal en la pagina, uno de mas deja un "+
				"dato fuera, y cambiados de orden ponen un recurso donde va una fecha. "+
				"Traducir() no lo puede cazar: cuando el formateo no casa devuelve la "+
				"plantilla sin formatear, que es un verde silencioso",
				f.Clave, nucleo, ingles, f.Texto, en)
		}
	}

	// LA OTRA DIRECCION: el catalogo no lleva motivos que el motor no escriba.
	//
	// Una huerfana aqui no es peso muerto: es una explicacion que alguien
	// redacto para un veredicto que ya no se da, y que sigue leyendose como si
	// el producto la usara.
	declaradas := map[string]bool{}
	for _, f := range cadenas {
		declaradas[f.Clave] = true
	}
	for _, k := range c.Claves() {
		if strings.HasPrefix(k, "evidencia.motivo.") && !declaradas[k] {
			t.Errorf("el catalogo lleva %q y nucleo/estado no la escribe en ninguna rama de "+
				"Calcular", k)
		}
	}
}

// TestElContrasteDeHuecosSabePonerseRojo es el control negativo de la tercera
// afirmacion del test de arriba.
//
// Hace falta porque esa afirmacion nacio VERDE sobre las once claves reales, y
// una comparacion que nunca se ha visto fallar no distingue «las traducciones
// estan bien» de «estoy comparando dos cosas que siempre son iguales». Aqui se
// le ponen delante las tres formas de romperla, que son tres y no una: un hueco
// de menos, uno de mas, y los mismos cambiados de sitio.
//
// LA DE ORDEN ES LA QUE JUSTIFICA COMPARAR SECUENCIAS y no contar: un recuento
// la deja pasar, y su efecto es pintar el nombre de un recurso donde el lector
// espera una fecha.
func TestElContrasteDeHuecosSabePonerseRojo(t *testing.T) {
	const bueno = "%s recurso(s) no pasan la comprobacion, plazo de remediacion hasta el %s"
	casos := []struct {
		que   string
		malo  string
		mismo bool
	}{
		{"la traduccion correcta", "%s resource(s) do not pass, window until %s", true},
		{"un hueco de menos", "%s resources do not pass, window until the given date", false},
		{"un hueco de mas", "%s resource(s) (%s) do not pass, window until %s", false},
		{"del tipo cambiado", "%d resource(s) do not pass, window until %s", false},
	}
	for _, c := range casos {
		igual := strings.Join(verbos(bueno), ",") == strings.Join(verbos(c.malo), ",")
		if igual != c.mismo {
			t.Errorf("%s: el contraste dice que los huecos %s casan y %s.\n  base: %q\n"+
				"  otra: %q", c.que, siNo(igual), siNo(c.mismo)+" era lo esperado",
				bueno, c.malo)
		}
	}
}

func siNo(b bool) string {
	if b {
		return "SI"
	}
	return "NO"
}
