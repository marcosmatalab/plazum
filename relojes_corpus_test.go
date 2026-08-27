package plazum

import (
	"sort"
	"testing"
	"time"

	"plazum/nucleo/corpus"
	"plazum/nucleo/ventana"
)

// LA DIRECCION QUE UN DORADO NO SABE EXPRESAR.
//
// Un caso dorado dice "con estos hechos, este hito vence en esta fecha", y el
// ejecutor lo comprueba filtrando por el hito esperado. O sea que un dorado dice
// lo que TIENE que salir y no dice nada de lo que NO tiene que salir.
//
// Eso deja fuera media familia de fallos, y es la misma de siempre (invariante
// 7): cuando una comprobacion recorre una lista para contrastarla con otra, la
// direccion que falta es la que muerde. Aqui muerde asi: si al hito del plazo
// general se le olvida la clase, ese hito rige SIEMPRE, y un incidente con
// fallecimiento le ensena al operador dos fechas para la misma obligacion, la
// del art. 73.4 y la del 73.2, sin ninguna forma de saber cual es la suya. Los
// nueve dorados del paquete siguen en verde, porque cada uno mira su hito.
//
// Se demostro: quitandole la clase a notificacion_fallecimiento, los dorados NO
// se ponen rojos. Este fichero es lo que faltaba.
//
// Lo que se comprueba es una propiedad del TEXTO, no del motor: cuando un
// articulo dice "no obstante lo dispuesto en el apartado 2", el plazo del
// apartado 2 DESAPARECE, no convive.

// escenarioDeClase es un incidente clasificado y lo que la norma dice que tiene
// que salir: el conjunto EXACTO de hitos con fecha.
type escenarioDeClase struct {
	Nombre     string
	Paquete    string // URN
	Obligacion string
	Hechos     map[string]string
	// Determinados es el conjunto EXACTO de hitos que pueden quedar con fecha.
	// Ni uno mas ni uno menos: el de mas es el que muerde (dos fechas para la
	// misma obligacion) y el de menos esconde una obligacion. Vacio = ninguno,
	// que es lo que tiene que pasar cuando falta la clasificacion que decide el
	// limite.
	Determinados []string
	PorQue       string // el articulo que desplaza a los demas
}

var escenariosDeClase = []escenarioDeClase{
	{
		Nombre:     "ai-act: el fallecimiento desplaza al plazo general del art. 73.2",
		Paquete:    "urn:eu:reg:2024:1689",
		Obligacion: "aiact.art73.notificacion_de_incidente_grave",
		Hechos: map[string]string{
			"conocimiento_incidente_grave": "2026-09-01T14:00:00Z",
			"incidente_fallecimiento":      "2026-09-01T15:00:00Z",
		},
		Determinados: []string{"notificacion_fallecimiento"},
		PorQue: "art. 73.4: \"No obstante lo dispuesto en el apartado 2, en caso de fallecimiento " +
			"de una persona...\". Los quince dias del 73.2 no conviven con los diez del 73.4",
	},
	{
		Nombre:     "ai-act: la infraccion generalizada desplaza al plazo general del art. 73.2",
		Paquete:    "urn:eu:reg:2024:1689",
		Obligacion: "aiact.art73.notificacion_de_incidente_grave",
		Hechos: map[string]string{
			"conocimiento_incidente_grave":      "2026-09-01T14:00:00Z",
			"incidente_infraccion_generalizada": "2026-09-01T15:00:00Z",
		},
		Determinados: []string{"notificacion_infraccion_generalizada"},
		PorQue: "art. 73.3: \"No obstante lo dispuesto en el apartado 2 del presente articulo...\". " +
			"Dos dias, y los quince del 73.2 no salen",
	},
	{
		Nombre:     "mdr: la muerte desplaza al plazo general del art. 87.3",
		Paquete:    "urn:eu:reg:2017:745",
		Obligacion: "mdr.art87.notificacion_de_incidente_grave",
		Hechos: map[string]string{
			"conocimiento_incidente_grave": "2026-09-01T14:00:00Z",
			"muerte_o_deterioro_grave":     "2026-09-01T15:00:00Z",
		},
		Determinados: []string{"notificacion_muerte_o_deterioro"},
		PorQue: "art. 87.5: \"No obstante lo dispuesto en el apartado 3, en caso de muerte o " +
			"deterioro grave imprevisto...\". Diez dias, y los quince del 87.3 no salen",
	},
	{
		Nombre:     "mdr: la amenaza para la salud publica desplaza al plazo general",
		Paquete:    "urn:eu:reg:2017:745",
		Obligacion: "mdr.art87.notificacion_de_incidente_grave",
		Hechos: map[string]string{
			"conocimiento_incidente_grave": "2026-09-01T14:00:00Z",
			"amenaza_grave_salud_publica":  "2026-09-01T15:00:00Z",
		},
		Determinados: []string{"notificacion_amenaza_salud_publica"},
		PorQue:       "art. 87.4, con la misma formula: dos dias, y los quince no conviven con ellos",
	},
	{
		Nombre:     "nis2: un prestador de confianza no ve tambien el plazo de 72 horas",
		Paquete:    "urn:eu:dir:2022:2555",
		Obligacion: "nis2.art23_4.notificacion_de_incidente_significativo",
		Hechos: map[string]string{
			"constancia_incidente_significativo":  "2026-09-01T14:00:00Z",
			"prestador_de_servicios_de_confianza": "2026-01-01T00:00:00Z",
		},
		// La alerta temprana NO lleva clase (24 horas para todos, art. 23.4.a), asi
		// que sale con fecha y TIENE que salir. Lo que no puede salir es la
		// notificacion de 72 horas ni el informe final que cuelga de ella.
		Determinados: []string{"alerta_temprana", "notificacion_incidente_prestador_de_confianza"},
		PorQue: "art. 23.4, parrafo segundo: para un prestador de confianza la notificacion es de " +
			"24 horas, no de 72, y las dos no conviven",
	},
	{
		Nombre:     "ai-act: sin clasificar no se calla, pero tampoco se inventa una fecha",
		Paquete:    "urn:eu:reg:2024:1689",
		Obligacion: "aiact.art73.notificacion_de_incidente_grave",
		Hechos: map[string]string{
			"conocimiento_incidente_grave": "2026-09-01T14:00:00Z",
		},
		Determinados: nil, // los tres esperan a que el obligado clasifique
		PorQue: "arts. 73.2, 73.3 y 73.4: el limite lo decide como se clasifique el incidente, " +
			"y clasificar lo hace el obligado. Una lista vacia se leeria como \"nada que hacer\"",
	},
}

