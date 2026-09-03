package calendario

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/pantalla"
	"github.com/marcosmatalab/plazum/nucleo/ventana"
	"github.com/marcosmatalab/plazum/superficies/camino"
)

// EL CATALOGO ESPIA, y por que no se usa el de verdad.
//
// Los tests del acta usan `catalogo.Nuevo()` y comprueban texto traducido. Aqui
// no se puede todavia: las cadenas de esta pantalla las redacta otro frente, asi
// que el catalogo real devuelve la clave tal cual. Un test que comprobara
// «contiene la frase del descargo» pasaria por contener la clave, que es
// exactamente el fallo que se quiere cazar.
//
// El espia resuelve las dos mitades: traduce a un marcador inconfundible (para
// poder afirmar que la pantalla PINTA lo que pide) y ANOTA cada clave pedida
// (para poder afirmar QUE pide, que es lo unico que un catalogo ausente no
// puede falsear).
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

// marca es como sale una clave por la plantilla con el espia puesto.
func marca(clave string) string { return "[[" + clave + "]]" }

// fuenteDoble es un doble de Fuente.
type fuenteDoble struct {
	d   Derivado
	hay bool
	err error
}

func (f fuenteDoble) Actual() (Derivado, bool, error) { return f.d, f.hay, f.err }

func dia(a, m, d int) time.Time { return time.Date(a, time.Month(m), d, 12, 0, 0, 0, time.UTC) }

func pantallaDePrueba(t *testing.T, f Fuente) (*Superficie, *catalogoEspia) {
	t.Helper()
	esp := nuevoEspia()
	// CON EL CAMINO ENTERO, como la monta el producto: la barra lateral sale del
	// canonico y no de una lista escrita aqui.
	s, err := NuevaPantalla(OpcionesPantalla{
		Fuente: f, Catalogo: esp, Base: BasePorDefecto, Estatico: "/estatico",
		CaminoRuta: camino.BasePorDefecto + "/", CaminoClave: camino.ClaveTitulo,
		Pasos: camino.Canonico(),
		Ahora: func() time.Time { return dia(2026, 9, 3) },
	})
	if err != nil {
		t.Fatal(err)
	}
	return s, esp
}

func pedir(t *testing.T, s *Superficie, ruta string) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, ruta, nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

