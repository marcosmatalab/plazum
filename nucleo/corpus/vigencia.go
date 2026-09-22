package corpus

import (
	"errors"
	"fmt"
	"strings"
	"time"
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

// Vigencia acota cuando existe algo. Hasta vacio significa abierta por arriba,
// que es el caso normal: una norma en vigor no declara cuando la derogaran.
//
// Las dos fechas admiten fecha sola (2026-01-01) o instante RFC3339. La fecha
// sola de Hasta cubre el DIA ENTERO: "vigente hasta el 4 de mayo de 2024" es
// hasta el final de ese dia, no hasta su primer segundo. Ver VigenteEn.
type Vigencia struct {
	Desde string `json:"desde"`
	Hasta string `json:"hasta,omitempty"`

	// Origen dice si esta fecha es la del paquete o es propia de este punto, y
	// es OBLIGATORIO en toda obligacion con reloj. Vocabulario cerrado:
	// "heredada" o "propia".
	//
	// POR QUE UN CAMPO Y NO UNA COMPARACION. Porque la comparacion ya se puede
	// hacer (basta mirar si la fecha coincide con la del paquete) y no dice lo
	// que hace falta saber: si coincide PORQUE alguien comprobo que la norma
	// tiene una sola fecha aplicable a ese punto, o si coincide porque se copio.
	// Las dos producen exactamente el mismo JSON, y la segunda es la que
	// fabrico los dos errores de fecha del 02-09-2026.
	//
	// LA HERENCIA SILENCIOSA ES EL PATRON, NO EL ERROR. Medido ese dia: 94 de
	// las 120 obligaciones con reloj llevaban la fecha del paquete. Ninguna de
	// las 94 estaba mal, y ninguna de las 94 lo decia: heredar era el valor por
	// defecto, y un valor por defecto que se acierta 94 veces es un valor por
	// defecto que nadie revisa la vez 95.
	//
	// El valor cero (cadena vacia) NO se interpreta: es error, como en
	// OrigenDelIntervalo y por el mismo motivo. Aqui el lado permisivo seria
	// "heredada", que es justamente el que sale solo.
	Origen string `json:"origen,omitempty"`

	// Alternativas son lecturas discrepantes de la propia VIGENCIA, con su cita.
	//
	// POR QUE HACIA FALTA. El mecanismo de divergencias existia solo para el
	// COMPUTO (HitoSpec.Alternativas: la norma da dos cifras y no dice cual
	// rige). El 27-08-2026 aparecio la otra mitad del problema, que es la
	// discrepancia sobre DESDE CUANDO obliga algo: el AI Act publicado en el
	// DOUE dice una fecha de aplicacion y un acuerdo politico posterior que
	// todavia no se ha publicado dice otra. Las dos son ciertas como lo que
	// son, y ninguna de las dos se puede callar: la del DOUE es la unica que
	// vincula hoy, y la del acuerdo es la que decide si el cliente arranca un
	// proyecto de doce meses este trimestre o el que viene.
	//
	// LO QUE MANDA ES Desde/Hasta, SIEMPRE. Una alternativa no cambia NUNCA lo
	// que devuelve VigenteEn: se calcula lo declarado, y la divergencia se
	// ensena. El valor cero (nil, sin alternativas) es la lectura publicada a
	// secas, que es la restrictiva y la comprobable: el invariante 8 al derecho.
	// Una alternativa que moviera el calculo convertiria un rumor de prensa en
	// derecho aplicado, que es exactamente lo contrario de este producto.
	Alternativas []LecturaVigencia `json:"alternativas,omitempty"`
}

// LecturaVigencia es una interpretacion discrepante de desde (o hasta) cuando
// obliga algo. Necesita cita SIEMPRE: sin ella no es una lectura, es una
// opinion, y el linter la rechaza.
type LecturaVigencia struct {
	ID    string `json:"id"`
	Desde string `json:"desde,omitempty"`
	Hasta string `json:"hasta,omitempty"`
	Cita  string `json:"cita"`

	// Espera nombra el item de vigilancia del que cuelga esta lectura: el
	// hecho de FUERA que, cuando ocurra, obliga a revisarla.
	//
	// POR QUE EXISTE, y lo pago este proyecto en un dia. El 27-08-2026 este
	// corpus afirmaba que dos fechas del AI Act salidas del omnibus digital
	// "NO VINCULAN porque no estan publicadas en el DOUE". Llevaban publicadas
	// treinta y cuatro dias (Reglamento (UE) 2026/1744, de 8 de julio de 2026).
	// Lo encontro una revision, no una puerta.
	//
	// Una lectura divergente es, por definicion, una apuesta sobre algo que
	// todavia no ha pasado. Sin decir DE QUE cuelga, nadie sabe cuando dejo de
	// ser cierta, y una divergencia que envejece mal es peor que no tenerla:
	// se ensena al cliente al lado de la fecha que vincula.
	//
	// Vacio es legal: hay lecturas que no esperan a nada (dos formas de contar
	// el mismo plazo no dependen de ningun evento). Lo que NO es legal es
	// nombrar un item que no existe, ni que exista un item que no nombre
	// ninguna lectura: las dos direcciones se comprueban en vigilancia_test.go.
	Espera string `json:"espera,omitempty"`
}

// rango es una vigencia ya interpretada, con el fin normalizado al ULTIMO
// instante cubierto.
type rango struct {
	desde     time.Time
	hasta     time.Time
	sinInicio bool // no declara desde: si es de una obligacion, hereda del paquete
	abierta   bool // no declara hasta: sigue en vigor
}

// fechaDeVigencia lee una de las dos formas que escriben los paquetes. Devuelve
// ademas si venia con hora, porque de eso depende donde termina un "hasta".
func fechaDeVigencia(campo, s string) (t time.Time, conHora bool, err error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true, nil
	}
	t, err = time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("%w: %s=%q. Se escribe como fecha "+
			"(2026-01-31) o como instante RFC3339 (2026-01-31T23:59:59Z)",
			ErrVigenciaIlegible, campo, s)
	}
	return t, false, nil
}

