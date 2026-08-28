package pantalla

import (
	"testing"
	"time"
)

func fechaEn(a int, m time.Month, d int, obl, marco, cad, origen string) Fecha {
	return Fecha{
		Vence: time.Date(a, m, d, 23, 59, 59, 0, time.UTC),
		Marco: marco, Obligacion: obl, Titulo: obl, Hito: obl,
		Cadencia: cad, OrigenDelIntervalo: origen,
	}
}

// Una obligacion que vence cuatro veces sigue siendo UNA obligacion.
//
// Es el error que hundiria la seccion entera: contar fechas donde hay que
// contar deberes infla justo el numero que esto existe para desinflar. Una
// trimestral da cuatro citas al ano y no son cuatro obligaciones.
func TestUnCicloCuentaObligacionesYNoFechas(t *testing.T) {
	fs := []Fecha{
		fechaEn(2026, 9, 30, "trimestral", "urn:a", "P3M", "propuesto"),
		fechaEn(2026, 12, 30, "trimestral", "urn:a", "P3M", "propuesto"),
		fechaEn(2027, 3, 30, "trimestral", "urn:a", "P3M", "propuesto"),
		fechaEn(2027, 6, 30, "trimestral", "urn:a", "P3M", "propuesto"),
	}
	cs, sueltas := agruparEnCiclos(fs, nil)
	if sueltas != 0 {
		t.Fatalf("no hay plazos unicos y dice %d", sueltas)
	}
	if len(cs) != 1 {
		t.Fatalf("una sola cadencia y salen %d ciclos", len(cs))
	}
	c := cs[0]
	if c.Obligaciones != 1 {
		t.Errorf("cuatro fechas de la MISMA obligacion se cuentan como %d obligaciones",
			c.Obligaciones)
	}
	if len(c.Sentadas) != 4 {
		t.Errorf("cuatro meses distintos son cuatro sentadas y dice %d", len(c.Sentadas))
	}
	if c.ConFecha != 1 || c.EsperandoDato != 0 {
		t.Errorf("con fecha %d, esperando %d", c.ConFecha, c.EsperandoDato)
	}
}

// Una sentada con dos marcos es la pieza de composicion: es lo que un catalogo
// de controles no sabe decir.
func TestUnaSentadaJuntaMarcosDistintos(t *testing.T) {
	fs := []Fecha{
		fechaEn(2027, 4, 10, "una", "urn:a", "P24M", "propuesto"),
		fechaEn(2027, 4, 20, "otra", "urn:b", "P24M", "suelo_legal"),
	}
	cs, _ := agruparEnCiclos(fs, nil)
	if len(cs) != 1 || len(cs[0].Sentadas) != 1 {
		t.Fatalf("dos fechas del mismo mes y ciclo son UNA sentada: %+v", cs)
	}
	s := cs[0].Sentadas[0]
	if len(s.Marcos) != 2 {
		t.Errorf("la sentada cubre dos marcos y dice %d: %v", len(s.Marcos), s.Marcos)
	}
	if cs[0].Obligaciones != 2 || cs[0].Alineables != 2 {
		t.Errorf("dos obligaciones, las dos adelantables: %d y %d",
			cs[0].Obligaciones, cs[0].Alineables)
	}
}

// EL VALOR CERO ES EL RESTRICTIVO (invariante 8): lo que no dice de quien es su
// numero NO se puede adelantar.
//
// Se recorren las DOS formas de la nada: el campo vacio y un valor que no esta
// en el vocabulario. Las dos tienen que dar "no se mueve", porque proponer
// adelantar algo cuyo regimen no se conoce es proponer un incumplimiento.
func TestSoloSeAdelantaLoQueDeclaraQuePuede(t *testing.T) {
	casos := []struct {
		origen string
		puede  bool
	}{
		{"suelo_legal", true},
		{"propuesto", true},
		{"fijado", false},
		{"", false},              // la nada, forma 1
		{"vete_a_saber", false},  // la nada, forma 2: vocabulario nuevo
		{"SUELO_LEGAL", false},   // ni siquiera con otra caja
		{" suelo_legal ", false}, // ni con espacios
	}
	for _, c := range casos {
		f := Fecha{OrigenDelIntervalo: c.origen}
		if f.PuedeAdelantarse() != c.puede {
			t.Errorf("origen %q: PuedeAdelantarse=%v y esperaba %v",
				c.origen, f.PuedeAdelantarse(), c.puede)
		}
		s := SinFecha{OrigenDelIntervalo: c.origen}
		if s.PuedeAdelantarse() != c.puede {
			t.Errorf("origen %q en SinFecha: PuedeAdelantarse=%v y esperaba %v",
				c.origen, s.PuedeAdelantarse(), c.puede)
		}
	}
}

