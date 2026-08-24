// Package corpus define el formato de paquete normativo y lo que se deriva de el.
//
// La decision de diseno que sostiene este paquete: un paquete no declara solo
// obligaciones. Declara ademas los tipos de entidad con su esquema de atributos,
// las preguntas de alcance, las plantillas de entregable y los tipos de recurso
// que necesita. De ahi se derivan, sin escribir una linea por norma:
//
//	EsquemaUI    los formularios de la interfaz
//	Entrevista   el cuestionario de alcance, ordenado por obligaciones desbloqueadas
//	Entregables  los documentos, con trazabilidad obligacion -> plantilla -> campo
//	Conectores   que recolectores hacen falta y cuales no
//
// La propiedad que esto compra, y que es verificable en CI: anadir la norma 31
// no toca ni un fichero fuera de su directorio de paquete. Un GRC cuyo corpus es
// un arbol plano de requisitos cargado desde Excel no puede tener esta propiedad,
// porque no tiene donde declarar nada de lo anterior.
package corpus

import (
	"fmt"
	"sort"
	"strings"
)

// Clase determina que se puede distribuir del paquete. Es la frontera legal,
// y el linter la hace cumplir en vez de confiar en que nadie se equivoque.
type Clase uint8

const (
	// Importado: catalogo ya publicado en dominio publico y legible por maquina.
	// Se distribuye entero. NIST 800-53 y CSF 2.0 en OSCAL, CC0 1.0.
	Importado Clase = iota
	// Transcrito: texto de BOE o DOUE. Se distribuye entero con las obligaciones
	// formales de reutilizacion cumplidas. Art. 13 TRLPI y Decision 2011/833/UE.
	Transcrito
	// Referencial: identificadores y mapeo propio, SIN texto normativo. El cliente
	// aporta su copia licenciada. ISO, PCI DSS, SOC 2, TISAX.
	Referencial
	// Delegado: no se distribuye nada. La comprobacion la ejecuta una herramienta
	// externa que ya tiene la licencia del contenido. CIS Benchmarks via OpenSCAP,
	// Trivy o Prowler.
	Delegado
)

func (c Clase) String() string {
	return [...]string{"importado", "transcrito", "referencial", "delegado"}[c]
}

// LimiteTextoReferencial es el numero maximo de caracteres de texto normativo
// que puede llevar una obligacion de un paquete referencial. Un identificador y
// un titulo corto caben; el enunciado de un control, no. El limite es
// deliberadamente conservador: la zona gris se resuelve a la baja.
const LimiteTextoReferencial = 120

// Vigencia acota cuando existe algo. Nil en Hasta significa abierto.
type Vigencia struct {
	Desde string `json:"desde"`
	Hasta string `json:"hasta,omitempty"`
}

// TipoAtributo es el tipo de un atributo de entidad. La interfaz se genera de aqui.
type TipoAtributo uint8

const (
	Texto TipoAtributo = iota
	Entero
	Booleano
	Fecha
	Enumerado // usa Escala para el orden, si lo tiene
)

func (t TipoAtributo) String() string {
	return [...]string{"texto", "entero", "booleano", "fecha", "enumerado"}[t]
}

// Atributo describe un dato de una entidad. Genera un campo de formulario.
type Atributo struct {
	Nombre   string       `json:"nombre"`
	Tipo     TipoAtributo `json:"tipo"`
	Valores  []string     `json:"valores,omitempty"` // solo enumerado
	Escala   string       `json:"escala,omitempty"`  // ref a una Escala declarada
	Obligado bool         `json:"obligado"`
	Ayuda    string       `json:"ayuda,omitempty"`
	Cita     string       `json:"cita"` // de donde sale que este dato importa
}

// TipoEntidad es un tipo de sujeto que el paquete introduce.
type TipoEntidad struct {
	Nombre      string     `json:"nombre"`
	Descripcion string     `json:"descripcion"`
	Atributos   []Atributo `json:"atributos"`
}

// Pregunta de alcance. La entrevista es la union de las preguntas de los
// paquetes instalados, ordenada por cuantas obligaciones desbloquea cada una.
type Pregunta struct {
	ID         string   `json:"id"`
	Texto      string   `json:"texto"`
	Cita       string   `json:"cita"`
	Entidad    string   `json:"entidad"`    // que TipoEntidad rellena
	Atributo   string   `json:"atributo"`   // que Atributo fija
	Desbloquea []string `json:"desbloquea"` // IDs de obligacion, para ordenar
	Ayuda      string   `json:"ayuda,omitempty"`
}

