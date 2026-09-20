package plazum

import (
	"go/build"
	"go/build/constraint"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// UN TEST QUE SALE DE LA SUITE BLOQUEANTE TIENE QUE SEGUIR CORRIENDO EN ALGUN
// SITIO, Y ESE SITIO SE COMPRUEBA.
//
// # El movimiento que obliga a esta puerta
//
// El 20-09-2026 `main` estaba rojo por `frescura_test.go`, que no mide el
// producto: mide si `docs/marcador.md` dice de cuando son sus notas. Cruzo el
// umbral de 14 dias el 19-09 y puso rojo un arbol donde no habia ni un fallo de
// comportamiento. El criterio que sale de ahi, y que gobierna toda la raiz:
//
//   - puerta sobre el COMPORTAMIENTO del producto: bloquea el commit, porque lo
//     que afirma es «este cambio esta mal», y el cambio es de quien commitea;
//   - puerta sobre el ESTADO DEL REPOSITORIO: no bloquea el commit, porque lo
//     que afirma es «este documento envejecio», y eso no lo causo el commit.
//
// La segunda se saca a un workflow con horario. `frescura_test.go` es la
// primera, y por eso lleva `//go:build frescura`.
//
// # POR QUE ESTA PUERTA, Y NO SOLO EL TRASLADO
//
// Porque una etiqueta de construccion es **la forma mas silenciosa que existe en
// Go de apagar un test para siempre**. `go test ./...` deja de compilar el
// fichero y no dice absolutamente nada: ni «no tests to run», ni «no test
// files», ni una linea en la salida. Es el verde vacio que `puerta.sh` existe
// para cazar, entrando por la unica puerta que `puerta.sh` no vigila, porque
// `puerta.sh` cuenta los casos de lo que SE EJECUTA y aqui el fichero ni
// siquiera llega al compilador.
//
// O sea que el traslado tiene dos mitades: la etiqueta, que quita; y el
// workflow, que devuelve. Un traslado a medias (etiqueta sin workflow, o
// workflow que deja de nombrar la etiqueta) es indistinguible de un borrado, y
// **nadie se entera nunca**. Esto es lo que las cablea.
//
// # Como decide si un fichero esta fuera, y por que no lo decide leyendo texto
//
// No busca la cadena `//go:build`: se lo pregunta a `go/build`, que es el mismo
// codigo que usa la cadena de herramientas para decidir que compila. Un
// detector propio de constraints seria una segunda implementacion de una regla
// que ya tiene duena, y la forma de que dos esten de acuerdo y mande la otra.
//
// # Las dos direcciones, que son el fallo que se comete solo
//
//   - fichero fuera del build y ningun workflow que lo corra: se apago un test;
//   - workflow que pasa `-tags X` y ningun fichero con la etiqueta X: el
//     workflow lleva desde entonces corriendo lo mismo que la suite, o sea
//     nada nuevo, y su puerta sale verde sin haber mirado lo que dice mirar.
//
// # Y el cardinal, con igualdad exacta
//
// `FicherosFueraDeLaSuiteBloqueante` topa cuantos ficheros de la raiz pueden
// estar fuera. Con igualdad exacta en los dos sentidos a proposito: sin el, la
// forma barata de aprobar cualquier puerta molesta de la raiz es ponerle una
// etiqueta y un cron, y esta puerta lo bendeciria una por una sin que el total
// se moviera nunca de «todo declarado». Sacar un test de la suite tiene que
// costar tocar este numero y decir por que.
//
// # LO QUE ESTA PUERTA NO MIRA, dicho aqui y no descubierto luego
//
// No comprueba que el workflow programado **se haya ejecutado**. Un cron puede
// no dispararse nunca (GitHub desactiva los horarios de un repositorio sin
// actividad a los 60 dias) y esto seguiria verde: lo que afirma es que el cable
// esta puesto, no que la corriente pase. Al otro lado de esa frontera no mira
// nadie hoy, y se dice en vez de suponerlo.
const (
	// FicherosFueraDeLaSuiteBloqueante son los ficheros `*_test.go` de la raiz
	// que el build por defecto NO compila. Hoy 1: `frescura_test.go`.
	FicherosFueraDeLaSuiteBloqueante = 1
)

// etiquetaDeExclusion devuelve la etiqueta que saca a un fichero del build por
// defecto.
//
// EXIGE LA FORMA SIMPLE `//go:build <etiqueta>` y rechaza cualquier otra. No es
// pereza: de una expresion con `&&`, `||` o `!` no se puede derivar que hay que
// pasarle a `-tags` sin reimplementar la evaluacion, y la salida barata seria
// suponer una etiqueta y dar por cableado lo que no se ha comprobado. De las
// tres formas de la nada, la peligrosa es la tercera: presente y no
// interpretable es error, nunca el valor por defecto.
func etiquetaDeExclusion(t *testing.T, fichero string) string {
	t.Helper()
	b, err := os.ReadFile(fichero) // #nosec G304 -- ruta que sale de os.ReadDir(".")
	if err != nil {
		t.Fatalf("no puedo leer %s: %v", fichero, err)
	}
	for _, linea := range strings.Split(string(b), "\n") {
		linea = strings.TrimRight(linea, "\r")
		if !constraint.IsGoBuild(linea) {
			continue
		}
		expr, err := constraint.Parse(linea)
		if err != nil {
			t.Fatalf("%s trae una linea //go:build que no parsea (%q): %v", fichero, linea, err)
		}
		tag, simple := expr.(*constraint.TagExpr)
		if !simple {
			t.Fatalf("%s sale del build por defecto con una expresion compuesta (%q).\n"+
				"  Esta puerta solo sabe cablear la forma simple `//go:build <etiqueta>`, "+
				"porque de una expresion compuesta no puede derivar que hay que pasarle a "+
				"-tags sin volver a implementar la evaluacion de constraints.\n"+
				"  Suponer una etiqueta seria dar por cableado lo que no se ha comprobado, "+
				"que es justo el fallo que este fichero existe para impedir.\n"+
				"  Arreglo: dejarlo en una sola etiqueta, o ensenar a esta puerta a "+
				"evaluar la expresion y decir por que hacia falta.", fichero, linea)
		}
		return tag.Tag
	}
	t.Fatalf("%s no lo compila el build por defecto y no lleva ninguna linea //go:build.\n"+
		"  Entonces lo que lo saca es el sufijo del nombre (_windows.go, _amd64.go...) y "+
		"esta puerta no puede decir con que -tags devolverlo a la vida.\n"+
		"  Arreglo: si es un test de estado del repositorio, sacarlo con una etiqueta "+
		"explicita; si es especifico de una plataforma, no es de esta familia.", fichero)
	return ""
}

// disparadoresDeWorkflow devuelve las claves del bloque `on:` de un workflow.
func disparadoresDeWorkflow(cuerpo string) map[string]bool {
	sangria := func(s string) int { return len(s) - len(strings.TrimLeft(s, " ")) }
	lineas := strings.Split(cuerpo, "\n")
	out := map[string]bool{}
	for i, l := range lineas {
		if strings.TrimSpace(strings.TrimRight(l, "\r")) != "on:" || sangria(l) != 0 {
			continue
		}
		for j := i + 1; j < len(lineas); j++ {
			cruda := strings.TrimRight(lineas[j], "\r")
			if strings.TrimSpace(cruda) == "" || strings.HasPrefix(strings.TrimSpace(cruda), "#") {
				continue
			}
			if sangria(cruda) == 0 {
				break
			}
			clave := strings.TrimSpace(cruda)
			if sangria(cruda) == 2 && strings.Contains(clave, ":") &&
				!strings.HasPrefix(clave, "-") {
				out[strings.TrimSpace(strings.SplitN(clave, ":", 2)[0])] = true
			}
			if strings.HasPrefix(clave, "- cron:") {
				out["cron"] = true
			}
		}
	}
	return out
}

// puertasConEtiqueta devuelve, por etiqueta, los workflows que la corren.
//
// Casa por `-tags <etiqueta>` DENTRO de una linea `puerta "`, que es el unico
// sitio donde este repositorio ejecuta tests en CI: un `go test` suelto lo
// prohibe TestNingunWorkflowInvocaGoTestSinContarLosCasos, asi que buscar solo
// aqui no deja ninguna via abierta.
func puertasConEtiqueta(cuerpos map[string]string) map[string][]string {
	out := map[string][]string{}
	for nombre, cuerpo := range cuerpos {
		for _, l := range strings.Split(cuerpo, "\n") {
			limpia := strings.TrimSpace(strings.TrimRight(l, "\r"))
			if !strings.HasPrefix(limpia, "puerta \"") {
				continue
			}
			campos := strings.Fields(limpia)
			for i, c := range campos {
				switch {
				case c == "-tags" && i+1 < len(campos):
					out[campos[i+1]] = append(out[campos[i+1]], nombre)
				case strings.HasPrefix(c, "-tags="):
					out[strings.TrimPrefix(c, "-tags=")] = append(
						out[strings.TrimPrefix(c, "-tags=")], nombre)
				}
			}
		}
	}
	return out
}

func TestTodoTestFueraDeLaSuiteBloqueanteLoCorreUnWorkflowProgramado(t *testing.T) {
	entradas, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("no puedo leer la raiz: %v", err)
	}

	fuera := map[string]string{} // fichero -> etiqueta
	dentro := 0
	for _, e := range entradas {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		// Se lo pregunta a go/build, que es quien de verdad decide.
		casa, err := build.Default.MatchFile(".", e.Name())
		if err != nil {
			// Un fichero que no se puede interpretar NO es un fichero que entra.
			t.Fatalf("go/build no ha podido decidir si %s entra en el build por defecto: %v.\n"+
				"  No poder decidirlo no es que entre: es un dato que hay y no se entiende, "+
				"y darlo por dentro absolveria de mas.", e.Name(), err)
		}
		if casa {
			dentro++
			continue
		}
		fuera[e.Name()] = etiquetaDeExclusion(t, e.Name())
	}

	if dentro == 0 {
		t.Fatal("ni un solo fichero de test de la raiz entra en el build por defecto: " +
			"esta puerta estaria leyendo mal el arbol entero y saldria verde igual")
	}

	// EL CARDINAL, CON IGUALDAD EXACTA Y EN LOS DOS SENTIDOS.
	if len(fuera) != FicherosFueraDeLaSuiteBloqueante {
		t.Errorf("hay %d ficheros de test de la raiz fuera del build por defecto (%v) y "+
			"FicherosFueraDeLaSuiteBloqueante dice %d.\n"+
			"  Si ha SUBIDO: alguien acaba de sacar un test de la suite bloqueante. Eso se "+
			"hace a proposito y con su motivo, no de paso.\n"+
			"  Si ha BAJADO: un test volvio a la suite y este numero se quedo viejo, o sea "+
			"que el techo dejo de topar nada.\n"+
			"  Arreglo: mover el numero EN EL MISMO COMMIT y decir por que se movio.",
			len(fuera), nombresOrdenados(fuera), FicherosFueraDeLaSuiteBloqueante)
	}

	cuerpos := workflows(t)
	porEtiqueta := puertasConEtiqueta(cuerpos)

	// DIRECCION 1: todo fichero fuera lo corre un workflow, y ese workflow tiene
	// horario y no pinta el estado de main.
	for fichero, etiqueta := range fuera {
		quienes := porEtiqueta[etiqueta]
		if len(quienes) == 0 {
			t.Errorf("%s esta fuera del build por defecto con `//go:build %s` y NINGUN "+
				"workflow corre una puerta con `-tags %s`.\n"+
				"  O sea que ese test no se ejecuta en ningun sitio y nada se pone rojo por "+
				"ello: `go test ./...` ni siquiera lo compila, asi que no hay «no tests to "+
				"run» que contar. Es un borrado con la forma de un traslado.\n"+
				"  Arreglo: el workflow que lo corra, o quitarle la etiqueta y devolverlo a "+
				"la suite.", fichero, etiqueta, etiqueta)
			continue
		}
		for _, w := range quienes {
			disp := disparadoresDeWorkflow(cuerpos[w])
			if !disp["schedule"] || !disp["cron"] {
				t.Errorf("%s corre `-tags %s` (por %s) y no declara `schedule:` con su "+
					"`cron:`.\n"+
					"  Un test sacado de la suite y colgado de un workflow que solo se lanza "+
					"a mano es un test que no corre: nadie lo lanza.", w, etiqueta, fichero)
			}
			if disp["push"] || disp["pull_request"] {
				t.Errorf("%s corre `-tags %s` (por %s) y ademas se dispara en %v.\n"+
					"  Entonces vuelve a pintar el estado de un commit de codigo, que es "+
					"exactamente lo que el traslado deshizo: una puerta sobre el ESTADO DEL "+
					"REPOSITORIO no bloquea un cambio que no la causo.",
					w, etiqueta, fichero, clavesDe(disp, "push", "pull_request"))
			}
		}
	}

	// DIRECCION 2, la que se olvida: una etiqueta que el workflow pasa y que no
	// saca a nadie del build hace que ese workflow corra lo mismo que la suite.
	for etiqueta, quienes := range porEtiqueta {
		usada := false
		for _, e := range fuera {
			if e == etiqueta {
				usada = true
			}
		}
		if !usada {
			t.Errorf("%v corren una puerta con `-tags %s` y ningun fichero de la raiz sale "+
				"del build por defecto con esa etiqueta.\n"+
				"  Esa puerta lleva desde entonces ejecutando lo mismo que la suite normal, "+
				"o sea nada de lo que dice mirar, y sale verde. Es el verde vacio con la "+
				"forma de una puerta especializada.\n"+
				"  Arreglo: o la etiqueta esta mal escrita, o el fichero que la llevaba "+
				"volvio a la suite y este workflow sobra.", quienes, etiqueta)
		}
	}

	t.Logf("MEDIDO: %d ficheros de test en la raiz dentro del build por defecto, %d fuera "+
		"(%v), %d etiquetas corridas por workflows programados",
		dentro, len(fuera), nombresOrdenados(fuera), len(porEtiqueta))
}

