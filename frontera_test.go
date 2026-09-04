package plazum

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// LA PUERTA DE LA MATRIZ DE FRONTERAS: quien vigila a la comprobacion.
//
// # Que agujero cierra, y cuanto costo
//
// `.github/frontera.sh` decide si el merge de un frente se acepta o se rechaza.
// Nunca se habia comprobado a si misma, y el 03-09-2026 dio un FALSO POSITIVO:
// acuso al frente D de tocar 74 ficheros fuera de su columna, y 71 eran de los
// frentes A y C que ya estaban en `main`. La causa es que los frentes REBASAN
// mientras la campana corre, asi que el diff contra el inicio de la campana
// incluye todo lo que otros ya integraron.
//
// UN FALSO POSITIVO AQUI NO ES RUIDO: es rechazar el merge de un frente limpio,
// o sea tirar horas de trabajo por un script mal escrito. Y con cuatro frentes a
// la vez, es la clase de error que se comete una vez por campana.
//
// # Por eso el control negativo va en las DOS direcciones
//
// Una puerta de frontera que solo se demuestra contra un frente sucio esta medio
// probada, y la mitad que falta es justo la que fallo: que un frente LIMPIO no
// sea rechazado. Las dos mitades tienen caso aqui, y la del rebase tiene el
// suyo propio porque es el escenario exacto que produjo el falso positivo.
//
// # Se ejecuta el script de verdad, no una reimplementacion
//
// Contra un repositorio git sintetico, construido aqui. Reimplementar la logica
// en Go para probarla seria probar la copia: el dia que las dos se separen, la
// que decide los merges es la de bash y la que esta verde es la de Go.

// gitEn corre git dentro del repositorio sintetico. La identidad va por entorno
// para no depender de la configuracion global de la maquina, que en CI no hay.
func gitEn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	c.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=puerta", "GIT_AUTHOR_EMAIL=puerta@plazum.invalid",
		"GIT_COMMITTER_NAME=puerta", "GIT_COMMITTER_EMAIL=puerta@plazum.invalid",
		"GIT_CONFIG_GLOBAL="+filepath.Join(dir, "sin-config"),
		"GIT_CONFIG_SYSTEM="+filepath.Join(dir, "sin-config"),
	)
	salida, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, salida)
	}
	return string(salida)
}

// commitDe escribe un fichero y lo commitea. El contenido es el propio nombre:
// aqui lo unico que importa es QUE ruta cambia, no que dice dentro.
func commitDe(t *testing.T, dir, ruta, mensaje string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(ruta))
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(ruta+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitEn(t, dir, "add", "-A")
	gitEn(t, dir, "commit", "-m", mensaje)
}

// primeraRutaDe saca del PROPIO script una ruta que pertenece a ese frente.
//
// SE LEE DE LA MATRIZ Y NO SE ESCRIBE AQUI, y esa es la unica forma de que esta
// puerta sobreviva a la siguiente campana: la matriz cambia cada tramo, y un
// test que cablee `paquetes/soc2/` se pone rojo el dia que el frente A pasa a
// otra cosa. Un rojo asi no dice «la frontera falla», dice «el test envejecio»,
// y es la clase de rojo que se arregla borrando el test.
func primeraRutaDe(t *testing.T, frente string) string {
	t.Helper()
	b, err := os.ReadFile(".github/frontera.sh") // #nosec G304 -- ruta constante del repositorio
	if err != nil {
		t.Fatalf("no puedo leer el script de fronteras: %v", err)
	}
	prefijo := "rebanada_" + frente + "=\""
	for _, linea := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(linea, prefijo) {
			continue
		}
		cuerpo := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(linea), prefijo), "\"")
		campos := strings.Fields(cuerpo)
		if len(campos) == 0 {
			t.Fatalf("la columna del frente %s esta vacia en la matriz", frente)
		}
		ruta := campos[0]
		if strings.HasSuffix(ruta, "/") {
			return ruta + "cualquiera.txt"
		}
		return ruta
	}
	t.Fatalf("la matriz no declara la rebanada %s. Si se han renombrado, esta "+
		"puerta hay que reapuntarla, no borrarla", frente)
	return ""
}

