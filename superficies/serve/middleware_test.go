package serve

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"
)

// --- cabeceras ---

// La cabecera puesta en el camino feliz no sirve de nada si falta en el 403 o
// en el 404, que es donde acaba el atacante. Se comprueban los cinco codigos
// por los que se sale de verdad.
func TestTodasLasRespuestasLlevanLasCabecerasDeSeguridad(t *testing.T) {
	app := NuevoEnrutador()
	app.Manejar(http.MethodGet, "/app/ok", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	app.Manejar(http.MethodGet, "/app/panico", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("a proposito")
	}))
	s, sesion := servidorConSesion(t, Config{App: app})

	casos := []struct {
		nombre string
		hacer  func() *httptest.ResponseRecorder
	}{
		{"200", func() *httptest.ResponseRecorder {
			return pedir(s, http.MethodGet, "/app/ok", sesion, nil)
		}},
		{"404", func() *httptest.ResponseRecorder {
			return pedir(s, http.MethodGet, "/app/no-existe", sesion, nil)
		}},
		{"403 sin CSRF", func() *httptest.ResponseRecorder {
			return pedir(s, http.MethodPost, "/app/ok", sesion, nil)
		}},
		{"405 TRACE", func() *httptest.ResponseRecorder {
			return pedir(s, http.MethodTrace, "/app/ok", sesion, nil)
		}},
		{"500 panic", func() *httptest.ResponseRecorder {
			return pedir(s, http.MethodGet, "/app/panico", sesion, nil)
		}},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			rec := c.hacer()
			for _, cab := range cabecerasEsperadas() {
				if rec.Header().Get(cab) == "" {
					t.Errorf("la respuesta %d de %s no lleva %s. Una cabecera que solo "+
						"aparece en el camino feliz no protege el camino en el que acaba "+
						"el atacante", rec.Code, c.nombre, cab)
				}
			}
		})
	}
}

// La CSP no se puede debilitar sin que el servidor se niegue a arrancar. Con el
// control negativo pegado: las politicas de abajo TIENEN que ser rechazadas.
func TestUnaCSPDebilitadaImpideArrancar(t *testing.T) {
	if err := validarCSP(CSPPorDefecto); err != nil {
		t.Fatalf("la CSP por defecto no se valida a si misma: %v", err)
	}
	malas := map[string]string{
		"inline":              "default-src 'self'; script-src 'self' 'unsafe-inline'; frame-ancestors 'none'",
		"eval":                "default-src 'self'; script-src 'unsafe-eval'; frame-ancestors 'none'",
		"comodin":             "default-src *; frame-ancestors 'none'",
		"esquema":             "default-src 'self'; script-src https:; frame-ancestors 'none'",
		"sin frame-ancestors": "default-src 'self'; script-src 'self'",
		"sin nada":            "img-src 'self'; frame-ancestors 'none'",
	}
	for nombre, csp := range malas {
		t.Run(nombre, func(t *testing.T) {
			if err := validarCSP(csp); err == nil {
				t.Fatalf("validarCSP ha aceptado %q", csp)
			}
			if _, err := Nuevo(Config{CSP: csp}); err == nil {
				t.Fatalf("el servidor ha arrancado con la CSP %q", csp)
			}
		})
	}
	// Y una CSP legitima con data: en las imagenes no puede dar falso positivo.
	buena := "default-src 'self'; img-src 'self' data:; frame-ancestors 'none'"
	if err := validarCSP(buena); err != nil {
		t.Fatalf("validarCSP rechaza una politica razonable (%q): %v", buena, err)
	}
	// Debilitarla a sabiendas si se puede, y tiene que ser explicito.
	if _, err := Nuevo(Config{
		CSP:                     "default-src 'self' 'unsafe-inline'; frame-ancestors 'none'",
		CSPDebilitadaAProposito: true,
	}); err != nil {
		t.Fatalf("con CSPDebilitadaAProposito el servidor tiene que arrancar: %v", err)
	}
}

// --- cookies ---

