// Package ventana implementa la aritmetica temporal de obligaciones normativas.
//
// Reglas semanticas, fijadas aqui porque el derecho no las fija y una
// implementacion ingenua las confunde:
//
//	horas y minutos -> tiempo absoluto transcurrido. 24 h son 24 h reales
//	                   aunque haya cambio de hora por medio.
//	dias naturales  -> dias de calendario a la misma hora de pared, en la
//	                   zona del calendario. Un dia que cruza el cambio de
//	                   hora dura 23 o 25 horas reales, no 24.
//	dias habiles    -> se avanza dia a dia saltando sabados, domingos y
//	                   festivos del calendario aplicable.
//	meses           -> de fecha a fecha, con recorte al ultimo dia del mes
//	                   destino. 31 de enero + 1 mes = 28 o 29 de febrero.
//
// Orden de aplicacion cuando una duracion mezcla campos: meses, dias, horas.
// El orden importa y se fija aqui por escrito.
package ventana

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// Duracion es un subconjunto de ISO 8601 suficiente para plazos normativos.
type Duracion struct {
	Meses int
	Dias  int
	Horas int
	Mins  int
	// Indeterminado marca las obligaciones tipo "sin dilacion indebida":
	// no hay limite legal, solo un objetivo interno que fija la organizacion.
	Indeterminado bool
}

var reISO = regexp.MustCompile(`^P(?:(\d+)M)?(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?)?$`)

// AnosMaximosDeUnaDuracion es el techo de dominio de un plazo, en anos.
//
// # Por que hay techo, y no solo control de desbordamiento
//
// Porque arreglar el desbordamiento no basta: `P3000000000D` cabe de sobra en un
// int64, no desborda nada, y el bucle de dias habiles de `calendario.go`
// descuenta de uno en uno, asi que la peticion que lo pida no vuelve. Un plazo
// tiene que ser calculable, no solo representable.
//
// # De donde sale el numero, medido y no puesto a ojo
//
// Del corpus, el 11-09-2026: de las 314 duraciones ISO que hay en `paquetes/`,
// la mayor en meses es `P120M` (diez anos, una retencion documental), la mayor
// en dias `P90D` y la mayor en horas `PT72H`. Cien anos deja tres ordenes de
// magnitud de holgura sobre el plazo legal mas largo que existe en el corpus,
// asi que este techo no puede saltar sobre trabajo legitimo. Si algun dia hace
// falta mas, se sube aqui y se dice que norma lo pide.
//
// Los cuatro techos por campo se DERIVAN de esta cifra y no se escriben: cuatro
// numeros sueltos son cuatro sitios donde uno se queda viejo.
const AnosMaximosDeUnaDuracion = 100

// Techos por campo, derivados. El ano se toma de 366 dias para que el techo sea
// una cota superior y no haga falta hablar de bisiestos.
const (
	maxMeses = 12 * AnosMaximosDeUnaDuracion
	maxDias  = 366 * AnosMaximosDeUnaDuracion
	maxHoras = maxDias * 24
	maxMins  = maxHoras * 60
)

// ParseDuracion acepta P1M, P14D, PT24H, PT4H, PT72H, P1MT12H.
//
// # UN NUMERO QUE NO SE ENTIENDE ES UN ERROR, NUNCA UN VALOR
//
// Aqui vivia un `v, _ := strconv.Atoi(x)` con el error descartado, y el
// razonamiento que lo sostenia era correcto a medias: la expresion regular solo
// casa `\d+`, asi que no puede haber letras. Lo que falta es que **`Atoi`
// tambien falla por DESBORDAMIENTO, y al fallar devuelve el valor SATURADO**, o
// sea que `P99999999999999999999D` salia sin error con `Dias = MaxInt64`. En
// naturales el vencimiento caia el dia ANTERIOR a la base porque `AddDate` da la
// vuelta; en habiles el bucle no terminaba.
//
// Y entra desde un `paquete.json`: el linter carga el paquete porque esto no
// protesta, asi que un cambio de DATOS colgaba el producto. El invariante 2 dice
// que toda norma vive en su paquete de datos, y un paquete de datos no puede ser
// un vector de ejecucion.
//
// Es la tercera forma de la nada del invariante 8 —presente y no
// interpretable— en la aritmetica del reloj legal, y por eso el valor por
// defecto esta prohibido: se devuelve error o se devuelve el numero, nunca un
// cero ni un maximo disfrazado de dato.
//
// LO VIGILA: TestUnaDuracionQueNoCabeNoSeInterpretaComoOtraCosa, con
// TestElTechoDeUnaDuracionDejaPasarLoQueCabe del otro lado.
func ParseDuracion(s string) (Duracion, error) {
	if s == "indeterminado" {
		return Duracion{Indeterminado: true}, nil
	}
	m := reISO.FindStringSubmatch(s)
	if m == nil {
		return Duracion{}, fmt.Errorf("duracion no reconocida: %q", s)
	}
	// "P" y "PT" casan con el patron pero no son duraciones validas: sin ningun
	// componente no hay plazo. Lo destapo el test de ida y vuelta del parser.
	if m[1] == "" && m[2] == "" && m[3] == "" && m[4] == "" {
		return Duracion{}, fmt.Errorf("duracion sin componentes: %q", s)
	}
	meses, err := campoDeDuracion(s, "meses", m[1], maxMeses)
	if err != nil {
		return Duracion{}, err
	}
	dias, err := campoDeDuracion(s, "dias", m[2], maxDias)
	if err != nil {
		return Duracion{}, err
	}
	horas, err := campoDeDuracion(s, "horas", m[3], maxHoras)
	if err != nil {
		return Duracion{}, err
	}
	mins, err := campoDeDuracion(s, "minutos", m[4], maxMins)
	if err != nil {
		return Duracion{}, err
	}
	return Duracion{Meses: meses, Dias: dias, Horas: horas, Mins: mins}, nil
}

