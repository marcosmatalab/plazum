package calendario

import (
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/pantalla"
)

// LA CUENTA SE EXPLICA A SI MISMA, Y CUANDO NO PUEDE LO DICE.
//
// Tres cosas que la pagina sabia y no contaba, las tres de docs/hallazgos-d11.md:
//
//	que sus cubos no cuadran      `Calendario.Cuadra()` existia, y sus dos
//	                              puertas corren contra el corpus publicado y
//	                              contra el dato sintetico. El calendario DEL
//	                              CLIENTE no pasa por ninguna de las dos, asi que
//	                              su descuadre no lo veia nadie: lo que sobra o
//	                              falta simplemente no sale. El escalado lo dice
//	                              desde su primer dia; el calendario, no
//	que UNA cifra hay que creersela  las otras trece se abren, y la que no se
//	                              pintaba exactamente igual. Un numero que hay
//	                              que creerse indistinguible de uno comprobable
//	                              convierte los trece comprobables en trece
//	                              numeros mas que hay que creerse
//	de QUE se compone la suma     `= 218 + 9 + 1` es comprobable y se lee como
//	                              una formula. Los signos no se traducen y se
//	                              quedan; lo que faltaba era la frase

// calendarioQueNoCuadra es el dato sintetico que recorre la rama del descuadre.
//
// POR QUE HACE FALTA UNO PROPIO. `calendarioConVencidas` cuadra a proposito
// (esta escrito en su comentario, y tiene que ser asi: si no cuadrara, la puerta
// de la particion estaria roja sin que hubiera ningun fallo de producto). O sea
// que ninguna entrada de este paquete recorria la rama del aviso, y el corpus
// publicado tampoco: si lo hiciera, seria un fallo abierto. Un aviso que ninguna
// entrada alcanza es un aviso que no existe, y la mutacion lo dejaria verde
// porque no habria nada que romper.
//
// EL DESCUADRE SE METE EN LA LISTA Y NO EN EL CONTADOR, a proposito: asi la
// particion de `Instalados` (37 = 30 + 2 + 1 + 3 + 1) SIGUE cuadrando y el unico
// rojo es el que se busca. Cambiar el contador habria roto dos cosas a la vez y
// no se sabria cual de las dos ve la puerta.
func calendarioQueNoCuadra() pantalla.Calendario {
	cal := calendarioConVencidas()
	// `HitosYaCesados` sigue diciendo 1 y su lista pasa a tener DOS hitos.
	cal.RelojesYaCesados = []pantalla.RelojDescartado{{
		Marco: "urn:demo:m4", Obligacion: "m4.o1", Titulo: "Registro derogado",
		Articulo: "art. 3", Hitos: []string{"anotacion", "conservacion"},
		Regla: "dejo de obligar antes de esta ventana, el 2024-05-05",
	}}
	return cal
}

// calendarioConUnaCifraEnCeroYFilasDetras es el descuadre INVISIBLE.
//
// Es el que ninguna otra puerta de esta pantalla puede ver: la cifra vale cero,
// asi que no se pinta; sin cifra no hay seccion; sin seccion no hay filas que
// contar. Los dos hitos retenidos detras no salen en ninguna parte de la pagina
// y nadie los echa de menos, porque nadie llego a saber que existian.
func calendarioConUnaCifraEnCeroYFilasDetras() pantalla.Calendario {
	cal := calendarioQueNoCuadra()
	cal.HitosYaCesados = 0
	// Y la particion se ajusta para que el unico descuadre siga siendo el que se
	// busca: 36 = 30 + 2 + 0 + 3 + 1.
	cal.HitosDelCorpus = 36
	return cal
}