func TestLaCookieDeSesionLlevaHttpOnlySecureYSameSite(t *testing.T) {
	s := servidorDePrueba(t, Config{HayAdmin: func(context.Context) (bool, error) { return true, nil }})
	rec := httptest.NewRecorder()
	s.ponerCookie(rec, "un-identificador", time.Hour)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("se esperaba una cookie y hay %d", len(cookies))
	}
	c := cookies[0]
	if !c.HttpOnly {
		t.Error("la cookie de sesion no lleva HttpOnly: cualquier script la lee")
	}
	if !c.Secure {
		t.Error("la cookie de sesion no lleva Secure: viaja en claro por http")
	}
	if c.SameSite != http.SameSiteLaxMode && c.SameSite != http.SameSiteStrictMode {
		t.Errorf("la cookie de sesion lleva SameSite=%v: otra pagina puede arrastrarla", c.SameSite)
	}
	if c.Domain != "" {
		t.Errorf("la cookie declara Domain=%q: viajaria a todos los subdominios y uno "+
			"comprometido se lleva la sesion", c.Domain)
	}
	if !strings.HasPrefix(c.Name, "__Host-") {
		t.Errorf("la cookie se llama %q y no lleva el prefijo __Host-, que es lo que hace "+
			"que el navegador impida plantarla desde un subdominio", c.Name)
	}
	// Control negativo: con CookieInsegura el nombre TIENE que cambiar, porque
	// __Host- sin Secure lo tira el navegador sin decir nada.
	si := servidorDePrueba(t, Config{CookieInsegura: true,
		HayAdmin: func(context.Context) (bool, error) { return true, nil }})
	rec2 := httptest.NewRecorder()
	si.ponerCookie(rec2, "x", time.Hour)
	c2 := rec2.Result().Cookies()[0]
	if c2.Secure {
		t.Error("con CookieInsegura la cookie sigue llevando Secure")
	}
	if strings.HasPrefix(c2.Name, "__Host-") {
		t.Error("con CookieInsegura la cookie conserva el prefijo __Host-, que exige " +
			"Secure: el navegador la tiraria y el operador veria un bucle sin mensaje")
	}
}

// --- rate limit ---

func TestElRateLimitDeAutenticacionCortaDespuesDeNIntentos(t *testing.T) {
	reloj := &relojControlado{t: time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)}
	s := servidorDePrueba(t, Config{
		Reloj:      reloj.Ahora,
		LimiteAuth: Limite{Maximo: 3, Ventana: 5 * time.Minute},
		HayAdmin:   func(context.Context) (bool, error) { return true, nil },
	})
	for i := 1; i <= 3; i++ {
		if rec := pedir(s, http.MethodPost, "/entrar", "", nil); rec.Code == http.StatusTooManyRequests {
			t.Fatalf("el intento %d de 3 ya ha sido cortado", i)
		}
	}
	rec := pedir(s, http.MethodPost, "/entrar", "", nil)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("el cuarto intento de autenticacion respondio %d y tenia que ser 429", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Error("el 429 no dice cuando volver a intentarlo")
	}
	// Pasada la ventana, se vuelve a atender: un corte que no se levanta es
	// una denegacion de servicio permanente.
	reloj.Avanzar(6 * time.Minute)
	if rec := pedir(s, http.MethodPost, "/entrar", "", nil); rec.Code == http.StatusTooManyRequests {
		t.Fatal("pasada la ventana el rate limit sigue cortando")
	}
}

// El rate limit general NO puede confundirse con el de autenticacion: si una
// pantalla normal consumiera el cubo estrecho, el operador se quedaria fuera
// por navegar.
func TestElCuboGeneralYElDeAutenticacionSonDistintos(t *testing.T) {
	s := servidorDePrueba(t, Config{
		LimiteAuth:    Limite{Maximo: 2, Ventana: time.Minute},
		LimiteGeneral: Limite{Maximo: 100, Ventana: time.Minute},
		HayAdmin:      func(context.Context) (bool, error) { return true, nil },
	})
	for i := 0; i < 20; i++ {
		if rec := pedir(s, http.MethodGet, "/salud", "", nil); rec.Code != http.StatusOK {
			t.Fatalf("la peticion general %d respondio %d: el trafico normal esta cayendo "+
				"en el cubo de autenticacion", i, rec.Code)
		}
	}
	if rec := pedir(s, http.MethodPost, "/entrar", "", nil); rec.Code == http.StatusTooManyRequests {
		t.Fatal("el primer POST a /entrar ya viene cortado: los cubos estan mezclados")
	}
}

