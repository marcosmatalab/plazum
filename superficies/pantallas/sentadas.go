package pantallas

// LAS SENTADAS EN LA PANTALLA: el numero que convierte un muro en un plan.
//
// # Que problema resuelve, con su cardinal
//
// Un panel que dice «130 obligaciones te alcanzan» es un muro. No es falso, y
// por eso es peor que si lo fuera: quien lo lee no puede hacer nada con ese
// numero, y la reaccion normal a un muro es dejar de mirarlo. Lo que falta al
// lado no es tranquilizar, es decir CUANTAS VECES HAY QUE SENTARSE: veintiocho
// obligaciones anuales de nueve capitulos distintos no son veintiocho reuniones.
//
// # ES TRASVASE, NO MOTOR NUEVO, y eso es lo que hay que saber de este fichero
//
// El calculo entero vive en `nucleo/pantalla/ciclos.go` desde antes (`Ciclo`,
// `Sentada`, `agruparEnCiclos`, `Calendario.Ciclos`) y lo imprime por terminal
// `cmd/plazum/sentadas.go`. Aqui NO se vuelve a agrupar nada: se lee `cal.Ciclos`
// del MISMO `Derivar12Meses` que ya alimenta las cuatro cifras del panel. Si
// hubiera una segunda agrupacion escrita en la superficie, el dia que
// discreparan una de las dos estaria mintiendo y no habria forma de saber cual.
//
// # LO QUE AGRUPA Y LO QUE NO, DICHO EN LA PANTALLA (D-13)
//
// Esto NO es un plan sobre todo el corpus, y callarlo seria vender una
// composicion que no existe. Agrupa las obligaciones PERIODICAS, que son las que
// tienen una cadencia; las demas (un plazo que corre una vez, una obligacion
// puntual, un deber permanente) no se sientan a un ritmo y no salen aqui.
//
// El cardinal va A LA VISTA y se DERIVA, nunca se escribe: son
// `ObligacionesEnCiclo()` de `len(Destinos)`, o sea las periodicas de todas las
// que tienen reloj. Medido sobre el corpus entero el 11-09-2026, con todo
// aplicable: **130 periodicas de 263 con reloj**, sobre 556 obligaciones en
// total. Los tres numeros son distintos y confundirlos es lo que hace que una
// cifra parezca mas de lo que es: 556 es el corpus, 263 es lo que tiene reloj, y
// solo 130 se repiten.
//
// # EL DIA UNO DE UN CLIENTE, QUE ES CUANDO MAS FALTA HACE
//
// Sin expediente no hay ningun hecho declarado, asi que los relojes que arrancan
// de un hecho del operador NO producen fecha: sobre el corpus entero, las 130
// periodicas estan las 130 esperando un dato. `Sentadas()` vale **cero** ese dia,
// y eso NO se pinta como «no hay nada que hacer»: se pinta el ritmo, que si se
// sabe, y al lado cuantas esperan un dato. Un ciclo existe aunque su primera
// fecha no se pueda calcular todavia, y saber que vas a tener que sentarte cuatro
// veces al ano es util antes de registrar nada.
//
// Es la misma decision que ya tomo el imprimirSentadas del terminal, y se copia
// a proposito en vez de reinventarla.

import (
	"sort"
	"strings"

	"github.com/marcosmatalab/plazum/nucleo/pantalla"
)

// VistaSentadas es la seccion entera.
//
// EL VALOR CERO NO PINTA NADA (`Hay` en false), que es el restrictivo: sin
// ciclos no hay ritmo que contar, y una seccion vacia que dice «0 sentadas»
// afirmaria que no hay nada periodico cuando lo que pasa es que no hay corpus.
type VistaSentadas struct {
	Hay bool
	// Sentadas son las veces que hay que sentarse en la ventana, contando solo
	// lo que YA tiene fecha.
	Sentadas int
	// Periodicas son las obligaciones que se repiten a un ritmo fijo, y
	// ConReloj son todas las que tienen reloj de cualquier clase. La segunda es
	// el denominador honesto: 556 obligaciones del corpus no son 556 relojes.
	Periodicas int
	ConReloj   int
	// Marcos distintos que cubren esas periodicas. Es el numero que hace que
	// esto sea composicion entre marcos y no un resumen.
	Marcos int
	// EsperandoDato son las periodicas que todavia no han producido fecha
	// porque falta el hecho del que arranca su ciclo. Se cuenta y no se
	// esconde: el dia uno son todas.
	EsperandoDato int
	Ciclos        []VistaCiclo
}

