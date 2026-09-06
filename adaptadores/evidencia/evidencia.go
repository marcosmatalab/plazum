// Package evidencia empareja lo que una norma pide con lo que el cliente ya
// tiene escrito, y lo hace SIN MODELO.
//
// # Que contesta, y que NO contesta
//
// Contesta: «este parrafo de TU politica, pagina 4, habla de esto». Es la pieza
// 3 de `docs/ia.md`, el mapeo de la evidencia que ya tiene.
//
// NO contesta si eso CUMPLE. Y la frontera esta puesta ahi a proposito, no por
// prudencia general: decidir que un parrafo satisface una obligacion es un
// juicio, y equivocarse tiene dos formas y las dos son caras. Un falso «lo
// cumples» esconde un incumplimiento detras de una pantalla verde. Un falso «no
// lo cumples» es acusar en falso, que es el unico error que un producto de
// cumplimiento no puede cometer ni una vez. Lo que si se puede hacer sin
// juicio, y es lo que de verdad ahorra la tarde, es LLEVAR A LA PERSONA AL
// PARRAFO. Eso es esto.
//
// # POR QUE ES DETERMINISTA, y por que eso es la decision y no una limitacion
//
// Es el mismo argumento que la pieza 2 (`superficies/pantallas/consecuencia.go`)
// y por los mismos tres motivos:
//
//   - es RECONTABLE: BM25 con parametros fijos da el mismo resultado dos veces,
//     y una pantalla que le dice a alguien donde mirar en su propia politica
//     tiene que poder explicarse;
//   - funciona con PLAZUM_SIN_IA=1, o sea que refuerza la puerta del invariante
//     9 en vez de gastarla;
//   - y marca donde SI hace falta el modelo, que es en la PARAFRASIS: esto
//     encuentra «revision anual» en un documento que dice «revision anual», y no
//     lo encuentra en uno que dice «cada doce meses». Ese hueco esta medido
//     abajo y es exactamente el trabajo del modelo, cuando llegue.
//
// # TODO HALLAZGO PASA POR EL VERIFICADOR, aunque hoy sea tautologia
//
// La cita que sale de aqui se verifica por hash contra la fuente antes de
// devolverse, igual que si la hubiera escrito un modelo. HOY eso es casi una
// tautologia, porque la cita la copia el buscador del propio fragmento, y se
// dice en voz alta para que nadie la lea como una garantia que no es.
//
// Lo que compra es la costura: el dia que la propuesta la escriba un modelo, la
// linea que verifica ya esta puesta y en el sitio correcto. Una costura que se
// anade despues es una costura que alguien se salta.
package evidencia

import (
	"errors"
	"fmt"
	"strings"

	"github.com/marcosmatalab/plazum/adaptadores/busqueda"
	"github.com/marcosmatalab/plazum/adaptadores/ia"
	"github.com/marcosmatalab/plazum/puertos"
)

// MinimoAciertos es cuantos terminos DISCRIMINANTES de lo que pide la norma
// tienen que aparecer en el fragmento para que se ensene.
//
// POR QUE UN CONTAJE DE TERMINOS Y NO UN UMBRAL DE PUNTUACION. La puntuacion de
// BM25 no esta normalizada: su valor depende del tamano del indice y de la
// frecuencia de los terminos, asi que un umbral absoluto significa una cosa
// sobre una politica de tres paginas y otra distinta sobre un inventario de
// doscientas. Un umbral que cambia de significado segun el documento no es un
// umbral, es una loteria con cara de constante.
//
// # LA PALABRA QUE HACE EL TRABAJO ES «DISCRIMINANTES», Y COSTO UN ROJO
//
// La primera version contaba aciertos a secas, y su medida contra el corpus
// real dio 291 de 549 obligaciones «con evidencia» en una politica de doce
// parrafos. Un 53 %, que era una cifra excelente y falsa: la muestra enseno que
// el marcado de contenido sintetico del AI Act, el contenido de la notificacion
// del art. 33.3 del RGPD y las instrucciones al usuario del CRA casaban los
// tres con el MISMO parrafo, el del registro de accesos, porque compartian
// «sistemas» e «informacion».
//
// El fallo no era el numero dos: era que un termino que aparece en casi todos
// los fragmentos NO DICE NADA SOBRE CUAL ES EL BUENO. Casa con todos por
// construccion. Asi que antes de buscar se tiran los terminos cuya `df` pasa de
// `FraccionDiscriminante`, y el minimo se cuenta sobre los que quedan.
//
// Y ES DE LA FAMILIA: la cifra se equivocaba A FAVOR. Es la quinta vez que una
// medida de este proyecto se equivoca y la quinta en la direccion que nos
// favorece. La caza no fue una lectura del codigo: fue mirar la muestra de lo
// que la medida estaba contando como acierto.
const MinimoAciertos = 4

