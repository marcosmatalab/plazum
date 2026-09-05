package uar

import (
	"io"
	"net/http"
	"strings"
)

// LA SUBIDA DEL CENSO POR EL NAVEGADOR, que es lo que saca esta pantalla del
// terminal.
//
// # Por que existe, con el numero delante
//
// El estado vacio de esta pantalla contaba dos ordenes de terminal
// (`accesos ver` para sellar la campana y `serve` con sus dos ficheros para
// servirla). Medidas por el arnes del TTFV, esas dos son 3m0s de las 20m21s del
// camino guiado entero, sobre un presupuesto de 15m0s. No es que estuvieran mal
// escritas: es que **no habia otro camino**, y una pantalla que manda al
// terminal para llegar al valor es exactamente lo que prohibe la puerta D11-a.
//
// # LO QUE ESTA PIEZA NO HACE, dicho para que nadie lo suponga
//
// No decide nada del dominio. No sella, no cuenta cubos, no abre la campana:
// valida lo que llega por la frontera (que hay fichero, que cabe, que se dice de
// que sistema es, que hay un autor) y se lo pasa al adaptador, que es quien
// conoce el disco. El sellado lo sigue haciendo `nucleo/censo` y la apertura
// `nucleo/accesos`, exactamente igual que cuando entraba por la orden.
//
// # Y NO SUSTITUYE A LA ORDEN, la desplaza
//
// `plazum accesos ver` sigue existiendo y sigue siendo la via de quien
// automatiza. Lo que deja de ser es **obligatoria para llegar a la primera
// pantalla util**, que es lo unico que el TTFV mide.

// Aperturas es quien sabe abrir una campana con un fichero que llega por el
// navegador.
//
// Es un puerto aparte de Campanas a proposito, y no un metodo mas de aquel:
// LEER una campana y CREARLA son dos permisos distintos y dos capacidades
// distintas. Una instalacion puede tener montada la lectura sobre ficheros que
// gestiona otro sistema y no querer que la pantalla escriba en ellos, y con un
// solo interfaz eso no se puede decir.
//
// SU VALOR CERO (nil en Opciones) ES EL RESTRICTIVO: sin adaptador, la pantalla
// no pinta el formulario y dice por que, en vez de pintar un boton que contesta
// 500. Invariante 8 en una frontera de construccion.
type Aperturas interface {
	// Abrir sella una instantanea del censo con los datos que llegan y registra
	// su apertura. El error que devuelva se ensena TAL CUAL: los errores de
	// nucleo/censo ya traen la causa y el arreglo dentro, y envolverlos en
	// "no se pudo subir el fichero" cambiaria un error accionable por uno que
	// no lo es.
	//
	// datos es el contenido del CSV; sistema es de que sistema son esas cuentas
	// (el fichero no lo sabe) y quien es el sujeto de la sesion que lo sube.
	Abrir(datos []byte, sistema, quien string) error
}

// MaxCSVDelCenso acota el fichero que se acepta por esta pantalla.
//
// SE DECLARA AQUI Y SE DICE EN EL AVISO, en vez de dejarlo al limite de cuerpo
// del servidor. Son dos topes distintos y hacen falta los dos: el de
// superficies/serve protege el proceso de un POST de un gigabyte y no sabe de
// que iba la peticion, asi que su unico mensaje posible es generico. Este sabe
// que lo que se estaba subiendo es un censo, asi que puede decir cuanto cabe y
// que hacer si no cabe.
//
// El tamano sale de la cuenta, no del gusto: una fila de censo son unos 80
// bytes con cuenta, permiso y rotulo, asi que 2 MiB son del orden de 25.000
// filas. Un censo mayor que eso no se revisa a mano en una campana, se parte.
const MaxCSVDelCenso = 2 << 20

// maxMemoriaDelFormulario es lo que se tiene en RAM antes de tirar de fichero
// temporal. Por debajo del tope de arriba a proposito: el resto lo pone el
// paquete mime/multipart en disco y se borra al terminar la peticion.
const maxMemoriaDelFormulario = 1 << 20

// margenDelSobre es lo que el multipart anade por encima del fichero:
// fronteras, cabeceras de parte y el campo del sistema. 64 KiB sobran para
// eso y no llegan a cambiar el orden de magnitud del tope.
const margenDelSobre = 64 << 10

// CampoDelFichero y CampoDelSistema son los nombres de los dos campos del
// formulario. Estan aqui y no escritos en la plantilla y en el handler por
// separado porque son un contrato entre los dos, y un contrato escrito dos
// veces se corrige una vez.
const (
	CampoDelFichero = "fichero"
	CampoDelSistema = "sistema"
)

