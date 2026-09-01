// Package auditoria es el programa de auditoria interna con ARRASTRE entre
// ciclos.
//
// QUE ES Y QUE NO ES. No es "un sitio donde apuntar auditorias": es el objeto
// que responde la pregunta que ningun GRC contesta bien y que un auditor externo
// hace siempre, **que se ha quedado sin auditar y desde cuando**. Una auditoria
// interna se planifica por ciclos (tres anos es lo habitual) y lo que decide si
// el programa vale no es lo que se audito, es lo que NO y cuanto lleva sin
// auditarse. Ese numero, hoy, lo lleva alguien en una hoja de calculo o no lo
// lleva nadie.
//
// EL ARRASTRE SON DOS COSAS, Y LAS DOS SE CALCULAN, no se apuntan:
//
//	COBERTURA   lo que el ciclo anterior no llego a auditar entra en el
//	            siguiente con su edad. Un programa que empieza cada ciclo de
//	            cero es un programa que puede no auditar nunca una unidad y
//	            no enterarse.
//	HALLAZGOS   los abiertos siguen abiertos aunque cambie el ciclo, y llevan
//	            CUANTOS CICLOS llevan asi. Un hallazgo con tres ciclos encima
//	            no es un hallazgo, es una decision de no arreglarlo que nadie
//	            ha escrito.
//
// LO QUE SE AUDITA SALE DEL CORPUS, NUNCA DE AQUI (invariante 2). Una Unidad es
// (paquete, obligacion): quien construye el programa las saca de los paquetes
// instalados y de lo que el alcance dice que aplica. Este paquete no sabe el
// nombre de ninguna norma y no puede saberlo.
//
// LA LEY DE CONSERVACION: toda unidad del alcance cae en EXACTAMENTE UN cubo por
// ciclo (auditada, diferida con motivo, sin auditar), y la suma da el alcance.
// Un programa que presenta "hemos auditado 40" sin decir sobre cuantas es una
// cifra que no significa nada.
//
// Y LO NO AUDITADO NO ES UNA NO CONFORMIDAD. Es una ausencia de dato, y la
// diferencia importa igual que en el calendario y en la UAR: decir "incumple" de
// algo que solo esta sin mirar es acusar en falso, y quien lo lea deja de
// creerse el resto de la pantalla.
//
// # El diferimiento, escrito con la leccion de la excusa ya aprendida
//
// Diferir es la valvula: sin ella, un programa con una unidad imposible de
// auditar este ano se queda bloqueado y la regla se acaba saltando por otro
// sitio. Pero un diferimiento es un HECHO y se contrasta contra el alcance,
// igual que una excusa se contrasta contra lo ilegible del censo:
//
//   - la unidad tiene que estar EN EL ALCANCE del programa; diferir algo que no
//     se iba a auditar no difiere nada;
//   - no se difiere dos veces la misma unidad en el mismo ciclo;
//   - quien, por que y cuando, obligatorios;
//   - y NO cuenta como cobertura. Un diferimiento explica por que falta, no
//     hace que deje de faltar.
//
// Esto se escribe ASI DESDE EL PRINCIPIO y no despues de que lo encuentre una
// revision: el mismo agujero acababa de costar un P1 en la excusa de la UAR, y
// un arreglo que solo tapa el caso encontrado y no mira a sus hermanos deja el
// siguiente hallazgo servido.
package auditoria

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var (
	// ErrFueraDelAlcance: el hecho apunta a una unidad que el programa no
	// tiene. Es la guarda del invariante 7 aqui.
	ErrFueraDelAlcance = errors.New("auditoria: la unidad no esta en el alcance del programa")
	// ErrHechoIncompleto: falta quien, por que o cuando.
	ErrHechoIncompleto = errors.New("auditoria: al hecho le faltan datos")
	// ErrSinEfecto: el hecho no cambia nada. Un hecho que no cambia nada en un
	// registro append-only es ruido que alguien tendra que interpretar.
	ErrSinEfecto = errors.New("auditoria: el hecho no cambia nada")
	// ErrPrograma: falta algo para abrir el programa.
	ErrPrograma = errors.New("auditoria: faltan datos para abrir el programa")
	// ErrHallazgoDesconocido: se intenta cerrar un hallazgo que no consta.
	ErrHallazgoDesconocido = errors.New("auditoria: ese hallazgo no consta en el programa")
)

// Unidad es lo auditable, y sale del CORPUS.
//
// El par (paquete, obligacion) es la identidad, y las dos mitades vienen de
// datos: este paquete no nombra ninguna norma y no puede (invariante 2). La
// version del paquete viaja al lado porque un programa que se auditó contra la
// version 0.2 de un paquete y se lee contra la 0.3 esta comparando dos alcances
// distintos, y eso hay que poder verlo.
type Unidad struct {
	Paquete string
	Version string
	// Obligacion es el identificador dentro del paquete.
	Obligacion string
	// Titulo es para que una persona lo reconozca. NO identifica.
	Titulo string
}

