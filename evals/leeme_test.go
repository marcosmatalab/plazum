package evals

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// LA TABLA DEL README Y LOS FICHEROS QUE HAY, EN LAS DOS DIRECCIONES.
//
// # Por que esto necesita puerta y no basta con la regla escrita
//
// `CLAUDE.md` dice desde el 08-09-2026 que toda pieza de IA nace con su conjunto
// dorado o no entra, y `evals/README.md` lleva la tabla de los tres conjuntos con
// su cadencia y su estado. Una regla en prosa y una tabla a mano es exactamente
// la afirmacion acompanada: el dia que entre la pieza 2, la tabla puede seguir
// diciendo «por escribir» de algo que ya esta, o dar por escrito algo que no.
//
// # Las dos direcciones, y la segunda es la que se olvida
//
//	1. Todo conjunto que la tabla da por ESCRITO existe y carga con el arnes del
//	   producto. Sin esto, la tabla puede prometer un conjunto que no hay, que es
//	   lo que convierte «publicamos la precision de nuestra IA» en una frase.
//	2. Todo directorio de conjunto que existe esta en la tabla. Sin esto, un
//	   conjunto escrito y no anunciado no cuenta para nadie, y la tabla se queda
//	   diciendo «por escribir» de algo que lleva semanas hecho.
//
// # Lo que esta puerta NO mira, dicho
//
// La PRECISION. Que el conjunto pase es cosa de `TestElConjuntoDoradoDeCitasPasaEntero`
// y de los suyos; esto solo comprueba que lo que se anuncia y lo que hay son lo
// mismo. Y no puede exigir que exista un conjunto por cada pieza de IA, porque
// hoy no hay forma mecanica de enumerar «las piezas de IA»: eso lo sostiene la
// regla de CLAUDE.md y la revision, no un test, y decirlo es la mitad honesta.

const rutaDelLeemeDeEvals = "README.md"

// Fila de la tabla, leida por su columna de DIRECTORIO:
//
//	| conjunto | `citas/` | cadencia | estado |
//
// Se ancla en el directorio y no en el nombre a proposito. El nombre es prosa
// («extraccion de obligaciones», con tilde y espacios) y cambia por gusto; el
// directorio es lo que hay en el disco, que es justo lo que esta puerta compara.
var reFilaDeConjunto = regexp.MustCompile(
	`(?m)^\|[^|]*\|\s*` + "`" + `([a-z]+)/` + "`" + `\s*\|([^|]*)\|([^|]*)\|`)

func TestElLeemeDeEvalsYElArbolSeApuntanEnLasDosDirecciones(t *testing.T) {
	b, err := os.ReadFile(rutaDelLeemeDeEvals) // #nosec G304 -- ruta constante del repositorio
	if err != nil {
		t.Fatalf("no puedo leer %s: %v", rutaDelLeemeDeEvals, err)
	}
	filas := reFilaDeConjunto.FindAllStringSubmatch(string(b), -1)
	if len(filas) < 3 {
		t.Fatalf("la tabla de conjuntos de %s trae %d filas y hoy son tres. O la tabla ha "+
			"encogido, o el patron ha dejado de casar y esto esta midiendo el vacio",
			rutaDelLeemeDeEvals, len(filas))
	}

	declarados := map[string]bool{}
	escritos := map[string]bool{}
	for _, f := range filas {
		nombre := strings.TrimSpace(f[1])
		declarados[nombre] = true
		// «por escribir» en el estado es la unica forma de decir que aun no esta.
		if !strings.Contains(strings.ToLower(f[3]), "por escribir") {
			escritos[nombre] = true
		}
	}
	if len(escritos) == 0 {
		t.Fatal("la tabla no da por escrito ni un conjunto: o el hito de publicar la precision " +
			"de la IA no lo sostiene nada, o el patron del estado ha dejado de casar")
	}

	// DIRECCION 1: lo que la tabla da por escrito, existe y carga.
	for nombre := range escritos {
		ruta := filepath.Join(nombre, "dorados.json")
		if _, err := os.Stat(ruta); err != nil {
			t.Errorf("%s da por ESCRITO el conjunto %q y no hay ningun %s.\n"+
				"  Una tabla que promete un conjunto que no existe convierte «publicamos la "+
				"precision de nuestra IA» en una frase.", rutaDelLeemeDeEvals, nombre, ruta)
			continue
		}
		c, err := Cargar(ruta)
		if err != nil {
			t.Errorf("el conjunto %q existe y no carga con el arnes del producto: %v.\n"+
				"  Un conjunto que no carga no mide nada, y ademas no se distingue de uno que "+
				"mide y aprueba", nombre, err)
			continue
		}
		if len(c.Casos) == 0 {
			t.Errorf("el conjunto %q carga y no tiene ni un caso: es un conjunto vacio con "+
				"cara de conjunto", nombre)
		}
	}

	// DIRECCION 2, la que se olvida: lo que hay, esta en la tabla.
	entradas, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("no puedo leer el directorio de evals: %v", err)
	}
	for _, e := range entradas {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(e.Name(), "dorados.json")); err != nil {
			continue // un directorio sin conjunto no es un conjunto
		}
		if !declarados[e.Name()] {
			t.Errorf("existe el conjunto %s/dorados.json y la tabla de %s no lo nombra.\n"+
				"  Es la direccion que se olvida: un conjunto escrito y no anunciado no cuenta "+
				"para nadie, y la tabla se queda diciendo «por escribir» de algo que lleva "+
				"semanas hecho.", e.Name(), rutaDelLeemeDeEvals)
		}
		if !escritos[e.Name()] {
			t.Errorf("existe el conjunto %s/dorados.json y la tabla de %s todavia lo da por "+
				"escribir.", e.Name(), rutaDelLeemeDeEvals)
		}
	}
	t.Logf("conjuntos: %d declarados, %d escritos", len(declarados), len(escritos))
}

// CONTROL NEGATIVO DEL PATRON DE FILA.
//
// El fallo probable es el de siempre en un README lleno de tablas: casar filas de
// otra. Este fichero tiene ademas una tabla de motivos y otra de formato.
func TestElPatronDeLaTablaDeConjuntosNoCazaOtrasTablas(t *testing.T) {
	casos := []struct {
		nombre string
		fuente string
		quiero string
	}{
		{"la fila de un conjunto, leida por su directorio",
			"| **citas** (el verificador) | `citas/` | **cada PR** | **28 casos** |\n", "citas"},
		{"la tabla de motivos, que no tiene columna de directorio",
			"| hash_ilegible | el hash no se puede leer | descartada |\n", ""},
		{"una celda que parece un directorio y no lo es",
			"| algo | `docs/guia.md` | x | y |\n", ""},
		{"un directorio con mayusculas o guiones no es nombre de conjunto",
			"| algo | `Extraccion-2/` | x | y |\n", ""},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			m := reFilaDeConjunto.FindStringSubmatch(c.fuente)
			got := ""
			if m != nil {
				got = strings.TrimSpace(m[1])
			}
			if got != c.quiero {
				t.Errorf("ha sacado %q y esperaba %q: la puerta estaria leyendo la tabla "+
					"equivocada", got, c.quiero)
			}
		})
	}
}
