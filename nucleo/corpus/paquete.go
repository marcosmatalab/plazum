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

	"plazum/nucleo/ventana"
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

// ---------------------------------------------------------------------------
// La licencia de la FUENTE y la atribucion, que son dos cosas distintas.
//
// POR QUE NO BASTA CON `clase` Y CON `licencia`. La clase dice QUE se puede
// distribuir del paquete y el linter la hace cumplir. `licencia` es texto libre
// donde el autor explica el regimen con sus palabras. Ninguna de las dos
// responde la pregunta que de verdad obliga: la Decision 2011/833/UE autoriza
// reutilizar el DOUE **con atribucion**, y una atribucion que vive en la cabeza
// de quien escribio el paquete no es una atribucion. Tiene que ser un dato,
// tiene que viajar dentro del paquete y tiene que poder salir en pantalla.
//
// Por eso son DOS campos y no uno:
//
//	LicenciaFuente  el regimen, de un vocabulario CERRADO. Es lo que se
//	                comprueba contra la clase. Una fuente nueva no entra
//	                escribiendo una cadena distinta: entra con su constante
//	                aqui y su fila en docs/LICENCIAS.md, igual que una
//	                dependencia entra por DEPENDENCIAS.md.
//	Atribucion      el aviso literal que hay que ENSENAR a quien usa el
//	                producto. Es texto, no clave de catalogo, y no se traduce:
//	                un aviso de derechos parafraseado por la interfaz deja de
//	                ser el aviso.
// ---------------------------------------------------------------------------

// LicenciaFuente es el regimen de derechos de la fuente de la que sale el
// contenido del paquete. Vocabulario cerrado a proposito: ver arriba.
type LicenciaFuente string

const (
	// BOETRLPI13: disposicion legal espanola publicada en el BOE. El art. 13
	// del texto refundido de la Ley de Propiedad Intelectual deja las
	// disposiciones legales fuera de la proteccion; las condiciones de
	// reutilizacion del BOE exigen citar la fuente.
	BOETRLPI13 LicenciaFuente = "boe-trlpi-13"
	// DOUEDecision2011833: texto publicado en el DOUE o en EUR-Lex. La
	// Decision 2011/833/UE autoriza la reutilizacion CON ATRIBUCION, y esa
	// atribucion es obligacion, no cortesia.
	DOUEDecision2011833 LicenciaFuente = "doue-decision-2011-833"
	// DominioPublicoEEUU: obra de la administracion federal de los Estados
	// Unidos, sin derechos de autor federales.
	DominioPublicoEEUU LicenciaFuente = "dominio-publico-eeuu"
	// SinLicenciaDeTexto: no hay derecho a redistribuir el texto y por eso el
	// paquete no lo lleva. Identificador y titulo corto; la copia licenciada
	// la aporta el cliente. Es el regimen del estrato referencial.
	SinLicenciaDeTexto LicenciaFuente = "sin-licencia-de-texto"
	// LaTieneLaHerramienta: no se distribuye nada porque la licencia del
	// contenido la tiene la herramienta externa que lo comprueba. Es el
	// regimen del estrato delegado.
	LaTieneLaHerramienta LicenciaFuente = "la-tiene-la-herramienta"
	// RISPConAtribucion: reutilizacion de informacion del sector publico,
	// permitida con atribucion y sin desnaturalizar el contenido.
	RISPConAtribucion LicenciaFuente = "risp-con-atribucion"
	// DelProyecto: datos creados por este proyecto. No hay tercero con
	// derechos, y la atribucion es la del propio proyecto.
	DelProyecto LicenciaFuente = "del-proyecto"
)

// licenciasPorClase dice que regimenes admite cada estrato. La coherencia se
// comprueba porque los dos campos se pueden escribir por separado: un paquete
// que se declara referencial y dice traer el texto del BOE esta mintiendo en
// uno de los dos sitios, y hay que pararlo antes de saber en cual.
var licenciasPorClase = map[Clase][]LicenciaFuente{
	Importado:   {DominioPublicoEEUU},
	Transcrito:  {BOETRLPI13, DOUEDecision2011833},
	Referencial: {SinLicenciaDeTexto},
	Delegado:    {LaTieneLaHerramienta},
	Propio:      {DelProyecto, RISPConAtribucion},
}