// LA PUERTA: con los cubos descuadrados, la pagina lo dice, con los dos numeros
// y diciendo CUAL cifra no cuadra.
func TestLaPaginaAvisaDeQueSusCubosNoCuadran(t *testing.T) {
	s, _ := pantallaDePrueba(t, fuenteDoble{d: Derivado{
		Calendario: calendarioQueNoCuadra(), Organizacion: "Acme SL"}, hay: true})
	codigo, cuerpo := pedir(t, s, BasePorDefecto+"/")
	if codigo != http.StatusOK {
		t.Fatalf("GET %s/ ha respondido %d", BasePorDefecto, codigo)
	}
	cuenta := seccionDeLaCuenta(t, cuerpo)
	// Los cubos suman 2 y la cifra dice 1.
	if !strings.Contains(cuenta, marca("calendario.pantalla.cuenta.descuadre[2 1]")) {
		t.Errorf(`la lista de #ya-cesados trae 2 hitos y su cifra dice 1, y la pagina no lo
  dice con los dos numeros.

  Un descuadre que no se pinta no lo ve nadie: el hito que sobra o falta
  simplemente NO SALE, y nadie echa de menos lo que nunca vio. El escalado pinta
  este mismo aviso desde su primer dia.
--- la cuenta ---
%s`, recorta(cuenta, 900))
	}
	// Y DICE CUAL, que es la mitad util del aviso: con catorce cifras, «esto no
	// cuadra» sin nombre obliga a ir a buscarlo a mano.
	if !strings.Contains(cuenta, marca("calendario.pantalla.cuenta.ya_cesados[1]")) {
		t.Errorf("el aviso de descuadre no nombra la cifra que no cuadra:\n%s",
			recorta(cuenta, 900))
	}

	// CONTROL NEGATIVO: cuadrando, el aviso NO sale. Sin esto, un aviso que
	// saliera siempre pasaria la comprobacion de arriba sin decir nada, y ademas
	// entrenaria al lector a ignorarlo, que es como se pierde el unico aviso que
	// de verdad importa.
	s2, _ := pantallaDePrueba(t, fuenteDoble{d: Derivado{
		Calendario: calendarioConVencidas(), Organizacion: "Acme SL"}, hay: true})
	_, sano := pedir(t, s2, BasePorDefecto+"/")
	if strings.Contains(sano, "calendario.pantalla.cuenta.descuadre") {
		t.Errorf("con las catorce cifras cuadrando la pagina sigue avisando de descuadre:\n%s",
			recorta(seccionDeLaCuenta(t, sano), 900))
	}
}

// EL DESCUADRE QUE NINGUNA OTRA PUERTA PUEDE VER: la cifra en cero.
func TestUnaCifraEnCeroConFilasDetrasNoSeCaeDeLaPaginaEnSilencio(t *testing.T) {
	cal := calendarioConUnaCifraEnCeroYFilasDetras()
	s, _ := pantallaDePrueba(t, fuenteDoble{d: Derivado{
		Calendario: cal, Organizacion: "Acme SL"}, hay: true})
	_, cuerpo := pedir(t, s, BasePorDefecto+"/")

	// Que la premisa es la que se dice: la cifra no se pinta y su seccion
	// tampoco. Sin esto, el test podria estar mirando el caso facil.
	if strings.Contains(cuerpo, `id="`+AnclaYaCesados+`"`) {
		t.Fatal("la seccion #ya-cesados se pinta, asi que este test no esta recorriendo el " +
			"caso de la cifra en cero")
	}
	cuenta := seccionDeLaCuenta(t, cuerpo)
	// Los cubos suman 2 y la cifra dice 0.
	if !strings.Contains(cuenta, marca("calendario.pantalla.cuenta.descuadre[2 0]")) {
		t.Errorf(`una cifra en CERO con dos hitos retenidos detras no dice nada.

  Es el descuadre que no se ve de ninguna otra forma: sin cifra no hay seccion,
  sin seccion no hay filas que contar, y los dos hitos no salen en ninguna parte
  de esta pagina. Filtrar el contraste por «se pinta» deja fuera justo este caso.
--- la cuenta ---
%s`, recorta(cuenta, 900))
	}
}

