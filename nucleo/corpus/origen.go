package corpus

import (
	"fmt"
	"strings"
)

// EL ORIGEN DE UN CAMPO DE PLANTILLA, declarado.
//
// POR QUE EXISTE ESTE FICHERO. `CampoPlantilla.Origen` dice, desde el primer
// dia, que el valor "debe ser derivable" y que nunca es texto libre. El linter
// comprobaba UNA cosa: que la cadena no estuviera vacia. La gramatica no estaba
// escrita en ningun sitio, asi que cada paquete se invento la suya, y el
// resultado se midio el 29-08-2026 sobre los 52 origenes del corpus:
//
//	48 tenian una base real (una obligacion o una entidad del paquete)
//	 2 apuntaban a un esquema de nombres QUE NADIE USA: `iso27001.anexo_a.*`,
//	   cuando las 93 obligaciones del anexo se llaman `iso27001.a.5.1`. El glob
//	   no casaba con nada y el entregable habria salido con dos huecos
//	 4 se escribian SIN PREFIJO (`organizacion.sector` en vez de
//	   `entidad:organizacion.sector`), o sea el mismo dato en dos gramaticas
//	13 sufijos distintos, inventados uno a uno
//
// Es la subfamilia "alcanzabilidad, no existencia" en la fila que
// docs/pendientes.md dejaba abierta con estas palabras: «plantillas | el campo
// declara `origen`, ¿hay algun camino que rellene ese origen?». La respuesta
// era que no lo comprobaba nadie.
//
// LA DISTINCION QUE ORDENA TODO ESTO, y no es de forma sino de fondo: un campo
// o lo DERIVA plazum o lo APORTA una persona. Las dos cosas son legitimas y la
// segunda no es un defecto: un informe de auditoria tiene hallazgos que escribe
// el auditor, y ninguna maquina los va a producir. Lo que no es legitimo es que
// las dos se escriban igual, porque entonces un hueco que nadie va a rellenar
// tiene el mismo aspecto que un dato que sale solo. El mensaje del propio
// linter ya lo decia — "un entregable no puede tener huecos que rellene un
// humano SIN TRAZABILIDAD" — y lo que faltaba era justo la trazabilidad: decir
// cual es cual.

// ClaseOrigen es el prefijo: de que parte del sistema sale el valor.
type ClaseOrigen uint8

const (
	// DeObligacion: el valor sale del cumplimiento de una obligacion, o de un
	// dato derivado de su reloj.
	DeObligacion ClaseOrigen = iota
	// DeEntidad: el valor es un atributo declarado del alcance.
	DeEntidad
	// DeIncidente: el valor sale del objeto Incidente que dispara el entregable.
	DeIncidente
)

var prefijos = map[string]ClaseOrigen{
	"obligacion": DeObligacion,
	"entidad":    DeEntidad,
	"incidente":  DeIncidente,
}

func (c ClaseOrigen) String() string {
	for k, v := range prefijos {
		if v == c {
			return k
		}
	}
	return "clase de origen desconocida"
}

// sufijosDerivables son los que CALCULA plazum a partir de datos que ya tiene.
// Cerrado: anadir uno exige escribir tambien de donde sale.
var sufijosDerivables = map[string]string{
	"ultimo_hito": "la fecha del ultimo vencimiento cumplido del reloj de esa obligacion",
	"primer_hito": "la fecha del primer vencimiento de su reloj",
	"aplicable":   "si la aplicabilidad alcanza a esa obligacion en este alcance",
	"exclusion":   "el articulo por el que la aplicabilidad NO la alcanza",
}

// sufijosAportados son los que escribe una PERSONA. No son un defecto: son la
// mitad del trabajo que ninguna maquina hace. Van declarados para que un
// entregable pueda decir, campo a campo, cuales rellena solo y cuales espera.
var sufijosAportados = map[string]string{
	"":                "el cumplimiento de la obligacion, redactado",
	"evidencia":       "la evidencia que la obligacion pide",
	"hallazgos":       "los hallazgos de la revision",
	"estado":          "el estado declarado por quien responde",
	"plan":            "el plan de accion",
	"roles":           "los roles designados",
	"metricas":        "las metricas recogidas",
	"seleccion":       "la seleccion de medidas y su porque",
	"correspondencia": "la correspondencia entre medidas",
	"responsable":     "quien firma",
	"metodologia":     "la metodologia seguida",
}

