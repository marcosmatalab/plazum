package plazum

import (
	"sort"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/aplicabilidad"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// EL TRINQUETE DE ALCANZABILIDAD, TERCERA MITAD: EL PUENTE ENTRE LA ENTREVISTA
// Y EL MOTOR.
//
// # Como salio
//
// Buscando el hueco que impide que `plazum serve` produzca el alcance.json que
// su propio camino guiado pide en dos pasos (`plazum calendario --alcance` y
// `plazum escalado --alcance`). El hueco no era el boton de descarga: es que
// LAS DOS MITADES HABLAN IDIOMAS DISTINTOS.
//
//	la entrevista de superficies/pantallas recoge SI o NO sobre un id de
//	pregunta, y nada mas: superficies/pantallas.De solo lee los parametros
//	`si` y `no`;
//	el motor de aplicabilidad consume HECHOS TIPADOS, `ambito(sistema, publico)`,
//	`categoria(sistema, alta)`, con su valor dentro.
//
// Y nadie comprobaba que lo primero pudiera producir lo segundo. Es el CUARTO
// caso de la familia de este bloque, y el mas caro de los cuatro: la pantalla
// de Alcance esta construida y verde, el motor esta construido y verde, y la
// junta entre los dos no la mira ninguna puerta.
//
// # Por que es caro de verdad
//
// Un alcance.json exportado hoy desde la entrevista cargaria SIN ERROR y
// llevaria dentro mucho menos de lo que aparenta, y esa es la peor combinacion
// que este producto puede producir: un fichero con la FORMA de la respuesta
// completa. De las 41 preguntas del corpus, 36 no pueden convertirse en un
// hecho que el motor lea, y de esas hay 16 cuyo atributo las reglas usan
// SIEMPRE con dos o mas argumentos: la entrevista no pregunta el valor, asi que
// afirmarlas exigiria inventarselo. El operador leeria un calendario corto como
// «no me alcanza casi nada», que es una respuesta que plazum no puede dar sin
// saberla.
//
// # Que hace esta puerta, y que NO hace
//
// No arregla el puente: eso es una decision de producto (o la entrevista
// aprende a llevar valores, o el paquete declara como se traduce una pregunta a
// un hecho) y no se toma dentro de un test. Lo que hace es MEDIRLO desde el
// arbol y CONGELARLO con su cardinal, para que el numero no se mueva en
// silencio en ninguna de las dos direcciones.

// PreguntasQueNoLleganAlMotor es EL CARDINAL DEL HUECO, y es un trinquete.
//
// Es el numero de preguntas del corpus instalado cuya respuesta NO se puede
// convertir hoy en un hecho que el motor entienda. Se comprueba por igualdad
// EXACTA, igual que PUERTAS_ESPERADAS en comprobar.sh, y por la misma razon:
//
//	si SUBE, alguien ha ensanchado el hueco y tiene que enterarse el mismo dia;
//	si BAJA, alguien lo ha estrechado y tiene que bajar el numero aqui, en el
//	mismo commit. Un techo que solo se comprueba por arriba se queda viejo y
//	deja de molestar, y este numero existe justamente para molestar.
//
// Hoy, 02-09-2026, sobre las 41 preguntas del corpus instalado: 5 se pueden
// traducir desde un si/no (su atributo lo usan las reglas como predicado
// unario), 16 necesitan un VALOR que la entrevista no pregunta, y 20 tienen un
// atributo que no usa ninguna regla, o sea que su respuesta no llega a ningun
// sitio.
const PreguntasQueNoLleganAlMotor = 36

// TotalDePreguntasDelCorpus se congela por la misma razon: sin el, el hueco de
// arriba se podria "cerrar" borrando preguntas, que es la forma barata de bajar
// un numero sin arreglar nada.
const TotalDePreguntasDelCorpus = 41

// puenteDeUnaPregunta es en que estado esta una pregunta respecto del motor.
type puenteDeUnaPregunta uint8

const (
	// TraducibleDesdeSiNo: alguna regla usa su atributo como predicado de UN
	// solo argumento, o sea que afirmar el hecho no necesita ningun valor y un
	// «si» de la entrevista basta para producirlo.
	traducibleDesdeSiNo puenteDeUnaPregunta = iota
	// NecesitaValor: las reglas usan su atributo con dos o mas argumentos. La
	// entrevista no pregunta el valor, asi que la respuesta no se puede afirmar
	// sin inventarselo, y inventarlo es lo que no se hace.
	necesitaValor
	// SinPredicado: ninguna regla del corpus usa ese atributo. La respuesta no
	// llega a ningun sitio, ni con valor ni sin el.
	sinPredicado
)

// aridadesDeLosPredicados lee, del corpus REAL y con el parser de verdad, con
// cuantos argumentos usa cada regla cada predicado.
//
// Se usa aplicabilidad.ParsearRegla y no una expresion regular a proposito: la
// regla ya tiene un parser en el producto, y medir con un segundo parser
// escrito al lado seria medir otra cosa el dia que los dos se separen.
func aridadesDeLosPredicados(t *testing.T, ps []*corpus.Paquete) map[string]map[int]bool {
	t.Helper()
	out := map[string]map[int]bool{}
	anotar := func(a aplicabilidad.Atomo) {
		if out[a.Pred] == nil {
			out[a.Pred] = map[int]bool{}
		}
		out[a.Pred][len(a.Args)] = true
	}
	reglas := 0
	for _, p := range ps {
		for _, rs := range p.Aplicabilidad.Reglas {
			r, err := aplicabilidad.ParsearRegla(rs.Regla)
			if err != nil {
				// El linter de paquetes ya rechaza una regla ilegible, asi que
				// aqui esto solo puede pasar si el corpus esta roto: se dice y
				// no se traga, porque tragarlo bajaria el cardinal sin motivo.
				t.Fatalf("%s/%s no parsea y el linter deberia haberlo impedido: %v",
					p.URN, rs.ID, err)
			}
			reglas++
			anotar(r.Cabeza)
			for _, a := range r.Cuerpo {
				anotar(a)
			}
			for _, a := range r.Negados {
				anotar(a)
			}
		}
	}
	if reglas == 0 {
		t.Fatal("el corpus instalado no trae ni una regla de aplicabilidad: esta puerta " +
			"estaria midiendo el vacio")
	}
	return out
}

// puenteDeLaEntrevista clasifica cada pregunta del corpus.
func puenteDeLaEntrevista(t *testing.T, ps []*corpus.Paquete) map[puenteDeUnaPregunta][]string {
	t.Helper()
	aridades := aridadesDeLosPredicados(t, ps)
	out := map[puenteDeUnaPregunta][]string{}
	for _, q := range corpus.Entrevista(ps) {
		usos, hay := aridades[q.Atributo]
		switch {
		case !hay || q.Atributo == "":
			out[sinPredicado] = append(out[sinPredicado], q.ID)
		case usos[1]:
			// Basta con que ALGUNA regla lo use como unario: con eso, un «si»
			// produce un hecho que alguna regla lee.
			out[traducibleDesdeSiNo] = append(out[traducibleDesdeSiNo], q.ID)
		default:
			out[necesitaValor] = append(out[necesitaValor], q.ID)
		}
	}
	for k := range out {
		sort.Strings(out[k])
	}
	return out
}

// TestElHuecoEntreLaEntrevistaYElMotorNoCreceEnSilencio es el trinquete.
//
// NACIO ROJA SOBRE EL ARBOL REAL en el sentido que importa: nadie sabia que el
// numero era 37 de 41 hasta que se midio, y el hueco llevaba ahi desde que
// existen las dos mitades. Ninguna mutacion habria dado eso, porque nadie lo
// habia metido.
func TestElHuecoEntreLaEntrevistaYElMotorNoCreceEnSilencio(t *testing.T) {
	ps, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatalf("cargar el corpus: %v", err)
	}
	puente := puenteDeLaEntrevista(t, ps)

	traducibles := len(puente[traducibleDesdeSiNo])
	conValor := len(puente[necesitaValor])
	huerfanas := len(puente[sinPredicado])
	total := traducibles + conValor + huerfanas
	noLlegan := conValor + huerfanas

	if total != TotalDePreguntasDelCorpus {
		t.Errorf("el corpus trae %d preguntas y la constante dice %d.\n"+
			"  Si han crecido, sube el numero; si han menguado, comprueba que no se esta "+
			"cerrando el hueco borrando preguntas, que es la forma barata de bajar un "+
			"numero sin arreglar nada.", total, TotalDePreguntasDelCorpus)
	}

	if noLlegan != PreguntasQueNoLleganAlMotor {
		direccion := "HA CRECIDO: alguien ha ensanchado el hueco"
		if noLlegan < PreguntasQueNoLleganAlMotor {
			direccion = "HA MENGUADO: alguien lo ha estrechado, y hay que bajar la constante " +
				"en este mismo commit"
		}
		t.Errorf("preguntas que NO llegan al motor: %d, y la constante dice %d. %s.\n"+
			"  necesitan un valor que la entrevista no pregunta (%d): %v\n"+
			"  su atributo no lo usa ninguna regla (%d): %v\n"+
			"  traducibles desde un si/no (%d): %v",
			noLlegan, PreguntasQueNoLleganAlMotor, direccion,
			conValor, puente[necesitaValor],
			huerfanas, puente[sinPredicado],
			traducibles, puente[traducibleDesdeSiNo])
	}

	// EL CONTROL POSITIVO DE LA CLASIFICACION. Sin esto, un clasificador que
	// metiera TODO en un solo cubo cuadraria igual con las dos constantes y la
	// puerta seria un contador de preguntas con nombre bonito.
	if traducibles == 0 {
		t.Error("ninguna pregunta sale como traducible desde un si/no, y hay al menos una " +
			"(ens.q.datos_personales, sobre el predicado unario trata_datos_personales). " +
			"El clasificador esta metiendo todo en el mismo cubo")
	}
	if conValor == 0 || huerfanas == 0 {
		t.Errorf("la clasificacion ha dejado un cubo vacio (con valor: %d, huerfanas: %d). "+
			"Hoy los tres tienen contenido, asi que un cubo vacio es el clasificador roto",
			conValor, huerfanas)
	}
}

