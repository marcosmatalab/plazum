package plazum

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

const rutaDelBacklog = "docs/pendientes.md"

// reFilaDeBacklog casa una fila de la tabla: `| 12 | lo que queda |`.
var reFilaDeBacklog = regexp.MustCompile(`(?m)^\|\s*(\d+)\s*\|\s*(.+?)\s*\|`)

// LOS TRES CARDINALES DEL BACKLOG SALEN DE SU PROPIA TABLA.
//
// # Que cambio, y por que la puerta tuvo que cambiar con ello
//
// Hasta el 22-09-2026 estos tres numeros se derivaban del relato: un fichero de
// 2.862 lineas con los items numerados dentro. Al archivar ese relato fuera del
// arbol, una puerta que lo leyera habria pasado a depender de la historia de
// git, o sea que en un clon superficial se habria saltado en silencio y los
// cardinales se habrian quedado sin vigilancia justo donde mas se miran.
//
// Asi que el backlog pasa a ser AUTOCONTENIDO: una fila por elemento abierto, y
// la cuenta sale de contar filas. Es mas barato de vigilar y, sobre todo, es lo
// que hace falta para que el fichero sirva de backlog: antes habia que leer
// 2.862 lineas para saber que queda.
//
// # Por que estos tres numeros necesitan puerta
//
// Porque su fallo probable FAVORECE. Lo que pasa solo es que alguien cierre un
// elemento, lo quite de la tabla y no baje la cuenta de la cabecera: entonces la
// lista parece mas larga de lo que es, que se lee como honestidad y es lo
// contrario. En la direccion opuesta pasa lo mismo: anadir una fila sin subir la
// cuenta esconde deuda.
func TestLosCardinalesDelBacklogSalenDeLaTabla(t *testing.T) {
	doc := leerFichero(t, rutaDelBacklog)

	p0 := filasDeSeccion(t, doc, "## P0, bloqueantes", "## P1, dentro de la etapa")
	p1 := filasDeSeccion(t, doc, "## P1, dentro de la etapa", "## P2, deuda conocida")
	p2 := filasDeSeccion(t, doc, "## P2, deuda conocida", "## El relato completo")

	if p0 == 0 || p1 < 5 || p2 < 5 {
		t.Fatalf("he contado %d P0, %d P1 y %d P2: con esos numeros el lector de filas ha "+
			"dejado de casar y esto esta midiendo el vacio", p0, p1, p2)
	}

	frase := fmt.Sprintf("**%d elementos: %d P0, %d P1 y %d P2.**", p0+p1+p2, p0, p1, p2)
	if !strings.Contains(doc, frase) {
		t.Errorf(`la cabecera del backlog no cuadra con su tabla.

  De las filas salen %d elementos: %d P0, %d P1 y %d P2.

  El fallo probable de esta cifra FAVORECE: lo que pasa solo es que alguien
  cierre un elemento, lo quite de la tabla y no baje la cuenta, y entonces la
  lista parece mas larga de lo que es.

  Arreglo: poner esta frase, tal cual, en %s:
    %s`, p0+p1+p2, p0, p1, p2, rutaDelBacklog, frase)
	}
	t.Logf("backlog: %d elementos (%d P0, %d P1, %d P2)", p0+p1+p2, p0, p1, p2)
}

// NINGUNA FILA DEL BACKLOG SE QUEDA SIN DECIR QUE ES.
//
// Una tabla de ochenta filas es util mientras cada fila diga algo; en cuanto
// admite «pendiente» o una celda vacia, vuelve a hacer falta abrir el relato,
// que es de donde veniamos. El suelo es corto a proposito: acusa la celda vacia
// y el marcador de posicion, no la brevedad.
func TestNingunaFilaDelBacklogEstaVacia(t *testing.T) {
	doc := leerFichero(t, rutaDelBacklog)
	vacias := 0
	for _, m := range reFilaDeBacklog.FindAllStringSubmatch(doc, -1) {
		texto := strings.TrimSpace(m[2])
		if len(texto) >= 15 && !strings.EqualFold(texto, "pendiente") && texto != "TODO" {
			continue
		}
		vacias++
		t.Errorf("la fila %s del backlog dice %q, que no dice nada.\n"+
			"  Una tabla que admite celdas vacias obliga a abrir el relato para saber que "+
			"queda, que es justo de donde venimos.", m[1], texto)
	}
	t.Logf("%d filas, %d sin contenido", len(reFilaDeBacklog.FindAllString(doc, -1)), vacias)
}

// EL CONTROL NEGATIVO del lector de filas.
//
// Su fallo probable es contar de mas: las tablas de cabecera del documento
// (niveles, destinos del archivo) tambien son tablas, y un lector que no acote
// por seccion las sumaria a los cardinales sin que se note, porque el numero
// seguiria pareciendo razonable.
func TestElLectorDeFilasDelBacklogAcusaYSeCalla(t *testing.T) {
	doc := "## A\n\n| # | Que |\n|---|---|\n| 1 | uno |\n| 2 | dos |\n\n" +
		"## B\n\n| # | Que |\n|---|---|\n| 7 | siete |\n\n## C\n"
	if n := filasDeSeccion(t, doc, "## A", "## B"); n != 2 {
		t.Errorf("cuento %d filas en A y hay 2", n)
	}
	if n := filasDeSeccion(t, doc, "## B", "## C"); n != 1 {
		t.Errorf("cuento %d filas en B y hay 1", n)
	}
	// Y la fila de separacion `|---|---|` NO es una fila de datos.
	if reFilaDeBacklog.MatchString("|---|---|") {
		t.Error("el lector cuenta la linea de separacion de la tabla como una fila")
	}
	// Ni la cabecera, que no empieza por un numero.
	if reFilaDeBacklog.MatchString("| # | Que |") {
		t.Error("el lector cuenta la cabecera de la tabla como una fila")
	}
}

// filasDeSeccion cuenta las filas numeradas entre dos cabeceras.
func filasDeSeccion(t *testing.T, doc, desde, hasta string) int {
	t.Helper()
	i := strings.Index(doc, desde)
	if i < 0 {
		t.Fatalf("%s no trae la seccion %q: si se renombro, este contador se quedo viejo "+
			"y estaria contando otra cosa", rutaDelBacklog, desde)
	}
	j := strings.Index(doc[i+len(desde):], hasta)
	if j < 0 {
		t.Fatalf("encuentro %q y no encuentro %q despues: sin cierre, el tramo se comeria "+
			"el resto del documento y contaria filas de otras secciones", desde, hasta)
	}
	return len(reFilaDeBacklog.FindAllString(doc[i:i+len(desde)+j], -1))
}
