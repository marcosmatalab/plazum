package auditoria

import (
	"fmt"
	"sort"
	"strings"
)

// Cobertura es el cubo en el que cae una unidad del alcance en este ciclo.
type Cobertura string

const (
	// Auditada: hay al menos una sesion de este ciclo que la incluye.
	Auditada Cobertura = "auditada"
	// Diferida: alguien dijo por escrito por que no se audita este ciclo. NO es
	// cobertura: explica por que falta, no hace que deje de faltar.
	Diferida Cobertura = "diferida con motivo"
	// SinAuditar: ni lo uno ni lo otro. NO es una no conformidad.
	SinAuditar Cobertura = "sin auditar"
)

// CoberturasPosibles es el vocabulario entero, para que una pantalla pinte los
// cubos vacios. Uno que solo aparece cuando tiene algo dentro es un cubo que
// nadie echa de menos.
func CoberturasPosibles() []Cobertura { return []Cobertura{Auditada, Diferida, SinAuditar} }

// LaFraseDeLoNoAuditado es el patron de la casa, aplicado aqui.
//
// Una unidad sin auditar NO ES UNA NO CONFORMIDAD. Puede estar perfectamente
// implantada y lo unico que falte es que alguien la mire. Presentarlo como
// hallazgo convierte un programa a medias en una lista de acusaciones falsas.
const LaFraseDeLoNoAuditado = "Esto NO dice que estas obligaciones se incumplan: dice que en este " +
	"ciclo no consta que nadie las haya auditado."

// LaFraseDeLaIndependencia acompana SIEMPRE a un conflicto de independencia.
//
// Es la otra mitad de la doctrina del falso positivo, y la que se olvida: aqui
// el dato SI consta, y aun asi presentarlo solo acusa. Que el auditor sea quien
// responde de lo auditado es lo que la exigencia mira, pero en una empresa de
// veinte personas puede no haber otra persona, y esa es una respuesta legitima
// que se escribe. Lo que no vale es que no conste.
const LaFraseDeLaIndependencia = "Esto NO dice que la auditoria este mal hecha: dice que quien " +
	"audito es la misma persona que responde de lo auditado. En una organizacion pequena puede " +
	"no haber otra, y esa es una respuesta legitima que hay que escribir. Lo que no vale es que " +
	"no conste."

// LaFraseDeLoSinResponsable acompana a las unidades de las que no consta quien
// responde.
//
// Es la tercera rama de Independencia, la que hoy se salta en silencio: sin
// responsable asignado no hay conflicto que ver, pero tampoco hay comprobacion
// hecha, y las dos cosas no son la misma. Un cubo que solo aparece cuando tiene
// algo dentro es un cubo que nadie echa de menos.
const LaFraseDeLoSinResponsable = "Esto NO dice que nadie responda de estas unidades: dice que " +
	"no consta quien, asi que la independencia del auditor no se ha podido mirar."

// LaFraseDeLaSalidaDelAlcance acompana a lo que el ciclo anterior arrastraba y
// este ciclo ya no tiene en el alcance.
//
// Se dice y no se arrastra, que son las dos mitades: arrastrarla daria un
// programa que echa de menos para siempre algo que la organizacion dejo de
// tener, y callarla dejaria que una obligacion desapareciera del programa sin
// que constara que desaparecio.
const LaFraseDeLaSalidaDelAlcance = "Esto NO dice que estas unidades se dejaran sin cumplir: " +
	"dice que el alcance de este ciclo ya no las incluye, y que por eso dejan de arrastrarse."

