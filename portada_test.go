package plazum

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TechoDePalabrasDelREADME es lo que la portada puede ocupar.
//
// # El fallo que lo trae, con su cardinal
//
// El 22-09-2026 el README eran **2.617 palabras, 147 lineas y CERO imagenes**.
// Quien llega a este repositorio y le da sesenta segundos no llega al final del
// primer scroll, y lo que hay en ese scroll son tres parrafos sobre el tamano
// del binario. El producto tiene seis pantallas accesibles con axe-core en cero
// violaciones y **no se veia ninguna**.
//
// # Por que un techo y no una recomendacion
//
// Porque el README crece por el mismo motivo por el que crecio: cada cosa que
// se anade parece pequena al lado de lo que ya hay. Un techo no impide anadir,
// obliga a decidir que sale, que es la decision que nadie toma sola.
//
// El margen es amplio a proposito. Esta puerta NO esta para saltar en cada
// edicion: hoy son unas 970 palabras y el techo son 1.200, o sea que solo se
// pone roja cuando la portada ha vuelto a ser un documento. Una puerta que
// saltara en cada retoque ensenaria a esquivarla, y eso vale mas que la
// precision que perderia.
const TechoDePalabrasDelREADME = 1200

// MinimoDeCapturasEnElREADME es el suelo de imagenes de la portada.
//
// Va con suelo y no con techo porque el fallo medido fue por defecto: cero
// imagenes con veintidos capturas generadas y versionadas en el arbol, que las
// produce un test y que nadie veia nunca.
const MinimoDeCapturasEnElREADME = 3

var reImagenDelREADME = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

// LA PORTADA CABE EN UN MINUTO Y ENSENA EL PRODUCTO.
//
// # Lo que NO comprueba, dicho
//
// No comprueba que el README diga cosas ciertas: eso lo hacen, cada uno de lo
// suyo, el parrafo de ingenieria, los cuatro numeros del corpus y la tabla de
// las cifras. Esto comprueba las dos propiedades de la portada que no eran de
// ninguna de esas puertas y que por eso se cayeron las dos a la vez.
func TestLaPortadaCabeEnUnMinutoYEnsenaElProducto(t *testing.T) {
	b, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("no puedo leer el README: %v", err)
	}
	readme := string(b)

	if n := len(strings.Fields(readme)); n > TechoDePalabrasDelREADME {
		t.Errorf(`el README tiene %d palabras y el techo son %d.

  La decision que toca, y que este techo existe para que no se tome por inercia:
  lo que se acaba de anadir, ¿tiene que leerlo quien llega en sesenta segundos?
  Si la respuesta es no, su sitio es docs/ con un enlace desde aqui, que es lo
  que se hizo el 22-09-2026 con el presupuesto del binario y con la cobertura de
  la v1: los dos bloques se mudaron ENTEROS, con sus marcadores y sus puertas.`,
			n, TechoDePalabrasDelREADME)
	}

	imagenes := reImagenDelREADME.FindAllStringSubmatch(readme, -1)
	if len(imagenes) < MinimoDeCapturasEnElREADME {
		t.Errorf("el README ensena %d imagenes y el suelo son %d.\n"+
			"  El arbol tiene 22 capturas versionadas en superficies/pantallas/capturas/, "+
			"que las genera un test, y hasta el 22-09-2026 no se enlazaba ninguna.",
			len(imagenes), MinimoDeCapturasEnElREADME)
	}

	// Y CADA IMAGEN EXISTE Y LLEVA TEXTO ALTERNATIVO. Una portada con la imagen
	// rota es peor que una sin imagen, y un alt vacio deja fuera a quien usa
	// lector de pantalla, que en un producto que presume de axe-core en cero
	// violaciones seria contradecirse en la primera pagina.
	for _, m := range imagenes {
		alt, ruta := m[1], m[2]
		if strings.HasPrefix(ruta, "http") {
			continue
		}
		if _, err := os.Stat(ruta); err != nil {
			t.Errorf("el README enlaza la imagen %q y no existe: %v.\n"+
				"  Una captura rota en la portada es la primera impresion del repositorio.",
				ruta, err)
		}
		if len(strings.TrimSpace(alt)) < 10 {
			t.Errorf("la imagen %q lleva el texto alternativo %q, que no describe nada.\n"+
				"  Este producto publica axe-core en cero violaciones: contradecirlo en la "+
				"portada es la peor pagina donde hacerlo.", ruta, alt)
		}
	}
	t.Logf("%d palabras (techo %d), %d imagenes (suelo %d), todas existentes y con alt",
		len(strings.Fields(readme)), TechoDePalabrasDelREADME,
		len(imagenes), MinimoDeCapturasEnElREADME)
}

// EL CONTROL NEGATIVO: el lector de imagenes tiene que acusar y callarse.
//
// Sin esto, una expresion que no casara nada haria que el suelo de imagenes
// saltara siempre (eso se ve) o, peor, una que casara enlaces normales daria por
// imagen cualquier `[texto](ruta)` del README, que son una docena, y el suelo
// pasaria sin que hubiera ni una captura.
func TestElLectorDeImagenesDelREADMEAcusaYSeCalla(t *testing.T) {
	for _, c := range []struct {
		nombre string
		texto  string
		quiero int
	}{
		{"una imagen", "![el calendario en uso](a/b.png)", 1},
		{"un enlace normal no es una imagen", "[docs](docs/ia.md)", 0},
		{"enlace e imagen juntos", "[docs](docs/ia.md) y ![foto de algo](a.png)", 1},
		{"imagen dentro de un enlace", "[![insignia](a.svg)](https://ejemplo)", 1},
	} {
		if n := len(reImagenDelREADME.FindAllString(c.texto, -1)); n != c.quiero {
			t.Errorf("%s: cuento %d imagenes y esperaba %d en %q", c.nombre, n, c.quiero, c.texto)
		}
	}
	// Y LA RUTA QUE SACA ES LA RUTA, no el texto alternativo: si los grupos
	// estuvieran cambiados, la comprobacion de existencia miraria el alt y no
	// encontraria nunca el fichero, o sea que acusaria siempre.
	m := reImagenDelREADME.FindStringSubmatch("![el calendario en uso](a/b.png)")
	if m[1] != "el calendario en uso" || m[2] != "a/b.png" {
		t.Errorf("los grupos salen cambiados: alt=%q ruta=%q", m[1], m[2])
	}
}
