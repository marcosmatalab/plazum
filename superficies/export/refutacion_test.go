package export

import (
	"bytes"
	"crypto/ed25519"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/ledger"
)

// LA PUERTA DEL INDICE RENUMERADO, de la revision hostil de la casilla.
//
// La propiedad atacada: "lo que el borrado legal borro no reaparece", y aguanta
// sobre ficheros que NADIE ha verificado todavia. Es una decision declarada del
// export: no comprueba firmas a proposito, porque negarse a filtrar no puede
// depender de que la firma de la lapida cuadre.
//
// La pregunta que la tumbo: la guarda casaba lapida con entrada POR EL INDICE.
// Y el indice de una entrada es un entero del fichero JSON que no protege
// ninguna firma. Renumerar la entrada suprimida, y mover su clave divulgada a la
// casilla nueva, son dos ediciones de texto: la lapida se queda intacta, con su
// firma buena y su base legal, y el contenido borrado sale en claro al SIEM de
// un tercero con retencion propia. Comprobado contra el BINARIO y contra el
// expediente demo publicado, no solo aqui:
//
//	$ sed 's/"indice": 2,/"indice": 99,/; s/^    "2": "/    "99": "/' suprimido.json > renumerado.json
//	$ ./plazum export renumerado.json | grep -c sede-electronica
//	1
//
// nucleo/ledger ya habia aprendido esta leccion entera: la lapida se firma sobre
// HashEntrada porque el indice solo no ata nada ("una lapida legitima se pegaba
// tal cual en otra cadena y suprimia alli lo que ocupara el mismo indice"), y de
// ahi salio su ErrLapidaDeOtraEntrada. La senal estaba dentro del fichero, en la
// propia lapida, y el export no la miraba.
//
// LA MUTACION que vigila esto: borrar en contenidoDe las lineas que preguntan
// por c.porHash. Con ellas fuera este test se pone rojo y el de al lado
// (TestElExportNoFiltraLoQueElBorradoLegalBorro) sigue verde, que es justo por
// lo que hace falta un test aparte.
func TestLaGuardaDelBorradoLegalMiraElHashDeLaLapidaYNoSoloElIndice(t *testing.T) {
	e := nuevoEscenario(t, cargaConCentinelas(t),
		carga(t, obsDePrueba("control.dos", "recurso-que-si-sale", true)))
	e.suprime(t, 0, "urn:demo:ley art. 17", "2026-09-18T07:30:00Z", "control.uno")

	// EL ATAQUE, y es una edicion de fichero, no una llamada al ledger: la
	// entrada suprimida cambia de numero y su clave divulgada le sigue. Ni la
	// lapida ni su firma se tocan.
	e.exp.Cadena.Entradas[0].Indice = 99
	e.exp.ClavesEntradas[99] = e.exp.ClavesEntradas[0]
	delete(e.exp.ClavesEntradas, 0)
	// Y la atribucion del expediente, que es la otra senal de supresion, sigue
	// apuntando al indice viejo: si el test la moviera, estaria probando el
	// camino de al lado y no la guarda del hash.
	if e.exp.SupresionesDeEvidencia[0].Entrada != 0 {
		t.Fatal("la atribucion tiene que seguir en el indice viejo: si no, el contenido " +
			"lo retendria la otra guarda y este test no vigilaria nada")
	}

	// La senal que tiene que parar esto esta DENTRO del fichero: la lapida
	// sigue firmada sobre el hash de esta misma entrada.
	if !bytes.Equal(e.exp.Cadena.Lapidas[0].HashEntrada, e.exp.Cadena.Entradas[0].Hash) {
		t.Fatal("el ataque necesita que la lapida siga atada por hash a la entrada " +
			"renumerada: sin esa atadura no habria senal disponible y el hallazgo seria otro")
	}

	salida, _ := exportar(t, e.exp)
	for _, c := range centinelas {
		if strings.Contains(salida, c) {
			t.Errorf("el contenido de una entrada BORRADA CON BASE LEGAL sale en claro al "+
				"fichero del SIEM tras renumerar su indice (%q).\n"+
				"  La lapida sigue en el fichero, con su HashEntrada apuntando a esta misma "+
				"entrada, y la guarda no la mira.\n"+
				"  Arreglo: indexar las lapidas tambien por hex(HashEntrada) y retener el "+
				"contenido de toda entrada cuyo Hash este en ese conjunto.\n--- fichero ---\n%s",
				c, salida)
		}
	}
	if !strings.Contains(salida, "aunque su indice ya no cuadre") {
		t.Errorf("el evento no dice que la retencion viene del hash y no del indice. El "+
			"receptor tiene que poder distinguir esto de un borrado limpio:\n%s", salida)
	}
}

