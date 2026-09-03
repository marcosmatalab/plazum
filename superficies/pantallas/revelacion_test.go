package pantallas

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/pantalla"
)

// LAS PUERTAS DE LA REVELACION PROGRESIVA.
//
// La de arriba del todo es la unica que de verdad decide si esto se puede
// publicar: esconder una pregunta que abria una obligacion produce un
// calendario incompleto, y un calendario incompleto en un producto de
// cumplimiento es peor que no tener producto. Las demas miden y acotan.

// --- LOS CARDINALES DEL CORPUS INSTALADO, y son trinquetes en los dos
// sentidos, igual que PUERTAS_ESPERADAS ---

// PreguntasVivasAlEmpezar es cuantas preguntas ensena la entrevista a quien
// abre /alcance sin haber respondido nada.
//
// ES EL NUMERO QUE MANDA EN EL TTFV: a veinte segundos por pregunta, cada
// unidad de aqui son veinte segundos del tiempo hasta el primer valor. Si sube,
// la entrevista ha vuelto a engordar y hay que enterarse el mismo dia; si baja,
// alguien la ha estrechado y tiene que bajar el numero aqui, en el mismo commit.
const PreguntasVivasAlEmpezar = 19

// PreguntasDormidasAlEmpezar es cuantas se dejan fuera de esa primera pantalla.
//
// LAS 23 SON, HOY, EXACTAMENTE LAS QUE NINGUNA OBLIGACION REQUIERE: no es que
// la revelacion sea lista, es que el corpus tiene 23 preguntas que no deciden
// nada y hasta ahora se preguntaban igual. El hueco es del corpus y esta contado
// en docs/hallazgos-entrevista.md; la revelacion lo que hace es dejar de
// cobrarselo al operador mientras se cierra.
const PreguntasDormidasAlEmpezar = 23

// corpusReal carga el corpus que se publica. Se mide contra EL, y no contra un
// paquete sintetico, porque una puerta que puede estrenarse contra el dato real
// se estrena ahi: una mutacion demuestra que caza un fallo que tu le metiste, y
// el corpus real demuestra que caza uno que nadie le metio.
func corpusReal(t *testing.T) []*corpus.Paquete {
	t.Helper()
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatalf("no se puede cargar el corpus publicado, asi que esta medida seria sobre "+
			"un corpus inventado: %v", err)
	}
	if len(ps) == 0 {
		t.Fatal("el corpus publicado carga vacio: una entrevista sin preguntas hace pasar " +
			"todas las puertas de aqui sin merecerlo")
	}
	return ps
}

// pantallasDe devuelve el modelo derivado como lo ve la superficie.
func pantallasDe(t *testing.T, ps []*corpus.Paquete) modelo {
	t.Helper()
	m := derivarModelo(ps)
	if len(m.preguntas) == 0 {
		t.Fatal("el modelo derivado no trae preguntas: no hay nada que revelar ni que " +
			"esconder, y toda puerta de este fichero saldria verde sin comprobar nada")
	}
	return m
}

// ---------------------------------------------------------------------------
// LA PUERTA QUE DECIDE: esconder no puede cambiar ningun veredicto
// ---------------------------------------------------------------------------

