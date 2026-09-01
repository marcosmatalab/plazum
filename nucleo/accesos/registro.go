package accesos

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/censo"
	"github.com/marcosmatalab/plazum/nucleo/ledger"
)

// LA PERSISTENCIA DE UNA CAMPANA, Y LO QUE DELIBERADAMENTE NO SE GUARDA.
//
// Una campana de accesos vive semanas: se abre, la revisan cuatro personas a lo
// largo de un mes y se cierra. Sin persistencia el motor es una biblioteca, no
// un producto. Pero lo que hay que persistir NO es el censo: es el FLUJO DE
// HECHOS, que es lo unico que no se puede volver a derivar.
//
// LA DECISION, Y SU PRECIO DICHO EN VOZ ALTA. Las filas del censo (identificador
// de cuenta, permiso y rotulo de una persona) **no se guardan en ningun sitio**.
// Una campana se reconstruye desde DOS cosas: el fichero CSV, que lo tiene quien
// lo subio, y el ledger, que tiene los hechos. El precio es que hay que traer el
// fichero en cada operacion, y es incomodo. Lo que se compra es que plazum no
// tenga a la plantilla de nadie en disco, y que el ledger -- que es la pieza que
// VIAJA, se copia, se ancla y se ensena a un auditor -- no lleve un solo nombre.
//
// Y LA PRECISION QUE FALTARIA SI NO SE DIJERA: la huella de fila es un
// SEUDONIMO, no un anonimato. Quien pueda adivinar identificadores de cuenta
// puede confirmar una conjetura probandola contra la huella. Lo que impide es
// que quien lea el ledger se lleve la lista; no lo que un atacante que ya
// sospecha algo pueda comprobar. Decirlo de mas seria vender una garantia que
// esto no da.
//
// EL EMPAREJAMIENTO, que es la pregunta de siempre (invariante 7). Los hechos
// del ledger apuntan a las filas por HuellaDeFila(sello, clave), y las dos
// mitades estan dentro de lo sellado: la clave es sistema|cuenta|permiso y el
// sello cubre el fichero mas como se leyo. Nunca por indice ni por posicion en
// el CSV: reordenar el fichero no mueve ninguna decision. Y si el fichero
// cambia, el sello cambia, ninguna huella casa y la campana **se niega a
// reabrirse** en vez de aplicar decisiones viejas a filas nuevas, que es la
// forma exacta en que una revocacion acabaria firmada sobre la persona que no
// era.

var (
	// ErrSelloDistinto: el fichero no es el que se reviso.
	ErrSelloDistinto = errors.New("accesos: el fichero no es el mismo sobre el que se abrio la campana")
	// ErrSinIngesta: el ledger no tiene la apertura de esta campana.
	ErrSinIngesta = errors.New("accesos: en el ledger no consta la apertura de esta campana")
	// ErrHechoHuerfano: un hecho apunta a una fila que no esta en el censo.
	ErrHechoHuerfano = errors.New("accesos: un hecho del ledger no casa con ninguna fila del censo")
)

// Los tipos de entrada del ledger que usa una campana. El vocabulario es el del
// ledger (observacion, afirmacion, decision, excepcion) y encaja sin forzarlo.
const (
	TipoApertura = "observacion"
	TipoDecision = "decision"
	TipoExcusa   = "excepcion"
	TipoCierre   = "afirmacion"
)

// Sujeto es el asunto de las entradas de una campana. Se compone aqui para que
// no haya dos formas de escribirlo, que es como se pierden entradas al leer.
func Sujeto(campana string) string { return "accesos/" + campana }

// HuellaDeFila es el seudonimo de una fila dentro de UNA campana.
//
// Va salada con el sello, asi que la misma cuenta en dos campanas da dos
// huellas: quien tenga los dos ledgers no puede cruzarlos para seguir a una
// persona entre revisiones. Con el fichero delante, cualquiera recalcula la
// huella y comprueba que la decision es de esa fila.
func HuellaDeFila(sello, clave string) string {
	h := sha256.New()
	fmt.Fprintf(h, "%d:%s|%d:%s", len(sello), sello, len(clave), clave)
	return hex.EncodeToString(h.Sum(nil))
}

