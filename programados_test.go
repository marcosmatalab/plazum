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
// # LA TERCERA EXIGENCIA, Y ES LA QUE ESTA PUERTA NO TENIA (20-09-2026)
//
// La primera version exigia que el fichero CORRIERA en algun sitio y se quedo
// ahi. Era la mitad, y la otra mitad la midio quien leyo el traslado: una
// etiqueta no saca el fichero de `go test ./...`, lo saca del COMPILADOR, asi
// que tampoco lo miran `go vet ./...` ni `go build ./...`. Con una llamada rota
// dentro de `frescura_test.go`:
//
//	go vet ./...                  -> 0
//	go build ./...                -> 0
//	go test ./...                 -> 0
//	go vet -tags frescura ./...   -> 1   undefined: esto_no_existe
//
// O sea que los tres pasos que deciden si un commit entra dicen que si, el
// fichero roto sobrevive el empujon, y lo destapa el cron hasta 24 h despues
// **disfrazado**: un fallo de compilacion dentro del trabajo llamado «frescura»
// se lee como que las notas volvieron a caducar. Aviso equivocado, commit
// equivocado, autor equivocado.
//
// Y ES EXACTAMENTE EL OTRO-LADO QUE EL GODOC DE ABAJO YA SE OBLIGABA A NOMBRAR
// Y NOMBRO MAL. Decia que lo que queda fuera es si el cron llega a dispararse, y
// el hueco de verdad estaba un paso antes: no era si el test CORRE, era si el
// fichero COMPILA. La leccion no es que faltara una linea, es que la pregunta
// «¿que queda justo fuera?» se contesto en el eje en el que yo estaba pensando.
//
// Asi que la exigencia es triple y las tres van atadas a la misma etiqueta
// derivada del arbol, sin escribirla dos veces:
//
//	COMPILA   un `go vet -tags X` en un workflow que corra en push o
//	          pull_request, o sea en la CI que decide si el commit entra;
//	CORRE     una `puerta` con `-tags X` en un workflow con cron y sin push;
//	Y AL REVES los dos, porque un vet o una puerta con una etiqueta que ya no
//	          saca a nadie del build llevan desde entonces mirando el vacio.
//
// La frontera que esta version deja abierta, dicha en su sitio y no aqui, en el
// apartado de lo que no mira.
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
// Uno: no comprueba que el workflow programado **se haya ejecutado**. Un cron
// puede no dispararse nunca (GitHub desactiva los horarios de un repositorio sin
// actividad a los 60 dias) y esto seguiria verde: afirma que el cable esta
// puesto, no que la corriente pase.
//
// Dos: `go vet` compila el fichero y no lo EJECUTA. Un test etiquetado que
// compile y falle sigue apareciendo solo en el cron, que es lo correcto: su
// veredicto es justo lo que el traslado saco de la suite bloqueante. Lo que
// vuelve al commit es el deber de compilar, no el de estar en verde.
//
// # Y EL HUECO QUE SI SE CERRO, porque el registro de huecos tambien caduca
//
// Aqui vivia un tercero, y duro un commit: la exigencia de arriba pedia que el
// `go vet -tags X` EXISTIERA y no que llegara al fichero. Se midio con
// `go list`: `./nucleo/...` **no** incluye el paquete de la raiz y `./...` si,
// o sea que acotar el patron devolvia el fallo entero con la CI en verde. Se
// cierra exigiendo `./...` exacto, y el porque de «exacto» en vez de «un patron
// que alcance la raiz» esta al lado de la comprobacion.
//
// Se cuenta cerrado y no se borra a proposito: **un hueco que se tapa y se borra
// del sitio donde estaba escrito deja el documento diciendo que nunca existio**,
// y entonces nadie puede saber si esta lista es corta porque se mira o porque se
// limpia.
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

