package camino_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/superficies/camino"
)

// LA SEGUNDA LISTA DE IDIOMAS, ATADA A LA PRIMERA.
//
// `camino.IdiomasDelArmazon()` existe porque `ClavesDelArmazon()` se llama sin
// contexto y no puede preguntarle a nadie que idiomas hay. Es una segunda lista,
// y en esta casa una segunda lista sin puerta es una lista que se queda vieja.
//
// EL FALLO QUE IMPIDE, dicho entero: el dia que entre el tercer idioma, su
// nombre saldria CRUDO («ui.idioma.fr») en el conmutador de las OCHO pantallas,
// y no se pondria rojo nada — el catalogo tendria la clave, pero nadie la
// pediria, asi que la puerta de cobertura tampoco hablaria.
//
// LAS DOS DIRECCIONES, porque la que falta es siempre la que se usa:
//
//	catalogo -> armazon   un idioma que el producto trae y el conmutador no
//	                      sabe nombrar. Es el caso de arriba.
//	armazon -> catalogo   un idioma que el conmutador ofrece y el producto no
//	                      trae. Seria un enlace a una pagina que cae al idioma
//	                      por defecto sin decir por que.
func TestLosIdiomasDelArmazonSonLosDelCatalogo(t *testing.T) {
	// LOS DEL CATALOGO SE LEEN DEL go:embed, y no del directorio, porque el
	// directorio NO es la verdad: `adaptadores/catalogo` nombra sus ficheros uno
	// a uno a proposito y lo dice con estas palabras, «dejar un de.json en
	// cadenas/ no hace nada». Contar ficheros daria por cargado un idioma que el
	// binario no lleva.
	fuente, err := os.ReadFile(filepath.Join("..", "..", "adaptadores", "catalogo",
		"catalogo.go"))
	if err != nil {
		t.Fatal(err)
	}
	var delCatalogo []string
	for _, linea := range strings.Split(string(fuente), "\n") {
		linea = strings.TrimSpace(linea)
		if !strings.HasPrefix(linea, "//go:embed") {
			continue
		}
		for _, campo := range strings.Fields(strings.TrimPrefix(linea, "//go:embed")) {
			if !strings.HasPrefix(campo, "cadenas/") ||
				!strings.HasSuffix(campo, ".json") {
				continue
			}
			delCatalogo = append(delCatalogo,
				strings.TrimSuffix(strings.TrimPrefix(campo, "cadenas/"), ".json"))
		}
	}
	if len(delCatalogo) < 2 {
		t.Fatalf("el go:embed de adaptadores/catalogo dio %d idioma(s) y hacen falta al "+
			"menos 2 para que este test compare algo. Si la directiva cambio de forma, "+
			"este lector se quedo viejo y hay que arreglarlo, no bajar el minimo",
			len(delCatalogo))
	}

	delArmazon := camino.IdiomasDelArmazon()
	sort.Strings(delCatalogo)
	sort.Strings(delArmazon)

	hay := func(xs []string, x string) bool {
		for _, v := range xs {
			if v == x {
				return true
			}
		}
		return false
	}
	for _, i := range delCatalogo {
		if !hay(delArmazon, i) {
			t.Errorf("el producto trae el idioma %q y camino.IdiomasDelArmazon() no lo "+
				"nombra.\nSu nombre saldria CRUDO en el conmutador de las ocho pantallas. "+
				"Arreglo: anadelo a idiomasDelArmazon y pon su cadena ui.idioma.%s en "+
				"LOS DOS ficheros", i, i)
		}
	}
	for _, i := range delArmazon {
		if !hay(delCatalogo, i) {
			t.Errorf("el conmutador ofrece el idioma %q y el producto no lo trae.\n"+
				"Seria un enlace a una pagina que cae al idioma por defecto sin decir "+
				"por que", i)
		}
	}
	t.Logf("%d idiomas, los mismos en el catalogo y en el armazon: %v",
		len(delArmazon), delArmazon)
}

