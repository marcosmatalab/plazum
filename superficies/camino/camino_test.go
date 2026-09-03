package camino

import (
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/catalogo"
)

func cat(t *testing.T) *catalogo.Catalogo {
	t.Helper()
	c, err := catalogo.Nuevo()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func superficie(t *testing.T, pasos []Paso) *Superficie {
	t.Helper()
	s, err := Nuevo(Opciones{Pasos: pasos, Catalogo: cat(t),
		Base: BasePorDefecto, Estatico: "/estatico"})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func pedir(t *testing.T, s *Superficie, ruta string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, ruta, nil)
	req.Header.Set("Accept-Language", "es")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

// LAS DOS FORMAS DE LA NADA, Y LAS DOS SE RECHAZAN (invariante 8).
//
// nil sale de olvidarse el campo; vacio-presente, de construir la lista y
// filtrarla entera. En Go la peligrosa es siempre la nil, porque es la que sale
// por descuido, y aqui las dos pintan la misma pantalla: una que dice que
// plazum no tiene camino, que es la mentira mas cara que esta pieza puede
// contar.
func TestUnCaminoVacioNoSeConstruye(t *testing.T) {
	for _, c := range []struct {
		nombre string
		pasos  []Paso
	}{
		{"nil", nil},
		{"vacio presente", []Paso{}},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			_, err := Nuevo(Opciones{Pasos: c.pasos, Catalogo: cat(t)})
			if !errors.Is(err, ErrSinPasos) {
				t.Fatalf("con el camino %s el error es %v y esperaba ErrSinPasos", c.nombre, err)
			}
		})
	}
	// CONTROL POSITIVO: con el camino canonico SI se construye. Sin esta rama,
	// un constructor que fallara siempre pasaria las dos de arriba.
	if _, err := Nuevo(Opciones{Pasos: Canonico(), Catalogo: cat(t),
		Base: BasePorDefecto}); err != nil {
		t.Fatalf("el camino canonico no se construye: %v", err)
	}
}

// UN PASO SIN SALIDA ES UN CALLEJON Y NO SALE DE AQUI (puerta D11-b).
//
// La forma de romper esto es la barata: anadir un paso al camino sin decir que
// se hace en el ni donde. El resultado seria una pantalla con un rotulo y nada
// mas, o sea alguien que llega y no tiene nada que hacer.
func TestUnPasoSinSalidaNoSeConstruye(t *testing.T) {
	casos := []struct {
		nombre    string
		paso      Paso
		centinela error
	}{
		{"ni pantalla ni orden",
			Paso{ID: "x", Titulo: "camino.titulo", Verbo: "camino.que_es"}, ErrPasoSinSalida},
		{"sin verbo",
			Paso{ID: "x", Titulo: "camino.titulo", Ruta: "/x"}, ErrPasoIncompleto},
		{"sin rotulo",
			Paso{ID: "x", Verbo: "camino.que_es", Ruta: "/x"}, ErrPasoIncompleto},
		{"sin identificador",
			Paso{Titulo: "camino.titulo", Verbo: "camino.que_es", Ruta: "/x"}, ErrPasoIncompleto},
		{"ruta a otro anfitrion",
			Paso{ID: "x", Titulo: "camino.titulo", Verbo: "camino.que_es",
				Ruta: "//evil.example/x"}, ErrRutaInvalida},
		{"ruta relativa",
			Paso{ID: "x", Titulo: "camino.titulo", Verbo: "camino.que_es",
				Ruta: "x"}, ErrRutaInvalida},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			err := Validar([]Paso{c.paso})
			if !errors.Is(err, c.centinela) {
				t.Fatalf("el error es %v y esperaba %v", err, c.centinela)
			}
		})
	}
	// Y un identificador repetido rompe la tira: no se puede decir en que paso
	// estas si hay dos que se llaman igual.
	dos := Paso{ID: "x", Titulo: "camino.titulo", Verbo: "camino.que_es", Comando: "plazum"}
	if err := Validar([]Paso{dos, dos}); !errors.Is(err, ErrPasoIncompleto) {
		t.Errorf("dos pasos con el mismo identificador y el error es %v", err)
	}
	// CONTROL POSITIVO: un paso entero pasa, por cada una de las dos salidas.
	for _, bueno := range []Paso{
		{ID: "x", Titulo: "camino.titulo", Verbo: "camino.que_es", Ruta: "/x"},
		{ID: "y", Titulo: "camino.titulo", Verbo: "camino.que_es", Comando: "plazum demo"},
	} {
		if err := Validar([]Paso{bueno}); err != nil {
			t.Errorf("el paso %q no pasa y tenia que pasar: %v", bueno.ID, err)
		}
	}
}

