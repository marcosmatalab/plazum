package expediente

import (
	"os"
	"testing"
)

// Genera el expediente de demostracion que consume la CLI.
// OBLIGO_ESCRIBIR_DEMO=1 go test ./nucleo/expediente -run TestGenerarDemo
// Sin esa variable, el test valida el expediente en memoria y no toca disco:
// el fichero de la raiz (expediente-demo.json) solo cambia a proposito.
func TestGenerarDemo(t *testing.T) {
	e := construirExpediente(t)
	b, err := e.Guardar()
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("OBLIGO_ESCRIBIR_DEMO") == "" {
		return // validado en memoria; no se reescribe el fichero de la raiz
	}
	if err := os.WriteFile("../../expediente-demo.json", b, 0o644); err != nil {
		t.Skip("no se pudo escribir la demo:", err)
	}
}
