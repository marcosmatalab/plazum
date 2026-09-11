package plazum

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// EL PARRAFO QUE LEE QUIEN DECIDE SI ESTO ES SERIO TAMBIEN SE ATA AL ARBOL.
//
// # El hallazgo que la trae, y su patron es mas importante que sus cifras
//
// La auditoria del 11-09-2026 conto las cifras publicadas del repositorio y
// encontro esto:
//
//	con puerta      285 hitos, 808 dorados, 266 relojes, 78/144, 54,2 %,
//	                12,1 MB, 25 puertas, 9 dorados de la demo   -> las OCHO exactas
//	sin puerta      el parrafo de ingenieria del README         -> 4 de 5 falsas
//
// **Todo lo que tiene puerta cuadra al digito y casi todo lo que no la tiene esta
// viejo.** No es mala suerte ni descuido: es que la regla de la casa —«todo
// documento que publique un numero lo ata a quien lo computa DEL ARBOL»— se
// habia aplicado al marcador, a la instantanea, al binario y a la cobertura de
// la v1, y NO al parrafo que lee primero quien evalua el proyecto.
//
// Las cuatro, con su medida:
//
//	publicado                  medido
//	1.022 casos de test        2.006
//	~32.000 lineas de codigo   75.187
//	~36.000 de test            105.991
//	9 workflows                12
//
// Las cuatro por lo bajo, que es la direccion de siempre: un numero que envejece
// solo siempre se queda corto, porque el repositorio crece. Y aqui la direccion
// no favorece —vender menos de lo que hay es lo contrario del sesgo habitual—,
// lo que confirma que la causa no es la intencion sino la ausencia de puerta.
//
// # Que se publica y que no, y por que la cobertura sale del parrafo
//
// Se publican las cuatro cifras que se DERIVAN DEL ARBOL con una orden barata y
// determinista. La quinta, «81,3 % de cobertura», no vuelve: una cobertura
// global es una foto que cuesta una ejecucion entera de la suite, asi que
// ninguna puerta barata puede sostenerla y volveria a envejecer igual. En su
// sitio va lo que SI tiene puerta y ademas es mas fuerte: el suelo duro del
// nucleo que CI exige en cada empujon.
//
// # Su limite, dicho
//
// Esto comprueba que los numeros del parrafo son los del arbol. NO comprueba que
// el parrafo diga cosas ciertas de todo lo demas: la prosa que los rodea sigue
// sin puerta, como toda la prosa. Lo que se cierra es la clase que se midio.
func TestElParrafoDeIngenieriaPublicaLoQueDiceElArbol(t *testing.T) {
	bloque := bloqueMarcado(t, "README.md", "ingenieria")

	for _, c := range []struct {
		nombre string
		valor  int
		porQue string
	}{
		{"casos de test escritos", casosDeTestEscritos(t),
			"es el numero que mas envejece, porque sube en cada bloque"},
		{"lineas de produccion", alMillar(lineasGoVersionadas(t, false)),
			"publicado con una tilde delante («~32.000») como si eso lo eximiera de ser cierto"},
		{"lineas de test", alMillar(lineasGoVersionadas(t, true)),
			"idem, y es la mitad que sostiene el argumento de que esto esta probado"},
		{"workflows de CI", workflowsDeCI(t),
			"tres de mas desde que entraron los de vigilancia y calendario"},
	} {
		if c.valor <= 0 {
			t.Fatalf("%s: el derivador devolvio %d, asi que esta puerta no esta midiendo nada",
				c.nombre, c.valor)
		}
		if !strings.Contains(bloque, conPuntoDeMiles(c.valor)) {
			t.Errorf(`el parrafo de ingenieria NO publica %s: el arbol dice %s.

  Por que importa: %s

  Es el primer parrafo de numeros que lee quien decide si esto es serio, y hasta
  hoy era el unico bloque de cifras del README sin puerta. La regla de la casa ya
  existia y se aplicaba al marcador, a la instantanea, al binario y a la cobertura
  de la v1; lo que faltaba era aplicarla aqui.

  Arreglo: poner la cifra que dice el arbol, con su separador de miles, dentro
  del bloque marcado <!-- ingenieria:inicio --> ... <!-- ingenieria:fin -->.`,
				c.nombre, conPuntoDeMiles(c.valor), c.porQue)
		}
	}

	// Y LA COBERTURA NO SE PUBLICA COMO FOTO: se publica el suelo que CI exige,
	// leido de ci.yml. Si alguien baja el suelo, el parrafo deja de cuadrar.
	suelo := sueloDeCoberturaDelNucleo(t)
	if !strings.Contains(bloque, suelo+" %") && !strings.Contains(bloque, suelo+"%") {
		t.Errorf("el parrafo no publica el suelo de cobertura del nucleo (%s %%), que es lo "+
			"que CI exige de verdad en cada empujon.\n"+
			"  Se publica el suelo y no una foto porque una cobertura global cuesta una "+
			"ejecucion entera de la suite: ninguna puerta barata la sostiene, y por eso la "+
			"que habia envejecio igual que las otras cuatro.", suelo)
	}
}

