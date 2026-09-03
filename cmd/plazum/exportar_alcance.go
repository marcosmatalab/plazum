package main

// `plazum alcance`: EL EXPORTADOR QUE CONVIERTE LA ENTREVISTA EN alcance.json.
//
// # El hueco que cierra, y donde estaba exactamente
//
// El camino guiado tenia dos pasos que piden un fichero que la interfaz no sabia
// producir: `plazum calendario --alcance` y `plazum escalado --alcance`. Quien
// respondia la entrevista en el navegador se encontraba con que sus respuestas
// vivian en la direccion de la pagina y no habia forma de convertirlas en el
// fichero que pedia el paso siguiente. El unico camino era escribir el JSON a
// mano, adivinando el nombre de los predicados, que es justo lo que el puente
// declarado por el paquete existe para que nadie tenga que hacer.
//
// La traduccion ya existia (`corpus.HechosDeLaEntrevista`) y estaba probada de
// punta a punta en `puente_e2e_test.go`, escribiendo un alcance.json, pasandolo
// por `cargarAlcance` y por `cmdCalendario`. Esa funcion de test ERA el
// exportador. Aqui esta, en el producto.
//
// # ESTE EXPORTADOR ES PARCIAL, y el cardinal va delante
//
// Se dice aqui, se dice en la ayuda y se dice en la salida de cada ejecucion,
// porque un exportador que finge exportar la entrevista entera produce un
// alcance corto y nadie se entera: obligaciones que no aparecen.
//
//	SOLO LOS BOOLEANOS. La entrevista web pregunta si/no y nada mas: sus
//	respuestas viajan como `si=<id>` y `no=<id>`. Los atributos CON VALOR (la
//	categoria de un sistema, el nivel de una informacion) no se pueden
//	contestar todavia en la pantalla, asi que este exportador no los emite.
//	Cuando la pantalla aprenda a preguntarlos, lo que hara es pasar por aqui.
//
//	SOLO LOS PAQUETES QUE DECLARAN EL PUENTE. Hoy es UNO, el piloto. El bloque
//	`hecho` es opcional mientras dura el piloto, y una respuesta sobre un
//	atributo cuyo paquete no lo declara NO se traduce: se cuenta y se dice de
//	que paquete es. Inventarse el predicado por convencion («se llama como el
//	atributo») es exactamente lo que el puente vino a cerrar.
//
//	UNA SOLA INSTANCIA. La entrevista no pregunta por instancias (el ENS
//	pregunta por CADA informacion y CADA servicio), asi que todas las
//	respuestas caen sobre el sujeto. Con varias informaciones eso las pisaria,
//	y por eso el sujeto se pide explicito en vez de inventarse uno.
//
// # Un «no» no afirma nada, y eso NO es una perdida
//
// En este motor la ausencia de un hecho no es su negacion. Hacer que un «no»
// afirmara `no_predicado(...)` meteria en el expediente una afirmacion que el
// operador no ha hecho. Los «no» se cuentan y se dicen; lo que no hacen es
// escribir nada.

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"

	"github.com/marcosmatalab/plazum/nucleo/aplicabilidad"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/pantalla"
	"github.com/marcosmatalab/plazum/superficies/pantallas"
)