// VistaCiclo es un ritmo de trabajo.
type VistaCiclo struct {
	// Cadencia es el codigo ISO-8601 tal cual (P12M). Es un DATO y no se
	// traduce.
	Cadencia string
	// Clave es la clave de catalogo del nombre del ritmo («anual»). VACIA si
	// esta superficie no sabe nombrar esa cadencia, y entonces la plantilla
	// pinta el codigo ISO en bruto.
	//
	// SE DEJA VACIA EN VEZ DE INVENTAR UN NOMBRE, que es la misma decision que
	// tomo el terminal: derivar «cada 7 meses» de un P7M seria correcto y
	// ademas invitaria a creer que el producto entiende esa cadencia mejor de lo
	// que la entiende. Un codigo ISO en bruto es feo y no miente.
	Clave        string
	Obligaciones int
	Marcos       int
	Sentadas     int
	// EsperandoDato, Alineables y Fijas son la contabilidad de dentro del
	// ciclo. Alineables son las que se pueden adelantar para juntarlas con otra
	// sentada; Fijas son las que la norma clava y que NUNCA entran en un consejo
	// de agrupacion.
	EsperandoDato int
	Alineables    int
	Fijas         int
}

// PuedeJuntarse dice si tiene sentido sugerir que se agrupen.
//
// CON UNA SOLA NO HAY NADA QUE ALINEAR y decirlo seria ruido. Es la misma
// condicion que el terminal, escrita aqui como metodo para que la plantilla no
// tenga que llevar la logica dentro de un `if` compuesto.
func (c VistaCiclo) PuedeJuntarse() bool {
	return c.Obligaciones > 1 && c.Alineables > 1
}

// cadenciasConNombre son las que esta superficie sabe nombrar, con su clave.
//
// # Por que una tabla y no una derivacion
//
// Por lo mismo que en el terminal: las que hay son las que el corpus usa, y una
// cadencia nueva sale con su codigo ISO en bruto. Lo que cambia aqui es que el
// NOMBRE no vive en esta tabla: vive en el catalogo, en los dos idiomas. La
// tabla solo dice «de esta cadencia se sabe el nombre», que es una decision de
// interfaz y no una palabra.
//
// SU CONTRATO SE COMPRUEBA EN LAS DOS DIRECCIONES: que toda clave de aqui este
// declarada en ClavesDeCatalogo(), y que toda cadencia que el corpus use de
// verdad tenga la suya. La segunda es la que impide que un ritmo real salga como
// «P6M» en la pantalla de alguien.
var cadenciasConNombre = map[string]string{
	"P1M":  "pantalla.hoy.sentadas.ciclo.mensual",
	"P2M":  "pantalla.hoy.sentadas.ciclo.bimestral",
	"P3M":  "pantalla.hoy.sentadas.ciclo.trimestral",
	"P4M":  "pantalla.hoy.sentadas.ciclo.cuatrimestral",
	"P6M":  "pantalla.hoy.sentadas.ciclo.semestral",
	"P12M": "pantalla.hoy.sentadas.ciclo.anual",
	"P24M": "pantalla.hoy.sentadas.ciclo.bienal",
	"P36M": "pantalla.hoy.sentadas.ciclo.trienal",
	// Las formas en anos del mismo ritmo. El corpus usa las de meses, pero
	// ISO-8601 admite las dos y un paquete puede escribir cualquiera.
	"P1Y": "pantalla.hoy.sentadas.ciclo.anual",
	"P2Y": "pantalla.hoy.sentadas.ciclo.bienal",
	"P3Y": "pantalla.hoy.sentadas.ciclo.trienal",
}

// clavesDeCadencia son las claves distintas de la tabla, ordenadas. Las usa
// ClavesDeCatalogo() para no escribirlas dos veces.
func clavesDeCadencia() []string {
	vistas := map[string]bool{}
	for _, k := range cadenciasConNombre {
		vistas[k] = true
	}
	out := make([]string, 0, len(vistas))
	for k := range vistas {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// sentadasDe traspasa los ciclos del calendario a la vista.
//
// NO AGRUPA NADA. Todo lo que hay aqui son lecturas de `cal`, y esa es la
// propiedad que hace que este fichero no pueda discrepar del terminal ni del
// calendario: no hay una segunda cuenta que pueda separarse.
func sentadasDe(cal pantalla.Calendario) VistaSentadas {
	v := VistaSentadas{
		Hay:        len(cal.Ciclos) > 0,
		Sentadas:   cal.Sentadas(),
		Periodicas: cal.ObligacionesEnCiclo(),
		ConReloj:   len(cal.Destinos),
		Marcos:     cal.MarcosEnCiclo(),
	}
	for _, c := range cal.Ciclos {
		v.EsperandoDato += c.EsperandoDato
		v.Ciclos = append(v.Ciclos, VistaCiclo{
			Cadencia:      c.Cadencia,
			Clave:         cadenciasConNombre[strings.ToUpper(c.Cadencia)],
			Obligaciones:  c.Obligaciones,
			Marcos:        len(c.Marcos),
			Sentadas:      len(c.Sentadas),
			EsperandoDato: c.EsperandoDato,
			Alineables:    c.Alineables,
			Fijas:         c.Fijas,
		})
	}
	return v
}