// MinimoAbsoluto es el suelo cuando la consulta es tan corta que no tiene
// MinimoAciertos terminos discriminantes que ofrecer.
//
// POR QUE EL MINIMO NO PUEDE SER UN NUMERO FIJO A SECAS, medido el 06-09-2026 y
// no supuesto: las obligaciones del corpus son articulos largos (decenas de
// terminos) y las preguntas de la entrevista son frases de una linea. Con 4
// fijo, un articulo del ENS empareja bien y la pregunta «tienes copias de
// seguridad» no empareja NUNCA, porque no tiene cuatro terminos discriminantes
// que dar. Con 2 fijo, la pregunta empareja y el articulo empareja con
// cualquier cosa: 291 de 549 obligaciones, casi todas mal.
//
// Asi que se exige lo que la consulta PUEDE dar: cuatro terminos, o todos los
// que tenga si tiene menos, y nunca menos de dos.
const MinimoAbsoluto = 2

// FraccionDiscriminante es en que parte del indice puede aparecer un termino
// para que siga diciendo algo sobre CUAL fragmento es el bueno.
//
// Un tercio, y el numero es una decision: por encima de eso el termino casa con
// tantos fragmentos que su presencia no elige ninguno. No se puede afinar
// mirando este numero solo, porque lo que hay que mirar es la MUESTRA, y por
// eso el test contra el corpus real la imprime.
const FraccionDiscriminante = 0.34

// MinimoTermino es la longitud por debajo de la cual un termino de la consulta
// no cuenta. Junto con `vacias`, es lo que hace que MinimoAciertos signifique
// algo: sin filtrar, «de la que» ya son tres aciertos.
const MinimoTermino = 4

// vacias son las palabras que no aportan a la busqueda, en los dos idiomas que
// carga el catalogo.
//
// NO ES UNA LISTA DE NORMAS ni de contenido normativo, asi que no toca el
// invariante 2: son articulos, preposiciones y verbos auxiliares. Van escritas
// y no derivadas de una frecuencia porque una lista derivada del propio corpus
// cambia cuando cambia el corpus, y entonces el mismo documento del cliente
// daria hallazgos distintos segun que paquetes tenga instalados quien mira.
var vacias = map[string]bool{
	// castellano
	"para": true, "como": true, "cuando": true, "donde": true, "porque": true,
	"esta": true, "este": true, "esto": true, "esos": true, "esas": true,
	"sobre": true, "entre": true, "desde": true, "hasta": true, "segun": true,
	"debe": true, "debera": true, "deben": true, "puede": true, "pueden": true,
	"todos": true, "todas": true, "cualquier": true, "mismo": true, "misma": true,
	"tiene": true, "tienen": true, "haber": true, "hacer": true, "ser": true,
	// ingles
	"that": true, "this": true, "these": true, "those": true, "with": true,
	"from": true, "shall": true, "must": true, "will": true, "have": true,
	"been": true, "which": true, "when": true, "where": true, "such": true,
	"other": true, "than": true, "into": true, "each": true, "any": true,
}

// Los errores. Centinelas, por lo mismo que en el resto del bloque: quien llama
// tiene que poder distinguir «no habia con que buscar» de «no se encontro
// nada», que son las dos primeras formas de la nada.
var (
	ErrSinIndice      = errors.New("evidencia: sin indice sobre el que buscar")
	ErrSinVerificador = errors.New("evidencia: sin verificador con el que comprobar la cita")
	ErrSinConsultas   = errors.New("evidencia: sin nada que buscar")
)

// Consulta es lo que pide una norma, con su identificador.
//
// El texto es el de la obligacion o el de la pregunta de la entrevista, tal
// como lo escribe el paquete. No se reformula: reformularlo aqui seria una
// segunda copia de lo que dice la norma, y la que se queda vieja es siempre la
// copia.
type Consulta struct {
	ID    string
	Texto string
}

// Hallazgo es un parrafo del cliente que habla de lo que pide una norma.
//
// NO DICE QUE LO CUMPLA. El nombre del tipo es deliberado: es un hallazgo, no
// un veredicto.
type Hallazgo struct {
	ConsultaID string
	// Fuente y Referencia son de donde sale: el identificador del fragmento y
	// «pagina 4», o «fragmento 7, pagina desconocida» cuando no se supo.
	Fuente     string
	Referencia string
	// Cita es el texto LITERAL del documento del cliente, ya verificado por
	// hash contra la fuente.
	Cita       string
	Puntuacion float64
	Aciertos   int
}

