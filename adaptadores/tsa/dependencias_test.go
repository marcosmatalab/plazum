package tsa

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/digitorus/timestamp"
)

// La puerta que sostiene lo que vendorizar NO arregla.
//
// POR QUE EXISTE, y es el filo de esta casilla. Desde que pkcs7 vive
// vendorizado en internal/pkcs7, ningun fichero nuestro lo importa por su ruta
// de modulo. La tentacion inmediata, y es razonable, es borrar de go.mod la
// linea que quedo marcada `// indirect`.
//
// Borrarla trae el panico de vuelta. github.com/digitorus/timestamp SIGUE
// importando pkcs7: timestamp.Parse llama a pkcs7.Parse sobre los mismos bytes
// del expediente, y el go.mod de timestamp pide la version de 2023, que es
// justo la que reventaba con dos bytes (0x30 0x84: una SEQUENCE que declara
// cuatro bytes de longitud y no los trae). Sin la linea explicita en NUESTRO
// go.mod, la seleccion de version minima elige la de timestamp y el verificador
// vuelve a ser tumbable por cualquiera que mande un token roto.
//
// Asi que la puerta no mira la version, que es un dato que envejece: mira el
// COMPORTAMIENTO. Si el pkcs7 transitivo vuelve a ser uno que revienta, esto se
// pone rojo con el arreglo escrito.

// bytesQueReventaban son entradas que hicieron entrar en panico a ber2der de
// pkcs7. La primera la encontro el fuzzing de este adaptador y esta commiteada
// en testdata/fuzz/FuzzVerificarOffline.
var bytesQueReventaban = map[string][]byte{
	"SEQUENCE que declara 4 bytes de longitud y no los trae": {0x30, 0x84},
	"longitud larga truncada":                                {0x30, 0x84, 0xff, 0xff, 0xff},
	"longitud indefinida sin terminador":                     {0x30, 0x80},
	"tag de numero alto cortado":                             {0x3f, 0x81},
}

// entraEnPanico ejecuta f y devuelve lo que recupero, o nil. El panico se
// contiene aqui a proposito: un panico sin recuperar aborta el binario de test
// ENTERO y se lleva por delante el resultado de todos los demas, que es como se
// tapa un hallazgo sin querer.
func entraEnPanico(f func()) (recuperado any) {
	defer func() { recuperado = recover() }()
	f()
	return nil
}

func TestElPkcs7TransitivoNoEsElQueRevienta(t *testing.T) {
	for nombre, b := range bytesQueReventaban {
		t.Run(nombre, func(t *testing.T) {
			r := entraEnPanico(func() { _, _ = timestamp.Parse(b) })
			if r == nil {
				return
			}
			t.Fatalf("timestamp.Parse ha entrado en panico con %d bytes (%x): %v\n"+
				"  timestamp parsea el token con el pkcs7 DE AGUAS ARRIBA, no con la copia\n"+
				"  vendorizada, y el go.mod de timestamp pide la version de 2023, que es la\n"+
				"  que revienta. Alguien ha quitado de nuestro go.mod la linea\n"+
				"    require github.com/digitorus/pkcs7 v0.0.0-20250729175123-57bd227bfa2f // indirect\n"+
				"  creyendo que ya no hacia falta por estar vendorizado.\n"+
				"  Efecto: cualquiera tumba el verificador mandando un expediente con el\n"+
				"  token roto.\n"+
				"  Arreglo: devolver esa linea a go.mod y dejar el porque escrito al lado.",
				len(b), b, r)
		})
	}
}

// CONTROL NEGATIVO del detector, no del hecho. Sin esto no se sabe si el test
// de arriba vigila algo o si `entraEnPanico` devuelve siempre nil.
func TestElDetectorDePanicosDetectaUnPanico(t *testing.T) {
	if r := entraEnPanico(func() { panic("esto es a proposito") }); r == nil {
		t.Fatal("entraEnPanico no ha visto un panico puesto a mano. Mientras eso pase, " +
			"TestElPkcs7TransitivoNoEsElQueRevienta esta dando verde sin mirar nada")
	}
	// Y un indice fuera de rango, que es la forma exacta del panico que nos
	// costo esta dependencia, no un panic() literal.
	if r := entraEnPanico(func() {
		var v []byte
		_ = fmt.Sprint(v[3])
	}); r == nil {
		t.Fatal("entraEnPanico no ha visto un index out of range, que es la forma del " +
			"panico que vigila el test de arriba")
	}
	if r := entraEnPanico(func() {}); r != nil {
		t.Fatalf("falso positivo: entraEnPanico dice que una funcion sana ha reventado (%v)", r)
	}
}

// Y el aviso ANTES de que el panico vuelva: que la linea siga en go.mod. El
// test de arriba caza el efecto; este caza la causa y dice el nombre del
// fichero que hay que tocar.
func TestGoModSigueFijandoElPkcs7Transitivo(t *testing.T) {
	b, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("no puedo leer go.mod (%v). Si el fichero se movio, esta puerta estaria "+
			"comprobando el vacio", err)
	}
	s := string(b)
	if !strings.Contains(s, "github.com/digitorus/pkcs7") {
		t.Fatal("go.mod ya no fija github.com/digitorus/pkcs7.\n" +
			"  No se puede quitar aunque nuestro codigo no lo importe: lo importa\n" +
			"  github.com/digitorus/timestamp, y su propio go.mod pide la version de 2023,\n" +
			"  la del panico alcanzable con dos bytes. Sin la linea, MVS elige esa.\n" +
			"  Arreglo: go get github.com/digitorus/pkcs7@v0.0.0-20250729175123-57bd227bfa2f\n" +
			"  y dejar el porque escrito al lado de la linea.")
	}
	if strings.Contains(s, "github.com/digitorus/pkcs7 v0.0.0-20230713084857") {
		t.Fatal("go.mod ha vuelto a la version de pkcs7 de 2023, que es la del panico. " +
			"Arreglo: subirla a v0.0.0-20250729175123-57bd227bfa2f como minimo")
	}
}
