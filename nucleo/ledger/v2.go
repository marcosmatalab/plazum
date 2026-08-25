package ledger

// Ledger v2: el formato con AEAD comprometido y borrado legal.
//
// Por que existe el compromiso: AES-GCM no es key-committing. Un escritor
// malicioso puede fabricar un cifrado que descifra valido bajo dos claves
// distintas y ensenar un contenido al auditor y otro al juzgado con la misma
// cadena valida. La etiqueta HMAC de la clave cierra esa puerta: descifrar
// exige que la clave verifique el compromiso ADEMAS del tag de GCM.
//
// El ataque se conoce como "invisible salamanders":
//   - Dodis, Grubbs, Ristenpart y Woodage, "Fast Message Franking: From
//     Invisible Salamanders to Encryptment", CRYPTO 2018 (eprint 2019/016),
//     que es de donde sale el termino.
//   - Grubbs, "Hunting Invisible Salamanders: Cryptographic Insecurity with
//     Attacker-Controlled Keys", Black Hat USA 2020, la charla accesible.
//
// Borrar = destruir la clave de la entrada en el keystore y anadir una lapida
// firmada con la base legal. La cadena queda intacta, las raices publicadas no
// cambian, y la verificacion informa de la supresion en vez de fallar.

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

const (
	etiquetaCompromiso = "plazum/commit/v1"
	// etiquetaLapida da dominio propio a la firma de la lapida. Sin dominio
	// separado, una firma hecha para una cosa se puede presentar como si fuera
	// de otra.
	etiquetaLapida = "plazum/lapida/v1|"
)

// Centinelas de la cadena v2. Mismo motivo que los de ledger.go: la
// verificacion de una lapida tiene SIETE motivos de rechazo distintos y varios
// comparten palabras con los de otros caminos ("firma invalida" tambien la dice
// el checkpoint, "suprimida con base legal" tambien la dice Borrar). Sin
// identidad, un test que afirma con una subcadena da verde aunque lo que se
// haya detectado sea otra cosa, que es justo lo que no puede pasar en las
// comprobaciones que sostienen el valor probatorio.
//
// Los mensajes no cambian: el centinela entra con %w donde ya estaba su texto.
var (
	// ErrClaveNoCompromete: la clave no comprometio este cifrado. Es la defensa
	// contra los invisible salamanders, comprobada ANTES de llegar a GCM.
	ErrClaveNoCompromete = errors.New("la clave no compromete este cifrado")
	// ErrEntradaSuprimida: la entrada existe pero se borro con base legal.
	ErrEntradaSuprimida = errors.New("suprimida con base legal")
	// ErrSinBaseLegal: se intento borrar sin citar la base legal.
	ErrSinBaseLegal = errors.New("borrar exige base legal citada (Ley 2/2023 art. 32, RGPD art. 17...)")
	// ErrYaSuprimida: se intento borrar dos veces la misma entrada.
	ErrYaSuprimida = errors.New("ya esta suprimida con base legal")
	// ErrLapidaSinBaseLegal: una lapida del fichero viene sin base legal.
	ErrLapidaSinBaseLegal = errors.New("sin base legal")
	// ErrLapidaFueraDeRango: la lapida suprime una entrada que no existe.
	ErrLapidaFueraDeRango = errors.New("una supresion de algo que no existe no es una supresion")
	// ErrLapidaDuplicada: dos lapidas para la misma entrada inflan el recuento.
	ErrLapidaDuplicada = errors.New("informar dos veces la misma supresion falsea el recuento")
	// ErrLapidaDeOtraEntrada: la lapida esta firmada sobre otra entrada. Es la
	// comprobacion EXPLICITA del transplante, distinta de que falle la firma:
	// un test que no las separa no sabe cual de las dos capas lo paro.
	ErrLapidaDeOtraEntrada = errors.New("firmada sobre otra entrada; viene de otra cadena o de otro momento")
	// ErrFirmaLapida: la firma de la lapida no verifica contra la clave del
	// operador que aporta el receptor. Distinto de ErrFirmaCheckpoint.
	ErrFirmaLapida = errors.New("firma invalida")
)

// EntradaV2 es una entrada cifrada con compromiso de clave. El hash de la
// cadena se calcula sobre la envoltura cifrada, asi que borrar el contenido
// (destruir la clave) no toca la cadena.
type EntradaV2 struct {
	Indice     uint64 `json:"indice"`
	Previo     []byte `json:"previo"`
	Nonce      []byte `json:"nonce"`
	Cifrado    []byte `json:"cifrado"`
	Compromiso []byte `json:"compromiso"`
	Hash       []byte `json:"hash"`
}

