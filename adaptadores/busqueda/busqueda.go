// Package busqueda es la busqueda de texto completo sobre el corpus, con
// ranking BM25. Cero dependencias externas.
//
// QUE ES Y QUE NO ES. La casilla de ETAPAS.md dice "Busqueda FTS5 (BM25)". Esto
// es el BM25 sin el FTS5: un indice invertido en memoria, construido al
// arrancar desde los paquetes ya cargados, con la MISMA funcion de ranking y
// los mismos parametros por defecto que `bm25()` de SQLite (k1=1,2 y b=0,75).
//
// POR QUE ASI, dicho entero porque es un apartamiento de la casilla y no se
// disimula:
//
//   - FTS5 entra con `modernc.org/sqlite`, y eso es una DEPENDENCIA. Este
//     repositorio se compila hoy con cero (`go.mod` sin una sola linea
//     `require`), lo vigila `TestElBinarioNoLlevaNingunaDependenciaExterna`, y
//     una puerta de CI corre la suite entera con GOPROXY=off. Anadirla no es
//     una linea en go.mod: es una decision de producto con su fila en
//     DEPENDENCIAS.md.
//   - Y el tamano medido dice que hoy no hace falta: el corpus transcrito son
//     28.675 tokens en 321 obligaciones, 3.099 terminos distintos. Eso cabe en
//     memoria de sobra y se recorre en microsegundos. FTS5 empieza a valer la
//     pena con los documentos que sube el cliente, que es la pieza 1 y 7 de
//     `docs/ia.md` y llega despues.
//
// El contrato de este paquete esta escrito para que el cambio a FTS5 sea un
// adaptador nuevo y no una reescritura: entra `[]Documento`, sale
// `[]Resultado` ordenado. La peticion formal de la dependencia, con licencia y
// porque, esta en `docs/hallazgos-ia.md`.
//
// LO QUE ESTE PAQUETE NO HACE, y no es un olvido: no decide si un texto se
// puede citar. Eso es la frontera legal (invariante 3) y vive en
// `adaptadores/ia`, que es quien construye los documentos que entran aqui. Un
// indice no puede ser la guarda de una frontera legal, porque su trabajo es
// encontrar cosas.
package busqueda

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

// Los parametros de BM25, con los valores por defecto de FTS5.
//
// No son ajustables desde fuera A PROPOSITO: un parametro de ranking que el
// operador puede mover convierte "que sale primero" en algo que depende de la
// instalacion, y entonces un eval que mide precision deja de medir nada
// comparable entre dos maquinas.
const (
	k1 = 1.2
	b  = 0.75
)

// Documento es una unidad citable ya decidida por quien construye el indice.
//
// Hash es el campo por el que este indice se empareja con el verificador de
// citas, y es el sha256 del texto. Va aqui, y no un indice de posicion, por el
// invariante 7: nadie firma el orden de una lista, asi que reordenar el corpus
// no puede mover a que articulo apunta un resultado.
type Documento struct {
	ID       string
	Hash     string
	Marco    string
	Articulo string
	Texto    string
}

// Resultado es un documento con su puntuacion.
type Resultado struct {
	Documento
	Puntuacion float64
	// Aciertos es cuantos terminos DISTINTOS de la consulta aparecen en el
	// documento. Sale porque la puntuacion sola no distingue un documento que
	// casa con los tres terminos de uno que casa mucho con uno solo, y esa es
	// justo la diferencia que necesita ver quien construye el contexto de un
	// modelo.
	Aciertos int
}

// Los errores del indice. Todos son centinelas para que quien llama pueda
// distinguirlos, que es lo que separa "no se pudo comprobar" de "no hay nada".
var (
	ErrDocumentoSinID    = errors.New("busqueda: documento sin identificador")
	ErrDocumentoSinHash  = errors.New("busqueda: documento sin hash de fuente")
	ErrDocumentoSinTexto = errors.New("busqueda: documento cuyo texto no da ni un termino")
	ErrIDRepetido        = errors.New("busqueda: dos documentos con el mismo identificador")
	ErrSinTope           = errors.New("busqueda: tope de resultados no declarado")
	ErrConsultaVacia     = errors.New("busqueda: consulta vacia")
	ErrConsultaIlegible  = errors.New("busqueda: consulta que no da ni un termino")
)

type aparicion struct {
	doc   int
	veces int
}

// Indice es un indice invertido inmutable. Se construye entero y no se toca:
// una estructura de solo lectura se puede compartir entre peticiones sin
// candado y sin sorpresas en el detector de carreras.
type Indice struct {
	docs      []Documento
	largos    []int
	largoDeUn float64
	post      map[string][]aparicion
}

