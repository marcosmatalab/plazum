package corpus

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/estado"
	"github.com/marcosmatalab/plazum/nucleo/historia"
	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// LO QUE LA ETAPA 4 NECESITA DEL MOTOR, medido antes de escribir una linea del
// objeto Incidente.
//
// Dos consumidores piden LA MISMA forma y ninguno de los dos existe todavia:
//
//	(a) El art. 81.12 de MiCA: revisar la evaluacion de idoneidad "PARA CADA
//	    CLIENTE, al menos cada dos anos despues de la evaluacion inicial". Una
//	    cadencia de 24 meses por cliente, anclada en la fecha inicial DE CADA
//	    UNO. Esta escrito en el LEEME de mica como lo unico que ese paquete no
//	    hace, y este es el motivo.
//
//	(b) El art. 33 del RGPD y las otras dieciocho notificatorias de incidente
//	    del corpus: las 72 horas corren POR INCIDENTE, desde el primer
//	    conocimiento de ESE incidente. Una organizacion con tres incidentes
//	    abiertos tiene tres relojes de la MISMA obligacion con tres fechas
//	    distintas.
//
// Son el mismo reloj: N instancias de UNA obligacion para UN sujeto, cada una
// anclada en su propio hecho. Aqui se mide si el motor lo hace y, si no, DONDE
// exactamente se rompe.
//
// POR QUE SE MIDE ANTES Y NO SE RAZONA. Ya ha pagado dos veces: la medicion de
// la familia A (nucleo/ventana/familia_a_test.go) pidio los hitos encadenados y
// el `Tope`, y descarto `Secuencia` por tener el arranque cableado en la
// estructura. Escribir treinta y tres relojes contra una primitiva que no hace
// lo que hace falta es escribirlos dos veces.
//
// EL RESULTADO, adelantado porque es la conclusion de las cuatro mediciones:
//
//	LAS PRIMITIVAS YA VALEN Y NO HACE FALTA UNA NUEVA. Lo que falta esta
//	encima de ellas, es UNA SOLA COSA, y es la IDENTIDAD DEL OBJETO.
//
// Las dos puntas de la cadena ya la admiten: `historia` porque su clave es una
// cadena que elige quien llama, y las primitivas de `ventana` porque toman el
// ancla en cada llamada. Los dos eslabones de en medio no: el HECHO (una clave
// por nombre, y el nombre lo escribe el paquete, que no conoce los objetos del
// cliente) y el HITO (uno por obligacion, y el motor ya PROHIBE que se repita).
//
// De ahi sale la unica decision de diseno que el objeto Incidente no puede
// aplazar: el incidente trae su identidad desde su primera linea, y esa
// identidad viaja por la cadena entera. Un paquete nombra la FORMA del hecho
// ("conocimiento_del_incidente"); el objeto aporta la INSTANCIA.

// obligacionPorObjeto es el reloj del art. 33 escrito como lo escribiria hoy un
// paquete: un plazo con UN disparador nombrado. No cablea ninguna norma (eso lo
// hace un paquete de corpus), solo la forma.
func obligacionPorObjeto() Obligacion {
	return Obligacion{
		ID:       "medicion.notificacion_por_incidente",
		ClaseE2E: "notificatoria",
		Temporalidad: &Temporalidad{
			Primitiva:  "plazo",
			Hito:       "notificacion_inicial",
			Limite:     "PT72H",
			Regimen:    RegimenSpec{Computo: "naturales", Cierre: "exacto"},
			Disparador: map[string]string{"hecho": "conocimiento_del_incidente"},
		},
	}
}

// ---------------------------------------------------------------------------
// Medicion 1: la primitiva. RESULTADO POSITIVO.
// ---------------------------------------------------------------------------

