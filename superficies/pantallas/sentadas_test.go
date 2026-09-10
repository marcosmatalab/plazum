package pantallas

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/pantalla"
)

// EL CUADRE DE LAS SENTADAS: la pantalla no cuenta, LEE.
//
// # Que se afirma y por que hace falta afirmarlo
//
// La seccion de sentadas es trasvase: todo lo que pinta sale de `cal.Ciclos`,
// que ya calculo `nucleo/pantalla`. Eso es facil de decir y facil de romper sin
// que nadie lo note, porque una suma escrita en la superficie da un numero
// plausible: si alguien contara aqui las obligaciones periodicas recorriendo las
// fechas en vez de leer `ObligacionesEnCiclo()`, una obligacion trimestral
// contaria cuatro veces y el numero seguiria pareciendo razonable.
//
// Por eso el cuadre compara contra el CALENDARIO y no contra una constante: lo
// que se comprueba no es que salga 5, es que salga lo mismo que dice quien lo
// calculo.

// TestLasSentadasDeLaPantallaSonLasDelCalendario es el cuadre.
func TestLasSentadasDeLaPantallaSonLasDelCalendario(t *testing.T) {
	ps := []*corpus.Paquete{paqueteConRitmos()}
	ahora := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	cal := pantalla.Derivar12Meses(ps, pantalla.TodoAplica, nil, ahora)
	if len(cal.Ciclos) < 3 {
		t.Fatalf("el doble da %d ciclos y tiene que dar varios: este cuadre estaria "+
			"comparando el vacio", len(cal.Ciclos))
	}
	v := sentadasDe(cal)

	if !v.Hay {
		t.Fatal("con ciclos en el calendario, la vista dice que no hay sentadas")
	}
	if v.Periodicas != cal.ObligacionesEnCiclo() {
		t.Errorf("la vista dice %d periodicas y el calendario dice %d",
			v.Periodicas, cal.ObligacionesEnCiclo())
	}
	if v.Marcos != cal.MarcosEnCiclo() {
		t.Errorf("la vista dice %d marcos y el calendario dice %d", v.Marcos, cal.MarcosEnCiclo())
	}
	if v.Sentadas != cal.Sentadas() {
		t.Errorf("la vista dice %d sentadas y el calendario dice %d", v.Sentadas, cal.Sentadas())
	}
	if v.ConReloj != len(cal.Destinos) {
		t.Errorf("la vista dice %d obligaciones con reloj y el calendario reparte %d destinos.\n"+
			"  El denominador de «lo que agrupa y lo que no» tiene que ser el conjunto que\n"+
			"  el calendario reparte, no el total del corpus: 556 obligaciones no son 556\n"+
			"  relojes, y usar el total haria que la fraccion pareciera mucho menor de lo\n"+
			"  que es", v.ConReloj, len(cal.Destinos))
	}
	if len(v.Ciclos) != len(cal.Ciclos) {
		t.Fatalf("la vista trae %d ciclos y el calendario %d", len(v.Ciclos), len(cal.Ciclos))
	}

	// LA LEY DE CONSERVACION DE ESTA SECCION, que es lo que caza una suma
	// escrita a mano: la suma de las obligaciones de cada ciclo tiene que ser
	// exactamente el total, porque un ciclo es una particion y no un filtro.
	suma, sumaSent := 0, 0
	for i, c := range v.Ciclos {
		orig := cal.Ciclos[i]
		if c.Cadencia != orig.Cadencia || c.Obligaciones != orig.Obligaciones ||
			c.Marcos != len(orig.Marcos) || c.Sentadas != len(orig.Sentadas) ||
			c.EsperandoDato != orig.EsperandoDato || c.Alineables != orig.Alineables ||
			c.Fijas != orig.Fijas {
			t.Errorf("el ciclo %s de la vista no es el del calendario:\n  vista %+v\n  calendario %+v",
				c.Cadencia, c, orig)
		}
		suma += c.Obligaciones
		sumaSent += c.Sentadas
	}
	if suma != v.Periodicas {
		t.Errorf("los ciclos suman %d obligaciones y el total dice %d. Un ciclo es una "+
			"PARTICION: si la suma no cuadra, o hay una obligacion en dos ciclos o hay "+
			"una que no esta en ninguno", suma, v.Periodicas)
	}
	if sumaSent != v.Sentadas {
		t.Errorf("los ciclos suman %d sentadas y el total dice %d", sumaSent, v.Sentadas)
	}
	// Y LO PERIODICO NO PUEDE SER MAS QUE LO QUE TIENE RELOJ. Una fraccion por
	// encima de uno en un agregado sube el total sin que nada lo nombre, que es
	// el fallo que ya se pago una vez con la cobertura de la v1.
	if v.Periodicas > v.ConReloj {
		t.Errorf("%d periodicas de %d con reloj: la fraccion pasa de uno, que es imposible",
			v.Periodicas, v.ConReloj)
	}
	t.Logf("cuadre: %d periodicas de %d con reloj, %d marcos, %d ciclos, %d sentadas",
		v.Periodicas, v.ConReloj, v.Marcos, len(v.Ciclos), v.Sentadas)
}

