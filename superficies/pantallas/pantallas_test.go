package pantallas

import (
	"errors"
	"fmt"
	"github.com/marcosmatalab/plazum/superficies/camino"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/pantalla"
)

// ---------------------------------------------------------------------------
// Las seis pantallas
// ---------------------------------------------------------------------------

// Las seis existen, responden y estan las seis en el menu de todas ellas.
//
// Lo del menu no es cosmetico: una pantalla que desaparece cuando no tiene
// datos deja al operador sin saber que existia, y eso no se arregla con
// documentacion porque nadie la lee al instalar.
func TestLasSeisPantallasRespondenYSiguenEnElMenu(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	seis := []pantalla.ID{pantalla.Alcance, pantalla.Hoy, pantalla.Controles,
		pantalla.Certificados, pantalla.Personas, pantalla.Estado}
	for _, id := range seis {
		w, cuerpo := pedir(t, s, "/"+string(id))
		if w.Code != http.StatusOK {
			t.Fatalf("GET /%s dio %d y esperaba 200", id, w.Code)
		}
		if tipo := w.Header().Get("Content-Type"); !strings.HasPrefix(tipo, "text/html") {
			t.Errorf("GET /%s sirve %q y esperaba text/html", id, tipo)
		}
		// El titulo de la pantalla, por su clave de catalogo.
		exige(t, cuerpo, rotulo("es", "pantalla."+string(id)+".titulo"))
		// Y las SEIS entradas de menu, esten llenas o vacias.
		for _, otra := range seis {
			exige(t, cuerpo, rotulo("es", "pantalla."+string(otra)+".titulo"))
		}
	}
}

// La raiz lleva a la primera pantalla en vez de dar un 404 o una pagina en
// blanco: quien acaba de instalar esto escribe la direccion del servidor y ya.
func TestLaRaizLlevaAAlcance(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	w, _ := pedir(t, s, "/")
	if w.Code != http.StatusSeeOther {
		t.Fatalf("GET / dio %d y esperaba 303", w.Code)
	}
	if destino := w.Header().Get("Location"); destino != "/alcance" {
		t.Errorf("GET / redirige a %q y esperaba /alcance", destino)
	}
}

// Las pantallas que salen del estado se pintan VACIAS PERO CON su explicacion,
// y ademas distinguen "no hay corpus" de "no hay estado", que son dos problemas
// con dos arreglos distintos.
func TestLasPantallasSinContenidoExplicanPorQueEstanVacias(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	for _, id := range []pantalla.ID{pantalla.Hoy, pantalla.Personas, pantalla.Estado} {
		w, cuerpo := pedir(t, s, "/"+string(id))
		if w.Code != http.StatusOK {
			t.Fatalf("GET /%s dio %d", id, w.Code)
		}
		exige(t, cuerpo,
			rotulo("es", "pantalla."+string(id)+".vacia"), // por que esta vacia
			rotulo("es", "origen.estado"),                 // de donde saldria
			rotulo("es", "vacia.que_hacer"),               // que hago yo ahora
		)
		prohibe(t, cuerpo, rotulo("es", "origen.corpus"))
		prohibe(t, cuerpo, rotulo("es", "vacia.sin_explicacion"))
	}
}

// Sin corpus instalado, las seis siguen ahi y las tres que salen del corpus
// dicen que lo que falta es instalar corpus, no completar el alcance.
func TestSinCorpusLasSeisSiguenYDicenQueFaltaCorpus(t *testing.T) {
	s, _ := superficie(t, nil)
	for _, id := range []pantalla.ID{pantalla.Alcance, pantalla.Controles, pantalla.Certificados} {
		w, cuerpo := pedir(t, s, "/"+string(id))
		if w.Code != http.StatusOK {
			t.Fatalf("GET /%s sin corpus dio %d", id, w.Code)
		}
		exige(t, cuerpo, rotulo("es", "pantalla."+string(id)+".sin_corpus"))
	}
}

// ---------------------------------------------------------------------------
// La derivacion a un clic
// ---------------------------------------------------------------------------

// enlaceDe saca de la pagina el href del enlace que sigue a un ancla dada.
var reEnlace = regexp.MustCompile(`href="([^"]*)"`)

// enlacesDePregunta devuelve los href de si y no de una pregunta, buscando en
// el bloque de esa pregunta.
func enlacesDePregunta(t *testing.T, cuerpo, id string) (si, no string) {
	t.Helper()
	inicio := strings.Index(cuerpo, `id="p-`+id+`"`)
	if inicio < 0 {
		t.Fatalf("la pagina no trae la pregunta %q", id)
	}
	fin := strings.Index(cuerpo[inicio:], "</li>")
	if fin < 0 {
		t.Fatalf("el bloque de la pregunta %q no cierra", id)
	}
	bloque := cuerpo[inicio : inicio+fin]
	for _, m := range reEnlace.FindAllStringSubmatch(bloque, -1) {
		u := strings.ReplaceAll(m[1], "&amp;", "&")
		v, err := url.Parse(u)
		if err != nil {
			continue
		}
		q := v.Query()
		for _, x := range q[ParamSi] {
			if x == id && si == "" {
				si = u
			}
		}
		for _, x := range q[ParamNo] {
			if x == id && no == "" {
				no = u
			}
		}
	}
	if si == "" || no == "" {
		t.Fatalf("la pregunta %q no ofrece si (%q) y no (%q)", id, si, no)
	}
	return si, no
}

