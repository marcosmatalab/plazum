package plazum

import (
	"sort"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/aplicabilidad"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// Las reglas de aplicabilidad del corpus, ejecutadas de verdad contra el motor.
//
// Por que este test existe y no basta con que el linter las lea. El linter dice
// que las reglas se PARSEAN; esto dice que DERIVAN lo que tienen que derivar. Un
// paquete puede tener veintitantas reglas impecables que no disparan ninguna
// obligacion porque el predicado del cuerpo no lo produce nadie, y el linter da
// verde igual. Sin este test, "las reglas viven en el paquete" seria cierto y
// vacio a la vez.
//
// Este fichero SI nombra normas reales, y esta en la raiz por eso: la raiz es
// donde el codigo se encuentra con paquetes/. Los tests de nucleo/ no pueden,
// que es lo que vigila TestNingunaNormaCableada.

// motorConElCorpus carga el corpus publicado y devuelve un motor con los
// programas de los paquetes que declaran reglas.
func motorConElCorpus(t *testing.T) (*aplicabilidad.Motor, int) {
	t.Helper()
	ps, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatalf("cargar el corpus: %v", err)
	}
	m := aplicabilidad.NuevoMotor()
	conReglas := 0
	for _, p := range ps {
		if len(p.Aplicabilidad.Reglas) == 0 {
			continue
		}
		prog, errs := p.Programa()
		if len(errs) != 0 {
			t.Fatalf("%s: el paquete no produce un programa valido: %v", p.URN, errs)
		}
		if err := m.Cargar(prog); err != nil {
			t.Fatalf("%s: el motor rechaza el programa: %v", p.URN, err)
		}
		conReglas++
	}
	if conReglas == 0 {
		t.Fatal("ningun paquete publicado declara reglas de aplicabilidad. Si eso es asi, " +
			"la aplicabilidad sigue viviendo en codigo Go y el invariante 2 es falso")
	}
	return m, conReglas
}

func aplicablesA(t *testing.T, m *aplicabilidad.Motor, sujeto string) []string {
	t.Helper()
	var out []string
	for _, h := range m.Consultar(aplicabilidad.A("aplica",
		aplicabilidad.V("O"), aplicabilidad.C(sujeto))) {
		out = append(out, h.Args[0])
	}
	sort.Strings(out)
	return out
}

// Un ayuntamiento con sede electronica: sector publico, categoria MEDIA,
// trata datos personales, tiene servicios externalizados y no usa nube.
//
// Se comprueban las dos direcciones, y la segunda es la que de verdad prueba
// algo: lo que TIENE que aplicarle, y lo que NO puede aplicarle. Un motor que
// derivara todo pasaria la primera mitad sin despeinarse.
func TestLasReglasDelCorpusDerivanLaAplicabilidadDeUnSujetoReal(t *testing.T) {
	m, _ := motorConElCorpus(t)
	for _, h := range []aplicabilidad.Hecho{
		aplicabilidad.H("ambito", "sede", "sector_publico"),
		aplicabilidad.H("categoria", "sede", "MEDIA"),
		aplicabilidad.H("trata_datos_personales", "sede"),
		aplicabilidad.H("servicios_externalizados", "sede"),
	} {
		h.Procedencia = "alcance declarado por el sujeto"
		m.Afirmar(h)
	}
	if _, err := m.Evaluar(); err != nil {
		t.Fatalf("evaluar: %v", err)
	}
	got := aplicablesA(t, m, "sede")
	tiene := map[string]bool{}
	for _, o := range got {
		tiene[o] = true
	}

	debe := []string{
		"ens.art31.auditoria_ordinaria",                           // art. 31.1, categoria MEDIA
		"ens.its_conformidad.certificacion_media_alta",            // ITS Conformidad III.3
		"ens.its_conformidad.publicidad_certificacion_media_alta", // ITS Conformidad V.3
		"ens.art3.2.analisis_riesgos_datos_personales",            // art. 3.2
		"ens.anexoII.mp.info.1",                                   // anexo II 5.7.1
		"ens.art13.5.poc_servicios_externalizados",                // art. 13.5
		"ens.art16.2.exigencia_a_proveedores_de_seguridad",        // art. 16.2
		"ens.anexoII.op.ext.1",                                    // anexo II 4.4.1
		"ens.ines.informe_anual",                                  // ITS Informe del Estado, III.2
		"ens.anexoI.reevaluacion_de_la_categoria",                 // anexo I.1
		"ens.art38.1.determinacion_de_la_conformidad",             // art. 38.1
		"ens.art40.2.determinacion_de_la_categoria",               // art. 40.2
	}
	for _, o := range debe {
		if !tiene[o] {
			t.Errorf("no se ha derivado %s, que si aplica. Derivadas: %v", o, got)
		}
	}

	noDebe := map[string]string{
		"ens.its_conformidad.autoevaluacion_basica":         "es de categoria BASICA (ITS Conformidad III.2) y este sistema es MEDIA",
		"ens.its_conformidad.publicidad_declaracion_basica": "es de categoria BASICA (ITS Conformidad IV.3)",
		"ens.anexoII.op.nub.1":                              "no usa servicios en la nube (anexo II 4.5.1)",
		"ens.art2.3.politica_sector_privado":                "es del sector publico, no contratista (art. 2.3)",
		"ens.art2.3.pliegos_conformidad":                    "es del sector publico, no contratista (art. 2.3)",
		"ens.art33.7.notificacion_al_incibe_sector_privado": "es del sector publico (art. 33.7)",
	}
	for o, porQue := range noDebe {
		if tiene[o] {
			t.Errorf("se ha derivado %s y NO aplica: %s. Una obligacion de mas es un coste "+
				"de mas que el cliente paga sin deberlo", o, porQue)
		}
	}
}

