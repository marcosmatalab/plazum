package plazum

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const (
	// MinimoDeEnlacesRelativos es el suelo que impide el verde por vacio.
	//
	// Un extractor que deje de casar no falla: devuelve cero enlaces y la puerta
	// pasa. Es el modo de fallo por defecto de cualquier comprobacion que
	// recorra un conjunto, y el unico que no se ve leyendo la salida.
	//
	// Hoy son 93. El suelo va por debajo y no en igualdad exacta a proposito:
	// este numero se mueve con cada enlace que alguien escribe o quita, y una
	// puerta que salta en cada edicion de documentacion ensena a esquivarla. Lo
	// que tiene que cazar es el derrumbe, no el vaiven.
	MinimoDeEnlacesRelativos = 80
	// MinimoDeDocumentosConEnlace es el mismo suelo por el otro eje: noventa
	// enlaces podrian salir todos de un solo documento. Hoy son 10.
	MinimoDeDocumentosConEnlace = 8
)

// reEnlaceMarkdown casa el destino de un enlace o de una imagen: `](destino)`.
var reEnlaceMarkdown = regexp.MustCompile(`\]\(([^)]*)\)`)

// TODO ENLACE RELATIVO DE UN DOCUMENTO RESUELVE, DESDE SU PROPIO DIRECTORIO.
//
// # El fallo que la trae, con su cardinal
//
// `ETAPAS.md` se mudo a `docs/` y sus enlaces al archivo de casillas no se
// reescribieron: seguian diciendo `docs/casillas.md`, que desde `docs/` resuelve
// a `docs/docs/casillas.md`. Eran **54 enlaces rotos** en el documento que
// enlaza el porque de cada casilla del plan, y estuvieron rotos desde la
// mudanza.
//
// **No lo cazo nada, y el motivo es el que importa.** Habia una puerta sobre
// esos enlaces (`TestElArchivoDeCasillasYElPlanSeApuntanEnLasDosDirecciones`) y
// comparaba ANCLAS: el ancla existia, en el fichero correcto, asi que sus dos
// direcciones cuadraban mientras el enlace no llevaba a ningun sitio. Peor: su
// expresion estaba fijada a `docs/casillas.md#`, o sea a la forma ROTA, asi que
// la unica puerta que los miraba estaba de su lado.
//
// La leccion es de clase, no de caso: **una puerta sobre el ANCLA no dice nada
// del FICHERO**, y un enlace tiene las dos mitades.
//
// # Por que hace falta saltarse los code spans, y no es un detalle
//
// `docs/puertas-nacidas-verdes.md` explica una trampa de otro detector
// escribiendo `[texto](ruta)` entre comillas invertidas, como ejemplo en prosa.
// Un extractor ingenuo lo lee como un enlace a `ruta`, no lo encuentra, y la
// puerta nace roja **por un falso positivo**. Y entonces pasa lo de siempre: se
// ablanda, se le pone una excepcion por fichero, y la excepcion es donde se
// esconde el enlace roto numero 55.
//
// Asi que se enmascaran los bloques cercados y los code spans ANTES de buscar, y
// el enmascarado tiene su propio control negativo, porque un enmascarador que se
// pase deja de ver enlaces de verdad y eso tambien da verde.
//
// # Su limite, dicho
//
// Comprueba que el FICHERO existe. No comprueba que el ancla exista dentro de
// el: eso lo hace la puerta de las casillas para los suyos, y para el resto no
// lo hace nadie. Tampoco mira enlaces de referencia (`[texto][ref]`), que hoy no
// hay ninguno en el arbol (medido: 0 lineas con `^[ref]: `).
func TestTodoEnlaceRelativoDeUnDocumentoResuelve(t *testing.T) {
	enlaces, documentos, rotos, ficheros := 0, 0, 0, 0
	for _, f := range ficherosVersionados(t, "*.md") {
		ficheros++
		b, err := os.ReadFile(f) // #nosec G304 -- rutas del propio arbol de git
		if err != nil {
			continue
		}
		dir := filepath.Dir(f)
		delDocumento := 0
		for _, e := range enlacesDe(string(b)) {
			destino, ok := rutaRelativa(e.destino)
			if !ok {
				continue
			}
			enlaces++
			delDocumento++
			if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(destino))); err != nil {
				rotos++
				t.Errorf(`%s:%d enlaza a %q y no existe.

  Resuelto desde el directorio del propio fichero, %s, sale %s.

  La forma en que esto pasa no es teclear mal: es MUDAR el fichero y no
  reescribir sus enlaces. Un enlace escrito desde la raiz sigue pareciendo
  correcto dentro de un subdirectorio, y en GitHub resuelve a una ruta con el
  directorio repetido.`,
					f, e.linea, e.destino, dir, filepath.ToSlash(filepath.Join(dir, destino)))
			}
		}
		if delDocumento > 0 {
			documentos++
		}
	}

	if enlaces < MinimoDeEnlacesRelativos || documentos < MinimoDeDocumentosConEnlace {
		t.Fatalf(`he comprobado %d enlaces relativos en %d documentos, y los suelos son %d y %d.

  Un extractor que deja de casar no falla: devuelve cero y la puerta pasa. Es el
  modo de fallo por defecto de cualquier comprobacion que recorra un conjunto, y
  el unico que no se ve leyendo la salida.`,
			enlaces, documentos, MinimoDeEnlacesRelativos, MinimoDeDocumentosConEnlace)
	}
	t.Logf("%d ficheros .md recorridos; %d enlaces relativos en %d de ellos, %d rotos",
		ficheros, enlaces, documentos, rotos)
}

