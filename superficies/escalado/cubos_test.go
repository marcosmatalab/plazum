package escalado

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	nescalado "github.com/marcosmatalab/plazum/nucleo/escalado"
)

// LOS OCHO CUBOS DEL ESCALADO, ROTULADOS.
//
// # El agujero que cierra, con su cardinal
//
// La plantilla pintaba `{{.Estado}}: {{.N}}`, o sea `nucleo/escalado.Estado` en
// crudo: OCHO valores en prosa castellana. En la pagina en INGLES salian en
// castellano, y en las dos salian sin ninguna explicacion. Es D11-a #3 de
// docs/hallazgos-d11.md, y la pantalla del acta ya lo habia resuelto con la
// familia `acta.cubo.*`, asi que habia patron y no habia nada que inventar.
//
// # Contra el argumento que estaba escrito, porque estaba escrito
//
// El godoc de `ClavesDeCatalogo` decia que traducirlos «crearia dos nombres para
// el mismo cubo en dos medios del mismo producto, que es como se pierde a alguien
// que compara una captura de pantalla con un log». El argumento es bueno y la
// conclusion era falsa, y el fallo esta en que compara los dos medios en el MISMO
// idioma. El lector ingles no tenia un nombre distinto para ese cubo: no tenia
// NINGUNO. Y la coherencia que aquel parrafo defendia se conserva donde de verdad
// se compara, que es en castellano, porque el rotulo espanol de cada cubo es
// LETRA POR LETRA la constante del nucleo.
//
// # Las dos direcciones, y ninguna sobra
//
//	del nucleo al catalogo   un estado nuevo sin rotulo saldria en crudo en la
//	                         pantalla de un cliente, y en el idioma que sea
//	del catalogo al nucleo   un rotulo que ya no nombra ningun estado es peso
//	                         muerto que hay que traducir a cada idioma nuevo, y
//	                         ademas es el sintoma de un renombrado a medias

func TestCadaEstadoDelEscaladoTieneSuRotuloYNingunoSobra(t *testing.T) {
	estados := nescalado.EstadosPosibles()
	// SUELO: sin el, un `EstadosPosibles` vaciado dejaria este test verde
	// habiendo recorrido la nada.
	if len(estados) != 8 {
		t.Fatalf("el nucleo declara %d estados y hoy son 8: o la particion ha cambiado y "+
			"esta puerta no se ha enterado, o esta mirando otra cosa", len(estados))
	}

	// DEL NUCLEO AL CATALOGO.
	vistas := map[string]nescalado.Estado{}
	for _, e := range estados {
		c, hay := ClaveDelCubo(e)
		if !hay {
			t.Errorf("el estado %q no tiene rotulo de catalogo.\n"+
				"  Arreglo: emparejarlo en el mapa `cubos` y redactarlo en es.json y en.json. "+
				"Sin eso sale la palabra del nucleo en crudo, en castellano, tambien en la "+
				"pagina en ingles", e)
			continue
		}
		if otro, repetida := vistas[c]; repetida {
			t.Errorf("la clave %q rotula %q y tambien %q: dos cubos distintos con el mismo "+
				"nombre en pantalla no se pueden distinguir", c, otro, e)
		}
		vistas[c] = e
	}

	// DEL CATALOGO AL NUCLEO. Se recorre el MAPA, que es el otro extremo:
	// recorrer otra vez EstadosPosibles solo repetiria la direccion de arriba.
	posibles := map[nescalado.Estado]bool{}
	for _, e := range estados {
		posibles[e] = true
	}
	for e, c := range cubos {
		if !posibles[e] {
			t.Errorf("el mapa de cubos rotula %q con %q y ese estado ya no esta en "+
				"EstadosPosibles(): es un renombrado a medias, y la clave se queda en los "+
				"dos idiomas sin que nadie la pinte", e, c)
		}
	}

	// Y LA LISTA PUBLICADA ES LA DE LOS OCHO, ni mas ni menos.
	if n := len(ClavesDeLosCubos()); n != len(estados) {
		t.Errorf("ClavesDeLosCubos() publica %d rotulos y hay %d estados", n, len(estados))
	}
}

