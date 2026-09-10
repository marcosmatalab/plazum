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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/estado"
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

// bloqueDePregunta recorta el <li> de una pregunta.
func bloqueDePregunta(t *testing.T, cuerpo, id string) string {
	t.Helper()
	inicio := strings.Index(cuerpo, `id="p-`+id+`"`)
	if inicio < 0 {
		t.Fatalf("la pagina no trae la pregunta %q", id)
	}
	fin := strings.Index(cuerpo[inicio:], "</li>")
	if fin < 0 {
		t.Fatalf("el bloque de la pregunta %q no cierra", id)
	}
	return cuerpo[inicio : inicio+fin]
}

// enlacesQueResponden devuelve TODOS los href del bloque de una pregunta que la
// CONTESTAN, sea con un si, con un no o con un valor.
//
// # Por que hace falta el general y no basta con el de si/no
//
// Desde que la entrevista sabe preguntar valores, una pregunta se contesta de
// dos formas distintas segun lo que declare el corpus, y un ayudante que solo
// supiera de si/no dejaria las 25 preguntas con valor sin probar por ninguno de
// los tests que navegan la entrevista. Los que necesitan las DOS direcciones
// (que aplique y que no aplique) siguen usando enlacesDePregunta, que exige que
// la pregunta sea de si/no y lo dice si no lo es.
//
// El filtro es «el enlace lleva una respuesta A ESTA pregunta»: asi el de
// deshacer, que esta en el mismo bloque y no contesta nada, se queda fuera.
func enlacesQueResponden(t *testing.T, cuerpo, id string) []string {
	t.Helper()
	var out []string
	for _, m := range reEnlace.FindAllStringSubmatch(bloqueDePregunta(t, cuerpo, id), -1) {
		u := strings.ReplaceAll(m[1], "&amp;", "&")
		v, err := url.Parse(u)
		if err != nil {
			continue
		}
		q := v.Query()
		contesta := len(q[ClaveValor(id)]) > 0
		for _, x := range append(q[ParamSi], q[ParamNo]...) {
			if x == id {
				contesta = true
			}
		}
		if contesta {
			out = append(out, u)
		}
	}
	if len(out) == 0 {
		t.Fatalf("la pregunta %q no ofrece ni un enlace que la conteste", id)
	}
	return out
}

var (
	reFormAbre = regexp.MustCompile(`(?s)<form[ >][^>]*action="([^"]*)"[^>]*>`)
	reInputTag = regexp.MustCompile(`(?s)<input[ >][^>]*>`)
	reAtributo = regexp.MustCompile(`([a-z]+)="([^"]*)"`)
)

// respuestaQueOfreceLaPagina devuelve una direccion que CONTESTA la pregunta
// usando solo lo que la pagina ofrece, sea un enlace o un formulario.
//
// # Por que hace falta, y por que no vale componer la consulta a mano
//
// Un enumerado se contesta con un enlace, pero un texto, un entero y una fecha
// se contestan escribiendo, y eso es un formulario: no hay ningun enlace que
// lleve la respuesta dentro, porque la respuesta no existe hasta que alguien la
// escribe. Los tests que RECORREN la entrevista (la sugerencia que avanza, la
// lista larga que conserva el modo) se quedaban parados en la primera pregunta
// de campo libre.
//
// Se compone lo que mandaria EL NAVEGADOR al enviar ese formulario: su action,
// sus campos ocultos y su campo de escritura, leidos del propio HTML. No es
// «construir una consulta a mano», que es lo que el comentario de esos tests
// prohibe con razon: si la pantalla deja de pintar los ocultos, esto pierde las
// respuestas anteriores exactamente igual que las perderia un operador, y el
// test se entera.
func respuestaQueOfreceLaPagina(t *testing.T, cuerpo, id, valor string) string {
	t.Helper()
	bloque := bloqueDePregunta(t, cuerpo, id)
	if !strings.Contains(bloque, "<form") {
		return enlacesQueResponden(t, cuerpo, id)[0]
	}
	m := reFormAbre.FindStringSubmatch(bloque)
	if m == nil {
		t.Fatalf("la pregunta %q trae un formulario sin action", id)
	}
	q := url.Values{}
	campo := ""
	for _, tag := range reInputTag.FindAllString(bloque, -1) {
		at := map[string]string{}
		for _, a := range reAtributo.FindAllStringSubmatch(tag, -1) {
			at[a[1]] = a[2]
		}
		switch at["type"] {
		case "hidden":
			q.Add(at["name"], strings.ReplaceAll(at["value"], "&amp;", "&"))
		case "text":
			campo = at["name"]
		}
	}
	if campo == "" {
		t.Fatalf("el formulario de la pregunta %q no trae campo donde escribir", id)
	}
	q.Set(campo, valor)
	return strings.ReplaceAll(m[1], "&amp;", "&") + "?" + q.Encode()
}

