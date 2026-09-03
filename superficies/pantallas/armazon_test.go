package pantallas

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/superficies/camino"
)

// LA BARRA LATERAL: el camino guiado en todas las pantallas.
//
// Que problema resuelve, y no es de estetica. Hasta aqui, las seis pantallas
// tenian un menu de seis entradas y nada mas: quien las abria veia una lista de
// sitios y ninguna respuesta a "¿y ahora que?". El orden en que se recorre
// plazum estaba escrito en superficies/camino y solo se veia entrando en la
// pantalla del camino, o sea que solo lo veia quien ya sabia que existia.
//
// Lo que estas puertas sostienen es que la tira SALE DEL CAMINO y no de una
// lista escrita al lado. Es la regla de la casa aplicada a una barra de
// navegacion: dos copias del mismo orden se separan el dia que una cambie, y el
// sintoma de esa separacion seria una barra que enlaza a un paso que ya no
// existe o que se calla uno nuevo.

// conCamino monta la superficie como la monta el producto: con el camino
// canonico en la barra lateral y con la vuelta a su pantalla.
func conCamino() func(*Opciones) {
	return func(o *Opciones) {
		o.Pasos = camino.Canonico()
		o.CaminoRuta = camino.BasePorDefecto + "/"
		o.CaminoClave = camino.ClaveTitulo
	}
}

// hrefs saca los destinos de los enlaces de la tira, en orden de aparicion.
var reTira = regexp.MustCompile(`(?s)<nav class="tira-camino".*?</nav>`)

func tiraDe(t *testing.T, cuerpo string) string {
	t.Helper()
	m := reTira.FindString(cuerpo)
	if m == "" {
		t.Fatal("la pagina no trae la tira del camino en la barra lateral")
	}
	return m
}

// La tira es EXACTAMENTE el camino canonico, en su orden y con sus numeros.
//
// Se compara contra camino.Canonico(), que es la fuente, y no contra una lista
// escrita aqui: un test que se mide contra su propia copia no comprueba nada.
func TestLaBarraLateralPintaElCaminoCanonicoEnteroYEnSuOrden(t *testing.T) {
	s, _ := superficie(t, corpusDemo(), conCamino())
	_, cuerpo := pedir(t, s, "/alcance")
	tira := tiraDe(t, cuerpo)

	pasos := camino.Canonico()
	if len(pasos) == 0 {
		t.Fatal("camino.Canonico() ha llegado vacio: este test estaria recorriendo la nada")
	}
	antes := -1
	for i, p := range pasos {
		rot := rotulo("es", p.Titulo)
		pos := strings.Index(tira, rot)
		if pos < 0 {
			t.Errorf("el paso %q (%s) no sale en la barra lateral. Un camino de %d pasos "+
				"del que se ven menos parece completo y no lo esta", p.ID, p.Titulo, len(pasos))
			continue
		}
		if pos < antes {
			t.Errorf("el paso %q sale ANTES que el anterior. El orden del camino no es "+
				"decorativo: cada paso consume lo que produce el de antes", p.ID)
		}
		antes = pos
		// Y su numero, que es lo que convierte una lista en un camino.
		numero := `<span class="numero">` + itoa(i+1) + `</span>`
		if !strings.Contains(tira, numero) {
			t.Errorf("el paso %d no lleva su numero en la barra lateral", i+1)
		}
	}
}

