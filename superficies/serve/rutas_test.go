package serve

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// La puerta central de este frente: NINGUNA ruta mutante se queda sin CSRF.
//
// Por que asi y no con una lista. Un middleware que se aplica "a casi todo" no
// protege de nada, porque el atacante busca justo la ruta que se quedo fuera. Y
// una lista de rutas protegidas escrita al lado del test se desincroniza el dia
// que alguien anade un handler, que es exactamente el dia en que hace falta.
// Aqui la lista la da el enrutador (EnumeradorDeRutas), y por cada ruta mutante
// se manda una PETICION DE VERDAD por la cadena completa de middleware: no se
// comprueba que exista una comprobacion, se comprueba que rechaza.
//
// El control negativo esta abajo, en dos formas: la cadena desnuda (sin el
// middleware) tiene que salir entera, y una ruta montada por fuera de la cadena
// tiene que salir ella sola.

// rutasMutantesSinCSRF enumera las rutas de enum y devuelve las mutantes que h
// atendio sin exigir token CSRF.
//
// La peticion lleva cookie de sesion VALIDA y no lleva token. Asi lo unico que
// falta es el token, y un 403 solo puede venir de haberlo echado en falta.
func rutasMutantesSinCSRF(t *testing.T, enum EnumeradorDeRutas, h http.Handler, cookie, sesion string) []string {
	t.Helper()
	var sin []string
	for _, ruta := range enum.Rutas() {
		if !ruta.Mutante() {
			continue
		}
		for _, metodo := range metodosAProbar(ruta) {
			req := httptest.NewRequest(metodo, urlDePatron(ruta.Patron), strings.NewReader(""))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.AddCookie(&http.Cookie{Name: cookie, Value: sesion})
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusForbidden {
				sin = append(sin, fmt.Sprintf("%s %s respondio %d y no 403",
					metodo, ruta.Patron, rec.Code))
			}
		}
	}
	return sin
}

// metodosAProbar: la ruta que declara metodo se prueba con el suyo; la que no
// declara ninguno atiende todos, asi que se prueban los cuatro que mutan.
func metodosAProbar(r Ruta) []string {
	if r.Metodo != "" {
		return []string{r.Metodo}
	}
	return []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
}

// urlDePatron convierte el patron de ServeMux en una URL concreta.
func urlDePatron(patron string) string {
	if i := strings.Index(patron, "/"); i > 0 {
		patron = patron[i:] // el patron podia traer host delante
	}
	var partes []string
	for _, seg := range strings.Split(strings.TrimPrefix(patron, "/"), "/") {
		switch {
		case strings.HasSuffix(seg, "...}") && strings.HasPrefix(seg, "{"):
			partes = append(partes, "a/b")
		case strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}"):
			partes = append(partes, "x")
		default:
			partes = append(partes, seg)
		}
	}
	return "/" + strings.Join(partes, "/")
}

func TestNingunaRutaMutanteSeQuedaSinCSRF(t *testing.T) {
	s, sesion := servidorConSesion(t, Config{App: aplicacionDePrueba()})
	sin := rutasMutantesSinCSRF(t, s, s.Handler(), s.nombreCookie(), sesion)
	if len(sin) > 0 {
		t.Fatalf("hay %d rutas mutantes que se atienden sin token CSRF:\n  %s\n\n"+
			"Una ruta mutante sin CSRF es la que el atacante va a buscar. La comprobacion "+
			"se aplica por METODO en ProtectorCSRF.Envolver, asi que si una ruta se ha "+
			"quedado fuera es que esta montada por fuera de la cadena de middleware.",
			len(sin), strings.Join(sin, "\n  "))
	}
	// Y que de verdad se ha enumerado algo: un enumerador vacio daria verde.
	mutantes := 0
	for _, r := range s.Rutas() {
		if r.Mutante() {
			mutantes++
		}
	}
	if mutantes < 3 {
		t.Fatalf("solo se han enumerado %d rutas mutantes. Con tan pocas, el verde de "+
			"arriba no demuestra nada: revisa que Rutas() este devolviendo lo que hay",
			mutantes)
	}
}

// Control negativo 1: la cadena desnuda. Sin el middleware, TODAS las rutas
// mutantes tienen que salir en el informe. Si no salen, el detector no detecta.
func TestElDetectorSaltaConLaCadenaDesnuda(t *testing.T) {
	s, sesion := servidorConSesion(t, Config{App: aplicacionDePrueba()})
	sin := rutasMutantesSinCSRF(t, s, s.enr, s.nombreCookie(), sesion)

	esperadas := 0
	for _, r := range s.Rutas() {
		if r.Mutante() {
			esperadas += len(metodosAProbar(r))
		}
	}
	if len(sin) != esperadas {
		t.Fatalf("sobre el enrutador SIN middleware el detector encontro %d rutas sin "+
			"CSRF y tenia que encontrar las %d mutantes. Un detector que no salta con el "+
			"agujero delante no demuestra nada cuando da verde:\n  %s",
			len(sin), esperadas, strings.Join(sin, "\n  "))
	}
}

