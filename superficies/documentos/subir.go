package documentos

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/marcosmatalab/plazum/adaptadores/ingesta"
)

// LA SUBIDA DE UN DOCUMENTO DEL CLIENTE.
//
// # ESTO ES UNA COPIA DELIBERADA DE superficies/uar/abrir.go, y se cita
//
// Aquel fichero resolvio esta misma frontera el 05-09-2026 y la resolvio bien.
// Lo que se copia, pieza por pieza, y lo que cada pieza compra:
//
//	MaxBytesReader con MARGEN     el sobre del multipart (fronteras, cabeceras de
//	                              parte, el nombre del fichero) no es el
//	                              documento pero viaja con el. Sin margen, un
//	                              fichero de EXACTAMENTE el tope se rechaza por
//	                              el peso del sobre, que es un rechazo que nadie
//	                              puede entender.
//	la supresion de G120        con su regla y su motivo escritos. gosec ve este
//	                              paquete solo, no sigue la asignacion a r.Body y
//	                              marca toda llamada a ParseMultipartForm. Va EN
//	                              EL FICHERO y no en el workflow porque gosec SI
//	                              corre en CI: una directiva dirigida a una
//	                              herramienta ausente afirma que alguien miro
//	                              esto, y no lo miro nadie.
//	RemoveAll diferido            el multipart deja ficheros temporales en disco
//	                              en cuanto pasa de maxMemoriaDelFormulario.
//	el error de parseo NO se pinta trae rutas de ficheros temporales y detalles
//	                              del transporte que no le dicen nada a quien
//	                              esta delante.
//	LimitReader(f, Max+1)         el byte de mas, para distinguir «justo en el
//	                              tope» de «se ha pasado». Con el limite exacto,
//	                              un fichero de Max+1 se leeria TRUNCADO y se
//	                              indexaria media politica sin que nadie lo
//	                              supiera, que es peor que rechazarla.
//
// # LAS TRES FORMAS DE LA NADA, y la tercera es la que este bloque anunciaba
//
// El invariante 8 dice que un campo obligatorio tiene tres respuestas y no dos:
// AUSENTE, PRESENTE Y EN BLANCO, y PRESENTE Y NO INTERPRETABLE. La tercera es
// siempre error y nunca el valor por defecto, y CLAUDE.md la anuncio con este
// caso exacto: «un PDF del que no se extrae nada interpretable es exactamente
// ese caso, y la tentacion va a ser el defecto silencioso».
//
// Aqui las tres se separan y ninguna se lee como otra:
//
//	ausente               no llega el campo del fichero      -> falta_fichero
//	presente y en blanco  llega y pesa cero bytes            -> vacio
//	presente e ilegible   llega, pesa, y de dentro no sale
//	                      texto que se pueda entender        -> no_se_entiende
//
// La tercera NO LA DECIDE ESTA SUPERFICIE: la decide `ingesta.Leer`, que ya
// distingue un fichero vacio de verdad (documento con cero fragmentos, sin
// error) de uno con contenido del que no sale texto util
// (ErrSinTextoInterpretable). Lo que hace esta superficie es no tragarse esa
// diferencia, que es la unica forma de perderla.
//
// # Y EL DESCARGO VIAJA CON EL ERROR
//
// «No se ha podido leer» no es «el documento no dice nada», y presentar lo
// segundo como lo primero es acusar en falso al documento de alguien.
// `ingesta.FraseDelNoLeido` es esa frase y es una constante del nucleo; aqui se
// pide por su clave de catalogo para que salga traducida, y la puerta que las
// ata letra por letra vive con las demas.

// MaximoDelDocumento acota el fichero que se acepta por esta pantalla.
//
// SE DECLARA AQUI Y SE DICE EN LA PANTALLA, en vez de dejarlo al limite de
// cuerpo del servidor. Son dos topes distintos y hacen falta los dos: el de
// `superficies/serve` protege el proceso de un POST de un gigabyte y no sabe de
// que iba la peticion, asi que su unico mensaje posible es generico. Este sabe
// que lo que se subia era un documento, asi que puede decir cuanto cabe.
//
// 4 MiB y no los 20 MiB de `ingesta.MaximoEntrada`, y el motivo es medido y no
// de gusto: dentro de `plazum serve` el cuerpo de toda peticion ya va acotado
// por debajo de eso, asi que un tope mayor aqui seria un numero que la pantalla
// promete y el servidor no cumple, y quien subiera 10 MiB se llevaria el error
// generico del anfitrion en vez del de esta pantalla.
const MaximoDelDocumento = 4 << 20

