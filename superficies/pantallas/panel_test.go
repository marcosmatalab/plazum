package pantallas

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// EL PANEL DE INICIO: las cuatro cifras y sus derivaciones.
//
// Lo que se vigila aqui no es que la pantalla sea bonita. Son tres propiedades
// que este producto no se puede permitir romper:
//
//	NINGUNA CIFRA SIN ENLACE. Un numero que no se puede abrir es un numero que
//	hay que creerse, y plazum se vende exactamente al reves.
//	UN CERO NO ES UNA AUSENCIA. Cuando no se ha podido contar, la cifra dice SIN
//	DATO, no cero. Es el invariante 8 en una pantalla.
//	LO NO CONSTATADO NO ES CULPA. La cifra de vencimientos pasados lleva su
//	descargo dentro de la misma tarjeta, con numero y sin el.

// reCifra recorta cada tarjeta de cifra del panel.
var reCifra = regexp.MustCompile(`(?s)<li class="cifra[^"]*">.*?</li>`)

func cifrasDe(t *testing.T, cuerpo string) []string {
	t.Helper()
	c := reCifra.FindAllString(cuerpo, -1)
	if len(c) == 0 {
		t.Fatal("la pantalla Hoy no trae ni una cifra: el panel de inicio no se esta pintando")
	}
	return c
}

// NINGUNA CIFRA SE QUEDA SIN SU DERIVACION, y la derivacion EXISTE.
//
// No basta con que haya un href: un enlace a un ancla que no esta en ninguna
// parte lleva al mismo sitio que ninguno, y un enlace a una ruta que contesta
// 404 es peor, porque parece que va a algun lado. Se comprueban las dos formas.
func TestNingunaCifraDelPanelSeQuedaSinSuDerivacion(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	w, cuerpo := pedir(t, s, "/hoy")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /hoy dio %d", w.Code)
	}
	cifras := cifrasDe(t, cuerpo)
	anclas, rutas := 0, 0
	for _, c := range cifras {
		m := regexp.MustCompile(`href="([^"]+)"`).FindStringSubmatch(c)
		if m == nil {
			t.Errorf("hay una cifra sin enlace, o sea un numero que hay que creerse.\n"+
				"--- cifra ---\n%s", c)
			continue
		}
		destino := strings.ReplaceAll(m[1], "&amp;", "&")
		ruta, ancla, hayAncla := strings.Cut(destino, "#")
		if hayAncla {
			anclas++
			// El ancla tiene que existir en la pagina que la enlaza.
			if !strings.Contains(cuerpo, `id="`+ancla+`"`) {
				t.Errorf("la cifra enlaza a #%s y esa seccion no existe en la pagina: el "+
					"numero se abre a la nada", ancla)
			}
		}
		if ruta == "" {
			continue
		}
		rutas++
		wr, _ := pedir(t, s, ruta)
		if wr.Code != http.StatusOK {
			t.Errorf("la cifra enlaza a %q y esa direccion contesta %d", ruta, wr.Code)
		}
	}
	// Las dos formas tienen que haberse recorrido. Si el panel se quedara con
	// una sola, esta puerta seguiria verde sin comprobar la otra.
	if anclas == 0 || rutas == 0 {
		t.Fatalf("se han comprobado %d anclas y %d rutas: falta una de las dos formas de "+
			"derivacion", anclas, rutas)
	}
}

// LA CIFRA Y SU DERIVACION CUADRAN.
//
// Es la puerta D11-c donde mas se ve: si el numero grande dice tres y la lista
// de debajo tiene dos filas, una de las dos miente y no hay forma de saber cual.
// Se comprueba con un corpus que produce un vencimiento pasado, o sea con la
// cifra que de verdad se puede leer mal.
func TestLaCifraYLaListaQueLaAbreDicenLoMismo(t *testing.T) {
	s, _ := superficie(t, []*corpus.Paquete{paqueteVencido()})
	_, cuerpo := pedir(t, s, "/hoy")

	n := numeroDeLaCifra(t, cuerpo, "pantalla.hoy.cifra.sin_constancia")
	if n == 0 {
		t.Fatal("el paquete de prueba tenia que producir al menos un vencimiento pasado. " +
			"Sin el, esta puerta no comprueba nada: es el control positivo del descargo")
	}
	seccion := seccionConID(t, cuerpo, "sin-constancia")
	filas := strings.Count(seccion, `<li class="veredicto`)
	if filas != n {
		t.Errorf("la cifra dice %d y su derivacion trae %d filas. Un numero que no cuadra "+
			"con la lista que lo abre es peor que no tener el numero", n, filas)
	}
}

