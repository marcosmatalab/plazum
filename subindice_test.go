package plazum

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// LA PUERTA DEL SUBINDICE DE PLATAFORMA: el marcador deja de escribirse a mano.
//
// # De donde sale
//
// El 04-09-2026 se pidio publicar un subindice nuevo, «plataforma open source,
// publicable», sobre doce de las diecisiete dimensiones de la rubrica. Un
// subindice es, por construccion, un numero que sube al quitar dimensiones
// vacias: es la forma mas limpia que hay de mejorar una nota sin construir
// nada. Asi que nace con puerta el mismo dia que nace el numero, y no despues.
//
// La regla del estratega que esto tiene que hacer contestable, literal: «si en
// algun momento un cambio de definicion sube un numero sin que suba nada real,
// se dice en voz alta».
//
// # Los tres ficheros, y por que cada dato vive en uno solo
//
//	docs/diseno.md      LOS PESOS. No son de este frente y no se tocan aqui.
//	                    Que la puerta los lea de ahi es lo que impide que un
//	                    subindice se arregle moviendo un peso.
//	docs/instantanea.md LAS NOTAS. Una nota es un juicio y ningun test la puede
//	                    verificar; lo que si se puede es exigir que sea LA MISMA
//	                    en los dos sitios donde aparece.
//	docs/marcador.md    LA MEMBRESIA y LAS CIFRAS PUBLICADAS.
//
// El marcador REPITE peso y nota de cada dimension, porque un lector de fuera
// tiene que poder recalcular sin abrir otro fichero. La copia existe para el
// lector y no puede separarse de su origen: la puerta la compara celda a celda.
// Es el mismo trato que el bloque de cobertura del README.
//
// # LAS DOS DIRECCIONES, y aqui pesan mas que en ningun otro numero
//
// Todo se compara con igualdad exacta: que una cifra BAJE rompe igual que si
// sube. Y no basta con el ponderado, que tiene dos decimales y se traga los
// movimientos pequenos: bajar D9 (peso 3) de 9,7 a 9,6 deja el subindice en el
// mismo 8,32 redondeado. Por eso se vigila tambien el NUMERADOR, que se mueve
// 0,3 y se publica con un decimal. Se vigilan los cuatro valores de cada linea.
//
// # Lo que esta puerta NO puede hacer, dicho aqui y no descubierto luego
//
// No puede impedir que alguien escriba 9,0 donde hoy hay un 6,5. Una nota es un
// juicio. Lo unico mecanico contra eso es que subir una nota sube los DOS
// numeros publicados, mientras que mover la membresia sube solo uno: por eso el
// global se publica al lado y nunca en lugar del subindice.

const (
	rutaDelDiseno       = "docs/diseno.md"
	rutaDeLaInstantanea = "docs/instantanea.md"
	rutaDelMarcador     = "docs/marcador.md"
)

// LAS TRES FILAS SE RECONOCEN POR SU PRIMERA CELDA, QUE TIENE QUE SER EXACTAMENTE
// EL IDENTIFICADOR.
//
// Es la guarda contra el fallo probable de estos tres ficheros, que estan llenos
// de tablas que hablan de dimensiones: la tabla de movimiento de pesos de
// diseno.md abre sus filas con «D3 Cobertura por estratos...» y la de excluidas
// del marcador con «D5 Conectores WASM...». Ninguna de las dos casa aqui, porque
// entre el identificador y la barra solo se admite espacio.
var (
	// diseno.md: | D1 | nombre | 12 | **9,7** | estado | que sostiene |
	// El peso puede venir en negrita: D-20 marco asi los cuatro que movio.
	reFilaDelDiseno = regexp.MustCompile(
		`(?m)^\|\s*(D\d{1,2})\s*\|[^|]*\|\s*\*{0,2}(\d{1,2})\*{0,2}\s*\|`)
	// instantanea.md: | D1 | nombre | 9,7 | **9,0** | que sostiene |
	// La nota de HOY es la que va en negrita; la de diseno, al lado, no.
	reFilaDeLaInstantanea = regexp.MustCompile(
		`(?m)^\|\s*(D\d{1,2})\s*\|[^|]*\|[^|]*\|\s*\*\*(\d{1,2},\d)\*\*\s*\|`)
	// marcador.md: | D1 | nombre | 12 | 9,0 | dentro |
	reFilaDelMarcador = regexp.MustCompile(
		`(?m)^\|\s*(D\d{1,2})\s*\|[^|]*\|\s*(\d{1,2})\s*\|\s*(\d{1,2},\d)\s*\|\s*(dentro|fuera)\s*\|`)
)

