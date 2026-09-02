package plazum

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// LA PUERTA DEL SUBCONJUNTO DE LA v1: el porcentaje deja de contarse a mano.
//
// # De donde sale
//
// El 02-09-2026 dos recuentos del MISMO corpus dieron 123 y 124, y la diferencia
// era un paquete de frontera (`eni`, un reloj) que estaba contado dentro por uno
// y fuera por otro. Ninguno de los dos estaba equivocado: la lista de que
// paquetes forman los 12 marcos de la v1 no existia en ningun sitio como dato,
// solo como prosa en D-19. Un denominador que cada uno cuenta a mano produce
// exactamente eso.
//
// Ahora la lista es `paquetes/marcos-v1.json` y el calculo sale de ahi.
//
// # Por que un fichero de datos y no una constante en Go
//
// El invariante 2 prohibe cablear identificadores de norma en literales de
// cadena, y una lista de quince nombres de norma en Go rompe el build. Es la
// prohibicion funcionando: la lista es dato, y el dato vive bajo `paquetes/`.
//
// # Las dos direcciones, que es lo que impide que envejezca
//
// Todo paquete del arbol tiene que estar declarado dentro o fuera, y todo nombre
// declarado tiene que existir. Un paquete nuevo que nadie clasifique rompe la
// puerta, porque el silencio es como se cuela un paquete de frontera: es la
// misma forma del trinquete de alcanzabilidad.

// rutaDeMarcosV1 y rutaDelREADME se resuelven desde la raiz del repositorio,
// que es donde corre este paquete de test.
const (
	rutaDeMarcosV1 = "paquetes/marcos-v1.json"
	rutaDelREADME  = "README.md"
)

type marcoV1 struct {
	Paquete string `json:"paquete"`
	// Censados es un PUNTERO a proposito, y es el invariante 8 en un fichero de
	// datos: `null` (el censo no ha verificado este paquete) y `0` (el censo lo
	// verifico y la norma no trae ni un reloj) son cosas OPUESTAS, y con un int
	// a secas las dos llegarian como cero. El cero de `iso27001` esta contado y
	// defendido; el de `soc2` no existe.
	Censados          *int   `json:"censados"`
	Porque            string `json:"porque"`
	SinVerificarPorId string `json:"sin_verificar_porque"`
	Familia           string `json:"familia"`
	Aviso             string `json:"aviso"`
}

type fueraV1 struct {
	Paquete string `json:"paquete"`
	Porque  string `json:"porque"`
}

type declaracionV1 struct {
	Marcos []marcoV1 `json:"marcos"`
	Fuera  []fueraV1 `json:"fuera"`
}

func leerMarcosV1(t *testing.T) declaracionV1 {
	t.Helper()
	b, err := os.ReadFile(rutaDeMarcosV1) // #nosec G304 -- ruta constante del repositorio
	if err != nil {
		t.Fatalf("no puedo leer %s: %v", rutaDeMarcosV1, err)
	}
	var d declaracionV1
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatalf("%s no parsea: %v", rutaDeMarcosV1, err)
	}
	return d
}

// paquetesDelArbol enumera los directorios de paquetes/ que traen paquete.json.
// Es la misma regla que usa corpus.Cargar, para que las dos vean lo mismo.
func paquetesDelArbol(t *testing.T) []string {
	t.Helper()
	ents, err := os.ReadDir("paquetes")
	if err != nil {
		t.Fatalf("no puedo leer paquetes/: %v", err)
	}
	var out []string
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join("paquetes", e.Name(), "paquete.json")); err == nil {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

// relojesPorPaquete cuenta las obligaciones con bloque `temporalidad`, leyendo
// los paquete.json. Es el NUMERADOR.
func relojesPorPaquete(t *testing.T) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, n := range paquetesDelArbol(t) {
		b, err := os.ReadFile(filepath.Join("paquetes", n, "paquete.json")) // #nosec G304 -- recorre el arbol del repositorio
		if err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		var p struct {
			Obligaciones []struct {
				Temporalidad *json.RawMessage `json:"temporalidad"`
			} `json:"obligaciones"`
		}
		if err := json.Unmarshal(b, &p); err != nil {
			t.Fatalf("%s/paquete.json no parsea: %v", n, err)
		}
		for _, o := range p.Obligaciones {
			if o.Temporalidad != nil {
				out[n]++
			}
		}
	}
	return out
}

