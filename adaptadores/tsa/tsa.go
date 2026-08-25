// Package tsa ancla los checkpoints del ledger hacia fuera con sellos de
// tiempo RFC 3161, y los verifica sin red.
//
// Por que existe: la cadena del ledger prueba coherencia interna, no fecha.
// Sin un sello externo, el emisor puede fabricar la cadena entera a posteriori
// y sale impecable, que es justo por lo que ledger.verificarCheckpoint rechaza
// un checkpoint sin anclaje. El anclaje es lo que fija el CUANDO contra un
// tercero.
//
// De ahi las tres reglas de este adaptador:
//
//  1. Cadena de reserva. Las TSAs se prueban en orden y una caida no bloquea
//     nada: si no responde ninguna, el hash se encola y el checkpoint sigue su
//     curso con el anclaje pendiente. Sellar devuelve ErrEncolado, que el
//     llamante trata como "todavia no", no como error.
//  2. Nada que venga de la red se da por bueno. Un token recien traido se
//     verifica offline ANTES de devolverlo; una TSA que responde basura cuenta
//     como TSA caida y se pasa a la siguiente.
//  3. La verificacion no toca la red, nunca: ni OCSP, ni CRL, ni AIA. El
//     expediente tiene que verificar en la maquina del auditor, sin salida a
//     internet y meses despues. VerificarOffline es determinista dados el
//     hash, el token y las anclas.
package tsa

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/digitorus/pkcs7"
	"github.com/digitorus/timestamp"

	"dutiq/puertos"
)

// La firma del puerto se comprueba en tiempo de compilacion.
var _ puertos.Anclaje = (*Cadena)(nil)

const (
	tipoConsulta = "application/timestamp-query"
	// Un token RFC 3161 son unos pocos KB. El limite evita que una TSA
	// hostil o rota nos haga leer sin fin.
	maxRespuesta = 1 << 20
	esperaPorTSA = 10 * time.Second
)

// ErrEncolado lo devuelve Sellar cuando ninguna TSA respondio y el hash quedo
// en la cola local. NO es un fallo del checkpoint: es un anclaje pendiente.
// Comprobarlo con errors.Is, nunca comparando el texto.
var ErrEncolado = errors.New("anclaje encolado, ninguna TSA respondio")

// Autoridad es una TSA configurada.
type Autoridad struct {
	Nombre string
	URL    string
}

// Cadena implementa puertos.Anclaje con reserva y cola.
type Cadena struct {
	// Autoridades se prueban EN ORDEN. Con una sola no hay cadena de reserva,
	// y el aviso lo da Revisar().
	Autoridades []Autoridad

	// Anclas son las raices que el verificador acepta. Sin ellas la
	// verificacion seria circular: el token trae su propio certificado y
	// darlo por bueno porque se firma a si mismo no prueba nada.
	Anclas *x509.CertPool

	// Cola guarda los hashes que no se pudieron sellar. Si es nil, una TSA
	// caida se pierde en vez de reintentarse.
	Cola *Cola

	HTTP  *http.Client
	Ahora func() time.Time
}

func (c *Cadena) cliente() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	// Con timeout siempre: una TSA que acepta la conexion y no contesta
	// bloquearia el cierre del checkpoint, que es justo lo que no puede pasar.
	return &http.Client{Timeout: esperaPorTSA}
}

func (c *Cadena) ahora() time.Time {
	if c.Ahora != nil {
		return c.Ahora().UTC()
	}
	return time.Now().UTC()
}

// Revisar comprueba la configuracion y explica que se pierde con cada hueco.
// No falla: informa. La decision de arrancar asi es del operador.
func (c *Cadena) Revisar() []string {
	var avisos []string
	switch len(c.Autoridades) {
	case 0:
		avisos = append(avisos, "no hay ninguna TSA configurada: los checkpoints saldran sin anclaje "+
			"y un checkpoint sin anclaje no verifica")
	case 1:
		avisos = append(avisos, "solo hay una TSA configurada: no hay cadena de reserva, "+
			"y el dia que se caiga todo anclaje se queda en la cola")
	}
	if c.Anclas == nil {
		avisos = append(avisos, "no hay anclas de confianza cargadas: no se podra verificar ningun sello")
	}
	if c.Cola == nil {
		avisos = append(avisos, "no hay cola local: si fallan todas las TSAs el anclaje se pierde "+
			"en vez de reintentarse")
	}
	return avisos
}

