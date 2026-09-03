package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/marcosmatalab/plazum/nucleo/accesos"
	"github.com/marcosmatalab/plazum/nucleo/censo"
	"github.com/marcosmatalab/plazum/nucleo/ledger"
	"github.com/marcosmatalab/plazum/superficies/camino"
	"github.com/marcosmatalab/plazum/superficies/serve"
	"github.com/marcosmatalab/plazum/superficies/uar"
)

// EL ADAPTADOR QUE LE DA UNA CAMPANA A LA PANTALLA.
//
// La superficie no sabe de ficheros ni de ledgers, y esto es lo que los conoce.
// Reconstruye la campana en CADA peticion desde el fichero mas el ledger, que es
// consecuencia directa de no guardar el censo: no hay estado en memoria que se
// quede viejo, y dos personas revisando a la vez ven lo mismo.
//
// EL COSTE, DICHO: releer y reconstruir por peticion es trabajo. Con un censo de
// cinco mil filas son unos milisegundos y no compensa cachear todavia; el dia
// que compense, lo que hay que cachear es la LECTURA del fichero por su hash, no
// la campana, porque la campana cambia con cada hecho.
type campanaEnFichero struct {
	fichero string
	ledger  string
	id      string
}

func (c campanaEnFichero) Abierta() (*accesos.Campana, error) {
	l, err := leerLedger(c.ledger)
	if err != nil {
		return nil, err
	}
	ap, subioLo, cuando, ok := datosDeApertura(l, c.id)
	if !ok {
		return nil, fmt.Errorf("en %s no consta la apertura de la campana %q.\n"+
			"  Arreglo: abrirla con `plazum accesos ver --fichero %s --ledger %s --campana %s`",
			c.ledger, c.id, c.fichero, c.ledger, c.id)
	}
	datos, err := os.ReadFile(c.fichero) // #nosec G304 -- la ruta la da el operador al arrancar el servidor
	if err != nil {
		return nil, fmt.Errorf("no se puede leer %s: %w", c.fichero, err)
	}
	ins, err := censo.Tomar(datos, censo.Opciones{
		Sistema: ap.Sistema, Fuente: ap.Fuente, Quien: subioLo, Tomada: cuando,
		Retencion: ap.Retencion, Columnas: censo.ColumnasHabituales(),
	})
	if err != nil {
		return nil, err
	}
	return accesos.Reconstruir(c.id, ins, l, nil)
}

func (c campanaEnFichero) Anotar(e ledger.Entrada) error {
	l, err := leerLedger(c.ledger)
	if err != nil {
		return err
	}
	return anotarEnLedger(c.ledger, l, e)
}

// enumerador es quien sabe decir sus patrones. Lo implementan las dos
// superficies, y es como sus rutas entran en la puerta de CSRF de serve.
type enumerador interface{ Patrones() []string }

// compuesto es la union de dos superficies QUE ADEMAS SIGUE SABIENDO DECIR SUS
// PATRONES.
//
// AQUI ESTUVO EL AGUJERO, y lo encontro mirar la puerta antes de darla por
// buena. `serve.Servidor.Rutas()` hace un type switch sobre Config.App: si
// implementa EnumeradorDePatrones, sus rutas entran en la puerta que ENUMERA y
// le manda una peticion mutante a cada una. Un `http.ServeMux` pelado no
// implementa nada de eso, asi que componer con uno habria dejado fuera de esa
// puerta **las rutas mutantes nuevas Y las de pantallas**, que hasta hoy si
// entraban. Sin un solo test en rojo: el CSRF por metodo las habria seguido
// cubriendo, y la puerta que enumera habria pasado a mirar cero rutas de
// aplicacion y a decir que todo bien.
//
// Es la familia de siempre: una guarda que deja de mirar no se pone roja, se
// queda callada.
type compuesto struct {
	http.Handler
	patrones []string
}

func (c compuesto) Patrones() []string { return append([]string(nil), c.patrones...) }

// montaje es una superficie con el prefijo bajo el que cuelga.
type montaje struct {
	prefijo string
	h       http.Handler
}

// montarSuperficies compone las superficies que sirve `plazum serve`.
//
// TODAS VIVEN BAJO LA MISMA RAIZ y no se pisan: pantallas se queda con "/" y
// cada una de las demas con su prefijo. El ServeMux de Go elige el patron mas
// especifico, asi que el orden de registro no decide nada, que es justo lo que
// se quiere de una composicion.
//
// UN MONTAJE NIL SE SALTA, no se registra vacio: `mux.Handle` con un handler nil
// entra en panico al primer visitante, y un servidor que revienta al abrirlo es
// peor que una pantalla que no esta.
func montarSuperficies(app http.Handler, bajo ...montaje) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/", app)
	handlers := []http.Handler{app}
	for _, m := range bajo {
		if m.h == nil || m.prefijo == "" {
			continue
		}
		mux.Handle(m.prefijo, m.h)
		handlers = append(handlers, m.h)
	}
	if len(handlers) == 1 {
		return app // no hay nada montado debajo: la composicion no aporta nada
	}

	var patrones []string
	for _, h := range handlers {
		if e, ok := h.(enumerador); ok {
			patrones = append(patrones, e.Patrones()...)
		}
	}
	return compuesto{Handler: mux, patrones: patrones}
}

