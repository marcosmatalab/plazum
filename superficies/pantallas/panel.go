package pantallas

// EL PANEL DE INICIO: Hoy, con cifras grandes y cada una abierta hasta su
// derivacion.
//
// # Que problema resuelve
//
// Hoy era una pantalla vacia con una explicacion. Correcta y muda: quien abre
// plazum a las nueve de la manana quiere cuatro numeros y saber de donde salen,
// y lo que encontraba era una frase que decia que algun dia habria numeros.
//
// # De donde salen los numeros, que es lo unico que importa aqui
//
// De nucleo/pantalla.Derivar12Meses, o sea del MISMO motor que escribe el
// fichero del calendario por terminal. No hay una segunda cuenta escrita en la
// superficie: si las dos discreparan, una de las dos estaria mintiendo y no
// habria forma de saber cual.
//
// La aplicabilidad que se le pasa sale de las RESPUESTAS DE LA ENTREVISTA, no de
// un perfil de arranque, y por eso ninguna fila sale marcada como supuesta: aqui
// no se supone nada sobre la organizacion. Y los HECHOS van vacios porque no hay
// expediente todavia, lo cual no se disimula: es justo lo que explica que casi
// todos los relojes esten esperando un dato del operador, y esa cuenta se pinta
// AL LADO del numero de vencimientos, no en un pie.
//
// # Las tres reglas del panel
//
//  1. NINGUNA CIFRA SIN ENLACE. Una cifra que no se puede abrir es una cifra que
//     hay que creerse, y este producto se vende exactamente al reves. Hay una
//     puerta que lo exige.
//  2. UN CERO NO ES UNA AUSENCIA. Cuando no hay corpus instalado, "0 te alcanzan"
//     seria una respuesta a una pregunta que nadie ha podido calcular: eso se
//     pinta como SIN DATO, con su motivo. Es el invariante 8 en una pantalla.
//  3. LO NO CONSTATADO NO ES CULPA. La cifra de vencimientos pasados lleva su
//     descargo PEGADO, dentro de la misma tarjeta, y lo lleva tanto cuando vale
//     cero como cuando vale catorce. Un numero grande en un panel se lee como una
//     acusacion, y acusar en falso es el unico error que un producto de
//     cumplimiento no puede cometer ni una vez.

import (
	"net/url"
	"sort"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/pantalla"
)

// copiaConsulta duplica unos valores de consulta para poder anadirles algo sin
// tocar los del que llama. Se copia y no se comparte porque el mismo mapa de
// respuestas alimenta varios enlaces de la misma pagina, y un Set sobre el
// original se los llevaria todos por delante.
func copiaConsulta(v url.Values) url.Values {
	c := url.Values{}
	for k, vs := range v {
		c[k] = append([]string(nil), vs...)
	}
	return c
}

// VentanaDeLaSemana es lo que cuenta como "vence esta semana".
//
// Siete dias naturales desde el instante de la peticion, no "hasta el domingo":
// un lunes por la manana las dos cosas coinciden, y un viernes por la tarde la
// segunda dice que no vence nada mientras el martes hay una notificacion. La
// unidad de un plazo legal es el dia, no la semana de calendario.
const VentanaDeLaSemana = 7 * 24 * time.Hour

