package main

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/marcosmatalab/plazum/adaptadores/instalacion"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/superficies/pantallas"
)

// LA CONSECUENCIA DE CONTESTAR QUE SI, CALCULADA POR EL MOTOR (pieza 2).
//
// # Que hace, en una linea
//
// Deriva DOS VECES: con las respuestas que hay, y con las respuestas que hay mas
// esta pregunta contestada que si. La diferencia entre los dos conjuntos de
// obligaciones aplicables es la consecuencia.
//
// # POR QUE ESTO NO USA IA, ESTANDO EN EL BLOQUE DE IA
//
// Porque la pregunta que responde es una pregunta de REGLAS, y las reglas las
// tiene el producto: «¿que se te activa si dices que si?» la contesta el mismo
// motor de aplicabilidad que decide de verdad, y por tanto la respuesta es
// exacta y recontable. Pedirsela a un modelo seria cambiar una respuesta
// verificable por una plausible en la unica pantalla donde alguien va a cambiar
// lo que declara sobre su organizacion.
//
// La IA hace falta en la mitad de al lado: leer el PDF que sube el cliente
// (piezas 1, 3 y 7). Ahi hay texto libre y no hay reglas.
//
// # LA MISMA CONVERSION QUE TODO LO DEMAS, Y ESO ES LA MITAD DEL DISENO
//
// Las respuestas se convierten en hechos con `exportarAlcance`, que es
// exactamente la funcion que usa `plazum alcance` y la que publica el alcance de
// la instalacion. Una segunda conversion aqui habria dado dos motores para la
// misma cuenta, y el dia que discreparan, la consecuencia que se le ensena a
// alguien ANTES de contestar no seria la que ocurre DESPUES de contestar. Ese
// fallo no da error en ningun sitio: da una pantalla que miente y un cliente que
// deja de creerse el producto entero.
type consecuenciasDeLaEntrevista struct {
	paquetes []*corpus.Paquete
	quienEs  func() instalacion.Identidad

	// mu protege la memoria de abajo. Esta superficie se sirve concurrentemente.
	mu sync.Mutex
	// deLaBase es la derivacion de las respuestas TAL COMO ESTAN, memorizada
	// para una sola pagina.
	//
	// # Por que hace falta memorizarla, con el numero delante
	//
	// La pantalla pinta hasta 19 preguntas y cada una necesita la base para
	// restar. Sin memoria cada pregunta deriva DOS veces (la base y la
	// hipotesis): 19 x 2 = 38 derivaciones por peticion. Con ella, 1 + 19 = 20.
	// Es casi la mitad del coste de la pagina, y el cardinal esta escrito
	// porque es lo que hace falta para decidir si vale la pena.
	//
	// LA CLAVE ES LA CONSULTA ENTERA, no un contador de peticiones: dos
	// pantallas distintas del mismo servidor tienen respuestas distintas, y una
	// memoria indexada por cualquier otra cosa le daria a una la base de la
	// otra. Eso no seria lento, seria falso.
	claveBase string
	base      map[string]bool
	titulos   map[string]tituloDeObligacion
}

// tituloDeObligacion es lo que se ensena de una obligacion que se activa.
type tituloDeObligacion struct {
	Titulo string
	Marco  string
}

// PresupuestoDeLaConsecuencia es lo que puede tardar el calculo de UNA pregunta.
//
// SE DECLARA PORQUE ES UN COSTE POR PREGUNTA Y LA PANTALLA PINTA DIECINUEVE.
// Un calculo de 50 ms por pregunta son casi mil milisegundos de pagina, y
// entonces la pieza que existe para que la entrevista se termine seria la razon
// de que se abandone. Lo vigila su test con el corpus real.
const PresupuestoDeLaConsecuencia = 25 * time.Millisecond

var _ pantallas.Consecuencias = (*consecuenciasDeLaEntrevista)(nil)

// De calcula la consecuencia de contestar que si a una pregunta.
func (c *consecuenciasDeLaEntrevista) De(_ context.Context, respuestas url.Values,
	pregunta string) (pantallas.Consecuencia, error) {

	id := strings.TrimSpace(pregunta)
	if id == "" {
		return pantallas.Consecuencia{}, fmt.Errorf("consecuencia sin pregunta")
	}
	base, titulos, err := c.baseDe(respuestas)
	if err != nil {
		return pantallas.Consecuencia{}, err
	}
	// LA RESPUESTA HIPOTETICA SE ANADE SOBRE UNA COPIA. Mutar el url.Values que
	// llega dejaria la pregunta contestada de verdad en la pantalla que la
	// estaba solo preguntando, y ademas la siguiente pregunta heredaria la
	// anterior: las diecinueve consecuencias se irian acumulando y la ultima
	// diria lo que pasa si contestas que si a todo.
	con := copiaCon(respuestas, id)
	conEsta, _, err := c.derivar(con)
	if err != nil {
		return pantallas.Consecuencia{}, err
	}

	var nuevas []string
	for ob := range conEsta {
		if !base[ob] {
			nuevas = append(nuevas, ob)
		}
	}
	// EL ORDEN ES EL DEL IDENTIFICADOR y no el del mapa: recorrer un mapa de Go
	// da un orden distinto en cada peticion, asi que sin esto la misma pregunta
	// ensenaria tres titulos distintos al recargar. Una pantalla que cambia sola
	// no se puede comprobar ni creer.
	sort.Strings(nuevas)

	out := pantallas.Consecuencia{N: len(nuevas)}
	marcos := map[string]bool{}
	for i, ob := range nuevas {
		t := titulos[ob]
		if i < pantallas.TresQueSeEnsenan && t.Titulo != "" {
			out.Titulos = append(out.Titulos, t.Titulo)
		}
		if t.Marco != "" {
			marcos[t.Marco] = true
		}
	}
	for m := range marcos {
		out.Marcos = append(out.Marcos, m)
	}
	sort.Strings(out.Marcos)
	return out, nil
}

