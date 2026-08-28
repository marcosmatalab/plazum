package main

// LAS SENTADAS: la seccion que convierte un calendario que espanta en uno que
// ordena.
//
// EL PROBLEMA, con su numero. El anexo del Reglamento de Ejecucion (UE)
// 2024/2690 mete 47 obligaciones de golpe. Es el numero de la norma y hay que
// ensenarlo (sumarlo es justamente lo que nadie mas hace), pero un 47 solo en
// pantalla no le dice a un CISO que tiene que hacer: le dice que esta perdido.
//
// EL NUMERO QUE FALTABA AL LADO es cuantas VECES HAY QUE SENTARSE. Veintiocho
// obligaciones anuales de nueve capitulos distintos no son veintiocho
// reuniones. Ese segundo numero se calcula (nucleo/pantalla/ciclos.go) y es la
// primera pieza de composicion entre marcos que este producto computa en vez de
// prometer.
//
// LO QUE ESTA SECCION NO HACE: no esconde nada. Las fechas siguen saliendo una
// a una en el listado por meses, con su articulo y su cita. Esto va DELANTE
// porque es lo que se lee primero, no en lugar de.

import (
	"fmt"
	"io"

	"github.com/marcosmatalab/plazum/nucleo/pantalla"
)

// nombreDelCiclo traduce una cadencia ISO-8601 al castellano de un calendario.
//
// Se hace con una tabla y no con una derivacion generica a proposito: las que
// hay son las que el corpus usa, y una cadencia nueva sale con su codigo ISO en
// bruto, que es feo pero no miente. Inventar "cada 7 meses" a partir de P7M
// seria correcto y ademas invitaria a creer que el producto entiende esa
// cadencia mejor de lo que la entiende.
var nombresDeCiclo = map[string]string{
	"P1M":  "mensual",
	"P2M":  "bimestral",
	"P3M":  "trimestral",
	"P4M":  "cuatrimestral",
	"P6M":  "semestral",
	"P12M": "anual",
	"P24M": "bienal",
	"P36M": "trienal",
	"P1Y":  "anual",
	"P2Y":  "bienal",
	"P3Y":  "trienal",
}

func nombreDelCiclo(cad string) string {
	if n, ok := nombresDeCiclo[cad]; ok {
		return n
	}
	return cad
}

func plural(n int, uno, varios string) string {
	if n == 1 {
		return uno
	}
	return varios
}

// imprimirSentadas escribe la seccion. `detalle` expande cada sentada con lo
// que cubre; sin el, sale el resumen.
func imprimirSentadas(w io.Writer, cal pantalla.Calendario, detalle bool) {
	if len(cal.Ciclos) == 0 {
		return
	}
	sentadas := cal.Sentadas()
	obligaciones := cal.ObligacionesEnCiclo()
	marcos := cal.MarcosEnCiclo()

	fmt.Fprintf(w, "LAS SENTADAS: %d %s de %d %s en %d %s al ano\n\n",
		obligaciones, plural(obligaciones, "obligacion periodica", "obligaciones periodicas"),
		marcos, plural(marcos, "marco", "marcos"),
		sentadas, plural(sentadas, "sentada", "sentadas"))
	fmt.Fprintln(w, "    Las obligaciones no son ceremonias. Lo que se repite al mismo ritmo y")
	fmt.Fprintln(w, "    vence el mismo mes se despacha de una vez, aunque venga de marcos")
	fmt.Fprintln(w, "    distintos. Esto es lo que hay que hacer; el detalle de cada fecha, con")
	fmt.Fprintln(w, "    su articulo y su cita, sigue abajo mes a mes.")
	fmt.Fprintln(w)

	for _, c := range cal.Ciclos {
		fmt.Fprintf(w, "  ciclo %s (%s): %d %s de %d %s\n",
			nombreDelCiclo(c.Cadencia), c.Cadencia,
			c.Obligaciones, plural(c.Obligaciones, "obligacion", "obligaciones"),
			len(c.Marcos), plural(len(c.Marcos), "marco", "marcos"))

		if c.ConFecha > 0 {
			fmt.Fprintf(w, "      %d con fecha, en %d %s\n",
				c.ConFecha, len(c.Sentadas), plural(len(c.Sentadas), "sentada", "sentadas"))
		}
		// LOS QUE ESPERAN UN DATO NO SON UN HUECO, son la mitad del mensaje el
		// dia uno de un cliente: tiene el ciclo y no tiene la fecha, y saber
		// cuantas veces al ano tendra que sentarse es util antes de registrar
		// nada.
		if c.EsperandoDato > 0 {
			fmt.Fprintf(w, "      %d esperando un dato tuyo (la ultima vez que lo hiciste)\n",
				c.EsperandoDato)
		}

		// EL CONSEJO, y solo cuando de verdad hay algo que juntar. Con una sola
		// sentada no hay nada que alinear y decirlo seria ruido.
		if c.Obligaciones > 1 && c.Alineables > 1 {
			fmt.Fprintf(w, "      se pueden juntar: %d de las %d se pueden adelantar",
				c.Alineables, c.Obligaciones)
			if c.Fijas > 0 {
				fmt.Fprintf(w, "; %d no, porque la norma da el numero exacto", c.Fijas)
			}
			fmt.Fprintln(w, ".")
		} else if c.Fijas == c.Obligaciones && c.Obligaciones > 0 {
			fmt.Fprintln(w, "      no se pueden mover: la norma da el numero exacto.")
		}

		for _, s := range c.Sentadas {
			fmt.Fprintf(w, "      %s de %d: %d %s de %d %s\n",
				mesesEnEspanol[s.Clave], s.Ano,
				len(s.Fechas), plural(len(s.Fechas), "fecha", "fechas"),
				len(s.Marcos), plural(len(s.Marcos), "marco", "marcos"))
			if !detalle {
				continue
			}
			for _, f := range s.Fechas {
				marca := ""
				if f.Supuesta {
					marca = "  [supuesto]"
				}
				mueve := "adelantable"
				if !f.PuedeAdelantarse() {
					mueve = "NO se mueve"
				}
				fmt.Fprintf(w, "          %s  %s%s\n", f.Vence.Format("02"), f.Titulo, marca)
				fmt.Fprintf(w, "                %s  art. %s  (%s)\n", f.Marco, f.Articulo, mueve)
			}
		}
		fmt.Fprintln(w)
	}

	// LO QUE NO ENTRA EN NINGUNA SENTADA, dicho y no callado. Un plazo unico no
	// es una ceremonia que se repite, y meterlo aqui le diria al operador que
	// puede adelantarlo para juntarlo con otra cosa. La mayoria de los plazos
	// unicos de este corpus son notificaciones de incidente, asi que el consejo
	// equivocado ahi es caro.
	if cal.FechasSueltas > 0 {
		fmt.Fprintf(w, "  Y %d %s que no entra%s en ninguna sentada: son plazos unicos, no\n",
			cal.FechasSueltas, plural(cal.FechasSueltas, "fecha", "fechas"),
			plural(cal.FechasSueltas, "", "n"))
		fmt.Fprintln(w, "  ceremonias que se repiten. Salen en el listado por meses.")
		fmt.Fprintln(w)
	}
}