// EL CONTROL NEGATIVO DEL ENMASCARADO, que es donde esta puerta se rompe.
//
// Tiene dos fallos probables y son opuestos:
//
//   - **Se queda corto**: ve el `[texto](ruta)` que `docs/puertas-nacidas-verdes.md`
//     escribe entre comillas invertidas como ejemplo, nace roja por un falso
//     positivo, y alguien la ablanda con una excepcion por fichero. Esa excepcion
//     es donde se esconde el enlace roto numero 55.
//   - **Se pasa**: enmascara de mas y deja de ver enlaces de verdad. Entonces da
//     verde sin mirar, que es peor porque no se nota.
//
// Los dos se recorren aqui, con fuentes sinteticas.
func TestElExtractorDeEnlacesSaltaElCodigoYVeElRestoo(t *testing.T) {
	destinos := func(s string) []string {
		var out []string
		for _, e := range enlacesDe(s) {
			out = append(out, e.destino)
		}
		return out
	}
	iguales := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	for _, c := range []struct {
		nombre string
		fuente string
		quiero []string
	}{
		{"un enlace normal", "ver [el plan](ETAPAS.md) hoy", []string{"ETAPAS.md"}},
		{"una imagen tambien", "![una captura](fotos/a.png)", []string{"fotos/a.png"}},
		{"dentro de comillas invertidas NO", "la trampa es `[texto](ruta)` y nada mas", nil},
		{"con comillas dobles tampoco", "asi: ``[texto](ruta)`` y ya", nil},
		{"dentro de un bloque cercado NO",
			"antes\n```\n[texto](ruta)\n```\ndespues", nil},
		{"un bloque con lenguaje tampoco",
			"antes\n```bash\nver [x](y.md)\n```\ndespues", nil},
		{"un bloque con tildes tampoco",
			"antes\n~~~\n[texto](ruta)\n~~~\ndespues", nil},
		// LA DIRECCION QUE SE OLVIDA: que el enmascarado no se coma lo de al lado.
		{"un enlace DESPUES de un code span cerrado",
			"pon `algo` y luego [el plan](ETAPAS.md)", []string{"ETAPAS.md"}},
		{"un enlace ANTES de un code span",
			"[el plan](ETAPAS.md) y luego `algo`", []string{"ETAPAS.md"}},
		{"un enlace DESPUES de un bloque cerrado",
			"```\nx\n```\n\nver [el plan](ETAPAS.md)", []string{"ETAPAS.md"}},
		{"dos enlaces con un code span en medio",
			"[a](a.md) `x` [b](b.md)", []string{"a.md", "b.md"}},
	} {
		if got := destinos(c.fuente); !iguales(got, c.quiero) {
			t.Errorf("%s: saca %v y esperaba %v", c.nombre, got, c.quiero)
		}
	}

	// Y EL CASO REAL, no uno sintetico: la linea del repositorio que habria
	// hecho nacer roja a esta puerta por un falso positivo.
	b, err := os.ReadFile("docs/puertas-nacidas-verdes.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range enlacesDe(string(b)) {
		if e.destino == "ruta" {
			t.Errorf("el extractor ve el `[texto](ruta)` que %s escribe como ejemplo en "+
				"prosa. Esta puerta naceria roja por un falso positivo, y la salida barata "+
				"seria exceptuar ese fichero", "docs/puertas-nacidas-verdes.md")
		}
	}
}

