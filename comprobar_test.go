package plazum

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// La auditoria del LAZO LOCAL. `puertas_test.go` vigila que ningun workflow
// invoque `go test` a pelo; este vigila lo mismo en el sitio donde la trampa
// mordio de verdad, que es la maquina de desarrollo.
//
// POR QUE EXISTE. La tercera mordida de la misma trampa no fue en CI: fue un
// `go test . -run "Paquetes"` que salio `ok` porque el patron no casaba con el
// test que importaba, y ese `ok` se llevo a un informe. La puerta estaba puesta,
// pero estaba a diez minutos y un empujon de distancia del sitio donde se decide
// si algo esta hecho.
//
// El arreglo es `comprobar.sh`, el objetivo unico local, que NO declara las
// puertas sino que las LEE de los workflows. Este fichero es lo que impide que
// esa lectura se quede vieja en silencio, que es la unica forma en la que un
// script asi falla.

const rutaComprobar = "comprobar.sh"

func comprobar(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(rutaComprobar)
	if err != nil {
		t.Fatalf("no esta el objetivo unico local (%s): %v.\n"+
			"  Sin el, el lazo de desarrollo vuelve a ser `go test` a pelo, que sale con "+
			"codigo 0 cuando no ha comprobado nada", rutaComprobar, err)
	}
	return string(b)
}

// puertasDeclaradasEnCI cuenta las invocaciones de puerta() de todos los
// workflows, contando igual que las cuenta comprobar.sh.
func puertasDeclaradasEnCI(t *testing.T) int {
	t.Helper()
	n := 0
	for _, cuerpo := range workflows(t) {
		for _, linea := range strings.Split(cuerpo, "\n") {
			if strings.HasPrefix(strings.TrimSpace(linea), "puerta \"") {
				n++
			}
		}
	}
	return n
}

var reEsperadas = regexp.MustCompile(`(?m)^PUERTAS_ESPERADAS=(\d+)`)

// La puerta central: el lazo local corre TODAS las puertas de CI.
//
// comprobar.sh las lee de los workflows, asi que en cuanto la lectura funcione
// las corre todas por construccion. Lo que puede pasar sin que nadie lo note es
// que la lectura deje de funcionar (cambia la forma de la invocacion, se mueve
// el directorio) y el script se ponga a correr el vacio. Contra eso el script
// declara PUERTAS_ESPERADAS y compara; este test comprueba que ese numero es el
// de hoy, en los dos sentidos: si CI gana una puerta y el lazo local no se
// entera, esto se pone rojo.
func TestElLazoLocalCorreTodasLasPuertasDeCI(t *testing.T) {
	s := comprobar(t)
	m := reEsperadas.FindStringSubmatch(s)
	if m == nil {
		t.Fatalf("%s no declara PUERTAS_ESPERADAS. Sin ese numero, el dia que la extraccion "+
			"deje de casar el script correra CERO puertas y saldra verde, que es exactamente "+
			"la familia de fallos contra la que se escribio", rutaComprobar)
	}
	declaradas, err := strconv.Atoi(m[1])
	if err != nil {
		t.Fatal(err)
	}
	enCI := puertasDeclaradasEnCI(t)
	if enCI == 0 {
		t.Fatal("no se ha encontrado ni una llamada a puerta() en los workflows. O nadie las " +
			"usa, o este test ha dejado de reconocerlas, y las dos cosas son el mismo problema")
	}
	if declaradas != enCI {
		t.Errorf("%s declara PUERTAS_ESPERADAS=%d y en .github/workflows/*.yml hay %d.\n"+
			"  Si CI ha ganado una puerta, el lazo local no la esta corriendo y un verde local\n"+
			"  ya no dice lo que decia. Si la ha perdido, alguien ha recortado la vigilancia.\n"+
			"  Arreglo: poner PUERTAS_ESPERADAS a %d en el mismo commit, y decir por que cambio.",
			rutaComprobar, declaradas, enCI, enCI)
	}
	t.Logf("%d puertas en CI, %d declaradas en el lazo local", enCI, declaradas)
}