// Y NO HAY NINGUNA LISTA DE PASOS ESCRITA EN LA PLANTILLA.
//
// Es la mitad que de verdad importa: la puerta de arriba seguiria verde si
// alguien escribiera los seis rotulos a mano en base.html, y entonces el dia
// que el camino gane un septimo paso la barra ensenaria seis para siempre. Aqui
// se mira el fichero, no la pagina.
func TestLaPlantillaNoLlevaNingunPasoEscritoAMano(t *testing.T) {
	var texto string
	for _, f := range plantillasEnDisco(t) {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		texto += string(b)
	}
	if !strings.Contains(texto, "tira-camino") {
		t.Fatal("las plantillas no traen la tira del camino: este detector no tiene nada " +
			"que vigilar")
	}
	pasos := camino.Canonico()
	if len(pasos) == 0 {
		t.Fatal("camino.Canonico() ha llegado vacio")
	}
	// Se miran el ROTULO, el VERBO y la RUTA, que son las tres cosas que
	// podrian copiarse aqui y quedarse viejas. El identificador del paso NO se
	// mira, y lo dice el primer rojo de esta puerta: `p.ID` vale "derivacion" y
	// la plantilla de Alcance lleva un `class="derivacion"` que no tiene nada
	// que ver con el camino. Buscar el identificador suelto acusa a la
	// maquetacion, que es justo el fallo probable de un detector por subcadena.
	for _, p := range pasos {
		for _, clave := range []string{p.Titulo, p.Verbo} {
			if clave == "" {
				continue
			}
			if strings.Contains(texto, `"`+clave+`"`) {
				t.Errorf("las plantillas de esta superficie nombran %q. Los pasos del camino "+
					"llegan como DATO (Opciones.Pasos, que sale de camino.Canonico()): "+
					"escribir uno aqui abre una segunda lista, y la segunda lista se queda "+
					"vieja el dia que el camino cambie", clave)
			}
		}
		if p.Ruta != "" && strings.Contains(texto, `href="`+p.Ruta+`"`) {
			t.Errorf("las plantillas escriben a mano la ruta %q de un paso del camino. La "+
				"ruta la declara superficies/camino y quien monta la lee de alli", p.Ruta)
		}
	}
}

// DONDE ESTAS, y donde no estas.
//
// El paso actual se resuelve por la RUTA de la pantalla, asi que una pantalla
// que no es ningun paso (Hoy, Certificados, Personas, Estado) no puede marcar
// ninguno: marcar el primero "por si acaso" seria decirle al operador que esta
// en un sitio en el que no esta.
func TestLaBarraLateralDiceEnQuePasoEstasYCallaCuandoNoEstasEnNinguno(t *testing.T) {
	s, _ := superficie(t, corpusDemo(), conCamino())
	casos := []struct{ ruta, paso string }{
		{"/alcance", "alcance"},
		{"/controles", "derivacion"},
		{"/hoy", ""},
		{"/certificados", ""},
		{"/personas", ""},
		{"/estado", ""},
	}
	marcados := 0
	for _, c := range casos {
		w, cuerpo := pedir(t, s, c.ruta)
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s dio %d", c.ruta, w.Code)
		}
		tira := tiraDe(t, cuerpo)
		n := strings.Count(tira, `aria-current="step"`)
		if c.paso == "" {
			if n != 0 {
				t.Errorf("%s no es ningun paso del camino y la barra marca %d como actual",
					c.ruta, n)
			}
			// Y tampoco el rotulo escrito.
			if strings.Contains(tira, rotulo("es", "ui.aqui")) {
				t.Errorf("%s no es ningun paso y la barra dice 'estas aqui'", c.ruta)
			}
			continue
		}
		marcados++
		if n != 1 {
			t.Errorf("%s tenia que marcar exactamente un paso y marca %d", c.ruta, n)
		}
		// El marcado es el que toca: se busca el rotulo de ESE paso dentro del
		// bloque del enlace que lleva aria-current.
		titulo := ""
		for _, p := range camino.Canonico() {
			if p.ID == c.paso {
				titulo = p.Titulo
			}
		}
		if titulo == "" {
			t.Fatalf("el camino canonico ya no tiene el paso %q: este caso no prueba nada", c.paso)
		}
		i := strings.Index(tira, `aria-current="step"`)
		fin := strings.Index(tira[i:], "</a>")
		if fin < 0 {
			t.Fatal("el enlace del paso actual no cierra")
		}
		bloque := tira[i : i+fin]
		if !strings.Contains(bloque, rotulo("es", titulo)) {
			t.Errorf("%s marca como actual un paso que no es %q.\n--- bloque ---\n%s",
				c.ruta, c.paso, bloque)
		}
		// EL COLOR NO ES EL UNICO PORTADOR: el paso actual lo dice escrito.
		if !strings.Contains(bloque, rotulo("es", "ui.aqui")) {
			t.Errorf("%s marca el paso actual solo con la clase, sin decirlo con palabras. "+
				"Quien no distingue el indigo del gris se queda sin saber donde esta", c.ruta)
		}
	}
	if marcados == 0 {
		t.Fatal("ningun caso ha llegado a marcar un paso: esta puerta no ha comprobado su " +
			"mitad afirmativa")
	}
}

