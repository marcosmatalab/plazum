package escalado

// LA PUERTA D11-c EN EL ESCALADO: ninguna cifra de esta pantalla se queda sin
// abrir.
//
// # El agujero que cierra, con su cardinal
//
// El 04-09-2026 esta pantalla aprendio a DECIR sus ocho cubos (la familia
// `escalado.cubo.*`), y el informe de aquel dia lo dejo escrito con todas las
// letras: «traducir el rotulo NO es darle su derivacion». `estado: N` se leia y
// seguia sin enlace a los escalones que lo componen, con los `Trabajos` y sus
// `Pasos` pintados justo encima. Cardinal de entonces: OCHO cifras sin
// derivacion, mas `planificados`. Hoy son CERO, y de dos maneras distintas,
// que son las mismas dos que ya usa el calendario:
//
//	CON LISTA (los cubos)     cada uno enlaza a /cubo/<estado>, que trae los
//	                          escalones de ese estado y solo esos. Se cuentan a
//	                          mano y tienen que salir N.
//	CON PARTICION (el total)  `planificados` es la SUMA de los cubos, escrita al
//	                          lado. No hay que creerse el numero: hay que sumar
//	                          unos numeros que ya estan en la pagina, y cada uno
//	                          de ellos se abre por su cuenta.
//
// La particion NO es circular, que es la trampa que documenta
// superficies/calendario/cuenta.go: los sumandos se sostienen SOLOS (cada cubo
// tiene su lista), asi que la ecuacion abre el total y no al reves.
//
// # POR QUE CAMPO CASA UN ESCALON CON SU CUBO, dicho en voz alta
//
// Por `nucleo/escalado.Estado`, LETRA POR LETRA, que es la constante del
// vocabulario cerrado del motor. Es la misma cadena que produce el recuento
// (`Plan.Cuenta` va indexada por ella), la misma que lleva dentro cada
// `nucleo/escalado.Paso`, y la misma que viaja en la direccion. NUNCA por la
// posicion en la lista de cubos ni por el orden de `EstadosPosibles()`:
// insertar un estado noveno moveria el emparejamiento entero sin que nada se
// rompiera (invariante 7). Aqui no hay nada firmado que proteger, pero la
// familia del fallo es la misma y se cierra igual.
//
// # Y EL RECUENTO SE CONTRASTA CONTRA LA LISTA, porque son dos origenes
//
// `Plan.Cuenta` y `Plan.Trabajos` llegan por el MISMO interfaz y NO tienen por
// que cuadrar: `Fuente` es un puerto, y quien lo implemente puede darlos
// separados. Una cabecera que dice 3 sobre una lista de 2 es peor que una cifra
// sin enlace, porque el enlace prometia que se podia comprobar. Es el P0 que se
// cobro el calendario el 04-09-2026, en otra pantalla y con la misma forma.
//
// El adaptador que monta hoy (`cmd/plazum`) los deriva de los mismos pasos, asi
// que cuadran por construccion y esta rama NO la recorre el dato real: por eso
// tiene control positivo con dato sintetico, que es lo unico que la hace
// existir (M47).

import (
	"net/url"
	"strings"

	nescalado "github.com/marcosmatalab/plazum/nucleo/escalado"
)

// SegmentoDelCubo es el tramo de ruta bajo el que se abre una cifra de la
// cuenta. Se escribe UNA vez: el patron que se registra y el enlace que se
// pinta salen los dos de aqui, porque dos copias de una direccion se separan y
// el sintoma es una cifra que enlaza a un 404, o sea la cifra huerfana con una
// capa de pintura encima.
const SegmentoDelCubo = "/cubo/"

// SegmentoDeMandar es la ruta de la pagina que dice como se disparan los
// avisos de verdad.
//
// Vive al lado del anterior por lo mismo: es un contrato entre el enrutador,
// la plantilla que enlaza y el test, y escrito tres veces se corrige una.
const SegmentoDeMandar = "/mandar"

// SinDerivacionEsperadas es cuantas cifras de esta pantalla NO se pueden abrir.
//
// EL CARDINAL SE ESCRIBE PARA QUE MOLESTE, y se compara con IGUALDAD EXACTA:
// tiene que ponerse rojo cuando sube (alguien pinta un numero nuevo y no lo
// abre) y tambien cuando baja (alguien cierra el hueco y deja el numero viejo
// puesto, que miente hacia arriba para siempre).
const SinDerivacionEsperadas = 0

// EnlaceDelCubo compone la direccion en la que se abre un cubo.
//
// EL ESTADO VIAJA ESCAPADO COMO SEGMENTO DE RUTA porque cinco de los ocho
// llevan espacios («sin destinatario», «colapsado en un escalon anterior»). Se
// usa url.PathEscape y no QueryEscape: aquel deja el espacio como %20, que es
// lo que un segmento admite, y este lo deja como «+», que en una ruta es un
// signo mas literal y no casaria con la constante del nucleo.
func EnlaceDelCubo(base, estado string) string {
	return base + SegmentoDelCubo + url.PathEscape(estado)
}

// EscalonVista es un escalon del plan con el trabajo del que cuelga, listo para
// pintar dentro de la derivacion de un cubo.
//
// LLEVA SU TRABAJO DENTRO y no se agrupa por trabajo a proposito: quien abre
// «sin destinatario: 4» quiere contar cuatro filas, y una lista agrupada obliga
// a sumar cabeceras para llegar al mismo numero. Es el mismo motivo por el que
// el calendario cuenta filas y no obligaciones.
type EscalonVista struct {
	// Obligacion, Titulo e Hito vienen del corpus y viajan TAL CUAL.
	Obligacion string
	Titulo     string
	Hito       string
	Vence      string
	Nivel      int
	Cuando     string
	Figura     string
	Persona    string
	Motivo     string
	// Saldria dice que este escalon lleva aviso. Ninguna peticion a esta
	// pantalla manda nada; ver el encabezado del paquete.
	Saldria bool
}