// EL VALOR CERO NO PINTA NADA, en las DOS formas de la nada.
//
// Sin ciclos no hay ritmo que contar, y una seccion que dijera «0 obligaciones
// periodicas» afirmaria que no hay nada que se repita cuando lo que pasa es que
// no hay corpus. Es el invariante 8 en una seccion de pantalla.
func TestSinCiclosLaSeccionDeSentadasNoSePinta(t *testing.T) {
	for _, c := range []struct {
		que string
		ps  []*corpus.Paquete
	}{
		{"corpus nil", nil},
		{"corpus presente y vacio", []*corpus.Paquete{}},
	} {
		t.Run(c.que, func(t *testing.T) {
			cal := pantalla.Derivar12Meses(c.ps, pantalla.TodoAplica, nil,
				time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC))
			if v := sentadasDe(cal); v.Hay {
				t.Errorf("sin corpus la vista dice que hay sentadas: %+v", v)
			}
			s, _ := superficie(t, c.ps)
			_, cuerpo := pedir(t, s, "/hoy")
			if strings.Contains(cuerpo, `id="sentadas"`) {
				t.Error("sin corpus la pantalla pinta la seccion de sentadas")
			}
		})
	}
	// EL CONTROL POSITIVO: con ciclos SI se pinta. Sin esta mitad, una seccion
	// que no se pintara nunca pasaria lo de arriba con nota.
	s, _ := superficie(t, []*corpus.Paquete{paqueteConRitmos()})
	if _, cuerpo := pedir(t, s, "/hoy"); !strings.Contains(cuerpo, `id="sentadas"`) {
		t.Errorf("con ciclos la pantalla NO pinta la seccion de sentadas:\n%s", cuerpo)
	}
}

// reNumeros saca los numeros de un trozo de HTML.
var reNumeros = regexp.MustCompile(`\b\d+\b`)

// LOS NUMEROS QUE SALEN EN LA PANTALLA SON LOS DEL CALENDARIO.
//
// El cuadre de arriba compara la VISTA con el calendario; este compara el HTML
// SERVIDO con el calendario, que es la mitad que aquel no puede dar: una
// plantilla que se dejara un `{{...}}` sin pintar, o que pintara el campo
// equivocado, pasaria el primero entero.
func TestLosNumerosDeLaSeccionDeSentadasSalenEnLaPagina(t *testing.T) {
	ps := []*corpus.Paquete{paqueteConRitmos()}
	s, _ := superficie(t, ps)
	_, cuerpo := pedir(t, s, "/hoy")
	seccion := trozoDeSentadas(t, cuerpo)

	cal := pantalla.Derivar12Meses(ps, pantalla.TodoAplica, nil, s.ahora())
	v := sentadasDe(cal)

	// Los dos numeros del titular y del alcance tienen que estar.
	for _, n := range []int{v.Periodicas, v.Marcos, v.ConReloj} {
		if !contieneNumero(seccion, n) {
			t.Errorf("la seccion no ensena el numero %d.\n%s", n, seccion)
		}
	}
	// Y EL CODIGO ISO DE CADA CADENCIA, que es el dato que permite ir a mirar
	// que ritmo es sin fiarse del nombre que le hayamos puesto.
	for _, c := range v.Ciclos {
		if !strings.Contains(seccion, c.Cadencia) {
			t.Errorf("la seccion no ensena la cadencia %s", c.Cadencia)
		}
	}
}

