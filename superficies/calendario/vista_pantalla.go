package calendario

import (
	"embed"
	"io/fs"
	"sort"

	"github.com/marcosmatalab/plazum/nucleo/pantalla"
)

//go:embed plantillas
var plantillasFS embed.FS

// Plantillas expone el sistema de ficheros embebido, por si alguien quiere
// montar otro motor de render.
func Plantillas() fs.FS { return plantillasFS }

// LA REGLA DEL FICHERO. Los ROTULOS son claves de catalogo y los traduce quien
// pinta; el CONTENIDO del corpus (el titulo de una obligacion, su articulo, su
// cita, la derivacion del motor) viaja TAL CUAL, en el idioma de su paquete.
// Traducir texto transcrito del BOE crea obra derivada y se sale de la
// estratificacion de licencias del corpus.
//
// Las fechas se formatean en Go y llegan como cadena. No es pereza: una fecha
// formateada en la plantilla se formatea distinto en cada plantilla que la
// pinte, y este calendario ya sale por dos medios (pantalla y terminal).

// Vista es lo unico que ve la plantilla.
type Vista struct {
	Idioma   string
	Base     string
	Estatico string
	// Titulo es la CLAVE de catalogo.
	Titulo string
	// Aviso es texto ya resuelto: viene de un error, no del catalogo.
	Aviso string
	// Camino es la vuelta al camino guiado. Vacio, no se pinta.
	Camino EnlaceCamino
	// ICS es la direccion del mismo calendario en iCalendar.
	ICS string

	// SinAlcance es el estado vacio: existe la pantalla y no hay de que
	// derivar. Se pinta con su siguiente paso (puerta D11-b).
	SinAlcance bool

	Organizacion string
	Desde        string
	Hasta        string
	// Supuesto dice que el alcance entero sale de un perfil de arranque.
	Supuesto bool

	Vencidas []VencidaVista
	Meses    []MesVista
	Estrenos []TransicionVista
	Ceses    []TransicionVista
	SinFecha []SinFechaVista
	Cuenta   CuentaVista
	// SinNingunaFecha dice que no hay ni un vencimiento en la ventana. Es un
	// estado distinto de SinAlcance: aqui SI hay respuestas, y lo que dicen es
	// que no vence nada en doce meses.
	SinNingunaFecha bool
}

// EnlaceCamino es la vuelta al camino guiado: direccion y CLAVE del rotulo.
type EnlaceCamino struct {
	URL   string
	Clave string
}

// Hay dice si se pinta. Las dos mitades o ninguna; medio enlace se rechaza al
// construir la superficie.
func (e EnlaceCamino) Hay() bool { return e.URL != "" && e.Clave != "" }

// VencidaVista es una obligacion cuyo plazo ya paso y de cuyo cumplimiento no
// consta nada. NO ES UNA ACUSACION: ver la frase que la acompana en la plantilla.
type VencidaVista struct {
	Desde      string
	Ciclos     int
	Marco      string
	Titulo     string
	Articulo   string
	Hito       string
	Regla      string
	Supuesta   bool
	VariosCicl bool
}

// FechaVista es un vencimiento pintable.
type FechaVista struct {
	Vence        string
	Marco        string
	Titulo       string
	Articulo     string
	Hito         string
	Regla        string
	Aviso        string
	Supuesta     bool
	Divergencias []DivergenciaVista
}

// DivergenciaVista es OTRA lectura del mismo plazo. El motor no elige en
// silencio y esta pantalla no las esconde.
type DivergenciaVista struct {
	Lectura string
	Vence   string
}

// MesVista agrupa las fechas de un mes. Clave es la del catalogo (ui.mes.1).
type MesVista struct {
	Clave  string
	Ano    int
	Fechas []FechaVista
}

// TransicionVista sirve para los estrenos y para los ceses: las dos son el
// mismo dato (una fecha en la que cambia si algo te obliga) y se pintan igual.
type TransicionVista struct {
	Dia      string
	Marco    string
	Titulo   string
	Articulo string
	Hitos    int
	Supuesta bool
}

// SinFechaVista es un reloj que obliga y no ha producido fecha, con su motivo.
// Motivo es una CLAVE de catalogo que declara nucleo/pantalla.
type SinFechaVista struct {
	Marco    string
	Titulo   string
	Articulo string
	Hito     string
	Motivo   string
	Regla    string
}

