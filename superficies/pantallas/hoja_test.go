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

// UsoDeUnaClase es como una plantilla usa una clase.
//
// # Por que hace falta distinguir, y lo dijo una mutacion
//
// Hay dos poblaciones de clases y necesitan reglas de forma distinta:
//
//	LA QUE VA SOLA        `<p class="error">`. Necesita una regla que la
//	                      alcance SOLA. `.principal.error` la menciona y no la
//	                      alcanza, porque exige las dos clases a la vez: es
//	                      exactamente la confusion que docs/hallazgos-d11.md
//	                      dejo anotada sobre esta misma clase.
//	LA QUE MODIFICA       `class="principal hoy"`, `class="pregunta dormida"`.
//	                      Existe PARA combinarse, asi que un selector compuesto
//	                      (`.principal.hoy`) es su regla correcta, y exigirle
//	                      una regla suelta seria pedir lo contrario de lo que
//	                      hace.
//
// La diferencia no se decide a ojo: se lee de la plantilla. Una clase que en
// algun sitio es la UNICA de su elemento es de la primera poblacion.
type UsoDeUnaClase struct {
	// Donde son los ficheros en los que sale.
	Donde []string
	// AlgunaVezSola dice si en algun elemento es la unica clase.
	AlgunaVezSola bool
}

