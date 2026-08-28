package pantallas

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/pantalla"
)

// La puerta de la ATRIBUCION del corpus en el pie.
//
// QUE OBLIGACION SOSTIENE. La Decision 2011/833/UE autoriza reutilizar el DOUE
// **con atribucion**, y las condiciones de reutilizacion del BOE exigen citar la
// fuente. Las dos son obligaciones hacia quien USA el contenido, no hacia quien
// lee el repositorio: un docs/LICENCIAS.md se lo cuenta al que abre el codigo
// fuente y no se lo cuenta al CISO que tiene la pantalla delante. Asi que el
// aviso viaja dentro del paquete y sale en pantalla, y esto es lo que se pone
// rojo cuando deja de salir.
//
// POR QUE NO BASTA CON LAS PUERTAS QUE YA HAY. El caso dorado de nucleo/pantalla
// compara el MODELO byte a byte, asi que caza que la derivacion deje de traer
// las fuentes. Lo que no caza es que el modelo las traiga y la plantilla no las
// pinte, que es exactamente el borrado que ocurre en la practica: alguien limpia
// el pie porque le sobra ruido y el modelo sigue igual de verde.
//
// POR QUE EN TODAS LAS PANTALLAS. El contenido del corpus se usa en Alcance (el
// texto y la cita de cada pregunta), en Controles y en Certificados. Un aviso
// que solo saliera en una de ellas dejaria las otras dos sin atribuir. Es el
// mismo razonamiento del descargo de asesoramiento juridico, y por comodidad de
// quien lea esto dentro de un ano: ver descargo_test.go.

// TestLaAtribucionDelCorpusSaleEnElPieDeTodasLasPantallas.
func TestLaAtribucionDelCorpusSaleEnElPieDeTodasLasPantallas(t *testing.T) {
	ps := corpusDemo()
	if len(ps) < 2 {
		t.Fatal("hacen falta dos paquetes para que este test distinga que se atribuyen los " +
			"dos y no solo el primero")
	}
	// Los avisos de los dos paquetes tienen que ser distintos, o pintar solo
	// uno pasaria el test dos veces.
	if ps[0].Atribucion == ps[1].Atribucion {
		t.Fatal("los dos paquetes del corpus de prueba traen el mismo aviso, asi que este " +
			"test no distingue si se pierde uno")
	}

	casos := []struct {
		que    string
		rutas  []string
		codigo int
	}{
		{"las seis pantallas", []string{
			"/alcance", "/hoy", "/controles", "/certificados", "/personas", "/estado",
		}, http.StatusOK},
		{"el alcance respondido y las tablas filtradas", []string{
			"/alcance?si=alfa.q.categoria&si=alfa.q.nombre&si=beta.q.riesgo",
			"/controles?f=aplica", "/certificados?si=alfa.q.categoria",
		}, http.StatusOK},
		{"la pagina de error, que no es ninguna pantalla", []string{"/no-existe"},
			http.StatusNotFound},
	}

	s, _ := superficie(t, ps)
	for _, c := range casos {
		t.Run(c.que, func(t *testing.T) {
			for _, ruta := range c.rutas {
				w, cuerpo := pedir(t, s, ruta)
				if w.Code != c.codigo {
					t.Fatalf("GET %s dio %d y esperaba %d", ruta, w.Code, c.codigo)
				}
				for _, p := range ps {
					if !strings.Contains(cuerpo, p.Atribucion) {
						t.Errorf("GET %s se sirve SIN la atribucion de %s.\n"+
							"  El producto ensena contenido de ese paquete y la condicion "+
							"de reutilizacion es atribuirlo a quien lo usa, no en un fichero "+
							"del repositorio.\n"+
							"  Arreglo: el pie lo pinta superficies/pantallas/plantillas/"+
							"base.html a partir de Marco.Fuentes, que llena nucleo/pantalla.",
							ruta, p.URN)
						continue
					}
					// La fuente se cita con el enlace DERIVADO del identificador
					// estable, no con un campo copiado del fichero de datos. Si
					// alguien vuelve a guardar la URL como dato, esto sigue en
					// verde pero el paquete deja de cargar: la puerta de eso es
					// el linter (ErrFuenteHeredada).
					if !strings.Contains(cuerpo, p.Enlace()) {
						t.Errorf("GET %s atribuye %s sin citar su fuente (%q), que es lo que "+
							"las condiciones del BOE y la Decision 2011/833/UE piden citar",
							ruta, p.URN, p.Enlace())
					}
					// Y el identificador estable tiene que estar en la pagina,
					// venga dentro de la direccion (un ELI) o en su propio
					// hueco (ISO, PCI DSS). Se comprueba sobre el cuerpo entero
					// y no sobre el hueco a proposito: lo que se le debe al
					// lector es poder identificar la norma, no un span concreto.
					if !strings.Contains(cuerpo, p.Identificador.Valor) {
						t.Errorf("GET %s cita %s sin su identificador estable (%q): si el "+
							"enlace derivado deja de resolver, no queda en pantalla nada "+
							"que diga que norma es", ruta, p.URN, p.Identificador.Valor)
					}
				}
				// Y en el PIE, no suelto en medio de la pagina: se comprueba
				// por posicion contra el contenido principal, igual que el
				// descargo. Solo si el aviso esta: si falta, ya lo ha dicho
				// el bucle de arriba, y un segundo error sobre la posicion
				// -1 solo despista a quien lea la salida.
				iAviso := strings.LastIndex(cuerpo, ps[0].Atribucion)
				iPrincipal := strings.Index(cuerpo, "id=\"principal\"")
				if iAviso >= 0 && iPrincipal >= 0 && iAviso < iPrincipal {
					t.Errorf("GET %s lleva la atribucion ANTES del contenido principal "+
						"(posiciones %d y %d): es un aviso permanente de pie, no una "+
						"cabecera", ruta, iAviso, iPrincipal)
				}
			}
		})
	}
}

