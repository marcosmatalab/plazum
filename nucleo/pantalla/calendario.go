package pantalla

// EL CALENDARIO: los proximos doce meses del corpus instalado.
//
// Es una DERIVACION, no un motor. Todo lo que hay debajo existe desde hace
// semanas (el reloj legal en nucleo/ventana, la aplicabilidad en
// nucleo/aplicabilidad, el corpus en nucleo/corpus): esto es lo que faltaba para
// que un CISO vea sus fechas sin leer JSON.
//
// POR QUE VIVE EN nucleo/pantalla Y NO EN cmd/plazum. Porque `plazum calendario`
// y la pantalla Hoy de `plazum serve` tienen que decir LO MISMO, y la unica
// forma de garantizarlo es que salga del mismo sitio. Una derivacion metida en
// el comando queda fuera del alcance de los dorados byte a byte de este paquete
// y fuera del alcance de la superficie web, y entonces las dos versiones
// empiezan a separarse el primer dia.
//
// LAS TRES PROPIEDADES QUE SOSTIENE, y las tres tienen test:
//
//	determinista  dos ejecuciones con el mismo instante dan el mismo orden. El
//	              orden es total, con desempate por (obligacion, hito), porque
//	              recorrer un mapa de Go no tiene orden y quince fechas del mismo
//	              dia saldrian barajadas.
//	sin reloj     el instante entra como dato (invariante 1). Este fichero no
//	              llama a time.Now() ni una vez.
//	honesto       lo que no produce fecha NO desaparece: sale en SinFecha con el
//	              motivo. Una agenda que solo ensena lo que pudo calcular le hace
//	              creer al que la lee que lo demas no existe.

import (
	"sort"
	"strconv"
	"time"

	"plazum/nucleo/corpus"
	"plazum/nucleo/ventana"
)

// Fecha es un vencimiento listo para pintar, con de donde sale.
type Fecha struct {
	Vence time.Time
	// Marco es el URN del paquete. Es lo unico que hay: el formato de corpus
	// todavia no tiene nombre corto de marco, asi que en pantalla sale
	// "urn:es:rd:2022:311" y no "ENS". Esta anotado en docs/pendientes.md.
	Marco      string
	Obligacion string
	Titulo     string
	Articulo   string
	Cita       string
	Hito       string
	Estado     ventana.EstadoVenc
	// Regla es la derivacion completa del motor: de que hecho arranca, que
	// computo se aplico y que traslado. Es lo que convierte una fecha en una
	// fecha defendible.
	Regla string
	Aviso string
	// Divergencias son las otras lecturas del MISMO plazo, con su cita. El
	// motor no elige en silencio y el calendario tampoco las esconde.
	Divergencias []ventana.Divergencia
	// NoAntesDe es el suelo cuando la fecha final todavia no se sabe.
	NoAntesDe time.Time
	// Supuesta dice que esta obligacion se deriva de un hecho SUPUESTO (por
	// ejemplo el que pone un perfil de arranque) y no de uno declarado por la
	// organizacion. Un calendario que no distingue las dos cosas convierte una
	// conjetura en una obligacion, y eso se paga en la primera reunion.
	Supuesta bool
}

// SinFecha es un reloj que existe y no ha producido fecha, con el motivo. No es
// un hueco: es informacion.
type SinFecha struct {
	Marco      string
	Obligacion string
	Titulo     string
	Articulo   string
	Hito       string
	// Motivo es una clave de catalogo, nunca texto: quien pinta esto decide el
	// idioma. Ver ClavesDelCalendario.
	Motivo string
	Regla  string
}

// Mes agrupa las fechas de un mes natural.
type Mes struct {
	Ano int
	Mes time.Month
	// Clave es la clave de catalogo del nombre del mes (ui.mes.1 a ui.mes.12).
	// El nombre no se cablea aqui porque este paquete no sabe en que idioma se
	// va a pintar, y "September" en una agenda en espanol es un fallo que se ve
	// desde la otra punta de la sala.
	Clave  string
	Fechas []Fecha
}

