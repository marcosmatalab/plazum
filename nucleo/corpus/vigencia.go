package corpus

import (
	"errors"
	"fmt"
	"strings"
)

// De quien es la fecha de vigencia de una obligacion.
const (
	// VigenciaHeredada: la fecha es la del paquete porque la norma tiene UNA
	// SOLA fecha aplicable a este punto, y alguien lo comprobo.
	VigenciaHeredada = "heredada"
	// VigenciaPropia: este punto tiene su propia fecha, distinta de la del
	// paquete, porque la norma la difiere o porque el instrumento es otro.
	VigenciaPropia = "propia"
)

var (
	// ErrSinOrigenDeVigencia es la herencia silenciosa: una obligacion con
	// reloj que no dice si su fecha es la del paquete a proposito o por copia.
	ErrSinOrigenDeVigencia = errors.New("obligacion con reloj sin origen de vigencia")
	// ErrOrigenDeVigenciaDesconocido: vocabulario cerrado de dos valores.
	ErrOrigenDeVigenciaDesconocido = errors.New("origen de vigencia fuera del vocabulario")
	// ErrHeredadaQueNoCoincide: dice heredar y trae otra fecha.
	ErrHeredadaQueNoCoincide = errors.New("vigencia heredada que no es la del paquete")
	// ErrPropiaQueCoincide: dice ser propia y es exactamente la del paquete.
	ErrPropiaQueCoincide = errors.New("vigencia propia identica a la del paquete")
)

// LA HERENCIA SILENCIOSA DEJA DE SER POSIBLE.
//
// EL AGUJERO, MEDIDO ANTES DE ESCRIBIR ESTO (02-09-2026): de las 120
// obligaciones con reloj del corpus, **94 llevaban exactamente la fecha de su
// paquete**. Ninguna de las 94 estaba mal. Y ninguna de las 94 lo decia.
//
// Ese es el problema entero: heredar era el valor por defecto, la copia y la
// comprobacion producen el MISMO JSON, y un defecto que se acierta noventa y
// cuatro veces es un defecto que nadie revisa la vez noventa y cinco. Las dos
// fechas equivocadas de ese mismo dia (art. 111.4 del AI Act y el paquete `eni`
// entero) salieron por ahi: se copio una fecha sin preguntarse cual de las tres
// era.
//
// **Lo que este linter obliga a hacer no es acertar: es DECIR.** Una obligacion
// con reloj declara `origen`, y las dos respuestas son afirmaciones distintas
// que alguien firma:
//
//	heredada  «esta norma tiene una sola fecha aplicable a este punto y la
//	          comprobe»
//	propia    «este punto tiene fecha propia, y aqui esta»
//
// Y se comprueba la COHERENCIA de lo declarado con lo escrito, que es lo unico
// mecanico que se puede exigir: heredada tiene que coincidir con la del paquete
// y propia tiene que diferir. Lo que el linter NO puede comprobar es que la
// comprobacion se hiciera; para eso esta la linea de verificacion del commit
// (invariante 10) y el contraste contra la instantanea, que vive en
// vigencias_test.go de la raiz.
//
// EL VALOR CERO ES ERROR, no "heredada" (invariante 8). El lado permisivo aqui
// es justamente el que sale solo: si el vacio significara heredar, este linter
// no cambiaria nada, porque las 94 seguirian siendo silenciosas.
func (p *Paquete) validarOrigenDeVigencia(anotar func(error)) {
	for _, o := range p.Obligaciones {
		if o.Temporalidad == nil {
			continue // sin reloj, la fecha no mueve ningun vencimiento
		}
		origen := strings.TrimSpace(o.Vigencia.Origen)
		if origen == "" {
			anotar(fmt.Errorf("%w: %s. Su fecha de vigencia (%q) no dice si es la del paquete "+
				"(%q) porque la norma tiene UNA SOLA fecha aplicable a este punto, o si es "+
				"propia. Las dos respuestas producen el mismo JSON y por eso hay que escribirla: "+
				"heredar por copia es lo que fabrico los dos errores de fecha del 02-09-2026. "+
				"Arreglo: `\"origen\": \"heredada\"` o `\"origen\": \"propia\"`, y la linea de "+
				"verificacion en el cuerpo del commit (invariante 10)",
				ErrSinOrigenDeVigencia, o.ID, o.Vigencia.Desde, p.Vigencia.Desde))
			continue
		}
		if origen != VigenciaHeredada && origen != VigenciaPropia {
			anotar(fmt.Errorf("%w: %s declara origen %q y solo hay dos: %q y %q",
				ErrOrigenDeVigenciaDesconocido, o.ID, origen, VigenciaHeredada, VigenciaPropia))
			continue
		}
		mismo := o.Vigencia.Desde == p.Vigencia.Desde
		if origen == VigenciaHeredada && !mismo {
			anotar(fmt.Errorf("%w: %s dice heredar la vigencia del paquete y trae %q, cuando el "+
				"paquete dice %q. O la hereda y son la misma, o es propia y lo dice",
				ErrHeredadaQueNoCoincide, o.ID, o.Vigencia.Desde, p.Vigencia.Desde))
		}
		if origen == VigenciaPropia && mismo {
			anotar(fmt.Errorf("%w: %s dice tener vigencia propia y es exactamente la del paquete "+
				"(%q). Puede ser cierto por casualidad, y precisamente por eso no se admite: "+
				"«propia» tiene que poder distinguirse de «copiada», y si la fecha coincide no "+
				"se distingue. Si la norma de verdad hace coincidir las dos, se declara "+
				"`heredada`, que es lo que un lector va a entender",
				ErrPropiaQueCoincide, o.ID, o.Vigencia.Desde))
		}
	}
}
