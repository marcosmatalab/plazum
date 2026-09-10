package metadatos_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/ingesta"
	"github.com/marcosmatalab/plazum/adaptadores/metadatos"
)

// doc monta un documento de un fragmento por linea.
func doc(lineas ...string) ingesta.Documento {
	d := ingesta.Documento{Formato: ingesta.TextoPlano}
	for i, l := range lineas {
		d.Fragmentos = append(d.Fragmentos, ingesta.Fragmento{Orden: i, Pagina: 1, Texto: l})
	}
	return d
}

func fuenteDe(orden int) string { return "aportado:abc:" + string(rune('0'+orden)) }

func proponer(t *testing.T, lineas ...string) map[metadatos.Campo][]metadatos.Propuesta {
	t.Helper()
	out := map[metadatos.Campo][]metadatos.Propuesta{}
	for _, p := range metadatos.Proponer(doc(lineas...), fuenteDe) {
		out[p.Campo] = append(out[p.Campo], p)
	}
	return out
}

// LOS CUATRO CAMPOS SALEN DE UN DOCUMENTO NORMAL.
//
// Es el control POSITIVO de todo lo demas: sin esta mitad, un extractor que no
// propusiera nunca nada pasaria todos los tests de «no se inventa nada» con
// nota, que es la forma mas barata de tener un paquete que no hace nada.
func TestDeUnaFichaNormalSalenLosCuatroCampos(t *testing.T) {
	got := proponer(t,
		"Politica de seguridad de la informacion",
		"Fecha: 2026-01-15",
		"Alcance: los sistemas de la sede de Madrid",
		"Firmado por: Marta Ruiz",
		"Valido hasta: 2027-01-15",
	)
	quiero := map[metadatos.Campo]string{
		metadatos.Fecha:     "2026-01-15",
		metadatos.Alcance:   "los sistemas de la sede de Madrid",
		metadatos.Firmante:  "Marta Ruiz",
		metadatos.Caducidad: "2027-01-15",
	}
	for campo, valor := range quiero {
		ps := got[campo]
		if len(ps) == 0 {
			t.Errorf("no se propone nada para %s", campo)
			continue
		}
		if ps[0].Valor != valor {
			t.Errorf("%s propone %q y tenia que proponer %q", campo, ps[0].Valor, valor)
		}
		// Y CADA PROPUESTA TRAE SU CITA, que es lo que se verifica por hash. Una
		// propuesta sin cita no se puede contrastar contra el documento, o sea
		// que es exactamente lo que la puerta antialucinacion existe para no
		// ensenar.
		if ps[0].Cita == "" {
			t.Errorf("%s se propone SIN CITA", campo)
			continue
		}
		if !strings.Contains(ps[0].Cita, valor) && campo != metadatos.Fecha &&
			campo != metadatos.Caducidad {
			t.Errorf("%s: la cita %q no contiene el valor %q", campo, ps[0].Cita, valor)
		}
		if ps[0].Fuente == "" {
			t.Errorf("%s se propone sin identificador de fuente", campo)
		}
	}
}

// LAS TRES FORMAS DE ESCRIBIR UNA FECHA, y una cuarta que NO es una fecha.
func TestLasFechasSeNormalizanYLoQueNoEsUnaFechaNoSePropone(t *testing.T) {
	for _, c := range []struct {
		linea string
		valor string // vacio: no tiene que proponerse nada
	}{
		{"Fecha: 2026-01-15", "2026-01-15"},
		{"Fecha: 15/01/2026", "2026-01-15"},
		{"Fecha: 15.01.2026", "2026-01-15"},
		{"Fecha: 15 de enero de 2026", "2026-01-15"},
		{"Fecha: 15 de Enero de 2026", "2026-01-15"},
		// LA TERCERA FORMA DE LA NADA (invariante 8): el campo esta, trae algo,
		// y no se entiende. La respuesta es NO PROPONER, nunca un valor por
		// defecto: una fecha inventada en una ficha se acepta de un clic.
		{"Fecha: proximamente", ""},
		{"Fecha: ayer por la tarde", ""},
		// Y UNA FECHA QUE NO EXISTE TAMPOCO SE PROPONE. Un 31 de febrero
		// guardado como caducidad da un plazo que ningun reloj puede calcular.
		{"Fecha: 2026-02-31", ""},
		{"Fecha: 31/02/2026", ""},
		// Un ano fuera de rango es casi siempre otra cosa.
		{"Fecha: 15/01/1204", ""},
	} {
		t.Run(c.linea, func(t *testing.T) {
			ps := proponer(t, c.linea)[metadatos.Fecha]
			switch {
			case c.valor == "" && len(ps) > 0:
				t.Errorf("se propone %q y no tenia que proponerse nada.\n"+
					"  Un dato que esta y no se entiende es un error, nunca el valor por "+
					"defecto: aqui el valor por defecto seria una fecha inventada",
					ps[0].Valor)
			case c.valor != "" && len(ps) == 0:
				t.Errorf("no se propone nada y tenia que proponerse %q", c.valor)
			case c.valor != "" && ps[0].Valor != c.valor:
				t.Errorf("se propone %q y tenia que proponerse %q", ps[0].Valor, c.valor)
			}
		})
	}
}

