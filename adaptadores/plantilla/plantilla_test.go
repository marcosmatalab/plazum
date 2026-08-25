package plantilla

import (
	"errors"
	"fmt"
	"html"
	"io"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"dutiq/puertos"
)

// Los identificadores que aparecen aqui son SINTETICOS (urn:demo:...). No
// pueden ser normas reales: el invariante 2 del proyecto prohibe
// identificadores de norma en el codigo, ficheros de test incluidos.

// catalogo es un doble minimo de puertos.Catalogo. Devuelve la clave con el
// idioma delante, que es lo que permite comprobar que se eligio el juego de
// plantillas del idioma correcto.
type catalogo struct {
	idiomas []string
	textos  map[string]string
}

func (c catalogo) Traducir(idioma, clave string, args ...any) string {
	if t, ok := c.textos[clave]; ok {
		if len(args) > 0 && strings.Contains(t, "%") {
			return fmt.Sprintf(t, args...)
		}
		return t
	}
	s := idioma + ":" + clave
	for _, a := range args {
		s += fmt.Sprintf(" %v", a)
	}
	return s
}

func (c catalogo) Idiomas() []string         { return c.idiomas }
func (c catalogo) Faltantes(string) []string { return nil }

var _ puertos.Catalogo = catalogo{}

func ficheros() fstest.MapFS {
	return fstest.MapFS{
		"pagina.html": &fstest.MapFile{Data: []byte(
			`{{define "pagina"}}<p>{{t "rotulo"}}</p><p>{{.Corpus}}</p>` +
				`<p>{{t "conargs" .N}}</p>{{end}}`)},
		"otra.html": &fstest.MapFile{Data: []byte(`{{define "otra"}}otra{{end}}`)},
	}
}

func nuevoDePrueba(t *testing.T, cat puertos.Catalogo) *Motor {
	t.Helper()
	m, err := Nuevo(ficheros(), cat, "*.html")
	if err != nil {
		t.Fatalf("construir el motor: %v", err)
	}
	return m
}

type datos struct {
	Corpus string
	N      int
}

func render(t *testing.T, m *Motor, idioma string, d datos) string {
	t.Helper()
	var b strings.Builder
	if err := m.Render(&b, "pagina", d, idioma); err != nil {
		t.Fatalf("render en %q: %v", idioma, err)
	}
	return b.String()
}

// ---------------------------------------------------------------------------

// El motor se niega a construirse cuando no puede hacer su trabajo, en vez de
// construirse y dar paginas rotas.
func TestElMotorSeNiegaACosasQueDanPaginasRotas(t *testing.T) {
	casos := []struct {
		nombre   string
		sistema  fstest.MapFS
		cat      puertos.Catalogo
		patrones []string
		quiero   error
	}{
		{"sin catalogo", ficheros(), nil, nil, ErrSinCatalogo},
		{"catalogo sin idiomas", ficheros(), catalogo{}, nil, ErrSinCatalogo},
		{"idioma repetido", ficheros(), catalogo{idiomas: []string{"es", "es"}}, nil,
			ErrSinCatalogo},
		{"ningun fichero casa", ficheros(), catalogo{idiomas: []string{"es"}},
			[]string{"*.tmpl"}, ErrSinPlantillas},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			_, err := Nuevo(c.sistema, c.cat, c.patrones...)
			if !errors.Is(err, c.quiero) {
				t.Fatalf("error %v, esperaba uno que fuera %v", err, c.quiero)
			}
		})
	}
	if _, err := Nuevo(nil, catalogo{idiomas: []string{"es"}}); !errors.Is(err, ErrSinPlantillas) {
		t.Fatalf("sin sistema de ficheros: %v", err)
	}
}

// Hay un juego de plantillas por idioma, y el que se usa es el que se pide.
func TestCadaIdiomaUsaSuCatalogo(t *testing.T) {
	m := nuevoDePrueba(t, catalogo{idiomas: []string{"es", "en"}})
	if got := render(t, m, "es", datos{}); !strings.Contains(got, "es:rotulo") {
		t.Errorf("en es salio %q", got)
	}
	if got := render(t, m, "en", datos{}); !strings.Contains(got, "en:rotulo") {
		t.Errorf("en en salio %q", got)
	}
	// Y los argumentos llegan al catalogo, que es lo que hace que un contador
	// se pueda escribir en cada idioma con su gramatica.
	if got := render(t, m, "es", datos{N: 3}); !strings.Contains(got, "es:conargs 3") {
		t.Errorf("los argumentos no llegan al catalogo: %q", got)
	}
}

