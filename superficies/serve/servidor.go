// Package serve es la superficie HTTP de dutiq: el servidor, las sesiones y la
// seguridad web que la etapa 2 exige como puerta.
//
// Lo que hay aqui y lo que no. Aqui esta el servidor (arrancar, parar,
// cabeceras, CSRF, rate limit), el almacen de sesiones, el flujo de primer
// administrador y el servicio de ficheros estaticos. NO estan las pantallas ni
// las plantillas de la interfaz: eso se construye aparte y se entrega a este
// paquete como un http.Handler en Config.App.
//
// Esa frontera es la razon de que puertos.Servidor no exponga enrutado. Quien
// construye las rutas las declara; quien arranca solo arranca. La consecuencia
// practica, y la unica que importa: la comprobacion de CSRF NO se aplica por
// ruta, se aplica por METODO, asi que cubre tambien las rutas de un handler que
// este paquete no ha visto nunca. Una lista de rutas protegidas se desincroniza
// el dia que alguien anade un handler, que es exactamente el dia en que hace
// falta.
package serve

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"dutiq/adaptadores/secretos"
	"dutiq/puertos"
)

// --- rutas y enrutador ---

// Ruta es una ruta declarada, tal y como la conoce el enrutador.
type Ruta struct {
	// Metodo HTTP. Vacio significa "cualquier metodo".
	Metodo string
	// Patron en la sintaxis de http.ServeMux ("/entrar", "/perimetro/{id}").
	Patron string
	// Auth marca las rutas de autenticacion, que van al cubo estrecho del
	// rate limit. Lo decide quien declara la ruta, no una lista aparte.
	Auth bool
}

func (r Ruta) String() string {
	if r.Metodo == "" {
		return "* " + r.Patron
	}
	return r.Metodo + " " + r.Patron
}

// Mutante dice si esta ruta puede cambiar estado y por tanto exige CSRF. Una
// ruta sin metodo declarado cuenta como mutante: atiende POST tambien.
func (r Ruta) Mutante() bool {
	if r.Metodo == "" {
		return true
	}
	return EsMetodoMutante(r.Metodo)
}

// EnumeradorDeRutas lo implementa todo handler que sepa decir sus rutas.
//
// Existe para que la puerta de CSRF pueda preguntarle al router en vez de
// leer una lista escrita a mano al lado. Un handler de pantallas que lo
// implemente entra automaticamente en esa puerta; uno que no, queda cubierto
// igual por la comprobacion por metodo, pero no se puede enumerar.
type EnumeradorDeRutas interface {
	Rutas() []Ruta
}

// EnumeradorDePatrones lo implementa un handler que sabe decir sus patrones de
// http.ServeMux ("GET /alcance") sin conocer el tipo Ruta de este paquete.
//
// Existe para no acoplar las superficies. superficies/pantallas monta sus seis
// pantallas y ya lleva su propio registro de patrones, pero importar este
// paquete solo para hablar de Ruta la ataria a serve, y las dos tienen que poder
// sustituirse por separado. Asi la que posee el vocabulario es la que se adapta,
// que es donde debe estar el coste.
type EnumeradorDePatrones interface {
	Patrones() []string
}

// rutaDePatron traduce "GET /alcance" a una Ruta. Un patron sin metodo cuenta
// como ruta sin metodo, o sea mutante, que es el lado seguro: si alguien
// registra un patron sin metodo, entra en la puerta de CSRF en vez de colarse.
func rutaDePatron(p string) Ruta {
	metodo, patron, hayMetodo := strings.Cut(strings.TrimSpace(p), " ")
	if !hayMetodo {
		return Ruta{Patron: strings.TrimSpace(p)}
	}
	return Ruta{Metodo: strings.ToUpper(metodo), Patron: strings.TrimSpace(patron)}
}

// Enrutador es un http.ServeMux que ademas sabe decir que rutas tiene.
//
// http.ServeMux no lo dice, y esa es toda la razon de que este tipo exista: sin
// poder enumerar, la puerta de CSRF tendria que ser una lista mantenida a mano
// junto al codigo, y una lista a mano se desincroniza.
type Enrutador struct {
	mux   *http.ServeMux
	rutas []Ruta
}

// NuevoEnrutador construye un enrutador vacio.
func NuevoEnrutador() *Enrutador {
	return &Enrutador{mux: http.NewServeMux()}
}

// Manejar declara una ruta. metodo vacio significa cualquier metodo.
func (e *Enrutador) Manejar(metodo, patron string, h http.Handler) {
	e.declarar(Ruta{Metodo: metodo, Patron: patron}, h)
}

// ManejarAuth declara una ruta de autenticacion: la que se puede probar mil
// veces con provecho. Va al cubo estrecho del rate limit.
func (e *Enrutador) ManejarAuth(metodo, patron string, h http.Handler) {
	e.declarar(Ruta{Metodo: metodo, Patron: patron, Auth: true}, h)
}

