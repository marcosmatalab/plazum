package pantallas

import (
	"net/http"
	"strings"
	"testing"

	"plazum/nucleo/corpus"
	"plazum/nucleo/pantalla"
)

// La puerta del descargo de asesoramiento juridico en el pie.
//
// POR QUE HACE FALTA UNA PUERTA PROPIA Y NO BASTA CON LA QUE YA HAY.
// TestLasClavesDeCatalogoSonExactamenteLasQueLaInterfazPide compara la lista de
// claves con las que las plantillas piden, en las dos direcciones, asi que
// borrar el `{{t "ui.pie.no_asesoramiento"}}` de base.html y dejar la clave en
// claves.go se pone rojo. Lo que NO caza es el borrado COHERENTE, que es el que
// de verdad ocurre: alguien quita la linea de la plantilla, ve el test rojo,
// quita tambien la clave de la lista, y todo vuelve a verde con la pantalla ya
// sin descargo. Esta puerta mira el fondo y no la contabilidad: el pie tiene que
// estar, en TODAS las pantallas, tambien cuando la pantalla es un error o esta
// vacia.
//
// Por que en todas y no solo en la principal: el descargo protege de una lectura
// concreta, la de "la herramienta dice que me aplica el articulo 31", y esa
// lectura se hace mirando la tabla de Controles o el resultado del alcance, no
// la portada. Un pie que solo sale en la portada es un pie que no se lee.

// clavePie es la del descargo. Se escribe aqui a proposito, separada de
// clavesFijas: si esta puerta leyera la clave de la misma lista que vigila,
// renombrarla en la lista renombraria tambien lo que el test busca y la puerta
// seguiria verde sobre una pantalla distinta.
const clavePie = "ui.pie.no_asesoramiento"

func TestElDescargoSaleEnElPieDeTodasLasPantallas(t *testing.T) {
	esperado := rotulo("es", clavePie)

	casos := []struct {
		que    string
		ps     []*corpus.Paquete
		rutas  []string
		codigo int
	}{
		{"las seis pantallas con corpus", corpusDemo(), []string{
			"/alcance", "/hoy", "/controles", "/certificados", "/personas", "/estado",
		}, http.StatusOK},
		{"el alcance ya respondido, que es donde se lee un veredicto", corpusDemo(), []string{
			"/alcance?si=alfa.q.categoria&si=alfa.q.nombre&si=beta.q.riesgo",
			"/controles?f=aplica", "/certificados?si=alfa.q.categoria",
		}, http.StatusOK},
		{"sin corpus instalado", nil, []string{
			"/alcance", "/controles", "/certificados", "/hoy",
		}, http.StatusOK},
		{"la pagina de error", corpusDemo(), []string{"/no-existe"}, http.StatusNotFound},
	}

	for _, c := range casos {
		t.Run(c.que, func(t *testing.T) {
			s, _ := superficie(t, c.ps)
			for _, ruta := range c.rutas {
				w, cuerpo := pedir(t, s, ruta)
				if w.Code != c.codigo {
					t.Fatalf("GET %s dio %d y esperaba %d", ruta, w.Code, c.codigo)
				}
				if !strings.Contains(cuerpo, esperado) {
					t.Errorf("GET %s se sirve SIN el descargo de asesoramiento juridico "+
						"(falta el rotulo de %q).\n"+
						"  Esta pantalla dice que obligaciones aplican y con que cita, que es "+
						"justo la lectura de la que el descargo protege.\n"+
						"  Arreglo: el pie lo pinta la plantilla base "+
						"(superficies/pantallas/plantillas/base.html) y el texto lo pone el "+
						"catalogo, en es y en en.", ruta, clavePie)
					continue
				}
				// Y en el PIE, no suelto en medio: la ultima seccion de la
				// pagina. Se comprueba por posicion contra el cierre del
				// cuerpo, que es lo unico que hay siempre.
				iPie := strings.LastIndex(cuerpo, esperado)
				iPrincipal := strings.Index(cuerpo, "id=\"principal\"")
				if iPrincipal >= 0 && iPie < iPrincipal {
					t.Errorf("GET %s lleva el descargo ANTES del contenido principal "+
						"(posiciones %d y %d). En el pie se lee al terminar de mirar el "+
						"veredicto; encima, se lee antes de saber de que va", ruta, iPie, iPrincipal)
				}
			}
		})
	}
}

// Control negativo de la puerta de arriba: se demuestra que el detector muerde.
//
// Sin esto, `strings.Contains` de un rotulo que el doble de catalogo genera
// SIEMPRE podria estar dando verde por el formato del doble y no por la
// plantilla. Aqui se sirve la misma pagina con un catalogo que devuelve otra
// cosa para esa clave y se comprueba que la comprobacion de arriba fallaria.
func TestControlNegativoElDetectorDelPieMuerde(t *testing.T) {
	cat := nuevoCatalogo()
	cat.textos = map[string]string{clavePie: "un pie que no dice nada"}
	s, _ := superficie(t, corpusDemo(), func(o *Opciones) { o.Catalogo = cat })
	_, cuerpo := pedir(t, s, "/alcance")
	if strings.Contains(cuerpo, rotulo("es", clavePie)) {
		t.Fatal("el detector encuentra el rotulo del descargo en una pagina donde el catalogo " +
			"devuelve OTRO texto para esa clave. Entonces no esta mirando la pagina: esta " +
			"encontrando la cadena en algun sitio que no es el pie, y la puerta de arriba " +
			"daria verde con el descargo borrado")
	}
	if !strings.Contains(cuerpo, "un pie que no dice nada") {
		t.Fatal("la pagina no lleva NI el descargo ni el texto sustituto, asi que este control " +
			"negativo no ha llegado a pintar el pie y no demuestra nada")
	}
}

// El descargo no se le puede escapar a nadie que anada una pantalla.
//
// nucleo/pantalla.Derivar es quien decide cuantas pantallas hay. Si manana
// aparece una septima, esta puerta la sirve y le exige el pie sin que nadie
// tenga que acordarse de anadirla a una lista.
func TestUnaPantallaNuevaTambienLlevaElDescargo(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	derivadas := pantalla.Derivar(nil)
	if len(derivadas) < 6 {
		t.Fatalf("nucleo/pantalla deriva %d pantallas y son al menos 6. Si el modelo se ha "+
			"vaciado, este test estaria recorriendo la nada y dando verde", len(derivadas))
	}
	for _, p := range derivadas {
		w, cuerpo := pedir(t, s, "/"+string(p.ID))
		if w.Code != http.StatusOK {
			t.Errorf("la pantalla %s del modelo no se sirve (%d): una pantalla que el nucleo "+
				"deriva y la superficie no atiende es un hueco, con descargo o sin el",
				p.ID, w.Code)
			continue
		}
		if !strings.Contains(cuerpo, rotulo("es", clavePie)) {
			t.Errorf("la pantalla %s se sirve sin el descargo del pie", p.ID)
		}
	}
}