// LA DEUDA, CONTADA Y NO LEIDA.
//
// Dos pasos del camino todavia no se recorren sin salir de `plazum serve`: el
// calendario y el escalado. Existen, tienen su orden y salen en la pantalla con
// el comando que los hace hoy, pero no son pantalla.
//
// Este test es un TRINQUETE: el dia que uno de los dos gane su pantalla se pone
// rojo y obliga a bajar el numero en el mismo commit. Sin el, la deuda se
// quedaria en un comentario, y un aviso en un comentario no viaja.
func TestLosPasosQueTodaviaNoSonPantallaSonLosDeclarados(t *testing.T) {
	// CERO, DESDE EL 03-09-2026: los dos que quedaban (calendario y escalado)
	// ganaron su pantalla. El trinquete NO se borra al vaciarse, y sigue
	// apretando en la direccion contraria: el dia que alguien meta un paso
	// nuevo sin pantalla, este test se pone rojo y le obliga a decir por que
	// entra en el camino sin ella. Un trinquete que se borra cuando su cuenta
	// llega a cero es un trinquete que hay que volver a escribir.
	quiero := []string{}
	tengo := SinPantalla()
	if strings.Join(tengo, ",") != strings.Join(quiero, ",") {
		t.Fatalf("los pasos sin pantalla son %v y estaban declarados %v.\n"+
			"  Si uno ha ganado su pantalla: quitarlo de aqui y celebrarlo.\n"+
			"  Si ha aparecido uno nuevo: decir por que entra en el camino sin pantalla.",
			tengo, quiero)
	}
	// Y cada uno de ellos tiene su orden. Un paso sin pantalla y sin orden seria
	// el callejon que Validar prohibe, comprobado aqui sobre el camino de verdad.
	// Hoy el bucle no recorre nada, y por eso la propiedad se comprueba tambien
	// sobre un camino SINTETICO en TestUnPasoSinPantallaSigueExigiendoSuOrden:
	// una rama que ninguna entrada alcanza es una rama que no existe (M47).
	for _, id := range tengo {
		var visto bool
		for _, p := range Canonico() {
			if p.ID == id {
				visto = true
				if strings.TrimSpace(p.Comando) == "" {
					t.Errorf("el paso %q no es pantalla y no dice con que orden se hace hoy", id)
				}
			}
		}
		if !visto {
			t.Errorf("SinPantalla() nombra %q y el camino canonico no lo tiene", id)
		}
	}
}

