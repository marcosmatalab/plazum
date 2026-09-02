package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// LA CADENA ENTERA: de la entrevista al calendario, por el CLI de verdad.
//
// # Que cierra
//
// El camino guiado ofrece dos ordenes que piden un fichero que la interfaz no
// sabia producir (`plazum calendario --alcance` y `plazum escalado --alcance`).
// El hueco NO era el boton de descarga: era que la entrevista y el motor
// hablaban idiomas distintos y no habia traduccion declarada en ninguna parte.
//
// Con el puente declarado por el paquete (nucleo/corpus/puente.go) la traduccion
// existe y esta validada contra las reglas. Lo que este test comprueba es la
// otra mitad, que es la que de verdad le importa a quien compra: que lo que sale
// de ahi SIRVE, o sea que se escribe como alcance.json, lo carga el comando de
// verdad y produce un calendario con fechas dentro.
//
// # Por que no se prueba contra un doble
//
// Porque el fallo que se busca vive justo en las juntas: que el alcance escrito
// cargue (`estricto.Decodificar` rechaza un campo de mas y el producto ya se
// estrello una vez asi, con `notas_de_las_fechas`), que el sujeto case con el
// que consulta el motor, y que las obligaciones derivadas tengan reloj. Un doble
// pasaria las tres sin tocar ninguna.
//
// # Lo que este test NO dice
//
// No dice que `plazum serve` ya exporte esto: la pantalla todavia no sabe
// preguntar valores, y esa es la parte A. Dice que **cuando lo haga, la cadena
// de detras esta probada**, que es lo que evita construir la pantalla contra un
// formato que despues no carga.

// alcanceDelPuente escribe un alcance.json a partir de la entrevista traducida
// por el producto. Es, literalmente, lo que hara el exportador.
func alcanceDelPuente(t *testing.T, dir, nombre string, soloBooleanos bool) string {
	t.Helper()
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatalf("cargar el corpus: %v", err)
	}
	var piloto *corpus.Paquete
	for _, p := range ps {
		if p.DeclaraPuente() {
			piloto = p
			break
		}
	}
	if piloto == nil {
		t.Fatal("ningun paquete declara el puente: este test mediria el vacio")
	}

	// El MISMO escenario que mide el piloto no se puede importar (vive en el
	// paquete de test de la raiz), asi que se escribe el minimo que enciende
	// obligaciones CON RELOJ, que es lo que este test necesita comprobar.
	// LOS BOOLEANOS Y LOS QUE LLEVAN VALOR, SEPARADOS. La separacion no es
	// cosmetica: es lo que permite comprobar que la mitad con valor hace algo,
	// y esa mitad es la que un fallo de aridad rompe en silencio.
	booleanos := []corpus.RespuestaDeEntrevista{
		{Entidad: "sistema", Instancia: "sis", Atributo: "trata_datos_personales", Si: true},
		{Entidad: "sistema", Instancia: "sis", Atributo: "servicios_externalizados", Si: true},
	}
	conValor := []corpus.RespuestaDeEntrevista{
		{Entidad: "sistema", Instancia: "sis", Atributo: "ambito", Valor: "sector_publico"},
		{Entidad: "informacion", Instancia: "inf", Atributo: "manejada_por_el_sistema", Valor: "sis"},
		{Entidad: "informacion", Instancia: "inf", Atributo: "nivel_confidencialidad", Valor: "ALTO"},
		{Entidad: "informacion", Instancia: "inf", Atributo: "nivel_integridad", Valor: "ALTO"},
		{Entidad: "informacion", Instancia: "inf", Atributo: "nivel_disponibilidad", Valor: "ALTO"},
		{Entidad: "informacion", Instancia: "inf", Atributo: "nivel_autenticidad", Valor: "ALTO"},
		{Entidad: "informacion", Instancia: "inf", Atributo: "nivel_trazabilidad", Valor: "ALTO"},
	}
	entrevista := booleanos
	if !soloBooleanos {
		entrevista = append(append([]corpus.RespuestaDeEntrevista{}, booleanos...), conValor...)
	}
	hechos, err := corpus.HechosDeLaEntrevista(piloto, entrevista)
	if err != nil {
		t.Fatalf("traduciendo la entrevista: %v", err)
	}
	if len(hechos) == 0 {
		t.Fatal("la traduccion no ha dado ni un hecho: este test estaria midiendo un alcance vacio")
	}

	// SE ESCRIBE CON LA MISMA ESTRUCTURA QUE LEE cargarAlcance, y eso es lo que
	// se esta probando: `estricto.Decodificar` rechaza un campo de mas, y el
	// producto ya escribio una vez un fichero que despues no cargaba.
	type hechoJSON struct {
		Pred string   `json:"pred"`
		Args []string `json:"args"`
	}
	doc := map[string]any{
		"organizacion": "Organismo de prueba",
		"sujeto":       "sis",
		"descripcion":  "alcance derivado de la entrevista por el puente declarado del paquete",
		"hechos":       []hechoJSON{},
	}
	var hs []hechoJSON
	for _, h := range hechos {
		hs = append(hs, hechoJSON{Pred: h.Pred, Args: h.Args})
	}
	doc["hechos"] = hs

	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(dir, nombre)
	if err := os.WriteFile(ruta, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return ruta
}

