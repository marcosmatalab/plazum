// Package canal entrega avisos de escalado por correo y por Teams.
//
// LO QUE ESTE PAQUETE NO PUEDE HACER, POR CONSTRUCCION, y es la decision que
// ordena todo lo demas: NO IMPORTA NADA DE `nucleo/`. Ni el incidente, ni el
// expediente, ni el corpus, ni siquiera el escalado que decide a quien avisar.
// Recibe cuatro cadenas — canal, destinatario, asunto y cuerpo — y las lleva.
//
// No es purismo de capas. Un adaptador de canal que PUEDE leer el estado de
// cumplimiento acaba mandandolo, igual que un adaptador de telemetria que puede
// leerlo acaba mandandolo (ver adaptadores/latido). La diferencia entre "no lo
// mandamos" y "no podemos mandarlo" es la que un comprador puede comprobar sin
// leerse el codigo, y hay un test que lee los imports de este paquete y se pone
// rojo si aparece cualquier `nucleo/`.
//
// LA LISTA BLANCA ES DE ANFITRIONES, Y SU VALOR CERO NO ENVIA NADA. Un
// `Config{}` sin anfitriones permitidos no manda a ninguna parte, y eso es a
// proposito (invariante 8): el valor cero de una frontera de confianza tiene
// que ser el restrictivo. Si el vacio significara "todos", olvidarse del campo
// abriria el canal a cualquier destino, que es exactamente la forma en que
// estas cosas salen mal.
//
// LOS SECRETOS NO SALEN NI EN LOS ERRORES. La contrasena de SMTP y la URL del
// webhook de Teams son credenciales: la primera es obvia y la segunda no lo
// parece, pero un webhook de Teams es una URL que cualquiera que la tenga puede
// usar para escribir en ese canal. Ninguna de las dos aparece en un error, en
// un log ni en un mensaje. Hay un test que planta un centinela en las dos y
// recorre todos los caminos de fallo.
package canal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	ErrCanalDesconocido   = errors.New("canal no configurado")
	ErrDestinoNoPermitido = errors.New("anfitrion fuera de la lista de permitidos")
	ErrSinPermitidos      = errors.New("no hay ningun anfitrion permitido")
	ErrSinDestinatario    = errors.New("envio sin destinatario")
	ErrEntregaFallida     = errors.New("la entrega fallo")
)

// Transporte es un canal concreto. La interfaz es minima a proposito: si un
// transporte necesitara saber de que obligacion se trata, este paquete tendria
// que conocerla, y entonces podria mandarla.
type Transporte interface {
	// Nombre es como se le llama en la configuracion: "email", "teams".
	Nombre() string
	// Anfitrion es a donde escribe, para poder contrastarlo con la lista
	// blanca ANTES de abrir nada.
	Anfitrion() string
	// Entregar lleva el mensaje. El error que devuelva NO puede llevar
	// credenciales dentro.
	Entregar(destinatario, asunto, cuerpo string) error
}

// Config es lo que el operador declara.
type Config struct {
	// Permitidos son los anfitriones a los que este binario puede escribir.
	// VACIO SIGNIFICA NINGUNO.
	Permitidos []string
	// Reloj da el instante. Se inyecta para que los tests no dependan del
	// reloj de la maquina; vacio es time.Now.
	Reloj func() time.Time
}

// Mensajero implementa puertos.Notificacion sobre uno o mas transportes.
type Mensajero struct {
	cfg    Config
	tr     map[string]Transporte
	mu     sync.Mutex
	ultimo map[string]time.Time
}

// Nuevo construye el mensajero. Un transporte cuyo anfitrion no este permitido
// SE RECHAZA AL CONSTRUIR y no al enviar: descubrir que el canal no vale el dia
// que hay que avisar es descubrirlo tarde.
func Nuevo(cfg Config, transportes ...Transporte) (*Mensajero, error) {
	if cfg.Reloj == nil {
		cfg.Reloj = time.Now
	}
	m := &Mensajero{cfg: cfg, tr: map[string]Transporte{}, ultimo: map[string]time.Time{}}
	for _, t := range transportes {
		if err := m.permitido(t.Anfitrion()); err != nil {
			return nil, fmt.Errorf("el canal %q escribe a %q y %w. Arreglo: anade ese "+
				"anfitrion a la lista de permitidos, o quita el canal",
				t.Nombre(), t.Anfitrion(), err)
		}
		m.tr[t.Nombre()] = t
	}
	return m, nil
}