// licenciasProhibidas es LISTA NEGRA, no lista de pendientes.
//
// Estan aqui, con su motivo, porque alguien las va a volver a proponer y tiene
// que encontrarse el porque en vez de una casilla vacia. El motivo sale en el
// error del linter: quien lo intente lee por que no, no "valor invalido".
//
// El razonamiento completo, en docs/LICENCIAS.md.
var licenciasProhibidas = map[LicenciaFuente]string{
	"cc-by-nc-nd": "el NC prohibe el uso comercial y este producto se vende; el ND prohibe " +
		"cualquier adaptacion, y un paquete de corpus ES una adaptacion. Es la licencia de " +
		"los CIS Controls",
	"cc-by-nc-sa": "el NC prohibe el uso comercial y el SA obliga a relicenciar lo derivado. " +
		"Es la licencia de los CIS Benchmarks: se leen con una herramienta que ya tiene la " +
		"licencia (clase delegado), no se copian",
	"cc-by-nd": "el ND prohibe cualquier adaptacion, y un paquete de corpus ES una " +
		"adaptacion. Es la licencia del marco gratuito del SCF",
	"repositorio-de-terceros": "la licencia de un repositorio no alcanza al contenido que " +
		"quien lo subio no poseia. Un MIT o un Apache sobre un volcado de una norma ajena no " +
		"da ningun derecho sobre la norma. Solo fuente primaria",
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

// MinimoDelMotivoDeSubconjunto es el SUELO de Dorado.SubconjuntoPorque, o sea
// el unico limite del formato que aprieta por abajo.
//
// Por que hay suelo: el campo existe para que renunciar a la exhaustividad
// CUESTE. Un campo de texto libre sin suelo se rellena con "n/a", "TODO" o
// "por ahora", y entonces el opt-out vuelve a ser el booleano que no queremos,
// solo que escrito con letras.
//
// Por que 40 y no otro numero: un motivo util tiene que nombrar dos cosas, que
// hito queda fuera y por que, y la frase mas corta que hace las dos ronda los
// cuarenta caracteres ("los otros tres hitos los fija el art. 5.2" son 41). Por
// debajo de ahi no cabe un argumento, solo una etiqueta. Es un suelo bajo a
// proposito: lo que corta es el relleno, no al autor que se explica.
//
// El techo del mismo campo es LimiteDerivacionReferencial, por el mismo motivo
// que la cita_del_esperado: es razonamiento del autor, no texto de un tercero.
const MinimoDelMotivoDeSubconjunto = 40

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
	// ErrLecturaVigenciaSinCita: una lectura divergente de la vigencia sin la
	// cita de donde sale. Sin cita no es una lectura, es una opinion, y el
	// producto entero se apoya en que cada fecha se pueda seguir hasta su
	// fuente.
	ErrLecturaVigenciaSinCita = errors.New("lectura divergente de vigencia sin cita")
	// ErrLecturaVigenciaVacia: una lectura que no mueve ni el desde ni el
	// hasta. No dice nada y ocupa el sitio de la que si diria algo.
	ErrLecturaVigenciaVacia = errors.New("lectura divergente de vigencia sin fecha")
	// ErrLecturaVigenciaQueNoDiverge: una lectura identica a la declarada. Se
	// leeria como un desacuerdo donde no lo hay, que es peor que no decir nada.
	ErrLecturaVigenciaQueNoDiverge = errors.New("lectura divergente de vigencia que no diverge")
	// ErrObligacionSinID: una obligacion sin identificador no se puede citar,
	// ni referenciar desde una pregunta, ni seguir en el expediente.
	ErrObligacionSinID = errors.New("obligacion sin id")
	// ErrSinLicenciaFuente: el paquete no declara de que regimen sale su
	// contenido. Sin eso no se sabe si se puede redistribuir ni a quien hay
	// que atribuirlo, y las dos cosas son obligaciones, no metadatos.
	ErrSinLicenciaFuente = errors.New("paquete sin licencia_fuente")
	// ErrLicenciaFuenteDesconocida: un regimen que no esta en el vocabulario.
	ErrLicenciaFuenteDesconocida = errors.New("licencia_fuente fuera del vocabulario")
	// ErrLicenciaProhibida: un regimen de la lista negra. No es "todavia no".
	ErrLicenciaProhibida = errors.New("licencia_fuente prohibida en este proyecto")
	// ErrLicenciaFuenteIncoherente: el regimen declarado no es de los que
	// admite la clase. Uno de los dos campos miente y no se sabe cual.
	ErrLicenciaFuenteIncoherente = errors.New("licencia_fuente incoherente con la clase")
	// ErrSinAtribucion: el paquete no trae el aviso que hay que ensenar. Un
	// LICENCIAS.md en el repositorio no cumple una obligacion de atribucion
	// hacia quien USA el producto: el aviso tiene que viajar con el paquete.
	ErrSinAtribucion = errors.New("paquete sin atribucion")
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

	// Alternativas son lecturas discrepantes de la propia VIGENCIA, con su cita.
	//
	// POR QUE HACIA FALTA. El mecanismo de divergencias existia solo para el
	// COMPUTO (HitoSpec.Alternativas: la norma da dos cifras y no dice cual
	// rige). El 27-08-2026 aparecio la otra mitad del problema, que es la
	// discrepancia sobre DESDE CUANDO obliga algo: el AI Act publicado en el
	// DOUE dice una fecha de aplicacion y un acuerdo politico posterior que
	// todavia no se ha publicado dice otra. Las dos son ciertas como lo que
	// son, y ninguna de las dos se puede callar: la del DOUE es la unica que
	// vincula hoy, y la del acuerdo es la que decide si el cliente arranca un
	// proyecto de doce meses este trimestre o el que viene.
	//
	// LO QUE MANDA ES Desde/Hasta, SIEMPRE. Una alternativa no cambia NUNCA lo
	// que devuelve VigenteEn: se calcula lo declarado, y la divergencia se
	// ensena. El valor cero (nil, sin alternativas) es la lectura publicada a
	// secas, que es la restrictiva y la comprobable: el invariante 8 al derecho.
	// Una alternativa que moviera el calculo convertiria un rumor de prensa en
	// derecho aplicado, que es exactamente lo contrario de este producto.
	Alternativas []LecturaVigencia `json:"alternativas,omitempty"`
}

// LecturaVigencia es una interpretacion discrepante de desde (o hasta) cuando
// obliga algo. Necesita cita SIEMPRE: sin ella no es una lectura, es una
// opinion, y el linter la rechaza.
type LecturaVigencia struct {
	ID    string `json:"id"`
	Desde string `json:"desde,omitempty"`
	Hasta string `json:"hasta,omitempty"`
	Cita  string `json:"cita"`

	// Espera nombra el item de vigilancia del que cuelga esta lectura: el
	// hecho de FUERA que, cuando ocurra, obliga a revisarla.
	//
	// POR QUE EXISTE, y lo pago este proyecto en un dia. El 27-08-2026 este
	// corpus afirmaba que dos fechas del AI Act salidas del omnibus digital
	// "NO VINCULAN porque no estan publicadas en el DOUE". Llevaban publicadas
	// treinta y cuatro dias (Reglamento (UE) 2026/1744, de 8 de julio de 2026).
	// Lo encontro una revision, no una puerta.
	//
	// Una lectura divergente es, por definicion, una apuesta sobre algo que
	// todavia no ha pasado. Sin decir DE QUE cuelga, nadie sabe cuando dejo de
	// ser cierta, y una divergencia que envejece mal es peor que no tenerla:
	// se ensena al cliente al lado de la fecha que vincula.
	//
	// Vacio es legal: hay lecturas que no esperan a nada (dos formas de contar
	// el mismo plazo no dependen de ningun evento). Lo que NO es legal es
	// nombrar un item que no existe, ni que exista un item que no nombre
	// ninguna lectura: las dos direcciones se comprueban en vigilancia_test.go.
	Espera string `json:"espera,omitempty"`
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
	Primitiva string `json:"primitiva"`          // puntual|periodica|continua|plazo|observacion|secuencia
	Hito      string `json:"hito,omitempty"`     // nombre del hito (por defecto "ocurrencia" / "limite")
	Cadencia  string `json:"cadencia,omitempty"` // periodica: ISO-8601 (P24M)
	Limite    string `json:"limite,omitempty"`   // plazo: ISO-8601 (P10D, PT72H)
	// En es el instante que fija LA NORMA, para la primitiva puntual. No es un
	// hecho del obligado y no se cuenta desde nada: la fecha esta escrita en el
	// texto legal.
	//
	// El caso que lo trajo: el art. 111.4 del AI Act, anadido por el Reglamento
	// (UE) 2026/1744, obliga a los proveedores de sistemas que ya estaban en el
	// mercado a cumplir el art. 50.2 "a mas tardar el 2 de diciembre de 2026".
	// No hay disparador que valga: la fecha es esa para todos.
	//
	// Se escribe con la hora dentro (2026-12-02T23:59:59Z) porque una primitiva
	// puntual no tiene regimen y por tanto no sabe cerrar el dia. Poner solo la
	// fecha significaria vencer a las 00:00, que es un dia entero de menos.
	En         string            `json:"en,omitempty"`
	Regimen    RegimenSpec       `json:"regimen"`
	Disparador map[string]string `json:"disparador,omitempty"` // p.ej. {"hecho": "ultima_auditoria"}

	// Hitos son los hitos ENCADENADOS de un plazo, para las normas que
	// escalonan la misma obligacion en varias notificaciones.
	//
	// POR QUE NO BASTA CON Hito Y Limite. Hasta el 26-08-2026 una obligacion
	// solo podia declarar UN hito con UN limite contado desde el disparador.
	// La familia A del censo (notificacion escalonada de incidente, once
	// fuentes y treinta y tres relojes) no cabe ahi por dos motivos que el
	// texto legal dice literalmente:
	//
	//	- la notificacion intermedia y la final cuentan desde la REMISION DE LA
	//	  INICIAL, no desde el incidente;
	//	- y sus limites los decide el NIVEL que asigna el propio obligado.
	//
	// Cuando Hitos viene relleno, manda; Hito y Limite se quedan para el caso
	// simple, que es la mayoria del corpus.
	Hitos []HitoSpec `json:"hitos,omitempty"`
}

