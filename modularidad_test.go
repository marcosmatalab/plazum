package plazum

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/internal/modulo"
)

// QUE NO CREZCA EN SILENCIO. NO QUE NO CREZCA.
//
// # De donde sale esto
//
// De mirar el repositorio con la pregunta de modularidad, el 08-09-2026. Las
// cifras, sin tests y medidas con una orden:
//
//	nucleo/corpus           7.783 lineas   21 ficheros   paquete.go: 2.566
//	cmd/plazum             10.269 lineas   33 ficheros   fan-out: 38
//	superficies/pantallas   4.949 lineas   14 ficheros
//	puertos                   690 lineas    3 ficheros   fan-in: 24
//
// El fan-in de `puertos` esta BIEN y no se toca: para eso existe un puerto.
//
// # Por que `cmd/plazum` es lo que es, y por que eso no es culpa de nadie
//
// La causa esta escrita en el propio repositorio, en `serve_evidencia.go`: *«el
// cable vive en cmd/plazum porque es el unico sitio que conoce a los tres»*. Cada
// vez es la decision correcta. Sumadas dan diez mil lineas y un fan-out de 38, y
// `corpus.go` (866 lineas) y `exportar_alcance.go` (821) ya no son cables.
//
// # El instrumento, y aqui me aparto del encargo con su medida delante
//
// El encargo pedia techo de LINEAS con igualdad exacta en los dos sitios. Antes de
// escribirlo se estimo con que frecuencia saltaria sobre trabajo legitimo, que es
// lo que manda la regla de la casa, y la respuesta descarta la igualdad exacta
// para las lineas: **70 commits tocaron `nucleo/corpus` entre el 20-08 y el
// 08-09**, o sea 3,7 al dia. Una igualdad exacta sobre su recuento de lineas
// saltaria en casi todos, y una puerta que salta siempre entrena a esquivarla.
//
// Asi que se parte en dos instrumentos, cada uno con la forma que le toca:
//
//	LINEAS      -> TECHO. Salta al CRECER por encima de una barra declarada, y
//	               encogerse no rompe. Es lo proporcionado para un numero que se
//	               mueve todos los dias.
//	FAN-OUT     -> IGUALDAD EXACTA, como PUERTAS_ESPERADAS. Cambia solo cuando
//	               alguien mete un import nuevo, y ese es exactamente el momento
//	               en que hay que decidir si eso era un cable o un adaptador que
//	               se quedo en el vestibulo. Medido: el fan-out fue 0, 17, 24, 30,
//	               33, 35 y 39 entre el 26-08 y el 08-09, o sea que salta cada dos
//	               o tres dias. Es frecuente, y es el punto: cada salto lleva una
//	               decision de una linea y un pensamiento.
//
// # Y la decision de 4a, dicha en vez de dejada por inercia
//
// `nucleo/corpus` NO se parte hoy, y el motivo no es que no compense a largo
// plazo: es que partirlo bien son PAQUETES (esquema, linter, cargador) y no
// ficheros, porque dentro de un paquete de Go no hay frontera que separar. Eso es
// un refactor con churn de imports en todo el arbol y riesgo de regresion real, y
// este bloque es de tijera y andamiaje, no de refactor. Lo que si queda es que la
// proxima vez sea una decision: el techo esta puesto rozando el valor de hoy.

