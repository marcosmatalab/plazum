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

// borradorPaquete es la misma forma que corpus.Paquete mas los dos campos de
// procedencia que la ingesta ya sabe rellenar.
type borradorPaquete struct {
	URN     string `json:"urn"`
	Version string `json:"version"`
	Clase   int    `json:"clase"`
	// Licencia es el campo que ya existe en el formato. LicenciaFuente y
	// Atribucion son los dos que llegan con la ingesta y que un frente
	// posterior hara obligatorios; salen ya con el nombre definitivo.
	Licencia       string             `json:"licencia"`
	LicenciaFuente string             `json:"licencia_fuente"`
	Atribucion     string             `json:"atribucion"`
	Fuente         string             `json:"fuente"`
	Consolidado    bool               `json:"consolidado"`
	Vigencia       vigenciaBorrador   `json:"vigencia"`
	Obligaciones   []obligacionBorrad `json:"obligaciones"`
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
		LicenciaFuente: e.LicenciaFuente,
		Atribucion:     e.Atribucion,
		Fuente:         e.Fuente.URLDocumento,
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
