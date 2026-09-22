package plazum

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

// EL PLAN TIENE QUE PODER LEERSE COMO PLAN.
//
// # El riesgo, que no es de forma
//
// El 08-09-2026 `docs/ETAPAS.md` pesaba **121.327 bytes** y el 81 % de sus caracteres
// vivia dentro de las lineas de casilla: 143 casillas con una mediana de 315
// caracteres, 32 por encima de mil y una de **6.063**, con seis correcciones
// fechadas dentro. Quien lo abre puede leer «que rigor» o «aqui hay alguien que
// no cierra nada», y la frontera entre las dos lecturas es mas fina de lo que
// parece.
//
// **La solucion no es quitar rigor, es sacarlo del camino.** El porque largo vive
// en `docs/casillas.md`, entero y sin tocar, y la casilla se queda con una linea y
// su enlace. Despues de la tijera: 50.239 bytes, mediana 148, maximo 573.
//
// # Por que esta puerta no es de las que entrenan a esquivarlas
//
// La regla de la casa dice que una puerta que salta SIEMPRE no protege. Esta
// salta cuando alguien escribe una casilla larga, que es exactamente el
// comportamiento que se quiere cortar, y su arreglo es mover el parrafo al
// archivo, que cuesta un minuto y no pierde una palabra. No hay escape ni hace
// falta: el escape seria el sitio por donde volveria a entrar.
//
// # Y la segunda direccion es la que hace que el archivo no sea un cajon
//
// Un archivo al que se tira lo que estorba y del que nadie vuelve es peor que una
// casilla larga, porque ademas parece orden. Por eso la puerta recorre las DOS
// direcciones: toda casilla que enlaza tiene su ancla, y **toda ancla tiene su
// casilla**. La que falta es siempre la segunda.

const (
	rutaDelArchivoDeCasillas = "docs/casillas.md"

	// TechoDeLaCasilla es el maximo de caracteres de una linea de casilla.
	//
	// De donde sale el numero: la mediana de las 143 casillas del 08-09-2026 era
	// 315 y el corte se hizo por encima de 600, que deja pasar las que ya cabian
	// y saca las 50 que no. No es una cifra redonda elegida a ojo: es la que
	// separa «un titulo con su matiz» de «un razonamiento dentro de un plan».
	TechoDeLaCasilla = 600
)

var (
	reCasillaEntera = regexp.MustCompile(`(?m)^- \[[ x]\] (.*)$`)
	// LA RUTA ES RELATIVA A `docs/ETAPAS.md`, QUE ES DONDE SE ESCRIBE EL ENLACE.
	//
	// Decia `docs/casillas.md#`, o sea la ruta desde la RAIZ, y ahi estuvo el
	// problema: cuando el plan se mudo a `docs/` los 54 enlaces no se
	// reescribieron, en GitHub resolvian a `docs/docs/casillas.md`, y esta puerta
	// no podia verlo porque estaba casando contra la forma ROTA. Comparaba
	// ANCLAS, y el ancla existia en el fichero correcto: las dos direcciones
	// cuadraban mientras el enlace no llevaba a ningun sitio.
	//
	// No se acepta la forma vieja como alternativa a proposito. Tolerar las dos
	// es como sobrevive la rota. Que la ruta RESUELVA lo vigila
	// `TestTodoEnlaceRelativoDeUnDocumentoResuelve`, que es una puerta distinta y
	// mira otra cosa: esta comprueba el ancla, aquella el fichero.
	reEnlaceAlArchivo = regexp.MustCompile(
		`\(\[por qué\]\(casillas\.md#([a-z0-9-]+)\)\)`)
	reAnclaDelArchivo = regexp.MustCompile(`(?m)^<a id="([a-z0-9-]+)"></a>$`)
)

func leerArchivoDeCasillas(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(rutaDelArchivoDeCasillas) // #nosec G304 -- ruta constante
	if err != nil {
		t.Fatalf("no puedo leer %s: %v", rutaDelArchivoDeCasillas, err)
	}
	return string(b)
}