// nombresOrdenados imprime el mapa de forma estable, que es lo que hace util un
// mensaje de error entre dos ejecuciones.
func nombresOrdenados(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+" (-tags "+v+")")
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func clavesDe(m map[string]bool, candidatas ...string) []string {
	var out []string
	for _, c := range candidatas {
		if m[c] {
			out = append(out, c)
		}
	}
	return out
}

// CONTROL NEGATIVO DE LAS TRES MITADES, con ficheros y workflows sinteticos.
//
// Sin esto, las dos direcciones de arriba son ramas que hoy no recorre ninguna
// entrada real: el arbol tiene un solo fichero fuera y esta bien cableado, asi
// que la puerta nace verde y una mutacion la dejaria verde igual porque no hay
// nada que romper. Es M47 otra vez, y la defensa es la misma, dato sintetico que
// recorra la rama contraria.
func TestElDetectorDeFicherosFueraDeLaSuiteFunciona(t *testing.T) {
	dir := t.TempDir()

	escribir := func(nombre, contenido string) {
		if err := os.WriteFile(filepath.Join(dir, nombre), []byte(contenido), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	escribir("dentro_test.go", "package p\n")
	escribir("fuera_test.go", "//go:build inventada\n\npackage p\n")

	casas := func(nombre string) bool {
		casa, err := build.Default.MatchFile(dir, nombre)
		if err != nil {
			t.Fatalf("MatchFile(%s): %v", nombre, err)
		}
		return casa
	}
	if !casas("dentro_test.go") {
		t.Error("un fichero sin constraint no entra en el build por defecto: entonces esta " +
			"puerta acusaria a la suite entera y su primer arreglo seria aflojarla")
	}
	if casas("fuera_test.go") {
		t.Error("un fichero con `//go:build inventada` entra en el build por defecto: " +
			"entonces el detector no ve ninguna exclusion y esta puerta es decorativa")
	}
}

func TestElDetectorDeWorkflowsProgramadosFunciona(t *testing.T) {
	const programado = "name: x\n\non:\n  schedule:\n    - cron: '17 6 * * *'\n" +
		"  workflow_dispatch:\n\njobs:\n  j:\n    steps:\n" +
		"      - run: |\n          puerta \"frescura\" 3 . -tags frescura\n"
	const bloqueante = "name: x\n\non:\n  push:\n    branches: [main]\n  pull_request:\n\n" +
		"jobs:\n  j:\n    steps:\n" +
		"      - run: |\n          puerta \"frescura\" 3 . -tags frescura\n"
	const aMano = "name: x\n\non:\n  workflow_dispatch:\n\njobs:\n  j:\n    steps:\n" +
		"      - run: |\n          puerta \"frescura\" 3 . -tags frescura\n"

	t.Run("el programado trae schedule y cron y no trae push", func(t *testing.T) {
		d := disparadoresDeWorkflow(programado)
		if !d["schedule"] || !d["cron"] {
			t.Errorf("no reconoce el horario: %v", d)
		}
		if d["push"] || d["pull_request"] {
			t.Errorf("se inventa un disparador de commit: %v", d)
		}
	})
	t.Run("el bloqueante se delata", func(t *testing.T) {
		d := disparadoresDeWorkflow(bloqueante)
		if !d["push"] || !d["pull_request"] {
			t.Errorf("no ve que pinta el estado de un commit: %v", d)
		}
		if d["cron"] {
			t.Errorf("se inventa un cron que no esta: %v", d)
		}
	})
	t.Run("el de solo a mano no cuenta como programado", func(t *testing.T) {
		d := disparadoresDeWorkflow(aMano)
		if d["schedule"] || d["cron"] {
			t.Errorf("un workflow_dispatch suelto pasa por horario: %v", d)
		}
	})

	t.Run("la etiqueta se saca de la linea de puerta, en sus dos formas", func(t *testing.T) {
		got := puertasConEtiqueta(map[string]string{
			"suelto.yml": programado,
			"igual.yml": "jobs:\n  j:\n    steps:\n      - run: |\n" +
				"          puerta \"x\" 3 . -tags=otra\n",
		})
		if len(got["frescura"]) != 1 || got["frescura"][0] != "suelto.yml" {
			t.Errorf("no saca la etiqueta de `-tags frescura`: %v", got)
		}
		if len(got["otra"]) != 1 {
			t.Errorf("no saca la etiqueta de `-tags=otra`: %v", got)
		}
	})
	t.Run("un go test suelto NO cuenta como puerta con etiqueta", func(t *testing.T) {
		got := puertasConEtiqueta(map[string]string{
			"malo.yml": "jobs:\n  j:\n    steps:\n      - run: go test . -tags frescura\n",
		})
		if len(got) != 0 {
			t.Errorf("un `go test` a pelo cuenta como cable: %v.\n"+
				"  Entonces la forma de aprobar esta puerta seria justo la que "+
				"TestNingunWorkflowInvocaGoTestSinContarLosCasos prohibe, y las dos "+
				"puertas se estarian contradiciendo", got)
		}
	})
	t.Run("una linea de puerta sin -tags no aporta etiquetas", func(t *testing.T) {
		got := puertasConEtiqueta(map[string]string{
			"ci.yml": "      - run: |\n          puerta \"suite completa\" 2200 ./...\n",
		})
		if len(got) != 0 {
			t.Errorf("se inventa una etiqueta donde no hay ninguna: %v", got)
		}
	})
}
