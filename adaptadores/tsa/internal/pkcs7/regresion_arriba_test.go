package pkcs7

import (
	"crypto/x509/pkix"
	"strings"
	"testing"
)

// Las entradas que aguas arriba dice que hacian panico, fijadas aqui.
//
// POR QUE ESTE FICHERO EXISTE. El triaje de los 40 commits de aguas arriba
// (LEEME.md, apartado "El triaje del 26-08-2026") encontro
// `3562fcf934a0 Fix out-of-bounds panic in ber2der on malformed BER input`
// (CWE-125 / CWE-193), fechado el 21-07-2026, o sea DESPUES de la version
// fijada de esta copia (29-07-2025). Parecia un P0: el parser que come los
// bytes del tercero, con un panico alcanzable, y nosotros del lado viejo.
//
// MEDIDO, NO SUPUESTO: no lo es. Las dos guardas del parche ya estan en esta
// copia y las cuatro entradas devuelven error. El commit viene de la
// ascendencia de mozilla-services, que digitorus fusiono en agosto de 2026
// (`d75a4a2076bb Merge mozilla-services/pkcs7 master ancestry`): reintroduce
// historia cuyo cambio de codigo ya estaba aqui.
//
// Y esa es la leccion del triaje entero, por la que estos casos se quedan
// clavados: **contar commits no mide nada**. Un "40 commits por delante" puede
// ser, y en este caso es en su mayor parte, historia fusionada. Lo que mide es
// comparar el codigo, que es lo que hace
// TestLosDosParsersSiguenSiendoElMismoCodigo.
func TestLasEntradasQueHacianPanicoAguasArribaDevuelvenError(t *testing.T) {
	casos := []struct {
		nombre string
		bytes  []byte
	}{
		{"1F 80: etiqueta de numero alto que se sale", []byte{0x1F, 0x80}},
		{"1F 05: etiqueta de numero alto truncada", []byte{0x1F, 0x05}},
		{"30 81: longitud larga sin sus octetos", []byte{0x30, 0x81}},
		{"30 84 01: cuatro octetos de longitud y solo uno disponible", []byte{0x30, 0x84, 0x01}},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			// El panico se caza con recover y no se deja subir: un panico en un
			// test aborta el BINARIO entero de pruebas, y entonces los demas
			// casos no llegan a correr y el informe cuenta menos de lo que
			// parece. Es una trampa ya cometida en este repositorio.
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("PANICO con %s: %v\n"+
						"  Este es el parser que come los bytes de un tercero. Un panico aqui "+
						"lo dispara cualquiera que mande un sello mal formado.\n"+
						"  Arreglo: portar 3562fcf934a0 de digitorus/pkcs7 (las guardas "+
						"`offset >= berLen` y `offset+numberOfBytes > berLen` en readObject)", c.nombre, r)
				}
			}()
			if _, err := ber2der(c.bytes); err == nil {
				t.Errorf("%s ha salido SIN error. No es un panico, pero un BER truncado que "+
					"se da por bueno es peor: lo que salga de ahi se trata como contenido", c.nombre)
			}
			if _, err := Parse(c.bytes); err == nil {
				t.Errorf("%s: Parse lo ha aceptado", c.nombre)
			}
		})
	}
}

// El recorte 6, por el lado del que lo recibe: el error dice que ha pasado.
//
// Con la rama vieja, un sello firmado con DSA acababa rechazado igual, pero con
// "x509: cannot verify signature: algorithm unimplemented", que sale de dentro
// de crypto/x509 y no dice ni que el algoritmo era DSA ni que hacer.
func TestElRechazoDeDSADiceQueHaPasadoYComoSeArregla(t *testing.T) {
	// La rama se alcanza por getSignatureAlgorithm, que es interna; se prueba
	// por ahi porque es donde vive la decision.
	dsa := pkix.AlgorithmIdentifier{Algorithm: OIDDigestAlgorithmDSASHA1}
	sha256 := pkix.AlgorithmIdentifier{Algorithm: OIDDigestAlgorithmSHA256}
	_, err := getSignatureAlgorithm(dsa, sha256)
	if err == nil {
		t.Fatal("HALLAZGO: una firma DSA se da por verificable. DSA esta retirado y " +
			"crypto/x509 no lo implementa")
	}
	for _, quiero := range []string{"DSA", "Arreglo:"} {
		if !strings.Contains(err.Error(), quiero) {
			t.Errorf("el error no menciona %q, asi que quien lo lee no sabe ni que ha pasado "+
				"ni que hacer: %v", quiero, err)
		}
	}
}

// CONTROL NEGATIVO del recorte 6: lo que SI se verifica sigue verificandose.
// Sin esto, un getSignatureAlgorithm que devolviera error a todo pasaria el
// test de arriba y dejaria de comprobar cualquier sello.
func TestElRecorteDeDSANoSeLlevaPorDelanteLoQueSiSeVerifica(t *testing.T) {
	sha256 := pkix.AlgorithmIdentifier{Algorithm: OIDDigestAlgorithmSHA256}
	for _, caso := range []struct {
		nombre string
		alg    pkix.AlgorithmIdentifier
	}{
		{"RSA", pkix.AlgorithmIdentifier{Algorithm: OIDEncryptionAlgorithmRSA}},
		{"ECDSA con SHA-256", pkix.AlgorithmIdentifier{Algorithm: OIDDigestAlgorithmECDSASHA256}},
	} {
		if _, err := getSignatureAlgorithm(caso.alg, sha256); err != nil {
			t.Errorf("%s tenia que seguir verificandose y da %v. El recorte 6 solo saca DSA",
				caso.nombre, err)
		}
	}
}
