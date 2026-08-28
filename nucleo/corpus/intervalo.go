package corpus

// De quien es el numero de una cadencia.
//
// LA DISTINCION QUE ESTE FICHERO CONVIERTE EN DATO. "Revisar al menos una vez
// al ano" y "revisar a intervalos planificados" son la misma frase para un
// lector distraido y obligaciones OPUESTAS para un inspector: la primera pone
// un techo legal al intervalo y la segunda no pone nada. La diferencia decide
// lo unico que el cliente necesita saber, que es si puede mover el numero y
// hacia donde.
//
// Hasta hoy se distinguia leyendo el campo `articulo` (`anexo, punto N` contra
// `ritual plazum sobre N`), que funciona entre personas que conocen el acuerdo
// y no es un dato: nada impedia escribir un intervalo propuesto con cara de
// intervalo legal, y nada lo habria dicho.
//
// La decision entera, con la tabla de quien puede mover que, en
// docs/decisiones.md D-12.

import (
	"errors"
	"fmt"
	"strings"
)

// Los tres origenes del intervalo de una cadencia. Vocabulario CERRADO: un
// valor fuera de esta lista no carga, porque el switch que decide que puede
// mover el cliente no puede tener rama por defecto (seria elegir por el una de
// las dos respuestas opuestas).
const (
	// IntervaloSueloLegal: la norma fija un MINIMO de frecuencia ("al menos una
	// vez al ano", "como minimo anualmente"). El numero declarado es el
	// intervalo MAXIMO tolerado: el cliente puede apretarlo y no aflojarlo.
	IntervaloSueloLegal = "suelo_legal"
	// IntervaloPropuesto: la norma manda revisar y no da numero ("a intervalos
	// planificados", "periodicamente"). El numero es de plazum, va con su
	// justificacion escrita, y el cliente lo mueve en las dos direcciones.
	IntervaloPropuesto = "propuesto"
	// IntervaloFijado: la norma da el numero exacto, y no como minimo. No se
	// toca ni para apretar.
	IntervaloFijado = "fijado"
)

var origenesDelIntervalo = map[string]bool{
	IntervaloSueloLegal: true, IntervaloPropuesto: true, IntervaloFijado: true,
}

var (
	// ErrSinOrigenDelIntervalo: una periodica que no dice de quien es su numero.
	ErrSinOrigenDelIntervalo = errors.New("cadencia sin origen_del_intervalo")
	// ErrOrigenDelIntervaloDesconocido: un valor fuera del vocabulario cerrado.
	ErrOrigenDelIntervaloDesconocido = errors.New("origen_del_intervalo fuera del vocabulario")
	// ErrOrigenDelIntervaloFueraDeSitio: lo declara algo que no es una cadencia.
	ErrOrigenDelIntervaloFueraDeSitio = errors.New("origen_del_intervalo en una primitiva que no es periodica")
	// ErrIntervaloDeLaNormaSinCita: dice que el numero lo da la norma y no dice
	// que articulo lo da.
	ErrIntervaloDeLaNormaSinCita = errors.New("intervalo de la norma sin cita_del_intervalo")
	// ErrIntervaloPropuestoSinJustificacion: el numero es nuestro y no hay
	// argumento. Un numero sin argumento es un numero inventado.
	ErrIntervaloPropuestoSinJustificacion = errors.New("intervalo propuesto sin justificacion_del_intervalo")
	// ErrIntervaloConLasDosExplicaciones: cita y justificacion a la vez. Una de
	// las dos miente y no se sabe cual.
	ErrIntervaloConLasDosExplicaciones = errors.New("intervalo con cita y justificacion a la vez")
	// ErrIntervaloDeLaNormaEnUnRitual: un ritual de plazum que dice que su
	// numero lo da la norma.
	ErrIntervaloDeLaNormaEnUnRitual = errors.New("ritual de plazum con intervalo de la norma")
	// ErrIntervaloPropuestoFueraDeUnRitual: una obligacion que se queda el
	// numero de plazum sin decirlo en el campo que el usuario lee.
	ErrIntervaloPropuestoFueraDeUnRitual = errors.New("intervalo propuesto en una obligacion que no se declara ritual")
	// ErrIntervaloPropuestoSinCuandoCambiarlo: el numero es nuestro y no dice
	// bajo que supuestos el cliente deberia moverlo.
	ErrIntervaloPropuestoSinCuandoCambiarlo = errors.New("intervalo propuesto sin cuando_cambiarlo")
)