// Lapida registra una supresion con su base legal, firmada por el operador.
type Lapida struct {
	EntradaBorrada uint64 `json:"entrada_borrada"`
	// HashEntrada es el hash de la envoltura cifrada que se suprimio. Ata la
	// lapida a esa entrada y solo a esa.
	HashEntrada []byte `json:"hash_entrada"`
	// Cadena identifica la cadena a la que pertenece la supresion: la raiz
	// Merkle de las entradas en el momento de borrar. Impide transplantarla.
	Cadena    string `json:"cadena"`
	BaseLegal string `json:"base_legal"`
	Instante  string `json:"instante"`
	Firma     []byte `json:"firma"`
}

// Keystore guarda la clave de cada entrada SEPARADA de la cadena: se replica
// aparte y con retencion corta, para que destruir una clave sea destruirla
// tambien en los backups dentro del plazo declarado (35 dias por defecto).
type Keystore struct {
	claves map[uint64][]byte
}

func NuevoKeystore() *Keystore { return &Keystore{claves: map[uint64][]byte{}} }

func (k *Keystore) guardar(i uint64, clave []byte) { k.claves[i] = clave }

// Destruir borra la clave: el contenido de esa entrada pasa a ser irrecuperable.
func (k *Keystore) Destruir(i uint64) {
	if c, ok := k.claves[i]; ok {
		for j := range c {
			c[j] = 0
		}
		delete(k.claves, i)
	}
}

func (k *Keystore) clave(i uint64) ([]byte, bool) { c, ok := k.claves[i]; return c, ok }

// CadenaV2 es la cadena de entradas v2 con sus lapidas.
type CadenaV2 struct {
	Entradas []EntradaV2 `json:"entradas"`
	Lapidas  []Lapida    `json:"lapidas"`
	// Checkpoints son los mismos que los de la cadena v1: raiz Merkle sobre los
	// hashes de las entradas, firmada, con su sello RFC 3161.
	//
	// HALLAZGO DE REVISION HOSTIL: la v2 se construyo sin checkpoints, asi que
	// el expediente seguia llevando la v1 (la unica con raiz, firma y anclaje)
	// y todo lo que la v2 aporta (compromiso de clave, lapidas, keystore) se
	// quedaba fuera del camino que recorre un tercero.
	Checkpoints []Checkpoint `json:"checkpoints"`
	// ClavesDeclaradas es lo que el emisor DICE haber usado para firmar. No
	// decide nada: la verificacion usa Confianza.ClavesConfiables, que viene
	// del receptor. Se conserva porque la diferencia entre lo declarado y lo
	// real es informacion util para el auditor.
	ClavesDeclaradas []string `json:"claves_declaradas,omitempty"`
}

// hashesDeEntradas es lo que entra en el arbol Merkle: el hash de la envoltura
// cifrada, que no cambia al destruir la clave. Por eso borrar no rompe la raiz.
func (c *CadenaV2) hashesDeEntradas() []string {
	hs := make([]string, 0, len(c.Entradas))
	for _, e := range c.Entradas {
		hs = append(hs, hex.EncodeToString(e.Hash))
	}
	return hs
}

// Cerrar emite un checkpoint firmado y anclado sobre lo acumulado.
func (c *CadenaV2) Cerrar(priv ed25519.PrivateKey, instante time.Time,
	anclaje string, token []byte) Checkpoint {
	cp := construirCheckpoint(c.hashesDeEntradas(), priv, instante, anclaje, token)
	c.Checkpoints = append(c.Checkpoints, cp)
	return cp
}

// PruebaInclusion da la ruta Merkle de una entrada, para que un tercero
// compruebe que esta en el checkpoint sin tener la cadena entera.
func (c *CadenaV2) PruebaInclusion(indice uint64, cp Checkpoint) ([]string, error) {
	return rutaInclusion(c.hashesDeEntradas(), indice, cp)
}

func compromisoDe(clave, nonce []byte) []byte {
	m := hmac.New(sha256.New, clave)
	m.Write([]byte(etiquetaCompromiso))
	m.Write(nonce)
	return m.Sum(nil)
}

func hashEntradaV2(e EntradaV2) []byte {
	h := sha256.New()
	var idx [8]byte
	binary.BigEndian.PutUint64(idx[:], e.Indice)
	h.Write(idx[:])
	h.Write(e.Previo)
	h.Write(e.Nonce)
	h.Write(e.Cifrado)
	h.Write(e.Compromiso)
	return h.Sum(nil)
}

