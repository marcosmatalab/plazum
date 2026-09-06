package plazum

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ESCRIBIR EL MOTIVO NO ES LO MISMO QUE PONERLE PUERTA.
//
// # De donde sale esta puerta
//
// El 06-09-2026 sobrevivio una mutacion (M12) que dejaba encendido el bloque de
// la consecuencia sin calculadora, o sea que la pantalla afirmaba en las
// diecinueve preguntas de la entrevista que contestar que si NO ACTIVA NADA: una
// afirmacion sobre el cumplimiento de alguien hecha por un campo sin rellenar.
//
// Lo que hace de ese fallo una FAMILIA y no un descuido es que el godoc del
// puerto `pantallas.Consecuencias` ya decia exactamente eso, con esas palabras,
// escrito por el mismo autor y en el mismo commit. El peligro estaba enunciado.
// Lo que no habia era nadie mirando. Un comentario que avisa y ninguna puerta
// que vigile es la afirmacion acompanada escrita por el propio autor: quien lo
// lea dara por revisado lo que no se reviso, y con mas motivo que en cualquier
// otra de sus formas, porque el aviso demuestra que alguien lo penso.
//
// # Que exige, exactamente
//
// Dos mitades, y la frontera entre las dos esta puesta donde esta a proposito.
//
//   - MITAD A, el conjunto acotado y barrido entero: todo INTERFAZ EXPORTADO
//     cuyo godoc (o los comentarios de sus metodos) enuncie un peligro con el
//     vocabulario de abajo tiene que decir quien lo vigila. Son las fronteras de
//     confianza del producto y son pocas (el test imprime cuantas), o sea que la
//     puerta salta sobre trabajo legitimo casi nunca: solo al declarar un interfaz
//     nuevo que enuncie un peligro, que es exactamente el momento en que hay que
//     contestar la pregunta.
//   - MITAD B, la marca, para lo que venga en cualquier otro sitio: un bloque de
//     comentario que lleve una linea `PELIGRO:` tiene que llevar tambien su
//     linea de vigilancia. Es la via para marcar a mano lo que el vocabulario no
//     caza.
//
// Y en las dos mitades, la vigilancia se escribe de una de estas dos formas:
//
//	// LO VIGILA: TestQueSea, TestOtro
//	// NADIE LO VIGILA: el motivo, dicho.
//
// LOS TESTS QUE SE NOMBRAN TIENEN QUE EXISTIR. Sin esa comprobacion la regla se
// cumpliria escribiendo un nombre plausible, que es la misma familia otra vez y
// en su forma mas barata: un identificador con la FORMA de lo verificable es
// justo lo que hace que nadie vaya a verificarlo.
//
// # SU LIMITE, DICHO CON SU CARDINAL
//
// La mitad A mira los interfaces exportados del arbol. El resto del arbol tiene
// muchas mas lineas de comentario que enuncian un peligro con este mismo
// vocabulario, y esta puerta NO LAS MIRA. El cardinal no se escribe aqui: lo
// DERIVA y lo imprime TestElHuecoQueEstaPuertaNoMiraSeCuenta, porque un numero
// escrito a mano en un comentario es la mitad que caduca sin que nadie la
// vigile. No es un
// descuido y no se arregla subiendo el alcance: exigirlas todas pondria roja
// esta puerta en casi cualquier commit que toque un comentario, y una puerta que
// salta siempre entrena a esquivarla. Lo que cierra ese hueco no es la puerta,
// es la marca `PELIGRO:` puesta a mano donde el peligro sea real.
var vocabularioDePeligro = []string{
	"es peor que",
	"seria peor",
	"se leeria como",
	"no protege de nada",
	"acusar en falso",
	"es un incumplimiento",
	"es una fuga",
}

var (
	reInterfazExportado = regexp.MustCompile(`^type ([A-Z][A-Za-z0-9]*) interface \{`)
	reVigilancia        = regexp.MustCompile(`^\s*//\s*(NADIE LO VIGILA|LO VIGILA)\s*:\s*(\S.*)$`)
	rePeligro           = regexp.MustCompile(`^\s*//\s*PELIGRO\s*:`)
	reNombreDeTest      = regexp.MustCompile(`\bTest[A-Za-z0-9_]+`)
)

// bloqueVigilado dice si un trozo de comentario declara quien lo vigila, y con
// que nombres. El segundo valor son los tests nombrados, para poder comprobar
// aparte que existen.
func bloqueVigilado(lineas []string) (bool, []string) {
	var nombres []string
	vigilado := false
	for _, l := range lineas {
		m := reVigilancia.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		vigilado = true
		if m[1] == "LO VIGILA" {
			nombres = append(nombres, reNombreDeTest.FindAllString(m[2], -1)...)
		}
	}
	return vigilado, nombres
}

