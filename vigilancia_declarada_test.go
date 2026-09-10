package plazum

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"
)

// LA TERCERA MITAD DE LA VIGILANCIA DECLARADA: toda linea que diga quien vigila
// algo se comprueba, este donde este.
//
// # La escalada, y por que se ejecuta hoy
//
// `CLAUDE.md` dejo escrita la condicion de antemano, para no tener que decidirla
// en caliente: **si la clase vuelve a fallar dos veces, `NADIE LO VIGILA` deja de
// ser texto libre y pasa a llevar fecha y registro, con su cardinal vigilado por
// igualdad exacta**. Se ha cumplido, en tres dias:
//
//	07-09-2026  nucleo/estado/veredicto_test.go  un `NADIE LO VIGILA` puesto a
//	            mano sobre una frase que ya era falsa, en un `_test.go`
//	10-09-2026  superficies/pantallas/hoja_test.go  un godoc que prometia «la
//	            lista tiene su propia guarda mas abajo» y no habia ninguna
//
// Las dos en `_test.go`, que es donde `godoc_vigilado_test.go` NO mira
// (godoc_vigilado_test.go:128, que se salta los ficheros de test al enumerar
// interfaces).
//
// # EL BARRIDO QUE QUEDO APUNTADO SIN CONTAR, CONTADO (10-09-2026)
//
//	0    declaraciones `NADIE LO VIGILA` en todo el arbol
//	16   declaraciones `LO VIGILA`, 14 en produccion y 2 en `_test.go`
//	1    de esas 16 nombraba un test que NO EXISTIA
//	0    marcas `PELIGRO:`
//
// **El cero de la primera fila es el dato que cambia la forma de esta escalada**,
// y se dice en vez de disimularlo: las ocho apariciones textuales de
// `NADIE LO VIGILA` en el arbol son las del propio `godoc_vigilado_test.go` (su
// ejemplo de godoc, su expresion regular, dos mensajes de error y tres casos de
// su control negativo) mas una mencion historica en `veredicto_test.go:58` que
// habla del que se quito. Ninguna es una declaracion. O sea que el registro nace
// **vacio**, y lo que compra no es limpiar lo que hay: es que **anadir el
// primero tenga que ser deliberado**, porque hoy escribir uno es gratis y
// silencioso.
//
// **Y el 1 de la tercera fila es el hallazgo del barrido**:
// `nucleo/corpus/puente_estado.go:78` decia
// `// LO VIGILA: TestElPuenteNoMueveUnaFechaMasDeLoDeclarado` y ese test no
// existia. Es la forma exacta contra la que el godoc de `godoc_vigilado_test.go`
// avisa («un nombre con la forma de lo verificable es justo lo que hace que
// nadie vaya a verificarlo»), escrita en el arbol y sin que nadie la cazara: la
// marca esta sobre un bloque `var (...)`, que no es un interfaz exportado ni
// lleva `PELIGRO:`, y esas eran las dos unicas posiciones vigiladas. El test se
// escribio (`nucleo/corpus/puente_estado_test.go`) en vez de borrar la marca.
//
// # LO QUE ESTA ESCALADA NO CIERRA, dicho con su forma
//
// El segundo de los dos fallos que la dispararon, el de `hoja_test.go`, **esta
// puerta no lo habria cazado**, y merece decirse en vez de contarlo como
// cubierto. Aquella promesa no usaba el vocabulario: decia «la lista tiene su
// propia guarda mas abajo», que es prosa libre. Lo que se vigila aqui es que
// **quien usa el vocabulario diga la verdad**; una promesa de guarda escrita con
// otras palabras sigue siendo invisible.
//
// Cerrar eso pide un detector de prosa («guarda», «lo comprueba», «mas abajo»)
// sobre comentarios, y su fallo probable es acusar en falso a cualquier
// comentario que hable de comprobaciones, que en este arbol son cientos. Una
// puerta asi saltaria casi siempre y entrenaria a esquivarla, que es peor que no
// tenerla. Queda apuntado en `docs/pendientes.md` sin cerrar y con su motivo.

