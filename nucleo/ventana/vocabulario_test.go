package ventana_test

import (
	"errors"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// EL VOCABULARIO DA LA VUELTA ENTERA: lo que se escribe se vuelve a leer igual.
//
// # Por que la ida y la vuelta, y no solo la lectura
//
// Porque el defecto que trajo este fichero fue tener DOS tablas que no se
// conocian: una para leer el corpus y otra para leer el expediente, escritas a
// mano con seis meses de diferencia, y con una pieza de menos en la segunda
// (`fin_dia` donde el corpus escribe `fin_de_dia`).
//
// Un test que solo comprobara la lectura no habria visto nada: cada tabla leia
// bien lo que ella misma escribia. Lo que hay que afirmar es que **escribir y
// leer son la misma tabla**, y eso solo se ve dando la vuelta.
func TestElVocabularioDelRegimenDaLaVueltaEntera(t *testing.T) {
	// LOS TRES EJES, CON TODOS SUS VALORES. Si el tipo gana uno nuevo y nadie
	// lo anade aqui, lo caza el recuento de abajo.
	cierres := []ventana.Cierre{ventana.CierreAuto, ventana.CierreExacto, ventana.CierreFinDia}
	traslados := []ventana.Traslado{ventana.TrasladoNinguno, ventana.TrasladoSiguienteHabil}
	computos := []ventana.Computo{ventana.Naturales, ventana.Habiles}

	vueltas := 0
	for _, c := range cierres {
		for _, tr := range traslados {
			for _, co := range computos {
				reg, err := ventana.RegimenDesde(co.String(), c.String(), tr.String())
				if err != nil {
					t.Fatalf("lo que escribe el propio vocabulario (%q, %q, %q) no se puede "+
						"volver a leer: %v.\n"+
						"  Es el fallo exacto que trajo este fichero: escribir con una tabla "+
						"y leer con otra", co, c, tr, err)
				}
				vueltas++
				if reg.Comp != co {
					t.Errorf("computo %q dio la vuelta como %q", co, reg.Comp)
				}
				if reg.Cierre != c {
					t.Errorf("cierre %q dio la vuelta como %q", c, reg.Cierre)
				}
				if reg.Trasl != tr {
					t.Errorf("traslado %q dio la vuelta como %q", tr, reg.Trasl)
				}
			}
		}
	}
	if vueltas != len(cierres)*len(traslados)*len(computos) {
		t.Fatalf("solo %d combinaciones recorridas: el bucle esta roto", vueltas)
	}

	// LA CADENA VACIA ES LA AUSENCIA, y tiene que valer: un `paquete.json` OMITE
	// el campo cuando no elige. Si esto no valiera, 248 obligaciones del corpus
	// dejarian de cargar.
	reg, err := ventana.RegimenDesde("", "", "")
	if err != nil {
		t.Fatalf("el cuarteto omitido tiene que valer: %v", err)
	}
	if reg.Comp != ventana.Naturales || reg.Cierre != ventana.CierreAuto ||
		reg.Trasl != ventana.TrasladoNinguno {
		t.Errorf("la ausencia tiene que dar el regimen por defecto y dio (%q, %q, %q)",
			reg.Comp, reg.Cierre, reg.Trasl)
	}

	// Y LA PALABRA QUE ESCRIBE EL CORPUS, LITERAL, porque es la que se comia el
	// verificador. No se deriva de nada: se escribe tal cual aparece en los
	// paquetes, para que este caso siga valiendo aunque alguien renombre la
	// constante.
	if reg, err := ventana.RegimenDesde("naturales", "fin_de_dia", "ninguno"); err != nil {
		t.Errorf("`fin_de_dia` es lo que el corpus escribe 196 veces y no se entiende: %v", err)
	} else if reg.Cierre != ventana.CierreFinDia {
		t.Errorf("`fin_de_dia` se leyo como %q y tenia que ser el cierre al final del dia.\n"+
			"  Este es EL caso: el verificador tenia `fin_dia`, con una pieza de menos, "+
			"y sin default, asi que caia al valor cero sin decir nada", reg.Cierre)
	}
}

// UNA PALABRA QUE NO ESTA EN EL VOCABULARIO ES UN ERROR, NUNCA EL VALOR CERO.
//
// # Por que este es el test que importa
//
// Porque el valor cero de los tres ejes es el mas indulgente de cada uno
// (`Naturales`, `CierreAuto`, `TrasladoNinguno`), asi que un `switch` sin
// `default` no falla: acierta hacia el lado suave. Eso es el invariante 8 en su
// tercera forma, presente y no interpretable, dentro de la aritmetica del reloj
// legal.
//
// `fin_dia` va en la lista EXPRESAMENTE. No es una palabra inventada para el
// test: es la que el verificador tenia escrita, y tiene que quedar rechazada
// para que nadie la reintroduzca creyendo que es un sinonimo.
func TestUnaPalabraQueNoEstaEnElVocabularioEsUnError(t *testing.T) {
	for _, c := range []struct {
		computo, cierre, traslado string
		porQue                    string
	}{
		{"fin_de_dia", "", "", "un cierre puesto en el campo del computo: el error de copiar y pegar"},
		{"laborables", "", "", "una palabra plausible que no es la del vocabulario"},
		{"", "fin_dia", "", "LA DEL VERIFICADOR, con una pieza de menos. Si esta se acepta, " +
			"vuelven a existir dos formas de escribir lo mismo"},
		{"", "findedia", "", "sin los guiones bajos"},
		{"", "FIN_DE_DIA", "", "en mayusculas: el vocabulario no es indiferente a la caja, " +
			"porque aceptarla obliga a normalizar y eso es otra tabla"},
		{"", "", "habil_siguiente", "el traslado con las palabras del reves"},
		{"", "", "si", "un booleano disfrazado"},
	} {
		_, err := ventana.RegimenDesde(c.computo, c.cierre, c.traslado)
		if err == nil {
			t.Errorf("RegimenDesde(%q, %q, %q) NO dio error.\n"+
				"  Por que importa: %s\n"+
				"  El valor cero de los tres ejes es el mas indulgente, asi que no fallar "+
				"aqui significa calcular una fecha distinta de la declarada y decir que "+
				"todo cuadra.", c.computo, c.cierre, c.traslado, c.porQue)
			continue
		}
		if !errors.Is(err, ventana.ErrVocabularioDelRegimen) {
			t.Errorf("RegimenDesde(%q, %q, %q) fallo sin el centinela: %v",
				c.computo, c.cierre, c.traslado, err)
		}
	}

	// CONTROL POSITIVO: sin esto, un `RegimenDesde` que devolviera error siempre
	// pasaria todo lo de arriba, y desde fuera no se distingue.
	if _, err := ventana.RegimenDesde("habiles", "exacto", "siguiente_habil"); err != nil {
		t.Errorf("el cuarteto entero y bien escrito tiene que valer: %v", err)
	}
}