func (e *Enrutador) declarar(r Ruta, h http.Handler) {
	patron := r.Patron
	if r.Metodo != "" {
		patron = r.Metodo + " " + r.Patron
	}
	e.mux.Handle(patron, h)
	e.rutas = append(e.rutas, r)
}

// Rutas devuelve las rutas declaradas, en el orden en que se declararon.
func (e *Enrutador) Rutas() []Ruta {
	return append([]Ruta(nil), e.rutas...)
}

func (e *Enrutador) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.mux.ServeHTTP(w, r)
}

var _ EnumeradorDeRutas = (*Enrutador)(nil)

// --- configuracion ---

// Config es todo lo que el servidor necesita saber. El cero de cada campo es un
// valor seguro y utilizable: Config{} arranca un servidor que no deja entrar a
// nadie y lo dice por que, en vez de uno que deja entrar a cualquiera.
type Config struct {
	// App es la aplicacion: las pantallas. Se monta en "/" y sus rutas quedan
	// cubiertas por la comprobacion de CSRF aunque este paquete no las conozca.
	// Si ademas implementa EnumeradorDeRutas o EnumeradorDePatrones, entran
	// en la puerta que las ENUMERA, que es la que caza una ruta mutante nueva
	// el mismo dia que se escribe.
	App http.Handler

	// Sesion es el almacen de sesiones. Nil construye el de este paquete.
	Sesion puertos.Sesion
	// Secretos es la fuente de aleatoriedad. Nil usa adaptadores/secretos.
	Secretos puertos.Secretos

	// Estaticos es el sistema de ficheros (normalmente un go:embed) que se
	// sirve bajo /estatico/. Ahi es donde vive htmx vendorizado.
	Estaticos fs.FS

	// Autenticar comprueba credenciales y devuelve el sujeto de la sesion.
	// Nil deniega siempre, diciendo que falta el almacen de usuarios.
	Autenticar func(ctx context.Context, usuario, secreto string) (string, error)
	// HayAdmin dice si ya existe algun administrador. Nil responde "no".
	HayAdmin func(ctx context.Context) (bool, error)
	// CrearAdmin crea el primer administrador. Nil falla diciendo que falta.
	CrearAdmin func(ctx context.Context, usuario, secreto string) error

	// Salida es donde se imprime el arranque, incluido el token de primer
	// administrador. Nil es os.Stdout. NUNCA se escribe ahi nada mas que en
	// el arranque, y el token no va a ningun otro sitio.
	Salida io.Writer
	// Reloj se inyecta para poder probar caducidades sin dormir. Nil es
	// time.Now.
	Reloj func() time.Time

	// DuracionSesion de una sesion autenticada. Cero son 8 horas.
	DuracionSesion time.Duration
	// DuracionTokenPrimerAdmin. Cero es una hora.
	DuracionTokenPrimerAdmin time.Duration

	// CookieInsegura quita el atributo Secure de la cookie de sesion. Solo
	// para una prueba en una red de confianza y sin TLS. Cambia ademas el
	// nombre de la cookie, porque el prefijo __Host- exige Secure.
	CookieInsegura bool

	// HostsPermitidos es la lista de nombres por los que se accede. Vacio
	// acepta cualquiera.
	HostsPermitidos []string
	// ProxiesDeConfianza es cuantos proxies propios hay delante. Cero, que es
	// lo correcto sin proxy, hace que X-Forwarded-For se ignore por completo.
	ProxiesDeConfianza int

	// LimiteAuth y LimiteGeneral son los dos cubos del rate limit.
	LimiteAuth    Limite
	LimiteGeneral Limite
	// MaxCuerpo en bytes de una peticion. Cero son 4 MiB.
	MaxCuerpo int64

	// CertificadoTLS y ClaveTLS son rutas a un certificado y su clave en PEM.
	// Puestas las dos, dutiq termina TLS el mismo, sin proxy delante. El
	// camino recomendado sigue siendo el proxy (docs/tls.md); esto es la
	// alternativa para quien no tiene ninguno.
	//
	// No hay emision automatica de certificados (ACME, Let's Encrypt): la
	// libreria que lo hace es una dependencia nueva y esa decision no se toma
	// aqui. Consta en docs/tls.md con el rodeo, que es certbot fuera del
	// proceso y estas dos rutas apuntando a lo que deja.
	CertificadoTLS string
	ClaveTLS       string

	// CSP y HSTS sustituyen los valores por defecto.
	CSP  string
	HSTS string
	// CSPDebilitadaAProposito deja pasar una CSP que validarCSP rechazaria.
	// Existe para que debilitarla sea una decision escrita y no un descuido.
	CSPDebilitadaAProposito bool
}