// CampoPlantilla es un hueco de un entregable documental.
type CampoPlantilla struct {
	Nombre string `json:"nombre"`
	// Origen dice de donde sale el valor. Debe ser derivable: un atributo de
	// entidad, un estado de control o una obligacion. Nunca texto libre de un LLM.
	Origen string `json:"origen"`
}

// Plantilla es un entregable documental versionado.
type Plantilla struct {
	ID     string           `json:"id"`
	Titulo string           `json:"titulo"`
	Cita   string           `json:"cita"`
	Campos []CampoPlantilla `json:"campos"`
}

// TipoRecurso es un recurso canonico que el paquete necesita observar.
// De aqui sale que conectores hacen falta, y cuales no aportan nada.
type TipoRecurso string

// Obligacion es el atomo. Aqui va solo lo que el resto del sistema necesita
// del paquete; la temporalidad completa vive en ventana y la aplicabilidad en
// aplicabilidad.
type Obligacion struct {
	ID         string        `json:"id"`
	Articulo   string        `json:"articulo"`
	TextoLegal string        `json:"texto_legal,omitempty"` // vacio en referencial y delegado
	Cita       string        `json:"cita"`
	Vigencia   Vigencia      `json:"vigencia"`
	Entregable string        `json:"entregable,omitempty"` // ref a Plantilla.ID
	Recursos   []TipoRecurso `json:"recursos,omitempty"`
	// Delegado dice que herramienta externa comprueba esto. Obligatorio y solo
	// permitido en paquetes de clase Delegado.
	Delegado  string   `json:"delegado,omitempty"`
	Preguntas []string `json:"preguntas,omitempty"` // IDs de Pregunta que la desbloquean
}

// Paquete es la unidad de distribucion del corpus.
type Paquete struct {
	URN          string        `json:"urn"`
	Version      string        `json:"version"`
	Clase        Clase         `json:"clase"`
	Licencia     string        `json:"licencia"`
	Fuente       string        `json:"fuente"`      // enlace exigido por las condiciones del BOE
	Consolidado  bool          `json:"consolidado"` // obliga al aviso de texto informativo
	Vigencia     Vigencia      `json:"vigencia"`
	Entidades    []TipoEntidad `json:"entidades,omitempty"`
	Preguntas    []Pregunta    `json:"preguntas,omitempty"`
	Obligaciones []Obligacion  `json:"obligaciones"`
	Plantillas   []Plantilla   `json:"plantillas,omitempty"`
	Escalas      []string      `json:"escalas,omitempty"`
}

// ---------------------------------------------------------------------------
// El linter. Rechaza lo que no es seguro en vez de ejecutarlo a ver que pasa.
// ---------------------------------------------------------------------------

