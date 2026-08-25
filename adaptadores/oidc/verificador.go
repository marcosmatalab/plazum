package oidc

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"math/big"
	"strings"
	"time"
)

// LimiteTokenBytes acota el tamano del ID token que se acepta parsear.
//
// Un ID token real ronda el kilobyte y medio; los de Entra ID con muchos grupos
// llegan a cuatro o cinco. Treinta y dos kibibytes es holgadisimo y deja fuera
// el cuerpo enorme que alguien mande para que gastemos base64 y JSON antes de
// llegar a comprobar la firma. El orden importa: primero el tamano, luego el
// parseo, luego la firma.
const LimiteTokenBytes = 32 << 10

// ErrToken es el error de un ID token que no se acepta. Todos los rechazos de
// la verificacion lo envuelven, para que quien llama pueda distinguir "el token
// no vale" de "no se pudo hablar con el IdP".
var ErrToken = errors.New("ID token rechazado")

// Verificador comprueba ID tokens contra un emisor y un juego de claves.
//
// Es seguro para uso concurrente. No guarda estado de la verificacion: el
// resultado depende solo del token, de las claves y del instante que se pasa.
type Verificador struct {
	cfg    Configuracion
	claves *Claves
}

// NuevoVerificador construye el verificador. Valida la configuracion aqui, al
// arrancar, y no en el primer login.
func NuevoVerificador(cfg Configuracion, claves *Claves) (*Verificador, error) {
	if err := cfg.validar(); err != nil {
		return nil, err
	}
	if claves == nil {
		return nil, fmt.Errorf("%w: no hay juego de claves. Sin JWKS no se verifica nada, "+
			"y un verificador que no verifica es peor que ninguno", ErrConfiguracion)
	}
	return &Verificador{cfg: cfg, claves: claves}, nil
}

// Esperado es lo que la verificacion exige ademas de lo que ya sabe por
// configuracion. Va aparte porque cambia en cada peticion y la configuracion no.
type Esperado struct {
	// Nonce es el que se genero al iniciar el flujo. Si no esta vacio, el
	// token TIENE que traerlo y coincidir. Vacio significa que quien llama
	// declara que no hubo nonce, y eso solo es legitimo fuera del flujo de
	// autenticacion; en el flujo lo pone [Autenticador] siempre.
	Nonce string
}

