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
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
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
	// Propio: datos creados por el proyecto (demo, calendarios, equivalencias).
	// Licencia propia declarada (Apache-2.0 por defecto). Sin restricciones de
	// texto: no hay tercero con derechos.
	Propio
)

// Valida dice si la clase es una de las declaradas.
//
// Existe porque Clase es un uint8 que llega de un fichero JSON que aporta un
// tercero, y un valor fuera de rango no es una rareza teorica: es la forma de
// esquivar el linter legal. Ver Paquete.Validar.
func (c Clase) Valida() bool { return c <= Propio }

// String NUNCA hace panic.
//
// HALLAZGO DEL FRENTE DE CORPUS: antes indexaba un array de cinco elementos con
// c directamente, asi que Clase(9).String() reventaba con index out of range. El
// valor viene de un fichero de datos de origen no fiable, o sea que un paquete
// malformado tumbaba a cualquiera que se limitara a listar el corpus.
func (c Clase) String() string {
	nombres := [...]string{"importado", "transcrito", "referencial", "delegado", "propio"}
	if int(c) >= len(nombres) {
		return fmt.Sprintf("clase invalida (%d)", uint8(c))
	}
	return nombres[c]
}

// LimiteTextoReferencial es el numero maximo de caracteres de texto normativo
// que puede llevar un campo de PROSA de un paquete referencial o delegado. Un
// identificador y un titulo corto caben; el enunciado de un control, no. El
// limite es deliberadamente conservador: la zona gris se resuelve a la baja.
//
// Se mide en bytes y no en runas, a proposito y a la baja: un texto acentuado
// gasta mas bytes que caracteres, asi que el limite aprieta mas justo donde el
// texto es prosa de verdad. Contar runas le regalaria al que copia un catalogo
// de pago casi el doble de sitio.
const LimiteTextoReferencial = 120

// LimiteCitaReferencial es el techo de los campos que son REFERENCIA y no
// prosa: la cita de un articulo, un URN, una clave de formulario, una fecha,
// un enlace de fuente o la declaracion de licencia.
//
// Por que tienen techo propio y no el de la prosa: una cita es un localizador
// ("CAT/DEMO 9999:2026 A.5.1") y ademas el sitio donde el paquete explica por
// que apunta ahi, asi que pasa de 120 caracteres con toda legitimidad; el
// corpus de hoy llega a 229. Y por que tienen techo, en vez de quedar libres:
// un campo de texto libre sin limite es un canal por el que se cuela el
// enunciado de un control, y el linter no sabe distinguir un localizador largo
// de un parrafo copiado. Se le pone tope en vez de dejarlo abierto.
const LimiteCitaReferencial = 300

// LimiteDerivacionReferencial es el techo del unico campo que no es ni prosa ni
// localizador: la cita_del_esperado de un caso dorado.
//
// Un dorado bien escrito justifica su fecha PASO A PASO ("desde el miercoles
// 22-04-2026 los diez habiles son 23, 24, 27..."), y esa aritmetica la escribe
// quien autora el paquete, no el catalogo de pago. Bajarle el techo a 300
// obligaria a resumir justo la parte que hace auditable el caso, que es lo
// contrario de lo que se busca. El corpus de hoy llega a 438.
const LimiteDerivacionReferencial = 600

// Los errores del formato que se comprueban por identidad, no por el texto del
// mensaje. Un test que busque "clase" con strings.Contains lo encuentra dentro
// de "clase_e2e" y da verde con el fallo delante: eso ya paso aqui.
var (
	// ErrTextoRedistribuido: un campo de PROSA pasa del limite en un paquete
	// que no puede redistribuir texto de un tercero. Es la frontera legal.
	ErrTextoRedistribuido = errors.New("texto de un tercero por encima del limite de la clase")
	// ErrCitaDesbordada: un campo de REFERENCIA o de DERIVACION pasa de su
	// techo. No es necesariamente texto normativo, pero ya no es un localizador.
	ErrCitaDesbordada = errors.New("campo de referencia por encima del limite de la clase")
	// ErrVigenciaIlegible: una fecha de vigencia que no se puede leer. Viene de
	// un fichero de datos de un tercero, asi que no es una rareza teorica.
	ErrVigenciaIlegible = errors.New("fecha de vigencia ilegible")
	// ErrVigenciaInvertida: desde posterior a hasta. Una obligacion asi no esta
	// vigente NUNCA, y casi siempre es un error de tecleo, no una derogacion.
	ErrVigenciaInvertida = errors.New("vigencia con desde posterior a hasta")
	// ErrVigenciaSinDesde: no hay fecha de inicio y no hay de donde heredarla.
	ErrVigenciaSinDesde = errors.New("vigencia sin fecha de inicio")
	// ErrObligacionSinID: una obligacion sin identificador no se puede citar,
	// ni referenciar desde una pregunta, ni seguir en el expediente.
	ErrObligacionSinID = errors.New("obligacion sin id")
)