// Calendario es lo que se pinta, con su propia contabilidad.
type Calendario struct {
	Desde time.Time
	Hasta time.Time
	Meses []Mes
	// SinFecha son los relojes que no dieron fecha, ordenados igual.
	SinFecha []SinFecha

	// La contabilidad honesta. Los cuatro numeros se ensenan juntos porque cada
	// uno solo miente si se lee sin los otros tres.
	RelojesDelCorpus  int // cuantas obligaciones con reloj hay instaladas
	RelojesEnVigor    int // cuantas estaban en vigor en el instante de calculo
	RelojesAplicables int // cuantas derivo la aplicabilidad
	FueraDeLaVentana  int // con fecha, pero fuera de los doce meses
}

// motivos de SinFecha, como claves de catalogo.
const (
	MotivoPendienteDeHecho = "ui.calendario.pendiente"
	MotivoSinPlazoLegal    = "ui.calendario.sin_plazo_legal"
	MotivoSinEjecutor      = "ui.calendario.sin_ejecutor"
)

// ClavesDelCalendario son TODAS las claves de catalogo que el calendario puede
// emitir. El inventario del catalogo las lee de aqui: una clave que se emite y
// no esta traducida sale como el identificador en bruto en la pantalla de un
// cliente.
func ClavesDelCalendario() []string {
	out := []string{MotivoPendienteDeHecho, MotivoSinPlazoLegal, MotivoSinEjecutor}
	for m := 1; m <= 12; m++ {
		out = append(out, "ui.mes."+strconv.Itoa(m))
	}
	return out
}

// Aplicable dice si una obligacion le alcanza al sujeto, y si eso se deriva de
// un hecho supuesto. La firma es una funcion y no el motor de aplicabilidad
// entero por dos motivos: este paquete no tiene por que saber de Datalog, y asi
// el calendario se puede probar sin montar un programa.
//
// EL VALOR CERO ES EL RESTRICTIVO, y es el invariante 8 en una frontera nueva:
// una funcion nil significa que NO SE HA FILTRADO NADA, y eso hay que decirlo en
// voz alta en vez de dejar que se lea como "todo aplica". Por eso Derivar12Meses
// exige la funcion y quien no quiera filtrar pasa TodoAplica, que se llama asi
// para que se lea en el sitio de la llamada.
type Aplicable func(idObligacion string) (aplica bool, supuesta bool)

// TodoAplica no filtra nada. Se escribe en el sitio de la llamada a proposito:
// un calendario sin filtrar le ensena a un banco las obligaciones de un
// fabricante de productos sanitarios, y eso hay que pedirlo, no heredarlo.
func TodoAplica(string) (bool, bool) { return true, false }

