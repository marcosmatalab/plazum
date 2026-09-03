package calendario

import (
	"net/http"
	"strings"
	"testing"
)

// LA MITAD QUE LA PUERTA D11-c NO HACIA: el numero cuadra con la lista.
//
// Su cabecera lo decia en voz alta y era cierto: «no comprueba que el numero
// cuadre con las filas de la seccion que lo abre». No se podia, porque
// nucleo/pantalla guardaba los descartes como contadores. Ahora los retiene EN
// LA UNIDAD DE SU CONTADOR, asi que para las cinco secciones de descarte si se
// puede, y no comprobarlo seria dejar puesto el fallo que D11-c persigue con una
// capa de pintura encima: un enlace que lleva a una lista mas corta que el
// numero que se pulso es peor que ningun enlace.
//
// SIGUE SIN PODERSE para las otras cuatro, y por un motivo que no es deuda: el
// numero de vencidos cuenta OCURRENCIAS y su lista trae una fila por obligacion
// con sus ciclos al lado. Ahi el contraste no es contar filas.
func TestCadaSeccionDeDescarteCuentaExactamenteSuCifra(t *testing.T) {
	var v Vista
	v.rellenarCon(Derivado{Calendario: calendarioConVencidas(), Organizacion: "Acme SL"})

	// LAS DOS MITADES, y las dos se recorren: las secciones que hablan de TI y
	// las que cuentan el corpus entero salen de la misma llamada y tienen que
	// cuadrar igual. Mirar solo una de las dos era el hueco de esta puerta el dia
	// que la lista se partio en dos.
	todas := append(append([]DescarteVista(nil), v.Tuyas...), v.Descartes...)
	if len(todas) == 0 {
		t.Fatal("el dato sintetico no produce ni una seccion abierta: esta puerta " +
			"recorreria el vacio")
	}
	// Emparejamiento POR ANCLA, que es el campo que enlaza la cifra con la
	// seccion, y nunca por el orden de las dos listas: reordenar
	// CifrasDeLaCuenta no puede mover una seccion a otra cabecera.
	porAncla := map[string]CifraDeLaCuenta{}
	for _, c := range CifrasDeLaCuenta(v.Cuenta) {
		if c.Derivacion == CifraConSeccion {
			porAncla[c.Ancla] = c
		}
	}
	vistas := 0
	for _, d := range todas {
		c, hay := porAncla[d.Ancla]
		if !hay {
			t.Errorf("la seccion #%s no la abre ninguna cifra: es una lista a la que nadie "+
				"llega y que nadie ha pedido", d.Ancla)
			continue
		}
		vistas++
		if d.Clave != c.Clave {
			t.Errorf("la seccion #%s se rotula con %q y su cifra con %q: son dos frases y "+
				"se van a separar", d.Ancla, d.Clave, c.Clave)
		}
		if d.N != c.N {
			t.Errorf("la cabecera de #%s dice %d y su cifra dice %d", d.Ancla, d.N, c.N)
		}
		if len(d.Filas) != d.N {
			t.Errorf(`la seccion #%s se titula con %d y trae %d filas.

  Quien pulse esa cifra va a contar las filas, y le van a salir %d. Un numero que
  no cuadra con su lista hace que se deje de leer la pantalla entera, y con
  razon: es peor que no tener el enlace.`, d.Ancla, d.N, len(d.Filas), len(d.Filas))
		}
	}
	// SUELO: las siete secciones tienen que haberse recorrido. Sin esto, un dato
	// sintetico que dejara varias cifras a cero daria verde sin mirarlas.
	if vistas != 7 {
		t.Fatalf("se han recorrido %d secciones abiertas y hoy son 7 (3 tuyas y 4 del "+
			"corpus): el dato sintetico ha dejado alguna sin datos y esta puerta no la "+
			"esta mirando", vistas)
	}
	// LA FILA POR HITO, que es el caso que hace falta tener: una obligacion con
	// tres hitos aporta tres al numero. Si la lista fuera por obligacion, aqui
	// habria menos filas que numero y arriba ya habria saltado.
	for _, d := range todas {
		if d.Ancla != AnclaEmpiezanTarde {
			continue
		}
		if len(d.Filas) < 3 {
			t.Errorf("la seccion de los que empiezan tarde trae %d filas y el dato sintetico "+
				"pone tres hitos en dos obligaciones: sin ese caso, contar filas contra la "+
				"cifra da igual y esta puerta no vigila nada", len(d.Filas))
		}
	}
}

