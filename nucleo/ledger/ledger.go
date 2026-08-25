// Package ledger es el registro append-only del expediente: cada afirmacion,
// observacion y decision entra encadenada por hash y se cierra en puntos de
// control firmados.
//
// Lo que una cadena de hashes SI da: detectar que alguien edito o borro una
// entrada intermedia sin rehacer todo lo posterior.
//
// Lo que NO da, y conviene decirlo antes de que lo diga un auditor: si el
// operador controla el binario y la base, puede recalcular la cadena entera.
// La propiedad probatoria la aporta el anclaje EXTERNO del punto de control:
// sellado RFC 3161 contra una TSA, publicacion del checkpoint en un sitio que
// el operador no controla, o sello cualificado eIDAS de un prestador espanol
// para el cierre anual. La cadena hace el anclaje barato, porque basta anclar
// una raiz al dia en vez de cada evidencia.
package ledger

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Centinelas de la verificacion. Existen para que un test (y un llamador) pueda
// afirmar DE QUE se queja la verificacion sin mirar el texto del mensaje.
//
// Por que hacen falta: varios errores del MISMO camino comparten palabras. En
// verificarCheckpointContra hay dos que dicen "sello" (no haber con que
// comprobarlo y que el sello no verifique) y dos que dicen "firma invalida"
// (la del checkpoint aqui y la de la lapida en v2.go). Un test que afirmaba
// con strings.Contains daba verde aunque el fallo detectado fuera otro.
//
// El texto de los mensajes no cambia: sigue diciendo causa, arreglo y cita,
// que es lo accionable. El centinela solo anade identidad.
var (
	// ErrEntradaAlterada: el contenido de una entrada no cuadra con su hash.
	// La comparten la cadena v1 y la v2: son cadenas distintas, asi que un
	// test sobre una nunca puede recibir el error de la otra.
	ErrEntradaAlterada = errors.New("contenido alterado")
	// ErrSinClavesConfiables: el receptor no aporto ninguna clave.
	ErrSinClavesConfiables = errors.New("el RECEPTOR no ha aportado ninguna clave confiable")
	// ErrClaveNoReconocida: firmado con una clave que el receptor no conoce.
	ErrClaveNoReconocida = errors.New("firmado con una clave que el receptor no reconoce")
	// ErrFirmaCheckpoint: la firma del checkpoint no verifica. Distinto de
	// ErrFirmaLapida aunque el texto coincida: lo que se comprueba es otra cosa.
	ErrFirmaCheckpoint = errors.New("firma invalida")
	// ErrSinVerificadorDeSello: no se inyecto Confianza.VerificarSello, asi que
	// el anclaje no se comprueba. NO es lo mismo que ErrSelloNoVerifica.
	ErrSinVerificadorDeSello = errors.New("no hay con que comprobar el sello de tiempo")
	// ErrSelloNoVerifica: habia con que comprobarlo y el sello no paso.
	ErrSelloNoVerifica = errors.New("el sello de tiempo no verifica")
)

type Entrada struct {
	Seq         uint64          `json:"seq"`
	Instante    time.Time       `json:"instante"`
	Tipo        string          `json:"tipo"` // observacion | afirmacion | decision | excepcion
	Sujeto      string          `json:"sujeto"`
	Paquete     string          `json:"paquete"`
	PaqueteHash string          `json:"paquete_hash"` // que version de la norma regia
	Carga       json.RawMessage `json:"carga"`
	Actor       string          `json:"actor"`
	HashPrevio  string          `json:"hash_previo"`
	HashCadena  string          `json:"hash_cadena"`
}

type Checkpoint struct {
	Hasta            uint64    `json:"hasta"`
	Instante         time.Time `json:"instante"`
	RaizMerkle       string    `json:"raiz_merkle"`
	Firma            string    `json:"firma"`
	ClavePub         string    `json:"clave_pub"`
	AnclajeDeclarado string    `json:"anclaje_declarado"` // etiqueta legible: que TSA sello esto
	// Token es el sello RFC 3161 en crudo sobre la raiz Merkle. Sin el, el
	// campo Anclaje era texto libre que no comprobaba nadie.
	//
	// HALLAZGO DE REVISION HOSTIL: verificarCheckpoint solo miraba que Anclaje
	// no estuviera vacio, asi que "me lo acabo de inventar" valia lo mismo que
	// un sello real, y el mensaje de error prometia lo contrario.
	Token []byte `json:"token,omitempty"`
}

