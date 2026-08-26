package pkcs7

import (
	"testing"
)

// La amplificacion de ber2der, medida y fijada por los dos lados.
//
// QUE ES. La transcodificacion de BER a DER puede devolver MUCHO mas de lo que
// se le da. En readObject, un objeto construido de longitud DEFINIDA devuelve
// contentEnd, que es la longitud DECLARADA, y no el offset que sus hijos
// consumieron de verdad. Un hijo que se pasa de largo se traga bytes que el
// abuelo vuelve a leer como si fueran su siguiente hermano, asi que esos bytes
// salen dos veces. Anidado, se multiplica. Es la confusion clasica del
// "excess walk" de BER.
//
// CUANTO, medido con las entradas que encontro nuestro fuzzing:
//
//	331 bytes ->   159.693  (x482)
//	631 bytes -> 1.197.909  (x1.898)
//	931 bytes -> 2.542.305  (x2.731)
//
// La razon se aplana hacia x4.000: cada dos bytes de entrada anaden unos ocho
// mil de salida. No es exponencial. Es un factor constante enorme.
//
// POR QUE ESTE TEST EXISTE EN VEZ DE UN ARREGLO. Tres cosas:
//
//  1. No es un fallo introducido aqui. Esta en el pkcs7 de aguas arriba, en la
//     version fijada y tambien en la de cabeza (el diff de ber.go entre las dos
//     no toca esto). Hay que reportarlo alli.
//  2. Arreglarlo en esta copia no quitaria la exposicion. Cadena.verificar
//     llama PRIMERO a timestamp.Parse, que parsea el mismo token con el pkcs7
//     de aguas arriba. Una guarda aqui dentro no llegaria a ejecutarse.
//  3. La defensa que si funciona esta en el llamante y ya esta puesta:
//     Cadena.VerificarOffline e Instante rechazan un token de mas de maxToken
//     (32 KiB) antes de parsearlo, en adaptadores/tsa/tsa.go, y eso acota los
//     dos caminos.
//
// Asi que lo que hace este test es CLAVAR el numero, por arriba y por abajo:
//
//   - si empeora, el tope de maxToken deja de ser suficiente y hay que bajarlo;
//   - si mejora, o sea si alguien arregla readObject aqui o aguas arriba, hay
//     que enterarse, porque entonces el tope se puede reconsiderar y este
//     comentario miente.
//
// Una constante escrita en un comentario envejece en silencio. Medida en cada
// `go test`, no.
func TestBer2derAmplificaYPorEsoElTokenLlevaTope(t *testing.T) {
	// La entrada la encontro el fuzzer de este paquete; esta commiteada en
	// testdata/fuzz/FuzzParseNoSeFiaDeNingunToken/67cadf62e490ab99.
	entrada := []byte("0\x800\n00000000000\n00000000000\n00000000000\n00000000000" +
		"\n00000000000\n00000000000\n00000000000\n000000000001000X" +
		"0000000000000000000000000000000000000000000000" +
		"\"00000000000000000000000000000000000" +
		"\"00000000000000000000000000000000000" +
		"\"00000000000000000000000000000000000" +
		"\"000000000000000000000000           \"\n \n0\n0\n0\n0\n0\n0\n0\n0\n0\n0" +
		"\n\n\n000000000\x00\x00")

	salida, err := ber2der(entrada)
	if err != nil {
		t.Fatalf("la entrada de referencia ya no transcodifica (%v). O ber2der ha cambiado, "+
			"o esta cadena se ha copiado mal. Sin ella, este test no mide nada y el tope de "+
			"maxToken en adaptadores/tsa se queda sin justificacion medida", err)
	}
	razon := float64(len(salida)) / float64(len(entrada))
	t.Logf("%d bytes de entrada producen %d de salida (x%.0f)", len(entrada), len(salida), razon)

	// Por arriba: si amplifica mas de lo medido, el tope de 32 KiB deja de
	// acotar el peor caso dentro del presupuesto de memoria.
	if razon > 1000 {
		t.Fatalf("la amplificacion ha EMPEORADO: x%.0f, y estaba medida en x482.\n"+
			"  maxToken en adaptadores/tsa/tsa.go esta dimensionado con ese numero:\n"+
			"  32 KiB por la razon asintotica medida (~x4.000) daban unos 130 MB de\n"+
			"  memoria transitoria, dentro del presupuesto de 256 MB del proyecto.\n"+
			"  Arreglo: volver a medir la asintota y bajar maxToken en el mismo commit.", razon)
	}

	// Por abajo: si alguien lo arregla, que no pase desapercibido.
	if razon < 100 {
		t.Fatalf("la amplificacion ha MEJORADO: x%.0f, y estaba medida en x482.\n"+
			"  Eso es una buena noticia y por eso este test la caza: significa que alguien\n"+
			"  ha arreglado readObject, aqui o aguas arriba, y entonces el comentario de\n"+
			"  maxToken en adaptadores/tsa/tsa.go dice algo que ya no es verdad.\n"+
			"  Arreglo: volver a medir, actualizar el numero aqui y en tsa.go, y decidir a\n"+
			"  conciencia si el tope se relaja.", razon)
	}
}
