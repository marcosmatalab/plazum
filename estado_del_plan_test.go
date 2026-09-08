package plazum

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// EL ESTADO DEL PLAN VA CON TRES NUMEROS, Y LOS TRES SALEN DEL ARBOL.
//
// # De donde sale la regla
//
// Decision del 03-09-2026: el contador de casillas NO mide el trabajo de
// corpus. Una campana que escribio 39 relojes, tres paquetes nuevos, dos
// pantallas y el armazon visual movio CUATRO casillas, porque las casillas
// estan escritas como puertas y una puerta no se cierra a medias. Marcar a ojo
// habria dado veinte casillas y un producto peor.
//
// Asi que el estado se publica con cifras que dicen cosas distintas: casillas
// cerradas (cuanto del plan esta terminado de verdad) y relojes escritos
// (cuanto corpus existe). Ninguna de las dos sola dice la verdad.
//
// # Y la tercera, que es la que decide la fecha (08-09-2026)
//
// Las dos de arriba cuentan el plan ENTERO, y el plan entero llega hasta la
// etapa 8. Lo que decide si la v1 sale no es eso: son las casillas abiertas de
// las etapas 3 y 4 y del bloque de la v1, que es un conjunto mucho mas pequeno
// y el unico que bloquea. Vivia en prosa de un informe y se contaba a mano, con
// el resultado previsible: dos recuentos del mismo dia dieron **30 y 31**, y el
// arbol dice 31. El total, 65, coincidia en los dos, que es lo que hace peor al
// fallo: cuando el denominador cuadra, nadie va a mirar el numerador.
//
// # Y las tres con puerta, porque un numero sin puerta se mueve solo
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
	reBloqueanteDeclarado = regexp.MustCompile(
		`(?s)<!-- estado:inicio -->.*?\*\*(\d+) de (\d+) abiertas\*\* en el conjunto que ` +
			`bloquea la v1.*?<!-- estado:fin -->`)

	// LA DERIVA. Ver TestElEstadoDelPlanPublicaSuDeriva.
	//
	// El ancla es la instantanea ANTERIOR, que es un hecho del pasado y no
	// caduca; todo lo demas se deriva de ella y del arbol de hoy. Y desde el
	// 08-09-2026 lleva ademas SU SHA, para que el hecho del pasado se pueda ir
	// a mirar en vez de creerselo.
	reInstantaneaAnterior = regexp.MustCompile(
		"Instantanea anterior: \\*\\*(\\d+) de (\\d+), el (\\d{2}-\\d{2}-\\d{4})\\*\\*, " +
			"en `([0-9a-f]{7,40})`")
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

// LA TERCERA CIFRA: EL CONJUNTO QUE BLOQUEA LA v1, DERIVADO.
//
// # Por que no puede vivir en prosa
//
// Es el numero que decide si la v1 sale, y hasta hoy se contaba a mano en un
// informe. El 08-09-2026 dos recuentos del mismo conjunto, hechos el mismo dia,
// dieron **30 y 31 abiertas**; el arbol dice **31**. Y el total, **65**,
// coincidia en los dos: cuando el denominador cuadra, el numerador no se
// vuelve a mirar.
//
// # El conjunto se declara por ENCABEZAMIENTO, no por rango de lineas
//
// Un rango (`de la linea 94 a la 199`) se queda viejo en cuanto alguien mueve
// una seccion, y se queda viejo EN SILENCIO, que es justo lo que este numero no
// puede permitirse. Con encabezamientos, mover una seccion no cambia el
// conjunto y RENOMBRARLA pone el test rojo diciendo cual falta, que es la
// respuesta correcta: si «## Etapa 4» pasa a llamarse otra cosa, la pregunta de
// que entra en el conjunto bloqueante vuelve a estar abierta y la contesta una
// persona, no una expresion regular que sigue casando con menos de lo que
// creia.
//
// LO QUE ESTA PUERTA NO MIRA, dicho: que las tres secciones sean LAS QUE
// BLOQUEAN. Eso es una decision de plan (D-19 y D-22), no un hecho del arbol, y
// esta escrita aqui al lado. Si manana la v1 pasa a depender de la etapa 6, hay
// que venir a cambiarla a mano y nada se pondra rojo por no haberlo hecho.
var seccionesQueBloqueanLaV1 = []string{"## Etapa 3", "## Etapa 4", "## v1"}

