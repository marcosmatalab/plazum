// Package oidc es la entrada de identidad: descubrimiento OIDC, JWKS,
// verificacion del ID token y el flujo de codigo de autorizacion con PKCE.
//
// # Por que a mano y sin dependencias
//
// Esto vive en la frontera de confianza. El ID token lo emite alguien de fuera
// y lo transporta el navegador del usuario, o sea que cada byte que se parsea
// aqui viene de un sitio en el que no mandamos. Un verificador de JWT que se
// salta un paso no falla ruidosamente: deja entrar a cualquiera y todo parece
// funcionar. Por eso el codigo esta escrito con crypto/rsa, crypto/ecdsa,
// crypto/sha256, encoding/base64, encoding/json y net/http de la biblioteca
// estandar, y por eso cada comprobacion tiene un test que falsifica el token
// justo de esa forma y demuestra que se rechaza.
//
// # La regla que gobierna la verificacion
//
// El algoritmo lo decide QUIEN VERIFICA, nunca el token. La cabecera del JWT
// propone un `alg` y una `kid`; aqui las dos son entrada no fiable. El `alg`
// tiene que estar en la lista blanca Y ser compatible con el tipo de la clave
// que dice el JWKS; si el JWKS declara `alg` para esa clave, tiene que
// coincidir. `alg: none` no se acepta jamas, no porque este en una lista negra
// sino porque no esta en la blanca.
//
// # El instante entra como dato
//
// [Verificador.Verificar] recibe `ahora`. No llama a time.Now. Sin eso no se
// puede probar un token caducado sin dormir el test, y un test que duerme se
// acaba borrando. Es la misma disciplina que el nucleo, aplicada aqui porque
// aqui tambien decide una comprobacion de seguridad.
//
// # Lo que este paquete NO hace
//
//   - No monta un servidor ni registra rutas: entrega piezas que el que
//     construye el servidor cablea. Ver [Autenticador].
//   - No refresca tokens ni habla con el endpoint de userinfo. El ID token
//     trae la identidad; lo demas es superficie de ataque sin comprador.
//   - No implementa SAML. Esta apuntado para el ano 2.
package oidc

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// AlgoritmosPorDefecto es la lista blanca de firma con la que se arranca si no
// se dice otra cosa.
//
// Son los que emiten de verdad Entra ID (RS256), Okta (RS256), Google (RS256)
// y Keycloak (RS256 o ES256). Estan las variantes de 384 y 512 porque un IdP
// puede configurarse asi y rechazarlas seria un fallo de interoperabilidad sin
// ganancia de seguridad.
//
// NO estan, y no es olvido: `none` (no es firma), `HS256` y las demas HMAC.
// Las HMAC con JWKS son el ataque de confusion de algoritmo de manual: la
// clave publica RSA del IdP, que es publica, pasaria a ser el secreto HMAC y
// cualquiera podria firmar. Que no esten aqui hace ese ataque imposible de
// expresar, no solo dificil.
var AlgoritmosPorDefecto = []string{
	"RS256", "RS384", "RS512",
	"PS256", "PS384", "PS512",
	"ES256", "ES384", "ES512",
}

// MargenRelojMaximo acota el margen de reloj configurable.
//
// El margen existe porque el reloj del IdP y el nuestro no son el mismo y un
// token recien emitido puede parecer futuro por unos segundos. Pero un margen
// grande es una caducidad grande: con quince minutos, un token robado sigue
// valiendo quince minutos despues de expirar. Cinco es el techo, y la
// configuracion que lo pase falla al construirse, no en produccion.
const MargenRelojMaximo = 5 * time.Minute

// MargenRelojPorDefecto es lo que se usa si no se dice nada.
const MargenRelojPorDefecto = 60 * time.Second

