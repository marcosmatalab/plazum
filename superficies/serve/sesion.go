package serve

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"sync"
	"time"

	"dutiq/puertos"
)

// Sesion es la implementacion en memoria de puertos.Sesion.
//
// En memoria a proposito en la etapa 2: el adaptador de almacen todavia no
// existe, y una sesion en memoria tiene una propiedad que conviene entender
// antes de sustituirla, porque es una decision y no un accidente: reiniciar
// dutiq echa a todo el mundo. Para un producto que se instala una vez y se
// actualiza con `dutiq update`, eso es aceptable y ademas es la vuelta atras
// mas barata que existe ante una sospecha de sesion robada.
//
// Tres cosas que no son evidentes al leer el codigo:
//
//  1. La clave del mapa NO es el identificador de sesion, es su SHA-256. El
//     identificador solo existe en la cookie del navegador. Asi un volcado de
//     memoria, o el dia que esto se persista, no entrega sesiones utilizables:
//     entrega hashes, igual que una tabla de contrasenas.
//  2. Los tokens CSRF se guardan tambien hasheados y se comparan en tiempo
//     constante contra TODOS los de la sesion, sin salir antes de tiempo.
//  3. Hay tope de sesiones vivas y de tokens por sesion. Sin tope, un cliente
//     que pide formularios en bucle hace crecer el proceso hasta que el
//     planificador de obligaciones se queda sin memoria, que en este producto
//     es un incumplimiento, no una caida.
type Sesion struct {
	secretos puertos.Secretos
	ahora    func() time.Time

	maxSesiones  int
	maxTokens    int
	maxDuracion  time.Duration
	bytesToken   int
	bytesSesionN int

	mu       sync.Mutex
	sesiones map[[32]byte]*entradaSesion
}

type entradaSesion struct {
	sujeto string
	hasta  time.Time
	// efimera marca las sesiones previas a la autenticacion, las que solo
	// existen para llevar el token CSRF del formulario de entrada.
	efimera bool
	// tokens son los SHA-256 de los tokens CSRF vivos de ESTA sesion, del mas
	// antiguo al mas reciente. Se recortan por el principio al llegar al tope.
	tokens [][32]byte
}

// OpcionesSesion configura el almacen de sesiones. El cero de cada campo es un
// valor seguro, asi que OpcionesSesion{} basta para empezar.
type OpcionesSesion struct {
	// Secretos es la fuente de aleatoriedad. Obligatoria: sin ella no se
	// pueden emitir identificadores, y el fallback silencioso a un generador
	// cualquiera es justo lo que no queremos.
	Secretos puertos.Secretos
	// Reloj se inyecta para poder probar la caducidad sin dormir. Nil es
	// time.Now.
	Reloj func() time.Time
	// MaxSesiones vivas a la vez. Cero es el valor por defecto (10000).
	MaxSesiones int
	// MaxTokensPorSesion vivos a la vez. Cero es el valor por defecto (32).
	MaxTokensPorSesion int
	// MaxDuracion es el techo de lo que puede durar una sesion, pida lo que
	// pida el llamante. Cero es el valor por defecto (24 h).
	MaxDuracion time.Duration
}

// Valores por defecto del almacen de sesiones.
const (
	sesionesPorDefecto  = 10000
	tokensPorDefecto    = 32
	duracionMaximaPorDf = 24 * time.Hour
	bytesDeSesion       = 32 // 256 bits de identificador
	bytesDeTokenCSRF    = 32
)

// Errores del almacen de sesiones. Se exponen para que el middleware pueda
// distinguir "no hay sesion" de "no hay aleatoriedad", que se responden
// distinto: lo primero es 403 y lo segundo es 500.
var (
	// ErrSesionNoValida cubre a la vez "no existe", "caduco" y "se cerro". El
	// que pregunta no aprende nada de la respuesta, que es lo que se quiere.
	ErrSesionNoValida = errors.New("sesion no valida: no existe, ha caducado o se cerro")
	// ErrSinToken es la peticion que llega sin token CSRF ninguno.
	ErrSinToken = errors.New("peticion sin token CSRF")
	// ErrTokenAjeno es el token que no pertenece a esta sesion. Es el ataque
	// del que protege el CSRF, y por eso tiene nombre propio.
	ErrTokenAjeno = errors.New("token CSRF que no pertenece a esta sesion")
)

