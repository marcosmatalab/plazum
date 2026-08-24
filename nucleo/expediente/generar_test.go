package expediente

import (
	"os"
	"testing"
)

// Genera el expediente de demostracion que consume la CLI.
// go test ./expediente -run TestGenerarDemo -args -escribir
func TestGenerarDemo(t *testing.T) {
	e := construirExpediente(t)
	b, err := e.Guardar()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("../expediente-demo.json", b, 0o644); err != nil {
		t.Skip("no se pudo escribir la demo:", err)
	}
}