// camposDeIncidente es el vocabulario CERRADO de lo que un incidente puede
// aportar a un entregable. Cerrado porque cada entrada tiene que tener a
// alguien que la conteste: una entrada sin respuesta es un hueco con nombre,
// que es peor que un hueco.
var camposDeIncidente = map[string]string{
	"id":                    "el identificador del incidente",
	"ocurrio":               "el instante del mundo en que empezo",
	"primer_conocimiento":   "el instante en que la organizacion tuvo constancia",
	"clasificacion_vigente": "la clasificacion que rige ahora",
}

// Origen es un `origen` de plantilla ya leido.
type Origen struct {
	Clase ClaseOrigen
	// Patron es el id de la obligacion (puede llevar *), el nombre de la
	// entidad, o el campo del incidente.
	Patron string
	// Sufijo es lo que se pide DE esa base. En DeEntidad es el atributo.
	Sufijo string
	// Derivable dice si lo calcula plazum (true) o lo aporta una persona
	// (false). Es la distincion que hace util a este tipo: un entregable puede
	// decir que sabe rellenar y que esta esperando, en vez de sacar todos los
	// huecos iguales.
	Derivable bool
	// Que describe el origen en una frase, para el mensaje de error y para la
	// pantalla que pida el dato que falta.
	Que string
}

// ParseOrigen lee la cadena y falla con el arreglo puesto. No resuelve la
// referencia: eso lo hace el linter, que es quien tiene el paquete delante.
func ParseOrigen(s string) (Origen, error) {
	pref, resto, hay := strings.Cut(s, ":")
	if !hay {
		return Origen{}, fmt.Errorf("el origen %q no dice de donde sale: le falta el prefijo. "+
			"Los que hay son %s. Arreglo: si es un atributo del alcance, se escribe "+
			"\"entidad:%s\"", s, listarClases(prefijos), s)
	}
	clase, ok := prefijos[pref]
	if !ok {
		return Origen{}, fmt.Errorf("el origen %q empieza por %q, que no es una clase de "+
			"origen. Los que hay son %s", s, pref, listarClases(prefijos))
	}
	if resto == "" {
		return Origen{}, fmt.Errorf("el origen %q no dice a QUE se refiere", s)
	}

	switch clase {
	case DeEntidad:
		ent, atr, hay := strings.Cut(resto, ".")
		if !hay || ent == "" || atr == "" {
			return Origen{}, fmt.Errorf("el origen %q tiene que nombrar la entidad y el "+
				"atributo (entidad:organizacion.sector)", s)
		}
		return Origen{Clase: clase, Patron: ent, Sufijo: atr, Derivable: true,
			Que: "el atributo " + atr + " de " + ent + ", declarado en el alcance"}, nil

	case DeIncidente:
		que, ok := camposDeIncidente[resto]
		if !ok {
			return Origen{}, fmt.Errorf("el origen %q pide %q al incidente, y un incidente "+
				"solo sabe contestar %s", s, resto, listar(camposDeIncidente))
		}
		return Origen{Clase: clase, Patron: resto, Derivable: true, Que: que}, nil
	}

	// DeObligacion: se prueba el sufijo MAS LARGO que case, y el recorrido va
	// por una lista ordenada y no por el mapa. Recorrer un mapa de Go da un
	// orden distinto en cada ejecucion, y una gramatica que dependa de eso deja
	// de ser una gramatica: es la misma razon por la que el empate de
	// clasificaciones se declara en vez de resolverse.
	mejor, hay := Origen{}, false
	for _, suf := range sufijosOrdenados() {
		base, ok := strings.CutSuffix(resto, "."+suf)
		if !ok || base == "" {
			continue
		}
		if hay && len(mejor.Sufijo) >= len(suf) {
			continue
		}
		que, derivable := sufijosDerivables[suf]
		if !derivable {
			que = sufijosAportados[suf]
		}
		mejor, hay = Origen{Clase: clase, Patron: base, Sufijo: suf,
			Derivable: derivable, Que: que}, true
	}
	if hay {
		return mejor, nil
	}
	// El sufijo `campo.<letra>` es la letra de un apartado, y se admite con
	// cualquier letra porque las letras las pone el boletin, no nosotros.
	if base, letra, ok := cortarLetra(resto); ok {
		return Origen{Clase: clase, Patron: base, Sufijo: "campo." + letra, Derivable: false,
			Que: "lo que pide la letra " + letra + ") del apartado"}, nil
	}
	return Origen{Clase: clase, Patron: resto, Derivable: false,
		Que: sufijosAportados[""]}, nil
}

