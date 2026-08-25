package main

// BOE: legislacion consolidada por la API de datos abiertos.
//
// LO QUE SE INVESTIGO ANTES DE ESCRIBIR ESTO, para que nadie tenga que repetirlo.
// La API vive en /datosabiertos/api/legislacion-consolidada y tiene seis
// entradas; aqui se usan tres:
//
//	/id/{id}/metadatos   titulo, rango, fechas, derogacion, url_eli y la marca de
//	                     ultima actualizacion, que es la mitad izquierda de la
//	                     tabla de vigilancia.
//	/id/{id}/texto       el texto consolidado con TODAS sus versiones.
//	?query=...           la busqueda, que es como se resuelve un ELI a su id.
//
// El texto se estructura en <bloque> (un articulo, una disposicion, un anexo) y
// cada bloque tiene una o mas <version>, una por cada vez que lo han modificado,
// con la fecha de entrada en vigor y el id de la norma modificadora. De ahi sale
// todo lo demas: el texto vigente a una fecha, quien lo modifico y desde cuando.
//
// EL ELI NO SE RESUELVE RASCANDO HTML. La pagina del ELI trae el identificador
// dentro, pero eso es HTML de presentacion y cambia sin avisar. Se resuelve por
// la API: se busca por fecha de disposicion y se casa contra el campo url_eli
// que la propia API devuelve. Si ninguno casa, se dice que no se encontro; no se
// coge "el que mas se parece".

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"strings"
)

const (
	anfitrionBOE = "www.boe.es"
	baseAPIBOE   = "https://www.boe.es/datosabiertos/api/legislacion-consolidada"

	// LicenciaBOE y AtribucionBOE viajan pegadas al dato. La segunda frase de
	// la atribucion no es cortesia: las condiciones de reutilizacion del BOE
	// exigen que todo texto consolidado reutilizado diga expresamente que es un
	// texto consolidado de caracter meramente informativo.
	LicenciaBOE = "Texto legal reproducible: art. 13 TRLPI (las disposiciones legales no son " +
		"objeto de propiedad intelectual) y condiciones de reutilizacion de la Agencia Estatal BOE."
	AtribucionBOE = "Fuente de los datos: Agencia Estatal Boletin Oficial del Estado " +
		"(https://www.boe.es). Texto consolidado de caracter meramente informativo. " +
		"Esta reutilizacion no tiene caracter oficial."
)

// --- respuestas de la API, solo los campos que se usan ---

type sobreBOE struct {
	Codigo string `xml:"status>code"`
	Texto  string `xml:"status>text"`
}

type respuestaMetadatosBOE struct {
	sobreBOE
	Meta struct {
		FechaActualizacion string `xml:"fecha_actualizacion"`
		Identificador      string `xml:"identificador"`
		Rango              string `xml:"rango"`
		FechaDisposicion   string `xml:"fecha_disposicion"`
		NumeroOficial      string `xml:"numero_oficial"`
		Titulo             string `xml:"titulo"`
		FechaPublicacion   string `xml:"fecha_publicacion"`
		FechaVigencia      string `xml:"fecha_vigencia"`
		EstatusDerogacion  string `xml:"estatus_derogacion"`
		FechaDerogacion    string `xml:"fecha_derogacion"`
		URLELI             string `xml:"url_eli"`
		URLHTML            string `xml:"url_html_consolidada"`
	} `xml:"data>metadatos"`
}

type respuestaBusquedaBOE struct {
	sobreBOE
	Items []struct {
		Identificador string `xml:"identificador"`
		Titulo        string `xml:"titulo"`
		URLELI        string `xml:"url_eli"`
	} `xml:"data>item"`
}

type respuestaTextoBOE struct {
	sobreBOE
	Bloques []bloqueBOE `xml:"data>texto>bloque"`
}

type bloqueBOE struct {
	ID             string       `xml:"id,attr"`
	Tipo           string       `xml:"tipo,attr"`
	Titulo         string       `xml:"titulo,attr"`
	FechaCaducidad string       `xml:"fecha_caducidad,attr"`
	Versiones      []versionBOE `xml:"version"`
}

