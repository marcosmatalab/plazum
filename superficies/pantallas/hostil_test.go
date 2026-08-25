package pantallas

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"dutiq/nucleo/corpus"
)

// La pasada del atacante.
//
// El corpus lo escribe un tercero: es entrada hostil, no dato de confianza. Y
// la peticion la escribe cualquiera. Lo que se comprueba aqui es lo que pasa
// cuando los dos hacen lo peor que se les ocurre.

// elGuion es el trozo que no puede salir vivo de ninguna plantilla.
const elGuion = `<script>alert(1)</script>`

// paqueteHostil mete el mismo guion en TODOS los campos de texto que la
// interfaz pinta. Si alguno se escapase mal, este test lo dice y ademas dice
// cual, porque cada campo lleva su marca.
func paqueteHostil() *corpus.Paquete {
	con := func(campo string) string { return elGuion + "[" + campo + "]" }
	return &corpus.Paquete{
		URN: "urn:demo:" + con("urn"), Version: con("version"), Clase: corpus.Propio,
		Licencia: con("licencia"), Fuente: `" onload="alert(2)`,
		Entidades: []corpus.TipoEntidad{{
			Nombre: con("entidad"), Descripcion: con("descripcion"),
			Atributos: []corpus.Atributo{{
				Nombre: con("atributo"), Tipo: corpus.Enumerado,
				Valores: []string{con("valor")}, Obligado: true,
				Ayuda: con("ayuda_atributo"), Cita: con("cita_atributo")}},
		}},
		Preguntas: []corpus.Pregunta{{
			ID: con("id_pregunta"), Texto: con("texto_pregunta"),
			Cita: con("cita_pregunta"), Entidad: con("entidad"),
			Atributo: con("atributo"), Ayuda: con("ayuda_pregunta"),
			Desbloquea: []string{con("id_obligacion")}}},
		Obligaciones: []corpus.Obligacion{{
			ID: con("id_obligacion"), Articulo: con("articulo"),
			Cita: con("cita_obligacion"), ClaseE2E: con("clase"),
			Entregable: con("id_plantilla"),
			Preguntas:  []string{con("id_pregunta")},
			Temporalidad: &corpus.Temporalidad{Primitiva: con("primitiva"),
				Cadencia: con("cadencia"), Limite: con("limite")}}},
		Plantillas: []corpus.Plantilla{{
			ID: con("id_plantilla"), Titulo: con("titulo_plantilla"),
			Cita: con("cita_plantilla")}},
	}
}

// paqueteRoto es un paquete con las dos erratas que no dan error en ningun
// sitio: una obligacion condicionada a una pregunta que el paquete no declara,
// y un entregable declarado por una obligacion que tampoco existe.
func paqueteRoto() *corpus.Paquete {
	return &corpus.Paquete{
		URN: "urn:demo:roto", Version: "1", Clase: corpus.Propio,
		Preguntas: []corpus.Pregunta{{ID: "roto.q.hay", Texto: "Una pregunta que si existe",
			Cita: "demo roto art. 1", Desbloquea: []string{"roto.o.si"}}},
		Obligaciones: []corpus.Obligacion{
			{ID: "roto.o.si", Articulo: "1", Cita: "demo roto art. 1",
				ClaseE2E: "documental", Preguntas: []string{"roto.q.hay"}},
			{ID: "roto.o.colgada", Articulo: "2", Cita: "demo roto art. 2",
				ClaseE2E: "documental", Preguntas: []string{"roto.q.no-existe"}},
		},
		Plantillas: []corpus.Plantilla{{ID: "roto.pl.sola", Titulo: "Sin dueno",
			Cita: "demo roto art. 3"}},
	}
}