// Valores por defecto de Config.
const (
	duracionSesionPorDefecto = 8 * time.Hour
	maxCuerpoPorDefecto      = 4 << 20 // 4 MiB
	// SujetoAnonimo es el sujeto de la sesion previa a la autenticacion, la
	// que solo existe para llevar el token CSRF del formulario de entrada.
	SujetoAnonimo = "anonimo:sin-autenticar"
	// duracionAnonima es corta: es una sesion para rellenar un formulario.
	duracionAnonima = 15 * time.Minute
)

// EsAnonimo dice si un sujeto es el de una sesion previa a la autenticacion.
// Quien monte pantallas TIENE que preguntarlo: una sesion anonima existe y es
// valida, y aun asi no ha entrado nadie.
func EsAnonimo(sujeto string) bool { return sujeto == SujetoAnonimo }

// --- servidor ---

// plazosHTTP son los plazos del http.Server. No estan en Config a proposito:
// no son una preferencia del operador, son la defensa contra la conexion que
// abre y no manda nada, y aflojarlos desde fuera solo serviria para eso.
// Se aparta en un tipo para poder acortarlos en los tests y comprobar que la
// defensa existe de verdad, en vez de mirar que el campo esta puesto.
type plazosHTTP struct {
	leerCabecera time.Duration
	leer         time.Duration
	escribir     time.Duration
	inactivo     time.Duration
}

var plazosPorDefecto = plazosHTTP{
	leerCabecera: 10 * time.Second,
	leer:         60 * time.Second,
	escribir:     120 * time.Second,
	inactivo:     120 * time.Second,
}

// Servidor implementa puertos.Servidor.
type Servidor struct {
	cfg      Config
	ses      puertos.Sesion
	efimeras *Sesion // el mismo almacen, si es el nuestro, para sesiones anonimas
	sec      puertos.Secretos
	ahora    func() time.Time
	salida   io.Writer

	enr      *Enrutador
	mapaAuth *http.ServeMux
	handler  http.Handler
	plt      *template.Template

	admin    *tokenPrimerAdmin
	hayAdmin atomic.Bool
	plazos   plazosHTTP

	mu      sync.Mutex
	srv     *http.Server
	cerrado bool
}

var _ puertos.Servidor = (*Servidor)(nil)

// Nuevo construye el servidor. Falla, en vez de arrancar a medias, cuando la
// configuracion pedida no puede proteger lo que dice proteger.
func Nuevo(cfg Config) (*Servidor, error) {
	s := &Servidor{cfg: cfg, sec: cfg.Secretos, ahora: cfg.Reloj, salida: cfg.Salida,
		plazos: plazosPorDefecto}
	if s.sec == nil {
		s.sec = secretos.Nuevo()
	}
	if s.ahora == nil {
		s.ahora = time.Now
	}
	if s.salida == nil {
		s.salida = os.Stdout
	}
	if cfg.DuracionSesion <= 0 {
		s.cfg.DuracionSesion = duracionSesionPorDefecto
	}
	if cfg.MaxCuerpo <= 0 {
		s.cfg.MaxCuerpo = maxCuerpoPorDefecto
	}

	csp := cfg.CSP
	if csp == "" {
		csp = CSPPorDefecto
	}
	if !cfg.CSPDebilitadaAProposito {
		if err := validarCSP(csp); err != nil {
			return nil, fmt.Errorf("la CSP configurada no protege: %w", err)
		}
	}

	if cfg.Sesion != nil {
		s.ses = cfg.Sesion
	} else {
		ses, err := NuevaSesion(OpcionesSesion{Secretos: s.sec, Reloj: s.ahora})
		if err != nil {
			return nil, err
		}
		s.ses, s.efimeras = ses, ses
	}
	if propia, ok := s.ses.(*Sesion); ok {
		s.efimeras = propia
	}

	plt, err := template.New("serve").Parse(plantillasBase)
	if err != nil {
		return nil, fmt.Errorf("las plantillas de arranque no compilan: %w", err)
	}
	s.plt = plt

	s.admin = nuevoTokenPrimerAdmin()
	s.construirRutas()
	return s, nil
}

