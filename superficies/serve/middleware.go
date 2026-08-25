package serve

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"plazum/puertos"
)

// Los middlewares de seguridad de la superficie web.
//
// Por que son middleware y no un puerto. Se propuso un puerto `Seguridad` y se
// retiro (consta en docs/puertos-propuestas.md): una capa de seguridad
// enchufable no encaja en un producto cuya tesis es que el receptor no se fia,
// porque lo enchufable se desenchufa. La puerta de esto no es una interfaz, es
// CI arrancando el binario y mirando la respuesta real
// (.github/workflows/etapa2-seguridad-web.yml).

// Nombres de las piezas que el navegador tiene que conocer. Estan aqui como
// constantes porque las plantillas del frente de pantallas las necesitan y no
// puede haber dos verdades sobre como se llama el campo del formulario.
const (
	// CookieSesion es el nombre de la cookie de sesion. El prefijo __Host- lo
	// impone el navegador: obliga a Secure, a Path=/ y a que NO haya Domain,
	// con lo que un subdominio comprometido (o cualquiera que pueda escribir
	// cookies para el dominio padre) no puede plantar una cookie de sesion.
	// Es proteccion que da el cliente gratis y que se pierde en cuanto alguien
	// renombra esto sin saber por que.
	CookieSesion = "__Host-plazum_sesion"
	// CookieSesionInsegura es el nombre que se usa cuando el operador ha
	// pedido explicitamente cookies sin Secure (ver Config.CookieInsegura).
	// Tiene que ser OTRO nombre: el prefijo __Host- exige Secure y el navegador
	// tiraria la cookie sin decir nada, que es un fallo mudo.
	CookieSesionInsegura = "plazum_sesion"
	// CabeceraCSRF es de donde lee el token una peticion de htmx.
	CabeceraCSRF = "X-CSRF-Token"
	// CampoCSRF es el name del input oculto de un formulario normal.
	CampoCSRF = "csrf"
)

// metodosSeguros es la lista de metodos que NO mutan estado y por tanto no
// exigen token CSRF.
//
// Es una LISTA BLANCA y eso es la mitad del diseno. Con una lista negra de
// metodos mutantes, el dia que alguien anada un handler que responde a un
// metodo que no esta en la lista, ese handler queda sin proteger y nadie se
// entera. Con lista blanca, lo que no se ha pensado se exige.
var metodosSeguros = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodOptions: true,
}

// EsMetodoMutante dice si un metodo exige token CSRF. Todo lo que no sea
// GET, HEAD u OPTIONS lo exige, incluido lo que todavia no existe.
func EsMetodoMutante(metodo string) bool {
	return !metodosSeguros[strings.ToUpper(strings.TrimSpace(metodo))]
}

// --- cabeceras ---

// Cabeceras son las cabeceras de seguridad que se ponen en TODA respuesta.
type Cabeceras struct {
	// CSP es la Content-Security-Policy. Vacio usa CSPPorDefecto.
	CSP string
	// HSTS es la Strict-Transport-Security. Vacio usa HSTSPorDefecto.
	HSTS string
}

// Los valores por defecto. Se pueden sustituir, no debilitar sin decirlo:
// validarCSP rechaza las relajaciones tipicas.
const (
	// CSPPorDefecto no admite nada inline. Es lo que permite que una
	// inyeccion en un campo de texto del alcance no se convierta en script.
	CSPPorDefecto = "default-src 'self'; script-src 'self'; style-src 'self'; " +
		"img-src 'self' data:; font-src 'self'; connect-src 'self'; " +
		"object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'"

	// HSTSPorDefecto son dos anos, con subdominios. Sin preload: meterse en la
	// lista de precarga de los navegadores es irreversible en la practica y esa
	// decision es del operador, no nuestra.
	HSTSPorDefecto = "max-age=63072000; includeSubDomains"
)

