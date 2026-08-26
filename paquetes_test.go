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
	// Redundante con el linter (que ya rechaza un paquete sin identificador, y
	// asi se comprobo mutandolo), pero es la frontera legal: se deja escrita
	// aqui para que siga habiendo puerta si algun dia el linter relaja la regla.
	//
	// Se mira el ENLACE DERIVADO y no el campo: un identificador declarado del
	// que no salga una direccion no cita nada, y ese es justo el fallo que
	// dejaria un tipo nuevo del vocabulario sin su rama en Enlace.
	for _, p := range ps {
		if p.Identificador.Valor == "" {
			t.Errorf("%s sin identificador de fuente", p.URN)
		}
		if p.Enlace() == "" {
			t.Errorf("%s declara el identificador %+v y de el no sale ninguna direccion: "+
				"a un tipo del vocabulario le falta su rama en corpus.Identificador.Enlace",
				p.URN, p.Identificador)
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

// ---------------------------------------------------------------------------
// LA DIRECCION NO ES DATO. Las dos puertas de la derivacion de enlaces.
//
// La propiedad que se compra: si manana un editor reorganiza su sitio, se
// cambia UNA funcion (corpus.Identificador.Enlace) y no treinta y un ficheros
// de datos. Esa promesa solo es cierta mientras se cumplan dos cosas, y aqui
// se comprueban las dos.
// ---------------------------------------------------------------------------

// anfitrionesDeEditor son los sitios de los que el corpus deriva enlaces. Van
// partidos ("boe" + ".es") A PROPOSITO: si se escribieran enteros, este mismo
// fichero se cazaria a si mismo y la puerta habria que apagarla, que es peor
// que no tenerla.
var anfitrionesDeEditor = []string{
	"eur-lex.europa" + ".eu",
	"www.boe" + ".es",
	"www.iso" + ".org",
	"pcisecuritystandards" + ".org",
	"csrc.nist" + ".gov",
}

// El corpus publicado no guarda NINGUNA direccion como identificador, salvo en
// los paquetes que declaran la valvula de escape, y esos declaran su motivo.
//
// El linter ya lo rechaza al cargar; esto lo mira sobre el corpus REAL y sobre
// los BYTES del fichero, que es lo que caza el otro fallo: que alguien vuelva a
// escribir el campo `fuente` del formato viejo y el cargador lo ignore en
// silencio el dia que ese campo salga del tipo.
func TestNingunPaquetePublicadoGuardaUnaDireccionComoIdentificador(t *testing.T) {
	ps, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatalf("el corpus publicado no carga: %v", err)
	}
	if len(ps) < MinimoDeMarcos {
		t.Fatalf("solo %d paquetes cargados: esta puerta estaria mirando medio corpus", len(ps))
	}
	conValvula := 0
	for _, p := range ps {
		if p.Identificador.Tipo == corpus.SinIdentificador {
			conValvula++
			if p.Identificador.Motivo == "" {
				t.Errorf("%s usa la valvula de escape sin motivo", p.URN)
			}
			continue
		}
		enElValor := false
		enElEnlace := false
		for _, h := range anfitrionesDeEditor {
			if strings.Contains(p.Identificador.Valor, h) {
				enElValor = true
			}
			if strings.Contains(p.Enlace(), h) {
				enElEnlace = true
			}
		}
		if enElValor {
			t.Errorf("%s guarda una direccion (%q) donde va el identificador. Una "+
				"direccion como dato se rompe el dia que la pagina se mueve, que es "+
				"justo lo que este formato retira", p.URN, p.Identificador.Valor)
		}
		// Y la otra mitad, que es la que caza que la derivacion deje de
		// derivar: del identificador TIENE que salir la direccion del editor.
		// Con solo la mitad de arriba, una funcion que devolviera el valor tal
		// cual pasaria (el valor no tiene anfitrion, luego no hay hallazgo) y
		// el corpus se quedaria sin enlace sin que nadie dijera nada.
		if !enElEnlace {
			t.Errorf("%s declara tipo %q y de el sale %q, que no apunta a ningun editor "+
				"conocido. O la derivacion dejo de componer la direccion, o este tipo "+
				"apunta a un sitio que esta puerta no conoce",
				p.URN, p.Identificador.Tipo, p.Enlace())
		}
	}
	// La valvula tiene que seguir siendo la EXCEPCION. Si un dia la usan todos,
	// el formato no ha comprado nada y hay que decirlo en voz alta.
	if conValvula*2 >= len(ps) {
		t.Errorf("%d de %d paquetes usan la valvula de escape: eso ya no es una excepcion "+
			"con motivo, es el formato viejo con otro nombre", conValvula, len(ps))
	}

	// Y los bytes: ni un paquete.json puede traer el campo del formato viejo.
	dirs, err := directoriosPublicados("paquetes")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range dirs {
		b, err := os.ReadFile(filepath.Join("paquetes", d, "paquete.json"))
		if err != nil {
			continue
		}
		if strings.Contains(string(b), `"fuente"`) {
			t.Errorf("paquetes/%s/paquete.json todavia declara el campo fuente del formato "+
				"viejo. Hoy el linter lo rechaza; el dia que ese campo salga del tipo, "+
				"encoding/json lo ignoraria en silencio", d)
		}
	}
}

// UNA SOLA FUNCION COMPONE DIRECCIONES DE EDITOR.
//
// Si el anfitrion de EUR-Lex apareciera en tres sitios, cambiarlo seria cambiar
// tres sitios, y el segundo se olvidaria. Esta puerta enumera donde puede
// aparecer y falla con cualquier otro.
//
// LAS DOS EXCEPCIONES, dichas para que consten:
//
//	nucleo/corpus/identificador.go   es LA funcion. Es donde tiene que estar.
//	herramientas/ingestanorma/       es el extractor: su trabajo es hablar con
//	                                 los portales del BOE y de EUR-Lex, asi que
//	                                 conoce sus direcciones por definicion. No
//	                                 deriva la fuente de ningun paquete: escribe
//	                                 el identificador y deja que se derive.
//
// Los ficheros _test.go de la raiz, cmd/ y herramientas/ quedan fuera del
// barrido (ficherosGo ya los salta) porque ahi un test tiene que poder escribir
// la direccion que espera; el que compone en produccion es el que importa.
func TestSoloUnaFuncionComponeDireccionesDeEditor(t *testing.T) {
	permitidos := map[string]bool{
		filepath.FromSlash("nucleo/corpus/identificador.go"): true,
	}
	vistos := map[string]bool{}
	for _, f := range ficherosGo(t) {
		rel := strings.TrimPrefix(filepath.ToSlash(f), "./")
		if strings.HasPrefix(rel, "herramientas/ingestanorma/") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, h := range anfitrionesDeEditor {
			if !strings.Contains(string(b), h) {
				continue
			}
			if permitidos[filepath.FromSlash(rel)] {
				vistos[h] = true
				continue
			}
			t.Errorf("%s escribe el anfitrion %q. La direccion de un editor se compone en "+
				"corpus.Identificador.Enlace y en ningun otro sitio: repartida, cambiarla "+
				"es cambiar varios sitios y el segundo se olvida", rel, h)
		}
	}
	// Control de que la puerta MIRA donde debe: si el fichero de la derivacion
	// dejara de tener los anfitriones, esto pasaria en vacio.
	//
	// Se exigen TODOS y no "alguno", y esa es la mitad que importa: los tests
	// de nucleo/corpus comprueban la derivacion contra sus propias constantes
	// (es lo correcto alli: lo que prueban es que el dato no puede mover el
	// anfitrion, sea cual sea). Con "alguno", cambiar una constante a un sitio
	// ajeno pasaria las dos puertas a la vez. Aqui estan escritos a mano, fuera
	// de ese paquete, que es lo unico que hace de esto un ancla.
	if len(vistos) != len(anfitrionesDeEditor) {
		t.Fatalf("nucleo/corpus/identificador.go escribe %d de los %d anfitriones de "+
			"editor (%v). O la derivacion apunta a otro sitio, o el barrido no llega a "+
			"ese fichero y esta puerta esta pasando en vacio", len(vistos),
			len(anfitrionesDeEditor), vistos)
	}
}