// paqueteSinPreguntas declara datos y obligaciones pero ninguna pregunta de
// alcance. Es un paquete perfectamente valido, y deja la entrevista sin nada
// que responder: la pantalla tiene que decirlo en vez de ensenar una lista
// vacia sin explicacion.
func paqueteSinPreguntas() *corpus.Paquete {
	return &corpus.Paquete{
		URN: "urn:demo:sinpreguntas", Version: "1", Clase: corpus.Propio,
		Entidades: []corpus.TipoEntidad{{Nombre: "sujeto",
			Atributos: []corpus.Atributo{{Nombre: "nif", Tipo: corpus.Texto,
				Obligado: true, Cita: "demo sinp art. 1"}}}},
		Obligaciones: []corpus.Obligacion{{ID: "sinp.o.1", Articulo: "1",
			Cita: "demo sinp art. 1", ClaseE2E: "documental"}},
	}
}

// paqueteGrande produce n obligaciones. Sirve para el corpus enorme y para la
// paginacion.
func paqueteGrande(n int) *corpus.Paquete {
	p := &corpus.Paquete{URN: "urn:demo:grande", Version: "1", Clase: corpus.Propio,
		Preguntas: []corpus.Pregunta{{ID: "grande.q.1", Texto: "Una pregunta",
			Cita: "demo grande art. 1"}}}
	for i := 0; i < n; i++ {
		p.Obligaciones = append(p.Obligaciones, corpus.Obligacion{
			ID: fmt.Sprintf("grande.o.%05d", i), Articulo: fmt.Sprint(i),
			Cita: "demo grande art. " + fmt.Sprint(i), ClaseE2E: "documental",
			Preguntas: []string{"grande.q.1"}})
	}
	return p
}

// ---------------------------------------------------------------------------
// Corpus hostil
// ---------------------------------------------------------------------------

// Un paquete con un guion en cada campo no ejecuta nada en ninguna pantalla, y
// el texto sigue viendose, escapado.
func TestUnCorpusConGuionesNoEjecutaNada(t *testing.T) {
	s, _ := superficie(t, []*corpus.Paquete{paqueteHostil()})
	hostil := paqueteHostil()
	rutas := []string{"/alcance", "/controles", "/certificados", "/hoy", "/personas",
		"/estado", "/alcance?si=" + hostil.Preguntas[0].ID}
	for _, ruta := range rutas {
		w, cuerpo := pedir(t, s, ruta)
		if w.Code != http.StatusOK {
			t.Fatalf("%s dio %d con corpus hostil", ruta, w.Code)
		}
		if strings.Contains(cuerpo, elGuion) {
			i := strings.Index(cuerpo, elGuion)
			t.Fatalf("%s: el guion sale SIN ESCAPAR. Contexto:\n...%s...",
				ruta, cuerpo[max(0, i-160):min(i+160, len(cuerpo))])
		}
		if strings.Contains(cuerpo, `onload="alert(2)`) {
			t.Fatalf("%s: se ha colado un manejador desde el corpus", ruta)
		}
	}
	// Y el escapado no es "borrarlo todo": el texto sigue ahi, escapado, que
	// es lo que hace que el operador vea lo que dice su paquete.
	_, alcance := pedir(t, s, "/alcance")
	exige(t, alcance, html.EscapeString(elGuion+"[texto_pregunta]"))
}

// El mismo guion, pero puesto por el CATALOGO. El catalogo es un fichero de
// datos, no marcado: si un dia se decidiera devolver template.HTML desde
// Traducir, esto se pone rojo.
func TestElTextoDelCatalogoTampocoInyectaMarcado(t *testing.T) {
	cat := nuevoCatalogo()
	cat.textos = map[string]string{"alcance.intro": elGuion}
	s, err := Nuevo(Opciones{Paquetes: corpusDemo(), Catalogo: cat})
	if err != nil {
		t.Fatal(err)
	}
	_, cuerpo := pedir(t, s, "/alcance")
	if strings.Contains(cuerpo, elGuion) {
		t.Fatal("el texto del catalogo se ha pintado como marcado. Un catalogo es un " +
			"fichero de datos y un fichero de datos no inyecta HTML")
	}
	exige(t, cuerpo, html.EscapeString(elGuion))
}