// SellarComprometido cifra un contenido con AES-256-GCM y devuelve ademas la
// etiqueta de compromiso de la clave. Exportado porque el almacen de blobs usa
// exactamente el mismo regimen.
func SellarComprometido(clave, nonce, contenido []byte) (cifrado, compromiso []byte, err error) {
	if len(clave) != 32 {
		return nil, nil, errors.New("la clave debe ser de 32 bytes")
	}
	if len(nonce) != 12 {
		return nil, nil, errors.New("el nonce debe ser de 12 bytes")
	}
	bloque, err := aes.NewCipher(clave)
	if err != nil {
		return nil, nil, err
	}
	gcm, err := cipher.NewGCM(bloque)
	if err != nil {
		return nil, nil, err
	}
	return gcm.Seal(nil, nonce, contenido, []byte(etiquetaCompromiso)), compromisoDe(clave, nonce), nil
}

// AbrirComprometido verifica el compromiso ANTES de descifrar: una clave que
// no comprometio este cifrado se rechaza aunque GCM la aceptara.
func AbrirComprometido(clave, nonce, cifrado, compromiso []byte) ([]byte, error) {
	if !hmac.Equal(compromisoDe(clave, nonce), compromiso) {
		return nil, fmt.Errorf("%w: clave equivocada o sustituida", ErrClaveNoCompromete)
	}
	bloque, err := aes.NewCipher(clave)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(bloque)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, cifrado, []byte(etiquetaCompromiso))
}

// Anadir cifra el contenido con la clave dada (32 bytes, generada por el
// llamador con crypto/rand fuera del nucleo), la guarda en el keystore y
// encadena la entrada. El nonce (12 bytes) tambien lo aporta el llamador:
// el nucleo no tiene fuentes de aleatoriedad, como no tiene reloj.
func (c *CadenaV2) Anadir(ks *Keystore, clave, nonce, contenido []byte) (EntradaV2, error) {
	cifrado, compromiso, err := SellarComprometido(clave, nonce, contenido)
	if err != nil {
		return EntradaV2{}, err
	}
	var previo []byte
	if n := len(c.Entradas); n > 0 {
		previo = c.Entradas[n-1].Hash
	}
	e := EntradaV2{
		Indice:     uint64(len(c.Entradas)),
		Previo:     previo,
		Nonce:      nonce,
		Cifrado:    cifrado,
		Compromiso: compromiso,
	}
	e.Hash = hashEntradaV2(e)
	ks.guardar(e.Indice, clave)
	c.Entradas = append(c.Entradas, e)
	return e, nil
}

// Leer descifra una entrada. Si su clave fue destruida, dice por que no puede.
func (c *CadenaV2) Leer(ks *Keystore, indice uint64) ([]byte, error) {
	if indice >= uint64(len(c.Entradas)) {
		return nil, fmt.Errorf("no existe la entrada %d", indice)
	}
	if l := c.lapidaDe(indice); l != nil {
		return nil, fmt.Errorf("entrada %d %w %s el %s",
			indice, ErrEntradaSuprimida, l.BaseLegal, l.Instante)
	}
	clave, ok := ks.clave(indice)
	if !ok {
		return nil, fmt.Errorf("entrada %d: clave no disponible (destruida sin lapida?)", indice)
	}
	e := c.Entradas[indice]
	return AbrirComprometido(clave, e.Nonce, e.Cifrado, e.Compromiso)
}

// Borrar destruye la clave y firma la lapida con su base legal. Sin base legal
// no hay borrado: la supresion sin justificar es exactamente lo que un ledger
// probatorio existe para impedir.
func (c *CadenaV2) Borrar(ks *Keystore, priv ed25519.PrivateKey, indice uint64, baseLegal, instante string) (Lapida, error) {
	if baseLegal == "" {
		return Lapida{}, ErrSinBaseLegal
	}
	if indice >= uint64(len(c.Entradas)) {
		return Lapida{}, fmt.Errorf("no existe la entrada %d", indice)
	}
	if l := c.lapidaDe(indice); l != nil {
		return Lapida{}, fmt.Errorf("la entrada %d %w %s el %s",
			indice, ErrYaSuprimida, l.BaseLegal, l.Instante)
	}
	l := Lapida{
		EntradaBorrada: indice,
		HashEntrada:    append([]byte(nil), c.Entradas[indice].Hash...),
		Cadena:         raizMerkle(c.hashesDeEntradas()),
		BaseLegal:      baseLegal,
		Instante:       instante,
	}
	l.Firma = ed25519.Sign(priv, l.contenidoFirmado())
	ks.Destruir(indice)
	c.Lapidas = append(c.Lapidas, l)
	return l, nil
}

