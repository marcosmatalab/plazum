package main

// EUR-Lex: articulado del DOUE por CELEX.
//
// LO QUE SE INVESTIGO ANTES DE ESCRIBIR ESTO, y que corrige lo que uno espera.
//
// La entrada "de manual" (eur-lex.europa.eu/legal-content/ES/TXT/XML/?uri=CELEX:...)
// YA NO SIRVE: comprobado contra dos reglamentos, devuelve HTTP 200 con la
// portada del Diario Oficial, no el XML. Es el peor fallo posible, porque un
// parser descuidado ve un 200 y se cree que la norma no tiene articulos.
//
// La entrada que si sirve es Cellar, el servicio de la Oficina de Publicaciones,
// por negociacion de contenido sobre publications.europa.eu/resource/celex/{CELEX}:
//
//	Accept: application/xml;notice=object   la ficha (titulo oficial, ELI, fecha
//	                                        de ultima modificacion). 7 KB.
//	Accept: application/xhtml+xml           el texto. Sirve para los actos
//	                                        antiguos, para los nuevos y para las
//	                                        versiones consolidadas.
//	Accept-Language: spa                    la version en castellano.
//
// El Formex (application/xml;type=fmx4) existe para los actos de hasta 2023 y NO
// para los posteriores (comprobado: un reglamento de 2024 responde 404 diciendo
// que no hay ese flujo). El XHTML esta en los tres casos, asi que hay un solo
// parser y no tres.
//
// Ese XHTML tiene dos dialectos, y los dos hay que aguantarlos:
//
//	original      <p class="oj-ti-art">, parrafos en <p class="oj-normal">
//	consolidado   <p class="title-article-norm">, parrafos en <div class="norm">
//	              con el numero de apartado en un <span class="no-parag">
//
// Lo que NO cambia entre dialectos, y por eso es de lo que se tira, es el
// esqueleto ELI: <div class="eli-subdivision" id="art_33"> con su
// <div class="eli-title" id="art_33.tit_1">. Los anexos son eli-container con
// id anx_I. Todo lo demas (recitales rct_N, citas cit_N, capitulos cpt_N) se
// deja fuera por el prefijo del id, no por la clase, porque en el consolidado
// los articulos van ANIDADOS dentro de contenedores de capitulo.

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

const (
	anfitrionCellar = "publications.europa.eu"
	baseCellar      = "https://publications.europa.eu/resource/celex/"
	anfitrionEURLex = "eur-lex.europa.eu"

	// LicenciaDOUE y AtribucionDOUE viajan pegadas al dato. La atribucion NO es
	// opcional: la Decision 2011/833/UE autoriza la reutilizacion y su art. 6.2.a
	// deja que se exija indicar la procedencia, y EUR-Lex la exige.
	LicenciaDOUE = "Reutilizacion autorizada por la Decision 2011/833/UE de la Comision, " +
		"de 12 de diciembre de 2011, sobre la reutilizacion de documentos de la Comision, " +
		"con indicacion de la fuente (art. 6.2.a)."
	AtribucionDOUE = "(c) Union Europea, https://eur-lex.europa.eu. Solo el texto publicado " +
		"en la edicion electronica del Diario Oficial de la Union Europea es autentico; " +
		"esta reproduccion no lo es."
)

// --- la ficha de Cellar ---

type noticiaCellar struct {
	Expresion struct {
		Titulos []struct {
			Valor string `xml:"VALUE"`
		} `xml:"EXPRESSION_TITLE"`
		UltimaModificacion struct {
			Valor string `xml:"VALUE"`
		} `xml:"LASTMODIFICATIONDATE"`
		Obra struct {
			IguaQue []struct {
				URI struct {
					Valor string `xml:"VALUE"`
					Tipo  string `xml:"TYPE"`
				} `xml:"URI"`
			} `xml:"SAMEAS"`
		} `xml:"EXPRESSION_BELONGS_TO_WORK"`
	} `xml:"EXPRESSION"`
}

// parsearNoticiaCellar saca titulo, ELI y fecha de ultima modificacion.
func parsearNoticiaCellar(b []byte) (titulo, eli, actualizada string, err error) {
	var n noticiaCellar
	if e := xml.Unmarshal(b, &n); e != nil {
		return "", "", "", fmt.Errorf("%w: la ficha de Cellar no es el XML de siempre (%v). "+
			"Arreglo: repite con -sin-cache y mira la respuesta cruda en la cache",
			ErrRespuestaIlegible, e)
	}
	for _, t := range n.Expresion.Titulos {
		if len(normalizarTexto(t.Valor)) > len(titulo) {
			titulo = normalizarTexto(t.Valor) // hay un titulo corto y el oficial; gana el largo
		}
	}
	for _, s := range n.Expresion.Obra.IguaQue {
		if s.URI.Tipo == "eli" {
			eli = eliDeCaraAlPublico(s.URI.Valor)
		}
	}
	actualizada = strings.TrimSpace(n.Expresion.UltimaModificacion.Valor)
	if titulo == "" {
		return "", "", "", fmt.Errorf("%w: la ficha de Cellar no trae titulo. "+
			"Arreglo: comprueba el CELEX en EUR-Lex; si el CELEX existe pero no tiene ficha, "+
			"la norma no esta publicada en castellano", ErrNormaNoEncontrada)
	}
	return titulo, eli, actualizada, nil
}

