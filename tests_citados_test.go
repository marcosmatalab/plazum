package plazum

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// UN DOCUMENTO QUE NOMBRA UN TEST NOMBRA UNO QUE EXISTE.
//
// # El agujero, y por que lo dejaba pasar el guardian que ya habia
//
// `TestTodaVigilanciaDeclaradaNombraUnTestQueExiste` recorre TODO el arbol
// exigiendo que cada `LO VIGILA:` nombre un test real. Y sólo mira **ficheros de
// codigo**: una promesa de vigilancia escrita en un `.md` es invisible para el.
//
// Eso deja fuera justo donde mas duele, porque un `.md` es lo que lee alguien de
// fuera. `DEPENDENCIAS.md` decia *«Lo vigila `TestTimestampSoloConstruyeLaPeticion`,
// que recorre el AST del paquete y falla si alguien vuelve a usar `timestamp`
// para otra cosa»*, y ese test **no existe**: se borro al salir la dependencia,
// tal como su propio godoc habia previsto, y la frase se quedo. Un identificador
// con la forma de lo verificable es justo lo que hace que nadie vaya a
// verificarlo.
//
// # Medido antes de escribirla, que es lo que decide si es viable
//
// El 11-09-2026, sobre los ficheros versionados: **140 nombres de test citados en
// `.md`, 6 que no existian**. Con seis, la puerta es barata y su lista de
// excepciones cabe en la mano. Con sesenta habria sido otra conversacion.
//
// Y de los seis, **dos eran mios y de esta misma semana**: uno se renombro al
// cambiarle el ancla a la puerta de vigencias y dos documentos se quedaron con el
// nombre viejo. O sea que no es una clase que se arregle con cuidado.
//
// # Por que esta puerta NO detecta prosa, y eso es lo que la hace viable
//
// No busca «lo vigila» ni ninguna formula. Busca **un identificador entre
// comillas simples que empiece por `Test` o `Fuzz`**, que es mecanico y no tiene
// juicio dentro. El godoc de `vigilancia_declarada_test.go` explica por que un
// detector de prosa no entra: su fallo probable es acusar a cualquier comentario
// que hable de comprobaciones, y una puerta que salta casi siempre entrena a
// esquivarla. Esta no puede equivocarse asi.
func TestTodoTestQueCitaUnDocumentoExiste(t *testing.T) {
	existen := map[string]bool{}
	reFunc := regexp.MustCompile(`(?m)^func ((?:Test|Fuzz)[A-Za-z0-9_]+)`)
	for _, f := range ficherosVersionados(t, "*_test.go") {
		b, err := os.ReadFile(f) // #nosec G304 -- ruta que da git ls-files
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range reFunc.FindAllSubmatch(b, -1) {
			existen[string(m[1])] = true
		}
	}
	if len(existen) < 500 {
		t.Fatalf("solo %d tests encontrados en el arbol: el recorrido esta roto y esta puerta "+
			"acusaria a todo el mundo", len(existen))
	}

	reCita := regexp.MustCompile("`((?:Test|Fuzz)[A-Z0-9][A-Za-z0-9_]*)`")
	citados, huerfanos := 0, map[string][]string{}
	for _, f := range ficherosVersionados(t, "*.md") {
		b, err := os.ReadFile(f) // #nosec G304 -- ruta que da git ls-files
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range reCita.FindAllStringSubmatch(string(b), -1) {
			n := m[1]
			citados++
			if existen[n] || mencionesHistoricasDeTests[n] != "" {
				continue
			}
			huerfanos[n] = append(huerfanos[n], f)
		}
	}
	if citados < 50 {
		t.Fatalf("solo %d nombres de test citados en los .md: el recorrido esta roto", citados)
	}

	nombres := make([]string, 0, len(huerfanos))
	for n := range huerfanos {
		nombres = append(nombres, n)
	}
	sort.Strings(nombres)
	for _, n := range nombres {
		t.Errorf(`%s se cita en %s y NO EXISTE en el arbol.

  Un documento que nombra un test esta diciendo «esto lo comprueba alguien», y
  quien lo lea dara por revisado lo que no revisa nadie. Es la afirmacion
  acompanada en su forma mas pura, y el identificador con forma de verificable es
  justo lo que hace que nadie vaya a verificarlo.

  Tres salidas, y las tres valen:
    - el test existe con otro nombre       -> corrige la cita
    - el test se borro y la frase se quedo -> reescribe la frase en pasado y
                                              nombra lo que vigila HOY
    - la mencion es historica a proposito  -> anadela a mencionesHistoricasDeTests
                                              con su motivo, que es un gesto
                                              deliberado y aparece en el diff`,
			n, strings.Join(huerfanos[n], ", "))
	}

	if !t.Failed() {
		t.Logf("MEDIDO: %d nombres de test citados en los .md versionados, %d tests en el "+
			"arbol, %d mencion(es) historica(s) declarada(s).",
			citados, len(existen), len(mencionesHistoricasDeTests))
	}
}

