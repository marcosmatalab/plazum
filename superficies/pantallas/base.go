package pantallas

// El prefijo de montaje, y por que la barra inicial no basta.
//
// EL HALLAZGO. Lo saco CodeQL en su PRIMERA pasada sobre el repositorio ya
// publico, el 28-08-2026 (`go/bad-redirect-check`, medium):
//
//	This is a check that this value, which flows into a redirect, has a leading
//	slash, but not that it does not have '/' or '\' in its second position.
//
// La comprobacion que habia era `HasPrefix(base, "/") && !HasSuffix(base, "/")`.
// Y el valor va a parar, concatenado, a `http.Redirect` en el manejador de la
// raiz. Para un navegador `//evil.com` NO es una ruta local: es una URL
// relativa al protocolo, o sea otro dominio. `/\evil.com` consigue lo mismo en
// los navegadores que normalizan la barra invertida a barra, que son todos.
//
// LO QUE ESTO ES, Y LO QUE NO ES. `Base` la pone el operador en su propio
// arranque (`plazum serve --base`), no llega de una peticion, asi que no hay
// aqui un atacante remoto: quien escribe `--base=//evil.com` se lo hace a si
// mismo. Decir otra cosa seria inflar el hallazgo.
//
// LO QUE SI ES, Y POR ESO SE ARREGLA: **el mensaje de error prometia mas de lo
// que la comprobacion cumplia.** Decia "usa algo como /ui", o sea afirmaba un
// contrato -- esto es una ruta local -- que el codigo no verificaba. Es
// exactamente la leccion de M14 en este repositorio: una afirmacion de
// proteccion es una claim, y las claims se verifican. Y el contrato importa
// aunque hoy el unico que lo escribe sea el operador, porque `Base` es
// justamente el tipo de valor que acaba saliendo de una plantilla de
// despliegue, de una variable de entorno o de un chart, y ahi ya no lo teclea
// una persona que sepa lo que significa.
//
// COMO QUEDA LA REGLA: una ruta local absoluta empieza por una barra y sigue
// por algo que no sea barra ni barra invertida. Ademas se rechaza la barra
// invertida en cualquier posicion (un prefijo de montaje no la lleva nunca, y
// los navegadores la normalizan) y todo caracter de control o espacio, porque
// los navegadores los eliminan antes de resolver la URL y `/ /evil.com` acaba
// siendo `//evil.com`.

import (
	"fmt"
	"strings"
)

// validarBase comprueba que el prefijo de montaje es una ruta local absoluta.
// Devuelve siempre un error envuelto en ErrBase.
func validarBase(base string) error {
	malo := func(porque string) error {
		return fmt.Errorf("%w: %q, %s. Arreglo: usa \"\" para montar en la raiz, o algo "+
			"como \"/ui\": barra inicial, sin barra final, y el segundo caracter ni barra "+
			"ni barra invertida. El prefijo se concatena y se entrega a http.Redirect, y "+
			"para un navegador \"//lo-que-sea\" no es una ruta de este sitio sino otro "+
			"dominio", ErrBase, base, porque)
	}
	if !strings.HasPrefix(base, "/") {
		return malo("le falta la barra inicial")
	}
	if strings.HasSuffix(base, "/") {
		return malo("sobra la barra final")
	}
	// La segunda barra es la que convierte la ruta en una URL relativa al
	// protocolo. Con `base == "/"` no llegamos aqui: lo ha parado la barra
	// final, que en ese caso es tambien la inicial.
	if len(base) > 1 && (base[1] == '/' || base[1] == '\\') {
		return malo("el segundo caracter es una barra, y \"//host\" es otro dominio")
	}
	if strings.Contains(base, `\`) {
		return malo("lleva una barra invertida, que el navegador normaliza a barra")
	}
	for i := 0; i < len(base); i++ {
		if base[i] <= ' ' || base[i] == 0x7f {
			return malo(fmt.Sprintf("lleva un caracter de control o un espacio en la "+
				"posicion %d, y el navegador los quita antes de resolver la URL", i))
		}
	}
	return nil
}
