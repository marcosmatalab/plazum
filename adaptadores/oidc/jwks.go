package oidc

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"
)

// MaxClaves es el tope de claves que se aceptan de un JWKS.
//
// Un IdP real publica entre una y cuatro (la vigente, la siguiente y alguna en
// rotacion). Un JWKS con miles de claves no es un IdP con muchas claves: es un
// IdP hostil convirtiendo cada verificacion en un recorrido caro, o gastando
// nuestra memoria. Cincuenta deja sitio de sobra para cualquier rotacion
// razonable y cierra la puerta a lo demas.
const MaxClaves = 50

// BitsMinimosRSA es el tamano minimo de modulo RSA que se acepta.
//
// Una clave RSA de 512 bits se factoriza en un portatil. Aceptarla porque el
// JWKS la publique seria dejar que el IdP elija cuanta seguridad tenemos, que
// es la misma clase de error que dejar que el token elija el algoritmo.
const BitsMinimosRSA = 2048

// IntervaloRecargaPorDefecto es lo minimo que pasa entre dos lecturas del JWKS.
//
// La razon es concreta y esta en el encargo: un `kid` desconocido dispara una
// recarga, porque asi es como se sobrevive a una rotacion de claves sin
// reiniciar. Sin tope, alguien que mande mil tokens con mil `kid` inventados
// provoca mil peticiones al IdP, y eso es un ataque de amplificacion contra el
// IdP con nuestra IP en los logs. Con tope, mil `kid` inventados provocan como
// mucho una peticion por minuto.
const IntervaloRecargaPorDefecto = time.Minute

// ErrJWKS es el error de un juego de claves que no se puede usar.
var ErrJWKS = errors.New("JWKS invalido")

// ErrClaveDesconocida es el error de un `kid` que no esta en el JWKS ni
// aparecio al recargarlo. Se distingue del resto porque es el caso normal de
// una rotacion a medias, y quien llama puede querer reintentar mas tarde.
var ErrClaveDesconocida = errors.New("kid no encontrado en el JWKS")

// jwk es una clave del juego, en el formato de RFC 7517. Solo se leen los
// campos que se usan.
type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Uso string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

type juego struct {
	Claves []jwk `json:"keys"`
}

// clave es una clave ya parseada y lista para verificar.
type clave struct {
	kid string
	// alg es el algoritmo que el JWKS declara para esta clave. Vacio si el
	// JWKS no lo dice, que es legitimo y frecuente.
	alg string
	pub crypto.PublicKey
}

// Claves es el juego de claves del IdP, cacheado, con recarga acotada.
//
// Es seguro para uso concurrente: el login de una empresa de 200 personas un
// lunes a las nueve son 200 verificaciones a la vez, y la recarga tiene que
// ocurrir una sola vez.
type Claves struct {
	uri       string
	cliente   *http.Client
	intervalo time.Duration

	// recargaMu serializa las lecturas del JWKS. Es un candado aparte del que
	// protege el mapa, y no es un detalle de eficiencia.
	//
	// HALLAZGO de la pasada del atacante. Con un solo candado, la cache fria
	// se comporta asi: 64 personas entran a la vez el lunes a las nueve, las
	// 64 miran la cache, las 64 la ven vacia, y las 64 salen a por el JWKS.
	// El test medio 18 lecturas donde tenia que haber 1. El limite de
	// intervalo no lo tapaba, porque el intervalo solo cuenta desde la
	// PRIMERA lectura y ahi todavia no habia ninguna. Es una estampida contra
	// el IdP provocada por el uso normal, y bastaria un login masivo para
	// que el IdP nos limite a nosotros.
	recargaMu sync.Mutex

	mu            sync.Mutex
	porKid        map[string]clave
	todas         []clave
	ultimaRecarga time.Time
	ultimoError   error
	recargas      int
	cargado       bool
}

// NuevasClaves crea el juego de claves apuntando al jwks_uri del
// descubrimiento. No sale a la red todavia: la primera lectura ocurre en la
// primera verificacion, para que el arranque no dependa de que el IdP este
// levantado.
func NuevasClaves(uri string, cliente *http.Client) *Claves {
	if cliente == nil {
		cliente = ClientePorDefecto()
	}
	return &Claves{
		uri:       uri,
		cliente:   cliente,
		intervalo: IntervaloRecargaPorDefecto,
		porKid:    map[string]clave{},
	}
}

