package main

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/superficies/pantallas"
)

// LO QUE LA ENTREVISTA CONTESTA CON UN VALOR LLEGA AL MOTOR, Y SE MIDE.
//
// # El agujero que cierran estos tests
//
// El exportador traducia si/no y apartaba a un cubo toda respuesta cuyo atributo
// declara la forma `con_valor`: la pantalla no sabia preguntar el valor, asi que
// no habia valor que poner. Eso son 25 de las 68 preguntas del corpus, y su
// efecto no es una pantalla incompleta sino un alcance CORTO, o sea obligaciones
// que no aparecen presentadas como si no tocaran.
//
// Ahora la pantalla las pregunta y viajan como `v.<id>=<valor>`. Estos tests
// comprueban las dos mitades que importan: que producen HECHOS CON SU VALOR
// DENTRO, y que un valor que no se entiende NO se toma por «sin contestar».

// preguntaConValorDelCorpus es una pregunta de forma `con_valor` con sus valores.
//
// SE BUSCA EN EL CORPUS EN VEZ DE ESCRIBIRLA, y no es comodidad: escrita a mano,
// el dia que el corpus la renombre este test exportaria cero hechos y saldria
// verde por el motivo equivocado, que es la trampa que el resto de este fichero
// ya tenia identificada. Se casa por (paquete, entidad, atributo), que es la
// misma clave con la que el puente busca la declaracion.
func preguntaConValorDelCorpus(t *testing.T) (id string, valores []string) {
	t.Helper()
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatalf("cargar el corpus: %v", err)
	}
	type clave struct{ urn, entidad, atributo string }
	attr := map[clave]corpus.Atributo{}
	for _, p := range ps {
		for _, e := range p.Entidades {
			for _, a := range e.Atributos {
				attr[clave{p.URN, e.Nombre, a.Nombre}] = a
			}
		}
	}
	for _, q := range corpus.Entrevista(ps) {
		a, hay := attr[clave{q.Paquete, q.Entidad, q.Atributo}]
		if !hay || a.Hecho == nil || a.Hecho.Forma != corpus.PuenteConValor {
			continue
		}
		if a.Tipo == corpus.Enumerado && len(a.Valores) > 0 {
			return q.ID, a.Valores
		}
	}
	t.Fatal("el corpus instalado no trae ni una pregunta de enumerado con forma " +
		"`con_valor`. Este fichero estaria midiendo el vacio: sin ella no hay nada que " +
		"la entrevista pueda contestar con un valor")
	return "", nil
}