// Vigencia acota cuando existe algo. Hasta vacio significa abierta por arriba,
// que es el caso normal: una norma en vigor no declara cuando la derogaran.
//
// Las dos fechas admiten fecha sola (2026-01-01) o instante RFC3339. La fecha
// sola de Hasta cubre el DIA ENTERO: "vigente hasta el 4 de mayo de 2024" es
// hasta el final de ese dia, no hasta su primer segundo. Ver VigenteEn.
type Vigencia struct {
	Desde string `json:"desde"`
	Hasta string `json:"hasta,omitempty"`
}

// rango es una vigencia ya interpretada, con el fin normalizado al ULTIMO
// instante cubierto.
type rango struct {
	desde     time.Time
	hasta     time.Time
	sinInicio bool // no declara desde: si es de una obligacion, hereda del paquete
	abierta   bool // no declara hasta: sigue en vigor
}

// fechaDeVigencia lee una de las dos formas que escriben los paquetes. Devuelve
// ademas si venia con hora, porque de eso depende donde termina un "hasta".
func fechaDeVigencia(campo, s string) (t time.Time, conHora bool, err error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true, nil
	}
	t, err = time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("%w: %s=%q. Se escribe como fecha "+
			"(2026-01-31) o como instante RFC3339 (2026-01-31T23:59:59Z)",
			ErrVigenciaIlegible, campo, s)
	}
	return t, false, nil
}

// interpretar convierte la vigencia declarada en un rango comparable.
func (v Vigencia) interpretar() (rango, error) {
	var r rango
	if v.Desde == "" {
		r.sinInicio = true
	} else {
		d, _, err := fechaDeVigencia("desde", v.Desde)
		if err != nil {
			return rango{}, err
		}
		r.desde = d
	}
	if v.Hasta == "" {
		r.abierta = true
		return r, nil
	}
	h, conHora, err := fechaDeVigencia("hasta", v.Hasta)
	if err != nil {
		return rango{}, err
	}
	if !conHora {
		// Fecha sola: cubre hasta el ultimo instante de ese dia. Lo contrario
		// deroga la norma a las 00:00 del dia que el BOE dice que sigue viva.
		h = h.AddDate(0, 0, 1).Add(-time.Nanosecond)
	}
	r.hasta = h
	if !r.sinInicio && r.desde.After(r.hasta) {
		return rango{}, fmt.Errorf("%w: desde=%q hasta=%q. Asi no esta vigente en ningun "+
			"instante; casi siempre es un error de tecleo en el fichero del paquete",
			ErrVigenciaInvertida, v.Desde, v.Hasta)
	}
	return r, nil
}

// cubre dice si el rango contiene el instante t. Desde es inclusivo y hasta
// tambien (ya normalizado al ultimo instante cubierto).
func (r rango) cubre(t time.Time) bool {
	if t.Before(r.desde) {
		return false
	}
	return r.abierta || !t.After(r.hasta)
}

// interseccion es el rango en el que se solapan dos vigencias. Es lo que hace
// falta para una obligacion: no puede estar en vigor fuera de la vigencia de su
// norma, aunque su propia vigencia diga otra cosa.
func (r rango) interseccion(o rango) rango {
	x := rango{desde: r.desde, hasta: r.hasta, abierta: r.abierta && o.abierta}
	if o.sinInicio {
		x.sinInicio = r.sinInicio
	} else if r.sinInicio || o.desde.After(r.desde) {
		x.desde, x.sinInicio = o.desde, false
	}
	switch {
	case r.abierta:
		x.hasta = o.hasta
	case o.abierta:
		x.hasta = r.hasta
	case o.hasta.Before(r.hasta):
		x.hasta = o.hasta
	}
	return x
}

// VigenteEn dice si algo con esta vigencia esta en vigor en el instante t.
//
// El instante ENTRA COMO DATO. El nucleo no lee el reloj (invariante 1), y aqui
// eso no es una regla de estilo: la vigencia decide que obligaciones salen en el
// expediente, y un expediente que se recalcula distinto manana no es verificable.
//
// Ante una vigencia que no se puede leer devuelve (false, err) y NUNCA
// (true, err): la respuesta segura a "no se que dice este fichero" es no dar la
// obligacion por vigente sin que alguien lo mire.
func (v Vigencia) VigenteEn(t time.Time) (bool, error) {
	r, err := v.interpretar()
	if err != nil {
		return false, err
	}
	if r.sinInicio {
		return false, fmt.Errorf("%w: hasta=%q. Sin desde no se sabe cuando empezo a "+
			"exigirse; declara vigencia.desde", ErrVigenciaSinDesde, v.Hasta)
	}
	return r.cubre(t), nil
}

// EnVigor dice si la obligacion o esta en vigor en el instante t, dentro de este
// paquete.
//
// Dos decisiones, y las dos se notan en el resultado:
//
//	herencia      una obligacion que no declara vigencia usa la del paquete. Es
//	              lo normal: la mayoria de los articulos nacen con su norma.
//	interseccion  la vigencia de la obligacion se corta con la del paquete. Una
//	              obligacion no puede exigirse antes de que su norma exista ni
//	              despues de que la deroguen, diga lo que diga su propio campo.
//	              Sin esto, un paquete mal escrito (o escrito de mala fe) alarga
//	              una obligacion mas alla de la norma que la sostiene.
func (p *Paquete) EnVigor(o Obligacion, t time.Time) (bool, error) {
	rp, err := p.Vigencia.interpretar()
	if err != nil {
		return false, fmt.Errorf("paquete %s: %w", p.URN, err)
	}
	ro, err := o.Vigencia.interpretar()
	if err != nil {
		return false, fmt.Errorf("paquete %s, obligacion %s: %w", p.URN, o.ID, err)
	}
	x := rp.interseccion(ro)
	if x.sinInicio {
		return false, fmt.Errorf("%w: paquete %s, obligacion %s. Ni la obligacion ni su "+
			"paquete declaran vigencia.desde", ErrVigenciaSinDesde, p.URN, o.ID)
	}
	return x.cubre(t), nil
}

