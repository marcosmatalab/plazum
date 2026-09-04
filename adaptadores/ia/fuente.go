package ia

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/marcosmatalab/plazum/adaptadores/busqueda"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// Procedencia dice DE DONDE sale un texto citable, y es lo primero que mira el
// verificador.
//
// POR QUE ES UN CAMPO Y NO UN DETALLE. Un documento que sube el cliente y un
// articulo del corpus firmado se ven exactamente igual una vez son texto. Si
// entran al mismo saco, un PDF que diga "el articulo 5 obliga a cifrar" produce
// una cita que RESUELVE, porque esa frase esta literalmente en un documento que
// el sistema tiene, y la pantalla la ensenaria como si viniera de la norma. Eso
// no es una alucinacion del modelo: es una inyeccion via documento, y es entrada
// adversaria declarada (ETAPAS.md, bloque IA de la v1).
//
// EL VALOR CERO ESTA PROHIBIDO. `Ninguna` no es "cualquiera": es "no lo has
// dicho", y una fuente que no dice de donde viene no entra. Invariante 8.
type Procedencia uint8

const (
	// Ninguna es el valor cero y NO es una procedencia valida.
	Ninguna Procedencia = iota
	// Corpus es texto que viene de un paquete del corpus, o sea de un fichero
	// firmado con Ed25519 cuyo contenido no puede cambiar sin romper la firma.
	Corpus
	// Aportado es texto de un documento que subio el cliente. No lo firma
	// nadie y puede contener lo que sea, incluida una instruccion dirigida al
	// modelo.
	Aportado
)

func (p Procedencia) String() string {
	switch p {
	case Corpus:
		return "corpus firmado"
	case Aportado:
		return "documento aportado por el cliente"
	default:
		return "procedencia sin declarar"
	}
}

// Valida dice si la procedencia es una de las declaradas. El cero no lo es.
func (p Procedencia) Valida() bool { return p == Corpus || p == Aportado }

// Fuente es una unidad de texto sobre la que se puede resolver una cita.
//
// El texto original se guarda TAL CUAL viene del paquete, y aparte se guarda su
// forma con los espacios colapsados, que es contra la que se casa la cita, con
// el mapa que devuelve cada runa de esa forma a su sitio en el original. Asi lo
// que se ensena al final sale del ORIGINAL y no de la forma normalizada ni,
// mucho menos, de lo que escribio el modelo.
type Fuente struct {
	ID          string
	Hash        string
	Marco       string
	Articulo    string
	Clase       string
	Procedencia Procedencia
	// Citable dice si de esta fuente se puede ensenar texto. Ver ClaseCitable.
	Citable bool
	Texto   string

	normal string
	mapa   []int
	origen []rune
}

var (
	ErrFuenteSinID          = errors.New("ia: fuente sin identificador")
	ErrIDConSeparador       = errors.New("ia: identificador con el separador del hash dentro")
	ErrFuenteSinProcedencia = errors.New("ia: fuente sin procedencia declarada")
	ErrFuenteConHashFalso   = errors.New("ia: fuente cuyo hash no es el que le corresponde")
	ErrFuenteRepetida       = errors.New("ia: dos fuentes con el mismo hash")
)

// separador es lo que va entre el identificador y el texto al calcular el hash.
//
// ES UN BYTE NULO Y NO UN SALTO DE LINEA por una razon concreta: con un
// separador que puede aparecer en cualquiera de las dos partes, dos parejas
// distintas dan la misma cadena. Con "\n", la fuente (id "a", texto "b\nc") y
// la fuente (id "a\nb", texto "c") hashean igual, y entonces el separador no
// separa nada. Un byte nulo no aparece en un identificador, y ademas se
// comprueba que no aparece.
const separador = "\x00"

// HashDe calcula el hash de una unidad citable.
//
// POR QUE ENTRA EL IDENTIFICADOR Y NO SOLO EL TEXTO, que es un cambio de diseno
// y sale de un ROJO SOBRE EL CORPUS REAL, no de una mutacion (medido el
// 04-09-2026). Hasheando solo el texto:
//
//	528 obligaciones del corpus dan 495 hashes distintos.
//	29 hashes tienen mas de una obligacion detras, o sea 33 obligaciones
//	  quedan TAPADAS por otra, y cual gana lo decide el orden de un mapa.
//
// Y no son casos raros ni erratas: `iso27001.4.2` y `iso42001.4.2` son dos
// normas con la misma estructura de clausulas y el mismo titulo corto, que es
// exactamente lo que un paquete referencial guarda. Lo mismo con seis rituales
// de un mismo paquete. O sea que el choque es NORMAL y va a crecer con cada
// marco nuevo.
//
// Con el texto solo, una cita legitima habria resuelto a un articulo y la
// pantalla habria dicho "el articulo X dice esto" nombrando OTRO articulo. No
// es una mentira sobre el contenido, es una mentira sobre la atribucion, y en
// un producto de cumplimiento es igual de cara.
//
// La identidad de una unidad citable es la pareja (identificador, texto), y las
// dos van DENTRO de lo firmado: el paquete se firma entero con Ed25519. Sigue
// cumpliendo el invariante 7, y ahora sin colisiones.
func HashDe(id, texto string) string {
	suma := sha256.Sum256([]byte(id + separador + texto))
	return hex.EncodeToString(suma[:])
}

