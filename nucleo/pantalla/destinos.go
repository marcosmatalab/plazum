package pantalla

// EL DESTINO DE CADA RELOJ: la ley de conservacion de extremo a extremo.
//
// POR QUE HACIA FALTA UNA TERCERA LEY, teniendo ya dos que cuadraban. El
// 29-08-2026 se perdieron 46 relojes en la derivacion (`return nil, nil` para
// una periodica sin su hecho de arranque: ni fecha, ni fila, ni numero), y las
// DOS leyes de conservacion que ya existian dieron verde durante todo el
// episodio. No fallaron: es que ninguna de las dos miraba ahi.
//
//	particion por TIEMPO    en vigor + estrenan + ya cesados + empiezan despues
//	                        + ilegibles = instalados. Cuadraba.
//	particion por ALCANCE   alcanzados + no alcanzados = en vigor. Cuadraba.
//
// El hueco vivia EXACTAMENTE ENTRE LAS DOS, en el tramo que ninguna cubria:
// entre «te alcanza» y «sale en pantalla». Dos particiones que se equilibran
// cada una por su lado no componen una ley: componen dos leyes con una junta
// sin vigilar, y la junta es siempre donde se rompe.
//
// LA LECCION, y es lo que hace de esto una familia y no un caso: la rama
// equivalente de `ventana.Plazo` YA LLEVABA EL AVISO ESCRITO desde semanas
// antes, con estas palabras: «una lista vacia se leeria como "nada que hacer",
// que es el peor error posible aqui». Estaba en un comentario, a treinta lineas
// de la rama que lo incumplia, y no sirvio de nada. **Un aviso en un comentario
// no viaja; una ley en un test si.**
//
// QUE DICE ESTA LEY: todo reloj instalado acaba en EXACTAMENTE UN destino, y
// los destinos son o una fila que el usuario ve, o una ausencia CON NOMBRE Y
// MOTIVO. Una cadena, no tramos.
//
// Y POR QUE NO ES ARITMETICA. Sumar contadores no vale aqui: los de arriba
// cuentan HITOS y los de abajo (fechas, vencidas, mas alla) cuentan
// VENCIMIENTOS, que una periodica multiplica. Una suma que mezclara los dos
// cuadraria por casualidad o no cuadraria nunca. Esto es un ETIQUETADO: cada
// obligacion con reloj recibe una etiqueta y solo una, y el test comprueba
// ademas que la etiqueta que promete una fila TIENE esa fila. Eso es lo que
// cierra la junta.

