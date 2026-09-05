package serve

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/marcosmatalab/plazum/puertos"
)

// El primer administrador.
//
// El problema que resuelve: una instalacion recien arrancada no tiene ningun
// usuario, asi que la primera pantalla no puede pedir credenciales que nadie
// tiene todavia. La respuesta habitual de la industria es un usuario por
// defecto (admin/admin) o un endpoint de registro abierto hasta que alguien lo
// use. Las dos son la misma cosa: una ventana en la que cualquiera que llegue
// antes que el operador se queda con la instalacion.
//
// Aqui la ventana se cierra con un secreto que solo ve quien arranco el
// proceso. Se imprime UNA VEZ por la salida estandar, no se escribe en ningun
// fichero de plazum, caduca, y solo sirve una vez.
//
// Lo que este diseno NO puede evitar, dicho aqui y en docs/tls.md porque
// callarlo seria vender humo: si plazum arranca como servicio de systemd, su
// salida estandar la recoge el journal, que es persistente y lo leen todos los
// del grupo systemd-journal. Por eso se detecta si la salida es un terminal y,
// cuando no lo es, se avisa en la propia impresion.

// sinAlmacenDeUsuarios es lo que se contesta cuando este servidor se monto sin
// forma de crear cuentas.
//
// LA PANTALLA HABLA CON QUIEN ESTA DELANTE, y quien esta delante es un
// responsable de cumplimiento a las nueve de la manana, sin documentacion y sin
// soporte. La version anterior de este mensaje decia «este servidor se construyo
// sin Config.CrearAdmin [...] Arreglo para quien lo cablea: pasa la funcion al
// construir serve.Config», que es una frase para el que compila el binario y no
// para el que lo usa: quien la leia no podia hacer nada con ella y no sabia si
// habia hecho algo mal.
//
// El orden es: que pasa, que se hace, y al final una linea para quien monta.
// Esa ultima linea se queda porque este paquete se puede montar suelto (lo hace
// el servidor de prueba de la puerta de seguridad web), y en ese caso el que
// esta delante SI es quien lo cablea.
const sinAlmacenDeUsuarios = "Este plazum se ha arrancado sin sitio donde guardar las " +
	"cuentas, asi que no puede crear el primer administrador ni dejar entrar a nadie. No " +
	"es un fallo de lo que has hecho tu.\n\n" +
	"Arreglo: para plazum (Ctrl+C en el terminal donde corre) y arrancalo con la orden " +
	"del producto, que es la que trae el almacen de usuarios puesto:\n\n" +
	"    plazum serve\n\n" +
	"Si quieres decidir donde vive el fichero de cuentas, anade --usuarios con la ruta:\n\n" +
	"    plazum serve --usuarios /var/lib/plazum/usuarios.json\n\n" +
	"Nota para quien integre este servidor en otro programa: falta Config.CrearAdmin al " +
	"construir serve.Config."

// duracionTokenAdminPorDefecto es una hora. Suficiente para que el operador
// vaya del terminal al navegador aunque se pare a mirar el proxy; corto para
// que no se quede vivo toda la tarde en el journal.
const duracionTokenAdminPorDefecto = time.Hour

// longitudMinimaSecreto es el suelo de la contrasena del primer administrador.
const longitudMinimaSecreto = 12

// tokenPrimerAdmin guarda el token de instalacion. Nunca guarda el token: solo
// su SHA-256, igual que una tabla de contrasenas, para que un volcado de
// memoria no lo entregue en claro.
type tokenPrimerAdmin struct {
	mu        sync.Mutex
	hash      [32]byte
	emitido   bool
	hasta     time.Time
	usado     bool
	reservado bool
}

func nuevoTokenPrimerAdmin() *tokenPrimerAdmin { return &tokenPrimerAdmin{} }

