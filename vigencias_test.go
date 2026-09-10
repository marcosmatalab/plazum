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

type aplicacionInstantanea struct {
	Desde   string `json:"desde"`
	Alcance string `json:"alcance"`
	Apoyo   string `json:"apoyo"`
}

type fuenteInstantanea struct {
	Identificador    string `json:"identificador"`
	URNSugerido      string `json:"urn_sugerido"`
	FechaDisposicion string `json:"fecha_disposicion"`
	FechaPublicacion string `json:"fecha_publicacion"`
	FechaVigencia    string `json:"fecha_vigencia"`
	// Aplicacion son los escalones que declara la fuente. Un acto entra en vigor
	// UNA vez y puede aplicarse por partes en fechas distintas, y lo que obliga
	// a un cliente es lo segundo: el CRA entra en vigor el 10-12-2024 y su art.
	// 14 no se aplica hasta el 11-09-2026.
	Aplicacion []aplicacionInstantanea `json:"aplicacion"`
}

// fechasDeclaradas son todas las fechas que la FUENTE pone sobre la mesa para
// una norma: la entrada en vigor y cada escalon de aplicacion. Es el conjunto
// contra el que se mide lo que escribe un paquete.
func (f fuenteInstantanea) fechasDeclaradas() map[string]bool {
	out := map[string]bool{}
	if f.FechaVigencia != "" {
		out[f.FechaVigencia] = true
	}
	for _, a := range f.Aplicacion {
		if a.Desde != "" {
			out[a.Desde] = true
		}
	}
	return out
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
		Cita     string `json:"cita"`
		Vigencia struct {
			Desde string `json:"desde"`
		} `json:"vigencia"`
		Temporalidad *json.RawMessage `json:"temporalidad"`
	} `json:"obligaciones"`
}

// actosQueNombraLaCita devuelve los URN de los actos con instantanea que la cita
// de una obligacion menciona por su ano y su numero.
//
// # POR QUE HACE FALTA, y es la leccion entera del 10-09-2026
//
// Una obligacion puede vivir en un paquete y sacar su texto de OTRO acto: en un
// consolidado, unos apartados son del acto base y otros los metio un
// modificativo. Sin esto, la unica fuente contra la que se puede contrastar una
// fecha es la del paquete, y entonces la pregunta que se contesta es «esta fecha
// es de LA FUENTE» cuando la que hay que contestar es «esta fecha es de LA
// FUENTE DE ESTA OBLIGACION». Son distintas exactamente en el caso que importa,
// y ese caso estaba en produccion: `eidas2.art24_3.publicacion_de_la_revocacion`
// llevaba el 20-05-2024 (vigor del Reglamento 2024/1183) siendo texto base del
// 910/2014, y la puerta decia que CASABA porque el URN del paquete nombraba al
// modificativo. Acerto la fecha del acto equivocado y se puso verde.
//
// # LAS DOS FORMAS, y por que las dos
//
// La UE escribe sus actos «910/2014» hasta 2014 y «2024/1183» desde 2015, asi
// que el mismo par (ano, numero) aparece en los dos ordenes segun la edad de la
// norma. Se buscan las dos porque acertar cual toca exigiria saber la regla de
// numeracion del ano, y equivocarse dejaria de ver el acto en silencio.
//
// # SU LIMITE, MEDIDO Y NO SUPUESTO
//
// Una cita que no nombra ningun acto con instantanea no dice nada, y entonces
// quien manda es el acto del paquete. Medido el 10-09-2026: 5 de 359
// obligaciones de paquetes con instantanea tienen la cita muda, o sea que el
// ancla de la cita alcanza al 98,6 %. Y ese respaldo es solido porque
// TestElURNDeUnPaqueteNombraElMismoActoQueSuIdentificador garantiza que el acto
// del paquete es el del texto; las dos puertas se sostienen la una a la otra.
func actosQueNombraLaCita(cita string, inst map[string]instantaneaMin) []string {
	var out []string
	for urn := range inst {
		t := strings.Split(urn, ":")
		if len(t) < 3 {
			continue
		}
		ano, num := t[len(t)-2], t[len(t)-1]
		if strings.Contains(cita, num+"/"+ano) || strings.Contains(cita, ano+"/"+num) {
			out = append(out, urn)
		}
	}
	sort.Strings(out)
	return out
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
	// El suelo es el mismo MinimoDeMarcos de paquetes_test.go: 20 desde que
	// `demo-empresa` se fue a demo/ el 10-09-2026.
	if len(out) < MinimoDeMarcos {
		t.Fatalf("solo %d paquetes leidos: se ha roto la lectura", len(out))
	}
	return out
}