// EL DESCARGO VIAJA CON EL DATO, y viaja SIEMPRE.
//
// Con las dos ramas recorridas de verdad, que es lo que M47 dejo escrito: una
// rama de descargo que ninguna entrada alcanza es una rama que no existe, y la
// mutacion la deja verde porque no hay nada que romper. Aqui la rama con numero
// se recorre con un corpus que produce vencimientos pasados, y la de cero con el
// corpus de demostracion, que no produce ninguno.
func TestElDescargoDeLoNoConstatadoVaDentroDeLaMismaTarjetaQueElNumero(t *testing.T) {
	casos := []struct {
		que     string
		ps      []*corpus.Paquete
		conDato bool
	}{
		{"con vencimientos pasados", []*corpus.Paquete{paqueteVencido()}, true},
		{"sin ninguno", corpusDemo(), false},
	}
	vistos := 0
	for _, c := range casos {
		t.Run(c.que, func(t *testing.T) {
			s, _ := superficie(t, c.ps)
			_, cuerpo := pedir(t, s, "/hoy")

			n := numeroDeLaCifra(t, cuerpo, "pantalla.hoy.cifra.sin_constancia")
			if c.conDato && n == 0 {
				t.Fatal("este caso tenia que traer vencimientos pasados y trae cero: la " +
					"rama acusatoria no se ha recorrido y el descargo no vigila nada")
			}
			if !c.conDato && n != 0 {
				t.Fatalf("este caso tenia que traer cero y trae %d", n)
			}
			vistos++

			// La tarjeta de esa cifra, entera, tiene que llevar el descargo
			// DENTRO. No vale que este en la pagina: tiene que estar donde se
			// lee el numero, porque es el numero el que se lee como acusacion.
			tarjeta := tarjetaDeLaCifra(t, cuerpo, "pantalla.hoy.cifra.sin_constancia")
			if !strings.Contains(tarjeta, rotulo("es", "pantalla.hoy.sin_constancia.descargo")) {
				t.Errorf("la cifra de vencimientos sin constancia se pinta SIN su descargo "+
					"al lado.\n"+
					"  Un numero grande en un panel se lee como una acusacion, y lo que "+
					"dice ese numero es que en las respuestas no consta nada, no que se "+
					"haya incumplido.\n"+
					"  Arreglo: el descargo lo pone panel.go en VistaCifra.Descargo y lo "+
					"pinta hoy.html dentro de la tarjeta.\n--- tarjeta ---\n%s", tarjeta)
			}
			// Y tambien en la seccion que abre la cifra: quien llega por el
			// ancla no ha visto la tarjeta.
			seccion := seccionConID(t, cuerpo, "sin-constancia")
			if !strings.Contains(seccion, rotulo("es", "pantalla.hoy.sin_constancia.descargo")) {
				t.Error("la lista de vencimientos sin constancia sale sin el descargo. Quien " +
					"llega por el ancla no ha leido la tarjeta de arriba")
			}
		})
	}
	if vistos != 2 {
		t.Fatalf("solo se han recorrido %d de las 2 ramas del descargo", vistos)
	}
}

