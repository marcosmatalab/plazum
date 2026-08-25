package pantalla

import (
	"strings"
	"testing"

	"dutiq/nucleo/corpus"
)

// La atribucion, derivada.
//
// QUE PROPIEDAD SE SOSTIENE AQUI: el aviso de derechos de cada paquete
// instalado llega al modelo de TODAS las pantallas, en orden estable y sin
// repetirse. Es la mitad de nucleo de la obligacion de atribuir; la otra mitad
// (que ademas se pinte) la vigila superficies/pantallas.
//
// Los paquetes son SINTETICOS (urn:demo:...). No pueden ser normas reales: el
// invariante 2 prohibe identificadores de norma en el codigo, tests incluidos.

func paqueteConAtribucion(urn, atribucion string) *corpus.Paquete {
	return &corpus.Paquete{
		URN: urn, Version: "1", Clase: corpus.Propio,
		LicenciaFuente: corpus.DelProyecto,
		Atribucion:     atribucion,
		Fuente:         "https://ejemplo.invalid/" + urn,
		Obligaciones: []corpus.Obligacion{{
			ID: urn + ".o", Articulo: "1", Cita: "demo art. 1", ClaseE2E: "documental",
		}},
	}
}

// TestLaAtribucionLlegaAlModeloDeTodasLasPantallas.
//
// Por que en todas y no solo en las que salen del corpus: el contenido del
// corpus se usa en tres de ellas, la obligacion acompana al uso, y una regla con
// excepciones es una regla que alguien aplica mal el dia que anade la septima
// pantalla. Si manana Derivar devuelve una pantalla mas, este test se la exige
// sin que nadie tenga que acordarse.
func TestLaAtribucionLlegaAlModeloDeTodasLasPantallas(t *testing.T) {
	ps := []*corpus.Paquete{
		paqueteConAtribucion("urn:demo:uno", "Aviso de derechos del primero."),
		paqueteConAtribucion("urn:demo:dos", "Aviso de derechos del segundo."),
	}
	pantallas := Derivar(ps)
	if len(pantallas) == 0 {
		t.Fatal("sin pantallas no hay nada que comprobar")
	}
	for _, p := range pantallas {
		if len(p.Fuentes) != len(ps) {
			t.Fatalf("la pantalla %q lleva %d fuentes y hay %d paquetes instalados. Un "+
				"aviso de atribucion que no sale en una pantalla es un aviso que no se da",
				p.ID, len(p.Fuentes), len(ps))
		}
		for i, f := range p.Fuentes {
			if f.Atribucion == "" {
				t.Errorf("la pantalla %q trae la fuente %s sin atribucion", p.ID, f.URN)
			}
			if f.Enlace == "" {
				t.Errorf("la pantalla %q trae la fuente %s sin enlace a la fuente oficial, "+
					"que es justo lo que las condiciones de reutilizacion exigen citar",
					p.ID, f.URN)
			}
			if i > 0 && p.Fuentes[i-1].URN >= f.URN {
				t.Errorf("la pantalla %q trae las fuentes desordenadas (%q antes de %q): el "+
					"modelo dejaria de ser comparable byte a byte",
					p.ID, p.Fuentes[i-1].URN, f.URN)
			}
		}
	}
}

// Control negativo: se demuestra que la atribucion viene DEL PAQUETE y no de
// ningun sitio de aqui. Dos corpus que solo se diferencian en el texto del aviso
// tienen que dar dos modelos distintos.
//
// Sin esto, una derivacion que escribiera el aviso a mano (o que se lo tragara)
// pasaria el test de arriba sin despeinarse.
func TestControlNegativoLaAtribucionSaleDelPaqueteYNoDeAqui(t *testing.T) {
	uno := Derivar([]*corpus.Paquete{paqueteConAtribucion("urn:demo:uno", "Aviso A.")})
	otro := Derivar([]*corpus.Paquete{paqueteConAtribucion("urn:demo:uno", "Aviso B.")})

	if uno[0].Fuentes[0].Atribucion == otro[0].Fuentes[0].Atribucion {
		t.Fatal("cambiar el aviso del paquete no cambia el modelo, asi que la pantalla no " +
			"esta ensenando lo que el paquete declara: esta ensenando otra cosa")
	}
	if !strings.Contains(uno[0].Fuentes[0].Atribucion, "Aviso A") {
		t.Fatalf("el modelo no lleva el aviso del paquete: %q", uno[0].Fuentes[0].Atribucion)
	}
}

// Sin corpus instalado no hay a quien atribuir, y eso es correcto: la lista sale
// vacia y no con una fila fantasma.
func TestSinCorpusNoHayFuentesQueAtribuir(t *testing.T) {
	for _, p := range Derivar(nil) {
		if len(p.Fuentes) != 0 {
			t.Errorf("la pantalla %q lista %d fuentes sin corpus instalado", p.ID, len(p.Fuentes))
		}
	}
}

// El mismo paquete dos veces no da dos avisos. Cargar ya rechaza dos paquetes
// con el mismo URN, pero Derivar recibe la lista que le pasen: un aviso repetido
// se lee como un fallo del producto, y ademas rompe el orden estable.
func TestUnPaqueteRepetidoNoDuplicaSuAtribucion(t *testing.T) {
	p := paqueteConAtribucion("urn:demo:uno", "Aviso de derechos.")
	pantallas := Derivar([]*corpus.Paquete{p, p})
	if n := len(pantallas[0].Fuentes); n != 1 {
		t.Fatalf("dos veces el mismo paquete dan %d avisos y tiene que dar 1", n)
	}
}

// El orden de las fuentes no puede depender de como recorrio el directorio el
// cargador: si dependiera, el caso dorado se pondria rojo un dia sin que nadie
// hubiera tocado nada, y a partir de ahi se ignora.
func TestLasFuentesNoDependenDelOrdenDeLosPaquetes(t *testing.T) {
	a := paqueteConAtribucion("urn:demo:uno", "Aviso uno.")
	b := paqueteConAtribucion("urn:demo:dos", "Aviso dos.")
	directo := Derivar([]*corpus.Paquete{a, b})[0].Fuentes
	alreves := Derivar([]*corpus.Paquete{b, a})[0].Fuentes
	if len(directo) != len(alreves) {
		t.Fatalf("%d fuentes en un orden y %d en el otro", len(directo), len(alreves))
	}
	for i := range directo {
		if directo[i] != alreves[i] {
			t.Fatalf("la fuente %d cambia con el orden de entrada: %+v vs %+v",
				i, directo[i], alreves[i])
		}
	}
}
