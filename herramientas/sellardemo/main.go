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
//	DUTIQ_ESCRIBIR_DEMO=1 go test ./nucleo/expediente -run TestGenerarDemo
//
// El orden importa y no es arbitrario: la raiz Merkle sale de las entradas y no
// depende del sello, asi que se puede sellar primero y firmar despues. Al reves
// no funcionaria, porque la firma del checkpoint SI cubre el digest del token.
package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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

func ejecutar() error {
	b, err := os.ReadFile(rutaDemo) // #nosec G304 -- ruta fija del repo
	if err != nil {
		return fmt.Errorf("no puedo leer %s: %w; ejecuta esto desde la raiz del repositorio", rutaDemo, err)
	}
	var d soloLaRaiz
	if err := json.Unmarshal(b, &d); err != nil {
		return fmt.Errorf("%s no es JSON valido: %w", rutaDemo, err)
	}
	if len(d.Cadena.Checkpoints) == 0 {
		return fmt.Errorf("%s no tiene ningun checkpoint que sellar", rutaDemo)
	}
	raiz, err := hex.DecodeString(d.Cadena.Checkpoints[0].RaizMerkle)
	if err != nil {
		return fmt.Errorf("la raiz Merkle no es hexadecimal: %w", err)
	}
	fmt.Printf("raiz Merkle a sellar: %s\n", d.Cadena.Checkpoints[0].RaizMerkle)

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
	fmt.Println("  DUTIQ_ESCRIBIR_DEMO=1 go test ./nucleo/expediente -run TestGenerarDemo")
	return nil
}
