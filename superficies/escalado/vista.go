package escalado

import (
	"embed"
	"io/fs"
	"sort"
	"time"

	nescalado "github.com/marcosmatalab/plazum/nucleo/escalado"
	"github.com/marcosmatalab/plazum/superficies/camino"
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
	// Inicio es la raiz del SITIO, para el enlace de la marca del armazon.
	Inicio string
	// Tira es la barra lateral con el camino entero, marcando este paso.
	Tira []camino.PasoTira

	// Los tres estados que no son «aqui esta el plan».
	SinSesion  bool
	SinAlcance bool
	// CuboNoExiste dice que se ha pedido abrir una cifra que esta pagina no
	// pinta. Es un estado propio y NO una derivacion vacia: una lista vacia se
	// leeria como «ese cubo vale cero», que es una afirmacion sobre el plan
	// cuando lo unico que pasa es que la peticion no nombra ninguna cifra.
	CuboNoExiste bool

	Organizacion string
	// ComoMandar es la orden que si manda. Texto tal cual.
	ComoMandar string

	Trabajos []TrabajoVista
	Cuenta   []CuboVista
	// Partes son los sumandos de Planificados, escritos al lado del total. Es
	// la segunda forma de abrir una cifra (la primera es el enlace): quien lee
	// «12 se compone de 5 + 4 + 3» no tiene que creerse el 12, tiene que sumar
	// tres numeros que ya estan en la pagina y que se abren por su cuenta.
	Partes []ParteDelPlan
	// Derivacion es UNA cifra abierta. Solo va relleno en la ruta del cubo.
	Derivacion *DerivacionCubo
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
	// cerrado. Se conserva porque es el RESPALDO: si un estado nuevo llega sin
	// clave, la pantalla dice la palabra del nucleo en vez de un hueco.
	Estado string
	// EstadoClave es la clave de catalogo del mismo estado. Vacia significa que
	// nadie lo ha emparejado, y entonces se pinta Estado: el valor cero es el
	// respaldo honesto, no el silencio.
	EstadoClave string
	Motivo      string
	// Saldria dice que este escalon lleva aviso, o sea que se mandaria si
	// alguien pidiera mandar. Ninguna peticion a esta pantalla lo hace.
	//
	// AQUI NO SE DICE POR QUE CANAL NI A QUE DIRECCION, y no es un olvido: el
	// canal se configura en la orden que manda (--smtp, --teams), no en el
	// servidor, asi que esta pantalla no lo sabe. Pintar un campo de canal
	// vacio en todas las filas seria un campo huerfano con cara de dato.
	Saldria bool
}

// CuboVista es un estado del vocabulario cerrado con su recuento Y SU ENLACE.
//
// EL ENLACE NO ES UN ADORNO: es la puerta D11-c. Hasta el 04-09-2026 esta
// pantalla pintaba `estado: N` a secas, o sea ocho numeros que habia que
// creerse en la unica pieza del producto cuyo efecto sale de la organizacion.
// Ver cubos.go.
type CuboVista struct {
	// Estado es la palabra del nucleo, y es el RESPALDO cuando no hay clave.
	// Es ademas LA IDENTIDAD por la que el enlace casa con su lista.
	Estado string
	// Clave es la clave de catalogo del rotulo. Vacia, se pinta Estado.
	Clave string
	N     int
	// URL abre esta cifra. NUNCA vacia en un cubo que se pinta: es lo que la
	// puerta D11-c exige, y un cubo que se pintara sin ella seria exactamente
	// la cifra huerfana que este campo viene a quitar.
	URL string
}

const formatoDeDia = "2006-01-02"