// UNA CIFRA QUE NO SE HA PODIDO CONTAR NO ES UN CERO.
//
// Sin corpus instalado no hay nada que contar, y "0 te alcanzan" seria la
// respuesta a una pregunta que nadie ha calculado. Son dos frases distintas y
// solo una de ellas es un numero.
func TestSinCorpusLasCifrasDicenSinDatoYNoCero(t *testing.T) {
	s, _ := superficie(t, nil)
	w, cuerpo := pedir(t, s, "/hoy")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /hoy sin corpus dio %d", w.Code)
	}
	cifras := cifrasDe(t, cuerpo)
	for _, c := range cifras {
		if strings.Contains(c, `<span class="n">`) {
			t.Errorf("sin corpus instalado, una cifra sale con numero.\n"+
				"  Un cero dice 'he contado y no hay nada' y aqui no se ha contado nada: "+
				"son dos cosas distintas y la barata de confundir es la que hace dano.\n"+
				"--- cifra ---\n%s", c)
		}
		if !strings.Contains(c, rotulo("es", "pantalla.hoy.cifra.sin_dato")) {
			t.Errorf("una cifra sin dato no lo dice.\n--- cifra ---\n%s", c)
		}
		// Y el motivo, que es lo que separa "no hay corpus" de "no hay estado".
		if !strings.Contains(c, rotulo("es", "pantalla.hoy.cifra.sin_corpus")) {
			t.Errorf("una cifra sin dato no dice por que.\n--- cifra ---\n%s", c)
		}
	}
	// Control positivo de la puerta: CON corpus, las mismas cifras si traen
	// numero. Sin este tramo, una plantilla que nunca pintara numeros pasaria
	// el bucle de arriba con nota.
	s2, _ := superficie(t, corpusDemo())
	_, con := pedir(t, s2, "/hoy")
	if !strings.Contains(con, `<span class="n">`) {
		t.Fatal("con corpus instalado, el panel tampoco pinta ni un numero: esta puerta " +
			"estaria dando verde sobre una pantalla que no cuenta nada")
	}
}

// UNA CIFRA CON DATO NO PINTA EL MOTIVO DE NO TENERLO.
//
// LA PUERTA NACIO DE UN FALLO REAL, no de una mutacion. Al arrancar `plazum
// serve` contra el corpus de verdad por primera vez, la pantalla decia
//
//	16   marcos instalados
//	     No hay ningun paquete normativo instalado, asi que no hay nada que contar.
//
// en dos lineas seguidas y dentro de la misma tarjeta. La causa era un solo
// campo haciendo dos trabajos (el motivo de la ausencia y el matiz del numero),
// y un campo compartido se pinta siempre. Ninguna de las otras puertas lo veia,
// porque todas miraban el numero.
//
// Y no lo habria visto ningun test de superficie con el corpus de prueba
// tampoco: hace falta pedir la pantalla CON corpus y leer lo que dice al lado
// del numero, que es justo lo que nadie hace.
func TestUnaCifraConDatoNoDiceQueNoHayDato(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	_, cuerpo := pedir(t, s, "/hoy")
	sinCorpus := rotulo("es", "pantalla.hoy.cifra.sin_corpus")
	sinDato := rotulo("es", "pantalla.hoy.cifra.sin_dato")

	conNumero := 0
	for _, c := range cifrasDe(t, cuerpo) {
		if !strings.Contains(c, `<span class="n">`) {
			continue
		}
		conNumero++
		if strings.Contains(c, sinCorpus) {
			t.Errorf("una cifra trae numero y ademas dice que no hay corpus instalado. "+
				"La pantalla se contradice a si misma en dos lineas seguidas.\n"+
				"--- cifra ---\n%s", c)
		}
		if strings.Contains(c, sinDato) {
			t.Errorf("una cifra trae numero y ademas se rotula 'sin dato'.\n"+
				"--- cifra ---\n%s", c)
		}
	}
	if conNumero == 0 {
		t.Fatal("ninguna cifra trae numero con el corpus de demostracion: esta puerta " +
			"estaria recorriendo la rama que no le toca")
	}
	// La direccion contraria: SIN corpus, ninguna cifra pinta un matiz que solo
	// tiene sentido con el numero delante.
	s2, _ := superficie(t, nil)
	_, vacio := pedir(t, s2, "/hoy")
	for _, c := range cifrasDe(t, vacio) {
		if strings.Contains(c, rotulo("es", "pantalla.hoy.cifra.sin_responder")) ||
			strings.Contains(c, rotulo("es", "pantalla.hoy.vence_semana.esperando")) {
			t.Errorf("sin corpus, una cifra pinta un matiz que solo se lee con un numero "+
				"delante.\n--- cifra ---\n%s", c)
		}
	}
}

