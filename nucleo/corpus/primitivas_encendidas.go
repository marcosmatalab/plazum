package corpus

import "errors"

// EL CENSO DE PRIMITIVAS ENCENDIDAS: la mitad barata del trinquete de
// alcanzabilidad.
//
// # Que agujero cierra
//
// El motor de `nucleo/ventana` tiene primitivas construidas y probadas. Que una
// primitiva tenga sus dorados en verde NO significa que un paquete pueda
// usarla: entre las dos cosas esta este fichero, que es el unico sitio donde un
// `"primitiva": "x"` de un paquete.json se convierte en un calculo. Si aqui no
// hay rama, la primitiva existe para el codigo y no existe para el corpus, que
// es tanto como decir que no existe: el invariante 2 dice que toda norma vive en
// su paquete de datos, y una primitiva que solo se enciende escribiendo Go
// convierte el siguiente marco que la necesite en un cambio de producto.
//
// Paso tres veces en cuatro dias, y ninguna puso roja una puerta:
//
//	`maximo`, construida y probada desde semanas antes y apagada para el
//	corpus, con ocho retenciones del CRA esperandola sin que nadie lo notara;
//	`Secuencia`, que nunca tuvo rama aqui y se borro el 02-09-2026 cuando se
//	midio que ademas no servia para lo unico que se penso usarla;
//	la pantalla del acta, entera y sin montar, que es el mismo fallo en la
//	otra punta del producto y por eso el trinquete tiene dos mitades.
//
// La razon de que ninguna puerta lo viera es que cada mitad pasaba la suya: los
// dorados de la primitiva estaban verdes y el linter de paquetes tambien. Nadie
// miraba la JUNTA.
//
// # Por que un censo escrito y no solo una medicion
//
// Medir es facil: se cuentan las primitivas del motor, se cuentan las ramas de
// VencimientosDe y se cuentan las que usan los paquetes. Lo que la medicion no
// puede dar es el MOTIVO, y sin motivo un hueco es indistinguible de un olvido.
// Por eso cada primitiva declara su estado y, si no esta en uso, por que no lo
// esta y CUANTOS relojes contados la esperan. Un hueco sin numero se olvida; un
// hueco con numero molesta hasta que se cierra.
//
// El cero de EstadoDePrimitiva NO es un estado, es el olvido, y esta prohibido:
// es la misma forma que `origen_del_intervalo` (vocabulario cerrado, valor cero
// invalido) porque el fallo que hay que impedir es el mismo, el silencio.

// ErrPrimitivaSinEjecutor: la obligacion declara una primitiva que el motor
// tiene y que VencimientosDe no sabe construir. Es centinela para que el
// trinquete pregunte por identidad y no por una subcadena del mensaje.
var ErrPrimitivaSinEjecutor = errors.New("primitiva sin ejecutor en el corpus")

// EstadoDePrimitiva es en que situacion esta una primitiva del motor respecto
// del corpus. Vocabulario cerrado.
type EstadoDePrimitiva uint8

const (
	// PrimitivaSinDeclarar es el VALOR CERO Y ES INVALIDO. No significa "no
	// pasa nada": significa que alguien anadio una primitiva al motor y no
	// vino aqui. Es exactamente el estado que produjo los tres casos de
	// arriba, asi que es el que tiene que romper la puerta.
	PrimitivaSinDeclarar EstadoDePrimitiva = iota
	// PrimitivaEnUso: hay al menos un paquete publicado que la declara. Es el
	// unico estado que no necesita motivo, porque el corpus es su motivo.
	PrimitivaEnUso
	// PrimitivaApagada: VencimientosDe la sabe construir y ningun paquete la
	// usa todavia. La primitiva esta disponible para el corpus: lo que falta es
	// escribir los relojes, no tocar codigo.
	PrimitivaApagada
	// PrimitivaSinCablear: el motor la tiene y VencimientosDe NO la construye,
	// asi que un paquete NO PUEDE declararla sin tocar Go. Es el estado en el
	// que estuvo `maximo` con ocho relojes esperando, y es el que rompe el
	// invariante 2.
	PrimitivaSinCablear
)

func (e EstadoDePrimitiva) String() string {
	switch e {
	case PrimitivaEnUso:
		return "en uso"
	case PrimitivaApagada:
		return "apagada"
	case PrimitivaSinCablear:
		return "sin cablear"
	default:
		// El cero se nombra por lo que es. Decir "desconocido" seria suavizar
		// justo el caso que la puerta existe para cazar.
		return "SIN DECLARAR (valor cero)"
	}
}

