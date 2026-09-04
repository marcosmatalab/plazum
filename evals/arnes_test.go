package evals

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/marcosmatalab/plazum/adaptadores/busqueda"
	"github.com/marcosmatalab/plazum/adaptadores/ia"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/puertos"
)

const rutaCitas = "citas/dorados.json"

// -------------------------------------------------------------------------
// EL CONJUNTO DORADO.
// -------------------------------------------------------------------------

func TestElConjuntoDoradoDeCitasPasaEntero(t *testing.T) {
	c, err := Cargar(rutaCitas)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Ejecutar(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != len(c.Casos) {
		t.Fatalf("%d resultados para %d casos", len(res), len(c.Casos))
	}
	fallos := 0
	for _, r := range res {
		if r.Paso() {
			continue
		}
		fallos++
		t.Errorf(`caso %s: veredicto %q/%q, se esperaba %q/%q

  POR QUE ESTE CASO EXISTE: %s

  lo que devolvio el arnes: %v`,
			r.Caso.ID, r.Veredicto, r.Motivo, r.Caso.Veredicto, r.Caso.Motivo,
			r.Caso.Porque, r.Detalle)
	}
	if fallos == 0 {
		t.Logf("%d casos dorados de citas, todos con su veredicto", len(res))
	}
}

// UN CONJUNTO QUE NO RECORRE TODAS LAS RAMAS NO MIDE LO QUE DICE MEDIR.
//
// Un motivo de descarte que ningun caso ejercita es una rama del verificador
// que nadie prueba, y ademas es la que alguien podria borrar sin que se pusiera
// nada rojo. Se exige en los DOS sentidos: todos los motivos tienen caso, y
// todos los motivos que los casos usan existen.
func TestElConjuntoDoradoRecorreTodasLasRamasDelVerificador(t *testing.T) {
	c, err := Cargar(rutaCitas)
	if err != nil {
		t.Fatal(err)
	}
	usados := map[string]int{}
	aceptadas := 0
	for _, k := range c.Casos {
		if k.Veredicto == Aceptada {
			aceptadas++
			continue
		}
		usados[k.Motivo]++
	}
	for _, m := range MotivosConocidos() {
		if usados[m] == 0 {
			t.Errorf(`ningun caso dorado ejercita el motivo %q.

  Una rama del verificador que ningun caso recorre es una rama que nadie
  prueba, y el dia que alguien la borre esto seguira en verde. El conjunto
  dorado tiene que cubrir el vocabulario entero de descartes.`, m)
		}
	}
	// Y EL CONTROL POSITIVO DEL CONJUNTO ENTERO. Sin casos aceptados, todos
	// los descartes los pasaria un verificador que rechaza absolutamente todo,
	// y el eval publicaria una precision perfecta sobre un producto inservible.
	if aceptadas < 3 {
		t.Fatalf("solo %d casos aceptados en todo el conjunto. Un conjunto que solo pide "+
			"descartes lo pasa entero un verificador que dice que no a todo", aceptadas)
	}
	t.Logf("%d casos: %d aceptados y %d descartados, cubriendo los %d motivos",
		len(c.Casos), aceptadas, len(c.Casos)-aceptadas, len(MotivosConocidos()))
}

// LO QUE SE ENSENA DE UN CASO ACEPTADO SALE DE LA FUENTE.
func TestLoAceptadoDevuelveTextoDeLaFuenteYNoDelCaso(t *testing.T) {
	c, err := Cargar(rutaCitas)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Ejecutar(c)
	if err != nil {
		t.Fatal(err)
	}
	textos := map[string]string{}
	for _, f := range c.Fuentes {
		textos[f.ID] = f.Texto
	}
	comprobados := 0
	for _, r := range res {
		if r.Veredicto != Aceptada {
			continue
		}
		comprobados++
		fuente := textos[r.Caso.HashDe]
		if !strings.Contains(fuente, r.Cita) {
			t.Errorf("caso %s: lo que se ensenaria (%q) no es un trozo literal del texto "+
				"de la fuente. Si lo que acaba en pantalla no sale de la fuente, sale del "+
				"modelo", r.Caso.ID, r.Cita)
		}
	}
	if comprobados == 0 {
		t.Fatal("ningun caso aceptado que comprobar: este test estaria en verde sin mirar nada")
	}
}

// EL CARDINAL DEL LEEME SE DERIVA, NO SE ESCRIBE.
//
// `evals/README.md` dice cuantos casos tiene el conjunto de citas. Ese numero
// es una AFIRMACION ACOMPANADA: el dato tiene puerta (el conjunto se ejecuta
// entero) y la prosa que lo cita no la tiene, asi que el dia que alguien anada
// un caso el fichero pasa a mentir y nadie se entera. Es la familia que este
// repositorio lleva cuatro apariciones persiguiendo, y la peor de las cuatro
// fue justamente aquella en la que quien mentia era la explicacion.
//
// La regla mecanica es «todo motivo que cite un cardinal lo DERIVA». En un
// markdown no se puede derivar, asi que lo siguiente mas barato es esto: una
// puerta que lee el numero del texto y lo contrasta con el fichero.
func TestElCardinalDelLeemeCuadraConElConjunto(t *testing.T) {
	c, err := Cargar(rutaCitas)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("no puedo leer el LEEME (%v). Si se movio, esta puerta estaria "+
			"comprobando el vacio", err)
	}
	re := regexp.MustCompile(`\*\*(\d+) casos\*\*`)
	m := re.FindStringSubmatch(string(b))
	if m == nil {
		t.Fatalf("el LEEME no dice cuantos casos tiene el conjunto de citas.\n" +
			"  Se espera la forma **N casos**. Un LEEME sin el cardinal no miente, pero\n" +
			"  tampoco deja ponerle techo ni enterarse de si el conjunto ENCOGE, que es\n" +
			"  la mitad del motivo de tenerlo.")
	}
	dicho, err := strconv.Atoi(m[1])
	if err != nil {
		t.Fatal(err)
	}
	if dicho != len(c.Casos) {
		t.Errorf(`el LEEME dice %d casos y el conjunto tiene %d.

  El dato tiene puerta y la prosa que lo cita no la tenia, asi que se corregia
  solo el numero de dentro y el LEEME se quedaba describiendo un conjunto que
  ya no existe. Un dato falso se contrasta; una explicacion falsa se cree.

  Arreglo: actualiza el LEEME en el mismo commit que anade o quita casos.`,
			dicho, len(c.Casos))
	}
}