// LA casilla. Desde Alcance, un clic en una respuesta ensena inmediatamente que
// obligaciones alcanzan al sujeto y POR QUE, sin pasar por ninguna pantalla de
// configuracion.
//
// Se comprueban las dos direcciones. Solo la primera pasaria con una derivacion
// que dijera "aplica" a todo.
func TestLaDerivacionAUnClicMueveObligacionesYDiceElPorQue(t *testing.T) {
	s, _ := superficie(t, corpusDemo())

	_, inicio := pedir(t, s, "/alcance")
	// De partida, la obligacion condicionada NO puede estar entre las que
	// aplican: nadie ha respondido nada.
	if seccionAplican(inicio) != "" && strings.Contains(seccionAplican(inicio), "alfa.o.auditoria") {
		t.Fatal("sin responder nada, una obligacion condicionada aparece como aplicable. " +
			"Eso es afirmar alcance sin datos, que es el fallo mas caro que cabe aqui")
	}

	si, no := enlacesDePregunta(t, inicio, "alfa.q.categoria")

	// El clic no sale de Alcance: no hay pantalla de configuracion en medio.
	if u, _ := url.Parse(si); u.Path != "/alcance" {
		t.Errorf("responder lleva a %q y tiene que quedarse en /alcance: la derivacion es "+
			"a un clic, no a un clic y un formulario", u.Path)
	}

	// Un clic en "si": la obligacion pasa a aplicar y la pagina dice por que,
	// con la cita del articulo del que sale la pregunta.
	w, conSi := pedir(t, s, si)
	if w.Code != http.StatusOK {
		t.Fatalf("seguir el enlace de respuesta dio %d", w.Code)
	}
	aplican := seccionAplican(conSi)
	if !strings.Contains(aplican, "alfa.o.auditoria") {
		t.Errorf("tras responder que si, la obligacion condicionada no aparece entre las "+
			"que aplican.\n--- seccion ---\n%s", aplican)
	}
	exige(t, conSi,
		rotulo("es", "derivacion.respondiste_si"), // el tipo de motivo
		"demo alfa art. 3",                        // la cita, del corpus, tal cual
		"Que categoria tiene el sistema",          // el texto de la pregunta, tal cual
	)

	// Un clic en "no": la misma obligacion pasa a no aplicar, y desaparece de
	// las que aplican.
	_, conNo := pedir(t, s, no)
	if strings.Contains(seccionAplican(conNo), "alfa.o.auditoria") {
		t.Error("tras responder que no, la obligacion sigue entre las que aplican. " +
			"Una derivacion que solo suma no deriva nada")
	}
	_, controles := pedir(t, s, "/controles?"+strings.SplitN(no, "?", 2)[1])
	exige(t, controles, rotulo("es", "derivacion.respondiste_no"), rotulo("es", "estado.no_aplica"))
}

// seccionAplican recorta el bloque de "las que aplican" del panel de Alcance.
func seccionAplican(cuerpo string) string {
	i := strings.Index(cuerpo, rotulo("es", "alcance.derivacion.aplican"))
	if i < 0 {
		return ""
	}
	j := strings.Index(cuerpo[i:], "</ul>")
	if j < 0 {
		return cuerpo[i:]
	}
	return cuerpo[i : i+j]
}

// Una obligacion sin preguntas alcanza a todo el mundo desde el primer momento,
// y lo dice. Es la que evita el "esto esta vacio, no me aplica nada" del primer
// minuto.
func TestUnaObligacionSinCondicionesAplicaDesdeElPrimerMomento(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	_, cuerpo := pedir(t, s, "/alcance")
	exige(t, cuerpo, "beta.o.notificacion", rotulo("es", "derivacion.sin_condiciones"))
}

// Las preguntas salen ordenadas por cuantas obligaciones desbloquea cada una, y
// la primera sin responder viene marcada como la siguiente. Es lo que responde
// "¿por donde empiezo?" sin documentacion.
func TestLaPrimeraPreguntaEsLaQueMasDesbloqueaYVieneSugerida(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	_, cuerpo := pedir(t, s, "/alcance")

	orden := ordenDePreguntas(cuerpo)
	quiero := []string{}
	for _, q := range pantalla.Derivar(corpusDemo())[0].Preguntas {
		quiero = append(quiero, q.ID)
	}
	if !reflect.DeepEqual(orden, quiero) {
		t.Errorf("las preguntas salen en %v y el modelo las ordena %v. El orden lo fija "+
			"nucleo/pantalla por obligaciones desbloqueadas y aqui solo se pinta", orden, quiero)
	}
	if len(orden) == 0 {
		t.Fatal("no hay preguntas en la pagina")
	}
	// La sugerida es la primera sin responder.
	i := strings.Index(cuerpo, `id="p-`+orden[0]+`"`)
	j := strings.Index(cuerpo, rotulo("es", "alcance.siguiente"))
	if j < i || j > i+800 {
		t.Errorf("la marca de siguiente pregunta no esta en la primera sin responder")
	}
	// Y al responder la primera, la sugerencia se mueve a la segunda.
	si, _ := enlacesDePregunta(t, cuerpo, orden[0])
	_, despues := pedir(t, s, si)
	i2 := strings.Index(despues, `id="p-`+orden[1]+`"`)
	j2 := strings.Index(despues, rotulo("es", "alcance.siguiente"))
	if j2 < i2 {
		t.Error("tras responder la primera, la sugerencia no se ha movido a la siguiente")
	}
}

var reIDPregunta = regexp.MustCompile(`id="p-([^"]+)"`)

func ordenDePreguntas(cuerpo string) []string {
	var out []string
	for _, m := range reIDPregunta.FindAllStringSubmatch(cuerpo, -1) {
		out = append(out, m[1])
	}
	return out
}

// El estado de la entrevista viaja en la direccion, asi que la pagina se puede
// compartir: dos peticiones con la misma direccion dan exactamente la misma
// pagina, y una direccion con las respuestas puestas se abre con el alcance ya
// derivado sin haber tocado nada.
func TestElAlcanceViajaEnLaDireccionYSePuedeCompartir(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	destino := "/alcance?si=alfa.q.categoria&no=beta.q.riesgo"
	_, uno := pedir(t, s, destino)
	_, dos := pedir(t, s, destino)
	if uno != dos {
		t.Fatal("la misma direccion da dos paginas distintas: la superficie no es determinista")
	}
	exige(t, uno, rotulo("es", "derivacion.respondiste_si"))
	// Y el enlace a Controles se lleva las respuestas puestas.
	if !strings.Contains(uno, "/controles?no=beta.q.riesgo&amp;si=alfa.q.categoria") {
		t.Error("el enlace a Controles no arrastra las respuestas, asi que pasar de una " +
			"pantalla a otra perderia el alcance que se acaba de responder")
	}
}

