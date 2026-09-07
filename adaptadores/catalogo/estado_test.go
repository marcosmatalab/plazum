package catalogo

import (
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/estado"
)

// EL CATALOGO DICE DEL MOTIVO EXACTAMENTE LO QUE DICE EL MOTOR.
//
// # El defecto que esta puerta cierra, y por que ninguna existente lo veia
//
// `estado.Calcular` escribia el porque de cada estado en castellano y esa cadena
// viajaba CRUDA hasta la celda de la tabla de controles. La pagina inglesa
// imprimia espanol, y las puertas de i18n de este repositorio no podian verlo
// **porque todas vigilan el catalogo y aquello no era una clave**: la forma de
// publicar sin traducir aqui es no pasar por el catalogo.
//
// # Por que este test es hermano del del acta y no de los demas
//
// Las otras cadenas de la interfaz son de la interfaz: si alguien las reescribe,
// cambia un rotulo. Estas no. Son la DERIVACION del motor, que ademas sale por
// dos medios: el expediente la lleva en espanol resuelto (sin navegador delante
// que la traduzca) y la pantalla la pinta por clave. Si las dos se separan, el
// documento que verifica un auditor y la pagina que mira un CISO explican de
// forma distinta el mismo veredicto.
//
// Y ESTAS CADENAS LLEVAN HUECOS, a diferencia de las del acta: dos redacciones
// que digan lo mismo con los `%s` en distinto orden colocan un recurso donde va
// una fecha. Quien vigila eso NO es este test, y esta dicho abajo con su motivo.
//
// SE COMPARA SIN TILDES, por lo mismo que en el acta: las constantes de Go de
// este repositorio van sin tildes por convencion y el catalogo lleva espanol
// escrito para leerse. Lo que tiene que coincidir son las PALABRAS.
func TestElCatalogoDiceDelEstadoLoMismoQueElNucleo(t *testing.T) {
	c := nuevoParaTest(t)
	cadenas := estado.CadenasDelEstado()
	// El suelo es el numero real de hoy. Puesto en 1 solo protegeria del cero,
	// y lo que hace falta es enterarse cuando el conjunto MENGUA: una rama de
	// Calcular que pierda su clave vuelve a escribir frases.
	if len(cadenas) != 11 {
		t.Fatalf("nucleo/estado declara %d motivos y son once: si ha entrado una rama nueva "+
			"en Calcular, su cadena tiene que pasar por aqui; y si ha salido una, hay que "+
			"borrarla del catalogo", len(cadenas))
	}
	for _, f := range cadenas {
		es := c.Traducir("es", f.Clave)
		if es == f.Clave {
			t.Errorf("el catalogo no tiene %q, asi que la pantalla la pintaria en crudo",
				f.Clave)
			continue
		}
		if sinTildes(es) != sinTildes(f.Texto) {
			t.Errorf("la clave %q dice cosas distintas en el expediente y en la pantalla.\n"+
				"  nucleo:   %q\n  catalogo: %q\n"+
				"  Arreglo: copiar el texto de nucleo/estado a es.json. Y si lo que cambio "+
				"fue la constante de nucleo, MIRAR TAMBIEN EL INGLES: este test no lo puede "+
				"comprobar y su commit tiene que decir si se reviso y por que",
				f.Clave, f.Texto, es)
		}
		// EL INGLES EXISTE Y NO ES EL ESPANOL. No se puede comprobar que diga lo
		// mismo (para eso haria falta otra constante, y entonces habria dos
		// fuentes), pero si que alguien lo escribio: una clave que cae al idioma
		// por defecto deja la pagina inglesa con el motivo en espanol, que es
		// exactamente el defecto que esta puerta existe para impedir.
		en := c.Traducir("en", f.Clave)
		if en == f.Clave {
			t.Errorf("la clave %q no tiene ingles, asi que la pagina inglesa imprimiria "+
				"espanol: es el defecto entero, otra vez", f.Clave)
			continue
		}
		if sinTildes(en) == sinTildes(f.Texto) && !esCortaYComun(f.Texto) {
			t.Errorf("la clave %q tiene el mismo texto en ingles que en espanol (%q), asi que "+
				"o no se tradujo o se copio", f.Clave, en)
		}
	}

	// POR QUE AQUI NO SE COMPRUEBAN LOS HUECOS, que es lo primero que apetece
	// escribir y seria una TERCERA implementacion de la misma cifra.
	//
	// La primera version de este test contrastaba los `%s` del nucleo contra los
	// del ingles. Lo cazo la mutacion M5 de la pasada 2: al quitarle un hueco a
	// una traduccion se pusieron rojos DOS tests, el mio y
	// TestPuertaI18nElFormateoCasaEntreIdiomas, que ya existia y que compara los
	// verbos de TODA clave contra el idioma por defecto.
	//
	// Y es redundante de verdad, no por parecido: arriba se exige que el texto
	// del nucleo y el de `es.json` sean la MISMA cadena letra por letra, asi que
	// tienen los mismos huecos por construccion; aquella exige que `en.json`
	// tenga los mismos que `es.json`. El triangulo se cierra solo.
	//
	// Se dice aqui en vez de borrarlo en silencio porque un hueco que parece
	// hueco invita a taparlo otra vez, y taparlo es como se consigue que dos
	// implementaciones esten de acuerdo y la que mande sea la tercera.

	// LA OTRA DIRECCION: el catalogo no lleva motivos que el motor no escriba.
	//
	// Una huerfana aqui no es peso muerto: es una explicacion que alguien
	// redacto para un veredicto que ya no se da, y que sigue leyendose como si
	// el producto la usara.
	declaradas := map[string]bool{}
	for _, f := range cadenas {
		declaradas[f.Clave] = true
	}
	for _, k := range c.Claves() {
		if strings.HasPrefix(k, "evidencia.motivo.") && !declaradas[k] {
			t.Errorf("el catalogo lleva %q y nucleo/estado no la escribe en ninguna rama de "+
				"Calcular", k)
		}
	}
}
