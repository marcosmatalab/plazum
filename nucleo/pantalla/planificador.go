package pantalla

import "time"

// El vigilante del vigilante: el estado del planificador, derivado.
//
// # Por que esto vive en el nucleo y no en el adaptador
//
// Un producto que se vende diciendo "no pierdas nunca la conformidad" no puede
// morir en silencio. El fallo que hay que cazar no es que el ordenador se
// apague (eso se nota), es que el planificador deje de correr ciclos mientras
// todo lo demas sigue en pie: la pantalla se abre, el corpus esta instalado, y
// los plazos vencen sin que nadie los mire. Desde fuera es indistinguible de
// que no venza nada.
//
// La regla que decide si un planificador esta muerto es una regla de dominio
// con un numero dentro (24 horas), asi que vive aqui, con el instante entrando
// como DATO, y no en el adaptador que lee el fichero de marcas. Que la regla
// este en un solo sitio es lo que permite que la pantalla Hoy, el comando de la
// terminal y el diagnostico digan exactamente lo mismo.
//
// # La direccion del aviso, que es lo unico que importa de este fichero
//
// El aviso es PARA EL OPERADOR y se calcula con SUS relojes. Ni una sola de las
// reglas del planificador mira el canal del latido, y eso no es una casualidad
// del codigo de hoy: es la propiedad que sostiene la pieza entera. Si el
// veredicto del planificador dependiera de que nuestro receptor conteste,
// entonces UNA CAIDA NUESTRA SE LEERIA COMO UNA CAIDA SUYA, y el operador
// aprenderia en dos semanas a ignorar el aviso, que es peor que no tenerlo.
//
// Lo vigila TestUnCanalRotoNoPuedeEnsuciarElVeredictoDelPlanificador, con su
// mutacion demostrada.
//
// El canal se informa aparte, nunca por encima de Aviso, y su rotulo dice con
// esas palabras que lo que falla es el canal hacia nosotros y no su
// planificador.

// UmbralDeSilencio es el silencio a partir del cual el planificador se declara
// muerto. Es el numero de la casilla de la etapa 2 ("aviso si calla 24 h") y
// esta aqui, en una constante exportada, para que la pantalla, la terminal y la
// documentacion no puedan discrepar.
const UmbralDeSilencio = 24 * time.Hour

// Marcas es lo que la instalacion sabe de si misma, como datos.
//
// Deliberadamente NO hay aqui ni un puntero a un fichero ni una funcion que lea
// nada: quien sepa leer el estado lo lee y rellena esto. Asi el veredicto se
// puede probar entero con una tabla de instantes.
type Marcas struct {
	// UltimoCiclo es cuando el planificador termino su ultimo ciclo. Cero
	// significa que no ha terminado ninguno todavia.
	UltimoCiclo time.Time
	// LatidoActivado dice si el pulso al receptor esta encendido. Por
	// defecto NO, y esa es la unica postura defendible en un producto cuya
	// tesis es que el receptor no se fia del emisor.
	LatidoActivado bool
	// UltimoPulso es el ultimo pulso ACEPTADO por el destino. Cero
	// significa que ninguno llego nunca.
	UltimoPulso time.Time
	// FalloElUltimoIntento dice si el ultimo intento de pulso no llego. Es
	// lo que hace util el smoke test del canal: probar y ver el resultado en
	// la pantalla sin esperar 24 horas.
	FalloElUltimoIntento bool
}

// Nivel es la gravedad de un veredicto de vigilancia.
//
// Son los mismos tres que usa puertos.Diagnostico y con el mismo significado.
// No se importa aquel: el nucleo no importa puertos, y duplicar tres constantes
// es mas barato que romper la regla de dependencias.
type Nivel uint8

const (
	// NivelCorrecto: late, o esta apagado a proposito.
	NivelCorrecto Nivel = iota
	// NivelAviso: algo se esta perdiendo, pero los plazos se siguen mirando.
	NivelAviso
	// NivelRoto: los plazos NO se estan mirando.
	NivelRoto
)

func (n Nivel) String() string {
	switch n {
	case NivelCorrecto:
		return "correcto"
	case NivelAviso:
		return "aviso"
	case NivelRoto:
		return "roto"
	}
	return "desconocido"
}

