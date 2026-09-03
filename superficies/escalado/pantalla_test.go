package escalado

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	nescalado "github.com/marcosmatalab/plazum/nucleo/escalado"
	"github.com/marcosmatalab/plazum/superficies/camino"
)

// EL CATALOGO ESPIA. Mismo motivo que en la pantalla del calendario: las cadenas
// de esta pantalla las redacta otro frente, asi que el catalogo real devuelve la
// clave tal cual y un test que comprobara «contiene la frase» pasaria por
// contener la clave. El espia traduce a un marcador inconfundible y ANOTA que se
// pide, que es lo unico que un catalogo ausente no puede falsear.
type catalogoEspia struct {
	mu      sync.Mutex
	pedidas map[string]bool
}

func nuevoEspia() *catalogoEspia { return &catalogoEspia{pedidas: map[string]bool{}} }

func (c *catalogoEspia) Traducir(idioma, clave string, args ...any) string {
	c.mu.Lock()
	c.pedidas[clave] = true
	c.mu.Unlock()
	if len(args) == 0 {
		return "[[" + clave + "]]"
	}
	return "[[" + clave + fmt.Sprintf("%v", args) + "]]"
}

func (c *catalogoEspia) Idiomas() []string         { return []string{"es"} }
func (c *catalogoEspia) Faltantes(string) []string { return nil }
func (c *catalogoEspia) pidio(clave string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.pedidas[clave]
}
func (c *catalogoEspia) claves() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, 0, len(c.pedidas))
	for k := range c.pedidas {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func marca(clave string) string { return "[[" + clave + "]]" }

func dia(a, m, d int) time.Time { return time.Date(a, time.Month(m), d, 9, 0, 0, 0, time.UTC) }

// fuenteDoble es un doble de Fuente. NO MANDA NADA, igual que la de verdad: el
// interfaz no tiene por donde.
type fuenteDoble struct {
	p   Plan
	hay bool
	err error
}

func (f fuenteDoble) EnSeco() (Plan, bool, error) { return f.p, f.hay, f.err }

// planDePrueba recorre las dos mitades de un escalon: el que saldria (lleva
// Aviso) y el que NO sale (lleva estado y motivo). Sin las dos, la mitad de la
// plantilla no la ejecuta nadie.
func planDePrueba() Plan {
	return Plan{
		Organizacion: "Acme SL",
		ComoMandar:   "plazum escalado --alcance alcance.json --mandar",
		Trabajos: []Trabajo{{
			Obligacion: "m1.o1", Titulo: "Notificacion de incidente grave",
			Hito: "inicial", Vence: dia(2026, 10, 1),
			Pasos: []nescalado.Paso{
				{
					Nivel: 1, Cuando: dia(2026, 9, 24), Figura: "m1.responsable",
					Persona: "Bea Nunez", Estado: nescalado.Pendiente,
					Aviso: &nescalado.Aviso{
						Obligacion: "m1.o1", Titulo: "Notificacion de incidente grave",
						Hito: "inicial", Vence: dia(2026, 10, 1),
						Figura: "m1.responsable", Nivel: 1, Enlace: "http://localhost:8443/x",
					},
				},
				{
					Nivel: 2, Cuando: dia(2026, 9, 30), Figura: "m1.direccion",
					Estado: nescalado.SinDestinatario,
					Motivo: "la figura m1.direccion no tiene persona en esta organizacion",
				},
			},
		}},
		Cuenta: map[nescalado.Estado]int{
			nescalado.Pendiente: 1, nescalado.SinDestinatario: 1,
		},
		Planificados: 2,
		Faltas: []nescalado.Falta{
			{Figura: "m1.direccion", Titulo: "Direccion", Paquete: "urn:demo:m1"},
		},
	}
}

func pantallaDePrueba(t *testing.T, f Fuente, quien func(*http.Request) string) (
	*Superficie, *catalogoEspia) {

	t.Helper()
	esp := nuevoEspia()
	// CON EL CAMINO ENTERO, como la monta el producto.
	s, err := Nuevo(Opciones{
		Fuente: f, Catalogo: esp, Base: BasePorDefecto, Estatico: "/estatico",
		CaminoRuta: camino.BasePorDefecto + "/", CaminoClave: camino.ClaveTitulo,
		Pasos: camino.Canonico(), Quien: quien,
	})
	if err != nil {
		t.Fatal(err)
	}
	return s, esp
}

func conSesion(*http.Request) string { return "ciso" }

func pedir(t *testing.T, s *Superficie, metodo, ruta string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(metodo, ruta, nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

// LA PROPIEDAD QUE SOSTIENE ESTA PANTALLA ENTERA: NO MANDA NADA.
//
// Es la unica pieza del producto cuyo efecto sale de la organizacion, y un aviso
// disparado por error llega al correo de una persona. La comprobacion no es «he
// leido el codigo y no hay envio»: es que no exista ninguna via.
//
//	ninguna ruta registrada es distinta de GET;
//	el HTML no trae ni un <form, ni un <button, ni un method=;
//	y todo metodo mutante contra la ruta se rechaza.
//
// LAS TRES, porque cada una sola se puede cumplir con las otras rotas: se puede
// registrar solo GET y pintar un formulario que apunte a otra superficie, y se
// puede no pintar formulario y registrar un POST que un cliente llame a mano.
func TestLaPantallaDelEscaladoNoTienePorDondeMandarNada(t *testing.T) {
	s, _ := pantallaDePrueba(t, fuenteDoble{p: planDePrueba(), hay: true}, conSesion)

	ps := s.Patrones()
	if len(ps) == 0 {
		t.Fatal("la pantalla no registra ninguna ruta: esta comprobacion mide el vacio")
	}
	for _, p := range ps {
		if !strings.HasPrefix(p, "GET ") {
			t.Errorf("la ruta %q no es GET. Esta pantalla no puede tener una sola ruta "+
				"mutante: el dia que exista un boton de mandar sera una superficie detras "+
				"del CSRF, y sera otra decision, no una heredada de esta", p)
		}
	}

	codigo, cuerpo := pedir(t, s, http.MethodGet, BasePorDefecto+"/")
	if codigo != http.StatusOK {
		t.Fatalf("GET %s/ ha respondido %d", BasePorDefecto, codigo)
	}
	for _, prohibido := range []string{"<form", "<button", "method=", "formaction"} {
		if strings.Contains(strings.ToLower(cuerpo), prohibido) {
			t.Errorf("el HTML de la pantalla del escalado contiene %q. Aqui no puede haber "+
				"nada que se pueda enviar: un aviso disparado por error llega al correo de "+
				"una persona y eso no se deshace", prohibido)
		}
	}

	for _, metodo := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		if codigo, _ := pedir(t, s, metodo, BasePorDefecto+"/"); codigo == http.StatusOK {
			t.Errorf("un %s a la pantalla del escalado ha contestado 200", metodo)
		}
	}

	// Y LA PROMESA ESTA ESCRITA EN LA PAGINA, arriba del todo y antes de ningun
	// dato: quien la abre tiene que saber que esto no ha mandado nada ANTES de
	// leer a quien se escribiria.
	if !strings.Contains(cuerpo, marca("escalado.pantalla.en_seco")) {
		t.Errorf("la pantalla no dice que esta en seco:\n%s", recorta(cuerpo, 800))
	}
	posPromesa := strings.Index(cuerpo, marca("escalado.pantalla.en_seco"))
	posDato := strings.Index(cuerpo, "Bea Nunez")
	if posDato >= 0 && posPromesa > posDato {
		t.Error("la promesa de que no se ha mandado nada sale DESPUES del primer nombre de " +
			"persona. Se lee al reves de como hay que leerla")
	}
}

// SIN SESION NO SE ENSENA EL PLAN.
//
// Aqui hay nombres de personas y el reparto de responsabilidades de cumplimiento
// de la organizacion, que es el mapa que quiere quien prepara un ataque
// dirigido. Y el camino guiado sigue pintandose: es justo el estado en el que
// alguien se queda mirando una pagina que no le dice nada.
func TestSinSesionNoSalenLosNombresYSigueElCamino(t *testing.T) {
	s, _ := pantallaDePrueba(t, fuenteDoble{p: planDePrueba(), hay: true}, nil)
	codigo, cuerpo := pedir(t, s, http.MethodGet, BasePorDefecto+"/")
	if codigo != http.StatusUnauthorized {
		t.Errorf("sin sesion la pantalla contesta %d y tiene que contestar 401", codigo)
	}
	for _, dato := range []string{"Bea Nunez", "m1.responsable", "Acme SL"} {
		if strings.Contains(cuerpo, dato) {
			t.Errorf("sin sesion la pantalla ensena %q, que es dato de personas de la "+
				"organizacion", dato)
		}
	}
	if !strings.Contains(cuerpo, `href="`+camino.BasePorDefecto+`/"`) {
		t.Errorf("sin sesion la pantalla no enlaza de vuelta al camino guiado:\n%s",
			recorta(cuerpo, 600))
	}

	// CONTROL POSITIVO: con sesion, esos mismos datos SI salen. Sin esto, el
	// test de arriba pasaria con una pantalla que no ensena nada nunca.
	conS, _ := pantallaDePrueba(t, fuenteDoble{p: planDePrueba(), hay: true}, conSesion)
	if _, cuerpo := pedir(t, conS, http.MethodGet, BasePorDefecto+"/"); !strings.Contains(
		cuerpo, "Bea Nunez") {
		t.Error("con sesion la pantalla tampoco ensena el nombre de quien recibiria el aviso, " +
			"asi que la comprobacion de arriba se cumple por no ensenar nada nunca")
	}
}

// EL ESTADO VACIO EXISTE Y DICE EL SIGUIENTE PASO (puerta D11-b).
func TestSinAlcanceLaPantallaDelEscaladoExisteIgual(t *testing.T) {
	s, esp := pantallaDePrueba(t, nil, conSesion)
	codigo, cuerpo := pedir(t, s, http.MethodGet, BasePorDefecto+"/")
	if codigo != http.StatusOK {
		t.Fatalf("sin fuente la pantalla contesta %d y tiene que existir", codigo)
	}
	for _, clave := range []string{
		"escalado.pantalla.sin_alcance.titulo",
		"escalado.pantalla.sin_alcance.que_es",
		"escalado.pantalla.sin_alcance.paso",
	} {
		if !esp.pidio(clave) {
			t.Errorf("el estado vacio no pide %q", clave)
		}
	}
	if !strings.Contains(cuerpo, `href="`+camino.BasePorDefecto+`/"`) {
		t.Error("el estado vacio no enlaza de vuelta al camino guiado")
	}
}

// UN FALLO AL PLANIFICAR NO SE CONVIERTE EN «no hay nada que avisar».
func TestUnFalloAlPlanificarNoSePintaComoPlanVacio(t *testing.T) {
	s, _ := pantallaDePrueba(t, fuenteDoble{err: errDePrueba{}}, conSesion)
	codigo, cuerpo := pedir(t, s, http.MethodGet, BasePorDefecto+"/")
	if codigo != http.StatusInternalServerError {
		t.Errorf("con la fuente rota la pantalla contesta %d y tiene que contestar 500", codigo)
	}
	if !strings.Contains(cuerpo, "el alcance no se entiende") {
		t.Errorf("la pantalla no dice que ha fallado:\n%s", recorta(cuerpo, 600))
	}
}

type errDePrueba struct{}

func (errDePrueba) Error() string { return "el alcance no se entiende" }

// LA LEY DE CONSERVACION, IMPRESA Y CON SU DESCUADRE VISIBLE.
//
// Si los cubos no suman lo planificado hay avisos que no estan en ningun sitio,
// y eso es un fallo del producto, no del operador. Se ensena donde el operador
// mira: un descuadre que solo se loguea es un aviso perdido en silencio.
func TestElDescuadreDeLosCubosSeVeEnLaPantalla(t *testing.T) {
	p := planDePrueba()
	p.Planificados = 5 // los cubos siguen sumando 2
	s, _ := pantallaDePrueba(t, fuenteDoble{p: p, hay: true}, conSesion)
	_, cuerpo := pedir(t, s, http.MethodGet, BasePorDefecto+"/")
	if !strings.Contains(cuerpo, marca("escalado.pantalla.cuenta.descuadre[2 5]")) {
		t.Errorf("los cubos suman 2 y se planificaron 5, y la pantalla no lo dice con los dos "+
			"numeros:\n%s", recorta(cuerpo, 900))
	}

	// CONTROL NEGATIVO: cuadrando, el aviso NO sale. Sin esto, un aviso que
	// saliera siempre pasaria la comprobacion de arriba y no diria nada.
	s2, _ := pantallaDePrueba(t, fuenteDoble{p: planDePrueba(), hay: true}, conSesion)
	if _, cuerpo := pedir(t, s2, http.MethodGet, BasePorDefecto+"/"); strings.Contains(
		cuerpo, marca("escalado.pantalla.cuenta.descuadre")) {
		t.Error("con los cubos cuadrados la pantalla sigue avisando de descuadre")
	}
}

// EL INVENTARIO DE CLAVES, EN LAS DOS DIRECCIONES. Mismo motivo que en el
// calendario: una clave que se pide y no se publica sale en crudo en la pantalla
// de un cliente, y una publicada que nadie pide es peso muerto en cada idioma.
func TestElInventarioDeClavesDelEscaladoCuadra(t *testing.T) {
	publicadas := map[string]bool{}
	for _, k := range ClavesDeCatalogo() {
		publicadas[k] = true
	}
	if len(publicadas) < 15 {
		t.Fatalf("la pantalla publica %d claves y son muchas menos de las que pinta", len(publicadas))
	}

	esp := nuevoEspia()
	descuadrado := planDePrueba()
	descuadrado.Planificados = 5
	vacio := planDePrueba()
	vacio.Trabajos, vacio.Cuenta, vacio.Planificados, vacio.Faltas = nil, nil, 0, nil
	estados := []struct {
		f     Fuente
		quien func(*http.Request) string
	}{
		{nil, conSesion},
		{fuenteDoble{p: planDePrueba(), hay: true}, nil},
		{fuenteDoble{p: planDePrueba(), hay: true}, conSesion},
		{fuenteDoble{p: descuadrado, hay: true}, conSesion},
		{fuenteDoble{p: vacio, hay: true}, conSesion},
	}
	for _, e := range estados {
		s, err := Nuevo(Opciones{
			Fuente: e.f, Catalogo: esp, Base: BasePorDefecto, Estatico: "/estatico",
			CaminoRuta: camino.BasePorDefecto + "/", CaminoClave: camino.ClaveTitulo,
			Pasos: camino.Canonico(), Quien: e.quien,
		})
		if err != nil {
			t.Fatal(err)
		}
		pedir(t, s, http.MethodGet, BasePorDefecto+"/")
	}
	pedidas := map[string]bool{"escalado.pantalla.error_render": true}
	for _, k := range esp.claves() {
		pedidas[k] = true
	}
	// LOS ROTULOS DE LOS PASOS NO SON DE ESTA PANTALLA: llegan como dato desde
	// camino.Canonico() y los declara el camino. Se eximen por su DECLARACION y
	// no por un prefijo, que eximiria tambien una clave que nadie declara.
	delCamino := map[string]bool{}
	for _, k := range camino.ClavesDeCatalogo() {
		delCamino[k] = true
	}
	if len(delCamino) < 6 {
		t.Fatalf("el camino declara %d claves: esta exencion estaria eximiendo el vacio",
			len(delCamino))
	}

	for k := range pedidas {
		if publicadas[k] || delCamino[k] {
			continue
		}
		t.Errorf("la pantalla pide %q y no la publica ni ClavesDeCatalogo() ni el camino: "+
			"saldra en crudo en la pantalla de un cliente", k)
	}
	for k := range publicadas {
		if !pedidas[k] {
			t.Errorf("ClavesDeCatalogo() publica %q y la pantalla no la pide en ninguno de sus "+
				"cinco estados. O sobra, o hay un estado que este test no recorre", k)
		}
	}
}

// LAS DOS MITADES DEL ENLACE AL CAMINO, O NINGUNA.
func TestMedioEnlaceAlCaminoSeRechaza(t *testing.T) {
	for _, c := range []struct{ ruta, clave string }{
		{camino.BasePorDefecto + "/", ""},
		{"", camino.ClaveTitulo},
		{"//otro-sitio/", camino.ClaveTitulo},
	} {
		if _, err := Nuevo(Opciones{
			Catalogo: nuevoEspia(), CaminoRuta: c.ruta, CaminoClave: c.clave,
		}); err == nil {
			t.Errorf("se acepta CaminoRuta=%q CaminoClave=%q y tenia que rechazarse",
				c.ruta, c.clave)
		}
	}
	if _, err := Nuevo(Opciones{Catalogo: nuevoEspia()}); err != nil {
		t.Errorf("el valor cero del enlace se rechaza: %v", err)
	}
}

func recorta(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// LA BARRA LATERAL ES EL CAMINO CANONICO Y MARCA EL PASO DEL ESCALADO.
//
// Aqui importa mas que en ninguna otra pantalla: esta se abre para contestar una
// pregunta suelta («si esto avisara ahora, ¿a quien escribiria?»), casi nunca
// navegando desde la portada, asi que sin barra no dice de que recorrido forma
// parte ni que hay algo antes.
//
// Y SE COMPRUEBA CON SESION Y SIN ELLA. El estado sin sesion es justo aquel en
// el que alguien se queda mirando una pagina que no le dice nada: si la barra
// solo saliera con sesion, faltaria donde mas hace falta.
func TestLaBarraLateralSaleConSesionYSinElla(t *testing.T) {
	pasos := camino.Canonico()
	if len(pasos) < 6 {
		t.Fatalf("el camino declara %d pasos: este test recorreria casi nada", len(pasos))
	}
	rotuloDelPaso := ""
	for _, p := range pasos {
		if p.ID == camino.IDDelEscalado {
			rotuloDelPaso = marca(p.Titulo)
		}
	}
	if rotuloDelPaso == "" {
		t.Fatalf("el camino canonico ya no declara el paso %q", camino.IDDelEscalado)
	}

	for _, caso := range []struct {
		nombre string
		quien  func(*http.Request) string
	}{
		{"con sesion", conSesion},
		{"sin sesion", nil},
	} {
		s, _ := pantallaDePrueba(t, fuenteDoble{p: planDePrueba(), hay: true}, caso.quien)
		_, cuerpo := pedir(t, s, http.MethodGet, BasePorDefecto+"/")

		antes := -1
		for _, p := range pasos {
			pos := strings.Index(cuerpo, marca(p.Titulo))
			if pos < 0 {
				t.Errorf("%s: el paso %q no sale en la barra lateral", caso.nombre, p.ID)
				continue
			}
			if pos < antes {
				t.Errorf("%s: el paso %q sale fuera de orden", caso.nombre, p.ID)
			}
			antes = pos
		}
		i := strings.Index(cuerpo, `aria-current="step"`)
		if i < 0 {
			t.Errorf("%s: la barra lateral no marca ningun paso como actual", caso.nombre)
			continue
		}
		if strings.Contains(cuerpo[i+1:], `aria-current="step"`) {
			t.Errorf("%s: la barra lateral marca DOS pasos como actual", caso.nombre)
		}
		cola := cuerpo[i:]
		if fin := strings.Index(cola, "</li>"); fin > 0 {
			cola = cola[:fin]
		}
		if !strings.Contains(cola, rotuloDelPaso) {
			t.Errorf("%s: el paso marcado como actual no es el del escalado (%s):\n%s",
				caso.nombre, rotuloDelPaso, recorta(cola, 400))
		}
	}
}