// emitir crea el token y devuelve el valor en claro UNA sola vez. Emitir otra
// vez invalida el anterior: es lo que hace que "reinicia plazum" sea la
// respuesta correcta cuando el operador pierde el token.
func (t *tokenPrimerAdmin) emitir(sec puertos.Secretos, ahora time.Time, duracion time.Duration) (string, error) {
	if duracion <= 0 {
		duracion = duracionTokenAdminPorDefecto
	}
	// 32 bytes son 256 bits: adivinarlo a ciegas no es una amenaza, y ademas
	// la ruta que lo consume esta en el cubo estrecho del rate limit.
	claro, err := sec.Token(32)
	if err != nil {
		return "", fmt.Errorf("no se puede emitir el token de primer administrador: %w", err)
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.hash = sha256.Sum256([]byte(claro))
	t.emitido = true
	t.hasta = ahora.Add(duracion)
	t.usado = false
	t.reservado = false
	return claro, nil
}

// hayTokenVivo dice si hay una instalacion en curso.
func (t *tokenPrimerAdmin) hayTokenVivo(ahora time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.emitido && !t.usado && ahora.Before(t.hasta)
}

// caducaEn devuelve el instante de caducidad.
func (t *tokenPrimerAdmin) caducaEn() time.Time {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.hasta
}

var (
	errSinInstalacion = errors.New("plazum no esta esperando ninguna instalacion: al " +
		"arrancar no imprimio ningun token de primer administrador.\n\n" +
		"Si esta instalacion es nueva y de verdad no hay ningun administrador todavia, " +
		"para plazum (Ctrl+C en el terminal donde corre) y vuelve a arrancarlo con la " +
		"misma orden de antes, por ejemplo:\n\n" +
		"    plazum serve\n\n" +
		"Al arrancar imprime un recuadro con un token de un solo uso. Copialo entero y " +
		"vuelve a esta pagina")
	errTokenCaducado = errors.New("el token de primer administrador ha caducado: solo vale " +
		"una hora.\n\n" +
		"Arreglo: para plazum (Ctrl+C en el terminal donde corre) y vuelve a arrancarlo:\n\n" +
		"    plazum serve\n\n" +
		"Imprimira uno nuevo y el viejo dejara de valer")
	errTokenUsado = errors.New("el token de primer administrador ya se uso. Solo sirve una " +
		"vez. Si el administrador se creo, entra por /entrar; si crees que lo uso otra " +
		"persona, para plazum, revisa quien pudo leer la salida del arranque y empieza de nuevo")
	errTokenEnUso = errors.New("hay otro intento de instalacion en curso con este token. " +
		"Arreglo: espera unos segundos y vuelve a intentarlo")
	errTokenNoCoincide = errors.New("el token no coincide con el que plazum imprimio al " +
		"arrancar. Arreglo: copialo entero, sin espacios; si lo has perdido, para plazum y " +
		"vuelve a arrancarlo para que emita otro")
)

// reservar comprueba el token y lo deja apartado mientras se crea el
// administrador. No lo marca usado todavia: si la creacion falla (una
// contrasena que el almacen rechaza, por ejemplo) el operador se quedaria sin
// token por un error suyo, y tendria que reiniciar el servicio para reintentar.
// Se libera con liberar() y se quema de verdad con consumir().
func (t *tokenPrimerAdmin) reservar(claro string, ahora time.Time) error {
	if claro == "" {
		return errTokenNoCoincide
	}
	recibido := sha256.Sum256([]byte(strings.TrimSpace(claro)))
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.emitido {
		return errSinInstalacion
	}
	if t.usado {
		return errTokenUsado
	}
	if !ahora.Before(t.hasta) {
		return errTokenCaducado
	}
	if t.reservado {
		return errTokenEnUso
	}
	if subtle.ConstantTimeCompare(t.hash[:], recibido[:]) != 1 {
		return errTokenNoCoincide
	}
	t.reservado = true
	return nil
}

func (t *tokenPrimerAdmin) liberar() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.reservado = false
}

// consumir quema el token para siempre y borra su hash.
func (t *tokenPrimerAdmin) consumir() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.usado = true
	t.reservado = false
	t.hash = [32]byte{}
}

// invalidar deja el token inservible sin marcarlo como "usado con exito". Se
// llama cuando se descubre que ya existe un administrador.
func (t *tokenPrimerAdmin) invalidar() { t.consumir() }

// --- arranque ---

