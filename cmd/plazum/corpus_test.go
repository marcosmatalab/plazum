package main

// Las puertas del corpus que viaja en la release.
//
// QUE SE VIGILA AQUI, y por que cada cosa:
//
//  1. QUE LA HUELLA SEA UNA HUELLA. Si no cambia cuando cambia el corpus, no
//     esta comprobando nada y todo lo demas de este fichero es decoracion.
//  2. QUE LA NADA NO ABRA LA PUERTA (invariante 8). El valor cero de
//     `anclaCorpus` es la cadena vacia y es el que sale por olvidar el -ldflags.
//     Se recorren las TRES formas: ausente, presente y vacia, y presente y no
//     interpretable.
//  3. QUE UN .tar.gz DE FUERA NO ESCRIBA DONDE NO DEBE. La huella se comprueba
//     DESPUES de extraer, asi que entre la primera lectura y esa comprobacion
//     lo unico que hay son las guardas de extraerCorpus.
//  4. QUE UN FALLO NO DEJE NADA PUESTO. Media instalacion es peor que ninguna:
//     el calendario leeria un corpus que nadie verifico.

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// anclaDePrueba pone y quita el ancla del binario alrededor de un caso. Es una
// variable de paquete, asi que estos casos NO pueden ser paralelos: dos casos
// pisandose el ancla darian verdes y rojos al azar.
func anclaDePrueba(t *testing.T, valor string) {
	t.Helper()
	previo := anclaCorpus
	anclaCorpus = valor
	t.Cleanup(func() { anclaCorpus = previo })
}

// corpusMinimo escribe un arbol con la forma que corpus.Cargar espera.
func corpusMinimo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	escribir(t, filepath.Join(dir, "marco-a", "paquete.json"), `{"urn":"urn:x:a"}`)
	escribir(t, filepath.Join(dir, "marco-a", "pruebas", "uno.json"), `{"caso":"uno"}`)
	escribir(t, filepath.Join(dir, "marco-b", "paquete.json"), `{"urn":"urn:x:b"}`)
	escribir(t, filepath.Join(dir, "LEEME.md"), "documentacion del corpus")
	return dir
}

func escribir(t *testing.T, ruta, contenido string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(ruta), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte(contenido), 0o600); err != nil {
		t.Fatal(err)
	}
}

// ---------------------------------------------------------------------------
// 1. La huella es una huella
// ---------------------------------------------------------------------------

