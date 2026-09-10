package main

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/pantalla"
	"github.com/marcosmatalab/plazum/superficies/pantallas"
)

// EL TERMINAL Y LA PANTALLA DICEN LOS MISMOS NUMEROS.
//
// # Que se afirma, y por que aqui
//
// `cmd/plazum/sentadas.go` imprime las sentadas por terminal y
// `superficies/pantallas` las pinta en Hoy. Los dos leen `cal.Ciclos`, o sea que
// NO hay dos cuentas. Eso es facil de decir y no lo comprobaba nadie: el dia que
// alguno de los dos sume por su cuenta —porque el otro campo «era mas comodo»—
// las dos salidas empezarian a discrepar y no habria forma de saber cual miente.
//
// Este test vive en `cmd/plazum` porque es el unico paquete que puede llamar a
// `imprimirSentadas` (es `package main`, no se puede importar) Y ademas importar
// la superficie. Es la misma razon por la que el trinquete de alcanzabilidad
// vive aqui.
//
// # LO QUE NO COMPARA, dicho
//
// Las PALABRAS. El terminal escribe «obligaciones periodicas» en castellano
// cableado y la pantalla lo saca del catalogo en dos idiomas; exigir que
// coincidan seria atar la pantalla al terminal, que es justo al reves de como
// tiene que ir. Lo que se compara son los NUMEROS, que es donde una segunda
// cuenta se delataria.
//
// El terminal no pasa por el catalogo y esa es su deuda propia, anotada en
// docs/pendientes.md: aqui no se arregla y no se disimula.
//
// # POR QUE UN DOBLE SIN PREGUNTAS Y NO EL CORPUS INSTALADO
//
// Porque los dos lados tienen que mirar EL MISMO ALCANCE o el cuadre compara
// otra cosa. El terminal recibe un calendario ya derivado; la pantalla deriva el
// suyo con la aplicabilidad que sale de LA ENTREVISTA. Sobre el corpus real esas
// dos no coinciden, y la primera version de este test lo destapo en su primera
// ejecucion: **130 obligaciones periodicas con todo aplicable y 126 con la
// entrevista sin responder** (medido el 11-09-2026), porque cuatro periodicas
// cuelgan de una pregunta. No es un defecto del producto y por eso no se
// arregla ahi: era el arnes comparando dos alcances.
//
// Con un paquete SIN PREGUNTAS los dos alcances coinciden por construccion, y
// entonces lo unico que puede hacer discrepar a las dos salidas es que alguna
// cuente por su cuenta, que es exactamente lo que este test existe para
// descartar.

var reCifra = regexp.MustCompile(`\b\d+\b`)

