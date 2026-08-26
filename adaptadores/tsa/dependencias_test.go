package tsa

import (
	"fmt"
	"os"
	"strings"
	"testing"

	// El pkcs7 de AGUAS ARRIBA, importado A PROPOSITO y solo aqui.
	//
	// Es lo unico que este fichero vigila, asi que tiene que tocarlo
	// directamente. Antes se llegaba a el a traves de timestamp.Parse, y eso
	// dejo de valer el 26-08-2026 por dos motivos: nuestro codigo ya no llama a
	// timestamp.Parse (lo prohibe TestTimestampSoloConstruyeLaPeticion), y una
	// puerta que llega a lo que vigila por un camino que el producto ya no usa
	// esta midiendo otra cosa.
	arriba "github.com/digitorus/pkcs7"
)

// La puerta que sostiene lo que vendorizar NO arregla.
//
// POR QUE EXISTE, y QUE HA CAMBIADO el 26-08-2026.
//
// Desde que pkcs7 vive vendorizado en internal/pkcs7, ningun fichero de
// produccion nuestro lo importa por su ruta de modulo. La tentacion inmediata,
// y es razonable, es borrar de go.mod la linea marcada `// indirect`.
//
// EL MOTIVO VIEJO ERA MAS GRAVE Y YA NO APLICA: `timestamp.Parse` llamaba al
// pkcs7 de aguas arriba sobre los mismos bytes del expediente, asi que la
// version de 2023 (la del panico con dos bytes, 0x30 0x84) era ALCANZABLE por
// cualquiera que mandara un token roto. Eso se acabo: el TSTInfo se lee ahora
// con encoding/asn1 sobre el contenido de la copia vendorizada, y ningun
// codigo nuestro llega ya al pkcs7 transitivo.
//
// EL MOTIVO QUE QUEDA es mas pequeño y sigue siendo suficiente: `timestamp`
// sigue importando pkcs7, asi que el paquete ENTRA EN EL BINARIO aunque no se
// llame. Distribuir una version con un fallo conocido, aunque hoy sea
// inalcanzable, es exactamente lo que un analisis de composicion de software le
// va a senalar al comprador, y "no lo llamamos" no es algo que el comprador
// pueda comprobar sin leerse el codigo.
//
// **El arreglo de verdad es el objetivo declarado en DEPENDENCIAS.md**: que
// `timestamp` salga tambien, y con ella pkcs7 del grafo de modulos. Mientras
// tanto, esta puerta.
//
// Y sigue sin mirar la version, que es un dato que envejece: mira el
// COMPORTAMIENTO.

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
			r := entraEnPanico(func() { _, _ = arriba.Parse(b) })
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