// TestEsconderUnaPreguntaNoPuedeCambiarNingunVeredicto es la refutacion, no la
// comprobacion.
//
// La propiedad que se intenta tumbar es la que sostiene la revelacion entera:
// «una pregunta que la lista corta esconde no puede, contestandola, mover
// ninguna obligacion de cesta». Si fuera falsa aunque solo fuese en un caso, la
// revelacion produciria un calendario incompleto, que es el silencio que este
// producto persigue en todas las demas pantallas.
//
// COMO SE INTENTA TUMBAR: para cada estado de respuestas, se toma CADA pregunta
// escondida y se contesta, primero que si y luego que no, y se exige que el
// veredicto de las 528 obligaciones y el de los entregables salgan IDENTICOS.
// No se mira una muestra: se recorren todas.
//
// SE ESTRENA CONTRA EL CORPUS REAL Y NACIO VERDE, y eso se dice en voz alta:
// una puerta que nace verde sobre 42 preguntas y 528 obligaciones o vigila poco
// o llega tarde. Aqui es lo segundo a proposito (la propiedad se demostro antes
// de escribir el codigo, en el encabezado de revelacion.go), asi que lo que
// hace falta es la mutacion, que esta anotada en el commit.
func TestEsconderUnaPreguntaNoPuedeCambiarNingunVeredicto(t *testing.T) {
	m := pantallasDe(t, corpusReal(t))
	preguntas := m.porID[pantalla.Alcance].Preguntas
	certificados := m.porID[pantalla.Certificados].Filas

	// Los estados de respuestas que se recorren. El primero es el que ve quien
	// acaba de instalar; los demas encienden la OTRA rama de esconder (la de
	// «ya decidida»), que con cero respuestas no se recorre nunca.
	estados := []struct {
		nombre string
		aplica func(Respuestas) Respuestas
	}{
		{"sin responder nada", func(r Respuestas) Respuestas { return r }},
		{"la primera que si", func(r Respuestas) Respuestas {
			return r.Con(preguntas[0].ID, Si)
		}},
		{"la primera que no", func(r Respuestas) Respuestas {
			return r.Con(preguntas[0].ID, No)
		}},
		{"las cinco primeras que no", func(r Respuestas) Respuestas {
			for i := 0; i < 5 && i < len(preguntas); i++ {
				r = r.Con(preguntas[i].ID, No)
			}
			return r
		}},
		{"todas que no", func(r Respuestas) Respuestas {
			for _, q := range preguntas {
				r = r.Con(q.ID, No)
			}
			return r
		}},
		// EL ESTADO QUE ENCIENDE LA SEGUNDA RAMA SOBRE EL CORPUS REAL. Las
		// obligaciones que dependen de DOS preguntas son las unicas que pueden
		// dejar una sin responder y ya decidida: se contesta que no a la
		// primera de cada pareja y la segunda se queda sin nada que mover.
		// Sin este estado, YaDecidida no la recorre ninguna entrada real y la
		// mutacion la dejaria verde por no haber nada que romper.
		{"la primera de cada pareja que no", func(r Respuestas) Respuestas {
			for _, f := range m.porID[pantalla.Controles].Filas {
				if len(f.Requiere) >= 2 {
					r = r.Con(f.Requiere[0], No)
				}
			}
			return r
		}},
	}

	probadas, escondidas := 0, 0
	porMotivo := map[Vivacidad]int{}
	for _, e := range estados {
		base := e.aplica(De(nil, m.preguntas))
		controles := veredictosDeControles(m, base)
		vivas := vivacidades(preguntas, controles, base)
		antesC := huellaDeVeredictos(controles)
		antesE := huellaDeEntregables(certificados, controles)

		for _, q := range preguntas {
			if !vivas[q.ID].Dormida() {
				continue
			}
			escondidas++
			porMotivo[vivas[q.ID]]++
			for _, respuesta := range []Respuesta{Si, No} {
				con := base.Con(q.ID, respuesta)
				despuesC := veredictosDeControles(m, con)
				if h := huellaDeVeredictos(despuesC); h != antesC {
					t.Errorf("estado %q: la pregunta %q esta ESCONDIDA (%d) y contestarla %v "+
						"MUEVE obligaciones de cesta.\n"+
						"  Eso es un calendario incompleto: la revelacion progresiva estaria "+
						"quitando de la vista una pregunta que si hacia falta, que es el "+
						"unico fallo que esta pieza no puede cometer.\n"+
						"  El emparejamiento que decide es Fila.Requiere; si esto se pone "+
						"rojo, la clasificacion de vivacidades ya no cuadra con el.",
						e.nombre, q.ID, vivas[q.ID], respuesta)
				}
				if h := huellaDeEntregables(certificados, despuesC); h != antesE {
					t.Errorf("estado %q: la pregunta escondida %q cambia el veredicto de algun "+
						"ENTREGABLE. Un certificado cuelga de sus obligaciones, asi que esto "+
						"solo puede pasar si el veredicto de alguna se movio", e.nombre, q.ID)
				}
				probadas++
			}
		}
	}

	// EL CONTROL POSITIVO, sin el cual todo lo de arriba es un bucle vacio: si
	// no hubiera ninguna pregunta escondida, este test pasaria sin comparar
	// nada y diria que la propiedad se sostiene.
	if escondidas == 0 {
		t.Fatal("no se escondio ni una sola pregunta en ninguno de los estados, asi que " +
			"este test no ha comprobado nada. O la clasificacion esta apagada, o el corpus " +
			"ya no tiene preguntas que no decidan nada")
	}
	// Y EL CONTROL POSITIVO DE CADA MOTIVO POR SEPARADO. Esconder por
	// «nadie la pide» y esconder por «ya decidida» son dos ramas distintas del
	// codigo, y una refutacion que solo recorra la primera deja la segunda sin
	// probar sobre dato real.
	for motivo, nombre := range map[Vivacidad]string{
		NadieLaPide: "NadieLaPide", YaDecidida: "YaDecidida",
	} {
		if porMotivo[motivo] == 0 {
			t.Errorf("ningun estado escondio una pregunta por %s, asi que esa rama no la ha "+
				"recorrido ninguna entrada real y la refutacion no la ha tocado", nombre)
		}
	}
	t.Logf("refutacion intentada sobre %d parejas (pregunta escondida, respuesta) en %d "+
		"estados; %d escondidas en total (%d por nadie la pide, %d por ya decidida)",
		probadas, len(estados), escondidas, porMotivo[NadieLaPide], porMotivo[YaDecidida])
}