// HitoSpec es un hito de un plazo escalonado.
type HitoSpec struct {
	ID     string `json:"id"`
	Limite string `json:"limite"` // ISO-8601; vacio o "indeterminado" = la norma no fija limite
	// DesdeHito encadena: el reloj de este hito arranca cuando se CUMPLE aquel,
	// no cuando ocurre el disparador. Vacio = desde el disparador.
	DesdeHito string `json:"desde_hito,omitempty"`
	// Clase es el hecho que tiene que constar para que este hito rija. Vacio =
	// rige siempre. Es como se expresa "los plazos dependen del nivel que
	// asigne el obligado": la clasificacion es un hecho con su instante, asi
	// que una reclasificacion posterior es otro hecho y manda la mas reciente.
	Clase string `json:"clase,omitempty"`
	// Alternativas son lecturas discrepantes del mismo plazo, con su cita. El
	// motor calcula todas, usa Limite y ensena la divergencia: no elige en
	// silencio.
	Alternativas []LecturaSpec `json:"alternativas,omitempty"`
	// Tope es un SEGUNDO limite del mismo hito que corre desde otro hecho y que
	// acorta al principal cuando vence antes. Los dos vinculan a la vez.
	Tope *TopeSpec `json:"tope,omitempty"`
	// Regimen propio de ESTE hito. Nil = el de la obligacion.
	//
	// POR QUE HACE FALTA. Una notificacion escalonada mezcla plazos de dos
	// naturalezas en la MISMA obligacion: el art. 14 del CRA da 24 y 72 HORAS
	// para las dos primeras y un MES para el informe final. Y el regimen no es
	// el mismo: el art. 3.4 del Reglamento 1182/71 traslada al habil siguiente
	// el vencimiento que cae en inhabil "expresado de cualquier modo, salvo en
	// horas", y el 3.2.b hace terminar el plazo en dias o meses al expirar la
	// ultima hora del ultimo dia. O sea que las horas vencen en un instante
	// exacto sin traslado y los meses a fin de dia con traslado.
	//
	// Con un solo regimen por obligacion habia que elegir: partir la obligacion
	// en dos (y entonces el informe final no puede encadenarse al hito del que
	// cuelga, porque desde_hito no cruza obligaciones) o aplicar el regimen de
	// las horas a los meses. Lo segundo da una fecha MAS TEMPRANA que la legal,
	// que es el lado inofensivo, pero sigue siendo una fecha equivocada, y este
	// producto se vende por dar la fecha buena.
	//
	// El motor ya lo soportaba: ventana.Hito lleva su propio Regimen desde el
	// principio. Lo que faltaba era poder decirlo desde un paquete.
	Regimen *RegimenSpec `json:"regimen,omitempty"`
	Nota    string       `json:"nota,omitempty"`
}

// TopeSpec es el segundo limite de un hito, contado desde otro hecho. Ver
// ventana.Tope para el porque; aqui solo esta la forma que escribe un paquete.
type TopeSpec struct {
	Desde  string `json:"desde"`
	Limite string `json:"limite"`
	// Caduca: el valor cero (false) es el RESTRICTIVO, el tope vincula siempre.
	// Caducar hay que pedirlo, y con cita.
	Caduca bool   `json:"caduca,omitempty"`
	Cita   string `json:"cita"`
}

