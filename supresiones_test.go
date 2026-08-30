package plazum

import (
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// LA DIRECTIVA DIRIGIDA A UNA HERRAMIENTA QUE NADIE EJECUTA.
//
// POR QUE EXISTE, con su fecha. El 30-08-2026 CI rechazo un commit por un G304
// de gosec. La anotacion que se habia escrito para callarlo era
// `//nolint:gosec`, que es la directiva de golangci-lint, y golangci-lint no
// corre en este repositorio. O sea: una supresion que no suprimia nada, con
// exactamente el mismo aspecto que una puesta.
//
// El mismo barrido encontro la hermana: una `//nolint:staticcheck` en
// adaptadores/tsa/tsa_test.go, inerte por el mismo motivo, escrita meses antes
// y que nadie habia mirado desde entonces.
//
// ES LA RAMA QUE NUNCA SE EJECUTA, ESCRITA EN UN COMENTARIO. La familia entera
// esta en docs/pendientes.md y su forma es siempre la misma: algo que se lee
// como una guarda y que ningun camino recorre. Aqui es peor que en el codigo,
// porque una directiva ademas AFIRMA que alguien miro el hallazgo y decidio que
// era un falso positivo. Quien la lea dara por revisado lo que no se reviso.
//
// LA REGLA, MECANICA: este arbol solo admite directivas de supresion de
// herramientas que CI EJECUTA. Ni una mas. La lista de las que corren no se
// declara aqui, se LEE de los workflows, por el mismo motivo por el que
// comprobar.sh lee las puertas: una segunda lista es una lista que se queda
// vieja, y este proyecto ya la ha visto quedarse vieja catorce veces.
//
// O la herramienta entra en CI, o la directiva sale. El 30-08-2026 se eligio lo
// primero para staticcheck, y su primera pasada encontro 16 cosas.

// formasDeSupresion son las directivas conocidas del ecosistema de Go, con la
// herramienta que las lee.
//
// La tabla es de FORMAS, no de herramientas permitidas: quien decide si una
// forma vale es herramientasDeCI(), que mira el workflow. Aqui solo se dice
// "esto es una directiva y la lee fulano".
var formasDeSupresion = []struct {
	Prefijo     string
	Herramienta string
	Ejemplo     string
}{
	{"#nosec", "gosec", "// #nosec G304 -- motivo"},
	{"nolint", "golangci-lint", "//nolint:errcheck // motivo"},
	{"lint:file-ignore", "staticcheck", "//lint:file-ignore SA1019 motivo"},
	{"lint:ignore", "staticcheck", "//lint:ignore SA1019 motivo"},
	{"revive:", "revive", "//revive:disable-next-line"},
	{"noinspection", "el IDE de JetBrains", "//noinspection GoUnusedFunction"},
	{"gocyclo:ignore", "gocyclo", "//gocyclo:ignore"},
	{"exhaustive:ignore", "exhaustive", "//exhaustive:ignore"},
}

// supresion es una directiva localizada en el arbol.
type supresion struct {
	Fichero     string
	Linea       int
	Texto       string
	Herramienta string
	Ejemplo     string
}

// clasificarComentario dice si un comentario ES una directiva, y de quien.
//
// EL ANCLAJE ES LO IMPORTANTE Y ES DELIBERADO: se exige que el texto EMPIECE por
// la directiva. Un comentario que HABLA de una directiva no es una directiva, y
// en este arbol hay tres que hablan de ellas (la cabecera de ber.go, dos en
// herramientas/cotejapkcs7). Una version sin anclar los daria por hallazgos, y
// un detector con falsos positivos se acaba desactivando, que es la unica forma
// segura de perder una puerta.
func clasificarComentario(texto string) (supresion, bool) {
	t := strings.TrimSpace(texto)
	t = strings.TrimPrefix(t, "//")
	t = strings.TrimPrefix(t, "/*")
	t = strings.TrimSpace(t)
	for _, f := range formasDeSupresion {
		if strings.HasPrefix(t, f.Prefijo) {
			return supresion{Texto: t, Herramienta: f.Herramienta, Ejemplo: f.Ejemplo}, true
		}
	}
	return supresion{}, false
}

// supresionesDelArbol recorre TODOS los .go del repositorio, los de test
// incluidos. Los de test sobre todo: las dos directivas inertes que motivaron
// esto vivian en ficheros _test.go, que es donde nadie mira.
func supresionesDelArbol(t *testing.T) []supresion {
	t.Helper()
	var out []supresion
	err := filepath.WalkDir(".", func(ruta string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".claude", ".git", "testdata", "corpus_datos":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(ruta, ".go") {
			return nil
		}
		fset := token.NewFileSet()
		a, err := parser.ParseFile(fset, ruta, nil, parser.ParseComments)
		if err != nil {
			return err
		}
		for _, grupo := range a.Comments {
			for _, c := range grupo.List {
				s, ok := clasificarComentario(c.Text)
				if !ok {
					continue
				}
				s.Fichero = filepath.ToSlash(ruta)
				s.Linea = fset.Position(c.Pos()).Line
				out = append(out, s)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// reGoRun localiza las herramientas que CI ejecuta con `go run modulo@version`,
// que es como se invocan todas hoy (govulncheck, gosec, staticcheck) y como
// exige DEPENDENCIAS.md que se invoquen: con la version FIJADA, para que la
// puerta no se ponga roja o verde sin que nadie toque el codigo.
var reGoRun = regexp.MustCompile(`go run ([A-Za-z0-9._~/-]+)@[^\s]+`)

// herramientasDeCI devuelve el nombre corto de cada herramienta que los
// workflows ejecutan de verdad.
//
// SE SALTAN LOS COMENTARIOS DEL YAML a proposito: la cabecera del paso de
// staticcheck cita la directiva inerte que lo trajo, y un detector que leyera
// comentarios se creeria que golangci-lint corre. Es el mismo anclaje que
// arriba, y la primera version de TestNingunWorkflowInvocaGoTest ya se equivoco
// una vez justo asi.
func herramientasDeCI(t *testing.T) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, cuerpo := range workflows(t) {
		for k := range herramientasEn(cuerpo) {
			out[k] = true
		}
	}
	return out
}

// herramientasDeCIYml son solo las del workflow principal, que es de donde
// comprobar.sh las extrae.
//
// LA DISTINCION IMPORTA y hoy no se nota, porque las tres estan en ci.yml. Para
// que una directiva sea legitima basta que CUALQUIER workflow ejecute la
// herramienta; para que el lazo local este completo, lo que tiene que cuadrar es
// lo que corre el workflow del que el script lee. Comparar los dos conjuntos
// entre si daria un rojo que nadie puede arreglar: subir HERRAMIENTAS_ESPERADAS
// y que el script no encuentre esa herramienta donde mira.
func herramientasDeCIYml(t *testing.T) map[string]bool {
	t.Helper()
	cuerpo, ok := workflows(t)["ci.yml"]
	if !ok {
		t.Fatal("no esta .github/workflows/ci.yml, que es de donde comprobar.sh extrae las " +
			"herramientas de seguridad. Si se renombro, hay que decirselo a comprobar.sh y aqui")
	}
	return herramientasEn(cuerpo)
}

func herramientasEn(cuerpo string) map[string]bool {
	out := map[string]bool{}
	for _, linea := range strings.Split(cuerpo, "\n") {
		l := strings.TrimSpace(linea)
		if strings.HasPrefix(l, "#") {
			continue // un comentario que cita el comando no lo ejecuta
		}
		for _, m := range reGoRun.FindAllStringSubmatch(l, -1) {
			out[path.Base(m[1])] = true
		}
	}
	return out
}

// LA PUERTA. Ninguna directiva de supresion apunta a una herramienta ausente.
func TestNingunaDirectivaDeSupresionApuntaAUnaHerramientaQueNoCorre(t *testing.T) {
	corren := herramientasDeCI(t)
	if len(corren) == 0 {
		t.Fatal("no se ha encontrado NI UNA herramienta ejecutada en los workflows.\n" +
			"  O el job de seguridad desaparecio, o la forma de la invocacion cambio y este\n" +
			"  test lleva desde entonces midiendo contra el vacio, que daria por ilegitima\n" +
			"  cualquier directiva y por eso se para aqui en vez de acusar.")
	}
	halladas := supresionesDelArbol(t)
	if len(halladas) == 0 {
		t.Fatal("no se ha encontrado NI UNA directiva de supresion en todo el arbol.\n" +
			"  Habia 58 `#nosec` cuando esto se escribio, asi que o se han ido todas de\n" +
			"  golpe o el detector ha dejado de reconocerlas. Las dos cosas son el mismo\n" +
			"  problema: a partir de ahora esta puerta estaria dando verde sin mirar nada.")
	}
	for _, s := range halladas {
		if corren[s.Herramienta] {
			continue
		}
		t.Errorf("%s:%d tiene una directiva de %s, que CI NO EJECUTA:\n    %s\n"+
			"  Una supresion dirigida a una herramienta ausente no silencia nada y se lee\n"+
			"  igual que una puesta: ademas AFIRMA que alguien miro el hallazgo y decidio\n"+
			"  que era un falso positivo. Quien la lea dara por revisado lo que no se reviso.\n"+
			"  Las que corren hoy: %s.\n"+
			"  Arreglo, y solo hay dos: o la herramienta entra en el job de seguridad de\n"+
			"  .github/workflows/ci.yml (y entonces la directiva es legitima), o la\n"+
			"  directiva sale y el hallazgo se arregla de verdad.\n"+
			"  (Si la herramienta se anadio con un `uses:` en vez de `go run modulo@version`,\n"+
			"  este test no sabe verla todavia y hay que ensenarle.)",
			s.Fichero, s.Linea, s.Herramienta, s.Texto, strings.Join(ordenado(corren), ", "))
	}
	t.Logf("%d directivas de supresion, y las herramientas que corren son: %s",
		len(halladas), strings.Join(ordenado(corren), ", "))
}

// Y CADA DIRECTIVA DICE QUE CALLA Y POR QUE.
//
// Una supresion sin regla apaga TODAS las comprobaciones de esa linea, no la que
// molestaba: el dia que esa linea gane un problema distinto, nadie se enterara.
// Y una supresion sin motivo no es una decision, es un silencio: quien la lea
// dentro de un ano no tiene con que decidir si sigue valiendo.
func TestCadaDirectivaDeSupresionNombraSuReglaYSuMotivo(t *testing.T) {
	// `#nosec` acepta la forma sin regla y sin motivo, asi que la exigencia la
	// pone este test. `lint:ignore` exige la regla por sintaxis propia, pero no
	// que el motivo diga nada, asi que se le mira igual.
	reNosec := regexp.MustCompile(`^#nosec\s+G\d+(,\s*G\d+)*\s+--\s+\S`)
	reLint := regexp.MustCompile(`^lint:(file-)?ignore\s+[A-Z]{1,4}\d+(,[A-Z]{1,4}\d+)*\s+\S`)
	vistas := 0
	for _, s := range supresionesDelArbol(t) {
		var ok bool
		var forma string
		switch {
		case strings.HasPrefix(s.Texto, "#nosec"):
			ok, forma = reNosec.MatchString(s.Texto), "// #nosec G304 -- por que aqui no aplica"
		case strings.HasPrefix(s.Texto, "lint:"):
			ok, forma = reLint.MatchString(s.Texto), "//lint:ignore SA1019 por que aqui no aplica"
		default:
			continue // de otra herramienta: la puerta de arriba ya la rechaza
		}
		vistas++
		if !ok {
			t.Errorf("%s:%d suprime sin nombrar la regla o sin decir el motivo:\n    %s\n"+
				"  Sin regla se callan TODAS las comprobaciones de esa linea, no la que\n"+
				"  molestaba, y el dia que esa linea gane un problema distinto nadie se entera.\n"+
				"  Sin motivo no es una decision, es un silencio.\n"+
				"  Forma: %s", s.Fichero, s.Linea, s.Texto, forma)
		}
	}
	if vistas == 0 {
		t.Fatal("ninguna directiva de gosec ni de staticcheck en todo el arbol: o han " +
			"desaparecido las 58 de golpe, o este test ha dejado de reconocerlas")
	}
	t.Logf("%d directivas de gosec y staticcheck revisadas", vistas)
}

// CONTROL NEGATIVO, en las dos direcciones.
//
// Sin esto no se sabe si el detector vigila o si acompana: un detector que no
// reconoce nada da exactamente el mismo verde que un arbol limpio. Y el reverso
// importa igual, porque el fallo probable de este detector no es dejar pasar una
// directiva, es acusar a la prosa que las menciona.
func TestElDetectorDeSupresionesReconoceLoQueEsYNoLoQueParece(t *testing.T) {
	casos := []struct {
		nombre      string
		texto       string
		herramienta string // "" si no es directiva
	}{
		{"la que costo el rojo", "//nolint:gosec", "golangci-lint"},
		{"la hermana inerte", "//nolint:staticcheck // Subjects basta para contar", "golangci-lint"},
		{"la legitima de gosec", "// #nosec G304 -- la ruta la teclea el operador", "gosec"},
		{"la legitima de staticcheck", "//lint:ignore SA1019 deposito propio", "staticcheck"},
		{"la de fichero entero", "//lint:file-ignore SA1019 todo el fichero", "staticcheck"},
		{"revive", "//revive:disable-next-line", "revive"},
		{"el IDE", "//noinspection GoUnusedExportedFunction", "el IDE de JetBrains"},

		// Y LA PROSA, que es donde este detector se puede equivocar. Los tres
		// primeros estan copiados del arbol tal cual.
		{"prosa de ber.go", "// ber.go va VERBATIM salvo tres comentarios `#nosec G115` sobre el codificador", ""},
		{"prosa del cotejador", "// comentarios `#nosec` que gosec exige, y esos no cambian lo que el parser", ""},
		{"prosa del cotejador 2", "// comentarios, porque un `#nosec` o una cabecera de procedencia no cambian lo", ""},
		{"prosa que empieza por la palabra", "// nolint no es una palabra que aparezca en prosa, pero por si acaso", "golangci-lint"},
		{"comentario normal", "// la ruta la elige el operador en su propia maquina", ""},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			s, ok := clasificarComentario(c.texto)
			if c.herramienta == "" {
				if ok {
					t.Fatalf("se ha tomado por directiva de %s un comentario que solo habla de "+
						"ellas:\n    %s\n  Un detector con falsos positivos se acaba "+
						"desactivando, que es la unica forma segura de perder una puerta",
						s.Herramienta, c.texto)
				}
				return
			}
			if !ok {
				t.Fatalf("no se reconoce como directiva:\n    %s\n  Y si el detector no la ve, "+
					"la puerta da verde con la directiva inerte puesta, que es exactamente el "+
					"caso que la trajo", c.texto)
			}
			if s.Herramienta != c.herramienta {
				t.Fatalf("se atribuye a %s y la lee %s:\n    %s", s.Herramienta, c.herramienta, c.texto)
			}
		})
	}
	// El caso con dientes: la del 30-08-2026 apuntaba a golangci-lint, que no
	// corre. Se comprueba aqui, contra el workflow de verdad, que el veredicto
	// del detector sobre ESA directiva sigue siendo "ilegitima".
	corren := herramientasDeCI(t)
	if corren["golangci-lint"] {
		t.Skip("golangci-lint ha entrado en CI: `//nolint` ya es legitimo y este caso sobra")
	}
	s, _ := clasificarComentario("//nolint:gosec")
	if corren[s.Herramienta] {
		t.Fatalf("el detector cree que %s corre y no corre", s.Herramienta)
	}
}

// LA OTRA MITAD: las que SI corren tienen que estar tambien en el lazo local.
//
// El rojo del 30-08-2026 tuvo dos causas, no una. La directiva inerte fue la
// primera; la segunda fue que `./comprobar.sh` dijo "24 puertas, todas en verde"
// SIN HABER EJECUTADO gosec, porque el paso de seguridad no es una `puerta()` y
// el script solo corria puertas. Un lazo local que no cubre un paso bloqueante
// de CI produce un verde que no significa lo que parece, y ese verde acabo en un
// informe. Sin este test, la proxima herramienta que entre en CI repite el
// agujero entero.
func TestElLazoLocalCorreLasHerramientasDeSeguridadDeCI(t *testing.T) {
	s := comprobar(t)
	corren := herramientasDeCIYml(t)
	if len(corren) == 0 {
		t.Fatal("ninguna herramienta encontrada en ci.yml: ver el test de arriba")
	}
	m := regexp.MustCompile(`(?m)^HERRAMIENTAS_ESPERADAS=(\d+)`).FindStringSubmatch(s)
	if m == nil {
		t.Fatalf("%s no declara HERRAMIENTAS_ESPERADAS. Sin ese numero, el dia que la "+
			"extraccion deje de casar el script correra CERO herramientas y saldra verde, "+
			"que es la misma familia contra la que existe PUERTAS_ESPERADAS", rutaComprobar)
	}
	if m[1] != itoa(len(corren)) {
		t.Errorf("%s declara HERRAMIENTAS_ESPERADAS=%s y en ci.yml hay %d (%s).\n"+
			"  Si CI ha ganado una herramienta de seguridad, el lazo local no la esta\n"+
			"  corriendo y un verde local ya no dice lo que decia. Si la ha perdido, alguien\n"+
			"  ha recortado la vigilancia.\n"+
			"  Arreglo: poner HERRAMIENTAS_ESPERADAS a %d en el mismo commit, y decir por que.",
			rutaComprobar, m[1], len(corren), strings.Join(ordenado(corren), ", "), len(corren))
	}
	// Y NO SE NOMBRAN AQUI. El primer arreglo del 30-08-2026 fue escribir gosec a
	// mano en comprobar.sh, y duro un dia: la siguiente herramienta llego esa
	// misma tarde. La invocacion se LEE de ci.yml o no se lee.
	if !strings.Contains(s, "ci.yml") {
		t.Errorf("%s no lee las herramientas de .github/workflows/ci.yml.\n"+
			"  Si las nombra a mano, es una segunda lista, y una segunda lista es una lista\n"+
			"  que se queda vieja: la primera vez tardo un dia en quedarse vieja.", rutaComprobar)
	}
	for _, linea := range strings.Split(s, "\n") {
		l := strings.TrimSpace(linea)
		if strings.HasPrefix(l, "#") || strings.HasPrefix(l, "echo") {
			continue // los comentarios y la ayuda si pueden citar el comando
		}
		if reGoRun.MatchString(l) {
			t.Errorf("%s invoca una herramienta a mano:\n    %s\n"+
				"  La invocacion tiene que salir de ci.yml. Copiada aqui, el dia que CI "+
				"cambie la version quedaran dos, y la local sera la vieja.", rutaComprobar, l)
		}
	}
}

func ordenado(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	return string(d)
}