// TestTodoPaqueteEstaDeclaradoDentroOFueraDeLaV1 es la mitad que impide que la
// lista envejezca. Las dos direcciones.
func TestTodoPaqueteEstaDeclaradoDentroOFueraDeLaV1(t *testing.T) {
	d := leerMarcosV1(t)
	arbol := paquetesDelArbol(t)
	if len(arbol) < 30 {
		t.Fatalf("bajo paquetes/ hay %d paquetes y hoy son al menos 30: este recorrido esta "+
			"midiendo el vacio", len(arbol))
	}

	declarado := map[string]string{}
	for _, m := range d.Marcos {
		if otro, repetido := declarado[m.Paquete]; repetido {
			t.Errorf("%s sale dos veces en marcos-v1.json (ya estaba como %q)", m.Paquete, otro)
		}
		declarado[m.Paquete] = "dentro"
	}
	for _, f := range d.Fuera {
		if otro, repetido := declarado[f.Paquete]; repetido {
			t.Errorf("%s esta a la vez dentro y fuera de la v1 (%q). Una de las dos sobra, y "+
				"mientras las dos esten, el porcentaje depende de cual se lea primero",
				f.Paquete, otro)
		}
		declarado[f.Paquete] = "fuera"
		if strings.TrimSpace(f.Porque) == "" {
			t.Errorf("%s esta fuera de la v1 y no dice por que. Un paquete excluido sin motivo "+
				"es indistinguible de un olvido, que es exactamente lo que paso con `eni`",
				f.Paquete)
		}
	}

	// DIRECCION 1: todo paquete del arbol esta clasificado.
	for _, n := range arbol {
		if _, hay := declarado[n]; !hay {
			t.Errorf("el paquete %s no sale en marcos-v1.json, ni dentro ni fuera.\n"+
				"  El silencio es como se cuela un paquete de frontera: `eni` estuvo contado "+
				"dentro por un recuento y fuera por otro, y la diferencia era un reloj.\n"+
				"  Arreglo: declararlo en `marcos` (con su `censados`, o `null` si el censo "+
				"no lo ha verificado) o en `fuera` con su motivo.", n)
		}
	}
	// DIRECCION 2: todo nombre declarado existe.
	enElArbol := map[string]bool{}
	for _, n := range arbol {
		enElArbol[n] = true
	}
	for nombre := range declarado {
		if !enElArbol[nombre] {
			t.Errorf("marcos-v1.json declara el paquete %s y no existe en el arbol. O se "+
				"renombro, o se borro y nadie limpio la lista", nombre)
		}
	}

	// Y TODO MARCO DE DENTRO DICE POR QUE ESTA, y si no tiene censo, por que no.
	for _, m := range d.Marcos {
		if strings.TrimSpace(m.Porque) == "" {
			t.Errorf("%s esta dentro de la v1 y no dice por que", m.Paquete)
		}
		if m.Censados == nil && strings.TrimSpace(m.SinVerificarPorId) == "" {
			t.Errorf("%s no trae censo verificado y no dice por que no. Un hueco sin motivo "+
				"se lee como deuda y como decision a la vez", m.Paquete)
		}
		if m.Censados != nil && strings.TrimSpace(m.SinVerificarPorId) != "" {
			t.Errorf("%s trae censo (%d) Y la explicacion de por que no lo trae. Una de las "+
				"dos es vieja", m.Paquete, *m.Censados)
		}
		if m.Censados != nil && *m.Censados < 0 {
			t.Errorf("%s declara un censo negativo (%d)", m.Paquete, *m.Censados)
		}
	}
}