// Derivar12Meses es la derivacion entera. `ahora` entra como dato: este paquete
// no llama a time.Now() (invariante 1, vigilado por arquitectura_test.go).
func Derivar12Meses(ps []*corpus.Paquete, aplica Aplicable, hechos ventana.Hechos,
	ahora time.Time) Calendario {

	if aplica == nil {
		// No se adivina. Una funcion nil es un error de quien llama, y
		// tratarla como "todo aplica" seria el valor cero permisivo que el
		// invariante 8 prohibe.
		aplica = func(string) (bool, bool) { return false, false }
	}
	hasta := ahora.AddDate(1, 0, 0)
	cal := Calendario{Desde: ahora, Hasta: hasta}

	var fechas []Fecha
	var sin []SinFecha

	for _, p := range ps {
		for _, o := range p.Obligaciones {
			if o.Temporalidad == nil {
				continue
			}
			cal.RelojesDelCorpus++

			// La vigencia primero: un reloj de una obligacion que todavia no
			// obliga (o que ya no) no es una fecha del calendario de nadie. El
			// error de vigencia se trata como "no en vigor", que es lo
			// restrictivo, y no se traga: sale como SinFecha.
			vigente, err := p.EnVigor(o, ahora)
			if err != nil {
				sin = append(sin, SinFecha{Marco: p.URN, Obligacion: o.ID,
					Titulo: o.TituloLegible(), Articulo: o.Articulo,
					Motivo: MotivoSinEjecutor, Regla: "vigencia ilegible: " + err.Error()})
				continue
			}
			if !vigente {
				continue
			}
			cal.RelojesEnVigor++

			ok, supuesta := aplica(o.ID)
			if !ok {
				continue
			}
			cal.RelojesAplicables++

			vs, err := corpus.VencimientosDe(o, hechos, hasta)
			if err != nil {
				// Una primitiva sin ejecutor es una fila, no un silencio. Hoy
				// solo `puntual`, `periodica` y `plazo` tienen ejecutor.
				sin = append(sin, SinFecha{Marco: p.URN, Obligacion: o.ID,
					Titulo: o.TituloLegible(), Articulo: o.Articulo,
					Motivo: MotivoSinEjecutor, Regla: err.Error()})
				continue
			}
			for _, v := range vs {
				switch v.Estado {
				case ventana.Determinado:
					if v.Vence.Before(ahora) || !v.Vence.Before(hasta) {
						cal.FueraDeLaVentana++
						continue
					}
					fechas = append(fechas, Fecha{
						Vence: v.Vence, Marco: p.URN, Obligacion: o.ID,
						Titulo: o.TituloLegible(), Articulo: o.Articulo, Cita: o.Cita,
						Hito: v.Hito, Estado: v.Estado, Regla: v.Regla, Aviso: v.Aviso,
						Divergencias: v.Divergencias, NoAntesDe: v.NoAntesDe,
						Supuesta: supuesta,
					})
				case ventana.PendienteDeHecho:
					sin = append(sin, SinFecha{Marco: p.URN, Obligacion: o.ID,
						Titulo: o.TituloLegible(), Articulo: o.Articulo, Hito: v.Hito,
						Motivo: MotivoPendienteDeHecho, Regla: v.Regla})
				case ventana.SinPlazoLegal:
					sin = append(sin, SinFecha{Marco: p.URN, Obligacion: o.ID,
						Titulo: o.TituloLegible(), Articulo: o.Articulo, Hito: v.Hito,
						Motivo: MotivoSinPlazoLegal, Regla: v.Regla})
				}
			}
		}
	}

	ordenarFechas(fechas)
	ordenarSinFecha(sin)
	cal.SinFecha = sin
	cal.Meses = agrupar(fechas)
	return cal
}

// ordenarFechas fija un orden TOTAL. El desempate por (obligacion, hito) no es
// celo: quince relojes anuales arrancados el mismo dia vencen en el mismo
// instante, y sin desempate salen en un orden distinto en cada ejecucion porque
// el recorrido de los paquetes viene de un mapa. Un dorado byte a byte lo caza
// una vez de cada tres, que es la peor forma de rojo.
func ordenarFechas(f []Fecha) {
	sort.SliceStable(f, func(i, j int) bool {
		if !f[i].Vence.Equal(f[j].Vence) {
			return f[i].Vence.Before(f[j].Vence)
		}
		if f[i].Obligacion != f[j].Obligacion {
			return f[i].Obligacion < f[j].Obligacion
		}
		return f[i].Hito < f[j].Hito
	})
}

func ordenarSinFecha(s []SinFecha) {
	sort.SliceStable(s, func(i, j int) bool {
		if s[i].Obligacion != s[j].Obligacion {
			return s[i].Obligacion < s[j].Obligacion
		}
		return s[i].Hito < s[j].Hito
	})
}

// agrupar parte las fechas en meses naturales. Se agrupa en la zona en la que
// viene el instante, que es la del regimen que lo calculo: agrupar en UTC un
// vencimiento de las 23:59:59 en Madrid lo manda al mes siguiente uno de cada
// dos meses.
func agrupar(f []Fecha) []Mes {
	var out []Mes
	for _, x := range f {
		a, m, _ := x.Vence.Date()
		if n := len(out); n > 0 && out[n-1].Ano == a && out[n-1].Mes == m {
			out[n-1].Fechas = append(out[n-1].Fechas, x)
			continue
		}
		out = append(out, Mes{Ano: a, Mes: m, Clave: "ui.mes." + strconv.Itoa(int(m)),
			Fechas: []Fecha{x}})
	}
	return out
}

// Total es cuantas fechas hay en la ventana. Se calcula y no se guarda para que
// no pueda quedarse vieja.
func (c Calendario) Total() int {
	n := 0
	for _, m := range c.Meses {
		n += len(m.Fechas)
	}
	return n
}
