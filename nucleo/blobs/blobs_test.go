package blobs

import (
	"bytes"
	"errors"
	"testing"

	"dutiq/nucleo/ledger"
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
	// Por identidad y no por texto: el otro rechazo posible de Abrir es
	// ErrBlobSustituido, y afirmar "sustituido" contra "sustituida" es una
	// vocal de margen.
	if _, err := Abrir(k(2), b); !errors.Is(err, ledger.ErrClaveNoCompromete) {
		t.Fatalf("clave equivocada debe fallar el compromiso: %v", err)
	}
}

// Control negativo de la direccion: un blob con direccion ajena se detecta
// aunque la clave y el cifrado sean validos.
func TestBlobSustituidoSeDetecta(t *testing.T) {
	b1, _ := Sellar(k(1), n(1), []byte("contenido real"))
	b2, _ := Sellar(k(1), n(2), []byte("contenido sustituto"))
	b2.Hash = b1.Hash // el atacante conserva la direccion y cambia el contenido
	// Tiene que parar la comprobacion de la DIRECCION, no la del compromiso:
	// la clave es la buena y el cifrado es valido, asi que si esto fallara por
	// ErrClaveNoCompromete el test estaria probando otra cosa.
	_, err := Abrir(k(1), b2)
	if !errors.Is(err, ErrBlobSustituido) {
		t.Fatalf("un blob con direccion ajena debe detectarse por la direccion: %v", err)
	}
	if errors.Is(err, ledger.ErrClaveNoCompromete) {
		t.Fatalf("fallo el compromiso, no la direccion: %v", err)
	}
}

// La contrapartida de afirmar por identidad: si nadie mira ya el texto, nada
// impide que un refactor deje el mensaje en "error inesperado". Los dos
// mensajes de Abrir se pinan enteros, porque son lo que lee quien opera.
func TestLosMensajesDeAbrirSiguenSiendoAccionables(t *testing.T) {
	b1, _ := Sellar(k(1), n(1), []byte("contenido real"))
	b2, _ := Sellar(k(1), n(2), []byte("contenido sustituto"))
	b2.Hash = b1.Hash

	_, err := Abrir(k(1), b2)
	quiere := "el contenido no corresponde a la direccion " + b1.Hash[:12] + ": blob sustituido"
	if err == nil || err.Error() != quiere {
		t.Fatalf("mensaje de blob sustituido:\n  quiero %q\n  tengo  %v", quiere, err)
	}

	_, err = Abrir(k(2), b1)
	quiere = "la clave no compromete este cifrado: clave equivocada o sustituida"
	if err == nil || err.Error() != quiere {
		t.Fatalf("mensaje de clave equivocada:\n  quiero %q\n  tengo  %v", quiere, err)
	}
}
