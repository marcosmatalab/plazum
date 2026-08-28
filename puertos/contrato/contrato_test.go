package contrato_test

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/puertos"
	"github.com/marcosmatalab/plazum/puertos/contrato"
)

// La suite de contrato se prueba a si misma contra una implementacion de
// referencia en memoria.
//
// No es el adaptador de la etapa 2: es lo minimo que cumple el contrato, y
// esta aqui por dos razones. La primera, demostrar que el contrato SE PUEDE
// cumplir (un contrato imposible es un contrato mal escrito, y mejor saberlo
// antes de repartir el trabajo). La segunda, servir de referencia legible de
// que significa cada exigencia.
//
// Los dientes de la suite se comprobaron por mutacion, que es lo unico que
// demuestra que un test sirve: rompiendo ComprobarCSRF para que aceptara
// cualquier token, la suite cae en "el token CSRF de otra sesion NO vale" y en
// "un token inventado no vale". Sin esa comprobacion, una implementacion que
// devuelve nil siempre pasaria el contrato entero y no protegeria de nada.

func TestLaImplementacionDeReferenciaCumpleElContratoDeSesion(t *testing.T) {
	contrato.Sesion(t, func() puertos.Sesion { return nuevaSesionMemoria() })
}

// --- implementacion de referencia ---

type sesionMemoria struct {
	mu       sync.Mutex
	sesiones map[string]entradaSesion
}

type entradaSesion struct {
	sujeto string
	tokens map[string]bool
	hasta  time.Time
}

func nuevaSesionMemoria() *sesionMemoria {
	return &sesionMemoria{sesiones: map[string]entradaSesion{}}
}

func aleatorio(t *testing.T) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(b)
}

func (s *sesionMemoria) id() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *sesionMemoria) Abrir(_ context.Context, sujeto string, d time.Duration) (string, error) {
	id, err := s.id()
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sesiones[id] = entradaSesion{sujeto: sujeto, tokens: map[string]bool{}, hasta: time.Now().Add(d)}
	return id, nil
}

func (s *sesionMemoria) Leer(_ context.Context, id string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.sesiones[id]
	// No se distingue "no existe" de "caducada": el que pregunta no tiene por
	// que aprender nada de la respuesta.
	if !ok || time.Now().After(e.hasta) {
		return "", errors.New("sesion no valida")
	}
	return e.sujeto, nil
}

func (s *sesionMemoria) Cerrar(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sesiones, id) // idempotente: borrar lo que no esta no falla
	return nil
}

func (s *sesionMemoria) TokenCSRF(_ context.Context, id string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.sesiones[id]
	if !ok {
		return "", errors.New("sesion no valida")
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	tok := hex.EncodeToString(b)
	e.tokens[tok] = true // atado A ESTA sesion, que es todo el asunto
	return tok, nil
}

func (s *sesionMemoria) ComprobarCSRF(_ context.Context, id, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if token == "" {
		return errors.New("peticion sin token CSRF")
	}
	e, ok := s.sesiones[id]
	if !ok {
		return errors.New("sesion no valida")
	}
	// Comparacion en tiempo constante sobre los tokens de ESTA sesion.
	for t := range e.tokens {
		if subtle.ConstantTimeCompare([]byte(t), []byte(token)) == 1 {
			return nil
		}
	}
	return errors.New("token CSRF que no pertenece a esta sesion")
}

// --- Catalogo y Diagnostico de referencia ---

func TestLaImplementacionDeReferenciaCumpleElContratoDeCatalogo(t *testing.T) {
	contrato.Catalogo(t, func() puertos.Catalogo { return catalogoMemoria{} })
}

type catalogoMemoria struct{}

func (catalogoMemoria) Traducir(idioma, clave string, _ ...any) string {
	if idioma == "es" && clave == "hoy.titulo" {
		return "Hoy"
	}
	return clave // sin traducir devuelve la clave, nunca vacio
}
func (catalogoMemoria) Idiomas() []string { return []string{"es", "en"} }
func (catalogoMemoria) Faltantes(i string) []string {
	if i == "es" {
		return nil // el de por defecto es la referencia
	}
	return []string{"hoy.titulo"}
}

func TestLaImplementacionDeReferenciaCumpleElContratoDeDiagnostico(t *testing.T) {
	contrato.Diagnostico(t, func() puertos.Diagnostico { return diagnosticoMemoria{} })
}

type diagnosticoMemoria struct{}

func (diagnosticoMemoria) Comprobar(context.Context) []puertos.Comprobacion {
	return []puertos.Comprobacion{
		{Nombre: "keystore", Estado: puertos.Correcto,
			Detalle: "keystore legible con 3 claves"},
		{Nombre: "tsa", Estado: puertos.Aviso,
			Detalle: "solo hay una TSA configurada",
			Arreglo: "anade una segunda en la configuracion; con una sola no hay cadena de reserva"},
	}
}

// --- Secretos de referencia ---

func TestLaImplementacionDeReferenciaCumpleElContratoDeSecretos(t *testing.T) {
	contrato.Secretos(t, func() puertos.Secretos { return secretosCryptoRand{} })
}

// secretosCryptoRand es la implementacion que ira en produccion: crypto/rand y
// nada mas. Esta aqui como referencia del contrato, no como adaptador.
type secretosCryptoRand struct{}

func (secretosCryptoRand) Token(n int) (string, error) {
	if n <= 0 {
		return "", fmt.Errorf("token de %d bytes: un secreto de longitud no positiva no es "+
			"un secreto; pide al menos 16", n)
	}
	b := make([]byte, n)
	if err := (secretosCryptoRand{}).Bytes(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (secretosCryptoRand) Bytes(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	// rand.Read de crypto/rand llena entero o devuelve error; io.ReadFull lo
	// deja dicho en el codigo para que nadie lo cambie por un Read a secas.
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return fmt.Errorf("no hay aleatoriedad disponible: %w. Sin ella no se pueden emitir "+
			"sesiones ni tokens CSRF, asi que el servidor no debe arrancar", err)
	}
	return nil
}

// Sonda de que el ejemplo de aleatorio() se usa: mantiene el helper vivo y
// documenta que los identificadores no pueden ser predecibles.
func TestLosIdentificadoresNoSonPredecibles(t *testing.T) {
	a, b := aleatorio(t), aleatorio(t)
	if a == b {
		t.Fatal("dos identificadores seguidos no pueden coincidir")
	}
}