// ---------------------------------------------------------------------------
// La UI generada desde el corpus
// ---------------------------------------------------------------------------

// LA segunda casilla, y la propiedad de frente del producto: se instala un
// paquete que este codigo no ha visto nunca, escrito solo como datos, y la
// interfaz cambia sola. Cero ficheros Go tocados, cero plantillas tocadas.
func TestUnPaqueteNuevoCambiaLaInterfazSinTocarCodigo(t *testing.T) {
	nuevo := &corpus.Paquete{
		URN: "urn:demo:gamma", Version: "1", Clase: corpus.Propio,
		Entidades: []corpus.TipoEntidad{{Nombre: "proveedor", Atributos: []corpus.Atributo{
			{Nombre: "critico", Tipo: corpus.Booleano, Obligado: true,
				Ayuda: "ayuda que solo existe en este paquete",
				Cita:  "demo gamma art. 7"}}}},
		Preguntas: []corpus.Pregunta{{ID: "gamma.q.critico",
			Texto: "Tienes proveedores criticos", Cita: "demo gamma art. 7",
			Entidad: "proveedor", Atributo: "critico",
			Desbloquea: []string{"gamma.o.contrato"}}},
		Obligaciones: []corpus.Obligacion{{ID: "gamma.o.contrato", Articulo: "7",
			Cita: "demo gamma art. 7", ClaseE2E: "documental",
			Preguntas: []string{"gamma.q.critico"}}},
	}

	s, _ := superficie(t, corpusDemo())
	_, antes := pedir(t, s, "/alcance")
	prohibe(t, antes, "Tienes proveedores criticos", "demo gamma art. 7")

	s.Recargar(append(corpusDemo(), nuevo))
	_, despues := pedir(t, s, "/alcance")
	exige(t, despues,
		"Tienes proveedores criticos",           // la pregunta, del corpus
		"ayuda que solo existe en este paquete", // la ayuda, del corpus
		"demo gamma art. 7",                     // la cita, del corpus
		"urn:demo:gamma",                        // quien lo pide
		"critico",                               // el campo derivado del esquema
	)
	_, controles := pedir(t, s, "/controles")
	exige(t, controles, "gamma.o.contrato")
}

// Los campos del formulario salen de corpus.EsquemaUI, con el dato preguntado
// una sola vez y diciendo QUE PAQUETES lo piden. Es lo que convierte "rellena
// esto" en "esto lo piden estas dos normas".
func TestElFormularioSaleDelEsquemaYDiceQuienPideCadaDato(t *testing.T) {
	// Dos paquetes que piden el MISMO atributo: se pregunta una vez.
	otro := paqueteBeta()
	otro.Entidades = append(otro.Entidades, corpus.TipoEntidad{
		Nombre: "sistema", Atributos: []corpus.Atributo{
			{Nombre: "categoria", Tipo: corpus.Enumerado,
				Valores: []string{"BAJA", "MEDIA", "ALTA"}, Cita: "demo beta art. 5"}}})

	s, _ := superficie(t, []*corpus.Paquete{paqueteAlfa(), otro})
	_, cuerpo := pedir(t, s, "/alcance")

	if n := strings.Count(cuerpo, `<span class="etiqueta">categoria</span>`); n != 1 {
		t.Errorf("el atributo categoria sale %d veces en el formulario y tiene que salir "+
			"una: dos normas que piden el mismo dato lo preguntan una vez", n)
	}
	// Y se dice que los dos paquetes lo piden.
	i := strings.Index(cuerpo, `<span class="etiqueta">categoria</span>`)
	bloque := cuerpo[i:min(i+900, len(cuerpo))]
	exige(t, bloque, "urn:demo:alfa", "urn:demo:beta", rotulo("es", "alcance.campos.lo_piden"))
	// Los valores del enumerado tambien salen del corpus.
	exige(t, cuerpo, "BAJA", "MEDIA", "ALTA")
}

// El modelo no depende del orden en que llegan los paquetes, y la pagina
// tampoco: el cargador recorre un directorio y ese orden no esta garantizado.
func TestLaPaginaNoDependeDelOrdenDeLosPaquetes(t *testing.T) {
	directo, _ := superficie(t, []*corpus.Paquete{paqueteAlfa(), paqueteBeta()})
	alreves, _ := superficie(t, []*corpus.Paquete{paqueteBeta(), paqueteAlfa()})
	for _, ruta := range []string{"/alcance", "/controles", "/certificados"} {
		_, uno := pedir(t, directo, ruta)
		_, dos := pedir(t, alreves, ruta)
		if uno != dos {
			t.Errorf("%s cambia si los paquetes llegan en otro orden", ruta)
		}
	}
}

// ---------------------------------------------------------------------------
// Certificados: se lee contra obligaciones, no contra preguntas
// ---------------------------------------------------------------------------

// En Certificados, Fila.Requiere son IDs de OBLIGACION y no de pregunta. Si se
// leyera igual que Controles, todos los entregables saldrian pendientes para
// siempre porque ninguna de esas "preguntas" existe.
func TestCertificadosSeEvaluaContraLasObligacionesQueLoPiden(t *testing.T) {
	s, _ := superficie(t, corpusDemo())

	// Sin responder: la plantilla que piden dos obligaciones esta pendiente.
	_, sinResponder := pedir(t, s, "/certificados")
	exige(t, sinResponder, "alfa.pl.informe", rotulo("es", "estado.pendiente"))
	prohibe(t, sinResponder, rotulo("es", "derivacion.pregunta_desconocida"))

	// Respondiendo que si a la pregunta que desbloquea una de ellas, la
	// plantilla pasa a hacer falta.
	_, conSi := pedir(t, s, "/certificados?si=alfa.q.categoria")
	i := strings.Index(conSi, "alfa.pl.informe")
	if i < 0 {
		t.Fatal("no esta la plantilla en la tabla")
	}
	fila := conFilaDesde(conSi, i)
	exige(t, fila, rotulo("es", "estado.aplica"), rotulo("es", "derivacion.lo_pide_y_aplica"))

	// Y la plantilla que ninguna obligacion pide se ensena como huerfana en
	// vez de pedirle al operador que rellene papeleo sin motivo.
	exige(t, sinResponder, "beta.pl.huerfana", rotulo("es", "derivacion.entregable_huerfano"))
}

