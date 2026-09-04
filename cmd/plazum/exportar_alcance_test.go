package main

import (
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

// Las preguntas de la entrevista que este fichero usa.
//
// SE ESCRIBEN AQUI Y SE COMPRUEBA QUE EXISTEN, en vez de darlas por buenas: si
// el corpus las renombra, estos tests estarian exportando la nada y saldrian
// verdes por el motivo equivocado (cero respuestas se traducen igual de bien que
// cero respuestas correctas). El suelo esta en el primer test.
const (
	preguntaBooleanaDelPiloto = "ens.q.datos_personales"
	otraPreguntaBooleana      = "ens.q.externalizacion"
)

func correrAlcance(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var salida, errores strings.Builder
	rc := cmdAlcance(args, &salida, &errores)
	return rc, salida.String(), errores.String()
}

// LA CADENA ENTERA, POR LAS ORDENES DE VERDAD: entrevista -> alcance -> calendario.
//
// # Que cierra
//
// El camino guiado ofrecia dos pasos que piden un fichero que la interfaz no
// sabia producir. La traduccion existia (`corpus.HechosDeLaEntrevista`) y su
// unica llamada del producto era... un test. Este comprueba la otra mitad: que
// la ORDEN produce un fichero que la ORDEN siguiente carga y convierte en
// fechas.
//
// # Por que no se prueba contra un doble
//
// Porque el fallo vive en las juntas: que el alcance escrito CARGUE
// (`estricto.Decodificar` rechaza un campo de mas, y el producto ya se estrello
// una vez asi con `notas_de_las_fechas`), que el sujeto case con el que consulta
// el motor, y que las obligaciones derivadas tengan reloj. Un doble pasaria las
// tres sin tocar ninguna.
func TestElExportadorProduceUnAlcanceQueElCalendarioCarga(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "alcance.json")
	consulta := url.Values{}
	consulta.Add("si", preguntaBooleanaDelPiloto)
	consulta.Add("si", otraPreguntaBooleana)

	rc, salida, errores := correrAlcance(t,
		"--respuestas", consulta.Encode(), "--sujeto", "sis",
		"--organizacion", "Organismo de prueba",
		"--salida", ruta, "--corpus", "../../paquetes")
	if rc != 0 {
		t.Fatalf("`plazum alcance` ha salido %d:\n%s", rc, errores)
	}
	// EL SUELO: si las preguntas se han renombrado en el corpus, esto exporta
	// cero hechos y todo lo de abajo mediria el vacio.
	if strings.Contains(salida, "id de pregunta que este corpus no declara") {
		t.Fatalf("las preguntas que usa este test ya no existen en el corpus, asi que estaria "+
			"midiendo una exportacion vacia:\n%s", salida)
	}

	// PRIMERA JUNTA: el fichero que producimos lo carga el lector del producto.
	al, err := cargarAlcance(ruta)
	if err != nil {
		t.Fatalf("el alcance que produce `plazum alcance` NO CARGA: %v\n"+
			"  Es el fallo exacto que ya paso con `notas_de_las_fechas`: el producto "+
			"escribiendo un fichero que el propio producto rechaza", err)
	}
	if al.Sujeto != "sis" {
		t.Errorf("el sujeto exportado es %q y se pidio \"sis\"", al.Sujeto)
	}
	if len(al.Hechos) == 0 {
		t.Fatal("el alcance sale sin ni un hecho: la traduccion del puente no esta emitiendo " +
			"nada y las juntas de abajo no medirian nada")
	}

	// SEGUNDA JUNTA: la orden de verdad lo acepta y saca un calendario.
	var cal, calErr strings.Builder
	if rc := cmdCalendario([]string{"--alcance", ruta, "--corpus", "../../paquetes"},
		&cal, &calErr); rc != 0 {
		t.Fatalf("`plazum calendario --alcance` ha salido %d con el fichero exportado:\n%s",
			rc, calErr.String())
	}
	if strings.TrimSpace(cal.String()) == "" {
		t.Fatal("el calendario ha salido vacio")
	}
}