// Clave es la identidad. Nunca por indice ni por posicion en una lista: el
// alcance se reordena cada vez que el corpus gana un paquete.
func (u Unidad) Clave() string { return u.Paquete + "|" + u.Obligacion }

// Ciclo es el periodo que cubre un programa.
type Ciclo struct {
	// Nombre es como lo llama la organizacion ("2026-2028").
	Nombre string
	Desde  time.Time
	Hasta  time.Time
}

func (c Ciclo) valido() bool {
	return strings.TrimSpace(c.Nombre) != "" && !c.Desde.IsZero() && !c.Hasta.IsZero() &&
		c.Hasta.After(c.Desde)
}

// Cubre dice si un instante cae dentro del ciclo. Los dos extremos entran: una
// auditoria del ultimo dia del ciclo es del ciclo.
func (c Ciclo) Cubre(t time.Time) bool {
	return !t.Before(c.Desde) && !t.After(c.Hasta)
}

// Clase de hallazgo. Vocabulario CERRADO, por lo mismo que el de incidente y el
// de veredicto: una clase libre convierte el programa en un cajon de notas, y lo
// que sostiene el arrastre es que "no conformidad mayor" signifique una cosa.
type Clase uint8

const (
	// NoConformidadMayor: fallo del sistema, no de un caso.
	NoConformidadMayor Clase = iota
	// NoConformidadMenor: un caso aislado.
	NoConformidadMenor
	// Observacion: no incumple, pero va a incumplir.
	Observacion
	// Oportunidad: ni incumple ni va a incumplir. Se registra porque el valor
	// de una auditoria interna que solo encuentra fallos se agota rapido.
	Oportunidad
)

var nombresDeClase = [...]string{"no conformidad mayor", "no conformidad menor",
	"observacion", "oportunidad de mejora"}

func (c Clase) String() string {
	if int(c) < len(nombresDeClase) {
		return nombresDeClase[c]
	}
	return fmt.Sprintf("clase desconocida (%d)", uint8(c))
}

func (c Clase) Valida() bool { return int(c) < len(nombresDeClase) }

// Exige dice si la clase obliga a un plan de accion. Las observaciones y las
// oportunidades no: confundirlas con las no conformidades es como un programa
// acaba con cuarenta "hallazgos abiertos" que nadie mira.
func (c Clase) Exige() bool { return c == NoConformidadMayor || c == NoConformidadMenor }

// ClaseDe traduce el nombre que viaja fuera del proceso.
//
// El vocabulario es cerrado y por eso esto puede fallar: una clase que no se
// reconoce NO se convierte en el cero (que es "no conformidad mayor", la mas
// grave) ni en la ultima. Es el invariante 8 en una frontera de lectura, y aqui
// el error va en las dos direcciones: caer a "mayor" acusa de mas, y caer a
// "oportunidad" esconde un incumplimiento.
func ClaseDe(nombre string) (Clase, error) {
	for i, n := range nombresDeClase {
		if n == nombre {
			return Clase(i), nil
		}
	}
	return 0, fmt.Errorf("%w: clase %q no reconocida (%s). No se toma ninguna por defecto: caer "+
		"a la primera acusa de mas y caer a la ultima esconde un incumplimiento",
		ErrHechoIncompleto, nombre, strings.Join(nombresDeClase[:], ", "))
}

// Sesion es una auditoria concreta: quien la hizo, cuando y sobre que unidades.
//
// Es un HECHO INMUTABLE. Ampliar el alcance de una auditoria ya hecha no se
// edita: se registra otra sesion.
type Sesion struct {
	ID string
	// Auditor por identificador estable, no por nombre.
	Auditor  string
	Cuando   time.Time
	Unidades []string // claves de Unidad
	// Alcance es el texto que describe que se miro. Lo escribe una persona.
	Alcance string
}

// Hallazgo es lo que sale de una sesion. Tambien inmutable: se cierra con OTRO
// hecho, no editando este.
type Hallazgo struct {
	ID     string
	Sesion string
	Unidad string // clave de Unidad
	Clase  Clase
	// Texto lo escribe el auditor.
	Texto  string
	Quien  string
	Cuando time.Time
}

// Cierre de un hallazgo. Es el segundo hecho, y lleva su propia evidencia.
type CierreDeHallazgo struct {
	Hallazgo string
	Quien    string
	Cuando   time.Time
	// Como es que se hizo para cerrarlo. Sin esto, "cerrado" no dice nada.
	Como string
}