// mencionesHistoricasDeTests son los nombres que un documento cita A PROPOSITO
// sabiendo que ya no existen, porque esta contando lo que paso.
//
// # Por que una lista y no un patron
//
// Porque un patron («si la frase esta en pasado, no acuses») es un detector de
// prosa, y eso es lo que esta puerta evita a proposito. Una lista obliga a un
// gesto deliberado: anadir una entrada es un cambio en este fichero, aparece en
// el diff, y hay que escribir por que. Es la misma forma que
// `LosDescargosSinVigilancia`.
//
// El valor de cada entrada es el motivo, y no vale «pendiente»: tiene que decir
// que paso con ese test, porque eso es lo que la revision de dentro de un tiempo
// necesita.
var mencionesHistoricasDeTests = map[string]string{
	"TestTimestampSoloConstruyeLaPeticion": "se murio de exito el 26-08-2026: vigilaba que " +
		"github.com/digitorus/timestamp se usara solo para armar la consulta RFC 3161, y al " +
		"escribirse las cuarenta lineas del TimeStampReq la dependencia salio entera de go.mod. " +
		"Lo citan DEPENDENCIAS.md y el godoc de adaptadores/tsa/frontera_test.go, los dos " +
		"contando la historia y los dos nombrando ya lo que vigila hoy.",
	"TestElPkcs7TransitivoNoEsElQueRevienta": "es el hallazgo 16 de docs/pendientes.md: la " +
		"puerta que llegaba a lo que vigila POR UN CAMINO QUE EL PRODUCTO YA NO USA. Comprobaba " +
		"el pkcs7 de aguas arriba llamando a timestamp.Parse, y al quitarse timestamp se quedo " +
		"midiendo el camino y no lo vigilado. Se cita para contar eso.",
	"TestBer2derNoEsCuadraticoSobreLaEntrada": "el test existe con otro nombre desde que se " +
		"midio la amplificacion de verdad: es TestBer2derAmplificaYPorEsoElTokenLlevaTope, en " +
		"adaptadores/tsa/internal/pkcs7/amplificacion_test.go. El LEEME lo cita por el nombre " +
		"viejo al contar como se llego a el.",
	"TestMain": "no es un test de este repositorio: es el gancho de la biblioteca estandar de " +
		"Go, y docs/pendientes.md lo nombra para hablar de un panico dentro de el.",
}

// EL CONTROL NEGATIVO: el detector tiene que ver una cita rota y dejar pasar una
// buena.
//
// Sin esto, la puerta de arriba pasaria con una expresion regular que no casara
// nunca —todo verde, cero citas— o con una lista de excepciones que tragara
// cualquier nombre. Las dos cosas dan verde sobre un arbol sano.
func TestElDetectorDeTestsCitadosAcusaYSeCalla(t *testing.T) {
	re := regexp.MustCompile("`((?:Test|Fuzz)[A-Z0-9][A-Za-z0-9_]*)`")

	for _, c := range []struct {
		texto  string
		quiero string
		porQue string
	}{
		{"Lo vigila `TestAlgoQueExiste`, que recorre el AST.", "TestAlgoQueExiste",
			"es la forma exacta en que DEPENDENCIAS.md escribia la cita rota"},
		{"con `FuzzParserDeCorpus` sobre entrada adversaria", "FuzzParserDeCorpus",
			"el fuzzing tambien es una puerta y tambien se cita"},
		{"la tabla de `CardinalesVigiladosDeLaInstantanea` dice cuantas", "",
			"NO es un test: un identificador que no empieza por Test ni Fuzz no se toca, " +
				"porque acusarlo convertiria la puerta en un detector de identificadores"},
		{"Test sin comillas no cuenta, TestNiEsteTampoco", "",
			"sin comillas es prosa, y ahi empieza la deteccion de prosa que esta puerta evita"},
		{"`Testing` y `Testigo` no son tests", "",
			"el nombre tiene que seguir con mayuscula o digito tras `Test`: `Testing` casa " +
				"con Test+ing, asi que si esto acusara habria que revisar el patron"},
	} {
		m := re.FindStringSubmatch(c.texto)
		hay := ""
		if m != nil {
			hay = m[1]
		}
		if hay != c.quiero {
			t.Errorf("sobre %q el detector saco %q y se esperaba %q.\n  Por que importa: %s",
				c.texto, hay, c.quiero, c.porQue)
		}
	}

	// Y LA LISTA DE EXCEPCIONES NO PUEDE ESTAR VACIA DE MOTIVOS: una entrada sin
	// explicar es una excepcion que nadie puede revisar.
	for n, motivo := range mencionesHistoricasDeTests {
		if len(strings.TrimSpace(motivo)) < 40 {
			t.Errorf("la mencion historica %q no dice que paso con ese test (%q).\n"+
				"  Sin motivo, la lista deja de ser un registro y pasa a ser una forma de "+
				"callar la puerta", n, motivo)
		}
	}
}
