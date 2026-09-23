package comprobaciones

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// claveDeLaNotaCasual es lo que el descargo tiene que nombrar. Se nombra un
// TEST y no una idea porque un nombre de test se puede ir a buscar, y
// `TestTodoTestQueCitaUnDocumentoExiste` rompe si deja de existir.
const claveDeLaNotaCasual = "TestLasDosCifrasQueCoincidenPorCasualidadLoDicen"

var (
	reCasillasDelPlan   = regexp.MustCompile(`\*\*(\d+) de (\d+) casillas\*\*`)
	reCoberturaEstricta = regexp.MustCompile(
		`(\d+) relojes \*\*cuyo intervalo lo escribe la norma\*\*, sobre (\d+) puntos`)
)

// DOS CIFRAS CON LA MISMA FORMA, EN DOS DOCUMENTOS QUE SE LEEN JUNTOS, TIENEN
// QUE DECIR QUE NO MIDEN LO MISMO.
//
// # El riesgo, que es de lectura y no de calculo
//
// `docs/ETAPAS.md` publica «N de M casillas» y `docs/cobertura-v1.md` publica «N
// relojes sobre M puntos». No tienen ninguna relacion:
//
//	el del plan       casillas del plan, derivadas del arbol de ese fichero
//	                  por estado_del_plan_test.go
//	el de cobertura   relojes con intervalo de la norma sobre puntos censados,
//	                  y su denominador sale de sumar las siete filas con
//	                  denominador de paquetes/marcos-v1.json
//
// Las dos estan bien computadas y cada una tiene su puerta. Lo que no tiene
// puerta es la LECTURA: quien vea dos fracciones de la misma forma en dos
// documentos que se leen juntos va a querer cuadrarlas, y cuadrarlas corrompe
// una en silencio.
//
// # POR QUE LA CONDICION SE QUITO, que es lo que esta puerta aprendio
//
// Hasta el 22-09-2026 el descargo solo se exigia MIENTRAS LAS CIFRAS FUERAN
// IGUALES: si dejaban de coincidir, esto registraba una linea y **volvia sin
// afirmar nada**. El razonamiento escrito era que «si dejan de parecerse, la
// confusion se deshace sola».
//
// Es falso, y se vio al ir a marcar dos casillas. El plan pasaba de 78/144 a
// 79/145 y la cobertura se quedaba en 78/144: **mas confundibles, no menos**,
// porque una diferencia de uno se lee como un descuadre que alguien deberia
// arreglar. O sea que la puerta se apagaba sola EXACTAMENTE en el momento de
// maximo riesgo, y ademas dejaba vieja la prosa de los dos documentos, que
// seguia diciendo «hoy coinciden».
//
// Lo que hace confundibles a estas dos cifras no es su VALOR, es su FORMA y el
// sitio donde se leen. Eso no cambia mientras las dos existan, asi que el
// descargo se exige siempre.
//
// # Y no se convierte en una puerta que salta sobre trabajo legitimo
//
// Porque no pide mantener un numero: pide mantener una frase. Y el dia que uno
// de los dos documentos deje de publicar su cifra, el lector de arriba no casa y
// esto PARA en rojo, asi que quien la quite se encuentra la nota en el mismo
// sitio y decide a la vez.
func TestLasDosCifrasQueCoincidenPorCasualidadLoDicen(t *testing.T) {
	etapas := leerFichero(t, rutaDeEtapas)
	cobertura := leerFichero(t, rutaDeCoberturaV1)

	// LAS CIFRAS SE LEEN DE LOS DOCUMENTOS, no se escriben aqui: escribirlas
	// seria una tercera copia, y entonces la que manda seria otra.
	mc := reCasillasDelPlan.FindStringSubmatch(etapas)
	if mc == nil {
		t.Fatalf("%s ya no publica «N de M casillas» con esa forma. Si cambio de "+
			"redaccion, este lector se quedo viejo y hay que arreglarlo: sin el, la puerta "+
			"pasaria sin comparar nada", rutaDeEtapas)
	}
	mr := reCoberturaEstricta.FindStringSubmatch(cobertura)
	if mr == nil {
		t.Fatalf("%s ya no publica la cobertura con esa forma. Mismo caso: un lector que "+
			"no encuentra nada no puede dar verde", rutaDeCoberturaV1)
	}

	casNum, casDen := enteroDeCifra(t, mc[1]), enteroDeCifra(t, mc[2])
	cobNum, cobDen := enteroDeCifra(t, mr[1]), enteroDeCifra(t, mr[2])

	faltan := notasQueFaltan(t, []documentoConCifra{
		{rutaDeEtapas, etapas, casNum, casDen},
		{rutaDeCoberturaV1, cobertura, cobNum, cobDen},
	})

	switch {
	case casNum == cobNum && casDen == cobDen:
		t.Logf("casillas del plan %d de %d y cobertura %d sobre %d: coinciden ENTERAS "+
			"hoy, y es casualidad", casNum, casDen, cobNum, cobDen)
	case casNum == cobNum || casDen == cobDen:
		t.Logf("casillas del plan %d de %d y cobertura %d sobre %d: coincide una mitad",
			casNum, casDen, cobNum, cobDen)
	default:
		t.Logf("casillas del plan %d de %d y cobertura %d sobre %d: ya no coinciden, y el "+
			"descargo se exige igual, porque lo confundible es la forma y no el valor",
			casNum, casDen, cobNum, cobDen)
	}
	if faltan == 0 {
		t.Logf("los dos documentos traen el descargo")
	}
}

