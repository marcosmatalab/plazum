package pantallas

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// El doble
// ---------------------------------------------------------------------------

// evidenciasFalsas contesta lo que se le pone al construirla. NO reparte por
// numero de llamada: cuantas veces se la llame depende del corpus de prueba y
// del estado que se este recorriendo, asi que un doble que dependa de eso hace
// que la cobertura de una rama dependa de un numero que este fichero no
// controla. Es la leccion de `consecuenciasFalsas`, con su cardinal.
type evidenciasFalsas struct {
	por   map[string]Evidencia
	falla error
	veces int
}

func (e *evidenciasFalsas) De(context.Context) (map[string]Evidencia, error) {
	e.veces++
	if e.falla != nil {
		return nil, e.falla
	}
	return e.por, nil
}

func conEvidencia(al Alcances, quien string, ev Evidencias) func(*Opciones) {
	base := conGuardado(al, quien)
	return func(o *Opciones) {
		base(o)
		o.Evidencia = ev
	}
}

// idsDeControles devuelve los identificadores de todas las filas que pinta la
// tabla de controles del corpus de prueba, en su orden.
//
// Se sacan DE LA PAGINA y no de una lista escrita al lado por lo mismo que el
// resto de esta superficie: escribir aqui identificadores de norma lo prohibe
// el linter de normas cableadas, y con razon.
func idsDeControles(t *testing.T) []string {
	t.Helper()
	s, _ := superficie(t, corpusDemo())
	_, cuerpo := pedir(t, s, "/controles?pagina=1")
	var out []string
	const abre = `<th scope="row"><code>`
	for {
		i := strings.Index(cuerpo, abre)
		if i < 0 {
			break
		}
		cuerpo = cuerpo[i+len(abre):]
		j := strings.Index(cuerpo, "</code>")
		if j < 0 {
			break
		}
		out = append(out, cuerpo[:j])
		cuerpo = cuerpo[j:]
	}
	return out
}

// unaObligacionDelDemo devuelve el id de la primera fila que pinta la tabla de
// controles del corpus de prueba, para poder darle evidencia sin escribir a
// mano un identificador de norma (que ademas el linter prohibe).
func unaObligacionDelDemo(t *testing.T) string {
	t.Helper()
	s, _ := superficie(t, corpusDemo())
	_, cuerpo := pedir(t, s, "/controles")
	i := strings.Index(cuerpo, `<th scope="row"><code>`)
	if i < 0 {
		t.Fatal("la tabla de controles no pinta ni una fila: el arnes no ejerce nada")
	}
	resto := cuerpo[i+len(`<th scope="row"><code>`):]
	j := strings.Index(resto, "</code>")
	if j < 0 {
		t.Fatal("fila sin cerrar")
	}
	return resto[:j]
}

// ---------------------------------------------------------------------------
// LA FRONTERA DE SESION (invariante 12)
// ---------------------------------------------------------------------------

