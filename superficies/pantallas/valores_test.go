package pantallas

import (
	"net/url"
	"strings"
	"testing"
)

// LA ENTREVISTA CON VALORES, VISTA DESDE FUERA Y DESDE DENTRO.
//
// Lo que se prueba aqui no es que la pantalla pinte un desplegable bonito, es
// que NO AFIRME NADA QUE NADIE HAYA CONTESTADO, y eso tiene tres formas de
// fallar y cada una tiene su test:
//
//	afirmando por defecto        una opcion preseleccionada afirma sin que
//	                             nadie conteste;
//	afirmando lo que no se
//	entiende                     un valor presente y no interpretable tomado
//	                             por «sin contestar» (o peor, por bueno);
//	afirmando lo que no viene
//	del corpus                   un `v.<lo que sea>` de una peticion fabricada
//	                             entrando en el estado de la pantalla.

// NINGUNA OPCION VIENE ELEGIDA AL ABRIR LA PAGINA, y es LA propiedad.
//
// Es la razon de que un enumerado se pinte como enlaces y no como desplegable.
// Con un `<select>`, quien abre la pagina y no contesta nada esta mandando la
// primera opcion en cuanto pulse cualquier boton de la pagina: con
// `designado(E,"entidad_financiera")` eso enciende 28 obligaciones y una es
// NOTIFICATORIA ante el supervisor, y una actuacion indebida ante el supervisor
// no se deshace.
func TestAlAbrirLaEntrevistaNingunValorEstaElegido(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	_, cuerpo := pedir(t, s, "/alcance")

	bloque := bloqueDePregunta(t, cuerpo, "alfa.q.categoria")
	if strings.Contains(bloque, "elegido") {
		t.Errorf("al abrir la pagina hay una opcion marcada como elegida:\n%s\n"+
			"  Un valor por defecto afirma sin que nadie conteste, y esta pantalla no puede "+
			"afirmar nada que el operador no haya dicho", bloque)
	}
	// Y EL CONTROL POSITIVO: cuando SI se contesta, se marca. Sin esto, una
	// pantalla que no marcara nunca nada pasaria el test de arriba.
	enlaces := enlacesQueResponden(t, cuerpo, "alfa.q.categoria")
	_, despues := pedir(t, s, enlaces[0])
	if !strings.Contains(bloqueDePregunta(t, despues, "alfa.q.categoria"), "elegido") {
		t.Error("tras contestar, ninguna opcion sale marcada. Entonces el test de arriba " +
			"pasa porque esta pantalla no marca nunca nada, no porque no afirme por defecto")
	}
	// Y el campo libre nace vacio por lo mismo.
	libre := bloqueDePregunta(t, cuerpo, "alfa.q.nombre")
	if !strings.Contains(libre, `value=""`) {
		t.Errorf("el campo de texto no nace vacio:\n%s", libre)
	}
}

// LAS TRES FORMAS DE LA NADA EN LA LECTURA DE LA DIRECCION (invariante 8).
//
// Las tres se recorren y las tres dan resultados DISTINTOS. Si dieran lo mismo,
// la tercera se estaria tomando por la nada, que es inventarse un valor.
func TestLasTresFormasDeLaNadaEnUnValor(t *testing.T) {
	m := derivarModelo(corpusDemo())
	const id = "alfa.q.categoria"

	for _, caso := range []struct {
		nombre string
		q      url.Values
		quiere EstadoDelValor
		valor  string
	}{
		{"ausente", url.Values{}, ValorAusente, ""},
		{"presente y vacio", url.Values{ClaveValor(id): {""}}, ValorSinContestar, ""},
		{"presente y solo espacios", url.Values{ClaveValor(id): {"   "}}, ValorSinContestar, ""},
		{"presente y bueno", url.Values{ClaveValor(id): {"MEDIA"}}, ValorPuesto, "MEDIA"},
		{"presente y NO interpretable",
			url.Values{ClaveValor(id): {"ALTISIMA"}}, ValorNoInterpretable, ""},
		{"presente dos veces",
			url.Values{ClaveValor(id): {"BAJA", "ALTA"}}, ValorContradictorio, ""},
		{"presente dos veces con el mismo valor",
			url.Values{ClaveValor(id): {"BAJA", "BAJA"}}, ValorContradictorio, ""},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			r := De(caso.q, m.preguntas, m.voc)
			if got := r.EstadoDelValorDe(id); got != caso.quiere {
				t.Errorf("estado %v y se esperaba %v", got, caso.quiere)
			}
			if got := r.Valor(id); got != caso.valor {
				t.Errorf("valor %q y se esperaba %q", got, caso.valor)
			}
			// LO QUE NO SE ENTIENDE NO AFIRMA. Ni como el valor bueno, ni como
			// «sin contestar» con efectos: no produce respuesta.
			if r.EstadoDelValorDe(id).EsError() && r.Valor(id) != "" {
				t.Error("un valor que no se entiende se ha conservado, asi que puede acabar " +
					"reflejado en un enlace o en la pagina")
			}
		})
	}
}