// construirRutas declara las rutas propias y compone la cadena de middleware.
func (s *Servidor) construirRutas() {
	e := NuevoEnrutador()
	e.Manejar(http.MethodGet, "/salud", http.HandlerFunc(s.salud))
	e.Manejar(http.MethodGet, "/entrar", http.HandlerFunc(s.entrarFormulario))
	e.ManejarAuth(http.MethodPost, "/entrar", http.HandlerFunc(s.entrar))
	e.Manejar(http.MethodPost, "/salir", http.HandlerFunc(s.salir))
	e.Manejar(http.MethodGet, "/primer-admin", http.HandlerFunc(s.primerAdminFormulario))
	e.ManejarAuth(http.MethodPost, "/primer-admin", http.HandlerFunc(s.primerAdmin))
	if s.cfg.Estaticos != nil {
		e.Manejar(http.MethodGet, "/estatico/", Estaticos("/estatico/", s.cfg.Estaticos))
	}
	app := s.cfg.App
	if app == nil {
		app = http.HandlerFunc(s.sinAplicacion)
	}
	e.Manejar("", "/", app)
	s.enr = e

	// El clasificador de rutas de autenticacion sale del PROPIO enrutador, no
	// de una lista escrita al lado: si manana se anade una ruta de auth, entra
	// sola en el cubo estrecho.
	s.mapaAuth = http.NewServeMux()
	for _, r := range s.Rutas() {
		if !r.Auth {
			continue
		}
		patron := r.Patron
		if r.Metodo != "" {
			patron = r.Metodo + " " + r.Patron
		}
		s.mapaAuth.Handle(patron, http.NotFoundHandler())
	}

	limitadores := Limitadores{
		Auth:    NuevoLimitador(s.limiteAuth()),
		General: NuevoLimitador(s.limiteGeneral()),
		EsAuth:  s.esRutaDeAuth,
		Clave:   func(r *http.Request) string { return ClaveCliente(r, s.cfg.ProxiesDeConfianza) },
		Reloj:   s.ahora,
	}
	protector := ProtectorCSRF{Sesion: s.ses, Cookie: s.nombreCookie()}

	// El orden importa y esta escrito de fuera hacia dentro:
	//   1. recuperar     un panic no puede dejar al operador sin respuesta
	//   2. cabeceras     LO MAS FUERA POSIBLE, para que TODA respuesta las
	//                    lleve: tambien el 405 de un TRACE y el 421 de un Host
	//                    falsificado, que son justo donde acaba el atacante.
	//                    Se ponen antes de llamar hacia dentro, asi que el 500
	//                    de un panic tambien sale con ellas.
	//   3. limitarCuerpo antes de que nadie lea el cuerpo
	//   4. metodos       TRACE y CONNECT no llegan a ningun handler
	//   5. host          se corta el Host falsificado en la puerta
	//   6. rate limit    antes del CSRF: probar tokens tambien se limita
	//   7. identificar   deja el sujeto en el contexto para las pantallas
	//   8. CSRF          por metodo, sobre todo lo que haya debajo
	//   9. primer admin  redirige a la instalacion mientras no haya nadie
	var h http.Handler = s.enr
	h = s.exigirPrimerAdmin(h)
	h = protector.Envolver(h)
	h = s.identificar(h)
	h = limitadores.Envolver(h)
	h = hostPermitido(s.cfg.HostsPermitidos, h)
	h = rechazarMetodosProhibidos(h)
	h = limitarCuerpo(s.cfg.MaxCuerpo, h)
	h = Cabeceras{CSP: s.cfg.CSP, HSTS: s.cfg.HSTS}.Envolver(h)
	h = recuperar(s.registrar, h)
	s.handler = h
}

func (s *Servidor) limiteAuth() Limite {
	if s.cfg.LimiteAuth.Maximo > 0 && s.cfg.LimiteAuth.Ventana > 0 {
		return s.cfg.LimiteAuth
	}
	return LimiteAuthPorDefecto
}

func (s *Servidor) limiteGeneral() Limite {
	if s.cfg.LimiteGeneral.Maximo > 0 && s.cfg.LimiteGeneral.Ventana > 0 {
		return s.cfg.LimiteGeneral
	}
	return LimiteGeneralPorDefecto
}

// esRutaDeAuth pregunta al clasificador construido desde el enrutador.
func (s *Servidor) esRutaDeAuth(r *http.Request) bool {
	_, patron := s.mapaAuth.Handler(r)
	return patron != ""
}

// Handler devuelve la aplicacion entera ya envuelta. Es lo que se pone en un
// httptest.Server, y lo que garantiza que el test mira lo mismo que el
// navegador y no una version sin middleware.
func (s *Servidor) Handler() http.Handler { return s.handler }

// Rutas devuelve las rutas propias mas las de la aplicacion, si esta sabe
// decirlas.
func (s *Servidor) Rutas() []Ruta {
	rutas := s.enr.Rutas()
	switch enum := s.cfg.App.(type) {
	case EnumeradorDeRutas:
		rutas = append(rutas, enum.Rutas()...)
	case EnumeradorDePatrones:
		// La aplicacion montada sabe decir sus patrones aunque no conozca
		// nuestro tipo Ruta. Se traducen aqui, que es donde vive el
		// vocabulario, y asi sus rutas entran en la puerta que ENUMERA y no
		// solo en la que comprueba por metodo.
		//
		// La aplicacion se monta en "/", asi que su patron ya es la
		// direccion que se pide.
		for _, p := range enum.Patrones() {
			rutas = append(rutas, rutaDePatron(p))
		}
	}
	return rutas
}

