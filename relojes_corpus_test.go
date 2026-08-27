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
// que salir: exactamente un vencimiento con fecha, y cual.
type escenarioDeClase struct {
	Nombre     string
	Paquete    string // URN
	Obligacion string
	Hechos     map[string]string
	SoloHito   string // el unico hito que puede quedar Determinado
	PorQue     string // el articulo que desplaza a los demas
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
		SoloHito: "notificacion_fallecimiento",
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
		SoloHito: "notificacion_infraccion_generalizada",
		PorQue: "art. 73.3: \"No obstante lo dispuesto en el apartado 2 del presente articulo...\". " +
			"Dos dias, y los quince del 73.2 no salen",
	},
	{
		Nombre:     "ai-act: sin clasificar no se calla, pero tampoco se inventa una fecha",
		Paquete:    "urn:eu:reg:2024:1689",
		Obligacion: "aiact.art73.notificacion_de_incidente_grave",
		Hechos: map[string]string{
			"conocimiento_incidente_grave": "2026-09-01T14:00:00Z",
		},
		SoloHito: "", // ninguno determinado: los tres esperan a que el obligado clasifique
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
			switch {
			case e.SoloHito == "" && len(determinados) != 0:
				t.Errorf("con el incidente SIN clasificar el motor da fecha a %v. El limite lo "+
					"decide la clasificacion (%s), asi que dar una fecha aqui es inventarsela",
					determinados, e.PorQue)
			case e.SoloHito != "" && len(determinados) != 1:
				t.Errorf("han quedado %d hitos con fecha (%v) y solo puede quedar uno, %q.\n"+
					"  %s\n"+
					"  Dos fechas para la misma obligacion es peor que ninguna: el operador no "+
					"tiene forma de saber cual es la suya.",
					len(determinados), determinados, e.SoloHito, e.PorQue)
			case e.SoloHito != "" && determinados[0] != e.SoloHito:
				t.Errorf("el hito con fecha es %q y tenia que ser %q. %s",
					determinados[0], e.SoloHito, e.PorQue)
			}
		})
	}
}
