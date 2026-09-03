package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/pantalla"
	"github.com/marcosmatalab/plazum/superficies/acta"
	"github.com/marcosmatalab/plazum/superficies/calendario"
	"github.com/marcosmatalab/plazum/superficies/camino"
	escaladoWeb "github.com/marcosmatalab/plazum/superficies/escalado"
	"github.com/marcosmatalab/plazum/superficies/pantallas"
	"github.com/marcosmatalab/plazum/superficies/serve"
)

// LA PUERTA DEL CAMINO GUIADO: DE PUNTA A PUNTA Y SIN CALLEJONES.
//
// Que vigila, dicho como propiedad y no como lista de comprobaciones: que desde
// `plazum serve` se pueda recorrer el camino entero sin teclear una direccion a
// mano y sin quedarse en ninguna pantalla desde la que no se pueda seguir.
//
// POR QUE HACIA FALTA. Antes de esto, la pantalla del acta estaba construida y
// no la montaba nadie (cero direcciones del producto llevaban a ella), la
// revision de accesos estaba montada y nadie enlazaba a ella, y el orden en que
// se recorre esto no constaba en ningun sitio. Nada de eso ponia rojo un solo
// test, porque cada pieza pasaba la suya: es el fallo de las juntas, que es
// donde no mira ninguna de las dos partes.
//
// COMO SE COMPRUEBA, y esto es lo que separa esta puerta de un test de humo: NO
// se recorre una lista de rutas escrita aqui. Se recorre camino.Canonico(), que
// es la declaracion que el producto usa para montar, asi que un paso nuevo entra
// en la puerta el dia que se declara, y un paso cuya ruta cambie se comprueba en
// su ruta nueva. Una lista escrita al lado del test seria la segunda copia y se
// desincronizaria justo el dia que hiciera falta.

