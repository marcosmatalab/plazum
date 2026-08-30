// Package redactado reduce una URL de configuracion a lo unico que se puede
// decir de ella en un error, en un log o en un informe: su anfitrion.
//
// POR QUE EXISTE. Una URL que configura el operador PUEDE SER UNA CREDENCIAL, y
// cuales lo son no es una propiedad que el codigo pueda saber: el webhook de
// Teams es una URL que basta tener para escribir en ese canal; el destino de un
// "dead man's switch" (healthchecks.io y equivalentes) lleva el identificador
// secreto en la ruta; una TSA de pago lleva el token en la consulta. Las tres se
// ven exactamente igual que una URL publica.
//
// Por eso la regla no es "redacta las que sean secretas", que obliga a acertar,
// sino "no sale ninguna entera", que no obliga a nada. El anfitrion es lo que
// hace falta para diagnosticar — con quien no se pudo hablar — y es lo unico que
// no puede ser el secreto.
//
// DONDE ACABAN ESTOS ERRORES, que es lo que lo hace importante: en el log del
// operador, en la pantalla, y sobre todo en el bloque copiable de `plazum doctor
// --issue`, que esta hecho PARA PEGARLO EN UN ISSUE PUBLICO. Un token que llegue
// ahi lo publica quien pide ayuda.
//
// LO QUE ESTE PAQUETE NO ARREGLA, dicho para que no se confunda con una
// garantia: un error de `http.Client` LLEVA LA URL ENTERA DENTRO, asi que
// envolverlo con %w la filtra igual aunque el mensaje de fuera este redactado.
// Contra eso no hay funcion: hay no envolverlo. La comprobacion que si lo caza
// es la de centinela, que planta un valor en la URL configurada y recorre los
// caminos de fallo.
package redactado

import (
	"net/url"
	"strings"
)

// SinAnfitrion es lo que se dice cuando ni siquiera se puede extraer el
// anfitrion. Es una cadena y no un vacio a proposito: un error que dice "no
// puedo hablar con " y se corta ahi parece un error del producto.
const SinAnfitrion = "un destino ilegible"

// Anfitrion devuelve el anfitrion (con puerto si lo trae) de una URL de
// configuracion. Nunca devuelve la ruta, ni la consulta, ni el fragmento, ni el
// usuario, que son los tres sitios donde vive un secreto en una URL.
//
// POR QUE NO VALE url.Redacted(). Redacted solo sustituye la CONTRASENA del
// userinfo por "xxxxx" y deja intactos ruta, consulta y fragmento, que es
// justamente donde los servicios de webhook y de ping ponen su secreto. Usarlo
// creyendo que redacta es peor que no usar nada, porque parece hecho.
func Anfitrion(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return SinAnfitrion
	}
	return u.Host
}
