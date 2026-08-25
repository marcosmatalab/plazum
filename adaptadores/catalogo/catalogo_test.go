package catalogo

import (
	"os"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"dutiq/puertos"
	"dutiq/puertos/contrato"
)

func nuevoParaTest(t *testing.T) *Catalogo {
	t.Helper()
	c, err := Nuevo()
	if err != nil {
		t.Fatalf("el catalogo embebido no carga, asi que el producto no arranca: %v", err)
	}
	return c
}

func TestCumpleElContratoDeCatalogo(t *testing.T) {
	c := nuevoParaTest(t)
	// Se comparte la misma instancia entre subtests a proposito: un catalogo
	// es inmutable despues de Cargar, y si alguna vez deja de serlo, este
	// test es el primero que se entera.
	contrato.Catalogo(t, func() puertos.Catalogo { return c })
}

// ---------------------------------------------------------------------------
// Las puertas de i18n. Son las que mira el CI.
// ---------------------------------------------------------------------------

// Un hueco en ingles es una clave en crudo en la pantalla de un usuario ingles.
// Avisar no vale: una traduccion incompleta que solo avisa se queda incompleta
// para siempre.
func TestPuertaI18nNingunIdiomaTieneHuecos(t *testing.T) {
	c := nuevoParaTest(t)
	for _, idioma := range c.Idiomas() {
		if f := c.Faltantes(idioma); len(f) != 0 {
			t.Errorf("al idioma %q le faltan %d claves que %q si tiene: %v.\n"+
				"Arreglo: traducirlas en adaptadores/catalogo/cadenas/%s.json. Sin esto, "+
				"el usuario de ese idioma ve la clave en crudo en la pantalla",
				idioma, len(f), PorDefecto, f, idioma)
		}
	}
}

// Y al reves: una clave que solo existe en ingles es una clave muerta o un
// renombrado a medias. Nadie la va a ensenar nunca, y el que la escribio cree
// que si.
func TestPuertaI18nNingunIdiomaTieneClavesMuertas(t *testing.T) {
	c := nuevoParaTest(t)
	for _, idioma := range c.Idiomas() {
		if s := c.Sobrantes(idioma); len(s) != 0 {
			t.Errorf("el idioma %q tiene %d claves que %q no tiene: %v.\n"+
				"Arreglo: o se anaden a %s.json, o se borran. Una clave que solo existe en "+
				"un idioma no la ensena nadie", idioma, len(s), PorDefecto, s, PorDefecto)
		}
	}
}

// El descuadre de formateo es el defecto de traduccion mas facil de cometer:
// el traductor se come el %s o lo duplica.
func TestPuertaI18nElFormateoCasaEntreIdiomas(t *testing.T) {
	c := nuevoParaTest(t)
	for _, idioma := range c.Idiomas() {
		if d := c.Descuadres(idioma); len(d) != 0 {
			t.Errorf("en %q hay %d claves cuyos verbos de formateo no casan con %q: %v.\n"+
				"Arreglo: la traduccion tiene que llevar los mismos %%s y en el mismo orden. "+
				"Si el orden cambia en ese idioma, se usa %%[1]s", idioma, len(d), PorDefecto, d)
		}
	}
}

// El recorte de la promesa de aleman, hecho ejecutable.
//
// El mecanismo admite N idiomas y se cargan DOS. El aleman llega cuando exista
// un partner DACH que lo revise, no antes, y hasta entonces no se promete en
// ningun sitio: ni en la web, ni en el README, ni en un desplegable con una
// bandera en gris. Este test es el que convierte esa frase en una puerta: para
// anadir un idioma hay que tocarlo, y tocarlo obliga a mirar esta explicacion.
func TestSoloSeCarganEsYEnMientrasNoHayaPartnerQueRevise(t *testing.T) {
	c := nuevoParaTest(t)
	got := c.Idiomas()
	if len(got) != 2 || got[0] != "es" || got[1] != "en" {
		t.Fatalf("los idiomas cargados son %v y tienen que ser [es en], con es el primero", got)
	}

	entradas, err := os.ReadDir("cadenas")
	if err != nil {
		t.Fatal(err)
	}
	var ficheros []string
	for _, e := range entradas {
		ficheros = append(ficheros, e.Name())
	}
	if len(ficheros) != 2 || ficheros[0] != "en.json" || ficheros[1] != "es.json" {
		t.Fatalf("en cadenas/ hay %v y tiene que haber exactamente en.json y es.json.\n"+
			"Un idioma nuevo no es un fichero que aparece: es una decision. Hace falta "+
			"quien revise ese idioma y responda de lo que dice la interfaz en el, y "+
			"mientras no lo haya, el idioma no se promete", ficheros)
	}
}

