package obligo

import (
	"fmt"
	"testing"

	"obligo/nucleo/corpus"
)

type derivados struct{ campos, preguntas, trazas, recursos int }

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