// EL CONTROL NEGATIVO DEL CLASIFICADOR, que decide que enlace se comprueba.
//
// Su fallo probable es descartar de mas: una URL absoluta y un ancla suelto NO
// son rutas del arbol, pero si el clasificador descartara tambien las rutas con
// `#` o con mayusculas, la puerta dejaria de mirar justo los enlaces que mas se
// mudan, y el suelo no lo notaria porque seguiria habiendo cientos.
func TestElClasificadorDeEnlacesDescartaLoJustoo(t *testing.T) {
	for _, c := range []struct {
		nombre  string
		entrada string
		quiero  string
		mira    bool
	}{
		{"una ruta relativa", "casillas.md", "casillas.md", true},
		{"con ancla", "casillas.md#el-ancla", "casillas.md", true},
		{"un subdirectorio", "lanzamiento/gif.sh", "lanzamiento/gif.sh", true},
		{"subir un nivel", "../ETAPAS.md", "../ETAPAS.md", true},
		{"con titulo detras", `a.md "el titulo"`, "a.md", true},
		{"entre angulos", "<a b.md>", "a b.md", true},
		{"con un espacio escapado", "docs/a%20b.md", "docs/a b.md", true},
		{"http no", "https://ejemplo/x.md", "", false},
		{"http sin s tampoco", "http://ejemplo/x.md", "", false},
		{"correo no", "mailto:a@b", "", false},
		{"un ancla sola no", "#la-seccion", "", false},
		{"vacio no", "", "", false},
	} {
		got, mira := rutaRelativa(c.entrada)
		if mira != c.mira {
			t.Errorf("%s: mira=%v y esperaba %v (entrada %q)", c.nombre, mira, c.mira, c.entrada)
			continue
		}
		if mira && got != c.quiero {
			t.Errorf("%s: saca %q y esperaba %q", c.nombre, got, c.quiero)
		}
	}
}

// enlaceEnDocumento es un destino con la linea en que aparece.
type enlaceEnDocumento struct {
	destino string
	linea   int
}

// enlacesDe saca los destinos de los enlaces de un documento, saltandose el
// codigo.
//
// El enmascarado sustituye por espacios en vez de borrar, a proposito: asi los
// desplazamientos no se mueven y la linea que se reporta es la de verdad. Un
// error que nombra la linea equivocada manda a mirar otro sitio, que cuesta mas
// que no nombrarla.
func enlacesDe(doc string) []enlaceEnDocumento {
	m := []byte(doc)
	enmascararBloques(m)
	enmascararCodeSpans(m)

	var out []enlaceEnDocumento
	for _, idx := range reEnlaceMarkdown.FindAllSubmatchIndex(m, -1) {
		out = append(out, enlaceEnDocumento{
			destino: strings.TrimSpace(doc[idx[2]:idx[3]]),
			linea:   1 + strings.Count(doc[:idx[0]], "\n"),
		})
	}
	return out
}

// enmascararBloques tapa los bloques cercados (``` y ~~~), incluidas sus vallas.
func enmascararBloques(m []byte) {
	dentro, valla := false, ""
	i := 0
	for i < len(m) {
		fin := i
		for fin < len(m) && m[fin] != '\n' {
			fin++
		}
		linea := string(m[i:fin])
		recortada := strings.TrimLeft(linea, " \t")
		abre := ""
		if strings.HasPrefix(recortada, "```") {
			abre = "```"
		} else if strings.HasPrefix(recortada, "~~~") {
			abre = "~~~"
		}
		switch {
		case !dentro && abre != "":
			dentro, valla = true, abre
			tapar(m, i, fin)
		case dentro && abre == valla:
			dentro = false
			tapar(m, i, fin)
		case dentro:
			tapar(m, i, fin)
		}
		i = fin + 1
	}
}

// enmascararCodeSpans tapa los tramos entre runs de comillas invertidas del
// mismo tamano, que es la regla de CommonMark.
func enmascararCodeSpans(m []byte) {
	i := 0
	for i < len(m) {
		if m[i] != '`' {
			i++
			continue
		}
		ini := i
		for i < len(m) && m[i] == '`' {
			i++
		}
		n := i - ini
		// Se busca el cierre: un run de EXACTAMENTE n comillas invertidas.
		j := i
		for j < len(m) {
			if m[j] != '`' {
				j++
				continue
			}
			k := j
			for k < len(m) && m[k] == '`' {
				k++
			}
			if k-j == n {
				tapar(m, ini, k)
				i = k
				break
			}
			j = k
		}
		if j >= len(m) {
			// Sin cierre no hay code span: la apertura era texto normal.
			return
		}
	}
}

// tapar sustituye por espacios conservando los saltos de linea.
func tapar(m []byte, desde, hasta int) {
	for k := desde; k < hasta && k < len(m); k++ {
		if m[k] != '\n' {
			m[k] = ' '
		}
	}
}

// rutaRelativa dice si un destino es una ruta del arbol y la normaliza.
func rutaRelativa(destino string) (string, bool) {
	d := strings.TrimSpace(destino)
	d = strings.TrimSuffix(strings.TrimPrefix(d, "<"), ">")
	// Un titulo detras del destino: [t](a.md "el titulo").
	if i := strings.IndexAny(d, " \t"); i >= 0 && !strings.HasPrefix(destino, "<") {
		d = d[:i]
	}
	if d == "" || strings.HasPrefix(d, "#") {
		return "", false
	}
	for _, esquema := range []string{"http://", "https://", "mailto:", "ftp://", "//"} {
		if strings.HasPrefix(strings.ToLower(d), esquema) {
			return "", false
		}
	}
	if i := strings.IndexAny(d, "#?"); i >= 0 {
		d = d[:i]
	}
	if d == "" {
		return "", false
	}
	d = strings.ReplaceAll(d, "%20", " ")
	return d, true
}
