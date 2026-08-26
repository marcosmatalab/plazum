// Package pantalla deriva el MODELO de las pantallas desde el corpus instalado.
//
// Por que existe, y por que no es un puerto. Se propuso como puerto (UIGenerada)
// y se retiro: un puerto existe para poder sustituir la implementacion, y aqui
// no queremos dos derivaciones distintas del mismo corpus, queremos una y
// comprobable. Consta en docs/puertos-propuestas.md.
//
// Que hace y que no. Aqui se decide QUE se ensena y en QUE ORDEN, a partir de
// los paquetes instalados y de nada mas. No se decide como se pinta: eso es del
// puerto Plantilla y de su adaptador. La frontera esta puesta ahi a proposito,
// porque es la que hace verificable la promesa de producto: anadir la norma 31
// es un fichero de datos, y la interfaz cambia sola.
//
// Reglas duras, las tres:
//
//	determinista  los mismos paquetes dan exactamente el mismo modelo, campo a
//	              campo y en el mismo orden. Sin mapas recorridos sin ordenar,
//	              sin reloj, sin aleatoriedad. Lo prueban los casos dorados de
//	              testdata/, que son ficheros y se comparan byte a byte.
//	sin texto     los rotulos de la interfaz son CLAVES de catalogo, no texto.
//	              El texto de la interfaz lo pone el puerto Catalogo.
//	sin traducir  el contenido que viene del corpus (etiqueta de un atributo,
//	              ayuda, cita, texto de una pregunta) viaja tal cual, en el
//	              idioma del paquete, y NO pasa por el catalogo. Traducir texto
//	              transcrito del BOE crea obra derivada y se sale de la
//	              estratificacion de licencias: un paquete en otro idioma es un
//	              paquete distinto con su fuente, no una traduccion al vuelo.
package pantalla

import (
	"sort"
	"strings"
	"time"

	"plazum/nucleo/corpus"
)

// ID identifica una pantalla. Son las seis de la etapa 2 y estan aqui, no en el
// adaptador web, porque el modelo tiene que poder comprobarse sin levantar nada.
type ID string

const (
	Alcance      ID = "alcance"
	Hoy          ID = "hoy"
	Controles    ID = "controles"
	Certificados ID = "certificados"
	Personas     ID = "personas"
	Estado       ID = "estado"
)

// Origen dice de donde sale el contenido de una pantalla. Importa para el
// operador: una pantalla vacia porque no hay corpus instalado y una pantalla
// vacia porque todavia no hay estado son dos problemas distintos con dos
// arreglos distintos, y confundirlos es una llamada de soporte.
type Origen uint8

const (
	// DelCorpus: el contenido se deriva de los paquetes instalados. Sin
	// paquetes esta vacia, y el arreglo es instalar corpus.
	DelCorpus Origen = iota
	// DelEstado: el contenido sale del expediente y del reloj. Sin estado
	// esta vacia, y el arreglo es completar el alcance.
	DelEstado
)

func (o Origen) String() string {
	switch o {
	case DelCorpus:
		return "corpus"
	case DelEstado:
		return "estado"
	}
	return "desconocido"
}

// Pantalla es una pantalla ya derivada.
type Pantalla struct {
	ID     ID     `json:"id"`
	Titulo string `json:"titulo"` // CLAVE de catalogo, nunca texto
	Origen Origen `json:"origen"`
	// Vacia dice si el modelo salio sin contenido, y PorQue lo explica en
	// clave de catalogo. Una pantalla en blanco sin explicacion es la forma
	// mas rapida de perder a quien acaba de instalar esto.
	Vacia  bool   `json:"vacia"`
	PorQue string `json:"porque,omitempty"`

	Preguntas []Pregunta `json:"preguntas,omitempty"`
	Campos    []Campo    `json:"campos,omitempty"`
	Filas     []Fila     `json:"filas,omitempty"`

	// Planificador es el estado del vigilante, y solo lo lleva Hoy. Va en el
	// modelo y no en la superficie porque la regla que decide si un
	// planificador esta muerto es de dominio, con un numero dentro, y tiene
	// que ser la misma en la pantalla, en la terminal y en el diagnostico.
	//
	// Es puntero para que las otras cinco pantallas no lo lleven: una
	// pantalla que trae un veredicto de vigilancia en cero se leeria como un
	// veredicto, y "todo a cero" es justo el aspecto que tiene "correcto".
	Planificador *Planificador `json:"planificador,omitempty"`

	// Fuentes son los paquetes instalados con su aviso de derechos. Van en
	// TODAS las pantallas, no solo en las que pintan contenido del corpus, y
	// eso es una decision, no un descuido: ver derivarFuentes.
	Fuentes []Fuente `json:"fuentes,omitempty"`
}