import (
	"sort"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// Destino es donde acaba un reloj instalado. Vocabulario cerrado.
type Destino string

// Los destinos VISIBLES: el reloj produce al menos una fila que el usuario ve.
const (
	// DestinoConFecha: al menos un vencimiento dentro de la ventana.
	DestinoConFecha Destino = "con fecha en la ventana"
	// DestinoVencida: su vencimiento mas reciente ya paso.
	DestinoVencida Destino = "vencida"
	// DestinoSinFecha: obliga y no hay fecha, con su motivo (falta un hecho,
	// la norma no da plazo, o no hay ejecutor para su primitiva).
	DestinoSinFecha Destino = "sin fecha, con motivo"
	// DestinoEstrena: todavia no obliga y empezara dentro de la ventana.
	DestinoEstrena Destino = "estrena dentro de la ventana"
)

// Los destinos de AUSENCIA: el reloj no produce fila, y se dice por que.
const (
	// DestinoMasAllaDeLaVentana: todos sus vencimientos caen despues.
	DestinoMasAllaDeLaVentana Destino = "todos sus vencimientos, mas alla de la ventana"
	// DestinoNoTeAlcanza: en vigor, y la aplicabilidad dice que no es tuyo.
	DestinoNoTeAlcanza Destino = "en vigor y no te alcanza"
	// DestinoYaCeso: dejo de obligar antes de la ventana.
	DestinoYaCeso Destino = "dejo de obligar antes de la ventana"
	// DestinoEmpiezaDespues: empezara a obligar despues de la ventana.
	DestinoEmpiezaDespues Destino = "empieza a obligar despues de la ventana"
	// DestinoVigenciaIlegible: su vigencia no se puede leer.
	DestinoVigenciaIlegible Destino = "vigencia ilegible"
)

// DestinosConocidos es el vocabulario cerrado.
//
// Cerrado a proposito: un destino nuevo que nadie anada aqui rompe la ley, que
// es justo lo que se quiere el dia que alguien meta una rama nueva en la
// derivacion y se olvide de etiquetarla.
var DestinosConocidos = map[Destino]bool{
	DestinoConFecha: true, DestinoVencida: true, DestinoSinFecha: true,
	DestinoEstrena: true, DestinoMasAllaDeLaVentana: true, DestinoNoTeAlcanza: true,
	DestinoYaCeso: true, DestinoEmpiezaDespues: true, DestinoVigenciaIlegible: true,
}

// EsVisible dice si el destino promete una fila en pantalla.
func (d Destino) EsVisible() bool {
	switch d {
	case DestinoConFecha, DestinoVencida, DestinoSinFecha, DestinoEstrena:
		return true
	}
	return false
}

// anotarDestino etiqueta una obligacion. La PRIMERA etiqueta manda y las
// siguientes se ignoran a proposito.
//
// POR QUE LA PRIMERA Y NO LA ULTIMA. Un plazo con tres hitos puede dar una
// fecha en uno y quedarse pendiente en otro; los dos son ciertos y el reloj sale
// en pantalla igual. Lo que esta ley vigila no es cual de las dos filas es "la
// buena", es que NO HAYA CERO. Las llamadas van en orden de fuerza (primero lo
// que se ve, luego la ausencia), asi que quedarse con la primera se queda con la
// mas informativa sin tener que ordenar nada despues.
func (c *Calendario) anotarDestino(id string, d Destino) {
	if c.Destinos == nil {
		c.Destinos = map[string]Destino{}
	}
	if _, ya := c.Destinos[id]; !ya {
		c.Destinos[id] = d
	}
}

// RelojesSinDestino devuelve los identificadores de obligacion CON RELOJ que no
// han recibido etiqueta. Con la derivacion sana, la lista esta vacia.
//
// Vive aqui y no dentro del test para que el control negativo pase por este
// mismo codigo: un detector que solo se ha ejecutado contra el caso bueno no ha
// demostrado que sepa decir que no.
func RelojesSinDestino(ps []*corpus.Paquete, c Calendario) []string {
	var sin []string
	for _, p := range ps {
		for _, o := range p.Obligaciones {
			if o.Temporalidad == nil {
				continue
			}
			if _, ok := c.Destinos[o.ID]; !ok {
				sin = append(sin, o.ID)
			}
		}
	}
	sort.Strings(sin)
	return sin
}

// DestinosQuePrometenFilaSinTenerla devuelve las obligaciones cuya etiqueta dice
// que salen en pantalla y que NO aparecen en la lista correspondiente.
//
// ES LA MITAD QUE CIERRA LA JUNTA. Etiquetar es barato: si la etiqueta se
// pusiera sin que la fila existiera, la ley se cumpliria y el reloj seguiria sin
// verse, que es exactamente el fallo que esto persigue. Aqui se comprueba que la
// promesa tiene detras una fila de verdad.
func DestinosQuePrometenFilaSinTenerla(c Calendario) []string {
	en := map[string]map[Destino]bool{}
	marca := func(id string, d Destino) {
		if en[id] == nil {
			en[id] = map[Destino]bool{}
		}
		en[id][d] = true
	}
	for _, m := range c.Meses {
		for _, f := range m.Fechas {
			marca(f.Obligacion, DestinoConFecha)
		}
	}
	for _, v := range c.Vencidas {
		marca(v.Obligacion, DestinoVencida)
	}
	for _, s := range c.SinFecha {
		marca(s.Obligacion, DestinoSinFecha)
	}
	for _, e := range c.Estrenos {
		marca(e.Obligacion, DestinoEstrena)
	}
	var rotas []string
	for id, d := range c.Destinos {
		if !d.EsVisible() {
			continue
		}
		if !en[id][d] {
			rotas = append(rotas, id+" dice "+string(d)+" y no sale en esa lista")
		}
	}
	sort.Strings(rotas)
	return rotas
}
