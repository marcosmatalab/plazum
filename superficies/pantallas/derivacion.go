package pantallas

import (
	"net/url"
	"sort"

	"github.com/marcosmatalab/plazum/nucleo/pantalla"
)

// La derivacion a un clic.
//
// Que es y que NO es. Aqui se lee lo que el modelo de nucleo/pantalla ya dice
// (que preguntas hay que responder para saber si una obligacion alcanza al
// sujeto) y se cruza con lo que el operador acaba de responder. Con eso, cada
// clic en la entrevista mueve obligaciones entre tres cestas y ensena de que
// respuesta y de que articulo sale cada movimiento.
//
// Esto NO es el motor de aplicabilidad. nucleo/aplicabilidad decide de verdad,
// con Datalog, sobre HECHOS de las entidades del sujeto, y esos hechos salen
// del expediente, que en esta etapa todavia no existe. Lo de aqui es un avance
// de alcance a partir de las preguntas declaradas por el paquete, y la interfaz
// lo dice con esas palabras. Confundir las dos cosas en un producto que vigila
// plazos legales seria el peor fallo posible de todos los que caben aqui, asi
// que se separa por escrito, en el nombre de los tipos y en la pantalla.

// Estado es en que cesta cae una obligacion con las respuestas dadas.
type Estado uint8

const (
	// Pendiente: falta responder alguna de sus preguntas. Es el estado por
	// defecto, y es el conservador: mientras no se sepa, no se afirma nada.
	Pendiente Estado = iota
	// Aplica: todas sus preguntas estan respondidas que si, o no tiene
	// ninguna y por tanto alcanza a todo el mundo.
	Aplica
	// NoAplica: al menos una de sus preguntas esta respondida que no.
	NoAplica
)

// Clave es la clave de catalogo del rotulo del estado. El rotulo lo pone el
// catalogo; aqui solo se elige cual.
func (e Estado) Clave() string {
	switch e {
	case Aplica:
		return "estado.aplica"
	case NoAplica:
		return "estado.no_aplica"
	}
	return "estado.pendiente"
}

// String da el valor que viaja en la direccion de la pagina como filtro.
// Es identificador, no texto: nunca se ensena sin pasar por el catalogo.
func (e Estado) String() string {
	switch e {
	case Aplica:
		return "aplica"
	case NoAplica:
		return "no_aplica"
	}
	return "pendiente"
}

// estadoDeFiltro lee el valor del filtro que llega en la peticion. Cualquier
// cosa que no sea uno de los tres se trata como "sin filtro", y NO se devuelve
// al navegador: un valor desconocido reflejado en la pagina es la mitad de un
// XSS reflejado, y no hay razon para tener esa mitad.
func estadoDeFiltro(v string) (Estado, bool) {
	switch v {
	case "aplica":
		return Aplica, true
	case "no_aplica":
		return NoAplica, true
	case "pendiente":
		return Pendiente, true
	}
	return Pendiente, false
}

// Respuesta es lo que el operador ha contestado a una pregunta.
type Respuesta uint8

const (
	SinResponder Respuesta = iota
	Si
	No
	// Contradictoria: la misma pregunta llega respondida que si Y que no.
	// No se resuelve a favor de ninguna de las dos: se trata como sin
	// responder y se avisa en la pantalla. Elegir una en silencio seria
	// afirmar alcance sobre una entrada que no dice nada.
	Contradictoria
)

// Respuestas es la entrevista respondida, tal como llega en la direccion.
//
// Solo contiene IDs de pregunta QUE EXISTEN en el corpus instalado. Lo que
// llega y no existe se descarta al construirla y no vuelve a aparecer en
// ningun sitio: ni se cuenta, ni se ensena, ni se copia a los enlaces.
type Respuestas struct {
	porID map[string]Respuesta
	orden []string // los IDs conocidos, en el orden de la entrevista
	// porValor y estadoDe son LA MITAD CON VALOR. Ver valores.go.
	//
	// porValor solo lleva los valores que se entienden: uno que no se entiende
	// NO se guarda, para que no pueda acabar reflejado en un enlace ni en la
	// pagina por descuido. Lo que queda de el es su estado, que es lo que la
	// pantalla necesita para decir que llego algo y no se pudo usar.
	porValor map[string]string
	estadoDe map[string]EstadoDelValor
	// voc es lo que el corpus declara de cada pregunta: que tipo de dato pide y
	// que valores admite. El valor cero no conoce ninguna, y entonces todo se
	// pregunta con si/no, que es el restrictivo. Ver Vocabulario.
	voc Vocabulario
}

