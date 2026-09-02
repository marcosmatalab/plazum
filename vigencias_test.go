package plazum

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// LAS TRES FECHAS DE UNA NORMA, CONTRASTADAS CONTRA LA INSTANTANEA.
//
// EL AGUJERO QUE CIERRA, y salio dos veces con el invariante 10 ya escrito:
//
//	paquetes/ai-act  art. 111.4 fechado el 2026-07-24, que es la PUBLICACION del
//	                 omnibus en el DOUE. Su art. 4 dice "a los tres dias": el 27
//	paquetes/eni     el paquete entero fechado el 2010-01-29, que es la
//	                 PUBLICACION del RD 4/2010. Su disposicion final tercera dice
//	                 "el dia siguiente al de su publicacion": el 30
//
// Las dos son el mismo error y las dos las cometio quien habia escrito el
// invariante que lo prohibe. De ahi la leccion que este fichero convierte en
// puerta: **escribir una regla no la implanta**. La diferencia entre el
// invariante 3 y el 10 no era la importancia, era que el 3 tenia linter.
//
// POR QUE ESTE TEST PUEDE EXISTIR: las instantaneas del BOE traen las tres
// fechas COMO DATO, no como prosa (`fecha_disposicion`, `fecha_publicacion`,
// `fecha_vigencia`), asi que el contraste es mecanico y no necesita red. Las de
// Cellar no las traen, y eso se dice en vez de disimularse: ver el segundo test.
//
// Y EL MENSAJE DISTINGUE LOS DOS FALLOS, porque no son el mismo:
// una fecha que no cuadra con ninguna es un error cualquiera; una fecha que
// cuadra EXACTAMENTE con la de publicacion es la conflacion, y se dice por su
// nombre para que quien la lea sepa que buscar en el resto del paquete.

type fuenteInstantanea struct {
	Identificador    string `json:"identificador"`
	URNSugerido      string `json:"urn_sugerido"`
	FechaDisposicion string `json:"fecha_disposicion"`
	FechaPublicacion string `json:"fecha_publicacion"`
	FechaVigencia    string `json:"fecha_vigencia"`
}

type instantaneaMin struct {
	Fuente fuenteInstantanea `json:"fuente"`
	Huella string            `json:"huella"`
}

type paqueteMin struct {
	URN      string `json:"urn"`
	Vigencia struct {
		Desde string `json:"desde"`
	} `json:"vigencia"`
	Obligaciones []struct {
		ID       string `json:"id"`
		Vigencia struct {
			Desde string `json:"desde"`
		} `json:"vigencia"`
		Temporalidad *json.RawMessage `json:"temporalidad"`
	} `json:"obligaciones"`
}

