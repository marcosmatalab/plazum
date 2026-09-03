package alcances

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func almacen(t *testing.T) (*Almacen, string) {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), NombreDelFichero)
	a, err := Abrir(Opciones{Ruta: ruta, Reloj: func() time.Time {
		return time.Date(2026, 9, 4, 9, 0, 0, 0, time.UTC)
	}})
	if err != nil {
		t.Fatalf("abrir un almacen nuevo: %v", err)
	}
	return a, ruta
}

func escribir(t *testing.T, contenido string) string {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), NombreDelFichero)
	if err := os.WriteFile(ruta, []byte(contenido), 0o600); err != nil {
		t.Fatal(err)
	}
	return ruta
}

// ---------------------------------------------------------------------------
// La ida y la vuelta, POR IDENTIDAD DE CADA RESPUESTA
// ---------------------------------------------------------------------------

// TestLoGuardadoVuelveEnteroTrasReabrir es la propiedad entera del paquete: lo
// que una persona contesta hoy tiene que estar manana.
//
// SE COMPARA POR IDENTIDAD, PREGUNTA A PREGUNTA, Y EN LOS DOS SENTIDOS. Contar
// que salen tantas como entraron deja pasar dos cambios que se cancelan: una
// respuesta que se pierde y otra que aparece dan el mismo cardinal y son un
// alcance distinto. El campo por el que casa es el ID DE PREGUNTA, que es lo
// que se escribe en la fila y lo que la superficie compara contra el corpus.
func TestLoGuardadoVuelveEnteroTrasReabrir(t *testing.T) {
	a, ruta := almacen(t)
	ctx := context.Background()
	// LOS IDENTIFICADORES SON SINTETICOS Y TIENEN QUE SERLO. La primera version
	// de este test los escribio con los prefijos de tres marcos reales y
	// `TestNingunaNormaCableada` la puso roja: toda norma vive en su paquete de
	// datos o no vive, y eso vale tambien para un test. Este almacen no conoce
	// ningun corpus, asi que los ids de aqui son cadenas y nada mas.
	quiero := map[string]Respuesta{
		"alfa.q.una":  Si,
		"beta.q.otra": No,
		"gamma.q.mas": Si,
	}
	for id, r := range quiero {
		if err := a.Responder(ctx, "CISO", id, r); err != nil {
			t.Fatalf("responder %s: %v", id, err)
		}
	}

	otro, err := Abrir(Opciones{Ruta: ruta})
	if err != nil {
		t.Fatalf("reabrir: %v", err)
	}
	vuelto, err := otro.De(ctx, "ciso")
	if err != nil {
		t.Fatalf("leer: %v", err)
	}

	// IDA: todo lo que se guardo vuelve, con el MISMO valor.
	for id, r := range quiero {
		v, hay := vuelto.Respuestas[id]
		if !hay {
			t.Errorf("la respuesta a %q no ha vuelto: el trabajo de una persona ha "+
				"desaparecido entre dos arranques", id)
			continue
		}
		if v != r {
			t.Errorf("la respuesta a %q vuelve como %q y se guardo %q", id, v, r)
		}
	}
	// VUELTA: no ha aparecido ninguna que nadie contesto. Sin esta direccion,
	// una respuesta perdida y otra inventada dan el mismo cardinal.
	for id, v := range vuelto.Respuestas {
		if _, esperada := quiero[id]; !esperada {
			t.Errorf("ha vuelto una respuesta a %q (%q) que nadie contesto", id, v)
		}
	}
	if vuelto.Actualizado.IsZero() {
		t.Error("el alcance vuelve sin marca de actualizacion, asi que la pantalla no puede " +
			"decirle a nadie cuando se guardo lo suyo")
	}
}

// ---------------------------------------------------------------------------
// Las tres formas de la nada (invariante 8)
// ---------------------------------------------------------------------------