// Fuente es un paquete instalado visto desde el lado de los derechos: de donde
// sale su contenido y que puede hacer con el quien lo tiene delante.
//
// POR QUE ESTO ES PARTE DEL MODELO Y NO UN FICHERO DEL REPOSITORIO. La Decision
// 2011/833/UE autoriza reutilizar el DOUE con ATRIBUCION. Un LICENCIAS.md en el
// repositorio le cuenta eso a quien lee el codigo fuente; no se lo cuenta a
// quien USA el producto, que es a quien hay que contarselo. Asi que el aviso
// viaja en el paquete, entra en el modelo derivado y sale en pantalla, con una
// puerta que se pone roja si deja de salir.
//
// Atribucion es TEXTO del corpus y no clave de catalogo: no se traduce. Un
// aviso de derechos parafraseado por la interfaz deja de ser el aviso, y ademas
// traducirlo cae en la misma regla que traducir el texto transcrito.
type Fuente struct {
	URN            string `json:"urn"`
	Version        string `json:"version"`
	LicenciaFuente string `json:"licencia_fuente"`
	Atribucion     string `json:"atribucion"`
	// Enlace es la fuente oficial del paquete, DERIVADA de su identificador
	// estable (corpus.Identificador.Enlace) y no copiada de un campo del
	// fichero de datos. Puede no ser una direccion web (un paquete propio
	// apunta a su LEEME): quien pinte esto decide si hace enlace o texto, que
	// es presentacion y no modelo.
	//
	// Se deriva AQUI, al construir el modelo de la pantalla, y no al cargar el
	// paquete: el enlace no es dato del corpus, es una vista de su
	// identificador. Guardarlo en el paquete lo volveria a convertir en dato, y
	// una direccion como dato es lo que se rompe el dia que la pagina se mueve.
	Enlace string `json:"enlace"`
	// Identificador es el identificador estable del paquete ("ISO/IEC
	// 27001:2022", "4.0"). Va en el modelo porque es lo que SOBREVIVE a que el
	// editor mueva sus paginas: si el enlace derivado deja de resolver, esto
	// sigue diciendo que norma es.
	//
	// Sale VACIO cuando el enlace ya lo lleva dentro, y eso no esconde nada:
	// en un ELI la direccion es el prefijo del editor mas el identificador
	// entero, asi que ensenar los dos es imprimir la misma cadena dos veces en
	// el pie de todas las paginas. Donde el enlace NO lo lleva es donde de
	// verdad hace falta, y son los dos casos que importan: ISO (su catalogo
	// esta indexado por un numero de registro, no por la designacion) y PCI
	// DSS (una sola biblioteca para todas las versiones, asi que sin la
	// version a la vista el lector no sabe que documento coger).
	Identificador string `json:"identificador,omitempty"`
}

// Pregunta es una pregunta de la entrevista de alcance, ya ordenada por cuantas
// obligaciones desbloquea. Nunca se ensena un catalogo de controles en frio.
type Pregunta struct {
	ID          string   `json:"id"`
	Texto       string   `json:"texto"` // del corpus, en el idioma del paquete
	Ayuda       string   `json:"ayuda,omitempty"`
	Cita        string   `json:"cita"`
	Entidad     string   `json:"entidad"`
	Atributo    string   `json:"atributo"`
	Paquete     string   `json:"paquete"`
	NDesbloquea int      `json:"n_desbloquea"`
	Desbloquea  []string `json:"desbloquea,omitempty"`
}

// Peticion es UNA norma pidiendo el dato, con la cita y la ayuda que da ELLA.
// Viaja tal cual desde el corpus, sin pasar por el catalogo: es contenido del
// paquete, en el idioma del paquete.
type Peticion struct {
	Paquete string `json:"paquete"`
	Cita    string `json:"cita"`
	Ayuda   string `json:"ayuda,omitempty"`
}

