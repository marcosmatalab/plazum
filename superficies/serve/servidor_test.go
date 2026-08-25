package serve

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"plazum/adaptadores/secretos"
)

// bufferSeguro es un io.Writer que se puede leer desde otra goroutine mientras
// el servidor escribe en el.
type bufferSeguro struct {
	mu sync.Mutex
	b  strings.Builder
}

func (x *bufferSeguro) Write(p []byte) (int, error) {
	x.mu.Lock()
	defer x.mu.Unlock()
	return x.b.Write(p)
}

func (x *bufferSeguro) String() string {
	x.mu.Lock()
	defer x.mu.Unlock()
	return x.b.String()
}

var reDireccion = regexp.MustCompile(`escuchando en (\S+)`)

// arrancarDePrueba levanta el servidor de verdad en un puerto libre y devuelve
// su direccion, leida de lo que el propio servidor imprime.
//
// Se lee de la salida a proposito: es la unica forma que tiene el operador de
// saber en que puerto acabo cuando pasa :0 o cuando el sistema le da otro, y si
// esa linea se rompe, este test se entera.
func arrancarDePrueba(t *testing.T, s *Servidor, salida *bufferSeguro) string {
	t.Helper()
	ctx, cancelar := context.WithCancel(context.Background())
	fin := make(chan error, 1)
	go func() { fin <- s.Arrancar(ctx, "127.0.0.1:0") }()
	t.Cleanup(func() {
		cancelar()
		select {
		case err := <-fin:
			if err != nil {
				t.Errorf("Arrancar devolvio %v en un cierre ordenado; tenia que devolver nil", err)
			}
		case <-time.After(20 * time.Second):
			t.Error("el servidor no ha cerrado en 20 s tras cancelar el contexto")
		}
	})

	limite := time.Now().Add(10 * time.Second)
	for time.Now().Before(limite) {
		if m := reDireccion.FindStringSubmatch(salida.String()); m != nil {
			return m[1]
		}
		select {
		case err := <-fin:
			t.Fatalf("el servidor termino antes de anunciarse: %v", err)
		case <-time.After(10 * time.Millisecond):
		}
	}
	t.Fatalf("el servidor no ha anunciado su direccion:\n%s", salida.String())
	return ""
}

func TestArrancarSirveDeVerdadYElContextoLoCierra(t *testing.T) {
	salida := &bufferSeguro{}
	s := servidorDePrueba(t, Config{
		Salida:   salida,
		HayAdmin: func(context.Context) (bool, error) { return true, nil },
	})
	dir := arrancarDePrueba(t, s, salida)

	resp, err := http.Get("http://" + dir + "/salud") // #nosec G107 -- direccion del propio servidor de prueba
	if err != nil {
		t.Fatalf("no responde en %s: %v", dir, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/salud respondio %d", resp.StatusCode)
	}
	// Y las cabeceras llegan por el cable, no solo por el recorder.
	for _, cab := range cabecerasEsperadas() {
		if resp.Header.Get(cab) == "" {
			t.Errorf("la respuesta real no lleva %s", cab)
		}
	}
}

// El puerto ocupado es lo primero que le pasa al que instala esto un martes por
// la manana. Tiene que decirlo con el arreglo, no con un errno.
func TestElPuertoOcupadoSeExplicaConSuArreglo(t *testing.T) {
	ocupado, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ocupado.Close() }()

	s := servidorDePrueba(t, Config{HayAdmin: func(context.Context) (bool, error) { return true, nil }})
	err = s.Arrancar(context.Background(), ocupado.Addr().String())
	if err == nil {
		t.Fatal("arrancar sobre un puerto ocupado no ha fallado")
	}
	texto := err.Error()
	for _, debe := range []string{"no se puede escuchar", "ss -ltnp", "otro puerto"} {
		if !strings.Contains(texto, debe) {
			t.Errorf("el error del puerto ocupado no menciona %q:\n%s", debe, texto)
		}
	}
}