// VistaCifra es un numero grande del panel, con su derivacion detras.
//
// Rotulo, SinDato, Matiz y Descargo son CLAVES de catalogo. N es un numero.
// Titulo y Cita de las filas de abajo vienen del corpus y viajan tal cual.
type VistaCifra struct {
	// Rotulo es la clave de catalogo de lo que cuenta esta cifra.
	Rotulo string
	// N es el numero, y solo significa algo si HayDato.
	N int
	// HayDato distingue "he contado y salen cero" de "no he podido contar".
	// Son dos frases distintas y solo la primera es un cero.
	HayDato bool
	// SinDato es la clave del motivo de que NO haya numero. Solo se pinta
	// cuando HayDato es falso.
	//
	// VA SEPARADA DE Matiz, y no es una separacion de estilo. Los dos campos
	// eran uno solo y el primer arranque contra el corpus real lo destapo: la
	// pantalla decia "16 marcos instalados" y debajo "no hay ningun paquete
	// normativo instalado". Una pantalla que se contradice a si misma en dos
	// lineas seguidas se deja de leer entera, y con razon. El motivo de una
	// ausencia y el matiz de un numero son dos frases distintas y no pueden
	// compartir campo, porque el campo compartido se pinta siempre.
	SinDato string
	// Matiz es la clave de lo que hay que saber para leer bien el numero.
	// Solo se pinta cuando HayDato es cierto.
	Matiz string
	// MatizN es el numero que Matiz lleva dentro, si lo lleva.
	MatizN int
	// Descargo es la clave del descargo que viaja CON el dato. Vacia en las
	// cifras que no acusan de nada.
	Descargo string
	// URL es la derivacion: donde se ve de que sale este numero. NUNCA vacia.
	URL string
	// Tono elige el color de la cifra, y es SIEMPRE redundante con el rotulo:
	// el color no puede ser el unico portador de significado.
	Tono string
}

// VistaVencimiento es una fila de "vence esta semana".
type VistaVencimiento struct {
	pantalla.Fecha
}

// VistaVencida es una fila de "sin constancia": un vencimiento que ya paso y
// que en las respuestas del operador no consta cerrado.
type VistaVencida struct {
	pantalla.Vencida
}

// VistaMarco es un marco instalado con lo que aporta al alcance.
type VistaMarco struct {
	// URN es el identificador del paquete, tal cual. No se traduce.
	URN string
	// Aplica, Pendiente y NoAplica son las obligaciones de ese marco en cada
	// cesta.
	Aplica    int
	Pendiente int
	NoAplica  int
	// URL lleva a las filas de ese marco. Como la tabla de Controles no filtra
	// por marco todavia, lleva a la tabla entera con el alcance puesto, que es
	// donde estan esas filas con su columna de paquete a la vista.
	URL string
}