// LecturaSpec es una interpretacion discrepante de un plazo.
type LecturaSpec struct {
	ID     string `json:"id"`
	Limite string `json:"limite"`
	Cita   string `json:"cita"`
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
	Caso       string            `json:"caso"`
	Obligacion string            `json:"obligacion"`
	Hechos     map[string]string `json:"hechos"` // clave -> fecha RFC3339 o 2006-01-02

	// Esperado es el conjunto COMPLETO de vencimientos que el motor tiene que
	// devolver con esos hechos. Ni uno de menos ni uno de mas.
	//
	// POR QUE ES UNA LISTA, Y POR QUE ES EXHAUSTIVA. Hasta el 27-08-2026 el
	// esperado era UN vencimiento y el ejecutor filtraba por su hito: un dorado
	// decia lo que TIENE que salir y no decia NADA de lo que NO tiene que
	// salir. Eso deja fuera media familia de fallos, la misma de siempre
	// (invariante 7): cuando una comprobacion recorre una lista para
	// contrastarla con otra, la direccion que falta es la que muerde.
	//
	// Muerde asi, y esta medido: quitandole la clase al hito del plazo general
	// del art. 73 del AI Act, ese hito rige SIEMPRE, y un incidente con
	// fallecimiento le ensena al operador DOS fechas para la misma obligacion
	// (la del 73.4 y la del 73.2) sin ninguna forma de saber cual es la suya.
	// Los doce dorados del paquete seguian en verde, porque cada uno miraba su
	// hito. Ahora esa mutacion pone rojos varios dorados.
	//
	// EL EMPAREJAMIENTO ES POR HITO, que es una identidad DENTRO del dato, no
	// por indice ni por orden (invariante 7): reordenar la lista no puede
	// cambiar lo que se compara con que. Por eso `hito` es obligatorio en cada
	// fila y no puede repetirse dentro de un dorado.
	Esperado []EsperadoDorado `json:"esperado"`

	// SubconjuntoPorque renuncia a la exhaustividad, y es una CADENA CON EL
	// MOTIVO en vez de un booleano a proposito.
	//
	// El invariante 8 dice que en una frontera el valor cero tiene que ser el
	// RESTRICTIVO. El valor cero de un bool es `false`, y un `exhaustivo: bool`
	// tendria el problema al reves (olvidarse del campo aflojaria la
	// comprobacion). Con una cadena, el valor cero (vacia) significa
	// EXHAUSTIVO, que es lo duro, y relajarlo cuesta escribir por que y para
	// que hito. Ademas deja el motivo consultable en el propio dato en vez de
	// en la cabeza de quien escribio el caso.
	//
	// Solo relaja UNA de las dos direcciones: la de "sobra". Las filas
	// declaradas se siguen exigiendo todas, y con su fecha exacta.
	SubconjuntoPorque string `json:"subconjunto_porque,omitempty"`

	CitaDelEsperado string `json:"cita_del_esperado"`
}