func TestDosMotivosNoPuedenApuntarAlMismoCentinela(t *testing.T) {
	// Si dos claves del vocabulario apuntaran al mismo error, una de las dos
	// seria inalcanzable: nombreDelMotivo devuelve la primera que case, y el
	// recorrido de un mapa en Go es aleatorio, asi que el conjunto dorado
	// fallaria una de cada dos ejecuciones y pareceria un problema del arnes.
	vistos := map[error]string{}
	for _, m := range MotivosConocidos() {
		e := motivos[m]
		if ya, hay := vistos[e]; hay {
			t.Errorf("los motivos %q y %q apuntan al mismo centinela", ya, m)
		}
		vistos[e] = m
	}
	if len(vistos) != len(motivos) {
		t.Fatalf("%d centinelas distintos para %d motivos", len(vistos), len(motivos))
	}
}

// -------------------------------------------------------------------------
// EL ARNES SE DEFIENDE DE SUS PROPIOS FICHEROS.
//
// Un conjunto dorado es un fichero que edita una persona con prisa. Si el arnes
// tomara valores por defecto ante un campo que falta, un caso mal escrito
// pasaria SIEMPRE y el eval publicaria una precision que no ha medido.
// -------------------------------------------------------------------------

func TestUnConjuntoMalFormadoNoSeCarga(t *testing.T) {
	const cabecera = `{"nombre":"n","porque":"p","fuentes":[{"id":"f","marco":"m",` +
		`"articulo":"a","clase":"transcrito","procedencia":"corpus",` +
		`"texto":"un texto largo y corriente que se puede citar sin problemas"}],"casos":[`

	casos := []struct {
		nombre    string
		caso      string
		centinela error
	}{
		{
			"sin veredicto",
			`{"id":"k","porque":"p","cita":"un texto largo y corriente","hash_de":"f"}`,
			ErrCasoSinVeredicto,
		},
		{
			"veredicto presente y no interpretable",
			`{"id":"k","porque":"p","cita":"x","hash_de":"f","veredicto":"casi"}`,
			ErrVeredictoIlegible,
		},
		{
			"descartada sin motivo",
			`{"id":"k","porque":"p","cita":"x","hash_de":"f","veredicto":"descartada"}`,
			ErrCasoSinMotivo,
		},
		{
			"motivo que no existe",
			`{"id":"k","porque":"p","cita":"x","hash_de":"f","veredicto":"descartada","motivo":"porque_si"}`,
			ErrMotivoDesconocido,
		},
		{
			"aceptada con motivo de descarte",
			`{"id":"k","porque":"p","cita":"x","hash_de":"f","veredicto":"aceptada","motivo":"cita_corta"}`,
			ErrMotivoDesconocido,
		},
		{
			"el hash dicho de dos formas",
			`{"id":"k","porque":"p","cita":"x","hash_de":"f","hash_literal":"","veredicto":"aceptada"}`,
			ErrHashAmbiguo,
		},
		{
			"sin decir de que fuente sale",
			`{"id":"k","porque":"p","cita":"x","veredicto":"aceptada"}`,
			ErrHashSinDeclarar,
		},
		{
			"nombra una fuente que no esta",
			`{"id":"k","porque":"p","cita":"x","hash_de":"no-existe","veredicto":"aceptada"}`,
			ErrFuenteDesconocida,
		},
		{
			"dos casos con el mismo id",
			`{"id":"k","porque":"p","cita":"x","hash_de":"f","veredicto":"aceptada"},` +
				`{"id":"k","porque":"p","cita":"y","hash_de":"f","veredicto":"aceptada"}`,
			ErrIDRepetido,
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			ruta := escribir(t, cabecera+c.caso+"]}")
			cj, err := Cargar(ruta)
			if !errors.Is(err, c.centinela) {
				t.Fatalf("carga (%v, %v), se esperaba %v", cj, err, c.centinela)
			}
		})
	}

	t.Run("un campo que el formato no conoce", func(t *testing.T) {
		// Sin DisallowUnknownFields, un "veredito" mal escrito cargaria el
		// caso con el veredicto vacio, y el error llegaria disfrazado de otra
		// cosa. Peor: "veredito":"aceptada" con un veredicto vacio caeria en
		// ErrCasoSinVeredicto, que dice la verdad y no dice DONDE.
		ruta := escribir(t, cabecera+
			`{"id":"k","porque":"p","cita":"x","hash_de":"f","veredito":"aceptada"}]}`)
		if _, err := Cargar(ruta); err == nil {
			t.Fatal("un campo desconocido se ha cargado en silencio")
		}
	})

	t.Run("conjunto sin casos", func(t *testing.T) {
		ruta := escribir(t, cabecera+"]}")
		if _, err := Cargar(ruta); !errors.Is(err, ErrConjuntoSinCasos) {
			t.Fatalf("un conjunto vacio se carga: %v", err)
		}
	})

	t.Run("procedencia que no se entiende", func(t *testing.T) {
		ruta := escribir(t, `{"nombre":"n","porque":"p","fuentes":[{"id":"f","marco":"m",`+
			`"articulo":"a","clase":"transcrito","procedencia":"quiza","texto":"un texto"}],`+
			`"casos":[{"id":"k","porque":"p","cita":"un texto largo y corriente de verdad",`+
			`"hash_de":"f","veredicto":"aceptada"}]}`)
		c, err := Cargar(ruta)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Ejecutar(c); !errors.Is(err, ErrProcedenciaIlegible) {
			t.Fatalf("una procedencia ilegible se ejecuta: %v", err)
		}
	})

	t.Run("clase que no se entiende", func(t *testing.T) {
		ruta := escribir(t, `{"nombre":"n","porque":"p","fuentes":[{"id":"f","marco":"m",`+
			`"articulo":"a","clase":"referncial","procedencia":"corpus","texto":"un texto"}],`+
			`"casos":[{"id":"k","porque":"p","cita":"un texto largo y corriente de verdad",`+
			`"hash_de":"f","veredicto":"aceptada"}]}`)
		c, err := Cargar(ruta)
		if err != nil {
			t.Fatal(err)
		}
		// "referncial" mal escrito NO puede caer en el valor cero de
		// corpus.Clase, que es `importado` y SI es citable: un caso que prueba
		// la frontera legal pasaria a probar lo contrario, en verde.
		if _, err := Ejecutar(c); !errors.Is(err, ErrClaseIlegible) {
			t.Fatalf("una clase mal escrita se ejecuta: %v", err)
		}
	})

	t.Run("un documento aportado con clase del corpus", func(t *testing.T) {
		ruta := escribir(t, `{"nombre":"n","porque":"p","fuentes":[{"id":"f","marco":"m",`+
			`"articulo":"a","clase":"transcrito","procedencia":"aportado","texto":"un texto"}],`+
			`"casos":[{"id":"k","porque":"p","cita":"un texto largo y corriente de verdad",`+
			`"hash_de":"f","veredicto":"aceptada"}]}`)
		c, err := Cargar(ruta)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Ejecutar(c); !errors.Is(err, ErrClaseIlegible) {
			t.Fatalf("un PDF del cliente con estrato legal se ejecuta: %v.\n"+
				"  Ponerle clase a un documento aportado seria decidir su frontera legal a "+
				"mano, y por ahi se cuela un texto ajeno con permiso escrito por quien lo "+
				"subio", err)
		}
	})

	// CONTROL POSITIVO: el conjunto bien escrito carga y se ejecuta. Sin esto,
	// los trece de arriba los pasaria igual un Cargar que devuelve error
	// siempre.
	t.Run("CONTROL POSITIVO: el conjunto bien escrito carga", func(t *testing.T) {
		ruta := escribir(t, cabecera+
			`{"id":"k","porque":"p","cita":"un texto largo y corriente que se puede citar",`+
			`"hash_de":"f","veredicto":"aceptada"}]}`)
		c, err := Cargar(ruta)
		if err != nil {
			t.Fatalf("un conjunto bien escrito no carga: %v", err)
		}
		res, err := Ejecutar(c)
		if err != nil {
			t.Fatal(err)
		}
		if len(res) != 1 || !res[0].Paso() {
			t.Fatalf("el caso bueno no pasa: %+v", res)
		}
	})
}