// permitido contrasta un anfitrion con la lista blanca.
//
// La comparacion es por anfitrion EXACTO, sin comodines y sin sufijos: un
// permitido "ejemplo.com" no autoriza "malo.ejemplo.com" ni "ejemplo.com.malo".
// Los comodines en listas de anfitriones son la forma clasica de que una lista
// blanca deje de serlo sin que nadie lo note.
func (m *Mensajero) permitido(anfitrion string) error {
	if len(m.cfg.Permitidos) == 0 {
		return fmt.Errorf("%w: sin lista de permitidos no se escribe a ninguna parte, que "+
			"es lo que tiene que pasar cuando nadie ha configurado nada", ErrSinPermitidos)
	}
	if anfitrion == "" {
		return fmt.Errorf("%w: el canal no dice a donde escribe", ErrDestinoNoPermitido)
	}
	for _, p := range m.cfg.Permitidos {
		if strings.EqualFold(p, anfitrion) {
			return nil
		}
	}
	return fmt.Errorf("%w: %q no esta entre los %d permitidos", ErrDestinoNoPermitido,
		anfitrion, len(m.cfg.Permitidos))
}

// Enviar entrega por el canal que se le diga.
//
// El error que sale de aqui es el del transporte, y la unica cosa que este
// paquete hace con el es NO enriquecerlo con nada que pueda ser credencial.
func (m *Mensajero) Enviar(canal, destinatario, asunto, cuerpo string) error {
	t, hay := m.tr[canal]
	if !hay {
		return fmt.Errorf("%w: %q. Los configurados son %v", ErrCanalDesconocido, canal,
			m.canales())
	}
	if strings.TrimSpace(destinatario) == "" {
		return fmt.Errorf("%w por %s: un aviso sin destinatario no es un aviso, y mandarlo "+
			"a la nada se leeria como enviado", ErrSinDestinatario, canal)
	}
	// La lista blanca se vuelve a mirar AQUI y no solo al construir: un
	// transporte cuyo anfitrion cambie en caliente (una redireccion, una
	// reconfiguracion) no se cuela por haber pasado una vez.
	if err := m.permitido(t.Anfitrion()); err != nil {
		return err
	}
	if err := t.Entregar(destinatario, asunto, cuerpo); err != nil {
		return fmt.Errorf("%w por %s: %v", ErrEntregaFallida, canal, err)
	}
	m.mu.Lock()
	m.ultimo[canal] = m.cfg.Reloj()
	m.mu.Unlock()
	return nil
}