// UN VALOR QUE NO SE ENTIENDE NI SE PINTA NI VIAJA, y se dice que llego.
//
// Las dos mitades importan y son distintas: no reflejar lo que mando el cliente
// (que es la mitad de un XSS reflejado, y no hay razon para tener esa mitad) y
// DECIR que algo llego, porque callarlo lo haria indistinguible de no haber
// contestado, que es justo lo que el invariante 8 prohibe.
func TestUnValorQueNoSeEntiendeNiSePintaNiViajaPeroSeDice(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	const basura = "ESTA-CADENA-NO-ES-UN-VALOR-DECLARADO"
	_, cuerpo := pedir(t, s, "/alcance?"+ClaveValor("alfa.q.categoria")+"="+basura)

	if strings.Contains(cuerpo, basura) {
		t.Errorf("la pagina devuelve el valor que llego en la peticion. Un valor que no se "+
			"ha podido comprobar no vuelve a salir: es la mitad de un XSS reflejado, y "+
			"ademas se arrastraria a cada clic siguiente.\n%s",
			bloqueDePregunta(t, cuerpo, "alfa.q.categoria"))
	}
	if !strings.Contains(cuerpo, rotulo("es", "alcance.valores.no_se_entienden")) {
		t.Error("la pagina no dice que ha llegado una respuesta que no se entiende. " +
			"Callarlo la hace indistinguible de no haber contestado, y quien la mando " +
			"creeria que no llego")
	}
	// Y NO CUENTA COMO RESPONDIDA. Si contara, la barra de progreso diria que la
	// entrevista avanza con una respuesta que no se ha usado.
	m := derivarModelo(corpusDemo())
	r := De(url.Values{ClaveValor("alfa.q.categoria"): {basura}}, m.preguntas, m.voc)
	if r.Respondidas() != 0 {
		t.Errorf("una respuesta que no se entiende cuenta como respondida (%d)",
			r.Respondidas())
	}
}