// Auditar registra una sesion. Es un hecho inmutable.
//
// LA GUARDA DEL EMPAREJAMIENTO (invariante 7): las unidades de la sesion tienen
// que estar en el alcance del programa, y casan por la clave (paquete|obligacion),
// que sale de los datos del corpus. Nunca por posicion en una lista: el alcance
// se reordena cada vez que el corpus gana un paquete.
//
// Y LA SESION TIENE QUE CAER DENTRO DEL CICLO. Una auditoria de hace cuatro anos
// no cubre este ciclo, y admitirla en silencio es como un programa presenta
// cobertura completa sin haber auditado nada: la comprobacion de arrastre entera
// se apoya en que "auditada" signifique "auditada EN ESTE CICLO".
func (p *Programa) Auditar(s Sesion) error {
	var faltan []string
	if strings.TrimSpace(s.ID) == "" {
		faltan = append(faltan, "id de la sesion")
	}
	if strings.TrimSpace(s.Auditor) == "" {
		faltan = append(faltan, "auditor (identificador estable, no el nombre)")
	}
	if s.Cuando.IsZero() {
		faltan = append(faltan, "cuando")
	}
	if len(s.Unidades) == 0 {
		faltan = append(faltan, "al menos una unidad: una sesion que no mira nada no es una sesion")
	}
	if len(faltan) > 0 {
		return fmt.Errorf("%w: %s", ErrHechoIncompleto, strings.Join(faltan, "; "))
	}
	if !p.ciclo.Cubre(s.Cuando) {
		return fmt.Errorf("%w: la sesion es del %s y el ciclo %q va del %s al %s.\n"+
			"  Una auditoria de fuera del ciclo no cubre este ciclo. Admitirla seria presentar "+
			"cobertura de trabajo que no se hizo en el periodo que se esta certificando.\n"+
			"  Arreglo: registrarla en el programa de su ciclo, que es donde cuenta",
			ErrFueraDelAlcance, s.Cuando.Format("2006-01-02"), p.ciclo.Nombre,
			p.ciclo.Desde.Format("2006-01-02"), p.ciclo.Hasta.Format("2006-01-02"))
	}
	var fuera []string
	for _, u := range s.Unidades {
		if _, esta := p.alcance[u]; !esta {
			fuera = append(fuera, u)
		}
	}
	if len(fuera) > 0 {
		sort.Strings(fuera)
		return fmt.Errorf("%w: la sesion dice haber auditado %s que no esta%s en el alcance (%s).\n"+
			"  Se empareja por paquete|obligacion, que sale del corpus instalado y no de aqui.\n"+
			"  Si de verdad se audito eso, lo que falta es meterlo en el alcance del programa: "+
			"auditar fuera del alcance sube el recuento de cobertura sin cubrir nada de lo que "+
			"se iba a cubrir",
			ErrFueraDelAlcance, plural(len(fuera), "1 unidad", "%d unidades"),
			plural(len(fuera), "", "n"), listaCorta(fuera))
	}
	p.sesiones = append(p.sesiones, s)
	return nil
}

// Diferir deja una unidad fuera de la cobertura del ciclo, con motivo.
//
// ESCRITO CON LA LECCION DE LA EXCUSA YA APRENDIDA, y no despues de que lo
// encuentre una revision: la unidad tiene que estar en el alcance, no se difiere
// dos veces, y quien/motivo/cuando son obligatorios. El agujero equivalente en
// la excusa de la UAR (un rango que no se contrastaba contra nada) costo un P1
// el mismo dia, y un arreglo que solo tapa el caso encontrado deja el siguiente
// hallazgo servido.
func (p *Programa) Diferir(d Diferimiento) error {
	if strings.TrimSpace(d.Quien) == "" || strings.TrimSpace(d.Motivo) == "" || d.Cuando.IsZero() {
		return fmt.Errorf("%w: un diferimiento sin quien, sin por que o sin cuando es una unidad "+
			"que desaparece de la cobertura y nadie responde de ella", ErrHechoIncompleto)
	}
	if _, esta := p.alcance[d.Unidad]; !esta {
		return fmt.Errorf("%w: %q.\n"+
			"  Diferir algo que no se iba a auditar no difiere nada, y deja en el registro un "+
			"hecho que parece una decision y no lo es",
			ErrFueraDelAlcance, d.Unidad)
	}
	for _, ya := range p.diferimientos {
		if ya.Unidad == d.Unidad {
			return fmt.Errorf("%w: la unidad %q ya estaba diferida en este ciclo (por %s, %s).\n"+
				"  No se anade: un hecho que no cambia nada en un registro append-only es ruido "+
				"que alguien tendra que interpretar dentro de un ano",
				ErrSinEfecto, d.Unidad, ya.Quien, ya.Cuando.Format("2006-01-02"))
		}
	}
	// Y SI YA ESTA AUDITADA, diferirla es contradecirse. No se admite: el
	// informe diria a la vez que se miro y que se dejo de mirar.
	if p.CoberturaDe(d.Unidad) == Auditada {
		return fmt.Errorf("%w: la unidad %q ya esta auditada en este ciclo.\n"+
			"  Diferir lo que ya se miro deja el informe diciendo a la vez que se hizo y que se "+
			"dejo para el ciclo siguiente", ErrSinEfecto, d.Unidad)
	}
	p.diferimientos = append(p.diferimientos, d)
	return nil
}