const ayudaAlcance = `plazum alcance: convierte las respuestas de la entrevista en alcance.json.

  plazum alcance --cuenta ciso --sujeto sis --organizacion "Acme SL"
  plazum alcance --url "http://localhost:8443/alcance?si=X&si=Y&no=Z" \
                 --sujeto sis --organizacion "Acme SL" [--salida alcance.json]
  plazum alcance --respuestas "si=X&si=Y" --sujeto sis --organizacion "Acme SL"
  plazum alcance --importar alcance.json --cuenta ciso        (la vuelta)

  --cuenta       saca las respuestas de las que TIENES GUARDADAS en esa cuenta,
                 las que contestaste en el navegador. Es la forma que no obliga
                 a copiar ninguna direccion. Con --importar, es la cuenta en la
                 que se meten las respuestas del fichero.
  --url          pega la direccion ENTERA de la barra del navegador cuando
                 tengas la entrevista respondida. Es la forma que no obliga a
                 entender el formato.
  --respuestas   solo la parte de la consulta, si prefieres componerla tu.
  --importar     LA VUELTA: lee el bloque de respuestas de un alcance.json y lo
                 guarda en la cuenta que diga --cuenta, para seguir contestando
                 desde el navegador. Dice cuantas ha metido y cuantas no, y por
                 que.
  --datos        directorio de datos de la instalacion, donde vive el fichero de
                 respuestas guardadas. Por defecto ".", igual que en serve.
  --sujeto       el nombre con el que las reglas hablan de tu organizacion. Sin
                 el, el motor deriva las obligaciones de nadie.
  --organizacion como se llama, para que salga en el calendario.
  --salida       donde se escribe. Por defecto alcance.json.
  --corpus       directorio de paquetes instalados. Por defecto "paquetes".

  ESTE EXPORTADOR ES PARCIAL Y CADA EJECUCION DICE CUANTO. Hoy traduce los
  booleanos de los paquetes que declaran el puente, y dice cuantas respuestas
  ha dejado fuera y por que. Un exportador que fingiera exportarlo todo
  produciria un alcance corto sin que nadie se enterara, y eso son obligaciones
  que no aparecen.

  Despues:
    plazum calendario --alcance alcance.json
    plazum escalado   --alcance alcance.json
    plazum serve      --alcance alcance.json
`

// alcanceExportado es la forma que se escribe en disco.
//
// SE ESCRIBE CON LOS MISMOS NOMBRES QUE LEE cargarAlcance, y eso no se da por
// hecho: `estricto.Decodificar` rechaza un campo de mas, y el producto ya
// escribio una vez un fichero que despues el propio producto no cargaba (fue
// `notas_de_las_fechas`). Hay un test que hace la ida y la vuelta.
type alcanceExportado struct {
	Organizacion string `json:"organizacion"`
	Sujeto       string `json:"sujeto"`
	Descripcion  string `json:"descripcion"`
	// Respuestas es LO QUE SE CONTESTO, y Hechos es LO QUE SE AFIRMA. No son lo
	// mismo y por eso son dos bloques.
	//
	// # Por que hacia falta el primero, y por que la vuelta no existia sin el
	//
	// El bloque de hechos es DELIBERADAMENTE CON PERDIDAS: un «no» no afirma
	// nada (en este motor la ausencia de un hecho no es su negacion) y una
	// respuesta de un paquete que todavia no declara el puente tampoco. Eso esta
	// bien para el motor y hace IMPOSIBLE la vuelta: de un alcance.json con solo
	// hechos no se pueden recuperar las respuestas, porque la mitad de ellas no
	// dejaron rastro. Quien exportara e importara veria desaparecer sus «no» sin
	// una linea en ningun sitio.
	//
	// El bloque de respuestas es el registro de lo contestado, entero, y es lo
	// que hace que la ida y la vuelta CONSERVEN. El campo por el que casa al
	// volver es `pregunta`, el identificador que declara el paquete y que viaja
	// tambien en la direccion de la pantalla; ni por posicion, ni por orden, ni
	// por el nombre del campo, que puede repetirse entre preguntas.
	//
	// Los nombres son LOS QUE YA LEIA `cargarAlcance` (campo, valor, pregunta):
	// el formato admitia este bloque desde el principio y el exportador no lo
	// escribia.
	Respuestas []respuestaDeJSON `json:"respuestas"`
	Hechos     []hechoDeJSON     `json:"hechos"`
}

type respuestaDeJSON struct {
	Campo    string `json:"campo"`
	Valor    string `json:"valor"`
	Pregunta string `json:"pregunta"`
}

type hechoDeJSON struct {
	Pred string   `json:"pred"`
	Args []string `json:"args"`
}

