package tsa

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/hex"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/digitorus/timestamp"
)

// El instante que sella la TSA falsa. Fijo: el test tiene que ser reproducible.
var instanteSello = time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)

// pki es una CA de pruebas con su certificado de sellado. Generar RSA 2048
// cuesta, asi que se genera una vez por proceso y se reparte.
type pki struct {
	raiz     *x509.Certificate
	hoja     *x509.Certificate
	privHoja *rsa.PrivateKey
	pool     *x509.CertPool
}

var (
	pkiBuena     *pki
	pkiUnaVez    sync.Once
	pkiIntrusa   *pki
	intrusaUnVez sync.Once
)

func buena(t *testing.T) *pki {
	t.Helper()
	pkiUnaVez.Do(func() { pkiBuena = generarPKI(t, "dutiq test CA") })
	return pkiBuena
}

// intrusa es otra CA distinta, con la que se firman sellos que NO deben colar.
func intrusa(t *testing.T) *pki {
	t.Helper()
	intrusaUnVez.Do(func() { pkiIntrusa = generarPKI(t, "CA que nadie ha declarado") })
	return pkiIntrusa
}

func generarPKI(t *testing.T, nombre string) *pki {
	t.Helper()
	privRaiz, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	plantillaRaiz := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: nombre},
		NotBefore:             instanteSello.Add(-24 * time.Hour),
		NotAfter:              instanteSello.Add(24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	derRaiz, err := x509.CreateCertificate(rand.Reader, plantillaRaiz, plantillaRaiz,
		&privRaiz.PublicKey, privRaiz)
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := x509.ParseCertificate(derRaiz)
	if err != nil {
		t.Fatal(err)
	}

	privHoja, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	plantillaHoja := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: nombre + " sellado de tiempo"},
		NotBefore:    instanteSello.Add(-24 * time.Hour),
		NotAfter:     instanteSello.Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		// Sin esto el certificado no sirve para sellar tiempo, y la
		// verificacion lo tiene que rechazar.
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageTimeStamping},
		BasicConstraintsValid: true,
	}
	derHoja, err := x509.CreateCertificate(rand.Reader, plantillaHoja, raiz, &privHoja.PublicKey, privRaiz)
	if err != nil {
		t.Fatal(err)
	}
	hoja, err := x509.ParseCertificate(derHoja)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(raiz)
	return &pki{raiz: raiz, hoja: hoja, privHoja: privHoja, pool: pool}
}

// servidor levanta una TSA falsa que sella de verdad con la PKI dada.
// conCert a false produce un token SIN certificado, que es el caso que
// timestamp.Parse deja pasar sin comprobar ninguna firma.
func servidor(t *testing.T, p *pki, conCert bool) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		req, err := timestamp.ParseRequest(b)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ts := timestamp.Timestamp{
			HashAlgorithm:     req.HashAlgorithm,
			HashedMessage:     req.HashedMessage,
			Time:              instanteSello,
			Nonce:             req.Nonce,
			Policy:            asn1.ObjectIdentifier{1, 2, 3, 4, 1},
			SerialNumber:      big.NewInt(42),
			AddTSACertificate: conCert,
		}
		resp, err := ts.CreateResponse(p.hoja, p.privHoja)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/timestamp-reply")
		_, _ = w.Write(resp)
	}))
	t.Cleanup(s.Close)
	return s
}

// caida es una TSA que esta ahi pero no sirve.
func caida(t *testing.T, codigo int) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(codigo)
	}))
	t.Cleanup(s.Close)
	return s
}

func hashDe(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:]
}

func colaEn(t *testing.T) *Cola {
	t.Helper()
	return NuevaCola(filepath.Join(t.TempDir(), "anclajes.json"))
}

// --- Camino feliz ---

