package plazum

import (
	"encoding/json"
	"os"
	"regexp"
	"sort"
	"testing"
)

// TODA AFIRMACION DE LA INTERFAZ SOBRE LO QUE EL PRODUCTO HACE O NO HACE SE ATA
// AL COMPORTAMIENTO. No se escribe al lado.
//
// # De donde sale esta puerta
//
// Segundo hallazgo de la pasada del comprador sobre el binario publicado (R4,
// 06-09-2026). La pantalla que explica el camino guiado decia, con estas
// palabras, «plazum todavia no guarda tus respuestas». Las guarda desde el
// 04-09-2026. La frase era verdad el dia que se escribio y dejo de serlo dos
// dias despues sin que nada se pusiera rojo, porque una cadena de catalogo no
// tiene ninguna relacion mecanica con el codigo del que habla.
//
// Es la familia de la AFIRMACION ACOMPANADA en el sitio donde mas cara sale: no
// en un comentario que lee un autor, sino en la pantalla que lee el cliente. Y
// con el agravante de siempre, que la mitad que caduca es la que no tiene
// puerta: el rotulo se cambia por gusto, el comportamiento se cambia por
// trabajo, y nadie los mira juntos.
//
// # Que exige
//
// Toda cadena del catalogo que afirme algo sobre el comportamiento del producto
// entra en el registro de abajo, y el registro nombra EL TEST QUE EJERCE ESE
// COMPORTAMIENTO. El test tiene que existir. Las dos direcciones:
//
//   - una cadena nueva que afirme y no este en el registro pone esto rojo, que
//     es el momento exacto en que hay que contestar la pregunta;
//   - una entrada del registro cuya cadena ya no existe o ya no afirma nada
//     tambien lo pone, porque si no el registro envejece igual que la prosa que
//     vino a vigilar.
//
// # SU LIMITE, DICHO
//
// Esta puerta NO comprueba que la afirmacion sea CIERTA: eso no lo puede hacer
// un test generico, porque exigiria entender la frase. Lo que hace es obligar a
// que sea CONTESTABLE, que es lo que fallo: quien hubiera tenido que escribir la
// linea de «plazum todavia no guarda tus respuestas» habria ido a buscar el test
// que lo demuestra y se habria encontrado con
// TestLoQueSeContestaSeGuardaYVuelveSinNadaEnLaDireccion, que demuestra lo
// contrario. La puerta no descubre la mentira, pone la pregunta delante de
// alguien.
//
// Y admite «SIN COMPROBACION» con motivo, a proposito: una atadura inventada
// seria peor que un hueco declarado, porque el hueco se cuenta y la mentira no.
// El cardinal se imprime al final.
type atadura struct {
	// test es el que EJERCE el comportamiento del que habla la cadena. No vale
	// un test que solo compruebe que la cadena aparece en la pagina: eso mide
	// que el rotulo esta, que es justo la mitad que no caduca.
	test string
	// motivo solo cuando no hay test, y entonces se cuenta como hueco.
	motivo string
}

var afirmacionesDelProducto = map[string]atadura{
	"acta.descargo.empate_de_clasificacion": {
		test: "TestLoQueConstaYNoEsCulpaTambienLlevaSuFrase",
	},
	"acta.parrafo.plazum_no_escribe": {
		test: "TestTodaLaProsaDelActaDiceDeDondeSalenSusPalabras",
	},
	"acta.parrafo.sin_quien_asistio": {
		motivo: "quien asistio a la revision entra por el acta y no hay hoy un caso " +
			"que componga un acta SIN asistentes y afirme que no sale ningun nombre",
	},
	"calendario.pantalla.descarte.no_es_tuyo": {
		test: "TestLoQueNoSaleDeTusRespuestasVaDetrasDeTodoLoTuyo",
	},
	"calendario.pantalla.ics_sin_alcance": {
		test: "TestElICSSaleDelMismoCalendarioYNoSaleVacioSinAlcance",
	},
	"calendario.pantalla.sin_alcance.que_es": {
		test: "TestSinAlcanceLaPantallaExisteYDiceComoSalirDeAhi",
	},
	"camino.sin_progreso": {
		test: "TestLoQueSeContestaSeGuardaYVuelveSinNadaEnLaDireccion",
	},
	"camino.verbo.escalado": {
		test: "TestElEscaladoEnSecoNiMandaNiTocaElDiario",
	},
	"evidencia.descargo": {
		test: "TestSinPruebaYSinObservacionNoSonLoMismo",
	},
	"evidencia.ilegible": {
		test: "TestLaEvidenciaIlegibleNoSeLeeComoQueNadieHaRecolectado",
	},
	"error.alcance_ilegible": {
		test: "TestUnAlmacenQueNoSeLeeNoSeConvierteEnUnaEntrevistaEnBlanco",
	},
	"escalado.pantalla.no_manda_solo": {
		test: "TestLaPantallaDelEscaladoNoTienePorDondeMandarNada",
	},
	"uar.cierre.que_es": {
		test: "TestElCierreBloqueadoDiceQueFaltaSinSacarteDeLaPantalla",
	},
	"uar.sin_campana.que_es": {
		test: "TestUnaCampanaSeReconstruyeDesdeElFicheroYElLedger",
	},
	"uar.sin_campana.por_que_el_fichero": {
		motivo: "«no guarda tu lista de personas» es una propiedad del adaptador de " +
			"disco, y lo que hay hoy prueba que las filas se releen, no que no se " +
			"guarden",
	},
	"ui.pie.no_asesoramiento": {
		motivo: "no es una afirmacion sobre el comportamiento del binario sino el " +
			"descargo juridico del producto; no hay comportamiento que ejercer",
	},

	// LAS CINCO QUE ENTRARON CON LA TERCERA FORMA DE reAfirmacion, el
	// 10-09-2026. Las cinco absuelven, o sea dicen que algo NO pasa, y las cinco
	// se leen mientras alguien decide que declara sobre su organizacion.
	"alcance.consecuencia.ninguna": {
		test: "TestLaConsecuenciaQueSeEnsenaEsLaQueOcurre",
	},
	"alcance.dormidas.nadie_la_pide": {
		test: "TestEsconderUnaPreguntaNoPuedeCambiarNingunVeredicto",
	},
	"alcance.dormidas.porque": {
		test: "TestEsconderUnaPreguntaNoPuedeCambiarNingunVeredicto",
	},
	"alcance.dormidas.titulo": {
		test: "TestEsconderUnaPreguntaNoPuedeCambiarNingunVeredicto",
	},
	"alcance.guardado.huerfanas": {
		test: "TestUnaRespuestaGuardadaQueYaNoTienePreguntaSeDice",
	},
}

