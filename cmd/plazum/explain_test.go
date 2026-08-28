package main

import (
	"bytes"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/catalogo"
	"github.com/marcosmatalab/plazum/nucleo/expediente"
)

// La puerta del descargo de `plazum explain`.
//
// Que vigila: que la salida que un operador imprime y se lleva a una reunion
// diga que esto NO es asesoramiento juridico. `explain` es la orden que mas se
// parece a un dictamen (dice que obligacion aplica, por que articulo y para que
// fecha) y es justo la que no lo es.
//
// Se comprueba sobre la salida REAL de cmdExplain y no sobre la constante: una
// constante bien escrita que nadie imprime es exactamente el fallo que esta
// puerta existe para cazar.

// expedienteDeEjemplo carga el expediente demo del repo. Si no esta, el test
// falla en vez de saltarse: un t.Skip aqui convierte esta puerta en un adorno el
// dia que alguien mueva el fichero.
func expedienteDeEjemplo(t *testing.T) *expediente.Expediente {
	t.Helper()
	ruta := filepath.Join("..", "..", "expediente-demo.json")
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("no encuentro %s: %v.\nSin expediente no hay nada que explicar y esta "+
			"puerta no comprobaria nada. Arreglo: ajusta la ruta si el fichero se movio", ruta, err)
	}
	e, err := expediente.Cargar(b)
	if err != nil {
		t.Fatalf("el expediente demo no carga: %v", err)
	}
	return e
}

func TestExplainTerminaConElDescargoDeAsesoramiento(t *testing.T) {
	var salida, errores bytes.Buffer
	if codigo := cmdExplain(expedienteDeEjemplo(t), &salida, &errores); codigo != 0 {
		t.Fatalf("cmdExplain devolvio %d: %s", codigo, errores.String())
	}
	texto := salida.String()

	// El fondo: que diga que no es asesoramiento juridico. Se buscan las dos
	// piezas por separado y NORMALIZADAS por espacios en blanco, porque el
	// texto va ajustado a 76 columnas y un salto de linea puede caer en medio.
	plano := strings.Join(strings.Fields(texto), " ")
	for _, trozo := range []string{"no presta asesoramiento jurídico"} {
		if !strings.Contains(plano, trozo) {
			t.Errorf("la salida de `plazum explain` no lleva el descargo (falta %q).\n"+
				"Esta orden dice que obligacion aplica, por que articulo y para que fecha, en un "+
				"tono que se parece a un dictamen y no lo es. El descargo lo imprime cmdExplain "+
				"al final, en cmd/plazum/explain.go.\n--- salida ---\n%s", trozo, texto)
		}
	}

	// Y que vaya al FINAL: un descargo entre los paquetes y los relojes se lee
	// como una nota al pie de la seccion anterior y no del resultado.
	if i := strings.Index(plano, "no presta asesoramiento"); i >= 0 && i < len(plano)/2 {
		t.Errorf("el descargo sale en el primer tramo de la salida (posicion %d de %d) y tiene "+
			"que cerrarla", i, len(plano))
	}

	// Por salida estandar, no por la de errores: quien hace
	// `plazum explain x.json > informe.txt` tiene que llevarselo dentro.
	if strings.Contains(errores.String(), "asesoramiento") {
		t.Error("el descargo se ha escrito por la salida de errores. Redirigir la salida a un " +
			"fichero lo dejaria fuera del informe, que es justo donde hace falta")
	}
}

// El descargo de reserva y el del catalogo dicen lo mismo.
//
// Una copia sin puerta se desvia el primer dia que alguien mejora la redaccion
// de una sola de las dos, y entonces hay dos descargos distintos para el mismo
// producto, que es un descargo que alguien tendra que interpretar.
func TestElDescargoDeReservaDiceLoMismoQueElCatalogo(t *testing.T) {
	cat, err := catalogo.Nuevo()
	if err != nil {
		t.Fatalf("el catalogo no carga: %v", err)
	}
	delCatalogo := cat.Traducir(cat.Idiomas()[0], claveDescargo)
	if delCatalogo == claveDescargo {
		t.Fatalf("el catalogo no tiene la clave %q: devuelve la clave en crudo. La orden "+
			"`plazum explain` estaria imprimiendo un identificador donde tiene que ir un "+
			"descargo", claveDescargo)
	}
	if delCatalogo != descargoDeReserva {
		t.Errorf("el descargo de reserva de cmd/plazum/explain.go se ha separado del catalogo.\n"+
			"  catalogo (%s): %q\n  reserva:        %q\n"+
			"Arreglo: copiar el del catalogo, que es el que ve la web, en descargoDeReserva.",
			cat.Idiomas()[0], delCatalogo, descargoDeReserva)
	}
}

