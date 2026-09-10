package plazum

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/busqueda"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// LAS LENGUAS DEL CORPUS CABEN EN LA TABLA DE PLEGADO.
//
// # De donde sale esta puerta
//
// De una afirmacion que se quedo sin su mitad. El godoc de
// `adaptadores/busqueda/tokenizar.go` decia «hoy el corpus es castellano e
// ingles, MEDIDO», y el 10-09-2026 se midio: CERO obligaciones en ingles sobre
// 559 con texto legal. Una afirmacion que cita su propia prueba y no tiene
// ninguna es peor que una sin pruebas, porque quien la lee deja de comprobarla.
//
// # Que afirma, y por que importa de verdad desde D-25
//
// Que toda lengua en la que el corpus trae texto esta dentro de lo que la tabla
// de plegado sabe normalizar. Antes de D-25 el corpus tenia una sola lengua y
// esto era casi una tautologia; desde D-25 una obligacion puede declarar
// `versiones_linguisticas`, asi que el corpus PUEDE ganar lenguas escribiendo un
// paquete, sin tocar codigo.
//
// El dia que entre corpus en griego o en cirilico, esta tabla no lo pliega y el
// sintoma seria silencioso: las busquedas sobre ese texto simplemente
// encontrarian menos, sin que nada se pusiera rojo.
//
// # LO QUE ESTA PUERTA NO PROMETE, dicho porque es la otra mitad
//
// Nada sobre los DOCUMENTOS DEL CLIENTE. Ese es el otro conjunto que entra al
// mismo indice y no lo controlamos: llegan en la lengua que sea. Esta puerta
// mide lo que decidimos nosotros, que es el corpus, y ahi acaba.
func TestLasLenguasDelCorpusCabenEnLaTablaDePlegado(t *testing.T) {
	// LAS LENGUAS SE DERIVAN DEL ARBOL, no se escriben aqui: una lista al lado
	// del test es la segunda copia que se queda vieja, que es exactamente el
	// defecto que trajo esta puerta.
	ficheros, err := filepath.Glob(filepath.Join("paquetes", "*", "paquete.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(ficheros) < 15 {
		t.Fatalf("solo %d paquetes: el arnes no ha encontrado el corpus", len(ficheros))
	}

	lenguas := map[string]int{corpus.LenguaDelTextoLegal: 0}
	conTexto := 0
	for _, f := range ficheros {
		b, err := os.ReadFile(f) // #nosec G304 -- ruta de un glob del propio repositorio
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			Obligaciones []struct {
				TextoLegal string `json:"texto_legal"`
				Versiones  map[string]struct {
					Texto string `json:"texto"`
				} `json:"versiones_linguisticas"`
			} `json:"obligaciones"`
		}
		if err := json.Unmarshal(b, &doc); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for _, o := range doc.Obligaciones {
			if o.TextoLegal != "" {
				lenguas[corpus.LenguaDelTextoLegal]++
				conTexto++
			}
			for lengua, v := range o.Versiones {
				if v.Texto != "" {
					lenguas[lengua]++
				}
			}
		}
	}

	// LA TABLA SE PREGUNTA, no se copia: se le da a `Tokenizar` una letra de
	// cada lengua y se mira si la pliega. Preguntarle al plegado directamente
	// obligaria a exportarlo, y este test no tiene por que conocer su forma.
	//
	// La muestra por lengua es minima a proposito: lo que se comprueba no es que
	// la tabla sea completa, es que la lengua NO trae escritura fuera del latin,
	// que es lo unico que esta tabla puede cubrir.
	muestras := map[string]string{
		"es": "ñáéíóúü",
		"en": "abcdefg",
		// Las que romperian: se declaran para que el control negativo de abajo
		// tenga con que trabajar, no porque esten en el corpus.
		"el": "αβγδε",
		"bg": "абвгд",
	}

	var fuera []string
	for lengua := range lenguas {
		muestra, hay := muestras[lengua]
		if !hay {
			t.Errorf("el corpus trae texto en %q y este test no sabe con que muestra "+
				"comprobarla.\n  No se da por buena: una lengua nueva en el corpus es "+
				"exactamente el caso que esta puerta existe para ver. Anade su muestra.",
				lengua)
			continue
		}
		if !latinaEntera(muestra) {
			fuera = append(fuera, lengua)
		}
	}
	for _, lengua := range fuera {
		t.Errorf("el corpus trae texto en %q y la tabla de plegado de "+
			"adaptadores/busqueda NO cubre su escritura.\n"+
			"  El sintoma seria silencioso: las busquedas sobre ese texto encontrarian "+
			"menos y nada se pondria rojo.\n"+
			"  Arreglo: o entra la dependencia de normalizacion Unicode "+
			"(golang.org/x/text, con su fila en DEPENDENCIAS.md) o crece la tabla.", lengua)
	}

	claves := make([]string, 0, len(lenguas))
	for l := range lenguas {
		claves = append(claves, l)
	}
	sort.Strings(claves)
	for _, l := range claves {
		t.Logf("  %s: %d obligacion(es) con texto", l, lenguas[l])
	}
	if conTexto < 100 {
		t.Fatalf("solo %d obligaciones con texto legal: el recorrido se ha roto y esta "+
			"puerta estaria midiendo el vacio", conTexto)
	}
}

// EL CONTROL NEGATIVO: el detector tiene que acusar una escritura no latina.
//
// Sin esto, la puerta de arriba pasaria con un `latinaEntera` que devolviera
// true siempre, que es justo lo que parece mientras el corpus sea castellano.
func TestElDetectorDeEscrituraNoLatinaAcusaYSeCalla(t *testing.T) {
	// SE CALLA con lo que la tabla si pliega.
	for _, m := range []string{"ñáéíóúü", "abcdefg", "àèìòù", "čćđšž"} {
		if !latinaEntera(m) {
			t.Errorf("el detector acusa %q, que es latin y la tabla lo pliega. Una puerta "+
				"que no se puede satisfacer se afloja", m)
		}
	}
	// ACUSA lo que no.
	for _, m := range []string{"αβγδε", "абвгд", "日本語", "العربية"} {
		if latinaEntera(m) {
			t.Errorf("el detector NO acusa %q, que esta fuera del latin. Corpus en esa "+
				"escritura entraria sin que nadie lo viera", m)
		}
	}
}

// latinaEntera dice si toda la muestra sobrevive al tokenizador como letras
// latinas, o sea si la tabla de plegado puede con ella.
func latinaEntera(muestra string) bool {
	ts := busqueda.Tokenizar(muestra)
	if len(ts) == 0 {
		return false
	}
	for _, t := range ts {
		for _, r := range t {
			// Despues de tokenizar y plegar, lo que la tabla cubre queda en
			// ASCII. Lo que no cubre sigue fuera, y por ahi se ve.
			if r > 127 {
				return false
			}
		}
	}
	return true
}