// Caracteres de control y UTF-8 invalido en el corpus: ni panico, ni salida
// rota. El paquete llega de un fichero que escribe un tercero.
func TestUnCorpusConBasuraBinariaNoRompeLaPagina(t *testing.T) {
	p := paqueteAlfa()
	p.Preguntas[0].Texto = "control\x00\x08\x1b[31m y utf8 roto \xff\xfe fin"
	p.Obligaciones[0].Cita = "\x00\x01\x02"
	s, _ := superficie(t, []*corpus.Paquete{p})
	for _, ruta := range []string{"/alcance", "/controles", "/certificados"} {
		w, cuerpo := pedir(t, s, ruta)
		if w.Code != http.StatusOK {
			t.Fatalf("%s dio %d", ruta, w.Code)
		}
		if !utf8.ValidString(cuerpo) {
			t.Errorf("%s devuelve UTF-8 invalido: el navegador puede resincronizar por "+
				"donde quiera, y ahi es donde se cuelan las inyecciones", ruta)
		}
	}
}

// Una etiqueta de 100 KB no tumba la pagina ni la deja sin cerrar.
func TestUnaEtiquetaEnormeNoTumbaLaPagina(t *testing.T) {
	p := paqueteAlfa()
	p.Entidades[0].Atributos[0].Ayuda = strings.Repeat("A", 100*1024)
	p.Preguntas[0].Texto = strings.Repeat("B", 100*1024)
	s, _ := superficie(t, []*corpus.Paquete{p})
	w, cuerpo := pedir(t, s, "/alcance")
	if w.Code != http.StatusOK {
		t.Fatalf("dio %d", w.Code)
	}
	if !strings.HasSuffix(strings.TrimSpace(cuerpo), "</html>") {
		t.Error("la pagina no termina: se ha cortado a mitad")
	}
}

// Un corpus de 5000 obligaciones se pagina en vez de convertirse en una pagina
// de varios megabytes, y sigue respondiendo en un tiempo razonable.
func TestUnCorpusEnormeSePaginaYNoSeVaDeTamano(t *testing.T) {
	s, _ := superficie(t, []*corpus.Paquete{paqueteGrande(5000)})

	empieza := time.Now()
	w, cuerpo := pedir(t, s, "/controles")
	tardo := time.Since(empieza)
	if w.Code != http.StatusOK {
		t.Fatalf("dio %d", w.Code)
	}
	if len(cuerpo) > 2<<20 {
		t.Errorf("la pagina pesa %d bytes con 5000 obligaciones. La paginacion no esta "+
			"acotando nada", len(cuerpo))
	}
	if n := strings.Count(cuerpo, `<th scope="row">`); n > PorPaginaPorDefecto {
		t.Errorf("se han pintado %d filas y el limite es %d", n, PorPaginaPorDefecto)
	}
	if tardo > 5*time.Second {
		t.Errorf("una peticion con 5000 obligaciones ha tardado %v", tardo)
	}
	// Y el panel de Alcance no se convierte en un listado de 5000 lineas.
	_, alcance := pedir(t, s, "/alcance")
	if n := strings.Count(alcance, `class="veredicto`); n > MaxAplican+MaxProximas {
		t.Errorf("el panel de Alcance pinta %d veredictos y el limite es %d",
			n, MaxAplican+MaxProximas)
	}
}

// Un paquete con erratas que no dan error en ningun otro sitio: la interfaz las
// ensena en vez de tragarselas. Una obligacion colgada de una pregunta que no
// existe se quedaria pendiente para siempre sin que nadie supiera por que.
func TestLasErratasDelCorpusSeVenEnLaInterfaz(t *testing.T) {
	s, _ := superficie(t, []*corpus.Paquete{paqueteRoto()})
	_, controles := pedir(t, s, "/controles")
	exige(t, controles, "roto.o.colgada", rotulo("es", "derivacion.pregunta_desconocida"))
	_, certificados := pedir(t, s, "/certificados")
	exige(t, certificados, "roto.pl.sola", rotulo("es", "derivacion.entregable_huerfano"))
}

