package pantallas

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/pantalla"
)

// LA ENTREVISTA APRENDE A PREGUNTAR VALORES.
//
// # El agujero que cierra, con su cardinal
//
// Hasta hoy esta superficie leia DOS parametros, `si` y `no`, y nada mas. Toda
// pregunta se pintaba como un boton de si y otro de no, incluidas las que piden
// un valor: «¿Que categoria alcanza el sistema segun el anexo I?» se contestaba
// que si.
//
// Medido sobre el corpus instalado el 04-09-2026, casando por (paquete,
// entidad, atributo) y con CERO preguntas sin casar: de las 68 preguntas de la
// entrevista, 33 son booleanas y 35 piden un valor (28 enumerado, 4 texto, 3
// fecha). Cruzado con lo que declara el puente de cada atributo, las que se
// perdian son las de forma `con_valor`: 25 (23 enumerado y 2 texto). Su «si» no
// llegaba al motor, y el exportador las apartaba a un cubo
// (`CuentaDeLaExportacion.ConValor`) porque el puente de su atributo afirma
// `predicado(instancia, valor)` y no habia valor que poner.
//
// LAS OTRAS 10 CON VALOR NO SON DEUDA DE ESTA REBANADA: su atributo declara
// `no_llega_al_motor` con su motivo escrito, o sea que se recogen y no afirman
// nada a proposito. Se preguntan igual, y mejor que antes, porque una fecha
// preguntada como si/no no es una fecha.
//
// La consecuencia no es una pantalla incompleta, es un alcance CORTO: menos
// obligaciones de las que tocan, presentadas como si no tocaran. Absolver de
// mas es el error caro en cumplimiento, porque el que acusa lo corrige el
// usuario al leerlo y el que absuelve lo descubre el inspector.
//
// # Por que NO hay ningun desplegable, y es la decision de seguridad del fichero
//
// La forma obvia de preguntar un enumerado es un `<select>`. Se ha descartado, y
// el motivo es exactamente el que el godoc de `corpus.PuenteAfirmaSiConValor`
// escribe para descartar el rodeo del enumerado de un solo valor:
//
//	UN DESPLEGABLE SIEMPRE TRAE ALGO SELECCIONADO. Con `designado` puesto por
//	defecto a "entidad_financiera", quien abre la pagina y no contesta nada
//	afirma que esta designado como entidad financiera, y ese hecho solo
//	enciende 28 obligaciones de las que una es NOTIFICATORIA ante el
//	supervisor. Un umbral escrito de menos en una notificatoria no cuesta
//	horas: provoca una actuacion indebida ante el supervisor, y eso no se
//	deshace.
//
// Un desplegable con una primera opcion «sin contestar» resuelve eso POR
// CONVENCION: hay que acordarse de ponerla, de que vaya primera y de que su
// valor sea vacio, en cada sitio que pinte uno. Aqui se resuelve POR
// CONSTRUCCION: un enumerado se pinta como UN ENLACE POR VALOR, igual que los
// botones de si y no que esta superficie lleva pintando desde el principio.
// «Sin contestar» no es una opcion que haya que recordar poner, es NO HABER
// PULSADO NINGUNO, y la ausencia del parametro en la direccion es literalmente
// la nada. No hay estado por defecto que afirme porque no hay estado por
// defecto.
//
// Los tipos sin lista cerrada (texto, entero, fecha) si necesitan un campo
// libre, y ahi el valor cero del campo es la cadena vacia, que tampoco afirma.
//
// # Las tres formas de la nada, y la tercera no es la nada (invariante 8)
//
//	ausente                    el parametro no viene. Nadie ha contestado.
//	                           No afirma nada, y es el valor cero.
//	presente y vacio           el parametro viene vacio. Es «deshacer»: quien
//	                           contesto quiere dejarlo sin contestar. Tampoco
//	                           afirma nada, y es una respuesta legitima.
//	presente y NO interpretable   un dato que HAY y no se entiende: una fecha
//	                           que pone "ayer", un entero que pone "muchos", un
//	                           enumerado con un valor que no esta en la lista
//	                           que declara el paquete, o dos valores para la
//	                           misma pregunta. NO es «sin contestar»: es ERROR,
//	                           se dice en pantalla y NUNCA se toma por el valor
//	                           por defecto, porque tomarlo por la nada es
//	                           inventarse un valor, y aqui inventarse un valor
//	                           mete una afirmacion falsa en el alcance del
//	                           cliente.
//
// Y las dos primeras se leen con una funcion y la obligatoria con OTRA
// (`leerValorOpcional` y `leerValorObligatorio`): un campo obligatorio y uno
// opcional son dos preguntas distintas, y meterlas en una con valor por defecto
// es por donde se cuela el cero.

