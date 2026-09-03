package calendario

import (
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/superficies/camino"
)

// LA PUERTA D11-c EN EL CALENDARIO.
//
// La puerta existia para el panel de inicio y NO existia aqui, y esta pantalla
// pinta CATORCE numeros al pie. El porque, con el cardinal de lo que se abre y
// de lo que no, esta en cuenta.go, al lado de la declaracion.
//
// LO QUE ESTA PUERTA NO HACE, dicho para que no se le suponga: no comprueba que
// el numero cuadre con las filas de la seccion que lo abre. Ese contraste lo
// hace el panel de inicio, donde numero y lista estan en la misma unidad; aqui
// no se puede en general, porque el numero de vencidos cuenta OCURRENCIAS y la
// lista trae una fila por obligacion con sus ciclos al lado. Lo que si se
// comprueba es que el destino EXISTE y no esta vacio, que es lo que distingue un
// enlace de un enlace roto.

// LOS CAMPOS DE LA CUENTA SE ENUMERAN DEL TIPO, no de una lista escrita aqui.
//
// Es la mitad que nadie puede olvidarse de actualizar: quien anada un campo a
// CuentaVista pasa por el compilador, no por este fichero.
func camposDeCuentaVista() []string {
	tipo := reflect.TypeOf(CuentaVista{})
	out := make([]string, 0, tipo.NumField())
	for i := 0; i < tipo.NumField(); i++ {
		out = append(out, tipo.Field(i).Name)
	}
	sort.Strings(out)
	return out
}

// TestCadaCampoDeLaCuentaDeclaraSuDerivacion cruza el tipo con la declaracion en
// los DOS sentidos.
//
// El sentido que falta es siempre el que se usa: sin el primero, un campo nuevo
// se suma en la contabilidad y no se pinta; sin el segundo, la declaracion sigue
// pintando una cifra de un campo que ya no existe.
func TestCadaCampoDeLaCuentaDeclaraSuDerivacion(t *testing.T) {
	campos := camposDeCuentaVista()
	// El suelo protege del verde por vacio: si la reflexion deja de casar, todo
	// lo de abajo recorreria la nada.
	if len(campos) < 14 {
		t.Fatalf("CuentaVista trae %d campos (%v) y hoy son al menos 14. O la contabilidad "+
			"se ha encogido, o esta puerta esta midiendo el vacio", len(campos), campos)
	}
	declarados := CamposDeclaradosDeLaCuenta()

	// SENTIDO 1: todo campo del tipo esta declarado.
	enLaLista := map[string]bool{}
	for _, c := range declarados {
		enLaLista[c] = true
	}
	for _, campo := range campos {
		if !enLaLista[campo] {
			t.Errorf("CuentaVista.%s no sale en CifrasDeLaCuenta.\n"+
				"  La contabilidad lo suma y la pantalla no lo pinta: es un numero que el "+
				"producto calcula y esconde, que es peor que no calcularlo.\n"+
				"  Arreglo: anadirlo en cuenta.go, con su clave de catalogo y su derivacion.",
				campo)
		}
	}

	// SENTIDO 2: toda cifra declarada corresponde a un campo del tipo.
	delTipo := map[string]bool{}
	for _, c := range campos {
		delTipo[c] = true
	}
	for _, campo := range declarados {
		if !delTipo[campo] {
			t.Errorf("CifrasDeLaCuenta declara el campo %q y CuentaVista no lo tiene. La "+
				"declaracion se ha quedado vieja y esta pintando una cifra fantasma", campo)
		}
	}

	// SENTIDO 3: ni valor cero, ni declaracion a medias.
	sinDerivacion := 0
	for _, c := range CifrasDeLaCuenta(CuentaVista{}) {
		switch c.Derivacion {
		case CifraSinDeclarar:
			t.Errorf("la cifra de CuentaVista.%s esta con el VALOR CERO (%s). El cero no es "+
				"un estado, es el olvido: o se abre en una seccion, o se dice por que no",
				c.Campo, c.Derivacion)
		case CifraConSeccion:
			if strings.TrimSpace(c.Ancla) == "" {
				t.Errorf("la cifra de CuentaVista.%s dice abrirse en una seccion y no dice en "+
					"cual: el enlace no lleva a ninguna parte", c.Campo)
			}
			if c.Motivo != "" {
				t.Errorf("la cifra de CuentaVista.%s se abre y ademas trae Motivo (%q). Ese "+
					"campo es el de las que NO se abren", c.Campo, c.Motivo)
			}
		case CifraSinDerivacion:
			sinDerivacion++
			if strings.TrimSpace(c.Motivo) == "" {
				t.Errorf("la cifra de CuentaVista.%s no se puede abrir y no dice por que.\n"+
					"  Un hueco sin motivo se lee como deuda y como decision a la vez, y la "+
					"lectura barata es la que deja el numero huerfano para siempre", c.Campo)
			}
			if c.Ancla != "" {
				t.Errorf("la cifra de CuentaVista.%s dice no poder abrirse y trae ancla %q. O "+
					"se abre o no se abre", c.Campo, c.Ancla)
			}
		}
		if strings.TrimSpace(c.Clave) == "" {
			t.Errorf("la cifra de CuentaVista.%s no tiene clave de catalogo: saldria como un "+
				"numero suelto sin decir que cuenta", c.Campo)
		}
	}

	// EL HUECO ESTA TOPADO, CON IGUALDAD EXACTA Y EN LOS DOS SENTIDOS.
	//
	// Con un `>` solo, el hueco podria crecer sin que nadie se enterara; con un
	// `<` solo, cerrarlo no obligaria a bajar el numero y el cardinal se
	// quedaria mintiendo hacia arriba. Es la misma forma que PUERTAS_ESPERADAS.
	if sinDerivacion != SinDerivacionEsperadas {
		t.Errorf("hay %d cifras sin derivacion y SinDerivacionEsperadas dice %d.\n"+
			"  Si han subido, se ha anadido una cifra huerfana y D11-c dice que ninguna "+
			"queda huerfana en silencio.\n"+
			"  Si han bajado, alguien ha abierto una cifra y no ha bajado el numero: el "+
			"cardinal es lo unico que hace que este hueco moleste hasta que se cierre.",
			sinDerivacion, SinDerivacionEsperadas)
	}
}

