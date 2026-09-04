package ia

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/puertos"
)

// EL INTERRUPTOR ES LA SEGUNDA PUERTA DEL INVARIANTE 9, y este fichero es el
// que hace que sea una puerta y no una casilla.
//
// TODOS LOS CASOS FIJAN LA VARIABLE ELLOS MISMOS, incluidos los que quieren que
// este apagada. No es ceremonia: la puerta de CI corre la suite entera con
// PLAZUM_SIN_IA=1 puesta en el entorno, asi que un test que dependa del valor
// ambiental daria un resultado en un paso y otro en el otro, y las dos veces
// verde. Un test que cambia de significado segun el paso que lo corre no
// comprueba nada, y ademas es indistinguible de uno que si.

func TestElInterruptorDistingueLasTresFormasDeLaNada(t *testing.T) {
	casos := []struct {
		valor    string
		poner    bool
		apagada  bool
		ilegible bool
	}{
		// AUSENTE: la IA funciona. Es el defecto del producto.
		{poner: false, apagada: false},
		// PRESENTE Y VACIA: es la otra forma de la nada, y aqui significa lo
		// mismo. Quien exporta una variable vacia no esta pidiendo apagar nada.
		{valor: "", poner: true, apagada: false},
		// AFIRMATIVOS.
		{valor: "1", poner: true, apagada: true},
		{valor: "true", poner: true, apagada: true},
		{valor: "TRUE", poner: true, apagada: true},
		{valor: "si", poner: true, apagada: true},
		{valor: "yes", poner: true, apagada: true},
		{valor: "on", poner: true, apagada: true},
		{valor: "  1  ", poner: true, apagada: true},
		// NEGATIVOS. ESTOS SON EL MOTIVO DE QUE ESTA FUNCION EXISTA: con el
		// `os.Getenv(v) != ""` obvio, los cuatro APAGARIAN la IA, que es lo
		// contrario de lo que dicen.
		{valor: "0", poner: true, apagada: false},
		{valor: "false", poner: true, apagada: false},
		{valor: "no", poner: true, apagada: false},
		{valor: "off", poner: true, apagada: false},
		// PRESENTE Y NO INTERPRETABLE: la tercera forma. Error, nunca defecto.
		{valor: "quiza", poner: true, ilegible: true},
		{valor: "2", poner: true, ilegible: true},
		{valor: "sin-ia", poner: true, ilegible: true},
		{valor: "verdadero", poner: true, ilegible: true},
	}

	for _, c := range casos {
		nombre := "sin poner"
		if c.poner {
			nombre = "valor " + strconvQuote(c.valor)
		}
		t.Run(nombre, func(t *testing.T) {
			// t.Setenv primero para que quede registrada la restauracion, y
			// despues se quita si el caso lo pide.
			t.Setenv(Variable, c.valor)
			if !c.poner {
				if err := os.Unsetenv(Variable); err != nil {
					t.Fatal(err)
				}
			}
			apagada, err := Apagada()
			if c.ilegible {
				if !errors.Is(err, ErrInterruptorIlegible) {
					t.Fatalf("con %q sale (%v, %v).\n"+
						"  Un valor presente que no se entiende NO puede tomar un defecto en\n"+
						"  ninguna de las dos direcciones: si se toma por apagado, la IA se\n"+
						"  cae sin motivo; si se toma por encendido, los datos de un CISO\n"+
						"  salen de su maquina cuando el creia haberlo apagado.",
						c.valor, apagada, err)
				}
				if !strings.Contains(err.Error(), "Arreglo:") {
					t.Errorf("el error no dice como arreglarlo: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("con %q sale error: %v", c.valor, err)
			}
			if apagada != c.apagada {
				t.Fatalf("con %q apagada=%v, se esperaba %v", c.valor, apagada, c.apagada)
			}
		})
	}
}

// EL FALLO CONCRETO QUE ESTE DISENO EVITA, escrito como test para que no se
// pueda volver atras sin que algo se ponga rojo.
func TestElInterruptorNoSeEscribeConGetenvDistintoDeVacio(t *testing.T) {
	for _, negativo := range []string{"0", "false", "no", "off"} {
		t.Setenv(Variable, negativo)
		apagada, err := Apagada()
		if err != nil {
			t.Fatalf("con %q: %v", negativo, err)
		}
		if apagada {
			t.Fatalf("%s=%q apaga la IA.\n"+
				"  Eso es lo que hace `os.Getenv(Variable) != \"\"`, y es lo contrario de\n"+
				"  lo que escribio quien lo puso. Un interruptor que hace lo contrario de\n"+
				"  lo que se le pide es peor que no tenerlo.", Variable, negativo)
		}
		// Y con el negativo puesto, un adaptador de modelo se puede construir.
		if err := ExigeEncendida(); err != nil {
			t.Fatalf("con %s=%q, ExigeEncendida da %v", Variable, negativo, err)
		}
	}
}

func TestExigeEncendidaParaCuandoDebe(t *testing.T) {
	t.Run("apagada: para y lo dice sin llamarlo fallo", func(t *testing.T) {
		t.Setenv(Variable, "1")
		err := ExigeEncendida()
		if !errors.Is(err, ErrIADesactivada) {
			t.Fatalf("con el interruptor puesto, ExigeEncendida da %v", err)
		}
		if !strings.Contains(err.Error(), "no es un fallo") {
			t.Errorf("el mensaje presenta el modo sin IA como una averia: %v", err)
		}
	})
	t.Run("ilegible: para, y NO por ErrIADesactivada", func(t *testing.T) {
		t.Setenv(Variable, "quiza")
		err := ExigeEncendida()
		if !errors.Is(err, ErrInterruptorIlegible) {
			t.Fatalf("con un valor ilegible, ExigeEncendida da %v", err)
		}
		if errors.Is(err, ErrIADesactivada) {
			t.Error("un valor ilegible se esta tratando como 'apagada'. No es lo mismo: " +
				"uno es una decision del operador y el otro es que no se sabe cual es")
		}
	})
	t.Run("CONTROL POSITIVO: encendida deja pasar", func(t *testing.T) {
		t.Setenv(Variable, "0")
		if err := ExigeEncendida(); err != nil {
			t.Fatalf("con la IA encendida, ExigeEncendida da %v.\n"+
				"  Sin este caso, los dos de arriba los pasaria igual una funcion que "+
				"devuelve error siempre.", err)
		}
	})
}

// EL VERIFICADOR NO ES IA Y TIENE QUE SEGUIR FUNCIONANDO CON LA IA APAGADA.
//
// Es la mitad que se olvida del invariante 9: "la suite entera en verde con
// PLAZUM_SIN_IA=1" no significa que todo se apague, significa que lo que no es
// IA sigue en pie. El verificador de citas es una comparacion de cadenas y un
// sha256: si se apagara con el interruptor, el modo sin IA seria un modo sin
// producto.
func TestElVerificadorFuncionaIgualConLaIAApagada(t *testing.T) {
	t.Setenv(Variable, "1")
	if err := ExigeEncendida(); err == nil {
		t.Fatal("el interruptor no esta puesto en este caso")
	}
	v, tr, _, _, _ := verificadorDePrueba(t)
	p := puertos.Propuesta{
		Cita:       "a mas tardar 72 horas despues de que haya tenido constancia de ella",
		HashFuente: tr.Hash,
	}
	if _, err := v.Verificar(p); err != nil {
		t.Fatalf("con la IA apagada, el verificador de citas ha dejado de verificar: %v", err)
	}
}

func strconvQuote(s string) string { return `"` + s + `"` }