func escribir(t *testing.T, contenido string) string {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), "conjunto.json")
	if err := os.WriteFile(ruta, []byte(contenido), 0o600); err != nil {
		t.Fatal(err)
	}
	return ruta
}

// -------------------------------------------------------------------------
// EL BARRIDO SOBRE EL CORPUS REAL.
//
// POR QUE ESTE BARRIDO EXISTE ADEMAS DEL CONJUNTO DORADO, y por que es el que
// mas ha dado: una mutacion demuestra que la puerta caza un fallo QUE TU LE
// METISTE; un rojo sobre dato real demuestra que caza uno que nadie le metio.
//
// ESTE NACIO ROJO (04-09-2026). Con el hash calculado solo sobre el texto, las
// 528 obligaciones del corpus daban 495 hashes distintos: 29 hashes con mas de
// una obligacion detras y 33 obligaciones tapadas por otra, con el ganador
// decidido por el orden de un mapa. Ninguna mutacion lo habria encontrado,
// porque nadie sabia que estaba ahi. El arreglo esta en ia.HashDe, y este
// barrido es lo que lo vigila.
// -------------------------------------------------------------------------

func corpusReal(t *testing.T) []*corpus.Paquete {
	t.Helper()
	ps, err := corpus.Cargar("../paquetes")
	if err != nil {
		t.Fatalf("cargando el corpus real: %v", err)
	}
	if len(ps) < 20 {
		t.Fatalf("solo %d paquetes: el corpus no se esta encontrando y este barrido daria "+
			"verde sin mirar nada", len(ps))
	}
	return ps
}

