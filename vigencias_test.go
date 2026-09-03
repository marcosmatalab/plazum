package plazum

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// EL LADO CIEGO, MEDIDO EN LA UNIDAD QUE IMPORTA: RELOJES, NO INSTANTANEAS.
//
// # De donde sale, y es la mutacion mas cara de la campana del 03-09-2026
//
// El frente B movio QUINCE MESES la vigencia del art. 14.6 del CRA (la fecha
// mas cercana de todo el corpus: aplicable el 11-09-2026) y no se puso roja ni
// una puerta. Y si tiene efecto: el reloj DESAPARECE del calendario. O sea que
// el fallo no sale como error, sale como SILENCIO, que es la unica forma que un
// producto de cumplimiento no puede permitirse: nadie echa de menos una fila
// que nunca vio.
//
// # Por que este test MIDE y no verifica
//
// Verificar una fecha exige la fuente, y las instantaneas de Cellar no traen
// las tres fechas como dato: viven en la prosa del articulo de entrada en
// vigor. Eso ya estaba dicho aqui arriba, contado en INSTANTANEAS y topado a
// diez.
//
// Contar instantaneas era contar la unidad equivocada. Diez instantaneas mudas
// pueden ser un reloj o ciento cincuenta, y lo que esta expuesto son los
// RELOJES: cada uno con su vigencia, y cada vigencia mal puesta es una fila que
// aparece o desaparece del calendario de un cliente. La distancia entre los dos
// numeros es la distancia entre «hay un hueco» y «el hueco cubre la mitad del
// corpus».
//
// El numero es un TECHO: si sube, el corpus esta creciendo por el lado ciego y
// hay que decirlo en el mismo commit.
func TestSeCuentanLosRelojesQueNadiePuedeContrastar(t *testing.T) {
	inst := leerInstantaneas(t)
	paqs := leerPaquetes(t)

	// SE CASA POR EL URN DE LA NORMA, que es la identidad que comparten los dos
	// lados. Y las dos formas de no poder contrastar se cuentan APARTE, porque
	// no son la misma: «sin instantanea» es que no hay fuente ingerida, e
	// «instantanea muda» es que la hay y no dice las fechas. Meterlas en un solo
	// numero taparia cual de los dos arreglos hace falta.
	contrastables, mudas, sinInstantanea := 0, 0, 0
	var deQuien []string
	for urn, p := range paqs {
		relojes := 0
		for _, o := range p.Obligaciones {
			if o.Temporalidad != nil {
				relojes++
			}
		}
		if relojes == 0 {
			continue
		}
		i, hay := inst[urn]
		switch {
		case !hay:
			sinInstantanea += relojes
			deQuien = append(deQuien, fmt.Sprintf("%s (%d, sin instantanea)", urn, relojes))
		case i.Fuente.FechaVigencia == "":
			mudas += relojes
			deQuien = append(deQuien, fmt.Sprintf("%s (%d, instantanea muda)", urn, relojes))
		default:
			contrastables += relojes
		}
	}
	sort.Strings(deQuien)
	expuestos := mudas + sinInstantanea
	t.Logf("relojes del corpus por si su vigencia se puede contrastar: %d CONTRASTABLES, "+
		"%d EXPUESTOS (%d con instantanea muda, %d sin instantanea). Quienes: %v",
		contrastables, expuestos, mudas, sinInstantanea, deQuien)

	if contrastables == 0 {
		t.Fatal("ningun reloj del corpus tiene su vigencia contrastable: o se ha roto el " +
			"emparejamiento por URN, o este recorrido esta midiendo el vacio")
	}

	// EL TECHO. Baja cuando el ingestor de Cellar aprenda a sacar las tres
	// fechas; sube solo A PROPOSITO y diciendo por que en el mismo commit.
	// 196 de 230 el 03-09-2026. La puerta nacio ROJA contra un techo de 175
	// puesto a ojo, que es lo que pasa cuando se adivina un cardinal en vez de
	// medirlo: el numero real es peor. De esos 196, unos 39 no pueden ser
	// contrastables NUNCA (los referenciales no tienen instantanea posible por
	// el invariante 3, y la demo es sintetica), asi que lo que de verdad se
	// puede recuperar son unos 157, y todos por la misma via: que el ingestor
	// de Cellar saque las tres fechas de la prosa del articulo de entrada en
	// vigor.
	const maximoRelojesExpuestos = 196
	if expuestos > maximoRelojesExpuestos {
		t.Errorf("hay %d relojes cuya vigencia no puede contrastar nadie y el techo declarado "+
			"es %d. No es un fallo del corpus: es que el lado ciego ha crecido. Y lo que vive "+
			"ahi no falla ruidosamente, DESAPARECE del calendario. Arreglo: o el ingestor de "+
			"Cellar saca las tres fechas, o se sube este numero a proposito y se dice por que",
			expuestos, maximoRelojesExpuestos)
	}
}