// NINGUN PASO DE LA BARRA ES UN CALLEJON. Todos llevan a algun sitio: los que
// son pantalla, a la suya; los que todavia no lo son, a la pantalla del camino,
// que es donde esta la orden exacta que los hace hoy.
//
// Y los que no son pantalla lo DICEN con su rotulo, no solo con un color mas
// palido: quien pulse esperando el calendario y llegue a una lista de pasos ha
// perdido un clic y algo de confianza.
func TestNingunPasoDeLaBarraLateralEsUnCallejon(t *testing.T) {
	s, _ := superficie(t, corpusDemo(), conCamino())
	_, cuerpo := pedir(t, s, "/alcance")
	tira := tiraDe(t, cuerpo)

	// Un enlace por paso, ni uno menos.
	enlaces := regexp.MustCompile(`href="([^"]*)"`).FindAllStringSubmatch(tira, -1)
	if len(enlaces) != len(camino.Canonico()) {
		t.Fatalf("la barra pinta %d enlaces y el camino tiene %d pasos: alguno se ha quedado "+
			"sin destino", len(enlaces), len(camino.Canonico()))
	}
	for _, m := range enlaces {
		u := strings.ReplaceAll(m[1], "&amp;", "&")
		if u == "" || u == "#" {
			t.Error("hay un paso con un enlace vacio en la barra lateral")
		}
		if !strings.HasPrefix(u, "/") || strings.HasPrefix(u, "//") {
			t.Errorf("el paso enlaza a %q, que no es una direccion de este sitio", u)
		}
	}
	// Los que todavia no son pantalla salen apagados Y con su apunte escrito.
	sinPantalla := camino.SinPantalla()
	if len(sinPantalla) == 0 {
		t.Skip("ya no queda ningun paso sin pantalla: esta mitad no tiene nada que probar")
	}
	if n := strings.Count(tira, rotulo("es", "ui.paso_por_terminal")); n != len(sinPantalla) {
		t.Errorf("hay %d pasos que todavia no son pantalla y la barra lo dice %d veces. "+
			"Un paso que lleva a otro sitio del que promete tiene que decirlo",
			len(sinPantalla), n)
	}
}

// LA TIRA LLEVA LAS RESPUESTAS PUESTAS, y solo en los pasos que las usan.
//
// Las respuestas de la entrevista viajan en la direccion y no se guardan (la
// propia pantalla de Alcance lo dice con esas palabras), asi que una barra con
// enlaces pelados se come el trabajo de quien la use. Es el fallo que el camino
// guiado ya tuvo una vez, y aqui reaparece en TODAS las pantallas a la vez.
//
// Y no se le cuelgan a todos: al acta y a la revision de accesos no les dice
// nada el alcance, asi que arrastrarles la consulta seria llevar un dato hasta
// una pantalla que no lo entiende.
func TestLaBarraLateralNoSeComeLasRespuestasDeLaEntrevista(t *testing.T) {
	s, _ := superficie(t, corpusDemo(), conCamino())
	_, cuerpo := pedir(t, s, "/alcance?si=alfa.q.categoria&no=beta.q.riesgo")
	tira := tiraDe(t, cuerpo)

	destinos := map[string]string{}
	for _, m := range regexp.MustCompile(`href="([^"]*)"`).FindAllStringSubmatch(tira, -1) {
		u := strings.ReplaceAll(m[1], "&amp;", "&")
		ruta, consulta, _ := strings.Cut(u, "?")
		destinos[ruta] = consulta
	}
	conAlcance, sinAlcance := 0, 0
	for _, p := range camino.Canonico() {
		if !p.EsPantalla() {
			continue
		}
		consulta, hay := destinos[p.Ruta]
		if !hay {
			t.Errorf("la barra no enlaza al paso %q en %q", p.ID, p.Ruta)
			continue
		}
		if p.LlevaAlcance {
			conAlcance++
			if !strings.Contains(consulta, "si=alfa.q.categoria") ||
				!strings.Contains(consulta, "no=beta.q.riesgo") {
				t.Errorf("el paso %q trabaja con el alcance y su enlace en la barra va sin "+
					"las respuestas (%q): pulsarlo borraria la entrevista", p.ID, consulta)
			}
			continue
		}
		sinAlcance++
		if consulta != "" {
			t.Errorf("el paso %q no trabaja con el alcance y su enlace arrastra %q hasta una "+
				"pantalla que no sabe leerlo", p.ID, consulta)
		}
	}
	// Las dos mitades tienen que haberse recorrido, o esto solo prueba una.
	if conAlcance == 0 || sinAlcance == 0 {
		t.Fatalf("se han comprobado %d pasos con alcance y %d sin el: falta una de las dos "+
			"direcciones", conAlcance, sinAlcance)
	}
}