// Arrancar sirve hasta que el contexto se cancela.
func (s *Servidor) Arrancar(ctx context.Context, direccion string) error {
	if strings.TrimSpace(direccion) == "" {
		return errors.New("Arrancar sin direccion. Arreglo: pasa algo como " +
			"127.0.0.1:8443 (solo esta maquina) o :8443 (todas las interfaces)")
	}
	if err := s.validarTLS(); err != nil {
		return err
	}
	// Se escucha ANTES de servir para poder distinguir "el puerto esta
	// ocupado" de "el servidor se cayo luego": son dos problemas distintos y
	// el operador merece que se lo digamos.
	ln, err := net.Listen("tcp", direccion)
	if err != nil {
		return fmt.Errorf("no se puede escuchar en %s: %w.\n"+
			"Si dice que la direccion ya esta en uso, hay otro proceso con ese puerto: "+
			"en Linux miralo con `ss -ltnp | grep %s` y para ese proceso, o arranca dutiq "+
			"en otro puerto.\n"+
			"Si dice que el permiso esta denegado, es un puerto por debajo de 1024 y dutiq "+
			"no corre como root a proposito: usa un puerto alto y pon el proxy delante "+
			"(docs/tls.md)", direccion, err, puertoDe(direccion))
	}

	if err := s.anunciar(ctx, ln.Addr().String()); err != nil {
		_ = ln.Close()
		return err
	}

	srv := &http.Server{
		Handler: s.handler,
		// Sin estos plazos, una conexion que abre y no manda nada se queda
		// ocupando un hilo para siempre: es el ataque mas barato que existe
		// contra un servidor HTTP y no necesita ni ancho de banda.
		ReadHeaderTimeout: s.plazos.leerCabecera,
		ReadTimeout:       s.plazos.leer,
		WriteTimeout:      s.plazos.escribir,
		IdleTimeout:       s.plazos.inactivo,
		MaxHeaderBytes:    1 << 16,
		TLSConfig:         tlsMinimo(),
	}
	s.mu.Lock()
	if s.cerrado {
		s.mu.Unlock()
		_ = ln.Close()
		return errors.New("este servidor ya se paro; construye otro con serve.Nuevo")
	}
	s.srv = srv
	s.mu.Unlock()

	// El contexto es quien manda: al cancelarse se cierra en orden.
	listo := make(chan struct{})
	defer close(listo)
	go func() {
		select {
		case <-ctx.Done():
			// context.WithoutCancel y no el ctx de arriba, que acaba de
			// cancelarse: Shutdown con un contexto ya cancelado vuelve al
			// instante y corta las peticiones en curso, que es justo lo
			// contrario de cerrar en orden. Y no context.Background, para no
			// perder los valores que traiga el contexto del llamante.
			cierre, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
			defer cancelar()
			_ = srv.Shutdown(cierre)
		case <-listo:
		}
	}()

	if s.cfg.CertificadoTLS != "" || s.cfg.ClaveTLS != "" {
		err = srv.ServeTLS(ln, s.cfg.CertificadoTLS, s.cfg.ClaveTLS)
	} else {
		err = srv.Serve(ln)
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// tlsMinimo es la configuracion de TLS cuando dutiq lo termina el mismo.
//
// TLS 1.2 como suelo y no 1.0: el auditor que llegue con un escaner va a
// marcar cualquier cosa por debajo, y con razon. No se fija lista de cifrados
// porque en TLS 1.3 la elige la biblioteca y en 1.2 la de Go ya excluye las
// rotas; fijarla a mano aqui seria congelar hoy una lista que envejece.
func tlsMinimo() *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS12}
}

// validarTLS comprueba que el par certificado y clave esta completo y se puede
// leer, ANTES de escuchar. Si no, el operador descubriria el problema en la
// primera visita y no en el arranque.
func (s *Servidor) validarTLS() error {
	cert, clave := s.cfg.CertificadoTLS, s.cfg.ClaveTLS
	if cert == "" && clave == "" {
		return nil
	}
	if cert == "" || clave == "" {
		return errors.New("para terminar TLS hacen falta las DOS rutas, " +
			"Config.CertificadoTLS y Config.ClaveTLS. Arreglo: pon las dos, o ninguna y " +
			"pon un proxy con TLS delante (docs/tls.md)")
	}
	if _, err := tls.LoadX509KeyPair(cert, clave); err != nil {
		return fmt.Errorf(
			"no se puede usar el certificado %q con la clave %q: %w.\n"+
				"Arreglo: comprueba que las dos rutas existen, que estan en PEM y que la "+
				"clave es la de ese certificado. Si vienen de certbot, el certificado es "+
				"fullchain.pem y la clave privkey.pem", cert, clave, err)
	}
	return nil
}

