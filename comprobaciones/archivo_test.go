package comprobaciones

import (
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/internal/modulo"
)

// MinimoDeReferenciasAlArchivo es el suelo que impide el verde por vacio.
//
// El 22-09-2026, al archivar, eran 48 referencias en 34 ficheros. Si manana son
// cero, esta puerta no ha mejorado: ha dejado de mirar.
const MinimoDeReferenciasAlArchivo = 30

// baseDelArchivo compone el prefijo de una referencia archivada.
//
// La ruta del repositorio NO se escribe aqui: se lee de `go.mod`, que es donde
// vive. Escribirla seria una segunda copia, y una segunda copia no se rompe
// cuando la primera cambia: se queda vieja y sigue dando verde. Lo prohibe
// `TestNadieCableaLaRutaDelModulo`, que es quien caza este fichero si alguien lo
// intenta.
func baseDelArchivo(t *testing.T) string {
	t.Helper()
	mod, err := modulo.Ruta()
	if err != nil {
		t.Fatal(err)
	}
	return "https://" + mod + "/blob/"
}

// reReferenciaAlArchivo casa una referencia a un fichero archivado: el commit
// fijado y la ruta que tenia dentro de ese commit.
func reReferenciaAlArchivo(t *testing.T) *regexp.Regexp {
	t.Helper()
	return regexp.MustCompile(regexp.QuoteMeta(baseDelArchivo(t)) +
		`([0-9a-f]{40})/((?:[A-Za-z0-9_-]+/)*[A-Za-z0-9_-]+\.[A-Za-z0-9]+)`)
}

// TODA REFERENCIA AL ARCHIVO APUNTA AL MISMO COMMIT, Y ESA RUTA EXISTE AHI.
//
// # Que se archivo y por que
//
// El 22-09-2026 salieron del arbol tres piezas de bitacora: los 19 cuadernos de
// `docs/hallazgos/`, el relato de las familias de `docs/bitacora/` y el censo de
// relojes. Eran **13.221 de las 24.266 lineas de markdown del repositorio**, o
// sea que `docs/` era mayoritariamente diario y quien lo abria no podia
// distinguir la documentacion del relato.
//
// No se borraron: viven en un commit fijado, y cada referencia del arbol apunta
// a ese commit por su SHA completo. Un SHA no se mueve, asi que el enlace no
// puede pudrirse por arriba.
//
// # Las dos formas en que esto SI se pudre, y son las que se vigilan
//
//  1. **Dos anclas distintas.** Alguien archiva algo nuevo, fija OTRO commit, y
//     a partir de ahi el arbol cita dos puntos del pasado sin decirlo. Entonces
//     «el archivo» deja de ser un sitio y pasa a ser dos, y nadie sabe cual
//     manda. Se exige UNA sola ancla, exacta.
//  2. **Una ruta que no existia ahi.** Un enlace escrito de memoria, o copiado
//     de otro con la ruta cambiada, tiene la FORMA de lo verificable, y esa
//     forma es justo lo que hace que nadie vaya a verificarlo. Se comprueba
//     contra el objeto de git, que esta en cualquier clon completo.
//
// # Su limite, dicho
//
// La comprobacion de existencia necesita la historia. En un clon superficial
// (`fetch-depth: 1`) el objeto no esta, y entonces esa mitad se SALTA y lo dice:
// «no se pudo comprobar» no es «esta bien». La mitad de la forma y la del ancla
// unica corren siempre, y el lazo local corre sobre un clon completo en cada
// empujon, que es donde de verdad se escriben estos enlaces.
func TestTodaReferenciaAlArchivoApuntaAlMismoCommitYExiste(t *testing.T) {
	re := reReferenciaAlArchivo(t)
	anclas := map[string][]string{} // sha -> ficheros que lo citan
	rutas := map[string][]string{}  // "sha:ruta" -> ficheros
	total := 0

	for _, f := range ficherosVersionados(t, "*") {
		if !tieneExtensionDeTexto(f) {
			continue
		}
		b, err := os.ReadFile(f) // #nosec G304 -- rutas del propio arbol de git
		if err != nil {
			continue
		}
		for _, m := range re.FindAllStringSubmatch(string(b), -1) {
			// El godoc de esta funcion contiene el patron, no una referencia.
			if strings.HasSuffix(f, "archivo_test.go") {
				continue
			}
			total++
			anclas[m[1]] = append(anclas[m[1]], f)
			rutas[m[1]+":"+m[2]] = append(rutas[m[1]+":"+m[2]], f)
		}
	}

	if total < MinimoDeReferenciasAlArchivo {
		t.Fatalf(`he encontrado %d referencias al archivo y el suelo son %d.

  Si el archivo ha dejado de citarse, esta puerta no ha mejorado: ha dejado de
  mirar. Y si el patron ha dejado de casar, esto esta midiendo el vacio.`,
			total, MinimoDeReferenciasAlArchivo)
	}

	// ---- 1. UNA sola ancla -------------------------------------------------
	if len(anclas) != 1 {
		var detalle []string
		for sha, fs := range anclas {
			detalle = append(detalle, sha[:12]+" citado por "+strings.Join(primeros(fs, 3), ", "))
		}
		sort.Strings(detalle)
		t.Errorf(`el arbol cita %d commits distintos del archivo y tiene que citar UNO.

  Con dos anclas, «el archivo» deja de ser un sitio y pasa a ser dos, y nadie
  sabe cual manda. Si hace falta archivar algo nuevo, se archiva y se reescriben
  TODAS las referencias al commit nuevo, en el mismo commit.

  Los que hay:
    %s`, len(anclas), strings.Join(detalle, "\n    "))
		return
	}

	var ancla string
	for sha := range anclas {
		ancla = sha
	}
	t.Logf("%d referencias al archivo, todas al commit %s", total, ancla[:12])

	// ---- 2. Cada ruta existe en ese commit ---------------------------------
	if err := exec.Command("git", "cat-file", "-e", ancla+"^{commit}").Run(); err != nil {
		t.Skipf("SALTADO, no comprobado: el commit %s no esta en este clon (%v). "+
			"En un clon superficial no hay historia que mirar. NO es verde.", ancla[:12], err)
	}
	var claves []string
	for k := range rutas {
		claves = append(claves, k)
	}
	sort.Strings(claves)
	for _, k := range claves {
		sha, ruta, _ := strings.Cut(k, ":")
		if err := exec.Command("git", "cat-file", "-e", sha+":"+ruta).Run(); err != nil {
			t.Errorf(`%s no existe en el commit %s, y lo citan: %s

  Un enlace al archivo tiene la FORMA de lo verificable, y esa forma es lo que
  hace que nadie vaya a comprobarlo. Comprueba la ruta con:
    git cat-file -e %s:%s`,
				ruta, sha[:12], strings.Join(primeros(rutas[k], 3), ", "), sha[:12], ruta)
		}
	}
	t.Logf("%d rutas distintas, todas presentes en el commit del archivo", len(claves))
}