// TestUnAlcanceSacadoDelPuenteCargaYDaCalendario recorre la cadena entera.
func TestUnAlcanceSacadoDelPuenteCargaYDaCalendario(t *testing.T) {
	ruta := alcanceDelPuente(t, t.TempDir(), "alcance.json", false)

	// PRIMERA JUNTA: el fichero que producimos lo carga el lector del producto.
	al, err := cargarAlcance(ruta)
	if err != nil {
		t.Fatalf("el alcance que produce el puente NO CARGA: %v\n"+
			"  Es el fallo exacto que ya paso con `notas_de_las_fechas`: el producto "+
			"escribiendo un fichero que el propio producto rechaza", err)
	}
	if al.Sujeto == "" || len(al.Hechos) == 0 {
		t.Fatalf("el alcance carga vacio: sujeto=%q hechos=%d", al.Sujeto, len(al.Hechos))
	}

	// SEGUNDA JUNTA: la orden de verdad lo acepta y saca un calendario.
	var salida, errores strings.Builder
	rc := cmdCalendario([]string{"--alcance", ruta, "--corpus", "../../paquetes"},
		&salida, &errores)
	if rc != 0 {
		t.Fatalf("`plazum calendario --alcance` ha salido %d con el alcance del puente:\n%s",
			rc, errores.String())
	}
	texto := salida.String()
	if strings.TrimSpace(texto) == "" {
		t.Fatal("el calendario ha salido vacio")
	}

	// TERCERA JUNTA, Y LA QUE IMPORTA: el calendario trae obligaciones DE
	// VERDAD, no una pagina que dice que no hay nada. Es la diferencia entre
	// «la cadena funciona» y «la cadena no revienta».
	if strings.Contains(texto, "0 obligaciones") {
		t.Errorf("el calendario dice cero obligaciones con el alcance del puente puesto.\n"+
			"  Entonces la traduccion produce hechos que el motor no casa, que es justo lo "+
			"que el puente existe para impedir.\n%s", recorta(texto, 600))
	}
	// Al menos una fecha con formato de fecha: sin esto, un calendario que
	// listara obligaciones sin reloj pasaria por bueno.
	if !strings.Contains(texto, "202") {
		t.Errorf("el calendario no trae ni una fecha. Las obligaciones derivadas no tienen "+
			"reloj, o no se estan pintando:\n%s", recorta(texto, 600))
	}
}