// Campo es un campo de formulario. Paquetes dice quien pide el dato: es lo que
// convierte "rellena esto" en "esto lo piden estas tres normas".
//
// Peticiones dice POR QUE lo pide cada una. Antes solo sobrevivia la cita de
// una de las tres, asi que a "por que me piden este dato" se respondia con el
// articulo de la norma que quedara primera por orden de URN. Ayuda y Cita se
// mantienen, con ese mismo significado, para no romper a quien ya las pinta.
type Campo struct {
	Entidad    string     `json:"entidad"`
	Atributo   string     `json:"atributo"`
	Etiqueta   string     `json:"etiqueta"`
	Tipo       string     `json:"tipo"`
	Valores    []string   `json:"valores,omitempty"`
	Obligado   bool       `json:"obligado"`
	Ayuda      string     `json:"ayuda,omitempty"`
	Cita       string     `json:"cita"`
	Paquetes   []string   `json:"paquetes"`
	Peticiones []Peticion `json:"peticiones"`
}

// Fila es una fila de tabla. Sirve para Controles y para Certificados: la forma
// es la misma y el contenido lo pone quien deriva.
type Fila struct {
	ID       string            `json:"id"`
	Paquete  string            `json:"paquete"`
	Columnas map[string]string `json:"columnas"`
	// Requiere son los IDs de pregunta que hay que responder para saber si
	// esta fila aplica. Vacio significa que aplica siempre.
	Requiere []string `json:"requiere,omitempty"`
}

// Entorno es lo que la derivacion necesita saber ADEMAS del corpus: el
// instante y las marcas de la propia instalacion.
//
// Va en un tipo propio y no en dos parametros sueltos porque lo que entra por
// aqui va a crecer (el expediente, el reloj legal), y cada crecimiento seria si
// no una firma nueva que rompe a todos los frentes a la vez.
type Entorno struct {
	// Ahora es el instante desde el que se juzga. En cero, la vigilancia lo
	// dice en vez de dar nada por bueno.
	Ahora time.Time
	// Marcas es lo que la instalacion sabe de si misma.
	Marcas Marcas
}

// Derivar construye las seis pantallas desde los paquetes instalados.
//
// Es DerivarCon con el entorno vacio, y sigue existiendo con esta firma porque
// hay codigo compilando contra ella. El entorno vacio no finge estar bien: la
// pantalla Hoy sale diciendo que no se le paso el instante.
func Derivar(ps []*corpus.Paquete) []Pantalla { return DerivarCon(ps, Entorno{}) }

// DerivarCon construye las seis pantallas desde los paquetes instalados y el
// entorno.
//
// Recibe []*corpus.Paquete y no un esquema ya masticado a proposito: la
// derivacion ES el contrato, y asi se puede comprobar entera con un fichero de
// entrada y un fichero de salida.
func DerivarCon(ps []*corpus.Paquete, e Entorno) []Pantalla {
	vigilancia := Vigilar(e.Marcas, e.Ahora)
	out := []Pantalla{
		derivarAlcance(ps),
		{ID: Hoy, Titulo: "pantalla.hoy.titulo", Origen: DelEstado,
			Vacia: true, PorQue: "pantalla.hoy.vacia", Planificador: &vigilancia},
		derivarControles(ps),
		derivarCertificados(ps),
		{ID: Personas, Titulo: "pantalla.personas.titulo", Origen: DelEstado,
			Vacia: true, PorQue: "pantalla.personas.vacia"},
		{ID: Estado, Titulo: "pantalla.estado.titulo", Origen: DelEstado,
			Vacia: true, PorQue: "pantalla.estado.vacia"},
	}
	// La atribucion se pone AQUI, en un solo sitio y para todas, y no dentro
	// de cada derivador. Si cada pantalla se la pusiera a si misma, la
	// septima pantalla que alguien anada nacera sin ella, y una atribucion
	// que se pierde al crecer el producto no es una atribucion.
	fuentes := derivarFuentes(ps)
	for i := range out {
		// Copia por pantalla, no la misma rebanada seis veces. Compartir el
		// array de respaldo hace que tocar el aviso de una pantalla toque el
		// de las otras cinco, y ese es justo el tipo de acoplamiento que no se
		// ve en el diff: seis pantallas que parecen independientes y no lo
		// son. Son seis copias de una lista de treinta: no se nota.
		out[i].Fuentes = append([]Fuente(nil), fuentes...)
	}
	return out
}

