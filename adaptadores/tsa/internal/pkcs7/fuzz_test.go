package pkcs7

import (
	"bytes"
	"crypto/x509"
	"errors"
	"os"
	"testing"
	"time"
)

// Fuzzing propio del parser vendorizado.
//
// POR QUE ESTE FICHERO ES LA MITAD DEL VALOR DE VENDORIZAR. Este paquete come
// bytes de un tercero: el token que trae el expediente lo firma una TSA, pero
// quien nos entrega el fichero es el emisor, de quien explicitamente no nos
// fiamos. Mientras la libreria era una dependencia, su fuzzing era el de otro y
// solo corria en el CI de otro; una version fijada por pseudo-version envejecio
// tres anos con un panico alcanzable con dos bytes dentro. Aqui el codigo esta
// en el repositorio y su fuzzing corre en cada `go test`.
//
// EL INVARIANTE NO ES SOLO "NO REVIENTA". Un parser que no entra en panico y
// devuelve un token dado por bueno es peor que uno que revienta. Lo que se
// afirma sobre CUALQUIER cadena de bytes, la traiga quien la traiga:
//
//	1. no entra en panico (lo comprueba el propio motor de fuzzing);
//	2. Parse no devuelve nunca las dos cosas a la vez ni ninguna de las dos:
//	   o token o error;
//	3. es determinista: dos parseos de los mismos bytes dan el mismo veredicto;
//	4. NINGUN token verifica contra un almacen de confianza vacio. Esta es la
//	   de verdad: si el fuzzer encontrara una cadena que sale valida sin una
//	   sola raiz que la respalde, el anclaje del expediente no valdria nada;
//	5. sin instante no verifica NADA, ni el token bueno. Es lo que hace que el
//	   veredicto no dependa del reloj de la maquina que verifica;
//	6. la transcodificacion BER a DER es idempotente, que es lo minimo que se
//	   le pide a una canonicalizacion: si dos formas distintas de los mismos
//	   bytes salen adelante, el hash del token no identifica al sello.
//
// Lo que aqui NO se afirma, y el hueco es el hallazgo: que no amplifique. Se
// intento tres veces (x2, x4, x64) y el fuzzer tumbo las tres. ber2der SI
// amplifica, hasta x482 medido con 331 bytes. El numero esta clavado por los
// dos lados en TestBer2derAmplificaYPorEsoElTokenLlevaTope, y la defensa esta
// en el llamante, que rechaza un token de mas de maxToken antes de parsearlo.
//
// EL CORPUS SEMILLA lleva un token de una TSA de verdad (el del expediente de
// demostracion, testdata/token-real.der). Sin el, el fuzzer pasa el rato
// rebotando en "input data is empty" y no llega nunca a la parte interesante,
// que es la que se recorre CUANDO la estructura es casi valida.

// tokenReal es el sello RFC 3161 autentico del expediente de demostracion.
// Sirve de semilla y de control positivo: si dejara de parsearse, el resto de
// las afirmaciones de este fichero se estarian haciendo sobre basura.
func tokenReal(t testing.TB) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/token-real.der")
	if err != nil {
		t.Fatalf("no esta el token real de semilla (%v). Sin el, el fuzzing de este "+
			"paquete solo explora el camino de los bytes invalidos. "+
			"Arreglo: extraerlo otra vez del campo token del primer checkpoint de "+
			"expediente-demo.json", err)
	}
	return b
}

// almacenVacio es un almacen de confianza NO nil y sin una sola raiz. Es el
// estado del auditor que aun no ha cargado nada, y contra el no puede verificar
// ni el sello mas legitimo del mundo.
func almacenVacio() x509.VerifyOptions {
	return x509.VerifyOptions{
		Roots:       x509.NewCertPool(),
		CurrentTime: time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC),
		KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageTimeStamping},
	}
}