func TestNingunaCasillaDelPlanSeComeElPlan(t *testing.T) {
	texto := leerEtapas(t)
	casillas := reCasillaEntera.FindAllStringSubmatch(texto, -1)
	if len(casillas) < 100 {
		t.Fatalf("docs/ETAPAS.md trae %d casillas y hoy son mas de cien: el patron ha dejado "+
			"de casar y esta puerta estaria midiendo el vacio", len(casillas))
	}

	var largas []string
	for _, c := range casillas {
		if n := len([]rune(c[0])); n > TechoDeLaCasilla {
			largas = append(largas, fmt.Sprintf("    %d caracteres: %s...", n,
				recortarCasilla(c[1], 90)))
		}
	}
	if len(largas) > 0 {
		t.Errorf("%d casillas de docs/ETAPAS.md pasan de %d caracteres:\n%s\n"+
			"  El plan se lee para saber por donde va el proyecto, y una casilla con el "+
			"razonamiento dentro deja de poder leerse como plan.\n"+
			"  Arreglo, que no pierde una palabra: el porque largo va a %s con su ancla, y "+
			"la casilla se queda con su linea y el enlace ([por qué](...)).",
			len(largas), TechoDeLaCasilla, strings.Join(largas, "\n"),
			rutaDelArchivoDeCasillas)
	}
	t.Logf("%d casillas, ninguna por encima de %d caracteres", len(casillas),
		TechoDeLaCasilla)
}

func TestElArchivoDeCasillasYElPlanSeApuntanEnLasDosDirecciones(t *testing.T) {
	texto := leerEtapas(t)
	archivo := leerArchivoDeCasillas(t)

	enlazadas := map[string]bool{}
	for _, m := range reEnlaceAlArchivo.FindAllStringSubmatch(texto, -1) {
		enlazadas[m[1]] = true
	}
	ancladas := map[string]bool{}
	for _, m := range reAnclaDelArchivo.FindAllStringSubmatch(archivo, -1) {
		ancladas[m[1]] = true
	}
	if len(enlazadas) == 0 || len(ancladas) == 0 {
		t.Fatalf("el plan enlaza %d anclas y el archivo declara %d: con cero en cualquiera "+
			"de los dos lados esta puerta esta midiendo el vacio, no la coherencia",
			len(enlazadas), len(ancladas))
	}

	// DIRECCION 1: un enlace del plan que no lleva a ningun sitio.
	for _, a := range ordenadas(enlazadas) {
		if !ancladas[a] {
			t.Errorf("docs/ETAPAS.md enlaza a %s#%s y ese ancla no existe.\n"+
				"  Un enlace roto en el plan es peor que no tenerlo: quien lo siga se lleva "+
				"la impresion de que el porque esta escrito y no lo encuentra.",
				rutaDelArchivoDeCasillas, a)
		}
	}
	// DIRECCION 2, la que hace que el archivo no sea un cajon.
	for _, a := range ordenadas(ancladas) {
		if !enlazadas[a] {
			t.Errorf("%s declara el ancla %q y ninguna casilla de docs/ETAPAS.md la enlaza.\n"+
				"  Es la mitad que convierte un archivo en un cajon: texto que salio del "+
				"plan y del que ya no vuelve nadie.\n"+
				"  Arreglo: o la casilla lo enlaza, o el bloque sale del archivo.",
				rutaDelArchivoDeCasillas, a)
		}
	}
	t.Logf("%d anclas, enlazadas en las dos direcciones", len(ancladas))
}

