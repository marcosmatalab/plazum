package plazum

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// LOS CUADERNOS DE HALLAZGOS TIENEN INDICE, Y EL INDICE VA EN LAS DOS
// DIRECCIONES.
//
// # Qué se movió y por qué
//
// El 08-09-2026 `docs/` tenía 42 ficheros en el primer nivel y **18 eran
// cuadernos de hallazgos**, 454.200 bytes, el 32,8 % del directorio. Mezclados
// con `guia.md`, `diseno.md` y `decisiones.md`, que son los que se leen para
// trabajar. Quien abre `docs/` no puede distinguir el plan de la bitácora.
//
// **No se resumió, no se fundió y no se borró ni un byte**: consolidar dieciocho
// relatos fechados en un resumen sin autor sería tirar justo lo que hace útil un
// cuaderno de hallazgos. Lo único que cambió es la carpeta, y las 45 referencias
// que apuntaban a ellos se reescribieron en el mismo commit.
//
// # Y por qué esto necesita puerta
//
// Un directorio con dieciocho ficheros y un índice a mano es un índice que se
// queda viejo en el segundo cuaderno. Las dos direcciones:
//
//	1. Todo cuaderno del directorio está en la tabla. Sin esto, un cuaderno
//	   nuevo entra sin que nadie lo vea y el directorio vuelve a ser un montón.
//	2. Toda fila de la tabla tiene su cuaderno. Sin esto, la tabla cita ficheros
//	   que ya no existen, que es un índice mintiendo con cara de orden.
//
// La que se olvida es siempre la segunda, así que va con su mutación.

const rutaDeLosHallazgos = "docs/hallazgos"

var reFilaDeCuaderno = regexp.MustCompile(`(?m)^\| \[([a-z0-9-]+)\]\(([a-z0-9-]+\.md)\) \|`)

func TestElIndiceDeLosCuadernosYElDirectorioSeApuntanEnLasDosDirecciones(t *testing.T) {
	entradas, err := os.ReadDir(rutaDeLosHallazgos)
	if err != nil {
		t.Fatalf("no puedo leer %s: %v", rutaDeLosHallazgos, err)
	}
	enDisco := map[string]bool{}
	for _, e := range entradas {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".md") || n == "LEEME.md" {
			continue
		}
		enDisco[strings.TrimSuffix(n, ".md")] = true
	}

	b, err := os.ReadFile(filepath.Join(rutaDeLosHallazgos, "LEEME.md")) // #nosec G304 -- ruta constante del repositorio
	if err != nil {
		t.Fatalf("no puedo leer el indice de %s: %v", rutaDeLosHallazgos, err)
	}
	enElIndice := map[string]bool{}
	for _, m := range reFilaDeCuaderno.FindAllStringSubmatch(string(b), -1) {
		if m[1]+".md" != m[2] {
			t.Errorf("la fila del cuaderno %q enlaza a %q: el rotulo y el fichero no son el "+
				"mismo, asi que el indice esta diciendo dos cosas distintas", m[1], m[2])
		}
		enElIndice[m[1]] = true
	}
	// El suelo protege del verde por vacio en los dos lados a la vez.
	if len(enDisco) < 10 || len(enElIndice) < 10 {
		t.Fatalf("hay %d cuadernos en disco y %d filas en el indice, y hoy son mas de diez: "+
			"o el directorio ha adelgazado, o uno de los dos patrones ha dejado de casar y "+
			"esto esta midiendo el vacio", len(enDisco), len(enElIndice))
	}

	for n := range enDisco {
		if !enElIndice[n] {
			t.Errorf("%s/%s.md existe y el indice no lo nombra.\n"+
				"  Un cuaderno que entra sin pasar por el indice es como este directorio "+
				"vuelve a ser un monton: dieciocho ficheros y ninguna forma de saber cual "+
				"mirar.", rutaDeLosHallazgos, n)
		}
	}
	for n := range enElIndice {
		if !enDisco[n] {
			t.Errorf("el indice nombra el cuaderno %q y no hay ningun %s/%s.md.\n"+
				"  Es la direccion que se olvida: un indice que cita ficheros que ya no "+
				"existen miente con cara de orden.", n, rutaDeLosHallazgos, n)
		}
	}
	t.Logf("%d cuadernos, todos en el indice y en las dos direcciones", len(enDisco))
}

// NINGUNA REFERENCIA AL SITIO VIEJO SOBREVIVE.
//
// La mudanza reescribio 45 referencias en 32 ficheros. Esta puerta existe para
// la 46: un enlace a `docs/hallazgos-x.md` escrito manana no da error en ningun
// sitio, se queda roto en silencio y quien lo siga se lleva la impresion de que
// el hallazgo esta escrito y no lo encuentra.
//
// Recorre el arbol de git (`git ls-files`) y no el disco, para no tropezar con
// los worktrees de `.claude/`, que llevan copias del arbol de antes de la mudanza.
func TestNingunaReferenciaApuntaAlSitioViejoDeLosHallazgos(t *testing.T) {
	salida, err := gitDice("ls-files")
	if err != nil {
		t.Skipf("SALTADO, no comprobado: git no contesta aqui (%v). NO es verde", err)
	}
	viejo := regexp.MustCompile(`docs/hallazgos-[a-z0-9-]+\.md`)
	mirados, rotos := 0, 0
	for _, ruta := range strings.Split(salida, "\n") {
		ruta = strings.TrimSpace(ruta)
		if ruta == "" || !tieneExtensionDeTexto(ruta) {
			continue
		}
		b, err := os.ReadFile(ruta) // #nosec G304 -- rutas del propio arbol de git
		if err != nil {
			continue
		}
		mirados++
		for _, m := range viejo.FindAllString(string(b), -1) {
			rotos++
			t.Errorf("%s enlaza a %q, que ya no existe: los cuadernos viven en %s/ desde "+
				"el 08-09-2026.\n"+
				"  Arreglo: %s", ruta, m, rutaDeLosHallazgos,
				strings.Replace(m, "docs/hallazgos-", "docs/hallazgos/", 1))
		}
	}
	if mirados < 100 {
		t.Fatalf("he mirado %d ficheros de texto del arbol y hoy son cientos: este "+
			"recorrido esta midiendo el vacio", mirados)
	}
	t.Logf("%d ficheros de texto del arbol recorridos, %d referencias al sitio viejo",
		mirados, rotos)
}

func tieneExtensionDeTexto(ruta string) bool {
	for _, e := range []string{".md", ".go", ".yml", ".yaml", ".sh", ".json", ".txt", ".html"} {
		if strings.HasSuffix(ruta, e) {
			return true
		}
	}
	return false
}