func TestArrancarSinDireccionLoDiceConEjemplos(t *testing.T) {
	s := servidorDePrueba(t, Config{HayAdmin: func(context.Context) (bool, error) { return true, nil }})
	err := s.Arrancar(context.Background(), "   ")
	if err == nil {
		t.Fatal("arrancar sin direccion no ha fallado")
	}
	if !strings.Contains(err.Error(), "8443") {
		t.Errorf("el error no da un ejemplo de direccion: %v", err)
	}
}

// El certificado se comprueba ANTES de escuchar. Si no, el operador se entera
// en la primera visita, con el servicio ya arriba y el navegador diciendo algo
// que no se parece a "te falta la clave".
func TestUnCertificadoTLSIncompletoOIlegibleImpideArrancar(t *testing.T) {
	dir := t.TempDir()
	casos := []struct {
		nombre   string
		cfg      Config
		contiene string
	}{
		{"solo el certificado", Config{CertificadoTLS: filepath.Join(dir, "c.pem")}, "las DOS rutas"},
		{"solo la clave", Config{ClaveTLS: filepath.Join(dir, "k.pem")}, "las DOS rutas"},
		{"los dos, pero no existen", Config{
			CertificadoTLS: filepath.Join(dir, "c.pem"),
			ClaveTLS:       filepath.Join(dir, "k.pem"),
		}, "fullchain.pem"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			c.cfg.HayAdmin = func(context.Context) (bool, error) { return true, nil }
			s := servidorDePrueba(t, c.cfg)
			err := s.Arrancar(context.Background(), "127.0.0.1:0")
			if err == nil {
				t.Fatal("ha arrancado con el certificado incompleto o ilegible")
			}
			if !strings.Contains(err.Error(), c.contiene) {
				t.Errorf("el error no menciona %q: %v", c.contiene, err)
			}
		})
	}
}

func TestPararSinHaberArrancadoEsInocuoYRepetible(t *testing.T) {
	s := servidorDePrueba(t, Config{HayAdmin: func(context.Context) (bool, error) { return true, nil }})
	for i := 0; i < 3; i++ {
		if err := s.Parar(context.Background()); err != nil {
			t.Fatalf("Parar (%d) devolvio %v", i, err)
		}
	}
}

