package calendario

// LA PANTALLA DEL CALENDARIO.
//
// # Que faltaba
//
// El calendario era el segundo paso del camino guiado y NO ERA UNA PANTALLA. El
// modelo lo derivaba nucleo/pantalla desde hacia semanas, el fichero iCalendar
// lo escribia este mismo paquete, y la unica forma de verlo era `plazum
// calendario` en una terminal. O sea que quien levantaba `plazum serve`, hacia
// la entrevista y seguia el camino, llegaba al paso dos y se encontraba una
// orden que copiar. El camino lo decia en voz alta (esa es la puerta D11-b) y
// eso no lo convierte en menos hueco: era el hueco mejor documentado del
// producto.
//
// # Esta superficie NO deriva nada
//
// Ni carga corpus, ni lee un alcance, ni llama al motor de aplicabilidad. Recibe
// un `pantalla.Calendario` ya derivado y lo pinta. Es la misma decision que la
// pantalla del acta: si esta pieza pudiera cambiar un numero habria dos
// calendarios, el de la terminal y el del navegador, y el dia que se separaran
// nadie sabria cual manda. Lo que se pinta aqui y lo que imprime `plazum
// calendario` salen de la MISMA derivacion.
//
// # Sin alcance, la pantalla existe igual
//
// Puerta D11-b, la misma que el acta y la revision de accesos: una pantalla que
// desaparece cuando no hay datos deja al operador sin saber que existia. Sin
// alcance se pinta el estado vacio, que cuenta de que se compone un calendario y
// da la orden exacta que produce el fichero que falta. Lo que NO se hace es
// fingir que hay calendario.
//
// # Lo vencido se presenta como DATO QUE FALTA, nunca como culpa
//
// Es la regla de la casa y aqui es donde mas cara sale: un vencimiento pasado
// sin registro de cumplimiento NO es un incumplimiento, es una ausencia de dato,
// y plazum no sabe distinguirlos. La frase va PEGADA a la lista y no en un pie,
// y tiene su control positivo en el test (con un calendario sintetico que
// recorre esa rama, porque un descargo que ninguna entrada alcanza es un
// descargo que no existe).
//
// # Todas las rutas son GET
//
// Un calendario no se edita: se deriva. La segunda ruta sirve el mismo
// calendario en iCalendar, con el escritor que ya vivia en este paquete, para
// llevarselo a Outlook sin volver a la terminal.

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/marcosmatalab/plazum/adaptadores/plantilla"
	"github.com/marcosmatalab/plazum/nucleo/pantalla"
	"github.com/marcosmatalab/plazum/puertos"
	"github.com/marcosmatalab/plazum/superficies/camino"
)

// BasePorDefecto es el prefijo bajo el que se monta esta pantalla, y
// FicheroICS el nombre del fichero que sirve la segunda ruta.
//
// Se exportan porque quien monta necesita el primero para colgarla y el camino
// guiado lo necesita para declarar su paso: escribir "/calendario" en dos sitios
// es tener dos, y el sintoma de que se separan es un 404 en un paso del camino.
const (
	BasePorDefecto = "/calendario"
	FicheroICS     = "plazum.ics"
)

var (
	// ErrSinCatalogo: sin catalogo la pantalla saldria con las claves crudas.
	ErrSinCatalogo = errors.New("calendario: pantalla sin catalogo")
	// ErrCamino: el enlace de vuelta al camino guiado llego a medias.
	ErrCamino = errors.New("calendario: enlace al camino guiado invalido")
)

// Derivado es un calendario ya calculado, con de quien es.
//
// Supuesto viaja aqui y no dentro de cada fila porque es una propiedad del
// ALCANCE entero: o las respuestas son del operador, o son las conjeturas de un
// perfil de arranque. Las filas ya traen su propio `Supuesta` para el caso mixto
// y esta pantalla pinta las dos cosas.
type Derivado struct {
	Calendario pantalla.Calendario
	// Organizacion es de quien es este calendario, tal y como lo declara su
	// alcance. Viaja como texto y no como clave: es un dato del cliente.
	Organizacion string
	// Supuesto dice que el alcance entero sale de un perfil de arranque y no
	// de respuestas del operador. Un calendario que no distingue las dos cosas
	// convierte una conjetura en una obligacion.
	Supuesto bool
}

