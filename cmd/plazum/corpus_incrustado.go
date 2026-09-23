package main

// El corpus que viaja dentro del binario, y cuando se usa.
//
// Hasta el 23-09-2026 un plazum instalado como dice el README (binario de la
// release o `go install ...@latest`) no traia corpus: en un directorio vacio,
// `plazum calendario` salia con "el corpus de paquetes no carga: open paquetes".
// Ahora el corpus publicado va dentro del binario (paquetes.Ficheros) y es el
// valor por defecto. El porque y lo que cuesta, en docs/decisiones.md D-30.
//
// QUE CORPUS USA UNA ORDEN, por este orden:
//
//	--corpus <dir> tecleado    ese, y si no carga es un error: quien lo escribe
//	                           sabe lo que quiere y no se le cambia por otro.
//	paquetes/ donde se ejecuta el del disco: el del repositorio, el de la imagen
//	                           o uno instalado con `plazum corpus --instalar`,
//	                           que es como se actualiza sin cambiar de programa.
//	ninguno de los dos         el que viaja dentro del binario.
//
// POR QUE SE ESCRIBE EN DISCO Y NO SE LEE DE MEMORIA. corpus.Cargar lee un
// directorio, y el nucleo no cambia para esto. Y un corpus en disco se puede
// abrir: quien quiera saber de donde sale una fecha abre el JSON con cualquier
// editor, que es una de las razones por las que el corpus nunca fue un blob. Se
// escribe en la cache de usuario, en un directorio con el nombre de su huella, y
// se reutiliza solo si su huella cuadra: una copia tocada a mano se rehace.
//
// LO VIGILAN: TestElCorpusIncrustadoEsElArbolPublicado,
// TestUnaOrdenSinCorpusEnDiscoUsaElIncrustado y
// TestLaCopiaDelCorpusIncrustadoSeRehaceSiNoCuadra

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/marcosmatalab/plazum/paquetes"
)

// origenCorpus es de donde sale el corpus de una orden.
type origenCorpus int

const (
	corpusTecleado origenCorpus = iota
	corpusEnDisco
	corpusIncrustado
)

// elegirCorpus decide de donde sale el corpus sin escribir nada: tecleado es si
// el operador paso --corpus, dir es lo que vale la opcion.
func elegirCorpus(tecleado bool, dir string) origenCorpus {
	if tecleado {
		return corpusTecleado
	}
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return corpusEnDisco
	}
	return corpusIncrustado
}

// opcionTecleada dice si el operador paso la opcion, y no solo si vale lo que
// vale por defecto: `--corpus paquetes` tecleado y no tecleado se parecen, y
// solo el primero es una orden de usar ese directorio y ningun otro.
func opcionTecleada(fs *flag.FlagSet, nombre string) bool {
	vista := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == nombre {
			vista = true
		}
	})
	return vista
}

// corpusResuelto aplica elegirCorpus a una orden ya parseada: si toca el
// incrustado, lo deja en disco y cambia *dir por su ruta. Devuelve false, con
// el motivo escrito en errores, si no ha podido.
func corpusResuelto(fs *flag.FlagSet, dir *string, errores io.Writer) bool {
	if elegirCorpus(opcionTecleada(fs, "corpus"), *dir) != corpusIncrustado {
		return true
	}
	ruta, huella, err := corpusIncrustadoEnDisco(directorioDeCache())
	if err != nil {
		fmt.Fprintf(errores, "error: no hay corpus en %q y el que viaja dentro de este binario no se "+
			"ha podido dejar en disco: %v.\n"+
			"  Arreglo: pasa --corpus con un directorio de paquetes, o instala uno con\n"+
			"  `plazum corpus --instalar plazum-corpus.tar.gz`.\n", *dir, err)
		return false
	}
	fmt.Fprintf(errores, "corpus: el que viaja dentro de este binario (huella %s), en %s\n",
		huella[:16], ruta)
	*dir = ruta
	return true
}

// directorioDeCache es donde se deja el corpus incrustado. Es una variable para
// que los tests no escriban en la cache de quien los corre.
var directorioDeCache = directorioDeCacheDelSistema

// directorioDeCacheDelSistema es la cache de usuario si la hay; si no (un
// usuario de servicio sin HOME), el temporal del sistema.
func directorioDeCacheDelSistema() string {
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "plazum", "corpus")
}

// huellaDeFS resume un corpus que no esta en disco con el MISMO calculo que
// HuellaDeArbol (resumirHuella) y el mismo filtro (entraEnElCorpus).
func huellaDeFS(fsys fs.FS) (string, error) {
	var entradas []entradaHuella
	err := fs.WalkDir(fsys, ".", func(ruta string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !entraEnElCorpus(ruta) {
			return nil
		}
		b, err := fs.ReadFile(fsys, ruta)
		if err != nil {
			return err
		}
		suma := sha256.Sum256(b)
		entradas = append(entradas, entradaHuella{rel: ruta, sum: hex.EncodeToString(suma[:])})
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(entradas) == 0 {
		return "", errors.New("el corpus incrustado esta vacio")
	}
	return resumirHuella(entradas), nil
}

// corpusIncrustadoEnDisco deja paquetes.Ficheros en base/<huella> y devuelve la
// ruta y la huella. Si ya esta y su huella cuadra, no escribe nada.
func corpusIncrustadoEnDisco(base string) (ruta, huella string, err error) {
	return copiarCorpusADisco(paquetes.Ficheros, base)
}

func copiarCorpusADisco(fsys fs.FS, base string) (ruta, huella string, err error) {
	huella, err = huellaDeFS(fsys)
	if err != nil {
		return "", "", err
	}
	ruta = filepath.Join(base, huella)
	if h, err := HuellaDeArbol(ruta); err == nil && h == huella {
		return ruta, huella, nil
	}

	if err := os.MkdirAll(base, 0o750); err != nil {
		return "", "", err
	}
	temporal, err := os.MkdirTemp(base, "escribiendo-")
	if err != nil {
		return "", "", err
	}
	defer func() { _ = os.RemoveAll(temporal) }()

	err = fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		destino := filepath.Join(temporal, filepath.FromSlash(p))
		if d.IsDir() {
			return os.MkdirAll(destino, 0o750)
		}
		b, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		return os.WriteFile(destino, b, 0o600)
	})
	if err != nil {
		return "", "", err
	}
	if h, err := HuellaDeArbol(temporal); err != nil || h != huella {
		return "", "", fmt.Errorf("la copia escrita en %s no da la huella del corpus incrustado "+
			"(%s frente a %s, %v)", temporal, h, huella, err)
	}

	// Una copia vieja o tocada se sustituye entera. Si otro proceso ha dejado
	// la buena mientras tanto, el rename falla y se usa la suya, que se
	// comprueba igual.
	_ = os.RemoveAll(ruta)
	if err := os.Rename(temporal, ruta); err != nil {
		if h, e := HuellaDeArbol(ruta); e == nil && h == huella {
			return ruta, huella, nil
		}
		return "", "", err
	}
	return ruta, huella, nil
}