// Un `Plazo` no sabe nada de sujetos ni de objetos: toma el ancla de los hechos
// que le pasan EN CADA LLAMADA. Tres incidentes son tres llamadas y tres fechas
// correctas, y lo mismo vale para los tres clientes del 81.12 con `Periodica`.
//
// Se mide para poder decir con dato lo contrario de lo que parece: el reloj por
// objeto NO pide una primitiva nueva. Si esta medicion sale mal, la etapa 4
// empieza en nucleo/ventana; sale bien, asi que empieza donde debe.
func TestLaPrimitivaYaSabeContarPorObjeto(t *testing.T) {
	reg := ventana.Regimen{
		Cal:    ventana.NuevoCalendario("utc-v1", "medicion", "corpus", time.UTC),
		Comp:   ventana.Naturales,
		Cierre: ventana.CierreExacto,
	}
	d72, err := ventana.ParseDuracion("PT72H")
	if err != nil {
		t.Fatal(err)
	}
	p := ventana.Plazo{
		Disparador: "conocimiento",
		Hitos:      []ventana.Hito{{ID: "notificacion_inicial", Limite: d72, Reg: reg}},
	}

	quiere := map[string]string{
		"2026-03-02T09:00:00Z": "2026-03-05T09:00:00Z",
		"2026-03-04T18:30:00Z": "2026-03-07T18:30:00Z",
		"2026-07-11T23:00:00Z": "2026-07-14T23:00:00Z",
	}
	for desde, hasta := range quiere {
		vs := p.Vencimientos(ventana.Hechos{"conocimiento": instante(t, desde)}, time.Time{})
		if len(vs) != 1 || vs[0].Estado != ventana.Determinado {
			t.Fatalf("incidente conocido el %s: %+v", desde, vs)
		}
		if got := vs[0].Vence.Format(time.RFC3339); got != hasta {
			t.Errorf("incidente conocido el %s: el motor dice %s y son 72 horas, o sea %s",
				desde, got, hasta)
		}
	}
}

// Y la otra punta: `historia` tambien esta lista, porque su clave la elige
// quien llama. `PrimerConocimiento` responde por incidente sin cambiar nada:
// basta con que la clave SEA el incidente.
//
// Esto no es un adorno de la medicion, es la mitad util: dice que el eje
// bitemporal del art. 33 no hay que construirlo en la etapa 4, hay que USARLO,
// y dice ademas donde vive la identidad que falta (en quien llama, no en el
// paquete).
func TestLaHistoriaBitemporalYaDistingueDosIncidentes(t *testing.T) {
	var h historia.Historia
	// El mismo tipo de suceso, dos incidentes. Ocurrieron el mismo dia; se
	// supieron con nueve dias de diferencia, que es lo que separa sus plazos.
	h.Registrar(historia.CambioEstado{
		Prueba: "incidente/2026-014", De: estado.Pass, A: estado.FailVencido,
		InstanteHecho:    instante(t, "2026-03-01T04:00:00Z"),
		InstanteRegistro: instante(t, "2026-03-02T09:00:00Z"),
		Causa:            "observacion",
	})
	h.Registrar(historia.CambioEstado{
		Prueba: "incidente/2026-015", De: estado.Pass, A: estado.FailVencido,
		InstanteHecho:    instante(t, "2026-03-01T04:00:00Z"),
		InstanteRegistro: instante(t, "2026-03-11T17:00:00Z"),
		Causa:            "observacion",
	})

	uno, ok := h.PrimerConocimiento("incidente/2026-014", estado.FailVencido)
	if !ok || !uno.Equal(instante(t, "2026-03-02T09:00:00Z")) {
		t.Fatalf("primer conocimiento del 014: %v (%v)", uno, ok)
	}
	dos, ok := h.PrimerConocimiento("incidente/2026-015", estado.FailVencido)
	if !ok || !dos.Equal(instante(t, "2026-03-11T17:00:00Z")) {
		t.Fatalf("primer conocimiento del 015: %v (%v)", dos, ok)
	}
	if uno.Equal(dos) {
		t.Fatal("dos incidentes con el mismo primer conocimiento: la clave no los separa")
	}
}

