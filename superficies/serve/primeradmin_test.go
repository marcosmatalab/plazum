package serve

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// hostLocal es el Host con el que van las peticiones de estos tests. Importa:
// httptest.NewRequest pone "example.com", y con ese Host el servidor detecta
// (bien) que la cookie Secure no volveria por http y corta el flujo con un
// diagnostico. Un operador de verdad entra por localhost o con TLS delante.
const hostLocal = "localhost"

// almacenDeUsuarios es el doble del almacen que todavia no existe: lo minimo
// para poder ejercitar el flujo entero de instalacion.
type almacenDeUsuarios struct {
	mu       sync.Mutex
	usuarios map[string]string
	// fallo, si esta puesto, hace que crear falle. Sirve para comprobar que un
	// fallo al crear NO quema el token de un solo uso.
	fallo error
	// mentirNoHay hace que el almacen diga "no hay administrador" aunque lo
	// haya. Con eso, el segundo intento de instalacion solo se puede rechazar
	// por el token, que es lo que se quiere aislar.
	mentirNoHay bool
	// antesDeCrear se llama dentro de CrearAdmin, con el token ya apartado.
	antesDeCrear func()
}

func nuevoAlmacen() *almacenDeUsuarios {
	return &almacenDeUsuarios{usuarios: map[string]string{}}
}

func (a *almacenDeUsuarios) hay(context.Context) (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.mentirNoHay {
		return false, nil
	}
	return len(a.usuarios) > 0, nil
}

