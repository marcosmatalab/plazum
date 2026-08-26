package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// Las tres lentes, la cache y el orden, contra un TMview de mentira.
//
// POR QUE ESTA HERRAMIENTA MERECE ESTO. Su unico producto es una PRUEBA, y con
// esa prueba se decide un nombre. Ya fallo dos veces de la misma forma: la
// cabecera decia "EUIPO" mirara donde mirara, y el transporte podia contestar
// vacio sin que nada lo dijera. Las dos veces la salida se leia exactamente
// igual que una buena.
//
// Cada lente lleva su CONTROL NEGATIVO al lado, y no es el generico "algo
// falla": por cada lente se comprueba que encuentra lo suyo Y que NO se lleva
// por delante lo de las otras dos. Una lente que se dispara con todo no
// distingue nada, que es como el semaforo llego a decir ROJO siempre.

// tmviewDeMentira responde como TMview: un mapa de termino a marcas.
//
// Se registra QUE se le ha preguntado, porque parte de lo que hay que fijar es
// que la lente 3 pregunta por las subcadenas y no se las inventa.
type tmviewDeMentira struct {
	*httptest.Server
	porTermino map[string][]map[string]any
	preguntas  []string
}

func servidorTMview(t *testing.T, porTermino map[string][]map[string]any) *tmviewDeMentira {
	t.Helper()
	f := &tmviewDeMentira{porTermino: porTermino}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var pet struct {
			BasicSearch string `json:"basicSearch"`
			Page        string `json:"page"`
		}
		if err := json.NewDecoder(r.Body).Decode(&pet); err != nil {
			t.Errorf("el cribador mando un cuerpo ilegible: %v", err)
		}
		f.preguntas = append(f.preguntas, pet.BasicSearch)
		marcas := f.porTermino[pet.BasicSearch]
		if pet.Page != "1" {
			marcas = nil // una sola pagina: la segunda viene vacia y cierra el bucle
		}
		if marcas == nil {
			marcas = []map[string]any{}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"tradeMarks": marcas})
	}))
	t.Cleanup(f.Close)
	return f
}

// marcaDe arma un registro de TMview con lo que el cribador mira.
func marcaDe(numero, nombre string, clases ...int) map[string]any {
	if len(clases) == 0 {
		clases = []int{9, 42}
	}
	return map[string]any{
		"applicationNumber": numero,
		"tmName":            nombre,
		"tmOffice":          "EM",
		"tradeMarkStatus":   "Registered",
		"tradeMarkType":     "Word",
		"niceClass":         clases,
		"applicantName":     []string{"Titular " + numero},
	}
}

func cribaContra(url string) *criba {
	return &criba{
		oficina: "EM", clases: []int{9, 42}, base: url, sinCache: true,
		espera: time.Nanosecond, http: &http.Client{Timeout: 10 * time.Second},
	}
}

// El canario tiene que responder en todos los escenarios, o el programa entero
// se para antes de cribar. Se le da su propia respuesta siempre.
func conCanario(m map[string][]map[string]any) map[string][]map[string]any {
	if m == nil {
		m = map[string][]map[string]any{}
	}
	m[terminoCanario] = []map[string]any{marcaDe("canario", "TECNOLOGIA")}
	return m
}

// ---------------------------------------------------------------------------
// Las tres lentes, cada una con su control negativo
// ---------------------------------------------------------------------------

// LENTE 1, COLISION: existe una marca que se llama IGUAL que el candidato. Es
// la que todo el mundo hace.
func TestLente1ColisionEncuentraLoIgualYNoLoDemas(t *testing.T) {
	s := servidorTMview(t, conCanario(map[string][]map[string]any{
		// La consulta por el candidato devuelve tres cosas a la vez: una que se
		// llama igual, una que lo contiene y una que no es ni lo uno ni lo otro.
		"plazum": {
			marcaDe("001", "PLAZUM"),      // colision
			marcaDe("002", "MIPLAZUMPRO"), // contenedora
			marcaDe("003", "OTRACOSA"),    // ni una ni otra
		},
	}))
	h, err := cribaContra(s.URL).cribar("plazum")
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Colisiones) != 1 || h.Colisiones[0].Numero != "001" {
		t.Fatalf("la lente 1 tenia que encontrar exactamente PLAZUM: %+v", h.Colisiones)
	}
	// CONTROL NEGATIVO de la lente: lo que NO es una colision no cae aqui. Una
	// lente que se lleva todo lo que devuelve la consulta no distingue nada.
	for _, m := range h.Colisiones {
		if m.Numero == "002" || m.Numero == "003" {
			t.Errorf("la lente 1 se ha llevado %s, que no se llama igual", m.Numero)
		}
	}
	if h.Riesgo() != "ROJO" {
		t.Errorf("una marca registrada que se llama IGUAL en nuestras clases tiene que "+
			"pintar ROJO y pinta %q", h.Riesgo())
	}
}