// reLineaDeCuenta recorta cada <li> de la seccion de la cuenta.
var reLineaDeCuenta = regexp.MustCompile(`(?s)<li>.*?</li>`)

// seccionDeLaCuenta recorta el bloque de la cuenta de la pagina.
//
// Se para si no lo encuentra: sin el recorte, buscar los enlaces en la pagina
// entera los encontraria en cualquier otra seccion y la puerta daria verde
// mirando a otro sitio.
func seccionDeLaCuenta(t *testing.T, cuerpo string) string {
	t.Helper()
	i := strings.Index(cuerpo, `<section class="cuenta">`)
	if i < 0 {
		t.Fatalf("la pagina no trae la seccion de la cuenta, asi que esta puerta no esta "+
			"mirando ninguna cifra:\n%s", recorta(cuerpo, 900))
	}
	resto := cuerpo[i:]
	j := strings.Index(resto, "</section>")
	if j < 0 {
		t.Fatal("la seccion de la cuenta no se cierra")
	}
	return resto[:j]
}

// TestNingunaCifraDelCalendarioQueSePuedeAbrirSeQuedaSinEnlace es la puerta
// sobre la RESPUESTA, no sobre la declaracion.
//
// La de arriba lee texto y comprueba que lo escrito es coherente consigo mismo.
// Nada de eso impide declarar un ancla que la plantilla no pinta, ni pintar un
// enlace a una seccion que no existe, que son los dos errores que dejan la cifra
// igual de huerfana con una declaracion tranquilizadora encima.
func TestNingunaCifraDelCalendarioQueSePuedeAbrirSeQuedaSinEnlace(t *testing.T) {
	s, _ := pantallaDePrueba(t, fuenteDoble{d: Derivado{
		Calendario: calendarioConVencidas(), Organizacion: "Acme SL"}, hay: true})
	codigo, cuerpo := pedir(t, s, BasePorDefecto+"/")
	if codigo != http.StatusOK {
		t.Fatalf("GET %s/ ha respondido %d", BasePorDefecto, codigo)
	}
	seccion := seccionDeLaCuenta(t, cuerpo)
	lineas := reLineaDeCuenta.FindAllString(seccion, -1)
	if len(lineas) == 0 {
		t.Fatal("la seccion de la cuenta no trae ni una linea: el dato sintetico ha dejado " +
			"de producir cifras y esta puerta recorre el vacio")
	}

	conEnlace := 0
	for _, c := range CifrasDeLaCuenta(vistaDePrueba(t, cuerpo)) {
		if c.Derivacion != CifraConSeccion {
			continue
		}
		if !c.SePinta() {
			continue
		}
		conEnlace++
		enlace := `href="#` + c.Ancla + `"`
		if !strings.Contains(seccion, enlace) {
			t.Errorf("la cifra de CuentaVista.%s se declara abrible en #%s y la seccion de "+
				"la cuenta NO pinta su enlace.\n"+
				"  Un numero que no se puede abrir es un numero que hay que creerse, y este "+
				"producto se vende exactamente al reves.\n--- la cuenta ---\n%s",
				c.Campo, c.Ancla, recorta(seccion, 900))
			continue
		}
		// Y EL DESTINO EXISTE. Un enlace a un ancla que no esta en ninguna
		// parte lleva al mismo sitio que ninguno, y ademas parece que lleva.
		if !strings.Contains(cuerpo, `id="`+c.Ancla+`"`) {
			t.Errorf("la cifra de CuentaVista.%s enlaza a #%s y esa seccion no existe en la "+
				"pagina: el numero se abre a la nada", c.Campo, c.Ancla)
		}
	}
	// Las cuatro que hoy se abren tienen que haberse recorrido de verdad.
	esperadas := len(CifrasDeLaCuenta(CuentaVista{})) - SinDerivacionEsperadas
	if conEnlace != esperadas {
		t.Fatalf("se han comprobado %d cifras abribles y hoy son %d. El dato sintetico ya no "+
			"las recorre todas, asi que esta puerta esta dejando alguna sin mirar",
			conEnlace, esperadas)
	}
}