// NINGUN ENLACE LLEVA UN `v.` VACIO, Y ESTE TEST LO ENCONTRO UNA MUTACION.
//
// # Como salio, y por que se queda
//
// La linea de `Consulta()` que solo copia el valor cuando AFIRMA se muto a «copia
// siempre que el parametro estuviera presente» y NADA SE PUSO ROJO: como un
// valor que no se entiende no se conserva, lo que se copiaba era la cadena
// vacia, y el contenido de la pagina no cambiaba. La mutacion no cazada es el
// hallazgo, no el fallo.
//
// Lo que si cambia, y no lo miraba nadie, es LA DIRECCION, que en esta
// superficie es el artefacto que se comparte y se guarda en marcadores:
//
//	dejar una pregunta sin contestar dejaba un `v.<id>=` pegado a cada enlace
//	de la pagina PARA SIEMPRE, o sea que «deshacer» no devolvia al estado
//	limpio;
//	y ese parametro vacio se vuelve a leer como «el operador eligio no
//	contestar», que es una afirmacion que nadie hizo: la nada de verdad es que
//	el parametro NO ESTE.
func TestNingunEnlaceLlevaUnValorVacio(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	const id = "alfa.q.categoria"

	for _, caso := range []struct{ nombre, ruta string }{
		{"tras deshacer", "/alcance?" + ClaveValor(id) + "="},
		{"con un valor que no se entiende", "/alcance?" + ClaveValor(id) + "=NO_ES_UN_VALOR"},
		{"con dos valores", "/alcance?" + ClaveValor(id) + "=BAJA&" + ClaveValor(id) + "=ALTA"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			_, cuerpo := pedir(t, s, caso.ruta)
			for _, m := range reEnlace.FindAllStringSubmatch(cuerpo, -1) {
				u, err := url.Parse(strings.ReplaceAll(m[1], "&amp;", "&"))
				if err != nil {
					continue
				}
				for k, vs := range u.Query() {
					if !strings.HasPrefix(k, ParamValor+".") {
						continue
					}
					for _, v := range vs {
						if strings.TrimSpace(v) == "" {
							t.Errorf("el enlace %q lleva %s vacio.\n"+
								"  La nada es que el parametro NO ESTE: uno vacio se lee "+
								"despues como «el operador eligio no contestar», que es una "+
								"afirmacion que nadie hizo, y ademas se queda pegado a todos "+
								"los enlaces para siempre", m[1], k)
						}
					}
				}
			}
		})
	}

	// CONTROL POSITIVO: cuando el valor SI se entiende, los enlaces lo llevan.
	// Sin esto, una pantalla que no copiara nunca ningun valor pasaria el test
	// de arriba, y ademas perderia la entrevista en cada clic.
	_, bueno := pedir(t, s, "/alcance?"+ClaveValor(id)+"=ALTA")
	visto := false
	for _, m := range reEnlace.FindAllStringSubmatch(bueno, -1) {
		u, err := url.Parse(strings.ReplaceAll(m[1], "&amp;", "&"))
		if err != nil {
			continue
		}
		for _, v := range u.Query()[ClaveValor(id)] {
			if v == "ALTA" {
				visto = true
			}
		}
	}
	if !visto {
		t.Error("ningun enlace lleva el valor contestado, asi que el test de arriba pasa " +
			"porque esta pantalla no copia nunca ningun valor: cada clic perderia la " +
			"entrevista entera")
	}
}

// UN `v.` DE UNA PREGUNTA QUE EL CORPUS NO DECLARA NO ENTRA EN EL ESTADO.
//
// Es la misma guarda que ya tenian `si` y `no`, y la razon es la misma: lo que
// decide que preguntas hay es el corpus instalado, no la peticion. Sin ella, una
// direccion fabricada mete claves en el estado de la pantalla, que se copian a
// cada enlace que la pagina pinta.
func TestUnValorDeUnaPreguntaInventadaNoEntra(t *testing.T) {
	m := derivarModelo(corpusDemo())
	q := url.Values{
		ClaveValor("no.existe.esta.pregunta"): {"ALTA"},
		ClaveValor("alfa.q.categoria"):        {"ALTA"},
	}
	r := De(q, m.preguntas, m.voc)
	if r.EstadoDelValorDe("no.existe.esta.pregunta") != ValorAusente {
		t.Error("una pregunta que el corpus no declara ha entrado en el estado de la pantalla")
	}
	c := r.Consulta()
	if len(c[ClaveValor("no.existe.esta.pregunta")]) > 0 {
		t.Errorf("la consulta reconstruida arrastra la pregunta inventada: %v", c)
	}
	// CONTROL POSITIVO: la que SI declara el corpus sigue ahi. Sin esto, una
	// lectura que descartara todo pasaria este test.
	if r.Valor("alfa.q.categoria") != "ALTA" {
		t.Error("la pregunta que el corpus SI declara tampoco ha entrado, asi que esto no " +
			"esta midiendo el filtro sino una lectura que descarta todo")
	}
}

