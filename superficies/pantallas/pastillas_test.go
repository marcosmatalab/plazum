package pantallas

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// NINGUNA PASTILLA DE ESTADO SE PINTA CON EL COLOR DE ALARMA.
//
// # Que regla es esta, y por que vive en la hoja de estilo
//
// La regla de la casa dice que en toda pantalla que ensena pasado, lo no
// constatado se presenta como DATO QUE FALTA y nunca como culpa, y que el
// descargo va con el dato. Eso ya lo vigilan varios tests, y todos miran
// PALABRAS: que la frase este, que este pegada al numero, que no salga cuando no
// hay nada que descargar.
//
// Falta la otra mitad, que es por donde se cuela la acusacion sin escribirla:
// EL COLOR. Una pastilla que dice "sin revisar" en rojo de alarma acusa igual
// que la frase que el texto se cuida de no escribir, y encima acusa antes,
// porque el color se lee antes que la palabra. Ninguna puerta de este
// repositorio miraba eso: axe comprueba que el par CONTRASTA, no que signifique
// lo que se quiere decir, y el test de contraste mide numeros y no semantica.
//
// La ocasion concreta: la revision de accesos pinta ahora el estado de cada fila
// como pastilla, y dos de sus cuatro estados son "todavia nadie lo ha mirado".
// Ese es exactamente el caso del hallazgo de la cadencia anclada antes de la
// norma: no consta que se hiciera, y no consta no es lo mismo que no se hizo.
//
// # Como se comprueba
//
// Se leen las reglas de .estado.* de la hoja y se exige que ninguna nombre el
// token de alarma. La alarma sigue existiendo y se sigue usando donde SI hay
// algo roto (el veredicto del planificador, los avisos): eso es el control
// positivo del detector, y va comprobado aqui mismo, porque un detector que no
// encuentra la alarma en ningun sitio daria verde sobre una hoja sin colores.

// tokensDeAlarma son los que significan "algo esta mal", no "falta un dato".
var tokensDeAlarma = []string{"--alerta", "--alerta-fondo"}

