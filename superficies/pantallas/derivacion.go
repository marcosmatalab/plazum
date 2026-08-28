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
}

// De construye las respuestas a partir de los valores de la consulta y del
// conjunto de preguntas que el corpus declara.
//
// conocidas viene del modelo derivado, no de la peticion: es lo que hace que
// una peticion adversaria no pueda meter nada en el estado de la pantalla.
func De(valores url.Values, conocidas []pantalla.Pregunta) Respuestas {
	r := Respuestas{porID: make(map[string]Respuesta, len(conocidas))}
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
	return r
}

// Dice devuelve la respuesta a una pregunta.
func (r Respuestas) Dice(id string) Respuesta { return r.porID[id] }

// Respondidas cuenta las preguntas con respuesta utilizable. Una contradictoria
// no cuenta: no dice nada.
func (r Respuestas) Respondidas() int {
	n := 0
	for _, v := range r.porID {
		if v == Si || v == No {
			n++
		}
	}
	return n
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

// Con devuelve las respuestas con una pregunta puesta a un valor. No muta.
func (r Respuestas) Con(id string, q Respuesta) Respuestas {
	otra := Respuestas{porID: make(map[string]Respuesta, len(r.porID)+1), orden: r.orden}
	for k, v := range r.porID {
		otra.porID[k] = v
	}
	if q == SinResponder {
		delete(otra.porID, id)
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