type versionBOE struct {
	IDNorma          string `xml:"id_norma,attr"`
	FechaPublicacion string `xml:"fecha_publicacion,attr"`
	FechaVigencia    string `xml:"fecha_vigencia,attr"`
	Cuerpo           []byte `xml:",innerxml"`
}

// decodificarBOE desenvuelve una respuesta de la API y traduce el codigo de
// estado a un error del dominio. El codigo viaja DENTRO del XML: la API contesta
// HTTP 200 con un 404 dentro, asi que mirar solo el HTTP deja pasar el vacio.
func decodificarBOE(b []byte, destino interface{ estado() (string, string) }) error {
	if err := xml.Unmarshal(b, destino); err != nil {
		return fmt.Errorf("%w: el BOE devolvio algo que no es su XML de siempre (%v). "+
			"Arreglo: repite con -sin-cache; si persiste, mira la respuesta cruda en la cache",
			ErrRespuestaIlegible, err)
	}
	codigo, texto := destino.estado()
	switch codigo {
	case "", "200":
		return nil
	case "404":
		return fmt.Errorf("%w: el BOE responde 404 (%s). Arreglo: comprueba el identificador; "+
			"solo hay texto consolidado de las normas que el BOE consolida", ErrNormaNoEncontrada, texto)
	case "400":
		return fmt.Errorf("%w: el BOE responde 400 (%s). Arreglo: el identificador tiene la forma "+
			"BOE-A-AAAA-NNNNN", ErrIdentificadorInvalido, texto)
	default:
		return fmt.Errorf("%w: el BOE responde %s (%s)", ErrRespuestaIlegible, codigo, texto)
	}
}

func (s sobreBOE) estado() (string, string) { return s.Codigo, s.Texto }

// --- ELI ---

// piezasELI son las partes de un ELI del BOE que hacen falta para buscarlo y
// para proponer un URN de paquete.
type piezasELI struct {
	Rango  string // rd, l, lo, rdl...
	Ano    string
	Mes    string
	Dia    string
	Numero string
	Base   string // el ELI sin sufijos de version, tal cual se compara
}

// partirELI lee https://www.boe.es/eli/es/{rango}/{aaaa}/{mm}/{dd}/{num}[/con...]
//
// Se comprueba el ANFITRION, y esa comprobacion es la frontera legal metida en
// codigo: solo se ingiere de la fuente primaria. Un espejo en GitHub con
// licencia permisiva no vale, porque quien lo subio no era dueno del texto.
func partirELI(crudo string) (piezasELI, error) {
	u, err := url.Parse(strings.TrimSpace(crudo))
	if err != nil {
		return piezasELI{}, fmt.Errorf("%w: %q no es una URL (%v)", ErrIdentificadorInvalido, crudo, err)
	}
	if !anfitrionAutorizado(u.Host) {
		return piezasELI{}, fmt.Errorf("%w: %q. Solo se ingiere de fuente primaria (%s). "+
			"La licencia de un repositorio de terceros no alcanza al texto normativo que "+
			"quien lo subio no poseia",
			ErrFuenteNoAutorizada, u.Host, strings.Join(anfitrionesPrimarios, ", "))
	}
	partes := strings.Split(strings.Trim(u.Path, "/"), "/")
	// eli / es / rango / aaaa / mm / dd / numero
	if len(partes) < 7 || partes[0] != "eli" {
		return piezasELI{}, fmt.Errorf("%w: %q no tiene forma de ELI del BOE. "+
			"Arreglo: usa la forma https://%s/eli/es/RANGO/AAAA/MM/DD/NUMERO (la que sale en "+
			"la ficha de la norma, campo ELI)", ErrIdentificadorInvalido, crudo, anfitrionBOE)
	}
	p := piezasELI{Rango: partes[2], Ano: partes[3], Mes: partes[4], Dia: partes[5], Numero: partes[6]}
	if len(p.Ano) != 4 || len(p.Mes) != 2 || len(p.Dia) != 2 {
		return piezasELI{}, fmt.Errorf("%w: %q tiene la fecha mal formada. "+
			"Arreglo: en el ELI la fecha va con cuatro digitos de ano y dos de mes y dia "+
			"(AAAA/MM/DD), y es la fecha de la DISPOSICION, no la de publicacion",
			ErrIdentificadorInvalido, crudo)
	}
	p.Base = fmt.Sprintf("https://%s/eli/%s/%s/%s/%s/%s/%s",
		anfitrionBOE, partes[1], p.Rango, p.Ano, p.Mes, p.Dia, p.Numero)
	return p, nil
}