// Envolver pone las cabeceras ANTES de llamar al handler, para que un handler
// que sepa lo que hace (el de estaticos con su Cache-Control) pueda cambiarlas,
// y para que esten puestas aunque el handler escriba y salga.
func (c Cabeceras) Envolver(h http.Handler) http.Handler {
	csp := c.CSP
	if csp == "" {
		csp = CSPPorDefecto
	}
	hsts := c.HSTS
	if hsts == "" {
		hsts = HSTSPorDefecto
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cab := w.Header()
		cab.Set("Content-Security-Policy", csp)
		// HSTS se manda SIEMPRE, tambien sobre http. RFC 6797 dice que no se
		// mande fuera de transporte seguro, y aqui se hace a sabiendas: el
		// navegador la ignora sobre http (misma RFC, 8.1), asi que no cuesta
		// nada, y el operador que mas la necesita es justo el que puso un
		// proxy con TLS delante y no se lo dijo a plazum. Una proteccion que
		// depende de acordarse de una opcion es una proteccion que no esta.
		cab.Set("Strict-Transport-Security", hsts)
		cab.Set("X-Frame-Options", "DENY")
		cab.Set("X-Content-Type-Options", "nosniff")
		cab.Set("Referrer-Policy", "no-referrer")
		cab.Set("Cross-Origin-Opener-Policy", "same-origin")
		cab.Set("Cross-Origin-Resource-Policy", "same-origin")
		cab.Set("X-Permitted-Cross-Domain-Policies", "none")
		cab.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
		// Por defecto nada se cachea: las pantallas llevan estado de
		// cumplimiento de la organizacion y un proxy intermedio no tiene por
		// que guardarlas. El handler de estaticos lo sobrescribe.
		cab.Set("Cache-Control", "no-store")
		h.ServeHTTP(w, r)
	})
}

// fuentesQueNoRestringen son valores de origen que, puestos en una directiva,
// la dejan sin efecto. Se comparan como TOKEN COMPLETO y no como subcadena:
// buscar "*" dentro de la politica daria falso positivo con cualquier comodin
// legitimo de un nombre de dominio, y buscar "data:" chocaria con el img-src
// data: que si es razonable.
var fuentesQueNoRestringen = map[string]bool{
	"'unsafe-inline'": true,
	"'unsafe-eval'":   true,
	"'unsafe-hashes'": true,
	"*":               true,
	"http:":           true,
	"https:":          true,
	"data:":           false, // solo se rechaza en script-src, ver abajo
}

// validarCSP rechaza una politica debilitada. Se llama al construir el
// servidor: si el operador sustituye la CSP por una que no protege, el servidor
// no arranca en vez de arrancar aparentando que protege.
func validarCSP(csp string) error {
	bajo := strings.ToLower(csp)
	if !strings.Contains(bajo, "default-src") && !strings.Contains(bajo, "script-src") {
		return errors.New("la CSP no declara ni default-src ni script-src, asi que no " +
			"restringe de donde se cargan los scripts. Arreglo: parte de serve.CSPPorDefecto")
	}
	if !strings.Contains(bajo, "frame-ancestors") {
		return errors.New("la CSP no declara frame-ancestors, asi que la interfaz se puede " +
			"meter en un iframe ajeno. Arreglo: anade frame-ancestors 'none'")
	}
	for _, directiva := range strings.Split(bajo, ";") {
		campos := strings.Fields(strings.TrimSpace(directiva))
		if len(campos) == 0 {
			continue
		}
		nombre, fuentes := campos[0], campos[1:]
		for _, f := range fuentes {
			malo := fuentesQueNoRestringen[f]
			if f == "data:" && (nombre == "script-src" || nombre == "default-src") {
				malo = true
			}
			if malo {
				return fmt.Errorf("la CSP pone %q en %s, que deja esa directiva sin "+
					"efecto. Arreglo: quitalo; si de verdad hace falta, pon "+
					"Config.CSPDebilitadaAProposito y deja escrito por que", f, nombre)
			}
		}
	}
	return nil
}

// --- limites de peticion ---

