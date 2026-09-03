package pantallas

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// El doble del almacen
// ---------------------------------------------------------------------------

// almacenFalso es un Alcances en memoria. Guarda lo mismo que el de verdad y
// ademas anota QUE se le pidio, que es lo que permite comprobar que la
// superficie no escribe cuando no debe.
type almacenFalso struct {
	mu     sync.Mutex
	por    map[string]map[string]Respuesta
	cuando map[string]time.Time
	// falla, si no es nil, es lo que devuelve toda operacion. Es el arnes del
	// caso «el almacen no se puede leer», que es donde vive la tentacion de
	// degradar a «no has contestado nada».
	falla error
	// escrituras cuenta las llamadas que MUTAN. Un cero aqui, en un caso que
	// tenia que rechazarse, es la afirmacion de que no se escribio.
	escrituras int
}

func nuevoAlmacenFalso() *almacenFalso {
	return &almacenFalso{por: map[string]map[string]Respuesta{}, cuando: map[string]time.Time{}}
}

func (a *almacenFalso) De(_ context.Context, usuario string) (AlcanceGuardado, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.falla != nil {
		return AlcanceGuardado{}, a.falla
	}
	out := AlcanceGuardado{Respuestas: map[string]Respuesta{}, Actualizado: a.cuando[usuario]}
	for k, v := range a.por[usuario] {
		out.Respuestas[k] = v
	}
	return out, nil
}

func (a *almacenFalso) Responder(_ context.Context, usuario, pregunta string, r Respuesta) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.falla != nil {
		return a.falla
	}
	a.escrituras++
	if a.por[usuario] == nil {
		a.por[usuario] = map[string]Respuesta{}
	}
	a.por[usuario][pregunta] = r
	a.cuando[usuario] = time.Date(2026, 9, 4, 9, 30, 0, 0, time.UTC)
	return nil
}

func (a *almacenFalso) Olvidar(_ context.Context, usuario, pregunta string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.falla != nil {
		return a.falla
	}
	a.escrituras++
	delete(a.por[usuario], pregunta)
	a.cuando[usuario] = time.Date(2026, 9, 4, 9, 30, 0, 0, time.UTC)
	return nil
}

func (a *almacenFalso) Reemplazar(_ context.Context, usuario string, rs map[string]Respuesta) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.falla != nil {
		return a.falla
	}
	a.escrituras++
	nuevo := map[string]Respuesta{}
	for k, v := range rs {
		nuevo[k] = v
	}
	a.por[usuario] = nuevo
	a.cuando[usuario] = time.Date(2026, 9, 4, 9, 30, 0, 0, time.UTC)
	return nil
}

func (a *almacenFalso) tiene(usuario string) map[string]Respuesta {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := map[string]Respuesta{}
	for k, v := range a.por[usuario] {
		out[k] = v
	}
	return out
}

func (a *almacenFalso) escrituraCuenta() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.escrituras
}

// conGuardado monta la superficie COMO LA MONTA EL PRODUCTO: con almacen, con
// sujeto de sesion y con token.
func conGuardado(al Alcances, quien string) func(*Opciones) {
	return func(o *Opciones) {
		o.Alcances = al
		o.Quien = func(*http.Request) string { return quien }
		o.Tokens = func(*http.Request) (string, error) { return "tok3n", nil }
	}
}

// enviar hace un POST al guardado con los campos dados.
func enviar(t *testing.T, s *Superficie, campos url.Values) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/alcance", strings.NewReader(campos.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}

// pedirPost envia el formulario poniendo el token por su cuenta. Es lo que usa
// el barrido de claves de catalogo, que solo quiere pintar las paginas de fallo
// del guardado.
func pedirPost(t *testing.T, s *Superficie, campos url.Values) {
	t.Helper()
	c := url.Values{CampoCSRF: {"tok3n"}}
	for k, vs := range campos {
		c[k] = vs
	}
	enviar(t, s, c)
}

// ---------------------------------------------------------------------------
// El criterio: contestar, cerrar, volver
// ---------------------------------------------------------------------------