func TestSellaYVerifica(t *testing.T) {
	p := buena(t)
	s := servidor(t, p, true)
	c := &Cadena{
		Autoridades: []Autoridad{{Nombre: "falsa", URL: s.URL}},
		Anclas:      p.pool,
		Cola:        colaEn(t),
	}
	h := hashDe("raiz merkle del checkpoint")
	token, err := c.Sellar(h)
	if err != nil {
		t.Fatalf("la TSA responde, tenia que sellar: %v", err)
	}
	if err := c.VerificarOffline(h, token); err != nil {
		t.Fatalf("el token recien sellado tiene que verificar: %v", err)
	}
	inst, err := Instante(token)
	if err != nil {
		t.Fatal(err)
	}
	if !inst.Equal(instanteSello) {
		t.Fatalf("el sello dice %s y la TSA sello en %s", inst, instanteSello)
	}
	pend, err := c.Cola.Pendientes()
	if err != nil {
		t.Fatal(err)
	}
	if len(pend) != 0 {
		t.Fatalf("un sello conseguido no deja nada en la cola, hay %d", len(pend))
	}
}

// --- Cadena de reserva ---

func TestLaSegundaTSASalvaElAnclajeCuandoLaPrimeraCae(t *testing.T) {
	p := buena(t)
	mala := caida(t, http.StatusInternalServerError)
	s := servidor(t, p, true)
	c := &Cadena{
		Autoridades: []Autoridad{
			{Nombre: "la que se cayo", URL: mala.URL},
			{Nombre: "la de reserva", URL: s.URL},
		},
		Anclas: p.pool,
		Cola:   colaEn(t),
	}
	h := hashDe("checkpoint con la primera TSA caida")
	token, err := c.Sellar(h)
	if err != nil {
		t.Fatalf("con una TSA de reserva viva el anclaje no puede fallar: %v", err)
	}
	if err := c.VerificarOffline(h, token); err != nil {
		t.Fatalf("el token de la reserva tiene que verificar: %v", err)
	}
	pend, _ := c.Cola.Pendientes()
	if len(pend) != 0 {
		t.Fatal("si la reserva sello, no hay nada pendiente")
	}
}

// Una TSA que responde 200 y firma con una CA que no declaramos cuenta como
// caida: se pasa a la siguiente en vez de devolver un token inservible.
func TestUnaTSAQueFirmaConCAAjenaCuentaComoCaida(t *testing.T) {
	p := buena(t)
	otra := servidor(t, intrusa(t), true)
	s := servidor(t, p, true)
	c := &Cadena{
		Autoridades: []Autoridad{
			{Nombre: "responde pero no encadena", URL: otra.URL},
			{Nombre: "la buena", URL: s.URL},
		},
		Anclas: p.pool,
		Cola:   colaEn(t),
	}
	h := hashDe("checkpoint")
	token, err := c.Sellar(h)
	if err != nil {
		t.Fatalf("tenia que caer a la segunda: %v", err)
	}
	if err := c.VerificarOffline(h, token); err != nil {
		t.Fatalf("el token devuelto tiene que ser el bueno: %v", err)
	}
}

// --- Una TSA caida no bloquea nada ---

func TestConLasDosCaidasSeEncolaYNoEsUnFalloDuro(t *testing.T) {
	p := buena(t)
	c := &Cadena{
		Autoridades: []Autoridad{
			{Nombre: "caida 1", URL: caida(t, http.StatusInternalServerError).URL},
			{Nombre: "caida 2", URL: caida(t, http.StatusBadGateway).URL},
		},
		Anclas: p.pool,
		Cola:   colaEn(t),
		Ahora:  func() time.Time { return instanteSello },
	}
	h := hashDe("checkpoint sin TSA viva")
	token, err := c.Sellar(h)
	if token != nil {
		t.Fatal("sin TSA no hay token que devolver")
	}
	// La forma del error es lo que decide si el checkpoint sigue o se para.
	if !errors.Is(err, ErrEncolado) {
		t.Fatalf("tiene que ser ErrEncolado para que el llamante siga adelante, y fue: %v", err)
	}
	pend, err := c.Cola.Pendientes()
	if err != nil {
		t.Fatal(err)
	}
	if len(pend) != 1 {
		t.Fatalf("el anclaje tiene que quedar en la cola, hay %d", len(pend))
	}
}