// eliDeCaraAlPublico pasa el ELI interno de la Oficina de Publicaciones al que
// resuelve en EUR-Lex, que es el que se pega en un paquete y el que abre un
// humano. Es un cambio de anfitrion sobre el mismo camino, no una invencion.
func eliDeCaraAlPublico(u string) string {
	const marca = "/resource/eli/"
	i := strings.Index(u, marca)
	if i < 0 {
		return u
	}
	return "https://" + anfitrionEURLex + "/eli/" + u[i+len(marca):]
}

// --- el CELEX ---

// validarCELEX comprueba la forma antes de meterlo en una URL. Es la puerta que
// impide que un identificador con barras o dos puntos salga del camino
// /resource/celex/ y acabe pidiendo cualquier otra cosa.
func validarCELEX(c string) (string, error) {
	s := strings.ToUpper(strings.TrimSpace(c))
	if len(s) < 8 || len(s) > 40 {
		return "", fmt.Errorf("%w: %q no tiene forma de CELEX. Arreglo: usa el numero CELEX "+
			"que sale en la ficha de EUR-Lex, por ejemplo 3AAAARNNNN para un reglamento",
			ErrIdentificadorInvalido, c)
	}
	if s[0] < '0' || s[0] > '9' {
		return "", fmt.Errorf("%w: %q no empieza por el digito del sector", ErrIdentificadorInvalido, c)
	}
	for i := 0; i < len(s); i++ {
		ch := s[i]
		ok := (ch >= '0' && ch <= '9') || (ch >= 'A' && ch <= 'Z') || ch == '-' || ch == '(' || ch == ')'
		if !ok {
			return "", fmt.Errorf("%w: %q lleva el caracter %q, que no cabe en un CELEX. "+
				"Arreglo: un CELEX solo tiene digitos, letras y, en el consolidado, un guion "+
				"con la fecha", ErrIdentificadorInvalido, c, string(ch))
		}
	}
	return s, nil
}

// --- el texto ---

// prefijos de id ELI que interesan, con el tipo que les corresponde.
var prefijosELI = map[string]string{"art_": "precepto", "anx_": "anexo"}

// clases que marcan el rotulo de la division (el "Articulo 33", el "ANEXO I").
// Las dos ramas del dialecto, y el prefijo de seccion por si el rotulo del anexo
// llega por ahi.
func esRotulo(clase string) bool {
	for _, c := range []string{"ti-art", "title-article", "doc-ti", "ti-section", "title-division"} {
		if strings.Contains(clase, c) {
			return true
		}
	}
	return false
}

// parsearXHTMLEurLex convierte el XHTML de Cellar en articulos.
func parsearXHTMLEurLex(cuerpo []byte, titulo, urlDoc string) ([]Articulo, error) {
	dec := xml.NewDecoder(bytes.NewReader(cuerpo))
	dec.Entity = xml.HTMLEntity
	dec.Strict = false // el XHTML de la OP declara una DTD externa que no se resuelve

	corta := citaCorta(titulo)
	var (
		out      []Articulo
		ac       *acumulador
		actual   Articulo
		profArt  int
		enTitulo int
		rotulos  []trozo // los trozos cerrados antes del eli-title
		prof     int
		vistos   int
	)
	abierto := false

	cerrarArticuloActual := func() {
		if !abierto {
			return
		}
		ac.cerrar(profArt) // lo que quedara sin volcar al cerrarse la division
		repartirTrozosEurLex(&actual, rotulos, ac.trozos)
		out = append(out, actual)
		abierto, ac, rotulos = false, nil, nil
	}

	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: el XHTML de EUR-Lex no se puede recorrer (%v)",
				ErrRespuestaIlegible, err)
		}
		vistos++
		if vistos > maxTokensPorDocumento {
			return nil, fmt.Errorf("%w: el documento pasa de %d elementos; se descarta por si "+
				"es una entrada adversaria", ErrRespuestaIlegible, maxTokensPorDocumento)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			prof++
			nombre := t.Name.Local
			if !abierto {
				if tipo, id, ok := divisionELI(t); ok {
					abierto = true
					ac = &acumulador{}
					actual = Articulo{ID: id, Tipo: tipo}
					profArt, enTitulo, rotulos = prof, 0, nil
				}
				continue
			}
			// dentro de un articulo
			if nombre == "div" && strings.Contains(atributo(t, "class"), "eli-title") {
				ac.cerrar(prof)
				rotulos = append(rotulos, ac.trozos...)
				ac.trozos = nil
				enTitulo = prof
				continue
			}
			switch nombre {
			case "p", "li":
				ac.cerrar(prof)
				ac.marcarClase(atributo(t, "class"))
			case "table":
				ac.cerrar(prof)
				ac.enTabla++
			case "tr", "td", "th":
				ac.cerrar(prof)
			case "img":
				ac.literal(" [imagen omitida] ")
			case "br":
				ac.literal(" ")
			case "span":
				ac.marcarClase(atributo(t, "class"))
			}
		case xml.EndElement:
			if abierto {
				nombre := t.Name.Local
				switch nombre {
				case "p", "li":
					ac.cerrar(prof)
				case "tr":
					ac.emitirFila(prof)
				case "table":
					ac.emitirFila(prof)
					if ac.enTabla > 0 {
						ac.enTabla--
					}
				}
				if enTitulo > 0 && prof == enTitulo {
					ac.cerrar(prof)
					actual.Titulo = juntar(ac.trozos)
					ac.trozos = nil
					enTitulo = 0
				} else if prof == profArt+1 {
					ac.cerrar(prof)
				}
				if prof == profArt {
					cerrarArticuloActual()
				}
			}
			prof--
		case xml.CharData:
			if abierto {
				ac.texto(t)
			}
		}
	}
	cerrarArticuloActual()
	cerrarTodos(out, corta, urlDoc)
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: no hay ni un div con id art_N en el documento. "+
			"Arreglo: comprueba que el CELEX es de un acto legislativo y no de un anuncio; "+
			"si lo es, EUR-Lex ha cambiado el marcado ELI y hay que revisar el parser",
			ErrSinArticulos)
	}
	return out, nil
}

