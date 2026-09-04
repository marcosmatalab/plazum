package main

// `plazum corpus`: el corpus real viaja en la release, y llega comprobado.
//
// EL P0 QUE ESTE FICHERO EXISTE PARA CERRAR. Hasta hoy la prueba de la maquina
// limpia pasaba con codigo 0 y llegaba al calendario con el corpus de
// DEMOSTRACION: un paquete y tres relojes. Los treinta marcos vivian en el
// repositorio y no salian de el. Publicar asi significa que el primer plazum
// publico del mundo es una demo vacia, y que la primera impresion de todo el que
// lo pruebe es «esto no trae nada».
//
// LA DECISION, Y POR QUE NO ES go:embed.
//
// Habia dos formas de que el corpus viajara: dentro del binario (go:embed) o al
// lado, como activo firmado de la release. Se ha elegido la segunda, y las
// razones son de producto antes que de tamano:
//
//  1. EL CORPUS CAMBIA EN EL CALENDARIO DEL BOE, NO EN EL DEL SOFTWARE. Un
//     omnibus que mueve una vigencia obliga a publicar corpus nuevo el mismo
//     dia. Con el corpus dentro del binario, eso es recompilar tres sistemas
//     por dos arquitecturas, volver a firmar seis artefactos y pedirle a todo
//     el mundo que se baje otra vez doce megas para cambiar una fecha. Con el
//     corpus al lado es un fichero de trescientos kilobytes.
//  2. `plazum update` YA EXISTE Y YA SABE VOLVER ATRAS. Un corpus empotrado no
//     se puede actualizar sin recompilar, asi que el actualizador se quedaria
//     sin la mitad de su trabajo.
//  3. UN PRODUCTO DE CUMPLIMIENTO TIENE QUE DEJAR MIRAR SU CORPUS. Dos megas de
//     JSON dentro de un ejecutable no los abre un abogado. Un .tar.gz si.
//
// Y HAY UNA CUARTA RAZON QUE NO ES DE MERITO Y SE DICE IGUAL: go:embed no puede
// salir del directorio de su paquete, asi que empotrar `paquetes/` exigiria un
// fichero Go dentro de `paquetes/` o en la raiz del modulo, y ninguno de los dos
// esta en la columna de esta rebanada. La decision se sostiene sola por las tres
// de arriba, pero quien la revise merece saber que la particion tambien la
// empujaba.
//
// LO QUE LA SEGUNDA FORMA OBLIGA A CONSTRUIR, Y ES LO QUE HAY AQUI ABAJO. Un
// corpus que viaja al lado del binario es un corpus del que hay que poder decir
// DE DONDE VIENE. Si no, se ha cambiado una demo vacia por algo peor: un corpus
// en el que se confia sin razon. Contra eso hay un ancla.
//
// EL ANCLA. Al construir un binario para una release se le mete dentro, con
// -ldflags -X, la huella del arbol de corpus que se publica en esa misma
// release. El binario lo firma cosign, asi que la huella viaja bajo esa firma.
// Instalar un corpus compara su huella contra ese ancla ANTES de dejarlo caer en
// su sitio.
//
// LAS TRES RESPUESTAS, Y LA NADA NO ES NINGUNA DE LAS DOS BUENAS (invariante 8).
// El valor cero de `anclaCorpus` es la cadena vacia, que es lo que sale por
// olvidarse de la bandera de compilacion. Si la cadena vacia significara «no
// compruebes», un binario construido a mano instalaria cualquier cosa que se
// llamara corpus, y el fallo iria en la direccion permisiva, que es la que nadie
// mira. Aqui la cadena vacia significa NO PUEDO COMPROBARLO, y no puedo
// comprobarlo no autoriza a instalar: se para y se dice como aportar un ancla.
//
//	ancla vacia            -> «este binario no sabe cual es su corpus». Se para.
//	ancla y cuadra         -> es el corpus que se publico con este binario.
//	ancla y no cuadra      -> se para, y se dicen las DOS huellas.
//
// Y COMO SE ACTUALIZA SIN RECOMPILAR, que es el requisito que un ancla fija
// podria haberse cargado. El ancla contesta «es este el corpus que salio con
// este binario», no «esta autorizado este corpus». Un corpus mas nuevo que el
// binario NUNCA va a cuadrar con el ancla, y eso es correcto y no puede ser un
// muro: para ese caso esta --huella-esperada, donde el operador aporta el ancla
// que ha leido en la pagina de la release. Sigue siendo una comprobacion
// mecanica; lo unico que cambia es de donde sale el ancla.
//
// LO QUE NO HAY, A PROPOSITO: una bandera de «instala sin comprobar». Un si/no
// se teclea por costumbre y termina en todos los guiones de todo el mundo;
// --huella-esperada obliga a ir a mirar la fuente autoritativa, que es
// exactamente el paso que se quiere forzar.

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// anclaCorpus es la huella del arbol de corpus que se publica junto a este
// binario. La inyecta el workflow de release al compilar:
//
//	go build -ldflags "-X main.anclaCorpus=<64 hex>" ./cmd/plazum
//
// Vacia en cualquier binario construido a mano. Vacia NO significa «adelante»:
// significa que este binario no sabe cual es su corpus, y eso se dice.
var anclaCorpus string

