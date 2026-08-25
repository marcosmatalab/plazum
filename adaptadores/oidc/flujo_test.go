package oidc

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"dutiq/puertos"
	"dutiq/puertos/contrato"
)

// TestElDobleDeSesionCumpleElContrato. El doble de este paquete no es un
// maniqui: pasa la misma suite que exigira cualquier implementacion de verdad.
//
// Sin esto, los tests del flujo demostrarian que se abre "algo", no que se abre
// una sesion. Y esta escrito aqui, no importado del test de otro frente: dos
// frentes que comparten un paquete de test se atan por el peor sitio.
func TestElDobleDeSesionCumpleElContrato(t *testing.T) {
	contrato.Sesion(t, func() puertos.Sesion { return nuevaSesionMemoria() })
}

// autenticadorDePrueba monta el flujo completo contra el IdP falso.
func autenticadorDePrueba(t *testing.T, i *idpFalso) (*Autenticador, *sesionMemoria) {
	t.Helper()
	ses := nuevaSesionMemoria()
	a, err := NuevoAutenticador(context.Background(), configPrueba(i), ses,
		&secretosDeterministas{}, Opciones{Cliente: ClientePorDefecto()})
	if err != nil {
		t.Fatalf("construir el autenticador: %v", err)
	}
	return a, ses
}

// iniciar arranca un flujo y devuelve los parametros con los que se mando al
// navegador.
func iniciar(t *testing.T, a *Autenticador, destino string) url.Values {
	t.Helper()
	u, err := a.Iniciar(context.Background(), ahoraFijo, destino)
	if err != nil {
		t.Fatalf("iniciar: %v", err)
	}
	p, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}
	return p.Query()
}

// TestElFlujoCompletoAbreSesion es el control negativo de todo este fichero: el
// camino honrado tiene que funcionar de punta a punta.
func TestElFlujoCompletoAbreSesion(t *testing.T) {
	i := nuevoIdP(t)
	a, ses := autenticadorDePrueba(t, i)
	q := iniciar(t, a, "/hoy")

	// El IdP acuna el ID token con el nonce que se le mando.
	cuerpo := i.cuerpoBueno(clientePrueba)
	cuerpo["nonce"] = q.Get("nonce")
	i.mu.Lock()
	i.idTokenCanje = i.acunar(t, map[string]any{"alg": "RS256", "kid": i.kid, "typ": "JWT"}, cuerpo)
	i.mu.Unlock()

	idSesion, id, destino, err := a.Retorno(context.Background(), ahoraFijo, url.Values{
		"state": {q.Get("state")}, "code": {"codigo-del-idp"},
	})
	if err != nil {
		t.Fatalf("el retorno honrado tiene que funcionar: %v", err)
	}
	if idSesion == "" {
		t.Fatal("no se abrio sesion")
	}
	if id.Sujeto != "usuario-0001" {
		t.Errorf("sujeto %q", id.Sujeto)
	}
	if destino != "/hoy" {
		t.Errorf("destino %q, se esperaba /hoy", destino)
	}
	suj, err := ses.Leer(context.Background(), idSesion)
	if err != nil {
		t.Fatalf("la sesion abierta tiene que leerse: %v", err)
	}
	if suj != "usuario-0001" {
		t.Errorf("la sesion se abrio para %q y no para el `sub` del IdP", suj)
	}
}

