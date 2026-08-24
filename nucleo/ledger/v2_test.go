package ledger

import (
	"bytes"
	"crypto/ed25519"
	"strings"
	"testing"
)

func claveFija(b byte) []byte {
	c := make([]byte, 32)
	for i := range c {
		c[i] = b
	}
	return c
}
func nonceFijo(b byte) []byte {
	n := make([]byte, 12)
	for i := range n {
		n[i] = b
	}
	return n
}

func cadenaDePrueba(t *testing.T) (*CadenaV2, *Keystore) {
	t.Helper()
	c, ks := &CadenaV2{}, NuevoKeystore()
	for i := byte(0); i < 4; i++ {
		if _, err := c.Anadir(ks, claveFija(i+1), nonceFijo(i+1), []byte{'d', 'a', 't', 'o', i}); err != nil {
			t.Fatal(err)
		}
	}
	return c, ks
}

func TestCadenaV2SellaYLee(t *testing.T) {
	c, ks := cadenaDePrueba(t)
	got, err := c.Leer(ks, 2)
	if err != nil || !bytes.Equal(got, []byte{'d', 'a', 't', 'o', 2}) {
		t.Fatalf("leer: %v %q", err, got)
	}
}

// El control negativo del compromiso: una clave distinta se rechaza ANTES de
// llegar a GCM. Es la defensa contra los invisible salamanders.
func TestClaveSustituidaSeRechazaPorCompromiso(t *testing.T) {
	c, _ := cadenaDePrueba(t)
	e := c.Entradas[1]
	_, err := AbrirComprometido(claveFija(9), e.Nonce, e.Cifrado, e.Compromiso)
	if err == nil || !strings.Contains(err.Error(), "no compromete") {
		t.Fatalf("una clave sustituida debe fallar el compromiso, no GCM: %v", err)
	}
}

func TestCompromisoManipuladoSeRechaza(t *testing.T) {
	c, ks := cadenaDePrueba(t)
	c.Entradas[1].Compromiso[0] ^= 1
	if _, err := c.Leer(ks, 1); err == nil {
		t.Fatal("un compromiso manipulado no puede abrir")
	}
}

func TestBorrarDestruyeYDejaLapidaVerificable(t *testing.T) {
	c, ks := cadenaDePrueba(t)
	pub, priv, _ := ed25519.GenerateKey(deterministico{})
	if _, err := c.Borrar(ks, priv, 1, "Ley 2/2023 art. 32", "2026-08-24T10:00:00Z"); err != nil {
		t.Fatal(err)
	}
	// irrecuperable
	if _, err := c.Leer(ks, 1); err == nil || !strings.Contains(err.Error(), "suprimida con base legal") {
		t.Fatalf("la entrada borrada debe decir su base legal: %v", err)
	}
	// la cadena sigue verificando entera, y el informe lista la supresion
	inf, err := c.Verificar(pub)
	if err != nil {
		t.Fatalf("la cadena debe seguir integra tras el borrado: %v", err)
	}
	if len(inf.Suprimidas) != 1 || !strings.Contains(inf.Suprimidas[0], "Ley 2/2023") {
		t.Fatalf("el informe debe listar la supresion con su base: %v", inf.Suprimidas)
	}
	// y las demas entradas siguen legibles
	if _, err := c.Leer(ks, 3); err != nil {
		t.Fatalf("borrar una entrada no toca las demas: %v", err)
	}
}

func TestBorrarSinBaseLegalSeRechaza(t *testing.T) {
	c, ks := cadenaDePrueba(t)
	_, priv, _ := ed25519.GenerateKey(deterministico{})
	if _, err := c.Borrar(ks, priv, 1, "", "2026-08-24T10:00:00Z"); err == nil {
		t.Fatal("sin base legal no hay borrado")
	}
}

func TestLapidaConFirmaAjenaNoVerifica(t *testing.T) {
	c, ks := cadenaDePrueba(t)
	pubBuena, _, _ := ed25519.GenerateKey(deterministico{})
	_, privAjena, _ := ed25519.GenerateKey(deterministico2{})
	if _, err := c.Borrar(ks, privAjena, 1, "RGPD art. 17", "2026-08-24T10:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Verificar(pubBuena); err == nil || !strings.Contains(err.Error(), "firma invalida") {
		t.Fatalf("una lapida firmada por otra clave debe fallar: %v", err)
	}
}

func TestEntradaAlteradaRompeLaCadenaV2(t *testing.T) {
	c, _ := cadenaDePrueba(t)
	pub, _, _ := ed25519.GenerateKey(deterministico{})
	c.Entradas[2].Cifrado[0] ^= 1
	if _, err := c.Verificar(pub); err == nil || !strings.Contains(err.Error(), "alterado") {
		t.Fatalf("una entrada alterada debe romper la verificacion: %v", err)
	}
}

// lectores deterministas para generar claves de test sin crypto/rand
type deterministico struct{}

func (deterministico) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(i*7 + 1)
	}
	return len(p), nil
}

type deterministico2 struct{}

func (deterministico2) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(i*11 + 3)
	}
	return len(p), nil
}