// EL AVISO DE LA PANTALLA SUBSUME LO QUE YA COMPROBABA EL NUCLEO.
//
// `Calendario.Cuadra()` contrasta ocho contadores contra sus ocho listas y
// devuelve un error de desarrollador, en prosa castellana, que no se puede
// pintar. Lo que la pantalla hace es la MISMA comprobacion expresada en lo que
// el lector puede contar, y ademas cubre las dos particiones y las cuatro
// secciones que el nucleo no mira.
//
// Esta puerta ata las dos: si el nucleo dice que algo no cuadra, la pagina tiene
// que estar diciendolo tambien. Sin ella, las dos comprobaciones se separan y la
// pantalla se queda callada sobre un descuadre que el nucleo ya sabia.
func TestSiElNucleoDiceQueNoCuadraLaPaginaTambienLoDice(t *testing.T) {
	// CONTROL POSITIVO: el nucleo se queja y la pagina lo pinta.
	cal := calendarioQueNoCuadra()
	if err := cal.Cuadra(); err == nil {
		t.Fatal("el dato sintetico del descuadre CUADRA para el nucleo, asi que esta puerta " +
			"no esta atando nada")
	}
	var v Vista
	v.rellenarCon(Derivado{Calendario: cal, Organizacion: "Acme SL"})
	if len(v.Descuadres) == 0 {
		t.Error("el nucleo dice que el calendario no cuadra y la pantalla no pinta ni un " +
			"aviso: las dos comprobaciones se han separado, y la que ve el cliente es esta")
	}

	// CONTROL NEGATIVO: con el nucleo conforme, la pantalla no se inventa un
	// descuadre. Es el fallo probable de esta puerta, porque un contraste mal
	// emparejado acusa a la pagina correcta.
	sano := calendarioConVencidas()
	if err := sano.Cuadra(); err != nil {
		t.Fatalf("el dato sintetico sano NO cuadra para el nucleo: %v", err)
	}
	var v2 Vista
	v2.rellenarCon(Derivado{Calendario: sano, Organizacion: "Acme SL"})
	if len(v2.Descuadres) != 0 {
		t.Errorf("la pantalla se inventa %d descuadres sobre un calendario que cuadra: %+v",
			len(v2.Descuadres), v2.Descuadres)
	}
}

// EL CONTRASTE CUBRE TODA CIFRA QUE SE ABRE, EN LOS DOS SENTIDOS.
//
// El sentido que falta es siempre el que se usa. Sin el primero, una cifra nueva
// que se abra en una seccion no la contrasta nadie, y ademas saldria como
// descuadre FALSO en cuanto valiera algo distinto de cero (el mapa no la trae,
// asi que su derivacion contaria 0). Sin el segundo, la vista sigue contando una
// seccion que ya no existe y ese numero no lo mira nadie.
func TestElContrasteDeLaCuentaCubreTodaCifraQueSeAbre(t *testing.T) {
	var v Vista
	v.rellenarCon(Derivado{Calendario: calendarioConVencidas(), Organizacion: "Acme SL"})
	contadas := v.contadas(calendarioConVencidas())

	quieren := CamposQueSeContrastan(v.Cifras)
	// SUELO: sin el, una lista vaciada dejaria los dos bucles recorriendo nada.
	if len(quieren) < 9 {
		t.Fatalf("solo %d cifras se abren en una seccion y hoy son al menos 9: esta puerta "+
			"esta mirando otra cosa", len(quieren))
	}
	for _, campo := range quieren {
		if _, hay := contadas[campo]; !hay {
			t.Errorf("la cifra de CuentaVista.%s se abre en una seccion y nadie cuenta su "+
				"derivacion.\n"+
				"  Arreglo: anadirla en Vista.contadas. Sin eso no se contrasta, y en cuanto "+
				"valga algo distinto de cero la pagina avisara de un descuadre que no existe",
				campo)
		}
	}
	quiereN := map[string]bool{}
	for _, c := range quieren {
		quiereN[c] = true
	}
	var sobran []string
	for campo := range contadas {
		if !quiereN[campo] {
			sobran = append(sobran, campo)
		}
	}
	sort.Strings(sobran)
	for _, campo := range sobran {
		t.Errorf("Vista.contadas cuenta la derivacion de %q y ninguna cifra se abre por ahi: "+
			"o es un renombrado a medias, o es un contraste que no mira nadie", campo)
	}
}