// maxMemoriaDelFormulario es lo que se tiene en RAM antes de tirar de fichero
// temporal. Por debajo del tope de arriba a proposito: el resto lo pone
// mime/multipart en disco y se borra al terminar la peticion.
const maxMemoriaDelFormulario = 1 << 20

// margenDelSobre es lo que el multipart anade por encima del fichero.
//
// 64 KiB sobran y no cambian el orden de magnitud del tope. Y va con su medida,
// porque la ventana que compra NO es exactamente el margen: el sobre lleva
// dentro el NOMBRE del fichero, asi que un nombre largo la estrecha. Lo que el
// margen garantiza es que un fichero de exactamente MaximoDelDocumento entra;
// cuanto mas alla entra, depende del nombre y no se promete.
const margenDelSobre = 64 << 10

// CampoDelFichero es el nombre del campo del formulario. Esta aqui y no escrito
// en la plantilla y en el handler por separado porque es un contrato entre los
// dos, y un contrato escrito dos veces se corrige una vez.
const CampoDelFichero = "documento"

// CampoCSRF es el name del input oculto. Coincide con el de superficies/serve a
// proposito: es el mismo middleware el que lo lee.
const CampoCSRF = "csrf"

// MaxLargoDelNombre acota el nombre que declara el navegador.
//
// El nombre NO decide nada (el formato lo reconoce el contenido), pero se pinta,
// asi que un nombre de un megabyte seria una pagina de un megabyte. Se recorta
// en vez de rechazarse porque es un adorno: rechazar una politica por como se
// llama su fichero seria un rechazo que nadie entiende.
const MaxLargoDelNombre = 120

