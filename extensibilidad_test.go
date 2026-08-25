package dutiq

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// urnsProhibidos son identificadores de norma concretos. Si alguno aparece en el
// codigo, es que la norma esta cableada, y entonces "anadir la norma 31 no toca
// codigo" es falso.
//
// Esta es la propiedad que separa este diseno de un GRC cuyo corpus es un arbol
// plano de requisitos cargado desde una hoja de Excel: alli la temporalidad, las
// preguntas de alcance y los entregables no tienen donde declararse, asi que
// acaban en el codigo o no existen.
var urnsProhibidos = []string{
	"ens@", "rgpd@", "nis2@", "dora@", "cra@", "iso27001@", "iso22301@",
	"iso42001@", "pcidss@", "soc2@", "tisax@", "cis@", "nist80053@", "csf@",
	"lopdgdd@", "eidas@", "mica@", "psd2@", "dga@", "dataact@", "aiact@",
	// y el esquema URN real de los paquetes (el que usa paquetes/):
	"urn:es:", "urn:eu:", "urn:iso:",
}

// idsDeObligacion son los mismos marcos con el separador que usan los
// identificadores de sus obligaciones: ens.art31.auditoria, cra.art14.alerta.
//
// HALLAZGO del frente de expediente: la lista de arriba solo caza los URN de
// PAQUETE, los que llevan arroba o el esquema urn:. Un identificador de
// OBLIGACION no contiene ninguno de los dos, asi que se podia cablear media
// norma sin que nadie dijera nada.
//
// Van en una lista aparte porque NO se buscan igual. Estos si necesitan
// frontera por la derecha, y los de arriba no. Ver coincide.
var idsDeObligacion = []string{
	"ens.", "rgpd.", "nis2.", "dora.", "cra.", "iso27001.", "iso22301.",
	"iso42001.", "pcidss.", "soc2.", "tisax.", "cis.", "nist80053.", "csf.",
	"lopdgdd.", "eidas.", "mica.", "psd2.", "dga.", "dataact.", "aiact.",
}

