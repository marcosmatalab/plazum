package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/superficies/pantallas"
	"github.com/marcosmatalab/plazum/superficies/serve"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/superficies/calendario"
	"github.com/marcosmatalab/plazum/superficies/camino"
	"github.com/marcosmatalab/plazum/superficies/documentos"
	"github.com/marcosmatalab/plazum/superficies/escalado"
)

// servidorDeLasOcho monta las OCHO plantillas, que es lo que esta puerta afirma.
//
// No se reusa `servidorDelCamino` porque ese arnes monta los pasos del camino y
// `documentos` NO es uno (se entra a proposito, ver su encabezado): con el, la
// puerta daba 404 en esa ruta y se quedaba en 14 de 16. Un arnes que no monta lo
// que la puerta dice medir es la medida que no ejerce el sistema.
func servidorDeLasOcho(t *testing.T, ps []*corpus.Paquete) http.Handler {
	t.Helper()
	quien := func(*http.Request) string { return "ciso" }
	cat := catDePrueba(t)

	app, err := pantallas.Nuevo(pantallas.Opciones{
		Paquetes:    ps,
		Catalogo:    cat,
		CaminoRuta:  camino.BasePorDefecto + "/",
		CaminoClave: camino.ClaveTitulo,
	})
	if err != nil {
		t.Fatal(err)
	}
	act, err := construirActa(cat, quien, nil)
	if err != nil {
		t.Fatal(err)
	}
	rev, err := construirUAR(opcionesUAR{Catalogo: cat, Quien: quien})
	if err != nil {
		t.Fatal(err)
	}
	cal, err := construirCalendario(cat, nil)
	if err != nil {
		t.Fatal(err)
	}
	esc, err := construirEscalado(cat, quien, nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := construirDocumentos(cat, quien, nil, nuevosIndicesPorCuenta(ps), nil)
	if err != nil {
		t.Fatal(err)
	}
	srv, err := serve.Nuevo(serve.Config{
		App: montarSuperficies(app,
			append(montajesDelCamino(caminoDePrueba(t), act, rev, cal, esc),
				montajesFueraDelCamino(doc)...)...),
		CookieInsegura: true,
		Salida:         &strings.Builder{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return srv.Handler()
}

// LA PUERTA DEL CONMUTADOR DE IDIOMA, sobre el conjunto que se despliega.
//
// # QUE AFIRMA, y por que sobre las OCHO y no sobre una
//
// Que TODA pantalla del producto responde en el idioma ELEGIDO aunque
// `Accept-Language` diga lo contrario. Se mide sobre las ocho plantillas
// montadas juntas porque el conmutador vive en el armazon compartido: una
// superficie que se olvide de invocarlo, o que resuelva el idioma por su cuenta,
// solo se ve mirandolas todas. Es el mismo motivo por el que la puerta del
// enlace de vuelta se mide sobre el conjunto y no sobre cada pantalla.
//
// # LAS DOS DIRECCIONES, Y POR QUE LA SEGUNDA NO ES DECORACION
//
// Se pide cada pantalla dos veces: con el navegador en castellano pidiendo
// ingles, y con el navegador en ingles pidiendo castellano. Con una sola
// direccion, un producto que sirviera SIEMPRE en ingles pasaria la mitad y
// nadie se enteraria. La contraria es la que descarta esa explicacion.
//
// # COMO SE COMPRUEBA QUE SALIO EN EL IDIOMA PEDIDO
//
// Por `<html lang>`, que es el idioma que la pagina DECLARA, y ademas por una
// cadena del catalogo que sea distinta en los dos idiomas. Solo con `lang` se
// aprobaria escribiendo el atributo y sirviendo el resto en castellano, que es
// el camino barato de esta medida; solo con la cadena, una pagina podria llevar
// el texto bien y mentir en `lang`, que es un fallo de accesibilidad real.
func TestLasOchoPantallasRespondenEnElIdiomaElegido(t *testing.T) {
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatal(err)
	}
	h := servidorDeLasOcho(t, ps)

	cat := catDePrueba(t)
	// EL TESTIGO SE SACA DEL CATALOGO REAL y no se escribe aqui: una cadena
	// escrita al lado del test es una tercera copia, y la que manda seria la
	// otra. Se usa el rotulo del conmutador, que existe en las ocho por
	// construccion y esta traducido de verdad ("Idioma" / "Language").
	const clave = "ui.idioma.rotulo"
	testigo := map[string]string{
		"es": cat.Traducir("es", clave),
		"en": cat.Traducir("en", clave),
	}
	if testigo["es"] == testigo["en"] || testigo["es"] == "" || testigo["en"] == "" {
		t.Fatalf("el testigo %q vale %q en es y %q en en. Si no son distintos y no vacios, "+
			"este test no puede distinguir un idioma de otro y estaria dando verde por "+
			"nada", clave, testigo["es"], testigo["en"])
	}

	// LAS OCHO PLANTILLAS, por la ruta que las pinta. Son ocho y no siete
	// porque el calendario tiene dos (la rejilla y la pagina de una cifra).
	// LAS RUTAS DE LOS PASOS SALEN DEL CAMINO CANONICO (camino.RutaDe) y no de
	// un literal: un literal se queda viejo el dia que un paso se remonte, y el
	// sintoma seria esta puerta dando 404 y pareciendo que la pantalla no
	// respeta el idioma.
	rutaDe := func(id string) string {
		r, hay := camino.RutaDe(id)
		if !hay {
			t.Fatalf("el camino canonico ya no declara el paso %q: esta puerta se quedo "+
				"vieja y hay que arreglarla, no quitar la fila", id)
		}
		return r
	}
	rutas := []struct{ nombre, ruta string }{
		{"pantallas (base.html)", "/alcance"},
		{"camino", camino.BasePorDefecto + "/"},
		{"acta", rutaDe(camino.IDDelActa)},
		{"calendario", calendario.BasePorDefecto + "/"},
		// La SEGUNDA plantilla del calendario, que es la que abre una cifra.
		{"calendario (cifra)", calendario.BasePorDefecto + "/" + calendario.RutaNoAlcanzados},
		{"escalado", escalado.BasePorDefecto + "/"},
		{"uar", rutaDe(camino.IDDeLaUAR)},
		{"documentos", documentos.BasePorDefecto + "/"},
	}

	comprobadas := 0
	for _, r := range rutas {
		for _, c := range []struct{ cabecera, pido string }{
			// Navegador en castellano pidiendo ingles: es la casilla.
			{"es-ES,es;q=0.9", "en"},
			// Y la contraria, que es la que descarta el sesgo.
			{"en-GB,en;q=0.9", "es"},
		} {
			t.Run(r.nombre+" con "+c.cabecera+" pide "+c.pido, func(t *testing.T) {
				sep := "?"
				if strings.Contains(r.ruta, "?") {
					sep = "&"
				}
				req := httptest.NewRequest(http.MethodGet,
					r.ruta+sep+camino.ParametroIdioma+"="+c.pido, nil)
				req.Header.Set("Accept-Language", c.cabecera)
				w := httptest.NewRecorder()
				h.ServeHTTP(w, req)

				cuerpo := w.Body.String()
				// UN 401 O UN 404 NO VALEN COMO VERDE. Una pantalla que no
				// responde no demuestra que respete el idioma, y sin esto la
				// puerta pasaria entera el dia que el montaje se rompa.
				if w.Code != http.StatusOK {
					t.Fatalf("%s dio %d y no 200. Una pantalla que no responde no puede "+
						"demostrar nada sobre su idioma", r.ruta, w.Code)
				}
				if !strings.Contains(cuerpo, `<html lang="`+c.pido+`"`) {
					t.Errorf("%s no declara lang=%q.\nUna pagina que declara un idioma que "+
						"no lleva dentro rompe los lectores de pantalla", r.ruta, c.pido)
				}
				if !strings.Contains(cuerpo, testigo[c.pido]) {
					t.Errorf("%s no trae el texto en %s (falta %q).\nDeclarar el idioma en "+
						"lang y servir el texto en otro es como se aprueba esta medida sin "+
						"traducir nada", r.ruta, c.pido, testigo[c.pido])
				}
				// Y NO TRAE EL DEL OTRO: sin esto, una pagina que pintara los
				// dos idiomas a la vez pasaria.
				otro := "es"
				if c.pido == "es" {
					otro = "en"
				}
				if strings.Contains(cuerpo, testigo[otro]) {
					t.Errorf("%s trae ADEMAS el rotulo en %s (%q): la pagina esta mezclando "+
						"idiomas", r.ruta, otro, testigo[otro])
				}
				comprobadas++
			})
		}
	}

	// EL CARDINAL, con igualdad exacta, para que tambien rompa cuando el
	// conjunto ENCOJA: una pantalla que deje de montarse saldria de la lista sin
	// que nada mas se pusiera rojo.
	if quiero := len(rutas) * 2; comprobadas != quiero {
		t.Errorf("se comprobaron %d combinaciones y tenian que ser %d (%d plantillas x 2 "+
			"direcciones). O una pantalla dejo de responder, o la lista se quedo vieja",
			comprobadas, quiero, len(rutas))
	}
	// EL RESUMEN SOLO SI LA MEDIDA SE SOSTUVO. Sin esto, la salida roja termina
	// diciendo «todas en el idioma elegido» tres lineas debajo del error que
	// dice lo contrario, que es la afirmacion acompanada en el sitio donde mas
	// se cree, el final del log.
	if t.Failed() {
		return
	}
	t.Logf("%d plantillas x 2 direcciones = %d combinaciones, todas en el idioma elegido "+
		"pese al Accept-Language contrario", len(rutas), comprobadas)
}

// Y EL CONTROL NEGATIVO: sin elegir nada, manda Accept-Language.
//
// Sin esta mitad, el test de arriba pasaria igual si el conmutador hubiera
// SUSTITUIDO la negociacion en vez de anteponerse a ella, y entonces un
// navegador en ingles sin cookie veria el producto en castellano. Es la
// regresion mas probable de este cambio y la que nadie mira.
func TestSinElegirNadaSigueMandandoAcceptLanguage(t *testing.T) {
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatal(err)
	}
	h := servidorDeLasOcho(t, ps)

	for _, c := range []struct{ cabecera, espero string }{
		{"en-GB,en;q=0.9", "en"},
		{"es-ES,es;q=0.9", "es"},
		// Y una que no es ninguno de los dos: cae al de por defecto, sin error.
		{"fr-FR,fr;q=0.9", "es"},
	} {
		req := httptest.NewRequest(http.MethodGet, "/alcance", nil)
		req.Header.Set("Accept-Language", c.cabecera)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if !strings.Contains(w.Body.String(), `<html lang="`+c.espero+`"`) {
			t.Errorf("con Accept-Language %q se esperaba lang=%q.\nEl conmutador tiene que "+
				"ANTEPONERSE a la negociacion, no sustituirla", c.cabecera, c.espero)
		}
	}
}

// Y LA ELECCION SOBREVIVE A LA SIGUIENTE PAGINA, que es lo que la hace util.
//
// Se comprueba de extremo a extremo: se pide con parametro, se recoge la cookie
// que la respuesta escribe, y se pide OTRA pantalla mandando esa cookie y el
// Accept-Language contrario. Sin esta puerta, el conmutador podria estar
// funcionando por el parametro y no persistir nada, y el sintoma seria que se
// vuelve al castellano al pulsar cualquier enlace.
func TestLaEleccionDeIdiomaSobreviveALaSiguientePagina(t *testing.T) {
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatal(err)
	}
	h := servidorDeLasOcho(t, ps)

	// 1. Se elige ingles en una pantalla.
	req := httptest.NewRequest(http.MethodGet, "/alcance?"+camino.ParametroIdioma+"=en", nil)
	req.Header.Set("Accept-Language", "es-ES,es;q=0.9")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var galleta *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == camino.CookieIdioma {
			galleta = c
		}
	}
	if galleta == nil {
		t.Fatalf("elegir idioma no dejo la cookie %q. Sin ella la eleccion dura una "+
			"pagina, y el sintoma es que se vuelve al castellano al pulsar un enlace",
			camino.CookieIdioma)
	}

	// 2. Y se pide OTRA pantalla, sin parametro y con la cabecera contraria.
	req2 := httptest.NewRequest(http.MethodGet, calendario.BasePorDefecto+"/", nil)
	req2.Header.Set("Accept-Language", "es-ES,es;q=0.9")
	req2.AddCookie(galleta)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req2)

	if !strings.Contains(w2.Body.String(), `<html lang="en"`) {
		t.Errorf("la segunda pantalla volvio al castellano pese a la cookie.\n" +
			"La eleccion tiene que sobrevivir a la navegacion o el conmutador no sirve")
	}
}