// TestLaHuellaCambiaConCadaCosaQueCambiaElCorpus.
//
// EL CONTROL POSITIVO Y EL NEGATIVO EN EL MISMO CASO, que es lo que hace que
// esto valga: se comprueba que la huella NO cambia con lo que no importa (el
// orden en que el sistema de ficheros devuelva las entradas, el codigo Go de al
// lado) y que SI cambia con cada una de las cuatro formas de tocar el corpus.
//
// Sin la mitad de arriba, una funcion que devolviera una constante distinta cada
// vez pasaria la mitad de abajo entera.
func TestLaHuellaCambiaConCadaCosaQueCambiaElCorpus(t *testing.T) {
	base := corpusMinimo(t)
	h0, err := HuellaDeArbol(base)
	if err != nil {
		t.Fatal(err)
	}

	// Estable entre dos lecturas del mismo arbol.
	if h1, err := HuellaDeArbol(base); err != nil || h1 != h0 {
		t.Fatalf("la huella del mismo arbol cambia entre lecturas: %q vs %q (%v)", h0, h1, err)
	}

	// Un fichero .go al lado NO cambia la huella: es del build del repositorio,
	// no del corpus. Si esto rompiera, cada test que R2 anadiera en paquetes/
	// invalidaria el corpus publicado sin cambiar ni una obligacion.
	escribir(t, filepath.Join(base, "marco-a", "algo_test.go"), "package a")
	if h, err := HuellaDeArbol(base); err != nil || h != h0 {
		t.Fatalf("un fichero .go ha cambiado la huella del corpus: %q vs %q (%v)", h0, h, err)
	}

	casos := []struct {
		nombre string
		hacer  func(dir string)
	}{
		{"cambiar el contenido de un paquete", func(d string) {
			escribir(t, filepath.Join(d, "marco-a", "paquete.json"), `{"urn":"urn:x:a","plazo":96}`)
		}},
		{"anadir un paquete", func(d string) {
			escribir(t, filepath.Join(d, "marco-c", "paquete.json"), `{"urn":"urn:x:c"}`)
		}},
		{"quitar un paquete", func(d string) {
			if err := os.RemoveAll(filepath.Join(d, "marco-b")); err != nil {
				t.Fatal(err)
			}
		}},
		// EL CASO QUE UN RESUMEN INGENUO NO CAZA: mover un fichero de sitio sin
		// cambiar ni un byte de su contenido. Si la huella fuera la suma de las
		// sumas, esto no la movería, y un paquete entero podria cambiar de
		// identidad sin que se notara.
		{"mover un fichero de paquete sin tocar su contenido", func(d string) {
			b, err := os.ReadFile(filepath.Join(d, "marco-a", "paquete.json"))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(filepath.Join(d, "marco-a", "paquete.json")); err != nil {
				t.Fatal(err)
			}
			escribir(t, filepath.Join(d, "marco-b", "robado.json"), string(b))
		}},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			dir := corpusMinimo(t)
			antes, err := HuellaDeArbol(dir)
			if err != nil {
				t.Fatal(err)
			}
			c.hacer(dir)
			despues, err := HuellaDeArbol(dir)
			if err != nil {
				t.Fatal(err)
			}
			if antes == despues {
				t.Fatalf("%s NO ha movido la huella (%s). Una huella que no cambia cuando "+
					"cambia el corpus no comprueba nada", c.nombre, antes)
			}
		})
	}
}

// TestUnCorpusVacioNoTieneHuellaValida.
//
// Un directorio vacio tiene resumen, y ese resumen seria un numero perfectamente
// respetable para un corpus sin ni un paquete dentro. Publicar eso significa que
// empaquetar el directorio equivocado da un activo valido y firmado que instala
// nada, y el fallo se descubre el dia que alguien mira su calendario vacio.
func TestUnCorpusVacioNoTieneHuellaValida(t *testing.T) {
	if _, err := HuellaDeArbol(t.TempDir()); err == nil {
		t.Fatal("un directorio vacio ha dado huella. Empaquetar el sitio equivocado " +
			"produciria un activo firmado que instala un corpus vacio")
	}
	if _, err := HuellaDeArbol(filepath.Join(t.TempDir(), "no-existe")); err == nil {
		t.Fatal("un directorio que no existe ha dado huella")
	}
}

// ---------------------------------------------------------------------------
// 2. La nada no abre la puerta (invariante 8)
// ---------------------------------------------------------------------------

// TestSinAnclaNoSeInstalaNada recorre las tres formas de la nada.
//
// Las tres son distintas y las tres salen por sitios distintos:
//
//	ausente / vacia    el -ldflags que nadie puso. Es la que sale por olvido.
//	no interpretable   un -ldflags mal escrito. Es un dato QUE HAY y no se
//	                   entiende, y tomarlo por ausencia seria inventarse un
//	                   valor; tomarlo por "no cuadra" mandaria a arreglar el
//	                   corpus cuando lo roto es el binario.
//
// Ninguna de las tres puede acabar en "adelante".
func TestSinAnclaNoSeInstalaNada(t *testing.T) {
	casos := []struct {
		nombre string
		ancla  string
		quiere error
	}{
		{"ancla vacia (el -ldflags que nadie puso)", "", ErrSinAncla},
		{"ancla con espacios, que es vacia disfrazada", "   ", ErrSinAncla},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			anclaDePrueba(t, strings.TrimSpace(c.ancla))
			_, _, err := anclaAUsar("")
			if !errors.Is(err, c.quiere) {
				t.Fatalf("con el ancla %q se ha obtenido %v y hacia falta %v. "+
					"Una nada que autoriza es la forma permisiva del valor cero",
					c.ancla, err, c.quiere)
			}
		})
	}

	// LA TERCERA FORMA: presente y no interpretable. Tiene que dar error, y un
	// error DISTINTO del de ausencia, porque el arreglo es otro.
	for _, malo := range []string{"no-es-hexadecimal", "abc", strings.Repeat("z", 64)} {
		t.Run("ancla ilegible "+malo, func(t *testing.T) {
			anclaDePrueba(t, malo)
			_, _, err := anclaAUsar("")
			if err == nil {
				t.Fatalf("un ancla ilegible (%q) ha pasado por buena", malo)
			}
			if errors.Is(err, ErrSinAncla) {
				t.Fatalf("un ancla ilegible (%q) se ha confundido con la ausencia de ancla. "+
					"Son dos averias distintas con dos arreglos distintos", malo)
			}
		})
	}
}

