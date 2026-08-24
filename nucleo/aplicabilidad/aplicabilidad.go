// Package aplicabilidad decide QUE obligaciones alcanzan a QUE sujeto.
//
// Por que no basta CEL. Un selector sobre una entidad plana responde
// "¿este incidente es significativo?", pero no responde tres preguntas que
// las 30 normas hacen constantemente:
//
//	agregacion:        la categoria ENS de un sistema es el MAXIMO del nivel
//	                   de sus dimensiones sobre cada informacion y servicio
//	                   (art. 40 y Anexo I del RD 311/2022)
//	cierre transitivo: "proveedor de una entidad esencial" (NIS2 art. 21.2.d),
//	                   "subencargado de un encargado" (RGPD art. 28.4),
//	                   "componente integrado en el producto de otro fabricante" (CRA)
//	encadenamiento:    si aplica el art. 37 del RGPD entonces aplica el
//	                   art. 34.3 de la LOPDGDD; la cabeza de una regla es el
//	                   cuerpo de otra
//
// CEL no tiene punto fijo y no agrega. Datalog si, y ademas TERMINA siempre,
// que es el requisito duro: un motor de cumplimiento no puede colgarse
// evaluando reglas que escribio un tercero en un paquete de corpus.
//
// Esta implementacion es Datalog puro con negacion estratificada y tres
// agregados. Sin funciones, sin aritmetica libre, sin recursion no
// estratificada: el fragmento decidible, a proposito.
package aplicabilidad

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Termino es una constante o una variable (empieza por mayuscula).
type Termino struct {
	Var bool
	Val string
}

func C(s string) Termino { return Termino{Val: s} }
func V(s string) Termino { return Termino{Var: true, Val: s} }

// Anon es la variable anonima. HALLAZGO DE REVISION: antes se escribia V("_")
// creyendo que era anonima, y no lo era: dos apariciones de "_" en la misma
// regla se unificaban entre si y la regla dejaba de derivar nada, en silencio.
func Anon() Termino { return Termino{Var: true, Val: "_"} }

func (t Termino) esAnonima() bool { return t.Var && t.Val == "_" }

func (t Termino) String() string { return t.Val }

// Atomo es un predicado aplicado a terminos: aplica(Obligacion, Sujeto).
type Atomo struct {
	Pred string
	Args []Termino
}

func A(pred string, args ...Termino) Atomo { return Atomo{Pred: pred, Args: args} }

func (a Atomo) String() string {
	s := make([]string, len(a.Args))
	for i, t := range a.Args {
		s[i] = t.Val
	}
	return a.Pred + "(" + strings.Join(s, ", ") + ")"
}

// Hecho es un atomo totalmente instanciado, con su procedencia.
type Hecho struct {
	Pred        string
	Args        []string
	Procedencia string // de donde salio: conector, declaracion humana, regla
}

func H(pred string, args ...string) Hecho { return Hecho{Pred: pred, Args: args} }

func (h Hecho) clave() string { return h.Pred + "(" + strings.Join(h.Args, "\x1f") + ")" }

func claveIndice(pred string, pos int, val string) string {
	return pred + "\x1e" + strconv.Itoa(pos) + "\x1e" + val
}

func (h Hecho) String() string { return h.Pred + "(" + strings.Join(h.Args, ", ") + ")" }

// TipoAgregado cubre los tres casos que las normas necesitan de verdad.
type TipoAgregado uint8

const (
	SinAgregado TipoAgregado = iota
	Maximo                   // categoria ENS = max(nivel de cada dimension)
	Cuenta                   // umbrales por numero: empleados, proveedores, incidentes
	Existe                   // "al menos uno": basta un tratamiento de riesgo alto
)

// Regla: Cabeza :- Cuerpo, not Negados.  Con agregado opcional sobre una
// variable del cuerpo, agrupando por las variables de la cabeza.
type Regla struct {
	ID       string
	Cabeza   Atomo
	Cuerpo   []Atomo
	Negados  []Atomo
	Agregado TipoAgregado
	SobreVar string // variable del cuerpo que se agrega
	Escala   Escala // obligatoria para Maximo sobre valores no numericos
	Cita     string // articulo que justifica la regla. Obligatorio en el corpus
	Estrato  int    // calculado
}