// EL PASO SIN PANTALLA SALE, Y SALE SIN ENLACE.
//
// Las dos mitades importan y son contrarias. No se esconde, porque un camino de
// seis pasos del que se ven cuatro parece completo y no lo esta. Y no se enlaza,
// porque un enlace que no lleva a ningun sitio es peor que no tenerlo: quien lo
// pulse se lleva un 404 y deja de creerse el resto de la pagina.
func TestElPasoSinPantallaSaleConSuOrdenYSinEnlace(t *testing.T) {
	// SOBRE UN CAMINO SINTETICO desde el 03-09-2026, y no sobre el canonico.
	//
	// El canonico ya no tiene ningun paso sin pantalla, asi que este test se
	// quedo sin rama que recorrer y lo dijo el mismo, con su propio Fatal. La
	// respuesta NO es borrarlo: la plantilla sigue teniendo esa rama y Validar
	// sigue aceptando un paso sin pantalla que traiga su orden, o sea que la
	// capacidad esta viva. Lo que le faltaba era una entrada que la alcanzara.
	//
	// Se le da un camino de dos pasos, uno con pantalla y otro sin ella, que es
	// el minimo que ejercita las dos mitades contrarias que este test compara.
	pasos := []Paso{
		{ID: "alcance", Titulo: "camino.paso.alcance", Verbo: "camino.verbo.alcance",
			Ruta: "/alcance"},
		{ID: "sintetico", Titulo: "camino.paso.calendario", Verbo: "camino.verbo.calendario",
			Comando: "plazum algo --con-sus-banderas"},
	}
	s := superficie(t, pasos)
	codigo, cuerpo := pedir(t, s, BasePorDefecto+"/")
	if codigo != http.StatusOK {
		t.Fatalf("la pantalla del camino ha respondido %d", codigo)
	}
	sinPantalla := 0
	for _, p := range pasos {
		if p.EsPantalla() {
			continue
		}
		sinPantalla++
		if !strings.Contains(cuerpo, p.Comando) {
			t.Errorf("el paso %q no ensena su orden %q", p.ID, p.Comando)
		}
		if strings.Contains(cuerpo, `href="`+p.Ruta+`"`) && p.Ruta != "" {
			t.Errorf("el paso %q no es pantalla y la pagina lo enlaza igual", p.ID)
		}
	}
	if sinPantalla == 0 {
		t.Fatal("ningun paso del camino esta sin pantalla, asi que este test no ha recorrido " +
			"la rama que dice mirar")
	}
	// CONTROL POSITIVO DE LA OTRA RAMA: los que si son pantalla llevan enlace.
	// Sin esto, una pagina que no enlazara nada pasaria la comprobacion de
	// arriba entera.
	for _, p := range pasos {
		if p.EsPantalla() && !strings.Contains(cuerpo, `href="`+p.Ruta+`"`) {
			t.Errorf("el paso %q es pantalla y la pagina no lo enlaza", p.ID)
		}
	}
}

// ESTA PANTALLA NO DICE QUE NADA ESTE HECHO, Y LO DICE.
//
// plazum no guarda las respuestas de la entrevista y sin expediente no consta
// que ningun paso se haya dado. Una marca de "hecho" aqui, o una barra de
// progreso, seria inventarse el dato mas caro del producto; y callarse que no
// se sabe deja al lector suponiendo lo que quiera. La frase va CON la lista, no
// en un pie, que es el patron de la casa.
func TestElCaminoNoDiceQueNingunPasoEsteHecho(t *testing.T) {
	s := superficie(t, Canonico())
	_, cuerpo := pedir(t, s, BasePorDefecto+"/")
	frase := cat(t).Traducir("es", "camino.sin_progreso")
	if frase == "camino.sin_progreso" {
		t.Fatal("el catalogo no tiene la frase, asi que este test no mide nada")
	}
	if !strings.Contains(cuerpo, frase) {
		t.Errorf("la pantalla del camino no dice que aqui no consta que nada este hecho.\n" +
			"  Sin esa frase, seis pasos en fila se leen como una lista de tareas con su\n" +
			"  estado, y plazum no sabe el estado de ninguna")
	}
	// Y la frase no acusa: dice que no consta, no que no se haya hecho.
	if strings.Contains(strings.ToLower(frase), "no has hecho") {
		t.Errorf("la frase acusa en vez de decir que falta el dato: %q", frase)
	}
}

