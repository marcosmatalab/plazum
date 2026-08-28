package corpus

import (
	"strings"
	"testing"
)

// MISMO TEXTO LEGAL, MISMA CADENCIA.
//
// EL CASO QUE LA TRAJO, cazado ANTES de escribirlo en el corpus: al proponer los
// intervalos de las 34 cadencias sin numero del anexo de 2024/2690, los puntos
// 12.2.3 y 12.3.3 salieron a P24M y P12M. Tienen el mismo texto legal palabra
// por palabra y son puntos adyacentes de la misma seccion. Nada lo habria dicho,
// y un lector que abra las dos fichas seguidas encuentra la contradiccion en un
// minuto.

func gemelas(textoA, textoB, cadA, cadB, porqueA, porqueB string) *Paquete {
	p := enciendeElReloj(base())
	mk := func(id, texto, cad, porque string) Obligacion {
		return Obligacion{
			ID: id, Articulo: "ritual plazum sobre anexo, punto " + id, ClaseE2E: "procedimental",
			TextoLegal: texto, Cita: "cita de prueba",
			Vigencia: Vigencia{Desde: "2022-05-05"}, Recursos: []TipoRecurso{"Hallazgo"},
			Temporalidad: &Temporalidad{
				Primitiva: "periodica", Hito: "h" + id, Cadencia: cad,
				Regimen:            RegimenSpec{Computo: "naturales", Cierre: "fin_de_dia"},
				Disparador:         map[string]string{"hecho": "ultima_" + id},
				OrigenDelIntervalo: IntervaloPropuesto,
				JustificacionDelIntervalo: "El intervalo lo pone plazum porque el punto manda " +
					"hacerlo a intervalos planificados y no da ningun numero.",
				CadenciaDistintaPorque: porque,
			},
		}
	}
	p.Obligaciones = []Obligacion{mk("a", textoA, cadA, porqueA), mk("b", textoB, cadB, porqueB)}
	p.Aplicabilidad = Aplicabilidad{Reglas: []ReglaSpec{
		{ID: "r1", Cita: "art. 2.1", Regla: `aplica("a", S) :- en_ambito(S)`},
		{ID: "r2", Cita: "art. 2.1", Regla: `aplica("b", S) :- en_ambito(S)`},
	}}
	return p
}

const mismoTexto = "Las entidades pertinentes revisaran y, cuando proceda, actualizaran la " +
	"politica a intervalos planificados o cuando se produzcan incidentes significativos."

func TestDosObligacionesConElMismoTextoNoPuedenLlevarCadenciasDistintas(t *testing.T) {
	p := gemelas(mismoTexto, mismoTexto, "P24M", "P12M", "", "")
	errs := p.Validar()
	if !hay(errs, ErrCadenciaGemelaDiscrepante) {
		t.Fatalf("mismo texto y distinta cadencia tiene que caerse: %v", errs)
	}
	// SE QUEJA DE LAS DOS, no de una. Con dos obligaciones no hay una canonica,
	// asi que senalar solo a una seria elegir por el autor cual es la normal.
	n := 0
	for _, e := range errs {
		if strings.Contains(e.Error(), "mismo texto legal con cadencias distintas") {
			n++
		}
	}
	if n != 2 {
		t.Errorf("%d quejas, esperaba 2: las dos del grupo tienen que argumentarlo", n)
	}
}

// El origen tambien cuenta. Dos textos identicos con el mismo numero pero uno
// declarado suelo legal y otro propuesto dicen cosas opuestas sobre si el
// cliente lo puede mover, que es peor que discrepar en el numero.
func TestElMismoTextoConOrigenesDistintosTampocoPasa(t *testing.T) {
	p := gemelas(mismoTexto, mismoTexto, "P12M", "P12M", "", "")
	p.Obligaciones[0].Articulo = "anexo, punto a"
	p.Obligaciones[0].Temporalidad.OrigenDelIntervalo = IntervaloSueloLegal
	p.Obligaciones[0].Temporalidad.JustificacionDelIntervalo = ""
	p.Obligaciones[0].Temporalidad.CitaDelIntervalo =
		"Reglamento X, punto a: revisaran AL MENOS UNA VEZ AL ANO la politica."
	if !hay(p.Validar(), ErrCadenciaGemelaDiscrepante) {
		t.Fatal("el mismo texto no puede dar un suelo legal y un intervalo propuesto")
	}
}

