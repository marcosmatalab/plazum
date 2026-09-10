// Package documentos es la ruta por la que el cliente sube lo que YA TIENE
// escrito, y la pantalla que le dice donde mira cada norma dentro de ello.
//
// Es la pieza 3 de `docs/ia.md`, el mapeo de la evidencia que ya tiene, y toda
// la cadena que hay debajo existia en piezas sueltas desde antes: `ingesta` lee
// el fichero, `ia` lo convierte en fuentes citables y le pone hash, `busqueda`
// lo indexa y `evidencia` empareja. Lo que faltaba era la ruta por la que entra
// un documento y la pantalla por la que sale un hallazgo.
//
// # LO QUE ESTA PANTALLA DICE Y LO QUE NO DICE, que es la frontera entera
//
// Dice: «este parrafo de TU politica, pagina 4, habla de esto». No dice si eso
// CUMPLE, y la frontera esta puesta ahi a proposito y no por prudencia general.
// El encabezado de `adaptadores/evidencia` la razona entera y aqui no se repite:
// un falso «lo cumples» esconde un incumplimiento detras de una pantalla verde,
// y un falso «no lo cumples» es acusar en falso, que es el unico error que un
// producto de cumplimiento no puede cometer ni una vez.
//
// Esa frontera no vive en la buena intencion de esta pantalla: vive en el TIPO.
// `Hallazgo` no tiene ni un campo con forma de juicio, y eso lo vigila una
// puerta que aplica el vocabulario del invariante 13 —el mismo, leido de donde
// vive, no copiado— a los tipos que cruzan esta frontera.
//
// # POR QUE ESTA SUPERFICIE NO ES UN PASO DEL CAMINO GUIADO
//
// Porque el camino tiene presupuesto y esto lo reventaria. El modelo del TTFV
// cobra 45 s de lectura por cada paso, la medida de hoy es 14m38s y el
// presupuesto son 15m0s: una pantalla mas, aunque saliera EN BLANCO, deja el
// camino en 15m23s y reabre D11-e, que es la casilla que decide la fecha de la
// v1 y que se cerro el 10-09-2026.
//
// Asi que se entra A PROPOSITO, desde un enlace en `/alcance` con su cardinal al
// lado, que es la misma forma con la que se entra a la seccion de campos y a la
// pagina de una cifra. Consta en `cmd/plazum/alcanzabilidad.go` como
// MontadaFueraDelCamino con este motivo.
//
// # Y SE SIRVE CON SESION (invariante 12), contestado antes de escribir el cable
//
// La pregunta del invariante 12 se contesta una vez y por escrito: ¿esta
// superficie se sirve con sesion o sin ella? CON sesion, y no es una eleccion de
// comodidad. Lo que se sube es la politica de seguridad de una organizacion, o
// sea el documento interno de una persona; y lo que sale es en que controles
// mira. Servir eso sin sesion seria publicar el documento de alguien a un
// anonimo, que es el fallo textual del invariante 12 un piso por debajo.
//
// Sin sesion, GET y POST contestan 401 con una pantalla propia que NO dice si
// hay documentos subidos: quien llega aqui sin haber entrado no tiene que
// enterarse de que hay algo detras. Es el mismo trato que `superficies/uar`.
//
// # LO QUE ESTE PAQUETE NO HACE
//
// No lee ficheros, no indexa, no empareja y no conoce el corpus. Valida lo que
// llega por la frontera (que hay fichero, que cabe, que no viene vacio) y se lo
// da al puerto. El cable que junta ingesta, ia, busqueda y evidencia vive en
// `cmd/plazum`, que es el unico sitio que los conoce a los cuatro.
package documentos

