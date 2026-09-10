package documentos

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/catalogo"
	"github.com/marcosmatalab/plazum/adaptadores/ingesta"
	"github.com/marcosmatalab/plazum/superficies/camino"
)

// ---------------------------------------------------------------------------
// El arnes
// ---------------------------------------------------------------------------

// almacenFalso es un doble que guarda en memoria y POR CUENTA.
//
// Guarda por cuenta aunque sea un doble, y no es celo: un doble que ignorara la
// cuenta haria pasar el test de la fuga sin que el producto la impidiera, que es
// la forma mas barata de tener una puerta que no vigila.
type almacenFalso struct {
	docs  map[string][]ResumenDeDocumento
	halla map[string][]Hallazgo
	// err se devuelve en Subir; errLeer, en Documentos y Hallazgos.
	err     error
	errLeer error
	// sinFragmentos hace que Subir devuelva un resumen de CERO fragmentos, que
	// es el fichero presente y vacio de verdad.
	sinFragmentos bool
	subidas       int
	// LA FICHA (pieza 7). `ficha` son las propuestas que este doble devuelve y
	// `aceptados` lo que se ha confirmado, POR CUENTA como todo lo demas.
	ficha     map[string][]PropuestaDeFicha
	aceptados map[string][]CampoAceptado
	// sinFicha hace que Subir no proponga nada, para recorrer la rama de «no se
	// ha reconocido ningun campo», que no es lo mismo que no haber subido nada.
	sinFicha bool
}

func nuevoAlmacenFalso() *almacenFalso {
	return &almacenFalso{
		docs:      map[string][]ResumenDeDocumento{},
		halla:     map[string][]Hallazgo{},
		ficha:     map[string][]PropuestaDeFicha{},
		aceptados: map[string][]CampoAceptado{},
	}
}

func (a *almacenFalso) Ficha(_ context.Context, quien string) ([]PropuestaDeFicha, error) {
	if a.errLeer != nil {
		return nil, a.errLeer
	}
	return a.ficha[quien], nil
}

func (a *almacenFalso) Aceptados(_ context.Context, quien string) ([]CampoAceptado, error) {
	if a.errLeer != nil {
		return nil, a.errLeer
	}
	return a.aceptados[quien], nil
}

// Aceptar guarda con QUIEN, que es la mitad que importa de esta ruta. El doble
// lo guarda de verdad y no lo finge: un doble que ignorara el sujeto haria pasar
// el test de «lo aceptado lleva nombre» sin que el producto lo llevara.
func (a *almacenFalso) Aceptar(_ context.Context, quien string, p PropuestaDeFicha) error {
	if a.err != nil {
		return a.err
	}
	a.aceptados[quien] = append(a.aceptados[quien], CampoAceptado{
		Campo: p.Campo, Valor: p.Valor, Quien: quien, Cuando: "2026-09-11T09:00:00Z",
		Documento: p.Documento, Parrafo: p.Parrafo,
	})
	return nil
}

func (a *almacenFalso) Subir(_ context.Context, quien, nombre string, datos []byte) (
	ResumenDeDocumento, error) {

	a.subidas++
	if a.err != nil {
		return ResumenDeDocumento{}, a.err
	}
	r := ResumenDeDocumento{Fichero: nombre, Huella: fmt.Sprintf("%064x", len(datos)),
		Fragmentos: 1, Paginas: 2}
	if a.sinFragmentos {
		return ResumenDeDocumento{Fichero: nombre, Fragmentos: 0}, nil
	}
	a.docs[quien] = append(a.docs[quien], r)
	a.halla[quien] = append(a.halla[quien], Hallazgo{
		Obligacion: "demo.o.uno", Titulo: "Revision anual del plan",
		Marco: "urn:demo:m1", Parrafo: "La revision del plan se hace cada doce meses.",
		Pagina: 2, Fragmento: 1, Documento: nombre,
	})
	if !a.sinFicha {
		a.ficha[quien] = append(a.ficha[quien], PropuestaDeFicha{
			Campo: "documentos.ficha.campo.fecha", Valor: "2026-01-15",
			Parrafo: "Fecha: 2026-01-15", Documento: nombre, Pagina: 1, Fragmento: 0,
			Huella: r.Huella,
		})
	}
	return r, nil
}

func (a *almacenFalso) Documentos(_ context.Context, quien string) ([]ResumenDeDocumento, error) {
	if a.errLeer != nil {
		return nil, a.errLeer
	}
	return a.docs[quien], nil
}

func (a *almacenFalso) Hallazgos(_ context.Context, quien string) ([]Hallazgo, error) {
	if a.errLeer != nil {
		return nil, a.errLeer
	}
	return a.halla[quien], nil
}

