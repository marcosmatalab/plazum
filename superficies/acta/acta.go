// Package acta es la pantalla del acta de revision por la direccion.
//
// # Que decide esta superficie
//
// Nada del documento. Los cubos, los repartos, la ley de conservacion, los
// descargos y el orden los pone nucleo/acta, que ademas produce el mismo
// documento en texto para imprimirlo. Aqui solo se decide que enlace lleva a
// donde. Si esta pantalla pudiera cambiar un numero, habria dos actas.
//
// # Todas las rutas son GET, y es una decision
//
// Un acta no se edita desde aqui: se compone de lo que ya esta registrado en el
// programa de auditoria, en la campana de accesos y en el registro de
// incidentes, y cada uno de esos se toca en su propia superficie. Un formulario
// aqui seria una segunda via para cambiar lo que el acta dice, que es
// exactamente lo que un documento probatorio no puede tener. Hay una puerta que
// enumera las rutas y le manda una peticion mutante a cada una.
//
// # Detras de sesion, y esto no es comodidad de interfaz
//
// El acta lleva NOMBRES DE PERSONAS, y mas que la pantalla de la UAR: quien
// audito cada unidad, quien difirio y con que motivo, quien decidio cada acceso,
// quien excuso una linea ilegible y quien asistio a la revision. La UAR ensena
// los sujetos revisados; esta ensena ademas a los ACTORES, que es una lista de
// quien hizo que dentro de la organizacion. Servirla a quien no ha entrado seria
// publicar eso.
//
// Lo que el acta NO ensena por defecto es el rotulo con el que el IdP nombra
// cada cuenta revisada, porque el compositor lo apaga con el valor cero de
// `acta.Entradas.ConNombresDelCenso`. Quien quiera ver a las personas del censo
// tiene la pantalla de la UAR, que es de quien es ese dato.
//
// # La derivacion, que aqui SI es un clic
//
// Cada cifra del acta enlaza a su derivacion por la referencia estable que le da
// el nucleo (1.2.3 = seccion, reparto, cubo). En el board pack impreso esa
// referencia manda al apendice; aqui manda a una pagina con la lista entera de
// lo que compone el numero. Es la misma promesa por dos medios, y por eso la
// referencia la calcula el nucleo y no esta superficie: si la calculara aqui, el
// papel y la pantalla podrian numerar distinto.
package acta

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/marcosmatalab/plazum/adaptadores/plantilla"
	"github.com/marcosmatalab/plazum/nucleo/acta"
	"github.com/marcosmatalab/plazum/puertos"
)

// Actas es de donde sale el acta que se pinta.
//
// La superficie no sabe de ficheros, de ledgers ni de como se compone: eso es
// del adaptador que la monta.
type Actas interface {
	// Ultima devuelve el acta que se ensena. Devolver (Acta{}, false, nil)
	// significa "todavia no hay ninguna", que NO es un error: es el estado
	// vacio, y esta pantalla lo sabe pintar con su siguiente paso.
	Ultima() (acta.Acta, bool, error)
}

// Opciones para construir la superficie.
type Opciones struct {
	// Fuente puede ser nil: entonces la pantalla existe y dice como tener un acta.
	Fuente Actas
	// Catalogo traduce. Obligatorio: sin el, la pantalla saldria con las claves.
	Catalogo puertos.Catalogo
	// Base es el prefijo bajo el que se monta, sin barra final.
	Base string
	// Estatico es de donde cuelga el CSS.
	Estatico string
	// CaminoRuta y CaminoClave son la vuelta al CAMINO GUIADO: la direccion de
	// la pantalla que dice en que orden se recorre plazum, y la clave de
	// catalogo de su rotulo. Los pone quien monta, que es el unico que sabe
	// donde esta montado el camino.
	//
	// EL VALOR CERO ES NO PINTAR NADA, que es el restrictivo. Lo que no se
	// admite es MEDIO enlace: una direccion sin rotulo o un rotulo sin
	// direccion se rechazan al construir, porque las dos mitades pintan un
	// enlace roto en la pantalla que un consejo va a leer.
	CaminoRuta  string
	CaminoClave string
	// Quien devuelve quien esta mirando. Nil, o cadena vacia, es "no ha
	// entrado", y entonces no se pinta el acta.
	//
	// Es el invariante 8 en una frontera de construccion: el valor cero de "no
	// se quien es" tiene que ser "no ensenes nombres", no "ensenalos".
	Quien func(*http.Request) string
}

// Superficie es el http.Handler.
type Superficie struct {
	o        Opciones
	mux      *http.ServeMux
	motor    *plantilla.Motor
	patrones []string
}

// Nuevo construye la superficie y registra sus rutas.
func Nuevo(o Opciones) (*Superficie, error) {
	if o.Catalogo == nil {
		return nil, errors.New("acta: falta el catalogo. Sin el, el acta saldria con las claves " +
			"en vez de las palabras, y este es el documento que lee un consejo")
	}
	if err := validarCamino(o.CaminoRuta, o.CaminoClave); err != nil {
		return nil, err
	}
	o.Base = strings.TrimSuffix(o.Base, "/")
	m, err := plantilla.Nuevo(plantillasFS, o.Catalogo, "plantillas/*.html")
	if err != nil {
		return nil, fmt.Errorf("acta: no se pueden cargar las plantillas: %w", err)
	}
	s := &Superficie{o: o, mux: http.NewServeMux(), motor: m}
	// Los patrones llevan la Base dentro, para que quien monte no tenga que
	// acordarse del StripPrefix. Un montaje que se olvida el prefijo no da un
	// error: da 404 en todas las rutas, que se lee como "la pantalla no existe".
	s.registrar("GET "+s.o.Base+"/{$}", s.ver)
	s.registrar("GET "+s.o.Base+"/derivacion/{ref}", s.derivacion)
	return s, nil
}