// cortarLetra parte "x.y.campo.a" en ("x.y", "a").
func cortarLetra(s string) (string, string, bool) {
	base, letra, hay := strings.Cut(s, ".campo.")
	if !hay || base == "" || len(letra) != 1 || letra[0] < 'a' || letra[0] > 'z' {
		return "", "", false
	}
	return base, letra, true
}

func listar(m map[string]string) string {
	var ks []string
	for k := range m {
		if k != "" {
			ks = append(ks, k)
		}
	}
	return ordenado(ks)
}

func listarClases(m map[string]ClaseOrigen) string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	return ordenado(ks)
}

// sufijosOrdenados es la union de los dos vocabularios, en orden estable.
func sufijosOrdenados() []string {
	var ks []string
	for _, tabla := range []map[string]string{sufijosDerivables, sufijosAportados} {
		for k := range tabla {
			if k != "" {
				ks = append(ks, k)
			}
		}
	}
	for i := 1; i < len(ks); i++ {
		for j := i; j > 0 && ks[j] < ks[j-1]; j-- {
			ks[j], ks[j-1] = ks[j-1], ks[j]
		}
	}
	return ks
}

func ordenado(ks []string) string {
	for i := 1; i < len(ks); i++ {
		for j := i; j > 0 && ks[j] < ks[j-1]; j-- {
			ks[j], ks[j-1] = ks[j-1], ks[j]
		}
	}
	return strings.Join(ks, ", ")
}

// casaPatron dice si un id casa con el patron, que puede llevar un `*` que
// significa "cualquier cosa". Un patron sin `*` es una igualdad.
func casaPatron(patron, id string) bool {
	antes, despues, hay := strings.Cut(patron, "*")
	if !hay {
		return patron == id
	}
	return len(id) >= len(antes)+len(despues) &&
		strings.HasPrefix(id, antes) && strings.HasSuffix(id, despues)
}

// validarOrigenesDePlantilla comprueba que cada campo de cada entregable tenga
// un origen que SE PUEDA LEER y que APUNTE A ALGO QUE EXISTE.
//
// Las dos cosas, y la segunda es la que faltaba. Que la cadena no este vacia no
// dice nada: `iso27001.anexo_a.*` no estaba vacio y no casaba con ninguna de las
// 93 obligaciones del anexo, que se llaman `iso27001.a.5.1`. Un campo cuyo
// origen no existe es un hueco que nadie va a rellenar, y el entregable sale con
// el hueco SIN DECIRLO, que es la forma de esta familia: no deja pasar algo
// malo, deja de producir algo bueno.
func (p *Paquete) validarOrigenesDePlantilla(e func(string, ...any)) {
	var ids []string
	for _, o := range p.Obligaciones {
		ids = append(ids, o.ID)
	}
	ents := map[string]map[string]bool{}
	for _, te := range p.Entidades {
		at := map[string]bool{}
		for _, a := range te.Atributos {
			at[a.Nombre] = true
		}
		ents[te.Nombre] = at
	}

	for _, t := range p.Plantillas {
		for _, c := range t.Campos {
			o, err := ParseOrigen(c.Origen)
			if err != nil {
				e("plantilla %s, campo %s: %v", t.ID, c.Nombre, err)
				continue
			}
			switch o.Clase {
			case DeObligacion:
				casan := 0
				for _, id := range ids {
					if casaPatron(o.Patron, id) {
						casan++
					}
				}
				if casan == 0 {
					e("plantilla %s, campo %s: el origen %q apunta a %q y este paquete no "+
						"tiene ninguna obligacion que case. El entregable saldria con un "+
						"hueco y sin decir que lo tiene. Arreglo: corregir el id, o el "+
						"patron si el esquema de nombres del paquete es otro",
						t.ID, c.Nombre, c.Origen, o.Patron)
				}
			case DeEntidad:
				at, hay := ents[o.Patron]
				if !hay {
					e("plantilla %s, campo %s: el origen %q nombra la entidad %q, que el "+
						"paquete no declara", t.ID, c.Nombre, c.Origen, o.Patron)
					continue
				}
				if !at[o.Sufijo] {
					e("plantilla %s, campo %s: el origen %q pide el atributo %q de %q, y esa "+
						"entidad no lo declara. Un atributo que no se declara no se le "+
						"pregunta a nadie, asi que el campo se quedaria vacio para siempre",
						t.ID, c.Nombre, c.Origen, o.Sufijo, o.Patron)
				}
			}
		}
	}
}