// CONTROL NEGATIVO de lo anterior: sin lapida, esos mismos valores SI salen aunque
// la entrada este renumerada. Sin esto, el verde de arriba seria compatible con
// un export que dejara de sacar contenido en cuanto un indice no cuadra.
func TestUnaEntradaRenumeradaSinLapidaSigueSacandoSuContenido(t *testing.T) {
	e := nuevoEscenario(t, cargaConCentinelas(t))
	e.exp.Cadena.Entradas[0].Indice = 99
	e.exp.ClavesEntradas[99] = e.exp.ClavesEntradas[0]
	delete(e.exp.ClavesEntradas, 0)
	if len(e.exp.Cadena.Lapidas) != 0 {
		t.Fatal("el control negativo no puede tener lapidas")
	}
	salida, _ := exportar(t, e.exp)
	for _, c := range centinelas {
		if !strings.Contains(salida, c) {
			t.Fatalf("sin lapida, el valor %q tampoco sale de una entrada renumerada, asi "+
				"que el test de arriba pasa por vacuidad.\n--- fichero ---\n%s", c, salida)
		}
	}
}

// Una lapida sin HashEntrada (la que fabrica a mano un fichero de prueba, o una
// de una version vieja) no puede casar con toda entrada que tampoco lo traiga:
// eso taparia la cadena entera y el receptor dejaria de ver nada.
func TestUnaLapidaSinHashNoTapaLasDemasEntradas(t *testing.T) {
	e := nuevoEscenario(t, cargaConCentinelas(t),
		carga(t, obsDePrueba("control.dos", "recurso-que-si-sale", true)))
	e.exp.Cadena.Lapidas = []ledger.Lapida{{
		EntradaBorrada: 0,
		BaseLegal:      "urn:demo:ley art. 17",
		Instante:       "2026-09-18T07:30:00Z",
	}}
	e.exp.Cadena.Entradas[1].Hash = nil // ni siquiera con el hash vacio al otro lado
	salida, _ := exportar(t, e.exp)
	if !strings.Contains(salida, "recurso-que-si-sale") {
		t.Errorf("una lapida sin hash de entrada ha tapado el contenido de OTRA entrada:\n%s",
			salida)
	}
}

// Que la renumeracion es una manipulacion detectable, y por otra capa: la cadena
// no verifica. O sea que la senal existe, esta al alcance del export y el export
// elegia no mirarla. Lo dice aqui para que nadie cierre el hallazgo con "ya lo
// caza plazum verify": el camino documentado (temporizador de systemd que
// exporta cada hora) no verifica nada, y lo que entra en el SIEM no se deshace.
func TestLaCadenaRenumeradaNoVerifica(t *testing.T) {
	e := nuevoEscenario(t, cargaConCentinelas(t))
	e.suprime(t, 0, "urn:demo:ley art. 17", "2026-09-18T07:30:00Z", "control.uno")
	e.exp.Cadena.Entradas[0].Indice = 99
	pub, _ := operador().Public().(ed25519.PublicKey)
	if _, err := e.exp.Cadena.Verificar(ledger.Confianza{ClaveOperador: pub}); err == nil {
		t.Fatal("la cadena renumerada verifica: entonces la renumeracion no seria una " +
			"manipulacion detectable y el hallazgo tendria otra forma")
	}
}