// Control negativo de la puerta de arriba: se demuestra que el detector muerde.
//
// Sin esto, `strings.Contains` podria estar dando verde por encontrar la cadena
// en cualquier otro sitio de la pagina. Aqui se sirve el MISMO corpus con otro
// aviso y se comprueba que el detector deja de encontrar el anterior y encuentra
// el nuevo: o sea que esta mirando lo que el paquete declara.
func TestControlNegativoElDetectorDeLaAtribucionMuerde(t *testing.T) {
	ps := corpusDemo()
	viejo := ps[0].Atribucion
	nuevo := "OTRO AVISO DE DERECHOS QUE NO SE PARECE AL ANTERIOR"
	ps[0].Atribucion = nuevo

	s, _ := superficie(t, ps)
	_, cuerpo := pedir(t, s, "/alcance")
	if strings.Contains(cuerpo, viejo) {
		t.Fatal("el detector encuentra el aviso VIEJO en una pagina servida con otro aviso. " +
			"Entonces no esta mirando lo que declara el paquete, y la puerta de arriba " +
			"daria verde con el pie borrado")
	}
	if !strings.Contains(cuerpo, nuevo) {
		t.Fatal("la pagina no lleva NI el aviso viejo NI el nuevo, asi que este control " +
			"negativo no ha llegado a pintar el pie y no demuestra nada")
	}
}

// Una pantalla que alguien anada manana tambien atribuye.
//
// nucleo/pantalla.Derivar decide cuantas pantallas hay. La septima se sirve por
// su ruta sin que nadie toque el enrutado, asi que tambien tiene que llevar el
// aviso sin que nadie se acuerde de anadirla a una lista.
func TestUnaPantallaNuevaTambienAtribuyeElCorpus(t *testing.T) {
	ps := corpusDemo()
	s, _ := superficie(t, ps)
	derivadas := pantalla.Derivar(nil)
	if len(derivadas) < 6 {
		t.Fatalf("nucleo/pantalla deriva %d pantallas y son al menos 6: si el modelo se ha "+
			"vaciado, este test recorreria la nada", len(derivadas))
	}
	for _, p := range derivadas {
		w, cuerpo := pedir(t, s, "/"+string(p.ID))
		if w.Code != http.StatusOK {
			t.Errorf("la pantalla %s del modelo no se sirve (%d)", p.ID, w.Code)
			continue
		}
		if !strings.Contains(cuerpo, ps[0].Atribucion) {
			t.Errorf("la pantalla %s se sirve sin la atribucion del corpus", p.ID)
		}
	}
}