// calendarioConVencidas es el dato sintetico que recorre TODAS las ramas de la
// pantalla, incluida la de acusacion.
//
// POR QUE SINTETICO Y POR QUE COMPLETO. Toda rama de acusacion o de descargo
// exige un control POSITIVO que la recorra: un descargo que ninguna entrada
// alcanza es un descargo que no existe, y una mutacion lo deja verde porque no
// hay nada que romper. La rama de vencidos es justo esa, y ademas es la que no
// se alcanza con un corpus recien instalado, que es el estado en el que estan
// los tests que no la buscan a proposito.
func calendarioConVencidas() pantalla.Calendario {
	return pantalla.Calendario{
		Desde: dia(2026, 9, 3), Hasta: dia(2027, 9, 3),
		Vencidas: []pantalla.Vencida{{
			Desde: dia(2024, 1, 15), Ciclos: 3, Marco: "urn:demo:m1", Obligacion: "m1.o1",
			Titulo: "Revision anual del plan", Articulo: "art. 7.1", Hito: "revision",
			Regla: "cadencia anual desde ultima_revision",
		}},
		Meses: []pantalla.Mes{{
			Ano: 2026, Mes: time.October, Clave: "ui.mes.10",
			Fechas: []pantalla.Fecha{{
				Vence: dia(2026, 10, 20), Marco: "urn:demo:m1", Obligacion: "m1.o2",
				Titulo: "Informe trimestral", Articulo: "art. 9", Hito: "informe",
				// SUPUESTA A PROPOSITO: la marca de conjetura es una rama de la
				// plantilla, y sin ninguna fila que la lleve esa rama no la
				// recorre nadie. Lo encontro el inventario de claves de abajo,
				// que nacio rojo por esto.
				Supuesta: true,
				Regla:    "plazo de 30 dias desde el cierre", Aviso: "el traslado cae en festivo",
				Divergencias: []ventana.Divergencia{
					{Lectura: "dias habiles", Vence: dia(2026, 10, 28)},
				},
			}},
		}},
		Estrenos: []pantalla.Estreno{{
			Desde: dia(2026, 11, 1), Marco: "urn:demo:m2", Obligacion: "m2.o1",
			Titulo: "Notificacion de incidente", Articulo: "art. 14", Hitos: 2,
		}},
		Ceses: []pantalla.Cese{{
			Hasta: dia(2027, 1, 1), Marco: "urn:demo:m3", Obligacion: "m3.o1",
			Titulo: "Registro transitorio", Articulo: "disp. trans. 2", Hitos: 1,
		}},
		SinFecha: []pantalla.SinFecha{{
			Marco: "urn:demo:m1", Obligacion: "m1.o3", Titulo: "Copias de seguridad",
			Articulo: "art. 12", Hito: "copia", Motivo: pantalla.MotivoPendienteDeHecho,
			Regla: "espera la fecha de la ultima copia",
		}},
		HitosDelCorpus: 40, HitosEnVigor: 30, HitosAplicables: 5,
		MasAllaDeLaVentana: 2, VencimientosPasados: 3, VencimientosAntesDeLaVigencia: 1,
		HitosQueEstrenan: 2, HitosQueCesan: 1, HitosNoAlcanzados: 25,
		HitosYaCesados: 1, HitosQueEmpiezanDespues: 3, HitosConVigenciaIlegible: 1,

		// LAS LISTAS DE LOS CINCO DESCARTES QUE AHORA SE ABREN, y cada una
		// CUADRA con su contador de arriba: es lo que la seccion promete al
		// pintar la cifra como cabecera.
		//
		// `empiezan tarde` va con TRES hitos en DOS obligaciones a proposito, y
		// es el caso que hay que tener aqui: una obligacion escalonada aporta
		// tres al numero y una sola linea si la lista fuera por obligacion. Sin
		// este caso, la puerta que cuenta filas contra la cifra estaria verde
		// sobre datos donde cada obligacion tiene exactamente un hito, que es
		// justo donde el fallo no se ve.
		RelojesYaCesados: []pantalla.RelojDescartado{{
			Marco: "urn:demo:m4", Obligacion: "m4.o1", Titulo: "Registro derogado",
			Articulo: "art. 3", Hitos: []string{"anotacion"},
			Regla: "dejo de obligar antes de esta ventana, el 2024-05-05",
		}},
		RelojesQueEmpiezanDespues: []pantalla.RelojDescartado{{
			Marco: "urn:demo:m5", Obligacion: "m5.o1", Titulo: "Retencion documental",
			Articulo: "art. 20", Hitos: []string{"fin_de_la_retencion"},
			Regla: "empieza a obligar el 2027-12-11, despues del 2027-09-03",
		}, {
			Marco: "urn:demo:m5", Obligacion: "m5.o2", Titulo: "Notificacion escalonada",
			Articulo: "art. 21", Hitos: []string{"alerta_temprana", "informe_final"},
			Regla: "empieza a obligar el 2027-12-11, despues del 2027-09-03",
		}},
		RelojesConVigenciaIlegible: []pantalla.RelojDescartado{{
			Marco: "urn:demo:m6", Obligacion: "m6.o1", Titulo: "Punto sin vigencia legible",
			Articulo: "art. 1", Hitos: []string{"limite"},
			Regla: "vigencia ilegible: no-es-una-fecha",
		}},
		VencimientosMasAlla: []pantalla.VencimientoDescartado{{
			Vence: dia(2028, 2, 1), Marco: "urn:demo:m1", Obligacion: "m1.o4",
			Titulo: "Revision decenal", Articulo: "art. 30", Hito: "revision",
			Regla: "cadencia de 10 anos desde ultima_revision",
		}, {
			Vence: dia(2029, 2, 1), Marco: "urn:demo:m1", Obligacion: "m1.o4",
			Titulo: "Revision decenal", Articulo: "art. 30", Hito: "revision",
			Regla: "cadencia de 10 anos desde ultima_revision",
		}},
		// LA RAMA DEL DESCARGO, con su control positivo: sin una fila aqui, la
		// unica seccion que ensena fechas PASADAS no la recorre nadie y su
		// frase no la comprueba nada.
		VencimientosAnterioresALaVigencia: []pantalla.VencimientoDescartado{{
			Vence: dia(2023, 1, 15), Marco: "urn:demo:m1", Obligacion: "m1.o1",
			Titulo: "Revision anual del plan", Articulo: "art. 7.1", Hito: "revision",
			Regla: "la ocurrencia cae antes del 2024-11-07, que es cuando esta obligacion " +
				"empezo a obligar: ese dia la norma no obligaba",
		}},
	}
}

