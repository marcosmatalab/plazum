package main

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/catalogo"
	"github.com/marcosmatalab/plazum/superficies/serve"
	"github.com/marcosmatalab/plazum/superficies/uar"
)

// falsaApp es una superficie que sabe decir sus patrones, como pantallas.
type falsaApp struct{ patrones []string }

func (f falsaApp) ServeHTTP(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }
func (f falsaApp) Patrones() []string                               { return f.patrones }

func catDePrueba(t *testing.T) *catalogo.Catalogo {
	t.Helper()
	c, err := catalogo.Nuevo()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func uarDePrueba(t *testing.T) *uar.Superficie {
	t.Helper()
	// Con operador: sin el, la pantalla contesta 401 y no ensena el censo, que
	// es lo correcto (lleva nombres de personas) y otra cosa que la que estos
	// tests miden.
	u, err := construirUAR(opcionesUAR{
		Catalogo: catDePrueba(t),
		Quien:    func(*http.Request) string { return "ciso" },
	})
	if err != nil {
		t.Fatal(err)
	}
	return u
}

// SIN SESION NO SE ENSENA EL CENSO.
//
// Su contenido es dato personal: nombres de personas y sus permisos. Servirla a
// quien no ha entrado la convierte en un directorio de empleados publicado sin
// querer.
//
// YA NO ES LA UNICA CON DATO PERSONAL (01-09-2026), y esta linea decia que si.
// superficies/acta exige sesion por lo mismo y no lleva lo mismo: esta ensena a
// los SUJETOS revisados, aquella a los ACTORES, o sea quien hizo que dentro de
// la organizacion. La frase de cara al usuario y el godoc de la superficie se
// corrigieron entonces; este comentario se quedo con la version vieja, que es
// como una afirmacion deja de ser verdad sin que nadie la toque.
func TestSinSesionLaPantallaDeAccesosNoEnsenaElCenso(t *testing.T) {
	u, err := construirUAR(opcionesUAR{Catalogo: catDePrueba(t)}) // sin Quien
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	u.ServeHTTP(rec, httptest.NewRequest("GET", "/uar/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("codigo %d, se esperaba 401", rec.Code)
	}
	cuerpo := rec.Body.String()
	if !strings.Contains(cuerpo, catDePrueba(t).Traducir("es", "uar.sin_sesion.titulo")) {
		t.Errorf("no dice que hay que entrar:\n%s", cuerpo)
	}
	// Y no se cuela ni una fila ni el estado vacio con sus ordenes.
	for _, prohibido := range []string{"<table", "plazum accesos ver"} {
		if strings.Contains(cuerpo, prohibido) {
			t.Errorf("sin sesion se ensena %q:\n%s", prohibido, cuerpo)
		}
	}
}

// COMPONER NO PUEDE PERDER LA ENUMERACION DE RUTAS.
//
// ESTE TEST EXISTE POR UN AGUJERO QUE ESTUVO A PUNTO DE ENTRAR. `serve` decide
// si las rutas de la aplicacion entran en la puerta que ENUMERA haciendo un type
// switch sobre Config.App: si implementa EnumeradorDePatrones, a cada ruta
// mutante suya le manda una peticion de verdad y exige un 403. Componer las dos
// superficies con un `http.ServeMux` pelado devuelve algo que NO implementa esa
// interfaz, asi que la puerta habria pasado a mirar CERO rutas de aplicacion
// -- las nuevas y las que ya entraban -- y a decir que todo bien.
//
// No habria puesto rojo nada: el CSRF por metodo las sigue cubriendo. Lo que se
// pierde en silencio es la puerta que comprueba que las cubre.
func TestComponerLasSuperficiesNoPierdeLaEnumeracionDeRutas(t *testing.T) {
	app := falsaApp{patrones: []string{"GET /alcance", "GET /hoy"}}
	u := uarDePrueba(t)
	compuesta := montarUAR(app, u)

	e, ok := compuesta.(interface{ Patrones() []string })
	if !ok {
		t.Fatal("la composicion no sabe decir sus patrones. `serve` decide con un type switch " +
			"sobre Config.App si las rutas de la aplicacion entran en la puerta que enumera: sin " +
			"esta interfaz, esa puerta pasa a mirar cero rutas y sigue verde")
	}
	tiene := map[string]bool{}
	for _, p := range e.Patrones() {
		tiene[p] = true
	}
	// Las de la aplicacion siguen estando...
	for _, p := range app.patrones {
		if !tiene[p] {
			t.Errorf("la composicion ha perdido el patron %q de la aplicacion", p)
		}
	}
	// ...y las mutantes nuevas tambien.
	for _, p := range u.Patrones() {
		if !tiene[p] {
			t.Errorf("la composicion ha perdido el patron %q de la revision de accesos", p)
		}
	}
	if len(e.Patrones()) != len(app.patrones)+len(u.Patrones()) {
		t.Errorf("patrones: %v", e.Patrones())
	}
}

// Y LAS RUTAS MUTANTES DE LA UAR LLEGAN AL SERVIDOR DE VERDAD, con su 403 sin
// token.
//
// No se comprueba que exista una comprobacion: se manda una peticion mutante por
// la cadena completa de middleware y se exige que la rechace.
func TestLasRutasMutantesDeLaUARExigenTokenCSRF(t *testing.T) {
	u := uarDePrueba(t)
	srv, err := serve.Nuevo(serve.Config{
		App:            montarUAR(falsaApp{patrones: []string{"GET /hoy"}}, u),
		CookieInsegura: true,
		Salida:         &strings.Builder{},
	})
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Handler()

	mutantes := 0
	for _, p := range u.Patrones() {
		metodo, ruta, ok := strings.Cut(p, " ")
		if !ok || metodo != "POST" {
			continue
		}
		mutantes++
		req := httptest.NewRequest(metodo, ruta, strings.NewReader(""))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s ha respondido %d sin token CSRF, y tenia que ser 403", p, rec.Code)
		}
	}
	if mutantes != 3 {
		t.Fatalf("se han probado %d rutas mutantes y la superficie declara 3: o han cambiado, "+
			"o este test esta midiendo el vacio", mutantes)
	}
	// Y el servidor las ENUMERA, que es la otra mitad: sin esto, la puerta de
	// serve las ignoraria aunque el middleware las cubriera.
	// Ruta guarda el metodo aparte del patron, asi que se pregunta por su
	// String(). La primera version de este test filtraba por Patron esperando
	// "POST /uar/..." dentro y salia vacia: habria dado por buena la enumeracion
	// mirando un campo que nunca la lleva.
	var vistas []string
	for _, r := range srv.Rutas() {
		if strings.HasPrefix(r.String(), "POST /uar/") {
			vistas = append(vistas, r.String())
		}
	}
	sort.Strings(vistas)
	quiero := []string{"POST /uar/cerrar", "POST /uar/decidir", "POST /uar/excusar"}
	if strings.Join(vistas, "|") != strings.Join(quiero, "|") {
		t.Fatalf("el servidor enumera %v y la superficie declara %v", vistas, quiero)
	}
}

// SIN FICHEROS CONFIGURADOS LA PANTALLA EXISTE IGUAL.
//
// Es la puerta D11-b llevada al arranque: una pantalla que desaparece cuando no
// hay datos deja al operador sin saber que existia. Es el mismo fallo que
// `plazum serve` tuvo con su propia orden, que estuvo semanas implementada y sin
// aparecer en la lista de uso.
func TestSinFicherosDeAccesosLaPantallaSigueExistiendoYExplicaComoUsarla(t *testing.T) {
	u := uarDePrueba(t)
	req := httptest.NewRequest("GET", "/uar/", nil)
	rec := httptest.NewRecorder()
	u.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("codigo %d: la pantalla no existe sin campana configurada", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "plazum accesos ver --fichero") {
		t.Errorf("y no dice como configurarla:\n%s", rec.Body.String())
	}
}
