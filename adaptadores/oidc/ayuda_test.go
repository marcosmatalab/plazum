package oidc

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// El instante fijo con el que corren los tests. Nada de time.Now: el instante
// entra como dato, igual que en el nucleo, y por eso se puede probar un token
// caducado sin dormir.
var ahoraFijo = time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)

// ---------------------------------------------------------------------------
// El IdP falso
// ---------------------------------------------------------------------------

// idpFalso levanta un emisor de OpenID Connect de mentira: publica su documento
// de descubrimiento, su JWKS, y acuna ID tokens que se pueden falsificar campo
// a campo.
//
// Existe para que cada test de esta carpeta pueda decir "falsifico ESTO" y nada
// mas, con el resto del token bien formado. Sin eso, un test que rechaza un
// token no demuestra que rechaza la falsificacion: puede estar rechazando por
// cualquier otro motivo.
type idpFalso struct {
	t   *testing.T
	srv *httptest.Server

	privRSA *rsa.PrivateKey
	privEC  *ecdsa.PrivateKey
	kid     string
	kidEC   string

	mu              sync.Mutex
	peticionesJWKS  int
	peticionesToken int
	// Ajustes que un test puede tocar para volver al IdP hostil.
	retardoJWKS    time.Duration
	clavesExtra    int
	ocultarClaves  bool
	cuerpoJWKS     string
	respuestaToken func(w http.ResponseWriter, r *http.Request) bool
	sinPKCE        bool
	// idTokenCanje es lo que devuelve el token endpoint. Si esta vacio, se
	// acuna uno bueno con el nonce que se guarde en nonceCanje.
	idTokenCanje string
	// verificadorRecibido guarda el code_verifier que llego al canje, para
	// comprobar que PKCE viaja de verdad.
	verificadorRecibido  string
	autorizacionRecibida map[string]string
}

func nuevoIdP(t *testing.T) *idpFalso {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	privEC, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	i := &idpFalso{
		t:       t,
		privRSA: priv,
		privEC:  privEC,
		kid:     "clave-rsa-1",
		kidEC:   "clave-ec-1",
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", i.descubrimiento)
	mux.HandleFunc("/jwks", i.jwks)
	mux.HandleFunc("/token", i.token)
	mux.HandleFunc("/autorizar", func(w http.ResponseWriter, r *http.Request) {
		i.mu.Lock()
		i.autorizacionRecibida = map[string]string{}
		for k := range r.URL.Query() {
			i.autorizacionRecibida[k] = r.URL.Query().Get(k)
		}
		i.mu.Unlock()
	})
	i.srv = httptest.NewServer(mux)
	t.Cleanup(i.srv.Close)
	return i
}

func (i *idpFalso) emisor() string { return i.srv.URL }

func (i *idpFalso) descubrimiento(w http.ResponseWriter, _ *http.Request) {
	d := map[string]any{
		"issuer":                                i.srv.URL,
		"authorization_endpoint":                i.srv.URL + "/autorizar",
		"token_endpoint":                        i.srv.URL + "/token",
		"jwks_uri":                              i.srv.URL + "/jwks",
		"id_token_signing_alg_values_supported": []string{"RS256", "ES256"},
	}
	i.mu.Lock()
	sin := i.sinPKCE
	i.mu.Unlock()
	if sin {
		d["code_challenge_methods_supported"] = []string{"plain"}
	} else {
		d["code_challenge_methods_supported"] = []string{"S256"}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(d)
}

func (i *idpFalso) jwks(w http.ResponseWriter, _ *http.Request) {
	i.mu.Lock()
	i.peticionesJWKS++
	retardo := i.retardoJWKS
	extra := i.clavesExtra
	ocultar := i.ocultarClaves
	crudo := i.cuerpoJWKS
	i.mu.Unlock()

	if retardo > 0 {
		time.Sleep(retardo)
	}
	if crudo != "" {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(crudo))
		return
	}
	var claves []map[string]string
	if !ocultar {
		claves = append(claves, jwkRSA(i.kid, &i.privRSA.PublicKey))
		claves = append(claves, jwkEC(i.kidEC, &i.privEC.PublicKey))
	}
	for n := 0; n < extra; n++ {
		claves = append(claves, jwkRSA(fmt.Sprintf("relleno-%d", n), &i.privRSA.PublicKey))
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"keys": claves})
}