// EL DETECTOR SABE DECIR QUE SI Y QUE NO, y la mutacion se hace FUERA de la
// lista que el propio test conoce.
//
// Se muta la DECLARACION, no el dato: una particion cuyos sumandos no dan su
// total, y una seccion contada de menos. Las dos son las dos ramas del switch,
// y sin recorrerlas por separado media funcion no la ejecuta nadie.
func TestElDetectorDeDescuadresSabeDecirQueSiYQueNo(t *testing.T) {
	sanas := []CifraDeLaCuenta{
		{Campo: "Total", Clave: "k.total", N: 5, Siempre: true,
			Derivacion: CifraConParticion, Partes: []ParteDeCifra{
				{Campo: "A", N: 3}, {Campo: "B", N: 2}}},
		{Campo: "A", Clave: "k.a", N: 3, Derivacion: CifraConSeccion,
			Ancla: "a", Cuadre: CuadreFilas},
		{Campo: "Opaca", Clave: "k.opaca", N: 9, Derivacion: CifraSinDerivacion,
			Motivo: "no se abre"},
	}
	contadas := map[string]int{"A": 3}
	if d := DescuadresDeLaCuenta(sanas, contadas); len(d) != 0 {
		t.Errorf("el detector acusa a una cuenta que cuadra: %+v", d)
	}

	// RAMA 1: la particion no da su total.
	rota := append([]CifraDeLaCuenta(nil), sanas...)
	rota[0].Partes = []ParteDeCifra{{Campo: "A", N: 3}, {Campo: "B", N: 1}}
	d := DescuadresDeLaCuenta(rota, contadas)
	if len(d) != 1 || d[0].Campo != "Total" || d[0].Suma != 4 || d[0].N != 5 {
		t.Errorf("el detector no caza una particion que suma 4 bajo un total de 5: %+v", d)
	}

	// RAMA 2: la seccion trae menos filas que su cifra.
	if d := DescuadresDeLaCuenta(sanas, map[string]int{"A": 2}); len(d) != 1 ||
		d[0].Campo != "A" || d[0].Suma != 2 {
		t.Errorf("el detector no caza una seccion de 2 filas bajo una cifra de 3: %+v", d)
	}

	// LAS DOS FORMAS DE LA NADA, y hacen lo mismo a proposito (invariante 8):
	// «no me han pasado nada» y «me han pasado que no hay nada» son los dos «esa
	// cifra no la ha contrastado nadie», que es lo que hay que decir en voz alta.
	// La forma peligrosa seria la permisiva, saltarse la cifra que no aparece.
	for nombre, m := range map[string]map[string]int{"nil": nil, "vacio": {}} {
		d := DescuadresDeLaCuenta(sanas, m)
		if len(d) != 1 || d[0].Campo != "A" {
			t.Errorf("con el mapa %s el detector da %+v, y una cifra que se abre y que nadie "+
				"contrasta tiene que salir", nombre, d)
		}
	}

	// Y LA QUE NO SE ABRE NO SE ACUSA: `Opaca` vale 9 y no la contrasta nadie
	// porque no promete nada. Acusarla seria pedirle una comprobacion que su
	// declaracion dice que no tiene.
	for _, x := range DescuadresDeLaCuenta(sanas, contadas) {
		if x.Campo == "Opaca" {
			t.Error("el detector acusa a la cifra que se declara sin derivacion")
		}
	}
}

