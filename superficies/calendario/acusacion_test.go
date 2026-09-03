package calendario

import (
	"net/http"
	"strings"
	"testing"
)

// LA COLOCACION AFIRMA, y por eso se vigila.
//
// # El P1 que lo trae
//
// Cuatro de las secciones que abren una cifra del pie se calculan ANTES de la
// aplicabilidad: `estrenan`, `ya cesados`, `empiezan tarde` e `ilegibles`. Su
// numero y su lista cuadran (que era el requisito de D11-c) y cuentan TODO el
// corpus, te alcance o no. Ni el rotulo ni las filas afirman en ningun sitio
// que la obligacion sea tuya.
//
// Y aun asi acusan, porque estan en TU calendario. Un lector que abre una
// pagina titulada «Calendario de Acme SL» y encuentra a media altura «21 que
// empiezan a obligar mas alla de esta ventana» lee que esas 21 le van a obligar.
// La pagina no lo dice; el sitio si. En una pantalla que ensena pasado, esa es
// la unica clase de error que un producto de cumplimiento no puede cometer.
//
// # Que se arregla aqui y que NO, con su cardinal
//
// SE ARREGLA LA COLOCACION: las secciones que hablan de TI (las tres que se
// calculan DESPUES de la aplicabilidad: `alcanzados`, `mas alla`, `antes de
// vigor`) van arriba, con lo tuyo, y las CUATRO del corpus bajan detras de todo
// lo que si te obliga. Un lector que llega al final ya ha visto todo lo suyo.
//
// NO SE ARREGLA LA NOTA, y es la mitad que falta: una frase al frente del bloque
// diciendo que esa lista sale del corpus entero y no de tus respuestas. Necesita
// UNA clave de catalogo (`calendario.pantalla.descarte.no_es_tuyo`) y
// `adaptadores/catalogo/cadenas/` es de otra columna en este tramo. Va pedida en
// docs/hallazgos-d11.md, con su texto propuesto, como P1.
//
// # Por que esta puerta mira el ORDEN y no la declaracion
//
// Porque el reparto lo hace un mapa en `seccionesDe`, y comprobarlo contra ese
// mismo mapa seria preguntarle a la respuesta por la respuesta. Lo que le pasa
// al lector es el ORDEN DE LA PAGINA, asi que es el orden lo que se mide.

// anclasQueHablanDeTi son las secciones que salen DESPUES de la aplicabilidad.
//
// El descargo de lo vencido y las fechas de los meses entran aqui aunque no sean
// secciones de cifra abierta: lo que se afirma es que TODO lo tuyo va delante,
// no solo lo tuyo que abre un numero.
var anclasQueHablanDeTi = []string{
	AnclaVencidas, AnclaFechas, AnclaEstrenos, AnclaCeses, AnclaSinFecha,
	AnclaAlcanzados, AnclaMasAlla, AnclaAntesDeVigor,
}

// anclasDelCorpusEntero son las que se calculan ANTES de la aplicabilidad.
var anclasDelCorpusEntero = []string{
	AnclaEstrenan, AnclaYaCesados, AnclaEmpiezanTarde, AnclaIlegibles,
}

