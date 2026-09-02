package plazum

import (
	"sort"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/aplicabilidad"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// EL PILOTO DEL PUENTE: se mide si el diseno mueve el numero, con UN paquete.
//
// # Por que un piloto y no los treinta
//
// Declarar el puente en los 30 paquetes cuesta treinta veces mas que en uno, y
// si el diseno esta mal se descubre igual. Asi que se pilota con el que mas
// preguntas tiene de los 12 marcos de la v1 (diecisiete) y se mide. Si el
// numero no se mueve, el diseno esta mal y ha costado un paquete.
//
// # Lo que este test SI puede afirmar, y antes no
//
// El 02-09-2026 se intento contar cuantas obligaciones derivaria un alcance
// sacado de la entrevista y NO SE PUDO: para contarlo hay que decir que hecho
// produce cada respuesta, y esa traduccion no existia en ninguna parte, asi que
// cualquier numero habria salido de una regla inventada para medir. Se
// descartaron dos resultados incompatibles por eso.
//
// Ahora la traduccion la declara el paquete (`hecho` en cada atributo, ver
// nucleo/corpus/puente.go) y la valida el linter contra las reglas del propio
// paquete. Asi que el numero de abajo **no depende de ningun mapeo inventado
// por el test**: sale de lo que el paquete afirma de si mismo.
//
// # Y lo que sigue sin poder afirmar, dicho
//
// Cuantas obligaciones derivan depende de QUE CONTESTE el operador: con
// `ambito` a sector publico salen unas y con sector privado contratista salen
// otras, y eso no es una limitacion de la medida, es lo que hace un motor de
// aplicabilidad. Por eso el escenario va escrito y con nombre, y el numero se
// lee siempre junto a el.

// escenarioDelPiloto son las respuestas de UN caso concreto. Va declarado y con
// nombre porque el numero depende de el: un porcentaje de derivacion sin decir
// que se contesto es un numero sin sujeto.
//
// Se elige el caso mas comun del ENS y el que mas obligaciones deberia
// encender: sector publico, con datos personales, con servicios externalizados
// y en la nube, y una informacion y un servicio de nivel alto.
type respuestaDelPiloto struct {
	Entidad   string // el tipo de entidad, tal y como lo declara el paquete
	Instancia string // a que sujeto concreto se refiere
	Atributo  string
	Si        bool   // para los booleanos
	Valor     string // para los que llevan valor
}

var escenarioDelPiloto = []respuestaDelPiloto{
	{Entidad: "sistema", Instancia: "sis", Atributo: "ambito", Valor: "sector_publico"},
	{Entidad: "sistema", Instancia: "sis", Atributo: "trata_datos_personales", Si: true},
	{Entidad: "sistema", Instancia: "sis", Atributo: "servicios_externalizados", Si: true},
	{Entidad: "sistema", Instancia: "sis", Atributo: "usa_servicios_en_la_nube", Si: true},
	{Entidad: "sistema", Instancia: "sis", Atributo: "preexistente_al_ens", Si: true},

	{Entidad: "informacion", Instancia: "inf", Atributo: "manejada_por_el_sistema", Valor: "sis"},
	{Entidad: "informacion", Instancia: "inf", Atributo: "nivel_confidencialidad", Valor: "ALTO"},
	{Entidad: "informacion", Instancia: "inf", Atributo: "nivel_integridad", Valor: "ALTO"},
	{Entidad: "informacion", Instancia: "inf", Atributo: "nivel_disponibilidad", Valor: "ALTO"},
	{Entidad: "informacion", Instancia: "inf", Atributo: "nivel_autenticidad", Valor: "ALTO"},
	{Entidad: "informacion", Instancia: "inf", Atributo: "nivel_trazabilidad", Valor: "ALTO"},

	{Entidad: "servicio", Instancia: "srv", Atributo: "prestado_por_el_sistema", Valor: "sis"},
	{Entidad: "servicio", Instancia: "srv", Atributo: "nivel_confidencialidad", Valor: "MEDIO"},
	{Entidad: "servicio", Instancia: "srv", Atributo: "nivel_integridad", Valor: "MEDIO"},
	{Entidad: "servicio", Instancia: "srv", Atributo: "nivel_disponibilidad", Valor: "MEDIO"},
	{Entidad: "servicio", Instancia: "srv", Atributo: "nivel_autenticidad", Valor: "MEDIO"},
	{Entidad: "servicio", Instancia: "srv", Atributo: "nivel_trazabilidad", Valor: "MEDIO"},
}

// ObligacionesQueDerivaElPiloto es el resultado de la medida, congelado por
// igualdad exacta. Es EL numero que decide si el diseno del puente sirve.
//
// Si sube, el puente esta llegando mas lejos y hay que decirlo. Si baja sin que
// nadie borre reglas, algo se ha desconectado en silencio, que es exactamente
// la familia de fallos que este bloque persigue.
// MEDIDO, NO ESTIMADO: 25. La primera version de esta constante llevaba un 24
// escrito a ojo y la puerta la corrigio en el estreno, que es exactamente para
// lo que estaba.
//
// Y el 25 dice mas de lo que parece: NO son 25 obligaciones del paquete piloto.
// Son 25 de TODO el corpus, porque los predicados se comparten entre paquetes.
// Contestar que el sistema trata datos personales enciende obligaciones del
// RGPD, y describir la informacion que maneja enciende una de interoperabilidad.
// Eso es el corpus funcionando como se diseno, y es lo que un recuento por
// paquete no habria visto.
const ObligacionesQueDerivaElPiloto = 25

// hechosDelPuente traduce las respuestas a hechos USANDO LA DECLARACION DEL
// PAQUETE, no una tabla escrita aqui.
//
// Esta funcion es la mitad que faltaba del producto, escrita en un test para
// medirla antes de construirla: cuando la pantalla aprenda a preguntar valores
// (la parte A), lo que hara sera exactamente esto.
func hechosDelPuente(t *testing.T, p *corpus.Paquete,
	respuestas []respuestaDelPiloto) []aplicabilidad.Hecho {

	t.Helper()
	// El indice de atributos declarados, por entidad y nombre.
	type clave struct{ entidad, atributo string }
	decl := map[clave]*corpus.HechoDeAtributo{}
	for _, e := range p.Entidades {
		for _, a := range e.Atributos {
			if a.Hecho != nil {
				h := *a.Hecho
				decl[clave{e.Nombre, a.Nombre}] = &h
			}
		}
	}
	if len(decl) == 0 {
		t.Fatalf("%s no declara el puente en ningun atributo: este test mediria el vacio",
			p.URN)
	}

	var out []aplicabilidad.Hecho
	for _, r := range respuestas {
		h, hay := decl[clave{r.Entidad, r.Atributo}]
		if !hay {
			t.Fatalf("el escenario contesta %s.%s y el paquete no declara el puente de ese "+
				"atributo. O el escenario esta viejo, o falta declararlo", r.Entidad, r.Atributo)
		}
		switch h.Forma {
		case corpus.PuenteNoLlegaAlMotor:
			t.Fatalf("el escenario contesta %s.%s, que el paquete declara como %q. Contestar "+
				"algo que no llega al motor y contarlo como derivacion seria inflar el numero",
				r.Entidad, r.Atributo, corpus.PuenteNoLlegaAlMotor)
		case corpus.PuenteAfirmaSi:
			// UN «NO» NO AFIRMA NADA. Es la regla del puente y aqui se ve por
			// que importa: si un «no» afirmara algo, el escenario estaria
			// metiendo en el motor una afirmacion que el operador no ha hecho.
			if r.Si {
				out = append(out, aplicabilidad.H(h.Predicado, r.Instancia))
			}
		case corpus.PuenteConValor:
			if strings.TrimSpace(r.Valor) == "" {
				t.Fatalf("%s.%s lleva valor y el escenario no lo da", r.Entidad, r.Atributo)
			}
			out = append(out, aplicabilidad.H(h.Predicado, r.Instancia, r.Valor))
		default:
			t.Fatalf("forma desconocida %q en %s.%s", h.Forma, r.Entidad, r.Atributo)
		}
	}
	return out
}

// TestElPuenteDeclaradoDerivaObligacionesDeVerdad es la medida del piloto.
//
// Monta el motor EXACTAMENTE como lo monta `plazum calendario` (cargar los
// programas de todos los paquetes con reglas, afirmar los hechos, evaluar), le
// da los hechos que salen de la declaracion del paquete y cuenta.
func TestElPuenteDeclaradoDerivaObligacionesDeVerdad(t *testing.T) {
	ps, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatalf("cargar el corpus: %v", err)
	}
	piloto := paqueteConPuente(t, ps)

	motor := aplicabilidad.NuevoMotor()
	cargados := 0
	for _, p := range ps {
		if len(p.Aplicabilidad.Reglas) == 0 {
			continue
		}
		prog, errs := p.Programa()
		if len(errs) > 0 {
			t.Fatalf("las reglas de %s no compilan: %v", p.URN, errs)
		}
		if err := motor.Cargar(prog); err != nil {
			t.Fatalf("el motor rechaza las reglas de %s: %v", p.URN, err)
		}
		cargados++
	}
	if cargados < 2 {
		t.Fatalf("solo se han cargado %d programas: el motor no tiene contra que derivar",
			cargados)
	}

	for _, h := range hechosDelPuente(t, piloto, escenarioDelPiloto) {
		hecho := h
		hecho.Procedencia = "declarado en la entrevista"
		motor.Afirmar(hecho)
	}
	if _, err := motor.Evaluar(); err != nil {
		t.Fatalf("evaluando: %v", err)
	}

	// LAS OBLIGACIONES QUE LE ALCANZAN AL SISTEMA. Se consulta igual que
	// `aplicablesDe`, por el sujeto del escenario.
	vistas := map[string]bool{}
	for _, h := range motor.Consultar(aplicabilidad.A("aplica",
		aplicabilidad.V("O"), aplicabilidad.C("sis"))) {
		vistas[h.Args[0]] = true
	}
	derivadas := len(vistas)

	if derivadas != ObligacionesQueDerivaElPiloto {
		t.Errorf("el puente declarado de %s deriva %d obligaciones y la constante dice %d.\n"+
			"  Si ha SUBIDO, el puente llega mas lejos y hay que decirlo aqui.\n"+
			"  Si ha BAJADO sin que nadie borre reglas, algo se ha desconectado en silencio.\n"+
			"  Derivadas: %v", piloto.URN, derivadas, ObligacionesQueDerivaElPiloto,
			ordenadas(vistas))
	}

	// EL PILOTO TIENE QUE MOVER EL NUMERO. Si con el puente declarado y la
	// entrevista entera contestada no deriva NADA, el diseno esta mal y es lo
	// que este piloto existe para descubrir barato.
	if derivadas == 0 {
		t.Fatal("con el puente declarado y el escenario entero contestado no deriva ni una " +
			"obligacion. El diseno del puente no sirve, y descubrirlo con un paquete en vez " +
			"de con treinta es exactamente para lo que estaba el piloto")
	}
	t.Logf("piloto %s: %d obligaciones derivadas con el escenario declarado, desde %d "+
		"hechos que salen de la declaracion del paquete",
		piloto.URN, derivadas, len(hechosDelPuente(t, piloto, escenarioDelPiloto)))
}