// TestInstalarSeNiegaSinAnclaYNoDejaNada es la de arriba pero por el comando
// entero, porque una funcion que devuelve el error correcto y un comando que lo
// ignora son compatibles.
func TestInstalarSeNiegaSinAnclaYNoDejaNada(t *testing.T) {
	anclaDePrueba(t, "")
	dir := t.TempDir()
	tarball := filepath.Join(dir, "corpus.tar.gz")
	empaquetarPara(t, corpusMinimo(t), tarball)

	destino := filepath.Join(dir, "paquetes")
	var salida, errores bytes.Buffer
	rc := cmdInstalarCorpus(tarball, destino, "", false, &salida, &errores)
	if rc == 0 {
		t.Fatal("se ha instalado un corpus sin nada contra que comprobarlo")
	}
	if _, err := os.Stat(destino); err == nil {
		t.Fatal("no se instalo y aun asi el destino existe: media instalacion es peor que ninguna")
	}
	if !strings.Contains(errores.String(), "--huella-esperada") {
		t.Fatalf("el error no dice como salir de esto. Un error de cumplimiento sin "+
			"arreglo manda a leer el codigo fuente. Salida:\n%s", errores.String())
	}
	sinRastro(t, filepath.Dir(destino))
}

// TestUnaHuellaQueNoCuadraNoInstalaYLoDiceEnLasDosDirecciones.
func TestUnaHuellaQueNoCuadraNoInstalaYLoDiceEnLasDosDirecciones(t *testing.T) {
	origen := corpusMinimo(t)
	real, err := HuellaDeArbol(origen)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	tarball := filepath.Join(dir, "corpus.tar.gz")
	empaquetarPara(t, origen, tarball)
	destino := filepath.Join(dir, "paquetes")

	otra := strings.Repeat("a", 64)
	anclaDePrueba(t, otra)

	var salida, errores bytes.Buffer
	if rc := cmdInstalarCorpus(tarball, destino, "", false, &salida, &errores); rc == 0 {
		t.Fatal("se ha instalado un corpus cuya huella no cuadra con el ancla")
	}
	if _, err := os.Stat(destino); err == nil {
		t.Fatal("no cuadraba y aun asi ha quedado un corpus puesto")
	}
	// LAS DOS HUELLAS EN EL MENSAJE, y no solo "no cuadra". Sin las dos, quien
	// lo lea no puede ir a la pagina de la release a comparar, que es
	// exactamente lo que el mensaje le pide que haga.
	texto := errores.String()
	for _, quiere := range []string{otra, real} {
		if !strings.Contains(texto, quiere) {
			t.Fatalf("el error no dice la huella %s. Salida:\n%s", quiere, texto)
		}
	}
	sinRastro(t, filepath.Dir(destino))

	// Y la vuelta: con el ancla buena, el mismo fichero SI entra. Sin esto, una
	// funcion que se negara siempre pasaria todo lo de arriba.
	anclaDePrueba(t, real)
	salida.Reset()
	errores.Reset()
	if rc := cmdInstalarCorpus(tarball, destino, "", false, &salida, &errores); rc != 0 {
		t.Fatalf("el corpus correcto NO ha entrado: rc=%d\n%s", rc, errores.String())
	}
	if _, err := os.Stat(filepath.Join(destino, "marco-a", "paquete.json")); err != nil {
		t.Fatalf("dice que instalo y el paquete no esta: %v", err)
	}
}