// LosDescargosSinVigilancia es el REGISTRO de los `NADIE LO VIGILA` del arbol.
//
// # Por que un registro y no solo la marca
//
// Porque la marca sola es texto libre: se escribe en un momento en que es
// verdad, y despues nadie vuelve. Con registro, **anadir uno es un cambio en
// este fichero**, o sea un gesto deliberado que aparece en el diff y que hay que
// justificar en el commit; y **quitarlo tambien se nota**, que es la mitad que
// una lista sin igualdad exacta no da.
//
// # La clave, y por que no es el numero de linea
//
// Un numero de linea se mueve solo. La clave es `fichero:simbolo`, donde el
// simbolo es la primera palabra reconocible de lo que el bloque documenta, que
// es lo que de verdad identifica el descargo. Si el simbolo se renombra, la
// entrada queda huerfana y esta puerta lo dice, que es exactamente lo que se
// quiere: un descargo cuyo sujeto ya no existe es un descargo que ya no descarga
// de nada.
//
// # NACIO VACIO Y DEJO DE ESTARLO AL DIA SIGUIENTE, que es la prueba de que sirve
//
// El barrido del 10-09-2026 no encontro ni una declaracion en el arbol, asi que
// este registro nacio vacio: lo que compraba no era limpiar lo que habia, era que
// **escribir el primero tuviera que ser deliberado**.
//
// **El 11-09-2026 entro el primero**, con la puerta del campo `articulo` de las
// obligaciones bilingues. Y entro por el camino que se habia previsto: el
// descargo iba a existir de todas formas, porque la `cita` enumera apartados y
// nadie la contrasta; sin registro se habria escrito como una frase mas en un
// godoc y no lo habria contado nadie. Con registro hubo que venir aqui, ponerle
// fecha y escribir que impide poner la puerta.
//
// El cardinal no se lee de esta prosa: lo vigila
// `CardinalDeDescargosSinVigilancia` por igualdad exacta, y el numero vivo lo
// imprime `TestTodaVigilanciaDeclaradaNombraUnTestQueExiste`.
var LosDescargosSinVigilancia = map[string]descargoSinVigilancia{
	// EL PRIMERO, y entro el 11-09-2026 con la puerta del campo `articulo`. Es
	// el registro haciendo lo que se escribio para hacer: el descargo existia de
	// todas formas (la cita enumera apartados y nadie la contrasta), y sin
	// registro se habria escrito como prosa libre y no lo habria contado nadie.
	"apartados_test.go:TestElCampoArticuloDeUnaObligacionBilingueDiceLosApartadosQueTraeElTexto": {
		Fecha: "11-09-2026",
		Motivo: "la puerta contrasta el campo `articulo` contra el texto, y la `cita` " +
			"tambien enumera apartados sin que nadie la contraste. No se le pone puerta " +
			"porque la cita es prosa y nombra A PROPOSITO articulos ajenos, que son las " +
			"remisiones del propio texto legal: la de mdr.art87 cita el art. 92, " +
			"apartados 5 y 7, y el art. 88. Un lector de N.M sobre la cita convertiria " +
			"cada remision en un apartado declarado, o sea que su fallo probable es " +
			"acusar a una cita correcta, y una puerta que acusa en falso se acaba " +
			"borrando. Se revisa cuando el formato separe la referencia estructurada de " +
			"la prosa de la cita, que hoy viajan en el mismo campo.",
	},
}

// descargoSinVigilancia es una fila del registro.
type descargoSinVigilancia struct {
	// Fecha es cuando se escribio, en DD-MM-AAAA. La misma que lleva la marca en
	// el codigo, y se comprueba que casan: si solo estuviera en un sitio, el
	// otro envejeceria sin que nadie lo notara.
	Fecha string
	// Motivo es por que no hay puerta. No vale «pendiente»: tiene que decir que
	// impide ponerla, porque eso es lo que la revision de dentro de un tiempo
	// necesita para decidir si sigue siendo cierto.
	Motivo string
}

// CardinalDeDescargosSinVigilancia es cuantos hay, por IGUALDAD EXACTA.
//
// Igual que `PUERTAS_ESPERADAS`: asi rompe tambien cuando el conjunto ENCOGE,
// que es cuando hay que venir a quitar la fila. Un techo solo por arriba deja
// que los descargos desaparezcan en silencio, y entonces el registro pasa a
// contar lo que hubo.
const CardinalDeDescargosSinVigilancia = 1