func TestCadaUnidadCitableDelCorpusRealTieneUnHashSuyo(t *testing.T) {
	fs, err := ia.FuentesDelCorpus(corpusReal(t))
	if err != nil {
		t.Fatal(err)
	}
	porHash := map[string]string{}
	porTexto := map[string][]string{}
	for _, f := range fs {
		if ya, hay := porHash[f.Hash]; hay {
			t.Errorf(`%s y %s comparten hash.

  Con dos obligaciones bajo el mismo hash, cual resuelve una cita lo decide el
  orden de una lista, y el orden no lo firma nadie (invariante 7). La pantalla
  diria "el articulo X dice esto" nombrando otro articulo: no es una mentira
  sobre el contenido, es una mentira sobre la atribucion.`, ya, f.ID)
		}
		porHash[f.Hash] = f.ID
		porTexto[f.Texto] = append(porTexto[f.Texto], f.ID)
	}
	if len(porHash) != len(fs) {
		t.Fatalf("%d hashes distintos para %d fuentes", len(porHash), len(fs))
	}

	// Y EL CARDINAL DEL PROBLEMA QUE ESTO RESUELVE, medido y dicho en voz alta:
	// cuantas obligaciones del corpus comparten TEXTO con otra. No es un fallo
	// (dos normas con la misma estructura de clausulas comparten titulo corto
	// legitimamente) pero es el numero que hacia falta ver.
	choques, tapadas := 0, 0
	for _, ids := range porTexto {
		if len(ids) > 1 {
			choques++
			tapadas += len(ids) - 1
		}
	}
	t.Logf("corpus real: %d unidades citables, %d hashes distintos; %d textos repetidos "+
		"que tapaban a %d obligaciones cuando el hash era solo del texto",
		len(fs), len(porHash), choques, tapadas)
}

