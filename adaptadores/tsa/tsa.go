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

	"plazum/adaptadores/tsa/internal/pkcs7"
	"plazum/puertos"
)

// La firma del puerto se comprueba en tiempo de compilacion.
var _ puertos.Anclaje = (*Cadena)(nil)

const (
	tipoConsulta = "application/timestamp-query"
	// Un token RFC 3161 son unos pocos KB. El limite evita que una TSA
	// hostil o rota nos haga leer sin fin.
	maxRespuesta = 1 << 20
	esperaPorTSA = 10 * time.Second

	// maxToken acota el token ANTES de parsearlo. maxRespuesta de arriba solo
	// cubre lo que llega por HTTP de una TSA; el token que se verifica llega
	// dentro del expediente, que lo aporta alguien de quien no nos fiamos y que
	// no pasa por ningun LimitReader.
	//
	// POR QUE HACE FALTA UN TOPE, y no basta con un parser que no revienta. El
	// fuzzing del pkcs7 vendorizado midio que la transcodificacion de BER a DER
	// AMPLIFICA, y mucho. En readObject, un objeto construido de longitud
	// DEFINIDA devuelve la longitud DECLARADA y no la que sus hijos consumieron
	// de verdad; los bytes que un hijo se traga de mas los vuelve a leer el
	// abuelo como si fueran su siguiente hermano, y salen dos veces. Anidado,
	// se multiplica.
	//
	// Medido en esta maquina, con entradas que encontro el fuzzer y que estan
	// commiteadas en internal/pkcs7/testdata/fuzz:
	//
	//	   331 bytes ->    159.693 (x482)
	//	   631 bytes ->  1.197.909 (x1.898)
	//	   931 bytes ->  2.542.305 (x2.731)
	//
	// La razon se aplana hacia x4.000: cada dos bytes de entrada anaden unos
	// ocho mil de salida. No es exponencial, pero un factor de cuatro mil sobre
	// una entrada sin tope si es una denegacion de servicio, y el token no
	// pasaba por ningun tope: llega dentro del expediente, no por HTTP.
	//
	// EL TOPE ES LA UNICA DEFENSA EFECTIVA HOY, y esto importa: verificar()
	// llama primero a timestamp.Parse, que parsea el token con el pkcs7 de
	// AGUAS ARRIBA, no con la copia vendorizada. Una guarda dentro de nuestra
	// copia no llegaria a ejecutarse. Acotar la entrada acota los dos caminos.
	//
	// 32 KiB: siete veces el token del expediente de demostracion (4.636
	// bytes), con sitio de sobra para un QTSP con cadena larga y certificados
	// RSA-4096, y deja el peor caso conocido en unos 130 MB de memoria
	// transitoria, dentro del presupuesto de 256 MB del proyecto. Mas bajo
	// arriesga rechazar un sello legitimo, que en este producto es peor fallo
	// que gastar memoria: acusa al emisor de un problema del receptor.
	maxToken = 32 << 10
)

// ErrTokenDemasiadoGrande lo devuelve VerificarOffline cuando el token supera
// maxToken. Comprobarlo con errors.Is, nunca comparando el texto.
var ErrTokenDemasiadoGrande = errors.New("el sello de tiempo es demasiado grande para ser un token RFC 3161")

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

// Los avisos de Revisar, con nombre. Son cuatro textos fijos y son la unica
// salida de la funcion, asi que se comparan por identidad y no por subcadena:
// avisoUnaSolaTSA y avisoSinCola dicen los dos "cola", y un test que buscara
// esa palabra no distinguiria "falta la cola" de "solo hay una TSA".
const (
	avisoSinTSA = "no hay ninguna TSA configurada: los checkpoints saldran sin anclaje " +
		"y un checkpoint sin anclaje no verifica"
	avisoUnaSolaTSA = "solo hay una TSA configurada: no hay cadena de reserva, " +
		"y el dia que se caiga todo anclaje se queda en la cola"
	avisoSinAnclas = "no hay anclas de confianza cargadas: no se podra verificar ningun sello"
	avisoSinCola   = "no hay cola local: si fallan todas las TSAs el anclaje se pierde " +
		"en vez de reintentarse"
)