var (
	// reDeclaracionDeVigilancia casa una linea de comentario que declare
	// vigilancia. Anclada a `//` al principio de la linea A PROPOSITO: asi no
	// caza las apariciones dentro de literales de cadena, que en este arbol son
	// los ejemplos y los casos de prueba del propio guardian. Se comprobo: sin
	// el ancla, el barrido acusaba ocho declaraciones que no lo son.
	reDeclaracionDeVigilancia = regexp.MustCompile(
		`^\s*//\s*(NADIE LO VIGILA|LO VIGILA)\s*(\([^)]*\))?\s*:\s*(\S.*)$`)
	// reFechaDeDescargo exige DD-MM-AAAA dentro del parentesis.
	reFechaDeDescargo = regexp.MustCompile(`^\((\d{2}-\d{2}-\d{4})\)$`)
	reTestNombrado    = regexp.MustCompile(`\bTest[A-Za-z0-9_]+`)
)

// laFormaAutorreferente es el `LO VIGILA` que no nombra ningun test porque el
// test es el fichero en el que esta escrito.
//
// SE ADMITE Y NO SE PROHIBE, y el motivo es que es la forma HONESTA de un caso
// real: `demo_fuera_del_corpus_test.go` documenta lo que vigila en su propio
// encabezado. Obligarlo a nombrarse a si mismo produciria una linea que casa la
// letra y no dice nada mas que la que ya hay.
const laFormaAutorreferente = "este mismo test"

// vigilanciaDeclarada es una linea de vigilancia encontrada en el arbol.
type vigilanciaDeclarada struct {
	fichero string
	linea   int
	niega   bool // true si es NADIE LO VIGILA
	fecha   string
	texto   string
	tests   []string
	simbolo string
}

// TestTodaVigilanciaDeclaradaNombraUnTestQueExiste es la puerta.
//
// # Por que recorre TODO el arbol, `_test.go` incluidos
//
// Porque es justo donde fallaron los dos casos que dispararon esta escalada. La
// mitad A de `godoc_vigilado_test.go` enumera interfaces exportados de ficheros
// de produccion, y la mitad B busca la marca `PELIGRO:`; una linea de vigilancia
// escrita en cualquier otro sitio —sobre un `var`, sobre una funcion, dentro de
// un `_test.go`— no la miraba nadie.
//
// AQUI NO HAY RIESGO DE SALTAR SOBRE TRABAJO LEGITIMO, que es la pregunta que
// hay que hacerse antes de escribir una puerta: esto solo mira lineas que YA
// usan el vocabulario a proposito. No obliga a nadie a declarar vigilancia; solo
// exige que quien la declare diga la verdad.
func TestTodaVigilanciaDeclaradaNombraUnTestQueExiste(t *testing.T) {
	existen := testsDelArbol(t)
	decls := vigilanciasDelArbol(t)

	// EL SUELO. Si el barrido deja de encontrar lineas, esta puerta seguiria
	// verde recorriendo la nada. Va holgado por debajo a proposito, para que
	// quitar una no lo dispare: lo que vigila un numero exacto es el registro de
	// los descargos, no esto.
	//
	// Y NO SE ESCRIBE AQUI CUANTAS HAY HOY. La frase que habia decia «hoy son
	// dieciseis» y el 11-09-2026 ya eran veintidos, o sea prosa caducada dentro
	// del guardian de la prosa caducada. El numero vivo sale por el t.Logf del
	// final, que lo deriva del barrido.
	if len(decls) < 10 {
		t.Fatalf("el barrido encuentra %d lineas de vigilancia en el arbol, que son muy "+
			"pocas: el recorrido o la expresion regular han dejado de casar, y esta "+
			"puerta estaria midiendo el vacio", len(decls))
	}

	niegan, afirman := 0, 0
	for _, d := range decls {
		if d.niega {
			niegan++
			// UN DESCARGO LLEVA FECHA. Sin ella no se puede saber si sigue
			// siendo cierto, y un descargo que nadie puede fechar es
			// indistinguible de uno que caduco hace meses.
			if d.fecha == "" {
				t.Errorf("%s:%d un `NADIE LO VIGILA` sin fecha.\n"+
					"  Arreglo: `// NADIE LO VIGILA (DD-MM-AAAA): motivo`, y su fila en\n"+
					"  LosDescargosSinVigilancia.\n"+
					"  Un descargo sin fecha no se puede revisar, y uno que no se revisa se\n"+
					"  convierte en la afirmacion acompanada escrita por su propio autor.\n"+
					"  decia: %s", d.fichero, d.linea, d.texto)
				continue
			}
			if _, err := time.Parse("02-01-2006", d.fecha); err != nil {
				t.Errorf("%s:%d la fecha del descargo (%q) no es DD-MM-AAAA",
					d.fichero, d.linea, d.fecha)
			}
			continue
		}
		afirman++
		if strings.Contains(strings.ToLower(d.texto), laFormaAutorreferente) {
			continue
		}
		if len(d.tests) == 0 {
			t.Errorf("%s:%d un `LO VIGILA` que no nombra ningun test.\n"+
				"  Arreglo: nombrar el test, o usar `NADIE LO VIGILA (fecha): motivo`.\n"+
				"  Una vigilancia declarada y sin sujeto afirma que alguien miro, que es\n"+
				"  la mitad que hace peligrosa a esta familia.\n  decia: %s",
				d.fichero, d.linea, d.texto)
			continue
		}
		for _, n := range d.tests {
			if _, ok := existen[n]; !ok {
				t.Errorf("%s:%d dice que lo vigila %s, y ese test NO EXISTE en el arbol.\n"+
					"  Un identificador con la FORMA de lo verificable es justo lo que hace\n"+
					"  que nadie vaya a verificarlo. Es el caso que encontro el barrido del\n"+
					"  10-09-2026 en nucleo/corpus/puente_estado.go:78.\n"+
					"  Arreglo: escribir el test, o corregir el nombre, o bajarlo a\n"+
					"  `NADIE LO VIGILA (fecha): motivo`.",
					d.fichero, d.linea, n)
			}
		}
	}
	t.Logf("%d lineas de vigilancia en el arbol: %d nombran un test, %d son descargos",
		len(decls), afirman, niegan)
}

