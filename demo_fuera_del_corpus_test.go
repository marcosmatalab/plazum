package plazum

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	demoempresa "github.com/marcosmatalab/plazum/demo/demo-empresa"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// LA EMPRESA INVENTADA NO SE INSTALA EN LA CASA DE NADIE.
//
// # Que paso, y por que es un P0 y no una molestia
//
// Hasta el 10-09-2026 el paquete del demo vivia bajo `paquetes/`, y todo lo que
// hay bajo `paquetes/` viaja entero: el Dockerfile lo copia sin filtro a
// /datos/paquetes y la release lo empaqueta con `corpus --empaquetar paquetes`,
// cuyo unico filtro (entraEnElCorpus, en cmd/plazum/corpus.go) excluye el codigo
// Go y nada mas. O sea que una empresa sintetica con siete obligaciones y dos
// figuras inventadas se instalaba DENTRO del corpus real de quien se bajara el
// producto.
//
// Y no se queda en ruido en una pantalla: `plazum escalado` manda avisos a las
// figuras que el alcance declare, asi que el dano tiene forma de aviso de
// cumplimiento con destinatario que no existe, saliendo de una instalacion de
// verdad. Un producto de continuidad de cumplimiento que se inventa
// destinatarios pierde lo unico que vende.
//
// # Por que la puerta mira el URN EMPOTRADO y no un literal
//
// Porque un literal seria un identificador cableado (invariante 2) y ademas
// mentiria el dia que el demo cambiara de nombre. El URN se lee de los MISMOS
// BYTES que viajan dentro del binario (demoempresa.Ficheros), o sea de lo que
// `plazum demo` instala de verdad. Si alguien empotra otro paquete, esta puerta
// habla de ese.
//
// # Y por que ademas se leen el Dockerfile y el workflow de release
//
// Porque sin eso la afirmacion seria sobre `paquetes/`, y `paquetes/` solo
// importa mientras siga siendo lo que se publica. El camino barato de esta
// puerta no es devolver el demo a `paquetes/`, que es lo que salta a la vista:
// es anadir una linea al Dockerfile que copie tambien `demo/`, y entonces el
// demo volveria a instalarse con esta puerta en verde. Se cierra enumerando lo
// que cada uno de los dos publica y exigiendo que sea exactamente la raiz que
// aqui se comprueba.
//
// LO VIGILA: este mismo test, y se ha visto fallar. Tres mutaciones el
// 10-09-2026, cada una contra una mitad distinta, con su salida roja en el
// cuerpo del commit:
//
//	los datos del demo copiados a paquetes/   rojo en la afirmacion de arriba
//	`COPY demo /datos/paquetes` en el Docker   rojo en el subtest del Dockerfile
//	el demo fuera de corpusDelArbol            rojo en el linter de regimenes
//
// La tercera es la que importa entender: NO la caza este test, la caza
// TestTodoPaquetePublicadoDeclaraSuRegimenYSuAtribucion, porque el demo es el
// unico paquete del arbol con regimen `del-proyecto`. Es la prueba de que sacar
// el demo del linter no pasa desapercibido, que era el camino barato de todo
// este arreglo.
func TestElCorpusQueSeInstalaNoLlevaElPaqueteDelDemo(t *testing.T) {
	urnDelDemo := urnEmpotradoDelDemo(t)

	// CONTROL POSITIVO, primero: el URN que se busca tiene que existir en algun
	// sitio. Una puerta que busca un URN que ya no escribe nadie esta verde
	// mirando el vacio, que es como se aprueba esto sin arreglar nada.
	dem, err := corpus.Cargar(DirDelDemo)
	if err != nil {
		t.Fatalf("%s/ no carga: %v", DirDelDemo, err)
	}
	hallado := false
	for _, p := range dem {
		if p.URN == urnDelDemo {
			hallado = true
		}
	}
	if !hallado {
		t.Fatalf("el URN empotrado en el binario no aparece en %s/, asi que esta puerta "+
			"estaria comprobando que el corpus publicado no contiene algo que no existe.\n"+
			"  Arreglo: el paquete que se empotra y el que vive en %s/ tienen que ser el "+
			"mismo, que es la razon entera de que el fichero de empotrado viva ahi",
			DirDelDemo, DirDelDemo)
	}

	// LO QUE VIAJA AL CLIENTE.
	publicados, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatalf("el corpus publicado no carga: %v", err)
	}
	if len(publicados) < MinimoDeMarcos {
		t.Fatalf("el corpus publicado trae %d paquetes y el suelo es %d: esta puerta estaria "+
			"mirando medio corpus", len(publicados), MinimoDeMarcos)
	}
	for _, p := range publicados {
		if p.URN == urnDelDemo {
			t.Errorf("el paquete del demo esta en paquetes/, o sea DENTRO de la imagen y del "+
				"tar de la release, o sea dentro del corpus de quien se baje el producto.\n"+
				"  Lo que eso significa: `plazum escalado` puede mandar avisos a las figuras "+
				"que declare, y son figuras inventadas.\n"+
				"  Arreglo: el paquete del demo vive en %s/ y se distribuye empotrado en el "+
				"binario, no en el arbol de datos", DirDelDemo)
		}
	}

	// LOS DOS QUE PUBLICAN, enumerados: si manana uno de ellos publicara otra
	// raiz, todo lo de arriba seguiria siendo cierto y daria igual.
	t.Run("el Dockerfile copia paquetes y nada mas al corpus instalado", func(t *testing.T) {
		copias := raicesQueElDockerfileInstala(t)
		if len(copias) == 0 {
			t.Fatalf("el Dockerfile ya no copia ningun corpus a /datos/paquetes, o el patron " +
				"dejo de casar. Sin eso esta afirmacion no vigila nada")
		}
		for _, r := range copias {
			if r != "paquetes" {
				t.Errorf("el Dockerfile instala %q en el corpus de la imagen, y la unica raiz "+
					"que este test comprueba que no lleva el demo es paquetes/.\n"+
					"  O esa raiz entra tambien en la comprobacion de arriba, o el demo puede "+
					"volver a viajar con esta puerta en verde", r)
			}
		}
	})

	t.Run("la release empaqueta paquetes y nada mas", func(t *testing.T) {
		raices := raicesQueLaReleaseEmpaqueta(t)
		if len(raices) == 0 {
			t.Fatalf("el workflow de release ya no invoca `corpus --empaquetar`, o el patron " +
				"dejo de casar. Sin eso esta afirmacion no vigila nada")
		}
		for _, r := range raices {
			if r != "paquetes" {
				t.Errorf("la release empaqueta %q y este test solo comprueba paquetes/", r)
			}
		}
	})
}