// interpretar convierte la vigencia declarada en un rango comparable.
func (v Vigencia) interpretar() (rango, error) {
	var r rango
	if v.Desde == "" {
		r.sinInicio = true
	} else {
		d, _, err := fechaDeVigencia("desde", v.Desde)
		if err != nil {
			return rango{}, err
		}
		r.desde = d
	}
	if v.Hasta == "" {
		r.abierta = true
		return r, nil
	}
	h, conHora, err := fechaDeVigencia("hasta", v.Hasta)
	if err != nil {
		return rango{}, err
	}
	if !conHora {
		// Fecha sola: cubre hasta el ultimo instante de ese dia. Lo contrario
		// deroga la norma a las 00:00 del dia que el BOE dice que sigue viva.
		h = h.AddDate(0, 0, 1).Add(-time.Nanosecond)
	}
	r.hasta = h
	if !r.sinInicio && r.desde.After(r.hasta) {
		return rango{}, fmt.Errorf("%w: desde=%q hasta=%q. Asi no esta vigente en ningun "+
			"instante; casi siempre es un error de tecleo en el fichero del paquete",
			ErrVigenciaInvertida, v.Desde, v.Hasta)
	}
	return r, nil
}

// cubre dice si el rango contiene el instante t. Desde es inclusivo y hasta
// tambien (ya normalizado al ultimo instante cubierto).
func (r rango) cubre(t time.Time) bool {
	if t.Before(r.desde) {
		return false
	}
	return r.abierta || !t.After(r.hasta)
}

