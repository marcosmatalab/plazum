package plazum

import (
	"bytes"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/ia"
	"github.com/marcosmatalab/plazum/internal/modulo"
)

// EL INVARIANTE 9, VIGILADO: la IA vive en adaptadores y superficies. `nucleo/`
// no conoce el puerto de IA y no lo importa nunca.
//
// POR QUE ESTAS PUERTAS EXISTEN ANTES QUE EL ADAPTADOR. La doctrina de IA
// (`docs/ia.md`) dice que el cumplimiento sigue siendo determinista y que la IA
// solo entra donde hay friccion. Esa frase, sin una puerta, es marketing: se
// escribe el dia del anuncio y se erosiona el martes siguiente, cuando alguien
// necesita "solo una llamadita" desde el motor para desempatar un caso.
//
// Escribirlas ahora, con el adaptador sin construir, tiene un motivo concreto:
// la unica forma de que un invariante aguante es que este puesto ANTES de que
// haya presion para saltarselo.
//
// Y la segunda puerta, la de la suite con la IA apagada, es la que convierte
// "el nucleo es determinista" de eslogan en hecho comprobable por cualquiera en
// dos minutos.

// ficherosGo recorre un subarbol devolviendo los .go, saltando lo que no es
// nuestro.
func ficherosGoDe(t *testing.T, raiz string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(raiz, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".claude", "testdata", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(p, ".go") {
			out = append(out, p)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("no puedo recorrer %s: %v", raiz, err)
	}
	return out
}

// TestElNucleoNoConoceLaIA.
//
// Se comprueban DOS cosas y no una, porque la primera sola no basta:
//
//  1. `nucleo/` no importa `plazum/puertos`. Esto ya lo cubre por transitividad
//     la regla general de arquitectura (el nucleo solo importa
//     `plazum/nucleo/...`), y se comprueba aparte igualmente: una regla general
//     que alguien afloje un dia se lleva por delante este invariante sin que
//     nadie lo relacione. Nombrarlo hace que el rojo diga de que va.
//
//  2. El nucleo NI SIQUIERA NOMBRA la IA. Sin esto, alguien puede copiar el
//     interfaz al nucleo para "no importar puertos" y cumplir la letra
//     rompiendo el fondo. El nucleo no tiene que saber que la IA existe.
func TestElNucleoNoConoceLaIA(t *testing.T) {
	// LA LISTA ES CORTA A PROPOSITO. La primera version incluia "Modelo", y se
	// disparo con `nucleo/pantalla`, que tiene un Modelo DE VISTA: una palabra
	// del dominio de las pantallas que no tiene nada que ver con la IA.
	//
	// Un detector que grita por todo se acaba desactivando, que es peor que no
	// tenerlo. Se quedan solo los nombres que no pueden significar otra cosa.
	// "Propuesta" se queda porque es el nombre EXACTO del unico tipo que la IA
	// puede devolver, y verlo en el nucleo significaria que alguien copio el
	// contrato ahi.
	nombresDeIA := []string{"Asistente", "Propuesta", "LLM", "Ollama"}
	fset := token.NewFileSet()
	mirados := 0

	mod, err := modulo.Ruta()
	if err != nil {
		t.Fatal(err)
	}
	for _, ruta := range ficherosGoDe(t, "nucleo") {
		mirados++
		a, err := parser.ParseFile(fset, ruta, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("no puedo parsear %s: %v", ruta, err)
		}
		for _, imp := range a.Imports {
			v := strings.Trim(imp.Path.Value, `"`)
			if esElPuertoDeIA(v, mod) {
				t.Errorf(`%s importa %s.

  El puerto Asistente vive ahi, y el nucleo no puede conocerlo: si el motor
  puede llamar a la IA, el cumplimiento deja de ser determinista y la promesa
  del producto se cae. Invariante 9 de CLAUDE.md.

  Arreglo: lo que hace falta que el nucleo sepa entra como DATO, calculado
  fuera. Es la misma forma que el instante (invariante 1).`, ruta, v)
			}
		}

		codigo := sinComentarios(t, ruta)
		for _, n := range nombresDeIA {
			if !strings.Contains(codigo, n) {
				continue
			}
			t.Errorf(`%s nombra %q en su codigo.

  El nucleo no tiene que saber que la IA existe. Copiar el interfaz aqui para
  no importar plazum/puertos cumple la letra del invariante 9 y rompe el fondo.

  Si es un falso positivo (una palabra que coincide), cambiale el nombre a esa
  cosa: el coste de renombrar una variable es menor que el de tener una puerta
  que no se puede leer de un vistazo.`, ruta, n)
		}
	}
	// Suelo: si el recorrido deja de encontrar el nucleo, "no lo nombra" se
	// leeria igual que "todo en orden".
	if mirados < 20 {
		t.Fatalf("solo se han mirado %d ficheros de nucleo/. Si el directorio se movio, esta "+
			"puerta estaria auditando el vacio", mirados)
	}
	if !t.Failed() {
		t.Logf("%d ficheros de nucleo/, ninguno conoce la IA", mirados)
	}
}

// esElPuertoDeIA dice si una ruta de import trae el puerto Asistente.
//
// Va en una funcion suya, y no en linea, POR UNA RAZON CONCRETA: esta rama de
// la puerta NO SE PUEDE DEMOSTRAR CON UNA MUTACION. Anadir
// `import _ "plazum/puertos"` a cualquier fichero del nucleo no da un test en
// rojo: da un CICLO DE IMPORTS, porque `plazum/puertos` importa
// `plazum/nucleo/corpus`. El compilador se queja antes que el test.
//
//	package plazum/adaptadores/actualizador
//	    imports plazum/puertos from actualizador.go
//	    imports plazum/nucleo/corpus from puertos.go
//
// Es la trampa que este repositorio tiene escrita ("una mutacion que no compila
// no produce lineas --- FAIL") con una cara nueva: aqui la mutacion no es que
// se me olvidara hacerla compilar, es que NO PUEDE compilar.
//
// La comprobacion se queda igual, por dos motivos. El ciclo existe hoy porque
// `puertos` mira al nucleo; el dia que alguien separe el puerto de IA en un
// paquete que no lo mire, el ciclo desaparece y esto es lo unico que queda. Y
// una puerta que depende de un efecto secundario del grafo de paquetes es una
// puerta que nadie sabe que esta ahi.
//
// Se demuestra sobre un fichero sintetico, en TestElDetectorDeImportDeIAFunciona.
func esElPuertoDeIA(ruta, mod string) bool {
	return ruta == mod+"/puertos" || strings.HasPrefix(ruta, mod+"/puertos/")
}

// CONTROL DEL DETECTOR de la rama que no se puede mutar.
func TestElDetectorDeImportDeIAFunciona(t *testing.T) {
	mod, err := modulo.Ruta()
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	// La fuente sintetica se COMPONE con la ruta leida de go.mod. Escrito el
	// prefijo a mano, el dia que el modulo se renombre este control estaria
	// demostrando que el detector reconoce un import que ya no existe: verde,
	// y sobre otro repositorio.
	fuente := "package x\n\nimport _ \"" + mod + "/puertos\"\n"
	a, err := parser.ParseFile(fset, "sintetico.go", fuente, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	visto := false
	for _, imp := range a.Imports {
		if esElPuertoDeIA(strings.Trim(imp.Path.Value, `"`), mod) {
			visto = true
		}
	}
	if !visto {
		t.Fatalf("el detector no ve un import de %s/puertos. Mientras eso pase, la rama "+
			"de imports de TestElNucleoNoConoceLaIA esta dando verde sin mirar nada", mod)
	}
	// CONTROL NEGATIVO: no grita con lo que si puede importar el nucleo.
	for _, sano := range []string{mod + "/nucleo/ventana", "time", mod + "/nucleo/corpus"} {
		if esElPuertoDeIA(sano, mod) {
			t.Errorf("falso positivo: %q no es el puerto de IA", sano)
		}
	}
}

// sinComentarios devuelve el codigo de un fichero sin sus comentarios.
//
// Hace falta porque este mismo fichero, y varios del nucleo, EXPLICAN en prosa
// por que la IA no entra ahi. Un detector que lea los comentarios caza la
// explicacion del invariante como si fuera su violacion, que es un falso
// positivo que acaba desactivando la puerta. Ya paso una vez, con el detector
// del coste cuadratico de ber2der.
func sinComentarios(t *testing.T, ruta string) string {
	t.Helper()
	fset := token.NewFileSet()
	a, err := parser.ParseFile(fset, ruta, nil, 0) // 0 = sin comentarios
	if err != nil {
		t.Fatalf("no puedo parsear %s: %v", ruta, err)
	}
	var b bytes.Buffer
	if err := printer.Fprint(&b, fset, a); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// TestLaIASoloViveEnAdaptadoresYSuperficies.
//
// La otra mitad del invariante 9: el puerto se declara en `puertos/` y solo lo
// pueden USAR adaptadores y superficies. Ni `cmd/`, ni `herramientas/`, ni el
// nucleo.
//
// `cmd/` esta fuera a proposito y no es celo: es donde vive el CLI, que es el
// camino por el que un auditor verifica un expediente. Una llamada a un modelo
// ahi convertiria `plazum verify` en algo que puede dar veredictos distintos en
// dos ejecuciones.
//
// Y `evals/` ENTRA EN EL RECORRIDO Y NO EN LOS PERMITIDOS, que es lo que se
// anadio el 04-09-2026 al construir el arnes. El conjunto dorado de citas es
// DETERMINISTA a proposito: corre en cada PR, sin red, sin GPU y sin dinero, y
// sigue en verde con PLAZUM_SIN_IA=1. El dia que alguien meta ahi una llamada a
// un modelo, ese eval deja de poder correr en cada PR y nadie lo notaria hasta
// que la factura o el nightly lo dijeran. Los evals que SI necesitan modelo
// llegan despues y van por su propio camino.
func TestLaIASoloViveEnAdaptadoresYSuperficies(t *testing.T) {
	permitidos := []string{"adaptadores/", "superficies/", "puertos/"}
	mirados, usos := 0, 0

	for _, raiz := range []string{"nucleo", "cmd", "herramientas", "adaptadores", "superficies", "puertos", "evals"} {
		if _, err := os.Stat(raiz); err != nil {
			continue
		}
		for _, ruta := range ficherosGoDe(t, raiz) {
			mirados++
			codigo := sinComentarios(t, ruta)
			if !strings.Contains(codigo, "Asistente") {
				continue
			}
			usos++
			normal := filepath.ToSlash(ruta)
			ok := false
			for _, p := range permitidos {
				if strings.HasPrefix(normal, p) {
					ok = true
				}
			}
			if !ok {
				t.Errorf(`%s usa el puerto Asistente, y solo pueden hacerlo %v.

  Invariante 9: la IA vive en adaptadores y superficies. Fuera de ahi, una
  llamada a un modelo mete no-determinismo en un camino que tiene que dar el
  mismo resultado en dos maquinas cualesquiera.`, ruta, permitidos)
			}
		}
	}
	if mirados < 50 {
		t.Fatalf("solo se han mirado %d ficheros: el recorrido no esta encontrando el "+
			"proyecto y esta puerta daria verde sin mirar nada", mirados)
	}
	// Suelo del otro lado: si NADIE nombra el puerto, o se ha borrado o el
	// detector ha dejado de encontrarlo. Hoy lo nombran puertos/ y el doc.go de
	// adaptadores.
	if usos == 0 {
		t.Fatal("nadie nombra el puerto Asistente en todo el repositorio. O se ha borrado " +
			"(entonces borra tambien esta puerta y el invariante 9) o el detector ha dejado " +
			"de encontrarlo, y eso deja el invariante sin vigilancia")
	}
	if !t.Failed() {
		t.Logf("%d ficheros mirados, %d nombran el puerto, todos donde deben", mirados, usos)
	}
}

// TestLaSuiteCorreConLaIADesactivada: existe el paso de CI que lo comprueba.
//
// LO QUE ESTA PUERTA ES HOY, dicho sin adornos: **hoy es casi vacia**, porque
// no hay adaptador de IA que desactivar. Se escribe igual, y por dos motivos
// que no son ceremonia.
//
// El primero: el interruptor tiene que existir ANTES que el adaptador, o el
// adaptador se escribira sin pensar en poder apagarlo. Un producto que no se
// puede ejecutar sin su IA no tiene "modo sin IA": tiene una IA obligatoria con
// una casilla que no hace nada.
//
// El segundo, y es el que importa para el comprador: el dia que la IA exista,
// esta puerta convierte "el nucleo es determinista" de eslogan en hecho
// comprobable en dos minutos por cualquiera que clone el repositorio. Si algun
// test necesita la IA para estar verde, la IA ha entrado en el camino del
// cumplimiento y hay que sacarla.
func TestLaSuiteCorreConLaIADesactivada(t *testing.T) {
	// LA VARIABLE SE LEE DEL CODIGO QUE LA INTERPRETA, no se escribe aqui.
	//
	// Escrita a mano seria una segunda lista, y una segunda lista es una lista
	// que se queda vieja: el dia que alguien renombrara la constante, el paso
	// de CI seguiria exportando el nombre viejo, no apagaria nada, y esta
	// puerta seguiria en verde comprobando que ci.yml menciona una cadena que
	// ya no lee nadie. Un interruptor desconectado con su casilla puesta.
	variable := ia.Variable
	b, err := os.ReadFile(filepath.Join(".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("no puedo leer ci.yml (%v). Si el fichero se movio, esta puerta estaria "+
			"comprobando el vacio", err)
	}
	if !strings.Contains(string(b), variable) {
		t.Errorf(`ci.yml no tiene el paso que corre la suite con %s.

  Invariante 9, segunda puerta: la suite entera tiene que pasar con el
  adaptador de IA desactivado. Si algun test necesita la IA para estar verde,
  la IA ha entrado en el camino del cumplimiento.

  Hoy ese paso es casi vacio, porque no hay adaptador. Se pone antes que el
  adaptador a proposito: un interruptor que se anade despues es un interruptor
  que el adaptador no sabe honrar.`, variable)
	}
	// EL VALOR QUE CI EXPORTA TIENE QUE APAGAR LA IA DE VERDAD.
	//
	// Sin esto, la puerta comprueba que ci.yml MENCIONA la variable, no que la
	// use bien. `PLAZUM_SIN_IA: "0"`, `"yes "` con un espacio o `"desactivada"`
	// mencionan la variable igual y no apagan nada: el paso correria la suite
	// con la IA ENCENDIDA mientras su titulo dice lo contrario, que es
	// exactamente el interruptor con la casilla puesta y el cable cortado.
	//
	// El valor se saca del fichero y se pasa por la MISMA funcion que lo lee en
	// produccion, asi que las dos interpretaciones no pueden separarse.
	m := regexp.MustCompile(regexp.QuoteMeta(variable) + `:\s*"?([^"\s#]+)"?`).
		FindStringSubmatch(string(b))
	if m == nil {
		t.Fatalf("ci.yml nombra %s pero no se le ve un valor asignado. Sin valor, el paso "+
			"no exporta nada y la suite corre con la IA encendida", variable)
	}
	t.Setenv(variable, m[1])
	apagada, err := ia.Apagada()
	if err != nil {
		t.Fatalf(`ci.yml exporta %s=%q y el codigo que lo lee no lo entiende: %v

  Un valor que no se entiende no cae del lado de "apagada": es un error, a
  proposito. Asi que ese paso de CI no correria la suite sin IA, la haria
  fallar entera por otro motivo.`, variable, m[1], err)
	}
	if !apagada {
		t.Errorf(`ci.yml exporta %s=%q, que NO apaga la IA.

  El paso se llama "la suite entera con la IA desactivada" y estaria corriendo
  la suite con la IA encendida. Es la casilla puesta y el cable cortado, que es
  peor que no tener la casilla: quien lea el workflow dara por comprobado lo
  que no se comprueba.

  Arreglo: %s: "1".`, variable, m[1], variable)
	}
	if !t.Failed() {
		t.Logf("ci.yml exporta %s=%q y %s.Apagada() lo lee como apagada", variable, m[1], "ia")
	}
}