// colaDelURN devuelve el «ano:numero» final de un URN, que es lo que identifica
// al ACTO. Sirve para distinguir dos cosas que se ven igual desde fuera: una
// norma que no tiene instantanea, y una que la tiene y esta guardada bajo otro
// nombre.
func colaDelURN(urn string) string {
	trozos := strings.Split(urn, ":")
	if len(trozos) < 2 {
		return ""
	}
	return trozos[len(trozos)-2] + ":" + trozos[len(trozos)-1]
}

// UN PAQUETE Y SU INSTANTANEA TIENEN QUE LLAMAR IGUAL A LA MISMA NORMA.
//
// # El fallo que lo trae, y son 48 relojes
//
// `nis2-tecnica` declara `urn:eu:reg-ejec:2024:2690` y su instantanea, ingerida
// de Cellar con su huella, esta guardada como `urn:eu:reg:2024:2690`. Es el
// mismo Reglamento de Ejecucion. Nadie lo noto porque el unico que cruzaba los
// dos lados era un recuento, y un recuento trata «no casa» como una CATEGORIA
// («sin instantanea») en vez de como un error.
//
// Es el invariante 7 en su forma mas pura: la comprobacion empareja dos
// conjuntos, y cuando la identidad se escribe distinta a cada lado el
// emparejamiento falla EN SILENCIO y el resultado se lee como un hueco legitimo.
// Aqui el precio fue que 48 relojes (el paquete mas denso del corpus) figuraran
// como «sin fuente ingerida» teniendo la fuente ingerida al lado.
//
// # Por que es un error y no otra categoria mas
//
// Porque tiene arreglo y las otras no. Un referencial no puede tener
// instantanea nunca (invariante 3) y un acto que no se ha ingerido hay que
// ingerirlo; esto es dos nombres para lo mismo, y se arregla escribiendo uno.
func TestUnPaqueteYSuInstantaneaLlamanIgualALaMismaNorma(t *testing.T) {
	inst := leerInstantaneas(t)
	paqs := leerPaquetes(t)

	porCola := map[string]string{}
	for urn := range inst {
		if c := colaDelURN(urn); c != "" {
			porCola[c] = urn
		}
	}

	mirados := 0
	for urn, p := range paqs {
		if _, hay := inst[urn]; hay {
			continue
		}
		mirados++
		otro, hay := porCola[colaDelURN(urn)]
		if !hay {
			continue // sin instantanea de verdad: es un hueco, no un descuadre
		}
		relojes := 0
		for _, o := range p.Obligaciones {
			if o.Temporalidad != nil {
				relojes++
			}
		}
		t.Errorf("el paquete dice %q y su instantanea esta guardada como %q: es la misma "+
			"norma con dos nombres, y el cruce falla EN SILENCIO.\n"+
			"  Consecuencia medida: sus %d relojes figuran como «sin fuente ingerida» "+
			"teniendo la fuente al lado, asi que ninguna comprobacion de fechas los "+
			"alcanza.\n"+
			"  Es el invariante 7: se empareja por una identidad, y cuando cada lado la "+
			"escribe distinta el resultado se lee como un hueco legitimo.\n"+
			"  Arreglo: que los dos escriban el mismo URN. No vale anadir una tabla de "+
			"equivalencias, que es una segunda identidad y el problema es tener dos.",
			urn, otro, relojes)
	}
	if mirados == 0 {
		t.Fatal("todos los paquetes tienen instantanea con su URN exacto, asi que este " +
			"recorrido no ha mirado ni un caso: o es verdad y esta puerta pasa a vigilar el " +
			"futuro, o el emparejamiento esta roto y da falsos aciertos")
	}
	if !t.Failed() {
		t.Logf("MEDIDO: %d paquetes sin instantanea con su URN exacto, y ninguno la tiene "+
			"guardada bajo otro nombre", mirados)
	}
}