// ---------------------------------------------------------------------------
// Peticion adversaria
// ---------------------------------------------------------------------------

// Una respuesta a una pregunta que no existe se descarta entera: ni se cuenta,
// ni se ensena, ni se copia a los enlaces de la pagina.
func TestUnaRespuestaInventadaNoEntraEnLaPagina(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	veneno := "no.existe." + elGuion
	w, cuerpo := pedir(t, s, "/alcance?si="+veneno+"&no=tampoco.existe")
	if w.Code != http.StatusOK {
		t.Fatalf("dio %d", w.Code)
	}
	prohibe(t, cuerpo, elGuion, "tampoco.existe", "no.existe.")
	// Y el progreso no cuenta lo que no existe.
	exige(t, cuerpo, rotulo("es", "alcance.progreso")+" 0 3")
}

// Respuestas contradictorias: ni si ni no. La obligacion se queda pendiente y
// la pantalla avisa. Elegir una en silencio seria afirmar alcance sobre una
// entrada que no dice nada.
func TestUnaRespuestaContradictoriaNoAfirmaNada(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	_, cuerpo := pedir(t, s, "/alcance?si=alfa.q.categoria&no=alfa.q.categoria")
	exige(t, cuerpo, rotulo("es", "alcance.pregunta.contradictoria"))
	if strings.Contains(seccionAplican(cuerpo), "alfa.o.auditoria") {
		t.Error("una respuesta contradictoria ha hecho aplicar una obligacion")
	}
	_, controles := pedir(t, s, "/controles?si=alfa.q.categoria&no=alfa.q.categoria")
	fila := conFilaDesde(controles, strings.Index(controles, "alfa.o.auditoria"))
	exige(t, fila, rotulo("es", "estado.pendiente"),
		rotulo("es", "derivacion.respuesta_contradictoria"))
	prohibe(t, fila, rotulo("es", "estado.aplica"), rotulo("es", "estado.no_aplica"))
	// Y el aviso no se pierde al navegar: la contradiccion sigue en la
	// direccion que compone la propia pagina, con sus dos respuestas.
	exige(t, controles, "no=alfa.q.categoria&amp;si=alfa.q.categoria")
}

// Una consulta desmesurada se corta antes de parsearla, y el error no repite lo
// que mando el cliente.
func TestUnaConsultaDesmesuradaSeCortaYNoSeRefleja(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	veneno := strings.Repeat("si=alfa.q.categoria&", 2000) + "x=" + elGuion
	w, cuerpo := pedir(t, s, "/alcance?"+veneno)
	if w.Code != http.StatusRequestURITooLong {
		t.Fatalf("dio %d y esperaba 414", w.Code)
	}
	exige(t, cuerpo, rotulo("es", "error.consulta_larga"))
	prohibe(t, cuerpo, elGuion, html.EscapeString(elGuion))
}

// Los parametros de paginacion y filtro con basura dentro: se saneen o se
// ignoren, pero nunca se reflejan ni revientan.
func TestLosParametrosConBasuraNiRevientanNiSeReflejan(t *testing.T) {
	s, _ := superficie(t, []*corpus.Paquete{paqueteGrande(600)})
	for _, consulta := range []string{
		"p=" + elGuion, "p=-5", "p=0", "p=99999999999999999999", "p=999",
		"f=" + elGuion, "f=aplica'--", "f=", "p=2&f=nada",
		"si=&no=", "si=%00%01", "si=" + strings.Repeat("%2e", 200),
	} {
		w, cuerpo := pedir(t, s, "/controles?"+consulta)
		if w.Code != http.StatusOK {
			t.Errorf("/controles?%s dio %d", consulta, w.Code)
			continue
		}
		prohibe(t, cuerpo, elGuion)
		if strings.Contains(cuerpo, "aplica'--") {
			t.Errorf("/controles?%s refleja el valor del filtro", consulta)
		}
		if !strings.HasSuffix(strings.TrimSpace(cuerpo), "</html>") {
			t.Errorf("/controles?%s devuelve una pagina cortada", consulta)
		}
	}
}