// EsperadoDorado es UNA fila del conjunto esperado: un hito y lo que la norma
// dice de el.
//
// LOS ESTADOS CUENTAN. Un vencimiento "pendiente de hecho" o "sin plazo legal"
// es un RESULTADO del motor, no un hueco: la norma exige la accion y el reloj
// no puede dar fecha todavia (o no la da nunca). Un conjunto exhaustivo los
// incluye, porque si no, el dorado volveria a callar sobre la mitad de lo que
// ve el operador en pantalla.
type EsperadoDorado struct {
	// Hito es la identidad de la fila: por aqui casa con el vencimiento del
	// motor. Obligatorio siempre, tambien cuando la obligacion tiene un solo
	// hito, porque emparejar "el unico que hay" es emparejar por posicion.
	Hito string `json:"hito"`

	// Vence es la fecha exacta, en RFC3339 o 2006-01-02. Obligatoria cuando el
	// estado es "determinado" (o sea, por defecto) y PROHIBIDA en los otros
	// dos: un vencimiento sin fecha no la tiene, y declararla seria afirmar
	// algo que el motor no dice.
	Vence string `json:"vence,omitempty"`

	// Estado es el vocabulario cerrado de ventana.EstadoVenc: "determinado",
	// "pendiente de hecho" o "sin plazo legal". Vacio significa "determinado",
	// y eso NO es un valor cero permisivo: determinado con fecha obligatoria es
	// la afirmacion MAS fuerte que una fila puede hacer. Declarar cualquiera de
	// los otros dos afirma otra cosa, no menos cosa.
	Estado string `json:"estado,omitempty"`
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
	URN      string `json:"urn"`
	Version  string `json:"version"`
	Clase    Clase  `json:"clase"`
	Licencia string `json:"licencia"`
	// LicenciaFuente es el regimen de derechos de la fuente, del vocabulario
	// cerrado de arriba. OBLIGATORIO: el linter no carga un paquete sin el.
	LicenciaFuente LicenciaFuente `json:"licencia_fuente"`
	// Atribucion es el aviso literal que hay que ENSENAR a quien usa el
	// producto. OBLIGATORIO, y en todos los estratos: donde hay obligacion de
	// atribuir dice a quien, y donde no la hay dice que puede hacer el lector
	// con ese contenido, que es la misma pregunta desde el otro lado.
	//
	// Es texto y no clave de catalogo: no se traduce. Ver nucleo/pantalla.
	Atribucion string `json:"atribucion"`
	// Identificador es de donde sale el contenido, guardado como IDENTIDAD y
	// no como direccion. El enlace que exigen las condiciones de reutilizacion
	// se DERIVA de el al pintar, con Identificador.Enlace. Ver identificador.go.
	Identificador Identificador `json:"identificador"`
	// FuenteHeredada es el campo `fuente` del formato viejo, que llevaba la URL
	// completa. Sigue leyendose SOLO para rechazarlo con un error que diga que
	// hacer: si se quitara del tipo, encoding/json lo ignoraria en silencio y
	// quien lo escribio se quedaria creyendo que su paquete cita la fuente.
	// Lleva `omitempty` a proposito: solo se lee, nunca se escribe, y si algun
	// dia se serializa un Paquete el campo retirado no puede reaparecer.
	FuenteHeredada string        `json:"fuente,omitempty"`
	Consolidado    bool          `json:"consolidado"` // obliga al aviso de texto informativo
	Vigencia       Vigencia      `json:"vigencia"`
	Entidades      []TipoEntidad `json:"entidades,omitempty"`
	Preguntas      []Pregunta    `json:"preguntas,omitempty"`
	Obligaciones   []Obligacion  `json:"obligaciones"`
	Plantillas     []Plantilla   `json:"plantillas,omitempty"`
	Escalas        []string      `json:"escalas,omitempty"`
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
//	derivacion  dos campos, los dos de un dorado y los dos razonamiento del
//	            autor sobre su propio caso, no texto de un tercero: la
//	            cita_del_esperado (por que esa fecha, con la cuenta hecha) y el
//	            subconjunto_porque (por que ese caso renuncia a la
//	            exhaustividad). Legitimamente largos los dos.
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

// lecturasDeVigencia clasifica los campos de las lecturas divergentes de una
// vigencia. Se escribe aparte porque Vigencia aparece en dos sitios del formato
// (la cabecera del paquete y cada obligacion) y una copia de esto en cada sitio
// seria una copia que se queda vieja.
//
// ID, Desde y Hasta son DERIVACION: un identificador nuestro y dos fechas, o sea
// la forma del dato, no el enunciado de nadie. Cita es REFERENCIA, por el mismo
// motivo exacto que HitoSpec.Alternativas[].Cita: su trabajo es senalar de donde
// sale la lectura discrepante, y sin techo ahi una "cita" se convierte en la
// transcripcion de un referencial por la puerta de atras.
func lecturasDeVigencia(prefijo, donde string, v Vigencia, uno func(string, string, string, tipoCampo)) {
	for _, l := range v.Alternativas {
		uno(prefijo+".Alternativas[].ID", donde, l.ID, derivacion)
		uno(prefijo+".Alternativas[].Desde", donde, l.Desde, derivacion)
		uno(prefijo+".Alternativas[].Hasta", donde, l.Hasta, derivacion)
		uno(prefijo+".Alternativas[].Cita", donde, l.Cita, referencia)
		// Espera es DERIVACION: es el identificador de un item nuestro, no
		// texto de nadie.
		uno(prefijo+".Alternativas[].Espera", donde, l.Espera, derivacion)
	}
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
	// LicenciaFuente es vocabulario cerrado, o sea que su longitud la decide
	// este fichero y no el paquete. Se emite igual para que el control de
	// exhaustividad lo vea: un campo del formato que no aparece aqui es un
	// campo que la frontera legal no mira.
	uno("Paquete.LicenciaFuente", donde, string(p.LicenciaFuente), referencia)
	// Atribucion es REFERENCIA por la misma razon que Licencia: es la
	// declaracion de derechos, que es justo donde el paquete tiene que poder
	// explicarse. Lleva techo, no barra libre.
	uno("Paquete.Atribucion", donde, p.Atribucion, referencia)
	// El identificador de la fuente. Todo REFERENCIA: un tipo de vocabulario
	// cerrado, un localizador, una clave de catalogo y el motivo por el que un
	// editor no tiene identificador. Ninguno es sitio para el enunciado de un
	// control, y los cuatro llevan techo igual.
	uno("Paquete.Identificador.Tipo", donde, string(p.Identificador.Tipo), referencia)
	uno("Paquete.Identificador.Valor", donde, p.Identificador.Valor, referencia)
	uno("Paquete.Identificador.Registro", donde, p.Identificador.Registro, referencia)
	uno("Paquete.Identificador.Motivo", donde, p.Identificador.Motivo, referencia)
	// El campo del formato viejo se mira igual mientras siga en el tipo: un
	// campo que se lee y no se clasifica es un campo que la frontera legal no
	// vigila, aunque su unico destino sea el error del linter.
	uno("Paquete.FuenteHeredada", donde, p.FuenteHeredada, referencia)
	uno("Paquete.Vigencia.Desde", donde, p.Vigencia.Desde, referencia)
	uno("Paquete.Vigencia.Hasta", donde, p.Vigencia.Hasta, referencia)
	lecturasDeVigencia("Paquete.Vigencia", donde, p.Vigencia, uno)
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
		lecturasDeVigencia("Paquete.Obligaciones[].Vigencia", d, o.Vigencia, uno)
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
			// En es DERIVACION: una fecha, o sea la forma del dato. No cabe ahi
			// el enunciado de nadie.
			uno("Paquete.Obligaciones[].Temporalidad.En", d, t.En, derivacion)
			uno("Paquete.Obligaciones[].Temporalidad.Regimen.Computo", d, t.Regimen.Computo, referencia)
			uno("Paquete.Obligaciones[].Temporalidad.Regimen.Cierre", d, t.Regimen.Cierre, referencia)
			uno("Paquete.Obligaciones[].Temporalidad.Regimen.Traslado", d, t.Regimen.Traslado, referencia)
			// Los hitos de un plazo escalonado. Casi todo DERIVACION: son
			// identificadores nuestros y duraciones ISO-8601, o sea la forma
			// del reloj y no el enunciado de nadie. Nada de esto es sitio donde
			// pueda colarse el texto de un catalogo de pago.
			//
			// Las dos excepciones estan pensadas y son las que importan:
			//
			//	- Nota es PROSA, porque es donde el autor del paquete explica
			//	  una decision de lectura ("la norma da dos cifras y no dice
			//	  cual"). Es nuestra prosa sobre la norma, no la norma.
			//	- Alternativas[].Cita es REFERENCIA, porque su trabajo es
			//	  senalar donde dice la norma la lectura discrepante. Sin techo
			//	  ahi, una "cita" se convierte en la transcripcion de un
			//	  referencial por la puerta de atras.
			for _, h := range t.Hitos {
				uno("Paquete.Obligaciones[].Temporalidad.Hitos[].ID", d, h.ID, derivacion)
				uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Limite", d, h.Limite, derivacion)
				uno("Paquete.Obligaciones[].Temporalidad.Hitos[].DesdeHito", d, h.DesdeHito, derivacion)
				uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Clase", d, h.Clase, derivacion)
				uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Nota", d, h.Nota, prosa)
				for _, a := range h.Alternativas {
					uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Alternativas[].ID", d, a.ID, derivacion)
					uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Alternativas[].Limite", d, a.Limite, derivacion)
					uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Alternativas[].Cita", d, a.Cita, referencia)
				}
				if hr := h.Regimen; hr != nil {
					uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Regimen.Computo", d, hr.Computo, referencia)
					uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Regimen.Cierre", d, hr.Cierre, referencia)
					uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Regimen.Traslado", d, hr.Traslado, referencia)
				}
				if tp := h.Tope; tp != nil {
					uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Tope.Desde", d, tp.Desde, derivacion)
					uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Tope.Limite", d, tp.Limite, derivacion)
					// La cita del tope es REFERENCIA por el mismo motivo que la
					// de una lectura alternativa: senala donde dice la norma el
					// segundo plazo, y sin techo se convierte en transcripcion.
					uno("Paquete.Obligaciones[].Temporalidad.Hitos[].Tope.Cita", d, tp.Cita, referencia)
				}
			}
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
		// Las tres columnas del conjunto esperado son REFERENCIA y ninguna es
		// prosa: un identificador de hito, una fecha y una palabra de un
		// vocabulario cerrado de tres. Ninguna es sitio para el enunciado de un
		// control, y las tres llevan techo igual, porque "identificador de
		// hito" es una cadena libre como cualquier otra.
		for _, e := range dor.Esperado {
			uno("Paquete.Dorados[].Esperado[].Hito", d, e.Hito, referencia)
			uno("Paquete.Dorados[].Esperado[].Vence", d, e.Vence, referencia)
			uno("Paquete.Dorados[].Esperado[].Estado", d, e.Estado, referencia)
		}
		// SubconjuntoPorque es DERIVACION, el SEGUNDO campo de esa clase: es el
		// razonamiento del autor sobre su propio caso, igual que la
		// cita_del_esperado, no texto transcrito de nadie. Con el techo de
		// referencia (300) la renuncia habria que escribirla en telegrama justo
		// donde el formato esta pidiendo un argumento. Y lleva ademas suelo,
		// que es lo que ningun otro campo tiene: MinimoDelMotivoDeSubconjunto.
		uno("Paquete.Dorados[].SubconjuntoPorque", d, dor.SubconjuntoPorque, derivacion)
		// CitaDelEsperado es DERIVACION: el porque de la fecha esperada, con la
		// cuenta hecha. Techo propio y alto, ver LimiteDerivacionReferencial.
		uno("Paquete.Dorados[].CitaDelEsperado", d, dor.CitaDelEsperado, derivacion)
	}

	// La aplicabilidad, que ESTABA FUERA DE LA FRONTERA. Es el P1 numero 1 del
	// corpus por su otra mitad: el limite se amplio a los veinte y pico campos
	// del formato, y el bloque de reglas se quedo fuera del barrido porque vive
	// en otro fichero y tiene su propio linter. Su linter comprueba que la
	// regla se PARSEA; no comprueba cuanto texto lleva dentro, y una regla es
	// una cadena libre con literales dentro:
	//
	//	aplica("<aqui cabe el enunciado entero de un control de pago>", S) :- ...
	//
	// Todas son REFERENCIA y ninguna es prosa: un identificador de predicado,
	// una cita de articulo, un nombre de escala y una regla en un dialecto
	// formal son localizadores, no texto escrito para leerse. La regla mas
	// larga del corpus de hoy gasta 116 bytes, o sea que el techo de 300 no
	// aprieta a nadie legitimo y si corta el parrafo copiado.
	donde = "aplicabilidad de " + p.URN
	varios("Paquete.Aplicabilidad.Exporta[]", donde, p.Aplicabilidad.Exporta, referencia)
	for i, rs := range p.Aplicabilidad.Reglas {
		d := "regla " + rs.ID
		if rs.ID == "" {
			d = fmt.Sprintf("regla %d (sin id)", i)
		}
		uno("Paquete.Aplicabilidad.Reglas[].ID", d, rs.ID, referencia)
		uno("Paquete.Aplicabilidad.Reglas[].Cita", d, rs.Cita, referencia)
		uno("Paquete.Aplicabilidad.Reglas[].Regla", d, rs.Regla, referencia)
		uno("Paquete.Aplicabilidad.Reglas[].Agregado", d, rs.Agregado, referencia)
		uno("Paquete.Aplicabilidad.Reglas[].Sobre", d, rs.Sobre, referencia)
		if e := rs.Escala; e != nil {
			uno("Paquete.Aplicabilidad.Reglas[].Escala.Nombre", d, e.Nombre, referencia)
			varios("Paquete.Aplicabilidad.Reglas[].Escala.Orden[]", d, e.Orden, referencia)
		}
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

// validarLicenciaFuente exige los dos campos de higiene legal y los cruza con
// la clase.
//
// Se comprueba en TODAS las clases, tambien en las que no tienen a nadie a
// quien atribuir. Un referencial no le debe atribucion a ISO, pero quien abre
// la pantalla sigue teniendo que saber que ese paquete no trae el texto y que
// la copia la pone el, y ese es exactamente el mismo campo. Hacerlo opcional
// para media tabla es dejar la mitad de las pantallas en blanco.
func (p *Paquete) validarLicenciaFuente(anotar func(error)) {
	switch lf := p.LicenciaFuente; {
	case lf == "":
		anotar(fmt.Errorf("%w: %s. Sin ella no se sabe si el texto se puede redistribuir "+
			"ni a quien hay que atribuirlo. Arreglo: declara licencia_fuente con uno de "+
			"los regimenes de docs/LICENCIAS.md (%s)",
			ErrSinLicenciaFuente, p.URN, listaDeLicencias(p.Clase)))
	default:
		if motivo, prohibida := licenciasProhibidas[lf]; prohibida {
			anotar(fmt.Errorf("%w: %s declara %q. %s. No es un pendiente, es un no: "+
				"docs/LICENCIAS.md lo explica y ahi consta por que",
				ErrLicenciaProhibida, p.URN, lf, motivo))
			break
		}
		if !licenciaConocida(lf) {
			anotar(fmt.Errorf("%w: %s declara %q, que no existe. El vocabulario es cerrado "+
				"a proposito: una fuente nueva entra con su constante en nucleo/corpus y su "+
				"fila en docs/LICENCIAS.md, no escribiendo otra cadena",
				ErrLicenciaFuenteDesconocida, p.URN, lf))
			break
		}
		if p.Clase.Valida() && !admite(p.Clase, lf) {
			anotar(fmt.Errorf("%w: %s es de clase %s y declara %q. La clase admite %s. "+
				"Uno de los dos campos miente y el linter no puede saber cual: arregla el "+
				"que este mal antes de publicar",
				ErrLicenciaFuenteIncoherente, p.URN, p.Clase, lf, listaDeLicencias(p.Clase)))
		}
	}
	if p.Atribucion == "" {
		anotar(fmt.Errorf("%w: %s. La Decision 2011/833/UE autoriza reutilizar el DOUE con "+
			"atribucion, y una atribucion que no viaja con el paquete no se puede ensenar a "+
			"quien usa el producto. Arreglo: escribe en atribucion el aviso literal que "+
			"tiene que salir en pantalla", ErrSinAtribucion, p.URN))
	}
}

func licenciaConocida(lf LicenciaFuente) bool {
	for _, lista := range licenciasPorClase {
		for _, x := range lista {
			if x == lf {
				return true
			}
		}
	}
	return false
}

func admite(c Clase, lf LicenciaFuente) bool {
	for _, x := range licenciasPorClase[c] {
		if x == lf {
			return true
		}
	}
	return false
}

// listaDeLicencias enumera los regimenes de una clase, en orden estable, para
// que el error diga que hay que escribir y no solo que lo escrito esta mal.
func listaDeLicencias(c Clase) string {
	lista := licenciasPorClase[c]
	if len(lista) == 0 {
		var todas []string
		for _, l := range licenciasPorClase {
			for _, x := range l {
				todas = append(todas, string(x))
			}
		}
		sort.Strings(todas)
		return strings.Join(todas, ", ")
	}
	out := make([]string, 0, len(lista))
	for _, x := range lista {
		out = append(out, string(x))
	}
	return strings.Join(out, ", ")
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
	validarLecturasDeVigencia("paquete "+p.URN, p.Vigencia, anotar)
	for _, o := range p.Obligaciones {
		if _, err := o.Vigencia.interpretar(); err != nil {
			anotar(fmt.Errorf("obligacion %s: %w", o.ID, err))
		}
		validarLecturasDeVigencia("obligacion "+o.ID, o.Vigencia, anotar)
	}
}

// validarLecturasDeVigencia comprueba las lecturas divergentes de una vigencia.
//
// Las cuatro cosas que se exigen, y las cuatro por el mismo motivo: una lectura
// divergente se le ENSENA al cliente al lado de la fecha que vincula, asi que
// tiene que poder defenderse sola.
//
//	id      para poder nombrarla en la pantalla y en el expediente.
//	cita    de donde sale. Sin cita es una opinion.
//	fecha   al menos una de las dos, o no dice nada.
//	que diverja de verdad. Una lectura identica a la declarada se lee como un
//	        desacuerdo que no existe, y eso hace dudar de la fecha buena.
func validarLecturasDeVigencia(donde string, v Vigencia, anotar func(error)) {
	vistos := map[string]bool{}
	for i, l := range v.Alternativas {
		nombre := l.ID
		if nombre == "" {
			nombre = fmt.Sprintf("#%d", i)
			anotar(fmt.Errorf("%s: la lectura divergente de vigencia %s no tiene id, "+
				"asi que no se puede nombrar donde se ensene", donde, nombre))
		}
		if vistos[l.ID] && l.ID != "" {
			anotar(fmt.Errorf("%s: la lectura divergente de vigencia %q esta declarada "+
				"dos veces", donde, l.ID))
		}
		vistos[l.ID] = true
		if strings.TrimSpace(l.Cita) == "" {
			anotar(fmt.Errorf("%w: %s, lectura %s. Escribe de donde sale esa otra fecha "+
				"(instrumento, articulo, y si no esta publicada, dilo)",
				ErrLecturaVigenciaSinCita, donde, nombre))
		}
		if l.Desde == "" && l.Hasta == "" {
			anotar(fmt.Errorf("%w: %s, lectura %s. Una lectura sin desde ni hasta no "+
				"discrepa de nada", ErrLecturaVigenciaVacia, donde, nombre))
			continue
		}
		// Se interpreta con las MISMAS reglas que la declarada: una lectura
		// divergente que no se puede leer es peor que ninguna, porque se ensena.
		if _, err := (Vigencia{Desde: l.Desde, Hasta: l.Hasta}).interpretar(); err != nil {
			anotar(fmt.Errorf("%s, lectura %s: %w", donde, nombre, err))
			continue
		}
		if l.Desde == v.Desde && l.Hasta == v.Hasta {
			anotar(fmt.Errorf("%w: %s, lectura %s dice lo mismo que la vigencia declarada "+
				"(desde=%q hasta=%q). Borrala, o corrige la que este mal",
				ErrLecturaVigenciaQueNoDiverge, donde, nombre, l.Desde, l.Hasta))
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
	p.validarLicenciaFuente(anotar)
	p.validarIdentificador(anotar)
	p.validarVigencias(anotar)

	p.validarAplicabilidad(e)

	if p.URN == "" {
		e("paquete sin urn")
	}
	if p.Version == "" {
		e("paquete %s sin version", p.URN)
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
		validarEsperadoDeDorado(d, e)
		porObl[d.Obligacion]++
	}
	for _, o := range p.Obligaciones {
		if o.Temporalidad == nil {
			continue
		}
		// EL MINIMO DE TRES DORADOS SOLO ALCANZA A LO QUE SE PUEDE CALCULAR.
		//
		// Lo destapo el art. 67.1 del RDL 19/2018 (notificacion de incidentes de
		// pago), que obliga a notificar "de forma inmediata" y NO DA NINGUN
		// NUMERO. El motor sabe decir eso: limite indeterminado, estado "sin
		// plazo legal", y se mide el tiempo transcurrido. Pero un dorado fija
		// una FECHA esperada, y de un plazo sin numero no sale ninguna: los
		// tres dorados que el linter exigia no se pueden escribir.
		//
		// Lo que hacia la regla anterior era empujar al autor a QUITARLE EL
		// RELOJ a esa obligacion para que el paquete cargara, y entonces el
		// producto deja de ensenar el cronometro y la obligacion se lee como
		// una mas sin urgencia. La regla castigaba justo la transcripcion
		// honesta.
		//
		// La regla nueva no abre un agujero: la exencion solo vale cuando NINGUN
		// limite es computable, y ademas exige que cada hito lleve NOTA. Asi el
		// hueco es una decision escrita y consultable, no una omision. Sin la
		// nota, el camino barato (dejar el limite vacio para librarse de los
		// dorados) vuelve a estar cerrado.
		if computables(*o.Temporalidad) {
			if porObl[o.ID] < 3 {
				e("obligacion %s declara reloj computable y tiene %d dorados (minimo 3: normal, "+
					"borde de calendario, y ocurrencia u variante)", o.ID, porObl[o.ID])
			}
			continue
		}
		if len(o.Temporalidad.Hitos) == 0 {
			e("obligacion %s declara un reloj sin numero con la forma simple (hito y limite). "+
				"Un plazo que la norma no cuantifica se escribe con `hitos`, para que cada uno "+
				"lleve su `nota` diciendo que dice la norma en vez del numero que no da", o.ID)
			continue
		}
		for _, h := range o.Temporalidad.Hitos {
			if strings.TrimSpace(h.Nota) == "" {
				e("obligacion %s, hito %s: no fija limite y no dice por que. Un plazo sin numero "+
					"se queda sin los tres dorados que el linter exige a los demas, asi que la "+
					"nota es lo unico que queda para que el hueco sea una decision y no un "+
					"descuido: escribe que dice la norma (de forma inmediata, sin demora "+
					"indebida) y su cita", o.ID, h.ID)
			}
		}
	}
	return errs
}

// validarEsperadoDeDorado comprueba la FORMA del conjunto esperado. Lo que dice
// (que las fechas sean las que da la norma) lo comprueba EjecutarDorado contra
// el motor; aqui solo se rechaza un esperado que no pueda afirmar nada.
//
// LAS DOS FORMAS DE LA NADA SE MIRAN POR SEPARADO (invariante 8), y las dos se
// rechazan, con mensajes distintos porque son fallos distintos:
//
//	nil (campo ausente o null)  se le olvido al autor. Es la forma peligrosa,
//	                            porque sale sola: un dorado sin esperado cuenta
//	                            para el minimo de tres y no afirma nada.
//	lista vacia presente        el autor quiso decir "el motor no devuelve
//	                            nada". Hoy eso no puede ser cierto para ningun
//	                            reloj del formato, y ademas se cumple SOLO: el
//	                            horizonte del ejecutor sale de la ultima fecha
//	                            declarada, asi que una lista vacia da horizonte
//	                            cero, y con horizonte cero una `periodica` no
//	                            devuelve ocurrencias. La afirmacion se hace
//	                            verdadera a si misma. El dia que una primitiva
//	                            devuelva legitimamente el vacio, se declarara
//	                            con una palabra que lo diga, no con la ausencia
//	                            de filas.
func validarEsperadoDeDorado(d Dorado, e func(string, ...any)) {
	switch {
	case d.Esperado == nil:
		e("dorado %q sin esperado: un caso que no declara ningun vencimiento no afirma "+
			"nada y aun asi cuenta para el minimo de tres por reloj. Arreglo: "+
			"\"esperado\": [{\"hito\": \"...\", \"vence\": \"...\"}], con TODOS los "+
			"vencimientos que el motor da con esos hechos", d.Caso)
	case len(d.Esperado) == 0:
		e("dorado %q con esperado vacio: una lista sin filas dice \"el motor no devuelve "+
			"nada\", y eso hoy no es cierto para ningun reloj (un plazo devuelve una fila "+
			"por hito, aunque sea pendiente de hecho o sin plazo legal). Ademas se cumple "+
			"sola: sin fechas declaradas el horizonte es cero y una periodica tampoco "+
			"devuelve nada. Arreglo: escribe las filas que salen", d.Caso)
	}
	vistos := map[string]bool{}
	for i, esp := range d.Esperado {
		if esp.Hito == "" {
			e("dorado %q, fila %d del esperado sin hito: el emparejamiento con el motor se "+
				"hace POR HITO y no por posicion, asi que una fila sin hito no casa con "+
				"nada. Arreglo: pon el id del hito de la obligacion (en una periodica, "+
				"\"nombre#n\")", d.Caso, i+1)
			continue
		}
		if vistos[esp.Hito] {
			e("dorado %q declara el hito %q dos veces en el esperado: el motor devuelve un "+
				"vencimiento por hito, asi que una de las dos filas no se comprobaria "+
				"contra nada y quedaria verde diga lo que diga", d.Caso, esp.Hito)
		}
		vistos[esp.Hito] = true

		estado := ventana.Determinado
		if esp.Estado != "" {
			var err error
			if estado, err = ventana.ParseEstadoVenc(esp.Estado); err != nil {
				e("dorado %q, hito %q: %v. Vacio significa \"determinado\"", d.Caso, esp.Hito, err)
				continue
			}
		}
		if estado == ventana.Determinado && esp.Vence == "" {
			e("dorado %q, hito %q: estado determinado sin vence. Determinado quiere decir "+
				"que hay fecha y hora exactas, asi que sin fecha la fila no afirma nada. "+
				"Arreglo: pon la fecha, o declara el estado que de verdad da el motor "+
				"(%q o %q)", d.Caso, esp.Hito,
				ventana.PendienteDeHecho.String(), ventana.SinPlazoLegal.String())
		}
		if estado != ventana.Determinado && esp.Vence != "" {
			e("dorado %q, hito %q: estado %q Y vence %q a la vez. Un vencimiento que no esta "+
				"determinado NO tiene fecha, asi que declararla es afirmar algo que el "+
				"motor no dice. Arreglo: quita una de las dos", d.Caso, esp.Estado, esp.Vence)
		}
	}
	// El opt-out: si esta, tiene que ser un argumento. Ver
	// MinimoDelMotivoDeSubconjunto.
	motivo := strings.TrimSpace(d.SubconjuntoPorque)
	if d.SubconjuntoPorque != "" && len(motivo) < MinimoDelMotivoDeSubconjunto {
		e("dorado %q: subconjunto_porque con %d caracteres utiles (minimo %d). Renunciar a "+
			"la exhaustividad tiene que costar un argumento: di QUE hitos quedan fuera y "+
			"POR QUE este caso no los afirma. Un motivo que no cabe en esa frase es una "+
			"etiqueta, y entonces el campo seria el booleano que este formato no quiere",
			d.Caso, len(motivo), MinimoDelMotivoDeSubconjunto)
	}
}

// computables dice si el reloj declarado produce alguna FECHA. Un plazo cuyos
// limites son todos indeterminados obliga y no se puede calcular, que son dos
// cosas ciertas a la vez.
func computables(t Temporalidad) bool {
	indeterminado := func(s string) bool { return s == "" || s == "indeterminado" }
	if t.Primitiva != "plazo" {
		return true // periodica, puntual y las demas siempre dan fecha
	}
	if len(t.Hitos) == 0 {
		return !indeterminado(t.Limite)
	}
	for _, h := range t.Hitos {
		if !indeterminado(h.Limite) {
			return true
		}
	}
	return false
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
	// Peticiones dice POR QUE lo pide cada uno, una entrada por paquete DISTINTO
	// y en orden de URN. Ayuda y Cita de arriba son las de la primera, que es lo
	// que habia antes de esto y se mantiene para no romper a quien ya las lee.
	//
	// OJO: no se emparejan por indice con Paquetes. Paquetes cuenta una vez por
	// declaracion, asi que un paquete que declare la misma entidad dos veces
	// sale dos veces ahi y una sola aqui. Esa cuenta inflada es anterior a
	// Peticiones y se deja como esta a proposito, para no cambiarle la forma a
	// quien ya la lee; queda apuntada.
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