// NuevaFuente construye una fuente CALCULANDO su hash.
//
// El hash no se acepta de fuera A PROPOSITO. Si quien construye la fuente
// pudiera decir cual es su hash, un hash equivocado se convertiria en una
// mentira permanente: el verificador resolveria una cita contra un texto que no
// es el que el hash dice. Calcularlo aqui hace que el emparejamiento entre una
// propuesta y su fuente vaya por un campo DERIVADO de lo firmado, que es lo que
// pide el invariante 7: nadie firma el orden ni la posicion, pero el contenido
// del paquete si va firmado, y este hash sale de ese contenido.
func NuevaFuente(id, marco, articulo, clase string, p Procedencia, citable bool, texto string) (Fuente, error) {
	if id == "" {
		return Fuente{}, fmt.Errorf("%w. Arreglo: toda fuente entra con el identificador "+
			"de su obligacion, que es por donde se le vuelve a encontrar", ErrFuenteSinID)
	}
	if strings.Contains(id, separador) {
		return Fuente{}, fmt.Errorf("%w. Arreglo: un identificador con un byte nulo dentro "+
			"puede hacer que dos fuentes distintas den el mismo hash, y entonces una tapa "+
			"a la otra", ErrIDConSeparador)
	}
	if !p.Valida() {
		return Fuente{}, fmt.Errorf("%w: %s. Arreglo: di si el texto viene del corpus "+
			"firmado o de un documento que subio el cliente. El valor cero no es "+
			"'cualquiera', es 'no lo has dicho'", ErrFuenteSinProcedencia, id)
	}
	normal, mapa := normalizar(texto)
	return Fuente{
		ID:          id,
		Hash:        HashDe(id, texto),
		Marco:       marco,
		Articulo:    articulo,
		Clase:       clase,
		Procedencia: p,
		Citable:     citable && texto != "",
		Texto:       texto,
		normal:      normal,
		mapa:        mapa,
		origen:      []rune(texto),
	}, nil
}

// ClaseCitable dice si de un paquete de esa clase se puede ENSENAR texto.
//
// ES LA FRONTERA LEGAL (invariante 3) LLEVADA AL VERIFICADOR, y es la que
// convierte el apartado 3 de `docs/ia.md` en una consecuencia mecanica en vez
// de una promesa sobre el comportamiento de un modelo:
//
//	importado   catalogo en dominio publico, se distribuye entero. Citable.
//	transcrito  texto de BOE o DOUE con la reutilizacion cumplida. Citable.
//	propio      datos del proyecto. No hay tercero con derechos. Citable.
//	referencial identificadores y mapeo propio, SIN texto normativo. El campo
//	            de texto de estos paquetes lleva un titulo corto que escribimos
//	            nosotros bajo el limite de 120 caracteres del linter, y NO es el
//	            enunciado de la norma. Ensenarlo como cita seria decir que dice
//	            eso, y no lo dice. NO citable.
//	delegado    no se distribuye nada. NO citable.
//
// LA MEDIDA, del corpus real el 04-09-2026: 321 obligaciones transcritas con
// 182.782 runas de texto, media de 569; y 200 referenciales con 9.223 runas,
// media de 46 y la mas corta de 12. Esos 46 caracteres de media son la prueba
// de que ahi no hay texto normativo: no cabe.
//
// Y SE DECIDE POR LA CLASE DEL PAQUETE, no por una lista de marcos escrita
// aqui. Es la pregunta de la pasada 1: un paquete referencial nuevo entra sin
// tocar una linea de Go y ya nace no citable.
func ClaseCitable(c corpus.Clase) bool {
	switch c {
	case corpus.Importado, corpus.Transcrito, corpus.Propio:
		return true
	default:
		return false
	}
}