// Los estaticos se sirven por nombre exacto contra un mapa: no hay ruta que
// recorrer, asi que no hay travesia posible. Se comprueba igualmente.
func TestNoSePuedeSalirDelDirectorioDeEstaticos(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	for _, ruta := range []string{
		"/estatico/../pantallas.go", "/estatico/..%2f..%2fgo.mod",
		"/estatico/%2e%2e%2fgo.mod", "/estatico/plantillas/base.html",
		"/estatico/", "/estatico/no-existe.js",
	} {
		w, cuerpo := pedir(t, s, ruta)
		if w.Code == http.StatusOK && strings.Contains(cuerpo, "package pantallas") {
			t.Fatalf("%s ha servido codigo fuente", ruta)
		}
		if w.Code == http.StatusOK && strings.Contains(cuerpo, "module dutiq") {
			t.Fatalf("%s ha servido go.mod", ruta)
		}
	}
}

// La cabecera de idioma la escribe el cliente y acaba en <html lang>: se sanea.
func TestElIdiomaPedidoSeSanea(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	for _, cabecera := range []string{
		`es"><script>alert(1)</script>`, "es-ES,es;q=0.9", "xx-XX", "*",
		strings.Repeat("a", 200), "", "en",
	} {
		r := httptest.NewRequest(http.MethodGet, "/alcance", nil)
		r.Header.Set("Accept-Language", cabecera)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		cuerpo := w.Body.String()
		if w.Code != http.StatusOK {
			t.Fatalf("Accept-Language %q dio %d", cabecera, w.Code)
		}
		prohibe(t, cuerpo, elGuion)
		// El idioma declarado tiene que ser uno de los cargados: una pagina
		// que declara un idioma que no lleva dentro rompe a los lectores de
		// pantalla.
		if !strings.Contains(cuerpo, `<html lang="es">`) &&
			!strings.Contains(cuerpo, `<html lang="en">`) {
			t.Errorf("Accept-Language %q ha producido un lang que no es de los cargados",
				cabecera)
		}
	}
	// Y con en, el catalogo elegido es el de en.
	r := httptest.NewRequest(http.MethodGet, "/alcance", nil)
	r.Header.Set("Accept-Language", "en-GB")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	exige(t, w.Body.String(), rotulo("en", "pantalla.alcance.titulo"))
}

// idiomaPedido, a solas. El test de arriba pasa por Resolver, que ya cae al
// idioma por defecto, asi que por si solo no demuestra que la cabecera se
// sanee: se comprueba aqui la funcion, que es donde esta la comprobacion.
func TestIdiomaPedidoRechazaLoQueNoEsUnaEtiquetaDeIdioma(t *testing.T) {
	malas := []string{
		`es"><script>alert(1)</script>`, "*", strings.Repeat("a", 200),
		"es;q=0.9;evil<>", "es/../en", "es\x00", "", "  ", "es,en",
	}
	for _, cabecera := range malas {
		r := httptest.NewRequest(http.MethodGet, "/alcance", nil)
		r.Header.Set("Accept-Language", cabecera)
		if got := idiomaPedido(r); !etiquetaPlausible(got) {
			t.Errorf("Accept-Language %q ha dado el idioma %q, que no es una etiqueta "+
				"de idioma. Acaba en <html lang> y en la eleccion de catalogo", cabecera, got)
		}
	}
	buenas := map[string]string{"es": "es", "en-GB": "en-GB", "es-ES,es;q=0.9": "es-ES",
		" fr ": "fr"}
	for cabecera, quiero := range buenas {
		r := httptest.NewRequest(http.MethodGet, "/alcance", nil)
		r.Header.Set("Accept-Language", cabecera)
		if got := idiomaPedido(r); got != quiero {
			t.Errorf("Accept-Language %q dio %q y esperaba %q", cabecera, got, quiero)
		}
	}
}

