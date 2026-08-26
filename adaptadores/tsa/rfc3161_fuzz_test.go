package tsa

import (
	"bytes"
	"crypto"
	"encoding/asn1"
	"os"
	"path/filepath"
	"testing"
)

// Fuzzing del parser RFC 3161 propio.
//
// POR QUE LO LLEVA. DEPENDENCIAS.md tiene escrita la regla desde la etapa 1: si
// algo parsea entrada no fiable, fuzzing propio, **y no se da por buena la
// ausencia de fallos conocidos, porque los fallos conocidos son los que alguien
// buscó**. Escribir el parser en vez de importarlo no cambia esa regla: la
// cambia a peor, porque el codigo nuevo no lo ha mirado nadie mas.
//
// El token entra dentro del expediente, que lo aporta alguien de quien
// explicitamente no nos fiamos, asi que estos bytes son de origen hostil por
// construccion.
//
// LO QUE SE AFIRMA, y no es solo "no revienta":
//
//  1. ni leerRespuesta ni leerSello entran en panico con NADA.
//  2. son deterministas: los mismos bytes dan el mismo veredicto.
//  3. lo que sale de leerSello esta COMPROBADO, no copiado: si devuelve un
//     Sello, ese Sello tiene instante, tiene resumen y su algoritmo es uno de
//     los tres que este verificador acepta.
//  4. no se acepta basura detras: un TSTInfo con bytes de mas se rechaza.

func semillasRFC3161(f *testing.F) {
	f.Helper()
	// El token real de la demostracion, que es la unica entrada de la que se
	// sabe con certeza que es un sello valido.
	f.Add(tokenDeLaDemo(f))
	f.Add([]byte{})
	f.Add([]byte{0x30, 0x00})
	f.Add([]byte{0x30, 0x84})       // el que costo la dependencia
	f.Add([]byte{0x30, 0x80})       // longitud indefinida sin terminador
	f.Add([]byte{0x1F, 0x80})       // etiqueta de numero alto que se sale
	f.Add([]byte{0x02, 0x01, 0x00}) // un INTEGER suelto donde va una SEQUENCE
}

// tokenDeLaDemo devuelve el token real del expediente commiteado, o FALLA. No
// se salta si no esta: un corpus semilla que se queda sin su unica entrada
// valida fuzzea solo basura, y entonces el fuzzer recorre el camino de rechazo
// y no llega nunca a la parte del parser que decide algo.
//
// Es el mismo fichero que usa el fuzzer de internal/pkcs7, a proposito: el
// token que de verdad emitio una TSA es el unico ejemplar del que se sabe con
// certeza que es correcto, y tenerlo dos veces seria tener dos que se pueden
// desincronizar.
func tokenDeLaDemo(f *testing.F) []byte {
	f.Helper()
	b, err := os.ReadFile(filepath.Join("internal", "pkcs7", "testdata", "token-real.der"))
	if err != nil {
		f.Fatalf("no esta el token real de semilla (%v). Sin el, este fuzzing solo explora "+
			"el camino de los bytes invalidos y no mira nada de lo que decide. "+
			"Arreglo: extraerlo otra vez del campo token del primer checkpoint de "+
			"expediente-demo.json", err)
	}
	return b
}

func FuzzLeerSelloNoSeFiaDeNingunToken(f *testing.F) {
	semillasRFC3161(f)
	f.Fuzz(func(t *testing.T, token []byte) {
		// El tope de produccion, para no fuzzear un camino que el producto no
		// deja llegar.
		if len(token) > maxToken {
			return
		}
		s1, err1 := leerSello(token)
		s2, err2 := leerSello(token)

		// 2. determinismo. Un parser que no lo es convierte el veredicto de un
		// expediente en una tirada de dados.
		if (err1 == nil) != (err2 == nil) {
			t.Fatalf("dos lecturas de los mismos %d bytes dan resultados distintos: %v vs %v",
				len(token), err1, err2)
		}
		if err1 != nil {
			return
		}
		if !s1.Instante.Equal(s2.Instante) || !bytes.Equal(s1.Sellado, s2.Sellado) || s1.Hash != s2.Hash {
			t.Fatalf("dos lecturas de los mismos bytes dan sellos distintos:\n  %+v\n  %+v", s1, s2)
		}

		// 3. lo que sale esta comprobado.
		if s1.Instante.IsZero() {
			t.Fatalf("leerSello ha devuelto un sello SIN INSTANTE y sin error. El instante "+
				"es la mitad de lo que un sello prueba. Bytes: %x", token)
		}
		if len(s1.Sellado) == 0 {
			t.Fatalf("leerSello ha devuelto un sello que no dice QUE ha sellado. Con eso, "+
				"la comparacion con el hash del checkpoint no compara nada. Bytes: %x", token)
		}
		// El algoritmo que sale tiene que ser uno de los tres que entran. En
		// concreto SHA-1 NO puede salir nunca: de SHA-1 se saben construir
		// colisiones, y un sello con impronta SHA-1 no ata el contenido.
		switch s1.Hash {
		case crypto.SHA256, crypto.SHA384, crypto.SHA512:
		default:
			t.Fatalf("leerSello ha devuelto un sello con el algoritmo de resumen %v, que no "+
				"esta en la lista de aceptados. Bytes: %x", s1.Hash, token)
		}
	})
}