// EL MAPEO ESTADO -> ROTULO VIVE AQUI, EN LA SUPERFICIE, y nucleo/escalado no
// se entera.
//
// # Por que se traducen ahora, cuando el godoc de abajo decia que no
//
// Decia esto: «son vocabulario cerrado y viajan con sus palabras, las mismas que
// imprime la terminal; traducirlos crearia dos nombres para el mismo cubo en dos
// medios del mismo producto». El argumento se sostiene y la conclusion no,
// porque compara los dos medios en el MISMO idioma y el problema estaba en el
// otro: la pagina en ingles pintaba «suprimido por una ventana de silencio», o
// sea que quien lee la interfaz en ingles no tenia UN nombre para ese cubo, no
// tenia NINGUNO. La coherencia entre pantalla y terminal se conserva donde de
// verdad se compara, que es en castellano, porque el rotulo espanol de cada cubo
// es LETRA POR LETRA la constante del nucleo.
//
// Es el mismo problema que el acta resolvio con la familia acta.cubo.*, y con el
// mismo argumento: los NUMEROS se entienden en cualquier idioma y las PALABRAS
// no, asi que media traduccion deja al lector viendo «sin destinatario: 1» sin
// saber si eso es un fallo suyo. Salio como D11-a #3 en docs/hallazgos-d11.md.
//
// # Y por que el mapa y no un metodo en el nucleo
//
// Porque una clave de catalogo es vocabulario de INTERFAZ y el nucleo no tiene
// interfaz. Un `func (e Estado) Clave()` en nucleo/escalado ataria el motor a
// como se rotula una pantalla, y ese acoplamiento no se deshace despues.
var cubos = map[nescalado.Estado]string{
	nescalado.Pendiente:       "escalado.cubo.pendiente",
	nescalado.SinDestinatario: "escalado.cubo.sin_destinatario",
	nescalado.Colapsado:       "escalado.cubo.colapsado",
	nescalado.EnSilencio:      "escalado.cubo.en_silencio",
	nescalado.Enviado:         "escalado.cubo.enviado",
	nescalado.Entregado:       "escalado.cubo.entregado",
	nescalado.Fallido:         "escalado.cubo.fallido",
	nescalado.Atendido:        "escalado.cubo.atendido",
}

// ClaveDelCubo da la clave de catalogo del rotulo de un estado.
//
// Devuelve DOS valores a proposito. Una version que devolviera solo la cadena
// tendria que elegir entre devolver "" (y la pantalla pintaria un hueco donde
// va el nombre de un cubo) o inventarse una clave que el catalogo no tiene (y
// la pantalla pintaria el identificador en crudo). Las dos son peores que
// decir que no se sabe y dejar que quien pinta use la palabra del nucleo, que
// es cierta aunque este en otro idioma.
func ClaveDelCubo(e nescalado.Estado) (string, bool) {
	c, hay := cubos[e]
	return c, hay
}

// ClavesDeLosCubos son los ocho rotulos, ordenados.
//
// SE DERIVAN DE EstadosPosibles() Y NO DEL MAPA, y esa direccion es la que
// importa: recorrer el mapa daria las claves que hay, que es justo lo que no se
// quiere saber. Recorriendo la particion del nucleo, un estado nuevo sin
// emparejar se cae de esta lista, el inventario del catalogo lo echa de menos y
// la puerta de los cubos lo dice con su nombre.
func ClavesDeLosCubos() []string {
	out := make([]string, 0, len(cubos))
	for _, e := range nescalado.EstadosPosibles() {
		if c, hay := cubos[e]; hay {
			out = append(out, c)
		}
	}
	sort.Strings(out)
	return out
}