// LENTE 2, CONTENEDORAS: una marca registrada CONTIENE al candidato.
func TestLente2ContenedorasEncuentraLoQueEnvuelveYNoLoDemas(t *testing.T) {
	s := servidorTMview(t, conCanario(map[string][]map[string]any{
		"vencia": {
			marcaDe("010", "AVENCIA"),  // contenedora: vencia dentro de avencia
			marcaDe("011", "VENTANIA"), // parecida y NO contenedora
		},
	}))
	h, err := cribaContra(s.URL).cribar("vencia")
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Contenedoras) != 1 || h.Contenedoras[0].Numero != "010" {
		t.Fatalf("la lente 2 tenia que encontrar AVENCIA y solo AVENCIA: %+v", h.Contenedoras)
	}
	if len(h.Colisiones) != 0 {
		t.Errorf("AVENCIA no se llama igual que vencia: no es colision: %+v", h.Colisiones)
	}
	// CONTROL NEGATIVO: VENTANIA no contiene "vencia" y no puede aparecer.
	for _, m := range h.Contenedoras {
		if m.Numero == "011" {
			t.Error("la lente 2 se ha llevado VENTANIA, que no contiene el candidato")
		}
	}
	// Y la cobertura: vencia son 6 de 7 letras de avencia, 86%, que a efectos
	// practicos es identidad. Por encima del umbral rojo.
	if h.Contenedoras[0].Nivel != nivelRojo {
		t.Errorf("vencia cubre el 86%% de avencia y el semaforo dice %q (cobertura %.2f)",
			h.Contenedoras[0].Nivel, h.Contenedoras[0].Cobertura)
	}
}

// LENTE 3, CONTENIDAS: el candidato CONTIENE una marca registrada. ESTA es la
// que nadie mira y la que costo DUTIQ.
func TestLente3ContenidasEsLaQueEncuentraUtiqDentroDeDutiq(t *testing.T) {
	s := servidorTMview(t, conCanario(map[string][]map[string]any{
		// La consulta por el candidato entero no devuelve nada: DUTIQ estaba
		// limpio por la lente 1 y por la 2. Ese fue el problema.
		"dutiq": {},
		// La subcadena si.
		"utiq": {marcaDe("018838934", "Utiq")},
		// Y una subcadena que devuelve algo que NO se llama como ella: no cuenta.
		"duti": {marcaDe("020", "DUTIFUL")},
	}))
	h, err := cribaContra(s.URL).cribar("dutiq")
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Colisiones) != 0 || len(h.Contenedoras) != 0 {
		t.Fatalf("por las lentes 1 y 2, DUTIQ salia limpio. Si aqui aparece algo, este "+
			"test ya no reproduce el fallo que costo la marca: %+v %+v", h.Colisiones, h.Contenedoras)
	}
	var utiq *marca
	for i := range h.Contenidas {
		if h.Contenidas[i].Numero == "018838934" {
			utiq = &h.Contenidas[i]
		}
	}
	if utiq == nil {
		t.Fatalf("HALLAZGO: la lente 3 no encuentra UTIQ dentro de DUTIQ. Es la lente que "+
			"esta herramienta existe para tener, y sin ella se vuelve a perder la marca: %+v",
			h.Contenidas)
	}
	// CONTROL NEGATIVO: DUTIFUL no se llama "duti", asi que la subcadena no lo
	// trae. Solo cuenta la coincidencia EXACTA con la subcadena; si no, cada
	// subcadena arrastraria media base de datos.
	for _, m := range h.Contenidas {
		if m.Numero == "020" {
			t.Error("la lente 3 se ha llevado DUTIFUL, que no ES la subcadena, solo empieza por ella")
		}
	}
	// Y la cobertura es lo que separa esto de una casualidad: utiq son 4 de 5
	// letras de dutiq, el 80%.
	if utiq.Nivel != nivelRojo {
		t.Errorf("utiq cubre el 80%% de dutiq y el semaforo dice %q (cobertura %.2f). Ese "+
			"numero es lo unico que separa este caso de un acronimo de tres letras",
			utiq.Nivel, utiq.Cobertura)
	}
	if utiq.Coincidente != "utiq" {
		t.Errorf("no se dice por que subcadena se encontro, asi que quien lea la prueba no "+
			"puede reproducirla: %q", utiq.Coincidente)
	}
}

