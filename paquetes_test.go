package plazum

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"plazum/nucleo/corpus"
)

// MinimoDeMarcos es el suelo del corpus publicado: los 30 marcos de
// paquetes/CORPUS.md. Es un numero declarado a proposito y no un ">= 2": el
// barrido de mutacion enseno que con el umbral viejo se podian perder 29
// paquetes sin que ninguna puerta se enterara.
const MinimoDeMarcos = 30

// MinimoDeDorados son los relojes insignia que el proyecto promete en verde
// (ENS art. 31 e INES, RGPD art. 33, CRA art. 14.1). Bajar de aqui tiene que
// ser una decision, no un descuido.
const MinimoDeDorados = 12

// directoriosPublicados enumera los directorios de paquetes/. La convencion es
// que todo directorio bajo paquetes/ es un paquete publicado; no hay directorios
// de adorno.
func directoriosPublicados(raiz string) ([]string, error) {
	ents, err := os.ReadDir(raiz)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, e := range ents {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Strings(dirs)
	return dirs, nil
}

// paquetesSinManifiesto devuelve los directorios publicados que NO tienen
// paquete.json y que por tanto corpus.Cargar se salta en silencio.
//
// BARRIDO DE MUTACION, el hallazgo de este fichero: Cargar hace
// `if os.IsNotExist(err) { continue }`, asi que renombrar
// paquetes/ens/paquete.json borraba el ENS del corpus entero (sus obligaciones,
// sus relojes y sus 24 dorados) y las dos puertas de abajo seguian en verde,
// porque una decia "al menos 2 paquetes" y la otra "al menos 3 dorados". Un
// paquete que no carga cumple "pasa el linter" de forma vacia.
func paquetesSinManifiesto(raiz string) ([]string, error) {
	dirs, err := directoriosPublicados(raiz)
	if err != nil {
		return nil, err
	}
	var sin []string
	for _, d := range dirs {
		if _, err := os.Stat(filepath.Join(raiz, d, "paquete.json")); err != nil {
			sin = append(sin, d)
		}
	}
	return sin, nil
}

// TestNingunPaqueteSeCaeDelCorpusEnSilencio es la puerta que faltaba: lo que
// esta publicado tiene que estar cargado. Si no, "todo paquete publicado pasa el
// linter" se cumple sin comprobar nada.
func TestNingunPaqueteSeCaeDelCorpusEnSilencio(t *testing.T) {
	dirs, err := directoriosPublicados("paquetes")
	if err != nil {
		t.Fatalf("no puedo leer paquetes/: %v", err)
	}
	if len(dirs) < MinimoDeMarcos {
		t.Fatalf("paquetes/ tiene %d directorios y CORPUS.md promete al menos %d marcos: "+
			"borrado, renombrado, o el corpus ha encogido sin decirlo",
			len(dirs), MinimoDeMarcos)
	}
	sin, err := paquetesSinManifiesto("paquetes")
	if err != nil {
		t.Fatal(err)
	}
	if len(sin) > 0 {
		t.Errorf("estos directorios de paquetes/ no tienen paquete.json y corpus.Cargar "+
			"se los salta sin decir nada, asi que sus obligaciones, sus relojes y sus "+
			"dorados desaparecen del producto con todas las puertas en verde: %v", sin)
	}
	ps, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatalf("el corpus publicado no carga: %v", err)
	}
	if len(ps) != len(dirs) {
		t.Errorf("hay %d directorios publicados y solo %d paquetes cargados: %d se han "+
			"caido del corpus", len(dirs), len(ps), len(dirs)-len(ps))
	}
}

// Control negativo: se demuestra que la comprobacion de arriba salta cuando
// debe, sobre un corpus sintetico al que se le quita el manifiesto a un
// paquete. Sin esto, el verde no prueba nada.
func TestLaComprobacionDeCorpusCompletoSaltaCuandoDebe(t *testing.T) {
	raiz := t.TempDir()
	escribir := func(nombre, contenido string) {
		d := filepath.Join(raiz, nombre)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if contenido == "" {
			return // directorio publicado SIN manifiesto
		}
		if err := os.WriteFile(filepath.Join(d, "paquete.json"), []byte(contenido), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	escribir("norma-a", paqueteDemo("urn:demo:a", "sistema", "categoria"))
	escribir("norma-b", paqueteDemo("urn:demo:b", "proveedor", "criticidad"))

	sin, err := paquetesSinManifiesto(raiz)
	if err != nil {
		t.Fatal(err)
	}
	if len(sin) != 0 {
		t.Fatalf("falso positivo sobre un corpus completo: %v", sin)
	}
	ps, err := corpus.Cargar(raiz)
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 2 {
		t.Fatalf("esperaba 2 paquetes cargados y hay %d", len(ps))
	}

	// Y ahora la mutacion: al paquete b se le renombra el manifiesto.
	viejo := filepath.Join(raiz, "norma-b", "paquete.json")
	if err := os.Rename(viejo, viejo+".bak"); err != nil {
		t.Fatal(err)
	}
	sin, err = paquetesSinManifiesto(raiz)
	if err != nil {
		t.Fatal(err)
	}
	if len(sin) != 1 || sin[0] != "norma-b" {
		t.Fatalf("la comprobacion no ve el paquete sin manifiesto: %v", sin)
	}
	ps, err = corpus.Cargar(raiz)
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 1 {
		t.Fatalf("Cargar debia devolver 1 paquete tras la mutacion y devolvio %d", len(ps))
	}
	dirs, err := directoriosPublicados(raiz)
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) == len(dirs) {
		t.Fatal("la comparacion directorios/cargados no distingue el corpus mutilado")
	}
}

// TestTodosLosPaquetesPublicadosPasanElLinter es la puerta de la semana 0:
// un paquete que no pasa el linter no entra al repositorio. Cargar ya ejecuta
// el linter y rechaza el directorio entero si algo esta mal.
func TestTodosLosPaquetesPublicadosPasanElLinter(t *testing.T) {
	ps, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatalf("el corpus publicado no pasa el linter: %v", err)
	}
	if len(ps) < MinimoDeMarcos {
		t.Fatalf("el corpus publicado tiene %d paquetes y CORPUS.md promete al menos %d",
			len(ps), MinimoDeMarcos)
	}
	// Redundante con el linter (que ya rechaza un paquete sin fuente, y asi se
	// comprobo mutandolo), pero es la frontera legal: se deja escrita aqui para
	// que siga habiendo puerta si algun dia el linter relaja la regla.
	for _, p := range ps {
		if p.Fuente == "" {
			t.Errorf("%s sin fuente", p.URN)
		}
	}
}