// repoDeFrontera monta un repositorio con la MISMA `.github/frontera.sh` que usa
// la campana. Se copia el fichero real: una copia editada probaria otra cosa.
func repoDeFrontera(t *testing.T) string {
	t.Helper()
	origen, err := os.ReadFile(".github/frontera.sh") // #nosec G304 -- ruta constante del repositorio
	if err != nil {
		t.Fatalf("no puedo leer el script de fronteras: %v", err)
	}
	dir := t.TempDir()
	gitEn(t, dir, "init", "-b", "main")
	destino := filepath.Join(dir, ".github")
	if err := os.MkdirAll(destino, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destino, "frontera.sh"), origen, 0o700); err != nil { // #nosec G302 -- script ejecutable en un temporal del test
		t.Fatal(err)
	}
	gitEn(t, dir, "add", "-A")
	gitEn(t, dir, "commit", "-m", "arranque")
	return dir
}

// correrFrontera devuelve el codigo de salida y la salida completa.
//
// EL CODIGO SE LEE DEL PROCESO Y NO DE UNA TUBERIA. Leer un codigo de salida a
// traves de un `head` da el de `head`, que es 0 casi siempre: ya paso el
// 03-09-2026 y enseno una frontera rota como verde.
func correrFrontera(t *testing.T, bash, dir string, args ...string) (int, string) {
	t.Helper()
	c := exec.Command(bash, append([]string{".github/frontera.sh"}, args...)...)
	c.Dir = dir
	salida, err := c.CombinedOutput()
	if err == nil {
		return 0, string(salida)
	}
	var salir *exec.ExitError
	if ok := asExitError(err, &salir); ok {
		return salir.ExitCode(), string(salida)
	}
	t.Fatalf("no he podido ejecutar el script: %v\n%s", err, salida)
	return -1, ""
}

func asExitError(err error, destino **exec.ExitError) bool {
	e, ok := err.(*exec.ExitError)
	if ok {
		*destino = e
	}
	return ok
}