// LA PUERTA QUE MAS IMPORTA DE ESTA PANTALLA.
//
// Lo no constatado se presenta como DATO QUE FALTA, nunca como culpa. Un
// vencimiento pasado sin registro de cumplimiento no es un incumplimiento: es
// una ausencia de dato, y plazum no sabe distinguirlos. Acusar en falso es el
// unico error que un producto de cumplimiento no puede cometer ni una vez.
//
// CONTROL POSITIVO Y CONTROL NEGATIVO, los dos, porque uno solo no dice nada:
// con vencidas, la frase TIENE que salir; sin vencidas, la seccion entera no
// existe y la frase no se pide, o si no estaria pintando un descargo de algo que
// no se ha dicho.
func TestLoVencidoSaleConSuDescargoPegado(t *testing.T) {
	const laFrase = "calendario.pantalla.vencido.frase"

	s, esp := pantallaDePrueba(t, fuenteDoble{d: Derivado{
		Calendario: calendarioConVencidas(), Organizacion: "Acme SL"}, hay: true})
	codigo, cuerpo := pedir(t, s, BasePorDefecto+"/")
	if codigo != http.StatusOK {
		t.Fatalf("GET %s/ ha respondido %d", BasePorDefecto, codigo)
	}
	if !esp.pidio(laFrase) {
		t.Errorf("la pantalla pinta %d obligaciones vencidas y NO pide el descargo %q.\n"+
			"  Sin esa frase, la seccion se lee como una acusacion de incumplimiento, y lo "+
			"que plazum sabe es otra cosa: que en las respuestas del operador no consta que "+
			"se hiciera.", 1, laFrase)
	}
	if !strings.Contains(cuerpo, marca(laFrase)) {
		t.Errorf("el descargo se pide y no llega a la pagina. Cuerpo:\n%s", recorta(cuerpo, 900))
	}
	// Y LA FILA ESTA, o el descargo estaria descargando de nada.
	if !strings.Contains(cuerpo, "Revision anual del plan") {
		t.Error("la pantalla no pinta la obligacion vencida, asi que el descargo de arriba " +
			"acompana a una seccion vacia")
	}

	// CONTROL NEGATIVO: sin vencidas no se pide el descargo.
	cal := calendarioConVencidas()
	cal.Vencidas = nil
	s2, esp2 := pantallaDePrueba(t, fuenteDoble{d: Derivado{Calendario: cal}, hay: true})
	if _, cuerpo := pedir(t, s2, BasePorDefecto+"/"); strings.Contains(cuerpo, marca(laFrase)) {
		t.Error("sin ninguna obligacion vencida la pantalla sigue pintando el descargo de lo " +
			"vencido. Entonces la comprobacion de arriba no mide que salga con el dato: " +
			"saldria siempre")
	}
	if esp2.pidio(laFrase) {
		t.Error("sin vencidas, la pantalla pide igual la clave del descargo")
	}
}

