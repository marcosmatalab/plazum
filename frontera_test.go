package plazum

import (
	"os"
	"os/exec"
	"path/filepath"
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
	prefijo := "frente_" + frente + "=\""
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
	t.Fatalf("la matriz no declara el frente %s. Si los frentes se han renombrado, esta "+
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
	suyo := primeraRutaDe(t, "A")
	ajeno := primeraRutaDe(t, "D")
	otroAjeno := primeraRutaDe(t, "C")
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

		codigo, salida := correrFrontera(t, bash, dir, "A", base, "frente-a")
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

		codigo, salida := correrFrontera(t, bash, dir, "A", base, "frente-a")
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

		codigo, salida := correrFrontera(t, bash, dir, "A", baseVieja, "frente-a")
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

		codigo, salida := correrFrontera(t, bash, dir, "A", base, "frente-a")
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
	suyo := primeraRutaDe(t, "A")
	ajeno := primeraRutaDe(t, "D")

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
