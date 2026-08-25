package dutiq

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dutiq/nucleo/corpus"
	"dutiq/superficies/pantallas"
	"dutiq/superficies/serve"
)

// El cableado de las dos superficies, probado donde se encuentran.
//
// Se construyeron en frentes distintos y contra los mismos puertos congelados,
// a proposito: serve no conoce las pantallas y las pantallas no conocen serve.
// Eso esta bien mientras alguien compruebe que al juntarlas siguen siendo lo
// que cada una prometia por separado, y ese alguien es este fichero. Vive en la
// raiz porque es el unico sitio que puede importar las dos sin acoplarlas.
//
// Lo que se comprueba no es que "arranque": es que las dos puertas de seguridad
// que cada frente construyo sigan cubriendo el conjunto, que es justo lo que
// deja de ser cierto cuando dos piezas correctas se montan mal.

// catalogoDePrueba cubre lo justo: devuelve la clave. La interfaz sale sin
// texto, y eso aqui da igual, porque lo que se mira son las rutas.
type catalogoDePrueba struct{}

func (catalogoDePrueba) Traducir(_, clave string, _ ...any) string { return clave }
func (catalogoDePrueba) Idiomas() []string                         { return []string{"es"} }
func (catalogoDePrueba) Faltantes(string) []string                 { return nil }

func superficieMontada(t *testing.T) *serve.Servidor {
	t.Helper()
	ps, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatalf("cargar el corpus: %v", err)
	}
	app, err := pantallas.Nuevo(pantallas.Opciones{Paquetes: ps, Catalogo: catalogoDePrueba{}})
	if err != nil {
		t.Fatalf("construir las pantallas: %v", err)
	}
	s, err := serve.Nuevo(serve.Config{App: app})
	if err != nil {
		t.Fatalf("construir el servidor: %v", err)
	}
	return s
}

// Las rutas de las pantallas TIENEN que entrar en la enumeracion del servidor.
//
// El frente de serve dejo escrito, honestamente, que las pantallas quedaban
// cubiertas por la comprobacion por metodo pero NO entraban en la puerta que
// enumera. La diferencia importa: la comprobacion por metodo protege hoy, y la
// enumeracion es la que caza una ruta mutante NUEVA el mismo dia que alguien la
// escribe. Se cerro con EnumeradorDePatrones, y esto lo comprueba de verdad,
// con las dos superficies reales y no con un doble.
func TestLasRutasDeLasPantallasEntranEnLaEnumeracionDelServidor(t *testing.T) {
	s := superficieMontada(t)

	vistas := map[string]bool{}
	for _, r := range s.Rutas() {
		vistas[r.Patron] = true
	}
	// Las seis pantallas, por su ruta.
	for _, ruta := range []string{"/alcance", "/hoy", "/controles", "/certificados",
		"/personas", "/estado"} {
		if !vistas[ruta] {
			t.Errorf("la ruta %q de las pantallas no sale en Servidor.Rutas(). Queda cubierta "+
				"por la comprobacion de CSRF por metodo, pero NO entra en la puerta que "+
				"enumera, que es la que caza una ruta mutante nueva el dia que se escribe. "+
				"Enumeradas: %v", ruta, vistas)
		}
	}
	// Y las propias del servidor siguen ahi: montar la aplicacion no puede
	// haberse llevado por delante las de arranque.
	if !vistas["/salud"] {
		t.Errorf("las rutas propias del servidor han desaparecido de la enumeracion: %v", vistas)
	}
}

// Ninguna ruta del conjunto muta. Es la misma propiedad que cada frente vigila
// en su lado, comprobada sobre la suma, que es lo que se despliega.
func TestNingunaRutaDelConjuntoMutaSinPasarPorCSRF(t *testing.T) {
	s := superficieMontada(t)
	h := s.Handler()

	probadas := 0
	for _, r := range s.Rutas() {
		ruta := strings.ReplaceAll(r.Patron, "{$}", "")
		ruta = strings.ReplaceAll(ruta, "{fichero}", "dutiq.css")
		if strings.Contains(ruta, "{") {
			continue // patron con comodin que no sabemos rellenar
		}
		for _, metodo := range []string{http.MethodPost, http.MethodPut, http.MethodDelete,
			http.MethodPatch} {
			req := httptest.NewRequest(metodo, ruta, strings.NewReader(""))
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			probadas++
			// Se acepta cualquier rechazo: 403 del CSRF, 405 del metodo, 404 si
			// la ruta solo existe para GET, 3xx de la redireccion a instalacion.
			// Lo que NO se acepta es un 2xx: eso seria una mutacion atendida.
			if w.Code >= 200 && w.Code < 300 {
				t.Errorf("%s %s devolvio %d. Una peticion mutante atendida sin CSRF es la que "+
					"el atacante va a buscar", metodo, ruta, w.Code)
			}
		}
	}
	if probadas < 24 {
		t.Fatalf("solo se han probado %d combinaciones. Con tan pocas el verde no demuestra "+
			"nada: revisa que Rutas() este devolviendo lo que hay", probadas)
	}
}

// Las cabeceras de seguridad del servidor llegan tambien a las paginas de la
// aplicacion, no solo a las suyas. Es el fallo tipico de montar una aplicacion
// por debajo: el middleware envuelve el enrutador propio y la aplicacion se
// cuelga por fuera.
func TestLasCabecerasDeSeguridadLleganALasPantallas(t *testing.T) {
	s := superficieMontada(t)
	req := httptest.NewRequest(http.MethodGet, "/controles", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	obligatorias := []string{
		"Content-Security-Policy",
		"X-Frame-Options",
		"X-Content-Type-Options",
		"Referrer-Policy",
	}
	for _, c := range obligatorias {
		if w.Header().Get(c) == "" {
			t.Errorf("la pagina de pantallas sale sin %s. El middleware envuelve el enrutador "+
				"propio y la aplicacion se ha quedado por fuera de la cadena", c)
		}
	}
}