func FuzzParseNoSeFiaDeNingunToken(f *testing.F) {
	real := tokenReal(f)
	f.Add(real)
	// Prefijos del token bueno: estructura valida cortada por la mitad, que es
	// donde viven los "leer mas alla del final".
	for _, n := range []int{2, 3, 5, 16, 64, 256, 1024, len(real) - 1} {
		if n > 0 && n < len(real) {
			f.Add(real[:n])
		}
	}
	f.Add([]byte{})
	f.Add([]byte{0x30, 0x84})                               // el panico que nos costo esta dependencia
	f.Add([]byte{0x30, 0x00})                               // SEQUENCE vacia
	f.Add([]byte{0x30, 0x80, 0x00, 0x00})                   // longitud indefinida
	f.Add([]byte{0x30, 0x84, 0xff, 0xff, 0xff})             // longitud enorme y truncada
	f.Add([]byte{0x30, 0x81, 0x00})                         // longitud larga con cero delante
	f.Add([]byte{0x24, 0x80, 0x04, 0x01, 0x41, 0x00, 0x00}) // OCTET STRING troceada
	f.Add([]byte{0x30, 0x03, 0x06, 0x01, 0x2a})             // ContentInfo con OID corto
	f.Add([]byte{0xbf, 0x81, 0x01, 0x02, 0x30, 0x00})       // tag de numero alto
	f.Add([]byte("esto no es ASN.1 ni de lejos"))

	f.Fuzz(func(t *testing.T, token []byte) {
		// 6. la transcodificacion BER a DER, que es la que toca los bytes.
		//
		// VA PRIMERO, Y NO ES CASUAL. Estaba al final, detras del `return` que
		// se toma cuando Parse falla, y una mutacion del codificador de
		// longitudes (lengthLength devolviendo siempre 4) dejo este bloque sin
		// ejecutar NI UNA VEZ: Parse pasaba a fallar sobre todas las semillas,
		// el fuzz salia en verde y la mutacion parecia no cazada. ber2der tiene
		// su propio contrato y se comprueba cuando ber2der responde, no cuando
		// responde otro.
		if der, err := ber2der(token); err == nil {
			otra, err := ber2der(der)
			if err != nil {
				t.Fatalf("ber2der falla sobre su propia salida (%v). Una canonicalizacion "+
					"que no acepta lo que ella misma produce no es una canonicalizacion. "+
					"Entrada: %x", err, token)
			}
			if !bytes.Equal(der, otra) {
				t.Fatalf("ber2der no es idempotente sobre %d bytes: la segunda pasada cambia "+
					"el resultado (%d -> %d bytes). Dos formas distintas de los mismos bytes "+
					"significa que el hash del token no identifica al sello",
					len(token), len(der), len(otra))
			}
			// AQUI NO SE AFIRMA NADA SOBRE LA AMPLIFICACION, y el hueco es
			// deliberado: ber2der AMPLIFICA, esta medido, y una cota
			// multiplicativa aqui seria una puerta que se cae sola.
			//
			// Lo que paso, porque la historia es el hallazgo: la primera
			// version afirmaba x2 y el fuzzer la tumbo en dos minutos (50 bytes
			// -> 123). Se subio a x4 y la tumbo otra vez (125 -> 667, x5,3). Se
			// subio a x64 y la tumbo otra vez (331 -> 159.693, x482). Las tres
			// entradas estan commiteadas en testdata/fuzz y siguen siendo
			// semillas utiles.
			//
			// La afirmacion de dos caras, con el numero medido, vive en
			// TestBer2derAmplificaYPorEsoElTokenLlevaTope. La defensa de verdad
			// vive en el llamante: Cadena.VerificarOffline rechaza un token de
			// mas de maxToken ANTES de parsearlo.
		}

		p7, err := Parse(token)

		// 2. o token o error, nunca las dos cosas y nunca ninguna.
		if (p7 == nil) == (err == nil) {
			t.Fatalf("Parse ha devuelto p7=%v y err=%v sobre %d bytes. Un llamante que mire "+
				"solo el error se comeria un nil, o daria por bueno un token con error",
				p7 != nil, err, len(token))
		}

		// 3. determinismo.
		p7b, errb := Parse(token)
		if (err == nil) != (errb == nil) {
			t.Fatalf("dos parseos de los mismos %d bytes dan veredictos distintos: %v y %v; "+
				"la verificacion offline tiene que ser reproducible", len(token), err, errb)
		}
		if err != nil {
			return
		}
		if !bytes.Equal(p7.Content, p7b.Content) || len(p7.Certificates) != len(p7b.Certificates) {
			t.Fatalf("dos parseos de los mismos %d bytes dan contenidos distintos", len(token))
		}

		// 4. nada verifica sin una raiz que lo respalde.
		if err := p7.VerifyWithOpts(almacenVacio()); err == nil {
			t.Fatalf("HALLAZGO GRAVE: un token de %d bytes VERIFICA contra un almacen de "+
				"confianza vacio. Si esto pasa, el anclaje del expediente no prueba nada: "+
				"cualquiera puede fabricar un sello y el verificador lo acepta. "+
				"Bytes: %x", len(token), token)
		}

		// 5. sin instante no verifica nada.
		sinInstante := almacenVacio()
		sinInstante.CurrentTime = time.Time{}
		if err := p7.VerifyWithOpts(sinInstante); !errors.Is(err, ErrSinInstante) {
			t.Fatalf("sin opts.CurrentTime la verificacion tenia que negarse con "+
				"ErrSinInstante y ha devuelto %v. Esa rama es la que consultaba el reloj "+
				"de la maquina aguas arriba, y con ella el mismo expediente sale valido "+
				"hoy e invalido dentro de cinco anos", err)
		}

		// 7. coherencia entre los dos caminos: si Parse ha salido bien, ber2der,
		// que Parse llama primero, tiene que haber salido bien tambien.
		if _, err := ber2der(token); err != nil {
			t.Fatalf("Parse ha salido bien y ber2der, que Parse llama primero, ha fallado "+
				"con %v: los dos caminos tienen que contar lo mismo", err)
		}
	})
}