// Confianza es lo que aporta el RECEPTOR, nunca el fichero.
//
// HALLAZGO DE REVISION HOSTIL, el bloqueante de la etapa 1: las claves
// confiables vivian como campo con etiqueta json dentro del ledger, o sea que
// el emisor se escribia las claves con las que se comprueba su propia firma, y
// la guarda de "hay al menos una" no servia de nada. Todo lo que el receptor
// debe aportar entra por aqui, por parametro, y por ningun otro sitio.
type Confianza struct {
	// ClavesConfiables son las claves publicas que el receptor ya conocia, en
	// hex, obtenidas de su registro y no del expediente.
	ClavesConfiables []string
	// VerificarSello comprueba el token RFC 3161 contra la raiz Merkle. Se
	// inyecta porque nucleo/ no importa nada externo y el verificador de
	// sellos vive en adaptadores/tsa. Si es nil, el checkpoint no verifica:
	// un anclaje que nadie comprueba no es un anclaje.
	VerificarSello func(hash, token []byte) error
	// ClaveOperador verifica las lapidas de la cadena v2.
	ClaveOperador ed25519.PublicKey
}

type Ledger struct {
	Entradas    []Entrada    `json:"entradas"`
	Checkpoints []Checkpoint `json:"checkpoints"`
	// ClavesDeclaradas es lo que el emisor DICE haber usado. No decide nada:
	// la verificacion usa Confianza.ClavesConfiables, que viene del receptor.
	// Se conserva porque una discrepancia entre lo declarado y lo real es
	// informacion util para el auditor, no un fallo que haya que esconder.
	ClavesDeclaradas []string `json:"claves_declaradas,omitempty"`
}

// hashEntrada usa serializacion canonica y NO descarta el error.
//
// HALLAZGO DE REVISION. La primera version hacia `b, _ := json.Marshal(e)`. Con
// una Carga que no fuera JSON valido, Marshal fallaba, b quedaba nil y TODA
// entrada asi hasheaba a sha256("") = e3b0c442... Es decir, dos entradas
// distintas con el mismo hash: falsificacion directa del registro append-only.
// Y ademas la Carga se reserializaba tal cual, asi que reindentar el fichero
// rompia la cadena: un proxy que normalice JSON era indistinguible de un ataque.
func hashEntrada(e Entrada) (string, error) {
	e.HashCadena = ""
	can, err := canonicalizar(e.Carga)
	if err != nil {
		return "", fmt.Errorf("carga no canonicalizable: %w", err)
	}
	e.Carga = can
	b, err := json.Marshal(e)
	if err != nil {
		return "", fmt.Errorf("entrada no serializable: %w", err)
	}
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:]), nil
}