// Validar comprueba las invariantes del paquete. Devuelve todos los fallos, no
// solo el primero, porque quien escribe un paquete quiere arreglarlos de una vez.
func (p *Paquete) Validar() []error {
	var errs []error
	e := func(f string, a ...any) { errs = append(errs, fmt.Errorf(f, a...)) }

	if p.URN == "" {
		e("paquete sin urn")
	}
	if p.Version == "" {
		e("paquete %s sin version", p.URN)
	}
	if p.Fuente == "" {
		e("paquete %s sin fuente: las condiciones de reutilizacion del BOE y la "+
			"Decision 2011/833/UE exigen citar la fuente con enlace", p.URN)
	}

	plantillas := map[string]bool{}
	for _, t := range p.Plantillas {
		plantillas[t.ID] = true
		if t.Cita == "" {
			e("plantilla %s sin cita normativa", t.ID)
		}
		for _, c := range t.Campos {
			if c.Origen == "" {
				e("plantilla %s campo %s sin origen: un entregable no puede tener "+
					"huecos que rellene un humano sin trazabilidad", t.ID, c.Nombre)
			}
		}
	}

	preguntas := map[string]bool{}
	entidades := map[string]map[string]bool{}
	for _, te := range p.Entidades {
		at := map[string]bool{}
		for _, a := range te.Atributos {
			at[a.Nombre] = true
			if a.Cita == "" {
				e("entidad %s atributo %s sin cita: si no se sabe de que articulo "+
					"sale el dato, no se le pregunta al usuario", te.Nombre, a.Nombre)
			}
			if a.Tipo == Enumerado && len(a.Valores) == 0 {
				e("entidad %s atributo %s es enumerado y no declara valores", te.Nombre, a.Nombre)
			}
		}
		entidades[te.Nombre] = at
	}
	for _, q := range p.Preguntas {
		preguntas[q.ID] = true
		if q.Cita == "" {
			e("pregunta %s sin cita", q.ID)
		}
		at, ok := entidades[q.Entidad]
		if !ok {
			e("pregunta %s apunta a la entidad %s, que el paquete no declara", q.ID, q.Entidad)
		} else if !at[q.Atributo] {
			e("pregunta %s apunta al atributo %s.%s, que no existe", q.ID, q.Entidad, q.Atributo)
		}
		if len(q.Desbloquea) == 0 {
			e("pregunta %s no desbloquea ninguna obligacion: es una pregunta que no "+
				"sirve para nada y no se le hace al usuario", q.ID)
		}
	}

	obl := map[string]bool{}
	for _, o := range p.Obligaciones {
		if obl[o.ID] {
			e("obligacion %s duplicada", o.ID)
		}
		obl[o.ID] = true
		if o.Cita == "" {
			e("obligacion %s sin cita normativa", o.ID)
		}
		if o.Entregable != "" && !plantillas[o.Entregable] {
			e("obligacion %s declara el entregable %s, que el paquete no incluye",
				o.ID, o.Entregable)
		}
		for _, q := range o.Preguntas {
			if !preguntas[q] {
				e("obligacion %s referencia la pregunta %s, que no existe", o.ID, q)
			}
		}
		// La frontera legal, comprobada por el linter y no por buena voluntad.
		switch p.Clase {
		case Referencial:
			if len(o.TextoLegal) > LimiteTextoReferencial {
				e("obligacion %s: paquete referencial con %d caracteres de texto legal "+
					"(limite %d). ISO, PCI DSS, SOC 2 y TISAX no autorizan la "+
					"redistribucion de su texto: identificador y titulo corto, nada mas",
					o.ID, len(o.TextoLegal), LimiteTextoReferencial)
			}
			if o.Delegado != "" {
				e("obligacion %s: solo un paquete delegado declara herramienta externa", o.ID)
			}
		case Delegado:
			if o.TextoLegal != "" {
				e("obligacion %s: un paquete delegado no distribuye texto. La licencia "+
					"del contenido la tiene la herramienta que lo comprueba", o.ID)
			}
			if o.Delegado == "" {
				e("obligacion %s: paquete delegado sin herramienta declarada. CIS "+
					"Benchmarks es CC BY-NC-SA: incompatible con AGPL y con vender, "+
					"asi que se lee la salida de quien si tiene la licencia", o.ID)
			}
		default:
			if o.Delegado != "" {
				e("obligacion %s: solo un paquete delegado declara herramienta externa", o.ID)
			}
		}
	}
	// Las preguntas apuntan a obligaciones que existen.
	for _, q := range p.Preguntas {
		for _, id := range q.Desbloquea {
			if !obl[id] {
				e("pregunta %s dice desbloquear %s, que no es una obligacion del paquete",
					q.ID, id)
			}
		}
	}
	return errs
}

// ---------------------------------------------------------------------------
// Lo derivado. Nada de esto se escribe por norma: sale del paquete.
// ---------------------------------------------------------------------------

// CampoUI es un campo de formulario generado desde el modelo.
type CampoUI struct {
	Entidad  string
	Atributo string
	Etiqueta string
	Tipo     string
	Valores  []string
	Obligado bool
	Ayuda    string
	Cita     string
	Paquetes []string // que paquetes necesitan este dato
}

// EsquemaUI deriva los formularios de la interfaz de los paquetes instalados.
// Un atributo pedido por tres normas se pregunta una vez y se dice quien lo pide.
func EsquemaUI(ps []*Paquete) []CampoUI {
	idx := map[string]*CampoUI{}
	var orden []string
	for _, p := range ps {
		for _, te := range p.Entidades {
			for _, a := range te.Atributos {
				k := te.Nombre + "." + a.Nombre
				c, ok := idx[k]
				if !ok {
					c = &CampoUI{
						Entidad: te.Nombre, Atributo: a.Nombre,
						Etiqueta: a.Nombre, Tipo: a.Tipo.String(),
						Valores: a.Valores, Obligado: a.Obligado,
						Ayuda: a.Ayuda, Cita: a.Cita,
					}
					idx[k] = c
					orden = append(orden, k)
				}
				c.Obligado = c.Obligado || a.Obligado
				c.Paquetes = append(c.Paquetes, p.URN)
			}
		}
	}
	sort.Strings(orden)
	out := make([]CampoUI, 0, len(orden))
	for _, k := range orden {
		out = append(out, *idx[k])
	}
	return out
}

