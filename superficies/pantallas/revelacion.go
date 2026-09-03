package pantallas

import (
	"net/url"

	"github.com/marcosmatalab/plazum/nucleo/pantalla"
)

// LA REVELACION PROGRESIVA DE LA ENTREVISTA.
//
// # El problema, medido antes de escribir esto
//
// El TTFV del camino guiado costaba 18m56s sobre un presupuesto de 15m0s, y el
// cuello de botella tenia nombre: la entrevista de /alcance pintaba las 42
// preguntas del corpus instalado, TODAS, siempre. A veinte segundos por
// pregunta eso son catorce minutos y medio, el 76 % del total. Un CISO que
// contesta cuatro preguntas y ve su primer calendario vuelve; uno que ve
// cuarenta y dos se va.
//
// # La regla, y por que es la unica que se puede defender
//
// Una pregunta se ensena mientras pueda CAMBIAR ALGO. Se deja fuera cuando se
// puede demostrar que no, y solo hay dos formas de demostrarlo:
//
//	NadieLaPide   ninguna obligacion del corpus instalado la nombra en su
//	              `preguntas`, o sea que su respuesta no entra en el veredicto
//	              de nada. No es una decision de producto: es un defecto del
//	              corpus que hasta ahora no se veia, y la pantalla lo dice.
//	YaDecidida    todas las obligaciones que la nombran estan ya decididas
//	              (aplican o no aplican) por respuestas anteriores, asi que
//	              contestarla no mueve ninguna cesta.
//
// # POR QUE ESCONDER ASI NO PUEDE ESCONDER UNA PREGUNTA QUE HACIA FALTA
//
// Es la pregunta que hay que contestar antes de enviar esto a nadie, porque
// esconder una pregunta que abria una obligacion produce un calendario
// incompleto, que es el mismo silencio que se persigue en el resto del
// producto. La demostracion es corta y se apoya en una sola cosa:
// `evaluarControl` lee la respuesta de una pregunta EXCLUSIVAMENTE a traves de
// `Fila.Requiere`. Con eso:
//
//	NadieLaPide  la pregunta no esta en ningun `Requiere`, asi que ninguna
//	             evaluacion la consulta: responderla no cambia nada, ni ahora
//	             ni despues.
//	YaDecidida   las obligaciones que la nombran estan en Aplica o en NoAplica.
//	             Una en Aplica tiene TODAS sus preguntas respondidas que si, y
//	             por tanto esta tambien lo esta (y una respondida nunca se
//	             esconde, ver abajo). Una en NoAplica lo esta porque otra
//	             pregunta suya vale «no», y la rama de negativas gana sobre
//	             todas las demas en `evaluarControl`: ningun «si» posterior la
//	             devuelve a Aplica.
//
// La demostracion no se deja en este comentario: hay una puerta que la ejecuta
// sobre el corpus REAL y sobre corpus sinteticos, probando cada pregunta
// escondida con las dos respuestas posibles y exigiendo que el veredicto de las
// 528 obligaciones no se mueva ni una casilla.
//
// # Y LO QUE SE ESCONDE SE PUEDE VER, CON SU CARDINAL
//
// Una entrevista que oculta preguntas sin decir que existen es la otra forma
// del silencio. La pantalla dice cuantas ha dejado fuera y enlaza a la lista
// entera, donde cada una lleva escrito POR QUE no decide nada. Todo con
// navegacion del servidor: un parametro en la direccion, ningun JavaScript,
// ningun `display:none`.

// Vivacidad dice si una pregunta puede todavia cambiar el veredicto de alguna
// obligacion con las respuestas que hay.
type Vivacidad uint8

const (
	// Viva es EL VALOR CERO Y ES EL RESTRICTIVO, a proposito: una pregunta se
	// ensena mientras no se DEMUESTRE que no mueve nada. Si esta clasificacion
	// se rompiera y devolviera el cero para todo, la entrevista volveria a ser
	// la de antes (larga y completa), que es el fallo caro pero no el fallo
	// peligroso. Al reves seria una entrevista que esconde sin motivo.
	Viva Vivacidad = iota
	// NadieLaPide: ninguna obligacion del corpus instalado la nombra en su
	// `preguntas`. Su respuesta no entra en el veredicto de nada.
	NadieLaPide
	// YaDecidida: las obligaciones que la nombran ya estan decididas por
	// respuestas anteriores.
	YaDecidida
)

// Clave es la clave de catalogo que explica por que una pregunta esta dormida.
// La Viva no tiene: no hay nada que explicar.
func (v Vivacidad) Clave() string {
	switch v {
	case NadieLaPide:
		return "alcance.dormidas.nadie_la_pide"
	case YaDecidida:
		return "alcance.dormidas.ya_decidida"
	}
	return ""
}