// Una conexion que abre y no manda nada es el ataque mas barato que hay contra
// un servidor HTTP: no necesita ancho de banda, solo descriptores. El plazo de
// lectura de cabecera es lo que la corta.
func TestUnaConexionQueNoMandaNadaSeCierraSola(t *testing.T) {
	salida := &bufferSeguro{}
	s := servidorDePrueba(t, Config{
		Salida:   salida,
		HayAdmin: func(context.Context) (bool, error) { return true, nil },
	})
	s.plazos.leerCabecera = 300 * time.Millisecond
	dir := arrancarDePrueba(t, s, salida)

	conn, err := net.Dial("tcp", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	b := make([]byte, 1)
	if _, err := conn.Read(b); err == nil {
		t.Fatal("el servidor ha contestado algo a una conexion que no mando nada")
	} else if ne, ok := err.(net.Error); ok && ne.Timeout() {
		t.Fatal("la conexion muda sigue abierta pasados 5 s. Sin plazo de lectura de " +
			"cabecera, mil conexiones mudas dejan el servidor sin descriptores")
	} else if err != io.EOF {
		// Cualquier otro error de lectura tambien vale: significa cerrada.
		t.Logf("la conexion se cerro con %v, que tambien es cerrar", err)
	}
}

// --- entrada y salida ---

func TestEntrarAbreSesionYSalirLaCierra(t *testing.T) {
	s, _, _, token := servidorSinInstalar(t, Config{})
	if rec := instalar(t, s, token, "ciso@ejemplo", "una-contrasena-larga"); rec.Code != http.StatusSeeOther {
		t.Fatal("no se ha podido instalar")
	}

	// Entrar por el formulario, como el navegador.
	get := httptest.NewRequest(http.MethodGet, "/entrar", nil)
	get.Host = hostLocal
	recGet := httptest.NewRecorder()
	s.Handler().ServeHTTP(recGet, get)
	if recGet.Code != http.StatusOK {
		t.Fatalf("GET /entrar respondio %d", recGet.Code)
	}
	m := reCSRF.FindStringSubmatch(recGet.Body.String())
	if m == nil {
		t.Fatal("el formulario de entrada no trae token CSRF")
	}
	cookiesForm := recGet.Result().Cookies()
	sesionAnonima := cookiesForm[0].Value

	post := httptest.NewRequest(http.MethodPost, "/entrar",
		strings.NewReader(CampoCSRF+"="+m[1]+"&usuario=ciso@ejemplo&secreto=una-contrasena-larga"))
	post.Host = hostLocal
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, c := range cookiesForm {
		post.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, post)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("entrar respondio %d: %s", rec.Code, rec.Body.String())
	}
	nuevas := rec.Result().Cookies()
	if len(nuevas) == 0 {
		t.Fatal("entrar no ha dejado cookie de sesion")
	}
	sesionReal := nuevas[0].Value

	// Fijacion de sesion: el identificador TIENE que cambiar al subir de
	// privilegio, y el anterior tiene que dejar de valer.
	if sesionReal == sesionAnonima {
		t.Fatal("entrar conserva el identificador de la sesion anonima. Quien plantase " +
			"una cookie antes de entrar se quedaria dentro con ella")
	}
	if _, err := s.ses.Leer(context.Background(), sesionAnonima); err == nil {
		t.Fatal("la sesion anonima del formulario sigue viva despues de entrar")
	}
	if suj, err := s.ses.Leer(context.Background(), sesionReal); err != nil || suj != "ciso@ejemplo" {
		t.Fatalf("la sesion nueva no lleva al sujeto correcto: %q, %v", suj, err)
	}

	// Salir la cierra y borra la cookie.
	tokSalir, err := s.ses.TokenCSRF(context.Background(), sesionReal)
	if err != nil {
		t.Fatal(err)
	}
	salir := httptest.NewRequest(http.MethodPost, "/salir", strings.NewReader(CampoCSRF+"="+tokSalir))
	salir.Host = hostLocal
	salir.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	salir.AddCookie(&http.Cookie{Name: s.nombreCookie(), Value: sesionReal})
	recSalir := httptest.NewRecorder()
	s.Handler().ServeHTTP(recSalir, salir)
	if recSalir.Code != http.StatusSeeOther {
		t.Fatalf("salir respondio %d", recSalir.Code)
	}
	if _, err := s.ses.Leer(context.Background(), sesionReal); err == nil {
		t.Fatal("tras salir, la sesion sigue viva: el boton de salir no protege de nada")
	}
	if c := recSalir.Result().Cookies(); len(c) == 0 || c[0].MaxAge >= 0 {
		t.Error("salir no ha borrado la cookie del navegador")
	}
}

// Credenciales malas no dicen cual de las dos: si el mensaje distingue, se
// enumeran usuarios gratis.
func TestUnasCredencialesMalasNoDicenCualEstaMal(t *testing.T) {
	s, _, _, token := servidorSinInstalar(t, Config{})
	if rec := instalar(t, s, token, "ciso@ejemplo", "una-contrasena-larga"); rec.Code != http.StatusSeeOther {
		t.Fatal("no se ha podido instalar")
	}
	cuerpos := []string{
		"usuario=ciso@ejemplo&secreto=mala",   // usuario que existe
		"usuario=no-existe@ejemplo&secreto=x", // usuario que no existe
	}
	var respuestas []string
	for _, c := range cuerpos {
		get := httptest.NewRequest(http.MethodGet, "/entrar", nil)
		get.Host = hostLocal
		recGet := httptest.NewRecorder()
		s.Handler().ServeHTTP(recGet, get)
		m := reCSRF.FindStringSubmatch(recGet.Body.String())
		if m == nil {
			t.Fatal("sin token CSRF en el formulario")
		}
		post := httptest.NewRequest(http.MethodPost, "/entrar",
			strings.NewReader(CampoCSRF+"="+m[1]+"&"+c))
		post.Host = hostLocal
		post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		for _, ck := range recGet.Result().Cookies() {
			post.AddCookie(ck)
		}
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, post)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("credenciales malas respondieron %d", rec.Code)
		}
		respuestas = append(respuestas, reCSRF.ReplaceAllString(rec.Body.String(), "CSRF"))
	}
	// La unica diferencia admisible entre las dos respuestas es el usuario que
	// se devuelve al formulario, que ya lo escribio el propio visitante.
	a := strings.ReplaceAll(respuestas[0], "ciso@ejemplo", "USUARIO")
	b := strings.ReplaceAll(respuestas[1], "no-existe@ejemplo", "USUARIO")
	if a != b {
		t.Fatalf("la respuesta distingue el usuario que existe del que no:\n--- a ---\n%s\n--- b ---\n%s", a, b)
	}
}