// LA VALVULA: la diferencia se puede declarar, y entonces calla. Lo que la
// regla exige no es que sean iguales, es que la diferencia este ESCRITA.
func TestLaDiferenciaDeclaradaEnLasDosCalla(t *testing.T) {
	razon := "Hablan de politicas distintas: la de manipulacion de activos cambia con cada " +
		"revision del inventario y la de soportes extraibles apenas cambia."
	p := gemelas(mismoTexto, mismoTexto, "P24M", "P12M", razon, razon)
	if hay(p.Validar(), ErrCadenciaGemelaDiscrepante) {
		t.Fatalf("con la diferencia declarada en las dos, la regla calla: %v", p.Validar())
	}
}

// Y NO BASTA CON QUE LA DECLARE UNA. Es la mitad del diseno: si valiera con una,
// el autor la pondria en la que le resulte comoda y la otra quedaria sin decir
// nada, que es exactamente el estado que la regla persigue.
func TestNoBastaConQueLoDeclareUnaDeLasDos(t *testing.T) {
	razon := "Hablan de politicas distintas y una cambia mucho mas deprisa que la otra, " +
		"segun el ritmo al que se mueve el inventario que la alimenta."
	p := gemelas(mismoTexto, mismoTexto, "P24M", "P12M", razon, "")
	if !hay(p.Validar(), ErrCadenciaGemelaDiscrepante) {
		t.Fatal("la que no lo declara sigue sin decir por que")
	}
}

// Una etiqueta no es un argumento, igual que en el resto de la familia.
func TestUnaRazonDeEtiquetaNoValeComoExcepcion(t *testing.T) {
	p := gemelas(mismoTexto, mismoTexto, "P24M", "P12M", "son distintas", "son distintas")
	if !hay(p.Validar(), ErrCadenciaGemelaDiscrepante) {
		t.Fatal("una razon por debajo del suelo de caracteres no es una razon")
	}
}

// CONTROL NEGATIVO 1: textos DISTINTOS pueden llevar lo que quieran. Si esta
// regla se disparara ahi, pediria la misma cadencia a todo el corpus.
func TestTextosDistintosPuedenLlevarCadenciasDistintas(t *testing.T) {
	p := gemelas(mismoTexto, "Otra cosa completamente distinta que la norma manda revisar.",
		"P24M", "P12M", "", "")
	if hay(p.Validar(), ErrCadenciaGemelaDiscrepante) {
		t.Fatal("dos textos distintos no son gemelos")
	}
}

// CONTROL NEGATIVO 2: mismo texto y misma cadencia, que es el caso normal y el
// que ya existe en el corpus (los seis rituales de iso42001).
func TestElMismoTextoConLaMismaCadenciaCalla(t *testing.T) {
	p := gemelas(mismoTexto, mismoTexto, "P12M", "P12M", "", "")
	if hay(p.Validar(), ErrCadenciaGemelaDiscrepante) {
		t.Fatal("mismo texto y misma cadencia es justo lo que la regla pide")
	}
}

// LOS ESPACIOS NO ABREN UNA PUERTA. Sin normalizar, un salto de linea de mas
// convierte dos textos identicos en distintos y la regla se apaga sola: da
// verde y parece que ha mirado, que es el peor modo de fallo de una guarda.
func TestUnSaltoDeLineaDeMasNoApagaLaRegla(t *testing.T) {
	p := gemelas(mismoTexto, strings.Replace(mismoTexto, " ", "\n  ", 1), "P24M", "P12M", "", "")
	if !hay(p.Validar(), ErrCadenciaGemelaDiscrepante) {
		t.Fatal("el mismo texto con otro espaciado sigue siendo el mismo texto")
	}
}

// Sin texto legal no hay nada que comparar, y agrupar por el vacio juntaria todo
// con todo. Un paquete referencial no trae texto: ahi la regla no aplica, en vez
// de aplicar sobre la nada.
func TestSinTextoLegalLaReglaNoAgrupa(t *testing.T) {
	p := gemelas("", "", "P24M", "P12M", "", "")
	if hay(p.Validar(), ErrCadenciaGemelaDiscrepante) {
		t.Fatal("dos obligaciones sin texto legal no son gemelas: no hay texto que comparar")
	}
}