// Dormida dice si esta vivacidad deja la pregunta fuera de la lista corta.
func (v Vivacidad) Dormida() bool { return v != Viva }

// vivacidades clasifica las preguntas de la entrevista contra los veredictos
// que hay con las respuestas dadas.
//
// LA DIRECCION QUE SE RECORRE ES `Fila.Requiere`, Y NO `Pregunta.Desbloquea`,
// y no es un detalle de implementacion: son dos listas distintas del corpus que
// nadie cruza, y en el corpus instalado DISCREPAN en 23 de las 42 preguntas
// (invariante 7: emparejar por lo que de verdad decide, no por lo que declara
// la otra punta). `Desbloquea` es lo que la pregunta DICE que abre y solo se
// usa para ordenar; `Requiere` es lo que la evaluacion LEE de verdad. Clasificar
// por `Desbloquea` daria 42 preguntas vivas y ninguna dormida, o sea la
// entrevista de antes con una capa de pintura.
func vivacidades(qs []pantalla.Pregunta, vs []Veredicto, r Respuestas) map[string]Vivacidad {
	pide := make(map[string][]int, len(qs))
	for i, v := range vs {
		// Requiere en nil y Requiere vacio-presente son la misma cosa AQUI, y
		// las dos son correctas: una obligacion sin preguntas no condiciona a
		// ninguna. El bucle no se ejecuta en ninguno de los dos casos.
		for _, id := range v.Fila.Requiere {
			pide[id] = append(pide[id], i)
		}
	}
	out := make(map[string]Vivacidad, len(qs))
	for _, q := range qs {
		// UNA PREGUNTA YA RESPONDIDA NO SE ESCONDE NUNCA, pase lo que pase con
		// el resto de la clasificacion. Dos razones y las dos bastan solas:
		// esconder lo que alguien acaba de contestar borra su trabajo de la
		// pantalla, y sin la pregunta a la vista no hay forma de deshacer la
		// respuesta sin editar la direccion a mano. La contradictoria entra
		// aqui tambien: su aviso tiene que seguir viendose.
		if r.Dice(q.ID) != SinResponder {
			out[q.ID] = Viva
			continue
		}
		quienes := pide[q.ID]
		if len(quienes) == 0 {
			out[q.ID] = NadieLaPide
			continue
		}
		out[q.ID] = YaDecidida
		for _, i := range quienes {
			if vs[i].Estado == Pendiente {
				out[q.ID] = Viva
				break
			}
		}
	}
	return out
}

// --- que parte de la entrevista se pide, y como se lee de la direccion ---

// ParamVer es el parametro que abre la entrevista entera. Viaja en la
// direccion como todo lo demas de esta superficie: la pagina larga se comparte
// y se marca igual que la corta.
const ParamVer = "ver"

// VerTodas es el UNICO valor admitido de ParamVer.
const VerTodas = "todas"

// Modo es cuanta entrevista se pinta.
type Modo uint8

const (
	// ModoVivas es EL VALOR CERO: solo las preguntas que todavia deciden algo.
	ModoVivas Modo = iota
	// ModoTodas: la entrevista entera, con las dormidas marcadas y con el
	// motivo de cada una.
	ModoTodas
)

// conTodas devuelve la misma consulta con el modo largo puesto. No muta la que
// recibe menos de lo imprescindible: `Consulta()` ya devuelve una copia nueva
// en cada llamada.
func conTodas(q url.Values) url.Values {
	q.Set(ParamVer, VerTodas)
	return q
}

// modoPedido lee el parametro OPCIONAL `ver` de la consulta.
//
// TRES CASOS Y NO DOS, que es el invariante 8 en su tercera forma. Ausente y
// presente-pero-no-interpretable NO son la misma cosa y no se pueden leer con
// la misma linea:
//
//	ausente                    la lista corta. Es el valor por defecto y es
//	                           una respuesta legitima: no se pidio nada.
//	presente y conocido        lo que pida.
//	presente y no interpretable   ERROR, nunca el valor por defecto. «?ver=»,
//	                           «?ver=basura» y «?ver=todas&ver=todas» son un
//	                           dato que HAY y que no se entiende, y tomarlo por
//	                           la nada es inventarse un valor.
//
// Devuelve (modo, interpretable). Quien llama contesta 404 cuando no lo es: la
// direccion pedida no es ninguna pagina de esta superficie, y decirlo es mas
// honrado que servir una pagina distinta de la que se pidio.
func modoPedido(v url.Values) (Modo, bool) {
	valores, presente := v[ParamVer]
	if !presente {
		return ModoVivas, true
	}
	if len(valores) != 1 {
		// Repetido: dos valores no se pueden reducir a uno sin elegir por el
		// que pregunta.
		return ModoVivas, false
	}
	if valores[0] == VerTodas {
		return ModoTodas, true
	}
	return ModoVivas, false
}