// EL CONTROL NEGATIVO del lector de referencias.
//
// Sus dos fallos probables son opuestos y los dos dan un numero plausible: una
// expresion que no case nada hace saltar el suelo (eso se ve), pero una que case
// CUALQUIER enlace a github daria por referencia al archivo media documentacion
// del repositorio, y entonces «una sola ancla» fallaria por motivos que no son.
func TestElLectorDeReferenciasAlArchivoAcusaYSeCalla(t *testing.T) {
	const sha = "a1ef407850fba1887977cc255e7eddfd9173d962"
	re := reReferenciaAlArchivo(t)
	base := baseDelArchivo(t)
	repo := strings.TrimSuffix(base, "/blob/")
	for _, c := range []struct {
		nombre string
		texto  string
		sha    string
		ruta   string
	}{
		{"una referencia normal",
			"ver " + base + sha + "/docs/hallazgos/ia.md",
			sha, "docs/hallazgos/ia.md"},
		{"dentro de un enlace markdown",
			"[el cuaderno](" + base + sha + "/docs/censo-relojes.md)",
			sha, "docs/censo-relojes.md"},
	} {
		m := re.FindStringSubmatch(c.texto)
		if m == nil {
			t.Errorf("%s: el lector no ve la referencia en %q", c.nombre, c.texto)
			continue
		}
		if m[1] != c.sha || m[2] != c.ruta {
			t.Errorf("%s: el lector saca sha=%q ruta=%q y esperaba sha=%q ruta=%q",
				c.nombre, m[1], m[2], c.sha, c.ruta)
		}
	}
	// Y SE CALLA con lo que NO es una referencia al archivo.
	for _, c := range []struct{ nombre, texto string }{
		{"la portada de un repositorio", repo},
		{"una release", repo + "/releases/latest"},
		{"una rama en vez de un commit", base + "main/docs/ia.md"},
		{"un sha corto", base + "a1ef407/docs/ia.md"},
	} {
		if re.MatchString(c.texto) {
			t.Errorf("%s: el lector da por referencia al archivo %q, y no lo es. "+
				"Un lector demasiado ancho hace fallar «una sola ancla» por motivos que no son",
				c.nombre, c.texto)
		}
	}
}

func primeros(xs []string, n int) []string {
	sort.Strings(xs)
	if len(xs) <= n {
		return xs
	}
	return append(append([]string{}, xs[:n]...), "...")
}

func tieneExtensionDeTexto(ruta string) bool {
	for _, e := range []string{".md", ".go", ".yml", ".yaml", ".sh", ".json", ".txt", ".html", ".conf"} {
		if strings.HasSuffix(ruta, e) {
			return true
		}
	}
	return false
}
