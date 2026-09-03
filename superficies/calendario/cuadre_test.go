package calendario

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// LA PREGUNTA QUE FALTABA: ¿puede una cifra abrirse a una lista que NO cuadra
// con ella?
//
// # Por que es peor que no tener enlace
//
// Un numero sin enlace es un numero que hay que creerse, y eso ya es malo. Un
// numero CON enlace es una promesa: «pulsa y cuenta las filas». Si al contarlas
// salen otras, lo que se rompe no es una cifra, es la pantalla entera, porque
// quien lo vea ya no sabe cual de los catorce numeros mirar. D11-c existe para
// impedir el primer caso y esta puerta para impedir el segundo.
//
// # Lo que ya habia y por que no llegaba
//
// `TestCadaSeccionDeDescarteCuentaExactamenteSuCifra` compara `d.N` con
// `len(d.Filas)` en el MODELO, y solo para las secciones de descarte. Deja
// fuera las dos mitades donde vive el fallo:
//
//	las secciones que NO son descartes   los meses, los vencidos, los ceses y
//	                                     los relojes sin fecha se pintan con su
//	                                     propia rama de plantilla y ninguna
//	                                     puerta contrastaba su cifra
//	la PLANTILLA                         el modelo puede cuadrar y la plantilla
//	                                     pintar otra cosa: una fila por
//	                                     obligacion bajo una cabecera que cuenta
//	                                     hitos es exactamente eso
//
// Esta cuenta las filas EN LA RESPUESTA, que es lo que cuenta el lector.
//
// # Como nacio, y se dice
//
// Nacio VERDE sobre el dato sintetico que habia, y el verde era de suerte: la
// cifra `cesan` cuenta HITOS y su seccion pintaba una fila por OBLIGACION, pero
// el unico cese del dato tenia un solo hito, asi que 1 = 1. Se le puso al cese
// un segundo hito (que es lo que hace toda notificacion escalonada) y la puerta
// se puso roja diciendo «la cabecera de #ceses cuenta 3 y la seccion pinta 2
// filas». El corpus publicado NO alcanza esta rama: sus tres perfiles dan
// `cesan` = 0, asi que aqui el dato sintetico no es comodidad, es la unica
// entrada que existe.

// reFila recorta cada <li> de un trozo de pagina.
var reFila = regexp.MustCompile(`(?s)<li>.*?</li>`)

// recorteDeSeccion devuelve el trozo de la pagina que ocupa la seccion con ese
// ancla, del elemento que la abre a su cierre.
//
// Se para si no encuentra el ancla: sin el recorte, contar <li> en la pagina
// entera daria el mismo numero para todas las secciones y esta puerta estaria
// midiendo la pagina, no la seccion.
func recorteDeSeccion(t *testing.T, cuerpo, ancla string) string {
	t.Helper()
	marca := `id="` + ancla + `"`
	i := strings.Index(cuerpo, marca)
	if i < 0 {
		t.Fatalf("la pagina no trae ninguna seccion con id=%q, asi que la cifra que enlaza "+
			"ahi se abre a la nada", ancla)
	}
	// El nombre del elemento que abre, leido hacia atras hasta su '<'. Hace
	// falta para saber que cierre buscar: las secciones de descarte son
	// <section> y el envoltorio de los meses es un <div>, y cortar por el
	// cierre equivocado se lleva media pagina dentro.
	abre := strings.LastIndex(cuerpo[:i], "<")
	if abre < 0 {
		t.Fatalf("el ancla %q no cuelga de ningun elemento", ancla)
	}
	nombre := cuerpo[abre+1 : i]
	if j := strings.IndexAny(nombre, " \t\n"); j >= 0 {
		nombre = nombre[:j]
	}
	resto := cuerpo[abre:]
	cierre := "</" + nombre + ">"
	k := strings.Index(resto, cierre)
	if k < 0 {
		t.Fatalf("la seccion %q no se cierra con %s", ancla, cierre)
	}
	return resto[:k]
}

