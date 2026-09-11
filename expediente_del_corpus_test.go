package plazum

import (
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/aplicabilidad"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/expediente"
)

// EL CIRCULO CERRADO: SE EMITE UN EXPEDIENTE DEL CORPUS REAL Y SE VERIFICA.
//
// # Que convierte esta puerta en demostracion, y no en argumento
//
// `docs/modelo-de-amenaza.md` promete que **un tercero recalcula desde cero y
// obtiene exactamente lo mismo, o le dice donde no coincide**. Hasta el
// 11-09-2026 esa promesa se apoyaba en un solo fichero: el expediente de
// demostracion, montado a mano desde un escenario, con cuatro hitos, cinco
// aplicables y dos estados TECLEADOS. El verificador los recalculaba y comparaba
// contra lo que alguien habia ajustado hasta que cuadrara.
//
// Un motor con un solo consumidor no esta probado, esta de acuerdo consigo
// mismo. Y el precio ya se estaba cobrando: el verificador leia `fin_dia` donde
// el corpus escribe `fin_de_dia`, y **el fixture nunca escribio ninguna de las
// dos**, asi que ese `case` no lo ejerio ninguna entrada, nunca.
//
// Esta puerta cierra el circulo con las dos puntas ATADAS AL ARBOL: el corpus
// publicado entra por un lado y el verificador de verdad sale por el otro.
//
// # LO QUE ESTA PUERTA NO PRUEBA, dicho para que su verde no valga de mas
//
// No prueba que el motor calcule bien. El emisor deriva las reclamaciones
// llamando a `construirPlazo`, que es la misma funcion que usa `Verificar`, asi
// que si el motor se equivocara se equivocarian los dos igual. Lo que prueba es
// otra cosa y es la que faltaba:
//
//  1. Que el expediente que sale del corpus esta COMPLETO: nada de lo que el
//     verificador exige se queda sin emitir.
//  2. Que el VOCABULARIO CRUZA. Este es el que importa: con 191 obligaciones
//     declarando `cierre: "fin_de_dia"`, ese texto viaja ahora de verdad desde
//     un `paquete.json` hasta el `switch` del verificador.
//  3. Que los digests, las anclas y las vigencias cuadran sobre datos reales y
//     no sobre tres paquetes de mentira.
//
// Que el motor calcule bien lo sostienen los 808 dorados, que se derivan del
// TEXTO legal y no de la implementacion. Son dos afirmaciones distintas y hacen
// falta las dos.
func TestUnExpedienteEmitidoDelCorpusRealSeVerificaSinDiscrepancias(t *testing.T) {
	paqs, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatalf("el corpus publicado no carga: %v", err)
	}
	if len(paqs) < MinimoDeMarcos {
		t.Fatalf("solo %d paquetes cargados: esta puerta estaria emitiendo sobre casi nada",
			len(paqs))
	}

	comoEstaba := time.Date(2026, 9, 11, 9, 0, 0, 0, time.UTC)

	// LOS HECHOS DEL RELOJ SE DERIVAN DEL CORPUS, no se escriben aqui.
	//
	// Cada obligacion de plazo declara de que hecho cuelga (`disparador.hecho`),
	// y aqui se le da a ese hecho un instante sintetico. El instante es de
	// mentira —no hay una organizacion de verdad detras— y el NOMBRE del hecho
	// es real, que es lo que hace que el reloj se construya como se construiria
	// en produccion. Escribir la lista a mano seria la segunda lista de siempre:
	// el dia que una obligacion cambie de disparador, esto seguiria disparando el
	// viejo y la puerta seguiria verde.
	disparo := comoEstaba.Add(-48 * time.Hour)
	hechosDelReloj := map[string]map[string]time.Time{}
	conDisparador := 0
	for _, p := range paqs {
		for _, o := range p.Obligaciones {
			if o.Temporalidad == nil || o.Temporalidad.Primitiva != "plazo" {
				continue
			}
			h := o.Temporalidad.Disparador["hecho"]
			if h == "" {
				continue
			}
			conDisparador++
			hechosDelReloj[o.ID] = map[string]time.Time{h: disparo}
		}
	}
	if conDisparador == 0 {
		t.Fatal("ninguna obligacion de plazo declara disparador: el derivador de hechos se " +
			"ha roto y esta puerta emitiria un expediente sin un solo reloj")
	}

	e, res, err := expediente.Emitir(expediente.Emision{
		Paquetes:       paqs,
		Organizacion:   "organizacion de prueba",
		Alcance:        "el corpus publicado entero",
		HechosDelReloj: hechosDelReloj,
		Calendario: expediente.CalendarioDeclarado{
			ID: "utc-v1", Zona: "UTC", Ambito: "prueba", Fuente: "sin festivos",
		},
		ComoEstaba: comoEstaba,
	})
	if err != nil {
		t.Fatalf(`el corpus publicado NO SE PUEDE EMITIR: %v

  Esto es lo que un comprador tendria delante al pedir su expediente. Si el
  corpus no se puede emitir, el diferenciador del producto no existe para el:
  puede verificar el expediente de demostracion y no puede generar el suyo.`, err)
	}

	// EL SUELO, para que un emisor que devuelva un expediente vacio no pase.
	if res.Relojes == 0 || res.Reclamaciones == 0 {
		t.Fatalf("se emitio un expediente con %d relojes y %d reclamaciones: un expediente "+
			"sin fechas no verifica nada y su verde no significa nada", res.Relojes, res.Reclamaciones)
	}

	// Y LA VERIFICACION, con el verificador de verdad. El receptor aporta sus
	// anclas: aqui son las del contenido emitido, porque nadie ha manipulado
	// nada. Los ataques que cambian una cosa u otra tienen su suite en
	// nucleo/expediente/hostil_test.go.
	anclas := map[string]string{}
	for _, p := range e.Paquetes {
		anclas[p.URN] = p.Digest
	}
	inf := expediente.Verificar(e, expediente.ContextoReceptor{Anclas: anclas})

	// LAS DISCREPANCIAS DE CADENA SE SEPARAN DE LAS DEMAS, y eso no es aflojar.
	//
	// Este expediente se emite SIN cadena de custodia: no hay observaciones que
	// cifrar ni sello de tiempo que pedir, porque no hay una organizacion de
	// verdad detras. La cadena tiene su propia suite entera (ledger v2, lapidas,
	// checkpoints anclados) y no es lo que esta puerta existe para mirar.
	//
	// Lo que SI se exige a cero es todo lo demas: corpus, vigencia, aplicabilidad
	// y relojes. Que es donde vivia el defecto.
	var deLaCadena, delResto []string
	for _, d := range inf.Discrepancias {
		if d.Que == "cadena" {
			deLaCadena = append(deLaCadena, d.Que)
			continue
		}
		delResto = append(delResto, d.Que+": esperado "+d.Esperado+", obtenido "+d.Obtenido)
	}
	if len(delResto) > 0 {
		t.Errorf(`el expediente emitido del corpus real trae %d discrepancia(s) que no son de la cadena.

  %v

  El emisor y el verificador salen del MISMO arbol y calculan con las MISMAS
  funciones, asi que una discrepancia aqui no es que el motor se equivoque: es
  que el emisor no emite algo que el verificador exige, o que una palabra no
  cruza de un lado al otro. Lo segundo es lo que trajo esta puerta: el corpus
  escribe «fin_de_dia» 191 veces y el verificador leia «fin_dia».`,
			len(delResto), delResto)
	}

	// LOS DOS TRAMOS QUE ESTA PUERTA NO EJERCE, con su cardinal y su motivo.
	//
	// Un verde que no dice lo que no ha recorrido es la mitad de un verde. Aqui
	// hay dos huecos y los dos son por la misma causa: no hay una organizacion
	// de verdad detras.
	//
	//  1. LA APLICABILIDAD DERIVA CERO. Los programas del corpus se CARGAN y se
	//     EVALUAN —si una regla de un paquete no compilara, esto seria rojo—,
	//     pero sin hechos de la organizacion no hay nada de donde derivar, asi
	//     que el contraste declarado/derivado compara dos conjuntos vacios. Para
	//     llenarlo hace falta una entrevista contestada, que es exactamente lo
	//     que pone encima de la mesa el siguiente paso de D-27: el subcomando que
	//     lee un alcance publicado.
	//  2. LA CADENA VA VACIA, por lo dicho arriba.
	//
	// Se cuentan en vez de suponerse: el dia que el emisor reciba hechos, este
	// numero deja de ser cero y la puerta pasa a mirar un tramo mas.
	if len(e.Programas) == 0 {
		t.Error("no se cargo ni un programa de aplicabilidad del corpus: el tramo de reglas " +
			"no se esta ejerciendo, ni siquiera al cargar")
	}
	t.Logf("MEDIDO sobre el corpus publicado: %d paquetes, %d obligaciones, %d relojes de plazo "+
		"emitidos, %d reclamaciones recalculadas.\n"+
		"  Fuera, con su cardinal: %d obligacion(es) de plazo sin hecho que las dispare, "+
		"%d reloj(es) de una primitiva que RelojDeclarado no representa (%v).\n"+
		"  NO EJERCIDO, y por eso se cuenta: %d programa(s) de aplicabilidad cargados y "+
		"evaluados pero %d aplicables derivados, porque no se le dan hechos de ninguna "+
		"organizacion; y %d discrepancia(s) de cadena, que va vacia.",
		res.Paquetes, res.Obligaciones, res.Relojes, res.Reclamaciones,
		res.SinHechos, res.PrimitivasNoEmitidas, res.PrimitivasFuera,
		len(e.Programas), res.Aplicables, len(deLaCadena))
}