// TestLaMatrizDeFronterasAcusaAlSucioYNoAlLimpio recorre las dos direcciones.
func TestLaMatrizDeFronterasAcusaAlSucioYNoAlLimpio(t *testing.T) {
	bash := buscarBash(t)
	suyo := primeraRutaDe(t, "1")
	ajeno := primeraRutaDe(t, "0")
	otroAjeno := primeraRutaDe(t, "2")
	if suyo == ajeno || suyo == otroAjeno {
		t.Fatalf("dos frentes de la matriz empiezan por la misma ruta (%q), asi que este "+
			"test no puede distinguir lo propio de lo ajeno y estaria midiendo el vacio",
			suyo)
	}

	t.Run("un frente dentro de su columna NO se rechaza", func(t *testing.T) {
		dir := repoDeFrontera(t)
		base := strings.TrimSpace(gitEn(t, dir, "rev-parse", "HEAD"))
		gitEn(t, dir, "checkout", "-b", "frente-a")
		commitDe(t, dir, suyo, "el frente A escribe lo suyo")

		codigo, salida := correrFrontera(t, bash, dir, "1", base, "frente-a")
		if codigo != 0 {
			t.Errorf("un frente limpio ha sido RECHAZADO (codigo %d):\n%s\n"+
				"  Esta es la mitad cara del error: rechazar a quien no rompio nada "+
				"significa tirar su trabajo por un fallo de quien integra.", codigo, salida)
		}
	})

	t.Run("un fichero fuera de su columna SI se rechaza, y se nombra", func(t *testing.T) {
		dir := repoDeFrontera(t)
		base := strings.TrimSpace(gitEn(t, dir, "rev-parse", "HEAD"))
		gitEn(t, dir, "checkout", "-b", "frente-a")
		commitDe(t, dir, suyo, "lo suyo")
		commitDe(t, dir, ajeno, "y lo que no es suyo")

		codigo, salida := correrFrontera(t, bash, dir, "1", base, "frente-a")
		if codigo == 0 {
			t.Fatalf("un frente que toca la columna de otro ha pasado:\n%s", salida)
		}
		if !strings.Contains(salida, ajeno) {
			t.Errorf("el rechazo no nombra el fichero infractor, asi que no es accionable:\n%s",
				salida)
		}
		if strings.Contains(salida, suyo) {
			t.Errorf("el rechazo acusa tambien a un fichero que SI es suyo:\n%s\n"+
				"  Una lista con inocentes dentro se lee por encima, que es como se cuela "+
				"la unica linea que importaba.", salida)
		}
	})

	// EL CASO QUE PRODUJO EL FALSO POSITIVO, Y ES EL QUE JUSTIFICA EL SCRIPT
	// ENTERO. El frente rebasa sobre un `main` que ya trae trabajo de otro. Su
	// diff contra la base VIEJA incluye lo de ese otro, y sin merge-base contra
	// la cabeza de integracion la comprobacion le acusa de tocarlo.
	t.Run("un frente rebasado sobre main no hereda la culpa de lo ya integrado", func(t *testing.T) {
		dir := repoDeFrontera(t)
		baseVieja := strings.TrimSpace(gitEn(t, dir, "rev-parse", "HEAD"))

		gitEn(t, dir, "checkout", "-b", "frente-a")
		commitDe(t, dir, suyo, "lo del frente A")

		// Mientras tanto, otro frente ya se integro en main.
		gitEn(t, dir, "checkout", "main")
		commitDe(t, dir, ajeno, "el frente D, ya fusionado")
		commitDe(t, dir, otroAjeno, "el frente C, ya fusionado")

		gitEn(t, dir, "checkout", "frente-a")
		gitEn(t, dir, "rebase", "main")

		codigo, salida := correrFrontera(t, bash, dir, "1", baseVieja, "frente-a")
		if codigo != 0 {
			t.Errorf("un frente rebasado sobre main ha sido rechazado por trabajo AJENO "+
				"que ya estaba integrado (codigo %d):\n%s\n"+
				"  Es exactamente el falso positivo del 03-09-2026: 74 ficheros «fuera de "+
				"su columna» de los que 71 eran de otros frentes, ya en main.\n"+
				"  La base se calcula con merge-base contra la rama de integracion de "+
				"AHORA, no contra la referencia que le pasen: si esa referencia es vieja, "+
				"el merge-base con ella sigue siendo ella.", codigo, salida)
		}
	})

	// Y LA GUARDA DEL VERDE VACIO, que es la familia de `go test` saliendo 0
	// cuando el patron no casa nada.
	t.Run("una rama sin trabajo no cuenta como frontera respetada", func(t *testing.T) {
		dir := repoDeFrontera(t)
		base := strings.TrimSpace(gitEn(t, dir, "rev-parse", "HEAD"))
		gitEn(t, dir, "checkout", "-b", "frente-a")

		codigo, salida := correrFrontera(t, bash, dir, "1", base, "frente-a")
		if codigo == 0 {
			t.Errorf("una rama que no cambia nada ha dado «frontera respetada»:\n%s\n"+
				"  Cero ficheros fuera de la columna es literalmente cierto y no significa "+
				"nada: significa que no hay trabajo, o que la base esta mal.", salida)
		}
	})
}

