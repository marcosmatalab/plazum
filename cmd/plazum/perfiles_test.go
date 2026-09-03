package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// EL TRINQUETE DE LOS PERFILES: ninguna designacion del corpus se queda sin
// contestar en un perfil de arranque.
//
// # El agujero que cierra
//
// Un perfil de arranque monta hechos SUPUESTOS a partir de tres datos gruesos.
// Los relojes que cuelgan de un hecho que ningun perfil afirma NO SE VEN desde
// ningun arranque: existen en el corpus, estan probados, y quien teclea
// `plazum calendario --pais=ES --sector=...` no los ve nunca. Medido el
// 03-09-2026: 3 de los 15 relojes que entraron el dia anterior estaban asi.
//
// Y no era un olvido de nadie en particular: el hueco no tiene forma de
// aparecer. El perfil no falla, el corpus no falla, el calendario sale bien; lo
// unico que pasa es que una parte del corpus es invisible desde el unico camino
// que un comprador recorre en sus primeros diez segundos.
//
// # Las dos respuestas validas, y la tercera que se prohibe
//
//	AFIRMARLO      el perfil lo pone en sus hechos o en una de sus bandas, con
//	               su `porque` y su confianza. Solo vale cuando el sector de
//	               verdad lo implica.
//	NO SUPONERLO   el perfil lo nombra en `no_supone`, que es la mitad util del
//	               diseno: quien lee «no me sale DORA» sin saber por que asume
//	               que el producto no lo tiene.
//	CALLARLO       prohibido. Es indistinguible de las otras dos para quien lo
//	               lea, y es lo que produjo los tres relojes invisibles.
//
// # Por que la lista de designaciones se LEE del corpus
//
// Escribirla aqui seria una segunda copia, y ademas cablearia identificadores de
// normas en un fichero de codigo, que es el invariante 2. Se saca de las reglas
// de aplicabilidad de los paquetes instalados: el dia que un paquete nuevo pida
// una designacion, esta puerta la pide en los tres perfiles sin que nadie tenga
// que acordarse.

// designacionEnRegla caza `designado(E, "loquesea")` dentro del texto de una
// regla. Se lee de la regla y no de una lista, por lo de arriba.
var designacionEnRegla = regexp.MustCompile(`designado\(\s*\w+\s*,\s*"([a-z0-9_]+)"\s*\)`)