func TestEncolarDosVecesElMismoHashNoLoDuplica(t *testing.T) {
	q := colaEn(t)
	h := hashDe("mismo checkpoint")
	for i := 0; i < 3; i++ {
		if err := q.Encolar(h, instanteSello); err != nil {
			t.Fatal(err)
		}
	}
	pend, _ := q.Pendientes()
	if len(pend) != 1 {
		t.Fatalf("el mismo checkpoint es un solo anclaje pendiente, hay %d", len(pend))
	}
}

func TestLaColaSobreviveAlReinicio(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "anclajes.json")
	h := hashDe("pendiente de antes del reinicio")
	if err := NuevaCola(ruta).Encolar(h, instanteSello); err != nil {
		t.Fatal(err)
	}
	// Otra Cola sobre el mismo fichero: es lo que pasa al reiniciar.
	pend, err := NuevaCola(ruta).Pendientes()
	if err != nil {
		t.Fatal(err)
	}
	if len(pend) != 1 {
		t.Fatalf("los anclajes pendientes no pueden evaporarse al reiniciar, quedan %d", len(pend))
	}
}

func TestReintentarSellaLoPendienteCuandoLaTSAVuelve(t *testing.T) {
	p := buena(t)
	s := servidor(t, p, true)
	q := colaEn(t)
	h := hashDe("checkpoint que espero su turno")
	if err := q.Encolar(h, instanteSello); err != nil {
		t.Fatal(err)
	}
	c := &Cadena{
		Autoridades: []Autoridad{{Nombre: "ya vuelve", URL: s.URL}},
		Anclas:      p.pool,
		Cola:        q,
		Ahora:       func() time.Time { return instanteSello },
	}
	resueltos, err := c.Reintentar(instanteSello)
	if err != nil {
		t.Fatal(err)
	}
	if len(resueltos) != 1 {
		t.Fatalf("la TSA responde: tenia que resolver el pendiente, resolvio %d", len(resueltos))
	}
	if err := c.VerificarOffline(resueltos[0].Hash, resueltos[0].Token); err != nil {
		t.Fatalf("el token del reintento tiene que verificar: %v", err)
	}
	pend, _ := q.Pendientes()
	if len(pend) != 0 {
		t.Fatalf("resuelto es resuelto, la cola tiene que quedar vacia y tiene %d", len(pend))
	}
}

func TestReintentarConLaTSAAunCaidaAnotaElIntentoYNoFalla(t *testing.T) {
	p := buena(t)
	q := colaEn(t)
	h := hashDe("sigue sin haber TSA")
	if err := q.Encolar(h, instanteSello); err != nil {
		t.Fatal(err)
	}
	c := &Cadena{
		Autoridades: []Autoridad{{Nombre: "sigue caida", URL: caida(t, http.StatusInternalServerError).URL}},
		Anclas:      p.pool,
		Cola:        q,
		Ahora:       func() time.Time { return instanteSello },
	}
	resueltos, err := c.Reintentar(instanteSello)
	if err != nil {
		t.Fatalf("que la TSA siga caida es el caso normal de Reintentar, no un error: %v", err)
	}
	if len(resueltos) != 0 {
		t.Fatal("no habia TSA: no se pudo resolver nada")
	}
	pend, _ := q.Pendientes()
	if len(pend) != 1 || pend[0].Intentos != 1 {
		t.Fatalf("el intento tiene que quedar anotado para el backoff: %+v", pend)
	}
}

func TestElBackoffEsperaAntesDeVolverAIntentar(t *testing.T) {
	p := Pendiente{Hash: "aa", Encolado: instanteSello, Intentos: 1, Ultimo: instanteSello}
	if p.toca(instanteSello.Add(esperaBase)) {
		t.Fatal("con un intento hecho hay que esperar mas que la espera base")
	}
	if !p.toca(instanteSello.Add(esperaTope)) {
		t.Fatal("pasado el tope siempre toca reintentar")
	}
	nuevo := Pendiente{Hash: "bb", Encolado: instanteSello}
	if !nuevo.toca(instanteSello) {
		t.Fatal("un pendiente sin intentos se intenta ya")
	}
}

