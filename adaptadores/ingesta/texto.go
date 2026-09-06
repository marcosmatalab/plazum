package ingesta

import "strings"

// LimiteFragmento es lo mas largo que puede ser una unidad citable, en runas.
//
// POR QUE HAY UN LIMITE, y por que este. Un fragmento es lo que acaba
// resolviendo una cita y lo que acaba viendo una persona en pantalla. Uno
// demasiado corto no dice nada por si solo («...y ademas.»); uno demasiado
// largo convierte «esto lo dice tu politica» en «esto esta en algun sitio de
// estas tres paginas», que es exactamente la clase de cita que no se puede
// comprobar de un vistazo. Un parrafo de politica de seguridad tipico cabe de
// sobra en esto.
const LimiteFragmento = 1200

// MinimoFragmento descarta las lineas que no son texto: numeros de pagina,
// separadores, una letra suelta de un encabezado partido.
const MinimoFragmento = 12

// fragmentarTexto parte texto plano en unidades citables por parrafos.
//
// Se parte por LINEA EN BLANCO y no por punto: en un documento de cumplimiento
// los puntos aparecen dentro de las referencias («art. 5.1.b»), asi que partir
// por punto produce fragmentos cortados en mitad de una cita legal, que es la
// peor forma posible de cortar en este dominio concreto.
func fragmentarTexto(t string) (frags []Fragmento, truncado bool) {
	t = strings.ReplaceAll(t, "\r\n", "\n")
	if len([]rune(t)) > MaximoExtraido {
		t = string([]rune(t)[:MaximoExtraido])
		truncado = true
	}
	orden := 0
	for _, bloque := range strings.Split(t, "\n\n") {
		for _, trozo := range partirLargo(limpiar(bloque)) {
			if len([]rune(trozo)) < MinimoFragmento {
				continue
			}
			if len(frags) >= MaximoFragmentos {
				return frags, true
			}
			orden++
			frags = append(frags, Fragmento{Orden: orden, Texto: trozo})
		}
	}
	return frags, truncado
}

// limpiar colapsa los espacios de un bloque sin tocar su contenido.
func limpiar(s string) string { return strings.Join(strings.Fields(s), " ") }

// partirLargo corta un bloque que pasa del limite, por espacio y no por runa:
// cortar a mitad de palabra produce un fragmento cuya cita no casa con nada de
// lo que una persona buscaria.
func partirLargo(s string) []string {
	if len([]rune(s)) <= LimiteFragmento {
		if s == "" {
			return nil
		}
		return []string{s}
	}
	var out []string
	var actual []string
	n := 0
	for _, p := range strings.Fields(s) {
		l := len([]rune(p))
		if n+l+1 > LimiteFragmento && len(actual) > 0 {
			out = append(out, strings.Join(actual, " "))
			actual, n = nil, 0
		}
		// LA EXCEPCION, Y LA ENCONTRO EL FUZZ (06-09-2026, entrada minimizada en
		// testdata). Una «palabra» mas larga que el limite entero no es una
		// palabra: es una cadena sin espacios, que es lo que produce un PDF con
		// la codificacion equivocada, un base64 pegado o un ataque. Sin este
		// corte, el limite del fragmento no era un limite, y un fragmento sin
		// limite acaba siendo una cita que no se puede comprobar de un vistazo.
		for l > LimiteFragmento {
			rs := []rune(p)
			out = append(out, string(rs[:LimiteFragmento]))
			p = string(rs[LimiteFragmento:])
			l = len([]rune(p))
		}
		if p == "" {
			continue
		}
		actual = append(actual, p)
		n += l + 1
	}
	if len(actual) > 0 {
		out = append(out, strings.Join(actual, " "))
	}
	return out
}