// UN VALOR SOBRE UNA PREGUNTA DE SI/NO ES UN DATO QUE NO SE ENTIENDE.
//
// Un booleano no tiene donde poner un valor. Aceptarlo seria dar por buena una
// respuesta que no se puede comprobar contra nada, que es el lado permisivo del
// valor cero (invariante 8): sin vocabulario, se rechaza.
func TestUnValorSobreUnaPreguntaDeSiNoNoSeEntiende(t *testing.T) {
	m := derivarModelo(corpusDemo())
	r := De(url.Values{ClaveValor("beta.q.riesgo"): {"lo que sea"}}, m.preguntas, m.voc)
	if got := r.EstadoDelValorDe("beta.q.riesgo"); got != ValorNoInterpretable {
		t.Errorf("un valor sobre un booleano sale %v y tenia que salir ValorNoInterpretable",
			got)
	}
	// Y CON EL VOCABULARIO VACIO, TODO VALOR SE RECHAZA. Es el valor cero de la
	// estructura de opciones que cruza la frontera, y tiene que ser el
	// restrictivo: sin lista con la que comprobar, no se acepta nada.
	sinVoc := De(url.Values{ClaveValor("alfa.q.categoria"): {"MEDIA"}},
		m.preguntas, Vocabulario{})
	if got := sinVoc.EstadoDelValorDe("alfa.q.categoria"); got != ValorNoInterpretable {
		t.Errorf("con el vocabulario en su valor cero, un valor sale %v. El valor cero de "+
			"unas opciones que cruzan una frontera tiene que ser el RESTRICTIVO: sin lista "+
			"con la que comprobar, dar el valor por bueno es el lado permisivo", got)
	}
}

// UN VALOR Y UN SI SOBRE LA MISMA PREGUNTA NO AFIRMAN NADA.
//
// Son dos respuestas distintas a la misma pregunta y elegir una en silencio
// seria afirmar un alcance que nadie afirmo. Pasa con un enlace fabricado y con
// uno viejo mezclado con uno nuevo.
func TestUnValorYUnSiSobreLaMismaPreguntaSonContradictorios(t *testing.T) {
	m := derivarModelo(corpusDemo())
	q := url.Values{
		ParamSi:                        {"alfa.q.categoria"},
		ClaveValor("alfa.q.categoria"): {"ALTA"},
	}
	r := De(q, m.preguntas, m.voc)
	if got := r.EstadoDelValorDe("alfa.q.categoria"); got != ValorContradictorio {
		t.Errorf("estado %v y tenia que ser ValorContradictorio", got)
	}
	if r.Valor("alfa.q.categoria") != "" {
		t.Error("con las dos formas de contestar a la vez se ha elegido una en silencio")
	}
	if !r.SinContestar("alfa.q.categoria") {
		t.Error("una pregunta contestada de dos formas distintas cuenta como contestada. " +
			"Lo unico honesto con una entrada que se contradice es darla por sin responder")
	}
}

// UNA PREGUNTA CON VALOR NO OFRECE «NO», Y ESO ES UNA CORRECCION.
//
// Hasta esta rebanada, un «no» a «¿que categoria alcanza el sistema?» escondia
// todas las obligaciones que dependen de la categoria. Absolver de golpe por una
// respuesta que nadie puede dar en serio es el error caro: el que acusa lo
// corrige quien lee, el que absuelve lo descubre el inspector.
func TestUnaPreguntaConValorNoOfreceNo(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	_, cuerpo := pedir(t, s, "/alcance")
	for _, u := range enlacesQueResponden(t, cuerpo, "alfa.q.categoria") {
		v, err := url.Parse(strings.ReplaceAll(u, "&amp;", "&"))
		if err != nil {
			t.Fatal(err)
		}
		for _, x := range v.Query()[ParamNo] {
			if x == "alfa.q.categoria" {
				t.Errorf("la pregunta con valor ofrece un enlace de «no»: %s.\n"+
					"  Un «no» a una pregunta que pide CUAL no significa nada, y esconde de "+
					"golpe todas las obligaciones que dependen de ella", u)
			}
		}
	}
	// CONTROL POSITIVO: la de si/no SI lo ofrece. Sin esto, el test pasaria en
	// una pantalla que no pintara ningun «no» en ninguna parte.
	if _, no := enlacesDePregunta(t, cuerpo, "beta.q.riesgo"); no == "" {
		t.Error("la pregunta de si/no tampoco ofrece «no»")
	}
}

