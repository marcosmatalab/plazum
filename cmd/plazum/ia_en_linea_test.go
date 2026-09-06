package main

import (
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/ia"
	"github.com/marcosmatalab/plazum/superficies/camino"
)

// PUERTA: NI UNA PESTANA DE CHAT.
//
// # Que afirma, y por que es una decision de producto y no de estilo
//
// La IA de plazum va EN LINEA, en el punto de friccion, con su cita visible y
// dos botones (docs/ia.md §5). No hay un sitio aparte al que ir a hablar con
// ella. Si lo hubiera, cuatro cosas se romperian a la vez:
//
//   - La cita dejaria de estar al lado del dato que justifica. Una respuesta de
//     chat con una cita dentro es una respuesta que hay que creerse; una
//     propuesta pegada a la pregunta que modifica se comprueba mirando.
//   - El invariante 9 se volveria decorativo. Con la IA en su propia pestana, el
//     camino determinista y el camino con IA son dos productos, y el que se usa
//     es el segundo.
//   - El TTFV dejaria de medir lo que la gente hace, porque el trabajo se iria a
//     una pantalla que el camino guiado no recorre.
//   - Y la friccion seguiria donde estaba: quien no sabe que contestar en la
//     pregunta 7 no abre otra pestana, abandona.
//
// # COMO SE COMPRUEBA, Y POR QUE ASI
//
// Enumerando las rutas del servidor REAL y buscando el vocabulario de un chat.
// No se mira el codigo ni los nombres de los ficheros: se mira lo que el
// producto SIRVE, que es lo unico que un cliente puede abrir.
//
// EL VOCABULARIO ES CERRADO Y SE DECLARA. Una lista de palabras prohibidas es
// una lista que se queda corta, y eso se dice en vez de disimularse: esta puerta
// caza la pestana que alguien anada llamandola como se llaman, no una que se
// llame `/asistente-de-redaccion-inteligente`. Lo que la hace util no es cubrir
// todos los nombres posibles: es que el dia que alguien monte un chat tenga que
// elegir un nombre raro A PROPOSITO, y eso ya no es un descuido.
func TestNiUnaPestanaDeChat(t *testing.T) {
	srv := servidorDelCamino(t, nil, func(*http.Request) string { return "ciso" })
	rutas := srv.Rutas()
	if len(rutas) < 8 {
		t.Fatalf("el servidor enumera %d rutas: esta puerta estaria mirando casi nada",
			len(rutas))
	}
	for _, r := range rutas {
		ruta := strings.ToLower(r.String())
		for _, palabra := range vocabularioDeChat {
			if strings.Contains(ruta, palabra) {
				t.Errorf(`el producto sirve la ruta %s, que suena a una pestana de chat.

  La IA de plazum va EN LINEA, en el punto de friccion, con su cita al lado del
  dato que justifica y dos botones. Un sitio aparte al que ir a hablar con ella
  separa la cita de lo que justifica, convierte el modo sin IA en otro producto,
  y deja la friccion donde estaba: quien no sabe que contestar en la pregunta 7
  no abre otra pestana, abandona.`, r.String())
			}
		}
	}

	// Y EL CAMINO GUIADO TAMPOCO DECLARA UN PASO ASI. Es la otra puerta por la
	// que entraria: no como ruta suelta, sino como septimo paso del camino, que
	// es donde se le daria bendicion de producto.
	for _, p := range camino.Canonico() {
		id := strings.ToLower(p.ID + " " + p.Ruta)
		for _, palabra := range vocabularioDeChat {
			if strings.Contains(id, palabra) {
				t.Errorf("el camino guiado declara el paso %q, que suena a una pestana de "+
					"chat. La IA va dentro de los pasos que ya hay, no como un paso mas",
					p.ID)
			}
		}
	}

	// CONTROL NEGATIVO DEL DETECTOR, en las dos direcciones. Sin esto, un
	// detector roto (una lista vacia, un ToLower que se cae) daria verde sobre
	// cualquier cosa y esta puerta seria un comentario.
	if len(vocabularioDeChat) < 4 {
		t.Fatalf("el vocabulario de chat tiene %d palabras: es demasiado corto para decir "+
			"nada", len(vocabularioDeChat))
	}
	falsa := "GET /asistente/{$}"
	caza := false
	for _, palabra := range vocabularioDeChat {
		if strings.Contains(strings.ToLower(falsa), palabra) {
			caza = true
		}
	}
	if !caza {
		t.Error("el detector no reconoce una ruta que SI es una pestana de chat: la lista de " +
			"palabras ha dejado de cubrir el caso obvio")
	}
}