func leerInstantaneas(t *testing.T) map[string]instantaneaMin {
	t.Helper()
	out := map[string]instantaneaMin{}
	dirs, err := filepath.Glob(filepath.Join("corpus-vigilancia", "*", "instantanea.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range dirs {
		b, err := os.ReadFile(f) // #nosec G304 -- ruta del propio repositorio, no de entrada
		if err != nil {
			t.Fatal(err)
		}
		var i instantaneaMin
		if err := json.Unmarshal(b, &i); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if i.Fuente.URNSugerido != "" {
			out[i.Fuente.URNSugerido] = i
		}
	}
	if len(out) < 10 {
		t.Fatalf("solo %d instantaneas con urn: se ha roto la lectura y este test dejaria de "+
			"probar nada", len(out))
	}
	return out
}

func leerPaquetes(t *testing.T) map[string]paqueteMin {
	t.Helper()
	out := map[string]paqueteMin{}
	fs, err := filepath.Glob(filepath.Join("paquetes", "*", "paquete.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		b, err := os.ReadFile(f) // #nosec G304 -- ruta del propio repositorio, no de entrada
		if err != nil {
			t.Fatal(err)
		}
		var p paqueteMin
		if err := json.Unmarshal(b, &p); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if p.URN != "" {
			out[p.URN] = p
		}
	}
	if len(out) < 30 {
		t.Fatalf("solo %d paquetes leidos: se ha roto la lectura", len(out))
	}
	return out
}

// NINGUNA VIGENCIA DEL CORPUS ES LA FECHA DE PUBLICACION DE SU NORMA.
func TestNingunaVigenciaEsLaFechaDePublicacionDeSuNorma(t *testing.T) {
	inst := leerInstantaneas(t)
	paqs := leerPaquetes(t)

	contrastados := 0
	for urn, p := range paqs {
		i, hay := inst[urn]
		if !hay || i.Fuente.FechaVigencia == "" {
			continue // sin instantanea con fechas: lo cuenta el otro test
		}
		contrastados++
		mirar := func(donde, fecha string) {
			if fecha == "" || fecha == i.Fuente.FechaVigencia {
				return
			}
			if fecha == i.Fuente.FechaPublicacion {
				t.Errorf("%s: %s dice %s, que es la FECHA DE PUBLICACION de la norma, no la de "+
					"entrada en vigor (%s).\n"+
					"  Es la conflacion del invariante 10 («de 8 de julio» y «publicado el 8 de "+
					"julio» no son lo mismo), y no es un error aislado: cuando aparece una, hay "+
					"que mirar las demas fechas del mismo paquete.\n"+
					"  Las tres, de la instantanea %s (huella %s): acto %s, publicacion %s, "+
					"vigor %s.",
					urn, donde, fecha, i.Fuente.FechaVigencia, i.Fuente.Identificador,
					i.Huella, i.Fuente.FechaDisposicion, i.Fuente.FechaPublicacion,
					i.Fuente.FechaVigencia)
				return
			}
			if fecha == i.Fuente.FechaDisposicion {
				t.Errorf("%s: %s dice %s, que es la fecha DEL ACTO, no la de entrada en vigor "+
					"(%s). Instantanea %s, huella %s.",
					urn, donde, fecha, i.Fuente.FechaVigencia, i.Fuente.Identificador, i.Huella)
				return
			}
			// Una fecha que no es ninguna de las tres puede ser legitima (un
			// apartado con vigencia diferida), asi que no se acusa: se exige
			// que lo diga. Eso lo vigila el linter del paquete, no este test.
		}
		mirar("la vigencia del paquete", p.Vigencia.Desde)
	}
	if contrastados < 4 {
		t.Fatalf("solo se han contrastado %d paquetes contra su instantanea: o se han movido las "+
			"instantaneas, o los urn han dejado de casar, y en los dos casos este test estaria "+
			"verde sin mirar nada", contrastados)
	}
}

// LO QUE ESTE CONTRASTE NO ALCANZA, DICHO CON NOMBRES Y NO EN GENERAL.
//
// Las instantaneas de Cellar no traen las tres fechas como dato: viven dentro
// del texto del articulo de entrada en vigor, en prosa. Asi que hoy el contraste
// mecanico solo alcanza a las normas del BOE.
//
// Este test no arregla eso: lo MIDE y lo deja escrito, para que «el corpus esta
// contrastado» no se lea como «todo el corpus esta contrastado». Un limite que
// se cuenta es un limite; un limite que se supone es un agujero.
func TestSeDiceCuantoAlcanzaElContrasteDeFechas(t *testing.T) {
	inst := leerInstantaneas(t)
	var conFechas, sinFechas []string
	for urn, i := range inst {
		if i.Fuente.FechaVigencia != "" {
			conFechas = append(conFechas, urn)
		} else {
			sinFechas = append(sinFechas, urn)
		}
	}
	sort.Strings(conFechas)
	sort.Strings(sinFechas)
	t.Logf("contraste mecanico de fechas: %d instantaneas lo permiten, %d no.\n"+
		"  CON las tres fechas como dato (BOE): %v\n"+
		"  SIN ellas (Cellar, viven en la prosa del articulo de entrada en vigor): %v",
		len(conFechas), len(sinFechas), conFechas, sinFechas)

	if len(conFechas) == 0 {
		t.Fatal("ninguna instantanea trae las tres fechas: el contraste del test anterior no " +
			"esta mirando nada")
	}
	// El numero de las que NO se pueden contrastar es el que tiene que bajar. Si
	// sube sin que nadie lo diga, el corpus esta creciendo por el lado ciego.
	const maximoSinContrastar = 10
	if len(sinFechas) > maximoSinContrastar {
		t.Errorf("hay %d instantaneas sin fechas contrastables y el maximo declarado es %d.\n"+
			"  No es un fallo del corpus, es que el lado ciego ha crecido: o el ingestor de "+
			"Cellar aprende a sacar las tres fechas, o se sube este numero A PROPOSITO y en el "+
			"mismo commit se dice por que.\n  Las de mas: %v", len(sinFechas), maximoSinContrastar, sinFechas)
	}
}