// contenidoFirmado ata la lapida a UNA entrada concreta de UNA cadena concreta.
//
// HALLAZGO DE REVISION HOSTIL: antes era indice || base legal || instante. Sin
// el hash de la entrada ni identidad de cadena, una lapida legitima se pegaba
// tal cual en otra cadena y suprimia alli lo que ocupara el mismo indice: una
// supresion legal reciclada para tapar otra cosa, con la firma buena del
// operador. Ahora el hash de la envoltura cifrada entra en la firma, asi que la
// lapida solo vale para esa entrada, y la raiz de la cadena la ata al sitio.
func (l Lapida) contenidoFirmado() []byte {
	var idx [8]byte
	binary.BigEndian.PutUint64(idx[:], l.EntradaBorrada)
	m := append([]byte(etiquetaLapida), idx[:]...)
	m = append(m, l.HashEntrada...)
	m = append(m, []byte(l.Cadena)...)
	return append(m, []byte(l.BaseLegal+"|"+l.Instante)...)
}

func (c *CadenaV2) lapidaDe(indice uint64) *Lapida {
	for i := range c.Lapidas {
		if c.Lapidas[i].EntradaBorrada == indice {
			return &c.Lapidas[i]
		}
	}
	return nil
}

// InformeV2 es el resultado de verificar: la cadena, y las supresiones con su
// base legal. Un verificador independiente reporta "suprimida con base legal X
// el dia Y", nunca "posible manipulacion".
type InformeV2 struct {
	Entradas   int
	Suprimidas []string
}

// Verificar recalcula la cadena entera y valida las lapidas contra la clave
// publica del operador (que aporta el RECEPTOR, no el emisor).
func (c *CadenaV2) Verificar(cf Confianza) (InformeV2, error) {
	inf := InformeV2{Entradas: len(c.Entradas)}
	var previo []byte
	for i, e := range c.Entradas {
		if e.Indice != uint64(i) {
			return inf, fmt.Errorf("entrada %d: indice declarado %d", i, e.Indice)
		}
		if !bytes.Equal(e.Previo, previo) {
			return inf, fmt.Errorf("entrada %d: no encadena con la anterior", i)
		}
		if !bytes.Equal(e.Hash, hashEntradaV2(e)) {
			return inf, fmt.Errorf("entrada %d: %w", i, ErrEntradaAlterada)
		}
		previo = e.Hash
	}

	// La clave del operador la aporta el receptor. Comprobar el tamano antes de
	// usarla: ed25519.Verify hace panic con una clave de tamano equivocado, y
	// un fichero de anclas mal copiado no puede tumbar al verificador.
	if len(c.Lapidas) > 0 && len(cf.ClaveOperador) != ed25519.PublicKeySize {
		return inf, fmt.Errorf("hay %d lapida(s) que verificar y la clave del operador que aporta "+
			"el receptor mide %d bytes en vez de %d; revisa el fichero de claves",
			len(c.Lapidas), len(cf.ClaveOperador), ed25519.PublicKeySize)
	}
	vistas := map[uint64]bool{}
	for _, l := range c.Lapidas {
		if l.BaseLegal == "" {
			return inf, fmt.Errorf("lapida de la entrada %d %w", l.EntradaBorrada, ErrLapidaSinBaseLegal)
		}
		if l.EntradaBorrada >= uint64(len(c.Entradas)) {
			return inf, fmt.Errorf("lapida de la entrada %d, pero la cadena tiene %d entradas; %w",
				l.EntradaBorrada, len(c.Entradas), ErrLapidaFueraDeRango)
		}
		if vistas[l.EntradaBorrada] {
			return inf, fmt.Errorf("la entrada %d tiene mas de una lapida; %w",
				l.EntradaBorrada, ErrLapidaDuplicada)
		}
		vistas[l.EntradaBorrada] = true
		if !bytes.Equal(l.HashEntrada, c.Entradas[l.EntradaBorrada].Hash) {
			return inf, fmt.Errorf("lapida de la entrada %d: %w", l.EntradaBorrada, ErrLapidaDeOtraEntrada)
		}
		if !ed25519.Verify(cf.ClaveOperador, l.contenidoFirmado(), l.Firma) {
			return inf, fmt.Errorf("lapida de la entrada %d: %w", l.EntradaBorrada, ErrFirmaLapida)
		}
		inf.Suprimidas = append(inf.Suprimidas,
			fmt.Sprintf("entrada %d suprimida con base legal %s el %s", l.EntradaBorrada, l.BaseLegal, l.Instante))
	}

	for _, cp := range c.Checkpoints {
		if err := verificarCheckpointContra(cp, c.hashesDeEntradas(), cf); err != nil {
			return inf, err
		}
	}
	return inf, nil
}