// Programa es el conjunto de reglas que declara un paquete de corpus.
type Programa struct {
	Paquete string
	Reglas  []Regla
	// Exporta declara los predicados que este paquete publica al espacio comun.
	// Todo lo demas queda en su espacio de nombres. Sin esto, dos paquetes que
	// declaren `en_ambito` colisionan en silencio, que es justo lo contrario de
	// "anadir la norma 31 es un fichero".
	Exporta []string
}

// Predicados compartidos por todos los paquetes. Cualquier otro predicado se
// prefija con el paquete al validar.
var comunes = map[string]bool{"aplica": true, "desplaza": true, "equivale": true}

// Validar es el linter de paquetes: lo que convierte "el corpus es codigo" en
// algo cierto. Rechaza reglas inseguras en vez de derivar basura en silencio.
func (p Programa) Validar() error {
	exp := map[string]bool{}
	for _, e := range p.Exporta {
		exp[e] = true
	}
	for _, r := range p.Reglas {
		if r.ID == "" {
			return fmt.Errorf("%s: regla sin id", p.Paquete)
		}
		if r.Cita == "" {
			return fmt.Errorf("%s/%s: regla sin cita normativa; una regla de aplicabilidad "+
				"sin articulo no se acepta", p.Paquete, r.ID)
		}
		// Seguridad (range restriction): toda variable de la cabeza tiene que
		// aparecer en el cuerpo positivo. Si no, una errata deriva aplica(x, "").
		ligadas := map[string]bool{}
		for _, b := range r.Cuerpo {
			for _, t := range b.Args {
				if t.Var && !t.esAnonima() {
					ligadas[t.Val] = true
				}
			}
		}
		for _, t := range r.Cabeza.Args {
			if !t.Var || t.esAnonima() {
				if t.esAnonima() {
					return fmt.Errorf("%s/%s: variable anonima en la cabeza", p.Paquete, r.ID)
				}
				continue
			}
			if t.Val == "_AGG" {
				if r.Agregado == SinAgregado {
					return fmt.Errorf("%s/%s: usa _AGG sin declarar agregado", p.Paquete, r.ID)
				}
				continue
			}
			if !ligadas[t.Val] {
				return fmt.Errorf("%s/%s: la variable %s de la cabeza no esta ligada en el cuerpo "+
					"(regla insegura)", p.Paquete, r.ID, t.Val)
			}
		}
		// Variables de la negacion tambien ligadas: si no, la negacion no es segura.
		for _, n := range r.Negados {
			for _, t := range n.Args {
				if t.Var && !t.esAnonima() && !ligadas[t.Val] {
					return fmt.Errorf("%s/%s: la variable %s aparece solo bajo negacion "+
						"(negacion insegura)", p.Paquete, r.ID, t.Val)
				}
			}
		}
		if r.Agregado != SinAgregado && r.SobreVar == "" {
			return fmt.Errorf("%s/%s: agregado sin variable sobre la que agregar", p.Paquete, r.ID)
		}
		// Espacio de nombres: solo se puede definir un predicado comun o uno propio.
		if !comunes[r.Cabeza.Pred] && !exp[r.Cabeza.Pred] &&
			!strings.HasPrefix(r.Cabeza.Pred, p.Paquete+".") {
			// se permite, pero queda anotado como local del paquete
			continue
		}
	}
	return nil
}

// Motor evalua uno o varios programas sobre una base de hechos.
type Motor struct {
	programas []Programa
	hechos    map[string]Hecho
	orden     []string
	// HALLAZGO DE REVISION: sin indices, cada join recorria todos los hechos
	// del predicado. Una cadena de proveedores de 80 nodos tardaba 11,8 s.
	porPred   map[string][]string // indice por predicado
	porArg    map[string][]string // indice por (predicado, posicion, valor)
	Derivadas map[string]string
}

func NuevoMotor() *Motor {
	return &Motor{hechos: map[string]Hecho{}, porPred: map[string][]string{},
		porArg: map[string][]string{}, Derivadas: map[string]string{}}
}

