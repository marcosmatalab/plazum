package pantalla

import (
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// periodicaDe construye un paquete con una obligacion periodica anclada en un
// hecho, para poder fabricar incumplimientos de laboratorio.
func periodicaDe(id, art, hecho, cadencia, vigenteDesde string) corpus.Obligacion {
	return corpus.Obligacion{
		ID: id, Articulo: art, Titulo: id, Cita: "c", ClaseE2E: "documental",
		TextoLegal: "texto",
		Vigencia:   corpus.Vigencia{Desde: vigenteDesde},
		Temporalidad: &corpus.Temporalidad{
			Primitiva: "periodica", Hito: "h", Cadencia: cadencia,
			Regimen:    corpus.RegimenSpec{Computo: "naturales", Cierre: "fin_de_dia", Traslado: "ninguno"},
			Disparador: map[string]string{"hecho": hecho},
		},
	}
}

// Lo que YA VENCIO sale con fila, no con un numero bajo la etiqueta del futuro.
//
// EL FALLO: la derivacion metia en el mismo cubo lo pasado y lo posterior a la
// ventana, y el cubo se imprimia como «fechas mas alla de los doce meses». Un
// incumplimiento en curso salia contado como algo del futuro lejano.
//
// Es la fila mas importante que un producto de continuidad de cumplimiento
// puede imprimir, y era la unica que no se imprimia.
func TestLoQueYaVencioSaleConFilaYNoComoFuturo(t *testing.T) {
	ahora := time.Date(2026, 8, 27, 9, 0, 0, 0, time.UTC)
	ps := []*corpus.Paquete{paqueteConRelojes("urn:demo:v",
		periodicaDe("v.vieja", "1", "x", "P12M", "2020-01-01"),
	)}
	// Ultima vez: hace cuatro anos y medio. Con ciclo anual, cuatro
	// vencimientos pasados y una sola noticia.
	cal := Derivar12Meses(ps, TodoAplica,
		ventana.Hechos{"x": time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)}, ahora)

	if len(cal.Vencidas) != 1 {
		t.Fatalf("%d obligaciones vencidas, esperaba 1: cuatro anos de incumplimiento anual "+
			"son cuatro vencimientos y UNA noticia", len(cal.Vencidas))
	}
	v := cal.Vencidas[0]
	if v.Ciclos != 4 {
		t.Errorf("%d ciclos, esperaba 4", v.Ciclos)
	}
	// EL MAS ANTIGUO, que es el que contesta "¿desde cuando?".
	quiero := time.Date(2023, 1, 15, 23, 59, 59, 0, time.UTC)
	if !v.Desde.Equal(quiero) {
		t.Errorf("la fila dice que vencio el %s y el mas antiguo es el %s. Se ensena el mas "+
			"antiguo porque es el que contesta la pregunta de un inspector",
			v.Desde.Format("2006-01-02"), quiero.Format("2006-01-02"))
	}
	if cal.VencimientosPasados != 4 {
		t.Errorf("%d vencimientos pasados, esperaba 4", cal.VencimientosPasados)
	}
	// Y NO se cuentan como futuro: ese era el fallo.
	if cal.MasAllaDeLaVentana != 0 {
		t.Errorf("%d contados como posteriores a la ventana, y estan todos DETRAS",
			cal.MasAllaDeLaVentana)
	}
}

