package corpus

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/marcosmatalab/plazum/nucleo/aplicabilidad"
)

// EL PUENTE ENTRE LA ENTREVISTA Y EL MOTOR, DECLARADO POR EL PAQUETE.
//
// # El agujero que cierra
//
// La entrevista recoge respuestas sobre atributos de entidades. El motor de
// aplicabilidad consume hechos. Entre las dos cosas hay una traduccion, y hasta
// hoy NO EXISTIA EN NINGUNA PARTE: ni en el paquete, ni en el motor, ni en la
// superficie. Funcionaba por una convencion no escrita (el predicado se llama
// como el atributo, con aridad 1 si es booleano y 2 si lleva valor) que nadie
// comprobaba.
//
// Medido el 02-09-2026 sobre el corpus instalado: de 41 preguntas, 36 no
// llegaban al motor, y 20 de esas tenian un atributo que NO USA NINGUNA REGLA.
// Con la convencion sin escribir, esas veinte no dan error en ningun sitio: la
// respuesta del operador se recoge, se pinta, y no afirma nada.
//
// # Por que lo declara el PAQUETE y no el codigo
//
// Invariante 2, y no es una formalidad. Que «¿tratas datos personales?» produzca
// `trata_datos_personales(E)` es CONOCIMIENTO NORMATIVO: dice que ese hecho es
// el que las reglas de esa norma miran. Si esa tabla la escribe Go, hemos
// cableado una norma en el codigo por la puerta de atras, y el siguiente marco
// que necesite otro mapeo sera un cambio de producto en vez de un cambio de
// datos.
//
// # El emparejamiento, y por que campo casa (invariante 7)
//
// Casa por el NOMBRE DEL PREDICADO, y las dos puntas viven DENTRO del mismo
// paquete firmado: el `hecho` del atributo y la regla que lo lee estan en el
// mismo paquete.json. No hay indice ni posicion por medio. Y se comprueba en
// las dos direcciones: que el predicado declarado lo use alguna regla con la
// aridad que le toca, y que la aridad que le toca salga del TIPO del atributo,
// no de lo que el autor creyera.
//
// # Lo que este fichero NO hace todavia
//
// No obliga a los 30 paquetes a declararlo. El bloque es opcional y se valida
// CUANDO ESTA; el piloto es un solo paquete, a proposito, para medir si el
// diseno mueve el numero antes de pagarlo treinta veces. Cuando el piloto
// demuestre que sirve, esto pasa a obligatorio y el valor cero (no declararlo)
// sera el que rompa, como en `origen_del_intervalo`.

var (
	// ErrPuenteFormaDesconocida: `forma` fuera del vocabulario cerrado.
	ErrPuenteFormaDesconocida = errors.New("forma del hecho fuera del vocabulario")
	// ErrPuenteFormaNoCasaConTipo: un booleano que dice llevar valor, o al reves.
	ErrPuenteFormaNoCasaConTipo = errors.New("la forma del hecho no casa con el tipo del atributo")
	// ErrPuenteSinPredicado: dice afirmar algo y no dice que.
	ErrPuenteSinPredicado = errors.New("hecho sin predicado")
	// ErrPuentePredicadoSobrante: declara callejon Y predicado.
	ErrPuentePredicadoSobrante = errors.New("callejon con predicado")
	// ErrPuenteSinPorque: un callejon sin motivo escrito.
	ErrPuenteSinPorque = errors.New("callejon del puente sin motivo")
	// ErrPuentePredicadoHuerfano: ninguna regla del paquete usa ese predicado.
	ErrPuentePredicadoHuerfano = errors.New("predicado que ninguna regla usa")
	// ErrPuenteAridadDistinta: las reglas lo usan con otra aridad.
	ErrPuenteAridadDistinta = errors.New("predicado usado con otra aridad")
	// ErrPuenteValorHuerfano: el predicado y la aridad casan, y los VALORES que
	// el atributo puede tomar no los prueba ninguna regla en ese hueco.
	ErrPuenteValorHuerfano = errors.New("valores que ninguna regla mira")
	// ErrPuenteSinValorFijo: la cuarta forma dice afirmar una constante y no
	// dice cual. Sin ella el hecho saldria con el segundo argumento vacio.
	ErrPuenteSinValorFijo = errors.New("hecho sin el valor que afirma")
	// ErrPuenteValorFijoSobrante: una forma que NO es la cuarta trae `valor`.
	// Es la familia de `porque` en un atributo que si llega al motor: alguien
	// escribio dos cosas y una es vieja.
	ErrPuenteValorFijoSobrante = errors.New("valor fijo en una forma que no lo usa")
)