// De construye las respuestas a partir de los valores de la consulta y del
// conjunto de preguntas que el corpus declara.
//
// conocidas viene del modelo derivado, no de la peticion: es lo que hace que
// una peticion adversaria no pueda meter nada en el estado de la pantalla. Y
// voc tambien: los valores que se admiten en un enumerado los declara el
// paquete, no los propone quien pide la pagina.
//
// # Una pregunta CON VALOR que llega con un si o un no de la forma antigua
//
// Se conserva, y no es un descuido. Esta superficie lleva desde el principio
// pintando toda pregunta como un si/no, asi que hay enlaces compartidos y
// cuentas guardadas llenos de `si=<pregunta con valor>`. Rechazarlos convertiria
// cada uno de esos enlaces en una pagina de error el dia del despliegue.
//
// Lo que ese si/no hace y lo que NO hace estan separados: sigue moviendo la
// derivacion provisional de la pantalla, como hasta hoy, y NUNCA produce un
// hecho, tambien como hasta hoy (el exportador lo apartaba a su cubo). Lo que
// afirma es el valor, y solo el valor.
//
// Y si llegan LAS DOS COSAS sobre la misma pregunta, es contradictorio y no
// afirma nada: son dos respuestas distintas a la misma pregunta y elegir una en
// silencio seria afirmar un alcance que nadie afirmo.
func De(valores url.Values, conocidas []pantalla.Pregunta, voc Vocabulario) Respuestas {
	r := Respuestas{
		porID:    make(map[string]Respuesta, len(conocidas)),
		porValor: map[string]string{},
		estadoDe: map[string]EstadoDelValor{},
		voc:      voc,
	}
	valida := make(map[string]bool, len(conocidas))
	for _, q := range conocidas {
		valida[q.ID] = true
		r.orden = append(r.orden, q.ID)
	}
	marcar := func(id string, quiere Respuesta) {
		if !valida[id] {
			return
		}
		switch anterior := r.porID[id]; {
		case anterior == SinResponder:
			r.porID[id] = quiere
		case anterior != quiere:
			r.porID[id] = Contradictoria
		}
	}
	for _, id := range valores[ParamSi] {
		marcar(id, Si)
	}
	for _, id := range valores[ParamNo] {
		marcar(id, No)
	}
	// LOS VALORES SE LEEN RECORRIENDO LAS PREGUNTAS CONOCIDAS, no las claves de
	// la consulta. Es la misma guarda que arriba y por la misma razon: asi una
	// peticion adversaria no puede meter en el estado de la pantalla una clave
	// `v.<lo que sea>` que el corpus no declara.
	for _, q := range conocidas {
		valor, estado := leerValorOpcional(valores, q.ID, voc.campo(q.ID))
		if estado == ValorAusente {
			continue
		}
		if estado == ValorPuesto && r.porID[q.ID] != SinResponder {
			// Un valor Y un si/no sobre la misma pregunta.
			estado = ValorContradictorio
			valor = ""
		}
		r.estadoDe[q.ID] = estado
		if estado == ValorPuesto {
			r.porValor[q.ID] = valor
		}
	}
	return r
}

// Valor devuelve lo contestado a una pregunta con valor. Cadena vacia si no hay
// respuesta utilizable, que incluye el caso de que llegara algo y no se
// entendiera: ese valor no se conserva a proposito.
func (r Respuestas) Valor(id string) string { return r.porValor[id] }

// EstadoDelValorDe dice en cual de los cinco casos esta la respuesta con valor.
// El valor cero (ValorAusente) es el de una pregunta que nadie ha tocado.
func (r Respuestas) EstadoDelValorDe(id string) EstadoDelValor { return r.estadoDe[id] }

// PideValor dice si esta pregunta se contesta con un valor en vez de con si/no.
func (r Respuestas) PideValor(id string) bool { return r.voc.Tipo(id).PideValor() }

// Vocabulario devuelve lo que el corpus declara de las preguntas.
func (r Respuestas) Vocabulario() Vocabulario { return r.voc }

// ValoresQueNoSeEntienden cuenta las preguntas a las que llego un dato que hay y
// no se entiende (la tercera forma de la nada, que no es la nada).
//
// SE CUENTA Y SE ENSENA. Sin este numero, un valor que no se entiende se
// comportaria exactamente igual que no haber contestado, y esa es la confusion
// que el invariante 8 prohibe: quien mando algo veria su pregunta en blanco y
// creeria que no llego a mandarlo.
func (r Respuestas) ValoresQueNoSeEntienden() int {
	n := 0
	for _, e := range r.estadoDe {
		if e.EsError() {
			n++
		}
	}
	return n
}