// Planificador es el veredicto sobre el planificador y, aparte, sobre el canal.
//
// Clave y Arreglo son CLAVES de catalogo, como todo rotulo de interfaz de este
// paquete. Horas es un numero y viaja como argumento de la clave: la forma
// plural la decide el catalogo, que es quien sabe en que idioma escribe.
type Planificador struct {
	Nivel   Nivel  `json:"nivel"`
	Clave   string `json:"clave"`
	Arreglo string `json:"arreglo,omitempty"`
	// Horas es el silencio en horas enteras, o -1 cuando no se puede saber.
	// Se redondea hacia abajo: "lleva 0 horas" es cierto y "lleva 1 hora"
	// cuando lleva 40 minutos, no.
	Horas int `json:"horas"`
	// UmbralHoras es el numero a partir del cual esto se declara roto. Va en
	// el modelo para que la pantalla pueda decirlo sin cablearlo.
	UmbralHoras int `json:"umbral_horas"`
	// Canal es el estado del pulso, informado APARTE y nunca por encima de
	// NivelAviso. Ver el encabezado del fichero.
	Canal Canal `json:"canal"`
}

// Canal es el estado del pulso hacia el receptor del latido.
type Canal struct {
	Activado bool   `json:"activado"`
	Nivel    Nivel  `json:"nivel"`
	Clave    string `json:"clave"`
	Arreglo  string `json:"arreglo,omitempty"`
	// Horas desde el ultimo pulso aceptado, o -1 si no lo hubo o no se sabe.
	Horas int `json:"horas"`
	// Descargo es la clave que dice, con esas palabras, que un canal en
	// silencio NO significa que el planificador este parado. Va siempre, no
	// solo cuando el canal falla: un descargo que solo sale en el caso malo
	// es un descargo que se lee cuando ya se ha sacado la conclusion.
	Descargo string `json:"descargo"`
}

// Las claves de catalogo que puede emitir la vigilancia.
//
// Estan en constantes y no sueltas en el codigo porque ClavesDelPlanificador
// las publica para el catalogo, y una lista escrita al lado de otra lista se
// desincroniza el dia que alguien anade un estado. Aqui la lista de abajo se
// construye con estas mismas constantes.
const (
	ClavePlanificadorLate        = "aviso.planificador.late"
	ClavePlanificadorCallado     = "aviso.planificador.callado"
	ClavePlanificadorNunca       = "aviso.planificador.nunca"
	ClavePlanificadorFuturo      = "aviso.planificador.futuro"
	ClavePlanificadorSinInstante = "aviso.planificador.sin_instante"

	ClaveArreglaCallado     = "aviso.planificador.arregla_callado"
	ClaveArreglaNunca       = "aviso.planificador.arregla_nunca"
	ClaveArreglaFuturo      = "aviso.planificador.arregla_futuro"
	ClaveArreglaSinInstante = "aviso.planificador.arregla_sin_instante"

	ClaveLatidoApagado = "aviso.latido.apagado"
	ClaveLatidoLate    = "aviso.latido.late"
	ClaveLatidoCallado = "aviso.latido.callado"
	ClaveLatidoNunca   = "aviso.latido.nunca"
	ClaveLatidoFallo   = "aviso.latido.fallo"

	ClaveArreglaLatidoCallado = "aviso.latido.arregla_callado"
	ClaveArreglaLatidoNunca   = "aviso.latido.arregla_nunca"
	ClaveArreglaLatidoFallo   = "aviso.latido.arregla_fallo"

	// ClaveLatidoNoEsTuPlanificador es el descargo de direccion. Es la clave
	// mas importante de esta lista: dice que el silencio del canal es
	// nuestro problema y no suyo.
	ClaveLatidoNoEsTuPlanificador = "aviso.latido.no_es_tu_planificador"
)

// ClavesDelPlanificador devuelve TODAS las claves de catalogo que la vigilancia
// puede emitir, ordenadas y sin repetidos.
//
// Existe para que la superficie web publique su inventario de claves sin
// copiarlas: un estado nuevo aqui aparece solo en el catalogo, y si nadie lo
// traduce, el CI lo dice antes de que salga en crudo en la pantalla de un
// cliente. Lo comprueba TestTodaClaveQueLaVigilanciaEmiteEstaPublicada, que
// recorre los estados alcanzables en vez de fiarse de esta lista.
func ClavesDelPlanificador() []string {
	return []string{
		ClavePlanificadorLate, ClavePlanificadorCallado, ClavePlanificadorNunca,
		ClavePlanificadorFuturo, ClavePlanificadorSinInstante,
		ClaveArreglaCallado, ClaveArreglaNunca, ClaveArreglaFuturo, ClaveArreglaSinInstante,
		ClaveLatidoApagado, ClaveLatidoLate, ClaveLatidoCallado, ClaveLatidoNunca,
		ClaveLatidoFallo,
		ClaveArreglaLatidoCallado, ClaveArreglaLatidoNunca, ClaveArreglaLatidoFallo,
		ClaveLatidoNoEsTuPlanificador,
	}
}