// ParamValor es el PREFIJO de la clave con la que un valor viaja en la
// direccion: `v.<id de pregunta>`.
//
// # Por que un prefijo y no un parametro repetido como `si` y `no`
//
// `si=<id>` funciona porque la respuesta cabe en el nombre del parametro. Un
// valor no: haria falta empaquetar dos cosas (que pregunta y que valor) en una
// sola cadena, con un separador que hay que escapar, y un separador que hay que
// escapar es un sitio donde meter lo que no toca. Con el prefijo, quien parsea
// es `url.ParseQuery`, que ya sabe hacerlo, y el id de la pregunta llega entero
// sin que nadie lo trocee.
//
// No colisiona con `ver`: el prefijo lleva el punto dentro.
const ParamValor = "v"

// ClaveValor compone la clave de la consulta para una pregunta.
func ClaveValor(id string) string { return ParamValor + "." + id }

// MaxLargoValor acota lo que se admite en un campo libre.
//
// Es una puerta, no un adorno: sin ella, un texto de un megabyte entra en el
// estado de la pantalla, se copia a cada uno de los enlaces que la pagina pinta
// y convierte una peticion en una pagina de varios megabytes. Lo que pasa de
// aqui NO se recorta a la medida (recortar en silencio es inventarse un valor
// distinto del que mandaron): se trata como no interpretable.
const MaxLargoValor = 200

// TipoDeCampo dice como se pregunta un atributo y como se lee su respuesta.
//
// EL VALOR CERO ES CampoSiNo, Y ES EL RESTRICTIVO (invariante 8). Una pregunta
// de la que no se sabe el tipo (porque el corpus no la declara, o porque el
// vocabulario llego vacio) se pinta como el si/no de siempre, que es la forma
// que NO afirma cuando no se pulsa; y cualquier `v.<id>` que llegue sobre ella
// es un dato presente y no interpretable, porque no hay con que interpretarlo.
// El sentido contrario (dar por bueno un valor que nadie puede comprobar) es el
// permisivo, y es el que sale por olvidarse.
type TipoDeCampo uint8

const (
	// CampoSiNo: booleano. Se contesta con los dos botones de siempre.
	CampoSiNo TipoDeCampo = iota
	// CampoOpcion: enumerado. Un enlace por valor declarado, y ninguno pulsado
	// de partida.
	CampoOpcion
	// CampoTexto: campo libre, sin mas forma que el largo maximo.
	CampoTexto
	// CampoEntero: campo libre que tiene que parsear como entero.
	CampoEntero
	// CampoFecha: campo libre en aaaa-mm-dd.
	CampoFecha
)

// PideValor dice si este tipo se contesta con un valor en vez de con si/no.
func (t TipoDeCampo) PideValor() bool { return t != CampoSiNo }

// FormatoFecha es el unico que se admite. Se escribe una vez y se usa para leer
// y para decirselo al operador: dos copias de un formato son dos formatos.
const FormatoFecha = "2006-01-02"

// campoDePregunta es lo que el corpus declara del atributo de una pregunta.
type campoDePregunta struct {
	tipo    TipoDeCampo
	valores []string
	valido  map[string]bool
}

