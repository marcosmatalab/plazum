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

	"plazum/puertos"
)

// DuracionSesionPorDefecto es lo que dura la sesion que se abre tras entrar.
//
// Ocho horas es una jornada. No se hereda del `exp` del ID token, que suele ser
// de una hora: el ID token es la prueba de que el usuario se autentico, no la
// duracion de su trabajo aqui.
const DuracionSesionPorDefecto = 8 * time.Hour

// ErrAdmision es el rechazo de alguien que el IdP autentico correctamente pero
// que no puede entrar aqui. El caso tipico es el usuario desactivado en SCIM.
//
// Se distingue de ErrToken a proposito: el token era bueno. Mezclarlos haria que
// el operador buscara el problema en la configuracion del IdP, que esta bien.
var ErrAdmision = errors.New("autenticado en el IdP, pero no admitido en plazum")

// Autenticador es el flujo completo: iniciar, volver, verificar y abrir sesion.
//
// Lo que entrega son dos metodos, no rutas. El que construya el servidor decide
// en que URL cuelgan; este paquete no conoce ningun router, a proposito.
type Autenticador struct {
	cfg      Configuracion
	prov     Descubrimiento
	ver      *Verificador
	pend     *Pendientes
	ses      puertos.Sesion
	sec      puertos.Secretos
	cliente  *http.Client
	duracion time.Duration

	// Admision decide si una identidad ya verificada puede entrar. Nil
	// significa que entra cualquiera que el IdP autentique.
	//
	// Es un campo y no una dependencia porque este paquete NO tiene que
	// conocer el directorio de SCIM: la regla la pone quien cablea, y asi
	// "el usuario desactivado en el IdP deja de poder entrar" se cumple sin
	// que identidad y aprovisionamiento se importen entre si.
	Admision func(Identidad) error
}

// Opciones son los ajustes del autenticador que tienen valor por defecto util.
type Opciones struct {
	// Cliente HTTP con el que se habla con el IdP. Nil, ClientePorDefecto.
	Cliente *http.Client
	// VidaPeticion es lo que vive un `state` sin usar. Cero,
	// VidaPeticionPorDefecto.
	VidaPeticion time.Duration
	// DuracionSesion es lo que dura la sesion abierta. Cero,
	// DuracionSesionPorDefecto.
	DuracionSesion time.Duration
	// IntervaloRecargaJWKS es el minimo entre dos lecturas del JWKS. Cero,
	// IntervaloRecargaPorDefecto.
	IntervaloRecargaJWKS time.Duration
}

// NuevoAutenticador descubre el IdP, valida lo que se puede validar y deja el
// flujo listo.
//
// Falla al construir y no al primer login: un `client_id` mal pegado tiene que
// romper el arranque de la instancia, con un mensaje que diga donde se corrige,
// no un martes por la tarde delante de un usuario.
func NuevoAutenticador(ctx context.Context, cfg Configuracion, ses puertos.Sesion, sec puertos.Secretos, op Opciones) (*Autenticador, error) {
	if err := cfg.validar(); err != nil {
		return nil, err
	}
	if ses == nil {
		return nil, fmt.Errorf("%w: falta el puerto de Sesion. Sin el, autenticar no abre "+
			"nada y el usuario vuelve a la pantalla de entrar", ErrConfiguracion)
	}
	if sec == nil {
		return nil, fmt.Errorf("%w: falta el puerto de Secretos. El state, el nonce y el "+
			"verificador PKCE son aleatoriedad criptografica: sin fuente, no se generan",
			ErrConfiguracion)
	}
	cliente := op.Cliente
	if cliente == nil {
		cliente = ClientePorDefecto()
	}
	prov, err := Descubrir(ctx, cliente, cfg.Emisor)
	if err != nil {
		return nil, err
	}
	// Si el IdP publica sus metodos de PKCE y S256 no esta, el flujo no puede
	// dar la garantia que promete. Mejor romper aqui que creer que hay PKCE.
	if len(prov.MetodosCodigoPKCE) > 0 && !contiene(prov.MetodosCodigoPKCE, "S256") {
		return nil, fmt.Errorf("%w: el IdP declara los metodos de PKCE %v y no incluye S256. "+
			"plazum no usa `plain`, que manda el verificador en claro por la misma URL y no "+
			"protege de nada", ErrDescubrimiento, prov.MetodosCodigoPKCE)
	}
	claves := NuevasClaves(prov.JWKS, cliente).ConIntervaloRecarga(op.IntervaloRecargaJWKS)
	ver, err := NuevoVerificador(cfg, claves)
	if err != nil {
		return nil, err
	}
	dur := op.DuracionSesion
	if dur <= 0 {
		dur = DuracionSesionPorDefecto
	}
	return &Autenticador{
		cfg:      cfg,
		prov:     prov,
		ver:      ver,
		pend:     NuevasPendientes(op.VidaPeticion),
		ses:      ses,
		sec:      sec,
		cliente:  cliente,
		duracion: dur,
	}, nil
}

