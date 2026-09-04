package plazum

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// LA PUERTA DEL RECIPIENTE VACIO.
//
// # La cicatriz
//
// Hasta el 04-09-2026 `go.sum` estaba RASTREADO por git y tenia CERO lineas. No
// llego asi por diseno: llego por una contaminacion. Construir cosign dentro de
// este arbol metio 529 lineas de sumas ajenas en el fichero, y la limpieza saco
// las lineas y dejo el fichero, vacio y rastreado.
//
// Un fichero vacio y rastreado no es neutro: es una INVITACION. Existe, git lo
// vigila, y la proxima herramienta que pase escribiendo sumas lo va a llenar
// sin CREAR nada, o sea sin aparecer como fichero nuevo para quien mira `git
// status` por encima. Quitado el contenido, la forma de la contaminacion se
// quedo entera.
//
// Un modulo con cero dependencias no necesita `go.sum`: comprobado el
// 04-09-2026 en un modulo de juguete con este mismo `go.mod`, `go mod tidy` no
// lo crea y `go mod tidy -diff` no lo echa de menos.
//
// # Por que esta puerta tiene DOS DIRECCIONES, y la segunda es la que la
// justifica
//
// La direccion facil es la de hoy: cero dependencias, luego `go.sum` fuera del
// indice y en `.gitignore`.
//
// La otra es la que importa, porque es la que convierte esa linea de
// `.gitignore` de riesgo en guarda. **Ignorar `go.sum` es correcto exactamente
// mientras no haya dependencias, y es un agujero el minuto siguiente**:
// `go.sum` fija los hashes de lo que se descarga, o sea que es la unica defensa
// del modulo contra que aguas arriba cambie por debajo. Ignorado y con
// dependencias dentro, nadie podria reconstruir lo que se compilo, y la linea
// habria pasado de higiene a agujero sin que nadie la tocara y sin que nada
// avisara.
//
// Asi que la puerta lee `go.mod` y cambia de exigencia con el:
//
//	sin require   ->  go.sum NO rastreado  Y  ignorado
//	con require   ->  go.sum SI rastreado  Y  NO ignorado
//
// El dia que entre la primera dependencia esto se pone rojo pidiendo que se
// quite la linea del `.gitignore`, en el mismo commit que su fila de
// DEPENDENCIAS.md y que el cambio a proposito de
// TestElBinarioNoLlevaNingunaDependenciaExterna. Que ese dia haya que tocar
// tres cosas a la vez es justo lo que se busca.
//
// Es el invariante 8 en su forma de proceso: el valor cero de esta
// configuracion —un fichero que sigue ahi porque nadie lo quito— tenia que ser
// el restrictivo y era el permisivo.

// reRequire reconoce una dependencia de verdad en go.mod, en las dos formas que
// admite el formato: la linea suelta y el bloque.
//
// SE ANCLA AL PRINCIPIO DE LINEA Y SE EXIGE ALGO DETRAS, porque el fallo
// probable de esta expresion es cazar la palabra dentro de un comentario. Este
// go.mod no tiene comentarios hoy, pero una puerta que depende de que un
// fichero no gane un comentario no es una puerta.
var reRequire = regexp.MustCompile(`(?m)^require\s+[^\s(]`)

var reBloqueRequire = regexp.MustCompile(`(?ms)^require\s+\(\s*(.*?)^\)`)

// hayDependencias dice si go.mod declara alguna. Devuelve tambien lo que ha
// leido, para que el mensaje de error pueda ensenarlo.
func hayDependencias(t *testing.T) (bool, string) {
	t.Helper()
	b, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("no puedo leer go.mod (%v). Sin el, esta puerta no sabe cual de sus dos "+
			"direcciones aplica, y elegir una a ciegas seria peor que no comprobar", err)
	}
	texto := string(b)
	if reRequire.MatchString(texto) {
		return true, texto
	}
	if m := reBloqueRequire.FindStringSubmatch(texto); m != nil {
		for _, l := range strings.Split(m[1], "\n") {
			l = strings.TrimSpace(l)
			if l != "" && !strings.HasPrefix(l, "//") {
				return true, texto
			}
		}
	}
	return false, texto
}

// rastreado pregunta a git, no al disco. Un go.sum que existe en el directorio
// de trabajo pero no esta en el indice es exactamente el estado bueno cuando
// una herramienta acaba de escribirlo: lo que se vigila es el INDICE.
func rastreado(t *testing.T, ruta string) bool {
	t.Helper()
	salida, err := exec.Command("git", "ls-files", "--", ruta).Output()
	if err != nil {
		t.Fatalf("no puedo preguntar a git si %s esta rastreado (%v). No saberlo NO es "+
			"que no lo este: es el invariante 8 aplicado a esta misma puerta", ruta, err)
	}
	return strings.TrimSpace(string(salida)) != ""
}