// La agregacion del anexo I: la categoria NO se declara, se calcula como el
// maximo del nivel de cada dimension sobre cada informacion y servicio, y de la
// categoria cuelga la auditoria del art. 31.
//
// Es la cadena entera en un solo test: hecho del sujeto, agregado, regla
// intermedia, obligacion. Si alguna pieza se rompe, aqui se ve.
func TestLaCategoriaSeDerivaDelMaximoDeLasDimensiones(t *testing.T) {
	m, _ := motorConElCorpus(t)
	for _, h := range []aplicabilidad.Hecho{
		aplicabilidad.H("ambito", "padron", "sector_publico"),
		aplicabilidad.H("maneja", "padron", "datos-de-empadronamiento"),
		aplicabilidad.H("nivel_dimension", "datos-de-empadronamiento", "confidencialidad", "MEDIO"),
		aplicabilidad.H("nivel_dimension", "datos-de-empadronamiento", "integridad", "BAJO"),
		aplicabilidad.H("nivel_dimension", "datos-de-empadronamiento", "disponibilidad", "BAJO"),
	} {
		m.Afirmar(h)
	}
	if _, err := m.Evaluar(); err != nil {
		t.Fatalf("evaluar: %v", err)
	}

	cat := m.Consultar(aplicabilidad.A("categoria", aplicabilidad.C("padron"), aplicabilidad.V("C")))
	if len(cat) != 1 {
		t.Fatalf("se esperaba una sola categoria derivada y hay %d: %v", len(cat), cat)
	}
	if cat[0].Args[1] != "MEDIA" {
		t.Fatalf("el maximo de MEDIO, BAJO y BAJO es MEDIO, o sea categoria MEDIA "+
			"(RD 311/2022 anexo I apartado 2), y salio %s", cat[0].Args[1])
	}

	tiene := map[string]bool{}
	for _, o := range aplicablesA(t, m, "padron") {
		tiene[o] = true
	}
	if !tiene["ens.art31.auditoria_ordinaria"] {
		t.Error("de la categoria MEDIA cuelga la auditoria bienal del art. 31.1, y no se derivo")
	}
	if tiene["ens.its_conformidad.autoevaluacion_basica"] {
		t.Error("la autoevaluacion es de categoria BASICA; con MEDIA no aplica")
	}
}

// Control negativo de los dos de arriba: si el sujeto cambia, la aplicabilidad
// cambia. Sin esto, un motor que derivara siempre lo mismo pasaria los tests
// anteriores y no probaria que las reglas MIRAN los hechos.
func TestCambiarUnHechoCambiaLaAplicabilidad(t *testing.T) {
	derivar := func(nivel string) map[string]bool {
		m, _ := motorConElCorpus(t)
		m.Afirmar(aplicabilidad.H("ambito", "s", "sector_publico"))
		m.Afirmar(aplicabilidad.H("maneja", "s", "i"))
		m.Afirmar(aplicabilidad.H("nivel_dimension", "i", "confidencialidad", nivel))
		if _, err := m.Evaluar(); err != nil {
			t.Fatalf("evaluar: %v", err)
		}
		out := map[string]bool{}
		for _, o := range aplicablesA(t, m, "s") {
			out[o] = true
		}
		return out
	}
	bajo, medio := derivar("BAJO"), derivar("MEDIO")

	if !bajo["ens.its_conformidad.autoevaluacion_basica"] {
		t.Error("con la unica dimension en BAJO la categoria es BASICA y toca autoevaluacion")
	}
	if bajo["ens.art31.auditoria_ordinaria"] {
		t.Error("con categoria BASICA no es exigible la auditoria del art. 31: art. 31.2")
	}
	if !medio["ens.art31.auditoria_ordinaria"] {
		t.Error("con la dimension en MEDIO la categoria es MEDIA y toca auditoria bienal")
	}
	if medio["ens.its_conformidad.autoevaluacion_basica"] {
		t.Error("con categoria MEDIA no toca autoevaluacion, toca certificacion")
	}
}

// Toda obligacion derivada tiene que poder explicarse: que regla la derivo y de
// que articulo sale esa regla. Es lo que responde "por que me aplica esto", que
// es la pregunta que separa esto de una lista de controles.
func TestTodaObligacionDerivadaSabeDecirPorQue(t *testing.T) {
	m, _ := motorConElCorpus(t)
	m.Afirmar(aplicabilidad.H("ambito", "s", "sector_publico"))
	m.Afirmar(aplicabilidad.H("categoria", "s", "ALTA"))
	if _, err := m.Evaluar(); err != nil {
		t.Fatalf("evaluar: %v", err)
	}
	derivadas := m.Consultar(aplicabilidad.A("aplica",
		aplicabilidad.V("O"), aplicabilidad.C("s")))
	if len(derivadas) == 0 {
		t.Fatal("no se derivo ninguna obligacion, asi que este test no prueba nada")
	}
	for _, h := range derivadas {
		exp := m.Explicar(h)
		if strings.TrimSpace(exp) == "" {
			t.Errorf("%s se deriva y no sabe decir por que", h)
		}
	}
}