// abrir recibe el CSV, lo valida en la frontera y se lo da al adaptador.
func (s *Superficie) abrir(w http.ResponseWriter, r *http.Request) {
	idi := s.idioma(r)
	// 1. EL AUTOR. Igual que en decidir, excusar y cerrar: una instantanea del
	//    censo sin quien la subio no se puede atar a nadie y no certifica nada.
	//    Sin sesion no se pinta el aviso sobre la pantalla, se devuelve la
	//    pantalla de sin sesion con su 401: quien llega aqui sin haber entrado
	//    no tiene que enterarse de que hay un censo detras.
	quien := ""
	if s.o.Quien != nil {
		quien = strings.TrimSpace(s.o.Quien(r))
	}
	if quien == "" {
		v, _ := s.vista(r)
		v.SinSesion = true
		s.pintar(w, r, v, http.StatusUnauthorized)
		return
	}
	// 2. EL ADAPTADOR. Nil es el valor cero restrictivo, y llegar aqui con nil
	//    significa que alguien ha enviado el formulario a mano, porque sin
	//    adaptador la plantilla no lo pinta.
	if s.o.Abrir == nil {
		s.conAviso(w, r, s.o.Catalogo.Traducir(idi, "uar.subir.sin_almacen"))
		return
	}
	// EL CUERPO SE ACOTA AQUI, Y NO SOLO EN QUIEN MONTA.
	//
	// `superficies/serve` ya pone un tope de 4 MiB a toda peticion, asi que
	// dentro de `plazum serve` esto es la segunda vuelta. Hace falta igual
	// por dos motivos, y el segundo es el bueno: esta superficie es un
	// http.Handler que alguien puede montar en otro sitio, y una que
	// depende de que su anfitrion la proteja no esta protegida, esta de
	// suerte. Y lo dice tambien gosec (G120), que ve este paquete solo y
	// tiene razon en lo que puede ver.
	//
	// EL MARGEN es para el sobre del multipart: las fronteras, las
	// cabeceras de cada parte y el campo del sistema no son el censo pero
	// viajan con el, y sin margen un fichero de exactamente el tope se
	// rechazaria por el peso del sobre.
	r.Body = http.MaxBytesReader(w, r.Body, MaxCSVDelCenso+margenDelSobre)
	// #nosec G120 -- el cuerpo va acotado en la linea de ARRIBA con
	// http.MaxBytesReader, que es justo lo que esta regla pide; gosec no sigue
	// la asignacion a r.Body y marca toda llamada a ParseMultipartForm. La
	// supresion nombra su regla y su motivo, y va aqui y no en el workflow
	// porque gosec SI corre en CI: una directiva dirigida a una herramienta
	// ausente afirmaria que alguien miro esto y no lo miro nadie.
	if err := r.ParseMultipartForm(maxMemoriaDelFormulario); err != nil {
		// EL ERROR DE PARSEO NO SE ENSENA TAL CUAL. Trae rutas de ficheros
		// temporales y detalles del transporte que no le dicen nada a quien
		// esta delante, y el arreglo es siempre el mismo.
		s.conAviso(w, r, s.o.Catalogo.Traducir(idi, "uar.subir.no_se_lee"))
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()
	// 3. EL SISTEMA, OBLIGATORIO. Ausente y presente-en-blanco se tratan igual
	//    (las dos formas de la nada), y las dos son error: no hay valor por
	//    defecto porque no lo puede haber. Una lista de cuentas sin de que
	//    sistema es no se puede cotejar con nada.
	sistema := strings.TrimSpace(r.PostFormValue(CampoDelSistema))
	if sistema == "" {
		s.conAviso(w, r, s.o.Catalogo.Traducir(idi, "uar.subir.falta_sistema"))
		return
	}
	f, _, err := r.FormFile(CampoDelFichero)
	if err != nil {
		s.conAviso(w, r, s.o.Catalogo.Traducir(idi, "uar.subir.falta_fichero"))
		return
	}
	defer func() { _ = f.Close() }()
	// UN BYTE DE MAS, para poder distinguir «justo en el tope» de «se ha
	// pasado»: con el LimitReader al tamano exacto, un fichero de exactamente
	// MaxCSVDelCenso+1 se leeria truncado y se sellaria un censo al que le
	// falta la ultima fila, que es peor que rechazarlo.
	datos, err := io.ReadAll(io.LimitReader(f, MaxCSVDelCenso+1))
	if err != nil {
		s.conAviso(w, r, s.o.Catalogo.Traducir(idi, "uar.subir.no_se_lee"))
		return
	}
	if len(datos) > MaxCSVDelCenso {
		s.conAviso(w, r, s.o.Catalogo.Traducir(idi, "uar.subir.demasiado_grande"))
		return
	}
	// 4. EL FICHERO VACIO ES SU PROPIO CASO. Dejarlo pasar daria un censo de
	//    cero filas sellado y una campana abierta sin nada dentro, o sea una
	//    pantalla que dice que no hay nada que revisar cuando lo que pasa es
	//    que la subida salio mal. Es la tercera forma de la nada: hay un
	//    fichero, y no se entiende como censo.
	if len(datos) == 0 {
		s.conAviso(w, r, s.o.Catalogo.Traducir(idi, "uar.subir.vacio"))
		return
	}
	if err := s.o.Abrir.Abrir(datos, sistema, quien); err != nil {
		// TAL CUAL. Los errores de censo.Tomar dicen que columna falta, con que
		// separador se leyo y como se arregla; traducirlos a "el fichero no
		// vale" tiraria justo lo que sirve.
		s.conAviso(w, r, err.Error())
		return
	}
	s.volver(w, r)
}