// EL ESTADO VACIO EXISTE Y DICE EL SIGUIENTE PASO (puerta D11-b).
func TestSinAlcanceLaPantallaExisteYDiceComoSalirDeAhi(t *testing.T) {
	s, esp := pantallaDePrueba(t, nil)
	codigo, cuerpo := pedir(t, s, BasePorDefecto+"/")
	if codigo != http.StatusOK {
		t.Fatalf("sin fuente la pantalla contesta %d, y tiene que existir: una pantalla que "+
			"desaparece cuando no hay datos deja al operador sin saber que existia", codigo)
	}
	for _, clave := range []string{
		"calendario.pantalla.sin_alcance.titulo",
		"calendario.pantalla.sin_alcance.que_es",
		"calendario.pantalla.sin_alcance.paso",
	} {
		if !esp.pidio(clave) {
			t.Errorf("el estado vacio no pide %q", clave)
		}
	}
	// LA VUELTA AL CAMINO SIGUE AHI, que es justo el estado en el que mas falta
	// hace saber por donde se sigue.
	if !strings.Contains(cuerpo, `href="/camino/"`) {
		t.Errorf("el estado vacio no enlaza de vuelta al camino guiado:\n%s", recorta(cuerpo, 600))
	}
	// Y NO SE PINTA NI UNA CIFRA DE CALENDARIO: sin alcance no hay cuenta que dar.
	if strings.Contains(cuerpo, marca("calendario.pantalla.cuenta.titulo")) {
		t.Error("sin alcance la pantalla pinta la contabilidad del calendario, que sale de un " +
			"calendario que no existe")
	}
}

// UN FALLO AL DERIVAR NO SE CONVIERTE EN «no te vence nada».
//
// Son dos cosas distintas y solo una es inocua. Es el invariante 8 en la
// pantalla que mas se mira: si el alcance no se puede leer, quien mire tiene que
// saber que hay algo roto, no creerse que no le vence nada.
func TestUnFalloAlDerivarNoSePintaComoCalendarioVacio(t *testing.T) {
	s, _ := pantallaDePrueba(t, fuenteDoble{err: errDePrueba{}})
	codigo, cuerpo := pedir(t, s, BasePorDefecto+"/")
	if codigo != http.StatusInternalServerError {
		t.Errorf("con la fuente rota la pantalla contesta %d y tiene que contestar 500: un "+
			"200 con la pagina vacia se lee como «no tienes nada»", codigo)
	}
	if !strings.Contains(cuerpo, "el alcance no se entiende") {
		t.Errorf("la pantalla no dice que ha fallado:\n%s", recorta(cuerpo, 600))
	}
}

type errDePrueba struct{}

func (errDePrueba) Error() string { return "el alcance no se entiende" }

// EL FICHERO iCALENDAR SALE DEL MISMO DERIVADO, y sin alcance NO SALE VACIO.
//
// Un .ics con cero eventos importado en Outlook se lee como «no tengo nada que
// hacer», que es la afirmacion mas cara que este producto puede hacer sin datos.
// Por eso sin alcance se contesta 404 y no un fichero.
func TestElICSSaleDelMismoCalendarioYNoSaleVacioSinAlcance(t *testing.T) {
	s, _ := pantallaDePrueba(t, fuenteDoble{d: Derivado{
		Calendario: calendarioConVencidas(), Organizacion: "Acme SL"}, hay: true})
	codigo, cuerpo := pedir(t, s, BasePorDefecto+"/"+FicheroICS)
	if codigo != http.StatusOK {
		t.Fatalf("GET del .ics ha respondido %d", codigo)
	}
	if !strings.HasPrefix(cuerpo, "BEGIN:VCALENDAR") {
		t.Errorf("lo servido no es un iCalendar: empieza por %q", recorta(cuerpo, 60))
	}
	// El evento del mes tiene que estar: si no, el fichero sale del calendario
	// equivocado o de uno vacio.
	if !strings.Contains(cuerpo, "Informe trimestral") {
		t.Error("el .ics no lleva el vencimiento que la pantalla pinta: los dos tenian que " +
			"salir del mismo Derivado")
	}

	sinFuente, _ := pantallaDePrueba(t, nil)
	if codigo, _ := pedir(t, sinFuente, BasePorDefecto+"/"+FicheroICS); codigo != http.StatusNotFound {
		t.Errorf("sin alcance el .ics contesta %d. Un calendario vacio importado en Outlook "+
			"dice «no tienes nada que hacer», y eso plazum no lo sabe", codigo)
	}
}

