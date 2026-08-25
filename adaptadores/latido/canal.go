package latido

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Canal es por donde sale el pulso.
//
// Es una interfaz por la misma razon que lo es el canal del actualizador: lo
// que hay que poder probar es el opt-in, el contenido exacto de lo que sale y
// el efecto sobre las marcas, y nada de eso necesita una red. Con la interfaz,
// la suite entera corre sin abrir un socket y sigue probando el codigo que de
// verdad se ejecuta.
//
// Entregar recibe el destino en cada llamada y no lo guarda: el destino vive en
// el estado del operador, que es quien lo elige.
type Canal interface {
	// Entregar manda el cuerpo al destino. Devuelve error si no llego.
	Entregar(ctx context.Context, destino string, cuerpo []byte) error
}

// ErrCanal: el pulso no llego. Centinela, para errors.Is.
var ErrCanal = errors.New("el pulso no ha llegado al destino")

// AgenteDeUsuario es lo unico que el pulso dice de si mismo ademas de sus dos
// campos. No lleva version del producto: la version es un dato mas, y "si
// dudas, no lo mandes".
const AgenteDeUsuario = "plazum-latido"

// EsperaPorDefecto acota lo que un pulso puede tardar. Un pulso que se queda
// colgado bloquearia el ciclo del planificador, o sea que el canal hacia
// nosotros pararia su vigilancia: exactamente al reves de lo que esta pieza
// existe para hacer.
const EsperaPorDefecto = 10 * time.Second

// MaxRespuesta acota lo que se lee de la respuesta. El receptor no manda nada
// que nos importe (la respuesta se descarta), asi que esto es solo para no
// dejar que un destino hostil nos haga leer un gigabyte.
const MaxRespuesta = 4 << 10

// CanalHTTP entrega el pulso por HTTPS.
//
// Lo que NO hace, que es donde estan las decisiones:
//
//	no sigue redirecciones   una redireccion mueve el pulso a otra maquina, o
//	                         le anade una parte de consulta, sin que el
//	                         operador se entere. El destino es el que el
//	                         escribio o ninguno.
//	no guarda cookies        un receptor que pone una cookie convierte pulsos
//	                         anonimos en una sesion, que es justo lo que el
//	                         identificador aleatorio evita.
//	no comprime              para que las cabeceras que salen sean exactamente
//	                         las tres que la lista blanca declara.
type CanalHTTP struct {
	// Cliente es opcional. Sin el se construye uno con las tres reglas de
	// arriba puestas.
	Cliente *http.Client
	// Espera acota la peticion. 0 usa EsperaPorDefecto.
	Espera time.Duration
}

var _ Canal = CanalHTTP{}

// clienteSeguro es el cliente por defecto: sin redirecciones, sin cookies, sin
// compresion y con espera acotada.
func (c CanalHTTP) clienteSeguro() *http.Client {
	if c.Cliente != nil {
		return c.Cliente
	}
	espera := c.Espera
	if espera <= 0 {
		espera = EsperaPorDefecto
	}
	return &http.Client{
		Timeout: espera,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return fmt.Errorf("%w: el destino redirige a %s. Una redireccion mueve el "+
				"pulso a otra maquina sin que tu lo sepas. Arreglo: pon en --destino la "+
				"direccion final", ErrCanal, req.URL.Redacted())
		},
		Transport: &http.Transport{DisableCompression: true},
	}
}

// Entregar manda el cuerpo con POST.
func (c CanalHTTP) Entregar(ctx context.Context, destino string, cuerpo []byte) error {
	if err := ComprobarDestino(destino); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, destino, bytes.NewReader(cuerpo))
	if err != nil {
		return fmt.Errorf("%w: no puedo construir la peticion a %s: %w", ErrCanal, destino, err)
	}
	// Estas tres y ninguna mas. La lista blanca de frontera_test.go lo
	// comprueba sobre la peticion que de verdad llega al otro lado.
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", AgenteDeUsuario)
	req.ContentLength = int64(len(cuerpo))

	resp, err := c.clienteSeguro().Do(req)
	if err != nil {
		return fmt.Errorf("%w: %s no contesta: %w. Arreglo: comprueba la salida a internet "+
			"y el destino. Esto NO afecta a tus plazos: el aviso de las 24 horas se "+
			"calcula con tu reloj y no depende de este canal", ErrCanal, destino, err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, MaxRespuesta))
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("%w: %s ha contestado %d. Arreglo: si es tu propio receptor, "+
			"comprueba que acepta POST de JSON en esa ruta", ErrCanal, destino, resp.StatusCode)
	}
	return nil
}

// CanalMemoria guarda lo que se le entrega, sin salir a ningun sitio.
//
// Es el canal de las pruebas y tambien el que hace posible probar el opt-in de
// verdad: la comprobacion de que con el latido apagado NO SALE NADA se hace
// mirando que aqui no ha llegado ni una entrega, que es una afirmacion sobre el
// producto y no sobre una maqueta de red.
type CanalMemoria struct {
	mu sync.Mutex
	// Fallo, si no es nil, es lo que devuelve Entregar. Sirve para el caso
	// de canal roto sin romper nada de verdad.
	Fallo error
	// Entregas son los cuerpos entregados, en orden.
	Entregas [][]byte
	// Destinos son los destinos de cada entrega, en el mismo orden.
	Destinos []string
}

var _ Canal = (*CanalMemoria)(nil)

// Entregar apunta la entrega y devuelve Fallo.
func (c *CanalMemoria) Entregar(_ context.Context, destino string, cuerpo []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Fallo != nil {
		// Un canal que falla NO apunta la entrega: lo que no llego no
		// llego, y contarlo como entrega taparia justo el caso que se
		// quiere probar.
		return c.Fallo
	}
	c.Entregas = append(c.Entregas, append([]byte(nil), cuerpo...))
	c.Destinos = append(c.Destinos, destino)
	return nil
}

// N es cuantas entregas se han hecho.
func (c *CanalMemoria) N() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.Entregas)
}

// Ultima es el ultimo cuerpo entregado.
func (c *CanalMemoria) Ultima() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.Entregas) == 0 {
		return nil
	}
	return c.Entregas[len(c.Entregas)-1]
}
