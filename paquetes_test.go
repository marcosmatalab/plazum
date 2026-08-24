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

// TestLosDoradosPublicadosPasanContraElMotor es la puerta de cobertura: cada
// reloj del corpus publicado se recalcula con el motor y se compara con el
// esperado derivado del texto. Si discrepan, gana el dorado.
func TestLosDoradosPublicadosPasanContraElMotor(t *testing.T) {
	ps, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, p := range ps {
		for _, e := range corpus.EjecutarDorados(p) {
			t.Errorf("%s: %v", p.URN, e)
		}
		total += len(p.Dorados)
	}
	if total < 3 {
		t.Fatalf("el corpus publicado tiene %d dorados; el paquete ens debe traer al menos sus 3", total)
	}
}