// TODAS LAS RUTAS SON GET. Un calendario no se edita: se deriva.
func TestLaPantallaDelCalendarioSoloSirveGET(t *testing.T) {
	s, _ := pantallaDePrueba(t, nil)
	ps := s.Patrones()
	if len(ps) < 2 {
		t.Fatalf("la pantalla registra %d rutas y tiene al menos dos (la pagina y el .ics): "+
			"esta comprobacion estaria mirando el vacio", len(ps))
	}
	for _, p := range ps {
		if !strings.HasPrefix(p, "GET ") {
			t.Errorf("la ruta %q no es GET. Esta pantalla no muta nada, y una ruta mutante "+
				"aqui seria una segunda via para cambiar lo que el calendario dice", p)
		}
	}
	// Y de verdad: un POST a la pagina no se atiende.
	req := httptest.NewRequest(http.MethodPost, BasePorDefecto+"/", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Error("un POST a la pantalla del calendario ha contestado 200")
	}
}

// EL INVENTARIO DE CLAVES, EN LAS DOS DIRECCIONES.
//
// POR QUE HACE FALTA. `ClavesDeCatalogo()` es el contrato que esta superficie
// publica para que el catalogo se redacte. Si se queda corto, una clave sale en
// CRUDO en la pantalla de un cliente y ningun test lo ve; si le sobra, el
// catalogo lleva peso muerto que hay que traducir a cada idioma nuevo. Las dos
// direcciones se comprueban rindiendo la pantalla en TODOS sus estados con el
// catalogo espia y cruzando lo pedido contra lo publicado.
func TestElInventarioDeClavesCubreExactamenteLoQueLaPantallaPide(t *testing.T) {
	publicadas := map[string]bool{}
	for _, k := range ClavesDeCatalogo() {
		publicadas[k] = true
	}
	if len(publicadas) < 20 {
		t.Fatalf("la pantalla publica %d claves y son muchas menos de las que pinta: o el "+
			"contrato se ha vaciado, o este test mide otra cosa", len(publicadas))
	}

	// LOS DOS ESTADOS, para que ninguna clave se quede sin pedir. Con uno solo,
	// la mitad del inventario saldria como sobrante.
	esp := nuevoEspia()
	// UN CAMINO CON UN PASO SIN PANTALLA, y hace falta desde el 03-09-2026.
	//
	// Cuando calendario y escalado ganaron su pantalla, el camino canonico se
	// quedo SIN ningun paso sin pantalla, y la rama del armazon que pinta «por
	// terminal» dejo de tener quien la recorriera. Este test lo dijo con esas
	// palabras: publica ui.paso_por_terminal y ningun estado la pide.
	//
	// La respuesta NO es quitar la clave. Validar SIGUE aceptando un paso sin
	// pantalla que traiga su orden y la plantilla sigue teniendo su rama: la
	// capacidad esta viva y lo que le faltaba era una entrada. Es M47 aplicado
	// a un rotulo: una rama que ninguna entrada alcanza es una rama que no
	// existe.
	conPasoSinPantalla := append(append([]camino.Paso(nil), camino.Canonico()...),
		camino.Paso{ID: "sintetico", Titulo: "camino.paso.acta", Verbo: "camino.verbo.acta",
			Comando: "plazum algo --con-sus-banderas"})

	for _, e := range []struct {
		f     Fuente
		pasos []camino.Paso
	}{
		{nil, camino.Canonico()},
		{fuenteDoble{d: Derivado{Calendario: calendarioConVencidas(), Organizacion: "Acme SL",
			Supuesto: true}, hay: true}, camino.Canonico()},
		{fuenteDoble{d: Derivado{Calendario: pantalla.Calendario{
			Desde: dia(2026, 9, 3), Hasta: dia(2027, 9, 3)}}, hay: true}, camino.Canonico()},
		{fuenteDoble{d: Derivado{Calendario: calendarioConVencidas(), Organizacion: "Acme SL"},
			hay: true}, conPasoSinPantalla},
	} {
		s, err := NuevaPantalla(OpcionesPantalla{
			Fuente: e.f, Catalogo: esp, Base: BasePorDefecto, Estatico: "/estatico",
			CaminoRuta: camino.BasePorDefecto + "/", CaminoClave: camino.ClaveTitulo,
			Pasos: e.pasos,
		})
		if err != nil {
			t.Fatal(err)
		}
		if codigo, _ := pedir(t, s, BasePorDefecto+"/"); codigo != http.StatusOK {
			t.Fatalf("un estado de la pantalla ha contestado %d", codigo)
		}
	}
	// Y las dos claves de error, que no salen renderizando: se piden desde Go.
	pedidas := map[string]bool{
		"calendario.pantalla.error_render":    true,
		"calendario.pantalla.error_fuente":    true,
		"calendario.pantalla.ics_sin_alcance": true,
	}
	for _, k := range esp.claves() {
		pedidas[k] = true
	}
	// LOS ROTULOS DE LOS PASOS NO SON DE ESTA PANTALLA. Llegan como DATO desde
	// camino.Canonico() y los declara el camino, que es quien los escribe;
	// publicarlos aqui seria una segunda copia del camino dentro del contrato de
	// una superficie que solo lo pinta. Se eximen por su DECLARACION y no por un
	// prefijo: un prefijo eximiria tambien una clave del camino que nadie
	// hubiera declarado en ningun sitio, que es el caso que hay que cazar.
	delCamino := map[string]bool{}
	for _, k := range camino.ClavesDeCatalogo() {
		delCamino[k] = true
	}
	if len(delCamino) < 6 {
		t.Fatalf("el camino declara %d claves y son muchas menos de las que tiene: esta "+
			"exencion estaria eximiendo el vacio", len(delCamino))
	}
	for k := range pedidas {
		if publicadas[k] || delCamino[k] {
			continue
		}
		t.Errorf("la pantalla pide la clave %q y no la publica ni ClavesDeCatalogo() ni el "+
			"camino.\n"+
			"  Quien redacte el catalogo no se enterara de que hace falta, y saldra en "+
			"crudo en la pantalla de un cliente", k)
	}
	for k := range publicadas {
		if pedidas[k] {
			continue
		}
		// Los motivos de «sin fecha» y los doce meses los declara
		// nucleo/pantalla y solo se piden los que el dato de prueba alcanza.
		if strings.HasPrefix(k, "ui.mes.") || strings.HasPrefix(k, "ui.calendario.") {
			continue
		}

		t.Errorf("ClavesDeCatalogo() publica %q y la pantalla no la pide en ninguno de sus "+
			"estados. O sobra, o hay un estado que este test no recorre", k)
	}
}