// NINGUNA VIGENCIA DEL CORPUS ES LA FECHA DE PUBLICACION DE SU NORMA.
//
// # Lo que se amplio el 03-09-2026, y NACIO VERDE
//
// Hasta ese dia esto solo miraba la vigencia DEL PAQUETE, y solo en las seis
// normas del BOE. Ahora mira tambien la de cada OBLIGACION y alcanza a las
// diecisiete normas del corpus con instantanea, porque el ingestor aprendio a
// sacar las tres fechas de la ficha de la obra de Cellar.
//
// Y SE DICE EN VOZ ALTA: nacio VERDE sobre 336 fechas reales. Una puerta que
// nace verde sobre el corpus entero o vigila poco o llego tarde, y las dos cosas
// hay que saberlas. Aqui es lo segundo: las dos conflaciones que la trajeron
// (ai-act art. 111.4 y el paquete ens entero) ya estaban corregidas cuando el
// alcance crecio. Lo que este ensanche compra no es un hallazgo de hoy, es que
// las 300 fechas que antes no miraba nadie ya no puedan ponerse mal en silencio.
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
		// Y LA DE CADA OBLIGACION, que es donde vive el reloj de verdad. La
		// vigencia del paquete la mira todo el mundo; la de una obligacion
		// suelta no la mira nadie, y es la que decide si una fila sale en el
		// calendario de un cliente o no sale.
		for _, o := range p.Obligaciones {
			mirar("la vigencia de "+o.ID, o.Vigencia.Desde)
		}
	}
	if contrastados < 15 {
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
	//
	// BAJO DE 10 A 0 EL 03-09-2026, cuando el ingestor aprendio a sacar las tres
	// fechas de la ficha `notice=branch` de Cellar. El cero es el techo mas
	// exigente que hay: cualquier instantanea nueva que no traiga fechas pone
	// esto rojo el mismo dia, en vez de esconderse dentro de un margen.
	const maximoSinContrastar = 0
	if len(sinFechas) > maximoSinContrastar {
		t.Errorf("hay %d instantaneas sin fechas contrastables y el maximo declarado es %d.\n"+
			"  No es un fallo del corpus, es que el lado ciego ha crecido: o el ingestor de "+
			"Cellar aprende a sacar las tres fechas, o se sube este numero A PROPOSITO y en el "+
			"mismo commit se dice por que.\n  Las de mas: %v", len(sinFechas), maximoSinContrastar, sinFechas)
	}
}