// TestPKCEViajaYEsS256. PKCE, incluso con secreto de cliente.
func TestPKCEViajaYEsS256(t *testing.T) {
	i := nuevoIdP(t)
	cfg := configPrueba(i)
	cfg.ClienteSecreto = "hay-secreto-y-aun-asi-pkce"
	a, err := NuevoAutenticador(context.Background(), cfg, nuevaSesionMemoria(),
		&secretosDeterministas{}, Opciones{Cliente: ClientePorDefecto()})
	if err != nil {
		t.Fatal(err)
	}
	q := iniciar(t, a, "/")
	if q.Get("code_challenge_method") != "S256" {
		t.Fatalf("code_challenge_method %q; `plain` manda el verificador en claro por la "+
			"misma URL y no protege de nada", q.Get("code_challenge_method"))
	}
	if q.Get("code_challenge") == "" {
		t.Fatal("no se mando code_challenge: sin el, PKCE es un adorno")
	}

	cuerpo := i.cuerpoBueno(clientePrueba)
	cuerpo["nonce"] = q.Get("nonce")
	i.mu.Lock()
	i.idTokenCanje = i.acunar(t, map[string]any{"alg": "RS256", "kid": i.kid, "typ": "JWT"}, cuerpo)
	i.mu.Unlock()

	if _, _, _, err := a.Retorno(context.Background(), ahoraFijo, url.Values{
		"state": {q.Get("state")}, "code": {"c"},
	}); err != nil {
		t.Fatalf("retorno: %v", err)
	}
	i.mu.Lock()
	verificador := i.verificadorRecibido
	i.mu.Unlock()
	if verificador == "" {
		t.Fatal("el canje no llevo code_verifier: el desafio del inicio no se puede comprobar")
	}
	if DesafioPKCE(verificador) != q.Get("code_challenge") {
		t.Fatalf("el verificador del canje no casa con el desafio del inicio. PKCE ata las dos " +
			"mitades del flujo, y si no casan es que no ata nada")
	}
	if len(verificador) < 43 || len(verificador) > 128 {
		t.Errorf("el verificador PKCE ocupa %d caracteres y RFC 7636 pide entre 43 y 128",
			len(verificador))
	}
}

// TestElStateValeUnaVez. Sin un solo uso, el `state` no protege del CSRF del
// flujo: quien capture uno lo reinyecta.
func TestElStateValeUnaVez(t *testing.T) {
	i := nuevoIdP(t)
	a, _ := autenticadorDePrueba(t, i)
	q := iniciar(t, a, "/")

	cuerpo := i.cuerpoBueno(clientePrueba)
	cuerpo["nonce"] = q.Get("nonce")
	i.mu.Lock()
	i.idTokenCanje = i.acunar(t, map[string]any{"alg": "RS256", "kid": i.kid, "typ": "JWT"}, cuerpo)
	i.mu.Unlock()

	consulta := url.Values{"state": {q.Get("state")}, "code": {"c"}}
	if _, _, _, err := a.Retorno(context.Background(), ahoraFijo, consulta); err != nil {
		t.Fatalf("CONTROL NEGATIVO EN ROJO: el primer uso tiene que funcionar: %v", err)
	}
	_, _, _, err := a.Retorno(context.Background(), ahoraFijo, consulta)
	if err == nil {
		t.Fatal("el MISMO state se pudo usar dos veces. Un state reutilizable no protege del " +
			"CSRF del flujo: quien capture uno lo reinyecta")
	}
	if !errors.Is(err, ErrEstado) {
		t.Errorf("se rechazo por otro motivo: %v", err)
	}
}