// LAS TRES CIFRAS PUBLICADAS, LEIDAS DE SU BLOQUE Y DE NINGUN OTRO SITIO.
//
// Cada expresion exige los dos marcadores, para que ninguna pueda cazar un
// numero de otra seccion del documento. Es el patron del bloque cobertura-v1
// del README, y por el mismo motivo: el fichero entero esta lleno de cifras
// parecidas, y una expresion suelta cogeria la primera que se le pareciera.
var (
	reSubindicePublicado = regexp.MustCompile(
		`(?s)<!-- marcador:inicio -->.*?publicable: (\d{1,2},\d\d)\*\*, sobre \*\*(\d+) ` +
			`dimensiones\*\* y \*\*(\d+) puntos de peso\*\*, con numerador \*\*(\d+,\d)\*\*` +
			`.*?<!-- marcador:fin -->`)
	reGlobalPublicado = regexp.MustCompile(
		`(?s)<!-- marcador:inicio -->.*?Global de las (\d+) dimensiones: (\d{1,2},\d\d)\*\*, ` +
			`sobre \*\*(\d+) puntos de peso\*\*, con numerador \*\*(\d+,\d)\*\*` +
			`.*?<!-- marcador:fin -->`)
	reExcluidasPublicado = regexp.MustCompile(
		`(?s)<!-- marcador:inicio -->.*?Las (\d+) dimensiones excluidas, medidas aparte: ` +
			`(\d{1,2},\d\d)\*\*, sobre \*\*(\d+) puntos de peso\*\*, con numerador \*\*(\d+,\d)\*\*` +
			`.*?<!-- marcador:fin -->`)
)

func leerDoc(t *testing.T, ruta string) string {
	t.Helper()
	b, err := os.ReadFile(ruta) // #nosec G304 -- ruta constante del repositorio
	if err != nil {
		t.Fatalf("no puedo leer %s: %v", ruta, err)
	}
	return string(b)
}

// coma convierte «8,32» en 8.32. Los documentos se escriben en espanol y el
// separador decimal es la coma; convertirlo aqui, en un solo sitio, evita que
// cada llamante se invente su propio parseo.
func coma(t *testing.T, s, que string) float64 {
	t.Helper()
	v, err := strconv.ParseFloat(strings.Replace(s, ",", ".", 1), 64)
	if err != nil {
		t.Fatalf("%s vale %q y no es un numero: %v", que, s, err)
	}
	return v
}

// dosDecimales redondea alejandose del cero, que es como redondea una persona
// al escribir el documento. Se compara el valor REDONDEADO con el publicado y
// con igualdad exacta, en vez de con una tolerancia: una tolerancia es holgura,
// y la holgura de un numero cuyo fallo probable es favorecerte se gasta siempre
// en la misma direccion.
func dosDecimales(v float64) float64 {
	return math.Round(v*100) / 100
}

// unDecimal hace lo mismo con el numerador, que se publica con un decimal
// porque todo producto peso x nota lo tiene.
func unDecimal(v float64) float64 {
	return math.Round(v*10) / 10
}

func pesosDelDiseno(t *testing.T) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, m := range reFilaDelDiseno.FindAllStringSubmatch(leerDoc(t, rutaDelDiseno), -1) {
		if _, ya := out[m[1]]; ya {
			t.Fatalf("%s declara %s dos veces con peso: la rubrica tiene una fila por "+
				"dimension y esta puerta estaria sumando una de mas", rutaDelDiseno, m[1])
		}
		p, err := strconv.Atoi(m[2])
		if err != nil {
			t.Fatalf("el peso de %s en %s (%q) no es un entero: %v", m[1], rutaDelDiseno, m[2], err)
		}
		out[m[1]] = p
	}
	if len(out) != 17 {
		t.Fatalf("%s da %d dimensiones con peso y la rubrica tiene 17.\n"+
			"  O la tabla de §14 ha cambiado de forma y esta puerta estaria midiendo el "+
			"vacio, o la rubrica ha crecido y hay que decirlo.", rutaDelDiseno, len(out))
	}
	return out
}

func notasDeLaInstantanea(t *testing.T) map[string]float64 {
	t.Helper()
	out := map[string]float64{}
	for _, m := range reFilaDeLaInstantanea.FindAllStringSubmatch(leerDoc(t, rutaDeLaInstantanea), -1) {
		if _, ya := out[m[1]]; ya {
			t.Fatalf("%s declara la nota de %s dos veces", rutaDeLaInstantanea, m[1])
		}
		out[m[1]] = coma(t, m[2], "la nota de hoy de "+m[1])
	}
	if len(out) != 17 {
		t.Fatalf("%s da %d notas de hoy y las dimensiones son 17.\n"+
			"  La nota de HOY es la que va en negrita en la tabla de autoevaluacion. Si la "+
			"tabla ha cambiado de forma, esta puerta estaria ponderando sobre un subconjunto "+
			"sin decirlo.", rutaDeLaInstantanea, len(out))
	}
	return out
}

// filaDelMarcador es lo que el marcador AFIRMA de cada dimension.
type filaDelMarcador struct {
	Peso   int
	Nota   float64
	Dentro bool
}