func conFilaDesde(cuerpo string, i int) string {
	inicio := strings.LastIndex(cuerpo[:i], "<tr")
	if inicio < 0 {
		inicio = i
	}
	fin := strings.Index(cuerpo[inicio:], "</tr>")
	if fin < 0 {
		return cuerpo[inicio:]
	}
	return cuerpo[inicio : inicio+fin]
}

// ---------------------------------------------------------------------------
// El catalogo: los rotulos son claves, y la lista de claves esta completa
// ---------------------------------------------------------------------------

// El contrato con quien escriba el catalogo: estas claves, todas, o hay huecos.
// Se comprueba en las DOS direcciones barriendo la superficie entera.
//
// Una sola direccion no sirve: si solo se comprobara que lo pedido esta en la
// lista, la lista podria tener cien claves inventadas; si solo se comprobara lo
// contrario, la lista podria estar vacia.
func TestLasClavesDeCatalogoSonExactamenteLasQueLaInterfazPide(t *testing.T) {
	pedidas := map[string]int{}
	barrer := func(ps []*corpus.Paquete, rutas ...string) {
		s, cat := superficie(t, ps)
		for _, ruta := range rutas {
			pedir(t, s, ruta)
		}
		for k, v := range cat.vistas() {
			pedidas[k] += v
		}
	}
	// Corpus normal, en todos los estados de la entrevista que existen.
	barrer(corpusDemo(),
		"/alcance", "/hoy", "/controles", "/certificados", "/personas", "/estado",
		"/alcance?si=alfa.q.categoria&si=alfa.q.nombre&si=beta.q.riesgo",
		"/alcance?no=alfa.q.categoria",
		"/alcance?si=alfa.q.categoria&no=alfa.q.categoria", // contradictoria
		"/controles?f=aplica", "/controles?f=pendiente", "/controles?f=no_aplica",
		"/controles?si=alfa.q.categoria&no=alfa.q.categoria",
		"/controles?no=alfa.q.categoria",
		"/certificados?si=alfa.q.categoria",
		"/certificados?no=alfa.q.categoria&no=alfa.q.nombre", // entregable que no hace falta
		"/no-existe",                          // 404
		"/alcance?"+strings.Repeat("x", 9000), // 414
	)
	// Corpus vacio: las claves de "no hay corpus instalado". /hoy entra aqui
	// porque es donde el panel de inicio pinta SIN DATO en vez de un cero, y
	// esa rama no la alcanza ningun otro barrido.
	barrer(nil, "/alcance", "/controles", "/certificados", "/hoy")
	// Un corpus con un vencimiento YA PASADO: es el control positivo de la
	// cifra de "sin constancia" y de su descargo. Sin el, la rama que escribe
	// esa fila no la recorre nadie.
	barrer([]*corpus.Paquete{paqueteVencido()}, "/hoy")
	// Corpus con una obligacion condicionada a una pregunta que no existe, y
	// un entregable que ninguna obligacion pide.
	barrer([]*corpus.Paquete{paqueteRoto()}, "/alcance", "/controles", "/certificados")
	// Un corpus con tantas filas que hay paginacion, y con tantas aplicables
	// que el panel de Alcance tiene que recortar.
	barrer([]*corpus.Paquete{paqueteGrande(600)},
		"/controles", "/controles?p=2", "/alcance?si=grande.q.1")
	// Un corpus con obligaciones y sin ninguna pregunta de alcance.
	barrer([]*corpus.Paquete{paqueteSinPreguntas()}, "/alcance")
	// LA REVELACION PROGRESIVA, con sus DOS motivos. Cada uno necesita su
	// entrada: el corpus de demostracion no tiene ninguna pregunta dormida, asi
	// que sin estos dos barridos las seis claves de la familia se declararian y
	// no las pediria nadie, que es como se queda una rama sin traducir hasta que
	// se la encuentra un cliente.
	barrer([]*corpus.Paquete{paqueteConHuerfana()},
		"/alcance",                        // la lista corta, con su cardinal y su enlace
		"/alcance?"+ParamVer+"="+VerTodas, // la larga, con el motivo de cada una
	)
	barrer([]*corpus.Paquete{paqueteQueSeApaga()},
		"/alcance?no=alfa.q.categoria&"+ParamVer+"="+VerTodas)

	// EL GUARDADO, con sus TRES situaciones y sus SEIS errores.
	//
	// Hace falta un barrido propio por lo mismo que la revelacion progresiva:
	// estas once claves solo se piden cuando la superficie se monta CON almacen,
	// y sin este bloque se quedarian declaradas y sin pedir, o pedidas y sin
	// declarar. Las dos direcciones se ven aqui.
	//
	// Y cada situacion necesita SU entrada, porque son ramas excluyentes: la de
	// «esto es lo tuyo» no se alcanza con la direccion respondida, y la de
	// «esto viene de un enlace» no se alcanza sin ella.
	{
		al := nuevoAlmacenFalso()
		_ = al.Responder(t.Context(), "ciso", "alfa.q.categoria", Si)
		_ = al.Responder(t.Context(), "ciso", "ya.no.existe", No) // la huerfana
		s, cat := superficie(t, corpusDemo(), conGuardado(al, "ciso"))
		pedir(t, s, "/alcance")                             // en_tu_cuenta, cuando, huerfanas
		pedir(t, s, "/alcance?"+ParamSi+"=alfa.q.nombre")   // desde_enlace, adoptar
		pedirPost(t, s, url.Values{"accion": {"loquesea"}}) // accion_desconocida
		pedirPost(t, s, url.Values{"accion": {"si"}, "pregunta": {"no.existe"}})
		for k, v := range cat.vistas() {
			pedidas[k] += v
		}
	}
	{
		// SIN SESION: el 403 de la escritura sin autor.
		al := nuevoAlmacenFalso()
		s, cat := superficie(t, corpusDemo(), conGuardado(al, ""))
		pedirPost(t, s, url.Values{"accion": {"limpiar"}})
		for k, v := range cat.vistas() {
			pedidas[k] += v
		}
	}
	{
		// EL ALMACEN ROTO: la lectura que no se degrada y la escritura que no
		// dice haber guardado. Son las dos ramas que no alcanza ningun barrido
		// con un almacen que funciona.
		al := nuevoAlmacenFalso()
		al.falla = errors.New("el disco dice que no")
		s, cat := superficie(t, corpusDemo(), conGuardado(al, "ciso"))
		pedir(t, s, "/alcance")
		pedirPost(t, s, url.Values{"accion": {"si"}, "pregunta": {"alfa.q.categoria"}})
		for k, v := range cat.vistas() {
			pedidas[k] += v
		}
	}
	// El formulario ilegible: un cuerpo que ParseForm no entiende. Se manda a
	// mano porque url.Values no sabe componer uno roto.
	{
		al := nuevoAlmacenFalso()
		s, cat := superficie(t, corpusDemo(), conGuardado(al, "ciso"))
		r := httptest.NewRequest(http.MethodPost, "/alcance", strings.NewReader("%zz=1"))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		s.ServeHTTP(httptest.NewRecorder(), r)
		for k, v := range cat.vistas() {
			pedidas[k] += v
		}
	}

	// LA BARRA LATERAL CON EL CAMINO PUESTO, que es como la monta el producto.
	// Sin este barrido, los rotulos de los pasos y las tres palabras de la tira
	// se quedarian declaradas y sin pedir, o pedidas y sin declarar: las dos
	// direcciones se ven aqui.
	//
	// Se piden /alcance (un paso del camino: marca el actual y lleva las
	// respuestas), /hoy (que NO es paso: no marca ninguno) y un 404, porque la
	// pagina de error tambien pinta la barra.
	{
		s, cat := superficie(t, corpusDemo(), conCamino())
		for _, ruta := range []string{"/alcance", "/alcance?si=alfa.q.categoria",
			"/hoy", "/controles", "/no-existe"} {
			pedir(t, s, ruta)
		}
		for k, v := range cat.vistas() {
			pedidas[k] += v
		}
	}

	// La pantalla Hoy en TODOS los estados del vigilante. Sin esto, el
	// barrido solo alcanza el estado por defecto (el planificador no ha
	// corrido nunca, el latido apagado) y las claves de los demas veredictos
	// se quedan sin traducir hasta que un cliente se las encuentra en crudo
	// el dia que su planificador se para, que es el peor dia posible.
	for _, opt := range vigilancias() {
		s, cat := superficie(t, corpusDemo(), opt)
		pedir(t, s, "/hoy")
		for k, v := range cat.vistas() {
			pedidas[k] += v
		}
	}

	// La clave de "vacia sin explicacion" solo se alcanza si el modelo trae
	// una pantalla vacia sin PorQue, que hoy no puede pasar. Se cubre a mano
	// para no dejarla fuera de la lista ni sin comprobar.
	pedidas["vacia.sin_explicacion"]++

	// UN CAMINO CON UN PASO SIN PANTALLA, y hace falta desde el 03-09-2026.
	//
	// Cuando calendario y escalado ganaron su pantalla, el camino canonico se
	// quedo SIN ningun paso sin pantalla, y la rama del armazon que pinta «por
	// terminal» dejo de tener quien la recorriera. Este test lo dijo: la lista
	// declara ui.paso_por_terminal y nadie la pide.
	//
	// La respuesta NO es quitar la clave. Validar SIGUE aceptando un paso sin
	// pantalla que traiga su orden y la plantilla sigue teniendo su rama: la
	// capacidad esta viva y lo que le faltaba era una entrada. Quitarla dejaria
	// esa rama pintando un hueco el dia que vuelva a haber un paso asi, que es
	// peor que las dos cosas. Es M47 aplicado a un rotulo.
	conPasoSinPantalla := append(append([]camino.Paso(nil), camino.Canonico()...),
		camino.Paso{ID: "sintetico", Titulo: "camino.paso.acta", Verbo: "camino.verbo.acta",
			Comando: "plazum algo --con-sus-banderas"})
	sSint, catSint := superficie(t, corpusDemo(), func(o *Opciones) {
		o.Pasos = conPasoSinPantalla
	})
	pedir(t, sSint, "/hoy")
	for k, v := range catSint.vistas() {
		pedidas[k] += v
	}

	tengo := claves(pedidas)
	quiero := ClavesDeCatalogo()
	if !reflect.DeepEqual(tengo, quiero) {
		faltan, sobran := diferencia(tengo, quiero), diferencia(quiero, tengo)
		t.Errorf("ClavesDeCatalogo() no coincide con lo que la interfaz pide.\n"+
			"  la interfaz pide y la lista no declara: %v\n"+
			"    (esas saldrian como clave cruda en pantalla)\n"+
			"  la lista declara y nadie pide: %v\n"+
			"    (esas obligan a traducir texto que no se ve)", faltan, sobran)
	}
}