// casillasDeLaSeccion cuenta cerradas y abiertas por seccion de nivel dos.
// Devuelve el conteo indexado por el PREFIJO declarado arriba, y no por el
// encabezamiento entero, porque los encabezamientos de este fichero llevan
// dentro su duracion estimada y sus decisiones, y atarlos enteros seria atar
// una frase que cambia por gusto.
func casillasDeLaSeccion(texto string, prefijos []string) map[string][2]int {
	cuenta := map[string][2]int{}
	actual := ""
	for _, linea := range strings.Split(texto, "\n") {
		if strings.HasPrefix(linea, "## ") {
			actual = ""
			for _, p := range prefijos {
				if strings.HasPrefix(linea, p) {
					actual = p
					break
				}
			}
			continue
		}
		if actual == "" {
			continue
		}
		c := cuenta[actual]
		switch {
		case strings.HasPrefix(linea, "- [x] "):
			c[0]++
		case strings.HasPrefix(linea, "- [ ] "):
			c[1]++
		}
		cuenta[actual] = c
	}
	return cuenta
}

func TestElConjuntoQueBloqueaLaV1LoCuentaElArbol(t *testing.T) {
	texto := leerEtapas(t)
	cuenta := casillasDeLaSeccion(texto, seccionesQueBloqueanLaV1)

	abiertas, total := 0, 0
	for _, s := range seccionesQueBloqueanLaV1 {
		c, hay := cuenta[s]
		if !hay || c[0]+c[1] == 0 {
			t.Fatalf("ETAPAS.md no trae ninguna casilla bajo «%s».\n"+
				"  O la seccion se ha renombrado, o se ha movido fuera del fichero. Las "+
				"dos cosas reabren la pregunta de que conjunto bloquea la v1, y la "+
				"contesta una persona: hay que venir a %s a decirlo.", s, "estado_del_plan_test.go")
		}
		abiertas += c[1]
		total += c[0] + c[1]
	}

	b := reBloqueanteDeclarado.FindStringSubmatch(texto)
	if b == nil {
		t.Fatalf("ETAPAS.md no publica la tercera cifra del estado, con «**N de M "+
			"abiertas** en el conjunto que bloquea la v1» dentro del bloque de estado.\n"+
			"  Es la que decide la fecha, y sin ella se cuenta a mano en un informe: hoy "+
			"serian **%d de %d abiertas**.", abiertas, total)
	}
	decAbiertas := aEntero(t, b[1])
	decTotal := aEntero(t, b[2])
	if decAbiertas != abiertas || decTotal != total {
		t.Errorf("ETAPAS.md declara %d de %d abiertas en el conjunto bloqueante y del "+
			"arbol salen %d de %d.\n"+
			"  Igualdad exacta en los dos sentidos a proposito: que el conjunto ENCOJA "+
			"sin que nadie lo note es la otra mitad del mismo fallo.",
			decAbiertas, decTotal, abiertas, total)
	}

	for _, s := range seccionesQueBloqueanLaV1 {
		c := cuenta[s]
		t.Logf("  %-12s %d cerradas, %d abiertas", s, c[0], c[1])
	}
	t.Logf("conjunto que bloquea la v1: %d de %d abiertas", abiertas, total)
}