// Y la lente 3 pregunta de verdad por las subcadenas. Sin esto, un cribador que
// no consultara ninguna daria "sin hallazgos" en la lente que mas importa, y la
// tabla se leeria igual.
func TestLaLente3PreguntaPorTodasLasSubcadenasDeCuatroLetrasOMas(t *testing.T) {
	s := servidorTMview(t, conCanario(nil))
	if _, err := cribaContra(s.URL).cribar("dutiq"); err != nil {
		t.Fatal(err)
	}
	preguntado := map[string]bool{}
	for _, p := range s.preguntas {
		preguntado[p] = true
	}
	for _, sub := range []string{"dutiq", "duti", "utiq", "dut", "uti", "tiq"} {
		if !preguntado[sub] {
			t.Errorf("no se ha preguntado por %q, y sale de subcadenas(%q). La lente 3 no "+
				"puede encontrar lo que no consulta", sub, "dutiq")
		}
	}
	// CONTROL NEGATIVO: no se pregunta por lo que no es subcadena propia. Un
	// cribador que consultara de mas gastaria una API publica y gratuita que no
	// nos debe nada, y ademas traeria ruido que hay que descartar despues.
	for _, noSub := range []string{"dq", "q", "utiqx", "xdutiq"} {
		if preguntado[noSub] {
			t.Errorf("se ha preguntado por %q, que no es subcadena de dutiq", noSub)
		}
	}
}

// ---------------------------------------------------------------------------
// El transporte vacio, que es el agujero que se encontro
// ---------------------------------------------------------------------------

// LA PROPIEDAD QUE IMPORTA, y no es que el canario devuelva error: es que el
// PROGRAMA no imprime tabla cuando el transporte esta roto.
//
// cribar() contra un transporte vacio devuelve, correctamente, cero hallazgos:
// no ha encontrado nada porque no hay nada que encontrar EN LO QUE LE HAN
// CONTESTADO. Lo unico que separa eso de un nombre limpio es que el canario
// corre ANTES. O sea que la propiedad vive en el ORDEN, y el orden solo se
// puede comprobar por aqui.
func TestConElTransporteVacioNoSeImprimeNingunaTabla(t *testing.T) {
	// Un servidor que contesta 200 con la lista vacia a TODO, canario incluido.
	s := servidorTMview(t, nil)

	var salida, errores bytes.Buffer
	codigo := ejecutar([]string{"-candidatos", "plazum", "-extremo", s.URL,
		"-sin-cache", "-cache", t.TempDir()}, &salida, &errores)

	if codigo == 0 {
		t.Fatalf("HALLAZGO: con el transporte contestando vacio el programa ha salido con 0.\n"+
			"  salida:\n%s", salida.String())
	}
	if salida.Len() != 0 {
		t.Errorf("HALLAZGO: se ha impreso algo pese a no haber llegado a preguntar. Una "+
			"tabla que dice 'sin hallazgos' porque el transporte esta roto se lee EXACTAMENTE "+
			"igual que una que dice que el nombre esta limpio:\n%s", salida.String())
	}
	if !strings.Contains(errores.String(), "canario") {
		t.Errorf("el error no dice que ha fallado el canario, asi que nadie sabe por donde "+
			"mirar: %s", errores.String())
	}

	// CONTROL NEGATIVO: con transporte sano, el mismo mandato imprime su tabla y
	// sale con 0. Sin esto, lo de arriba pasaria igual si el programa fallara
	// siempre.
	sano := servidorTMview(t, conCanario(map[string][]map[string]any{
		"plazum": {marcaDe("001", "PLAZUM")},
	}))
	salida.Reset()
	errores.Reset()
	codigo = ejecutar([]string{"-candidatos", "plazum", "-extremo", sano.URL,
		"-sin-cache", "-cache", t.TempDir()}, &salida, &errores)
	if codigo != 0 {
		t.Fatalf("con transporte sano tenia que salir 0 y salio %d: %s", codigo, errores.String())
	}
	if !strings.Contains(salida.String(), "PLAZUM") {
		t.Errorf("con transporte sano no se ha impreso la tabla:\n%s", salida.String())
	}
}