// derivarFuentes lista los paquetes instalados con su aviso de derechos.
//
// POR QUE EN LAS SEIS PANTALLAS Y NO EN UNA SOLA. La obligacion de atribuir
// acompana al uso del contenido, y el contenido del corpus se usa en Alcance
// (el texto de cada pregunta y su cita), en Controles y en Certificados. Una
// pagina de avisos legales aparte, a la que se llega si a uno le da por
// buscarla, deja las tres pantallas donde de verdad se lee la norma sin el
// aviso. Es el mismo razonamiento que ya se escribio para el descargo de
// asesoramiento juridico, y por la misma razon: un aviso que solo sale en la
// portada es un aviso que no se lee.
//
// Y por que tambien en las tres que salen del estado: porque el corpus esta
// instalado igual, y porque una regla con excepciones es una regla que alguien
// aplica mal el dia que anade una pantalla.
//
// Orden por URN y sin repetidos: el modelo tiene que ser el mismo aunque el
// cargador recorra el directorio en otro orden. corpus.Cargar ya rechaza dos
// paquetes con el mismo URN, pero Derivar recibe una lista que le pasa quien
// sea, y un aviso repetido dos veces se lee como un fallo del producto.
func derivarFuentes(ps []*corpus.Paquete) []Fuente {
	var out []Fuente
	vistos := map[string]bool{}
	for _, p := range ps {
		if p == nil || vistos[p.URN] {
			continue
		}
		vistos[p.URN] = true
		out = append(out, Fuente{
			URN:            p.URN,
			Version:        p.Version,
			LicenciaFuente: string(p.LicenciaFuente),
			Atribucion:     p.Atribucion,
			// El enlace se DERIVA aqui del identificador estable. Es el unico
			// sitio del modelo que llama a Enlace, y Enlace es la unica
			// funcion del producto que convierte identidad en direccion.
			Enlace:        p.Enlace(),
			Identificador: identificadorVisible(p),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].URN < out[j].URN })
	return out
}

// identificadorVisible devuelve el identificador estable SOLO cuando dice algo
// que el enlace derivado no diga ya. Ver el comentario de Fuente.Identificador.
//
// La comprobacion es por contenido y no por tipo de identificador a proposito:
// si manana un esquema nuevo mete el identificador dentro de la direccion, esto
// deja de repetirlo sin que nadie tenga que acordarse de anadirlo a una lista.
func identificadorVisible(p *corpus.Paquete) string {
	v := p.Identificador.Valor
	if v == "" || strings.Contains(p.Enlace(), v) {
		return ""
	}
	return v
}

func derivarAlcance(ps []*corpus.Paquete) Pantalla {
	p := Pantalla{ID: Alcance, Titulo: "pantalla.alcance.titulo", Origen: DelCorpus}
	for _, q := range corpus.Entrevista(ps) {
		p.Preguntas = append(p.Preguntas, Pregunta{
			ID: q.ID, Texto: q.Texto, Ayuda: q.Ayuda, Cita: q.Cita,
			Entidad: q.Entidad, Atributo: q.Atributo, Paquete: q.Paquete,
			NDesbloquea: q.NDesbloquea, Desbloquea: q.Desbloquea,
		})
	}
	for _, c := range corpus.EsquemaUI(ps) {
		// corpus.EsquemaUI deja Paquetes en el orden en que llegaron los
		// paquetes, asi que el modelo dependeria del orden de la llamada.
		// Se ordena aqui: el mismo corpus tiene que dar el mismo modelo
		// aunque el cargador recorra el directorio en otro orden.
		quien := append([]string(nil), c.Paquetes...)
		sort.Strings(quien)
		// corpus.EsquemaUI ya devuelve las peticiones en orden de URN, que es
		// el mismo orden que quien: se copian, no se reordenan.
		porque := make([]Peticion, 0, len(c.Peticiones))
		for _, x := range c.Peticiones {
			porque = append(porque, Peticion{Paquete: x.Paquete, Cita: x.Cita, Ayuda: x.Ayuda})
		}
		p.Campos = append(p.Campos, Campo{
			Entidad: c.Entidad, Atributo: c.Atributo, Etiqueta: c.Etiqueta,
			Tipo: c.Tipo, Valores: c.Valores, Obligado: c.Obligado,
			Ayuda: c.Ayuda, Cita: c.Cita, Paquetes: quien, Peticiones: porque,
		})
	}
	if len(p.Preguntas) == 0 && len(p.Campos) == 0 {
		p.Vacia, p.PorQue = true, "pantalla.alcance.sin_corpus"
	}
	return p
}