// panel arma el panel de inicio entero.
func (s *Superficie) panel(m modelo, resp Respuestas, controles []Veredicto) Panel {
	ahora := s.ahora()
	res := resumir(controles)

	// La aplicabilidad que ve el calendario ES la de la entrevista. supuesta va
	// SIEMPRE a false y no es un descuido: aqui no hay perfil de arranque que
	// suponga nada sobre la organizacion, solo lo que el operador ha
	// respondido. El dia que esta pantalla acepte un perfil, ese false pasa a
	// ser el dato del perfil y las filas salen marcadas.
	estado := make(map[string]Estado, len(controles))
	for _, c := range controles {
		estado[c.Fila.ID] = c.Estado
	}
	aplica := func(id string) (bool, bool) { return estado[id] == Aplica, false }

	// Los HECHOS van vacios: no hay expediente. No se disimula, se cuenta.
	cal := pantalla.Derivar12Meses(m.paquetes, aplica, nil, ahora)

	p := Panel{HayCorpus: len(m.paquetes) > 0}
	hasta := ahora.Add(VentanaDeLaSemana)
	for _, mes := range cal.Meses {
		for _, f := range mes.Fechas {
			if !f.Vence.Before(ahora) && f.Vence.Before(hasta) {
				p.Semana = append(p.Semana, VistaVencimiento{Fecha: f})
			}
		}
	}
	for _, v := range cal.Vencidas {
		p.Vencidas = append(p.Vencidas, VistaVencida{Vencida: v})
	}
	p.EsperandoDato = len(cal.SinFecha)
	p.Marcos = marcosDeControles(controles, s.enlace(rutaDe(pantalla.Controles), resp.Consulta()))

	q := resp.Consulta()
	conFiltro := func(e Estado) string {
		c := copiaConsulta(q)
		c.Set(ParamFiltro, e.String())
		return s.enlace(rutaDe(pantalla.Controles), c)
	}

	// LAS CUATRO CIFRAS. El orden no es decorativo: primero lo que vence (lo
	// unico con hora), despues lo que no consta (lo unico que se puede leer
	// como acusacion, y por eso va con su descargo pegado), despues el alcance
	// y por ultimo los marcos, que es contexto.
	p.Cifras = []VistaCifra{
		{
			Rotulo: "pantalla.hoy.cifra.vence_semana", N: len(p.Semana), HayDato: p.HayCorpus,
			SinDato: "pantalla.hoy.cifra.sin_corpus", URL: s.base + rutaDe(pantalla.Hoy) + "#vence-esta-semana",
			Tono: "pendiente",
		},
		{
			Rotulo: "pantalla.hoy.cifra.sin_constancia", N: len(p.Vencidas), HayDato: p.HayCorpus,
			SinDato: "pantalla.hoy.cifra.sin_corpus",
			// EL DESCARGO VIAJA CON EL DATO, no en una nota al pie, y viaja
			// tambien cuando el dato es cero: lo que hay que desmentir no es el
			// numero, es la lectura de la palabra "sin constancia".
			Descargo: "pantalla.hoy.sin_constancia.descargo",
			URL:      s.base + rutaDe(pantalla.Hoy) + "#sin-constancia",
			Tono:     "neutro",
		},
		{
			Rotulo: "pantalla.hoy.cifra.te_alcanzan", N: res.Aplica, HayDato: p.HayCorpus,
			SinDato: "pantalla.hoy.cifra.sin_corpus", URL: conFiltro(Aplica), Tono: "aplica",
		},
		{
			Rotulo: "pantalla.hoy.cifra.marcos", N: len(p.Marcos), HayDato: p.HayCorpus,
			SinDato: "pantalla.hoy.cifra.sin_corpus",
			URL:     s.base + rutaDe(pantalla.Hoy) + "#marcos", Tono: "neutro",
		},
	}
	// EL MATIZ DE "VENCE ESTA SEMANA", y sin el ese numero miente por omision.
	// Sin expediente no hay ningun hecho declarado, asi que los relojes que
	// arrancan de un hecho del operador NO producen fecha: son los que salen en
	// EsperandoDato. Un "0 vence esta semana" con siete relojes esperando un
	// dato tuyo se lee como "no tienes nada que hacer", que es lo contrario de
	// lo que pasa.
	if p.EsperandoDato > 0 {
		p.Cifras[0].Matiz = "pantalla.hoy.vence_semana.esperando"
		p.Cifras[0].MatizN = p.EsperandoDato
	}
	// Y si la entrevista esta sin responder, el alcance solo cuenta las
	// obligaciones que alcanzan a todo el mundo. Se dice donde se mira.
	if p.HayCorpus && len(q) == 0 {
		p.Cifras[2].Matiz = "pantalla.hoy.cifra.sin_responder"
	}
	return p
}

// Panel es lo que la plantilla de Hoy pinta encima de la vigilancia.
type Panel struct {
	HayCorpus     bool
	Cifras        []VistaCifra
	Semana        []VistaVencimiento
	Vencidas      []VistaVencida
	EsperandoDato int
	Marcos        []VistaMarco
}

// marcosDeControles reparte las obligaciones por marco, ordenadas por cuantas
// alcanzan al sujeto y, a igualdad, por identificador, que es lo unico estable.
func marcosDeControles(controles []Veredicto, url string) []VistaMarco {
	porURN := map[string]*VistaMarco{}
	var orden []string
	for _, c := range controles {
		urn := c.Fila.Paquete
		v, ok := porURN[urn]
		if !ok {
			v = &VistaMarco{URN: urn, URL: url}
			porURN[urn] = v
			orden = append(orden, urn)
		}
		switch c.Estado {
		case Aplica:
			v.Aplica++
		case Pendiente:
			v.Pendiente++
		case NoAplica:
			v.NoAplica++
		}
	}
	out := make([]VistaMarco, 0, len(orden))
	for _, urn := range orden {
		out = append(out, *porURN[urn])
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Aplica != out[j].Aplica {
			return out[i].Aplica > out[j].Aplica
		}
		return out[i].URN < out[j].URN
	})
	return out
}