// Configuracion es lo que el administrador pega de su IdP.
//
// Los nombres de los campos son los que el propio IdP usa en su pantalla, para
// que copiar y pegar no exija traducir: `issuer`, `client_id`,
// `client_secret`, `redirect_uri`.
type Configuracion struct {
	// Emisor es el `issuer` del IdP, tal cual, incluida la barra final si la
	// lleva. Se compara byte a byte contra el `iss` del token: un emisor
	// "parecido" es un emisor distinto.
	Emisor string
	// ClienteID es el identificador de la aplicacion registrada.
	ClienteID string
	// ClienteSecreto es el secreto de la aplicacion. JAMAS aparece en un
	// error, en un log ni en el String de este tipo. Vacio significa cliente
	// publico, que con PKCE es legitimo.
	ClienteSecreto string
	// RedirectURI es la URL de retorno registrada en el IdP. Se manda al IdP
	// y se manda al canje, y en los dos sitios sale de AQUI, nunca de la
	// peticion: una redirect_uri que viene de fuera es un redirector abierto.
	RedirectURI string
	// Ambitos son los scopes. Vacio significa openid, profile, email.
	Ambitos []string
	// Algoritmos es la lista blanca de firma. Vacio significa
	// AlgoritmosPorDefecto.
	Algoritmos []string
	// MargenReloj tolera la deriva entre el reloj del IdP y el nuestro. Cero
	// significa MargenRelojPorDefecto. Mas de MargenRelojMaximo no se acepta.
	MargenReloj time.Duration
}

// String redacta el secreto. Existe para que un `%v` distraido en un log o en
// un mensaje de error no publique la credencial del cliente.
//
// No es cosmetica: un client_secret en una traza es un incidente de seguridad
// con notificacion, no una molestia.
func (c Configuracion) String() string {
	sec := "(vacio)"
	if c.ClienteSecreto != "" {
		sec = "(redactado)"
	}
	return fmt.Sprintf("oidc.Configuracion{Emisor:%q ClienteID:%q ClienteSecreto:%s RedirectURI:%q}",
		c.Emisor, c.ClienteID, sec, c.RedirectURI)
}

// GoString redacta igual que String, para el verbo %#v.
func (c Configuracion) GoString() string { return c.String() }

// ErrConfiguracion es el error de una configuracion que no se puede usar. Se
// devuelve al construir, no al autenticar: un IdP mal configurado tiene que
// romper el arranque, no el primer login de un martes por la tarde.
var ErrConfiguracion = errors.New("configuracion de OIDC invalida")

// validar comprueba lo que se puede comprobar sin salir a la red, y cada
// mensaje dice donde se arregla en el IdP.
func (c *Configuracion) validar() error {
	if strings.TrimSpace(c.Emisor) == "" {
		return fmt.Errorf("%w: falta el emisor. Es el campo `issuer` del IdP; "+
			"en Entra ID es https://login.microsoftonline.com/<tenant>/v2.0 y en Okta "+
			"https://<dominio>.okta.com/oauth2/default", ErrConfiguracion)
	}
	u, err := url.Parse(c.Emisor)
	if err != nil || u.Host == "" {
		return fmt.Errorf("%w: el emisor %q no es una URL. Pega el valor `issuer` "+
			"del documento de descubrimiento del IdP, sin comillas ni espacios",
			ErrConfiguracion, c.Emisor)
	}
	if !esquemaSeguro(u) {
		return fmt.Errorf("%w: el emisor %q no usa https. Solo se admite http contra "+
			"127.0.0.1 o localhost, y eso es para desarrollo: en produccion un emisor "+
			"sin TLS deja el ID token a la vista de la red", ErrConfiguracion, c.Emisor)
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("%w: el emisor %q lleva query o fragmento, y el `iss` de "+
			"OpenID Connect no los lleva. Quita todo lo que vaya detras de la ruta",
			ErrConfiguracion, c.Emisor)
	}
	if strings.TrimSpace(c.ClienteID) == "" {
		return fmt.Errorf("%w: falta el cliente_id. Es el `Application (client) ID` "+
			"de Entra ID o el `Client ID` de la integracion de Okta", ErrConfiguracion)
	}
	if strings.TrimSpace(c.RedirectURI) == "" {
		return fmt.Errorf("%w: falta la redirect_uri. Tiene que ser la MISMA cadena "+
			"que esta registrada en el IdP, caracter a caracter: "+
			"https://<tu-plazum>/auth/retorno", ErrConfiguracion)
	}
	r, err := url.Parse(c.RedirectURI)
	if err != nil || r.Host == "" || !r.IsAbs() {
		return fmt.Errorf("%w: la redirect_uri %q no es una URL absoluta. Tiene que "+
			"empezar por https:// e incluir el host", ErrConfiguracion, c.RedirectURI)
	}
	if !esquemaSeguro(r) {
		return fmt.Errorf("%w: la redirect_uri %q no usa https. El codigo de "+
			"autorizacion viaja por ahi", ErrConfiguracion, c.RedirectURI)
	}
	if r.Fragment != "" {
		return fmt.Errorf("%w: la redirect_uri %q lleva fragmento, y OAuth 2.0 lo "+
			"prohibe. Quita todo lo que vaya detras de la almohadilla",
			ErrConfiguracion, c.RedirectURI)
	}
	if c.MargenReloj < 0 {
		return fmt.Errorf("%w: el margen de reloj no puede ser negativo (%s)",
			ErrConfiguracion, c.MargenReloj)
	}
	if c.MargenReloj > MargenRelojMaximo {
		return fmt.Errorf("%w: el margen de reloj %s pasa del maximo %s. Un margen "+
			"grande es una caducidad grande: un token robado sigue valiendo todo ese "+
			"tiempo despues de expirar. Si el reloj del IdP y el tuyo difieren mas que "+
			"eso, el arreglo es NTP, no el margen", ErrConfiguracion, c.MargenReloj, MargenRelojMaximo)
	}
	for _, a := range c.Algoritmos {
		if !algoritmoConocido(a) {
			return fmt.Errorf("%w: el algoritmo %q no se admite. Los admitidos son %s. "+
				"Las familias HMAC (HS256 y companeras) estan fuera a proposito: con un "+
				"JWKS convierten la clave PUBLICA del IdP en el secreto de firma",
				ErrConfiguracion, a, strings.Join(AlgoritmosPorDefecto, ", "))
		}
	}
	return nil
}

