package plazum

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/pantalla"
	"github.com/marcosmatalab/plazum/nucleo/ventana"
	calendarioWeb "github.com/marcosmatalab/plazum/superficies/calendario"
	"github.com/marcosmatalab/plazum/superficies/camino"
	"github.com/marcosmatalab/plazum/superficies/pantallas"
)

// TODA DIRECTIVA QUE APORTE OBLIGACIONES AVISA DE QUE ES UNA DIRECTIVA, DONDE
// SALEN SUS OBLIGACIONES.
//
// # Que se estaba diciendo
//
// Una directiva de la Union no obliga a nadie por si misma. Obliga la norma
// nacional que la transpone, y mientras no exista, no obliga NADA. Hasta hoy la
// tabla de controles le decia «Te aplica» a un espanol sobre NIS2, que no esta
// transpuesta: nueve de sus filas llevaban el aviso metido a mano dentro del
// campo `cita`, en cuatro redacciones distintas (seis nombraban el RD 43/2021 y
// cuatro no), y el resto no lo llevaba.
//
// El commit 7e4850d convirtio esa prosa repetida en el campo `transposicion`,
// con linter en las dos direcciones. Y ahi se quedo: el campo no lo pintaba
// nadie. O sea que el aviso paso de estar mal dicho diez veces a no estar dicho
// ninguna, que es peor, y ninguna puerta lo noto porque las puertas miraban el
// LINTER y no la PANTALLA. Es el modo de fallo que la casilla de las familias de
// guardas dice haber cerrado: el comportamiento estaba y la enumeracion no.
//
// # Por que ENUMERA y no comprueba uno
//
// Porque comprobar uno es lo que ya se hacia. Esta puerta recorre TODOS los
// paquetes del arbol que declaran transposicion, los reparte en dos cubos y
// exige de cada cubo lo que le corresponde:
//
//	aporta obligaciones   su aviso tiene que salir en /controles y en la ficha
//	                      del calendario. Es el caso de nis2-ue.
//	aporta cero           no puede salir en ningun sitio porque no tiene ni una
//	                      fila, y eso NO es un fallo: es un esqueleto. Se cuenta
//	                      y se dice, en vez de callarse, que es como un cubo
//	                      vacio se convierte en la excusa de que la puerta no
//	                      vigile nada.
//
// # El universo es pequeno y se dice en voz alta
//
// Hoy son TRES paquetes con bloque de transposicion en todo el arbol, y solo UNO
// aporta obligaciones. Una puerta que vigila un caso es «comprobar uno» con
// forma de censo, asi que el censo va por IGUALDAD EXACTA en los dos cubos: el
// dia que entre una directiva nueva, o que csrd o psd2 dejen de estar vacios,
// esto se pone rojo y obliga a mirar si su aviso sale.
//
// # Como se sabe que un paquete es una directiva, sin escribir ningun urn
//
// Por `p.Transposicion != nil`, y no es un atajo: el linter de corpus garantiza
// las dos direcciones (ErrDirectivaSinTransposicion y
// ErrTransposicionEnLoQueNoEsDirectiva), asi que «tiene bloque» y «es directiva»
// son el mismo conjunto o el paquete no carga. Escribir aqui `urn:eu:dir` seria
// ademas un identificador de norma cableado, que rompe el invariante 2.
//
// # LO QUE ESTA PUERTA NO MIRA, dicho al escribirla
//
// Cuenta SITIOS DE PINTURA. No dice nada de si el aviso es CIERTO: `consta`,
// `comprobado` y `como` los escribe una persona mirando el BOE un dia concreto y
// nada del arbol vuelve a mirar. `nis2-ue` dice «comprobado el 2026-08-26». La
// frescura de ese dato no tiene puerta, y es lo que hace que «no consta» siga
// siendo verdad. Queda contado en docs/pendientes.md.
//
// Y no cubre los otros sitios donde sale un marco: el `.ics` que se descarga a
// Outlook, la pantalla Hoy, la pagina de una cifra y el terminal. Estan contados
// en docs/pendientes.md con su cardinal.
func TestTodaDirectivaAvisaDondeSalenSusObligaciones(t *testing.T) {
	conObligaciones, sinObligaciones := directivasDelArbol(t)

	// EL CENSO, por igualdad exacta en los dos cubos.
	const (
		DirectivasQueAportanObligaciones = 1
		DirectivasVacias                 = 2
	)
	if len(conObligaciones) != DirectivasQueAportanObligaciones {
		t.Errorf("hay %d directivas que aportan obligaciones (%v) y el censo dice %d.\n"+
			"  Si ha SUBIDO, hay una directiva nueva en el escaparate y su aviso tiene que "+
			"salir: sube el numero DESPUES de comprobar que sale.\n"+
			"  Si ha BAJADO, un marco ha dejado de aportar obligaciones y hay que saber por "+
			"que.", len(conObligaciones), conObligaciones, DirectivasQueAportanObligaciones)
	}
	if len(sinObligaciones) != DirectivasVacias {
		t.Errorf("hay %d directivas sin ni una obligacion (%v) y el censo dice %d.\n"+
			"  Un esqueleto no puede ensenar su aviso porque no tiene donde; lo que no puede "+
			"pasar es que deje de estar vacio sin que nadie mire si su aviso sale.",
			len(sinObligaciones), sinObligaciones, DirectivasVacias)
	}
	t.Logf("directivas en el arbol: %d con obligaciones %v, %d vacias %v",
		len(conObligaciones), conObligaciones, len(sinObligaciones), sinObligaciones)

	// Y AHORA LO QUE IMPORTA: que el aviso salga donde salen sus obligaciones.
	ps := corpusPublicado(t)
	tabla := servidorDeControles(t, ps)
	calendario := paginaDe(t, servidorDelCalendario(t, ps), calendarioWeb.BasePorDefecto+"/")

	for _, urn := range conObligaciones {
		aviso := marcaDelAviso(t, ps, urn)

		// LA TABLA SE PAGINA, asi que no vale pedir una pagina y mirar: desde
		// D-13 se pintan 25 filas de 563, y las de este marco pueden estar en la
		// septima. Se busca LA PAGINA DONDE SALEN SUS FILAS y se exige el aviso
		// EN ESA, que es la afirmacion que importa: el aviso al lado del dato, y
		// no en algun sitio del sitio.
		pagina, n := paginaConLasFilasDe(t, tabla, urn)
		if n == 0 {
			t.Errorf("%s aporta obligaciones y ninguna pagina de /controles trae una fila "+
				"suya: o el filtro las esconde o la paginacion no llega", urn)
		} else if !strings.Contains(pagina, aviso) {
			t.Errorf("%s aporta obligaciones y su aviso de directiva NO sale en la pagina de "+
				"la tabla donde salen sus %d filas.\n"+
				"  Ahi es donde el producto dice «Te aplica», y una directiva no transpuesta "+
				"no le aplica todavia a nadie: el aviso tiene que leerse en esa misma pagina, "+
				"no en otra.", urn, n)
		}

		if !strings.Contains(calendario, aviso) {
			t.Errorf("%s aporta obligaciones y su aviso de directiva NO sale en el "+
				"calendario, que es donde salen sus fechas.", urn)
		}
	}

	// EL CONTROL POSITIVO DE LA MEDIDA. Si el calendario saliera vacio, lo de
	// arriba pasaria sin haber pintado nada: un cuerpo vacio no contiene el
	// aviso, pero tampoco contiene una ficha.
	if !strings.Contains(calendario, `class="fuente"`) {
		t.Error("el calendario no trae ni una ficha, asi que esta puerta estaria " +
			"comprobando que un aviso no falta en una pagina que no ensena nada")
	}
}