// EL AI ACT, y aqui la direccion que importa es la SEGUNDA. Este marco reparte
// las cuatro obligaciones de transparencia del art. 50 entre DOS papeles
// distintos, y esa es toda la diferencia entre ensenarle a alguien cuatro
// obligaciones o ensenarle dos.
//
// Quien despliega un sistema de reconocimiento de emociones no fabrica nada:
// los apartados 1 y 2 del art. 50 son del PROVEEDOR, y ensenarselos es cobrarle
// un trabajo que la norma no le pide. Y al reves: el art. 73 es del proveedor de
// alto riesgo, asi que a quien solo despliega no le aparece.
func TestElAiActRepartePorPapelLasObligacionesDeTransparencia(t *testing.T) {
	m, _ := motorConElCorpus(t)
	for _, h := range []aplicabilidad.Hecho{
		// Un hospital que USA un sistema de alto riesgo del anexo III y ademas
		// despliega uno de reconocimiento de emociones. No fabrica ninguno.
		aplicabilidad.H("papel_ia", "hospital", "responsable_del_despliegue"),
		aplicabilidad.H("riesgo_ia", "hospital", "alto_anexo_iii"),
		// Y el fabricante del sistema que el hospital usa.
		aplicabilidad.H("papel_ia", "fabricante", "proveedor"),
		aplicabilidad.H("riesgo_ia", "fabricante", "alto_anexo_iii"),
	} {
		h.Procedencia = "papel declarado por el sujeto"
		m.Afirmar(h)
	}
	if _, err := m.Evaluar(); err != nil {
		t.Fatalf("evaluar: %v", err)
	}

	tieneDe := func(sujeto string) map[string]bool {
		out := map[string]bool{}
		for _, o := range aplicablesA(t, m, sujeto) {
			out[o] = true
		}
		return out
	}
	hospital, fabricante := tieneDe("hospital"), tieneDe("fabricante")

	debe := map[string][]string{
		"hospital": {
			"aiact.art50_3.reconocimiento_de_emociones_y_categorizacion_biometrica", // art. 50.3
			"aiact.art50_4.ultrasuplantacion_y_texto_de_interes_publico",            // art. 50.4
			"aiact.art50.informacion_antes_de_la_primera_interaccion",               // art. 50.5, alcanza a los dos
		},
		"fabricante": {
			"aiact.art50_1.interaccion_directa_con_personas",          // art. 50.1
			"aiact.art50_2.marcado_de_contenido_sintetico",            // art. 50.2
			"aiact.art111_4.marcado_de_lo_ya_comercializado",          // art. 111.4
			"aiact.art50.informacion_antes_de_la_primera_interaccion", // art. 50.5
			"aiact.art73.notificacion_de_incidente_grave",             // art. 73.1
			"aiact.art73_6.investigacion_posterior_al_incidente",      // art. 73.6
		},
	}
	for sujeto, ids := range debe {
		tiene := hospital
		if sujeto == "fabricante" {
			tiene = fabricante
		}
		for _, id := range ids {
			if !tiene[id] {
				t.Errorf("%s: no se ha derivado %s, que si aplica", sujeto, id)
			}
		}
	}

	// La direccion contraria, con el articulo de cada exclusion.
	noDebe := map[string]map[string]string{
		"hospital": {
			"aiact.art50_1.interaccion_directa_con_personas": "el art. 50.1 obliga al PROVEEDOR, y este sujeto solo despliega",
			"aiact.art50_2.marcado_de_contenido_sintetico":   "el art. 50.2 obliga al PROVEEDOR: el marcado legible por maquina se pone al generar la salida",
			"aiact.art73.notificacion_de_incidente_grave":    "el art. 73.1 obliga al PROVEEDOR de sistemas de alto riesgo introducidos en el mercado",
			"aiact.art111_4.marcado_de_lo_ya_comercializado": "el art. 111.4 sirve al art. 50.2, que es del PROVEEDOR: quien solo despliega no marca la salida de un modelo que no ha entrenado",
		},
		"fabricante": {
			"aiact.art50_3.reconocimiento_de_emociones_y_categorizacion_biometrica": "el art. 50.3 obliga al RESPONSABLE DEL DESPLIEGUE, que es quien expone a las personas",
			"aiact.art50_4.ultrasuplantacion_y_texto_de_interes_publico":            "el art. 50.4 obliga al RESPONSABLE DEL DESPLIEGUE, que es quien publica",
		},
	}
	for sujeto, mapa := range noDebe {
		tiene := hospital
		if sujeto == "fabricante" {
			tiene = fabricante
		}
		for id, porQue := range mapa {
			if tiene[id] {
				t.Errorf("%s: se ha derivado %s y NO aplica: %s. Una obligacion de mas es un "+
					"coste de mas que el cliente paga sin deberlo", sujeto, id, porQue)
			}
		}
	}
}

