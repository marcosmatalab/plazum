package pantallas

// LA PUERTA DE LA HOJA DE ESTILO: ninguna clase que una plantilla pinte se queda
// sin regla.
//
// # El agujero que cierra, y como se habia medido hasta hoy
//
// A MANO. La rebanada 1 del tramo 2 conto «3 clases usadas por estas dos
// plantillas y sin regla propia» (`.error` en `<p>`, `.particion` y `.sin-abrir`)
// y lo dejo escrito en docs/hallazgos-d11.md como P2. Un cardinal contado a mano
// es un cardinal que nadie vuelve a contar: la clase decimocuarta entra sin
// regla y nadie se entera, porque el HTML sigue siendo valido, el contraste
// sigue pasando y axe no dice nada. El unico sintoma es que la pagina se lee
// como un volcado de terminal, y eso no lo mira ninguna puerta de este arbol.
//
// El primer barrido con esta puerta encontro MAS de tres: las secciones enteras
// del calendario y del plan de avisos (`.cuenta`, `.trabajo`, `.faltas`,
// `.mandar`, `.derivacion-cifra`, `.descarte`) tampoco tenian ninguna regla. Las
// tres contadas eran las que se habian mirado, no las que habia.
//
// # Que NO comprueba, dicho para que nadie se fie de mas
//
// Que la regla sea BUENA. Comprueba que exista, que es la diferencia entre una
// clase que dispone algo y una clase que es un comentario en el HTML. Lo que se
// ve con los ojos sigue estando en las capturas de los dos temas.
//
// # Y la trampa que tiene, que es la de siempre
//
// Su fallo probable es acusar en falso, y por dos vias: una clase que aparece
// dentro de un comentario de plantilla, y una regla que existe pero escrita de
// una forma que este buscador no reconoce. Las dos se cierran igual: se recortan
// los comentarios ANTES de buscar (que es exactamente el arreglo que
// docs/hallazgos-d11.md pedia para la puerta de la raiz) y se busca el selector
// por delimitador y no por subcadena, para que `.error` no case con
// `.error-viejo`. Con su control negativo en las dos direcciones.

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// LAS PLANTILLAS QUE SE MIRAN. Son las de las superficies con pantalla, que son
// las que sirven esta hoja. Se escriben aqui y se cruzan con lo que hay en el
// disco: un directorio de plantillas nuevo que nadie anada a esta lista deja de
// vigilarse en silencio, asi que la lista tiene su propia guarda mas abajo.
var directoriosDePlantillas = []string{
	"plantillas",
	"../camino/plantillas",
	"../camino/armazon",
	"../acta/plantillas",
	"../calendario/plantillas",
	"../escalado/plantillas",
	"../uar/plantillas",
}

// LaHojaDeEstilo es donde vive la unica hoja del producto.
const laHojaDeEstilo = "estatico/plazum.css"

// reComentarioDePlantilla es un comentario de text/template, que NO llega al
// navegador.
//
// SE RECORTA ANTES DE BUSCAR. Un comentario que explique por que una clase se
// llama como se llama no pinta nada, y acusar por el es exactamente el falso
// positivo que docs/hallazgos-d11.md dejo anotado para la puerta de la raiz:
// «una puerta que acusa en falso se acaba borrando, y entonces no vigila nada».
var reComentarioDePlantilla = regexp.MustCompile(`(?s)\{\{-?\s*/\*.*?\*/\s*-?\}\}`)

// reComentarioHTML es un comentario de HTML. Mismo motivo.
var reComentarioHTML = regexp.MustCompile(`(?s)<!--.*?-->`)

// reAtributoClase saca el valor de un atributo class.
var reAtributoClase = regexp.MustCompile(`class="([^"]*)"`)

// reAccionDePlantilla es un `{{...}}` dentro del valor de un atributo class.
//
// Se quita entero: `class="pregunta{{if .Dormida}} dormida{{end}}"` pinta dos
// clases y una condicion, y la condicion no es una clase. Lo que queda tras
// quitarla son las palabras, que es lo que se busca.
var reAccionDePlantilla = regexp.MustCompile(`\{\{[^}]*\}\}`)