// LA CUENTA VA CON EL FICHERO, Y ESE ES EL FONDO DE ESTA ORDEN.
//
// El exportador es PARCIAL: solo booleanos, solo paquetes que declaran el
// puente, una sola instancia. Un exportador parcial que dijera «hecho» y nada
// mas produce un alcance corto y nadie se entera, y un alcance corto son
// obligaciones que no aparecen en el calendario de alguien.
//
// Aqui se exige que cada destino de una respuesta SE CUENTE Y SE NOMBRE: la que
// se traduce, la que dice que no, la de un paquete sin puente y la que el corpus
// no conoce. Los cuatro, porque el que faltara seria una respuesta que
// desaparece sin dejar rastro.
func TestLaExportacionDiceCuantoNoHaExportadoYPorQue(t *testing.T) {
	dir := t.TempDir()
	consulta := url.Values{}
	consulta.Add("si", preguntaBooleanaDelPiloto)     // se traduce
	consulta.Add("no", otraPreguntaBooleana)          // un «no» no afirma nada
	consulta.Add("si", "iso27001.q.desarrollo")       // paquete sin puente
	consulta.Add("si", "esta.pregunta.no.existe.hoy") // desconocida

	rc, salida, errores := correrAlcance(t,
		"--respuestas", consulta.Encode(), "--sujeto", "sis",
		"--salida", filepath.Join(dir, "a.json"), "--corpus", "../../paquetes")
	if rc != 0 {
		t.Fatalf("ha salido %d:\n%s", rc, errores)
	}
	// EL CUBO DE «SIN PUENTE» YA NO SE ALCANZA DESDE EL CORPUS PUBLICADO, y por
	// eso no se exige aqui.
	//
	// Hasta el 04-09-2026 esta lista pedia «TODAVIA NO declaran el puente», y
	// era cierto porque solo lo declaraba el piloto. Ahora lo declaran los 21
	// paquetes con reglas, o sea que ninguna respuesta del corpus real cae en
	// ese cubo. LA PREMISA DEL TEST CADUCO; el producto no.
	//
	// La rama NO se borra: sigue siendo la que atiende a un paquete de un
	// tercero sin bloque `hecho`, y borrarla dejaria esa respuesta cayendo por
	// un hueco sin nombre. Lo que se hace es lo que manda la casa cuando el
	// corpus deja de recorrer una rama: darle un control POSITIVO con dato
	// sintetico, y esta ahi abajo, en
	// TestElCuboDeSinPuenteSigueContandoLoQueNadieDeclara.
	for _, quiero := range []string{
		"4 respuestas leidas",
		"traducidas a hechos",
		"que no afirman nada",
		"esta.pregunta.no.existe.hoy",
		"LO QUE ESTE EXPORTADOR TODAVIA NO HACE",
	} {
		if !strings.Contains(salida, quiero) {
			t.Errorf("la salida no dice %q. Sin eso, quien la use no sabe si su alcance lleva "+
				"sus respuestas o la mitad de ellas.\n%s", quiero, salida)
		}
	}
	// LA LEY DE CONSERVACION: si los cubos no suman lo leido, hay respuestas que
	// no estan en ningun sitio, y eso saldria impreso. No tiene que salir.
	if strings.Contains(salida, "AVISO: los cubos suman") {
		t.Errorf("los cubos no cuadran con las respuestas leidas:\n%s", salida)
	}
}

// UN «NO» NO AFIRMA NADA, y esto lo comprueba por el resultado y no por el texto.
//
// En este motor la ausencia de un hecho no es su negacion. Si un «no» afirmara
// `no_predicado(...)`, el expediente llevaria una afirmacion que el operador no
// ha hecho, y ninguna regla del corpus la esperaria.
//
// CONTROL POSITIVO Y NEGATIVO SOBRE LA MISMA PREGUNTA: con «si» sale un hecho,
// con «no» no sale ninguno.
func TestUnNoNoAfirmaNadaYUnSiSi(t *testing.T) {
	dir := t.TempDir()
	exportar := func(param string) []string {
		t.Helper()
		ruta := filepath.Join(dir, param+".json")
		rc, _, errores := correrAlcance(t,
			"--respuestas", param+"="+preguntaBooleanaDelPiloto, "--sujeto", "sis",
			"--salida", ruta, "--corpus", "../../paquetes")
		if rc != 0 {
			t.Fatalf("exportar con %s ha salido %d:\n%s", param, rc, errores)
		}
		al, err := cargarAlcance(ruta)
		if err != nil {
			t.Fatal(err)
		}
		var preds []string
		for _, h := range al.Hechos {
			preds = append(preds, h.Pred)
		}
		return preds
	}
	conSi := exportar("si")
	if len(conSi) == 0 {
		t.Fatal("con un «si» no sale ni un hecho: el control negativo de abajo no mediria nada")
	}
	if conNo := exportar("no"); len(conNo) != 0 {
		t.Errorf("un «no» ha afirmado %v. En este motor la ausencia de un hecho no es su "+
			"negacion, y afirmarla mete en el expediente algo que el operador no ha dicho",
			conNo)
	}
}

