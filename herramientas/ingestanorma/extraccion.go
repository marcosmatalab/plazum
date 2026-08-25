package main

// El modelo comun de la ingesta: lo que sale de BOE y de EUR-Lex tiene la misma
// forma, porque lo que hay aguas abajo (quien autora un paquete, y la pagina de
// vigilancia) es lo mismo en los dos casos.
//
// Dos campos son OBLIGATORIOS y no llevan omitempty a proposito: licencia_fuente
// y atribucion. La frontera legal de este proyecto permite transcribir BOE
// (art. 13 TRLPI) y DOUE (Decision 2011/833/UE), y la del DOUE lo permite CON
// ATRIBUCION. Una atribucion que hay que acordarse de poner mas tarde es una
// atribucion que se pierde, asi que viaja pegada al dato desde la extraccion.

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// EsquemaIngesta versiona el JSON intermedio. Sube cuando cambie la forma, para
// que quien lea una extraccion vieja lo sepa en vez de adivinarlo.
const EsquemaIngesta = "plazum/ingesta/v1"

// Los errores del dominio, con centinela. Se comprueban con errors.Is y NUNCA
// comparando el texto del mensaje: el texto cambia al mejorarlo y el test que lo
// compara se queda verde sin comprobar nada.
var (
	// ErrFuenteNoAutorizada: se pidio ingerir de un sitio que no es fuente
	// primaria. Es la regla legal, no una lista de conveniencia: la licencia de
	// un repositorio de terceros no alcanza al texto normativo que su autor no
	// poseia.
	ErrFuenteNoAutorizada = errors.New("origen que no es fuente primaria")
	// ErrNormaNoEncontrada: la fuente respondio, pero no tiene esa norma.
	ErrNormaNoEncontrada = errors.New("la fuente no tiene esa norma")
	// ErrRespuestaIlegible: la fuente respondio algo que no se puede parsear.
	ErrRespuestaIlegible = errors.New("respuesta de la fuente ilegible")
	// ErrSinArticulos: se parseo bien y no salio ni un articulo. Casi siempre
	// significa que la fuente cambio de formato, y es justo el caso en el que
	// callarse dejaria el almacen de vigilancia lleno de vacios.
	ErrSinArticulos = errors.New("la norma se descargo pero no tiene articulos")
	// ErrIdentificadorInvalido: el ELI o el CELEX no tienen la forma que dicen.
	ErrIdentificadorInvalido = errors.New("identificador de norma invalido")
)

// Extraccion es lo que la herramienta escribe: una norma convertida en
// articulos, con su cita y su procedencia. NO es un paquete de corpus, y no se
// escribe en paquetes/: un humano autoriza el paquete, siempre.
type Extraccion struct {
	Esquema string `json:"esquema"`
	Fuente  Origen `json:"fuente"`
	// LicenciaFuente y Atribucion: obligatorios, sin omitempty. Ver la cabecera.
	LicenciaFuente string     `json:"licencia_fuente"`
	Atribucion     string     `json:"atribucion"`
	Obtenido       string     `json:"obtenido"` // RFC3339, cuando se descargo
	Huella         string     `json:"huella"`   // sobre los articulos normalizados
	Articulos      []Articulo `json:"articulos"`
}

// Origen es de donde viene el texto, con todo lo que hace falta para volver a
// buscarlo y para citarlo.
type Origen struct {
	Jurisdiccion  string `json:"jurisdiccion"` // es | ue
	Identificador string `json:"identificador"`
	ELI           string `json:"eli,omitempty"`
	// URLDocumento es la pagina CITABLE, la que abre una persona para verificar
	// la cita. URLDatos es la URL de la que se bajo el XML. Las dos, porque no
	// son la misma y las dos hacen falta: una para el paquete, la otra para
	// repetir la descarga.
	URLDocumento string `json:"url_documento"`
	URLDatos     string `json:"url_datos"`
	Titulo       string `json:"titulo"`
	CitaCorta    string `json:"cita_corta"` // "Real Decreto 311/2022", para las citas
	// URNSugerido es una PROPUESTA de identificador de paquete derivada del ELI
	// o del CELEX. Sugerido, no decidido: quien autora el paquete manda.
	URNSugerido      string `json:"urn_sugerido,omitempty"`
	Consolidado      bool   `json:"consolidado"`
	FechaDisposicion string `json:"fecha_disposicion,omitempty"`
	FechaPublicacion string `json:"fecha_publicacion,omitempty"`
	FechaVigencia    string `json:"fecha_vigencia,omitempty"`
	Derogada         bool   `json:"derogada"`
	FechaDerogacion  string `json:"fecha_derogacion,omitempty"`
	// ActualizadaEn es la marca de tiempo que la FUENTE declara para su ultima
	// actualizacion. Es la mitad izquierda de la tabla de vigilancia
	// (fecha de la fuente hacia fecha del paquete): sin ella no hay track record.
	ActualizadaEn string `json:"actualizada_en,omitempty"`
	// TextoVigenteEn: la fecha a la que se pidio el texto, si se pidio una
	// concreta. Vacio significa "la ultima version".
	TextoVigenteEn string `json:"texto_vigente_en,omitempty"`
}