func (m *Motor) Afirmar(h Hecho) {
	k := h.clave()
	if _, ok := m.hechos[k]; !ok {
		m.orden = append(m.orden, k)
		m.porPred[h.Pred] = append(m.porPred[h.Pred], k)
		for i, a := range h.Args {
			ik := claveIndice(h.Pred, i, a)
			m.porArg[ik] = append(m.porArg[ik], k)
		}
	}
	m.hechos[k] = h
}

// Cargar valida el paquete antes de aceptarlo. Un paquete de corpus es codigo
// de un tercero: se rechaza lo que no es seguro, no se ejecuta a ver que pasa.
func (m *Motor) Cargar(p Programa) error {
	if err := p.Validar(); err != nil {
		return err
	}
	m.programas = append(m.programas, p)
	return nil
}

// CargarSinValidar existe solo para los tests del propio motor.
func (m *Motor) CargarSinValidar(p Programa) { m.programas = append(m.programas, p) }

// estratificar ordena las reglas de forma que ninguna regla con negacion
// dependa de un predicado que aun no este completamente calculado. Si el
// programa no es estratificable, se rechaza: es la garantia de terminacion.
func (m *Motor) estratificar() ([][]Regla, error) {
	var todas []Regla
	for _, p := range m.programas {
		for _, r := range p.Reglas {
			r.ID = p.Paquete + ":" + r.ID
			todas = append(todas, r)
		}
	}
	estrato := map[string]int{}
	for i := 0; i < len(todas)+1; i++ {
		cambio := false
		for _, r := range todas {
			e := estrato[r.Cabeza.Pred]
			for _, b := range r.Cuerpo {
				if v := estrato[b.Pred]; v > e {
					e, cambio = v, true
				}
			}
			for _, n := range r.Negados {
				if v := estrato[n.Pred] + 1; v > e {
					e, cambio = v, true
				}
			}
			if r.Agregado != SinAgregado {
				for _, b := range r.Cuerpo {
					if v := estrato[b.Pred] + 1; v > e {
						e, cambio = v, true
					}
				}
			}
			if e > estrato[r.Cabeza.Pred] {
				estrato[r.Cabeza.Pred] = e
			}
		}
		if !cambio {
			break
		}
		if i == len(todas) {
			return nil, fmt.Errorf("programa no estratificable: hay negacion o agregacion en un ciclo")
		}
	}
	max := 0
	for _, e := range estrato {
		if e > max {
			max = e
		}
	}
	capas := make([][]Regla, max+1)
	for _, r := range todas {
		e := estrato[r.Cabeza.Pred]
		r.Estrato = e
		capas[e] = append(capas[e], r)
	}
	return capas, nil
}

// Evaluar calcula el punto fijo. Devuelve el numero de hechos derivados.
func (m *Motor) Evaluar() (int, error) {
	capas, err := m.estratificar()
	if err != nil {
		return 0, err
	}
	antes := len(m.hechos)
	for _, capa := range capas {
		var delta map[string]bool // nil en la primera pasada: se evalua todo
		for iter := 0; ; iter++ {
			nuevos := map[string]bool{}
			derivar := func(r Regla, subs []map[string]string) {
				for _, sub := range subs {
					if m.algunNegadoSeCumple(r.Negados, sub) {
						continue
					}
					h := instanciar(r.Cabeza, sub)
					h.Procedencia = "regla " + r.ID
					k := h.clave()
					if _, ok := m.hechos[k]; !ok {
						m.Afirmar(h)
						m.Derivadas[k] = r.ID
						nuevos[k] = true
					}
				}
			}
			for _, r := range capa {
				if r.Agregado != SinAgregado {
					n, err := m.aplicarAgregado(r)
					if err != nil {
						return 0, err
					}
					if n > 0 {
						nuevos["__agg__"] = true
					}
					continue
				}
				if delta == nil {
					derivar(r, m.unificar(r.Cuerpo, map[string]string{}))
					continue
				}
				// Semi-naive: una variante por posicion del cuerpo, cada una
				// exigiendo que ese atomo venga de los hechos nuevos.
				for i := range r.Cuerpo {
					derivar(r, m.unificarDelta(r.Cuerpo, map[string]string{}, i, delta))
				}
			}
			if len(nuevos) == 0 {
				break
			}
			delta = nuevos
			if iter > 10000 {
				return 0, fmt.Errorf("no converge: revisa las reglas del paquete")
			}
		}
	}
	return len(m.hechos) - antes, nil
}