// CargaDeApertura es lo que se anota al subir el fichero. Recuentos, hashes y
// quien lo subio: ni un nombre, ni un identificador de cuenta, ni un permiso.
type CargaDeApertura struct {
	Campana         string `json:"campana"`
	Hash            string `json:"hash_fichero"`
	Sello           string `json:"sello"`
	Sistema         string `json:"sistema"`
	Fuente          string `json:"fuente"`
	Retencion       string `json:"retencion"`
	Accesos         int    `json:"accesos"`
	LineasDeDatos   int    `json:"lineas_de_datos"`
	LineasIlegibles int    `json:"lineas_ilegibles"`
	FilasRepetidas  int    `json:"filas_repetidas"`
	Codificacion    string `json:"codificacion"`
	Separador       string `json:"separador"`
}

// CargaDeDecision es un hecho de revision.
type CargaDeDecision struct {
	Campana   string `json:"campana"`
	Sello     string `json:"sello"`
	Huella    string `json:"huella_de_fila"`
	Veredicto string `json:"veredicto"`
	Motivo    string `json:"motivo,omitempty"`
	A         string `json:"delegado_en,omitempty"`
}

// CargaDeExcusa deja una linea ilegible fuera del cierre.
type CargaDeExcusa struct {
	Campana string `json:"campana"`
	Sello   string `json:"sello"`
	Desde   int    `json:"desde"`
	Hasta   int    `json:"hasta"`
	Motivo  string `json:"motivo"`
}

// CargaDeCierre es la firma de la campana.
type CargaDeCierre struct {
	Campana         string `json:"campana"`
	Sello           string `json:"sello"`
	Hash            string `json:"hash_fichero"`
	Accesos         int    `json:"accesos"`
	Decididos       int    `json:"decididos"`
	LineasExcusadas int    `json:"lineas_excusadas"`
	FilasRepetidas  int    `json:"filas_repetidas"`
}

// AperturaComoEntrada arma la entrada de ledger de la subida.
func AperturaComoEntrada(ins censo.Instantanea, campana string) (ledger.Entrada, error) {
	carga, err := json.Marshal(CargaDeApertura{
		Campana:         campana,
		Hash:            ins.Hash,
		Sello:           ins.Sello(),
		Sistema:         ins.Sistema,
		Fuente:          ins.Fuente,
		Retencion:       ins.Retencion,
		Accesos:         len(ins.Filas),
		LineasDeDatos:   ins.LineasDeDatos,
		LineasIlegibles: ins.LineasCubiertas() - len(ins.Filas) - len(ins.Duplicadas),
		FilasRepetidas:  len(ins.Duplicadas),
		Codificacion:    ins.Notas.Codificacion,
		Separador:       ins.Notas.Separador,
	})
	if err != nil {
		return ledger.Entrada{}, err
	}
	return ledger.Entrada{
		Instante: ins.Tomada,
		Tipo:     TipoApertura,
		Sujeto:   Sujeto(campana),
		Actor:    ins.Quien,
		Carga:    carga,
	}, nil
}

// DecisionComoEntrada arma la entrada de ledger de una decision.
//
// La decision NO lleva la clave de la fila, lleva su huella. Es la unica
// diferencia entre un ledger que se puede ensenar y uno que hay que custodiar.
func DecisionComoEntrada(d Decision, sello, campana string) (ledger.Entrada, error) {
	if !d.Veredicto.Valido() {
		return ledger.Entrada{}, fmt.Errorf("%w: veredicto %d", ErrDecision, d.Veredicto)
	}
	carga, err := json.Marshal(CargaDeDecision{
		Campana:   campana,
		Sello:     sello,
		Huella:    HuellaDeFila(sello, d.Fila),
		Veredicto: d.Veredicto.String(),
		Motivo:    d.Motivo,
		A:         d.A,
	})
	if err != nil {
		return ledger.Entrada{}, err
	}
	return ledger.Entrada{
		Instante: d.Cuando,
		Tipo:     TipoDecision,
		Sujeto:   Sujeto(campana),
		Actor:    d.Quien,
		Carga:    carga,
	}, nil
}

