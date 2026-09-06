// Package ingesta convierte un fichero que sube el cliente en fragmentos de
// texto citables, o en un error. Nunca en un valor por defecto.
//
// # Para que existe
//
// Es la clave de boveda del bloque de IA de adopcion: las piezas 1, 3, 4 y 7 de
// `docs/ia.md`, la busqueda sobre los documentos del cliente y la mitad que
// falta de los evals adversariales dependen todas de que exista una forma de
// meter un documento en el sistema. Y ninguna se puede construir antes.
//
// # LA TERCERA FORMA DE LA NADA MANDA AQUI (invariante 8)
//
// Un fichero tiene TRES estados y no dos, y confundir el tercero con el segundo
// es absolver de mas con cara de dato:
//
//	AUSENTE                     no hay fichero. No es asunto de este paquete.
//	PRESENTE Y VACIO            hay fichero y no dice nada. Es un dato: cero
//	                            fragmentos, sin error.
//	PRESENTE Y NO INTERPRETABLE hay fichero y NO SE ENTIENDE. Es SIEMPRE un
//	                            error, nunca cero fragmentos.
//
// La tentacion, aqui, es enorme y tiene buena cara: un PDF escaneado, un PDF con
// una fuente incrustada sin mapa a Unicode, un fichero cortado a la mitad, un
// .pdf que en realidad es un .docx renombrado. Los cuatro dan cero texto util, y
// devolver «cero fragmentos» dejaria al cliente mirando una entrevista vacia
// convencido de que su politica no dice nada. Eso no es un fallo de comodidad:
// el sistema estaria afirmando algo sobre el contenido de un documento que no ha
// conseguido leer.
//
// # EL FORMATO LO DECIDE EL CONTENIDO, NO EL NOMBRE
//
// La extension la elige quien sube el fichero, o sea la parte de la que no nos
// fiamos. Se mira el contenido y punto; el nombre solo sirve para decirlo en
// pantalla cuando los dos no coinciden, que es informacion util y no un error.
//
// # LOS LIMITES SON DUROS Y ESTAN DECLARADOS
//
// Un fichero que sube un desconocido es entrada adversaria. Los tres limites de
// abajo se aplican SIEMPRE y no se pueden desactivar desde fuera: sin ellos un
// fichero de 3 KB puede pedir gigabytes de memoria (bomba de descompresion),
// que es la forma barata de tirar el servidor de cumplimiento de alguien.
//
// # LO QUE ESTE PAQUETE NO HACE
//
// No decide si un texto se puede citar ni de donde viene: eso es
// `adaptadores/ia`, que marca todo esto como `Aportado` y nunca como corpus. Un
// documento del cliente que diga «el articulo 5 obliga a cifrar» produce una
// cita que RESUELVE, y lo unico que impide ensenarla como si fuera la norma es
// esa procedencia. Aqui solo se saca texto.
package ingesta

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Los limites, en bytes y en unidades. Duros, declarados, y no configurables.
const (
	// MaximoEntrada es lo mas grande que se acepta leer. Una politica de
	// seguridad, un inventario de activos o un Excel de controles exportado a
	// texto no llegan a esto ni de lejos; lo que llega a esto es otra cosa.
	MaximoEntrada = 20 << 20 // 20 MiB
	// MaximoExtraido es el tope del texto que sale, YA DESCOMPRIMIDO. Es el
	// limite que importa: el de la entrada no protege de una bomba.
	MaximoExtraido = 8 << 20 // 8 MiB
	// MaximoFragmentos es cuantas unidades citables salen como mucho.
	MaximoFragmentos = 20000
)

// MinimoInterpretable es cuantas runas utiles tiene que dar un documento con
// contenido para considerarse leido.
//
// EL NUMERO ES UNA DECISION Y SE DICE. Por debajo de esto no se puede
// distinguir «un documento con una linea» de «un documento que no hemos sabido
// leer y del que se han colado cuatro caracteres sueltos», y ante esa duda la
// respuesta es error: absolver de mas es el fallo caro.
const MinimoInterpretable = 24

// FraccionInterpretable es que proporcion del texto extraido tiene que ser
// letra, digito, espacio o puntuacion corriente.
//
// Un PDF con una fuente incrustada sin mapa a Unicode no da cero bytes: da
// bytes, y son basura. Sin este contraste, esa basura pasaria por texto, se
// indexaria, y una cita podria resolver contra ella.
const FraccionInterpretable = 0.80

// Formato es lo que este paquete ha reconocido MIRANDO EL CONTENIDO.
type Formato uint8