// VigentesEn devuelve las obligaciones en vigor en el instante t, en el orden en
// que las declara el paquete.
//
// Existe porque el campo vigencia llevaba tiempo declarandose y no entrando en
// ningun calculo, o sea que una obligacion derogada seguia saliendo en la
// interfaz y en el expediente. Con normas que se modifican cada pocos anos eso
// no es una funcionalidad que falta, es una respuesta incorrecta.
//
// Quien la use para pintar una pantalla tiene ademas que DECIR que ha pasado:
// una obligacion que desaparece de la lista sin explicacion se lee como un
// fallo del producto, no como una derogacion.
func (p *Paquete) VigentesEn(t time.Time) ([]Obligacion, error) {
	var out []Obligacion
	for _, o := range p.Obligaciones {
		ok, err := p.EnVigor(o, t)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, o)
		}
	}
	return out, nil
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

// clasesE2E son las cinco maneras de implantar una obligacion de extremo a
// extremo (guia, Anexo B). La clase primaria es obligatoria: sin ella no se
// puede medir la profundidad ni decidir la cadena de implantacion.
var clasesE2E = map[string]bool{
	"observable": true, "documental": true, "procedimental": true,
	"notificatoria": true, "remediacion": true,
}

// Temporalidad es el reloj declarado de la obligacion, como datos: la
// primitiva del motor de ventana, su cadencia o limite, y el regimen.
type Temporalidad struct {
	Primitiva  string            `json:"primitiva"`          // puntual|periodica|continua|plazo|observacion|secuencia
	Hito       string            `json:"hito,omitempty"`     // nombre del hito (por defecto "ocurrencia" / "limite")
	Cadencia   string            `json:"cadencia,omitempty"` // periodica: ISO-8601 (P24M)
	Limite     string            `json:"limite,omitempty"`   // plazo: ISO-8601 (P10D, PT72H)
	Regimen    RegimenSpec       `json:"regimen"`
	Disparador map[string]string `json:"disparador,omitempty"` // p.ej. {"hecho": "ultima_auditoria"}
}

// RegimenSpec es el regimen de computo declarado por el paquete.
type RegimenSpec struct {
	Computo  string `json:"computo"`            // naturales | habiles
	Cierre   string `json:"cierre,omitempty"`   // exacto | fin_de_dia | (vacio = auto)
	Traslado string `json:"traslado,omitempty"` // ninguno | siguiente_habil
}

// Escalon es un paso de la cadena de escalado de la obligacion.
type Escalon struct {
	Tras string `json:"tras"` // ISO-8601, admite sufijo _antes (P60D_antes)
	A    string `json:"a"`    // rol destinatario
}

// Dorado es un caso de prueba derivado DEL TEXTO legal, no de la
// implementacion: si el motor y el dorado discrepan, gana el dorado.
type Dorado struct {
	Caso            string            `json:"caso"`
	Obligacion      string            `json:"obligacion"`
	Hechos          map[string]string `json:"hechos"` // clave -> fecha RFC3339 o 2006-01-02
	Esperado        EsperadoDorado    `json:"esperado"`
	CitaDelEsperado string            `json:"cita_del_esperado"`
}

// EsperadoDorado fija el resultado que el motor debe reproducir.
type EsperadoDorado struct {
	Vence string `json:"vence"`          // RFC3339
	Hito  string `json:"hito,omitempty"` // que ocurrencia (periodica: nombre#n)
}

// Obligacion es el atomo. Aqui va solo lo que el resto del sistema necesita
// del paquete; la temporalidad completa vive en ventana y la aplicabilidad en
// aplicabilidad.
type Obligacion struct {
	ID       string `json:"id"`
	Articulo string `json:"articulo"`
	// Titulo es la etiqueta legible de la obligacion, la que sale en una lista
	// de controles. OPCIONAL a proposito: hacerla obligatoria obligaria a
	// reescribir hoy los 30 paquetes del corpus, y un formato que se rompe al
	// crecer no lo adopta nadie. Cuando falta, TituloLegible da el respaldo.
	//
	// Lleva el limite de la frontera legal como cualquier otra prosa: un titulo
	// es justo donde alguien pega el enunciado de un control de un catalogo de
	// pago, y lo pega de buena fe, porque "es solo el titulo".
	Titulo     string        `json:"titulo,omitempty"`
	TextoLegal string        `json:"texto_legal,omitempty"` // vacio en referencial y delegado
	Cita       string        `json:"cita"`
	Vigencia   Vigencia      `json:"vigencia"`
	Entregable string        `json:"entregable,omitempty"` // ref a Plantilla.ID
	Recursos   []TipoRecurso `json:"recursos,omitempty"`
	// Delegado dice que herramienta externa comprueba esto. Obligatorio y solo
	// permitido en paquetes de clase Delegado.
	Delegado  string   `json:"delegado,omitempty"`
	Preguntas []string `json:"preguntas,omitempty"` // IDs de Pregunta que la desbloquean

	// La extension e2e (Anexo B): clase primaria obligatoria, facetas
	// opcionales, reloj declarado y cadena de escalado.
	ClaseE2E     string        `json:"clase_e2e"`
	Facetas      []string      `json:"facetas,omitempty"`
	Temporalidad *Temporalidad `json:"temporalidad,omitempty"`
	Escalado     []Escalon     `json:"escalado,omitempty"`
}