// clasesDeLasPlantillas devuelve todas las clases que se pintan, con el fichero
// donde salen y con como se usan.
func clasesDeLasPlantillas(t *testing.T) map[string]*UsoDeUnaClase {
	t.Helper()
	out := map[string]*UsoDeUnaClase{}
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
				// LA ACCION SE SUSTITUYE POR UN ESPACIO Y NO SE BORRA, y eso
				// importa aqui: `class="pregunta{{if .X}} dormida{{end}}"`
				// pinta a veces una clase y a veces dos, asi que `pregunta` NO
				// cuenta como «sola». Lo que la accion pueda anadir se cuenta
				// como acompanamiento, que es el lado conservador: pedir regla
				// suelta de mas es un rojo que se lee, darla por acompanada de
				// mas es una clase sin regla que pasa.
				partes := strings.Fields(reAccionDePlantilla.ReplaceAllString(m[1], " x"))
				soloUna := len(partes) == 1
				for _, c := range strings.Fields(
					reAccionDePlantilla.ReplaceAllString(m[1], " ")) {
					u := out[c]
					if u == nil {
						u = &UsoDeUnaClase{}
						out[c] = u
					}
					u.Donde = append(u.Donde, filepath.ToSlash(f))
					if soloUna {
						u.AlgunaVezSola = true
					}
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

// tieneRegla dice si la hoja declara una regla que alcance a esa clase SOLA.
//
// # Las dos formas de decir que si cuando la respuesta es no
//
//  1. POR DELANTE: la subcadena. `.error` casaria dentro de `.error-viejo`, y una
//     clase sin regla pasaria por vigilada. Se exige un DELIMITADOR detras: lo
//     que puede seguir a un selector de clase en CSS.
//
//  2. POR DETRAS: el selector COMPUESTO, y esta la encontro una mutacion sobre
//     esta misma puerta. `.principal.error` menciona `.error` y NO alcanza a un
//     `<p class="error">`: exige las DOS clases a la vez. Contarla es exactamente
//     la confusion que docs/hallazgos-d11.md dejo anotada («en plazum.css no hay
//     ninguna regla para .error salvo .principal.error, que es otra cosa»), o sea
//     que la puerta reproducia el error que venia a vigilar. Se rechaza cuando lo
//     que hay pegado por delante es OTRA clase; un elemento (`p.error`) si vale,
//     porque alcanza a todo `<p>` con esa clase, que es el caso de verdad.
func tieneRegla(hoja, clase string) bool {
	i := 0
	for {
		j := strings.Index(hoja[i:], "."+clase)
		if j < 0 {
			return false
		}
		inicio := i + j
		i = inicio + len(clase) + 1
		if i >= len(hoja) {
			return false
		}
		if !esDelimitadorDeSelector(hoja[i]) {
			continue
		}
		if calificadaPorOtraClase(hoja, inicio) {
			continue
		}
		return true
	}
}

// mencionada dice si la hoja nombra esa clase en algun selector, compuesto o no.
//
// Es la comprobacion de las clases MODIFICADORAS: su regla correcta es
// `.principal.hoy`, y exigirles una regla suelta seria pedir lo contrario de lo
// que hacen. Sigue exigiendo el delimitador, asi que `.tabla` no se da por
// declarada porque exista `.tabla-vieja`.
func mencionada(hoja, clase string) bool {
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
		if esDelimitadorDeSelector(hoja[i]) {
			return true
		}
	}
}

// esDelimitadorDeSelector dice si ese byte puede seguir a un selector de clase.
func esDelimitadorDeSelector(b byte) bool {
	switch b {
	case ' ', ',', '{', ':', '\n', '\r', '\t', '>', '+', '~', '.', '[':
		return true
	}
	return false
}

// calificadaPorOtraClase mira hacia ATRAS desde el punto de un selector de clase
// y dice si lo que hay pegado es otra clase.
//
// Se anda hacia atras sobre los caracteres que pueden formar un nombre de clase;
// si lo primero que aparece antes es un punto, el selector es compuesto
// (`.principal.error`) y no alcanza a la clase sola.
func calificadaPorOtraClase(hoja string, puntoDeLaClase int) bool {
	k := puntoDeLaClase - 1
	for k >= 0 {
		c := hoja[k]
		esNombre := c == '-' || c == '_' ||
			(c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
		if !esNombre {
			break
		}
		k--
	}
	return k >= 0 && hoja[k] == '.'
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
	for c, uso := range clases {
		// LA QUE VA SOLA exige una regla que la alcance sola; la que solo
		// modifica se conforma con que la hoja la NOMBRE, porque su regla
		// correcta es un selector compuesto.
		if uso.AlgunaVezSola {
			if tieneRegla(css, c) {
				continue
			}
		} else if mencionada(css, c) {
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
		sort.Strings(uso.Donde)
		como := "modifica a otra"
		if uso.AlgunaVezSola {
			como = "va sola, asi que necesita regla propia"
		}
		huerfanas = append(huerfanas,
			c+"  ["+como+"]  ("+strings.Join(unicos(uso.Donde), ", ")+")")
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

	// EL SELECTOR COMPUESTO, que es la forma de decir que si cuando es que no.
	// Lo encontro una mutacion sobre esta puerta: `.principal.error` menciona
	// `.error` y no alcanza a un `<p class="error">`, porque exige las dos
	// clases a la vez. Contarla era reproducir la confusion que esta puerta
	// existe para vigilar.
	const soloCompuesta = `.vacio, .principal.vacia, .principal.error { max-width: 60ch; }`
	if tieneRegla(soloCompuesta, "error") {
		t.Error("`.principal.error` cuenta como regla de `.error`, y no alcanza a un " +
			"<p class=\"error\">: exige las dos clases a la vez")
	}
	// LA OTRA DIRECCION, y sin ella lo de arriba se cumpliria rechazandolo todo:
	// la primera clase de un selector compuesto SI esta alcanzada, un selector
	// con elemento delante tambien, y uno descendente tambien.
	for _, c := range []string{"vacio", "principal"} {
		if !tieneRegla(soloCompuesta, c) {
			t.Errorf("la clase %q si esta alcanzada por %q y el buscador dice que no",
				c, soloCompuesta)
		}
	}
	if !tieneRegla(`p.error { color: red; }`, "error") {
		t.Error("`p.error` no cuenta como regla de `.error`, y alcanza a todo <p> con esa clase")
	}
	if !tieneRegla(`.resumen .e-aplica .n { color: green; }`, "e-aplica") {
		t.Error("un selector descendente no cuenta como regla de la clase que nombra")
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
