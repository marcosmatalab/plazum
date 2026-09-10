package corpus

import "testing"

// EL TEXTO EN OTRA LENGUA DICE SI ES LA QUE SE PIDIO, y las dos ramas hacen
// falta.
//
// Sin el segundo valor, una superficie que sirve la pagina en ingles no puede
// saber si el parrafo que va a pintar esta en ingles, y la unica forma de
// averiguarlo seria comparar cadenas, que no hace nadie. El sintoma seria la
// pagina inglesa enseñando derecho en castellano SIN decirlo, que es exactamente
// el agujero que D-25 vino a cerrar.
//
// LAS TRES FORMAS DE LA NADA (invariante 8), cada una con su caso:
//
//	lengua VACIA          no se pidio nada, asi que no puede contestarse «te di
//	                      lo que pediste». Castellano con hay=false.
//	lengua SIN VERSION    presente y sin nada detras: cae al castellano
//	                      DICIENDOLO, porque un texto en la lengua equivocada es
//	                      mejor que ninguno, y callarlo no.
//	version PRESENTE Y VACIA  el mapa la trae pero su texto es "": es la forma
//	                      que sale de una ingesta a medias, y NO puede contarse
//	                      como version buena. Cae al castellano con hay=false.
func TestElTextoEnOtraLenguaDiceSiEsLaQueSePidio(t *testing.T) {
	o := Obligacion{
		TextoLegal: "el texto en castellano",
		VersionesLinguisticas: map[string]VersionLinguistica{
			"en": {Texto: "the english text", Enlace: "e", Celex: "c",
				Consultado: "2026-09-10"},
			// Una version presente y VACIA, que es la que sale de una ingesta a
			// medias. Tiene que comportarse como si no estuviera.
			"fr": {Texto: "", Enlace: "e", Celex: "c", Consultado: "2026-09-10"},
		},
	}

	for _, c := range []struct {
		nombre, pido, quieroTexto string
		quieroHay                 bool
		porQueImporta             string
	}{
		{
			nombre: "la lengua que hay", pido: "en",
			quieroTexto: "the english text", quieroHay: true,
			porQueImporta: "es la casilla entera: el derecho de la UE en ingles",
		},
		{
			nombre: "el castellano, que es la del texto legal", pido: "es",
			quieroTexto: "el texto en castellano", quieroHay: true,
			porQueImporta: "pedir la lengua del texto legal SI es que te dieron la que pediste",
		},
		{
			nombre: "una lengua sin version cae al castellano DICIENDOLO", pido: "de",
			quieroTexto: "el texto en castellano", quieroHay: false,
			porQueImporta: "si dijera true, la pagina alemana pintaria castellano sin avisar",
		},
		{
			nombre: "una version presente y VACIA no cuenta como version", pido: "fr",
			quieroTexto: "el texto en castellano", quieroHay: false,
			porQueImporta: "una ingesta a medias deja el hueco, y un hueco que se " +
				"anuncia como version es peor que no tenerla",
		},
		{
			nombre: "la lengua vacia no es una peticion", pido: "",
			quieroTexto: "el texto en castellano", quieroHay: false,
			porQueImporta: "no se pidio nada, asi que no puede contestarse que se dio " +
				"lo que se pidio",
		},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			texto, hay := o.TextoEnLengua(c.pido)
			if texto != c.quieroTexto {
				t.Errorf("pidiendo %q salio %q y se esperaba %q. Por que importa: %s",
					c.pido, texto, c.quieroTexto, c.porQueImporta)
			}
			if hay != c.quieroHay {
				t.Errorf("pidiendo %q el segundo valor es %v y se esperaba %v. "+
					"Por que importa: %s", c.pido, hay, c.quieroHay, c.porQueImporta)
			}
		})
	}

	// Y EL VALOR CERO DE LA OBLIGACION: sin versiones y sin texto, no se inventa
	// nada y sobre todo no se dice que si.
	var vacia Obligacion
	if texto, hay := vacia.TextoEnLengua("en"); texto != "" || hay {
		t.Errorf("una obligacion sin texto devolvio (%q, %v) y tenia que devolver "+
			"(\"\", false): un referencial no tiene texto legal y decir que si lo tiene "+
			"en ingles seria inventarselo", texto, hay)
	}
}