// ValoresPuestos cuenta las preguntas contestadas con un valor utilizable.
//
// Es el cardinal de lo que el almacen de alcances NO sabe guardar todavia: su
// frontera habla de Si y de No, y un valor no cabe ahi. Se cuenta para poder
// decirlo en pantalla, no para taparlo.
func (r Respuestas) ValoresPuestos() int {
	n := 0
	for _, e := range r.estadoDe {
		if e.Afirma() {
			n++
		}
	}
	return n
}

// ConValor devuelve las respuestas con una pregunta puesta a un valor. No muta.
//
// El valor se comprueba contra lo que declara el corpus igual que si hubiera
// llegado por la direccion: esta funcion la usan los enlaces de la pantalla, y
// un enlace que produjera un valor que la lectura de entrada rechaza seria un
// boton que lleva a un error.
func (r Respuestas) ConValor(id, valor string) Respuestas {
	otra := r.copiar()
	v := url.Values{}
	v.Set(ClaveValor(id), valor)
	limpio, estado := leerValorOpcional(v, id, r.voc.campo(id))
	delete(otra.porValor, id)
	delete(otra.estadoDe, id)
	if estado == ValorPuesto {
		otra.porValor[id] = limpio
		otra.estadoDe[id] = estado
		// CONTESTAR CON UN VALOR BORRA EL SI/NO DE LA FORMA ANTIGUA sobre esa
		// misma pregunta. Si no, el primer clic en una opcion producirian las
		// dos respuestas a la vez, o sea una contradiccion fabricada por la
		// propia pantalla.
		delete(otra.porID, id)
	}
	return otra
}

// SinValor deja una pregunta con valor sin contestar. Es el «deshacer», y la
// forma que toma es la ausencia del parametro: la nada de verdad.
func (r Respuestas) SinValor(id string) Respuestas {
	otra := r.copiar()
	delete(otra.porValor, id)
	delete(otra.estadoDe, id)
	return otra
}

func (r Respuestas) copiar() Respuestas {
	otra := Respuestas{
		porID:    make(map[string]Respuesta, len(r.porID)+1),
		porValor: make(map[string]string, len(r.porValor)+1),
		estadoDe: make(map[string]EstadoDelValor, len(r.estadoDe)+1),
		orden:    r.orden,
		voc:      r.voc,
	}
	for k, v := range r.porID {
		otra.porID[k] = v
	}
	for k, v := range r.porValor {
		otra.porValor[k] = v
	}
	for k, v := range r.estadoDe {
		otra.estadoDe[k] = v
	}
	return otra
}

// Dice devuelve la respuesta a una pregunta. SOLO LA MITAD DE SI/NO: una
// pregunta con valor contestada devuelve SinResponder por aqui, y eso no es un
// descuido sino el significado del tipo. Para «esta pregunta esta contestada,
// de la forma que sea» esta SinContestar.
func (r Respuestas) Dice(id string) Respuesta { return r.porID[id] }

// SinContestar dice si a esta pregunta le falta respuesta, MIRANDO LAS DOS
// MITADES.
//
// # El fallo que existe para no repetir, y salio en un test de verdad
//
// La pantalla elegia la pregunta sugerida preguntando `Dice(id) ==
// SinResponder`, que solo mira el si/no. En cuanto una pregunta con valor se
// pudo contestar, contestarla la dejaba marcada como «empieza por esta» para
// siempre: el valor quedaba puesto y visible, la obligacion se movia de cesta, y
// la flecha seguia apuntando a la pregunta que acababas de responder. La
// entrevista no avanzaba nunca.
//
// Salio de `TestLaPrimeraPreguntaEsLaQueMasDesbloqueaYVieneSugerida`, que
// recorre la pantalla como un operador, y NO de leer el diff.
//
// Una respuesta contradictoria (de cualquiera de las dos mitades) cuenta como
// sin contestar: es lo unico honesto que se puede hacer con una entrada que se
// contradice, y es lo que esta superficie ya hacia con los si/no.
func (r Respuestas) SinContestar(id string) bool {
	if e := r.estadoDe[id]; e.Afirma() {
		return false
	}
	d := r.porID[id]
	return d == SinResponder || d == Contradictoria
}

