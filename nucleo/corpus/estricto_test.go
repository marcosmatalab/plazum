package corpus

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/estricto"
)

// Un campo que el formato no declara PARA la carga del corpus.
//
// EL CASO ES REAL Y ES DE HACE UNAS HORAS. Las 34 cadencias del anexo de
// 2024/2690 se escribieron con un campo `cuando_cambiarlo` que todavia no
// existia en `Temporalidad`. Con el decodificador por defecto habrian cargado
// las 34, el linter habria dado verde, las puertas habrian dado verde, y el
// unico dato que un CISO usa para adaptar el numero a su casa se habria perdido
// sin una linea de aviso. Lo vio una pasada de cierre que fue a leer el
// esquema: o sea, la suerte.
//
// Se prueban las dos alturas del fichero, porque son dos ramas distintas del
// decodificador: un campo inventado en la raiz del paquete y otro dentro de la
// temporalidad de una obligacion, que es donde ocurrio de verdad.
func TestCargarRechazaUnCampoQueElFormatoNoDeclara(t *testing.T) {
	casos := []struct{ nombre, json, campo string }{
		{
			nombre: "en la raiz del paquete",
			campo:  "verssion",
			json: `{
  "urn":"x@1","version":"1","verssion":"1","clase":4,
  "identificador":{"tipo":"eli","valor":"http://data.europa.eu/eli/reg/2024/2847/oj"},
  "obligaciones":[]}`,
		},
		{
			// EL CASO DE VERDAD: una letra menos en cuando_cambiarlo, dentro de
			// la temporalidad de una obligacion.
			nombre: "dentro de la temporalidad de una obligacion",
			campo:  "cuando_cambiarl",
			json: `{
  "urn":"x@1","version":"1","clase":4,
  "identificador":{"tipo":"eli","valor":"http://data.europa.eu/eli/reg/2024/2847/oj"},
  "obligaciones":[{
    "id":"a","cita":"art. 1","clase_e2e":"documental","texto_legal":"texto",
    "temporalidad":{"primitiva":"periodica","hito":"h","cadencia":"P12M",
      "regimen":{"computo":"naturales","cierre":"fin_de_dia","traslado":"ninguno"},
      "disparador":{"hecho":"ultima_h"},
      "cuando_cambiarl":"esto se descartaria en silencio"}}]}`,
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			dir := t.TempDir()
			escribirPaquete(t, dir, "conCampoInventado", c.json)
			_, err := Cargar(dir)
			if err == nil {
				t.Fatalf("el paquete ha cargado con %q dentro. Ese dato no llega a ninguna "+
					"parte y nadie se entera: es indistinguible de no haberlo escrito", c.campo)
			}
			if !errors.Is(err, estricto.ErrCampoDesconocido) {
				t.Fatalf("ha fallado por otra cosa, no por el campo desconocido: %v", err)
			}
			if !strings.Contains(err.Error(), c.campo) {
				t.Errorf("el error no NOMBRA el campo: %v", err)
			}
			// Y dice en que fichero, que con 37 paquetes en el corpus es la
			// mitad de lo que hace falta para arreglarlo.
			if !strings.Contains(err.Error(), "conCampoInventado") {
				t.Errorf("el error no dice EN QUE PAQUETE: %v", err)
			}
		})
	}
}

// Lo mismo en un caso dorado, que es la otra puerta de entrada de datos escritos
// a mano en un paquete.
func TestCargarRechazaUnCampoInventadoEnUnDorado(t *testing.T) {
	dir := t.TempDir()
	escribirPaquete(t, dir, "condorado", `{
  "urn":"x@1","version":"1","clase":4,
  "identificador":{"tipo":"eli","valor":"http://data.europa.eu/eli/reg/2024/2847/oj"},
  "obligaciones":[]}`)
	if err := os.MkdirAll(dir+"/condorado/pruebas", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/condorado/pruebas/caso.json",
		[]byte(`{"caso":"c","obligacion":"a","hechos":{},"esperado":[],"cita_del_esperadoo":"x"}`),
		0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Cargar(dir)
	if err == nil {
		t.Fatal("el dorado ha cargado con un campo inventado dentro")
	}
	if !errors.Is(err, estricto.ErrCampoDesconocido) {
		t.Fatalf("ha fallado por otra cosa: %v", err)
	}
	if !strings.Contains(err.Error(), "cita_del_esperadoo") {
		t.Errorf("el error no nombra el campo: %v", err)
	}
}

// CONTROL: el corpus publicado carga. Sin esto, un decodificador que rechazara
// todo pasaria los dos tests de arriba con nota, y el linter de paquetes
// (TestTodosLosPaquetesPublicadosPasanElLinter) es quien lo vigila de verdad,
// pero vive en la raiz y no se ejecuta con este paquete.
func TestElCorpusPublicadoCargaConDecodificacionEstricta(t *testing.T) {
	ps, err := Cargar("../../paquetes")
	if err != nil {
		t.Fatalf("el corpus publicado ha dejado de cargar: %v", err)
	}
	if len(ps) < 30 {
		t.Fatalf("solo han cargado %d paquetes: o el corpus ha adelgazado, o esta "+
			"comprobacion esta mirando otro directorio", len(ps))
	}
}