// urlBusquedaPorFecha construye la consulta que devuelve todas las normas
// consolidadas dictadas ese dia. Son pocas, y luego se casa por url_eli, que es
// una comparacion exacta y no una heuristica de parecido.
func urlBusquedaPorFecha(p piezasELI) string {
	q := fmt.Sprintf(`{"query":{"query_string":{"query":"fecha_disposicion:%s%s%s"}}}`, p.Ano, p.Mes, p.Dia)
	v := url.Values{}
	v.Set("query", q)
	v.Set("limit", "-1")
	return baseAPIBOE + "?" + v.Encode()
}

// resolverELIBOE elige, entre lo que devolvio la busqueda, la norma cuyo ELI
// coincide exactamente con el pedido.
func resolverELIBOE(cuerpo []byte, p piezasELI) (string, error) {
	var r respuestaBusquedaBOE
	if err := decodificarBOE(cuerpo, &r); err != nil {
		return "", err
	}
	for _, it := range r.Items {
		if mismoELI(it.URLELI, p.Base) {
			return it.Identificador, nil
		}
	}
	return "", fmt.Errorf("%w: ninguna de las %d normas dictadas el %s-%s-%s tiene el ELI %s. "+
		"Arreglo: comprueba el ELI en la ficha de la norma; ojo a la fecha, que en el ELI es la "+
		"de la DISPOSICION y no la de publicacion",
		ErrNormaNoEncontrada, len(r.Items), p.Dia, p.Mes, p.Ano, p.Base)
}

// mismoELI compara dos ELI ignorando el esquema y los sufijos de version
// (/con, /dof, /spa) que el BOE cuelga del mismo identificador.
func mismoELI(a, b string) bool {
	corta := func(s string) string {
		s = strings.TrimSpace(strings.ToLower(s))
		s = strings.TrimPrefix(strings.TrimPrefix(s, "https://"), "http://")
		return strings.TrimRight(s, "/")
	}
	return corta(a) == corta(b)
}

// --- texto ---

// opcionesBOE gobierna que bloques salen y que version de cada uno.
type opcionesBOE struct {
	// Referencia es la fecha (AAAAMMDD) a la que se quiere el texto. Nunca
	// vacia: quien llama pone la de hoy si no le pidieron otra. Sin fecha de
	// referencia no se puede decidir si un bloque caducado sigue mostrandose.
	Referencia string
	// Todo incluye tambien preambulo, firma y encabezados de estructura.
	Todo bool
}

// parsearTextoBOE convierte el XML del texto consolidado en articulos.
func parsearTextoBOE(cuerpo []byte, meta respuestaMetadatosBOE, urlDoc string, op opcionesBOE) ([]Articulo, error) {
	var r respuestaTextoBOE
	if err := decodificarBOE(cuerpo, &r); err != nil {
		return nil, err
	}
	corta := citaCorta(meta.Meta.Titulo)
	out := make([]Articulo, 0, len(r.Bloques))
	for _, b := range r.Bloques {
		tipo := tipoDeBloque(b)
		if !op.Todo && (tipo == "estructura" || tipo == "preambulo" || tipo == "firma") {
			continue
		}
		v, ok := versionVigente(b, op.Referencia)
		if !ok {
			continue // el bloque no existia todavia a la fecha pedida
		}
		ps, err := parrafosDeVersionBOE(v.Cuerpo)
		if err != nil {
			return nil, fmt.Errorf("%w: bloque %q: %v", ErrRespuestaIlegible, b.ID, err)
		}
		a := Articulo{
			Referencia:    b.Titulo,
			ID:            b.ID,
			Tipo:          tipo,
			VigenciaDesde: guion(v.FechaVigencia),
		}
		if v.IDNorma != "" && v.IDNorma != meta.Meta.Identificador {
			a.ModificadoPor = v.IDNorma
		}
		repartirParrafosBOE(&a, ps)
		if b.FechaCaducidad != "" && b.FechaCaducidad <= op.Referencia {
			a.Derogado = true
		}
		out = append(out, a)
	}
	cerrarTodos(out, corta, urlDoc)
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: %d bloques leidos y ninguno dio articulo. "+
			"Arreglo: prueba con -todo; si tampoco sale nada, el BOE ha cambiado el formato del "+
			"nodo texto y hay que revisar el parser", ErrSinArticulos, len(r.Bloques))
	}
	return out, nil
}