// TestVerificarSaleConUnoCuandoNoHaPodidoComprobar.
//
// «No he podido comprobarlo» y «esta bien» son cosas distintas, y un guion que
// las confunda lee una maquina sin ancla como una maquina verificada. Es la
// misma regla que separa un paso saltado de un paso aprobado en la prueba de la
// maquina limpia.
func TestVerificarSaleConUnoCuandoNoHaPodidoComprobar(t *testing.T) {
	dir := corpusMinimo(t)
	anclaDePrueba(t, "")
	var salida, errores bytes.Buffer
	if rc := cmdVerificarCorpus(dir, "", &salida, &errores); rc == 0 {
		t.Fatal("--verificar ha salido con 0 sin haber comprobado nada. " +
			"Un guion que lea ese 0 dara por verificada una maquina que no lo esta")
	}
}

// ---------------------------------------------------------------------------
// 3. El .tar.gz es entrada adversaria
// ---------------------------------------------------------------------------

// TestUnTarballHostilNoEscribeFueraDeSuSitio.
//
// La huella se comprueba DESPUES de extraer, asi que estas guardas son lo unico
// que hay entre un fichero de fuera y el disco. Cada caso es una familia entera.
func TestUnTarballHostilNoEscribeFueraDeSuSitio(t *testing.T) {
	casos := []struct {
		nombre  string
		entrada func(tw *tar.Writer)
	}{
		{"ruta que sube con ..", func(tw *tar.Writer) {
			ponerFichero(tw, "../fuera.json", "robado")
		}},
		{"ruta que sube por dentro", func(tw *tar.Writer) {
			ponerFichero(tw, "marco/../../fuera.json", "robado")
		}},
		{"ruta absoluta", func(tw *tar.Writer) {
			ponerFichero(tw, "/etc/plazum.json", "robado")
		}},
		{"ruta con barra invertida", func(tw *tar.Writer) {
			ponerFichero(tw, `..\fuera.json`, "robado")
		}},
		{"ruta con unidad de disco", func(tw *tar.Writer) {
			ponerFichero(tw, `C:/fuera.json`, "robado")
		}},
		// UN ENLACE NO ES UN FICHERO. Si se aceptara, el enlace apuntaria fuera
		// del destino y la escritura SIGUIENTE lo seguiria: dos entradas
		// inocentes por separado que juntas escriben donde quieran.
		{"enlace simbolico", func(tw *tar.Writer) {
			_ = tw.WriteHeader(&tar.Header{
				Name: "salida", Typeflag: tar.TypeSymlink, Linkname: "/etc",
			})
		}},
		{"enlace duro", func(tw *tar.Writer) {
			_ = tw.WriteHeader(&tar.Header{
				Name: "salida", Typeflag: tar.TypeLink, Linkname: "/etc/passwd",
			})
		}},
		{"un dispositivo", func(tw *tar.Writer) {
			_ = tw.WriteHeader(&tar.Header{Name: "disco", Typeflag: tar.TypeChar})
		}},
		// La cabecera dice un tamano y el cuerpo trae otro: confiar en la
		// cabecera es confiar en el emisor.
		{"la cabecera miente sobre el tamano", func(tw *tar.Writer) {
			_ = tw.WriteHeader(&tar.Header{
				Name: "a/paquete.json", Typeflag: tar.TypeReg, Size: 1 << 20, Mode: 0o644,
			})
			_, _ = tw.Write([]byte("corto"))
		}},
		{"un fichero mas grande que el tope", func(tw *tar.Writer) {
			_ = tw.WriteHeader(&tar.Header{
				Name: "a/gordo.json", Typeflag: tar.TypeReg, Size: maxBytesFichero + 1, Mode: 0o644,
			})
		}},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			dir := t.TempDir()
			tarball := filepath.Join(dir, "hostil.tar.gz")
			escribirTar(t, tarball, c.entrada)

			destino := filepath.Join(dir, "destino")
			if err := os.MkdirAll(destino, 0o750); err != nil {
				t.Fatal(err)
			}
			if _, err := extraerCorpus(tarball, destino); err == nil {
				t.Fatalf("%s se ha extraido sin protestar", c.nombre)
			}
			// Y NADA FUERA DEL DESTINO. Que la funcion devuelva error no
			// demuestra que no escribiera antes de devolverlo.
			if _, err := os.Stat(filepath.Join(dir, "fuera.json")); err == nil {
				t.Fatalf("%s: ha escrito FUERA del destino", c.nombre)
			}
		})
	}
}