const (
	// FanOutDeCmdPlazum son los paquetes DEL MODULO que `cmd/plazum` importa
	// directamente. Con igualdad exacta y en las dos direcciones: que BAJE
	// tambien rompe, porque una capa que adelgaza sin que nadie lo note es la
	// otra mitad del mismo fallo, y ademas seria la buena noticia que nadie
	// contaria.
	// 10-09-2026: 38 -> 43, y la decision que esta puerta pide esta TOMADA y
	// escrita. Los cinco que entran son adaptadores/{ingesta,ia,busqueda,
	// evidencia} y superficies/documentos, o sea la cadena entera de la pieza 3.
	// Son CABLE y no adaptador en el vestibulo, que es la pregunta exacta que
	// hace el mensaje del test: `cmd/plazum` es el unico sitio que conoce a los
	// cinco a la vez, que es literalmente el criterio que dejo escrito
	// serve_evidencia.go. Meter el cable dentro de la superficie habria obligado
	// a que superficies/documentos importara el corpus para componer las
	// consultas, y entonces su contrato dejaria de ser un puerto.
	//
	// 11-09-2026: 43 -> 44, y es UNO: `adaptadores/metadatos`, la extraccion de
	// la ficha de un documento (pieza 7). Misma decision y mismo criterio: es un
	// ADAPTADOR con su sitio propio, y lo que entra en `cmd/plazum` es el CABLE
	// que lo junta con `ia` (para verificar por hash) y con `documentos` (para
	// pintarlo y para guardar quien acepta). El adaptador no conoce a ninguno de
	// los dos: recibe un `ingesta.Documento` y devuelve propuestas.
	FanOutDeCmdPlazum = 44

	// TechoDeNucleoCorpus es la barra de lineas de codigo (sin tests) del
	// paquete. Hoy son 8.045: la barra deja poco margen a proposito, para que la
	// siguiente razon-de-cambio que entre por ahi obligue a decidir si le toca
	// paquete propio.
	//
	// 22-09-2026: 8.000 -> 8.100, y el motivo hay que decirlo entero porque
	// subir un techo es la forma barata de aprobar esta puerta sin arreglar
	// nada. Lo que subio NO es codigo: `paquete.go` se partio por razon de
	// cambio (frontera legal, vocabulario de errores, vigencia, esquema,
	// clasificacion de campos de texto, linter y derivaciones) y cada fichero
	// paga su godoc de cabecera. Son 57 lineas de PROSA que explican el corte,
	// no 57 de logica, y es exactamente lo que este techo existe para obligar a
	// decidir: la respuesta fue "son razones de cambio distintas y salen a
	// fichero propio", no "sube el numero y sigue".
	TechoDeNucleoCorpus = 8100

	// TechoDelFicheroGoMayor es la barra del fichero .go de produccion mas
	// grande del arbol, sea cual sea.
	//
	// # Por que no es un techo sobre un fichero con nombre
	//
	// Porque lo era, y se quedo viejo el dia que hizo su trabajo. Vigilaba
	// `nucleo/corpus/paquete.go`, que eran 2.652 lineas y de verdad era el mayor
	// del repositorio; partido, aquel fichero baja a 190 y el mayor pasa a ser
	// otro, que la puerta no miraba. Una puerta atada a un nombre deja de
	// vigilar en cuanto el problema se muda, y el aviso lo da el silencio:
	// habria seguido verde con un fichero de tres mil lineas al lado.
	//
	// Hoy el mayor es `superficies/pantallas/pantallas.go` con 1.263 lineas. El
	// margen es corto a proposito, igual que lo era el anterior.
	TechoDelFicheroGoMayor = 1300
)