func TestElFicheroAusenteEsLaUnicaNadaQueNoEsError(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), NombreDelFichero)
	a, err := Abrir(Opciones{Ruta: ruta})
	if err != nil {
		t.Fatalf("un fichero que no existe es una instalacion sin respuestas todavia, no un "+
			"error: %v", err)
	}
	if a.Cuentas() != 0 {
		t.Errorf("un almacen nuevo dice tener %d cuentas", a.Cuentas())
	}
}

func TestUnFicheroPresenteYVacioNoSeLeeComoNadieHaContestado(t *testing.T) {
	ruta := escribir(t, "")
	_, err := Abrir(Opciones{Ruta: ruta})
	if !errors.Is(err, ErrAlmacenVacio) {
		t.Fatalf("un fichero de cero bytes tiene que ser un error y no «aqui no hay "+
			"respuestas»: leerlo asi ensena la entrevista en blanco a quien ya la "+
			"respondio. err = %v", err)
	}
	// Y el error dice como se arregla, que es lo que separa un centinela de un
	// «error inesperado».
	if !strings.Contains(err.Error(), "Arreglo") {
		t.Errorf("el error de fichero vacio no dice como se arregla: %v", err)
	}
}

// TestUnDatoQueHayYNoSeEntiendeEsErrorYNuncaElValorPorDefecto recorre la TERCERA
// forma de la nada, que es la que se cuela: un campo presente cuyo valor no se
// interpreta NO es el valor por defecto.
func TestUnDatoQueHayYNoSeEntiendeEsErrorYNuncaElValorPorDefecto(t *testing.T) {
	casos := []struct {
		nombre    string
		contenido string
		centinela error
	}{
		{"sin version", `{"alcances":[]}`, ErrAlmacenSinVersion},
		{"version de otro plazum", `{"version":99,"alcances":[]}`, ErrVersionDesconocida},
		{"no es json", `{esto no`, ErrAlmacenIlegible},
		{"respuesta ausente", `{"version":1,"alcances":[{"usuario":"ciso",
			"actualizado":"2026-09-04T09:00:00Z","respuestas":[{"pregunta":"q1"}]}]}`,
			ErrAlmacenIlegible},
		{"respuesta presente y vacia", `{"version":1,"alcances":[{"usuario":"ciso",
			"actualizado":"2026-09-04T09:00:00Z","respuestas":[{"pregunta":"q1","respuesta":""}]}]}`,
			ErrAlmacenIlegible},
		{"respuesta que no se entiende", `{"version":1,"alcances":[{"usuario":"ciso",
			"actualizado":"2026-09-04T09:00:00Z","respuestas":[{"pregunta":"q1","respuesta":"quizas"}]}]}`,
			ErrAlmacenIlegible},
		{"instante que no se entiende", `{"version":1,"alcances":[{"usuario":"ciso",
			"actualizado":"ayer","respuestas":[]}]}`, ErrAlmacenIlegible},
		{"pregunta vacia", `{"version":1,"alcances":[{"usuario":"ciso",
			"actualizado":"2026-09-04T09:00:00Z","respuestas":[{"pregunta":"","respuesta":"si"}]}]}`,
			ErrAlmacenIlegible},
		{"dos cuentas iguales", `{"version":1,"alcances":[
			{"usuario":"ciso","actualizado":"2026-09-04T09:00:00Z","respuestas":[]},
			{"usuario":"CISO","actualizado":"2026-09-04T09:00:00Z","respuestas":[]}]}`,
			ErrAlmacenIlegible},
		{"la misma pregunta con dos respuestas", `{"version":1,"alcances":[{"usuario":"ciso",
			"actualizado":"2026-09-04T09:00:00Z","respuestas":[
			{"pregunta":"q1","respuesta":"si"},{"pregunta":"q1","respuesta":"no"}]}]}`,
			ErrAlmacenIlegible},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			_, err := Abrir(Opciones{Ruta: escribir(t, c.contenido)})
			if !errors.Is(err, c.centinela) {
				t.Fatalf("%s tenia que fallar con %v y da %v.\n"+
					"  Un dato que HAY y no se entiende no es la nada: tomarlo por el valor "+
					"por defecto es inventarse una respuesta que nadie dio", c.nombre,
					c.centinela, err)
			}
		})
	}
}

