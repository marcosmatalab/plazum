package busqueda

import (
	"strings"
	"unicode"
)

// Tokenizar parte un texto en terminos comparables, de forma DETERMINISTA.
//
// Hace tres cosas y ninguna mas:
//
//  1. corta por todo lo que no es letra ni digito (Unicode, no ASCII).
//  2. pasa a minusculas.
//  3. pliega los diacriticos latinos a su letra base.
//
// POR QUE SE PLIEGAN LOS DIACRITICOS AQUI Y NO EN EL VERIFICADOR DE CITAS, que
// es la decision que importa de este fichero. Son dos preguntas distintas:
//
//	buscar     "que articulo habla de notificacion" tiene que encontrar
//	           "notificacion" y "notificacion" escrito con tilde. Quien teclea
//	           una consulta no siempre tiene el teclado de la norma, y una
//	           busqueda que no lo pliega no encuentra nada y parece rota.
//	citar      una cita que difiere del texto de la fuente EN UN SOLO
//	           CARACTER no es ese texto. Ahi no se pliega nada: se compara
//	           literal. Ver adaptadores/ia/verificador.go.
//
// Plegar en la busqueda es AMPLIAR lo que se encuentra, que como mucho cuesta
// un resultado de mas que la persona descarta. Plegar en la cita seria
// ESTRECHAR lo que se rechaza, que es aceptar un texto que el modelo escribio
// distinto. La misma operacion, y en un lado ayuda y en el otro es un agujero.
//
// La enye se pliega a la ene, igual que hace el tokenizador `unicode61` de FTS5
// con remove_diacritics=2. En castellano son letras distintas y esto las junta:
// se acepta a proposito, porque quien busca "espana" quiere encontrar "Espana"
// con enye, y el coste es que "ano" y "ano" caen en el mismo termino. Es un
// coste de RECALL al alza en un buscador, no un fallo de correccion en una
// cita.
func Tokenizar(texto string) []string {
	campos := strings.FieldsFunc(texto, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := make([]string, 0, len(campos))
	for _, c := range campos {
		t := normalizarTermino(c)
		if t == "" {
			continue
		}
		out = append(out, t)
	}
	return out
}

// plegado es la tabla de diacriticos latinos que se pliegan a su letra base.
//
// Va como tabla explicita y no como llamada a una biblioteca de normalizacion
// Unicode POR UNA RAZON DE DEPENDENCIAS: la normalizacion NFD de verdad vive en
// golang.org/x/text, y este binario se compila con cero dependencias externas
// (DEPENDENCIAS.md). Una tabla de 60 entradas cubre el latin de la UE, se lee
// entera de un vistazo y no arrastra un modulo.
//
// LO QUE ESTA TABLA NO CUBRE, dicho para que conste y no se confunda con
// normalizacion Unicode completa: el griego, el cirilico y las lenguas que la
// UE tambien publica en el DOUE. El dia que entre corpus en esas lenguas, esto
// deja de bastar y hay que decidir entre la dependencia y una tabla mayor. Hoy
// el corpus es castellano e ingles, medido, y esto lo cubre entero.
const plegado = "" +
	"àa áa âa ãa äa åa " +
	"èe ée êe ëe " +
	"ìi íi îi ïi " +
	"òo óo ôo õo öo øo " +
	"ùu úu ûu üu " +
	"ýy ÿy " +
	"ñn çc " +
	"ăa ąa ćc čc ďd đd ęe ěe ģg " +
	"īi įi ıi ķk ļl ľl łl ńn ņn ňn " +
	"őo ōo ŕr ŗr řr śs şs šs ţt ťt " +
	"űu ųu ūu ůu źz żz žz"

// pliegues se construye una vez, al arrancar el binario.
var pliegues = construirPliegues()

func construirPliegues() map[rune]rune {
	m := map[rune]rune{}
	for _, par := range strings.Fields(plegado) {
		rs := []rune(par)
		if len(rs) != 2 {
			// Una entrada mal escrita en la tabla NO se salta en silencio: se
			// salta y ya, porque este mapa se construye antes de que exista un
			// *testing.T al que decirselo. Lo que lo caza es
			// TestLaTablaDePlegadoEstaBienFormada, que la recorre entera.
			continue
		}
		m[rs[0]] = rs[1]
	}
	return m
}

func normalizarTermino(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.ToLower(s) {
		if p, ok := pliegues[r]; ok {
			r = p
		}
		b.WriteRune(r)
	}
	return b.String()
}