// CONTROL NEGATIVO: la puerta de arriba sabe decir que no.
//
// Se separa a mano una lista de su numero y se exige que el mismo contraste lo
// vea. Sin esto, el verde podria venir de un recorrido que no compara nada.
func TestElContrasteDeLasSeccionesDeDescarteCazaUnDescuadre(t *testing.T) {
	cal := calendarioConVencidas()
	// El mismo descuadre que produciria una lista por OBLIGACION bajo una
	// cabecera que cuenta HITOS: la obligacion escalonada aporta tres y solo se
	// pintaria una vez.
	cal.RelojesQueEmpiezanDespues[1].Hitos = cal.RelojesQueEmpiezanDespues[1].Hitos[:1]

	var v Vista
	v.rellenarCon(Derivado{Calendario: cal, Organizacion: "Acme SL"})

	todas := append(append([]DescarteVista(nil), v.Tuyas...), v.Descartes...)
	descuadres := 0
	for _, d := range todas {
		if len(d.Filas) != d.N {
			descuadres++
		}
	}
	if descuadres != 1 {
		t.Fatalf("quitado un hito de una obligacion que aporta tres a su cifra, el contraste "+
			"tenia que ver UN descuadre y vio %d", descuadres)
	}
	// Y LA MITAD POSITIVA, sin la cual esto no demuestra nada: el resto sigue
	// cuadrando, o sea que el detector no dice que si a todo.
	if len(todas) != 7 {
		t.Fatalf("%d secciones abiertas, esperaba 7", len(todas))
	}
}

// LAS SECCIONES SE PINTAN DE VERDAD, con su ancla y sus filas.
//
// La de arriba mira el MODELO. Nada de eso impide que la plantilla se olvide de
// pintarlas, que es el error que deja la declaracion tranquilizadora encima de
// una cifra igual de huerfana. Aqui se levanta la pantalla y se lee la respuesta.
func TestLasSeccionesDeDescarteSePintanConSuAncla(t *testing.T) {
	s, _ := pantallaDePrueba(t, fuenteDoble{d: Derivado{
		Calendario: calendarioConVencidas(), Organizacion: "Acme SL"}, hay: true})
	codigo, cuerpo := pedir(t, s, BasePorDefecto+"/")
	if codigo != http.StatusOK {
		t.Fatalf("GET %s/ ha respondido %d", BasePorDefecto, codigo)
	}
	for _, ancla := range []string{AnclaAlcanzados, AnclaEstrenan, AnclaYaCesados,
		AnclaEmpiezanTarde, AnclaIlegibles, AnclaMasAlla, AnclaAntesDeVigor} {
		if !strings.Contains(cuerpo, `id="`+ancla+`"`) {
			t.Errorf("la seccion #%s no se pinta: su cifra enlaza a la nada", ancla)
		}
	}
	// Y EL CONTENIDO DE VERDAD, no solo el ancla. Una seccion vacia con el ancla
	// puesta pasa el contraste de arriba y no ensena nada.
	for _, texto := range []string{
		"Notificacion en tres fases",   // alcanzados, la de tres hitos
		"Registro derogado",            // ya cesados
		"Notificacion escalonada",      // empiezan tarde, la de tres hitos
		"Punto sin vigencia legible",   // ilegibles
		"Revision decenal",             // mas alla
		"ese dia la norma no obligaba", // el descargo de las anteriores a la vigencia
	} {
		if !strings.Contains(cuerpo, texto) {
			t.Errorf("la pagina no trae %q, que es contenido de una seccion de descarte", texto)
		}
	}
}

