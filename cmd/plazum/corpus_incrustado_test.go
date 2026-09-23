package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/marcosmatalab/plazum/paquetes"
)

// EL CORPUS INCRUSTADO ES EXACTAMENTE EL QUE PUBLICA LA RELEASE. La directiva de
// embed de paquetes/ nombra los patrones a mano, y un fichero de corpus que no casara con
// ninguno se quedaria fuera del binario sin que nada fallara. La huella es la
// del tarball de la release, asi que si coinciden, coinciden fichero a fichero.
func TestElCorpusIncrustadoEsElArbolPublicado(t *testing.T) {
	enDisco, err := HuellaDeArbol(filepath.Join("..", "..", "paquetes"))
	if err != nil {
		t.Fatal(err)
	}
	dentro, err := huellaDeFS(paquetes.Ficheros)
	if err != nil {
		t.Fatal(err)
	}
	if dentro != enDisco {
		t.Errorf("el corpus que viaja dentro del binario no es el de paquetes/.\n"+
			"  dentro  %s\n  disco   %s\n"+
			"  Arreglo: mira la directiva go:embed de paquetes/incrustado.go; un fichero\n"+
			"  nuevo de corpus que no case con sus patrones se queda fuera.", dentro, enDisco)
	}
}

func TestUnaOrdenSinCorpusEnDiscoUsaElIncrustado(t *testing.T) {
	cache := t.TempDir()
	antes := directorioDeCache
	directorioDeCache = func() string { return cache }
	t.Cleanup(func() { directorioDeCache = antes })

	// UN DIRECTORIO VACIO, que es donde se queda quien instala el binario de la
	// release o hace `go install`. Hasta el 23-09-2026 esto salia con
	// "el corpus de paquetes no carga: open paquetes".
	t.Chdir(t.TempDir())
	var salida, errores bytes.Buffer
	rc := cmdCalendario([]string{"--pais=ES", "--sector=servicios-digitales", "--empleados=200"},
		&salida, &errores)
	if rc != 0 {
		t.Fatalf("calendario en un directorio vacio sale con %d:\n%s", rc, errores.String())
	}
	if !strings.Contains(errores.String(), "viaja dentro de este binario") {
		t.Errorf("no dice de donde sale el corpus que ha usado:\n%s", errores.String())
	}
	if !strings.Contains(salida.String(), "PROXIMOS DOCE MESES") {
		t.Errorf("no ha salido el calendario:\n%s", salida.String())
	}

	// --corpus TECLEADO manda, aunque no exista: no se le cambia por otro.
	salida.Reset()
	errores.Reset()
	rc = cmdCalendario([]string{"--corpus", "no-existe", "--pais=ES",
		"--sector=servicios-digitales", "--empleados=200"}, &salida, &errores)
	if rc == 0 || strings.Contains(errores.String(), "viaja dentro de este binario") {
		t.Errorf("con --corpus tecleado a un directorio que no existe ha usado el incrustado "+
			"(rc %d):\n%s", rc, errores.String())
	}

	// Y UN paquetes/ EN DISCO MANDA sobre el incrustado: es como se actualiza.
	if got := elegirCorpus(false, "no-existe"); got != corpusIncrustado {
		t.Errorf("sin nada en disco elige %d y tiene que elegir el incrustado", got)
	}
	if err := os.Mkdir("paquetes", 0o750); err != nil {
		t.Fatal(err)
	}
	if got := elegirCorpus(false, "paquetes"); got != corpusEnDisco {
		t.Errorf("con paquetes/ en disco elige %d y tiene que elegir el del disco", got)
	}
	if got := elegirCorpus(true, "no-existe"); got != corpusTecleado {
		t.Errorf("con --corpus tecleado elige %d y tiene que elegir el tecleado", got)
	}
}

func TestLaCopiaDelCorpusIncrustadoSeRehaceSiNoCuadra(t *testing.T) {
	fsys := fstest.MapFS{
		"a/paquete.json":     {Data: []byte(`{"urn":"x"}`)},
		"a/pruebas/uno.json": {Data: []byte(`{}`)},
		"CORPUS.md":          {Data: []byte("# corpus\n")},
	}
	base := t.TempDir()
	ruta, huella, err := copiarCorpusADisco(fsys, base)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(ruta) != huella {
		t.Errorf("la copia vive en %s y tiene que vivir en un directorio con su huella %s", ruta, huella)
	}

	// Alguien toca la copia a mano: la siguiente vez se rehace, no se usa.
	tocado := filepath.Join(ruta, "a", "paquete.json")
	if err := os.WriteFile(tocado, []byte(`{"urn":"otro"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if h, _ := HuellaDeArbol(ruta); h == huella {
		t.Fatal("la mutacion no ha cambiado la huella, asi que esta comprobacion no mide nada")
	}
	ruta2, _, err := copiarCorpusADisco(fsys, base)
	if err != nil {
		t.Fatal(err)
	}
	if h, err := HuellaDeArbol(ruta2); err != nil || h != huella {
		t.Errorf("la copia tocada no se ha rehecho: huella %s y tiene que ser %s (%v)", h, huella, err)
	}
}