func (i *idpFalso) token(w http.ResponseWriter, r *http.Request) {
	i.mu.Lock()
	i.peticionesToken++
	fn := i.respuestaToken
	i.mu.Unlock()
	if fn != nil && fn(w, r) {
		return
	}
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	i.mu.Lock()
	i.verificadorRecibido = r.PostForm.Get("code_verifier")
	tok := i.idTokenCanje
	i.mu.Unlock()
	if tok == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid_grant", "error_description": "el test no preparo ningun id_token",
		})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"id_token": tok, "token_type": "Bearer", "access_token": "no-se-usa",
	})
}

func (i *idpFalso) llamadasJWKS() int {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.peticionesJWKS
}

func jwkRSA(kid string, pub *rsa.PublicKey) map[string]string {
	e := big.NewInt(int64(pub.E)).Bytes()
	return map[string]string{
		"kty": "RSA", "kid": kid, "use": "sig", "alg": "RS256",
		"n": base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(e),
	}
}

func jwkEC(kid string, pub *ecdsa.PublicKey) map[string]string {
	tam := (pub.Curve.Params().BitSize + 7) / 8
	return map[string]string{
		"kty": "EC", "kid": kid, "use": "sig", "alg": "ES256", "crv": "P-256",
		"x": base64.RawURLEncoding.EncodeToString(rellenar(pub.X.Bytes(), tam)),
		"y": base64.RawURLEncoding.EncodeToString(rellenar(pub.Y.Bytes(), tam)),
	}
}

func rellenar(b []byte, n int) []byte {
	if len(b) >= n {
		return b
	}
	out := make([]byte, n)
	copy(out[n-len(b):], b)
	return out
}

// cuerpoBueno es el ID token honrado del que parten TODOS los tests de
// falsificacion. Que exista uno solo es lo que hace que el control negativo
// signifique algo: se falsifica un campo y lo demas queda igual.
func (i *idpFalso) cuerpoBueno(clienteID string) map[string]any {
	return map[string]any{
		"iss":            i.emisor(),
		"sub":            "usuario-0001",
		"aud":            clienteID,
		"exp":            ahoraFijo.Add(time.Hour).Unix(),
		"iat":            ahoraFijo.Add(-time.Minute).Unix(),
		"nonce":          "nonce-de-la-peticion",
		"email":          "ana@ejemplo.es",
		"email_verified": true,
		"name":           "Ana Ejemplo",
	}
}

// acunar firma un token con la cabecera y el cuerpo que le den. Es
// deliberadamente tonta: no valida nada, porque su trabajo es producir tokens
// invalidos.
func (i *idpFalso) acunar(t *testing.T, cab, cuerpo map[string]any) string {
	t.Helper()
	alg, _ := cab["alg"].(string)
	cb, err := json.Marshal(cab)
	if err != nil {
		t.Fatal(err)
	}
	pb, err := json.Marshal(cuerpo)
	if err != nil {
		t.Fatal(err)
	}
	firmado := base64.RawURLEncoding.EncodeToString(cb) + "." + base64.RawURLEncoding.EncodeToString(pb)
	var firma []byte
	switch alg {
	case "none", "":
		firma = nil
	case "RS256":
		h := sha256.Sum256([]byte(firmado))
		firma, err = rsa.SignPKCS1v15(rand.Reader, i.privRSA, crypto.SHA256, h[:])
		if err != nil {
			t.Fatal(err)
		}
	case "ES256":
		h := sha256.Sum256([]byte(firmado))
		r, s, err := ecdsa.Sign(rand.Reader, i.privEC, h[:])
		if err != nil {
			t.Fatal(err)
		}
		firma = append(rellenar(r.Bytes(), 32), rellenar(s.Bytes(), 32)...)
	case "HS256":
		// El ataque de confusion de algoritmo: se firma HMAC usando como
		// secreto la clave PUBLICA del IdP, que es publica.
		firma = hmacSHA256(i.privRSA.PublicKey.N.Bytes(), []byte(firmado))
	default:
		t.Fatalf("el IdP falso no sabe acunar con %q", alg)
	}
	return firmado + "." + base64.RawURLEncoding.EncodeToString(firma)
}