// EL CONMUTADOR MANDA SOBRE Accept-Language, que es la casilla entera.
//
// LAS DOS DIRECCIONES, y las dos hacen falta: un test que solo comprobara «pido
// ingles y sale ingles» pasaria igual si el producto sirviera SIEMPRE en ingles.
// La que descarta esa explicacion es la contraria.
//
// Y LAS TRES FORMAS DE LA NADA (invariante 8), cada una con su caso.
func TestElIdiomaElegidoManda(t *testing.T) {
	m := motorDeMentira{defecto: "es", cargados: []string{"es", "en"}}

	for _, c := range []struct {
		nombre    string
		cabecera  string
		consulta  string
		cookie    string
		quiero    string
		persiste  bool
		porQueImp string
	}{
		{
			nombre:   "el parametro gana a la cabecera contraria",
			cabecera: "es-ES,es;q=0.9", consulta: "?idioma=en",
			quiero: "en", persiste: true,
			porQueImp: "es la casilla: un ingles con el navegador en castellano",
		},
		{
			nombre:   "y en la direccion contraria, que es la que descarta el sesgo",
			cabecera: "en-GB,en;q=0.9", consulta: "?idioma=es",
			quiero: "es", persiste: true,
			porQueImp: "si esto fallara, el producto serviria siempre en ingles",
		},
		{
			nombre:   "la cookie sobrevive a la cabecera contraria",
			cabecera: "es-ES,es;q=0.9", cookie: "en",
			quiero: "en", persiste: false,
			porQueImp: "sin esto la eleccion duraria una pagina",
		},
		{
			nombre:   "el parametro gana a la cookie",
			cabecera: "es", cookie: "en", consulta: "?idioma=es",
			quiero: "es", persiste: true,
			porQueImp: "cambiar de idioma tiene que funcionar estando ya cambiado",
		},
		{
			nombre:   "sin nada, manda Accept-Language",
			cabecera: "en-US,en;q=0.9",
			quiero:   "en", persiste: false,
			porQueImp: "el conmutador no puede haberse cargado la negociacion",
		},
		// LAS TRES FORMAS DE LA NADA.
		{
			nombre:   "parametro AUSENTE cae en la cabecera",
			cabecera: "en", quiero: "en", persiste: false,
			porQueImp: "la nada de verdad",
		},
		{
			nombre:   "parametro PRESENTE Y VACIO no es una eleccion",
			cabecera: "en", consulta: "?idioma=",
			quiero: "en", persiste: false,
			porQueImp: "?idioma= es una preferencia que falta, no el defecto",
		},
		{
			nombre:   "parametro PRESENTE Y RARO no se acepta NI SE PERSISTE",
			cabecera: "en", consulta: "?idioma=zz",
			quiero: "en", persiste: false,
			porQueImp: "persistir lo no interpretable seria inventarse un valor, " +
				"y ademas para siempre",
		},
		{
			nombre:   "cookie PRESENTE Y RARA cae en la cabecera",
			cabecera: "en", cookie: "zz",
			quiero: "en", persiste: false,
			porQueImp: "una cookie que no se entiende no es una eleccion",
		},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/alcance"+c.consulta, nil)
			if c.cabecera != "" {
				r.Header.Set("Accept-Language", c.cabecera)
			}
			if c.cookie != "" {
				r.AddCookie(&http.Cookie{Name: camino.CookieIdioma, Value: c.cookie})
			}
			idi, persiste := camino.Elegido(r, m)
			if idi != c.quiero {
				t.Errorf("salio %q y se esperaba %q. Por que importa: %s",
					idi, c.quiero, c.porQueImp)
			}
			if persiste != c.persiste {
				t.Errorf("persistir=%v y se esperaba %v. Por que importa: %s",
					persiste, c.persiste, c.porQueImp)
			}
		})
	}
}