// Anotar registra un hallazgo. Inmutable: se cierra con OTRO hecho.
func (p *Programa) Anotar(h Hallazgo) error {
	var faltan []string
	if strings.TrimSpace(h.ID) == "" {
		faltan = append(faltan, "id del hallazgo")
	}
	if strings.TrimSpace(h.Quien) == "" {
		faltan = append(faltan, "quien lo anota")
	}
	if strings.TrimSpace(h.Texto) == "" {
		faltan = append(faltan, "texto: un hallazgo sin descripcion no se puede cerrar ni discutir")
	}
	if h.Cuando.IsZero() {
		faltan = append(faltan, "cuando")
	}
	if !h.Clase.Valida() {
		faltan = append(faltan, "clase valida ("+strings.Join(nombresDeClase[:], ", ")+")")
	}
	if len(faltan) > 0 {
		return fmt.Errorf("%w: %s", ErrHechoIncompleto, strings.Join(faltan, "; "))
	}
	if _, esta := p.alcance[h.Unidad]; !esta {
		return fmt.Errorf("%w: el hallazgo %q apunta a %q.\n"+
			"  Un hallazgo sobre algo que no esta en el alcance no se puede arrastrar: el ciclo "+
			"siguiente no sabria contra que unidad comprobar si se cerro",
			ErrFueraDelAlcance, h.ID, h.Unidad)
	}
	for _, ya := range p.hallazgos {
		if ya.ID == h.ID {
			return fmt.Errorf("%w: ya consta un hallazgo con el id %q. Cambiar uno anotado no se "+
				"hace encima: se anota otro", ErrSinEfecto, h.ID)
		}
	}
	p.hallazgos = append(p.hallazgos, h)
	return nil
}

// Cerrar cierra un hallazgo. Es el segundo hecho.
func (p *Programa) Cerrar(c CierreDeHallazgo) error {
	if strings.TrimSpace(c.Quien) == "" || strings.TrimSpace(c.Como) == "" || c.Cuando.IsZero() {
		return fmt.Errorf("%w: un cierre sin quien, sin COMO y sin cuando no dice nada.\n"+
			"  El \"como\" es lo que un auditor externo va a pedir: \"cerrado\" a secas no es "+
			"evidencia de nada", ErrHechoIncompleto)
	}
	if !p.existeHallazgo(c.Hallazgo) {
		return fmt.Errorf("%w: %q.\n"+
			"  Se cierran hallazgos de ESTE programa. Si es de un ciclo anterior, viene por el "+
			"arrastre y se cierra en el programa donde se anoto",
			ErrHallazgoDesconocido, c.Hallazgo)
	}
	if p.estaCerrado(c.Hallazgo) {
		return fmt.Errorf("%w: el hallazgo %q ya estaba cerrado", ErrSinEfecto, c.Hallazgo)
	}
	p.cierres = append(p.cierres, c)
	return nil
}

func (p *Programa) existeHallazgo(id string) bool {
	for _, h := range p.hallazgos {
		if h.ID == id {
			return true
		}
	}
	return false
}

func (p *Programa) estaCerrado(id string) bool {
	for _, c := range p.cierres {
		if c.Hallazgo == id {
			return true
		}
	}
	return false
}

// CoberturaDe dice en que cubo cae una unidad.
//
// EL ORDEN IMPORTA: auditada gana a diferida. Si una unidad se difirio en enero
// y se acabo auditando en octubre, esta auditada; lo contrario dejaria una
// unidad mirada contada como pendiente.
func (p *Programa) CoberturaDe(clave string) Cobertura {
	for _, s := range p.sesiones {
		for _, u := range s.Unidades {
			if u == clave {
				return Auditada
			}
		}
	}
	for _, d := range p.diferimientos {
		if d.Unidad == clave {
			return Diferida
		}
	}
	return SinAuditar
}

