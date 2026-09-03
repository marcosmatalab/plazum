package corpus

import (
	"errors"
	"fmt"
)

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

// Explicacion es el motivo COMPUESTO con su cardinal, y existe porque la prosa
// es la mitad que caduca.
//
// El 03-09-2026 el cardinal de `preaviso` subio de 7 a 8 y el motivo siguio
// diciendo que los siete estaban FUERA de los 12 marcos de la v1. El octavo
// esta DENTRO, o sea que la deuda habia pasado a bloquear la v1 y este fichero
// afirmaba lo contrario, con la forma de una decision tomada. El cardinal tenia
// puerta; la explicacion no. La prosa es la que caduca porque nadie la vigila.
//
// Asi que el numero se DERIVA del campo que la puerta vigila y no se escribe:
// `Motivo` cuenta el porque y no puede repetir la cifra, y de juntarlos se
// encarga esto.
func (d DeclaracionDePrimitiva) Explicacion() string {
	if d.Estado == PrimitivaEnUso {
		return "en uso: hay paquetes publicados que la declaran"
	}
	return fmt.Sprintf("%s (%s): %s. La esperan %d relojes contados en %s",
		d.Estado, "primitiva del motor", d.Motivo, d.RelojesEsperando, d.DondeSeCuentan)
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
			"codigo: un paquete puede usarla hoy sin tocar Go, lo que falta es escribir " +
			"los relojes. Y la deuda TOCA la v1: entre los que la esperan esta el art. " +
			"60.4.f del AI Act, la prorroga de las pruebas en condiciones reales, que " +
			"exige notificacion previa a la autoridad de vigilancia del mercado. Es un " +
			"plazo que corre hacia atras desde una fecha que ELIGE el obligado, o sea " +
			"exactamente esta primitiva. Los demas caen fuera de los 12 marcos del " +
			"escaparate. CORREGIDO el 03-09-2026: el motivo anterior afirmaba que TODOS " +
			"quedaban fuera de la v1, y lo seguia afirmando despues de que el cardinal " +
			"subiera, porque el cardinal tenia puerta y la prosa no",
		RelojesEsperando: 8,
		DondeSeCuentan: "docs/censo-relojes.md, la familia del preaviso contractual, mas " +
			"docs/hallazgos-censo-b.md H3 (el del art. 60.4.f del AI Act)",
	},
}
