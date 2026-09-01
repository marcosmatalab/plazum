package catalogo

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// La frontera legal del catalogo, escrita en codigo y no en un comentario.
//
// Lo que defiende: que por el catalogo no se cuele texto de una norma. Ese
// texto tiene una licencia y una fuente propias (art. 13 TRLPI, Decision
// 2011/833/UE para el BOE y el DOUE; nada transcribible en ISO, PCI DSS, SOC 2,
// TISAX o CIS), y una version nuestra en otro idioma ya no es la norma, es obra
// nuestra presentada como si lo fuera. El idioma del corpus va por paquete.
//
// Como se defiende, en tres capas y por orden de fuerza:
//
//  1. ESPACIO DE NOMBRES CERRADO. Una clave vive en uno de los espacios de
//     abajo o no vive. Eso deja fuera de raiz una clave con nombre de norma,
//     que ademas seria una norma cableada en el sentido del invariante 2.
//  2. FORMA DE ROTULO. Una cadena de interfaz es corta, de una sola linea, sin
//     marcado y sin identificadores de paquete. Un parrafo largo aqui casi
//     siempre es texto de una norma.
//  3. TRIPWIRE DE CITA. Los patrones de abajo cazan lo que suena a norma
//     ("articulo 31", "Real Decreto", "Reglamento (UE)", "Annex I").
//
// Lo que estas tres capas NO demuestran, dicho para que conste: que una frase
// concreta no salga del corpus instalado. Eso no se puede saber sin mirar el
// corpus, y mirarlo en tiempo de ejecucion seria atar el catalogo a los
// paquetes. Se comprueba en frontera_test.go, contrastando cada valor contra el
// texto legal de todos los paquetes por trozos de seis palabras, con su control
// negativo. Y lo que se le escapa a eso tambien esta dicho alli.

// maxRunasValor: un rotulo de interfaz no es un parrafo.
//
// El numero se elige con el vecino delante: el linter de corpus corta en 120
// caracteres el texto normativo que puede llevar un paquete referencial. Aqui
// se admite el doble, porque un mensaje de error accionable (causa, arreglo y
// cita) no cabe en 120, pero un articulo de ley tampoco cabe en 240.
const maxRunasValor = 240

// espaciosDeClave es la lista CERRADA de familias de cadenas de la interfaz.
//
// Ampliarla es una decision consciente: un espacio nuevo es una familia de
// rotulos nueva. Y jamas puede ser el identificador de una norma, porque el
// corpus no entra en el catalogo.
//
// La lista sale de lo que la interfaz pide de verdad, que declara
// superficies/pantallas.ClavesDeCatalogo(). No se inventan espacios "por si
// acaso": un espacio sin cadenas dentro es una invitacion a meter ahi lo que no
// cabe en ningun sitio.
var espaciosDeClave = []string{
	"ui",         // marco de la pagina: marca, salto de accesibilidad, pie
	"pantalla",   // titulos y explicaciones de vacio de las pantallas derivadas
	"menu",       // navegacion entre pantallas
	"origen",     // de donde sale el contenido de una pantalla
	"vacia",      // que hacer cuando una pantalla no tiene contenido
	"alcance",    // la entrevista y su derivacion a un clic
	"derivacion", // el por que de cada veredicto
	"estado",     // el veredicto de aplicabilidad de una obligacion
	"filtro",     // filtros de las tablas
	"tabla",      // tablas de Controles y Certificados
	"columna",    // cabeceras de columna
	"error",      // errores accionables de la peticion
	"aviso",      // avisos de la herramienta al operador
	// uar: la revision de accesos, que es la PRIMERA superficie que muta.
	//
	// Se anade el 01-09-2026 como decision consciente, que es lo que esta lista
	// pide. Es una familia de rotulos nueva de verdad y no un sinonimo de las de
	// arriba: aquellas rotulan pantallas que solo LEEN, y estas rotulan
	// formularios que cambian estado (que se hace con un acceso, por que se
	// revoca, a quien se delega, que se excusa y con que motivo).
	//
	// Y no es el identificador de ninguna norma: "revision de accesos" es una
	// practica que piden varios marcos con nombres distintos, no un marco. El
	// dia que una clave de aqui empiece a oler a articulo, la caza el mismo
	// tripwire que a las demas.
	"uar",
}