import (
	"context"
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
// Se declara aqui y no en `superficies/camino` como los pasos, y esa es
// justamente la diferencia: los pasos del camino sacan su ruta de la
// declaracion del camino, y esto no es un paso. Ver el encabezado.
const BasePorDefecto = "/documentos"

// RutaDeSubir es el segmento de la ruta que muta.
//
// Va aqui, al lado de la base, porque es un contrato entre el enrutador, la
// plantilla que pinta el formulario y la puerta que enumera rutas mutantes. Un
// contrato escrito en tres sitios se corrige en uno.
const RutaDeSubir = "/subir"

// Almacen es quien sabe guardar lo que sube una cuenta y mapearlo contra el
// corpus.
//
// # Por que la cuenta va en la FIRMA y no en el constructor
//
// Porque el alcance de este dato es la CUENTA y no la instalacion (invariante
// 12), y un puerto que no lleva la cuenta en la firma no puede distinguirlas: la
// implementacion tendria que sacarla de algun sitio, y el unico sitio disponible
// seria una variable compartida, que es exactamente la fuga. Con la cuenta en la
// firma, servir el documento de otro exige escribirlo a proposito.
//
// SU VALOR CERO (nil en Opciones) ES EL RESTRICTIVO: sin almacen, la pantalla NO
// pinta el formulario, dice por que, y la ruta que muta NI SIQUIERA SE REGISTRA.
// Es el invariante 8 en una frontera de construccion, y con la vuelta de tuerca
// que este caso pide: un formulario que contesta 500 es peor que no tener
// formulario, pero una ruta mutante registrada sin nada detras es peor que las
// dos.
//
// LO VIGILA: TestSinAlmacenNoHayFormularioNiRutaQueMute, que comprueba las dos
// mitades (ni formulario ni ruta) y trae su control positivo, porque una
// superficie que no registrara nunca nada pasaria la segunda con nota.
type Almacen interface {
	// Subir lee el documento, lo indexa PARA ESA CUENTA y devuelve el resumen
	// de lo que ha entrado.
	//
	// El error que devuelva puede ser de `adaptadores/ingesta`, y esos errores
	// dicen que ha pasado y como se arregla. Esta superficie los traduce a una
	// clave de catalogo por su CENTINELA y nunca imprime su texto: un error del
	// nucleo escrito en castellano impreso en la pagina inglesa es el defecto
	// que ya se pago una vez.
	Subir(ctx context.Context, quien, nombre string, datos []byte) (ResumenDeDocumento, error)
	// Documentos son los que esa cuenta tiene subidos ahora mismo.
	Documentos(ctx context.Context, quien string) ([]ResumenDeDocumento, error)
	// Hallazgos mapea lo que esa cuenta ha subido contra lo que pide el corpus.
	//
	// Devuelve HECHOS: que parrafo del documento del cliente habla de que
	// obligacion, con su cita ya verificada por hash. Nunca un juicio.
	Hallazgos(ctx context.Context, quien string) ([]Hallazgo, error)
}

// ResumenDeDocumento es lo que se sabe de un documento que ya esta subido.
//
// NO LLEVA EL CONTENIDO. La pantalla de esta superficie ensena hallazgos, no el
// documento entero: volver a pintar la politica del cliente aqui seria un visor
// de PDF, que no es el producto.
type ResumenDeDocumento struct {
	// Fichero es el nombre que declaraba el navegador, ya saneado. No decide
	// nada: el formato lo reconoce el contenido.
	//
	// SE LLAMA ASI Y NO `Nombre` A PROPOSITO. El censo de procedencia del texto
	// contrasta la clasificacion contra las plantillas POR NOMBRE DE CAMPO, asi
	// que dos tipos que llamen igual a cosas de procedencia distinta apagan su
	// mitad negativa para ese nombre en TODO el arbol. `Nombre` ya esta cogido
	// por `VistaOculto.Nombre`, que es el nombre de un campo de formulario y no
	// es prosa; esto son las palabras de una persona. Llamarlos igual habria
	// costado una excepcion mas en `TestElHuecoDelContrasteDeProcedenciaSeCuenta`,
	// o sea ensanchar un agujero que existe para cerrarse.
	Fichero string
	// Huella es el sha256 del fichero, que es como se identifica un documento
	// aportado. Subir el mismo dos veces no duplica nada.
	Huella string
	// Fragmentos es cuantas unidades citables han entrado en el indice.
	Fragmentos int
	// Paginas es cuantas tiene el original. 0 en un formato sin paginas, y se
	// dice que es 0 en vez de fingir una.
	Paginas int
	// Truncado dice que se llego a un limite y hay contenido que NO se leyo.
	// Se pinta: leer media politica y decirlo es util, leer media y callarlo es
	// mentir.
	Truncado bool
	// NombreEnganoso dice que la extension no casa con lo que hay dentro. No es
	// un error, es informacion: la extension la elige quien sube el fichero.
	NombreEnganoso bool
}

// Hallazgo es un parrafo del documento del cliente que habla de lo que pide una
// norma.
//
// NO DICE QUE LO CUMPLA, y el tipo esta escrito para que no pueda decirlo: no
// hay campo booleano, ni puntuacion, ni estado. Lo vigila la puerta del
// invariante 13 sobre este tipo.
type Hallazgo struct {
	// Obligacion es el identificador de la obligacion del corpus.
	Obligacion string
	// Titulo y Marco salen del CORPUS y viajan tal cual, en el idioma del
	// paquete: son palabras de una norma y no se traducen.
	Titulo string
	Marco  string
	// Parrafo es el texto LITERAL del documento del cliente, ya verificado por
	// hash contra la fuente. Son las palabras de quien lo escribio y no pasan
	// por el catalogo.
	//
	// SE LLAMA ASI Y NO `Cita` POR LO MISMO QUE `Fichero` NO SE LLAMA `Nombre`,
	// y ademas porque aqui el nombre miente un poco menos: en el resto de este
	// arbol una `Cita` es una cita de una NORMA (siete campos, todos del
	// corpus), y esto es justo lo contrario, las palabras de quien monta la
	// organizacion. Un nombre que significa dos cosas opuestas segun el tipo es
	// la misma trampa que se cobro los nombres de tipo de esta superficie.
	Parrafo string
	// Pagina y Fragmento dicen DONDE esta dentro de su documento, y viajan como
	// NUMEROS y no como frase hecha.
	//
	// # Por que numeros y no la cadena que ya componia el nucleo
	//
	// Porque `adaptadores/ia` la componia en castellano («pagina 4», «fragmento
	// 7, pagina desconocida») y esta pantalla la habria impreso tal cual en la
	// pagina inglesa. Es el defecto por el que nacio roja la puerta de la
	// procedencia del texto, y la salida que esa puerta nombra no es declararlo
	// como deuda: es emitir clave y argumentos. La frase la pone el catalogo y
	// estos dos son sus datos.
	//
	// PAGINA CERO NO ES LA PAGINA CERO: es «no se sabe», y entonces la pantalla
	// lo dice con esas palabras en vez de poner una. Una pagina equivocada manda
	// a alguien a mirar donde no esta y le hace dudar del resto de la pantalla,
	// que es mas caro que no decirle la pagina.
	Pagina    int
	Fragmento int
	// Documento es el nombre del fichero del que sale.
	Documento string
}

// Errores de construccion, como centinelas.
var (
	ErrSinCatalogo = errors.New("documentos: falta el catalogo")
	ErrCamino      = errors.New("documentos: enlace al camino guiado invalido")
)

// Opciones construye la superficie.
type Opciones struct {
	// Catalogo pone el texto de la interfaz. Obligatorio: sin el, la pagina
	// saldria con las claves crudas.
	Catalogo puertos.Catalogo
	// Almacen es quien guarda y mapea.
	//
	// EL VALOR CERO ES NO TENER LA CAPACIDAD, y es el restrictivo de verdad y no
	// de boquilla: con `Almacen` nil la ruta que muta NO SE REGISTRA, asi que no
	// hay handler al que llegar, y la pantalla lo dice en vez de pintar un boton
	// que contestaria 500. La alternativa —registrar la ruta y rechazar dentro—
	// deja el rechazo colgando de que un `if` no se caiga; esto lo deja colgando
	// de que la ruta no exista.
	Almacen Almacen
	// Base es el prefijo bajo el que se monta, sin barra final.
	Base string
	// Estatico es de donde cuelgan el CSS y la marca, que sirve otra superficie.
	Estatico string
	// Raiz es el prefijo del SITIO, para componer el enlace al inicio.
	Raiz string
	// CaminoRuta y CaminoClave son la vuelta al camino guiado. Las dos o
	// ninguna: medio enlace da una entrada de menu rota.
	CaminoRuta  string
	CaminoClave string
	// Pasos es el camino entero, para la barra lateral. Esta superficie NO es
	// uno de esos pasos y no se marca ninguno: pintar la barra sin marcar nada
	// es lo que hace la pantalla del camino, y por lo mismo.
	//
	// EL VALOR CERO ES NO PINTAR LA BARRA LATERAL, y aqui si es lo correcto,
	// a diferencia de `camino.Pasos`, donde el cero esta PROHIBIDO. La razon es
	// que alli los pasos SON la pantalla y aqui son el marco: una barra que no
	// se pinta se nota y no engana a nadie, mientras que un camino guiado sin
	// pasos seria una pantalla que dice que plazum no tiene camino.
	//
	// LO QUE NO SE HACE ES RELLENARLO con camino.Canonico() cuando llega vacio.
	// Esta superficie no decide cual es el camino: si lo compusiera por su
	// cuenta, el dia que el cableado se olvide de pasarlo la barra seguiria
	// pintando una version suya, y dos sitios distintos ensenarian dos caminos.
	Pasos []camino.Paso
	// Tokens emite el token CSRF de esta peticion. Nil es NO PINTAR NINGUN
	// FORMULARIO y decir por que, que es el restrictivo.
	Tokens func(*http.Request) (string, error)
	// Quien devuelve quien esta operando. Nil o cadena vacia es NO HAY SESION,
	// y entonces todo contesta 401 con la pantalla de sin sesion.
	Quien func(*http.Request) string
	// AlFallar recibe los errores que no se le pueden ensenar a quien esta
	// delante. Nil los descarta.
	AlFallar func(error)
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
		return nil, fmt.Errorf("%w. Sin el, la pantalla saldria con las claves en vez de "+
			"las palabras", ErrSinCatalogo)
	}
	if err := validarCamino(o.CaminoRuta, o.CaminoClave); err != nil {
		return nil, err
	}
	// LOS PASOS LOS JUZGA EL MISMO VALIDADOR que la pantalla del camino. Dos
	// jueces de la misma propiedad acaban discrepando, y el dia que discrepen
	// esta barra y la del camino ensenaran caminos distintos.
	if len(o.Pasos) > 0 {
		if err := camino.Validar(o.Pasos); err != nil {
			return nil, fmt.Errorf("documentos: el camino que se va a pintar en la barra "+
				"lateral no es recorrible: %w", err)
		}
	}
	o.Base = strings.TrimSuffix(o.Base, "/")
	m, err := plantilla.Nuevo(camino.ConArmazon(plantillasFS), o.Catalogo,
		"plantillas/*.html", camino.PatronDelArmazon)
	if err != nil {
		return nil, fmt.Errorf("documentos: no se pueden cargar las plantillas: %w", err)
	}
	s := &Superficie{o: o, mux: http.NewServeMux(), motor: m}
	s.registrar("GET "+s.o.Base+"/{$}", s.ver)
	// LA RUTA QUE MUTA SOLO EXISTE SI HAY DONDE ESCRIBIR. No es una comodidad:
	// una ruta mutante registrada sin almacen detras es una ruta que la puerta
	// de CSRF tiene que cubrir, que un escaner encuentra y que contesta 500. El
	// valor cero de una capacidad es no tenerla, y aqui eso llega hasta el
	// enrutador.
	if s.o.Almacen != nil {
		s.registrar("POST "+s.o.Base+RutaDeSubir, s.subir)
	}
	return s, nil
}

