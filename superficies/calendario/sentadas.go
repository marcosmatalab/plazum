package calendario

// LAS SENTADAS EN EL CALENDARIO: el ritmo encima del muro (pieza 4).
//
// # Por que la MISMA seccion vive en dos pantallas, y no es duplicar
//
// En Hoy se pinta el RITMO y aqui se pinta ademas CUANTAS VECES, y la diferencia
// no es de gusto: es que en Hoy el numero de sentadas NO SE PUEDE SABER. El
// panel de inicio deriva su calendario con los hechos vacios, asi que ninguna
// periodica tiene fecha y `Sentadas()` vale cero siempre. Este calendario deriva
// del ALCANCE PUBLICADO, que si trae hechos, y por eso aqui el numero existe.
//
// Lo que NO se duplica es el calculo: las dos leen `cal.Ciclos` del mismo
// `nucleo/pantalla`, y hay un cuadre en `cmd/plazum` que lo comprueba contra la
// salida de terminal. Lo que se escribe dos veces son las plantillas y sus
// claves, porque son dos superficies con dos catalogos y dos idiomas cada una, y
// compartirlas obligaria a que una importara a la otra.
//
// # Y VA ENCIMA DEL LISTADO POR MESES
//
// Es el sitio y no un detalle: debajo de una lista de ciento y pico fechas, el
// numero que las ordena llega cuando ya te has rendido. La lista sigue entera
// abajo, con su articulo y su cita; esto va delante porque es lo que se lee
// primero.

import (
	"sort"

	"github.com/marcosmatalab/plazum/nucleo/pantalla"
)

// VistaSentadasCal es la seccion.
//
// SE LLAMA ASI Y NO `VistaSentadas` A PROPOSITO: el censo de procedencia del
// texto contrasta por NOMBRE DE TIPO, asi que dos superficies con un tipo del
// mismo nombre se pagan la deuda la una a la otra en silencio. Ya paso una vez
// con esta misma familia.
//
// EL VALOR CERO NO PINTA NADA (`Hay` en false), que es el restrictivo.
type VistaSentadasCal struct {
	Hay bool
	// Sentadas son las veces que hay que sentarse en la ventana. AQUI SI puede
	// ser mayor que cero, a diferencia de Hoy: este calendario tiene hechos.
	Sentadas int
	// Periodicas de ConReloj: lo que esta seccion agrupa, y sobre cuanto.
	Periodicas int
	ConReloj   int
	Marcos     int
	// EsperandoDato son las periodicas que todavia no tienen fecha porque falta
	// el hecho del que arranca su ciclo. Se cuenta y no se esconde.
	EsperandoDato int
	Ciclos        []CicloVista
}

// CicloVista es un ritmo de trabajo.
type CicloVista struct {
	// Cadencia es el codigo ISO-8601 tal cual. Es un DATO.
	Cadencia string
	// Clave es la clave de catalogo del nombre del ritmo, o VACIA si esta
	// superficie no sabe nombrar esa cadencia; entonces se pinta el codigo en
	// bruto, que es feo y no miente.
	Clave         string
	Obligaciones  int
	Marcos        int
	Sentadas      int
	EsperandoDato int
	Alineables    int
	Fijas         int
}

// PuedeJuntarse dice si tiene sentido sugerir que se agrupen. Con una sola no
// hay nada que alinear y decirlo seria ruido.
func (c CicloVista) PuedeJuntarse() bool {
	return c.Obligaciones > 1 && c.Alineables > 1
}

// cadenciasConNombre son las que esta superficie sabe nombrar.
//
// ES UNA SEGUNDA TABLA Y SE DICE. `superficies/pantallas` tiene la suya, con las
// mismas cadencias y otras claves, porque cada superficie tiene su espacio de
// nombres en el catalogo. Lo que NO se duplica es el calculo; esto es
// vocabulario de interfaz, y la alternativa —un paquete compartido de rotulos—
// obligaria a que las dos superficies importaran a un tercero para decir
// «anual». Queda anotado en docs/pendientes.md por si algun dia son tres.
var cadenciasConNombre = map[string]string{
	"P1M":  "calendario.pantalla.sentadas.ciclo.mensual",
	"P2M":  "calendario.pantalla.sentadas.ciclo.bimestral",
	"P3M":  "calendario.pantalla.sentadas.ciclo.trimestral",
	"P4M":  "calendario.pantalla.sentadas.ciclo.cuatrimestral",
	"P6M":  "calendario.pantalla.sentadas.ciclo.semestral",
	"P12M": "calendario.pantalla.sentadas.ciclo.anual",
	"P24M": "calendario.pantalla.sentadas.ciclo.bienal",
	"P36M": "calendario.pantalla.sentadas.ciclo.trienal",
	"P1Y":  "calendario.pantalla.sentadas.ciclo.anual",
	"P2Y":  "calendario.pantalla.sentadas.ciclo.bienal",
	"P3Y":  "calendario.pantalla.sentadas.ciclo.trienal",
}

// clavesDeCadencia son las claves distintas, ordenadas. Las usa el inventario de
// claves para no escribirlas dos veces.
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

// sentadasDe traspasa los ciclos del calendario a la vista. NO AGRUPA NADA.
func sentadasDe(cal pantalla.Calendario) VistaSentadasCal {
	v := VistaSentadasCal{
		Hay:        len(cal.Ciclos) > 0,
		Sentadas:   cal.Sentadas(),
		Periodicas: cal.ObligacionesEnCiclo(),
		ConReloj:   len(cal.Destinos),
		Marcos:     cal.MarcosEnCiclo(),
	}
	for _, c := range cal.Ciclos {
		v.EsperandoDato += c.EsperandoDato
		v.Ciclos = append(v.Ciclos, CicloVista{
			Cadencia:      c.Cadencia,
			Clave:         cadenciasConNombre[c.Cadencia],
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