func (a *almacenDeUsuarios) crear(_ context.Context, usuario, secreto string) error {
	if a.antesDeCrear != nil {
		a.antesDeCrear()
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.fallo != nil {
		return a.fallo
	}
	a.usuarios[usuario] = secreto
	return nil
}

func (a *almacenDeUsuarios) autenticar(_ context.Context, usuario, secreto string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if s, ok := a.usuarios[usuario]; ok && s == secreto && secreto != "" {
		return usuario, nil
	}
	return "", errors.New("credenciales incorrectas")
}

// servidorSinInstalar construye un servidor recien arrancado, sin usuarios, con
// el token de instalacion ya emitido, y devuelve el token en claro tal y como
// lo habria leido el operador en su terminal.
func servidorSinInstalar(t *testing.T, cfg Config) (*Servidor, *almacenDeUsuarios, *strings.Builder, string) {
	t.Helper()
	alm := nuevoAlmacen()
	salida := &strings.Builder{}
	cfg.HayAdmin = alm.hay
	cfg.CrearAdmin = alm.crear
	if cfg.Autenticar == nil {
		cfg.Autenticar = alm.autenticar
	}
	cfg.Salida = salida
	if cfg.LimiteAuth.Maximo == 0 {
		cfg.LimiteAuth = Limite{Maximo: 1000, Ventana: time.Minute}
	}
	if cfg.LimiteGeneral.Maximo == 0 {
		cfg.LimiteGeneral = Limite{Maximo: 10000, Ventana: time.Minute}
	}
	s, err := Nuevo(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.anunciar(context.Background(), "127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	return s, alm, salida, tokenImpreso(t, salida.String())
}

var reToken = regexp.MustCompile(`\b[0-9a-f]{64}\b`)

func tokenImpreso(t *testing.T, salida string) string {
	t.Helper()
	m := reToken.FindAllString(salida, -1)
	if len(m) == 0 {
		t.Fatalf("el arranque no ha impreso ningun token de 64 hex:\n%s", salida)
	}
	if len(m) > 1 {
		t.Fatalf("el arranque ha impreso %d tokens; tiene que imprimirse UNA vez", len(m))
	}
	return m[0]
}

var reCSRF = regexp.MustCompile(`name="` + CampoCSRF + `" value="([^"]+)"`)

// instalar hace el flujo completo del navegador: GET del formulario (que abre
// la sesion anonima y trae el token CSRF) y POST con los datos.
func instalar(t *testing.T, s *Servidor, token, usuario, secreto string) *httptest.ResponseRecorder {
	t.Helper()
	get := httptest.NewRequest(http.MethodGet, "/primer-admin", nil)
	get.Host = hostLocal
	recGet := httptest.NewRecorder()
	s.Handler().ServeHTTP(recGet, get)

	var csrf string
	var cookies []*http.Cookie
	if recGet.Code == http.StatusOK {
		m := reCSRF.FindStringSubmatch(recGet.Body.String())
		if m == nil {
			t.Fatalf("el formulario de instalacion no trae campo %q:\n%s",
				CampoCSRF, recGet.Body.String())
		}
		csrf, cookies = m[1], recGet.Result().Cookies()
		if len(cookies) == 0 {
			t.Fatal("el formulario de instalacion no ha abierto sesion: sin sesion no hay CSRF")
		}
	} else {
		// El formulario ya no se ofrece (por ejemplo porque el administrador ya
		// existe). Se fabrican sesion y token CSRF a mano para poder mandar el
		// POST igualmente: lo que se quiere comprobar es que el POST se rechaza,
		// no que la pantalla no se pinte.
		id, err := s.ses.Abrir(context.Background(), "curioso", time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		tok, err := s.ses.TokenCSRF(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		csrf = tok
		cookies = []*http.Cookie{{Name: s.nombreCookie(), Value: id}}
	}

	cuerpo := strings.NewReader(
		CampoCSRF + "=" + csrf + "&token=" + token + "&usuario=" + usuario + "&secreto=" + secreto)
	post := httptest.NewRequest(http.MethodPost, "/primer-admin", cuerpo)
	post.Host = hostLocal
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range cookies {
		post.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, post)
	return rec
}

func TestElFlujoDePrimerAdministradorFuncionaDeExtremoAExtremo(t *testing.T) {
	s, alm, _, token := servidorSinInstalar(t, Config{})

	rec := instalar(t, s, token, "ciso@ejemplo", "una-contrasena-larga")
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("la instalacion respondio %d y tenia que redirigir: %s", rec.Code, rec.Body.String())
	}
	hay, _ := alm.hay(context.Background())
	if !hay {
		t.Fatal("la instalacion dijo que si y no ha creado ningun administrador")
	}
	// Y deja sesion abierta: el operador no tiene que volver a escribir lo que
	// acaba de escribir.
	if len(rec.Result().Cookies()) == 0 {
		t.Error("tras crear el administrador no se ha abierto sesion")
	}
}

// La propiedad central: UN SOLO USO. Usado, no vale.
func TestElTokenDePrimerAdministradorSoloSirveUnaVez(t *testing.T) {
	s, _, _, token := servidorSinInstalar(t, Config{})
	if rec := instalar(t, s, token, "ciso@ejemplo", "una-contrasena-larga"); rec.Code != http.StatusSeeOther {
		t.Fatalf("la primera instalacion fallo: %d %s", rec.Code, rec.Body.String())
	}
	// El mismo token, otra vez. Ahora ademas ya hay administrador, asi que se
	// tiene que rechazar por las dos razones.
	rec := instalar(t, s, token, "atacante", "otra-contrasena-larga")
	if rec.Code == http.StatusSeeOther {
		t.Fatal("el token de un solo uso ha servido dos veces: quien lo lea en el journal " +
			"se hace administrador despues de que el operador ya lo sea")
	}
	if rec.Code != http.StatusConflict && rec.Code != http.StatusForbidden {
		t.Fatalf("el segundo uso respondio %d; se esperaba 409 o 403", rec.Code)
	}
}

// Y el mismo un solo uso AISLADO del "ya hay administrador".
//
// Este test existe por un hallazgo del barrido de mutacion: con el token
// convertido en reutilizable, el test de arriba seguia verde, porque el segundo
// intento lo rechazaba la comprobacion de "ya existe administrador" y no el
// token. Aqui el almacen miente y sigue diciendo que no hay ninguno, asi que lo
// unico que puede rechazar el segundo intento es el token quemado.
func TestElTokenSeQuemaAunqueElAlmacenSigaDiciendoQueNoHayAdministrador(t *testing.T) {
	s, alm, _, token := servidorSinInstalar(t, Config{})
	if rec := instalar(t, s, token, "ciso@ejemplo", "una-contrasena-larga"); rec.Code != http.StatusSeeOther {
		t.Fatal("la primera instalacion no ha funcionado")
	}
	alm.mentirNoHay = true
	s.hayAdmin.Store(false) // como si el proceso no se hubiera enterado

	rec := instalar(t, s, token, "atacante", "otra-contrasena-larga")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("el segundo uso del token respondio %d y tenia que ser 403 por token "+
			"quemado. Si esto pasa, 'un solo uso' lo esta sosteniendo la comprobacion de "+
			"'ya hay administrador' y no el token", rec.Code)
	}
	alm.mu.Lock()
	_, hayAtacante := alm.usuarios["atacante"]
	alm.mu.Unlock()
	if hayAtacante {
		t.Fatal("el segundo uso ha creado un administrador")
	}
	// Y a nivel de la pieza, sin HTTP por medio.
	if err := s.admin.reservar(token, s.ahora()); err == nil {
		t.Fatal("el token usado se sigue reservando")
	}
}

// Caducado, tampoco vale.
func TestElTokenDePrimerAdministradorCaduca(t *testing.T) {
	reloj := &relojControlado{t: time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)}
	s, _, _, token := servidorSinInstalar(t, Config{
		Reloj:                    reloj.Ahora,
		DuracionTokenPrimerAdmin: 30 * time.Minute,
	})
	reloj.Avanzar(31 * time.Minute)

	// Ni siquiera se llega a pintar el formulario.
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/primer-admin", nil))
	if rec.Code != http.StatusConflict {
		t.Fatalf("con el token caducado, GET /primer-admin respondio %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "arrancarlo") {
		t.Errorf("el mensaje no dice como salir del atasco: %s", rec.Body.String())
	}
	// Y el token viejo tampoco pasa por la puerta de atras.
	if err := s.admin.reservar(token, reloj.Ahora()); err == nil {
		t.Fatal("un token caducado sigue reservandose")
	}
}

func TestUnTokenInventadoNoCreaAdministrador(t *testing.T) {
	s, alm, _, _ := servidorSinInstalar(t, Config{})
	rec := instalar(t, s, strings.Repeat("f", 64), "atacante", "una-contrasena-larga")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("un token inventado respondio %d y tenia que ser 403", rec.Code)
	}
	if hay, _ := alm.hay(context.Background()); hay {
		t.Fatal("un token inventado ha creado administrador")
	}
}

// Reiniciar es la respuesta a "he perdido el token", y tiene que invalidar el
// anterior: si no, un token perdido sigue vivo en el journal.
func TestReiniciarEmiteOtroTokenYElAnteriorDejaDeValer(t *testing.T) {
	s, _, salida, primero := servidorSinInstalar(t, Config{})
	salida.Reset()
	if err := s.anunciar(context.Background(), "127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	segundo := tokenImpreso(t, salida.String())
	if primero == segundo {
		t.Fatal("el segundo arranque ha impreso el mismo token")
	}
	if err := s.admin.reservar(primero, s.ahora()); err == nil {
		t.Fatal("el token del arranque anterior sigue valiendo. Entonces perder un token " +
			"no se arregla reiniciando, y el viejo se queda vivo en el journal")
	}
	if err := s.admin.reservar(segundo, s.ahora()); err != nil {
		t.Fatalf("el token nuevo no vale: %v", err)
	}
}

// Si ya hay administrador, no se imprime ningun token: imprimirlo seria abrir
// una puerta en un sistema ya instalado, cada vez que se reinicia el servicio.
func TestConAdministradorYaCreadoNoSeImprimeToken(t *testing.T) {
	alm := nuevoAlmacen()
	if err := alm.crear(context.Background(), "ciso", "x"); err != nil {
		t.Fatal(err)
	}
	salida := &strings.Builder{}
	s, err := Nuevo(Config{HayAdmin: alm.hay, CrearAdmin: alm.crear, Salida: salida})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.anunciar(context.Background(), "127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	if reToken.MatchString(salida.String()) {
		t.Fatalf("se ha impreso un token de instalacion con un administrador ya creado:\n%s",
			salida.String())
	}
	if !strings.Contains(salida.String(), "/entrar") {
		t.Errorf("el arranque no dice por donde se entra:\n%s", salida.String())
	}
}

// Si no se puede saber si hay administrador, NO se arranca. Fallar abierto aqui
// seria imprimir un token de instalacion en un sistema que a lo mejor ya lo esta.
func TestSiNoSePuedeSaberSiHayAdministradorNoSeArranca(t *testing.T) {
	s, err := Nuevo(Config{
		HayAdmin:   func(context.Context) (bool, error) { return false, errors.New("base ilegible") },
		CrearAdmin: func(context.Context, string, string) error { return nil },
		Salida:     &strings.Builder{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.anunciar(context.Background(), "127.0.0.1:0"); err == nil {
		t.Fatal("se ha anunciado el arranque sin poder saber si ya hay administrador")
	}
}

// Un fallo al crear (contrasena corta, almacen que la rechaza) NO puede quemar
// el token: el operador tendria que reiniciar el servicio por una errata.
func TestUnFalloAlCrearNoQuemaElToken(t *testing.T) {
	s, alm, _, token := servidorSinInstalar(t, Config{})

	if rec := instalar(t, s, token, "ciso", "corta"); rec.Code != http.StatusBadRequest {
		t.Fatalf("una contrasena de 5 caracteres respondio %d", rec.Code)
	}
	alm.fallo = errors.New("el almacen dice que no")
	if rec := instalar(t, s, token, "ciso", "una-contrasena-larga"); rec.Code != http.StatusBadRequest {
		t.Fatalf("un fallo del almacen respondio %d", rec.Code)
	}
	alm.fallo = nil
	if rec := instalar(t, s, token, "ciso", "una-contrasena-larga"); rec.Code != http.StatusSeeOther {
		t.Fatalf("tras dos fallos que no eran del token, la instalacion buena respondio "+
			"%d. Un token quemado por una errata obliga a reiniciar el servicio", rec.Code)
	}
}

// Dos instalaciones a la vez con el mismo token: solo una puede crear. Es la
// diferencia entre "un solo uso" y "un solo uso salvo carrera".
func TestDosInstalacionesSimultaneasConElMismoTokenSoloCreanUna(t *testing.T) {
	s, alm, _, token := servidorSinInstalar(t, Config{})
	arranque := make(chan struct{})
	alm.antesDeCrear = func() { <-arranque }

	resultados := make(chan int, 2)
	for i := 0; i < 2; i++ {
		go func(i int) {
			// Cada goroutine hace su propio flujo completo.
			defer func() {
				if p := recover(); p != nil {
					resultados <- -1
				}
			}()
			resultados <- instalarSinT(s, token, "admin", "una-contrasena-larga")
		}(i)
	}
	// Se deja pasar a las dos a la vez.
	time.Sleep(50 * time.Millisecond)
	close(arranque)

	a, b := <-resultados, <-resultados
	exitos := 0
	for _, c := range []int{a, b} {
		if c == http.StatusSeeOther {
			exitos++
		}
	}
	if exitos != 1 {
		t.Fatalf("de dos instalaciones simultaneas con el mismo token han salido %d bien "+
			"(codigos %d y %d). Un solo uso que se salta con una carrera no es un solo uso",
			exitos, a, b)
	}
	alm.mu.Lock()
	n := len(alm.usuarios)
	alm.mu.Unlock()
	if n != 1 {
		t.Fatalf("se han creado %d administradores", n)
	}
}

// instalarSinT es instalar() sin *testing.T, para poder llamarla desde una
// goroutine sin usar t desde varios sitios.
func instalarSinT(s *Servidor, token, usuario, secreto string) int {
	get := httptest.NewRequest(http.MethodGet, "/primer-admin", nil)
	get.Host = hostLocal
	recGet := httptest.NewRecorder()
	s.Handler().ServeHTTP(recGet, get)
	m := reCSRF.FindStringSubmatch(recGet.Body.String())
	if m == nil {
		return recGet.Code
	}
	cuerpo := strings.NewReader(
		CampoCSRF + "=" + m[1] + "&token=" + token + "&usuario=" + usuario + "&secreto=" + secreto)
	post := httptest.NewRequest(http.MethodPost, "/primer-admin", cuerpo)
	post.Host = hostLocal
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range recGet.Result().Cookies() {
		post.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, post)
	return rec.Code
}

// Mientras no hay administrador, todo lleva a la instalacion. Sin esto el
// operador aterriza en una pantalla de entrada con credenciales que no existen.
func TestMientrasNoHayAdministradorTodoLlevaALaInstalacion(t *testing.T) {
	app := NuevoEnrutador()
	app.Manejar(http.MethodGet, "/app/hoy", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	s, _, _, _ := servidorSinInstalar(t, Config{App: app})

	for _, ruta := range []string{"/", "/app/hoy", "/entrar"} {
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ruta, nil))
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/primer-admin" {
			t.Errorf("%s respondio %d hacia %q; tenia que llevar a /primer-admin",
				ruta, rec.Code, rec.Header().Get("Location"))
		}
	}
	// Menos la sonda de vida, que tiene que responder desde el primer segundo:
	// es lo que mira el arranque del contenedor.
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/salud", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("/salud respondio %d antes de instalar", rec.Code)
	}
}

// Y en cuanto hay administrador, la instalacion desaparece.
func TestConAdministradorCreadoLaInstalacionDejaDeAtenderse(t *testing.T) {
	s, _, _, token := servidorSinInstalar(t, Config{})
	if rec := instalar(t, s, token, "ciso", "una-contrasena-larga"); rec.Code != http.StatusSeeOther {
		t.Fatal("la instalacion no ha funcionado")
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/primer-admin", nil))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/entrar" {
		t.Fatalf("con administrador creado, /primer-admin respondio %d hacia %q",
			rec.Code, rec.Header().Get("Location"))
	}
}

// El token viaja en el CUERPO del formulario y nunca en la URL: en la URL
// acabaria en el historial, en el Referer y en el log del proxy.
func TestElTokenDeInstalacionNoSeAceptaPorLaURL(t *testing.T) {
	s, alm, _, token := servidorSinInstalar(t, Config{})
	get := httptest.NewRequest(http.MethodGet, "/primer-admin", nil)
	get.Host = hostLocal
	recGet := httptest.NewRecorder()
	s.Handler().ServeHTTP(recGet, get)
	m := reCSRF.FindStringSubmatch(recGet.Body.String())
	if m == nil {
		t.Fatal("sin campo CSRF en el formulario")
	}
	cuerpo := strings.NewReader(CampoCSRF + "=" + m[1] + "&usuario=x&secreto=una-contrasena-larga")
	post := httptest.NewRequest(http.MethodPost, "/primer-admin?token="+token, cuerpo)
	post.Host = hostLocal
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range recGet.Result().Cookies() {
		post.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, post)
	if rec.Code == http.StatusSeeOther {
		t.Fatal("el token en la cadena de consulta ha valido")
	}
	if hay, _ := alm.hay(context.Background()); hay {
		t.Fatal("se ha creado administrador con el token por la URL")
	}
}

// La impresion del token dice lo que el operador necesita: donde ir, cuanto
// dura, que solo sirve una vez y que hacer si lo pierde.
func TestLaImpresionDelTokenDiceLoQueElOperadorNecesita(t *testing.T) {
	_, _, salida, _ := servidorSinInstalar(t, Config{})
	texto := salida.String()
	for _, debe := range []string{"/primer-admin", "un solo uso", "UNA vez", "arrancarlo"} {
		if !strings.Contains(texto, debe) {
			t.Errorf("el arranque no menciona %q:\n%s", debe, texto)
		}
	}
	// Y avisa de que la salida no es un terminal, que es el caso de systemd.
	if !strings.Contains(texto, "journal") {
		t.Errorf("no se avisa de que la salida puede quedar guardada en el journal:\n%s", texto)
	}
}