// LA TIRA NO MARCA NINGUN PASO CUANDO NO ESTAS EN EL CAMINO.
//
// Es el valor cero honesto: una pantalla que no es un paso (Personas, Estado)
// pide la tira con un identificador que no existe. Marcar el primero "por si
// acaso" le diria al operador que esta en un sitio en el que no esta.
func TestLaTiraNoMarcaNadaFueraDelCamino(t *testing.T) {
	for _, actual := range []string{"", "personas", "no-existe"} {
		for _, e := range Tira(Canonico(), "", actual) {
			if e.Actual {
				t.Errorf("con actual=%q la tira marca el paso %q", actual, e.ID)
			}
		}
	}
	// CONTROL POSITIVO: con un paso de verdad, marca ese y solo ese.
	marcados := 0
	for _, e := range Tira(Canonico(), "", "acta") {
		if e.Actual {
			marcados++
			if e.ID != "acta" {
				t.Errorf("marca %q y esperaba acta", e.ID)
			}
		}
	}
	if marcados != 1 {
		t.Errorf("marca %d pasos y esperaba 1", marcados)
	}
	// Y el paso sin pantalla no lleva direccion, ni siquiera con prefijo.
	for _, e := range Tira(Canonico(), "/ui", "") {
		ruta, esPantalla := RutaDe(e.ID)
		if esPantalla && e.URL != "/ui"+ruta {
			t.Errorf("el paso %q sale con URL %q y esperaba %q", e.ID, e.URL, "/ui"+ruta)
		}
		if !esPantalla && e.URL != "" {
			t.Errorf("el paso %q no es pantalla y la tira le ha puesto la URL %q", e.ID, e.URL)
		}
	}
}

// EL CAMINO NO SE COME LAS RESPUESTAS DE LA ENTREVISTA.
//
// POR QUE EXISTE, y es el hallazgo de intentar tumbar una propiedad que este
// paquete daba por buena. Las respuestas de la entrevista NO SE GUARDAN: viajan
// en la direccion de la pagina, y la pantalla de Alcance lo dice con esas
// palabras. Con los enlaces pelados, el recorrido /alcance?si=... -> camino ->
// Alcance devolvia al operador una entrevista EN BLANCO. O sea que el sitio que
// existe para no perder a nadie era el unico sitio del producto capaz de borrar
// el trabajo de quien lo usaba.
//
// Y no a todos los pasos: al acta y a la revision de accesos no les dice nada el
// alcance, asi que arrastrarles la consulta seria llevar un dato hasta una
// pantalla que no lo lee.
func TestElCaminoLlevaLasRespuestasALosPasosQueLasUsan(t *testing.T) {
	s := superficie(t, Canonico())
	codigo, cuerpo := pedir(t, s, BasePorDefecto+"/?si=alfa.q.uno&no=beta.q.dos")
	if codigo != http.StatusOK {
		t.Fatalf("codigo %d", codigo)
	}
	conAlcance, sinAlcance := 0, 0
	for _, p := range Canonico() {
		if !p.EsPantalla() {
			continue
		}
		conConsulta := `href="` + p.Ruta + `?no=beta.q.dos&amp;si=alfa.q.uno"`
		pelado := `href="` + p.Ruta + `"`
		if p.LlevaAlcance {
			conAlcance++
			if !strings.Contains(cuerpo, conConsulta) {
				t.Errorf("el paso %q usa el alcance y su enlace va pelado: pulsarlo borra las "+
					"respuestas. Esperaba %s", p.ID, conConsulta)
			}
		} else {
			sinAlcance++
			if !strings.Contains(cuerpo, pelado) {
				t.Errorf("el paso %q no usa el alcance y su enlace lo lleva igual: es "+
					"arrastrar un dato hasta una pantalla que no lo lee", p.ID)
			}
		}
	}
	// LAS DOS RAMAS TIENEN QUE HABERSE RECORRIDO. Una comprobacion que solo pasa
	// por una de las dos no dice nada de la otra.
	if conAlcance == 0 || sinAlcance == 0 {
		t.Fatalf("pasos con alcance: %d, sin alcance: %d. Con un cero, este test solo mira "+
			"una de las dos ramas", conAlcance, sinAlcance)
	}
	// Y SIN CONSULTA, LOS ENLACES VAN PELADOS: no se inventa un interrogante.
	_, limpio := pedir(t, s, BasePorDefecto+"/")
	if strings.Contains(limpio, "?si=") || strings.Contains(limpio, `="/alcance?"`) {
		t.Errorf("sin consulta la pagina se inventa una:\n%s", limpio)
	}
}