func obligacionDelCorpus(t *testing.T, urn, id string) corpus.Obligacion {
	t.Helper()
	ps, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatalf("cargar el corpus: %v", err)
	}
	for _, p := range ps {
		if p.URN != urn {
			continue
		}
		for _, o := range p.Obligaciones {
			if o.ID == id {
				return o
			}
		}
		t.Fatalf("el paquete %s no tiene la obligacion %s", urn, id)
	}
	t.Fatalf("el corpus publicado no tiene el paquete %s", urn)
	return corpus.Obligacion{}
}

func TestUnPlazoDesplazadoPorLaClaseNoConviveConElQueDesplaza(t *testing.T) {
	if len(escenariosDeClase) == 0 {
		t.Fatal("no hay escenarios: este test estaria dando verde sin comprobar nada")
	}
	for _, e := range escenariosDeClase {
		t.Run(e.Nombre, func(t *testing.T) {
			o := obligacionDelCorpus(t, e.Paquete, e.Obligacion)
			hechos := ventana.Hechos{}
			for k, v := range e.Hechos {
				x, err := time.Parse(time.RFC3339, v)
				if err != nil {
					t.Fatalf("hecho %q: %v", k, err)
				}
				hechos[k] = x
			}
			// El horizonte no acota un plazo, pero se pasa uno holgado para no
			// depender de un valor cero que manana signifique otra cosa.
			vs, err := corpus.VencimientosDe(o, hechos, time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
			if err != nil {
				t.Fatalf("%v", err)
			}
			if len(vs) == 0 {
				t.Fatal("el motor no ha devuelto ningun vencimiento: sin clasificar tampoco se " +
					"calla, porque una lista vacia se lee como \"nada que hacer\"")
			}
			var determinados []string
			for _, v := range vs {
				if v.Estado == ventana.Determinado {
					determinados = append(determinados, v.Hito)
				}
			}
			sort.Strings(determinados)
			quiero := append([]string(nil), e.Determinados...)
			sort.Strings(quiero)

			// LAS DOS DIRECCIONES, y la segunda es la que un dorado no sabe
			// decir: que SOBRE un hito con fecha.
			esperado := map[string]bool{}
			for _, h := range quiero {
				esperado[h] = true
			}
			visto := map[string]bool{}
			for _, h := range determinados {
				visto[h] = true
				if !esperado[h] {
					t.Errorf("el hito %q ha quedado con fecha y NO puede. %s\n"+
						"  Dos fechas para la misma obligacion es peor que ninguna: el operador no "+
						"tiene forma de saber cual es la suya.\n"+
						"  Con fecha: %v. Solo podian: %v", h, e.PorQue, determinados, quiero)
				}
			}
			for _, h := range quiero {
				if !visto[h] {
					t.Errorf("el hito %q tenia que quedar con fecha y no ha salido. %s\n"+
						"  Con fecha: %v", h, e.PorQue, determinados)
				}
			}
		})
	}
}
