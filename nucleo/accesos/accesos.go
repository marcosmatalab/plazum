// Package accesos es la campana de revision de accesos (UAR) sobre una
// instantanea firmada.
//
// LO QUE ES, EN UNA FRASE: un flujo de DECISIONES INMUTABLES sobre las filas de
// una instantanea concreta, con la ley de conservacion delante y un cierre que
// no se puede firmar mientras quede algo sin explicar.
//
// TODA CONCLUSION CUELGA DEL SELLO. La campana guarda el sello de la instantanea
// (que cubre el fichero MAS quien lo subio, cuando y de que sistema dijo que
// era) y ninguna decision se admite para una clave que no este en ella. Una
// revision sobre datos que nadie puede reproducir no certifica nada: quien reciba
// el informe tiene que poder pedir el fichero, recalcular el hash, releerlo y
// encontrar las mismas filas.
//
// Y EL EMPAREJAMIENTO ES POR LA CLAVE, QUE ESTA DENTRO DEL SELLO (invariante 7):
// sistema|cuenta|permiso. Nunca por indice ni por posicion en el CSV. Reordenar
// el fichero no puede mover ni una decision, y por eso el fichero se puede
// volver a exportar sin invalidar lo decidido... siempre que el hash cuadre, que
// es justo lo que la campana comprueba.
//
// LAS DECISIONES SON HECHOS, NO UN CAMPO QUE SE EDITA. Es la doctrina de
// nucleo/incidente aplicada aqui: cambiar de opinion no borra la decision
// anterior, anade una nueva con su instante y su autor, y vale la mas reciente.
// Lo va a consumir el acta 9.3, que necesita poder decir quien decidio que y
// cuando, no solo en que quedo la cosa.
//
// DELEGAR NO ES DECIDIR. Es lo que mas facil seria colar como tercer veredicto y
// seria falso: delegar traslada la revision a otra persona y deja el acceso SIN
// REVISAR. Una campana que se cierra con delegaciones pendientes ha certificado
// que alguien miro lo que nadie miro.
//
// "SIN REVISAR" ES VISIBLE Y BLOQUEA EL CIERRE. Es la unica regla de este
// paquete que no se negocia. Una campana UAR que certifica completitud con filas
// caidas en silencio es la peor afirmacion falsa que puede hacer este producto,
// porque la firma una persona con nombre y apellidos y la ensena a un auditor.
// Lo que si existe es la EXCUSA: una linea ilegible se puede dejar fuera del
// cierre, pero con quien la excusa y por que, escrito y contado en el informe.
// La salida es "dicho", nunca "callado".
//
// Y AL REVES, PORQUE EL ERROR SIMETRICO CUESTA IGUAL: una fila sin revisar al
// cerrar la ventana NO ES UN INCUMPLIMIENTO. Es una ausencia de dato, y el
// informe lo dice con el dato al lado, no en una nota al pie.
package accesos

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/censo"
)

var (
	// ErrFilaDesconocida: la decision no casa con ninguna fila de la
	// instantanea. Es la guarda del invariante 7.
	ErrFilaDesconocida = errors.New("accesos: la decision no corresponde a ninguna fila de la instantanea")
	// ErrCampanaCerrada: una campana cerrada no admite hechos nuevos.
	ErrCampanaCerrada = errors.New("accesos: la campana ya esta cerrada")
	// ErrSinCerrar: el cierre no se puede firmar porque queda algo sin explicar.
	ErrSinCerrar = errors.New("accesos: quedan accesos sin revisar")
	// ErrDecision: la decision esta mal formada.
	ErrDecision = errors.New("accesos: la decision no esta completa")
	// ErrCampana: falta algo para abrir la campana.
	ErrCampana = errors.New("accesos: faltan datos para abrir la campana")
)

// Veredicto es el vocabulario CERRADO de lo que se puede decir de un acceso.
//
// Cerrado a proposito, igual que el de incidente: un veredicto libre convierte
// la campana en un cajon de notas, y lo que sostiene el informe es que aprobar,
// revocar y delegar signifiquen exactamente una cosa cada uno.
type Veredicto uint8