// CuentaDeLaExportacion es el cardinal de lo que ha pasado, entero.
//
// LA LEY DE CONSERVACION APLICADA A UNA EXPORTACION: toda respuesta leida cae
// en exactamente uno de los cubos. Si la suma no cuadra hay respuestas que no
// estan en ningun sitio, y eso lo tiene que ver quien ejecuta la orden, porque
// significa que su alcance esta corto y no sabe por que.
type CuentaDeLaExportacion struct {
	// Leidas son las respuestas que traia la direccion, si y no.
	Leidas int
	// Desconocidas son ids de pregunta que el corpus instalado no declara.
	// Vienen de una direccion vieja o de otro corpus, y NO se descartan en
	// silencio.
	Desconocidas []string
	// Negativas son los «no». No afirman nada a proposito.
	Negativas int
	// SinPuente son las respuestas cuyo ATRIBUTO no declara el puente, contadas
	// por paquete. Es el cardinal de lo que este exportador todavia no puede.
	//
	// SE MIRA EL ATRIBUTO Y NO EL PAQUETE, y ese cambio arreglo un fallo real.
	// Antes se preguntaba `p.DeclaraPuente()`, que es cierto en cuanto UN
	// atributo del paquete lo declara: una respuesta sobre otro atributo del
	// mismo paquete pasaba el filtro, llegaba al puente y lo hacia FALLAR
	// ENTERO, con un mensaje escrito para quien cablea. Un exportador que se
	// cae no exporta a medias: no exporta nada.
	SinPuente map[string]int
	// ConValor son las respuestas de un atributo CON VALOR (una categoria, un
	// nivel) que la entrevista ha contestado con un si.
	//
	// SE CUENTAN Y SE DICEN, no se traducen y no revientan. El puente de esos
	// atributos afirma `predicado(instancia, valor)` y la entrevista web solo
	// sabe preguntar si/no, asi que no hay valor que poner: mandarlas al puente
	// lo hacia fallar entero. Es la misma clase que SinPuente (una capacidad
	// que falta, dicha con su cardinal) y no un dato roto.
	//
	// SALIO ESCRIBIENDO LA IDA Y VUELTA, no leyendo el codigo: exportar las
	// respuestas GUARDADAS de una cuenta puede tocar cualquier pregunta que la
	// entrevista pinte, y la entrevista pinta tambien estas. Por la puerta de
	// --url el fallo existia igual desde el primer dia y nadie lo habia pisado
	// porque los tests elegian preguntas booleanas.
	ConValor []string
	// Traducidas son las que han producido hechos.
	Traducidas int
	// Hechos son los hechos emitidos. Puede ser distinto de Traducidas: un
	// atributo declarado `no_llega_al_motor` se recoge y no afirma nada.
	Hechos int
}

// Suma es lo que tiene que cuadrar con Leidas.
func (c CuentaDeLaExportacion) Suma() int {
	n := len(c.Desconocidas) + c.Negativas + c.Traducidas + len(c.ConValor)
	for _, v := range c.SinPuente {
		n += v
	}
	return n
}

func cmdAlcance(args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("plazum alcance", flag.ContinueOnError)
	fs.SetOutput(errores)
	fs.Usage = func() { fmt.Fprint(errores, ayudaAlcance) }
	direccion := fs.String("url", "", "la direccion entera de la entrevista respondida")
	respuestas := fs.String("respuestas", "", "solo la parte de consulta (si=X&no=Y)")
	laCuenta := fs.String("cuenta", "", "cuenta de la que salen (o a la que entran) las respuestas guardadas")
	importar := fs.String("importar", "", "alcance.json del que recuperar las respuestas hacia la cuenta")
	datos := fs.String("datos", ".", "directorio de datos de la instalacion")
	sujeto := fs.String("sujeto", "", "nombre con el que las reglas hablan de tu organizacion")
	organizacion := fs.String("organizacion", "", "como se llama tu organizacion")
	rutaSalida := fs.String("salida", "alcance.json", "donde se escribe el alcance")
	dirCorpus := fs.String("corpus", "paquetes", "directorio de paquetes instalados")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	// LA VUELTA VA PRIMERO Y SE EXCLUYE DE TODO LO DEMAS. Con --importar no se
	// escribe ningun alcance: se leen respuestas de uno y se meten en la cuenta.
	if strings.TrimSpace(*importar) != "" {
		return cmdImportarAlcance(*importar, *laCuenta, *datos, *dirCorpus, salida, errores)
	}

	consulta, codigo := consultaDeLaEntrevista(fuenteDeLasRespuestas{
		direccion: *direccion, consulta: *respuestas, cuenta: *laCuenta, datos: *datos,
	}, errores)
	if codigo != 0 {
		return codigo
	}
	if strings.TrimSpace(*sujeto) == "" {
		fmt.Fprintln(errores, "falta --sujeto.")
		fmt.Fprintln(errores, "  Es el nombre con el que las reglas de aplicabilidad hablan de tu")
		fmt.Fprintln(errores, "  organizacion, y ademas la instancia sobre la que caen tus respuestas.")
		fmt.Fprintln(errores, "  No se inventa uno: con el sujeto equivocado, el motor deriva las")
		fmt.Fprintln(errores, "  obligaciones de nadie y el calendario sale vacio sin decir por que.")
		return 2
	}

	ps, err := corpus.Cargar(strings.TrimSpace(*dirCorpus))
	if err != nil {
		fmt.Fprintf(errores, "el corpus de %s no carga: %v\n", *dirCorpus, err)
		return 1
	}
	if len(ps) == 0 {
		fmt.Fprintf(errores, "no hay ni un paquete en %s. Sin corpus no hay preguntas que "+
			"traducir. Prueba `plazum demo` primero.\n", *dirCorpus)
		return 1
	}

	doc, cuenta, err := exportarAlcance(ps, consulta, strings.TrimSpace(*sujeto),
		strings.TrimSpace(*organizacion))
	if err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}

	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}
	if err := escribirConFsync(strings.TrimSpace(*rutaSalida), append(b, '\n')); err != nil {
		fmt.Fprintf(errores, "el alcance NO se ha escrito en %s: %v\n", *rutaSalida, err)
		return 1
	}

	imprimirCuentaDeLaExportacion(salida, cuenta, *rutaSalida)
	return 0
}