// CuentaVista es la contabilidad honesta, entera.
//
// SE PINTAN LOS CEROS DE LOS CUATRO PRIMEROS y se ocultan los de los descartes.
// No es capricho: los cuatro primeros son la particion (instalados, en vigor,
// alcanzados, en la ventana) y un cero ahi es informacion; un descarte en cero
// es una linea que no dice nada y empuja la que si dice algo fuera de la vista.
type CuentaVista struct {
	Instalados    int
	EnVigor       int
	Alcanzados    int
	EnLaVentana   int
	MasAlla       int
	Pasados       int
	AntesDeVigor  int
	SinFecha      int
	Estrenan      int
	Cesan         int
	NoAlcanzados  int
	YaCesados     int
	EmpiezanTarde int
	Ilegibles     int
}

// rellenarCon vuelca el calendario derivado en la vista.
//
// Aqui no se decide nada del calendario: se ordena para pintar. El orden de las
// secciones si es decision de producto y es la misma que la salida de terminal:
// lo VENCIDO va delante de todo, porque de todo lo que este calendario puede
// decir es lo unico que ya no admite planificacion.
func (v *Vista) rellenarCon(d Derivado) {
	cal := d.Calendario
	v.Organizacion = d.Organizacion
	v.Supuesto = d.Supuesto
	v.Desde = cal.Desde.Format(formatoDeDia)
	v.Hasta = cal.Hasta.Format(formatoDeDia)
	v.SinNingunaFecha = cal.Total() == 0

	for _, x := range cal.Vencidas {
		v.Vencidas = append(v.Vencidas, VencidaVista{
			Desde: x.Desde.Format(formatoDeDia), Ciclos: x.Ciclos, Marco: x.Marco,
			Titulo: x.Titulo, Articulo: x.Articulo, Hito: x.Hito, Regla: x.Regla,
			Supuesta: x.Supuesta, VariosCicl: x.Ciclos > 1,
		})
	}
	for _, m := range cal.Meses {
		mv := MesVista{Clave: m.Clave, Ano: m.Ano}
		for _, f := range m.Fechas {
			fv := FechaVista{
				Vence: f.Vence.Format(formatoDeInstante), Marco: f.Marco, Titulo: f.Titulo,
				Articulo: f.Articulo, Hito: f.Hito, Regla: f.Regla, Aviso: f.Aviso,
				Supuesta: f.Supuesta,
			}
			for _, dv := range f.Divergencias {
				fv.Divergencias = append(fv.Divergencias, DivergenciaVista{
					Lectura: dv.Lectura, Vence: dv.Vence.Format(formatoDeInstante),
				})
			}
			mv.Fechas = append(mv.Fechas, fv)
		}
		v.Meses = append(v.Meses, mv)
	}
	for _, e := range cal.Estrenos {
		v.Estrenos = append(v.Estrenos, TransicionVista{
			Dia: e.Desde.Format(formatoDeDia), Marco: e.Marco, Titulo: e.Titulo,
			Articulo: e.Articulo, Hitos: e.Hitos, Supuesta: e.Supuesta,
		})
	}
	for _, c := range cal.Ceses {
		v.Ceses = append(v.Ceses, TransicionVista{
			Dia: c.Hasta.Format(formatoDeDia), Marco: c.Marco, Titulo: c.Titulo,
			Articulo: c.Articulo, Hitos: c.Hitos, Supuesta: c.Supuesta,
		})
	}
	for _, s := range cal.SinFecha {
		v.SinFecha = append(v.SinFecha, SinFechaVista{
			Marco: s.Marco, Titulo: s.Titulo, Articulo: s.Articulo, Hito: s.Hito,
			Motivo: s.Motivo, Regla: s.Regla,
		})
	}
	v.Cuenta = CuentaVista{
		Instalados: cal.HitosDelCorpus, EnVigor: cal.HitosEnVigor,
		Alcanzados: cal.HitosAplicables, EnLaVentana: cal.Total(),
		MasAlla: cal.MasAllaDeLaVentana, Pasados: cal.VencimientosPasados,
		AntesDeVigor: cal.VencimientosAntesDeLaVigencia, SinFecha: len(cal.SinFecha),
		Estrenan: cal.HitosQueEstrenan, Cesan: cal.HitosQueCesan,
		NoAlcanzados: cal.HitosNoAlcanzados, YaCesados: cal.HitosYaCesados,
		EmpiezanTarde: cal.HitosQueEmpiezanDespues, Ilegibles: cal.HitosConVigenciaIlegible,
	}
}