const (
	// Aprobar: el acceso sigue siendo correcto. Es una decision, y termina.
	Aprobar Veredicto = iota
	// Revocar: el acceso sobra. Tambien termina, y ademas genera trabajo fuera
	// de plazum, que este paquete no finge hacer.
	Revocar
	// Delegar: NO ES UNA DECISION. Traslada la revision a otra persona y deja el
	// acceso sin revisar. Esta en el vocabulario porque pasa siempre y porque
	// dejarlo fuera obligaria a fingirlo con una aprobacion.
	Delegar
)

var nombresDeVeredicto = [...]string{"aprobar", "revocar", "delegar"}

func (v Veredicto) String() string {
	if int(v) < len(nombresDeVeredicto) {
		return nombresDeVeredicto[v]
	}
	return fmt.Sprintf("veredicto desconocido (%d)", uint8(v))
}

func (v Veredicto) Valido() bool { return int(v) < len(nombresDeVeredicto) }

// Termina dice si el veredicto cierra la revision de ese acceso.
func (v Veredicto) Termina() bool { return v == Aprobar || v == Revocar }

// Decision es un HECHO. Se registra y no se toca: cambiar de opinion es otra
// decision con otro instante, no una edicion de esta.
type Decision struct {
	// Fila es la clave (sistema|cuenta|permiso), que esta dentro del sello.
	Fila      string
	Veredicto Veredicto
	// Quien decide, por identificador estable. El nombre no identifica.
	Quien  string
	Cuando time.Time
	Motivo string
	// A es a quien se delega. Solo tiene sentido con Delegar.
	A string
}

// Estado es el cubo en el que cae un acceso. Todo acceso esta en exactamente
// uno, y la suma tiene que dar el total: es la ley de conservacion otra vez.
type Estado string

const (
	Aprobada   Estado = "aprobada"
	Revocada   Estado = "revocada"
	Delegada   Estado = "delegada, aun sin revisar"
	SinRevisar Estado = "sin revisar"
)

// EstadosPosibles es el vocabulario entero, para que una pantalla pueda pintar
// los cubos vacios. Un cubo que solo aparece cuando tiene algo dentro es un cubo
// que nadie echa de menos.
func EstadosPosibles() []Estado {
	return []Estado{Aprobada, Revocada, Delegada, SinRevisar}
}

// Termina dice si el estado cierra la revision de ese acceso.
func (e Estado) Termina() bool { return e == Aprobada || e == Revocada }

// Excusa deja una linea ilegible fuera del cierre, CON NOMBRE Y MOTIVO.
//
// POR QUE EXISTE y por que no es una puerta trasera: sin ella, una sola linea
// ilegible de cinco mil deja la campana sin poder cerrarse nunca, y una regla
// que no se puede cumplir se acaba saltando por otro sitio. Con ella la salida
// existe y esta contada: el informe dice cuantas lineas se excusaron, quien y
// por que. La diferencia entre esto y callarlo es la unica que importa aqui.
type Excusa struct {
	Desde  int
	Hasta  int
	Quien  string
	Motivo string
	Cuando time.Time
}

// Cierre es la firma de la campana. Lleva dentro todo lo que hace falta para
// contrastarlo sin volver a tener el fichero.
type Cierre struct {
	Quien            string
	Cuando           time.Time
	Sello            string
	HashDelFichero   string
	Accesos          int
	Decididos        int
	LineasIlegibles  int
	LineasExcusadas  int
	FilasDuplicadas  int
	LineasDeDatos    int
	VeredictoPorFila map[string]Veredicto
}

// Campana es la revision de una instantanea concreta.
type Campana struct {
	id  string
	ins censo.Instantanea

	// revisores dice quien tiene que revisar cada acceso. La clave es la de la
	// fila; el valor, el identificador estable de la persona.
	revisores map[string]string

	abierta    time.Time
	decisiones []Decision
	excusas    []Excusa
	cierre     *Cierre

	// indice es el conjunto de claves de la instantanea. Existe para que
	// Registrar pueda rechazar en O(1) una decision sobre una fila que no
	// existe, que es la guarda del invariante 7.
	indice map[string]bool
}