// TestUnaRespuestaConValorProduceUnHechoConSuValorDentro es la casilla.
func TestUnaRespuestaConValorProduceUnHechoConSuValorDentro(t *testing.T) {
	id, valores := preguntaConValorDelCorpus(t)
	dir := t.TempDir()
	ruta := filepath.Join(dir, "alcance.json")

	consulta := url.Values{}
	consulta.Set(pantallas.ClaveValor(id), valores[0])

	rc, salida, errores := correrAlcance(t,
		"--respuestas", consulta.Encode(), "--sujeto", "sis",
		"--organizacion", "Organismo de prueba",
		"--salida", ruta, "--corpus", "../../paquetes")
	if rc != 0 {
		t.Fatalf("`plazum alcance` con un valor ha salido %d:\n%s\n%s", rc, salida, errores)
	}

	b, err := os.ReadFile(ruta) // #nosec G304 -- la ruta la compone el propio test
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Hechos []struct {
			Pred string   `json:"pred"`
			Args []string `json:"args"`
		} `json:"hechos"`
		Respuestas []struct {
			Valor    string `json:"valor"`
			Pregunta string `json:"pregunta"`
		} `json:"respuestas"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}

	// EL HECHO TIENE DOS ARGUMENTOS Y EL SEGUNDO ES EL VALOR. Es la unica forma
	// de distinguir «la cadena no revienta» de «la cadena LLEVA lo que dice
	// llevar»: con un hecho de un solo argumento el fichero se escribe igual, el
	// calendario carga igual, y no casa con ninguna regla.
	conValor := 0
	for _, h := range doc.Hechos {
		if len(h.Args) == 2 && h.Args[1] == valores[0] {
			conValor++
		}
	}
	if conValor == 0 {
		t.Errorf("la respuesta %q=%q no ha producido ningun hecho con ese valor dentro.\n"+
			"  hechos: %+v\n"+
			"  Sin el segundo argumento el hecho no casa con ninguna regla y no da error en\n"+
			"  ningun sitio, que es el agujero entero que esta rebanada cierra",
			id, valores[0], doc.Hechos)
	}

	// Y EL REGISTRO DE LO CONTESTADO GUARDA EL VALOR, no un «si». Sin esto la
	// ida y la vuelta perderia la respuesta: volveria como un booleano.
	visto := false
	for _, r := range doc.Respuestas {
		if r.Pregunta == id {
			visto = true
			if r.Valor != valores[0] {
				t.Errorf("el bloque de respuestas guarda %q para %q y se contesto %q. La "+
					"vuelta devolveria otra respuesta", r.Valor, id, valores[0])
			}
		}
	}
	if !visto {
		t.Errorf("la respuesta %q no aparece en el bloque de respuestas del fichero, asi que "+
			"la vuelta la perderia entera", id)
	}
}

// TestUnValorQueNoSeEntiendeNoSeToma es LA TERCERA FORMA DE LA NADA, y no es la
// nada (invariante 8).
//
// Los tres casos se recorren aqui, y los tres dan resultados DISTINTOS, que es
// justo lo que hay que demostrar:
//
//	ausente                el parametro no viene. Se exporta sin ese hecho, y
//	                       con codigo 0: no contestar es legitimo.
//	presente y vacio       el parametro viene vacio. Igual que ausente: es el
//	                       «deshacer» de la pantalla.
//	presente y NO interpretable   un valor que el paquete no declara. NO se toma
//	                       por «sin contestar»: se para, no se escribe fichero y
//	                       se dice cual.
//
// Si los tres dieran lo mismo, el tercero se estaria tomando por la nada, que es
// inventarse un valor del operador.
func TestUnValorQueNoSeEntiendeNoSeToma(t *testing.T) {
	id, valores := preguntaConValorDelCorpus(t)

	for _, caso := range []struct {
		nombre   string
		valores  []string // lo que se pone en la consulta
		quiereRC int
	}{
		// La ausencia se representa sin tocar la consulta: ver mas abajo.
		{"presente y vacio", []string{""}, 0},
		{"presente y no interpretable", []string{"ESTE_VALOR_NO_LO_DECLARA_NINGUN_PAQUETE"}, 1},
		{"presente dos veces con dos valores", []string{valores[0], "OTRO"}, 1},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			dir := t.TempDir()
			ruta := filepath.Join(dir, "alcance.json")
			consulta := url.Values{}
			// SIEMPRE VIAJA ADEMAS UNA RESPUESTA BUENA. Sin ella, el caso vacio
			// se pararia por «esta direccion no trae ninguna respuesta» y
			// estariamos midiendo esa guarda y no esta.
			consulta.Add("si", preguntaBooleanaDelPiloto)
			for _, v := range caso.valores {
				consulta.Add(pantallas.ClaveValor(id), v)
			}
			rc, salida, errores := correrAlcance(t,
				"--respuestas", consulta.Encode(), "--sujeto", "sis",
				"--salida", ruta, "--corpus", "../../paquetes")
			if rc != caso.quiereRC {
				t.Fatalf("ha salido %d y se esperaba %d\n%s\n%s", rc, caso.quiereRC,
					salida, errores)
			}
			if caso.quiereRC == 0 {
				return
			}
			// NO SE ESCRIBE NADA. Un fichero a medias es peor que ninguno: se
			// carga sin error y deriva menos obligaciones.
			if _, err := os.Stat(ruta); err == nil {
				t.Error("se ha escrito el alcance con una respuesta que no se entiende " +
					"dentro. Un alcance al que le falta justo lo que el operador creia " +
					"haber contestado deriva menos obligaciones y no lo dice")
			}
			// Y SE DICE CUAL, porque «algo no se entiende» sin decir que es una
			// pantalla que no se puede arreglar.
			if !strings.Contains(errores, id) {
				t.Errorf("el error no nombra la pregunta %q:\n%s", id, errores)
			}
		})
	}

	// EL CONTROL POSITIVO DE LA RAMA QUE ABSUELVE: con el parametro AUSENTE la
	// misma orden sale bien.
	//
	// Sin esto, un exportador que se parara SIEMPRE pasaria los tres casos de
	// arriba y no habria forma de saberlo. Es la comprobacion de que lo que se
	// esta midiendo es el valor que no se entiende y no la orden entera.
	t.Run("ausente", func(t *testing.T) {
		dir := t.TempDir()
		ruta := filepath.Join(dir, "alcance.json")
		consulta := url.Values{}
		consulta.Add("si", preguntaBooleanaDelPiloto)
		rc, salida, errores := correrAlcance(t,
			"--respuestas", consulta.Encode(), "--sujeto", "sis",
			"--salida", ruta, "--corpus", "../../paquetes")
		if rc != 0 {
			t.Fatalf("sin el parametro la orden falla (%d), asi que los casos de arriba no "+
				"estan midiendo el valor sino la orden:\n%s\n%s", rc, salida, errores)
		}
		if _, err := os.Stat(ruta); err != nil {
			t.Errorf("sin el parametro no se ha escrito el alcance: %v", err)
		}
	})
}

// LA PUERTA DE --cuenta AVISA DE LO QUE NO PUEDE TRAER, y --url no avisa.
//
// # El agujero que esto tapa, y salio en la pasada del comprador
//
// Un CISO que contesta la entrevista entera en el navegador y despues exporta
// con `--cuenta` se lleva SOLO los si/no: el almacen guarda una respuesta como
// Si o como No y un valor no cabe ahi. Y esas respuestas no aparecen ni siquiera
// como un cubo de la cuenta, porque nunca llegaron a la consulta: son AUSENTES,
// y ausente es una respuesta legitima.
//
// O sea que es la unica puerta capaz de producir un alcance corto SIN QUE NINGUN
// CARDINAL LO DIGA. El aviso es lo unico que la separa de absolver en silencio.
//
// LAS DOS DIRECCIONES, porque un aviso que sale siempre es ruido: sale por
// --cuenta y NO sale por --url, que si las trae.
func TestLaPuertaDeLaCuentaAvisaDeLoQueNoPuedeTraer(t *testing.T) {
	id, valores := preguntaConValorDelCorpus(t)
	datos := t.TempDir()

	// Se mete una respuesta en la cuenta por la puerta de la vuelta, que es la
	// que el producto ofrece, en vez de fabricar el fichero del almacen a mano.
	origen := filepath.Join(datos, "origen.json")
	consulta := url.Values{}
	consulta.Add("si", preguntaBooleanaDelPiloto)
	if rc, _, errores := correrAlcance(t,
		"--respuestas", consulta.Encode(), "--sujeto", "sis",
		"--salida", origen, "--corpus", "../../paquetes"); rc != 0 {
		t.Fatalf("preparando el alcance de origen: %d\n%s", rc, errores)
	}
	if rc, _, errores := correrAlcance(t, "--importar", origen, "--cuenta", "ciso",
		"--datos", datos, "--corpus", "../../paquetes"); rc != 0 {
		t.Fatalf("metiendo las respuestas en la cuenta: %d\n%s", rc, errores)
	}

	rc, _, errores := correrAlcance(t, "--cuenta", "ciso", "--datos", datos,
		"--sujeto", "sis", "--salida", filepath.Join(datos, "a.json"),
		"--corpus", "../../paquetes")
	if rc != 0 {
		t.Fatalf("exportando desde la cuenta: %d\n%s", rc, errores)
	}
	if !strings.Contains(errores, "se contestan con un VALOR") {
		t.Errorf("exportando desde la cuenta no se avisa de que las respuestas con valor no "+
			"vienen.\n  Es la unica puerta que puede dar un alcance corto sin que ningun "+
			"cubo lo diga, porque esas respuestas no llegan a la consulta y «ausente» es "+
			"una respuesta legitima.\n%s", errores)
	}

	// LA OTRA DIRECCION: por --url, que SI las trae, no se avisa. Un aviso que
	// sale siempre es ruido, y quien lo lea dejara de leerlo.
	conValor := url.Values{}
	conValor.Add("si", preguntaBooleanaDelPiloto)
	conValor.Set(pantallas.ClaveValor(id), valores[0])
	rc, _, errores = correrAlcance(t, "--respuestas", conValor.Encode(),
		"--sujeto", "sis", "--salida", filepath.Join(datos, "b.json"),
		"--corpus", "../../paquetes")
	if rc != 0 {
		t.Fatalf("exportando desde la direccion: %d\n%s", rc, errores)
	}
	if strings.Contains(errores, "se contestan con un VALOR") {
		t.Errorf("por --url, que SI trae las respuestas con valor, se avisa igual:\n%s",
			errores)
	}
}

// TestElCuboDeSinPuenteSigueContandoLoQueNadieDeclara es el CONTROL POSITIVO de
// una rama que el corpus publicado ya no recorre.
//
// # Por que hace falta, y por que no vale con borrar la rama
//
// Hasta el 04-09-2026 el corpus tenia paquetes sin bloque `hecho` y el cubo
// `SinPuente` se llenaba solo. Ahora lo declaran los 21 paquetes con reglas, o
// sea que ninguna respuesta del corpus real cae ahi. Una rama que ninguna
// entrada alcanza es una rama que no existe, y la mutacion la deja verde porque
// no hay nada que romper: es M47.
//
// Borrarla seria peor. Esta superficie carga los paquetes que haya INSTALADOS,
// y el bloque `hecho` es opcional a proposito: un paquete de un tercero o uno a
// medio escribir puede no traerlo, y entonces sus respuestas tienen que salir
// contadas y con el nombre de su paquete delante, no caerse por un hueco.
//
// Asi que se recorre con dato sintetico, llamando al exportador directamente.
func TestElCuboDeSinPuenteSigueContandoLoQueNadieDeclara(t *testing.T) {
	sinPuente := &corpus.Paquete{
		URN: "urn:demo:sin-puente", Version: "2026.1", Clase: corpus.Propio,
		Entidades: []corpus.TipoEntidad{{
			Nombre: "sistema", Descripcion: "un sistema",
			Atributos: []corpus.Atributo{{
				Nombre: "cosa", Tipo: corpus.Booleano, Cita: "demo art. 1",
				// SIN bloque `hecho`: es exactamente el caso.
			}},
		}},
		Preguntas: []corpus.Pregunta{{
			ID: "sinpuente.q.cosa", Texto: "Pasa la cosa", Cita: "demo art. 1",
			Entidad: "sistema", Atributo: "cosa"}},
	}
	consulta := url.Values{}
	consulta.Add("si", "sinpuente.q.cosa")

	_, cuenta, err := exportarAlcance([]*corpus.Paquete{sinPuente}, consulta, "sis", "Acme")
	if err != nil {
		t.Fatalf("exportar: %v", err)
	}
	if cuenta.SinPuente[sinPuente.URN] != 1 {
		t.Errorf("la respuesta de un paquete sin bloque `hecho` no ha caido en el cubo de "+
			"«sin puente»: %+v.\n"+
			"  Una respuesta que desaparece sin dejar rastro es la forma mas cara de fallar "+
			"aqui: obligaciones que no aparecen y nadie sabe por que", cuenta)
	}
	if cuenta.Traducidas != 0 {
		t.Errorf("se ha traducido %d respuesta(s) de un paquete que no declara el puente. "+
			"Eso solo se puede hacer inventandose el predicado, que es lo que el puente "+
			"vino a cerrar", cuenta.Traducidas)
	}
	// LA LEY DE CONSERVACION TAMBIEN AQUI.
	if cuenta.Suma() != cuenta.Leidas {
		t.Errorf("los cubos suman %d y se leyo %d", cuenta.Suma(), cuenta.Leidas)
	}
}

// TestUnaPreguntaContestadaQueSiYQueNoNoAfirmaNada: la misma pregunta con `si` y
// con `no` en la misma direccion.
//
// # El fallo que esto cierra, y estaba en el producto
//
// El exportador recorria `si` y despues `no` por separado, asi que `si=X&no=X`
// producia el hecho de X Y ADEMAS lo contaba como negativa. La pantalla, en
// cambio, la trata como contradictoria y la da por sin responder. O sea que el
// fichero AFIRMABA lo que la pantalla presentaba como no contestado, y las dos
// mitades del producto decian cosas distintas de la misma direccion.
//
// Se arregla leyendo con la MISMA funcion que lee la pantalla.
func TestUnaPreguntaContestadaQueSiYQueNoNoAfirmaNada(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "alcance.json")
	consulta := url.Values{}
	consulta.Add("si", preguntaBooleanaDelPiloto)
	consulta.Add("no", preguntaBooleanaDelPiloto)
	consulta.Add("si", otraPreguntaBooleana) // para que el fichero no salga vacio

	rc, salida, errores := correrAlcance(t,
		"--respuestas", consulta.Encode(), "--sujeto", "sis",
		"--salida", ruta, "--corpus", "../../paquetes")
	if rc != 0 {
		t.Fatalf("ha salido %d:\n%s\n%s", rc, salida, errores)
	}
	b, err := os.ReadFile(ruta) // #nosec G304 -- la ruta la compone el propio test
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Respuestas []struct {
			Pregunta string `json:"pregunta"`
		} `json:"respuestas"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	for _, r := range doc.Respuestas {
		if r.Pregunta == preguntaBooleanaDelPiloto {
			t.Errorf("la pregunta %q llega contestada que si Y que no, y aun asi aparece en "+
				"el fichero.\n"+
				"  La pantalla la da por sin responder. Que el fichero afirme lo que la "+
				"pantalla presenta como no contestado son dos juicios distintos sobre la "+
				"misma direccion, y el que gana es el que afirma de mas",
				preguntaBooleanaDelPiloto)
		}
	}

	// CONTROL POSITIVO: la OTRA pregunta, contestada una sola vez, si esta. Sin
	// esto, un exportador que no escribiera nunca el bloque de respuestas
	// pasaria este test.
	otra := false
	for _, r := range doc.Respuestas {
		if r.Pregunta == otraPreguntaBooleana {
			otra = true
		}
	}
	if !otra {
		t.Errorf("la pregunta %q, contestada una sola vez, tampoco aparece: entonces esto no "+
			"esta midiendo la contradiccion, esta midiendo un bloque vacio", otraPreguntaBooleana)
	}
}