// EL AMBITO DE 2024/2690 NO ES EL AMBITO DE NIS2, y esta es la direccion que
// importa. El art. 1 del Reglamento de Ejecucion da una lista CERRADA de once
// tipos (DNS, registros de dominio de primer nivel, nube, centros de datos, CDN,
// servicios gestionados, seguridad gestionada, mercados en linea, motores de
// busqueda, redes sociales y prestadores de servicios de confianza) a los que
// llama "entidades pertinentes". Una entidad esencial o importante de NIS2 que
// no sea de esos once tipos NO tiene los requisitos tecnicos del anexo.
//
// Es exactamente el fallo que un catalogo de controles comete y que aqui no se
// puede cometer: dar por bueno que "si te aplica NIS2, te aplica todo lo de
// NIS2". Un hospital es entidad esencial y no es ninguno de los once.
func TestElReglamentoTecnicoDeNis2NoAlcanzaATodaEntidadDeNis2(t *testing.T) {
	m, _ := motorConElCorpus(t)
	for _, h := range []aplicabilidad.Hecho{
		// Un proveedor de servicios de computacion en nube: es de los once.
		aplicabilidad.H("papel_nis2_tecnica", "proveedor_nube", "entidad_pertinente"),
		aplicabilidad.H("designado", "proveedor_nube", "entidad_esencial_o_importante"),
		// Un hospital: entidad esencial de NIS2 y NINGUNO de los once tipos.
		aplicabilidad.H("designado", "hospital", "entidad_esencial_o_importante"),
	} {
		h.Procedencia = "papel declarado por el sujeto"
		m.Afirmar(h)
	}
	if _, err := m.Evaluar(); err != nil {
		t.Fatalf("evaluar: %v", err)
	}

	tieneDe := func(sujeto string) map[string]bool {
		out := map[string]bool{}
		for _, o := range aplicablesA(t, m, sujeto) {
			out[o] = true
		}
		return out
	}
	nube, hospital := tieneDe("proveedor_nube"), tieneDe("hospital")

	delAnexo := []string{
		"nis2tec.anexo.1_1_2.revision_de_la_politica",
		"nis2tec.anexo.2_1_4.revision_de_la_evaluacion_de_riesgos",
		"nis2tec.anexo.10_1_3.revision_de_la_asignacion_de_personal",
	}

	// DIRECCION 1: al proveedor de nube le alcanzan los tres puntos del anexo.
	for _, id := range delAnexo {
		if !nube[id] {
			t.Errorf("proveedor de nube: no se ha derivado %s, y el art. 1 lo nombra "+
				"expresamente entre las entidades pertinentes", id)
		}
	}

	// DIRECCION 2, la que muerde: al hospital NO, con el articulo de la exclusion.
	for _, id := range delAnexo {
		if hospital[id] {
			t.Errorf("hospital: se ha derivado %s y NO aplica. El art. 1 del Reglamento de "+
				"Ejecucion (UE) 2024/2690 alcanza a once tipos de infraestructura digital y "+
				"servicios de confianza, y un hospital no es ninguno: es entidad esencial de "+
				"la Directiva (UE) 2022/2555 por el anexo I, que es otra cosa. Una obligacion "+
				"de mas es un coste de mas que el cliente paga sin deberlo", id)
		}
	}

	// Y la comprobacion que impide que este test se cumpla solo: el hospital
	// tiene que tener ALGO, o estariamos midiendo un motor vacio.
	if len(hospital) == 0 {
		t.Fatal("el hospital no ha derivado ninguna obligacion, asi que la direccion 2 " +
			"se cumple sola y no demuestra nada")
	}
}