// EL CONTROL POSITIVO: los ocho salen POR EL CATALOGO en la pagina.
//
// Que el mapa este completo no dice que la plantilla lo use. Se levanta la
// pantalla con los ocho cubos llenos y se lee la respuesta, que es donde el
// fallo se ve: la version anterior tenia el vocabulario completo, en el nucleo,
// y lo pintaba en crudo igual.
func TestLosOchoCubosSalenTraducidosEnLaPagina(t *testing.T) {
	s, esp := pantallaDePrueba(t, fuenteDoble{p: planConLosOchoCubos(), hay: true}, conSesion)
	codigo, cuerpo := pedir(t, s, http.MethodGet, BasePorDefecto+"/")
	if codigo != http.StatusOK {
		t.Fatalf("GET %s/ ha respondido %d", BasePorDefecto, codigo)
	}
	for _, e := range nescalado.EstadosPosibles() {
		c, hay := ClaveDelCubo(e)
		if !hay {
			continue // ya lo dice la puerta de arriba
		}
		if !esp.pidio(c) {
			t.Errorf("la pagina no pide %q, asi que el cubo %q sigue saliendo en crudo", c, e)
		}
		if !strings.Contains(cuerpo, marca(c)) {
			t.Errorf("la pagina no pinta el rotulo %q del cubo %q", c, e)
		}
	}

	// CONTROL NEGATIVO: la palabra del nucleo YA NO se escribe a pelo en la
	// cuenta. Sin esto, una plantilla que pintara las dos cosas pasaria la
	// comprobacion de arriba y el lector ingles seguiria viendo castellano.
	//
	// DOS RECORTES, y los dos hicieron falta. El primero es la seccion de la
	// cuenta: los MOTIVOS de los escalones que no salen son texto del nucleo y
	// siguen (y deben seguir) apareciendo arriba, asi que buscar en la pagina
	// entera acusaria a lo que esta bien.
	//
	// El segundo es quitar los MARCADORES del catalogo espia, y este nacio de un
	// rojo: la clave `escalado.cubo.pendiente` CONTIENE la palabra «pendiente»,
	// asi que buscarla en `[[escalado.cubo.pendiente]]` la encuentra siempre y
	// este control acusaba a la pagina correcta. Es la misma trampa que
	// documenta ingles_test.go («se miran los VALORES y no las claves»): un
	// identificador de este repositorio esta escrito en castellano, asi que se
	// parece a la cadena que nombra.
	//
	// Y EL TERCERO, del 04-09-2026, es la misma trampa por tercera vez: desde
	// que cada cubo se abre (puerta D11-c), la palabra del nucleo viaja DENTRO
	// DE LA DIRECCION del enlace, escapada, porque es la identidad por la que la
	// derivacion casa con su lista. `href="/escalado/cubo/pendiente"` no es un
	// rotulo y nadie lo lee como tal, pero contiene la cadena, asi que este
	// control volvio a acusar a la pagina correcta en el mismo sitio y por la
	// misma razon. NO DEBILITA EL CONTROL: si la plantilla volviera a pintar
	// `{{.Estado}}` como texto visible, ese texto no esta en ningun href y se
	// sigue cazando. Lo demuestra el control negativo del propio recorte, mas
	// abajo.
	cuenta := sinDirecciones(sinMarcadores(seccionDeLaCuenta(t, cuerpo)))
	for _, e := range nescalado.EstadosPosibles() {
		if strings.Contains(cuenta, string(e)) {
			t.Errorf("la cuenta sigue escribiendo %q, la palabra del nucleo, en vez de su "+
				"rotulo:\n%s", e, recorta(cuenta, 600))
		}
	}
}

// reMarcador es lo que el catalogo espia pinta por cada clave traducida.
var reMarcador = regexp.MustCompile(`(?s)\[\[.*?\]\]`)

// sinMarcadores quita del texto lo que puso el espia, para poder preguntar que
// queda escrito A PELO.
func sinMarcadores(s string) string { return reMarcador.ReplaceAllString(s, "") }

// reDireccion es el atributo href de un enlace.
var reDireccion = regexp.MustCompile(`href="[^"]*"`)

// sinDirecciones quita las direcciones, que llevan IDENTIDADES y no rotulos.
//
// La palabra del nucleo viaja dentro del enlace de cada cubo porque es la
// identidad por la que la derivacion casa con su lista (invariante 7): en el
// href es un identificador, no una palabra que nadie lea. Quitarlo antes de
// preguntar es lo mismo que quitar los marcadores del espia, y por lo mismo.
func sinDirecciones(s string) string { return reDireccion.ReplaceAllString(s, "") }