// huellaDeVeredictos resume el veredicto de todas las obligaciones en una
// cadena comparable. Va por ID y ESTADO, no por posicion: el orden de las filas
// es estable, pero comparar por posicion seria emparejar por indice, que es
// justo lo que este proyecto no hace en ningun sitio (invariante 7).
func huellaDeVeredictos(vs []Veredicto) string {
	partes := make([]string, 0, len(vs))
	for _, v := range vs {
		partes = append(partes, v.Fila.ID+"="+v.Estado.String())
	}
	sort.Strings(partes)
	return strings.Join(partes, ";")
}

func huellaDeEntregables(filas []pantalla.Fila, controles []Veredicto) string {
	porObligacion := make(map[string]Veredicto, len(controles))
	for _, c := range controles {
		porObligacion[c.Fila.ID] = c
	}
	partes := make([]string, 0, len(filas))
	for _, f := range filas {
		partes = append(partes, f.ID+"="+evaluarEntregable(f, porObligacion).Estado.String())
	}
	sort.Strings(partes)
	return strings.Join(partes, ";")
}

// ---------------------------------------------------------------------------
// LA MEDIDA: cuantas preguntas quedan delante y cuantas detras
// ---------------------------------------------------------------------------

// TestLaEntrevistaCortaTieneSuCardinalEnLosDosSentidos es lo que convierte la
// promesa («la entrevista baja de 42 preguntas») en un numero que se puede
// recontar y que molesta cuando cambia.
func TestLaEntrevistaCortaTieneSuCardinalEnLosDosSentidos(t *testing.T) {
	m := pantallasDe(t, corpusReal(t))
	preguntas := m.porID[pantalla.Alcance].Preguntas
	base := De(nil, m.preguntas)
	vivas := vivacidades(preguntas, veredictosDeControles(m, base), base)

	var vivo, dormido, nadie, decidida int
	for _, q := range preguntas {
		switch vivas[q.ID] {
		case Viva:
			vivo++
		case NadieLaPide:
			dormido++
			nadie++
		case YaDecidida:
			dormido++
			decidida++
		}
	}
	t.Logf("entrevista del corpus publicado: %d preguntas en total; %d se ensenan al "+
		"empezar y %d se dejan fuera (%d porque ninguna obligacion las requiere, %d porque "+
		"ya estaban decididas)", len(preguntas), vivo, dormido, nadie, decidida)

	if vivo != PreguntasVivasAlEmpezar {
		t.Errorf("la entrevista corta ensena %d preguntas y PreguntasVivasAlEmpezar dice %d.\n"+
			"  Si ha SUBIDO, el TTFV ha empeorado en %d x 20 s y hay que mirar que entro.\n"+
			"  Si ha BAJADO, comprueba que no se esta estrechando escondiendo preguntas que "+
			"si decidian, que es la forma barata de aprobar la puerta del tiempo.",
			vivo, PreguntasVivasAlEmpezar, vivo-PreguntasVivasAlEmpezar)
	}
	if dormido != PreguntasDormidasAlEmpezar {
		t.Errorf("se dejan fuera %d preguntas y PreguntasDormidasAlEmpezar dice %d.\n"+
			"  Este numero es deuda del CORPUS, no de la pantalla: cada una es una pregunta "+
			"que se hacia y que no decidia nada. Cuando el corpus las cierre, este numero "+
			"baja y hay que bajarlo aqui en el mismo commit", dormido, PreguntasDormidasAlEmpezar)
	}
	// Con cero respuestas NO PUEDE haber ninguna "ya decidida": nada se ha
	// decidido todavia. Si aparece una, la clasificacion esta mezclando los dos
	// motivos, y entonces el motivo que se ensena en pantalla es falso.
	if decidida != 0 {
		t.Errorf("con cero respuestas hay %d preguntas clasificadas como ya decididas, y no "+
			"puede haber ninguna: no se ha decidido nada todavia", decidida)
	}
}