// TestSinSesionNoSaleNiUnDatoDeEvidenciaYLaPaginaLoDice es LA puerta de A2.
//
// `/controles` se sirve SIN SESION (es un paso alcanzable del camino), y una
// observacion es dato de la INSTALACION. De las dos cosas juntas sale que
// pintarla seria publicarla a quien no ha entrado: que un control falla, desde
// cuando y con que recolector es reconocimiento gratis sobre la postura de
// seguridad de una organizacion.
//
// Las tres afirmaciones, y las tres hacen falta:
//
//  1. sin sesion NO sale ni un dato de evidencia;
//  2. sin sesion la pagina DICE que no se ensena, en la pantalla que se queda;
//  3. y esa frase NO depende de si hay evidencia o no, porque «hay evidencia,
//     entra para verla» ya diria que la hay.
func TestSinSesionNoSaleNiUnDatoDeEvidenciaYLaPaginaLoDice(t *testing.T) {
	id := unaObligacionDelDemo(t)
	ev := &evidenciasFalsas{por: map[string]Evidencia{
		id: {Estado: "pass", Recolector: "recolector-secreto", Cierra: true,
			Recolectada: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)},
	}}

	// SIN SESION: el sujeto es vacio, que es como llega quien no ha entrado.
	s, cat := superficie(t, corpusDemo(), func(o *Opciones) { o.Evidencia = ev })
	w, cuerpo := pedir(t, s, "/controles")
	if w.Code != http.StatusOK {
		t.Fatalf("codigo %d: /controles es un paso ALCANZABLE y tiene que contestar 200 "+
			"sin sesion. Si esto cambia, cambia el cardinal de los pasos que exigen "+
			"sesion y hay que decirlo", w.Code)
	}
	if strings.Contains(cuerpo, "recolector-secreto") {
		t.Error("sin sesion sale el nombre del recolector, que es un dato de dentro " +
			"de la instalacion")
	}
	for _, clave := range []string{"columna.evidencia", "columna.antiguedad",
		"columna.procedencia", "evidencia.pass"} {
		if cat.vistas()[clave] > 0 {
			t.Errorf("sin sesion se pide %q, o sea que la evidencia esta en la pagina", clave)
		}
	}
	if cat.vistas()["evidencia.sin_sesion"] == 0 {
		t.Error("sin sesion la pagina no DICE que la evidencia no se ensena. " +
			"Sin esa frase no hemos protegido un dato: hemos escondido una carencia")
	}
	// Y NO SE HA LLAMADO AL ADAPTADOR. Llamarlo y tirar el resultado dejaria el
	// dato en la memoria del proceso que sirve la peticion anonima, y ademas
	// haria que el tiempo de respuesta delatara cuanta evidencia hay.
	if ev.veces != 0 {
		t.Errorf("sin sesion se ha llamado %d veces al adaptador de evidencia", ev.veces)
	}

	// LA TERCERA AFIRMACION: la frase es la MISMA sin una sola observacion.
	// Si dependiera de si hay evidencia, la propia frase seria la filtracion.
	vacio := &evidenciasFalsas{por: map[string]Evidencia{}}
	s2, cat2 := superficie(t, corpusDemo(), func(o *Opciones) { o.Evidencia = vacio })
	if _, c2 := pedir(t, s2, "/controles"); !strings.Contains(c2,
		rotulo("es", "evidencia.sin_sesion")) {
		t.Error("con cero observaciones la pagina dice OTRA cosa que con evidencia: " +
			"la propia frase delata si hay algo detras")
	}
	if cat2.vistas()["evidencia.sin_sesion"] == 0 {
		t.Error("no se pide la frase con el adaptador vacio")
	}
}

// TestConSesionLaEvidenciaSaleConSuEstadoSuFechaYQuienLoTrajo es el control
// positivo de la puerta de arriba. Sin el, la aprobaria una pantalla que no
// pinta evidencia nunca.
func TestConSesionLaEvidenciaSaleConSuEstadoSuFechaYQuienLoTrajo(t *testing.T) {
	id := unaObligacionDelDemo(t)
	al := nuevoAlmacenFalso()
	ev := &evidenciasFalsas{por: map[string]Evidencia{
		id: {
			Estado:      "pass",
			Motivo:      "todas las observaciones satisfacen el predicado",
			Recolectada: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			Recolector:  "manual",
			Cierra:      true,
			Predicado:   "cada privilegio consta asignado a una funcion declarada",
		},
	}}
	s, cat := superficie(t, corpusDemo(), conEvidencia(al, "ana@ejemplo", ev))
	w, cuerpo := pedir(t, s, "/controles")
	if w.Code != http.StatusOK {
		t.Fatalf("codigo %d", w.Code)
	}
	if ev.veces != 1 {
		t.Errorf("el adaptador se ha llamado %d veces, y la pagina se pinta una vez", ev.veces)
	}
	// LAS TRES COSAS QUE PIDE LA CASILLA: estado, antiguedad y procedencia.
	for _, clave := range []string{"columna.evidencia", "columna.antiguedad",
		"columna.procedencia", "evidencia.pass", "evidencia.recolectada"} {
		if cat.vistas()[clave] == 0 {
			t.Errorf("con sesion no se pide %q", clave)
		}
	}
	if !strings.Contains(cuerpo, "manual") {
		t.Error("no sale quien trajo el dato")
	}
	// Y EL DESCARGO, que en esta pantalla es obligatorio porque ensena pasado.
	if cat.vistas()["evidencia.descargo"] == 0 {
		t.Error("la pantalla ensena estado de evidencia y NO lleva su descargo")
	}
	// La frase de la sesion NO sale cuando hay sesion: un aviso que sale
	// siempre no avisa de nada.
	if cat.vistas()["evidencia.sin_sesion"] > 0 {
		t.Error("con sesion sigue saliendo la frase de que hace falta entrar")
	}
}