// Parar cierra en orden. Idempotente.
func (s *Servidor) Parar(ctx context.Context) error {
	s.mu.Lock()
	srv := s.srv
	s.cerrado = true
	s.mu.Unlock()
	if srv == nil {
		return nil
	}
	err := srv.Shutdown(ctx)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func puertoDe(direccion string) string {
	if _, p, err := net.SplitHostPort(direccion); err == nil {
		return p
	}
	return direccion
}

func (s *Servidor) registrar(err error) {
	fmt.Fprintln(s.salida, "dutiq serve:", err)
}

// --- identificacion ---

type claveContexto struct{}

// SujetoDe devuelve el sujeto de la sesion de esta peticion, si hay sesion.
//
// Quien monte pantallas tiene que comprobar ademas EsAnonimo: una sesion
// anonima es valida y no ha entrado nadie.
func SujetoDe(r *http.Request) (string, bool) {
	v, ok := r.Context().Value(claveContexto{}).(string)
	return v, ok
}

// identificar lee la cookie y deja el sujeto en el contexto. No decide nada:
// autorizar es de quien monta la pantalla, que es el unico que sabe que exige
// cada una.
func (s *Servidor) identificar(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(s.nombreCookie())
		if err != nil || c.Value == "" {
			h.ServeHTTP(w, r)
			return
		}
		sujeto, err := s.ses.Leer(r.Context(), c.Value)
		if err != nil {
			h.ServeHTTP(w, r)
			return
		}
		h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claveContexto{}, sujeto)))
	})
}

// --- cookies ---

func (s *Servidor) nombreCookie() string {
	if s.cfg.CookieInsegura {
		return CookieSesionInsegura
	}
	return CookieSesion
}

