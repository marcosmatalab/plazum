package acta

import (
	"io/fs"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/superficies/camino"
)

// LA BARRA LATERAL EN EL ACTA.
//
// Que problema resuelve, y no es de estetica. Hasta aqui esta pantalla armaba su
// propio cuerpo y su unica salida era el enlace de vuelta al camino. Quien la
// abria (y esta se abre desde un correo o desde el orden del dia de un consejo,
// casi nunca navegando desde la portada) no tenia forma de saber en que paso del
// camino estaba, cual era el siguiente, ni que existieran las otras pantallas.
// Eran tres superficies de cuatro sin armazon, contadas en docs/pendientes.md.
//
// Lo que estas puertas sostienen es lo mismo que en superficies/pantallas: la
// tira SALE DEL CAMINO y no de una lista escrita al lado, y el paso marcado es
// el del acta y no otro.

// conCamino monta la superficie como la monta el producto: con el camino
// canonico en la barra lateral y con la vuelta a su pantalla.
func conCamino(t *testing.T, f Actas, conSesion bool) *Superficie {
	t.Helper()
	o := Opciones{
		Fuente: f, Catalogo: cat(t), Base: "/acta", Estatico: "/estatico",
		CaminoRuta:  camino.BasePorDefecto + "/",
		CaminoClave: camino.ClaveTitulo,
		Pasos:       camino.Canonico(),
	}
	if conSesion {
		o.Quien = func(*http.Request) string { return "u-042" }
	}
	s, err := Nuevo(o)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

var reTira = regexp.MustCompile(`(?s)<nav class="tira-camino".*?</nav>`)

func tiraDe(t *testing.T, cuerpo string) string {
	t.Helper()
	m := reTira.FindString(cuerpo)
	if m == "" {
		t.Fatal("la pagina del acta no trae la tira del camino en la barra lateral")
	}
	return m
}

// La tira es EXACTAMENTE el camino canonico, en su orden y con sus numeros.
//
// Se compara contra camino.Canonico(), que es la fuente, y no contra una lista
// escrita aqui: un test que se mide contra su propia copia no comprueba nada.
func TestElActaPintaElCaminoCanonicoEnteroEnSuBarraLateral(t *testing.T) {
	s := conCamino(t, &fuente{a: actaDePrueba(t), hay: true}, true)
	cuerpo := pedir(t, s, "GET", "/acta/").Body.String()
	tira := tiraDe(t, cuerpo)

	pasos := camino.Canonico()
	if len(pasos) == 0 {
		t.Fatal("camino.Canonico() ha llegado vacio: este test estaria recorriendo la nada")
	}
	antes := -1
	for i, p := range pasos {
		rot := rotulo(t, p.Titulo)
		pos := strings.Index(tira, rot)
		if pos < 0 {
			t.Errorf("el paso %q (%s) no sale en la barra lateral del acta. Un camino de %d "+
				"pasos del que se ven menos parece completo y no lo esta", p.ID, p.Titulo,
				len(pasos))
			continue
		}
		if pos < antes {
			t.Errorf("el paso %q sale ANTES que el anterior: el orden del camino no es "+
				"decorativo, cada paso consume lo que produce el de antes", p.ID)
		}
		antes = pos
		if !strings.Contains(tira, `<span class="numero">`+itoa(i+1)+`</span>`) {
			t.Errorf("el paso %d no lleva su numero en la barra lateral del acta", i+1)
		}
	}
}

// EL PASO MARCADO ES EL DEL ACTA, y se marca de tres formas y no solo con color.
//
// El color solo no vale y esto no es una preferencia: quien no distingue el
// indigo del gris tiene que poder LEER donde esta. Van aria-current="step" y el
// rotulo escrito, ademas del fondo.
func TestLaBarraLateralDelActaMarcaElPasoDelActaYNingunOtro(t *testing.T) {
	s := conCamino(t, &fuente{a: actaDePrueba(t), hay: true}, true)
	tira := tiraDe(t, pedir(t, s, "GET", "/acta/").Body.String())

	if n := strings.Count(tira, `aria-current="step"`); n != 1 {
		t.Fatalf("hay %d pasos marcados como el actual y tiene que haber exactamente 1: "+
			"cero deja al operador sin saber donde esta y dos le mienten", n)
	}
	// Y ES EL DEL ACTA. Se busca el rotulo del paso cuyo ID declara el camino,
	// no la palabra "acta" escrita aqui: el identificador es camino.IDDelActa y
	// se usa DENTRO de la lista canonica, asi que renombrarlo mueve las dos
	// puntas a la vez.
	var titulo string
	for _, p := range camino.Canonico() {
		if p.ID == camino.IDDelActa {
			titulo = p.Titulo
		}
	}
	if titulo == "" {
		t.Fatal("el camino canonico no trae ningun paso con el ID del acta: o el paso ha " +
			"desaparecido, o la constante ya no es la que usa la lista")
	}
	li := trozoDelPaso(t, tira, rotulo(t, titulo))
	if !strings.Contains(li, `aria-current="step"`) {
		t.Errorf("el paso del acta no lleva aria-current=\"step\": un lector de pantalla no "+
			"diria donde esta quien lo usa.\n%s", li)
	}
	if !strings.Contains(li, rotulo(t, "ui.aqui")) {
		t.Errorf("el paso del acta no lleva el rotulo escrito de \"estas aqui\". El color no "+
			"puede ser la unica senal.\n%s", li)
	}
	if !strings.Contains(li, "actual") {
		t.Errorf("el paso del acta no lleva la clase que lo destaca.\n%s", li)
	}
}

// LA BARRA SE PINTA EN LOS CUATRO ESTADOS DE LA PANTALLA, y los dos que mas
// importan son los vacios.
//
// Sin sesion y sin acta son justo los dos en los que quien llega se queda
// mirando una pagina que no le dice nada: son en los que la orientacion vale
// mas, no menos. Es la misma razon por la que el enlace de vuelta se pinta ahi
// (puerta D11-b) y por la que este test los recorre uno a uno en vez de
// conformarse con el caso feliz.
func TestLaBarraLateralDelActaSalgaLoQueSalgaEnElCuerpo(t *testing.T) {
	llena := &fuente{a: actaDePrueba(t), hay: true}
	casos := []struct {
		nombre    string
		f         Actas
		conSesion bool
		ruta      string
	}{
		{"sin sesion", llena, false, "/acta/"},
		{"sin acta", nil, true, "/acta/"},
		{"con acta", llena, true, "/acta/"},
		{"una cifra abierta", llena, true, "/acta/derivacion/1.1.1"},
		{"una referencia que no existe", llena, true, "/acta/derivacion/9.9.9"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			s := conCamino(t, c.f, c.conSesion)
			cuerpo := pedir(t, s, "GET", c.ruta).Body.String()
			tira := tiraDe(t, cuerpo)
			// Y con el camino entero dentro, no con la etiqueta vacia.
			for _, p := range camino.Canonico() {
				if !strings.Contains(tira, rotulo(t, p.Titulo)) {
					t.Errorf("en el estado %q falta el paso %q en la barra", c.nombre, p.ID)
				}
			}
		})
	}
}