// TestUnTarballVacioNoCuentaComoCorpusInstalado.
//
// Un tar sin entradas se extrae sin error y deja un directorio vacio. Esa es la
// forma silenciosa de instalar nada: rc=0, un mensaje de exito, y un calendario
// que manana no tiene ni una fecha.
func TestUnTarballVacioNoCuentaComoCorpusInstalado(t *testing.T) {
	dir := t.TempDir()
	tarball := filepath.Join(dir, "vacio.tar.gz")
	escribirTar(t, tarball, func(tw *tar.Writer) {})
	destino := filepath.Join(dir, "destino")
	if err := os.MkdirAll(destino, 0o750); err != nil {
		t.Fatal(err)
	}
	if _, err := extraerCorpus(tarball, destino); err == nil {
		t.Fatal("un tar vacio se ha aceptado como corpus")
	}
}

// TestNombreSeguroAceptaLoNormal es el control positivo de las guardas de
// arriba. Sin el, una funcion que dijera que no a todo pasaria los diez casos
// hostiles y ademas impediria instalar el corpus de verdad.
func TestNombreSeguroAceptaLoNormal(t *testing.T) {
	for _, bueno := range []string{
		"paquete.json", "ens/paquete.json", "ens/pruebas/caso.json", "./ens/paquete.json",
	} {
		if _, err := nombreSeguro(bueno); err != nil {
			t.Fatalf("nombreSeguro(%q) ha rechazado una ruta normal del corpus: %v", bueno, err)
		}
	}
}

// ---------------------------------------------------------------------------
// 4. La vuelta entera, y el destino que ya existe
// ---------------------------------------------------------------------------

// TestElCorpusSobreviveALaVueltaEntera: empaquetar y desempaquetar tiene que
// dar el MISMO arbol, no uno parecido. Se compara por huella, que es lo que
// decide en produccion.
func TestElCorpusSobreviveALaVueltaEntera(t *testing.T) {
	origen := corpusMinimo(t)
	antes, err := HuellaDeArbol(origen)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	tarball := filepath.Join(dir, "c.tar.gz")
	empaquetarPara(t, origen, tarball)

	destino := filepath.Join(dir, "salida")
	if err := os.MkdirAll(destino, 0o750); err != nil {
		t.Fatal(err)
	}
	if _, err := extraerCorpus(tarball, destino); err != nil {
		t.Fatal(err)
	}
	despues, err := HuellaDeArbol(destino)
	if err != nil {
		t.Fatal(err)
	}
	if antes != despues {
		t.Fatalf("la vuelta ha cambiado el corpus: %s -> %s", antes, despues)
	}

	// Y EL EMPAQUETADO ES REPRODUCIBLE. Sin esto, la suma del activo cambia en
	// cada ejecucion de la release y nadie de fuera puede comprobar que el
	// fichero publicado sale de este corpus.
	otro := filepath.Join(dir, "c2.tar.gz")
	empaquetarPara(t, origen, otro)
	if !bytes.Equal(leer(t, tarball), leer(t, otro)) {
		t.Fatal("dos empaquetados del mismo corpus dan ficheros distintos: el activo " +
			"de la release no seria reproducible")
	}
}