// limitarCuerpo corta el cuerpo de la peticion. Sin esto, un POST de un
// gigabyte se come la memoria del proceso antes de que ningun handler decida
// nada.
func limitarCuerpo(max int64, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, max)
		}
		h.ServeHTTP(w, r)
	})
}

// metodosProhibidos son los que no se atienden nunca. TRACE devuelve la
// peticion entera, cabeceras incluidas, que es como se roba una cookie
// HttpOnly (Cross-Site Tracing). CONNECT convertiria esto en un proxy abierto.
var metodosProhibidos = map[string]bool{
	http.MethodTrace:   true,
	http.MethodConnect: true,
}

func rechazarMetodosProhibidos(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if metodosProhibidos[strings.ToUpper(r.Method)] {
			responder(w, http.StatusMethodNotAllowed,
				"metodo no atendido. TRACE devolveria la peticion entera al cliente, "+
					"cookies incluidas, y CONNECT convertiria plazum en un proxy abierto.")
			return
		}
		h.ServeHTTP(w, r)
	})
}

// hostPermitido rechaza las peticiones cuya cabecera Host no esta en la lista.
//
// La cabecera Host la escribe el cliente. Si alguna vez se construye una URL
// absoluta con ella (un enlace en un correo de escalado, por ejemplo), un Host
// falsificado convierte ese correo en un enlace al servidor del atacante. Aqui
// se corta en la puerta; y ademas, en este paquete no se construyen URL
// absolutas a partir de r.Host, que es la otra mitad del arreglo.
func hostPermitido(hosts []string, h http.Handler) http.Handler {
	if len(hosts) == 0 {
		return h
	}
	permitidos := make(map[string]bool, len(hosts))
	for _, x := range hosts {
		permitidos[strings.ToLower(strings.TrimSpace(soloHost(x)))] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !permitidos[strings.ToLower(soloHost(r.Host))] {
			// No se refleja el Host recibido en la respuesta: seria devolverle
			// al atacante su propia cadena.
			responder(w, http.StatusMisdirectedRequest,
				"esta instalacion de plazum no atiende peticiones dirigidas a ese nombre "+
					"de servidor. Si eres el operador, anade el nombre por el que se "+
					"entra a la lista de hosts permitidos de la configuracion de plazum; "+
					"esta explicado en docs/tls.md.")
			return
		}
		h.ServeHTTP(w, r)
	})
}

// soloHost quita el puerto y los corchetes de IPv6.
func soloHost(hp string) string {
	if h, _, err := net.SplitHostPort(hp); err == nil {
		return h
	}
	return strings.Trim(hp, "[]")
}

// recuperar convierte un panic de un handler en un 500 sin traza.
//
// net/http ya recupera el panic por conexion, pero cierra la conexion sin
// respuesta: el navegador ensena un error de red y el operador no sabe si es
// plazum o la wifi. Aqui se responde, y la traza NO viaja al cliente: seria
// entregarle al atacante el mapa del proceso.
func recuperar(registrar func(error), h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if p := recover(); p != nil {
				// http.ErrAbortHandler es el centinela con el que un handler
				// pide abortar la conexion a proposito. Se deja pasar.
				if err, ok := p.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(p)
				}
				if registrar != nil {
					registrar(fmt.Errorf("panic atendiendo %s: %v", r.URL.Path, p))
				}
				responder(w, http.StatusInternalServerError,
					"error interno. Se ha registrado en la salida del servidor; "+
						"no se envia el detalle al navegador a proposito.")
			}
		}()
		h.ServeHTTP(w, r)
	})
}

// --- CSRF ---

// ProtectorCSRF exige token en toda peticion mutante.
//
// El puerto Sesion emite el token y lo comprueba; esta pieza es la que lo
// EXIGE. Emitir bien un token que nadie exige no protege de nada, y esa mitad
// no se puede vigilar desde el puerto: se vigila con rutas_test.go, que
// enumera las rutas del enrutador y le manda una peticion mutante a cada una.
type ProtectorCSRF struct {
	Sesion puertos.Sesion
	// Cookie es el nombre de la cookie que lleva el identificador de sesion.
	Cookie string
}