// Revisar comprueba la configuracion y explica que se pierde con cada hueco.
// No falla: informa. La decision de arrancar asi es del operador.
func (c *Cadena) Revisar() []string {
	var avisos []string
	switch len(c.Autoridades) {
	case 0:
		avisos = append(avisos, avisoSinTSA)
	case 1:
		avisos = append(avisos, avisoUnaSolaTSA)
	}
	if c.Anclas == nil {
		avisos = append(avisos, avisoSinAnclas)
	}
	if c.Cola == nil {
		avisos = append(avisos, avisoSinCola)
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
	cuerpo, err := construirPeticion(hash, nonce)
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
	token, err := leerRespuesta(b)
	if err != nil {
		return nil, err
	}
	// El nonce se comprueba sobre el TSTInfo DEL TOKEN, que es lo que va
	// firmado, y no sobre un campo suelto de la respuesta. La respuesta no la
	// firma nadie: un intermediario puede poner ahi el nonce que quiera.
	sello, err := leerSello(token)
	if err != nil {
		return nil, err
	}
	if sello.Nonce == nil || sello.Nonce.Cmp(nonce) != 0 {
		return nil, errors.New("el nonce del sello no coincide con el de la peticion: " +
			"la respuesta no es a esta peticion, o es una guardada de antes")
	}
	return token, nil
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
	// teniamos fijada, de 2023; aguas arriba lo arreglaron el 2025-07-29.
	// Desde la etapa 2 ese pkcs7 vive vendorizado en internal/pkcs7 y su
	// parser lo fuzzea nuestra propia suite en cada CI, no la de un tercero
	// (adaptadores/tsa/internal/pkcs7/LEEME.md).
	//
	// timestamp sigue siendo dependencia externa y sigue parseando estos mismos
	// bytes por su cuenta, asi que la frontera de confianza no es solo nuestra.
	//
	// El recover se queda igualmente, y no como parche: un parser de ASN.1
	// colocado justo en la frontera de confianza no puede tener la capacidad de
	// tumbar al verificador, sea de quien sea. Las semillas viven en testdata y
	// corren en cada go test.
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
	// El tope va ANTES de tocar ningun parser: el trabajo que hace el
	// verificador sobre bytes de un tercero tiene que estar acotado por el
	// tamano de esos bytes, y el parser de ASN.1 amplifica (ver maxToken).
	if len(token) > maxToken {
		return fmt.Errorf("%w: son %d bytes y el tope es %d. "+
			"Un sello autentico son unos pocos KB (el del expediente de demostracion, 4636). "+
			"Un token de este tamano no viene de una TSA, viene de alguien que quiere que el "+
			"verificador gaste memoria: el transcodificador de BER a DER puede multiplicar por "+
			"cuatro mil lo que se le da. Se rechaza el sello sin parsearlo",
			ErrTokenDemasiadoGrande, len(token), maxToken)
	}
	if c.Anclas == nil {
		return errors.New("no hay anclas de confianza cargadas; " +
			"el token trae su propio certificado y aceptarlo porque se firma a si mismo seria circular. " +
			"Carga las raices de las TSAs en Cadena.Anclas")
	}

	// UN SOLO PARSEO, y de ahi sale todo. Antes el veredicto (que se sello y
	// cuando) venia de timestamp.Parse y la firma se comprobaba sobre otro
	// pkcs7: dos lecturas de los mismos bytes que no ataba nada.
	p7, err := pkcs7.Parse(token)
	if err != nil {
		return fmt.Errorf("el token no es un CMS legible: %w", err)
	}
	sello, err := selloDelContenido(p7)
	if err != nil {
		return err
	}
	if sello.Hash != crypto.SHA256 {
		return fmt.Errorf("el sello usa %v y aqui se sella con SHA-256", sello.Hash)
	}
	if !bytes.Equal(sello.Sellado, hash) {
		return fmt.Errorf("el sello es de otro contenido: sella %x y se esperaba %x",
			sello.Sellado, hash)
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
		CurrentTime:   sello.Instante,
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
	// El mismo tope que en VerificarOffline, y por el mismo motivo: esta es la
	// otra puerta por la que los bytes del expediente llegan a un parser de
	// ASN.1. Un tope que solo esta en una de las dos no es un tope.
	if len(token) > maxToken {
		return time.Time{}, fmt.Errorf("%w: son %d bytes y el tope es %d",
			ErrTokenDemasiadoGrande, len(token), maxToken)
	}
	sello, err := leerSello(token)
	if err != nil {
		return time.Time{}, err
	}
	return sello.Instante, nil
}