// EL CONTROL POSITIVO DE LA RAMA QUE LA VERSION ANTERIOR SALTABA.
//
// La version de antes del 22-09-2026 no exigia nada cuando las cifras dejaban de
// coincidir, y esa rama no la recorria ningun dato: era un descargo que ninguna
// entrada alcanzaba, o sea un descargo que no existe. Aqui se recorre con datos
// sinteticos, incluido el caso peor, que es **cerca pero distinto**.
func TestElDescargoSeExigeAunqueLasCifrasDejenDeCoincidir(t *testing.T) {
	const conNota = "publica **79 de 145 casillas** y lo dice: " + claveDeLaNotaCasual
	const sinNota = "publica **79 de 145 casillas** y no dice nada"

	for _, c := range []struct {
		nombre string
		docs   []documentoConCifra
		faltan int
	}{
		{"coinciden enteras, con nota",
			[]documentoConCifra{{"a", conNota, 78, 144}, {"b", conNota, 78, 144}}, 0},
		{"coinciden enteras, sin nota",
			[]documentoConCifra{{"a", sinNota, 78, 144}, {"b", sinNota, 78, 144}}, 2},
		// EL CASO PEOR, y el que la version anterior dejaba pasar: una diferencia
		// de uno se lee como un descuadre que alguien deberia arreglar.
		{"cerca pero distintas, sin nota",
			[]documentoConCifra{{"a", sinNota, 79, 145}, {"b", sinNota, 78, 144}}, 2},
		{"lejos, sin nota",
			[]documentoConCifra{{"a", sinNota, 12, 40}, {"b", sinNota, 78, 144}}, 2},
		{"solo una la trae",
			[]documentoConCifra{{"a", conNota, 79, 145}, {"b", sinNota, 78, 144}}, 1},
	} {
		var falso testing.T
		if got := notasQueFaltan(&falso, c.docs); got != c.faltan {
			t.Errorf("%s: cuento %d documentos sin descargo y esperaba %d",
				c.nombre, got, c.faltan)
		}
		if falso.Failed() != (c.faltan > 0) {
			t.Errorf("%s: el detector %s, y con %d documentos sin descargo tenia que %s",
				c.nombre, siFallo(falso.Failed()), c.faltan, siFallo(c.faltan > 0))
		}
	}
}