// Verificar comprueba el ID token y devuelve la identidad si pasa todo.
//
// El instante entra como dato a proposito: sin eso no se puede probar un token
// caducado sin dormir el test, y un test que duerme se acaba borrando o se
// vuelve inestable. Es la misma disciplina del nucleo.
//
// El orden de las comprobaciones no es casual. Lo barato y lo que no depende de
// criptografia va primero (tamano, forma, algoritmo), la firma va en medio, y
// las reclamaciones DESPUES de la firma: leer `iss` o `exp` de un token cuya
// firma no se ha comprobado es leer datos del atacante.
func (v *Verificador) Verificar(ctx context.Context, token string, ahora time.Time, esp Esperado) (Identidad, error) {
	var id Identidad

	if len(token) == 0 {
		return id, fmt.Errorf("%w: no hay token", ErrToken)
	}
	if len(token) > LimiteTokenBytes {
		return id, fmt.Errorf("%w: el token ocupa %d bytes y el maximo es %d",
			ErrToken, len(token), LimiteTokenBytes)
	}
	partes := strings.Split(token, ".")
	if len(partes) != 3 {
		return id, fmt.Errorf("%w: un JWS compacto tiene tres partes separadas por punto "+
			"y este tiene %d. Un JWE (cinco partes) tampoco vale: plazum no descifra ID "+
			"tokens, configura el IdP para que los firme sin cifrar", ErrToken, len(partes))
	}
	if partes[0] == "" || partes[1] == "" {
		return id, fmt.Errorf("%w: cabecera o cuerpo vacios", ErrToken)
	}
	// La firma vacia es el caso de `alg: none`, y se corta aqui ademas de en
	// la lista blanca. Dos cierres para el mismo agujero es deliberado: es el
	// fallo clasico de los verificadores de JWT.
	if partes[2] == "" {
		return id, fmt.Errorf("%w: el token no trae firma. Es la forma que toma "+
			"`alg: none`, y no se acepta nunca", ErrToken)
	}

	crudoCabecera, err := base64.RawURLEncoding.DecodeString(partes[0])
	if err != nil {
		return id, fmt.Errorf("%w: la cabecera no es base64url sin relleno: %v", ErrToken, err)
	}
	var cab struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
		Typ string `json:"typ"`
		Enc string `json:"enc"`
		Cty string `json:"cty"`
	}
	if err := json.Unmarshal(crudoCabecera, &cab); err != nil {
		return id, fmt.Errorf("%w: la cabecera no es JSON: %v", ErrToken, err)
	}
	if cab.Enc != "" {
		return id, fmt.Errorf("%w: la cabecera trae `enc`, o sea que es un token cifrado. "+
			"plazum no descifra ID tokens", ErrToken)
	}
	// `typ` es opcional en un ID token, pero cuando viene y dice otra cosa, el
	// token no es un ID token. El caso que importa es `at+jwt`: un access token
	// de RFC 9068 firmado por el mismo IdP con la misma clave, que pasaria la
	// firma. Aceptarlo seria confundir "esta firmado por el IdP" con "es la
	// prueba de que este usuario acaba de autenticarse".
	if cab.Typ != "" && !strings.EqualFold(cab.Typ, "JWT") {
		return id, fmt.Errorf("%w: la cabecera dice `typ: %s` y se esperaba un ID token. "+
			"Un access token firmado por el mismo IdP pasaria la firma y no es una prueba "+
			"de autenticacion", ErrToken, cab.Typ)
	}

	// El algoritmo lo decide el verificador, no el token. Primero la lista
	// blanca de la configuracion; despues, ya con la clave delante, la
	// compatibilidad con el tipo de clave. Al reves seria dejar que el token
	// eligiera.
	if !v.enListaBlanca(cab.Alg) {
		return id, fmt.Errorf("%w: el algoritmo %q no esta en la lista blanca (%s). "+
			"El token no elige con que se le verifica", ErrToken, cab.Alg,
			strings.Join(v.cfg.algoritmos(), ", "))
	}

	k, err := v.claves.Buscar(ctx, cab.Kid, ahora)
	if err != nil {
		return id, fmt.Errorf("%w: %v", ErrToken, err)
	}
	// Si el JWKS declara el algoritmo de la clave, manda el JWKS: el emisor ha
	// dicho para que sirve esa clave y el token no puede reinterpretarla.
	if k.alg != "" && k.alg != cab.Alg {
		return id, fmt.Errorf("%w: el token dice %q y el JWKS declara %q para la clave %q. "+
			"El uso de una clave lo fija quien la publica", ErrToken, cab.Alg, k.alg, k.kid)
	}
	if err := comprobarFirma(cab.Alg, k.pub, partes[0]+"."+partes[1], partes[2]); err != nil {
		return id, fmt.Errorf("%w: %v", ErrToken, err)
	}

	// A partir de aqui el contenido esta firmado por una clave del emisor, y
	// solo a partir de aqui se puede creer lo que dice.
	crudoCuerpo, err := base64.RawURLEncoding.DecodeString(partes[1])
	if err != nil {
		return id, fmt.Errorf("%w: el cuerpo no es base64url sin relleno: %v", ErrToken, err)
	}
	var r reclamaciones
	if err := json.Unmarshal(crudoCuerpo, &r); err != nil {
		return id, fmt.Errorf("%w: el cuerpo no es JSON: %v", ErrToken, err)
	}
	if err := v.comprobarReclamaciones(&r, ahora, esp); err != nil {
		return id, err
	}

	id = Identidad{
		Sujeto:           r.Sub,
		Emisor:           r.Iss,
		Correo:           r.Email,
		CorreoVerificado: r.EmailVerificado != nil && *r.EmailVerificado,
		Nombre:           r.Name,
		Emitido:          time.Unix(*r.Iat, 0).UTC(),
		Caduca:           time.Unix(*r.Exp, 0).UTC(),
	}
	if r.AuthTime != nil {
		id.Autenticado = time.Unix(*r.AuthTime, 0).UTC()
	}
	return id, nil
}

func (v *Verificador) enListaBlanca(alg string) bool {
	for _, a := range v.cfg.algoritmos() {
		if a == alg {
			return true
		}
	}
	return false
}

// reclamaciones es el subconjunto del cuerpo que se usa.
//
// Los tiempos son punteros a proposito: hace falta distinguir "vale cero" de
// "no viene". Un `exp` ausente decodificado como 0 seria un token caducado en
// 1970, que se rechaza por casualidad; con puntero se rechaza por el motivo
// correcto y el mensaje lo dice.
type reclamaciones struct {
	Iss             string    `json:"iss"`
	Sub             string    `json:"sub"`
	Aud             audiencia `json:"aud"`
	Azp             string    `json:"azp"`
	Exp             *int64    `json:"exp"`
	Nbf             *int64    `json:"nbf"`
	Iat             *int64    `json:"iat"`
	AuthTime        *int64    `json:"auth_time"`
	Nonce           string    `json:"nonce"`
	Email           string    `json:"email"`
	EmailVerificado *bool     `json:"email_verified"`
	Name            string    `json:"name"`
}

// audiencia acepta las dos formas que permite la especificacion: una cadena o
// un array de cadenas.
type audiencia []string

