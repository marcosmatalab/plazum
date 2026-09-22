package plazum

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

const (
	rutaDelIndiceDePendientes  = "docs/pendientes.md"
	rutaDelArchivoDePendientes = "docs/bitacora/pendientes-historico.md"
)

var (
	reSeccionDelArchivo = regexp.MustCompile(`(?m)^## (.+)$`)
	reEnlaceDelIndice   = regexp.MustCompile(`\(bitacora/pendientes-historico\.md#([^)]+)\)`)
	reItemNumerado      = regexp.MustCompile(`(?m)^(\d+)\.\s`)
)

// EL INDICE DE PENDIENTES Y EL ARCHIVO SE APUNTAN EN LAS DOS DIRECCIONES.
//
// # Que se movio y por que
//
// El 22-09-2026 `docs/pendientes.md` eran 2.856 lineas y 125 cabeceras, con lo
// abierto y lo cerrado mezclados: quien llegaba no podia saber que queda por
// hacer sin leerlo entero, y lo primero que leia era una auditoria de agosto. Un
// registro de pendientes que no se puede barrer en un minuto deja de usarse como
// registro.
//
// Y se contradecia a si mismo, que es lo que de verdad lo trae aqui. Su
// preambulo decia «un P0 no entra aqui» y habia DOS, y decia «cuando algo se
// cierra se borra de aqui» y habia secciones marcadas CERRADO. Un revisor hostil
// encuentra eso en un minuto y a partir de ahi desconfia del resto, que es
// injusto porque el resto aguanta.
//
// No se resumio ni se borro nada: el cuerpo se mudo entero a la bitacora con sus
// anclas, igual que los cuadernos de hallazgos el 08-09-2026. Y el nombre
// `docs/pendientes.md` se queda donde estaba a proposito, porque 37 sitios del
// arbol lo citan y reescribirlos era la forma de que unos cuantos acabaran
// apuntando al fichero equivocado.
//
// # Las dos direcciones, y cual es la que se olvida
//
//  1. Toda seccion del archivo esta en el indice. Sin esto, una familia entra
//     sin que nadie la vea y el archivo vuelve a ser un monton.
//  2. Todo enlace del indice lleva a una seccion que existe. Es la que se
//     olvida: un indice que cita secciones que ya no estan miente con cara de
//     orden.
func TestElIndiceDePendientesYElArchivoSeApuntanEnLasDosDirecciones(t *testing.T) {
	indice := leerFichero(t, rutaDelIndiceDePendientes)
	archivo := leerFichero(t, rutaDelArchivoDePendientes)

	enElArchivo := map[string]string{} // ancla -> titulo
	for _, m := range reSeccionDelArchivo.FindAllStringSubmatch(archivo, -1) {
		enElArchivo[anclaDeGitHub(m[1])] = m[1]
	}
	enElIndice := map[string]bool{}
	for _, m := range reEnlaceDelIndice.FindAllStringSubmatch(indice, -1) {
		enElIndice[m[1]] = true
	}
	// El suelo protege del verde por vacio en los dos lados a la vez.
	if len(enElArchivo) < 10 || len(enElIndice) < 10 {
		t.Fatalf("el archivo trae %d secciones y el indice %d enlaces, y hoy son mas de "+
			"diez en los dos: o uno ha adelgazado, o un patron ha dejado de casar y esto "+
			"esta midiendo el vacio", len(enElArchivo), len(enElIndice))
	}

	for ancla, titulo := range enElArchivo {
		if !enElIndice[ancla] {
			t.Errorf("el archivo trae la seccion %q y el indice no la nombra.\n"+
				"  Una familia que entra sin pasar por el indice es como este fichero "+
				"volvio a ser 2.856 lineas que nadie barre.", titulo)
		}
	}
	for ancla := range enElIndice {
		if _, hay := enElArchivo[ancla]; !hay {
			t.Errorf("el indice enlaza a #%s y el archivo no tiene esa seccion.\n"+
				"  Es la direccion que se olvida: un indice que cita lo que ya no existe "+
				"miente con cara de orden.", ancla)
		}
	}
	t.Logf("%d secciones, todas en el indice y en las dos direcciones", len(enElArchivo))
}

