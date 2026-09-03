package plazum

import (
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// EL ESTADO DEL PLAN VA CON DOS NUMEROS, Y LOS DOS SALEN DEL ARBOL.
//
// # De donde sale la regla
//
// Decision del 03-09-2026: el contador de casillas NO mide el trabajo de
// corpus. Una campana que escribio 39 relojes, tres paquetes nuevos, dos
// pantallas y el armazon visual movio CUATRO casillas, porque las casillas
// estan escritas como puertas y una puerta no se cierra a medias. Marcar a ojo
// habria dado veinte casillas y un producto peor.
//
// Asi que el estado se publica con dos cifras que dicen cosas distintas:
// casillas cerradas (cuanto del plan esta terminado de verdad) y relojes
// escritos (cuanto corpus existe). Ninguna de las dos sola dice la verdad.
//
// # Y las dos con puerta, porque un numero sin puerta se mueve solo
//
// Es la misma doctrina que el porcentaje de la v1 y que los cuatro numeros del
// README. Con igualdad exacta y en los dos sentidos: que una cifra BAJE tiene
// que romper igual que si sube, porque un conjunto que encoge sin que nadie lo
// note es la otra mitad del mismo fallo.

const rutaDeEtapas = "ETAPAS.md"

// Las casillas se cuentan ANCLADAS al principio de linea. Sin el ancla, una
// casilla citada dentro del texto de otra contaria como casilla, y este fichero
// esta lleno de prosa que habla de casillas.
var (
	reCerrada = regexp.MustCompile(`(?m)^- \[x\] `)
	reAbierta = regexp.MustCompile(`(?m)^- \[ \] `)
)

// El bloque que el plan AFIRMA. Se lee de ETAPAS.md y no de una constante de Go
// porque ETAPAS.md es lo que se mira para saber por donde va el proyecto: si el
// numero del plan y el del arbol se separan, el que engana es el del plan.
var (
	reCasillasDeclaradas = regexp.MustCompile(
		`(?s)<!-- estado:inicio -->.*?\*\*(\d+) de (\d+) casillas\*\*.*?<!-- estado:fin -->`)
	reRelojesDeclarados = regexp.MustCompile(
		`(?s)<!-- estado:inicio -->.*?\*\*(\d+) relojes escritos\*\*.*?<!-- estado:fin -->`)
)

func leerEtapas(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(rutaDeEtapas) // #nosec G304 -- ruta constante del repositorio
	if err != nil {
		t.Fatalf("no puedo leer %s: %v", rutaDeEtapas, err)
	}
	return string(b)
}

// relojesDelCorpus cuenta las obligaciones con reloj de TODO el corpus, con el
// cargador del producto. No es lo mismo que los hitos (un plazo escalonado
// declara varios) ni que la cobertura de la v1 (que se restringe a doce
// marcos): es cuanto corpus con reloj existe, que es la segunda cifra del
// estado.
func relojesDelCorpus(t *testing.T) int {
	t.Helper()
	ps, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatalf("cargar el corpus: %v", err)
	}
	n := 0
	for _, p := range ps {
		for _, o := range p.Obligaciones {
			if o.Temporalidad != nil {
				n++
			}
		}
	}
	if n < 100 {
		t.Fatalf("el corpus cargado trae %d relojes: este recorrido esta midiendo el vacio", n)
	}
	return n
}

func TestElEstadoDelPlanLoComputaUnTestYNoUnaPersona(t *testing.T) {
	texto := leerEtapas(t)
	cerradas := len(reCerrada.FindAllString(texto, -1))
	abiertas := len(reAbierta.FindAllString(texto, -1))
	total := cerradas + abiertas
	if total < 100 {
		t.Fatalf("ETAPAS.md tiene %d casillas y hoy son mas de cien: el patron ha dejado de "+
			"casar y esta puerta estaria midiendo el vacio", total)
	}

	m := reCasillasDeclaradas.FindStringSubmatch(texto)
	if m == nil {
		t.Fatalf("ETAPAS.md no trae el bloque de estado entre <!-- estado:inicio --> y "+
			"<!-- estado:fin --> con «**N de M casillas**».\n"+
			"  Sin ese bloque, el estado del plan vuelve a contarse a mano. Hoy serian "+
			"**%d de %d casillas**.", cerradas, total)
	}
	decCerradas, _ := strconv.Atoi(m[1])
	decTotal, _ := strconv.Atoi(m[2])
	if decCerradas != cerradas || decTotal != total {
		t.Errorf("ETAPAS.md declara %d de %d casillas y el fichero tiene %d de %d.\n"+
			"  Arreglo: actualizar el bloque de estado en el mismo commit que marca la "+
			"casilla. Si el TOTAL ha subido, es que se han abierto casillas nuevas, y eso "+
			"tambien es informacion.", decCerradas, decTotal, cerradas, total)
	}

	relojes := relojesDelCorpus(t)
	r := reRelojesDeclarados.FindStringSubmatch(texto)
	if r == nil {
		t.Fatalf("ETAPAS.md no dice cuantos relojes hay escritos, con «**N relojes "+
			"escritos**» dentro del bloque de estado.\n"+
			"  Es la segunda cifra, y existe porque la primera no mide el trabajo de "+
			"corpus: hoy serian **%d relojes escritos**.", relojes)
	}
	decRelojes, _ := strconv.Atoi(r[1])
	if decRelojes != relojes {
		t.Errorf("ETAPAS.md declara %d relojes escritos y el corpus tiene %d.\n"+
			"  Las dos cifras del estado se mueven en commits distintos a proposito: "+
			"escribir corpus sube esta y no la otra, y cerrar una puerta sube la otra y "+
			"no esta.", decRelojes, relojes)
	}

	t.Logf("estado del plan: %d de %d casillas cerradas, %d relojes escritos",
		cerradas, total, relojes)
}

// CONTROL NEGATIVO DE LOS DOS CONTADORES.
//
// El de casillas tiene un fallo probable muy concreto: contar casillas citadas
// dentro de la prosa de otra, que en este fichero abundan. El de la declaracion
// tiene el de siempre: casar cualquier numero del documento.
func TestLosContadoresDelEstadoCuentanLoSuyoYNoLoDelVecino(t *testing.T) {
	muestra := "- [x] una cerrada\n" +
		"- [ ] una abierta\n" +
		"  - [x] una anidada, que NO cuenta: el ancla es la columna cero\n" +
		"texto que menciona - [x] dentro de una frase, y tampoco cuenta\n" +
		"- [x] otra cerrada\n"
	if got := len(reCerrada.FindAllString(muestra, -1)); got != 2 {
		t.Errorf("el contador de cerradas ha visto %d y son 2: esta cazando casillas que "+
			"viven dentro de la prosa de otra", got)
	}
	if got := len(reAbierta.FindAllString(muestra, -1)); got != 1 {
		t.Errorf("el contador de abiertas ha visto %d y es 1", got)
	}

	casos := []struct {
		nombre string
		fuente string
		casa   bool
	}{
		{"el bloque, con sus dos numeros",
			"<!-- estado:inicio -->\n**57 de 135 casillas**\n<!-- estado:fin -->\n", true},
		{"fuera del bloque no vale",
			"por ahi arriba dice **57 de 135 casillas** y no esta en el bloque\n", false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if casa := reCasillasDeclaradas.MatchString(c.fuente); casa != c.casa {
				t.Errorf("ha casado %t y esperaba %t: la puerta estaria vigilando un numero "+
					"cualquiera de ETAPAS.md", casa, c.casa)
			}
		})
	}
}