// EL DESCARGO DE LA SECCION QUE ENSENA PASADO, con sus dos controles.
//
// De las cinco secciones nuevas solo una ensena FECHAS PASADAS al lado de una
// obligacion, y por tanto solo una se puede leer como una acusacion: las
// ocurrencias anteriores a la entrada en vigor de su norma. Quien reviso su
// politica en 2022 no incumplia en 2023 un reglamento en vigor desde 2024, y
// pintar «vencio el 2023-01-15» sin decirlo es acusar en falso.
//
// El descargo va en DOS sitios y los dos se comprueban: en la cabecera, que es
// el rotulo de la cifra («que NO son incumplimientos»), y en la derivacion de
// cada fila, que dice con que dato se decidio.
func TestLaSeccionDeLoAnteriorALaVigenciaNoAcusa(t *testing.T) {
	s, _ := pantallaDePrueba(t, fuenteDoble{d: Derivado{
		Calendario: calendarioConVencidas(), Organizacion: "Acme SL"}, hay: true})
	_, cuerpo := pedir(t, s, BasePorDefecto+"/")

	// CONTROL POSITIVO: con una fila dentro, la cabecera y el descargo salen.
	//
	// SE RECORTA LA SECCION primero. Buscar la clave en la pagina entera la
	// encontraria en el pie, donde tambien se pinta, y entonces esta puerta
	// daria verde con la cabecera de la seccion en blanco: el descargo estaria
	// abajo y las fechas pasadas arriba, sin nada al lado.
	seccion := recorteDe(t, cuerpo, AnclaAntesDeVigor)
	if !strings.Contains(seccion, "[[calendario.pantalla.cuenta.antes_de_vigor[") {
		t.Errorf("la seccion de lo anterior a la vigencia se pinta sin su rotulo, que es "+
			"donde va escrito que NO son incumplimientos:\n%s", recorta(seccion, 600))
	}
	if !strings.Contains(seccion, "ese dia la norma no obligaba") {
		t.Error("la fila anterior a la vigencia no dice por que no es un incumplimiento: " +
			"una fecha pasada al lado de una obligacion, sin esa frase, es una acusacion")
	}

	// CONTROL NEGATIVO: sin ninguna fila, la seccion no existe y el descargo
	// tampoco se pide. Un descargo pintado sin nada que descargar es ruido, y
	// ademas dejaria esta puerta verde sin haber recorrido la rama de verdad.
	cal := calendarioConVencidas()
	cal.VencimientosAntesDeLaVigencia = 0
	cal.VencimientosAnterioresALaVigencia = nil
	s2, _ := pantallaDePrueba(t, fuenteDoble{d: Derivado{
		Calendario: cal, Organizacion: "Acme SL"}, hay: true})
	_, sinEllas := pedir(t, s2, BasePorDefecto+"/")
	if strings.Contains(sinEllas, `id="`+AnclaAntesDeVigor+`"`) {
		t.Error("sin ni una ocurrencia anterior a la vigencia, la seccion no tiene que " +
			"existir: una seccion vacia con ese titulo insinua algo que no ha pasado")
	}
}

// recorteDe recorta una seccion de descarte de la pagina, por su ancla.
//
// Se para si no la encuentra. Sin el recorte, buscar un rotulo en la pagina
// entera lo encontraria en el pie de la cuenta, que pinta la MISMA clave, y la
// puerta daria verde mirando a otro sitio: es el mismo fallo que
// seccionDeLaCuenta documenta para su propio recorte.
func recorteDe(t *testing.T, cuerpo, ancla string) string {
	t.Helper()
	i := strings.Index(cuerpo, `id="`+ancla+`"`)
	if i < 0 {
		t.Fatalf("la pagina no trae la seccion #%s, asi que esta puerta no esta mirando "+
			"ninguna fila:\n%s", ancla, recorta(cuerpo, 900))
	}
	resto := cuerpo[i:]
	j := strings.Index(resto, "</section>")
	if j < 0 {
		t.Fatalf("la seccion #%s no se cierra", ancla)
	}
	return resto[:j]
}

// CONTROL NEGATIVO DEL RECORTE. Su forma de mentir en silencio es devolver la
// pagina entera: entonces el rotulo del pie contaria como el de la seccion.
func TestElRecorteDeUnaSeccionDeDescarteNoSeLlevaLaPaginaEntera(t *testing.T) {
	s, _ := pantallaDePrueba(t, fuenteDoble{d: Derivado{
		Calendario: calendarioConVencidas(), Organizacion: "Acme SL"}, hay: true})
	_, cuerpo := pedir(t, s, BasePorDefecto+"/")
	seccion := recorteDe(t, cuerpo, AnclaAntesDeVigor)

	if len(seccion) >= len(cuerpo) {
		t.Fatal("el recorte devuelve la pagina entera: cualquier rotulo de cualquier " +
			"seccion contaria como el de esta")
	}
	// RAMA NEGATIVA: dentro NO esta lo que solo hay fuera.
	for _, fuera := range []string{
		"Registro derogado",                        // otra seccion de descarte
		marca("calendario.pantalla.titulo"),        // el <h1>
		marca("calendario.pantalla.cuenta.titulo"), // el pie
	} {
		if strings.Contains(seccion, fuera) {
			t.Errorf("el recorte se ha llevado dentro %q, que vive fuera: esta midiendo mas "+
				"pagina de la que dice", fuera)
		}
	}
	// RAMA POSITIVA: dentro esta lo que solo hay dentro.
	if !strings.Contains(seccion, "ese dia la norma no obligaba") {
		t.Error("el recorte no trae ni la derivacion de su unica fila: esta midiendo menos " +
			"pagina de la que dice")
	}
}