// SIN PASOS NO SE PINTA BARRA, y esa es la mitad restrictiva (invariante 8).
//
// Las DOS FORMAS DE LA NADA se recorren: nil (el olvido de quien monta) y
// vacio-presente (una lista construida y filtrada entera). Las dos tienen que
// dar la misma pantalla que habia antes de que existiera el armazon, porque la
// alternativa (rellenar con el canonico) convertiria un olvido en una barra
// plausible que enlaza a rutas donde nadie ha montado nada, y el sintoma serian
// 404 desde el documento que lee un consejo.
func TestSinPasosElActaNoSeInventaLaBarra(t *testing.T) {
	for _, c := range []struct {
		nombre string
		pasos  []camino.Paso
	}{
		{"nil", nil},
		{"vacio presente", []camino.Paso{}},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			s, err := Nuevo(Opciones{
				Fuente: &fuente{a: actaDePrueba(t), hay: true}, Catalogo: cat(t),
				Base: "/acta", Estatico: "/estatico", Pasos: c.pasos,
				Quien: func(*http.Request) string { return "u-042" },
			})
			if err != nil {
				t.Fatal(err)
			}
			cuerpo := pedir(t, s, "GET", "/acta/").Body.String()
			if reTira.MatchString(cuerpo) {
				t.Error("con el camino vacio se ha pintado una barra igualmente: eso enlaza " +
					"a pantallas que quien monta no ha montado")
			}
			// Pero la pagina sigue estando entera: no pintar barra no es no
			// pintar nada.
			if !strings.Contains(cuerpo, rotulo(t, "acta.titulo.documento")) {
				t.Error("sin barra la pantalla se ha quedado sin su titulo: el armazon se " +
					"ha llevado por delante el documento")
			}
		})
	}
}

// UN CAMINO ROTO NO SE PINTA A MEDIAS: no se arranca.
//
// Y lo juzga EL MISMO validador que la pantalla del camino (camino.Validar), no
// una segunda comprobacion escrita aqui: dos jueces de la misma propiedad acaban
// discrepando, y el dia que discrepen la barra del acta y la pantalla del camino
// ensenarian caminos distintos sin que nada se pusiera rojo.
func TestUnCaminoRotoNoConstruyeLaPantallaDelActa(t *testing.T) {
	_, err := Nuevo(Opciones{
		Catalogo: cat(t), Base: "/acta", Estatico: "/estatico",
		// Un paso sin pantalla y sin orden: un callejon.
		Pasos: []camino.Paso{{ID: "x", Titulo: "t", Verbo: "v"}},
	})
	if err == nil {
		t.Fatal("un camino con un paso sin salida ha construido la pantalla del acta")
	}
	if !strings.Contains(err.Error(), "barra") {
		t.Errorf("el error no dice que el problema es la barra lateral: %v", err)
	}
}

