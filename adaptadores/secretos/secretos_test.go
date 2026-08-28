package secretos_test

import (
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/secretos"
	"github.com/marcosmatalab/plazum/puertos"
	"github.com/marcosmatalab/plazum/puertos/contrato"
)

// La suite de contrato es la puerta: si esto no pasa, el adaptador no cumple lo
// que el resto de la etapa 2 da por hecho.
func TestCryptoRandCumpleElContratoDeSecretos(t *testing.T) {
	contrato.Secretos(t, func() puertos.Secretos { return secretos.Nuevo() })
}

// Lo que el contrato no exige y este adaptador si: por debajo de 128 bits no
// emite. El contrato solo obliga a rechazar longitudes no positivas porque es
// lo que TODA implementacion tiene que cumplir; aqui se sube el suelo.
func TestPorDebajoDeCientoVeintiochoBitsNoEmite(t *testing.T) {
	s := secretos.Nuevo()
	for _, n := range []int{1, 4, 8, 15} {
		tok, err := s.Token(n)
		if err == nil {
			t.Errorf("Token(%d) devolvio %q sin error; %d bytes se agotan a fuerza bruta",
				n, tok, n)
			continue
		}
		// Error accionable: tiene que decir cuanto se pidio y cuanto hay que pedir.
		if !strings.Contains(err.Error(), "minimo") {
			t.Errorf("Token(%d) falla sin decir cual es el minimo: %v", n, err)
		}
	}
	if _, err := s.Token(secretos.Minimo); err != nil {
		t.Errorf("Token(%d) es exactamente el minimo y tiene que valer: %v", secretos.Minimo, err)
	}
}

// Un token tiene que ser hexadecimal decodificable: si no lo es, el que lo
// guarde en una cookie se encontrara caracteres que hay que escapar.
func TestElTokenEsHexadecimalDeVerdad(t *testing.T) {
	tok, err := secretos.Nuevo().Token(32)
	if err != nil {
		t.Fatal(err)
	}
	b, err := hex.DecodeString(tok)
	if err != nil {
		t.Fatalf("el token %q no decodifica como hexadecimal: %v", tok, err)
	}
	if len(b) != 32 {
		t.Fatalf("decodifica a %d bytes y se pidieron 32", len(b))
	}
	if strings.ToLower(tok) != tok {
		t.Fatalf("el token trae mayusculas (%q): dos representaciones del mismo "+
			"secreto invitan a comparar mal", tok)
	}
}

// La fuente se comparte entre goroutines desde el primer dia (cada peticion
// HTTP emite tokens). Si repite bajo concurrencia, dos sesiones distintas
// pueden acabar con el mismo identificador.
func TestBajoConcurrenciaNoRepite(t *testing.T) {
	const goroutines, porGoroutine = 16, 64
	s := secretos.Nuevo()

	var mu sync.Mutex
	vistos := make(map[string]bool, goroutines*porGoroutine)
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < porGoroutine; j++ {
				tok, err := s.Token(16)
				if err != nil {
					t.Error(err)
					return
				}
				mu.Lock()
				if vistos[tok] {
					t.Errorf("token repetido bajo concurrencia: %q", tok)
				}
				vistos[tok] = true
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
}

// Control negativo de la suite de contrato: se demuestra que una fuente rota
// (la que devuelve siempre lo mismo, que es como se rompe esto en la practica
// cuando alguien deja puesta una semilla fija) NO pasa el contrato. Sin esto,
// el verde de arriba no demuestra que el contrato tenga dientes.
//
// Se ejecuta relanzando el propio binario de test: un *testing.T no se puede
// fabricar a mano para observar si falla, asi que se observa el codigo de
// salida del proceso hijo, que es la unica senal honesta.
func TestElContratoDeSecretosCazaUnaFuenteQueRepite(t *testing.T) {
	if os.Getenv(marcaControlNegativo) == "1" {
		// Rama hija: aqui SE ESPERA que el contrato ponga el test en rojo.
		contrato.Secretos(t, func() puertos.Secretos { return fuenteQueRepite{} })
		return
	}
	// #nosec G204 -- se relanza el propio binario de test (os.Args[0]) con un
	// patron literal; no hay entrada de usuario en la linea de ordenes.
	hijo := exec.Command(os.Args[0], "-test.run=^"+t.Name()+"$", "-test.count=1")
	hijo.Env = append(os.Environ(), marcaControlNegativo+"=1")
	salida, err := hijo.CombinedOutput()
	if err == nil {
		t.Fatalf("una fuente que devuelve siempre el mismo token HA PASADO el contrato de "+
			"Secretos. Entonces el contrato no vigila nada y el verde de este paquete "+
			"no significa nada.\nsalida del hijo:\n%s", salida)
	}
	// Y ademas tiene que caer por donde debe, no por un fallo de fontaneria.
	if !strings.Contains(string(salida), "tokens seguidos iguales") {
		t.Fatalf("el hijo fallo, pero no por la repeticion de tokens. Si cae por otra cosa, "+
			"este control negativo no demuestra lo que dice.\nsalida:\n%s", salida)
	}
}

const marcaControlNegativo = "PLAZUM_CONTROL_NEGATIVO_SECRETOS"

// fuenteQueRepite es la mutacion: aleatoriedad constante.
type fuenteQueRepite struct{}

func (fuenteQueRepite) Token(n int) (string, error) {
	if n <= 0 {
		return "", errors.New("longitud no positiva")
	}
	return strings.Repeat("a", 2*n), nil
}
func (fuenteQueRepite) Bytes(b []byte) error {
	for i := range b {
		b[i] = 0xAA
	}
	return nil
}
