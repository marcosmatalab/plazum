// Comando sellardemo: sella el expediente de demostracion contra una TSA REAL.
//
// Por que existe. Lo primero que hace cualquiera es `dutiq verify` sobre el
// demo. Si el demo lleva un sello de relleno, eso falla, y es la peor primera
// impresion posible para un producto cuya tesis es la verificabilidad. La
// solucion no es un atajo en el verificador (eso seria una puerta trasera en la
// unica pieza que no puede tenerla), es que el demo lleve un sello autentico.
//
// Se ejecuta A MANO y muy de vez en cuando: solo cuando cambia el contenido del
// expediente demo, porque entonces cambia su raiz Merkle y el sello viejo deja
// de cuadrar. El CI NUNCA lo ejecuta: sale a la red, y el CI es hermetico.
//
// Uso:
//
//	go run ./herramientas/sellardemo
//
// Y despues, para que el sello entre en el expediente y quede firmado con el:
//
//	go run ./herramientas/generardemo -escribir
//
// EL HUEVO Y LA GALLINA, que aparece cada vez que cambia el CONTENIDO de la
// cadena y no solo el sello. generardemo se niega a escribir un expediente que
// no verifica, y no verifica sin un sello que cubra su raiz Merkle. Pero la raiz
// nueva solo existe despues de construirlo. Con -raiz se rompe el circulo: el
// error de generardemo dice cual es la raiz que esperaba, se le pasa aqui, y
// despues generardemo ya puede escribir.
//
//	go run ./herramientas/generardemo             # falla y dice "se esperaba <raiz>"
//	go run ./herramientas/sellardemo -raiz <raiz>
//	go run ./herramientas/generardemo -escribir
//
// El orden importa y no es arbitrario: la raiz Merkle sale de las entradas y no
// depende del sello, asi que se puede sellar primero y firmar despues. Al reves
// no funcionaria, porque la firma del checkpoint SI cubre el digest del token.
package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dutiq/adaptadores/tsa"
)

const (
	rutaDemo  = "expediente-demo.json"
	rutaSello = "nucleo/expediente/testdata/sello-demo.bin"
)

// soloLaRaiz lee del demo lo unico que hace falta: la raiz Merkle del primer
// checkpoint. Se parsea a mano y no con el tipo Expediente para que esta
// herramienta no se rompa cada vez que el formato crezca.
type soloLaRaiz struct {
	Cadena struct {
		Checkpoints []struct {
			RaizMerkle string `json:"raiz_merkle"`
		} `json:"checkpoints"`
	} `json:"cadena"`
}

func main() {
	if err := ejecutar(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// LongitudRaiz son los bytes de un SHA-256, que es lo que sella una TSA aqui.
const LongitudRaiz = 32

// ErrRaizIlegible: lo que se ha dado como raiz no es un SHA-256 en hexadecimal.
// Con centinela y no comparando texto, que es la convencion del proyecto.
var ErrRaizIlegible = errors.New("la raiz a sellar no es un SHA-256 en hexadecimal")

// raizASellar decide QUE se sella. De la bandera si viene, y del expediente
// publicado si no.
func raizASellar(bandera string) ([]byte, string, error) {
	hexa := strings.TrimSpace(bandera)
	origen := "la bandera -raiz"

	if hexa == "" {
		b, err := os.ReadFile(rutaDemo) // #nosec G304 -- ruta fija del repo
		if err != nil {
			return nil, "", fmt.Errorf("no puedo leer %s: %w; ejecuta esto desde la raiz "+
				"del repositorio, o pasa la raiz con -raiz", rutaDemo, err)
		}
		var d soloLaRaiz
		if err := json.Unmarshal(b, &d); err != nil {
			return nil, "", fmt.Errorf("%s no es JSON valido: %w", rutaDemo, err)
		}
		if len(d.Cadena.Checkpoints) == 0 {
			return nil, "", fmt.Errorf("%s no tiene ningun checkpoint que sellar", rutaDemo)
		}
		hexa = d.Cadena.Checkpoints[0].RaizMerkle
		origen = rutaDemo
	}

	raiz, err := hex.DecodeString(hexa)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %q (de %s): %v", ErrRaizIlegible, hexa, origen, err)
	}
	if len(raiz) != LongitudRaiz {
		// Sin esto, media raiz pegada de un mensaje de error se sella tan
		// tranquila y el sello resultante no cubre nada.
		return nil, "", fmt.Errorf("%w: %q (de %s) son %d bytes y un SHA-256 son %d",
			ErrRaizIlegible, hexa, origen, len(raiz), LongitudRaiz)
	}
	return raiz, hexa, nil
}

func ejecutar() error {
	banderaRaiz := flag.String("raiz", "",
		"raiz Merkle a sellar, en hexadecimal. Vacio = la del expediente publicado. "+
			"Se usa cuando la raiz nueva todavia no existe en disco")
	flag.Parse()

	raiz, hexa, err := raizASellar(*banderaRaiz)
	if err != nil {
		return err
	}
	fmt.Printf("raiz Merkle a sellar: %s\n", hexa)

	cadena, err := tsa.PorDefecto()
	if err != nil {
		return err
	}
	fmt.Printf("pidiendo sello a %d TSAs en orden...\n", len(cadena.Autoridades))
	token, err := cadena.Sellar(raiz)
	if err != nil {
		return fmt.Errorf("no se pudo sellar: %w", err)
	}
	// Sellar ya verifica antes de devolver, pero se comprueba otra vez aqui de
	// forma explicita: lo que se va a commitear tiene que verificar.
	if err := cadena.VerificarOffline(raiz, token); err != nil {
		return fmt.Errorf("el sello obtenido no verifica: %w", err)
	}
	inst, err := tsa.Instante(token)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(rutaSello), 0o750); err != nil {
		return err
	}
	if err := os.WriteFile(rutaSello, token, 0o600); err != nil {
		return err
	}
	fmt.Printf("sello de %s guardado en %s (%d bytes)\n", inst.Format("2006-01-02 15:04:05 MST"), rutaSello, len(token))
	fmt.Println()
	fmt.Println("ahora, para que entre en el expediente y quede firmado con el:")
	fmt.Println("  go run ./herramientas/generardemo -escribir")
	return nil
}
