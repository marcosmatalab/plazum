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
}

// reAfirmacion caza las dos formas en que este catalogo habla del producto: con
// su nombre delante, y sin el cuando el sujeto se sobreentiende.
var reAfirmacion = regexp.MustCompile(
	`(?i)plazum (no |todav[ií]a |a[uú]n |guarda|sabe|elige|deja|presta|manda|env[ií]a)` +
		`|no manda nada|no se lo inventa`)

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

	t.Logf("%d cadenas afirman algo sobre el producto, %d atadas a un test que lo "+
		"ejerce y %d declaradas sin comprobacion", len(afirman),
		len(afirmacionesDelProducto)-huecos, huecos)
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
	}
	for _, c := range casos {
		if got := reAfirmacion.MatchString(c.texto); got != c.afirma {
			t.Errorf("%q: detectado %v, esperaba %v", c.texto, got, c.afirma)
		}
	}
}