// TestLoQueSeContestaSeGuardaYVuelveSinNadaEnLaDireccion es el criterio entero
// del frente, comprobado sobre la superficie: se contesta, la respuesta se
// escribe, la redireccion NO lleva las respuestas dentro, y al volver a
// `/alcance` a secas estan.
func TestLoQueSeContestaSeGuardaYVuelveSinNadaEnLaDireccion(t *testing.T) {
	al := nuevoAlmacenFalso()
	s, _ := superficie(t, corpusDemo(), conGuardado(al, "ciso"))

	w := enviar(t, s, url.Values{
		CampoCSRF: {"tok3n"}, "accion": {"si"}, "pregunta": {"alfa.q.categoria"}})
	if w.Code != http.StatusSeeOther {
		t.Fatalf("guardar contesta %d y tenia que redirigir con 303: sin redireccion, "+
			"recargar reenvia el formulario.\n%s", w.Code, w.Body.String())
	}
	destino := w.Header().Get("Location")
	if strings.Contains(destino, ParamSi+"=") || strings.Contains(destino, ParamNo+"=") {
		t.Errorf("la redireccion lleva las respuestas dentro (%q). Si viajan en la direccion, "+
			"no se ha guardado nada: se ha vuelto a lo de antes con un POST delante", destino)
	}
	if al.tiene("ciso")["alfa.q.categoria"] != Si {
		t.Fatalf("la respuesta no ha llegado al almacen: %v", al.tiene("ciso"))
	}

	// Y AL VOLVER, SIN NADA EN LA DIRECCION, ESTA.
	w2, cuerpo := pedir(t, s, "/alcance")
	if w2.Code != http.StatusOK {
		t.Fatalf("volver a /alcance da %d", w2.Code)
	}
	exige(t, cuerpo,
		rotulo("es", "alcance.pregunta.respondida_si"),
		rotulo("es", "alcance.guardado.en_tu_cuenta"),
		rotulo("es", "alcance.guardado.cuando"),
	)
	prohibe(t, cuerpo, rotulo("es", "alcance.derivacion.no_guardado"))
}

// Y la vuelta alcanza a las SEIS pantallas, no solo a la entrevista. Si solo
// Alcance leyera de la cuenta, quien entrase por Controles veria su tabla en
// blanco y la entrevista con todo respondido: el producto contandose dos cosas
// distintas a si mismo.
func TestLoGuardadoAlcanzaATodasLasPantallas(t *testing.T) {
	al := nuevoAlmacenFalso()
	_ = al.Responder(context.Background(), "ciso", "alfa.q.categoria", No)
	s, _ := superficie(t, corpusDemo(), conGuardado(al, "ciso"))

	_, cuerpo := pedir(t, s, "/controles?f=no_aplica")
	// alfa.o.auditoria depende de alfa.q.categoria: con un «no» guardado tiene
	// que estar en la cesta de las que no aplican.
	if !strings.Contains(cuerpo, "alfa.o.auditoria") {
		t.Errorf("Controles no refleja lo guardado en la cuenta: filtrando por «no aplica» no "+
			"sale la obligacion que el «no» acaba de descartar.\n%s", primerosMil(cuerpo))
	}
}

func primerosMil(s string) string {
	if len(s) > 1000 {
		return s[:1000]
	}
	return s
}

// ---------------------------------------------------------------------------
// Contra el atacante
// ---------------------------------------------------------------------------

// UNA ESCRITURA SIN AUTOR NO SE ATIENDE. Sin esta guarda, el alcance de una
// organizacion acabaria en un cajon sin dueno que leeria el siguiente que
// entrase sin identificarse.
func TestSinSesionNoSeGuardaNada(t *testing.T) {
	al := nuevoAlmacenFalso()
	s, _ := superficie(t, corpusDemo(), conGuardado(al, "" /* nadie ha entrado */))

	w := enviar(t, s, url.Values{
		CampoCSRF: {"tok3n"}, "accion": {"si"}, "pregunta": {"alfa.q.categoria"}})
	if w.Code != http.StatusForbidden {
		t.Errorf("guardar sin sesion contesta %d y tenia que ser 403", w.Code)
	}
	if al.escrituraCuenta() != 0 {
		t.Errorf("se ha escrito %d veces sin autor", al.escrituraCuenta())
	}
	// Y la pantalla no ofrece botones que no funcionan: sin sesion sigue
	// diciendo lo de siempre.
	_, cuerpo := pedir(t, s, "/alcance")
	exige(t, cuerpo, rotulo("es", "alcance.derivacion.no_guardado"))
	if strings.Contains(cuerpo, `name="accion"`) {
		t.Error("sin sesion la pantalla pinta formularios de guardado, o sea botones que " +
			"contestan 403 sin decir por que")
	}
}

