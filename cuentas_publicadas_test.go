package plazum

import (
	"os"
	"regexp"
	"strconv"
	"testing"

	"plazum/nucleo/corpus"
)

// LAS CUENTAS QUE EL PROYECTO PUBLICA DE SI MISMO TIENEN QUE SALIR DEL CORPUS.
//
// EL FALLO, y es de la familia "una segunda lista es una lista que se queda
// vieja": el README decia "57 hitos y 126 casos dorados", paquetes/CORPUS.md
// decia 58 y 129, y la cuenta que imprime `plazum calendario` decia "39 relojes
// instalados". Tres numeros distintos para lo mismo en tres sitios, y el unico
// que salia del corpus de verdad era el del calendario, que ademas contaba en
// otra unidad. Nadie mintio: se anadio el art. 111.4 del AI Act con sus tres
// dorados y solo se actualizo uno de los tres sitios, que es lo que pasa
// siempre.
//
// LA UNIDAD ES EL HITO en todo lo que ve el usuario, porque es lo que produce
// fechas. Una obligacion con tres hitos escalonados (alerta, notificacion,
// informe final) da tres fechas, y llamarla "un reloj" esconde dos tercios del
// trabajo que le espera al operador.
//
// Esta puerta no comprueba prosa: extrae los numeros con una expresion regular
// y los compara con lo que devuelve `corpus.Cargar`. Si alguien reescribe la
// frase y se lleva el numero por delante, la puerta se pone roja diciendo que
// no encontro la cuenta, que es lo correcto: una cuenta que se puede borrar sin
// que nadie se entere vuelve a ser una cuenta que se queda vieja.

// contarCorpusPublicado devuelve los hitos de reloj y los dorados que hay de
// verdad en paquetes/, en la unidad que se publica.
func contarCorpusPublicado(t *testing.T) (hitos, dorados int) {
	t.Helper()
	ps, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatalf("el corpus publicado no carga: %v", err)
	}
	for _, p := range ps {
		for _, o := range p.Obligaciones {
			tm := o.Temporalidad
			if tm == nil {
				continue
			}
			// Sin `hitos` la temporalidad declara UNO en el campo `hito`: el
			// suelo es uno y nunca cero. Es la misma regla que
			// pantalla.hitosDeclarados, y si las dos se separan la puerta lo
			// dira, porque la de arriba alimenta la pantalla y esta el README.
			if n := len(tm.Hitos); n > 0 {
				hitos += n
			} else {
				hitos++
			}
		}
		dorados += len(p.Dorados)
	}
	return hitos, dorados
}

func numeroAntesDe(t *testing.T, fichero, patron string) int {
	t.Helper()
	b, err := os.ReadFile(fichero)
	if err != nil {
		t.Fatalf("no se puede leer %s: %v", fichero, err)
	}
	re := regexp.MustCompile(patron)
	m := re.FindSubmatch(b)
	if m == nil {
		t.Fatalf("en %s no aparece la cuenta que busca el patron %q. Si has reescrito la "+
			"frase, vuelve a poner el numero: una cuenta publicada que se puede borrar sin "+
			"que nadie se entere es una cuenta que se queda vieja", fichero, patron)
	}
	n, err := strconv.Atoi(string(m[1]))
	if err != nil {
		t.Fatalf("en %s la cuenta %q no es un numero", fichero, m[1])
	}
	return n
}

func TestLasCuentasPublicadasSalenDelCorpusYNoDeLaMemoria(t *testing.T) {
	hitos, dorados := contarCorpusPublicado(t)

	for _, c := range []struct {
		fichero string
		patron  string
		quiero  int
		unidad  string
	}{
		{"README.md", `(\d+) hitos y \d+ casos dorados`, hitos, "hitos"},
		{"README.md", `\d+ hitos y (\d+) casos dorados`, dorados, "dorados"},
		{"paquetes/CORPUS.md", `\*\*(\d+) hitos de reloj y \d+ dorados\*\*`, hitos, "hitos"},
		{"paquetes/CORPUS.md", `\*\*\d+ hitos de reloj y (\d+) dorados\*\*`, dorados, "dorados"},
	} {
		if got := numeroAntesDe(t, c.fichero, c.patron); got != c.quiero {
			t.Errorf("%s dice %d %s y el corpus tiene %d. Gana el corpus: se actualiza el "+
				"documento, no se relaja la puerta", c.fichero, got, c.unidad, c.quiero)
		}
	}
}

// LA UNIDAD, dicha en voz alta. Que los dos numeros cuadren no basta si cada
// sitio los llama de otra forma: "relojes" a veces significa obligacion con
// temporalidad y a veces hito, y esas dos cosas se diferencian en 19 unidades
// hoy (39 obligaciones, 58 hitos). Donde se publica una cuenta, se nombra la
// unidad.
func TestLasCuentasPublicadasDicenSuUnidad(t *testing.T) {
	for _, c := range []struct{ fichero, quiero string }{
		{"README.md", "hitos"},
		{"paquetes/CORPUS.md", "hitos de reloj"},
	} {
		b, err := os.ReadFile(c.fichero)
		if err != nil {
			t.Fatalf("no se puede leer %s: %v", c.fichero, err)
		}
		if !regexp.MustCompile(`\d+ ` + regexp.QuoteMeta(c.quiero)).Match(b) {
			t.Errorf("%s publica una cuenta sin decir que la unidad es %q", c.fichero, c.quiero)
		}
	}
}