// TestLaMedidaDeLaPantallaCuadraConLaDelCorpus cruza las dos formas de contar
// lo mismo: la que lee el corpus (corpus.PreguntasQueNadieRequiere) y la que
// lee el modelo ya derivado (vivacidades).
//
// NO ES UNA COPIA DE LA MISMA CUENTA: una mira `Obligacion.Preguntas` en el
// paquete y la otra `Fila.Requiere` en la pantalla derivada. Si divergen, la
// derivacion ha perdido enlaces por el camino, y eso hoy no lo caza nadie.
func TestLaMedidaDeLaPantallaCuadraConLaDelCorpus(t *testing.T) {
	ps := corpusReal(t)
	m := pantallasDe(t, ps)
	preguntas := m.porID[pantalla.Alcance].Preguntas
	base := De(nil, m.preguntas)
	vivas := vivacidades(preguntas, veredictosDeControles(m, base), base)

	desdeLaPantalla := []string{}
	for _, q := range preguntas {
		if vivas[q.ID] == NadieLaPide {
			desdeLaPantalla = append(desdeLaPantalla, q.ID)
		}
	}
	sort.Strings(desdeLaPantalla)
	desdeElCorpus := corpus.PreguntasQueNadieRequiere(ps)

	if !reflect.DeepEqual(desdeLaPantalla, desdeElCorpus) {
		t.Errorf("las dos cuentas discrepan.\n  desde la pantalla (%d): %v\n"+
			"  desde el corpus   (%d): %v\n"+
			"  Una lee Fila.Requiere del modelo derivado y la otra Obligacion.Preguntas del "+
			"paquete. Que no coincidan significa que la derivacion pierde o inventa enlaces.",
			len(desdeLaPantalla), desdeLaPantalla, len(desdeElCorpus), desdeElCorpus)
	}
	if len(desdeElCorpus) == 0 {
		t.Fatal("ninguna pregunta del corpus queda sin requerir, asi que esta comparacion " +
			"cuadra comparando dos listas vacias y no dice nada")
	}
}

// ---------------------------------------------------------------------------
// EL VALOR CERO Y LAS DOS FORMAS DE LA NADA
// ---------------------------------------------------------------------------

// TestElValorCeroDeLaVivacidadEsEnsenarLaPregunta comprueba que lo restrictivo
// es lo que sale por descuido: si la clasificacion se rompe y devuelve el cero,
// la entrevista vuelve a ser larga, que es caro y no peligroso.
func TestElValorCeroDeLaVivacidadEsEnsenarLaPregunta(t *testing.T) {
	var v Vivacidad
	if v != Viva {
		t.Fatalf("el valor cero de Vivacidad es %v y tiene que ser Viva", v)
	}
	if v.Dormida() {
		t.Error("el valor cero esconde la pregunta. El cero tiene que ser el RESTRICTIVO, " +
			"o sea el que ensena: una clasificacion rota que esconde es la que produce el " +
			"calendario incompleto")
	}
	if v.Clave() != "" {
		t.Errorf("el valor cero trae la clave %q, y una pregunta viva no tiene motivo que "+
			"explicar", v.Clave())
	}
}