// designacionesDelCorpus enumera las designaciones que alguna regla exige.
func designacionesDelCorpus(t *testing.T) []string {
	t.Helper()
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatalf("cargar el corpus: %v", err)
	}
	vistas := map[string]bool{}
	for _, p := range ps {
		for _, r := range p.Aplicabilidad.Reglas {
			for _, m := range designacionEnRegla.FindAllStringSubmatch(r.Regla, -1) {
				vistas[m[1]] = true
			}
		}
	}
	out := make([]string, 0, len(vistas))
	for k := range vistas {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestTodoPerfilContestaATodaDesignacionDelCorpus es la puerta.
//
// SE ESTRENO CONTRA EL CORPUS REAL antes que contra ninguna mutacion, y nacio
// ROJA: los tres perfiles se callaban `adherido_a_resolucion_extrajudicial_de_
// conflictos` y el de fabricante de software se callaba ademas
// `mantiene_sistema_de_informacion_crediticia`. Ninguna mutacion habria
// encontrado eso, porque nadie sabia que estaba ahi.
func TestTodoPerfilContestaATodaDesignacionDelCorpus(t *testing.T) {
	designaciones := designacionesDelCorpus(t)
	// El suelo protege del verde por vacio: si la expresion deja de casar, la
	// lista sale vacia y el bucle de abajo no recorre nada.
	if len(designaciones) < 8 {
		t.Fatalf("del corpus salen %d designaciones (%v) y hoy hay al menos 8. O el corpus ha "+
			"adelgazado, o esta puerta ha dejado de ver por donde se piden",
			len(designaciones), designaciones)
	}
	ps, err := cargarPerfiles()
	if err != nil {
		t.Fatalf("cargar los perfiles: %v", err)
	}
	if len(ps) < 3 {
		t.Fatalf("hay %d perfiles empotrados y hoy son al menos 3", len(ps))
	}

	for _, p := range ps {
		afirmadas := designacionesQueAfirma(p)
		nombradas := strings.Join(p.NoSupone, "\n")
		for _, d := range designaciones {
			if afirmadas[d] {
				continue
			}
			if strings.Contains(nombradas, d) {
				continue
			}
			t.Errorf("el perfil %q no dice NADA de la designacion %q.\n"+
				"  Los relojes que cuelgan de ella no se ven desde ningun arranque de este "+
				"perfil, y quien lo use no tiene forma de enterarse de que existen.\n"+
				"  Arreglo: o el perfil la afirma (con su porque y su confianza, si el sector "+
				"de verdad la implica), o la nombra en `no_supone` con el articulo que la "+
				"exige. Callarla es indistinguible de las otras dos.", p.ID, d)
		}
	}
}

// designacionesQueAfirma da las designaciones que un perfil pone como hecho, en
// su lista principal o en cualquiera de sus bandas.
//
// LAS BANDAS CUENTAN, y hay que decirlo: una designacion afirmada solo a partir
// de cincuenta empleados SE AFIRMA, aunque no en todos los arranques. Ignorarlas
// haria que esta puerta pidiera un `no_supone` de algo que el perfil si supone,
// y entonces el fichero diria las dos cosas.
func designacionesQueAfirma(p perfil) map[string]bool {
	out := map[string]bool{}
	anotar := func(hs []hechoSupuesto) {
		for _, h := range hs {
			if h.Pred != "designado" || len(h.Args) < 2 {
				continue
			}
			out[h.Args[len(h.Args)-1]] = true
		}
	}
	anotar(p.Hechos)
	for _, b := range p.Bandas {
		anotar(b.Hechos)
	}
	return out
}

// CONTROL NEGATIVO DEL LECTOR DE DESIGNACIONES.
//
// Todo cuelga de la expresion, y tiene dos formas de mentir en silencio: no
// casar con nada (y entonces la puerta de arriba pasaria a recorrer el vacio y
// seguiria verde, que es el fallo que se lee como bueno) o casar con cualquier
// cosa. Se recorren las dos ramas con texto escrito aqui, que es lo unico que
// permite comprobar la rama negativa.
func TestElLectorDeDesignacionesDistingueLasDosRespuestas(t *testing.T) {
	// RAMA POSITIVA, con la forma exacta que usa el corpus.
	const buena = `aplica("x.y", E) :- trata_datos_personales(E), designado(E, "una_cosa")`
	m := designacionEnRegla.FindAllStringSubmatch(buena, -1)
	if len(m) != 1 || m[1-1][1] != "una_cosa" {
		t.Fatalf("el lector no caza una designacion escrita como la escribe el corpus: %v", m)
	}
	// RAMA NEGATIVA: un predicado que NO es designado no se cuenta.
	const ajena = `aplica("x.y", E) :- otra_cosa(E, "una_cosa")`
	if m := designacionEnRegla.FindAllStringSubmatch(ajena, -1); len(m) != 0 {
		t.Errorf("el lector caza %v en una regla que no nombra designado: esta diciendo que si "+
			"a todo, y entonces la puerta pediria en los perfiles designaciones que no existen", m)
	}
}

// LOS PERFILES SON DATOS Y NO CODIGO, y esta es la mitad que lo comprueba desde
// el otro lado: los ficheros que se leen son los EMPOTRADOS, no unos sueltos en
// disco. Sin esto, un perfil podria vivir en el directorio de trabajo y
// `plazum calendario --pais=ES` funcionaria en la maquina del que lo escribio y
// en ninguna otra.
func TestLosPerfilesViajanDentroDelBinario(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if _, err := os.Stat(filepath.Join(dir, "perfiles")); err == nil {
		t.Fatal("el directorio de prueba no esta vacio, asi que no prueba nada")
	}
	ps, err := cargarPerfiles()
	if err != nil {
		t.Fatalf("los perfiles no cargan fuera del arbol del repositorio: %v", err)
	}
	if len(ps) < 3 {
		t.Errorf("desde un directorio vacio solo cargan %d perfiles", len(ps))
	}
}