func (a *audiencia) UnmarshalJSON(b []byte) error {
	var una string
	if err := json.Unmarshal(b, &una); err == nil {
		*a = audiencia{una}
		return nil
	}
	var varias []string
	if err := json.Unmarshal(b, &varias); err != nil {
		return errors.New("`aud` no es ni una cadena ni un array de cadenas")
	}
	*a = varias
	return nil
}

func (v *Verificador) comprobarReclamaciones(r *reclamaciones, ahora time.Time, esp Esperado) error {
	margen := v.cfg.margen()

	// `iss` exacto. Comparacion byte a byte, sin normalizar barras finales ni
	// mayusculas: "parecido" no existe aqui.
	if r.Iss != v.cfg.Emisor {
		return fmt.Errorf("%w: el emisor del token es %q y se esperaba %q", ErrToken, r.Iss, v.cfg.Emisor)
	}
	if strings.TrimSpace(r.Sub) == "" {
		return fmt.Errorf("%w: el token no trae `sub`, que es el identificador del usuario. "+
			"Sin el no hay a quien abrirle sesion", ErrToken)
	}
	// `aud` tiene que CONTENER el cliente_id.
	if !r.Aud.contiene(v.cfg.ClienteID) {
		return fmt.Errorf("%w: la audiencia del token es %v y no incluye el cliente_id %q. "+
			"Si acabas de configurar el IdP, el cliente_id pegado no es el de esta "+
			"aplicacion", ErrToken, []string(r.Aud), v.cfg.ClienteID)
	}
	// Con varias audiencias, OpenID Connect exige `azp` y que sea el nuestro.
	// Es lo que impide que un token emitido para OTRA aplicacion del mismo
	// tenant, que nos lista de paso en `aud`, sirva para entrar aqui.
	if len(r.Aud) > 1 {
		if r.Azp == "" {
			return fmt.Errorf("%w: el token trae %d audiencias y no trae `azp`. Con varias "+
				"audiencias hay que saber para que aplicacion se emitio", ErrToken, len(r.Aud))
		}
		if r.Azp != v.cfg.ClienteID {
			return fmt.Errorf("%w: el token se emitio para la aplicacion %q (`azp`) y esta es "+
				"%q. Un token de otra aplicacion del mismo IdP no abre sesion aqui",
				ErrToken, r.Azp, v.cfg.ClienteID)
		}
	}
	// `exp` obligatorio.
	if r.Exp == nil {
		return fmt.Errorf("%w: el token no trae `exp`. Un token sin caducidad no caduca, "+
			"y eso no es un ID token", ErrToken)
	}
	exp := time.Unix(*r.Exp, 0).UTC()
	if !ahora.Before(exp.Add(margen)) {
		return fmt.Errorf("%w: el token caduco a las %s y ahora son las %s (margen de reloj %s)",
			ErrToken, exp.Format(time.RFC3339), ahora.UTC().Format(time.RFC3339), margen)
	}
	// `nbf` opcional; si viene, se respeta.
	if r.Nbf != nil {
		nbf := time.Unix(*r.Nbf, 0).UTC()
		if ahora.Before(nbf.Add(-margen)) {
			return fmt.Errorf("%w: el token no vale hasta las %s y ahora son las %s "+
				"(margen de reloj %s). Si esto pasa siempre, el reloj de esta maquina y el "+
				"del IdP no estan sincronizados: el arreglo es NTP",
				ErrToken, nbf.Format(time.RFC3339), ahora.UTC().Format(time.RFC3339), margen)
		}
	}
	// `iat` obligatorio en OpenID Connect, y no puede estar en el futuro mas
	// alla del margen: un `iat` futuro es un reloj roto o un token fabricado.
	if r.Iat == nil {
		return fmt.Errorf("%w: el token no trae `iat`. OpenID Connect lo exige", ErrToken)
	}
	iat := time.Unix(*r.Iat, 0).UTC()
	if iat.After(ahora.Add(margen)) {
		return fmt.Errorf("%w: el token dice haberse emitido a las %s, que es el futuro "+
			"(ahora son las %s, margen de reloj %s)",
			ErrToken, iat.Format(time.RFC3339), ahora.UTC().Format(time.RFC3339), margen)
	}
	if exp.Before(iat) {
		return fmt.Errorf("%w: el token caduca (%s) antes de emitirse (%s)",
			ErrToken, exp.Format(time.RFC3339), iat.Format(time.RFC3339))
	}
	// `nonce`: ata el token a la peticion que lo origino. Sin el, un ID token
	// valido capturado en otro sitio se puede reinyectar en nuestro retorno.
	if esp.Nonce != "" {
		if r.Nonce == "" {
			return fmt.Errorf("%w: se pidio `nonce` y el token no lo trae. El IdP tiene que "+
				"devolver en el ID token el nonce que se le mando", ErrToken)
		}
		if subtle.ConstantTimeCompare([]byte(r.Nonce), []byte(esp.Nonce)) != 1 {
			return fmt.Errorf("%w: el `nonce` del token no es el de esta peticion. Es "+
				"exactamente lo que pasa cuando se reinyecta un token de otra sesion", ErrToken)
		}
	}
	return nil
}