// EL CONTROL POSITIVO DEL CRUCE: la palabra del corpus llega al expediente.
//
// # Por que no basta con la puerta de arriba
//
// Porque aquella exige CERO discrepancias, y un emisor que no emitiera ni un
// reloj con cierre forzado tambien daria cero. El verde por vacio y el verde por
// acierto no se distinguen desde fuera, que es la familia entera de este
// repositorio.
//
// Aqui se afirma lo contrario en positivo: que `fin_de_dia` —la palabra exacta
// que el verificador NO entendia— sale de un `paquete.json`, viaja dentro de un
// `HitoDeclarado` y se vuelve a leer como el cierre al final del dia.
func TestLaPalabraDelCorpusLlegaEnteraAlExpediente(t *testing.T) {
	paqs, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatal(err)
	}

	// SE BUSCA EN EL ARBOL una obligacion de plazo cuyo paquete declare el
	// cierre al final del dia. No se nombra ninguna: nombrarla ataria esta
	// puerta a un identificador y la rompiria el dia que ese paquete cambie.
	var elegida *corpus.Obligacion
	for _, p := range paqs {
		for i := range p.Obligaciones {
			o := &p.Obligaciones[i]
			if o.Temporalidad != nil && o.Temporalidad.Primitiva == "plazo" &&
				o.Temporalidad.Regimen.Cierre == "fin_de_dia" &&
				o.Temporalidad.Disparador["hecho"] != "" {
				elegida = o
				break
			}
		}
		if elegida != nil {
			break
		}
	}
	if elegida == nil {
		t.Skip("no hay en el corpus ninguna obligacion de plazo con cierre `fin_de_dia` y " +
			"disparador: sin ella este control positivo no tiene que ejercer")
	}

	ahora := time.Date(2026, 9, 11, 9, 0, 0, 0, time.UTC)
	e, _, err := expediente.Emitir(expediente.Emision{
		Paquetes: paqs,
		HechosDelReloj: map[string]map[string]time.Time{
			elegida.ID: {elegida.Temporalidad.Disparador["hecho"]: ahora.Add(-24 * time.Hour)},
		},
		Calendario: expediente.CalendarioDeclarado{ID: "utc-v1", Zona: "UTC",
			Ambito: "prueba", Fuente: "sin festivos"},
		ComoEstaba: ahora,
		Hechos:     []aplicabilidad.Hecho{},
	})
	if err != nil {
		t.Fatal(err)
	}

	hay := false
	for _, r := range e.Relojes {
		if r.Obligacion != elegida.ID {
			continue
		}
		for _, h := range r.Hitos {
			hay = true
			if h.Cierre != "fin_de_dia" {
				t.Errorf(`el cierre llego al expediente como %q y el paquete escribe "fin_de_dia".

  Es la palabra exacta que el verificador no entendia. Si el emisor la cambia por
  el camino, el defecto vuelve por el otro lado: el expediente diria una cosa y
  el paquete otra, y la verificacion no tendria de que quejarse.`, h.Cierre)
			}
		}
	}
	if !hay {
		t.Fatalf("la obligacion %s tiene reloj de plazo en el corpus y no salio ningun hito "+
			"en el expediente: este control positivo no ha ejercido nada", elegida.ID)
	}
}