// Respondidas cuenta las preguntas con respuesta utilizable. Una contradictoria
// no cuenta: no dice nada.
func (r Respuestas) Respondidas() int {
	n := 0
	for id, v := range r.porID {
		if (v == Si || v == No) && !r.estadoDe[id].Afirma() {
			// La condicion de la derecha evita contar dos veces la misma
			// pregunta. No puede darse hoy (un valor puesto borra el si/no y un
			// si/no con valor sale contradictorio), y se escribe igual: el dia
			// que alguien cambie una de esas dos reglas, la barra de progreso
			// pasaria de 68 preguntas a 71 sin que nadie tocara la barra.
			n++
		}
	}
	return n + r.ValoresPuestos()
}

// Contradictorias cuenta las preguntas respondidas que si y que no a la vez.
func (r Respuestas) Contradictorias() int {
	n := 0
	for _, v := range r.porID {
		if v == Contradictoria {
			n++
		}
	}
	return n
}

// Consulta reconstruye la parte de la direccion que lleva las respuestas.
//
// Se reconstruye desde el estado ya saneado y NUNCA se copia la consulta que
// llego: asi lo que un tercero meta en un enlace no sobrevive al primer clic.
// El orden es el de la entrevista, que es estable, para que la misma entrevista
// de siempre la misma direccion y las paginas se puedan comparar y cachear.
func (r Respuestas) Consulta() url.Values {
	v := url.Values{}
	for _, id := range r.orden {
		// EL VALOR VA PRIMERO Y SE ESCRIBE SOLO SI SE ENTIENDE.
		//
		// Uno que no se entiende NO se copia al enlace, y es deliberado, al
		// contrario que la contradiccion de si/no de abajo. Reflejar en la
		// pagina un valor que llego de la peticion y que no se ha podido
		// comprobar es la mitad de un XSS reflejado, y ademas lo arrastraria a
		// cada clic siguiente. Lo que sobrevive de el es el cardinal
		// (ValoresQueNoSeEntienden), que se ensena en la pagina a la que llego.
		if r.estadoDe[id].Afirma() {
			v.Set(ClaveValor(id), r.porValor[id])
		}
		switch r.porID[id] {
		case Si:
			v.Add(ParamSi, id)
		case No:
			v.Add(ParamNo, id)
		case Contradictoria:
			// Se conserva tal cual para que el aviso no desaparezca al
			// navegar: si se limpiara aqui, la pantalla siguiente diria
			// que esa pregunta esta sin responder y el operador nunca
			// sabria que mando dos respuestas.
			v.Add(ParamSi, id)
			v.Add(ParamNo, id)
		}
	}
	return v
}

// Con devuelve las respuestas con una pregunta puesta a un si o un no. No muta.
//
// SinResponder borra tambien el valor, porque el boton que la llama es el mismo
// «deshacer» para las dos mitades: quien lo pulsa esta diciendo que esa pregunta
// vuelve a estar sin contestar, y dejar el valor puesto la dejaria contestada.
func (r Respuestas) Con(id string, q Respuesta) Respuestas {
	otra := r.copiar()
	if q == SinResponder {
		delete(otra.porID, id)
		delete(otra.porValor, id)
		delete(otra.estadoDe, id)
	} else {
		otra.porID[id] = q
	}
	return otra
}

// Motivo es un trozo del "por que" de una obligacion: la respuesta concreta que
// la metio en su cesta, con la cita de donde sale la pregunta.
//
// Sin esto, una lista de obligaciones es una opinion. Con esto, el operador
// puede discutir cada linea con su auditor senalando el articulo.
type Motivo struct {
	// Clave de catalogo que explica el tipo de motivo.
	Clave string
	// PreguntaID, y el texto y la cita de la pregunta cuando existe. El
	// texto y la cita vienen del corpus y NO pasan por el catalogo.
	PreguntaID string
	Texto      string
	Cita       string
	Respuesta  Respuesta
	// Valor es lo contestado cuando la pregunta pide un valor. Viene del corpus
	// (es uno de los valores que declara el atributo, o un texto que ya paso la
	// lectura de entrada) y por eso se ensena tal cual, sin pasar por el
	// catalogo, igual que el texto y la cita de la pregunta.
	//
	// Vacio cuando la pregunta es de si/no o cuando llego algo que no se
	// entendio: lo que no se entiende no se conserva.
	Valor string
}