// CONTROL NEGATIVO: sin los hechos del puente, el mismo comando NO deriva lo
// mismo. Sin esto, el test de arriba pasaria aunque el calendario ensenara el
// corpus entero por cualquier otra razon.
func TestSinLosHechosDelPuenteElCalendarioNoDerivaLoMismo(t *testing.T) {
	dir := t.TempDir()
	conPuente := alcanceDelPuente(t, dir, "entero.json", false)

	// El mismo alcance, con el mismo sujeto, y SIN hechos.
	pelado := filepath.Join(dir, "pelado.json")
	b, err := json.MarshalIndent(map[string]any{
		"organizacion": "Organismo de prueba",
		"sujeto":       "sis",
		"descripcion":  "el mismo sujeto, sin una sola respuesta de la entrevista",
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pelado, b, 0o600); err != nil {
		t.Fatal(err)
	}

	medir := func(ruta string) string {
		var salida, errores strings.Builder
		if rc := cmdCalendario([]string{"--alcance", ruta, "--corpus", "../../paquetes"},
			&salida, &errores); rc != 0 {
			t.Fatalf("calendario con %s ha salido %d: %s", ruta, rc, errores.String())
		}
		return salida.String()
	}
	conHechos, sinHechos := medir(conPuente), medir(pelado)
	if conHechos == sinHechos {
		t.Error("el calendario sale IGUAL con las respuestas de la entrevista y sin ellas. " +
			"Entonces lo que pinta no depende de lo que el operador contesta, y el test de " +
			"arriba no esta midiendo el puente")
	}
	if len(conHechos) <= len(sinHechos) {
		t.Errorf("con las respuestas puestas el calendario no es mayor que sin ellas "+
			"(%d frente a %d caracteres): el puente no esta anadiendo obligaciones",
			len(conHechos), len(sinHechos))
	}
}

func recorta(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// EL CONTROL QUE FALTABA, Y LO ENCONTRO UNA MUTACION.
//
// La primera version de este fichero comprobaba tres cosas del calendario: que
// no saliera vacio, que trajera fechas y que fuera distinto del de un alcance
// sin hechos. LAS TRES SEGUIAN SIENDO CIERTAS con la mitad del puente rota.
//
// La mutacion: en corpus.HechosDeLaEntrevista, que un atributo `con_valor`
// emitiera `predicado(instancia)` en vez de `predicado(instancia, valor)`. El
// piloto de la raiz se puso rojo de golpe (de 25 obligaciones a 6), y ESTE
// FICHERO SIGUIO VERDE, porque los booleanos solos ya bastaban para que el
// calendario tuviera obligaciones y fechas.
//
// O sea que era una puerta de humo: comprobaba que la cadena no revienta, no
// que la cadena LLEVE lo que dice llevar. Es la familia de siempre, y la caza
// la pregunta fija: si una comprobacion recorre una lista, hay que preguntar si
// la direccion contraria tambien se recorre.
//
// Se cierra con el tercer punto de comparacion: entero > solo booleanos > nada.
func TestLaMitadConValorDelPuenteAnadeObligaciones(t *testing.T) {
	dir := t.TempDir()
	entero := alcanceDelPuente(t, dir, "entero.json", false)
	soloBool := alcanceDelPuente(t, dir, "solo-booleanos.json", true)

	medir := func(ruta string) string {
		var salida, errores strings.Builder
		if rc := cmdCalendario([]string{"--alcance", ruta, "--corpus", "../../paquetes"},
			&salida, &errores); rc != 0 {
			t.Fatalf("calendario con %s ha salido %d: %s", ruta, rc, errores.String())
		}
		return salida.String()
	}
	conTodo, soloConBooleanos := medir(entero), medir(soloBool)

	if conTodo == soloConBooleanos {
		t.Error("el calendario sale IGUAL con las respuestas que llevan valor y sin ellas.\n" +
			"  Entonces la mitad `con_valor` del puente no esta afirmando nada que el motor " +
			"case, y las otras comprobaciones de este fichero seguirian verdes porque los " +
			"booleanos solos ya dan calendario.")
	}
	if len(conTodo) <= len(soloConBooleanos) {
		t.Errorf("con las respuestas de valor puestas el calendario no crece (%d frente a %d "+
			"caracteres): la mitad con valor no anade obligaciones",
			len(conTodo), len(soloConBooleanos))
	}
	// Y LOS BOOLEANOS SOLOS TIENEN QUE DAR ALGO, o esta comparacion seria entre
	// un calendario y una pagina vacia, que se cumple por el motivo equivocado.
	if strings.TrimSpace(soloConBooleanos) == "" {
		t.Fatal("con solo los booleanos el calendario sale vacio: la comparacion de arriba " +
			"no estaria midiendo lo que cree")
	}
}