// clasesDeLasPlantillas devuelve todas las clases que se pintan, con el fichero
// donde salen.
func clasesDeLasPlantillas(t *testing.T) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	ficheros := 0
	for _, dir := range directoriosDePlantillas {
		entradas, err := filepath.Glob(filepath.Join(dir, "*.html"))
		if err != nil {
			t.Fatalf("glob de %s: %v", dir, err)
		}
		if len(entradas) == 0 {
			t.Errorf("el directorio de plantillas %q no tiene ni un .html.\n"+
				"  O se ha movido, y esta puerta ha dejado de mirarlo, o la lista de "+
				"directorios se ha quedado vieja", dir)
			continue
		}
		for _, f := range entradas {
			// #nosec G304 -- la ruta sale del glob de una lista escrita en este
			// mismo fichero, no de una entrada del usuario.
			b, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("leyendo %s: %v", f, err)
			}
			ficheros++
			cuerpo := string(b)
			cuerpo = reComentarioDePlantilla.ReplaceAllString(cuerpo, "")
			cuerpo = reComentarioHTML.ReplaceAllString(cuerpo, "")
			for _, m := range reAtributoClase.FindAllStringSubmatch(cuerpo, -1) {
				valor := reAccionDePlantilla.ReplaceAllString(m[1], " ")
				for _, c := range strings.Fields(valor) {
					out[c] = append(out[c], filepath.ToSlash(f))
				}
			}
		}
	}
	// SUELO: sin el, un glob que dejara de casar daria verde habiendo recorrido
	// la nada, que es el verde vacio de siempre.
	if ficheros < 8 {
		t.Fatalf("solo se han leido %d plantillas: esta puerta estaria mirando un trozo del "+
			"producto", ficheros)
	}
	if len(out) < 40 {
		t.Fatalf("solo se han encontrado %d clases distintas: el extractor no esta "+
			"funcionando", len(out))
	}
	return out
}

// tieneRegla dice si la hoja declara una regla para esa clase.
//
// SE BUSCA `.clase` SEGUIDO DE UN DELIMITADOR y no como subcadena: con
// subcadena, `.error` casaria con `.error-viejo` y una clase sin regla pasaria
// por vigilada. Los delimitadores son los que pueden seguir a un selector de
// clase en CSS.
func tieneRegla(hoja, clase string) bool {
	i := 0
	for {
		j := strings.Index(hoja[i:], "."+clase)
		if j < 0 {
			return false
		}
		i += j + len(clase) + 1
		if i >= len(hoja) {
			return false
		}
		switch hoja[i] {
		case ' ', ',', '{', ':', '\n', '\r', '\t', '>', '+', '~', '.', '[':
			return true
		}
	}
}

// reFamilia casa el prefijo de una familia de clases dentro de un selector.
//
// Se compone en caliente porque el prefijo sale de la plantilla; se escapa
// porque una clase puede llevar guiones, que en una expresion regular no son
// literales dentro de un conjunto.
func familiaConAlgunaRegla(hoja, prefijo string) bool {
	re := regexp.MustCompile(`\.` + regexp.QuoteMeta(prefijo) + `[A-Za-z0-9_-]+`)
	return re.MatchString(hoja)
}

func hoja(t *testing.T) string {
	t.Helper()
	// #nosec G304 -- ruta constante de este paquete.
	b, err := os.ReadFile(laHojaDeEstilo)
	if err != nil {
		t.Fatalf("leyendo la hoja de estilo: %v", err)
	}
	if len(b) < 5000 {
		t.Fatalf("la hoja mide %d bytes: esta puerta estaria mirando otra cosa", len(b))
	}
	return string(b)
}

// ClasesSinReglaEsperadas es cuantas clases se pintan sin que la hoja diga nada
// de ellas.
//
// EL CARDINAL VA CON IGUALDAD EXACTA Y EN LOS DOS SENTIDOS. Por arriba, porque
// una clase nueva sin regla es una pagina que se lee como un volcado de
// terminal y nadie se entera. Por abajo, porque un hueco que se cierra y deja su
// numero puesto miente hacia arriba para siempre.
const ClasesSinReglaEsperadas = 0

func TestNingunaClaseDeLasPlantillasSeQuedaSinRegla(t *testing.T) {
	css := hoja(t)
	clases := clasesDeLasPlantillas(t)

	var huerfanas []string
	for c, donde := range clases {
		if tieneRegla(css, c) {
			continue
		}
		// LAS FAMILIAS, que son clases a medias y no clases huerfanas.
		//
		// `class="estado e-{{.Estado}}"` deja `e-` cuando se quita la accion: no
		// es una clase, es el PREFIJO de una familia que la plantilla completa
		// con un dato (`e-aplica`, `e-pendiente`, `n-roto`). Se exige que la
		// hoja declare al menos una regla de esa familia, que es lo unico
		// comprobable desde aqui: cual de los valores llega depende del dato.
		//
		// NO ES UNA EXENCION: una familia sin ni una regla sigue saliendo
		// huerfana, y el control negativo de abajo lo recorre.
		if strings.HasSuffix(c, "-") && familiaConAlgunaRegla(css, c) {
			continue
		}
		sort.Strings(donde)
		huerfanas = append(huerfanas, c+"  ("+strings.Join(unicos(donde), ", ")+")")
	}
	sort.Strings(huerfanas)
	if len(huerfanas) != ClasesSinReglaEsperadas {
		t.Errorf("hay %d clases pintadas sin ninguna regla en la hoja y "+
			"ClasesSinReglaEsperadas dice %d.\n"+
			"  Una clase sin regla no rompe nada visible: el HTML sigue siendo valido, el "+
			"contraste sigue pasando y axe no dice nada. Lo unico que pasa es que esa parte de "+
			"la pagina se lee como un volcado de terminal.\n  %s",
			len(huerfanas), ClasesSinReglaEsperadas, strings.Join(huerfanas, "\n  "))
	}
}