// Fuente es de donde sale el calendario que se pinta.
//
// Esta superficie no sabe de ficheros, de corpus ni de motores: eso es del
// adaptador que la monta. Es el mismo contrato que `acta.Actas`, y por lo mismo.
type Fuente interface {
	// Actual devuelve el calendario de ahora. Devolver (Derivado{}, false, nil)
	// significa «no hay alcance del que derivarlo», que NO es un error: es el
	// estado vacio, y esta pantalla lo sabe pintar con su siguiente paso.
	//
	// SE PREGUNTA EN CADA PETICION y no se cachea al arrancar: un calendario
	// calculado al levantar el servidor cuenta lo que vencia el dia que se
	// levanto, y un servidor que lleva tres semanas en pie estaria dando las
	// fechas de hace tres semanas. Es la misma razon por la que el acta se
	// recompone en cada peticion.
	Actual() (Derivado, bool, error)
}

// Opciones de la pantalla.
//
// Se llama asi y no `Opciones` porque ese nombre ya es del escritor iCalendar de
// este paquete, que existia antes. Renombrarlo habria sido un cambio de firma en
// una pieza probada para ganar un nombre mas corto.
type OpcionesPantalla struct {
	// Fuente puede ser nil: entonces la pantalla existe y dice como tener un
	// calendario. Es la puerta D11-b.
	Fuente Fuente
	// Catalogo pone el texto. Obligatorio.
	Catalogo puertos.Catalogo
	// Base es el prefijo bajo el que se monta, sin barra final.
	Base string
	// Estatico es de donde cuelga el CSS, que sirve otra superficie.
	Estatico string
	// CaminoRuta y CaminoClave son la vuelta al camino guiado. El valor cero
	// (las dos vacias) es no pintar nada, que es el restrictivo; MEDIO enlace
	// se rechaza al construir, porque las dos mitades pintan un enlace roto.
	CaminoRuta  string
	CaminoClave string
	// Pasos es EL CAMINO ENTERO, para la barra lateral compartida. Lo pasa
	// quien monta, con camino.Canonico().
	//
	// EL VALOR CERO ES NO PINTAR BARRA, que es el restrictivo (invariante 8):
	// rellenarlo aqui con el canonico cuando llega vacio convertiria un olvido
	// de quien monta en una barra plausible que enlaza a rutas donde nadie ha
	// colgado nada.
	Pasos []camino.Paso
	// Raiz es el prefijo del SITIO del que cuelgan las rutas de los pasos, no
	// el de esta pantalla. Suele ser "" y por eso su valor cero vale. Se
	// distingue de Base a proposito: usar Base para el enlace de la marca
	// mandaria a "/calendario/" en vez de a la portada.
	Raiz string
	// Ahora es el instante del sello del fichero iCalendar. Opcional; sin el,
	// time.Now en UTC. Entra como funcion y no como valor porque el sello es de
	// CADA descarga, no del arranque del servidor.
	Ahora func() time.Time
}

// Superficie es el http.Handler de la pantalla del calendario.
type Superficie struct {
	o        OpcionesPantalla
	mux      *http.ServeMux
	motor    *plantilla.Motor
	patrones []string
}

var _ http.Handler = (*Superficie)(nil)

