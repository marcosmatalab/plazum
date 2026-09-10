package estado

import (
	"reflect"
	"strings"
	"testing"
)

// EL INVARIANTE 13, VIGILADO: un recolector entrega HECHOS, jamas veredictos.
//
// # Por que la puerta va sobre el TIPO y no sobre los recolectores
//
// Porque hoy no hay ninguno: `puertos.Recoleccion` no lo implementa nadie
// (medido el 07-09-2026). Una puerta escrita sobre «la salida de un recolector»
// nace vigilando el vacio, que es el verde mas facil de conseguir y el que este
// repositorio lleva catorce hallazgos aprendiendo a no confundir con un verde.
//
// Lo que si existe hoy y es la frontera de verdad es el TIPO: la firma del
// puerto es `Recolectar(...) ([]estado.Observacion, string, error)`, asi que
// TODO lo que un recolector puede decir cabe en esta estructura y en ninguna
// otra. Un conector no puede devolver un veredicto que `Observacion` no sepa
// transportar. Vigilar el tipo vigila a todos los recolectores que existan
// nunca, incluidos los que escriba un tercero en WASM, que es justo la familia
// donde el autor del codigo no somos nosotros.
//
// # Que se prohibe, exactamente
//
// Un campo cuyo nombre afirme CUMPLIMIENTO. «47 de 52 cuentas tenian MFA el
// 07-09-2026» es una observacion; «cumples el 4.2.3» es un juicio, y el juicio
// lo compone `Calcular` con la prueba del paquete delante y lo confirma una
// persona.
//
// # EL OTRO LADO DE LA FRONTERA, contestado por escrito
//
// `Satisfecho` es un bool y PARECE un veredicto. Se queda, y se queda con su
// significado escrito: es el resultado de un PREDICADO MECANICO DECLARADO EN LA
// PRUEBA, o sea el hecho «el predicado dio verdadero sobre este recurso», no la
// afirmacion «esto cumple». La diferencia no es de matiz: con el predicado
// declarado en el paquete, quien lea el expediente puede ir a mirar QUE se
// evaluo; sin el, un bool a true es un veredicto disfrazado de dato y nadie
// puede contrastarlo.
//
// Esa mitad NO la vigila este test, y SI la vigila otra: el predicado es
// obligatorio desde A1, lo exige `corpus.validarPruebas` con
// `ErrPruebaSinPredicado` (nucleo/corpus/prueba.go:247).
//
// LO VIGILA: TestUnRecolectorNoPuedeDevolverUnVeredicto para la mitad del tipo,
// y TestLasFormasDeRomperUnaPrueba (caso «sin predicado») para la del
// predicado obligatorio.
//
// # LA FRASE QUE ESTABA AQUI ERA FALSA, Y NO LA CAZO NADA
//
// Hasta el 07-09-2026 este bloque decia que el predicado «llegara con el linter
// de A1» y que hasta entonces Satisfecho estaba «documentado y no exigido». A1
// entro dos commits despues de escribirlo y lo dejo exigido, asi que la frase
// paso a describir un mundo que ya no existia.
//
// Y el detalle que la hace doctrina y no descuido: llevaba un `NADIE LO VIGILA`
// puesto a mano, o sea la marca que existe para declarar un hueco, Y ESA MARCA
// NO LA VALIDO NADIE, porque `godoc_vigilado_test.go` se salta los ficheros de
// test (godoc_vigilado_test.go:128). Un descargo escrito en un sitio donde la
// puerta no mira es exactamente el hueco que el descargo decia estar tapando.

// vocabularioDeVeredicto son las raices que delatan un juicio en el nombre de un
// campo. Lista NEGRA a proposito, y es la excepcion razonada a la regla de la
// casa de preferir listas blancas: una lista blanca sobre nombres de campo
// obligaria a venir aqui a dar de alta cada campo nuevo y legitimo de una
// estructura de datos que va a crecer, y una puerta que salta en cada campo
// nuevo entrena a esquivarla. Lo que se persigue aqui es concreto y corto: las
// palabras con las que se escribe un juicio de cumplimiento.
var vocabularioDeVeredicto = []string{
	"cumpl",     // Cumple, Cumplimiento, Incumple
	"conform",   // Conforme, Conformidad, NoConforme
	"veredicto", //
	"aprobad",   // Aprobado
	"apto",      //
	"pasa",      // Pasa, Pasado
	"suspend",   // Suspendido
	"valorac",   // Valoracion
	"puntuac",   // Puntuacion
	"nota",      //
	"riesgo",    // un recolector no puntua riesgo: eso es E7
	"grave",     // Gravedad
	"criticid",  // Criticidad
	"severid",   // Severidad
	// Y «satisf», que es el que de verdad importa de esta lista: `Satisfecho`
	// es un bool con nombre inocente que AFIRMA que algo esta satisfecho. Si no
	// estuviera aqui, el unico campo con forma de veredicto que hoy existe
	// pasaria sin que nadie lo justificara, y la puerta vigilaria solo los
	// veredictos que llegaran con mala letra.
	"satisf", // Satisfecho, Satisfactorio
}

