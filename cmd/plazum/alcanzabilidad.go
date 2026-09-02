package main

// EL TRINQUETE DE ALCANZABILIDAD: ninguna superficie se queda sin cablear en
// silencio.
//
// # El agujero que cierra, con sus tres casos
//
// En cuatro dias aparecieron tres piezas terminadas SIN EL CABLE, y ninguna
// puso roja una sola puerta:
//
//	la primitiva `maximo`, encendida en el motor y apagada para el corpus, con
//	ocho retenciones del CRA esperandola sin que nadie lo hubiera notado;
//	`superficies/acta`, entera y con sus tests en verde, sin una sola direccion
//	del producto que llevara a ella;
//	el camino guiado, que enlazaba a la entrevista con el enlace puesto y
//	VACIO, o sea que se comia las respuestas de quien lo usaba.
//
// La causa de los tres es la misma y no es el descuido: cada mitad pasaba SU
// puerta. Habia puertas que verifican piezas y ninguna que verificara juntas.
// Este fichero y su test son la mitad de superficies de la que falta; la otra
// mitad, la de las primitivas del motor, esta en primitivas_alcanzables_test.go.
//
// # Por que la declaracion vive AQUI y no al lado del test
//
// Porque este es el fichero que monta. Quien anade un montaje pasa por aqui, y
// quien anade una superficie y NO pasa por aqui es exactamente el caso que hay
// que cazar. Una declaracion escrita junto al test seria una segunda copia que
// se separa del cableado el dia que el cableado cambie, que es el dia en que
// importa.
//
// # El valor cero esta prohibido, y ese es el fondo del asunto
//
// El estado de una superficie es vocabulario cerrado y su cero NO es un estado:
// es el olvido. Es la misma forma que `origen_del_intervalo` en el corpus, y por
// la misma razon (invariante 8): en una frontera, el valor que sale por
// descuido tiene que ser el que se niega, nunca el permisivo. Si el cero
// significara «ya se montara», el silencio volveria a ser un estado valido, y el
// silencio es justo lo que produjo los tres casos de arriba.
//
// # Lo que el test hace con esto
//
// No se cree ni una palabra. Enumera las superficies RECORRIENDO EL ARBOL,
// mira cuales sirven HTTP leyendo su AST, cruza el resultado con este censo en
// los dos sentidos, y para las que se declaran montadas levanta el servidor de
// verdad y comprueba que contestan. Una declaracion optimista aqui no sirve de
// nada: el arbol y el servidor son los que mandan.

// EstadoDeSuperficie es que se hace con una superficie que sirve HTTP.
// Vocabulario cerrado.
type EstadoDeSuperficie uint8

const (
	// SinDeclarar es el VALOR CERO Y ES INVALIDO. Una superficie que llega
	// aqui con el cero es una que nadie declaro, que es el estado en el que
	// estuvo `superficies/acta` mientras nadie la montaba.
	SinDeclarar EstadoDeSuperficie = iota
	// Montada: `plazum serve` la cuelga y se llega a ella desde el camino
	// guiado sin teclear la direccion. Es la unica que promete alcanzabilidad,
	// y el test la comprueba contra el servidor levantado.
	Montada
	// MontadaFueraDelCamino: `plazum serve` la cuelga y NO es un paso del
	// camino guiado. Existe porque no todo lo que se sirve es un paso del
	// recorrido del comprador, pero exige motivo: una pantalla a la que solo
	// se llega tecleando la direccion es una pantalla que solo encuentra quien
	// ya sabia que existia.
	MontadaFueraDelCamino
	// NoMontadaAProposito: `plazum serve` NO la cuelga, y esta bien. Exige
	// motivo Y la orden o la configuracion que la levanta, porque una
	// superficie que no se puede levantar de ninguna forma no esta
	// deliberadamente apagada: esta muerta.
	NoMontadaAProposito
)

func (e EstadoDeSuperficie) String() string {
	switch e {
	case Montada:
		return "montada y en el camino"
	case MontadaFueraDelCamino:
		return "montada fuera del camino"
	case NoMontadaAProposito:
		return "no montada a proposito"
	default:
		// El cero se nombra por lo que es. Llamarlo «desconocido» suavizaria
		// justo el caso que esto existe para cazar.
		return "SIN DECLARAR (valor cero)"
	}
}

