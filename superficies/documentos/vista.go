package documentos

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/marcosmatalab/plazum/superficies/camino"
)

//go:embed plantillas
var plantillasFS embed.FS

// Plantillas expone el sistema de ficheros embebido, por si alguien quiere
// montar otro motor de render.
func Plantillas() fs.FS { return plantillasFS }

// VistaDeDocumentos es lo unico que ve la plantilla.
//
// La regla es la de las demas superficies: los ROTULOS son claves de catalogo y
// el CONTENIDO viaja tal cual. Aqui el contenido es de dos procedencias
// distintas y ninguna se traduce: el texto de una norma (el titulo de una
// obligacion, que escribe el paquete) y el texto de una persona (la cita de su
// propia politica). Traducir cualquiera de los dos seria reescribir lo que dijo
// otro.
type VistaDeDocumentos struct {
	Idioma   string
	Base     string
	Estatico string
	// Titulo es la CLAVE de catalogo.
	Titulo string

	// AvisoClave es la CLAVE del rechazo y AvisoArgs sus datos.
	//
	// VIAJA COMO CLAVE Y NO COMO FRASE, que es la regla de la casa y aqui es
	// obligatorio: los rechazos de esta pantalla nacen de errores de
	// `adaptadores/ingesta`, que estan escritos en castellano. Resolverlos en Go
	// y mandar la frase hecha imprimiria castellano en la pagina inglesa, que es
	// el defecto exacto por el que nacio roja la puerta de la procedencia del
	// texto.
	//
	// Y la clave viaja ademas hasta el HTML como `data-aviso`, sin pintarse: es
	// lo que permite a una puerta afirmar QUE guarda ha rechazado sin comparar
	// prosa. Un test que compara el texto de un error se cae el dia que alguien
	// lo mejore, y entonces la reaccion barata es aflojarlo hasta que solo
	// compruebe que hubo rechazo, que es justo lo que no basta: un rechazo por el
	// motivo equivocado es un verde que enmascara.
	AvisoClave string
	// AvisoArgs son DATOS (un numero de MiB) y no se traducen.
	AvisoArgs []string

	CSRF      string
	CampoCSRF string
	// PuedeMutar dice si se pinta el formulario. Falso sin token.
	PuedeMutar bool
	// SinAlmacen dice que esta instalacion no sabe guardar documentos. Es el
	// valor cero del puerto, y la pantalla lo DICE en vez de pintar un boton
	// que no va a funcionar.
	SinAlmacen bool
	// SinSesion dice que no hay quien opere. La pantalla que se pinta entonces
	// no cuenta si hay documentos ni cuantos: quien llega sin haber entrado no
	// tiene que enterarse de que hay algo detras.
	SinSesion bool
	// Ilegible dice que el almacen esta y no se ha podido leer. Es la TERCERA
	// forma de la nada a nivel de pantalla, y no se puede confundir con las
	// otras dos: «todavia no has subido nada» y «no he podido mirar» son dos
	// cosas distintas, y presentar la segunda como la primera deja a alguien
	// tranquilo sobre un almacen roto.
	Ilegible bool

	// CampoDelFichero es el nombre del campo del formulario. Viaja a la
	// plantilla en vez de escribirse alli para que el contrato entre el
	// handler y el HTML se corrija en un solo sitio.
	CampoDelFichero string
	// MaximoMiB es el tope, en MiB, para poder decirlo en la pantalla ANTES de
	// que alguien intente subir algo que no cabe.
	MaximoMiB int

	// Documentos son los que esta cuenta tiene subidos.
	Documentos []ResumenDeDocumento
	// Hallazgos son los parrafos que hablan de lo que pide cada norma.
	Hallazgos []Hallazgo
	// LA FICHA PROPUESTA (pieza 7) y lo que ya se ha confirmado. Van juntas y
	// separadas a proposito: lo propuesto lleva un boton y lo aceptado lleva un
	// nombre, y confundirlos seria enseñar como dato algo que nadie ha mirado.
	Ficha     []PropuestaDeFicha
	Aceptados []CampoAceptado

	// Camino es la vuelta al camino guiado. Valor cero: no se pinta nada.
	Camino EnlaceCamino
	// Inicio es el enlace a la portada del sitio.
	Inicio string
	// Tira es el camino entero para la barra lateral, SIN NINGUN PASO MARCADO:
	// esta pantalla no es uno de sus pasos. Marcar uno diria que estas en el, y
	// no lo estas.
	Tira []camino.PasoTira
	// Idiomas es el conmutador. Vacio no pinta nada: con un solo idioma
	// cargado no hay nada que conmutar.
	Idiomas []camino.OpcionDeIdioma
}

// EnlaceCamino es la vuelta al camino guiado: la direccion y la CLAVE del
// rotulo. Las dos o ninguna, que se comprueba al construir.
type EnlaceCamino struct {
	URL   string
	Clave string
}