// CONTROL NEGATIVO, en las dos direcciones y sobre los dos patrones.
//
// El fallo probable de esta familia es el de siempre: un patron que casa de mas.
// `reCasillaEntera` tiene que anclar en columna cero (este repositorio esta lleno
// de prosa que cita casillas), y `reEnlaceAlArchivo` no puede casar un enlace
// cualquiera a `docs/casillas.md` que no sea el del pie de una casilla.
func TestLosPatronesDelArchivoDeCasillasNoCasanDeMas(t *testing.T) {
	muestra := "- [x] una casilla corta\n" +
		"  - [ ] una anidada, que NO es una casilla del plan\n" +
		"un parrafo que menciona - [x] dentro de una frase\n"
	if got := len(reCasillaEntera.FindAllString(muestra, -1)); got != 1 {
		t.Errorf("el patron de casilla ha visto %d y es 1: esta cazando casillas que viven "+
			"dentro de la prosa de otra", got)
	}

	casos := []struct {
		nombre string
		fuente string
		quiero string
	}{
		{"el pie de una casilla", "- [x] algo ([por qué](casillas.md#el-ancla))",
			"el-ancla"},
		{"una mencion en prosa no vale",
			"ver casillas.md#el-ancla para el detalle", ""},
		{"un enlace sin el rotulo tampoco",
			"- [x] algo ([detalle](casillas.md#el-ancla))", ""},
		// LA FORMA ROTA NO CUENTA, y este caso es el que faltaba. `docs/casillas.md`
		// escrito DESDE `docs/` resuelve a `docs/docs/casillas.md`, que no existe.
		// Sobrevivio 54 veces porque esta puerta casaba justo contra esa forma.
		{"la ruta desde la raiz, que desde docs/ esta rota",
			"- [x] algo ([por qué](docs/casillas.md#el-ancla))", ""},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			m := reEnlaceAlArchivo.FindStringSubmatch(c.fuente)
			got := ""
			if m != nil {
				got = m[1]
			}
			if got != c.quiero {
				t.Errorf("ha sacado %q y esperaba %q: la puerta estaria contando enlaces "+
					"que no son el pie de una casilla", got, c.quiero)
			}
		})
	}

	// Y el ancla, que es lo que sostiene la direccion 2.
	if m := reAnclaDelArchivo.FindStringSubmatch("<a id=\"uno-dos\"></a>\n"); m == nil ||
		m[1] != "uno-dos" {
		t.Errorf("el patron de ancla no reconoce la forma que el archivo escribe: %v", m)
	}
	if reAnclaDelArchivo.MatchString("texto <a id=\"uno\"></a> en medio de una linea\n") {
		t.Error("el patron de ancla casa dentro de una linea: entonces cualquier mencion " +
			"cuenta como ancla y la direccion 2 se cumple sola")
	}
}

