package calendario

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// LA TERCERA FORMA DE ABRIR UNA CIFRA, con sus tres puertas.
//
// # Que se decidio y por que no es una excusa
//
// Tres de las cinco cifras que seguian huerfanas (instalados, en vigor,
// alcanzados) llevaban un motivo compartido: «no son descartes, son el corpus
// entero mirado de tres formas, y enumerarlas seria pintar centenares de
// obligaciones que no son tuyas». Era cierto y no era un motivo: describia por
// que no se pueden ENUMERAR y callaba que una cifra tambien se comprueba
// SUMANDO.
//
// Las dos particiones que las componen existen desde el 28-08-2026 y las
// comprueba `nucleo/pantalla/contabilidad_test.go`:
//
//	por tiempo    instalados = en vigor + estrenan + ya cesados + empiezan tarde
//	                           + ilegibles
//	por alcance   en vigor    = alcanzados + no alcanzados
//
// O sea que la pantalla tenia la demostracion escrita y se la guardaba. Ahora la
// escribe al lado de la cifra, en cifras y sin palabras, que ademas se lee igual
// en los dos idiomas.
//
// # Las tres puertas, y ninguna sobra
//
//	que los sumandos EXISTAN     un sumando que no es campo de CuentaVista es
//	                             una suma de numeros inventados
//	que la suma DE                una particion que no cuadra es la misma
//	                             promesa rota que un enlace a una lista corta
//	que no haya CIRCULOS         `a = b + c` con `c = a - b` son dos ecuaciones
//	                             ciertas, dos sumas que cuadran y cero cifras
//	                             comprobadas. Es la unica forma de mentir que
//	                             tiene esta manera de abrir, y no la tiene el
//	                             enlace: por eso hace falta una puerta propia
//
// # Y la que queda sin abrir, dicha con su cardinal
//
// UNA: `no alcanzados`. No es un olvido y no se puede arreglar sin romper D-13
// (enumerarla serian entre 145 y 201 filas ajenas en los tres perfiles
// publicados) ni sin circularidad (seria demostrarla con `en vigor`, que se
// demuestra con ella). Es el ancla: el UNICO numero de la pagina que hay que
// creerse, y su puerta de terminal esta escrita en su motivo.

// TestLaParticionDeCadaCifraSumaExactamenteEsaCifra es la puerta aritmetica.
func TestLaParticionDeCadaCifraSumaExactamenteEsaCifra(t *testing.T) {
	var v Vista
	v.rellenarCon(Derivado{Calendario: calendarioConVencidas(), Organizacion: "Acme SL"})
	cifras := CifrasDeLaCuenta(v.Cuenta)

	delTipo := map[string]int{}
	for _, c := range cifras {
		delTipo[c.Campo] = c.N
	}

	conParticion := 0
	for _, c := range cifras {
		if c.Derivacion != CifraConParticion {
			continue
		}
		conParticion++
		if len(c.Partes) < 2 {
			t.Errorf("la cifra de CuentaVista.%s dice abrirse por particion con %d sumandos. "+
				"Una particion de uno no demuestra nada: es la misma cifra con otro nombre",
				c.Campo, len(c.Partes))
			continue
		}
		suma := 0
		for _, p := range c.Partes {
			n, hay := delTipo[p.Campo]
			if !hay {
				t.Errorf("la particion de CuentaVista.%s suma %q, que no es una cifra de esta "+
					"cuenta: es un numero inventado dentro de una demostracion", c.Campo, p.Campo)
				continue
			}
			// EL SUMANDO TRAE SU PROPIO VALOR, y tiene que ser el de la cifra
			// que nombra. Sin esto, la particion podria pintar «218» al lado de
			// una linea que dice «30» y las dos saldrian de la misma pagina.
			if n != p.N {
				t.Errorf("la particion de CuentaVista.%s pinta %d para %q y esa cifra vale %d "+
					"en la misma pagina: el lector ve dos numeros distintos para lo mismo",
					c.Campo, p.N, p.Campo, n)
			}
			suma += p.N
		}
		if suma != c.N {
			t.Errorf(`la particion de CuentaVista.%s suma %d y la cifra dice %d.

  La pagina escribe esa suma al lado del numero. Si no da, quien la lea tiene
  delante una demostracion que no demuestra, que es peor que no tener ninguna.`,
				c.Campo, suma, c.N)
		}
	}
	// SUELO: sin el, quitar la particion de las dos cifras dejaria esta puerta
	// verde recorriendo el vacio.
	if conParticion != 2 {
		t.Fatalf("se han recorrido %d cifras con particion y hoy son 2 (instalados y en "+
			"vigor): esta puerta esta midiendo otra cosa", conParticion)
	}
}

// TestNingunaCifraSeDemuestraConsigoMisma es la puerta de la circularidad.
func TestNingunaCifraSeDemuestraConsigoMisma(t *testing.T) {
	if malas := CifrasQueSeDerivanEnCirculo(CifrasDeLaCuenta(CuentaVista{})); len(malas) != 0 {
		t.Errorf(`hay cifras que se demuestran consigo mismas:

%s

  Dos ecuaciones ciertas que son la misma ecuacion escrita del derecho y del
  reves no comprueban nada, y dejan las dos cifras marcadas como abiertas.`,
			strings.Join(malas, "\n"))
	}
}