// TestTodoPaquetePublicadoDeclaraSuRegimenYSuAtribucion es la puerta de higiene
// legal del corpus, y no es redundante con el linter aunque lo parezca.
//
// El linter comprueba que los dos campos ESTAN y que el regimen es coherente con
// la clase. Lo que no puede comprobar es que la atribucion diga algo: "n/a"
// cumple "no vacio" y no atribuye nada. Aqui se mira el FONDO en el caso donde
// la atribucion es una obligacion y no una cortesia, que es el del DOUE: la
// Decision 2011/833/UE autoriza la reutilizacion a cambio de atribuir, asi que un
// paquete de esa fuente tiene que nombrar a quien atribuye.
//
// Se mira sobre el corpus REAL y no sobre un paquete de prueba: el fallo que
// esto caza es que alguien rellene los 30 paquetes de una tacada con un texto de
// relleno, que es exactamente el riesgo de anadir un campo obligatorio a un
// corpus que ya existe.
func TestTodoPaquetePublicadoDeclaraSuRegimenYSuAtribucion(t *testing.T) {
	ps, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatalf("el corpus publicado no carga: %v", err)
	}
	if len(ps) < MinimoDeMarcos {
		t.Fatalf("solo %d paquetes cargados: este test estaria mirando medio corpus", len(ps))
	}
	vistas := map[corpus.LicenciaFuente]int{}
	for _, p := range ps {
		if p.LicenciaFuente == "" {
			t.Errorf("%s no declara licencia_fuente", p.URN)
		}
		if p.Atribucion == "" {
			t.Errorf("%s no declara atribucion", p.URN)
			continue
		}
		vistas[p.LicenciaFuente]++
		// El relleno se caza por longitud: un aviso de derechos util no cabe
		// en veinte caracteres, y "n/a" o "pendiente" no llegan.
		if len(p.Atribucion) < 40 {
			t.Errorf("%s tiene una atribucion de %d caracteres (%q). Eso no es un aviso de "+
				"derechos, es un hueco relleno", p.URN, len(p.Atribucion), p.Atribucion)
		}
		// Y el fondo, donde la atribucion es la condicion de la autorizacion.
		if p.LicenciaFuente == corpus.DOUEDecision2011833 &&
			!strings.Contains(p.Atribucion, "Unión Europea") {
			t.Errorf("%s reutiliza el DOUE y su atribucion no nombra a quien atribuye: %q.\n"+
				"  La Decision 2011/833/UE autoriza la reutilizacion A CAMBIO de atribuir: "+
				"sin nombrar al titular, el paquete usa el texto sin cumplir la condicion.",
				p.URN, p.Atribucion)
		}
		if p.LicenciaFuente == corpus.BOETRLPI13 && !strings.Contains(p.Atribucion, "BOE") {
			t.Errorf("%s reproduce texto del BOE y su atribucion no cita la fuente: %q",
				p.URN, p.Atribucion)
		}
	}
	// Y el corpus publicado tiene que ejercer de verdad los regimenes que el
	// proyecto declara. Si un dia queda uno solo, esta puerta habria dejado de
	// medir la estratificacion y solo mediria que un campo no esta vacio.
	if len(vistas) < 4 {
		t.Errorf("el corpus publicado solo usa %d regimenes de licencia_fuente (%v) y "+
			"docs/LICENCIAS.md describe una estratificacion de varios", len(vistas), vistas)
	}
}

// TestLosDoradosPublicadosPasanContraElMotor es la puerta de cobertura: cada
// reloj del corpus publicado se recalcula con el motor y se compara con el
// esperado derivado del texto. Si discrepan, gana el dorado.
func TestLosDoradosPublicadosPasanContraElMotor(t *testing.T) {
	ps, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	conDorados := 0
	for _, p := range ps {
		for _, e := range corpus.EjecutarDorados(p) {
			t.Errorf("%s: %v", p.URN, e)
		}
		total += len(p.Dorados)
		if len(p.Dorados) > 0 {
			conDorados++
		}
	}
	if total < MinimoDeDorados {
		t.Fatalf("el corpus publicado tiene %d dorados y los relojes insignia son al menos %d",
			total, MinimoDeDorados)
	}
	if conDorados < 4 {
		t.Fatalf("solo %d paquetes traen dorados; ENS, RGPD, CRA e ISO 27001 los tienen: "+
			"uno se ha quedado sin cobertura o se ha caido del corpus", conDorados)
	}
}