func (a audiencia) contiene(s string) bool {
	for _, x := range a {
		if subtle.ConstantTimeCompare([]byte(x), []byte(s)) == 1 {
			return true
		}
	}
	return false
}

// comprobarFirma verifica la firma exigiendo que el algoritmo case con el TIPO
// de la clave.
//
// Aqui esta la mitad importante de la regla del encargo. La lista blanca sola
// no basta: con ella, un token que dice ES256 verificado contra una clave RSA
// tendria que fallar por casualidad. Esta funcion lo hace fallar por decision,
// y con nombre.
func comprobarFirma(alg string, pub crypto.PublicKey, firmado, firmaB64 string) error {
	firma, err := base64.RawURLEncoding.DecodeString(firmaB64)
	if err != nil {
		return fmt.Errorf("la firma no es base64url sin relleno: %w", err)
	}
	h, tam, err := resumen(alg, firmado)
	if err != nil {
		return err
	}
	switch p := pub.(type) {
	case *rsa.PublicKey:
		switch {
		case strings.HasPrefix(alg, "RS"):
			if err := rsa.VerifyPKCS1v15(p, tam, h, firma); err != nil {
				return errors.New("la firma no valida contra la clave que dice el `kid`")
			}
			return nil
		case strings.HasPrefix(alg, "PS"):
			if err := rsa.VerifyPSS(p, tam, h, firma, &rsa.PSSOptions{
				SaltLength: rsa.PSSSaltLengthEqualsHash,
				Hash:       tam,
			}); err != nil {
				return errors.New("la firma no valida contra la clave que dice el `kid`")
			}
			return nil
		default:
			return fmt.Errorf("el token dice %q y la clave del `kid` es RSA: el algoritmo "+
				"tiene que casar con el tipo de la clave, no al reves", alg)
		}
	case *ecdsa.PublicKey:
		if !strings.HasPrefix(alg, "ES") {
			return fmt.Errorf("el token dice %q y la clave es de curva eliptica: el algoritmo "+
				"tiene que casar con el tipo de la clave, no al reves", alg)
		}
		// La curva la fija el algoritmo, no la clave: ES256 es P-256 y punto.
		// Si no casan, la firma podria validar con una clave que el IdP publico
		// para otra cosa.
		esperada := map[string]int{"ES256": 32, "ES384": 48, "ES512": 66}[alg]
		if bytesCurva(p) != esperada {
			return fmt.Errorf("el token dice %q, que es la curva de %d bytes, y la clave %q "+
				"es de %d. No se verifica cruzado", alg, esperada, p.Curve.Params().Name, bytesCurva(p))
		}
		// En JWS la firma ECDSA es R||S en crudo, cada uno del tamano de la
		// curva. Un DER de X.509 aqui NO se acepta: aceptar los dos formatos
		// da dos codificaciones para la misma firma y abre maleabilidad.
		if len(firma) != 2*esperada {
			return fmt.Errorf("la firma ECDSA ocupa %d bytes y %s exige %d (R y S en crudo, "+
				"sin DER)", len(firma), alg, 2*esperada)
		}
		r := new(big.Int).SetBytes(firma[:esperada])
		s := new(big.Int).SetBytes(firma[esperada:])
		if !ecdsa.Verify(p, h, r, s) {
			return errors.New("la firma no valida contra la clave que dice el `kid`")
		}
		return nil
	default:
		return fmt.Errorf("tipo de clave no admitido para verificar %q", alg)
	}
}

func bytesCurva(p *ecdsa.PublicKey) int {
	return (p.Curve.Params().BitSize + 7) / 8
}

// resumen calcula el hash del material firmado con el tamano que pide el
// algoritmo.
func resumen(alg, firmado string) ([]byte, crypto.Hash, error) {
	var h hash.Hash
	var cual crypto.Hash
	switch alg {
	case "RS256", "PS256", "ES256":
		h, cual = sha256.New(), crypto.SHA256
	case "RS384", "PS384", "ES384":
		h, cual = sha512.New384(), crypto.SHA384
	case "RS512", "PS512", "ES512":
		h, cual = sha512.New(), crypto.SHA512
	default:
		return nil, 0, fmt.Errorf("algoritmo %q sin funcion resumen conocida", alg)
	}
	h.Write([]byte(firmado))
	return h.Sum(nil), cual, nil
}