func filasDelMarcador(t *testing.T) map[string]filaDelMarcador {
	t.Helper()
	out := map[string]filaDelMarcador{}
	for _, m := range reFilaDelMarcador.FindAllStringSubmatch(leerDoc(t, rutaDelMarcador), -1) {
		if _, ya := out[m[1]]; ya {
			t.Fatalf("%s clasifica %s dos veces: una dimension esta dentro o fuera, no las dos",
				rutaDelMarcador, m[1])
		}
		p, err := strconv.Atoi(m[2])
		if err != nil {
			t.Fatalf("el peso de %s en %s (%q) no es un entero: %v", m[1], rutaDelMarcador, m[2], err)
		}
		out[m[1]] = filaDelMarcador{Peso: p, Nota: coma(t, m[3], "la nota de "+m[1]), Dentro: m[4] == "dentro"}
	}
	return out
}

// ponderado es la unica aritmetica de este fichero: suma de (peso x nota) sobre
// suma de pesos. Devuelve tambien el numerador porque es la cifra con dientes.
func ponderado(pesos map[string]int, notas map[string]float64, ids []string) (num float64, peso int) {
	for _, id := range ids {
		num += float64(pesos[id]) * notas[id]
		peso += pesos[id]
	}
	return unDecimal(num), peso
}

// cifraPublicada es una linea del bloque del marcador.
type cifraPublicada struct {
	Valor     float64
	Cardinal  int
	Peso      int
	Numerador float64
}

func leerCifra(t *testing.T, re *regexp.Regexp, doc, que string, ordenCardinalPrimero bool) cifraPublicada {
	t.Helper()
	m := re.FindStringSubmatch(doc)
	if m == nil {
		t.Fatalf("el bloque del marcador no publica %s con el patron %q.\n"+
			"  Sin esa linea la puerta no vigila esa cifra, y una cifra sin puerta se mueve "+
			"sola: es de donde viene la mitad de este repositorio.", que, re)
	}
	// La linea del subindice pone el valor primero y las otras dos el cardinal
	// primero, porque asi se leen mejor. El orden se declara aqui en vez de
	// escribir tres funciones casi iguales.
	iValor, iCardinal := 1, 2
	if ordenCardinalPrimero {
		iValor, iCardinal = 2, 1
	}
	c := cifraPublicada{Valor: coma(t, m[iValor], que)}
	n, err := strconv.Atoi(m[iCardinal])
	if err != nil {
		t.Fatalf("el cardinal de %s (%q) no es un entero: %v", que, m[iCardinal], err)
	}
	c.Cardinal = n
	p, err := strconv.Atoi(m[3])
	if err != nil {
		t.Fatalf("el peso de %s (%q) no es un entero: %v", que, m[3], err)
	}
	c.Peso = p
	c.Numerador = coma(t, m[4], "el numerador de "+que)
	return c
}

// contrastar compara los cuatro valores de una linea publicada con los cuatro
// que salen del dato. Sin tolerancia y en las dos direcciones.
func contrastar(t *testing.T, que string, pub cifraPublicada, num float64, peso, cardinal int) {
	t.Helper()
	calc := dosDecimales(num / float64(peso))
	if pub.Cardinal != cardinal {
		t.Errorf("%s: el marcador dice %d dimensiones y la tabla clasifica %d.\n"+
			"  Mover una dimension de un lado a otro cambia cuatro numeros publicados a la "+
			"vez, y este es el primero.", que, pub.Cardinal, cardinal)
	}
	if pub.Peso != peso {
		t.Errorf("%s: el marcador dice %d puntos de peso y los pesos de %s suman %d.\n"+
			"  Arreglo: los pesos NO se corrigen aqui. Si el de una dimension ha cambiado, "+
			"cambio en la rubrica y este documento lo copia.", que, pub.Peso, rutaDelDiseno, peso)
	}
	if math.Abs(pub.Numerador-num) > 1e-9 {
		t.Errorf("%s: el marcador publica un numerador de %.1f y el dato da %.1f.\n"+
			"  Es la cifra con dientes: el ponderado redondeado se traga un movimiento de una "+
			"decima en una dimension de peso bajo, y el numerador no. Si esto se ha movido, "+
			"una nota se ha movido.", que, pub.Numerador, num)
	}
	if math.Abs(pub.Valor-calc) > 1e-9 {
		t.Errorf("%s: el marcador publica %.2f y el dato da %.2f (%.1f / %d).\n"+
			"  Arreglo: actualizar docs/marcador.md en el mismo commit que mueve la nota. Y "+
			"si el numero ha SUBIDO sin que se haya construido nada, esta es la puerta que "+
			"existe para que se diga en voz alta.", que, pub.Valor, calc, num, peso)
	}
}