// DerivacionCubo es UNA cifra de la cuenta, abierta entera.
type DerivacionCubo struct {
	// Estado es la palabra del nucleo. Es la identidad y es el respaldo del
	// rotulo cuando no hay clave.
	Estado string
	// Clave es la clave de catalogo del rotulo. Vacia: se pinta Estado.
	Clave string
	// N es lo que decia la cabecera de la que se ha venido.
	N int
	// Escalones es lo que compone esa cifra. Su longitud TIENE que ser N.
	Escalones []EscalonVista
	// Descuadre dice que no lo es. Se pinta: una lista mas corta que su
	// cabecera, callada, es la promesa rota que el enlace venia a evitar.
	Descuadre bool
	// Volver es la direccion de la pantalla de la que sale esta cifra.
	Volver string
}

// ParteDelPlan es un sumando de `planificados`.
//
// Lleva el ESTADO ademas del numero porque el estado es la identidad por la que
// la puerta cruza el sumando con el cubo que lo produce (invariante 7): por
// posicion, reordenar la cuenta cambiaria de que se dice compuesto el total sin
// que nada se rompiera.
type ParteDelPlan struct {
	Estado string
	Clave  string
	N      int
}

// escalonesDelEstado saca de los trabajos los escalones de UN estado.
//
// Casa por `nucleo/escalado.Estado` letra por letra. El orden es el de los
// trabajos y, dentro de cada uno, el de los escalones: es el mismo orden en el
// que la pagina principal los pinta, y asi quien baja de una a otra los
// reconoce en vez de tener que buscarlos.
func escalonesDelEstado(p Plan, estado string) []EscalonVista {
	var out []EscalonVista
	for _, t := range p.Trabajos {
		for _, paso := range t.Pasos {
			if string(paso.Estado) != estado {
				continue
			}
			ev := EscalonVista{
				Obligacion: t.Obligacion, Titulo: t.Titulo, Hito: t.Hito,
				Vence:   t.Vence.Format(formatoDeDia),
				Nivel:   paso.Nivel,
				Figura:  paso.Figura,
				Persona: paso.Persona,
				Motivo:  paso.Motivo,
				Saldria: paso.Aviso != nil,
			}
			if !paso.Cuando.IsZero() {
				ev.Cuando = paso.Cuando.Format(formatoDeDia)
			}
			out = append(out, ev)
		}
	}
	return out
}

// EstadosQueSePintan son los cubos que la pagina principal ensena, en su orden.
//
// SALE DE LA MISMA FUNCION QUE PINTA LA CUENTA, no de una segunda copia: lo que
// se puede abrir es exactamente lo que se pinta, y las dos listas se separarian
// el dia que una cambiara. Un cero no se pinta, asi que tampoco se abre: no hay
// cifra que abrir.
func EstadosQueSePintan(p Plan) []string {
	var v Vista
	v.rellenarCon(p, "")
	out := make([]string, 0, len(v.Cuenta))
	for _, c := range v.Cuenta {
		out = append(out, c.Estado)
	}
	return out
}

// derivar abre un cubo. Devuelve (derivacion, hay).
//
// LAS TRES FORMAS DE LA ENTRADA, y no son dos (invariante 8, tercera forma):
//
//	segmento vacio          no se ha pedido ninguna cifra. 404.
//	estado que SI se pinta  la lista entera de lo que lo compone.
//	estado que no se pinta  ES UN DATO QUE HAY Y NO SE ENTIENDE: un estado
//	                        desconocido, o uno cuyo cubo vale cero y por tanto
//	                        no es ninguna cifra de esta pagina. 404, NUNCA una
//	                        lista vacia: una derivacion vacia se leeria como
//	                        «este numero es cero», que es una afirmacion sobre
//	                        el plan y no sobre la peticion.
func derivar(p Plan, estado, base string) (DerivacionCubo, bool) {
	if strings.TrimSpace(estado) == "" {
		return DerivacionCubo{}, false
	}
	n, sePinta := 0, false
	for _, e := range EstadosQueSePintan(p) {
		if e == estado {
			sePinta = true
			break
		}
	}
	if !sePinta {
		return DerivacionCubo{}, false
	}
	n = p.Cuenta[nescalado.Estado(estado)]
	d := DerivacionCubo{
		Estado:    estado,
		N:         n,
		Escalones: escalonesDelEstado(p, estado),
		Volver:    base + "/",
	}
	d.Clave, _ = ClaveDelCubo(nescalado.Estado(estado))
	// EL CONTRASTE, sobre lo que el lector puede contar. Ver el encabezado: los
	// dos lados llegan por el mismo puerto y no tienen por que cuadrar.
	d.Descuadre = len(d.Escalones) != n
	return d, true
}

// particionDelPlan son los sumandos de `planificados`, en el orden en que la
// pagina pinta los cubos.
//
// SALE DE LA CUENTA YA PINTADA y no de un recorrido nuevo del plan: si saliera
// de otro sitio, la frase «se compone de» podria cuadrar con un conjunto de
// numeros distinto del que el lector tiene delante, que es la unica forma de
// que una particion mienta sin que sus cuentas fallen.
func particionDelPlan(cuenta []CuboVista) []ParteDelPlan {
	out := make([]ParteDelPlan, 0, len(cuenta))
	for _, c := range cuenta {
		out = append(out, ParteDelPlan{Estado: c.Estado, Clave: c.Clave, N: c.N})
	}
	return out
}