func TestElTerminalYLaPantallaCuentanLasMismasSentadas(t *testing.T) {
	ps := []*corpus.Paquete{paqueteDeRitmos()}
	ahora := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	cal := pantalla.Derivar12Meses(ps, pantalla.TodoAplica, nil, ahora)
	if len(cal.Ciclos) < 3 {
		t.Fatalf("el doble da %d ciclos y tiene que dar varios: este cuadre estaria "+
			"comparando el vacio", len(cal.Ciclos))
	}

	// 1. EL TERMINAL.
	var b strings.Builder
	imprimirSentadas(&b, cal, false)
	salida := b.String()
	if !strings.Contains(salida, "LAS SENTADAS") {
		t.Fatalf("el terminal no ha impreso la seccion:\n%s", salida)
	}

	// 2. LOS NUMEROS QUE LOS DOS TIENEN QUE DECIR, sacados del calendario y no
	//    escritos aqui: un numero escrito al lado del test es la tercera cuenta.
	quiero := map[string]int{
		"obligaciones periodicas": cal.ObligacionesEnCiclo(),
		"marcos":                  cal.MarcosEnCiclo(),
		"sentadas":                cal.Sentadas(),
	}
	for que, n := range quiero {
		if n == 0 {
			// Un cero no se puede buscar en el texto sin cazar cualquier otro
			// cero. Se salta CON SU MOTIVO en vez de dar el caso por comprobado.
			t.Logf("%s vale 0 y no se contrasta por texto: "+
				"buscar un cero cazaria cualquier otro numero de la salida", que)
			continue
		}
		if !cifraEnTexto(salida, n) {
			t.Errorf("el terminal no dice %d %s, y es lo que dice el calendario:\n%s",
				n, que, salida)
		}
	}

	// 3. Y CADA CICLO, con su cadencia y su cuenta de obligaciones.
	for _, c := range cal.Ciclos {
		if !strings.Contains(salida, c.Cadencia) {
			t.Errorf("el terminal no nombra el ciclo %s", c.Cadencia)
		}
	}

	// 4. LA PANTALLA, contra los MISMOS numeros y sobre el HTML servido de
	//    verdad. Se compara la salida y no una estructura intermedia a
	//    proposito: lo que hay que descartar es que las dos superficies cuenten
	//    por su cuenta, y eso solo se ve en lo que cada una publica.
	//
	//    CADA NUMERO SE BUSCA EN SU PARRAFO Y NO EN LA SECCION ENTERA, y esto lo
	//    dijo una mutacion. La primera version buscaba en todo el bloque, y la
	//    M3 (`Periodicas` sustituido por una cuenta falsa) la dejo VERDE: el
	//    numero correcto seguia apareciendo, pero en otro sitio, porque la frase
	//    de «N de tus M con reloj» lleva dos numeros y uno de ellos casaba. Un
	//    contraste que solo exige que la cifra este en alguna parte aprueba la
	//    cifra puesta en el sitio equivocado, que en una pantalla es exactamente
	//    igual de falso.
	seccion := seccionDeSentadasDeHoy(t, ps, ahora)
	titular := parrafoDeClase(t, seccion, "titular")
	alcance := parrafoDeClase(t, seccion, "alcance")
	if !cifraEnTexto(titular, cal.ObligacionesEnCiclo()) {
		t.Errorf("el titular de la pantalla no dice %d obligaciones periodicas, y es lo "+
			"que cuenta el terminal:\n%s", cal.ObligacionesEnCiclo(), titular)
	}
	if !cifraEnTexto(titular, cal.MarcosEnCiclo()) {
		t.Errorf("el titular de la pantalla no dice %d marcos, y es lo que cuenta el "+
			"terminal:\n%s", cal.MarcosEnCiclo(), titular)
	}
	if !cifraEnTexto(alcance, cal.ObligacionesEnCiclo()) {
		t.Errorf("la frase de lo que agrupa no dice %d periodicas:\n%s",
			cal.ObligacionesEnCiclo(), alcance)
	}
	// Y CADA CICLO, con su cadencia y su cuenta EN SU PROPIA FICHA.
	for i, c := range cal.Ciclos {
		if !strings.Contains(seccion, c.Cadencia) {
			t.Errorf("la pantalla no nombra el ciclo %s y el terminal si", c.Cadencia)
			continue
		}
		ficha := fichaDeCiclo(t, seccion, i)
		if !cifraEnTexto(ficha, c.Obligaciones) {
			t.Errorf("la ficha del ciclo %s no dice sus %d obligaciones:\n%s",
				c.Cadencia, c.Obligaciones, ficha)
		}
	}
	t.Logf("cuadre terminal/pantalla: %d periodicas, %d marcos, "+
		"%d ciclos, %d sentadas", cal.ObligacionesEnCiclo(), cal.MarcosEnCiclo(),
		len(cal.Ciclos), cal.Sentadas())
}

// seccionDeSentadasDeHoy monta la superficie como la monta el producto y
// devuelve el trozo de <section> de las sentadas.
//
// SE RECORTA LA SECCION Y SE PARA SI NO CASA: buscar los numeros en la pagina
// entera haria que este cuadre se satisficiera con lo que pinta cualquier otra
// seccion de Hoy, que ademas cuenta obligaciones.
func seccionDeSentadasDeHoy(t *testing.T, ps []*corpus.Paquete, ahora time.Time) string {
	t.Helper()
	s, err := pantallas.Nuevo(pantallas.Opciones{
		Paquetes: ps, Catalogo: catDePrueba(t),
		Ahora: func() time.Time { return ahora },
	})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/hoy", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("/hoy contesta %d", w.Code)
	}
	cuerpo := w.Body.String()
	i := strings.Index(cuerpo, `id="sentadas"`)
	if i < 0 {
		t.Fatalf("Hoy no pinta la seccion de sentadas, asi que "+
			"este cuadre no compara nada:\n%s", cuerpo)
	}
	j := strings.Index(cuerpo[i:], "</section>")
	if j < 0 {
		t.Fatal("la seccion de sentadas no se cierra: el recorte no puede casar")
	}
	return cuerpo[i : i+j]
}