// NuevaSesion construye el almacen. Falla si no se le da fuente de secretos:
// arrancar sin ella daria sesiones adivinables y es preferible no arrancar.
func NuevaSesion(o OpcionesSesion) (*Sesion, error) {
	if o.Secretos == nil {
		return nil, errors.New(
			"NuevaSesion sin OpcionesSesion.Secretos: los identificadores de sesion " +
				"saldrian de un generador cualquiera. Arreglo: pasa secretos.Nuevo()")
	}
	s := &Sesion{
		secretos:     o.Secretos,
		ahora:        o.Reloj,
		maxSesiones:  o.MaxSesiones,
		maxTokens:    o.MaxTokensPorSesion,
		maxDuracion:  o.MaxDuracion,
		bytesToken:   bytesDeTokenCSRF,
		bytesSesionN: bytesDeSesion,
		sesiones:     map[[32]byte]*entradaSesion{},
	}
	if s.ahora == nil {
		s.ahora = time.Now
	}
	if s.maxSesiones <= 0 {
		s.maxSesiones = sesionesPorDefecto
	}
	if s.maxTokens <= 0 {
		s.maxTokens = tokensPorDefecto
	}
	if s.maxDuracion <= 0 {
		s.maxDuracion = duracionMaximaPorDf
	}
	return s, nil
}

var _ puertos.Sesion = (*Sesion)(nil)

// Abrir crea una sesion nueva para sujeto. El identificador se emite aqui y
// nunca se acepta del cliente: eso es lo que cierra la fijacion de sesion,
// porque un identificador que el atacante planto antes de la autenticacion no
// llega a existir en el almacen.
func (s *Sesion) Abrir(ctx context.Context, sujeto string, duracion time.Duration) (string, error) {
	return s.abrir(ctx, sujeto, duracion, false)
}

// AbrirEfimera abre una sesion PREVIA a la autenticacion: la que solo existe
// para que el formulario de entrada pueda llevar token CSRF.
//
// No forma parte del puerto y existe por un motivo concreto. Exigir CSRF en
// todo POST, incluido el de entrar, obliga a tener sesion antes de entrar. Si
// esas sesiones fueran indistinguibles de las de verdad, cualquiera que pida el
// formulario en bucle llenaria el almacen y dejaria a la organizacion sin poder
// autenticarse, que es un ataque barato y aburrido. Marcandolas, cuando la
// tabla se llena se tira la efimera mas antigua en vez de fallar: el precio de
// esa expulsion es que alguien tenga que recargar el formulario, y el de fallar
// seria que nadie pueda entrar.
func (s *Sesion) AbrirEfimera(ctx context.Context, sujeto string, duracion time.Duration) (string, error) {
	return s.abrir(ctx, sujeto, duracion, true)
}

func (s *Sesion) abrir(ctx context.Context, sujeto string, duracion time.Duration, efimera bool) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if duracion <= 0 {
		return "", fmt.Errorf(
			"duracion de sesion %v: una sesion sin duracion positiva nace caducada o "+
				"no caduca nunca, y las dos cosas son fallos. Arreglo: pide una duracion "+
				"entre un minuto y %v", duracion, s.maxDuracion)
	}
	if duracion > s.maxDuracion {
		return "", fmt.Errorf(
			"duracion de sesion %v por encima del techo %v: una sesion muy larga es una "+
				"credencial robada que sigue valiendo semanas. Arreglo: baja la duracion "+
				"o sube OpcionesSesion.MaxDuracion a sabiendas", duracion, s.maxDuracion)
	}

	id, err := s.secretos.Token(s.bytesSesionN)
	if err != nil {
		return "", fmt.Errorf("no se puede emitir identificador de sesion: %w", err)
	}

	ahora := s.ahora()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgarCaducadas(ahora)

	// Cuando el almacen esta lleno, ANTES de negarse se tira la sesion efimera
	// mas antigua. Eso es lo que impide que una avalancha de visitas al
	// formulario de entrada deje sin poder autenticarse a la organizacion: lo
	// que se pierde es un formulario a medio rellenar, que se arregla
	// recargando, y lo que se salva es entrar.
	//
	// Aqui habia ademas un cupo aparte para las efimeras, la mitad de la tabla.
	// Lo quito el barrido de mutacion de este frente: quitando el cupo, ni un
	// solo test se ponia rojo, y al mirar por que resulto que era redundante.
	// Con la expulsion de abajo, abrir una sesion autenticada funciona igual
	// con la tabla llena de efimeras, asi que el cupo solo anadia una rama que
	// nada cubria. Una defensa que no se puede poner roja no es una defensa,
	// es codigo.
	if len(s.sesiones) >= s.maxSesiones && !s.expulsarEfimeraMasVieja() {
		return "", fmt.Errorf(
			"hay %d sesiones vivas y el tope es %d. No se abren mas para no agotar la "+
				"memoria del proceso. Arreglo: si es trafico legitimo sube "+
				"OpcionesSesion.MaxSesiones; si no lo es, revisa el rate limit de "+
				"autenticacion", len(s.sesiones), s.maxSesiones)
	}
	s.sesiones[sha256.Sum256([]byte(id))] = &entradaSesion{
		sujeto:  sujeto,
		hasta:   ahora.Add(duracion),
		efimera: efimera,
	}
	return id, nil
}