func recortarCasilla(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// EstadosDeCasillaQueSeCUENTAN son las dos unicas marcas admitidas.
//
// La lista va escrita aqui y no derivada del fichero, que es lo que la hace
// util: una lista derivada aceptaria la marca nueva el dia que alguien la
// escriba, que es justo el dia en que hay que enterarse.
var EstadosDeCasillaQueSeCuentan = []string{" ", "x"}

// reMarcaDeCasilla casa CUALQUIER marca, no solo las validas. Es la diferencia
// con `reCasillaEntera`, y es el punto entero de esta puerta.
var reMarcaDeCasilla = regexp.MustCompile(`(?m)^- \[(.)\] `)

// NINGUNA CASILLA LLEVA UNA MARCA QUE NO CUENTE NADIE.
//
// # El agujero, medido el 22-09-2026
//
// `docs/ETAPAS.md` tenia DOS casillas marcadas `[~]`, y `[~]` no lo miraba
// ninguna puerta de este repositorio:
//
//	estado_del_plan_test.go   cuenta `^- \[x\] ` y `^- \[ \] `   -> no las suma
//	casillas_test.go          casa `^- \[[ x]\] `                -> no las mide
//
// O sea que eran dos casillas del plan que **no entraban en el total, no
// entraban en las cerradas, y no pasaban por el techo de longitud**. Las dos
// tenian mas de 1.300 caracteres, que es el doble del techo, y llevaban ahi
// meses. Y las dos afirmaban cosas que habian dejado de ser ciertas, que es
// exactamente lo que el techo existe para que no pase: una casilla larga deja de
// leerse como casilla y pasa a ser un sitio donde la prosa envejece sin que
// nadie la relea.
//
// **El total publicado cuadraba igualmente**, y eso es lo que hizo invisible el
// agujero: 78 cerradas mas 66 abiertas daban los 144 que el documento declaraba.
// Una tercera marca no descuadra nada, simplemente no esta.
//
// # Por que una marca nueva es un agujero y no una comodidad
//
// Porque lo que una marca intermedia dice de verdad es «esto no esta hecho pero
// tampoco quiero contarlo como pendiente», y esa es una casilla que no aparece
// en ninguna de las dos cifras del estado. El plan tiene DOS numeros a proposito
// para que ninguno mienta solo; un tercer estado los hace mentir a los dos a la
// vez.
func TestNingunaCasillaLlevaUnaMarcaQueNadieCuenta(t *testing.T) {
	texto := leerEtapas(t)
	marcas := reMarcaDeCasilla.FindAllStringSubmatch(texto, -1)
	if len(marcas) < 100 {
		t.Fatalf("docs/ETAPAS.md trae %d casillas y hoy son mas de cien: el patron ha "+
			"dejado de casar y esto esta midiendo el vacio", len(marcas))
	}
	raras := map[string]int{}
	for _, m := range marcas {
		if !contieneMarca(EstadosDeCasillaQueSeCuentan, m[1]) {
			raras[m[1]]++
		}
	}
	for marca, n := range raras {
		t.Errorf(`%d casilla(s) llevan la marca "[%s]", que no la cuenta ninguna puerta.

  Las dos cifras del estado salen de contar "[x]" y "[ ]". Una marca distinta no
  descuadra el total, simplemente NO ESTA: ni en las cerradas, ni en las
  abiertas, ni en el techo de longitud de la casilla.

  Eso ya paso: el 22-09-2026 habia dos casillas en "[~]", las dos con mas del
  doble del techo de caracteres, y las dos afirmando cosas que habian dejado de
  ser ciertas.

  Arreglo: decidir. Si esta hecha, "[x]"; si no, "[ ]". Lo que no cabe es una
  tercera respuesta que no cuenta nadie.`, n, marca)
	}
	t.Logf("%d casillas, %d con marca no contada", len(marcas), len(raras))
}

// EL CONTROL NEGATIVO del detector de marcas.
//
// Sus dos fallos probables son opuestos. Si casa de menos (por ejemplo, solo
// `[ ]` y `[x]`) es el agujero de siempre con otra cara: la marca rara no
// aparece y la puerta da verde. Si casa de mas, acusa a cualquier linea de lista
// del documento y la puerta salta sobre trabajo legitimo.
func TestElDetectorDeMarcasDeCasillaAcusaYSeCalla(t *testing.T) {
	for _, c := range []struct {
		nombre string
		fuente string
		marca  string
		casa   bool
	}{
		{"cerrada", "- [x] algo", "x", true},
		{"abierta", "- [ ] algo", " ", true},
		{"la marca que se colo", "- [~] algo", "~", true},
		{"una mayuscula tampoco cuenta", "- [X] algo", "X", true},
		{"un guion", "- [-] algo", "-", true},
		{"una lista normal no es una casilla", "- algo", "", false},
		{"sin espacio detras tampoco", "- [x]algo", "", false},
		{"en medio de una linea no", "texto - [x] algo", "", false},
	} {
		m := reMarcaDeCasilla.FindStringSubmatch(c.fuente)
		if (m != nil) != c.casa {
			t.Errorf("%s: casa=%v y esperaba %v en %q", c.nombre, m != nil, c.casa, c.fuente)
			continue
		}
		if c.casa && m[1] != c.marca {
			t.Errorf("%s: saca la marca %q y esperaba %q", c.nombre, m[1], c.marca)
		}
	}
	// Y LAS DOS VALIDAS SON LAS DOS, ni una mas: si alguien anade "~" a la
	// lista para que deje de acusar, este caso lo dice.
	if len(EstadosDeCasillaQueSeCuentan) != 2 ||
		!contieneMarca(EstadosDeCasillaQueSeCuentan, " ") ||
		!contieneMarca(EstadosDeCasillaQueSeCuentan, "x") {
		t.Errorf("las marcas admitidas son %v y tienen que ser exactamente \" \" y \"x\".\n"+
			"  Anadir una tercera aqui no arregla el agujero: lo legaliza.",
			EstadosDeCasillaQueSeCuentan)
	}
}

func contieneMarca(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
