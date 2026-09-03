package escalado

import (
	"embed"
	"io/fs"
	"sort"
	"time"

	nescalado "github.com/marcosmatalab/plazum/nucleo/escalado"
)

//go:embed plantillas
var plantillasFS embed.FS

// Plantillas expone el sistema de ficheros embebido.
func Plantillas() fs.FS { return plantillasFS }

// Plan es el resultado de planificar EN SECO: que avisos saldrian y a quien.
//
// Lo arma quien monta, con nucleo/escalado, y esta superficie solo lo pinta.
type Plan struct {
	// Organizacion es de quien es este plan. Texto, no clave: dato del cliente.
	Organizacion string
	// Trabajos son los vencimientos que tienen escalones declarados. Una
	// obligacion SIN escalones no es un trabajo del lazo: es una fila del
	// calendario, y meterla aqui haria que el recuento contara como «sin
	// destinatario» obligaciones que nunca quisieron avisar a nadie.
	Trabajos []Trabajo
	// Cuenta es la ley de conservacion: todo escalon planificado cae en
	// exactamente un estado. Se pinta entera.
	Cuenta map[nescalado.Estado]int
	// Planificados es el total. Si no cuadra con la suma de Cuenta, hay avisos
	// que no estan en ningun sitio, y eso se dice en la pantalla.
	Planificados int
	// Faltas son las figuras que ningun alcance ha rellenado. NO es un fallo de
	// entrega: es un dato que falta, y se dice ANTES de que haga falta el aviso.
	Faltas []nescalado.Falta
	// ComoMandar es la orden exacta que dispara los avisos de verdad. Va como
	// texto y no como clave: una orden de terminal no se traduce.
	ComoMandar string
}

// Trabajo es un vencimiento con sus escalones resueltos.
type Trabajo struct {
	Obligacion string
	Titulo     string
	Hito       string
	Vence      time.Time
	Pasos      []nescalado.Paso
}

// Vista es lo unico que ve la plantilla.
type Vista struct {
	Idioma   string
	Base     string
	Estatico string
	// Titulo es la CLAVE de catalogo.
	Titulo string
	// Aviso es texto ya resuelto: viene de un error.
	Aviso  string
	Camino EnlaceCamino

	// Los dos estados que no son «aqui esta el plan».
	SinSesion  bool
	SinAlcance bool

	Organizacion string
	// ComoMandar es la orden que si manda. Texto tal cual.
	ComoMandar string

	Trabajos []TrabajoVista
	Cuenta   []CuboVista
	// Planificados y Suma se pintan los dos: si no coinciden hay avisos que no
	// estan en ningun cubo, y quien lea esto tiene que verlo. Es un fallo del
	// producto, no del operador, y la pantalla lo dice asi.
	Planificados int
	Suma         int
	Descuadre    bool
	// SinAvisos dice que el plan sale vacio. Es un estado distinto de
	// SinAlcance: aqui SI hay respuestas, y lo que dicen es que ninguna
	// obligacion que te alcanza declara escalones o ninguna vence en la ventana.
	SinAvisos bool
	Faltas    []string
}

// EnlaceCamino es la vuelta al camino guiado: direccion y CLAVE del rotulo.
type EnlaceCamino struct {
	URL   string
	Clave string
}

// Hay dice si se pinta. Las dos mitades o ninguna.
func (e EnlaceCamino) Hay() bool { return e.URL != "" && e.Clave != "" }

// TrabajoVista es un vencimiento con sus escalones, listo para pintar.
type TrabajoVista struct {
	Obligacion string
	Titulo     string
	Hito       string
	Vence      string
	Pasos      []PasoVista
}

// PasoVista es un escalon resuelto.
//
// Saldria separa las dos mitades: un escalon que va a salir lleva persona,
// fecha y canal; uno que no sale lleva su MOTIVO, que es obligatorio en el
// nucleo. Un cubo sin motivo es una etiqueta, no una explicacion.
type PasoVista struct {
	Nivel   int
	Cuando  string
	Figura  string
	Persona string
	// Estado viaja como texto del nucleo, que es donde vive el vocabulario
	// cerrado. No se traduce: son los mismos ocho nombres que imprime la
	// terminal, y dos nombres distintos para el mismo cubo en dos medios es
	// justo lo que hace que nadie sepa cual es cual.
	Estado string
	Motivo string
	// Saldria dice que este escalon lleva aviso, o sea que se mandaria si
	// alguien pidiera mandar. Ninguna peticion a esta pantalla lo hace.
	//
	// AQUI NO SE DICE POR QUE CANAL NI A QUE DIRECCION, y no es un olvido: el
	// canal se configura en la orden que manda (--smtp, --teams), no en el
	// servidor, asi que esta pantalla no lo sabe. Pintar un campo de canal
	// vacio en todas las filas seria un campo huerfano con cara de dato.
	Saldria bool
}

