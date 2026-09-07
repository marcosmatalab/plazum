package corpus

import (
	"errors"
	"fmt"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/estado"
	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// EL PUENTE ENTRE LA PRUEBA DEL PAQUETE Y LA PRUEBA DEL EXPEDIENTE.
//
// Es el eslabon 2 de los siete de D-22, y hasta hoy no existia: `estado.Prueba`
// solo se construia en `herramientas/generardemo`, desde un escenario de demo y
// nunca desde el corpus. O sea que el formato existia, el motor existia, y no
// habia una sola linea que los uniera.
//
// # POR QUE VIVE AQUI Y NO EN `nucleo/estado`
//
// Porque la direccion importa y no es simetrica. `nucleo/estado` es la maquina
// de estados y no sabe nada de paquetes: si importara `nucleo/corpus`, el motor
// que decide si algo consta dependeria del formato de los datos normativos, y
// entonces un cambio de esquema tocaria la maquina. Al reves no: el corpus ya
// sabe que existe un motor al que alimenta.
//
// # EL EMPAREJAMIENTO, DICHO (invariante 7)
//
// La prueba del paquete casa con la observacion del recolector por
// `Prueba.ID` <-> `Observacion.Prueba`, y con la obligacion por
// `Prueba.Obligacion` -> `estado.Prueba.Control`. **Los dos son identidades y
// ninguno es una posicion**: reordenar las pruebas de un paquete, o las
// obligaciones, no mueve ni un emparejamiento.
//
// Y los dos estan DENTRO de lo firmado. `Prueba.ID` y `Prueba.Obligacion` viven
// en `paquete.json`, que se firma entero con Ed25519 al publicarlo;
// `Observacion.Prueba` viaja dentro de la entrada del ledger, que va firmada y
// encadenada. Nadie firma el orden de ninguna de las dos listas, y por eso
// ninguna de las dos se usa para emparejar.
//
// # EL CAMBIO DE NOMBRE ES LA TRAMPA, Y SE DICE
//
// `estado.Prueba` llama `Control` a lo que aqui se llama `Obligacion`. No es un
// sinonimo inocente: `nucleo/expediente/expediente.go:709` decide la
// aplicabilidad comparando `pr.Control` contra la lista de obligaciones
// aplicables, **y solo si `pr.Control != ""`** — o sea que una prueba que
// cruzara el puente con el control vacio se evaluaria como aplicable siempre.
// Por eso este puente se niega a producir una `estado.Prueba` sin control.
//
// # TRES CAMPOS DE NUEVE NO CRUZAN, Y HAY QUE SABER CUALES
//
//	Recurso     se queda aqui. `estado.Prueba` no lo tiene: en el motor, el
//	            recurso viaja en cada Observacion, que es donde de verdad esta.
//	Predicado   SI cruza, y no por el nombre: viaja en `Descripcion`, que es el
//	            unico hueco del tipo del expediente donde cabe QUE se evaluo.
//	            Sin el, el invariante 13 se queda a medias: sostiene
//	            `Observacion.Satisfecho` diciendo que es «el resultado de un
//	            predicado mecanico declarado en la prueba», y un auditor que
//	            lea el expediente no tendria donde ir a mirar cual.
//	Cierra      se queda aqui, y por lo mismo: es lo que separa «consta» de
//	            «consta este aspecto», y sin el una faceta en verde absuelve
//	            una obligacion documental entera.
//
// # LA APROXIMACION DEL MES, DECLARADA CON SU CARDINAL
//
// `estado.Prueba.TTL` y `SLA` son `time.Duration`, que no sabe de calendarios;
// `Prueba.TTL` es ISO-8601 y admite meses. Al cruzar, un mes se cuenta como 30
// dias. **Eso mueve fechas**, a diferencia de la comparacion de `duracionPositiva`,
// cuyo godoc dice expresamente que su aproximacion no calcula ninguna fecha:
// aqui SI se calcula, porque `estado.Calcular` hace `Recolectada.Add(TTL)`.
//
// El error maximo, contado: un P6M sobre un semestre real (181 a 184 dias) da
// 180, o sea que la caducidad se adelanta entre 1 y 4 dias. Se acepta y se dice
// porque adelantarla es el lado seguro (un dato se declara viejo antes, nunca
// despues), y porque la alternativa —cambiar el tipo de `estado.Prueba`— toca
// el formato firmado del expediente y es otra casilla.
//
// LO VIGILA: TestElPuenteNoMueveUnaFechaMasDeLoDeclarado
var (
	ErrPuenteSinPrueba   = errors.New("corpus: no se puede cruzar una prueba vacia")
	ErrPuenteSinControl  = errors.New("corpus: prueba sin obligacion al cruzar al expediente")
	ErrPuenteTTLIlegible = errors.New(
		"corpus: la prueba trae un plazo que el expediente no puede representar")
)

// DiasPorMes es como se cuenta un mes al cruzar al expediente. Ver el bloque de
// arriba: es una aproximacion declarada, no un descuido.
const DiasPorMes = 30

// AlExpediente convierte una prueba del paquete en la prueba que entiende el
// motor de estado.
//
// No se llama `Estado()` ni `Convertir()`: el nombre dice A DONDE va, porque lo
// que importa de esta funcion es que es la unica frontera por la que una prueba
// escrita en un paquete llega al documento que firma alguien.
func (pr Prueba) AlExpediente() (estado.Prueba, error) {
	if pr.ID == "" {
		return estado.Prueba{}, fmt.Errorf("%w: sin identificador", ErrPuenteSinPrueba)
	}
	if pr.Obligacion == "" {
		return estado.Prueba{}, fmt.Errorf("%w: la prueba %q. Arreglo: el linter ya lo "+
			"exige al cargar, asi que si esto salta es que alguien ha construido una "+
			"Prueba a mano. Una prueba con el control vacio se evalua como aplicable "+
			"SIEMPRE (nucleo/expediente/expediente.go:709)", ErrPuenteSinControl, pr.ID)
	}
	ttl, err := aDuracion(pr.TTL)
	if err != nil {
		return estado.Prueba{}, fmt.Errorf("%w: %s, ttl %q: %v", ErrPuenteTTLIlegible, pr.ID, pr.TTL, err)
	}
	sla, err := aDuracion(pr.SLA)
	if err != nil {
		return estado.Prueba{}, fmt.Errorf("%w: %s, sla %q: %v", ErrPuenteTTLIlegible, pr.ID, pr.SLA, err)
	}
	var activa time.Time
	if pr.Activa != "" {
		activa, err = time.Parse("2006-01-02", pr.Activa)
		if err != nil {
			return estado.Prueba{}, fmt.Errorf("%w: %s, activa %q: %v",
				ErrPuenteTTLIlegible, pr.ID, pr.Activa, err)
		}
	}
	return estado.Prueba{
		ID:         pr.ID,
		Control:    pr.Obligacion,
		TTL:        ttl,
		SLA:        sla,
		Activa:     activa,
		PassPorDef: pr.PassPorDefecto,
		// Descripcion lleva el PREDICADO y no el titulo de nada. Es el unico
		// hueco del tipo del expediente donde cabe lo que se evaluo, y sin el
		// `Satisfecho` llega al documento sin nada que lo sostenga.
		Descripcion: pr.Predicado,
	}, nil
}

// aDuracion convierte un plazo ISO-8601 del paquete a la duracion del motor.
//
// Los meses se cuentan a DiasPorMes. Ver el bloque del encabezado: la
// aproximacion mueve fechas y por eso se declara aqui y no se esconde en una
// funcion auxiliar sin comentario.
func aDuracion(s string) (time.Duration, error) {
	d, err := ventana.ParseDuracion(s)
	if err != nil {
		return 0, err
	}
	if d.Indeterminado {
		return 0, errors.New("un plazo indeterminado no se puede convertir a una duracion fija")
	}
	return time.Duration(d.Meses)*DiasPorMes*24*time.Hour +
		time.Duration(d.Dias)*24*time.Hour +
		time.Duration(d.Horas)*time.Hour +
		time.Duration(d.Mins)*time.Minute, nil
}
