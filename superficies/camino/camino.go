// Package camino declara EL CAMINO GUIADO y lo sirve como pantalla.
//
// # Que problema resuelve
//
// Las piezas del producto estaban construidas y sueltas. Quien levantaba
// `plazum serve` veia seis pantallas en un menu, y no habia forma de saber que
// despues de responder la entrevista tocaba el calendario, ni que existia el
// acta, ni que la revision de accesos estaba montada en /uar/. Lo unico que
// enlazaba las piezas era leer el codigo o adivinar la direccion, y las dos son
// la misma cosa: no enterarse.
//
// Aqui vive el ORDEN, escrito una vez y en un solo sitio: alcance, calendario,
// derivacion, acta, revision de accesos y escalado. De ese orden salen dos
// cosas, y las dos leen la MISMA lista, que es lo que impide que se separen:
//
//	la pantalla del camino, que numera los pasos y lleva a cada uno;
//	la tira que las demas superficies pintan arriba, para que desde cualquier
//	sitio se vea donde estas y cual es el paso siguiente.
//
// # Un paso sin salida no existe
//
// Todo paso tiene que llevar a algun sitio: o tiene pantalla (Ruta) o tiene la
// orden exacta que lo hace hoy (Comando). Un paso sin ninguna de las dos es un
// callejon, y la puerta D11-b dice que un estado vacio sin verbo no sale de
// aqui. Se comprueba al construir, no en una revision: Nuevo se niega y dice
// que paso es.
//
// Y por eso los dos pasos que TODAVIA NO SON PANTALLA (calendario y escalado)
// entran igual en el camino, con su orden en la mano. Esconderlos habria dejado
// un camino de cuatro pasos que parece completo; ensenarlos con la orden que
// los hace hoy dice la verdad y ademas deja la deuda a la vista, contada por un
// test.
//
// # Esta pantalla no sabe si has hecho algo
//
// No pinta progreso, y no es una carencia de esta version: plazum no guarda las
// respuestas de la entrevista (viajan en la direccion) y sin expediente no
// consta que nada se haya hecho. Una barra de progreso aqui estaria inventando
// el dato mas caro del producto. Cuando el expediente exista, el progreso sale
// de el o no sale.
package camino

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/marcosmatalab/plazum/adaptadores/plantilla"
	"github.com/marcosmatalab/plazum/puertos"
)

// Errores de construccion, como centinelas: quien los compruebe lo hace con
// errors.Is y no buscando una subcadena que manana se redacta distinto.
var (
	// ErrSinPasos: el camino llego vacio. Las DOS formas de la nada, nil y
	// vacio-presente, son este error (invariante 8): un camino vacio pintaria
	// una pantalla que dice que no hay camino, que es la mentira mas cara que
	// esta pieza puede contar.
	ErrSinPasos = errors.New("camino sin pasos")
	// ErrPasoSinSalida: un paso que ni tiene pantalla ni tiene orden.
	ErrPasoSinSalida = errors.New("paso del camino sin salida")
	// ErrPasoIncompleto: falta el identificador, el rotulo o el verbo.
	ErrPasoIncompleto = errors.New("paso del camino incompleto")
	// ErrRutaInvalida: una ruta que no es una ruta de este sitio.
	ErrRutaInvalida = errors.New("ruta de paso invalida")
	// ErrSinCatalogo: sin catalogo la pantalla saldria con las claves crudas.
	ErrSinCatalogo = errors.New("camino sin catalogo")
)

// BasePorDefecto es el prefijo bajo el que se monta la pantalla del camino, y
// ClaveTitulo es la clave de catalogo de su rotulo.
//
// Se exportan las dos porque quien monta necesita las dos para pintar el enlace
// de vuelta desde las demas superficies, y escribirlas alli seria una segunda
// copia de un dato que solo puede tener una.
const (
	BasePorDefecto = "/camino"
	ClaveTitulo    = "camino.titulo"
)

// RutaDe devuelve la ruta declarada de un paso, y si ese paso es pantalla.
//
// EXISTE PARA QUE NADIE VUELVA A ESCRIBIR "/acta/" A MANO. Quien monta las
// superficies tiene que montarlas donde el camino dice que estan, y el unico
// modo de que eso no se separe es que lea de aqui en vez de acordarse. La
// puerta de extremo a extremo comprueba la otra mitad: que cada ruta declarada
// conteste de verdad en el servidor montado.
func RutaDe(id string) (string, bool) {
	for _, p := range canonico {
		if p.ID == id {
			return p.Ruta, p.EsPantalla()
		}
	}
	return "", false
}