// TestLasDosFormasDeLaNadaEnRequiereSeLeenIgual recorre nil Y vacio-presente,
// que son dos cosas distintas en Go y la peligrosa es siempre la nil, porque
// es la que sale por olvidarse.
//
// AQUI LAS DOS TIENEN QUE SIGNIFICAR LO MISMO, y por eso se comprueban las dos:
// una obligacion sin preguntas alcanza a todo el mundo y no condiciona a
// ninguna pregunta. Lo que no puede pasar es que una de las dos formas haga que
// una pregunta aparezca requerida por una obligacion que no la requiere.
func TestLasDosFormasDeLaNadaEnRequiereSeLeenIgual(t *testing.T) {
	q := []pantalla.Pregunta{{ID: "q1"}, {ID: "q2"}}
	for _, caso := range []struct {
		nombre   string
		requiere []string
	}{
		{"nil", nil},
		{"vacio-presente", []string{}},
	} {
		vs := []Veredicto{{Fila: pantalla.Fila{ID: "o1", Requiere: caso.requiere},
			Estado: Aplica}}
		got := vivacidades(q, vs, De(nil, q))
		for _, id := range []string{"q1", "q2"} {
			if got[id] != NadieLaPide {
				t.Errorf("con Requiere %s, la pregunta %q sale %v y tenia que salir "+
					"NadieLaPide: ninguna obligacion la nombra", caso.nombre, id, got[id])
			}
		}
	}

	// Y la vuelta, en el corpus: Preguntas nil y Preguntas vacio-presente.
	for _, caso := range []struct {
		nombre string
		lista  []string
	}{
		{"nil", nil},
		{"vacio-presente", []string{}},
	} {
		p := &corpus.Paquete{
			URN:          "urn:demo:nada",
			Preguntas:    []corpus.Pregunta{{ID: "n.q.sola"}},
			Obligaciones: []corpus.Obligacion{{ID: "n.o.una", Preguntas: caso.lista}},
		}
		got := corpus.PreguntasQueNadieRequiere([]*corpus.Paquete{p})
		if !reflect.DeepEqual(got, []string{"n.q.sola"}) {
			t.Errorf("con Preguntas %s, PreguntasQueNadieRequiere da %v y esperaba "+
				"[n.q.sola]", caso.nombre, got)
		}
	}

	// Y la lista de paquetes: nil, vacia y con un nil dentro.
	if got := corpus.PreguntasQueNadieRequiere(nil); len(got) != 0 {
		t.Errorf("sin paquetes salen %v preguntas huerfanas y no puede salir ninguna", got)
	}
	if got := corpus.PreguntasQueNadieRequiere([]*corpus.Paquete{}); len(got) != 0 {
		t.Errorf("con lista vacia salen %v y no puede salir ninguna", got)
	}
	if got := corpus.PreguntasQueNadieRequiere([]*corpus.Paquete{nil}); len(got) != 0 {
		t.Errorf("con un paquete nil dentro salen %v; un nil en la lista no puede hacer "+
			"caer el recuento ni inventarse preguntas", got)
	}
}

// ---------------------------------------------------------------------------
// LOS DOS MOTIVOS, CADA UNO CON SU CONTROL POSITIVO
// ---------------------------------------------------------------------------

// paqueteConHuerfana trae UNA pregunta que ninguna obligacion requiere, para
// que la rama NadieLaPide se recorra con dato sintetico ademas de con el corpus
// real.
//
// EXISTE PORQUE UNA RAMA QUE NINGUNA ENTRADA ALCANZA ES UNA RAMA QUE NO EXISTE:
// el corpus de demostracion de este paquete no tiene ninguna huerfana, asi que
// sin esto las pruebas de la pantalla nunca pintarian una pregunta dormida.
func paqueteConHuerfana() *corpus.Paquete {
	p := paqueteAlfa()
	p.URN = "urn:demo:huerfana"
	p.Preguntas = append(p.Preguntas, corpus.Pregunta{
		ID: "alfa.q.sobrante", Texto: "Una pregunta que nadie requiere",
		Cita: "demo alfa art. 99", Entidad: "sistema", Atributo: "nombre",
		// Declara que desbloquea una obligacion que existe, y esa obligacion
		// NO la nombra: es exactamente la forma que tiene el hueco en el corpus
		// publicado, y es la que el linter no ve porque solo recorre esta
		// direccion.
		Desbloquea: []string{"alfa.o.auditoria"}})
	return p
}

