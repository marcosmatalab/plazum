package comprobaciones

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// LAS COMPROBACIONES DEL REPOSITORIO CORREN DESDE SU RAIZ.
//
// Hasta el 23-09-2026 estos ficheros vivian en la raiz del repositorio, como
// paquete `plazum` sin una linea de produccion, y eran 67: en la pagina de
// GitHub empujaban el README por debajo de una lista de tests. Se mudaron a
// comprobaciones/ enteros y sin tocar una ruta, porque todos leen el arbol con
// rutas relativas a la raiz ("README.md", "paquetes", ".github/workflows") y
// reescribir cada una era cambiar decenas de sitios para no ganar nada.
//
// Lo que se hace en su lugar es UNA cosa: antes de correr ningun test, el
// proceso se muda a la raiz, que es el primer directorio hacia arriba con un
// go.mod. `go test` arranca cada paquete en su propio directorio, asi que sin
// esto todas las rutas relativas apuntarian dentro de comprobaciones/.
//
// LO VIGILA: TestLaRaizEsElPrimerDirectorioConGoModHaciaArriba
func TestMain(m *testing.M) {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "no se sabe desde donde corre la suite: %v\n", err)
		os.Exit(2)
	}
	raiz, err := raizDelRepositorio(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}
	if err := os.Chdir(raiz); err != nil {
		fmt.Fprintf(os.Stderr, "no se puede entrar en la raiz %s: %v\n", raiz, err)
		os.Exit(2)
	}
	os.Exit(m.Run())
}

// dirDeLasComprobaciones es donde viven estos ficheros, relativo a la raiz.
const dirDeLasComprobaciones = "comprobaciones"

// errSinGoMod es la ausencia de raiz. No se toma el directorio de partida como
// raiz por defecto: con la raiz equivocada, las comprobaciones que buscan un
// fichero y no lo encuentran darian por buena su ausencia.
var errSinGoMod = errors.New("no hay ningun go.mod subiendo desde el directorio de la suite")

// raizDelRepositorio sube desde dir hasta el primer directorio que contiene un
// go.mod y lo devuelve.
func raizDelRepositorio(dir string) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !info.IsDir() {
			return dir, nil
		}
		padre := filepath.Dir(dir)
		if padre == dir {
			return "", fmt.Errorf("%w (desde %s)", errSinGoMod, dir)
		}
		dir = padre
	}
}

// LA RAIZ NO VUELVE A LLENARSE DE FICHEROS GO. Es la consecuencia de la
// mudanza, y sin puerta el siguiente test que alguien escriba en la raiz,
// donde "siempre estuvieron", la deshace en silencio.
func TestLaRaizDelRepositorioNoTieneFicherosGo(t *testing.T) {
	entradas, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var sobran []string
	for _, e := range entradas {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".go" {
			sobran = append(sobran, e.Name())
		}
	}
	if len(sobran) > 0 {
		t.Errorf("la raiz tiene %d ficheros Go (%v).\n"+
			"  Las comprobaciones del repositorio viven en %s/ desde el 23-09-2026 "+
			"(docs/decisiones.md, D-28): en la raiz empujaban el README fuera de la "+
			"pagina del repositorio. Arreglo: mover el fichero a %s/ con package %s.",
			len(sobran), sobran, dirDeLasComprobaciones, dirDeLasComprobaciones,
			dirDeLasComprobaciones)
	}
}

func TestLaRaizEsElPrimerDirectorioConGoModHaciaArriba(t *testing.T) {
	base := t.TempDir()
	hondo := filepath.Join(base, "a", "b", "c")
	if err := os.MkdirAll(hondo, 0o750); err != nil {
		t.Fatal(err)
	}

	// SIN go.mod: tiene que ser error, y no el directorio de partida. Se parte
	// de la raiz del volumen, que no tiene padre, para recorrer la rama de error
	// sin depender de lo que haya por encima de TempDir en esta maquina.
	volumen := filepath.VolumeName(base) + string(filepath.Separator)
	if _, err := os.Stat(filepath.Join(volumen, "go.mod")); err == nil {
		t.Fatalf("hay un go.mod en %s: la rama de error no se puede recorrer en esta maquina", volumen)
	}
	if got, err := raizDelRepositorio(volumen); !errors.Is(err, errSinGoMod) {
		t.Errorf("desde %s, sin go.mod, tiene que salir errSinGoMod y sale %q, %v", volumen, got, err)
	}

	// CON go.mod en la base y otro mas abajo: manda el primero hacia arriba.
	if err := os.WriteFile(filepath.Join(base, "go.mod"), []byte("module x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	medio := filepath.Join(base, "a")
	if err := os.WriteFile(filepath.Join(medio, "go.mod"), []byte("module y\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := raizDelRepositorio(hondo)
	if err != nil {
		t.Fatal(err)
	}
	if got != medio {
		t.Errorf("desde %s la raiz tiene que ser %s, el primer go.mod hacia arriba, y sale %s",
			hondo, medio, got)
	}

	// UN DIRECTORIO LLAMADO go.mod no es un go.mod.
	falso := filepath.Join(t.TempDir(), "p")
	if err := os.MkdirAll(filepath.Join(falso, "go.mod"), 0o750); err != nil {
		t.Fatal(err)
	}
	if got, err := raizDelRepositorio(falso); err == nil && got == falso {
		t.Errorf("un directorio llamado go.mod ha contado como raiz en %s", falso)
	}

	// Y EL PROCESO YA ESTA EN LA RAIZ DE ESTE REPOSITORIO: la mudanza de TestMain
	// se hizo, que es lo que el resto de la suite da por hecho.
	if _, err := os.Stat("go.mod"); err != nil {
		t.Errorf("la suite no corre desde la raiz del repositorio: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dirDeLasComprobaciones, "raiz_del_repositorio_test.go")); err != nil {
		t.Errorf("el directorio de trabajo tiene go.mod pero no es este repositorio: %v", err)
	}
}
