package calendario

import "github.com/marcosmatalab/plazum/nucleo/pantalla"

// avisosDelMarco busca los avisos de UN marco por su URN.
//
// # Por que una busqueda y no un indice
//
// Porque emparejar por posicion es el ataque 13 (invariante 7): la lista de
// avisos y la lista de fechas se construyen por separado, y casarlas por indice
// haria que insertar un paquete moviera el aviso de NIS2 a otra norma sin que
// nada se pusiera rojo. `Marco` es `corpus.Paquete.URN`, el primer campo del
// fichero de datos y la identidad del paquete en todo el producto.
//
// LO QUE ESO GARANTIZA HOY, dicho sin adornos: los paquetes de corpus NO se
// firman, asi que el URN hereda la garantia del arbol de git y ninguna otra. Lo
// que si garantiza es que el emparejamiento no depende del orden, que es lo que
// esta regla existe para impedir.
//
// # Devuelve nil y no un aviso vacio
//
// Un marco sin bloque `transposicion` no es una directiva, y no decir nada es la
// respuesta correcta: un aviso en blanco al lado de una fecha se lee como que
// hay algo que mirar. El valor cero es callarse.
func avisosDelMarco(avisos []pantalla.AvisoDeMarco, marco string) []pantalla.AvisoDeMarco {
	var out []pantalla.AvisoDeMarco
	for _, a := range avisos {
		if a.Marco == marco {
			out = append(out, a)
		}
	}
	return out
}
