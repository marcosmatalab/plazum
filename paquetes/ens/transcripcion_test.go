package ens

import (
	"strings"
	"testing"
	"unicode"

	"plazum/nucleo/corpus"
)

// La puerta de la transcripcion: un texto legal en castellano al que le faltan
// las tildes NO es el texto legal.
//
// POR QUE HACE FALTA. El estrato "transcrito" del corpus se sostiene sobre
// reproducir con fidelidad citando la fuente (art. 13 TRLPI para el BOE,
// Decision 2011/833/UE para el DOUE). Los paquetes rgpd y cra llevaban su texto
// con los diacriticos comidos, seguramente por haber pasado por un filtro que
// los quita, y ninguna puerta se entero: el linter mira la clase, la longitud y
// la cita, y la ortografia no la mira nadie.
//
// DONDE TENDRIA QUE VIVIR ESTO. En el linter de nucleo/corpus, aplicado a todo
// paquete de clase Transcrito. Vive aqui, y acotado a los tres paquetes de este
// frente, porque nucleo/corpus lo esta cambiando otro frente ahora mismo y una
// puerta compartida no se cuela desde un worktree. Consta en el informe.
//
// COMO SE COMPRUEBA, y esto es lo que hace que no sea una heuristica. Las dos
// reglas de abajo son ortografia del castellano sin excepcion ni homografo:
//
//  1. una palabra que termina en -cion o en -sion es aguda acabada en n, asi
//     que lleva tilde SIEMPRE (notificacion no existe, notificación si). El
//     plural NO entra: naciones y obligaciones son llanas acabadas en s y van
//     sin tilde, que es justo la trampa de escribir la regla a lo bruto.
//  2. una lista corta de formas que en castellano solo existen con tilde y no
//     tienen ninguna palabra distinta detras que se escriba igual. Se han
//     dejado FUERA a proposito las ambiguas de verdad: mas (adverbio frente a
//     conjuncion), esta (demostrativo frente a verbo), practica y politica
//     (nombre frente a verbo), y las formas en -ara de los verbos en -ar, que
//     son futuro con tilde e imperfecto de subjuntivo sin ella.
var transcritosDeEsteFrente = map[string]bool{
	"urn:es:rd:2022:311":   true, // ENS
	"urn:eu:reg:2016:679":  true, // RGPD
	"urn:eu:reg:2024:2847": true, // CRA
}

// sinTildeYSiempreConElla son formas que en castellano no existen sin tilde.
var sinTildeYSiempreConElla = map[string]bool{
	"despues": true, "traves": true, "asi": true, "aqui": true, "segun": true,
	"tambien": true, "ademas": true, "dia": true, "dias": true,
	"articulo": true, "articulos": true, "minimo": true, "minimos": true,
	"maximo": true, "maximos": true, "unico": true, "unica": true,
	"unicos": true, "unicas": true, "tecnico": true, "tecnica": true,
	"tecnicos": true, "tecnicas": true, "fisico": true, "fisica": true,
	"fisicos": true, "fisicas": true, "electronico": true, "electronica": true,
	"electronicos": true, "electronicas": true, "sera": true, "seran": true,
	"debera": true, "deberan": true, "podra": true, "podran": true,
	"tendra": true, "tendran": true, "hara": true, "haran": true,
}

// palabras parte el texto en palabras: rachas de letras, apostrofes y guiones
// aparte. Se hace a mano y no con una expresion regular porque \b de Go es ASCII
// y aqui el texto tiene justo lo que \b no sabe mirar.
func palabras(s string) []string {
	var out []string
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) {
			b.WriteRune(r)
			continue
		}
		if b.Len() > 0 {
			out = append(out, b.String())
			b.Reset()
		}
	}
	if b.Len() > 0 {
		out = append(out, b.String())
	}
	return out
}

// faltaLaTilde devuelve el motivo si la palabra esta escrita sin la tilde que le
// toca, y cadena vacia si esta bien.
func faltaLaTilde(p string) string {
	b := strings.ToLower(p)
	if strings.HasSuffix(b, "cion") || strings.HasSuffix(b, "sion") {
		return "termina en -cion o -sion, que es aguda acabada en n y lleva tilde siempre"
	}
	if sinTildeYSiempreConElla[b] {
		return "en castellano esa forma solo existe con tilde"
	}
	return ""
}

func TestElTextoTranscritoConservaSusTildes(t *testing.T) {
	ps, err := corpus.Cargar("..")
	if err != nil {
		t.Fatalf("el corpus publicado no carga: %v", err)
	}
	vistos := 0
	for _, p := range ps {
		if !transcritosDeEsteFrente[p.URN] {
			continue
		}
		if p.Clase != corpus.Transcrito {
			t.Errorf("%s ya no es de clase transcrito (%s): esta puerta habla de texto "+
				"reproducido literalmente, revisa si sigue teniendo sentido", p.URN, p.Clase)
			continue
		}
		vistos++
		for _, o := range p.Obligaciones {
			for _, w := range palabras(o.TextoLegal) {
				if motivo := faltaLaTilde(w); motivo != "" {
					t.Errorf("%s / %s: %q %s. Un texto legal sin sus diacriticos no es el "+
						"texto legal, y el estrato transcrito se sostiene sobre reproducirlo "+
						"con fidelidad citando la fuente %s", p.URN, o.ID, w, motivo, p.Fuente)
				}
			}
		}
	}
	if vistos != len(transcritosDeEsteFrente) {
		t.Fatalf("se esperaba comprobar %d paquetes transcritos y se comprobaron %d: "+
			"alguno se ha caido del corpus y esta puerta paso en vacio",
			len(transcritosDeEsteFrente), vistos)
	}
}

// Control negativo del detector: sobre texto escrito a mano tiene que cazar lo
// que debe y callar en lo que no. Sin esto, el verde de arriba no prueba que el
// detector mire nada.
func TestElDetectorDeTildesSaltaCuandoDebeYCallaCuandoDebe(t *testing.T) {
	caza := []string{
		"notificacion", "violacion", "dilacion", "decision", "prevision",
		"despues", "traves", "unica", "fisicas", "articulo",
	}
	for _, w := range caza {
		if faltaLaTilde(w) == "" {
			t.Errorf("%q esta sin tilde y el detector no lo caza", w)
		}
	}
	calla := []string{
		// Con su tilde puesta.
		"notificación", "violación", "decisión", "después", "través", "única",
		// Plurales llanos acabados en s: van SIN tilde y son la trampa.
		"notificaciones", "obligaciones", "naciones", "decisiones", "previsiones",
		// Ambiguas de verdad, fuera de la lista a proposito.
		"mas", "esta", "practica", "politica", "notificara", "publico",
		// Palabras corrientes que no terminan en -cion ni -sion.
		"seguridad", "sistema", "responsable", "riesgo", "horas",
	}
	for _, w := range calla {
		if motivo := faltaLaTilde(w); motivo != "" {
			t.Errorf("%q esta bien escrita y el detector la caza por %q. Un detector que "+
				"grita por todo se acaba desactivando", w, motivo)
		}
	}
}
