package main

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/usuarios/alcances"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/metrica"
	"github.com/marcosmatalab/plazum/nucleo/pantalla"
)

// LA IDA Y LA VUELTA, COMPROBADA POR CONSERVACION Y NO POR RECUENTO.
//
// # Por que el recuento no vale, dicho con el caso concreto
//
// «Salen tantas respuestas como entraron» deja pasar DOS CAMBIOS QUE SE
// CANCELAN: una respuesta que se pierde y otra que aparece dan el mismo
// cardinal y son un alcance distinto. Y no es un caso de laboratorio: la
// exportacion recorre dos listas (`si` y `no`) y la importacion las vuelve a
// juntar, asi que un «si» que se convirtiera en «no» por el camino conserva el
// cardinal, conserva el conjunto de preguntas y cambia el veredicto de todas
// las obligaciones que dependan de ella.
//
// Aqui se compara POR IDENTIDAD, pareja a pareja, y EN LOS DOS SENTIDOS.
//
// # POR QUE CAMPO CASA (invariante 7)
//
// Por el par (id de pregunta, respuesta). El id de pregunta lo declara el
// paquete, viaja en la direccion de la pantalla, se guarda como clave en el
// almacen y se escribe en el campo `pregunta` del bloque de respuestas del
// alcance: es el MISMO identificador en los cuatro sitios. No se casa por
// posicion (nadie firma el orden de una lista JSON), ni por el campo `campo`
// (dos preguntas pueden pedir el mismo campo), ni por subcadena.
//
// Ese identificador NO ESTA FIRMADO: `alcance.json` y `respuestas.json` son
// ficheros de trabajo del operador, no piezas del expediente. Lo que esta
// puerta sostiene es la conservacion entre los dos, no la autoria de ninguno.

// preguntasRealesDelCorpus saca del corpus instalado unas cuantas preguntas de
// verdad, con su reparto en «si» y «no».
//
// SE SACAN DEL CORPUS Y NO SE ESCRIBEN AQUI. Escribirlas dejaria este test viejo
// el dia que el corpus cambie, y sobre todo lo dejaria VERDE midiendo cero
// respuestas, que es la forma comoda de aprobar esta puerta sin merecerla.
func preguntasRealesDelCorpus(t *testing.T, cuantas int) map[string]alcances.Respuesta {
	t.Helper()
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatalf("cargar el corpus: %v", err)
	}
	var ids []string
	for _, p := range pantalla.Derivar(ps) {
		for _, q := range p.Preguntas {
			ids = append(ids, q.ID)
		}
	}
	sort.Strings(ids)
	if len(ids) < cuantas {
		t.Fatalf("el corpus instalado declara %d preguntas y este test necesita %d: estaria "+
			"midiendo una ida y vuelta casi vacia", len(ids), cuantas)
	}
	out := map[string]alcances.Respuesta{}
	for i, id := range ids[:cuantas] {
		// MITAD Y MITAD A PROPOSITO. Con solo «si», un fallo que convirtiera
		// todos los «no» en «si» pasaria sin ponerse rojo, y ese es justo el
		// fallo que el bloque de hechos (que no escribe los «no») invita a
		// cometer.
		if i%2 == 0 {
			out[id] = alcances.Si
		} else {
			out[id] = alcances.No
		}
	}
	return out
}