// tipoDeBloque traduce el atributo tipo del BOE a lo que le importa a quien
// autora un paquete. Los anexos llegan como "encabezado" igual que un
// "CAPITULO I", asi que se distinguen por la clase del primer parrafo
// (anexo_num), que es marcado de la fuente y no una corazonada sobre el titulo.
func tipoDeBloque(b bloqueBOE) string {
	switch b.Tipo {
	case "precepto":
		return "precepto"
	case "preambulo":
		return "preambulo"
	case "firma":
		return "firma"
	case "encabezado":
		for _, v := range b.Versiones {
			if bytes.Contains(v.Cuerpo, []byte(`class="anexo_num"`)) {
				return "anexo"
			}
		}
		return "estructura"
	default:
		return b.Tipo
	}
}

// versionVigente elige la version del bloque en vigor a la fecha de referencia:
// la ultima cuya entrada en vigor no sea posterior. Una version sin fecha de
// vigencia se toma como vigente desde siempre, que es como la publica el BOE
// para el texto original.
func versionVigente(b bloqueBOE, referencia string) (versionBOE, bool) {
	var elegida versionBOE
	encontrada := false
	for _, v := range b.Versiones {
		if v.FechaVigencia != "" && referencia != "" && v.FechaVigencia > referencia {
			continue
		}
		if encontrada && v.FechaVigencia < elegida.FechaVigencia {
			continue // desordenadas: gana la mas reciente que no sea futura
		}
		elegida, encontrada = v, true
	}
	return elegida, encontrada
}

// clases del BOE que son encabezado del bloque y no cuerpo.
var clasesEncabezado = map[string]bool{
	"articulo": true, "sangrado_articulo": true, "anexo_num": true,
	"capitulo_num": true, "titulo_num": true, "seccion": true, "subseccion": true,
}

// clases que son el titulo corto de un anexo o capitulo.
var clasesTitulo = map[string]bool{
	"anexo_tit": true, "capitulo_tit": true, "titulo_tit": true,
}

// repartirParrafosBOE separa encabezado, titulo, notas y cuerpo.
func repartirParrafosBOE(a *Articulo, ps []trozo) {
	var encabezado, titulo string
	var notas []string
	for _, p := range ps {
		switch {
		case p.Nota || p.Clase == "nota_pie":
			notas = append(notas, p.Texto)
		case encabezado == "" && clasesEncabezado[p.Clase]:
			encabezado = p.Texto
		case clasesTitulo[p.Clase] && titulo == "":
			titulo = p.Texto
		default:
			a.Parrafos = append(a.Parrafos, p.Texto)
		}
	}
	if titulo == "" {
		titulo = tituloTrasLaReferencia(encabezado, a.Referencia)
	}
	a.Titulo = titulo
	a.Nota = strings.Join(notas, " ")
	if derogadoPorTexto(strings.Join(a.Parrafos, " ")) {
		a.Derogado = true
	}
}