// campoDeDuracion lee un campo numerico de una duracion. Vacio es cero, que es
// la ausencia legitima; cualquier otra cosa que no sea un numero dentro del
// techo es error.
//
// # LAS DOS GUARDAS NO SON INDEPENDIENTES, Y LO DIJO UNA MUTACION
//
// Al mutar esto el 11-09-2026 salio algo que no se esperaba: **quitar el control
// del error de `Atoi` y dejar el techo deja la puerta VERDE**. El motivo es que
// sobre una entrada que ya casa con `\d+`, el unico fallo posible de `Atoi` es
// el de rango, y al fallar por rango devuelve `MaxInt64`, que esta muy por
// encima del techo. O sea que **el techo tapa la rama del `Atoi` entera**.
//
// La rama se queda, y se dice por que en vez de dejarla ahi sin explicacion:
//
//  1. Es el tratamiento correcto de un error, y quitarla obliga a escribir otra
//     vez `v, _ := strconv.Atoi(x)`, que es el olor exacto que trajo el P0.
//  2. Es independiente del techo: el dia que alguien suba
//     `AnosMaximosDeUnaDuracion` a algo cercano a `MaxInt64` —o cambie la
//     expresion regular—, esta rama pasa a ser la unica que queda.
//
// Lo que NO se hace es fingir que las dos guardas se comprueban por separado.
// Las tres mutaciones, con lo que dio cada una: quitar el `Atoi` SOBREVIVE;
// quitar el techo se caza por `P3000000000D`, que cabe en un int64 y cuelga el
// bucle de habiles; quitar las dos se caza por las seis.
func campoDeDuracion(entera, nombre, x string, techo int) (int, error) {
	if x == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(x)
	if err != nil {
		// El error de Atoi NO se descarta y NO se convierte en el valor
		// saturado: aqui es donde entraba MaxInt64 haciendose pasar por un
		// plazo. Hoy el techo de abajo llega antes; ver el godoc.
		return 0, fmt.Errorf("duracion %q: los %s (%q) no son un numero que quepa: %w. "+
			"Arreglo: un plazo normativo se escribe con el numero que dice la norma",
			entera, nombre, recortarParaElError(x), err)
	}
	if v > techo {
		return 0, fmt.Errorf("duracion %q: %d %s pasa del techo de %d, que son los %d anos "+
			"de AnosMaximosDeUnaDuracion. Un plazo tiene que ser CALCULABLE y no solo "+
			"representable: en dias habiles se avanza dia a dia, asi que un numero enorme "+
			"no da un vencimiento lejano, deja de contestar. "+
			"Arreglo: comprueba el plazo contra la norma; si de verdad es mayor, sube "+
			"AnosMaximosDeUnaDuracion diciendo que norma lo pide",
			entera, v, nombre, techo, AnosMaximosDeUnaDuracion)
	}
	return v, nil
}

// recortarParaElError acota lo que se pega en un mensaje: el numero que provoca
// el fallo puede tener cientos de digitos y el error acaba en un log.
func recortarParaElError(x string) string {
	const max = 24
	if len(x) <= max {
		return x
	}
	return x[:max] + "..."
}

func (d Duracion) String() string {
	if d.Indeterminado {
		return "indeterminado"
	}
	s := "P"
	if d.Meses > 0 {
		s += fmt.Sprintf("%dM", d.Meses)
	}
	if d.Dias > 0 {
		s += fmt.Sprintf("%dD", d.Dias)
	}
	if d.Horas > 0 || d.Mins > 0 {
		s += "T"
		if d.Horas > 0 {
			s += fmt.Sprintf("%dH", d.Horas)
		}
		if d.Mins > 0 {
			s += fmt.Sprintf("%dM", d.Mins)
		}
	}
	if s == "P" {
		return "P0D"
	}
	return s
}

func diasEnMes(y int, m time.Month) int {
	return time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// sumarMeses suma meses de fecha a fecha con recorte al ultimo dia del mes
// destino. Se implementa a mano porque time.AddDate normaliza por desbordamiento
// (31 de enero + 1 mes le da 3 de marzo), que es lo contrario de lo que dice
// el articulo 30.4 de la Ley 39/2015.
func sumarMeses(t time.Time, n int, loc *time.Location) time.Time {
	t = t.In(loc)
	y, m, d := t.Date()
	hh, mm, ss := t.Clock()
	total := (y*12 + int(m) - 1) + n
	ny, nm := total/12, time.Month(total%12+1)
	if last := diasEnMes(ny, nm); d > last {
		d = last
	}
	return time.Date(ny, nm, d, hh, mm, ss, 0, loc)
}