// DORA excluye por TAMANO y por REGIMEN, y las dos exclusiones se comprueban en
// las dos direcciones.
//
// Es la primera vez que este corpus usa la NEGACION del dialecto sobre una
// norma de verdad, y es donde mas facil es equivocarse: una exclusion mal
// escrita no da error, da obligaciones de mas, que el cliente paga sin deberlas.
//
// Tres apartados de DORA excluyen expresamente a las microempresas (arts. 8.7,
// 24.6 y 26.1) y uno excluye ademas a las entidades del art. 16.1 parrafo
// primero, las del marco simplificado (art. 26.1). El resto alcanza a toda
// entidad financiera.
func TestDoraExcluyeALaMicroempresaYAlMarcoSimplificado(t *testing.T) {
	m, _ := motorConElCorpus(t)
	for _, h := range []aplicabilidad.Hecho{
		// Un banco mediano: entidad financiera y nada mas.
		aplicabilidad.H("designado", "banco", "entidad_financiera"),
		// Una microempresa financiera: menos de diez personas y <= 2 M EUR.
		aplicabilidad.H("designado", "micro", "entidad_financiera"),
		aplicabilidad.H("designado", "micro", "microempresa_dora"),
		// Una entidad del marco simplificado del art. 16.1, que NO es micro.
		aplicabilidad.H("designado", "simplificada", "entidad_financiera"),
		aplicabilidad.H("designado", "simplificada", "marco_simplificado_dora"),
	} {
		h.Procedencia = "papel declarado por el sujeto"
		m.Afirmar(h)
	}
	if _, err := m.Evaluar(); err != nil {
		t.Fatalf("evaluar: %v", err)
	}
	tieneDe := func(sujeto string) map[string]bool {
		out := map[string]bool{}
		for _, o := range aplicablesA(t, m, sujeto) {
			out[o] = true
		}
		return out
	}
	banco, micro, simple := tieneDe("banco"), tieneDe("micro"), tieneDe("simplificada")

	soloGrandes := map[string]string{
		"dora.art8_7.evaluacion_del_riesgo_de_sistemas_heredados":                 "art. 8.7",
		"dora.art24_6.pruebas_de_los_sistemas_que_sustentan_funciones_esenciales": "art. 24.6",
		"dora.art26_1.pruebas_de_penetracion_basadas_en_amenazas":                 "art. 26.1",
	}
	todas := map[string]string{
		"dora.art8_1.revision_de_la_clasificacion_de_activos":            "art. 8.1",
		"dora.art8_2.revision_de_los_escenarios_de_riesgo":               "art. 8.2",
		"dora.art13_5.informe_anual_de_hallazgos_al_organo_de_direccion": "art. 13.5",
		"dora.art28_3.comunicacion_anual_de_acuerdos_tic":                "art. 28.3",
	}

	// DIRECCION 1: al banco le alcanzan las tres restringidas y las generales.
	for id, art := range soloGrandes {
		if !banco[id] {
			t.Errorf("banco: no se ha derivado %s (%s), y no es microempresa", id, art)
		}
	}
	for id, art := range todas {
		for quien, s := range map[string]map[string]bool{"banco": banco, "micro": micro,
			"simplificada": simple} {
			if !s[id] {
				t.Errorf("%s: no se ha derivado %s (%s), que alcanza a toda entidad financiera",
					quien, id, art)
			}
		}
	}

	// DIRECCION 2, la que muerde: a la microempresa NO le alcanzan las tres.
	for id, art := range soloGrandes {
		if micro[id] {
			t.Errorf("microempresa: se ha derivado %s y el %s dice «las entidades financieras "+
				"QUE NO SEAN MICROEMPRESAS». Una obligacion de mas es un coste de mas que el "+
				"cliente paga sin deberlo", id, art)
		}
	}
	// Y a la del marco simplificado le alcanzan las de tamano pero NO el art. 26.1.
	if !simple["dora.art8_7.evaluacion_del_riesgo_de_sistemas_heredados"] {
		t.Error("marco simplificado: el art. 8.7 solo excluye microempresas, y esta no lo es")
	}
	if simple["dora.art26_1.pruebas_de_penetracion_basadas_en_amenazas"] {
		t.Error("marco simplificado: se ha derivado el art. 26.1, que excluye expresamente a " +
			"las entidades del art. 16.1 parrafo primero ADEMAS de a las microempresas. Las " +
			"dos exclusiones son distintas y hay que respetar las dos")
	}

	// Y el suelo que impide que esto se cumpla solo: los tres sujetos tienen
	// que tener ALGO, o estariamos midiendo un motor vacio.
	for quien, s := range map[string]map[string]bool{"banco": banco, "micro": micro,
		"simplificada": simple} {
		if len(s) == 0 {
			t.Fatalf("%s no ha derivado ni una obligacion: el motor esta vacio y estas dos "+
				"direcciones no demuestran nada", quien)
		}
	}
}