// parrafosDeVersionBOE recorre el HTML incrustado en una <version>. Lo que
// decide donde acaba un parrafo, en el BOE, es el elemento <p>: la fuente marca
// cada uno con su clase y no anida bloques dentro de bloques.
func parrafosDeVersionBOE(cuerpo []byte) ([]trozo, error) {
	dec := xml.NewDecoder(bytes.NewReader(cuerpo))
	dec.Entity = xml.HTMLEntity
	ac := &acumulador{}
	vistos := 0
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		// Freno contra una entrada adversaria: un cuerpo hinchado a proposito
		// no puede tumbar la herramienta ni llenar la memoria.
		vistos++
		if vistos > maxTokensPorBloque {
			return nil, fmt.Errorf("el bloque pasa de %d elementos; se descarta por si es "+
				"una entrada adversaria", maxTokensPorBloque)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p", "li":
				ac.cerrar(0)
				ac.marcarClase(atributo(t, "class"))
			case "table":
				ac.cerrar(0)
				ac.enTabla++
			case "tr", "td", "th":
				ac.cerrar(0)
			case "blockquote":
				ac.cerrar(0)
				ac.enNota++
			case "img":
				ac.literal(" [imagen omitida] ")
			case "br":
				ac.literal(" ")
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "p", "li":
				ac.cerrar(0)
			case "tr":
				ac.emitirFila(0)
			case "table":
				ac.emitirFila(0)
				if ac.enTabla > 0 {
					ac.enTabla--
				}
			case "blockquote":
				ac.cerrar(0)
				if ac.enNota > 0 {
					ac.enNota--
				}
			}
		case xml.CharData:
			ac.texto(t)
		}
	}
	ac.cerrar(0)
	return ac.trozos, nil
}

// maxTokensPorBloque es el freno contra un cuerpo hinchado a proposito. El anexo
// mas grande del corpus de hoy anda por los veinte mil, asi que doscientos mil
// deja sitio de sobra y sigue siendo un techo.
const maxTokensPorBloque = 200000

func atributo(e xml.StartElement, nombre string) string {
	for _, a := range e.Attr {
		if a.Name.Local == nombre {
			return a.Value
		}
	}
	return ""
}

// guion pasa AAAAMMDD a AAAA-MM-DD, que es como se escriben las fechas en todo
// el resto del proyecto. Lo que no tenga ocho digitos se devuelve tal cual: no
// se inventa una fecha a partir de algo que no la parece.
func guion(s string) string {
	if len(s) != 8 {
		return s
	}
	for i := 0; i < 8; i++ {
		if s[i] < '0' || s[i] > '9' {
			return s
		}
	}
	return s[:4] + "-" + s[4:6] + "-" + s[6:]
}

// instanteISO pasa la marca compacta del BOE (AAAAMMDDTHHmmSSZ, que es ISO 8601
// basico y valido) a la forma extendida, que es la que usa el resto del proyecto
// y la que se puede comparar y ordenar junto a la que devuelve EUR-Lex. La tabla
// publica de vigilancia mezcla las dos fuentes en la misma columna.
//
// Lo que no tenga esa forma exacta se devuelve tal cual: no se inventa una fecha.
func instanteISO(s string) string {
	if len(s) != 16 || s[8] != 'T' || s[15] != 'Z' {
		return s
	}
	for _, i := range []int{0, 1, 2, 3, 4, 5, 6, 7, 9, 10, 11, 12, 13, 14} {
		if s[i] < '0' || s[i] > '9' {
			return s
		}
	}
	return s[:4] + "-" + s[4:6] + "-" + s[6:8] + "T" +
		s[9:11] + ":" + s[11:13] + ":" + s[13:15] + "Z"
}

// origenDeMetadatos rellena la ficha de procedencia con lo que dice el BOE.
func origenDeMetadatos(m respuestaMetadatosBOE, p piezasELI, urlDoc, aFecha string) Origen {
	md := m.Meta
	eli := md.URLELI
	if eli == "" {
		eli = p.Base
	}
	numero := md.NumeroOficial
	if i := strings.Index(numero, "/"); i > 0 {
		numero = numero[:i]
	}
	return Origen{
		Jurisdiccion:     "es",
		Identificador:    md.Identificador,
		ELI:              eli,
		URLDocumento:     urlDoc,
		Titulo:           normalizarTexto(md.Titulo),
		CitaCorta:        citaCorta(md.Titulo),
		URNSugerido:      urnSugeridoES(p.Rango, p.Ano, numero),
		Consolidado:      true,
		FechaDisposicion: guion(md.FechaDisposicion),
		FechaPublicacion: guion(md.FechaPublicacion),
		FechaVigencia:    guion(md.FechaVigencia),
		Derogada:         md.EstatusDerogacion == "S",
		FechaDerogacion:  guion(md.FechaDerogacion),
		ActualizadaEn:    instanteISO(md.FechaActualizacion),
		TextoVigenteEn:   aFecha,
	}
}