// EL CONTROL NEGATIVO: el detector tiene que ver una cifra que no cuadra.
//
// Sin esto, la puerta de arriba pasaria con un `bloqueMarcado` que devolviera
// siempre el README entero —donde casi cualquier numero aparece en algun
// sitio— o con unos derivadores que devolvieran lo mismo que el texto por
// casualidad. Las dos cosas dan verde.
func TestElDetectorDelParrafoDeIngenieriaAcusaYSeCalla(t *testing.T) {
	// SE CALLA con el bloque que trae la cifra.
	if !strings.Contains("hay **2.006** casos", conPuntoDeMiles(2006)) {
		t.Error("el formateador de miles no produce lo que el README escribe")
	}
	// ACUSA cuando la cifra es otra, aunque se parezca mucho.
	for _, n := range []int{2005, 2007, 20060, 206} {
		if strings.Contains("hay **2.006** casos", conPuntoDeMiles(n)) {
			t.Errorf("el detector daria por bueno %d dentro de un texto que dice 2.006", n)
		}
	}
	// Y EL BLOQUE ES EL BLOQUE, no el fichero entero: si `bloqueMarcado`
	// devolviera todo el README, una cifra escrita en cualquier otro parrafo
	// valdria por la del parrafo de ingenieria.
	entero, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	bloque := bloqueMarcado(t, "README.md", "ingenieria")
	if len(bloque) >= len(entero) {
		t.Errorf("bloqueMarcado devuelve %d bytes y el README tiene %d: no esta acotando nada",
			len(bloque), len(entero))
	}
}

// bloqueMarcado devuelve lo que hay entre <!-- x:inicio --> y <!-- x:fin -->.
func bloqueMarcado(t *testing.T, fichero, marca string) string {
	t.Helper()
	b, err := os.ReadFile(fichero) // #nosec G304 -- fichero del propio repositorio
	if err != nil {
		t.Fatal(err)
	}
	ini, fin := "<!-- "+marca+":inicio -->", "<!-- "+marca+":fin -->"
	s := string(b)
	i, j := strings.Index(s, ini), strings.Index(s, fin)
	if i < 0 || j < 0 || j < i {
		t.Fatalf("%s no tiene el bloque marcado %q. Sin marcas, la puerta no sabe que trozo "+
			"vigila y acabaria dando por bueno cualquier numero escrito en cualquier parrafo",
			fichero, marca)
	}
	return s[i+len(ini) : j]
}