// ---------------------------------------------------------------------------
// Medicion 2: el hecho. AQUI SE ROMPE, Y DE DOS MANERAS.
// ---------------------------------------------------------------------------

// PRIMERA FORMA: el paquete nombra un hecho y el operador tiene tres.
//
// `Temporalidad.Disparador["hecho"]` es UNA cadena escrita en el paquete. Un
// paquete de corpus no puede conocer los incidentes de un cliente, asi que no
// puede nombrarlos: lo unico que puede escribir es la FORMA del hecho. Si el
// alcance trae las fechas por objeto (una clave por incidente), el disparador
// del paquete no casa con ninguna y el motor contesta lo unico que sabe
// contestar: que el reloj no ha arrancado.
//
// Lo que se mide es la frase que saldria en pantalla: con TRES incidentes
// abiertos y sus tres fechas registradas, el producto dice "falta la fecha".
func TestConTresIncidentesRegistradosElRelojDiceQueNoHaArrancado(t *testing.T) {
	o := obligacionPorObjeto()
	hechos := ventana.Hechos{
		"conocimiento_del_incidente.2026-014": instante(t, "2026-03-02T09:00:00Z"),
		"conocimiento_del_incidente.2026-015": instante(t, "2026-03-11T17:00:00Z"),
		"conocimiento_del_incidente.2026-016": instante(t, "2026-03-12T08:00:00Z"),
	}
	vs, err := VencimientosDe(o, hechos, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 {
		t.Fatalf("MEDICION CADUCADA: el motor devuelve %d vencimientos y este test se "+
			"escribio midiendo que devolvia 1. Si ya sabe contar por objeto, reescribe la "+
			"medicion y la conclusion de la cabecera: %+v", len(vs), vs)
	}
	if vs[0].Estado != ventana.PendienteDeHecho {
		t.Fatalf("MEDICION CADUCADA: con las fechas por objeto el motor contesta %q, y se "+
			"midio que contestaba %q", vs[0].Estado, ventana.PendienteDeHecho)
	}
	// La medicion, dicha entera: no es que falte precision, es que los tres
	// incidentes NO EXISTEN para el motor. No salen ni siquiera como tres filas
	// pendientes.
	t.Logf("MEDIDO: 3 incidentes registrados -> 1 vencimiento, estado %q, regla: %s",
		vs[0].Estado, vs[0].Regla)
}

// SEGUNDA FORMA, y es la cara: si el alcance usa LA CLAVE QUE EL PAQUETE
// NOMBRA, el segundo incidente PISA al primero. `ventana.Hechos` es un
// map[string]time.Time, o sea una fecha por nombre.
//
// El resultado no es un error ni una fila de menos: es UNA FECHA, la del ultimo
// que se escribio, presentada con la misma cara que cualquier otra. El
// incidente cuyo plazo vencia antes desaparece sin dejar rastro, que es el
// descarte silencioso (docs/pendientes.md) en su version mas cara: aqui lo que
// se pierde no es una fila de calendario, es una notificacion a la autoridad.
func TestElSegundoIncidentePISAAlPrimeroEnElMapaDeHechos(t *testing.T) {
	o := obligacionPorObjeto()
	const clave = "conocimiento_del_incidente"

	primero := instante(t, "2026-03-02T09:00:00Z") // vence el 05 a las 09:00
	segundo := instante(t, "2026-03-11T17:00:00Z") // vence el 14 a las 17:00

	hechos := ventana.Hechos{clave: primero}
	hechos[clave] = segundo // el operador registra el segundo incidente

	vs, err := VencimientosDe(o, hechos, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 1 || vs[0].Estado != ventana.Determinado {
		t.Fatalf("MEDICION CADUCADA: %+v", vs)
	}
	quiere := segundo.Add(72 * time.Hour)
	if !vs[0].Vence.Equal(quiere) {
		t.Fatalf("MEDICION CADUCADA: el motor da %s y el mapa se queda con el ultimo (%s)",
			vs[0].Vence.Format(time.RFC3339), quiere.Format(time.RFC3339))
	}
	// Lo que se mide es la perdida, no la fecha: el plazo del primer incidente
	// no esta en ningun sitio de la respuesta.
	perdido := primero.Add(72 * time.Hour)
	for _, v := range vs {
		if v.Vence.Equal(perdido) {
			t.Fatal("MEDICION CADUCADA: el plazo del primer incidente sigue ahi")
		}
	}
	t.Logf("MEDIDO: dos incidentes bajo la clave que nombra el paquete -> 1 vencimiento "+
		"(%s). El del primer incidente (%s), que vencia NUEVE DIAS ANTES, no sale",
		vs[0].Vence.Format(time.RFC3339), perdido.Format(time.RFC3339))
}

// ---------------------------------------------------------------------------
// Medicion 3: el hito. LA IDENTIDAD YA ESTA OCUPADA, Y A PROPOSITO.
// ---------------------------------------------------------------------------

// El hito es la identidad por la que casa un vencimiento con la fila de un
// dorado (invariante 7) y por la que la ley de conservacion etiqueta destinos.
// Tres incidentes darian tres vencimientos con el hito "notificacion_inicial",
// o sea tres cosas distintas con el mismo nombre.
//
// Y esto NO es un hueco que nadie hubiera visto: el motor ya lo prohibe con su
// propio mensaje, y el linter tambien. Que la puerta exista y este bien es
// justo lo que hace la medicion concluyente: el hito no se puede reutilizar
// para llevar el objeto, porque hay dos guardas escritas para impedirlo.
//
// Se mide por el linter porque es el camino que un autor de paquete recorreria:
// escribir el dorado del 81.12 con una fila por cliente.
func TestUnDoradoPorObjetoNoSePuedeESCRIBIRConLaIdentidadDeHoy(t *testing.T) {
	// El dorado que pediria el art. 81.12: dos clientes, la misma revision.
	d := Dorado{
		Caso:       "dos clientes, cada uno con su ciclo de 24 meses",
		Obligacion: "medicion.revision_de_idoneidad",
		Hechos: map[string]string{
			"evaluacion_inicial": "2025-01-15",
		},
		Hasta: "2027-06-01T23:59:59Z",
		Esperado: []EsperadoDorado{
			{Hito: "revision_de_idoneidad#1", Vence: "2027-01-15T23:59:59Z"},
			{Hito: "revision_de_idoneidad#1", Vence: "2027-05-20T23:59:59Z"},
		},
		CitaDelEsperado: "medicion, no es un caso de corpus",
	}

	var errs []string
	validarEsperadoDeDorado(d, func(f string, a ...any) {
		errs = append(errs, strings.TrimSpace(strings.ReplaceAll(fmt.Sprintf(f, a...), "\n", " ")))
	})

	var elQueImporta string
	for _, e := range errs {
		if strings.Contains(e, "dos veces en el esperado") {
			elQueImporta = e
		}
	}
	if elQueImporta == "" {
		t.Fatalf("MEDICION CADUCADA: el linter ya no rechaza dos filas con el mismo hito, "+
			"asi que la identidad del hito habria dejado de ser unica. Revisa la conclusion "+
			"de la cabecera antes de seguir. Errores: %v", errs)
	}
	t.Logf("MEDIDO: el formato de dorado rechaza el caso por objeto, y con razon: %s",
		elQueImporta)

	// La otra mitad, para que no se lea como un problema del formato de prueba:
	// el MOTOR tiene la misma guarda, escrita en EjecutarDorado ("el motor
	// devuelve el hito %q dos veces, asi que emparejar por hito dejaria uno de
	// los dos sin comprobar"). Son dos guardas independientes diciendo lo
	// mismo, asi que el hito no lleva objetos: lleva hitos.
}

// ---------------------------------------------------------------------------
// La conclusion, escrita como test para que caduque si deja de ser cierta.
// ---------------------------------------------------------------------------

// LO QUE FALTA, EN UNA FRASE: el hecho tiene nombre y no tiene instancia.
//
// Este test no mide nada nuevo: fija la forma del arreglo para que, si alguien
// la implementa de otra manera, sea una decision y no un descuido. Un hecho por
// objeto necesita DOS datos, y hoy `ventana.Hechos` solo lleva uno:
//
//	nombre   lo escribe el PAQUETE  "conocimiento_del_incidente"  (existe hoy)
//	objeto   lo aporta el ALCANCE   "incidente/2026-014"          (no existe)
//
// Y el objeto tiene que viajar la cadena entera, porque cada eslabon empareja
// por identidad y hoy todos emparejan por una identidad que no lo lleva:
// hecho -> vencimiento -> fila de dorado -> destino de la ley de conservacion.
// Meterlo en la mitad de la cadena produce lo de siempre: una junta sin
// vigilar, que es donde se rompio la ley de conservacion la semana pasada.
//
// El test comprueba la unica parte mecanizable de eso: que el nombre del hecho
// que escribe un paquete NO contiene hoy ninguna instancia. Si algun dia un
// paquete mete un identificador de objeto dentro del nombre del hecho (que es
// el atajo que va a tentar a alguien), esto se pone rojo: ese atajo hace que el
// paquete tenga que conocer los objetos del cliente, que es exactamente lo que
// no puede hacer.
func TestNingunPaqueteNombraUnObjetoDentroDeUnHecho(t *testing.T) {
	p, err := Cargar("../../paquetes")
	if err != nil {
		t.Fatalf("cargando el corpus: %v", err)
	}
	// Un identificador de objeto se reconoce por lo que es: una instancia
	// concreta metida en un nombre generico. Se busca la forma que tentaria a
	// alguien (un ano con numero de expediente, o un id con barra).
	sospechoso := func(s string) bool {
		return strings.Contains(s, "/") || strings.Contains(s, "#")
	}
	n := 0
	for _, paq := range p {
		for _, o := range paq.Obligaciones {
			if o.Temporalidad == nil {
				continue
			}
			n++
			for k, v := range o.Temporalidad.Disparador {
				if sospechoso(v) {
					t.Errorf("%s: el disparador %q vale %q, que parece llevar un objeto "+
						"dentro. Un paquete nombra la FORMA del hecho; la instancia la "+
						"aporta el alcance (ver la cabecera de este fichero)", o.ID, k, v)
				}
			}
			for _, r := range o.Temporalidad.ReabrePor {
				if sospechoso(r) {
					t.Errorf("%s: la reapertura %q parece llevar un objeto dentro", o.ID, r)
				}
			}
			// Y `Efecto`, que es el UNICO nombre de hecho que tiene un
			// preaviso: `validarPreaviso` le prohibe declarar disparador, asi
			// que hasta hoy esta comprobacion recorria dos campos y a esa
			// primitiva no le miraba ninguno. Lo vio el frente A al escribir el
			// primer preaviso del AI Act (03-09-2026). Es la misma forma que el
			// fallo del ejecutor de dorados: una guarda escrita cuando solo
			// habia una manera de nombrar el hecho, y una primitiva nueva que
			// la nombra de otra.
			if sospechoso(o.Temporalidad.Efecto) {
				t.Errorf("%s: la fecha de efecto %q parece llevar un objeto dentro. En un "+
					"preaviso es el unico nombre de hecho que hay, asi que si se cuela ahi "+
					"no se cuela en ningun otro sitio", o.ID, o.Temporalidad.Efecto)
			}
		}
	}
	if n == 0 {
		t.Fatal("ningun reloj recorrido: el test no ha comprobado nada")
	}
	// El resumen tranquilizador SOLO si no ha fallado nada. Imprimir "ninguno
	// con instancia" al lado de un fallo que dice justo lo contrario ya ha
	// pasado en este repositorio dos veces.
	if !t.Failed() {
		t.Logf("MEDIDO: %d relojes del corpus, ninguno con instancia dentro del nombre del hecho", n)
	}
}