// UNA CONSULTA DESMESURADA NI SE RECORTA NI SE CUELA.
//
// Recortarla dejaria media entrevista con cara de entrevista entera, que es peor
// que no llevarla; colarla entera convierte una peticion en una pagina enorme.
// Se contesta 414, que es lo que hace la superficie hermana con lo mismo.
func TestUnaConsultaDesmesuradaNoEntraEnLosEnlaces(t *testing.T) {
	s := superficie(t, Canonico())
	larga := "si=" + strings.Repeat("x", MaxConsulta+1)
	codigo, cuerpo := pedir(t, s, BasePorDefecto+"/?"+larga)
	if codigo != http.StatusRequestURITooLong {
		t.Fatalf("codigo %d y esperaba 414", codigo)
	}
	if strings.Contains(cuerpo, strings.Repeat("x", 100)) {
		t.Error("la respuesta refleja lo que mando el cliente")
	}
	// CONTROL POSITIVO: justo por debajo del limite SI pasa. Sin esto, un
	// limite de cero dejaria el test de arriba en verde para siempre.
	corta := "si=" + strings.Repeat("x", 10)
	if codigo, _ := pedir(t, s, BasePorDefecto+"/?"+corta); codigo != http.StatusOK {
		t.Errorf("una consulta corta ha dado %d: el limite esta rechazando todo", codigo)
	}
}

// NINGUNA RUTA DE ESTA SUPERFICIE MUTA. La lista sale del enrutador y no de una
// lista escrita al lado del test, que es la que se desincroniza el dia que
// alguien anade un handler.
func TestNingunaRutaDeEstaSuperficieMuta(t *testing.T) {
	s := superficie(t, Canonico())
	probadas := 0
	for _, p := range s.Patrones() {
		metodo, ruta, ok := strings.Cut(p, " ")
		if !ok {
			t.Fatalf("el patron %q no lleva metodo, asi que atiende cualquiera", p)
		}
		if metodo != http.MethodGet {
			t.Errorf("el patron %q no es GET. Esta pantalla no cambia nada y no puede tener "+
				"una ruta que lo parezca", p)
		}
		ruta = strings.ReplaceAll(ruta, "{$}", "")
		for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodDelete,
			http.MethodPatch} {
			req := httptest.NewRequest(m, ruta, strings.NewReader(""))
			rec := httptest.NewRecorder()
			s.ServeHTTP(rec, req)
			probadas++
			if rec.Code >= 200 && rec.Code < 300 {
				t.Errorf("%s %s ha respondido %d: una mutacion atendida", m, ruta, rec.Code)
			}
		}
	}
	if probadas < 4 {
		t.Fatalf("se han probado %d combinaciones: el enrutador no esta devolviendo lo que "+
			"hay", probadas)
	}
}

// EL CONTRATO DE CLAVES CASA CON LO QUE PIDE LA PLANTILLA, en las dos
// direcciones: no puede quedarse corto (saldria una clave cruda en la pantalla
// de un cliente) ni sobrarle nada (peso muerto que hay que traducir a cada
// idioma nuevo).
func TestElContratoDeClavesCasaConLoQuePideLaPlantilla(t *testing.T) {
	// SE LEEN LAS DOS: la plantilla de esta pantalla y la del ARMAZON
	// COMPARTIDO, que esta pantalla renderiza igual. Leer solo la primera
	// dejaria fuera del contrato los rotulos de la barra lateral, y el sintoma
	// seria un "estas aqui" en crudo en la pantalla de un cliente.
	var b []byte
	for _, f := range []struct {
		sistema fs.FS
		nombre  string
	}{
		{plantillasFS, "plantillas/camino.html"},
		{armazonFS, "armazon/armazon.html"},
	} {
		trozo, err := fs.ReadFile(f.sistema, f.nombre)
		if err != nil {
			t.Fatal(err)
		}
		b = append(b, trozo...)
	}
	pide := map[string]bool{}
	for _, m := range regexp.MustCompile(`\{\{-?\s*t\s+"([^"]+)"`).
		FindAllStringSubmatch(string(b), -1) {
		pide[m[1]] = true
	}
	if len(pide) < 5 {
		t.Fatalf("la plantilla pide %d claves literales: o se ha vaciado, o esta expresion "+
			"ha dejado de reconocerlas y el test mide el vacio", len(pide))
	}
	// Las que pone el codigo Go y las de los pasos, que la plantilla nombra con
	// `t .Titulo` y por tanto no aparecen como literal.
	pide["camino.error.render"] = true
	pide["camino.error.consulta_larga"] = true
	// Y ClaveTitulo, que la pide DOS veces el codigo: aqui como titulo de la
	// pagina, y en las demas superficies como rotulo del enlace de vuelta.
	pide[ClaveTitulo] = true
	for _, p := range canonico {
		pide[p.Titulo] = true
		pide[p.Verbo] = true
	}
	declara := map[string]bool{}
	for _, k := range ClavesDeCatalogo() {
		declara[k] = true
	}
	for k := range pide {
		if !declara[k] {
			t.Errorf("se pide la clave %q y ClavesDeCatalogo() no la declara: saldria en crudo", k)
		}
	}
	for k := range declara {
		if !pide[k] {
			t.Errorf("ClavesDeCatalogo() declara %q y no la pide nadie", k)
		}
	}
}

