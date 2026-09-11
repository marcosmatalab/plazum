package ventana_test

import (
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// UNA DURACION QUE NO CABE ES UN ERROR, NO OTRA DURACION.
//
// # El agujero que cierra, y por que llega desde un paquete.json
//
// `ParseDuracion` leia los numeros con `v, _ := strconv.Atoi(x)`, con el error
// descartado. El razonamiento parecia solido —la expresion regular solo casa
// `\d+`, asi que no puede haber letras— y le falta la mitad: **`Atoi` tambien
// falla por DESBORDAMIENTO, y cuando falla devuelve el valor SATURADO**. O sea
// que `P99999999999999999999D` salia sin error con `Dias = MaxInt64`.
//
// Lo que pasa despues, medido el 11-09-2026 y no supuesto:
//
//	naturales  base.AddDate(0, 0, MaxInt64) DA LA VUELTA y el vencimiento cae el
//	           dia ANTERIOR a la base. Una obligacion que vence antes de empezar
//	hábiles    el bucle de calendario.go descuenta de uno en uno desde MaxInt64:
//	           no termina, y la pantalla que lo pida se cuelga
//
// Y el camino de entrada es el peor posible: **un `paquete.json`**. El linter
// carga el paquete sin protestar porque `ParseDuracion` no protesta, asi que un
// cambio de DATOS cuelga el producto. Eso es exactamente lo que el invariante 2
// dice que no puede pasar: una norma vive en su paquete de datos, y un paquete
// de datos no puede ser un vector de ejecucion.
//
// Es la tercera forma de la nada del invariante 8 —presente y no
// interpretable— en la aritmetica del reloj legal.
//
// # EL TECHO NO ES SOLO EL DESBORDAMIENTO, y por eso hay dos afirmaciones
//
// Arreglar solo el `Atoi` deja fuera `P3000000000D`, que cabe en un int64, no
// desborda nada y sigue colgando el bucle de habiles durante dias. Asi que hay
// un techo de dominio, y va derivado de una sola cifra explicable
// (`AnosMaximosDeUnaDuracion`) en vez de cuatro numeros sueltos.
//
// El techo se eligio con el corpus medido delante el 11-09-2026, no a ojo: de
// las 314 duraciones ISO que hay en `paquetes/`, la mayor en meses es `P120M`,
// la mayor en dias `P90D` y la mayor en horas `PT72H`. Cien anos deja tres
// ordenes de magnitud de holgura sobre el plazo legal mas largo del corpus, asi
// que esta puerta no puede saltar sobre trabajo legitimo, que es la pregunta que
// hay que hacerse antes de escribir una.
//
// # EL CONTROL POSITIVO DEL CORPUS NO SE ESCRIBE AQUI, y se dice donde vive
//
// Seria una tercera implementacion de la misma cifra. Ya existe: el linter carga
// `paquetes/` entero en `paquetes_test.go`, y toda duracion del corpus pasa por
// `ParseDuracion` al cargarse. Si este techo dejara fuera una duracion real, ese
// test se pone rojo solo.
func TestUnaDuracionQueNoCabeNoSeInterpretaComoOtraCosa(t *testing.T) {
	for _, c := range []struct {
		entrada string
		porQue  string
	}{
		{
			entrada: "P99999999999999999999D",
			porQue: "es el caso literal: Atoi desborda y devuelve MaxInt64, y AddDate da " +
				"la vuelta al ano. El vencimiento cae ANTES de la base",
		},
		{
			entrada: "P99999999999999999999M",
			porQue:  "lo mismo por el lado de los meses, que se suman antes que los dias",
		},
		{
			entrada: "PT99999999999999999999H",
			porQue: "y por el de las horas, que ademas se convierten a time.Duration, que " +
				"es int64 de nanosegundos y desborda mucho antes",
		},
		{
			entrada: "PT99999999999999999999M",
			porQue:  "y los minutos, que es el cuarto campo y el que se olvida",
		},
		{
			entrada: "P" + strings.Repeat("9", 400) + "D",
			porQue:  "un numero absurdamente largo sigue casando con \\d+ y no puede colar",
		},
		{
			entrada: "P3000000000D",
			porQue: "ESTE NO DESBORDA NADA: cabe de sobra en un int64. Cuelga igual el " +
				"bucle de dias habiles, que descuenta de uno en uno. Por eso el arreglo " +
				"no puede ser solo mirar el error de Atoi",
		},
	} {
		t.Run(c.entrada[:min(len(c.entrada), 28)], func(t *testing.T) {
			d, err := ventana.ParseDuracion(c.entrada)
			if err == nil {
				t.Fatalf(`%q se ha interpretado como %+v EN VEZ DE DAR ERROR.

  Por que importa: %s

  Esto llega desde un paquete.json. El linter carga el paquete porque
  ParseDuracion no protesta, y entonces un cambio de DATOS cuelga el producto o
  le da a un cliente un vencimiento anterior a la base. El invariante 2 dice que
  toda norma vive en su paquete de datos; un paquete de datos que ejecuta no
  entra en ese trato.

  Arreglo: el error de strconv.Atoi NO se descarta (falla por desbordamiento y
  devuelve el valor saturado, que es la tercera forma de la nada del invariante
  8), y ademas hay un techo de dominio para lo que cabe pero no termina.`,
					c.entrada, d, c.porQue)
			}
		})
	}
}

// EL CONTROL POSITIVO, Y EL BORDE EXACTO DEL TECHO.
//
// Sin esto, la puerta de arriba pasaria con un `ParseDuracion` que devolviera
// error SIEMPRE, y eso no se distingue desde fuera de uno que funciona. Y el
// borde va porque un techo que se comprueba con `>=` en vez de `>` deja fuera el
// valor que si es valido, y ese fallo es invisible mientras nadie lo escriba.
func TestElTechoDeUnaDuracionDejaPasarLoQueCabe(t *testing.T) {
	// Lo que el corpus usa de verdad, medido el 11-09-2026 sobre paquetes/: el
	// maximo en meses es P120M, en dias P90D y en horas PT72H.
	for _, s := range []string{"P1M", "P120M", "P14D", "P90D", "PT24H", "PT72H", "PT30M", "P1MT12H"} {
		if _, err := ventana.ParseDuracion(s); err != nil {
			t.Errorf("%q es una duracion del corpus y ha dado error: %v.\n"+
				"  Una puerta que no se puede satisfacer se afloja, y aflojarla aqui es "+
				"volver a dejar entrar el desbordamiento", s, err)
		}
	}

	// EL BORDE, derivado de la constante y no escrito: justo el techo entra y
	// uno mas no.
	max := ventana.AnosMaximosDeUnaDuracion
	for _, c := range []struct {
		dentro, fuera string
	}{
		{sprintfD("P%dM", 12*max), sprintfD("P%dM", 12*max+1)},
		{sprintfD("P%dD", 366*max), sprintfD("P%dD", 366*max+1)},
		{sprintfD("PT%dH", 366*max*24), sprintfD("PT%dH", 366*max*24+1)},
		{sprintfD("PT%dM", 366*max*24*60), sprintfD("PT%dM", 366*max*24*60+1)},
	} {
		if _, err := ventana.ParseDuracion(c.dentro); err != nil {
			t.Errorf("%q es justo el techo y tiene que entrar: %v", c.dentro, err)
		}
		if _, err := ventana.ParseDuracion(c.fuera); err == nil {
			t.Errorf("%q pasa del techo y ha entrado", c.fuera)
		}
	}

	// Y QUE LO QUE ENTRA SIGUE SIENDO CALCULABLE, que es de lo que iba todo
	// esto: el techo mas alto que se acepta no da la vuelta al calendario.
	base := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	d, err := ventana.ParseDuracion(sprintfD("P%dD", 366*max))
	if err != nil {
		t.Fatal(err)
	}
	if fin := base.AddDate(0, 0, d.Dias); !fin.After(base) {
		t.Errorf("con la duracion mas larga que se acepta, el vencimiento (%s) NO es "+
			"posterior a la base (%s): el techo sigue permitiendo dar la vuelta",
			fin.Format(time.RFC3339), base.Format(time.RFC3339))
	}
}

func sprintfD(formato string, n int) string {
	return strings.Replace(formato, "%d", itoa(n), 1)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