// Resolver nunca falla: un idioma desconocido cae al de por defecto y una
// etiqueta con region cae a su primaria. Importa porque el resultado acaba en
// <html lang>, y una pagina que declara un idioma que no lleva dentro rompe a
// los lectores de pantalla.
func TestResolverNuncaFallaYCaeDondeDebe(t *testing.T) {
	m := nuevoDePrueba(t, catalogo{idiomas: []string{"es", "en"}})
	casos := map[string]string{
		"es": "es", "en": "en", "es-ES": "es", "en_GB": "en",
		"": "es", "xx": "es", "xx-YY": "es", "-": "es", "e": "es",
	}
	for pedido, quiero := range casos {
		if got := m.Resolver(pedido); got != quiero {
			t.Errorf("Resolver(%q) = %q y esperaba %q", pedido, got, quiero)
		}
	}
	// Y lo resuelto es siempre uno de los cargados.
	cargados := map[string]bool{}
	for _, i := range m.Idiomas() {
		cargados[i] = true
	}
	for pedido := range casos {
		if !cargados[m.Resolver(pedido)] {
			t.Errorf("Resolver(%q) devuelve un idioma que no esta cargado", pedido)
		}
	}
}

// Una plantilla que no existe da un error que dice cuales hay. Un "no
// encontrada" a secas obliga a leer codigo fuente para saber que se pide.
func TestUnaPlantillaQueNoExisteLoDiceYEnumeraLasQueHay(t *testing.T) {
	m := nuevoDePrueba(t, catalogo{idiomas: []string{"es"}})
	err := m.Render(io.Discard, "no-existe", nil, "es")
	if !errors.Is(err, ErrNoExiste) {
		t.Fatalf("error %v, esperaba ErrNoExiste", err)
	}
	for _, n := range m.Nombres() {
		if !strings.Contains(err.Error(), n) {
			t.Errorf("el error no menciona la plantilla %q que si existe: %v", n, err)
		}
	}
	if len(m.Nombres()) < 2 {
		t.Fatalf("solo hay %d plantillas cargadas: el test no probaria nada", len(m.Nombres()))
	}
	// Y escribir en nada tampoco revienta.
	if err := m.Render(nil, "pagina", nil, "es"); !errors.Is(err, ErrRender) {
		t.Fatalf("Render(nil, ...) dio %v", err)
	}
}

// El texto que pone el catalogo se escapa como cualquier otro dato. Un catalogo
// es un fichero de datos y un fichero de datos no inyecta marcado; si un dia
// `t` devolviera template.HTML, esto se pone rojo.
func TestElTextoDelCatalogoSeEscapa(t *testing.T) {
	const guion = `<script>alert(1)</script>`
	m := nuevoDePrueba(t, catalogo{idiomas: []string{"es"},
		textos: map[string]string{"rotulo": guion}})
	got := render(t, m, "es", datos{})
	if strings.Contains(got, guion) {
		t.Fatalf("el catalogo ha inyectado marcado: %q", got)
	}
	if !strings.Contains(got, html.EscapeString(guion)) {
		t.Errorf("el texto del catalogo no aparece escapado: %q", got)
	}
}

// Y lo que viene del corpus, igual. El corpus lo escribe un tercero.
func TestElContenidoDelCorpusSeEscapa(t *testing.T) {
	m := nuevoDePrueba(t, catalogo{idiomas: []string{"es"}})
	got := render(t, m, "es", datos{Corpus: `urn:demo:x "><img src=x onerror=alert(1)>`})
	if strings.Contains(got, "onerror=alert(1)") && !strings.Contains(got, "&lt;img") {
		t.Fatalf("el contenido del corpus ha entrado sin escapar: %q", got)
	}
}

// El motor se usa desde varias goroutines a la vez, porque un servidor HTTP lo
// hace. Con -race, esto es lo que separa "no lo he visto fallar" de "no puede".
func TestRenderConcurrenteEsSeguro(t *testing.T) {
	m := nuevoDePrueba(t, catalogo{idiomas: []string{"es", "en"}})
	var g sync.WaitGroup
	for i := 0; i < 16; i++ {
		g.Add(1)
		go func(i int) {
			defer g.Done()
			idioma := []string{"es", "en", "es-ES", "xx"}[i%4]
			for j := 0; j < 50; j++ {
				var b strings.Builder
				if err := m.Render(&b, "pagina", datos{Corpus: "x", N: j}, idioma); err != nil {
					t.Errorf("render concurrente: %v", err)
					return
				}
			}
		}(i)
	}
	g.Wait()
}

// El mismo dato y el mismo idioma dan exactamente la misma salida. Sin esto, la
// superficie no puede prometer que la misma direccion da la misma pagina.
func TestElRenderEsDeterminista(t *testing.T) {
	m := nuevoDePrueba(t, catalogo{idiomas: []string{"es"}})
	uno := render(t, m, "es", datos{Corpus: "a", N: 1})
	for i := 0; i < 20; i++ {
		if otro := render(t, m, "es", datos{Corpus: "a", N: 1}); otro != uno {
			t.Fatalf("la salida cambia entre ejecuciones:\n%q\n%q", uno, otro)
		}
	}
}