// algoritmos devuelve la lista blanca efectiva.
func (c *Configuracion) algoritmos() []string {
	if len(c.Algoritmos) == 0 {
		return AlgoritmosPorDefecto
	}
	return c.Algoritmos
}

// margen devuelve el margen de reloj efectivo.
func (c *Configuracion) margen() time.Duration {
	if c.MargenReloj == 0 {
		return MargenRelojPorDefecto
	}
	return c.MargenReloj
}

// ambitos devuelve los scopes efectivos.
func (c *Configuracion) ambitos() []string {
	if len(c.Ambitos) == 0 {
		return []string{"openid", "profile", "email"}
	}
	return c.Ambitos
}

func algoritmoConocido(a string) bool {
	for _, x := range AlgoritmosPorDefecto {
		if x == a {
			return true
		}
	}
	return false
}

// esquemaSeguro exige https, con la excepcion del bucle local.
//
// La excepcion no es una puerta trasera: en 127.0.0.1 y ::1 no hay red que
// escuchar, y sin ella no se puede levantar un IdP de prueba ni correr los
// tests de este paquete, que es tanto como decir que no se puede probar la
// verificacion. Es la misma excepcion que hacen las guias de OAuth para los
// clientes nativos.
func esquemaSeguro(u *url.URL) bool {
	if u.Scheme == "https" {
		return true
	}
	if u.Scheme != "http" {
		return false
	}
	h := u.Hostname()
	return h == "127.0.0.1" || h == "::1" || h == "localhost"
}

// Identidad es lo que sale de un ID token verificado. Es el minimo que hace
// falta para abrir sesion y para casar con el directorio de SCIM.
type Identidad struct {
	// Sujeto es el `sub`: el identificador estable del usuario EN EL IdP. Es
	// lo que se guarda, no el correo: el correo cambia cuando alguien se casa
	// y el `sub` no.
	Sujeto string
	// Emisor es el `iss` verificado.
	Emisor string
	// Correo, si el IdP lo publica.
	Correo string
	// CorreoVerificado es el `email_verified` tal cual llega. Falso tambien
	// cuando el IdP no lo publica: no se supone verificado lo que nadie dijo.
	CorreoVerificado bool
	// Nombre para mostrar, si viene.
	Nombre string
	// Emitido es el `iat`.
	Emitido time.Time
	// Caduca es el `exp`.
	Caduca time.Time
	// Autenticado es el `auth_time` si el IdP lo publica; cero si no.
	Autenticado time.Time
}