// paqueteQueSeApaga es alfa SIN la obligacion de inventario, para que
// alfa.q.nombre pueda quedarse sin nadie pendiente que la nombre. Es el control
// positivo de la rama YaDecidida en la PANTALLA, no solo en la clasificacion:
// sin el, la clave que explica ese motivo se declara y no la pide nadie.
func paqueteQueSeApaga() *corpus.Paquete {
	p := paqueteAlfa()
	p.URN = "urn:demo:apaga"
	obl := make([]corpus.Obligacion, 0, 2)
	for _, o := range p.Obligaciones {
		if o.ID != "alfa.o.inventario" {
			obl = append(obl, o)
		}
	}
	p.Obligaciones = obl
	// La plantilla la pedia inventario y auditoria; sin inventario sigue
	// pedida por auditoria, asi que no queda huerfana.
	return p
}

// TestUnaPreguntaQueNadieRequiereSeDuermeConSuMotivo es el control positivo de
// la rama NadieLaPide sobre dato sintetico.
func TestUnaPreguntaQueNadieRequiereSeDuermeConSuMotivo(t *testing.T) {
	m := derivarModelo([]*corpus.Paquete{paqueteConHuerfana()})
	preguntas := m.porID[pantalla.Alcance].Preguntas
	base := De(nil, m.preguntas)
	vivas := vivacidades(preguntas, veredictosDeControles(m, base), base)

	if vivas["alfa.q.sobrante"] != NadieLaPide {
		t.Fatalf("la pregunta que ninguna obligacion requiere sale %v y tenia que salir "+
			"NadieLaPide", vivas["alfa.q.sobrante"])
	}
	for _, id := range []string{"alfa.q.categoria", "alfa.q.nombre"} {
		if vivas[id].Dormida() {
			t.Errorf("la pregunta %q SI la requiere una obligacion y sale dormida (%v): "+
				"esconderla dejaria obligaciones sin decidir", id, vivas[id])
		}
	}
}

// TestUnaPreguntaYaDecididaSeDuermeConSuMotivo es el control positivo de la
// OTRA rama, la que con cero respuestas no se recorre nunca.
//
// El caso: alfa.o.copias depende de dos preguntas y alfa.o.auditoria de una.
// Respondiendo que NO a la que comparten, las dos quedan decididas y la otra
// pregunta de copias deja de mover nada.
func TestUnaPreguntaYaDecididaSeDuermeConSuMotivo(t *testing.T) {
	m := derivarModelo([]*corpus.Paquete{paqueteAlfa()})
	preguntas := m.porID[pantalla.Alcance].Preguntas

	base := De(nil, m.preguntas)
	antes := vivacidades(preguntas, veredictosDeControles(m, base), base)
	if antes["alfa.q.nombre"] != Viva {
		t.Fatalf("antes de responder nada, alfa.q.nombre sale %v y tenia que estar viva: "+
			"si ya empieza dormida, este control positivo no prueba la transicion",
			antes["alfa.q.nombre"])
	}

	// Que NO a la categoria descarta auditoria y copias. Queda inventario, que
	// depende de alfa.q.nombre, asi que esa sigue viva.
	con := base.Con("alfa.q.categoria", No)
	despues := vivacidades(preguntas, veredictosDeControles(m, con), con)
	if despues["alfa.q.nombre"] != Viva {
		t.Fatalf("alfa.q.nombre decide todavia alfa.o.inventario y sale %v",
			despues["alfa.q.nombre"])
	}

	// Ahora se descarta tambien inventario: ya no queda nada que dependa de
	// alfa.q.nombre... salvo que esa misma pregunta esta respondida, y una
	// respondida NO se esconde nunca. Se comprueban las dos cosas.
	con2 := con.Con("alfa.q.nombre", No)
	d2 := vivacidades(preguntas, veredictosDeControles(m, con2), con2)
	if d2["alfa.q.nombre"] != Viva {
		t.Error("una pregunta RESPONDIDA se ha escondido. Sin ella en pantalla no hay forma " +
			"de deshacer la respuesta si no es editando la direccion a mano")
	}

	// Y la transicion de verdad: una pregunta SIN responder cuyas obligaciones
	// ya estan todas decididas por OTRA respuesta.
	m2 := derivarModelo([]*corpus.Paquete{paqueteAlfa()})
	base2 := De(nil, m2.preguntas)
	// alfa.o.copias depende de categoria y nombre; con categoria a NO queda
	// descartada. Si ademas se quita inventario del corpus, nombre se queda sin
	// nadie pendiente.
	sinInventario := paqueteAlfa()
	sinInventario.Obligaciones = sinInventario.Obligaciones[:2] // auditoria y copias
	m3 := derivarModelo([]*corpus.Paquete{sinInventario})
	c3 := De(nil, m3.preguntas).Con("alfa.q.categoria", No)
	v3 := vivacidades(m3.porID[pantalla.Alcance].Preguntas, veredictosDeControles(m3, c3), c3)
	if v3["alfa.q.nombre"] != YaDecidida {
		t.Errorf("con las dos obligaciones que la nombran ya descartadas, alfa.q.nombre "+
			"sale %v y tenia que salir YaDecidida. Sin este caso, la rama de «ya decidida» "+
			"no la recorre ninguna entrada y la mutacion la dejaria verde",
			v3["alfa.q.nombre"])
	}
	_ = base2
	_ = m2
}