// «VALIDO HASTA» ES CADUCIDAD Y NO FECHA, aunque las dos etiquetas casen.
//
// Es el caso que obliga a mirar la etiqueta mas especifica primero. Sin ese
// orden, «Valido hasta: 2027-01-01» se propondria como FECHA del documento, o
// sea que la ficha diria que el documento es de 2027 y no diria cuando caduca.
// Las dos mitades importan y por eso se comprueban las dos.
func TestLaCaducidadNoSeLeeComoLaFechaDelDocumento(t *testing.T) {
	got := proponer(t, "Valido hasta: 2027-01-01")
	if ps := got[metadatos.Caducidad]; len(ps) != 1 || ps[0].Valor != "2027-01-01" {
		t.Errorf("«Valido hasta» no sale como caducidad: %+v", ps)
	}
	if ps := got[metadatos.Fecha]; len(ps) != 0 {
		t.Errorf("«Valido hasta» tambien sale como FECHA del documento (%+v), y entonces "+
			"la ficha dice que el documento es de 2027", ps)
	}
	// Y LA DIRECCION CONTRARIA: una linea con las dos etiquetas da las dos
	// propuestas, cada una a su campo.
	got = proponer(t, "Fecha: 2026-01-15  Valido hasta: 2027-01-15")
	if ps := got[metadatos.Fecha]; len(ps) != 1 || ps[0].Valor != "2026-01-15" {
		t.Errorf("con las dos etiquetas, la fecha del documento sale mal: %+v", ps)
	}
	if ps := got[metadatos.Caducidad]; len(ps) != 1 || ps[0].Valor != "2027-01-15" {
		t.Errorf("con las dos etiquetas, la caducidad sale mal: %+v", ps)
	}
}

// UN DOCUMENTO SIN NADA RECONOCIBLE NO PROPONE NADA Y NO ES UN ERROR.
//
// No haber encontrado una fecha NO dice que el documento no la tenga: dice que
// aqui no se ha reconocido ninguna. Son dos cosas distintas y solo una de ellas
// es una carencia del cliente.
func TestUnDocumentoSinFichaNoProponeNada(t *testing.T) {
	ps := metadatos.Proponer(doc(
		"Este documento describe el procedimiento de gestion de incidentes.",
		"Todo incidente se registra en el sistema de tickets.",
	), fuenteDe)
	if len(ps) != 0 {
		t.Errorf("de un texto sin ficha salen %d propuestas: %+v", len(ps), ps)
	}
}

// SIN FORMA DE IDENTIFICAR EL FRAGMENTO NO SE PROPONE NADA.
//
// Es el valor cero restrictivo (invariante 8): una propuesta sin fuente no se
// puede verificar por hash, y una propuesta que no se puede verificar es
// exactamente la que la puerta antialucinacion existe para no ensenar. Se
// recorren las DOS formas de la nada.
func TestSinFuenteNoSeProponeNada(t *testing.T) {
	d := doc("Fecha: 2026-01-15")
	if ps := metadatos.Proponer(d, nil); len(ps) != 0 {
		t.Errorf("con fuenteDe nil salen %d propuestas: una propuesta sin fuente no se "+
			"puede verificar", len(ps))
	}
	if ps := metadatos.Proponer(d, func(int) string { return "" }); len(ps) != 0 {
		t.Errorf("con fuenteDe devolviendo vacio salen %d propuestas", len(ps))
	}
	// CONTROL POSITIVO: con fuente SI se propone. Sin esta mitad, un extractor
	// roto pasaria lo de arriba.
	if ps := metadatos.Proponer(d, fuenteDe); len(ps) != 1 {
		t.Errorf("con fuente salen %d propuestas y tenia que salir una", len(ps))
	}
}

