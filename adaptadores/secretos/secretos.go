// Package secretos es la fuente de aleatoriedad del producto: crypto/rand y
// nada mas.
//
// Es la implementacion de puertos.Secretos que se instala. No hay aqui ninguna
// fuente determinista ni ninguna variable de paquete que se pueda sustituir, y
// eso es deliberado: el modo en que esta pieza falla de verdad no es un sesgo
// estadistico, es que alguien deja puesto un generador de pruebas y el binario
// que se instala emite los mismos identificadores de sesion que el test. Una
// fuente reproducible vive en el fichero _test.go que la necesita, se inyecta
// donde hace falta, y por tanto no puede llegar a produccion sin que se vea en
// la construccion del servidor.
//
// Contrato: puertos/contrato.Secretos, que se ejecuta en secretos_test.go.
package secretos

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/marcosmatalab/plazum/puertos"
)

// Minimo es la longitud mas corta, en bytes, que este adaptador acepta emitir.
//
// 16 bytes son 128 bits, que es el suelo por debajo del cual un identificador
// de sesion deja de ser inadivinable a fuerza bruta distribuida. El puerto solo
// exige rechazar longitudes no positivas; aqui se rechaza ademas todo lo que
// quede por debajo de este suelo, porque el error que se ve en la practica no
// es pedir cero bytes, es pedir cuatro "que ya son muchos".
const Minimo = 16

// CryptoRand emite secretos leyendo de crypto/rand.Reader.
//
// No tiene estado: se puede copiar, compartir entre goroutines y construir en
// cualquier sitio. crypto/rand.Reader es seguro para uso concurrente.
type CryptoRand struct{}

// Nuevo devuelve la fuente de secretos de produccion.
func Nuevo() CryptoRand { return CryptoRand{} }

// Comprobacion en compilacion de que este adaptador satisface el puerto.
var _ puertos.Secretos = CryptoRand{}

// Token devuelve n bytes de aleatoriedad en hexadecimal, o sea 2n caracteres.
func (c CryptoRand) Token(n int) (string, error) {
	if n < Minimo {
		return "", fmt.Errorf(
			"se han pedido %d bytes de secreto y el minimo son %d (128 bits): "+
				"un identificador de sesion o un token CSRF mas corto se adivina "+
				"a fuerza bruta. Arreglo: pide al menos %d",
			n, Minimo, Minimo)
	}
	b := make([]byte, n)
	if err := c.Bytes(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Bytes llena b entero. Nunca a medias: un relleno parcial deja ceros al final
// y nadie lo mira, asi que un fallo silencioso se convierte en un secreto que
// se adivina.
func (CryptoRand) Bytes(b []byte) error {
	if len(b) == 0 {
		// Pedir cero bytes es inocuo, no un error: el llamante puede estar
		// llenando un buffer de longitud calculada.
		return nil
	}
	// io.ReadFull y no rand.Read a secas: crypto/rand.Read ya llena entero o
	// devuelve error, pero dejarlo escrito impide que manana alguien lo cambie
	// por un Read cuyo n corto no mire nadie.
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return fmt.Errorf(
			"el sistema no da aleatoriedad (%w). Sin ella no se pueden emitir "+
				"sesiones ni tokens CSRF, asi que el servidor no debe arrancar. "+
				"Arreglo: en Linux comprueba que /dev/urandom existe y es legible "+
				"por el usuario que ejecuta plazum; en contenedores muy recortados "+
				"suele faltar el nodo del dispositivo", err)
	}
	return nil
}