// fuenteDeLasRespuestas son las TRES formas de decir de donde salen, tal cual
// llegan de la linea de ordenes.
type fuenteDeLasRespuestas struct {
	direccion string // --url
	consulta  string // --respuestas
	cuenta    string // --cuenta
	datos     string // --datos, donde vive el fichero de respuestas guardadas
}

// consultaDeLaEntrevista admite las tres formas y NO LAS MEZCLA.
//
// Con --cuenta se sacan las que ya estan guardadas, que es la forma que no
// obliga a copiar nada. Con --url se pega la direccion entera de la barra del
// navegador. Con --respuestas se compone a mano. Dos a la vez se rechazan:
// dirian dos cosas distintas y no habria forma de saber cual manda, y elegir
// una en silencio es exportar un alcance que el operador no ha pedido.
func consultaDeLaEntrevista(f fuenteDeLasRespuestas, errores io.Writer) (url.Values, int) {
	direccion, respuestas := strings.TrimSpace(f.direccion), strings.TrimSpace(f.consulta)
	cuenta := strings.TrimSpace(f.cuenta)
	dadas := 0
	for _, x := range []string{direccion, respuestas, cuenta} {
		if x != "" {
			dadas++
		}
	}
	if dadas > 1 {
		fmt.Fprintln(errores, "has dado mas de una fuente de respuestas a la vez, y dicen")
		fmt.Fprintln(errores, "cosas distintas.")
		fmt.Fprintln(errores, "  Elige una: --cuenta (las que tienes guardadas), --url (la")
		fmt.Fprintln(errores, "  direccion entera del navegador) o --respuestas (solo la consulta).")
		return nil, 2
	}
	if cuenta != "" {
		return consultaDeLaCuenta(cuenta, f.datos, errores)
	}
	switch {
	case direccion != "":
		u, err := url.Parse(direccion)
		if err != nil {
			// LA DIRECCION NO SE ENVUELVE EN EL ERROR. La regla de la casa dice
			// que una URL de configuracion no viaja entera a un error, y aunque
			// esta la acaba de teclear el operador, el error puede acabar en el
			// bloque copiable de `plazum doctor --issue`, que existe para pegarlo
			// en un issue publico. Se dice QUE falla, no CON QUE.
			fmt.Fprintln(errores, "--url no es una direccion que se pueda leer.")
			fmt.Fprintln(errores, "  Pega la direccion entera de la barra del navegador, con su")
			fmt.Fprintln(errores, "  parte de consulta (la que empieza por ?si=...).")
			return nil, 2
		}
		v, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			fmt.Fprintln(errores, "la parte de consulta de --url no se entiende.")
			return nil, 2
		}
		if len(v) == 0 {
			fmt.Fprintln(errores, "--url no lleva ninguna respuesta dentro.")
			fmt.Fprintln(errores, "  Las respuestas de la entrevista viajan EN LA DIRECCION (plazum")
			fmt.Fprintln(errores, "  no las guarda), asi que la direccion que se pega tiene que ser la")
			fmt.Fprintln(errores, "  de la entrevista YA RESPONDIDA, con su ?si=... detras.")
			return nil, 2
		}
		return v, 0
	case respuestas != "":
		v, err := url.ParseQuery(strings.TrimPrefix(respuestas, "?"))
		if err != nil {
			fmt.Fprintln(errores, "--respuestas no se entiende: se escribe como si=X&si=Y&no=Z")
			return nil, 2
		}
		return v, 0
	default:
		fmt.Fprint(errores, ayudaAlcance)
		return nil, 2
	}
}

