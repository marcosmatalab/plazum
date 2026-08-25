package plazum

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"plazum/nucleo/corpus"
)

type derivados struct{ campos, preguntas, trazas, recursos int }

// TestElPaqueteDemoEsValidoYDeriva vigila la ayuda misma.
//
// La propiedad "anadir una norma no toca codigo" descansa entera en que
// paqueteDemo produzca un paquete que el linter acepta y del que el sistema
// deriva formularios, entrevista, entregables y conectores. Si el paquete demo
// se rompe, o si medirDerivados se traga el fallo del linter, lo unico que
// queda es una comparacion de ceros.
//
// Del barrido de mutacion: vaciar la cita de la obligacion del paquete demo
// pone rojo esto (el linter la exige) y silenciar el error de Cargar en
// medirDerivados deja de ser invisible, porque aqui se exige que lo derivado no
// sea cero.
func TestElPaqueteDemoEsValidoYDeriva(t *testing.T) {
	dir := t.TempDir()
	d := filepath.Join(dir, "norma-demo")
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	manifiesto := paqueteDemo("urn:demo:ayuda", "sistema", "categoria")
	if err := os.WriteFile(filepath.Join(d, "paquete.json"), []byte(manifiesto), 0o644); err != nil {
		t.Fatal(err)
	}
	ps, err := corpus.Cargar(dir)
	if err != nil {
		t.Fatalf("el paquete demo no pasa el linter, asi que la prueba de extensibilidad "+
			"no prueba nada: %v", err)
	}
	if len(ps) != 1 {
		t.Fatalf("esperaba 1 paquete cargado y hay %d", len(ps))
	}
	got := medirDerivados(t, dir)
	if got.campos == 0 || got.preguntas == 0 || got.trazas == 0 || got.recursos == 0 {
		t.Fatalf("un paquete demo valido tiene que derivar en las cuatro superficies "+
			"y ha derivado %+v: o el paquete no vale, o medirDerivados se esta "+
			"tragando el fallo", got)
	}
}

func medirDerivados(t *testing.T, dir string) derivados {
	t.Helper()
	ps, err := corpus.Cargar(dir)
	if err != nil {
		t.Fatal(err)
	}
	return derivados{
		campos:    len(corpus.EsquemaUI(ps)),
		preguntas: len(corpus.Entrevista(ps)),
		trazas:    len(corpus.Trazabilidad(ps)),
		recursos:  len(corpus.Conectores(ps)),
	}
}

// paqueteDemo genera un paquete valido parametrizado. El codigo del test no sabe
// que norma es: solo compone datos.
func paqueteDemo(urn, entidad, atributo string) string {
	return fmt.Sprintf(`{
  "urn": %q, "version": "1.0.0", "clase": 1,
  "licencia": "art. 13 TRLPI", "fuente": "https://www.boe.es/", "consolidado": true,
  "licencia_fuente": "boe-trlpi-13",
  "atribucion": "Texto de una disposicion legal, reproducido citando la fuente enlazada.",
  "vigencia": {"desde": "2022-05-05"},
  "entidades": [{"nombre": %q, "descripcion": "d", "atributos": [
     {"nombre": %q, "tipo": 4, "valores": ["BAJO","MEDIO","ALTO"],
      "obligado": true, "cita": "art. 1"}]}],
  "preguntas": [{"id": %q, "texto": "?", "cita": "art. 1",
     "entidad": %q, "atributo": %q, "desbloquea": [%q]}],
  "obligaciones": [{"id": %q, "articulo": "1", "clase_e2e": "observable", "texto_legal": "t",
     "cita": "art. 1", "vigencia": {"desde": "2022-05-05"},
     "entregable": %q, "recursos": [%q], "preguntas": [%q]}],
  "plantillas": [{"id": %q, "titulo": "T", "cita": "art. 1",
     "campos": [{"nombre": "c", "origen": "entidad:x.y"}]}]
}`,
		urn, entidad, atributo,
		urn+".q", entidad, atributo, urn+".o",
		urn+".o", urn+".t", entidad, urn+".q",
		urn+".t")
}