func TestLoQueNoSaleDeTusRespuestasVaDetrasDeTodoLoTuyo(t *testing.T) {
	s, _ := pantallaDePrueba(t, fuenteDoble{d: Derivado{
		Calendario: calendarioConVencidas(), Organizacion: "Acme SL"}, hay: true})
	codigo, cuerpo := pedir(t, s, BasePorDefecto+"/")
	if codigo != http.StatusOK {
		t.Fatalf("GET %s/ ha respondido %d", BasePorDefecto, codigo)
	}

	donde := func(ancla string) int {
		i := strings.Index(cuerpo, `id="`+ancla+`"`)
		if i < 0 {
			// SE PARA. Una seccion que no se pinta dejaria esta puerta
			// comparando ocho posiciones contra tres, y el verde vendria de lo
			// que falta y no de lo que esta bien colocado.
			t.Fatalf("la pagina no pinta la seccion #%s, asi que esta puerta no puede decir "+
				"donde esta: el dato sintetico ha dejado de llenarla", ancla)
		}
		return i
	}

	// El ultimo sitio que ocupa algo tuyo.
	ultimoTuyo, cualTuyo := -1, ""
	for _, a := range anclasQueHablanDeTi {
		if i := donde(a); i > ultimoTuyo {
			ultimoTuyo, cualTuyo = i, a
		}
	}
	for _, a := range anclasDelCorpusEntero {
		if i := donde(a); i < ultimoTuyo {
			t.Errorf(`la seccion #%s sale ANTES de #%s, que es tuya.

  #%s se calcula antes de la aplicabilidad: cuenta el corpus entero, te alcance o
  no. Puesta entre lo tuyo, la colocacion insinua que te obligara, y ni el rotulo
  ni las filas lo dicen. En una pantalla que ensena pasado, eso es acusar en
  falso, que es el unico error que este producto no puede cometer ni una vez.`,
				a, cualTuyo, a)
		}
	}
	// Y LAS DOS LISTAS SON LAS DE VERDAD: si el reparto de `seccionesDe` cambia
	// y esta lista no, la puerta seguiria verde vigilando un reparto que ya no
	// existe. Se cruza con la vista en los dos sentidos.
	var v Vista
	v.rellenarCon(Derivado{Calendario: calendarioConVencidas(), Organizacion: "Acme SL"})
	tuyas := map[string]bool{}
	for _, d := range v.Tuyas {
		if !d.Tuyo {
			t.Errorf("la seccion #%s esta en Tuyas y no se declara tuya", d.Ancla)
		}
		tuyas[d.Ancla] = true
	}
	for _, d := range v.Descartes {
		if d.Tuyo {
			t.Errorf("la seccion #%s se declara tuya y esta con las del corpus", d.Ancla)
		}
		if tuyas[d.Ancla] {
			t.Errorf("la seccion #%s sale en las dos listas", d.Ancla)
		}
	}
	for _, a := range anclasDelCorpusEntero {
		if tuyas[a] {
			t.Errorf("la seccion #%s se calcula ANTES de la aplicabilidad y la vista la pone "+
				"con lo tuyo: cuenta el corpus entero y decir que es tuya es acusar", a)
		}
	}
}

// CONTROL POSITIVO DEL REPARTO: la rama «es tuya» la recorre alguien.
//
// Un reparto en el que todo cayera del lado del corpus pasaria la puerta de
// arriba entera (no habria nada tuyo delante que violar) y no habria repartido
// nada. Es M47 aplicado aqui: una rama que ninguna entrada recorre es una rama
// que no existe.
func TestElRepartoDeSeccionesRecorreLasDosRamas(t *testing.T) {
	var v Vista
	v.rellenarCon(Derivado{Calendario: calendarioConVencidas(), Organizacion: "Acme SL"})
	if len(v.Tuyas) == 0 {
		t.Error("ninguna seccion cae del lado de lo tuyo: la rama no la recorre nadie y la " +
			"separacion no separa nada")
	}
	if len(v.Descartes) == 0 {
		t.Error("ninguna seccion cae del lado del corpus: idem por el otro lado")
	}
	// EL VALOR CERO ES EL RESTRICTIVO, y se comprueba: una cifra que el mapa de
	// reparto no nombra tiene que bajar con las del corpus, que es el lado que
	// no afirma nada sobre el lector. Se prueba con una cifra inventada, o sea
	// FUERA de la lista que el propio test conoce.
	cal := calendarioConVencidas()
	inventada := []CifraDeLaCuenta{{Campo: "YaCesados", Clave: "k", N: cal.HitosYaCesados,
		Derivacion: CifraConSeccion, Ancla: "inventada", Cuadre: CuadreFilas}}
	tuyas, delCorpus := seccionesDe(cal, inventada)
	if len(tuyas) != 0 {
		t.Error("una cifra que el reparto no nombra ha subido con lo tuyo. El valor cero " +
			"tiene que ser el restrictivo: el olvido baja al lado que no acusa")
	}
	if len(delCorpus) != 1 {
		t.Errorf("la cifra sin clasificar tenia que bajar con las del corpus y hay %d",
			len(delCorpus))
	}
}