// X-Forwarded-For inventado. Sin proxy declarado, esa cabecera NO se mira: si
// se mirara, un atacante estrena cubo en cada intento y el rate limit no existe.
func TestElRateLimitNoSeSaltaInventandoXForwardedFor(t *testing.T) {
	s := servidorDePrueba(t, Config{
		LimiteAuth: Limite{Maximo: 3, Ventana: 5 * time.Minute},
		HayAdmin:   func(context.Context) (bool, error) { return true, nil },
	})
	var ultimo int
	for i := 0; i < 10; i++ {
		rec := pedir(s, http.MethodPost, "/entrar", "", map[string]string{
			"X-Forwarded-For": "10.0.0." + string(rune('0'+i)),
			"X-Real-IP":       "10.1.0." + string(rune('0'+i)),
			"Forwarded":       "for=10.2.0." + string(rune('0'+i)),
		})
		ultimo = rec.Code
	}
	if ultimo != http.StatusTooManyRequests {
		t.Fatalf("tras 10 intentos con una cabecera de reenvio distinta cada vez, la "+
			"ultima respondio %d. Si X-Forwarded-For se cree por defecto, el rate limit "+
			"de autenticacion no existe: basta cambiar una cabecera", ultimo)
	}
}

// Con proxy declarado si se mira, pero contando desde la DERECHA: lo que el
// cliente pudo inventar queda a la izquierda y no se usa.
func TestConProxyDeConfianzaSeCuentaDesdeLaDerecha(t *testing.T) {
	casos := []struct {
		nombre  string
		proxies int
		xff     string
		clave   string
	}{
		{"un proxy, el cliente invento la primera", 1, "9.9.9.9, 203.0.113.7", "203.0.113.7"},
		{"dos proxies", 2, "9.9.9.9, 203.0.113.7, 10.0.0.1", "203.0.113.7"},
		{"vienen menos de las que deberia: se cae a la conexion real", 2, "203.0.113.7", "192.0.2.1"},
		{"sin proxies declarados no se mira", 0, "203.0.113.7", "192.0.2.1"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("X-Forwarded-For", c.xff)
			if got := ClaveCliente(r, c.proxies); got != c.clave {
				t.Fatalf("clave %q, se esperaba %q", got, c.clave)
			}
		})
	}
}

// Una direccion IPv6 sola no es un cliente: a un cliente domestico le sobra una
// red /64 entera. Limitar por direccion completa seria no limitar.
func TestLasDireccionesIPv6SeAgrupanPorPrefijo(t *testing.T) {
	a := httptest.NewRequest(http.MethodGet, "/", nil)
	a.RemoteAddr = "[2001:db8:1:2::1]:443"
	b := httptest.NewRequest(http.MethodGet, "/", nil)
	b.RemoteAddr = "[2001:db8:1:2::dead:beef]:443"
	otra := httptest.NewRequest(http.MethodGet, "/", nil)
	otra.RemoteAddr = "[2001:db8:1:3::1]:443"

	if ClaveCliente(a, 0) != ClaveCliente(b, 0) {
		t.Fatalf("dos direcciones del mismo /64 caen en cubos distintos (%s vs %s): "+
			"cambiar de direccion dentro de tu propia red esquivaria el rate limit",
			ClaveCliente(a, 0), ClaveCliente(b, 0))
	}
	if ClaveCliente(a, 0) == ClaveCliente(otra, 0) {
		t.Fatal("dos /64 distintos comparten cubo: el rate limit castigaria a terceros")
	}
}

// Control negativo del limitador: con la ventana deslizante rota (contar solo
// los aceptados), quien insiste recupera turno. Se comprueba que NO pasa.
func TestQuienInsisteNoRecuperaTurnoDentroDeLaVentana(t *testing.T) {
	l := NuevoLimitador(Limite{Maximo: 2, Ventana: time.Minute})
	base := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	if !l.Permitir("x", base) || !l.Permitir("x", base.Add(time.Second)) {
		t.Fatal("los dos primeros intentos tenian que pasar")
	}
	for i := 2; i < 30; i++ {
		if l.Permitir("x", base.Add(time.Duration(i)*time.Second)) {
			t.Fatalf("el intento %d ha pasado dentro de la ventana", i)
		}
	}
	// Al salir de la ventana desde el ULTIMO intento, se vuelve a atender.
	if !l.Permitir("x", base.Add(2*time.Minute)) {
		t.Fatal("pasada la ventana entera, el cubo sigue cerrado")
	}
}