// Hay dice si el enlace esta entero. Medio enlace no se pinta: una direccion sin
// rotulo da un ancla vacia y un rotulo sin direccion da texto que no lleva a
// ningun sitio.
func (e EnlaceCamino) Hay() bool { return e.URL != "" && e.Clave != "" }

// vista arma el modelo. Aqui NO se decide nada del dominio: se pregunta.
//
// # Las cuatro respuestas posibles, y las tres ultimas se confunden solas
//
//	sin sesion    401 y la pantalla no cuenta si hay algo detras
//	sin almacen   200, sin formulario, y dice que esta instalacion no sabe
//	              guardar documentos
//	almacen roto  200, y dice que NO HA PODIDO MIRAR. No es «no has subido
//	              nada»: un fichero que no se lee no es un fichero vacio
//	todo bien     200 con lo que haya, que puede ser nada
func (s *Superficie) vista(r *http.Request) (VistaDeDocumentos, int) {
	v := VistaDeDocumentos{
		Idioma: s.idioma(r), Base: s.o.Base, Estatico: s.o.Estatico,
		Titulo:          "documentos.titulo",
		CampoDelFichero: CampoDelFichero,
		MaximoMiB:       MaximoDelDocumento >> 20,
		// EL CAMINO SE PINTA EN TODOS LOS ESTADOS, incluidos el de sin sesion y
		// el de sin almacen. Son justo los dos en los que quien llega se queda
		// mirando una pagina que no le dice nada.
		Camino: EnlaceCamino{URL: s.o.CaminoRuta, Clave: s.o.CaminoClave},
		Inicio: camino.InicioDe(s.o.Raiz),
		// SIN MARCAR NINGUN PASO. El identificador vacio deja la tira entera y
		// no senala ninguno, que es lo que hace la pantalla del camino y por lo
		// mismo: esta pantalla no es un paso.
		Tira: camino.TiraDe(s.o.Pasos, s.o.Raiz, s.o.CaminoRuta, "", ""),
		// Los idiomas del conmutador. Se componen aqui, donde hay peticion e
		// idioma actual: el enlace de cada uno es ESTA misma pagina con la
		// consulta intacta, y eso no se puede saber desde la plantilla.
		Idiomas: camino.OpcionesDeIdioma(r.URL.Path, s.motor, s.idioma(r), ""),
	}

	quien := s.quien(r)
	if quien == "" {
		v.SinSesion = true
		return v, http.StatusUnauthorized
	}
	if s.o.Almacen == nil {
		v.SinAlmacen = true
		return v, http.StatusOK
	}

	// EL TOKEN. Sin el no se pinta el formulario: un boton que no puede
	// funcionar es peor que no tener boton.
	if s.o.Tokens != nil {
		if t, err := s.o.Tokens(r); err == nil && t != "" {
			v.CSRF, v.CampoCSRF, v.PuedeMutar = t, CampoCSRF, true
		} else if err != nil {
			s.fallo(err)
		}
	}

	docs, err := s.o.Almacen.Documentos(r.Context(), quien)
	if err != nil {
		// UN ALMACEN QUE NO SE LEE NO ES UN ALMACEN VACIO. El error se anota
		// por el canal de quien monta y la pantalla dice que no ha podido
		// mirar, en vez de una lista vacia que se leeria como «no has subido
		// nada». Es el invariante 8 en su tercera forma, bajado a la pantalla.
		s.fallo(err)
		v.Ilegible = true
		return v, http.StatusOK
	}
	v.Documentos = docs
	if len(docs) == 0 {
		// Sin documentos no se piden hallazgos: la respuesta seria vacia y una
		// llamada que no puede aportar nada es una llamada que puede fallar
		// para nada.
		return v, http.StatusOK
	}

	hs, err := s.o.Almacen.Hallazgos(r.Context(), quien)
	if err != nil {
		s.fallo(err)
		v.Ilegible = true
		return v, http.StatusOK
	}
	v.Hallazgos = hs

	// LA FICHA PROPUESTA (pieza 7). Va DESPUES de los hallazgos y con el mismo
	// trato: si el almacen no se puede leer, la pantalla dice que no ha podido
	// mirar en vez de ensenar una ficha vacia, que se leeria como «tu documento
	// no tiene fecha».
	fs, err := s.o.Almacen.Ficha(r.Context(), quien)
	if err != nil {
		s.fallo(err)
		v.Ilegible = true
		return v, http.StatusOK
	}
	v.Ficha = fs
	ac, err := s.o.Almacen.Aceptados(r.Context(), quien)
	if err != nil {
		s.fallo(err)
		v.Ilegible = true
		return v, http.StatusOK
	}
	v.Aceptados = ac
	return v, http.StatusOK
}