func hojaDeEstilo(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("estatico/plazum.css")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// regla es un bloque de la hoja: su selector y su cuerpo.
type regla struct{ sel, cuerpo string }

// reglasDe saca los bloques cuyo selector empieza por el prefijo dado.
//
// DEVUELVE UNA LISTA Y NO UN MAPA, y esto costo un rojo: el mismo selector sale
// mas de una vez en la hoja a proposito (thead th se declara para la pantalla y
// otra vez dentro de @media print, donde deja de estar pegado). Con un mapa, la
// segunda aparicion se comia a la primera y el test afirmaba lo contrario de lo
// que pasa en pantalla. Es la cascada de CSS: aqui no hay una regla por
// selector, hay todas.
func reglasDe(t *testing.T, hoja, prefijo string) []regla {
	t.Helper()
	var out []regla
	re := regexp.MustCompile(`(?m)^([^{}/@][^{}]*)\{([^{}]*)\}`)
	for _, m := range re.FindAllStringSubmatch(hoja, -1) {
		sel := strings.TrimSpace(m[1])
		if strings.HasPrefix(sel, prefijo) {
			out = append(out, regla{sel: sel, cuerpo: m[2]})
		}
	}
	return out
}

func TestNingunaPastillaDeEstadoSePintaConElColorDeAlarma(t *testing.T) {
	hoja := hojaDeEstilo(t)

	// CONTROL POSITIVO DEL DETECTOR, primero: la alarma existe en la hoja y se
	// usa en algun sitio que NO es una pastilla. Sin esto, una hoja que hubiera
	// perdido el token entero daria verde y este test no vigilaria nada.
	for _, tok := range tokensDeAlarma {
		if !strings.Contains(hoja, tok+":") {
			t.Fatalf("la hoja no declara %s: este detector estaria buscando un color que "+
				"no existe y daria verde siempre", tok)
		}
		if strings.Count(hoja, "var("+tok+")") == 0 {
			t.Fatalf("nadie usa %s en la hoja: o la alarma ha desaparecido, o el detector "+
				"ha dejado de reconocer como se usa un token", tok)
		}
	}

	pastillas := reglasDe(t, hoja, ".estado")
	if len(pastillas) < 3 {
		t.Fatalf("se han encontrado %d reglas de pastilla y hay mas: el detector esta "+
			"mirando otra cosa", len(pastillas))
	}
	for _, r := range pastillas {
		for _, tok := range tokensDeAlarma {
			if strings.Contains(r.cuerpo, "var("+tok+")") {
				t.Errorf("la pastilla %q se pinta con %s. Una pastilla de estado dice lo que "+
					"CONSTA, y varias de ellas dicen que todavia no consta nada: el color de "+
					"alarma acusa de un incumplimiento donde solo hay un dato que falta, y lo "+
					"hace antes de que nadie lea la palabra que va dentro", r.sel, tok)
			}
		}
	}
}

// Y TODA PASTILLA LLEVA SU ROTULO ESCRITO DENTRO, en las plantillas que las usan.
//
// El color acompana; quien manda es la palabra. Un estado que solo se distingue
// por el fondo deja fuera a quien no distingue los colores y a quien imprime en
// blanco y negro, que en esta pantalla es media revision de accesos.
//
// Se miran las plantillas de las superficies que pintan pastillas, y las dos
// direcciones: que la clase este y que dentro del elemento haya una llamada al
// catalogo. Un `<span class="estado e-x"></span>` vacio pasaria el primer
// control y no diria nada.
func TestTodaPastillaDeEstadoLlevaSuRotuloEscritoDentro(t *testing.T) {
	ficheros := []string{
		"plantillas/tabla.html",
		"../uar/plantillas/uar.html",
	}
	re := regexp.MustCompile(`(?s)<span class="estado[^"]*"[^>]*>(.*?)</span>`)
	vistas := 0
	for _, f := range ficheros {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("no puedo leer %s: %v. Si la plantilla ha cambiado de sitio, esta "+
				"puerta ha dejado de mirarla", f, err)
		}
		ms := re.FindAllStringSubmatch(string(b), -1)
		if len(ms) == 0 {
			t.Errorf("%s no pinta ninguna pastilla de estado: o ha dejado de hacerlo, o el "+
				"detector no la reconoce y este test mide el vacio", f)
			continue
		}
		for _, m := range ms {
			vistas++
			dentro := strings.TrimSpace(m[1])
			if !strings.Contains(dentro, "{{t ") && !strings.Contains(dentro, "{{t.") {
				t.Errorf("en %s hay una pastilla sin rotulo del catalogo dentro: %q. El "+
					"color no puede ser la unica senal del estado", f, dentro)
			}
		}
	}
	if vistas < 2 {
		t.Fatalf("solo se han mirado %d pastillas: el detector no esta recorriendo las "+
			"plantillas que las pintan", vistas)
	}
}

