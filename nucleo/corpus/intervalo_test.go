package corpus

import (
	"errors"
	"strings"
	"testing"
)

// DE QUIEN ES EL NUMERO DE UNA CADENCIA.
//
// EL AGUJERO QUE ESTO CIERRA, y no era de codigo sino de acuerdo entre
// personas. "Revisar al menos una vez al ano" y "revisar a intervalos
// planificados" son la misma frase para un lector distraido y obligaciones
// OPUESTAS para un inspector: la primera pone un techo legal al intervalo y la
// segunda no pone nada. Hasta hoy la diferencia se leia en el campo `articulo`
// (`anexo, punto N` contra `ritual plazum sobre N`), o sea que vivia en la
// cabeza de quien escribia el paquete: nada impedia declarar un intervalo
// propuesto con cara de intervalo legal, y nada lo habria dicho.
//
// El momento importa y por eso se hizo antes de seguir: con tres cadencias
// escritas esto es un campo, con cuarenta es una migracion.
//
// La decision entera, con la tabla de quien puede mover que, en D-12.

func periodicaDe(origen, cita, justif, articulo string) *Paquete {
	p := enciendeElReloj(base())
	p.Obligaciones[0].Articulo = articulo
	p.Obligaciones[0].Temporalidad = &Temporalidad{
		Primitiva: "periodica", Cadencia: "P12M",
		Regimen:                   RegimenSpec{Computo: "naturales", Cierre: "fin_de_dia"},
		OrigenDelIntervalo:        origen,
		CitaDelIntervalo:          cita,
		JustificacionDelIntervalo: justif,
	}
	return p
}

const (
	citaBuena   = "RD 311/2022, art. 31.1: auditoria regular ordinaria AL MENOS CADA DOS ANOS"
	justifBuena = "Doce meses es el intervalo que espera una entidad de certificacion entre auditorias externas, y la norma no da ninguno"
)

// LAS DOS FORMAS DE LA NADA (invariante 8), que aqui son tres: campo ausente,
// cadena vacia y cadena de solo espacios. La tercera es la que se cuela cuando
// alguien "rellena" el campo para que el linter calle.
func TestUnaCadenciaSinDecirDeQuienEsSuNumeroNoCarga(t *testing.T) {
	for _, c := range []struct{ nombre, origen string }{
		{"campo ausente", ""},
		{"solo espacios", "   "},
		{"tabulador", "\t"},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			p := periodicaDe(c.origen, "", "", "31")
			if !hay(p.Validar(), ErrSinOrigenDelIntervalo) {
				t.Fatalf("una periodica que no dice de quien es su numero tiene que caerse: %v",
					p.Validar())
			}
		})
	}
}

// EL VOCABULARIO ES CERRADO Y NO TIENE RAMA POR DEFECTO. Un valor inventado no
// puede caer del lado permisivo ni del restrictivo: los dos serian elegir por
// el autor una de las dos respuestas opuestas.
func TestUnOrigenDelIntervaloInventadoNoCarga(t *testing.T) {
	p := periodicaDe("recomendado", citaBuena, "", "31")
	if !hay(p.Validar(), ErrOrigenDelIntervaloDesconocido) {
		t.Fatal("un origen fuera del vocabulario tiene que caerse")
	}
}

// El campo describe la CADENCIA de una periodica. En un plazo no hay intervalo
// del que hablar, y declararlo ahi sugiere una libertad sobre el limite que no
// existe: el plazo del art. 33 del RGPD no se "aprieta", se cumple.
func TestElOrigenDelIntervaloEnUnPlazoNoCarga(t *testing.T) {
	p := enciendeElReloj(base())
	p.Obligaciones[0].Temporalidad = &Temporalidad{
		Primitiva: "plazo", Hito: "notificacion", Limite: "PT72H",
		Regimen:            RegimenSpec{Computo: "naturales", Cierre: "exacto", Traslado: "ninguno"},
		Disparador:         map[string]string{"hecho": "conocimiento"},
		OrigenDelIntervalo: IntervaloSueloLegal,
		CitaDelIntervalo:   citaBuena,
	}
	if !hay(p.Validar(), ErrOrigenDelIntervaloFueraDeSitio) {
		t.Fatal("un plazo no tiene intervalo, asi que no puede declarar su origen")
	}
}

// Si el numero lo da la norma, hay que decir QUE articulo y CON QUE PALABRAS.
// Una cita de tres palabras no se puede contrastar sin abrir el boletin, que es
// justo lo que el proyecto promete que no hace falta.
func TestUnIntervaloDeLaNormaSinCitaUtilNoCarga(t *testing.T) {
	for _, c := range []struct{ nombre, cita string }{
		{"sin cita", ""},
		{"cita de etiqueta", "art. 31"},
		{"cita de solo espacios", strings.Repeat(" ", 80)},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			p := periodicaDe(IntervaloSueloLegal, c.cita, "", "31")
			if !hay(p.Validar(), ErrIntervaloDeLaNormaSinCita) {
				t.Fatalf("decir que el numero lo da la norma exige decir cual: %v", p.Validar())
			}
		})
	}
	// Y `fijado` recorre la misma puerta, que es la que se olvida por ser el
	// valor menos frecuente de los tres.
	p := periodicaDe(IntervaloFijado, "", "", "31")
	if !hay(p.Validar(), ErrIntervaloDeLaNormaSinCita) {
		t.Fatal("fijado tambien dice que el numero lo da la norma")
	}
}