func unicos(xs []string) []string {
	visto := map[string]bool{}
	var out []string
	for _, x := range xs {
		if !visto[x] {
			visto[x] = true
			out = append(out, x)
		}
	}
	return out
}

// CONTROL NEGATIVO EN LAS DOS DIRECCIONES.
//
// El fallo probable de esta puerta es acusar a lo que esta bien, y su fallo
// silencioso es dar por vigilada una clase que no lo esta. Las dos se prueban
// contra el buscador, que es donde vive la decision.
func TestElBuscadorDeReglasDistingueUnaReglaDeUnParecido(t *testing.T) {
	const css = `
p.error { color: red; }
.tarjeta, .cifra { padding: 1px; }
.filtro:hover { background: none; }
.marco-tabla > table { min-width: max-content; }
.error-viejo-que-ya-no-existe { color: blue; }
`
	for _, c := range []string{"error", "tarjeta", "cifra", "filtro", "marco-tabla"} {
		if !tieneRegla(css, c) {
			t.Errorf("la clase %q tiene regla y el buscador dice que no", c)
		}
	}
	// LA QUE NO DEBE CASAR: `.sin-regla` no esta, y `.error-viejo` NO hace que
	// `.error-viejo-que-ya-no-existe`... al reves: lo que se prueba es que una
	// clase que solo aparece como PREFIJO de otra no cuenta como vigilada.
	for _, c := range []string{"sin-regla", "err", "tarjet", "error-viejo"} {
		if tieneRegla(css, c) {
			t.Errorf("la clase %q no tiene regla propia y el buscador dice que si: una clase "+
				"sin regla estaria pasando por vigilada", c)
		}
	}

	// LA RAMA DE LAS FAMILIAS, con sus dos direcciones. Es la unica exencion
	// que esta puerta admite, asi que tiene que poder decir que no: una familia
	// que la hoja no declara en ningun valor sigue siendo huerfana.
	const conFamilia = `.e-aplica { color: green; } .e-pendiente { color: orange; }`
	if !familiaConAlgunaRegla(conFamilia, "e-") {
		t.Error("la familia `e-` tiene reglas y el detector dice que no: la exencion no " +
			"cubriria la clase que la plantilla compone con un dato")
	}
	if familiaConAlgunaRegla(conFamilia, "z-") {
		t.Error("la familia `z-` no tiene ni una regla y el detector la exime: una familia " +
			"entera sin declarar pasaria por vigilada")
	}
	// Y NO SE EXIME POR EL GUION SUELTO: `.e-` a secas no es una regla de nada.
	if familiaConAlgunaRegla(`.e- { color: red; }`, "e-") {
		t.Error("un selector `.e-` sin valor cuenta como familia declarada")
	}
}

// CONTROL NEGATIVO DEL RECORTE DE COMENTARIOS.
//
// Su fallo probable es el simetrico: llevarse por delante HTML de verdad y
// dejar de ver las clases que tenia que ver. Si eso pasara, esta puerta daria
// verde sobre una pagina entera sin reglas.
func TestElRecorteDeComentariosNoSeLlevaElHTMLQueVigila(t *testing.T) {
	const plantilla = `{{- /* aqui se explica class="inventada-en-un-comentario" */ -}}
<!-- y aqui tambien: class="inventada-en-html" -->
<div class="de-verdad otra{{if .X}} condicional{{end}}">x</div>`
	cuerpo := reComentarioHTML.ReplaceAllString(
		reComentarioDePlantilla.ReplaceAllString(plantilla, ""), "")
	var vistas []string
	for _, m := range reAtributoClase.FindAllStringSubmatch(cuerpo, -1) {
		valor := reAccionDePlantilla.ReplaceAllString(m[1], " ")
		vistas = append(vistas, strings.Fields(valor)...)
	}
	sort.Strings(vistas)
	quiero := "condicional de-verdad otra"
	if got := strings.Join(vistas, " "); got != quiero {
		t.Errorf("el extractor ve %q y tenia que ver %q.\n"+
			"  Si ve de menos, esta puerta da verde sobre clases que no ha mirado; si ve de "+
			"mas, acusa por una clase que vive dentro de un comentario y no pinta nada",
			got, quiero)
	}
}