// paginaConLasFilasDe devuelve la primera pagina de la tabla que trae filas de
// ese marco, y cuantas trae.
//
// Recorre las paginas en vez de pedir una: la tabla pagina de 25 en 25 sobre 563
// filas, asi que mirar solo la primera seria comprobar el aviso de los paquetes
// que van primero por orden de URN y de ninguno mas.
func paginaConLasFilasDe(t *testing.T, h http.Handler, urn string) (string, int) {
	t.Helper()
	marca := `<td class="paquete">` + urn
	for p := 1; p <= 40; p++ {
		cuerpo := paginaDe(t, h, fmt.Sprintf("/controles?%s=todos&%s=%d",
			pantallas.ParamFiltro, pantallas.ParamPagina, p))
		if n := strings.Count(cuerpo, marca); n > 0 {
			return cuerpo, n
		}
		if !strings.Contains(cuerpo, `<tr class="e-`) {
			break // se acabaron las filas
		}
	}
	return "", 0
}

// TestElAvisoDeDirectivaNoSaleDondeNoHayDirectiva es el control negativo, y es
// la mitad que impide que la puerta de arriba se cumpla pintando el aviso en
// todas partes.
//
// Un aviso que saliera en las 563 filas seria a la vez inutil y falso: diria que
// el ENS es una directiva.
func TestElAvisoDeDirectivaNoSaleDondeNoHayDirectiva(t *testing.T) {
	ps := corpusPublicado(t)
	conAviso := map[string]bool{}
	for _, a := range pantalla.Derivar(ps) {
		for _, av := range a.Avisos {
			conAviso[av.Marco] = true
		}
	}
	if len(conAviso) == 0 {
		t.Fatal("ningun marco trae aviso, asi que este control negativo no separa nada")
	}
	sin := 0
	for _, p := range ps {
		if p.Transposicion == nil {
			if conAviso[p.URN] {
				t.Errorf("%s no declara transposicion y le sale aviso de directiva", p.URN)
			}
			sin++
		}
	}
	if sin == 0 {
		t.Fatal("todos los paquetes publicados declaran transposicion: no hay contra que " +
			"contrastar")
	}
	t.Logf("marcos con aviso: %d; sin aviso: %d", len(conAviso), sin)
}

