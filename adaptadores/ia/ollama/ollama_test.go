package ollama

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/ia"
)

// servidorDeMentira devuelve un Ollama falso y el contador de peticiones que ha
// recibido. El CONTADOR es la mitad del valor: sin el, "la IA esta apagada" se
// comprueba mirando un error, y un error se puede devolver despues de haber
// mandado la peticion.
func servidorDeMentira(t *testing.T, responde func(w http.ResponseWriter, r *http.Request)) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	var n atomic.Int64
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		responde(w, r)
	}))
	t.Cleanup(s.Close)
	return s, &n
}

func respuestaBuena(w http.ResponseWriter, _ *http.Request) {
	interior, _ := json.Marshal(salida{
		Diff:       "marcar la obligacion como cubierta",
		Cita:       "un trozo literal del texto de la fuente",
		HashFuente: strings.Repeat("ab", 32),
	})
	_ = json.NewEncoder(w).Encode(respuesta{Response: string(interior)})
}

// -------------------------------------------------------------------------
// LA SEGUNDA PUERTA DEL INVARIANTE 9, MEDIDA EN BYTES Y NO EN ERRORES.
// -------------------------------------------------------------------------

func TestConLaIAApagadaNoSaleNiUnaPeticionDeLaMaquina(t *testing.T) {
	s, peticiones := servidorDeMentira(t, respuestaBuena)

	t.Setenv(ia.Variable, "1")
	c, err := Nuevo(s.URL, "modelo-de-prueba:1")
	if !errors.Is(err, ia.ErrIADesactivada) {
		t.Fatalf("con %s=1 el adaptador se construye: (%v, %v)", ia.Variable, c, err)
	}
	if c != nil {
		t.Fatal("se devuelve un cliente ademas del error: quien ignore el error tendra un " +
			"adaptador de modelo vivo con la IA apagada")
	}
	if n := peticiones.Load(); n != 0 {
		t.Fatalf("con la IA apagada han salido %d peticiones.\n"+
			"  La puerta no es que devuelva error: es que NO SALGA NADA de la maquina.\n"+
			"  Un error devuelto despues de mandar la peticion se lee igual desde el\n"+
			"  codigo y es lo contrario desde la red del cliente.", n)
	}

	// CONTROL POSITIVO. Sin esto, lo de arriba lo pasaria igual un adaptador
	// roto que nunca habla con nadie, y la puerta estaria midiendo una averia.
	t.Setenv(ia.Variable, "0")
	c, err = Nuevo(s.URL, "modelo-de-prueba:1")
	if err != nil {
		t.Fatalf("con la IA encendida el adaptador no se construye: %v", err)
	}
	if _, err := c.Proponer("una tarea", []byte("unos textos")); err != nil {
		t.Fatalf("con la IA encendida Proponer falla: %v", err)
	}
	if n := peticiones.Load(); n != 1 {
		t.Fatalf("con la IA encendida han salido %d peticiones, tenia que salir 1", n)
	}
}

func TestUnInterruptorIlegibleParaElAdaptadorYNoLoEnciende(t *testing.T) {
	s, peticiones := servidorDeMentira(t, respuestaBuena)
	t.Setenv(ia.Variable, "quiza")
	if _, err := Nuevo(s.URL, "modelo-de-prueba:1"); !errors.Is(err, ia.ErrInterruptorIlegible) {
		t.Fatalf("con un interruptor ilegible el adaptador da %v", err)
	}
	if n := peticiones.Load(); n != 0 {
		t.Fatalf("con el interruptor ilegible han salido %d peticiones. Un valor que no se "+
			"entiende no puede caer del lado de 'enciende y habla con la red'", n)
	}
}

// -------------------------------------------------------------------------
// LA TERCERA FORMA DE LA NADA EN LA RESPUESTA DEL MODELO.
// -------------------------------------------------------------------------