// algunoCasa dice si alguno de los valores del atributo esta entre las
// constantes que las reglas prueban.
//
// Basta con UNO, y no se exigen todos a proposito: un enumerado puede tener un
// valor que APAGA en vez de encender (el sector privado del ENS frente al
// publico), y ese valor legitimamente no aparece en ninguna regla.
func algunoCasa(valores []string, constantes map[string]bool) bool {
	for _, v := range valores {
		if constantes[v] {
			return true
		}
	}
	return false
}

func ordenadasCadenas(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Las tres formas del puente. Vocabulario cerrado, y la cadena vacia NO es una
// de ellas: un atributo con `hecho` presente y `forma` vacia es un olvido, no
// una respuesta.
const (
	// PuenteAfirmaSi: atributo BOOLEANO. Un «si» afirma `predicado(instancia)`.
	//
	// UN «NO» NO AFIRMA NADA, y es deliberado. En este motor la ausencia de un
	// hecho no es su negacion: hay negacion explicita (`Regla.Negados`) para
	// quien la quiera. Hacer que un «no» afirmara `no_predicado(...)` meteria en
	// el expediente una afirmacion que el operador no ha hecho.
	PuenteAfirmaSi = "afirma_si"
	// PuenteConValor: atributo con VALOR (enumerado, texto, entero, fecha). La
	// respuesta afirma `predicado(instancia, valor)`.
	PuenteConValor = "con_valor"
	// PuenteNoLlegaAlMotor: este atributo NO alimenta ninguna regla, y se dice.
	//
	// Es una respuesta valida y hay que escribirla: hoy hay 20 preguntas en el
	// corpus asi, y como nadie lo declaraba eran indistinguibles de un olvido.
	// Un atributo que no llega al motor sigue valiendo (sale en el formulario,
	// documenta el alcance, viaja al expediente); lo que no puede es hacer creer
	// que deriva algo.
	PuenteNoLlegaAlMotor = "no_llega_al_motor"
	// PuenteAfirmaSiConValor: atributo BOOLEANO cuyo «si» afirma
	// `predicado(instancia, CONSTANTE)`, con la constante escrita en el propio
	// bloque.
	//
	// # Por que hace falta una cuarta forma y no vale el rodeo
	//
	// Medido el 04-09-2026: la piden 19 hechos de 8 de los 15 marcos (los cinco
	// `adopta(E,"<marco>")`, los cuatro `designado` de la ley espanola de
	// proteccion de datos, los cuatro del reglamento financiero, los tres de la
	// directiva de ciberseguridad, los dos de su transposicion y el papel del
	// reglamento de ejecucion). Sin ella, la unica salida era declararlos como
	// enumerado de un solo valor, y el linter lo aceptaba.
	//
	// LO QUE DESCARTA EL RODEO NO ES ELEGANCIA, ES UN DANO CONCRETO: un
	// desplegable de una sola opcion que la superficie mande por defecto afirma
	// que quien contesta esta designado como entidad financiera, y eso enciende
	// una obligacion NOTIFICATORIA ante el supervisor. Ese hecho solo enciende
	// 28 obligaciones. Un umbral escrito de menos en una notificatoria no cuesta
	// horas: provoca una actuacion indebida ante el supervisor, y eso no se
	// deshace.
	//
	// # Por que ESTA forma si es segura
	//
	// Porque hereda la propiedad de PuenteAfirmaSi: UN «NO» NO AFIRMA NADA, y
	// el valor por defecto de un booleano en un formulario es «no». La
	// diferencia con el rodeo es exactamente esa: un enumerado siempre tiene un
	// valor seleccionado, un booleano sin marcar no afirma.
	PuenteAfirmaSiConValor = "afirma_si_valor"
)

var formasDelPuente = map[string]bool{
	PuenteAfirmaSi: true, PuenteConValor: true, PuenteNoLlegaAlMotor: true,
	PuenteAfirmaSiConValor: true,
}

// afirmaConValorFijo dice si la forma lleva la constante escrita en el bloque.
// Existe para que la comprobacion de «valor sobrante» no se escriba tres veces
// con el riesgo de que una se quede vieja.
func afirmaConValorFijo(forma string) bool { return forma == PuenteAfirmaSiConValor }

// HechoDeAtributo es lo que un atributo afirma en el motor de aplicabilidad.
type HechoDeAtributo struct {
	// Forma es cual de las tres. Obligatoria: la cadena vacia no es una forma.
	Forma string `json:"forma"`
	// Predicado es el nombre del hecho. Obligatorio salvo en el callejon, donde
	// tiene que estar VACIO: un callejon con predicado afirma que alguien penso
	// donde enchufarlo y no lo enchufo, que es peor que no decir nada.
	Predicado string `json:"predicado,omitempty"`
	// Porque es obligatorio en el callejon y solo ahi. Sin motivo, un atributo
	// que no llega al motor se lee como deuda y como decision a la vez.
	Porque string `json:"porque,omitempty"`
	// Valor es la CONSTANTE que afirma un `afirma_si_valor`, y solo esa forma.
	// Obligatorio ahi y prohibido en las otras tres: una forma que no lo usa y
	// lo trae escrito es alguien que cambio de idea a medias.
	Valor string `json:"valor,omitempty"`
}

// AridadEsperada dice con cuantos argumentos tiene que usarse el predicado.
// Sale de la FORMA, no de lo que el autor creyera: es lo que hace comprobable
// el emparejamiento.
func (h HechoDeAtributo) AridadEsperada() int {
	switch h.Forma {
	case PuenteAfirmaSi:
		return 1
	default:
		// `con_valor` y `afirma_si_valor` afirman los dos con dos argumentos.
		// La diferencia entre ellas no es la aridad: es de DONDE sale el
		// segundo, de la respuesta o del propio bloque.
		return 2
	}
}

// DeclaraPuente dice si este paquete ha declarado el puente en algun atributo.
// Se usa para medir cuantos paquetes lo han adoptado sin obligar a ninguno.
func (p *Paquete) DeclaraPuente() bool {
	for _, e := range p.Entidades {
		for _, a := range e.Atributos {
			if a.Hecho != nil {
				return true
			}
		}
	}
	return false
}

// aridadesDeSusReglas devuelve, para este paquete, con cuantos argumentos usa
// cada regla cada predicado. Se parsea con el parser del producto y no con una
// expresion regular: medir con un segundo parser es medir otra cosa el dia que
// los dos se separen.
func (p *Paquete) aridadesDeSusReglas() map[string]map[int]bool {
	out := map[string]map[int]bool{}
	anotar := func(a aplicabilidad.Atomo) {
		if out[a.Pred] == nil {
			out[a.Pred] = map[int]bool{}
		}
		out[a.Pred][len(a.Args)] = true
	}
	for _, rs := range p.Aplicabilidad.Reglas {
		r, err := aplicabilidad.ParsearRegla(rs.Regla)
		if err != nil {
			continue // ya lo dice el validador de reglas; aqui se ignora
		}
		anotar(r.Cabeza)
		for _, a := range r.Cuerpo {
			anotar(a)
		}
		for _, a := range r.Negados {
			anotar(a)
		}
	}
	return out
}

// valoresQueMiranSusReglas devuelve, para el SEGUNDO argumento de cada
// predicado usado con aridad 2, que constantes prueban las reglas de este
// paquete y si alguna lo usa con VARIABLE.
//
// # Por que hace falta, y no lo cubria la aridad
//
// El emparejamiento del puente se recorria en las dos direcciones por el NOMBRE
// del predicado y por la ARIDAD, y las dos son ciertas. La direccion que no se
// recorria en ninguna es la del VALOR: un atributo enumerado puede declarar el
// predicado bueno, con la aridad buena, y unos valores que ninguna regla mira.
// El hecho se afirma, no casa con nada, y no da error en ningun sitio, que es
// exactamente el estado del que el bloque `hecho` venia a sacarnos.
//
// Medido sobre el corpus el 04-09-2026: un atributo con valores
// `no_certificado|en_certificacion|certificado` declarando el predicado `adopta`
// pasaba el linter entero, y las reglas que lo leen solo prueban
// `adopta(E, "<identificador del marco>")`.
//
// # La variable manda sobre la constante, y por eso se anota aparte
//
// Si ALGUNA regla usa ese hueco con una variable, la regla acepta cualquier
// valor y no hay nada que comprobar. Es el caso de los niveles del ENS
// (`nivel_disponibilidad(X, N)`), donde ninguna regla nombra BAJO ni ALTO: los
// compara despues. Exigir solapamiento ahi seria rechazar corpus correcto, que
// es la unica forma de que un linter nuevo acabe apagado.
func (p *Paquete) valoresQueMiranSusReglas() (map[string]map[string]bool, map[string]bool) {
	consts := map[string]map[string]bool{}
	conVariable := map[string]bool{}
	anotar := func(a aplicabilidad.Atomo) {
		if len(a.Args) != 2 {
			return
		}
		t := a.Args[1]
		if t.Var {
			conVariable[a.Pred] = true
			return
		}
		if consts[a.Pred] == nil {
			consts[a.Pred] = map[string]bool{}
		}
		consts[a.Pred][t.Val] = true
	}
	for _, rs := range p.Aplicabilidad.Reglas {
		r, err := aplicabilidad.ParsearRegla(rs.Regla)
		if err != nil {
			continue // ya lo dice el validador de reglas; aqui se ignora
		}
		anotar(r.Cabeza)
		for _, a := range r.Cuerpo {
			anotar(a)
		}
		for _, a := range r.Negados {
			anotar(a)
		}
	}
	return consts, conVariable
}

// validarPuente comprueba el bloque `hecho` de cada atributo QUE LO DECLARE.
//
// Es opcional a proposito mientras dura el piloto. Lo que no es opcional es que
// si esta, sea cierto: un puente declarado y falso es peor que ninguno, porque
// afirma que la respuesta del operador llega a una regla que no la lee.
func (p *Paquete) validarPuente(anotar func(error)) {
	aridades := p.aridadesDeSusReglas()
	constantesDelPuente, valoresDelPuente := p.valoresQueMiranSusReglas()
	for _, e := range p.Entidades {
		for _, a := range e.Atributos {
			h := a.Hecho
			if h == nil {
				continue
			}
			donde := fmt.Sprintf("%s/%s.%s", p.URN, e.Nombre, a.Nombre)
			forma := strings.TrimSpace(h.Forma)
			pred := strings.TrimSpace(h.Predicado)
			porque := strings.TrimSpace(h.Porque)

			if !formasDelPuente[forma] {
				anotar(fmt.Errorf("%w: %s declara forma %q. Las cuatro son %q (booleano: un "+
					"si afirma el hecho), %q (la respuesta afirma el hecho con su valor), %q "+
					"(booleano: un si afirma el hecho con la constante que este bloque "+
					"escribe) y %q (este atributo no alimenta ninguna regla, y se dice con su "+
					"motivo)",
					ErrPuenteFormaDesconocida, donde, forma,
					PuenteAfirmaSi, PuenteConValor, PuenteAfirmaSiConValor,
					PuenteNoLlegaAlMotor))
				continue
			}

			if forma == PuenteNoLlegaAlMotor {
				if pred != "" {
					anotar(fmt.Errorf("%w: %s declara que no llega al motor Y nombra el "+
						"predicado %q. Un callejon con predicado afirma que alguien penso "+
						"donde enchufarlo y no lo enchufo: o se enchufa, o sale el nombre",
						ErrPuentePredicadoSobrante, donde, pred))
				}
				if porque == "" {
					anotar(fmt.Errorf("%w: %s dice que no llega al motor y no dice por que. "+
						"Sin motivo no se distingue de un olvido, que es justo lo que eran "+
						"las 20 preguntas huerfanas del corpus antes de este bloque",
						ErrPuenteSinPorque, donde))
				}
				if strings.TrimSpace(h.Valor) != "" {
					anotar(fmt.Errorf("%w: %s declara que no llega al motor Y trae el valor "+
						"%q. Es la misma familia que el predicado sobrante: alguien penso "+
						"que afirmaba y se quedo a medias",
						ErrPuenteValorFijoSobrante, donde, h.Valor))
				}
				continue
			}

			// A PARTIR DE AQUI EL ATRIBUTO AFIRMA ALGO, y hay que poder creerselo.
			if porque != "" {
				anotar(fmt.Errorf("%w: %s afirma un hecho Y trae el `porque` del callejon. "+
					"Una de las dos es vieja", ErrPuentePredicadoSobrante, donde))
			}
			if pred == "" {
				anotar(fmt.Errorf("%w: %s dice afirmar un hecho y no dice cual",
					ErrPuenteSinPredicado, donde))
				continue
			}
			// LA FORMA TIENE QUE CASAR CON EL TIPO DEL ATRIBUTO. Un booleano no
			// tiene valor que poner de segundo argumento, y un enumerado con
			// tres valores no se puede afirmar sin decir cual: las dos
			// confusiones producen un hecho que el motor nunca casa.
			esBooleano := a.Tipo == Booleano

			// EL VALOR FIJO: obligatorio en su forma, prohibido en las otras.
			//
			// Prohibido y no ignorado, que es la diferencia que importa: un
			// `con_valor` que trae un valor escrito parece decir «y ademas
			// siempre este», y no lo dice. Se lee mal en la unica direccion que
			// hace dano, la de creer que afirma mas de lo que afirma.
			valorFijo := strings.TrimSpace(h.Valor)
			if afirmaConValorFijo(forma) {
				if valorFijo == "" {
					anotar(fmt.Errorf("%w: %s declara %q y no dice QUE constante afirma. El "+
						"hecho saldria con el segundo argumento vacio y no casaria con "+
						"ninguna regla, sin dar error en ningun sitio",
						ErrPuenteSinValorFijo, donde, forma))
					continue
				}
			} else if valorFijo != "" {
				anotar(fmt.Errorf("%w: %s declara la forma %q y trae el valor %q, que solo "+
					"usa %q. En %q el segundo argumento sale de la RESPUESTA, asi que este "+
					"escrito no se usa y hace creer que si",
					ErrPuenteValorFijoSobrante, donde, forma, valorFijo,
					PuenteAfirmaSiConValor, PuenteConValor))
			}

			if forma == PuenteAfirmaSi && !esBooleano {
				anotar(fmt.Errorf("%w: %s es de tipo %q y declara %q. Solo un booleano se "+
					"afirma sin valor; este atributo tiene valores y afirmarlo sin ellos "+
					"produce un hecho que ninguna regla casa",
					ErrPuenteFormaNoCasaConTipo, donde, a.Tipo, PuenteAfirmaSi))
				continue
			}
			if forma == PuenteConValor && esBooleano {
				anotar(fmt.Errorf("%w: %s es booleano y declara %q. Un booleano no tiene "+
					"valor que poner de segundo argumento; si lo que quieres es que su «si» "+
					"afirme una constante, la forma es %q, que la escribe aqui",
					ErrPuenteFormaNoCasaConTipo, donde, PuenteConValor, PuenteAfirmaSiConValor))
				continue
			}
			// Y LA CUARTA EXIGE BOOLEANO POR LA MISMA RAZON QUE LA PRIMERA, que
			// es su razon de ser: lo que la hace segura frente al rodeo del
			// enumerado de un solo valor es que un booleano SIN MARCAR no
			// afirma, y un desplegable siempre trae algo seleccionado.
			if afirmaConValorFijo(forma) && !esBooleano {
				anotar(fmt.Errorf("%w: %s es de tipo %q y declara %q, que es para booleanos. "+
					"Un atributo con valores propios afirma el suyo con %q; lo que esta forma "+
					"aporta es que un «no» no afirme nada, y eso solo lo da un booleano",
					ErrPuenteFormaNoCasaConTipo, donde, a.Tipo, forma, PuenteConValor))
				continue
			}

			// EL EMPAREJAMIENTO, EN LAS DOS DIRECCIONES. Casa por el nombre del
			// predicado, y las dos puntas estan en este mismo paquete firmado.
			usos, hay := aridades[pred]
			if !hay {
				anotar(fmt.Errorf("%w: %s afirma %q y ninguna regla de este paquete usa ese "+
					"predicado. La respuesta del operador se recogeria, se pintaria y no "+
					"derivaria nada, que es el estado en el que estaban 20 preguntas del "+
					"corpus. Arreglo: o se escribe la regla que lo lee, o el atributo declara "+
					"%q con su motivo. Predicados que si usa: %v",
					ErrPuentePredicadoHuerfano, donde, pred, PuenteNoLlegaAlMotor,
					clavesDeAridades(aridades)))
				continue
			}
			if quiere := h.AridadEsperada(); !usos[quiere] {
				anotar(fmt.Errorf("%w: %s afirma %q con %d argumento(s) por su forma %q, y las "+
					"reglas lo usan con %v. Un hecho con la aridad cambiada no casa con "+
					"ninguna regla y no da error en ningun sitio",
					ErrPuenteAridadDistinta, donde, pred, quiere, forma, ordenar(usos)))
				continue
			}

			// LA TERCERA DIRECCION DEL EMPAREJAMIENTO: EL VALOR.
			//
			// El nombre del predicado casa y la aridad casa, y aun asi el hecho
			// puede no casar con ninguna regla, porque lo que las reglas prueban
			// son CONSTANTES en ese hueco. Sin esto, un enumerado con los
			// valores de otra cosa pasa el linter entero.
			// Y PARA LA CUARTA FORMA LA CONSTANTE ES UNA SOLA, asi que la
			// comprobacion es mas dura: no basta con que ALGUNA case, tiene que
			// casar ESA. Si no casa, el «si» del operador produce un hecho que
			// ninguna regla mira, que es el agujero entero de este bloque.
			if afirmaConValorFijo(forma) && !valoresDelPuente[pred] &&
				len(constantesDelPuente[pred]) > 0 && !constantesDelPuente[pred][valorFijo] {
				anotar(fmt.Errorf("%w: %s afirma %q con la constante %q, y ninguna regla de "+
					"este paquete prueba ese valor en ese hueco: solo miran %v. El nombre "+
					"casa y la aridad casa, asi que el hecho se afirma, no casa con ninguna "+
					"regla y no da error en ningun sitio",
					ErrPuenteValorHuerfano, donde, pred, valorFijo,
					ordenadasCadenas(constantesDelPuente[pred])))
			}

			if forma == PuenteConValor && a.Tipo == Enumerado && len(a.Valores) > 0 {
				if !valoresDelPuente[pred] && len(constantesDelPuente[pred]) > 0 {
					if !algunoCasa(a.Valores, constantesDelPuente[pred]) {
						anotar(fmt.Errorf("%w: %s afirma %q con sus valores %v, y ninguna regla "+
							"de este paquete prueba ninguno de ellos en ese hueco: las reglas "+
							"solo miran %v. El nombre del predicado casa y la aridad casa, asi "+
							"que el hecho se afirma, no casa con ninguna regla y no da error en "+
							"ningun sitio, que es justo lo que este bloque existe para cerrar. "+
							"Arreglo: o los valores del atributo son los que la regla prueba, o "+
							"la regla prueba los del atributo",
							ErrPuenteValorHuerfano, donde, pred, a.Valores,
							ordenadasCadenas(constantesDelPuente[pred])))
					}
				}
			}
		}
	}
}

func clavesDeAridades(m map[string]map[int]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func ordenar(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

// RespuestaDeEntrevista es lo que el operador contesta sobre UNA instancia.
//
// La instancia importa y por eso viaja: el ENS no pregunta por «el sistema» en
// abstracto, pregunta por CADA informacion y CADA servicio que maneja, y los
// hechos que salen son `nivel_confidencialidad(esa_informacion, ALTO)`. Sin
// instancia, todas las respuestas caerian sobre el mismo sujeto y tres
// informaciones distintas se pisarian entre si.
type RespuestaDeEntrevista struct {
	Entidad   string
	Instancia string
	Atributo  string
	// Si es la respuesta de un booleano. Un `false` NO afirma nada: ver
	// PuenteAfirmaSi.
	Si bool
	// Valor es la respuesta de un atributo con valor.
	Valor string
}

// HechosDeLaEntrevista traduce respuestas a hechos USANDO LA DECLARACION DEL
// PAQUETE, que es lo unico que puede saber que hecho produce cada respuesta.
//
// VIVE EN EL PRODUCTO Y NO EN UN TEST a proposito. Nacio dentro del test del
// piloto, para medir antes de construir, y ahi se quedaba a un paso de ser la
// pieza que le falta a la pantalla: cuando la entrevista aprenda a preguntar
// valores, lo que hara es llamar aqui. Una funcion medida en un test y despues
// reescrita en el producto son dos implementaciones que se separan, y la que se
// despliega es la que nadie midio.
//
// LOS ERRORES SON ERRORES Y NO SE DEGRADAN A SILENCIO (invariante 8, tercera
// forma): una respuesta a un atributo que el paquete no declara, o sin el valor
// que su forma exige, es un dato presente y no interpretable. Devolverlo como
// «ningun hecho» haria que la derivacion saliera corta sin que nadie lo supiera,
// que es la forma mas cara de fallar aqui: obligaciones que no aparecen.
func HechosDeLaEntrevista(p *Paquete, rs []RespuestaDeEntrevista) ([]aplicabilidad.Hecho, error) {
	type clave struct{ entidad, atributo string }
	decl := map[clave]HechoDeAtributo{}
	for _, e := range p.Entidades {
		for _, a := range e.Atributos {
			if a.Hecho != nil {
				decl[clave{e.Nombre, a.Nombre}] = *a.Hecho
			}
		}
	}
	var out []aplicabilidad.Hecho
	for i, r := range rs {
		h, hay := decl[clave{r.Entidad, r.Atributo}]
		if !hay {
			return nil, fmt.Errorf("%s: la respuesta %d es sobre %s.%s y el paquete no declara "+
				"el puente de ese atributo. No se descarta en silencio: una respuesta que no "+
				"llega al motor sin decirlo hace que falten obligaciones y nadie lo sepa",
				p.URN, i, r.Entidad, r.Atributo)
		}
		if strings.TrimSpace(r.Instancia) == "" {
			return nil, fmt.Errorf("%s: la respuesta %d sobre %s.%s no dice de que instancia "+
				"habla. Sin instancia, dos informaciones distintas se pisarian",
				p.URN, i, r.Entidad, r.Atributo)
		}
		switch h.Forma {
		case PuenteNoLlegaAlMotor:
			// No es error: el paquete ya declaro que esta respuesta no alimenta
			// ninguna regla, y eso esta escrito con su motivo. Se recoge y no
			// afirma nada, que es exactamente lo prometido.
			continue
		case PuenteAfirmaSi:
			if r.Si {
				out = append(out, aplicabilidad.H(h.Predicado, r.Instancia))
			}
		case PuenteAfirmaSiConValor:
			// El «si» afirma la constante DEL PAQUETE, no la respuesta: aqui el
			// operador dice si o no, y que constante se afirma lo decidio quien
			// escribio la norma. Un «no» no afirma nada, igual que en
			// PuenteAfirmaSi, y esa es la propiedad entera de esta forma.
			if r.Si {
				out = append(out, aplicabilidad.H(h.Predicado, r.Instancia, h.Valor))
			}
		case PuenteConValor:
			if strings.TrimSpace(r.Valor) == "" {
				return nil, fmt.Errorf("%s: la respuesta %d sobre %s.%s lleva valor por su "+
					"forma %q y llega vacia", p.URN, i, r.Entidad, r.Atributo, h.Forma)
			}
			out = append(out, aplicabilidad.H(h.Predicado, r.Instancia, r.Valor))
		default:
			return nil, fmt.Errorf("%s: %s.%s declara la forma %q, que no existe",
				p.URN, r.Entidad, r.Atributo, h.Forma)
		}
	}
	return out, nil
}