// El control negativo: un fichero legitimo TIENE que cargar. Sin esto, una
// version de Abrir que fallara siempre pasaria el test de arriba entero.
func TestUnFicheroBienEscritoCarga(t *testing.T) {
	ruta := escribir(t, `{"version":1,"alcances":[{"usuario":"ciso",
		"actualizado":"2026-09-04T09:00:00Z","respuestas":[
		{"pregunta":"q1","respuesta":"si"},{"pregunta":"q2","respuesta":"no"}]}]}`)
	a, err := Abrir(Opciones{Ruta: ruta})
	if err != nil {
		t.Fatalf("un almacen bien escrito no carga: %v", err)
	}
	al, err := a.De(context.Background(), "ciso")
	if err != nil {
		t.Fatal(err)
	}
	if al.Respuestas["q1"] != Si || al.Respuestas["q2"] != No {
		t.Fatalf("las respuestas no se han leido: %v", al.Respuestas)
	}
}

// LeerRespuesta es la frontera de entrada de un solo campo, y se recorre entera
// porque es la que va a repetirse en cada formulario de la v1.
func TestLeerRespuestaSoloAdmiteSiYNo(t *testing.T) {
	for _, bueno := range []struct {
		v string
		r Respuesta
	}{{"si", Si}, {"no", No}} {
		if r, err := LeerRespuesta(bueno.v); err != nil || r != bueno.r {
			t.Errorf("LeerRespuesta(%q) = %v, %v", bueno.v, r, err)
		}
	}
	for _, malo := range []string{"", " ", "SI", "si ", "quizas", "true", "1", "sí"} {
		if _, err := LeerRespuesta(malo); !errors.Is(err, ErrRespuestaNoInterpretable) {
			t.Errorf("LeerRespuesta(%q) tenia que ser un error y da %v", malo, err)
		}
	}
	// Y el error no devuelve entera la entrada: repetirla sin acotar es la
	// mitad de un reflejado.
	largo := strings.Repeat("a", 4000)
	err := func() error { _, e := LeerRespuesta(largo); return e }()
	if len(err.Error()) > 500 {
		t.Errorf("el error de una respuesta ilegible mide %d caracteres: esta devolviendo la "+
			"entrada entera", len(err.Error()))
	}
}

// ---------------------------------------------------------------------------
// Contra el atacante: una cuenta no lee ni pisa el alcance de otra
// ---------------------------------------------------------------------------

func TestUnaCuentaNoLeeNiPisaElAlcanceDeOtra(t *testing.T) {
	a, _ := almacen(t)
	ctx := context.Background()
	if err := a.Responder(ctx, "ciso", "q.suya", Si); err != nil {
		t.Fatal(err)
	}
	if err := a.Responder(ctx, "becario", "q.del.becario", No); err != nil {
		t.Fatal(err)
	}

	dela, err := a.De(ctx, "becario")
	if err != nil {
		t.Fatal(err)
	}
	if _, hay := dela.Respuestas["q.suya"]; hay {
		t.Error("el alcance del becario trae una respuesta de la cuenta del CISO. Un almacen " +
			"donde una cuenta ve lo de otra no es estado de la cuenta, es un cajon comun")
	}
	// Y no la ha pisado: reemplazar el alcance entero de una cuenta no toca el
	// de la otra. Es el fallo tipico de un volcado que se guarda mal.
	if err := a.Reemplazar(ctx, "becario", nil); err != nil {
		t.Fatal(err)
	}
	del, err := a.De(ctx, "ciso")
	if err != nil {
		t.Fatal(err)
	}
	if del.Respuestas["q.suya"] != Si {
		t.Errorf("borrar el alcance del becario se ha llevado por delante el del CISO: %v",
			del.Respuestas)
	}
}