// directivasDelArbol reparte los paquetes con bloque de transposicion en los dos
// cubos, mirando los TRES directorios del arbol.
func directivasDelArbol(t *testing.T) (conObligaciones, sinObligaciones []string) {
	t.Helper()
	for _, raiz := range []string{"paquetes", DirDeEsqueletos, DirDelDemo} {
		ps, err := corpus.Cargar(raiz)
		if err != nil {
			t.Fatalf("no puedo cargar %s/: %v", raiz, err)
		}
		for _, p := range ps {
			if p.Transposicion == nil {
				continue
			}
			if len(p.Obligaciones) > 0 {
				conObligaciones = append(conObligaciones, p.URN)
			} else {
				sinObligaciones = append(sinObligaciones, p.URN)
			}
		}
	}
	sort.Strings(conObligaciones)
	sort.Strings(sinObligaciones)
	return conObligaciones, sinObligaciones
}

// marcaDelAviso compone lo que la pagina tiene que traer para ESE marco.
//
// Sale del catalogo real por su CLAVE y de los DATOS del paquete, no de un
// literal: un literal se queda viejo el dia que alguien mejore la redaccion, y
// entonces esta puerta buscaria algo que no existe, o sea daria verde sobre
// cualquier cosa.
func marcaDelAviso(t *testing.T, ps []*corpus.Paquete, urn string) string {
	t.Helper()
	for _, p := range ps {
		if p.URN != urn || p.Transposicion == nil || len(p.Transposicion.Estado) == 0 {
			continue
		}
		e := p.Transposicion.Estado[0]
		clave, datos := pantalla.ClaveDirectivaNoConsta, []any{e.Pais, e.Comprobado}
		if e.Consta {
			clave, datos = pantalla.ClaveDirectivaConsta, []any{e.Pais}
		}
		return catalogoReal(t).Traducir("es", clave, datos...)
	}
	t.Fatalf("%s no trae bloque de transposicion con estado", urn)
	return ""
}

func paginaDe(t *testing.T, h http.Handler, destino string) string {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, destino, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("%s devuelve %d y no 200", destino, w.Code)
	}
	return w.Body.String()
}

// ---------------------------------------------------------------------------
// El arnes: las dos superficies montadas con el corpus PUBLICADO.
//
// Con el corpus real y no con uno sintetico a proposito: lo que esta puerta
// afirma es sobre el producto que se instala, y un corpus de mentira con una
// directiva de mentira demostraria que el cable funciona, no que el aviso sale.
// ---------------------------------------------------------------------------

func corpusPublicado(t *testing.T) []*corpus.Paquete {
	t.Helper()
	ps, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatalf("el corpus publicado no carga: %v", err)
	}
	if len(ps) < MinimoDeMarcos {
		t.Fatalf("solo %d paquetes: esta puerta estaria midiendo medio corpus", len(ps))
	}
	return ps
}

func servidorDeControles(t *testing.T, ps []*corpus.Paquete) http.Handler {
	t.Helper()
	s, err := pantallas.Nuevo(pantallas.Opciones{
		Paquetes: ps, Catalogo: catalogoReal(t),
		CaminoRuta: camino.BasePorDefecto + "/", CaminoClave: camino.ClaveTitulo,
		Pasos: camino.Canonico(),
	})
	if err != nil {
		t.Fatalf("montando las pantallas: %v", err)
	}
	return s
}

// fuenteDelCalendario sirve el calendario derivado del corpus publicado.
//
// El instante entra como dato (invariante 1) y es el mismo en cada ejecucion:
// una puerta cuyo veredicto dependa del dia en que se ejecute es una bomba con
// la mecha encendida.
type fuenteDelCalendario struct{ cal pantalla.Calendario }

func (f fuenteDelCalendario) Actual() (calendarioWeb.Derivado, bool, error) {
	return calendarioWeb.Derivado{Calendario: f.cal, Organizacion: "acme"}, true, nil
}

func servidorDelCalendario(t *testing.T, ps []*corpus.Paquete) http.Handler {
	t.Helper()
	ahora := time.Date(2026, 9, 10, 9, 0, 0, 0, time.UTC)
	cal := pantalla.Derivar12Meses(ps, pantalla.TodoAplica, ventana.Hechos{}, ahora)
	s, err := calendarioWeb.NuevaPantalla(calendarioWeb.OpcionesPantalla{
		Fuente: fuenteDelCalendario{cal: cal}, Catalogo: catalogoReal(t),
		Base:       calendarioWeb.BasePorDefecto,
		CaminoRuta: camino.BasePorDefecto + "/", CaminoClave: camino.ClaveTitulo,
		Pasos: camino.Canonico(),
	})
	if err != nil {
		t.Fatalf("montando el calendario: %v", err)
	}
	return s
}