// EL VALOR CERO ES NO PINTAR NADA, y es el restrictivo.
//
// Sin pasos, esta superficie no sabe donde esta montado el acta ni la revision
// de accesos ni si estan montadas siquiera. Rellenar el camino con el canonico
// "por comodidad" convertiria un olvido de quien monta en seis enlaces a un 404
// en todas las pantallas, que es peor que no tener barra.
func TestSinPasosLaBarraLateralNoSeInventaElCamino(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	for _, ruta := range []string{"/alcance", "/hoy", "/no-existe"} {
		_, cuerpo := pedir(t, s, ruta)
		if strings.Contains(cuerpo, "tira-camino") {
			t.Errorf("%s pinta la tira del camino sin que nadie haya pasado los pasos", ruta)
		}
		// Y las seis pantallas siguen ahi: la barra pierde el camino, no el menu.
		if !strings.Contains(cuerpo, `id="navegacion"`) {
			t.Errorf("%s se ha quedado ademas sin el menu de las pantallas", ruta)
		}
	}
}

// Un camino roto no se pinta a medias: se rechaza al construir, con el mismo
// juez que usa la pantalla del camino.
func TestUnCaminoRotoNoLlegaAPintarse(t *testing.T) {
	malos := []struct {
		que   string
		pasos []camino.Paso
	}{
		{"un paso sin salida", []camino.Paso{
			{ID: "x", Titulo: "camino.paso.alcance", Verbo: "camino.verbo.alcance"}}},
		{"un paso sin verbo", []camino.Paso{
			{ID: "x", Titulo: "camino.paso.alcance", Ruta: "/alcance"}}},
		{"dos pasos con el mismo identificador", []camino.Paso{
			{ID: "x", Titulo: "camino.paso.alcance", Verbo: "v", Ruta: "/alcance"},
			{ID: "x", Titulo: "camino.paso.acta", Verbo: "v", Ruta: "/acta/"}}},
		{"una ruta a otro anfitrion", []camino.Paso{
			{ID: "x", Titulo: "camino.paso.alcance", Verbo: "v", Ruta: "//evil.example/x"}}},
	}
	for _, c := range malos {
		_, err := Nuevo(Opciones{Catalogo: nuevoCatalogo(), Paquetes: corpusDemo(),
			Pasos: c.pasos})
		if err == nil {
			t.Errorf("%s: la superficie se construye igual, asi que la barra lateral pintaria "+
				"un camino que no se puede recorrer", c.que)
		}
	}
	// Control positivo: el canonico si se construye. Sin esto, un constructor
	// que rechazara TODO pasaria el bucle de arriba con nota.
	if _, err := Nuevo(Opciones{Catalogo: nuevoCatalogo(), Paquetes: corpusDemo(),
		Pasos: camino.Canonico()}); err != nil {
		t.Fatalf("el camino canonico se rechaza: %v", err)
	}
}