// ---------------------------------------------------------------------------
// Traducir bajo ataque
// ---------------------------------------------------------------------------

func TestUnaClaveSinTraducirDevuelveLaClaveYNoElCastellano(t *testing.T) {
	c, err := Cargar(fstest.MapFS{
		"es.json": &fstest.MapFile{Data: []byte(`{"ui.guardar":"Guardar","ui.cancelar":"Cancelar"}`)},
		"en.json": &fstest.MapFile{Data: []byte(`{"ui.guardar":"Save"}`)},
	}, "es", "en")
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Traducir("en", "ui.cancelar"); got != "ui.cancelar" {
		t.Fatalf("una clave sin traducir devolvio %q y tenia que devolver la clave.\n"+
			"Caer al castellano esconde el hueco: el usuario ingles ve media pagina en "+
			"otro idioma y nadie se entera de que falta una cadena", got)
	}
	if f := c.Faltantes("en"); len(f) != 1 || f[0] != "ui.cancelar" {
		t.Fatalf("Faltantes tenia que cazar ese hueco y devolvio %v", f)
	}
}

func TestUnLocaleConRegionSeNormalizaYNoSeQuedaSinTraduccion(t *testing.T) {
	c := nuevoParaTest(t)
	esperado := c.Traducir("en", "ui.guardar")
	for _, locale := range []string{"en-GB", "en_US", "EN", " en ", "en-Latn-GB"} {
		if got := c.Traducir(locale, "ui.guardar"); got != esperado {
			t.Errorf("el locale %q dio %q y tenia que dar %q: un navegador manda lo que "+
				"quiere en Accept-Language y eso no puede decidir si el usuario ve la "+
				"interfaz o ve claves", locale, got, esperado)
		}
	}
}

func TestUnIdiomaInventadoCaeAlPorDefectoYNoDevuelveVacio(t *testing.T) {
	c := nuevoParaTest(t)
	esperado := c.Traducir(PorDefecto, "ui.guardar")
	for _, idioma := range []string{"xx", "de", "", "??", "es-ES-x-hackme"} {
		if got := c.Traducir(idioma, "ui.guardar"); got != esperado {
			t.Errorf("el idioma %q dio %q y tenia que caer al de por defecto (%q)",
				idioma, got, esperado)
		}
	}
}

// El formateo no puede escupir la basura de fmt en una pantalla. Un %!s(MISSING)
// en una etiqueta le dice al comprador que el producto esta roto por dentro.
func TestElFormateoNuncaEscupeBasuraDeFmt(t *testing.T) {
	c, err := Cargar(fstest.MapFS{
		"es.json": &fstest.MapFile{Data: []byte(`{
			"tabla.pedido_por": "Lo piden %s",
			"tabla.dos_verbos": "%s y %s",
			"tabla.porcentaje": "Cubierto al 80%",
			"tabla.sin_verbo": "Alcance"
		}`)},
	}, "es")
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre string
		clave  string
		args   []any
	}{
		{"un argumento de menos", "tabla.dos_verbos", []any{"uno"}},
		{"un argumento de mas", "tabla.pedido_por", []any{"uno", "dos"}},
		{"argumentos sin verbo que los reciba", "tabla.sin_verbo", []any{"uno"}},
		{"un porcentaje literal que parece un verbo", "tabla.porcentaje", []any{"uno"}},
		{"un tipo que el verbo no admite", "tabla.pedido_por", []any{func() {}}},
	}
	for _, cs := range casos {
		t.Run(cs.nombre, func(t *testing.T) {
			got := c.Traducir("es", cs.clave, cs.args...)
			if strings.Contains(got, "%!") {
				t.Fatalf("Traducir devolvio %q: la basura de fmt no puede llegar a la "+
					"pantalla. El descuadre se caza en CI con Descuadres, y aqui se "+
					"degrada a la cadena sin formatear", got)
			}
			if got == "" {
				t.Fatal("Traducir no puede devolver vacio: la etiqueta desaparece y nadie " +
					"se entera de que falta")
			}
		})
	}
	// Y el caso bueno sigue formateando, no vaya a ser que la proteccion se
	// haya comido tambien lo que funciona.
	if got := c.Traducir("es", "tabla.pedido_por", "tres normas"); got != "Lo piden tres normas" {
		t.Fatalf("el formateo legitimo dio %q", got)
	}
	// Sin argumentos no se formatea: un % suelto es un porcentaje.
	if got := c.Traducir("es", "tabla.porcentaje"); got != "Cubierto al 80%" {
		t.Fatalf("una cadena con un porcentaje literal y sin argumentos dio %q", got)
	}
}