// coberturaDeLaV1 computa el porcentaje SOBRE EL MISMO CONJUNTO en numerador y
// denominador, que es la parte que un recuento a mano se salta.
//
// Un paquete sin censo verificado sale de LOS DOS. Meterlo solo en el numerador
// (sus relojes escritos) contra un denominador que no lo incluye da un
// porcentaje que sube al escribir paquetes que nadie ha censado, o sea un
// numero que premia justo lo que no se ha medido.
func coberturaDeLaV1(t *testing.T) (escritos, censados, sinCenso int, pct float64) {
	t.Helper()
	d := leerMarcosV1(t)
	relojes := relojesPorPaquete(t)
	for _, m := range d.Marcos {
		if m.Censados == nil {
			sinCenso++
			continue
		}
		censados += *m.Censados
		escritos += relojes[m.Paquete]
	}
	if censados == 0 {
		t.Fatal("el denominador ha salido cero: no hay nada que dividir y el porcentaje seria " +
			"una invencion")
	}
	return escritos, censados, sinCenso, 100 * float64(escritos) / float64(censados)
}

// porcentajeDeclarado lee del README el numero que el proyecto AFIRMA.
//
// Se lee del README y no de una constante de Go porque el README es lo que un
// tercero mira: si el numero del README y el del arbol se separan, el que
// enganda es el del README, asi que es el que tiene que estar atado.
var reCobertura = regexp.MustCompile(
	`(?s)<!-- cobertura-v1:inicio -->.*?\*\*([0-9]+(?:,[0-9]+)?) %\*\*.*?<!-- cobertura-v1:fin -->`)

func porcentajeDeclarado(t *testing.T) (float64, string) {
	t.Helper()
	b, err := os.ReadFile(rutaDelREADME) // #nosec G304 -- ruta constante del repositorio
	if err != nil {
		t.Fatalf("no puedo leer %s: %v", rutaDelREADME, err)
	}
	m := reCobertura.FindSubmatch(b)
	if m == nil {
		t.Fatalf("el README no trae el bloque de cobertura de la v1 entre los marcadores " +
			"<!-- cobertura-v1:inicio --> y <!-- cobertura-v1:fin --> con su porcentaje en " +
			"negrita.\n  Sin ese bloque, esta puerta no vigila nada y el numero del README " +
			"vuelve a moverse solo, que es de donde venimos")
	}
	crudo := strings.Replace(string(m[1]), ",", ".", 1)
	v, err := strconv.ParseFloat(crudo, 64)
	if err != nil {
		t.Fatalf("el porcentaje del README (%q) no es un numero: %v", m[1], err)
	}
	return v, crudo
}

// TestElPorcentajeDeLaV1LoComputaUnTestYNoUnaPersona es la puerta.
//
// UN NUMERO SIN PUERTA SE MUEVE SOLO. El README afirma una cobertura; aqui se
// computa del arbol y se contrasta. La tolerancia es de una decima porque el
// README escribe una decima: no es holgura, es la precision del dato declarado.
func TestElPorcentajeDeLaV1LoComputaUnTestYNoUnaPersona(t *testing.T) {
	escritos, censados, sinCenso, pct := coberturaDeLaV1(t)
	declarado, crudo := porcentajeDeclarado(t)

	if math.Abs(pct-declarado) > 0.05 {
		t.Errorf("el README declara %s %% de cobertura de la v1 y el arbol da %.1f %% "+
			"(%d relojes escritos de %d censados).\n"+
			"  Arreglo: actualiza el bloque cobertura-v1 del README. Si el numero ha BAJADO "+
			"sin que nadie borre relojes, es que el denominador ha crecido: alguien ha "+
			"censado un paquete que antes no lo estaba, y eso es una buena noticia que "+
			"tiene que constar.", crudo, pct, escritos, censados)
	}

	// EL HUECO DEL DENOMINADOR, CON SU CARDINAL. Sin esto, el porcentaje se lee
	// como si cubriera los quince marcos, y cubre diez.
	if sinCenso == 0 {
		t.Errorf("ningun marco de la v1 sale sin censo verificado, y hoy son varios " +
			"(referenciales que no se pueden censar sin la norma delante). O se han " +
			"censado todos, y entonces hay que actualizar esta puerta y el README, o el " +
			"lector de marcos-v1.json no esta viendo los `censados: null`")
	}
	if !strings.Contains(leerREADME(t), fmt.Sprintf("%d de", sinCenso)) {
		t.Errorf("el README no dice que %d de los marcos de la v1 quedan FUERA del "+
			"porcentaje por no tener censo verificado.\n"+
			"  Un porcentaje sin esa frase se lee como si cubriera los quince marcos, y "+
			"cubre los que tienen denominador", sinCenso)
	}
	t.Logf("cobertura de la v1: %d escritos / %d censados = %.1f %%; %d marcos sin censo "+
		"verificado, fuera del calculo", escritos, censados, pct, sinCenso)
}