// ClaseDeNombre traduce el nombre de una clase al valor del nucleo.
//
// LA TABLA NO ESTA ESCRITA: se compone recorriendo las clases que declara el
// nucleo y pidiendoles su nombre. Escrita a mano seria una segunda lista, y una
// segunda lista es una lista que se queda vieja: el dia que entrara una clase
// nueva, esto seguiria contestando lo de siempre.
//
// Un nombre que no se reconoce es ERROR y no una clase por defecto. El valor
// cero de corpus.Clase es `importado`, que SI es citable, asi que un
// "referncial" mal escrito convertiria un paquete sin texto normativo en uno
// citable, en verde y sin que nadie lo notara.
func ClaseDeNombre(s string) (corpus.Clase, bool) {
	for c := corpus.Clase(0); c <= corpus.Propio; c++ {
		if c.String() == s {
			return c, true
		}
	}
	return 0, false
}

// FuentesDelCorpus convierte los paquetes cargados en fuentes citables.
//
// Devuelve TODAS las obligaciones, tambien las no citables, y no las filtra:
// una fuente no citable que existe permite contestar "de este marco no tenemos
// el texto", que es una respuesta util, en vez de "no existe esa fuente", que
// suena a fallo del producto. La que filtra es Documentos.
func FuentesDelCorpus(ps []*corpus.Paquete) ([]Fuente, error) {
	var out []Fuente
	for _, p := range ps {
		citable := ClaseCitable(p.Clase)
		for _, o := range p.Obligaciones {
			f, err := NuevaFuente(o.ID, p.URN, o.Articulo, p.Clase.String(),
				Corpus, citable, o.TextoLegal)
			if err != nil {
				return nil, fmt.Errorf("construyendo la fuente de %s: %w", o.ID, err)
			}
			out = append(out, f)
		}
	}
	return out, nil
}

// Documentos son las fuentes que PUEDEN indexarse para buscar.
//
// FILTRA POR CITABLE, y esa es la mitad del valor de esta funcion. Meter texto
// no citable en el indice de busqueda pondria ese texto en el contexto del
// modelo, y entonces el modelo lo parafrasearia dentro de una propuesta cuya
// cita apunta a otro sitio. El verificador lo rechazaria, pero el dano ya
// estaria hecho: el texto de un catalogo privativo habria salido del proceso.
// Lo que no entra al contexto no puede salir por ningun lado.
func Documentos(fs []Fuente) []busqueda.Documento {
	var out []busqueda.Documento
	for _, f := range fs {
		if !f.Citable {
			continue
		}
		out = append(out, busqueda.Documento{
			ID:       f.ID,
			Hash:     f.Hash,
			Marco:    f.Marco,
			Articulo: f.Articulo,
			Texto:    f.Texto,
		})
	}
	return out
}

// normalizar colapsa las series de espacios en uno solo y recorta los extremos,
// devolviendo ademas, por cada runa de salida, el indice de la runa de entrada
// de la que sale.
//
// POR QUE SOLO LOS ESPACIOS Y NADA MAS. El salto de linea y la sangria de un
// JSON son un artefacto del transporte: el mismo articulo escrito en una linea
// o en cinco es el mismo articulo. Todo lo demas NO se toca: ni comillas
// curvas por rectas, ni composicion Unicode, ni mayusculas. Cada uno de esos
// pliegues seria aceptar como cita un texto que el modelo escribio con
// caracteres distintos de los de la fuente, y elegir caracteres es exactamente
// lo que aqui no se le confia.
//
// LO QUE ESO CUESTA, medido y no supuesto: el corpus transcrito no contiene NI
// UNA marca combinante (0 de 182.782 runas, medido el 04-09-2026), asi que
// todas sus letras acentuadas son precompuestas y una cita en NFC casa. Una
// cita en NFD no casaria, y el verificador lo dice con ese motivo concreto en
// vez de con un "no aparece" a secas. Lo vigila un caso del arnes de evals
// sobre el corpus real: el dia que entre corpus con marcas combinantes, se
// pone rojo y hay que decidir, no descubrirlo en produccion.
func normalizar(s string) (string, []int) {
	origen := []rune(s)
	out := make([]rune, 0, len(origen))
	mapa := make([]int, 0, len(origen))
	enEspacio := false
	primerEspacio := 0
	for i, r := range origen {
		if unicode.IsSpace(r) {
			if !enEspacio {
				primerEspacio = i
			}
			enEspacio = true
			continue
		}
		if enEspacio && len(out) > 0 {
			out = append(out, ' ')
			mapa = append(mapa, primerEspacio)
		}
		enEspacio = false
		out = append(out, r)
		mapa = append(mapa, i)
	}
	return string(out), mapa
}

// tieneMarcasCombinantes dice si un texto trae marcas combinantes Unicode, o
// sea acentos escritos como caracter aparte (NFD). Sirve para dar un motivo
// concreto cuando una cita no casa, en vez de un "no aparece" que no dice que
// hacer.
func tieneMarcasCombinantes(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Mn, r) {
			return true
		}
	}
	return false
}
