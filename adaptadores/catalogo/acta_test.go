package catalogo

import (
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/acta"
)

// EL CATALOGO DEL ACTA DICE EN ESPANOL EXACTAMENTE LO QUE DICE EL NUCLEO.
//
// POR QUE ESTE TEST ES DISTINTO DE TODOS LOS DEMAS DEL CATALOGO. Las otras
// cadenas de la interfaz son de la interfaz: si alguien las reescribe, cambia un
// rotulo. Las de acta.* no: son el MISMO documento en otro medio. El board pack
// impreso lo compone nucleo/acta y sale sin navegador; la pantalla lo pinta con
// el catalogo. Si las dos versiones se separan, el papel y el navegador dicen
// cosas distintas del mismo acta, y una de las dos es la que firma el consejo.
//
// El caso caro no es un rotulo: es un DESCARGO. Un cubo que en papel dice "esto
// NO dice que se incumpla" y en pantalla dice otra cosa es exactamente el fallo
// que este producto no puede cometer ni una vez.
//
// SE COMPARA SIN TILDES, y no es dejadez: los identificadores y las constantes
// de Go de este repositorio van sin tildes por convencion, y el catalogo lleva
// espanol escrito para leerse. Lo que tiene que coincidir son las PALABRAS.
func TestElCatalogoDiceDelActaLoMismoQueElNucleo(t *testing.T) {
	c := nuevoParaTest(t)
	cadenas := acta.CadenasDelActa()
	if len(cadenas) < 60 {
		t.Fatalf("nucleo/acta declara %d cadenas y son muchas menos de las que tiene: o el "+
			"contrato se ha vaciado, o este test esta mirando otra cosa", len(cadenas))
	}
	for _, f := range cadenas {
		es := c.Traducir("es", f.Clave)
		if es == f.Clave {
			t.Errorf("el catalogo no tiene %q, asi que la pantalla la pintaria en crudo", f.Clave)
			continue
		}
		if sinTildes(es) != sinTildes(f.Texto) {
			t.Errorf("la clave %q dice cosas distintas en el papel y en la pantalla.\n"+
				"  nucleo:   %q\n  catalogo: %q\n"+
				"  Arreglo: copiar el texto de nucleo/acta a es.json. Y si lo que cambio fue "+
				"la frase de nucleo, MIRAR TAMBIEN EL INGLES: este test no lo puede comprobar "+
				"y su commit tiene que decir si se reviso y por que",
				f.Clave, f.Texto, es)
		}
		// Y EL INGLES EXISTE Y NO ES EL ESPANOL. No se puede comprobar que diga
		// lo mismo (para eso haria falta otra constante, y entonces habria dos
		// fuentes), pero si que alguien lo escribio: una clave que cae al idioma
		// por defecto deja la pantalla en ingles con el descargo en espanol, que
		// es media traduccion, que es la peor de las tres salidas.
		en := c.Traducir("en", f.Clave)
		if en == f.Clave {
			t.Errorf("la clave %q no tiene ingles", f.Clave)
			continue
		}
		if sinTildes(en) == sinTildes(f.Texto) && !esCortaYComun(f.Texto) {
			t.Errorf("la clave %q tiene el mismo texto en ingles que en espanol (%q), asi que "+
				"o no se tradujo o se copio", f.Clave, en)
		}
	}

	// LA OTRA DIRECCION: el catalogo no lleva cadenas del acta que el acta no
	// pida. Una huerfana aqui no es peso muerto como en el resto del catalogo:
	// es una frase que alguien escribio para este documento y que el documento
	// no dice, o sea un descargo que se cree puesto y no lo esta.
	declaradas := map[string]bool{}
	for _, f := range cadenas {
		declaradas[f.Clave] = true
	}
	for _, k := range c.Claves() {
		if strings.HasPrefix(k, "acta.") && !declaradas[k] {
			t.Errorf("el catalogo lleva %q y nucleo/acta no la declara", k)
		}
	}
}

// LOS TRECE DESCARGOS, uno a uno, y con la forma exigida en los dos idiomas.
//
// Trece y no catorce: la frase de lo que plazum no escribe (la de la seccion de
// la direccion) tiene la misma forma y el mismo cuidado, pero no va pegada a
// ningun numero, asi que no es un descargo, es un parrafo. Se cuenta aparte para
// que "catorce frases que no acusan" no se convierta en "trece cubos" sin que
// nadie lo note.
//
// Es el hermano del test de nucleo que comprueba que ningun descargo acusa. Aqui
// se comprueba lo mismo del INGLES, que es donde no hay constante que lo ate: su
// garantia es de revision, no mecanica, y lo unico mecanico que se puede exigir
// es la forma. El patron son dos mitades, "esto NO dice <lo que se leeria mal>"
// y "dice que <lo que consta>", y una traduccion que se coma la primera
// convierte el descargo en la acusacion.
func TestNingunDescargoDelActaAcusaEnNingunIdioma(t *testing.T) {
	c := nuevoParaTest(t)
	descargos := acta.DescargosDelActa()
	if len(descargos) != 13 {
		t.Fatalf("son trece descargos y este test ve %d: si ha entrado uno nuevo, tiene que "+
			"pasar por aqui", len(descargos))
	}
	formas := map[string][2]string{
		"es": {"esto no dice", ": dice que"},
		"en": {"this does not say", ": it says"},
	}
	for _, d := range descargos {
		for _, idioma := range []string{"es", "en"} {
			v := strings.ToLower(sinTildes(c.Traducir(idioma, d.Clave)))
			f := formas[idioma]
			if !strings.HasPrefix(v, f[0]) {
				t.Errorf("[%s] %s: el descargo no empieza negando lo que se podria leer mal, "+
					"asi que se lee como la acusacion: %q", idioma, d.Clave, v)
			}
			if !strings.Contains(v, f[1]) {
				t.Errorf("[%s] %s: el descargo niega y no dice que es lo que si consta, que es "+
					"la mitad util: %q", idioma, d.Clave, v)
			}
		}
	}
	// Control negativo: el detector sabe decir que no.
	if strings.HasPrefix("You failed to audit these obligations.", "this does not say") {
		t.Fatal("el detector da por bueno un texto que acusa")
	}
}

// esCortaYComun deja pasar los rotulos de una o dos palabras que legitimamente
// se escriben igual en los dos idiomas. Sin esto, el test exigiria traducir
// "sha256" o inventarse sinonimos.
func esCortaYComun(s string) bool { return len(strings.Fields(s)) <= 2 }

// sinTildes normaliza para comparar palabras y no acentos. Los identificadores y
// las constantes de este repositorio van sin tildes por convencion, y el
// catalogo lleva espanol escrito para leerse.
// Se hace con un Replacer y no con golang.org/x/text porque el nucleo tiene cero
// dependencias y anadir una al arbol para quitar cinco tildes seria pagar una
// linea de DEPENDENCIAS.md por nada. Es el mismo de superficies/uar.
func sinTildes(s string) string {
	r := strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o",
		"ú", "u", "ñ", "n", "Á", "A", "É", "E", "Í", "I",
		"Ó", "O", "Ú", "U", "Ñ", "N")
	return r.Replace(s)
}