// Vocabulario dice, por pregunta, que tipo de dato pide y que valores admite.
//
// # De donde sale, y por que no de pantalla.Pregunta
//
// `pantalla.Pregunta` trae el paquete, la entidad y el atributo, y NO trae el
// tipo ni los valores: la entrevista derivada no los necesitaba mientras todo
// se contestaba con si o no. Se leen del corpus, del PAQUETE QUE DECLARA LA
// PREGUNTA, y no del esquema de campos combinado: el esquema une atributos con
// el mismo nombre de dos paquetes distintos, y quien valida despues el valor es
// `corpus.HechosDeLaEntrevista` contra el paquete concreto. Validar contra la
// union y afirmar contra el paquete serian dos juicios distintos sobre el mismo
// dato, y el dia que se separaran ganaria el permisivo.
//
// # El emparejamiento, y por que campo casa (invariante 7)
//
// Casa por (paquete, entidad, atributo), los tres campos que la propia pregunta
// declara y que viven DENTRO del paquete firmado, igual que casa el puente. No
// hay indice, ni posicion, ni orden: reordenar las entidades de un paquete no
// mueve ni un emparejamiento.
//
// EL VALOR CERO NO CONOCE NINGUNA PREGUNTA, y eso lo deja todo en CampoSiNo, que
// es el restrictivo. Ver TipoDeCampo.
type Vocabulario struct {
	porID map[string]campoDePregunta
}

// VocabularioDe lee del corpus el tipo y los valores de cada pregunta.
func VocabularioDe(ps []*corpus.Paquete, qs []pantalla.Pregunta) Vocabulario {
	type clave struct{ urn, entidad, atributo string }
	attr := map[clave]corpus.Atributo{}
	for _, p := range ps {
		if p == nil {
			continue
		}
		for _, e := range p.Entidades {
			for _, a := range e.Atributos {
				attr[clave{p.URN, e.Nombre, a.Nombre}] = a
			}
		}
	}
	v := Vocabulario{porID: make(map[string]campoDePregunta, len(qs))}
	for _, q := range qs {
		a, hay := attr[clave{q.Paquete, q.Entidad, q.Atributo}]
		if !hay {
			// La pregunta nombra un atributo que su paquete no declara. Se deja
			// FUERA del vocabulario, o sea en CampoSiNo, que es como se pintaba
			// hasta hoy: una errata del corpus no puede convertirse en un campo
			// que acepte lo que sea.
			continue
		}
		c := campoDePregunta{tipo: tipoDeAtributo(a)}
		if c.tipo == CampoOpcion {
			c.valores = append([]string(nil), a.Valores...)
			c.valido = make(map[string]bool, len(a.Valores))
			for _, x := range a.Valores {
				c.valido[x] = true
			}
			if len(c.valores) == 0 {
				// UN ENUMERADO SIN VALORES NO SE PUEDE PREGUNTAR. El linter del
				// corpus ya lo rechaza, pero esta superficie recibe los paquetes
				// de quien la monta y no puede darlo por hecho: sin lista, no hay
				// con que comprobar la respuesta, asi que se vuelve al si/no.
				c = campoDePregunta{tipo: CampoSiNo}
			}
		}
		v.porID[q.ID] = c
	}
	return v
}

// tipoDeAtributo traduce el tipo del corpus al de la superficie.
//
// El default es CampoSiNo por lo mismo que el valor cero: un tipo que esta
// superficie no conozca todavia se pregunta de la forma que no afirma sola.
func tipoDeAtributo(a corpus.Atributo) TipoDeCampo {
	switch a.Tipo {
	case corpus.Booleano:
		return CampoSiNo
	case corpus.Enumerado:
		return CampoOpcion
	case corpus.Texto:
		return CampoTexto
	case corpus.Entero:
		return CampoEntero
	case corpus.Fecha:
		return CampoFecha
	}
	return CampoSiNo
}

// campo devuelve lo declarado de una pregunta. Lo que no esta declarado es
// CampoSiNo, que es el restrictivo.
func (v Vocabulario) campo(id string) campoDePregunta { return v.porID[id] }

// Tipo dice como se pregunta esta pregunta.
func (v Vocabulario) Tipo(id string) TipoDeCampo { return v.porID[id].tipo }

// Valores da los valores declarados de un enumerado, o nada.
func (v Vocabulario) Valores(id string) []string { return v.porID[id].valores }