// Cuenta es la LEY DE CONSERVACION del programa: toda unidad del alcance cae en
// exactamente un cubo, y los vacios tambien salen.
func (p *Programa) Cuenta() map[Cobertura]int {
	out := map[Cobertura]int{}
	for _, c := range CoberturasPosibles() {
		out[c] = 0
	}
	for _, k := range p.orden {
		out[p.CoberturaDe(k)]++
	}
	return out
}

// Cuadra dice si los cubos suman el alcance.
func (p *Programa) Cuadra() bool {
	n := 0
	for _, v := range p.Cuenta() {
		n += v
	}
	return n == len(p.orden)
}

// Abiertos son los hallazgos de este programa sin cerrar, en orden estable.
func (p *Programa) Abiertos() []Hallazgo {
	var out []Hallazgo
	for _, h := range p.hallazgos {
		if !p.estaCerrado(h.ID) {
			out = append(out, h)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Hallazgos devuelve el flujo entero, cerrados incluidos. Es lo que consume el
// acta: quien encontro que y cuando, no solo en que quedo.
func (p *Programa) Hallazgos() []Hallazgo { return append([]Hallazgo(nil), p.hallazgos...) }

// Sesiones devuelve las auditorias registradas.
func (p *Programa) Sesiones() []Sesion { return append([]Sesion(nil), p.sesiones...) }

// Diferimientos devuelve los diferimientos, con quien y por que.
func (p *Programa) Diferimientos() []Diferimiento {
	return append([]Diferimiento(nil), p.diferimientos...)
}

// ParaElCicloSiguiente CALCULA el arrastre. No se apunta: se deriva de lo que
// hay, que es lo unico que impide que alguien lo escriba a mano y se quede
// viejo.
//
// La edad se acumula: una unidad que ya venia arrastrada con 2 y que este ciclo
// tampoco se audita sale con 3. Ese numero es la pregunta del auditor externo, y
// es la que hoy nadie contesta.
//
// Y LO DIFERIDO ARRASTRA IGUAL QUE LO NO AUDITADO. Es la decision que mas facil
// seria colar al reves: un diferimiento con su motivo se siente resuelto, y no
// lo esta. Diferir explica por que falta; no hace que deje de faltar. Si lo
// diferido no arrastrara, tres ciclos de diferimientos razonables darian una
// unidad que no se audita nunca y un programa que dice que va bien.
func (p *Programa) ParaElCicloSiguiente() Arrastre {
	sig := Arrastre{
		SinAuditar: map[string]int{},
		Abiertos:   map[string]int{},
		DeCiclo:    p.ciclo.Nombre,
	}
	for _, k := range p.orden {
		if p.CoberturaDe(k) == Auditada {
			continue
		}
		sig.SinAuditar[k] = p.arrastre.SinAuditar[k] + 1
	}
	for _, h := range p.Abiertos() {
		sig.Abiertos[h.ID] = p.arrastre.Abiertos[h.ID] + 1
	}
	// Los que venian abiertos del ciclo anterior y no se cerraron aqui siguen
	// contando. No estan en p.hallazgos porque se anotaron en otro programa: si
	// no se sumaran, cambiar de ciclo pondria a cero la edad de un hallazgo, que
	// es exactamente como se pierde de vista uno de tres anos.
	for id, edad := range p.arrastre.Abiertos {
		if _, ya := sig.Abiertos[id]; ya {
			continue
		}
		if p.estaCerrado(id) {
			continue
		}
		sig.Abiertos[id] = edad + 1
	}
	return sig
}

// SinAuditarDesdeHace devuelve las unidades no cubiertas con su edad en ciclos,
// contando el actual. Es la lista que se ensena.
func (p *Programa) SinAuditarDesdeHace() []UnidadPendiente {
	var out []UnidadPendiente
	for _, k := range p.orden {
		cob := p.CoberturaDe(k)
		if cob == Auditada {
			continue
		}
		out = append(out, UnidadPendiente{
			Unidad:  p.alcance[k],
			Estado:  cob,
			Ciclos:  p.arrastre.SinAuditar[k] + 1,
			DeCiclo: p.arrastre.DeCiclo,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Ciclos != out[j].Ciclos {
			return out[i].Ciclos > out[j].Ciclos // lo mas viejo primero
		}
		return out[i].Unidad.Clave() < out[j].Unidad.Clave()
	})
	return out
}

// UnidadPendiente es una unidad sin cubrir, con cuanto lleva asi.
type UnidadPendiente struct {
	Unidad  Unidad
	Estado  Cobertura
	Ciclos  int
	DeCiclo string
}

// Frase describe la pendiente SIN acusar. Va con el dato, no en una nota.
func (u UnidadPendiente) Frase() string {
	if u.Ciclos <= 1 {
		return fmt.Sprintf("%s: %s %s", u.Unidad.Clave(), u.Estado, u.Antiguedad())
	}
	return fmt.Sprintf("%s: %s, %s", u.Unidad.Clave(), u.Estado, u.Antiguedad())
}

// Antiguedad es cuanto lleva sin cubrirse, en palabras y SIN la unidad delante.
//
// Existe separada de Frase porque el acta la pone en la fila de cada unidad, al
// lado de una clave que ya esta impresa: repetirla ahi sobra, y copiar el "y van
// N ciclos" en el otro paquete seria tener la misma frase en dos sitios, que es
// como se corrige en uno solo.
//
// El caso de un ciclo dice "en este ciclo" y no "1 ciclo": el valor cero del
// mapa de arrastre da Ciclos = 1, que es la afirmacion mas pequena que se puede
// hacer, y ponerle numero la haria sonar a medida cuando es un minimo.
func (u UnidadPendiente) Antiguedad() string {
	if u.Ciclos <= 1 {
		return "en este ciclo"
	}
	return fmt.Sprintf("y van %d ciclos seguidos", u.Ciclos)
}

// ConflictoDeIndependencia es un auditor que se audita a si mismo.
//
// POR QUE ESTA COMPROBACION EXISTE Y POR QUE NO DECIDE NADA. La norma exige
// objetividad e imparcialidad del auditor, y eso es un juicio que plazum no
// puede hacer. Lo que SI es mecanico es una cosa: si quien audito una unidad es
// la misma persona que la organizacion tiene asignada como responsable de ella,
// eso es exactamente lo que la exigencia mira, y hoy no lo comprueba nadie hasta
// que llega el auditor externo.
//
// Se DICE, no se rechaza: en una empresa de veinte personas puede no haber otra
// persona, y esa es una respuesta legitima que se escribe. Lo que no es legitimo
// es que no conste.
type ConflictoDeIndependencia struct {
	Unidad  string
	Sesion  string
	Persona string
}

func (c ConflictoDeIndependencia) Frase() string {
	return fmt.Sprintf("en la sesion %s, %s audito %s, que es la unidad de la que esa misma "+
		"persona es responsable. No lo decide plazum: en una organizacion pequena puede no haber "+
		"otra persona, y esa es una respuesta legitima que hay que escribir. Lo que no vale es "+
		"que no conste", c.Sesion, c.Persona, c.Unidad)
}

// Independencia contrasta las sesiones contra quien es responsable de cada
// unidad. El mapa lo aporta la organizacion (SCIM, asignacion manual), no este
// paquete.
func (p *Programa) Independencia(responsables map[string]string) []ConflictoDeIndependencia {
	var out []ConflictoDeIndependencia
	for _, s := range p.sesiones {
		for _, u := range s.Unidades {
			r, hay := responsables[u]
			if !hay || strings.TrimSpace(r) == "" {
				continue // sin responsable asignado no hay conflicto que ver, hay otro hueco
			}
			if r == s.Auditor {
				out = append(out, ConflictoDeIndependencia{Unidad: u, Sesion: s.ID, Persona: r})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Sesion != out[j].Sesion {
			return out[i].Sesion < out[j].Sesion
		}
		return out[i].Unidad < out[j].Unidad
	})
	return out
}

func plural(n int, uno, varios string) string {
	if n == 1 {
		return uno
	}
	if !strings.Contains(varios, "%d") {
		return varios
	}
	return fmt.Sprintf(varios, n)
}

func listaCorta(xs []string) string {
	if len(xs) > 5 {
		return strings.Join(xs[:5], ", ") + ", ..."
	}
	return strings.Join(xs, ", ")
}