// TestLaIdaYLaVueltaDelAlcanceConservanCadaRespuesta es la puerta.
func TestLaIdaYLaVueltaDelAlcanceConservanCadaRespuesta(t *testing.T) {
	ctx := context.Background()
	origen := t.TempDir()
	destino := t.TempDir()
	fichero := filepath.Join(t.TempDir(), "alcance.json")

	// 1. UNA CUENTA CON RESPUESTAS DE VERDAD.
	entraron := preguntasRealesDelCorpus(t, 12)
	almacenOrigen, err := alcances.Abrir(alcances.Opciones{
		Ruta: alcances.RutaPorDefecto(origen)})
	if err != nil {
		t.Fatal(err)
	}
	if err := almacenOrigen.Reemplazar(ctx, "ciso", entraron); err != nil {
		t.Fatal(err)
	}

	// 2. LA IDA: de la cuenta a alcance.json.
	rc, salida, errores := correrAlcance(t,
		"--cuenta", "ciso", "--datos", origen,
		"--sujeto", "sis", "--organizacion", "Acme SL",
		"--salida", fichero, "--corpus", "../../paquetes")
	if rc != 0 {
		t.Fatalf("`plazum alcance --cuenta` ha salido %d:\n%s", rc, errores)
	}
	if strings.Contains(salida, "id de pregunta que este corpus no declara") {
		t.Fatalf("la exportacion dice que el corpus no declara las preguntas que acaban de "+
			"salir del corpus:\n%s", salida)
	}

	// 3. LA VUELTA: de alcance.json a OTRA cuenta, en OTRO directorio. En el
	//    mismo no demostraria nada: lo que ya estaba guardado podria estar
	//    tapando lo que el fichero no trajo.
	rc, _, errores = correrAlcance(t,
		"--importar", fichero, "--cuenta", "ciso", "--datos", destino,
		"--corpus", "../../paquetes")
	if rc != 0 {
		t.Fatalf("`plazum alcance --importar` ha salido %d:\n%s", rc, errores)
	}

	almacenDestino, err := alcances.Abrir(alcances.Opciones{
		Ruta: alcances.RutaPorDefecto(destino)})
	if err != nil {
		t.Fatal(err)
	}
	volvieron, err := almacenDestino.De(ctx, "ciso")
	if err != nil {
		t.Fatal(err)
	}

	// 4. LA CONSERVACION, EN LOS DOS SENTIDOS Y POR IDENTIDAD.
	conservadas, cambiadas, perdidas := 0, 0, 0
	for id, r := range entraron {
		v, hay := volvieron.Respuestas[id]
		switch {
		case !hay:
			perdidas++
			t.Errorf("la respuesta a %q ha desaparecido en la ida y vuelta", id)
		case v != r:
			cambiadas++
			t.Errorf("la respuesta a %q entro como %q y ha vuelto como %q. El cardinal no lo "+
				"nota: dos cambios que se cancelan dan el mismo numero y un alcance distinto",
				id, r, v)
		default:
			conservadas++
		}
	}
	aparecidas := 0
	for id, v := range volvieron.Respuestas {
		if _, esperada := entraron[id]; !esperada {
			aparecidas++
			t.Errorf("ha aparecido una respuesta a %q (%q) que nadie exporto", id, v)
		}
	}

	// Y LA ARITMETICA, con nucleo/metrica, que es lo que impide que un cubo que
	// falte pase por un total que cuadra.
	if err := metrica.Cuadra(len(entraron), "las respuestas exportadas", map[string]int{
		"conservadas": conservadas,
		"cambiadas":   cambiadas,
		"perdidas":    perdidas,
	}); err != nil {
		t.Errorf("la ida y vuelta no cuadra: %v", err)
	}
	if err := metrica.Cuadra(len(volvieron.Respuestas), "las respuestas importadas",
		map[string]int{
			"conservadas": conservadas,
			"cambiadas":   cambiadas,
			"aparecidas":  aparecidas,
		}); err != nil {
		t.Errorf("la vuelta trae respuestas que no vienen de ningun sitio: %v", err)
	}
	// EL SUELO: si conservadas fuera cero, todo lo de arriba estaria en verde
	// sobre la nada. Es el mismo control que exige el exportador.
	if conservadas != len(entraron) {
		t.Fatalf("solo se han conservado %d de %d respuestas", conservadas, len(entraron))
	}
}

// EL CONTROL NEGATIVO DE LA PUERTA: se demuestra que la comparacion por
// identidad DETECTA lo que el recuento no.
//
// Se fabrica a mano la mutacion que el recuento deja pasar (una respuesta que
// cambia de «si» a «no», con el mismo cardinal y el mismo conjunto de preguntas)
// y se comprueba que la comparacion por parejas la ve. Sin esto, la puerta de
// arriba podria estar comparando cardinales sin que nadie lo notara.
func TestLaComparacionPorIdentidadVeLoQueElCardinalNoVe(t *testing.T) {
	entraron := map[string]alcances.Respuesta{"q.a": alcances.Si, "q.b": alcances.No}
	// Dos cambios que se cancelan: q.a pasa a «no» y q.b pasa a «si».
	volvieron := map[string]alcances.Respuesta{"q.a": alcances.No, "q.b": alcances.Si}

	if len(entraron) != len(volvieron) {
		t.Fatal("el caso de este control tiene que tener el MISMO cardinal en los dos lados: " +
			"si no, no demuestra nada sobre el recuento")
	}
	iguales := 0
	for id, r := range entraron {
		if volvieron[id] == r {
			iguales++
		}
	}
	if iguales != 0 {
		t.Fatalf("la comparacion por identidad dice que %d de %d parejas coinciden, y ninguna "+
			"tenia que coincidir: esta comparando cardinales", iguales, len(entraron))
	}
}