// Paso es un tramo del camino guiado.
//
// Titulo y Verbo son CLAVES de catalogo, nunca texto: el camino se lee en los
// dos idiomas igual que el resto de la interfaz. Ruta y Comando son lo unico
// que se escribe tal cual, porque una direccion y una orden de terminal no se
// traducen.
type Paso struct {
	// ID es el identificador estable. No se ensena: sirve para decir en que
	// paso esta la pantalla que pinta la tira.
	ID string
	// Titulo es la clave del rotulo del paso.
	Titulo string
	// Verbo es la clave de QUE SE HACE aqui, en imperativo. Es lo que
	// convierte una lista de pantallas en un camino.
	Verbo string
	// Ruta es la direccion del paso bajo la raiz del servidor. Vacia
	// significa que este paso todavia no es una pantalla.
	Ruta string
	// Comando es la orden que hace este paso HOY, cuando no hay pantalla.
	// Es lo que impide que un paso sin pantalla sea un callejon.
	Comando string
	// LlevaAlcance dice si este paso trabaja con las respuestas de la
	// entrevista, y por tanto si el enlace tiene que llevarlas puestas.
	//
	// POR QUE EXISTE, y salio de intentar tumbar una propiedad que este
	// paquete daba por buena. Las respuestas de la entrevista NO se guardan:
	// viajan en la direccion de la pagina, y la propia pantalla de Alcance lo
	// dice con esas palabras. Con el enlace pelado, ir al camino guiado desde
	// una entrevista respondida y volver BORRABA las respuestas: el camino que
	// existe para no perder a nadie se comia el trabajo de quien lo usaba.
	//
	// Se declara por paso y no se lleva a todos: al acta y a la revision de
	// accesos no les dice nada el alcance (salen del programa de auditoria y
	// del censo), asi que colgarles la consulta seria arrastrar un dato que
	// nadie lee hasta una pantalla que no lo entiende.
	LlevaAlcance bool
}

// EsPantalla dice si este paso se puede recorrer sin salir de `plazum serve`.
func (p Paso) EsPantalla() bool { return p.Ruta != "" }

// Canonico es el camino guiado de plazum, en su orden.
//
// EL ORDEN NO ES DECORATIVO y no se reordena por gusto: cada paso consume lo
// que produce el anterior. Sin responder la entrevista no hay alcance del que
// derivar fechas; sin fechas no hay nada que llevarse al calendario; sin saber
// que obligacion te alcanza no hay derivacion que abrir; el acta se compone de
// lo que ya consta, y la revision de accesos es una de las cosas de las que se
// compone; y el escalado es lo que hace que un vencimiento llegue a una persona
// en vez de quedarse en una pantalla que nadie abre.
//
// Devuelve una COPIA. La lista es la fuente de la verdad de dos consumidores
// (la pantalla y la tira) y quien la reciba no puede moverla debajo del otro.
func Canonico() []Paso { return append([]Paso(nil), canonico...) }

var canonico = []Paso{
	{
		ID: "alcance", Titulo: "camino.paso.alcance", Verbo: "camino.verbo.alcance",
		Ruta: "/alcance", LlevaAlcance: true,
	},
	{
		ID: "calendario", Titulo: "camino.paso.calendario", Verbo: "camino.verbo.calendario",
		// TODAVIA NO ES PANTALLA. El modelo del calendario existe en
		// nucleo/pantalla y el fichero iCalendar lo escribe
		// superficies/calendario, pero `plazum serve` no lo sirve: hoy se saca
		// por terminal. Se dice con la orden delante en vez de callarlo.
		//
		// LA ORDEN SE PEGA Y FUNCIONA, y por eso lleva un perfil de arranque de
		// verdad y no un hueco tipo --sector=SECTOR: esto sale en un bloque que
		// invita a copiar, y una orden que falla al pegarla es un callejon con
		// luz. La primera version llevaba --empleados=N y ni siquiera parseaba.
		// Hay una puerta que las parsea todas.
		Comando:      "plazum calendario --pais=ES --sector=fabricante-software --empleados=250",
		LlevaAlcance: true,
	},
	{
		ID: "derivacion", Titulo: "camino.paso.derivacion", Verbo: "camino.verbo.derivacion",
		Ruta: "/controles", LlevaAlcance: true,
	},
	{
		ID: "acta", Titulo: "camino.paso.acta", Verbo: "camino.verbo.acta",
		Ruta: "/acta/",
	},
	{
		ID: "uar", Titulo: "camino.paso.uar", Verbo: "camino.verbo.uar",
		Ruta: "/uar/",
	},
	{
		ID: "escalado", Titulo: "camino.paso.escalado", Verbo: "camino.verbo.escalado",
		// TODAVIA NO ES PANTALLA, igual que el calendario. Y la orden se
		// ensena tal cual porque en seco no manda nada: es lo primero que
		// alguien quiere probar y lo ultimo que quiere disparar sin querer.
		Comando: "plazum escalado --alcance alcance.json",
	},
}

