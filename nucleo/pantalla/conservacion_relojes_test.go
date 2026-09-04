package pantalla

import (
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// LA LEY QUE FALTABA: todo reloj que te alcanza sale por algun lado.
//
// EL FALLO QUE LA TRAE, y es el descarte silencioso mas caro que ha tenido este
// motor. `corpus.VencimientosDe` hacia, para una `periodica` cuyo hecho de
// arranque no consta, un `return nil, nil`: lista vacia. La obligacion no
// producia fecha, no producia fila de «sin fecha» y no se contaba en ningun
// cubo. Desaparecia.
//
// El numero, medido el 29-08-2026 sobre el perfil de servicios digitales con el
// reglamento tecnico de NIS2 encendido:
//
//	55 hitos alcanzados por la aplicabilidad
//	 6 fechas
//	 3 filas sin fecha
//	--
//	46 en ningun sitio
//
// Y no lo cazaba nada. La contabilidad del calendario tiene dos particiones
// exhaustivas (por tiempo y por aplicabilidad) y las dos cuadraban: el hueco
// estaba DESPUES, entre «te alcanza» y «sale en pantalla», que era justo el
// tramo sin ley.
//
// LO QUE ESTA LEY DICE: una obligacion en vigor que la aplicabilidad alcanza
// APARECE EN ALGUN SITIO. La nada no es una respuesta. Es la version de esta
// familia que se puede comprobar sumando, igual que la conservacion de la
// contabilidad, y cubre el tramo que aquella no miraba.
//
// # LOS SITIOS SON TRES, Y ERAN DOS HASTA EL 04-09-2026
//
// La ley enumeraba `Fechas` y `SinFecha`, y con eso acusaba al producto de
// perder un reloj que el producto SI ENSENA: `aiact.art111_2` es el primero
// del corpus cuyo unico vencimiento cae MAS ALLA de la ventana, asi que sale
// en `VencimientosMasAlla`, con su fecha, comprobado ejecutando Derivar12Meses.
//
// O SEA QUE LA LEY ESTABA CORTA, NO EL PRODUCTO. Y el arreglo NO es sacar ese
// reloj del conjunto esperado, que seria bajar la afirmacion para que la puerta
// pase: es nombrar la tercera respuesta, que existia y no estaba escrita.
//
// Lo encontro el corpus, no una revision: la rebanada del corpus del tramo 3
// escribio el primer reloj que cae en ese caso, y esta ley se puso roja sobre
// dato real. Ninguna mutacion lo habria encontrado, porque nadie sabia que ese
// caso no estaba cubierto.
func TestTodoRelojAlcanzadoSaleEnAlgunSitio(t *testing.T) {
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatalf("no puedo cargar el corpus publicado: %v", err)
	}
	ahora := time.Date(2026, 8, 27, 9, 0, 0, 0, time.UTC)

	// Se recorre TODO el corpus (aplica = todo) y SIN NI UN HECHO. Es el peor
	// caso a proposito: el dia uno de un cliente, cuando ninguna cadencia tiene
	// todavia su fecha de arranque, que es exactamente el escenario en el que
	// el fallo se comia 46 relojes.
	cal := Derivar12Meses(ps, TodoAplica, ventana.Hechos{}, ahora)

	salen := map[string]bool{}
	for _, m := range cal.Meses {
		for _, f := range m.Fechas {
			salen[f.Obligacion] = true
		}
	}
	for _, s := range cal.SinFecha {
		salen[s.Obligacion] = true
	}
	// LA TERCERA RESPUESTA. Un reloj cuyo unico vencimiento cae despues de los
	// doce meses SALE, con su fecha, en la lista de mas alla de la ventana. No
	// contarla aqui hacia que la ley acusara al producto de perder lo que si
	// ensena, que es la unica clase de error que una puerta no puede cometer:
	// si acusa en falso, deja de leerse.
	for _, v := range cal.VencimientosMasAlla {
		salen[v.Obligacion] = true
	}

	// Las que deberian salir: en vigor en ese instante y con reloj declarado.
	esperadas, perdidas := 0, []string{}
	for _, p := range ps {
		for _, o := range p.Obligaciones {
			if o.Temporalidad == nil {
				continue
			}
			vigente, err := p.EnVigor(o, ahora)
			if err != nil {
				continue // la vigencia ilegible ya sale por SinFecha con su motivo
			}
			if !vigente {
				continue // los estrenos y los ceses tienen sus propias listas
			}
			esperadas++
			if !salen[o.ID] {
				perdidas = append(perdidas, o.ID)
			}
		}
	}

	// Suelo: sin el, un corpus que dejara de cargar relojes daria verde sin
	// haber comprobado ni uno.
	if esperadas < 50 {
		t.Fatalf("solo %d obligaciones con reloj en vigor: el recorrido no esta viendo el "+
			"corpus y esta ley estaria comprobando el vacio", esperadas)
	}
	// Y EL SUELO DE LA TERCERA RESPUESTA, por la misma razon que el de arriba:
	// si la lista de mas alla se vaciara, este recorrido dejaria de aportar
	// nada y la ley volveria a ser la de dos sitios SIN QUE NADIE LO NOTARA.
	// Hoy tiene contenido; el dia que no lo tenga, hay que enterarse.
	if len(cal.VencimientosMasAlla) == 0 {
		t.Errorf("la lista de vencimientos mas alla de la ventana esta vacia, asi que la " +
			"tercera respuesta de esta ley no la recorre ninguna entrada.\n" +
			"  Es M47: una rama que nadie alcanza es una rama que no existe, y la mutacion " +
			"la deja verde porque no hay nada que romper.")
	}
	if len(perdidas) > 0 {
		muestra := perdidas
		if len(muestra) > 8 {
			muestra = muestra[:8]
		}
		t.Errorf(`%d de %d obligaciones con reloj en vigor NO salen ni con fecha ni sin ella.

  Desaparecen: no hay fila, no hay motivo y no hay numero. Para quien lee el
  calendario es indistinguible de que no existieran, que es el peor error que
  puede cometer un producto de cumplimiento.

  Una obligacion que espera un dato del operador NO es una obligacion que no
  existe: es el estado normal de toda cadencia el dia uno de un cliente.

  Las primeras: %v`, len(perdidas), esperadas, muestra)
	}
	// El log tranquilizador SOLO si no ha saltado nada: "todas con fila"
	// impreso al lado de una lista de perdidas es una afirmacion falsa en la
	// salida de la puerta. La mutacion M30 lo enseno tal cual.
	if !t.Failed() {
		t.Logf("%d obligaciones con reloj en vigor, todas con fila", esperadas)
	}
}

// CONTROL NEGATIVO de la ley: se comprueba que sabe decir que NO.
//
// Sin esto, el verde de arriba podria venir de un recorrido que no mira nada. Se
// le da un calendario al que le falta a proposito una obligacion y se exige que
// el mismo emparejamiento la eche en falta.
func TestLaLeyDeConservacionDeRelojesCazaUnaAusencia(t *testing.T) {
	salen := map[string]bool{"a": true, "b": true}
	todas := []string{"a", "b", "c"}
	var perdidas []string
	for _, id := range todas {
		if !salen[id] {
			perdidas = append(perdidas, id)
		}
	}
	if len(perdidas) != 1 || perdidas[0] != "c" {
		t.Fatalf("el emparejamiento de la ley no ve la ausencia: %v", perdidas)
	}
}