// EL PANEL SE CALCULA EN CADA PETICION, con el reloj de la peticion.
//
// Es el mismo fallo natural que ya tiene su puerta en el vigilante: lo comodo es
// meter estos numeros en el modelo derivado, que se calcula una vez al arrancar.
// Con eso, un servidor levantado tres semanas seguiria diciendo que algo vence
// esta semana mucho despues de que hubiera vencido.
//
// Se mueve el reloj alrededor de una fecha fija que pone la propia norma del
// paquete de prueba: antes, esta en la semana; despues, esta en los pasados.
func TestElPanelSeCalculaConElRelojDeLaPeticion(t *testing.T) {
	// El vencimiento del paquete de prueba: 2020-06-30 23:59:59Z.
	vence := time.Date(2020, 6, 30, 23, 59, 59, 0, time.UTC)
	antes := vence.Add(-48 * time.Hour)
	despues := vence.Add(30 * 24 * time.Hour)

	ps := []*corpus.Paquete{paqueteVencido()}

	s, _ := superficie(t, ps, relojFijo(antes))
	_, cuerpo := pedir(t, s, "/hoy")
	if n := numeroDeLaCifra(t, cuerpo, "pantalla.hoy.cifra.vence_semana"); n != 1 {
		t.Errorf("dos dias antes del vencimiento, 'vence esta semana' dice %d y esperaba 1", n)
	}
	if n := numeroDeLaCifra(t, cuerpo, "pantalla.hoy.cifra.sin_constancia"); n != 0 {
		t.Errorf("dos dias antes del vencimiento, 'sin constancia' dice %d y esperaba 0. "+
			"Acusar de no haber hecho algo que todavia no toca es acusar en falso", n)
	}

	s2, _ := superficie(t, ps, relojFijo(despues))
	_, cuerpo2 := pedir(t, s2, "/hoy")
	if n := numeroDeLaCifra(t, cuerpo2, "pantalla.hoy.cifra.vence_semana"); n != 0 {
		t.Errorf("un mes despues, 'vence esta semana' dice %d y esperaba 0", n)
	}
	if n := numeroDeLaCifra(t, cuerpo2, "pantalla.hoy.cifra.sin_constancia"); n != 1 {
		t.Errorf("un mes despues, 'sin constancia' dice %d y esperaba 1", n)
	}
}

// El vigilante del planificador NO se ha ido de Hoy al convertirla en panel.
//
// Es lo unico de esta pantalla que puede estar diciendo "tus plazos no se estan
// mirando", y el riesgo de anadir cuatro cifras encima es exactamente ese: que
// se quede debajo de todo y no lo lea nadie, o que se caiga en la reforma.
func TestElPanelNoSeLlevaPorDelanteAlVigilante(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	_, cuerpo := pedir(t, s, "/hoy")
	exige(t, cuerpo,
		`class="vigilancia n-`,
		rotulo("es", "pantalla.hoy.planificador"),
		rotulo("es", "pantalla.hoy.canal"),
		rotulo("es", "pantalla.hoy.plazos"),
	)
}

// ---------------------------------------------------------------------------
// Utillaje
// ---------------------------------------------------------------------------

// tarjetaDeLaCifra devuelve el <li> de la cifra rotulada con esa clave.
func tarjetaDeLaCifra(t *testing.T, cuerpo, clave string) string {
	t.Helper()
	rot := rotulo("es", clave)
	for _, c := range cifrasDe(t, cuerpo) {
		if strings.Contains(c, rot) {
			return c
		}
	}
	t.Fatalf("no hay ninguna cifra rotulada %q en el panel", clave)
	return ""
}

// numeroDeLaCifra devuelve el numero de esa cifra, o falla si no lleva numero.
func numeroDeLaCifra(t *testing.T, cuerpo, clave string) int {
	t.Helper()
	tarjeta := tarjetaDeLaCifra(t, cuerpo, clave)
	m := regexp.MustCompile(`<span class="n">(\d+)</span>`).FindStringSubmatch(tarjeta)
	if m == nil {
		t.Fatalf("la cifra %q no lleva numero.\n--- tarjeta ---\n%s", clave, tarjeta)
	}
	n := 0
	for _, c := range m[1] {
		n = n*10 + int(c-'0')
	}
	return n
}

// seccionConID recorta la seccion del panel que abre una cifra.
func seccionConID(t *testing.T, cuerpo, id string) string {
	t.Helper()
	i := strings.Index(cuerpo, `id="`+id+`"`)
	if i < 0 {
		t.Fatalf("la pagina no trae la seccion %q", id)
	}
	fin := strings.Index(cuerpo[i:], "</section>")
	if fin < 0 {
		t.Fatalf("la seccion %q no cierra", id)
	}
	return cuerpo[i : i+fin]
}