// TestSinAdaptadorDeEvidenciaLaTablaNoPrometeNada: el valor cero del puerto.
//
// Y la segunda mitad es la que importa: tampoco sale la frase de «entra para
// verla». Prometer un estado que no va a llegar es peor que no ofrecerlo, y esa
// frase sola le diria a quien entre que hay algo detras cuando no lo hay.
func TestSinAdaptadorDeEvidenciaLaTablaNoPrometeNada(t *testing.T) {
	al := nuevoAlmacenFalso()
	s, cat := superficie(t, corpusDemo(), conGuardado(al, "ana@ejemplo"))
	_, cuerpo := pedir(t, s, "/controles")
	for _, clave := range []string{"columna.evidencia", "evidencia.sin_sesion",
		"evidencia.descargo", "evidencia.sin_prueba"} {
		if cat.vistas()[clave] > 0 {
			t.Errorf("sin adaptador de evidencia se pide %q", clave)
		}
	}
	// CONTROL POSITIVO de que este montaje SI llega a pintar la tabla: sin
	// esto, la afirmacion de arriba la aprobaria una pagina en blanco.
	if cat.vistas()["columna.estado"] == 0 {
		t.Fatalf("este montaje no llega ni a pintar la tabla:\n%s", cuerpo[:min(400, len(cuerpo))])
	}
}

// ---------------------------------------------------------------------------
// LAS TRES FORMAS DE LA NADA, QUE AQUI SON TRES DISTINTAS
// ---------------------------------------------------------------------------

// TestSinPruebaYSinObservacionNoSonLoMismo. Las dos dan «no hay dato» y se
// arreglan por sitios distintos: la primera la arregla quien escribe el
// paquete, la segunda quien monta el recolector. Colapsarlas manda a la
// persona equivocada.
func TestSinPruebaYSinObservacionNoSonLoMismo(t *testing.T) {
	al := nuevoAlmacenFalso()

	// (1) SIN PRUEBA DECLARADA: el adaptador no trae entrada para la fila.
	sinPrueba := &evidenciasFalsas{por: map[string]Evidencia{}}
	s, cat := superficie(t, corpusDemo(), conEvidencia(al, "ana@ejemplo", sinPrueba))
	pedir(t, s, "/controles")
	if cat.vistas()["evidencia.sin_prueba"] == 0 {
		t.Error("una obligacion sin prueba declarada no lo dice")
	}

	// (2) CON PRUEBA Y SIN OBSERVACIONES: hay entrada, y dice que nadie ha
	// recolectado. El estado lo pone el motor (Obsoleto), no la pantalla.
	id := unaObligacionDelDemo(t)
	conPrueba := &evidenciasFalsas{por: map[string]Evidencia{
		id: {Estado: "obsoleto", SinObservaciones: true, Cierra: true,
			Motivo: "no hay ninguna observacion para esta prueba"},
	}}
	s2, cat2 := superficie(t, corpusDemo(), conEvidencia(al, "ana@ejemplo", conPrueba))
	pedir(t, s2, "/controles")
	if cat2.vistas()["evidencia.obsoleto"] == 0 {
		t.Error("una prueba sin observaciones no sale con su estado del motor")
	}
	if cat2.vistas()["evidencia.sin_fecha"] == 0 {
		t.Error("sin observaciones no se dice que no hay fecha; un time.Time cero " +
			"formateado daria el ano 1, que es peor que decir que no se sabe")
	}
}