// EL PLAN CRECE MAS DEPRISA DE LO QUE SE CIERRA, Y ESO NO SE VE EN NINGUN SITIO.
//
// # El hueco que cierra
//
// El bloque de estado publica CERRADAS y TOTAL, y las dos con puerta. Lo que no
// publica es la DERIVA: cuantas se han abierto y cuantas se han cerrado desde la
// instantanea anterior. Y esa es la cifra que dice si el plan converge.
//
// Los deltas, las tasas y las pendientes NO SE ESCRIBEN AQUI, y eso es
// deliberado: este godoc los llevaba y se quedaron viejos, diciendo «72 de 142»
// y «+41 y +42» cuando el arbol daba otra cosa. Un numero derivado escrito en
// prosa es la afirmacion acompanada otra vez, y aqui sobra: el test los imprime
// con t.Logf cada vez que corre.
//
// **Que el plan crezca no es malo por si solo** y hay que decirlo, o esta cifra
// se lee como un reproche: D-22 abrio siete casillas porque destapo trabajo real
// que ya estaba dentro de otras dos, y el trabajo estaba igual antes de
// contarlo. Lo que si es malo es que **no se viera**: con dos cifras que solo
// suben, un re-corte que ensancha el plan se lee igual que uno que lo estrecha.
//
// # Por que el ancla se escribe y todo lo demas se deriva
//
// La instantanea anterior es un HECHO DEL PASADO: no puede quedarse vieja,
// porque describe un dia que ya paso. Los deltas, las tasas y las pendientes
// salen de ella y del arbol de hoy, asi que ninguno se puede escribir a mano ni
// quedarse viejo.
//
// # Y desde el 08-09-2026 el hecho del pasado se puede ir a mirar
//
// «No caduca» y «es cierto» no son lo mismo, y hasta hoy solo estaba lo
// primero. El ancla se leia de otra linea del mismo fichero y nada la ataba al
// arbol, mientras de ella colgaban SEIS cifras publicadas (dos deltas, dos tasas
// y dos pendientes): un ancla desviada las desviaba las seis y dejaba el test en
// verde, porque todas casaban con la misma mentira.
//
// Ahora el ancla lleva su SHA y este test lo recomputa: `git show <sha>:ETAPAS.md`
// contado con los mismos dos patrones, y `%ad` del propio commit contra la fecha
// declarada.
//
// LA TRAMPA QUE HIZO FALLAR EL CALCULO A MANO, escrita aqui para que no vuelva:
// la orden que producia el ancla era
//
//	git log --format=%h --until="2026-08-25 23:59:59" -1 -- ETAPAS.md
//
// y **mezcla dos relojes**. `--until` (como `--since`) filtra por la fecha de
// COMMIT, y `%ad` imprime la de AUTOR. En un arbol rebasado las dos se separan,
// y entonces el trabajo de un dia se le atribuye al anterior o al siguiente. Se
// escogen por autor con `--date-order` y `%ad`, o se dice que se esta filtrando
// por fecha de commit; lo que no vale es imprimir una y filtrar por la otra.
// (En este ancla las dos coinciden, que es exactamente por lo que habria colado.)
//
// LO QUE ESTA PUERTA SIGUE SIN MIRAR, con su cardinal: **una** cosa, que el
// commit del ancla sea el ULTIMO de ese dia. Comprobarlo obliga a recorrer la
// historia, y la historia no esta siempre: ver `anclaDelArbol`.
func TestElEstadoDelPlanPublicaSuDeriva(t *testing.T) {
	texto := leerEtapas(t)
	cerradas := len(reCerrada.FindAllString(texto, -1))
	total := cerradas + len(reAbierta.FindAllString(texto, -1))

	ant := reInstantaneaAnterior.FindStringSubmatch(texto)
	if ant == nil {
		t.Fatalf("ETAPAS.md no declara la instantanea anterior con «Instantanea anterior: " +
			"**N de M, el DD-MM-AAAA**, en `<sha>`».\n" +
			"  Sin ancla no hay deriva, y sin deriva las dos cifras del estado solo pueden " +
			"subir: un re-corte que ensancha el plan se lee igual que uno que lo estrecha.\n" +
			"  Y sin SHA el ancla no se puede ir a mirar: de ella cuelgan seis cifras " +
			"publicadas, asi que un ancla desviada las desvia las seis y deja esto verde.")
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

	comprobarElAnclaContraElArbol(t, ant[4], antCerradas, antTotal, ant[3])

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

// gitDice corre git y devuelve su salida. El error se devuelve, no se convierte
// en fallo: quien llama decide si «git no contesta» es rojo o es «no se pudo
// mirar», que no son lo mismo y confundirlos hace que una maquina sin historia
// se lea como una historia verificada.
func gitDice(args ...string) (string, error) {
	salida, err := exec.Command("git", args...).Output()
	return strings.TrimSpace(string(salida)), err
}

// anclaDelArbol recomputa el ancla de la deriva desde el propio repositorio.
//
// LAS RESPUESTAS SON TRES Y NO DOS (invariante 8): el ancla CUADRA, el ancla NO
// CUADRA, y NO SE PUDO MIRAR. La tercera no es verde y se dice en voz alta, con
// su motivo, igual que `comprobar.sh` distingue una puerta saltada de una puerta
// en verde.
//
// # Donde corre de verdad, dicho antes de que nadie lo suponga
//
// Hace falta la historia. `actions/checkout` clona con profundidad 1 por
// defecto, y en un clon superficial el objeto del ancla NO ESTA: ahi la
// respuesta correcta es la tercera, no un rojo. Por eso el job que corre la
// suite pide `fetch-depth: 0` en `.github/workflows/ci.yml`; si alguien se lo
// quita, esto se convierte en un salto declarado y deja de vigilar sin ponerse
// rojo, que es la unica forma en que esta comprobacion puede desaparecer.
//
// # Lo que NO comprueba, con su cardinal: uno
//
// Que el commit del ancla sea el ULTIMO commit de ese dia. Eso obliga a recorrer
// la historia entera del dia y no solo un objeto, y no lo hace: lo que si
// comprueba es que la fecha de AUTOR del commit citado es la declarada y que su
// arbol trae exactamente las casillas declaradas, que es de donde cuelgan las
// seis cifras derivadas.
func comprobarElAnclaContraElArbol(t *testing.T, sha string, cerradas, total int, fecha string) {
	t.Helper()

	if _, err := gitDice("rev-parse", "--is-inside-work-tree"); err != nil {
		t.Logf("SALTADO, no comprobado: git no contesta aqui (%v), asi que el ancla "+
			"%s se queda sin recomputar. NO es verde: es que no se pudo mirar", err, sha)
		return
	}
	if s, err := gitDice("rev-parse", "--is-shallow-repository"); err == nil && s == "true" {
		t.Logf("SALTADO, no comprobado: el clon es SUPERFICIAL, asi que el objeto %s no "+
			"esta aqui aunque el ancla sea correcta. Con fetch-depth: 0 esto se "+
			"recomputa; sin el, no. NO es verde", sha)
		return
	}

	// A partir de aqui hay historia, asi que un objeto que falta ya no es «no
	// se pudo mirar»: es un ancla que nombra un commit que no existe.
	if _, err := gitDice("cat-file", "-e", sha+"^{commit}"); err != nil {
		t.Errorf("el ancla de la deriva nombra el commit %s y este repositorio no lo "+
			"tiene, teniendo la historia entera.\n"+
			"  Un SHA que no resuelve es peor que ninguno: tiene la FORMA de lo "+
			"verificable, que es lo que hace que nadie vaya a verificarlo.", sha)
		return
	}

	autor, err := gitDice("show", "--no-patch", "--format=%ad", "--date=format:%d-%m-%Y", sha)
	if err != nil {
		t.Errorf("no puedo leer la fecha de autor de %s: %v", sha, err)
		return
	}
	if autor != fecha {
		t.Errorf("el ancla dice que %s es del %s y su fecha de AUTOR es %s.\n"+
			"  Cuidado con la trampa que produjo este ancla: `git log --until` filtra "+
			"por fecha de COMMIT y `%%ad` imprime la de AUTOR. Mezclarlas atribuye el "+
			"trabajo de un dia al anterior.", sha, fecha, autor)
	}

	arbol, err := gitDice("show", sha+":"+rutaDeEtapas)
	if err != nil {
		t.Errorf("no puedo leer %s en %s: %v", rutaDeEtapas, sha, err)
		return
	}
	vc := len(reCerrada.FindAllString(arbol, -1))
	vt := vc + len(reAbierta.FindAllString(arbol, -1))
	if vc != cerradas || vt != total {
		t.Errorf("el ancla declara %d de %d el %s y el arbol de %s trae %d de %d.\n"+
			"  De este ancla cuelgan SEIS cifras publicadas (dos deltas, dos tasas y dos "+
			"pendientes): desviarla las desvia las seis a la vez, y las seis siguen "+
			"cuadrando entre si.", cerradas, total, fecha, sha, vc, vt)
		return
	}
	t.Logf("ancla recomputada del arbol: %s (%s) trae %d de %d", sha, autor, vc, vt)
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

// CONTROL NEGATIVO DE LOS TRES CONTADORES.
//
// El de casillas tiene un fallo probable muy concreto: contar casillas citadas
// dentro de la prosa de otra, que en este fichero abundan. El de la declaracion
// tiene el de siempre: casar cualquier numero del documento. Y el de secciones
// tiene el suyo, que es el peor de los tres porque no se ve: seguir contando
// despues de que la seccion se acabe, o sea meter en el conjunto bloqueante las
// casillas de la etapa 5.
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

	// EL CONTADOR POR SECCION, con la seccion que NO entra pegada detras.
	doc := "# cabecera\n" +
		"- [x] una casilla suelta antes de la primera seccion, que no es de nadie\n" +
		"## Etapa 3 (6-8 FdS): lo que sea\n" +
		"- [x] cerrada de la 3\n" +
		"- [ ] abierta de la 3\n" +
		"### subseccion, que NO corta la seccion\n" +
		"- [ ] otra abierta de la 3\n" +
		"## Etapa 5 (post-v1): esta NO entra\n" +
		"- [ ] abierta de la 5\n" +
		"- [ ] otra de la 5\n"
	cuenta := casillasDeLaSeccion(doc, seccionesQueBloqueanLaV1)
	if got := cuenta["## Etapa 3"]; got != [2]int{1, 2} {
		t.Errorf("la etapa 3 sale %v y son [1 2]: o se come la subseccion, o sigue "+
			"contando despues de que la seccion se acabe", got)
	}
	if _, hay := cuenta["## Etapa 4"]; hay {
		t.Error("cuenta una etapa 4 que este documento no tiene: el conjunto bloqueante " +
			"se estaria llenando de sitios que no existen")
	}
	if suma := len(cuenta); suma != 1 {
		t.Errorf("ha reconocido %d secciones y solo hay una de las tres: la etapa 5 se "+
			"esta colando en el conjunto que decide la fecha de la v1", suma)
	}
}