// ---------------------------------------------------------------------------
// La cache
// ---------------------------------------------------------------------------

func TestLaCacheAhorraConsultasYNoCambiaElVeredicto(t *testing.T) {
	s := servidorTMview(t, conCanario(map[string][]map[string]any{
		"dutiq": {},
		"utiq":  {marcaDe("018838934", "Utiq")},
	}))
	dir := t.TempDir()
	c := &criba{oficina: "EM", clases: []int{9, 42}, base: s.URL, cache: dir,
		espera: time.Nanosecond, http: &http.Client{Timeout: 10 * time.Second}}

	primera, err := c.cribar("dutiq")
	if err != nil {
		t.Fatal(err)
	}
	if primera.Consultas == 0 {
		t.Fatal("la primera pasada no ha salido a la red")
	}
	if primera.DesdeCache != 0 {
		t.Fatalf("la primera pasada ha leido %d de una cache que estaba vacia", primera.DesdeCache)
	}

	segunda, err := c.cribar("dutiq")
	if err != nil {
		t.Fatal(err)
	}
	if segunda.DesdeCache == 0 {
		t.Error("la segunda pasada no ha usado la cache: se vuelve a martillear una API " +
			"publica y gratuita que no nos debe nada")
	}
	// Y EL VEREDICTO NO CAMBIA. Una cache que devuelve otra cosa que la red es
	// peor que no tener cache: la prueba dependeria de si alguien borro un
	// directorio.
	if len(segunda.Contenidas) != len(primera.Contenidas) || segunda.Riesgo() != primera.Riesgo() {
		t.Fatalf("la cache cambia el veredicto.\n  con red:   %s, %d contenidas\n"+
			"  de cache:  %s, %d contenidas", primera.Riesgo(), len(primera.Contenidas),
			segunda.Riesgo(), len(segunda.Contenidas))
	}
	// Y hay ficheros de verdad, con la oficina en el nombre: la misma subcadena
	// en dos registros distintos no puede compartir entrada.
	entradas, err := os.ReadDir(dir)
	if err != nil || len(entradas) == 0 {
		t.Fatalf("no se ha escrito nada en la cache (%v)", err)
	}
	for _, e := range entradas {
		if !strings.HasPrefix(e.Name(), "EM-") {
			t.Errorf("la entrada de cache %q no lleva la oficina en el nombre: una consulta "+
				"a la OEPM se serviria con la respuesta de EUIPO", e.Name())
		}
	}
}

// Una cache ilegible no puede envenenar el veredicto: se ignora y se pregunta.
func TestUnaCacheCorruptaNoSeCuelaComoRespuesta(t *testing.T) {
	s := servidorTMview(t, conCanario(map[string][]map[string]any{
		"utiq":  {marcaDe("018838934", "Utiq")},
		"dutiq": {},
	}))
	dir := t.TempDir()
	c := &criba{oficina: "EM", clases: []int{9, 42}, base: s.URL, cache: dir,
		espera: time.Nanosecond, http: &http.Client{Timeout: 10 * time.Second}}

	// Se envenena la entrada de "utiq" con basura ANTES de cribar.
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(c.rutaCache("utiq"), []byte("{no es json"), 0o600); err != nil {
		t.Fatal(err)
	}

	h, err := c.cribar("dutiq")
	if err != nil {
		t.Fatal(err)
	}
	// Tiene que haber ido a la red y haber encontrado UTIQ igualmente.
	encontrado := false
	for _, m := range h.Contenidas {
		if m.Numero == "018838934" {
			encontrado = true
		}
	}
	if !encontrado {
		t.Fatal("HALLAZGO: una entrada de cache ilegible ha hecho desaparecer un hallazgo. " +
			"Un fichero corrupto en un directorio de cache no puede limpiar un nombre")
	}
}

// ---------------------------------------------------------------------------
// La salida
// ---------------------------------------------------------------------------