// El descargo existe en TODOS los idiomas cargados, no solo en el de por
// defecto. Un idioma nuevo que se olvide el descargo dejaria a ese operador sin
// el, y Faltantes() del catalogo lo cazaria solo si alguien lo mira.
func TestElDescargoEstaEnTodosLosIdiomasDelCatalogo(t *testing.T) {
	cat, err := catalogo.Nuevo()
	if err != nil {
		t.Fatalf("el catalogo no carga: %v", err)
	}
	idiomas := cat.Idiomas()
	if len(idiomas) < 2 {
		t.Fatalf("el catalogo declara %d idioma(s). Con uno solo este test no compara nada",
			len(idiomas))
	}
	for _, idioma := range idiomas {
		v := cat.Traducir(idioma, claveDescargo)
		if v == "" || v == claveDescargo {
			t.Errorf("el idioma %q no tiene descargo: Traducir devuelve %q", idioma, v)
		}
	}
}

// La base de zonas horarias viaja dentro del binario.
//
// POR QUE ES UN TEST ESTATICO Y NO UNA COMPROBACION EN EJECUCION. Este test
// corre en maquinas que SI tienen zoneinfo (la de desarrollo, el runner de CI),
// y ahi time.LoadLocation("Europe/Madrid") funciona igual con el import puesto
// que quitado: una comprobacion en ejecucion daria verde en los dos casos, o
// sea no vigilaria nada. Lo que rompe es la imagen scratch, y eso lo comprueba
// el trabajo `imagen` de .github/workflows/etapa2-distribucion.yml ejecutando
// `plazum verify` DENTRO de ella. Aqui se vigila la causa; alli, el efecto.
//
// Lo que pasaba sin el import, medido en una imagen scratch de verdad:
//
//	DISCREPA   reloj de rgpd.art33.notificacion
//	           recalculado: zona horaria "Europe/Madrid": unknown time zone Europe/Madrid
//	NO VERIFICA: 6 discrepancia(s).
//
// O sea el verificador acusaba al emisor de haber falseado su expediente cuando
// el roto era el receptor.
func TestElCLITraeSuPropiaBaseDeZonasHorarias(t *testing.T) {
	fs := token.NewFileSet()
	paquetes, err := parser.ParseDir(fs, ".", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("no puedo leer el paquete: %v", err)
	}
	pkg, hay := paquetes["main"]
	if !hay {
		t.Fatal("no se ha encontrado el paquete main en cmd/plazum. Si el directorio se movio, " +
			"este test estaria mirando el vacio y dando verde")
	}
	for _, f := range pkg.Files {
		for _, imp := range f.Imports {
			if imp.Path != nil && imp.Path.Value == `"time/tzdata"` {
				if imp.Name == nil || imp.Name.Name != "_" {
					t.Errorf("time/tzdata se importa con nombre %v y tiene que ser un import "+
						"en blanco", imp.Name)
				}
				return
			}
		}
	}
	t.Errorf(`el paquete main de cmd/plazum NO importa _ "time/tzdata".
  Sin esa linea el binario resuelve las zonas horarias con /usr/share/zoneinfo del
  sistema, y una imagen minima (scratch, distroless-static) no lo trae. El efecto
  no es un error claro: es que plazum verify responde NO VERIFICA sobre un
  expediente correcto, o sea acusa al emisor de un fallo del receptor.
  Arreglo: devolver el import a cmd/plazum/zonas.go, donde esta el porque entero.`)
}
