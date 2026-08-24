package ledger

import "testing"

// FuzzAbrirComprometido: entradas arbitrarias jamas tumban el verificador.
func FuzzAbrirComprometido(f *testing.F) {
	f.Add(make([]byte, 32), make([]byte, 12), []byte("cifrado"), []byte("compromiso"))
	f.Add([]byte("corta"), []byte("n"), []byte{}, []byte{})
	f.Fuzz(func(t *testing.T, clave, nonce, cifrado, compromiso []byte) {
		_, _ = AbrirComprometido(clave, nonce, cifrado, compromiso) // sin panicos
	})
}
