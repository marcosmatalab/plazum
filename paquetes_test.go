package obligo

import (
	"testing"

	"obligo/nucleo/corpus"
)

// TestTodosLosPaquetesPublicadosPasanElLinter es la puerta de la semana 0:
// un paquete que no pasa el linter no entra al repositorio. Cargar ya ejecuta
// el linter y rechaza el directorio entero si algo esta mal.
func TestTodosLosPaquetesPublicadosPasanElLinter(t *testing.T) {
	ps, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatalf("el corpus publicado no pasa el linter: %v", err)
	}
	if len(ps) < 2 {
		t.Fatalf("esperaba al menos ens y demo-empresa, hay %d", len(ps))
	}
	for _, p := range ps {
		if p.Fuente == "" {
			t.Errorf("%s sin fuente", p.URN)
		}
	}
}