// Diferimiento deja una unidad fuera de la cobertura del ciclo, CON MOTIVO.
type Diferimiento struct {
	Unidad string
	Quien  string
	Motivo string
	Cuando time.Time
}

// Programa es el objeto. Sus hechos son privados y solo se anaden.
type Programa struct {
	id      string
	ciclo   Ciclo
	alcance map[string]Unidad
	orden   []string // el alcance en orden estable

	sesiones      []Sesion
	hallazgos     []Hallazgo
	cierres       []CierreDeHallazgo
	diferimientos []Diferimiento

	// arrastre es lo que viene del ciclo anterior. Se inyecta al abrir: este
	// paquete no va a buscarlo, porque de donde salga (otro fichero, el ledger,
	// otra instalacion) es cosa del adaptador.
	arrastre Arrastre
}

// Arrastre es lo que un ciclo le pasa al siguiente.
type Arrastre struct {
	// SinAuditar son las claves de unidad que el ciclo anterior no cubrio, con
	// cuantos ciclos seguidos llevan sin auditarse.
	SinAuditar map[string]int
	// Abiertos son los hallazgos que siguen sin cerrar, con cuantos ciclos
	// llevan abiertos.
	Abiertos map[string]int
	// DeCiclo es el nombre del ciclo del que viene, para poder decirlo.
	DeCiclo string
	// Salidas son las unidades que venian arrastradas y YA NO ESTAN en el
	// alcance de este ciclo. No se arrastran (seguirian echandose de menos para
	// siempre) y no se callan: que una unidad salga del alcance es una decision
	// del alcance, y si nadie la ve, una obligacion puede desaparecer del
	// programa sin que conste que desaparecio.
	Salidas []string
}

// Abrir construye el programa. El arrastre puede ser el cero: un primer ciclo no
// arrastra nada, y eso es distinto de "no lo hemos mirado".
func Abrir(id string, c Ciclo, alcance []Unidad, arr Arrastre) (*Programa, error) {
	var faltan []string
	if strings.TrimSpace(id) == "" {
		faltan = append(faltan, "id del programa")
	}
	if !c.valido() {
		faltan = append(faltan, "ciclo con nombre, desde y hasta (y hasta despues de desde)")
	}
	if len(alcance) == 0 {
		faltan = append(faltan, "al menos una unidad en el alcance: un programa sin alcance no "+
			"puede decir que se ha quedado sin auditar, que es lo unico que hace")
	}
	if len(faltan) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrPrograma, strings.Join(faltan, "; "))
	}
	p := &Programa{id: id, ciclo: c, alcance: map[string]Unidad{}, arrastre: normalizar(arr)}
	for _, u := range alcance {
		if strings.TrimSpace(u.Paquete) == "" || strings.TrimSpace(u.Obligacion) == "" {
			return nil, fmt.Errorf("%w: una unidad sin paquete o sin obligacion no se puede "+
				"emparejar con nada", ErrPrograma)
		}
		if _, ya := p.alcance[u.Clave()]; ya {
			continue // el alcance es un conjunto: repetir una unidad no la audita dos veces
		}
		p.alcance[u.Clave()] = u
		p.orden = append(p.orden, u.Clave())
	}
	sort.Strings(p.orden)

	// EL ARRASTRE SE CONTRASTA CONTRA EL ALCANCE DE AHORA. Una unidad que el
	// ciclo anterior no audito y que ya no esta en el alcance no se arrastra:
	// se dice que salio. Arrastrarla daria un programa que echa de menos para
	// siempre algo que la organizacion dejo de tener.
	for clave := range p.arrastre.SinAuditar {
		if _, esta := p.alcance[clave]; !esta {
			delete(p.arrastre.SinAuditar, clave)
			p.arrastre.Salidas = append(p.arrastre.Salidas, clave)
		}
	}
	sort.Strings(p.arrastre.Salidas)
	return p, nil
}

func normalizar(a Arrastre) Arrastre {
	if a.SinAuditar == nil {
		a.SinAuditar = map[string]int{}
	}
	if a.Abiertos == nil {
		a.Abiertos = map[string]int{}
	}
	return a
}

func (p *Programa) ID() string                 { return p.id }
func (p *Programa) Ciclo() Ciclo               { return p.ciclo }
func (p *Programa) Alcance() int               { return len(p.orden) }
func (p *Programa) DelCicloAnterior() Arrastre { return p.arrastre }

// Unidades devuelve el alcance en orden estable.
func (p *Programa) Unidades() []Unidad {
	out := make([]Unidad, 0, len(p.orden))
	for _, k := range p.orden {
		out = append(out, p.alcance[k])
	}
	return out
}