// Abrir crea la campana. NO falla porque falten revisores: eso se pregunta con
// SinRevisor y se ensena el dia uno, igual que las figuras de escalado sin
// persona. Una campana que no se puede abrir hasta tenerlo todo asignado es una
// campana que nadie abre.
func Abrir(id string, ins censo.Instantanea, abierta time.Time, revisores map[string]string) (*Campana, error) {
	var faltan []string
	if strings.TrimSpace(id) == "" {
		faltan = append(faltan, "id de la campana")
	}
	if abierta.IsZero() {
		faltan = append(faltan, "instante de apertura (en nucleo no se lee el reloj, entra como dato)")
	}
	if ins.Sello() == "" || ins.Hash == "" {
		faltan = append(faltan, "instantanea sellada (usar censo.Tomar)")
	}
	if len(ins.Filas) == 0 {
		faltan = append(faltan, "al menos un acceso que revisar: la instantanea no trae ninguno")
	}
	if !ins.Cuadra() {
		faltan = append(faltan, "una instantanea cuadrada: esta no explica todas las lineas del "+
			"fichero, y revisar sobre un censo incompleto es certificar de mas")
	}
	if len(faltan) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrCampana, strings.Join(faltan, "; "))
	}
	c := &Campana{
		id:        id,
		ins:       ins,
		abierta:   abierta,
		revisores: map[string]string{},
		indice:    make(map[string]bool, len(ins.Filas)),
	}
	for k, v := range revisores {
		c.revisores[k] = v
	}
	for _, f := range ins.Filas {
		c.indice[f.Clave()] = true
	}
	return c, nil
}

func (c *Campana) ID() string                     { return c.id }
func (c *Campana) Sello() string                  { return c.ins.Sello() }
func (c *Campana) Instantanea() censo.Instantanea { return c.ins }
func (c *Campana) Abierta() time.Time             { return c.abierta }

// Cerrada dice si ya se firmo.
func (c *Campana) Cerrada() bool { return c.cierre != nil }

// Registrar anade un hecho.
//
// LA GUARDA DEL INVARIANTE 7 ESTA AQUI: la decision se empareja con la fila POR
// LA CLAVE, y la clave tiene que existir en la instantanea. Sin esto, una
// decision podria apuntar a un acceso que no se reviso nunca (o que existia en
// otra exportacion del mismo fichero) y el recuento de "decididos" subiria sin
// que ningun acceso real cambiara de cubo.
func (c *Campana) Registrar(d Decision) error {
	if c.Cerrada() {
		return fmt.Errorf("%w (%s, por %s): un hecho posterior al cierre convertiria el informe "+
			"firmado en una foto de algo que ya no es. Si hay que revisar de nuevo, se abre otra "+
			"campana sobre una instantanea nueva",
			ErrCampanaCerrada, c.cierre.Cuando.Format(time.RFC3339), c.cierre.Quien)
	}
	var faltan []string
	if !d.Veredicto.Valido() {
		faltan = append(faltan, fmt.Sprintf("veredicto valido (%s)", strings.Join(nombresDeVeredicto[:], ", ")))
	}
	if strings.TrimSpace(d.Quien) == "" {
		faltan = append(faltan, "quien decide (identificador estable, no el nombre)")
	}
	if d.Cuando.IsZero() {
		faltan = append(faltan, "cuando")
	}
	if d.Veredicto == Revocar && strings.TrimSpace(d.Motivo) == "" {
		faltan = append(faltan, "motivo: revocar un acceso le quita a alguien una herramienta de "+
			"trabajo, y quien lo sufra va a preguntar por que")
	}
	if d.Veredicto == Delegar && strings.TrimSpace(d.A) == "" {
		faltan = append(faltan, "a quien se delega: una delegacion sin destinatario deja el acceso "+
			"sin revisar y sin nadie que lo revise, que es peor que no delegar")
	}
	if len(faltan) > 0 {
		return fmt.Errorf("%w: %s", ErrDecision, strings.Join(faltan, "; "))
	}
	if !c.indice[d.Fila] {
		return fmt.Errorf("%w: %q.\n"+
			"  La campana revisa la instantanea de sello %s, y esa clave no esta en ella.\n"+
			"  Se empareja por sistema|cuenta|permiso, que va DENTRO de lo sellado, nunca por la "+
			"posicion en el fichero: si el CSV se reordena, las decisiones siguen donde estaban.\n"+
			"  Arreglo: comprobar que la decision es de esta campana y no de una exportacion "+
			"distinta del mismo sistema",
			ErrFilaDesconocida, d.Fila, corto(c.ins.Sello()))
	}
	c.decisiones = append(c.decisiones, d)
	// Delegar ademas MUEVE al revisor: si no lo hiciera, la delegacion seria una
	// nota y el acceso seguiria esperando a quien ya dijo que no era suyo.
	if d.Veredicto == Delegar {
		c.revisores[d.Fila] = d.A
	}
	return nil
}