// EL VALOR SOBREVIVE A LA NAVEGACION, Y EL CAMPO LIBRE NO PIERDE LO ANTERIOR.
//
// Un formulario GET DESCARTA la consulta de su `action`, asi que si los campos
// ocultos faltaran, escribir un texto borraria el resto de la entrevista. Es el
// fallo que nadie mira porque el formulario «funciona».
func TestElCampoLibreNoPierdeLoQueYaEstabaContestado(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	// SE PIDE LA LISTA LARGA a proposito. En la corta, contestar las tres
	// preguntas deja dormidas a las que ya no deciden nada y desaparecen de la
	// pagina: se comprobaria que no se ven, no que no se han perdido.
	partida := "/alcance?" + ParamVer + "=" + VerTodas +
		"&" + ClaveValor("alfa.q.categoria") + "=ALTA&" + ParamSi + "=beta.q.riesgo"
	_, cuerpo := pedir(t, s, partida)

	const texto = "el sistema de facturacion"
	destino := respuestaQueOfreceLaPagina(t, cuerpo, "alfa.q.nombre", texto)
	_, despues := pedir(t, s, destino)

	// LAS TRES RESPUESTAS SIGUEN AHI, cada una en la forma en que la pinta su
	// tipo: el texto en el valor del campo, la opcion marcada, y el si/no con
	// su rotulo de respondida.
	if !strings.Contains(bloqueDePregunta(t, despues, "alfa.q.nombre"), `value="`+texto+`"`) {
		t.Errorf("el texto que se acaba de escribir no vuelve en el campo:\n%s",
			bloqueDePregunta(t, despues, "alfa.q.nombre"))
	}
	if !strings.Contains(bloqueDePregunta(t, despues, "alfa.q.categoria"), "elegido") {
		t.Errorf("tras escribir en el campo libre se ha perdido la respuesta con valor de "+
			"otra pregunta.\n"+
			"  Un formulario GET descarta la consulta de su action: si los ocultos no "+
			"llevan la entrevista dentro, contestar una pregunta borra las demas\n%s",
			bloqueDePregunta(t, despues, "alfa.q.categoria"))
	}
	if !strings.Contains(despues, rotulo("es", "alcance.pregunta.respondida_si")) {
		t.Error("tras escribir en el campo libre se ha perdido la respuesta de si/no")
	}
	// Y EL PROGRESO LAS CUENTA LAS TRES. Es la comprobacion que no depende de
	// como se pinte cada tipo.
	m := derivarModelo(corpusDemo())
	u, err := url.Parse(destino)
	if err != nil {
		t.Fatal(err)
	}
	if n := De(u.Query(), m.preguntas, m.voc).Respondidas(); n != 3 {
		t.Errorf("tras contestar tres preguntas el progreso dice %d", n)
	}
}

// UNA DIRECCION QUE SOLO LLEVA VALORES MANDA SOBRE LA CUENTA.
//
// La regla de esta superficie es «si la direccion trae respuestas, mandan las de
// la direccion». Mientras una respuesta solo podia ser un si o un no, mirar esos
// dos parametros era mirarlas todas. Con la tercera forma, un enlace compartido
// que llevara SOLO valores se leia como «la direccion no trae nada» y la pantalla
// ensenaba lo de la cuenta de quien abriera el enlace, comiendose las del enlace
// sin decirlo.
func TestUnEnlaceQueSoloLlevaValoresMandaSobreLaCuenta(t *testing.T) {
	al := nuevoAlmacenFalso()
	if err := al.Responder(t.Context(), "ciso", "beta.q.riesgo", Si); err != nil {
		t.Fatal(err)
	}
	s, _ := superficie(t, corpusDemo(), conGuardado(al, "ciso"))

	// Sin nada en la direccion se ve lo de la cuenta.
	_, propio := pedir(t, s, "/alcance")
	if !strings.Contains(propio, rotulo("es", "alcance.guardado.en_tu_cuenta")) {
		t.Fatal("sin nada en la direccion no se esta ensenando lo de la cuenta, asi que " +
			"esta comparacion no mide nada")
	}

	// Con SOLO un valor en la direccion, manda la direccion.
	_, delEnlace := pedir(t, s, "/alcance?"+ClaveValor("alfa.q.categoria")+"=ALTA")
	if !strings.Contains(delEnlace, rotulo("es", "alcance.guardado.desde_enlace")) {
		t.Error("una direccion que solo lleva valores no se reconoce como respondida, asi " +
			"que la pantalla ensena lo de la cuenta y se come lo del enlace sin decirlo")
	}
	if !strings.Contains(delEnlace, "ALTA") {
		t.Error("el valor del enlace no se esta pintando")
	}
}