// exportarAlcance es EL EXPORTADOR. Vive aqui, en el producto, y no en un test.
//
// # Por que empareja como empareja (invariante 7)
//
// Casa por el ID DE PREGUNTA, que es lo que viaja en la direccion, y ese id lo
// declara el propio paquete: `pantalla.Derivar` lo deriva del corpus y trae con
// el la entidad, el atributo y el paquete de los que salio. No hay indice ni
// posicion por medio, y un id que el corpus instalado no declara NO se traduce a
// nada: se cuenta como desconocido y se dice.
//
// # Los cinco destinos de una respuesta, y no hay un sexto
//
//	desconocida    su id no lo declara el corpus instalado
//	negativa       es un «no», y un «no» no afirma nada en este motor
//	sin puente     su ATRIBUTO no declara la traduccion todavia
//	con valor      su atributo pide un VALOR y la entrevista solo sabe si/no
//	traducida      produce hechos (o cero, si el paquete la declara callejon)
//
// La cuenta los recorre todos y su suma tiene que dar las leidas. Un destino que
// faltara seria una respuesta que desaparece sin dejar rastro, que es la forma
// mas cara de fallar aqui: obligaciones que no aparecen y nadie sabe por que.
func exportarAlcance(ps []*corpus.Paquete, consulta url.Values, sujeto, organizacion string) (
	alcanceExportado, CuentaDeLaExportacion, error) {

	// LAS PREGUNTAS SALEN DEL CORPUS INSTALADO, no de la peticion: es lo que
	// impide que una direccion adversaria meta atributos que nadie declara.
	porID := map[string]pantalla.Pregunta{}
	for _, p := range pantalla.Derivar(ps) {
		for _, q := range p.Preguntas {
			porID[q.ID] = q
		}
	}
	// LA FORMA DEL PUENTE, POR ATRIBUTO Y NO POR PAQUETE.
	//
	// `p.DeclaraPuente()` dice si el paquete lo declara EN ALGUN atributo, y
	// preguntar eso dejaba pasar al puente respuestas de otros atributos del
	// mismo paquete, que lo hacian fallar entero. Aqui se pregunta por el
	// atributo concreto, que es lo que el puente va a mirar despues.
	type atributoDelCorpus struct{ urn, entidad, atributo string }
	formaDelPuente := map[atributoDelCorpus]string{}
	porURN := map[string]*corpus.Paquete{}
	for _, p := range ps {
		porURN[p.URN] = p
		for _, e := range p.Entidades {
			for _, a := range e.Atributos {
				if a.Hecho != nil {
					formaDelPuente[atributoDelCorpus{p.URN, e.Nombre, a.Nombre}] = a.Hecho.Forma
				}
			}
		}
	}

	cuenta := CuentaDeLaExportacion{SinPuente: map[string]int{}}
	// porPaquete agrupa las respuestas traducibles. La traduccion la hace CADA
	// PAQUETE con su propia declaracion: es conocimiento normativo y no puede
	// vivir en una tabla comun.
	porPaquete := map[string][]corpus.RespuestaDeEntrevista{}

	// El orden importa para que dos ejecuciones con la misma direccion den el
	// mismo fichero: un mapa de url.Values se recorre en orden aleatorio.
	// contestadas es el REGISTRO de lo que se respondio, entero, y va al fichero
	// aparte de los hechos. Ver el godoc de alcanceExportado.Respuestas: sin
	// esto la vuelta no existe, porque los «no» y las respuestas sin puente no
	// dejan ningun hecho detras y desaparecerian del fichero sin decirlo.
	var contestadas []respuestaDeJSON
	clasificar := func(ids []string, esSi bool) {
		for _, id := range ids {
			cuenta.Leidas++
			q, conocida := porID[id]
			if !conocida {
				cuenta.Desconocidas = append(cuenta.Desconocidas, id)
				continue
			}
			valor := "no"
			if esSi {
				valor = "si"
			}
			contestadas = append(contestadas, respuestaDeJSON{
				// El campo se compone de la entidad y el atributo QUE DECLARA EL
				// PAQUETE, igual que lo escribe el alcance del demo. Es
				// informativo: quien empareja al volver es `pregunta`, porque
				// dos preguntas pueden pedir el mismo campo sobre instancias
				// distintas y el campo no las distingue.
				Campo: q.Entidad + "." + q.Atributo, Valor: valor, Pregunta: id,
			})
			if !esSi {
				// UN «NO» NO AFIRMA NADA, y no es una perdida: en este motor la
				// ausencia de un hecho no es su negacion, y afirmar
				// `no_predicado(...)` meteria en el expediente algo que el
				// operador no ha dicho.
				cuenta.Negativas++
				continue
			}
			forma, declarado := formaDelPuente[atributoDelCorpus{q.Paquete, q.Entidad, q.Atributo}]
			if !declarado {
				cuenta.SinPuente[q.Paquete]++
				continue
			}
			if forma == corpus.PuenteConValor {
				// UN ATRIBUTO CON VALOR CONTESTADO CON UN SI. El puente afirma
				// `predicado(instancia, valor)` y aqui no hay valor que poner,
				// porque la entrevista web solo sabe preguntar si o no.
				// Mandarla al puente lo hace fallar ENTERO y no exportar nada.
				cuenta.ConValor = append(cuenta.ConValor, id)
				continue
			}
			cuenta.Traducidas++
			porPaquete[q.Paquete] = append(porPaquete[q.Paquete], corpus.RespuestaDeEntrevista{
				Entidad: q.Entidad, Instancia: sujeto, Atributo: q.Atributo, Si: true,
			})
		}
	}
	clasificar(consulta[pantallas.ParamSi], true)
	clasificar(consulta[pantallas.ParamNo], false)
	sort.Strings(cuenta.Desconocidas)
	sort.Strings(cuenta.ConValor)

	var hechos []aplicabilidad.Hecho
	for _, urn := range ordenarClaves(porPaquete) {
		hs, err := corpus.HechosDeLaEntrevista(porURN[urn], porPaquete[urn])
		if err != nil {
			// NO SE DEGRADA A «ningun hecho». El puente falla cuando una
			// respuesta no se puede traducir, y tragarselo daria un alcance
			// corto sin que nadie lo supiera.
			return alcanceExportado{}, cuenta, fmt.Errorf("traduciendo las respuestas de %s: %w.\n"+
				"  No se exporta a medias: un alcance al que le faltan hechos deriva menos\n"+
				"  obligaciones y no lo dice", urn, err)
		}
		hechos = append(hechos, hs...)
	}
	cuenta.Hechos = len(hechos)

	// EL ORDEN DEL BLOQUE DE RESPUESTAS ES ESTABLE (por id de pregunta) para que
	// dos ejecuciones con la misma entrada den el mismo fichero byte a byte. Sin
	// esto, el orden lo decidiria el de `url.Values`, que es el de llegada, y
	// dos exportaciones de lo mismo darian ficheros distintos.
	sort.Slice(contestadas, func(i, j int) bool {
		return contestadas[i].Pregunta < contestadas[j].Pregunta
	})
	doc := alcanceExportado{
		Organizacion: organizacion, Sujeto: sujeto,
		Descripcion: "alcance derivado de la entrevista por el puente que declara cada paquete",
		Respuestas:  contestadas,
		Hechos:      []hechoDeJSON{},
	}
	if doc.Respuestas == nil {
		doc.Respuestas = []respuestaDeJSON{}
	}
	for _, h := range hechos {
		doc.Hechos = append(doc.Hechos, hechoDeJSON{Pred: h.Pred, Args: h.Args})
	}
	return doc, cuenta, nil
}