// etiquetasDeLasLineas devuelve, por etiqueta, los workflows con una linea que
// cumple `quiero` y pasa `-tags <etiqueta>` (o `-tags=<etiqueta>`).
//
// Las dos formas de escribir la bandera se reconocen a proposito: son la misma
// orden para `go` y distintas para un `strings.Contains`, asi que aceptar solo
// una convertiria «lo escribi con un igual» en «esta puerta no me ve».
//
// LOS COMENTARIOS SE SALTAN, Y NO ES UNA PRECAUCION TEORICA: esta puerta se cazo
// a si misma el 20-09-2026. El comentario que escribi en `ci.yml` para explicar
// por que existe el vet etiquetado contiene la frase `go vet -tags <etiqueta>`,
// y el extractor se la trago como si `<etiqueta>` fuera una etiqueta de verdad,
// lo cual puso roja la direccion contraria con una etiqueta que no existe. Es
// la misma familia que `supresiones_test.go` ya tenia contada: el fallo
// probable de un detector que lee codigo es acusar a la prosa que habla de ese
// codigo. Aqui, ademas, lo peligroso no es solo el falso positivo: una linea
// COMENTADA que nombre la etiqueta correcta haria pasar la exigencia de
// compilar sin que nadie compile nada.
func etiquetasDeLasLineas(cuerpos map[string]string, quiero func(string) bool) map[string][]string {
	out := map[string][]string{}
	for nombre, cuerpo := range cuerpos {
		for _, l := range strings.Split(cuerpo, "\n") {
			limpia := strings.TrimSpace(strings.TrimRight(l, "\r"))
			if strings.HasPrefix(limpia, "#") || !quiero(limpia) {
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

// puertasConEtiqueta devuelve, por etiqueta, los workflows que la EJECUTAN.
//
// Casa dentro de una linea `puerta "`, que es el unico sitio donde este
// repositorio ejecuta tests en CI: un `go test` suelto lo prohibe
// TestNingunWorkflowInvocaGoTestSinContarLosCasos, asi que buscar solo aqui no
// deja ninguna via abierta.
func puertasConEtiqueta(cuerpos map[string]string) map[string][]string {
	return etiquetasDeLasLineas(cuerpos, func(l string) bool {
		return strings.HasPrefix(l, "puerta \"")
	})
}

// esLineaDeVet dice si una linea de workflow ejecuta un `go vet`.
//
// Se ancla al PRINCIPIO de la linea, con o sin el `run:` del YAML delante, y no
// por `strings.Contains`: una linea que mencione `go vet` en medio de otra cosa
// no ejecuta ningun vet, y darla por buena es lo mismo que creerse un comentario.
func esLineaDeVet(l string) bool {
	// Las tres formas en que una orden aparece en un workflow: suelta dentro de
	// un bloque `run: |`, como `run: <orden>`, y como el primer paso de una
	// lista, `- run: <orden>`. Reconocer solo la del medio deja una forma
	// legitima de escribir el paso invisible para esta puerta.
	l = strings.TrimSpace(strings.TrimPrefix(l, "-"))
	l = strings.TrimSpace(strings.TrimPrefix(l, "run:"))
	return strings.HasPrefix(l, "go vet ")
}

// vetDeWorkflow es un `go vet -tags X <patron>` encontrado en un workflow.
type vetDeWorkflow struct {
	Workflow string
	Patrones []string // los operandos de paquete, sin las banderas
}

// vetsConEtiqueta devuelve, por etiqueta, los vet que la COMPILAN.
//
// Es la mitad que faltaba: ejecutar y compilar son dos deberes distintos y se
// reparten en dos sitios distintos. El veredicto de un test sobre documentos
// viejos es del horario; que su fichero compile es del commit.
//
// SE DEVUELVE TAMBIEN EL PATRON DE PAQUETES, y no es un extra. Medido el
// 20-09-2026 con `go list`: `./nucleo/...` NO incluye el paquete de la raiz y
// `./...` si. O sea que un vet acotado deja el fichero etiquetado de la raiz sin
// compilar, sale verde con el fichero roto dentro, y la exigencia de arriba
// quedaria cumplida por un paso que no mira lo que dice mirar.
func vetsConEtiqueta(cuerpos map[string]string) map[string][]vetDeWorkflow {
	out := map[string][]vetDeWorkflow{}
	for nombre, cuerpo := range cuerpos {
		for _, l := range strings.Split(cuerpo, "\n") {
			limpia := strings.TrimSpace(strings.TrimRight(l, "\r"))
			if strings.HasPrefix(limpia, "#") || !esLineaDeVet(limpia) {
				continue
			}
			etiquetas, patrones := banderaTagsYOperandos(limpia)
			for _, e := range etiquetas {
				out[e] = append(out[e], vetDeWorkflow{Workflow: nombre, Patrones: patrones})
			}
		}
	}
	return out
}

// banderaTagsYOperandos parte una linea `... go vet -tags X ./...` en las
// etiquetas que pasa y los operandos de paquete que le quedan.
//
// Los operandos son lo que viene despues del verbo y no empieza por guion, con
// el valor de una bandera separada (`-tags X`) descontado. Es deliberadamente
// tonto: no entiende de banderas de `go vet` que no conozca, asi que si alguien
// mete una con valor separado, su valor se contara como patron y la puerta se
// pondra roja. Roja de mas es el lado correcto en el que equivocarse aqui:
// cuesta leer una linea, mientras que tragarse un operando de menos deja pasar
// exactamente el vet acotado que esto existe para prohibir.
func banderaTagsYOperandos(linea string) (etiquetas, patrones []string) {
	campos := strings.Fields(linea)
	// El verbo: todo lo de antes es `run:`, `-`, `go`, o el propio `vet`.
	inicio := 0
	for i, c := range campos {
		if c == "vet" {
			inicio = i + 1
			break
		}
	}
	saltar := false
	for i := inicio; i < len(campos); i++ {
		c := campos[i]
		if saltar {
			saltar = false
			continue
		}
		switch {
		case c == "-tags" && i+1 < len(campos):
			etiquetas = append(etiquetas, campos[i+1])
			saltar = true
		case strings.HasPrefix(c, "-tags="):
			etiquetas = append(etiquetas, strings.TrimPrefix(c, "-tags="))
		case strings.HasPrefix(c, "-"):
			// Otra bandera. Se deja pasar sin consumir nada detras a proposito:
			// ver el godoc.
		default:
			patrones = append(patrones, c)
		}
	}
	return etiquetas, patrones
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
	vets := vetsConEtiqueta(cuerpos)

	// DIRECCION 0, LA QUE COMPILA: el fichero sale de `go test ./...` y tambien
	// del compilador, asi que `go vet ./...` y `go build ./...` dejan de verlo.
	// Alguien tiene que mirarlo EN EL COMMIT, o un fichero roto entra en main.
	for fichero, etiqueta := range fuera {
		bloqueantes := []string{}
		for _, v := range vets[etiqueta] {
			disp := disparadoresDeWorkflow(cuerpos[v.Workflow])
			if !disp["push"] && !disp["pull_request"] {
				continue
			}
			// EL ALCANCE, QUE ES EL HUECO QUE ESTA LINEA CIERRA (20-09-2026).
			//
			// La version anterior exigia que el vet EXISTIERA y no que llegara
			// al fichero, y lo dejo escrito como hueco conocido. Se midio con
			// `go list`: `go vet -tags frescura ./nucleo/...` no incluye el
			// paquete de la raiz y `./...` si. O sea que acotar el patron
			// devuelve el fallo entero con la CI en verde: el paso existe, la
			// letra se cumple, el fichero roto no lo compila nadie y vuelve a
			// aparecer un dia despues disfrazado de fallo de frescura.
			//
			// Se exige `./...` EXACTO y no «un patron que alcance la raiz».
			// Decidir si un patron alcanza un fichero pide expandirlo, o sea
			// ejecutar `go list` desde una puerta, que es lento y ademas mete
			// una segunda implementacion de la regla de expansion de Go. `./...`
			// es lo unico que no hay que interpretar: o esta o no esta. Cuesta
			// un rojo el dia que alguien tenga una razon legitima para acotar,
			// y entonces la razon se escribe y se cambia esta linea a mano, que
			// es exactamente lo que se quiere que pase.
			if len(v.Patrones) != 1 || v.Patrones[0] != "./..." {
				t.Errorf("%s hace `go vet -tags %s` sobre %v y tiene que ser sobre `./...` "+
					"exacto (por %s).\n"+
					"  Un patron mas estrecho puede no alcanzar el fichero de la raiz, y "+
					"entonces esta puerta afirma algo que no comprueba: el paso existe, sale "+
					"verde, y el fichero etiquetado sigue sin compilarse en ningun sitio "+
					"bloqueante. Medido con `go list`: `./nucleo/...` NO trae el paquete de "+
					"la raiz y `./...` si.\n"+
					"  Se pide `./...` exacto y no «un patron que alcance la raiz» porque lo "+
					"segundo obliga a expandir el patron, o sea a reimplementar la regla de "+
					"Go dentro de un test.\n"+
					"  Arreglo: `go vet -tags %s ./...`, o, si hay una razon de verdad para "+
					"acotarlo, escribirla y cambiar esta exigencia a mano.",
					v.Workflow, etiqueta, v.Patrones, fichero, etiqueta)
				continue
			}
			bloqueantes = append(bloqueantes, v.Workflow)
		}
		if len(bloqueantes) == 0 {
			t.Errorf("%s sale del build por defecto con `//go:build %s` y NINGUN workflow "+
				"que corra en push o pull_request hace `go vet -tags %s`.\n"+
				"  Una etiqueta no saca el fichero solo de la suite: lo saca del COMPILADOR, "+
				"asi que con una llamada rota dentro `go vet ./...`, `go build ./...` y "+
				"`go test ./...` salen los tres con 0 y el fichero roto entra en main.\n"+
				"  Y lo destapa el cron hasta 24 h despues DISFRAZADO: un fallo de "+
				"compilacion dentro del trabajo que se llama «%s» se lee como que ese test "+
				"volvio a fallar por lo suyo. Aviso equivocado, commit equivocado, autor "+
				"equivocado.\n"+
				"  Arreglo: un paso `go vet -tags %s ./...` en la CI bloqueante. Es el "+
				"deber de COMPILAR, que es del commit; el de estar en verde sigue siendo "+
				"del horario.",
				fichero, etiqueta, etiqueta, etiqueta, etiqueta)
		}
	}

	// Y AL REVES: un vet con una etiqueta que ya no saca a nadie del build lleva
	// desde entonces compilando exactamente lo mismo que el vet normal.
	for etiqueta, quienes := range vets {
		usada := false
		for _, e := range fuera {
			if e == etiqueta {
				usada = true
			}
		}
		if !usada {
			donde := []string{}
			for _, v := range quienes {
				donde = append(donde, v.Workflow)
			}
			t.Errorf("%v hacen `go vet -tags %s` y ningun fichero de la raiz sale del build "+
				"por defecto con esa etiqueta.\n"+
				"  Ese paso compila lo mismo que `go vet ./...`, o sea que cuesta un minuto "+
				"de CI y no mira nada que el de al lado no mirara ya.\n"+
				"  Arreglo: o la etiqueta esta mal escrita, o el fichero que la llevaba "+
				"volvio a la suite y este paso sobra.", donde, etiqueta)
		}
	}

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
		"(%v), %d etiquetas compiladas por la CI bloqueante y %d ejecutadas por workflows "+
		"programados", dentro, len(fuera), nombresOrdenados(fuera), len(vets), len(porEtiqueta))
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

	// EL EXTRACTOR DEL VET, que es la mitad que faltaba. Su fallo probable es el
	// contrario del de arriba: confundir el vet NORMAL con uno etiquetado, y
	// entonces cualquier repositorio con un `go vet ./...` aprobaria la
	// exigencia de compilar sin compilar nada nuevo.
	t.Run("el vet etiquetado se reconoce y el vet normal no", func(t *testing.T) {
		got := vetsConEtiqueta(map[string]string{
			"ci.yml": "    steps:\n      - name: vet\n        run: go vet ./...\n" +
				"      - name: vet con etiqueta\n        run: go vet -tags frescura ./...\n",
			"otro.yml": "      - run: go vet -tags=segunda ./...\n",
		})
		if len(got["frescura"]) != 1 || got["frescura"][0].Workflow != "ci.yml" {
			t.Errorf("no saca la etiqueta de `go vet -tags frescura`: %v", got)
		}
		if len(got["segunda"]) != 1 {
			t.Errorf("no saca la etiqueta de `go vet -tags=segunda`: %v", got)
		}
		if len(got) != 2 {
			t.Errorf("el `go vet ./...` de al lado ha aportado una etiqueta fantasma: %v.\n"+
				"  Entonces el vet normal, que es el que NO ve el fichero etiquetado, "+
				"valdria como prueba de que alguien lo compila", got)
		}
	})

	// LA PROSA QUE HABLA DEL VET NO ES UN VET. Este caso no es hipotetico: esta
	// puerta se cazo a si misma con el comentario que escribi en ci.yml para
	// explicar por que existe el paso, que contiene `go vet -tags <etiqueta>`.
	// Y el peligro gordo es el otro: una linea COMENTADA con la etiqueta buena
	// cumpliria la exigencia de compilar sin que nadie compile nada.
	t.Run("un comentario que menciona el vet no cuenta como vet", func(t *testing.T) {
		got := vetsConEtiqueta(map[string]string{
			"ci.yml": "      # exige un `go vet -tags <etiqueta>` en un workflow de push\n" +
				"      # run: go vet -tags frescura ./...\n" +
				"      - name: nada\n        run: echo mira go vet -tags colada ./...\n",
		})
		if len(got) != 0 {
			t.Errorf("la prosa cuenta como cable: %v.\n"+
				"  Con `<etiqueta>` es un falso positivo que cuesta un rojo tonto. Con "+
				"`frescura` comentado es lo caro: la exigencia de compilar quedaria "+
				"cumplida por una linea que no ejecuta nada.", got)
		}
	})
	// EL ALCANCE DEL VET, que es el hueco que cerro este ultimo bloque. Medido
	// con `go list` el 20-09-2026: `./nucleo/...` no trae el paquete de la raiz
	// y `./...` si, o sea que un vet acotado deja el fichero etiquetado sin
	// compilar y devuelve el fallo entero con la CI en verde.
	t.Run("el patron de paquetes del vet se lee, y el acotado se distingue", func(t *testing.T) {
		got := vetsConEtiqueta(map[string]string{
			"ancho.yml": "      - run: go vet -tags frescura ./...\n",
			"estrecho.yml": "    steps:\n      - name: vet acotado\n" +
				"        run: go vet -tags frescura ./nucleo/...\n",
		})
		por := map[string][]string{}
		for _, v := range got["frescura"] {
			por[v.Workflow] = v.Patrones
		}
		if len(por) != 2 {
			t.Fatalf("no ve los dos vet: %v", got)
		}
		if len(por["ancho.yml"]) != 1 || por["ancho.yml"][0] != "./..." {
			t.Errorf("no lee el patron ancho: %v.\n"+
				"  Si el patron no se lee, la exigencia no se puede comprobar y la puerta "+
				"vuelve a afirmar que alguien compila el fichero sin saberlo", por)
		}
		if len(por["estrecho.yml"]) != 1 || por["estrecho.yml"][0] != "./nucleo/..." {
			t.Errorf("no distingue el patron acotado: %v.\n"+
				"  Es el caso entero: con `./nucleo/...` el fichero de la raiz no se compila, "+
				"el vet sale verde con el fichero roto dentro y el bug vuelve con la CI en "+
				"verde", por)
		}
		// Y LA BANDERA NO ES UN OPERANDO. El fallo probable del partidor es
		// contar `frescura` como patron de paquetes, y entonces `./...` seria el
		// segundo y la exigencia de «exactamente ./...» se pondria roja sobre el
		// paso correcto, que es como se acaba aflojando una puerta buena.
		for _, p := range por["ancho.yml"] {
			if p == "frescura" || strings.HasPrefix(p, "-") {
				t.Errorf("el valor de -tags o una bandera se ha colado como patron: %v", por)
			}
		}
	})
	t.Run("un vet etiquetado en un workflow SOLO programado no es CI bloqueante", func(t *testing.T) {
		// El detector de etiquetas no sabe de disparadores a proposito: los
		// separa el test grande, cruzando con disparadoresDeWorkflow. Esto
		// comprueba que las dos piezas encajan, que es donde vive el fallo.
		cuerpo := "name: x\n\non:\n  schedule:\n    - cron: '1 2 * * *'\n\njobs:\n  j:\n" +
			"    steps:\n      - run: go vet -tags frescura ./...\n"
		got := vetsConEtiqueta(map[string]string{"solo-cron.yml": cuerpo})
		if len(got["frescura"]) != 1 {
			t.Fatalf("el extractor no ve el vet: %v", got)
		}
		d := disparadoresDeWorkflow(cuerpo)
		if d["push"] || d["pull_request"] {
			t.Errorf("lo daria por bloqueante: %v.\n"+
				"  Un vet colgado del mismo cron que el test no adelanta nada: el fichero "+
				"roto seguiria entrando en main y apareciendo 24 h despues", d)
		}
	})
}
