package plazum

import (
	"sort"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/aplicabilidad"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// EL PUENTE, MEDIDO EN TODOS LOS PAQUETES QUE LO DECLARAN.
//
// # Hubo un piloto, y termino
//
// Este fichero nacio midiendo UN paquete: declarar el puente en los treinta
// cuesta treinta veces mas que en uno, y si el diseno estaba mal se descubria
// igual. Se piloto con el que mas preguntas tiene y se midio. El diseno movio
// el numero, asi que el 04-09-2026 el puente se declaro en los 21 paquetes con
// reglas y esto paso a medirlos a todos, con el escenario maximo derivado de
// cada uno (ver escenarioMaximoDe y paquetesConPuente).
//
// LO QUE SE CONSERVA DEL PILOTO ES LO QUE VALIA: que el numero se MIDA y no se
// suponga, y que suba o baje con puerta.
//
// # Lo que este test SI puede afirmar, y antes no
//
// El 02-09-2026 se intento contar cuantas obligaciones derivaria un alcance
// sacado de la entrevista y NO SE PUDO: para contarlo hay que decir que hecho
// produce cada respuesta, y esa traduccion no existia en ninguna parte, asi que
// cualquier numero habria salido de una regla inventada para medir. Se
// descartaron dos resultados incompatibles por eso.
//
// Ahora la traduccion la declara el paquete (`hecho` en cada atributo, ver
// nucleo/corpus/puente.go) y la valida el linter contra las reglas del propio
// paquete. Asi que el numero de abajo **no depende de ningun mapeo inventado
// por el test**: sale de lo que el paquete afirma de si mismo.
//
// # Y lo que sigue sin poder afirmar, dicho
//
// Cuantas obligaciones derivan depende de QUE CONTESTE el operador: con
// `ambito` a sector publico salen unas y con sector privado contratista salen
// otras, y eso no es una limitacion de la medida, es lo que hace un motor de
// aplicabilidad. Por eso el escenario va escrito y con nombre, y el numero se
// lee siempre junto a el.

// PaquetesQueDeclaranElPuente es el cardinal de la ADOPCION, con igualdad
// exacta en los dos sentidos. Sube cuando un paquete declara su traduccion;
// baja cuando alguien se la quita, y ese es el caso que importa, porque un
// paquete sin bloque `hecho` deja de derivar EN SILENCIO.
//
// # De 1 a 21, y cual es el denominador
//
// El 04-09-2026 lo declararon los 21 paquetes del corpus QUE TIENEN REGLAS DE
// APLICABILIDAD, que es el denominador defendible: el puente traduce una
// respuesta a un hecho, y un hecho que ninguna regla lee no es un puente, es
// una afirmacion sin destino. Los otros 12 directorios de `paquetes/` son
// esqueletos sin reglas y sin obligaciones, y declarar el puente en ellos no
// seria posible (el linter exige que alguna regla use el predicado) ni util.
//
// El numero NO se escribe a mano en ningun otro sitio: quien quiera el
// denominador lo cuenta con `len(p.Aplicabilidad.Reglas) > 0`, que es lo que
// hace este mismo fichero mas abajo.
const PaquetesQueDeclaranElPuente = 21

// ObligacionesQueDerivaElPuente son las que enciende el ESCENARIO MAXIMO de
// todos los paquetes que lo declaran.
//
// # De 30 a 29, y la decima no se ha perdido: se ha dejado de suponer
//
// Hasta el 04-09-2026 esta medida usaba un escenario escrito A MANO, con un
// valor inventado para un atributo de texto libre, y daba 30. El escenario
// maximo se deriva de la declaracion y NO puede inventar ese valor: para un
// texto, un entero o una fecha, «el valor que mas enciende» no existe, porque
// las reglas prueban constantes concretas y el valor lo pone el operador.
//
// Asi que 29 es el techo DEMOSTRABLE y 30 era el techo con una suposicion
// dentro. La obligacion que falta no ha desaparecido: se ha quedado sin quien
// afirme que su valor es ese.
//
// # De 29 a 207 el 04-09-2026, y que significa el 207
//
// Es el techo de los 21 paquetes juntos: el escenario maximo de cada uno,
// afirmado a la vez sobre el mismo sujeto. Es una cota SUPERIOR y por
// construccion imposible (el mismo sujeto sale a la vez entidad financiera,
// microempresa, registro de nombres de dominio y fabricante de producto con
// elementos digitales), y eso es lo que mide: cuanto puede llegar a encender la
// declaracion del corpus entero, no cuanto le toca a nadie.
//
// LO QUE NO ES: no es «207 obligaciones para el que conteste que si a todo en
// la pantalla». La pantalla de hoy solo sabe mandar si/no, y las 25 preguntas
// de forma `con_valor` no tienen por donde llegar; ese cardinal aparte vive en
// entrevista_alcanza_al_motor_test.go.
const ObligacionesQueDerivaElPuente = 207

// hechosDelPuente llama a la traduccion del producto y falla el test si esta se
// niega. La traduccion vive en nucleo/corpus.HechosDeLaEntrevista.
func hechosDelPuente(t *testing.T, p *corpus.Paquete,
	rs []corpus.RespuestaDeEntrevista) []aplicabilidad.Hecho {

	t.Helper()
	hs, err := corpus.HechosDeLaEntrevista(p, rs)
	if err != nil {
		t.Fatalf("traduciendo la entrevista: %v", err)
	}
	return hs
}

// TestElPuenteDeclaradoDerivaObligacionesDeVerdad es la medida del piloto.
//
// Monta el motor EXACTAMENTE como lo monta `plazum calendario` (cargar los
// programas de todos los paquetes con reglas, afirmar los hechos, evaluar), le
// da los hechos que salen de la declaracion del paquete y cuenta.
func TestElPuenteDeclaradoDerivaObligacionesDeVerdad(t *testing.T) {
	ps, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatalf("cargar el corpus: %v", err)
	}
	conPuente := paquetesConPuente(t, ps)

	motor := aplicabilidad.NuevoMotor()
	cargados := 0
	for _, p := range ps {
		if len(p.Aplicabilidad.Reglas) == 0 {
			continue
		}
		prog, errs := p.Programa()
		if len(errs) > 0 {
			t.Fatalf("las reglas de %s no compilan: %v", p.URN, errs)
		}
		if err := motor.Cargar(prog); err != nil {
			t.Fatalf("el motor rechaza las reglas de %s: %v", p.URN, err)
		}
		cargados++
	}
	if cargados < 2 {
		t.Fatalf("solo se han cargado %d programas: el motor no tiene contra que derivar",
			cargados)
	}

	// SE AFIRMAN LOS HECHOS DE TODOS Y SE MIDE UNA VEZ, y no paquete a paquete
	// con el motor limpio, a proposito: las reglas de un marco pueden leer un
	// predicado que declara otro (los predicados son compartidos), asi que
	// medir en aislamiento daria un numero que el producto nunca produce.
	hechos := 0
	for _, p := range conPuente {
		for _, h := range hechosDelPuente(t, p, escenarioMaximoDe(p)) {
			hecho := h
			hecho.Procedencia = "declarado en la entrevista"
			motor.Afirmar(hecho)
			hechos++
		}
	}
	if _, err := motor.Evaluar(); err != nil {
		t.Fatalf("evaluando: %v", err)
	}

	// LAS OBLIGACIONES QUE LE ALCANZAN AL SUJETO. Se consulta igual que
	// `aplicablesDe`, por el sujeto del escenario.
	vistas := map[string]bool{}
	for _, h := range motor.Consultar(aplicabilidad.A("aplica",
		aplicabilidad.V("O"), aplicabilidad.C("sis"))) {
		vistas[h.Args[0]] = true
	}
	derivadas := len(vistas)

	// LOS DOS CARDINALES, con igualdad exacta y en los dos sentidos.
	//
	// El de PAQUETES es el que dice cuanto ha avanzado la adopcion, y tiene que
	// romper tambien cuando BAJE: un paquete que pierde su bloque `hecho` deja
	// de derivar en silencio, que es el fallo que este bloque existe para
	// cerrar. El de OBLIGACIONES es la medida.
	if len(conPuente) != PaquetesQueDeclaranElPuente {
		var urns []string
		for _, p := range conPuente {
			urns = append(urns, p.URN)
		}
		t.Errorf("declaran el puente %d paquetes y la constante dice %d.\n"+
			"  Si ha SUBIDO, es adopcion y hay que decirlo aqui con el numero nuevo.\n"+
			"  Si ha BAJADO, un paquete ha perdido su bloque `hecho` y sus respuestas han "+
			"dejado de derivar EN SILENCIO, que es justo lo que este bloque cierra.\n"+
			"  Son: %v", len(conPuente), PaquetesQueDeclaranElPuente, urns)
	}
	if derivadas != ObligacionesQueDerivaElPuente {
		t.Errorf("el puente declarado deriva %d obligaciones y la constante dice %d.\n"+
			"  Si ha SUBIDO, el puente llega mas lejos y hay que decirlo aqui.\n"+
			"  Si ha BAJADO sin que nadie borre reglas, algo se ha desconectado en silencio.\n"+
			"  Derivadas: %v", derivadas, ObligacionesQueDerivaElPuente, ordenadas(vistas))
	}

	// Y EL SUELO, que es lo que el piloto vino a contestar: si con el puente
	// declarado y todo contestado no deriva NADA, el diseno esta mal.
	if derivadas == 0 {
		t.Fatal("con el puente declarado y el escenario entero contestado no deriva ni una " +
			"obligacion. El diseno del puente no sirve, y descubrirlo pronto es exactamente " +
			"para lo que estaba el piloto")
	}
	t.Logf("puente: %d paquetes lo declaran, %d hechos afirmados, %d obligaciones derivadas",
		len(conPuente), hechos, derivadas)
}