// DeclaracionDePrimitiva es lo que se afirma de una primitiva del motor.
type DeclaracionDePrimitiva struct {
	Estado EstadoDePrimitiva
	// Motivo dice POR QUE no esta en uso. Obligatorio salvo en PrimitivaEnUso.
	// Un estado sin motivo es un hueco sin explicacion, que a los tres meses
	// nadie sabe si es deuda o decision.
	Motivo string
	// RelojesEsperando es EL CARDINAL: cuantos relojes del censo
	// (docs/censo-relojes.md) pide esta primitiva y hoy no se pueden escribir.
	// Cero es una respuesta, y es la mas importante de todas: una primitiva que
	// no espera ningun reloj es peso muerto, que es la medicion que se llevo
	// por delante a `Secuencia`.
	RelojesEsperando int
	// DondeSeCuentan es la seccion del censo de la que sale el cardinal. Sin
	// esto el numero no es contrastable, y un numero que nadie puede recontar
	// envejece sin que se note.
	DondeSeCuentan string
}

// PrimitivasDelCorpus es el censo, por el nombre que devuelve Primitiva.Nombre().
//
// SE COMPARA CON EL ARBOL EN LOS DOS SENTIDOS, y por eso no basta con que este
// al dia: una primitiva del motor que no salga aqui rompe la puerta, y una
// entrada de aqui que ya no exista en el motor tambien. Un censo que solo se
// comprueba en un sentido se convierte en una lista de cosas que hubo.
var PrimitivasDelCorpus = map[string]DeclaracionDePrimitiva{
	"puntual":   {Estado: PrimitivaEnUso},
	"periodica": {Estado: PrimitivaEnUso},
	"continua":  {Estado: PrimitivaEnUso},
	"plazo":     {Estado: PrimitivaEnUso},
	"maximo":    {Estado: PrimitivaEnUso},

	"preaviso": {
		Estado: PrimitivaApagada,
		Motivo: "cableada el 02-09-2026 (rama en VencimientosDe, validarPreaviso en el " +
			"linter) y todavia sin un solo paquete que la declare. NO es un hueco de " +
			"codigo: un paquete puede usarla hoy sin tocar Go. Lo que falta es escribir " +
			"los siete relojes, y los siete estan FUERA de los 12 marcos de la v1 (psd2, " +
			"mica, mdr, data-act), asi que la deuda no bloquea la v1 y por eso espera.",
		RelojesEsperando: 7,
		DondeSeCuentan:   "docs/censo-relojes.md, «Familia G: preaviso contractual»",
	},

	// LA QUE ESTA PEOR, Y LA ENCONTRO ESTA PUERTA AL NACER.
	//
	// `observacion` cumple HOY las tres condiciones que el 02-09-2026 se
	// dieron por buenas para borrar `Secuencia`, y las cumple entera:
	//
	//	1. VencimientosDe no la construye, o sea que ningun paquete.json
	//	   puede declararla (es lo que aquel comentario llamo "el corpus nunca
	//	   pudo usarla");
	//	2. ningun reloj contado la pide: el barrido del censo da cero;
	//	3. su ventana es un Intervalo cableado en la estructura, que es el
	//	   MISMO defecto por el que se midio que Secuencia no servia: un
	//	   paquete de corpus no puede saber entre que dos fechas observa un
	//	   cliente concreto.
	//
	// No se borra aqui porque borrar una primitiva es decision de producto y
	// esta puerta es de vigilancia, no de limpieza. Se declara con su cero
	// delante, que es lo que hace que la decision se tenga que tomar en vez de
	// aplazarse por no estar escrita en ningun sitio.
	"observacion": {
		Estado: PrimitivaSinCablear,
		Motivo: "el motor la tiene y VencimientosDe no la construye, asi que ningun paquete " +
			"puede declararla sin tocar Go (invariante 2). Y el cardinal es CERO: ningun " +
			"reloj contado la pide. Con las dos cosas a la vez esta en el mismo sitio en " +
			"el que estaba `Secuencia` el dia que se borro, mas el mismo defecto de " +
			"diseno (su `Ventana` es un Intervalo fijado en la estructura, que un paquete " +
			"no puede saber). CANDIDATA A BORRADO: decidirlo, no aplazarlo.",
		RelojesEsperando: 0,
		DondeSeCuentan: "docs/censo-relojes.md, barrido del 02-09-2026: cero apariciones " +
			"como forma de reloj",
	},
}