func ordenarClaves(m map[string][]corpus.RespuestaDeEntrevista) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// imprimirCuentaDeLaExportacion es la mitad que hace honesta a la otra.
//
// Sin esto, la orden escribiria un fichero y diria «hecho», y quien lo use no
// sabria si su alcance lleva sus respuestas o la mitad de ellas.
func imprimirCuentaDeLaExportacion(w io.Writer, c CuentaDeLaExportacion, ruta string) {
	fmt.Fprintf(w, "Alcance escrito en %s: %d hechos.\n\n", ruta, c.Hechos)
	fmt.Fprintln(w, "LA CUENTA, ENTERA")
	fmt.Fprintf(w, "    %3d respuestas leidas de la direccion\n", c.Leidas)
	fmt.Fprintf(w, "    %3d traducidas a hechos por el puente de su paquete\n", c.Traducidas)
	if c.Negativas > 0 {
		fmt.Fprintf(w, "    %3d respondidas que NO, que no afirman nada: en este motor la "+
			"ausencia\n", c.Negativas)
		fmt.Fprintln(w, "        de un hecho no es su negacion, y afirmarla meteria en tu")
		fmt.Fprintln(w, "        expediente algo que no has dicho.")
	}
	if len(c.SinPuente) > 0 {
		total := 0
		for _, v := range c.SinPuente {
			total += v
		}
		fmt.Fprintf(w, "    %3d de paquetes que TODAVIA NO declaran el puente, asi que no se\n", total)
		fmt.Fprintln(w, "        pueden traducir sin inventarse el predicado:")
		urns := make([]string, 0, len(c.SinPuente))
		for k := range c.SinPuente {
			urns = append(urns, k)
		}
		sort.Strings(urns)
		for _, u := range urns {
			fmt.Fprintf(w, "          %-42s %d\n", u, c.SinPuente[u])
		}
	}
	if len(c.ConValor) > 0 {
		fmt.Fprintf(w, "    %3d de un atributo CON VALOR (una categoria, un nivel) contestado\n",
			len(c.ConValor))
		fmt.Fprintln(w, "        con un si. Su norma no pregunta «si o no», pregunta CUAL, y la")
		fmt.Fprintln(w, "        entrevista todavia no sabe pedir ese dato, asi que no se puede")
		fmt.Fprintln(w, "        traducir sin inventarse el valor. Son:")
		for _, id := range c.ConValor {
			fmt.Fprintf(w, "          %s\n", id)
		}
	}
	if len(c.Desconocidas) > 0 {
		fmt.Fprintf(w, "    %3d con un id de pregunta que este corpus no declara (direccion vieja,\n",
			len(c.Desconocidas))
		fmt.Fprintln(w, "        u otro corpus). No se descartan en silencio; son:")
		for _, id := range c.Desconocidas {
			fmt.Fprintf(w, "          %s\n", id)
		}
	}
	// LA LEY DE CONSERVACION, IMPRESA. Si no cuadra hay respuestas que no estan
	// en ningun cubo, o sea un alcance corto sin motivo visible.
	if c.Suma() != c.Leidas {
		fmt.Fprintf(w, "\n    AVISO: los cubos suman %d y se leyeron %d respuestas. Faltan %d por\n",
			c.Suma(), c.Leidas, c.Leidas-c.Suma())
		fmt.Fprintln(w, "    explicar, y eso es un fallo del producto, no tuyo.")
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "LO QUE ESTE EXPORTADOR TODAVIA NO HACE")
	fmt.Fprintln(w, "    Los atributos CON VALOR (la categoria de un sistema, el nivel de una")
	fmt.Fprintln(w, "    informacion) no salen: la entrevista web solo sabe preguntar si o no.")
	fmt.Fprintln(w, "    Y todas tus respuestas caen sobre el mismo sujeto, porque la entrevista")
	fmt.Fprintln(w, "    tampoco pregunta todavia por CADA informacion y CADA servicio.")
	fmt.Fprintln(w, "    Si te falta algo en el calendario, eso es lo primero que mirar.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Ahora:")
	fmt.Fprintf(w, "    plazum calendario --alcance %s\n", ruta)
	fmt.Fprintf(w, "    plazum serve --alcance %s\n", ruta)
}