func diferencia(a, b []string) []string {
	en := map[string]bool{}
	for _, x := range b {
		en[x] = true
	}
	var out []string
	for _, x := range a {
		if !en[x] {
			out = append(out, x)
		}
	}
	return out
}

// Toda columna que el modelo produzca tiene su rotulo. El dia que
// nucleo/pantalla anada una columna, esto se pone rojo y dice cual, en vez de
// aparecer una cabecera con el nombre crudo en la interfaz de un cliente.
func TestNoHayColumnaSinRotulo(t *testing.T) {
	rotulada := map[string]bool{}
	for _, c := range ClavesDeCatalogo() {
		if resto, ok := strings.CutPrefix(c, "columna."); ok {
			rotulada[resto] = true
		}
	}
	vistas := map[string]bool{}
	for _, p := range pantalla.Derivar(corpusDemo()) {
		for _, f := range p.Filas {
			for k := range f.Columnas {
				vistas[k] = true
			}
		}
	}
	if len(vistas) == 0 {
		t.Fatal("el corpus de pruebas no produce ninguna columna: este test no probaria nada")
	}
	var sin []string
	for k := range vistas {
		if !rotulada[k] {
			sin = append(sin, k)
		}
	}
	sort.Strings(sin)
	if len(sin) > 0 {
		t.Errorf("nucleo/pantalla produce columnas sin rotulo: %v. Arreglo: anadelas a "+
			"columnasEnOrden y escribe su clave columna.<nombre> en el catalogo", sin)
	}
}

