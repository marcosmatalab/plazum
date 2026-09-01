package acta

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/adaptadores/catalogo"
	"github.com/marcosmatalab/plazum/nucleo/accesos"
	nucleo "github.com/marcosmatalab/plazum/nucleo/acta"
	"github.com/marcosmatalab/plazum/nucleo/auditoria"
	"github.com/marcosmatalab/plazum/nucleo/censo"
	"github.com/marcosmatalab/plazum/nucleo/incidente"
)

func dia(a, m, d int) time.Time { return time.Date(a, time.Month(m), d, 12, 0, 0, 0, time.UTC) }

var elPeriodo = nucleo.Periodo{Desde: dia(2026, 1, 1), Hasta: dia(2026, 12, 31)}

// fuente es un doble de Actas.
type fuente struct {
	a   nucleo.Acta
	hay bool
	err error
}

func (f *fuente) Ultima() (nucleo.Acta, bool, error) { return f.a, f.hay, f.err }

func cat(t *testing.T) *catalogo.Catalogo {
	t.Helper()
	c, err := catalogo.Nuevo()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// EL CASO COMPLETO, en pequeno pero con las cuatro secciones vivas: la pantalla
// tiene que poder pintar un acta con datos en las tres fuentes y con su seccion
// de la direccion, o los tests no recorren nada.
func actaDePrueba(t *testing.T) nucleo.Acta {
	t.Helper()
	p, err := auditoria.Abrir("prog-1", auditoria.Ciclo{
		Nombre: "2026-2028", Desde: dia(2026, 1, 1), Hasta: dia(2028, 12, 31),
	}, []auditoria.Unidad{
		{Paquete: "p1", Version: "0.1", Obligacion: "o1", Titulo: "Copias"},
		{Paquete: "p1", Version: "0.1", Obligacion: "o2", Titulo: "Accesos"},
	}, auditoria.Arrastre{DeCiclo: "2023-2025", SinAuditar: map[string]int{"p1|o2": 2}})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Auditar(auditoria.Sesion{ID: "S1", Auditor: "aud-1", Cuando: dia(2026, 3, 1),
		Unidades: []string{"p1|o1"}}); err != nil {
		t.Fatal(err)
	}

	ins, err := censo.Tomar([]byte("usuario;nombre;permiso\nu1;Bea Nunez;admin\nu2;Carlos Ortiz;lector\n"),
		censo.Opciones{Sistema: "erp", Fuente: "export", Quien: "u-042", Tomada: dia(2026, 6, 1),
			Retencion: "12 meses", Columnas: censo.ColumnasHabituales()})
	if err != nil {
		t.Fatal(err)
	}
	c, err := accesos.Abrir("uar-2026", ins, dia(2026, 6, 2), map[string]string{"erp|u1|admin": "jefa"})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Registrar(accesos.Decision{Fila: "erp|u1|admin", Veredicto: accesos.Aprobar,
		Quien: "jefa", Cuando: dia(2026, 6, 10)}); err != nil {
		t.Fatal(err)
	}

	i1, err := incidente.Abrir("INC-1", dia(2026, 2, 1), dia(2026, 2, 2), "guardia")
	if err != nil {
		t.Fatal(err)
	}

	a, err := nucleo.Componer(nucleo.Entradas{
		ID: "acta-2026", Organizacion: "Molduras del Norte SL", Periodo: elPeriodo,
		Programa:                p,
		Responsables:            map[string]string{"p1|o1": "aud-1"},
		Campana:                 c,
		HayRegistroDeIncidentes: true,
		Incidentes:              []*incidente.Incidente{i1},
		Esperadas: []nucleo.Notificacion{
			{Incidente: "INC-1", Hito: "n-inicial", Que: "notificacion inicial"},
		},
		QuienAsistio: []string{"Ana Perez"},
		Decisiones: []nucleo.Parrafo{{
			Frase: nucleo.Frase{Texto: "El consejo aprueba dos horas semanales para cerrar H1."},
			De:    nucleo.DeUnaPersona, Quien: "Ana Perez",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func superficie(t *testing.T, f Actas, conSesion bool) *Superficie {
	t.Helper()
	o := Opciones{Fuente: f, Catalogo: cat(t), Base: "/acta", Estatico: "/estatico"}
	if conSesion {
		o.Quien = func(*http.Request) string { return "u-042" }
	}
	s, err := Nuevo(o)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func pedir(t *testing.T, s *Superficie, metodo, ruta string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(metodo, ruta, nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}

// ---------------------------------------------------------------------------
// La superficie no muta
// ---------------------------------------------------------------------------

// NINGUNA RUTA DE ESTA SUPERFICIE MUTA NADA, y se comprueba preguntandole al
// ENRUTADOR, no a una lista escrita al lado del test.
//
// Un acta no se edita: se compone de lo que ya esta registrado, y cada fuente se
// toca en su propia pantalla. Un formulario aqui seria una segunda via para
// cambiar lo que el acta dice, que es lo que un documento probatorio no puede
// tener.
func TestNingunaRutaDeEstaSuperficieMuta(t *testing.T) {
	s := superficie(t, &fuente{a: actaDePrueba(t), hay: true}, true)
	ps := s.Patrones()
	sort.Strings(ps)
	for _, p := range ps {
		if !strings.HasPrefix(p, "GET ") {
			t.Errorf("la ruta %q no es GET: esta superficie no muta nada y esa promesa se rompe "+
				"con una sola ruta", p)
		}
	}
	for _, ruta := range []string{"/acta/", "/acta/derivacion/1.1.1"} {
		for _, m := range []string{"POST", "PUT", "PATCH", "DELETE"} {
			if w := pedir(t, s, m, ruta); w.Code == http.StatusOK {
				t.Errorf("%s %s responde 200: la superficie atiende un metodo mutante", m, ruta)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Sin sesion no se ensena el acta
// ---------------------------------------------------------------------------

// EL ACTA LLEVA UNA LISTA DE QUIEN HIZO QUE dentro de la organizacion, y mas que
// la pantalla de la UAR: quien audito, quien difirio y con que motivo, quien
// decidio cada acceso, quien excuso una linea y quien asistio.
//
// Se comprueba por donde duele: que NINGUNO de esos nombres salga en el cuerpo.
// Un test que solo mirara el codigo 401 daria verde con la pagina entera dentro.
func TestSinSesionNoSaleNiUnNombre(t *testing.T) {
	s := superficie(t, &fuente{a: actaDePrueba(t), hay: true}, false)
	for _, ruta := range []string{"/acta/", "/acta/derivacion/1.1.1"} {
		w := pedir(t, s, "GET", ruta)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s responde %d y tenia que ser 401", ruta, w.Code)
		}
		cuerpo := w.Body.String()
		for _, nombre := range []string{"aud-1", "jefa", "Ana Perez", "u-042", "Molduras"} {
			if strings.Contains(cuerpo, nombre) {
				t.Errorf("%s: sin sesion sale %q, que es un nombre de dentro de la organizacion",
					ruta, nombre)
			}
		}
		porQue := cat(t).Traducir("es", "acta.pantalla.sin_sesion.por_que")
		if !strings.Contains(cuerpo, escapado(porQue)) {
			t.Errorf("%s: la pantalla no dice por que no se sirve", ruta)
		}
	}
	// Control positivo: con sesion, los nombres SI salen. Sin esto, el test de
	// arriba estaria contento con una pantalla que no pinta nada nunca.
	con := superficie(t, &fuente{a: actaDePrueba(t), hay: true}, true)
	if !strings.Contains(pedir(t, con, "GET", "/acta/").Body.String(), "Molduras") {
		t.Error("con sesion, el acta tenia que pintarse")
	}
}

// EL ROTULO DEL CENSO NO LLEGA A LA PANTALLA, porque el compositor lo apaga por
// defecto. Es la minimizacion comprobada en la pieza que viaja, mirada desde el
// otro extremo del camino.
func TestElNombreDeUnaCuentaRevisadaNoLlegaALaPantalla(t *testing.T) {
	a := actaDePrueba(t)
	s := superficie(t, &fuente{a: a, hay: true}, true)

	// SE MIRAN LA PORTADA Y TODAS LAS DERIVACIONES, y esto lo escribio una
	// mutacion que sobrevivio: la primera version solo miraba la portada, y la
	// portada no lleva elementos, solo cubos y recuentos. Con el interruptor
	// forzado a encendido en el nucleo, el rotulo del censo salia en cada
	// derivacion abierta y el test seguia verde. Una pagina que no ensena el dato
	// no demuestra que el dato no salga: demuestra que ahi no se ensena nada.
	paginas := []string{pedir(t, s, "GET", "/acta/").Body.String()}
	for _, cs := range a.Cifras() {
		paginas = append(paginas, pedir(t, s, "GET", "/acta/derivacion/"+cs.Ref).Body.String())
	}
	if len(paginas) < 5 {
		t.Fatalf("solo se miran %d paginas: el acta de prueba tiene menos cifras de las que "+
			"hace falta para que esto pruebe algo", len(paginas))
	}
	for _, cuerpo := range paginas {
		for _, rotulo := range []string{"Bea Nunez", "Carlos Ortiz"} {
			if strings.Contains(cuerpo, rotulo) {
				t.Errorf("la pantalla saca %q, que es el nombre con el que el IdP llama a una "+
					"cuenta revisada. Para ver a las personas del censo esta la pantalla de la "+
					"UAR", rotulo)
			}
		}
	}
	// Y la identidad SI: sin ella el numero no se puede abrir, que es lo que
	// separa minimizar de esconder.
	abierta := pedir(t, s, "GET", "/acta/derivacion/2.1.1").Body.String()
	if !strings.Contains(abierta, "erp|u1|admin") {
		t.Error("sin la identidad de la fila, la derivacion no ensena de que sale el numero")
	}
}

// ---------------------------------------------------------------------------
// D11-b: el estado vacio trae su siguiente paso
// ---------------------------------------------------------------------------

func TestSinActaLaPantallaDiceQueHaceFalta(t *testing.T) {
	for _, f := range []Actas{nil, &fuente{}} {
		s := superficie(t, f, true)
		w := pedir(t, s, "GET", "/acta/")
		if w.Code != http.StatusOK {
			t.Errorf("el estado vacio responde %d y no es un error: es un estado", w.Code)
		}
		cuerpo := w.Body.String()
		if !strings.Contains(cuerpo, "Todav") {
			t.Error("la pantalla vacia no dice que no hay acta")
		}
		if !strings.Contains(cuerpo, "programa de auditor") {
			t.Error("la pantalla vacia no dice de que se compone un acta, asi que es un callejon")
		}
	}
}

// ---------------------------------------------------------------------------
// D11-c: cada numero es clicable hasta su derivacion
// ---------------------------------------------------------------------------

// TODA CIFRA DEL ACTA TIENE SU ENLACE EN LA PAGINA, Y EL ENLACE ABRE ESA CIFRA.
//
// Se recorren TODAS, no una muestra: una cifra sin enlace obliga a fiarse, y la
// que se queda sin el es siempre la que nadie miro.
func TestCadaNumeroDelActaSeAbreDesdeLaPantalla(t *testing.T) {
	a := actaDePrueba(t)
	s := superficie(t, &fuente{a: a, hay: true}, true)
	pagina := pedir(t, s, "GET", "/acta/").Body.String()
	if len(a.Cifras()) == 0 {
		t.Fatal("el acta de prueba no tiene cifras")
	}
	c := cat(t)
	for _, cs := range a.Cifras() {
		enlace := "/acta/derivacion/" + cs.Ref
		if !strings.Contains(pagina, `href="`+enlace+`"`) {
			t.Errorf("la cifra %s no tiene enlace en la pagina", cs.Ref)
			continue
		}
		w := pedir(t, s, "GET", enlace)
		if w.Code != http.StatusOK {
			t.Errorf("%s responde %d", enlace, w.Code)
			continue
		}
		abierta := w.Body.String()
		cubo := c.Traducir("es", cs.Cifra.Cubo.Clave)
		if !strings.Contains(abierta, cubo) {
			t.Errorf("%s abre una pagina que no nombra su cubo %q", enlace, cubo)
		}
		// Y ENSENA LA LISTA ENTERA, sin recortar.
		for _, el := range cs.Cifra.Elementos {
			if !strings.Contains(abierta, el.Clave) {
				t.Errorf("%s no ensena %q, asi que el numero sigue habiendo que creerselo",
					enlace, el.Clave)
			}
		}
	}
}

// LA PANTALLA Y EL PAPEL NUMERAN IGUAL.
//
// La referencia la calcula el NUCLEO y la pantalla la indexa por identidad, no
// la recalcula. Si divergieran, un consejero que leyera "[1.2.3]" en el board
// pack impreso abriria otra cosa en el navegador, que es la peor forma de
// romper la promesa de este documento: la que no se nota.
func TestLaPantallaNumeraComoElPapel(t *testing.T) {
	a := actaDePrueba(t)
	s := superficie(t, &fuente{a: a, hay: true}, true)
	pagina := pedir(t, s, "GET", "/acta/").Body.String()
	enPantalla := map[string]bool{}
	for _, m := range regexp.MustCompile(`\[(\d+\.\d+\.\d+)\]`).FindAllStringSubmatch(pagina, -1) {
		enPantalla[m[1]] = true
	}
	enPapel := map[string]bool{}
	for _, m := range regexp.MustCompile(`\[(\d+\.\d+\.\d+)\]`).FindAllStringSubmatch(a.Texto(), -1) {
		enPapel[m[1]] = true
	}
	if len(enPapel) == 0 {
		t.Fatal("el board pack impreso no lleva referencias")
	}
	for r := range enPapel {
		if !enPantalla[r] {
			t.Errorf("la referencia %s sale en el papel y no en la pantalla", r)
		}
	}
	for r := range enPantalla {
		if !enPapel[r] {
			t.Errorf("la referencia %s sale en la pantalla y no en el papel", r)
		}
	}
}

// UNA REFERENCIA QUE NO EXISTE DICE QUE NO, y no devuelve una pagina vacia con
// cara de dato: eso se leeria como "este numero no tiene nada detras", que es la
// afirmacion contraria a la que sostiene el documento entero.
func TestUnaReferenciaInventadaNoDevuelveUnaPaginaVacia(t *testing.T) {
	s := superficie(t, &fuente{a: actaDePrueba(t), hay: true}, true)
	w := pedir(t, s, "GET", "/acta/derivacion/9.9.9")
	if w.Code != http.StatusNotFound {
		t.Errorf("una referencia inventada responde %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "no existe") {
		t.Errorf("no dice que esa derivacion no existe: %s", w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// El descargo llega a la pantalla como lo escribe el nucleo
// ---------------------------------------------------------------------------

// LA FRASE DE LA PANTALLA ES LA DEL NUCLEO, letra por letra (sin tildes), Y VA
// PEGADA AL NUMERO.
//
// Una frase que vive en dos sitios se corrige en uno. Y una que va en una nota
// al pie deja de leerse, que es como una pantalla acaba acusando en falso sin
// que nadie haya escrito una acusacion.
func TestElDescargoLlegaALaPantallaYVaPegadoAlNumero(t *testing.T) {
	a := actaDePrueba(t)
	s := superficie(t, &fuente{a: a, hay: true}, true)
	pagina := pedir(t, s, "GET", "/acta/").Body.String()

	vistos := 0
	for _, cs := range a.Cifras() {
		if cs.Cifra.Valor() == 0 || cs.Cifra.Descargo.Vacia() {
			continue
		}
		vistos++
		frase := cat(t).Traducir("es", cs.Cifra.Descargo.Clave)
		if sinTildes(frase) != sinTildes(cs.Cifra.Descargo.Texto) {
			t.Errorf("%s: el catalogo y el nucleo dicen cosas distintas", cs.Ref)
		}
		i := strings.Index(pagina, "["+cs.Ref+"]")
		if i < 0 {
			t.Fatalf("la cifra %s no sale en la pagina", cs.Ref)
		}
		j := strings.Index(pagina[i:], escapado(frase))
		if j < 0 || j > 600 {
			t.Errorf("%s: la frase no va pegada al numero (distancia %d)", cs.Ref, j)
		}
	}
	if vistos == 0 {
		t.Fatal("ninguna cifra con datos lleva descargo, asi que este test no recorre nada")
	}
}

// ---------------------------------------------------------------------------
// Contrato del catalogo y de la plantilla
// ---------------------------------------------------------------------------

// LAS CLAVES DECLARADAS SON LAS QUE LA PLANTILLA PIDE, EN LAS DOS DIRECCIONES.
//
// Y hay una regla extra de esta superficie: la plantilla solo nombra con `t`
// claves de acta.pantalla.* y de ui.*. Las palabras del DOCUMENTO (cubos,
// repartos, descargos, titulos) llegan ya traducidas desde Go con la clave que
// declara el nucleo, porque varias llevan huecos que rellena un dato. Si la
// plantilla empezara a nombrarlas, habria dos caminos para la misma cadena.
func TestLasClavesDeclaradasSonLasQuePideLaPlantilla(t *testing.T) {
	b, err := plantillasFS.ReadFile("plantillas/acta.html")
	if err != nil {
		t.Fatal(err)
	}
	pide := map[string]bool{}
	for _, m := range regexp.MustCompile(`\{\{t "([^"]+)"`).FindAllStringSubmatch(string(b), -1) {
		pide[m[1]] = true
	}
	if len(pide) < 10 {
		t.Fatalf("la plantilla pide %d claves y son muchas menos de las que tiene: el detector "+
			"esta mirando otra cosa", len(pide))
	}
	declaradas := map[string]bool{}
	for _, k := range ClavesDeCatalogo() {
		declaradas[k] = true
	}
	for k := range pide {
		if !declaradas[k] {
			t.Errorf("la plantilla pide %q y ClavesDeCatalogo() no la declara", k)
		}
		if !strings.HasPrefix(k, "acta.pantalla.") && !strings.HasPrefix(k, "ui.") {
			t.Errorf("la plantilla nombra %q, que es una clave del DOCUMENTO. Esas llegan ya "+
				"traducidas desde Go: nombrarlas aqui abre un segundo camino para la misma "+
				"cadena", k)
		}
	}
	// LA OTRA DIRECCION, con el matiz que hace falta: una clave declarada puede
	// pedirla la plantilla O el codigo (el mensaje de error del render no pasa por
	// ninguna plantilla, porque se sirve justo cuando la plantilla ha fallado).
	// Lo que no puede es no pedirla nadie.
	enGo := clavesEnElCodigo(t)
	for k := range declaradas {
		if !pide[k] && !enGo[k] {
			t.Errorf("ClavesDeCatalogo() declara %q y no la pide ni la plantilla ni el codigo", k)
		}
	}
	// Y a la inversa sobre el espacio propio: toda clave acta.pantalla.* que el
	// codigo nombre tiene que estar declarada. El prefijo es lo que marca de quien
	// es la clave, asi que es lo que se puede exigir sin arrastrar las del nucleo.
	for k := range enGo {
		if strings.HasPrefix(k, "acta.pantalla.") && !declaradas[k] {
			t.Errorf("el codigo nombra %q y ClavesDeCatalogo() no la declara", k)
		}
	}
}

// clavesEnElCodigo saca los literales de clave de las fuentes de esta
// superficie. Se lee el fichero y no el AST porque lo que se busca es una
// cadena, y una cadena esta igual de presente en un literal que en un comentario.
func clavesEnElCodigo(t *testing.T) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	entradas, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	re := regexp.MustCompile(`"((?:acta|ui)\.[a-z0-9_.]+)"`)
	visto := false
	for _, e := range entradas {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		visto = true
		b, err := os.ReadFile(filepath.Join(".", e.Name())) // #nosec G304 -- ruta del propio paquete, fijada por el test
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range re.FindAllStringSubmatch(string(b), -1) {
			out[m[1]] = true
		}
	}
	if !visto {
		t.Fatal("no se ha leido ningun fichero de esta superficie")
	}
	return out
}

// LA PANTALLA NO LLEVA NADA QUE UNA CSP ESTRICTA BLOQUEE.
func TestLaPantallaNoLlevaNadaQueUnaCSPEstrictaBloquee(t *testing.T) {
	s := superficie(t, &fuente{a: actaDePrueba(t), hay: true}, true)
	for _, ruta := range []string{"/acta/", "/acta/derivacion/1.1.1"} {
		cuerpo := pedir(t, s, "GET", ruta).Body.String()
		for _, malo := range []string{"<script", " style=", " onclick=", " onload=", "javascript:"} {
			if strings.Contains(cuerpo, malo) {
				t.Errorf("%s lleva %q, que una CSP estricta bloquea", ruta, malo)
			}
		}
	}
}

// SIN CATALOGO NO SE CONSTRUYE. El valor cero de "no se traducir" tiene que ser
// no arrancar, no arrancar pintando las claves en la cara de un consejo.
func TestSinCatalogoLaSuperficieNoSeConstruye(t *testing.T) {
	if _, err := Nuevo(Opciones{Base: "/acta"}); err == nil {
		t.Fatal("se ha construido una superficie sin catalogo")
	}
}

func escapado(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&#34;", "'", "&#39;")
	return r.Replace(s)
}

func sinTildes(s string) string {
	r := strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ñ", "n",
		"Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U", "Ñ", "N")
	return r.Replace(s)
}