// Sin corpus instalado no hay a quien atribuir, y la pagina no puede inventarse
// un aviso ni dejar una lista vacia colgando. Es el control negativo del bloque
// entero: si la plantilla pintara algo fijo, aqui se veria.
func TestSinCorpusElPieNoInventaNingunAviso(t *testing.T) {
	s, _ := superficie(t, nil)
	_, cuerpo := pedir(t, s, "/alcance")
	if strings.Contains(cuerpo, `class="fuentes"`) {
		t.Error("sin corpus instalado la pagina pinta la lista de fuentes vacia. Una " +
			"seccion de avisos legales sin avisos se lee como un fallo")
	}
	// Y el descargo, que no depende del corpus, sigue estando: asi se sabe que
	// la pagina se ha pintado de verdad y este test no esta mirando un 500.
	if !strings.Contains(cuerpo, rotulo("es", clavePie)) {
		t.Fatal("la pagina sin corpus no trae ni el descargo, asi que no se ha pintado y " +
			"este test no comprueba nada")
	}
}

// El aviso NO pasa por el catalogo, y eso es una decision legal, no de estilo.
//
// Un aviso de derechos parafraseado por la interfaz deja de ser el aviso, y
// traducirlo cae en la misma regla que traducir el texto transcrito del BOE:
// crea obra derivada y se sale de la estratificacion de licencias. Aqui se sirve
// la misma pagina en dos idiomas y se exige que el aviso salga IDENTICO.
func TestElAvisoDeAtribucionNoSeTraduce(t *testing.T) {
	ps := corpusDemo()
	s, _ := superficie(t, ps)
	_, es := pedirConIdioma(t, s, "/alcance", "es")
	_, en := pedirConIdioma(t, s, "/alcance", "en")
	if es == en {
		t.Fatal("las dos paginas son identicas, asi que el idioma no ha cambiado nada y " +
			"este test no distingue si el aviso se tradujo")
	}
	for _, p := range ps {
		if !strings.Contains(es, p.Atribucion) || !strings.Contains(en, p.Atribucion) {
			t.Errorf("el aviso de %s no sale igual en los dos idiomas: el catalogo lo esta "+
				"tocando, y un aviso de derechos traducido ya no es el aviso", p.URN)
		}
	}
}

// pedirConIdioma sirve una ruta declarando un idioma en Accept-Language.
func pedirConIdioma(t *testing.T, s *Superficie, ruta, idioma string) (int, string) {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, ruta, nil)
	r.Header.Set("Accept-Language", idioma)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w.Code, w.Body.String()
}

// Y el mismo corpus real: los paquetes publicados traen su aviso, asi que la
// pantalla tiene algo que ensenar. Sin este caso, todo lo de arriba se sostiene
// sobre paquetes sinteticos que nunca pasaron el linter.
func TestElCorpusSinteticoDeEsteFicheroDeclaraLoQueElLinterExige(t *testing.T) {
	for _, p := range corpusDemo() {
		if p.LicenciaFuente == "" || p.Atribucion == "" {
			t.Fatalf("%s no declara licencia_fuente o atribucion, asi que el corpus de "+
				"prueba de esta superficie ya no se parece a uno que cargue", p.URN)
		}
		if errs := (&corpus.Paquete{
			URN: p.URN, Version: p.Version, Clase: p.Clase, Identificador: p.Identificador,
			LicenciaFuente: p.LicenciaFuente, Atribucion: p.Atribucion,
			Vigencia: p.Vigencia,
		}).Validar(); len(errs) > 0 {
			for _, e := range errs {
				if strings.Contains(e.Error(), "licencia_fuente") ||
					strings.Contains(e.Error(), "atribucion") {
					t.Errorf("%s: %v", p.URN, e)
				}
			}
		}
	}
}

