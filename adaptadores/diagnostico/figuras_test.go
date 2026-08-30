package diagnostico

// LA COMPROBACION DEL DIA UNO del escalado, con sus dos direcciones.
//
// Lo que vigila no es que el corpus este bien — eso ya lo hace el linter — sino
// que ESTA organizacion tenga a alguien detras de cada figura. Sin ella, el
// aviso se descubre roto el dia del incidente.

import (
	"context"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/puertos"
)

// Un paquete con dos figuras y dos escalones: una definida por la norma y otra
// propuesta, que es lo que hace falta para comprobar que se piden distinto.
const paqueteConFiguras = `{
  "urn": "urn:demo:x", "version": "1.0.0", "clase": 4,
  "licencia": "Apache-2.0",
  "identificador": {"tipo": "sin-identificador", "valor": "sintetico/de/prueba",
     "motivo": "datos sinteticos de prueba, no son una norma"},
  "licencia_fuente": "del-proyecto",
  "atribucion": "Datos sinteticos de prueba, sin tercero con derechos.",
  "vigencia": {"desde": "2026-01-01"},
  "roles": [
    {"id": "demo.figura_de_la_norma", "titulo": "Figura que nombra la norma",
     "origen": "definido_por_la_norma",
     "cita": "Norma sintetica de prueba, art. 1.2, letra a). Verificado el 30-08-2026."},
    {"id": "demo.figura_propuesta", "titulo": "Figura que propone plazum",
     "origen": "propuesto",
     "justificacion": "La norma sintetica no nombra a nadie, asi que plazum sugiere quien suele tenerla."}
  ],
  "obligaciones": [
    {"id": "demo.uno", "articulo": "1", "cita": "sintetico",
     "vigencia": {"desde": "2026-01-01"}, "clase_e2e": "documental",
     "escalado": [{"tras": "P30D_antes", "a": "demo.figura_de_la_norma"},
                  {"tras": "P5D", "a": "demo.figura_propuesta"}]}
  ]
}`

func TestElDoctorAvisaElDiaUnoDeLasFigurasQueNadieOcupa(t *testing.T) {
	t.Run("sin alcance: faltan todas y se dice cuantos avisos se pierden", func(t *testing.T) {
		o := opcionesSanas(t)
		o.Corpus = escribirCorpus(t, paqueteConFiguras)
		c := por(Nuevo(o).Comprobar(context.Background()), "figuras")
		if c.Estado != puertos.Aviso {
			t.Fatalf("sin nadie asignado tenia que avisar y salio %s: %s", c.Estado, c.Detalle)
		}
		// El numero de avisos que se pierden es LO QUE HACE ACCIONABLE la linea:
		// "faltan dos figuras" no mueve a nadie; "dos avisos no llegarian", si.
		if !strings.Contains(c.Detalle, "2 aviso(s) de escalado sin destinatario") {
			t.Errorf("el detalle no dice cuantos avisos se pierden: %s", c.Detalle)
		}
		// Y las dos clases de figura se piden distinto, porque no cuestan igual.
		if !strings.Contains(c.Detalle, "la nombra la norma") ||
			!strings.Contains(c.Detalle, "puedes cambiarla") {
			t.Errorf("el detalle no distingue la figura de la norma de la propuesta: %s", c.Detalle)
		}
		// LA DOCTRINA DEL FALSO POSITIVO: esto no acusa de nada.
		if !strings.Contains(c.Arreglo, "NO dice que hayas incumplido") {
			t.Errorf("el arreglo no lleva el descargo: %s", c.Arreglo)
		}
		for _, mala := range []string{"incumpl", "infracc", "sancion"} {
			if strings.Contains(strings.ToLower(c.Detalle), mala) {
				t.Errorf("el detalle dice %q: %s", mala, c.Detalle)
			}
		}
	})

	t.Run("con media asignada: solo falta la otra", func(t *testing.T) {
		o := opcionesSanas(t)
		o.Corpus = escribirCorpus(t, paqueteConFiguras)
		o.Alcance = "/ruta/del/alcance.json"
		o.Figuras = map[string]string{"demo.figura_de_la_norma": "ana"}
		c := por(Nuevo(o).Comprobar(context.Background()), "figuras")
		if c.Estado != puertos.Aviso {
			t.Fatalf("con una figura sin ocupar tenia que avisar y salio %s", c.Estado)
		}
		if !strings.Contains(c.Detalle, "1 de 2 figuras") {
			t.Errorf("el recuento no cuadra: %s", c.Detalle)
		}
		// Y dice DONDE mirar, que es el alcance que le pasaron.
		if !strings.Contains(c.Detalle, "/ruta/del/alcance.json") {
			t.Errorf("el detalle no dice de que alcance sale: %s", c.Detalle)
		}
	})

	// EL CONTROL POSITIVO, y sin el los dos de arriba no demuestran nada: una
	// comprobacion que avisara siempre daria exactamente el mismo verde.
	t.Run("con todas ocupadas: correcto", func(t *testing.T) {
		o := opcionesSanas(t)
		o.Corpus = escribirCorpus(t, paqueteConFiguras)
		o.Figuras = map[string]string{
			"demo.figura_de_la_norma": "ana", "demo.figura_propuesta": "beatriz",
		}
		c := por(Nuevo(o).Comprobar(context.Background()), "figuras")
		if c.Estado != puertos.Correcto {
			t.Fatalf("con todas ocupadas tenia que salir correcto y salio %s: %s",
				c.Estado, c.Detalle)
		}
	})

	// Un corpus sin figuras no es un problema: es un corpus que no escala.
	t.Run("un corpus sin figuras no inventa un problema", func(t *testing.T) {
		o := opcionesSanas(t)
		o.Corpus = escribirCorpus(t, paqueteSano)
		c := por(Nuevo(o).Comprobar(context.Background()), "figuras")
		if c.Estado != puertos.Correcto {
			t.Fatalf("un corpus sin figuras salio %s: %s", c.Estado, c.Detalle)
		}
	})
}