// enlacesDePregunta devuelve los href de si y no de una pregunta DE SI/NO.
//
// Falla si la pregunta pide un valor, y eso es deliberado: una pregunta con
// valor no tiene «no», porque un «no» a «¿que categoria alcanza el sistema?» no
// significa nada. Devolver una opcion cualquiera haciendola pasar por el «no»
// dejaria pasando un test que estaria probando otra cosa.
func enlacesDePregunta(t *testing.T, cuerpo, id string) (si, no string) {
	t.Helper()
	for _, u := range enlacesQueResponden(t, cuerpo, id) {
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
		t.Fatalf("la pregunta %q no ofrece si (%q) y no (%q). Si es una pregunta CON VALOR "+
			"esto es lo correcto y el test tiene que usar enlacesQueResponden: un enumerado "+
			"no tiene «no», y fabricarle uno seria inventarse una respuesta", id, si, no)
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

	// LA DIRECCION QUE SUMA SALE DE UNA PREGUNTA CON VALOR, y la que resta de
	// una de si/no. No es un capricho del test: una pregunta con valor NO TIENE
	// «no», porque un «no» a «¿que categoria alcanza el sistema?» no significa
	// nada, y fabricarle uno seria inventarse una respuesta.
	//
	// Que eso quite una via para llegar a «no aplica» es una CORRECCION y no una
	// perdida. Hasta hoy un «no» a esa pregunta escondia las filas que dependen
	// de la categoria del sistema, o sea que absolvia de golpe por una respuesta
	// que nadie podia dar en serio. Absolver de mas es el error caro: el que
	// acusa lo corrige quien lee, el que absuelve lo descubre el inspector.
	valor := enlacesQueResponden(t, inicio, "alfa.q.categoria")[0]
	_, no := enlacesDePregunta(t, inicio, "beta.q.riesgo")

	// El clic no sale de Alcance: no hay pantalla de configuracion en medio.
	if u, _ := url.Parse(valor); u.Path != "/alcance" {
		t.Errorf("responder lleva a %q y tiene que quedarse en /alcance: la derivacion es "+
			"a un clic, no a un clic y un formulario", u.Path)
	}

	// Un clic en una opcion: la obligacion pasa a aplicar y la pagina dice por
	// que, con la cita del articulo del que sale la pregunta.
	w, conValor := pedir(t, s, valor)
	if w.Code != http.StatusOK {
		t.Fatalf("seguir el enlace de respuesta dio %d", w.Code)
	}
	aplican := seccionAplican(conValor)
	if !strings.Contains(aplican, "alfa.o.auditoria") {
		t.Errorf("tras contestar la pregunta con un valor, la obligacion condicionada no "+
			"aparece entre las que aplican.\n--- seccion ---\n%s", aplican)
	}
	exige(t, conValor,
		rotulo("es", "derivacion.respondiste_valor"), // el tipo de motivo
		"demo alfa art. 3",                           // la cita, del corpus, tal cual
		"Que categoria tiene el sistema",             // el texto de la pregunta, tal cual
	)
	// Y EL VALOR CONTESTADO SE VE, tal cual lo declara el paquete. Sin esto la
	// pantalla diria «has respondido» sin decir que, y en una pregunta de tres
	// opciones eso no es una respuesta.
	exige(t, conValor, rotulo("es", "alcance.pregunta.valor.respondida"), "BAJA")

	// Un clic en "no": la obligacion que cuelga de esa pregunta pasa a no
	// aplicar, y desaparece de las que aplican.
	_, conNo := pedir(t, s, no)
	if strings.Contains(seccionAplican(conNo), "beta.o.evaluacion") {
		t.Error("tras responder que no, la obligacion sigue entre las que aplican. " +
			"Una derivacion que solo suma no deriva nada")
	}
	// `f=todos` porque lo que se busca es una fila NO_APLICA, y la tabla ensena
	// por defecto lo que si aplica desde el 08-09-2026. Quien quiere ver lo que
	// ha dejado de aplicarle lo pide, que es justo lo que hace este test.
	_, controles := pedir(t, s, "/controles?f=todos&"+strings.SplitN(no, "?", 2)[1])
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
	// Se contesta la primera con el primer enlace que la conteste, sea un «si»
	// o un valor: la sugerencia tiene que moverse igual en los dos casos.
	_, despues := pedir(t, s, enlacesQueResponden(t, cuerpo, orden[0])[0])
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
	// `f=todos`: sin responder nada, la obligacion del paquete nuevo esta
	// PENDIENTE, y la tabla ensena por defecto lo que aplica.
	_, controles := pedir(t, s, "/controles?f=todos")
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
	// `f=todos` en las dos: sin responder, la plantilla esta PENDIENTE y la
	// huerfana no aplica, y ninguna de las dos sale en la vista por defecto.
	_, sinResponder := pedir(t, s, "/certificados?f=todos")
	exige(t, sinResponder, "alfa.pl.informe", rotulo("es", "estado.pendiente"))
	prohibe(t, sinResponder, rotulo("es", "derivacion.pregunta_desconocida"))

	// Respondiendo que si a la pregunta que desbloquea una de ellas, la
	// plantilla pasa a hacer falta.
	_, conSi := pedir(t, s, "/certificados?f=todos&si=alfa.q.categoria")
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
		// LA SECCION DE CAMPOS ABIERTA (D-13). Sin este estado, la clave que
		// devuelve a la lista corta sale aqui como «publicada y nadie la pide»,
		// que es literalmente cierto y es lo que este test tiene que provocar.
		"/alcance?c=todas",
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

	// LA PUERTA A LOS DOCUMENTOS DEL CLIENTE, que necesita su propio barrido por
	// lo mismo que el guardado: sus tres claves solo se piden cuando quien monta
	// ha cableado la pantalla de la pieza 3, y el barrido normal la monta sin
	// ella. Sin este bloque quedarian declaradas y sin pedir.
	{
		s, cat := superficie(t, corpusDemo(), func(o *Opciones) {
			o.DocumentosRuta = "/documentos/"
			o.DocumentosConsultas = 12
		})
		pedir(t, s, "/alcance")
		for k, v := range cat.vistas() {
			pedidas[k] += v
		}
	}

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
		_ = al.Responder(t.Context(), "ciso", "alfa.q.categoria", Booleana(Si))
		_ = al.Responder(t.Context(), "ciso", "ya.no.existe", Booleana(No)) // la huerfana
		s, cat := superficie(t, corpusDemo(), conGuardado(al, "ciso"))
		pedir(t, s, "/alcance")                           // en_tu_cuenta, cuando, huerfanas
		pedir(t, s, "/alcance?"+ParamSi+"=alfa.q.nombre") // desde_enlace, adoptar
		// CON UN VALOR PUESTO Y LA CUENTA ABIERTA: es la unica combinacion en
		// la que sale el aviso de que el guardado no se lleva los valores, y
		// tiene que salir, porque guardar la mitad en silencio es la peor
		// version de ese boton.
		pedir(t, s, "/alcance?"+ClaveValor("alfa.q.categoria")+"=BAJA")
		pedirPost(t, s, url.Values{"accion": {"loquesea"}}) // accion_desconocida
		pedirPost(t, s, url.Values{"accion": {"si"}, "pregunta": {"no.existe"}})
		for k, v := range cat.vistas() {
			pedidas[k] += v
		}
	}
	{
		// CON QUIEN PUBLIQUE: el rotulo que avisa de que adoptar publica ademas
		// el alcance de la INSTALACION, que es el que ve cualquiera que abra
		// esta instalacion sin entrar. Solo se alcanza con Opciones.Publicar
		// puesto, o sea con la superficie montada como la monta el producto: sin
		// esta rama, esa clave se quedaria sin traducir hasta que la viera un
		// cliente en crudo, justo en la pantalla donde se decide publicar.
		al := nuevoAlmacenFalso()
		s, cat := superficie(t, corpusDemo(), conPublicacion(al, "ciso", &publicadorFalso{}))
		pedir(t, s, "/alcance?"+ParamSi+"=alfa.q.nombre")
		for k, v := range cat.vistas() {
			pedidas[k] += v
		}
	}
	{
		// CON LA CONSECUENCIA DE CADA PREGUNTA (pieza 2), y con sus DOS ramas:
		// la que activa obligaciones y la que no activa ninguna, que son dos
		// frases distintas. Sin esta entrada, sus cuatro rotulos saldrian aqui
		// como «publicados y nadie los pide», que es literalmente cierto.
		// LAS DOS RAMAS, CON DOS ENTRADAS. Cuantas preguntas de si/no pinta el
		// corpus de prueba no lo decide este fichero, asi que un doble que
		// reparta por numero de llamada deja una rama sin recorrer en cuanto
		// haya una sola. Dos entradas explicitas no dependen de nada.
		for _, n := range []int{5, 0} {
			s, cat := superficie(t, corpusDemo(), conConsecuencias(consecuenciasFalsas{n: n}))
			pedir(t, s, "/alcance?"+ParamVer+"="+VerTodas)
			for k, v := range cat.vistas() {
				pedidas[k] += v
			}
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
	// EL PAQUETE ROTO Y LA TABLA PAGINADA, que son dos claves cada uno y ninguna
	// se alcanza desde el corpus de demostracion entero: la errata del corpus
	// («pregunta_desconocida») necesita un paquete con erratas, y los dos
	// rotulos de paginacion necesitan MAS de una pagina, o sea una superficie
	// con la pagina corta. Los dos barridos van con `f=todos`, porque lo que
	// recorren son filas que la vista por defecto ya no ensena.
	{
		s, cat := superficie(t, []*corpus.Paquete{paqueteRoto()}, conCamino())
		pedir(t, s, "/controles?f=todos")
		pedir(t, s, "/certificados?f=todos")
		for k, v := range cat.vistas() {
			pedidas[k] += v
		}
	}
	{
		s, cat := superficie(t, corpusDemo(), conCamino(), func(o *Opciones) {
			o.PorPagina = 1
		})
		pedir(t, s, "/controles?f=todos&p=1")
		pedir(t, s, "/controles?f=todos&p=2")
		for k, v := range cat.vistas() {
			pedidas[k] += v
		}
	}

	{
		s, cat := superficie(t, corpusDemo(), conCamino())
		for _, ruta := range []string{"/alcance", "/alcance?si=alfa.q.categoria",
			"/hoy", "/controles", "/no-existe",
			// LA TABLA ENTERA Y LA TABLA PAGINADA, las dos, desde el
			// 08-09-2026. La vista por defecto ensena solo lo que aplica, asi
			// que sin `f=todos` los cuatro motivos de una fila que no aplica
			// (respondiste_no, lo_pide_y_no_aplica, pregunta_desconocida,
			// entregable_huerfano) se quedarian declarados y sin pedir, y sin
			// una tabla con mas de una pagina, los dos rotulos de paginacion
			// tambien. Es el barrido de claves: aqui se recorre lo que existe,
			// no lo que se ve al entrar.
			"/controles?f=todos", "/certificados?f=todos",
			// Y una fila que NO aplica y otra PENDIENTE, con sus motivos:
			// «respondiste que no» y «lo pide y no aplica» solo salen tras
			// contestar, y nunca en la vista por defecto.
			"/controles?f=todos&no=alfa.q.categoria",
			"/certificados?f=todos&no=alfa.q.categoria",
			// LOS CUATRO ESTADOS DE UNA PREGUNTA CON VALOR, y ninguno se
			// alcanza desde la pagina en blanco. Sin estas cuatro entradas, los
			// rotulos de la mitad con valor se quedan sin traducir hasta que se
			// los encuentre en crudo quien conteste una fecha mal, que es el
			// peor momento para descubrir que el catalogo esta corto.
			//
			//	contestada             el valor puesto, con su rotulo y el
			//	                       enlace de deshacer;
			//	el valor no se entiende  la TERCERA forma de la nada: hay dato y
			//	                       no se interpreta;
			//	dos valores            contradictoria;
			//	valor Y si a la vez    la otra contradiccion, la que mezcla las
			//	                       dos formas de contestar.
			"/alcance?v.alfa.q.categoria=BAJA",
			"/alcance?v.alfa.q.categoria=NO_EXISTE_ESTE_VALOR",
			"/alcance?v.alfa.q.categoria=BAJA&v.alfa.q.categoria=ALTA",
			"/alcance?v.alfa.q.categoria=BAJA&si=alfa.q.categoria",
		} {
			pedir(t, s, ruta)
		}
		for k, v := range cat.vistas() {
			pedidas[k] += v
		}
	}

	// LA PREGUNTA DE FECHA, que es la unica cuyo campo lleva pista de formato.
	// El corpus sintetico base no tiene ninguna, asi que se pide con uno que si.
	{
		s, cat := superficie(t, []*corpus.Paquete{paqueteConFecha()})
		pedir(t, s, "/alcance?"+ParamVer+"="+VerTodas)
		for k, v := range cat.vistas() {
			pedidas[k] += v
		}
	}

	// LA PREGUNTA CON VALOR EN CONTROLES, para que el motivo de la derivacion
	// («has respondido a esta pregunta», «llego un dato que no se entiende»)
	// tenga quien lo pida. En Alcance el panel solo detalla las que APLICAN, asi
	// que el motivo de una pendiente no sale por ahi.
	{
		s, cat := superficie(t, corpusDemo(), conCamino())
		for _, ruta := range []string{
			"/controles?f=todos&v.alfa.q.categoria=BAJA",
			"/controles?f=todos&v.alfa.q.categoria=NO_EXISTE_ESTE_VALOR",
		} {
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

	// LA EVIDENCIA, EN SUS TRES MONTAJES (A2 de D-22). Hacen falta los tres
	// porque son tres paginas distintas y ninguna pide las claves de las otras:
	// con sesion salen las columnas y los estados, sin sesion sale la frase de
	// que no se ensena, y con el adaptador roto sale la de ilegible.
	//
	// Los estados se recorren TODOS, y se reparten por las filas que haya en
	// vez de escribir una lista aqui: `estado.Todos()` sale del nucleo, asi que
	// un estado nuevo alli entra en este contrato solo.
	filas := idsDeControles(t)
	if len(filas) < 3 {
		t.Fatalf("el corpus de prueba pinta %d filas de control y hacen falta al menos "+
			"tres para recorrer con prueba, sin observaciones y sin prueba", len(filas))
	}
	// UNA PETICION POR ESTADO, y no un reparto entre las filas: cuantas filas
	// pinta el corpus de prueba no lo controla este fichero, y un reparto que
	// dependa de ese numero deja estados sin recorrer el dia que el corpus
	// cambie. Es la leccion de `consecuenciasFalsas`, otra vez.
	//
	// En cada peticion, la ultima fila se queda SIN entrada (esa es la de
	// evidencia.sin_prueba) y la penultima va sin fecha ni recolector (las de
	// sin_fecha y sin_recolector).
	var porID map[string]Evidencia
	for _, e := range estado.Todos() {
		porID = map[string]Evidencia{}
		for i, id := range filas[:len(filas)-1] {
			ev := Evidencia{Estado: e.String(), Cierra: i%2 == 0}
			if i < len(filas)-2 {
				ev.Recolectada = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
				ev.Recolector = "manual"
			} else {
				ev.SinObservaciones = true
			}
			porID[id] = ev
		}
		alEv := nuevoAlmacenFalso()
		sEv, catEv := superficie(t, corpusDemo(),
			conEvidencia(alEv, "ana@ejemplo", &evidenciasFalsas{por: porID}))
		pedir(t, sEv, "/controles?f=todos")
		for k, v := range catEv.vistas() {
			pedidas[k] += v
		}
	}

	// Y UNA PETICION POR MOTIVO, por lo mismo que una por estado y con una
	// razon mas: el estado se elige entre ocho y el motivo entre once, asi que
	// colgarlos del mismo recorrido dejaria tres motivos sin pedir nunca.
	//
	// Se recorren `estado.CadenasDelEstado()` y no una lista escrita aqui: los
	// motivos los declara el motor, y el dia que Calcular gane una rama, su
	// clave entra en este contrato sola y el catalogo se pone rojo hasta que
	// alguien la traduzca. Escribirlos seria una segunda copia del motor.
	//
	// LOS ARGUMENTOS VAN PUESTOS aunque a esta puerta le baste la clave: sin
	// ellos el motivo se pinta con los `%s` sin rellenar y la pagina que este
	// test recorre no seria la que ve nadie.
	for _, f := range estado.CadenasDelEstado() {
		porID = map[string]Evidencia{}
		for i, id := range filas[:len(filas)-1] {
			porID[id] = Evidencia{
				Estado: estado.Obsoleto.String(), Cierra: i%2 == 0,
				MotivoClave: f.Clave,
				MotivoArgs:  []string{"1", "2026-09-13"},
				Recolectada: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
				Recolector:  "manual",
			}
		}
		alMot := nuevoAlmacenFalso()
		sMot, catMot := superficie(t, corpusDemo(),
			conEvidencia(alMot, "ana@ejemplo", &evidenciasFalsas{por: porID}))
		pedir(t, sMot, "/controles?f=todos")
		for k, v := range catMot.vistas() {
			pedidas[k] += v
		}
	}

	sSin, catSin := superficie(t, corpusDemo(), func(o *Opciones) {
		o.Evidencia = &evidenciasFalsas{por: porID}
	})
	pedir(t, sSin, "/controles?f=todos")
	for k, v := range catSin.vistas() {
		pedidas[k] += v
	}

	alRota := nuevoAlmacenFalso()
	sRota, catRota := superficie(t, corpusDemo(),
		conEvidencia(alRota, "ana@ejemplo", &evidenciasFalsas{falla: errors.New("rota")}),
		func(o *Opciones) { o.AlFallar = func(error) {} })
	pedir(t, sRota, "/controles?f=todos")
	for k, v := range catRota.vistas() {
		pedidas[k] += v
	}

	// Y EL ESTADO CON DIRECTIVAS, con las dos ramas del aviso. Sin el, las dos
	// claves salen aqui como «publicadas y nadie las pide», que es literalmente
	// cierto: el corpus de demostracion no trae ninguna directiva. La respuesta
	// no es quitarlas, es traer la entrada que recorre su rama (M47).
	sDir, catDir := superficie(t, append(corpusDemo(), paqueteConDirectiva(false),
		paqueteConDirectiva(true)))
	pedir(t, sDir, "/controles?f=todos")
	for k, v := range catDir.vistas() {
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

// reFormulario casa la etiqueta de apertura de un formulario. La clase de
// caracteres detras de `form` esta escrita a mano y no con un atajo: <form>
// sin atributos tambien es un formulario, y ademas es el peor de todos, porque
// un formulario sin `method` vale POST para quien lo lea por encima y GET para
// el navegador.
var reFormulario = regexp.MustCompile(`(?s)<form[ >][^>]*>`)

// Y no hay ni un formulario QUE MUTE, que es la otra forma de mutar sin darse
// cuenta.
//
// # Por que esto dejo de ser «ni un formulario» el 04-09-2026, y que se
// conserva exactamente
//
// Hasta hoy esta puerta buscaba la subcadena `<form` y bastaba, porque sin
// almacen esta superficie era GET de arriba abajo y no tenia ningun formulario.
// Desde que la entrevista sabe preguntar valores hay uno: el campo donde se
// escribe un texto, un entero o una fecha, que no se puede pedir con un enlace
// porque el enlace tendria que traer la respuesta escrita de antemano.
//
// LO QUE SE CONSERVA ES LA PROPIEDAD, NO LA LETRA. Lo que hace peligroso a un
// formulario aqui es que MUTE sin token, y lo que muta es el POST: el
// middleware de superficies/serve exige CSRF POR METODO, asi que un GET no
// entra en esa puerta y tampoco la necesita, porque un GET de esta superficie
// no escribe en ningun sitio. La comprobacion pasa a ser exacta y MAS ESTRECHA
// en lo que importa: cada formulario tiene que declarar `method="get"`.
//
// Y declararlo, no omitirlo. El HTML sin `method` vale GET por defecto, asi que
// aceptar la omision dejaria pasar un formulario en el que alguien quiso poner
// POST y se le olvido, que es justo el descuido que esta puerta busca.
func TestNoHayFormulariosSinCSRF(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	for _, ruta := range []string{"/alcance", "/controles", "/certificados"} {
		_, cuerpo := pedir(t, s, ruta)
		for _, tag := range reFormulario.FindAllString(cuerpo, -1) {
			if !strings.Contains(tag, `method="get"`) {
				t.Errorf("%s trae el formulario %q, que no declara method=\"get\". Todo POST "+
					"de este producto lleva CSRF, y el CSRF lo emite el puerto Sesion, que "+
					"esta superficie no tiene. Un formulario sin method vale GET para el "+
					"navegador y no vale para esta puerta: se declara, para que un POST "+
					"olvidado se vea", ruta, tag)
			}
		}
	}
}

// EL CONTROL POSITIVO DE LA PUERTA DE ARRIBA, y sin el la puerta se habria
// quedado vacia al estrecharla.
//
// Si `reFormulario` no casara con nada (una expresion mal escrita, un cambio de
// plantilla), el bucle no daria ni una vuelta y el test saldria verde diciendo
// que no hay formularios que muten. Es la familia del `go test -run` que no
// casa con nada: un verde indistinguible del verde de verdad.
//
// Asi que se exige que la pagina de Alcance traiga AL MENOS UNO. Lo trae porque
// el corpus de prueba tiene una pregunta de texto (`alfa.q.nombre`), y si algun
// dia no lo trajera habria que enterarse, porque significaria que las preguntas
// de campo libre han dejado de pintarse.
func TestLaPuertaDeLosFormulariosMiraAlgunFormulario(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	_, cuerpo := pedir(t, s, "/alcance")
	if n := len(reFormulario.FindAllString(cuerpo, -1)); n == 0 {
		t.Fatal("la pagina de Alcance no trae ni un formulario, asi que la puerta de " +
			"TestNoHayFormulariosSinCSRF esta recorriendo una lista vacia y da verde sin " +
			"mirar nada. O la expresion no casa, o las preguntas de campo libre han dejado " +
			"de pintarse")
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

// LA TERCERA FAMILIA DEL NUCLEO EN ESTA PANTALLA: LO QUE SE DESCARTA SE CUENTA.
//
// # Qué descarte, y desde cuándo
//
// Desde el 08-09-2026 la tabla enseña por defecto sólo lo que TE APLICA. Eso es
// un descarte de cara al usuario, y D-13 dice lo que hay que hacer con uno: *«ni
// enumerar ni callar: un contador, y una puerta para verlos si quiere»*.
//
// Esta puerta comprueba las dos mitades de esa frase sobre la vista por defecto:
//
//	el CONTADOR   los cuatro chips traen su número, incluido el de lo que no se
//	              está pintando, así que el descarte se ve aunque no se lea;
//	la PUERTA     el chip de «todos» lleva su enlace y ese enlace TRAE las filas
//	              que la vista por defecto se dejó.
//
// # Y la suma, que es lo que hace esto comprobable y no decorativo
//
// Los tres estados tienen que sumar el total. Sin esa comprobación, un chip
// podría decir cualquier número y la pantalla seguiría pareciendo honesta: es la
// misma ley de conservación que `contabilidad_test.go` le exige al calendario,
// aplicada a la tabla.
func TestLoQueLaTablaNoEnsenaPorDefectoSeCuentaYSePuedeAbrir(t *testing.T) {
	s, _ := superficie(t, corpusDemo(), conCamino())
	_, defecto := pedir(t, s, "/controles")

	// EL CONTADOR: los cuatro chips con su numero.
	cuentas := map[string]int{}
	re := regexp.MustCompile(
		`<span class="rotulo">([^<]*)</span><span class="cuenta">(\d+)</span>`)
	for _, m := range re.FindAllStringSubmatch(defecto, -1) {
		n, err := strconv.Atoi(m[2])
		if err != nil {
			t.Fatalf("el chip %q no trae un numero: %q", m[1], m[2])
		}
		cuentas[m[1]] = n
	}
	if len(cuentas) != 4 {
		t.Fatalf("la vista por defecto trae %d chips y son cuatro (todos, aplica, "+
			"pendiente, no aplica): sin los cuatro, lo que no se pinta no se cuenta.\n"+
			"  chips: %v", len(cuentas), cuentas)
	}

	// LA LEY DE CONSERVACION: los tres estados suman el total.
	var total, suma int
	for k, n := range cuentas {
		if strings.Contains(k, "filtro.todos") {
			total = n
			continue
		}
		suma += n
	}
	if total == 0 {
		t.Fatal("el chip de «todos» dice cero: este recorrido estaria comprobando una tabla " +
			"vacia, que es donde cualquier suma cuadra")
	}
	if suma != total {
		t.Errorf("los tres estados suman %d y el total dice %d.\n"+
			"  Un chip que diga cualquier numero deja la pantalla pareciendo honesta sin "+
			"serlo: es la misma ley de conservacion que el calendario ya se exige.\n"+
			"  chips: %v", suma, total, cuentas)
	}

	// LA PUERTA: lo que no se pinta por defecto SI sale al pedirlo, y son mas.
	_, todos := pedir(t, s, "/controles?f="+FiltroTodos)
	filasDe := func(cuerpo string) int {
		return strings.Count(cuerpo, `<tr class="e-`)
	}
	if filasDe(todos) <= filasDe(defecto) {
		t.Errorf("la vista por defecto pinta %d filas y la de «todos» %d.\n"+
			"  Si no son mas, o el descarte no existe (y entonces esta puerta no vigila "+
			"nada) o el enlace de «todos» no trae lo que promete, que es peor: seria "+
			"contar lo que no se puede abrir.", filasDe(defecto), filasDe(todos))
	}
	// Y el enlace del chip esta en la pagina, no solo el numero: un contador sin
	// puerta es la mitad que D-13 rechaza.
	if !strings.Contains(defecto, ParamFiltro+"="+FiltroTodos) {
		t.Error("la vista por defecto cuenta lo que no ensena y NO pinta el enlace para " +
			"verlo. D-13 pide las dos cosas: un contador Y una puerta")
	}
}