// LAS DOS MITADES DEL ENLACE AL CAMINO, O NINGUNA.
//
// El valor cero (las dos vacias) es no pintar nada, que es el restrictivo. Lo
// que no se admite es medio enlace: una direccion sin rotulo pinta un enlace sin
// palabras y un rotulo sin direccion uno que no lleva a ningun sitio.
func TestMedioEnlaceAlCaminoSeRechazaAlConstruir(t *testing.T) {
	casos := []struct{ ruta, clave string }{
		{camino.BasePorDefecto + "/", ""},
		{"", camino.ClaveTitulo},
		{"//otro-sitio/", camino.ClaveTitulo},
	}
	for _, c := range casos {
		if _, err := NuevaPantalla(OpcionesPantalla{
			Catalogo: nuevoEspia(), CaminoRuta: c.ruta, CaminoClave: c.clave,
		}); err == nil {
			t.Errorf("se acepta CaminoRuta=%q CaminoClave=%q y tenia que rechazarse",
				c.ruta, c.clave)
		}
	}
	// CONTROL POSITIVO: el valor cero y el enlace entero se aceptan los dos.
	if _, err := NuevaPantalla(OpcionesPantalla{Catalogo: nuevoEspia()}); err != nil {
		t.Errorf("el valor cero del enlace (no pintar nada) se rechaza: %v", err)
	}
	if _, err := NuevaPantalla(OpcionesPantalla{
		Catalogo: nuevoEspia(), CaminoRuta: camino.BasePorDefecto + "/",
		CaminoClave: camino.ClaveTitulo,
	}); err != nil {
		t.Errorf("el enlace entero se rechaza: %v", err)
	}
}