// EL IDENTIFICADOR ESTABLE, EN PANTALLA, CUANDO EL ENLACE NO LO LLEVA.
//
// Que propiedad se sostiene: el pie tiene que dejar al lector identificar la
// norma aunque el enlace derivado deje de resolver. En un ELI eso ya lo hace la
// propia direccion (es el prefijo del editor mas el identificador entero); en
// los dos esquemas donde la direccion NO identifica nada por si sola, el
// identificador tiene su propio hueco.
//
// Se prueban LAS DOS DIRECCIONES, y esa es la mitad que importa: sin la segunda,
// una plantilla que pintara el identificador siempre pasaria igual, y el pie de
// todas las paginas repetiria la misma cadena dos veces por cada paquete.
//
// El paquete es SINTETICO (version 9.9, que no existe): lo que se prueba es la
// forma del esquema, no una norma.
func TestElIdentificadorSaleEnPantallaSoloCuandoElEnlaceNoLoLleva(t *testing.T) {
	conIdentificador := func(urn string, id corpus.Identificador) *corpus.Paquete {
		return &corpus.Paquete{
			URN: urn, Version: "1", Clase: corpus.Propio,
			LicenciaFuente: corpus.DelProyecto,
			Atribucion:     "Aviso de derechos de " + urn + ".",
			Identificador:  id,
			Obligaciones: []corpus.Obligacion{{
				ID: urn + ".o", Articulo: "1", Cita: "demo art. 1", ClaseE2E: "documental",
			}},
		}
	}
	// El enlace de este NO lleva la version dentro: PCI SSC sirve todas las
	// versiones desde una sola biblioteca.
	suelto := conIdentificador("urn:demo:suelto",
		corpus.Identificador{Tipo: corpus.VersionPCIDSS, Valor: "9.9"})
	// El enlace de este SI lo lleva: un ELI es prefijo mas identificador.
	dentro := conIdentificador("urn:demo:dentro",
		corpus.Identificador{Tipo: corpus.ELIUE, Valor: "reg/9999/2/oj"})

	if strings.Contains(suelto.Enlace(), suelto.Identificador.Valor) {
		t.Fatalf("el caso 'suelto' ya no lo es: %q lleva %q dentro, asi que este test "+
			"no probaria lo que dice", suelto.Enlace(), suelto.Identificador.Valor)
	}
	if !strings.Contains(dentro.Enlace(), dentro.Identificador.Valor) {
		t.Fatalf("el caso 'dentro' ya no lo es: %q no lleva %q dentro",
			dentro.Enlace(), dentro.Identificador.Valor)
	}

	s, _ := superficie(t, []*corpus.Paquete{suelto, dentro})
	_, cuerpo := pedir(t, s, "/alcance")

	hueco := `class="identificador">` + suelto.Identificador.Valor + `<`
	if !strings.Contains(cuerpo, hueco) {
		t.Errorf("el pie no ensena el identificador %q de %s. Su enlace derivado no lo "+
			"lleva dentro, asi que sin ese hueco el lector no tiene en pantalla nada "+
			"que identifique la norma el dia que la pagina se mueva",
			suelto.Identificador.Valor, suelto.URN)
	}
	repetido := `class="identificador">` + dentro.Identificador.Valor + `<`
	if strings.Contains(cuerpo, repetido) {
		t.Errorf("el pie repite el identificador %q de %s, que ya va dentro de su enlace "+
			"(%q). En el pie de todas las paginas, eso es la misma cadena dos veces por "+
			"paquete", dentro.Identificador.Valor, dentro.URN, dentro.Enlace())
	}
	// Y los dos enlaces derivados salen, que es lo que se cita.
	for _, p := range []*corpus.Paquete{suelto, dentro} {
		if !strings.Contains(cuerpo, p.Enlace()) {
			t.Errorf("el pie no cita la fuente derivada de %s (%q)", p.URN, p.Enlace())
		}
	}
}