// TituloLegible es la etiqueta que se ensena cuando hay que ensenar una sola
// linea de la obligacion. Devuelve siempre algo derivado de lo que hay, en este
// orden y por esta razon:
//
//	Titulo    lo que escribio quien autoro el paquete, si lo escribio.
//	Articulo  el localizador. En la practica ya trae etiqueta dentro ("Anexo II
//	          4.2.5 Mecanismo de autenticacion (usuarios externos) [op.acc.5]"),
//	          asi que es un respaldo legible, aunque no sea un titulo.
//	ID        el identificador. Feo, pero unico y citable: mejor eso que un
//	          hueco en blanco en una tabla de controles.
//
// Devuelve cadena vacia solo si la obligacion no tiene ninguna de las tres, y
// eso el linter ya no lo deja cargar (ErrObligacionSinID). Quien pinte esto no
// tiene que inventarse un texto de relleno: si llega vacio, es un fallo de
// carga, no una obligacion sin nombre.
func (o Obligacion) TituloLegible() string {
	switch {
	case o.Titulo != "":
		return o.Titulo
	case o.Articulo != "":
		return o.Articulo
	default:
		return o.ID
	}
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
	// Aplicabilidad son las reglas que deciden a quien alcanza cada
	// obligacion, en el dialecto Datalog estratificado. Van aqui, en el
	// fichero de datos, y no en codigo Go: es lo que hace cierto el
	// invariante 2 y lo que permite que el corpus se actualice con un
	// fichero firmado en vez de con una release del binario.
	Aplicabilidad Aplicabilidad `json:"aplicabilidad,omitempty"`
	// Dorados se carga desde pruebas/*.json del directorio del paquete; no se
	// declara en paquete.json.
	Dorados []Dorado `json:"-"`
}

// ---------------------------------------------------------------------------
// La frontera legal, campo a campo.
//
// EL AGUJERO QUE ESTO CIERRA. El limite de texto de un paquete referencial solo
// miraba texto_legal. Los otros veinte y pico campos de texto libre del formato
// (la ayuda de un atributo, la descripcion de una entidad, el texto de una
// pregunta, el titulo de una plantilla) no los miraba nadie, asi que el
// enunciado de un control de ISO, PCI DSS, SOC 2 o TISAX entraba por cualquiera
// de ellos y el linter no decia nada. Es el mismo agujero que la clase fuera de
// rango, por otra puerta: la unica frontera que este proyecto declara no
// negociable se esquivaba escribiendo el texto en el campo de al lado.
//
// EL CRITERIO, que es lo que hay que poder discutir. Cada campo de texto libre
// se clasifica en uno de dos tipos, y el tipo decide el limite:
//
//	prosa       texto escrito para que lo lea una persona. Es donde cabe el
//	            enunciado de un control, y es donde se cuela sin mala intencion,
//	            porque "es solo la ayuda" o "es solo el titulo". Limite corto.
//	referencia  identificador, localizador, clave de formulario, fecha, enlace o
//	            declaracion de licencia. No sustituye al texto normativo, y por
//	            eso "CAT/DEMO 9999:2026 A.5.1" tiene que seguir valiendo. Pero
//	            sigue siendo texto libre, asi que lleva techo, no barra libre.
//	derivacion  un solo campo: la cita_del_esperado de un dorado, que es el
//	            razonamiento del autor y por eso es legitimamente largo.
//
// NADIE QUEDA FUERA, y eso lo vigila un test: camposDeTexto tiene que enumerar
// TODOS los campos de cadena del formato. Si manana alguien anade un campo y se
// olvida de clasificarlo, el test de exhaustividad lo dice, porque un campo
// nuevo sin clasificar es exactamente por donde volveria a entrar el texto.
//
// LO QUE NO CIERRA, dicho para que conste. El limite es POR CAMPO: quien quiera
// copiar un catalogo entero puede repartirlo entre la ayuda, la descripcion y el
// titulo de cien obligaciones. Contra eso no hay linter que valga, hay revision
// del paquete; lo que el limite corta es el caso real, que es pegar el control
// de un tiron en el campo que tenia a mano.
// ---------------------------------------------------------------------------

// tipoCampo clasifica un campo de texto libre por lo que puede llevar dentro.
type tipoCampo uint8

const (
	prosa tipoCampo = iota
	referencia
	derivacion
)