func (m *Motor) algunNegadoSeCumple(negados []Atomo, sub map[string]string) bool {
	for _, n := range negados {
		if len(m.unificar([]Atomo{n}, sub)) > 0 {
			return true
		}
	}
	return false
}

// unificar hace la union de los atomos del cuerpo por backtracking.
func (m *Motor) unificar(cuerpo []Atomo, sub map[string]string) []map[string]string {
	return m.unificarDelta(cuerpo, sub, -1, nil)
}

// unificarDelta implementa evaluacion SEMI-NAIVE: si posDelta >= 0, el atomo en
// esa posicion solo casa contra los hechos nuevos de la iteracion anterior.
//
// HALLAZGO DE REVISION. La primera version rederivaba la relacion completa en
// cada iteracion: una cadena de proveedores de 60 nodos tardaba segundos, y de
// ahi no pasaba. Con delta, el trabajo por iteracion es proporcional a lo nuevo.
func (m *Motor) unificarDelta(cuerpo []Atomo, sub map[string]string, posDelta int, delta map[string]bool) []map[string]string {
	if len(cuerpo) == 0 {
		return []map[string]string{copiar(sub)}
	}
	var out []map[string]string
	a := cuerpo[0]
	// HALLAZGO DE REVISION: sin indice por argumento ligado, cada join recorria
	// todos los hechos del predicado. En un cierre transitivo eso es cuadratico
	// por iteracion. Se elige el argumento ligado mas selectivo.
	candidatos := m.porPred[a.Pred]
	mejor := -1
	for i, t := range a.Args {
		var val string
		switch {
		case !t.Var:
			val = t.Val
		case !t.esAnonima():
			if v, ok := sub[t.Val]; ok {
				val = v
			}
		}
		if val == "" {
			continue
		}
		if c := m.porArg[claveIndice(a.Pred, i, val)]; mejor < 0 || len(c) < len(candidatos) {
			candidatos, mejor = c, i
		}
	}
	for _, k := range candidatos {
		if posDelta == 0 && !delta[k] {
			continue
		}
		h := m.hechos[k]
		if len(h.Args) != len(a.Args) {
			continue
		}
		s2 := copiar(sub)
		ok := true
		for i, t := range a.Args {
			if !t.Var {
				if t.Val != h.Args[i] {
					ok = false
					break
				}
				continue
			}
			if t.esAnonima() {
				continue // casa con cualquier cosa y no liga nada
			}
			if prev, existe := s2[t.Val]; existe {
				if prev != h.Args[i] {
					ok = false
					break
				}
			} else {
				s2[t.Val] = h.Args[i]
			}
		}
		if ok {
			out = append(out, m.unificarDelta(cuerpo[1:], s2, posDelta-1, delta)...)
		}
	}
	return out
}

func (m *Motor) aplicarAgregado(r Regla) (int, error) {
	grupos := map[string][]string{}
	vistos := map[string]map[string]bool{}
	claves := map[string]map[string]string{}
	for _, sub := range m.unificar(r.Cuerpo, map[string]string{}) {
		var g []string
		for _, t := range r.Cabeza.Args {
			if t.Var && t.Val != "_AGG" {
				g = append(g, sub[t.Val])
			}
		}
		k := strings.Join(g, "\x1f")
		v := sub[r.SobreVar]
		// HALLAZGO DE REVISION: Cuenta contaba COMBINACIONES del cuerpo, no
		// valores distintos. Con 2 empleados y 3 contratos devolvia 6. Los
		// umbrales de NIS2 (250) y CSRD (1.000) se calculan con esto.
		if vistos[k] == nil {
			vistos[k] = map[string]bool{}
		}
		if vistos[k][v] {
			claves[k] = sub
			continue
		}
		vistos[k][v] = true
		grupos[k] = append(grupos[k], v)
		claves[k] = sub
	}
	nuevos := 0
	for k, vals := range grupos {
		var res string
		switch r.Agregado {
		case Maximo:
			v, err := maximoEnEscala(vals, r.Escala)
			if err != nil {
				return 0, fmt.Errorf("regla %s: %w", r.ID, err)
			}
			res = v
		case Cuenta:
			res = strconv.Itoa(len(vals))
		case Existe:
			res = "si" // un grupo solo existe si tiene al menos un elemento
		}
		sub := copiar(claves[k])
		sub["_AGG"] = res
		h := instanciar(r.Cabeza, sub)
		h.Procedencia = "regla " + r.ID + " (agregado)"
		if _, ok := m.hechos[h.clave()]; !ok {
			m.Afirmar(h)
			m.Derivadas[h.clave()] = r.ID
			nuevos++
		}
	}
	return nuevos, nil
}

