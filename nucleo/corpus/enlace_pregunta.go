package corpus

import "sort"

// EL ENLACE ENTRE UNA PREGUNTA Y UNA OBLIGACION SE DECLARA DOS VECES, Y NADIE
// CRUZABA LAS DOS.
//
// El formato lo escribe en las dos puntas:
//
//	Pregunta.Desbloquea   IDs de obligacion que la pregunta dice abrir
//	Obligacion.Preguntas  IDs de pregunta que hay que responder para saber si
//	                      la obligacion alcanza al sujeto
//
// El linter comprueba UNA de las dos direcciones: que lo que una pregunta dice
// desbloquear existe. NO comprueba la vuelta, y la vuelta es la que decide:
// toda la derivacion de la pantalla de alcance se evalua contra
// `Obligacion.Preguntas`, y `Desbloquea` solo ordena la entrevista. Una
// pregunta puede por tanto declarar que abre catorce obligaciones y que ninguna
// de las catorce la nombre: la entrevista la pinta arriba del todo, con su
// «decide 14 obligaciones» escrito al lado, y responderla no mueve nada.
//
// Es el invariante 7 en su forma de siempre: dos conjuntos que se emparejan y
// una sola direccion recorrida. La que falta es la que miente.
//
// ESTO NO ES UN ERROR DEL LINTER, y por eso no se anade alli: hay 23 preguntas
// asi en el corpus instalado, y cada una es una de dos cosas (falta la regla, o
// sobra la pregunta) que exigen leer la norma. Lo que se hace es MEDIRLO y
// dejarlo contado, que es lo que convierte un hueco en algo que molesta hasta
// que se cierra.

// PreguntasQueNadieRequiere devuelve, ordenados, los IDs de las preguntas del
// corpus a las que NINGUNA obligacion apunta desde su campo `preguntas`.
//
// Se mira contra TODOS los paquetes juntos y no paquete a paquete: nada impide
// que una obligacion de un paquete se condicione a una pregunta de otro, y
// contarlo por paquete daria huerfanas que no lo son.
//
// El emparejamiento es POR EL ID DE LA PREGUNTA, que es el mismo campo que lee
// la evaluacion de la pantalla. Emparejar por cualquier otra cosa mediria una
// relacion que el producto no usa.
func PreguntasQueNadieRequiere(ps []*Paquete) []string {
	requeridas := map[string]bool{}
	for _, p := range ps {
		if p == nil {
			continue
		}
		for _, o := range p.Obligaciones {
			// Preguntas en nil y Preguntas vacio-presente son la misma cosa
			// aqui, y las dos son legitimas: una obligacion sin preguntas
			// alcanza a todo el mundo. El bucle no corre en ninguno de los dos.
			for _, id := range o.Preguntas {
				requeridas[id] = true
			}
		}
	}
	var out []string
	for _, p := range ps {
		if p == nil {
			continue
		}
		for _, q := range p.Preguntas {
			if !requeridas[q.ID] {
				out = append(out, q.ID)
			}
		}
	}
	sort.Strings(out)
	return out
}
