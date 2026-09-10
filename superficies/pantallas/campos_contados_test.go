package pantallas

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// LO QUE LA SECCION DE CAMPOS NO ENSENA POR DEFECTO SE CUENTA Y SE PUEDE ABRIR.
//
// # Por que existe, y contra que
//
// El 10-09-2026 la seccion de campos de /alcance paso a enseñar por defecto solo
// los OBLIGADOS, que son los que hay que rellenar, y a contar el resto. Eso baja
// el TTFV de 15m4s a 14m38s, o sea que CIERRA una casilla.
//
// Y ahi esta el riesgo, porque es exactamente la forma de aprobar una medida sin
// arreglar nada: vaciar la seccion tambien habria bajado el numero, y mas. La
// diferencia entre las dos cosas es UNA: que el descarte se cuente y se pueda
// deshacer. Sin esta puerta, esa diferencia vive en la buena intencion de quien
// edite la plantilla el mes que viene.
//
// # Las tres afirmaciones, y la tercera es la que cierra el camino barato
//
//	el cardinal esta      hay un numero a la vista que dice cuantos no se pintan
//	el cardinal cuadra    pintados + contados == todos los que declara el corpus
//	abrirlos los trae     pedir la seccion entera trae MAS fichas que el defecto
//
// Sin la segunda, un contador que dijera cualquier cosa dejaria la pagina
// pareciendo honesta sin serlo. Sin la tercera, se podria contar bien y no tener
// donde abrirlos, que es esconder con un numero delante.
func TestLosCamposQueNoSeEnsenanPorDefectoSeCuentanYSePuedenAbrir(t *testing.T) {
	// CON EL CORPUS REAL: el sintetico tiene tres campos y todos obligados, asi
	// que no habria nada oculto y las tres afirmaciones pasarian sobre el vacio.
	s, _ := superficie(t, corpusReal(t))
	_, defecto := pedir(t, s, "/alcance")
	_, todos := pedir(t, s, "/alcance?"+ParamCampos+"="+VerTodas)

	fichas := func(cuerpo string) int {
		return strings.Count(cuerpo, `<span class="entidad">`)
	}
	pintados, completos := fichas(defecto), fichas(todos)
	if completos == 0 {
		t.Fatal("la seccion de campos sale vacia hasta pidiendola entera: este recorrido " +
			"estaria comprobando una pagina sin campos, que es donde cualquier cuenta cuadra")
	}

	// EL CARDINAL, leido de la pagina y no del modelo: lo que importa es lo que
	// lee quien abre la pantalla.
	m := reCardinalDeCamposOcultos.FindStringSubmatch(defecto)
	if m == nil {
		t.Fatalf("la vista por defecto no trae el cardinal de los campos que no ensena.\n"+
			"  Pinta %d fichas de %d y no dice que falten: eso es esconder, no resumir.",
			pintados, completos)
	}
	ocultos, err := strconv.Atoi(m[1])
	if err != nil {
		t.Fatalf("el cardinal no es un numero: %q", m[1])
	}

	if pintados+ocultos != completos {
		t.Errorf("la seccion pinta %d campos y dice que oculta %d, y en total son %d.\n"+
			"  Un contador que diga cualquier numero deja la pantalla pareciendo honesta sin "+
			"serlo: es la misma ley de conservacion que se exige el calendario.",
			pintados, ocultos, completos)
	}
	if ocultos == 0 {
		t.Error("el cardinal dice cero, asi que esta puerta estaria comprobando que se " +
			"cuenta un descarte que no existe")
	}
	if completos <= pintados {
		t.Errorf("pedir la seccion entera trae %d fichas y el defecto ya traia %d: lo que "+
			"no se pinta no se puede abrir, y entonces el numero de al lado solo dice que "+
			"algo falta", completos, pintados)
	}
	t.Logf("campos: %d pintados por defecto, %d contados, %d en total", pintados, ocultos,
		completos)
}

// reCardinalDeCamposOcultos lee el numero que la pagina pone al lado del enlace.
//
// Se ancla en la CLAVE de catalogo y no en el texto: el catalogo de pruebas
// imprime «[es:clave] args», asi que la puerta no depende de como este redactada
// la frase. Un literal del catalogo real se quedaria viejo el dia que alguien
// mejore la redaccion, y entonces esto buscaria algo que no existe.
var reCardinalDeCamposOcultos = regexp.MustCompile(
	`<span class="cardinal">\[es:alcance\.campos\.ocultos\] (\d+)</span>`)

// TestLaSeccionDeCamposNoSeVaciaParaAprobarLaMedida es la otra mitad: la seccion
// tiene que SEGUIR ENTREGANDO en la vista por defecto.
//
// Es la marca estructural del TTFV aplicada a una seccion en vez de a un paso: un
// numero baja igual de bien vaciando que resumiendo, y solo una de las dos cosas
// es mejorar el producto.
func TestLaSeccionDeCamposNoSeVaciaParaAprobarLaMedida(t *testing.T) {
	s, _ := superficie(t, corpusReal(t))
	_, defecto := pedir(t, s, "/alcance")
	n := strings.Count(defecto, `<span class="entidad">`)
	if n == 0 {
		t.Fatal("la vista por defecto no pinta ni una ficha de campo.\n" +
			"  Un contador solo, sin nada que contar al lado, es la seccion vaciada con un " +
			"numero puesto encima para que el TTFV baje.")
	}
	t.Logf("la vista por defecto sigue entregando %d fichas de campo", n)
}
