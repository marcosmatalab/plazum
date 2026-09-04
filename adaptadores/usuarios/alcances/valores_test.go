package alcances

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// LA PUERTA DE LAS RESPUESTAS CON VALOR.
//
// # El agujero que cierra, con su cardinal
//
// Hasta el 04-09-2026 este almacen solo sabia guardar «si» y «no», y la
// entrevista pregunta VALORES desde ese mismo dia. Medido sobre el corpus real
// de `paquetes/`: **35 de las 68 preguntas se contestan con un valor**.
//
// O sea que la mitad larga de la entrevista no cabia aqui. Y lo que no cabe no
// vuelve como descarte ni como cubo: vuelve como AUSENTE, que es una respuesta
// legitima. Un alcance al que le faltan 35 preguntas son obligaciones que no
// aparecen en el calendario de un cliente **sin que nada lo diga**, que es la
// peor forma de fallo que este producto puede tener.
//
// # Por que la ida y la vuelta, y no solo la escritura
//
// Porque el fallo de esta familia no es no escribir: es escribir algo que
// despues no se puede volver a leer, o que se lee como otra cosa. Todos los
// casos de aqui cierran el almacen y lo REABREN, que es lo unico que demuestra
// que el dato sobrevive a un reinicio.

func almacenNuevo(t *testing.T) (*Almacen, string) {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), "respuestas.json")
	a, err := Abrir(Opciones{Ruta: ruta})
	if err != nil {
		t.Fatalf("abrir: %v", err)
	}
	return a, ruta
}

func TestUnValorSobreviveAlReinicio(t *testing.T) {
	a, ruta := almacenNuevo(t)
	ctx := context.Background()

	// LAS DOS FORMAS A LA VEZ, en la misma cuenta. Un test que solo guardara
	// valores no demostraria que las dos conviven, que es justo lo que una
	// entrevista real hace: 35 con valor y 33 booleanas.
	if err := a.Responder(ctx, "ciso", "alfa.q.categoria", ConValor("ALTA")); err != nil {
		t.Fatalf("guardar un valor: %v", err)
	}
	if err := a.Responder(ctx, "ciso", "alfa.q.booleana", Booleana(Si)); err != nil {
		t.Fatalf("guardar un si: %v", err)
	}

	otro, err := Abrir(Opciones{Ruta: ruta})
	if err != nil {
		t.Fatalf("reabrir: %v", err)
	}
	al, err := otro.De(ctx, "ciso")
	if err != nil {
		t.Fatalf("leer: %v", err)
	}

	if got := al.Respuestas["alfa.q.categoria"]; !got.EsValor() || got.Valor != "ALTA" {
		t.Errorf("el valor ha vuelto como %#v y se guardo ConValor(\"ALTA\").\n"+
			"  Es el fallo que trajo esta puerta: 35 de las 68 preguntas del corpus se "+
			"contestan asi, y hasta el 04-09-2026 ninguna cabia aqui", got)
	}
	if got := al.Respuestas["alfa.q.booleana"]; got != Booleana(Si) {
		t.Errorf("el si ha vuelto como %#v: las dos formas tienen que convivir", got)
	}

	// Y QUE EL FICHERO LO DIGA DE LAS DOS FORMAS. Sin esto, un almacen que
	// guardara el valor solo en memoria pasaria la comprobacion de arriba en
	// cuanto alguien reusara el mismo objeto por descuido.
	b, err := os.ReadFile(ruta) // #nosec G304 -- fichero del propio test
	if err != nil {
		t.Fatalf("leer el fichero: %v", err)
	}
	if !strings.Contains(string(b), `"respuesta": "valor"`) ||
		!strings.Contains(string(b), `"valor": "ALTA"`) {
		t.Errorf("el fichero no trae la fila con sus dos campos:\n%s", b)
	}
}

// LAS TRES FORMAS DE NO SER UNA CONTESTACION, y ninguna se degrada a un valor
// por defecto. Es el invariante 8 y su tercera hermana: presente y no
// interpretable.
func TestUnaContestacionQueNoEsUnaDeLasDosFormasNoSeGuarda(t *testing.T) {
	a, _ := almacenNuevo(t)
	ctx := context.Background()

	casos := []struct {
		nombre string
		c      Contestacion
	}{
		{"el valor cero, que es el que sale por olvido", Contestacion{}},
		{"las dos formas a la vez", Contestacion{Booleana: Si, Valor: "ALTA"}},
		{"un valor que solo tiene espacios", ConValor("   ")},
		{"una booleana que no es ni si ni no", Contestacion{Booleana: Respuesta(9)}},
		{"un valor mas largo que el tope", ConValor(strings.Repeat("x", MaxLongitudDeValor+1))},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			err := a.Responder(ctx, "ciso", "q.1", c.c)
			if !errors.Is(err, ErrContestacionNoValida) {
				t.Fatalf("tenia que fallar con ErrContestacionNoValida y da %v", err)
			}
			// Y NO SE HA ESCRITO NADA. Un rechazo que ademas escribe es peor
			// que no rechazar: deja el disco con lo que se acaba de negar.
			al, err := a.De(ctx, "ciso")
			if err != nil {
				t.Fatalf("leer: %v", err)
			}
			if len(al.Respuestas) != 0 {
				t.Errorf("se ha rechazado la contestacion y aun asi hay %d respuestas guardadas",
					len(al.Respuestas))
			}
		})
	}
}

