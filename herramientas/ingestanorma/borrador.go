package main

// El borrador de paquete: el puente entre la extraccion y el trabajo de autoria.
//
// POR QUE ESTA DELIBERADAMENTE INCOMPLETO. Un paquete de corpus lo autoriza una
// persona, siempre: es la frontera entre "he bajado un texto" y "afirmo que esto
// obliga a alguien y este es su reloj". Un borrador que cargara tal cual seria
// una invitacion a commitear derecho sin leerlo.
//
// Asi que faltan a proposito, y son dos huecos INDEPENDIENTES:
//
//	id         cada obligacion sale sin identificador -> el linter lo rechaza
//	clase_e2e  cada obligacion sale sin clase        -> el linter lo rechaza
//
// Dos y no uno porque una sola comprobacion se puede quitar de un tiron sin
// darse cuenta. Un test lo fija: el borrador tiene que casar con el formato de
// paquete Y tiene que NO validar.
//
// Lo que si sale hecho es lo que no requiere criterio: la cita, el enlace con
// ancla al articulo, la vigencia, el texto legal transcrito, y la licencia y la
// atribucion de la fuente. Eso es justo lo que se copia mal a mano.

import "strings"

// claseTranscrito es corpus.Transcrito. Se escribe aqui como numero, y no se
// importa el paquete, para que la herramienta no arrastre al nucleo dentro de un
// binario de linea de comandos; el test de borrador lo cruza contra el tipo real
// para que este numero no pueda quedarse obsoleto en silencio.
const claseTranscrito = 1

// Los identificadores del vocabulario CERRADO de licencia_fuente. Van aqui como
// cadena por la misma razon que claseTranscrito, y con la misma red: el test del
// borrador los cruza contra las constantes de corpus, asi que no pueden
// quedarse obsoletos en silencio.
//
// Solo hay dos porque un borrador de ingesta es siempre transcrito: o viene del
// BOE o viene del DOUE. El regimen es el identificador; el parrafo que lo
// explica va en `licencia`, que es texto libre, y el aviso que hay que ENSENAR
// va en `atribucion`. Los tres son cosas distintas. Ver docs/LICENCIAS.md.
const (
	regimenBOE  = "boe-trlpi-13"
	regimenDOUE = "doue-decision-2011-833"
)

// Los identificadores del vocabulario CERRADO de identificador.tipo, con la
// misma red que los de arriba: el test los cruza contra las constantes de
// corpus. Solo hay dos porque un borrador de ingesta es siempre transcrito, y
// las dos fuentes primarias que se ingieren publican ELI.
//
// POR QUE EL BORRADOR NO ESCRIBE UNA URL. El formato de paquete guarda la
// procedencia como IDENTIFICADOR y deriva la direccion al pintar: una URL como
// dato se rompe el dia que la pagina se mueve. La extraccion trae el ELI
// completo (con anfitrion, que es como lo publica la fuente), asi que aqui se
// le quita el anfitrion y se queda la ruta, que es el identificador.
const (
	esquemaELIBOE = "eli-boe"
	esquemaELIUE  = "eli-ue"
)

// anfitrionesELI son los comienzos que hay que quitarle al ELI que publica la
// fuente para quedarse con la ruta. Se escriben con y sin "www" y con los dos
// esquemas porque los dos portales han servido las dos formas.
var anfitrionesELI = []string{
	"https://eur-lex.europa.eu/eli/", "http://eur-lex.europa.eu/eli/",
	"https://www.boe.es/eli/", "http://www.boe.es/eli/",
	"https://data.europa.eu/eli/", "http://data.europa.eu/eli/",
	"https://boe.es/eli/", "http://boe.es/eli/",
}

// identificadorBorrador es el bloque `identificador` de un paquete.
type identificadorBorrador struct {
	Tipo  string `json:"tipo"`
	Valor string `json:"valor"`
}

// rutaELI deja el ELI en su ruta: le quita el anfitrion si lo trae y las
// barras sobrantes. Devuelve cadena vacia si no hay ELI, y entonces el borrador
// sale con el identificador incompleto, que es un hueco VISIBLE (el linter lo
// rechaza) en vez de una URL colada por la puerta de atras.
func rutaELI(eli string) string {
	v := strings.TrimSpace(eli)
	for _, p := range anfitrionesELI {
		if strings.HasPrefix(v, p) {
			v = v[len(p):]
			break
		}
	}
	if strings.Contains(v, "://") {
		return "" // no se sabe de donde sale: mejor vacio que inventado
	}
	return strings.Trim(v, "/")
}

