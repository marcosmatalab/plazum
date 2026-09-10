package main

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/superficies/camino"
	"github.com/marcosmatalab/plazum/superficies/documentos"
	"github.com/marcosmatalab/plazum/superficies/pantallas"
)

// UNA SUPERFICIE MONTADA FUERA DEL CAMINO SE ALCANZA DESDE ALGUNA PANTALLA, o
// dice por que no.
//
// # El otro lado de la frontera que el censo dejaba abierto
//
// `MontadaFueraDelCamino` ya exigia MOTIVO, y su propio godoc dice, con estas
// palabras, que «una pantalla a la que solo se llega tecleando la direccion es
// una pantalla que solo encuentra quien ya sabia que existia». Eso era una
// afirmacion sin puerta: nada comprobaba que a esas superficies se llegara.
//
// Y el motivo de `documentos` lo decia por escrito («se llega a ella desde el
// enlace de /alcance»). Escribir el motivo no es lo mismo que ponerle puerta:
// una frase asi es verdad el dia que se escribe y deja de serlo sin que nada se
// ponga rojo, que es la familia entera de la afirmacion acompanada.
//
// # Que se afirma, y donde se busca
//
// Por cada superficie declarada MontadaFueraDelCamino, o su ruta sale como
// `href` DENTRO del `<main>` de alguna pantalla del servidor montado, o esta en
// `sinEnlaceEntrante` con su motivo.
//
// DENTRO DE `<main>` Y NO EN LA PAGINA ENTERA, por lo mismo que la puerta D11-b:
// el armazon compartido pinta la barra lateral con los seis pasos del camino en
// TODAS las paginas, asi que buscar en el cuerpo entero encontraria el marco de
// la ventana y no la habitacion.
//
// # Lo que esta puerta NO mira
//
// Que el enlace se VEA. Un `href` correcto dentro de un bloque plegado, o al
// final de una pagina larga, cumple la letra. Lo que separa las dos cosas es
// mirar la pantalla, y eso no lo hace ninguna puerta de este arbol.

// sinEnlaceEntrante son las superficies fuera del camino a las que NO se llega
// por un enlace, cada una con su motivo.
var sinEnlaceEntrante = map[string]string{
	"serve": "es QUIEN monta: no es una pantalla a la que se llegue, es la que sirve la " +
		"puerta de entrada y cuelga a las demas. Enlazarla desde dentro seria enlazar a " +
		"la raiz desde la raiz.",
	"camino": "es el camino mismo, y SI se enlaza, pero desde la tira de vuelta que pinta " +
		"el armazon compartido, o sea FUERA de <main> en todas las pantallas. Esta puerta " +
		"solo mira dentro de <main> a proposito, asi que aqui no se puede ver. Lo que " +
		"comprueba que ese enlace existe es TestElCaminoGuiadoNoTieneCallejones.",
}

// reHref saca los destinos de los enlaces.
var reHref = regexp.MustCompile(`href="([^"]*)"`)

