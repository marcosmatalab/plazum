package uar

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/superficies/camino"
)

// LA BARRA LATERAL EN LA REVISION DE ACCESOS: la cuarta de cuatro.
//
// POR QUE LLEGO LA ULTIMA, y merece constar. El frente que construyo el armazon
// tenia en su columna `superficies/uar/plantillas/` pero NO los ficheros Go de
// esta superficie, asi que pudo dejar lista la mitad de la plantilla y no pudo
// encender la otra. Se atuvo a la matriz y lo dijo, que es lo correcto: forzar
// un fichero de otra columna es como se pierde el trabajo ajeno. La cerro el
// integrador, que es de quien eran esos dos ficheros.
//
// LO QUE SE COMPRUEBA, y no es «que salga una barra»: que esta pantalla pinte
// EL MISMO camino que las demas, marcando SU paso, y que sin pasos siga siendo
// la de antes en vez de inventarse uno.

func conPasos(t *testing.T, pasos []camino.Paso) *Superficie {
	t.Helper()
	s, err := Nuevo(Opciones{
		Catalogo: cat(t), Base: "/uar", Estatico: "/estatico",
		Quien:       func(*http.Request) string { return "ciso" },
		CaminoRuta:  camino.BasePorDefecto + "/",
		CaminoClave: camino.ClaveTitulo,
		Pasos:       pasos,
	})
	if err != nil {
		t.Fatalf("construir la superficie: %v", err)
	}
	return s
}

func pagina(t *testing.T, s *Superficie) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/uar/", nil)
	req.Header.Set("Accept-Language", "es")
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	return rec.Body.String()
}

// TestLaUARPintaElCaminoEnteroYSeMarcaASiMisma es la mitad positiva.
func TestLaUARPintaElCaminoEnteroYSeMarcaASiMisma(t *testing.T) {
	pasos := camino.Canonico()
	cuerpo := pagina(t, conPasos(t, pasos))

	if !strings.Contains(cuerpo, `class="armazon"`) {
		t.Fatal("la pantalla de revision de accesos no trae el armazon compartido, asi que " +
			"quien entra aqui pierde la orientacion: no ve en que paso esta ni cuales quedan")
	}
	// EL CAMINO ENTERO, no solo los que son pantalla: un paso que desaparece de
	// la barra hace que el recorrido parezca mas corto de lo que es.
	for _, p := range pasos {
		rotulo := cat(t).Traducir("es", p.Titulo)
		if !strings.Contains(cuerpo, rotulo) {
			t.Errorf("la barra no nombra el paso %q (%q)", p.ID, rotulo)
		}
	}
	// Y SE MARCA EL SUYO, Y ESTO SE COMPRUEBA SOBRE EL ELEMENTO, NO SOBRE LA
	// PAGINA. La primera version solo miraba que el rotulo «estas aqui»
	// APARECIERA en algun sitio, y una mutacion que marcaba el paso del ACTA en
	// vez del de la UAR la dejaba VERDE: el rotulo estaba, sólo que en la fila
	// equivocada. Comprobar que algo aparece no es comprobar que aparece donde
	// tiene que aparecer.
	rotuloUAR := ""
	for _, p := range pasos {
		if p.ID == camino.IDDeLaUAR {
			rotuloUAR = cat(t).Traducir("es", p.Titulo)
		}
	}
	if rotuloUAR == "" {
		t.Fatal("el camino canonico no declara el paso de la revision de accesos: este test " +
			"estaria midiendo el vacio")
	}
	marcados := 0
	suyoMarcado := false
	for _, item := range strings.Split(cuerpo, `<li class="paso`)[1:] {
		if fin := strings.Index(item, "</li>"); fin >= 0 {
			item = item[:fin]
		}
		esActual := strings.HasPrefix(item, " actual")
		if esActual {
			marcados++
		}
		if strings.Contains(item, rotuloUAR) && esActual {
			suyoMarcado = true
		}
	}
	if marcados != 1 {
		t.Errorf("la barra marca %d pasos como actuales y tiene que marcar exactamente 1: "+
			"ninguno deja al operador sin saber donde esta, y dos le dicen que esta en dos "+
			"sitios", marcados)
	}
	if !suyoMarcado {
		t.Errorf("el paso marcado como actual NO es el de la revision de accesos (%q). La "+
			"barra esta diciendo que el operador esta en otra pantalla", rotuloUAR)
	}
}

// EL VALOR CERO ES NO PINTAR BARRA, y es el restrictivo.
//
// Sin pasos, esta pantalla sale como antes. Rellenarlo con el canonico cuando
// llega vacio convertiria un olvido de quien monta en una barra plausible que
// enlaza a donde el producto quizas no monta nada, y eso es peor que no tenerla:
// un enlace roto en la pantalla que lleva nombres de personas.
func TestSinPasosLaUARNoSeInventaUnCamino(t *testing.T) {
	for _, c := range []struct {
		nombre string
		pasos  []camino.Paso
	}{
		{"nil", nil},
		// LAS DOS FORMAS DE LA NADA, y la que se olvida es la segunda
		// (invariante 8): un slice vacio-presente no es lo mismo que nil, y
		// aqui las dos tienen que dar lo mismo.
		{"vacio presente", []camino.Paso{}},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			cuerpo := pagina(t, conPasos(t, c.pasos))
			if strings.Contains(cuerpo, `class="tira-camino"`) {
				t.Error("sin pasos se ha pintado la tira del camino: la pantalla se esta " +
					"inventando un recorrido que quien monta no ha dado")
			}
			// Y LA VUELTA AL CAMINO SIGUE AHI, que es la puerta D11-b: sin
			// barra, el enlace de vuelta es lo unico que impide el callejon.
			if !strings.Contains(cuerpo, camino.BasePorDefecto+"/") {
				t.Error("sin barra, la pantalla tampoco enlaza de vuelta al camino: eso si " +
					"es un callejon")
			}
		})
	}
}

// UN CAMINO QUE NO SE PUEDE RECORRER NO SE PINTA: se rechaza al construir.
//
// Lo juzga camino.Validar, el MISMO juez que la pantalla del camino. Dos jueces
// de la misma propiedad acaban discrepando, y el dia que discrepen esta barra y
// la pantalla del camino ensenarian caminos distintos.
func TestLaUARRechazaUnCaminoQueNoSeRecorre(t *testing.T) {
	roto := []camino.Paso{{ID: "x", Titulo: "t", Verbo: "v"}} // ni ruta ni comando
	if _, err := Nuevo(Opciones{
		Catalogo: cat(t), Base: "/uar", Estatico: "/estatico",
		Quien: func(*http.Request) string { return "ciso" },
		Pasos: roto,
	}); err == nil {
		t.Fatal("se ha construido la superficie con un paso que no lleva a ningun sitio")
	}
	// CONTROL POSITIVO: el canonico SI se acepta. Sin esta rama, un validador
	// que rechazara todo pasaria el caso de arriba por el motivo equivocado.
	if _, err := Nuevo(Opciones{
		Catalogo: cat(t), Base: "/uar", Estatico: "/estatico",
		Quien: func(*http.Request) string { return "ciso" },
		Pasos: camino.Canonico(),
	}); err != nil {
		t.Errorf("el camino canonico tampoco se acepta, asi que el validador dice que no a "+
			"todo: %v", err)
	}
}