// lineasDeGo cuenta lineas de los .go que NO son tests bajo un directorio.
func lineasDeGo(t *testing.T, dir string) (lineas, ficheros int) {
	t.Helper()
	err := filepath.WalkDir(dir, func(ruta string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(ruta, ".go") || strings.HasSuffix(ruta, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(ruta) // #nosec G304 -- rutas del propio arbol
		if err != nil {
			return err
		}
		lineas += strings.Count(string(b), "\n")
		ficheros++
		return nil
	})
	if err != nil {
		t.Fatalf("recorriendo %s: %v", dir, err)
	}
	if ficheros == 0 {
		t.Fatalf("%s no tiene ni un .go de produccion: este recorrido esta midiendo el "+
			"vacio, no el paquete", dir)
	}
	return lineas, ficheros
}

// importaDelModulo saca los imports DIRECTOS de un paquete que son del propio
// modulo, con `go list`, que es quien lo sabe.
//
// Se pregunta a la cadena de herramientas y no se parsea a mano por el motivo de
// siempre: una segunda implementacion de la misma cifra es como se consigue que
// dos esten de acuerdo y la que mande sea la otra.
func importaDelModulo(t *testing.T, paquete string) []string {
	t.Helper()
	lista, err := ordenGo("list", "-f", "{{range .Imports}}{{.}}\n{{end}}", "./"+paquete)
	if err != nil {
		t.Fatalf("go list sobre %s: %v.\n"+
			"  Sin `go list` no hay fan-out que medir, y adivinarlo parseando los imports a "+
			"mano seria una segunda implementacion de la misma cifra", paquete, err)
	}
	// La ruta del modulo se LEE de go.mod y no se escribe aqui. La primera
	// version de este fichero la cableaba, y `TestNadieCableaLaRutaDelModulo` se
	// puso roja encima al instante: una segunda copia de la ruta no se rompe
	// cuando la primera cambia, se queda vieja y sigue dando verde.
	ruta, err := modulo.Ruta()
	if err != nil {
		t.Fatalf("no puedo leer la ruta del modulo de go.mod: %v", err)
	}
	var out []string
	for _, l := range strings.Split(lista, "\n") {
		l = strings.TrimSpace(l)
		if modulo.EsDeCasa(l, ruta) {
			out = append(out, l)
		}
	}
	sort.Strings(out)
	return out
}

func TestElFanOutDeCmdPlazumNoCreceEnSilencio(t *testing.T) {
	imports := importaDelModulo(t, "cmd/plazum")
	if len(imports) != FanOutDeCmdPlazum {
		t.Errorf("`cmd/plazum` importa %d paquetes del modulo y FanOutDeCmdPlazum dice %d.\n\n"+
			"  ESTA PUERTA NO DICE QUE ESTE MAL. Dice que hay una decision que tomar, y es "+
			"esta: lo que se acaba de cablear ahi, ¿era un CABLE (el unico sitio que conoce a "+
			"los tres) o un ADAPTADOR que se ha quedado en el vestibulo?\n"+
			"  Si era un cable, sube el numero y sigue. Si era un adaptador, tiene sitio "+
			"propio en `adaptadores/` o en `superficies/`.\n"+
			"  Y si el numero ha BAJADO, tambien rompe a proposito: una capa que adelgaza es "+
			"una buena noticia y nadie la contaria si no rompiera.\n\n"+
			"  Los %d de hoy:\n    %s",
			len(imports), FanOutDeCmdPlazum, len(imports), strings.Join(imports, "\n    "))
	}
	t.Logf("cmd/plazum importa %d paquetes del modulo", len(imports))
}

func TestLosPaquetesGrandesTienenTechoYNoSorpresa(t *testing.T) {
	casos := []struct {
		dir   string
		techo int
		por   string
	}{
		{"nucleo/corpus", TechoDeNucleoCorpus,
			"esquema, linter y cargador viven juntos, y por esa misma puerta han entrado " +
				"el identificador por articulo, la transposicion y el bloque de pruebas. Son " +
				"razones de cambio distintas"},
	}
	for _, c := range casos {
		lineas, ficheros := lineasDeGo(t, c.dir)
		if lineas > c.techo {
			t.Errorf("`%s` tiene %d lineas de codigo en %d ficheros y su techo son %d.\n\n"+
				"  Por que hay techo aqui: %s.\n"+
				"  La decision que toca tomar, y que este techo existe para que no se tome "+
				"por inercia: lo que se acaba de meter, ¿es de este paquete o es una razon de "+
				"cambio nueva que merece el suyo?\n"+
				"  Las dos respuestas valen. Lo que no vale es no contestarla: si es de este "+
				"paquete, sube el techo en el mismo commit y di por que; si no lo es, sale.",
				c.dir, lineas, ficheros, c.techo, c.por)
		}
		t.Logf("%s: %d lineas en %d ficheros (techo %d)", c.dir, lineas, ficheros, c.techo)
	}

	// Y EL FICHERO MAYOR DEL ARBOL, quienquiera que sea hoy.
	mayor, lineasDelMayor, mirados := ficheroGoMayor(t)
	if mirados < 100 {
		t.Fatalf("he mirado %d ficheros .go de produccion y hoy son cientos: este recorrido "+
			"esta midiendo el vacio y daria verde con un fichero de tres mil lineas al lado",
			mirados)
	}
	if lineasDelMayor > TechoDelFicheroGoMayor {
		t.Errorf("`%s` tiene %d lineas y el techo del fichero mayor son %d.\n\n"+
			"  La decision que toca, y que este techo existe para que no se tome por "+
			"inercia: lo que ha crecido ahi dentro, ¿es una razon de cambio o son varias?\n"+
			"  Si son varias, salen a fichero propio, que es lo que se hizo el 22-09-2026 "+
			"con nucleo/corpus/paquete.go: 2.652 lineas repartidas en siete.\n"+
			"  Si de verdad es una sola, sube el techo EN EL MISMO COMMIT y di por que.",
			mayor, lineasDelMayor, TechoDelFicheroGoMayor)
	}
	t.Logf("el fichero .go de produccion mayor del arbol es %s con %d lineas (techo %d), "+
		"de %d mirados", mayor, lineasDelMayor, TechoDelFicheroGoMayor, mirados)
}

// ficheroGoMayor devuelve el .go de produccion con mas lineas del arbol de git.
//
// Recorre `git ls-files` y no el disco porque el disco trae los worktrees de
// de herramientas locales bajo `.claude/`, que son copias del arbol entero:
// mirarlos daria dos
// veces el mismo fichero y, peor, uno de OTRO commit podria ganar y poner roja
// esta puerta por algo que no esta en este arbol.
//
// Y descarta los `_test.go` por la misma razon que el contador de paquete: en
// este repositorio los tests son mas lineas que el codigo, asi que contarlos
// convertiria esta puerta en un castigo por escribir tests.
func ficheroGoMayor(t *testing.T) (ruta string, lineas, mirados int) {
	t.Helper()
	for _, f := range ficherosVersionados(t, "*.go") {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f) // #nosec G304 -- ruta que da git ls-files
		if err != nil {
			continue
		}
		mirados++
		if n := strings.Count(string(b), "\n"); n > lineas {
			ruta, lineas = f, n
		}
	}
	return ruta, lineas, mirados
}