func TestUnaSuperficieFueraDelCaminoSeAlcanzaDesdeAlgunaPantalla(t *testing.T) {
	// EL SERVIDOR REAL, con la puerta a los documentos cableada igual que la
	// cablea `plazum serve`: ruta y cardinal, que son las dos mitades.
	h := servidorConPuertaALosDocumentos(t)

	// LAS PANTALLAS DONDE SE BUSCA. Son las que una persona recorre: los seis
	// pasos del camino mas la portada del camino.
	rutas := []string{camino.BasePorDefecto + "/"}
	for _, p := range camino.Canonico() {
		rutas = append(rutas, p.Ruta)
	}
	if len(rutas) < 7 {
		t.Fatalf("se van a recorrer %d pantallas y son siete (los seis pasos y la portada "+
			"del camino): esta puerta estaria buscando en el vacio", len(rutas))
	}

	// Todos los href que salen DENTRO de <main>, en todas ellas.
	destinos := map[string]string{} // href -> primera pantalla donde sale
	for _, ruta := range rutas {
		codigo, cuerpo := pedirCamino(t, h, ruta)
		if codigo != http.StatusOK {
			t.Errorf("%s contesta %d, asi que lo que se recorre es una pagina de error",
				ruta, codigo)
			continue
		}
		dentro := principalDe(t, ruta, cuerpo)
		for _, m := range reHref.FindAllStringSubmatch(dentro, -1) {
			if _, ya := destinos[m[1]]; !ya {
				destinos[m[1]] = ruta
			}
		}
	}
	if len(destinos) < 5 {
		t.Fatalf("se han encontrado %d enlaces dentro de <main> en siete pantallas, y hay "+
			"muchos mas: la extraccion de <main> ha dejado de casar y esta puerta estaria "+
			"recorriendo la nada", len(destinos))
	}

	// LAS RUTAS DE CADA SUPERFICIE, sacadas de donde las declara el producto y
	// no escritas aqui: una ruta escrita en un test es una ruta que sigue dando
	// verde cuando el producto la mueve.
	base := map[string]string{
		"documentos": documentos.BasePorDefecto + "/",
		"camino":     camino.BasePorDefecto + "/",
		"serve":      "/",
	}

	comprobadas := 0
	for _, nombre := range clavesDelCensoDeSuperficies() {
		if SuperficiesHTTP[nombre].Estado != MontadaFueraDelCamino {
			continue
		}
		comprobadas++
		motivo, eximida := sinEnlaceEntrante[nombre]
		ruta, hayRuta := base[nombre]
		if !hayRuta {
			t.Errorf("superficies/%s se declara fuera del camino y esta puerta no sabe cual "+
				"es su ruta. Arreglo: anadirla al mapa `base`, sacandola de la constante "+
				"que declare la superficie", nombre)
			continue
		}
		_, alcanzable := destinos[ruta]
		switch {
		case !alcanzable && !eximida:
			t.Errorf("superficies/%s se monta fuera del camino y NINGUNA pantalla enlaza a "+
				"%q dentro de su <main>.\n"+
				"  Una pantalla a la que solo se llega tecleando la direccion es una "+
				"pantalla que solo encuentra quien ya sabia que existia, que es lo que el "+
				"godoc de MontadaFueraDelCamino avisa y lo que hasta hoy no comprobaba "+
				"nadie.\n"+
				"  Arreglo: enlazarla desde la pantalla donde tiene sentido, o declararla "+
				"en sinEnlaceEntrante con el motivo.", nombre, ruta)
		case alcanzable && eximida:
			// LA DIRECCION CONTRARIA. Una excusa para una superficie que ya SI
			// se enlaza es una excusa huerfana, y una lista de excusas
			// huerfanas es como un censo deja de vigilar sin que nadie se
			// entere.
			t.Errorf("sinEnlaceEntrante exime a superficies/%s y %q SI sale enlazada en %q. "+
				"Arreglo: quitar la entrada.\n  decia: %s",
				nombre, ruta, destinos[ruta], motivo)
		case alcanzable:
			t.Logf("superficies/%s se alcanza desde %s", nombre, destinos[ruta])
		}
	}
	// Y toda excusa apunta a una superficie que sigue estando fuera del camino.
	for nombre := range sinEnlaceEntrante {
		if SuperficiesHTTP[nombre].Estado != MontadaFueraDelCamino {
			t.Errorf("sinEnlaceEntrante exime a superficies/%q, que ya no se declara fuera "+
				"del camino. Arreglo: quitar la entrada", nombre)
		}
	}
	if comprobadas < 3 {
		t.Fatalf("solo se han comprobado %d superficies fuera del camino y hoy son tres "+
			"(serve, camino y documentos): esta puerta esta midiendo el vacio", comprobadas)
	}
}

