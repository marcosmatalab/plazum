package plazum

import (
	"sort"
	"testing"

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
// Y nadie comprobaba que lo primero pudiera producir lo segundo.
//
// # POR QUE ESTA MEDIDA CAMBIO DE ANCLA EL 04-09-2026, y era obligatorio
//
// Hasta ese dia esto se medía POR LA ARIDAD con la que las reglas usaban un
// atributo: si alguna regla lo usaba con un argumento, la pregunta «llegaba al
// motor»; si lo usaban siempre con dos, «necesitaba un valor»; si no lo usaba
// nadie, «no llegaba a ningun sitio». Era una heuristica razonable mientras la
// traduccion no existiera en ninguna parte, que era el estado del mundo cuando
// se escribio.
//
// Ahora la traduccion la DECLARA EL PAQUETE, atributo a atributo, en el bloque
// `hecho` (nucleo/corpus/puente.go), y la valida el linter contra las reglas.
// Mantener la heuristica al lado de la declaracion serian DOS IMPLEMENTACIONES
// DE LA MISMA MEDIDA, y el dia que se separaran ganaria la que nadie mira. Peor:
// la heuristica se equivoca ahora en las dos direcciones, y las dos se ven en el
// corpus de hoy.
//
//	un atributo BOOLEANO cuyo «si» afirma `predicado(instancia, CONSTANTE)`
//	(forma `afirma_si_valor`) sale por aridad como «necesita un valor que la
//	entrevista no pregunta», y es falso: la entrevista solo tiene que mandar el
//	si, porque la constante la pone el paquete. Son 14 preguntas del corpus;
//	un atributo declarado `no_llega_al_motor` con su motivo sale por aridad
//	como «traducible» en cuanto otro paquete use un predicado que se llame
//	igual, y no llega a ninguna parte a proposito.
//
// Asi que la medida se ancla a lo unico que es una AFIRMACION DEL CORPUS SOBRE
// SI MISMO, que es el bloque `hecho`. La heuristica desaparece: no se conserva
// «por si acaso», porque una segunda cuenta es lo que produce dos numeros
// incompatibles del mismo hecho.
//
// # Que hace esta puerta, y que NO hace
//
// No arregla el puente. Lo MIDE desde el arbol y lo CONGELA con sus cardinales,
// para que el numero no se mueva en silencio en ninguna de las dos direcciones.

// PreguntasQueNoLleganAlMotor es EL CARDINAL DEL HUECO, y es un trinquete.
//
// Es el numero de preguntas del corpus instalado cuya respuesta NO produce
// ningun hecho, y son de dos clases distintas que se cuentan juntas a
// proposito, porque las dos dejan al operador igual de lejos del calendario:
//
//	sin puente declarado   su atributo no trae bloque `hecho`. Es deuda: nadie
//	                       ha dicho todavia que afirma esa respuesta;
//	callejon declarado     su atributo declara `no_llega_al_motor` CON SU
//	                       MOTIVO. No es deuda, es una decision escrita y
//	                       auditable, y casi todas son hechos fechados que
//	                       alimentan el reloj en vez de la regla.
//
// Se comprueba por igualdad EXACTA, igual que PUERTAS_ESPERADAS en
// comprobar.sh, y por la misma razon: si SUBE, alguien ha ensanchado el hueco;
// si BAJA, alguien lo ha estrechado y tiene que bajar el numero aqui, en el
// mismo commit. Un techo que solo se comprueba por arriba se queda viejo.
//
// Hoy, 04-09-2026, sobre las 68 preguntas del corpus instalado: 0 sin puente
// declarado y 16 callejones con su motivo escrito.
const PreguntasQueNoLleganAlMotor = 16

// PreguntasQueLaPantallaSabeMandar es el OTRO cardinal, y existe porque sin el
// la medida de arriba se vuelve tramposa al reanclarla.
//
// Reanclar a la declaracion hace bajar PreguntasQueNoLleganAlMotor de golpe, y
// eso podria leerse como «el hueco se ha cerrado». No se ha cerrado: la
// entrevista web solo sabe mandar `si` y `no` (superficies/pantallas.De lee
// ParamSi y ParamNo, nada mas), asi que una pregunta cuya forma es `con_valor`
// produce un hecho EN EL CORPUS y hoy no tiene por donde llegar. Este numero
// cuenta las que si tienen por donde: las de forma `afirma_si` y
// `afirma_si_valor`, que son exactamente las que un si/no basta para afirmar.
//
// Es la regla de la casa sobre las cifras cuyo fallo probable es FAVORECERTE:
// la que baja sola lleva al lado la que no baja sola.
const PreguntasQueLaPantallaSabeMandar = 27

// TotalDePreguntasDelCorpus se congela por la misma razon: sin el, el hueco de
// arriba se podria "cerrar" borrando preguntas, que es la forma barata de bajar
// un numero sin arreglar nada.
const TotalDePreguntasDelCorpus = 68

// puenteDeUnaPregunta es en que estado esta una pregunta respecto del motor.
// Sale de la FORMA que declara su atributo, no de una heuristica.
type puenteDeUnaPregunta uint8

const (
	// sinPuenteDeclarado: su atributo no trae bloque `hecho`. Nadie ha dicho
	// que afirma esa respuesta, asi que no afirma nada.
	//
	// ES EL VALOR CERO DEL ENUMERADO A PROPOSITO (invariante 8): si el
	// clasificador se rompe y devuelve el cero, la pregunta cuenta como «no
	// llega», que es lo pesimista. Lo contrario haria que un fallo del
	// clasificador se leyera como un puente que no existe.
	sinPuenteDeclarado puenteDeUnaPregunta = iota
	// callejonDeclarado: `no_llega_al_motor`, con su motivo escrito.
	callejonDeclarado
	// necesitaValorQueLaPantallaNoManda: forma `con_valor`. Produce un hecho,
	// y el valor lo pone la respuesta, que es lo que la entrevista web no sabe
	// mandar todavia.
	necesitaValorQueLaPantallaNoManda
	// llegaConUnSi: `afirma_si` o `afirma_si_valor`. Un si basta.
	llegaConUnSi
)

// formasDeclaradas indexa la forma del puente por (paquete, entidad, atributo),
// que es la MISMA clave con la que corpus.HechosDeLaEntrevista busca la
// declaracion. Emparejar por otra cosa mediria una relacion que el producto no
// usa (invariante 7); y las tres partes de la clave viven dentro del paquete
// firmado, igual que el bloque `hecho` que se busca.
func formasDeclaradas(ps []*corpus.Paquete) map[[3]string]string {
	out := map[[3]string]string{}
	for _, p := range ps {
		for _, e := range p.Entidades {
			for _, a := range e.Atributos {
				if a.Hecho != nil {
					out[[3]string{p.URN, e.Nombre, a.Nombre}] = a.Hecho.Forma
				}
			}
		}
	}
	return out
}

// puenteDeLaEntrevista clasifica cada pregunta del corpus por lo que su
// atributo DECLARA.
func puenteDeLaEntrevista(t *testing.T, ps []*corpus.Paquete) map[puenteDeUnaPregunta][]string {
	t.Helper()
	formas := formasDeclaradas(ps)
	if len(formas) == 0 {
		t.Fatal("ningun atributo del corpus declara el puente: esta puerta estaria midiendo " +
			"el vacio, y todo caeria en el mismo cubo dando un verde sin contenido")
	}
	out := map[puenteDeUnaPregunta][]string{}
	for _, q := range corpus.Entrevista(ps) {
		f, hay := formas[[3]string{q.Paquete, q.Entidad, q.Atributo}]
		switch {
		case !hay:
			out[sinPuenteDeclarado] = append(out[sinPuenteDeclarado], q.ID)
		case f == corpus.PuenteNoLlegaAlMotor:
			out[callejonDeclarado] = append(out[callejonDeclarado], q.ID)
		case f == corpus.PuenteConValor:
			out[necesitaValorQueLaPantallaNoManda] =
				append(out[necesitaValorQueLaPantallaNoManda], q.ID)
		default:
			// afirma_si y afirma_si_valor: las dos se afirman con un si.
			out[llegaConUnSi] = append(out[llegaConUnSi], q.ID)
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

	sinPuente := len(puente[sinPuenteDeclarado])
	callejones := len(puente[callejonDeclarado])
	conValor := len(puente[necesitaValorQueLaPantallaNoManda])
	conUnSi := len(puente[llegaConUnSi])
	total := sinPuente + callejones + conValor + conUnSi
	noLlegan := sinPuente + callejones

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
			"  su atributo no declara el puente (%d): %v\n"+
			"  declaradas callejon con su motivo (%d): %v\n"+
			"  producen un hecho con el valor de la respuesta (%d): %v\n"+
			"  producen un hecho con un si (%d): %v",
			noLlegan, PreguntasQueNoLleganAlMotor, direccion,
			sinPuente, puente[sinPuenteDeclarado],
			callejones, puente[callejonDeclarado],
			conValor, puente[necesitaValorQueLaPantallaNoManda],
			conUnSi, puente[llegaConUnSi])
	}

	// EL SEGUNDO CARDINAL, y es el que no baja solo. Ver su godoc: sin el, el
	// reanclaje de arriba se leeria como que el hueco se ha cerrado.
	if conUnSi != PreguntasQueLaPantallaSabeMandar {
		t.Errorf("preguntas que la pantalla de hoy sabe mandar: %d, y la constante dice %d.\n"+
			"  Son las de forma afirma_si y afirma_si_valor: las unicas que un si/no basta "+
			"para afirmar. Las de forma con_valor (%d) producen un hecho en el corpus y no "+
			"tienen por donde llegar hasta que la entrevista aprenda a mandar valores.\n"+
			"  Son: %v", conUnSi, PreguntasQueLaPantallaSabeMandar, conValor,
			puente[llegaConUnSi])
	}

	// EL CONTROL POSITIVO DE LA CLASIFICACION. Sin esto, un clasificador que
	// metiera TODO en un solo cubo cuadraria igual con las constantes y la
	// puerta seria un contador de preguntas con nombre bonito.
	//
	// SE EXIGEN LOS TRES CUBOS QUE HOY TIENEN CONTENIDO Y NO LOS CUATRO: el de
	// `sinPuenteDeclarado` esta VACIO desde que los 21 paquetes con reglas
	// declaran su traduccion, y exigir que tenga contenido seria exigir que la
	// deuda no se cierre nunca. Su cero lo vigila la igualdad exacta de arriba.
	if conUnSi == 0 || conValor == 0 || callejones == 0 {
		t.Errorf("la clasificacion ha dejado vacio un cubo que hoy tiene contenido "+
			"(con un si: %d, con valor: %d, callejones: %d). Los tres tienen preguntas "+
			"dentro, asi que un cubo vacio aqui es el clasificador roto, no el corpus",
			conUnSi, conValor, callejones)
	}
}

// LO QUE ESTA PUERTA NO AFIRMA, Y POR QUE SE QUEDO FUERA.
//
// La primera version traia un segundo test que contaba CUANTAS obligaciones
// derivaria un alcance.json sacado de la entrevista. Se cayo sola y merece
// quedar escrito, porque el fallo es de los que se publican sin notarlos.
//
// Para contar eso hay que decir QUE HECHO produce un «si» a una pregunta, y ESA
// TRADUCCION NO EXISTIA EN NINGUNA PARTE cuando esto se escribio: no la
// declaraba el paquete, no la declaraba el motor y no la declaraba la
// superficie. O sea que cualquier numero que saliera de ahi seria el numero de
// una regla inventada para medir, con la FORMA de un dato verificable y sin nada
// detras. Ademas salia mal: la primera pasada dio «cero de 26 en el ENS» mirando
// cuerpo a cuerpo, y al hacerlo por punto fijo dio «16 de 171», porque los
// predicados se comparten entre paquetes y `en_ambito` lo deriva otra regla de
// otro paquete. Dos numeros incompatibles de la misma pregunta, y ninguno
// auditable.
//
// HOY LA TRADUCCION EXISTE Y ESE NUMERO SE MIDE, pero no aqui: lo mide
// puente_piloto_test.go, con el escenario maximo derivado de cada paquete y su
// propio cardinal. Escribirlo tambien aqui seria la misma segunda cuenta que
// este fichero acaba de quitarse de encima.