// LA PANTALLA NO LLEVA NADA QUE UNA CSP ESTRICTA BLOQUEE. Quien monte el
// servidor tiene que poder poner "script-src 'self'; style-src 'self'" sin
// negociar con esta superficie.
func TestLaPantallaNoLlevaNadaQueUnaCSPEstrictaBloquee(t *testing.T) {
	s := superficie(t, Canonico())
	_, cuerpo := pedir(t, s, BasePorDefecto+"/")
	for _, prohibido := range []string{"<script", " style=", " onclick=", " onload=", "javascript:"} {
		if strings.Contains(cuerpo, prohibido) {
			t.Errorf("la pagina lleva %q, que una CSP estricta bloquea", prohibido)
		}
	}
	if !strings.Contains(cuerpo, "plazum.css") {
		t.Error("la pagina no enlaza la hoja de estilo, asi que sale sin formato")
	}
}

// SIN CATALOGO NO SE CONSTRUYE: la pantalla saldria con las claves crudas, y
// esta es la primera que alguien abre.
func TestSinCatalogoLaSuperficieNoSeConstruye(t *testing.T) {
	if _, err := Nuevo(Opciones{Pasos: Canonico()}); !errors.Is(err, ErrSinCatalogo) {
		t.Fatalf("el error es %v y esperaba ErrSinCatalogo", err)
	}
}

// LA PROPIEDAD DEL PASO SIN PANTALLA SIGUE VIVA AUNQUE EL CAMINO YA NO TENGA
// NINGUNO, y por eso se comprueba con un camino SINTETICO.
//
// El 03-09-2026 los dos ultimos pasos sin pantalla ganaron la suya. Validar
// SIGUE aceptando un paso sin pantalla siempre que traiga su orden, y la
// plantilla del armazon sigue teniendo su rama. O sea que la capacidad existe y
// dejo de tener quien la recorriera: exactamente una rama que no existe.
//
// Se conserva la capacidad Y se le da entrada sintetica, en vez de borrarla.
// Borrarla dejaria Validar aceptando algo que la plantilla ya no sabria pintar,
// que es peor que las dos cosas.
func TestUnPasoSinPantallaSigueExigiendoSuOrden(t *testing.T) {
	conOrden := []Paso{
		{ID: "alcance", Titulo: "camino.paso.alcance", Verbo: "camino.verbo.alcance", Ruta: "/alcance"},
		{ID: "sintetico", Titulo: "camino.paso.acta", Verbo: "camino.verbo.acta",
			Comando: "plazum algo --con-sus-banderas"},
	}
	if err := Validar(conOrden); err != nil {
		t.Errorf("un paso sin pantalla PERO con su orden tiene que valer, y no vale: %v", err)
	}
	sinOrden := []Paso{
		{ID: "alcance", Titulo: "camino.paso.alcance", Verbo: "camino.verbo.alcance", Ruta: "/alcance"},
		{ID: "sintetico", Titulo: "camino.paso.acta", Verbo: "camino.verbo.acta"},
	}
	if err := Validar(sinOrden); err == nil {
		t.Error("un paso sin pantalla y sin orden es un callejon y se ha aceptado: se llega " +
			"y no hay nada que hacer")
	}
}