// anunciar imprime por donde se entra y, si hace falta, el token de primer
// administrador. Devuelve error solo cuando no se puede decidir con seguridad
// si ya hay administrador: en ese caso es preferible no arrancar a imprimir un
// token de instalacion en un sistema que ya esta instalado.
func (s *Servidor) anunciar(ctx context.Context, direccion string) error {
	esquema := "http"
	if s.cfg.CertificadoTLS != "" {
		esquema = "https"
	}
	fmt.Fprintf(s.salida, "\nplazum escuchando en %s\nAbre %s://%s/ en el navegador.\n",
		direccion, esquema, direccion)
	if esquema == "http" && !s.cfg.CookieInsegura {
		fmt.Fprintln(s.salida,
			"Nota: plazum no esta terminando TLS. Si entras por un nombre que no sea "+
				"localhost, pon un proxy con TLS delante o no podras iniciar sesion: la "+
				"cookie lleva Secure y el navegador no la devolveria. Esta explicado en "+
				"docs/tls.md.")
	}

	if s.cfg.HayAdmin == nil || s.cfg.CrearAdmin == nil {
		fmt.Fprintln(s.salida,
			"AVISO: este servidor se construyo sin almacen de usuarios "+
				"(Config.HayAdmin o Config.CrearAdmin sin cablear), asi que no se puede "+
				"crear el primer administrador ni entrar. Sirve para montar pantallas y "+
				"para nada mas.")
		return nil
	}

	hay, err := s.cfg.HayAdmin(ctx)
	if err != nil {
		return fmt.Errorf(
			"no se puede saber si ya existe un administrador (%w), asi que plazum no "+
				"arranca. Imprimir un token de instalacion sin estar seguro abriria una "+
				"puerta en un sistema que a lo mejor ya esta instalado. Arreglo: mira que "+
				"el almacen sea legible y vuelve a arrancar", err)
	}
	if hay {
		s.hayAdmin.Store(true)
		fmt.Fprintln(s.salida, "Ya hay administrador. Entra por /entrar.")
		return nil
	}

	claro, err := s.admin.emitir(s.sec, s.ahora(), s.cfg.DuracionTokenPrimerAdmin)
	if err != nil {
		return err
	}
	s.imprimirTokenDeInstalacion(claro)
	return nil
}

// imprimirTokenDeInstalacion escribe el token UNA vez, con lo que el operador
// necesita saber alrededor.
func (s *Servidor) imprimirTokenDeInstalacion(claro string) {
	duracion := s.cfg.DuracionTokenPrimerAdmin
	if duracion <= 0 {
		duracion = duracionTokenAdminPorDefecto
	}
	linea := strings.Repeat("=", 72)
	fmt.Fprintf(s.salida, `
%s
 plazum todavia no tiene ningun administrador.

 Abre  /primer-admin  en el navegador y pega este token de un solo uso:

     %s

 Caduca %s (dentro de %s) y solo sirve UNA vez.
 Si lo pierdes o caduca: para plazum y vuelve a arrancarlo. Imprimira otro y
 este dejara de valer. No hay forma de recuperarlo, y es a proposito.
%s
`, linea, claro, s.admin.caducaEn().Format("a las 15:04 del 2006-01-02"),
		duracion.String(), linea)

	if !s.salidaEsTerminal() {
		fmt.Fprintln(s.salida,
			"AVISO: la salida de plazum no es un terminal, asi que este token acaba de\n"+
				"quedarse escrito donde vaya esa salida. Si plazum corre como servicio de\n"+
				"systemd, eso es el journal, que es persistente y lo lee todo el grupo\n"+
				"systemd-journal. Cuando termines de crear el administrador, considera el\n"+
				"token quemado igualmente: ya lo esta, porque solo sirve una vez.")
	}
}