// DeclaracionDeSuperficie es lo que se afirma de un paquete de superficies/ que
// sirve HTTP.
type DeclaracionDeSuperficie struct {
	Estado EstadoDeSuperficie
	// Motivo es obligatorio salvo en Montada. Sin motivo, dentro de tres meses
	// nadie sabe si una superficie sin montar es deuda o decision, y la lectura
	// barata («ya se montara») es la que dejo el acta suelta.
	Motivo string
	// ComoSeLevanta es la orden o la configuracion que la pone en pie hoy,
	// obligatoria en NoMontadaAProposito. Es la puerta D11-b aplicada a una
	// superficie entera: un estado vacio sin verbo es un callejon.
	ComoSeLevanta string
	// PasoDelCamino es el ID del paso de superficies/camino que lleva a ella.
	// Obligatorio en Montada, y el test comprueba que ese paso EXISTE y que su
	// ruta contesta: es lo unico que convierte «esta montada» en «se llega».
	PasoDelCamino string
}

// SuperficiesHTTP es el censo, por el nombre del paquete bajo superficies/.
//
// SE CRUZA CON EL ARBOL EN LOS DOS SENTIDOS. Un paquete que sirva HTTP y no
// salga aqui rompe la puerta; una entrada de aqui que ya no exista, o que haya
// dejado de servir HTTP, tambien. Un censo comprobado en un solo sentido acaba
// siendo una lista de cosas que hubo.
var SuperficiesHTTP = map[string]DeclaracionDeSuperficie{
	// El servidor mismo. No es una pantalla: es quien cuelga a las demas.
	"serve": {
		Estado: MontadaFueraDelCamino,
		Motivo: "es el servidor, no un paso. Sirve la entrada (/entrar), la salud y el " +
			"flujo de primer administrador, y es quien monta a todas las de abajo. " +
			"Ponerlo como paso del camino seria decirle al comprador que visite el " +
			"armario donde estan los cables.",
	},
	"pantallas": {
		Estado:        Montada,
		PasoDelCamino: "alcance",
	},
	"camino": {
		Estado: MontadaFueraDelCamino,
		Motivo: "es el camino mismo. Un paso que lleva al camino que lo contiene es un " +
			"bucle, no un paso: se llega a el desde el menu de las seis pantallas y " +
			"desde la tira de vuelta que pintan las demas superficies.",
	},
	"acta": {
		Estado:        Montada,
		PasoDelCamino: "acta",
	},
	"uar": {
		Estado:        Montada,
		PasoDelCamino: "uar",
	},

	// LA QUE ESTA FUERA, Y ESTA BIEN QUE LO ESTE.
	//
	// Es el unico caso del arbol en el que «no montada» es la respuesta
	// correcta, y merece la pena decir por que, porque el reflejo al leer esta
	// puerta va a ser montarla:
	//
	// SCIM da de alta y de baja a personas. Su constructor FALLA sin token, a
	// proposito, para que no exista forma de construir un servidor SCIM abierto
	// ni por descuido. Montarlo en `plazum serve` obligaria a una de dos cosas,
	// y las dos son peores que no montarlo: o `serve` pide un token que casi
	// nadie va a usar, o lo monta sin el, que es una puerta trasera con forma
	// de estandar. Se levanta cuando el operador lo pide, con su credencial.
	"scim": {
		Estado: NoMontadaAProposito,
		Motivo: "da de alta y de baja a personas y su constructor exige token (no hay forma " +
			"de construirlo abierto). Colgarlo del servidor de la interfaz obligaria a " +
			"pedir una credencial que la mayoria de instalaciones no usa, o a montarlo " +
			"sin ella, que seria una puerta trasera con forma de estandar. Lo levanta " +
			"quien lo necesita, con su token.",
		ComoSeLevanta: "scim.Nuevo(scim.Opciones{Token: ...}) desde el proceso que integra " +
			"el IdP; todavia no tiene orden propia en cmd/plazum (deuda anotada: es la " +
			"casilla del escalado con jerarquia SCIM de la etapa 6).",
	},
}