// canonicalizar reserializa el JSON con claves ordenadas y sin espacios, de modo
// que reindentar el fichero no cambie el hash. Es JSON canonico en el espiritu
// de la RFC 8785, suficiente para los tipos que usa el ledger.
func canonicalizar(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return json.RawMessage("null"), nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	b, err := json.Marshal(v) // json.Marshal ordena las claves de los mapas
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

// Anadir encadena una entrada. Nunca se actualiza ni se borra nada.
func (l *Ledger) Anadir(e Entrada) (Entrada, error) {
	e.Seq = uint64(len(l.Entradas)) + 1
	e.HashPrevio = ""
	if n := len(l.Entradas); n > 0 {
		e.HashPrevio = l.Entradas[n-1].HashCadena
	}
	if c, err := canonicalizar(e.Carga); err == nil {
		e.Carga = c
	} else {
		return Entrada{}, fmt.Errorf("entrada %d: %w", e.Seq, err)
	}
	h, err := hashEntrada(e)
	if err != nil {
		return Entrada{}, fmt.Errorf("entrada %d: %w", e.Seq, err)
	}
	e.HashCadena = h
	l.Entradas = append(l.Entradas, e)
	return e, nil
}

// Verificar recorre la cadena entera y devuelve la primera rotura.
//
// Recibe la Confianza por parametro y NO lee ningun campo de confianza del
// propio fichero: ese era el bloqueante de la revision hostil.
func (l *Ledger) Verificar(cf Confianza) error {
	prev := ""
	for i, e := range l.Entradas {
		if e.Seq != uint64(i+1) {
			return fmt.Errorf("entrada %d: secuencia rota (%d)", i+1, e.Seq)
		}
		if e.HashPrevio != prev {
			return fmt.Errorf("entrada %d: no encadena con la anterior", e.Seq)
		}
		h, err := hashEntrada(e)
		if err != nil {
			return fmt.Errorf("entrada %d: %w", e.Seq, err)
		}
		if h != e.HashCadena {
			return fmt.Errorf("entrada %d: %w", e.Seq, ErrEntradaAlterada)
		}
		prev = e.HashCadena
	}
	for _, c := range l.Checkpoints {
		if err := l.verificarCheckpoint(c, cf); err != nil {
			return err
		}
	}
	return nil
}

// Separacion de dominio hoja e interno, como en la RFC 6962. Sin ella, un
// atacante puede presentar un nodo interno como si fuera una hoja.
func hashHoja(h string) string {
	s := sha256.Sum256(append([]byte{0x00}, []byte(h)...))
	return hex.EncodeToString(s[:])
}

func hashInterno(a, b string) string {
	s := sha256.Sum256(append([]byte{0x01}, []byte(a+b)...))
	return hex.EncodeToString(s[:])
}

func raizMerkle(hashes []string) string {
	if len(hashes) == 0 {
		s := sha256.Sum256(nil)
		return hex.EncodeToString(s[:])
	}
	nivel := make([]string, 0, len(hashes))
	for _, h := range hashes {
		nivel = append(nivel, hashHoja(h))
	}
	for len(nivel) > 1 {
		var sig []string
		for i := 0; i < len(nivel); i += 2 {
			if i+1 == len(nivel) {
				sig = append(sig, nivel[i])
				continue
			}
			sig = append(sig, hashInterno(nivel[i], nivel[i+1]))
		}
		nivel = sig
	}
	return nivel[0]
}

// Cerrar emite un punto de control firmado sobre todo lo acumulado.
//
// token es el sello RFC 3161 sobre la raiz Merkle, tal y como lo devuelve el
// adaptador de anclaje. Se admite vacio porque una TSA caida no puede bloquear
// el cierre (el adaptador lo encola y lo reintenta), pero un checkpoint sin
// token NO verifica: queda como pendiente de anclar, no como anclado.
func (l *Ledger) Cerrar(priv ed25519.PrivateKey, instante time.Time, anclaje string, token []byte) Checkpoint {
	hs := make([]string, 0, len(l.Entradas))
	for _, e := range l.Entradas {
		hs = append(hs, e.HashCadena)
	}
	c := construirCheckpoint(hs, priv, instante, anclaje, token)
	l.Checkpoints = append(l.Checkpoints, c)
	return c
}

// construirCheckpoint es la parte comun a la cadena v1 y la v2.
func construirCheckpoint(hashes []string, priv ed25519.PrivateKey, instante time.Time,
	anclaje string, token []byte) Checkpoint {
	c := Checkpoint{
		Hasta: uint64(len(hashes)), Instante: instante,
		RaizMerkle: raizMerkle(hashes), AnclajeDeclarado: anclaje, Token: token,
		ClavePub: hex.EncodeToString(priv.Public().(ed25519.PublicKey)),
	}
	c.Firma = hex.EncodeToString(ed25519.Sign(priv, []byte(c.mensaje())))
	return c
}

// mensaje incluye ahora el anclaje y el digest del token. Antes no: la firma
// cubria Hasta, Instante y RaizMerkle, asi que el sello se podia sustituir por
// otro sin invalidar nada. Dominio subido a v2 porque el mensaje firmado cambia.
func (c Checkpoint) mensaje() string {
	td := sha256.Sum256(c.Token)
	return fmt.Sprintf("dutiq-checkpoint-v2|%d|%s|%s|%s|%s",
		c.Hasta, c.Instante.UTC().Format(time.RFC3339), c.RaizMerkle,
		c.AnclajeDeclarado, hex.EncodeToString(td[:]))
}

// verificarCheckpointContra comprueba un checkpoint contra los hashes de las
// entradas y la confianza que aporta EL RECEPTOR. La comparten la cadena v1 y
// la v2: el checkpoint es el mismo objeto en las dos.
func verificarCheckpointContra(c Checkpoint, hashes []string, cf Confianza) error {
	if c.Hasta > uint64(len(hashes)) {
		return fmt.Errorf("checkpoint hasta %d pero solo hay %d entradas", c.Hasta, len(hashes))
	}
	if r := raizMerkle(hashes[:c.Hasta]); r != c.RaizMerkle {
		return fmt.Errorf("checkpoint %d: la raiz no cuadra con las entradas", c.Hasta)
	}
	if len(cf.ClavesConfiables) == 0 {
		return fmt.Errorf("checkpoint %d: %w; "+
			"sin eso la firma se comprobaria contra la clave que trae el propio fichero, "+
			"que no prueba nada. Cargalas de tu registro, no del expediente", c.Hasta, ErrSinClavesConfiables)
	}
	confiable := false
	for _, k := range cf.ClavesConfiables {
		if k == c.ClavePub {
			confiable = true
		}
	}
	if !confiable {
		return fmt.Errorf("checkpoint %d: %w", c.Hasta, ErrClaveNoReconocida)
	}
	pub, err := hex.DecodeString(c.ClavePub)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("checkpoint %d: clave publica invalida", c.Hasta)
	}
	firma, err := hex.DecodeString(c.Firma)
	if err != nil || !ed25519.Verify(pub, []byte(c.mensaje()), firma) {
		return fmt.Errorf("checkpoint %d: %w", c.Hasta, ErrFirmaCheckpoint)
	}
	// El anclaje, de verdad. Antes esto era c.AnclajeDeclarado != "".
	if cf.VerificarSello == nil {
		return fmt.Errorf("checkpoint %d: %w; "+
			"un anclaje que nadie verifica no es un anclaje. Inyecta Confianza.VerificarSello",
			c.Hasta, ErrSinVerificadorDeSello)
	}
	raiz, err := hex.DecodeString(c.RaizMerkle)
	if err != nil {
		return fmt.Errorf("checkpoint %d: la raiz Merkle no es hexadecimal", c.Hasta)
	}
	if err := cf.VerificarSello(raiz, c.Token); err != nil {
		return fmt.Errorf("checkpoint %d: %w: %w", c.Hasta, ErrSelloNoVerifica, err)
	}
	return nil
}

