package plazum

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// La auditoria de las puertas de CI, hecha desde dentro del repo.
//
// POR QUE ESTE TEST EXISTE. `go test` sale con codigo 0 en dos situaciones donde
// no ha comprobado nada, y las dos son verdes falsos indistinguibles de un verde
// de verdad:
//
//	go test -run TestQueYaNoSeLlamaAsi ./x   ->  "[no tests to run]", codigo 0
//	go test ./solo/paquetes/sin/tests/...    ->  "[no test files]",   codigo 0
//
// Comprobado en esta maquina, no supuesto. La primera muerde el dia que alguien
// renombra un test; la segunda, el dia que un directorio se mueve. Ninguna avisa.
//
// El arreglo vive en .github/puerta.sh, que cuenta los casos ejecutados y exige
// un minimo. Pero un arreglo que hay que acordarse de usar no es un arreglo, asi
// que este test lee los workflows y exige que NINGUNO invoque `go test`
// directamente: quien quiera ejecutar tests en CI llama a puerta(), que es la
// que sabe contar.
//
// Es la tercera guarda que no guardaba nada en dos semanas. La familia esta en
// docs/pendientes.md.

// exentas son las invocaciones de `go test` de CI que NO pasan por puerta(), con
// el motivo. La lista es corta a proposito: cada entrada es una puerta que hay
// que mirar a mano cuando cambie.
var exentas = map[string]string{
	// De momento ninguna. Si alguna vez hace falta una, va aqui CON SU MOTIVO
	// escrito, y ese motivo es lo que alguien tendra que releer el dia que la
	// puerta se ponga verde sin comprobar nada.
}

func workflows(t *testing.T) map[string]string {
	t.Helper()
	dir := filepath.Join(".github", "workflows")
	entradas, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("no puedo leer %s: %v", dir, err)
	}
	out := map[string]string{}
	for _, e := range entradas {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yml") {
			continue // .disabled fuera: no se ejecuta, no es una puerta
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		out[e.Name()] = string(b)
	}
	if len(out) < 3 {
		t.Fatalf("solo se han encontrado %d workflows activos. Si el directorio se movio, "+
			"este test estaria auditando el vacio y dando verde", len(out))
	}
	return out
}

// La regla es la mas simple que se puede comprobar: NINGUN workflow invoca
// `go test` directamente.
//
// La primera version de este test miraba si el fichero "usaba puerta" buscando
// la palabra suelta, y daba verde en dos workflows que la llevaban solo en un
// comentario ("su puerta de CI"). Falso negativo mio, del mismo tipo que los que
// este test existe para cazar. Ahora se busca la INVOCACION, que no aparece en
// prosa.
func TestNingunWorkflowInvocaGoTestSinContarLosCasos(t *testing.T) {
	vistos := 0
	for nombre, cuerpo := range workflows(t) {
		for _, linea := range strings.Split(cuerpo, "\n") {
			limpia := strings.TrimSpace(linea)
			if strings.HasPrefix(limpia, "#") {
				continue // un comentario que cita el comando no es un paso
			}
			if i := strings.Index(limpia, "#"); i >= 0 {
				limpia = strings.TrimSpace(limpia[:i])
			}
			i := strings.Index(limpia, "go test ")
			if i < 0 {
				continue
			}
			vistos++
			if _, ok := exentas[strings.TrimSpace(limpia[i:])]; ok {
				continue
			}
			t.Errorf("%s invoca go test directamente:\n    %s\n"+
				"  go test sale con CODIGO 0 cuando el patron -run no casa con nada y cuando\n"+
				"  el glob de paquetes no tiene tests, asi que esa linea puede llevar meses en\n"+
				"  verde sin comprobar nada.\n"+
				"  Arreglo: source .github/puerta.sh y puerta \"nombre\" MINIMO <argumentos>,\n"+
				"  que cuenta los casos ejecutados. O anadela a exentas CON SU MOTIVO.",
				nombre, limpia)
		}
	}
	t.Logf("%d invocaciones directas de go test en los workflows", vistos)
}