// TestNoSeSobrescribeUnCorpusInstaladoSinPedirlo.
//
// Instalar encima de otro corpus sin avisar cambia las fechas que ensena el
// calendario sin que nadie lo haya pedido, y eso en un producto de plazos es
// exactamente lo que no puede pasar en silencio.
func TestNoSeSobrescribeUnCorpusInstaladoSinPedirlo(t *testing.T) {
	origen := corpusMinimo(t)
	h, err := HuellaDeArbol(origen)
	if err != nil {
		t.Fatal(err)
	}
	anclaDePrueba(t, h)

	dir := t.TempDir()
	tarball := filepath.Join(dir, "c.tar.gz")
	empaquetarPara(t, origen, tarball)
	destino := filepath.Join(dir, "paquetes")
	escribir(t, filepath.Join(destino, "lo-mio", "paquete.json"), `{"urn":"urn:x:mio"}`)

	var salida, errores bytes.Buffer
	if rc := cmdInstalarCorpus(tarball, destino, "", false, &salida, &errores); rc == 0 {
		t.Fatal("ha sobrescrito un corpus ya instalado sin que nadie lo pidiera")
	}
	if _, err := os.Stat(filepath.Join(destino, "lo-mio", "paquete.json")); err != nil {
		t.Fatal("se nego a instalar y aun asi se llevo por delante lo que habia")
	}
	if !strings.Contains(errores.String(), "--forzar") {
		t.Fatalf("el error no dice como seguir. Salida:\n%s", errores.String())
	}

	// Con --forzar si, y entonces lo viejo TIENE que desaparecer: un corpus
	// mezclado con restos del anterior es un corpus cuya huella no cuadra con
	// nada y que nadie podria volver a verificar.
	salida.Reset()
	errores.Reset()
	if rc := cmdInstalarCorpus(tarball, destino, "", true, &salida, &errores); rc != 0 {
		t.Fatalf("--forzar no ha instalado: %s", errores.String())
	}
	if _, err := os.Stat(filepath.Join(destino, "lo-mio")); err == nil {
		t.Fatal("--forzar ha dejado restos del corpus anterior: la huella del destino " +
			"ya no cuadraria con la publicada")
	}
	if final, err := HuellaDeArbol(destino); err != nil || final != h {
		t.Fatalf("tras --forzar el destino no es el corpus publicado: %q vs %q (%v)", final, h, err)
	}
}

// ---------------------------------------------------------------------------
// utilidades
// ---------------------------------------------------------------------------

func empaquetarPara(t *testing.T, dir, destino string) {
	t.Helper()
	if _, err := empaquetarCorpus(dir, destino); err != nil {
		t.Fatal(err)
	}
}

func escribirTar(t *testing.T, ruta string, dentro func(tw *tar.Writer)) {
	t.Helper()
	f, err := os.Create(ruta) // #nosec G304 -- ruta de t.TempDir()
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	dentro(tw)
	_ = tw.Close()
	_ = gz.Close()
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func ponerFichero(tw *tar.Writer, nombre, contenido string) {
	_ = tw.WriteHeader(&tar.Header{
		Name: nombre, Typeflag: tar.TypeReg, Size: int64(len(contenido)), Mode: 0o644,
	})
	_, _ = tw.Write([]byte(contenido))
}

func leer(t *testing.T, ruta string) []byte {
	t.Helper()
	b, err := os.ReadFile(ruta) // #nosec G304 -- ruta de t.TempDir()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// sinRastro comprueba que un intento fallido no dejo el directorio temporal de
// la extraccion tirado. Un `.plazum-corpus-XXXX` abandonado con un corpus sin
// verificar dentro es basura con forma de corpus.
func sinRastro(t *testing.T, dir string) {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), ".plazum-corpus-") {
			t.Fatalf("ha quedado el temporal %s con un corpus sin verificar dentro", e.Name())
		}
	}
}