// Si el numero lo pone plazum, hay que decir por que ESE. El suelo de
// caracteres esta para que sea un argumento y no una etiqueta.
func TestUnIntervaloPropuestoSinArgumentoNoCarga(t *testing.T) {
	for _, c := range []struct{ nombre, justif string }{
		{"sin justificacion", ""},
		{"una etiqueta", "es lo razonable"},
		{"justo por debajo del suelo", strings.Repeat("x", minimoJustificacionDelIntervalo-1)},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			p := periodicaDe(IntervaloPropuesto, "", c.justif, "ritual plazum sobre 9.2.2")
			if !hay(p.Validar(), ErrIntervaloPropuestoSinJustificacion) {
				t.Fatalf("un numero de plazum sin argumento es un numero inventado: %v", p.Validar())
			}
		})
	}
}

// LAS DOS EXPLICACIONES A LA VEZ NO SON REDUNDANCIA, SON AMBIGUEDAD. Con cita
// de articulo Y justificacion de plazum, quien lea esto se queda sin saber cual
// manda, y la lectura que hara un auditor es la optimista.
func TestUnIntervaloConCitaYJustificacionNoCarga(t *testing.T) {
	p := periodicaDe(IntervaloSueloLegal, citaBuena, justifBuena, "31")
	if !hay(p.Validar(), ErrIntervaloConLasDosExplicaciones) {
		t.Fatal("cita y justificacion a la vez no dejan saber cual manda")
	}
}

// LOS DOS CAMPOS QUE TIENEN QUE DECIR LO MISMO. `articulo` es lo que el usuario
// LEE al lado de la fecha; `origen_del_intervalo` es lo que el producto usa
// para decidir si le deja moverla. Que puedan discrepar era el agujero entero.
func TestElArticuloYElOrigenDelIntervaloNoPuedenDiscrepar(t *testing.T) {
	// Un ritual que dice que su numero lo da la norma.
	p := periodicaDe(IntervaloSueloLegal, citaBuena, "", "ritual plazum sobre 9.2.2")
	if !hay(p.Validar(), ErrIntervaloDeLaNormaEnUnRitual) {
		t.Error("un ritual es, por definicion, un intervalo que la norma no da")
	}
	// Y una obligacion transcrita que se queda el numero de plazum sin decirlo
	// donde el usuario lo lee.
	q := periodicaDe(IntervaloPropuesto, "", justifBuena, "anexo, punto 3.2.1")
	if !hay(q.Validar(), ErrIntervaloPropuestoFueraDeUnRitual) {
		t.Error("si el numero es nuestro, el articulo tiene que decirlo")
	}
}

// CONTROL NEGATIVO: las tres formas correctas cargan sin una sola queja de esta
// familia. Sin esto, un linter que dijera siempre que no tambien pasaria todos
// los tests de arriba.
func TestLasTresFormasCorrectasDelIntervaloCargan(t *testing.T) {
	casos := []struct {
		nombre string
		p      *Paquete
	}{
		{"suelo legal con su cita", periodicaDe(IntervaloSueloLegal, citaBuena, "", "31")},
		{"fijado con su cita", periodicaDe(IntervaloFijado, citaBuena, "", "31")},
		{"propuesto con su argumento en un ritual",
			periodicaDe(IntervaloPropuesto, "", justifBuena, "ritual plazum sobre 9.2.2")},
	}
	deEstaFamilia := []error{
		ErrSinOrigenDelIntervalo, ErrOrigenDelIntervaloDesconocido,
		ErrOrigenDelIntervaloFueraDeSitio, ErrIntervaloDeLaNormaSinCita,
		ErrIntervaloPropuestoSinJustificacion, ErrIntervaloConLasDosExplicaciones,
		ErrIntervaloDeLaNormaEnUnRitual, ErrIntervaloPropuestoFueraDeUnRitual,
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			for _, e := range c.p.Validar() {
				for _, cent := range deEstaFamilia {
					if errors.Is(e, cent) {
						t.Errorf("una cadencia bien declarada no puede dar %v", e)
					}
				}
			}
		})
	}
}

// Y el reverso del reverso: una obligacion SIN reloj no tiene nada que declarar.
// Si esta puerta se pusiera roja ahi, pediria un origen de intervalo a los 129
// controles de un catalogo y seria imposible de cumplir.
func TestUnaObligacionSinRelojNoDeclaraOrigenDeIntervalo(t *testing.T) {
	p := enciendeElReloj(base()) // base() no trae temporalidad
	for _, e := range p.Validar() {
		if errors.Is(e, ErrSinOrigenDelIntervalo) {
			t.Fatalf("sin reloj no hay intervalo del que decir nada: %v", e)
		}
	}
}