// Excusar deja una linea ilegible fuera del cierre, dicho y contado.
func (c *Campana) Excusar(e Excusa) error {
	if c.Cerrada() {
		return fmt.Errorf("%w: no se excusa nada despues de firmar", ErrCampanaCerrada)
	}
	if strings.TrimSpace(e.Quien) == "" || strings.TrimSpace(e.Motivo) == "" || e.Cuando.IsZero() {
		return fmt.Errorf("%w: una excusa sin quien, sin por que o sin cuando es exactamente lo "+
			"que esto existe para no permitir: una linea que desaparece del recuento y nadie "+
			"responde de ella", ErrDecision)
	}
	if e.Hasta < e.Desde {
		e.Hasta = e.Desde
	}
	c.excusas = append(c.excusas, e)
	return nil
}

// Decisiones devuelve el flujo entero, en orden de registro. Es lo que consume
// el acta: quien decidio que y cuando, no solo en que quedo.
func (c *Campana) Decisiones() []Decision {
	return append([]Decision(nil), c.decisiones...)
}

// Vigente es la decision que manda hoy sobre un acceso: la mas reciente por
// instante. Los empates SE DECLARAN, no se resuelven: con dos decisiones
// distintas en el mismo instante, esto dice que hay empate y no elige.
func (c *Campana) Vigente(clave string) (Decision, bool, error) {
	var mejor Decision
	hay := false
	empate := false
	for _, d := range c.decisiones {
		if d.Fila != clave {
			continue
		}
		switch {
		case !hay || d.Cuando.After(mejor.Cuando):
			mejor, hay, empate = d, true, false
		case d.Cuando.Equal(mejor.Cuando) && d != mejor:
			empate = true
		}
	}
	if empate {
		return mejor, true, fmt.Errorf("el acceso %q tiene dos decisiones distintas en el mismo "+
			"instante (%s). No se elige una: quien firme la campana tiene que saber que aqui hubo "+
			"dos manos", clave, mejor.Cuando.Format(time.RFC3339))
	}
	return mejor, hay, nil
}

// EstadoDe dice en que cubo esta un acceso.
func (c *Campana) EstadoDe(clave string) Estado {
	d, hay, _ := c.Vigente(clave)
	if !hay {
		return SinRevisar
	}
	switch d.Veredicto {
	case Aprobar:
		return Aprobada
	case Revocar:
		return Revocada
	default:
		return Delegada
	}
}

// Cuenta es la LEY DE CONSERVACION de la campana. Todo acceso de la instantanea
// cae en exactamente un cubo, y los cubos vacios tambien salen.
func (c *Campana) Cuenta() map[Estado]int {
	out := map[Estado]int{}
	for _, e := range EstadosPosibles() {
		out[e] = 0
	}
	for _, f := range c.ins.Filas {
		out[c.EstadoDe(f.Clave())]++
	}
	return out
}

// Falta es un acceso que no tiene a quien preguntarle.
type Falta struct {
	Fila   string
	Rotulo string
}

// SinRevisor ES LA COMPROBACION DEL DIA UNO, la misma que
// escalado.FigurasSinPersona: un acceso sin revisor asignado no se descubre el
// dia del cierre, se ve al abrir la campana. Un aviso que llega cuando ya no se
// puede hacer nada no es un aviso.
func (c *Campana) SinRevisor() []Falta {
	var out []Falta
	for _, f := range c.ins.Filas {
		if strings.TrimSpace(c.revisores[f.Clave()]) != "" {
			continue
		}
		out = append(out, Falta{Fila: f.Clave(), Rotulo: f.Rotulo})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Fila < out[j].Fila })
	return out
}

// RevisorDe dice quien tiene que revisar un acceso.
func (c *Campana) RevisorDe(clave string) (string, bool) {
	p, ok := c.revisores[clave]
	return p, ok && strings.TrimSpace(p) != ""
}