// versionHuella va dentro del manifiesto que se resume, no al lado. Si algun dia
// cambia como se calcula la huella, la huella entera cambia y se nota; con la
// version fuera del resumen, dos algoritmos distintos podrian dar el mismo
// numero para arboles distintos y nadie se enteraria.
const versionHuella = "plazum-corpus-huella-v1"

// Los topes de la extraccion. Un .tar.gz es entrada adversaria hasta que se
// comprueba, y la comprobacion es POSTERIOR a extraer (hay que tener el arbol
// para calcular su huella). Entre medias, lo unico que protege el disco son
// estos numeros.
//
// Estan holgados sobre lo que hay (el corpus real son 303 ficheros y 2 MB) y muy
// por debajo de lo que llena una maquina.
const (
	maxFicherosCorpus = 20000
	maxBytesFichero   = 32 << 20  // 32 MiB por fichero
	maxBytesTotal     = 256 << 20 // 256 MiB descomprimidos
)

var (
	// ErrSinAncla: no hay contra que comprobar. Centinela y no un booleano
	// suelto, porque es la rama que decide si se instala o no.
	ErrSinAncla = errors.New("no hay ancla contra la que comprobar el corpus")
	// ErrHuellaNoCuadra: hay ancla y el corpus no es ese.
	ErrHuellaNoCuadra = errors.New("la huella del corpus no cuadra con el ancla")
)

