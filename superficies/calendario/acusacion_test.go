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
// Y DESDE EL 04-09-2026 SE ARREGLA LA NOTA, que era la mitad que faltaba: una
// frase al frente del bloque diciendo que esa lista sale del corpus entero y no
// de tus respuestas. Necesitaba UNA clave de catalogo
// (`calendario.pantalla.descarte.no_es_tuyo`), que entra con este mismo commit.
//
// POR QUE LA COLOCACION SOLA NO BASTABA, aunque el hallazgo la diera por media
// solucion. Bajar las cuatro secciones quita la insinuacion de que te obligan y
// NO dice lo que pasa de verdad, asi que deja al lector con la lectura contraria
// y igual de falsa: un bloque de listas cortas al final de TU calendario se lee
// como que plazum ya lo miro y decidio que eso no era tuyo. Eso es absolver de
// mas en silencio, que es el error simetrico de acusar en falso y es peor, porque
// una acusacion la corrige quien la lee y una absolucion la descubre quien te
// inspecciona. La nota es lo unico que dice la verdad entera: plazum todavia no
// ha mirado si alguna de estas te alcanza.
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

// LA NOTA, CON SUS DOS CONTROLES.
//
// # Por que el descargo necesita control POSITIVO y no basta con la mutacion
//
// Es M47 literal, y esa leccion la pago esta misma pantalla: un descargo que
// ninguna entrada alcanza es un descargo que no existe, y la mutacion lo deja
// verde porque no hay nada que romper. Asi que la rama se RECORRE: se pide la
// pagina con un calendario que si pinta el bloque del corpus entero, y se afirma
// que la nota sale CON el, delante de la primera seccion y no en un pie.
//
// # Y el negativo, que es la otra mitad
//
// Sin bloque no hay nota. Una frase que dijera «esta lista sale del corpus
// entero» sin ninguna lista debajo es ruido, y ademas dejaria esta puerta verde
// sin haber recorrido nada: una nota que sale siempre pasa el control positivo
// diga lo que diga la pagina.
func TestLaNotaDelCorpusEnteroVaConSuBloqueYDelanteDeEl(t *testing.T) {
	// CONTROL POSITIVO.
	s, esp := pantallaDePrueba(t, fuenteDoble{d: Derivado{
		Calendario: calendarioConVencidas(), Organizacion: "Acme SL"}, hay: true})
	codigo, cuerpo := pedir(t, s, BasePorDefecto+"/")
	if codigo != http.StatusOK {
		t.Fatalf("GET %s/ ha respondido %d", BasePorDefecto, codigo)
	}
	if !esp.pidio("calendario.pantalla.descarte.no_es_tuyo") {
		t.Fatal("la pagina no pide la nota del bloque del corpus entero, asi que la mitad " +
			"que faltaba del arreglo de la colocacion sigue faltando")
	}
	nota := strings.Index(cuerpo, marca("calendario.pantalla.descarte.no_es_tuyo"))
	if nota < 0 {
		t.Fatalf("la nota se pide y no se pinta:\n%s", recorta(cuerpo, 900))
	}
	// DELANTE DEL BLOQUE, no dentro de una seccion y no detras. Un descargo que
	// llega despues de las listas se lee cuando ya se han leido, o sea tarde: es
	// la misma razon por la que el descargo de lo vencido va pegado al dato y no
	// en el pie.
	primera := -1
	for _, a := range anclasDelCorpusEntero {
		i := strings.Index(cuerpo, `id="`+a+`"`)
		if i < 0 {
			continue
		}
		if primera < 0 || i < primera {
			primera = i
		}
	}
	if primera < 0 {
		t.Fatal("no se pinta ninguna seccion del corpus entero, asi que este control " +
			"positivo no esta recorriendo nada")
	}
	// Y ESTA DELANTE DE LAS CUATRO, no solo de la primera: una nota entre la
	// segunda y la tercera cumpliria «va antes de alguna» y llegaria tarde para
	// las que ya se leyeron.
	for _, a := range anclasDelCorpusEntero {
		if i := strings.Index(cuerpo, `id="`+a+`"`); i >= 0 && nota > i {
			t.Errorf("la nota sale DESPUES de la seccion #%s, que cuenta el corpus entero. "+
				"Un descargo que llega cuando la lista ya se ha leido no descarga nada", a)
		}
	}

	// CONTROL NEGATIVO: sin bloque, sin nota.
	//
	// Se vacian las CUATRO listas del corpus entero y sus contadores a la vez.
	// Vaciar solo las listas dejaria las cifras en pie, y entonces esta rama
	// estaria midiendo un descuadre y no la ausencia del bloque.
	cal := calendarioConVencidas()
	cal.HitosQueEstrenan, cal.RelojesQueEstrenan = 0, nil
	cal.HitosYaCesados, cal.RelojesYaCesados = 0, nil
	cal.HitosQueEmpiezanDespues, cal.RelojesQueEmpiezanDespues = 0, nil
	cal.HitosConVigenciaIlegible, cal.RelojesConVigenciaIlegible = 0, nil
	// Y la particion por tiempo se ajusta, para que el unico cambio sea el que
	// se busca: 30 = 30 + 0 + 0 + 0 + 0.
	cal.HitosDelCorpus = cal.HitosEnVigor
	s2, _ := pantallaDePrueba(t, fuenteDoble{d: Derivado{
		Calendario: cal, Organizacion: "Acme SL"}, hay: true})
	_, sinBloque := pedir(t, s2, BasePorDefecto+"/")
	for _, a := range anclasDelCorpusEntero {
		if strings.Contains(sinBloque, `id="`+a+`"`) {
			t.Fatalf("la seccion #%s se sigue pintando, asi que este control negativo no "+
				"esta recorriendo la ausencia del bloque", a)
		}
	}
	if strings.Contains(sinBloque, "calendario.pantalla.descarte.no_es_tuyo") {
		t.Error("sin ni una seccion del corpus entero, la pagina sigue pintando la nota que " +
			"habla de ellas: una frase sobre una lista que no existe es ruido, y una nota " +
			"que sale siempre no demuestra nada en el control positivo")
	}
}