// LOS TRES CARDINALES DEL INDICE SALEN DEL ARCHIVO.
//
// El indice publica «2 P0, 29 P1 y 49 P2» y ademas «N abiertos de M» en dos
// filas. Esas cifras son justo la clase que este repositorio lleva un mes
// persiguiendo: un cardinal escrito a mano se queda viejo en el segundo cierre,
// y ademas su fallo probable FAVORECE, porque lo que pasa solo es que algo se
// cierre y nadie baje el numero.
//
// # Que cuenta como CERRADO, declarado aqui y no adivinado
//
// Un item numerado esta cerrado si su texto empieza por `~~` (tachado) o si trae
// `**CERRADO` o `**Cerrado` en su primer parrafo. Se declara la regla en vez de
// inferirla porque una heuristica que nadie escribe es una heuristica que cambia
// sola, y entonces el numero cambia sin que nadie toque nada.
func TestLosCardinalesDelIndiceDePendientesSalenDelArchivo(t *testing.T) {
	indice := leerFichero(t, rutaDelIndiceDePendientes)
	archivo := leerFichero(t, rutaDelArchivoDePendientes)

	p0 := 0
	for _, m := range reSeccionDelArchivo.FindAllStringSubmatch(archivo, -1) {
		if strings.HasPrefix(m[1], "P0 ") {
			p0++
		}
	}
	p1Abiertos, p1Total := contarItems(t, archivo, "## P1", "## P2")
	p2Abiertos, p2Total := contarItems(t, archivo, "## P2", "## La familia: piezas terminadas sin el cable")

	if p0 == 0 || p1Total == 0 || p2Total == 0 {
		t.Fatalf("he contado %d P0, %d items de P1 y %d de P2: con un cero en cualquiera "+
			"de los tres este recorrido esta midiendo el vacio", p0, p1Total, p2Total)
	}

	for _, c := range []struct {
		que   string
		frase string
	}{
		{"la cuenta de las tres prioridades",
			fmt.Sprintf("**%d P0, %d P1 y %d P2.**", p0, p1Abiertos, p2Abiertos)},
		{"la fila de P1", fmt.Sprintf("**%d abiertos** de %d", p1Abiertos, p1Total)},
		{"la fila de P2", fmt.Sprintf("**%d abiertos** de %d", p2Abiertos, p2Total)},
	} {
		if !strings.Contains(indice, c.frase) {
			t.Errorf("%s no cuadra: del archivo sale %q y el indice no lo publica.\n"+
				"  Un cardinal escrito a mano se queda viejo en el segundo cierre, y su "+
				"fallo probable FAVORECE: lo que pasa solo es que algo se cierre y nadie "+
				"baje el numero, y entonces la lista parece mas larga de lo que es.\n"+
				"  Arreglo: poner esa frase, tal cual, en %s.",
				c.que, c.frase, rutaDelIndiceDePendientes)
		}
	}
	t.Logf("%d P0 abiertos, %d de %d P1 abiertos, %d de %d P2 abiertos",
		p0, p1Abiertos, p1Total, p2Abiertos, p2Total)
}

