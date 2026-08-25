package corpus

import (
	"errors"
	"fmt"
	"sort"

	"dutiq/nucleo/aplicabilidad"
)

// Las reglas de aplicabilidad, declaradas por el paquete.
//
// Esto es lo que faltaba para que el invariante 2 fuera cierto entero. El motor
// de Datalog estaba desde la etapa 1, pero solo se podia programar desde Go, o
// sea que las reglas del ENS vivian en un fichero de test llamado progENS. Con
// las reglas en codigo, actualizar el corpus es una release del binario en vez
// de un fichero de datos firmado, y sin fichero de datos firmado no hay
// suscripcion del corpus ni canal consultor.
//
// La sintaxis de las reglas vive en nucleo/aplicabilidad/texto.go. Aqui va lo
// que NO cabe en una linea de texto: el identificador, la cita, el agregado y
// su escala.

// Errores del linter de aplicabilidad, como centinelas. Un test que compruebe
// que una regla sin cita se rechaza tiene que poder hacerlo con errors.Is, no
// buscando una subcadena que puede aparecer en otro error del mismo camino.
var (
	ErrReglaSinID       = errors.New("regla de aplicabilidad sin id")
	ErrReglaSinCita     = errors.New("regla de aplicabilidad sin cita")
	ErrReglaRepetida    = errors.New("dos reglas de aplicabilidad con el mismo id")
	ErrReglaIlegible    = errors.New("regla de aplicabilidad que no se puede leer")
	ErrAgregadoInvalido = errors.New("agregado desconocido")
	ErrAgregadoSinVar   = errors.New("agregado sin variable sobre la que agregar")
	ErrEscalaVacia      = errors.New("escala declarada sin orden")
	ErrAgregadoSinUso   = errors.New("agregado declarado y no usado")
)

// EscalaSpec es una escala ordenada, de menor a mayor.
//
// Va como dato y no dentro de la sintaxis de la regla a proposito: una escala
// es una lista ordenada, y una lista ordenada se escribe mejor en JSON que en
// una gramatica que habria que fuzzear entera para nada.
type EscalaSpec struct {
	Nombre string   `json:"nombre"`
	Orden  []string `json:"orden"` // de menor a mayor: BAJO, MEDIO, ALTO
}

// ReglaSpec es una regla de aplicabilidad tal como la escribe un paquete.
type ReglaSpec struct {
	// ID identifica la regla dentro del paquete. Sale en las explicaciones:
	// es lo que responde "por que me aplica esto".
	ID string `json:"id"`
	// Cita es el articulo del que sale la regla. Obligatoria y sin excepcion:
	// una regla de aplicabilidad sin articulo es una opinion.
	Cita string `json:"cita"`
	// Regla en la sintaxis de superficie del dialecto. Ver
	// nucleo/aplicabilidad/texto.go para el lexico y sus tres trampas.
	Regla string `json:"regla"`
	// Agregado, cuando la regla agrega: maximo, cuenta o existe.
	Agregado string `json:"agregado,omitempty"`
	// Sobre es la variable del cuerpo que se agrega.
	Sobre string `json:"sobre,omitempty"`
	// Escala es obligatoria para maximo sobre valores no numericos. Sin ella
	// el motor no adivina: rechaza.
	Escala *EscalaSpec `json:"escala,omitempty"`
}

// Aplicabilidad es el bloque de reglas de un paquete.
type Aplicabilidad struct {
	// Exporta declara los predicados que este paquete publica al espacio
	// comun.
	//
	// AVISO: hoy es declaracion de intencion. El aislamiento por espacio de
	// nombres no esta implementado en el motor (ver nucleo/aplicabilidad), asi
	// que dos paquetes que declaren en_ambito SI colisionan. Mientras tanto, la
	// convencion es prefijar a mano los predicados propios en el fichero de
	// datos. P1.
	Exporta []string    `json:"exporta,omitempty"`
	Reglas  []ReglaSpec `json:"reglas,omitempty"`
}

var agregados = map[string]aplicabilidad.TipoAgregado{
	"maximo": aplicabilidad.Maximo,
	"cuenta": aplicabilidad.Cuenta,
	"existe": aplicabilidad.Existe,
}

// Programa traduce el bloque declarado al programa que entiende el motor.
//
// Devuelve TODOS los errores, no el primero: quien escribe un paquete de corpus
// esta escribiendo un fichero de datos y merece la lista entera de una vez, no
// una ronda de compilacion por errata.
func (p *Paquete) Programa() (aplicabilidad.Programa, []error) {
	prog := aplicabilidad.Programa{Paquete: p.URN, Exporta: p.Aplicabilidad.Exporta}
	var errs []error
	vistos := map[string]bool{}
	for i, rs := range p.Aplicabilidad.Reglas {
		donde := rs.ID
		if donde == "" {
			donde = fmt.Sprintf("regla %d", i)
		}
		if rs.ID == "" {
			errs = append(errs, fmt.Errorf("%s/%w: dale un id, es lo que sale en la "+
				"explicacion de por que aplica una obligacion", p.URN, ErrReglaSinID))
			continue
		}
		if vistos[rs.ID] {
			errs = append(errs, fmt.Errorf("%s/%s: %w. La segunda tapa a la primera en las "+
				"explicaciones", p.URN, rs.ID, ErrReglaRepetida))
			continue
		}
		vistos[rs.ID] = true
		if rs.Cita == "" {
			errs = append(errs, fmt.Errorf("%s/%s: %w. Una regla de aplicabilidad sin "+
				"articulo no se acepta: es lo que separa un motor de cumplimiento de una "+
				"hoja de calculo con opiniones", p.URN, rs.ID, ErrReglaSinCita))
			continue
		}
		r, err := aplicabilidad.ParsearRegla(rs.Regla)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s/%s: %w: %w", p.URN, donde, ErrReglaIlegible, err))
			continue
		}
		r.ID, r.Cita = rs.ID, rs.Cita
		if err := aplicarAgregado(&r, rs, p.URN); err != nil {
			errs = append(errs, err)
			continue
		}
		prog.Reglas = append(prog.Reglas, r)
	}
	// El linter del motor encima del nuestro: reglas inseguras, negacion no
	// segura, variables de cabeza sin ligar. No se duplica aqui.
	if err := prog.Validar(); err != nil {
		errs = append(errs, err)
	}
	return prog, errs
}

