package plazum

import (
	"encoding/json"
	"errors"
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
	"github.com/marcosmatalab/plazum/nucleo/metrica"
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

// relojesEscritos son los dos numeradores de un paquete, separados por QUIEN
// escribe el numero. Es la correccion del 03-09-2026 y la decision es del
// estratega: un ritual de plazum y un reloj que la norma escribe no son la
// misma afirmacion, y sumarlos deja subir la cobertura escribiendo rituales
// nuestros, o sea premia justo lo que no queremos incentivar.
type relojesEscritos struct {
	// DeLaNorma: el numero lo pone la norma. Es el NUMERADOR.
	DeLaNorma int
	// Rituales: el numero lo pone plazum (D-12). Sale AL LADO, con su propio
	// recuento, y nunca dentro del porcentaje.
	Rituales int
}

// relojesPorPaquete cuenta las obligaciones con bloque `temporalidad` de cada
// paquete, separadas por quien escribe el intervalo.
//
// SE CLASIFICA CON corpus.EsRitualDePlazum, que es el clasificador DEL
// PRODUCTO: el mismo que usa el linter para exigir que `articulo` y
// `origen_del_intervalo` digan lo mismo. Una segunda copia de esa regla aqui
// contaria otra cosa el dia que las dos se separen, y ese dia nadie estaria
// mirando esta cuenta.
func relojesPorPaquete(t *testing.T) map[string]relojesEscritos {
	t.Helper()
	out := map[string]relojesEscritos{}
	for _, n := range paquetesDelArbol(t) {
		b, err := os.ReadFile(filepath.Join("paquetes", n, "paquete.json")) // #nosec G304 -- recorre el arbol del repositorio
		if err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		var p struct {
			Obligaciones []struct {
				Articulo     string           `json:"articulo"`
				Temporalidad *json.RawMessage `json:"temporalidad"`
			} `json:"obligaciones"`
		}
		if err := json.Unmarshal(b, &p); err != nil {
			t.Fatalf("%s/paquete.json no parsea: %v", n, err)
		}
		c := out[n]
		for _, o := range p.Obligaciones {
			if o.Temporalidad == nil {
				continue
			}
			if corpus.EsRitualDePlazum(o.Articulo) {
				c.Rituales++
			} else {
				c.DeLaNorma++
			}
		}
		out[n] = c
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

// cuentaDeLaV1 son los tres numeros que el proyecto publica, y son tres porque
// dicen tres cosas distintas que sumadas mienten.
type cuentaDeLaV1 struct {
	// DeLaNorma y Censados son el porcentaje: relojes cuyo intervalo escribe la
	// norma, contra puntos que el censo verifico.
	DeLaNorma int
	Censados  int
	// Rituales son los relojes de plazum sobre esos mismos marcos. Van AL LADO,
	// con su propio recuento, y no dentro del porcentaje.
	Rituales int
	// SinCenso es el cardinal de marcos que no tienen denominador posible, y
	// RelojesSinCenso / RitualesSinCenso lo que tienen escrito. Para esos la
	// cifra honesta es «sin denominador, N escritos», nunca un cero: un cero se
	// lee como «medido y vacio», y no esta medido.
	SinCenso         int
	RelojesSinCenso  int
	RitualesSinCenso int
}

// pct pasa por nucleo/metrica en vez de dividir a pelo, y no es ceremonia: un
// denominador cero dividido a pelo da +Inf o NaN, que en un %.1f sale como
// «+Inf %» y en una plantilla sale como texto. Un valor imposible tiene que
// parar antes de convertirse en una cifra con forma de dato.
func (c cuentaDeLaV1) pct(t *testing.T) float64 {
	t.Helper()
	v, err := metrica.Fraccion{
		Parte: c.DeLaNorma, Total: c.Censados,
		QueEsParte: "relojes con intervalo de la norma", QueEsTotal: "puntos censados",
	}.Porcentaje()
	if err != nil {
		t.Fatalf("la cobertura de la v1 no se puede publicar: %v", err)
	}
	return v
}

// coberturaDeLaV1 computa el porcentaje SOBRE EL MISMO CONJUNTO en numerador y
// denominador, que es la parte que un recuento a mano se salta.
//
// # Un paquete sin censo verificado sale de LOS DOS
//
// Meterlo solo en el numerador (sus relojes escritos) contra un denominador que
// no lo incluye da un porcentaje que sube al escribir paquetes que nadie ha
// censado, o sea un numero que premia justo lo que no se ha medido.
//
// # Y UN RITUAL DE PLAZUM NO ENTRA EN EL NUMERADOR (decision del 03-09-2026)
//
// Esta es la segunda correccion de este numero en dos dias, y las dos han sido
// hacia abajo. La primera saco del numerador los rituales de un paquete
// REFERENCIAL (`iso27001` aportaba 6 arriba y 0 abajo) y discriminaba por el
// ESTRATO; esta discrimina por quien escribe el numero, que es la pregunta que
// de verdad se estaba haciendo, y deja el estrato fuera de esta cuenta. Saca
// los rituales de todos los paquetes:
// un ritual de plazum sobre un marco transcrito y un reloj que la norma escribe
// no son la misma afirmacion, y sumarlos deja subir la cobertura escribiendo
// rituales nuestros, que es exactamente el incentivo que no queremos.
//
// # Lo que esto le hace a `nis2-tecnica`, dicho aqui y no descubierto luego
//
// 44 de sus 48 puntos son cadencias que el anexo impone SIN numero, asi que son
// rituales por D-12 y salen del numerador: el paquete pasa de aportar 48 de 48 a
// aportar 4 de 48. Eso NO dice que falten 44 puntos por escribir (estan
// escritos, transcritos y con dorados): dice que en 44 el numero es de plazum y
// no de la norma. La cifra estricta mide una cosa mas dura que «cuanto hay
// escrito», y por eso va acompanada del recuento de rituales en vez de sola.
func coberturaDeLaV1(t *testing.T) cuentaDeLaV1 {
	t.Helper()
	d := leerMarcosV1(t)
	relojes := relojesPorPaquete(t)
	var c cuentaDeLaV1
	for _, m := range d.Marcos {
		r := relojes[m.Paquete]
		if m.Censados == nil {
			c.SinCenso++
			c.RelojesSinCenso += r.DeLaNorma
			c.RitualesSinCenso += r.Rituales
			continue
		}
		// NINGUN PAQUETE PUEDE APORTAR MAS ARRIBA QUE ABAJO, y esta guarda
		// nacio ROJA sobre dos paquetes reales.
		//
		// El 03-09-2026, tras la campana de cuatro frentes, `cra` tenia 23
		// relojes con cita y su censo contaba 22, y `nis2-ue` tenia 12 contra 9.
		// Una fraccion por encima de uno no es un redondeo: dice que el
		// denominador esta mal, y como el agregado los suma, un paquete al 105 %
		// SUBE el total sin que nada lo nombre. Es exactamente la forma de fallo
		// de esta metrica: equivocarse a favor.
		//
		// No se arregla bajando el numerador (los relojes estan escritos y
		// citados) sino reconociendo que esa fila del censo esta desmentida por
		// el propio paquete.
		//
		// SE VALIDA CON nucleo/metrica Y NO CON UN `if` DE AQUI, y esa es la
		// diferencia entre arreglar el caso y arreglar la familia. La primera
		// version de esta guarda era un `if` escrito a mano en esta funcion, y
		// habria dejado fuera a las demas cifras que plazum publica: la cuenta
		// del calendario, los cubos del escalado, el TTFV. Un valor imposible
		// se rechaza en un sitio o se rechaza en ninguno.
		por := metrica.Fraccion{
			Parte: r.DeLaNorma, Total: *m.Censados,
			QueEsParte: "relojes con cita escritos en " + m.Paquete,
			QueEsTotal: "puntos censados de " + m.Paquete,
		}
		if err := por.Validar(); err != nil && !errors.Is(err, metrica.ErrDenominadorCero) {
			// El denominador cero SI es legitimo aqui y esta contado: `iso27001`
			// declara 0 porque la norma no trae ni una cadencia numerica, y sus
			// relojes son todos rituales, asi que aporta 0 arriba y 0 abajo. Lo
			// que no es legitimo es la parte mayor que el total.
			t.Errorf("%v\n"+
				"  Arreglo: o el censo se recuenta contra la fuente primaria, o esa fila "+
				"pasa a `censados: null` con la refutacion escrita. Lo que no vale es "+
				"dejar un denominador que el propio paquete desmiente.", err)
		}
		c.Censados += *m.Censados
		c.DeLaNorma += r.DeLaNorma
		c.Rituales += r.Rituales
	}
	if c.Censados == 0 {
		t.Fatal("el denominador ha salido cero: no hay nada que dividir y el porcentaje seria " +
			"una invencion")
	}
	return c
}

// LAS TRES CIFRAS QUE EL README AFIRMA, LEIDAS DE SU BLOQUE.
//
// Se leen del README y no de constantes de Go porque el README es lo que un
// tercero mira: si el numero del README y el del arbol se separan, el que
// engana es el del README, asi que es el que tiene que estar atado.
//
// LAS TRES, y no solo el porcentaje: la decision del 03-09-2026 es que la
// cobertura y los rituales se publican JUNTOS y los dos los computa la puerta.
// Un porcentaje vigilado al lado de un «+N rituales» que no vigila nadie es
// medio numero atado, y la mitad suelta es la que se mueve.
//
// Cada expresion exige sus marcadores dentro, para que ninguna pueda cazar un
// numero de otra seccion del README.
var (
	reCobertura = regexp.MustCompile(
		`(?s)<!-- cobertura-v1:inicio -->.*?\*\*([0-9]+(?:,[0-9]+)?) %\*\*.*?<!-- cobertura-v1:fin -->`)
	reRituales = regexp.MustCompile(
		`(?s)<!-- cobertura-v1:inicio -->.*?\*\*\+([0-9]+) rituales de plazum\*\*.*?<!-- cobertura-v1:fin -->`)
	reSinDenominador = regexp.MustCompile(
		`(?s)<!-- cobertura-v1:inicio -->.*?\*\*([0-9]+) de los 15 marcos\*\*.*?<!-- cobertura-v1:fin -->`)
)

// cifraDelBloque saca un entero del bloque de cobertura del README.
func cifraDelBloque(t *testing.T, re *regexp.Regexp, que string) int {
	t.Helper()
	m := re.FindSubmatch([]byte(leerREADME(t)))
	if m == nil {
		t.Fatalf("el bloque cobertura-v1 del README no dice %s con el patron %q.\n"+
			"  Sin ese dato la puerta no vigila esa cifra y vuelve a moverse sola, que es "+
			"de donde venimos.", que, re)
	}
	v, err := strconv.Atoi(string(m[1]))
	if err != nil {
		t.Fatalf("%s del README (%q) no es un entero: %v", que, m[1], err)
	}
	return v
}

func porcentajeDeclarado(t *testing.T) (float64, string) {
	t.Helper()
	m := reCobertura.FindSubmatch([]byte(leerREADME(t)))
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
// UN NUMERO SIN PUERTA SE MUEVE SOLO. El README afirma una cobertura, un
// recuento de rituales y un cardinal de marcos sin denominador; los tres se
// computan aqui del arbol y se contrastan. La tolerancia del porcentaje es de
// una decima porque el README escribe una decima: no es holgura, es la
// precision del dato declarado.
//
// Y LA PUERTA VIGILA EN LAS DOS DIRECCIONES A PROPOSITO. La familia de esta
// metrica es que se equivoca A FAVOR: las dos correcciones que ha tenido la
// subieron, ninguna la bajo. Una cifra cuyo fallo probable es favorecerte
// necesita, como PUERTAS_ESPERADAS y HERRAMIENTAS_ESPERADAS, que cualquier
// separacion en cualquier sentido rompa, y no solo la que perjudica.
func TestElPorcentajeDeLaV1LoComputaUnTestYNoUnaPersona(t *testing.T) {
	c := coberturaDeLaV1(t)
	declarado, crudo := porcentajeDeclarado(t)

	if math.Abs(c.pct(t)-declarado) > 0.05 {
		t.Errorf("el README declara %s %% de cobertura de la v1 y el arbol da %.1f %% "+
			"(%d relojes con intervalo de la norma sobre %d censados).\n"+
			"  Arreglo: actualiza el bloque cobertura-v1 del README. Si el numero ha BAJADO "+
			"sin que nadie borre relojes, es que el denominador ha crecido: alguien ha "+
			"censado un paquete que antes no lo estaba, y eso es una buena noticia que "+
			"tiene que constar.", crudo, c.pct(t), c.DeLaNorma, c.Censados)
	}

	// LA SEGUNDA CIFRA, con la misma puerta que la primera.
	if r := cifraDelBloque(t, reRituales, "cuantos rituales de plazum hay"); r != c.Rituales {
		t.Errorf("el README dice +%d rituales de plazum sobre los marcos censados y el arbol "+
			"tiene %d.\n"+
			"  Esta cifra existe para que la de arriba se pueda leer: sin ella, un 48 %% se "+
			"lee como «falta la mitad del corpus» cuando lo que dice es «en la mitad el "+
			"numero lo pone plazum y no la norma».", r, c.Rituales)
	}

	// EL HUECO DEL DENOMINADOR, CON SU CARDINAL. Sin esto, el porcentaje se lee
	// como si cubriera los quince marcos, y cubre diez.
	if c.SinCenso == 0 {
		t.Errorf("ningun marco de la v1 sale sin censo verificado, y hoy son varios " +
			"(referenciales que no se pueden censar sin la norma delante). O se han " +
			"censado todos, y entonces hay que actualizar esta puerta y el README, o el " +
			"lector de marcos-v1.json no esta viendo los `censados: null`")
	}
	if n := cifraDelBloque(t, reSinDenominador, "cuantos marcos quedan sin denominador"); n != c.SinCenso {
		t.Errorf("el README dice que %d de los 15 marcos quedan fuera del porcentaje y son %d.\n"+
			"  Un porcentaje sin ese cardinal se lee como si cubriera los quince marcos, y "+
			"cubre los que tienen denominador.", n, c.SinCenso)
	}
	// Y LO QUE ESOS MARCOS TIENEN ESCRITO SE DICE, PORQUE UN CERO ALLI SERIA
	// MENTIRA. «Sin denominador» no es «vacio»: son 27 rituales y 2 relojes que
	// existen y que ninguna fraccion puede expresar. El frente A hizo lo
	// correcto negandose a proponer un 0 para SOC 2.
	escritoSinDenominador := fmt.Sprintf("%d rituales", c.RitualesSinCenso)
	if !strings.Contains(leerREADME(t), escritoSinDenominador) {
		t.Errorf("el README no dice cuanto hay escrito en los marcos sin denominador "+
			"(esperaba %q).\n"+
			"  Un marco sin censo posible se publica como «sin denominador, N escritos» y "+
			"nunca como un cero: el cero se lee como medido y vacio, y no esta medido.",
			escritoSinDenominador)
	}

	t.Logf("cobertura estricta de la v1: %d relojes de la norma / %d censados = %.1f %%; "+
		"+%d rituales de plazum; %d marcos sin denominador (%d relojes y %d rituales escritos)",
		c.DeLaNorma, c.Censados, c.pct(t), c.Rituales, c.SinCenso, c.RelojesSinCenso,
		c.RitualesSinCenso)
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