// Articulo es una unidad citable de la norma.
type Articulo struct {
	// Referencia es la CLAVE de la vigilancia, y por eso es el titulo del
	// bloque tal cual lo publica la fuente, normalizado de espacios: "Articulo
	// 31", "Disposicion adicional segunda", "ANEXO II".
	//
	// Por que no el id interno del bloque: los id del BOE ("a3-3") se derivan
	// de la posicion en la estructura, asi que insertar un articulo los
	// desplaza y el diff diria que cambio media norma. El titulo no se mueve.
	Referencia string `json:"referencia"`
	// ID es el identificador del bloque EN LA FUENTE. Sirve para el ancla de la
	// URL y para volver a pedir ese bloque suelto, no para casar versiones.
	ID       string   `json:"id"`
	Numero   string   `json:"numero,omitempty"`
	Tipo     string   `json:"tipo"` // precepto | anexo | preambulo | estructura | firma
	Titulo   string   `json:"titulo,omitempty"`
	Texto    string   `json:"texto"`
	Parrafos []string `json:"parrafos"`
	Cita     string   `json:"cita"`
	Fuente   string   `json:"fuente"` // URL con ancla al articulo concreto
	// VigenciaDesde es la fecha de entrada en vigor de ESTA version del
	// articulo, no de la norma. Es lo que distingue un articulo modificado en
	// 2024 dentro de una norma de 2022.
	VigenciaDesde string `json:"vigencia_desde,omitempty"`
	ModificadoPor string `json:"modificado_por,omitempty"`
	Derogado      bool   `json:"derogado"`
	Nota          string `json:"nota,omitempty"` // la nota al pie de la fuente
	Huella        string `json:"huella"`
}