func cmdCorpus(args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("plazum corpus", flag.ContinueOnError)
	fs.SetOutput(errores)
	huella := fs.String("huella", "", "imprime la huella del arbol de corpus que haya en ese directorio")
	empaquetar := fs.String("empaquetar", "", "empaqueta ese directorio de corpus en un .tar.gz")
	instalar := fs.String("instalar", "", "comprueba e instala un .tar.gz de corpus")
	verificar := fs.String("verificar", "", "compara la huella de ese directorio con el ancla de este binario")
	destino := fs.String("destino", "paquetes", "donde se instala el corpus")
	salidaTar := fs.String("salida", "", "fichero .tar.gz que escribe --empaquetar")
	esperada := fs.String("huella-esperada", "", "la huella que dice la pagina de la release; sustituye al ancla del binario")
	forzar := fs.Bool("forzar", false, "sobrescribe el destino si ya existe")
	fs.Usage = func() {
		fmt.Fprintln(errores, "uso: plazum corpus                              que corpus hay y si se puede confiar en el")
		fmt.Fprintln(errores, "     plazum corpus --instalar F.tar.gz          lo comprueba y lo instala en ./paquetes")
		fmt.Fprintln(errores, "     plazum corpus --verificar DIR              compara ese corpus con el ancla del binario")
		fmt.Fprintln(errores, "     plazum corpus --huella DIR                 imprime su huella y nada mas")
		fmt.Fprintln(errores, "     plazum corpus --empaquetar DIR --salida F  lo empaqueta (lo usa la release)")
		fmt.Fprintln(errores, "")
		fmt.Fprintln(errores, "El corpus real viaja como activo de la release, firmado, y NO dentro del")
		fmt.Fprintln(errores, "binario: cambia cuando cambia el BOE, no cuando cambia el programa.")
		fmt.Fprintln(errores, "Instalarlo comprueba su huella contra la que este binario lleva dentro.")
		fmt.Fprintln(errores, "")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	switch {
	case *huella != "":
		h, err := HuellaDeArbol(*huella)
		if err != nil {
			fmt.Fprintln(errores, "error:", err)
			return 1
		}
		fmt.Fprintln(salida, h)
		return 0
	case *empaquetar != "":
		return cmdEmpaquetar(*empaquetar, *salidaTar, salida, errores)
	case *instalar != "":
		return cmdInstalarCorpus(*instalar, *destino, *esperada, *forzar, salida, errores)
	case *verificar != "":
		return cmdVerificarCorpus(*verificar, *esperada, salida, errores)
	default:
		return cmdParteDelCorpus(*destino, salida)
	}
}

// ---------------------------------------------------------------------------
// La huella de un arbol
// ---------------------------------------------------------------------------

// QUE ENTRA EN LA HUELLA, dicho una sola vez y usado por los dos lados.
//
// La regla se escribe aqui y la usan --huella y --empaquetar, que es la unica
// forma de que el fichero empaquetado y el fichero resumido sean el mismo
// conjunto. Dos listas (una en el empaquetador y otra en el calculador) son dos
// listas que se separan, y la que se queda vieja da una huella que no cuadra con
// nada sin que nadie sepa por que.
//
// Se excluye el codigo Go y solo el codigo Go: `paquetes/` tiene tres ficheros
// .go que son del build del repositorio y no del corpus (el empotrado del demo y
// dos tests de transcripcion). Que R2 anada otro test no puede cambiar la huella
// del corpus, porque no cambia ni una obligacion.
func entraEnElCorpus(rel string) bool {
	return !strings.HasSuffix(rel, ".go")
}

// HuellaDeArbol resume un directorio de corpus en 64 caracteres.
//
// El manifiesto es texto y se ordena por ruta: la huella no depende de en que
// orden devuelva el sistema de ficheros sus entradas, ni de que sistema sea.
// Las rutas van con barra normal SIEMPRE, o el mismo corpus daria dos huellas
// distintas en Windows y en Linux, que es justo el fallo que una huella existe
// para no tener.
//
// Un enlace simbolico es ERROR y no un fichero que se salta. Saltarselo seria
// dejar fuera de la huella contenido que si esta en el arbol: la tercera forma
// de la nada del invariante 8, presente y no interpretable, tomada por ausencia.
func HuellaDeArbol(raiz string) (string, error) {
	info, err := os.Stat(raiz)
	if err != nil {
		return "", fmt.Errorf("no puedo leer el corpus en %q: %w. "+
			"Arreglo: comprueba la ruta, o instala el corpus con "+
			"`plazum corpus --instalar plazum-corpus.tar.gz`", raiz, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%q no es un directorio. La huella se calcula sobre el arbol "+
			"de paquetes, no sobre un fichero suelto", raiz)
	}

	type entrada struct{ rel, sum string }
	var entradas []entrada

	err = filepath.WalkDir(raiz, func(ruta string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("%s no es un fichero normal (es %s). El corpus es datos: "+
				"un enlace, una tuberia o un dispositivo dentro del arbol no se puede "+
				"resumir, y saltarselo dejaria fuera de la huella algo que si esta en el "+
				"arbol. Arreglo: quitalo del corpus", ruta, d.Type())
		}
		rel, err := filepath.Rel(raiz, ruta)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !entraEnElCorpus(rel) {
			return nil
		}
		b, err := os.ReadFile(ruta) // #nosec G304 -- la raiz la da el operador y rel sale de WalkDir sobre ella
		if err != nil {
			return err
		}
		suma := sha256.Sum256(b)
		entradas = append(entradas, entrada{rel: rel, sum: hex.EncodeToString(suma[:])})
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(entradas) == 0 {
		return "", fmt.Errorf("no hay ni un fichero de corpus en %q. "+
			"Un directorio vacio tiene huella, y esa huella no significa «corpus vacio»: "+
			"significa que se esta resumiendo el sitio equivocado. Arreglo: apunta al "+
			"directorio que contiene los paquetes", raiz)
	}

	sort.Slice(entradas, func(i, j int) bool { return entradas[i].rel < entradas[j].rel })

	h := sha256.New()
	// La version va DENTRO del resumen: ver el comentario de versionHuella.
	fmt.Fprintf(h, "%s\n", versionHuella)
	for _, e := range entradas {
		fmt.Fprintf(h, "%s\n%s\n", e.rel, e.sum)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ---------------------------------------------------------------------------
// Empaquetar: lo que corre la release
// ---------------------------------------------------------------------------

func cmdEmpaquetar(dir, destino string, salida, errores io.Writer) int {
	if destino == "" {
		fmt.Fprintln(errores, "error: falta --salida. Empaquetar sin decir donde escribe seria")
		fmt.Fprintln(errores, "  elegir el nombre del activo de la release por el operador.")
		return 2
	}
	h, err := HuellaDeArbol(dir)
	if err != nil {
		fmt.Fprintln(errores, "error:", err)
		return 1
	}
	n, err := empaquetarCorpus(dir, destino)
	if err != nil {
		fmt.Fprintln(errores, "error:", err)
		return 1
	}
	// La huella se imprime SOLA en la ultima linea para que el workflow la lea
	// sin recortar: el resto del parte va por la salida de errores.
	fmt.Fprintf(errores, "empaquetados %d ficheros de %s en %s\n", n, dir, destino)
	fmt.Fprintln(salida, h)
	return 0
}

func empaquetarCorpus(dir, destino string) (int, error) {
	f, err := os.Create(destino) // #nosec G304 -- ruta de salida que da el operador en su maquina
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()

	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)

	n := 0
	err = filepath.WalkDir(dir, func(ruta string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("%s no es un fichero normal: no se empaqueta", ruta)
		}
		rel, err := filepath.Rel(dir, ruta)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if !entraEnElCorpus(rel) {
			return nil
		}
		b, err := os.ReadFile(ruta) // #nosec G304 -- rel sale de WalkDir sobre el dir que dio el operador
		if err != nil {
			return err
		}
		// Cabecera MINIMA y sin metadatos de la maquina que empaqueta: ni
		// fecha, ni usuario, ni permisos heredados. Dos ejecuciones sobre el
		// mismo arbol tienen que dar el mismo .tar.gz, o la suma del activo
		// cambia sin que cambie ni una obligacion y nadie puede reproducirlo.
		if err := tw.WriteHeader(&tar.Header{
			Name:     rel,
			Mode:     0o644,
			Size:     int64(len(b)),
			Typeflag: tar.TypeReg,
		}); err != nil {
			return err
		}
		if _, err := tw.Write(b); err != nil {
			return err
		}
		n++
		return nil
	})
	if err != nil {
		return 0, err
	}
	if err := tw.Close(); err != nil {
		return 0, err
	}
	if err := gz.Close(); err != nil {
		return 0, err
	}
	return n, f.Close()
}

// ---------------------------------------------------------------------------
// Instalar: la unica puerta por la que entra un corpus de fuera
// ---------------------------------------------------------------------------

func cmdInstalarCorpus(tarball, destino, esperada string, forzar bool, salida, errores io.Writer) int {
	ancla, deDonde, err := anclaAUsar(esperada)
	if err != nil {
		fmt.Fprintln(errores, "error:", err)
		explicarFaltaDeAncla(errores)
		return 1
	}

	if _, err := os.Stat(destino); err == nil && !forzar {
		fmt.Fprintf(errores, "error: %s ya existe y no se sobrescribe solo.\n", destino)
		fmt.Fprintf(errores, "  Un corpus instalado encima de otro sin avisar cambia las fechas que\n")
		fmt.Fprintf(errores, "  ensena el calendario sin que nadie lo haya pedido.\n")
		fmt.Fprintf(errores, "  Arreglo: `plazum corpus --instalar %s --destino %s --forzar`,\n", tarball, destino)
		fmt.Fprintf(errores, "  o elige otro --destino.\n")
		return 1
	}

	// SE EXTRAE A UN LADO Y SE COMPRUEBA ANTES DE MOVER. Extraer directamente
	// sobre el destino y comprobar despues dejaria, en el caso malo, un corpus
	// sin verificar puesto donde el calendario lo va a leer, y un error por la
	// salida de errores que nadie asocia con las fechas que vera manana.
	padre := filepath.Dir(destino)
	if padre == "" {
		padre = "."
	}
	if err := os.MkdirAll(padre, 0o750); err != nil {
		fmt.Fprintln(errores, "error:", err)
		return 1
	}
	temporal, err := os.MkdirTemp(padre, ".plazum-corpus-")
	if err != nil {
		fmt.Fprintln(errores, "error:", err)
		return 1
	}
	limpio := false
	defer func() {
		if !limpio {
			_ = os.RemoveAll(temporal)
		}
	}()

	n, err := extraerCorpus(tarball, temporal)
	if err != nil {
		fmt.Fprintln(errores, "error:", err)
		return 1
	}

	h, err := HuellaDeArbol(temporal)
	if err != nil {
		fmt.Fprintln(errores, "error:", err)
		return 1
	}
	if h != ancla {
		fmt.Fprintf(errores, "error: %v.\n", ErrHuellaNoCuadra)
		fmt.Fprintf(errores, "  esperada (%s)  %s\n", deDonde, ancla)
		fmt.Fprintf(errores, "  la del fichero            %s\n", h)
		fmt.Fprintf(errores, "\n  NO se ha instalado nada. Esto significa una de dos cosas y las dos\n")
		fmt.Fprintf(errores, "  importan: o el fichero no es el que publico esta release, o es un\n")
		fmt.Fprintf(errores, "  corpus MAS NUEVO que este binario, que es lo normal cuando cambia una\n")
		fmt.Fprintf(errores, "  norma y no cambia el programa.\n")
		fmt.Fprintf(errores, "  Si es lo segundo: mira la huella en la pagina de la release y pasala\n")
		fmt.Fprintf(errores, "  con --huella-esperada. Si no cuadra tampoco con esa, no lo instales.\n")
		return 1
	}

	if forzar {
		if err := os.RemoveAll(destino); err != nil {
			fmt.Fprintln(errores, "error:", err)
			return 1
		}
	}
	if err := os.Rename(temporal, destino); err != nil {
		fmt.Fprintln(errores, "error:", err)
		return 1
	}
	limpio = true

	paquetes := contarPaquetes(destino)
	fmt.Fprintf(salida, "\n  Corpus instalado en %s: %d paquetes, %d ficheros.\n", destino, paquetes, n)
	fmt.Fprintf(salida, "  huella %s\n", h)
	fmt.Fprintf(salida, "  Comprobada contra %s.\n\n", deDonde)
	fmt.Fprintf(salida, "  Ya puedes ver tus fechas:\n\n")
	fmt.Fprintf(salida, "    plazum calendario --pais=ES --sector=fabricante-software --empleados=200\n\n")
	fmt.Fprintf(salida, "  Eso es un perfil de arranque y cada fila sale marcada como supuesta.\n")
	fmt.Fprintf(salida, "  Para lo tuyo de verdad, contesta la entrevista con `plazum serve` y\n")
	fmt.Fprintf(salida, "  pasa el resultado con --alcance.\n\n")
	return 0
}

// anclaAUsar contesta la pregunta que decide si se instala: contra que se
// compara. El orden importa y es este.
//
//  1. --huella-esperada, si la hay. Es el operador leyendo la pagina de la
//     release, y es lo unico que permite instalar un corpus mas nuevo que el
//     binario sin recompilar nada.
//  2. el ancla que el binario lleva dentro, puesta al construir la release.
//  3. NINGUNA, y entonces no se instala. Ver la cabecera del fichero.
func anclaAUsar(esperada string) (ancla, deDonde string, err error) {
	if esperada != "" {
		e, err := normalizarHuella(esperada)
		if err != nil {
			return "", "", err
		}
		return e, "la huella que has pasado con --huella-esperada", nil
	}
	if anclaCorpus != "" {
		a, err := normalizarHuella(anclaCorpus)
		if err != nil {
			return "", "", fmt.Errorf("el ancla que lleva dentro este binario es ilegible: %w. "+
				"Se construyo con un -ldflags mal puesto y no se puede confiar en el", err)
		}
		return a, "el ancla que este binario lleva dentro", nil
	}
	return "", "", ErrSinAncla
}

// normalizarHuella es la tercera forma de la nada del invariante 8: presente y
// no interpretable. Una huella con la longitud equivocada o con letras que no
// son hexadecimales es un dato QUE HAY y no se entiende, y tomarlo por ausencia
// (o por «pues no cuadra») es inventarse una respuesta: lo primero abriria la
// puerta, lo segundo mandaria a arreglar el corpus cuando lo roto es la huella.
func normalizarHuella(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) != sha256.Size*2 {
		return "", fmt.Errorf("una huella de corpus son %d caracteres hexadecimales y esta tiene %d: %q",
			sha256.Size*2, len(s), s)
	}
	if _, err := hex.DecodeString(s); err != nil {
		return "", fmt.Errorf("la huella %q no es hexadecimal: %w", s, err)
	}
	return s, nil
}

func explicarFaltaDeAncla(w io.Writer) {
	fmt.Fprintln(w, "  Este binario no se construyo con un corpus de referencia, asi que no")
	fmt.Fprintln(w, "  sabe cual deberia ser. NO instalar es la respuesta correcta: un corpus")
	fmt.Fprintln(w, "  que entra sin comprobar es peor que no tener corpus, porque son fechas")
	fmt.Fprintln(w, "  legales en las que alguien va a confiar.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  Dos arreglos, y los dos acaban en una comprobacion de verdad:")
	fmt.Fprintln(w, "    - usa el binario de una release publicada: lleva su ancla dentro.")
	fmt.Fprintln(w, "    - o pasa --huella-esperada con la huella que dice la pagina de la")
	fmt.Fprintln(w, "      release (viene en el fichero plazum-corpus.huella).")
}

// ---------------------------------------------------------------------------
// Extraer: entrada adversaria hasta que la huella diga lo contrario
// ---------------------------------------------------------------------------

// extraerCorpus desempaqueta en `destino`, que tiene que estar vacio.
//
// TODO LO QUE SALE DE UN .tar.gz ES DE FUERA. La huella se comprueba DESPUES de
// extraer, porque hace falta el arbol para calcularla, asi que entre la primera
// lectura y esa comprobacion no hay nada mas que estas guardas. Cada una tapa
// una familia entera y ninguna es teorica:
//
//	nombre absoluto o con ..   escribir fuera del destino (el zip-slip clasico)
//	nombre con \ o con unidad  lo mismo, escrito para Windows
//	tipo que no es fichero     un enlace dentro del tar apunta donde quiera y
//	                           la escritura siguiente lo sigue
//	tope de tamano y de cuenta la bomba de descompresion: trescientos kilobytes
//	                           que se convierten en el disco entero
func extraerCorpus(tarball, destino string) (int, error) {
	f, err := os.Open(tarball) // #nosec G304 -- ruta que teclea el operador en su maquina
	if err != nil {
		return 0, fmt.Errorf("no puedo abrir %q: %w. Arreglo: comprueba la ruta del "+
			"fichero de corpus que te has bajado de la release", tarball, err)
	}
	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return 0, fmt.Errorf("%q no es un .tar.gz legible: %w. Arreglo: vuelve a bajarlo "+
			"de la release; una descarga a medias da exactamente este error", tarball, err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	n := 0
	var total int64
	for {
		cab, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("el corpus empaquetado esta roto: %w", err)
		}

		rel, err := nombreSeguro(cab.Name)
		if err != nil {
			return 0, err
		}

		switch cab.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(filepath.Join(destino, rel), 0o750); err != nil {
				return 0, err
			}
			continue
		case tar.TypeReg:
			// sigue abajo
		default:
			return 0, fmt.Errorf("el corpus empaquetado trae %q, que no es un fichero "+
				"ni un directorio (tipo %q). Un corpus son datos: cualquier otra cosa "+
				"dentro es un intento de escribir donde no toca, y se para aqui",
				cab.Name, string(cab.Typeflag))
		}

		if n >= maxFicherosCorpus {
			return 0, fmt.Errorf("el corpus empaquetado trae mas de %d ficheros. "+
				"El corpus real son unos trescientos: esto no es un corpus", maxFicherosCorpus)
		}
		if cab.Size > maxBytesFichero {
			return 0, fmt.Errorf("%q ocupa %d bytes y el tope por fichero son %d",
				cab.Name, cab.Size, maxBytesFichero)
		}
		if total+cab.Size > maxBytesTotal {
			return 0, fmt.Errorf("el corpus empaquetado pasa de %d bytes descomprimidos. "+
				"Una bomba de descompresion se ve exactamente asi", maxBytesTotal)
		}

		ruta := filepath.Join(destino, rel)
		if err := os.MkdirAll(filepath.Dir(ruta), 0o750); err != nil {
			return 0, err
		}
		w, err := os.OpenFile(ruta, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) // #nosec G304 -- ruta validada por nombreSeguro
		if err != nil {
			return 0, err
		}
		// io.CopyN con el tope y no io.Copy: la cabecera dice un tamano y el
		// cuerpo puede traer otro. Confiar en la cabecera es confiar en el
		// emisor, que es de lo que va este fichero entero.
		escritos, err := io.CopyN(w, tr, cab.Size)
		cerrar := w.Close()
		if err != nil && !errors.Is(err, io.EOF) {
			return 0, err
		}
		if cerrar != nil {
			return 0, cerrar
		}
		if escritos != cab.Size {
			return 0, fmt.Errorf("%q dice ocupar %d bytes y trae %d", cab.Name, cab.Size, escritos)
		}
		total += escritos
		n++
	}
	if n == 0 {
		return 0, errors.New("el corpus empaquetado no trae ni un fichero. " +
			"Un tar vacio se extrae sin error y deja un directorio vacio, que es la " +
			"forma silenciosa de instalar nada")
	}
	return n, nil
}