// EL CONTROL POSITIVO DEL ENLACE, que es la mitad que la puerta de arriba no
// puede darse a si misma.
//
// Sin esto, un servidor montado SIN la puerta a los documentos dejaria la de
// arriba en rojo y la reaccion barata seria eximir `documentos` en
// sinEnlaceEntrante. Aqui se demuestra lo contrario: que el enlace aparece
// cuando se cablea y NO aparece cuando no se cablea, que son las dos mitades del
// valor cero de Opciones.DocumentosRuta.
func TestLaPuertaALosDocumentosSaleSoloSiSeCablea(t *testing.T) {
	// SIN CABLEAR: ni rastro.
	_, cuerpo := pedirCamino(t, servidorDelCamino(t, nil,
		func(*http.Request) string { return "ciso" }).Handler(), "/alcance")
	if strings.Contains(principalDe(t, "/alcance", cuerpo), documentos.BasePorDefecto) {
		t.Error("sin cablear la ruta, la pantalla del alcance ya enlaza a los documentos.\n" +
			"  El valor cero de Opciones.DocumentosRuta tiene que ser NO PINTAR NADA: un " +
			"enlace a una pantalla que quien monta no ha colgado es un 404 con buena letra")
	}
	// CABLEADA: sale, y con su cardinal al lado.
	_, cuerpo = pedirCamino(t, servidorConPuertaALosDocumentos(t), "/alcance")
	dentro := principalDe(t, "/alcance", cuerpo)
	if !strings.Contains(dentro, `href="`+documentos.BasePorDefecto+`/"`) {
		t.Fatalf("con la ruta cableada, la pantalla del alcance NO enlaza a los "+
			"documentos:\n%s", recortarPrincipal(dentro))
	}
	// EL CARDINAL VA CON EL ENLACE (D-13), y es de la INSTALACION: cuantas
	// obligaciones se pueden buscar dentro de un documento. Un enlace sin
	// numero no dice si detras hay algo.
	//
	// EL NUMERO SE DERIVA Y NO SE ESCRIBE: lo dice quien compone las consultas,
	// que es el mismo objeto que resuelve las busquedas. Escrito aqui seria una
	// segunda copia que sigue dando verde el dia que el corpus cambie.
	esperado := strconv.Itoa(nuevosIndicesPorCuenta(corpusInstalado(t)).Consultas())
	if !strings.Contains(dentro, esperado) {
		t.Errorf("el enlace a los documentos sale sin su cardinal (%s).\n"+
			"  Un enlace pelado no dice si detras hay una busqueda contra 3 obligaciones o "+
			"contra 500, y esa es justo la informacion que decide si merece la pena "+
			"entrar.\n%s", esperado, recortarPrincipal(dentro))
	}
}

// servidorConPuertaALosDocumentos monta el servidor con la pieza 3 cableada
// igual que la cablea `plazum serve`: el mismo objeto publica el cardinal y
// resuelve las busquedas.
func servidorConPuertaALosDocumentos(t *testing.T) http.Handler {
	t.Helper()
	ps := corpusInstalado(t)
	indice := nuevosIndicesPorCuenta(ps)
	if indice.Consultas() == 0 {
		t.Fatal("el corpus instalado no compone ni una consulta: el enlace saldria sin " +
			"cardinal y esta puerta mediria el vacio")
	}
	return servidorDelCamino(t, ps, func(*http.Request) string { return "ciso" },
		func(o *pantallas.Opciones) {
			o.DocumentosRuta = documentos.BasePorDefecto + "/"
			o.DocumentosConsultas = indice.Consultas()
		}).Handler()
}

func recortarPrincipal(s string) string {
	if len(s) > 1500 {
		return s[:1500] + "\n[...recortado]"
	}
	return s
}

// principalDe saca el <main> y para si no casa, porque una extraccion que
// devuelve la pagina entera convierte esta puerta en la que se satisface con el
// marco de la ventana.
func principalDe(t *testing.T, ruta, cuerpo string) string {
	t.Helper()
	i := strings.Index(cuerpo, "<main")
	j := strings.Index(cuerpo, "</main>")
	if i < 0 || j < 0 || j < i {
		t.Fatalf("no he podido recortar el <main> de %s. Sin recorte, esta puerta buscaria "+
			"en la barra lateral, que enlaza a los seis pasos en todas las paginas", ruta)
	}
	return cuerpo[i:j]
}