// Una clave no se resuelve en cadena. Si el valor de una clave es el nombre de
// otra, se devuelve tal cual: sin resolucion encadenada no hay ciclo posible,
// ni bomba de expansion, ni una cadena que dice algo distinto de lo que pone.
func TestElValorDeUnaClaveNoSeVuelveAResolver(t *testing.T) {
	c, err := Cargar(fstest.MapFS{
		"es.json": &fstest.MapFile{Data: []byte(`{
			"ui.a": "ui.b",
			"ui.b": "ui.a",
			"ui.c": "ui.c"
		}`)},
	}, "es")
	if err != nil {
		t.Fatal(err)
	}
	for clave, esperado := range map[string]string{
		"ui.a": "ui.b",
		"ui.b": "ui.a",
		"ui.c": "ui.c",
	} {
		if got := c.Traducir("es", clave); got != esperado {
			t.Fatalf("Traducir(%q) dio %q y tenia que dar %q sin resolver mas: una "+
				"resolucion encadenada admite ciclos y aqui no hay ninguno porque no "+
				"hay resolucion", clave, got, esperado)
		}
	}
}

// El plural. Lo pide la superficie de pantallas, que pasa un contador y no dos
// claves, porque la forma plural depende del idioma y quien pinta la pantalla
// no sabe en que idioma esta escribiendo.
func TestElPluralLoElijeElCatalogoSegunElContador(t *testing.T) {
	c, err := Cargar(fstest.MapFS{
		"es.json": &fstest.MapFile{Data: []byte(`{
			"menu.aplican": "%d aplica|%d aplican",
			"tabla.sin_verbo": "una fila|varias filas",
			"ui.simple": "Guardar"
		}`)},
	}, "es")
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		clave    string
		args     []any
		esperado string
	}{
		{"menu.aplican", []any{1}, "1 aplica"},
		{"menu.aplican", []any{0}, "0 aplican"},
		{"menu.aplican", []any{2}, "2 aplican"},
		{"menu.aplican", []any{int64(1)}, "1 aplica"},
		{"menu.aplican", []any{uint(1)}, "1 aplica"},
		// Sin contador se devuelve el plural, que es lo que suena bien con
		// una cantidad desconocida. Y sobre todo: NO se devuelven las dos
		// formas con la barra en medio, que es lo que veria el usuario.
		{"tabla.sin_verbo", nil, "varias filas"},
		{"tabla.sin_verbo", []any{"texto"}, "varias filas"},
		// Una cadena sin formas no cambia.
		{"ui.simple", nil, "Guardar"},
	}
	for _, cs := range casos {
		got := c.Traducir("es", cs.clave, cs.args...)
		if got != cs.esperado {
			t.Errorf("Traducir(%q, %v) = %q y se esperaba %q", cs.clave, cs.args, got, cs.esperado)
		}
		if strings.Contains(got, "|") {
			t.Errorf("Traducir(%q, %v) ha dejado la barra de las formas en la cadena: %q",
				cs.clave, cs.args, got)
		}
	}
}