// etiquetaPlausible: vacia (o sea, se descarto) o letras, digitos y guiones.
func etiquetaPlausible(s string) bool {
	if s == "" {
		return true
	}
	if len(s) > 35 {
		return false
	}
	for _, c := range s {
		ok := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-'
		if !ok {
			return false
		}
	}
	return true
}

// Y con un adaptador de plantillas que NO implementa Resolver, que es la
// interfaz opcional por la que la superficie averigua el idioma que se va a
// renderizar. Sin la comprobacion contra los idiomas del catalogo, la cabecera
// del cliente acababa tal cual en <html lang>.
func TestConOtroAdaptadorDePlantillasElIdiomaSigueSiendoUnoDeLosCargados(t *testing.T) {
	cat := nuevoCatalogo()
	s, err := Nuevo(Opciones{Paquetes: corpusDemo(), Catalogo: cat,
		Plantilla: plantillaSinResolver{}})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, "/alcance", nil)
	r.Header.Set("Accept-Language", "de-CH")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if got := w.Body.String(); got != "es" {
		t.Errorf("con de-CH se ha renderizado el idioma %q, y los cargados son %v",
			got, cat.Idiomas())
	}
}

// plantillaSinResolver escribe el idioma que le llega y nada mas: es lo minimo
// que satisface puertos.Plantilla, que es exactamente lo que otro adaptador
// puede ser.
type plantillaSinResolver struct{}

func (plantillaSinResolver) Render(w io.Writer, _ string, _ any, idioma string) error {
	_, err := io.WriteString(w, idioma)
	return err
}

// Servir y recargar el corpus a la vez. Con -race, esto es lo que separa "no
// he visto que falle" de "no puede fallar".
func TestServirYRecargarALaVezNoCorrompeNada(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	var g sync.WaitGroup
	for i := 0; i < 8; i++ {
		g.Add(1)
		go func(i int) {
			defer g.Done()
			for j := 0; j < 25; j++ {
				r := httptest.NewRequest(http.MethodGet, "/controles?si=alfa.q.categoria", nil)
				w := httptest.NewRecorder()
				s.ServeHTTP(w, r)
				if w.Code != http.StatusOK {
					t.Errorf("peticion concurrente dio %d", w.Code)
					return
				}
			}
		}(i)
	}
	g.Add(1)
	go func() {
		defer g.Done()
		for j := 0; j < 25; j++ {
			s.Recargar(corpusDemo())
			s.Recargar([]*corpus.Paquete{paqueteAlfa()})
		}
	}()
	g.Wait()
}

// ---------------------------------------------------------------------------
// Construccion
// ---------------------------------------------------------------------------

func TestLaSuperficieSeNiegaAConstruirseSinCatalogo(t *testing.T) {
	if _, err := Nuevo(Opciones{Paquetes: corpusDemo()}); err == nil {
		t.Fatal("se ha construido sin catalogo: la pagina saldria con las claves crudas")
	}
	for _, base := range []string{"ui", "/ui/", "/"} {
		if _, err := Nuevo(Opciones{Catalogo: nuevoCatalogo(), Base: base}); err == nil {
			t.Errorf("se ha aceptado Base=%q", base)
		}
	}
	s, err := Nuevo(Opciones{Catalogo: nuevoCatalogo(), Paquetes: corpusDemo(), Base: "/ui"})
	if err != nil {
		t.Fatalf("Base=/ui tiene que valer: %v", err)
	}
	_, cuerpo := pedir(t, s, "/alcance")
	exige(t, cuerpo, `href="/ui/controles`, `href="/ui/estatico/dutiq.css"`)
}