// Verificador da acceso al verificador construido, para quien quiera comprobar
// un token sin pasar por el flujo.
func (a *Autenticador) Verificador() *Verificador { return a.ver }

// Descubrimiento devuelve lo que se leyo del IdP. Lo usa el diagnostico para
// ensenar al operador contra que endpoints esta hablando de verdad.
func (a *Autenticador) Descubrimiento() Descubrimiento { return a.prov }

// Iniciar arranca el flujo: genera `state`, `nonce` y verificador PKCE, los
// guarda atados entre si, y devuelve la URL a la que mandar el navegador.
//
// destino es a donde volver dentro de plazum despues de entrar, y se guarda AQUI,
// no viaja por la URL. Un destino que llega en la peticion de retorno convierte
// el login en un redirector abierto con nuestro dominio delante.
func (a *Autenticador) Iniciar(ctx context.Context, ahora time.Time, destino string) (string, error) {
	_ = ctx
	estado, err := a.sec.Token(32)
	if err != nil {
		return "", fmt.Errorf("no hay aleatoriedad para el state: %w", err)
	}
	nonce, err := a.sec.Token(32)
	if err != nil {
		return "", fmt.Errorf("no hay aleatoriedad para el nonce: %w", err)
	}
	verificador, err := a.sec.Token(LongitudVerificadorPKCE)
	if err != nil {
		return "", fmt.Errorf("no hay aleatoriedad para el verificador PKCE: %w", err)
	}
	if estado == nonce || estado == verificador || nonce == verificador {
		return "", errors.New("la fuente de aleatoriedad ha devuelto el mismo valor dos veces: " +
			"no es aleatoria y el flujo no es seguro con ella")
	}
	if err := a.pend.Guardar(estado, peticion{
		nonce:       nonce,
		verificador: verificador,
		creada:      ahora,
		destino:     destinoSeguro(destino),
	}, ahora); err != nil {
		return "", err
	}
	u, err := url.Parse(a.prov.Autorizacion)
	if err != nil {
		return "", fmt.Errorf("%w: el authorization_endpoint no es una URL", ErrDescubrimiento)
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", a.cfg.ClienteID)
	q.Set("redirect_uri", a.cfg.RedirectURI)
	q.Set("scope", strings.Join(a.cfg.ambitos(), " "))
	q.Set("state", estado)
	q.Set("nonce", nonce)
	q.Set("code_challenge", DesafioPKCE(verificador))
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// destinoSeguro deja pasar solo rutas internas.
//
// Es la comprobacion que impide el redirector abierto por la puerta del
// parametro de destino: "//evil.example" y "https://evil.example" son URL
// absolutas aunque empiecen por barra, y un navegador las sigue fuera.
func destinoSeguro(d string) string {
	if d == "" || !strings.HasPrefix(d, "/") || strings.HasPrefix(d, "//") {
		return "/"
	}
	if strings.Contains(d, "\\") || strings.ContainsAny(d, "\r\n") {
		return "/"
	}
	return d
}

// Retorno cierra el flujo: valida el `state`, canjea el codigo con PKCE,
// verifica el ID token con su `nonce`, pasa la admision y abre sesion.
//
// consulta son los parametros de la peticion de retorno. Se leen SOLO `state`,
// `code`, `error` y `error_description`. En particular NO se lee `redirect_uri`
// ni ningun destino: los dos salen de lo que se guardo al iniciar.
func (a *Autenticador) Retorno(ctx context.Context, ahora time.Time, consulta url.Values) (string, Identidad, string, error) {
	var vacia Identidad
	// El IdP contesta con error en la propia redireccion cuando el usuario
	// cancela o cuando la aplicacion no esta bien registrada. Traducirlo aqui
	// evita que el operador vea una pantalla en blanco.
	if e := consulta.Get("error"); e != "" {
		return "", vacia, "", fmt.Errorf("%w: %s", ErrAdmision, explicarErrorIdP(e, consulta.Get("error_description")))
	}
	estado := consulta.Get("state")
	if estado == "" {
		return "", vacia, "", fmt.Errorf("%w: la peticion de retorno no trae `state`. Sin el "+
			"no hay forma de saber que este retorno corresponde a un login que empezo aqui, "+
			"que es justo el CSRF del que protege", ErrEstado)
	}
	pet, err := a.pend.Tomar(estado, ahora)
	if err != nil {
		return "", vacia, "", err
	}
	codigo := consulta.Get("code")
	if codigo == "" {
		return "", vacia, "", fmt.Errorf("%w: la peticion de retorno no trae `code`", ErrEstado)
	}
	idToken, err := a.canjear(ctx, codigo, pet.verificador)
	if err != nil {
		return "", vacia, "", err
	}
	id, err := a.ver.Verificar(ctx, idToken, ahora, Esperado{Nonce: pet.nonce})
	if err != nil {
		return "", vacia, "", err
	}
	if a.Admision != nil {
		if err := a.Admision(id); err != nil {
			return "", vacia, "", fmt.Errorf("%w: %v", ErrAdmision, err)
		}
	}
	// El sujeto de la sesion es el `sub` del IdP, no el correo: el correo
	// cambia y el `sub` no, y una sesion atada al correo se despega del
	// usuario el dia que alguien se casa.
	idSesion, err := a.ses.Abrir(ctx, id.Sujeto, a.duracion)
	if err != nil {
		return "", vacia, "", fmt.Errorf("no se pudo abrir sesion para %q: %w", id.Sujeto, err)
	}
	return idSesion, id, pet.destino, nil
}

// canjear cambia el codigo por el ID token en el token endpoint.
//
// Ningun error de esta funcion incluye el cuerpo de la peticion, y es
// deliberado: ese cuerpo lleva el `client_secret` y el `code_verifier`. Un
// secreto en una traza es un incidente, no una molestia.
func (a *Autenticador) canjear(ctx context.Context, codigo, verificador string) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", codigo)
	// La redirect_uri del canje sale de la configuracion, igual que la del
	// inicio. Es la comprobacion que ata el codigo a esta aplicacion.
	form.Set("redirect_uri", a.cfg.RedirectURI)
	form.Set("client_id", a.cfg.ClienteID)
	form.Set("code_verifier", verificador)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.prov.Token,
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	if a.cfg.ClienteSecreto != "" {
		// client_secret_basic, que es el metodo por defecto de OpenID Connect
		// y el unico que Entra ID y Okta aceptan siempre. El secreto va en la
		// cabecera, nunca en el cuerpo que se pudiera loguear.
		req.SetBasicAuth(url.QueryEscape(a.cfg.ClienteID), url.QueryEscape(a.cfg.ClienteSecreto))
	}
	resp, err := a.cliente.Do(req)
	if err != nil {
		return "", fmt.Errorf("no se pudo canjear el codigo en %s: %w", a.prov.Token, err)
	}
	defer func() { _ = resp.Body.Close() }()
	cuerpo, err := io.ReadAll(io.LimitReader(resp.Body, LimiteCuerpo+1))
	if err != nil {
		return "", fmt.Errorf("no se pudo leer la respuesta del canje: %w", err)
	}
	if int64(len(cuerpo)) > LimiteCuerpo {
		return "", fmt.Errorf("la respuesta del canje pasa de %d bytes y se corta sin parsear",
			LimiteCuerpo)
	}
	var r struct {
		IDToken     string `json:"id_token"`
		TipoToken   string `json:"token_type"`
		Error       string `json:"error"`
		Descripcion string `json:"error_description"`
	}
	// Se parsea aunque el estado no sea 200: el error de OAuth viene en el
	// cuerpo y es lo unico util que hay para explicarselo al operador.
	_ = json.Unmarshal(cuerpo, &r)
	if resp.StatusCode != http.StatusOK {
		if r.Error != "" {
			return "", fmt.Errorf("%w: %s", ErrAdmision, explicarErrorIdP(r.Error, r.Descripcion))
		}
		return "", fmt.Errorf("%w: el token endpoint respondio %s sin decir por que",
			ErrAdmision, resp.Status)
	}
	if r.IDToken == "" {
		return "", fmt.Errorf("%w: el canje salio bien y no vino `id_token`. Casi siempre "+
			"significa que el scope `openid` no esta pedido o que el IdP tiene la aplicacion "+
			"registrada como OAuth 2.0 y no como OpenID Connect", ErrAdmision)
	}
	return r.IDToken, nil
}

// explicarErrorIdP traduce los codigos de error de OAuth a algo que el
// administrador pueda arreglar sin leer una especificacion.
//
// Cada rama nombra el sitio concreto de Entra ID o de Okta donde se toca. Es la
// diferencia entre un martes por la tarde de veinte minutos y uno de tres horas.
func explicarErrorIdP(codigo, descripcion string) string {
	var arreglo string
	switch codigo {
	case "invalid_client":
		arreglo = "el IdP no reconoce las credenciales de esta aplicacion. Revisa el " +
			"cliente_id y el cliente_secreto. En Entra ID el secreto CADUCA: mira " +
			"Certificados y secretos y crea uno nuevo si la fecha ya paso"
	case "invalid_grant":
		arreglo = "el codigo de autorizacion no vale. Suele ser una de tres: ya se canjeo " +
			"(un refresco de la pagina de retorno lo hace), caduco, o la redirect_uri del " +
			"canje no es identica a la del registro"
	case "unauthorized_client":
		arreglo = "la aplicacion existe pero no tiene permitido este flujo. Habilita el " +
			"flujo de codigo de autorizacion en el registro de la aplicacion"
	case "invalid_request":
		arreglo = "el IdP considera mal formada la peticion. Lo mas comun es una " +
			"redirect_uri que no coincide caracter a caracter con la registrada, barra " +
			"final incluida"
	case "access_denied":
		arreglo = "el usuario cancelo, o una politica de acceso condicional lo bloqueo. " +
			"Si nadie cancelo, mira los registros de inicio de sesion del IdP"
	case "invalid_scope":
		arreglo = "alguno de los ambitos pedidos no esta concedido a la aplicacion. " +
			"Concede openid, profile y email en los permisos de la API"
	case "consent_required", "interaction_required":
		arreglo = "falta el consentimiento del administrador para los permisos pedidos. " +
			"Concedelo desde el registro de la aplicacion"
	default:
		arreglo = "consulta los registros de inicio de sesion del IdP con la marca de tiempo " +
			"de este intento"
	}
	if descripcion != "" {
		return fmt.Sprintf("el IdP respondio %q (%s). Arreglo: %s", codigo, recortar(descripcion, 300), arreglo)
	}
	return fmt.Sprintf("el IdP respondio %q. Arreglo: %s", codigo, arreglo)
}

// recortar acota el texto que un tercero nos manda antes de meterlo en un
// mensaje nuestro. Sin esto, el IdP decide cuanto ocupa nuestro log.
func recortar(s string, n int) string {
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}
		return r
	}, s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func contiene(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
