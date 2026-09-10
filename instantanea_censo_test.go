package plazum

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// EL CENSO DE LOS CARDINALES VIGILADOS DE LA INSTANTANEA.
//
// # Que problema resuelve, y por que no basta con lo que ya habia
//
// `instantanea_test.go` ata TRES cifras de la foto al arbol desde el 04-09-2026,
// y funciona. Lo que no tenia es CENSO: nadie decia cuantas cifras de esa foto
// estan vigiladas y cuantas no, asi que la respuesta honesta a «¿cuadra la
// instantanea con el repositorio?» era «tres de sus cifras si, y del resto no
// consta». Un documento que publica decenas de numeros y tiene tres atados se
// lee como un documento atado.
//
// # Por que el alcance es EL FICHERO ENTERO y no un rango de lineas
//
// Porque un rango de lineas se esquiva moviendo el numero una linea, y una
// columna se esquiva sacandolo de la columna. Este documento ya lo hace: la
// misma cifra de relojes aparece en la tabla de arriba, en el parrafo de la
// cabecera y en dos filas de la autoevaluacion. Cada cardinal vigilado se ancla
// en SU PROPIA FRASE sobre el texto completo, asi que da igual donde este.
//
// # La frontera con la mitad congelada, que es lo que este censo hace explicito
//
// El encabezado de la foto dice que se rehace entera o no se hace. Y a la vez
// tres de sus filas se estaban retocando celda a celda para que la puerta del
// 04-09 siguiera verde. Las dos cosas eran ciertas y se contradecian.
//
// Se separan: la TABLA DE MEDIDAS esta viva y sus cifras vigiladas se recomputan
// del arbol; la AUTOEVALUACION es del 04-09-2026, lo dice su columna «Medida»
// fila a fila, y no se retoca. Lo que el arbol ya desmiente de la autoevaluacion
// esta contado en docs/pendientes.md, que es el sitio donde una deuda con
// cardinal molesta hasta que se cierra.
//
// # El control negativo que exige el encargo
//
// La prosa historica de la cabecera SI puede citar el 230 como pasado, y tiene
// que poder. Aqui eso sale gratis y no por una excepcion escrita: cada cardinal
// se ancla en su frase, y «230 relojes (hoy 252)» no es ninguna de esas frases.
// Lo comprueba TestElCensoDeLaInstantaneaNoTocaLaProsaHistorica, que es la mitad
// sin la cual este diseno seria una lista de excepciones que hay que acertar.
//
// # NO NACE VERDE, y aqui esta la cuenta exacta
//
// De los seis cardinales, DOS estaban rojos sobre el documento tal y como estaba
// en el commit anterior, y sobre dato real y no sobre una mutacion:
//
//	paquetes de corpus         decia 33 y del arbol salen 20
//	paquetes con obligaciones  decia 21 y del arbol salen 20
//
// Los otros CUATRO no estaban ni bien ni mal: sus filas NO EXISTIAN en la foto.
// Se dice asi en vez de contarlos como estrenos rojos, que seria inflar la
// demostracion: una puerta que se estrena sobre una fila que ella misma trae no
// ha cazado nada, ha declarado algo.
//
// Y ademas se ha visto fallar por mutacion: cambiar cualquiera de los seis
// numeros de la foto la pone roja nombrando la cifra y el valor del arbol.

// CardinalesVigiladosDeLaInstantanea es cuantas cifras de la foto estan atadas al
// arbol.
//
// Igualdad exacta, como PUERTAS_ESPERADAS: si SUBE, alguien ha atado una cifra
// mas y tiene que constar; si BAJA, alguien ha dejado de vigilar una y eso no
// puede pasar en silencio, que es exactamente como se llega a un documento que
// se da la razon a si mismo.
const CardinalesVigiladosDeLaInstantanea = 6

// cardinalVigilado es una cifra de la foto con quien la computa DEL ARBOL.
//
// `computa` devuelve el valor de hoy. Nunca lee otro documento: un cardinal
// atado a otro documento no esta atado, esta de acuerdo, y un conjunto de
// documentos de acuerdo entre si es indistinguible de uno correcto.
//
// La UNICA excepcion es la cobertura, y esta dicha en su entrada: se compara
// contra el bloque del README porque ese bloque YA tiene puerta contra el arbol,
// y recomputarla aqui seria una tercera implementacion de la misma cifra.
type cardinalVigilado struct {
	que     string
	re      *regexp.Regexp
	computa func(t *testing.T) int
}

func cardinalesDeLaInstantanea() []cardinalVigilado {
	return []cardinalVigilado{
		{
			que: "los paquetes de corpus de la tabla de medidas",
			re: regexp.MustCompile(
				`(?m)^\|\s*Paquetes de corpus\s*\|\s*\*\*(\d+)\*\*`),
			computa: func(t *testing.T) int {
				dirs, err := directoriosPublicados("paquetes")
				if err != nil {
					t.Fatalf("no puedo leer paquetes/: %v", err)
				}
				return len(dirs)
			},
		},
		{
			que: "los paquetes con obligaciones escritas",
			re: regexp.MustCompile(
				`(?m)^\|\s*Paquetes de corpus\s*\|\s*\*\*\d+\*\*, de los cuales \*\*(\d+) con obligaciones`),
			computa: func(t *testing.T) int {
				n := 0
				for _, p := range corpusPublicado(t) {
					if len(p.Obligaciones) > 0 {
						n++
					}
				}
				return n
			},
		},
		{
			que: "los hitos de reloj del corpus publicado",
			re: regexp.MustCompile(
				`(?m)^\|\s*Hitos de reloj y casos dorados\s*\|\s*\*\*(\d+)\*\* hitos`),
			computa: func(t *testing.T) int {
				hitos, _ := contarCorpusPublicado(t)
				return hitos
			},
		},
		{
			que: "los casos dorados del corpus publicado",
			re: regexp.MustCompile(
				`(?m)^\|\s*Hitos de reloj y casos dorados\s*\|\s*\*\*\d+\*\* hitos y \*\*(\d+)\*\* dorados`),
			computa: func(t *testing.T) int {
				_, dorados := contarCorpusPublicado(t)
				return dorados
			},
		},
		{
			que: "las copias rotas del ensayo de restauracion",
			re: regexp.MustCompile(
				`(?m)^\|\s*Copias rotas del ensayo de restauración\s*\|\s*\*\*(\d+)\*\*`),
			computa: contarCopiasRotas,
		},
		{
			que: "las puertas D11 que siguen abiertas",
			re: regexp.MustCompile(
				`(?m)^\|\s*Puertas propias de D11 todavía abiertas\s*\|\s*\*\*(\d+)\*\* de 5`),
			computa: func(t *testing.T) int {
				return strings.Count(leerEtapas(t), "- [ ] **PUERTA D11-")
			},
		},
	}
}