// Envolver aplica la comprobacion a todo metodo mutante, sea cual sea la ruta.
// Por ruta seria una lista que se desincroniza; por metodo, lo que no se ha
// pensado queda exigido.
func (p ProtectorCSRF) Envolver(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !EsMetodoMutante(r.Method) {
			h.ServeHTTP(w, r)
			return
		}
		if p.Sesion == nil {
			// Sin almacen de sesiones no se puede comprobar nada, asi que no
			// se atiende NADA que mute. Fallar abierto aqui seria construir un
			// servidor sin CSRF por olvidar un campo.
			responder(w, http.StatusInternalServerError,
				"este servidor se construyo sin almacen de sesiones, asi que no puede "+
					"comprobar el token CSRF y no atiende peticiones que cambien estado.")
			return
		}
		if err := p.comprobar(r); err != nil {
			responder(w, http.StatusForbidden, err.Error())
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (p ProtectorCSRF) comprobar(r *http.Request) error {
	// 1. Origen. Es defensa en profundidad por delante del token: si el
	//    navegador dice de donde viene y no es de aqui, sobra mirar nada mas.
	if err := origenAceptable(r); err != nil {
		return err
	}
	// 2. Sesion. Sin cookie no hay con que atar el token, y un token que no
	//    esta atado a una sesion no es un token CSRF.
	c, err := r.Cookie(p.Cookie)
	if err != nil || c.Value == "" {
		return errors.New("peticion mutante sin cookie de sesion. Un token CSRF que no " +
			"esta atado a una sesion no protege de nada. Arreglo: entra de nuevo en " +
			"plazum y vuelve a enviar el formulario.")
	}
	// 3. Token. De la cabecera (htmx) o del cuerpo del formulario. NUNCA de la
	//    cadena de consulta: ahi acabaria en el Referer, en el historial del
	//    navegador y en el log de cualquier proxy por el que pase.
	token := r.Header.Get(CabeceraCSRF)
	if token == "" {
		token = r.PostFormValue(CampoCSRF)
	}
	if token == "" {
		return errors.New("peticion mutante sin token CSRF. Arreglo: el formulario tiene " +
			"que llevar el campo oculto " + CampoCSRF + ", y htmx la cabecera " +
			CabeceraCSRF + ".")
	}
	if err := p.Sesion.ComprobarCSRF(r.Context(), c.Value, token); err != nil {
		return fmt.Errorf("token CSRF rechazado: %v. Arreglo: recarga la pagina y vuelve "+
			"a enviarla; el token va atado a la sesion y caduca con ella", err)
	}
	return nil
}

// origenAceptable comprueba Origin y Sec-Fetch-Site cuando el navegador los
// manda. Cuando no los manda (curl, un cliente antiguo) no se rechaza por eso:
// la proteccion la sigue dando el token, que es lo que un tercero no puede leer.
func origenAceptable(r *http.Request) error {
	if s := r.Header.Get("Sec-Fetch-Site"); s == "cross-site" {
		return errors.New("peticion mutante marcada por el navegador como cross-site " +
			"(Sec-Fetch-Site). Otra pagina esta intentando enviar este formulario.")
	}
	o := r.Header.Get("Origin")
	if o == "" || o == "null" {
		return nil
	}
	u, err := url.Parse(o)
	if err != nil || u.Host == "" {
		return errors.New("la cabecera Origin de esta peticion no es una URL valida.")
	}
	if !strings.EqualFold(soloHost(u.Host), soloHost(r.Host)) {
		return errors.New("peticion mutante desde otro origen. Otra pagina esta " +
			"intentando enviar este formulario en tu nombre.")
	}
	return nil
}

// --- rate limit ---

// Limite es cuantos intentos caben en cuanta ventana.
type Limite struct {
	Maximo  int
	Ventana time.Duration
}

// Limites por defecto. El de autenticacion es deliberadamente estrecho: es el
// unico sitio donde probar mil veces sale a cuenta.
var (
	LimiteAuthPorDefecto    = Limite{Maximo: 10, Ventana: 5 * time.Minute}
	LimiteGeneralPorDefecto = Limite{Maximo: 600, Ventana: time.Minute}
)

// clavesMaximas es el techo de claves distintas que el limitador guarda.
const clavesMaximas = 200000

// Limitador es una ventana deslizante por clave.
//
// Cuenta TODOS los intentos, tambien los que ya rechazo. Consecuencia buscada:
// quien esta probando contrasenas no recupera el turno por seguir probando. Y
// consecuencia aceptada, dicha aqui para que nadie la descubra en produccion:
// varios operadores detras del mismo NAT comparten cubo, asi que el ataque de
// uno molesta a los demas de su oficina durante la ventana.
type Limitador struct {
	limite Limite

	mu     sync.Mutex
	golpes map[string][]time.Time
	// vaciados cuenta las veces que se tuvo que tirar el mapa entero por
	// presion de memoria. Se expone para el diagnostico: si crece, alguien
	// esta inundando desde muchas direcciones.
	vaciados int
}

// NuevoLimitador construye un limitador. Un Limite sin maximo o sin ventana
// cae al valor general por defecto en vez de no limitar nada.
func NuevoLimitador(l Limite) *Limitador {
	if l.Maximo <= 0 || l.Ventana <= 0 {
		l = LimiteGeneralPorDefecto
	}
	return &Limitador{limite: l, golpes: map[string][]time.Time{}}
}

// Permitir apunta un intento de clave y dice si se atiende.
//
// El instante entra como parametro y no se lee del reloj aqui: asi la
// caducidad de la ventana se puede probar sin dormir, que es la unica forma de
// que ese test corra en cada CI.
func (l *Limitador) Permitir(clave string, ahora time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.golpes) >= clavesMaximas {
		l.purgar(ahora)
		if len(l.golpes) >= clavesMaximas {
			// Se prefiere seguir atendiendo a dejar fuera a todo el mundo:
			// fallar cerrado aqui convierte una inundacion en una caida total.
			// Queda anotado para que el diagnostico lo pueda ensenar.
			l.golpes = map[string][]time.Time{}
			l.vaciados++
		}
	}

	corte := ahora.Add(-l.limite.Ventana)
	h := l.golpes[clave]
	n := 0
	for _, t := range h {
		if t.After(corte) {
			h[n] = t
			n++
		}
	}
	h = h[:n]

	permitido := len(h) < l.limite.Maximo
	// El tope del doble acota la memoria: mas alla no aporta informacion, la
	// decision ya esta tomada.
	if len(h) < 2*l.limite.Maximo {
		h = append(h, ahora)
	}
	if len(h) == 0 {
		delete(l.golpes, clave)
	} else {
		l.golpes[clave] = h
	}
	return permitido
}

// Vaciados devuelve cuantas veces se tiro el mapa por presion de memoria.
func (l *Limitador) Vaciados() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.vaciados
}