const (
	// Desconocido es el valor cero y NO es un formato: es «no lo he
	// reconocido». Nunca se procesa.
	Desconocido Formato = iota
	// TextoPlano es texto UTF-8: .txt, .md, un .csv, un export.
	TextoPlano
	// PDF es un documento PDF.
	PDF
)

func (f Formato) String() string {
	switch f {
	case TextoPlano:
		return "texto plano"
	case PDF:
		return "PDF"
	default:
		return "formato no reconocido"
	}
}

// Fragmento es una unidad de texto citable con su sitio dentro del documento.
//
// Pagina y Orden van los DOS porque no sirven para lo mismo: la pagina es lo que
// se le ensena a una persona («tu politica, pagina 4») y el orden es lo que hace
// que dos fragmentos del mismo documento no colisionen al construir su
// identificador. En un formato sin paginas, Pagina es 0 y se dice que es 0.
type Fragmento struct {
	Pagina int
	Orden  int
	Texto  string
}

// Documento es el resultado de leer un fichero.
type Documento struct {
	Formato Formato
	// Paginas es cuantas tiene el original. 0 en un formato sin paginas.
	Paginas int
	// Fragmentos puede estar VACIO sin que sea un error: un fichero presente y
	// vacio es un dato. Lo que no puede es estar vacio porque no se entendio.
	Fragmentos []Fragmento
	// NombreEnganosO dice que la extension del fichero no casa con lo que hay
	// dentro. No es un error: la extension la elige quien sube el fichero. Es
	// informacion para la pantalla.
	NombreEnganoso bool
	// Truncado dice que se llego a un limite y hay contenido que no se leyo.
	// Va aqui y no como error por lo mismo de siempre: leer media politica y
	// DECIRLO es util; leer media politica y callarlo es mentir.
	Truncado bool
}

// Los errores de este paquete. Todos centinela: quien llama tiene que poder
// distinguir «no se entiende» de «esta vacio» sin leer una cadena.
var (
	ErrDemasiadoGrande     = errors.New("ingesta: el fichero pasa del maximo")
	ErrFormatoNoReconocido = errors.New(
		"ingesta: el contenido del fichero no es ninguno de los formatos que se saben leer")
	ErrSinTextoInterpretable = errors.New(
		"ingesta: el fichero se ha leido y no ha salido texto que se pueda entender")
	ErrPDFCifrado = errors.New("ingesta: el PDF esta cifrado")
)

// FraseDelNoLeido es el descargo que acompana SIEMPRE al error de un documento
// que no se ha podido leer.
//
// Es una constante y no una redaccion suelta por lo mismo que las frases de
// `nucleo/acta`: es una promesa del producto y tiene que poder afirmarse sin
// que un test cuelgue de como esta escrita hoy. Y la promesa es la de siempre,
// bajada del calendario al disco: **no consta** y **no se ha podido leer** son
// dos cosas distintas, y presentar la segunda como la primera es acusar en
// falso al documento de alguien.
const FraseDelNoLeido = "NO se sigue de aqui que el documento no diga nada: se sigue " +
	"que plazum no ha podido leerlo"

// Leer extrae el texto de un fichero subido.
//
// nombre es el que declara el navegador y NO decide nada: solo se usa para
// avisar de que no casa con el contenido.
func Leer(nombre string, datos []byte) (Documento, error) {
	if len(datos) > MaximoEntrada {
		return Documento{}, fmt.Errorf(
			"%w: %d bytes, y el maximo son %d. Arreglo: sube el documento solo, sin "+
				"los adjuntos, o exportalo a texto",
			ErrDemasiadoGrande, len(datos), MaximoEntrada)
	}

	f := Reconocer(datos)
	doc := Documento{Formato: f, NombreEnganoso: !casaConLaExtension(nombre, f)}

	switch f {
	case TextoPlano:
		doc.Fragmentos, doc.Truncado = fragmentarTexto(string(datos))
	case PDF:
		paginas, frags, trunc, err := extraerPDF(datos)
		if err != nil {
			return Documento{}, err
		}
		doc.Paginas, doc.Fragmentos, doc.Truncado = paginas, frags, trunc
	default:
		return Documento{}, fmt.Errorf(
			"%w. Se saben leer: texto plano (UTF-8) y PDF. Arreglo: exporta el "+
				"documento a PDF o a texto y vuelve a subirlo. Y ojo, la extension no "+
				"decide: un fichero llamado .pdf que por dentro es otra cosa cae aqui",
			ErrFormatoNoReconocido)
	}

	// LA TERCERA FORMA DE LA NADA. Un fichero VACIO da cero fragmentos y no es
	// un error; uno con contenido del que no sale texto util, SI.
	if len(doc.Fragmentos) == 0 {
		if vacioDeVerdad(datos, f) {
			return doc, nil
		}
		return Documento{}, fmt.Errorf(
			"%w (%s, %d bytes). Lo que suele ser: un PDF escaneado sin capa de texto, "+
				"un PDF con la fuente incrustada sin mapa a Unicode, o un fichero "+
				"cortado. Arreglo: pasale un OCR o exporta el original a texto. %s",
			ErrSinTextoInterpretable, f, len(datos), FraseDelNoLeido)
	}

	util, total := cuentaInterpretable(doc.Fragmentos)
	if total < MinimoInterpretable || float64(util)/float64(total) < FraccionInterpretable {
		return Documento{}, fmt.Errorf(
			"%w (%s: %d runas, %d utiles, hace falta el %.0f%% y un minimo de %d). "+
				"Sale texto pero no se parece a lenguaje, que es lo que da una fuente "+
				"incrustada sin mapa a Unicode. Arreglo: exporta el original a texto. %s",
			ErrSinTextoInterpretable, f, total, util, FraccionInterpretable*100,
			MinimoInterpretable, FraseDelNoLeido)
	}
	return doc, nil
}