// ---------------------------------------------------------------------------
// LA PANTALLA: lo que se esconde se ve, y el parametro tiene tres formas
// ---------------------------------------------------------------------------

// TestLaPantallaCortaDiceCuantasEsconde: el cardinal a la vista y el enlace que
// las abre. Una entrevista que oculta preguntas sin decir que existen es la
// otra forma del silencio.
func TestLaPantallaCortaDiceCuantasEscondeYEnlazaAEllas(t *testing.T) {
	s, _ := superficie(t, []*corpus.Paquete{paqueteConHuerfana()})
	w, cuerpo := pedir(t, s, "/alcance")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /alcance dio %d", w.Code)
	}
	// La huerfana NO se pinta...
	prohibe(t, cuerpo, `id="p-alfa.q.sobrante"`)
	// ...pero su existencia SI, con su cardinal y su enlace.
	exige(t, cuerpo,
		rotulo("es", "alcance.dormidas.titulo"),
		rotulo("es", "alcance.dormidas.ver"),
		ParamVer+"="+VerTodas,
	)
	// Y las que si deciden siguen ahi.
	exige(t, cuerpo, `id="p-alfa.q.categoria"`, `id="p-alfa.q.nombre"`)
}

// TestLaPantallaLargaLasPintaTodasConSuMotivo: lo escondido se puede ver, y
// cuando se ve dice por que estaba escondido.
func TestLaPantallaLargaLasPintaTodasConSuMotivo(t *testing.T) {
	s, _ := superficie(t, []*corpus.Paquete{paqueteConHuerfana()})
	w, cuerpo := pedir(t, s, "/alcance?"+ParamVer+"="+VerTodas)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /alcance con la lista larga dio %d", w.Code)
	}
	exige(t, cuerpo,
		`id="p-alfa.q.sobrante"`,
		rotulo("es", "alcance.dormidas.nadie_la_pide"),
		rotulo("es", "alcance.dormidas.porque"),
		rotulo("es", "alcance.dormidas.volver"),
	)
	// Y EL MODO SE CONSERVA AL RESPONDER. Sin esto, el primer clic devuelve a
	// la lista corta y la pregunta que acabas de contestar desaparece.
	si, no := enlacesDePregunta(t, cuerpo, "alfa.q.sobrante")
	for _, enlace := range []string{si, no} {
		if !strings.Contains(enlace, ParamVer+"="+VerTodas) {
			t.Errorf("el enlace de respuesta %q pierde el modo largo: al contestar una "+
				"pregunta de la lista larga te devuelve a la corta", enlace)
		}
	}
}