// Los dos formatos de esta pantalla. Un vencimiento es un INSTANTE y no un dia
// (un plazo que vence a las 23:59 no es «ese dia»), asi que las fechas llevan
// hora y las transiciones no: el dia en que una norma empieza a obligar es un
// dia entero, y ponerle una hora seria inventarse una precision.
const (
	formatoDeDia      = "2006-01-02"
	formatoDeInstante = "2006-01-02 15:04"
)

// ClavesDeCatalogo son las claves que pide ESTA pantalla.
//
// POR QUE EXISTE, igual que en las demas superficies: Faltantes(idioma) responde
// «¿al ingles le falta lo que el espanol tiene?», y la pregunta que rompe la
// pantalla es otra, «¿cubre el catalogo lo que la interfaz pide?». Un catalogo
// completo en los dos idiomas y sin "calendario.pantalla.vencido.frase" pinta el
// descargo mas importante del producto como una clave, y Faltantes da verde.
//
// LOS NOMBRES DE LOS MESES Y LOS MOTIVOS DE «sin fecha» NO ESTAN AQUI: los
// declara nucleo/pantalla, que es quien los emite, y el inventario del catalogo
// los pide desde alli. Repetirlos seria una segunda copia.
func ClavesDeCatalogo() []string {
	out := append([]string(nil), claves...)
	// Los motivos de los relojes sin fecha se piden por su declarador, que es
	// quien sabe cuantos hay: escribirlos aqui se quedaria corto el dia que
	// aparezca un cuarto motivo.
	out = append(out, pantalla.ClavesDelCalendario()...)
	sort.Strings(out)
	return out
}

var claves = []string{
	// Marco, compartidas con las demas superficies.
	"ui.marca",
	"ui.saltar",
	"ui.pie.no_asesoramiento",

	"calendario.pantalla.titulo",
	"calendario.pantalla.error_render",
	"calendario.pantalla.error_fuente",

	// El estado vacio, con su siguiente paso (puerta D11-b).
	"calendario.pantalla.sin_alcance.titulo",
	"calendario.pantalla.sin_alcance.que_es",
	"calendario.pantalla.sin_alcance.paso",
	"calendario.pantalla.ics_sin_alcance",

	// La cabecera.
	"calendario.pantalla.de_quien",
	"calendario.pantalla.periodo",
	"calendario.pantalla.supuesto",
	"calendario.pantalla.aviso_perfil",
	"calendario.pantalla.ics",
	"calendario.pantalla.sin_ninguna_fecha",

	// LO VENCIDO, con su descargo pegado al dato y no en un pie.
	"calendario.pantalla.vencido.titulo",
	"calendario.pantalla.vencido.frase",
	"calendario.pantalla.vencido.desde",
	"calendario.pantalla.vencido.ciclos",

	// Las fechas.
	"calendario.pantalla.hito",
	"calendario.pantalla.derivacion",
	"calendario.pantalla.otra_lectura",

	// Las dos transiciones de la ventana.
	"calendario.pantalla.estrenos.titulo",
	"calendario.pantalla.estrenos.nota",
	"calendario.pantalla.ceses.titulo",
	"calendario.pantalla.ceses.nota",

	// Lo que obliga y no tiene fecha.
	"calendario.pantalla.sin_fecha.titulo",

	// La cuenta, entera.
	"calendario.pantalla.cuenta.titulo",
	"calendario.pantalla.cuenta.instalados",
	"calendario.pantalla.cuenta.en_vigor",
	"calendario.pantalla.cuenta.alcanzados",
	"calendario.pantalla.cuenta.en_la_ventana",
	"calendario.pantalla.cuenta.mas_alla",
	"calendario.pantalla.cuenta.pasados",
	"calendario.pantalla.cuenta.antes_de_vigor",
	"calendario.pantalla.cuenta.sin_fecha",
	"calendario.pantalla.cuenta.estrenan",
	"calendario.pantalla.cuenta.cesan",
	"calendario.pantalla.cuenta.no_alcanzados",
	"calendario.pantalla.cuenta.ya_cesados",
	"calendario.pantalla.cuenta.empiezan_tarde",
	"calendario.pantalla.cuenta.ilegibles",
}