// ErrCamino: el enlace de vuelta al camino guiado llego a medias.
var ErrCamino = errors.New("acta: enlace al camino guiado invalido")

// validarCamino comprueba el enlace de vuelta al camino guiado.
//
// SE COMPRUEBA AQUI Y TAMBIEN EN LAS OTRAS SUPERFICIES, a proposito: cada una
// recibe el dato por su cuenta y cada una lo pinta, asi que cada una tiene su
// frontera. Es la misma razon por la que la familia de las URL de configuracion
// lleva dos guardas (invariante 11): una sola no llega.
//
// LAS DOS MITADES O NINGUNA. Una direccion sin rotulo pinta un enlace sin
// palabras y un rotulo sin direccion pinta uno que no lleva a ningun sitio; las
// dos salen del mismo descuido. Y la direccion tiene que ser de este sitio: con
// dos barras al principio el navegador la lee como otro anfitrion.
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
			"lector del acta fuera de plazum", ErrCamino, ruta)
	}
	return nil
}

// registrar es el UNICO sitio por el que se registra una ruta, y anota el
// patron. Misma disciplina que en las otras superficies: la puerta que enumera
// rutas le pregunta al enrutador y no a una lista escrita al lado del test.
func (s *Superficie) registrar(patron string, h http.HandlerFunc) {
	s.patrones = append(s.patrones, patron)
	s.mux.HandleFunc(patron, h)
}

// Patrones son las rutas registradas.
func (s *Superficie) Patrones() []string { return append([]string(nil), s.patrones...) }

func (s *Superficie) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

func (s *Superficie) idioma(r *http.Request) string {
	return s.motor.Resolver(r.Header.Get("Accept-Language"))
}

// ver pinta el acta entera.
func (s *Superficie) ver(w http.ResponseWriter, r *http.Request) {
	v, codigo := s.vista(r)
	s.pintar(w, r, v, codigo)
}

// derivacion abre UNA cifra: la lista entera de lo que la compone.
func (s *Superficie) derivacion(w http.ResponseWriter, r *http.Request) {
	v, codigo := s.vista(r)
	if codigo != http.StatusOK || v.SinActa {
		s.pintar(w, r, v, codigo)
		return
	}
	a, _, _ := s.o.Fuente.Ultima()
	c, hay := a.Derivar(r.PathValue("ref"))
	if !hay {
		// UNA REFERENCIA QUE NO EXISTE NO DEVUELVE UNA PAGINA VACIA con cara de
		// dato: dice que no. Una derivacion vacia se leeria como "este numero no
		// tiene nada detras", que es la afirmacion contraria a la que sostiene
		// este documento.
		v.NoExiste = true
		s.pintar(w, r, v, http.StatusNotFound)
		return
	}
	idi := s.idioma(r)
	v.Derivacion = derivacionVista(c, idi, s.o.Catalogo, s.o.Base)
	s.pintar(w, r, v, http.StatusOK)
}

func (s *Superficie) pintar(w http.ResponseWriter, r *http.Request, v Vista, codigo int) {
	// El cuerpo se arma entero antes de tocar el ResponseWriter: html/template
	// escribe segun ejecuta, y un fallo a mitad dejaria media pagina con un 200.
	var b strings.Builder
	if err := s.motor.Render(&b, "pagina", v, s.idioma(r)); err != nil {
		http.Error(w, s.o.Catalogo.Traducir(s.idioma(r), "acta.pantalla.error_render"),
			http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(codigo)
	_, _ = w.Write([]byte(b.String()))
}

// vista arma el modelo. Aqui NO se decide nada del documento: se pregunta.
func (s *Superficie) vista(r *http.Request) (Vista, int) {
	idi := s.idioma(r)
	v := Vista{Idioma: idi, Base: s.o.Base, Estatico: s.o.Estatico, Titulo: "acta.titulo.documento"}
	// EL CAMINO SE PINTA EN TODOS LOS ESTADOS, incluidos el de sin sesion y el
	// de sin acta. Son justo los dos en los que alguien se queda mirando una
	// pagina que no le dice nada, asi que son en los que mas falta hace saber
	// por donde se sigue.
	v.Camino = EnlaceCamino{URL: s.o.CaminoRuta, Clave: s.o.CaminoClave}

	// SIN SESION NO SE ENSENA EL ACTA. Ver el encabezado del paquete: aqui hay
	// una lista de quien hizo que dentro de la organizacion.
	if s.o.Quien == nil || strings.TrimSpace(s.o.Quien(r)) == "" {
		v.SinSesion = true
		return v, http.StatusUnauthorized
	}
	if s.o.Fuente == nil {
		v.SinActa = true
		return v, http.StatusOK
	}
	a, hay, err := s.o.Fuente.Ultima()
	if err != nil {
		v.SinActa = true
		v.Aviso = err.Error()
		return v, http.StatusInternalServerError
	}
	if !hay {
		v.SinActa = true
		return v, http.StatusOK
	}
	v.rellenarCon(a, idi, s.o.Catalogo, s.o.Base)
	return v, http.StatusOK
}