// identificadorDe construye el bloque desde la procedencia de la extraccion.
func identificadorDe(o Origen) identificadorBorrador {
	tipo := esquemaELIBOE
	if o.Jurisdiccion == "ue" {
		tipo = esquemaELIUE
	}
	eli := o.ELI
	if eli == "" {
		eli = o.URLDocumento
	}
	return identificadorBorrador{Tipo: tipo, Valor: rutaELI(eli)}
}

// regimenDe traduce la jurisdiccion de la fuente al identificador del
// vocabulario. Por defecto el espanol: una jurisdiccion desconocida no puede
// acabar declarando el regimen de la Union.
func regimenDe(jurisdiccion string) string {
	if jurisdiccion == "ue" {
		return regimenDOUE
	}
	return regimenBOE
}

// borradorPaquete es la misma forma que corpus.Paquete mas los dos campos de
// procedencia que la ingesta ya sabe rellenar.
type borradorPaquete struct {
	URN     string `json:"urn"`
	Version string `json:"version"`
	Clase   int    `json:"clase"`
	// Licencia es el campo de texto libre que ya existe en el formato, donde
	// cabe el parrafo que explica el regimen. LicenciaFuente es el
	// IDENTIFICADOR de ese regimen, de un vocabulario cerrado, y Atribucion es
	// el aviso que el producto ensena. Los tres son obligatorios desde el
	// 26-08-2026: un paquete sin los dos ultimos no carga.
	Licencia       string `json:"licencia"`
	LicenciaFuente string `json:"licencia_fuente"`
	Atribucion     string `json:"atribucion"`
	// Identificador sustituye al campo `fuente` del formato viejo, que llevaba
	// la URL completa. La direccion se deriva del identificador al pintar.
	Identificador identificadorBorrador `json:"identificador"`
	Consolidado   bool                  `json:"consolidado"`
	Vigencia      vigenciaBorrador      `json:"vigencia"`
	Obligaciones  []obligacionBorrad    `json:"obligaciones"`
}

type vigenciaBorrador struct {
	Desde string `json:"desde"`
	Hasta string `json:"hasta,omitempty"`
}

type obligacionBorrad struct {
	// ID vacio A PROPOSITO: lo escribe quien autora, y hasta que lo escriba el
	// paquete no carga.
	ID       string `json:"id"`
	Articulo string `json:"articulo"`
	// ClaseE2E vacia A PROPOSITO: decidir si una obligacion es observable,
	// documental, procedimental, notificatoria o de remediacion es criterio
	// juridico, no una transformacion de texto.
	ClaseE2E   string           `json:"clase_e2e"`
	Titulo     string           `json:"titulo,omitempty"`
	TextoLegal string           `json:"texto_legal"`
	Cita       string           `json:"cita"`
	Fuente     string           `json:"fuente"`
	Vigencia   vigenciaBorrador `json:"vigencia"`
	// Nota lleva lo que la fuente dice sobre el articulo (por que norma se
	// modifico, cuando se derogo). No es del formato de paquete: esta aqui para
	// que quien autora lo lea y lo borre.
	Nota string `json:"_nota_de_la_fuente,omitempty"`
}

func borradorDe(e *Extraccion) borradorPaquete {
	b := borradorPaquete{
		URN:            e.Fuente.URNSugerido,
		Version:        "0.0.0-borrador",
		Clase:          claseTranscrito,
		Licencia:       e.LicenciaFuente,
		LicenciaFuente: regimenDe(e.Fuente.Jurisdiccion),
		Atribucion:     e.Atribucion,
		Identificador:  identificadorDe(e.Fuente),
		Consolidado:    e.Fuente.Consolidado,
		Vigencia:       vigenciaBorrador{Desde: e.Fuente.FechaVigencia},
	}
	if e.Fuente.Derogada && e.Fuente.FechaDerogacion != "" {
		b.Vigencia.Hasta = e.Fuente.FechaDerogacion
	}
	for _, a := range e.Articulos {
		if a.Derogado {
			continue // un articulo derogado no genera obligacion; se ve en la extraccion
		}
		desde := a.VigenciaDesde
		if desde == "" {
			desde = e.Fuente.FechaVigencia
		}
		b.Obligaciones = append(b.Obligaciones, obligacionBorrad{
			Articulo:   a.Referencia,
			Titulo:     a.Titulo,
			TextoLegal: strings.ReplaceAll(a.Texto, "\n\n", " "),
			Cita:       a.Cita,
			Fuente:     a.Fuente,
			Vigencia:   vigenciaBorrador{Desde: desde},
			Nota:       a.Nota,
		})
	}
	return b
}
