package pantallas

import (
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
)

// LA FUENTE VA AUTOALOJADA, Y LA HOJA NO PIDE NADA DE FUERA.
//
// # Por que hace falta otra puerta si ya esta TestHtmxVaVendorizadoYNoPorCDN
//
// Aquella mira el HTML: los src y los href de la pagina. Y una hoja de estilo
// pide recursos por su cuenta, con url(), sin que en el HTML aparezca ni una
// direccion. O sea que un `@import url(https://fonts.googleapis.com/...)` o un
// `src: url(https://fonts.gstatic.com/...)` metido en plazum.css pasa por
// delante de aquel test sin rozarlo, y con el pasa lo mismo que con htmx por
// CDN: un tercero se entera de que alguien esta mirando su cumplimiento, la
// pagina no se pinta en una red sin salida a internet, y la CSP de
// superficies/serve (que trae font-src 'self') la bloquea en silencio.
//
// Se mira lo que la hoja PIDE, no lo que la hoja dice: el fichero se lee de
// disco y se le sacan todos los url(), que es la unica forma de que un recurso
// nuevo entre en la comprobacion solo.
func TestLaHojaDeEstiloNoPideNadaDeFuera(t *testing.T) {
	hoja := leerHoja(t)

	// url(...) con o sin comillas.
	re := regexp.MustCompile(`url\(\s*["']?([^"')]+)["']?\s*\)`)
	referencias := re.FindAllStringSubmatch(hoja, -1)
	if len(referencias) == 0 {
		t.Fatal("la hoja no referencia ningun recurso: o la fuente autoalojada ha " +
			"desaparecido, o este detector no esta leyendo el fichero. Las dos cosas dejan " +
			"esta puerta mirando el vacio")
	}
	s, _ := superficie(t, corpusDemo())
	for _, m := range referencias {
		ref := strings.TrimSpace(m[1])
		if strings.HasPrefix(ref, "data:") {
			continue // va dentro de la propia hoja, no sale a ningun sitio
		}
		for _, fuera := range []string{"//", "http:", "https:"} {
			if strings.HasPrefix(ref, fuera) {
				t.Errorf("plazum.css pide %q, que sale de este sitio. La fuente y todo lo "+
					"demas van autoalojados en estatico/: por CDN, un tercero se entera de "+
					"quien esta mirando su cumplimiento y la pagina no se pinta en una red "+
					"sin salida", ref)
			}
		}
		if strings.HasPrefix(ref, "/") {
			t.Errorf("plazum.css pide %q con direccion absoluta. Tiene que ser relativa: la "+
				"hoja se sirve desde /estatico/, asi que una relativa resuelve sola tambien "+
				"cuando la superficie se monta bajo un prefijo (/ui), y una absoluta deja "+
				"el recurso sin cargar en ese montaje", ref)
			continue
		}
		// Y lo que pide tiene que EXISTIR embebido y saberse servir.
		if _, hay := ficherosEstaticos[ref]; !hay {
			t.Errorf("plazum.css pide %q y no esta entre los estaticos embebidos que la "+
				"superficie sabe servir. Arreglo: anadir el fichero a estatico/ y su "+
				"extension a tiposPorExtension", ref)
			continue
		}
		w, _ := pedir(t, s, "/estatico/"+ref)
		if w.Code != http.StatusOK {
			t.Errorf("GET /estatico/%s da %d: la hoja pide un recurso que el servidor no "+
				"entrega", ref, w.Code)
		}
	}
}

// La fuente viaja con su licencia al lado, igual que htmx. Distribuir tipografia
// ajena sin su licencia no es un descuido de forma: la SIL Open Font License
// exige que el aviso viaje con el fichero.
func TestLaFuenteViajaConSuLicencia(t *testing.T) {
	const fichero = "inter-var-latin.woff2"
	res, hay := ficherosEstaticos[fichero]
	if !hay {
		t.Fatalf("%q no esta embebido: la interfaz se quedaria con la fuente del sistema y "+
			"esta puerta no tendria nada que vigilar", fichero)
	}
	if res.tipo != "font/woff2" {
		t.Errorf("la fuente se sirve como %q y tiene que ser font/woff2: con nosniff puesto, "+
			"un tipo equivocado deja la pagina sin fuente sin decir por que", res.tipo)
	}
	// Es un woff2 de verdad y no un fichero renombrado. La firma son cuatro
	// bytes, wOF2 (RFC 8081).
	if len(res.cuerpo) < 4 || string(res.cuerpo[:4]) != "wOF2" {
		t.Errorf("%q no empieza por la firma wOF2: no es un woff2", fichero)
	}
	lic, hay := ficherosEstaticos["inter-LICENSE.txt"]
	if !hay {
		t.Fatal("la fuente viaja sin su licencia al lado")
	}
	if !strings.Contains(string(lic.cuerpo), "SIL OPEN FONT LICENSE") {
		t.Error("el fichero de licencia de la fuente no contiene la SIL Open Font License, " +
			"que es con la que se distribuye")
	}
}

// La pagina declara su icono, y el icono es nuestro.
//
// No es cosmetica: un producto sin icono aparece en la barra de pestanas como un
// folio en blanco, y quien tiene ocho pestanas abiertas no vuelve a la que no
// distingue. Se comprueba en TODAS las respuestas que sirve esta superficie,
// incluida la de error, porque es justo la pestana que uno quiere reconocer para
// cerrarla.
func TestTodaPaginaDeclaraElIconoYEsNuestro(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	re := regexp.MustCompile(`<link[^>]+rel="icon"[^>]*href="([^"]+)"`)
	for _, ruta := range []string{"/alcance", "/hoy", "/controles", "/certificados",
		"/personas", "/estado", "/no-existe"} {
		_, cuerpo := pedir(t, s, ruta)
		m := re.FindStringSubmatch(cuerpo)
		if m == nil {
			t.Errorf("%s se sirve sin <link rel=\"icon\">", ruta)
			continue
		}
		nombre, ok := strings.CutPrefix(m[1], "/estatico/")
		if !ok {
			t.Errorf("%s declara el icono en %q, que no sale de nuestros estaticos", ruta, m[1])
			continue
		}
		if _, hay := ficherosEstaticos[nombre]; !hay {
			t.Errorf("%s declara el icono %q y no esta embebido", ruta, nombre)
		}
	}
}

func leerHoja(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("estatico/plazum.css")
	if err != nil {
		t.Fatalf("no puedo leer la hoja de estilo: %v", err)
	}
	return string(b)
}
