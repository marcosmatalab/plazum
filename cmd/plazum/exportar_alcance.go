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

  plazum alcance --url "http://localhost:8443/alcance?si=X&si=Y&no=Z" \
                 --sujeto sis --organizacion "Acme SL" [--salida alcance.json]
  plazum alcance --respuestas "si=X&si=Y" --sujeto sis --organizacion "Acme SL"

  --url          pega la direccion ENTERA de la barra del navegador cuando
                 tengas la entrevista respondida. Es la forma que no obliga a
                 entender el formato.
  --respuestas   solo la parte de la consulta, si prefieres componerla tu.
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
	Organizacion string        `json:"organizacion"`
	Sujeto       string        `json:"sujeto"`
	Descripcion  string        `json:"descripcion"`
	Hechos       []hechoDeJSON `json:"hechos"`
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
	// SinPuente son las respuestas cuyo paquete no declara el puente, por
	// paquete. Es el cardinal de lo que este exportador todavia no puede.
	SinPuente map[string]int
	// Traducidas son las que han producido hechos.
	Traducidas int
	// Hechos son los hechos emitidos. Puede ser distinto de Traducidas: un
	// atributo declarado `no_llega_al_motor` se recoge y no afirma nada.
	Hechos int
}

// Suma es lo que tiene que cuadrar con Leidas.
func (c CuentaDeLaExportacion) Suma() int {
	n := len(c.Desconocidas) + c.Negativas + c.Traducidas
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

	consulta, codigo := consultaDeLaEntrevista(*direccion, *respuestas, errores)
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

// consultaDeLaEntrevista admite las dos formas y NO las mezcla.
//
// Con --url se pega la direccion entera de la barra del navegador, que es la
// forma que no obliga a entender el formato. Con --respuestas se compone a mano.
// Las dos a la vez se rechazan: dirian dos cosas distintas y no habria forma de
// saber cual manda.
func consultaDeLaEntrevista(direccion, respuestas string, errores io.Writer) (url.Values, int) {
	direccion, respuestas = strings.TrimSpace(direccion), strings.TrimSpace(respuestas)
	switch {
	case direccion != "" && respuestas != "":
		fmt.Fprintln(errores, "has dado --url Y --respuestas a la vez, y dicen dos cosas.")
		fmt.Fprintln(errores, "  Elige una: la direccion entera del navegador, o solo la consulta.")
		return nil, 2
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
// # Los cuatro destinos de una respuesta, y no hay un quinto
//
//	desconocida    su id no lo declara el corpus instalado
//	negativa       es un «no», y un «no» no afirma nada en este motor
//	sin puente     su paquete no declara la traduccion todavia
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
	declaraPuente := map[string]bool{}
	porURN := map[string]*corpus.Paquete{}
	for _, p := range ps {
		porURN[p.URN] = p
		declaraPuente[p.URN] = p.DeclaraPuente()
	}

	cuenta := CuentaDeLaExportacion{SinPuente: map[string]int{}}
	// porPaquete agrupa las respuestas traducibles. La traduccion la hace CADA
	// PAQUETE con su propia declaracion: es conocimiento normativo y no puede
	// vivir en una tabla comun.
	porPaquete := map[string][]corpus.RespuestaDeEntrevista{}

	// El orden importa para que dos ejecuciones con la misma direccion den el
	// mismo fichero: un mapa de url.Values se recorre en orden aleatorio.
	clasificar := func(ids []string, esSi bool) {
		for _, id := range ids {
			cuenta.Leidas++
			q, conocida := porID[id]
			if !conocida {
				cuenta.Desconocidas = append(cuenta.Desconocidas, id)
				continue
			}
			if !esSi {
				// UN «NO» NO AFIRMA NADA, y no es una perdida: en este motor la
				// ausencia de un hecho no es su negacion, y afirmar
				// `no_predicado(...)` meteria en el expediente algo que el
				// operador no ha dicho.
				cuenta.Negativas++
				continue
			}
			if !declaraPuente[q.Paquete] {
				cuenta.SinPuente[q.Paquete]++
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

	doc := alcanceExportado{
		Organizacion: organizacion, Sujeto: sujeto,
		Descripcion: "alcance derivado de la entrevista por el puente que declara cada paquete",
		Hechos:      []hechoDeJSON{},
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