// LA COOKIE SOLO SE ESCRIBE CUANDO HAY ELECCION NUEVA, y con sus atributos.
//
// El control positivo y el negativo van en el mismo test a proposito: sin el
// primero, una funcion que no escribiera nunca pasaria el segundo con nota.
func TestLaCookieDelIdiomaLlevaSusAtributos(t *testing.T) {
	m := motorDeMentira{defecto: "es", cargados: []string{"es", "en"}}

	// POSITIVO: con parametro valido se escribe.
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/alcance?idioma=en", nil)
	camino.RecordarIdioma(w, r, m)
	cs := w.Result().Cookies()
	if len(cs) != 1 {
		t.Fatalf("se escribieron %d cookies y se esperaba 1. Sin esta rama recorrida, "+
			"el caso negativo de abajo no demuestra nada", len(cs))
	}
	c := cs[0]
	if c.Name != camino.CookieIdioma || c.Value != "en" {
		t.Errorf("cookie %q=%q", c.Name, c.Value)
	}
	if !c.HttpOnly {
		t.Error("sin HttpOnly: no hay ningun caso en el que el navegador deba leerla")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite=%v y tiene que ser Lax: con Strict, llegar desde el enlace de "+
			"un correo de escalado ensenaria la pagina en otro idioma", c.SameSite)
	}
	if c.Path != "/" {
		t.Errorf("Path=%q: la preferencia vale para todo el sitio", c.Path)
	}
	if c.MaxAge != camino.MaxEdadIdioma {
		t.Errorf("MaxAge=%d y se esperaba %d: una preferencia que se olvida al cerrar "+
			"el navegador se lee como que el conmutador no funciona",
			c.MaxAge, camino.MaxEdadIdioma)
	}

	// NEGATIVO, en sus dos formas: sin parametro y con parametro que no se
	// entiende. La segunda es la que importa (invariante 8).
	for _, sin := range []string{"/alcance", "/alcance?idioma=zz", "/alcance?idioma="} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, sin, nil)
		camino.RecordarIdioma(w, r, m)
		if n := len(w.Result().Cookies()); n != 0 {
			t.Errorf("%s escribio %d cookie(s) y no debia: persistir algo que nadie "+
				"eligio deja una preferencia puesta para un ano", sin, n)
		}
	}
}

// EL ENLACE DE CADA IDIOMA CONSERVA LA PAGINA Y LA CONSULTA QUE LE DEN, y NO la
// que trae la peticion.
//
// La segunda mitad costo un rojo: la primera version leia `r.URL.Query()` y
// `TestUnaRespuestaInventadaNoEntraEnLaPagina` la puso roja, porque copiaba
// entrada adversaria al HTML.
func TestElConmutadorConservaLaPaginaYLaConsultaQueLeDan(t *testing.T) {
	m := motorDeMentira{defecto: "es", cargados: []string{"es", "en"}}

	ops := camino.OpcionesDeIdioma("/alcance", m, "es", "si=q1&no=q2")
	if len(ops) != 2 {
		t.Fatalf("salieron %d opciones y hay 2 idiomas", len(ops))
	}
	for _, o := range ops {
		if !strings.HasPrefix(o.URL, "/alcance?") {
			t.Errorf("%s no vuelve a la misma pagina: %q. Un conmutador que te manda a "+
				"la portada te obliga a volver a navegar y se deja de usar", o.Codigo, o.URL)
		}
		if !strings.Contains(o.URL, "si=q1") || !strings.Contains(o.URL, "no=q2") {
			t.Errorf("%s pierde la entrevista: %q", o.Codigo, o.URL)
		}
		if !strings.Contains(o.URL, "idioma="+o.Codigo) {
			t.Errorf("%s no pide su idioma: %q", o.Codigo, o.URL)
		}
	}
	if !ops[0].Actual || ops[1].Actual {
		t.Errorf("el actual mal marcado: %v", ops)
	}

	// EL VALOR CERO: con un solo idioma no hay conmutador.
	uno := motorDeMentira{defecto: "es", cargados: []string{"es"}}
	if o := camino.OpcionesDeIdioma("/alcance", uno, "es", ""); o != nil {
		t.Errorf("con un solo idioma salieron %d opciones. Un conmutador de una opcion "+
			"no conmuta y sugiere que hay algo que no hay", len(o))
	}

	// Y una consulta que no se entiende no se cuela: se descarta entera.
	rara := camino.OpcionesDeIdioma("/alcance", m, "es", "%zz")
	for _, o := range rara {
		if strings.Contains(o.URL, "%zz") {
			t.Errorf("una consulta ilegible acabo en el enlace: %q", o.URL)
		}
	}
}

// motorDeMentira es el MotorDeIdiomas mas corto que sirve.
type motorDeMentira struct {
	defecto  string
	cargados []string
}

func (m motorDeMentira) Idiomas() []string { return m.cargados }

func (m motorDeMentira) Resolver(pedido string) string {
	p := strings.ToLower(strings.TrimSpace(pedido))
	if i := strings.IndexAny(p, ",;"); i >= 0 {
		p = p[:i]
	}
	if i := strings.IndexAny(p, "-_"); i >= 0 {
		p = p[:i]
	}
	for _, c := range m.cargados {
		if c == p {
			return c
		}
	}
	return m.defecto
}