// paquetesConPuente devuelve TODOS los paquetes que declaran el puente, POR UNA
// PROPIEDAD y no por su nombre: escribir aqui el identificador de una norma
// rompe el invariante 2.
//
// # El piloto termino el 04-09-2026, y esta funcion es la decision que pedia
//
// Hasta ese dia esto devolvia UNO y se paraba con un `t.Fatal` en cuanto
// hubiera dos, con estas palabras: «se para aqui para que nadie lo amplie sin
// darse cuenta». La parada salto sola cuando el frente que iba a declarar los
// catorce restantes declaro el segundo. Funciono exactamente como se escribio.
//
// La decision: el puente deja de ser un piloto y pasa a medirse en cada paquete
// que lo declare, con el ESCENARIO MAXIMO de cada uno (ver escenarioMaximoDe).
// Lo que se conserva del piloto es lo que valia: que el numero se mida y no se
// suponga, y que baje o suba con puerta.
func paquetesConPuente(t *testing.T, ps []*corpus.Paquete) []*corpus.Paquete {
	t.Helper()
	var con []*corpus.Paquete
	for _, p := range ps {
		if p.DeclaraPuente() {
			con = append(con, p)
		}
	}
	if len(con) == 0 {
		t.Fatal("ningun paquete declara el puente. O se han borrado todos, o el lector no " +
			"esta viendo el bloque `hecho`, y entonces esta medida seria un verde vacio")
	}
	sort.Slice(con, func(i, j int) bool { return con[i].URN < con[j].URN })
	return con
}