func recorta(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// LA BARRA LATERAL ES EL CAMINO CANONICO, EN SU ORDEN Y MARCANDO ESTE PASO.
//
// Se compara contra camino.Canonico(), que es la fuente, y no contra una lista
// escrita aqui: un test que se mide contra su propia copia no comprueba nada.
// Y el paso MARCADO se comprueba aparte, porque una barra que sale entera y
// marca el paso equivocado le dice al operador que esta en un sitio en el que no
// esta, que es peor que no marcar ninguno.
func TestLaBarraLateralEsElCaminoEnteroYMarcaElCalendario(t *testing.T) {
	s, _ := pantallaDePrueba(t, fuenteDoble{d: Derivado{
		Calendario: calendarioConVencidas(), Organizacion: "Acme SL"}, hay: true})
	_, cuerpo := pedir(t, s, BasePorDefecto+"/")

	pasos := camino.Canonico()
	if len(pasos) < 6 {
		t.Fatalf("el camino declara %d pasos: este test recorreria casi nada", len(pasos))
	}
	antes := -1
	rotuloDelPaso := ""
	for _, p := range pasos {
		if p.ID == camino.IDDelCalendario {
			rotuloDelPaso = marca(p.Titulo)
		}
		pos := strings.Index(cuerpo, marca(p.Titulo))
		if pos < 0 {
			t.Errorf("el paso %q no sale en la barra lateral. Un camino de %d pasos del que "+
				"se ven menos parece completo y no lo esta", p.ID, len(pasos))
			continue
		}
		if pos < antes {
			t.Errorf("el paso %q sale fuera de orden en la barra lateral. El orden del camino "+
				"no es decorativo: cada paso consume lo que produce el anterior", p.ID)
		}
		antes = pos
	}
	if rotuloDelPaso == "" {
		t.Fatalf("el camino canonico ya no declara el paso %q, asi que esta pantalla estaria "+
			"marcando un paso que no existe", camino.IDDelCalendario)
	}

	// EL PASO MARCADO ES ESTE, y solo este.
	i := strings.Index(cuerpo, `aria-current="step"`)
	if i < 0 {
		t.Fatal("la barra lateral no marca ningun paso como actual: quien llegue no sabe " +
			"en que paso del camino esta")
	}
	if strings.Contains(cuerpo[i+1:], `aria-current="step"`) {
		t.Error("la barra lateral marca DOS pasos como actual")
	}
	// El rotulo del paso marcado va justo detras de la marca, dentro del mismo
	// enlace. Se busca en la cola y no en la pagina entera: el titulo del paso
	// tambien sale en el <h1>, y buscarlo suelto daria verde con la barra
	// marcando cualquier otro.
	cola := cuerpo[i:]
	if fin := strings.Index(cola, "</li>"); fin > 0 {
		cola = cola[:fin]
	}
	if !strings.Contains(cola, rotuloDelPaso) {
		t.Errorf("el paso marcado como actual no es el del calendario (%s). Una barra que "+
			"marca el paso equivocado le dice al operador que esta donde no esta.\n%s",
			rotuloDelPaso, recorta(cola, 400))
	}
}
