package plazum

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// DirDeEsqueletos es donde viven los paquetes que existen y todavia no declaran
// ni una obligacion.
//
// Se declara aqui y se usa desde los dos sitios que lo necesitan (este fichero y
// el recorrido de marcos-v1.json) para que el nombre del directorio no se
// escriba dos veces: una segunda copia se queda vieja el dia que se renombre, y
// el sintoma seria un recorrido mirando un directorio que ya no existe.
const DirDeEsqueletos = "esqueletos"

// A6: NINGUN PAQUETE PUBLICADO LLEGA VACIO.
//
// # El defecto que cierra, con su cardinal
//
// A 07-09-2026 `paquetes/` tenia **33 directorios y 21 con obligaciones**. Los
// otros doce cargaban, pasaban el linter, declaraban su URN, su clase y su
// atribucion, y **no decian ni una obligacion**. O sea: contaban como marco en
// todas partes y no entregaban nada.
//
// **Un escaparate que dice 33 y entrega 21 es el 8 de 72 aplicado al catalogo.**
// Y el precio no lo paga quien escribe el numero: lo paga quien instala plazum
// porque su marco esta en la lista, abre la pantalla y no encuentra ni una
// obligacion suya. Esa primera impresion no se repite.
//
// # Por que se mueven y no se borran
//
// Porque lo que traen no es basura: es el andamiaje verificado de un paquete
// futuro —el URN, el estrato legal, la licencia de la fuente y la atribucion—,
// y eso costo mirarlo una vez. Borrarlo obligaria a rehacerlo. Viven en
// `esqueletos/`, que no se publica, y siguen pasando el linter (ver abajo): un
// esqueleto que se pudre en silencio no vale mas que uno borrado.
//
// # Que se afirma aqui, en una frase
//
// Todo paquete bajo `paquetes/` declara al menos una obligacion, y todo paquete
// bajo `esqueletos/` declara CERO. Las dos direcciones, porque sin la segunda
// el directorio de esqueletos se convierte en el sitio donde se aparca trabajo
// terminado y nadie lo devuelve.
func TestNingunPaquetePublicadoLlegaVacio(t *testing.T) {
	publicados, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatalf("no se puede cargar el corpus publicado: %v", err)
	}
	if len(publicados) < MinimoDeMarcos {
		t.Fatalf("el corpus publicado trae %d paquetes y el suelo es %d: esta puerta estaria "+
			"mirando medio corpus", len(publicados), MinimoDeMarcos)
	}
	var vacios []string
	for _, p := range publicados {
		if len(p.Obligaciones) == 0 {
			vacios = append(vacios, p.URN)
		}
	}
	sort.Strings(vacios)
	if len(vacios) > 0 {
		t.Errorf("estos %d paquetes de paquetes/ no declaran ni una obligacion: %v\n"+
			"  Cargan, pasan el linter y cuentan como marco en el README, en CORPUS.md y en\n"+
			"  la pantalla de controles. Lo que no hacen es entregar nada.\n"+
			"  Arreglo: escribirles sus obligaciones, o moverlos a %s/ hasta que las tengan.\n"+
			"  Y si se mueven, hay numeros publicados que bajan con ellos: los suelos de\n"+
			"  MinimoDeMarcos, el README y paquetes/CORPUS.md tienen puerta y lo diran.",
			len(vacios), vacios, DirDeEsqueletos)
	}

	// LA OTRA DIRECCION: un esqueleto que ya tiene obligaciones vuelve.
	//
	// Sin esta mitad, `esqueletos/` seria el sitio donde se aparca trabajo hecho
	// y nadie lo devuelve al escaparate: el paquete estaria escrito, probado y
	// sin publicar, que es la forma cara de perder trabajo.
	esqueletos, err := corpus.Cargar(DirDeEsqueletos)
	if err != nil {
		t.Fatalf("no se puede cargar %s/: %v.\n"+
			"  Un esqueleto tiene que seguir pasando el linter: si no, se pudre sin que\n"+
			"  nadie lo vea y el dia que alguien lo escriba se encontrara el andamiaje roto",
			DirDeEsqueletos, err)
	}
	if len(esqueletos) == 0 {
		t.Fatalf("%s/ esta vacio y este recorrido estaria midiendo la nada", DirDeEsqueletos)
	}
	var conObligaciones []string
	for _, p := range esqueletos {
		if len(p.Obligaciones) > 0 {
			conObligaciones = append(conObligaciones, p.URN)
		}
	}
	sort.Strings(conObligaciones)
	if len(conObligaciones) > 0 {
		t.Errorf("estos %d paquetes de %s/ YA declaran obligaciones: %v\n"+
			"  Estan escritos y no se publican, que es la forma cara de perder trabajo.\n"+
			"  Arreglo: moverlos a paquetes/ y subir MinimoDeMarcos en el mismo commit.",
			len(conObligaciones), DirDeEsqueletos, conObligaciones)
	}
}