// UN ALCANCE SIN DUENO NO SE GUARDA. Es la guarda que impide que una peticion
// sin sesion escriba en un cajon que despues leeria cualquiera, y se recorre
// con las DOS formas de la nada del nombre: ausente y presente-en-blanco.
func TestSinUsuarioNoSeLeeNiSeEscribe(t *testing.T) {
	a, _ := almacen(t)
	ctx := context.Background()
	for _, nadie := range []string{"", "   ", "\t"} {
		if err := a.Responder(ctx, nadie, "q1", Si); !errors.Is(err, ErrSinUsuario) {
			t.Errorf("Responder con el usuario %q tenia que fallar con ErrSinUsuario y da %v",
				nadie, err)
		}
		if _, err := a.De(ctx, nadie); !errors.Is(err, ErrSinUsuario) {
			t.Errorf("De con el usuario %q tenia que fallar con ErrSinUsuario y da %v",
				nadie, err)
		}
		if err := a.Reemplazar(ctx, nadie, map[string]Respuesta{"q1": Si}); !errors.Is(err, ErrSinUsuario) {
			t.Errorf("Reemplazar con el usuario %q tenia que fallar y da %v", nadie, err)
		}
	}
	if a.Cuentas() != 0 {
		t.Errorf("se han creado %d cuentas escribiendo sin usuario", a.Cuentas())
	}
}

// El nombre se normaliza, asi que «CISO» y «ciso» son la misma persona y ven la
// misma entrevista. Sin esto, entrar escribiendo el nombre de otra forma
// ensenaria la entrevista en blanco.
func TestElNombreDeLaCuentaSeNormaliza(t *testing.T) {
	a, _ := almacen(t)
	ctx := context.Background()
	if err := a.Responder(ctx, "  CISO ", "q1", Si); err != nil {
		t.Fatal(err)
	}
	al, err := a.De(ctx, "ciso")
	if err != nil {
		t.Fatal(err)
	}
	if al.Respuestas["q1"] != Si {
		t.Fatalf("«CISO» y «ciso» tienen dos cajones distintos: %v", al.Respuestas)
	}
}

// ---------------------------------------------------------------------------
// Dos pestanas guardando a la vez
// ---------------------------------------------------------------------------

