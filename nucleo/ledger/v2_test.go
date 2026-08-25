package ledger

import (
	"bytes"
	"crypto/ed25519"
	"errors"
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
	if !errors.Is(err, ErrClaveNoCompromete) {
		t.Fatalf("una clave sustituida debe fallar el compromiso, no GCM: %v", err)
	}
}

func TestCompromisoManipuladoSeRechaza(t *testing.T) {
	c, ks := cadenaDePrueba(t)
	c.Entradas[1].Compromiso[0] ^= 1
	// Por identidad: si esto fallara por la lapida o por la clave ausente, el
	// test estaria dando por probado un compromiso que nadie comprobo.
	if _, err := c.Leer(ks, 1); !errors.Is(err, ErrClaveNoCompromete) {
		t.Fatalf("un compromiso manipulado no puede abrir, y tiene que ser el compromiso "+
			"quien lo pare: %v", err)
	}
}

func TestBorrarDestruyeYDejaLapidaVerificable(t *testing.T) {
	c, ks := cadenaDePrueba(t)
	pub, priv, _ := ed25519.GenerateKey(deterministico{})
	if _, err := c.Borrar(ks, priv, 1, "Ley 2/2023 art. 32", "2026-08-24T10:00:00Z"); err != nil {
		t.Fatal(err)
	}
	// Irrecuperable, y por la lapida: el otro motivo por el que Leer puede
	// negarse tras destruir la clave es "clave no disponible (destruida sin
	// lapida?)", que seria justo el fallo que este test tendria que cazar.
	_, err := c.Leer(ks, 1)
	if !errors.Is(err, ErrEntradaSuprimida) {
		t.Fatalf("la entrada borrada debe decir su base legal: %v", err)
	}
	// la cadena sigue verificando entera, y el informe lista la supresion
	inf, err := c.Verificar(Confianza{ClaveOperador: pub})
	if err != nil {
		t.Fatalf("la cadena debe seguir integra tras el borrado: %v", err)
	}
	// La linea del informe, entera y exacta. Es lo que lee un auditor, asi que
	// se compara completa: con una subcadena, un informe que perdiera el indice
	// o el instante seguiria pasando.
	quiere := "entrada 1 suprimida con base legal Ley 2/2023 art. 32 el 2026-08-24T10:00:00Z"
	if len(inf.Suprimidas) != 1 || inf.Suprimidas[0] != quiere {
		t.Fatalf("el informe debe listar la supresion con su base:\n  quiero %q\n  tengo  %v",
			quiere, inf.Suprimidas)
	}
	// y las demas entradas siguen legibles
	if _, err := c.Leer(ks, 3); err != nil {
		t.Fatalf("borrar una entrada no toca las demas: %v", err)
	}
}

func TestBorrarSinBaseLegalSeRechaza(t *testing.T) {
	c, ks := cadenaDePrueba(t)
	_, priv, _ := ed25519.GenerateKey(deterministico{})
	// El indice 1 existe y no esta suprimido: si esto fallara por cualquiera de
	// esos dos motivos, la guarda de la base legal podria no existir.
	if _, err := c.Borrar(ks, priv, 1, "", "2026-08-24T10:00:00Z"); !errors.Is(err, ErrSinBaseLegal) {
		t.Fatalf("sin base legal no hay borrado: %v", err)
	}
}

func TestLapidaConFirmaAjenaNoVerifica(t *testing.T) {
	c, ks := cadenaDePrueba(t)
	pubBuena, _, _ := ed25519.GenerateKey(deterministico{})
	_, privAjena, _ := ed25519.GenerateKey(deterministico2{})
	if _, err := c.Borrar(ks, privAjena, 1, "RGPD art. 17", "2026-08-24T10:00:00Z"); err != nil {
		t.Fatal(err)
	}
	// ErrFirmaLapida y no "firma invalida": ese mismo texto lo dice tambien el
	// checkpoint (ErrFirmaCheckpoint), asi que la subcadena no distingue cual
	// de las dos comprobaciones paro el ataque.
	_, err := c.Verificar(Confianza{ClaveOperador: pubBuena})
	if !errors.Is(err, ErrFirmaLapida) {
		t.Fatalf("una lapida firmada por otra clave debe fallar por su firma: %v", err)
	}
}

