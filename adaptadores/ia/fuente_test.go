package ia

import (
	"errors"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

func TestUnaFuenteSinProcedenciaNoSeConstruye(t *testing.T) {
	// EL VALOR CERO DE Procedencia ES `Ninguna`, y `Ninguna` NO significa
	// "cualquiera": significa "no lo has dicho". Es el invariante 8 en el lado
	// del dato, no en el de las opciones.
	_, err := NuevaFuente("x", "marco-de-prueba-1", "art. 1", "transcrito",
		Ninguna, true, "un texto cualquiera lo bastante largo para citarse")
	if !errors.Is(err, ErrFuenteSinProcedencia) {
		t.Fatalf("una fuente con el valor cero de Procedencia se ha construido: %v", err)
	}
	// Y una procedencia inventada tampoco.
	_, err = NuevaFuente("x", "marco-de-prueba-1", "art. 1", "transcrito",
		Procedencia(99), true, "un texto cualquiera lo bastante largo para citarse")
	if !errors.Is(err, ErrFuenteSinProcedencia) {
		t.Fatalf("Procedencia(99) se ha construido: %v", err)
	}
	// CONTROL POSITIVO en las dos direcciones declaradas.
	for _, p := range []Procedencia{Corpus, Aportado} {
		if _, err := NuevaFuente("x", "m", "a", "transcrito", p, true, "texto"); err != nil {
			t.Errorf("la procedencia %v no se acepta: %v", p, err)
		}
	}
}

func TestUnaFuenteSinIDNoSeConstruye(t *testing.T) {
	if _, err := NuevaFuente("", "m", "a", "transcrito", Corpus, true, "texto"); !errors.Is(err, ErrFuenteSinID) {
		t.Fatalf("una fuente sin ID se ha construido: %v", err)
	}
}

// LA FRONTERA LEGAL SE DECIDE POR LA CLASE DEL PAQUETE, no por una lista de
// marcos escrita en Go.
//
// Es la pregunta que faltaba en la pasada 1: un paquete referencial nuevo tiene
// que nacer no citable SIN TOCAR CODIGO. Si hubiera una lista de URN aqui,
// anadir el marco 31 seria un cambio de producto, y ademas romperia el
// invariante 2.
func TestLaCitabilidadSaleDeLaClaseYNoDeUnaListaDeMarcos(t *testing.T) {
	casos := []struct {
		clase   corpus.Clase
		citable bool
	}{
		{corpus.Importado, true},    // dominio publico, se distribuye entero
		{corpus.Transcrito, true},   // BOE y DOUE con la reutilizacion cumplida
		{corpus.Propio, true},       // datos del proyecto, sin tercero con derechos
		{corpus.Referencial, false}, // identificadores y mapeo, SIN texto normativo
		{corpus.Delegado, false},    // no se distribuye nada
	}
	for _, c := range casos {
		if got := ClaseCitable(c.clase); got != c.citable {
			t.Errorf("clase %v: citable=%v, se esperaba %v", c.clase, got, c.citable)
		}
	}
	// Una clase fuera de rango (que llega de un JSON de un tercero) NO es
	// citable. Es el valor cero restrictivo aplicado a un uint8 que viene de
	// fuera: lo desconocido no se ensena.
	if ClaseCitable(corpus.Clase(99)) {
		t.Error("una clase invalida sale citable. Un valor que llega de un fichero ajeno y " +
			"no se reconoce no puede caer del lado permisivo")
	}
}

func TestUnaObligacionSinTextoNoEsCitableAunqueSuClaseLoSea(t *testing.T) {
	f, err := NuevaFuente("x", "m", "a", "transcrito", Corpus, true, "")
	if err != nil {
		t.Fatal(err)
	}
	if f.Citable {
		t.Error("una fuente con el texto vacio sale citable. No hay nada que citar, y una " +
			"cita vacia contra un texto vacio casaria con cualquier cosa")
	}
}

// Documentos es el filtro que decide QUE VE EL MODELO.
func TestDocumentosDejaFueraLoQueNoSePuedeCitar(t *testing.T) {
	tr, re, pr, ap := corpusDePrueba(t)
	docs := Documentos([]Fuente{tr, re, pr, ap})
	if len(docs) != 3 {
		t.Fatalf("%d documentos indexables de 4 fuentes; el referencial tenia que quedarse "+
			"fuera y solo el", len(docs))
	}
	for _, d := range docs {
		if d.ID == re.ID {
			t.Fatal("el paquete referencial ha entrado al indice de busqueda.\n" +
				"  Lo que entra al indice acaba en el contexto del modelo, y lo que esta en\n" +
				"  el contexto sale parafraseado en alguna propuesta. Lo que no entra no\n" +
				"  puede salir por ningun lado.")
		}
		if d.Hash == "" {
			t.Errorf("el documento %s sale sin hash y no se podria citar", d.ID)
		}
	}
	// Las dos formas de la nada, tambien aqui.
	if n := len(Documentos(nil)); n != 0 {
		t.Errorf("Documentos(nil) da %d", n)
	}
	if n := len(Documentos([]Fuente{})); n != 0 {
		t.Errorf("Documentos(vacio) da %d", n)
	}
}

// normalizar tiene que devolver un mapa que recorte EL TROZO EXACTO del
// original. Si el mapa se desalinea aunque sea una runa, la pantalla resalta
// media palabra y ensena texto cortado con la cara de una cita.
func TestElMapaDeNormalizacionRecortaElTrozoExacto(t *testing.T) {
	casos := []string{
		"sin espacios raros",
		"con   espacios    multiples",
		"  con espacios al principio",
		"con espacios al final   ",
		"con\nsaltos\n  de linea\ty tabuladores",
		"\n\n  todo junto  \n\tal final\n",
		"acentos precompuestos: notificación y año",
	}
	for _, texto := range casos {
		normal, mapa := normalizar(texto)
		runasNormal := []rune(normal)
		runasOrigen := []rune(texto)
		if len(mapa) != len(runasNormal) {
			t.Fatalf("%q: el mapa tiene %d entradas y la forma normal %d runas",
				texto, len(mapa), len(runasNormal))
		}
		// Cada runa NO-espacio de la forma normal tiene que ser la misma que la
		// runa del original a la que apunta.
		for i, r := range runasNormal {
			if r == ' ' {
				continue
			}
			if runasOrigen[mapa[i]] != r {
				t.Errorf("%q: la runa %d de la forma normal es %q y el mapa apunta a %q",
					texto, i, string(r), string(runasOrigen[mapa[i]]))
			}
		}
		if strings.HasPrefix(normal, " ") || strings.HasSuffix(normal, " ") {
			t.Errorf("%q: la forma normal no esta recortada: %q", texto, normal)
		}
		if strings.Contains(normal, "  ") {
			t.Errorf("%q: la forma normal tiene espacios dobles: %q", texto, normal)
		}
	}
}

// El detector de marcas combinantes es lo que permite dar el motivo CONCRETO
// cuando una cita no casa por composicion Unicode. Las dos cadenas van con
// escapes y no escritas tal cual: un acento combinante en el fuente es un
// caracter invisible pegado a otro, no se ve en un diff, y un editor que
// normalice el fichero cambiaria el significado del test sin que nadie lo note.
func TestElDetectorDeMarcasCombinantes(t *testing.T) {
	const precompuesta = "notificaci\u00f3n" // o con tilde, un solo caracter
	const combinante = "notificacio\u0301n"  // o + acento, dos caracteres

	if tieneMarcasCombinantes(precompuesta) {
		t.Error("un acento precompuesto se esta contando como marca combinante: el \n" +
			"motivo del descarte por NFD saldria en citas que no tienen ese problema")
	}
	if !tieneMarcasCombinantes(combinante) {
		t.Error("un acento combinante no se detecta: el motivo del descarte por NFD " +
			"nunca se daria y la persona no sabria que buscar")
	}
	if tieneMarcasCombinantes("") {
		t.Error("la cadena vacia trae marcas combinantes")
	}
	// Y las dos cadenas TIENEN que ser distintas byte a byte, o este test
	// estaria comparando una cosa consigo misma.
	if precompuesta == combinante {
		t.Fatal("las dos formas han quedado iguales en el fuente: el test no compara nada")
	}
}
