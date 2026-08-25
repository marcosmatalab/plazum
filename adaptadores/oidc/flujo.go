package oidc

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"
	"time"
)

// VidaPeticionPorDefecto es lo que vive un `state` sin usar.
//
// Cinco minutos es el tiempo de teclear un usuario y una contrasena con MFA. No
// mas: cada `state` vivo es una entrada en memoria y una oportunidad de reuso.
const VidaPeticionPorDefecto = 5 * time.Minute

// MaxPeticionesEnVuelo acota cuantos flujos a medias se guardan a la vez.
//
// Sin tope, pulsar "entrar" en bucle sin completar nunca el flujo es una fuga de
// memoria remota y gratuita: cada pulsacion guarda un `state`, un `nonce` y un
// verificador PKCE que nadie va a reclamar. Con tope, lo que se llena se poda
// por lo caducado primero y, si aun asi no cabe, se rechaza el inicio con un
// error claro en vez de comerse la maquina.
const MaxPeticionesEnVuelo = 4096

// ErrEstado es el error del `state`: ausente, desconocido, caducado o ya usado.
// Los cuatro casos comparten error porque el que llama no debe distinguirlos:
// distinguirlos le dice a quien prueba si acerto.
var ErrEstado = errors.New("state invalido")

// peticion es lo que se guarda entre el inicio del flujo y el retorno.
type peticion struct {
	nonce       string
	verificador string
	creada      time.Time
	// Destino es a donde volver despues de entrar. Se guarda AQUI, atado al
	// state, y nunca se lee de la URL de retorno: un destino que viene en la
	// peticion de retorno es un redirector abierto con nuestro dominio.
	destino string
}

// Pendientes guarda los flujos a medias. Un `state` vale UNA vez: al tomarlo se
// borra, pasa lo que pase despues.
//
// Es seguro para uso concurrente.
type Pendientes struct {
	mu   sync.Mutex
	m    map[string]peticion
	vida time.Duration
	max  int
}

// NuevasPendientes crea el almacen. vida cero significa VidaPeticionPorDefecto.
func NuevasPendientes(vida time.Duration) *Pendientes {
	if vida <= 0 {
		vida = VidaPeticionPorDefecto
	}
	return &Pendientes{m: map[string]peticion{}, vida: vida, max: MaxPeticionesEnVuelo}
}

// Guardar registra un flujo iniciado.
func (p *Pendientes) Guardar(estado string, pet peticion, ahora time.Time) error {
	if estado == "" {
		return fmt.Errorf("%w: no se puede guardar un state vacio", ErrEstado)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.m) >= p.max {
		p.podar(ahora)
	}
	if len(p.m) >= p.max {
		return fmt.Errorf("%w: hay %d flujos de autenticacion a medias, que es el maximo. "+
			"Se rechaza iniciar otro en vez de gastar memoria sin limite; si esto pasa en "+
			"un sistema real, alguien esta pulsando entrar en bucle o hay un bot",
			ErrEstado, p.max)
	}
	if _, ya := p.m[estado]; ya {
		return fmt.Errorf("%w: ese state ya estaba en vuelo", ErrEstado)
	}
	p.m[estado] = pet
	return nil
}

// Tomar devuelve la peticion y la borra. Un `state` no se puede usar dos veces:
// eso es lo que convierte el `state` en proteccion de CSRF del flujo y no en un
// campo decorativo.
func (p *Pendientes) Tomar(estado string, ahora time.Time) (peticion, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	pet, ok := p.m[estado]
	if !ok {
		return peticion{}, fmt.Errorf("%w: no hay ninguna peticion de autenticacion con ese "+
			"state. O ya se uso, o caduco (%s), o esta peticion de retorno no la origino este "+
			"navegador", ErrEstado, p.vida)
	}
	// Se borra ANTES de comprobar la caducidad: si se borrase despues, un
	// state caducado se podria reintentar indefinidamente.
	delete(p.m, estado)
	if ahora.Sub(pet.creada) > p.vida {
		return peticion{}, fmt.Errorf("%w: la peticion de autenticacion caduco (vive %s). "+
			"Vuelve a pulsar entrar", ErrEstado, p.vida)
	}
	if ahora.Before(pet.creada) {
		return peticion{}, fmt.Errorf("%w: la peticion se creo en el futuro; el reloj de esta "+
			"maquina ha cambiado durante el flujo", ErrEstado)
	}
	return pet, nil
}

// EnVuelo dice cuantos flujos a medias hay. Sirve para los tests y para el
// diagnostico.
func (p *Pendientes) EnVuelo() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.m)
}

// podar borra lo caducado. Se llama con el candado tomado.
func (p *Pendientes) podar(ahora time.Time) {
	for k, v := range p.m {
		if ahora.Sub(v.creada) > p.vida {
			delete(p.m, k)
		}
	}
}

// DesafioPKCE calcula el `code_challenge` S256 de un verificador.
//
// S256 y solo S256. El metodo `plain` de la especificacion manda el verificador
// en claro por la misma URL que el resto y no protege de nada; existe para
// clientes que no tenian SHA-256 y aqui si lo hay. Se usa TAMBIEN con secreto de
// cliente, aunque la especificacion no lo exija: PKCE protege de la
// interceptacion del codigo, y el secreto no.
func DesafioPKCE(verificador string) string {
	suma := sha256.Sum256([]byte(verificador))
	return base64.RawURLEncoding.EncodeToString(suma[:])
}

// LongitudVerificadorPKCE es cuantos bytes de aleatoriedad lleva el
// verificador. RFC 7636 pide entre 43 y 128 caracteres; 32 bytes en hexadecimal
// son 64 caracteres, dentro del rango y con 256 bits de entropia.
const LongitudVerificadorPKCE = 32