// LO QUE ESTA PUERTA NO AFIRMA, Y POR QUE SE QUEDO FUERA.
//
// La primera version traia un segundo test que contaba CUANTAS obligaciones
// derivaria un alcance.json sacado de la entrevista. Se cayo sola y merece
// quedar escrito, porque el fallo es de los que se publican sin notarlos.
//
// Para contar eso hay que decir QUE HECHO produce un «si» a una pregunta, y ESA
// TRADUCCION NO EXISTE EN NINGUNA PARTE: no la declara el paquete, no la
// declara el motor y no la declara la superficie. O sea que cualquier numero
// que saliera de ahi seria el numero de una regla que me habria inventado yo
// para medir, con la FORMA de un dato verificable y sin nada detras. Ademas
// salia mal: la primera pasada dio «cero de 26 en el ENS» mirando cuerpo a
// cuerpo, y al hacerlo por punto fijo dio «16 de 171», porque los predicados se
// comparten entre paquetes y `en_ambito` lo deriva otra regla de otro paquete.
// Dos numeros incompatibles de la misma pregunta, y ninguno auditable.
//
// LO QUE SI SE AFIRMA ARRIBA no depende de ninguna traduccion inventada, y por
// eso se queda:
//
//	que 20 preguntas tienen un atributo que NO USA NINGUNA REGLA es un hecho
//	estructural del corpus: su respuesta no llega a ningun sitio, con mapeo o
//	sin el;
//	que 16 preguntas tienen su atributo usado SIEMPRE con dos o mas argumentos
//	es otro hecho estructural, y de el se sigue sin inventar nada que una
//	respuesta de si/no no lo puede afirmar: no hay donde poner el valor.
//
// La consecuencia (cuantas obligaciones saldrian) es real y es la que importa
// para el producto, pero no se puede medir hasta que exista la traduccion. Es
// justamente la decision que este hallazgo pone sobre la mesa.