// LA CIFRA QUE HAY QUE CREERSE LO DICE EN LA PAGINA.
func TestLaUnicaCifraSinDerivacionLoDiceEnLaPagina(t *testing.T) {
	s, esp := pantallaDePrueba(t, fuenteDoble{d: Derivado{
		Calendario: calendarioConVencidas(), Organizacion: "Acme SL"}, hay: true})
	_, cuerpo := pedir(t, s, BasePorDefecto+"/")
	cuenta := seccionDeLaCuenta(t, cuerpo)

	if !esp.pidio("calendario.pantalla.cuenta.sin_abrir") {
		t.Error("la pagina no pide el rotulo de la cifra que no se puede abrir")
	}
	// EL AVISO VA EN LA LINEA DE SU CIFRA, no en un pie: pegado al numero que
	// hay que creerse y no a treinta lineas de el.
	lineas := reLineaDeCuenta.FindAllString(cuenta, -1)
	if len(lineas) == 0 {
		t.Fatal("la cuenta no trae ni una linea")
	}
	conAviso := 0
	for _, l := range lineas {
		if strings.Contains(l, marca("calendario.pantalla.cuenta.sin_abrir")) {
			conAviso++
			if !strings.Contains(l, "calendario.pantalla.cuenta.no_alcanzados") {
				t.Errorf("el aviso de «no se puede abrir» esta en una linea que no es la de "+
					"la cifra sin derivacion: %s", l)
			}
		}
	}
	if conAviso != SinDerivacionEsperadas {
		t.Errorf("hay %d lineas con el aviso de «no se puede abrir» y las cifras sin "+
			"derivacion son %d.\n"+
			"  Si sobran, la pagina esta diciendo que no se pueden abrir cifras que si; si "+
			"faltan, hay un numero que hay que creerse pintado igual que los comprobables",
			conAviso, SinDerivacionEsperadas)
	}

	// CONTROL NEGATIVO SOBRE LA DECLARACION, no sobre el dato: ninguna cifra que
	// SI se abre lleva el aviso. Es el fallo probable, porque el aviso se pinta
	// dentro del mismo bucle que las catorce.
	for _, c := range CifrasDeLaCuenta(vistaDePrueba(t, cuerpo)) {
		if c.SinAbrir() == (c.Derivacion != CifraSinDerivacion) {
			t.Errorf("SinAbrir() dice %v de una cifra declarada %v", c.SinAbrir(), c.Derivacion)
		}
	}
}

// LA PARTICION SE LEE COMO UNA FRASE Y SIGUE SIENDO COMPROBABLE.
//
// Las dos mitades, porque cada una sola se puede cumplir rompiendo la otra: se
// puede escribir la frase y quitar la suma (y entonces la cifra vuelve a ser un
// numero que hay que creerse con una frase encima), y se puede dejar la suma sin
// frase (que es de donde se viene).
func TestLaParticionSeLeeComoUnaFraseYSigueSiendoComprobable(t *testing.T) {
	s, _ := pantallaDePrueba(t, fuenteDoble{d: Derivado{
		Calendario: calendarioConVencidas(), Organizacion: "Acme SL"}, hay: true})
	_, cuerpo := pedir(t, s, BasePorDefecto+"/")
	cuenta := seccionDeLaCuenta(t, cuerpo)

	conParticion := 0
	for _, c := range CifrasDeLaCuenta(vistaDePrueba(t, cuerpo)) {
		if c.Derivacion != CifraConParticion || !c.SePinta() {
			continue
		}
		conParticion++
	}
	if conParticion == 0 {
		t.Fatal("ninguna cifra se abre por particion: esta puerta recorre el vacio")
	}
	// SE BUSCA LA CLAVE CON SU ARGUMENTO ABIERTO, no `marca(clave)`: la frase
	// lleva el contador para que el CATALOGO elija entre «se compone de» y «se
	// componen de», asi que el espia la pinta como `[[clave[37]]]`. Buscar
	// `[[clave]]` daria cero y esta puerta se pondria roja acusando a una linea
	// correcta.
	frases := strings.Count(cuenta, "[[calendario.pantalla.cuenta.se_compone_de[")
	if frases != conParticion {
		t.Errorf("hay %d cifras abiertas por particion y %d frases «se compone de»:\n%s",
			conParticion, frases, recorta(cuenta, 900))
	}
	// Y LLEGA CON SU CONTADOR. Sin el numero, el catalogo devuelve la ultima
	// forma (el plural) siempre, y la linea queda mal escrita en el caso de uno.
	// Se comprueba con la cifra de `Instalados`, que vale 37 en el dato
	// sintetico: es la unica forma de saber que el argumento viaja de verdad.
	if !strings.Contains(cuenta, "[[calendario.pantalla.cuenta.se_compone_de[37]]]") {
		t.Errorf("la frase de la particion no recibe el contador de su cifra, asi que el "+
			"catalogo no puede elegir entre singular y plural:\n%s", recorta(cuenta, 900))
	}
	// Y LOS SIGNOS SIGUEN AHI. Son lo comprobable y no se traducen, asi que la
	// frase se anade delante y no sustituye a nada.
	if !strings.Contains(cuenta, "= 30 + 2 + 1 + 3 + 1") {
		t.Errorf("la suma de la particion por tiempo ha dejado de escribirse:\n%s",
			recorta(cuenta, 900))
	}
}