// ConIntervaloRecarga cambia el minimo entre recargas. Sirve para los tests y
// para un IdP que rote muy rapido; no para desactivarlo: cero se toma como el
// valor por defecto, no como "sin limite".
func (c *Claves) ConIntervaloRecarga(d time.Duration) *Claves {
	if d <= 0 {
		d = IntervaloRecargaPorDefecto
	}
	c.mu.Lock()
	c.intervalo = d
	c.mu.Unlock()
	return c
}

// Recargas dice cuantas veces se ha ido de verdad al IdP. Es lo que permite
// demostrar en un test que mil `kid` inventados no provocan mil peticiones.
func (c *Claves) Recargas() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.recargas
}

// Buscar devuelve la clave de un `kid`, recargando el JWKS si no la conoce y
// el intervalo lo permite.
//
// El caso del `kid` vacio: hay IdPs que no lo ponen. Se acepta SOLO si el JWKS
// tiene exactamente una clave, porque entonces no hay eleccion que hacer. Con
// dos o mas, un token sin `kid` se rechaza en vez de probar todas: probar todas
// convierte el juego de claves en un oraculo y borra la ventaja de la rotacion.
func (c *Claves) Buscar(ctx context.Context, kid string, ahora time.Time) (clave, error) {
	if k, ok := c.buscarEnCache(kid); ok {
		return k, nil
	}
	if err := c.recargar(ctx, ahora); err != nil {
		return clave{}, err
	}
	if k, ok := c.buscarEnCache(kid); ok {
		return k, nil
	}
	if kid == "" {
		return clave{}, fmt.Errorf("%w: el token no trae `kid` y el JWKS publica varias claves, "+
			"asi que no hay forma de saber cual uso el IdP. Configura el IdP para que emita "+
			"`kid` en la cabecera del ID token", ErrClaveDesconocida)
	}
	return clave{}, fmt.Errorf("%w: kid %q. Si el IdP acaba de rotar sus claves, se resuelve "+
		"solo en cuanto se pueda recargar el JWKS; si no, el token no lo firmo este emisor", ErrClaveDesconocida, kid)
}

func (c *Claves) buscarEnCache(kid string) (clave, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.cargado {
		return clave{}, false
	}
	if kid == "" {
		if len(c.todas) == 1 {
			return c.todas[0], true
		}
		return clave{}, false
	}
	k, ok := c.porKid[kid]
	return k, ok
}

