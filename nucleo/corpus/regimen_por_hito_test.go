package corpus

import (
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// EL REGIMEN POR HITO.
//
// Una notificacion escalonada mezcla plazos de dos naturalezas dentro de la
// MISMA obligacion: el art. 14 del CRA da 24 y 72 HORAS para las dos primeras
// notificaciones y un MES para el informe final. Y el regimen no es el mismo:
// el art. 3.2.b del Reglamento 1182/71 hace terminar el plazo en dias o meses
// al expirar la ultima hora del ultimo dia, y el 3.4 lo traslada al habil
// siguiente si cae en inhabil, "expresado de cualquier modo, SALVO EN HORAS".
//
// Con un solo regimen por obligacion habia que elegir entre partir la obligacion
// (y entonces el informe final no puede encadenarse al hito del que cuelga,
// porque desde_hito no cruza obligaciones) o darle a los meses el regimen de las
// horas, que produce una fecha mas temprana que la legal. Temprana es el lado
// inofensivo, pero sigue siendo una fecha equivocada.

func TestCadaHitoPuedeTraerSuPropioRegimen(t *testing.T) {
	o := Obligacion{
		ID: "demo.escalonada", ClaseE2E: "notificatoria",
		Vigencia: Vigencia{Desde: "2026-01-01"},
		Temporalidad: &Temporalidad{
			Primitiva: "plazo",
			// El regimen de la obligacion es el de las HORAS.
			Regimen:    RegimenSpec{Computo: "naturales", Cierre: "exacto", Traslado: "ninguno"},
			Disparador: map[string]string{"hecho": "conocimiento"},
			Hitos: []HitoSpec{
				{ID: "alerta", Limite: "PT24H"},
				{ID: "informe_final", Limite: "P14D",
					// ...y este hito trae el suyo, el de los DIAS.
					Regimen: &RegimenSpec{Computo: "naturales", Cierre: "fin_de_dia",
						Traslado: "siguiente_habil"}},
			},
		},
	}
	// Viernes 4 de septiembre de 2026 a las 09:00 UTC.
	hechos := ventana.Hechos{"conocimiento": time.Date(2026, 9, 4, 9, 0, 0, 0, time.UTC)}
	vs, err := VencimientosDe(o, hechos, time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	quiero := map[string]string{
		// 24 horas exactas, de hora a hora, sabado incluido: los inhabiles no
		// alcanzan a los plazos en horas.
		"alerta": "2026-09-05T09:00:00Z",
		// 14 dias naturales caen el viernes 18, al final del dia.
		"informe_final": "2026-09-18T23:59:59Z",
	}
	visto := map[string]bool{}
	for _, v := range vs {
		esperado, ok := quiero[v.Hito]
		if !ok {
			t.Errorf("hito inesperado: %s", v.Hito)
			continue
		}
		visto[v.Hito] = true
		if got := v.Vence.UTC().Format(time.RFC3339); got != esperado {
			t.Errorf("hito %s vence %s y tenia que vencer %s.\n"+
				"  Si los dos hitos salen con el mismo cierre, el regimen del hito no se esta "+
				"aplicando y el de la obligacion se lo esta comiendo.", v.Hito, got, esperado)
		}
	}
	for h := range quiero {
		if !visto[h] {
			t.Errorf("no ha salido el hito %s", h)
		}
	}
}

// CONTROL NEGATIVO: sin regimen propio, el hito hereda el de la obligacion. Si
// esto fallara, el campo nuevo habria cambiado el calculo de los relojes que ya
// estaban escritos sin declararlo.
func TestUnHitoSinRegimenPropioHeredaElDeLaObligacion(t *testing.T) {
	o := Obligacion{
		ID: "demo.simple", ClaseE2E: "notificatoria",
		Vigencia: Vigencia{Desde: "2026-01-01"},
		Temporalidad: &Temporalidad{
			Primitiva:  "plazo",
			Regimen:    RegimenSpec{Computo: "naturales", Cierre: "fin_de_dia", Traslado: "ninguno"},
			Disparador: map[string]string{"hecho": "conocimiento"},
			Hitos:      []HitoSpec{{ID: "informe", Limite: "P14D"}},
		},
	}
	hechos := ventana.Hechos{"conocimiento": time.Date(2026, 9, 4, 9, 0, 0, 0, time.UTC)}
	vs, err := VencimientosDe(o, hechos, time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 {
		t.Fatalf("se esperaba un vencimiento y hay %d", len(vs))
	}
	if got := vs[0].Vence.UTC().Format(time.RFC3339); got != "2026-09-18T23:59:59Z" {
		t.Errorf("vence %s: el hito no ha heredado el cierre a fin de dia de la obligacion", got)
	}
}