// EL CONTROL NEGATIVO del clasificador de abierto y cerrado.
//
// Su fallo probable es dar por abierto todo, que es la direccion que hincha la
// lista, o dar por cerrado todo, que es la que la vacia. Las dos dan un numero
// plausible y ninguna se nota leyendo.
func TestElClasificadorDeItemsDePendientesAcusaYSeCalla(t *testing.T) {
	sintetico := "\n## P1\n\n" +
		"1. **Uno abierto.** Queda por hacer.\n" +
		"2. ~~**Dos tachado.**~~ Se cerro.\n" +
		"3. **Tres.** **CERRADO el 01-01-2026** por lo que sea.\n" +
		"4. **Cuatro abierto.** Tambien queda.\n" +
		"\n## P2\n"
	abiertos, total := contarItems(t, sintetico, "## P1", "## P2")
	if total != 4 {
		t.Errorf("cuento %d items y el texto sintetico trae 4", total)
	}
	if abiertos != 2 {
		t.Errorf("cuento %d abiertos y el texto sintetico trae 2: el tachado y el CERRADO "+
			"no lo son", abiertos)
	}

	// Y EL ANCLA: si el slug se computara mal, la puerta de arriba acusaria a
	// las dos direcciones a la vez y el mensaje no diria por que.
	for _, c := range []struct{ titulo, quiero string }{
		{"P1", "p1"},
		{"La familia: guardas que no guardaban", "la-familia-guardas-que-no-guardaban"},
		{"El armazón llega a las cuatro superficies (03-09-2026)",
			"el-armazón-llega-a-las-cuatro-superficies-03-09-2026"},
		{"Lo que queda del `<li>` de /alcance", "lo-que-queda-del-li-de-alcance"},
	} {
		if got := anclaDeGitHub(c.titulo); got != c.quiero {
			t.Errorf("el ancla de %q sale %q y GitHub la escribe %q", c.titulo, got, c.quiero)
		}
	}
}

// contarItems cuenta los items numerados de una seccion y cuantos siguen
// abiertos. La regla de que cuenta como cerrado esta declarada arriba.
func contarItems(t *testing.T, texto, desde, hasta string) (abiertos, total int) {
	t.Helper()
	// EL DELIMITADOR SE BUSCA HACIA DELANTE, y esto no es un detalle: el archivo
	// abre con una seccion titulada «P2: la precision real...», o sea que buscar
	// "## P2" desde el principio lo encuentra ANTES que "## P1" y el tramo sale
	// del reves. Se vio en la primera ejecucion, y lo dijo el suelo de la
	// funcion en vez de devolver cero items en silencio.
	i := strings.Index(texto, "\n"+desde+"\n")
	if i < 0 {
		t.Fatalf("no encuentro la seccion %q: si se renombro, este contador se quedo "+
			"viejo y estaria contando otra cosa", desde)
	}
	j := strings.Index(texto[i+1:], "\n"+hasta)
	if j < 0 {
		t.Fatalf("encuentro %q y no encuentro %q despues: sin cierre, el tramo se comeria "+
			"el resto del fichero y contaria items de otras secciones", desde, hasta)
	}
	tramo := texto[i : i+1+j]
	for _, cuerpo := range reItemNumerado.Split(tramo, -1)[1:] {
		total++
		cabeza := cuerpo
		if len(cabeza) > 400 {
			cabeza = cabeza[:400]
		}
		if strings.HasPrefix(strings.TrimSpace(cabeza), "~~") ||
			strings.Contains(cabeza, "**CERRADO") ||
			strings.Contains(cabeza, "**Cerrado") {
			continue
		}
		abiertos++
	}
	return abiertos, total
}

// anclaDeGitHub reproduce como GitHub convierte una cabecera en ancla: a
// minusculas, fuera todo lo que no sea letra, digito, guion o espacio, y los
// espacios a guiones. Los acentos se conservan.
func anclaDeGitHub(titulo string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.ReplaceAll(titulo, "`", "")) {
		switch {
		case r == ' ':
			b.WriteRune('-')
		case r == '-' || r == '_':
			b.WriteRune(r)
		case esAlfanumericoDeAncla(r):
			b.WriteRune(r)
		}
	}
	return b.String()
}

func esAlfanumericoDeAncla(r rune) bool {
	switch {
	case r >= '0' && r <= '9', r >= 'a' && r <= 'z':
		return true
	}
	// Las vocales acentuadas y la enye del castellano, que GitHub conserva.
	return strings.ContainsRune("áéíóúüñ", r)
}