// recargar va al IdP si toca. Devuelve nil sin hacer nada cuando el intervalo
// no ha pasado y ya hay algo cargado: no es un error, es el limite haciendo su
// trabajo, y quien llama distingue el caso porque la clave sigue sin aparecer.
func (c *Claves) recargar(ctx context.Context, ahora time.Time) error {
	// Solo una lectura del JWKS a la vez. Quien llegue mientras otro esta en
	// la red espera aqui y, al entrar, se encuentra el trabajo ya hecho: la
	// comprobacion de intervalo de abajo le devuelve el resultado de la
	// lectura que acaba de terminar, sin salir otra vez.
	c.recargaMu.Lock()
	defer c.recargaMu.Unlock()

	c.mu.Lock()
	if !c.ultimaRecarga.IsZero() && ahora.Sub(c.ultimaRecarga) < c.intervalo {
		// Se devuelve el error de la ULTIMA lectura, no nil. Si el JWKS no se
		// pudo leer, quien llegue dentro del intervalo tiene que enterarse de
		// por que, en vez de recibir un "kid desconocido" que manda a mirar
		// donde no es.
		err := c.ultimoError
		c.mu.Unlock()
		return err
	}
	// Se marca ANTES de salir a la red, no despues. Si se marcara despues, el
	// intervalo empezaria a contar desde que TERMINA la lectura, y con un IdP
	// lento eso alarga la ventana en la que se puede volver a pedir.
	c.ultimaRecarga = ahora
	c.recargas++
	uri := c.uri
	cliente := c.cliente
	c.mu.Unlock()

	guardarError := func(err error) error {
		c.mu.Lock()
		c.ultimoError = err
		c.mu.Unlock()
		return err
	}

	cuerpo, err := traer(ctx, cliente, uri)
	if err != nil {
		return guardarError(fmt.Errorf("%w: no se pudo leer %s: %v", ErrJWKS, uri, err))
	}
	var j juego
	if err := json.Unmarshal(cuerpo, &j); err != nil {
		return guardarError(fmt.Errorf("%w: %s no devolvio un juego de claves JSON: %v", ErrJWKS, uri, err))
	}
	if len(j.Claves) == 0 {
		return guardarError(fmt.Errorf("%w: %s no publica ninguna clave. Sin claves no se puede "+
			"verificar ningun token", ErrJWKS, uri))
	}
	if len(j.Claves) > MaxClaves {
		return guardarError(fmt.Errorf("%w: %s publica %d claves y el maximo es %d. Un juego de claves "+
			"asi no es una rotacion, es un problema", ErrJWKS, uri, len(j.Claves), MaxClaves))
	}
	porKid := map[string]clave{}
	var todas []clave
	for _, k := range j.Claves {
		// `use` distinto de "sig" es una clave de cifrado: no verifica firmas.
		if k.Uso != "" && k.Uso != "sig" {
			continue
		}
		pub, err := parsearJWK(k)
		if err != nil {
			// Una clave rota no invalida el juego: el IdP puede publicar un
			// tipo que no conocemos (por ejemplo OKP) junto a las que si.
			// Invalidar todo por eso seria caerse por una clave que no se iba
			// a usar.
			continue
		}
		c2 := clave{kid: k.Kid, alg: k.Alg, pub: pub}
		todas = append(todas, c2)
		if k.Kid != "" {
			porKid[k.Kid] = c2
		}
	}
	if len(todas) == 0 {
		return guardarError(fmt.Errorf("%w: %s publica %d claves y ninguna es utilizable para verificar "+
			"firmas (se admiten RSA de %d bits o mas y EC P-256, P-384 y P-521)",
			ErrJWKS, uri, len(j.Claves), BitsMinimosRSA))
	}
	c.mu.Lock()
	c.porKid = porKid
	c.todas = todas
	c.cargado = true
	c.ultimoError = nil
	c.mu.Unlock()
	return nil
}

func parsearJWK(k jwk) (crypto.PublicKey, error) {
	switch k.Kty {
	case "RSA":
		n, err := enteroBase64(k.N)
		if err != nil {
			return nil, err
		}
		e, err := enteroBase64(k.E)
		if err != nil {
			return nil, err
		}
		if !e.IsInt64() || e.Int64() < 3 || e.Int64() > 1<<31 {
			return nil, errors.New("exponente RSA fuera de rango")
		}
		pub := &rsa.PublicKey{N: n, E: int(e.Int64())}
		if pub.N.BitLen() < BitsMinimosRSA {
			return nil, fmt.Errorf("clave RSA de %d bits, por debajo del minimo %d",
				pub.N.BitLen(), BitsMinimosRSA)
		}
		return pub, nil
	case "EC":
		var curva elliptic.Curve
		switch k.Crv {
		case "P-256":
			curva = elliptic.P256()
		case "P-384":
			curva = elliptic.P384()
		case "P-521":
			curva = elliptic.P521()
		default:
			return nil, fmt.Errorf("curva %q no admitida", k.Crv)
		}
		x, err := enteroBase64(k.X)
		if err != nil {
			return nil, err
		}
		y, err := enteroBase64(k.Y)
		if err != nil {
			return nil, err
		}
		pub := &ecdsa.PublicKey{Curve: curva, X: x, Y: y}
		// Un punto que no esta en la curva no es una clave: verificar con el
		// puede dar resultados que no significan nada. crypto/ecdsa ya no
		// acepta puntos fuera de la curva, pero comprobarlo aqui deja el
		// motivo escrito y falla al cargar en vez de al verificar.
		if !curva.IsOnCurve(x, y) {
			return nil, errors.New("el punto publicado no esta en la curva")
		}
		return pub, nil
	default:
		return nil, fmt.Errorf("tipo de clave %q no admitido", k.Kty)
	}
}

// enteroBase64 decodifica un entero big-endian en base64url sin relleno, que es
// como RFC 7518 codifica los parametros de las claves.
func enteroBase64(s string) (*big.Int, error) {
	if s == "" {
		return nil, errors.New("parametro de clave vacio")
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("parametro de clave que no es base64url sin relleno: %w", err)
	}
	return new(big.Int).SetBytes(b), nil
}