// EL PEOR FALSO POSITIVO DE ESA PANTALLA, y salio en su primera ejecucion.
//
// Una cadencia se ancla en un hecho del operador ("la ultima vez que lo hice"),
// y ese hecho puede ser MUY anterior a la norma. Quien reviso su politica en
// 2022 no incumplia en 2023 un reglamento que entro en vigor en noviembre de
// 2024. Acusar de un incumplimiento imposible destruye la confianza en la
// pantalla entera mas deprisa que no ensenarla.
func TestNoSeAcusaDeIncumplirAntesDeQueLaNormaObligara(t *testing.T) {
	ahora := time.Date(2026, 8, 27, 9, 0, 0, 0, time.UTC)
	ps := []*corpus.Paquete{paqueteConRelojes("urn:demo:vig",
		periodicaDe("vig.nueva", "1", "x", "P12M", "2024-11-07"),
	)}
	cal := Derivar12Meses(ps, TodoAplica,
		ventana.Hechos{"x": time.Date(2022, 1, 15, 0, 0, 0, 0, time.UTC)}, ahora)

	// Ocurrencias del ciclo: 2023, 2024, 2025 y 2026. Las dos primeras caen
	// ANTES del 2024-11-07 y no son incumplimientos de nada.
	if cal.VencimientosAntesDeLaVigencia != 2 {
		t.Errorf("%d ocurrencias anteriores a la vigencia, esperaba 2",
			cal.VencimientosAntesDeLaVigencia)
	}
	if cal.VencimientosPasados != 2 {
		t.Errorf("%d vencimientos pasados, esperaba 2 (2025 y 2026)", cal.VencimientosPasados)
	}
	if len(cal.Vencidas) != 1 {
		t.Fatalf("%d filas de vencidas, esperaba 1", len(cal.Vencidas))
	}
	quiero := time.Date(2025, 1, 15, 23, 59, 59, 0, time.UTC)
	if !cal.Vencidas[0].Desde.Equal(quiero) {
		t.Errorf(`la fila dice que vencio el %s, y la norma no obligaba hasta el 2024-11-07.

  Acusar de un incumplimiento anterior a la entrada en vigor es el peor falso
  positivo que puede dar esta pantalla: quien lo lea deja de creerse el resto.`,
			cal.Vencidas[0].Desde.Format("2006-01-02"))
	}
}

// Las vencidas salen de la mas antigua a la mas reciente: es el orden en que le
// pesan a quien tiene que responder por ellas.
func TestLasVencidasSalenDeLaMasAntiguaALaMasReciente(t *testing.T) {
	ahora := time.Date(2026, 8, 27, 9, 0, 0, 0, time.UTC)
	ps := []*corpus.Paquete{paqueteConRelojes("urn:demo:ord",
		periodicaDe("ord.reciente", "1", "reciente", "P12M", "2020-01-01"),
		periodicaDe("ord.antigua", "2", "antigua", "P12M", "2020-01-01"),
	)}
	cal := Derivar12Meses(ps, TodoAplica, ventana.Hechos{
		"reciente": time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
		"antigua":  time.Date(2021, 3, 1, 0, 0, 0, 0, time.UTC),
	}, ahora)

	if len(cal.Vencidas) != 2 {
		t.Fatalf("%d vencidas, esperaba 2", len(cal.Vencidas))
	}
	if cal.Vencidas[0].Obligacion != "ord.antigua" {
		t.Errorf("la primera es %q y tenia que ser la mas antigua",
			cal.Vencidas[0].Obligacion)
	}
}

// CONTROL: lo que vence DENTRO de la ventana no es una vencida.
//
// Sin este tramo, una derivacion que metiera todo en el cubo de vencidas
// pasaria los tres tests de arriba con nota.
func TestLoQueVenceDentroDeLaVentanaNoEsUnaVencida(t *testing.T) {
	ahora := time.Date(2026, 8, 27, 9, 0, 0, 0, time.UTC)
	ps := []*corpus.Paquete{paqueteConRelojes("urn:demo:ok",
		periodicaDe("ok.aldia", "1", "x", "P12M", "2020-01-01"),
	)}
	cal := Derivar12Meses(ps, TodoAplica,
		ventana.Hechos{"x": time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)}, ahora)

	if len(cal.Vencidas) != 0 || cal.VencimientosPasados != 0 {
		t.Errorf("una obligacion al dia se ha contado como vencida: %d filas, %d vencimientos",
			len(cal.Vencidas), cal.VencimientosPasados)
	}
	if cal.Total() != 1 {
		t.Errorf("%d fechas en la ventana, esperaba 1", cal.Total())
	}
}