// Mapear busca, por cada consulta, el fragmento del cliente que mas se le
// parece, y devuelve solo los que pasan el minimo.
//
// UNA CONSULTA SIN HALLAZGO NO ES UN FALLO Y NO SALE. Devolver un hallazgo
// flojo «por no dejarlo vacio» es la version de este paquete de rellenar con lo
// primero que suene bien: la pantalla diria que la politica del cliente habla de
// algo de lo que no habla, y quien lo lea dejaria de creerse el resto.
func Mapear(idx *busqueda.Indice, v *ia.Verificador, cs []Consulta) ([]Hallazgo, error) {
	if idx == nil {
		return nil, fmt.Errorf("%w. Arreglo: construye el indice con los documentos que "+
			"ha subido el cliente (ia.Documentos sobre ia.FuentesAportadas). Un indice "+
			"nil no es 'busca en todo': es que no hay donde buscar", ErrSinIndice)
	}
	if v == nil {
		return nil, fmt.Errorf("%w. Arreglo: pasa un verificador que admita ia.Aportado. "+
			"Sin el, lo que se ensena es lo que dijo el buscador y no lo que pone en el "+
			"documento, y esa diferencia es la puerta antialucinacion entera",
			ErrSinVerificador)
	}
	if len(cs) == 0 {
		return nil, fmt.Errorf("%w. Arreglo: pasa las obligaciones o las preguntas contra "+
			"las que mapear", ErrSinConsultas)
	}

	var out []Hallazgo
	for _, c := range cs {
		consulta := discriminantes(idx, c.Texto)
		if consulta == "" {
			// La consulta existe y no tiene ni un termino con contenido: es la
			// tercera forma de la nada, pero de una sola consulta y no de la
			// llamada, asi que se salta y no se inventa un hallazgo.
			continue
		}
		res, err := idx.Buscar(consulta, 1)
		if err != nil || len(res) == 0 {
			continue
		}
		r := res[0]
		if r.Aciertos < exigidos(consulta) {
			continue
		}
		ver, err := v.Verificar(puertos.Propuesta{
			Diff:       "senalar evidencia para " + c.ID,
			Cita:       r.Texto,
			HashFuente: r.Hash,
			Modelo:     "ninguno: busqueda determinista",
		})
		if err != nil {
			// Una cita que no verifica NO se ensena. Hoy esto no deberia
			// ocurrir nunca (la cita la copia el buscador del propio
			// fragmento), y por eso mismo si ocurre es que algo se ha
			// desincronizado entre el indice y las fuentes: callarlo seria
			// ensenar texto sin comprobar.
			continue
		}
		out = append(out, Hallazgo{
			ConsultaID: c.ID,
			Fuente:     ver.Fuente(),
			Referencia: ver.Articulo(),
			Cita:       ver.Cita(),
			Puntuacion: r.Puntuacion,
			Aciertos:   r.Aciertos,
		})
	}
	return out, nil
}

// exigidos es cuantos aciertos hacen falta para ESTA consulta: ver
// MinimoAbsoluto.
func exigidos(consulta string) int {
	n := len(strings.Fields(consulta))
	if n > MinimoAciertos {
		n = MinimoAciertos
	}
	if n < MinimoAbsoluto {
		n = MinimoAbsoluto
	}
	return n
}

// discriminantes deja de una consulta solo los terminos que dicen algo sobre
// CUAL fragmento del cliente es el bueno.
//
// Dos filtros y los dos hacen falta. El primero es del idioma y no del indice
// (largo minimo y palabras vacias): sin el, «de la que» ya son tres aciertos.
// El segundo es DEL INDICE, y es el que faltaba: un termino que esta en mas de
// FraccionDiscriminante de los fragmentos casa con casi todos por construccion,
// asi que contarlo como acierto es contar el tamano del documento.
func discriminantes(idx *busqueda.Indice, texto string) string {
	tope := int(float64(idx.Documentos()) * FraccionDiscriminante)
	if tope < 1 {
		// Con uno, dos o tres fragmentos cualquier termino esta en «muchos».
		// Se deja pasar todo y manda el minimo de aciertos: un indice asi es
		// demasiado pequeno para que la discriminacion signifique algo, y
		// fingir lo contrario seria un umbral inventado.
		tope = 1
	}
	var out []string
	visto := map[string]bool{}
	for _, t := range busqueda.Tokenizar(texto) {
		if len([]rune(t)) < MinimoTermino || vacias[t] || visto[t] {
			continue
		}
		visto[t] = true
		if df := idx.EnCuantos(t); df == 0 || df > tope {
			continue
		}
		out = append(out, t)
	}
	return strings.Join(out, " ")
}