// Sellar pide un sello a las TSAs en orden y devuelve el primer token que
// ademas verifica. Si no responde ninguna, encola el hash y devuelve
// ErrEncolado.
func (c *Cadena) Sellar(hash []byte) ([]byte, error) {
	if len(hash) != sha256.Size {
		return nil, fmt.Errorf("el hash a sellar son %d bytes y llegaron %d; "+
			"se sella el sha256 de la raiz Merkle del checkpoint", sha256.Size, len(hash))
	}
	if len(c.Autoridades) == 0 {
		return nil, errors.New("no hay TSAs configuradas; " +
			"anade al menos dos a Cadena.Autoridades o los checkpoints saldran sin anclaje")
	}

	var fallos []string
	for _, a := range c.Autoridades {
		token, err := c.pedir(a, hash)
		if err != nil {
			fallos = append(fallos, a.Nombre+": "+err.Error())
			continue
		}
		// Regla 2: lo que viene de la red se verifica antes de devolverlo.
		// Una TSA que firma con una clave que no encadena a nuestras anclas
		// es tan inservible como una caida, y se trata igual.
		if err := c.VerificarOffline(hash, token); err != nil {
			fallos = append(fallos, a.Nombre+" respondio un token que no verifica: "+err.Error())
			continue
		}
		return token, nil
	}

	if c.Cola != nil {
		if err := c.Cola.Encolar(hash, c.ahora()); err != nil {
			return nil, fmt.Errorf("no sello ninguna TSA (%s) y ademas fallo la cola local: %w; "+
				"el checkpoint se queda sin anclaje y sin reintento",
				strings.Join(fallos, "; "), err)
		}
	}
	return nil, fmt.Errorf("%w; probadas %d: %s", ErrEncolado, len(c.Autoridades), strings.Join(fallos, "; "))
}

// pedir hace la consulta RFC 3161 a una TSA y devuelve el token en crudo.
func (c *Cadena) pedir(a Autoridad, hash []byte) ([]byte, error) {
	// El nonce ata la respuesta a ESTA peticion: sin el, una respuesta
	// guardada para el mismo hash se puede reproducir.
	nonce, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 64))
	if err != nil {
		return nil, fmt.Errorf("no puedo generar el nonce: %w", err)
	}
	pet := timestamp.Request{
		HashAlgorithm: crypto.SHA256,
		HashedMessage: hash,
		// Sin certificado en la respuesta el token no se puede verificar
		// contra nuestras anclas, asi que no serviria para nada.
		Certificates: true,
		Nonce:        nonce,
	}
	cuerpo, err := pet.Marshal()
	if err != nil {
		return nil, fmt.Errorf("no puedo construir la consulta: %w", err)
	}

	resp, err := c.cliente().Post(a.URL, tipoConsulta, bytes.NewReader(cuerpo))
	if err != nil {
		return nil, fmt.Errorf("no responde (%s): %w", a.URL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d desde %s", resp.StatusCode, a.URL)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxRespuesta))
	if err != nil {
		return nil, fmt.Errorf("no puedo leer la respuesta: %w", err)
	}
	ts, err := timestamp.ParseResponse(b)
	if err != nil {
		return nil, fmt.Errorf("respuesta RFC 3161 ilegible: %w", err)
	}
	if ts.Nonce == nil || ts.Nonce.Cmp(nonce) != 0 {
		return nil, errors.New("el nonce no coincide: la respuesta no es a esta peticion")
	}
	return ts.RawToken, nil
}

// VerificarOffline comprueba un token sin tocar la red. Determinista: dados el
// mismo hash, token y anclas, dos maquinas cualesquiera dan el mismo veredicto,
// hoy y dentro de cinco anos.
//
// La frescura (que el sello no sea del futuro, que ordene contra otro sello) NO
// se comprueba aqui a proposito: exigiria un reloj y dejaria de ser
// determinista. Eso es cosa del llamante, que si tiene instante.
func (c *Cadena) VerificarOffline(hash []byte, token []byte) (err error) {
	// El parseo de ASN.1 lo hacen pkcs7 y timestamp, codigo de terceros, sobre
	// bytes que trae el expediente: alguien de quien no nos fiamos.
	//
	// El fuzzing encontro que un token de dos bytes (0x30 0x84: una SEQUENCE
	// que declara cuatro bytes de longitud y no los trae) reventaba
	// pkcs7.readObject con un index out of range. Estaba en la version que
	// teniamos fijada, de 2023; aguas arriba lo arreglaron el 2025-07-29 y la
	// dependencia ya esta subida a esa version, asi que ese caso concreto no
	// depende de este recover.
	//
	// El recover se queda igualmente, y no como parche: un parser de ASN.1
	// ajeno colocado justo en la frontera de confianza no puede tener la
	// capacidad de tumbar al verificador, lo arreglen rapido o no. Las
	// semillas viven en testdata y corren en cada go test.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("el token esta malformado y ha reventado el parser de ASN.1 (%v); "+
				"se rechaza el sello en vez de propagar el panico", r)
		}
	}()
	return c.verificar(hash, token)
}