// validarCamino comprueba el enlace de vuelta al camino guiado.
//
// LAS DOS MITADES O NINGUNA, y la direccion tiene que ser de este sitio: con dos
// barras al principio el navegador la lee como otro anfitrion, asi que el enlace
// que existe para no perder a nadie sacaria a quien lo pulse de plazum.
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
			"barra. Con dos, el navegador la lee como otro anfitrion", ErrCamino, ruta)
	}
	return nil
}

// registrar es el UNICO sitio por el que se registra una ruta, y anota el
// patron. Es la misma disciplina que en las demas superficies y por el mismo
// motivo: la puerta que enumera rutas mutantes le pregunta al enrutador, no a
// una lista escrita a mano al lado del test.
func (s *Superficie) registrar(patron string, h http.HandlerFunc) {
	s.patrones = append(s.patrones, patron)
	s.mux.HandleFunc(patron, h)
}

// Patrones son las rutas registradas, para que la puerta de CSRF de
// superficies/serve las enumere sin conocer este paquete.
func (s *Superficie) Patrones() []string { return append([]string(nil), s.patrones...) }

func (s *Superficie) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

// idioma resuelve el de la peticion contra los del catalogo.
func (s *Superficie) idioma(r *http.Request) string {
	return s.motor.Resolver(r.Header.Get("Accept-Language"))
}

