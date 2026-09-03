// Package escalado es la pantalla del plan de avisos, EN SECO.
//
// # Que faltaba
//
// El escalado era el ultimo paso del camino guiado y no era una pantalla: la
// unica forma de saber a quien escribiria plazum era `plazum escalado` en una
// terminal. El camino lo decia con la orden delante, que es la puerta D11-b, y
// eso no lo convierte en menos hueco: el ultimo paso del recorrido del comprador
// terminaba en un bloque de texto para copiar.
//
// # ESTA PANTALLA NO MANDA NADA, Y NO PORQUE SE LE HAYA OLVIDADO
//
// Es la unica pieza del producto cuyo efecto SALE DE LA ORGANIZACION: un aviso
// mal disparado llega al correo de una persona y eso no se deshace con un
// ctrl-z. Por eso aqui solo hay rutas GET, no hay ni un formulario, y no existe
// ningun camino por el que una peticion a esta superficie provoque un envio.
// Lo que se pinta es el plan: si esto se pusiera a avisar ahora, a quien
// escribiria y de que.
//
// Mandar de verdad sigue siendo una decision que se toma en la terminal, con
// `plazum escalado --mandar`, sus canales y su lista de anfitriones permitidos.
// La pantalla dice esa orden, que es lo que la hace util en vez de un callejon.
// El dia que exista un boton de mandar, sera una superficie MUTANTE detras del
// CSRF de serve, como la revision de accesos, y sera otra decision: no se hereda
// de esta.
//
// # Detras de sesion, y esto no es comodidad de interfaz
//
// El plan lleva NOMBRES DE PERSONAS: quien ocupa cada figura y a quien le
// llegaria cada escalon. Servirlo a quien no ha entrado seria publicar el
// organigrama de responsabilidades de cumplimiento de la organizacion, que es
// exactamente el mapa que quiere quien prepara un ataque dirigido.
//
// # Esta superficie no planifica
//
// Recibe el plan ya resuelto por nucleo/escalado y lo pinta. Es la misma
// decision que en el acta y en el calendario: si esta pieza pudiera decidir un
// escalon, habria dos planes, el de la terminal y el del navegador, y el dia que
// se separaran nadie sabria cual va a mandar los correos.
package escalado

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/marcosmatalab/plazum/adaptadores/plantilla"
	"github.com/marcosmatalab/plazum/puertos"
	"github.com/marcosmatalab/plazum/superficies/camino"
)

// BasePorDefecto es el prefijo bajo el que se monta esta pantalla.
//
// Se exporta porque quien monta lo necesita para colgarla y el camino guiado lo
// necesita para declarar su paso: escribir "/escalado" en dos sitios es tener
// dos, y el sintoma de que se separan es un 404 en un paso del camino.
const BasePorDefecto = "/escalado"

var (
	// ErrSinCatalogo: sin catalogo la pantalla saldria con las claves crudas.
	ErrSinCatalogo = errors.New("escalado: pantalla sin catalogo")
	// ErrCamino: el enlace de vuelta al camino guiado llego a medias.
	ErrCamino = errors.New("escalado: enlace al camino guiado invalido")
)

// Fuente es de donde sale el plan que se pinta.
//
// Esta superficie no sabe de corpus, de alcances ni de canales: eso es del
// adaptador que la monta. Mismo contrato que `acta.Actas` y `calendario.Fuente`.
type Fuente interface {
	// EnSeco devuelve el plan de ahora. Devolver (Plan{}, false, nil) significa
	// «no hay alcance del que derivarlo», que NO es un error: es el estado
	// vacio, y esta pantalla lo pinta con su siguiente paso.
	//
	// SE PREGUNTA EN CADA PETICION: un plan calculado al arrancar diria a quien
	// habria que avisar el dia que se levanto el servidor.
	//
	// EL NOMBRE DEL METODO ES LA PROMESA. No se llama Plan() ni Avisos(): se
	// llama EnSeco porque lo unico que este interfaz puede devolver es un plan
	// que no se ha ejecutado. Un adaptador que mandara algo al contestar aqui
	// estaria mintiendo en la firma, no solo en el comentario.
	EnSeco() (Plan, bool, error)
}

// Opciones construye la superficie.
type Opciones struct {
	// Fuente puede ser nil: entonces la pantalla existe y dice como tener un
	// plan. Es la puerta D11-b.
	Fuente Fuente
	// Catalogo pone el texto. Obligatorio.
	Catalogo puertos.Catalogo
	// Base es el prefijo bajo el que se monta, sin barra final.
	Base string
	// Estatico es de donde cuelga el CSS.
	Estatico string
	// CaminoRuta y CaminoClave son la vuelta al camino guiado. El valor cero
	// (las dos vacias) es no pintar nada; medio enlace se rechaza al construir.
	CaminoRuta  string
	CaminoClave string
	// Pasos es EL CAMINO ENTERO, para la barra lateral compartida. Lo pasa
	// quien monta, con camino.Canonico(). El valor cero es no pintar barra, que
	// es el restrictivo: una barra inventada aqui enlazaria a rutas donde quien
	// monta no ha colgado nada.
	Pasos []camino.Paso
	// Raiz es el prefijo del SITIO del que cuelgan las rutas de los pasos, no el
	// de esta pantalla. Suele ser "" y por eso su valor cero vale.
	Raiz string
	// Quien devuelve quien esta mirando. Nil, o cadena vacia, es «no ha
	// entrado», y entonces no se pinta el plan.
	//
	// Invariante 8 en una frontera de construccion: el valor cero de «no se
	// quien es» tiene que ser «no ensenes nombres», no «ensenalos».
	Quien func(*http.Request) string
}