func TestAtascadosAvisaDeLoQueYaNoEsUnCorte(t *testing.T) {
	q := colaEn(t)
	h := hashDe("lleva semanas sin sellar")
	if err := q.Encolar(h, instanteSello); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < intentosParaAvisar; i++ {
		if err := q.anotarIntento(hexDe(h), instanteSello); err != nil {
			t.Fatal(err)
		}
	}
	at, err := q.Atascados()
	if err != nil {
		t.Fatal(err)
	}
	if len(at) != 1 {
		t.Fatalf("tras %d intentos deja de ser un corte y hay que avisar, avisados %d",
			intentosParaAvisar, len(at))
	}
}

// --- Controles negativos de la verificacion ---

// El agujero de timestamp.Parse: solo verifica la firma SI el token trae
// certificados. Un token sin certificado le pasa sin comprobar nada, asi que
// si esta comprobacion se cae, cualquiera puede fabricar sellos.
func TestUnTokenSinCertificadoNoVerifica(t *testing.T) {
	p := buena(t)
	s := servidor(t, p, false) // sin certificado en la respuesta
	c := &Cadena{
		Autoridades: []Autoridad{{Nombre: "sin cert", URL: s.URL}},
		Anclas:      p.pool,
		Cola:        colaEn(t),
		Ahora:       func() time.Time { return instanteSello },
	}
	h := hashDe("checkpoint")
	if _, err := c.Sellar(h); !errors.Is(err, ErrEncolado) {
		t.Fatalf("un token sin certificado no se puede verificar, la TSA cuenta como caida: %v", err)
	}
}

func TestUnTokenDeOtroHashNoVerifica(t *testing.T) {
	p := buena(t)
	s := servidor(t, p, true)
	c := &Cadena{
		Autoridades: []Autoridad{{Nombre: "falsa", URL: s.URL}},
		Anclas:      p.pool,
	}
	token, err := c.Sellar(hashDe("lo que se sello de verdad"))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.VerificarOffline(hashDe("otro contenido distinto"), token); err == nil {
		t.Fatal("un sello de otro contenido no puede darse por bueno: seria reutilizar el sello ajeno")
	}
}

func TestUnTokenDeUnaCANoDeclaradaNoVerifica(t *testing.T) {
	buenaPKI := buena(t)
	s := servidor(t, intrusa(t), true)
	// Se sella contra la CA intrusa pero se verifica contra las anclas buenas.
	emisor := &Cadena{Autoridades: []Autoridad{{Nombre: "intrusa", URL: s.URL}}, Anclas: intrusa(t).pool}
	h := hashDe("checkpoint")
	token, err := emisor.Sellar(h)
	if err != nil {
		t.Fatal(err)
	}
	receptor := &Cadena{Anclas: buenaPKI.pool}
	if err := receptor.VerificarOffline(h, token); err == nil {
		t.Fatal("un sello firmado por una CA que el receptor no declara no puede verificar")
	}
}

func TestSinAnclasNoVerificaEnVezDeFingirQueSi(t *testing.T) {
	p := buena(t)
	s := servidor(t, p, true)
	c := &Cadena{Autoridades: []Autoridad{{Nombre: "falsa", URL: s.URL}}, Anclas: p.pool}
	h := hashDe("checkpoint")
	token, err := c.Sellar(h)
	if err != nil {
		t.Fatal(err)
	}
	sinAnclas := &Cadena{}
	if err := sinAnclas.VerificarOffline(h, token); err == nil {
		t.Fatal("sin anclas la verificacion seria circular: el token trae su propio certificado")
	}
}

func TestUnTokenManipuladoNoVerifica(t *testing.T) {
	p := buena(t)
	s := servidor(t, p, true)
	c := &Cadena{Autoridades: []Autoridad{{Nombre: "falsa", URL: s.URL}}, Anclas: p.pool}
	h := hashDe("checkpoint")
	token, err := c.Sellar(h)
	if err != nil {
		t.Fatal(err)
	}
	roto := make([]byte, len(token))
	copy(roto, token)
	roto[len(roto)/2] ^= 0xff
	if err := c.VerificarOffline(h, roto); err == nil {
		t.Fatal("un token con un byte cambiado no puede verificar")
	}
}

// --- La verificacion no toca la red ---

type transporteQueDelata struct{ usado bool }