// quien devuelve el sujeto de la sesion, o vacio.
func (s *Superficie) quien(r *http.Request) string {
	if s.o.Quien == nil {
		return ""
	}
	return strings.TrimSpace(s.o.Quien(r))
}

// ver pinta la pantalla.
func (s *Superficie) ver(w http.ResponseWriter, r *http.Request) {
	v, codigo := s.vista(r)
	s.pintar(w, r, v, codigo)
}

func (s *Superficie) pintar(w http.ResponseWriter, r *http.Request, v VistaDeDocumentos, codigo int) {
	// El cuerpo se arma entero antes de tocar el ResponseWriter: html/template
	// escribe segun ejecuta, y un fallo a mitad dejaria media pagina con un 200.
	var b strings.Builder
	if err := s.motor.Render(&b, "pagina", v, s.idioma(r)); err != nil {
		http.Error(w, s.o.Catalogo.Traducir(s.idioma(r), "documentos.error.render"),
			http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(codigo)
	_, _ = w.Write([]byte(b.String()))
}

// volver redirige a la pantalla despues de una mutacion, para que recargar no
// repita la subida.
func (s *Superficie) volver(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, s.o.Base+"/", http.StatusSeeOther)
}

// conAvisoDe vuelve a pintar la pantalla con el aviso arriba en vez de sacar a
// nadie de ella, y dice QUE GUARDA hablo.
//
// La clave viaja hasta el HTML como `data-aviso` y no se pinta: existe para que
// una puerta pueda afirmar cual de los rechazos ha ocurrido sin comparar prosa.
// Un test que compara el texto de un error se cae el dia que alguien lo mejore,
// y entonces la reaccion barata es aflojarlo hasta que solo compruebe que hubo
// rechazo, que es justo lo que no basta: un rechazo por el motivo equivocado es
// un verde que enmascara.
func (s *Superficie) conAvisoDe(w http.ResponseWriter, r *http.Request, clave string,
	args ...string) {

	v, _ := s.vista(r)
	v.AvisoClave = clave
	v.AvisoArgs = args
	s.pintar(w, r, v, http.StatusUnprocessableEntity)
}

// fallo anota lo que no se le puede ensenar a quien esta delante.
func (s *Superficie) fallo(err error) {
	if s.o.AlFallar != nil {
		s.o.AlFallar(err)
	}
}