func (c *Cadena) verificar(hash []byte, token []byte) error {
	if len(hash) != sha256.Size {
		return fmt.Errorf("el hash son %d bytes y llegaron %d", sha256.Size, len(hash))
	}
	if len(token) == 0 {
		return errors.New("no hay token que verificar; " +
			"un checkpoint con anclaje declarado y sin token es un anclaje que nadie puede comprobar")
	}
	if c.Anclas == nil {
		return errors.New("no hay anclas de confianza cargadas; " +
			"el token trae su propio certificado y aceptarlo porque se firma a si mismo seria circular. " +
			"Carga las raices de las TSAs en Cadena.Anclas")
	}

	ts, err := timestamp.Parse(token)
	if err != nil {
		return fmt.Errorf("token RFC 3161 ilegible: %w", err)
	}
	if ts.HashAlgorithm != crypto.SHA256 {
		return fmt.Errorf("el sello usa %v y aqui se sella con SHA-256", ts.HashAlgorithm)
	}
	if !bytes.Equal(ts.HashedMessage, hash) {
		return fmt.Errorf("el sello es de otro contenido: sella %x y se esperaba %x",
			ts.HashedMessage, hash)
	}

	// La firma y la cadena se comprueban aqui, no arriba. timestamp.Parse solo
	// llama a Verify() si el token trae certificados: uno sin certificado le
	// pasa sin que se compruebe ninguna firma. pkcs7 exige firmante y encadena
	// al certificado correcto por emisor y numero de serie, no por posicion.
	p7, err := pkcs7.Parse(token)
	if err != nil {
		return fmt.Errorf("el token no es un CMS legible: %w", err)
	}
	// Esto no es lo que sostiene la seguridad: VerifyWithOpts ya rechaza un
	// token sin certificado por su cuenta, porque no encuentra el certificado
	// del firmante. Se comprueba antes solo para dar el error util, que dice
	// que hay que pedir el sello con Certificates=true.
	if len(p7.Certificates) == 0 {
		return errors.New("el token no trae certificado, asi que no hay con que comprobar la firma; " +
			"pide el sello con Certificates=true")
	}
	intermedios := x509.NewCertPool()
	for _, cert := range p7.Certificates {
		intermedios.AddCert(cert)
	}
	// CurrentTime es el instante del sello: lo que hay que comprobar es que el
	// certificado era valido CUANDO se sello, no hoy. Un sello de 2026 sigue
	// valiendo en 2031 aunque el certificado haya caducado por el camino.
	if err := p7.VerifyWithOpts(x509.VerifyOptions{
		Roots:         c.Anclas,
		Intermediates: intermedios,
		CurrentTime:   ts.Time,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageTimeStamping},
	}); err != nil {
		return fmt.Errorf("la firma del sello no verifica contra las anclas de confianza: %w", err)
	}
	return nil
}

// Instante devuelve la fecha que declara un token ya verificado. Verificar
// primero: sobre un token sin verificar esta fecha no vale nada.
func Instante(token []byte) (inst time.Time, err error) {
	// Mismo aislamiento que en VerificarOffline: parsea el mismo ASN.1 de
	// terceros sobre los mismos bytes de origen no fiable.
	defer func() {
		if r := recover(); r != nil {
			inst, err = time.Time{}, fmt.Errorf(
				"el token esta malformado y ha reventado el parser de ASN.1 (%v)", r)
		}
	}()
	ts, err := timestamp.Parse(token)
	if err != nil {
		return time.Time{}, fmt.Errorf("token RFC 3161 ilegible: %w", err)
	}
	return ts.Time.UTC(), nil
}