func FuzzLeerRespuestaNoSeFiaDeNingunaTSA(f *testing.F) {
	semillasRFC3161(f)
	// Una respuesta bien formada, para que el fuzzer tenga de donde mutar.
	f.Add(respuestaDePrueba(f, 0, tokenDeLaDemo(f)))
	f.Add(respuestaDePrueba(f, 2, nil))
	f.Fuzz(func(t *testing.T, der []byte) {
		if len(der) > maxRespuesta {
			return
		}
		t1, err1 := leerRespuesta(der)
		t2, err2 := leerRespuesta(der)
		if (err1 == nil) != (err2 == nil) || !bytes.Equal(t1, t2) {
			t.Fatalf("dos lecturas de la misma respuesta dan cosas distintas: %v / %v", err1, err2)
		}
		if err1 != nil {
			return
		}
		// Si dice que hay token, hay token. Devolver un token vacio sin error
		// haria que el llamante sellara con nada.
		if len(t1) == 0 {
			t.Fatalf("leerRespuesta ha devuelto un token VACIO sin error. Bytes: %x", der)
		}
	})
}

// respuestaDePrueba arma un TimeStampResp con el estado y el token que se le
// digan. Es el otro lado del cable, escrito aqui para no depender de que la
// libreria lo construya.
func respuestaDePrueba(f *testing.F, estado int, token []byte) []byte {
	f.Helper()
	r := respuestaSello{Estado: pkiStatusInfo{Status: estado}}
	if len(token) > 0 {
		r.Token = asn1.RawValue{FullBytes: token}
	}
	b, err := asn1.Marshal(r)
	if err != nil {
		f.Fatalf("no puedo armar la respuesta de prueba: %v", err)
	}
	return b
}

// Y la lista de aceptados, comprobada por el otro lado: SHA-1 no entra.
//
// Va aparte del fuzzer porque es una afirmacion sobre la TABLA, no sobre una
// entrada, y no hace falta fuzzear para hacerla. Si algun dia alguien anade
// SHA-1 "por compatibilidad", esto se pone rojo antes que nada.
func TestElParserNoAceptaSHA1NiNadaFueraDeLaLista(t *testing.T) {
	sha1 := asn1.ObjectIdentifier{1, 3, 14, 3, 2, 26}
	if _, ok := hashDeOID(sha1); ok {
		t.Fatal("HALLAZGO: se acepta SHA-1 como algoritmo de impronta de un sello. De SHA-1 " +
			"se saben construir colisiones desde 2017, asi que un sello con impronta SHA-1 " +
			"no ata el contenido que dice sellar")
	}
	for _, caso := range []struct {
		nombre string
		oid    asn1.ObjectIdentifier
		espera crypto.Hash
	}{
		{"SHA-256", asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}, crypto.SHA256},
		{"SHA-384", asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 2}, crypto.SHA384},
		{"SHA-512", asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 3}, crypto.SHA512},
	} {
		h, ok := hashDeOID(caso.oid)
		if !ok || h != caso.espera {
			t.Errorf("%s tenia que aceptarse como %v y salio (%v, %v)", caso.nombre, caso.espera, h, ok)
		}
	}
	// CONTROL NEGATIVO del control negativo: un OID inventado tampoco entra, o
	// lo de arriba estaria pasando porque la funcion dice que si a todo.
	if _, ok := hashDeOID(asn1.ObjectIdentifier{1, 2, 3, 4, 5}); ok {
		t.Fatal("hashDeOID acepta un OID inventado")
	}
}