// paqueteConPuente encuentra el paquete piloto POR UNA PROPIEDAD, no por su
// nombre: escribir aqui el identificador de una norma rompe el invariante 2.
// Hoy solo hay uno que declare el puente, y el test exige que siga siendo asi
// mientras dure el piloto.
func paqueteConPuente(t *testing.T, ps []*corpus.Paquete) *corpus.Paquete {
	t.Helper()
	var con []*corpus.Paquete
	for _, p := range ps {
		if p.DeclaraPuente() {
			con = append(con, p)
		}
	}
	switch len(con) {
	case 0:
		t.Fatal("ningun paquete declara el puente. O se ha borrado el piloto, o el lector " +
			"no esta viendo el bloque `hecho`")
	case 1:
		return con[0]
	default:
		// NO ES UN FALLO, ES UNA DECISION QUE HAY QUE TOMAR. En cuanto haya un
		// segundo paquete con puente, esta medida deja de ser «el piloto» y
		// pasa a ser «los paquetes que lo declaran», con un escenario por cada
		// uno. Se para aqui para que nadie lo amplie sin darse cuenta.
		var urns []string
		for _, p := range con {
			urns = append(urns, p.URN)
		}
		sort.Strings(urns)
		t.Fatalf("hay %d paquetes con puente declarado (%v) y este test mide UN piloto con UN "+
			"escenario. El piloto ha terminado: toca decidir si el puente pasa a obligatorio "+
			"y darle escenario a cada paquete", len(con), urns)
	}
	return nil
}

