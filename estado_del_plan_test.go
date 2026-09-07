package plazum

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

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

	// LA DERIVA. Ver TestElEstadoDelPlanPublicaSuDeriva.
	//
	// El ancla es la instantanea ANTERIOR, que es un hecho del pasado y no
	// caduca; todo lo demas se deriva de ella y del arbol de hoy.
	reInstantaneaAnterior = regexp.MustCompile(
		`Instantanea anterior: \*\*(\d+) de (\d+), el (\d{2}-\d{2}-\d{4})\*\*`)
	reFechaDeEsta = regexp.MustCompile(`Esta es del \*\*(\d{2}-\d{2}-\d{4})\*\*`)
	reDeriva      = regexp.MustCompile(
		`\*\*(\d+) dias\*\* despues: \*\*\+(\d+) cerradas\*\* y \*\*\+(\d+) abiertas\*\*, ` +
			`\*\*([\d,]+) al dia contra ([\d,]+)\*\*, y las pendientes de \*\*(\d+) a (\d+)\*\*`)
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

// EL PLAN CRECE MAS DEPRISA DE LO QUE SE CIERRA, Y ESO NO SE VE EN NINGUN SITIO.
//
// # El hueco que cierra
//
// El bloque de estado publica CERRADAS y TOTAL, y las dos con puerta. Lo que no
// publica es la DERIVA: cuantas se han abierto y cuantas se han cerrado desde la
// instantanea anterior. Y esa es la cifra que dice si el plan converge.
//
// Medido del arbol el 08-09-2026, no estimado: **31 de 100 el 25-08-2026** y
// **72 de 142 hoy**, o sea **+41 cerradas y +42 abiertas en catorce dias**, 2,9
// al dia contra 3,0. Las pendientes han pasado de **69 a 70**.
//
// **Eso no es malo por si solo** y hay que decirlo, o esta cifra se lee como un
// reproche: D-22 abrio siete casillas porque destapo trabajo real que ya estaba
// dentro de otras dos, y el trabajo estaba igual antes de contarlo. Lo que si es
// malo es que **no se viera**: con dos cifras que solo suben, un re-corte que
// ensancha el plan se lee igual que uno que lo estrecha.
//
// # Por que el ancla se escribe y todo lo demas se deriva
//
// La instantanea anterior es un HECHO DEL PASADO: no puede quedarse vieja,
// porque describe un dia que ya paso. Los deltas, las tasas y las pendientes
// salen de ella y del arbol de hoy, asi que ninguno se puede escribir a mano ni
// quedarse viejo.
//
// LO QUE ESTA PUERTA NO MIRA, dicho: que el ancla sea CIERTA. Se puede escribir
// «31 de 100» de un dia en que fueran otros, y esta puerta no lo sabria. Se
// contrasta con una orden, que es la que produjo el numero de hoy:
//
//	c=$(git log --format=%h --until="2026-08-25 23:59:59" -1 -- ETAPAS.md)
//	git show $c:ETAPAS.md | grep -cE '^- \[x\] '
//
// Y el ancla de hoy salio de ahi y no de una nota: la primera version de este
// bloque decia «33 de 100» y «pendientes de 67 a 70», que era lo que yo tenia
// apuntado; el arbol dice 31 y 69. La diferencia no cambia la conclusion, y esa
// es justo la razon por la que habria colado.
func TestElEstadoDelPlanPublicaSuDeriva(t *testing.T) {
	texto := leerEtapas(t)
	cerradas := len(reCerrada.FindAllString(texto, -1))
	total := cerradas + len(reAbierta.FindAllString(texto, -1))

	ant := reInstantaneaAnterior.FindStringSubmatch(texto)
	if ant == nil {
		t.Fatalf("ETAPAS.md no declara la instantanea anterior con «Instantanea anterior: " +
			"**N de M, el DD-MM-AAAA**».\n" +
			"  Sin ancla no hay deriva, y sin deriva las dos cifras del estado solo pueden " +
			"subir: un re-corte que ensancha el plan se lee igual que uno que lo estrecha.")
	}
	esta := reFechaDeEsta.FindStringSubmatch(texto)
	if esta == nil {
		t.Fatal("ETAPAS.md no dice de que dia es esta instantanea, con «Esta es del " +
			"**DD-MM-AAAA**». Sin las dos fechas no se puede derivar ninguna tasa")
	}
	d := reDeriva.FindStringSubmatch(texto)
	if d == nil {
		t.Fatalf("ETAPAS.md declara la instantanea anterior y no publica la deriva.\n"+
			"  Hoy serian: **%d dias** despues: **+%d cerradas** y **+%d abiertas**, ...",
			0, cerradas, total)
	}

	antCerradas := aEntero(t, ant[1])
	antTotal := aEntero(t, ant[2])
	desde := aFecha(t, ant[3])
	hasta := aFecha(t, esta[1])

	dias := int(hasta.Sub(desde).Hours() / 24)
	if dias <= 0 {
		t.Fatalf("entre %s y %s hay %d dias: las fechas de la deriva estan al reves o son "+
			"la misma", ant[3], esta[1], dias)
	}
	quieroCerradas := cerradas - antCerradas
	quieroAbiertas := total - antTotal
	quieroPendAntes := antTotal - antCerradas
	quieroPendAhora := total - cerradas

	if got := aEntero(t, d[1]); got != dias {
		t.Errorf("la deriva dice %d dias y entre %s y %s hay %d", got, ant[3], esta[1], dias)
	}
	if got := aEntero(t, d[2]); got != quieroCerradas {
		t.Errorf("la deriva dice +%d cerradas y del arbol salen +%d (%d hoy, %d el %s)",
			got, quieroCerradas, cerradas, antCerradas, ant[3])
	}
	if got := aEntero(t, d[3]); got != quieroAbiertas {
		t.Errorf("la deriva dice +%d abiertas y del arbol salen +%d (%d hoy, %d el %s).\n"+
			"  Si este numero ha subido mas que el de cerradas, el plan crece mas deprisa "+
			"de lo que se cierra, y eso tiene que constar aunque el trabajo sea legitimo.",
			got, quieroAbiertas, total, antTotal, ant[3])
	}
	// LAS TASAS, DERIVADAS. Se comparan con una cifra decimal, que es como se
	// publican: exigir mas seria exigir que la prosa lleve seis decimales.
	comprobarTasa(t, "cerradas", d[4], quieroCerradas, dias)
	comprobarTasa(t, "abiertas", d[5], quieroAbiertas, dias)

	if got := aEntero(t, d[6]); got != quieroPendAntes {
		t.Errorf("la deriva dice que las pendientes eran %d y el %s eran %d",
			got, ant[3], quieroPendAntes)
	}
	if got := aEntero(t, d[7]); got != quieroPendAhora {
		t.Errorf("la deriva dice que las pendientes son %d y del arbol salen %d",
			got, quieroPendAhora)
	}

	t.Logf("deriva del plan: en %d dias, +%d cerradas (%.1f/dia) y +%d abiertas (%.1f/dia); "+
		"pendientes de %d a %d", dias, quieroCerradas, float64(quieroCerradas)/float64(dias),
		quieroAbiertas, float64(quieroAbiertas)/float64(dias), quieroPendAntes, quieroPendAhora)
}

func comprobarTasa(t *testing.T, que, declarada string, delta, dias int) {
	t.Helper()
	quiero := fmt.Sprintf("%.1f", float64(delta)/float64(dias))
	// El castellano escribe los decimales con coma y Go con punto.
	if strings.ReplaceAll(declarada, ",", ".") != quiero {
		t.Errorf("la deriva dice %s %s al dia y de %d en %d dias salen %s",
			declarada, que, delta, dias, strings.ReplaceAll(quiero, ".", ","))
	}
}

func aEntero(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(s)
	if err != nil {
		t.Fatalf("%q no es un numero: %v", s, err)
	}
	return n
}

func aFecha(t *testing.T, s string) time.Time {
	t.Helper()
	f, err := time.Parse("02-01-2006", s)
	if err != nil {
		t.Fatalf("%q no es una fecha DD-MM-AAAA: %v", s, err)
	}
	return f
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