func enuncianPeligro(lineas []string) []string {
	t := strings.ToLower(strings.Join(lineas, " "))
	var hallazgos []string
	for _, v := range vocabularioDePeligro {
		if strings.Contains(t, v) {
			hallazgos = append(hallazgos, v)
		}
	}
	return hallazgos
}

// interfazExportado es una frontera de confianza con su comentario alrededor:
// el godoc de arriba MAS los comentarios de dentro, porque el aviso puede estar
// en cualquiera de los dos y en `Consecuencias` estaba en los dos.
type interfazExportado struct {
	fichero string
	linea   int
	nombre  string
	texto   []string
}

func interfacesExportados(t *testing.T) []interfazExportado {
	t.Helper()
	var out []interfazExportado
	for _, f := range ficherosGo(t) {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f) // #nosec G304 -- ruta de la propia enumeracion del arbol
		if err != nil {
			t.Fatal(err)
		}
		ls := strings.Split(string(b), "\n")
		for i, l := range ls {
			m := reInterfazExportado.FindStringSubmatch(l)
			if m == nil {
				continue
			}
			var texto []string
			for j := i - 1; j >= 0 && strings.HasPrefix(strings.TrimSpace(ls[j]), "//"); j-- {
				texto = append(texto, ls[j])
			}
			for k := i + 1; k < len(ls) && !strings.HasPrefix(ls[k], "}"); k++ {
				texto = append(texto, ls[k])
			}
			out = append(out, interfazExportado{
				fichero: filepath.ToSlash(f), linea: i + 1, nombre: m[1], texto: texto,
			})
		}
	}
	if len(out) == 0 {
		t.Fatal("no se encontro ni un interfaz exportado: el detector no esta mirando nada")
	}
	return out
}