// reAfirmacion caza las formas en que este catalogo habla del producto: con su
// nombre delante, sin el cuando el sujeto se sobreentiende, y ABSOLVIENDO.
//
// # La tercera forma entro el 10-09-2026, y con ella la afirmacion mas
// # consecuente que hace este producto
//
// Las dos primeras formas cazaban 16 cadenas de 492. Todas hablan de lo que
// plazum HACE. Ninguna cazaba lo que plazum dice que NO PASA, que es la clase de
// frase que absuelve: «contestar que si aqui no activa ninguna obligacion
// nueva», «responderla no mueve nada», «no deciden nada todavia». Esas frases se
// leen en la pantalla donde alguien esta decidiendo que declara sobre su
// organizacion, y una que sea falsa esconde trabajo detras de una linea
// tranquilizadora.
//
// Con la tercera forma son 21 de 492, y las cinco que entran son exactamente esa
// clase. El cardinal lo imprime la puerta de abajo, derivado y no escrito aqui.
//
// # Su fallo probable, y por que el control negativo va en las dos direcciones
//
// No es acusar de mas: es que alguien reescriba una de esas cinco cadenas y la
// saque del censo sin tocar el producto, porque la clave dejaria de casar y la
// direccion 2 no protestaria (nunca habria estado en el registro). Contra eso no
// hay regexp que valga; lo que hay es que las cinco esten en el registro HOY, y
// que la direccion 2 se ponga roja el dia que una deje de afirmar. Queda dicho
// aqui porque es el hueco de esta puerta y no se tapa con mas alternativas.
var reAfirmacion = regexp.MustCompile(
	`(?i)plazum (no |todav[ií]a |a[uú]n |guarda|sabe|elige|deja|presta|manda|env[ií]a)` +
		`|no manda nada|no se lo inventa` +
		`|no activa ningun[ao]|no mueve nada|no deciden? nada|no cambian? hoy ninguna`)

// HuecosEsperados es cuantas afirmaciones estan declaradas SIN comprobacion.
//
// Es un techo por igualdad exacta y no un maximo, por lo mismo que
// PUERTAS_ESPERADAS: sin el, la forma barata de aprobar esta puerta es escribir
// `motivo:` en vez de atar el comportamiento, y eso se hace en una linea y no
// deja rastro. Con el, cada hueco nuevo obliga a subir un numero a la vista de
// todos, y cada hueco que se cierra obliga a bajarlo.
//
// Hoy, 10-09-2026: 3 de 21, o sea 18 atadas a un test que ejerce.
//
// Y el numero lo puso la puerta, no yo: lo escribi a 2 de memoria y la
// comprobacion dijo 3 en su primera ejecucion. Es la misma familia que se lleva
// cazadas seis veces en este repositorio, y la unica defensa sigue siendo la
// misma: el cardinal sale de la orden, no de la cabeza.
const HuecosEsperados = 3

func cadenasDelCatalogo(t *testing.T) map[string]string {
	t.Helper()
	b, err := os.ReadFile("adaptadores/catalogo/cadenas/es.json")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if len(m) < 100 {
		t.Fatalf("solo %d cadenas: el catalogo no se ha leido entero", len(m))
	}
	return m
}

