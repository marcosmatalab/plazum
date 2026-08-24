package blobs

import (
	"bytes"
	"strings"
	"testing"
)

func k(b byte) []byte {
	c := make([]byte, 32)
	for i := range c {
		c[i] = b
	}
	return c
}
func n(b byte) []byte {
	x := make([]byte, 12)
	for i := range x {
		x[i] = b
	}
	return x
}

func TestSellarYAbrir(t *testing.T) {
	contenido := []byte("acta de la revision de accesos Q3")
	b, err := Sellar(k(1), n(1), contenido)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Abrir(k(1), b)
	if err != nil || !bytes.Equal(got, contenido) {
		t.Fatalf("%v %q", err, got)
	}
}

func TestClaveEquivocadaFallaPorCompromiso(t *testing.T) {
	b, _ := Sellar(k(1), n(1), []byte("x"))
	if _, err := Abrir(k(2), b); err == nil || !strings.Contains(err.Error(), "no compromete") {
		t.Fatalf("clave equivocada debe fallar el compromiso: %v", err)
	}
}

// Control negativo de la direccion: un blob con direccion ajena se detecta
// aunque la clave y el cifrado sean validos.
func TestBlobSustituidoSeDetecta(t *testing.T) {
	b1, _ := Sellar(k(1), n(1), []byte("contenido real"))
	b2, _ := Sellar(k(1), n(2), []byte("contenido sustituto"))
	b2.Hash = b1.Hash // el atacante conserva la direccion y cambia el contenido
	if _, err := Abrir(k(1), b2); err == nil || !strings.Contains(err.Error(), "sustituido") {
		t.Fatalf("un blob con direccion ajena debe detectarse: %v", err)
	}
}