// subir recibe el documento, lo valida en la frontera y se lo da al almacen.
func (s *Superficie) subir(w http.ResponseWriter, r *http.Request) {
	// 1. EL AUTOR. Sin sesion no se pinta un aviso sobre la pantalla: se
	//    devuelve la pantalla de sin sesion con su 401. Quien llega aqui sin
	//    haber entrado no tiene que enterarse de que hay documentos detras. Es
	//    el mismo trato que superficies/uar/abrir.go.
	quien := s.quien(r)
	if quien == "" {
		v, _ := s.vista(r)
		v.SinSesion = true
		s.pintar(w, r, v, http.StatusUnauthorized)
		return
	}
	// 2. EL ALMACEN. Llegar aqui con nil es imposible por construccion (sin
	//    almacen la ruta no se registra), y se comprueba igual: una ruta que
	//    depende de que su constructor no se equivoque no esta protegida, esta
	//    de suerte.
	if s.o.Almacen == nil {
		s.conAvisoDe(w, r, "documentos.subir.sin_almacen")
		return
	}

	// EL CUERPO SE ACOTA AQUI, Y NO SOLO EN QUIEN MONTA. Esta superficie es un
	// http.Handler que alguien puede montar en otro sitio, y una que depende de
	// que su anfitrion la proteja no esta protegida.
	r.Body = http.MaxBytesReader(w, r.Body, MaximoDelDocumento+margenDelSobre)
	// #nosec G120 -- el cuerpo va acotado en la linea de ARRIBA con
	// http.MaxBytesReader, que es justo lo que esta regla pide; gosec no sigue
	// la asignacion a r.Body y marca toda llamada a ParseMultipartForm. La
	// supresion nombra su regla y su motivo, y va aqui y no en el workflow
	// porque gosec SI corre en CI.
	if err := r.ParseMultipartForm(maxMemoriaDelFormulario); err != nil {
		// EL ERROR DE PARSEO NO SE ENSENA TAL CUAL: trae rutas de ficheros
		// temporales y detalles del transporte.
		s.fallo(err)
		s.conAvisoDe(w, r, "documentos.subir.no_se_lee")
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	f, cabecera, err := r.FormFile(CampoDelFichero)
	if err != nil {
		// LA PRIMERA FORMA DE LA NADA: el campo no llega.
		s.conAvisoDe(w, r, "documentos.subir.falta_fichero")
		return
	}
	defer func() { _ = f.Close() }()

	// UN BYTE DE MAS, para poder distinguir «justo en el tope» de «se ha
	// pasado». Con el LimitReader al tamano exacto, un fichero de
	// MaximoDelDocumento+1 se leeria truncado y se indexaria media politica.
	datos, err := io.ReadAll(io.LimitReader(f, MaximoDelDocumento+1))
	if err != nil {
		s.fallo(err)
		s.conAvisoDe(w, r, "documentos.subir.no_se_lee")
		return
	}
	if len(datos) > MaximoDelDocumento {
		s.conAvisoDe(w, r, "documentos.subir.demasiado_grande",
			strconv.Itoa(MaximoDelDocumento>>20))
		return
	}
	// LA SEGUNDA FORMA DE LA NADA: el fichero esta y pesa cero. No es un
	// documento vacio que haya que indexar, es una subida que salio mal, y
	// dejarla pasar daria una pantalla que dice que tu politica no habla de
	// nada.
	if len(datos) == 0 {
		s.conAvisoDe(w, r, "documentos.subir.vacio")
		return
	}

	nombre := nombreLimpio(cabecera.Filename)
	res, err := s.o.Almacen.Subir(r.Context(), quien, nombre, datos)
	if err != nil {
		s.avisoDeIngesta(w, r, err)
		return
	}
	if res.Fragmentos == 0 {
		// LA TERCERA FORMA, en su version mas silenciosa: el fichero se ha
		// leido, no ha dado error, y no ha entrado ni una unidad citable. Eso
		// es un fichero presente y vacio de verdad, y decirlo es lo unico
		// honesto: sin esta rama la pantalla saldria con cero hallazgos y quien
		// la lea entenderia que su politica no habla de nada.
		s.conAvisoDe(w, r, "documentos.subir.sin_contenido")
		return
	}
	s.volver(w, r)
}

// avisoDeIngesta traduce un error de lectura a la clave que le corresponde.
//
// # POR CENTINELA Y NUNCA POR TEXTO
//
// `errors.Is` sobre los centinelas de `adaptadores/ingesta`, no un `strings`
// sobre su mensaje. Y el mensaje del nucleo NO se imprime: lo escribe
// `adaptadores/ingesta` en castellano, y pintarlo tal cual en la pagina inglesa
// es exactamente el defecto que hizo nacer roja la puerta de la procedencia del
// texto. Lo que se pinta es la clave, con sus datos aparte.
//
// EL DESCARGO VA CON EL ERROR, no en una nota al pie: las cuatro clases dicen
// que plazum no ha podido leer el documento, y ninguna dice que el documento no
// sirva. La clave de cada una lo lleva dentro.
func (s *Superficie) avisoDeIngesta(w http.ResponseWriter, r *http.Request, err error) {
	s.fallo(err)
	switch {
	case errors.Is(err, ingesta.ErrSinTextoInterpretable):
		s.conAvisoDe(w, r, "documentos.subir.no_se_entiende")
	case errors.Is(err, ingesta.ErrFormatoNoReconocido):
		s.conAvisoDe(w, r, "documentos.subir.formato")
	case errors.Is(err, ingesta.ErrPDFCifrado):
		s.conAvisoDe(w, r, "documentos.subir.cifrado")
	case errors.Is(err, ingesta.ErrDemasiadoGrande):
		s.conAvisoDe(w, r, "documentos.subir.demasiado_grande",
			strconv.Itoa(MaximoDelDocumento>>20))
	default:
		// LO QUE NO SE RECONOCE NO SE INVENTA. Un error que esta superficie no
		// sabe clasificar sale como el generico y se anota por el canal de
		// quien monta: mapearlo al mas parecido diria que el documento no se
		// entiende cuando lo que ha pasado es otra cosa.
		s.conAvisoDe(w, r, "documentos.subir.no_se_guarda")
	}
}

// nombreLimpio deja el nombre en algo que se pueda pintar.
//
// El navegador manda lo que quiere, incluida una ruta entera. Se queda la ultima
// parte, se recorta y se descartan los caracteres de control: el nombre es un
// ADORNO (el formato lo decide el contenido), asi que no puede ser una via de
// entrada a la pantalla.
func nombreLimpio(s string) string {
	s = strings.TrimSpace(s)
	// Las dos separaciones, porque el nombre lo compone el navegador de quien
	// sube y puede venir de Windows aunque el servidor no sea Windows.
	if i := strings.LastIndexAny(s, `/\`); i >= 0 {
		s = s[i+1:]
	}
	s = filepath.Base(s)
	var b strings.Builder
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			continue
		}
		b.WriteRune(r)
		if b.Len() >= MaxLargoDelNombre {
			break
		}
	}
	limpio := strings.TrimSpace(b.String())
	if limpio == "" || limpio == "." || limpio == ".." {
		// Un nombre que no queda en nada NO se inventa y no se deja vacio: la
		// pantalla tiene que poder nombrar el documento en la lista.
		return "(sin nombre)"
	}
	return limpio
}
