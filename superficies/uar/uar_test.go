package uar

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/adaptadores/catalogo"
	"github.com/marcosmatalab/plazum/nucleo/accesos"
	"github.com/marcosmatalab/plazum/nucleo/censo"
	"github.com/marcosmatalab/plazum/nucleo/ledger"
)

var (
	t0 = time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC)
	t1 = t0.Add(time.Hour)
)

const censoBase = "usuario;nombre;permiso\n" +
	"u1;Ana Martinez;admin\n" +
	"u1;Ana Martinez;lector\n" +
	"u2;Luis Gil;lector\n"

// fuente es un doble de Campanas que guarda los hechos en memoria.
type fuente struct {
	c       *accesos.Campana
	err     error
	anotado []ledger.Entrada
}

func (f *fuente) Abierta() (*accesos.Campana, error) { return f.c, f.err }
func (f *fuente) Anotar(e ledger.Entrada) error      { f.anotado = append(f.anotado, e); return nil }

func instantanea(t *testing.T, texto string) censo.Instantanea {
	t.Helper()
	ins, err := censo.Tomar([]byte(texto), censo.Opciones{
		Sistema: "erp", Fuente: "export", Quien: "u-042", Tomada: t0,
		Retencion: "12 meses", Columnas: censo.ColumnasHabituales(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return ins
}

func campana(t *testing.T, texto string, revisores map[string]string) *accesos.Campana {
	t.Helper()
	c, err := accesos.Abrir("uar-h2", instantanea(t, texto), t0, revisores)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func cat(t *testing.T) *catalogo.Catalogo {
	t.Helper()
	c, err := catalogo.Nuevo()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func superficie(t *testing.T, f Campanas, conToken bool) *Superficie {
	t.Helper()
	o := Opciones{
		Fuente: f, Catalogo: cat(t), Base: "/uar", Estatico: "/estatico",
		Ahora: func() time.Time { return t1 },
		Quien: func(*http.Request) string { return "ciso" },
	}
	if conToken {
		o.Tokens = func(*http.Request) (string, error) { return "tok-123", nil }
	}
	s, err := Nuevo(o)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func pedir(t *testing.T, s *Superficie, metodo, ruta string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if form == nil {
		r = httptest.NewRequest(metodo, ruta, nil)
	} else {
		r = httptest.NewRequest(metodo, ruta, strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}

// EL CONTRATO DE CLAVES, EN LAS DOS DIRECCIONES.
//
// Es la misma puerta que tiene superficies/pantallas y por el mismo motivo: una
// clave que la plantilla pide y el contrato no declara sale en crudo en la
// pantalla de un cliente; una que el contrato declara y nadie pide es peso
// muerto que hay que traducir a cada idioma nuevo.
func TestElContratoDeClavesCasaConLoQuePideLaPlantilla(t *testing.T) {
	b, err := plantillasFS.ReadFile("plantillas/uar.html")
	if err != nil {
		t.Fatal(err)
	}
	// Las claves que la plantilla pide de verdad: `t "clave"` y `t .Campo` no
	// cuenta (esas las pone el codigo y estan en la tabla de clavePorEstado).
	re := regexp.MustCompile(`t "([a-z][a-z0-9_]*(?:[.][a-z0-9_]+)+)"`)
	pide := map[string]bool{}
	for _, m := range re.FindAllStringSubmatch(string(b), -1) {
		pide[m[1]] = true
	}
	// Y las que pone el codigo Go, que la plantilla no puede declarar.
	for _, k := range clavePorEstado {
		pide[k] = true
	}
	for _, k := range []string{"uar.error.render", "uar.sin_campana.titulo", "uar.aviso.sin_autor",
		"uar.titulo", "uar.excusar.no_es_numero"} {
		pide[k] = true
	}
	if len(pide) < 30 {
		t.Fatalf("solo se han encontrado %d claves en la plantilla: o se ha vaciado, o esta "+
			"expresion ha dejado de reconocerlas y este test estaria midiendo el vacio", len(pide))
	}
	declara := map[string]bool{}
	for _, k := range ClavesDeCatalogo() {
		declara[k] = true
	}
	for k := range pide {
		if !declara[k] {
			t.Errorf("la plantilla pide %q y el contrato no la declara: saldria en crudo en la "+
				"pantalla de un cliente", k)
		}
	}
	for k := range declara {
		if !pide[k] {
			t.Errorf("el contrato declara %q y no la pide nadie: es peso muerto que hay que "+
				"traducir a cada idioma nuevo", k)
		}
	}
}

// LA FRASE DEL DESCARGO DICE LO MISMO EN LOS DOS SITIOS.
//
// nucleo/accesos la escribe para el informe de terminal y el catalogo para la
// pantalla, y son dos sitios. Se comparan SIN TILDES a proposito: los literales
// de Go de este repositorio van sin acentos y el catalogo va con ellos, que es
// la convencion de cada uno. Lo que este test vigila no es la ortografia, es que
// no se separen las PALABRAS.
func TestLaFraseDelDescargoEsLaMismaEnElNucleoYEnLaPantalla(t *testing.T) {
	enPantalla := cat(t).Traducir("es", "uar.no_consta")
	if enPantalla == "uar.no_consta" {
		t.Fatal("la clave no existe en el catalogo espanol")
	}
	if sinTildes(enPantalla) != sinTildes(accesos.LaFraseDeLoNoRevisado) {
		t.Errorf("las dos frases se han separado.\n  nucleo:   %s\n  catalogo: %s\n"+
			"  Una frase que vive en dos sitios se corrige en uno, y la que se queda vieja es "+
			"la que acusa en falso", accesos.LaFraseDeLoNoRevisado, enPantalla)
	}
}

func sinTildes(s string) string {
	r := strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ñ", "n",
		"Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U", "Ñ", "N")
	return r.Replace(s)
}

// PUERTA D11-b: EL ESTADO VACIO TRAE SU SIGUIENTE PASO.
//
// Sin campana configurada, esta pantalla no dice "no hay datos": dice la orden
// exacta que hay que teclear. Una pantalla vacia sin verbo es un callejon.
func TestSinCampanaLaPantallaDiceExactamenteQueHacer(t *testing.T) {
	s := superficie(t, nil, true)
	w := pedir(t, s, "GET", "/uar/", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("codigo %d", w.Code)
	}
	cuerpo := w.Body.String()
	for _, quiero := range []string{
		"plazum accesos ver --fichero",
		"plazum serve --accesos-fichero",
		"no guarda tu lista de personas",
	} {
		if !strings.Contains(cuerpo, quiero) {
			t.Errorf("el estado vacio no dice %q:\n%s", quiero, cuerpo)
		}
	}
	// Y NO ensena formularios de algo que no existe.
	if strings.Contains(cuerpo, "<form") {
		t.Error("hay formularios en una pantalla sin campana")
	}
}

// PUERTA D11-c: NINGUNA CIFRA QUEDA HUERFANA DE ENLACE.
//
// Cada cubo es un recuento CON su derivacion: pulsarlo ensena que accesos lo
// componen. Una cifra sin enlace obliga a fiarse.
func TestCadaCuboLlevaSuEnlaceALaDerivacion(t *testing.T) {
	c := campana(t, censoBase, nil)
	s := superficie(t, &fuente{c: c}, true)
	cuerpo := pedir(t, s, "GET", "/uar/", nil).Body.String()

	// Los cuatro cubos, incluidos los que valen cero: uno que solo aparece
	// cuando tiene algo dentro es un cubo que nadie echa de menos.
	for _, e := range accesos.EstadosPosibles() {
		enlace := "/uar/?cubo=" + enlaceDe(e)
		if !strings.Contains(cuerpo, enlace) {
			t.Errorf("el cubo %q no lleva enlace (%s):\n%s", e, enlace, cuerpo)
		}
	}
	// Y el enlace filtra de verdad.
	filtrado := pedir(t, s, "GET", "/uar/?cubo=sin-revisar", nil).Body.String()
	if !strings.Contains(filtrado, "u1") || !strings.Contains(filtrado, "u2") {
		t.Errorf("el filtro de sin-revisar no ensena los tres accesos:\n%s", filtrado)
	}
	if err := c.Registrar(accesos.Decision{Fila: "erp|u1|admin", Veredicto: accesos.Aprobar,
		Quien: "j", Cuando: t1}); err != nil {
		t.Fatal(err)
	}
	aprobados := pedir(t, s, "GET", "/uar/?cubo=aprobada", nil).Body.String()
	if !strings.Contains(aprobados, "admin") {
		t.Errorf("el filtro de aprobada no ensena el acceso aprobado:\n%s", aprobados)
	}
	if strings.Contains(aprobados, ">u2<") {
		t.Error("el filtro de aprobada ensena un acceso que no lo esta")
	}
}

// LA FRASE DEL DESCARGO, CON SUS DOS CONTROLES.
//
// El positivo: con accesos sin revisar, esta. El negativo, que importa igual:
// con todo revisado NO esta, porque una frase que sale siempre deja de leerse y
// entonces no protege de nada.
func TestLaPantallaNoAcusaDeLoQueSoloEsUnDatoQueFalta(t *testing.T) {
	c := campana(t, censoBase, nil)
	s := superficie(t, &fuente{c: c}, true)
	frase := cat(t).Traducir("es", "uar.no_consta")

	conPendientes := pedir(t, s, "GET", "/uar/", nil).Body.String()
	if !strings.Contains(conPendientes, frase) {
		t.Fatalf("la pantalla ensena accesos sin revisar SIN el descargo:\n%s", conPendientes)
	}
	for _, f := range c.Instantanea().Filas {
		if err := c.Registrar(accesos.Decision{Fila: f.Clave(), Veredicto: accesos.Aprobar,
			Quien: "j", Cuando: t1}); err != nil {
			t.Fatal(err)
		}
	}
	limpio := pedir(t, s, "GET", "/uar/", nil).Body.String()
	if strings.Contains(limpio, frase) {
		t.Errorf("el descargo sale con todo revisado. Una frase que sale siempre deja de "+
			"leerse:\n%s", limpio)
	}
}

// SIN TOKEN NO SE PINTA NINGUN FORMULARIO.
//
// Es el invariante 8 en una frontera de construccion: el valor cero de "no se
// como emitir un token" tiene que ser "no ensenes botones", no "ensenalos sin
// token". Un boton que no puede funcionar es peor que ninguno: quien lo pulse
// creera que ha decidido.
func TestSinTokenNoSePintaNingunFormulario(t *testing.T) {
	c := campana(t, censoBase, nil)
	sin := superficie(t, &fuente{c: c}, false)
	cuerpo := pedir(t, sin, "GET", "/uar/", nil).Body.String()
	if strings.Contains(cuerpo, "<form") {
		t.Errorf("hay formularios sin token CSRF:\n%s", cuerpo)
	}
	if !strings.Contains(cuerpo, cat(t).Traducir("es", "uar.sin_token")) {
		t.Errorf("y ademas no se dice por que faltan los botones:\n%s", cuerpo)
	}
	// Con token si, y el token va DENTRO del formulario.
	con := superficie(t, &fuente{c: c}, true)
	cuerpo = pedir(t, con, "GET", "/uar/", nil).Body.String()
	if !strings.Contains(cuerpo, `name="csrf" value="tok-123"`) {
		t.Errorf("el formulario no lleva el token:\n%s", cuerpo)
	}
}

// EL CICLO POR LA PANTALLA: decidir, y el hecho sale al ledger.
func TestDecidirDesdeLaPantallaAnotaElHecho(t *testing.T) {
	c := campana(t, censoBase, nil)
	f := &fuente{c: c}
	s := superficie(t, f, true)
	w := pedir(t, s, "POST", "/uar/decidir", url.Values{
		"fila": {"erp|u1|admin"}, "veredicto": {"revocar"}, "motivo": {"cambio de puesto"},
	})
	if w.Code != http.StatusSeeOther {
		t.Fatalf("codigo %d, se esperaba una redireccion: %s", w.Code, w.Body.String())
	}
	if c.EstadoDe("erp|u1|admin") != accesos.Revocada {
		t.Fatalf("la decision no ha entrado: %v", c.Cuenta())
	}
	if len(f.anotado) != 1 || f.anotado[0].Tipo != accesos.TipoDecision {
		t.Fatalf("no se ha anotado el hecho: %+v", f.anotado)
	}
	if f.anotado[0].Actor != "ciso" {
		t.Errorf("el hecho no lleva quien decidio: %q", f.anotado[0].Actor)
	}
	// Y LA CARGA NO LLEVA LA CLAVE, lleva su huella.
	if strings.Contains(string(f.anotado[0].Carga), "u1") {
		t.Errorf("el identificador de la cuenta viaja en el ledger:\n%s", f.anotado[0].Carga)
	}
}

// SE REDIRIGE DESPUES DE MUTAR, y no se pinta directamente.
//
// Con un formulario de revocacion, recargar la pagina despues de decidir es la
// diferencia entre revocar una vez y revocar cada vez que alguien pulsa F5.
func TestDespuesDeDecidirSeRedirigeParaQueRecargarNoRepita(t *testing.T) {
	c := campana(t, censoBase, nil)
	s := superficie(t, &fuente{c: c}, true)
	w := pedir(t, s, "POST", "/uar/decidir", url.Values{
		"fila": {"erp|u2|lector"}, "veredicto": {"aprobar"},
	})
	if w.Code != http.StatusSeeOther {
		t.Fatalf("codigo %d", w.Code)
	}
	if l := w.Header().Get("Location"); l != "/uar/" {
		t.Errorf("redirige a %q", l)
	}
}

// UNA DECISION MAL FORMADA NO SE ANOTA, y se dice en la propia pantalla.
//
// Sacar a alguien de la pantalla para decirle que le falta el motivo le hace
// perder lo que estaba mirando.
func TestUnaDecisionMalFormadaNoSeAnotaYSeDiceEnLaPantalla(t *testing.T) {
	c := campana(t, censoBase, nil)
	f := &fuente{c: c}
	s := superficie(t, f, true)
	w := pedir(t, s, "POST", "/uar/decidir", url.Values{
		"fila": {"erp|u1|admin"}, "veredicto": {"revocar"}, // sin motivo
	})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("codigo %d", w.Code)
	}
	if len(f.anotado) != 0 {
		t.Fatalf("se ha anotado una decision que se rechazo: %+v", f.anotado)
	}
	cuerpo := w.Body.String()
	if !strings.Contains(cuerpo, "motivo") {
		t.Errorf("no se dice que falta el motivo:\n%s", cuerpo)
	}
	// Y la pantalla sigue estando: el aviso va DENTRO, no en una pagina suelta.
	if !strings.Contains(cuerpo, "erp|u1|admin") {
		t.Errorf("se ha sacado al operador de la pantalla:\n%s", cuerpo)
	}
}

// UN HECHO SIN AUTOR NO ES UN HECHO.
func TestSinSaberQuienOperaNoSeAdmiteNingunaDecision(t *testing.T) {
	c := campana(t, censoBase, nil)
	f := &fuente{c: c}
	s, err := Nuevo(Opciones{Fuente: f, Catalogo: cat(t), Base: "/uar",
		Ahora:  func() time.Time { return t1 },
		Tokens: func(*http.Request) (string, error) { return "tok", nil },
		// Quien nil: nadie sabe quien esta operando.
	})
	if err != nil {
		t.Fatal(err)
	}
	w := pedir(t, s, "POST", "/uar/decidir", url.Values{
		"fila": {"erp|u2|lector"}, "veredicto": {"aprobar"},
	})
	if w.Code == http.StatusSeeOther {
		t.Fatal("se ha admitido una decision sin saber quien la toma")
	}
	if len(f.anotado) != 0 {
		t.Fatalf("se ha anotado: %+v", f.anotado)
	}
	// Y LO PARA ESTA PANTALLA, no el nucleo por detras.
	//
	// La primera version solo exigia que NO pasara, y la mutacion que quitaba la
	// comprobacion de la superficie salio VERDE: el nucleo rechaza igual una
	// decision sin autor, asi que la afirmacion viajaba de gorra en un fallo
	// ajeno. Es M95 otra vez. Se exige el mensaje de la pantalla, que ademas es
	// el que esta en los dos idiomas: el del nucleo es un error en espanol
	// dentro de una interfaz que puede estar en ingles.
	quiero := cat(t).Traducir("es", "uar.aviso.sin_autor")
	if !strings.Contains(w.Body.String(), quiero) {
		t.Errorf("lo ha parado otra cosa y no esta pantalla. Se esperaba:\n  %s\ny salio:\n%s",
			quiero, w.Body.String())
	}
}

// EL CIERRE BLOQUEADO ENSENA POR QUE, en la pantalla y no en un error suelto.
func TestElCierreBloqueadoDiceQueFaltaSinSacarteDeLaPantalla(t *testing.T) {
	c := campana(t, censoBase, nil)
	f := &fuente{c: c}
	s := superficie(t, f, true)
	w := pedir(t, s, "POST", "/uar/cerrar", url.Values{})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("codigo %d", w.Code)
	}
	if len(f.anotado) != 0 {
		t.Fatalf("se ha anotado un cierre que no se pudo hacer: %+v", f.anotado)
	}
	if !strings.Contains(w.Body.String(), "sin revisar") {
		t.Errorf("no dice que falta:\n%s", w.Body.String())
	}
}

// LAS RUTAS QUE MUTAN SON EXACTAMENTE LAS DECLARADAS.
//
// Patrones() existe para que la puerta de CSRF de superficies/serve las enumere
// sin conocer este paquete. Si una ruta mutante nueva no aparece ahi, se cuela
// por debajo de la puerta.
func TestLasRutasMutantesSonLasDeclaradasYNingunaMas(t *testing.T) {
	s := superficie(t, &fuente{c: campana(t, censoBase, nil)}, true)
	ps := s.Patrones()
	sort.Strings(ps)
	quiero := []string{"GET /uar/{$}", "POST /uar/cerrar", "POST /uar/decidir", "POST /uar/excusar"}
	if strings.Join(ps, "|") != strings.Join(quiero, "|") {
		t.Fatalf("rutas: %v, esperadas %v", ps, quiero)
	}
	// Y ninguna de las mutantes atiende un GET: si lo hiciera, se podria
	// disparar desde un enlace y el CSRF no la veria.
	for _, r := range []string{"/uar/decidir", "/uar/excusar", "/uar/cerrar"} {
		w := pedir(t, s, "GET", r, nil)
		if w.Code == http.StatusOK {
			t.Errorf("%s atiende GET: se podria disparar desde un enlace", r)
		}
	}
}

// LA PANTALLA NO DECIDE NADA DEL DOMINIO: pregunta.
//
// Se comprueba por donde mas duele: los cubos de la pantalla tienen que ser los
// del nucleo, no un recuento propio. Dos recuentos son dos motores.
func TestLosCubosDeLaPantallaSonLosDelNucleo(t *testing.T) {
	c := campana(t, censoBase, nil)
	if err := c.Registrar(accesos.Decision{Fila: "erp|u1|admin", Veredicto: accesos.Aprobar,
		Quien: "j", Cuando: t1}); err != nil {
		t.Fatal(err)
	}
	var v Vista
	v.Base = "/uar"
	v.rellenarCon(c, "es", cat(t))
	deNucleo := c.Cuenta()
	for _, cu := range v.Cubos {
		if cu.N != deNucleo[accesos.Estado(cu.Estado)] {
			t.Errorf("el cubo %q dice %d y el nucleo %d", cu.Estado, cu.N,
				deNucleo[accesos.Estado(cu.Estado)])
		}
	}
	suma := 0
	for _, cu := range v.Cubos {
		suma += cu.N
	}
	if suma != v.Accesos {
		t.Errorf("los cubos de la pantalla suman %d y hay %d accesos", suma, v.Accesos)
	}
}

// CERO JAVASCRIPT EN LINEA, CERO style= Y CERO on*.
//
// Es la regla de las plantillas de plazum, y aqui vale igual: con una CSP
// estricta ("script-src 'self'; style-src 'self'") nada de eso se ejecuta, y una
// pantalla que depende de ello se ve rota sin decir por que.
func TestLaPantallaNoLlevaNadaQueUnaCSPEstrictaBloquee(t *testing.T) {
	s := superficie(t, &fuente{c: campana(t, censoBase, nil)}, true)
	cuerpo := pedir(t, s, "GET", "/uar/", nil).Body.String()
	for _, prohibido := range []string{"<script>", " style=", " onclick=", " onsubmit=", "javascript:"} {
		if strings.Contains(cuerpo, prohibido) {
			t.Errorf("la pantalla lleva %q, que una CSP estricta bloquea", prohibido)
		}
	}
}

// EL P1 EN LA SUPERFICIE: el numero que no se entiende NO se convierte en cero.
//
// La primera version hacia `desde, _ := strconv.Atoi(...)`, asi que un campo
// vacio, ausente o con letras se volvia cero en silencio y una excusa {0,0} --
// que no excusa nada -- se iba al ledger append-only para siempre. Las dos
// formas de la nada (ausente y presente-vacio) se recorren aqui las dos, y la
// tercera cosa que Atoi confundia con ellas (presente y no numerico) tiene que
// dar 422 con aviso.
func TestUnaExcusaConUnNumeroQueNoSeEntiendeNoSeConvierteEnCero(t *testing.T) {
	const con = "usuario;nombre;permiso\nu1;Ana;admin\n;Sin Cuenta;lector\nu2;Luis;lector\n"

	casos := []struct {
		nombre string
		form   url.Values
		dice   string
	}{
		{"desde ausente", url.Values{"motivo": {"x"}}, "no es un número de línea"},
		{"desde presente y vacio", url.Values{"desde": {""}, "motivo": {"x"}}, "no es un número de línea"},
		{"desde con letras", url.Values{"desde": {"tres"}, "motivo": {"x"}}, "no es un número de línea"},
		{"hasta con letras", url.Values{"desde": {"3"}, "hasta": {"x"}, "motivo": {"x"}}, "no es un número de línea"},
		// Y el barrido, que la pantalla tiene que rechazar igual aunque el
		// `min="1" required` del HTML lo dejaria pasar por curl.
		{"el rango de barrido", url.Values{"desde": {"1"}, "hasta": {"999999"}, "motivo": {"x"}},
			"se pasa del final"},
		{"una linea legible", url.Values{"desde": {"2"}, "motivo": {"x"}},
			"esconde un acceso que habria que revisar"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			c := campana(t, con, nil)
			f := &fuente{c: c}
			s := superficie(t, f, true)
			w := pedir(t, s, "POST", "/uar/excusar", caso.form)
			if w.Code != http.StatusUnprocessableEntity {
				t.Fatalf("codigo %d, se esperaba 422", w.Code)
			}
			if len(f.anotado) != 0 {
				t.Fatalf("se ha anotado una excusa que se rechazo: %+v", f.anotado)
			}
			if !strings.Contains(w.Body.String(), caso.dice) {
				t.Errorf("el aviso no dice %q:\n%s", caso.dice, w.Body.String())
			}
		})
	}

	// CONTROL POSITIVO: la excusa legitima sigue pasando y llega al ledger. Sin
	// esto, todo lo de arriba se cumpliria rechazandolo todo.
	c := campana(t, con, nil)
	f := &fuente{c: c}
	s := superficie(t, f, true)
	w := pedir(t, s, "POST", "/uar/excusar", url.Values{
		"desde": {"3"}, "motivo": {"fila de prueba del IdP"},
	})
	if w.Code != http.StatusSeeOther {
		t.Fatalf("la excusa legitima da %d: %s", w.Code, w.Body.String())
	}
	if len(f.anotado) != 1 || f.anotado[0].Tipo != accesos.TipoExcusa {
		t.Fatalf("no se ha anotado: %+v", f.anotado)
	}
	// Y "hasta" vacio significa "la misma linea", que es el caso normal: las dos
	// formas de la nada valen lo mismo AQUI, y eso es una decision, no un
	// descuido.
	if len(c.Informar().Excusas) != 1 || c.Informar().Excusas[0].Hasta != 3 {
		t.Fatalf("hasta vacio no ha caido en la misma linea: %+v", c.Informar().Excusas)
	}
}
