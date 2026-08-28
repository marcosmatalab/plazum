package corpus

import (
	"errors"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/estricto"
)

// Una fuente del intervalo tiene que identificar un documento.
//
// EL PROBLEMA QUE ESTE CAMPO RESUELVE NO ES LINTABLE, y conviene decirlo antes
// que nada: el apoyo fantasma es un argumento que se apoya en algo que no
// existe, y suena exactamente igual que uno que se apoya en algo real. Ningun
// detector distingue "el sector lleva anos exigiendo revisar esto cada seis
// meses" de un razonamiento propio.
//
// Lo que el campo cambia es LA PREGUNTA. Con `fuentes_del_intervalo`, la pasada
// de coherencia no lee cada frase buscando un eco: pregunta una sola cosa,
// "¿por que este argumento no tiene fuente?", y eso se contesta mirando.
//
// Y lo poco que si es mecanico se comprueba aqui: que lo escrito IDENTIFIQUE
// algo. Los dos apoyos fantasma que salieron en la pasada de cierre de las 34
// (una curva de decaimiento de tasa de clic sin origen, una guia de fabricante
// sin decir cual) caen los dos por esta regla, porque ninguno de los dos nombra
// un documento.
func TestUnaFuenteDelIntervaloIdentificaUnDocumento(t *testing.T) {
	malas := []struct{ fuente, porque string }{
		{"", "una entrada vacia cuenta como apoyo y no apoya nada"},
		{"   ", "espacios en blanco, lo mismo"},
		{"NIST", "es una organizacion, no un documento"},
		{"ENISA", "igual"},
		{"la guia del fabricante", "no dice de que fabricante ni que guia: es el apoyo " +
			"fantasma del punto 6.3.3 tal cual"},
		{"estudios del sector sobre tasa de clic", "es el apoyo fantasma del punto 8.1.3: " +
			"suena a dato y no se puede ir a buscar"},
		{"practica habitual de la industria", "el apoyo fantasma en su forma pura"},
	}
	for _, m := range malas {
		if porque := fuenteVagaPorque(m.fuente); porque == "" {
			t.Errorf("fuenteVagaPorque(%q) la da por buena, y %s", m.fuente, m.porque)
		}
	}

	// Las que valen. Sin este tramo, un detector que rechazara todo pasaria el
	// de arriba con nota.
	buenas := []string{
		"NIST SP 800-92, seccion 4.3",
		"Reglamento (UE) 2024/2690, anexo, punto 2.2.3",
		"ENISA Threat Landscape 2025, capitulo de ransomware",
		"Real Decreto 311/2022, anexo II, medida op.exp.8",
		"https://www.first.org/cvss/v4.0/specification-document",
	}
	for _, b := range buenas {
		if porque := fuenteVagaPorque(b); porque != "" {
			t.Errorf("fuenteVagaPorque(%q) la rechaza y es una fuente legitima: %s", b, porque)
		}
	}
}

// Y el linter la rechaza AL CARGAR, en los tres origenes del intervalo.
//
// Se recorren los tres a proposito. El caso natural es `propuesto`, que es
// donde un apoyo fantasma sostiene un numero nuestro; dejar sin mirar los otros
// dos seria dejar abierto el camino que nadie recorre, que es siempre el que se
// acaba usando.
func TestElLinterRechazaUnaFuenteVagaEnLosTresOrigenes(t *testing.T) {
	casos := []struct{ nombre, temporalidad string }{
		{
			nombre: "propuesto",
			temporalidad: `"origen_del_intervalo":"propuesto",
        "justificacion_del_intervalo":"` + strings.Repeat("j", 80) + `",
        "cuando_cambiarlo":"` + strings.Repeat("c", 140) + `",`,
		},
		{
			nombre: "suelo_legal",
			temporalidad: `"origen_del_intervalo":"suelo_legal",
        "cita_del_intervalo":"` + strings.Repeat("k", 60) + `",`,
		},
		{
			nombre: "fijado",
			temporalidad: `"origen_del_intervalo":"fijado",
        "cita_del_intervalo":"` + strings.Repeat("k", 60) + `",`,
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			articulo := "art. 1"
			if c.nombre == "propuesto" {
				// Un intervalo propuesto tiene que vivir en un ritual de
				// plazum; si no, salta OTRO error y este test estaria
				// midiendo aquel.
				articulo = "ritual plazum sobre el art. 1"
			}
			crudo := `{
  "urn":"x@1","version":"1","clase":4,
  "identificador":{"tipo":"eli","valor":"http://data.europa.eu/eli/reg/2024/2847/oj"},
  "obligaciones":[{
    "id":"a","cita":"art. 1","articulo":"` + articulo + `","clase_e2e":"documental",
    "texto_legal":"texto",
    "temporalidad":{"primitiva":"periodica","hito":"h","cadencia":"P12M",
      "regimen":{"computo":"naturales","cierre":"fin_de_dia","traslado":"ninguno"},
      "disparador":{"hecho":"ultima_h"},
      ` + c.temporalidad + `
      "fuentes_del_intervalo":["la guia del fabricante"]}}]}`
			// Se valida el paquete decodificado y se recorren TODOS sus
			// errores. Cargar() solo devuelve el primero, y un paquete de
			// juguete arrastra otros fallos de linter (licencia_fuente,
			// atribucion) que no son de lo que va este test: mirar solo el
			// primero seria medir el orden de las comprobaciones.
			var p Paquete
			if err := estricto.Decodificar([]byte(crudo), &p, "el paquete de prueba"); err != nil {
				t.Fatal(err)
			}
			errs := p.Validar()
			vista := false
			for _, e := range errs {
				if errors.Is(e, ErrFuenteDelIntervaloVaga) {
					vista = true
				}
			}
			if !vista {
				t.Fatalf("el linter da por buena una fuente que no identifica ningun "+
					"documento. Los %d fallos que si vio: %v", len(errs), errs)
			}
		})
	}
}