// CuboVista es un estado del vocabulario cerrado con su recuento.
type CuboVista struct {
	Estado string
	N      int
}

const formatoDeDia = "2006-01-02"

// rellenarCon vuelca el plan en la vista.
func (v *Vista) rellenarCon(p Plan) {
	v.Organizacion = p.Organizacion
	v.ComoMandar = p.ComoMandar
	v.Planificados = p.Planificados
	for _, t := range p.Trabajos {
		tv := TrabajoVista{
			Obligacion: t.Obligacion, Titulo: t.Titulo, Hito: t.Hito,
			Vence: t.Vence.Format(formatoDeDia),
		}
		for _, paso := range t.Pasos {
			pv := PasoVista{
				Nivel: paso.Nivel, Figura: paso.Figura, Persona: paso.Persona,
				Estado: string(paso.Estado), Motivo: paso.Motivo,
			}
			if !paso.Cuando.IsZero() {
				pv.Cuando = paso.Cuando.Format(formatoDeDia)
			}
			if paso.Aviso != nil {
				pv.Saldria = true
			}
			tv.Pasos = append(tv.Pasos, pv)
		}
		v.Trabajos = append(v.Trabajos, tv)
	}
	// LA PARTICION SE RECORRE ENTERA Y EN SU ORDEN, no las claves del mapa: un
	// mapa se recorre distinto en cada peticion, y una pantalla cuyos cubos
	// bailan no se puede comparar entre dos visitas. Los ceros no se pintan.
	suma := 0
	for _, e := range nescalado.EstadosPosibles() {
		n := p.Cuenta[e]
		suma += n
		if n == 0 {
			continue
		}
		v.Cuenta = append(v.Cuenta, CuboVista{Estado: string(e), N: n})
	}
	v.Suma = suma
	v.Descuadre = suma != p.Planificados
	v.SinAvisos = p.Planificados == 0
	for _, f := range p.Faltas {
		v.Faltas = append(v.Faltas, f.Frase())
	}
}

// ClavesDeCatalogo son las claves que pide ESTA pantalla.
//
// LOS OCHO ESTADOS NO ESTAN AQUI: son vocabulario cerrado de nucleo/escalado y
// viajan con sus palabras, las mismas que imprime la terminal. Traducirlos aqui
// crearia dos nombres para el mismo cubo en dos medios del mismo producto, que
// es como se pierde a alguien que compara una captura de pantalla con un log.
func ClavesDeCatalogo() []string {
	out := append([]string(nil), claves...)
	sort.Strings(out)
	return out
}

var claves = []string{
	// Marco, compartidas con las demas superficies.
	"ui.marca",
	"ui.saltar",
	"ui.pie.no_asesoramiento",

	"escalado.pantalla.titulo",
	"escalado.pantalla.error_render",

	// SIN SESION. Aqui hay nombres de personas y el reparto de responsabilidades
	// de cumplimiento de la organizacion.
	"escalado.pantalla.sin_sesion.titulo",
	"escalado.pantalla.sin_sesion.por_que",

	// El estado vacio, con su siguiente paso (puerta D11-b).
	"escalado.pantalla.sin_alcance.titulo",
	"escalado.pantalla.sin_alcance.que_es",
	"escalado.pantalla.sin_alcance.paso",

	// LA PROMESA DE ESTA PANTALLA, y va arriba del todo.
	"escalado.pantalla.en_seco",
	"escalado.pantalla.como_mandar",

	"escalado.pantalla.de_quien",
	"escalado.pantalla.vence",
	"escalado.pantalla.hito",
	"escalado.pantalla.escalon",
	"escalado.pantalla.saldria",
	"escalado.pantalla.no_sale",
	"escalado.pantalla.sin_avisos",

	// Las figuras sin persona: un DATO QUE FALTA, dicho antes de que haga falta.
	"escalado.pantalla.faltas.titulo",
	"escalado.pantalla.faltas.nota",

	// La ley de conservacion, impresa.
	"escalado.pantalla.cuenta.titulo",
	"escalado.pantalla.cuenta.planificados",
	"escalado.pantalla.cuenta.descuadre",
}