// El umbral descarta, pero NO en silencio: si no se dice cuanto se ha dejado
// fuera, "sin hallazgos" se lee como "se ha mirado todo".
func TestLoQueQuedaBajoElUmbralSeCuentaYSeDice(t *testing.T) {
	h := Hallazgo{Candidato: "plazum", Contenidas: []marca{
		{Numero: "016915233", Nombre: "AZU", Coincidente: "azu", Clases: []int{9}},
	}}
	clasificar(&h)
	if h.Contenidas[0].Nivel != nivelRuido {
		t.Fatalf("azu son tres letras de seis, el 50%%: es ruido y sale %q", h.Contenidas[0].Nivel)
	}

	verTodo = false
	var sinTodo bytes.Buffer
	seccion(&sinTodo, "CONTENIDAS", h.Contenidas, "")
	if !strings.Contains(sinTodo.String(), "por debajo del umbral") {
		t.Errorf("HALLAZGO: lo descartado no se cuenta. Un umbral que descarta en silencio "+
			"hace que 'sin hallazgos' se lea como 'se ha mirado todo':\n%s", sinTodo.String())
	}
	if strings.Contains(sinTodo.String(), "016915233") {
		t.Errorf("sin -todo, el ruido no se pinta:\n%s", sinTodo.String())
	}

	// CONTROL NEGATIVO: con -todo SI se lista. Si no se listara, no habria forma
	// de auditar el umbral y habria que creerselo.
	verTodo = true
	defer func() { verTodo = false }()
	var conTodo bytes.Buffer
	seccion(&conTodo, "CONTENIDAS", h.Contenidas, "")
	if !strings.Contains(conTodo.String(), "016915233") {
		t.Errorf("con -todo el ruido tiene que listarse, o el umbral no se puede auditar:\n%s",
			conTodo.String())
	}
}

// La cabecera dice QUE REGISTRO se ha consultado, y el rotulo no se cablea.
// Ya mintio una vez: decia EUIPO mirara donde mirara, y con -oficina ES los
// numeros que salian eran espanoles.
func TestLaTablaDiceElRegistroDeVerdad(t *testing.T) {
	for _, caso := range []struct{ codigo, espera string }{
		{"EM", "EUIPO"},
		{"ES", "OEPM"},
		{"XX", "codigo de oficina XX"},
	} {
		var b bytes.Buffer
		imprimirTabla(&b, []Hallazgo{{Candidato: "plazum"}}, []int{9, 42}, caso.codigo)
		if !strings.Contains(b.String(), caso.espera) {
			t.Errorf("con -oficina %s la cabecera tenia que decir %q:\n%s",
				caso.codigo, caso.espera, b.String())
		}
	}
	// Y el paso manual sigue en la salida: es lo que descarto al finalista mejor.
	var b bytes.Buffer
	imprimirTabla(&b, []Hallazgo{{Candidato: "plazum"}}, []int{9, 42}, "EM")
	for _, quiero := range []string{"FALTA UN PASO", "dictamen juridico"} {
		if !strings.Contains(b.String(), quiero) {
			t.Errorf("la salida ya no dice %q, y sin eso se lee como un dictamen:\n%s",
				quiero, b.String())
		}
	}
}

func TestLaSalidaJSONLlevaLoQueHaceFaltaParaReproducirla(t *testing.T) {
	s := servidorTMview(t, conCanario(map[string][]map[string]any{
		"dutiq": {},
		"utiq":  {marcaDe("018838934", "Utiq")},
	}))
	var salida, errores bytes.Buffer
	if codigo := ejecutar([]string{"-candidatos", "dutiq", "-extremo", s.URL, "-json",
		"-sin-cache", "-cache", t.TempDir()}, &salida, &errores); codigo != 0 {
		t.Fatalf("salio %d: %s", codigo, errores.String())
	}
	var hs []Hallazgo
	if err := json.Unmarshal(salida.Bytes(), &hs); err != nil {
		t.Fatalf("la salida JSON no es JSON: %v\n%s", err, salida.String())
	}
	if len(hs) != 1 || len(hs[0].Contenidas) == 0 {
		t.Fatalf("el JSON no trae el hallazgo: %s", salida.String())
	}
	m := hs[0].Contenidas[0]
	if m.Coincidente == "" || m.Nivel == "" || m.Cobertura == 0 {
		t.Errorf("al JSON le falta lo que hace reproducible la prueba (por que subcadena, "+
			"que nivel, cuanta cobertura): %+v", m)
	}
	// Y el numero de consultas, que es como se sabe si la prueba salio de la red
	// o de una cache vieja.
	if hs[0].Consultas == 0 && hs[0].DesdeCache == 0 {
		t.Error("el JSON no dice cuantas consultas se hicieron, asi que no se puede saber " +
			"si esta prueba llego a preguntar")
	}
}