// NuevaPantalla construye la superficie y registra sus rutas.
func NuevaPantalla(o OpcionesPantalla) (*Superficie, error) {
	if o.Catalogo == nil {
		return nil, fmt.Errorf("%w: sin catalogo la pantalla saldria con las claves en vez de "+
			"las palabras, y esta es la pantalla que un CISO abre para saber que le vence",
			ErrSinCatalogo)
	}
	if err := validarCamino(o.CaminoRuta, o.CaminoClave); err != nil {
		return nil, err
	}
	// LOS PASOS LOS JUZGA EL MISMO VALIDADOR que la pantalla del camino. Dos
	// jueces de la misma propiedad acaban discrepando, y el dia que discrepen la
	// barra lateral y la pantalla del camino ensenarian caminos distintos.
	if len(o.Pasos) > 0 {
		if err := camino.Validar(o.Pasos); err != nil {
			return nil, fmt.Errorf("calendario: el camino que se va a pintar en la barra "+
				"lateral no es recorrible: %w", err)
		}
	}
	o.Base = strings.TrimSuffix(o.Base, "/")
	if o.Ahora == nil {
		o.Ahora = func() time.Time { return time.Now().UTC() }
	}
	// LAS PROPIAS MAS EL ARMAZON COMPARTIDO: la barra lateral de esta pantalla
	// es la misma que la de las demas y sale del mismo fichero. Copiar el
	// marcado aqui habria abierto la quinta copia de una barra de navegacion.
	m, err := plantilla.Nuevo(camino.ConArmazon(plantillasFS), o.Catalogo,
		"plantillas/*.html", camino.PatronDelArmazon)
	if err != nil {
		return nil, fmt.Errorf("calendario: no se pueden cargar las plantillas: %w", err)
	}
	s := &Superficie{o: o, mux: http.NewServeMux(), motor: m}
	// El patron lleva la Base dentro, igual que en las demas superficies: un
	// montaje que se olvida el prefijo no da error, da 404 en todo.
	s.registrar("GET "+s.o.Base+"/{$}", s.ver)
	s.registrar("GET "+s.o.Base+"/"+FicheroICS, s.ics)
	return s, nil
}

// validarCamino comprueba el enlace de vuelta al camino guiado.
//
// SE COMPRUEBA AQUI Y TAMBIEN EN LAS DEMAS SUPERFICIES, a proposito: cada una
// recibe el dato por su cuenta y cada una lo pinta, asi que cada una tiene su
// frontera. Una guarda escrita solo en quien monta se cae el dia que monte otro.
func validarCamino(ruta, clave string) error {
	if ruta == "" && clave == "" {
		return nil // el valor cero: no se pinta nada
	}
	if ruta == "" || clave == "" {
		return fmt.Errorf("%w: llega la direccion %q y el rotulo %q, y hacen falta los dos. "+
			"Arreglo: pasar CaminoRuta y CaminoClave juntos, o ninguno", ErrCamino, ruta, clave)
	}
	if !strings.HasPrefix(ruta, "/") || strings.HasPrefix(ruta, "//") {
		return fmt.Errorf("%w: la direccion del camino es %q y tiene que empezar por una sola "+
			"barra. Con dos, el navegador la lee como otro anfitrion y el enlace saca al "+
			"operador de plazum", ErrCamino, ruta)
	}
	return nil
}

// registrar es el UNICO sitio por el que se registra una ruta, y anota el
// patron para que la puerta de CSRF de serve pueda enumerarlas sin conocer este
// paquete.
func (s *Superficie) registrar(patron string, h http.HandlerFunc) {
	s.patrones = append(s.patrones, patron)
	s.mux.HandleFunc(patron, h)
}

// Patrones son las rutas registradas.
func (s *Superficie) Patrones() []string { return append([]string(nil), s.patrones...) }

func (s *Superficie) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

// Ruta es la direccion de esta pantalla, ya con su prefijo.
func (s *Superficie) Ruta() string { return s.o.Base + "/" }

func (s *Superficie) idioma(r *http.Request) string {
	return s.motor.Resolver(r.Header.Get("Accept-Language"))
}