// camposJustificados son los que suenan a juicio y NO lo son, cada uno con su
// motivo escrito. Un campo entra aqui a mano y con su linea, que es el gesto
// que esta puerta existe para forzar.
var camposJustificados = map[string]string{
	"Satisfecho": "es el resultado de un predicado mecanico declarado en la prueba, " +
		"o sea el hecho «el predicado dio verdadero», no la afirmacion «esto cumple». " +
		"El predicado es obligatorio: lo exige corpus.validarPruebas con " +
		"ErrPruebaSinPredicado.",
}

func TestUnRecolectorNoPuedeDevolverUnVeredicto(t *testing.T) {
	tipo := reflect.TypeOf(Observacion{})
	if tipo.NumField() == 0 {
		t.Fatal("Observacion sin campos: el detector no esta mirando nada")
	}
	for i := 0; i < tipo.NumField(); i++ {
		nombre := tipo.Field(i).Name
		motivo, justificado := camposJustificados[nombre]
		hallazgo := sueneAVeredicto(nombre)
		switch {
		case hallazgo != "" && !justificado:
			t.Errorf("Observacion.%s suena a veredicto (%q) y no esta justificado.\n"+
				"  Invariante 13: un recolector entrega HECHOS, jamas veredictos. El\n"+
				"  juicio lo compone estado.Calcular con la prueba del paquete delante\n"+
				"  y lo confirma una persona.\n"+
				"  Arreglo: o el campo sale, o entra en camposJustificados con el motivo\n"+
				"  de por que es un hecho y no un juicio.", nombre, hallazgo)
		case hallazgo == "" && justificado:
			// LA DIRECCION CONTRARIA. Una justificacion para un campo que ya no
			// suena a veredicto (porque se renombro, o porque se fue) es una
			// excusa huerfana, y una lista de excusas huerfanas es como una
			// lista negra deja de vigilar sin que nadie se entere.
			t.Errorf("camposJustificados justifica %q y ese campo ya no suena a "+
				"veredicto ni existe. Arreglo: quitar la entrada.\n  decia: %s",
				nombre, motivo)
		}
	}
	for nombre := range camposJustificados {
		if _, hay := tipo.FieldByName(nombre); !hay {
			t.Errorf("camposJustificados justifica %q, que ya no es un campo de "+
				"Observacion. Arreglo: quitar la entrada", nombre)
		}
	}
}

// sueneAVeredicto devuelve la raiz que ha casado, o vacio.
func sueneAVeredicto(nombre string) string {
	n := strings.ToLower(nombre)
	for _, r := range vocabularioDeVeredicto {
		if strings.Contains(n, r) {
			return r
		}
	}
	return ""
}

// TestElDetectorDeVeredictosSaltaCuandoDebe es el control negativo, en las dos
// direcciones: tiene que acusar los nombres con los que de verdad se escribe un
// juicio, y callarse con los nombres de un hecho.
func TestElDetectorDeVeredictosSaltaCuandoDebe(t *testing.T) {
	casos := []struct {
		nombre  string
		veredic bool
	}{
		// Los que un conector escribiria sin mala intencion, que es como entra
		// esto de verdad: el autor del conector cree que esta ayudando.
		{"Cumple", true},
		{"Incumplimiento", true},
		{"Conforme", true},
		{"Veredicto", true},
		{"Aprobado", true},
		{"Apto", true},
		{"Severidad", true},
		{"NivelDeRiesgo", true},
		{"Puntuacion", true},
		{"Satisfecho", true},
		// Y los hechos, que son los que NO pueden ponerse rojos, porque una
		// puerta que acusa a un campo legitimo se acaba desactivando.
		{"Recurso", false},
		{"Recolectada", false},
		{"Caduca", false},
		{"Recolector", false},
		{"HashCarga", false},
		{"ErrorRecol", false},
		{"Version", false},
		{"Prueba", false},
	}
	for _, c := range casos {
		if hay := sueneAVeredicto(c.nombre) != ""; hay != c.veredic {
			t.Errorf("%q: detectado %v, esperaba %v", c.nombre, hay, c.veredic)
		}
	}
}
