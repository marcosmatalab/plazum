package pantallas

import (
	"strings"
	"unicode/utf8"

	"plazum/nucleo/pantalla"
)

// Saneado del texto que llega del corpus.
//
// HALLAZGO DE LA PASADA DEL ATACANTE, y no era teorico: un paquete con bytes
// que no son UTF-8 valido (\xff\xfe) producia una pagina que tampoco lo era.
// Eso importa por una razon concreta: ante una secuencia invalida cada
// navegador resincroniza por donde quiere, y de ahi salen las inyecciones que
// esquivan un escapado por lo demas correcto. html/template escapa
// perfectamente y no arregla esto, porque no es un problema de escapado.
//
// Que se hace, y por que no es tocar la norma:
//
//	los bytes que no forman UTF-8 valido se sustituyen por U+FFFD. No eran
//	texto: eran bytes sueltos, y ningun texto normativo se pierde porque
//	ninguno estaba ahi.
//	los caracteres de control C0 (salvo tabulador y salto de linea) y el DEL
//	se quitan. No se ven, no dicen nada y sirven para partir cadenas de
//	comprobacion y para escribir secuencias de escape de terminal en un texto
//	que alguien puede acabar volcando a una consola.
//
// Se hace UNA VEZ, al derivar el modelo, y no en cada peticion: la derivacion
// es determinista y el corpus solo cambia al recargar.
//
// Deliberadamente NO se recorta ni se reescribe nada mas. Truncar el texto de
// un articulo en la pantalla donde se decide si se cumple es peor que una
// tabla ancha, y reescribirlo seria obra derivada.

// sanear deja una cadena del corpus en texto que un navegador lee igual que
// nosotros.
func sanear(s string) string {
	if s == "" {
		return s
	}
	if utf8.ValidString(s) && !tieneControl(s) {
		return s
	}
	s = strings.ToValidUTF8(s, "�")
	return strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' {
			return r
		}
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, s)
}

func tieneControl(s string) bool {
	for _, r := range s {
		if (r < 0x20 && r != '\t' && r != '\n') || r == 0x7f {
			return true
		}
	}
	return false
}

func sanearLista(xs []string) []string {
	for i, x := range xs {
		xs[i] = sanear(x)
	}
	return xs
}

// sanearPantallas recorre el modelo derivado entero. Los titulos y las claves
// de "por que esta vacia" NO se tocan: son claves de catalogo nuestras, no
// texto del corpus.
func sanearPantallas(ps []pantalla.Pantalla) []pantalla.Pantalla {
	for i := range ps {
		p := &ps[i]
		for j := range p.Preguntas {
			q := &p.Preguntas[j]
			q.ID, q.Texto, q.Ayuda = sanear(q.ID), sanear(q.Texto), sanear(q.Ayuda)
			q.Cita, q.Entidad, q.Atributo = sanear(q.Cita), sanear(q.Entidad), sanear(q.Atributo)
			q.Paquete = sanear(q.Paquete)
			q.Desbloquea = sanearLista(q.Desbloquea)
		}
		for j := range p.Campos {
			c := &p.Campos[j]
			c.Entidad, c.Atributo = sanear(c.Entidad), sanear(c.Atributo)
			c.Etiqueta, c.Tipo = sanear(c.Etiqueta), sanear(c.Tipo)
			c.Ayuda, c.Cita = sanear(c.Ayuda), sanear(c.Cita)
			c.Valores, c.Paquetes = sanearLista(c.Valores), sanearLista(c.Paquetes)
		}
		for j := range p.Filas {
			f := &p.Filas[j]
			f.ID, f.Paquete = sanear(f.ID), sanear(f.Paquete)
			f.Requiere = sanearLista(f.Requiere)
			if len(f.Columnas) > 0 {
				limpias := make(map[string]string, len(f.Columnas))
				for k, v := range f.Columnas {
					limpias[sanear(k)] = sanear(v)
				}
				f.Columnas = limpias
			}
		}
	}
	return ps
}