// CONTROL NEGATIVO DE LA CIRCULARIDAD, con sus dos direcciones.
//
// El detector se ejecuta contra una declaracion sintetica circular (la que
// tendria `no alcanzados = en vigor - alcanzados` con `en vigor = alcanzados +
// no alcanzados`) y contra una legitima donde dos cifras se apoyan en la misma
// tercera, que NO es un circulo. Sin la segunda mitad, un detector que dijera
// que si a todo pasaria la primera.
func TestElDetectorDeCirculosSabeDecirQueSiYQueNo(t *testing.T) {
	circular := []CifraDeLaCuenta{
		{Campo: "EnVigor", Derivacion: CifraConParticion, Partes: []ParteDeCifra{
			{Campo: "Alcanzados"}, {Campo: "NoAlcanzados"}}},
		{Campo: "NoAlcanzados", Derivacion: CifraConParticion, Partes: []ParteDeCifra{
			{Campo: "EnVigor"}, {Campo: "Alcanzados"}}},
		{Campo: "Alcanzados", Derivacion: CifraConSeccion, Ancla: "x", Cuadre: CuadreFilas},
	}
	if malas := CifrasQueSeDerivanEnCirculo(circular); len(malas) == 0 {
		t.Error("el detector no ha visto el circulo mas obvio que hay: `en vigor` y `no " +
			"alcanzados` demostrandose la una con la otra")
	}

	// LA MITAD QUE DEMUESTRA QUE NO DICE QUE SI A TODO. Dos cifras que se
	// apoyan en la misma tercera comparten un nodo y no forman circulo: un
	// detector escrito con `visitados` en vez de con el CAMINO actual acusaria
	// aqui, y el sintoma seria una puerta que hay que borrar.
	sana := []CifraDeLaCuenta{
		{Campo: "Instalados", Derivacion: CifraConParticion, Partes: []ParteDeCifra{
			{Campo: "EnVigor"}, {Campo: "Alcanzados"}}},
		{Campo: "EnVigor", Derivacion: CifraConParticion, Partes: []ParteDeCifra{
			{Campo: "Alcanzados"}, {Campo: "NoAlcanzados"}}},
		{Campo: "Alcanzados", Derivacion: CifraConSeccion, Ancla: "x", Cuadre: CuadreFilas},
		{Campo: "NoAlcanzados", Derivacion: CifraSinDerivacion, Motivo: "D-13"},
	}
	if malas := CifrasQueSeDerivanEnCirculo(sana); len(malas) != 0 {
		t.Errorf("el detector acusa a una declaracion sana: %v.\n"+
			"  Dos cifras apoyadas en la misma tercera NO son un circulo, y una puerta que "+
			"acusa en falso se acaba borrando", malas)
	}
}

// TestLaPaginaEscribeLaSumaDeCadaCifraConParticion es la puerta sobre la
// RESPUESTA, no sobre la declaracion.
//
// Declarar una particion y no pintarla deja la cifra igual de huerfana con una
// declaracion tranquilizadora encima: es el mismo error que declarar un ancla
// que la plantilla no pinta.
func TestLaPaginaEscribeLaSumaDeCadaCifraConParticion(t *testing.T) {
	s, _ := pantallaDePrueba(t, fuenteDoble{d: Derivado{
		Calendario: calendarioConVencidas(), Organizacion: "Acme SL"}, hay: true})
	codigo, cuerpo := pedir(t, s, BasePorDefecto+"/")
	if codigo != http.StatusOK {
		t.Fatalf("GET %s/ ha respondido %d", BasePorDefecto, codigo)
	}
	seccion := seccionDeLaCuenta(t, cuerpo)

	var v Vista
	v.rellenarCon(Derivado{Calendario: calendarioConVencidas(), Organizacion: "Acme SL"})
	escritas := 0
	for _, c := range CifrasDeLaCuenta(v.Cuenta) {
		if c.Derivacion != CifraConParticion {
			continue
		}
		escritas++
		partes := make([]string, 0, len(c.Partes))
		for _, p := range c.Partes {
			partes = append(partes, fmt.Sprint(p.N))
		}
		suma := "= " + strings.Join(partes, " + ")
		if !strings.Contains(seccion, suma) {
			t.Errorf("la cuenta no escribe la suma %q de CuentaVista.%s.\n"+
				"  La cifra se declara abierta por particion y la pagina no la pinta: es "+
				"una cifra huerfana con una declaracion encima.\n--- la cuenta ---\n%s",
				suma, c.Campo, recorta(seccion, 900))
		}
	}
	if escritas != 2 {
		t.Fatalf("se han buscado %d sumas y hoy son 2: esta puerta esta midiendo otra cosa",
			escritas)
	}
}