// Toda puerta declarada tiene un minimo POSITIVO. Un minimo de cero no protege
// de nada y es exactamente el estado del que veniamos.
func TestTodaPuertaDeclaraUnMinimoPositivo(t *testing.T) {
	llamadas := 0
	for nombre, cuerpo := range workflows(t) {
		for _, linea := range strings.Split(cuerpo, "\n") {
			limpia := strings.TrimSpace(linea)
			if !strings.HasPrefix(limpia, "puerta \"") {
				continue
			}
			llamadas++
			// puerta "nombre legible" MINIMO ./paquete/... [flags]
			cierre := strings.Index(limpia[8:], "\"")
			if cierre < 0 {
				t.Errorf("%s: llamada a puerta sin cerrar el nombre: %s", nombre, limpia)
				continue
			}
			resto := strings.Fields(strings.TrimSpace(limpia[8+cierre+1:]))
			if len(resto) < 2 {
				t.Errorf("%s: la llamada %q no trae minimo y paquetes", nombre, limpia)
				continue
			}
			min := resto[0]
			if min == "0" || strings.HasPrefix(min, "-") {
				t.Errorf("%s: la puerta declara minimo %q. Un minimo de cero o negativo no "+
					"protege de nada y es justo el estado del que veniamos: %s", nombre, min, limpia)
			}
			for _, c := range min {
				if c < '0' || c > '9' {
					t.Errorf("%s: el minimo de la puerta no es un numero (%q): %s",
						nombre, min, limpia)
					break
				}
			}
		}
	}
	if llamadas == 0 {
		t.Fatal("no se ha encontrado ni una llamada a puerta() en los workflows. O nadie las " +
			"usa, o este test ha dejado de reconocerlas, y las dos cosas son el mismo problema")
	}
	t.Logf("%d puertas declaradas", llamadas)
}

// El script de la puerta existe y trae lo que dice traer. Sin esto, los tests de
// arriba se contentan con que el workflow MENCIONE puerta.sh.
func TestElScriptDeLaPuertaExisteYCuentaCasos(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(".github", "puerta.sh"))
	if err != nil {
		t.Fatalf("el script de las puertas no esta: %v", err)
	}
	s := string(b)
	for _, quiero := range []string{
		"--- (PASS|FAIL|SKIP)", // cuenta casos, subtests incluidos
		"no tests to run",      // reconoce el patron que no casa
		"no test files",        // reconoce el glob sin tests
		"cerrar_puertas",       // y agrega el veredicto
	} {
		if !strings.Contains(s, quiero) {
			t.Errorf("puerta.sh no contiene %q, asi que no hace lo que estos tests dan por hecho",
				quiero)
		}
	}
	if !strings.Contains(s, `-lt "$minimo"`) {
		t.Error("puerta.sh no compara contra un minimo. Sin minimo solo protege del cero, " +
			"y el caso que de verdad pasa es que alguien borre la mitad de los casos")
	}
}

// Y la puerta de las puertas: que el repo siga teniendo los casos que tiene. Si
// un build tag, un fichero movido o un t.Skip mal puesto se lleva por delante
// media suite, `go test ./...` sigue saliendo verde.
//
// El recuento de aqui es ESTATICO (funciones Test/Fuzz y llamadas a t.Run
// escritas), asi que sale mas bajo que el de CI: una tabla de casos dentro de un
// bucle es un t.Run en el fuente y quince casos en la ejecucion. Hoy: 564
// escritos, 768 ejecutados. Los dos suelos miden cosas distintas y los dos
// sirven; este es el que corre en la maquina de desarrollo sin esperar al CI.
//
// El numero se sube cuando crece, en el mismo commit que lo hace crecer. Es
// deliberadamente incomodo: obliga a notar cuando MENGUA.
//
// 26-08-2026: 640 escritos y 847 ejecutados (medido en Windows), con las puertas
// de distribucion y del descargo dentro. Se sube el suelo a 630.
//
// 26-08-2026, frente de copias y restauracion: 749 escritos, con los 19 casos
// del ensayo de restauracion (herramientas/ensayocopia) dentro. Se sube a 749.
const MinimoDeCasos = 749

