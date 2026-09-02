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
)

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
)

var formasDelPuente = map[string]bool{
	PuenteAfirmaSi: true, PuenteConValor: true, PuenteNoLlegaAlMotor: true,
}

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
}

// AridadEsperada dice con cuantos argumentos tiene que usarse el predicado.
// Sale de la FORMA, no de lo que el autor creyera: es lo que hace comprobable
// el emparejamiento.
func (h HechoDeAtributo) AridadEsperada() int {
	if h.Forma == PuenteAfirmaSi {
		return 1
	}
	return 2
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

// validarPuente comprueba el bloque `hecho` de cada atributo QUE LO DECLARE.
//
// Es opcional a proposito mientras dura el piloto. Lo que no es opcional es que
// si esta, sea cierto: un puente declarado y falso es peor que ninguno, porque
// afirma que la respuesta del operador llega a una regla que no la lee.
func (p *Paquete) validarPuente(anotar func(error)) {
	aridades := p.aridadesDeSusReglas()
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
				anotar(fmt.Errorf("%w: %s declara forma %q. Las tres son %q (booleano: un si "+
					"afirma el hecho), %q (la respuesta afirma el hecho con su valor) y %q "+
					"(este atributo no alimenta ninguna regla, y se dice con su motivo)",
					ErrPuenteFormaDesconocida, donde, forma,
					PuenteAfirmaSi, PuenteConValor, PuenteNoLlegaAlMotor))
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
			if forma == PuenteAfirmaSi && !esBooleano {
				anotar(fmt.Errorf("%w: %s es de tipo %q y declara %q. Solo un booleano se "+
					"afirma sin valor; este atributo tiene valores y afirmarlo sin ellos "+
					"produce un hecho que ninguna regla casa",
					ErrPuenteFormaNoCasaConTipo, donde, a.Tipo, PuenteAfirmaSi))
				continue
			}
			if forma == PuenteConValor && esBooleano {
				anotar(fmt.Errorf("%w: %s es booleano y declara %q. Un booleano no tiene "+
					"valor que poner de segundo argumento",
					ErrPuenteFormaNoCasaConTipo, donde, PuenteConValor))
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