// TestElNonceValeUnaVez: el nonce se va con el state, y por tanto tampoco se
// reutiliza. Se comprueba aparte porque son dos propiedades distintas: un
// diseno podria borrar el state y dejar el nonce vivo en otro sitio.
func TestElNonceValeUnaVez(t *testing.T) {
	i := nuevoIdP(t)
	a, _ := autenticadorDePrueba(t, i)
	q1 := iniciar(t, a, "/")
	q2 := iniciar(t, a, "/")
	if q1.Get("nonce") == q2.Get("nonce") {
		t.Fatal("dos flujos distintos con el mismo nonce: el nonce no ata el token a SU peticion")
	}
	if q1.Get("state") == q2.Get("state") {
		t.Fatal("dos flujos distintos con el mismo state")
	}

	// El token del flujo 1 no vale para cerrar el flujo 2.
	cuerpo := i.cuerpoBueno(clientePrueba)
	cuerpo["nonce"] = q1.Get("nonce")
	i.mu.Lock()
	i.idTokenCanje = i.acunar(t, map[string]any{"alg": "RS256", "kid": i.kid, "typ": "JWT"}, cuerpo)
	i.mu.Unlock()

	_, _, _, err := a.Retorno(context.Background(), ahoraFijo, url.Values{
		"state": {q2.Get("state")}, "code": {"c"},
	})
	if err == nil {
		t.Fatal("un ID token con el nonce de OTRO flujo cerro este. Es exactamente la " +
			"reinyeccion de la que protege el nonce")
	}
	if !strings.Contains(err.Error(), "no es el de esta peticion") {
		t.Errorf("se rechazo por otro motivo: %v", err)
	}
	// Control negativo: con su propio nonce, ese mismo token cierra el flujo 1.
	if _, _, _, err := a.Retorno(context.Background(), ahoraFijo, url.Values{
		"state": {q1.Get("state")}, "code": {"c"},
	}); err != nil {
		t.Fatalf("CONTROL NEGATIVO EN ROJO: el token con SU nonce tiene que cerrar SU flujo: %v", err)
	}
}