// TestCadaCifraQueSeAbreCuadraConLasFilasDeSuSeccion es la puerta.
func TestCadaCifraQueSeAbreCuadraConLasFilasDeSuSeccion(t *testing.T) {
	s, _ := pantallaDePrueba(t, fuenteDoble{d: Derivado{
		Calendario: calendarioConVencidas(), Organizacion: "Acme SL"}, hay: true})
	codigo, cuerpo := pedir(t, s, BasePorDefecto+"/")
	if codigo != http.StatusOK {
		t.Fatalf("GET %s/ ha respondido %d", BasePorDefecto, codigo)
	}
	var v Vista
	v.rellenarCon(Derivado{Calendario: calendarioConVencidas(), Organizacion: "Acme SL"})

	comprobadas := 0
	for _, c := range CifrasDeLaCuenta(v.Cuenta) {
		if c.Derivacion != CifraConSeccion || !c.SePinta() {
			continue
		}
		if c.Cuadre == CuadreSinDeclarar {
			t.Errorf("la cifra de CuentaVista.%s se abre en #%s y no dice COMO se contrasta "+
				"con su seccion. El valor cero no es un estado, es el olvido: sin forma de "+
				"cuadre, el enlace promete una comprobacion que nadie hace", c.Campo, c.Ancla)
			continue
		}
		comprobadas++
		seccion := recorteDeSeccion(t, cuerpo, c.Ancla)
		filas := len(reFila.FindAllString(seccion, -1))
		switch c.Cuadre {
		case CuadreFilas:
			if filas != c.N {
				t.Errorf(`la cabecera de #%s cuenta %d y la seccion pinta %d filas.

  Quien pulse esa cifra va a contar las filas y le van a salir %d. Un numero que
  no cuadra con su lista es PEOR que un numero sin enlace: el enlace prometia que
  se podia comprobar.
  La cifra es CuentaVista.%s.`, c.Ancla, c.N, filas, filas, c.Campo)
			}
		case CuadreCiclos:
			// El unico caso donde contar filas no es el contraste. Se suma lo
			// que la fila dice, y se comprueba ademas que la fila lo DIGA: un
			// ciclo que no se pinta no lo puede sumar el lector.
			suma := 0
			for _, x := range v.Vencidas {
				suma += x.Ciclos
			}
			if suma != c.N {
				t.Errorf("la cabecera de #%s cuenta %d ocurrencias y los ciclos de sus filas "+
					"suman %d", c.Ancla, c.N, suma)
			}
			if filas == 0 {
				t.Errorf("la seccion #%s no pinta ni una fila", c.Ancla)
			}
		}
	}
	// SUELO: sin el, un dato sintetico que dejara todas las cifras a cero daria
	// verde sin haber contrastado ninguna.
	if comprobadas < 9 {
		t.Fatalf("solo se han contrastado %d cifras y hoy se abren al menos 9: el dato "+
			"sintetico ha dejado alguna sin pintar y esta puerta no la esta mirando",
			comprobadas)
	}
}

// CONTROL NEGATIVO DEL RECORTE. Su forma de mentir en silencio es devolver la
// pagina entera, y entonces todas las secciones tendrian las mismas filas.
func TestElRecorteDeUnaSeccionNoSeLlevaLaPaginaEntera(t *testing.T) {
	s, _ := pantallaDePrueba(t, fuenteDoble{d: Derivado{
		Calendario: calendarioConVencidas(), Organizacion: "Acme SL"}, hay: true})
	_, cuerpo := pedir(t, s, BasePorDefecto+"/")

	ceses := recorteDeSeccion(t, cuerpo, AnclaCeses)
	if len(ceses) >= len(cuerpo) {
		t.Fatal("el recorte de una seccion devuelve la pagina entera: cualquier fila de " +
			"cualquier seccion contaria como fila suya")
	}
	// RAMA NEGATIVA: dentro no esta lo que solo hay fuera.
	if strings.Contains(ceses, "Revision anual del plan") {
		t.Error("el recorte de #ceses se ha llevado dentro una fila de los vencidos")
	}
	// RAMA POSITIVA: dentro esta lo suyo.
	if !strings.Contains(ceses, "Registro transitorio") {
		t.Error("el recorte de #ceses no trae su propia fila: esta midiendo menos pagina " +
			"de la que dice")
	}
}