func caminoDePrueba(t *testing.T) *camino.Superficie {
	t.Helper()
	c, err := construirCamino(catDePrueba(t))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func actaDePrueba(t *testing.T) *acta.Superficie {
	t.Helper()
	// Con operador: sin el, la pantalla contesta 401 y ensena el estado de sin
	// sesion, que tambien lleva el camino y se comprueba aparte.
	a, err := construirActa(catDePrueba(t), func(*http.Request) string { return "ciso" }, nil)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// servidorDelCamino monta lo mismo que monta `plazum serve`, con el corpus que
// se le pase. Con ps nil las seis pantallas existen igual (se pintan vacias con
// su explicacion), que basta para medir las juntas; la entrevista necesita el de
// verdad, porque la superficie reconstruye la consulta desde las preguntas que
// conoce.
func servidorDelCamino(t *testing.T, ps []*corpus.Paquete,
	quien func(*http.Request) string) *serve.Servidor {

	t.Helper()
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
	// Las dos pantallas nuevas se montan SIN FUENTE, igual que el acta: asi
	// existen y pintan su estado vacio, que es lo que hace falta para medir las
	// juntas. Con fuente haria falta un alcance en disco, y esta puerta mide
	// cableado, no derivacion.
	cal, err := construirCalendario(cat, nil)
	if err != nil {
		t.Fatal(err)
	}
	esc, err := construirEscalado(cat, quien, nil)
	if err != nil {
		t.Fatal(err)
	}
	srv, err := serve.Nuevo(serve.Config{
		App: montarSuperficies(app,
			montajesDelCamino(caminoDePrueba(t), act, rev, cal, esc)...),
		CookieInsegura: true,
		Salida:         &strings.Builder{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return srv
}

// pedirCamino hace un GET por la cadena completa de middleware.
func pedirCamino(t *testing.T, h http.Handler, ruta string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, ruta, nil)
	req.Header.Set("Accept-Language", "es")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func TestElCaminoGuiadoNoTieneCallejones(t *testing.T) {
	cat := catDePrueba(t)
	h := servidorDelCamino(t, nil, func(*http.Request) string { return "ciso" }).Handler()

	pasos := camino.Canonico()
	if len(pasos) < 6 {
		t.Fatalf("el camino declara %d pasos y la casilla de la v1 nombra seis (alcance, "+
			"calendario, derivacion, acta, revision de accesos y escalado). Con menos, esta "+
			"puerta estaria midiendo otro camino", len(pasos))
	}

	// LA PANTALLA DEL CAMINO CONTESTA. Si esto falla, todo lo de abajo mide el
	// contenido de una pagina de error.
	codigo, portada := pedirCamino(t, h, camino.BasePorDefecto+"/")
	if codigo != http.StatusOK {
		t.Fatalf("GET %s/ ha respondido %d. Sin la pantalla del camino no hay camino: las "+
			"piezas vuelven a estar sueltas", camino.BasePorDefecto, codigo)
	}

	vueltaAlCamino := `href="` + camino.BasePorDefecto + `/"`
	conPantalla := 0
	for _, p := range pasos {
		rotulo := cat.Traducir("es", p.Titulo)
		if rotulo == p.Titulo {
			t.Errorf("el paso %q sale con la clave %q en crudo: el catalogo no la tiene",
				p.ID, p.Titulo)
		}
		if !strings.Contains(portada, rotulo) {
			t.Errorf("la pantalla del camino no nombra el paso %q (%q)", p.ID, rotulo)
		}
		// EL VERBO ES LO QUE CONVIERTE UNA LISTA DE PANTALLAS EN UN CAMINO. Sin
		// el, esto es un menu con otro nombre: dice donde ir y no dice a que.
		verbo := cat.Traducir("es", p.Verbo)
		if verbo == p.Verbo || !strings.Contains(portada, verbo) {
			t.Errorf("la pantalla del camino no dice que se hace en el paso %q. Un paso sin "+
				"verbo es un enlace suelto, no un paso", p.ID)
		}

		if !p.EsPantalla() {
			// SIN PANTALLA NO ES UN CALLEJON: sale la orden exacta que lo hace
			// hoy. Es la puerta D11-b aplicada a un paso.
			if p.Comando == "" || !strings.Contains(portada, p.Comando) {
				t.Errorf("el paso %q todavia no es pantalla y la pagina no dice con que orden "+
					"se hace hoy. Eso es un callejon: se llega y no hay nada que hacer", p.ID)
			}
			continue
		}
		conPantalla++

		// EL CAMINO ENLAZA AL PASO...
		if !strings.Contains(portada, `href="`+p.Ruta+`"`) {
			t.Errorf("la pantalla del camino no enlaza al paso %q en %q, asi que para llegar "+
				"hay que teclear la direccion", p.ID, p.Ruta)
		}
		// ...EL PASO CONTESTA DE VERDAD, por la cadena completa de middleware...
		codigo, cuerpo := pedirCamino(t, h, p.Ruta)
		if codigo == http.StatusNotFound {
			t.Errorf("GET %s (paso %q) ha respondido 404. El camino declara ese paso como "+
				"pantalla y `plazum serve` no la monta ahi: el enlace de la pantalla del "+
				"camino lleva a la nada", p.Ruta, p.ID)
			continue
		}
		// ...Y DESDE EL PASO SE PUEDE VOLVER. Sin esto, cada paso es una hoja
		// suelta: se entra y solo se sale por el boton de atras del navegador.
		if !strings.Contains(cuerpo, vueltaAlCamino) {
			t.Errorf("la pantalla del paso %q (%s, codigo %d) no enlaza de vuelta al camino "+
				"guiado. Se entra y no hay por donde seguir", p.ID, p.Ruta, codigo)
		}
	}
	if conPantalla < 4 {
		t.Fatalf("solo %d pasos del camino son pantalla, y hoy tienen que ser al menos cuatro "+
			"(alcance, derivacion, acta y revision de accesos). Con menos, esta puerta "+
			"apenas recorre nada", conPantalla)
	}
}

// LOS DOS ESTADOS EN LOS QUE MAS FALTA HACE: sin sesion.
//
// El acta y la revision de accesos llevan nombres de personas, asi que sin
// sesion contestan 401 y no ensenan nada. Ese es exactamente el momento en el
// que alguien se queda mirando una pagina que no le dice nada, o sea el momento
// en el que un callejon duele: la vuelta al camino tiene que seguir ahi.
func TestSinSesionElActaYLaUARSiguenEnsenandoElCamino(t *testing.T) {
	h := servidorDelCamino(t, nil, nil).Handler() // sin corpus y sin operador
	vuelta := `href="` + camino.BasePorDefecto + `/"`
	probadas := 0
	for _, id := range []string{"acta", "uar"} {
		ruta, esPantalla := camino.RutaDe(id)
		if !esPantalla {
			continue
		}
		probadas++
		codigo, cuerpo := pedirCamino(t, h, ruta)
		if codigo != http.StatusUnauthorized {
			t.Errorf("GET %s sin sesion ha respondido %d y esperaba 401: esta pantalla lleva "+
				"nombres de personas", ruta, codigo)
		}
		if !strings.Contains(cuerpo, vuelta) {
			t.Errorf("la pantalla de %q sin sesion no enlaza de vuelta al camino guiado: es "+
				"el estado en el que quien llega mas necesita saber por donde se sigue", id)
		}
	}
	if probadas != 2 {
		t.Fatalf("se han probado %d pantallas con sesion y tenian que ser 2: o han dejado de "+
			"ser pantalla, o este test esta midiendo el vacio", probadas)
	}
}

// CONTROL NEGATIVO DEL ENLACE DE VUELTA.
//
// Sin esto, las comprobaciones de arriba podrian estar pasando por cualquier
// otra razon (una direccion que aparezca en el CSS, un comentario, una
// coincidencia). Aqui se montan las MISMAS superficies sin camino y se exige que
// la direccion NO aparezca en ninguna: si aparece igual, lo que miden los tests
// de arriba no es el enlace que creen medir.
func TestControlNegativoSinCaminoNoAparecenEnlacesInventados(t *testing.T) {
	cat := catDePrueba(t)
	app, err := pantallas.Nuevo(pantallas.Opciones{Catalogo: cat}) // sin camino
	if err != nil {
		t.Fatal(err)
	}
	act, err := acta.Nuevo(acta.Opciones{
		Catalogo: cat, Base: "/acta", Estatico: "/estatico",
		Quien: func(*http.Request) string { return "ciso" },
	})
	if err != nil {
		t.Fatal(err)
	}
	srv, err := serve.Nuevo(serve.Config{
		App:    montarSuperficies(app, montaje{prefijo: "/acta/", h: act}),
		Salida: &strings.Builder{},
	})
	if err != nil {
		t.Fatal(err)
	}
	h := srv.Handler()
	vuelta := `href="` + camino.BasePorDefecto + `/"`
	for _, ruta := range []string{"/alcance", "/controles", "/acta/"} {
		_, cuerpo := pedirCamino(t, h, ruta)
		if strings.Contains(cuerpo, vuelta) {
			t.Errorf("%s pinta el enlace al camino sin que nadie se lo haya dado. Entonces la "+
				"puerta de arriba no esta midiendo el cableado, esta midiendo una constante",
				ruta)
		}
	}
}

// EL VIAJE DE IDA Y VUELTA NO BORRA LA ENTREVISTA.
//
// LA PROPIEDAD QUE SE INTENTO TUMBAR, y cayo. Las respuestas de la entrevista
// no se guardan en ningun sitio: viajan en la direccion de la pagina, y la
// pantalla de Alcance lo dice con esas palabras. El camino guiado se monto con
// enlaces pelados, asi que el recorrido natural
//
//	/alcance?si=... -> Camino guiado -> Alcance
//
// devolvia una entrevista EN BLANCO. O sea que la pieza construida para no
// perder a nadie era la unica del producto capaz de borrar el trabajo de quien
// la usaba, y ninguna de las puertas de reachability lo veia: todas miraban que
// el enlace existiera, ninguna que llevara algo dentro.
//
// Aqui se recorre el viaje entero contra el servidor montado, ida y vuelta.
func TestElViajeDeIdaYVueltaAlCaminoNoBorraLaEntrevista(t *testing.T) {
	// CON EL CORPUS DE VERDAD, y hace falta: la superficie de pantallas
	// RECONSTRUYE la consulta desde las preguntas que conoce, asi que un
	// identificador inventado se cae por el camino y el test estaria midiendo
	// el vacio. La pregunta se saca del corpus instalado, no se escribe aqui.
	ps := corpusInstalado(t)
	pregunta := unaPreguntaDeAlcance(t, ps)
	h := servidorDelCamino(t, ps, func(*http.Request) string { return "ciso" }).Handler()
	respuestas := "si=" + pregunta

	// IDA: desde Alcance con respuestas, el enlace al camino las lleva.
	codigo, alcance := pedirCamino(t, h, "/alcance?"+respuestas)
	if codigo != http.StatusOK {
		t.Fatalf("GET /alcance ha dado %d", codigo)
	}
	aCamino := `href="` + camino.BasePorDefecto + `/?` + respuestas + `"`
	if !strings.Contains(alcance, aCamino) {
		t.Errorf("desde la entrevista respondida, el enlace al camino va pelado, asi que "+
			"pulsarlo borra las respuestas.\n  Esperaba: %s", aCamino)
	}

	// VUELTA: el camino devuelve a Alcance con las mismas respuestas.
	codigo, hub := pedirCamino(t, h, camino.BasePorDefecto+"/?"+respuestas)
	if codigo != http.StatusOK {
		t.Fatalf("GET del camino con respuestas ha dado %d", codigo)
	}
	aAlcance := `href="/alcance?` + respuestas + `"`
	if !strings.Contains(hub, aAlcance) {
		t.Errorf("el camino devuelve a una entrevista en blanco.\n  Esperaba: %s", aAlcance)
	}

	// Y EL VIAJE COMPLETO: se sigue el enlace de vuelta y la entrevista trae la
	// respuesta puesta. Sin este tramo, lo de arriba solo compara cadenas
	// dentro de una pagina.
	codigo, vuelta := pedirCamino(t, h, "/alcance?"+respuestas)
	if codigo != http.StatusOK {
		t.Fatalf("la vuelta a Alcance ha dado %d", codigo)
	}
	if !strings.Contains(vuelta, rotuloDeRespondidaSi(t)) {
		t.Error("tras el viaje de ida y vuelta, la entrevista no dice que esa pregunta este " +
			"respondida: el camino se ha comido el trabajo")
	}
	// CONTROL POSITIVO DEL DETECTOR: sin respuestas, el enlace va pelado y no
	// se inventa ninguna. Sin esta rama, un enlace que llevara siempre la
	// misma cadena pasaria las comprobaciones de arriba.
	_, limpio := pedirCamino(t, h, "/alcance")
	if !strings.Contains(limpio, `href="`+camino.BasePorDefecto+`/"`) {
		t.Error("sin respuestas, el enlace al camino no va pelado: se esta inventando una " +
			"consulta que el operador no ha dado")
	}
}

// corpusInstalado carga el corpus del repositorio. Es el mismo que sirve
// `plazum serve` cuando se apunta a paquetes/.
func corpusInstalado(t *testing.T) []*corpus.Paquete {
	t.Helper()
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatalf("cargar el corpus: %v", err)
	}
	if len(ps) == 0 {
		t.Fatal("el corpus esta vacio, asi que este test no recorre ninguna entrevista")
	}
	return ps
}

// unaPreguntaDeAlcance da el identificador de una pregunta real. No se escribe
// aqui: un identificador de pregunta escrito al lado del test es la segunda
// copia, y el dia que el paquete lo renombre este test seguiria verde midiendo
// una pregunta que ya no existe.
func unaPreguntaDeAlcance(t *testing.T, ps []*corpus.Paquete) string {
	t.Helper()
	for _, p := range pantalla.Derivar(ps) {
		if p.ID == pantalla.Alcance && len(p.Preguntas) > 0 {
			return p.Preguntas[0].ID
		}
	}
	t.Fatal("el corpus instalado no trae ninguna pregunta de alcance")
	return ""
}

func rotuloDeRespondidaSi(t *testing.T) string {
	t.Helper()
	return catDePrueba(t).Traducir("es", "alcance.pregunta.respondida_si")
}

// LAS ORDENES QUE OFRECE EL CAMINO SE PEGAN Y FUNCIONAN.
//
// POR QUE EXISTE, y salio de la pasada del comprador: los dos pasos que
// todavia no son pantalla salen con su orden en un bloque de codigo, o sea en
// un bloque que INVITA A COPIAR. La primera version decia
// `plazum calendario --pais=ES --sector=SECTOR --empleados=N`, copiada de la
// linea de uso del propio comando, y al pegarla contesta
// `invalid value "N" for flag -empleados`. Una orden que falla al pegarla es un
// callejon con luz, que es el mismo fallo por el que existe
// TestServeSinCorpusSenalaElDelDemoCuandoEstaDelante.
//
// COMO SE COMPRUEBA SIN EJECUTAR EL TRABAJO. Se le da a cada orden un
// directorio de corpus VACIO. Entonces:
//
//	codigo 2   las opciones no parsean. Es el fallo que se busca.
//	codigo 1   parsean y el trabajo no se puede hacer aqui (no hay corpus, no
//	           hay fichero de alcance). Eso es correcto para este test.
//
// Distinguir los dos codigos es lo que hace que esto mida el pegado y no el
// entorno. Y las ordenes NO se escriben aqui: salen de camino.Canonico(), asi
// que una orden nueva entra en la puerta el dia que se declara.
func TestLasOrdenesQueOfreceElCaminoParsean(t *testing.T) {
	// Cada orden del camino, con quien la ejecuta. Si aparece una orden que no
	// esta aqui, el test se para: es la senal de que alguien anadio un paso con
	// una orden que nadie ha comprobado.
	ejecutores := map[string]func([]string, io.Writer, io.Writer) int{
		"calendario": cmdCalendario,
		"escalado":   cmdEscalado,
	}
	vacio := t.TempDir()
	probadas := 0
	for _, p := range camino.Canonico() {
		if p.Comando == "" {
			continue
		}
		campos := strings.Fields(p.Comando)
		if len(campos) < 2 || campos[0] != "plazum" {
			t.Errorf("el paso %q ofrece %q, que no es una orden de plazum", p.ID, p.Comando)
			continue
		}
		orden, args := campos[1], campos[2:]
		ejecutar, hay := ejecutores[orden]
		if !hay {
			t.Fatalf("el paso %q ofrece `plazum %s` y este test no sabe ejecutarla. "+
				"Anadela a ejecutores, o el camino estara ofreciendo una orden que nadie "+
				"comprueba", p.ID, orden)
		}
		probadas++
		var salida, errores strings.Builder
		rc := ejecutar(append(args, "--corpus", vacio), &salida, &errores)
		if rc == 2 {
			t.Errorf("el paso %q ofrece `%s` y al pegarla no parsea (codigo 2):\n%s",
				p.ID, p.Comando, errores.String())
		}
	}
	if probadas < 2 {
		t.Fatalf("solo se han probado %d ordenes y el camino declara dos pasos sin pantalla: "+
			"o han ganado su pantalla, o este test mide el vacio", probadas)
	}
	// CONTROL POSITIVO DEL DETECTOR: una orden mal escrita SI da 2. Sin esto, un
	// ejecutor que devolviera cualquier cosa menos 2 dejaria el test en verde
	// para siempre.
	var salida, errores strings.Builder
	if rc := cmdCalendario([]string{"--empleados=N"}, &salida, &errores); rc != 2 {
		t.Errorf("una opcion que no parsea ha devuelto %d y tenia que ser 2: este test no "+
			"esta midiendo el parseo", rc)
	}
	// Y LA ORDEN SE DESPACHA DE VERDAD desde main. Sin esto, el camino podria
	// ofrecer `plazum calendario` el dia que main dejara de tener ese caso.
	fuente, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	despacha, _ := ordenesDeMain(t, string(fuente))
	tiene := map[string]bool{}
	for _, o := range despacha {
		tiene[o] = true
	}
	for orden := range ejecutores {
		if !tiene[orden] {
			t.Errorf("el camino ofrece `plazum %s` y main.go no despacha esa orden", orden)
		}
	}
}

// MEDIO ENLACE NO SE CONSTRUYE, en las tres superficies que lo pintan.
//
// Una direccion sin rotulo pinta un enlace sin palabras y un rotulo sin
// direccion pinta uno que no lleva a ningun sitio. Las dos mitades salen del
// mismo descuido, y las dos dan una pantalla peor que la de antes. Y la
// direccion tiene que ser de este sitio: con dos barras al principio el
// navegador la lee como otro anfitrion, y el enlace que existe para no perder a
// nadie sacaria al operador de plazum.
func TestUnEnlaceDelCaminoAMediasNoSeConstruye(t *testing.T) {
	cat := catDePrueba(t)
	casos := []struct {
		nombre string
		url    string
		clave  string
	}{
		{"direccion sin rotulo", "/camino/", ""},
		{"rotulo sin direccion", "", camino.ClaveTitulo},
		{"direccion a otro anfitrion", "//evil.example/camino/", camino.ClaveTitulo},
		{"direccion relativa", "camino/", camino.ClaveTitulo},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if _, err := pantallas.Nuevo(pantallas.Opciones{
				Catalogo: cat, CaminoRuta: c.url, CaminoClave: c.clave,
			}); err == nil {
				t.Error("pantallas se ha construido con un enlace al camino que no vale")
			}
			if _, err := acta.Nuevo(acta.Opciones{
				Catalogo: cat, CaminoRuta: c.url, CaminoClave: c.clave,
			}); err == nil {
				t.Error("el acta se ha construido con un enlace al camino que no vale")
			}
		})
	}
	// CONTROL POSITIVO: el enlace entero SI se construye. Sin esta rama, un
	// validador que rechazara siempre pasaria las cuatro de arriba.
	if _, err := pantallas.Nuevo(pantallas.Opciones{
		Catalogo: cat, CaminoRuta: camino.BasePorDefecto + "/", CaminoClave: camino.ClaveTitulo,
	}); err != nil {
		t.Errorf("el enlace bueno tampoco se construye, asi que el validador dice que no a "+
			"todo: %v", err)
	}
}

// LAS DOS PANTALLAS QUE EL CAMINO TODAVIA NO DECLARA, PROBADAS EN EL SERVIDOR
// DE VERDAD.
//
// POR QUE HACE FALTA UNA PUERTA APARTE. TestElCaminoGuiadoNoTieneCallejones
// recorre camino.Canonico() y solo pide las rutas de los pasos que SON pantalla.
// El calendario y el escalado se montan hoy en su propia base porque el camino
// declara sus pasos SIN `Ruta` (ese fichero es de otro frente), asi que esa
// puerta NO LAS MIRA: estarian montadas, servidas y sin una sola comprobacion,
// que es exactamente el estado en el que estuvo la pantalla del acta.
//
// Aqui se piden por la cadena completa de middleware, en su prefijo real, y se
// les exige lo mismo que a cualquier paso: que contesten y que se pueda volver.
func TestElCalendarioYElEscaladoContestanEnElServidorMontado(t *testing.T) {
	h := servidorDelCamino(t, nil, func(*http.Request) string { return "ciso" }).Handler()
	vuelta := `href="` + camino.BasePorDefecto + `/"`

	casos := []struct{ nombre, ruta string }{
		{camino.IDDelCalendario,
			prefijoDeLaPantalla(camino.IDDelCalendario, calendario.BasePorDefecto) + "/"},
		{camino.IDDelEscalado,
			prefijoDeLaPantalla(camino.IDDelEscalado, escaladoWeb.BasePorDefecto) + "/"},
	}
	for _, c := range casos {
		codigo, cuerpo := pedirCamino(t, h, c.ruta)
		if codigo == http.StatusNotFound {
			t.Errorf("GET %s (%s) ha respondido 404: `plazum serve` no la monta ahi, asi que "+
				"la pantalla existe y no se llega a ella por ninguna direccion del producto",
				c.ruta, c.nombre)
			continue
		}
		if !strings.Contains(cuerpo, vuelta) {
			t.Errorf("la pantalla de %q (%s, codigo %d) no enlaza de vuelta al camino guiado. "+
				"Se entra y no hay por donde seguir", c.nombre, c.ruta, codigo)
		}
	}

	// Y EL .ics DEL CALENDARIO TAMBIEN ESTA COLGADO. Sin alcance contesta 404
	// (un calendario vacio importado en Outlook dice «no tienes nada que
	// hacer»), pero tiene que ser el 404 DE LA SUPERFICIE y no el del
	// enrutador: si el montaje estuviera mal, el cuerpo seria el 404 pelado de
	// Go y esta comprobacion habria pasado por el motivo equivocado.
	ruta := prefijoDeLaPantalla(camino.IDDelCalendario, calendario.BasePorDefecto) + "/" +
		calendario.FicheroICS
	codigo, cuerpo := pedirCamino(t, h, ruta)
	if codigo != http.StatusNotFound || strings.Contains(cuerpo, "404 page not found") {
		t.Errorf("GET %s ha respondido %d con el cuerpo %q. Sin alcance tiene que contestar "+
			"el 404 de la superficie, con su explicacion", ruta, codigo, recortar120(cuerpo))
	}
}

// recortar120 acorta un cuerpo para que quepa en un mensaje de fallo.
func recortar120(s string) string {
	if len(s) <= 120 {
		return s
	}
	return s[:120] + "..."
}