// Control negativo 2, el que de verdad puede pasar: alguien monta un handler
// mutante al lado del servidor en vez de dentro. El servidor sigue protegido y
// la ruta nueva no, y el detector tiene que senalar esa y solo esa.
func TestElDetectorSaltaConUnaRutaMontadaPorFueraDeLaCadena(t *testing.T) {
	s, sesion := servidorConSesion(t, Config{App: aplicacionDePrueba()})

	fuera := http.NewServeMux()
	fuera.Handle("/", s.Handler())
	fuera.Handle("POST /webhook", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	enum := enumeradorFijo(append(s.Rutas(), Ruta{Metodo: http.MethodPost, Patron: "/webhook"}))
	sin := rutasMutantesSinCSRF(t, enum, fuera, s.nombreCookie(), sesion)

	if len(sin) != 1 || !strings.Contains(sin[0], "/webhook") {
		t.Fatalf("con una ruta mutante montada por fuera de la cadena, el detector tenia "+
			"que senalar /webhook y solo /webhook. Encontro %d: %v", len(sin), sin)
	}
}

// Y la otra mitad: que el middleware no este rechazandolo TODO. Un protector
// que devuelve 403 siempre pasaria los tres tests de arriba y dejaria el
// producto inservible.
func TestConTokenValidoLaPeticionMutantePasa(t *testing.T) {
	s, sesion := servidorConSesion(t, Config{App: aplicacionDePrueba()})
	tok, err := s.ses.TokenCSRF(context.Background(), sesion)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/app/guardar", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set(CabeceraCSRF, tok)
	req.AddCookie(&http.Cookie{Name: s.nombreCookie(), Value: sesion})
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("con token valido, POST /app/guardar respondio %d y tenia que responder "+
			"200. Un protector que rechaza tambien lo legitimo no protege, estorba.\n%s",
			rec.Code, rec.Body.String())
	}
}

// El token vale tambien en el campo del formulario, que es como lo manda una
// pantalla sin htmx. Si solo valiera la cabecera, la mitad de la interfaz
// quedaria rota y alguien acabaria exceptuando rutas.
func TestElTokenValeTambienEnElCampoDelFormulario(t *testing.T) {
	s, sesion := servidorConSesion(t, Config{App: aplicacionDePrueba()})
	tok, err := s.ses.TokenCSRF(context.Background(), sesion)
	if err != nil {
		t.Fatal(err)
	}
	cuerpo := CampoCSRF + "=" + tok
	req := httptest.NewRequest(http.MethodPost, "/app/guardar", strings.NewReader(cuerpo))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: s.nombreCookie(), Value: sesion})
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("con el token en el campo %q del formulario respondio %d: %s",
			CampoCSRF, rec.Code, rec.Body.String())
	}
}

// El token NO puede colarse por la cadena de consulta: ahi acaba en el Referer,
// en el historial y en el log de cualquier proxy.
func TestElTokenEnLaCadenaDeConsultaNoVale(t *testing.T) {
	s, sesion := servidorConSesion(t, Config{App: aplicacionDePrueba()})
	tok, err := s.ses.TokenCSRF(context.Background(), sesion)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/app/guardar?"+CampoCSRF+"="+tok, nil)
	req.AddCookie(&http.Cookie{Name: s.nombreCookie(), Value: sesion})
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("un token en la cadena de consulta ha valido (%d). Ahi lo lee el proxy, "+
			"el historial y el Referer de la siguiente peticion", rec.Code)
	}
}

// --- utilidades ---

type enumeradorFijo []Ruta

func (e enumeradorFijo) Rutas() []Ruta { return []Ruta(e) }

// aplicacionDePrueba hace de frente de pantallas: un handler con sus propias
// rutas, que ademas sabe enumerarlas.
func aplicacionDePrueba() *Enrutador {
	e := NuevoEnrutador()
	e.Manejar(http.MethodGet, "/app/", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		e.Manejar(m, "/app/guardar", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
	}
	e.Manejar(http.MethodPost, "/app/perimetro/{id}/cerrar", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	return e
}

// servidorConSesion construye un servidor de pruebas con un administrador ya
// creado (para que la instalacion no se meta por medio) y una sesion abierta.
func servidorConSesion(t *testing.T, cfg Config) (*Servidor, string) {
	t.Helper()
	s := servidorDePrueba(t, cfg)
	id, err := s.ses.Abrir(context.Background(), "ciso@ejemplo", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return s, id
}

func servidorDePrueba(t *testing.T, cfg Config) *Servidor {
	t.Helper()
	if cfg.HayAdmin == nil {
		cfg.HayAdmin = func(context.Context) (bool, error) { return true, nil }
	}
	if cfg.CrearAdmin == nil {
		cfg.CrearAdmin = func(context.Context, string, string) error { return nil }
	}
	if cfg.Salida == nil {
		cfg.Salida = &strings.Builder{}
	}
	// Limites holgados: estos tests miran el CSRF, no el rate limit, y un 429
	// por medio seria un falso hallazgo.
	if cfg.LimiteAuth.Maximo == 0 {
		cfg.LimiteAuth = Limite{Maximo: 10000, Ventana: time.Minute}
	}
	if cfg.LimiteGeneral.Maximo == 0 {
		cfg.LimiteGeneral = Limite{Maximo: 100000, Ventana: time.Minute}
	}
	s, err := Nuevo(cfg)
	if err != nil {
		t.Fatalf("construir el servidor de pruebas: %v", err)
	}
	return s
}