// --- metodos, host, cuerpo, panic ---

func TestTRACEyCONNECTNoLleganANingunHandler(t *testing.T) {
	llego := false
	app := NuevoEnrutador()
	app.Manejar("", "/app/", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		llego = true
		w.WriteHeader(http.StatusOK)
	}))
	s, sesion := servidorConSesion(t, Config{App: app})
	for _, m := range []string{http.MethodTrace, http.MethodConnect} {
		rec := pedir(s, m, "/app/x", sesion, nil)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s respondio %d y tenia que ser 405", m, rec.Code)
		}
	}
	if llego {
		t.Fatal("TRACE o CONNECT han llegado al handler. TRACE devuelve la peticion " +
			"entera, cookies incluidas, que es como se roba una cookie HttpOnly")
	}
}

func TestUnHostFalsificadoSeRechazaYNoSeRefleja(t *testing.T) {
	s := servidorDePrueba(t, Config{
		HostsPermitidos: []string{"grc.ejemplo.es", "localhost:8443"},
		HayAdmin:        func(context.Context) (bool, error) { return true, nil },
	})
	r := httptest.NewRequest(http.MethodGet, "/salud", nil)
	r.Host = "atacante.example"
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, r)
	if rec.Code != http.StatusMisdirectedRequest {
		t.Fatalf("un Host que no esta en la lista respondio %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "atacante.example") {
		t.Fatal("la respuesta refleja el Host recibido: es devolverle al atacante su " +
			"propia cadena")
	}
	// Y el host legitimo sigue entrando, con y sin puerto.
	for _, host := range []string{"grc.ejemplo.es", "grc.ejemplo.es:8443", "localhost:9999"} {
		r := httptest.NewRequest(http.MethodGet, "/salud", nil)
		r.Host = host
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, r)
		if rec.Code != http.StatusOK {
			t.Errorf("el host legitimo %q respondio %d", host, rec.Code)
		}
	}
}

