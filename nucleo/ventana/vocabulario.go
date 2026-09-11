package ventana

import (
	"errors"
	"fmt"
)

// ErrVocabularioDelRegimen: una palabra del regimen que no esta en la tabla.
//
// Es centinela y no un error suelto porque quien llama tiene que poder
// distinguir «este paquete usa una palabra que no conozco» de «ha fallado la
// zona horaria»: lo primero es un dato mal escrito y lo segundo es la maquina.
var ErrVocabularioDelRegimen = errors.New("ventana: palabra desconocida en el regimen")

// EL VOCABULARIO DEL REGIMEN, EN UN SOLO SITIO Y PARA LAS DOS DIRECCIONES.
//
// # Por que existe este fichero, y es un hallazgo de auditoria del 11-09-2026
//
// El mismo cuarteto (computo, cierre, traslado) se traducia de texto a tipo en
// DOS sitios que no se conocian:
//
//	nucleo/corpus/dorados.go   switch cerrado, con default que devuelve error
//	nucleo/expediente          tres `if` y un `switch` SIN default
//
// Y no decian lo mismo. El corpus escribe `fin_de_dia` **196 veces**; el
// verificador tenia `case "fin_dia"`, con una sola pieza de menos, y esa cadena
// no la escribe nada mas en todo el arbol: existia unicamente en ese `case`.
//
// Nadie lo vio porque **nadie habia cruzado el vocabulario de verdad**. El unico
// productor de expedientes del repositorio es una herramienta de demo que monta
// el fichero desde un escenario escrito a mano, y ese escenario escribe `auto` y
// `exacto`: las dos unicas cadenas que el `switch` malo acertaba, una por tener
// `case` y la otra porque el valor cero coincide. Un motor con un solo consumidor
// no esta probado, esta de acuerdo consigo mismo.
//
// # Lo que habria costado, dicho con su cardinal
//
// El dia que el emisor saque los relojes del corpus, 191 obligaciones llevan
// `cierre: "fin_de_dia"`. Con el `switch` sin default, las 191 caian al valor
// cero (`CierreAuto`) **sin error y sin discrepancia**, que es una tercera
// salida que la promesa del modelo de amenaza no contempla: ni «coincide» ni «te
// digo donde no coincide», sino «coincide porque los dos se equivocaron igual».
//
// Y medido, porque el numero honesto no es 191: `CierreAuto` da fin del dia para
// plazos en dias o meses, asi que **coincide por suerte en las 170 que se pueden
// contrastar hoy**. Las que cambiarian el resultado son las de plazo SOLO en
// horas con cierre forzado a fin de dia, y hoy hay **cero**. O sea que el defecto
// es real, esta latente, y el dia que alguien escriba el primer SLA en horas con
// cierre forzado, el verificador le dara otra fecha y dira que todo cuadra.
//
// # La regla que sale, y por que el sitio es este paquete
//
// Un vocabulario que cruza una frontera se traduce en UNA funcion, y esa funcion
// vive donde viven los TIPOS. `Computo`, `Cierre` y `Traslado` son de aqui, asi
// que su escritura y su lectura tambien. Cualquier otro sitio seria una segunda
// lista, y una segunda lista es una lista que se queda vieja.
//
// Las dos direcciones van juntas a proposito: `RegimenDesde` lee y los
// `String()` escriben, contra la MISMA tabla. Un lector y un escritor separados
// vuelven a ser dos listas.
//
// LO VIGILA: TestElVocabularioDelRegimenDaLaVueltaEntera, que recorre los tres
// ejes en las dos direcciones, y TestUnaPalabraQueNoEstaEnElVocabularioEsUnError.

// Las palabras del vocabulario. Se nombran para que nadie las escriba a mano en
// un `case` y se coma una pieza, que es exactamente lo que paso con `fin_de_dia`.
const (
	ComputoNaturales = "naturales"
	ComputoHabiles   = "habiles"

	// CierreAutomatico es la AUSENCIA de eleccion: dias o meses vencen al final
	// del dia, y solo horas vencen en el instante exacto.
	CierreAutomatico = "auto"
	CierreEnExacto   = "exacto"
	CierreEnFinDeDia = "fin_de_dia"

	TrasladoSinTraslado    = "ninguno"
	TrasladoAlSiguienteHab = "siguiente_habil"
)

// RegimenDesde traduce el cuarteto de texto al tipo, o falla.
//
// # LA CADENA VACIA ES LA AUSENCIA, Y ESO NO ES LO MISMO QUE UN VALOR RARO
//
// Un `paquete.json` OMITE el campo cuando no elige (`omitempty`), asi que la
// cadena vacia llega y significa «no he elegido». Un expediente serializado, en
// cambio, escribe la palabra, asi que llega `auto`. Las dos son la misma cosa y
// las dos se aceptan, con el motivo escrito aqui en vez de repartido.
//
// Lo que NO se acepta es una palabra que no este en la tabla. Ese es el
// invariante 8 en su tercera forma —presente y no interpretable— y aqui el lado
// permisivo seria devolver el valor cero, que es justo el que un `switch` sin
// `default` regala: `CierreAuto`, `Naturales` y `TrasladoNinguno`, o sea los tres
// mas indulgentes de cada eje.
func RegimenDesde(computo, cierre, traslado string) (Regimen, error) {
	var reg Regimen
	switch computo {
	case "", ComputoNaturales:
		reg.Comp = Naturales
	case ComputoHabiles:
		reg.Comp = Habiles
	default:
		return Regimen{}, fmt.Errorf("%w: computo %q. Los que hay son %q y %q",
			ErrVocabularioDelRegimen, computo, ComputoNaturales, ComputoHabiles)
	}
	switch cierre {
	case "", CierreAutomatico:
		reg.Cierre = CierreAuto
	case CierreEnExacto:
		reg.Cierre = CierreExacto
	case CierreEnFinDeDia:
		reg.Cierre = CierreFinDia
	default:
		return Regimen{}, fmt.Errorf("%w: cierre %q. Los que hay son %q, %q y %q.\n"+
			"  Cuidado con la pieza de menos: `fin_dia` NO es `fin_de_dia`, y un "+
			"verificador que la escribia asi se comia el cierre de 191 obligaciones sin "+
			"decir nada",
			ErrVocabularioDelRegimen, cierre, CierreAutomatico, CierreEnExacto, CierreEnFinDeDia)
	}
	switch traslado {
	case "", TrasladoSinTraslado:
		reg.Trasl = TrasladoNinguno
	case TrasladoAlSiguienteHab:
		reg.Trasl = TrasladoSiguienteHabil
	default:
		return Regimen{}, fmt.Errorf("%w: traslado %q. Los que hay son %q y %q",
			ErrVocabularioDelRegimen, traslado, TrasladoSinTraslado, TrasladoAlSiguienteHab)
	}
	return reg, nil
}

// String escribe el cierre con la palabra del vocabulario, para que emitir y
// leer usen la misma tabla y no puedan separarse.
func (c Cierre) String() string {
	switch c {
	case CierreExacto:
		return CierreEnExacto
	case CierreFinDia:
		return CierreEnFinDeDia
	}
	return CierreAutomatico
}

// String, lo mismo para el traslado.
func (t Traslado) String() string {
	if t == TrasladoSiguienteHabil {
		return TrasladoAlSiguienteHab
	}
	return TrasladoSinTraslado
}