func TestUnaRespuestaQueNoSeEntiendeEsErrorYNoUnaPropuestaVacia(t *testing.T) {
	casos := []struct {
		nombre    string
		responde  func(http.ResponseWriter, *http.Request)
		centinela error
	}{
		{"no es JSON", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("<html>que tal</html>"))
		}, ErrRespuestaIlegible},
		{"JSON de Ollama con basura dentro", func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(respuesta{Response: "lo siento, no puedo ayudarte"})
		}, ErrRespuestaIlegible},
		{"cuerpo vacio", func(w http.ResponseWriter, _ *http.Request) {}, ErrRespuestaIlegible},
		{"error del servidor", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}, ErrEstadoInesperado},
		{"modelo que no existe", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}, ErrEstadoInesperado},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			s, _ := servidorDeMentira(t, c.responde)
			t.Setenv(ia.Variable, "0")
			cli, err := Nuevo(s.URL, "modelo-de-prueba:1")
			if err != nil {
				t.Fatal(err)
			}
			p, err := cli.Proponer("tarea", nil)
			if !errors.Is(err, c.centinela) {
				t.Fatalf("da (%+v, %v), se esperaba %v.\n"+
					"  Una respuesta presente que no se entiende NO es una propuesta vacia:\n"+
					"  es un dato que hay y no se entiende, y devolver el valor cero sin\n"+
					"  error mete una propuesta en blanco en el camino de una pantalla que\n"+
					"  dice 'el modelo propone'.", p, err, c.centinela)
			}
		})
	}

	// CONTROL POSITIVO: la respuesta buena si sale, con sus campos.
	s, _ := servidorDeMentira(t, respuestaBuena)
	t.Setenv(ia.Variable, "0")
	cli, err := Nuevo(s.URL, "modelo-de-prueba:1")
	if err != nil {
		t.Fatal(err)
	}
	p, err := cli.Proponer("tarea", []byte("contexto"))
	if err != nil {
		t.Fatalf("la respuesta buena da error: %v", err)
	}
	if p.Cita == "" || p.HashFuente == "" || p.Modelo != "modelo-de-prueba:1" {
		t.Errorf("la propuesta sale incompleta: %+v", p)
	}
	if len(p.DigestPrompt) != 64 {
		t.Errorf("DigestPrompt = %q; sin la huella del prompt no queda escrito con que se "+
			"pregunto cuando una persona confirme la propuesta", p.DigestPrompt)
	}
}

func TestElAdaptadorNoSeConstruyeSinLoQueNecesita(t *testing.T) {
	t.Setenv(ia.Variable, "0")
	casos := []struct {
		nombre    string
		endpoint  string
		modelo    string
		centinela error
	}{
		{"sin endpoint", "", "m", ErrSinEndpoint},
		{"endpoint de solo espacios", "   ", "m", ErrSinEndpoint},
		{"sin modelo", "http://127.0.0.1:11434", "", ErrSinModelo},
		{"esquema que no es http", "ftp://maquina/cosa", "m", ErrEndpointInvalido},
		{"sin anfitrion", "http:///api", "m", ErrEndpointInvalido},
		{"no es una direccion", "esto no es una url", "m", ErrEndpointInvalido},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if _, err := Nuevo(c.endpoint, c.modelo); !errors.Is(err, c.centinela) {
				t.Fatalf("da %v, se esperaba %v", err, c.centinela)
			}
		})
	}
	// CONTROL POSITIVO.
	if _, err := Nuevo("http://127.0.0.1:11434", "modelo-de-prueba:1"); err != nil {
		t.Fatalf("una configuracion buena no construye: %v", err)
	}
}

// El modelo va en la propuesta SIEMPRE, porque un eval sin modelo fijado no
// significa nada (docs/ia.md, arnes punto 8).
func TestLaPropuestaDiceQueModeloLaHizo(t *testing.T) {
	s, _ := servidorDeMentira(t, respuestaBuena)
	t.Setenv(ia.Variable, "0")
	cli, err := Nuevo(s.URL, "un-modelo-concreto:7b")
	if err != nil {
		t.Fatal(err)
	}
	p, err := cli.Proponer("tarea", nil)
	if err != nil {
		t.Fatal(err)
	}
	if p.Modelo != "un-modelo-concreto:7b" {
		t.Errorf("Modelo = %q", p.Modelo)
	}
}
