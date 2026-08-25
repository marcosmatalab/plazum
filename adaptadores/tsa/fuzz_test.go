package tsa

import (
	"crypto/sha256"
	"testing"
)

// FuzzVerificarOffline: un token arbitrario jamas tumba al verificador.
//
// Importa mas aqui que en el resto del proyecto porque este es el unico sitio
// donde el parseo lo hace codigo de terceros (pkcs7 y timestamp), sobre bytes
// que trae el expediente, o sea alguien de quien no nos fiamos. Un panico en
// el parser de ASN.1 seria una denegacion de servicio contra el verificador
// con solo mandar un token roto.
func FuzzVerificarOffline(f *testing.F) {
	h := sha256.Sum256([]byte("raiz merkle"))

	f.Add([]byte{})
	f.Add([]byte{0x30, 0x00})                   // SEQUENCE vacia
	f.Add([]byte{0x30, 0x80, 0x00, 0x00})       // longitud indefinida
	f.Add([]byte{0x30, 0x84, 0xff, 0xff, 0xff}) // longitud enorme y truncada
	f.Add([]byte("esto no es ASN.1 ni de lejos"))

	f.Fuzz(func(t *testing.T, token []byte) {
		// Sin anclas y con anclas: los dos caminos tienen que aguantar.
		sinAnclas := &Cadena{}
		_ = sinAnclas.VerificarOffline(h[:], token)

		conAnclas := &Cadena{Anclas: buena(t).pool}
		_ = conAnclas.VerificarOffline(h[:], token)

		// Instante parsea el mismo token por otro camino.
		_, _ = Instante(token)
	})
}