// ExcusaComoEntrada arma la entrada de ledger de una excusa.
func ExcusaComoEntrada(e Excusa, sello, campana string) (ledger.Entrada, error) {
	carga, err := json.Marshal(CargaDeExcusa{
		Campana: campana, Sello: sello, Desde: e.Desde, Hasta: e.Hasta, Motivo: e.Motivo,
	})
	if err != nil {
		return ledger.Entrada{}, err
	}
	return ledger.Entrada{
		Instante: e.Cuando, Tipo: TipoExcusa, Sujeto: Sujeto(campana), Actor: e.Quien, Carga: carga,
	}, nil
}

// CierreComoEntrada arma la entrada de ledger del cierre.
func CierreComoEntrada(c Cierre, campana string) (ledger.Entrada, error) {
	carga, err := json.Marshal(CargaDeCierre{
		Campana: campana, Sello: c.Sello, Hash: c.HashDelFichero,
		Accesos: c.Accesos, Decididos: c.Decididos,
		LineasExcusadas: c.LineasExcusadas, FilasRepetidas: c.FilasDuplicadas,
	})
	if err != nil {
		return ledger.Entrada{}, err
	}
	return ledger.Entrada{
		Instante: c.Cuando, Tipo: TipoCierre, Sujeto: Sujeto(campana), Actor: c.Quien, Carga: carga,
	}, nil
}

// Reconstruir levanta la campana desde el censo y el ledger.
//
// EL ORDEN DE LAS COMPROBACIONES ES EL DISENO. Primero se exige que conste la
// apertura, despues que su sello sea EL MISMO que el del fichero que se acaba de
// leer, y solo entonces se reproducen los hechos. Al reves, unas decisiones se
// aplicarian a filas de otro fichero antes de que nadie mirara si es el mismo
// fichero, y una revocacion acabaria firmada sobre la persona que no era.
func Reconstruir(campana string, ins censo.Instantanea, l ledger.Ledger,
	revisores map[string]string) (*Campana, error) {

	sujeto := Sujeto(campana)
	var apertura *CargaDeApertura
	var abierta time.Time
	for _, e := range l.Entradas {
		if e.Sujeto != sujeto || e.Tipo != TipoApertura {
			continue
		}
		var a CargaDeApertura
		if err := json.Unmarshal(e.Carga, &a); err != nil {
			return nil, fmt.Errorf("la apertura de %q no se puede leer: %w", campana, err)
		}
		apertura, abierta = &a, e.Instante
		break
	}
	if apertura == nil {
		return nil, fmt.Errorf("%w: %q.\n"+
			"  Campanas que si constan: %s.\n"+
			"  Arreglo: si es una campana nueva, se abre subiendo el fichero; si es antigua, "+
			"comprobar que el ledger es el que la tiene", ErrSinIngesta, campana,
			listaDeCampanas(l))
	}
	if apertura.Sello != ins.Sello() {
		return nil, fmt.Errorf("%w.\n"+
			"  El ledger dice que la campana %q se abrio sobre el sello %s y el fichero de ahora "+
			"da %s.\n"+
			"  No se aplican decisiones viejas a filas nuevas: una revocacion decidida sobre la "+
			"fila 12 de aquel fichero acabaria firmada sobre quien este hoy en la 12.\n"+
			"  Arreglo: traer el fichero exacto que se subio (su sha256 era %s), o abrir una "+
			"campana nueva sobre el fichero de ahora",
			ErrSelloDistinto, campana, corto(apertura.Sello), corto(ins.Sello()), corto(apertura.Hash))
	}

	c, err := Abrir(campana, ins, abierta, revisores)
	if err != nil {
		return nil, err
	}

	// El indice inverso: de huella a clave. Se construye desde el CENSO, no
	// desde el ledger, porque el censo es quien sabe que filas existen.
	porHuella := make(map[string]string, len(ins.Filas))
	for _, f := range ins.Filas {
		porHuella[HuellaDeFila(apertura.Sello, f.Clave())] = f.Clave()
	}

	var cierre *ledger.Entrada
	for i, e := range l.Entradas {
		if e.Sujeto != sujeto {
			continue
		}
		switch e.Tipo {
		case TipoDecision:
			var cd CargaDeDecision
			if err := json.Unmarshal(e.Carga, &cd); err != nil {
				return nil, fmt.Errorf("la decision de la entrada %d no se puede leer: %w", e.Seq, err)
			}
			clave, hay := porHuella[cd.Huella]
			if !hay {
				return nil, fmt.Errorf("%w: la entrada %d apunta a la huella %s.\n"+
					"  El sello cuadra, asi que el fichero es el mismo y esa huella tendria que "+
					"casar: o la entrada es de otra campana con el mismo nombre, o el ledger se "+
					"ha mezclado. No se descarta en silencio, porque una decision que desaparece "+
					"deja un acceso contado como sin revisar y a alguien firmando que lo reviso",
					ErrHechoHuerfano, e.Seq, corto(cd.Huella))
			}
			v, err := VeredictoDe(cd.Veredicto)
			if err != nil {
				return nil, fmt.Errorf("entrada %d: %w", e.Seq, err)
			}
			if err := c.Registrar(Decision{
				Fila: clave, Veredicto: v, Quien: e.Actor, Cuando: e.Instante,
				Motivo: cd.Motivo, A: cd.A,
			}); err != nil {
				return nil, fmt.Errorf("reproduciendo la entrada %d: %w", e.Seq, err)
			}
		case TipoExcusa:
			var ce CargaDeExcusa
			if err := json.Unmarshal(e.Carga, &ce); err != nil {
				return nil, fmt.Errorf("la excusa de la entrada %d no se puede leer: %w", e.Seq, err)
			}
			if err := c.Excusar(Excusa{
				Desde: ce.Desde, Hasta: ce.Hasta, Quien: e.Actor, Motivo: ce.Motivo,
				Cuando: e.Instante,
			}); err != nil {
				return nil, fmt.Errorf("reproduciendo la entrada %d: %w", e.Seq, err)
			}
		case TipoCierre:
			ent := l.Entradas[i]
			cierre = &ent
		}
	}

	// EL CIERRE SE REPRODUCE AL FINAL, y no cuando aparece.
	//
	// Si se aplicara en su turno, las decisiones posteriores en el fichero
	// chocarian con ErrCampanaCerrada y la reconstruccion fallaria por un orden
	// de lectura, no por un problema real. Y hay una razon mas fuerte: el cierre
	// tiene que volver a PASAR SUS PROPIAS COMPROBACIONES. Si alguien edito el
	// ledger para borrar una decision, el cierre ya no cuadra y esto se niega a
	// dar por cerrada una campana que hoy no se podria cerrar.
	if cierre != nil {
		if _, err := c.Cerrar(cierre.Actor, cierre.Instante); err != nil {
			return nil, fmt.Errorf("el ledger dice que la campana se cerro el %s y hoy no se "+
				"podria cerrar: %w.\n"+
				"  Es lo que se esperaria si al ledger le falta una decision. No se da por "+
				"cerrada: un cierre que no se sostiene con los hechos que hay delante no es un "+
				"cierre, es una afirmacion sin nada detras",
				cierre.Instante.Format(time.RFC3339), err)
		}
	}
	return c, nil
}