// El contenido del corpus NO pasa por el catalogo. Traducir texto transcrito
// del BOE crea obra derivada y se sale de la estratificacion de licencias del
// corpus: un paquete en otro idioma es un paquete distinto con su fuente.
//
// Se comprueba de la unica forma que no admite discusion: apuntando que claves
// se le piden al catalogo y exigiendo que ninguna sea un texto del corpus.
func TestElContenidoDelCorpusNoPasaPorElCatalogo(t *testing.T) {
	s, cat := superficie(t, corpusDemo())
	for _, ruta := range []string{"/alcance", "/controles", "/certificados",
		"/alcance?si=alfa.q.categoria"} {
		pedir(t, s, ruta)
	}
	delCorpus := map[string]bool{}
	for _, p := range corpusDemo() {
		for _, q := range p.Preguntas {
			delCorpus[q.Texto], delCorpus[q.Ayuda], delCorpus[q.Cita] = true, true, true
		}
		for _, o := range p.Obligaciones {
			delCorpus[o.Cita], delCorpus[o.Articulo] = true, true
		}
		for _, te := range p.Entidades {
			for _, a := range te.Atributos {
				delCorpus[a.Nombre], delCorpus[a.Ayuda], delCorpus[a.Cita] = true, true, true
			}
		}
		for _, pl := range p.Plantillas {
			delCorpus[pl.Titulo], delCorpus[pl.Cita] = true, true
		}
	}
	delete(delCorpus, "")
	for clave := range cat.vistas() {
		if delCorpus[clave] {
			t.Errorf("se le ha pedido al catalogo que traduzca %q, que es contenido del "+
				"corpus. El texto del corpus viaja tal cual, en el idioma del paquete: "+
				"traducirlo crea obra derivada", clave)
		}
	}
}

// Ningun rotulo de interfaz esta escrito a pelo en una plantilla: todos salen
// de `t "clave"`, y toda clave literal que aparezca en una plantilla tiene que
// estar declarada.
func TestLasPlantillasNoLlevanTextoDeInterfazEscritoAPelo(t *testing.T) {
	declaradas := map[string]bool{}
	for _, c := range ClavesDeCatalogo() {
		declaradas[c] = true
	}
	re := regexp.MustCompile(`\{\{-?\s*t\s+"([^"]+)"`)
	n := 0
	for _, f := range plantillasEnDisco(t) {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range re.FindAllStringSubmatch(string(b), -1) {
			n++
			if !declaradas[m[1]] {
				t.Errorf("%s pide la clave %q y ClavesDeCatalogo() no la declara: quien "+
					"escriba el catalogo no sabria que hace falta", filepath.Base(f), m[1])
			}
		}
	}
	if n < 20 {
		t.Fatalf("solo se han encontrado %d llamadas a `t` en las plantillas: o el "+
			"detector no mira, o la interfaz tiene texto a pelo", n)
	}
}

// plantillasEnDisco son TODAS las plantillas que esta superficie renderiza.
//
// SON DOS SITIOS Y NO UNO desde que la barra lateral se comparte: las propias, y
// el armazon que declara superficies/camino y que estas seis pantallas montan
// igual que el acta. Mirar solo el directorio propio dejaria sin vigilancia
// justo el fichero donde vive la tira del camino, que es lo que estas puertas
// existen para vigilar; el sintoma seria una puerta verde sobre un fichero que
// ya no contiene nada de lo que dice comprobar.
func plantillasEnDisco(t *testing.T) []string {
	t.Helper()
	var todas []string
	for _, patron := range []string{
		filepath.Join("plantillas", "*.html"),
		filepath.Join("..", "camino", "armazon", "*.html"),
	} {
		fs, err := filepath.Glob(patron)
		if err != nil {
			t.Fatalf("no puedo buscar plantillas con %q: %v", patron, err)
		}
		if len(fs) == 0 {
			t.Fatalf("el patron %q no encuentra ninguna plantilla en disco. Si el armazon "+
				"compartido ha cambiado de sitio, esta puerta ha dejado de mirarlo", patron)
		}
		todas = append(todas, fs...)
	}
	return todas
}

// ---------------------------------------------------------------------------
// Estaticos, CSP y grado cero de JavaScript
// ---------------------------------------------------------------------------

