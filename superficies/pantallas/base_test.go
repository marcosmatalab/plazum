package pantallas

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

// El prefijo de montaje no puede sacar al visitante del sitio.
//
// Sale del primer hallazgo de CodeQL sobre el repositorio ya publico
// (`go/bad-redirect-check`, 28-08-2026). El porque completo, en base.go.
//
// EL TEST TIENE DOS MITADES Y LA SEGUNDA ES LA QUE IMPORTA. La primera prueba
// el validador contra una lista, y una lista escrita al lado del test solo
// prueba lo que su autor penso. La segunda es la propiedad: **de todo Base que
// el constructor ACEPTE, la redireccion de la raiz sale local**. Esa se
// sostiene aunque manana alguien invente una forma de escribirlo que no este
// en la lista, porque no pregunta como es el prefijo sino a donde acaba
// mandando al navegador, que es lo unico que se queria proteger.
func TestElPrefijoDeMontajeNoPuedeSacarteDelSitio(t *testing.T) {
	rechazables := []struct{ base, porque string }{
		{"ui", "sin barra inicial no es una ruta absoluta"},
		{"/ui/", "con barra final la concatenacion sale con dos"},
		{"/", "la raiz se pide con la cadena vacia"},
		{"//evil.com", "URL relativa al protocolo: OTRO DOMINIO"},
		{"//evil.com/x", "lo mismo con camino detras"},
		{`/\evil.com`, "el navegador normaliza la barra invertida a barra"},
		{`/\\evil.com`, "y la doble tambien"},
		{"/ui\\panel", "una barra invertida en medio sigue siendo normalizable"},
		{"/ /evil.com", "el navegador quita el espacio y quedan dos barras"},
		{"/\t/evil.com", "y el tabulador"},
		{"/\n/evil.com", "y el salto de linea, que ademas parte cabeceras"},
		{"/\x00ui", "el nulo, que trunca en cuanto lo toca algo escrito en C"},
	}
	for _, c := range rechazables {
		if err := validarBase(c.base); err == nil {
			t.Errorf("validarBase(%q) lo da por bueno, y %s", c.base, c.porque)
		} else if !errors.Is(err, ErrBase) {
			t.Errorf("validarBase(%q) falla sin el centinela ErrBase: %v", c.base, err)
		}
	}

	// Los que tienen que valer. Sin este tramo, un validador que rechace TODO
	// pasaria el de arriba con nota.
	for _, base := range []string{"/ui", "/a/b/c", "/UI-2", "/ui.v2"} {
		if err := validarBase(base); err != nil {
			t.Errorf("validarBase(%q) lo rechaza y es un prefijo legitimo: %v", base, err)
		}
	}

	// La propiedad, extremo a extremo: lo que el constructor acepta, redirige
	// dentro del sitio. Se recorren TODOS los casos, los buenos y los malos:
	// de los malos se exige que no lleguen a construirse, y de los que se
	// construyan (base vacia incluida) se exige que su Location sea local.
	todos := []string{""}
	for _, c := range rechazables {
		todos = append(todos, c.base)
	}
	for _, b := range []string{"/ui", "/a/b/c", "/UI-2", "/ui.v2"} {
		todos = append(todos, b)
	}
	construidos := 0
	for _, base := range todos {
		s, err := Nuevo(Opciones{Catalogo: nuevoCatalogo(), Paquetes: corpusDemo(), Base: base})
		if err != nil {
			continue // rechazado en la puerta: no llega a redirigir
		}
		construidos++
		rec, _ := pedir(t, s, "/")
		if rec.Code != http.StatusSeeOther {
			t.Fatalf("Base=%q: la raiz devuelve %d y no una redireccion, asi que esta "+
				"comprobacion estaria mirando el vacio", base, rec.Code)
		}
		loc := rec.Header().Get("Location")
		if !esRutaLocal(loc) {
			t.Errorf("Base=%q se acepto y la raiz redirige a %q, que saca al visitante "+
				"del sitio", base, loc)
		}
	}
	// Suelo: si Nuevo empezara a fallar por cualquier otra razon, el bucle de
	// arriba no comprobaria ni un Location y saldria verde.
	if construidos < 5 {
		t.Fatalf("solo se han construido %d superficies de %d bases: la propiedad no se "+
			"ha llegado a comprobar", construidos, len(todos))
	}
}

// esRutaLocal es la pregunta del navegador, no la del validador: ¿esto se
// resuelve contra este sitio? Se escribe aparte a proposito, para no medir el
// codigo con la misma regla que lo produjo.
func esRutaLocal(loc string) bool {
	if !strings.HasPrefix(loc, "/") {
		return false
	}
	if len(loc) > 1 && (loc[1] == '/' || loc[1] == '\\') {
		return false
	}
	return !strings.Contains(loc, `\`)
}
