package corpus

import (
	"errors"
	"fmt"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// EL BLOQUE `pruebas`: EL ESLABON QUE FALTABA, Y ES DE DATOS (D-22).
//
// # Que enlaza, y por que no podia vivir en Go
//
// `puertos.Recoleccion`, `estado.Observacion`, `estado.Prueba` y
// `estado.Calcular` estan construidos desde la etapa 1. Lo que no habia era
// forma de que un paquete dijera **la obligacion X se comprueba con la prueba Y
// sobre el recurso Z, con TTL de 30 dias y SLA de 7**, asi que `estado.Calcular`
// lo llamaba un solo sitio y `puertos.Recoleccion` no lo implementaba nadie.
//
// Por el invariante 2 ese enlace no puede vivir en Go: una prueba escrita en
// codigo convierte el siguiente marco que la necesite en un cambio de producto.
// Aqui va el TIPO y su linter; el CONTENIDO va en cada `paquete.json`.
//
// # LA OBLIGACION Y LA PRUEBA NO SON LO MISMO, y confundirlas es lo que costo
//
// La obligacion dice **que exige la ley**. La prueba dice **que se mira para
// saber si consta**. Un articulo que obliga a «revisar los accesos al menos
// semestralmente» es una obligacion; «el 100 % de las cuentas del directorio
// tienen una decision de revision con menos de 183 dias» es su prueba.
//
// # POR QUE LA PRUEBA VA EN EL NIVEL DEL PAQUETE Y NO DENTRO DE LA OBLIGACION
//
// Porque una obligacion puede tener varias pruebas (el mismo articulo se
// comprueba sobre `Persona` y sobre `Sistema`) y porque asi la prueba NOMBRA a
// su obligacion en vez de heredarla de su posicion. Es el invariante 7: el
// emparejamiento va por un identificador que esta DENTRO de lo firmado, no por
// donde este escrito. Reordenar las obligaciones de un paquete no mueve ni una
// prueba.
//
// # EL PREDICADO ES OBLIGATORIO, Y ESO CIERRA LA MITAD QUE FALTABA DEL 13
//
// El invariante 13 dice que un recolector entrega hechos y no veredictos, y deja
// `estado.Observacion.Satisfecho` en pie con una condicion escrita: que sea el
// resultado de un **predicado mecanico declarado en la prueba**. Sin predicado
// obligatorio, esa condicion es prosa. Con el, un `Satisfecho` a true siempre
// tiene detras un texto que dice QUE se evaluo, y quien lea el expediente puede
// ir a mirarlo. Un bool sin predicado declarado es un veredicto disfrazado de
// dato.
type Prueba struct {
	// ID identifica la prueba dentro del paquete. Unico.
	ID string `json:"id"`
	// Obligacion es el id de la obligacion que esta prueba comprueba. Tiene que
	// existir en ESTE paquete: una prueba que apunta fuera es un enlace que
	// nadie puede seguir.
	Obligacion string `json:"obligacion"`
	// Recurso es sobre que se observa. Tiene que ser uno de los que declara la
	// obligacion en su campo `recursos`: observar algo que la obligacion nunca
	// dijo necesitar es inventarse el alcance de la comprobacion.
	Recurso TipoRecurso `json:"recurso"`
	// TTL es la frescura maxima admitida, en el subconjunto ISO-8601 de
	// `ventana.ParseDuracion` (P30D, P6M, PT24H).
	//
	// OBLIGATORIO Y MAYOR QUE CERO. El vacio NO significa «no caduca»: es el
	// cero permisivo del invariante 8, y aqui significaria que una observacion
	// de hace tres anos vale igual que la de esta manana, que es exactamente lo
	// que un auditor no acepta. Si una comprobacion de verdad no caduca nunca,
	// eso se dice con `pass_por_defecto`, que es otra cosa y se lee distinto.
	TTL string `json:"ttl"`
	// SLA es el plazo de remediacion antes de escalar, mismo formato.
	//
	// OBLIGATORIO Y NO MAYOR QUE EL TTL. Un SLA mas largo que la frescura es un
	// plazo que no se puede agotar: la observacion se queda obsoleta antes de
	// que venza, `estado.Calcular` pasa a Obsoleto y el FailVencido que tenia
	// que escalar no llega nunca. Es una escalada inalcanzable escrita como si
	// fuera un plazo generoso.
	SLA string `json:"sla"`
	// Activa es el rollout: antes de esta fecha la prueba no genera historico.
	// Opcional; si esta, tiene que ser una fecha legible.
	Activa string `json:"activa,omitempty"`
	// PassPorDefecto dice que el proveedor lo garantiza y no es configurable.
	PassPorDefecto bool `json:"pass_por_defecto,omitempty"`
	// Predicado es QUE se evalua, escrito para que una persona lo pueda leer y
	// contrastar. Obligatorio: ver el bloque de arriba.
	Predicado string `json:"predicado"`
	// Cierra dice si esta prueba en verde CIERRA la obligacion, o si solo
	// APORTA a un aspecto de ella.
	//
	// # POR QUE HACE FALTA, con el cardinal que lo obliga
	//
	// De las 133 obligaciones observables del corpus (07-09-2026), 99 lo son por
	// CLASE PRIMARIA y 34 por FACETA. Y no significan lo mismo:
	//
	//	clase primaria observable   lo que la norma exige ES observable. Una
	//	                            prueba en verde puede cerrarla.
	//	faceta observable           la obligacion es otra cosa (documental,
	//	                            procedimental) y ademas tiene un aspecto
	//	                            observable. Una prueba en verde aporta a ese
	//	                            aspecto y NO cierra la obligacion: su
	//	                            descargo de verdad es un documento que puede
	//	                            no existir.
	//
	// Sin este campo, la pantalla de controles pintaria «consta» sobre una
	// obligacion documental porque su aspecto observable dio verde. Eso es
	// ABSOLVER DE MAS CON CARA DE DATO, que es el error simetrico de acusar en
	// falso y el que nadie mira: quien lo lea deja de buscar el documento.
	//
	// OBLIGATORIO Y SIN VALOR CERO UTIL. `false` es una respuesta legitima y
	// frecuente, asi que no se puede distinguir de «no lo he dicho» mirando el
	// bool: se exige que el campo ESTE, y por eso es un puntero. El invariante 8
	// con su forma de JSON: un bool ausente y un bool a false son dos cosas
	// distintas y solo una es una decision.
	//
	// Se hace AHORA porque cuesta un campo y una prueba. Despues de la campana
	// de la etapa 3 costaria 133.
	Cierra *bool `json:"cierra"`
}

// Los centinelas del bloque. Uno por cada forma de romperlo, para que quien
// llame pueda distinguirlas sin leer una cadena.
var (
	ErrPruebaSinID           = errors.New("prueba sin id")
	ErrPruebaRepetida        = errors.New("dos pruebas con el mismo id")
	ErrPruebaSinObligacion   = errors.New("prueba sin obligacion")
	ErrPruebaHuerfana        = errors.New("prueba sobre una obligacion que no existe")
	ErrPruebaNoObservable    = errors.New("prueba sobre una obligacion que no es observable")
	ErrPruebaSinRecurso      = errors.New("prueba sin recurso")
	ErrPruebaRecursoAjeno    = errors.New("prueba sobre un recurso que su obligacion no declara")
	ErrPruebaTTLInvalido     = errors.New("prueba con ttl ausente o ilegible")
	ErrPruebaSLAInvalido     = errors.New("prueba con sla ausente o ilegible")
	ErrPruebaSLAMayorQueTTL  = errors.New("prueba con sla mayor que su ttl")
	ErrPruebaActivaIlegible  = errors.New("prueba con fecha activa ilegible")
	ErrPruebaSinPredicado    = errors.New("prueba sin predicado")
	ErrPruebaSinCierra       = errors.New("prueba sin decir si cierra la obligacion")
	ErrPruebaCierraUnaFaceta = errors.New(
		"prueba que dice cerrar una obligacion que solo es observable por faceta")
)

// EsObservable dice si de esta obligacion se puede comprobar algo por
// recoleccion.
//
// MIRA LOS DOS CAMPOS, y esa es la mitad que importa: `observable` puede ser la
// clase PRIMARIA de la obligacion o una de sus FACETAS, el linter valida los dos
// contra el mismo vocabulario, y los conjuntos son disjuntos. Contar uno solo
// publica un techo cuatro veces mas bajo que el real: medido el 07-09-2026 sobre
// el corpus, 99 por clase primaria y 34 por faceta, 133 en total sobre 549.
func (o Obligacion) EsObservable() bool {
	return o.ObservablePorClase() || o.ObservablePorFaceta()
}

// ObservablePorClase dice si lo que la norma exige ES observable. Una prueba en
// verde sobre una de estas puede CERRAR la obligacion.
func (o Obligacion) ObservablePorClase() bool { return o.ClaseE2E == "observable" }

// ObservablePorFaceta dice si la obligacion es otra cosa y ADEMAS tiene un
// aspecto observable. Una prueba en verde sobre una de estas aporta a ese
// aspecto y no cierra nada: el descargo de verdad sigue siendo el documento o el
// procedimiento que la clase primaria exige.
func (o Obligacion) ObservablePorFaceta() bool {
	if o.ObservablePorClase() {
		return false
	}
	for _, f := range o.Facetas {
		if f == "observable" {
			return true
		}
	}
	return false
}

// SinPrueba son las obligaciones observables que nadie comprueba todavia,
// PARTIDAS EN SUS DOS MITADES.
//
// La union no vale y por eso no se publica: las dos mitades no cuestan lo mismo
// ni significan lo mismo. Una obligacion observable POR CLASE sin prueba es una
// comprobacion que falta entera; una observable POR FACETA sin prueba es un
// aspecto sin cubrir de algo que se cierra por otra via. Un solo numero que
// sume las dos hace que la campana de la etapa 3 parezca mas homogenea de lo
// que es.
type SinPrueba struct {
	PorClase  []string
	PorFaceta []string
}

// Total es la suma, para quien de verdad quiera un solo numero. Existe para que
// nadie tenga que sumarlas a mano y se equivoque, no para sustituir a las dos.
func (s SinPrueba) Total() int { return len(s.PorClase) + len(s.PorFaceta) }

// ObservablesSinPrueba son las obligaciones observables de este paquete que
// nadie comprueba todavia.
//
// ES LA SEGUNDA DIRECCION DEL EMPAREJAMIENTO, y NO es fatal, a proposito y con
// su motivo: hoy las 133 obligaciones observables del corpus no tienen ni una
// prueba declarada, asi que exigirlo dejaria el corpus entero sin cargar. Una
// puerta que no puede estar verde nunca no es una puerta, es un rojo permanente,
// y un rojo permanente es tan invisible como un verde falso.
//
// Se devuelve CONTADO para que la casilla de la etapa 3 tenga de donde derivar
// su objetivo en vez de escribirlo a mano.
func (p *Paquete) ObservablesSinPrueba() SinPrueba {
	con := map[string]bool{}
	for _, pr := range p.Pruebas {
		con[pr.Obligacion] = true
	}
	var out SinPrueba
	for _, o := range p.Obligaciones {
		if con[o.ID] {
			continue
		}
		switch {
		case o.ObservablePorClase():
			out.PorClase = append(out.PorClase, o.ID)
		case o.ObservablePorFaceta():
			out.PorFaceta = append(out.PorFaceta, o.ID)
		}
	}
	return out
}

// validarPruebas es el linter del bloque.
func (p *Paquete) validarPruebas(anotar func(error)) {
	porID := map[string]*Obligacion{}
	for i := range p.Obligaciones {
		porID[p.Obligaciones[i].ID] = &p.Obligaciones[i]
	}
	vistas := map[string]bool{}

	for _, pr := range p.Pruebas {
		donde := fmt.Sprintf("paquete %s, prueba %q", p.URN, pr.ID)

		if pr.ID == "" {
			anotar(fmt.Errorf("%w: paquete %s. Arreglo: toda prueba lleva un "+
				"identificador, que es por donde la nombra una observacion",
				ErrPruebaSinID, p.URN))
			continue
		}
		if vistas[pr.ID] {
			anotar(fmt.Errorf("%w: %s. Arreglo: los identificadores de prueba son unicos "+
				"dentro del paquete; con dos iguales, las observaciones de una acaban "+
				"contadas en la otra y cual gana lo decide el orden del fichero",
				ErrPruebaRepetida, donde))
			continue
		}
		vistas[pr.ID] = true

		if pr.Obligacion == "" {
			anotar(fmt.Errorf("%w: %s. Arreglo: di que obligacion comprueba. Una prueba "+
				"sin obligacion no se puede colgar de nada", ErrPruebaSinObligacion, donde))
			continue
		}
		o, hay := porID[pr.Obligacion]
		if !hay {
			anotar(fmt.Errorf("%w: %s apunta a %q, que no esta en este paquete. Arreglo: "+
				"corrige el identificador, o mueve la prueba al paquete donde viva esa "+
				"obligacion", ErrPruebaHuerfana, donde, pr.Obligacion))
			continue
		}

		// LA PUERTA QUE PIDE D-22, y es la que separa un modelo de control de
		// una lista de deseos: si la obligacion no es observable, no hay nada
		// que recolectar, y una prueba ahi promete una comprobacion automatica
		// que ningun conector va a poder hacer.
		if !o.EsObservable() {
			anotar(fmt.Errorf("%w: %s comprueba %q, cuya clase es %q y cuyas facetas son "+
				"%v. Arreglo: o la obligacion se marca observable porque de verdad se "+
				"puede observar, o la prueba sobra. Una prueba sobre lo que no se observa "+
				"promete una comprobacion que ningun recolector puede hacer",
				ErrPruebaNoObservable, donde, pr.Obligacion, o.ClaseE2E, o.Facetas))
		}

		if pr.Recurso == "" {
			anotar(fmt.Errorf("%w: %s. Arreglo: di sobre que se observa. La obligacion "+
				"declara %v", ErrPruebaSinRecurso, donde, o.Recursos))
		} else if !declara(o.Recursos, pr.Recurso) {
			anotar(fmt.Errorf("%w: %s observa %q y su obligacion declara %v. Arreglo: o el "+
				"recurso entra en `recursos` de la obligacion porque la norma lo pide, o "+
				"la prueba mira otra cosa. Observar algo que la obligacion nunca dijo "+
				"necesitar es inventarse el alcance de la comprobacion",
				ErrPruebaRecursoAjeno, donde, pr.Recurso, o.Recursos))
		}

		ttl, okTTL := duracionPositiva(pr.TTL)
		if !okTTL {
			anotar(fmt.Errorf("%w: %s trae ttl %q. Arreglo: una duracion ISO-8601 mayor "+
				"que cero (P30D, P6M, PT24H). El vacio NO significa 'no caduca': eso se "+
				"dice con pass_por_defecto. Una prueba sin frescura hace que una "+
				"observacion de hace tres anos valga igual que la de esta manana",
				ErrPruebaTTLInvalido, donde, pr.TTL))
		}
		sla, okSLA := duracionPositiva(pr.SLA)
		if !okSLA {
			anotar(fmt.Errorf("%w: %s trae sla %q. Arreglo: una duracion ISO-8601 mayor "+
				"que cero. Sin plazo de remediacion no hay nada que escalar",
				ErrPruebaSLAInvalido, donde, pr.SLA))
		}
		if okTTL && okSLA && sla > ttl {
			anotar(fmt.Errorf("%w: %s tiene sla %q sobre ttl %q. Arreglo: baja el sla o "+
				"sube el ttl. Un sla mas largo que la frescura es un plazo que no se "+
				"puede agotar: la observacion se queda obsoleta antes de que venza y el "+
				"fallo que tenia que escalar no llega nunca",
				ErrPruebaSLAMayorQueTTL, donde, pr.SLA, pr.TTL))
		}

		if pr.Activa != "" {
			if _, err := time.Parse("2006-01-02", pr.Activa); err != nil {
				anotar(fmt.Errorf("%w: %s trae activa %q. Arreglo: una fecha AAAA-MM-DD, "+
					"o nada. Una fecha de rollout que no se entiende no es 'sin rollout': "+
					"es un dato que hay y no se entiende", ErrPruebaActivaIlegible,
					donde, pr.Activa))
			}
		}

		// CIERRA: obligatorio, y prohibido sobre una faceta.
		switch {
		case pr.Cierra == nil:
			anotar(fmt.Errorf("%w: %s. Arreglo: di `\"cierra\": true` si esta prueba en "+
				"verde cierra la obligacion, o `false` si solo aporta a un aspecto de "+
				"ella. El campo ausente y el campo a false son dos cosas distintas y solo "+
				"una es una decision: sin el, la pantalla pintaria «consta» sobre una "+
				"obligacion cuyo descargo de verdad es otro", ErrPruebaSinCierra, donde))
		case *pr.Cierra && o.ObservablePorFaceta():
			anotar(fmt.Errorf("%w: %s cierra %q, que es de clase %q y solo observable por "+
				"faceta. Arreglo: pon `\"cierra\": false`. Lo que la norma exige ahi no es "+
				"observable: es %s, y su descargo es un documento o un procedimiento que "+
				"esta prueba no mira. Darlo por cerrado porque su aspecto observable dio "+
				"verde es absolver de mas con cara de dato",
				ErrPruebaCierraUnaFaceta, donde, pr.Obligacion, o.ClaseE2E, o.ClaseE2E))
		}

		if pr.Predicado == "" {
			anotar(fmt.Errorf("%w: %s. Arreglo: escribe QUE se evalua, para que una "+
				"persona lo pueda leer y contrastar. Sin predicado declarado, el campo "+
				"Satisfecho de una observacion es un veredicto disfrazado de dato "+
				"(invariante 13)", ErrPruebaSinPredicado, donde))
		}
	}
}

func declara(rs []TipoRecurso, r TipoRecurso) bool {
	for _, x := range rs {
		if x == r {
			return true
		}
	}
	return false
}

// duracionPositiva parsea y exige que sea mayor que cero.
//
// Devuelve una duracion COMPARABLE en la unidad mas gruesa que hace falta para
// ordenar dos plazos, y eso tiene un limite que se dice: los meses se cuentan a
// 30 dias SOLO PARA COMPARAR ttl con sla. No se usa para calcular ninguna fecha
// (para eso esta `ventana.Sumar`, que sabe de calendarios), asi que la
// aproximacion no puede mover un vencimiento: como mucho hace que un P1M frente
// a un P30D se considere igual, y esos dos plazos no se distinguen a ojo
// tampoco.
func duracionPositiva(s string) (time.Duration, bool) {
	if s == "" {
		return 0, false
	}
	d, err := ventana.ParseDuracion(s)
	if err != nil || d.Indeterminado {
		return 0, false
	}
	total := time.Duration(d.Meses)*30*24*time.Hour +
		time.Duration(d.Dias)*24*time.Hour +
		time.Duration(d.Horas)*time.Hour +
		time.Duration(d.Mins)*time.Minute
	if total <= 0 {
		return 0, false
	}
	return total, true
}