// Vigilar juzga el planificador y el canal en un instante dado.
//
// El instante entra como dato, aqui tambien: `nucleo/` no lee el reloj, y
// ademas asi el veredicto se prueba con una tabla en vez de con un sleep.
func Vigilar(m Marcas, ahora time.Time) Planificador {
	p := Planificador{
		Horas:       -1,
		UmbralHoras: int(UmbralDeSilencio / time.Hour),
		Canal:       vigilarCanal(m, ahora),
	}
	switch {
	case ahora.IsZero():
		// Nadie paso el instante. Es un fallo de integracion, no del
		// operador, y por eso NO se dice "correcto": un vigilante que da
		// por bueno lo que no ha podido mirar es exactamente el fallo del
		// que esta pieza defiende.
		p.Nivel, p.Clave, p.Arreglo = NivelAviso, ClavePlanificadorSinInstante, ClaveArreglaSinInstante
	case m.UltimoCiclo.IsZero():
		// Nunca ha corrido un ciclo. Se distingue de "callado" a
		// proposito: en una instalacion recien hecha esto es lo normal, y
		// pintarlo en rojo el primer minuto ensena al operador a ignorar
		// el rojo, que es como se pierde el aviso que si importa.
		p.Nivel, p.Clave, p.Arreglo = NivelAviso, ClavePlanificadorNunca, ClaveArreglaNunca
	case m.UltimoCiclo.After(ahora):
		// La ultima marca esta en el futuro. Puede ser un salto de reloj,
		// un contenedor con la hora mal o un fichero de estado tocado a
		// mano. Da igual cual: restar daria un silencio NEGATIVO y eso se
		// leeria como "late" para siempre, o sea que una marca en el
		// futuro APAGARIA la alarma. Nunca correcto.
		p.Nivel, p.Clave, p.Arreglo = NivelAviso, ClavePlanificadorFuturo, ClaveArreglaFuturo
	default:
		silencio := ahora.Sub(m.UltimoCiclo)
		p.Horas = int(silencio / time.Hour)
		if silencio >= UmbralDeSilencio {
			p.Nivel, p.Clave, p.Arreglo = NivelRoto, ClavePlanificadorCallado, ClaveArreglaCallado
			break
		}
		p.Nivel, p.Clave = NivelCorrecto, ClavePlanificadorLate
	}
	return p
}

// vigilarCanal juzga el pulso. NUNCA pasa de NivelAviso, y su veredicto no
// entra en el del planificador.
//
// Que el canal este apagado es NivelCorrecto y no un aviso, y esto es una
// decision de producto escrita aqui para que no se pierda: el latido es opt-in,
// el valor por defecto es apagado, y un producto que pinta en amarillo el
// hecho de no haber activado la telemetria esta empujando a activarla. Eso es
// exactamente lo que hace todo el mundo y es lo que hace que nadie se fie.
func vigilarCanal(m Marcas, ahora time.Time) Canal {
	c := Canal{
		Activado: m.LatidoActivado,
		Horas:    -1,
		Descargo: ClaveLatidoNoEsTuPlanificador,
	}
	if !m.LatidoActivado {
		c.Nivel, c.Clave = NivelCorrecto, ClaveLatidoApagado
		return c
	}
	switch {
	case m.UltimoPulso.IsZero():
		c.Nivel, c.Clave, c.Arreglo = NivelAviso, ClaveLatidoNunca, ClaveArreglaLatidoNunca
	case ahora.IsZero() || m.UltimoPulso.After(ahora):
		// Sin instante, o con la marca en el futuro, no se puede medir
		// cuanto lleva. Se informa como aviso y sin horas, en vez de
		// inventar un numero.
		c.Nivel, c.Clave, c.Arreglo = NivelAviso, ClaveLatidoCallado, ClaveArreglaLatidoCallado
	default:
		c.Horas = int(ahora.Sub(m.UltimoPulso) / time.Hour)
		if ahora.Sub(m.UltimoPulso) >= UmbralDeSilencio {
			c.Nivel, c.Clave, c.Arreglo = NivelAviso, ClaveLatidoCallado, ClaveArreglaLatidoCallado
			break
		}
		c.Nivel, c.Clave = NivelCorrecto, ClaveLatidoLate
	}
	// El fallo del ultimo intento manda sobre el "late": el smoke test acaba
	// de decir que el canal no entrega, y eso es lo que hay que ensenar
	// aunque el pulso de hace dos horas si llegara.
	if m.FalloElUltimoIntento && c.Nivel == NivelCorrecto {
		c.Nivel, c.Clave, c.Arreglo = NivelAviso, ClaveLatidoFallo, ClaveArreglaLatidoFallo
	}
	return c
}
