package catalogo

import (
	"sort"
	"strings"
	"testing"
	"unicode"
)

// EL INGLES DEL CATALOGO ES BRITANICO, Y ESO ES UNA DECISION, NO UNA COSTUMBRE.
//
// Las cadenas del acta salieron en britanico (programme, organisation) sin que
// nadie lo escribiera en ningun sitio. Es la eleccion correcta para el
// comprador de plazum, que es europeo y lee normas escritas asi, pero una
// eleccion que no esta escrita no es una eleccion: es lo que salio, y lo
// siguiente que salga puede salir distinto. Un catalogo mitad britanico y mitad
// americano no se lee como una decision, se lee como que nadie mira.
//
// La decision esta escrita en docs/traducir.md seccion 8 y en el godoc de este
// paquete. Esto es lo que hace que la cadena numero 45 no llegue en americano
// POR DESCUIDO, que es distinto de que no llegue por acuerdo: una regla que solo
// vive en un documento se cumple mientras alguien se acuerde de ella.
//
// SE MIRAN LOS VALORES Y NO LAS CLAVES, y no es un detalle. La clave es un
// identificador y va en castellano sin tildes por convencion de la casa, asi
// que "acta.pantalla.organizacion" es correcta y su valor "Organisation"
// tambien. Un detector que mirase la linea entera se pondria rojo en la primera
// clave que hable de la organizacion, que es el falso positivo obvio de esta
// puerta y el motivo de que aqui se lea la tabla y no el fichero.
//
// LA LISTA ES CERRADA Y CORTA A PROPOSITO. La regla general del sufijo (-ize
// frente a -ise) no se puede escribir como subcadena: "size" y "seize" la
// cumplen y no son verbos. Se enumeran las palabras que de verdad aparecen en
// prosa de cumplimiento, y anadir una es una linea aqui.
var americanismos = map[string]string{
	"organization":   "organisation",
	"organizations":  "organisations",
	"organizational": "organisational",
	"authorization":  "authorisation",
	"authorizations": "authorisations",
	"authorize":      "authorise",
	"authorized":     "authorised",
	"authorizes":     "authorises",
	"recognize":      "recognise",
	"recognized":     "recognised",
	"recognizes":     "recognises",
	"prioritize":     "prioritise",
	"prioritized":    "prioritised",
	"minimize":       "minimise",
	"minimized":      "minimised",
	"maximize":       "maximise",
	"maximized":      "maximised",
	"summarize":      "summarise",
	"summarized":     "summarised",
	"standardize":    "standardise",
	"standardized":   "standardised",
	"normalize":      "normalise",
	"normalized":     "normalised",
	"analyze":        "analyse",
	"analyzed":       "analysed",
	"analyzes":       "analyses",
	"behavior":       "behaviour",
	"behaviors":      "behaviours",
	"behavioral":     "behavioural",
	"center":         "centre",
	"centers":        "centres",
	"defense":        "defence",
	"defenses":       "defences",
	"fulfill":        "fulfil",
	"fulfills":       "fulfils",
	"enrollment":     "enrolment",
	"canceled":       "cancelled",
	"labeled":        "labelled",
	"modeled":        "modelled",
	"catalog":        "catalogue",
	"catalogs":       "catalogues",
	// "program" es britanico legitimo cuando significa un programa de
	// ordenador. En plazum no significa eso ni una vez: el programa es el de
	// auditoria interna (9.2), que en britanico es "programme". Si algun dia
	// hace falta hablar de software, la salida es reescribir la frase, no
	// quitar la entrada, porque quitarla abre la puerta al otro sentido.
	"program":  "programme",
	"programs": "programmes",
}

// palabras parte un texto en palabras en minusculas. Se corta por lo que no es
// letra, asi que "programme," y "programme" son la misma palabra y "programming"
// no es "program". El apostrofo no corta, para que "auditor's" no se lea como
// dos.
func palabras(s string) []string {
	campos := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && r != '\''
	})
	return campos
}

func TestElInglesDelCatalogoEsBritanico(t *testing.T) {
	c := nuevoParaTest(t)
	claves := c.Claves()
	if len(claves) < 200 {
		t.Fatalf("el catalogo declara %d claves y son muchas menos de las que tiene: o se ha "+
			"vaciado, o este test esta mirando otra cosa", len(claves))
	}
	vistas := 0
	for _, k := range claves {
		// Se lee el valor CRUDO y no Traducir: una cadena con plural lleva sus
		// dos formas separadas por barra y Traducir sin contador devuelve solo
		// la ultima. El singular es justo la forma que menos se prueba.
		v, ok := c.valor("en", k)
		if !ok {
			// Un hueco en ingles lo caza la puerta de i18n. Aqui no se dobla.
			continue
		}
		for _, p := range palabras(v) {
			vistas++
			if br, malo := americanismos[p]; malo {
				t.Errorf("%s: %q es la grafia americana, y el ingles de plazum es britanico.\n"+
					"  Arreglo: %q. La decision y su porque estan en docs/traducir.md seccion 8.\n"+
					"  Cadena: %q", k, p, br, v)
			}
		}
	}
	if vistas < 2000 {
		t.Fatalf("se han mirado %d palabras del ingles y son muchas menos de las que hay: el "+
			"lector de la tabla no esta leyendo", vistas)
	}

	// CONTROL NEGATIVO EN LAS DOS DIRECCIONES, y las dos hacen falta.
	//
	// Que caza lo americano: sin esto, un mapa vacio da verde eterno.
	// Que NO acusa a lo britanico: es el fallo probable de esta puerta, porque
	// las dos grafias se parecen y una entrada mal copiada acusaria justo a la
	// que queremos.
	cazadas := map[string]bool{}
	for _, p := range palabras("The organization runs an audit program and analyzes behavior.") {
		if _, malo := americanismos[p]; malo {
			cazadas[p] = true
		}
	}
	for _, quiero := range []string{"organization", "program", "analyzes", "behavior"} {
		if !cazadas[quiero] {
			t.Errorf("el detector da por bueno %q, que es americano", quiero)
		}
	}
	for _, p := range palabras("The organisation runs an audit programme, analyses behaviour, " +
		"seizes the size of the prize and reprogrammes nothing.") {
		if br, malo := americanismos[p]; malo {
			t.Errorf("el detector acusa a %q, que es britanico, y propone %q", p, br)
		}
	}
}

// LA LISTA NO SE CONTRADICE A SI MISMA: ninguna forma britanica que propone
// esta ademas en el lado acusado. Una entrada mal copiada
// ("organisation": "organisation") daria un rojo eterno e imposible de
// arreglar escribiendo bien, que es la peor clase de puerta.
func TestNingunaGrafiaBritanicaEstaEnElLadoAcusado(t *testing.T) {
	var malas []string
	for us, br := range americanismos {
		if us == br {
			malas = append(malas, us+" se propone a si misma")
			continue
		}
		if _, tambien := americanismos[br]; tambien {
			malas = append(malas, br+" se propone y ademas se acusa")
		}
	}
	sort.Strings(malas)
	if len(malas) > 0 {
		t.Errorf("la lista de americanismos se contradice: %v", malas)
	}
}