// TestLaEvidenciaIlegibleNoSeLeeComoQueNadieHaRecolectado es la tercera forma
// de la nada (invariante 8). Es la que absuelve de mas: decirle a alguien que
// su instalacion no tiene nada recolectado cuando lo que pasa es que no hemos
// podido leer el fichero.
func TestLaEvidenciaIlegibleNoSeLeeComoQueNadieHaRecolectado(t *testing.T) {
	al := nuevoAlmacenFalso()
	var visto error
	rota := &evidenciasFalsas{falla: errors.New("json a medias")}
	s, cat := superficie(t, corpusDemo(),
		conEvidencia(al, "ana@ejemplo", rota),
		func(o *Opciones) { o.AlFallar = func(err error) { visto = err } })
	w, cuerpo := pedir(t, s, "/controles")
	if w.Code != http.StatusOK {
		t.Fatalf("codigo %d: la tabla de controles sigue sirviendo aunque la evidencia "+
			"no se pueda leer. Lo que no puede es callarlo", w.Code)
	}
	if cat.vistas()["evidencia.ilegible"] == 0 {
		t.Error("no se dice que la evidencia no se ha podido leer")
	}
	for _, clave := range []string{"columna.evidencia", "evidencia.obsoleto", "evidencia.sin_prueba"} {
		if cat.vistas()[clave] > 0 {
			t.Errorf("con la evidencia ilegible se pinta %q, o sea que se ha degradado "+
				"a «todavia nadie ha recolectado»", clave)
		}
	}
	if visto == nil {
		t.Error("el fallo no se ha reportado a quien monta el servidor")
	}
	_ = cuerpo
}

// ---------------------------------------------------------------------------
// C2 EN LA PANTALLA: aportar no es cerrar
// ---------------------------------------------------------------------------

// TestUnaPruebaQueAportaNoSePintaComoQueCierra. Es C2 llevado hasta el pixel:
// sin esto, una faceta en verde absolveria una obligacion documental entera y
// quien lo lea deja de buscar el documento.
func TestUnaPruebaQueAportaNoSePintaComoQueCierra(t *testing.T) {
	id := unaObligacionDelDemo(t)
	al := nuevoAlmacenFalso()

	aporta := &evidenciasFalsas{por: map[string]Evidencia{
		id: {Estado: "pass", Cierra: false,
			Recolectada: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)},
	}}
	s, cat := superficie(t, corpusDemo(), conEvidencia(al, "ana@ejemplo", aporta))
	pedir(t, s, "/controles")
	if cat.vistas()["evidencia.aporta"] == 0 {
		t.Error("una prueba que NO cierra la obligacion se pinta igual que una que si: " +
			"eso absuelve de mas con cara de dato")
	}

	// LA DIRECCION CONTRARIA. Sin ella, la puerta la aprobaria una pantalla que
	// pinta «aporta» siempre, que es igual de inutil por el otro lado.
	cierra := &evidenciasFalsas{por: map[string]Evidencia{
		id: {Estado: "pass", Cierra: true,
			Recolectada: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)},
	}}
	s2, cat2 := superficie(t, corpusDemo(), conEvidencia(al, "ana@ejemplo", cierra))
	pedir(t, s2, "/controles")
	if cat2.vistas()["evidencia.aporta"] > 0 {
		t.Error("una prueba que SI cierra sale marcada como que solo aporta")
	}
}

// ---------------------------------------------------------------------------
// LOS DOS VOCABULARIOS
// ---------------------------------------------------------------------------