// MiCA reparte por PAPEL y ademas por UMBRAL, y las dos se comprueban.
//
// El art. 22.1 no alcanza a todo emisor de fichas referenciadas a activos: solo
// a los que emitan por encima de 100 000 000 EUR. Un umbral mal escrito no da
// error, da una comunicacion trimestral a la autoridad que la entidad no debe.
func TestMicaRepartePorPapelYPorElUmbralDeCienMillones(t *testing.T) {
	m, _ := motorConElCorpus(t)
	for _, h := range []aplicabilidad.Hecho{
		// Emisor de fichas referenciadas a activos, por debajo del umbral.
		aplicabilidad.H("papel_mica", "emisor_pequeno", "emisor_de_fichas_referenciadas_a_activos"),
		// El mismo papel, por encima del umbral.
		aplicabilidad.H("papel_mica", "emisor_grande", "emisor_de_fichas_referenciadas_a_activos"),
		aplicabilidad.H("designado", "emisor_grande", "emision_de_ficha_superior_a_100m"),
		// Un proveedor de servicios de criptoactivos: otro papel, otras obligaciones.
		aplicabilidad.H("papel_mica", "casp", "proveedor_de_servicios_de_criptoactivos"),
	} {
		h.Procedencia = "papel declarado por el sujeto"
		m.Afirmar(h)
	}
	if _, err := m.Evaluar(); err != nil {
		t.Fatalf("evaluar: %v", err)
	}
	tieneDe := func(s string) map[string]bool {
		out := map[string]bool{}
		for _, o := range aplicablesA(t, m, s) {
			out[o] = true
		}
		return out
	}
	peq, gran, casp := tieneDe("emisor_pequeno"), tieneDe("emisor_grande"), tieneDe("casp")

	const trimestral = "mica.art22_1.comunicacion_trimestral_a_la_autoridad"
	const reserva = "mica.art30_1.actualizacion_de_la_informacion_publica_de_la_reserva"
	const conflictos = "mica.art72_4.revision_de_la_politica_de_conflictos_de_intereses"

	// DIRECCION 1: por encima del umbral, la comunicacion trimestral alcanza.
	if !gran[trimestral] {
		t.Error("emisor por encima de 100 M EUR: no se ha derivado la comunicacion trimestral " +
			"del art. 22.1")
	}
	// DIRECCION 2, la que muerde: por debajo del umbral, NO.
	if peq[trimestral] {
		t.Error("emisor por debajo de 100 M EUR: se ha derivado la comunicacion trimestral del " +
			"art. 22.1, y el apartado alcanza solo a las fichas «cuyo valor de emision sea " +
			"superior a 100 000 000 EUR». Una obligacion de mas frente a la autoridad no es " +
			"solo un coste: es una comunicacion que la entidad no debe hacer")
	}
	// Y el umbral NO se lleva por delante lo que si alcanza a los dos.
	for quien, s := range map[string]map[string]bool{"emisor_pequeno": peq, "emisor_grande": gran} {
		if !s[reserva] {
			t.Errorf("%s: el art. 30.1 alcanza a todo emisor de fichas referenciadas a activos, "+
				"sin umbral", quien)
		}
	}
	// EL PAPEL SEPARA: lo del proveedor de servicios no es del emisor y al reves.
	if casp[reserva] || casp[trimestral] {
		t.Error("proveedor de servicios de criptoactivos: se le han derivado obligaciones del " +
			"EMISOR de fichas referenciadas a activos, que es otro papel")
	}
	if peq[conflictos] || gran[conflictos] {
		t.Error("emisor de fichas: se le ha derivado el art. 72.4, que obliga al proveedor de " +
			"servicios de criptoactivos")
	}
	if !casp[conflictos] {
		t.Error("proveedor de servicios de criptoactivos: no se ha derivado el art. 72.4")
	}
	for quien, s := range map[string]map[string]bool{"emisor_pequeno": peq,
		"emisor_grande": gran, "casp": casp} {
		if len(s) == 0 {
			t.Fatalf("%s no ha derivado ni una obligacion: el motor esta vacio", quien)
		}
	}
}

// El registro del art. 27 de NIS2 NO alcanza a toda entidad esencial o
// importante, y el art. 23.4 SI.
//
// Es la misma trampa que el reglamento tecnico, esta vez DENTRO de un solo
// paquete y con dos obligaciones que viven en el mismo fichero: "si te aplica
// NIS2, te aplica todo lo de NIS2". El art. 23.4 (notificar un incidente
// significativo) alcanza a toda entidad esencial o importante; el art. 27.3
// (notificar cambios en la informacion de registro) dice "las entidades a que
// se refiere el apartado 1", y el art. 27.1 es una lista CERRADA de
// infraestructura digital: DNS, registros de nombres de dominio de primer
// nivel, entidades que prestan servicios de registro de nombres de dominio,
// nube, centros de datos, redes de distribucion de contenidos, servicios
// gestionados y de seguridad gestionados, mercados en linea, motores de
// busqueda en linea y plataformas de redes sociales.
//
// Por que importa mas que un coste de mas: el art. 27.3 es `notificatoria` y su
// entregable SALE de la organizacion. Escrito ancho, una electrica del anexo I
// presenta a su autoridad competente una notificacion de cambio de registro que
// nadie le pidio, y eso no se deshace.
func TestElRegistroDelArt27DeNis2NoAlcanzaATodaEntidadEsencial(t *testing.T) {
	m, _ := motorConElCorpus(t)
	for _, h := range []aplicabilidad.Hecho{
		// Un proveedor de servicios de centro de datos: esta en la lista del
		// art. 27.1 y ademas es entidad esencial o importante.
		aplicabilidad.H("papel_nis2_registro", "centro_de_datos", "entidad_del_art_27_1"),
		aplicabilidad.H("designado", "centro_de_datos", "entidad_esencial_o_importante"),
		// Una electrica: entidad esencial por el anexo I de la Directiva y
		// NINGUNO de los tipos del art. 27.1.
		aplicabilidad.H("designado", "electrica", "entidad_esencial_o_importante"),
	} {
		h.Procedencia = "papel declarado por el sujeto"
		m.Afirmar(h)
	}
	if _, err := m.Evaluar(); err != nil {
		t.Fatalf("evaluar: %v", err)
	}
	tieneDe := func(sujeto string) map[string]bool {
		out := map[string]bool{}
		for _, o := range aplicablesA(t, m, sujeto) {
			out[o] = true
		}
		return out
	}
	cpd, electrica := tieneDe("centro_de_datos"), tieneDe("electrica")

	const (
		registro  = "nis2.art27_3.notificacion_de_cambios_en_la_informacion_de_registro"
		incidente = "nis2.art23_4.notificacion_de_incidente_significativo"
	)

	// DIRECCION 1: al centro de datos le alcanzan las dos.
	if !cpd[registro] {
		t.Errorf("centro de datos: no se ha derivado %s, y el art. 27.1 nombra "+
			"expresamente a los proveedores de servicios de centro de datos", registro)
	}
	if !cpd[incidente] {
		t.Errorf("centro de datos: no se ha derivado %s, que alcanza a toda entidad "+
			"esencial o importante por el art. 23.1", incidente)
	}

	// DIRECCION 2, la que muerde: a la electrica le alcanza el art. 23.4 y NO el
	// art. 27.3, con el articulo de la exclusion. La primera comprobacion no es
	// decorativa: sin ella, la segunda se cumpliria sola el dia que el sujeto no
	// derive nada.
	if !electrica[incidente] {
		t.Errorf("electrica: no se ha derivado %s. Si esto falla, la comprobacion de abajo "+
			"se cumple sola porque el sujeto no tiene nada", incidente)
	}
	if electrica[registro] {
		t.Errorf("electrica: se ha derivado %s y NO aplica. El art. 27.3 de la Directiva "+
			"(UE) 2022/2555 dice «las entidades a que se refiere el apartado 1», y el art. "+
			"27.1 enumera once tipos de infraestructura digital entre los que no esta la "+
			"energia: una electrica es entidad esencial por el anexo I, que es otra cosa. "+
			"Esta obligacion es notificatoria y su entregable sale de la organizacion, asi "+
			"que de mas no cuesta horas: cuesta una actuacion indebida ante la autoridad "+
			"competente, y eso no se deshace", registro)
	}
}