// derivarControles: una fila por obligacion de los paquetes instalados.
//
// Requiere lleva las preguntas que la desbloquean, y por eso Controles no es un
// listado plano: el operador ve que le falta responder para saber si un control
// le aplica, en vez de un catalogo de cientos de requisitos sin filtrar.
//
// PUNTO DE INTEGRACION DE LA VIGENCIA, pendiente y a proposito. nucleo/corpus ya
// sabe responder si una obligacion esta en vigor en un instante dado
// (corpus.Paquete.EnVigor y corpus.VigentesEn, con el instante entrando como
// dato). Aqui todavia no se usa: filtrar por vigencia obliga a meter el instante
// en la firma de Derivar, y hay otro frente compilando contra ella ahora mismo.
// Cuando se haga, DOS cosas y no una:
//
//  1. filtrar con p.EnVigor(o, instante), y
//  2. DECIRLO, y decir CUAL DE LOS DOS CASOS ES. Una obligacion que desaparece
//     de la lista sin explicacion se lee como un fallo del producto.
//
// Los dos casos, medidos sobre el corpus publicado a 25-08-2026:
//
//	derogada          una obligacion de las 132 de un paquete transcrito
//	                  termino en 2024-05-05 y hoy sale como las demas. Filtrarla
//	                  esta bien; esconderla sin mas, no: quiere su columna
//	                  ("derogada el 2024-05-05") o su propia seccion.
//	todavia no vigente  otro paquete tiene UNA obligacion que empieza en
//	                  2026-09-11, o sea que un filtro a secas deja ese marco
//	                  entero en blanco hoy. El operador leeria "esto no me
//	                  aplica" cuando lo que pasa es "esto te aplica dentro de
//	                  dos semanas", que es la respuesta contraria.
func derivarControles(ps []*corpus.Paquete) Pantalla {
	p := Pantalla{ID: Controles, Titulo: "pantalla.controles.titulo", Origen: DelCorpus}
	for _, pq := range ps {
		for _, o := range pq.Obligaciones {
			cols := map[string]string{
				// El titulo es la etiqueta legible, con respaldo: el campo es
				// opcional en el formato, asi que la pantalla no puede ensenar
				// una celda vacia porque el paquete no lo declare todavia.
				"titulo":     o.TituloLegible(),
				"articulo":   o.Articulo,
				"cita":       o.Cita,
				"clase_e2e":  o.ClaseE2E,
				"entregable": o.Entregable,
			}
			if o.Temporalidad != nil {
				cols["primitiva"] = o.Temporalidad.Primitiva
				cols["cadencia"] = o.Temporalidad.Cadencia
				cols["limite"] = o.Temporalidad.Limite
			}
			for k, v := range cols {
				if v == "" {
					delete(cols, k)
				}
			}
			p.Filas = append(p.Filas, Fila{
				ID: o.ID, Paquete: pq.URN, Columnas: cols,
				Requiere: append([]string(nil), o.Preguntas...),
			})
		}
	}
	ordenarFilas(p.Filas)
	if len(p.Filas) == 0 {
		p.Vacia, p.PorQue = true, "pantalla.controles.sin_corpus"
	}
	return p
}

// derivarCertificados: una fila por plantilla de entregable que declare un
// paquete. Lo que el operador se lleva al auditor sale de aqui.
func derivarCertificados(ps []*corpus.Paquete) Pantalla {
	p := Pantalla{ID: Certificados, Titulo: "pantalla.certificados.titulo", Origen: DelCorpus}
	// Que obligacion pide cada plantilla: sin esto, una plantilla es un
	// formulario huerfano y nadie sabe por que tiene que rellenarlo.
	porPlantilla := map[string][]string{}
	for _, pq := range ps {
		for _, o := range pq.Obligaciones {
			if o.Entregable != "" {
				porPlantilla[o.Entregable] = append(porPlantilla[o.Entregable], o.ID)
			}
		}
	}
	for _, pq := range ps {
		for _, pl := range pq.Plantillas {
			ob := append([]string(nil), porPlantilla[pl.ID]...)
			sort.Strings(ob)
			cols := map[string]string{"titulo": pl.Titulo, "cita": pl.Cita}
			for k, v := range cols {
				if v == "" {
					delete(cols, k)
				}
			}
			p.Filas = append(p.Filas, Fila{
				ID: pl.ID, Paquete: pq.URN, Columnas: cols, Requiere: ob,
			})
		}
	}
	ordenarFilas(p.Filas)
	if len(p.Filas) == 0 {
		p.Vacia, p.PorQue = true, "pantalla.certificados.sin_corpus"
	}
	return p
}

// ordenarFilas fija el orden por (paquete, id). Es lo que hace comparables los
// casos dorados: sin orden total, dos ejecuciones con los mismos paquetes dan
// modelos distintos y el dorado deja de probar nada.
func ordenarFilas(f []Fila) {
	sort.SliceStable(f, func(i, j int) bool {
		if f[i].Paquete != f[j].Paquete {
			return f[i].Paquete < f[j].Paquete
		}
		return f[i].ID < f[j].ID
	})
}