// minimoJustificacionDelIntervalo: un argumento, no una etiqueta.
//
// Sesenta y no cuarenta (el suelo de subconjunto_porque) porque aqui hay que
// decir por que ESE numero y no otro, y eso no cabe en una etiqueta. "Es lo
// razonable" son 16 caracteres y no es un argumento; "es el intervalo que
// espera cualquier entidad de certificacion entre auditorias externas" son 68
// y si lo es.
const minimoJustificacionDelIntervalo = 60

// minimoCitaDelIntervalo: el articulo Y las palabras que dan el numero. Una
// cita que solo diga "art. 31" no deja comprobar nada sin abrir el boletin.
const minimoCitaDelIntervalo = 40

// minimoFuenteDelIntervalo: una fuente citable NOMBRA un documento, y nombrar
// un documento no cabe en menos. "NIST" es una organizacion; "NIST SP 800-92"
// es una fuente.
const minimoFuenteDelIntervalo = 12

// minimoCuandoCambiarlo: dos condiciones con su supuesto no caben en menos.
// Mas alto que los otros suelos porque aqui hay que decir DOS cosas, una por
// direccion, y una sola frase suele cubrir solo la de acortar.
const minimoCuandoCambiarlo = 120

// esRitualDePlazum dice si la obligacion se declara a si misma un ritual.
//
// Se mira el campo `articulo` porque es lo que el usuario LEE junto a la fecha:
// la convencion de D-12 es que ahi ponga "ritual plazum sobre N" cuando el
// numero es nuestro. Ese campo y `origen_del_intervalo` tienen que decir lo
// mismo, y que no puedan discrepar es la mitad del valor de esta puerta.
func esRitualDePlazum(articulo string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(articulo)), "ritual plazum")
}