func (t tipoCampo) limite() int {
	switch t {
	case prosa:
		return LimiteTextoReferencial
	case derivacion:
		return LimiteDerivacionReferencial
	default:
		return LimiteCitaReferencial
	}
}

func (t tipoCampo) centinela() error {
	if t == prosa {
		return ErrTextoRedistribuido
	}
	return ErrCitaDesbordada
}

// campoTexto es un campo de texto libre ya localizado dentro del paquete.
type campoTexto struct {
	// Campo es la ruta canonica en el formato (Paquete.Obligaciones[].Titulo).
	// Es la que casa el test de exhaustividad con la estructura de datos.
	Campo string
	// Donde es el sitio concreto (obligacion demo.auditoria_bienal), para que
	// el error diga que fila hay que arreglar y no solo que campo.
	Donde string
	Valor string
	Tipo  tipoCampo
}

// camposDeTexto enumera TODOS los campos de texto libre del paquete con su
// clasificacion. Es la unica lista, y el veredicto de cada campo esta escrito
// aqui al lado del campo, no en un documento aparte que nadie abre.
//
// Emite tambien los campos vacios: el linter no se entera de la diferencia
// (una cadena vacia nunca pasa de ningun limite) y el test de exhaustividad
// necesita verlos para comprobar que no falta ninguno.
func camposDeTexto(p *Paquete) []campoTexto {
	var cs []campoTexto
	uno := func(campo, donde, valor string, tipo tipoCampo) {
		cs = append(cs, campoTexto{Campo: campo, Donde: donde, Valor: valor, Tipo: tipo})
	}
	varios := func(campo, donde string, valores []string, tipo tipoCampo) {
		for _, v := range valores {
			uno(campo, donde, v, tipo)
		}
	}
	// Un mapa se recorre ordenado por clave: el linter tiene que dar los mismos
	// errores en el mismo orden en dos ejecuciones, o deja de ser comparable.
	mapa := func(campo, donde string, m map[string]string, tipo tipoCampo) {
		claves := make([]string, 0, len(m))
		for k := range m {
			claves = append(claves, k)
		}
		sort.Strings(claves)
		for _, k := range claves {
			uno(campo, donde+", clave "+k, m[k], tipo)
		}
	}

	// Cabecera del paquete. Todo referencia: el URN y la version son claves, la
	// fuente es un enlace, y la licencia es la declaracion de derechos, que es
	// justo donde el paquete TIENE que poder explicarse (el corpus de hoy gasta
	// 228 caracteres en explicar que un referencial no trae texto).
	donde := "paquete " + p.URN
	uno("Paquete.URN", donde, p.URN, referencia)
	uno("Paquete.Version", donde, p.Version, referencia)
	uno("Paquete.Licencia", donde, p.Licencia, referencia)
	uno("Paquete.Fuente", donde, p.Fuente, referencia)
	uno("Paquete.Vigencia.Desde", donde, p.Vigencia.Desde, referencia)
	uno("Paquete.Vigencia.Hasta", donde, p.Vigencia.Hasta, referencia)
	varios("Paquete.Escalas[]", donde, p.Escalas, referencia)

	for _, te := range p.Entidades {
		d := "entidad " + te.Nombre
		uno("Paquete.Entidades[].Nombre", d, te.Nombre, referencia)
		// Descripcion es PROSA: se ensena en el formulario y cabe entera la
		// definicion de alcance de un catalogo de pago.
		uno("Paquete.Entidades[].Descripcion", d, te.Descripcion, prosa)
		for _, a := range te.Atributos {
			da := d + ", atributo " + a.Nombre
			uno("Paquete.Entidades[].Atributos[].Nombre", da, a.Nombre, referencia)
			varios("Paquete.Entidades[].Atributos[].Valores[]", da, a.Valores, referencia)
			uno("Paquete.Entidades[].Atributos[].Escala", da, a.Escala, referencia)
			// Ayuda es PROSA, y es el campo mas tentador de todos: explicar un
			// control copiando el control es lo que sale solo al autorar.
			uno("Paquete.Entidades[].Atributos[].Ayuda", da, a.Ayuda, prosa)
			uno("Paquete.Entidades[].Atributos[].Cita", da, a.Cita, referencia)
		}
	}

	for _, q := range p.Preguntas {
		d := "pregunta " + q.ID
		uno("Paquete.Preguntas[].ID", d, q.ID, referencia)
		// Texto y Ayuda son PROSA: una pregunta de alcance se escribe con
		// palabras propias, no transcribiendo el requisito que la motiva.
		uno("Paquete.Preguntas[].Texto", d, q.Texto, prosa)
		uno("Paquete.Preguntas[].Ayuda", d, q.Ayuda, prosa)
		uno("Paquete.Preguntas[].Cita", d, q.Cita, referencia)
		uno("Paquete.Preguntas[].Entidad", d, q.Entidad, referencia)
		uno("Paquete.Preguntas[].Atributo", d, q.Atributo, referencia)
		varios("Paquete.Preguntas[].Desbloquea[]", d, q.Desbloquea, referencia)
	}

	for _, o := range p.Obligaciones {
		d := "obligacion " + o.ID
		uno("Paquete.Obligaciones[].ID", d, o.ID, referencia)
		// Articulo es PROSA aunque parezca un localizador: en el corpus real
		// lleva dentro la etiqueta del control ("Anexo II 4.2.5 Mecanismo de
		// autenticacion (usuarios externos) [op.acc.5]"), o sea que un catalogo
		// de pago cabria ahi tal cual.
		uno("Paquete.Obligaciones[].Articulo", d, o.Articulo, prosa)
		uno("Paquete.Obligaciones[].Titulo", d, o.Titulo, prosa)
		uno("Paquete.Obligaciones[].TextoLegal", d, o.TextoLegal, prosa)
		uno("Paquete.Obligaciones[].Cita", d, o.Cita, referencia)
		uno("Paquete.Obligaciones[].Vigencia.Desde", d, o.Vigencia.Desde, referencia)
		uno("Paquete.Obligaciones[].Vigencia.Hasta", d, o.Vigencia.Hasta, referencia)
		uno("Paquete.Obligaciones[].Entregable", d, o.Entregable, referencia)
		uno("Paquete.Obligaciones[].Delegado", d, o.Delegado, referencia)
		uno("Paquete.Obligaciones[].ClaseE2E", d, o.ClaseE2E, referencia)
		varios("Paquete.Obligaciones[].Facetas[]", d, o.Facetas, referencia)
		varios("Paquete.Obligaciones[].Preguntas[]", d, o.Preguntas, referencia)
		for _, r := range o.Recursos {
			uno("Paquete.Obligaciones[].Recursos[]", d, string(r), referencia)
		}
		if t := o.Temporalidad; t != nil {
			uno("Paquete.Obligaciones[].Temporalidad.Primitiva", d, t.Primitiva, referencia)
			uno("Paquete.Obligaciones[].Temporalidad.Hito", d, t.Hito, referencia)
			uno("Paquete.Obligaciones[].Temporalidad.Cadencia", d, t.Cadencia, referencia)
			uno("Paquete.Obligaciones[].Temporalidad.Limite", d, t.Limite, referencia)
			uno("Paquete.Obligaciones[].Temporalidad.Regimen.Computo", d, t.Regimen.Computo, referencia)
			uno("Paquete.Obligaciones[].Temporalidad.Regimen.Cierre", d, t.Regimen.Cierre, referencia)
			uno("Paquete.Obligaciones[].Temporalidad.Regimen.Traslado", d, t.Regimen.Traslado, referencia)
			mapa("Paquete.Obligaciones[].Temporalidad.Disparador[]", d, t.Disparador, referencia)
		}
		for _, esc := range o.Escalado {
			uno("Paquete.Obligaciones[].Escalado[].Tras", d, esc.Tras, referencia)
			uno("Paquete.Obligaciones[].Escalado[].A", d, esc.A, referencia)
		}
	}

	for _, pl := range p.Plantillas {
		d := "plantilla " + pl.ID
		uno("Paquete.Plantillas[].ID", d, pl.ID, referencia)
		// Titulo de plantilla es PROSA: es el nombre del entregable que ve el
		// auditor, y ahi cabe el enunciado del requisito que lo pide.
		uno("Paquete.Plantillas[].Titulo", d, pl.Titulo, prosa)
		uno("Paquete.Plantillas[].Cita", d, pl.Cita, referencia)
		for _, c := range pl.Campos {
			uno("Paquete.Plantillas[].Campos[].Nombre", d, c.Nombre, referencia)
			uno("Paquete.Plantillas[].Campos[].Origen", d, c.Origen, referencia)
		}
	}

	for _, dor := range p.Dorados {
		d := "dorado " + dor.Caso
		// Caso es PROSA: describe el supuesto con palabras propias. Los dorados
		// viajan dentro del paquete, asi que cuentan para la frontera.
		uno("Paquete.Dorados[].Caso", d, dor.Caso, prosa)
		uno("Paquete.Dorados[].Obligacion", d, dor.Obligacion, referencia)
		mapa("Paquete.Dorados[].Hechos[]", d, dor.Hechos, referencia)
		uno("Paquete.Dorados[].Esperado.Vence", d, dor.Esperado.Vence, referencia)
		uno("Paquete.Dorados[].Esperado.Hito", d, dor.Esperado.Hito, referencia)
		// CitaDelEsperado es DERIVACION: el porque de la fecha esperada, con la
		// cuenta hecha. Techo propio y alto, ver LimiteDerivacionReferencial.
		uno("Paquete.Dorados[].CitaDelEsperado", d, dor.CitaDelEsperado, derivacion)
	}
	return cs
}