func leerREADME(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(rutaDelREADME) // #nosec G304 -- ruta constante del repositorio
	if err != nil {
		t.Fatalf("no puedo leer %s: %v", rutaDelREADME, err)
	}
	return string(b)
}

// CONTROL NEGATIVO DEL LECTOR DEL README.
//
// Todo cuelga de una expresion regular, y una expresion regular que no casa
// nada haria `t.Fatalf` (eso se ve), pero una que casara CUALQUIER numero del
// README dejaria la puerta verde para siempre contra el primer porcentaje que
// encontrara. Se comprueba que lee el bloque y no otra cosa.
func TestElLectorDelPorcentajeLeeSuBloqueYNoOtroNumero(t *testing.T) {
	casos := []struct {
		nombre string
		fuente string
		quiere string
	}{
		{"el bloque, con su numero",
			"bla 81,3 % de cobertura\n<!-- cobertura-v1:inicio -->\nson **71,6 %** de los relojes\n<!-- cobertura-v1:fin -->\n",
			"71.6"},
		{"un numero de fuera del bloque no vale",
			"cobertura **99,9 %** en otra seccion\n", ""},
		{"bloque sin numero en negrita",
			"<!-- cobertura-v1:inicio -->\nun 71,6 % suelto\n<!-- cobertura-v1:fin -->\n", ""},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			m := reCobertura.FindStringSubmatch(c.fuente)
			if c.quiere == "" {
				if m != nil {
					t.Errorf("ha casado %q donde no tenia que casar nada: la puerta estaria "+
						"vigilando un numero cualquiera del README", m[1])
				}
				return
			}
			if m == nil {
				t.Fatal("no ha casado el bloque bueno, asi que la puerta no vigila nada")
			}
			if got := strings.Replace(m[1], ",", ".", 1); got != c.quiere {
				t.Errorf("ha leido %q y esperaba %q", got, c.quiere)
			}
		})
	}
}