// escenarioMaximoDe contesta que SI a todo lo contestable de un paquete: cada
// booleano marcado y cada enumerado en su primer valor.
//
// # Por que el maximo y no un escenario realista
//
// Porque lo que este test mide es el TECHO del puente: cuanto puede llegar a
// encender la declaracion de ese paquete. Un escenario realista mide ademas si
// la organizacion elegida es representativa, que es otra pregunta y con otra
// respuesta cada vez.
//
// Y CONTESTAR QUE SI A TODO ES PELIGROSO EN UN FORMULARIO Y NO AQUI, que es una
// distincion que conviene dejar escrita: el frente que midio esto descarto usar
// un enumerado de un solo valor precisamente porque la superficie lo mandaria
// por defecto y afirmaria cosas que el operador no ha dicho. Aqui no hay
// operador: hay una medida del alcance, y el maximo es el numero que la
// describe.
func escenarioMaximoDe(p *corpus.Paquete) []corpus.RespuestaDeEntrevista {
	var rs []corpus.RespuestaDeEntrevista
	for _, e := range p.Entidades {
		for _, a := range e.Atributos {
			if a.Hecho == nil {
				continue
			}
			r := corpus.RespuestaDeEntrevista{
				Entidad: e.Nombre, Instancia: "sis", Atributo: a.Nombre, Si: true,
			}
			if a.Hecho.Forma == corpus.PuenteConValor {
				// UN `con_valor` SIN DOMINIO ENUMERADO NO ENTRA EN EL MAXIMO, y
				// no es un olvido: para un texto, un entero o una fecha, «el
				// valor que mas enciende» no existe, porque el valor lo pone el
				// operador y las reglas prueban constantes concretas. Inventarle
				// uno daria un hecho que no casa con nada y contaria como
				// medido. Se salta, y el test cuenta cuantos se salta.
				if len(a.Valores) == 0 {
					continue
				}
				r.Valor = a.Valores[0]
			}
			rs = append(rs, r)
		}
	}
	return rs
}

// ordenadas da las claves de un conjunto en orden, para que el mensaje de fallo
// sea el mismo dos veces seguidas. Recorrer un mapa de Go no tiene orden, y una
// lista que cambia de orden en cada ejecucion hace que dos salidas iguales
// parezcan distintas.
func ordenadas(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