// TestLosDosVerdesDeLaPantallaNoCompartenNiClaveNiClase es la puerta del choque
// que el mapa encontro antes de que ocurriera: `pantallas.NoAplica` y
// `estado.NoAplica` se llaman IGUAL y significan cosas distintas.
//
// El de la izquierda dice «contestaste que no a una pregunta del paquete». El de
// la derecha dice «excluido por la declaracion de aplicabilidad». Con el mismo
// prefijo compartirian clave de catalogo y regla de la hoja, y estarian en dos
// columnas contiguas de la misma tabla.
func TestLosDosVerdesDeLaPantallaNoCompartenNiClaveNiClase(t *testing.T) {
	// Las claves: el prefijo tiene que ser distinto.
	if NoAplica.Clave() == "evidencia."+"no_aplica" {
		t.Fatal("los dos vocabularios comparten clave de catalogo")
	}
	if !strings.HasPrefix(NoAplica.Clave(), "estado.") {
		t.Fatalf("la clave del estado de alcance es %q y se esperaba el prefijo estado.",
			NoAplica.Clave())
	}

	// Y las clases: la de alcance es `e-` y la de evidencia `ev-`. Se comprueba
	// sobre la PLANTILLA, que es donde de verdad se decide, y no sobre una
	// constante escrita al lado de este test.
	b, err := fs.ReadFile(Plantillas(), "plantillas/tabla.html")
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)
	if !strings.Contains(html, `class="estado e-{{.Estado}}"`) {
		t.Error("la pastilla de alcance ha dejado de usar el prefijo e-")
	}
	if !strings.Contains(html, `class="evidencia ev-{{.Estado}}"`) {
		t.Error("la pastilla de evidencia no usa el prefijo ev-, asi que las dos " +
			"comparten las reglas de la hoja")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestElCISOSabePorQueDesdeCuandoYDeDondeSalioElDato es la pasada 3 convertida
// en puerta.
//
// La pregunta del comprador es literal: abre la pantalla, ve un control con su
// estado, ¿sabe POR QUE, DESDE CUANDO y DE DONDE salio el dato? En A1 no se
// podia ni formular porque la pantalla no ensenaba estado de evidencia. Con A2
// si, y la primera respuesta fue que NO: el predicado no se pintaba y la fecha
// salia como el volcado de un time.Time de Go.
//
// Las tres cosas se afirman POR SEPARADO porque son tres preguntas distintas y
// se pueden perder una a una.
func TestElCISOSabePorQueDesdeCuandoYDeDondeSalioElDato(t *testing.T) {
	id := unaObligacionDelDemo(t)
	al := nuevoAlmacenFalso()
	ev := &evidenciasFalsas{por: map[string]Evidencia{
		id: {
			Estado:      "fail_en_plazo",
			Motivo:      "1 recurso(s) fallan, plazo de remediacion hasta el 2026-09-13",
			Recolectada: time.Date(2026, 9, 6, 9, 0, 0, 0, time.UTC),
			Recolector:  "manual",
			Cierra:      true,
			Predicado:   "cada privilegio consta asignado a una funcion declarada",
		},
	}}
	s, _ := superficie(t, corpusDemo(), conEvidencia(al, "ana@ejemplo", ev))
	_, cuerpo := pedir(t, s, "/controles")

	// POR QUE. Dos frases y las dos hacen falta: el predicado dice QUE se mira
	// y el motivo dice por que ha salido ese estado.
	if !strings.Contains(cuerpo, "cada privilegio consta asignado") {
		t.Error("el PREDICADO no se pinta: el estado es una afirmacion sin nada detras")
	}
	if !strings.Contains(cuerpo, "plazo de remediacion hasta") {
		t.Error("el MOTIVO del motor no se pinta: no se sabe si falta un recurso o " +
			"caduco todo")
	}
	// Y NO ESCONDIDO EN UN title: un atributo title no lo lee un lector de
	// pantalla de forma fiable, no se imprime y en un movil no existe.
	if strings.Contains(cuerpo, `title="cada privilegio`) ||
		strings.Contains(cuerpo, `title="1 recurso`) {
		t.Error("la explicacion vive en un atributo title, que es esconderla")
	}

	// DESDE CUANDO. Una fecha legible, no el volcado de un time.Time.
	if !strings.Contains(cuerpo, "2026-09-06") {
		t.Error("no sale la fecha del dato")
	}
	if strings.Contains(cuerpo, "+0000 UTC") {
		t.Error("la fecha sale como el volcado de un time.Time de Go, que es la " +
			"pantalla de cumplimiento pareciendo un terminal")
	}

	// DE DONDE.
	if !strings.Contains(cuerpo, "manual") {
		t.Error("no sale quien trajo el dato")
	}
}