func (l *Ledger) verificarCheckpoint(c Checkpoint, cf Confianza) error {
	hs := make([]string, 0, len(l.Entradas))
	for _, e := range l.Entradas {
		hs = append(hs, e.HashCadena)
	}
	return verificarCheckpointContra(c, hs, cf)
}

// PruebaInclusion devuelve la ruta de hashes que permite a un tercero
// comprobar, sin la base entera, que una entrada esta en el checkpoint.
func (l *Ledger) PruebaInclusion(seq uint64, c Checkpoint) ([]string, error) {
	if seq == 0 {
		return nil, fmt.Errorf("las entradas de la cadena v1 empiezan en 1, no en 0")
	}
	hs := make([]string, 0, len(l.Entradas))
	for _, e := range l.Entradas {
		hs = append(hs, e.HashCadena)
	}
	return rutaInclusion(hs, seq-1, c)
}

// rutaInclusion construye la ruta Merkle de la hoja en la posicion indice
// (contando desde 0). La comparten la cadena v1 y la v2.
func rutaInclusion(hashes []string, indice uint64, c Checkpoint) ([]string, error) {
	if indice >= c.Hasta {
		return nil, fmt.Errorf("la entrada en la posicion %d no esta cubierta por el checkpoint", indice)
	}
	if c.Hasta > uint64(len(hashes)) {
		return nil, fmt.Errorf("el checkpoint dice cubrir %d entradas pero la cadena solo tiene %d; "+
			"comprueba el checkpoint antes de pedirle pruebas de inclusion", c.Hasta, len(hashes))
	}
	nivel := make([]string, 0, c.Hasta)
	for _, h := range hashes[:c.Hasta] {
		nivel = append(nivel, hashHoja(h))
	}
	// #nosec G115 -- indice < c.Hasta <= len(hashes), acotado arriba: cabe en int
	idx := int(indice)
	var ruta []string
	for len(nivel) > 1 {
		var sig []string
		for i := 0; i < len(nivel); i += 2 {
			if i+1 == len(nivel) {
				sig = append(sig, nivel[i])
				continue
			}
			if i == idx || i+1 == idx {
				if i == idx {
					ruta = append(ruta, "R:"+nivel[i+1])
				} else {
					ruta = append(ruta, "L:"+nivel[i])
				}
			}
			sig = append(sig, hashInterno(nivel[i], nivel[i+1]))
		}
		idx /= 2
		nivel = sig
	}
	return ruta, nil
}

// ComprobarInclusion la verifica sin acceso al ledger. Esto es lo que hace un
// auditor con su copia del expediente y el checkpoint publicado.
func ComprobarInclusion(hoja string, ruta []string, raiz string) bool {
	h := hashHoja(hoja)
	for _, paso := range ruta {
		if len(paso) < 2 {
			return false
		}
		if paso[0] == 'R' {
			h = hashInterno(h, paso[2:])
		} else {
			h = hashInterno(paso[2:], h)
		}
	}
	return h == raiz
}