var (
	formaDeClave = regexp.MustCompile(`^[a-z][a-z0-9]*(\.[a-z0-9_]+)+$`)

	// marcadoHTML se busca en el valor CRUDO. El escapado es trabajo de
	// html/template, pero un rotulo no necesita marcado para nada, y una
	// cadena con etiquetas dentro es una invitacion a que alguien la pinte
	// con template.HTML y se lleve un XSS almacenado en el catalogo.
	marcadoHTML = regexp.MustCompile(`(?i)<\s*[a-z/!?]|&#|&lt;|javascript:|data:text/html`)

	// marcadoresNormativos se buscan sobre el texto NORMALIZADO (minusculas,
	// sin tildes, un espacio por separador), asi que "art. 31" llega como
	// "art 31" y "Reglamento (UE) 2016/679" como "reglamento ue 2016 679".
	//
	// Piden numero a proposito donde pueden: "Ver el articulo" es un boton
	// legitimo, "articulo 31" es una cita. Un detector que grita por todo se
	// acaba desactivando, que es peor que no tenerlo.
	marcadoresNormativos = []*regexp.Regexp{
		regexp.MustCompile(`\bart(?:iculos|iculo)? \d`),
		regexp.MustCompile(`\bapartad[oa]s? \d`),
		regexp.MustCompile(`\banexo [ivx0-9]`),
		regexp.MustCompile(`\bdisposicion (adicional|transitoria|final|derogatoria)\b`),
		regexp.MustCompile(`\breal decreto\b`),
		regexp.MustCompile(`\b(reglamento|directiva) (ue|ce|cee)\b`),
		regexp.MustCompile(`\bley (organica )?\d`),
		regexp.MustCompile(`\bboe\b`),
		regexp.MustCompile(`\bdoue\b`),
		regexp.MustCompile(`\b(boletin|diario) oficial\b`),
		regexp.MustCompile(`\barticles? \d`),
		regexp.MustCompile(`\bsections? \d`),
		regexp.MustCompile(`\bannex [ivx0-9]`),
		regexp.MustCompile(`\b(regulation|directive) \(?eu\)?\b`),
	}
)

// motivoRechazo dice por que una entrada no puede vivir en el catalogo, o
// cadena vacia si puede. El texto es el que ve quien rompe la regla, asi que
// lleva causa y arreglo.
func motivoRechazo(clave, valor string) string {
	if m := motivoRechazoClave(clave); m != "" {
		return m
	}
	return motivoRechazoValor(valor)
}

func motivoRechazoClave(clave string) string {
	if !formaDeClave.MatchString(clave) {
		return "La forma de una clave es espacio.subespacio.detalle, en minusculas, sin " +
			"tildes y con al menos un punto. Arreglo: renombrarla."
	}
	espacio := clave[:strings.Index(clave, ".")]
	for _, e := range espaciosDeClave {
		if espacio == e {
			return ""
		}
	}
	return "El espacio de nombres " + espacio + " no existe. Los que hay son " +
		strings.Join(espaciosDeClave, ", ") + ". Arreglo: usar uno de esos. Y si lo que " +
		"se queria era una clave con nombre de norma, no: el catalogo lleva cadenas de la " +
		"interfaz, el idioma del corpus va dentro del paquete."
}