func (s *Servidor) ponerCookie(w http.ResponseWriter, id string, duracion time.Duration) {
	// #nosec G124 -- HttpOnly y SameSite van fijos; Secure sale de la
	// configuracion y por eso gosec no puede verlo constante. Su valor por
	// defecto es true (el cero de Config.CookieInsegura), quitarlo exige
	// escribirlo, y el valor real de los tres atributos lo comprueban
	// TestLaCookieDeSesionLlevaHttpOnlySecureYSameSite y el paso 5 de
	// .github/workflows/etapa2-seguridad-web.yml sobre la respuesta del binario.
	http.SetCookie(w, &http.Cookie{
		Name:  s.nombreCookie(),
		Value: id,
		Path:  "/",
		// Sin Domain a proposito: con Domain la cookie viaja a todos los
		// subdominios, y uno comprometido se lleva la sesion. Ademas el
		// prefijo __Host- lo prohibe.
		MaxAge:   int(duracion.Seconds()),
		HttpOnly: true,
		Secure:   !s.cfg.CookieInsegura,
		// Lax y no Strict: con Strict, llegar a dutiq desde el enlace de un
		// correo de escalado ensena la pantalla como si no hubieras entrado, y
		// el operador vuelve a autenticarse cada vez. Lax sigue impidiendo que
		// otra pagina mande un POST con tu cookie, que es lo que importa.
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *Servidor) borrarCookie(w http.ResponseWriter) {
	// #nosec G124 -- misma razon que en ponerCookie. Ademas esta cookie va
	// vacia y con MaxAge negativo: lo unico que hace es borrar la del navegador.
	http.SetCookie(w, &http.Cookie{
		Name:     s.nombreCookie(),
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   !s.cfg.CookieInsegura,
		SameSite: http.SameSiteLaxMode,
	})
}

// conexionSegura dice si esta peticion llego por un canal donde una cookie
// Secure viaja de vuelta.
func (s *Servidor) conexionSegura(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	// X-Forwarded-Proto solo se mira si el operador ha declarado que hay un
	// proxy propio delante. Si no, es una cabecera que escribe cualquiera.
	if s.cfg.ProxiesDeConfianza > 0 &&
		strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https") {
		return true
	}
	// Los navegadores tratan localhost como contexto seguro y si devuelven
	// cookies Secure sobre http alli.
	switch soloHost(r.Host) {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}

// avisoDeCookieQueNoVolvera es el diagnostico del fallo mas silencioso de todo
// este flujo: con cookie Secure sobre http y sin localhost, el navegador acepta
// la respuesta, tira la cookie y la siguiente peticion llega sin sesion. El
// operador ve un bucle de entrada sin ningun mensaje. Se detecta aqui y se dice.
func (s *Servidor) avisoDeCookieQueNoVolvera(r *http.Request) string {
	if s.cfg.CookieInsegura || s.conexionSegura(r) {
		return ""
	}
	return "esta peticion ha llegado por http sin TLS y sin ser localhost, y la cookie de " +
		"sesion de dutiq lleva el atributo Secure, asi que tu navegador la aceptaria y no " +
		"la devolveria nunca: entrarias en un bucle sin ningun mensaje.\n\n" +
		"Dos arreglos, por orden de preferencia:\n" +
		"  1. Pon un proxy con TLS delante. Es media pagina de configuracion y esta " +
		"escrita en docs/tls.md. Si el proxy ya esta, dile a dutiq cuantos proxies hay " +
		"delante para que se fie de X-Forwarded-Proto.\n" +
		"  2. Solo para una prueba en una red de confianza, arranca con la cookie sin " +
		"Secure (Config.CookieInsegura). No lo dejes asi: la sesion viaja en claro."

}

// --- handlers propios ---

func (s *Servidor) salud(w http.ResponseWriter, _ *http.Request) {
	// Sin version, sin nombre de maquina, sin nada: una sonda de vida que
	// cuenta cosas es reconocimiento gratis para el que la llame.
	responder(w, http.StatusOK, "ok")
}

func (s *Servidor) sinAplicacion(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	sujeto, hay := SujetoDe(r)
	if !hay || EsAnonimo(sujeto) {
		http.Redirect(w, r, "/entrar", http.StatusSeeOther)
		return
	}
	responder(w, http.StatusOK,
		"dutiq esta arrancado y has entrado como "+sujeto+", pero este binario se "+
			"construyo sin pantallas montadas (Config.App). No es un error de tu "+
			"instalacion: es que este servidor se esta usando suelto.")
}

// sesionParaFormulario devuelve el identificador de sesion de esta peticion, y
// si no hay ninguna abre una anonima. Es lo que permite que el formulario de
// entrada lleve token CSRF sin haber entrado todavia, y por tanto que NINGUNA
// ruta mutante quede exenta de la comprobacion.
func (s *Servidor) sesionParaFormulario(w http.ResponseWriter, r *http.Request) (string, error) {
	if c, err := r.Cookie(s.nombreCookie()); err == nil && c.Value != "" {
		if _, err := s.ses.Leer(r.Context(), c.Value); err == nil {
			return c.Value, nil
		}
	}
	id, err := s.abrirEfimera(r.Context())
	if err != nil {
		return "", err
	}
	s.ponerCookie(w, id, duracionAnonima)
	return id, nil
}

// abrirEfimera abre la sesion anonima. Usa el hueco reservado del almacen
// propio cuando lo hay, para que una avalancha de visitas al formulario de
// entrada no pueda dejar sin sitio a las sesiones de quien ya entro.
func (s *Servidor) abrirEfimera(ctx context.Context) (string, error) {
	if s.efimeras != nil {
		return s.efimeras.AbrirEfimera(ctx, SujetoAnonimo, duracionAnonima)
	}
	return s.ses.Abrir(ctx, SujetoAnonimo, duracionAnonima)
}

// datosPagina es lo que ve una plantilla de arranque.
type datosPagina struct {
	Titulo  string
	CSRF    string
	Campo   string
	Accion  string
	Aviso   string
	Error   string
	Mensaje string
	Boton   string
	Usuario string
	// PideToken anade el campo del token de un solo uso.
	PideToken bool
}

func (s *Servidor) pintar(w http.ResponseWriter, codigo int, d datosPagina) {
	d.Campo = CampoCSRF
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(codigo)
	if err := s.plt.ExecuteTemplate(w, "pagina", d); err != nil {
		s.registrar(fmt.Errorf("pintando %q: %w", d.Titulo, err))
	}
}

func (s *Servidor) entrarFormulario(w http.ResponseWriter, r *http.Request) {
	if s.debeIrAPrimerAdmin(r) {
		http.Redirect(w, r, "/primer-admin", http.StatusSeeOther)
		return
	}
	id, err := s.sesionParaFormulario(w, r)
	if err != nil {
		responder(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	tok, err := s.ses.TokenCSRF(r.Context(), id)
	if err != nil {
		responder(w, http.StatusServiceUnavailable,
			"no se puede emitir el token del formulario: "+err.Error())
		return
	}
	s.pintar(w, http.StatusOK, datosPagina{
		Titulo: "Entrar en dutiq",
		CSRF:   tok,
		Accion: "/entrar",
		Boton:  "Entrar",
		Aviso:  s.avisoDeCookieQueNoVolvera(r),
	})
}

func (s *Servidor) entrar(w http.ResponseWriter, r *http.Request) {
	if aviso := s.avisoDeCookieQueNoVolvera(r); aviso != "" {
		// No se intenta autenticar: aunque saliera bien, la cookie no volveria
		// y el operador se quedaria dando vueltas sin saber por que.
		responder(w, http.StatusBadRequest, aviso)
		return
	}
	usuario := strings.TrimSpace(r.PostFormValue("usuario"))
	secreto := r.PostFormValue("secreto")

	autenticar := s.cfg.Autenticar
	if autenticar == nil {
		responder(w, http.StatusServiceUnavailable,
			"este servidor se construyo sin Config.Autenticar, asi que no hay contra que "+
				"comprobar credenciales. Arreglo para quien lo cablea: pasa la funcion de "+
				"autenticacion al construir serve.Config.")
		return
	}
	sujeto, err := autenticar(r.Context(), usuario, secreto)
	if err != nil || sujeto == "" {
		// Ni se distingue usuario inexistente de contrasena mala, ni se dice
		// cuantos intentos quedan: las dos cosas son enumeracion gratis.
		s.pintar(w, http.StatusUnauthorized, datosPagina{
			Titulo:  "Entrar en dutiq",
			Accion:  "/entrar",
			Boton:   "Entrar",
			Error:   "Usuario o contrasena incorrectos.",
			Usuario: usuario,
			CSRF:    s.tokenParaReintento(r),
		})
		return
	}
	if EsAnonimo(sujeto) {
		responder(w, http.StatusInternalServerError,
			"Config.Autenticar ha devuelto el sujeto reservado de las sesiones anonimas. "+
				"Arreglo: devuelve el identificador real de la persona.")
		return
	}

	// La sesion anonima del formulario se cierra y se abre una NUEVA. Ese
	// cambio de identificador al subir de privilegio es lo que cierra la
	// fijacion de sesion: si alguien planto una cookie antes de entrar, deja
	// de valer justo en el momento en que valdria la pena.
	if c, err := r.Cookie(s.nombreCookie()); err == nil && c.Value != "" {
		_ = s.ses.Cerrar(r.Context(), c.Value)
	}
	id, err := s.ses.Abrir(r.Context(), sujeto, s.cfg.DuracionSesion)
	if err != nil {
		responder(w, http.StatusServiceUnavailable, "no se puede abrir sesion: "+err.Error())
		return
	}
	s.ponerCookie(w, id, s.cfg.DuracionSesion)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// tokenParaReintento emite un token nuevo para volver a pintar el formulario
// tras un fallo. Si no se puede, la pagina sale sin token y el siguiente envio
// se rechazara, que es preferible a pintar un token que no vale.
func (s *Servidor) tokenParaReintento(r *http.Request) string {
	c, err := r.Cookie(s.nombreCookie())
	if err != nil || c.Value == "" {
		return ""
	}
	tok, err := s.ses.TokenCSRF(r.Context(), c.Value)
	if err != nil {
		return ""
	}
	return tok
}

func (s *Servidor) salir(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(s.nombreCookie()); err == nil && c.Value != "" {
		_ = s.ses.Cerrar(r.Context(), c.Value)
	}
	s.borrarCookie(w)
	http.Redirect(w, r, "/entrar", http.StatusSeeOther)
}

// plantillasBase son las dos unicas pantallas que trae este paquete: entrar y
// primer administrador. No son "la interfaz": son lo que tiene que funcionar
// antes de que exista interfaz, y por eso van aqui y no en el frente de
// pantallas. Sin CSS ni scripts propios, para que se pinten con la CSP mas
// estrecha posible y sin depender de ningun estatico.
//
// El <main> y el <footer> no son decoracion: sin ellos axe-core encuentra dos
// violaciones (landmark-one-main y region) en la pagina de entrada, que es la
// PRIMERA que ve cualquiera en un despliegue con autenticacion. La puerta de
// accesibilidad de CI audita /entrar por eso mismo: auditar solo las seis
// pantallas derivadas dejaba fuera la unica que se ve antes de entrar.
const plantillasBase = `{{define "pagina"}}<!doctype html>
<html lang="es"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Titulo}}</title></head>
<body>
<main>
<h1>{{.Titulo}}</h1>
{{if .Aviso}}<p role="alert"><strong>Aviso.</strong> {{.Aviso}}</p>{{end}}
{{if .Error}}<p role="alert"><strong>{{.Error}}</strong></p>{{end}}
{{if .Mensaje}}<p>{{.Mensaje}}</p>{{end}}
{{if .Accion}}
<form method="post" action="{{.Accion}}">
<input type="hidden" name="{{.Campo}}" value="{{.CSRF}}">
{{if .PideToken}}
<p><label for="token">Token de un solo uso, el que dutiq imprimio al arrancar</label><br>
<input id="token" name="token" type="password" autocomplete="off" required size="70"></p>
{{end}}
<p><label for="usuario">Usuario</label><br>
<input id="usuario" name="usuario" type="text" autocomplete="username" required value="{{.Usuario}}"></p>
<p><label for="secreto">Contrasena</label><br>
<input id="secreto" name="secreto" type="password" autocomplete="current-password" required></p>
<p><button type="submit">{{.Boton}}</button></p>
</form>
{{end}}
</main>
<footer>
<hr>
<p><small>dutiq no presta asesoramiento juridico.</small></p>
</footer>
</body></html>
{{end}}`