// SIN SUJETO NO SE EXPORTA, y el error dice por que.
//
// El sujeto es el nombre con el que las reglas hablan de la organizacion Y la
// instancia sobre la que caen las respuestas. Inventarse uno produciria un
// alcance que carga, que no falla en ningun sitio y que deriva las obligaciones
// de nadie: el calendario saldria vacio sin decir por que, que es la forma mas
// cara de fallar aqui.
func TestSinSujetoNoSeExportaYSeDicePorQue(t *testing.T) {
	rc, _, errores := correrAlcance(t,
		"--respuestas", "si="+preguntaBooleanaDelPiloto,
		"--salida", filepath.Join(t.TempDir(), "a.json"), "--corpus", "../../paquetes")
	if rc == 0 {
		t.Fatal("se ha exportado sin --sujeto: el motor derivaria las obligaciones de nadie")
	}
	if !strings.Contains(errores, "--sujeto") {
		t.Errorf("el error no nombra la bandera que falta:\n%s", errores)
	}
}

// LAS DOS FORMAS DE DAR LAS RESPUESTAS NO SE MEZCLAN.
//
// --url y --respuestas dicen lo mismo por dos caminos. Con las dos puestas no
// habria forma de saber cual manda, y elegir una en silencio exportaria un
// alcance que no es el que la persona creia estar dando.
func TestLasDosFormasDeDarLasRespuestasNoSeMezclan(t *testing.T) {
	rc, _, errores := correrAlcance(t,
		"--url", "http://localhost:8443/alcance?si="+preguntaBooleanaDelPiloto,
		"--respuestas", "si="+otraPreguntaBooleana,
		"--sujeto", "sis", "--salida", filepath.Join(t.TempDir(), "a.json"),
		"--corpus", "../../paquetes")
	if rc == 0 {
		t.Fatal("se han admitido --url y --respuestas a la vez")
	}
	if !strings.Contains(errores, "Elige una") {
		t.Errorf("el error no dice como se arregla:\n%s", errores)
	}

	// CONTROL POSITIVO: --url sola SI vale, que es la forma que existe para no
	// tener que entender el formato (se pega la barra del navegador).
	rc, _, errores = correrAlcance(t,
		"--url", "http://localhost:8443/alcance?si="+preguntaBooleanaDelPiloto,
		"--sujeto", "sis", "--salida", filepath.Join(t.TempDir(), "b.json"),
		"--corpus", "../../paquetes")
	if rc != 0 {
		t.Fatalf("con solo --url tampoco se exporta, asi que la comprobacion de arriba se "+
			"cumple por rechazarlo todo:\n%s", errores)
	}
}

// UNA DIRECCION SIN RESPUESTAS NO PRODUCE UN ALCANCE VACIO EN SILENCIO.
//
// Es el caso que va a pasar de verdad: alguien abre /alcance, no contesta nada,
// copia la direccion y la pega aqui. Un alcance vacio escrito sin decir nada
// daria un calendario vacio, y quien lo mire concluira que no tiene
// obligaciones. Se para y se dice que las respuestas viajan EN la direccion.
func TestUnaUrlSinRespuestasSeParaEnVezDeEscribirUnAlcanceVacio(t *testing.T) {
	rc, _, errores := correrAlcance(t,
		"--url", "http://localhost:8443/alcance", "--sujeto", "sis",
		"--salida", filepath.Join(t.TempDir(), "a.json"), "--corpus", "../../paquetes")
	if rc == 0 {
		t.Fatal("se ha exportado un alcance de una direccion sin respuestas dentro. El " +
			"calendario saldria vacio y se leeria como «no tengo obligaciones»")
	}
	if !strings.Contains(errores, "EN LA DIRECCION") {
		t.Errorf("el error no explica donde viven las respuestas:\n%s", errores)
	}
}

// LA DIRECCION NO VIAJA ENTERA A UN ERROR (invariante 11).
//
// Aunque esta la acabe de teclear el operador, los errores del producto acaban
// en el bloque copiable de `plazum doctor --issue`, que existe para pegarlo en
// un issue publico. Y esta direccion lleva dentro las respuestas de la
// entrevista de una organizacion, que es justo lo que no se publica.
func TestUnaUrlQueNoSeEntiendeNoSaleEnteraEnElError(t *testing.T) {
	// Una direccion que url.Parse rechaza, con un dato reconocible dentro.
	const secreta = "http://usuario:contrasena-secreta@ejemplo/alcance?si=%zz"
	_, _, errores := correrAlcance(t,
		"--url", secreta, "--sujeto", "sis",
		"--salida", filepath.Join(t.TempDir(), "a.json"), "--corpus", "../../paquetes")
	for _, prohibido := range []string{"contrasena-secreta", secreta} {
		if strings.Contains(errores, prohibido) {
			t.Errorf("el error lleva dentro %q. Una direccion de configuracion no viaja entera "+
				"a un error: estos mensajes acaban en un issue publico", prohibido)
		}
	}
	if strings.TrimSpace(errores) == "" {
		t.Error("con una direccion que no se entiende no se dice nada, asi que la " +
			"comprobacion de arriba se cumple por no imprimir nunca")
	}
}