// TestElCruceEncuentraDosFrentesQueSePisanYNoInventaUnoQueNo es el segundo
// sentido de la matriz, con sus dos direcciones tambien.
//
// Este sentido NO lo ve el primero: dos frentes pueden estar los dos DENTRO de
// su columna y aun asi tocar el mismo fichero, si la matriz esta mal escrita.
func TestElCruceEncuentraDosFrentesQueSePisanYNoInventaUnoQueNo(t *testing.T) {
	bash := buscarBash(t)
	suyo := primeraRutaDe(t, "1")
	ajeno := primeraRutaDe(t, "0")

	t.Run("columnas disjuntas: sin cruces", func(t *testing.T) {
		dir := repoDeFrontera(t)
		base := strings.TrimSpace(gitEn(t, dir, "rev-parse", "HEAD"))
		gitEn(t, dir, "checkout", "-b", "frente-a")
		commitDe(t, dir, suyo, "A")
		gitEn(t, dir, "checkout", "main")
		gitEn(t, dir, "checkout", "-b", "frente-d")
		commitDe(t, dir, ajeno, "D")

		codigo, salida := correrFrontera(t, bash, dir, "--cruce", base, "frente-a", "frente-d")
		if codigo != 0 {
			t.Errorf("dos frentes con columnas disjuntas han dado cruce (codigo %d):\n%s",
				codigo, salida)
		}
	})

	t.Run("el mismo fichero en dos ramas: cruce, y se nombra", func(t *testing.T) {
		dir := repoDeFrontera(t)
		base := strings.TrimSpace(gitEn(t, dir, "rev-parse", "HEAD"))
		gitEn(t, dir, "checkout", "-b", "frente-a")
		commitDe(t, dir, suyo, "A escribe")
		gitEn(t, dir, "checkout", "main")
		gitEn(t, dir, "checkout", "-b", "frente-b")
		commitDe(t, dir, suyo, "B escribe lo mismo")

		codigo, salida := correrFrontera(t, bash, dir, "--cruce", base, "frente-a", "frente-b")
		if codigo == 0 {
			t.Fatalf("dos ramas que escriben el MISMO fichero no han dado cruce:\n%s", salida)
		}
		if !strings.Contains(salida, suyo) {
			t.Errorf("el cruce no dice que fichero se pisan:\n%s", salida)
		}
	})

	t.Run("una sola rama con contenido no es un cruce sin cruces", func(t *testing.T) {
		dir := repoDeFrontera(t)
		base := strings.TrimSpace(gitEn(t, dir, "rev-parse", "HEAD"))
		gitEn(t, dir, "checkout", "-b", "frente-a")
		commitDe(t, dir, suyo, "A")
		gitEn(t, dir, "checkout", "main")
		gitEn(t, dir, "checkout", "-b", "frente-vacio")

		codigo, salida := correrFrontera(t, bash, dir, "--cruce", base, "frente-a", "frente-vacio")
		if codigo == 0 {
			t.Errorf("con una sola rama con contenido ha dicho «sin cruces», que es un verde "+
				"vacio:\n%s", salida)
		}
	})
}