// TestUnStateCaducadoNoVale.
func TestUnStateCaducadoNoVale(t *testing.T) {
	i := nuevoIdP(t)
	ses := nuevaSesionMemoria()
	a, err := NuevoAutenticador(context.Background(), configPrueba(i), ses,
		&secretosDeterministas{}, Opciones{Cliente: ClientePorDefecto(), VidaPeticion: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	q := iniciar(t, a, "/")
	_, _, _, err = a.Retorno(context.Background(), ahoraFijo.Add(2*time.Minute), url.Values{
		"state": {q.Get("state")}, "code": {"c"},
	})
	if err == nil {
		t.Fatal("un state de hace dos minutos, con vida de uno, se acepto")
	}
	if !errors.Is(err, ErrEstado) {
		t.Errorf("se rechazo por otro motivo: %v", err)
	}
}

// TestUnRetornoSinStateNoEntra. Es la peticion que fabrica un atacante.
func TestUnRetornoSinStateNoEntra(t *testing.T) {
	i := nuevoIdP(t)
	a, ses := autenticadorDePrueba(t, i)
	for _, consulta := range []url.Values{
		{"code": {"codigo"}},
		{"state": {""}, "code": {"codigo"}},
		{"state": {"me-lo-invento"}, "code": {"codigo"}},
	} {
		if _, _, _, err := a.Retorno(context.Background(), ahoraFijo, consulta); err == nil {
			t.Fatalf("un retorno con %v abrio sesion", consulta)
		}
	}
	if n := len(ses.vivas); n != 0 {
		t.Fatalf("se abrieron %d sesiones desde retornos sin state valido", n)
	}
}

// TestLaRedirectURINuncaVieneDeLaPeticion.
//
// Es la comprobacion del redirector abierto. Si la redirect_uri o el destino
// salieran de la peticion, nuestro dominio se convierte en el trampolin de un
// enlace de phishing: la victima ve el dominio de su empresa y acaba en otro
// sitio.
func TestLaRedirectURINuncaVieneDeLaPeticion(t *testing.T) {
	i := nuevoIdP(t)
	a, _ := autenticadorDePrueba(t, i)

	// El inicio siempre manda la redirect_uri configurada, pase lo que pase.
	q := iniciar(t, a, "/")
	if q.Get("redirect_uri") != configPrueba(i).RedirectURI {
		t.Fatalf("redirect_uri %q, se esperaba la de la configuracion", q.Get("redirect_uri"))
	}

	// El destino de vuelta se sanea: nada que salga del sitio sobrevive.
	for entrada, esperado := range map[string]string{
		"/hoy":                         "/hoy",
		"/controles?filtro=abiertas":   "/controles?filtro=abiertas",
		"//evil.ejemplo/robar":         "/",
		"https://evil.ejemplo/robar":   "/",
		"http://evil.ejemplo":          "/",
		"/hoy\r\nSet-Cookie: a=b":      "/",
		"\\\\evil.ejemplo\\compartido": "/",
		"":                             "/",
	} {
		if got := destinoSeguro(entrada); got != esperado {
			t.Errorf("destinoSeguro(%q) = %q, se esperaba %q. Un destino que sale del sitio "+
				"convierte el login en un redirector abierto con nuestro dominio delante",
				entrada, got, esperado)
		}
	}

	// Y una redirect_uri metida en la peticion de retorno se ignora del todo:
	// el canje usa la configurada.
	cuerpo := i.cuerpoBueno(clientePrueba)
	cuerpo["nonce"] = q.Get("nonce")
	i.mu.Lock()
	i.idTokenCanje = i.acunar(t, map[string]any{"alg": "RS256", "kid": i.kid, "typ": "JWT"}, cuerpo)
	var recibida string
	i.respuestaToken = func(w http.ResponseWriter, r *http.Request) bool {
		_ = r.ParseForm()
		recibida = r.PostForm.Get("redirect_uri")
		return false
	}
	i.mu.Unlock()

	if _, _, destino, err := a.Retorno(context.Background(), ahoraFijo, url.Values{
		"state":        {q.Get("state")},
		"code":         {"c"},
		"redirect_uri": {"https://evil.ejemplo/robar"},
	}); err != nil {
		t.Fatalf("retorno: %v", err)
	} else if destino != "/" {
		t.Errorf("destino %q: el destino sale del state, no de la peticion", destino)
	}
	if recibida != configPrueba(i).RedirectURI {
		t.Fatalf("el canje mando redirect_uri %q. Tiene que ser la de la configuracion (%q): "+
			"si la peticion puede elegirla, el codigo se canjea contra el sitio del atacante",
			recibida, configPrueba(i).RedirectURI)
	}
}

// TestUnUsuarioNoAdmitidoNoAbreSesion. Es el gancho por el que un usuario
// desactivado en el aprovisionamiento deja de poder entrar aunque el IdP siga
// autenticandole.
func TestUnUsuarioNoAdmitidoNoAbreSesion(t *testing.T) {
	i := nuevoIdP(t)
	a, ses := autenticadorDePrueba(t, i)
	a.Admision = func(id Identidad) error {
		return errors.New("la cuenta de " + id.Sujeto + " esta desactivada")
	}
	q := iniciar(t, a, "/")
	cuerpo := i.cuerpoBueno(clientePrueba)
	cuerpo["nonce"] = q.Get("nonce")
	i.mu.Lock()
	i.idTokenCanje = i.acunar(t, map[string]any{"alg": "RS256", "kid": i.kid, "typ": "JWT"}, cuerpo)
	i.mu.Unlock()

	_, _, _, err := a.Retorno(context.Background(), ahoraFijo, url.Values{
		"state": {q.Get("state")}, "code": {"c"},
	})
	if err == nil {
		t.Fatal("un usuario rechazado por la admision abrio sesion")
	}
	if !errors.Is(err, ErrAdmision) {
		t.Errorf("el error tiene que distinguirse del token invalido: %v", err)
	}
	if len(ses.vivas) != 0 {
		t.Fatal("se abrio sesion pese al rechazo: la admision corre DESPUES de verificar y " +
			"ANTES de abrir, y aqui no fue asi")
	}
}

// TestUnIdPSinS256NoSeAcepta. Si el IdP declara que no hace S256, el flujo no
// puede dar la garantia que promete.
func TestUnIdPSinS256NoSeAcepta(t *testing.T) {
	i := nuevoIdP(t)
	i.mu.Lock()
	i.sinPKCE = true
	i.mu.Unlock()
	_, err := NuevoAutenticador(context.Background(), configPrueba(i), nuevaSesionMemoria(),
		&secretosDeterministas{}, Opciones{Cliente: ClientePorDefecto()})
	if err == nil {
		t.Fatal("un IdP que declara solo `plain` se acepto")
	}
	if !strings.Contains(err.Error(), "S256") {
		t.Errorf("el error no dice cual es el problema: %v", err)
	}
}

// TestUnaConfiguracionMalPegadaRompeAlArrancar. Cada mensaje tiene que decir
// donde se corrige, porque quien lo lee es el administrador de sistemas de una
// empresa de 200 personas un martes por la tarde.
func TestUnaConfiguracionMalPegadaRompeAlArrancar(t *testing.T) {
	base := Configuracion{
		Emisor:      "https://login.ejemplo/tenant/v2.0",
		ClienteID:   "cliente",
		RedirectURI: "https://dutiq.ejemplo/auth/retorno",
	}
	casos := []struct {
		nombre string
		toca   func(*Configuracion)
		pista  string
	}{
		{"sin emisor", func(c *Configuracion) { c.Emisor = "" }, "issuer"},
		{"emisor que no es URL", func(c *Configuracion) { c.Emisor = "login.ejemplo" }, "no es una URL"},
		{"emisor sin TLS", func(c *Configuracion) { c.Emisor = "http://login.ejemplo" }, "no usa https"},
		{"emisor con query", func(c *Configuracion) { c.Emisor += "?a=b" }, "query o fragmento"},
		{"sin cliente_id", func(c *Configuracion) { c.ClienteID = "" }, "Application (client) ID"},
		{"sin redirect_uri", func(c *Configuracion) { c.RedirectURI = "" }, "caracter a caracter"},
		{"redirect_uri relativa", func(c *Configuracion) { c.RedirectURI = "/auth/retorno" }, "URL absoluta"},
		{"redirect_uri con fragmento", func(c *Configuracion) { c.RedirectURI += "#x" }, "fragmento"},
		{"algoritmo HMAC", func(c *Configuracion) { c.Algoritmos = []string{"HS256"} }, "clave PUBLICA"},
		{"margen negativo", func(c *Configuracion) { c.MargenReloj = -time.Second }, "negativo"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			cfg := base
			c.toca(&cfg)
			err := cfg.validar()
			if err == nil {
				t.Fatal("esta configuracion tenia que romper al arrancar y no al primer login")
			}
			if !errors.Is(err, ErrConfiguracion) {
				t.Errorf("no es un ErrConfiguracion: %v", err)
			}
			if !strings.Contains(err.Error(), c.pista) {
				t.Errorf("el mensaje no dice donde se arregla. Dice: %v\nSe esperaba que "+
					"mencionara %q", err, c.pista)
			}
		})
	}
	// Control negativo: la configuracion buena valida.
	if err := base.validar(); err != nil {
		t.Fatalf("CONTROL NEGATIVO EN ROJO: una configuracion correcta se rechaza: %v", err)
	}
}

// TestLosFlujosAMediasNoCrecenSinLimite. Pulsar entrar en bucle sin completar
// nunca el flujo no puede ser una fuga de memoria remota y gratuita.
func TestLosFlujosAMediasNoCrecenSinLimite(t *testing.T) {
	p := NuevasPendientes(time.Minute)
	for n := 0; n < MaxPeticionesEnVuelo+50; n++ {
		err := p.Guardar(strings.Repeat("x", 3)+string(rune('a'+n%26))+itoa(n), peticion{
			creada: ahoraFijo,
		}, ahoraFijo)
		if err != nil {
			if n < MaxPeticionesEnVuelo {
				t.Fatalf("se rechazo el flujo %d, por debajo del maximo: %v", n, err)
			}
			break
		}
	}
	if p.EnVuelo() > MaxPeticionesEnVuelo {
		t.Fatalf("hay %d flujos a medias, por encima del maximo %d", p.EnVuelo(), MaxPeticionesEnVuelo)
	}
	// Y lo caducado se poda: pasado el tiempo, vuelve a caber.
	if err := p.Guardar("uno-mas", peticion{creada: ahoraFijo.Add(2 * time.Minute)},
		ahoraFijo.Add(2*time.Minute)); err != nil {
		t.Fatalf("tras caducar los anteriores tenia que caber uno nuevo: %v", err)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