// LO QUE EL GUARDADO NO SE LLEVA SE DICE, CON SU CARDINAL.
//
// El almacen guarda una respuesta como Si o como No y un valor no cabe ahi.
// Guardar la mitad en silencio es la peor version de ese boton: quien pulsa
// «guardar» y vuelve manana encuentra su entrevista a medias sin una linea que
// se lo explique.
func TestLaPantallaDiceQueElGuardadoNoSeLlevaLosValores(t *testing.T) {
	al := nuevoAlmacenFalso()
	s, _ := superficie(t, corpusDemo(), conGuardado(al, "ciso"))

	_, conValor := pedir(t, s, "/alcance?"+ClaveValor("alfa.q.categoria")+"=ALTA")
	if !strings.Contains(conValor, rotulo("es", "alcance.derivacion.valores_sin_guardar")) {
		t.Error("con un valor contestado y la cuenta abierta, la pantalla no dice que el " +
			"guardado no se lo lleva")
	}
	// CONTROL POSITIVO EN LA OTRA DIRECCION: sin valores no se dice nada, para
	// que el aviso signifique algo cuando salga.
	_, soloSiNo := pedir(t, s, "/alcance?"+ParamSi+"=beta.q.riesgo")
	if strings.Contains(soloSiNo, rotulo("es", "alcance.derivacion.valores_sin_guardar")) {
		t.Error("sin ninguna respuesta con valor la pantalla avisa igual, asi que el aviso " +
			"es ruido y quien lo lea dejara de leerlo")
	}
}

// LAS CUATRO CLASES DE CAMPO SALEN DEL TIPO QUE DECLARA EL CORPUS, y el valor
// cero es el que no afirma.
func TestElTipoDeCampoSaleDelCorpusYSuValorCeroEsElRestrictivo(t *testing.T) {
	m := derivarModelo(corpusDemo())
	for _, caso := range []struct {
		id     string
		quiere TipoDeCampo
	}{
		{"alfa.q.categoria", CampoOpcion},
		{"alfa.q.nombre", CampoTexto},
		{"beta.q.riesgo", CampoSiNo},
		{"no.existe", CampoSiNo}, // lo desconocido cae en el que no afirma solo
	} {
		if got := m.voc.Tipo(caso.id); got != caso.quiere {
			t.Errorf("%s sale %v y tenia que salir %v", caso.id, got, caso.quiere)
		}
	}
	// El valor cero del vocabulario entero: nada se sabe, todo es si/no.
	var cero Vocabulario
	if cero.Tipo("alfa.q.categoria") != CampoSiNo {
		t.Error("el valor cero del vocabulario no deja las preguntas en si/no, que es la " +
			"unica forma que no afirma cuando no se pulsa")
	}
	if len(cero.Valores("alfa.q.categoria")) != 0 {
		t.Error("el valor cero del vocabulario ofrece valores que no ha leido de ningun sitio")
	}
}

// UN VALOR DEMASIADO LARGO NO SE RECORTA: NO SE ENTIENDE.
//
// Recortar en silencio cambia el dato que mandaron por otro, que es la misma
// familia que tomar lo no interpretable por la nada.
func TestUnValorDemasiadoLargoNoSeRecorta(t *testing.T) {
	m := derivarModelo(corpusDemo())
	largo := strings.Repeat("a", MaxLargoValor+1)
	r := De(url.Values{ClaveValor("alfa.q.nombre"): {largo}}, m.preguntas, m.voc)
	if got := r.EstadoDelValorDe("alfa.q.nombre"); got != ValorNoInterpretable {
		t.Errorf("un valor de %d caracteres sale %v y tenia que salir ValorNoInterpretable",
			len(largo), got)
	}
	if r.Valor("alfa.q.nombre") != "" {
		t.Error("se ha conservado un trozo del valor largo: recortar en silencio cambia el " +
			"dato que mandaron por otro")
	}
	// CONTROL POSITIVO: uno del largo maximo si entra.
	justo := strings.Repeat("a", MaxLargoValor)
	if r2 := De(url.Values{ClaveValor("alfa.q.nombre"): {justo}},
		m.preguntas, m.voc); r2.Valor("alfa.q.nombre") != justo {
		t.Error("un valor del largo maximo tampoco entra, asi que el limite esta mal puesto")
	}
}