// TestElSubindiceDePlataformaLoComputaUnTestYNoUnaPersona es la puerta.
func TestElSubindiceDePlataformaLoComputaUnTestYNoUnaPersona(t *testing.T) {
	pesos := pesosDelDiseno(t)
	notas := notasDeLaInstantanea(t)
	filas := filasDelMarcador(t)

	// LAS DOS DIRECCIONES DE LA MEMBRESIA. Una dimension sin clasificar es como
	// se saca del subindice la que baja la media: nadie la mueve, simplemente
	// deja de estar en la tabla. Asi que el silencio rompe.
	var sinClasificar, sobrantes []string
	for id := range pesos {
		if _, ok := filas[id]; !ok {
			sinClasificar = append(sinClasificar, id)
		}
	}
	for id := range filas {
		if _, ok := pesos[id]; !ok {
			sobrantes = append(sobrantes, id)
		}
	}
	sort.Strings(sinClasificar)
	sort.Strings(sobrantes)
	if len(sinClasificar) > 0 {
		t.Errorf("el marcador no dice si %v estan dentro o fuera del subindice.\n"+
			"  Una dimension que desaparece de la tabla sale del subindice sin que nadie la "+
			"mueva, y eso es exactamente como se sube un indice sin construir nada.",
			sinClasificar)
	}
	if len(sobrantes) > 0 {
		t.Errorf("el marcador clasifica %v, que no estan en la rubrica de %s.\n"+
			"  O la rubrica ha perdido una dimension, o el marcador se ha inventado una.",
			sobrantes, rutaDelDiseno)
	}
	if t.Failed() {
		return
	}

	// Y LA COPIA NO PUEDE SEPARARSE DE SU ORIGEN. El marcador repite peso y
	// nota para que un lector de fuera pueda recalcular sin abrir otro fichero;
	// una copia que se mueve sola seria peor que no tenerla.
	var dentro, fuera, todas []string
	for id, f := range filas {
		todas = append(todas, id)
		if f.Peso != pesos[id] {
			t.Errorf("%s dice que %s pesa %d y %s dice %d.\n"+
				"  La copia del marcador existe para que se pueda recalcular sin abrir otro "+
				"fichero. El origen es la rubrica.", rutaDelMarcador, id, f.Peso, rutaDelDiseno, pesos[id])
		}
		if math.Abs(f.Nota-notas[id]) > 1e-9 {
			t.Errorf("%s dice que %s vale %.1f y %s dice %.1f.\n"+
				"  Una nota es un juicio y esta puerta no lo discute; lo que si exige es que "+
				"sea el MISMO juicio en los dos sitios.",
				rutaDelMarcador, id, f.Nota, rutaDeLaInstantanea, notas[id])
		}
		if f.Dentro {
			dentro = append(dentro, id)
		} else {
			fuera = append(fuera, id)
		}
	}
	sort.Strings(dentro)
	sort.Strings(fuera)
	sort.Strings(todas)
	if t.Failed() {
		return
	}

	numDentro, pesoDentro := ponderado(pesos, notas, dentro)
	numFuera, pesoFuera := ponderado(pesos, notas, fuera)
	numTodas, pesoTodas := ponderado(pesos, notas, todas)

	// La comprobacion barata de que la particion no se ha roto: el global no es
	// el promedio de los otros dos, es la fraccion de sus sumas.
	if unDecimal(numDentro+numFuera) != numTodas || pesoDentro+pesoFuera != pesoTodas {
		t.Fatalf("la particion no cuadra: dentro %.1f/%d + fuera %.1f/%d no da el total "+
			"%.1f/%d. Alguna dimension esta contada dos veces o ninguna",
			numDentro, pesoDentro, numFuera, pesoFuera, numTodas, pesoTodas)
	}

	doc := leerDoc(t, rutaDelMarcador)
	contrastar(t, "el subindice de plataforma",
		leerCifra(t, reSubindicePublicado, doc, "el subindice de plataforma", false),
		numDentro, pesoDentro, len(dentro))
	contrastar(t, "el global de las 17",
		leerCifra(t, reGlobalPublicado, doc, "el global de las 17", true),
		numTodas, pesoTodas, len(todas))
	contrastar(t, "las excluidas",
		leerCifra(t, reExcluidasPublicado, doc, "las excluidas", true),
		numFuera, pesoFuera, len(fuera))

	// EL HUECO, CON SU CARDINAL. Un subindice sin esta linea se lee como si
	// midiera el producto entero, y mide el 71,6 % del tablero.
	if len(fuera) == 0 {
		t.Errorf("el marcador no deja fuera ninguna dimension, asi que el subindice y el "+
			"global son el mismo numero y este documento no hace falta.\n"+
			"  O se han construido las cinco que faltaban, y entonces esto se celebra y se "+
			"retira la puerta, o el lector de %s no esta viendo los «fuera».", rutaDelMarcador)
	}

	t.Logf("marcador: subindice %.2f (%d dimensiones, %d de peso, numerador %.1f); "+
		"global %.2f (%d, %d, %.1f); excluidas %.2f (%d, %d, %.1f)",
		numDentro/float64(pesoDentro), len(dentro), pesoDentro, numDentro,
		numTodas/float64(pesoTodas), len(todas), pesoTodas, numTodas,
		numFuera/float64(pesoFuera), len(fuera), pesoFuera, numFuera)
	t.Logf("lo excluido pesa %d de %d, o sea el %.1f %% del tablero",
		pesoFuera, pesoTodas, 100*float64(pesoFuera)/float64(pesoTodas))
}