func TestElRepoNoPierdeLaMitadDeSuSuiteSinQueNadieLoNote(t *testing.T) {
	n := 0
	err := filepath.WalkDir(".", func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".claude" || d.Name() == "testdata") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(p, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		n += strings.Count(string(b), "\nfunc Test") + strings.Count(string(b), "\nfunc Fuzz")
		n += strings.Count(string(b), "t.Run(")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n < MinimoDeCasos {
		t.Errorf("se han contado %d casos escritos y el suelo son %d. O se ha borrado media "+
			"suite, o el recorrido ha dejado de ver un arbol. Si el recorte es intencionado, "+
			"baja MinimoDeCasos EN ESTE MISMO COMMIT y di por que", n, MinimoDeCasos)
	}
	t.Logf("%d casos escritos (suelo %d)", n, MinimoDeCasos)
}

// --- la sintaxis de los propios pasos de CI --------------------------------

// POR QUE EXISTE. Un paso de CI escrito en bash con un error de sintaxis no
// comprueba nada de lo que dice comprobar: bash ni siquiera llega a ejecutar el
// bloque. En este repo habia uno, en la puerta del TTFV, y llevaba ahi desde que
// se escribio:
//
//	awk '...' doctor.txt || {
//	  echo "PUERTA ROTA: doctor senala un problema sin decir como se arregla."
//	  exit 1
//	fi          <- abre con { y cierra con fi
//
// Ese paso es el que comprueba que `plazum doctor` no senala un problema sin
// decir como se arregla, y que el demo se deshace entero. Ninguna de las dos
// cosas se estaba comprobando. Un rojo de entorno disfrazado de rojo de producto
// sale igual de caro que un verde falso: nadie lo lee dos veces.
//
// La comprobacion es la mas barata que existe, `bash -n`, y no ejecuta nada.
//
// Los otros tests de este fichero miran QUE hacen los workflows. Este mira si lo
// que hacen se puede siquiera parsear.

// bloqueRun es un script de shell de un paso de CI, con de donde salio para que
// el fallo diga la linea del workflow y no la del script extraido.
type bloqueRun struct {
	Workflow string
	Linea    int // linea del `run:` dentro del workflow, empezando en 1
	Script   string
}

// bloquesRun extrae los scripts de los pasos `run:` de un workflow. Se hace a
// mano y sin dependencias (invariante 5) sobre la unica forma de YAML que usan
// estos ficheros: `run: |` con el cuerpo indentado, o `run: <una linea>`.
func bloquesRun(nombre, cuerpo string) []bloqueRun {
	lineas := strings.Split(cuerpo, "\n")
	sangria := func(s string) int { return len(s) - len(strings.TrimLeft(s, " ")) }

	var out []bloqueRun
	for i := 0; i < len(lineas); i++ {
		limpia := strings.TrimSpace(lineas[i])
		limpia = strings.TrimPrefix(limpia, "- ") // `- run:` dentro de la lista de pasos
		if !strings.HasPrefix(limpia, "run:") {
			continue
		}
		valor := strings.TrimSpace(strings.TrimPrefix(limpia, "run:"))
		if valor != "" && !strings.HasPrefix(valor, "|") && !strings.HasPrefix(valor, ">") {
			out = append(out, bloqueRun{nombre, i + 1, valor}) // una sola linea
			continue
		}
		clave := sangria(lineas[i])
		var cuerpoBloque []string
		j := i + 1
		for ; j < len(lineas); j++ {
			if strings.TrimSpace(lineas[j]) == "" {
				cuerpoBloque = append(cuerpoBloque, "")
				continue
			}
			if sangria(lineas[j]) <= clave {
				break
			}
			cuerpoBloque = append(cuerpoBloque, lineas[j])
		}
		// Se quita la sangria del bloque, que es la de su primera linea con
		// texto: dentro puede haber heredocs cuyo contenido depende de ella.
		dentro := 0
		for _, l := range cuerpoBloque {
			if strings.TrimSpace(l) != "" {
				dentro = sangria(l)
				break
			}
		}
		for k, l := range cuerpoBloque {
			if len(l) >= dentro {
				cuerpoBloque[k] = l[dentro:]
			}
		}
		out = append(out, bloqueRun{nombre, i + 1, strings.Join(cuerpoBloque, "\n")})
		i = j - 1
	}
	return out
}

// sinExpresiones sustituye las expresiones de Actions por un token: las sustituye
// el runner antes de que bash vea el script, asi que dejarlas seria pedirle a
// bash que parsee algo que nunca le llega.
func sinExpresiones(s string) string {
	for {
		i := strings.Index(s, "${{")
		if i < 0 {
			return s
		}
		j := strings.Index(s[i:], "}}")
		if j < 0 {
			return s[:i] + "EXPRESION_DE_ACTIONS"
		}
		s = s[:i] + "EXPRESION_DE_ACTIONS" + s[i+j+2:]
	}
}

// bashParsea devuelve el error de sintaxis de un script, o cadena vacia.
func bashParsea(t *testing.T, bash, script string) string {
	t.Helper()
	cmd := exec.Command(bash, "-n") // #nosec G204 -- ruta resuelta con LookPath, sin entrada del usuario
	cmd.Stdin = strings.NewReader(sinExpresiones(script))
	salida, err := cmd.CombinedOutput()
	if err == nil {
		return ""
	}
	if len(salida) == 0 {
		return err.Error()
	}
	return strings.TrimSpace(string(salida))
}

// buscarBash resuelve el interprete, y NO deja pasar el caso de que no este. Un
// t.Skip incondicional aqui seria el mismo verde falso que este fichero existe
// para cazar, asi que solo se salta en la maquina de desarrollo (Windows), donde
// bash puede no estar; en CI, que es donde estos scripts se ejecutan de verdad,
// si falta, falla.
func buscarBash(t *testing.T) string {
	t.Helper()
	bash, err := exec.LookPath("bash")
	if err == nil {
		return bash
	}
	if runtime.GOOS == "windows" {
		t.Skipf("sin bash en esta maquina (%v). Esta puerta se cierra en CI, que corre "+
			"en ubuntu y siempre lo tiene", err)
	}
	t.Fatalf("no hay bash en esta maquina (%v), y los pasos de CI estan escritos en bash. "+
		"Arreglo: instalarlo, no saltarse la comprobacion", err)
	return ""
}

func TestTodoPasoDeCIEsShellQueBashSabeParsear(t *testing.T) {
	bash := buscarBash(t)
	total := 0
	for nombre, cuerpo := range workflows(t) {
		for _, b := range bloquesRun(nombre, cuerpo) {
			total++
			if e := bashParsea(t, bash, b.Script); e != "" {
				t.Errorf("%s:%d el paso `run:` no es shell valido:\n%s\n"+
					"  Un bloque que no parsea NO comprueba nada de lo que dice comprobar:\n"+
					"  bash ni siquiera lo ejecuta. Y su rojo parece un rojo de producto.\n"+
					"  Arreglo: `bash -n` sobre el bloque hasta que calle.",
					b.Workflow, b.Linea, e)
			}
		}
	}
	// Sin suelo, el dia que estos ficheros cambien de forma este test recorreria
	// cero bloques y saldria verde.
	if total < 25 {
		t.Fatalf("solo se han extraido %d bloques `run:` de los workflows. O CI ha "+
			"adelgazado a la mitad, o el extractor ha dejado de reconocerlos, y en los "+
			"dos casos esta puerta estaria parseando el vacio", total)
	}
	t.Logf("%d bloques `run:` parseados", total)
}

// CONTROL NEGATIVO, con la forma exacta del error que vivio en el repo: un
// bloque que abre con { y cierra con fi. Sin esto, el verde de arriba no prueba
// que se este mirando nada.
func TestLaPuertaDeSintaxisDeCICazaUnBloqueRoto(t *testing.T) {
	bash := buscarBash(t)
	const workflowRoto = "name: inventado\n" +
		"jobs:\n" +
		"  x:\n" +
		"    steps:\n" +
		"      - name: un paso con la llave cerrada con fi\n" +
		"        run: |\n" +
		"          set -eu\n" +
		"          grep -q algo fichero || {\n" +
		"            echo \"PUERTA ROTA\"\n" +
		"            exit 1\n" +
		"          fi\n" +
		"      - name: un paso sano al lado\n" +
		"        run: |\n" +
		"          set -eu\n" +
		"          echo bien\n"

	bs := bloquesRun("inventado.yml", workflowRoto)
	if len(bs) != 2 {
		t.Fatalf("el extractor ha sacado %d bloques de un workflow con dos. Si no sabe "+
			"leerlos, su verde sobre los workflows de verdad no significa nada", len(bs))
	}
	if e := bashParsea(t, bash, bs[0].Script); e == "" {
		t.Fatal("bash -n da por bueno un bloque que abre con { y cierra con fi.\n" +
			"  Mientras esto pase, esta puerta no vigila nada: es el error exacto que\n" +
			"  vivio en .github/workflows/etapa2-ttfv.yml sin que nadie lo notara.")
	}
	if e := bashParsea(t, bash, bs[1].Script); e != "" {
		t.Fatalf("bash -n rechaza un bloque sano (%s). Con falsos positivos esta puerta "+
			"se desactiva sola la primera semana", e)
	}
}

// El script de los presupuestos existe y sigue comparando dos veces.
//
// Los tres numeros de ETAPAS.md (binario <25 MB, arranque <3 s, RAM <256 MB) se
// cumplen hoy con tanto margen que ninguno se va a ver fallar por si solo, asi
// que presupuesto() compara cada medida tambien contra un limite imposible y se
// pone rojo si esa pasa. Es lo que hace que la puerta se vea fallar en cada
// ejecucion.
//
// Quitar ese segundo tramo dejaria los seis pasos de CI en verde y nadie lo
// notaria, que es la unica forma silenciosa de romper esto: debilitar la
// comparacion la caza el propio control negativo, en ejecucion.
func TestElScriptDeLosPresupuestosComparaDosVeces(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(".github", "presupuesto.sh"))
	if err != nil {
		t.Fatalf("el script de los presupuestos no esta (%v), y seis pasos de CI lo "+
			"sourcean: .github/workflows/etapa2-ttfv.yml y etapa2-accesibilidad.yml", err)
	}
	s := string(b)
	for _, quiero := range []string{
		"_comparar()",            // una sola comparacion, compartida
		`_comparar "$medida" 0`,  // el control negativo, en cada llamada
		"cerrar_presupuestos",    // y el veredicto, que caza el cero medidas
		"_PRESUPUESTOS_CORRIDOS", // que cuenta cuantas medidas hubo
	} {
		if !strings.Contains(s, quiero) {
			t.Errorf("presupuesto.sh no contiene %q.\n"+
				"  Sin eso, los tres presupuestos de ETAPAS.md se comparan una sola vez y\n"+
				"  contra un limite que hoy sobra por mucho: nunca se veria fallar la\n"+
				"  puerta, y una puerta que nunca se ha visto fallar no es una puerta.",
				quiero)
		}
	}
}