// PreguntaEntrevista es una pregunta del cuestionario de alcance ya ordenada.
type PreguntaEntrevista struct {
	Pregunta
	Paquete     string
	NDesbloquea int
}

// Entrevista construye el cuestionario de alcance: la union de las preguntas de
// los paquetes instalados, ordenada por cuantas obligaciones desbloquea cada una.
// Nunca se ensena un catalogo de controles en frio.
func Entrevista(ps []*Paquete) []PreguntaEntrevista {
	var out []PreguntaEntrevista
	for _, p := range ps {
		for _, q := range p.Preguntas {
			out = append(out, PreguntaEntrevista{
				Pregunta: q, Paquete: p.URN, NDesbloquea: len(q.Desbloquea),
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].NDesbloquea != out[j].NDesbloquea {
			return out[i].NDesbloquea > out[j].NDesbloquea
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// Traza es la trazabilidad obligacion -> entregable -> campo, que es lo que
// convierte un generador de plantillas en una herramienta que implanta.
type Traza struct {
	Obligacion string
	Plantilla  string
	Campo      string
	Origen     string
}

// Trazabilidad devuelve el mapa completo. Si esta vacio para una obligacion que
// declara entregable, el linter ya lo habra rechazado.
func Trazabilidad(ps []*Paquete) []Traza {
	var out []Traza
	for _, p := range ps {
		pl := map[string]Plantilla{}
		for _, t := range p.Plantillas {
			pl[t.ID] = t
		}
		for _, o := range p.Obligaciones {
			if o.Entregable == "" {
				continue
			}
			for _, c := range pl[o.Entregable].Campos {
				out = append(out, Traza{o.ID, o.Entregable, c.Nombre, c.Origen})
			}
		}
	}
	return out
}

// NecesidadRecurso dice cuantas obligaciones dependen de un tipo de recurso.
// Es la respuesta a "que conector construyo primero" con un numero detras, en
// vez de con una intuicion.
type NecesidadRecurso struct {
	Recurso      TipoRecurso
	Obligaciones int
	Normas       []string
}

// Conectores ordena los tipos de recurso por cuantas obligaciones desbloquean.
func Conectores(ps []*Paquete) []NecesidadRecurso {
	n := map[TipoRecurso]*NecesidadRecurso{}
	for _, p := range ps {
		vistos := map[TipoRecurso]bool{}
		for _, o := range p.Obligaciones {
			for _, r := range o.Recursos {
				x, ok := n[r]
				if !ok {
					x = &NecesidadRecurso{Recurso: r}
					n[r] = x
				}
				x.Obligaciones++
				if !vistos[r] {
					x.Normas = append(x.Normas, p.URN)
					vistos[r] = true
				}
			}
		}
	}
	out := make([]NecesidadRecurso, 0, len(n))
	for _, x := range n {
		out = append(out, *x)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Obligaciones != out[j].Obligaciones {
			return out[i].Obligaciones > out[j].Obligaciones
		}
		return out[i].Recurso < out[j].Recurso
	})
	return out
}

// Cobertura es lo que se publica en COBERTURA.md. Un proyecto que publica lo que
// le falta es mas creible que uno que publica un porcentaje.
type Cobertura struct {
	Paquete        string
	Total          int
	ConEntregable  int
	ConRecurso     int
	Delegadas      int
	SinAutomatizar []string
}

// Medir calcula la cobertura sin redondear a favor.
func Medir(p *Paquete) Cobertura {
	c := Cobertura{Paquete: p.URN, Total: len(p.Obligaciones)}
	for _, o := range p.Obligaciones {
		if o.Entregable != "" {
			c.ConEntregable++
		}
		if len(o.Recursos) > 0 {
			c.ConRecurso++
		}
		if o.Delegado != "" {
			c.Delegadas++
		}
		if len(o.Recursos) == 0 && o.Delegado == "" {
			c.SinAutomatizar = append(c.SinAutomatizar, o.ID)
		}
	}
	return c
}

func (c Cobertura) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %d obligaciones\n", c.Paquete, c.Total)
	fmt.Fprintf(&b, "  con entregable documental   %d\n", c.ConEntregable)
	fmt.Fprintf(&b, "  con recurso observable      %d\n", c.ConRecurso)
	fmt.Fprintf(&b, "  delegadas a herramienta     %d\n", c.Delegadas)
	fmt.Fprintf(&b, "  sin automatizar             %d", len(c.SinAutomatizar))
	if len(c.SinAutomatizar) > 0 {
		fmt.Fprintf(&b, "  %s", strings.Join(c.SinAutomatizar, ", "))
	}
	return b.String()
}