func TestLaPuertaAntialucinacionSobreElCorpusReal(t *testing.T) {
	fs, err := ia.FuentesDelCorpus(corpusReal(t))
	if err != nil {
		t.Fatal(err)
	}
	v, err := ia.Estricto(fs)
	if err != nil {
		t.Fatal(err)
	}

	citables, noCitables, cortas := 0, 0, 0
	for _, f := range fs {
		// Una cita LITERAL del principio del propio texto de la fuente.
		trozo := []rune(f.Texto)
		if len(trozo) > 60 {
			trozo = trozo[:60]
		}
		cita := strings.TrimSpace(string(trozo))
		if len([]rune(strings.Join(strings.Fields(cita), " "))) < ia.MinimoCitaPorDefecto {
			cortas++
			continue
		}
		p := puertos.Propuesta{Cita: cita, HashFuente: f.Hash, Modelo: "sin modelo"}
		_, err := v.Verificar(p)

		if f.Citable {
			citables++
			if err != nil {
				t.Errorf(`%s (%s, clase %s): una cita LITERAL de su propio texto se descarta.

  motivo: %v

  Es la rama de descargo, y si falla el producto no ensena ni una cita buena.`,
					f.ID, f.Marco, f.Clase, err)
			}
			continue
		}
		noCitables++
		if !errors.Is(err, ia.ErrSinTextoCitable) {
			t.Errorf(`%s (%s, clase %s): su texto se puede citar, y no deberia.

  devuelto: %v

  De un marco de estrato %s no distribuimos texto normativo, asi que ensenar
  ese campo como cita seria decir que la norma dice eso. Invariante 3 y
  docs/ia.md seccion 3.`, f.ID, f.Marco, f.Clase, err, f.Clase)
		}
	}

	// LOS DOS SUELOS, porque cero acusaciones y cero descargos se leen igual
	// que "todo en orden".
	if citables < 300 {
		t.Fatalf("solo %d unidades citables recorridas: el barrido no esta llegando al "+
			"corpus", citables)
	}
	if noCitables < 100 {
		t.Fatalf("solo %d unidades no citables recorridas: la rama de la frontera legal "+
			"apenas se prueba", noCitables)
	}
	t.Logf("corpus real: %d citas literales aceptadas, %d rechazadas por estrato, "+
		"%d saltadas por texto demasiado corto", citables, noCitables, cortas)
}