// baseDe da la derivacion de las respuestas tal como estan, memorizada.
func (c *consecuenciasDeLaEntrevista) baseDe(respuestas url.Values) (
	map[string]bool, map[string]tituloDeObligacion, error) {

	clave := respuestas.Encode()
	c.mu.Lock()
	if c.claveBase == clave && c.base != nil {
		base, titulos := c.base, c.titulos
		c.mu.Unlock()
		return base, titulos, nil
	}
	c.mu.Unlock()

	base, titulos, err := c.derivar(respuestas)
	if err != nil {
		return nil, nil, err
	}
	c.mu.Lock()
	c.claveBase, c.base, c.titulos = clave, base, titulos
	c.mu.Unlock()
	return base, titulos, nil
}

// derivar convierte respuestas en el conjunto de obligaciones aplicables.
func (c *consecuenciasDeLaEntrevista) derivar(respuestas url.Values) (
	map[string]bool, map[string]tituloDeObligacion, error) {

	id := c.quienEs()
	sujeto, organizacion := id.Sujeto, id.Organizacion
	if strings.TrimSpace(sujeto) == "" {
		// SIN SUJETO NO HAY A QUIEN APLICARLE NADA, y eso es un error y no un
		// conjunto vacio: un conjunto vacio diria que contestar que si no activa
		// nada, que es una afirmacion sobre el cumplimiento de alguien salida de
		// una instalacion a medio configurar.
		return nil, nil, fmt.Errorf("esta instalacion todavia no dice de quien es, asi que " +
			"no se puede calcular a quien alcanza nada")
	}
	// LA MISMA CONVERSION QUE `plazum alcance`. Ver el encabezado.
	exp, _, err := exportarAlcance(c.paquetes, respuestas, sujeto, organizacion)
	if err != nil {
		return nil, nil, err
	}
	var al alcance
	al.Sujeto = exp.Sujeto
	al.Organizacion = exp.Organizacion
	for _, h := range exp.Hechos {
		al.Hechos = append(al.Hechos, struct {
			Pred string   `json:"pred"`
			Args []string `json:"args"`
		}{Pred: h.Pred, Args: h.Args})
	}
	d, err := montarMotor(c.paquetes, al)
	if err != nil {
		return nil, nil, err
	}
	fuera := map[string]bool{}
	for _, a := range aplicablesDe(d, al.Sujeto) {
		fuera[a.Obligacion] = true
	}
	return fuera, c.titulosDelCorpus(), nil
}

// titulosDelCorpus indexa el titulo y el marco de cada obligacion.
//
// SE RECORRE EL CORPUS Y NO SE GUARDA ENTRE PETICIONES a proposito: el corpus se
// carga una vez al arrancar y no cambia, asi que este mapa es siempre el mismo y
// construirlo cuesta un recorrido de 321 obligaciones. Guardarlo seria una
// tercera copia del corpus en memoria por un ahorro que no se mide.
func (c *consecuenciasDeLaEntrevista) titulosDelCorpus() map[string]tituloDeObligacion {
	out := map[string]tituloDeObligacion{}
	for _, p := range c.paquetes {
		for _, o := range p.Obligaciones {
			// EL MARCO ES LA URN, que es como lo nombra el resto del producto
			// (nucleo/pantalla usa p.URN en las cuatro listas del calendario).
			// Inventarle aqui un nombre corto seria una quinta forma de llamar a
			// lo mismo.
			out[o.ID] = tituloDeObligacion{Titulo: o.TituloLegible(), Marco: p.URN}
		}
	}
	return out
}

// copiaCon devuelve las respuestas con una pregunta mas contestada que SI.
//
// COPIA Y NO MUTA, y es la linea que impide el fallo mas caro de esta pieza: con
// mutacion, calcular la consecuencia de la pregunta 3 dejaria la 3 contestada
// para la 4, la 4 para la 5, y la ultima diria lo que pasa si contestas que si a
// TODO. Cada numero seria mayor que el anterior, la pantalla parecería
// funcionar, y nadie lo notaria sin recontar a mano.
func copiaCon(v url.Values, pregunta string) url.Values {
	out := make(url.Values, len(v)+1)
	for k, vs := range v {
		out[k] = append([]string(nil), vs...)
	}
	// Si ya estaba contestada que no, se quita esa respuesta: la pregunta es
	// «que pasa si contestas que si», no «que pasa si contestas las dos cosas».
	// Una pregunta con si y no a la vez es una contradiccion y el almacen la
	// rechaza, asi que dejarla daria un error en vez de una consecuencia.
	out[pantallas.ParamNo] = sinEl(out[pantallas.ParamNo], pregunta)
	if !contiene(out[pantallas.ParamSi], pregunta) {
		out[pantallas.ParamSi] = append(out[pantallas.ParamSi], pregunta)
	}
	return out
}

func sinEl(vs []string, x string) []string {
	var out []string
	for _, v := range vs {
		if v != x {
			out = append(out, v)
		}
	}
	return out
}

func contiene(vs []string, x string) bool {
	for _, v := range vs {
		if v == x {
			return true
		}
	}
	return false
}