// expulsarEfimeraMasVieja tira la efimera que antes caduca, que con duracion
// fija es la mas antigua. Devuelve si expulso alguna. Con el lock.
func (s *Sesion) expulsarEfimeraMasVieja() bool {
	var elegida [32]byte
	var cuando time.Time
	hay := false
	for k, e := range s.sesiones {
		if !e.efimera {
			continue
		}
		if !hay || e.hasta.Before(cuando) {
			elegida, cuando, hay = k, e.hasta, true
		}
	}
	if hay {
		delete(s.sesiones, elegida)
	}
	return hay
}

// Leer devuelve el sujeto de una sesion viva.
func (s *Sesion) Leer(ctx context.Context, id string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if id == "" {
		return "", ErrSesionNoValida
	}
	clave := sha256.Sum256([]byte(id))
	ahora := s.ahora()

	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.sesiones[clave]
	if !ok {
		return "", ErrSesionNoValida
	}
	if !ahora.Before(e.hasta) {
		// Caducada: se borra al descubrirla, no se deja de recuerdo.
		delete(s.sesiones, clave)
		return "", ErrSesionNoValida
	}
	return e.sujeto, nil
}

// Cerrar invalida la sesion y todos sus tokens CSRF. Idempotente.
func (s *Sesion) Cerrar(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if id == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sesiones, sha256.Sum256([]byte(id)))
	return nil
}

// TokenCSRF emite un token atado a ESTA sesion.
//
// Los tokens NO son de un solo uso, y es una decision. Con token de un solo uso
// se rompen las dos pestanas abiertas, el boton de atras y cualquier peticion
// de htmx que se solape con otra, y lo que se gana es proteccion contra un
// atacante que ya se ha leido el token, o sea contra alguien que ya tiene
// lectura de la pagina y por tanto no necesita el CSRF para nada. Lo que si
// caduca el token es que caduque o se cierre su sesion.
func (s *Sesion) TokenCSRF(ctx context.Context, id string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	// Se emite antes de tomar el lock: crypto/rand puede bloquear y no
	// conviene tener el almacen entero parado mientras.
	tok, err := s.secretos.Token(s.bytesToken)
	if err != nil {
		return "", fmt.Errorf("no se puede emitir token CSRF: %w", err)
	}

	clave := sha256.Sum256([]byte(id))
	ahora := s.ahora()

	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.sesiones[clave]
	if !ok || !ahora.Before(e.hasta) {
		return "", ErrSesionNoValida
	}
	e.tokens = append(e.tokens, sha256.Sum256([]byte(tok)))
	if len(e.tokens) > s.maxTokens {
		// Se tira el mas antiguo. Consecuencia visible y aceptada: una pestana
		// abierta desde hace mucho con un formulario viejo puede encontrarse el
		// token caducado y tendra que recargar.
		e.tokens = e.tokens[len(e.tokens)-s.maxTokens:]
	}
	return tok, nil
}

// ComprobarCSRF valida el token contra la sesion.
func (s *Sesion) ComprobarCSRF(ctx context.Context, id, token string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if token == "" {
		return ErrSinToken
	}
	clave := sha256.Sum256([]byte(id))
	buscado := sha256.Sum256([]byte(token))
	ahora := s.ahora()

	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.sesiones[clave]
	if !ok || !ahora.Before(e.hasta) {
		return ErrSesionNoValida
	}
	// Se recorren TODOS sin salir antes: salir en el primer acierto filtraria
	// por tiempo en que posicion esta el token.
	vale := 0
	for i := range e.tokens {
		vale |= subtle.ConstantTimeCompare(e.tokens[i][:], buscado[:])
	}
	if vale != 1 {
		return ErrTokenAjeno
	}
	return nil
}

// Vivas devuelve cuantas sesiones hay abiertas. Existe para el diagnostico y
// para los tests; no forma parte del puerto.
func (s *Sesion) Vivas() int {
	ahora := s.ahora()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purgarCaducadas(ahora)
	return len(s.sesiones)
}

// purgarCaducadas borra lo que ya no vale. Se llama con el lock tomado.
//
// Va aqui y no en una goroutine de fondo a proposito: un recolector propio es
// una goroutine mas que parar bien en el apagado, y el coste de recorrer unos
// miles de entradas al abrir sesion es despreciable frente a la peticion HTTP
// que lo provoca.
func (s *Sesion) purgarCaducadas(ahora time.Time) {
	for k, e := range s.sesiones {
		if !ahora.Before(e.hasta) {
			delete(s.sesiones, k)
		}
	}
}