// LA CITA REAL COLGADA DEL HASH EQUIVOCADO, sobre el corpus entero.
//
// Es el invariante 7 barrido: el texto es literal, real y del corpus, y va
// contra el hash de OTRA obligacion. Si el emparejamiento fuera por posicion o
// por un identificador escrito aparte, esto pasaria.
func TestUnaCitaRealDeOtroArticuloNoPasaSobreElCorpusReal(t *testing.T) {
	fs, err := ia.FuentesDelCorpus(corpusReal(t))
	if err != nil {
		t.Fatal(err)
	}
	v, err := ia.Estricto(fs)
	if err != nil {
		t.Fatal(err)
	}
	var citables []ia.Fuente
	for _, f := range fs {
		if f.Citable {
			citables = append(citables, f)
		}
	}
	probados, saltados := 0, 0
	for i, f := range citables {
		otra := citables[(i+1)%len(citables)]
		trozo := []rune(otra.Texto)
		if len(trozo) > 60 {
			trozo = trozo[:60]
		}
		cita := strings.Join(strings.Fields(string(trozo)), " ")
		if len([]rune(cita)) < ia.MinimoCitaPorDefecto {
			saltados++
			continue
		}
		// Si los dos textos empiezan igual, la cita SI esta en la fuente y el
		// caso no dice nada. Se salta y se cuenta.
		if strings.Contains(strings.Join(strings.Fields(f.Texto), " "), cita) {
			saltados++
			continue
		}
		probados++
		if _, err := v.Verificar(puertos.Propuesta{Cita: cita, HashFuente: f.Hash}); err == nil {
			t.Errorf("la cita literal de %s ha pasado colgada del hash de %s", otra.ID, f.ID)
		}
	}
	if probados < 300 {
		t.Fatalf("solo %d parejas probadas de %d citables (%d saltadas): el barrido no "+
			"esta cubriendo el corpus", probados, len(citables), saltados)
	}
	t.Logf("%d parejas cruzadas del corpus real, ninguna pasa; %d saltadas por texto "+
		"comun o demasiado corto", probados, saltados)
}

// LO QUE SE ENSENA ES LO QUE SE HA VERIFICADO, BARRIDO SOBRE EL CORPUS REAL.
//
// POR QUE ESTE TEST EXISTE, y llego de un aviso del frente de corpus: alli una
// mutacion sobrevivio porque un campo se comprobaba POR LONGITUD y no por
// contenido, asi que podia citar un plazo que la norma no dice con todos los
// dorados en verde. La forma general es: una comprobacion que mira la FORMA
// deja pasar lo que una que mira el CONTENIDO no dejaria.
//
// Aplicada aqui, la pregunta es: el verificador comprueba LA CITA, pero lo que
// acaba en pantalla es el TROZO DE LA FUENTE recortado por dos indices que
// salen del mapa de normalizacion. Un mapa desplazado una runa daria un recorte
// que empieza media palabra antes, la cita habria casado igual, y el test que
// compara `origen[Desde():Hasta()]` con `Cita()` seguiria verde porque los dos
// salen de los mismos indices. Es circular.
//
// Esto lo rompe: se contrasta lo devuelto contra la cita que se ENVIO, que es
// el unico dato que no viene de los indices. Sobre las 328 unidades citables
// del corpus real y sobre sus formas raras (saltos de linea, sangria, espacios
// multiples), que son justo las que mueven el mapa.
func TestLoQueSeEnsenaEsLoQueSeHaVerificadoSobreElCorpusReal(t *testing.T) {
	fs, err := ia.FuentesDelCorpus(corpusReal(t))
	if err != nil {
		t.Fatal(err)
	}
	v, err := ia.Estricto(fs)
	if err != nil {
		t.Fatal(err)
	}
	colapsar := func(s string) string { return strings.Join(strings.Fields(s), " ") }

	comprobadas, saltadas := 0, 0
	for _, f := range fs {
		if !f.Citable {
			continue
		}
		runas := []rune(f.Texto)
		// Tres recortes por fuente: el principio, el medio y el final. El del
		// medio y el del final son los que caen despues de los saltos de linea
		// y la sangria, o sea donde un mapa desplazado se nota.
		for _, corte := range [][2]int{{0, 60}, {len(runas) / 3, len(runas)/3 + 60}, {maxCero(len(runas) - 60), len(runas)}} {
			desde, hasta := corte[0], corte[1]
			if hasta > len(runas) {
				hasta = len(runas)
			}
			if desde >= hasta {
				saltadas++
				continue
			}
			enviada := colapsar(string(runas[desde:hasta]))
			if len([]rune(enviada)) < ia.MinimoCitaPorDefecto {
				saltadas++
				continue
			}
			ok, err := v.Verificar(puertos.Propuesta{Cita: enviada, HashFuente: f.Hash})
			if err != nil {
				t.Errorf("%s: un recorte literal de su propio texto no verifica: %v", f.ID, err)
				continue
			}
			comprobadas++
			if devuelto := colapsar(ok.Cita()); devuelto != enviada {
				t.Errorf(`%s: lo que se ensenaria NO es lo que se ha verificado.

  se verifico: %q
  se ensena:   %q

  El verificador comprueba la cita y la pantalla ensena el trozo de la fuente
  recortado por dos indices. Si los indices se desplazan, la cita casa igual y
  lo que sale es otro texto, con la cara de una cita comprobada.`,
					f.ID, enviada, devuelto)
			}
		}
	}
	if comprobadas < 600 {
		t.Fatalf("solo %d recortes comprobados (%d saltados): el barrido no esta "+
			"cubriendo el corpus", comprobadas, saltadas)
	}
	t.Logf("%d recortes del corpus real, todos ensenan exactamente lo verificado; "+
		"%d saltados por ser demasiado cortos", comprobadas, saltadas)
}