// SinPantalla son los identificadores de los pasos que todavia no se recorren
// sin salir de `plazum serve`.
//
// Existe para que la deuda se pueda CONTAR desde fuera y no haya que leerla.
// Un test la compara con la lista canonica en los dos sentidos: el dia que uno
// de estos dos gane su pantalla, la puerta se pone roja y obliga a bajar el
// numero en el mismo commit.
func SinPantalla() []string {
	var out []string
	for _, p := range canonico {
		if !p.EsPantalla() {
			out = append(out, p.ID)
		}
	}
	return out
}

// Validar comprueba que un camino es recorrible.
//
// Se exporta porque es la propiedad que esta pieza promete, y una propiedad que
// solo se comprueba dentro de un constructor no se puede probar desde fuera con
// un camino roto a proposito.
func Validar(pasos []Paso) error {
	// LAS DOS FORMAS DE LA NADA, y las dos son este error. nil sale de
	// olvidarse el campo; vacio-presente, de construir la lista y filtrarla
	// entera. Las dos pintan la misma pantalla mentirosa.
	if len(pasos) == 0 {
		return fmt.Errorf("%w: el camino guiado llego vacio, asi que la pantalla diria que "+
			"plazum no tiene camino. Arreglo: pasar camino.Canonico()", ErrSinPasos)
	}
	vistos := map[string]bool{}
	for i, p := range pasos {
		if p.ID == "" || p.Titulo == "" || p.Verbo == "" {
			return fmt.Errorf("%w: el paso %d no tiene identificador, rotulo o verbo. "+
				"Un paso sin verbo no dice que se hace en el, que es lo unico que lo "+
				"distingue de un enlace suelto", ErrPasoIncompleto, i+1)
		}
		if vistos[p.ID] {
			return fmt.Errorf("%w: el identificador %q sale dos veces, asi que la tira no "+
				"puede decir en que paso estas", ErrPasoIncompleto, p.ID)
		}
		vistos[p.ID] = true
		if p.Ruta == "" && p.Comando == "" {
			return fmt.Errorf("%w: el paso %q no tiene pantalla ni orden que lo haga, asi "+
				"que quien llegue a el no tiene nada que hacer. Arreglo: darle Ruta si ya "+
				"es pantalla, o Comando con la orden exacta mientras no lo sea",
				ErrPasoSinSalida, p.ID)
		}
		if p.Ruta != "" {
			if !strings.HasPrefix(p.Ruta, "/") || strings.HasPrefix(p.Ruta, "//") {
				return fmt.Errorf("%w: la ruta del paso %q es %q. Una ruta del camino es "+
					"de este sitio y empieza por una sola barra: con dos, el navegador la "+
					"lee como otro anfitrion y el camino saca al operador de plazum",
					ErrRutaInvalida, p.ID, p.Ruta)
			}
		}
	}
	return nil
}

// Enlace es un paso LISTO PARA PINTAR en la tira de otra superficie.
//
// Titulo sigue siendo una clave de catalogo: la traduce quien pinta, con el
// idioma de su peticion. URL vacia significa que el paso todavia no es una
// pantalla, y entonces se pinta apagado y sin enlace: un enlace que no lleva a
// ningun sitio es peor que no tenerlo.
type Enlace struct {
	Numero int
	Total  int
	ID     string
	Titulo string
	URL    string
	Actual bool
}

// Tira devuelve el camino entero listo para pintar, marcando el paso actual.
//
// actual es el ID del paso en el que esta quien pinta. Una cadena que no sea
// ninguno de los pasos NO es un error: la tira sale sin marcar ninguno, que es
// lo que corresponde a una pantalla que no esta en el camino (Personas, por
// ejemplo). Marcar el primero "por si acaso" seria decirle al operador que esta
// en un sitio en el que no esta.
func Tira(pasos []Paso, base, actual string) []Enlace {
	base = strings.TrimSuffix(base, "/")
	out := make([]Enlace, 0, len(pasos))
	for i, p := range pasos {
		e := Enlace{Numero: i + 1, Total: len(pasos), ID: p.ID, Titulo: p.Titulo,
			Actual: p.ID == actual}
		if p.EsPantalla() {
			e.URL = base + p.Ruta
		}
		out = append(out, e)
	}
	return out
}

// Opciones construye la superficie.
type Opciones struct {
	// Pasos es el camino. Obligatorio, y sin valor por defecto a proposito:
	// rellenarlo con el canonico cuando llega vacio convertiria un olvido en
	// una pantalla plausible.
	Pasos []Paso
	// Catalogo pone el texto. Obligatorio.
	Catalogo puertos.Catalogo
	// Base es el prefijo bajo el que se monta esta pantalla, sin barra final.
	Base string
	// Raiz es el prefijo del que cuelgan las rutas de los PASOS. Suele ser
	// "" (la raiz del servidor) y no tiene por que coincidir con Base.
	Raiz string
	// Estatico es de donde cuelga el CSS, que sirve otra superficie.
	Estatico string
}