// CONTROL NEGATIVO DEL CONTADOR.
//
// Su fallo probable es contar tests, que en este repositorio son mas lineas que
// el codigo (`cmd/plazum` tiene 10.269 de produccion y 11.400 de test). Un
// contador que los mezclara daria una cifra que sube cuando alguien escribe un
// test, o sea castigaria justo lo que se quiere premiar.
func TestElContadorDeLineasNoCuentaTests(t *testing.T) {
	dir := t.TempDir()
	escribir := func(nombre, cuerpo string) {
		if err := os.WriteFile(filepath.Join(dir, nombre), []byte(cuerpo), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	escribir("a.go", "package x\n\nfunc A() {}\n")
	escribir("a_test.go", "package x\n\nimport \"testing\"\n\nfunc TestA(t *testing.T) {}\n")
	escribir("leeme.md", "no es go\n")
	lineas, ficheros := lineasDeGo(t, dir)
	if ficheros != 1 {
		t.Errorf("ha contado %d ficheros y solo uno es codigo de produccion", ficheros)
	}
	if lineas != 3 {
		t.Errorf("ha contado %d lineas y el unico fichero de produccion tiene 3", lineas)
	}
}

// ordenGo corre la cadena de herramientas de Go y devuelve su salida.
//
// El error se devuelve y no se convierte en fallo: quien llama decide si «go no
// contesta» es rojo o es «no se pudo mirar», que no son lo mismo.
func ordenGo(args ...string) (string, error) {
	salida, err := exec.Command("go", args...).Output()
	return strings.TrimSpace(string(salida)), err
}