// htmx va vendorizado y lo servimos nosotros: ni una etiqueta apuntando a un
// tercero en la pagina donde el operador decide si cumple la ley.
func TestHtmxVaVendorizadoYNoPorCDN(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	_, cuerpo := pedir(t, s, "/alcance")

	for _, fuera := range []string{"//unpkg.com", "//cdn.", "https://", "http://"} {
		if strings.Contains(cuerpo, `src="`+fuera) || strings.Contains(cuerpo, `href="`+fuera) {
			t.Errorf("la pagina carga algo de fuera (%s): htmx y el CSS van vendorizados", fuera)
		}
	}
	re := regexp.MustCompile(`src="([^"]+)"|<link[^>]+href="([^"]+)"`)
	visto := 0
	for _, m := range re.FindAllStringSubmatch(cuerpo, -1) {
		ref := m[1] + m[2]
		nombre, ok := strings.CutPrefix(ref, "/estatico/")
		if !ok {
			t.Errorf("la pagina referencia %q, que no sale de nuestros estaticos", ref)
			continue
		}
		visto++
		w, _ := pedir(t, s, ref)
		if w.Code != http.StatusOK {
			t.Errorf("el estatico %q da %d", ref, w.Code)
		}
		if _, hay := ficherosEstaticos[nombre]; !hay {
			t.Errorf("%q no esta embebido", nombre)
		}
	}
	if visto < 2 {
		t.Fatalf("solo se han comprobado %d estaticos y esperaba al menos htmx y el CSS", visto)
	}
	// La licencia del codigo ajeno viaja con el.
	if _, hay := ficherosEstaticos["htmx-LICENSE.txt"]; !hay {
		t.Error("htmx viaja sin su licencia al lado. Distribuir codigo ajeno sin su " +
			"licencia no es un descuido de forma")
	}
}

// Todo lo embebido se puede servir. Un fichero embebido con una extension que
// no sabemos servir es peso muerto en el binario y una etiqueta rota.
func TestTodoLoEmbebidoSeSabeServir(t *testing.T) {
	if len(nombresEstaticos) == 0 {
		t.Fatal("no hay nada embebido: el go:embed no esta cogiendo los ficheros")
	}
	for _, n := range nombresEstaticos {
		if _, ok := ficherosEstaticos[n]; !ok {
			t.Errorf("%q esta embebido y no se sabe servir: anade su extension a "+
				"tiposPorExtension o quitalo del go:embed", n)
		}
	}
}

// Cero JavaScript en linea, cero estilos en linea, cero manejadores on*. Es lo
// que permite que quien monte el servidor ponga una CSP estricta sin negociar
// con esta superficie.
func TestLaPaginaAdmiteUnaCSPEstricta(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	for _, ruta := range []string{"/alcance", "/controles", "/certificados", "/hoy",
		"/no-existe", "/alcance?si=alfa.q.categoria"} {
		_, cuerpo := pedir(t, s, ruta)
		// Un <script> con cuerpo (no solo src) seria script en linea.
		for _, m := range regexp.MustCompile(`(?s)<script[^>]*>(.*?)</script>`).
			FindAllStringSubmatch(cuerpo, -1) {
			if strings.TrimSpace(m[1]) != "" {
				t.Errorf("%s: hay JavaScript en linea, y con script-src 'self' no se "+
					"ejecutaria: %.80s", ruta, m[1])
			}
		}
		if regexp.MustCompile(`\sstyle="`).MatchString(cuerpo) {
			t.Errorf("%s: hay un atributo style=, y con style-src 'self' no se aplicaria", ruta)
		}
		if m := regexp.MustCompile(`\son[a-z]+="`).FindString(cuerpo); m != "" {
			t.Errorf("%s: hay un manejador en linea (%s)", ruta, strings.TrimSpace(m))
		}
		if strings.Contains(cuerpo, "<style") {
			t.Errorf("%s: hay una etiqueta style en linea", ruta)
		}
	}
}