// VeredictoDe traduce el nombre que viaja en el ledger.
//
// El vocabulario es CERRADO y por eso esto puede fallar: un veredicto que no se
// reconoce no se convierte en el cero (que seria "aprobar", o sea el permisivo)
// sino en un error. Es el invariante 8 en una frontera de lectura.
func VeredictoDe(nombre string) (Veredicto, error) {
	for i, n := range nombresDeVeredicto {
		if n == nombre {
			return Veredicto(i), nil
		}
	}
	return 0, fmt.Errorf("%w: veredicto %q no reconocido (%s). No se toma el valor por defecto: "+
		"el cero es \"aprobar\", que es el permisivo, y aprobar por no entender una palabra es "+
		"exactamente lo que no puede pasar",
		ErrDecision, nombre, strings.Join(nombresDeVeredicto[:], ", "))
}

func listaDeCampanas(l ledger.Ledger) string {
	vistas := map[string]bool{}
	for _, e := range l.Entradas {
		if strings.HasPrefix(e.Sujeto, "accesos/") && e.Tipo == TipoApertura {
			vistas[strings.TrimPrefix(e.Sujeto, "accesos/")] = true
		}
	}
	if len(vistas) == 0 {
		return "ninguna"
	}
	out := make([]string, 0, len(vistas))
	for k := range vistas {
		out = append(out, k)
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}