// validarOrigenDelIntervalo exige que toda cadencia diga de quien es su numero,
// y cruza esa respuesta con lo demas que el paquete afirma.
//
// LAS TRES RESPUESTAS Y LO QUE EXIGE CADA UNA. `suelo_legal` y `fijado` dicen
// que el numero lo da la norma, asi que hay que decir QUE articulo lo da y con
// que palabras, y la obligacion no puede llamarse ritual. `propuesto` dice que
// el numero es de plazum, asi que hay que decir POR QUE ese y no otro, no puede
// haber articulo que citar, y el `articulo` tiene que declararlo ritual para
// que el usuario lo lea al lado de la fecha.
//
// LAS DOS EXPLICACIONES A LA VEZ SON UN ERROR, no una redundancia inofensiva.
// Un intervalo con cita de articulo Y justificacion de plazum no deja saber
// cual de las dos manda, y la lectura optimista (hay articulo, luego es legal)
// es la que hara un auditor.
func (p *Paquete) validarOrigenDelIntervalo(anotar func(error)) {
	for _, o := range p.Obligaciones {
		t := o.Temporalidad
		if t == nil {
			continue
		}
		origen := strings.TrimSpace(t.OrigenDelIntervalo)
		cita := strings.TrimSpace(t.CitaDelIntervalo)
		justif := strings.TrimSpace(t.JustificacionDelIntervalo)

		if t.Primitiva != "periodica" {
			if origen != "" || cita != "" || justif != "" {
				anotar(fmt.Errorf("%w: %s/%s es una %q. El origen del intervalo describe la "+
					"CADENCIA de una periodica; en un plazo o en una puntual no hay intervalo "+
					"del que hablar, y declararlo sugiere una libertad que no existe",
					ErrOrigenDelIntervaloFueraDeSitio, p.URN, o.ID, t.Primitiva))
			}
			continue
		}

		if origen == "" {
			anotar(fmt.Errorf("%w: %s/%s declara una cadencia de %q y no dice de quien es ese "+
				"numero. Sin eso el producto no puede decirle al cliente si lo puede mover y "+
				"hacia donde, que es lo unico que necesita saber. Los tres valores son %q (la "+
				"norma pone un minimo de frecuencia: solo se aprieta), %q (el numero lo pone "+
				"plazum y va con su justificacion) y %q (la norma da el numero exacto y no se "+
				"toca). Ver docs/decisiones.md D-12",
				ErrSinOrigenDelIntervalo, p.URN, o.ID, t.Cadencia,
				IntervaloSueloLegal, IntervaloPropuesto, IntervaloFijado))
			continue
		}
		if !origenesDelIntervalo[origen] {
			anotar(fmt.Errorf("%w: %s/%s declara %q. Los tres son %q, %q y %q",
				ErrOrigenDelIntervaloDesconocido, p.URN, o.ID, origen,
				IntervaloSueloLegal, IntervaloPropuesto, IntervaloFijado))
			continue
		}

		// LAS FUENTES SE MIRAN EN LOS TRES ORIGENES, no solo en `propuesto`.
		// El caso natural es el intervalo propuesto (ahi es donde un apoyo
		// fantasma sostiene un numero nuestro), pero dejar sin mirar las otras
		// dos ramas seria dejar abierto el camino que nadie recorre, que es el
		// que se usa. Invariante 8 con otra cara.
		for i, fu := range t.FuentesDelIntervalo {
			if porque := fuenteVagaPorque(fu); porque != "" {
				anotar(fmt.Errorf("%w: %s/%s, fuentes_del_intervalo[%d] = %q: %s",
					ErrFuenteDelIntervaloVaga, p.URN, o.ID, i, recortar(fu, 70), porque))
			}
		}

		if cita != "" && justif != "" {
			anotar(fmt.Errorf("%w: %s/%s trae cita_del_intervalo Y justificacion_del_intervalo. "+
				"O el numero lo da un articulo o lo pone plazum, y no las dos: con las dos, "+
				"quien lea esto se queda sin saber cual manda, y la lectura que hara un "+
				"auditor es la optimista (hay articulo, luego es legal)",
				ErrIntervaloConLasDosExplicaciones, p.URN, o.ID))
			continue
		}

		switch origen {
		case IntervaloSueloLegal, IntervaloFijado:
			if len(cita) < minimoCitaDelIntervalo {
				anotar(fmt.Errorf("%w: %s/%s dice que su cadencia de %q la da la norma (%s) y "+
					"su cita_del_intervalo tiene %d caracteres utiles (minimo %d). Di QUE "+
					"articulo da el numero y CON QUE PALABRAS: una cita que no se puede "+
					"contrastar sin abrir el boletin no es una cita",
					ErrIntervaloDeLaNormaSinCita, p.URN, o.ID, t.Cadencia, origen,
					len(cita), minimoCitaDelIntervalo))
			}
			if esRitualDePlazum(o.Articulo) {
				anotar(fmt.Errorf("%w: %s/%s se presenta como %q y a la vez dice que su numero "+
					"lo da la norma. Un ritual es, por definicion, un intervalo que pone plazum "+
					"porque la norma no lo da. Uno de los dos campos miente",
					ErrIntervaloDeLaNormaEnUnRitual, p.URN, o.ID, o.Articulo))
			}
		case IntervaloPropuesto:
			if len(justif) < minimoJustificacionDelIntervalo {
				anotar(fmt.Errorf("%w: %s/%s pone una cadencia de %q de su propia cosecha y la "+
					"justifica con %d caracteres utiles (minimo %d). Di por que ESE numero y no "+
					"otro. Un numero sin argumento es un numero inventado",
					ErrIntervaloPropuestoSinJustificacion, p.URN, o.ID, t.Cadencia,
					len(justif), minimoJustificacionDelIntervalo))
			}
			if len(strings.TrimSpace(t.CuandoCambiarlo)) < minimoCuandoCambiarlo {
				anotar(fmt.Errorf("%w: %s/%s pone el numero de su cadencia y no dice bajo que "+
					"supuestos moverlo. Un numero nuestro sin instrucciones de uso es una "+
					"imposicion disfrazada de dato: el cliente no sabe si puede tocarlo y ante "+
					"la duda no lo toca. Escribe en `cuando_cambiarlo` UNA condicion para "+
					"acortarlo y UNA para alargarlo, cada una con el supuesto que la hace "+
					"cierta (minimo %d caracteres, tiene %d)",
					ErrIntervaloPropuestoSinCuandoCambiarlo, p.URN, o.ID,
					minimoCuandoCambiarlo, len(strings.TrimSpace(t.CuandoCambiarlo))))
			}
			if !esRitualDePlazum(o.Articulo) {
				anotar(fmt.Errorf("%w: %s/%s pone el numero de su cadencia y su campo articulo "+
					"dice %q, que el usuario lee como si el intervalo saliera de ahi. Escribelo "+
					"como un ritual de plazum sobre ese punto: el cliente tiene derecho a saber "+
					"cual de las dos fechas le puede discutir un inspector",
					ErrIntervaloPropuestoFueraDeUnRitual, p.URN, o.ID, o.Articulo))
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Las fuentes del intervalo
// ---------------------------------------------------------------------------

// ErrFuenteDelIntervaloVaga: una fuente que no identifica ningun documento.
var ErrFuenteDelIntervaloVaga = errors.New("fuente del intervalo que no identifica nada")

// fuenteVagaPorque devuelve el motivo por el que una fuente no vale, o "" si
// vale.
//
// LA REGLA, y por que es tan corta. El problema real no es que la fuente sea
// mala: es que NO HAYA fuente y el argumento suene igual de bien. Contra eso el
// campo ya hace casi todo su trabajo por existir: la pasada de coherencia deja
// de leer cada frase buscando un eco y pregunta una sola cosa, "¿por que este
// argumento no tiene fuente?".
//
// Lo poco que se puede comprobar a maquina es que lo escrito IDENTIFIQUE algo.
// Un documento citable tiene un numero: el de la norma, el de la publicacion,
// el del articulo o el ano. "NIST" es una organizacion y "una guia del
// fabricante" no es nada; "NIST SP 800-92" y "Reglamento (UE) 2024/2690, anexo,
// punto 2.2.3" son fuentes. Los dos apoyos fantasma que salieron en la pasada
// de cierre de las 34 (una curva de decaimiento de tasa de clic sin origen, una
// guia de fabricante sin decir cual) caen los dos por esta regla.
//
// Lo que esto NO caza, y se dice para que nadie lo confunda con una garantia:
// una referencia inventada con pinta de real. Contra eso no hay linter, hay
// pasada humana; esto solo quita del camino lo que ni siquiera pretende serlo.
func fuenteVagaPorque(fu string) string {
	fu = strings.TrimSpace(fu)
	if fu == "" {
		return "esta vacia. Una entrada vacia en la lista de fuentes es peor que no " +
			"tener lista: cuenta como apoyo y no apoya nada"
	}
	if len(fu) < minimoFuenteDelIntervalo {
		return fmt.Sprintf("tiene %d caracteres (minimo %d) y no llega a nombrar un "+
			"documento", len(fu), minimoFuenteDelIntervalo)
	}
	if !tieneDigito(fu) {
		return "no lleva ni un numero. Un documento citable tiene numero de norma, de " +
			"publicacion, de articulo o al menos ano, y sin ninguno de los cuatro quien " +
			"lea esto no puede ir a buscarlo. \"NIST\" es una organizacion; \"NIST SP " +
			"800-92\" es una fuente"
	}
	return ""
}

func tieneDigito(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			return true
		}
	}
	return false
}

func recortar(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