// La derivacion a un clic funciona SIN JavaScript. htmx acelera, no habilita:
// cada interaccion es un enlace de verdad con su href, asi que sin htmx la
// pagina sigue navegando. Si un dia algo solo tuviera hx-get, esto se pone rojo.
func TestSinJavaScriptLaDerivacionSigueFuncionando(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	for _, ruta := range []string{"/alcance", "/controles", "/certificados"} {
		_, cuerpo := pedir(t, s, ruta)
		for _, m := range regexp.MustCompile(`<[a-z]+[^>]*\shx-(get|post|put|delete)="[^"]*"[^>]*>`).
			FindAllString(cuerpo, -1) {
			if !strings.Contains(m, "href=") && !strings.Contains(m, "action=") {
				t.Errorf("%s: %.120s solo funciona con htmx cargado. La interaccion tiene "+
					"que ser un enlace o un formulario de verdad", ruta, m)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Ninguna ruta muta: por eso esta superficie no puede olvidarse el CSRF
// ---------------------------------------------------------------------------

func TestNingunaRutaDeLaSuperficieMuta(t *testing.T) {
	s, _ := superficie(t, corpusDemo())

	// Las rutas SALEN DEL REGISTRO, no de una lista escrita al lado.
	//
	// Antes este test llevaba las ocho rutas a mano, y por eso no sostenia lo
	// que decia: anadiendo POST /guardar, una ruta que la lista no conocia,
	// seguia en verde. La mutacion que lo daba por bueno anadia un POST a una
	// ruta que YA ESTABA en la lista, o sea que se cazaba sola. Preguntando al
	// registro, una ruta nueva entra en la comprobacion el mismo dia que se
	// escribe.
	patrones := s.Patrones()
	if len(patrones) < 6 {
		t.Fatalf("se esperaban al menos las seis pantallas registradas y hay %d: %v",
			len(patrones), patrones)
	}

	var rutas []string
	for _, p := range patrones {
		metodo, ruta, hayMetodo := strings.Cut(p, " ")
		if !hayMetodo {
			t.Errorf("el patron %q no declara metodo. Un patron sin metodo acepta TODOS, "+
				"incluidos los mutantes", p)
			continue
		}
		if metodo != http.MethodGet && metodo != http.MethodHead {
			t.Errorf("la ruta %q esta registrada para %s. Esta superficie no muta nada; si un "+
				"dia lo hace, esa ruta tiene que pasar por el middleware de CSRF de quien "+
				"construye el servidor, y este test es el recordatorio", ruta, metodo)
		}
		// Los comodines de patron no se pueden pedir tal cual.
		ruta = strings.ReplaceAll(ruta, "{$}", "")
		ruta = strings.ReplaceAll(ruta, "{fichero}", "plazum.css")
		rutas = append(rutas, ruta)
	}

	// Y ademas se prueba de verdad contra el handler, porque un patron bien
	// escrito con un mux mal montado seguiria aceptando el POST.
	for _, ruta := range rutas {
		for _, metodo := range []string{http.MethodPost, http.MethodPut, http.MethodDelete,
			http.MethodPatch} {
			r := httptest.NewRequest(metodo, ruta, nil)
			w := httptest.NewRecorder()
			s.ServeHTTP(w, r)
			if w.Code != http.StatusMethodNotAllowed && w.Code != http.StatusNotFound {
				t.Errorf("%s %s dio %d y tenia que rechazarse", metodo, ruta, w.Code)
			}
		}
	}
}

// El registro solo sirve si NADIE se lo salta. Este test lee el AST del paquete
// y prohibe llamar al mux directamente fuera de registrar.
//
// Sin esto, el arreglo de arriba dura hasta el primer s.mux.HandleFunc escrito
// por costumbre, y volveriamos a tener una ruta que ninguna puerta mira.
func TestNadieRegistraUnaRutaSaltandoseElRegistro(t *testing.T) {
	fset := token.NewFileSet()
	paquete, err := parser.ParseDir(fset, ".", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var fuera []string
	for _, pkg := range paquete {
		for nombre, fichero := range pkg.Files {
			ast.Inspect(fichero, func(n ast.Node) bool {
				fn, ok := n.(*ast.FuncDecl)
				if !ok {
					return true
				}
				if fn.Name.Name == "registrar" {
					return false // el unico sitio donde vale
				}
				ast.Inspect(fn, func(m ast.Node) bool {
					sel, ok := m.(*ast.SelectorExpr)
					if !ok || (sel.Sel.Name != "HandleFunc" && sel.Sel.Name != "Handle") {
						return true
					}
					x, ok := sel.X.(*ast.SelectorExpr)
					if !ok || x.Sel.Name != "mux" {
						return true
					}
					fuera = append(fuera, fmt.Sprintf("%s:%d en %s", nombre,
						fset.Position(m.Pos()).Line, fn.Name.Name))
					return true
				})
				return false
			})
		}
	}
	if len(fuera) > 0 {
		t.Errorf("hay rutas registradas saltandose registrar(): %v. "+
			"La puerta de \"ninguna ruta muta\" pregunta a s.Patrones(), que solo conoce lo que "+
			"pasa por registrar. Una ruta registrada por fuera no la mira nadie, que es "+
			"exactamente el agujero que este test cierra", fuera)
	}
}

// Y no hay ni un formulario, que es la otra forma de mutar sin darse cuenta.
func TestNoHayFormulariosSinCSRF(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	for _, ruta := range []string{"/alcance", "/controles", "/certificados"} {
		_, cuerpo := pedir(t, s, ruta)
		if strings.Contains(cuerpo, "<form") {
			t.Errorf("%s trae un formulario. Todo POST de este producto lleva CSRF, y el "+
				"CSRF lo emite el puerto Sesion, que esta superficie no tiene", ruta)
		}
	}
}

// ---------------------------------------------------------------------------
// Escapado: ni un template.HTML en todo el frente
// ---------------------------------------------------------------------------

// template.HTML (y sus primos) desactivan el escapado de html/template. El
// corpus lo escribe un tercero, asi que aqui no puede haber ninguno. Esto lo
// comprueba leyendo el AST y no buscando una subcadena en el fichero, para que
// un alias de import no lo esquive.
func TestNoSeDesactivaElEscapadoEnNingunSitio(t *testing.T) {
	peligrosos := map[string]bool{"HTML": true, "JS": true, "CSS": true, "URL": true,
		"HTMLAttr": true, "JSStr": true, "Srcset": true}
	for _, dir := range []string{".", filepath.Join("..", "..", "adaptadores", "plantilla")} {
		fset := token.NewFileSet()
		paquetes, err := parser.ParseDir(fset, dir, nil, 0)
		if err != nil {
			t.Fatalf("%s: %v", dir, err)
		}
		for _, p := range paquetes {
			for nombre, f := range p.Files {
				alias := "template"
				for _, imp := range f.Imports {
					if strings.Trim(imp.Path.Value, `"`) == "html/template" && imp.Name != nil {
						alias = imp.Name.Name
					}
				}
				ast.Inspect(f, func(n ast.Node) bool {
					sel, ok := n.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					id, ok := sel.X.(*ast.Ident)
					if !ok || id.Name != alias || !peligrosos[sel.Sel.Name] {
						return true
					}
					t.Errorf("%s usa template.%s, que desactiva el escapado de "+
						"html/template. El corpus lo escribe un tercero: es entrada hostil",
						nombre, sel.Sel.Name)
					return true
				})
			}
		}
	}
}

// Control negativo del detector de arriba: sobre un fuente sintetico con un
// template.HTML y con un alias, tiene que encontrar los dos. Sin esto, el verde
// del test anterior no demuestra que el detector mire.
func TestElDetectorDeEscapadoSaltaCuandoDebe(t *testing.T) {
	fuente := `package x

import (
	tpl "html/template"
)

func a() tpl.HTML { return tpl.HTML("<b>") }
func b() tpl.JS   { return tpl.JS("1") }
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "sintetico.go", fuente, 0)
	if err != nil {
		t.Fatal(err)
	}
	alias := "template"
	for _, imp := range f.Imports {
		if strings.Trim(imp.Path.Value, `"`) == "html/template" && imp.Name != nil {
			alias = imp.Name.Name
		}
	}
	peligrosos := map[string]bool{"HTML": true, "JS": true}
	n := 0
	ast.Inspect(f, func(nd ast.Node) bool {
		sel, ok := nd.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		id, ok := sel.X.(*ast.Ident)
		if ok && id.Name == alias && peligrosos[sel.Sel.Name] {
			n++
		}
		return true
	})
	if n != 4 {
		t.Fatalf("el detector debia encontrar 4 usos (dos tipos y dos conversiones) "+
			"y encontro %d", n)
	}
}