func cat(t *testing.T) *catalogo.Catalogo {
	t.Helper()
	c, err := catalogo.Nuevo()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

type ajuste func(*Opciones)

func sinSesion(o *Opciones)  { o.Quien = nil }
func sinToken(o *Opciones)   { o.Tokens = nil }
func sinAlmacen(o *Opciones) { o.Almacen = nil }

func conAlmacen(a Almacen) ajuste { return func(o *Opciones) { o.Almacen = a } }

func superficie(t *testing.T, ajustes ...ajuste) *Superficie {
	t.Helper()
	o := Opciones{
		Catalogo: cat(t), Base: BasePorDefecto, Estatico: "/estatico",
		Almacen: nuevoAlmacenFalso(),
		Quien:   func(*http.Request) string { return "ciso@ejemplo" },
		Tokens:  func(*http.Request) (string, error) { return "tok-123", nil },
		Pasos:   camino.Canonico(),
	}
	for _, a := range ajustes {
		a(&o)
	}
	s, err := Nuevo(o)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func ver(t *testing.T, s *Superficie) (int, string) {
	t.Helper()
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(http.MethodGet, BasePorDefecto+"/", nil))
	return w.Code, w.Body.String()
}

// subir monta el multipart igual que lo monta un navegador.
func subir(t *testing.T, s *Superficie, nombre string, datos []byte) (int, string) {
	t.Helper()
	var cuerpo bytes.Buffer
	w := multipart.NewWriter(&cuerpo)
	if datos != nil {
		f, err := w.CreateFormFile(CampoDelFichero, nombre)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(datos); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, BasePorDefecto+RutaDeSubir, &cuerpo)
	r.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, r)
	return rec.Code, rec.Body.String()
}