// TestDosPestanasContestandoALaVezConservanLasDos es la razon de que Responder
// sea un DELTA y no un volcado.
//
// Con un volcado, la segunda pestana escribe el estado que leyo antes de que la
// primera guardara, y la respuesta de la primera desaparece sin un error en
// ningun sitio. Aqui se contestan N preguntas distintas en paralelo y tienen que
// estar LAS N.
func TestDosPestanasContestandoALaVezConservanLasDos(t *testing.T) {
	a, ruta := almacen(t)
	ctx := context.Background()
	const n = 24

	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = a.Responder(ctx, "ciso", "q."+string(rune('a'+i%26))+itoa(i), Si)
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("la pestana %d no ha podido guardar: %v", i, err)
		}
	}

	// Se comprueba SOBRE EL DISCO, no sobre la memoria: lo que importa es lo
	// que estara manana.
	otro, err := Abrir(Opciones{Ruta: ruta})
	if err != nil {
		t.Fatalf("reabrir tras la concurrencia: %v", err)
	}
	al, err := otro.De(ctx, "ciso")
	if err != nil {
		t.Fatal(err)
	}
	for i := range n {
		id := "q." + string(rune('a'+i%26)) + itoa(i)
		if al.Respuestas[id] != Si {
			t.Errorf("la respuesta %q se ha perdido en la concurrencia. Con un volcado en vez "+
				"de un delta, la ultima pestana en escribir se lleva por delante lo que "+
				"guardaron las demas", id)
		}
	}
	if len(al.Respuestas) != n {
		t.Errorf("han sobrevivido %d respuestas de %d", len(al.Respuestas), n)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// ---------------------------------------------------------------------------
// Olvidar y reemplazar
// ---------------------------------------------------------------------------

func TestOlvidarQuitaSoloEsaRespuesta(t *testing.T) {
	a, ruta := almacen(t)
	ctx := context.Background()
	_ = a.Responder(ctx, "ciso", "q1", Si)
	_ = a.Responder(ctx, "ciso", "q2", No)
	if err := a.Olvidar(ctx, "ciso", "q1"); err != nil {
		t.Fatal(err)
	}
	// Olvidar la que no esta no es un error: lo que se pedia (que no conste) ya
	// pasa.
	if err := a.Olvidar(ctx, "ciso", "q9"); err != nil {
		t.Errorf("olvidar una respuesta que no consta tenia que ser inocuo: %v", err)
	}
	otro, _ := Abrir(Opciones{Ruta: ruta})
	al, _ := otro.De(ctx, "ciso")
	if _, hay := al.Respuestas["q1"]; hay {
		t.Error("q1 sigue guardada despues de olvidarla")
	}
	if al.Respuestas["q2"] != No {
		t.Errorf("olvidar q1 se ha llevado q2: %v", al.Respuestas)
	}
}

func TestReemplazarConMapaVacioDejaLaCuentaSinRespuestas(t *testing.T) {
	a, ruta := almacen(t)
	ctx := context.Background()
	_ = a.Responder(ctx, "ciso", "q1", Si)
	if err := a.Reemplazar(ctx, "ciso", map[string]Respuesta{}); err != nil {
		t.Fatal(err)
	}
	otro, _ := Abrir(Opciones{Ruta: ruta})
	al, _ := otro.De(ctx, "ciso")
	if len(al.Respuestas) != 0 {
		t.Errorf("empezar de cero ha dejado %d respuestas: %v", len(al.Respuestas), al.Respuestas)
	}
	// Y la cuenta SIGUE existiendo con su marca: «he empezado de cero» y «nunca
	// he contestado» son dos cosas distintas, y la marca las separa.
	if al.Actualizado.IsZero() {
		t.Error("empezar de cero ha borrado la marca de actualizacion, asi que la pantalla no " +
			"puede distinguir «lo he limpiado» de «nunca he contestado»")
	}
}

// El valor cero de Respuesta esta prohibido en la escritura: guardarlo pondria
// en disco una respuesta vacia, que es lo que LeerRespuesta rechaza al volver.
func TestNoSeGuardaElValorCeroDeRespuesta(t *testing.T) {
	a, _ := almacen(t)
	ctx := context.Background()
	if err := a.Responder(ctx, "ciso", "q1", Ninguna); !errors.Is(err, ErrRespuestaNoInterpretable) {
		t.Fatalf("guardar el valor cero tenia que fallar y da %v", err)
	}
	if err := a.Reemplazar(ctx, "ciso", map[string]Respuesta{"q1": Ninguna}); !errors.Is(err,
		ErrRespuestaNoInterpretable) {
		t.Fatalf("reemplazar con el valor cero tenia que fallar y da %v", err)
	}
}

func TestUnIdentificadorDePreguntaConInvisiblesNoEntra(t *testing.T) {
	a, _ := almacen(t)
	ctx := context.Background()
	for _, malo := range []string{"", "  ", "q 1", "q\t1", "q\x001", strings.Repeat("q", 400)} {
		if err := a.Responder(ctx, "ciso", malo, Si); !errors.Is(err, ErrPreguntaNoValida) {
			t.Errorf("el identificador %q tenia que rechazarse y da %v", malo, err)
		}
	}
}

func TestAbrirSinRutaFalla(t *testing.T) {
	if _, err := Abrir(Opciones{}); !errors.Is(err, ErrSinRuta) {
		t.Fatalf("un almacen sin ruta se pierde al parar el proceso: %v", err)
	}
}

// El fichero se escribe entero o no se escribe: si el rename es atomico, no
// existe el instante en que el almacen esta a medias. Se comprueba lo que se
// puede comprobar sin cortar la corriente: que no quedan temporales tirados.
func TestNoQuedanTemporalesDespuesDeGuardar(t *testing.T) {
	a, ruta := almacen(t)
	ctx := context.Background()
	for i := range 5 {
		if err := a.Responder(ctx, "ciso", "q"+itoa(i), Si); err != nil {
			t.Fatal(err)
		}
	}
	entradas, err := os.ReadDir(filepath.Dir(ruta))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entradas {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("ha quedado un temporal tirado: %s", e.Name())
		}
	}
}