// testsDelArbol son los nombres de test que existen de verdad, para que
// `LO VIGILA:` no se pueda cumplir con un nombre plausible.
func testsDelArbol(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	re := regexp.MustCompile(`(?m)^func (Test[A-Za-z0-9_]+)\(`)
	err := filepath.WalkDir(".", func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".claude" || d.Name() == "testdata" || d.Name() == "corpus_datos") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(p, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(p) // #nosec G304 -- ruta de la propia enumeracion del arbol
		if err != nil {
			return err
		}
		for _, m := range re.FindAllStringSubmatch(string(b), -1) {
			out[m[1]] = filepath.ToSlash(p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < 500 {
		t.Fatalf("solo %d tests encontrados: el enumerador no esta mirando el arbol", len(out))
	}
	return out
}

// TestTodoPeligroEnunciadoEnUnaFronteraDiceQuienLoVigila es la mitad A.
func TestTodoPeligroEnunciadoEnUnaFronteraDiceQuienLoVigila(t *testing.T) {
	existen := testsDelArbol(t)
	fronteras := interfacesExportados(t)
	conPeligro := 0
	for _, i := range fronteras {
		hallazgos := enuncianPeligro(i.texto)
		if len(hallazgos) == 0 {
			continue
		}
		conPeligro++
		vigilado, nombres := bloqueVigilado(i.texto)
		if !vigilado {
			t.Errorf("%s:%d el godoc de %s enuncia un peligro (%v) y no dice quien lo vigila.\n"+
				"  Arreglo: una linea `// LO VIGILA: TestLoQueSea` con un test que exista,\n"+
				"  o `// NADIE LO VIGILA: <motivo>` si de verdad no hay ninguno.\n"+
				"  Escribir el motivo no es lo mismo que ponerle puerta.",
				i.fichero, i.linea, i.nombre, hallazgos)
			continue
		}
		for _, n := range nombres {
			if _, ok := existen[n]; !ok {
				t.Errorf("%s:%d %s dice que lo vigila %s, y ese test NO EXISTE.\n"+
					"  Un nombre con la forma de lo verificable es lo que hace que nadie lo verifique.",
					i.fichero, i.linea, i.nombre, n)
			}
		}
	}
	t.Logf("%d interfaces exportados, %d enuncian un peligro y lo dicen",
		len(fronteras), conPeligro)
	if conPeligro == 0 {
		t.Error("ningun interfaz enuncia un peligro: el vocabulario ya no caza nada y la puerta no vigila")
	}
}

// TestTodaMarcaDePeligroDiceQuienLaVigila es la mitad B: la marca puesta a mano
// en cualquier sitio del arbol.
func TestTodaMarcaDePeligroDiceQuienLaVigila(t *testing.T) {
	existen := testsDelArbol(t)
	marcas := 0
	for _, f := range ficherosGo(t) {
		b, err := os.ReadFile(f) // #nosec G304 -- ruta de la propia enumeracion del arbol
		if err != nil {
			t.Fatal(err)
		}
		for _, bl := range bloquesDeComentario(strings.Split(string(b), "\n")) {
			tienePeligro := false
			for _, l := range bl.lineas {
				if rePeligro.MatchString(l) {
					tienePeligro = true
				}
			}
			if !tienePeligro {
				continue
			}
			marcas++
			vigilado, nombres := bloqueVigilado(bl.lineas)
			if !vigilado {
				t.Errorf("%s:%d una marca PELIGRO: sin decir quien la vigila.\n"+
					"  Arreglo: `// LO VIGILA: TestX` o `// NADIE LO VIGILA: <motivo>`.",
					filepath.ToSlash(f), bl.linea)
				continue
			}
			for _, n := range nombres {
				if _, ok := existen[n]; !ok {
					t.Errorf("%s:%d la marca dice que la vigila %s, y ese test NO EXISTE.",
						filepath.ToSlash(f), bl.linea, n)
				}
			}
		}
	}
	t.Logf("%d marcas PELIGRO: en el arbol", marcas)
}

type bloqueDeComentario struct {
	linea  int
	lineas []string
}

func bloquesDeComentario(ls []string) []bloqueDeComentario {
	var out []bloqueDeComentario
	i := 0
	for i < len(ls) {
		if !strings.HasPrefix(strings.TrimSpace(ls[i]), "//") {
			i++
			continue
		}
		inicio := i
		var bl []string
		for i < len(ls) && strings.HasPrefix(strings.TrimSpace(ls[i]), "//") {
			bl = append(bl, ls[i])
			i++
		}
		out = append(out, bloqueDeComentario{linea: inicio + 1, lineas: bl})
	}
	return out
}

// TestElDetectorDePeligrosSaltaCuandoDebe es el control negativo, en las dos
// direcciones: tiene que acusar lo que hay que acusar y callarse con lo demas.
func TestElDetectorDePeligrosSaltaCuandoDebe(t *testing.T) {
	casos := []struct {
		nombre   string
		lineas   []string
		peligro  bool
		vigilado bool
		nombres  []string
	}{
		{
			nombre:  "el caso que costo la mutacion M12",
			lineas:  []string{"// una consecuencia en blanco se leeria como que no activa nada"},
			peligro: true,
		},
		{
			nombre: "el mismo, con su puerta nombrada",
			lineas: []string{
				"// una consecuencia en blanco se leeria como que no activa nada",
				"// LO VIGILA: TestElDetectorDePeligrosSaltaCuandoDebe",
			},
			peligro:  true,
			vigilado: true,
			nombres:  []string{"TestElDetectorDePeligrosSaltaCuandoDebe"},
		},
		{
			nombre: "y con la admision explicita",
			lineas: []string{
				"// esto es peor que no tenerlo",
				"// NADIE LO VIGILA: no hay test, y se dice.",
			},
			peligro:  true,
			vigilado: true,
		},
		{
			nombre: "un comentario normal no se acusa",
			lineas: []string{"// Publicar compone el alcance de la instalacion."},
		},
		{
			nombre: "NADIE LO VIGILA no se lee como LO VIGILA",
			lineas: []string{
				"// es peor que ninguno",
				"// NADIE LO VIGILA: aqui no se puede nombrar ningun test.",
			},
			peligro:  true,
			vigilado: true,
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if hay := len(enuncianPeligro(c.lineas)) > 0; hay != c.peligro {
				t.Errorf("peligro detectado %v, esperaba %v", hay, c.peligro)
			}
			vig, nom := bloqueVigilado(c.lineas)
			if vig != c.vigilado {
				t.Errorf("vigilado %v, esperaba %v", vig, c.vigilado)
			}
			if fmt.Sprint(nom) != fmt.Sprint(c.nombres) {
				t.Errorf("nombres %v, esperaba %v", nom, c.nombres)
			}
		})
	}
}

// TestElHuecoQueEstaPuertaNoMiraSeCuenta imprime el cardinal del limite. NO
// afirma un techo a proposito: el numero sube cada vez que alguien escribe un
// comentario honesto, asi que un techo saltaria sobre trabajo legitimo y
// entrenaria a esquivarlo. Se cuenta para que el hueco no se olvide, que es
// para lo que sirve un cardinal.
func TestElHuecoQueEstaPuertaNoMiraSeCuenta(t *testing.T) {
	dentro := map[string]bool{}
	for _, i := range interfacesExportados(t) {
		for _, l := range i.texto {
			dentro[l] = true
		}
	}
	fuera := 0
	for _, f := range ficherosGo(t) {
		b, err := os.ReadFile(f) // #nosec G304 -- ruta de la propia enumeracion del arbol
		if err != nil {
			t.Fatal(err)
		}
		for _, l := range strings.Split(string(b), "\n") {
			if !strings.HasPrefix(strings.TrimSpace(l), "//") || dentro[l] {
				continue
			}
			if len(enuncianPeligro([]string{l})) > 0 {
				fuera++
			}
		}
	}
	t.Logf("%d lineas de comentario enuncian un peligro FUERA de las fronteras que "+
		"esta puerta mira. Ahi solo llega la marca PELIGRO: puesta a mano.", fuera)
	if fuera == 0 {
		t.Error("cero: el contador del hueco no esta contando nada")
	}
}