// UNA CUENTA NO LEE NI PISA EL ALCANCE DE OTRA. La superficie escribe SIEMPRE
// bajo el sujeto de la sesion, y no bajo ningun campo del envio: si el usuario
// viniera del formulario, cambiarlo seria escribir en la cuenta de otro.
func TestNoSePuedeEscribirEnLaCuentaDeOtro(t *testing.T) {
	al := nuevoAlmacenFalso()
	_ = al.Responder(context.Background(), "jefa", "alfa.q.nombre", Si)
	s, _ := superficie(t, corpusDemo(), conGuardado(al, "becario"))

	// El envio intenta decir de quien es. La superficie no lee ese campo.
	w := enviar(t, s, url.Values{
		CampoCSRF: {"tok3n"}, "accion": {"no"}, "pregunta": {"alfa.q.nombre"},
		"usuario": {"jefa"}, "quien": {"jefa"}, "sujeto": {"jefa"}})
	if w.Code != http.StatusSeeOther {
		t.Fatalf("contesta %d", w.Code)
	}
	if al.tiene("jefa")["alfa.q.nombre"] != Si {
		t.Errorf("el envio del becario ha pisado la respuesta de la jefa: %v", al.tiene("jefa"))
	}
	if al.tiene("becario")["alfa.q.nombre"] != No {
		t.Errorf("la respuesta del becario no se ha guardado en SU cuenta: %v",
			al.tiene("becario"))
	}
	// Y lo que ve el becario es lo suyo, no lo de la jefa.
	_, cuerpo := pedir(t, s, "/alcance")
	exige(t, cuerpo, rotulo("es", "alcance.pregunta.respondida_no"))
}

// UNA PREGUNTA QUE EL CORPUS NO DECLARA NO ENTRA EN EL ALMACEN.
//
// El emparejamiento entre el envio y lo que existe casa por el ID DE PREGUNTA,
// y quien decide que ids hay es el corpus instalado, no la peticion. Sin esto,
// un envio fabricado engorda el fichero de la cuenta para siempre con
// identificadores que ninguna obligacion nombra.
func TestUnaPreguntaQueElCorpusNoDeclaraNoSeGuarda(t *testing.T) {
	al := nuevoAlmacenFalso()
	s, _ := superficie(t, corpusDemo(), conGuardado(al, "ciso"))
	for _, id := range []string{"inventada.q.loquesea", "", "   ", "../../etc/passwd"} {
		w := enviar(t, s, url.Values{
			CampoCSRF: {"tok3n"}, "accion": {"si"}, "pregunta": {id}})
		if w.Code != http.StatusBadRequest {
			t.Errorf("guardar la pregunta %q contesta %d y tenia que ser 400", id, w.Code)
		}
	}
	// Y la que falta del todo tambien: el campo es OBLIGATORIO en esta accion.
	if w := enviar(t, s, url.Values{CampoCSRF: {"tok3n"}, "accion": {"si"}}); w.Code != http.StatusBadRequest {
		t.Errorf("guardar sin pregunta contesta %d y tenia que ser 400", w.Code)
	}
	if al.escrituraCuenta() != 0 {
		t.Errorf("se ha escrito %d veces con preguntas que el corpus no declara",
			al.escrituraCuenta())
	}
}

// LAS TRES FORMAS DE LA NADA EN EL CAMPO `accion`, que es OBLIGATORIO: ausente,
// presente y vacio, y presente y no interpretable. Las tres son un error, y
// ninguna es «responder que si».
func TestUnaAccionQueNoSeEntiendeNoEscribeNada(t *testing.T) {
	al := nuevoAlmacenFalso()
	s, _ := superficie(t, corpusDemo(), conGuardado(al, "ciso"))
	casos := []struct {
		nombre string
		campos url.Values
	}{
		{"ausente", url.Values{CampoCSRF: {"tok3n"}, "pregunta": {"alfa.q.categoria"}}},
		{"presente y vacia", url.Values{CampoCSRF: {"tok3n"}, "accion": {""},
			"pregunta": {"alfa.q.categoria"}}},
		{"no interpretable", url.Values{CampoCSRF: {"tok3n"}, "accion": {"borralo-todo"},
			"pregunta": {"alfa.q.categoria"}}},
		{"repetida", url.Values{CampoCSRF: {"tok3n"}, "accion": {"si", "limpiar"},
			"pregunta": {"alfa.q.categoria"}}},
		{"con espacios", url.Values{CampoCSRF: {"tok3n"}, "accion": {" si "},
			"pregunta": {"alfa.q.categoria"}}},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if w := enviar(t, s, c.campos); w.Code != http.StatusBadRequest {
				t.Errorf("una accion %s contesta %d y tenia que ser 400: tomarla por el valor "+
					"por defecto es escribir algo que nadie pidio", c.nombre, w.Code)
			}
		})
	}
	if al.escrituraCuenta() != 0 {
		t.Errorf("se ha escrito %d veces con acciones que no se entienden", al.escrituraCuenta())
	}
}