// Superficie es el http.Handler de la pantalla del camino.
type Superficie struct {
	o        Opciones
	mux      *http.ServeMux
	motor    *plantilla.Motor
	patrones []string
}

var _ http.Handler = (*Superficie)(nil)

// Nuevo construye la superficie y registra su ruta.
func Nuevo(o Opciones) (*Superficie, error) {
	if o.Catalogo == nil {
		return nil, fmt.Errorf("%w: sin catalogo la pantalla del camino saldria con las "+
			"claves crudas, y esta es la primera pantalla que alguien abre", ErrSinCatalogo)
	}
	if err := Validar(o.Pasos); err != nil {
		return nil, err
	}
	o.Base = strings.TrimSuffix(o.Base, "/")
	o.Raiz = strings.TrimSuffix(o.Raiz, "/")
	m, err := plantilla.Nuevo(plantillasFS, o.Catalogo, "plantillas/*.html")
	if err != nil {
		return nil, fmt.Errorf("camino: no se pueden cargar las plantillas: %w", err)
	}
	s := &Superficie{o: o, mux: http.NewServeMux(), motor: m}
	// El patron lleva la Base dentro, igual que en las demas superficies: un
	// montaje que se olvida el prefijo no da error, da 404 en todo.
	s.registrar("GET "+s.o.Base+"/{$}", s.ver)
	return s, nil
}

// registrar es el UNICO sitio por el que se registra una ruta, y anota el
// patron para que la puerta de CSRF de serve pueda enumerarlas sin conocer
// este paquete.
func (s *Superficie) registrar(patron string, h http.HandlerFunc) {
	s.patrones = append(s.patrones, patron)
	s.mux.HandleFunc(patron, h)
}

// Patrones son las rutas registradas.
func (s *Superficie) Patrones() []string { return append([]string(nil), s.patrones...) }

func (s *Superficie) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

// Ruta es la direccion de esta pantalla, ya con su prefijo. Quien monte la tira
// en otra superficie la necesita para poder volver aqui.
func (s *Superficie) Ruta() string { return s.o.Base + "/" }

// MaxConsulta acota la parte de consulta que esta pantalla deja pasar a los
// enlaces. Es el mismo limite que la superficie de las pantallas, y por la misma
// razon: una peticion adversaria no puede convertirse en una pagina enorme.
const MaxConsulta = 8192

// ver pinta el camino entero.
func (s *Superficie) ver(w http.ResponseWriter, r *http.Request) {
	idioma := s.motor.Resolver(r.Header.Get("Accept-Language"))
	// LAS RESPUESTAS DE LA ENTREVISTA VIAJAN EN LA DIRECCION y no se guardan
	// en ningun sitio, asi que esta pantalla tiene que pasarlas a los enlaces
	// de los pasos que las usan. Sin esto, ir al camino desde una entrevista
	// respondida y volver borra el trabajo: el sitio que existe para no perder
	// a nadie seria el que te pierde.
	//
	// UNA CONSULTA DESMESURADA NO SE RECORTA EN SILENCIO NI SE CUELA ENTERA:
	// se contesta 414. Recortarla dejaria media entrevista con cara de
	// entrevista entera, que es peor que no llevarla.
	if len(r.URL.RawQuery) > MaxConsulta {
		http.Error(w, s.o.Catalogo.Traducir(idioma, "camino.error.consulta_larga"),
			http.StatusRequestURITooLong)
		return
	}
	// Se REESCRIBE desde url.Values en vez de pegar la cadena cruda: asi lo que
	// acaba en el enlace esta normalizado y escapado por la biblioteca, y no es
	// lo que el cliente escribio.
	consulta := r.URL.Query().Encode()

	v := Vista{Idioma: idioma, Estatico: s.o.Estatico, Titulo: ClaveTitulo}
	for i, p := range s.o.Pasos {
		pv := PasoVista{
			Numero: i + 1, Total: len(s.o.Pasos), ID: p.ID,
			Titulo: p.Titulo, Verbo: p.Verbo, Comando: p.Comando,
		}
		if p.EsPantalla() {
			pv.URL = s.o.Raiz + p.Ruta
			if p.LlevaAlcance && consulta != "" {
				pv.URL += "?" + consulta
			}
		}
		v.Pasos = append(v.Pasos, pv)
	}
	// El cuerpo se arma entero antes de tocar el ResponseWriter: html/template
	// escribe segun ejecuta, y un fallo a mitad dejaria media pagina con un 200.
	var b strings.Builder
	if err := s.motor.Render(&b, "pagina", v, idioma); err != nil {
		http.Error(w, s.o.Catalogo.Traducir(idioma, "camino.error.render"),
			http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}