// maxTokensPorDocumento: el reglamento mas largo del corpus anda por los
// doscientos mil elementos, asi que dos millones deja sitio y sigue siendo techo.
const maxTokensPorDocumento = 2000000

// divisionELI dice si un elemento abre una division que interesa y de que tipo.
func divisionELI(e xml.StartElement) (tipo, id string, ok bool) {
	if e.Name.Local != "div" {
		return "", "", false
	}
	id = atributo(e, "id")
	for p, t := range prefijosELI {
		if strings.HasPrefix(id, p) && !strings.Contains(id, ".") {
			return t, id, true
		}
	}
	return "", "", false
}

// repartirTrozosEurLex separa el rotulo (la referencia) del cuerpo.
//
// El rotulo es el primer trozo cerrado antes del eli-title. Se acepta por su
// clase cuando la fuente la marca, y por posicion y longitud cuando no: un
// rotulo de division es corto por definicion, y confundirlo con el primer
// parrafo se nota inmediatamente porque la referencia sale con doscientos
// caracteres.
func repartirTrozosEurLex(a *Articulo, rotulos, cuerpo []trozo) {
	// Sin eli-title (los anexos) el rotulo va en el cuerpo, de cabeza.
	cabeza := rotulos
	if len(cabeza) == 0 {
		cabeza, cuerpo = cuerpo, nil
	}
	if len(cabeza) > 0 && (esRotulo(cabeza[0].Clase) || len(cabeza[0].Texto) <= maxRotulo) {
		a.Referencia = cabeza[0].Texto
		// Anexos del dialecto original: el rotulo y el titulo del anexo son dos
		// parrafos seguidos de LA MISMA clase, sin eli-title de por medio. Solo
		// se toma el segundo como titulo cuando la clase coincide; si no, es ya
		// texto del anexo y llevarselo al titulo seria perderlo.
		if a.Titulo == "" && len(cabeza) > 1 && cabeza[1].Clase == cabeza[0].Clase {
			a.Titulo = cabeza[1].Texto
			cabeza = cabeza[1:]
		}
		cabeza = cabeza[1:]
	}
	if a.Referencia == "" {
		// Sin rotulo legible se usa el id de la fuente. Es feo pero es estable y
		// es verdad; inventarse la palabra "Articulo" seria meter en una cita
		// legal texto que el legislador no escribio.
		a.Referencia = a.ID
	}
	for _, t := range cabeza {
		a.Parrafos = append(a.Parrafos, t.Texto)
	}
	for _, t := range cuerpo {
		a.Parrafos = append(a.Parrafos, t.Texto)
	}
	if derogadoPorTexto(strings.Join(a.Parrafos, " ")) {
		a.Derogado = true
	}
}

// maxRotulo: un rotulo de division mas largo que esto no es un rotulo.
const maxRotulo = 80

func juntar(ts []trozo) string {
	partes := make([]string, 0, len(ts))
	for _, t := range ts {
		if t.Texto != "" {
			partes = append(partes, t.Texto)
		}
	}
	return strings.Join(partes, " ")
}

// origenDeCELEX rellena la ficha de procedencia de una norma del DOUE.
func origenDeCELEX(celex, titulo, eli, actualizada, urlDoc string) Origen {
	return Origen{
		Jurisdiccion:  "ue",
		Identificador: celex,
		ELI:           eli,
		URLDocumento:  urlDoc,
		Titulo:        titulo,
		CitaCorta:     citaCorta(titulo),
		URNSugerido:   urnSugeridoUE(celex),
		// Un CELEX con guion y fecha es una version consolidada; sin guion es el
		// acto tal cual se publico.
		Consolidado:   strings.Contains(celex, "-"),
		ActualizadaEn: actualizada,
	}
}