// El control POSITIVO de la rama de arriba: las cinco acciones del vocabulario
// SI se atienden. Sin esto, un guardado que rechazara todo pasaria el test
// anterior entero.
func TestLasCincoAccionesDelVocabularioSeAtienden(t *testing.T) {
	al := nuevoAlmacenFalso()
	s, _ := superficie(t, corpusDemo(), conGuardado(al, "ciso"))
	casos := []url.Values{
		{CampoCSRF: {"tok3n"}, "accion": {"si"}, "pregunta": {"alfa.q.categoria"}},
		{CampoCSRF: {"tok3n"}, "accion": {"no"}, "pregunta": {"alfa.q.categoria"}},
		{CampoCSRF: {"tok3n"}, "accion": {"olvidar"}, "pregunta": {"alfa.q.categoria"}},
		{CampoCSRF: {"tok3n"}, "accion": {"adoptar"}, ParamSi: {"alfa.q.nombre"}},
		{CampoCSRF: {"tok3n"}, "accion": {"limpiar"}},
	}
	for i, c := range casos {
		if w := enviar(t, s, c); w.Code != http.StatusSeeOther {
			t.Errorf("la accion %q (caso %d) contesta %d y tenia que redirigir",
				c.Get("accion"), i, w.Code)
		}
	}
	if al.escrituraCuenta() != len(casos) {
		t.Errorf("se han atendido %d de %d acciones", al.escrituraCuenta(), len(casos))
	}
}

// ADOPTAR GUARDA LO QUE LA PAGINA ENSENA, Y NADA MAS. Lo que el envio traiga y
// el corpus no declare se queda fuera, y una contradictoria (la misma pregunta
// que si y que no) no se guarda de ninguna de las dos formas: elegir una en
// silencio seria afirmar un alcance que nadie afirmo.
func TestAdoptarNoGuardaNiLoDesconocidoNiLoContradictorio(t *testing.T) {
	al := nuevoAlmacenFalso()
	s, _ := superficie(t, corpusDemo(), conGuardado(al, "ciso"))
	w := enviar(t, s, url.Values{
		CampoCSRF: {"tok3n"}, "accion": {"adoptar"},
		ParamSi: {"alfa.q.categoria", "alfa.q.nombre", "inventada.q.x"},
		ParamNo: {"alfa.q.nombre"},
	})
	if w.Code != http.StatusSeeOther {
		t.Fatalf("adoptar contesta %d", w.Code)
	}
	guardado := al.tiene("ciso")
	if guardado["alfa.q.categoria"] != Si {
		t.Errorf("no se ha guardado la respuesta buena: %v", guardado)
	}
	if _, hay := guardado["inventada.q.x"]; hay {
		t.Errorf("se ha guardado una pregunta que el corpus no declara: %v", guardado)
	}
	if _, hay := guardado["alfa.q.nombre"]; hay {
		t.Errorf("se ha guardado una pregunta contradictoria (si y no a la vez) eligiendo una "+
			"de las dos en silencio: %v", guardado)
	}
}

// ---------------------------------------------------------------------------
// El almacen que falla NO se degrada a «no has contestado nada»
// ---------------------------------------------------------------------------

func TestUnAlmacenQueNoSeLeeNoSeConvierteEnUnaEntrevistaEnBlanco(t *testing.T) {
	al := nuevoAlmacenFalso()
	al.falla = errors.New("el disco dice que no")
	var visto []error
	s, _ := superficie(t, corpusDemo(), conGuardado(al, "ciso"), func(o *Opciones) {
		o.AlFallar = func(e error) { visto = append(visto, e) }
	})
	w, cuerpo := pedir(t, s, "/alcance")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("con el almacen roto la pagina contesta %d y tenia que ser 500: ensenar la "+
			"entrevista en blanco le dice a quien la respondio que su trabajo no existio",
			w.Code)
	}
	exige(t, cuerpo, rotulo("es", "error.alcance_ilegible"))
	if len(visto) == 0 {
		t.Error("el fallo del almacen no ha llegado a AlFallar, asi que el operador no se " +
			"entera de que su fichero no se lee")
	}
}