// aceptar manda el formulario de confirmacion de un campo de la ficha.
func aceptar(t *testing.T, s *Superficie, campo, valor, huella string) (int, string) {
	t.Helper()
	v := url.Values{CampoDelCampo: {campo}, CampoDelValor: {valor}, CampoDeHuella: {huella}}
	r := httptest.NewRequest(http.MethodPost, BasePorDefecto+RutaDeAceptar,
		strings.NewReader(v.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, r)
	return rec.Code, rec.Body.String()
}

// avisoDe saca la clave del rechazo del atributo data-aviso.
//
// SE LEE LA CLAVE Y NO EL TEXTO, y es la leccion de la mutacion M2: un test que
// compara la prosa de un error se cae el dia que alguien la mejore, y entonces
// la reaccion barata es aflojarlo hasta que solo compruebe que hubo rechazo. Un
// rechazo por el motivo equivocado es un verde que enmascara.
func avisoDe(cuerpo string) string {
	i := strings.Index(cuerpo, `data-aviso="`)
	if i < 0 {
		return ""
	}
	resto := cuerpo[i+len(`data-aviso="`):]
	j := strings.Index(resto, `"`)
	if j < 0 {
		return ""
	}
	return resto[:j]
}

// ---------------------------------------------------------------------------
// La sesion (invariante 12)
// ---------------------------------------------------------------------------

// SIN SESION NO SE SIRVE NADA, Y TAMPOCO SE CUENTA QUE HAYA ALGO.
//
// Las dos mitades importan y la segunda es la que se olvida. Un 401 que
// contestara «no tienes permiso para ver los 3 documentos de esta cuenta» ya
// habria filtrado el dato: que existen y cuantos son. Lo que se sube aqui es la
// documentacion interna de una organizacion.
func TestSinSesionNoSeSirveNiSeCuentaQueHayaDocumentos(t *testing.T) {
	a := nuevoAlmacenFalso()
	a.docs["ciso@ejemplo"] = []ResumenDeDocumento{{Fichero: "politica.pdf", Fragmentos: 9}}
	a.halla["ciso@ejemplo"] = []Hallazgo{{Parrafo: "un parrafo secreto"}}

	s := superficie(t, conAlmacen(a), sinSesion)
	for _, c := range []struct {
		que     string
		codigo  int
		cuerpo  string
		metodo  string
		prohibe []string
	}{
		{que: "GET"},
		{que: "POST"},
	} {
		var codigo int
		var cuerpo string
		if c.que == "GET" {
			codigo, cuerpo = ver(t, s)
		} else {
			codigo, cuerpo = subir(t, s, "politica.pdf", []byte("hola que tal"))
		}
		if codigo != http.StatusUnauthorized {
			t.Errorf("%s sin sesion contesta %d y tenia que contestar 401", c.que, codigo)
		}
		for _, prohibido := range []string{"politica.pdf", "un parrafo secreto", "9"} {
			if strings.Contains(cuerpo, prohibido) {
				t.Errorf("%s sin sesion filtra %q, o sea que cuenta que hay algo detras.\n"+
					"  Un 401 que dice cuantos documentos hay ya ha filtrado el dato",
					c.que, prohibido)
			}
		}
	}
	// Y NO SE HA LLAMADO AL ALMACEN. Un 401 que primero consulta y despues
	// rechaza deja el dato leido en memoria del proceso y en los registros de
	// quien monta.
	if a.subidas != 0 {
		t.Errorf("sin sesion se ha llamado a Subir %d veces", a.subidas)
	}
}

// EL CONTROL POSITIVO DEL 401: con sesion, la pantalla SI sale y SI cuenta lo
// suyo. Sin esta mitad, una superficie que contestara 401 a todo el mundo
// pasaria la de arriba con nota.
func TestConSesionLaPantallaSaleYEnsenaLoDeEsaCuenta(t *testing.T) {
	s := superficie(t)
	codigo, cuerpo := subir(t, s, "politica.pdf", []byte("La revision del plan se hace cada doce meses."))
	if codigo != http.StatusSeeOther {
		t.Fatalf("la subida contesta %d y tenia que redirigir con 303: %s", codigo,
			avisoDe(cuerpo))
	}
	codigo, cuerpo = ver(t, s)
	if codigo != http.StatusOK {
		t.Fatalf("la pantalla contesta %d", codigo)
	}
	for _, quiero := range []string{"politica.pdf", "Revision anual del plan",
		"La revision del plan se hace cada doce meses.", "pagina 2"} {
		if !strings.Contains(cuerpo, quiero) {
			t.Errorf("la pantalla no ensena %q", quiero)
		}
	}
}

// LA FUGA ENTRE CUENTAS, en las DOS direcciones y con dos cuentas de verdad.
//
// Una cuenta sola no demuestra nada: el test pasaria porque no hay otra. Y sin
// el control positivo tampoco: una superficie que no ensenara nada a nadie
// tambien pasa «no se filtra».
func TestLoQueSubeUnaCuentaNoLoVeOtra(t *testing.T) {
	a := nuevoAlmacenFalso()
	quien := "ana@ejemplo"
	s, err := Nuevo(Opciones{
		Catalogo: cat(t), Base: BasePorDefecto, Almacen: a,
		Quien:  func(*http.Request) string { return quien },
		Tokens: func(*http.Request) (string, error) { return "tok", nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if codigo, cuerpo := subir(t, s, "de-ana.pdf", []byte("el parrafo de Ana")); codigo != http.StatusSeeOther {
		t.Fatalf("la subida de Ana contesta %d: %s", codigo, avisoDe(cuerpo))
	}
	// CONTROL POSITIVO: Ana SI ve lo suyo.
	if _, cuerpo := ver(t, s); !strings.Contains(cuerpo, "de-ana.pdf") {
		t.Fatal("Ana no ve su propio documento, asi que la comprobacion de abajo no " +
			"distingue «no hay fuga» de «no hay producto»")
	}
	// Y LUIS NO.
	quien = "luis@ejemplo"
	codigo, cuerpo := ver(t, s)
	if codigo != http.StatusOK {
		t.Fatalf("la pantalla de Luis contesta %d", codigo)
	}
	if strings.Contains(cuerpo, "de-ana.pdf") || strings.Contains(cuerpo, "el parrafo de Ana") {
		t.Error("Luis ve el documento de Ana.\n" +
			"  Es el invariante 12: lo que sube una persona es de su CUENTA, no de la " +
			"instalacion, y servirlo a otra es publicar el documento de alguien.")
	}
}

// ---------------------------------------------------------------------------
// El valor cero de las capacidades (invariante 8)
// ---------------------------------------------------------------------------

// SIN ALMACEN NO SE PINTA FORMULARIO Y LA RUTA QUE MUTA NI SE REGISTRA.
//
// Las dos mitades. La primera es la de siempre (un boton que contesta 500 es
// peor que no tener boton); la segunda es la que llega mas lejos: una ruta
// mutante registrada sin nada detras es una ruta que la puerta de CSRF tiene que
// cubrir y que un escaner encuentra.
func TestSinAlmacenNoHayFormularioNiRutaQueMute(t *testing.T) {
	s := superficie(t, sinAlmacen)
	codigo, cuerpo := ver(t, s)
	if codigo != http.StatusOK {
		t.Fatalf("sin almacen la pantalla contesta %d y tenia que contestar 200 diciendo "+
			"por que", codigo)
	}
	if strings.Contains(cuerpo, "<form") {
		t.Error("sin almacen se pinta un formulario que no puede funcionar")
	}
	if !strings.Contains(cuerpo, "documentos.sin_almacen.por_que") &&
		!strings.Contains(cuerpo, "no ha cableado") {
		t.Errorf("sin almacen la pantalla no dice por que no hay formulario:\n%s", cuerpo)
	}
	for _, p := range s.Patrones() {
		if strings.HasPrefix(p, "POST ") {
			t.Errorf("sin almacen se ha registrado la ruta mutante %q", p)
		}
	}
	// CONTROL POSITIVO: con almacen SI se registra, o lo de arriba se cumpliria
	// en una superficie que nunca registra nada.
	con := superficie(t)
	hay := false
	for _, p := range con.Patrones() {
		if strings.HasPrefix(p, "POST ") {
			hay = true
		}
	}
	if !hay {
		t.Error("con almacen tampoco se registra la ruta mutante: la comprobacion de " +
			"arriba no separa nada")
	}
}

// SIN TOKEN NO SE PINTA EL FORMULARIO. Es la misma familia y la otra capacidad.
func TestSinTokenNoSePintaElFormulario(t *testing.T) {
	s := superficie(t, sinToken)
	codigo, cuerpo := ver(t, s)
	if codigo != http.StatusOK {
		t.Fatalf("contesta %d", codigo)
	}
	if strings.Contains(cuerpo, `type="file"`) {
		t.Error("sin token se pinta el campo de fichero, o sea un formulario que el " +
			"middleware de CSRF va a rechazar")
	}
	if con := superficie(t); true {
		if _, c := ver(t, con); !strings.Contains(c, `type="file"`) {
			t.Error("con token tampoco se pinta: la comprobacion de arriba no separa nada")
		}
	}
}

// ---------------------------------------------------------------------------
// Las tres formas de la nada en la frontera de entrada (invariante 8)
// ---------------------------------------------------------------------------

// LAS TRES SE DISTINGUEN Y CADA UNA TIENE SU RECHAZO.
//
// Es el caso que CLAUDE.md anuncio con estas palabras: «un PDF del que no se
// extrae nada interpretable es exactamente ese caso, y la tentacion va a ser el
// defecto silencioso». Aqui la tentacion seria tratar las tres como «no ha
// llegado nada» y seguir, que dejaria a alguien mirando una pantalla que dice
// que su politica no habla de nada.
//
// SE AFIRMA LA CLAVE DEL RECHAZO Y NO SU TEXTO: un rechazo por el motivo
// equivocado es un verde que enmascara.
func TestLasTresFormasDeLaNadaEnLaSubidaSeDistinguen(t *testing.T) {
	for _, c := range []struct {
		que   string
		datos []byte
		err   error
		clave string
	}{
		{
			que:   "AUSENTE: no llega el campo del fichero",
			datos: nil,
			clave: "documentos.subir.falta_fichero",
		},
		{
			que:   "PRESENTE Y EN BLANCO: llega y pesa cero",
			datos: []byte{},
			clave: "documentos.subir.vacio",
		},
		{
			que:   "PRESENTE Y NO INTERPRETABLE: llega, pesa, y no sale texto",
			datos: []byte("%PDF-1.4 basura binaria sin capa de texto"),
			err:   fmt.Errorf("envuelto: %w", ingesta.ErrSinTextoInterpretable),
			clave: "documentos.subir.no_se_entiende",
		},
	} {
		t.Run(c.que, func(t *testing.T) {
			a := nuevoAlmacenFalso()
			a.err = c.err
			s := superficie(t, conAlmacen(a))
			codigo, cuerpo := subir(t, s, "x.pdf", c.datos)
			if codigo != http.StatusUnprocessableEntity {
				t.Fatalf("contesta %d y tenia que contestar 422", codigo)
			}
			if got := avisoDe(cuerpo); got != c.clave {
				t.Errorf("el rechazo es %q y tenia que ser %q.\n"+
					"  Las tres formas de la nada son tres respuestas distintas: "+
					"confundirlas es lo que convierte «no he podido leerlo» en «no dice "+
					"nada».", got, c.clave)
			}
		})
	}
}

// Y CADA CLASE DE ERROR DE LECTURA TIENE SU CLAVE, por CENTINELA y no por texto.
func TestCadaErrorDeLecturaSaleConSuPropiaClave(t *testing.T) {
	for _, c := range []struct {
		err   error
		clave string
	}{
		{ingesta.ErrSinTextoInterpretable, "documentos.subir.no_se_entiende"},
		{ingesta.ErrFormatoNoReconocido, "documentos.subir.formato"},
		{ingesta.ErrPDFCifrado, "documentos.subir.cifrado"},
		{ingesta.ErrDemasiadoGrande, "documentos.subir.demasiado_grande"},
		// LO QUE NO SE RECONOCE NO SE MAPEA AL MAS PARECIDO: sale el generico.
		{errors.New("algo que esta superficie no sabe clasificar"),
			"documentos.subir.no_se_guarda"},
	} {
		a := nuevoAlmacenFalso()
		a.err = fmt.Errorf("envuelto: %w", c.err)
		s := superficie(t, conAlmacen(a))
		_, cuerpo := subir(t, s, "x.pdf", []byte("contenido"))
		if got := avisoDe(cuerpo); got != c.clave {
			t.Errorf("%v sale como %q y tenia que salir como %q", c.err, got, c.clave)
		}
	}
}

// EL TOPE, CON EL BYTE DE MAS.
//
// Las dos direcciones: uno de exactamente el tope ENTRA (que es lo que el margen
// del sobre compra) y uno de un byte mas se RECHAZA. Sin la primera mitad, un
// tope mal puesto que rechazara el caso limite pasaria sin que nadie lo notara.
func TestUnFicheroDelTamanoDelTopeEntraYUnoDeUnByteMasNo(t *testing.T) {
	a := nuevoAlmacenFalso()
	s := superficie(t, conAlmacen(a))
	if codigo, cuerpo := subir(t, s, "justo.txt", bytes.Repeat([]byte("a"), MaximoDelDocumento)); codigo != http.StatusSeeOther {
		t.Errorf("un fichero de exactamente el tope contesta %d (%s) y tenia que entrar.\n"+
			"  Es lo que compra margenDelSobre: sin el, el peso del multipart rechaza el "+
			"caso limite y nadie entiende por que", codigo, avisoDe(cuerpo))
	}
	if codigo, cuerpo := subir(t, s, "pasado.txt", bytes.Repeat([]byte("a"), MaximoDelDocumento+1)); codigo == http.StatusSeeOther {
		t.Error("un fichero de un byte mas que el tope ha entrado")
	} else if got := avisoDe(cuerpo); got != "documentos.subir.demasiado_grande" && got != "documentos.subir.no_se_lee" {
		t.Errorf("un fichero pasado del tope sale como %q y tenia que decir que no cabe", got)
	}
}

// ---------------------------------------------------------------------------
// La tercera forma de la nada, en la pantalla
// ---------------------------------------------------------------------------

// UN ALMACEN QUE NO SE LEE NO ES UN ALMACEN VACIO.
//
// Es la misma familia bajada del disco a la pantalla: «todavia no has subido
// nada» y «no he podido mirar» se ven igual si esta rama no existe, y la primera
// deja a alguien tranquilo sobre un almacen roto.
func TestUnAlmacenQueNoSeLeeNoSeConvierteEnQueNoHasSubidoNada(t *testing.T) {
	a := nuevoAlmacenFalso()
	a.errLeer = errors.New("el almacen no contesta")
	var visto error
	s, err := Nuevo(Opciones{
		Catalogo: cat(t), Base: BasePorDefecto, Almacen: a,
		Quien:    func(*http.Request) string { return "ciso" },
		Tokens:   func(*http.Request) (string, error) { return "tok", nil },
		AlFallar: func(e error) { visto = e },
	})
	if err != nil {
		t.Fatal(err)
	}
	codigo, cuerpo := ver(t, s)
	if codigo != http.StatusOK {
		t.Fatalf("contesta %d", codigo)
	}
	if !strings.Contains(cuerpo, "no se ha podido mirar") {
		t.Errorf("la pantalla no dice que NO HA PODIDO MIRAR:\n%s", cuerpo)
	}
	if strings.Contains(cuerpo, "Todavia no has subido") {
		t.Error("con el almacen roto la pantalla dice que no has subido nada, que es " +
			"absolver de mas con cara de dato")
	}
	if visto == nil {
		t.Error("el error no ha salido por el canal de quien monta, asi que se ha " +
			"tragado en silencio")
	}
	// CONTROL POSITIVO: con el almacen sano y vacio, la pantalla SI dice que no
	// has subido nada. Sin esta mitad, una pantalla que dijera siempre «no he
	// podido mirar» pasaria lo de arriba.
	sano := superficie(t)
	if _, c := ver(t, sano); !strings.Contains(c, "Todavia no has subido") {
		t.Errorf("con el almacen sano y vacio la pantalla no dice que no has subido "+
			"nada:\n%s", c)
	}
}

// ---------------------------------------------------------------------------
// El descargo, que es la pantalla entera
// ---------------------------------------------------------------------------

// LA PANTALLA NO DICE QUE CUMPLAS NADA, y lo dice con esas palabras.
//
// Con su CONTROL POSITIVO: el descargo tiene que salir en la pantalla que ENSENA
// hallazgos, no solo estar en el catalogo. Una rama de descargo que ninguna
// entrada recorre es una rama que no existe (M47).
func TestLaPantallaDeHallazgosNoDiceQueCumplasNada(t *testing.T) {
	s := superficie(t)
	if codigo, cuerpo := subir(t, s, "politica.pdf", []byte("un parrafo")); codigo != http.StatusSeeOther {
		t.Fatalf("la subida contesta %d: %s", codigo, avisoDe(cuerpo))
	}
	_, cuerpo := ver(t, s)
	if !strings.Contains(cuerpo, "NO dice que cumplas nada") {
		t.Errorf("la pantalla que ensena hallazgos no trae el descargo.\n"+
			"  Sin esa frase, una lista de parrafos debajo de una lista de obligaciones se "+
			"lee como una lista de cosas cumplidas, que es el unico error que un producto "+
			"de cumplimiento no puede cometer.\n%s", cuerpo)
	}
	// Y NO HAY NINGUNA PALABRA DE VEREDICTO en la pantalla. Se comprueba sobre
	// el HTML servido y no sobre el tipo, que es otra puerta: aqui lo que se
	// vigila es lo que LEE una persona.
	for _, prohibida := range []string{"cumple", "conforme", "aprobado", "satisfecho"} {
		if strings.Contains(strings.ToLower(cuerpo), prohibida) &&
			!strings.Contains(strings.ToLower(cuerpo), "no dice que cumplas") {
			t.Errorf("la pantalla usa la palabra %q, que es un juicio", prohibida)
		}
	}
}

// LOS DOS SILENCIOS QUE NO SON EL MISMO, Y NINGUNO ES «TU DOCUMENTO NO DICE
// NADA».
//
// Son las dos ramas en las que plazum no ha encontrado algo, y las dos tienen la
// misma tentacion: contarlas como una carencia del cliente.
//
//	sin_contenido  el fichero se leyo y no rindio ni un parrafo citable
//	ninguno        hay documentos indexados y la busqueda no ha casado ninguna
//	               obligacion
//
// La primera dice algo de la LECTURA y la segunda del BUSCADOR, y las dos
// hablan de plazum. Un cliente que lea «tu politica no cubre nada» de una
// busqueda por palabras que no entiende sinonimos se lleva una acusacion falsa,
// y quien la lea deja de creerse el resto de la pantalla.
func TestNoHaberEncontradoNadaNoSeCuentaComoQueNoHayNada(t *testing.T) {
	// 1. EL FICHERO QUE SE LEE Y NO RINDE NI UN PARRAFO.
	aCero := nuevoAlmacenFalso()
	aCero.sinFragmentos = true
	s := superficie(t, conAlmacen(aCero))
	codigo, cuerpo := subir(t, s, "escaneo.pdf", []byte("   "))
	if codigo != http.StatusUnprocessableEntity {
		t.Fatalf("un documento sin ni un fragmento contesta %d y tenia que rechazarse "+
			"con 422", codigo)
	}
	if got := avisoDe(cuerpo); got != "documentos.subir.sin_contenido" {
		t.Fatalf("el rechazo es %q y tenia que ser documentos.subir.sin_contenido", got)
	}
	if !strings.Contains(cuerpo, "NO dice que tu documento no hable de nada") {
		t.Errorf("el rechazo por cero fragmentos NO trae su descargo.\n"+
			"  Sin el, «no ha entrado ni un parrafo» se lee como «tu documento esta "+
			"vacio», y la diferencia es de quien es la culpa.\n%s", cuerpo)
	}

	// 2. HAY DOCUMENTOS Y LA BUSQUEDA NO CASA NINGUNA OBLIGACION.
	aSin := nuevoAlmacenFalso()
	aSin.docs["ciso@ejemplo"] = []ResumenDeDocumento{
		{Fichero: "politica.pdf", Fragmentos: 40, Paginas: 12},
	}
	_, cuerpo = ver(t, superficie(t, conAlmacen(aSin)))
	if !strings.Contains(cuerpo, "NO dice que tus documentos no lo traten") {
		t.Errorf("con documentos y cero hallazgos la pantalla NO trae su descargo.\n"+
			"  Hoy la busqueda casa palabras y no ideas parecidas, asi que cero hallazgos "+
			"dice mucho mas del buscador que de la politica del cliente.\n%s", cuerpo)
	}
	// Y NO SE PINTA EL ESTADO VACIO, que diria lo contrario: aqui SI hay
	// documentos subidos y mandar a subirlos otra vez es no haber mirado.
	if strings.Contains(cuerpo, "Todavia no has subido") {
		t.Error("con documentos subidos y cero hallazgos la pantalla dice que no has " +
			"subido nada")
	}

	// CONTROL POSITIVO DE LAS DOS: con un documento que SI rinde parrafos y SI
	// casa, no sale ninguna de las dos frases. Sin esta mitad, una pantalla que
	// las pintara siempre pasaria lo de arriba entero.
	sano := superficie(t)
	if codigo, c := subir(t, sano, "politica.pdf", []byte("un parrafo")); codigo != http.StatusSeeOther {
		t.Fatalf("el control positivo no sube: %d (%s)", codigo, avisoDe(c))
	}
	_, cuerpo = ver(t, sano)
	for _, prohibida := range []string{"NO dice que tus documentos no lo traten",
		"Todavia no has subido"} {
		if strings.Contains(cuerpo, prohibida) {
			t.Errorf("con hallazgos de verdad la pantalla sigue diciendo %q", prohibida)
		}
	}
}

// ---------------------------------------------------------------------------
// El contrato de claves
// ---------------------------------------------------------------------------

// LAS CLAVES QUE DECLARA ESTA SUPERFICIE SON EXACTAMENTE LAS QUE PIDE, en las
// dos direcciones: no puede quedarse corta (una clave sin traducir sale cruda en
// la pantalla de un cliente) ni sobrarle (una clave traducida que nadie pide es
// texto muerto que el inventario da por bueno).
func TestLasClavesDeclaradasSonLasQueLaSuperficiePide(t *testing.T) {
	pedidas := map[string]bool{}
	espia := &catalogoEspia{real: cat(t), pedidas: pedidas}

	// TODOS LOS ESTADOS DE LA PANTALLA, uno por uno. El que falte deja sus
	// claves como «declaradas y nadie las pide», que es literalmente cierto.
	base := func(ajustes ...ajuste) *Superficie {
		o := Opciones{
			Catalogo: espia, Base: BasePorDefecto, Estatico: "/estatico",
			Almacen: nuevoAlmacenFalso(),
			Quien:   func(*http.Request) string { return "ciso" },
			Tokens:  func(*http.Request) (string, error) { return "tok", nil },
			Pasos:   camino.Canonico(),
		}
		for _, a := range ajustes {
			a(&o)
		}
		s, err := Nuevo(o)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	ver(t, base())           // vacio
	ver(t, base(sinSesion))  // sin sesion
	ver(t, base(sinAlmacen)) // sin almacen
	ver(t, base(sinToken))   // sin token
	conDocs := base()        // con documentos y hallazgos
	subir(t, conDocs, "p.pdf", []byte("un parrafo"))
	ver(t, conDocs)

	// Un documento truncado y con nombre enganoso, y otro sin paginas: tres
	// ramas de la lista que el camino normal no recorre.
	aRaro := nuevoAlmacenFalso()
	aRaro.docs["ciso"] = []ResumenDeDocumento{
		{Fichero: "a.pdf", Fragmentos: 2, Paginas: 3, Truncado: true, NombreEnganoso: true},
		{Fichero: "b.txt", Fragmentos: 1},
	}
	ver(t, base(conAlmacen(aRaro)))

	// Un almacen con documentos y CERO hallazgos.
	aSin := nuevoAlmacenFalso()
	aSin.docs["ciso"] = []ResumenDeDocumento{{Fichero: "c.txt", Fragmentos: 1}}
	ver(t, base(conAlmacen(aSin)))

	// UN HALLAZGO SIN PAGINA. Un .txt no tiene paginas, asi que el sitio del
	// parrafo se dice por numero de fragmento. El almacen falso siempre pone
	// pagina, asi que esta rama no la recorre el camino normal, y una rama que
	// ninguna entrada alcanza es una rama que no existe: su clave saldria cruda
	// el dia que alguien suba un .txt.
	aSinPag := nuevoAlmacenFalso()
	aSinPag.docs["ciso"] = []ResumenDeDocumento{{Fichero: "d.txt", Fragmentos: 5}}
	aSinPag.halla["ciso"] = []Hallazgo{{
		Obligacion: "demo.o.dos", Titulo: "Registro de incidentes",
		Marco: "urn:demo:m1", Parrafo: "Los incidentes se registran en el dia.",
		Pagina: 0, Fragmento: 4, Documento: "d.txt",
	}}
	ver(t, base(conAlmacen(aSinPag)))

	// El almacen roto.
	aRoto := nuevoAlmacenFalso()
	aRoto.errLeer = errors.New("roto")
	ver(t, base(conAlmacen(aRoto)))

	// Y todos los rechazos de la subida.
	for _, err := range []error{
		nil,
		ingesta.ErrSinTextoInterpretable, ingesta.ErrFormatoNoReconocido,
		ingesta.ErrPDFCifrado, ingesta.ErrDemasiadoGrande,
		errors.New("desconocido"),
	} {
		aErr := nuevoAlmacenFalso()
		if err != nil {
			aErr.err = fmt.Errorf("x: %w", err)
		}
		s := base(conAlmacen(aErr))
		subir(t, s, "x.pdf", nil)          // falta_fichero
		subir(t, s, "x.pdf", []byte{})     // vacio
		subir(t, s, "x.pdf", []byte("ok")) // el error de la vuelta
	}
	// EL DOCUMENTO QUE SE LEE Y NO DA NI UN FRAGMENTO: el fichero estaba y
	// estaba vacio de verdad.
	aCero := nuevoAlmacenFalso()
	aCero.sinFragmentos = true
	subir(t, base(conAlmacen(aCero)), "vacio.txt", []byte(" "))

	// LA FICHA (pieza 7), con sus DOS estados y sus cuatro campos.
	//
	// Los cuatro campos hacen falta uno a uno: el camino normal solo propone
	// fecha, asi que sin esto las claves de alcance, firmante y caducidad se
	// quedarian declaradas y sin pedir, o sea traducidas a dos idiomas y sin que
	// nadie sepa si salen bien.
	aFicha := nuevoAlmacenFalso()
	aFicha.docs["ciso"] = []ResumenDeDocumento{{Fichero: "p.pdf", Fragmentos: 2, Paginas: 1}}
	for _, campo := range []string{
		"documentos.ficha.campo.fecha", "documentos.ficha.campo.alcance",
		"documentos.ficha.campo.firmante", "documentos.ficha.campo.caducidad",
	} {
		aFicha.ficha["ciso"] = append(aFicha.ficha["ciso"], PropuestaDeFicha{
			Campo: campo, Valor: "un valor", Parrafo: "de aqui sale", Documento: "p.pdf",
			Pagina: 1, Huella: "hh",
		})
	}
	aFicha.aceptados["ciso"] = []CampoAceptado{{
		Campo: "documentos.ficha.campo.fecha", Valor: "2026-01-15", Quien: "ciso",
		Cuando: "2026-09-11T09:00:00Z", Documento: "p.pdf", Parrafo: "Fecha: 2026-01-15",
	}}
	ver(t, base(conAlmacen(aFicha)))

	// Y EL DOCUMENTO DEL QUE NO SALE NINGUN CAMPO: no es lo mismo que no haber
	// subido nada, y por eso tiene su propia frase.
	aSinFicha := nuevoAlmacenFalso()
	aSinFicha.sinFicha = true
	sf := base(conAlmacen(aSinFicha))
	subir(t, sf, "sinficha.pdf", []byte("un parrafo sin cabecera"))
	ver(t, sf)

	// LOS RECHAZOS DE LA CONFIRMACION, uno por clase.
	aAcep := nuevoAlmacenFalso()
	sAcep := base(conAlmacen(aAcep))
	subir(t, sAcep, "p.pdf", []byte("un parrafo"))
	aceptar(t, sAcep, "", "", "")                                   // falta_campo
	aceptar(t, sAcep, "documentos.ficha.campo.fecha", "otra", "hh") // no_casa
	fs, _ := aAcep.Ficha(context.Background(), "ciso")
	if len(fs) == 1 {
		aRoto := nuevoAlmacenFalso()
		aRoto.docs["ciso"] = aAcep.docs["ciso"]
		aRoto.ficha["ciso"] = fs
		aRoto.err = errors.New("no se guarda")
		aceptar(t, base(conAlmacen(aRoto)), fs[0].Campo, fs[0].Valor, fs[0].Huella)
	}
	// Y el cuerpo que no se puede parsear como formulario.
	rotoF := base()
	rf := httptest.NewRequest(http.MethodPost, BasePorDefecto+RutaDeAceptar,
		strings.NewReader("%zz"))
	rf.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rotoF.ServeHTTP(httptest.NewRecorder(), rf)

	// EL MULTIPART ROTO: el cuerpo llega y no se puede parsear. Es la rama del
	// error de parseo, la que NO ensena el error tal cual porque trae rutas de
	// ficheros temporales.
	roto := base()
	r := httptest.NewRequest(http.MethodPost, BasePorDefecto+RutaDeSubir,
		strings.NewReader("esto no es un multipart"))
	r.Header.Set("Content-Type", "multipart/form-data; boundary=noexiste")
	roto.ServeHTTP(httptest.NewRecorder(), r)

	// Y LAS DOS QUE NO SE PUEDEN PROVOCAR DESDE FUERA, con su motivo escrito.
	//
	// Se apuntan a mano en vez de quitarlas del contrato: una guarda que existe
	// en el codigo y cuya clave no esta traducida sale cruda el dia que se
	// dispare, y ese dia es justo el peor para descubrirlo.
	//
	//	sin_almacen   la ruta mutante NO SE REGISTRA sin almacen, asi que el
	//	              handler no se alcanza. La guarda se queda igual: una ruta
	//	              que depende de que su constructor no se equivoque no esta
	//	              protegida, esta de suerte.
	//	error.render  solo sale si html/template falla al ejecutar, que con una
	//	              plantilla embebida y compilada al construir no depende de
	//	              ninguna entrada.
	pedidas["documentos.subir.sin_almacen"] = true
	pedidas["documentos.error.render"] = true
	// El tope: la unica que no sale de las de arriba.
	pedidas["documentos.subir.demasiado_grande"] = true

	// LAS CLAVES DEL CAMINO NO SON DE ESTA SUPERFICIE. Llegan como DATO desde
	// camino.Canonico() y las declara el camino, que es quien las escribe;
	// publicarlas aqui seria una segunda copia del camino dentro del contrato de
	// una superficie que solo lo pinta. Se eximen por su DECLARACION y no por un
	// prefijo: un prefijo eximiria tambien una clave del camino que nadie
	// hubiera declarado en ningun sitio, que es el caso que hay que cazar.
	delCamino := map[string]bool{}
	for _, k := range camino.ClavesDeCatalogo() {
		delCamino[k] = true
	}
	if len(delCamino) < 6 {
		t.Fatalf("el camino declara %d claves y son muchas mas: esta exencion estaria "+
			"eximiendo el vacio", len(delCamino))
	}
	declaradas := map[string]bool{}
	for _, k := range ClavesDeCatalogo() {
		declaradas[k] = true
	}
	// LA EXENCION ES PRECISA: solo se exime la clave que declara el camino y que
	// esta superficie NO declara. Las del marco compartido (`ui.marca`,
	// `ui.saltar`, el descargo del pie) las declaran los dos, y eximirlas sin
	// mirar dejaria tres claves de esta lista sin nadie que las pida, o sea
	// texto muerto con cara de contrato.
	for k := range delCamino {
		if !declaradas[k] {
			delete(pedidas, k)
		}
	}
	var faltan, sobran []string
	for k := range pedidas {
		if !declaradas[k] {
			faltan = append(faltan, k)
		}
	}
	for k := range declaradas {
		if !pedidas[k] {
			sobran = append(sobran, k)
		}
	}
	sort.Strings(faltan)
	sort.Strings(sobran)
	if len(faltan) > 0 {
		t.Errorf("la superficie pide estas claves y ClavesDeCatalogo() no las declara: %v\n"+
			"  Esas saldrian crudas en la pantalla de un cliente", faltan)
	}
	if len(sobran) > 0 {
		t.Errorf("ClavesDeCatalogo() declara estas y ningun estado las pide: %v\n"+
			"  O sobran, o hay un estado que este test no recorre. Quitarlas sin mirar es "+
			"como se pierde el aviso que hara falta el dia que esa rama se recorra", sobran)
	}
}

// catalogoEspia anota que claves se piden y delega en el catalogo real.
type catalogoEspia struct {
	real    *catalogo.Catalogo
	pedidas map[string]bool
}

func (c *catalogoEspia) Traducir(idioma, clave string, args ...any) string {
	c.pedidas[clave] = true
	return c.real.Traducir(idioma, clave, args...)
}
func (c *catalogoEspia) Idiomas() []string           { return c.real.Idiomas() }
func (c *catalogoEspia) Faltantes(i string) []string { return c.real.Faltantes(i) }

// ---------------------------------------------------------------------------
// El montaje
// ---------------------------------------------------------------------------

// TODA RUTA REGISTRADA SALE POR Patrones(), que es lo que hace que la puerta de
// CSRF de superficies/serve las enumere sin conocer este paquete.
func TestTodaRutaRegistradaSaleEnPatrones(t *testing.T) {
	s := superficie(t)
	ps := s.Patrones()
	if len(ps) != 3 {
		t.Fatalf("hay %d patrones (%v) y son tres: la pantalla, la subida y la "+
			"confirmacion de un campo de la ficha", len(ps), ps)
	}
	quiero := map[string]bool{
		"GET " + BasePorDefecto + "/{$}":         true,
		"POST " + BasePorDefecto + RutaDeSubir:   true,
		"POST " + BasePorDefecto + RutaDeAceptar: true,
	}
	for _, p := range ps {
		if !quiero[p] {
			t.Errorf("patron inesperado: %q", p)
		}
	}
}

// MEDIO ENLACE AL CAMINO SE RECHAZA AL CONSTRUIR. Las dos mitades o ninguna: una
// direccion sin rotulo da un ancla vacia y un rotulo sin direccion da texto que
// no lleva a ningun sitio.
func TestMedioEnlaceAlCaminoSeRechazaAlConstruir(t *testing.T) {
	for _, c := range []struct {
		que         string
		ruta, clave string
		falla       bool
	}{
		{que: "las dos", ruta: "/camino/", clave: "camino.titulo"},
		{que: "ninguna"},
		{que: "solo la ruta", ruta: "/camino/", falla: true},
		{que: "solo el rotulo", clave: "camino.titulo", falla: true},
		{que: "otro anfitrion", ruta: "//otro.example/", clave: "camino.titulo", falla: true},
	} {
		_, err := Nuevo(Opciones{
			Catalogo: cat(t), Base: BasePorDefecto,
			CaminoRuta: c.ruta, CaminoClave: c.clave,
		})
		if c.falla && err == nil {
			t.Errorf("%s: se ha construido y tenia que fallar", c.que)
		}
		if !c.falla && err != nil {
			t.Errorf("%s: %v", c.que, err)
		}
	}
}

// SIN CATALOGO NO SE CONSTRUYE. Es el valor cero de lo obligatorio.
func TestSinCatalogoNoSeConstruye(t *testing.T) {
	if _, err := Nuevo(Opciones{Base: BasePorDefecto}); !errors.Is(err, ErrSinCatalogo) {
		t.Errorf("sin catalogo el error es %v y tenia que ser ErrSinCatalogo", err)
	}
}
