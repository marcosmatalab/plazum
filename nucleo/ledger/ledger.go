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
	"fmt"
	"time"
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
	Hasta      uint64    `json:"hasta"`
	Instante   time.Time `json:"instante"`
	RaizMerkle string    `json:"raiz_merkle"`
	Firma      string    `json:"firma"`
	ClavePub   string    `json:"clave_pub"`
	Anclaje    string    `json:"anclaje"` // TSA RFC 3161, testigo publico, sello eIDAS
}

type Ledger struct {
	Entradas    []Entrada    `json:"entradas"`
	Checkpoints []Checkpoint `json:"checkpoints"`
	// ClavesConfiables son las claves publicas que el receptor acepta, en hex.
	//
	// HALLAZGO DE REVISION: antes la clave publica venia DENTRO del checkpoint,
	// asi que rehacer la historia y firmarla con una clave nueva verificaba sin
	// error. Una firma solo vale contra una clave que el receptor ya conocia.
	ClavesConfiables []string `json:"claves_confiables,omitempty"`
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
func (l *Ledger) Verificar() error {
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
			return fmt.Errorf("entrada %d: contenido alterado", e.Seq)
		}
		prev = e.HashCadena
	}
	for _, c := range l.Checkpoints {
		if err := l.verificarCheckpoint(c); err != nil {
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
func (l *Ledger) Cerrar(priv ed25519.PrivateKey, instante time.Time, anclaje string) Checkpoint {
	var hs []string
	for _, e := range l.Entradas {
		hs = append(hs, e.HashCadena)
	}
	c := Checkpoint{
		Hasta: uint64(len(l.Entradas)), Instante: instante,
		RaizMerkle: raizMerkle(hs), Anclaje: anclaje,
		ClavePub: hex.EncodeToString(priv.Public().(ed25519.PublicKey)),
	}
	c.Firma = hex.EncodeToString(ed25519.Sign(priv, []byte(c.mensaje())))
	l.Checkpoints = append(l.Checkpoints, c)
	return c
}

func (c Checkpoint) mensaje() string {
	return fmt.Sprintf("dutiq-checkpoint-v1|%d|%s|%s", c.Hasta, c.Instante.UTC().Format(time.RFC3339), c.RaizMerkle)
}

func (l *Ledger) verificarCheckpoint(c Checkpoint) error {
	if c.Hasta > uint64(len(l.Entradas)) {
		return fmt.Errorf("checkpoint hasta %d pero solo hay %d entradas", c.Hasta, len(l.Entradas))
	}
	var hs []string
	for _, e := range l.Entradas[:c.Hasta] {
		hs = append(hs, e.HashCadena)
	}
	if r := raizMerkle(hs); r != c.RaizMerkle {
		return fmt.Errorf("checkpoint %d: la raiz no cuadra con las entradas", c.Hasta)
	}
	if len(l.ClavesConfiables) == 0 {
		return fmt.Errorf("checkpoint %d: el expediente no declara claves confiables; "+
			"una firma con la clave que aporta el propio firmante no prueba nada", c.Hasta)
	}
	confiable := false
	for _, k := range l.ClavesConfiables {
		if k == c.ClavePub {
			confiable = true
		}
	}
	if !confiable {
		return fmt.Errorf("checkpoint %d: firmado con una clave que no esta entre las confiables", c.Hasta)
	}
	pub, err := hex.DecodeString(c.ClavePub)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("checkpoint %d: clave publica invalida", c.Hasta)
	}
	firma, err := hex.DecodeString(c.Firma)
	if err != nil || !ed25519.Verify(pub, []byte(c.mensaje()), firma) {
		return fmt.Errorf("checkpoint %d: firma invalida", c.Hasta)
	}
	if c.Anclaje == "" {
		return fmt.Errorf("checkpoint %d: sin anclaje externo, la cadena solo prueba coherencia interna", c.Hasta)
	}
	return nil
}

// PruebaInclusion devuelve la ruta de hashes que permite a un tercero
// comprobar, sin la base entera, que una entrada esta en el checkpoint.
func (l *Ledger) PruebaInclusion(seq uint64, c Checkpoint) ([]string, error) {
	if seq == 0 || seq > c.Hasta {
		return nil, fmt.Errorf("la entrada %d no esta cubierta por el checkpoint", seq)
	}
	if c.Hasta > uint64(len(l.Entradas)) {
		return nil, fmt.Errorf("el checkpoint dice cubrir %d entradas pero el ledger solo tiene %d; "+
			"comprueba el checkpoint antes de pedirle pruebas de inclusion", c.Hasta, len(l.Entradas))
	}
	var nivel []string
	for _, e := range l.Entradas[:c.Hasta] {
		nivel = append(nivel, e.HashCadena)
	}
	for i := range nivel {
		nivel[i] = hashHoja(nivel[i])
	}
	// #nosec G115 -- seq <= c.Hasta <= len(l.Entradas), acotado en las guardas de arriba: cabe en int
	idx := int(seq - 1)
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