// NINGUN NUMERO DE ESTA SECCION SE LEE COMO UN VEREDICTO.
//
// Es la frontera de la casilla: agrupar por trabajo es decir CUANTAS VECES hay
// que sentarse, no decir que algo este cumplido. Un «4 al ano» al lado de una
// lista de obligaciones se lee como un plan; el mismo numero con la palabra
// «cumplido» al lado se lee como un veredicto que nadie ha emitido.
func TestLaSeccionDeSentadasNoDiceQueNadaEsteCumplido(t *testing.T) {
	// CON EL BORRADOR DE CATALOGO Y NO CON EL DOBLE, y es la diferencia entre
	// comprobar algo y no comprobar nada: el doble devuelve «[es:la.clave]», que
	// por construccion no contiene ninguna palabra de veredicto. Un test de
	// vocabulario contra un catalogo sin palabras sale verde siempre.
	s, _ := superficie(t, []*corpus.Paquete{paqueteConRitmos()}, func(o *Opciones) {
		o.Catalogo = catEs{textos: textoEs}
	})
	_, cuerpoHoy := pedir(t, s, "/hoy")
	seccion := trozoDeSentadas(t, cuerpoHoy)
	bajo := strings.ToLower(seccion)
	// EL SUELO DEL ARNES: si el catalogo no ha puesto palabras de verdad, lo de
	// abajo no esta comprobando nada y este test seria un verde vacio.
	if strings.Contains(seccion, "FALTA(") || !strings.Contains(bajo, "obligacion") {
		t.Fatalf("la seccion sale sin texto de verdad, asi que este test no comprueba "+
			"nada:\n%s", seccion)
	}
	for _, prohibida := range []string{"cumpl", "conform", "aprobad", "satisfech", "resuelt"} {
		if strings.Contains(bajo, prohibida) {
			t.Errorf("la seccion de sentadas usa %q, que es un juicio.\n"+
				"  Esta seccion dice cuantas veces hay que sentarse, no que algo este hecho.",
				prohibida)
		}
	}
	// Y LA FRASE DE LO QUE NO AGRUPA TIENE QUE ESTAR (D-13). Sin ella, «7
	// ritmos» debajo de «N te alcanzan» se lee como que todo lo tuyo cabe en
	// siete reuniones.
	if !strings.Contains(bajo, "periodico") {
		t.Errorf("la seccion no dice que solo agrupa lo PERIODICO.\n"+
			"  Es la mitad que impide leer el numero como un plan sobre todo el corpus.\n%s",
			seccion)
	}
}

// trozoDeSentadas recorta la seccion y para si no casa: buscar en la pagina
// entera haria que estas puertas se satisficieran con lo que pinta otra seccion.
func trozoDeSentadas(t *testing.T, cuerpo string) string {
	t.Helper()
	i := strings.Index(cuerpo, `id="sentadas"`)
	if i < 0 {
		t.Fatalf("la pagina no trae la seccion de sentadas:\n%s", cuerpo)
	}
	j := strings.Index(cuerpo[i:], "</section>")
	if j < 0 {
		t.Fatal("la seccion de sentadas no se cierra: el recorte no puede casar")
	}
	return cuerpo[i : i+j]
}

func contieneNumero(s string, n int) bool {
	q := strconv.Itoa(n)
	for _, m := range reNumeros.FindAllString(s, -1) {
		if m == q {
			return true
		}
	}
	return false
}