// purgar borra las claves sin golpes dentro de la ventana. Con el lock tomado.
func (l *Limitador) purgar(ahora time.Time) {
	corte := ahora.Add(-l.limite.Ventana)
	for k, h := range l.golpes {
		vivo := false
		for _, t := range h {
			if t.After(corte) {
				vivo = true
				break
			}
		}
		if !vivo {
			delete(l.golpes, k)
		}
	}
}

// ClaveCliente identifica al cliente para el rate limit.
//
// proxiesDeConfianza es cuantos proxies propios hay delante. Cero, que es el
// valor por defecto, significa NO MIRAR X-Forwarded-For: esa cabecera la
// escribe quien quiera, y confiar en ella por defecto es regalar el rate limit
// (una cabecera distinta en cada intento y cada intento estrena cubo).
//
// Con uno o mas, se cuenta DESDE LA DERECHA: la ultima entrada la pone el
// proxy inmediato, la penultima el anterior, y las de la izquierda son las que
// pudo inventar el cliente. Si vienen menos entradas de las que deberia haber
// puesto la cadena de proxies, se cae a la direccion real de la conexion en vez
// de creerse lo que llegue.
//
// Las direcciones IPv6 se agrupan por su prefijo /64: a un cliente IPv6 le
// suele sobrar una red entera, y limitar por direccion completa seria no
// limitar.
func ClaveCliente(r *http.Request, proxiesDeConfianza int) string {
	if proxiesDeConfianza > 0 {
		var partes []string
		for _, v := range r.Header.Values("X-Forwarded-For") {
			for _, p := range strings.Split(v, ",") {
				if p = strings.TrimSpace(p); p != "" {
					partes = append(partes, p)
				}
			}
		}
		if i := len(partes) - proxiesDeConfianza; i >= 0 && i < len(partes) {
			return normalizarIP(partes[i])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return normalizarIP(host)
}

func normalizarIP(s string) string {
	s = strings.Trim(strings.TrimSpace(s), "[]")
	// Puede venir con puerto (formato de algunos proxies).
	if h, _, err := net.SplitHostPort(s); err == nil {
		s = h
	}
	ip := net.ParseIP(strings.Trim(s, "[]"))
	if ip == nil {
		return "sin-ip:" + s
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.String()
	}
	return ip.Mask(net.CIDRMask(64, 128)).String() + "/64"
}

// Limitadores es el middleware de rate limit con dos cubos.
type Limitadores struct {
	// Auth es el cubo estrecho de las rutas de autenticacion.
	Auth *Limitador
	// General es el cubo ancho del resto.
	General *Limitador
	// EsAuth dice si esta peticion va a una ruta de autenticacion. Lo decide
	// el enrutador, no una lista escrita aparte.
	EsAuth func(*http.Request) bool
	// Clave identifica al cliente.
	Clave func(*http.Request) string
	// Reloj para poder probar sin dormir.
	Reloj func() time.Time
}

func (l Limitadores) Envolver(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ahora := time.Now
		if l.Reloj != nil {
			ahora = l.Reloj
		}
		cubo, limitador := "general", l.General
		if l.EsAuth != nil && l.EsAuth(r) {
			cubo, limitador = "auth", l.Auth
		}
		if limitador == nil {
			h.ServeHTTP(w, r)
			return
		}
		clave := l.Clave
		if clave == nil {
			// Sin funcion de clave, se cae a la direccion de la conexion. NO se
			// deja pasar sin limitar: un limitador que no limita por no tener
			// configurada la clave es peor que ninguno, porque parece que esta.
			clave = func(r *http.Request) string { return ClaveCliente(r, 0) }
		}
		espera := limitador.limite.Ventana
		if !limitador.Permitir(cubo+":"+clave(r), ahora()) {
			w.Header().Set("Retry-After", strconv.Itoa(int(espera.Seconds())))
			responder(w, http.StatusTooManyRequests,
				"demasiados intentos desde esta direccion. Vuelve a intentarlo dentro de "+
					espera.String()+". Si eres el operador de plazum y esto le esta pasando "+
					"a gente que no ha hecho nada raro, sube el limite de intentos en la "+
					"configuracion; si sois muchos detras de la misma salida a internet, "+
					"declara el proxy para que se cuente por persona y no por oficina.")
			return
		}
		h.ServeHTTP(w, r)
	})
}

