package plazum

import (
	"sort"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/aplicabilidad"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// EL NUMERO QUE DECIDE SI ESTO ES UN PRODUCTO, BAJO PUERTA.
//
// Cuantas obligaciones ve un CISO de una SaaS espanola de 200 personas que
// contesta la entrevista. Era 8 de 72 el 04-09-2026 por la manana; despues del
// tramo 2 es 72.
//
// # Por que necesita puerta y no un parrafo
//
// Es la unica cifra del proyecto que contesta «esto sirve para alguien». Estaba
// medida a mano, escrita en un informe y en un documento de hallazgos, y sin
// nada que la vigilara: o sea que podia moverse en silencio en las dos
// direcciones. Que BAJE es un producto que empeora sin que nadie se entere; que
// SUBA sin que nadie lo sepa es peor todavia, porque nos lo creeriamos.
//
// # EL ESCENARIO ES UN DATO Y VA ESCRITO ENTERO, y esa es la mitad del asunto
//
// El fallo probable de esta cifra es FAVORECERNOS, y no por un error de calculo:
// quien escribe el escenario controla el numerador Y el denominador. Contestar
// mas preguntas da mas obligaciones, asi que un escenario mas generoso «mejora»
// el producto sin tocar el producto.
//
// Contra eso, tres cosas:
//
//  1. Las respuestas van aqui, las siete, cada una con POR QUE es cierta de esta
//     organizacion. Un escenario que no se puede leer no se puede discutir.
//  2. Se congela el numero de RESPUESTAS, no solo el de obligaciones. Sin eso,
//     el numerador se sube contestando mas.
//  3. Se computa y se registra la DESCOMPOSICION: cuanto de lo que ve depende de
//     que el corpus lo declare y cuanto de que la pantalla sepa mandarlo. Un
//     numero que no dice de donde viene su ventaja es un numero que la esconde.
//
// # QUE NO ES ESTE NUMERO, dicho aqui y no descubierto luego
//
// NO es lo que esa organizacion debe cumplir. Es lo que EL CORPUS DE PLAZUM
// deriva de estas siete respuestas. La cobertura estricta de la v1 esta en el
// 51,4 % (73 relojes de la norma sobre 142 censados, mas 69 rituales de plazum,
// y 7 de los 15 marcos sin denominador): lo que de verdad le toca es MAS, y
// cuanto mas no se sabe. Publicar 72 como «sus obligaciones» seria la misma
// absolucion en falso que este tramo vino a cerrar, cometida un piso mas arriba.
//
// # Se puede repetir con el binario, y asi se verifico
//
//	plazum alcance --respuestas "si=ens.q.datos_personales&si=ens.q.externalizacion&\
//	  si=ens.q.nube&si=iso27001.q.adopcion&si=ley2023.q.canal&\
//	  si=nis2tec.q.entidad_pertinente&v.rgpd.q.papel=responsable" \
//	  --sujeto sis --organizacion "Acme SaaS SL" --salida ciso.json
//	plazum calendario --alcance ciso.json     # «72 alcanzados por la aplicabilidad»

// ObligacionesQueVeElCiso es la cifra, con igualdad exacta en los dos sentidos.
const ObligacionesQueVeElCiso = 72

// RespuestasDelEscenarioDelCiso es el denominador del esfuerzo: siete. Se
// congela para que el numero de arriba no se pueda subir contestando mas.
const RespuestasDelEscenarioDelCiso = 7

// ObligacionesDelCisoQueSoloLleganConValor es LA DESCOMPOSICION, y hoy vale
// CERO, que es un dato incomodo y por eso se escribe.
//
// De las 72, cuantas dependen de que la pantalla sepa mandar un VALOR y no solo
// un si/no. En este escenario, ninguna: la unica respuesta con valor es el papel
// del RGPD, y `responsable` no lo prueba ninguna regla (las que miran ese hueco
// buscan `encargado`). O sea que TODO el salto de 8 a 72 es corpus, y la
// pantalla aporta cero AQUI.
//
// Eso no dice que la pantalla no sirva: sobre la entrevista entera si aporta, y
// esta medido aparte. Dice que en el escenario que publicamos como titular, la
// ventaja es de la rebanada del puente y de nadie mas, y que atribuirsela a las
// dos seria repartir merito que no hay.
const ObligacionesDelCisoQueSoloLleganConValor = 0

// respuestaDelCiso es una respuesta del escenario con su justificacion.
type respuestaDelCiso struct {
	Entidad, Atributo string
	Si                bool
	Valor             string
	PorQue            string
}

// elEscenarioDelCiso: una SaaS espanola de 200 personas, privada, proveedora
// del sector publico. Las siete respuestas y por que cada una es cierta de ella.
var elEscenarioDelCiso = []respuestaDelCiso{
	{"sistema", "trata_datos_personales", true, "",
		"tiene clientes y empleados: no hay SaaS que no trate datos personales"},
	{"sistema", "usa_servicios_en_la_nube", true, "",
		"es SaaS; su producto vive en infraestructura de terceros"},
	{"sistema", "servicios_externalizados", true, "",
		"soporte, alojamiento y observabilidad son de terceros"},
	{"sgsi", "adopta_la_norma", true, "",
		"tiene ISO 27001, que es lo que le piden sus clientes empresa"},
	{"organizacion_ley2", "canal_de_denuncias_obligatorio", true, "",
		"200 empleados: el canal es obligatorio desde los cincuenta"},
	{"entidad_nis2_tecnica", "es_entidad_pertinente", true, "",
		"proveedor de servicios en nube, una de las once clases del art. 1 " +
			"del Reglamento de Ejecucion 2024/2690"},
	{"organizacion_rgpd", "papel_rgpd", false, "responsable",
		"decide los fines y medios del tratamiento de los datos de sus usuarios"},
}

// motorConTodoElCorpus monta el motor igual que lo monta `plazum calendario`.
func motorConTodoElCorpus(t *testing.T, ps []*corpus.Paquete) *aplicabilidad.Motor {
	t.Helper()
	m := aplicabilidad.NuevoMotor()
	cargados := 0
	for _, p := range ps {
		if len(p.Aplicabilidad.Reglas) == 0 {
			continue
		}
		prog, errs := p.Programa()
		if len(errs) > 0 {
			t.Fatalf("las reglas de %s no compilan: %v", p.URN, errs)
		}
		if err := m.Cargar(prog); err != nil {
			t.Fatalf("el motor rechaza las reglas de %s: %v", p.URN, err)
		}
		cargados++
	}
	if cargados < 2 {
		t.Fatalf("solo se han cargado %d programas: el motor no tiene contra que derivar "+
			"y esta puerta estaria midiendo el vacio", cargados)
	}
	return m
}

// alcanzadasPor deriva las obligaciones que le alcanzan al sujeto con las
// respuestas dadas, USANDO LA TRADUCCION DEL PRODUCTO y no una escrita aqui.
func alcanzadasPor(t *testing.T, rs []respuestaDelCiso) []string {
	t.Helper()
	ps, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatalf("cargar el corpus: %v", err)
	}
	// Cada respuesta se traduce con el paquete que declara ESE atributo. Casa
	// por (entidad, atributo), que es lo que el paquete firma; no por posicion
	// ni por el orden de la lista de paquetes, que es como se rompio el
	// ayudante del puente e2e (invariante 7).
	m := motorConTodoElCorpus(t, ps)
	traducidas := 0
	for _, r := range rs {
		var suyo *corpus.Paquete
		for _, p := range ps {
			for _, e := range p.Entidades {
				if e.Nombre != r.Entidad {
					continue
				}
				for _, a := range e.Atributos {
					if a.Nombre == r.Atributo && a.Hecho != nil {
						if suyo != nil && suyo.URN != p.URN {
							t.Fatalf("%s.%s lo declara mas de un paquete (%s y %s): el "+
								"escenario no dice cual, asi que esta medida seria ambigua",
								r.Entidad, r.Atributo, suyo.URN, p.URN)
						}
						suyo = p
					}
				}
			}
		}
		if suyo == nil {
			t.Fatalf("ningun paquete declara el puente de %s.%s.\n"+
				"  El escenario del CISO nombra un atributo que el corpus ya no tiene, asi "+
				"que esta puerta estaria midiendo otra cosa. Si la pregunta se ha "+
				"renombrado, se reapunta el escenario; si ha desaparecido, hay que decir "+
				"por que y que le pasa a la cifra.", r.Entidad, r.Atributo)
		}
		hs, err := corpus.HechosDeLaEntrevista(suyo, []corpus.RespuestaDeEntrevista{{
			Entidad: r.Entidad, Instancia: "sis", Atributo: r.Atributo,
			Si: r.Si, Valor: r.Valor,
		}})
		if err != nil {
			t.Fatalf("traduciendo %s.%s: %v", r.Entidad, r.Atributo, err)
		}
		for _, h := range hs {
			m.Afirmar(h)
			traducidas++
		}
	}
	if traducidas == 0 {
		t.Fatal("ninguna respuesta del escenario ha producido un hecho: el motor no tiene " +
			"nada afirmado y su cero saldria como si fuera una medida")
	}
	if _, err := m.Evaluar(); err != nil {
		t.Fatalf("evaluando: %v", err)
	}
	vistas := map[string]bool{}
	for _, h := range m.Consultar(aplicabilidad.A("aplica",
		aplicabilidad.V("O"), aplicabilidad.C("sis"))) {
		vistas[h.Args[0]] = true
	}
	out := make([]string, 0, len(vistas))
	for k := range vistas {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestLoQueVeElCisoDeLaSaasDeDoscientos(t *testing.T) {
	if n := len(elEscenarioDelCiso); n != RespuestasDelEscenarioDelCiso {
		t.Fatalf("el escenario trae %d respuestas y la constante dice %d.\n"+
			"  El numero de respuestas se congela A PROPOSITO: sin el, la cifra de "+
			"obligaciones se sube contestando mas, que es mejorar el escenario en vez de "+
			"mejorar el producto.", n, RespuestasDelEscenarioDelCiso)
	}
	// CADA RESPUESTA CON SU PORQUE. Un escenario sin justificacion escrita no se
	// puede discutir, y lo que no se puede discutir se acepta.
	for i, r := range elEscenarioDelCiso {
		if r.PorQue == "" {
			t.Errorf("la respuesta %d (%s.%s) no dice por que es cierta de esta "+
				"organizacion", i, r.Entidad, r.Atributo)
		}
	}

	ve := alcanzadasPor(t, elEscenarioDelCiso)
	if len(ve) != ObligacionesQueVeElCiso {
		direccion := "HA SUBIDO, y hay que decir de que"
		if len(ve) < ObligacionesQueVeElCiso {
			direccion = "HA BAJADO: el producto ve MENOS que ayer con las mismas respuestas, " +
				"y eso son obligaciones que han dejado de aparecer"
		}
		t.Errorf("el CISO de la SaaS de 200 ve %d obligaciones y la constante dice %d. %s.\n"+
			"  Se reproduce con el binario:\n"+
			"    plazum alcance --respuestas \"...\" --sujeto sis --salida ciso.json\n"+
			"    plazum calendario --alcance ciso.json\n"+
			"  Las que ve: %v", len(ve), ObligacionesQueVeElCiso, direccion, ve)
	}

	// LA DESCOMPOSICION: cuanto de esto depende de que la pantalla sepa mandar
	// un VALOR. Se mide quitando las respuestas con valor y restando.
	var soloSiNo []respuestaDelCiso
	for _, r := range elEscenarioDelCiso {
		if r.Valor == "" {
			soloSiNo = append(soloSiNo, r)
		}
	}
	if len(soloSiNo) == len(elEscenarioDelCiso) {
		t.Fatal("el escenario no tiene ninguna respuesta con valor, asi que la " +
			"descomposicion no mide nada y su cero seria un cero vacio. Es M47: una rama " +
			"que ninguna entrada recorre")
	}
	sinValor := alcanzadasPor(t, soloSiNo)
	dependenDelValor := len(ve) - len(sinValor)
	if dependenDelValor != ObligacionesDelCisoQueSoloLleganConValor {
		t.Errorf("de las %d que ve, %d dependen de una respuesta con VALOR y la constante "+
			"dice %d.\n"+
			"  Esta cifra es la que dice cuanto del titular es merito de la pantalla y "+
			"cuanto del corpus. Un numero que no dice de donde viene su ventaja es un "+
			"numero que la esconde.", len(ve), dependenDelValor,
			ObligacionesDelCisoQueSoloLleganConValor)
	}
	t.Logf("el CISO de la SaaS de 200 ve %d obligaciones con %d respuestas; "+
		"%d de ellas dependen de que la pantalla sepa mandar un valor (el resto es corpus)",
		len(ve), len(elEscenarioDelCiso), dependenDelValor)
}