func motivoRechazoValor(valor string) string {
	// El limite se mide por FORMA y no sobre el valor entero: una cadena con
	// singular y plural lleva dos rotulos dentro, y contarlos juntos castigaria
	// a quien pluraliza bien.
	for _, f := range strings.Split(valor, separadorDeFormas) {
		if n := utf8.RuneCountInString(f); n > maxRunasValor {
			return "El valor tiene " + strconv.Itoa(n) + " caracteres y el limite son " +
				strconv.Itoa(maxRunasValor) + ". Un rotulo de interfaz no es un parrafo, y " +
				"un parrafo aqui casi siempre es texto de una norma. Arreglo: acortarlo, o " +
				"llevarlo al paquete de corpus al que pertenece."
		}
	}
	if i := strings.IndexFunc(valor, unicode.IsControl); i >= 0 {
		return "El valor lleva un caracter de control (un salto de linea o un tabulador). " +
			"Un rotulo es de una linea. Arreglo: partirlo en dos claves."
	}
	// Los invisibles de formato (categoria Cf) no son de control, asi que la
	// comprobacion de arriba no los ve. Aqui importan porque un catalogo es
	// texto que escribe otra persona y que se pinta tal cual: una marca de
	// orden de escritura (U+202E) hace que en pantalla se lea al reves de lo
	// que pone el fichero, y un espacio de ancho cero parte una palabra que el
	// revisor lee entera. Es el ataque de trojan source aplicado a un rotulo.
	if i := strings.IndexFunc(valor, esInvisibleDeFormato); i >= 0 {
		return "El valor lleva un caracter invisible de formato (marca de orden de " +
			"escritura, espacio de ancho cero o similar). Sirve para que en pantalla se " +
			"lea algo distinto de lo que pone el fichero. Arreglo: quitarlo."
	}
	if marcadoHTML.MatchString(valor) {
		return "El valor lleva marcado HTML o un esquema de URL ejecutable. El marcado lo " +
			"pone la plantilla, no la cadena. Arreglo: dejar solo texto."
	}
	if strings.Contains(strings.ToLower(valor), "urn:") {
		return "El valor lleva un identificador de paquete de corpus. Arreglo: el catalogo " +
			"no nombra normas, las nombra el paquete."
	}
	norm := normalizar(valor)
	for _, m := range marcadoresNormativos {
		if m.MatchString(norm) {
			return "El valor parece una cita normativa (" + m.String() + "). El catalogo NO " +
				"transporta texto de una norma: traducirlo crea obra derivada y se sale de " +
				"la estratificacion de licencias del corpus. Arreglo: el texto legal va en " +
				"su paquete, en el idioma de su fuente; un paquete en otro idioma es un " +
				"paquete distinto con su propia fuente."
		}
	}
	return ""
}

func esInvisibleDeFormato(r rune) bool { return unicode.Is(unicode.Cf, r) }

// normalizar deja el texto en minusculas, sin tildes y con un solo espacio
// entre palabras, tirando todo lo que no sea letra o digito.
//
// Lo usan la frontera y el contraste contra el corpus, y tiene que ser el mismo
// para los dos: si uno normaliza distinto que el otro, el detector de texto del
// corpus mira una cadena que la frontera nunca vio.
func normalizar(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	anterior := ' '
	for _, r := range strings.ToLower(s) {
		if base, ok := sinTilde[r]; ok {
			r = base
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			anterior = r
			continue
		}
		if anterior != ' ' {
			b.WriteByte(' ')
			anterior = ' '
		}
	}
	return strings.TrimSpace(b.String())
}

// sinTilde cubre lo que aparece en castellano y en las lenguas vecinas del
// corpus. Lo que no este aqui se convierte en separador, que para comparar
// trozos de texto es el comportamiento seguro.
var sinTilde = map[rune]rune{
	'á': 'a', 'à': 'a', 'ä': 'a', 'â': 'a',
	'é': 'e', 'è': 'e', 'ë': 'e', 'ê': 'e',
	'í': 'i', 'ì': 'i', 'ï': 'i', 'î': 'i',
	'ó': 'o', 'ò': 'o', 'ö': 'o', 'ô': 'o',
	'ú': 'u', 'ù': 'u', 'ü': 'u', 'û': 'u',
	'ñ': 'n', 'ç': 'c',
}