// UltimoExito dice cuando fue la ultima entrega que salio bien.
//
// SOLO AVANZA CON UN EXITO DE VERDAD. Si avanzara al intentarlo, un canal roto
// se leeria como un canal vivo, que es el fallo que este dato existe para
// detectar: el smoke test de un canal que nadie usa desde hace meses.
func (m *Mensajero) UltimoExito(canal string) (string, error) {
	if _, hay := m.tr[canal]; !hay {
		return "", fmt.Errorf("%w: %q", ErrCanalDesconocido, canal)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	t, hay := m.ultimo[canal]
	if !hay {
		return "", nil // nunca ha entregado nada, y eso no es un error
	}
	return t.Format(time.RFC3339), nil
}

func (m *Mensajero) canales() []string {
	out := make([]string, 0, len(m.tr))
	for k := range m.tr {
		out = append(out, k)
	}
	ordenar(out)
	return out
}

func ordenar(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// ---------------------------------------------------------------------------
// Teams: un webhook
// ---------------------------------------------------------------------------

// Teams escribe en un canal de Teams por su webhook entrante.
//
// LA URL DEL WEBHOOK ES UNA CREDENCIAL, aunque no lo parezca: quien la tenga
// puede escribir en ese canal. Por eso vive en un campo privado, no sale en
// Anfitrion() (que devuelve solo el anfitrion) y no aparece en ningun error.
type Teams struct {
	webhook   *url.URL
	cliente   *http.Client
	anfitrion string
}

// NuevoTeams valida la URL al construir.
func NuevoTeams(webhook string, c *http.Client) (*Teams, error) {
	u, err := url.Parse(webhook)
	if err != nil || u.Host == "" {
		// El error de url.Parse puede llevar la URL entera dentro, asi que NO
		// se propaga: se dice que es ilegible y ya.
		return nil, errors.New("el webhook de Teams no es una URL con anfitrion. No se " +
			"reproduce aqui porque un webhook es una credencial")
	}
	if u.Scheme != "https" {
		return nil, errors.New("el webhook de Teams tiene que ser https: por http, la " +
			"credencial viaja en claro")
	}
	if c == nil {
		c = &http.Client{Timeout: 15 * time.Second}
	}
	return &Teams{webhook: u, cliente: c, anfitrion: u.Hostname()}, nil
}

func (t *Teams) Nombre() string    { return "teams" }
func (t *Teams) Anfitrion() string { return t.anfitrion }

// mensajeTeams es LO QUE VIAJA, y son tres campos. La lista blanca del cuerpo
// se comprueba en el test contra esta forma.
type mensajeTeams struct {
	Titulo string `json:"title"`
	Texto  string `json:"text"`
	Para   string `json:"summary"`
}

func (t *Teams) Entregar(destinatario, asunto, cuerpo string) error {
	b, err := json.Marshal(mensajeTeams{Titulo: asunto, Texto: cuerpo, Para: destinatario})
	if err != nil {
		return errors.New("el mensaje no se pudo serializar")
	}
	req, err := http.NewRequest(http.MethodPost, t.webhook.String(), bytes.NewReader(b))
	if err != nil {
		return errors.New("la peticion no se pudo construir")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.cliente.Do(req)
	if err != nil {
		// EL ERROR DE http.Client LLEVA LA URL ENTERA DENTRO, incluido el
		// token del webhook. Se sustituye por el anfitrion, que es lo unico
		// que hace falta para diagnosticar.
		return fmt.Errorf("no se pudo hablar con %s", t.anfitrion)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s contesto %d", t.anfitrion, resp.StatusCode)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Email: SMTP
// ---------------------------------------------------------------------------

// EnvioSMTP es la funcion que habla SMTP de verdad. Se inyecta para que los
// tests no necesiten un servidor y para que la credencial no tenga que viajar
// por ningun sitio raro.
type EnvioSMTP func(direccion string, de string, a []string, mensaje []byte) error

// Email manda por SMTP.
type Email struct {
	anfitrion string
	puerto    string
	de        string
	envio     EnvioSMTP
}

func NuevoEmail(direccion, de string, envio EnvioSMTP) (*Email, error) {
	anfitrion, puerto, err := net.SplitHostPort(direccion)
	if err != nil || anfitrion == "" {
		return nil, errors.New("la direccion del servidor SMTP se escribe anfitrion:puerto")
	}
	if !strings.Contains(de, "@") {
		return nil, errors.New("el remitente no parece una direccion de correo")
	}
	if envio == nil {
		return nil, errors.New("sin funcion de envio SMTP no hay canal de correo")
	}
	return &Email{anfitrion: anfitrion, puerto: puerto, de: de, envio: envio}, nil
}

func (e *Email) Nombre() string    { return "email" }
func (e *Email) Anfitrion() string { return e.anfitrion }

func (e *Email) Entregar(destinatario, asunto, cuerpo string) error {
	// LA CABECERA SE CONSTRUYE AQUI Y NO CONCATENANDO LO QUE LLEGA. Un asunto
	// con un salto de linea dentro inyectaria cabeceras SMTP: es la inyeccion
	// clasica de este protocolo, y se corta quitando los saltos.
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n"+
		"MIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		sinSaltos(e.de), sinSaltos(destinatario), sinSaltos(asunto), cuerpo)
	if err := e.envio(net.JoinHostPort(e.anfitrion, e.puerto), e.de,
		[]string{destinatario}, []byte(msg)); err != nil {
		// El error de net/smtp puede llevar la credencial de la autenticacion
		// dentro. Se reduce al anfitrion.
		return fmt.Errorf("no se pudo entregar en %s", e.anfitrion)
	}
	return nil
}

// sinSaltos quita CR y LF de un valor que va a una cabecera.
func sinSaltos(s string) string {
	return strings.NewReplacer("\r", " ", "\n", " ").Replace(s)
}