// Veredicto es una obligacion (o un entregable) con su estado y su por que.
type Veredicto struct {
	Fila    pantalla.Fila
	Estado  Estado
	Motivos []Motivo
}

// indicePreguntas mapea ID de pregunta a la pregunta derivada.
type indicePreguntas map[string]pantalla.Pregunta

func indexar(qs []pantalla.Pregunta) indicePreguntas {
	idx := make(indicePreguntas, len(qs))
	for _, q := range qs {
		idx[q.ID] = q
	}
	return idx
}

// evaluarControl decide el estado de una fila de Controles.
//
// En Controles, Fila.Requiere son IDs de PREGUNTA (lo pone asi el godoc de
// pantalla.Fila). Reglas, y son deliberadamente conservadoras:
//
//	sin preguntas          aplica, porque el paquete no la condiciona a nada
//	alguna respondida NO   no aplica, y se dice cual
//	todas respondidas SI   aplica, y se dicen todas
//	el resto               pendiente, y se dice que falta por responder
//
// Una pregunta contradictoria cuenta como sin responder, o sea que arrastra la
// obligacion a pendiente. Es lo unico honesto que se puede hacer con una
// entrada que se contradice.
func evaluarControl(f pantalla.Fila, r Respuestas, idx indicePreguntas) Veredicto {
	v := Veredicto{Fila: f}
	if len(f.Requiere) == 0 {
		v.Estado = Aplica
		v.Motivos = []Motivo{{Clave: "derivacion.sin_condiciones"}}
		return v
	}
	var negativas, positivas, faltan []Motivo
	for _, id := range f.Requiere {
		m := Motivo{PreguntaID: id, Respuesta: r.Dice(id)}
		if q, ok := idx[id]; ok {
			m.Texto, m.Cita = q.Texto, q.Cita
		} else {
			// El paquete condiciona la obligacion a una pregunta que el
			// paquete no declara. Es una errata del corpus que no da
			// error en ningun sitio y deja la obligacion colgada para
			// siempre. Se ensena en vez de tragarsela.
			m.Clave = "derivacion.pregunta_desconocida"
			faltan = append(faltan, m)
			continue
		}
		// LA MITAD CON VALOR VA PRIMERO Y MANDA. Una pregunta que pide un valor
		// se contesta con el valor; el si/no que pueda traer un enlace viejo
		// solo decide cuando no hay valor.
		//
		// UN VALOR PUESTO CUENTA COMO RESPUESTA POSITIVA, y hay que decir por
		// que, porque es una decision y no una obviedad. Esta derivacion NO es
		// el motor (ver el encabezado del fichero): lo que dice es «esta
		// obligacion depende de esta pregunta y esta pregunta ya esta
		// contestada». QUE valor haga que aplique de verdad lo decide el motor
		// de aplicabilidad con las reglas del paquete, y eso pasa por
		// corpus.HechosDeLaEntrevista, no por aqui. Entre dejarla PENDIENTE
		// (que esconde la obligacion) y darla por alcanzada, se elige lo que no
		// absuelve: una obligacion de mas la corrige quien lee, y una de menos
		// la descubre el inspector.
		if si := r.EstadoDelValorDe(id); r.PideValor(id) && si != ValorAusente {
			m.Valor = r.Valor(id)
			switch {
			case si.Afirma():
				m.Clave = "derivacion.respondiste_valor"
				positivas = append(positivas, m)
			case si.EsError():
				// UN DATO QUE HAY Y NO SE ENTIENDE NO ES UNA RESPUESTA. Arrastra
				// a pendiente, con su propio motivo: decir «sin responder»
				// haria creer que no llego nada.
				m.Clave = "derivacion.valor_no_se_entiende"
				faltan = append(faltan, m)
			default:
				m.Clave = "derivacion.sin_responder"
				faltan = append(faltan, m)
			}
			continue
		}
		switch m.Respuesta {
		case No:
			m.Clave = "derivacion.respondiste_no"
			negativas = append(negativas, m)
		case Si:
			m.Clave = "derivacion.respondiste_si"
			positivas = append(positivas, m)
		case Contradictoria:
			m.Clave = "derivacion.respuesta_contradictoria"
			faltan = append(faltan, m)
		default:
			m.Clave = "derivacion.sin_responder"
			faltan = append(faltan, m)
		}
	}
	switch {
	case len(negativas) > 0:
		v.Estado, v.Motivos = NoAplica, negativas
	case len(faltan) > 0:
		v.Estado, v.Motivos = Pendiente, faltan
	default:
		v.Estado, v.Motivos = Aplica, positivas
	}
	return v
}

