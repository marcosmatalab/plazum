package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// LimiteCuerpo es el tope de bytes que se leen de cualquier respuesta del IdP.
//
// El IdP es un tercero, y un tercero hostil (o simplemente roto) puede devolver
// un cuerpo infinito. Sin tope, el proceso se come la memoria y el producto que
// vigila plazos legales se cae mientras corre un plazo. 512 KiB es holgado: un
// documento de descubrimiento ronda los 3 KiB y un JWKS con diez claves RSA
// ronda los 6 KiB.
const LimiteCuerpo = 512 << 10

// TiempoLimitePorDefecto acota lo que se espera al IdP.
//
// Un IdP que responde despacio no puede convertirse en un IdP que bloquea el
// login de todos: sin plazo, cada peticion se queda con una goroutine y un
// socket hasta que el sistema operativo se canse.
const TiempoLimitePorDefecto = 5 * time.Second

// Descubrimiento es el subconjunto del documento de configuracion de OpenID
// Connect que se usa. Lo demas se ignora a proposito: cuanto menos se lea de un
// tercero, menos hay que validar.
type Descubrimiento struct {
	Emisor            string   `json:"issuer"`
	Autorizacion      string   `json:"authorization_endpoint"`
	Token             string   `json:"token_endpoint"`
	JWKS              string   `json:"jwks_uri"`
	FinSesion         string   `json:"end_session_endpoint"`
	AlgoritmosFirma   []string `json:"id_token_signing_alg_values_supported"`
	MetodosCodigoPKCE []string `json:"code_challenge_methods_supported"`
}

// ErrDescubrimiento es el error de un documento de descubrimiento que no sirve.
var ErrDescubrimiento = errors.New("descubrimiento de OIDC invalido")

// Descubrir lee el documento de configuracion del emisor y lo valida.
//
// Que se valida y por que:
//
//   - El `issuer` del documento tiene que ser EXACTAMENTE el emisor esperado.
//     Este es el paso que impide el IdP suplantado: sin el, quien controle el
//     DNS o el documento puede apuntar los endpoints a su servidor y seguir
//     diciendo que es el emisor de siempre.
//   - Los tres endpoints tienen que ser URL absolutas y con TLS. Uno en http
//     es el codigo de autorizacion viajando en claro.
//   - Los endpoints tienen que colgar del mismo host que el emisor. No lo pide
//     la especificacion, y es una restriccion nuestra a proposito: un documento
//     manipulado que apunte el token endpoint a otro dominio se para aqui. Si
//     algun IdP legitimo lo necesita, se cambia con su caso delante.
func Descubrir(ctx context.Context, cliente *http.Client, emisor string) (Descubrimiento, error) {
	var d Descubrimiento
	base, err := url.Parse(emisor)
	if err != nil || base.Host == "" {
		return d, fmt.Errorf("%w: el emisor %q no es una URL", ErrDescubrimiento, emisor)
	}
	// La regla de OpenID Connect Discovery: el emisor mas la ruta bien
	// conocida, sin colapsar la barra, porque el emisor puede llevar ruta.
	destino := strings.TrimSuffix(emisor, "/") + "/.well-known/openid-configuration"

	cuerpo, err := traer(ctx, cliente, destino)
	if err != nil {
		return d, fmt.Errorf("%w: no se pudo leer %s: %v. Comprueba que el emisor esta "+
			"bien escrito y que esta maquina llega al IdP (proxy, cortafuegos, DNS)",
			ErrDescubrimiento, destino, err)
	}
	if err := json.Unmarshal(cuerpo, &d); err != nil {
		return d, fmt.Errorf("%w: %s no devolvio JSON de configuracion (%v). "+
			"Si el emisor es correcto, esto suele ser un portal cautivo o un proxy "+
			"devolviendo su propia pagina", ErrDescubrimiento, destino, err)
	}
	if d.Emisor != emisor {
		return d, fmt.Errorf("%w: el documento dice que su emisor es %q y se esperaba %q. "+
			"No es un detalle de formato: si no coinciden byte a byte, o el emisor "+
			"configurado esta mal escrito (barra final incluida) o el documento no es "+
			"del IdP que crees", ErrDescubrimiento, d.Emisor, emisor)
	}
	campos := []struct{ nombre, valor string }{
		{"authorization_endpoint", d.Autorizacion},
		{"token_endpoint", d.Token},
		{"jwks_uri", d.JWKS},
	}
	for _, c := range campos {
		if strings.TrimSpace(c.valor) == "" {
			return d, fmt.Errorf("%w: el documento no trae %s. Un IdP de OpenID Connect "+
				"lo publica siempre; si falta, es que ese endpoint no es OIDC",
				ErrDescubrimiento, c.nombre)
		}
		u, err := url.Parse(c.valor)
		if err != nil || !u.IsAbs() || u.Host == "" {
			return d, fmt.Errorf("%w: %s vale %q, que no es una URL absoluta",
				ErrDescubrimiento, c.nombre, c.valor)
		}
		if !esquemaSeguro(u) {
			return d, fmt.Errorf("%w: %s vale %q y no usa https", ErrDescubrimiento, c.nombre, c.valor)
		}
		if !mismoHost(u, base) {
			return d, fmt.Errorf("%w: %s apunta a %q, que no es el host del emisor (%q). "+
				"Un documento de descubrimiento que manda los endpoints a otro dominio es "+
				"exactamente como se secuestra un flujo de OIDC", ErrDescubrimiento,
				c.nombre, u.Host, base.Host)
		}
	}
	return d, nil
}

func mismoHost(a, b *url.URL) bool {
	return strings.EqualFold(a.Host, b.Host)
}

// traer hace un GET acotado en tiempo y en bytes.
//
// El limite de bytes no se aplica solo al leer: si el cuerpo alcanza el tope se
// devuelve error en vez de parsear lo que cupo. Parsear un JSON truncado puede
// dar un objeto valido con la mitad de los campos, y esa es la peor de las tres
// salidas posibles.
func traer(ctx context.Context, cliente *http.Client, destino string) ([]byte, error) {
	if cliente == nil {
		cliente = ClientePorDefecto()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, destino, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := cliente.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("el IdP respondio %s", resp.Status)
	}
	cuerpo, err := io.ReadAll(io.LimitReader(resp.Body, LimiteCuerpo+1))
	if err != nil {
		return nil, err
	}
	if int64(len(cuerpo)) > LimiteCuerpo {
		return nil, fmt.Errorf("la respuesta pasa de %d bytes y se corta sin parsear", LimiteCuerpo)
	}
	return cuerpo, nil
}

// ClientePorDefecto es el cliente HTTP con el que se habla con el IdP si no se
// inyecta otro: con plazo, sin seguir redirecciones a ciegas y sin cookies.
//
// Las redirecciones se limitan a dos porque una cadena larga es una forma
// barata de hacernos gastar peticiones, y porque un redirect a un host distinto
// ya lo cierra la comprobacion de host del descubrimiento.
func ClientePorDefecto() *http.Client {
	return &http.Client{
		Timeout: TiempoLimitePorDefecto,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 2 {
				return errors.New("demasiadas redirecciones del IdP")
			}
			return nil
		},
	}
}