// salidaEsTerminal dice si la salida configurada es un dispositivo de caracter.
func (s *Servidor) salidaEsTerminal() bool {
	f, ok := s.salida.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// --- exigencia ---

// hayAdministrador responde con cache: una vez que hay administrador, ya no
// deja de haberlo dentro de este proceso.
//
// Si la consulta falla se responde "no hay", y eso NO abre nada: el token de
// instalacion solo se emite en el arranque y solo cuando entonces no habia
// administrador. Sin token emitido, la pantalla de instalacion no crea nada.
func (s *Servidor) hayAdministrador(ctx context.Context) bool {
	if s.hayAdmin.Load() {
		return true
	}
	if s.cfg.HayAdmin == nil {
		return false
	}
	hay, err := s.cfg.HayAdmin(ctx)
	if err != nil {
		return false
	}
	if hay {
		s.hayAdmin.Store(true)
	}
	return hay
}

// rutasDeInstalacion son las que siguen atendiendose mientras no hay
// administrador.
func rutaDeInstalacion(ruta string) bool {
	return ruta == "/primer-admin" || ruta == "/salud" || strings.HasPrefix(ruta, "/estatico/")
}

// debeIrAPrimerAdmin dice si esta peticion tiene que acabar en la instalacion.
func (s *Servidor) debeIrAPrimerAdmin(r *http.Request) bool {
	return !s.hayAdministrador(r.Context()) && s.cfg.CrearAdmin != nil
}

// exigirPrimerAdmin manda a la instalacion mientras no haya administrador. Sin
// esto, el operador que acaba de instalar llega a una pantalla de entrada que
// no puede usar, con credenciales que nadie tiene.
func (s *Servidor) exigirPrimerAdmin(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rutaDeInstalacion(r.URL.Path) || !s.debeIrAPrimerAdmin(r) {
			h.ServeHTTP(w, r)
			return
		}
		if !EsMetodoMutante(r.Method) {
			http.Redirect(w, r, "/primer-admin", http.StatusSeeOther)
			return
		}
		responder(w, http.StatusConflict,
			"plazum todavia no tiene administrador, asi que no atiende nada mas que la "+
				"instalacion. Abre /primer-admin y usa el token que plazum imprimio al "+
				"arrancar.")
	})
}

// --- pantallas ---

func (s *Servidor) primerAdminFormulario(w http.ResponseWriter, r *http.Request) {
	if s.hayAdministrador(r.Context()) {
		s.admin.invalidar()
		http.Redirect(w, r, "/entrar", http.StatusSeeOther)
		return
	}
	if s.cfg.CrearAdmin == nil {
		responder(w, http.StatusServiceUnavailable, sinAlmacenDeUsuarios)
		return
	}
	if !s.admin.hayTokenVivo(s.ahora()) {
		responder(w, http.StatusConflict, errSinInstalacion.Error())
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
		Titulo: "Crear el primer administrador de plazum",
		// LA PANTALLA DICE DONDE ESTA EL TOKEN, no solo que hace falta uno.
		// Quien llega aqui viene de arrancar plazum y no sabe todavia que su
		// terminal ha impreso un recuadro: sin esta frase, el camino es leerse
		// el codigo o adivinarlo.
		Mensaje: "Esta instalacion todavia no tiene ninguna cuenta, asi que este es el " +
			"unico sitio por donde se entra la primera vez. En el terminal donde acabas " +
			"de arrancar plazum hay un recuadro con un token de un solo uso: copialo " +
			"entero y pegalo aqui. Caduca en una hora y solo sirve una vez; si lo pierdes, " +
			"para plazum y vuelve a arrancarlo para que imprima otro. Elige tambien el " +
			"usuario y la contrasena del administrador: la contrasena necesita al menos " +
			"12 caracteres y no se puede recuperar despues.",
		CSRF:      tok,
		Accion:    "/primer-admin",
		Boton:     "Crear administrador",
		PideToken: true,
		// EL NOMBRE DE LA ORGANIZACION, si hay donde guardarlo.
		PideOrganizacion: s.cfg.FijarOrganizacion != nil,
		Aviso:            s.avisoDeCookieQueNoVolvera(r),
	})
}