func maxCero(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

// LA MEDIDA QUE SOSTIENE LA DECISION DE NO NORMALIZAR UNICODE.
//
// El verificador no pliega composicion Unicode, y eso solo es inocuo mientras
// el corpus lleve los acentos precompuestos. Es una afirmacion contable, asi
// que lleva puerta: el dia que entre corpus con marcas combinantes, esto se
// pone rojo y hay que DECIDIR, no descubrirlo cuando un cliente no encuentre su
// cita.
func TestElCorpusRealNoTraeMarcasCombinantes(t *testing.T) {
	fs, err := ia.FuentesDelCorpus(corpusReal(t))
	if err != nil {
		t.Fatal(err)
	}
	runas, marcas := 0, 0
	var primeras []string
	for _, f := range fs {
		if !f.Citable {
			continue
		}
		for _, r := range f.Texto {
			runas++
			if unicode.Is(unicode.Mn, r) {
				marcas++
				if len(primeras) < 5 {
					primeras = append(primeras, f.ID)
				}
			}
		}
	}
	if runas < 100000 {
		t.Fatalf("solo %d runas recorridas: el barrido no esta llegando al corpus", runas)
	}
	if marcas > 0 {
		t.Errorf(`el corpus citable trae %d marcas combinantes en %d runas (primeras: %v).

  El verificador de citas compara caracter a caracter y NO pliega composicion
  Unicode, a proposito. Mientras el corpus lleve los acentos precompuestos eso
  no cuesta nada; con texto descompuesto dentro, una cita correcta escrita en
  la forma normal deja de casar y el cliente ve descartes que no entiende.

  Hay que decidir: o el corpus se normaliza al entrar, o el verificador pliega
  (y entonces entra golang.org/x/text en DEPENDENCIAS.md). Lo que no vale es
  enterarse en produccion.`, marcas, runas, primeras)
	}
	t.Logf("%d runas citables del corpus real, %d marcas combinantes", runas, marcas)
}

// -------------------------------------------------------------------------
// LA BUSQUEDA SOBRE EL CORPUS REAL.
// -------------------------------------------------------------------------

func TestElIndiceDeBusquedaSoloLlevaLoQueSePuedeCitar(t *testing.T) {
	fs, err := ia.FuentesDelCorpus(corpusReal(t))
	if err != nil {
		t.Fatal(err)
	}
	docs := ia.Documentos(fs)
	citables := 0
	permitidos := map[string]bool{}
	for _, f := range fs {
		if f.Citable {
			citables++
			permitidos[f.ID] = true
		}
	}
	if len(docs) != citables {
		t.Fatalf("%d documentos indexables para %d unidades citables", len(docs), citables)
	}
	for _, d := range docs {
		if !permitidos[d.ID] {
			t.Errorf(`%s ha entrado al indice y no se puede citar.

  Lo que entra al indice acaba en el contexto del modelo, y lo que esta en el
  contexto sale parafraseado en alguna propuesta. El texto de un catalogo
  privativo no puede salir del proceso, y la forma de garantizarlo es que no
  entre.`, d.ID)
		}
	}

	i, err := busqueda.Nuevo(docs)
	if err != nil {
		t.Fatalf("el indice no se construye sobre el corpus real: %v", err)
	}
	if i.Documentos() != len(docs) {
		t.Fatalf("el indice tiene %d documentos y se le dieron %d", i.Documentos(), len(docs))
	}

	// Y QUE ENCUENTRA ALGO. Un indice que se construye y no encuentra nada da
	// exactamente el mismo verde que uno que funciona.
	res, err := i.Buscar("notificacion a la autoridad", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) == 0 {
		t.Fatal("una consulta corriente sobre el corpus entero no devuelve nada")
	}
	// TODO RESULTADO TIENE QUE TRAER SU HASH, o no se puede citar lo que se
	// encuentra y la busqueda no sirve para lo que se construyo.
	for _, r := range res {
		if r.Hash == "" {
			t.Errorf("el resultado %s viene sin hash", r.ID)
		}
	}
	t.Logf("%d documentos indexados del corpus real; la consulta de prueba devuelve %d "+
		"resultados, el primero %s (%s)", i.Documentos(), len(res), res[0].ID, res[0].Marco)
}

// EL PUENTE ENTERO, DE LA BUSQUEDA A LA CITA, SOBRE EL CORPUS REAL.
//
// Es el camino que va a recorrer el producto: se busca, se coge un resultado,
// se cita un trozo literal de su texto y se verifica contra el hash que el
// propio resultado trajo. Si este puente se rompe, cada mitad puede seguir en
// verde por su cuenta.
func TestElPuenteDeLaBusquedaALaCitaAguantaSobreElCorpusReal(t *testing.T) {
	fs, err := ia.FuentesDelCorpus(corpusReal(t))
	if err != nil {
		t.Fatal(err)
	}
	v, err := ia.Estricto(fs)
	if err != nil {
		t.Fatal(err)
	}
	i, err := busqueda.Nuevo(ia.Documentos(fs))
	if err != nil {
		t.Fatal(err)
	}
	consultas := []string{
		"notificacion a la autoridad de control",
		"registro de actividades de tratamiento",
		"auditoria interna del sistema de gestion",
		"medidas tecnicas y organizativas",
	}
	verificadas := 0
	for _, q := range consultas {
		res, err := i.Buscar(q, 3)
		if err != nil {
			t.Fatalf("consulta %q: %v", q, err)
		}
		for _, r := range res {
			trozo := []rune(r.Texto)
			if len(trozo) > 80 {
				trozo = trozo[:80]
			}
			cita := strings.Join(strings.Fields(string(trozo)), " ")
			if len([]rune(cita)) < ia.MinimoCitaPorDefecto {
				continue
			}
			ok, err := v.Verificar(puertos.Propuesta{Cita: cita, HashFuente: r.Hash})
			if err != nil {
				t.Errorf("el resultado %s de la consulta %q no verifica con su propio "+
					"hash: %v", r.ID, q, err)
				continue
			}
			if ok.Fuente() != r.ID {
				t.Errorf("la cita verificada apunta a %s y el resultado era %s",
					ok.Fuente(), r.ID)
			}
			verificadas++
		}
	}
	if verificadas < 8 {
		t.Fatalf("solo %d citas verificadas por el puente entero: no esta recorriendo "+
			"nada", verificadas)
	}
	t.Logf("%d citas del corpus real recorren el puente entero, de la consulta al hash "+
		"verificado", verificadas)
}

// LA BUSQUEDA Y EL VERIFICADOR NO SON IA, Y TIENEN QUE FUNCIONAR CON LA IA
// APAGADA. Es la mitad que se olvida del invariante 9.
func TestElPuenteEnteroFuncionaConLaIAApagada(t *testing.T) {
	t.Setenv(ia.Variable, "1")
	c, err := Cargar(rutaCitas)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Ejecutar(c)
	if err != nil {
		t.Fatalf("con la IA apagada el arnes de evals deja de ejecutarse: %v", err)
	}
	for _, r := range res {
		if !r.Paso() {
			t.Fatalf("con la IA apagada el caso %s cambia de resultado: %v", r.Caso.ID, r.Detalle)
		}
	}
}