// LA FRONTERA DE ENTRADA DEL FICHERO, que es la que va a leer lo que edite una
// persona a mano.
//
// LOS DOS DESACUERDOS ENTRE LOS DOS CAMPOS SON ERROR, y esa es la mitad que no
// se escribe sola: la tentacion es leer el campo que se entiende y seguir.
func TestUnaFilaQueSeContradiceNoSeLee(t *testing.T) {
	casos := []struct {
		nombre string
		fila   string
		quiero error
	}{
		{
			"se anuncia con valor y no lo trae",
			`{"pregunta":"q.1","respuesta":"valor"}`,
			ErrAlmacenIlegible,
		},
		{
			"se anuncia con valor y lo trae en blanco",
			`{"pregunta":"q.1","respuesta":"valor","valor":"  "}`,
			ErrAlmacenIlegible,
		},
		{
			"dice si Y ademas trae un valor",
			`{"pregunta":"q.1","respuesta":"si","valor":"ALTA"}`,
			ErrAlmacenIlegible,
		},
		{
			"una etiqueta que no es ninguna de las tres",
			`{"pregunta":"q.1","respuesta":"quizas"}`,
			ErrAlmacenIlegible,
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			ruta := filepath.Join(t.TempDir(), "respuestas.json")
			doc := `{"version":` + itoa(VersionDelAlmacen) + `,"alcances":[{"usuario":"ciso",` +
				`"actualizado":"2026-09-04T09:00:00Z","respuestas":[` + c.fila + `]}]}`
			if err := os.WriteFile(ruta, []byte(doc), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := Abrir(Opciones{Ruta: ruta})
			if !errors.Is(err, c.quiero) {
				t.Fatalf("tenia que fallar con %v y da %v", c.quiero, err)
			}
		})
	}

	// EL CONTROL POSITIVO, sin el cual lo de arriba solo demuestra que Abrir
	// falla: una fila BIEN formada de las dos clases tiene que leerse.
	t.Run("las dos filas buenas si se leen", func(t *testing.T) {
		ruta := filepath.Join(t.TempDir(), "respuestas.json")
		doc := `{"version":` + itoa(VersionDelAlmacen) + `,"alcances":[{"usuario":"ciso",` +
			`"actualizado":"2026-09-04T09:00:00Z","respuestas":[` +
			`{"pregunta":"q.1","respuesta":"si"},` +
			`{"pregunta":"q.2","respuesta":"valor","valor":"ALTA"}]}]}`
		if err := os.WriteFile(ruta, []byte(doc), 0o600); err != nil {
			t.Fatal(err)
		}
		a, err := Abrir(Opciones{Ruta: ruta})
		if err != nil {
			t.Fatalf("un almacen bien escrito no carga: %v", err)
		}
		al, err := a.De(context.Background(), "ciso")
		if err != nil {
			t.Fatal(err)
		}
		if al.Respuestas["q.1"] != Booleana(Si) {
			t.Errorf("q.1 ha vuelto como %#v", al.Respuestas["q.1"])
		}
		if al.Respuestas["q.2"] != ConValor("ALTA") {
			t.Errorf("q.2 ha vuelto como %#v", al.Respuestas["q.2"])
		}
	})
}

// UN FICHERO ESCRITO POR UN BINARIO ANTERIOR A LOS VALORES SE SIGUE LEYENDO.
//
// Es la compatibilidad hacia atras, y se comprueba con el fichero de verdad y no
// razonando sobre el codigo: una fila de si/no de ayer no trae el campo `valor`,
// asi que llega vacio, y el camino que lo lee tiene que pasar por encima sin
// tocarlo.
func TestLosFicherosDeAyerSeSiguenLeyendo(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "respuestas.json")
	// Exactamente la forma que escribia el binario de antes: sin campo `valor`.
	doc := `{"version":` + itoa(VersionDelAlmacen) + `,"alcances":[{"usuario":"ciso",` +
		`"actualizado":"2026-09-04T09:00:00Z","respuestas":[` +
		`{"pregunta":"q.1","respuesta":"si"},{"pregunta":"q.2","respuesta":"no"}]}]}`
	if err := os.WriteFile(ruta, []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}
	a, err := Abrir(Opciones{Ruta: ruta})
	if err != nil {
		t.Fatalf("un fichero de ayer tiene que seguir cargando y da: %v", err)
	}
	al, err := a.De(context.Background(), "ciso")
	if err != nil {
		t.Fatal(err)
	}
	if al.Respuestas["q.1"] != Booleana(Si) || al.Respuestas["q.2"] != Booleana(No) {
		t.Errorf("las respuestas de ayer no han vuelto igual: %#v", al.Respuestas)
	}
}