// coincide busca el identificador. La frontera que aplica depende de la familia,
// y la diferencia no es cosmetica.
//
// Por la IZQUIERDA siempre, porque sin ella "ens." convierte en hallazgo
// cualquier literal con "tokens." u "ordenes." dentro, y un detector que grita
// por todo se acaba desactivando, que es peor que no tenerlo.
//
// Por la DERECHA solo para los identificadores de obligacion, y esta la aprendi
// con un falso positivo propio: un mensaje de error que dice "el ENS, el RGPD y
// el CRA." se cazaba como norma cableada. Ahi "CRA." es prosa con punto final;
// un identificador continua (cra.art14, cis.1.1), asi que detras tiene que venir
// letra o digito.
//
// Y por que los URN de paquete NO llevan frontera derecha: porque exigirsela
// abre una evasion trivial. "ens@" + "2022.311" son dos literales que terminan
// justo en el prefijo, y con frontera derecha ninguno de los dos casa. Con los
// URN de paquete no hace falta el compromiso, porque "ens@" y "urn:es:" no
// aparecen jamas en prosa.
//
// Lo que se escapa, dicho para que conste: "ens." + "art31" evade la deteccion
// de identificador de obligacion. Se acepta, porque cerrarlo devuelve los falsos
// positivos, y porque cablear una norma de verdad necesita ademas su URN de
// paquete, que esa si se caza partida.
func alfanumerico(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func coincide(literal, ident string, exigirDerecha bool) bool {
	desde := 0
	for {
		i := strings.Index(literal[desde:], ident)
		if i < 0 {
			return false
		}
		i += desde
		fin := i + len(ident)
		izquierdaLibre := i == 0 || !alfanumerico(literal[i-1])
		derechaOK := !exigirDerecha || (fin < len(literal) && alfanumerico(literal[fin]))
		if izquierdaLibre && derechaOK {
			return true
		}
		desde = i + 1
	}
}

// arbolesConTestsVigilados son los arboles donde los ficheros _test.go tambien
// tienen prohibido cablear una norma.
//
// POR QUE NO ES TODO EL REPO. La raiz, cmd/ y herramientas/ son donde el codigo
// se ENCUENTRA con paquetes/: ahi un test que carga el corpus y comprueba que el
// ENS deriva la auditoria del art. 31 tiene que poder nombrar el ENS, porque de
// eso va el test. nucleo/ y adaptadores/ no: son autonomos por diseno, no
// conocen el directorio del corpus, y un identificador de norma ahi solo puede
// ser un escenario cableado a mano.
//
// POR QUE HIZO FALTA AMPLIARLO. Este detector excluia TODOS los _test.go, y por
// eso llevaba meses en verde con ocho ficheros de test de nucleo/ llenos de
// ens@, rgpd@, cra@ e iso27001@. Peor: las reglas de aplicabilidad del ENS
// vivian en un progENS escrito en Go dentro de nucleo/expediente/expediente_test.go.
// El invariante estaba escrito, el test estaba en verde, y el invariante era
// falso. Es exactamente el fallo del que este proyecto se defiende: un test que
// no mira donde esta el problema no es una puerta, es un adorno.
var arbolesConTestsVigilados = []string{"nucleo", "adaptadores"}

func testVigilado(ruta string) bool {
	limpia := filepath.ToSlash(ruta)
	for _, a := range arbolesConTestsVigilados {
		if limpia == a || strings.HasPrefix(limpia, a+"/") {
			return true
		}
	}
	return false
}

func ficherosGoEn(t *testing.T, raiz string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(raiz, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == "corpus_datos" || d.Name() == "testdata" || d.Name() == ".claude") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(p, ".go") {
			return nil
		}
		rel, err := filepath.Rel(raiz, p)
		if err != nil {
			return err
		}
		if strings.HasSuffix(p, "_test.go") && !testVigilado(rel) {
			return nil
		}
		out = append(out, p)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func ficherosGo(t *testing.T) []string {
	t.Helper()
	return ficherosGoEn(t, ".")
}

// normasCableadas devuelve las coincidencias de identificadores prohibidos en
// un fichero ya parseado. Extraida para poder demostrar con un control negativo
// que el detector salta cuando debe.
func normasCableadas(fset *token.FileSet, a *ast.File) []string {
	var hallazgos []string
	ast.Inspect(a, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		v := strings.ToLower(lit.Value)
		for _, u := range append(append([]string{}, urnsProhibidos...), idsDeObligacion...) {
			if coincide(v, u, strings.HasSuffix(u, ".")) {
				// Con la linea: un hallazgo que solo dice "ens@" en un
				// fichero de 500 lineas obliga a buscarlo a mano, y un
				// error que no es accionable acaba ignorado.
				pos := fset.Position(lit.Pos())
				hallazgos = append(hallazgos,
					fmt.Sprintf("%s (linea %d)", u, pos.Line))
			}
		}
		return true
	})
	return hallazgos
}

// TestNingunaNormaCableada es el test que hace verificable la extensibilidad.
// Falla el build si alguien mete el identificador de una norma en el codigo.
func TestNingunaNormaCableada(t *testing.T) {
	fset := token.NewFileSet()
	for _, f := range ficherosGo(t) {
		a, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for _, u := range normasCableadas(fset, a) {
			t.Errorf("%s: la norma %q esta cableada en el codigo. "+
				"Toda norma vive en su paquete de datos o no vive", f, u)
		}
	}
}

// TestElDetectorSaltaCuandoDebe es el control negativo del detector: un fuente
// sintetico con una norma cableada (por el esquema viejo y por el URN real)
// tiene que producir hallazgos. Sin esto, un verde no demuestra nada.
func TestElDetectorSaltaCuandoDebe(t *testing.T) {
	fset := token.NewFileSet()
	fuente := `package x
var a = "ens@" + "2022.311"
var b = "urn:es:" + "rd:2022:311"`
	a, err := parser.ParseFile(fset, "sintetico.go", fuente, 0)
	if err != nil {
		t.Fatal(err)
	}
	h := normasCableadas(fset, a)
	// Conjunto exacto, no cuenta. Un detector que devolviera "urn:es:" dos
	// veces y se dejara "ens@" tambien da 2, y pasaria.
	quiero := map[string]bool{"ens@": true, "urn:es:": true}
	visto := map[string]bool{}
	for _, x := range h {
		visto[strings.SplitN(x, " ", 2)[0]] = true
	}
	if len(h) != 2 || len(visto) != 2 {
		t.Fatalf("el detector debia encontrar 2 normas cableadas DISTINTAS y encontro %d (%d "+
			"distintas): %v", len(h), len(visto), h)
	}
	for u := range quiero {
		if !visto[u] {
			t.Errorf("el detector no encontro %q: %v", u, h)
		}
	}
	if !strings.Contains(h[0], "linea 2") {
		t.Errorf("el hallazgo tiene que decir fichero y linea para ser accionable, y dijo %q", h[0])
	}
}

// El control negativo de LA SELECCION DE FICHEROS, que es la mitad que estaba
// rota. El detector en si funcionaba; lo que fallaba es que no se le daban los
// ficheros donde estaba el problema.
//
// Se monta un arbol sintetico con la misma norma cableada en cuatro sitios y se
// comprueba que se miran dos y no se miran los otros dos. Si alguien vuelve a
// excluir los _test.go de nucleo/, o los incluye en la raiz, este test lo dice.
func TestElDetectorMiraLosTestsDeNucleoYAdaptadoresYNoLosDeLaRaiz(t *testing.T) {
	dir := t.TempDir()
	escribir := func(ruta string) {
		completa := filepath.Join(dir, filepath.FromSlash(ruta))
		if err := os.MkdirAll(filepath.Dir(completa), 0o755); err != nil {
			t.Fatal(err)
		}
		fuente := "package x\n\nvar v = \"ens@2022.311\"\n"
		if err := os.WriteFile(completa, []byte(fuente), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	vigilados := []string{
		"nucleo/algo/algo_test.go",
		"nucleo/raiz_test.go",
		"adaptadores/tsa/tsa_test.go",
		"nucleo/algo/produccion.go",
		"cmd/dutiq/main.go",
	}
	libres := []string{
		"raiz_test.go",
		"cmd/dutiq/main_test.go",
		"herramientas/x/x_test.go",
		"nucleo/algo/testdata/plantilla_test.go",
	}
	for _, r := range append(append([]string{}, vigilados...), libres...) {
		escribir(r)
	}

	visto := map[string]bool{}
	for _, f := range ficherosGoEn(t, dir) {
		rel, err := filepath.Rel(dir, f)
		if err != nil {
			t.Fatal(err)
		}
		visto[filepath.ToSlash(rel)] = true
	}
	for _, r := range vigilados {
		if !visto[r] {
			t.Errorf("%s tenia que estar vigilado y no se mira. Es justo el agujero por el que "+
				"progENS vivio en nucleo/expediente durante meses", r)
		}
	}
	for _, r := range libres {
		if visto[r] {
			t.Errorf("%s NO tenia que estar vigilado: ahi es donde el codigo se encuentra con "+
				"paquetes/ y un test tiene que poder nombrar la norma que carga", r)
		}
	}
}

// La frontera por la izquierda, que es lo que hace usable la lista ampliada.
//
// Anadir "ens." y "cis." a los identificadores prohibidos convierte en hallazgo
// cualquier literal con "tokens." o "ordenes." dentro si se busca por subcadena
// a secas, y un detector que grita por todo se acaba desactivando, que es peor
// que no tenerlo. Aqui se fija que caza lo que tiene que cazar y calla en lo
// que no.
func TestElDetectorDistingueLaNormaDeUnaPalabraQueLaContiene(t *testing.T) {
	caza := []string{
		`"ens.art31.auditoria"`,
		`"ens@2022.311"`,
		`"urn:es:rd:2022:311"`,
		`"la obligacion cis.1.1 dice"`,
		`"prefijo/cra.art14.alerta"`,
	}
	calla := []string{
		`"tokens.json"`,
		// Prosa, no identificador. Falso positivo propio, cazado a tiempo.
		`"esto vale para el ENS, el RGPD y el CRA."`,
		`"lo dice el ENS."`,
		`"ordenes.pendientes"`,
		`"resumens.txt"`,
		`"incidencis.log"`,
		`"demo.auditoria_bienal"`,
	}
	for _, lit := range caza {
		var alguno bool
		for _, u := range append(append([]string{}, urnsProhibidos...), idsDeObligacion...) {
			if coincide(strings.ToLower(lit), u, strings.HasSuffix(u, ".")) {
				alguno = true
			}
		}
		if !alguno {
			t.Errorf("%s tenia que cazarse y no se caza", lit)
		}
	}
	for _, lit := range calla {
		for _, u := range append(append([]string{}, urnsProhibidos...), idsDeObligacion...) {
			if coincide(strings.ToLower(lit), u, strings.HasSuffix(u, ".")) {
				t.Errorf("%s NO es una norma cableada y se caza por %q. Un detector que "+
					"grita por todo se acaba desactivando", lit, u)
			}
		}
	}
}

// TestNormaNuevaNoTocaCodigo comprueba la propiedad de frente: se anade un
// paquete que el codigo no ha visto nunca, escrito solo como datos, y la
// interfaz, la entrevista, los entregables y la lista de conectores cambian
// solas. Cero ficheros Go tocados.
func TestNormaNuevaNoTocaCodigo(t *testing.T) {
	dir := t.TempDir()
	escribir := func(nombre, json string) {
		d := filepath.Join(dir, nombre)
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "paquete.json"), []byte(json), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	escribir("norma-a", paqueteDemo("urn:demo:a", "sistema", "categoria"))

	antes := medirDerivados(t, dir)

	// La norma 31: nunca vista por el codigo, solo datos, y ademas introduce un
	// tipo de entidad y un recurso que no existian.
	escribir("norma-31", paqueteDemo("urn:demo:31", "proveedor", "criticidad"))

	despues := medirDerivados(t, dir)

	if despues.campos <= antes.campos {
		t.Errorf("la interfaz no ha crecido: %d -> %d", antes.campos, despues.campos)
	}
	if despues.preguntas <= antes.preguntas {
		t.Errorf("la entrevista no ha crecido: %d -> %d", antes.preguntas, despues.preguntas)
	}
	if despues.trazas <= antes.trazas {
		t.Errorf("los entregables no han crecido: %d -> %d", antes.trazas, despues.trazas)
	}
	if despues.recursos <= antes.recursos {
		t.Errorf("los conectores necesarios no han crecido: %d -> %d", antes.recursos, despues.recursos)
	}
}