// construirUAR arma la superficie. Devuelve nil solo si falla; SIN FICHEROS
// CONFIGURADOS DEVUELVE UNA SUPERFICIE IGUALMENTE, con la fuente vacia.
//
// Es deliberado y es la puerta D11-b: la pantalla tiene que existir para poder
// contarle a quien llegue como se configura. Una pantalla que desaparece cuando
// no hay datos deja al operador sin saber que existia, que es el mismo fallo que
// `plazum serve` tuvo con su propia orden.
func construirUAR(o opcionesUAR) (*uar.Superficie, error) {
	if o.Catalogo == nil {
		return nil, errors.New("uar: falta el catalogo")
	}
	var fuente uar.Campanas
	if strings.TrimSpace(o.Fichero) != "" && strings.TrimSpace(o.Ledger) != "" {
		fuente = campanaEnFichero{fichero: o.Fichero, ledger: o.Ledger, id: o.Campana}
	}
	base, _ := camino.RutaDe("uar")
	return uar.Nuevo(uar.Opciones{
		Fuente:   fuente,
		Catalogo: o.Catalogo,
		Base:     strings.TrimSuffix(base, "/"),
		Estatico: "/estatico",
		Tokens:   o.Tokens,
		Quien:    o.Quien,
		// La vuelta al camino guiado. Sin esto, esta pantalla es un callejon:
		// no tiene menu y nadie enlaza a ningun otro sitio desde aqui.
		CaminoRuta:  camino.BasePorDefecto + "/",
		CaminoClave: camino.ClaveTitulo,
	})
}

type opcionesUAR struct {
	Fichero  string
	Ledger   string
	Campana  string
	Catalogo interface {
		Traducir(idioma, clave string, args ...any) string
		Idiomas() []string
		Faltantes(idioma string) []string
	}
	Tokens func(*http.Request) (string, error)
	Quien  func(*http.Request) string
}

// tokensDeLaSesion emite el token CSRF de la peticion.
//
// Va aqui y no dentro de la superficie porque el nombre de la cookie depende de
// si hay TLS, que es una decision de despliegue, y el almacen de sesiones lo
// construye esta orden. La superficie solo sabe que le dan un token o no se lo
// dan, y si no se lo dan NO PINTA BOTONES.
func tokensDeLaSesion(ses *serve.Sesion, insegura bool) func(*http.Request) (string, error) {
	nombre := serve.CookieSesion
	if insegura {
		nombre = serve.CookieSesionInsegura
	}
	return func(r *http.Request) (string, error) {
		c, err := r.Cookie(nombre)
		if err != nil || c.Value == "" {
			return "", errors.New("sin cookie de sesion")
		}
		return ses.TokenCSRF(r.Context(), c.Value)
	}
}

// quienOpera saca el sujeto que dejo el middleware de serve en el contexto.
//
// Si no hay sesion devuelve cadena vacia, y entonces la superficie no admite
// ninguna decision: un hecho sin autor no es un hecho, y rellenarlo con
// "anonimo" seria firmar una campana en nombre de nadie.
//
// # UNA SESION ANONIMA TAMPOCO ES UN AUTOR, y esto era un fallo de verdad
//
// `serve.SujetoDe` devuelve dos cosas distintas que aqui se estaban leyendo
// como una: «no hay sesion» (cadena vacia) y «hay sesion Y NO HA ENTRADO
// NADIE», que es la sesion efimera con sujeto `serve.SujetoAnonimo`. Esa
// segunda la reparte el propio producto: **basta abrir `/entrar`**, porque el
// formulario necesita una sesion para poder emitir su token CSRF.
//
// El godoc de `SujetoDe` lo dice con esas palabras («quien monte pantallas
// tiene que comprobar ademas EsAnonimo») y esta funcion no lo comprobaba. Medido
// el 04-09-2026 sobre el binario, con el guardado del alcance ya cableado:
//
//	GET /alcance sin cookie ............. 200
//	GET /entrar (da cookie anonima) ..... 200
//	GET /alcance con esa cookie ......... 500
//	POST /alcance con esa cookie ........ 500
//
// O sea que **quien intentaba entrar y luego miraba la entrevista se encontraba
// una pagina rota**, porque el sujeto «anonimo:sin-autenticar» llegaba al
// almacen de alcances y `usuarios.NormalizarUsuario` lo rechaza (los dos puntos
// estan prohibidos, y estan prohibidos justamente para esto). La guarda de
// abajo salvaba de escribir en un cajon comun y no salvaba de la pagina rota.
//
// La comprobacion va AQUI y no en las superficies porque este es el unico
// fichero que conoce `superficies/serve`: las superficies solo saben que un
// sujeto vacio es «no hay autor».
//
// Y alcanza a las dos superficies que la usan: sin esto,
// `superficies/uar` habria acabado anotando una decision en el ledger a nombre
// de «anonimo:sin-autenticar», que es peor que la pagina rota porque no se
// deshace.
func quienOpera(r *http.Request) string {
	s, ok := serve.SujetoDe(r)
	if !ok || serve.EsAnonimo(s) {
		return ""
	}
	return s
}