// TestElDetectorDePaqueteVacioSabePonerseRojo es el control negativo, en las dos
// direcciones.
//
// Hace falta porque la puerta de arriba NACE VERDE por construccion: se escribe
// en el mismo commit que deja los dos directorios como tienen que estar, asi que
// su primer recorrido no puede encontrar nada. Sin este, no se distingue «los
// dos directorios estan bien» de «estoy contando obligaciones de una lista que
// siempre sale vacia».
//
// Se hace sobre paquetes SINTETICOS y no mutando el arbol: mover un directorio
// de verdad para ver el rojo dejaria el repositorio a medias si el test falla
// por otro motivo.
func TestElDetectorDePaqueteVacioSabePonerseRojo(t *testing.T) {
	vacio := &corpus.Paquete{URN: "urn:sintetico:vacio"}
	lleno := &corpus.Paquete{
		URN:          "urn:sintetico:lleno",
		Obligaciones: []corpus.Obligacion{{ID: "x.o.1"}},
	}
	cuentaVacios := func(ps []*corpus.Paquete) int {
		n := 0
		for _, p := range ps {
			if len(p.Obligaciones) == 0 {
				n++
			}
		}
		return n
	}
	if cuentaVacios([]*corpus.Paquete{vacio, lleno}) != 1 {
		t.Error("el detector no distingue un paquete sin obligaciones de uno con ellas, " +
			"asi que la mitad de arriba aprobaria un escaparate vacio")
	}
	if cuentaVacios([]*corpus.Paquete{lleno}) != 0 {
		t.Error("el detector acusa a un paquete que SI trae obligaciones: con un detector " +
			"asi la puerta seria un rojo permanente y se acabaria aflojando")
	}
	// Y LA DIRECCION CONTRARIA, que es la que devuelve trabajo al escaparate.
	conObl := func(ps []*corpus.Paquete) int {
		n := 0
		for _, p := range ps {
			if len(p.Obligaciones) > 0 {
				n++
			}
		}
		return n
	}
	if conObl([]*corpus.Paquete{vacio, lleno}) != 1 {
		t.Error("el detector de «esqueleto que ya esta escrito» no ve uno que lo esta")
	}
	if conObl([]*corpus.Paquete{vacio}) != 0 {
		t.Error("el detector de «esqueleto que ya esta escrito» acusa a uno que sigue vacio")
	}
}

// TestTodoEsqueletoSigueSiendoUnPaqueteLegible es la mitad barata que evita que
// `esqueletos/` se convierta en un cementerio.
//
// Lo que se afirma es concreto: cada esqueleto tiene su `paquete.json`, y el
// directorio entero carga con `corpus.Cargar`, que es el mismo linter que pasa
// el corpus publicado. Sin esto, apartar un paquete equivaldria a dejar de
// mirarlo, y el dia que alguien fuera a escribirlo se encontraria el andamiaje
// podrido: un URN mal formado, una atribucion que ya no vale, una licencia sin
// declarar.
func TestTodoEsqueletoSigueSiendoUnPaqueteLegible(t *testing.T) {
	ents, err := os.ReadDir(DirDeEsqueletos)
	if err != nil {
		t.Fatalf("no puedo leer %s/: %v", DirDeEsqueletos, err)
	}
	var dirs []string
	for _, e := range ents {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	if len(dirs) != EsqueletosEsperados {
		t.Errorf("%s/ tiene %d directorios y se esperaban %d.\n"+
			"  Si ha SUBIDO, alguien ha apartado un paquete: dilo en el commit y sube este\n"+
			"  numero. Si ha BAJADO, o se ha escrito (y entonces va a paquetes/) o se ha\n"+
			"  borrado, y borrar el andamiaje de un marco es una decision, no una limpieza.",
			DirDeEsqueletos, len(dirs), EsqueletosEsperados)
	}
	for _, d := range dirs {
		if _, err := os.Stat(filepath.Join(DirDeEsqueletos, d, "paquete.json")); err != nil {
			t.Errorf("%s/%s no tiene paquete.json, asi que no es un esqueleto: es un "+
				"directorio", DirDeEsqueletos, d)
		}
	}
}

// EsqueletosEsperados son los doce que salieron de paquetes/ el 08-09-2026.
//
// IGUALDAD EXACTA Y NO UN SUELO, y en las dos direcciones: por arriba, porque
// apartar un paquete tiene que costar tocar un numero y explicarlo; por abajo,
// porque el dia que uno se escriba y vuelva al escaparate hay que venir a
// bajarlo, y ese es el commit en el que ademas sube MinimoDeMarcos. Los dos
// numeros se mueven juntos o uno de los dos miente.
const EsqueletosEsperados = 12
