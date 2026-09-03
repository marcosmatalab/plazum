package camino

import (
	"html/template"
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// EL ARMAZON COMPARTIDO: las puertas de la pieza que ahora pintan cuatro
// superficies en vez de una.
//
// La barra lateral vivia dentro de superficies/pantallas y solo la tenian sus
// seis pantallas. Ahora la plantilla es una sola y la carga quien la quiera, asi
// que lo que hay que sostener es distinto de antes: que la union de sistemas de
// ficheros SIRVE de verdad los dos arboles (si sirviera solo uno, la superficie
// arrancaria sin barra o no arrancaria), que los nombres de plantilla no chocan
// (dos "pagina" en el mismo arbol se pisan sin decir nada) y que la tira sigue
// saliendo del camino y no de una lista escrita al lado.

// LA UNION SIRVE LOS DOS ARBOLES. Sin esto la superficie no arranca, pero el
// error diria "los patrones no casan con ningun fichero", que manda a mirar la
// directiva go:embed en vez de a la union.
func TestLaUnionDePlantillasSirveLosDosArboles(t *testing.T) {
	u := ConArmazon(plantillasFS)

	// El arbol PROPIO.
	if _, err := fs.ReadFile(u, "plantillas/camino.html"); err != nil {
		t.Errorf("la union no sirve la plantilla propia: %v", err)
	}
	// Y el del ARMAZON.
	if _, err := fs.ReadFile(u, "armazon/armazon.html"); err != nil {
		t.Errorf("la union no sirve la plantilla del armazon: %v", err)
	}
	// Y por los DOS caminos que usan de verdad fs.Glob y template.ParseFS:
	// ReadDir (para resolver el patron) y Open (para lo que caiga por el camino
	// lento). Comprobar solo ReadFile dejaria fuera justo el que hace que el
	// fichero aparezca en el patron.
	for _, dir := range []string{"plantillas", "armazon"} {
		e, err := fs.ReadDir(u, dir)
		if err != nil || len(e) == 0 {
			t.Errorf("la union no lista el directorio %q (%d entradas, err=%v): fs.Glob "+
				"no encontraria el fichero y el patron casaria con cero", dir, len(e), err)
		}
	}
	for _, f := range []string{"plantillas/camino.html", "armazon/armazon.html"} {
		fh, err := u.Open(f)
		if err != nil {
			t.Errorf("la union no abre %q: %v", f, err)
			continue
		}
		_ = fh.Close()
	}
	// Y lo que NO existe sigue sin existir: una union que responda a todo
	// convertiria un nombre mal escrito en un fichero vacio.
	if _, err := fs.ReadFile(u, "plantillas/no-existe.html"); err == nil {
		t.Error("la union ha servido un fichero que no existe en ninguno de los dos arboles")
	}
	// Y el glob que usa de verdad el motor casa con los dos.
	for _, patron := range []string{"plantillas/*.html", PatronDelArmazon} {
		m, err := fs.Glob(u, patron)
		if err != nil || len(m) == 0 {
			t.Errorf("el patron %q no casa con nada en la union (%v): asi arranca el motor "+
				"y asi fallaria", patron, err)
		}
	}
}

// SIN PLANTILLAS PROPIAS LA UNION NO PANICA. El olvido tiene que salir por donde
// sale todo lo demas (el error de arranque del motor), no por un nil dentro de
// la primera peticion de un cliente.
func TestLaUnionConNadaPropioDevuelveElArmazonYNoPanica(t *testing.T) {
	u := ConArmazon(nil)
	if _, err := fs.ReadFile(u, "armazon/armazon.html"); err != nil {
		t.Errorf("sin plantillas propias la union tendria que servir al menos el armazon: %v", err)
	}
}

// LOS NOMBRES NO CHOCAN, y este es el fallo que un motor de plantillas NO avisa:
// dos {{define "pagina"}} en el mismo arbol se pisan en silencio y gana el
// ultimo que se parsee, que depende del orden del glob.
func TestElArmazonNoDefineNingunNombreQueYaSeaDeUnaSuperficie(t *testing.T) {
	nombres := nombresDefinidosEn(t, armazonFS, "armazon/armazon.html")
	sort.Strings(nombres)
	esperados := NombresDelArmazon()
	sort.Strings(esperados)
	if strings.Join(nombres, ",") != strings.Join(esperados, ",") {
		t.Errorf("el armazon define %v y NombresDelArmazon() dice %v. La lista existe para "+
			"que la puerta de colision pueda compararla sin leer el fichero a ojo: si se "+
			"separan, la puerta vigila otra cosa", nombres, esperados)
	}
	// Y ninguno de esos nombres puede ser uno de los que define la pantalla del
	// camino, que es la superficie que carga las dos a la vez.
	propios := nombresDefinidosEn(t, plantillasFS, "plantillas/camino.html")
	for _, n := range nombres {
		for _, p := range propios {
			if n == p {
				t.Errorf("el armazon y la pantalla del camino definen los dos %q: en el "+
					"mismo arbol una se come a la otra y no lo dice nadie", n)
			}
		}
	}
	// Control positivo del detector: tiene que encontrar algo en las dos.
	if len(nombres) == 0 || len(propios) == 0 {
		t.Fatalf("el detector de definiciones ha encontrado %d en el armazon y %d en la "+
			"pantalla: esta mirando otra cosa", len(nombres), len(propios))
	}
}

func nombresDefinidosEn(t *testing.T, sistema fs.FS, fichero string) []string {
	t.Helper()
	b, err := fs.ReadFile(sistema, fichero)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, m := range regexp.MustCompile(`\{\{-?\s*define "([^"]+)"`).
		FindAllStringSubmatch(string(b), -1) {
		out = append(out, m[1])
	}
	return out
}

// EL ARMAZON SE PARSEA. Un error de sintaxis en la plantilla compartida rompe
// las cuatro superficies a la vez, asi que se comprueba aqui y no de rebote.
func TestElArmazonSeParsea(t *testing.T) {
	_, err := template.New("x").
		Funcs(template.FuncMap{"t": func(string, ...any) string { return "" }}).
		ParseFS(armazonFS, PatronDelArmazon)
	if err != nil {
		t.Fatalf("la plantilla compartida no se parsea: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TiraDe: la regla de la barra, escrita una vez
// ---------------------------------------------------------------------------

// LOS PASOS QUE TODAVIA NO SON PANTALLA LLEVAN A LA PANTALLA DEL CAMINO.
//
// No se ocultan (una tira de seis de la que se ven cuatro parece completa) y no
// se quedan sin enlace (un paso al que no se puede ir ensena a ignorar la barra
// entera). Van marcados con EsPantalla=false para que la plantilla los pinte
// apagados: sin ese campo, quien pulsara el calendario esperaria el calendario.
func TestLosPasosSinPantallaLlevanAlCaminoYSeMarcanComoTales(t *testing.T) {
	tira := TiraDe(Canonico(), "", BasePorDefecto+"/", "", "")
	if len(tira) != len(Canonico()) {
		t.Fatalf("la tira trae %d pasos y el camino tiene %d", len(tira), len(Canonico()))
	}
	sinPantalla := 0
	for _, p := range tira {
		if p.EsPantalla {
			continue
		}
		sinPantalla++
		if p.URL != BasePorDefecto+"/" {
			t.Errorf("el paso %q no es pantalla y su enlace es %q: tendria que llevar a la "+
				"pantalla del camino, que es donde esta la orden que lo hace hoy", p.ID, p.URL)
		}
	}
	// CONTROL POSITIVO: si el camino se quedara sin pasos por hacer, este test
	// no estaria comprobando nada y daria verde igual.
	if quedan := len(SinPantalla()); sinPantalla != quedan {
		t.Errorf("la tira marca %d pasos sin pantalla y el camino declara %d: el recuento "+
			"de la deuda y lo que se pinta se han separado", sinPantalla, quedan)
	}
	if sinPantalla == 0 {
		t.Skip("ya no quedan pasos sin pantalla: esta puerta se queda sin caso que recorrer " +
			"y hay que decidir si se retira")
	}
}

// LA CONSULTA SOLO SE CUELGA DE LOS PASOS QUE LEEN EL ALCANCE.
//
// Las respuestas de la entrevista viajan en la direccion y no se guardan, asi
// que un enlace pelado se come el trabajo de quien lo pulse: ese fallo ya lo
// tuvo el camino una vez. Y colgarsela a los que no la leen (el acta, la
// revision de accesos) seria arrastrar un dato hasta una pantalla que no lo
// entiende.
func TestLaConsultaViajaSoloEnLosPasosQueLeenElAlcance(t *testing.T) {
	const q = "p1=si&p2=no"
	tira := TiraDe(Canonico(), "", BasePorDefecto+"/", "", q)
	porID := map[string]Paso{}
	for _, p := range Canonico() {
		porID[p.ID] = p
	}
	conAlcance, sinAlcance := 0, 0
	for _, e := range tira {
		lleva := strings.Contains(e.URL, q)
		p := porID[e.ID]
		switch {
		case p.LlevaAlcance:
			conAlcance++
			if !lleva {
				t.Errorf("el paso %q lee el alcance y su enlace %q va pelado: pulsarlo "+
					"borra la entrevista de quien lo use", e.ID, e.URL)
			}
		case p.EsPantalla():
			sinAlcance++
			if lleva {
				t.Errorf("el paso %q no lee el alcance y su enlace %q arrastra la consulta "+
					"hasta una pantalla que no la entiende", e.ID, e.URL)
			}
		}
	}
	// CONTROL POSITIVO de las dos ramas: si el camino dejara de tener pasos de
	// una de las dos clases, la mitad de este test seria decorativa.
	if conAlcance == 0 || sinAlcance == 0 {
		t.Fatalf("el camino trae %d pasos con alcance y %d pantallas sin el: una de las dos "+
			"ramas de este test no recorre nada", conAlcance, sinAlcance)
	}
}

// EL PASO ACTUAL SE MARCA POR IDENTIFICADOR, y solo uno.
//
// Es el invariante 7 en la barra de navegacion: el emparejamiento entre el paso
// y su marca casa por Paso.ID, que es lo que declara el camino, y NUNCA por la
// posicion en la lista. Un identificador que no sea ninguno no marca nada, que
// es lo que le toca a una pantalla que no esta en el camino.
func TestElPasoActualSeMarcaPorIdentificadorYSoloUno(t *testing.T) {
	tira := TiraDe(Canonico(), "", BasePorDefecto+"/", IDDelActa, "")
	marcados := 0
	for _, e := range tira {
		if e.Actual {
			marcados++
			if e.ID != IDDelActa {
				t.Errorf("se ha marcado el paso %q y se pidio %q", e.ID, IDDelActa)
			}
		}
	}
	if marcados != 1 {
		t.Fatalf("hay %d pasos marcados y tiene que haber 1", marcados)
	}
	// Y UNA PANTALLA QUE NO ES NINGUN PASO no marca ninguno. Marcar el primero
	// "por si acaso" seria decirle al operador que esta donde no esta.
	for _, e := range TiraDe(Canonico(), "", BasePorDefecto+"/", "personas", "") {
		if e.Actual {
			t.Errorf("con un identificador que no es ningun paso se ha marcado %q", e.ID)
		}
	}
}

// SIN PASOS, NI TIRA. Las dos formas de la nada (invariante 8): la peligrosa es
// la nil, porque es la que sale por olvidarse.
func TestSinPasosNoHayTira(t *testing.T) {
	for _, c := range []struct {
		nombre string
		pasos  []Paso
	}{
		{"nil", nil},
		{"vacio presente", []Paso{}},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			if v := TiraDe(c.pasos, "", BasePorDefecto+"/", "", ""); v != nil {
				t.Errorf("con el camino %s ha salido una tira de %d pasos", c.nombre, len(v))
			}
		})
	}
}

// ---------------------------------------------------------------------------
// La pantalla del camino, ya con su armazon
// ---------------------------------------------------------------------------

// LA PANTALLA DEL CAMINO TAMBIEN LLEVA BARRA, y no marca ningun paso.
//
// Lo segundo no es un olvido: el camino no es uno de sus propios pasos. Marcar
// el primero seria decirle al operador que esta en el alcance cuando no lo esta.
func TestLaPantallaDelCaminoLlevaBarraYNoMarcaNingunPaso(t *testing.T) {
	s := superficie(t, Canonico())
	_, cuerpo := pedir(t, s, BasePorDefecto+"/")
	m := regexp.MustCompile(`(?s)<nav class="tira-camino".*?</nav>`).FindString(cuerpo)
	if m == "" {
		t.Fatal("la pantalla del camino no trae la barra lateral: es la unica de las cuatro " +
			"que se abre sin saber nada, asi que es la que menos puede quedarse sin marco")
	}
	if strings.Contains(m, `aria-current="step"`) {
		t.Error("la barra de la pantalla del camino marca un paso como el actual. El camino " +
			"no es uno de sus propios pasos")
	}
	if !strings.Contains(cuerpo, `<a class="marca"`) {
		t.Error("la pantalla del camino no trae la marca del producto en la barra")
	}
	// Y la marca lleva a la raiz del sitio, no a la pantalla del camino.
	href := regexp.MustCompile(`<a class="marca" href="([^"]*)"`).FindStringSubmatch(cuerpo)
	if href == nil || href[1] != "/" {
		t.Errorf("la marca apunta a %v y tiene que apuntar a la raiz del sitio", href)
	}
}

// LAS CLAVES DEL ARMAZON SON LAS QUE PIDE EL FICHERO, en las dos direcciones.
//
// Existe porque esa lista la copian las cuatro superficies en su contrato de
// catalogo: si se queda corta, sale una clave cruda en la barra lateral de las
// cuatro a la vez.
func TestLasClavesDelArmazonSonLasQuePideSuPlantilla(t *testing.T) {
	b, err := fs.ReadFile(armazonFS, "armazon/armazon.html")
	if err != nil {
		t.Fatal(err)
	}
	pide := map[string]bool{}
	for _, m := range regexp.MustCompile(`\{\{-?\s*t\s+"([^"]+)"`).
		FindAllStringSubmatch(string(b), -1) {
		pide[m[1]] = true
	}
	if len(pide) < 3 {
		t.Fatalf("el armazon pide %d claves literales: el detector esta midiendo el vacio",
			len(pide))
	}
	declara := map[string]bool{}
	for _, k := range ClavesDelArmazon() {
		declara[k] = true
	}
	for k := range pide {
		if !declara[k] {
			t.Errorf("el armazon pide %q y ClavesDelArmazon() no la declara: saldria en "+
				"crudo en la barra lateral de las cuatro superficies", k)
		}
	}
	for k := range declara {
		if !pide[k] {
			t.Errorf("ClavesDelArmazon() declara %q y el fichero no la pide: es peso muerto "+
				"que hay que traducir a cada idioma nuevo", k)
		}
	}
}