// EL CONTROL NEGATIVO DE LOS LECTORES.
//
// Su fallo probable es casar de mas: «N de M» aparece en prosa corriente, y un
// lector que cazara cualquier par de numeros compararia dos cifras que no son
// las publicadas y pediria el descargo por nada. El fallo contrario, no casar,
// ya para en rojo arriba.
func TestLosLectoresDeLasDosCifrasNoCasanDeMas(t *testing.T) {
	for _, c := range []struct {
		nombre string
		re     *regexp.Regexp
		fuente string
		quiero []string
	}{
		{"las casillas, con su forma", reCasillasDelPlan,
			"cerradas **78 de 144 casillas** y 266 relojes", []string{"78", "144"}},
		{"sin negrita no vale", reCasillasDelPlan, "cerradas 78 de 144 casillas", nil},
		{"otra cosa contada igual tampoco", reCasillasDelPlan,
			"**78 de 144 paquetes**", nil},
		{"la cobertura, con su forma", reCoberturaEstricta,
			"78 relojes **cuyo intervalo lo escribe la norma**, sobre 144 puntos",
			[]string{"78", "144"}},
		{"la cobertura sin el matiz no vale", reCoberturaEstricta,
			"78 relojes, sobre 144 puntos", nil},
	} {
		m := c.re.FindStringSubmatch(c.fuente)
		if c.quiero == nil {
			if m != nil {
				t.Errorf("%s: el lector casa %q y no deberia, saca %v", c.nombre, c.fuente, m[1:])
			}
			continue
		}
		if m == nil {
			t.Errorf("%s: el lector no ve la cifra en %q", c.nombre, c.fuente)
			continue
		}
		if m[1] != c.quiero[0] || m[2] != c.quiero[1] {
			t.Errorf("%s: saca %s/%s y esperaba %s/%s", c.nombre, m[1], m[2],
				c.quiero[0], c.quiero[1])
		}
	}
}

// documentoConCifra es un documento con la fraccion que publica.
type documentoConCifra struct {
	nombre string
	texto  string
	num    int
	den    int
}

// notasQueFaltan exige el descargo en cada documento y devuelve cuantos no lo
// traen. El descargo se exige SIEMPRE: ver el godoc de arriba.
func notasQueFaltan(t *testing.T, docs []documentoConCifra) int {
	t.Helper()
	faltan := 0
	for _, d := range docs {
		if strings.Contains(d.texto, claveDeLaNotaCasual) {
			continue
		}
		faltan++
		t.Errorf(`%s publica «%d de %d» y no dice que esa cifra no mide lo mismo que la
  del otro documento.

  Una cuenta casillas del plan y la otra relojes sobre puntos censados. No tienen
  ninguna relacion, y las dos tienen la misma forma en dos documentos que se leen
  juntos: quien vea las dos va a querer cuadrarlas, y cuadrarlas corrompe una en
  silencio.

  Que hoy se parezcan o no da igual, y esa es la correccion del 22-09-2026: una
  diferencia de uno se lee como un descuadre que alguien deberia arreglar, o sea
  que el riesgo es MAYOR cuando casi coinciden.

  Arreglo: la nota tiene que nombrar %s, que es esta puerta.`,
			d.nombre, d.num, d.den, claveDeLaNotaCasual)
	}
	return faltan
}

func siFallo(b bool) string {
	if b {
		return "fallo"
	}
	return "no fallo"
}

func leerFichero(t *testing.T, ruta string) string {
	t.Helper()
	b, err := os.ReadFile(ruta) // #nosec G304 -- ruta constante del propio repositorio
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// enteroDeCifra convierte una cifra leida de un documento.
//
// No reusa el ayudante de otro test del paquete a proposito: este PARA si la
// cifra no se entiende, en vez de devolver cero. Un cero silencioso aqui haria
// que dos cifras ilegibles «coincidieran» y la puerta pediria la nota por nada.
func enteroDeCifra(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(s)
	if err != nil {
		t.Fatalf("%q no es un numero: %v. Un cero por defecto haria que dos cifras "+
			"ilegibles se leyeran como iguales", s, err)
	}
	return n
}