// Los errores de uso salen con codigo distinto de los de fondo: 2 es "lo has
// llamado mal" y 1 es "no he podido cribar". Confundirlos hace que un guion no
// pueda distinguir un error de teclado de un nombre sucio.
func TestLosErroresDeUsoSeDistinguenDeLosDeFondo(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		args   []string
		espera int
	}{
		{"sin candidatos", []string{}, 2},
		{"una clase que no existe", []string{"-candidatos", "x", "-clases", "99"}, 2},
		{"una clase que no es numero", []string{"-candidatos", "x", "-clases", "nueve"}, 2},
		{"una bandera inventada", []string{"-candidatos", "x", "-loquesea"}, 2},
	} {
		var salida, errores bytes.Buffer
		if got := ejecutar(caso.args, &salida, &errores); got != caso.espera {
			t.Errorf("%s: esperaba %d y salio %d (%s)", caso.nombre, caso.espera, got, errores.String())
		}
		if salida.Len() != 0 {
			t.Errorf("%s: se ha impreso una tabla pese al error de uso:\n%s", caso.nombre, salida.String())
		}
	}
}

// Un candidato vacio entre comas no cuenta como candidato, pero tampoco tumba
// la ejecucion: "plazum,,vencia" son dos nombres.
func TestLasComasDeMasNoCuentanComoCandidatos(t *testing.T) {
	s := servidorTMview(t, conCanario(nil))
	var salida, errores bytes.Buffer
	if codigo := ejecutar([]string{"-candidatos", "plazum,,vencia,", "-extremo", s.URL,
		"-json", "-sin-cache", "-cache", t.TempDir()}, &salida, &errores); codigo != 0 {
		t.Fatalf("salio %d: %s", codigo, errores.String())
	}
	var hs []Hallazgo
	if err := json.Unmarshal(salida.Bytes(), &hs); err != nil {
		t.Fatal(err)
	}
	if len(hs) != 2 {
		t.Fatalf("dos candidatos y salieron %d: %s", len(hs), salida.String())
	}
}

// listaClases y recortar, que son de la salida y estaban sin tocar.
func TestLosAyudantesDeLaSalida(t *testing.T) {
	if got := listaClases(nil); got != "todas" {
		t.Errorf("sin clases la cabecera tiene que decir `todas` y dice %q", got)
	}
	if got := listaClases([]int{9, 42}); got != "9,42" {
		t.Errorf("listaClases: %q", got)
	}
	for _, caso := range []struct {
		s      string
		n      int
		espera string
	}{
		{"corto", 10, "corto"},
		{"exactamente", 11, "exactamente"},
		{"demasiado largo", 8, "demasia."},
		{"x", 1, "x"},
	} {
		if got := recortar(caso.s, caso.n); got != caso.espera {
			t.Errorf("recortar(%q, %d) = %q, esperaba %q", caso.s, caso.n, got, caso.espera)
		}
		if len(recortar(caso.s, caso.n)) > caso.n {
			t.Errorf("recortar(%q, %d) devuelve %d caracteres: la tabla se desalinea",
				caso.s, caso.n, len(recortar(caso.s, caso.n)))
		}
	}
}

// La cabecera de la tabla y el numero de consultas, juntos: es lo que hace que
// una prueba pegada en un documento se pueda auditar dentro de un ano.
func TestLaTablaDiceCuantoHaPreguntado(t *testing.T) {
	var b bytes.Buffer
	imprimirTabla(&b, []Hallazgo{{Candidato: "plazum", Consultas: 19, DesdeCache: 0}}, []int{9, 42}, "EM")
	if !strings.Contains(b.String(), "19 consultas") {
		t.Errorf("la tabla no dice cuantas consultas se hicieron:\n%s", b.String())
	}
	if !strings.Contains(b.String(), "sin hallazgos") {
		t.Errorf("un candidato sin nada tiene que decirlo:\n%s", b.String())
	}
}