// urnEmpotradoDelDemo lee el URN de los MISMOS bytes que viajan en el binario.
func urnEmpotradoDelDemo(t *testing.T) string {
	t.Helper()
	b, err := demoempresa.Ficheros.ReadFile("paquete.json")
	if err != nil {
		t.Fatalf("no puedo leer el paquete.json empotrado: %v", err)
	}
	var p struct {
		URN string `json:"urn"`
	}
	if err := json.Unmarshal(b, &p); err != nil {
		t.Fatalf("el paquete.json empotrado no es JSON: %v", err)
	}
	if strings.TrimSpace(p.URN) == "" {
		t.Fatalf("el paquete.json empotrado no declara urn")
	}
	return p.URN
}

// reCopiaDelCorpus caza la linea del Dockerfile que instala el corpus en la
// imagen. Se ancla en el DESTINO (/datos/paquetes) y no en el origen, que es lo
// que puede cambiar y lo que hay que leer.
var reCopiaDelCorpus = regexp.MustCompile(`(?m)^COPY[^\n]*?([A-Za-z0-9_./-]+) +/datos/paquetes\b`)

func raicesQueElDockerfileInstala(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, m := range reCopiaDelCorpus.FindAllStringSubmatch(leerDockerfile(t), -1) {
		out = append(out, m[1])
	}
	return out
}

// reEmpaquetado caza `corpus --empaquetar <raiz>` en el workflow de release.
var reEmpaquetado = regexp.MustCompile(`corpus --empaquetar +([A-Za-z0-9_./-]+)`)

func raicesQueLaReleaseEmpaqueta(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, m := range reEmpaquetado.FindAllStringSubmatch(leerRelease(t), -1) {
		out = append(out, m[1])
	}
	return out
}

// CONTROL NEGATIVO DE LOS DOS PATRONES.
//
// Los dos leen ficheros que no son Go y que se editan a mano, asi que su fallo
// probable no es acusar de mas: es dejar de casar y quedarse en verde sobre cero
// coincidencias. Arriba eso ya es un Fatal; aqui se comprueba ademas que cuando
// SI casan, leen la raiz correcta y no la palabra de al lado.
func TestLosPatronesQueLeenAQuienPublicaLeenLaRaizYNoElVecino(t *testing.T) {
	for _, c := range []struct {
		que    string
		re     *regexp.Regexp
		texto  string
		quiero string
	}{
		{
			que:    "COPY del Dockerfile",
			re:     reCopiaDelCorpus,
			texto:  "COPY --chown=65532:65532 paquetes /datos/paquetes\n",
			quiero: "paquetes",
		},
		{
			que:    "COPY con otra raiz",
			re:     reCopiaDelCorpus,
			texto:  "COPY --chown=65532:65532 demo /datos/paquetes\n",
			quiero: "demo",
		},
		{
			que:    "empaquetado de la release",
			re:     reEmpaquetado,
			texto:  "huella=$(go run ./cmd/plazum corpus --empaquetar paquetes --salida dist/x.tar.gz)",
			quiero: "paquetes",
		},
		{
			que:    "empaquetado con otra raiz",
			re:     reEmpaquetado,
			texto:  "go run ./cmd/plazum corpus --empaquetar demo --salida dist/x.tar.gz",
			quiero: "demo",
		},
	} {
		m := c.re.FindStringSubmatch(c.texto)
		if m == nil {
			t.Errorf("%s: el patron no casa nada en %q", c.que, c.texto)
			continue
		}
		if m[1] != c.quiero {
			t.Errorf("%s: el patron ha leido %q y la raiz es %q", c.que, m[1], c.quiero)
		}
	}
	// Y no casa donde no debe: una copia a otro destino no es la del corpus.
	if reCopiaDelCorpus.MatchString("COPY --chown=65532:65532 paquetes /datos/otra-cosa\n") {
		t.Errorf("el patron del Dockerfile caza copias que no van al corpus instalado")
	}
}