// CONTROL NEGATIVO DE LOS TRES LECTORES DE FILAS.
//
// El fallo probable de los tres es el mismo y esta a la vista en los propios
// documentos: hay tablas cuya primera celda EMPIEZA por el identificador y
// sigue con el nombre («D3 Cobertura por estratos...», «D5 Conectores WASM...»).
// Si el ancla se afloja, esas filas entran y la ponderacion suma dos veces la
// misma dimension con numeros de otra tabla.
func TestLosLectoresDelMarcadorNoCazanFilasDeOtrasTablas(t *testing.T) {
	casos := []struct {
		nombre   string
		re       *regexp.Regexp
		fuente   string
		esperado int
	}{
		{
			nombre: "el peso, con el identificador solo en su celda",
			re:     reFilaDelDiseno,
			fuente: "| D1 | Modelo | 12 | **9,7** | construido | porque |\n" +
				"| D3 | Cobertura | **6** | **9,5** | formato | porque |\n" +
				"| D3 Cobertura por estratos | 8 | **6** | baja porque |\n" +
				"| | **GLOBAL** | 109 | **9,60** | | |\n",
			esperado: 2,
		},
		{
			nombre: "la nota de hoy es la negrita, no la de diseno",
			re:     reFilaDeLaInstantanea,
			fuente: "| D1 | Modelo | 9,7 | **9,0** | porque |\n" +
				"| D5 | Conectores | 9,5 | **2,0** | nada construido |\n" +
				"| D5 Conectores WASM | 7 | no hay ABI | E6 |\n" +
				"| Paquetes Go | **60** |\n",
			esperado: 2,
		},
		{
			nombre: "la membresia, con sus cinco celdas",
			re:     reFilaDelMarcador,
			fuente: "| D1 | Modelo | 12 | 9,0 | dentro |\n" +
				"| D5 | Conectores | 7 | 2,0 | fuera |\n" +
				"| D5 Conectores WASM | 7 | no hay ABI ni host | E6 |\n" +
				"| D1 | 8,0 | **9,0** | decia que faltaban dos primitivas |\n",
			esperado: 2,
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got := len(c.re.FindAllStringSubmatch(c.fuente, -1))
			if got != c.esperado {
				t.Errorf("ha casado %d filas y esperaba %d: el ancla del identificador se ha "+
					"aflojado y la ponderacion estaria sumando filas de otras tablas", got, c.esperado)
			}
		})
	}
}

// CONTROL NEGATIVO DE LAS TRES CIFRAS PUBLICADAS.
//
// Su fallo probable es el de siempre en este repositorio: cazar cualquier numero
// parecido del documento. Las tres exigen los dos marcadores, asi que la misma
// frase fuera del bloque no vale.
func TestLasCifrasDelMarcadorSoloValenDentroDeSuBloque(t *testing.T) {
	linea := "- **Subíndice de plataforma open source, publicable: 8,32**, sobre " +
		"**12 dimensiones** y **78 puntos de peso**, con numerador **649,2**.\n"
	casos := []struct {
		nombre string
		fuente string
		casa   bool
	}{
		{"dentro del bloque", "<!-- marcador:inicio -->\n" + linea + "<!-- marcador:fin -->\n", true},
		{"fuera del bloque no vale", "en la introduccion dice " + linea, false},
		{"sin el cierre tampoco", "<!-- marcador:inicio -->\n" + linea, false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if casa := reSubindicePublicado.MatchString(c.fuente); casa != c.casa {
				t.Errorf("ha casado %t y esperaba %t: la puerta estaria vigilando un numero "+
					"cualquiera de %s", casa, c.casa, rutaDelMarcador)
			}
		})
	}
}