// CONTROL NEGATIVO DE LOS TRES RECORTES.
//
// Su fallo probable es llevarse por delante lo que tenian que dejar: un recorte
// demasiado ancho deja la cuenta vacia y entonces el control de arriba pasa
// diga lo que diga la pagina, que es un verde vacio con dos capas de pintura.
// Se comprueba que sobre un texto que SI escribe la palabra del nucleo a pelo,
// los tres recortes juntos la dejan pasar.
func TestLosRecortesDeLaCuentaNoSeLlevanLaPalabraQueVigilan(t *testing.T) {
	const crudo = `<li><a href="/escalado/cubo/pendiente" title="[[x.y]]">` +
		`<span class="cubo-n">3</span><span class="cubo-rotulo">pendiente</span></a></li>`
	limpio := sinDirecciones(sinMarcadores(crudo))
	if !strings.Contains(limpio, ">pendiente<") {
		t.Errorf("los recortes se llevan el rotulo visible que este control vigila: %q", limpio)
	}
	// Y LA OTRA DIRECCION: sobre la pagina buena, la direccion SI se va.
	if strings.Contains(sinDirecciones(crudo), "/escalado/cubo/") {
		t.Error("sinDirecciones no quita el href, asi que el control seguiria acusando a la " +
			"pagina correcta")
	}
}

// EL RESPALDO SE RECORRE, que si no es una rama que no existe.
//
// La plantilla tiene una rama para el estado SIN clave, en los DOS sitios donde
// se pinta un estado (el cubo de la cuenta y el escalon que no sale). Con el mapa
// completo no la pisa nadie, o sea M47 en su forma mas pura: una rama defensiva
// que nunca se ejecuta y de la que nadie sabe si funciona. Se recorre con un
// estado inventado, que es dato sintetico FUERA de la lista que el propio mapa
// conoce.
//
// # Lo que este test encontro al escribirse, que no era un fallo del rotulo
//
// El primer intento salio rojo con el cubo VACIO: un estado que no esta en
// `EstadosPosibles()` no se pintaba en la cuenta en absoluto, porque el bucle
// recorria la particion y consultaba el mapa. Su recuento no salia en ningun
// cubo, no se sumaba, y lo unico que quedaba de el era el aviso de descuadre,
// que dice que los numeros no cuadran y NO dice cual falta. Eso no era una rama
// muerta, era el mismo agujero que este tramo persigue, en la otra pantalla: lo
// que no sale nadie lo echa de menos. Se arreglo pintandolos detras, ordenados,
// con la palabra del nucleo por rotulo.
func TestUnEstadoSinRotuloSaleConLaPalabraDelNucleoYNoConUnHueco(t *testing.T) {
	const inventado = nescalado.Estado("un estado que el catalogo no conoce")
	if _, hay := ClaveDelCubo(inventado); hay {
		t.Fatal("el estado inventado tiene rotulo, asi que este test no recorre el respaldo")
	}
	p := planDePrueba()
	p.Cuenta = map[nescalado.Estado]int{inventado: 1}
	p.Planificados = 1
	p.Trabajos[0].Pasos = []nescalado.Paso{{
		Nivel: 1, Cuando: dia(2026, 9, 24), Figura: "m1.direccion",
		Estado: inventado, Motivo: "el motivo sigue estando, que es la mitad que importa",
	}}
	s, _ := pantallaDePrueba(t, fuenteDoble{p: p, hay: true}, conSesion)
	_, cuerpo := pedir(t, s, http.MethodGet, BasePorDefecto+"/")

	// EL CUBO DE LA CUENTA, con su nombre y con su numero. Un cubo cuyo nombre
	// es un hueco es peor que uno en otro idioma: el lector no tiene ni de que
	// se le esta contando uno.
	//
	// SE PREGUNTA POR LAS DOS PIEZAS Y NO POR LA CADENA "estado: N", que es como
	// estaba escrito hasta el 04-09-2026: desde que cada cubo se abre (puerta
	// D11-c) el numero y el rotulo van en dos elementos, y un test atado a la
	// forma vieja se pone rojo por un cambio de maquetacion sin que la propiedad
	// se haya movido. Lo que hay que exigir es que las dos esten, no como se
	// disponen.
	cuenta := seccionDeLaCuenta(t, cuerpo)
	if !strings.Contains(cuenta, `<span class="cubo-rotulo">`+string(inventado)+`</span>`) {
		t.Errorf("un estado sin rotulo no sale con su NOMBRE en la cuenta. El respaldo tiene "+
			"que ser la palabra del nucleo, que es cierta aunque este en otro idioma, y no un "+
			"hueco ni un cubo que desaparece:\n%s", recorta(cuenta, 600))
	}
	if !strings.Contains(cuenta, `<span class="cubo-n">1</span>`) {
		t.Errorf("un estado sin rotulo no sale con su NUMERO en la cuenta:\n%s",
			recorta(cuenta, 600))
	}
	// Y NO HAY DESCUADRE: el cubo suelto se suma como cualquier otro. Si esto
	// falla, el estado ha vuelto a caerse de la cuenta y solo queda el aviso.
	if strings.Contains(cuenta, marca("escalado.pantalla.cuenta.descuadre")) ||
		strings.Contains(cuenta, "escalado.pantalla.cuenta.descuadre") {
		t.Errorf("el cubo suelto no se suma, asi que la pagina avisa de descuadre en vez de "+
			"ensenarlo:\n%s", recorta(cuenta, 600))
	}
	// EL ESCALON QUE NO SALE, que es el otro sitio donde se pinta un estado.
	if !strings.Contains(cuerpo, "["+string(inventado)+"]") {
		t.Errorf("el escalon que no sale no nombra su estado: %s", recorta(cuerpo, 400))
	}
}