// interseccion es el rango en el que se solapan dos vigencias. Es lo que hace
// falta para una obligacion: no puede estar en vigor fuera de la vigencia de su
// norma, aunque su propia vigencia diga otra cosa.
func (r rango) interseccion(o rango) rango {
	x := rango{desde: r.desde, hasta: r.hasta, abierta: r.abierta && o.abierta}
	if o.sinInicio {
		x.sinInicio = r.sinInicio
	} else if r.sinInicio || o.desde.After(r.desde) {
		x.desde, x.sinInicio = o.desde, false
	}
	switch {
	case r.abierta:
		x.hasta = o.hasta
	case o.abierta:
		x.hasta = r.hasta
	case o.hasta.Before(r.hasta):
		x.hasta = o.hasta
	}
	return x
}

// VigenteEn dice si algo con esta vigencia esta en vigor en el instante t.
//
// El instante ENTRA COMO DATO. El nucleo no lee el reloj (invariante 1), y aqui
// eso no es una regla de estilo: la vigencia decide que obligaciones salen en el
// expediente, y un expediente que se recalcula distinto manana no es verificable.
//
// Ante una vigencia que no se puede leer devuelve (false, err) y NUNCA
// (true, err): la respuesta segura a "no se que dice este fichero" es no dar la
// obligacion por vigente sin que alguien lo mire.
func (v Vigencia) VigenteEn(t time.Time) (bool, error) {
	r, err := v.interpretar()
	if err != nil {
		return false, err
	}
	if r.sinInicio {
		return false, fmt.Errorf("%w: hasta=%q. Sin desde no se sabe cuando empezo a "+
			"exigirse; declara vigencia.desde", ErrVigenciaSinDesde, v.Hasta)
	}
	return r.cubre(t), nil
}

// EnVigor dice si la obligacion o esta en vigor en el instante t, dentro de este
// paquete.
//
// Dos decisiones, y las dos se notan en el resultado:
//
//	herencia      una obligacion que no declara vigencia usa la del paquete. Es
//	              lo normal: la mayoria de los articulos nacen con su norma.
//	interseccion  la vigencia de la obligacion se corta con la del paquete. Una
//	              obligacion no puede exigirse antes de que su norma exista ni
//	              despues de que la deroguen, diga lo que diga su propio campo.
//	              Sin esto, un paquete mal escrito (o escrito de mala fe) alarga
//	              una obligacion mas alla de la norma que la sostiene.
func (p *Paquete) EnVigor(o Obligacion, t time.Time) (bool, error) {
	rp, err := p.Vigencia.interpretar()
	if err != nil {
		return false, fmt.Errorf("paquete %s: %w", p.URN, err)
	}
	ro, err := o.Vigencia.interpretar()
	if err != nil {
		return false, fmt.Errorf("paquete %s, obligacion %s: %w", p.URN, o.ID, err)
	}
	x := rp.interseccion(ro)
	if x.sinInicio {
		return false, fmt.Errorf("%w: paquete %s, obligacion %s. Ni la obligacion ni su "+
			"paquete declaran vigencia.desde", ErrVigenciaSinDesde, p.URN, o.ID)
	}
	return x.cubre(t), nil
}

// VigentesEn devuelve las obligaciones en vigor en el instante t, en el orden en
// que las declara el paquete.
//
// Existe porque el campo vigencia llevaba tiempo declarandose y no entrando en
// ningun calculo, o sea que una obligacion derogada seguia saliendo en la
// interfaz y en el expediente. Con normas que se modifican cada pocos anos eso
// no es una funcionalidad que falta, es una respuesta incorrecta.
//
// Quien la use para pintar una pantalla tiene ademas que DECIR que ha pasado:
// una obligacion que desaparece de la lista sin explicacion se lee como un
// fallo del producto, no como una derogacion.
func (p *Paquete) VigentesEn(t time.Time) ([]Obligacion, error) {
	var out []Obligacion
	for _, o := range p.Obligaciones {
		ok, err := p.EnVigor(o, t)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, o)
		}
	}
	return out, nil
}