// Y el descuadre entre formas: si el singular lleva %d y el plural no, la
// etiqueta sale sin el numero justo en el caso que menos se prueba.
//
// El segundo caso lo trajo el barrido de mutacion, y es el que de verdad prueba
// la comprobacion. En el primero, el singular ingles ya descuadra contra el
// castellano, asi que la comparacion ENTRE IDIOMAS lo caza sola y la
// comprobacion de coherencia entre formas se podia borrar entera sin poner nada
// rojo. En el segundo, los dos idiomas empiezan igual y el descuadre esta en la
// forma de dentro: eso solo lo ve la coherencia.
func TestUnDescuadreEntreFormasSeCaza(t *testing.T) {
	casos := map[string]string{
		"el descuadre se ve ya en la primera forma": `{"menu.aplican": "one applies|%d apply"}`,
		"el descuadre esta en la segunda forma":     `{"menu.aplican": "%d applies|apply"}`,
	}
	for nombre, en := range casos {
		t.Run(nombre, func(t *testing.T) {
			c, err := Cargar(fstest.MapFS{
				"es.json": &fstest.MapFile{Data: []byte(`{"menu.aplican": "%d aplica|%d aplican"}`)},
				"en.json": &fstest.MapFile{Data: []byte(en)},
			}, "es", "en")
			if err != nil {
				t.Fatal(err)
			}
			if d := c.Descuadres("en"); len(d) != 1 || d[0] != "menu.aplican" {
				t.Fatalf("Descuadres devolvio %v y tenia que cazar menu.aplican: las formas "+
					"de una clave tienen que ser intercambiables entre si y casar con las "+
					"del idioma de referencia", d)
			}
		})
	}

	// Control negativo: un plural bien puesto no puede salir como descuadre.
	c, err := Cargar(fstest.MapFS{
		"es.json": &fstest.MapFile{Data: []byte(`{"menu.aplican": "%d aplica|%d aplican"}`)},
		"en.json": &fstest.MapFile{Data: []byte(`{"menu.aplican": "%d applies|%d apply"}`)},
	}, "es", "en")
	if err != nil {
		t.Fatal(err)
	}
	if d := c.Descuadres("en"); len(d) != 0 {
		t.Fatalf("un plural correcto sale como descuadre: %v", d)
	}
}

func TestIdiomasDevuelveUnaCopiaYNoElInterior(t *testing.T) {
	c := nuevoParaTest(t)
	lista := c.Idiomas()
	lista[0] = "en"
	if c.Idiomas()[0] != "es" {
		t.Fatal("reordenar la lista devuelta por Idiomas le cambia el idioma por defecto " +
			"al catalogo de todo el proceso")
	}
}

// El catalogo se lee desde varias peticiones a la vez. Este test existe para
// que el detector de carreras del CI tenga donde morder.
func TestVariasGoroutinasTraducenALaVez(t *testing.T) {
	c := nuevoParaTest(t)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = c.Traducir("en", "pantalla.hoy.titulo")
				_ = c.Faltantes("en")
				_ = c.Idiomas()
			}
		}()
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// Ficheros corruptos
// ---------------------------------------------------------------------------