// Reconocer dice que formato tiene un contenido, MIRANDOLO.
func Reconocer(datos []byte) Formato {
	if len(datos) == 0 {
		// Un fichero vacio se lee como texto vacio, que es lo que es. La
		// decision de si eso es un dato o un error no es de aqui.
		return TextoPlano
	}
	// Un PDF empieza por %PDF-, y la cabecera puede llevar basura delante
	// (lo permite la propia especificacion), asi que se busca cerca del
	// principio y no solo en el byte cero.
	if i := indiceDe(datos, []byte("%PDF-"), 1024); i >= 0 {
		return PDF
	}
	if esTextoUTF8(datos) {
		return TextoPlano
	}
	return Desconocido
}

// vacioDeVerdad distingue el fichero que no dice nada del que no se entiende.
// Un texto plano de cero bytes, o de solo espacios, esta vacio. Un PDF nunca lo
// esta: siempre tiene estructura, asi que cero fragmentos significa que no se
// ha sabido leer.
func vacioDeVerdad(datos []byte, f Formato) bool {
	return f == TextoPlano && strings.TrimSpace(string(datos)) == ""
}

// esTextoUTF8 dice si unos bytes son texto legible.
//
// Se exige UTF-8 valido Y que la inmensa mayoria sean caracteres imprimibles:
// un binario cualquiera puede ser UTF-8 valido por casualidad en trozos cortos,
// y un .docx (que es un zip) no lo es nunca. El byte nulo descarta de entrada,
// porque no aparece en texto de verdad.
func esTextoUTF8(datos []byte) bool {
	muestra := datos
	if len(muestra) > 8192 {
		muestra = muestra[:8192]
		// Recortar por la mitad de una runa daria invalido por el corte y no
		// por el contenido, asi que se retrocede hasta el principio de runa.
		for len(muestra) > 0 && !utf8.Valid(muestra) {
			muestra = muestra[:len(muestra)-1]
		}
	}
	if !utf8.Valid(muestra) || len(muestra) == 0 {
		return false
	}
	imprimibles, total := 0, 0
	for _, r := range string(muestra) {
		total++
		if r == 0 {
			return false
		}
		if unicode.IsPrint(r) || r == '\n' || r == '\r' || r == '\t' {
			imprimibles++
		}
	}
	return total > 0 && float64(imprimibles)/float64(total) >= 0.95
}

// casaConLaExtension compara lo que dice el nombre con lo que hay dentro.
func casaConLaExtension(nombre string, f Formato) bool {
	i := strings.LastIndex(nombre, ".")
	if i < 0 {
		// Sin extension no hay nada que contradecir.
		return true
	}
	switch strings.ToLower(nombre[i+1:]) {
	case "pdf":
		return f == PDF
	case "txt", "md", "markdown", "csv", "tsv", "json", "log":
		return f == TextoPlano
	default:
		// Una extension que no conocemos no afirma nada, asi que no engana.
		return true
	}
}

// cuentaInterpretable cuenta runas utiles y totales sobre todo el texto.
func cuentaInterpretable(fs []Fragmento) (util, total int) {
	for _, f := range fs {
		for _, r := range f.Texto {
			total++
			switch {
			case unicode.IsLetter(r), unicode.IsDigit(r), unicode.IsSpace(r):
				util++
			case unicode.IsPunct(r), unicode.IsSymbol(r):
				util++
			}
		}
	}
	return util, total
}

// indiceDe busca aguja en los primeros tope bytes de pajar.
func indiceDe(pajar, aguja []byte, tope int) int {
	if len(pajar) < tope {
		tope = len(pajar)
	}
	return strings.Index(string(pajar[:tope]), string(aguja))
}