// TODA ZONA QUE SE DESPLAZA SE ALCANZA CON EL TECLADO.
//
// LA PUERTA NACE DE UN ROJO DE VERDAD, no de una mutacion: al correr axe-core en
// local contra el servidor montado, en los dos temas y los dos idiomas, salieron
// OCHO violaciones de scrollable-region-focusable, todas sobre .marco-tabla, en
// /controles y /certificados. La causa fue la cabecera de tabla fija: para
// pegarla arriba hace falta que el marco tenga altura maxima y desplazamiento
// vertical, y una zona desplazable a la que no llega el tabulador deja la tabla
// entera de obligaciones fuera del alcance de quien no usa raton. En esta
// pantalla, esa tabla es toda la informacion.
//
// Se vigila desde Go y no solo desde axe por dos razones: axe corre en CI y esto
// corre en cada `go test`, y sobre todo axe solo mira las paginas que se le
// pasan, mientras que esto mira TODAS las que sirve la superficie.
func TestTodaZonaDesplazableSeAlcanzaConElTeclado(t *testing.T) {
	s, _ := superficie(t, corpusDemo(), conCamino())
	// El marco de tabla es hoy la unica zona con desplazamiento propio, y la
	// hoja de estilo es quien lo decide: se lee de ahi en vez de fiarse de una
	// lista, para que una zona nueva con overflow entre sola.
	hoja := leerHoja(t)
	if !strings.Contains(hoja, ".marco-tabla") {
		t.Fatal("la hoja ya no declara .marco-tabla: este detector no vigila nada")
	}
	vistas := 0
	for _, ruta := range []string{"/controles", "/certificados", "/hoy", "/alcance"} {
		_, cuerpo := pedir(t, s, ruta)
		for _, m := range regexp.MustCompile(`<div class="marco-tabla"[^>]*>`).
			FindAllString(cuerpo, -1) {
			vistas++
			if !strings.Contains(m, `tabindex="0"`) {
				t.Errorf("%s: una zona desplazable sin tabindex. Quien navega con el "+
					"tabulador no puede llegar a su contenido.\n  %s", ruta, m)
			}
			if !strings.Contains(m, "aria-label=") && !strings.Contains(m, "aria-labelledby=") {
				t.Errorf("%s: una zona desplazable enfocable y sin nombre. Un lector de "+
					"pantalla anuncia 'grupo' y ya.\n  %s", ruta, m)
			}
		}
	}
	if vistas == 0 {
		t.Fatal("no se ha encontrado ni una zona desplazable en las cuatro pantallas: el " +
			"detector no esta mirando el HTML que cree mirar")
	}
}

// itoa sin importar strconv en un fichero que solo lo necesita para un numero
// de un digito o dos.
func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}

var _ = filepath.Base

// NINGUN NOMBRE DE PLANTILLA SE DEFINE DOS VECES.
//
// Es el fallo que un motor de plantillas NO avisa, y por eso hace falta puerta:
// dos {{define "x"}} en el mismo arbol se pisan en silencio y gana el ultimo que
// se parsee, que depende del orden del glob. O sea que el sintoma no es un error
// sino una pagina que cambia de aspecto al renombrar un fichero.
//
// Sale ahora porque esta superficie carga DOS arboles: el suyo y el armazon
// compartido que declara superficies/camino. Mientras el marcado vivia entero
// aqui, la colision no podia existir; desde que se comparte, el dibujo de los
// estados vacios estuvo definido en los dos a la vez durante un rato, y nada se
// puso rojo. Este detector es el que faltaba.
func TestNingunNombreDePlantillaSeDefineDosVeces(t *testing.T) {
	re := regexp.MustCompile(`\{\{-?\s*define "([^"]+)"`)
	donde := map[string][]string{}
	for _, f := range plantillasEnDisco(t) {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range re.FindAllStringSubmatch(string(b), -1) {
			donde[m[1]] = append(donde[m[1]], filepath.Base(f))
		}
	}
	if len(donde) < 5 {
		t.Fatalf("se han encontrado %d definiciones de plantilla y hay muchas mas: el "+
			"detector esta mirando otra cosa", len(donde))
	}
	for nombre, ficheros := range donde {
		if len(ficheros) > 1 {
			t.Errorf("la plantilla %q se define en %v. En un mismo arbol una se come a la "+
				"otra y el motor no dice nada: gana la ultima que se parsee, o sea el orden "+
				"del glob", nombre, ficheros)
		}
	}
	// CONTROL POSITIVO: el armazon compartido tiene que estar entre lo mirado.
	// Sin el, este detector solo veria un arbol y la colision que existe para
	// cazar seria imposible por construccion.
	if _, hay := donde["tira-camino"]; !hay {
		t.Error("entre lo mirado no esta la tira del camino, que vive en el armazon " +
			"compartido: el detector no esta recorriendo los dos arboles")
	}
}