// HuellaDe normaliza y resume el contenido citable de un articulo. Es lo que
// decide si un articulo "ha cambiado" entre dos ejecuciones.
//
// Entra el titulo ademas del texto porque cambiarle el titulo a un articulo es
// un cambio normativo, y saldria "sin cambio" si solo se mirara el cuerpo.
// Entra tambien la marca de derogado: un articulo cuyo texto pasa a "(Derogado)"
// ya cambia de texto, pero uno marcado por fecha de caducidad no, y ese silencio
// seria justo el que no se puede permitir.
func (a Articulo) HuellaDe() string {
	h := sha256.New()
	esc := func(s string) {
		_, _ = h.Write([]byte(normalizarTexto(s)))
		_, _ = h.Write([]byte{0})
	}
	esc(a.Referencia)
	esc(a.Titulo)
	esc(a.Texto)
	if a.Derogado {
		esc("derogado")
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// HuellaDeExtraccion resume la norma entera: la lista ordenada de articulos con
// su huella. Cambia si cambia cualquier articulo, si aparece uno o si
// desaparece uno.
func HuellaDeExtraccion(as []Articulo) string {
	h := sha256.New()
	for _, a := range as {
		_, _ = h.Write([]byte(a.Referencia))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(a.Huella))
		_, _ = h.Write([]byte{0x1e})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// normalizarTexto deja el texto comparable entre dos descargas sin tocar lo que
// dice: series de blancos a un espacio y recorte por los extremos.
//
// El espacio duro no es una mania: BOE y EUR-Lex escriben el titulo del
// articulo con U+00A0 entre la palabra y el numero, y sin normalizarlo la clave
// de vigilancia de un articulo no casa con la que escribe una persona.
// unicode.IsSpace ya cubre U+00A0 y U+202F, que son los dos que aparecen.
func normalizarTexto(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	espacio := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			espacio = true
			continue
		}
		if espacio && b.Len() > 0 {
			b.WriteByte(' ')
		}
		espacio = false
		b.WriteRune(r)
	}
	return b.String()
}

// citaCorta recorta el titulo oficial hasta el punto en que deja de identificar
// la norma y empieza a describirla:
//
//	"Real Decreto 311/2022, de 3 de mayo, por el que se regula..."
//	    -> "Real Decreto 311/2022"
//	"Reglamento (UE) 2016/679 del Parlamento Europeo y del Consejo, de 27 de..."
//	    -> "Reglamento (UE) 2016/679"
//
// Es un recorte sintactico, no una interpretacion: se corta en ", de " (la
// formula de fecha que usan las dos fuentes) y, si no aparece, en la primera
// coma. Si tampoco hay coma se devuelve el titulo entero, porque inventarse un
// recorte seria peor que uno largo.
func citaCorta(titulo string) string {
	t := normalizarTexto(titulo)
	if i := strings.Index(t, ", de "); i > 0 {
		t = t[:i]
	} else if i := strings.Index(t, ","); i > 0 {
		t = t[:i]
	}
	// En el DOUE lo que identifica la norma termina antes del organo que la
	// dicta, y ese trozo va ANTES de la coma de la fecha.
	for _, corte := range []string{
		" del Parlamento Europeo", " del Consejo", " de la Comisi",
	} {
		if i := strings.Index(t, corte); i > 0 {
			t = t[:i]
		}
	}
	return strings.TrimSpace(t)
}

// citaDe compone la cita de un articulo a partir de la cita corta de la norma y
// la referencia del bloque. Es la cadena que quien autora el paquete pega en el
// campo de cita de la obligacion.
func citaDe(corta, referencia string) string {
	ref := normalizarTexto(referencia)
	if n := numeroDeArticulo(ref); n != "" {
		ref = "art. " + n
	}
	if corta == "" {
		return ref
	}
	return corta + ", " + ref
}

// numeroDeArticulo saca el "31" de "Articulo 31". Devuelve vacio si la
// referencia no es un articulo (una disposicion adicional, un anexo).
func numeroDeArticulo(referencia string) string {
	r := normalizarTexto(referencia)
	bajo := strings.ToLower(r)
	for _, p := range prefijosDeArticulo {
		if strings.HasPrefix(bajo, p) {
			n := strings.TrimSpace(r[len(p):])
			return strings.TrimSuffix(n, ".")
		}
	}
	return ""
}

// prefijosDeArticulo: con tilde y sin ella porque las dos fuentes escriben la
// palabra con tilde y un fixture recortado a mano puede no llevarla.
var prefijosDeArticulo = []string{"artículo ", "articulo ", "article "}

// tituloTrasLaReferencia separa "Auditoria de la seguridad" de
// "Articulo 31. Auditoria de la seguridad." Devuelve vacio si el encabezado no
// empieza por la referencia, porque entonces no se sabe donde acaba una y
// empieza la otra, y adivinarlo seria inventar.
func tituloTrasLaReferencia(encabezado, referencia string) string {
	e := normalizarTexto(encabezado)
	r := normalizarTexto(referencia)
	if r == "" || !strings.HasPrefix(strings.ToLower(e), strings.ToLower(r)) {
		return ""
	}
	resto := strings.TrimSpace(e[len(r):])
	resto = strings.TrimLeft(resto, ".:- –")
	return strings.TrimSuffix(strings.TrimSpace(resto), ".")
}

// anclar pega el ancla del bloque a la URL del documento. Sin ancla la fuente de
// un articulo apunta a una norma de trescientas paginas, y para verificar una
// cita eso no sirve de nada.
func anclar(url, id string) string {
	if url == "" || id == "" {
		return url
	}
	if i := strings.Index(url, "#"); i >= 0 {
		url = url[:i]
	}
	return url + "#" + id
}

// cerrarTodos es el ultimo paso de los dos parsers: asegura que la clave de
// vigilancia existe y es unica, y luego rellena lo derivado de cada articulo.
//
// POR QUE LA UNICIDAD SE COMPRUEBA AQUI Y NO SE DA POR HECHA. Lo encontro el
// fuzzing y despues se confirmo con datos reales: un bloque puede llegar sin
// rotulo, y dos bloques distintos pueden llevar el MISMO rotulo (una norma con
// dos "CAPITULO I", uno por titulo). La referencia es la clave con la que se
// comparan dos ejecuciones: dos articulos con la misma clave se tapan en el
// mapa, y la ejecucion siguiente canta una derogacion que no ha existido. Una
// vigilancia que inventa derogaciones es peor que no tener vigilancia.
func cerrarTodos(as []Articulo, corta, urlDoc string) {
	vistas := map[string]bool{}
	for i := range as {
		r := normalizarTexto(as[i].Referencia)
		if r == "" {
			r = as[i].ID // el rotulo falta: queda el id de la fuente
		}
		if r == "" {
			r = fmt.Sprintf("bloque sin rotulo %d", i+1)
		}
		if vistas[r] {
			// Se desempata con el id de la fuente, que es lo unico que
			// distingue dos bloques con el mismo rotulo. Si tampoco basta, con
			// la posicion, que es el ultimo recurso y se nota al leerlo.
			candidato := r + " [" + as[i].ID + "]"
			if as[i].ID == "" || vistas[candidato] {
				candidato = fmt.Sprintf("%s [%d]", r, i+1)
			}
			r = candidato
		}
		vistas[r] = true
		as[i].Referencia = r
		cerrarArticulo(&as[i], corta, urlDoc)
	}
}

// cerrarArticulo rellena lo derivado (texto, cita, fuente, huella) una vez que
// el parser de cada fuente ha puesto lo suyo. Centralizado a proposito: si cada
// parser compusiera su cita, las dos fuentes divergirian a la primera.
func cerrarArticulo(a *Articulo, corta, urlDoc string) {
	a.Referencia = normalizarTexto(a.Referencia)
	a.Titulo = normalizarTexto(a.Titulo)
	a.Numero = numeroDeArticulo(a.Referencia)
	limpios := make([]string, 0, len(a.Parrafos))
	for _, p := range a.Parrafos {
		if n := normalizarTexto(p); n != "" {
			limpios = append(limpios, n)
		}
	}
	a.Parrafos = limpios
	a.Texto = strings.Join(limpios, "\n\n")
	a.Cita = citaDe(corta, a.Referencia)
	a.Fuente = anclar(urlDoc, a.ID)
	a.Nota = normalizarTexto(a.Nota)
	a.Huella = a.HuellaDe()
}

// derogadoPorTexto reconoce la convencion con la que el BOE marca un precepto
// derogado: la version vigente del bloque no tiene mas cuerpo que "(Derogado)"
// o "(Suprimido)", y la nota al pie dice por que norma.
//
// Es una convencion editorial, no un campo, asi que se comprueba contra el
// cuerpo ENTERO y no por contencion: un articulo que hable de normas derogadas
// no puede confundirse con un articulo derogado.
func derogadoPorTexto(texto string) bool {
	t := strings.ToLower(normalizarTexto(texto))
	t = strings.Trim(t, "()[]. ")
	switch t {
	case "derogado", "derogada", "derogados", "derogadas",
		"suprimido", "suprimida", "sin contenido":
		return true
	}
	return false
}

// urnSugeridoES propone el identificador de paquete a partir de las piezas del
// ELI del BOE. No se cablea ninguna norma: entran el rango, el ano y el numero
// que traiga el ELI que se pida.
func urnSugeridoES(rango, ano, numero string) string {
	if rango == "" || ano == "" || numero == "" {
		return ""
	}
	return fmt.Sprintf("%s:%s:%s:%s:%s", "urn", "es", rango, ano, numero)
}

// urnSugeridoUE hace lo mismo desde un CELEX. El sector 3 (legislacion) lleva
// en la sexta posicion la letra del tipo de acto.
func urnSugeridoUE(celex string) string {
	c := strings.ToUpper(strings.TrimSpace(celex))
	if len(c) < 8 || c[0] != '3' {
		return "" // solo el sector de legislacion tiene forma de paquete
	}
	ano := c[1:5]
	tipo := map[byte]string{'R': "reg", 'L': "dir", 'D': "dec"}[c[5]]
	if tipo == "" {
		return ""
	}
	numero := strings.TrimLeft(c[6:], "0")
	if numero == "" {
		return ""
	}
	return fmt.Sprintf("%s:%s:%s:%s:%s", "urn", "eu", tipo, ano, numero)
}