// vistaDePrueba reconstruye la cuenta con la que se pinto la pagina, para poder
// preguntarle que cifras se pintaron.
//
// Se compone del MISMO calendario sintetico que la peticion, no de una copia
// escrita aqui: dos datos distintos harian que la puerta comparase una pagina
// con una cuenta que no es la suya.
func vistaDePrueba(t *testing.T, _ string) CuentaVista {
	t.Helper()
	var v Vista
	v.rellenarCon(Derivado{Calendario: calendarioConVencidas(), Organizacion: "Acme SL"})
	return v.Cuenta
}

// CONTROL NEGATIVO DEL RECORTE DE LA CUENTA.
//
// Todo cuelga de seccionDeLaCuenta, y su forma de mentir en silencio es
// devolver la pagina entera: entonces cualquier enlace de cualquier seccion
// contaria como el enlace de la cifra, y la puerta daria verde sobre una cuenta
// sin un solo enlace. Se comprueba que el recorte DEJA FUERA lo que tiene que
// dejar fuera.
func TestElRecorteDeLaCuentaNoSeLlevaLaPaginaEntera(t *testing.T) {
	s, _ := pantallaDePrueba(t, fuenteDoble{d: Derivado{
		Calendario: calendarioConVencidas(), Organizacion: "Acme SL"}, hay: true})
	_, cuerpo := pedir(t, s, BasePorDefecto+"/")
	seccion := seccionDeLaCuenta(t, cuerpo)

	if len(seccion) >= len(cuerpo) {
		t.Fatal("el recorte de la cuenta devuelve la pagina entera: cualquier enlace de " +
			"cualquier seccion contaria como el enlace de una cifra")
	}
	// RAMA NEGATIVA: dentro del recorte NO esta lo que solo hay fuera.
	for _, fuera := range []string{
		"Revision anual del plan",           // la fila de lo vencido
		marca("calendario.pantalla.titulo"), // el <h1>
		`href="` + camino.BasePorDefecto,    // la vuelta al camino
	} {
		if strings.Contains(seccion, fuera) {
			t.Errorf("el recorte de la cuenta se ha llevado dentro %q, que vive fuera de "+
				"ella: esta midiendo mas pagina de la que dice", fuera)
		}
	}
	// RAMA POSITIVA: dentro esta lo que solo hay dentro.
	if !strings.Contains(seccion, marca("calendario.pantalla.cuenta.titulo")) {
		t.Error("el recorte de la cuenta no trae ni su propio titulo: esta midiendo menos " +
			"pagina de la que dice")
	}
}