// EstadoDelValor es en cual de los casos esta la respuesta con valor de una
// pregunta. Son cinco y no dos, y las tres primeras son las tres formas de la
// nada del invariante 8.
type EstadoDelValor uint8

const (
	// ValorAusente: el parametro no viene. Nadie ha contestado. VALOR CERO, y
	// es el que no afirma nada.
	ValorAusente EstadoDelValor = iota
	// ValorSinContestar: el parametro viene y esta vacio. Es el «deshacer» de
	// un campo libre. Tampoco afirma nada, y es distinto de ausente porque el
	// operador SI ha pasado por aqui; se cuentan aparte para no contar como
	// contestada una pregunta que se acaba de vaciar.
	ValorSinContestar
	// ValorPuesto: hay respuesta y se entiende. Es la unica que afirma.
	ValorPuesto
	// ValorNoInterpretable: LA TERCERA HERMANA, y no es la nada. Hay un dato y
	// no se entiende. Es error: se dice y no se usa.
	ValorNoInterpretable
	// ValorContradictorio: la misma pregunta llega con dos valores distintos, o
	// con un valor Y un si/no de la forma antigua. No se resuelve a favor de
	// ninguno: elegir uno en silencio seria afirmar lo que nadie afirmo.
	ValorContradictorio
)

// Afirma dice si este estado produce una respuesta que llegue al motor. Existe
// para que la pregunta se conteste en un solo sitio: repartida por el fichero,
// el dia que se anada un sexto estado alguna copia se quedaria vieja.
func (e EstadoDelValor) Afirma() bool { return e == ValorPuesto }

// EsError dice si este estado es un dato que hay y no se entiende. Los dos que
// lo son se cuentan juntos en pantalla, porque para quien contesta son la misma
// noticia: «mandaste algo que no se ha podido usar».
func (e EstadoDelValor) EsError() bool {
	return e == ValorNoInterpretable || e == ValorContradictorio
}

// Clave es la clave de catalogo del aviso de este estado, vacia si no hay que
// avisar de nada.
func (e EstadoDelValor) Clave() string {
	switch e {
	case ValorNoInterpretable:
		return "alcance.pregunta.valor.no_se_entiende"
	case ValorContradictorio:
		return "alcance.pregunta.valor.contradictorio"
	}
	return ""
}

// leerValorOpcional es LA LECTURA DE LA FRONTERA DE ENTRADA de un campo que se
// puede dejar sin contestar, que en la entrevista son todos.
//
// Devuelve el valor SANEADO (nunca lo que llego tal cual) y en que caso esta.
// Un valor que no se entiende vuelve VACIO a proposito: asi no puede acabar
// reflejado en un enlace ni en la pagina por descuido, que es la mitad de un
// XSS reflejado y no hay razon para tener esa mitad.
func leerValorOpcional(v url.Values, id string, c campoDePregunta) (string, EstadoDelValor) {
	bruto, presente := v[ClaveValor(id)]
	if !presente {
		return "", ValorAusente
	}
	if len(bruto) != 1 {
		// Repetido: dos valores no se reducen a uno sin elegir por quien
		// contesta. Es el mismo criterio que `modoPedido`.
		return "", ValorContradictorio
	}
	crudo := strings.TrimSpace(bruto[0])
	if crudo == "" {
		return "", ValorSinContestar
	}
	if !c.tipo.PideValor() {
		// UN VALOR SOBRE UNA PREGUNTA QUE NO LO PIDE. Un booleano no tiene
		// donde poner un valor, asi que esto es un dato que hay y no se
		// entiende, no una respuesta a otra cosa. Es tambien el caso de una
		// pregunta cuyo tipo no conocemos, y ahi refusar es lo restrictivo.
		return "", ValorNoInterpretable
	}
	if len(crudo) > MaxLargoValor {
		return "", ValorNoInterpretable
	}
	if !interpretable(crudo, c) {
		return "", ValorNoInterpretable
	}
	return crudo, ValorPuesto
}