// TestElRegistroDeDescargosCuadraConElArbol es la igualdad exacta, en las dos
// direcciones.
//
// Sentido 1: todo descargo del arbol tiene su fila. Sentido 2: toda fila apunta
// a un descargo que sigue existiendo. Y el cardinal, exacto, para que encoger
// tambien rompa.
func TestElRegistroDeDescargosCuadraConElArbol(t *testing.T) {
	enElArbol := map[string]vigilanciaDeclarada{}
	for _, d := range vigilanciasDelArbol(t) {
		if !d.niega {
			continue
		}
		enElArbol[d.fichero+":"+d.simbolo] = d
	}

	// SENTIDO 1: lo que hay en el arbol esta registrado.
	for clave, d := range enElArbol {
		fila, hay := LosDescargosSinVigilancia[clave]
		if !hay {
			t.Errorf("%s:%d hay un `NADIE LO VIGILA` que NO esta en el registro.\n"+
				"  Arreglo: anadir la fila %q a LosDescargosSinVigilancia con su fecha y su\n"+
				"  motivo, y subir CardinalDeDescargosSinVigilancia.\n"+
				"  Escribir un descargo tiene que ser un gesto deliberado que aparezca en el\n"+
				"  diff: si es gratis y silencioso, se escribe sin pensarlo, que es como se\n"+
				"  escribio el que caduco el 07-09-2026.\n  decia: %s",
				d.fichero, d.linea, clave, d.texto)
			continue
		}
		// LAS DOS FECHAS SON LA MISMA. Si solo estuviera en un sitio, el otro
		// envejeceria sin que nadie lo notara, que es la familia entera.
		if fila.Fecha != d.fecha {
			t.Errorf("%s:%d el descargo lleva la fecha %q y el registro dice %q. Las dos "+
				"se escriben a la vez o una de las dos miente",
				d.fichero, d.linea, d.fecha, fila.Fecha)
		}
		if len(strings.TrimSpace(fila.Motivo)) < 20 {
			t.Errorf("la fila %q del registro no dice por que no hay puerta (%q).\n"+
				"  «Pendiente» no es un motivo: tiene que decir QUE impide ponerla, que es\n"+
				"  lo que la revision necesita para decidir si sigue siendo cierto",
				clave, fila.Motivo)
		}
	}

	// SENTIDO 2: lo registrado sigue existiendo. Sin esta mitad el registro
	// envejece hasta ser una lista de los descargos que hubo, y una fila
	// huerfana hace parecer revisado lo que ya no esta.
	for clave, fila := range LosDescargosSinVigilancia {
		if _, hay := enElArbol[clave]; !hay {
			t.Errorf("el registro trae %q y en el arbol no hay ningun `NADIE LO VIGILA` ahi.\n"+
				"  O se quito el descargo (y entonces esta fila sobra y hay que BAJAR\n"+
				"  CardinalDeDescargosSinVigilancia), o se renombro el simbolo.\n  decia: %s",
				clave, fila.Motivo)
		}
	}

	// EL CARDINAL, POR IGUALDAD EXACTA.
	if len(LosDescargosSinVigilancia) != CardinalDeDescargosSinVigilancia {
		t.Errorf("el registro trae %d filas y CardinalDeDescargosSinVigilancia dice %d.\n"+
			"  La igualdad es exacta a proposito: asi rompe tambien cuando el conjunto\n"+
			"  ENCOGE, que es cuando hay que venir a quitar la fila.",
			len(LosDescargosSinVigilancia), CardinalDeDescargosSinVigilancia)
	}
	t.Logf("%d descargos sin vigilancia en el arbol, %d registrados",
		len(enElArbol), len(LosDescargosSinVigilancia))
}