// evaluarEntregable decide el estado de una fila de Certificados.
//
// OJO, y es la trampa de este fichero: en Certificados, Fila.Requiere NO son
// preguntas, son IDs de OBLIGACION (las que declaran ese entregable; lo hace
// pantalla.derivarCertificados). O sea que un entregable no se evalua contra la
// entrevista sino contra el veredicto de las obligaciones que lo piden:
//
//	alguna de sus obligaciones aplica       hay que entregarlo
//	todas sus obligaciones no aplican       no hay que entregarlo
//	ninguna obligacion lo pide              es huerfano: nadie sabe por que
//	el resto                                pendiente
//
// El huerfano se ensena a proposito. Una plantilla que ningun articulo pide es
// papeleo que el operador rellenaria sin motivo, y es ademas un fallo del
// paquete que asi se ve desde la interfaz.
func evaluarEntregable(f pantalla.Fila, porObligacion map[string]Veredicto) Veredicto {
	v := Veredicto{Fila: f}
	if len(f.Requiere) == 0 {
		v.Estado = Pendiente
		v.Motivos = []Motivo{{Clave: "derivacion.entregable_huerfano"}}
		return v
	}
	aplican, pendientes, noAplican := 0, 0, 0
	for _, id := range f.Requiere {
		// Una obligacion que no esta en el mapa da el Veredicto cero, o sea
		// Pendiente y sin columnas. Es el comportamiento correcto y por eso
		// no hay caso especial: el motivo sale igualmente con el ID de la
		// obligacion, y el operador ve de que cuelga el documento.
		w := porObligacion[id]
		m := Motivo{Clave: "derivacion.lo_pide", PreguntaID: id,
			Texto: w.Fila.Columnas["articulo"], Cita: w.Fila.Columnas["cita"]}
		switch w.Estado {
		case Aplica:
			aplican++
			m.Clave = "derivacion.lo_pide_y_aplica"
		case NoAplica:
			noAplican++
			m.Clave = "derivacion.lo_pide_y_no_aplica"
		default:
			pendientes++
		}
		v.Motivos = append(v.Motivos, m)
	}
	switch {
	case aplican > 0:
		v.Estado = Aplica
	case pendientes > 0:
		v.Estado = Pendiente
	case noAplican > 0:
		v.Estado = NoAplica
	}
	return v
}

// Resumen son los tres contadores que el operador mira primero.
type Resumen struct {
	Aplica    int
	Pendiente int
	NoAplica  int
	Total     int
}

func resumir(vs []Veredicto) Resumen {
	var r Resumen
	for _, v := range vs {
		r.Total++
		switch v.Estado {
		case Aplica:
			r.Aplica++
		case NoAplica:
			r.NoAplica++
		default:
			r.Pendiente++
		}
	}
	return r
}

// columnasEnOrden son las columnas conocidas de una tabla, en el orden en que
// se pintan. La lista es fija A PROPOSITO, y no se descubre de los datos:
//
//   - una columna nueva necesita su clave de catalogo, y una clave que nadie
//     ha escrito sale como clave cruda en pantalla;
//   - el orden de las columnas de una tabla legal no puede depender del
//     alfabeto ni del paquete que se cargue primero.
//
// Cuando nucleo/pantalla anada una columna, esta lista se queda corta y hay un
// test que se pone rojo diciendo cual falta (TestNoHayColumnaSinRotulo).
var columnasEnOrden = []string{
	"articulo", "titulo", "cita", "clase_e2e",
	"primitiva", "cadencia", "limite", "entregable",
}

// columnasPresentes devuelve las columnas conocidas que alguna fila trae, en el
// orden fijo, mas las desconocidas al final ordenadas, para que una columna sin
// rotulo se vea en pantalla en vez de desaparecer en silencio.
func columnasPresentes(filas []Veredicto) (conocidas, desconocidas []string) {
	hay := map[string]bool{}
	for _, f := range filas {
		for k := range f.Fila.Columnas {
			hay[k] = true
		}
	}
	sabidas := map[string]bool{}
	for _, c := range columnasEnOrden {
		sabidas[c] = true
		if hay[c] {
			conocidas = append(conocidas, c)
		}
	}
	for k := range hay {
		if !sabidas[k] {
			desconocidas = append(desconocidas, k)
		}
	}
	sort.Strings(desconocidas)
	return conocidas, desconocidas
}