func aplicarAgregado(r *aplicabilidad.Regla, rs ReglaSpec, urn string) error {
	if rs.Agregado == "" {
		if rs.Sobre != "" || rs.Escala != nil {
			return fmt.Errorf("%s/%s: %w: se declara \"sobre\" o \"escala\" sin agregado, "+
				"asi que no hacen nada. Quitalos o declara el agregado", urn, rs.ID,
				ErrAgregadoSinUso)
		}
		return nil
	}
	tipo, ok := agregados[rs.Agregado]
	if !ok {
		nombres := make([]string, 0, len(agregados))
		for k := range agregados {
			nombres = append(nombres, k)
		}
		sort.Strings(nombres)
		return fmt.Errorf("%s/%s: %w %q. Los que hay son %v, y no hay mas a proposito: "+
			"son los tres que las normas necesitan de verdad", urn, rs.ID,
			ErrAgregadoInvalido, rs.Agregado, nombres)
	}
	if rs.Sobre == "" {
		return fmt.Errorf("%s/%s: %w. Di sobre que variable del cuerpo se agrega",
			urn, rs.ID, ErrAgregadoSinVar)
	}
	// La variable que se agrega tiene que estar en la CABEZA, y ahi se
	// sustituye por la variable interna del motor. El dialecto no escribe
	// _AGG: quien declara la regla pone la variable que agrega y se lee sola.
	//
	//	nivel_max(S, N) :- maneja(S, I), nivel_dimension(I, _, N)   sobre: N
	enCabeza := 0
	for i, t := range r.Cabeza.Args {
		if t.Var && t.Val == rs.Sobre {
			r.Cabeza.Args[i] = aplicabilidad.V(aplicabilidad.VarAgregada)
			enCabeza++
		}
	}
	if enCabeza == 0 {
		return fmt.Errorf("%s/%s: %w: se agrega sobre %s y %s no esta en la cabeza, asi que "+
			"el resultado del agregado no va a ninguna parte. Escribe %s(..., %s) en la "+
			"cabeza", urn, rs.ID, ErrAgregadoSinUso, rs.Sobre, rs.Sobre,
			r.Cabeza.Pred, rs.Sobre)
	}
	if enCabeza > 1 {
		return fmt.Errorf("%s/%s: %w: %s sale %d veces en la cabeza y solo se puede agregar "+
			"una vez por regla", urn, rs.ID, ErrAgregadoSinUso, rs.Sobre, enCabeza)
	}
	enCuerpo := false
	for _, a := range r.Cuerpo {
		for _, t := range a.Args {
			if t.Var && t.Val == rs.Sobre {
				enCuerpo = true
			}
		}
	}
	if !enCuerpo {
		return fmt.Errorf("%s/%s: %w: se agrega sobre %s y %s no aparece en el cuerpo. "+
			"No hay valores que agregar", urn, rs.ID, ErrAgregadoSinVar, rs.Sobre, rs.Sobre)
	}
	r.Agregado, r.SobreVar = tipo, rs.Sobre
	if rs.Escala != nil {
		if len(rs.Escala.Orden) == 0 {
			return fmt.Errorf("%s/%s: %w %q. Una escala sin orden no ordena nada",
				urn, rs.ID, ErrEscalaVacia, rs.Escala.Nombre)
		}
		r.Escala = aplicabilidad.Escala{Nombre: rs.Escala.Nombre, Orden: rs.Escala.Orden}
	}
	return nil
}

// validarAplicabilidad es la parte del linter que mira las reglas. Se llama
// desde Paquete.Validar.
//
// Ademas de que las reglas se lean, comprueba lo que solo se puede comprobar
// CRUZANDO reglas y obligaciones: que una regla que declara aplicar una
// obligacion apunte a una obligacion que existe en el paquete. Una regla que
// deriva aplica(x.art99, S) sobre una obligacion que no esta es una errata que
// no da error en ningun sitio y deja al sujeto sin la obligacion.
func (p *Paquete) validarAplicabilidad(e func(string, ...any)) {
	_, errs := p.Programa()
	for _, err := range errs {
		e("%v", err)
	}
	if len(p.Aplicabilidad.Reglas) == 0 {
		return
	}
	obligaciones := map[string]bool{}
	for _, o := range p.Obligaciones {
		obligaciones[o.ID] = true
	}
	for _, rs := range p.Aplicabilidad.Reglas {
		r, err := aplicabilidad.ParsearRegla(rs.Regla)
		if err != nil || r.Cabeza.Pred != "aplica" || len(r.Cabeza.Args) == 0 {
			continue
		}
		obj := r.Cabeza.Args[0]
		if obj.Var || obligaciones[obj.Val] {
			continue
		}
		e("%s/%s: la regla declara aplicar la obligacion %q y el paquete no la tiene. "+
			"Una errata aqui no da error en ningun sitio: deja al sujeto sin la obligacion "+
			"y nadie se entera", p.URN, rs.ID, obj.Val)
	}
}