// vigilanciasDelArbol recorre TODOS los .go del arbol, tests incluidos.
//
// SE RECORRE `git ls-files` Y NO EL DISCO, por lo mismo que las demas puertas de
// este fichero: los worktrees de `.claude/` llevan copias, y una declaracion que
// solo exista alli contaria como del arbol.
func vigilanciasDelArbol(t *testing.T) []vigilanciaDeclarada {
	t.Helper()
	salida, err := gitDice("ls-files", "*.go")
	if err != nil {
		t.Skipf("SALTADO, no comprobado: git no contesta aqui (%v). NO es verde", err)
	}
	var out []vigilanciaDeclarada
	ficheros := 0
	for _, ruta := range strings.Split(salida, "\n") {
		ruta = strings.TrimSpace(ruta)
		if ruta == "" {
			continue
		}
		b, err := os.ReadFile(ruta) // #nosec G304 -- rutas del propio arbol de git
		if err != nil {
			continue
		}
		ficheros++
		ls := strings.Split(string(b), "\n")
		for i, l := range ls {
			m := reDeclaracionDeVigilancia.FindStringSubmatch(l)
			if m == nil {
				continue
			}
			d := vigilanciaDeclarada{
				fichero: filepath.ToSlash(ruta),
				linea:   i + 1,
				niega:   m[1] == "NADIE LO VIGILA",
				texto:   strings.TrimSpace(m[3]),
				simbolo: simboloDocumentado(ls, i),
			}
			if m[2] != "" {
				if f := reFechaDeDescargo.FindStringSubmatch(m[2]); f != nil {
					d.fecha = f[1]
				}
			}
			if !d.niega {
				d.tests = reTestNombrado.FindAllString(m[3], -1)
			}
			out = append(out, d)
		}
	}
	if ficheros < 100 {
		t.Fatalf("he mirado %d ficheros .go y el arbol tiene cientos: este recorrido "+
			"esta midiendo el vacio", ficheros)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].fichero != out[j].fichero {
			return out[i].fichero < out[j].fichero
		}
		return out[i].linea < out[j].linea
	})
	return out
}