func (t *transporteQueDelata) RoundTrip(*http.Request) (*http.Response, error) {
	t.usado = true
	return nil, errors.New("aqui no se sale a la red")
}

// Prueba que VerificarOffline no usa el cliente HTTP de la Cadena. No prueba
// que x509 no salga por su cuenta, y no hace falta: el x509 de Go no descarga
// AIA, y aqui no se llama a OCSP ni a CRL en ninguna linea.
func TestVerificarNoUsaLaRed(t *testing.T) {
	p := buena(t)
	s := servidor(t, p, true)
	emisor := &Cadena{Autoridades: []Autoridad{{Nombre: "falsa", URL: s.URL}}, Anclas: p.pool}
	h := hashDe("checkpoint")
	token, err := emisor.Sellar(h)
	if err != nil {
		t.Fatal(err)
	}

	espia := &transporteQueDelata{}
	auditor := &Cadena{
		Anclas: p.pool,
		HTTP:   &http.Client{Transport: espia},
	}
	if err := auditor.VerificarOffline(h, token); err != nil {
		t.Fatalf("el auditor verifica sin red y meses despues: %v", err)
	}
	if espia.usado {
		t.Fatal("la verificacion ha salido a la red: deja de valer en la maquina del auditor")
	}
}

// El certificado de sellado caduca, pero el sello de 2026 sigue valiendo en
// 2031: lo que se comprueba es que el certificado valia CUANDO se sello.
func TestElSelloSigueValiendoCuandoElCertificadoYaCaduco(t *testing.T) {
	p := buena(t)
	s := servidor(t, p, true)
	c := &Cadena{Autoridades: []Autoridad{{Nombre: "falsa", URL: s.URL}}, Anclas: p.pool}
	h := hashDe("checkpoint de 2026")
	token, err := c.Sellar(h)
	if err != nil {
		t.Fatal(err)
	}
	// El certificado de la PKI de pruebas caduca 24 h despues del sello.
	// VerificarOffline no mira el reloj, asi que esto verifica siempre.
	if err := c.VerificarOffline(h, token); err != nil {
		t.Fatalf("un sello valido no puede dejar de serlo porque pase el tiempo: %v", err)
	}
}

func TestRevisarAvisaDeLaConfiguracionCoja(t *testing.T) {
	sola := &Cadena{Autoridades: []Autoridad{{Nombre: "unica", URL: "http://x"}}}
	avisos := sola.Revisar()
	if len(avisos) == 0 {
		t.Fatal("una sola TSA no es cadena de reserva y hay que decirlo")
	}
	p := buena(t)
	completa := &Cadena{
		Autoridades: []Autoridad{{Nombre: "a", URL: "http://a"}, {Nombre: "b", URL: "http://b"}},
		Anclas:      p.pool,
		Cola:        colaEn(t),
	}
	if avisos := completa.Revisar(); len(avisos) != 0 {
		t.Fatalf("una configuracion completa no tiene que avisar de nada: %v", avisos)
	}
}

func hexDe(b []byte) string { return hex.EncodeToString(b) }

// Por que VerificarOffline no se apoya en timestamp.Parse para la firma: la
// libreria solo llama a Verify() SI el token trae certificados, asi que un
// token sin certificado le pasa entero sin que se compruebe ninguna firma.
// Este test lo demuestra contra la libreria, no contra nuestro codigo, y es la
// razon de que la comprobacion de verdad se haga aparte con pkcs7. Si algun
// dia la libreria cambia y empieza a rechazarlo, este test se pondra rojo y lo
// que habra que revisar es el comentario, no el codigo.
func TestLaLibreriaTragaUnTokenSinCertificadoYPorEsoNoLeCreemos(t *testing.T) {
	p := buena(t)
	s := servidor(t, p, false) // responde sin certificado
	c := &Cadena{Anclas: p.pool}
	h := hashDe("checkpoint")

	// pedir NO verifica: trae el token tal cual lo dio la TSA.
	token, err := c.pedir(Autoridad{Nombre: "sin cert", URL: s.URL}, h)
	if err != nil {
		t.Fatalf("la TSA falsa tiene que responder algo: %v", err)
	}
	ts, err := timestamp.Parse(token)
	if err != nil {
		t.Fatalf("si la libreria ya rechaza el token sin certificado, revisa el comentario "+
			"de VerificarOffline: la premisa ha cambiado (%v)", err)
	}
	if ts.AddTSACertificate {
		t.Fatal("la TSA falsa tenia que responder sin certificado para que este test pruebe algo")
	}
	// Ahi esta: Parse ha dicho que si sin comprobar ninguna firma.
	if err := c.VerificarOffline(h, token); err == nil {
		t.Fatal("nuestra verificacion no puede aceptar un token cuya firma nadie ha comprobado")
	}
}