func TestUnCatalogoCorruptoNoCargaYDicePorQue(t *testing.T) {
	casos := []struct {
		nombre    string
		contenido string
		esperado  string // trozo que el error tiene que mencionar
	}{
		{"JSON roto", `{"ui.guardar": "Guardar",}`, "JSON ilegible"},
		{"no es un objeto", `["ui.guardar"]`, "objeto JSON"},
		{"valor anidado", `{"ui.guardar": {"corto": "Guardar"}}`, "no es una cadena"},
		{"valor numerico", `{"ui.guardar": 3}`, "no es una cadena"},
		{"clave repetida", `{"ui.guardar": "Guardar", "ui.guardar": "Grabar"}`, "dos veces"},
		{"vacio", `{}`, "ninguna cadena"},
		{"basura detras", `{"ui.guardar": "Guardar"} {"otra": "cosa"}`, "sobra contenido"},
	}
	for _, cs := range casos {
		t.Run(cs.nombre, func(t *testing.T) {
			_, err := Cargar(fstest.MapFS{
				"es.json": &fstest.MapFile{Data: []byte(cs.contenido)},
			}, "es")
			if err == nil {
				t.Fatalf("un catalogo con %s cargo sin protestar", cs.nombre)
			}
			if !strings.Contains(err.Error(), cs.esperado) {
				t.Fatalf("el error no menciona %q y dice: %v", cs.esperado, err)
			}
		})
	}

	t.Run("UTF-8 invalido", func(t *testing.T) {
		roto := []byte(`{"ui.guardar": "Guard` + "\xff" + `ar"}`)
		if _, err := Cargar(fstest.MapFS{
			"es.json": &fstest.MapFile{Data: roto},
		}, "es"); err == nil || !strings.Contains(err.Error(), "UTF-8") {
			t.Fatalf("un fichero que no es UTF-8 tiene que decirlo, y dio: %v", err)
		}
	})

	t.Run("fichero ausente", func(t *testing.T) {
		if _, err := Cargar(fstest.MapFS{
			"es.json": &fstest.MapFile{Data: []byte(`{"ui.guardar":"Guardar"}`)},
		}, "es", "en"); err == nil {
			t.Fatal("cargar un idioma sin su fichero de cadenas tiene que fallar, no dejar " +
				"un idioma vacio que devuelve claves en todas las pantallas")
		}
	})

	t.Run("sin idiomas", func(t *testing.T) {
		if _, err := Cargar(fstest.MapFS{}); err == nil {
			t.Fatal("un catalogo sin idiomas no puede renderizar nada y tiene que fallar")
		}
	})

	t.Run("sin sistema de ficheros", func(t *testing.T) {
		if _, err := Cargar(nil, "es"); err == nil {
			t.Fatal("cargar sin fs.FS tiene que dar un error que se pueda leer, no un panic")
		}
	})

	t.Run("idioma repetido", func(t *testing.T) {
		if _, err := Cargar(fstest.MapFS{
			"es.json": &fstest.MapFile{Data: []byte(`{"ui.guardar":"Guardar"}`)},
		}, "es", "es"); err == nil {
			t.Fatal("pedir el mismo idioma dos veces tiene que fallar: el segundo tapa al " +
				"primero en silencio")
		}
	})

	t.Run("codigo de idioma sucio", func(t *testing.T) {
		if _, err := Cargar(fstest.MapFS{
			"es.json": &fstest.MapFile{Data: []byte(`{"ui.guardar":"Guardar"}`)},
		}, "es-ES"); err == nil {
			t.Fatal("cargar 'es-ES' tiene que fallar: si el catalogo se registra con region " +
				"y se consulta sin ella, no lo encuentra nadie")
		}
	})
}

// Control negativo de todo lo de arriba: un catalogo bueno TIENE que cargar. Sin
// esto, un cargador que dijera que no a todo pasaria los siete casos anteriores.
func TestUnCatalogoBuenoCarga(t *testing.T) {
	c, err := Cargar(fstest.MapFS{
		"es.json": &fstest.MapFile{Data: []byte(`{"ui.guardar":"Guardar"}`)},
		"en.json": &fstest.MapFile{Data: []byte(`{"ui.guardar":"Save"}`)},
	}, "es", "en")
	if err != nil {
		t.Fatalf("un catalogo correcto no carga: el cargador dice que no a todo y las "+
			"comprobaciones de arriba no prueban nada: %v", err)
	}
	if got := c.Traducir("en", "ui.guardar"); got != "Save" {
		t.Fatalf("Traducir dio %q", got)
	}
}

func TestVerbosDeFormateo(t *testing.T) {
	casos := map[string][]string{
		"":                  nil,
		"Alcance":           nil,
		"Lo piden %s":       {"s"},
		"%s y %d":           {"s", "d"},
		"Cubierto al 80%":   {"?"},
		"100%% seguro":      nil,
		"%[2]s antes de %s": {"s", "s"},
		"%-10.4f":           {"f"},
	}
	for entrada, esperado := range casos {
		got := verbos(entrada)
		if strings.Join(got, ",") != strings.Join(esperado, ",") {
			t.Errorf("verbos(%q) = %v y se esperaba %v", entrada, got, esperado)
		}
	}
}