// EL TOPE POR CAMPO SE CRUZA Y NO SE PASA.
//
// Un documento adversario puede traer diez mil lineas que digan «Fecha:», y sin
// tope esto seria un amplificador de memoria por una subida de 4 MiB. Y ademas
// veinte candidatos no son mas informacion: son la pantalla que se estaba
// intentando evitar.
func TestElTopePorCampoSeRespeta(t *testing.T) {
	var lineas []string
	for i := 0; i < metadatos.MaxPropuestasPorCampo*4; i++ {
		lineas = append(lineas, "Fecha: 2026-01-15")
	}
	ps := metadatos.Proponer(doc(lineas...), fuenteDe)
	if len(ps) != metadatos.MaxPropuestasPorCampo {
		t.Errorf("de %d lineas salen %d propuestas y el tope es %d",
			len(lineas), len(ps), metadatos.MaxPropuestasPorCampo)
	}
}

// EL ORDEN ES EL DEL DOCUMENTO, no el de ninguna heuristica de calidad.
//
// Ordenar por «cual parece mejor» seria emitir un juicio por la puerta de atras:
// la primera de una lista se lee como la recomendada. Aqui la primera es la
// primera que aparece, y eso se puede comprobar mirando el documento.
func TestElOrdenDeLasPropuestasEsElDelDocumento(t *testing.T) {
	ps := metadatos.Proponer(doc(
		"Fecha: 2026-01-15",
		"Fecha: 2020-03-03",
		"Fecha: 2030-12-31",
	), fuenteDe)
	quiero := []string{"2026-01-15", "2020-03-03", "2030-12-31"}
	if len(ps) != len(quiero) {
		t.Fatalf("salen %d propuestas y tenian que salir %d", len(ps), len(quiero))
	}
	for i, q := range quiero {
		if ps[i].Valor != q {
			t.Errorf("la propuesta %d es %q y tenia que ser %q (el orden del documento)",
				i, ps[i].Valor, q)
		}
		if ps[i].Orden != i {
			t.Errorf("la propuesta %d dice venir del fragmento %d", i, ps[i].Orden)
		}
	}
}

// LA CITA ES LA LINEA ENTERA Y NO SOLO EL VALOR.
//
// «2027-01-01» suelto puede aparecer veinte veces en un documento, asi que
// verificarlo por hash no probaria de donde salio. La cita lleva la etiqueta
// dentro, que es lo que una persona necesita leer para decidir si acepta.
func TestLaCitaLlevaLaEtiquetaYNoSoloElValor(t *testing.T) {
	ps := proponer(t, "Firmado por: Marta Ruiz")[metadatos.Firmante]
	if len(ps) != 1 {
		t.Fatalf("salen %d propuestas", len(ps))
	}
	if !strings.Contains(strings.ToLower(ps[0].Cita), "firmado por") {
		t.Errorf("la cita %q no lleva la etiqueta dentro, asi que verificarla no dice de "+
			"donde salio el valor", ps[0].Cita)
	}
}

// NI UN CAMPO DE ESTE TIPO SUENA A VEREDICTO (invariante 13).
//
// Este paquete propone CAMPOS de una ficha, no juicios: no dice si el documento
// sirve para acreditar nada. La puerta que lo vigila de verdad esta en la raiz y
// aplica el vocabulario del nucleo; esta de aqui es la mitad que vive al lado del
// tipo, para que quien anada un campo lo lea antes de anadirlo.
func TestElTipoPropuestaNoLlevaNingunJuicio(t *testing.T) {
	for _, prohibido := range []string{
		"Cumple", "Conforme", "Valido", "Confianza", "Puntuacion", "Veredicto", "Satisfecho",
	} {
		if campoDePropuesta(prohibido) {
			t.Errorf("metadatos.Propuesta tiene un campo %q, que es un juicio.\n"+
				"  Lo que sale de aqui son hechos sobre el documento; si sirve para algo lo "+
				"decide una persona", prohibido)
		}
	}
	// CONTROL POSITIVO del detector: los campos que SI existen se encuentran.
	for _, hay := range []string{"Campo", "Valor", "Cita", "Fuente", "Pagina", "Orden"} {
		if !campoDePropuesta(hay) {
			t.Errorf("el detector no encuentra el campo %q, que existe: no esta mirando "+
				"el tipo y por tanto no comprueba nada", hay)
		}
	}
}

// campoDePropuesta dice si el tipo tiene ese campo, por reflexion.
func campoDePropuesta(nombre string) bool {
	_, hay := reflect.TypeOf(metadatos.Propuesta{}).FieldByName(nombre)
	return hay
}