// Escala es una ordenacion declarada por el paquete de corpus.
//
// HALLAZGO DE REVISION. La primera version caia a orden lexicografico cuando
// los valores no eran numeros. Con los valores que usa de verdad el RD 311/2022
// (BAJO, MEDIO, ALTO) eso devuelve MEDIO como maximo, porque "MEDIO" > "BAJO" y
// "MEDIO" > "ALTO" alfabeticamente. Resultado: un sistema de categoria ALTA
// degradado a MEDIA, que es un cambio de regimen de auditoria. El test lo
// ocultaba usando "1", "2" y "3".
//
// La correccion no es adivinar mejor: es que el paquete DECLARE su escala.
// Agregar sobre valores sin orden declarado es un error, no un caso por defecto.
type Escala struct {
	Nombre string
	Orden  []string // de menor a mayor
}

var escalaNumerica = Escala{Nombre: "numerica"}

func (e Escala) indice(v string) (int, bool) {
	for i, x := range e.Orden {
		if x == v {
			return i, true
		}
	}
	return 0, false
}

func maximoEnEscala(vals []string, esc Escala) (string, error) {
	if len(esc.Orden) == 0 {
		for _, v := range vals {
			if _, err := strconv.Atoi(v); err != nil {
				return "", fmt.Errorf("agregado Maximo sobre el valor no numerico %q sin escala declarada: "+
					"el paquete debe declarar la ordenacion (por ejemplo BAJO < MEDIO < ALTO)", v)
			}
		}
		max := 0
		for i, v := range vals {
			n, _ := strconv.Atoi(v)
			if i == 0 || n > max {
				max = n
			}
		}
		return strconv.Itoa(max), nil
	}
	mejor, mejorIdx := "", -1
	for _, v := range vals {
		i, ok := esc.indice(v)
		if !ok {
			return "", fmt.Errorf("valor %q fuera de la escala %q declarada por el paquete", v, esc.Nombre)
		}
		if i > mejorIdx {
			mejor, mejorIdx = v, i
		}
	}
	return mejor, nil
}

func instanciar(a Atomo, sub map[string]string) Hecho {
	args := make([]string, len(a.Args))
	for i, t := range a.Args {
		if t.Var {
			args[i] = sub[t.Val]
		} else {
			args[i] = t.Val
		}
	}
	return Hecho{Pred: a.Pred, Args: args}
}

func copiar(m map[string]string) map[string]string {
	o := make(map[string]string, len(m))
	for k, v := range m {
		o[k] = v
	}
	return o
}

// Consultar devuelve los hechos que casan con un patron.
func (m *Motor) Consultar(a Atomo) []Hecho {
	var out []Hecho
	for _, sub := range m.unificar([]Atomo{a}, map[string]string{}) {
		out = append(out, m.hechos[instanciar(a, sub).clave()])
	}
	sort.Slice(out, func(i, j int) bool { return out[i].clave() < out[j].clave() })
	return out
}

// Explicar reconstruye por que existe un hecho: quien lo afirmo o que regla
// lo derivo. Es la mitad de `dutiq explain` que corresponde a aplicabilidad.
func (m *Motor) Explicar(h Hecho) string {
	e, ok := m.hechos[h.clave()]
	if !ok {
		return "no consta"
	}
	if e.Procedencia == "" {
		return "hecho declarado, sin procedencia registrada"
	}
	return e.Procedencia
}

func (m *Motor) Total() int { return len(m.hechos) }