// LOS CUATRO NUMEROS DEL CORPUS QUE EL README AFIRMA, CONTADOS DEL ARBOL.
//
// Es la misma doctrina que el porcentaje de la v1, aplicada a lo que ya estaba
// escrito: UN NUMERO SIN PUERTA SE MUEVE SOLO. El README dice cuantos paquetes
// hay, cuantos traen relojes, cuantos hitos y cuantos casos dorados, y esos
// cuatro cambian CADA VEZ que alguien escribe corpus, que es varias veces por
// semana. Hasta hoy no los comprobaba nada: se actualizaban a mano o no se
// actualizaban.
//
// Y son los numeros de la portada, o sea los que mira quien llega. Un README
// que dice 477 dorados cuando hay 500 no es un error de redondeo: es la unica
// cifra que un tercero puede contrastar en dos minutos, y si no cuadra deja de
// creerse el resto de la pagina, con razon.
//
// SE CUENTAN CON EL CARGADOR DEL PRODUCTO (corpus.Cargar), no recorriendo los
// JSON a mano: contar con un segundo lector es contar otra cosa el dia que los
// dos se separen, y ademas el cargador es el que decide que es un paquete.
//
// El «dieciseis» del README paso a «16» para que esto lo pueda leer. Un numero
// escrito con letra es un numero que ninguna puerta vigila, y esa es razon
// suficiente.
func TestLosNumerosDelCorpusEnElREADMESalenDelArbol(t *testing.T) {
	ps, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatalf("cargar el corpus: %v", err)
	}
	paquetes, conReloj, hitos, dorados := len(ps), 0, 0, 0
	for _, p := range ps {
		n := 0
		for _, o := range p.Obligaciones {
			if o.Temporalidad == nil {
				continue
			}
			n++
			// Un plazo escalonado declara sus hitos; los demas relojes tienen
			// uno. Es la misma cuenta que hace el calendario al pintarlos.
			if len(o.Temporalidad.Hitos) > 0 {
				hitos += len(o.Temporalidad.Hitos)
			} else {
				hitos++
			}
		}
		if n > 0 {
			conReloj++
		}
		dorados += len(p.Dorados)
	}
	if paquetes < 30 || dorados < 100 {
		t.Fatalf("el corpus cargado trae %d paquetes y %d dorados: el recorrido esta midiendo "+
			"el vacio", paquetes, dorados)
	}

	readme := leerREADME(t)
	casos := []struct {
		que      string
		patron   string
		contado  int
		yQueHago string
	}{
		{"paquetes", `\*\*(\d+) paquetes\*\*`, paquetes, ""},
		{"paquetes con relojes reales", `\*\*(\d+) con relojes reales`, conReloj, ""},
		{"hitos", `con relojes reales: (\d+) hitos`, hitos, ""},
		{"casos dorados", `(\d+) casos dorados\*\*`, dorados,
			"si han subido es porque alguien ha escrito corpus, y esa es la cifra que mas " +
				"se mira de la portada"},
	}
	for _, c := range casos {
		t.Run(c.que, func(t *testing.T) {
			re := regexp.MustCompile(c.patron)
			m := re.FindStringSubmatch(readme)
			if m == nil {
				t.Fatalf("el README no dice cuantos %s hay con el patron %q. Si se ha "+
					"redactado de otra forma, esta puerta ha dejado de vigilar ese numero "+
					"y hay que actualizar el patron, no borrarlo", c.que, c.patron)
			}
			declarado, err := strconv.Atoi(m[1])
			if err != nil {
				t.Fatalf("%q no es un numero: %v", m[1], err)
			}
			if declarado != c.contado {
				extra := ""
				if c.yQueHago != "" {
					extra = "\n  " + c.yQueHago
				}
				t.Errorf("el README dice %d %s y el arbol tiene %d.%s\n"+
					"  Arreglo: actualiza el README en el mismo commit que mueve el numero.",
					declarado, c.que, c.contado, extra)
			}
		})
	}
}

// CONTROL NEGATIVO DE LOS PATRONES: cada uno tiene que leer SU numero y no el
// del vecino. Los cuatro viven en la misma frase del README, asi que un patron
// flojo cazaria el primer numero que encontrara y las cuatro comprobaciones
// medirian lo mismo.
func TestCadaPatronDelREADMELeeSuPropioNumero(t *testing.T) {
	frase := "**33 paquetes** con su estrato legal, de los cuales " +
		"**16 con relojes reales: 164 hitos y 477 casos dorados** que se ejecutan"
	quiere := map[string]string{
		`\*\*(\d+) paquetes\*\*`:          "33",
		`\*\*(\d+) con relojes reales`:    "16",
		`con relojes reales: (\d+) hitos`: "164",
		`(\d+) casos dorados\*\*`:         "477",
	}
	for patron, esperado := range quiere {
		m := regexp.MustCompile(patron).FindStringSubmatch(frase)
		if m == nil {
			t.Errorf("el patron %q no casa nada en la frase de referencia", patron)
			continue
		}
		if m[1] != esperado {
			t.Errorf("el patron %q ha leido %q y su numero es %q: esta cazando el del vecino",
				patron, m[1], esperado)
		}
	}
}