// --- estaticos ---

// Estaticos sirve un sistema de ficheros embebido (go:embed) bajo prefijo.
//
// Existe aqui, y no en el frente de pantallas, porque servir un fichero es
// trabajo de servidor: quien vendoriza htmx decide QUE se sirve, no COMO. Lo
// que aporta esta pieza sobre http.FileServer es lo que a http.FileServer le
// falta para esto: nada de listados de directorio, nada de rutas que salgan
// del arbol, tipo MIME explicito (con nosniff, un tipo mal puesto deja de
// cargar en vez de ejecutarse como otra cosa) y cacheo largo e inmutable, que
// es lo que hace que la segunda visita no vuelva a bajar htmx.
func Estaticos(prefijo string, fsys fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			responder(w, http.StatusMethodNotAllowed, "los estaticos solo se leen.")
			return
		}
		rel := strings.TrimPrefix(r.URL.Path, prefijo)
		// path.Clean resuelve los .. ANTES de que la ruta llegue a fs, asi que
		// /estatico/../servidor.go no sale del arbol.
		rel = strings.TrimPrefix(path.Clean("/"+rel), "/")
		// Esta comprobacion es REDUNDANTE y queda anotado, no escondido: el
		// barrido de mutacion la quito y no se puso rojo nada, porque
		// fs.ReadFile rechaza por su cuenta la ruta vacia, el "." y todo lo que
		// no pase fs.ValidPath. Se queda porque hace explicito en el codigo lo
		// que si no habria que ir a buscar a la documentacion de io/fs, y
		// porque si manana esto deja de usar fs.ReadFile la proteccion no se va
		// con el. Lo que NO se hace es contarla como una defensa vigilada.
		if rel == "" || rel == "." || !fs.ValidPath(rel) {
			http.NotFound(w, r)
			return
		}
		b, err := fs.ReadFile(fsys, rel)
		if err != nil {
			// No se distingue "no existe" de "es un directorio": el que
			// pregunta no tiene por que aprender la forma del arbol.
			http.NotFound(w, r)
			return
		}
		tipo := mime.TypeByExtension(path.Ext(rel))
		if tipo == "" {
			tipo = "application/octet-stream"
		}
		w.Header().Set("Content-Type", tipo)
		w.Header().Set("Content-Length", strconv.Itoa(len(b)))
		// Los estaticos son la unica excepcion al no-store de Cabeceras: van
		// versionados por el nombre del fichero y no llevan datos de nadie.
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodHead {
			return
		}
		// #nosec G705 -- b no viene de la peticion: es el contenido de un
		// fichero del sistema de ficheros embebido, y va con su tipo MIME
		// declarado y nosniff puesto. De la peticion solo viene QUE fichero, y
		// eso ya lo acota fs.ReadFile sobre un fs de solo lectura.
		if _, err := w.Write(b); err != nil {
			return
		}
	})
}