// ver pinta el calendario entero.
func (s *Superficie) ver(w http.ResponseWriter, r *http.Request) {
	v, codigo := s.vista(r)
	var b strings.Builder
	// El cuerpo se arma entero antes de tocar el ResponseWriter: html/template
	// escribe segun ejecuta, y un fallo a mitad dejaria media pagina con un 200.
	if err := s.motor.Render(&b, "pagina", v, s.idioma(r)); err != nil {
		http.Error(w, s.o.Catalogo.Traducir(s.idioma(r), "calendario.pantalla.error_render"),
			http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(codigo)
	_, _ = w.Write([]byte(b.String()))
}

// ics sirve el MISMO calendario en iCalendar.
//
// Se sirve desde aqui y no desde otra superficie porque el escritor ya vive en
// este paquete: la pantalla y el fichero salen del mismo `Derivado`, asi que no
// puede haber uno que diga una cosa y otro que diga otra.
//
// SIN ALCANCE NO SE SIRVE UN FICHERO VACIO: se contesta 404. Un `.ics` con cero
// eventos importado en Outlook se lee como «no tengo nada que hacer», que es la
// afirmacion mas cara que este producto puede hacer sin datos.
func (s *Superficie) ics(w http.ResponseWriter, r *http.Request) {
	idi := s.idioma(r)
	if s.o.Fuente == nil {
		http.Error(w, s.o.Catalogo.Traducir(idi, "calendario.pantalla.ics_sin_alcance"),
			http.StatusNotFound)
		return
	}
	d, hay, err := s.o.Fuente.Actual()
	if err != nil {
		http.Error(w, s.o.Catalogo.Traducir(idi, "calendario.pantalla.error_fuente"),
			http.StatusInternalServerError)
		return
	}
	if !hay {
		http.Error(w, s.o.Catalogo.Traducir(idi, "calendario.pantalla.ics_sin_alcance"),
			http.StatusNotFound)
		return
	}
	var b strings.Builder
	if err := Escribir(&b, d.Calendario, Opciones{
		Ahora: s.o.Ahora(), Organizacion: d.Organizacion,
	}); err != nil {
		http.Error(w, s.o.Catalogo.Traducir(idi, "calendario.pantalla.error_render"),
			http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	// El nombre del fichero es fijo y no sale de ningun dato del cliente: una
	// cabecera Content-Disposition compuesta con texto de fuera es una via de
	// inyeccion de cabecera, y aqui no hace ninguna falta tenerla.
	w.Header().Set("Content-Disposition", `attachment; filename="`+FicheroICS+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
}

// vista arma el modelo. Aqui NO se deriva nada: se pregunta y se ordena.
func (s *Superficie) vista(r *http.Request) (Vista, int) {
	v := Vista{
		Idioma: s.idioma(r), Base: s.o.Base, Estatico: s.o.Estatico,
		Titulo: "calendario.pantalla.titulo",
		Camino: EnlaceCamino{URL: s.o.CaminoRuta, Clave: s.o.CaminoClave},
		ICS:    s.o.Base + "/" + FicheroICS,
		Inicio: camino.InicioDe(s.o.Raiz),
		// LA BARRA LATERAL, marcando el paso del calendario. El identificador
		// sale del propio camino (camino.IDDelCalendario) y no de un literal
		// aqui: un literal se queda viejo el dia que el paso se renombre, y el
		// sintoma seria una barra que no marca nada y no dice nada al ponerse asi.
		Tira: camino.TiraDe(s.o.Pasos, s.o.Raiz, s.o.CaminoRuta,
			camino.IDDelCalendario, ""),
	}
	// EL CAMINO SE PINTA EN TODOS LOS ESTADOS, incluido el vacio. Es justo el
	// estado en el que alguien se queda mirando una pagina que no le dice nada,
	// asi que es en el que mas falta hace saber por donde se sigue.
	if s.o.Fuente == nil {
		v.SinAlcance = true
		return v, http.StatusOK
	}
	d, hay, err := s.o.Fuente.Actual()
	if err != nil {
		// UN FALLO AL DERIVAR NO SE CONVIERTE EN «no tienes nada». Son dos cosas
		// distintas y solo una es inocua: si el alcance no se puede leer, quien
		// mira tiene que saber que hay algo roto, no creerse que no le vence
		// nada. Invariante 8 en la pantalla que mas se mira.
		v.SinAlcance = true
		v.Aviso = err.Error()
		return v, http.StatusInternalServerError
	}
	if !hay {
		v.SinAlcance = true
		return v, http.StatusOK
	}
	v.rellenarCon(d)
	return v, http.StatusOK
}