// Superficie es el http.Handler.
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
		return nil, fmt.Errorf("%w: sin catalogo la pantalla saldria con las claves en vez de "+
			"las palabras, y aqui una de esas palabras es la que dice que no se ha mandado nada",
			ErrSinCatalogo)
	}
	if err := validarCamino(o.CaminoRuta, o.CaminoClave); err != nil {
		return nil, err
	}
	// LOS PASOS LOS JUZGA EL MISMO VALIDADOR que la pantalla del camino: dos
	// jueces de la misma propiedad acaban discrepando.
	if len(o.Pasos) > 0 {
		if err := camino.Validar(o.Pasos); err != nil {
			return nil, fmt.Errorf("escalado: el camino que se va a pintar en la barra "+
				"lateral no es recorrible: %w", err)
		}
	}
	o.Base = strings.TrimSuffix(o.Base, "/")
	// LAS PROPIAS MAS EL ARMAZON COMPARTIDO: una sola copia de la barra de
	// navegacion para todas las superficies con pantalla.
	m, err := plantilla.Nuevo(camino.ConArmazon(plantillasFS), o.Catalogo,
		"plantillas/*.html", camino.PatronDelArmazon)
	if err != nil {
		return nil, fmt.Errorf("escalado: no se pueden cargar las plantillas: %w", err)
	}
	s := &Superficie{o: o, mux: http.NewServeMux(), motor: m}
	// UNA SOLA RUTA Y ES GET. Ver el encabezado del paquete: aqui no hay ni un
	// verbo que mute nada, a proposito.
	s.registrar("GET "+s.o.Base+"/{$}", s.ver)
	return s, nil
}

// validarCamino comprueba el enlace de vuelta al camino guiado. Cada superficie
// tiene su frontera: una guarda escrita solo en quien monta se cae el dia que
// monte otro.
func validarCamino(ruta, clave string) error {
	if ruta == "" && clave == "" {
		return nil
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

func (s *Superficie) ver(w http.ResponseWriter, r *http.Request) {
	v, codigo := s.vista(r)
	// El cuerpo se arma entero antes de tocar el ResponseWriter: html/template
	// escribe segun ejecuta, y un fallo a mitad dejaria media pagina con un 200.
	var b strings.Builder
	if err := s.motor.Render(&b, "pagina", v, s.idioma(r)); err != nil {
		http.Error(w, s.o.Catalogo.Traducir(s.idioma(r), "escalado.pantalla.error_render"),
			http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(codigo)
	_, _ = w.Write([]byte(b.String()))
}

func (s *Superficie) vista(r *http.Request) (Vista, int) {
	v := Vista{
		Idioma: s.idioma(r), Base: s.o.Base, Estatico: s.o.Estatico,
		Titulo: "escalado.pantalla.titulo",
		Camino: EnlaceCamino{URL: s.o.CaminoRuta, Clave: s.o.CaminoClave},
		Inicio: camino.InicioDe(s.o.Raiz),
		// LA BARRA LATERAL, marcando el paso del escalado. El identificador sale
		// del propio camino y no de un literal aqui.
		Tira: camino.TiraDe(s.o.Pasos, s.o.Raiz, s.o.CaminoRuta,
			camino.IDDelEscalado, ""),
	}
	// EL CAMINO SE PINTA EN TODOS LOS ESTADOS, incluidos el de sin sesion y el
	// de sin alcance: son justo los dos en los que alguien se queda mirando una
	// pagina que no le dice nada.
	if s.o.Quien == nil || strings.TrimSpace(s.o.Quien(r)) == "" {
		v.SinSesion = true
		return v, http.StatusUnauthorized
	}
	if s.o.Fuente == nil {
		v.SinAlcance = true
		return v, http.StatusOK
	}
	p, hay, err := s.o.Fuente.EnSeco()
	if err != nil {
		// UN FALLO AL PLANIFICAR NO SE CONVIERTE EN «no hay nada que avisar».
		// Son dos cosas distintas y solo una es inocua.
		v.SinAlcance = true
		v.Aviso = err.Error()
		return v, http.StatusInternalServerError
	}
	if !hay {
		v.SinAlcance = true
		return v, http.StatusOK
	}
	v.rellenarCon(p)
	return v, http.StatusOK
}
