package pantallas

import (
	"os"
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

// reglasDe saca los bloques cuyo selector empieza por el prefijo dado.
func reglasDe(t *testing.T, hoja, prefijo string) map[string]string {
	t.Helper()
	out := map[string]string{}
	re := regexp.MustCompile(`(?m)^([^{}/@][^{}]*)\{([^{}]*)\}`)
	for _, m := range re.FindAllStringSubmatch(hoja, -1) {
		sel := strings.TrimSpace(m[1])
		if strings.HasPrefix(sel, prefijo) {
			out[sel] = m[2]
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
	for sel, cuerpo := range pastillas {
		for _, tok := range tokensDeAlarma {
			if strings.Contains(cuerpo, "var("+tok+")") {
				t.Errorf("la pastilla %q se pinta con %s. Una pastilla de estado dice lo que "+
					"CONSTA, y varias de ellas dicen que todavia no consta nada: el color de "+
					"alarma acusa de un incumplimiento donde solo hay un dato que falta, y lo "+
					"hace antes de que nadie lea la palabra que va dentro", sel, tok)
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