// ignorado consulta las reglas de ignorado de verdad, con `git check-ignore`,
// en vez de buscar una cadena en .gitignore. Buscar la cadena daria por buena
// una linea comentada, una linea en otra seccion o una negacion posterior.
//
// SE DISTINGUEN LOS TRES CODIGOS DE SALIDA, que es donde esta la trampa: 0 es
// «ignorado», 1 es «no ignorado» y cualquier otro es «no se ha podido
// preguntar», que no es ninguna de las dos.
func ignorado(t *testing.T, ruta string) bool {
	t.Helper()
	cmd := exec.Command("git", "check-ignore", "--no-index", "-q", "--", ruta)
	err := cmd.Run()
	if err == nil {
		return true
	}
	if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
		return false
	}
	t.Fatalf("no puedo preguntar a git si %s esta ignorado (%v). Un fallo de la consulta "+
		"no se puede leer como «no ignorado»: seria dar por comprobada la direccion "+
		"peligrosa sin haberla mirado", ruta, err)
	return false
}

func TestGoSumNoEsUnRecipienteVacio(t *testing.T) {
	conDeps, goMod := hayDependencias(t)
	estaRastreado := rastreado(t, "go.sum")
	estaIgnorado := ignorado(t, "go.sum")

	if conDeps {
		// LA DIRECCION QUE HOY NO SE RECORRE, escrita igual de completa. Un
		// descargo que ninguna entrada alcanza es un descargo que no existe, y
		// el control positivo que si lo recorre esta mas abajo.
		if !estaRastreado {
			t.Errorf(`go.mod declara dependencias y go.sum NO esta rastreado.

  go.sum fija el hash de cada modulo que se descarga: es la unica defensa de
  este repositorio contra que aguas arriba cambie por debajo. Sin el en el
  indice, nadie puede reconstruir lo que se compilo.

  Arreglo: quitar la linea /go.sum de .gitignore, `+"`go mod tidy`"+`, y
  `+"`git add go.sum`"+` en el mismo commit que la fila de DEPENDENCIAS.md.

go.mod:
%s`, goMod)
		}
		if estaIgnorado {
			t.Error(`go.mod declara dependencias y go.sum sigue en .gitignore.

  Esa linea se escribio el 04-09-2026 cuando el modulo tenia cero dependencias,
  y su propio comentario dice que caduca en cuanto entre la primera. Hoy ha
  entrado. Quitala: un go.sum ignorado con dependencias dentro es un agujero de
  suministro, no higiene.`)
		}
		return
	}

	// LA DIRECCION DE HOY.
	if estaRastreado {
		t.Error(`go.sum esta rastreado y go.mod no declara ninguna dependencia.

  Un modulo con cero dependencias no necesita go.sum: go mod tidy no lo crea.
  Si esta en el indice, o esta VACIO —y entonces es el recipiente que dejo la
  contaminacion de cosign, esperando a que la proxima herramienta lo llene sin
  que aparezca como fichero nuevo— o tiene lineas que nadie ha declarado en
  DEPENDENCIAS.md, que es lista cerrada.

  Arreglo: git rm --cached go.sum, y comprobar que DEPENDENCIAS.md sigue
  diciendo cero.`)
	}
	if !estaIgnorado {
		t.Error(`go.sum no esta en .gitignore y el modulo no tiene dependencias.

  Sin la linea, la proxima herramienta que escriba sumas en este arbol deja un
  fichero sin seguimiento que es facil de aniadir sin mirar. Con la linea, no
  se puede aniadir por descuido: hay que forzarlo.

  Arreglo: aniadir /go.sum a .gitignore, con el comentario que dice cuando
  caduca.`)
	}
}

// EL CONTROL NEGATIVO, en las DOS direcciones, sobre el lector y no sobre el
// arbol.
//
// El fallo probable de esta puerta no es el veredicto: es `hayDependencias`
// contestando que no cuando si. Si se equivoca en esa direccion, la puerta
// exigiria para siempre el regimen de «cero dependencias» y daria verde
// justamente el dia que hubiera algo que fijar, que es el unico dia que
// importa.
func TestElLectorDeGoModDistingueLasDosDirecciones(t *testing.T) {
	casos := []struct {
		nombre string
		texto  string
		quiero bool
	}{
		{"el go.mod de hoy, sin nada", "module x\n\ngo 1.24\n", false},
		{"un require suelto", "module x\n\ngo 1.24\n\nrequire modernc.org/sqlite v1.0.0\n", true},
		{"un bloque con una fila", "module x\n\ngo 1.24\n\nrequire (\n\tmodernc.org/sqlite v1.0.0\n)\n", true},
		{"un bloque VACIO no es una dependencia", "module x\n\ngo 1.24\n\nrequire (\n)\n", false},
		{"un bloque con solo un comentario tampoco", "module x\n\ngo 1.24\n\nrequire (\n\t// pendiente\n)\n", false},
		{"la palabra dentro de un comentario no cuenta", "module x\n\n// require modernc.org/sqlite\ngo 1.24\n", false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			hay := reRequire.MatchString(c.texto)
			if !hay {
				if m := reBloqueRequire.FindStringSubmatch(c.texto); m != nil {
					for _, l := range strings.Split(m[1], "\n") {
						l = strings.TrimSpace(l)
						if l != "" && !strings.HasPrefix(l, "//") {
							hay = true
							break
						}
					}
				}
			}
			if hay != c.quiero {
				t.Errorf("con %q he dicho %v y esperaba %v", c.texto, hay, c.quiero)
			}
		})
	}
}
