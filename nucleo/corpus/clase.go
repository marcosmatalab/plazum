// La FRONTERA LEGAL del formato: que se puede redistribuir de un paquete y bajo
// que regimen. Vive en su fichero porque es la razon de cambio que menos tiene
// que ver con las demas: la mueve el derecho de autor, no el motor.

package corpus

import (
	"fmt"
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
