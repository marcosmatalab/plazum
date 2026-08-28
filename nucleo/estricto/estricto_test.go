package estricto

import (
	"errors"
	"strings"
	"testing"
)

type ficha struct {
	ID     string `json:"id"`
	Cuando string `json:"cuando_cambiarlo"`
	Sub    *sub   `json:"sub,omitempty"`
}

type sub struct {
	Dentro string `json:"dentro"`
}

// La decodificacion estricta para, dice que campo y dice donde.
//
// EL CASO 1 ES EL HALLAZGO REAL, con su nombre: `cuando_cambiarl`, una letra
// menos. Asi es como se escribe de verdad un campo que no existe, y asi de poco
// se nota: con el decodificador por defecto el fichero carga, la ficha queda
// con el campo a cero y todo lo demas funciona.
func TestUnCampoQueElFormatoNoDeclaraParaLaCarga(t *testing.T) {
	casos := []struct {
		nombre string
		json   string
		campo  string
		linea  string
	}{
		{
			nombre: "una letra menos en el nombre del campo",
			json:   "{\n  \"id\": \"a\",\n  \"cuando_cambiarl\": \"texto\"\n}",
			campo:  "cuando_cambiarl",
			linea:  "linea 3",
		},
		{
			nombre: "un campo inventado dentro de un objeto anidado",
			json:   "{\n  \"id\": \"a\",\n  \"sub\": {\n    \"dentroo\": \"x\"\n  }\n}",
			campo:  "dentroo",
			linea:  "linea 4",
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			var f ficha
			err := Decodificar([]byte(c.json), &f, "la ficha")
			if err == nil {
				t.Fatalf("ha cargado con %q dentro, y ese dato no llega a ninguna parte", c.campo)
			}
			if !errors.Is(err, ErrCampoDesconocido) {
				t.Fatalf("falla sin el centinela ErrCampoDesconocido: %v", err)
			}
			if !strings.Contains(err.Error(), c.campo) {
				t.Errorf("el error no NOMBRA el campo %q: %v", c.campo, err)
			}
			// El sitio, no solo el nombre: en un paquete.json de dos mil lineas
			// con 45 obligaciones, "cuando_cambiarl" a secas no se encuentra.
			if !strings.Contains(err.Error(), c.linea) {
				t.Errorf("el error no dice DONDE (%s): %v", c.linea, err)
			}
		})
	}
}

// Lo legitimo carga, y sin esto lo de arriba lo pasaria un decodificador que
// rechazara todo.
func TestLoQueElFormatoSiDeclaraCarga(t *testing.T) {
	var f ficha
	if err := Decodificar([]byte(`{"id":"a","cuando_cambiarlo":"texto","sub":{"dentro":"x"}}`),
		&f, "la ficha"); err != nil {
		t.Fatalf("un fichero legitimo no carga: %v", err)
	}
	if f.ID != "a" || f.Cuando != "texto" || f.Sub == nil || f.Sub.Dentro != "x" {
		t.Fatalf("ha cargado mal: %+v", f)
	}
}

// LO QUE VIENE DETRAS DEL VALOR, que es la trampa de cambiar Unmarshal por
// Decoder.
//
// `json.Unmarshal` rechaza el contenido sobrante; `Decoder.Decode` se para en
// cuanto tiene un valor completo y no mira mas. O sea que la migracion que se
// hace para GANAR DisallowUnknownFields regala, en el mismo movimiento, una
// forma nueva de tragar en silencio: dos objetos pegados, un merge mal resuelto
// o basura al final pasarian sin una linea de aviso.
func TestNoSeAceptaNadaDetrasDelValor(t *testing.T) {
	casos := []struct{ nombre, json string }{
		{"dos objetos pegados, que es un merge mal resuelto", `{"id":"a"}{"id":"b"}`},
		{"un objeto y basura detras", `{"id":"a"} y aqui lo que sea`},
		{"un objeto y un cierre suelto", `{"id":"a"} ]`},
		{"un objeto repetido en la linea siguiente", "{\"id\":\"a\"}\n{\"id\":\"b\"}\n"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			var f ficha
			err := Decodificar([]byte(c.json), &f, "la ficha")
			if err == nil {
				t.Fatalf("ha cargado %q quedandose solo con el primer valor, y eso es "+
					"indistinguible de que el fichero estuviera bien", c.json)
			}
			if !errors.Is(err, ErrSobraContenido) {
				t.Fatalf("falla sin el centinela ErrSobraContenido: %v", err)
			}
		})
	}
}

// El extractor del nombre del campo se prueba aparte, porque va por texto sobre
// el mensaje de encoding/json y ese es el punto que puede envejecer.
//
// La degradacion tiene que ser a MENOS UTIL, nunca a incorrecto: si algun dia
// el mensaje de la biblioteca cambia, esto deja de reconocerlo y el error sale
// con el texto crudo, que sigue nombrando el campo.
func TestElExtractorDelNombreDelCampo(t *testing.T) {
	casos := []struct {
		err    string
		quiero string
		ok     bool
	}{
		{`json: unknown field "cuando_cambiarlo"`, "cuando_cambiarlo", true},
		{`la ficha: json: unknown field "x"`, "x", true},
		{`json: cannot unmarshal string into Go value of type int`, "", false},
		{`json: unknown field "sin cerrar`, "", false},
		{``, "", false},
	}
	for _, c := range casos {
		campo, ok := campoDesconocido(errors.New(c.err))
		if ok != c.ok || campo != c.quiero {
			t.Errorf("campoDesconocido(%q) = %q,%v y esperaba %q,%v", c.err, campo, ok, c.quiero, c.ok)
		}
	}
}