// EL PESO DE LO EXCLUIDO ES 31 Y NO 32, Y ESO SE COMPRUEBA SOLO.
//
// El encargo que pidio este subindice decia que las cinco excluidas «pesan 32 de
// 109». La rubrica da 31. El 32 sale de leer D8 en la columna «antes» de la
// tabla de movimiento de D-20 (6) en vez de en la «ahora» (5), y por eso no se
// ve al recontar: la suma esta bien hecha sobre un sumando de otra columna.
//
// Esto no es una anecdota que merezca un test propio por si misma: merece uno
// porque es la unica cifra del marcador que un lector podria dar por buena sin
// mirar, y porque el error tenia la forma de un dato verificable.
func TestElPesoDeLoExcluidoSaleDeLaRubricaYNoDeUnaNotaDeEncargo(t *testing.T) {
	pesos := pesosDelDiseno(t)
	filas := filasDelMarcador(t)
	suma := 0
	var ids []string
	for id, f := range filas {
		if !f.Dentro {
			suma += pesos[id]
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	total := 0
	for _, p := range pesos {
		total += p
	}
	if total != 109 {
		t.Fatalf("los pesos de la rubrica suman %d y la rubrica declara 109: si la suma ha "+
			"cambiado, un movimiento de pesos cambio tambien el denominador y no se puede "+
			"comparar con nada anterior", total)
	}
	var partes []string
	for _, id := range ids {
		partes = append(partes, fmt.Sprintf("%s %d", id, pesos[id]))
	}
	t.Logf("lo excluido: %s = %d de %d", strings.Join(partes, " + "), suma, total)
	if suma >= total {
		t.Fatalf("lo excluido pesa %d de %d: no queda nada dentro y el subindice no mide nada",
			suma, total)
	}
}

// LA DESCOMPOSICION DEL SUBINDICE, BAJO PUERTA. Cuanto de la ventaja es
// ALCANCE y cuanto es MERITO.
//
// # Por que hacia falta, y es la misma trampa de siempre
//
// El subindice tiene puerta desde que se publico. Su descomposicion, que es la
// unica frase que impide leerlo mal, NO LA TENIA: era prosa al lado de un
// numero vigilado. O sea la cuarta forma de la afirmacion acompanada, y la
// peor: el numero se corrige solo el dia que alguien mueve una nota, y la
// explicacion se queda diciendo que el 82 % de la ventaja es definicion sobre
// una distancia que ya no es esa. Un dato falso se contrasta; una explicacion
// falsa se cree.
//
// Regla que la trae, y vale para todo subindice futuro: UN NUMERO QUE NO DICE
// DE DONDE VIENE SU VENTAJA ES UN NUMERO QUE LA ESCONDE. Asi que la
// descomposicion se publica SIEMPRE, no una vez, y se computa del dato como
// las tres cifras de arriba.
//
// # Como se computa, y por que se puede
//
// La distancia entre el global honesto de partida y el subindice publicado se
// parte en dos causas independientes:
//
//	el DENOMINADOR  dejar cinco dimensiones fuera de la foto (alcance)
//	las NOTAS       lo que se construyo entre las dos fechas (merito)
//
// Y se miden por separado moviendo una cosa cada vez, que es lo unico que
// separa una causa de la otra. Las notas viejas no hacen falta guardarlas
// aparte: la tabla de las cuatro notas que se movieron las tiene, y las trece
// que no se movieron son, por definicion, las de hoy. Se reconstruyen de ahi,
// o sea del mismo documento que publica el resultado, que es lo que hace que
// no puedan separarse.
//
// # LOS MOVIMIENTOS SE DERIVAN, NO SE ESCRIBEN
//
// La tabla publica cuatro cifras y tres movimientos. Los tres movimientos son
// restas de las cuatro cifras, asi que esta puerta EXIGE que lo sean: un
// movimiento tecleado a mano es sospechoso entero, no solo en su ultimo
// decimal, porque es el sitio por donde una explicacion se separa de su dato.

// reFilaMovida lee la fila de una nota que se movio: la vieja y la de hoy. La
// de hoy va en negrita y la vieja no, igual que en la instantanea, y esa
// asimetria es la que impide que la fila se lea al reves.
var reFilaMovida = regexp.MustCompile(
	`(?m)^\|\s*(D\d{1,2})\s*\|\s*(\d{1,2},\d)\s*\|\s*\*\*(\d{1,2},\d)\*\*\s*\|`)

// reDescomposicion lee las cuatro cifras y los tres movimientos de la tabla.
var reDescomposicion = regexp.MustCompile(
	`(?s)\| nada \(punto de partida: global, notas del 26-08\) \| (\d,\d\d) \|` +
		`.*?\| \*\*s.lo el denominador\*\*[^|]*\| (\d,\d\d) \| \*\*\+(\d,\d\d)\*\* \|` +
		`.*?\| \*\*s.lo las notas\*\*[^|]*\| (\d,\d\d) \| \*\*\+(\d,\d\d)\*\* \|` +
		`.*?\| las dos cosas \(el sub.ndice publicado\) \| (\d,\d\d) \| \+(\d,\d\d) \|`)

// notaDe pondera sobre las dimensiones que el filtro admite, REUSANDO la
// aritmetica que ya hay en este fichero en vez de escribir una segunda.
//
// Una segunda implementacion de la misma suma es exactamente lo que se pide no
// hacer en el resto del repositorio (medir con un segundo parser es medir otra
// cosa el dia que los dos se separen), y aqui la tentacion era grande porque el
// filtro cambia y la suma no.
func notaDe(t *testing.T, pesos map[string]int, notas map[string]float64,
	admite func(string) bool) float64 {

	t.Helper()
	var ids []string
	for id := range pesos {
		if admite(id) {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids) // el orden no cambia la suma, pero un log reproducible si
	num, peso := ponderado(pesos, notas, ids)
	if peso == 0 {
		t.Fatalf("el filtro no ha admitido ni una dimension: la ponderacion seria una "+
			"division por cero y el cero saldria como si fuera una nota. Dimensiones "+
			"disponibles: %d", len(pesos))
	}
	return num / float64(peso)
}

// notasDel26 reconstruye las notas de la fecha de partida: las que la tabla
// dice que se movieron con su valor viejo, y las restantes iguales a las de
// hoy.
//
// SI LA TABLA CAMBIA DE FORMA Y NO DA NINGUNA FILA, SE PARA. Sin esa guarda
// esta puerta reconstruiria un pasado identico al presente, daria una
// descomposicion de cero y la llamaria verde, que es el fallo que mas duele
// aqui: el verde vendria de lo que falta.
func notasDel26(t *testing.T, hoy map[string]float64) map[string]float64 {
	t.Helper()
	out := map[string]float64{}
	for id, n := range hoy {
		out[id] = n
	}
	movidas := 0
	for _, m := range reFilaMovida.FindAllStringSubmatch(leerDoc(t, rutaDelMarcador), -1) {
		id := m[1]
		vieja := coma(t, m[2], "la nota vieja de "+id)
		nueva := coma(t, m[3], "la nota de hoy de "+id)
		if _, hay := hoy[id]; !hay {
			t.Fatalf("la tabla de movimientos habla de %s y la instantanea no lo tiene", id)
		}
		// LA FILA TIENE QUE CASAR CON LA INSTANTANEA EN SU MITAD DE HOY. Sin
		// esto, la tabla podria decir que una dimension esta hoy en 6,5
		// mientras la instantanea dice otra cosa, y las dos serian verdes por
		// separado porque nadie las compara.
		if nueva != hoy[id] {
			t.Errorf("la tabla de movimientos dice que %s esta hoy en %.1f y %s dice %.1f.\n"+
				"  Son el mismo numero contado dos veces, asi que una de las dos esta vieja.",
				id, nueva, rutaDeLaInstantanea, hoy[id])
		}
		if vieja == nueva {
			t.Errorf("%s sale en la tabla de las que SE MOVIERON con %.1f y %.1f, que es el "+
				"mismo numero. Una fila que no mueve nada infla el recuento de trabajo hecho",
				id, vieja, nueva)
		}
		out[id] = vieja
		movidas++
	}
	if movidas == 0 {
		t.Fatalf("la tabla de movimientos de %s no ha dado ni una fila.\n"+
			"  Esta puerta estaria comparando las notas de hoy contra si mismas, o sea dando "+
			"una descomposicion de cero y llamandola verde.", rutaDelMarcador)
	}
	if len(out) != 17 {
		t.Fatalf("las notas del 26-08 salen %d y las dimensiones son 17", len(out))
	}
	t.Logf("notas del 26-08 reconstruidas: %d movidas, %d heredadas de hoy",
		movidas, len(out)-movidas)
	return out
}

func TestLaDescomposicionDelSubindiceSaleDelDatoYNoDeLaProsa(t *testing.T) {
	pesos := pesosDelDiseno(t)
	hoy := notasDeLaInstantanea(t)
	viejas := notasDel26(t, hoy)
	filas := filasDelMarcador(t)

	dentro := func(id string) bool { return filas[id].Dentro }
	todas := func(string) bool { return true }

	partida := notaDe(t, pesos, viejas, todas)          // global, notas viejas
	soloDenominador := notaDe(t, pesos, viejas, dentro) // subindice, notas viejas
	soloNotas := notaDe(t, pesos, hoy, todas)           // global, notas de hoy
	lasDos := notaDe(t, pesos, hoy, dentro)             // el subindice publicado

	m := reDescomposicion.FindStringSubmatch(leerDoc(t, rutaDelMarcador))
	if m == nil {
		t.Fatalf("no encuentro la tabla de descomposicion en %s.\n"+
			"  TODO SUBINDICE SALE CON SU DESCOMPOSICION AL LADO: cuanto de su ventaja es\n"+
			"  alcance (el denominador) y cuanto es merito (las notas). Un numero que no dice\n"+
			"  de donde viene su ventaja es un numero que la esconde, y esta puerta existe\n"+
			"  para que no se pueda publicar sin ella.", rutaDelMarcador)
	}
	pPartida := coma(t, m[1], "el punto de partida publicado")
	pDen := coma(t, m[2], "el subindice con las notas viejas")
	pMovDen := coma(t, m[3], "el movimiento del denominador")
	pNotas := coma(t, m[4], "el global con las notas de hoy")
	pMovNotas := coma(t, m[5], "el movimiento de las notas")
	pLasDos := coma(t, m[6], "el subindice publicado")
	pMovLasDos := coma(t, m[7], "el movimiento total")

	// LAS CUATRO CIFRAS, CONTRA EL DATO, CON IGUALDAD EXACTA.
	for _, c := range []struct {
		que              string
		computado, dicho float64
	}{
		{"el punto de partida (global con las notas del 26-08)",
			dosDecimales(partida), pPartida},
		{"solo el denominador (subindice con las notas del 26-08)",
			dosDecimales(soloDenominador), pDen},
		{"solo las notas (global con las notas de hoy)",
			dosDecimales(soloNotas), pNotas},
		{"las dos cosas (el subindice publicado)",
			dosDecimales(lasDos), pLasDos},
	} {
		if c.computado != c.dicho {
			t.Errorf("%s: %s publica %.2f y el dato da %.2f.\n"+
				"  La descomposicion se computa de los pesos de %s, de las notas de %s y de\n"+
				"  la tabla de movimientos del propio marcador. Si no cuadra, o una nota se\n"+
				"  movio sin pasar por la tabla, o la tabla se quedo vieja.",
				c.que, rutaDelMarcador, c.dicho, c.computado,
				rutaDelDiseno, rutaDeLaInstantanea)
		}
	}

	// LOS TRES MOVIMIENTOS SE DERIVAN DEL DATO SIN REDONDEAR, Y ESO NO ES UN
	// DETALLE: ES LA CORRECCION DE ESTA PUERTA EL DIA QUE NACIO.
	//
	// La primera version restaba las cifras YA REDONDEADAS de la tabla y
	// acusaba al documento de dos movimientos falsos: +0,29 donde la resta de
	// los redondeos da 0,28, y +2,20 donde da 2,19. El documento tenia razon y
	// la puerta no: 6,4147 - 6,1257 = 0,2890, que redondeado son 0,29. LA
	// DIFERENCIA DE DOS REDONDEOS NO ES EL REDONDEO DE LA DIFERENCIA, y quien
	// se equivocaba era quien medía.
	//
	// Se deja escrito porque el fallo tenia la forma peligrosa: acusaba con dos
	// cifras concretas y un mensaje convincente, y las dos acusaciones iban en
	// la direccion de «te has redondeado a favor», que es justo la que uno esta
	// predispuesto a creerse de si mismo. Una puerta que acusa en falso gasta
	// la misma credibilidad que una pantalla que acusa en falso.
	//
	// Y QUEDA LA MITAD QUE SI ERA CIERTA: un lector que reste las dos cifras
	// impresas al lado obtiene 0,28 y no 0,29. Eso hay que decirlo en la tabla,
	// y esta puerta exige la nota que lo dice.
	for _, c := range []struct {
		que                              string
		computado, dicho, restaRedondeos float64
	}{
		{"del denominador", dosDecimales(soloDenominador - partida), pMovDen,
			dosDecimales(pDen - pPartida)},
		{"de las notas", dosDecimales(soloNotas - partida), pMovNotas,
			dosDecimales(pNotas - pPartida)},
		{"total", dosDecimales(lasDos - partida), pMovLasDos,
			dosDecimales(pLasDos - pPartida)},
	} {
		if c.computado != c.dicho {
			t.Errorf("el movimiento %s se publica como +%.2f y el dato da %.2f.\n"+
				"  TODO MOTIVO QUE CITE UN CARDINAL LO DERIVA, NO LO ESCRIBE. Un movimiento "+
				"tecleado a mano es sospechoso entero, no solo en su ultimo decimal.",
				c.que, c.dicho, c.computado)
		}
		// Y LA DISTANCIA CON LA RESTA INGENUA, QUE ES LA QUE HARA EL LECTOR.
		// Un centesimo lo explica el redondeo; mas de un centesimo es un
		// numero que viene de otro sitio.
		if d := math.Abs(c.computado - c.restaRedondeos); d > 0.0101 {
			t.Errorf("el movimiento %s (+%.2f) se separa %.2f de restar las dos cifras "+
				"publicadas a su lado (%.2f).\n"+
				"  Un centesimo lo explica el redondeo y se dice en la tabla. Mas de un "+
				"centesimo es un numero que viene de otro sitio.", c.que, c.dicho, d,
				c.restaRedondeos)
		}
	}

	// LA NOTA DEL REDONDEO TIENE QUE ESTAR. Sin ella, un lector que reste las
	// dos cifras impresas obtiene otra cosa y concluye que la tabla esta mal,
	// que es la unica forma de perder a un lector que SI comprueba.
	if !strings.Contains(leerDoc(t, rutaDelMarcador), "diferencia de dos redondeos") {
		t.Errorf("%s publica movimientos que no salen de restar las cifras de su lado y no "+
			"explica por que.\n"+
			"  Se redondea la resta sin redondear (0,2890 -> 0,29) y las cifras se redondean "+
			"aparte, asi que restar lo impreso puede dar un centesimo menos. Quien comprueba "+
			"merece leerlo antes de pensar que sobra un decimal.", rutaDelMarcador)
	}

	// Y EL REPARTO EN PORCENTAJE, que es la frase que la gente se lleva. Se
	// computa aqui y se registra: la tabla lo cita en prosa y la prosa es
	// justo lo que esta puerta ha venido a dejar de creerse.
	total := pLasDos - pPartida
	if total <= 0 {
		t.Fatalf("la distancia total publicada es %.2f: sin distancia no hay nada que "+
			"descomponer y esta tabla no deberia existir", total)
	}
	t.Logf("descomposicion computada: partida %.4f | solo denominador %.4f (+%.4f) | "+
		"solo notas %.4f (+%.4f) | las dos %.4f (+%.4f)",
		partida, soloDenominador, soloDenominador-partida, soloNotas, soloNotas-partida,
		lasDos, lasDos-partida)
	t.Logf("reparto: alcance %.1f %%, merito %.1f %%, interaccion %.1f %%",
		100*(pDen-pPartida)/total, 100*(pNotas-pPartida)/total,
		100*(pLasDos-pDen-pNotas+pPartida)/total)
}