// bueno acuna el token honrado completo: cabecera RS256 con kid conocido y el
// cuerpo de cuerpoBueno.
func (i *idpFalso) bueno(t *testing.T, clienteID string) string {
	t.Helper()
	return i.acunar(t, map[string]any{"alg": "RS256", "kid": i.kid, "typ": "JWT"}, i.cuerpoBueno(clienteID))
}

// ---------------------------------------------------------------------------
// Los dobles de los puertos
// ---------------------------------------------------------------------------

// sesionMemoria es una implementacion de puertos.Sesion en memoria, escrita
// aqui a proposito: importar el doble del test de otro frente ataria dos
// frentes por el peor sitio posible.
//
// Que sea una de verdad y no un maniqui se demuestra pasandole la suite de
// contrato en TestElDobleDeSesionCumpleElContrato.
type sesionMemoria struct {
	mu    sync.Mutex
	n     int
	vivas map[string]entradaSesion
}

type entradaSesion struct {
	sujeto string
	hasta  time.Time
	csrf   map[string]bool
}

func nuevaSesionMemoria() *sesionMemoria {
	return &sesionMemoria{vivas: map[string]entradaSesion{}}
}

func (s *sesionMemoria) Abrir(_ context.Context, sujeto string, d time.Duration) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.n++
	id := fmt.Sprintf("sesion-%d-%s", s.n, sujeto)
	s.vivas[id] = entradaSesion{sujeto: sujeto, hasta: ahoraFijo.Add(d), csrf: map[string]bool{}}
	return id, nil
}

func (s *sesionMemoria) Leer(_ context.Context, id string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.vivas[id]
	if !ok || !ahoraFijo.Before(e.hasta) {
		return "", errors.New("sesion inexistente, caducada o cerrada")
	}
	return e.sujeto, nil
}

func (s *sesionMemoria) Cerrar(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.vivas, id)
	return nil
}

func (s *sesionMemoria) TokenCSRF(_ context.Context, id string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.vivas[id]
	if !ok {
		return "", errors.New("no hay sesion a la que atar el token")
	}
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	tok := hex.EncodeToString(b[:])
	e.csrf[tok] = true
	s.vivas[id] = e
	return tok, nil
}

func (s *sesionMemoria) ComprobarCSRF(_ context.Context, id, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.vivas[id]
	if !ok {
		return errors.New("no hay sesion")
	}
	for t := range e.csrf {
		if subtle.ConstantTimeCompare([]byte(t), []byte(token)) == 1 {
			return nil
		}
	}
	return errors.New("token CSRF que no es de esta sesion")
}

// secretosDeterministas es un puertos.Secretos reproducible. Solo existe en los
// tests, que es exactamente el motivo por el que la aleatoriedad es un puerto.
type secretosDeterministas struct {
	mu sync.Mutex
	n  uint64
}

func (s *secretosDeterministas) Bytes(b []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := 0; i < len(b); {
		s.n++
		var sem [8]byte
		binary.BigEndian.PutUint64(sem[:], s.n)
		h := sha256.Sum256(sem[:])
		i += copy(b[i:], h[:])
	}
	return nil
}

func (s *secretosDeterministas) Token(n int) (string, error) {
	b := make([]byte, n)
	if err := s.Bytes(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// hmacSHA256 existe solo para acunar el token del ataque de confusion de
// algoritmo. No se usa en produccion: las familias HMAC no estan en la lista
// blanca, y este helper es lo que permite demostrarlo.
func hmacSHA256(clave, mensaje []byte) []byte {
	h := hmac.New(sha256.New, clave)
	h.Write(mensaje)
	return h.Sum(nil)
}