// validarFronteraLegal aplica el limite de la clase a todos los campos de texto.
//
// Solo corre en las clases que NO pueden redistribuir texto de un tercero:
//
//	Referencial  ISO, PCI DSS, SOC 2, TISAX. El cliente aporta su copia
//	             licenciada; el paquete solo puede traer identificadores.
//	Delegado     CIS, STIG. No se distribuye nada, y por eso texto_legal tiene
//	             que estar VACIO (eso se comprueba aparte, en Validar). El resto
//	             de campos llevan el mismo limite que un referencial: "nada de
//	             texto" no puede significar "ni siquiera una etiqueta", porque
//	             entonces la obligacion no se puede ni listar.
//
// Importado, Transcrito y Propio no llevan limite: en los tres hay derecho a
// redistribuir el texto entero (dominio publico, art. 13 TRLPI y Decision
// 2011/833/UE, o datos del propio proyecto).
func (p *Paquete) validarFronteraLegal(anotar func(error)) {
	switch {
	case p.Clase == Referencial || p.Clase == Delegado:
	case !p.Clase.Valida():
		// Una clase que no existe no acredita ningun derecho de redistribucion,
		// asi que se le aplica la frontera mas estricta. El paquete ademas no
		// carga, porque Validar rechaza la clase fuera de rango; esto esta aqui
		// para que la frontera no dependa de que ese otro chequeo siga vivo.
	default:
		return
	}
	for _, c := range camposDeTexto(p) {
		lim := c.Tipo.limite()
		if len(c.Valor) <= lim {
			continue
		}
		arreglo := "ISO, PCI DSS, SOC 2, TISAX y CIS no autorizan la redistribucion de su " +
			"texto: identificador y titulo corto, nada mas"
		if c.Tipo != prosa {
			arreglo = "un campo de referencia apunta al texto, no lo lleva dentro: " +
				"deja el localizador y quita el parrafo"
		}
		anotar(fmt.Errorf("%w: %s, campo %s con %d caracteres (limite %d en un paquete "+
			"de clase %s). %s",
			c.Tipo.centinela(), c.Donde, c.Campo, len(c.Valor), lim, p.Clase, arreglo))
	}
}