// contarCopiasRotas cuenta las invocaciones `comprobar` del ensayo de copias.
//
// Del workflow y no de una lista escrita aqui: el propio workflow tiene su suelo
// (`-lt 9`), y el nombre de su paso decia «ocho copias rotas» cuando eran nueve.
// Un cardinal en el nombre de un paso es lo que sale en la interfaz de GitHub, y
// ahi no lo vigila nadie.
func contarCopiasRotas(t *testing.T) int {
	t.Helper()
	f := leerDoc(t, ".github/workflows/etapa2-copias.yml")
	n := len(regexp.MustCompile(`(?m)^\s*comprobar\s+[a-z]`).FindAllString(f, -1))
	if n == 0 {
		t.Fatal("he contado CERO copias rotas en el ensayo. O la forma de la invocacion " +
			"cambio, o el paso se fue, y en los dos casos esto daria verde sin mirar nada")
	}
	return n
}

// TestElCensoDeLaInstantaneaCuadraConElArbol es la puerta.
func TestElCensoDeLaInstantaneaCuadraConElArbol(t *testing.T) {
	inst := leerDoc(t, rutaDeLaInstantanea)
	censo := cardinalesDeLaInstantanea()

	if len(censo) != CardinalesVigiladosDeLaInstantanea {
		t.Fatalf("el censo trae %d cardinales y la constante dice %d.\n"+
			"  Si ha SUBIDO, se ha atado una cifra mas y hay que decirlo aqui.\n"+
			"  Si ha BAJADO, se ha dejado de vigilar una, y eso es como se llega a un "+
			"documento que se da la razon a si mismo.",
			len(censo), CardinalesVigiladosDeLaInstantanea)
	}

	for _, c := range censo {
		quiero := c.computa(t)
		bruto := unico(t, c.re, inst, c.que)
		dice, err := strconv.Atoi(bruto)
		if err != nil {
			t.Errorf("%s: %q no es un numero", c.que, bruto)
			continue
		}
		if dice != quiero {
			t.Errorf("la instantanea dice %d en %s y del arbol salen %d.\n"+
				"  Gana el arbol: se actualiza el documento, no se relaja la puerta.",
				dice, c.que, quiero)
		}
	}
	if !t.Failed() {
		t.Logf("los %d cardinales vigilados de la instantanea cuadran con el arbol",
			len(censo))
	}
}

// TestElCensoDeLaInstantaneaNoTocaLaProsaHistorica es el control negativo que
// hace utilizable este diseno.
//
// La cabecera de la foto cuenta que el 04-09-2026 se quedaron viejos siete
// numeros en doce horas, y los cita: «230 relojes (hoy 252)». Esa frase es
// HISTORIA y tiene que poder seguir ahi. Una puerta que barriera «todo numero de
// esta zona» la pondria roja, y la reaccion barata seria borrar la unica parte
// del documento que cuenta como se equivoco.
//
// Aqui eso no pasa por una excepcion escrita, que habria que acertar, sino por
// el diseno: cada cardinal se ancla en SU frase. Este test lo demuestra en vez de
// darlo por supuesto.
func TestElCensoDeLaInstantaneaNoTocaLaProsaHistorica(t *testing.T) {
	historicas := []string{
		"contados: 230 relojes (hoy **252**), 51,4 % de cobertura de la v1 (hoy **56,7 %**)",
		"un TTFV de 15m51s (hoy **20m27s**)",
		"por la mañana eran 230 y 51,4 %",
	}
	for _, c := range cardinalesDeLaInstantanea() {
		for _, h := range historicas {
			if c.re.MatchString(h) {
				t.Errorf("el cardinal vigilado %q caza la prosa historica %q.\n"+
					"  Esa frase cuenta un numero del pasado y tiene que poder seguir ahi: "+
					"una puerta que la ponga roja acaba borrando la unica parte del "+
					"documento que cuenta como se equivoco.", c.que, h)
			}
		}
	}

	// CONTROL POSITIVO DE ESTE CONTROL NEGATIVO: los seis anclajes tienen que
	// cazar ALGO en el documento de verdad. Sin esto, un anclaje que no casara
	// nada pasaria las dos direcciones de arriba con nota.
	inst := leerDoc(t, rutaDeLaInstantanea)
	for _, c := range cardinalesDeLaInstantanea() {
		if !c.re.MatchString(inst) {
			t.Errorf("el cardinal vigilado %q no caza nada en la instantanea, asi que no "+
				"vigila nada", c.que)
		}
	}
}