// Nuevo construye el indice.
//
// EL VALOR CERO, dicho porque el invariante 8 obliga a decirlo por cada
// estructura que cruza una frontera. `Nuevo(nil)` y `Nuevo([]Documento{})`
// construyen los DOS un indice vacio, y los dos contestan cero resultados a
// cualquier consulta. Aqui la nada es restrictiva por si sola y no hace falta
// prohibirla: en un buscador no existe el "sin restriccion" que en un almacen
// de certificados significa "acepto cualquier CA". No hay comodin que se pueda
// dejar suelto. Las dos formas tienen su caso en el test igualmente, porque la
// afirmacion "aqui la nada es inocua" solo vale si alguien la recorre.
func Nuevo(docs []Documento) (*Indice, error) {
	i := &Indice{post: map[string][]aparicion{}}
	vistos := make(map[string]bool, len(docs))
	total := 0
	for n, d := range docs {
		if d.ID == "" {
			return nil, fmt.Errorf("%w: el documento %d de la lista no trae ID. "+
				"Arreglo: todo documento entra con el identificador de su obligacion, "+
				"que es por donde se le vuelve a encontrar", ErrDocumentoSinID, n)
		}
		if d.Hash == "" {
			return nil, fmt.Errorf("%w: %s. Arreglo: el hash es el campo por el que un "+
				"resultado se empareja con la fuente que el verificador de citas resuelve. "+
				"Sin el, un resultado no se puede citar", ErrDocumentoSinHash, d.ID)
		}
		if vistos[d.ID] {
			return nil, fmt.Errorf("%w: %s. Arreglo: dos documentos con el mismo ID hacen "+
				"que el orden de la lista decida cual gana, y el orden no lo firma nadie",
				ErrIDRepetido, d.ID)
		}
		vistos[d.ID] = true

		terminos := Tokenizar(d.Texto)
		if len(terminos) == 0 {
			return nil, fmt.Errorf("%w: %s. Arreglo: un documento presente cuyo texto no se "+
				"puede interpretar no es un documento vacio, es un dato que hay y no se "+
				"entiende. Se rechaza en vez de indexarlo mudo", ErrDocumentoSinTexto, d.ID)
		}

		idx := len(i.docs)
		i.docs = append(i.docs, d)
		i.largos = append(i.largos, len(terminos))
		total += len(terminos)

		cuenta := map[string]int{}
		for _, t := range terminos {
			cuenta[t]++
		}
		for t, v := range cuenta {
			i.post[t] = append(i.post[t], aparicion{doc: idx, veces: v})
		}
	}
	if len(i.docs) > 0 {
		i.largoDeUn = float64(total) / float64(len(i.docs))
	}
	return i, nil
}

// Documentos es cuantos documentos hay dentro. Existe para que quien monta una
// puerta pueda poner un suelo: un indice que se ha quedado vacio contesta cero
// resultados igual que un indice lleno al que no le casa la consulta, y las dos
// cosas se leen exactamente igual desde fuera.
func (i *Indice) Documentos() int { return len(i.docs) }

// Buscar devuelve hasta `tope` resultados ordenados por BM25.
//
// LOS TRES VALORES DEGENERADOS, uno por cada forma de la nada del invariante 8:
//
//	tope <= 0        NO significa "todos". Un tope sin declarar que devolviera
//	                 el corpus entero es el cero permisivo de manual, y ademas
//	                 mete 321 articulos en el contexto de un modelo. Error.
//	consulta ""      ausente. Error.
//	consulta "!!!"   PRESENTE Y NO INTERPRETABLE, que es la tercera forma. No
//	                 se puede tratar como la anterior ni devolver todo: es un
//	                 dato que hay y no se entiende, y eso siempre es error.
func (i *Indice) Buscar(consulta string, tope int) ([]Resultado, error) {
	if tope <= 0 {
		return nil, fmt.Errorf("%w: se ha pedido tope %d. Arreglo: di cuantos resultados "+
			"quieres. Un tope sin declarar no significa 'todos'", ErrSinTope, tope)
	}
	if consulta == "" {
		return nil, fmt.Errorf("%w. Arreglo: una consulta vacia no es 'ensename todo', "+
			"es una pregunta que falta", ErrConsultaVacia)
	}
	terminos := Tokenizar(consulta)
	if len(terminos) == 0 {
		return nil, fmt.Errorf("%w. Arreglo: la consulta trae caracteres pero ninguno es "+
			"letra ni digito, asi que no hay nada que buscar. Es un dato presente que no se "+
			"entiende, y por eso sale error y no una lista vacia", ErrConsultaIlegible)
	}

	n := float64(len(i.docs))
	puntos := map[int]float64{}
	aciertos := map[int]int{}
	vistos := map[string]bool{}
	for _, t := range terminos {
		if vistos[t] {
			// Un termino repetido en la consulta no puntua dos veces. BM25
			// puntua el documento contra el CONJUNTO de terminos; contar la
			// repeticion deja que quien escribe la consulta infle un resultado
			// escribiendo la misma palabra cinco veces.
			continue
		}
		vistos[t] = true
		apariciones := i.post[t]
		if len(apariciones) == 0 {
			continue
		}
		df := float64(len(apariciones))
		idf := math.Log(1 + (n-df+0.5)/(df+0.5))
		for _, a := range apariciones {
			f := float64(a.veces)
			norma := f + k1*(1-b+b*float64(i.largos[a.doc])/i.largoDeUn)
			puntos[a.doc] += idf * (f * (k1 + 1)) / norma
			aciertos[a.doc]++
		}
	}

	out := make([]Resultado, 0, len(puntos))
	for doc, p := range puntos {
		out = append(out, Resultado{Documento: i.docs[doc], Puntuacion: p, Aciertos: aciertos[doc]})
	}
	// EL DESEMPATE POR ID NO ES COSMETICO. Sin el, dos documentos con la misma
	// puntuacion salen en el orden en que el recorrido de un map los saco, que
	// en Go es aleatorio por diseno. Eso convierte al buscador en una fuente de
	// no-determinismo dentro del camino de la IA, y hace que un eval de
	// precision de resultados distintos en dos ejecuciones de la misma maquina.
	sort.Slice(out, func(a, z int) bool {
		if out[a].Puntuacion != out[z].Puntuacion {
			return out[a].Puntuacion > out[z].Puntuacion
		}
		return out[a].ID < out[z].ID
	})
	if len(out) > tope {
		out = out[:tope]
	}
	return out, nil
}