func TestUnAlmacenQueNoEscribeNoDiceQueHaGuardado(t *testing.T) {
	al := nuevoAlmacenFalso()
	al.falla = errors.New("disco lleno")
	s, _ := superficie(t, corpusDemo(), conGuardado(al, "ciso"))
	w := enviar(t, s, url.Values{
		CampoCSRF: {"tok3n"}, "accion": {"si"}, "pregunta": {"alfa.q.categoria"}})
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("con el almacen roto, guardar contesta %d y tenia que ser 500: una "+
			"redireccion aqui lleva a la pantalla de siempre y quien la mire dara por hecho "+
			"que su respuesta esta dentro", w.Code)
	}
	if !strings.Contains(w.Body.String(), rotulo("es", "error.no_se_ha_guardado")) {
		t.Error("la pagina de fallo no dice que NO se ha guardado")
	}
}

// ---------------------------------------------------------------------------
// El enlace compartido sigue funcionando y NO se cuenta como guardado
// ---------------------------------------------------------------------------

func TestUnEnlaceConRespuestasNoSeCuentaComoGuardado(t *testing.T) {
	al := nuevoAlmacenFalso()
	_ = al.Responder(context.Background(), "ciso", "alfa.q.nombre", Si)
	s, _ := superficie(t, corpusDemo(), conGuardado(al, "ciso"))

	_, cuerpo := pedir(t, s, "/alcance?"+ParamNo+"=alfa.q.categoria")
	exige(t, cuerpo,
		rotulo("es", "alcance.guardado.desde_enlace"), // esto viene del enlace
		rotulo("es", "alcance.guardado.adoptar"),      // y se puede adoptar
		rotulo("es", "alcance.pregunta.respondida_no"),
	)
	prohibe(t, cuerpo, rotulo("es", "alcance.guardado.en_tu_cuenta"))
	// Y ABRIR EL ENLACE NO ESCRIBE NADA: la cuenta sigue con lo suyo.
	if al.escrituraCuenta() != 1 { // la del montaje
		t.Errorf("abrir un enlace con respuestas ha escrito en la cuenta (%d escrituras). Un "+
			"GET que muta lo dispara cualquier cosa que precargue enlaces",
			al.escrituraCuenta())
	}
	if al.tiene("ciso")["alfa.q.nombre"] != Si {
		t.Errorf("abrir el enlace se ha llevado por delante lo guardado: %v", al.tiene("ciso"))
	}
	// El formulario de adopcion lleva DENTRO lo que la pagina ensena.
	if !strings.Contains(cuerpo, `name="no" value="alfa.q.categoria"`) {
		t.Error("el formulario de adopcion no lleva las respuestas del enlace dentro, asi que " +
			"el boton guardaria otra cosa distinta de la que se esta viendo")
	}
}

// ---------------------------------------------------------------------------
// Lo guardado que ya no existe se DICE, no se descarta en silencio
// ---------------------------------------------------------------------------

// Es la direccion contraria del emparejamiento (invariante 7): la direccion
// corpus -> almacen decide que se pinta, y esta, almacen -> corpus, dice lo que
// sobra. Sin ella, desinstalar un paquete hace desaparecer respuestas del
// recuento sin una linea en ningun sitio.
func TestUnaRespuestaGuardadaQueYaNoTienePreguntaSeDice(t *testing.T) {
	al := nuevoAlmacenFalso()
	_ = al.Responder(context.Background(), "ciso", "alfa.q.categoria", Si)
	_ = al.Responder(context.Background(), "ciso", "de.un.paquete.desinstalado", No)
	s, _ := superficie(t, corpusDemo(), conGuardado(al, "ciso"))

	_, cuerpo := pedir(t, s, "/alcance")
	exige(t, cuerpo, rotulo("es", "alcance.guardado.huerfanas"))
	// Y NO se pinta como una pregunta: lo que el corpus no declara no entra en
	// el estado de la pantalla, que es lo que `De` ya garantizaba.
	if strings.Contains(cuerpo, "de.un.paquete.desinstalado") {
		t.Error("una respuesta guardada cuya pregunta ya no existe se esta pintando como " +
			"pregunta de la entrevista")
	}
}