func TestEntradaAlteradaRompeLaCadenaV2(t *testing.T) {
	c, _ := cadenaDePrueba(t)
	pub, _, _ := ed25519.GenerateKey(deterministico{})
	c.Entradas[2].Cifrado[0] ^= 1
	if _, err := c.Verificar(Confianza{ClaveOperador: pub}); !errors.Is(err, ErrEntradaAlterada) {
		t.Fatalf("una entrada alterada debe romper la verificacion: %v", err)
	}
}

// La contrapartida de afirmar por identidad. Desde que los tests comprueban
// con errors.Is, nadie mira el texto, y el texto es la mitad del contrato:
// CLAUDE.md pide causa, arreglo y cita, no "error inesperado". Aqui se pinan
// enteros los mensajes de los caminos que pasaron a centinela, para que meter
// el %w no se lleve por delante lo que lee quien opera.
func TestLosMensajesDeLaCadenaV2SiguenSiendoAccionables(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(deterministico{})

	c, ks := cadenaDePrueba(t)
	if _, err := c.Borrar(ks, priv, 1, "Ley 2/2023 art. 32", "2026-08-24T10:00:00Z"); err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre string
		err    error
		quiere string
	}{
		{"leer una entrada suprimida", errorDe(c.Leer(ks, 1)),
			"entrada 1 suprimida con base legal Ley 2/2023 art. 32 el 2026-08-24T10:00:00Z"},
		{"borrar dos veces", errorDe(c.Borrar(ks, priv, 1, "RGPD art. 17", "x")),
			"la entrada 1 ya esta suprimida con base legal Ley 2/2023 art. 32 el 2026-08-24T10:00:00Z"},
		{"borrar sin base legal", errorDe(c.Borrar(ks, priv, 2, "", "x")),
			"borrar exige base legal citada (Ley 2/2023 art. 32, RGPD art. 17...)"},
		{"clave que no compromete",
			errorDe(AbrirComprometido(claveFija(9), c.Entradas[0].Nonce, c.Entradas[0].Cifrado, c.Entradas[0].Compromiso)),
			"la clave no compromete este cifrado: clave equivocada o sustituida"},
	}
	for _, k := range casos {
		if k.err == nil || k.err.Error() != k.quiere {
			t.Errorf("%s:\n  quiero %q\n  tengo  %v", k.nombre, k.quiere, k.err)
		}
	}

	// Y los de Verificar, que necesitan cada uno su cadena rota a proposito.
	rota, _ := cadenaDePrueba(t)
	rota.Entradas[2].Cifrado[0] ^= 1
	quiere := "entrada 2: contenido alterado"
	if _, err := rota.Verificar(Confianza{ClaveOperador: pub}); err == nil || err.Error() != quiere {
		t.Errorf("entrada alterada:\n  quiero %q\n  tengo  %v", quiere, err)
	}

	ajena, ksAjena := cadenaDePrueba(t)
	_, privAjena, _ := ed25519.GenerateKey(deterministico2{})
	if _, e := ajena.Borrar(ksAjena, privAjena, 1, "RGPD art. 17", "2026-08-24T10:00:00Z"); e != nil {
		t.Fatal(e)
	}
	quiere = "lapida de la entrada 1: firma invalida"
	if _, err := ajena.Verificar(Confianza{ClaveOperador: pub}); err == nil || err.Error() != quiere {
		t.Errorf("firma de lapida:\n  quiero %q\n  tengo  %v", quiere, err)
	}
}

// errorDe se queda solo con el error de las funciones que devuelven dos
// valores, para que la tabla de arriba quepa de un vistazo.
func errorDe[T any](_ T, err error) error { return err }

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