// Regresion del panico que encontro el fuzzing: 0x30 0x84 es una SEQUENCE que
// dice traer cuatro bytes de longitud y se queda sin bytes. pkcs7.readObject
// se salia del slice. La semilla vive ademas en testdata y corre en cada
// go test, pero esto deja dicho en el paquete que el rechazo tiene que ser
// limpio, no solo no reventar.
func TestUnTokenMalformadoSeRechazaEnVezDeReventar(t *testing.T) {
	p := buena(t)
	c := &Cadena{Anclas: p.pool}
	h := hashDe("checkpoint")
	for _, roto := range [][]byte{
		{0x30, 0x84},
		{0x30, 0x80, 0x00, 0x00},
		{0x30, 0x84, 0xff, 0xff, 0xff},
		[]byte("ni de lejos ASN.1"),
	} {
		if err := c.VerificarOffline(h, roto); err == nil {
			t.Fatalf("%x tenia que rechazarse", roto)
		}
		if _, err := Instante(roto); err == nil {
			t.Fatalf("Instante(%x) tenia que fallar", roto)
		}
	}
}

func TestPorDefectoTraeCadenaDeReservaYSusRaices(t *testing.T) {
	c, err := PorDefecto()
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Autoridades) < 2 {
		t.Fatalf("la configuracion de arranque tiene que traer al menos dos TSAs, trae %d",
			len(c.Autoridades))
	}
	if c.Anclas == nil {
		t.Fatal("PorDefecto tiene que traer las raices cargadas: un verificador sin raices no sirve offline")
	}
	// Lo unico que le falta a PorDefecto es la cola, que necesita una ruta en
	// disco y la pone el operador. Revisar tiene que decir eso y nada mas.
	avisos := c.Revisar()
	if len(avisos) != 1 || !strings.Contains(avisos[0], "cola") {
		t.Fatalf("con TSAs y raices puestas, el unico aviso pendiente es la cola: %v", avisos)
	}
}

// Las raices embebidas tienen que ser certificados de verdad, no un fichero
// vacio que haga que todo "verifique" por no comprobar nada.
func TestLasRaicesEmbebidasSonCertificadosReales(t *testing.T) {
	pool, err := RaicesPorDefecto()
	if err != nil {
		t.Fatal(err)
	}
	if pool == nil {
		t.Fatal("sin pool no hay verificacion posible")
	}
	if n := len(pool.Subjects()); n < 2 { //nolint:staticcheck // Subjects basta para contar
		t.Fatalf("tienen que estar las dos raices, hay %d", n)
	}
}

// La cola es lo unico con estado compartido del paquete: el planificador la
// toca mientras el cierre de un checkpoint encola. Este test no sustituye a
// -race (que aqui no corre por falta de compilador de C), pero si la
// exclusion estuviera mal, el fichero acabaria a medias o con entradas
// perdidas y eso si se ve.
func TestLaColaAguantaEscriturasConcurrentes(t *testing.T) {
	q := colaEn(t)
	const n = 24
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			h := sha256.Sum256([]byte{byte(i)})
			if err := q.Encolar(h[:], instanteSello); err != nil {
				t.Errorf("encolar %d: %v", i, err)
			}
			if _, err := q.Pendientes(); err != nil {
				t.Errorf("leer %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()
	pend, err := q.Pendientes()
	if err != nil {
		t.Fatalf("la cola tiene que quedar legible: %v", err)
	}
	if len(pend) != n {
		t.Fatalf("se encolaron %d hashes distintos y quedan %d: se ha perdido alguno", n, len(pend))
	}
}
