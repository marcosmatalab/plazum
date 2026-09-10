package plazum

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// DOS CIFRAS QUE COINCIDEN POR CASUALIDAD TIENEN QUE DECIRLO.
//
// # El riesgo, que es de lectura y no de calculo
//
// `ETAPAS.md` publica «78 de 144 casillas» y `README.md` publica «78 relojes
// sobre 144 puntos censados». Las DOS mitades coinciden hoy, y no tienen ninguna
// relacion:
//
//	el de ETAPAS.md   casillas del plan, derivadas del arbol de ese fichero
//	                  por estado_del_plan_test.go
//	el del README     relojes con intervalo de la norma sobre puntos censados,
//	                  y su denominador sale de sumar las siete filas con
//	                  denominador de paquetes/marcos-v1.json
//
// Las dos estan bien computadas y cada una tiene su puerta. Lo que no tiene
// puerta es la LECTURA: el dia que una se mueva, quien vea dos documentos que
// decian lo mismo y ahora no va a querer cuadrarlos, y cuadrarlos corrompe uno
// en silencio. No hay forma mecanica de impedir esa edicion; lo que si se puede
// es que los dos documentos avisen MIENTRAS el parecido exista.
//
// # Por que la puerta solo exige la nota cuando coinciden
//
// Porque es una guarda proporcionada: si dejan de parecerse, la confusion se
// deshace sola y obligar a mantener una nota sobre un parecido que ya no existe
// seria una puerta que salta sobre trabajo legitimo. Salta exactamente cuando el
// riesgo esta, y se calla cuando no.
func TestLasDosCifrasQueCoincidenPorCasualidadLoDicen(t *testing.T) {
	etapas := leerFichero(t, "ETAPAS.md")
	readme := leerFichero(t, "README.md")

	// LAS CIFRAS SE LEEN DE LOS DOCUMENTOS, no se escriben aqui: escribirlas
	// seria una tercera copia, y entonces la que manda seria otra.
	reCasillas := regexp.MustCompile(`\*\*(\d+) de (\d+) casillas\*\*`)
	reCobertura := regexp.MustCompile(`(\d+) relojes \*\*cuyo intervalo lo escribe la norma\*\*, sobre (\d+) puntos`)

	mc := reCasillas.FindStringSubmatch(etapas)
	if mc == nil {
		t.Fatalf("ETAPAS.md ya no publica «N de M casillas» con esa forma. Si cambio de " +
			"redaccion, este lector se quedo viejo y hay que arreglarlo: sin el, la puerta " +
			"pasaria sin comparar nada")
	}
	mr := reCobertura.FindStringSubmatch(readme)
	if mr == nil {
		t.Fatalf("README.md ya no publica la cobertura con esa forma. Mismo caso: un " +
			"lector que no encuentra nada no puede dar verde")
	}

	casNum, casDen := enteroDeCifra(t, mc[1]), enteroDeCifra(t, mc[2])
	cobNum, cobDen := enteroDeCifra(t, mr[1]), enteroDeCifra(t, mr[2])
	t.Logf("casillas del plan: %d de %d | cobertura del corpus: %d sobre %d",
		casNum, casDen, cobNum, cobDen)

	const claveEtapas = "TestLasDosCifrasQueCoincidenPorCasualidadLoDicen"
	coincide := casNum == cobNum || casDen == cobDen

	if !coincide {
		// No se exige la nota, y se dice por que: el parecido se deshizo y la
		// confusion con el. Que la nota SIGA no rompe nada.
		t.Logf("las cifras ya no se parecen (%d/%d frente a %d/%d): la nota deja de ser "+
			"obligatoria, aunque quedarse no estorba", casNum, casDen, cobNum, cobDen)
		return
	}

	for _, d := range []struct{ nombre, texto string }{
		{"ETAPAS.md", etapas},
		{"README.md", readme},
	} {
		if !strings.Contains(d.texto, claveEtapas) {
			t.Errorf(`%s publica una cifra que se PARECE a la del otro documento (%d/%d
  frente a %d/%d) y no dice que la coincidencia es casual.

  Las dos estan bien computadas y no tienen ninguna relacion: una cuenta casillas
  del plan y la otra relojes sobre puntos censados. El dia que una se mueva, quien
  lea los dos va a querer cuadrarlas, y cuadrarlas corrompe una en silencio.

  Arreglo: la nota tiene que nombrar %s, que es esta puerta.`,
				d.nombre, casNum, casDen, cobNum, cobDen, claveEtapas)
		}
	}
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