// EL SOLAPE DECLARADO TIENE QUE SER EXACTAMENTE EL DECLARADO, NI UNO MAS.
//
// # Por que hace falta una puerta para esto
//
// La matriz del tramo 2 declara UN solape a proposito (el catalogo de cadenas,
// entre la rebanada de la nota y la de los valores) y lo resuelve en el tiempo:
// la 1 entra en main antes de que arranque la 3. Esa decision es defendible.
//
// Lo que no es defendible es el SEGUNDO solape, el que entra sin que nadie lo
// decida, porque a partir de ese momento la frase «el unico solape declarado»
// del comentario de la matriz es falsa y nadie se entera: `--cruce` solo mira
// las ramas VIVAS, asi que un solape entre dos columnas que todavia no tienen
// rama no lo ve nadie hasta que ya hay trabajo dentro de las dos.
//
// Es exactamente el fallo del tramo 1 contado desde antes: la particion mal
// hecha se descubrio al fusionar, que es el momento mas caro.
//
// # Las dos direcciones, y la segunda es la que importa
//
// Si SOBRA un solape, la particion esta mal y hay que rehacerla antes de
// empezar. Si FALTA, alguien resolvio el que estaba declarado y no lo dijo
// aqui, y entonces la serializacion de la campana (la 1 antes que la 3) ya no
// hace falta y se esta pagando runway por nada.
func TestElUnicoSolapeDeLaMatrizEsElDeclarado(t *testing.T) {
	b, err := os.ReadFile(".github/frontera.sh") // #nosec G304 -- ruta constante del repositorio
	if err != nil {
		t.Fatalf("no puedo leer el script de fronteras: %v", err)
	}
	columnas := map[string][]string{}
	for _, linea := range strings.Split(string(b), "\n") {
		if !strings.HasPrefix(linea, "rebanada_") {
			continue
		}
		i := strings.Index(linea, "=\"")
		if i < 0 {
			continue
		}
		nombre := strings.TrimPrefix(linea[:i], "rebanada_")
		cuerpo := strings.TrimSuffix(linea[i+2:], "\"")
		columnas[nombre] = strings.Fields(cuerpo)
	}
	if len(columnas) < 2 {
		t.Fatalf("la matriz declara %d rebanadas y hacen falta al menos dos para que "+
			"haya solape que buscar. Esta puerta estaria dando un verde vacio", len(columnas))
	}

	// LO DECLARADO. Se escribe aqui, al lado de la afirmacion que vigila, y no
	// se lee del comentario del script: un comentario no es un dato, y leerlo
	// convertiria esta puerta en «el comentario dice lo que el comentario
	// dice», que es preguntarle a la respuesta por la respuesta.
	//
	// EL TRAMO 3 NO DECLARA NINGUNO, y llegar a cero costo rehacer la particion
	// antes de empezar: el catalogo de cadenas se va entero con el unico frente
	// de pantallas del tramo, y el frente de IA no toca pantalla a proposito.
	// El tramo 2 tuvo uno y lo resolvio en el tiempo (serializando dos
	// rebanadas); esa serializacion se quita al quitarse el solape, que es lo
	// que esta puerta exigio en voz alta cuando el mapa se vacio.
	declarados := map[string]string{}

	var nombres []string
	for n := range columnas {
		nombres = append(nombres, n)
	}
	sort.Strings(nombres)

	hallados := map[string]string{}
	pares := 0
	for i := 0; i < len(nombres); i++ {
		for j := i + 1; j < len(nombres); j++ {
			pares++
			a, z := nombres[i], nombres[j]
			for _, pa := range columnas[a] {
				for _, pz := range columnas[z] {
					// Solapan si son la misma ruta o si una es prefijo de
					// directorio de la otra: `superficies/` y
					// `superficies/calendario/` se pisan aunque no sean
					// iguales, y ese es el caso que se cuela leyendo por
					// encima.
					if pa == pz || strings.HasPrefix(pa, pz) || strings.HasPrefix(pz, pa) {
						hallados[a+"|"+z] = pa
					}
				}
			}
		}
	}

	for par, ruta := range hallados {
		esperada, ok := declarados[par]
		if !ok {
			t.Errorf("las rebanadas %s comparten %q y la matriz no lo declara.\n"+
				"  UN SOLAPE NO DECLARADO ES UNA PARTICION MAL HECHA, y se descubre al "+
				"fusionar, que es el momento mas caro. `--cruce` no lo ve: ese solo mira "+
				"las ramas vivas, asi que dos columnas que se pisan sin rama todavia pasan "+
				"desapercibidas hasta que las dos tienen trabajo dentro.\n"+
				"  O se reparte el fichero, o se declara aqui con como se resuelve.",
				strings.ReplaceAll(par, "|", " y "), ruta)
			continue
		}
		if esperada != ruta {
			t.Errorf("las rebanadas %s solapan en %q y lo declarado era %q",
				strings.ReplaceAll(par, "|", " y "), ruta, esperada)
		}
	}
	for par, ruta := range declarados {
		if _, ok := hallados[par]; !ok {
			t.Errorf("la matriz declara que las rebanadas %s solapan en %q y ya no solapan.\n"+
				"  Si alguien lo ha resuelto, se quita de aqui Y se quita la serializacion "+
				"que ese solape justificaba: la campana esta pagando runway por una "+
				"restriccion que ya no existe.", strings.ReplaceAll(par, "|", " y "), ruta)
		}
	}
	// CON CERO SOLAPES DECLARADOS, EL VERDE TIENE QUE VENIR DE HABER MIRADO.
	//
	// Los dos bucles de arriba recorren `declarados` y `hallados`, y si los dos
	// estan vacios ninguno hace nada: el test pasaria sin haber comparado una
	// sola pareja de columnas. Es el verde vacio otra vez, y aparece justo
	// cuando la particion es BUENA, que es cuando menos se mira.
	//
	// Se exige que se hayan comparado todas las parejas que hay: con N
	// rebanadas son N(N-1)/2.
	esperadas := len(nombres) * (len(nombres) - 1) / 2
	if pares != esperadas || pares == 0 {
		t.Fatalf("se han comparado %d parejas de columnas y con %d rebanadas hay %d.\n"+
			"  Un verde sin haber comparado nada no dice que la particion sea buena: dice "+
			"que no se ha mirado.", pares, len(nombres), esperadas)
	}
	t.Logf("solapes: %d hallados, %d declarados, %d parejas comparadas",
		len(hallados), len(declarados), pares)
}