// UNA FECHA QUE NO ES UNA FECHA NO ES UNA FECHA, y un entero que no es un
// entero tampoco. Es el `Atoi` con el error descartado, que convierte tres
// cosas distintas en el mismo cero.
func TestUnaFechaOUnEnteroQueNoSeEntiendenSonError(t *testing.T) {
	// Se construye el vocabulario a mano porque el corpus sintetico base no
	// tiene ni fecha ni entero, y el punto es el interprete, no el corpus.
	for _, caso := range []struct {
		nombre string
		campo  campoDePregunta
		bueno  string
		malos  []string
	}{
		{"fecha", campoDePregunta{tipo: CampoFecha}, "2026-01-15",
			[]string{"ayer", "15/01/2026", "2026-13-45", "2026-01", "2026-01-15T00:00:00Z"}},
		{"entero", campoDePregunta{tipo: CampoEntero}, "7",
			[]string{"muchos", "7.5", "7 personas", "0x7"}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			q := url.Values{ClaveValor("x"): {caso.bueno}}
			if _, e := leerValorOpcional(q, "x", caso.campo); e != ValorPuesto {
				t.Fatalf("el valor bueno %q sale %v: este test estaria midiendo un "+
					"interprete que lo rechaza todo", caso.bueno, e)
			}
			for _, malo := range caso.malos {
				q := url.Values{ClaveValor("x"): {malo}}
				v, e := leerValorOpcional(q, "x", caso.campo)
				if e != ValorNoInterpretable || v != "" {
					t.Errorf("%q sale (%q, %v) y tenia que salir no interpretable.\n"+
						"  Tomarlo por la nada, o por el cero, es inventarse un valor: son "+
						"tres cosas distintas y solo dos son la nada", malo, v, e)
				}
			}
		})
	}
}

// leerValorObligatorio ES OTRA FUNCION, y las tres formas de la nada colapsan
// ahi en la misma respuesta a proposito.
//
// Un campo obligatorio y uno opcional son dos preguntas distintas: en el
// opcional hay que distinguir «no contestaste» de «contestaste algo que no se
// entiende», y en el obligatorio lo unico que hace falta saber es si hay valor
// utilizable. Meterlas en una con valor por defecto es por donde se cuela el
// cero.
func TestElCampoObligatorioYElOpcionalSeLeenConDosFunciones(t *testing.T) {
	c := campoDePregunta{tipo: CampoOpcion, valido: map[string]bool{"ALTA": true},
		valores: []string{"ALTA"}}
	for _, caso := range []struct {
		nombre   string
		q        url.Values
		opcional EstadoDelValor
		vale     bool
	}{
		{"ausente", url.Values{}, ValorAusente, false},
		{"vacio", url.Values{ClaveValor("x"): {""}}, ValorSinContestar, false},
		{"basura", url.Values{ClaveValor("x"): {"NO_ES"}}, ValorNoInterpretable, false},
		{"bueno", url.Values{ClaveValor("x"): {"ALTA"}}, ValorPuesto, true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, e := leerValorOpcional(caso.q, "x", c); e != caso.opcional {
				t.Errorf("la lectura opcional da %v y se esperaba %v", e, caso.opcional)
			}
			v, ok := leerValorObligatorio(caso.q, "x", c)
			if ok != caso.vale {
				t.Errorf("la lectura obligatoria da %v y se esperaba %v", ok, caso.vale)
			}
			if !ok && v != "" {
				t.Errorf("la lectura obligatoria devuelve %q sin darlo por bueno: quien la "+
					"llame podria usarlo", v)
			}
		})
	}
}