// lineasExcusadas cuenta las lineas cubiertas por excusas, sin contar dos veces
// las que se solapan.
func (c *Campana) lineasExcusadas() map[int]bool {
	out := map[int]bool{}
	for _, e := range c.excusas {
		for l := e.Desde; l <= e.Hasta; l++ {
			out[l] = true
		}
	}
	return out
}

// lineasIlegiblesSinExcusar son las que aun bloquean el cierre.
func (c *Campana) lineasIlegiblesSinExcusar() []int {
	excusadas := c.lineasExcusadas()
	var out []int
	for _, il := range c.ins.Ilegibles {
		for l := il.Desde; l <= il.Hasta; l++ {
			if !excusadas[l] {
				out = append(out, l)
			}
		}
	}
	sort.Ints(out)
	return out
}

// Cerrar firma la campana, o dice EXACTAMENTE que lo impide.
func (c *Campana) Cerrar(quien string, cuando time.Time) (Cierre, error) {
	if c.Cerrada() {
		return *c.cierre, fmt.Errorf("%w el %s por %s", ErrCampanaCerrada,
			c.cierre.Cuando.Format(time.RFC3339), c.cierre.Quien)
	}
	if strings.TrimSpace(quien) == "" || cuando.IsZero() {
		return Cierre{}, fmt.Errorf("%w: un cierre sin quien firma y sin cuando no vale como "+
			"certificacion de nada", ErrCampana)
	}

	cuenta := c.Cuenta()
	pendientes := cuenta[SinRevisar] + cuenta[Delegada]
	ilegibles := c.lineasIlegiblesSinExcusar()
	if pendientes > 0 || len(ilegibles) > 0 {
		var partes []string
		if cuenta[SinRevisar] > 0 {
			partes = append(partes, fmt.Sprintf("%d sin revisar", cuenta[SinRevisar]))
		}
		if cuenta[Delegada] > 0 {
			partes = append(partes, fmt.Sprintf("%d delegados y todavia sin decidir (delegar "+
				"traslada la revision, no la termina)", cuenta[Delegada]))
		}
		if len(ilegibles) > 0 {
			partes = append(partes, fmt.Sprintf("%d lineas del fichero que no se pudieron leer "+
				"(%s) y que nadie ha excusado", len(ilegibles), listaCorta(ilegibles)))
		}
		return Cierre{}, fmt.Errorf("%w: %s.\n"+
			"  Una campana que se firma con esto pendiente afirma que se reviso lo que nadie "+
			"reviso, y la firma una persona con nombre y apellidos delante de un auditor.\n"+
			"  Salidas, y las dos dejan rastro: decidir lo que falta, o excusar por escrito las "+
			"lineas ilegibles con Excusar (quien y por que, y sale en el informe)",
			ErrSinCerrar, strings.Join(partes, "; "))
	}

	cierre := Cierre{
		Quien:            quien,
		Cuando:           cuando,
		Sello:            c.ins.Sello(),
		HashDelFichero:   c.ins.Hash,
		Accesos:          len(c.ins.Filas),
		Decididos:        cuenta[Aprobada] + cuenta[Revocada],
		LineasIlegibles:  c.ins.LineasCubiertas() - len(c.ins.Filas) - len(c.ins.Duplicadas),
		LineasExcusadas:  len(c.lineasExcusadas()),
		FilasDuplicadas:  len(c.ins.Duplicadas),
		LineasDeDatos:    c.ins.LineasDeDatos,
		VeredictoPorFila: map[string]Veredicto{},
	}
	for _, f := range c.ins.Filas {
		d, hay, _ := c.Vigente(f.Clave())
		if hay {
			cierre.VeredictoPorFila[f.Clave()] = d.Veredicto
		}
	}
	c.cierre = &cierre
	return cierre, nil
}

func corto(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	return h
}

func listaCorta(ns []int) string {
	if len(ns) > 6 {
		ns = ns[:6]
		return strings.Trim(strings.Join(strings.Fields(fmt.Sprint(ns)), ", "), "[]") + ", ..."
	}
	return strings.Trim(strings.Join(strings.Fields(fmt.Sprint(ns)), ", "), "[]")
}