// Un plazo unico NO entra en ninguna sentada, y se cuenta.
//
// Meterlo diria que se puede adelantar para juntarlo con otra cosa, y la
// mayoria de los plazos unicos de este corpus son notificaciones de incidente:
// el consejo equivocado ahi es caro. Y no contarlo seria un descarte mudo.
func TestUnPlazoUnicoNiSeAgrupaNiSePierde(t *testing.T) {
	fs := []Fecha{
		fechaEn(2026, 9, 30, "periodica", "urn:a", "P3M", "propuesto"),
		fechaEn(2026, 9, 15, "notificacion", "urn:a", "", ""), // sin cadencia: un plazo
	}
	cs, sueltas := agruparEnCiclos(fs, nil)
	if sueltas != 1 {
		t.Errorf("la fecha sin cadencia tiene que contarse como suelta, y cuenta %d", sueltas)
	}
	for _, c := range cs {
		for _, s := range c.Sentadas {
			for _, f := range s.Fechas {
				if f.Obligacion == "notificacion" {
					t.Error("un plazo unico se ha colado en una sentada: eso le dice al " +
						"operador que puede adelantarlo para juntarlo con otra cosa")
				}
			}
		}
	}
}

// Un reloj que espera un dato del operador TIENE ciclo aunque no tenga fecha.
//
// Es la mitad del mensaje el dia uno de un cliente: todas sus cadencias estan
// asi, y es justo el dia en que mas falta hace saber cuantas veces al ano habra
// que sentarse.
func TestLosRelojesQueEsperanUnDatoCuentanEnSuCiclo(t *testing.T) {
	sin := []SinFecha{
		{Marco: "urn:b", Obligacion: "espera", Motivo: MotivoPendienteDeHecho,
			Cadencia: "P12M", OrigenDelIntervalo: "propuesto"},
		// Estas dos NO tienen ciclo que ofrecer y no deben contarse.
		{Marco: "urn:b", Obligacion: "sin_plazo", Motivo: MotivoSinPlazoLegal,
			Cadencia: "P12M", OrigenDelIntervalo: "propuesto"},
		{Marco: "urn:b", Obligacion: "sin_ejecutor", Motivo: MotivoSinEjecutor},
	}
	fs := []Fecha{fechaEn(2026, 10, 31, "con_fecha", "urn:a", "P12M", "suelo_legal")}

	cs, _ := agruparEnCiclos(fs, sin)
	if len(cs) != 1 {
		t.Fatalf("una sola cadencia y salen %d ciclos", len(cs))
	}
	c := cs[0]
	if c.Obligaciones != 2 || c.ConFecha != 1 || c.EsperandoDato != 1 {
		t.Errorf("esperaba 2 obligaciones (1 con fecha, 1 esperando) y salen %d (%d/%d)",
			c.Obligaciones, c.ConFecha, c.EsperandoDato)
	}
	if len(c.Marcos) != 2 {
		t.Errorf("el ciclo cubre dos marcos y dice %d: %v", len(c.Marcos), c.Marcos)
	}
	// La que espera no crea sentada: una sentada es una cita en el calendario y
	// esta no tiene fecha.
	if len(c.Sentadas) != 1 {
		t.Errorf("solo una fecha, asi que una sentada, y dice %d", len(c.Sentadas))
	}
}