// El lazo local pasa por puerta.sh, no por go test.
//
// Es la misma regla que TestNingunWorkflowInvocaGoTestSinContarLosCasos aplica a
// CI, y esta aqui porque el sitio donde se afirma "esta hecho" es este, no aquel.
func TestElLazoLocalNoInvocaGoTestSinContarLosCasos(t *testing.T) {
	s := comprobar(t)
	if !strings.Contains(s, "source .github/puerta.sh") {
		t.Errorf("%s no hace source de .github/puerta.sh, asi que no cuenta casos", rutaComprobar)
	}
	if !strings.Contains(s, "cerrar_puertas") {
		t.Errorf("%s no llama a cerrar_puertas, asi que nadie agrega el veredicto", rutaComprobar)
	}
	for n, linea := range strings.Split(s, "\n") {
		limpia := strings.TrimSpace(linea)
		if strings.HasPrefix(limpia, "#") {
			continue // un comentario que cita el comando no es una invocacion
		}
		if i := strings.Index(limpia, "#"); i >= 0 {
			limpia = strings.TrimSpace(limpia[:i])
		}
		if strings.Contains(limpia, "go test ") {
			t.Errorf("%s:%d invoca go test directamente:\n    %s\n"+
				"  Es justo lo que este fichero existe para impedir: `go test` sale con codigo 0\n"+
				"  cuando el patron -run no casa con nada y cuando el glob no tiene tests.\n"+
				"  Arreglo: que la invocacion pase por puerta(), que cuenta los casos.",
				rutaComprobar, n+1, limpia)
		}
	}
}

// Una puerta de CI que corre con entorno propio y que en local corre SIN el no es
// la misma puerta: sale verde comprobando otra cosa. Hoy solo hay una
// (PLAZUM_SIN_IA, invariante 9). El dia que aparezca otra, esto se pone rojo
// hasta que se declare en el lazo local.
//
// Se miran los `env:` de PASO, no los de workflow: los de workflow de este repo
// son presupuestos (LIMITE_BINARIO_MB, PRESUPUESTO_TTFV_S) que consumen pasos
// que no son puertas, y meterlos aqui seria ruido, no vigilancia.
func TestTodaPuertaConEntornoPropioLoDeclaraElLazoLocal(t *testing.T) {
	local := comprobar(t)
	vistas := 0
	for nombre, cuerpo := range workflows(t) {
		for _, paso := range pasosDeWorkflow(cuerpo) {
			if !strings.Contains(paso, "puerta \"") {
				continue
			}
			for _, v := range variablesDeEntorno(paso) {
				vistas++
				if !strings.Contains(local, v) {
					t.Errorf("%s corre una puerta con %s puesta y %s no lo sabe.\n"+
						"  En local esa puerta correria sin esa variable, o sea comprobando otra\n"+
						"  cosa que la de CI, y saldria verde igual.\n"+
						"  Arreglo: declararla en el `case` de comprobar.sh.",
						nombre, v, rutaComprobar)
				}
			}
		}
	}
	t.Logf("%d variables de entorno de paso gobiernan una puerta", vistas)
}

// pasosDeWorkflow parte el cuerpo por `- name:`, que es donde empieza un paso.
func pasosDeWorkflow(cuerpo string) []string {
	lineas := strings.Split(cuerpo, "\n")
	var out []string
	var actual []string
	for _, l := range lineas {
		if strings.HasPrefix(strings.TrimSpace(l), "- name:") {
			if len(actual) > 0 {
				out = append(out, strings.Join(actual, "\n"))
			}
			actual = nil
		}
		actual = append(actual, l)
	}
	if len(actual) > 0 {
		out = append(out, strings.Join(actual, "\n"))
	}
	return out
}

var reVariable = regexp.MustCompile(`^([A-Z][A-Z0-9_]*):`)

// variablesDeEntorno saca los nombres del bloque `env:` de un paso.
func variablesDeEntorno(paso string) []string {
	lineas := strings.Split(paso, "\n")
	sangria := func(s string) int { return len(s) - len(strings.TrimLeft(s, " ")) }
	var out []string
	for i, l := range lineas {
		if strings.TrimSpace(l) != "env:" {
			continue
		}
		clave := sangria(l)
		for j := i + 1; j < len(lineas); j++ {
			if strings.TrimSpace(lineas[j]) == "" {
				continue
			}
			if sangria(lineas[j]) <= clave {
				break
			}
			if m := reVariable.FindStringSubmatch(strings.TrimSpace(lineas[j])); m != nil {
				out = append(out, m[1])
			}
		}
	}
	return out
}

// Y que el propio script se pueda parsear. Es la misma comprobacion que
// TestTodoPasoDeCIEsShellQueBashSabeParsear hace con los pasos de CI, por el
// mismo motivo: un script con un error de sintaxis no comprueba nada de lo que
// dice comprobar, y en Windows el error solo se ve al ejecutarlo.
func TestElLazoLocalEsShellQueBashSabeParsear(t *testing.T) {
	bash := buscarBash(t)
	if err := bashParsea(t, bash, comprobar(t)); err != "" {
		t.Errorf("%s no parsea:\n%s", rutaComprobar, err)
	}
}