// Una puerta rota tiene que DECIR que ha cazado, y decirlo en el shell de CI.
//
// El fallo que cierra este test, y es el sexto de la familia. GitHub ejecuta los
// pasos `bash` con `bash --noprofile --norc -e -o pipefail`. Con -e puesto, la
// linea `salida=$(go test ...)` de puerta.sh mataba el shell en el acto cuando
// go test fallaba, porque el estado de una asignacion es el del comando
// sustituido. La puerta se ponia roja imprimiendo UNA linea, la del ::group::, y
// nada mas: ni el conteo de casos, ni el mensaje con su arreglo, ni el nombre de
// lo que fallaba.
//
// Se vio en main, en un job de windows-latest, y costo media hora saber que
// habia pasado justamente porque el aparato que existe para explicarlo no
// llegaba a ejecutarse.
//
// La leccion, que es la que este test convierte en puerta: **una puerta se
// demuestra en el shell en el que CORRE, no en el del que la escribe.** Las
// cinco formas de fallo de puerta.sh se demostraron a mano en un shell sin -e.
// Por eso la sexta sobrevivio a la demostracion.
//
// Aqui se invoca a bash CON las mismas banderas que GitHub, contra un objetivo
// que falla seguro, y se exige que la explicacion salga.
func TestUnaPuertaRotaExplicaQueHaCazadoEnElShellDeCI(t *testing.T) {
	bash := buscarBash(t)

	// ./no/existe/... hace fallar a go test sin depender de que ningun test del
	// repositorio este rojo, que es justo lo que no se puede suponer aqui.
	guion := "source .github/puerta.sh; puerta 'prueba de la puerta' 1 ./no/existe/...; cerrar_puertas"

	// Las banderas EXACTAS del shell por defecto de GitHub para `bash`.
	cmd := exec.Command(bash, "--noprofile", "--norc", "-e", "-o", "pipefail", "-c", guion) // #nosec G204 -- ruta resuelta con LookPath y guion literal
	salida, err := cmd.CombinedOutput()

	if err == nil {
		t.Fatal("una puerta contra un objetivo que no existe tiene que salir en rojo, y ha " +
			"salido en verde. Eso es peor que el fallo que este test vigila")
	}

	texto := string(salida)
	for _, quiero := range []string{
		"PUERTA ROTA",
		"prueba de la puerta",
		"puertas rotas",
	} {
		if !strings.Contains(texto, quiero) {
			t.Errorf("la puerta rota no dice %q. Sale en rojo pero sin explicar nada, que es\n"+
				"lo que obliga a adivinar en vez de leer. Causa tipica: el -e que hereda de\n"+
				"GitHub mata el shell en `salida=$(go test ...)` antes de imprimir.\n"+
				"Arreglo: `set +e` en .github/puerta.sh, con su porque escrito al lado.\n"+
				"--- lo que imprimio ---\n%s", quiero, texto)
		}
	}
}

