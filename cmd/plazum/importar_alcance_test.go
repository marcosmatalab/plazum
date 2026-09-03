package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/usuarios/alcances"
	"github.com/marcosmatalab/plazum/nucleo/metrica"
)

// escribirAlcance deja un alcance.json a mano, con el bloque de respuestas que
// se le pida. Se compone con el tipo del producto para que un cambio de nombres
// de campo rompa aqui en vez de producir un fichero que no carga.
func escribirAlcance(t *testing.T, rs []respuestaDeJSON) string {
	t.Helper()
	doc := alcanceExportado{
		Organizacion: "Acme SL", Sujeto: "sis",
		Descripcion: "alcance de prueba", Respuestas: rs, Hechos: []hechoDeJSON{},
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "alcance.json")
	if err := os.WriteFile(ruta, append(b, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return ruta
}

// LOS CINCO DESTINOS DE UNA FILA, cada uno con su entrada.
//
// Un destino que ninguna entrada alcanzara seria un descargo que no existe, y
// la mutacion lo dejaria verde porque no habria nada que romper. Aqui se
// recorren los cinco y ademas se exige que la ley de conservacion cuadre, que
// es lo que impide que un destino nuevo se olvide.
func TestLaImportacionRepartTodasLasFilasYLoDice(t *testing.T) {
	reales := preguntasRealesDelCorpus(t, 2)
	var ids []string
	for id := range reales {
		ids = append(ids, id)
	}
	// El orden de un mapa no es estable; para este test da igual cual sea cual,
	// pero tienen que ser dos distintas.
	if len(ids) != 2 {
		t.Fatalf("se esperaban dos preguntas del corpus y hay %d", len(ids))
	}

	ruta := escribirAlcance(t, []respuestaDeJSON{
		{Campo: "a.b", Valor: "si", Pregunta: ids[0]},           // importada
		{Campo: "a.b", Valor: "si", Pregunta: ids[0]},           // repetida identica
		{Campo: "a.c", Valor: "no", Pregunta: ids[1]},           // importada
		{Campo: "org.empleados", Valor: "212", Pregunta: ""},    // sin pregunta
		{Campo: "s.cat", Valor: "ALTA", Pregunta: ids[0] + "x"}, // desconocida
	})
	// Una fila con pregunta conocida y valor que no es si/no: el destino «con
	// valor». Se anade aparte porque necesita una pregunta que el corpus SI
	// declare.
	ruta2 := escribirAlcance(t, []respuestaDeJSON{
		{Campo: "s.cat", Valor: "ALTA", Pregunta: ids[0]},
	})

	datos := t.TempDir()
	rc, salida, errores := correrAlcance(t, "--importar", ruta, "--cuenta", "ciso",
		"--datos", datos, "--corpus", "../../paquetes")
	if rc != 0 {
		t.Fatalf("importar ha salido %d:\n%s", rc, errores)
	}
	for _, trozo := range []string{
		"filas leidas", "guardadas en la cuenta",
		"repetian una pregunta que ya habia salido",
		"sin identificador de pregunta",
		"id de pregunta que este corpus no declara",
	} {
		if !strings.Contains(salida, trozo) {
			t.Errorf("la salida de la importacion no dice %q. Un destino que no se cuenta es "+
				"una respuesta que desaparece sin dejar rastro.\n%s", trozo, salida)
		}
	}
	// LA CONSERVACION, IMPRESA: si algun cubo faltara, la orden lo diria.
	if strings.Contains(salida, "AVISO") {
		t.Errorf("la ley de conservacion no cuadra en una importacion normal:\n%s", salida)
	}

	rc, salida, errores = correrAlcance(t, "--importar", ruta2, "--cuenta", "ciso",
		"--datos", t.TempDir(), "--corpus", "../../paquetes")
	if rc != 0 {
		t.Fatalf("importar el de valores ha salido %d:\n%s", rc, errores)
	}
	if !strings.Contains(salida, "atributo CON VALOR") {
		t.Errorf("una respuesta con un valor que no es si/no no se cuenta ni se nombra:\n%s",
			salida)
	}
	if strings.Contains(salida, "AVISO") {
		t.Errorf("la ley de conservacion no cuadra con una respuesta con valor:\n%s", salida)
	}

	// Y la cuenta tiene lo que tenia que tener, ni mas ni menos.
	almacen, err := alcances.Abrir(alcances.Opciones{Ruta: alcances.RutaPorDefecto(datos)})
	if err != nil {
		t.Fatal(err)
	}
	al, err := almacen.De(context.Background(), "ciso")
	if err != nil {
		t.Fatal(err)
	}
	if len(al.Respuestas) != 2 {
		t.Errorf("la cuenta ha quedado con %d respuestas y tenian que entrar 2: %v",
			len(al.Respuestas), al.Respuestas)
	}
}

// UNA FILA QUE DICE A QUE PREGUNTA CONTESTA Y NO DICE QUE CONTESTA ES UN ERROR,
// nunca «sin responder».
//
// Es la tercera forma de la nada (invariante 8) en esta frontera: la fila que
// falta es la nada; la fila que hay con el valor vacio es un dato roto, y
// tomarlo por la nada es inventarse una respuesta que nadie dio.
func TestUnaRespuestaConValorVacioEsErrorYNoSeGuardaNada(t *testing.T) {
	reales := preguntasRealesDelCorpus(t, 1)
	var id string
	for k := range reales {
		id = k
	}
	ruta := escribirAlcance(t, []respuestaDeJSON{{Campo: "a.b", Valor: "", Pregunta: id}})
	datos := t.TempDir()
	rc, _, errores := correrAlcance(t, "--importar", ruta, "--cuenta", "ciso",
		"--datos", datos, "--corpus", "../../paquetes")
	if rc == 0 {
		t.Fatal("una fila con pregunta y sin valor se ha importado como si nada. Es un dato " +
			"roto, y darlo por «sin responder» es inventarse una respuesta")
	}
	if !strings.Contains(errores, "no trae ningun valor") {
		t.Errorf("el error no dice que pasa:\n%s", errores)
	}
	// Y NO SE HA ESCRITO NADA: se para antes de tocar la cuenta.
	if _, err := os.Stat(alcances.RutaPorDefecto(datos)); err == nil {
		t.Error("se ha escrito el fichero de respuestas a pesar del error. Una importacion " +
			"que falla a medias deja una cuenta que no es ni la de antes ni la del fichero")
	}
}

// Dos filas que contestan la MISMA pregunta con valores distintos no se
// resuelven eligiendo una: cual ganara dependeria del orden del fichero, y el
// orden no lo firma nadie.
func TestDosRespuestasDistintasALaMismaPreguntaSeRechazan(t *testing.T) {
	reales := preguntasRealesDelCorpus(t, 1)
	var id string
	for k := range reales {
		id = k
	}
	ruta := escribirAlcance(t, []respuestaDeJSON{
		{Campo: "a.b", Valor: "si", Pregunta: id},
		{Campo: "a.b", Valor: "no", Pregunta: id},
	})
	rc, _, errores := correrAlcance(t, "--importar", ruta, "--cuenta", "ciso",
		"--datos", t.TempDir(), "--corpus", "../../paquetes")
	if rc == 0 {
		t.Fatal("un alcance con la misma pregunta contestada de dos formas se ha importado")
	}
	if !strings.Contains(errores, "el orden no lo firma nadie") {
		t.Errorf("el error no dice por que se rechaza:\n%s", errores)
	}
}

// SIN --cuenta NO SE IMPORTA. Las respuestas son de alguien.
func TestImportarSinCuentaNoEscribeEnUnCajonComun(t *testing.T) {
	ruta := escribirAlcance(t, []respuestaDeJSON{})
	rc, _, errores := correrAlcance(t, "--importar", ruta,
		"--datos", t.TempDir(), "--corpus", "../../paquetes")
	if rc != 2 {
		t.Fatalf("importar sin --cuenta ha salido %d y tenia que ser 2", rc)
	}
	if !strings.Contains(errores, "cajon comun") {
		t.Errorf("el error no explica por que hace falta la cuenta:\n%s", errores)
	}
}

// LAS TRES FUENTES NO SE MEZCLAN. Dos a la vez dicen dos cosas y elegir una en
// silencio exporta un alcance que el operador no ha pedido.
func TestLasTresFuentesDeRespuestasNoSeMezclan(t *testing.T) {
	casos := [][]string{
		{"--cuenta", "ciso", "--url", "http://x/alcance?si=a"},
		{"--cuenta", "ciso", "--respuestas", "si=a"},
		{"--url", "http://x/alcance?si=a", "--respuestas", "si=a"},
	}
	for _, c := range casos {
		args := append(append([]string{}, c...),
			"--sujeto", "sis", "--datos", t.TempDir(), "--corpus", "../../paquetes")
		rc, _, errores := correrAlcance(t, args...)
		if rc != 2 {
			t.Errorf("%v ha salido %d y tenia que ser 2", c, rc)
		}
		if !strings.Contains(errores, "mas de una fuente") {
			t.Errorf("%v: el error no dice que hay dos fuentes:\n%s", c, errores)
		}
	}
}

// Una cuenta sin nada guardado NO produce un alcance vacio: un fichero sin
// hechos deriva menos obligaciones y no lo dice, asi que es peor que no tener
// fichero.
func TestUnaCuentaSinRespuestasNoEscribeUnAlcanceVacio(t *testing.T) {
	rc, _, errores := correrAlcance(t, "--cuenta", "ciso", "--datos", t.TempDir(),
		"--sujeto", "sis", "--corpus", "../../paquetes")
	if rc == 0 {
		t.Fatal("una cuenta sin respuestas ha escrito un alcance igualmente")
	}
	if !strings.Contains(errores, "no tiene ninguna respuesta guardada") {
		t.Errorf("el error no dice que la cuenta esta vacia:\n%s", errores)
	}
}

// NI --url NI --respuestas ESCRIBEN UN ALCANCE VACIO.
//
// El caso que trae la mitad nueva es concreto: `plazum serve` tiene tambien una
// bandera `--respuestas` y ahi es un FICHERO. Quien se cruce las dos escribe
// `plazum alcance --respuestas respuestas.json`, eso se parsea como una consulta
// con una clave rara y cero respuestas, y hasta hoy salia un alcance.json sin ni
// un hecho con codigo 0. Un fichero sin hechos deriva menos obligaciones y no lo
// dice, asi que es peor que no tener fichero.
//
// Se recorren las dos formas de «ninguna respuesta»: la consulta vacia y la que
// trae algo que no es una respuesta.
func TestNiLaUrlNiLaConsultaEscribenUnAlcanceVacio(t *testing.T) {
	casos := []struct {
		nombre string
		args   []string
	}{
		{"url sin consulta", []string{"--url", "http://localhost:8443/alcance"}},
		{"url con consulta que no son respuestas",
			[]string{"--url", "http://localhost:8443/alcance?ver=todas"}},
		{"respuestas que en realidad es un fichero",
			[]string{"--respuestas", "respuestas.json"}},
		{"respuestas con algo que no es una respuesta", []string{"--respuestas", "ver=todas"}},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			salida := filepath.Join(t.TempDir(), "alcance.json")
			args := append(append([]string{}, c.args...),
				"--sujeto", "sis", "--salida", salida, "--corpus", "../../paquetes")
			rc, _, errores := correrAlcance(t, args...)
			if rc == 0 {
				t.Fatalf("%s ha salido 0: se ha escrito un alcance sin ni una respuesta "+
					"dentro, y eso son obligaciones que no aparecen sin que nadie lo diga",
					c.nombre)
			}
			if !strings.Contains(errores, "ninguna respuesta") {
				t.Errorf("%s: el error no dice que falta:\n%s", c.nombre, errores)
			}
			if _, err := os.Stat(salida); err == nil {
				t.Errorf("%s: se ha escrito el fichero igualmente", c.nombre)
			}
		})
	}
	// EL CONTROL POSITIVO: con respuestas de verdad SI se escribe. Sin esto, una
	// version que rechazara siempre pasaria el test entero.
	reales := preguntasRealesDelCorpus(t, 1)
	var id string
	for k := range reales {
		id = k
	}
	salida := filepath.Join(t.TempDir(), "alcance.json")
	rc, _, errores := correrAlcance(t, "--respuestas", "si="+id, "--sujeto", "sis",
		"--salida", salida, "--corpus", "../../paquetes")
	if rc != 0 {
		t.Fatalf("con una respuesta de verdad ha salido %d:\n%s", rc, errores)
	}
	if _, err := os.Stat(salida); err != nil {
		t.Fatalf("con una respuesta de verdad no se ha escrito el alcance: %v", err)
	}
}

// El control de la aritmetica de los cubos, con dato sintetico: si un destino
// se anade y nadie lo mete en Cubos(), la conservacion deja de cuadrar y esto
// se pone rojo.
func TestLosCubosDeLaImportacionCuadranConLoLeido(t *testing.T) {
	c := CuentaDeLaImportacion{
		Leidas: 7, Importadas: 3, SinPregunta: 1, Repetidas: 1,
		ConValor: []string{"a"}, Desconocidas: []string{"b"},
	}
	if err := metrica.Cuadra(c.Leidas, "las filas", c.Cubos()); err != nil {
		t.Fatalf("los cubos de la importacion no cuadran: %v", err)
	}
	// Y el control negativo: si falta un cubo, TIENE que romperse.
	sinUno := c.Cubos()
	delete(sinUno, "sin id de pregunta")
	if err := metrica.Cuadra(c.Leidas, "las filas", sinUno); err == nil {
		t.Fatal("quitando un cubo la conservacion sigue cuadrando, asi que esta comprobacion " +
			"no comprueba nada")
	}
}