// EL CONTROL NEGATIVO DEL CONTRASTE. Si `cifraEnTexto` casara por subcadena, un
// «13» encontraria el «3» y este cuadre aprobaria numeros que no estan.
func TestElContrasteDeCifrasNoCasaPorSubcadena(t *testing.T) {
	const texto = "LAS SENTADAS: 130 obligaciones periodicas de 14 marcos en 5 sentadas"
	for _, n := range []int{130, 14, 5} {
		if !cifraEnTexto(texto, n) {
			t.Errorf("no encuentra %d y esta escrito", n)
		}
	}
	for _, n := range []int{13, 3, 1, 0, 4} {
		if cifraEnTexto(texto, n) {
			t.Errorf("encuentra %d y NO esta: esta casando por subcadena, asi que el "+
				"cuadre aprobaria numeros que la salida no dice", n)
		}
	}
}

func cifraEnTexto(s string, n int) bool {
	q := strconv.Itoa(n)
	for _, m := range reCifra.FindAllString(s, -1) {
		if m == q {
			return true
		}
	}
	return false
}

// paqueteDeRitmos son periodicas de varias cadencias y SIN NINGUNA PREGUNTA.
//
// Sin preguntas, toda obligacion aplica sin condiciones, asi que la
// aplicabilidad que deriva la entrevista coincide con `TodoAplica` y los dos
// lados del cuadre miran el mismo alcance. Es el doble que hace que este test
// afirme lo que dice que afirma.
func paqueteDeRitmos() *corpus.Paquete {
	fija := corpus.RegimenSpec{Computo: "naturales", Cierre: "fin_de_dia"}
	obl := func(id, cad string) corpus.Obligacion {
		return corpus.Obligacion{
			ID: id, Articulo: "1", Cita: "demo cuadre art. 1",
			Vigencia: corpus.Vigencia{Desde: "2020-01-01"}, ClaseE2E: "procedimental",
			Temporalidad: &corpus.Temporalidad{
				Primitiva: "periodica", Hito: id, Cadencia: cad, Regimen: fija,
				OrigenDelIntervalo:        "propuesto",
				JustificacionDelIntervalo: "Intervalo sintetico de un doble de prueba.",
			},
		}
	}
	return &corpus.Paquete{
		URN: "urn:demo:cuadre", Version: "2026.1", Clase: corpus.Propio,
		Licencia:       "Apache-2.0",
		Identificador:  corpus.Identificador{Tipo: corpus.ELIUE, Valor: "reg/9999/4/oj"},
		LicenciaFuente: corpus.DelProyecto,
		Atribucion:     "Paquete sintetico de demostracion. Sin tercero con derechos.",
		Vigencia:       corpus.Vigencia{Desde: "2020-01-01"},
		Obligaciones: []corpus.Obligacion{
			obl("cuadre.o.anual.a", "P12M"),
			obl("cuadre.o.anual.b", "P12M"),
			obl("cuadre.o.semestral", "P6M"),
			obl("cuadre.o.trimestral", "P3M"),
		},
	}
}

// parrafoDeClase saca el <p class="X"> de un trozo de HTML, y para si no casa:
// un recorte que devolviera la seccion entera convertiria este cuadre en el que
// la mutacion M3 dejo verde.
func parrafoDeClase(t *testing.T, html, clase string) string {
	t.Helper()
	marca := `<p class="` + clase + `">`
	i := strings.Index(html, marca)
	if i < 0 {
		t.Fatalf("la seccion no trae ningun <p class=%q>:\n%s", clase, html)
	}
	resto := html[i+len(marca):]
	j := strings.Index(resto, "</p>")
	if j < 0 {
		t.Fatalf("el parrafo %q no se cierra", clase)
	}
	return resto[:j]
}

// fichaDeCiclo saca el i-esimo <li class="ciclo">.
func fichaDeCiclo(t *testing.T, html string, i int) string {
	t.Helper()
	const marca = `<li class="ciclo">`
	resto := html
	for n := 0; ; n++ {
		k := strings.Index(resto, marca)
		if k < 0 {
			t.Fatalf("la seccion no trae la ficha de ciclo numero %d", i)
		}
		resto = resto[k+len(marca):]
		fin := strings.Index(resto, "</li>")
		if fin < 0 {
			t.Fatal("una ficha de ciclo no se cierra")
		}
		if n == i {
			return resto[:fin]
		}
	}
}