// LO QUE ESCRIBE UN PAQUETE CONTRA LO QUE DECLARA LA FUENTE, CONTADO.
//
// # Por que este test CUENTA y no acusa
//
// La fuente declara, para cada norma, la entrada en vigor y cada escalon de
// aplicacion. Lo normal es que la fecha de una obligacion sea una de ellas. Pero
// NO todas las que no lo son estan mal, y hay dos razones legitimas y las dos
// estan en el corpus de hoy:
//
//	un acto MODIFICADO   la ficha de Cellar del acto base no recoge lo que le
//	                     movio un omnibus posterior. El Reglamento 2026/1744
//	                     movio dieciseis meses el capitulo III del AI Act, y la
//	                     ficha del 2024/1689 sigue diciendo la fecha original
//	una norma de segundo  las instrucciones tecnicas del ENS tienen su propia
//	nivel                 fecha, que no esta en la ficha del RD que las habilita
//
// Convertir esto en acusacion pondria rojo un paquete correcto, y una puerta que
// acusa en falso se acaba borrando. Asi que se MIDE, con techo: el numero puede
// no ser cero, pero no puede crecer sin que alguien lo diga.
//
// # EL ANCLA CAMBIO EL 10-09-2026, Y ES TODA LA LECCION DE ESE DIA
//
// Hasta ese dia, la fuente contra la que se median TODAS las fechas de un
// paquete era la del PAQUETE. Es decir: se contestaba «esta fecha es de la
// fuente» cuando la pregunta que hay que contestar es «esta fecha es de la
// fuente DE ESTA OBLIGACION». Las dos coinciden salvo en el caso que importa, y
// ese caso estaba en produccion:
//
//	eidas2.art24_3.publicacion_de_la_revocacion  vigencia 2024-05-20, heredada
//
// Es texto BASE del Reglamento 910/2014 (marca ▼B en la consolidada
// 02014R0910-20241018), y el 20-05-2024 es la entrada en vigor del Reglamento
// 2024/1183, que es OTRO acto. Casaba porque el urn del paquete nombraba al
// modificativo y este test emparejaba por urn. **Una guarda que confirma un dato
// malo es peor que no tenerla, porque quita las ganas de mirar**: la revision de
// las 336 fechas del corpus paso por encima de esta fila y la dio por buena.
//
// Ahora cada obligacion se mide contra los actos que NOMBRA SU CITA, y solo cae
// al acto del paquete cuando su cita no nombra ninguno (5 de 359, contado en el
// mensaje). Su propia cita ya lo delataba: era la unica de las tres de eidas2 que
// no decia «en la redaccion del Reglamento (UE) 2024/1183».
func TestSeCuentanLasVigenciasQueNoSonNingunaFechaDeSuPropiaFuente(t *testing.T) {
	inst := leerInstantaneas(t)
	paqs := leerPaquetes(t)

	casan, noCasan, citasMudas := 0, 0, 0
	// POR PAQUETE, y no solo el total: es lo que impide que un paquete entero se
	// esconda en la holgura de otro. Le paso a eni durante meses.
	porPaquete := map[string]int{}
	var deQuien []string
	for urn, p := range paqs {
		i, hay := inst[urn]
		if !hay {
			continue
		}
		declaradas := i.Fuente.fechasDeclaradas()
		if len(declaradas) == 0 {
			continue
		}
		// EL PAQUETE NO TIENE CITA: su fuente es su propio acto, y de eso
		// responde TestElURNDeUnPaqueteNombraElMismoActoQueSuIdentificador.
		sitios := []struct {
			donde, fecha string
			contra       map[string]bool
			actos        []string
		}{{"<el paquete>", p.Vigencia.Desde, declaradas, []string{urn}}}

		for _, o := range p.Obligaciones {
			// EL ANCLA ES LA CITA DE ESTA OBLIGACION, no el paquete. Si nombra
			// actos con instantanea, esos son su fuente, aunque no sean el del
			// paquete: en un consolidado, unos apartados son del acto base y
			// otros los metio un modificativo, y la fecha de cada uno sale del
			// suyo. La cita muda cae al acto del paquete, que es el unico
			// respaldo que queda, y por eso se cuenta aparte.
			nombrados := actosQueNombraLaCita(o.Cita, inst)
			contra, actos := declaradas, []string{urn}
			if len(nombrados) > 0 {
				contra, actos = map[string]bool{}, nombrados
				for _, a := range nombrados {
					for f := range inst[a].Fuente.fechasDeclaradas() {
						contra[f] = true
					}
				}
			} else if o.Vigencia.Desde != "" {
				citasMudas++
			}
			sitios = append(sitios, struct {
				donde, fecha string
				contra       map[string]bool
				actos        []string
			}{o.ID, o.Vigencia.Desde, contra, actos})
		}

		for _, s := range sitios {
			if s.fecha == "" {
				continue
			}
			if s.contra[s.fecha] {
				casan++
				continue
			}
			noCasan++
			porPaquete[urn]++
			deQuien = append(deQuien, fmt.Sprintf("%s %s = %s (contra %v)", urn, s.donde, s.fecha, s.actos))
		}
	}
	sort.Strings(deQuien)
	t.Logf("vigencias del corpus contra las fechas que declara SU PROPIA fuente: %d CASAN, %d no.\n"+
		"  Las que no casan no estan mal por no casar (un omnibus mueve fechas que la ficha del "+
		"acto base no recoge, y una instruccion tecnica tiene fecha propia), pero cada una "+
		"tiene que poder explicarse:\n  %v\n"+
		"  Y el limite del ancla, contado: %d obligacion(es) con la cita muda, o sea sin nombrar "+
		"ningun acto con instantanea. Esas caen al acto del paquete, que es de lo que responde "+
		"TestElURNDeUnPaqueteNombraElMismoActoQueSuIdentificador.",
		casan, noCasan, deQuien, citasMudas)

	if casan == 0 {
		t.Fatal("ninguna vigencia del corpus casa con una fecha de su fuente: o se ha roto el " +
			"emparejamiento por URN, o las instantaneas han dejado de traer fechas")
	}

	// EL TECHO NO SE ESCRIBE: SE DERIVA DE LAS RAZONES.
	//
	// # Por que dejo de ser un numero a mano, medido el 10-09-2026
	//
	// Porque el bloque que habia aqui tenia TRES cifras distintas en catorce
	// lineas y ninguna puerta las comparaba: el encabezado decia «17 de 336», la
	// lista de razones sumaba 20 y la constante decia 25. Y las tres convivian
	// porque el unico numero vigilado era el ultimo.
	//
	// Peor: faltaba un paquete ENTERO. Las cinco de `eni` (las Normas Tecnicas de
	// Interoperabilidad, con fecha propia) nunca aparecieron en la lista y
	// llevaban ahi desde antes, toleradas por cinco unidades de holgura que nadie
	// habia justificado. Con el techo puesto sobre el TOTAL, un paquete entero
	// puede esconderse dentro de la holgura de otro.
	//
	// # Lo que cambia, y es lo que impide que vuelva
	//
	// Las razones son DATOS, el techo es su suma, y la comprobacion es POR
	// PAQUETE y con igualdad exacta. Un paquete que gana una vigencia sin
	// explicarla ya no cabe en la holgura del vecino: rompe su propia fila.
	razones := map[string]struct {
		Cuantas int
		PorQue  string
	}{
		"urn:es:rd:2022:311": {12, "las instrucciones tecnicas del ENS (INES, " +
			"notificacion de incidentes, conformidad y auditoria) son resoluciones " +
			"con fecha propia de BOE, y no el RD 311/2022. Las tres de la ITS de " +
			"Auditoria (BOE-A-2018-4573) entraron el 10-09-2026 con vigencia " +
			"04-04-2018: acto 27-03-2018, publicacion 03-04-2018, en vigor al dia " +
			"siguiente, copiadas por separado de la API del BOE"},
		"urn:es:rd:2010:4": {5, "las Normas Tecnicas de Interoperabilidad del ENI " +
			"son resoluciones con fecha propia, igual que las ITS del ENS, y no el " +
			"RD 4/2010. Dos son de la Resolucion de 19-07-2011 (Documento " +
			"electronico, BOE-A-2011-13169, y Expediente electronico, " +
			"BOE-A-2011-13170), en vigor el 31-07-2011; tres son de la Resolucion " +
			"de 28-06-2012 (Politica de gestion de documentos e Intermediacion de " +
			"datos), en vigor el 27-07-2012. NUNCA habian estado en esta lista: " +
			"vivian dentro de la holgura del total, y por eso el techo ahora se " +
			"deriva y se comprueba por paquete"},
		"urn:es:rdl:2018:19": {4, "fechas diferidas del RDL 19/2018 que su ficha " +
			"no declara"},
		"urn:eu:dir:2022:2555": {2, "el 18-10-2024 del art. 41 (aplicacion de las " +
			"medidas nacionales), que Cellar no anota como hito de la Directiva"},
		"urn:eu:reg:2024:1689": {1, "art. 9.2, movido dieciseis meses por el " +
			"omnibus 2026/1744: el 02-12-2027 es una fecha CALCULADA sobre el " +
			"escalon del art. 113 y no la declara ninguna ficha, ni la del acto " +
			"base ni la del modificativo. BAJO DE 2 A 1 EL 10-09-2026 sin tocar " +
			"el corpus: el art. 111.4 lleva el 27-07-2026, que es la entrada en " +
			"vigor del propio omnibus, y desde que el ancla es la cita esa fila " +
			"casa contra el acto que su cita nombra en vez de figurar como " +
			"excepcion. Una excepcion menos que explicar a mano"},
	}

	// LA DE ai-act ES LA INTERESANTE y la que hay que mirar cuando una fila suba:
	// una fecha que la ficha del acto base no declara puede venir de un acto
	// modificador (bien) o de nadie (mal), y desde fuera se ven igual.
	maximoSinCasar := 0
	for _, r := range razones {
		maximoSinCasar += r.Cuantas
	}

	// LAS DOS DIRECCIONES, POR PAQUETE. La primera caza una vigencia nueva sin
	// explicar; la segunda caza una razon que se quedo vieja, que es la que
	// convierte esta lista en una lista de lo que hubo.
	for urn, r := range razones {
		if hay := porPaquete[urn]; hay != r.Cuantas {
			t.Errorf(`%s: %d vigencia(s) sin casar con la fuente y su razon declara %d.

  %s

  Si ha SUBIDO, hay una fecha que alguien escribio sin que la fuente la diga:
  puede estar bien (un acto modificado, una norma de segundo nivel) y puede ser
  la que borra una fila del calendario de un cliente sin que nada se ponga rojo.
  Si ha BAJADO, la razon se quedo vieja y sobra parte.

  La igualdad es exacta A PROPOSITO y por paquete: con un techo sobre el total,
  un paquete entero se esconde dentro de la holgura de otro, que es lo que le
  paso a eni durante meses.`, urn, hay, r.Cuantas, r.PorQue)
		}
	}
	for urn, hay := range porPaquete {
		if _, declarado := razones[urn]; !declarado {
			t.Errorf(`%s: %d vigencia(s) que no son ninguna fecha de su fuente, y este
  paquete no tiene razon declarada.

  Un paquete sin fila es exactamente como eni paso meses sin que nadie lo viera:
  cabia en la holgura del total. Escribe su razon, o corrige las fechas.`, urn, hay)
		}
	}

	if noCasan > maximoSinCasar {
		t.Errorf("hay %d vigencias que no son ninguna de las fechas que declara su fuente y el "+

			"techo es %d.\n  Cada una de mas es una fecha que alguien escribio sin que la fuente "+
			"la diga: puede estar bien (un acto modificado, una norma de segundo nivel) y puede "+
			"ser la que borra una fila del calendario de un cliente sin que nada se ponga rojo. "+
			"Se explica en el commit o se corrige", noCasan, maximoSinCasar)
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

	// EL TECHO. Sube solo A PROPOSITO y diciendo por que en el mismo commit.
	//
	// 196 de 230 el 03-09-2026 por la manana. La puerta nacio ROJA contra un
	// techo de 175 puesto a ojo, que es lo que pasa cuando se adivina un
	// cardinal en vez de medirlo: el numero real era peor. De esos 196, 39 no
	// podian ser contrastables NUNCA (los referenciales no tienen instantanea
	// posible por el invariante 3, y la demo es sintetica), asi que lo
	// recuperable eran 157.
	//
	// 39 de 230 esa misma tarde: se recuperaron los 157. No hizo falta ningun
	// parser de prosa. La ficha `notice=branch` de Cellar trae las tres fechas
	// COMO DATO al nivel de la obra, asi que el ingestor las saca de ahi y las
	// escribe en la instantanea; y las 154 que faltaban por instantanea muda
	// dejaron de faltar en once ejecuciones del ingestor. Las tres que quedaban
	// (eIDAS 2) no tenian instantanea y ahora la tienen.
	//
	// LOS 39 QUE QUEDAN NO BAJAN NUNCA, y por eso este techo ya no es una tarea
	// pendiente sino una frontera: son cinco referenciales (ISO 27001, ISO
	// 42001, PCI DSS, SOC 2, TISAX) que por el invariante 3 no pueden tener
	// instantanea, y la empresa de demostracion, que es sintetica.
	const maximoRelojesExpuestos = 39
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
	// Y LA DIRECCION CONTRARIA, QUE ES LA QUE NADIE RECORRE (invariante 7): una
	// instantanea que no la usa ningun paquete. No es un error, y por eso solo se
	// cuenta: hay dos hoy y las dos son actos MODIFICATIVOS sin paquete propio,
	// el Reglamento 2026/1744 (el omnibus del AI Act) y el Reglamento 2024/1183
	// (eIDAS 2). El segundo cambio de lado el 10-09-2026: hasta entonces la
	// huerfana era el 910/2014, el acto BASE, porque el paquete se llamaba por el
	// modificativo. Que ahora la huerfana sea el modificativo es la forma normal:
	// un consolidado se llama por el acto que consolida. Se cuenta porque el dia
	// que este numero crezca sin explicacion, lo que hay
	// detras es una norma ingerida y olvidada, que es un trabajo hecho que no
	// llega al calendario de nadie.
	usadas := map[string]bool{}
	for urn := range paqs {
		usadas[urn] = true
	}
	var huerfanas []string
	for urn := range inst {
		if !usadas[urn] {
			huerfanas = append(huerfanas, urn)
		}
	}
	sort.Strings(huerfanas)
	if !t.Failed() {
		t.Logf("MEDIDO: %d paquetes sin instantanea con su URN exacto, y ninguno la tiene "+
			"guardada bajo otro nombre.\n"+
			"  Y al reves: %d instantaneas que no usa ningun paquete, %v",
			mirados, len(huerfanas), huerfanas)
	}
}