func (s *Servidor) primerAdmin(w http.ResponseWriter, r *http.Request) {
	if aviso := s.avisoDeCookieQueNoVolvera(r); aviso != "" {
		responder(w, http.StatusBadRequest, aviso)
		return
	}
	if s.cfg.CrearAdmin == nil {
		responder(w, http.StatusServiceUnavailable, sinAlmacenDeUsuarios)
		return
	}
	// Que no se haya colado un administrador por otra via mientras el token
	// seguia vivo. Si lo hay, el token deja de valer aqui mismo.
	if s.hayAdministrador(r.Context()) {
		s.admin.invalidar()
		responder(w, http.StatusConflict,
			"ya existe un administrador, asi que la instalacion esta hecha y el token ha "+
				"dejado de valer. Entra por /entrar.")
		return
	}

	token := strings.TrimSpace(r.PostFormValue("token"))
	usuario := strings.TrimSpace(r.PostFormValue("usuario"))
	secreto := r.PostFormValue("secreto")
	organizacion := strings.TrimSpace(r.PostFormValue("organizacion"))

	if err := s.admin.reservar(token, s.ahora()); err != nil {
		responder(w, http.StatusForbidden, err.Error())
		return
	}
	// A partir de aqui el token esta apartado: cualquier salida tiene que
	// liberarlo o quemarlo, nunca dejarlo apartado.
	if usuario == "" {
		s.admin.liberar()
		responder(w, http.StatusBadRequest,
			"falta el nombre de usuario del administrador.")
		return
	}
	if len([]rune(secreto)) < longitudMinimaSecreto {
		s.admin.liberar()
		responder(w, http.StatusBadRequest, fmt.Sprintf(
			"la contrasena del administrador tiene que llegar a %d caracteres. Es la "+
				"unica credencial que existe en esta instalacion todavia.",
			longitudMinimaSecreto))
		return
	}
	// LA ORGANIZACION SE FIJA ANTES DE CREAR AL ADMINISTRADOR, y el orden es
	// la decision.
	//
	// Si fallara despues, el administrador ya estaria creado, el token
	// quemado y la instalacion sin nombre: no habria forma de volver a este
	// formulario, que es el unico sitio donde se pregunta. Al reves, un fallo
	// al crear al administrador deja el nombre puesto y el formulario se
	// repinta; el nombre no estorba y se puede cambiar despues.
	//
	// Es la misma regla que el orden de escrituras de la subida del censo:
	// primero lo que se puede repetir, despues lo que no.
	if s.cfg.FijarOrganizacion != nil {
		if organizacion == "" {
			s.admin.liberar()
			responder(w, http.StatusBadRequest,
				"falta el nombre de tu organizacion. No hay valor por defecto a "+
					"proposito: es de quien son las obligaciones que plazum calcula, y "+
					"un acta que no dice de quien es no es evidencia de nadie.")
			return
		}
		if err := s.cfg.FijarOrganizacion(r.Context(), organizacion); err != nil {
			s.admin.liberar()
			responder(w, http.StatusBadRequest,
				"no se ha podido guardar el nombre de la organizacion: "+err.Error()+
					"\nEl token sigue valiendo: corrige y vuelve a intentarlo.")
			return
		}
	}
	if err := s.cfg.CrearAdmin(r.Context(), usuario, secreto); err != nil {
		s.admin.liberar()
		responder(w, http.StatusBadRequest,
			"no se ha podido crear el administrador: "+err.Error()+
				"\nEl token sigue valiendo: corrige y vuelve a intentarlo.")
		return
	}

	// Creado: el token se quema para siempre.
	s.admin.consumir()
	s.hayAdmin.Store(true)

	// Sesion nueva, y la anonima del formulario a la basura: subir de
	// privilegio con el mismo identificador es fijacion de sesion.
	if c, err := r.Cookie(s.nombreCookie()); err == nil && c.Value != "" {
		_ = s.ses.Cerrar(r.Context(), c.Value)
	}
	sujeto := usuario
	if s.cfg.Autenticar != nil {
		if suj, err := s.cfg.Autenticar(r.Context(), usuario, secreto); err == nil && suj != "" {
			sujeto = suj
		}
	}
	id, err := s.ses.Abrir(r.Context(), sujeto, s.cfg.DuracionSesion)
	if err != nil {
		// El administrador SI se creo: no se puede fingir lo contrario.
		responder(w, http.StatusServiceUnavailable,
			"el administrador se ha creado, pero no se ha podido abrir su sesion ("+
				err.Error()+"). Arreglo: entra por /entrar con las credenciales que "+
				"acabas de elegir.")
		return
	}
	s.ponerCookie(w, id, s.cfg.DuracionSesion)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