// LA MARCA LLEVA A LA RAIZ DEL SITIO, no a "/acta/".
//
// Parece un detalle y es el fallo probable exacto de compartir el armazon: cada
// superficie tiene su propio Base, y reutilizarlo para el logo deja a quien se
// ha perdido dando vueltas dentro de la pantalla en la que se perdio.
func TestLaMarcaDelActaLlevaALaRaizYNoAlActa(t *testing.T) {
	s := conCamino(t, &fuente{a: actaDePrueba(t), hay: true}, true)
	cuerpo := pedir(t, s, "GET", "/acta/").Body.String()
	m := regexp.MustCompile(`<a class="marca" href="([^"]*)"`).FindStringSubmatch(cuerpo)
	if m == nil {
		t.Fatal("la pagina del acta no trae la marca: el armazon no se ha pintado")
	}
	if m[1] != "/" {
		t.Errorf("la marca del acta apunta a %q y tiene que apuntar a la raiz del sitio. "+
			"Con el prefijo de esta superficie, el logo te devuelve a la misma pantalla", m[1])
	}
}

// Y NO HAY NINGUNA LISTA DE PASOS ESCRITA EN LA PLANTILLA.
//
// Es la mitad que de verdad importa: la primera puerta seguiria verde si alguien
// escribiera los seis rotulos a mano, y entonces el dia que el camino gane un
// septimo paso la barra del acta ensenaria seis para siempre. Aqui se miran los
// FICHEROS (el propio y el compartido), no la pagina.
func TestNiElActaNiElArmazonEscribenUnPasoAMano(t *testing.T) {
	var texto string
	for _, f := range []struct {
		sistema fs.FS
		nombre  string
	}{
		{plantillasFS, "plantillas/acta.html"},
		{camino.PlantillasDelArmazon(), "armazon/armazon.html"},
	} {
		b, err := fs.ReadFile(f.sistema, f.nombre)
		if err != nil {
			t.Fatal(err)
		}
		texto += string(b)
	}
	if !strings.Contains(texto, "tira-camino") {
		t.Fatal("las plantillas leidas no traen la tira del camino: este detector no tiene " +
			"nada que vigilar")
	}
	pasos := camino.Canonico()
	if len(pasos) == 0 {
		t.Fatal("camino.Canonico() ha llegado vacio")
	}
	for _, p := range pasos {
		for _, clave := range []string{p.Titulo, p.Verbo} {
			if clave == "" {
				continue
			}
			if strings.Contains(texto, `"`+clave+`"`) {
				t.Errorf("una plantilla nombra %q. Los pasos del camino llegan como DATO "+
					"(camino.Canonico()): escribir uno aqui abre una segunda copia del "+
					"orden, y la segunda copia es la que se queda vieja", clave)
			}
		}
	}
}

// rotulo devuelve el texto con el que el catalogo de verdad pinta una clave.
//
// Se pregunta al catalogo en vez de escribir la palabra aqui: una cadena escrita
// en el test se queda vieja el dia que alguien mejore la redaccion, y entonces
// el test rojo acusa a un cambio que no rompio nada.
func rotulo(t *testing.T, clave string) string {
	t.Helper()
	s := cat(t).Traducir("es", clave)
	if s == "" || s == clave {
		t.Fatalf("el catalogo no traduce %q, asi que este test estaria buscando la clave "+
			"cruda y daria verde en cuanto la pantalla la pintara mal", clave)
	}
	return s
}

// trozoDelPaso recorta el <li> que contiene un rotulo, para poder afirmar cosas
// de UN paso y no de la tira entera.
//
// Sin esto, "la tira lleva aria-current" seria cierto aunque el marcado
// estuviera en otro paso, que es exactamente el fallo que este fichero busca.
func trozoDelPaso(t *testing.T, tira, rot string) string {
	t.Helper()
	i := strings.Index(tira, rot)
	if i < 0 {
		t.Fatalf("el rotulo %q no sale en la tira", rot)
	}
	desde := strings.LastIndex(tira[:i], "<li")
	hasta := strings.Index(tira[i:], "</li>")
	if desde < 0 || hasta < 0 {
		t.Fatalf("no puedo recortar el paso de %q en la tira", rot)
	}
	return tira[desde : i+hasta]
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