func TestTodaAfirmacionDeLaInterfazSeAtaAlComportamiento(t *testing.T) {
	cadenas := cadenasDelCatalogo(t)
	existen := testsDelArbol(t)

	// DIRECCION 1: lo que afirma y no esta en el registro.
	var afirman []string
	for k, v := range cadenas {
		if reAfirmacion.MatchString(v) {
			afirman = append(afirman, k)
		}
	}
	sort.Strings(afirman)
	for _, k := range afirman {
		if _, ok := afirmacionesDelProducto[k]; !ok {
			t.Errorf("la cadena %q afirma algo sobre lo que plazum hace o no hace y no "+
				"esta atada a ningun comportamiento.\n  Texto: %q\n"+
				"  Arreglo: una entrada en afirmacionesDelProducto con el test que lo "+
				"ejerce, o con el motivo de que no lo haya.", k, cadenas[k])
		}
	}

	// DIRECCION 2: lo que esta en el registro y ya no afirma nada. Sin esto el
	// registro envejece igual que la prosa que vino a vigilar.
	huecos := 0
	for k, a := range afirmacionesDelProducto {
		v, ok := cadenas[k]
		if !ok {
			t.Errorf("el registro ata la cadena %q, que ya no existe en el catalogo.\n"+
				"  Arreglo: quitar la entrada, o devolver la cadena si se borro por error.", k)
			continue
		}
		if !reAfirmacion.MatchString(v) {
			t.Errorf("el registro ata la cadena %q, que ya no afirma nada sobre el "+
				"producto.\n  Texto: %q\n  Arreglo: quitar la entrada.", k, v)
		}
		switch {
		case a.test != "" && a.motivo != "":
			t.Errorf("%q tiene test Y motivo: son excluyentes, o se ata o se dice que no", k)
		case a.test != "":
			if _, hay := existen[a.test]; !hay {
				t.Errorf("%q dice que la ata %s, y ese test NO EXISTE.\n"+
					"  Un nombre con la forma de lo verificable es lo que hace que "+
					"nadie lo verifique.", k, a.test)
			}
		case a.motivo == "":
			t.Errorf("%q no tiene ni test ni motivo: un hueco sin declarar es un hueco "+
				"que se olvida", k)
		default:
			huecos++
		}
	}

	// EL CARDINAL, CON SU DENOMINADOR. Sin el denominador, «21 cadenas afirman»
	// no dice si el detector mira todo el catalogo o un rincon.
	t.Logf("%d cadenas de %d afirman algo sobre el producto, %d atadas a un test que lo "+
		"ejerce y %d declaradas sin comprobacion", len(afirman), len(cadenas),
		len(afirmacionesDelProducto)-huecos, huecos)

	if huecos != HuecosEsperados {
		t.Errorf("hay %d afirmaciones declaradas sin comprobacion y HuecosEsperados dice %d.\n"+
			"  Si ha SUBIDO, alguien ha aprobado esta puerta escribiendo un motivo en vez de "+
			"atar el comportamiento, que es su camino barato y cuesta una linea.\n"+
			"  Si ha BAJADO, se ha cerrado un hueco y hay que bajarlo aqui, en el mismo "+
			"commit.", huecos, HuecosEsperados)
	}
}

// TestElDetectorDeAfirmacionesSaltaCuandoDebe es el control negativo, en las dos
// direcciones. La primera entrada es la frase EXACTA que el comprador leyo en el
// binario publicado el 06-09-2026, siendo ya falsa.
func TestElDetectorDeAfirmacionesSaltaCuandoDebe(t *testing.T) {
	casos := []struct {
		texto  string
		afirma bool
	}{
		{"plazum todavía no guarda tus respuestas: viajan en la dirección.", true},
		{"plazum no manda estos avisos por su cuenta.", true},
		{"En seco no manda nada.", true},
		{"plazum guarda tus respuestas, pero de eso no se sigue que el trabajo se haya hecho.", true},
		{"Elige las normas que te alcanzan y pulsa guardar.", false},
		{"Un calendario sale de tus respuestas y del corpus instalado.", false},
		// LA TERCERA FORMA, la que absuelve, en las dos direcciones.
		{"Contestar que si aqui no activa ninguna obligacion nueva.", true},
		{"Ninguna obligacion dice depender de esta pregunta, asi que responderla no mueve nada.", true},
		{"3 preguntas no deciden nada todavia", true},
		{"Las marcadas no cambian hoy ninguna obligacion.", true},
		// Y lo que NO puede cazar: prosa que habla de obligaciones sin absolver
		// de nada. Sin estas, la tercera forma seria un comodin.
		{"Ninguna obligacion vence en los proximos doce meses.", false},
		{"Esta pregunta decide 5 obligaciones.", false},
		{"Mueve la fecha si el hecho cambia.", false},
	}
	for _, c := range casos {
		if got := reAfirmacion.MatchString(c.texto); got != c.afirma {
			t.Errorf("%q: detectado %v, esperaba %v", c.texto, got, c.afirma)
		}
	}
}