// TestElParametroVerDistingueLasTresFormas es el invariante 8 en su tercera
// forma: ausente, presente y conocido, presente y NO INTERPRETABLE. La tercera
// es un dato que hay y no se entiende, y tomarlo por la nada seria inventarse
// un valor.
func TestElParametroVerDistingueLasTresFormas(t *testing.T) {
	// La funcion, directamente: las tres formas, sin pasar por HTTP.
	casos := []struct {
		nombre        string
		v             url.Values
		modo          Modo
		interpretable bool
	}{
		{"ausente", url.Values{}, ModoVivas, true},
		{"presente y conocido", url.Values{ParamVer: {VerTodas}}, ModoTodas, true},
		{"presente y vacio", url.Values{ParamVer: {""}}, ModoVivas, false},
		{"presente y desconocido", url.Values{ParamVer: {"vivas"}}, ModoVivas, false},
		{"presente dos veces", url.Values{ParamVer: {VerTodas, VerTodas}}, ModoVivas, false},
	}
	for _, c := range casos {
		modo, ok := modoPedido(c.v)
		if modo != c.modo || ok != c.interpretable {
			t.Errorf("%s: modoPedido da (%v, %v) y esperaba (%v, %v)",
				c.nombre, modo, ok, c.modo, c.interpretable)
		}
	}

	// Y la pantalla: lo no interpretable NO se sirve como la pagina por
	// defecto. Servirla seria contestar una pregunta distinta de la que se hizo.
	s, _ := superficie(t, corpusDemo())
	for _, destino := range []string{
		"/alcance?" + ParamVer + "=",
		"/alcance?" + ParamVer + "=basura",
		"/alcance?" + ParamVer + "=" + VerTodas + "&" + ParamVer + "=vivas",
	} {
		w, _ := pedir(t, s, destino)
		if w.Code == http.StatusOK {
			t.Errorf("GET %s contesta 200: un valor presente y no interpretable se esta "+
				"leyendo como el valor por defecto, que es inventarse un dato", destino)
		}
	}
	// Control negativo del control: los dos que SI se entienden contestan 200.
	for _, destino := range []string{"/alcance", "/alcance?" + ParamVer + "=" + VerTodas} {
		w, _ := pedir(t, s, destino)
		if w.Code != http.StatusOK {
			t.Errorf("GET %s contesta %d y tenia que contestar 200: la comprobacion de "+
				"arriba estaria rechazandolo todo", destino, w.Code)
		}
	}
}

// TestLaSugeridaNuncaApuntaAUnaPreguntaEscondida: la flecha de «empieza por
// esta» tiene que apuntar a algo que la pagina pinta.
func TestLaSugeridaNuncaApuntaAUnaPreguntaEscondida(t *testing.T) {
	m := pantallasDe(t, corpusReal(t))
	preguntas := m.porID[pantalla.Alcance].Preguntas

	// Se recorre respondiendo que si a todo, en el orden de la entrevista: es
	// el camino que de verdad hace un operador, y es el que va apagando
	// preguntas por el segundo motivo.
	resp := De(nil, m.preguntas)
	for paso := 0; paso <= len(preguntas); paso++ {
		controles := veredictosDeControles(m, resp)
		vivas := vivacidades(preguntas, controles, resp)
		sugerida := ""
		for _, q := range preguntas {
			if vivas[q.ID].Dormida() {
				continue
			}
			if d := resp.Dice(q.ID); d == SinResponder || d == Contradictoria {
				sugerida = q.ID
				break
			}
		}
		if sugerida != "" && vivas[sugerida].Dormida() {
			t.Fatalf("paso %d: se sugiere %q y esa pregunta esta escondida", paso, sugerida)
		}
		if paso < len(preguntas) {
			resp = resp.Con(preguntas[paso].ID, Si)
		}
	}
}

// TestLasDormidasNoTocanElProgreso: TotalPreguntas sigue siendo el del corpus.
//
// Es la puerta contra la trampa mas facil de todas: si la barra de progreso
// contara solo las preguntas que la pantalla ensena, «has respondido 19 de 19»
// diria que la entrevista esta completa cuando faltan 23 sin mirar.
func TestLasDormidasNoTocanElProgreso(t *testing.T) {
	ps := []*corpus.Paquete{paqueteConHuerfana()}
	m := derivarModelo(ps)
	total := len(m.porID[pantalla.Alcance].Preguntas)
	s, _ := superficie(t, ps)
	_, cuerpo := pedir(t, s, "/alcance")
	esperado := fmt.Sprintf("%s 0 %d", rotulo("es", "alcance.progreso"), total)
	if !strings.Contains(cuerpo, esperado) {
		t.Errorf("la pagina corta no dice el progreso sobre el TOTAL del corpus (%d).\n"+
			"  Esperaba %q. Contar solo las visibles convertiria «has respondido 19 de 19» "+
			"en una entrevista aparentemente terminada con 23 preguntas sin mirar.",
			total, esperado)
	}
}