// leerValorObligatorio es LA OTRA LECTURA, la del campo que TIENE que traer un
// valor, y es una funcion distinta a proposito (invariante 8).
//
// La usa quien va a convertir una respuesta en un hecho: ahi la ausencia ya no
// es una respuesta legitima, porque solo se llega si la pregunta consta como
// contestada. Las tres formas de la nada colapsan aqui en la misma respuesta
// («no hay valor utilizable»), y eso es correcto PRECISAMENTE porque en la otra
// funcion no colapsan: la distincion se hizo donde el dato entra, y aqui lo que
// se necesita saber es otra cosa.
//
// Devuelve el valor y si se puede usar. Quien llama NO tiene permitido seguir
// con la cadena vacia: `corpus.HechosDeLaEntrevista` se niega a traducir un
// `con_valor` vacio, y tragarse esa negativa daria un alcance corto sin que
// nadie lo supiera.
func leerValorObligatorio(v url.Values, id string, c campoDePregunta) (string, bool) {
	valor, estado := leerValorOpcional(v, id, c)
	if estado != ValorPuesto {
		return "", false
	}
	return valor, true
}

// interpretable dice si el valor se entiende para el tipo que declara el
// corpus. No recorta, no normaliza y no adivina: o se entiende o es error.
func interpretable(valor string, c campoDePregunta) bool {
	switch c.tipo {
	case CampoOpcion:
		// EL VALOR TIENE QUE SER UNO DE LOS QUE DECLARA EL PAQUETE. Sin esta
		// linea, `v.ens.q.categoria=ALTISIMA` entraria en el estado de la
		// pantalla, viajaria al exportador y produciria `categoria(sis,
		// "ALTISIMA")`, un hecho que ninguna regla casa y que no da error en
		// ningun sitio.
		return c.valido[valor]
	case CampoEntero:
		// EL ERROR DE Atoi NO SE DESCARTA. Es el caso literal del invariante 8:
		// con el error tragado, "muchos" y "" y el campo ausente serian el
		// mismo cero, y solo dos de los tres son la nada.
		_, err := strconv.Atoi(valor)
		return err == nil
	case CampoFecha:
		// Se parsea con el formato entero, no con un prefijo: "2026-13-45" no
		// es una fecha, y `time.Parse` lo dice.
		_, err := time.Parse(FormatoFecha, valor)
		return err == nil
	case CampoTexto:
		// Un texto libre se entiende siempre salvo que traiga caracteres que no
		// deberian viajar en una direccion. Los de control se rechazan en vez
		// de limpiarse: limpiar cambia el dato que mandaron por otro.
		return sinCaracteresDeControl(valor)
	}
	return false
}

// HayRespuestasEnLaDireccion dice si la consulta trae alguna respuesta de la
// entrevista, de cualquiera de las dos formas.
//
// # Por que no vale mirar solo `si` y `no`, y es lo que estaba escrito
//
// La regla de esta superficie es «si la direccion trae respuestas, mandan las de
// la direccion; si no, las de la cuenta». Mientras una respuesta solo podia ser
// un si o un no, preguntar por esos dos parametros era preguntar por todas.
// Ahora hay una tercera forma, y con la pregunta vieja un enlace compartido que
// llevara SOLO valores se leia como «la direccion no trae nada», o sea que la
// pantalla ensenaba las respuestas de la cuenta de quien abriera el enlace y se
// comia las del enlace sin decirlo. Es la version silenciosa del fallo que esta
// rebanada existe para cerrar.
//
// Las preguntas conocidas salen del corpus, no de la peticion, por lo mismo que
// en `De`: un `v.<lo que sea>` que el corpus no declara no es una respuesta y no
// puede decidir de donde sale lo que se pinta.
func HayRespuestasEnLaDireccion(v url.Values, conocidas []pantalla.Pregunta) bool {
	if len(v[ParamSi]) > 0 || len(v[ParamNo]) > 0 {
		return true
	}
	for _, q := range conocidas {
		if _, hay := v[ClaveValor(q.ID)]; hay {
			return true
		}
	}
	return false
}

// sinCaracteresDeControl dice si la cadena no lleva nada por debajo del espacio
// ni el DEL. Se mira por punto de codigo, no por bytes: un byte suelto de una
// secuencia UTF-8 no es un caracter de control y no hay que confundirlos.
func sinCaracteresDeControl(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}
