package corpus

import (
	"strings"
	"testing"
)

// Hallazgo del frente de corpus, verificado aqui antes de arreglarlo.
//
// La clase de un paquete es un uint8 que viene de un fichero JSON, o sea de
// fuera. Validar la mira con un switch que tiene default, y String() indexa un
// array de cinco. Las dos cosas juntas dan un agujero en la unica frontera que
// CLAUDE.md declara no negociable: el linter legal.

const textoLargoSimulado = "texto normativo simulado de una norma de pago que no se puede " +
	"redistribuir, repetido hasta pasar holgadamente del limite de ciento veinte caracteres " +
	"que impone el linter para los paquetes referenciales"

// Una clase fuera de rango esquiva el limite de texto: cae en el default del
// switch, que no comprueba nada. Un paquete que en la practica es referencial
// pero declara "clase": 9 redistribuye texto de ISO sin que nadie lo pare.
func TestHostilUnaClaseFueraDeRangoEsquivaLaFronteraLegal(t *testing.T) {
	p := &Paquete{
		URN: "urn:es:iso:27001", Version: "2022", Clase: Clase(9),
		Vigencia: Vigencia{Desde: "2022-01-01"},
		Obligaciones: []Obligacion{{
			ID: "iso27001.a.5.1", Articulo: "A.5.1", Cita: "ISO/IEC 27001:2022 A.5.1", ClaseE2E: "continua",
			TextoLegal: textoLargoSimulado,
		}},
	}
	if len(textoLargoSimulado) <= LimiteTextoReferencial {
		t.Fatalf("el texto de prueba tiene que pasar del limite para que el test pruebe algo: %d",
			len(textoLargoSimulado))
	}
	errs := p.Validar()
	if len(errs) == 0 {
		t.Fatalf("HALLAZGO: un paquete con clase 9 y %d caracteres de texto normativo valida "+
			"limpio. El switch de Validar tiene default, asi que una clase fuera de rango no "+
			"es referencial y no tiene limite. Es la frontera legal esquivada con un numero",
			len(textoLargoSimulado))
	}
	var mencionaClase bool
	for _, e := range errs {
		if strings.Contains(e.Error(), "fuera de rango") {
			mencionaClase = true
		}
	}
	if !mencionaClase {
		t.Fatalf("tiene que quejarse de la CLASE fuera de rango. Ojo: buscar solo la palabra "+
			"clase da un falso negativo, porque clase_e2e la contiene. Errores: %v", errs)
	}
}

// Y la misma clase fuera de rango revienta al imprimirla: String() indexa un
// array de cinco elementos con un uint8 que viene de un fichero.
func TestHostilUnaClaseFueraDeRangoNoRevientaAlImprimirse(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("HALLAZGO: Clase(9).String() hace panic (%v). El valor viene de un JSON "+
				"que aporta un tercero, asi que un paquete malformado tumba a quien lo liste", r)
		}
	}()
	for _, c := range []Clase{Importado, Transcrito, Referencial, Delegado, Propio, Clase(9), Clase(255)} {
		if s := c.String(); s == "" {
			t.Fatalf("la clase %d no puede imprimirse como cadena vacia", uint8(c))
		}
	}
}

// Control negativo de los dos: las clases validas siguen funcionando y el
// limite sigue aplicandose a los referenciales de verdad.
func TestHostilElLimiteSigueVigenteEnLosReferencialesDeVerdad(t *testing.T) {
	p := &Paquete{
		URN: "urn:es:iso:27001", Version: "2022", Clase: Referencial,
		Vigencia: Vigencia{Desde: "2022-01-01"},
		Obligaciones: []Obligacion{{
			ID: "iso27001.a.5.1", Articulo: "A.5.1", Cita: "ISO/IEC 27001:2022 A.5.1", ClaseE2E: "continua",
			TextoLegal: textoLargoSimulado,
		}},
	}
	if errs := p.Validar(); len(errs) == 0 {
		t.Fatal("un referencial con texto largo tiene que rechazarse: es la frontera legal")
	}
}