// simboloDocumentado busca hacia abajo la primera linea que no sea comentario ni
// vacia, y saca de ella el nombre de lo que el bloque documenta.
//
// ES APROXIMADO Y NO PASA NADA, y merece decirse: sirve para dar a la clave del
// registro algo mas estable que un numero de linea, no para analizar Go. Si no
// reconoce nada, devuelve `?`, y entonces la clave es fichero + `?`, que sigue
// siendo utilizable mientras no haya dos descargos sin simbolo en el mismo
// fichero. Ese caso lo cazaria el sentido 1 del registro, porque uno de los dos
// se quedaria sin fila.
func simboloDocumentado(ls []string, desde int) string {
	for i := desde + 1; i < len(ls) && i < desde+40; i++ {
		l := strings.TrimSpace(ls[i])
		if l == "" || strings.HasPrefix(l, "//") {
			continue
		}
		campos := strings.FieldsFunc(l, func(r rune) bool {
			return !(r == '_' || r == '.' ||
				(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'))
		})
		for _, c := range campos {
			switch c {
			case "func", "type", "var", "const", "return", "if", "for":
				continue
			}
			return c
		}
		return "?"
	}
	return "?"
}

// TestElDetectorDeVigilanciaDeclaradaAcusaYSeCalla es el control negativo, en
// las dos direcciones y sobre lineas sinteticas.
//
// Hace falta porque el registro nace VACIO: sin esto, la puerta de arriba
// recorreria dieciseis lineas que estan bien y no habria demostrado nunca que
// sabe ponerse roja. Y la direccion contraria importa igual: su fallo probable es
// acusar a la prosa que HABLA de estas marcas, que en este arbol es justo el
// fichero que las define.
func TestElDetectorDeVigilanciaDeclaradaAcusaYSeCalla(t *testing.T) {
	casos := []struct {
		nombre string
		linea  string
		casa   bool
		niega  bool
		fecha  string
		tests  []string
	}{
		{
			nombre: "una vigilancia normal",
			linea:  "// LO VIGILA: TestLoQueSea",
			casa:   true, tests: []string{"TestLoQueSea"},
		},
		{
			nombre: "dos tests en la misma linea",
			linea:  "// LO VIGILA: TestUno, TestDos",
			casa:   true, tests: []string{"TestUno", "TestDos"},
		},
		{
			nombre: "un descargo con su fecha",
			linea:  "// NADIE LO VIGILA (10-09-2026): no hay forma de provocarlo desde fuera.",
			casa:   true, niega: true, fecha: "10-09-2026",
		},
		{
			nombre: "un descargo SIN fecha sigue casando, para poder acusarlo",
			linea:  "// NADIE LO VIGILA: se me olvido la fecha.",
			casa:   true, niega: true,
		},
		{
			nombre: "indentada dentro de una funcion",
			linea:  "\t\t// LO VIGILA: TestDentroDeUnaFuncion",
			casa:   true, tests: []string{"TestDentroDeUnaFuncion"},
		},
		// LA MITAD QUE SE CALLA. Las tres formas en las que el vocabulario
		// aparece en este arbol sin ser una declaracion, y las tres tienen que
		// pasar de largo: si el detector las acusara, la reaccion barata seria
		// eximir el fichero que las contiene, que es el propio guardian.
		{
			nombre: "dentro de un literal de cadena no es una declaracion",
			linea:  `			"// NADIE LO VIGILA: no hay test, y se dice.",`,
		},
		{
			nombre: "un ejemplo dentro de un godoc tampoco",
			linea:  "//\t// NADIE LO VIGILA: el motivo, dicho.",
		},
		{
			nombre: "una mencion en prosa tampoco",
			linea:  "// Y el detalle que la hace doctrina: llevaba un `NADIE LO VIGILA` puesto a mano",
		},
		{
			nombre: "un comentario cualquiera tampoco",
			linea:  "// Publicar compone el alcance de la instalacion.",
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			m := reDeclaracionDeVigilancia.FindStringSubmatch(c.linea)
			if (m != nil) != c.casa {
				t.Fatalf("casa=%v y se esperaba %v sobre %q", m != nil, c.casa, c.linea)
			}
			if m == nil {
				return
			}
			if niega := m[1] == "NADIE LO VIGILA"; niega != c.niega {
				t.Errorf("niega=%v y se esperaba %v", niega, c.niega)
			}
			var fecha string
			if m[2] != "" {
				if f := reFechaDeDescargo.FindStringSubmatch(m[2]); f != nil {
					fecha = f[1]
				}
			}
			if fecha != c.fecha {
				t.Errorf("fecha=%q y se esperaba %q", fecha, c.fecha)
			}
			if !c.niega {
				tests := reTestNombrado.FindAllString(m[3], -1)
				if fmt.Sprint(tests) != fmt.Sprint(c.tests) {
					t.Errorf("tests=%v y se esperaban %v", tests, c.tests)
				}
			}
		})
	}
}
