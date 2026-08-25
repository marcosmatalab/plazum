package dutiq

import (
	"os"
	"path/filepath"
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
const MinimoDeCasos = 550

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