// LOS DOS MARCOS ESPANOLES DE LA V1, con la direccion que importa: la segunda.
//
// La Ley Organica 3/2018 y la Ley 2/2023 alcanzan a poblaciones distintas y
// dentro de cada una hay relojes que solo tocan a algunos. Las dos formas de
// equivocarse cuestan, y no lo mismo:
//
//	de menos  al obligado le falta un plazo y se entera cuando ya paso;
//	de mas    a quien no lo debe le sale una fecha ante un supervisor. Cinco de
//	          los siete relojes de estos dos paquetes son notificatorios, asi
//	          que una fila de mas no cuesta horas: cuesta una actuacion indebida
//	          ante la Agencia, ante la Autoridad Independiente de Proteccion del
//	          Informante o ante el Ministerio Fiscal.
//
// El emparejamiento se hace por el ID de la obligacion, que es la identidad que
// el propio paquete declara y la misma con la que la regla la nombra: si un id
// cambia, el linter de aplicabilidad ya rechaza la regla que apunta a una
// obligacion que no existe, asi que este test no puede quedarse verde contra un
// nombre muerto.
func TestLosMarcosEspanolesRepartenSusRelojesPorSujeto(t *testing.T) {
	derivar := func(sujeto string, hechos ...aplicabilidad.Hecho) map[string]bool {
		m, _ := motorConElCorpus(t)
		for _, h := range hechos {
			h.Procedencia = "alcance declarado por el sujeto"
			m.Afirmar(h)
		}
		if _, err := m.Evaluar(); err != nil {
			t.Fatalf("evaluar: %v", err)
		}
		out := map[string]bool{}
		for _, o := range aplicablesA(t, m, sujeto) {
			out[o] = true
		}
		return out
	}

	// Una empresa de mas de cincuenta trabajadores con canal interno, camaras y
	// delegado de proteccion de datos designado.
	pyme := derivar("pyme",
		aplicabilidad.H("trata_datos_personales", "pyme"),
		aplicabilidad.H("designado", "pyme", "obligado_a_sistema_interno_de_informacion"),
		aplicabilidad.H("designado", "pyme", "delegado_de_proteccion_de_datos_designado"),
		aplicabilidad.H("designado", "pyme", "trata_imagenes_de_videovigilancia"),
	)
	for _, o := range []string{
		"lopdgdd.art22_3.supresion_de_las_imagenes",
		"lopdgdd.art22_3.puesta_a_disposicion_de_la_autoridad",
		"lopdgdd.art34_3.comunicacion_del_delegado_a_la_autoridad",
		"lopdgdd.art36_4.comunicacion_de_la_vulneracion_relevante",
		"lopdgdd.art37_1.comunicacion_de_la_decision_al_afectado",
		"lopdgdd.art37_2.respuesta_del_delegado_a_la_reclamacion_remitida",
		"ley2-2023.art7_2.reunion_presencial_con_el_informante",
		"ley2-2023.art8_3.notificacion_del_nombramiento_o_cese_del_responsable",
		"ley2-2023.art9_2_c.acuse_de_recibo_al_informante",
		"ley2-2023.art9_2_d.respuesta_a_las_actuaciones_de_investigacion",
		"ley2-2023.art9_2_j.remision_al_ministerio_fiscal",
		"ley2-2023.art26_2.conservacion_maxima_del_libro_registro",
		"ley2-2023.art32_4.supresion_de_la_comunicacion_sin_actuaciones",
	} {
		if !pyme[o] {
			t.Errorf("no se ha derivado %s, que si aplica a una entidad con canal interno, "+
				"camaras y delegado designado", o)
		}
	}
	for o, porQue := range map[string]string{
		"lopdgdd.art20_1_c.notificacion_de_la_inclusion_al_afectado": "el deber de notificar la " +
			"inclusion es de LA ENTIDAD QUE MANTENGA el sistema de informacion crediticia " +
			"(art. 20.1.c, parrafo segundo), y esta no lo mantiene",
		"lopdgdd.art65_4.respuesta_del_responsable_a_la_reclamacion_remitida": "el art. 65.4, " +
			"parrafo segundo, solo alcanza a quien NO ha designado delegado; con delegado, la " +
			"Agencia remite al delegado y el plazo aplicable es el mes del art. 37.2",
	} {
		if pyme[o] {
			t.Errorf("se ha derivado %s y NO aplica: %s", o, porQue)
		}
	}

	// Una entidad que mantiene un fichero comun de solvencia patrimonial: trata
	// datos personales, no tiene canal interno de informacion, no hace
	// videovigilancia y no ha designado delegado ni esta adherida a mecanismos
	// de resolucion extrajudicial de conflictos.
	buro := derivar("buro",
		aplicabilidad.H("trata_datos_personales", "buro"),
		aplicabilidad.H("designado", "buro", "mantiene_sistema_de_informacion_crediticia"),
	)
	for _, o := range []string{
		"lopdgdd.art20_1_c.notificacion_de_la_inclusion_al_afectado",
		"lopdgdd.art65_4.respuesta_del_responsable_a_la_reclamacion_remitida",
	} {
		if !buro[o] {
			t.Errorf("no se ha derivado %s, que si aplica a quien mantiene el sistema "+
				"crediticio y no tiene delegado", o)
		}
	}
	for o, porQue := range map[string]string{
		"lopdgdd.art22_3.supresion_de_las_imagenes": "el art. 22.1 y 22.3 alcanzan a quien " +
			"trata imagenes por camaras o videocamaras, y esta entidad no lo hace",
		"lopdgdd.art22_3.puesta_a_disposicion_de_la_autoridad": "mismo art. 22.3: sin " +
			"videovigilancia no hay grabacion que poner a disposicion de nadie",
		"lopdgdd.art34_3.comunicacion_del_delegado_a_la_autoridad": "el art. 34.3 manda " +
			"comunicar designaciones, nombramientos y ceses de delegado, y aqui no hay delegado",
		"lopdgdd.art36_4.comunicacion_de_la_vulneracion_relevante": "el deber de documentar y " +
			"comunicar la vulneracion relevante es DEL DELEGADO (art. 36.4), y sin delegado " +
			"designado no hay quien la aprecie en el sentido de ese apartado",
		"lopdgdd.art37_1.comunicacion_de_la_decision_al_afectado": "el art. 37.1 arranca con " +
			"«cuando el responsable o el encargado del tratamiento hubieran designado un " +
			"delegado de proteccion de datos»",
		"lopdgdd.art37_2.respuesta_del_delegado_a_la_reclamacion_remitida": "el art. 37.2 " +
			"remite la reclamacion AL DELEGADO, y sin delegado la via es la del art. 65.4",
		"ley2-2023.art8_3.notificacion_del_nombramiento_o_cese_del_responsable": "sin Sistema " +
			"interno de informacion no hay Responsable del Sistema que nombrar ni cesar " +
			"(Ley 2/2023, art. 8.1), y esta es notificatoria ante la Autoridad Independiente " +
			"de Proteccion del Informante",
		"ley2-2023.art9_2_j.remision_al_ministerio_fiscal": "la remision al Ministerio Fiscal " +
			"es contenido minimo del procedimiento de gestion del art. 9, que solo tiene quien " +
			"esta obligado a un Sistema interno de informacion",
		"ley2-2023.art26_2.conservacion_maxima_del_libro_registro": "el libro-registro del " +
			"art. 26.1 es de los sujetos obligados a disponer de canal interno de informaciones",
		"ley2-2023.art32_4.supresion_de_la_comunicacion_sin_actuaciones": "el art. 32 regula " +
			"el tratamiento de datos EN EL SISTEMA INTERNO DE INFORMACION, que esta entidad " +
			"no tiene",
	} {
		if buro[o] {
			t.Errorf("se ha derivado %s y NO aplica: %s", o, porQue)
		}
	}

	// El descargo que ninguno de los dos anteriores recorre: estar adherido a
	// un mecanismo de resolucion extrajudicial de conflictos SIN tener delegado
	// tambien saca del plazo del art. 65.4. Sin este caso, esa mitad de la
	// negacion no la atraviesa nadie y estaria en el corpus sin haberse
	// ejecutado nunca.
	mediado := derivar("mediado",
		aplicabilidad.H("trata_datos_personales", "mediado"),
		aplicabilidad.H("designado", "mediado", "adherido_a_resolucion_extrajudicial_de_conflictos"),
	)
	if mediado["lopdgdd.art65_4.respuesta_del_responsable_a_la_reclamacion_remitida"] {
		t.Error("se ha derivado el plazo del art. 65.4 a una entidad adherida a mecanismos de " +
			"resolucion extrajudicial de conflictos. El parrafo segundo del art. 65.4 exige " +
			"las DOS ausencias: ni delegado designado ni adhesion. Con adhesion, la Agencia " +
			"remite al organismo de resolucion extrajudicial a los efectos del art. 38.2")
	}
}