func TestUnCuerpoEnormeSeCortaAntesDeLlegarAlHandler(t *testing.T) {
	var leidos int
	app := NuevoEnrutador()
	app.Manejar(http.MethodPost, "/app/tragar", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, 4096)
		for {
			n, err := r.Body.Read(b)
			leidos += n
			if err != nil {
				break
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	s, sesion := servidorConSesion(t, Config{App: app, MaxCuerpo: 1024})
	tok, err := s.ses.TokenCSRF(context.Background(), sesion)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/app/tragar", strings.NewReader(strings.Repeat("a", 100000)))
	req.Header.Set(CabeceraCSRF, tok)
	req.AddCookie(&http.Cookie{Name: s.nombreCookie(), Value: sesion})
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if leidos > 1024 {
		t.Fatalf("el handler ha leido %d bytes con el tope en 1024: un POST de un giga se "+
			"come la memoria antes de que nadie decida nada", leidos)
	}
}

func TestUnPanicDevuelveQuinientosYNoFiltraLaTraza(t *testing.T) {
	app := NuevoEnrutador()
	app.Manejar(http.MethodGet, "/app/panico", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("secreto-que-no-debe-salir")
	}))
	registro := &strings.Builder{}
	s, sesion := servidorConSesion(t, Config{App: app, Salida: registro})
	rec := pedir(s, http.MethodGet, "/app/panico", sesion, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("un panic respondio %d y tenia que ser 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "secreto-que-no-debe-salir") {
		t.Fatal("el detalle del panic ha viajado al navegador: es entregarle al atacante " +
			"el mapa del proceso")
	}
	if !strings.Contains(registro.String(), "secreto-que-no-debe-salir") {
		t.Fatal("el panic no se ha registrado en la salida del servidor: el operador se " +
			"queda sin saber que ha pasado")
	}
}

// --- origen ---

func TestUnaPeticionMutanteDesdeOtroOrigenSeRechazaAunqueTraigaTokenValido(t *testing.T) {
	s, sesion := servidorConSesion(t, Config{App: aplicacionDePrueba()})
	tok, err := s.ses.TokenCSRF(context.Background(), sesion)
	if err != nil {
		t.Fatal(err)
	}
	for _, cab := range []map[string]string{
		{"Origin": "https://atacante.example"},
		{"Sec-Fetch-Site": "cross-site"},
	} {
		cab[CabeceraCSRF] = tok
		rec := pedir(s, http.MethodPost, "/app/guardar", sesion, cab)
		if rec.Code != http.StatusForbidden {
			t.Errorf("con %v la peticion respondio %d y tenia que ser 403", cab, rec.Code)
		}
	}
	// Con el origen correcto si pasa.
	rec := pedir(s, http.MethodPost, "/app/guardar", sesion, map[string]string{
		"Origin":     "http://example.com",
		CabeceraCSRF: tok,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("con el Origin del propio host la peticion respondio %d", rec.Code)
	}
}

// El token de OTRA sesion no vale ni con cookie valida. Es el contrato de
// Sesion, pero comprobado a traves del middleware: es ahi donde importa.
func TestElTokenDeOtraSesionNoValeATravesDelMiddleware(t *testing.T) {
	s, victima := servidorConSesion(t, Config{App: aplicacionDePrueba()})
	atacante, err := s.ses.Abrir(context.Background(), "atacante", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	tokAtacante, err := s.ses.TokenCSRF(context.Background(), atacante)
	if err != nil {
		t.Fatal(err)
	}
	rec := pedir(s, http.MethodPost, "/app/guardar", victima,
		map[string]string{CabeceraCSRF: tokAtacante})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("el token de otra sesion ha valido (%d) en la sesion de la victima", rec.Code)
	}
}

// Los dos caminos en los que la pieza esta a medio construir. Fallan CERRADOS:
// un middleware al que le falta una dependencia y deja pasar es peor que no
// tenerlo, porque parece que esta.
func TestLasPiezasAMedioConstruirFallanCerradas(t *testing.T) {
	t.Run("CSRF sin almacen de sesiones no atiende nada que mute", func(t *testing.T) {
		h := ProtectorCSRF{Cookie: CookieSesion}.Envolver(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				t.Error("la peticion mutante ha llegado al handler sin comprobar CSRF")
				w.WriteHeader(http.StatusOK)
			}))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/lo-que-sea", nil))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("respondio %d y tenia que ser 500", rec.Code)
		}
	})

	t.Run("el rate limit sin funcion de clave sigue limitando", func(t *testing.T) {
		l := Limitadores{
			General: NuevoLimitador(Limite{Maximo: 2, Ventana: time.Minute}),
			// Clave a nil: es el olvido que se quiere cubrir.
		}
		h := l.Envolver(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		var ultimo int
		for i := 0; i < 4; i++ {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
			ultimo = rec.Code
		}
		if ultimo != http.StatusTooManyRequests {
			t.Fatalf("con Clave a nil la cuarta peticion respondio %d: el limitador no "+
				"limita, y aun asi parece que esta puesto", ultimo)
		}
	})
}

// --- estaticos ---

func TestLosEstaticosNiListanNiSalenDelArbol(t *testing.T) {
	fsys := fstest.MapFS{
		"htmx.min.js":     {Data: []byte("/* htmx */")},
		"sub/interno.css": {Data: []byte("body{}")},
	}
	h := Estaticos("/estatico/", fsys)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/estatico/htmx.min.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("el fichero embebido respondio %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Errorf("Content-Type %q: con nosniff puesto, un tipo mal declarado deja de "+
			"cargar en el navegador", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Errorf("Cache-Control %q: sin cacheo largo, htmx se baja en cada visita", cc)
	}

	for _, ruta := range []string{
		"/estatico/",
		"/estatico/sub",
		"/estatico/sub/",
		"/estatico/../servidor.go",
		"/estatico/..%2fservidor.go",
		"/estatico/no-existe.js",
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ruta, nil))
		if rec.Code == http.StatusOK {
			t.Errorf("%s ha devuelto 200: o lista directorio o sale del arbol", ruta)
		}
	}
	// Y no se escribe con ellos.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/estatico/htmx.min.js", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST a un estatico respondio %d", rec.Code)
	}
}

// --- utilidades ---

func pedir(s *Servidor, metodo, ruta, sesion string, cabeceras map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(metodo, ruta, strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if sesion != "" {
		req.AddCookie(&http.Cookie{Name: s.nombreCookie(), Value: sesion})
	}
	for k, v := range cabeceras {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	return rec
}

type relojControlado struct {
	mu sync.Mutex
	t  time.Time
}

func (r *relojControlado) Ahora() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.t
}

func (r *relojControlado) Avanzar(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.t = r.t.Add(d)
}