// --- utilidades de respuesta ---

// responder escribe una respuesta de texto plano.
//
// Texto plano y nunca HTML a proposito: estos mensajes se producen en el camino
// del error, donde es facil colar dentro algo que venia de la peticion. En
// text/plain con nosniff, aunque se colara, no se ejecuta.
func responder(w http.ResponseWriter, codigo int, mensaje string) {
	h := w.Header()
	h.Set("Content-Type", "text/plain; charset=utf-8")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Del("Content-Length")
	w.WriteHeader(codigo)
	// #nosec G705 -- gosec ve que aqui puede acabar algo derivado de la
	// peticion y tiene razon en el analisis, pero no en la conclusion: las dos
	// lineas de arriba fijan text/plain y nosniff en ESTA misma funcion, asi
	// que el navegador no interpreta nada de lo que se escriba. Esa es
	// justamente la razon de que estos mensajes no se pinten como HTML: se
	// producen en el camino del error, donde es facil colar dentro algo que
	// venia de fuera.
	_, _ = io.WriteString(w, mensaje+"\n")
}

// cabecerasEsperadas es la lista de cabeceras de seguridad que toda respuesta
// tiene que llevar. Vive aqui, junto a quien las pone, para que el test y el
// workflow de CI comprueben exactamente esta lista y no una copia envejecida.
func cabecerasEsperadas() []string {
	c := []string{
		"Content-Security-Policy",
		"Strict-Transport-Security",
		"X-Frame-Options",
		"X-Content-Type-Options",
		"Referrer-Policy",
		"Cross-Origin-Opener-Policy",
		"Cross-Origin-Resource-Policy",
		"Permissions-Policy",
	}
	sort.Strings(c)
	return c
}