// planConLosOchoCubos llena LA PARTICION ENTERA.
//
// Hace falta porque los cubos en cero no se pintan, asi que con el plan normal
// (que llena dos) seis de los ocho rotulos no los pediria ningun estado de la
// pantalla, y el inventario del catalogo los daria por sobrantes. Es la forma
// que tiene este arbol de decir «esa rama no la recorre nadie».
func planConLosOchoCubos() Plan {
	p := planDePrueba()
	cuenta := map[nescalado.Estado]int{}
	n := 0
	for _, e := range nescalado.EstadosPosibles() {
		n++
		cuenta[e] = n // numeros distintos: dos cubos iguales no se distinguirian
	}
	p.Cuenta = cuenta
	total := 0
	for _, v := range cuenta {
		total += v
	}
	// CUADRA A PROPOSITO: si no cuadrara, este plan pintaria ademas el aviso de
	// descuadre y el estado dejaria de ser el que dice ser.
	p.Planificados = total
	return p
}

// seccionDeLaCuenta recorta el bloque de la cuenta.
//
// Se para si no lo encuentra. Sin el recorte, buscar la palabra de un estado en
// la pagina entera la encontraria en el MOTIVO de un escalon que no sale, que es
// texto del nucleo legitimo, y el control negativo acusaria a lo que esta bien.
func seccionDeLaCuenta(t *testing.T, cuerpo string) string {
	t.Helper()
	i := strings.Index(cuerpo, `<section class="cuenta">`)
	if i < 0 {
		t.Fatalf("la pagina no trae la seccion de la cuenta:\n%s", recorta(cuerpo, 900))
	}
	resto := cuerpo[i:]
	j := strings.Index(resto, "</section>")
	if j < 0 {
		t.Fatal("la seccion de la cuenta no se cierra")
	}
	return resto[:j]
}

// CONTROL NEGATIVO DEL RECORTE: su forma de mentir en silencio es devolver la
// pagina entera, y entonces el motivo de un escalon contaria como texto de la
// cuenta y el control negativo de arriba se pondria rojo sin culpa de nadie.
func TestElRecorteDeLaCuentaDelEscaladoNoSeLlevaLaPaginaEntera(t *testing.T) {
	s, _ := pantallaDePrueba(t, fuenteDoble{p: planDePrueba(), hay: true}, conSesion)
	_, cuerpo := pedir(t, s, http.MethodGet, BasePorDefecto+"/")
	cuenta := seccionDeLaCuenta(t, cuerpo)
	if len(cuenta) >= len(cuerpo) {
		t.Fatal("el recorte de la cuenta devuelve la pagina entera")
	}
	// RAMA NEGATIVA: fuera se queda lo que solo hay fuera.
	if strings.Contains(cuenta, "Notificacion de incidente grave") {
		t.Error("el recorte de la cuenta se ha llevado dentro el titulo de un trabajo")
	}
	// RAMA POSITIVA: dentro esta lo suyo.
	if !strings.Contains(cuenta, marca("escalado.pantalla.cuenta.titulo")) {
		t.Error("el recorte de la cuenta no trae su propio titulo: mide menos pagina de la " +
			"que dice")
	}
}