// rellenarCon vuelca el plan en la vista.
//
// RECIBE LA BASE porque cada cubo sale de aqui con su enlace puesto: componerlo
// en la plantilla seria una segunda copia de la direccion, y el sintoma de que
// se separan es una cifra que enlaza a un 404.
func (v *Vista) rellenarCon(p Plan, base string) {
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
			// La clave si la hay; si no, se queda la palabra del nucleo.
			pv.EstadoClave, _ = ClaveDelCubo(paso.Estado)
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
	conocido := map[nescalado.Estado]bool{}
	for _, e := range nescalado.EstadosPosibles() {
		conocido[e] = true
		n := p.Cuenta[e]
		suma += n
		if n == 0 {
			continue
		}
		cv := CuboVista{Estado: string(e), N: n, URL: EnlaceDelCubo(base, string(e))}
		cv.Clave, _ = ClaveDelCubo(e)
		v.Cuenta = append(v.Cuenta, cv)
	}
	// Y LO QUE NO ESTA EN LA PARTICION TAMBIEN SE PINTA, detras y ordenado.
	//
	// Recorrer solo EstadosPosibles() dejaba fuera cualquier estado que el mapa
	// del plan trajera y la particion no nombrara: su recuento no salia en
	// ningun cubo y no se sumaba a `suma`, asi que lo unico que quedaba de el
	// era el aviso de descuadre, que dice que los numeros no cuadran y no dice
	// QUE falta. Lo que no sale nadie lo echa de menos, y eso es exactamente lo
	// que esta pantalla existe para no hacer.
	//
	// Sale con la palabra del nucleo por rotulo, porque no tiene otra, y por eso
	// el respaldo de `CuboVista.Clave` no es una rama defensiva muerta: es esta.
	// El orden es alfabetico y no el del mapa: un mapa se recorre distinto en
	// cada peticion y estos cubos bailarian entre dos visitas.
	var sueltos []nescalado.Estado
	for e := range p.Cuenta {
		if !conocido[e] {
			sueltos = append(sueltos, e)
		}
	}
	sort.Slice(sueltos, func(i, j int) bool { return sueltos[i] < sueltos[j] })
	for _, e := range sueltos {
		n := p.Cuenta[e]
		suma += n
		if n == 0 {
			continue
		}
		// TAMBIEN CON ENLACE, y esa es la mitad que faltaba: un estado suelto
		// sin rotulo emparejado seguia siendo una cifra huerfana aunque el
		// respaldo le diera un nombre. La derivacion casa por la palabra del
		// nucleo, que un estado suelto tiene igual que los ocho de la
		// particion.
		v.Cuenta = append(v.Cuenta, CuboVista{
			Estado: string(e), N: n, URL: EnlaceDelCubo(base, string(e))})
	}
	v.Suma = suma
	v.Descuadre = suma != p.Planificados
	// LA PARTICION DEL TOTAL. Se compone de lo YA pintado, no de un recorrido
	// nuevo: ver particionDelPlan.
	v.Partes = particionDelPlan(v.Cuenta)
	v.SinAvisos = p.Planificados == 0
	for _, f := range p.Faltas {
		v.Faltas = append(v.Faltas, f.Frase())
	}
}

// ClavesDeCatalogo son las claves que pide ESTA pantalla.
//
// LOS OCHO ESTADOS SI ESTAN AQUI DESDE EL 04-09-2026, y hasta ese dia no
// estaban: el godoc anterior decia que traducirlos crearia dos nombres para el
// mismo cubo. El porque del cambio, con lo que aquel argumento no miraba, esta
// en el comentario del mapa `cubos`.
//
// SE PIDEN POR ClavesDeLosCubos() Y NO ESCRITOS AQUI: escritos aqui serian una
// segunda copia del mapa, y la copia sin puerta es la que se queda vieja el dia
// que el nucleo estrene un noveno estado.
func ClavesDeCatalogo() []string {
	out := append([]string(nil), claves...)
	// Las del armazon compartido las declara quien lo escribe.
	out = append(out, camino.ClavesDelArmazon()...)
	out = append(out, ClavesDeLosCubos()...)
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
	// LA PARTICION DEL TOTAL, escrita como frase ademas de como suma. Los
	// signos `=` y `+` son lo comprobable y no se traducen, asi que la frase se
	// anade delante y no sustituye a nada. Lleva sus dos formas porque su
	// concordancia es con el sujeto que va delante: «1 aviso planificado SE
	// COMPONE de» y «12 avisos planificados SE COMPONEN de».
	"escalado.pantalla.cuenta.se_compone_de",

	// LA DERIVACION DE UNA CIFRA (puerta D11-c). Ver cubos.go: hasta hoy los
	// cubos se leian y no se abrian, que son dos cosas distintas.
	"escalado.pantalla.cubo.titulo",
	"escalado.pantalla.cubo.ver",
	"escalado.pantalla.cubo.volver",
	"escalado.pantalla.cubo.de_que_sale",
	// Y LOS DOS QUE DICEN QUE NO. El descuadre, cuando la lista no cuenta lo
	// que su cabecera; y la peticion de una cifra que esta pagina no pinta, que
	// NO se contesta con una lista vacia.
	"escalado.pantalla.cubo.descuadre",
	"escalado.pantalla.cubo.no_existe.titulo",
	"escalado.pantalla.cubo.no_existe.que_es",
}