func ordenadas(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// CONTROL POSITIVO DEL TRADUCTOR: un «no» no afirma nada.
//
// Es la regla que separa este puente de uno que se inventa datos, y sin este
// caso ninguna entrada la recorre: todo el escenario contesta que si. Un
// descargo que nadie recorre es un descargo que no existe (M47).
func TestUnNoDeLaEntrevistaNoAfirmaNadaEnElMotor(t *testing.T) {
	ps, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatalf("cargar el corpus: %v", err)
	}
	piloto := paqueteConPuente(t, ps)

	conSi := hechosDelPuente(t, piloto, []respuestaDelPiloto{
		{Entidad: "sistema", Instancia: "sis", Atributo: "trata_datos_personales", Si: true},
	})
	conNo := hechosDelPuente(t, piloto, []respuestaDelPiloto{
		{Entidad: "sistema", Instancia: "sis", Atributo: "trata_datos_personales", Si: false},
	})
	if len(conSi) != 1 {
		t.Fatalf("un «si» a un booleano tiene que afirmar exactamente un hecho, y afirma %d",
			len(conSi))
	}
	if len(conNo) != 0 {
		t.Errorf("un «no» ha afirmado %v. En este motor la ausencia de un hecho no es su "+
			"negacion, y hacer que un «no» afirme algo mete en el expediente una afirmacion "+
			"que el operador no ha hecho", conNo)
	}
}