// validarVigencias comprueba que las fechas de vigencia se pueden leer y que no
// van al reves. Se comprueba en el linter y no al usarlas porque una vigencia
// ilegible no es un caso raro de tiempo de ejecucion: es un fichero de datos de
// un tercero mal escrito, y el sitio de pararlo es la carga.
func (p *Paquete) validarVigencias(anotar func(error)) {
	if p.Vigencia.Desde == "" {
		anotar(fmt.Errorf("%w: paquete %s. Sin vigencia.desde no se sabe desde cuando se "+
			"exige nada de este paquete", ErrVigenciaSinDesde, p.URN))
	}
	if _, err := p.Vigencia.interpretar(); err != nil {
		anotar(fmt.Errorf("paquete %s: %w", p.URN, err))
	}
	for _, o := range p.Obligaciones {
		if _, err := o.Vigencia.interpretar(); err != nil {
			anotar(fmt.Errorf("obligacion %s: %w", o.ID, err))
		}
	}
}

// ---------------------------------------------------------------------------
// El linter. Rechaza lo que no es seguro en vez de ejecutarlo a ver que pasa.
// ---------------------------------------------------------------------------

// Validar comprueba las invariantes del paquete. Devuelve todos los fallos, no
// solo el primero, porque quien escribe un paquete quiere arreglarlos de una vez.
func (p *Paquete) Validar() []error {
	var errs []error
	e := func(f string, a ...any) { errs = append(errs, fmt.Errorf(f, a...)) }
	// anotar es para los errores con centinela, que se comprueban con errors.Is
	// y no buscando una subcadena del mensaje. Ese patron ya dio aqui siete
	// tests en verde con el fallo delante: uno buscaba "clase" y lo encontraba
	// dentro de "clase_e2e".
	anotar := func(err error) { errs = append(errs, err) }

	// La clase, ANTES que nada, porque de ella depende que limites se aplican.
	//
	// HALLAZGO DEL FRENTE DE CORPUS, y es de la frontera legal, no de estilo:
	// el switch por clase de mas abajo tiene un default, asi que una clase
	// fuera de rango no era referencial y por tanto no tenia limite de texto.
	// Un paquete con "clase": 9 y 200 caracteres de texto de ISO validaba
	// limpio. La unica frontera que este proyecto declara no negociable se
	// esquivaba escribiendo un numero distinto en un fichero JSON.
	//
	// Se comprueba aqui arriba y no dentro del switch para que sea imposible
	// llegar a las comprobaciones por clase con una clase que no existe.
	if !p.Clase.Valida() {
		e("paquete %s: clase %d fuera de rango (0 importado, 1 transcrito, 2 referencial, "+
			"3 delegado, 4 propio). Una clase desconocida no puede cargar: es la que decide "+
			"si se puede redistribuir el texto normativo", p.URN, uint8(p.Clase))
	}

	// La frontera legal y las vigencias, antes que la forma. Un paquete puede
	// tener veinte fallos de forma; el que hay que ver primero en la salida es
	// el que redistribuye texto que no se puede redistribuir.
	p.validarFronteraLegal(anotar)
	p.validarVigencias(anotar)

	p.validarAplicabilidad(e)

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
		if o.ID == "" {
			anotar(fmt.Errorf("%w: la obligacion %q del paquete %s no se puede citar, ni "+
				"referenciar desde una pregunta, ni seguir en el expediente",
				ErrObligacionSinID, o.TituloLegible(), p.URN))
		}
		if obl[o.ID] {
			e("obligacion %s duplicada", o.ID)
		}
		obl[o.ID] = true
		if o.Cita == "" {
			e("obligacion %s sin cita normativa", o.ID)
		}
		if !clasesE2E[o.ClaseE2E] {
			e("obligacion %s: clase_e2e %q invalida u omitida (observable, documental, "+
				"procedimental, notificatoria, remediacion). Sin clase no hay medida "+
				"de profundidad e2e", o.ID, o.ClaseE2E)
		}
		for _, f := range o.Facetas {
			if !clasesE2E[f] {
				e("obligacion %s: faceta %q invalida", o.ID, f)
			}
		}
		for _, esc := range o.Escalado {
			if esc.Tras == "" || esc.A == "" {
				e("obligacion %s: escalon sin plazo o sin destinatario", o.ID)
			}
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
		// El limite de texto de la clase NO se comprueba aqui: lo hace
		// validarFronteraLegal sobre TODOS los campos de texto del paquete, no
		// solo sobre texto_legal. Mirar un campo de veinte era el agujero.
		switch p.Clase {
		case Referencial:
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
	// Todo reloj exige sus dorados: minimo 3 por obligacion con temporalidad,
	// derivados del texto. Y ningun dorado puede apuntar a una obligacion que
	// no existe.
	porObl := map[string]int{}
	for _, d := range p.Dorados {
		if !obl[d.Obligacion] {
			e("dorado %q apunta a la obligacion %s, que no existe", d.Caso, d.Obligacion)
		}
		if d.CitaDelEsperado == "" {
			e("dorado %q sin cita_del_esperado: el esperado se deriva del texto, no de la implementacion", d.Caso)
		}
		porObl[d.Obligacion]++
	}
	for _, o := range p.Obligaciones {
		if o.Temporalidad != nil && porObl[o.ID] < 3 {
			e("obligacion %s declara reloj y tiene %d dorados (minimo 3: normal, "+
				"borde de calendario, y ocurrencia u variante)", o.ID, porObl[o.ID])
		}
	}
	return errs
}

// ---------------------------------------------------------------------------
// Lo derivado. Nada de esto se escribe por norma: sale del paquete.
// ---------------------------------------------------------------------------

// Peticion es UNA norma pidiendo un dato, con la cita y la ayuda que da ELLA.
//
// Es la respuesta a "por que me piden este dato", que es la primera pregunta de
// quien rellena un formulario de cumplimiento y la unica que convierte "rellena
// esto" en trabajo que se entiende.
type Peticion struct {
	Paquete string `json:"paquete"`
	Cita    string `json:"cita"`
	Ayuda   string `json:"ayuda,omitempty"`
}

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
	// Peticiones dice POR QUE lo pide cada uno, una entrada por paquete y en
	// orden de URN. Ayuda y Cita de arriba son las de la primera, que es lo
	// que habia antes de esto y se mantiene para no romper a quien ya las lee.
	Peticiones []Peticion
}

// EsquemaUI deriva los formularios de la interfaz de los paquetes instalados.
// Un atributo pedido por tres normas se pregunta una vez y se dice quien lo pide.
//
// HALLAZGO (nucleo/pantalla, caso dorado): cuando dos paquetes declaran el mismo
// atributo, el PRIMERO que se recorre fija la etiqueta, el tipo, los valores, la
// ayuda y la cita, y los demas solo suman su URN a Paquetes. Como el cargador
// recorre un directorio, ese "primero" no estaba garantizado: el mismo corpus
// daba formularios distintos entre ejecuciones, con otra ayuda y otra cita. Se
// recorre en orden de URN para que el resultado sea estable. Lo vigila
// TestElModeloNoDependeDelOrdenDeLosPaquetes, comprobado por mutacion.
//
// LA PERDIDA DE INFORMACION, ya cerrada. De las tres normas que piden el dato
// solo sobrevivia la ayuda y la cita de UNA, la de URN menor. Paquetes decia
// quienes eran, pero no por que lo pedia cada una, asi que al comprador que
// pregunta "por que me piden este dato" se le respondia con el articulo de una
// de tres, elegido por orden alfabetico. Ahora cada campo lleva Peticiones, una
// entrada por paquete con SU cita y SU ayuda.
//
// El arreglo es ADITIVO a proposito: Ayuda, Cita y Paquetes siguen exactamente
// donde estaban y significando lo mismo. Quitarlos habria roto a quien ya
// compilaba contra esta forma, y un frente no le cambia el suelo a otro.
func EsquemaUI(ps []*Paquete) []CampoUI {
	idx := map[string]*CampoUI{}
	var orden []string
	enOrden := append([]*Paquete(nil), ps...)
	sort.SliceStable(enOrden, func(i, j int) bool { return enOrden[i].URN < enOrden[j].URN })
	for _, p := range enOrden {
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
				// Una peticion por PAQUETE, no por declaracion. Un paquete que
				// declare dos veces la misma entidad no puede aparecer dos
				// veces diciendo dos cosas distintas sobre el mismo dato.
				yaPide := false
				for _, x := range c.Peticiones {
					if x.Paquete == p.URN {
						yaPide = true
						break
					}
				}
				if !yaPide {
					c.Peticiones = append(c.Peticiones, Peticion{
						Paquete: p.URN, Cita: a.Cita, Ayuda: a.Ayuda,
					})
				}
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