// nombreSeguro convierte el nombre que trae el tar en una ruta relativa que no
// puede salir del destino, o falla. No limpia y sigue: falla. Una ruta que se
// "arregla" sola es una ruta cuyo emisor descubre que puede mandar lo que
// quiera mientras alguien se lo corrija.
func nombreSeguro(nombre string) (string, error) {
	malo := func(motivo string) error {
		return fmt.Errorf("el corpus empaquetado trae la ruta %q, que %s. "+
			"Un corpus solo escribe dentro de su directorio, asi que esto se para aqui",
			nombre, motivo)
	}
	if nombre == "" {
		return "", malo("esta vacia")
	}
	if strings.Contains(nombre, `\`) {
		return "", malo("lleva una barra invertida (un tar usa siempre /)")
	}
	if strings.HasPrefix(nombre, "/") {
		return "", malo("es absoluta")
	}
	if filepath.VolumeName(nombre) != "" {
		return "", malo("lleva una unidad de disco")
	}
	for _, parte := range strings.Split(nombre, "/") {
		if parte == ".." {
			return "", malo("sube por encima de su directorio")
		}
	}
	limpia := filepath.ToSlash(filepath.Clean(nombre))
	if limpia == "." || strings.HasPrefix(limpia, "../") || filepath.IsAbs(limpia) {
		return "", malo("no queda dentro del destino al normalizarla")
	}
	return limpia, nil
}

// ---------------------------------------------------------------------------
// Verificar y el parte
// ---------------------------------------------------------------------------

func cmdVerificarCorpus(dir, esperada string, salida, errores io.Writer) int {
	h, err := HuellaDeArbol(dir)
	if err != nil {
		fmt.Fprintln(errores, "error:", err)
		return 1
	}
	ancla, deDonde, err := anclaAUsar(esperada)
	if err != nil {
		fmt.Fprintf(errores, "la huella de %s es\n  %s\n\n", dir, h)
		fmt.Fprintln(errores, "y no se ha podido comprobar contra nada:", err)
		explicarFaltaDeAncla(errores)
		// SE SALE CON 1 Y NO CON 0. «No he podido comprobarlo» no es «esta
		// bien»: confundirlos hace que una maquina sin ancla se lea como una
		// maquina verificada, que es el invariante 8 aplicado a este comando.
		return 1
	}
	if h != ancla {
		fmt.Fprintf(errores, "NO CUADRA.\n  esperada (%s)  %s\n  la de %s  %s\n", deDonde, ancla, dir, h)
		fmt.Fprintf(errores, "\n  Esto NO dice que el corpus este mal: dice que no es el que este binario\n")
		fmt.Fprintf(errores, "  publico. Si es uno mas nuevo, comprueba su huella en la pagina de la\n")
		fmt.Fprintf(errores, "  release y pasala con --huella-esperada.\n")
		return 1
	}
	fmt.Fprintf(salida, "CUADRA: %s es el corpus que se publico con este binario.\n", dir)
	fmt.Fprintf(salida, "  huella %s\n", h)
	fmt.Fprintf(salida, "  comprobada contra %s\n", deDonde)
	return 0
}

// cmdParteDelCorpus es lo que sale con `plazum corpus` a secas: que corpus hay y
// si se puede confiar en el. Siempre sale con 0 porque es un parte, no una
// comprobacion: quien quiera un codigo de salida usa --verificar.
func cmdParteDelCorpus(destino string, salida io.Writer) int {
	fmt.Fprintln(salida)
	if anclaCorpus == "" {
		fmt.Fprintln(salida, "  ancla de este binario   NINGUNA")
		fmt.Fprintln(salida, "    Este binario no se construyo con un corpus de referencia. No puede")
		fmt.Fprintln(salida, "    decir si un corpus es el suyo, asi que --instalar se negara salvo")
		fmt.Fprintln(salida, "    que le pases --huella-esperada.")
	} else {
		fmt.Fprintf(salida, "  ancla de este binario   %s\n", anclaCorpus)
	}

	h, err := HuellaDeArbol(destino)
	if err != nil {
		fmt.Fprintf(salida, "\n  corpus en %-13s NO HAY\n", destino)
		fmt.Fprintln(salida, "\n  Instalalo con el activo de la release:")
		fmt.Fprintln(salida, "    plazum corpus --instalar plazum-corpus.tar.gz")
		fmt.Fprintln(salida, "\n  O prueba el paseo de dos minutos, que no necesita corpus:")
		fmt.Fprintln(salida, "    plazum demo")
		fmt.Fprintln(salida)
		return 0
	}
	fmt.Fprintf(salida, "\n  corpus en %-13s %d paquetes\n", destino, contarPaquetes(destino))
	fmt.Fprintf(salida, "  su huella               %s\n", h)
	switch {
	case anclaCorpus == "":
		fmt.Fprintln(salida, "  estado                  SIN COMPROBAR (este binario no trae ancla)")
	case h == anclaCorpus:
		fmt.Fprintln(salida, "  estado                  CUADRA con el ancla de este binario")
	default:
		fmt.Fprintln(salida, "  estado                  NO CUADRA con el ancla de este binario")
		fmt.Fprintln(salida, "    No dice que este mal: dice que no es el que salio con este binario.")
	}
	fmt.Fprintln(salida)
	return 0
}

// contarPaquetes cuenta directorios con paquete.json dentro, que es exactamente
// lo que corpus.Cargar considera un paquete. Contar directorios a secas daria un
// numero mas alto y mas bonito, y seria mentira.
func contarPaquetes(dir string) int {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, e.Name(), "paquete.json")); err == nil {
			n++
		}
	}
	return n
}