// TestLoQueSeExtraeYLoQueSeResumenSonElMismoConjunto.
//
// # El agujero, encontrado revisando y no probando
//
// La huella se calcula sobre un SUBCONJUNTO del arbol: `entraEnElCorpus` deja
// fuera el codigo Go, para que un test que anada R2 en `paquetes/` no invalide
// el corpus publicado. Correcto.
//
// Pero `extraerCorpus` extraia el .tar.gz ENTERO. Los dos conjuntos casaban por
// NADA: un tarball podia traer ficheros .go de propina, aterrizar con ellos en
// el disco, y la huella cuadrar igual, porque lo colado no entraba en el resumen
// que decide si el corpus se instala. Es exactamente el invariante 7 con otro
// traje: dos conjuntos emparejados sin una identidad comun.
//
// plazum no ejecuta nada de `paquetes/`, asi que no es ejecucion de codigo. Lo
// que es, es peor de razonar: un corpus verificado que contiene ficheros que su
// verificacion no cubre. Y si el destino cae dentro de un modulo Go (alguien que
// instala el corpus en su clon del repositorio), esos .go si los ve el
// compilador de ese modulo.
//
// # Las dos direcciones
//
// Se rechaza el polizon, Y se comprueba que el corpus legitimo sigue entrando.
// Sin la segunda mitad, una funcion que dijera que no a todo pasaria la primera
// y romperia el producto entero.
func TestLoQueSeExtraeYLoQueSeResumeSonElMismoConjunto(t *testing.T) {
	dir := t.TempDir()

	// Direccion 1: un .go de propina no entra, y no deja nada.
	conPolizon := filepath.Join(dir, "polizon.tar.gz")
	escribirTar(t, conPolizon, func(tw *tar.Writer) {
		ponerFichero(tw, "marco-a/paquete.json", `{"urn":"urn:x:a"}`)
		ponerFichero(tw, "colado.go", "package main // esto no deberia aterrizar")
	})
	destino := filepath.Join(dir, "destino")
	if err := os.MkdirAll(destino, 0o750); err != nil {
		t.Fatal(err)
	}
	if _, err := extraerCorpus(conPolizon, destino); err == nil {
		t.Fatal("un .tar.gz con un fichero .go dentro se ha extraido.\n" +
			"  La huella no cubre los .go, asi que ese fichero habria aterrizado en el\n" +
			"  disco con la huella cuadrando: un corpus verificado con contenido que su\n" +
			"  verificacion no alcanza.")
	}
	if _, err := os.Stat(filepath.Join(destino, "colado.go")); err == nil {
		t.Fatal("el polizon ha aterrizado a pesar del error")
	}

	// Direccion 2, la que impide que esto sea un no a todo: el corpus de verdad
	// entra, y entra entero.
	origen := corpusMinimo(t)
	bueno := filepath.Join(dir, "bueno.tar.gz")
	empaquetarPara(t, origen, bueno)
	destino2 := filepath.Join(dir, "destino2")
	if err := os.MkdirAll(destino2, 0o750); err != nil {
		t.Fatal(err)
	}
	n, err := extraerCorpus(bueno, destino2)
	if err != nil {
		t.Fatalf("el corpus legitimo ya no se extrae: %v", err)
	}
	if n == 0 {
		t.Fatal("el corpus legitimo se ha extraido con cero ficheros")
	}
	// Y el conjunto extraido tiene la misma huella que el original, o sea que no
	// se ha quedado nada por el camino al apretar la regla.
	antes, err := HuellaDeArbol(origen)
	if err != nil {
		t.Fatal(err)
	}
	despues, err := HuellaDeArbol(destino2)
	if err != nil {
		t.Fatal(err)
	}
	if antes != despues {
		t.Fatalf("la regla nueva se ha llevado por delante parte del corpus: %s -> %s",
			antes, despues)
	}
}