// El control negativo del cardinal: sin huerfanas, el aviso NO sale. Un aviso
// que sale siempre es un aviso que nadie lee.
func TestSinHuerfanasNoSaleElAviso(t *testing.T) {
	al := nuevoAlmacenFalso()
	_ = al.Responder(context.Background(), "ciso", "alfa.q.categoria", Si)
	s, _ := superficie(t, corpusDemo(), conGuardado(al, "ciso"))
	_, cuerpo := pedir(t, s, "/alcance")
	prohibe(t, cuerpo, rotulo("es", "alcance.guardado.huerfanas"))
}

// ---------------------------------------------------------------------------
// La construccion: medio guardado se rechaza
// ---------------------------------------------------------------------------

func TestMedioGuardadoNoSeConstruye(t *testing.T) {
	al := nuevoAlmacenFalso()
	casos := []struct {
		nombre string
		o      Opciones
	}{
		{"almacen sin Quien", Opciones{Alcances: al,
			Tokens: func(*http.Request) (string, error) { return "t", nil }}},
		{"almacen sin Tokens", Opciones{Alcances: al,
			Quien: func(*http.Request) string { return "ciso" }}},
		{"almacen solo", Opciones{Alcances: al}},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			o := c.o
			o.Catalogo = nuevoCatalogo()
			if _, err := Nuevo(o); !errors.Is(err, ErrPersistencia) {
				t.Fatalf("%s tenia que rechazarse al construir y da %v.\n"+
					"  Medio guardado escribe en un cajon sin dueno, o pinta botones que "+
					"contestan 403", c.nombre, err)
			}
		})
	}
	// Y el control positivo: Quien y Tokens SIN almacen se admiten. Son utiles
	// por si solos, y sin almacen no se registra ninguna ruta que mute.
	o := Opciones{Catalogo: nuevoCatalogo(),
		Quien:  func(*http.Request) string { return "ciso" },
		Tokens: func(*http.Request) (string, error) { return "t", nil }}
	if _, err := Nuevo(o); err != nil {
		t.Fatalf("Quien y Tokens sin almacen tenian que admitirse: %v", err)
	}
}

// ---------------------------------------------------------------------------
// La ruta que muta: existe SOLO con almacen, y es exactamente una
// ---------------------------------------------------------------------------

// TestLaUnicaRutaQueMutaEsLaDelGuardado es el censo de rutas mutantes de esta
// superficie, y se cruza CON EL REGISTRO en los dos sentidos.
//
// Va aparte de TestNingunaRutaDeLaSuperficieMuta, que sigue existiendo y sigue
// valiendo: aquel prueba la superficie SIN almacen, que es como se sirve fuera
// de `plazum serve`, y sigue siendo GET de arriba abajo. Este prueba la de con
// almacen, y lo que exige es que la lista de lo que muta sea EXACTAMENTE la
// escrita aqui: una ruta mutante nueva que nadie declare se pone roja el mismo
// dia que se escribe.
func TestLaUnicaRutaQueMutaEsLaDelGuardado(t *testing.T) {
	mutantesDeclaradas := map[string]string{
		"POST /alcance": "guarda las respuestas de la entrevista en la cuenta de quien " +
			"ha entrado. Ver persistencia.go",
	}

	al := nuevoAlmacenFalso()
	s, _ := superficie(t, corpusDemo(), conGuardado(al, "ciso"))
	hay := map[string]bool{}
	for _, p := range s.Patrones() {
		metodo, _, ok := strings.Cut(p, " ")
		if !ok {
			t.Errorf("el patron %q no declara metodo, asi que acepta TODOS", p)
			continue
		}
		if metodo == http.MethodGet || metodo == http.MethodHead {
			continue
		}
		hay[p] = true
		if _, declarada := mutantesDeclaradas[p]; !declarada {
			t.Errorf("la ruta %q muta y el censo de este test no la declara.\n"+
				"  Toda ruta que escribe tiene que pasar por el middleware de CSRF de quien "+
				"monta, y decirse aqui con su motivo.", p)
		}
	}
	for p := range mutantesDeclaradas {
		if !hay[p] {
			t.Errorf("el censo declara la ruta mutante %q y el registro ya no la tiene: este "+
				"test se ha quedado viejo", p)
		}
	}
	// Y sin almacen NO hay ninguna: la superficie vuelve a ser GET-only.
	sinAlmacen, _ := superficie(t, corpusDemo())
	for _, p := range sinAlmacen.Patrones() {
		if metodo, _, _ := strings.Cut(p, " "); metodo != http.MethodGet && metodo != http.MethodHead {
			t.Errorf("sin almacen de alcances la superficie registra %q, que muta. Sin donde "+
				"guardar, esa ruta solo puede contestar un error", p)
		}
	}
}