// casosDeTestEscritos cuenta las funciones de test y de fuzzing de los ficheros
// VERSIONADOS. Se usa `git ls-files` y no un recorrido del disco a proposito:
// `.claude/worktrees/` tiene copias del arbol entero, y contarlas daba 67.966.
func casosDeTestEscritos(t *testing.T) int {
	t.Helper()
	n := 0
	re := regexp.MustCompile(`(?m)^func (Test|Fuzz)[A-Za-z0-9_]+`)
	for _, f := range ficherosVersionados(t, "*_test.go") {
		b, err := os.ReadFile(f) // #nosec G304 -- ruta que da git ls-files
		if err != nil {
			t.Fatal(err)
		}
		n += len(re.FindAll(b, -1))
	}
	return n
}

// lineasGoVersionadas cuenta lineas de los ficheros Go versionados, de test o de
// produccion.
func lineasGoVersionadas(t *testing.T, deTest bool) int {
	t.Helper()
	n := 0
	for _, f := range ficherosVersionados(t, "*.go") {
		if strings.HasSuffix(f, "_test.go") != deTest {
			continue
		}
		b, err := os.ReadFile(f) // #nosec G304 -- ruta que da git ls-files
		if err != nil {
			t.Fatal(err)
		}
		n += strings.Count(string(b), "\n")
	}
	return n
}

// workflowsDeCI cuenta los ficheros de workflow versionados.
func workflowsDeCI(t *testing.T) int {
	t.Helper()
	return len(ficherosVersionados(t, ".github/workflows/*.yml"))
}

// sueloDeCoberturaDelNucleo lee de ci.yml el umbral que exige, en vez de
// escribirlo aqui. Dos listas se separan; esta se lee de donde manda.
func sueloDeCoberturaDelNucleo(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(".github/workflows/ci.yml")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`cobertura del nucleo \(puerta dura (\d+)%\)`)
	m := re.FindSubmatch(b)
	if m == nil {
		t.Fatal("ci.yml ya no declara la puerta dura de cobertura del nucleo con el rotulo " +
			"que esta puerta busca: o se renombro el paso, o se quito el suelo, y las dos " +
			"cosas hay que verlas")
	}
	return string(m[1])
}

// ficherosVersionados devuelve lo que git tiene indexado para un patron.
func ficherosVersionados(t *testing.T, patron string) []string {
	t.Helper()
	salida, err := exec.Command("git", "ls-files", "-c", "-o", "--exclude-standard", patron).Output()
	if err != nil {
		t.Skipf("no se puede preguntar a git (%v): esta puerta necesita el indice para no "+
			"contar las copias de .claude/worktrees", err)
	}
	var out []string
	for _, l := range strings.Split(strings.ReplaceAll(string(salida), "\r\n", "\n"), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

// alMillar redondea al millar mas cercano.
//
// # POR QUE LAS LINEAS VAN REDONDEADAS Y LOS CASOS NO
//
// Porque un recuento exacto de lineas cambia con CADA edicion, y una puerta que
// se pone roja en cada commit por un numero que no significa nada a esa
// precision es la que «salta siempre» y entrena a esquivarla: a la tercera vez,
// actualizar el README pasa a ser una tecla y deja de ser una comprobacion.
//
// Redondear NO es volver a la tilde de «~32.000». La diferencia es que aqui la
// precision esta DECLARADA y la puerta la EXIGE: el numero publicado no puede
// desviarse mas de 500 lineas del arbol, mientras que la tilde de antes toleraba
// las 43.000 de diferencia que tenia. Una cifra redondeada y vigilada dice mas
// que una exacta que nadie mira.
//
// Los casos de test y los workflows van exactos porque no churnean: suben de uno
// en uno y solo cuando alguien escribe un test o un workflow.
func alMillar(n int) int {
	return (n + 500) / 1000 * 1000
}

// conPuntoDeMiles escribe un entero como lo escribe el README en castellano.
func conPuntoDeMiles(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	var partes []string
	for len(s) > 3 {
		partes = append([]string{s[len(s)-3:]}, partes...)
		s = s[:len(s)-3]
	}
	return fmt.Sprintf("%s.%s", s, strings.Join(partes, "."))
}