// TODA TABLA VA DENTRO DE SU MARCO, y el marco es el que se desplaza.
//
// # Por que es una regla y no una preferencia
//
// Una tabla suelta hace dos cosas malas a la vez, y las dos se vieron en una
// captura antes de tener esta puerta. Empuja el ancho del CUERPO de la pagina,
// asi que el desplazamiento horizontal se lo come todo (barra lateral incluida)
// en vez de quedarse dentro de la tabla; y deja la cabecera arriba del documento
// en vez de pegada a la tabla, asi que a la fila cuarenta ya no se sabe que
// columna es cual. En la revision de accesos eso es la diferencia entre revisar
// y firmar a ciegas, porque una campana de verdad trae cientos de filas.
//
// .marco-tabla es lo que arregla las dos: overflow:auto y max-height, y el
// thead pegado dentro. Y en papel se apaga entero, que es lo que hace que el
// board pack salga con la tabla completa.
//
// # Lo que se mira
//
// Los ficheros de plantilla de las tres superficies que pintan tablas. Se busca
// cada <table> y se exige que lo mas cercano por encima sea el marco. Es un
// detector por texto y no un parser de HTML a proposito: lo que puede
// estropearse aqui es que alguien anada una tabla nueva y se olvide del marco, y
// eso se ve mirando el orden de las etiquetas.
func TestTodaTablaVaDentroDeSuMarcoDesplazable(t *testing.T) {
	ficheros := []string{
		"plantillas/tabla.html",
		"../acta/plantillas/acta.html",
		"../uar/plantillas/uar.html",
	}
	tablas := 0
	for _, f := range ficheros {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("no puedo leer %s: %v. Si la plantilla ha cambiado de sitio, esta "+
				"puerta ha dejado de mirarla", f, err)
		}
		texto := string(b)
		desde := 0
		vistasAqui := 0
		for {
			i := strings.Index(texto[desde:], "<table")
			if i < 0 {
				break
			}
			i += desde
			desde = i + 6
			tablas++
			vistasAqui++
			// Lo mas cercano por encima tiene que ser la apertura del marco, no
			// su cierre: si el ultimo marco que hay antes ya se cerro, esta
			// tabla esta fuera.
			antes := texto[:i]
			abre := strings.LastIndex(antes, `class="marco-tabla"`)
			cierra := strings.LastIndex(antes, "</table>")
			if abre < 0 || abre < cierra {
				t.Errorf("%s: la tabla que empieza en el byte %d no esta dentro de un "+
					"marco-tabla. Sin el, la tabla empuja el ancho del cuerpo entero y la "+
					"cabecera deja de verse al bajar", filepath.Base(f), i)
			}
		}
		if vistasAqui == 0 {
			t.Errorf("%s no pinta ninguna tabla: o ha dejado de hacerlo, o el detector no "+
				"la reconoce y este test mide el vacio", f)
		}
	}
	if tablas < 3 {
		t.Fatalf("solo se han mirado %d tablas: el detector no esta recorriendo las "+
			"plantillas que las pintan", tablas)
	}
}

// Y EL MARCO SIGUE SIENDO EL QUE SE DESPLAZA, no la pagina.
//
// El control positivo del test de arriba es que encuentra tablas; el de este es
// que la clase existe en la hoja con las dos propiedades que la hacen util. Un
// marcado que envuelve todas las tablas en una clase que no hace nada pasaria el
// primero y no arreglaria nada.
func TestElMarcoDeLaTablaSeDesplazaElSoloYFijaSuCabecera(t *testing.T) {
	hoja := hojaDeEstilo(t)
	for _, exigido := range []string{
		".marco-tabla { overflow: auto;",
		".marco-tabla > table { min-width: max-content; }",
		"position: sticky;",
	} {
		if !strings.Contains(hoja, exigido) {
			t.Errorf("la hoja no trae %q. El marco sin desplazamiento propio no protege al "+
				"cuerpo de la pagina, y sin ancho natural la tabla se aprieta y parte las "+
				"palabras letra a letra", exigido)
		}
	}
	// La cabecera pegada tiene que ser la del thead y no otra cosa que use
	// sticky en la hoja, asi que se mira dentro de su regla.
	reglas := reglasDe(t, hoja, "thead th")
	if len(reglas) == 0 {
		t.Fatal("no hay ninguna regla de thead th en la hoja: el detector mide el vacio")
	}
	// Basta con que UNA de las apariciones la deje pegada: la de @media print la
	// suelta a proposito, porque en papel no hay nada que desplazar y una
	// cabecera pegada se repetiria encima del texto.
	pegada := false
	for _, r := range reglas {
		if strings.Contains(r.cuerpo, "position: sticky") {
			pegada = true
		}
	}
	if !pegada {
		t.Error("la cabecera de la tabla no se queda pegada al bajar: a la fila cuarenta " +
			"ya no se sabe que columna es cual")
	}
}