// vocabularioDeChat son los nombres con los que se llama a una pestana de chat.
//
// SE DECLARA AQUI Y NO SE DERIVA de nada, porque no hay de donde: es una lista de
// palabras. Lo que la sostiene es su control negativo, que exige que reconozca el
// caso obvio, y su limite escrito arriba.
var vocabularioDeChat = []string{
	"/chat", "/asistente", "/copilot", "/copiloto", "/pregunta-a", "/ia/",
}

// PUERTA: EL CAMINO COMPLETO EN VERDE CON LA IA APAGADA.
//
// # Que hacia falta para que esta puerta dejara de ser casi vacia
//
// La casilla de ETAPAS.md lo decia con esas palabras: «la puerta existe desde el
// invariante 9 y hoy es casi vacia». Y era cierto por dos motivos distintos que
// se arreglaron en dos dias distintos:
//
//   - Hasta el 05-09-2026, el recorrido del camino guiado hacia seis GET y no
//     contestaba nada, asi que con la IA apagada o encendida veia lo mismo:
//     seis pantallas vacias. Ahora contesta la entrevista, sube el censo y
//     publica el alcance, o sea que EJERCE el producto.
//   - Y hasta el 06-09-2026 no habia ninguna pieza del bloque de IA dentro del
//     camino. Ahora esta la pieza 2 (la consecuencia de cada pregunta), que es
//     determinista a proposito: con la IA apagada tiene que seguir saliendo, y
//     eso es lo que esta puerta comprueba de verdad.
//
// # LO QUE ESTA PUERTA VIGILA AQUI, y lo que vigila CI
//
// CI corre la suite ENTERA con PLAZUM_SIN_IA=1, incluido el recorrido del camino
// (`puerta "suite completa sin IA"`). Lo que no puede comprobar desde fuera es
// que la variable LLEGUE al `plazum serve` que ese recorrido arranca: si alguien
// le pusiera un Env propio al subproceso, la puerta de CI seguiria verde
// midiendo un servidor con la IA encendida. Eso es lo que se comprueba aqui.
func TestElInterruptorDeIALlegaAlProductoYNoSoloAlTest(t *testing.T) {
	// LAS DOS FORMAS DE LA NADA, y la peligrosa es la de arriba: con la
	// variable ausente, `Apagada()` tiene que decir que NO esta apagada, o el
	// producto se apagaria solo en toda instalacion que no la ponga.
	t.Setenv(ia.Variable, "")
	if err := os.Unsetenv(ia.Variable); err != nil {
		t.Fatal(err)
	}
	if apagada, err := ia.Apagada(); err != nil || apagada {
		t.Fatalf("sin la variable puesta, Apagada() = (%v, %v) y tenia que ser (false, nil)",
			apagada, err)
	}
	t.Setenv(ia.Variable, "1")
	apagada, err := ia.Apagada()
	if err != nil || !apagada {
		t.Fatalf("con %s=1, Apagada() = (%v, %v)", ia.Variable, apagada, err)
	}

	// Y LO QUE DE VERDAD IMPORTA: que el recorrido del camino no le corte el
	// entorno al `plazum serve` que arranca. Se comprueba sobre el FICHERO del
	// arnes, que es donde vive esa decision, y no sobre una ejecucion: una
	// ejecucion con la variable puesta pasaria igual si el subproceso la
	// ignorase, porque hoy ninguna pieza del camino usa el modelo.
	b, err := os.ReadFile("../../ttfv_camino_test.go")
	if err != nil {
		t.Fatalf("leyendo el arnes del camino: %v", err)
	}
	if reEnvPropio.Match(b) {
		t.Errorf(`el arnes del camino le pone un entorno propio al `+"`plazum serve`"+` que arranca.

  Con eso, la puerta de CI que corre la suite entera con %s=1 seguiria verde
  midiendo un servidor con la IA ENCENDIDA: la variable se quedaria en el
  proceso del test y no llegaria al producto. Es una guarda que deja de mirar y
  no se pone roja, se queda callada.

  Arreglo: no tocar cmd.Env, que hace que el subproceso herede el entorno.`,
			ia.Variable)
	}
}

// reEnvPropio caza la asignacion de un entorno propio al subproceso del arnes.
var reEnvPropio = regexp.MustCompile(`(?m)^\s*s\.cmd\.Env\s*=`)