// El bucle de entrada silencioso: cookie Secure sobre http sin localhost. Se
// detecta y se explica, en vez de dejar al operador dando vueltas.
func TestSobreHTTPSinTLSNiLocalhostSeExplicaElBucleDeEntrada(t *testing.T) {
	s, _, _, token := servidorSinInstalar(t, Config{})
	if rec := instalar(t, s, token, "ciso@ejemplo", "una-contrasena-larga"); rec.Code != http.StatusSeeOther {
		t.Fatal("no se ha podido instalar")
	}

	post := httptest.NewRequest(http.MethodPost, "/entrar", strings.NewReader(""))
	post.Host = "grc.interno.ejemplo:8443" // ni TLS, ni localhost
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	id, err := s.ses.Abrir(context.Background(), SujetoAnonimo, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	tok, err := s.ses.TokenCSRF(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	post.Header.Set(CabeceraCSRF, tok)
	post.AddCookie(&http.Cookie{Name: s.nombreCookie(), Value: id})
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, post)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("respondio %d; se esperaba un 400 con el diagnostico", rec.Code)
	}
	for _, debe := range []string{"Secure", "docs/tls.md", "CookieInsegura"} {
		if !strings.Contains(rec.Body.String(), debe) {
			t.Errorf("el diagnostico no menciona %q:\n%s", debe, rec.Body.String())
		}
	}
	// Y con CookieInsegura, el mismo caso pasa: el operador que acepta el
	// riesgo puede trabajar.
	si, _, _, _ := servidorSinInstalar(t, Config{CookieInsegura: true})
	if aviso := si.avisoDeCookieQueNoVolvera(post); aviso != "" {
		t.Errorf("con CookieInsegura sigue avisando: %s", aviso)
	}
}

// Un servidor sin almacen de usuarios lo dice, en vez de rechazar credenciales
// como si fueran malas.
func TestSinAutenticarCableadoSeDiceQueFalta(t *testing.T) {
	s := servidorDePrueba(t, Config{HayAdmin: func(context.Context) (bool, error) { return true, nil }})
	id, err := s.ses.Abrir(context.Background(), SujetoAnonimo, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	tok, err := s.ses.TokenCSRF(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	post := httptest.NewRequest(http.MethodPost, "/entrar", strings.NewReader(""))
	post.Host = hostLocal
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post.Header.Set(CabeceraCSRF, tok)
	post.AddCookie(&http.Cookie{Name: s.nombreCookie(), Value: id})
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, post)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("respondio %d; se esperaba 503 diciendo que falta Config.Autenticar", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Config.Autenticar") {
		t.Errorf("el mensaje no dice que falta: %s", rec.Body.String())
	}
}

// Las sesiones anonimas no pueden dejar sin sitio a las de quien ya entro.
func TestLasSesionesAnonimasNoDesalojanALasAutenticadas(t *testing.T) {
	ses, err := NuevaSesion(OpcionesSesion{Secretos: secretos.Nuevo(), MaxSesiones: 10})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	autenticadas := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		id, err := ses.Abrir(ctx, "ciso", time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		autenticadas = append(autenticadas, id)
	}
	// Cien visitas al formulario de entrada.
	for i := 0; i < 100; i++ {
		if _, err := ses.AbrirEfimera(ctx, SujetoAnonimo, time.Minute); err != nil {
			t.Fatalf("la visita anonima %d ha fallado: %v", i, err)
		}
	}
	for i, id := range autenticadas {
		if _, err := ses.Leer(ctx, id); err != nil {
			t.Fatalf("la sesion autenticada %d se ha perdido tras 100 visitas anonimas: %v. "+
				"Cualquiera que pida el formulario en bucle echaria a la organizacion", i, err)
		}
	}
}