// Ningun fichero de texto del repositorio lleva CRLF.
//
// De donde sale. `TestElDemoPublicadoSaleDeEsteGenerador` fallaba en
// windows-latest con "expediente-demo.json no es lo que sale del escenario", y
// pasaba en la maquina de desarrollo, que tambien es Windows. La diferencia no
// estaba en el codigo sino en `core.autocrlf`: el runner lo trae en `true` y
// convierte a CRLF al hacer checkout, la maquina de desarrollo lo tiene en
// `input`. El generador escribe LF, asi que la comparacion byte a byte comparaba
// dos cosas que se diferenciaban en un byte que nadie habia escrito.
//
// Por que merece una puerta y no solo un `.gitattributes`. Este proyecto compara
// ficheros BYTE A BYTE en cuatro sitios distintos, y todos se rompen a la vez
// con esto, **solo en la maquina de otro**. Ademas `.github/puerta.sh` y
// `.github/presupuesto.sh` se ejecutan con `source` desde bash: un script de
// shell con CRLF no es un script de shell.
//
// Si alguien borra el `.gitattributes`, o lo recorta, esto se pone rojo en la
// primera maquina que clone con la configuracion por defecto de Windows.
func TestNingunFicheroDeTextoLlevaCRLF(t *testing.T) {
	var mirados, saltados int
	err := filepath.WalkDir(".", func(ruta string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// .git y .claude no son del proyecto. El resto si, .github incluido,
			// que es donde viven los scripts que mas duele que lleven CRLF.
			if n := d.Name(); n == ".git" || n == ".claude" {
				return filepath.SkipDir
			}
			return nil
		}
		b, err := os.ReadFile(ruta) // #nosec G304 -- ruta que viene de recorrer el propio repo
		if err != nil {
			return err
		}
		// Binario: se detecta por el NUL, igual que hace git, en vez de por una
		// lista de extensiones que envejece.
		if bytes.IndexByte(b, 0) >= 0 {
			saltados++
			return nil
		}
		mirados++
		if i := bytes.Index(b, []byte("\r\n")); i >= 0 {
			linea := 1 + bytes.Count(b[:i], []byte("\n"))
			t.Errorf("%s:%d lleva CRLF.\n"+
				"  Este repositorio compara ficheros byte a byte y ejecuta scripts de shell\n"+
				"  con source: las dos cosas se rompen con CRLF, y se rompen solo en la\n"+
				"  maquina del que clona con otra configuracion de git.\n"+
				"  Arreglo: comprobar que .gitattributes sigue teniendo `* text=auto eol=lf`\n"+
				"  y renormalizar con `git add --renormalize .`", ruta, linea)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Sin suelo, el dia que este recorrido deje de encontrar ficheros saldria
	// verde sin haber mirado ninguno.
	if mirados < 100 {
		t.Fatalf("solo se han mirado %d ficheros de texto (%d binarios saltados). "+
			"El recorrido ya no recorre el repositorio", mirados, saltados)
	}
	t.Logf("%d ficheros de texto sin CRLF, %d binarios saltados", mirados, saltados)
}
