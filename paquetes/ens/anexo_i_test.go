// Package ens no tiene codigo de produccion: un paquete de corpus es un fichero
// de datos y nada mas. Este fichero de test existe por una sola razon, y esta
// aqui y no en la raiz porque es del paquete, no del motor.
//
// QUE PRUEBA. Que la cadena del anexo I del RD 311/2022 funciona ENTERA desde
// las preguntas que el paquete declara: el sujeto responde preguntas de alcance,
// de las respuestas salen hechos, el agregado toma el maximo sobre cada
// informacion y cada servicio, de ahi sale la categoria del sistema, y de la
// categoria cuelgan las obligaciones. Antes de esto la regla del anexo I estaba
// declarada y era correcta, pero los hechos que consume (maneja, nivel_dimension)
// no los recogia ninguna pregunta: solo se podian escribir a mano en un test. La
// categoria se DECLARABA y no se calculaba.
//
// LO QUE ESTE FICHERO NO ES. La conversion "respuesta -> hecho" que hace
// respuestasAHechos NO existe todavia en el producto: no hay superficie ni
// adaptador que la haga. La convencion que implementa (el predicado se llama
// como el atributo, el primer argumento es la instancia de entidad y el segundo
// el valor) es la que ya usaban las reglas de este paquete para el atributo
// ambito y para los booleanos. Aqui se escribe una vez, se ejecuta contra el
// paquete publicado y se deja a la vista, para que quien construya el alcance
// implante ESA y no otra. Mientras tanto, la cadena es cierta en los datos y le
// falta el ultimo tramo en el producto: consta en docs/pendientes.md.
package ens

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"dutiq/nucleo/aplicabilidad"
	"dutiq/nucleo/corpus"
)

// urnENS identifica el paquete dentro del corpus cargado. Este fichero SI puede
// nombrar la norma: TestNingunaNormaCableada solo vigila los _test.go de nucleo/
// y adaptadores/, porque aqui es donde el codigo se encuentra con paquetes/.
const urnENS = "urn:es:rd:2022:311"

// reglaDelAgregado es la regla insignia del paquete: la del anexo I.
const reglaDelAgregado = "nivel_maximo_de_las_dimensiones"

// dimensiones son las cinco del art. 40.2, transcrito en este mismo paquete:
// "perjuicio para la disponibilidad, autenticidad, integridad, confidencialidad
// o trazabilidad".
var dimensiones = []string{"disponibilidad", "autenticidad", "integridad", "confidencialidad", "trazabilidad"}

// entidadesDelAnexoI son los dos tipos de sujeto sobre los que el anexo I valora
// las dimensiones, con el atributo que los ata a su sistema.
var entidadesDelAnexoI = map[string]string{
	"informacion": "manejada_por_el_sistema",
	"servicio":    "prestado_por_el_sistema",
}

// paqueteENS carga el corpus publicado y devuelve este paquete. Carga por el
// camino de verdad (corpus.Cargar, con su linter y sus dorados) y no leyendo el
// JSON a mano: si el paquete no pasa el linter, este fichero tampoco pasa.
func paqueteENS(t *testing.T) *corpus.Paquete {
	t.Helper()
	ps, err := corpus.Cargar(filepath.Join(".."))
	if err != nil {
		t.Fatalf("el corpus publicado no carga: %v", err)
	}
	for _, p := range ps {
		if p.URN == urnENS {
			return p
		}
	}
	t.Fatalf("%s no esta en el corpus cargado", urnENS)
	return nil
}

// motorConENS devuelve un motor con SOLO las reglas de este paquete. Que el
// corpus entero encaje junto es lo que comprueba aplicabilidad_corpus_test.go de
// la raiz; aqui se aisla el paquete para que un fallo senale a quien lo tiene.
func motorConENS(t *testing.T, p *corpus.Paquete) *aplicabilidad.Motor {
	t.Helper()
	prog, errs := p.Programa()
	if len(errs) != 0 {
		t.Fatalf("%s no produce un programa valido: %v", p.URN, errs)
	}
	m := aplicabilidad.NuevoMotor()
	if err := m.Cargar(prog); err != nil {
		t.Fatalf("el motor rechaza el programa de %s: %v", p.URN, err)
	}
	return m
}

// Respuesta es lo que contesta el sujeto obligado a una pregunta de alcance,
// sobre una instancia concreta de entidad: este sistema, esta informacion, este
// servicio.
type Respuesta struct {
	Pregunta  string // id de una Pregunta declarada por el paquete
	Instancia string // a que sistema, informacion o servicio se refiere
	Valor     string
}

// respuestasAHechos convierte respuestas en hechos usando SOLO lo que el paquete
// declara: la pregunta dice a que entidad y a que atributo apunta, y el atributo
// dice de que tipo es y que valores admite. Ningun nombre de predicado esta
// escrito aqui.
//
// La guarda del enumerado no es decorativa. Un valor fuera de los declarados
// pasa el linter (que no ve respuestas) y revienta MAS TARDE, dentro del
// agregado, con "valor fuera de la escala": el fallo aparece al evaluar, en casa
// del cliente, y no al responder. Se corta aqui, que es donde entra.
func respuestasAHechos(t *testing.T, p *corpus.Paquete, rs []Respuesta) []aplicabilidad.Hecho {
	t.Helper()
	preguntas := map[string]corpus.Pregunta{}
	for _, q := range p.Preguntas {
		preguntas[q.ID] = q
	}
	atributos := map[string]corpus.Atributo{}
	for _, e := range p.Entidades {
		for _, a := range e.Atributos {
			atributos[e.Nombre+"."+a.Nombre] = a
		}
	}
	var out []aplicabilidad.Hecho
	for _, r := range rs {
		q, ok := preguntas[r.Pregunta]
		if !ok {
			t.Fatalf("el paquete %s no declara la pregunta %s, asi que esa respuesta no "+
				"se puede recoger por la interfaz", p.URN, r.Pregunta)
		}
		a, ok := atributos[q.Entidad+"."+q.Atributo]
		if !ok {
			t.Fatalf("la pregunta %s apunta a %s.%s, que el paquete no declara",
				q.ID, q.Entidad, q.Atributo)
		}
		if a.Tipo == corpus.Enumerado && !admitido(a.Valores, r.Valor) {
			t.Fatalf("la respuesta %q a %s no es uno de los valores que el paquete declara "+
				"para %s.%s (%v). Un valor inventado no lo caza el linter y hace fallar al "+
				"motor mas tarde, dentro del agregado",
				r.Valor, q.ID, q.Entidad, q.Atributo, a.Valores)
		}
		h := aplicabilidad.H(a.Nombre, r.Instancia, r.Valor)
		if a.Tipo == corpus.Booleano {
			if r.Valor != "si" {
				continue // un booleano en no no afirma nada
			}
			h = aplicabilidad.H(a.Nombre, r.Instancia)
		}
		h.Procedencia = "respuesta a " + q.ID
		out = append(out, h)
	}
	return out
}

func admitido(valores []string, v string) bool {
	for _, x := range valores {
		if x == v {
			return true
		}
	}
	return false
}

// derivar responde las preguntas, evalua y devuelve lo derivado para un sujeto.
func derivar(t *testing.T, rs []Respuesta) (*aplicabilidad.Motor, *corpus.Paquete) {
	t.Helper()
	p := paqueteENS(t)
	m := motorConENS(t, p)
	hs := respuestasAHechos(t, p, rs)
	if len(hs) != len(rs) {
		t.Fatalf("se respondieron %d preguntas y salieron %d hechos", len(rs), len(hs))
	}
	for _, h := range hs {
		m.Afirmar(h)
	}
	if _, err := m.Evaluar(); err != nil {
		t.Fatalf("evaluar: %v", err)
	}
	return m, p
}

func categoriaDe(t *testing.T, m *aplicabilidad.Motor, sistema string) []string {
	t.Helper()
	var out []string
	for _, h := range m.Consultar(aplicabilidad.A("categoria",
		aplicabilidad.C(sistema), aplicabilidad.V("C"))) {
		out = append(out, h.Args[1])
	}
	sort.Strings(out)
	return out
}

func aplicables(m *aplicabilidad.Motor, sistema string) map[string]bool {
	out := map[string]bool{}
	for _, h := range m.Consultar(aplicabilidad.A("aplica",
		aplicabilidad.V("O"), aplicabilidad.C(sistema))) {
		out[h.Args[0]] = true
	}
	return out
}

// alcanceDe monta las respuestas de un sistema con una informacion y un
// servicio, cada uno con el nivel que se le pase por dimension. Solo se
// responden las dimensiones con nivel: el anexo I no adscribe nivel alguno a la
// dimension que no se ve afectada, y eso NO es lo mismo que BAJO.
func alcanceDe(sistema, info, servicio string, nInfo, nServicio map[string]string) []Respuesta {
	rs := []Respuesta{
		{Pregunta: "ens.q.ambito", Instancia: sistema, Valor: "sector_publico"},
	}
	if info != "" {
		rs = append(rs, Respuesta{Pregunta: "ens.q.informacion.sistema", Instancia: info, Valor: sistema})
		for _, d := range dimensiones {
			if n, ok := nInfo[d]; ok {
				rs = append(rs, Respuesta{Pregunta: "ens.q.informacion." + d, Instancia: info, Valor: n})
			}
		}
	}
	if servicio != "" {
		rs = append(rs, Respuesta{Pregunta: "ens.q.servicio.sistema", Instancia: servicio, Valor: sistema})
		for _, d := range dimensiones {
			if n, ok := nServicio[d]; ok {
				rs = append(rs, Respuesta{Pregunta: "ens.q.servicio." + d, Instancia: servicio, Valor: n})
			}
		}
	}
	return rs
}

// ---------------------------------------------------------------------------
// 1. El modelo: las cinco dimensiones sobre informacion y sobre servicio, y una
//    pregunta que recoge cada dato. Lo segundo es lo que faltaba: sin pregunta,
//    el dato no entra por ninguna parte y la regla vive solo en un test.
// ---------------------------------------------------------------------------

func TestElAnexoITieneEntidadesInformacionYServicioConSusCincoDimensiones(t *testing.T) {
	p := paqueteENS(t)
	porNombre := map[string]corpus.TipoEntidad{}
	for _, e := range p.Entidades {
		porNombre[e.Nombre] = e
	}
	for entidad, relacion := range entidadesDelAnexoI {
		e, ok := porNombre[entidad]
		if !ok {
			t.Fatalf("el paquete no declara la entidad %q. Sin ella la categoria del anexo I "+
				"solo se puede declarar a mano, no calcular", entidad)
		}
		at := map[string]corpus.Atributo{}
		for _, a := range e.Atributos {
			at[a.Nombre] = a
		}
		if _, ok := at[relacion]; !ok {
			t.Errorf("la entidad %s no dice a que sistema pertenece (atributo %s), asi que "+
				"su nivel no puede subir la categoria de ningun sistema", entidad, relacion)
		}
		for _, d := range dimensiones {
			a, ok := at["nivel_"+d]
			if !ok {
				t.Errorf("la entidad %s no declara la dimension %s del art. 40.2", entidad, d)
				continue
			}
			if a.Tipo != corpus.Enumerado {
				t.Errorf("%s.nivel_%s tendria que ser enumerado y es %s", entidad, d, a.Tipo)
			}
			if a.Cita == "" {
				t.Errorf("%s.nivel_%s sin cita", entidad, d)
			}
			if a.Obligado {
				t.Errorf("%s.nivel_%s esta marcado como obligado, y el anexo I permite que una "+
					"dimension no se vea afectada y no se adscriba a nivel alguno", entidad, d)
			}
		}
	}
}

// El corazon del P1: cada atributo de informacion y de servicio lo recoge una
// pregunta del paquete. Un atributo sin pregunta es un hecho que solo se puede
// escribir a mano.
func TestCadaDatoDelAnexoILoRecogeUnaPreguntaDelPaquete(t *testing.T) {
	p := paqueteENS(t)
	recogido := map[string]string{}
	for _, q := range p.Preguntas {
		recogido[q.Entidad+"."+q.Atributo] = q.ID
	}
	for _, e := range p.Entidades {
		if _, ok := entidadesDelAnexoI[e.Nombre]; !ok {
			continue
		}
		for _, a := range e.Atributos {
			clave := e.Nombre + "." + a.Nombre
			if recogido[clave] == "" {
				t.Errorf("ninguna pregunta del paquete recoge %s. La regla del anexo I consume "+
					"ese dato, asi que sin pregunta la categoria se declara y no se calcula", clave)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// 2. Contra el atacante: los valores que se ofrecen tienen que ser EXACTAMENTE
//    la escala del agregado. Si divergen, el linter da verde y el motor falla
//    al evaluar, en casa del cliente.
// ---------------------------------------------------------------------------

func TestLosNivelesQueSeOfrecenSonLaEscalaDelAgregadoDelAnexoI(t *testing.T) {
	p := paqueteENS(t)
	var escala []string
	for _, r := range p.Aplicabilidad.Reglas {
		if r.ID == reglaDelAgregado {
			if r.Escala == nil {
				t.Fatalf("la regla %s agrega por maximo y no declara escala", reglaDelAgregado)
			}
			escala = r.Escala.Orden
		}
	}
	if len(escala) == 0 {
		t.Fatalf("el paquete ya no declara la regla %s, que es la del anexo I", reglaDelAgregado)
	}
	for _, e := range p.Entidades {
		if _, ok := entidadesDelAnexoI[e.Nombre]; !ok {
			continue
		}
		for _, a := range e.Atributos {
			if !strings.HasPrefix(a.Nombre, "nivel_") {
				continue
			}
			if strings.Join(a.Valores, ",") != strings.Join(escala, ",") {
				t.Errorf("%s.%s ofrece %v y el agregado del anexo I ordena %v. Cualquier valor "+
					"de mas pasa el linter y hace fallar al motor al evaluar, no al responder",
					e.Nombre, a.Nombre, a.Valores, escala)
			}
		}
	}
}

// La consecuencia de lo anterior, demostrada: si un nivel fuera de la escala
// llegara al motor, la evaluacion falla. Por eso la guarda de respuestasAHechos
// tiene que estar y por eso el test de arriba importa.
func TestUnNivelFueraDeLaEscalaHaceFallarLaEvaluacion(t *testing.T) {
	p := paqueteENS(t)
	m := motorConENS(t, p)
	m.Afirmar(aplicabilidad.H("manejada_por_el_sistema", "padron", "sede"))
	m.Afirmar(aplicabilidad.H("nivel_confidencialidad", "padron", "CRITICO"))
	if _, err := m.Evaluar(); err == nil {
		t.Fatal("un nivel que no esta en la escala del anexo I tiene que hacer fallar la " +
			"evaluacion, y paso sin decir nada")
	}
}

// Y la otra mitad: una respuesta con ese mismo valor no llega nunca al motor,
// porque el atributo declara sus valores. Se ejecuta contra un t aparte para
// poder comprobar que aborta.
func TestUnaRespuestaConNivelInventadoNoLlegaAlMotor(t *testing.T) {
	p := paqueteENS(t)
	var a corpus.Atributo
	for _, e := range p.Entidades {
		if e.Nombre != "informacion" {
			continue
		}
		for _, x := range e.Atributos {
			if x.Nombre == "nivel_confidencialidad" {
				a = x
			}
		}
	}
	if a.Nombre == "" {
		t.Fatal("informacion.nivel_confidencialidad no existe")
	}
	if admitido(a.Valores, "CRITICO") {
		t.Fatal("CRITICO no es un nivel del anexo I y el paquete lo admite")
	}
	for _, v := range []string{"BAJO", "MEDIO", "ALTO"} {
		if !admitido(a.Valores, v) {
			t.Errorf("%s es un nivel del anexo I y el paquete no lo admite", v)
		}
	}
}

// ---------------------------------------------------------------------------
// 3. La cadena entera, desde las preguntas: respuestas, agregado por maximo,
//    categoria derivada y obligacion aplicable. Sin ningun hecho escrito a mano.
// ---------------------------------------------------------------------------

func TestLaCategoriaSeCalculaDesdeLasRespuestasALasPreguntas(t *testing.T) {
	// El padron pide MEDIO en confidencialidad; la cita previa pide ALTO en
	// disponibilidad. El maximo del anexo I es ALTO, o sea categoria ALTA, y no
	// se ha respondido a ens.q.categoria en ningun momento.
	rs := alcanceDe("sede", "padron", "cita-previa",
		map[string]string{"confidencialidad": "MEDIO", "integridad": "BAJO", "disponibilidad": "BAJO"},
		map[string]string{"disponibilidad": "ALTO"})
	for _, r := range rs {
		if r.Pregunta == "ens.q.categoria" {
			t.Fatal("este test no puede responder a la pregunta de la categoria: lo que prueba " +
				"es que la categoria sale sin ella")
		}
	}
	m, _ := derivar(t, rs)

	cat := categoriaDe(t, m, "sede")
	if len(cat) != 1 || cat[0] != "ALTA" {
		t.Fatalf("el maximo de MEDIO, BAJO, BAJO y ALTO es ALTO, o sea categoria ALTA "+
			"(RD 311/2022 anexo I, apartado 2), y salio %v", cat)
	}

	// Y sabe decir por que, que es la pregunta que separa esto de una lista.
	h := m.Consultar(aplicabilidad.A("categoria", aplicabilidad.C("sede"), aplicabilidad.C("ALTA")))
	if len(h) != 1 || !strings.Contains(m.Explicar(h[0]), "categoria_alta") {
		t.Errorf("la categoria derivada tiene que decir que regla la derivo, y dijo %q",
			m.Explicar(h[0]))
	}

	tiene := aplicables(m, "sede")
	for _, o := range []string{
		"ens.art31.auditoria_ordinaria",
		"ens.its_conformidad.certificacion_media_alta",
		"ens.its_conformidad.publicidad_certificacion_media_alta",
		"ens.art40.2.determinacion_de_la_categoria",
	} {
		if !tiene[o] {
			t.Errorf("con categoria ALTA tiene que aplicar %s y no se derivo", o)
		}
	}
	if tiene["ens.its_conformidad.autoevaluacion_basica"] {
		t.Error("la autoevaluacion es solo de categoria BASICA (ITS de Conformidad, III.2)")
	}
}

// Control negativo: cambiar UNA respuesta cambia la categoria y con ella las
// obligaciones. Sin esto, un motor que derivara siempre ALTA pasaria el test de
// arriba sin despeinarse.
func TestCambiarUnaRespuestaCambiaLaCategoriaYLasObligaciones(t *testing.T) {
	casos := []struct {
		nombre      string
		info, servi map[string]string
		categoria   string
		aplica      string
		noAplica    string
	}{
		{"todo BAJO, categoria BASICA",
			map[string]string{"confidencialidad": "BAJO", "integridad": "BAJO"},
			map[string]string{"disponibilidad": "BAJO"},
			"BASICA", "ens.its_conformidad.autoevaluacion_basica", "ens.art31.auditoria_ordinaria"},
		{"una informacion en MEDIO, categoria MEDIA",
			map[string]string{"confidencialidad": "MEDIO", "integridad": "BAJO"},
			map[string]string{"disponibilidad": "BAJO"},
			"MEDIA", "ens.art31.auditoria_ordinaria", "ens.its_conformidad.autoevaluacion_basica"},
		{"solo el servicio en ALTO, categoria ALTA",
			map[string]string{"confidencialidad": "BAJO", "integridad": "BAJO"},
			map[string]string{"disponibilidad": "ALTO"},
			"ALTA", "ens.its_conformidad.certificacion_media_alta", "ens.its_conformidad.autoevaluacion_basica"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			m, _ := derivar(t, alcanceDe("sede", "padron", "cita-previa", c.info, c.servi))
			cat := categoriaDe(t, m, "sede")
			if len(cat) != 1 || cat[0] != c.categoria {
				t.Fatalf("se esperaba %s y salio %v", c.categoria, cat)
			}
			tiene := aplicables(m, "sede")
			if !tiene[c.aplica] {
				t.Errorf("con categoria %s tiene que aplicar %s", c.categoria, c.aplica)
			}
			if tiene[c.noAplica] {
				t.Errorf("con categoria %s NO puede aplicar %s: es una obligacion de mas que "+
					"el cliente paga sin deberla", c.categoria, c.noAplica)
			}
		})
	}
}

// Contra el atacante: el nivel de la informacion de OTRO sistema no puede subir
// la categoria de este. Es el fallo clasico del agregado mal agrupado, y en este
// dominio significa meter a un ayuntamiento entero en regimen de certificacion.
func TestElNivelDeOtroSistemaNoSubeLaCategoriaDeEste(t *testing.T) {
	rs := append(
		alcanceDe("sede", "padron", "", map[string]string{"confidencialidad": "BAJO"}, nil),
		alcanceDe("intervencion", "expedientes-sancionadores", "",
			map[string]string{"confidencialidad": "ALTO"}, nil)...)
	m, _ := derivar(t, rs)

	if cat := categoriaDe(t, m, "sede"); len(cat) != 1 || cat[0] != "BASICA" {
		t.Fatalf("la sede solo maneja una informacion en BAJO, o sea categoria BASICA, y salio %v. "+
			"Si sale ALTA, el agregado esta mezclando los dos sistemas", cat)
	}
	if cat := categoriaDe(t, m, "intervencion"); len(cat) != 1 || cat[0] != "ALTA" {
		t.Fatalf("intervencion maneja una informacion en ALTO, o sea categoria ALTA, y salio %v", cat)
	}
	if aplicables(m, "sede")["ens.art31.auditoria_ordinaria"] {
		t.Error("la sede es BASICA y la auditoria del art. 31.1 no le alcanza")
	}
	if !aplicables(m, "intervencion")["ens.art31.auditoria_ordinaria"] {
		t.Error("intervencion es ALTA y la auditoria del art. 31.1 si le alcanza")
	}
}

// Una dimension sin responder no vale BAJO: no se adscribe a nivel alguno. Si
// se tratara como BAJO, un sistema con una sola dimension valorada en ALTO
// seguiria saliendo ALTA (el maximo no cambia), asi que lo que se comprueba es
// lo contrario: que la ausencia no derive ningun hecho de nivel.
func TestUnaDimensionSinResponderNoAfirmaNingunNivel(t *testing.T) {
	m, p := derivar(t, alcanceDe("sede", "padron", "",
		map[string]string{"confidencialidad": "MEDIO"}, nil))
	niveles := m.Consultar(aplicabilidad.A(local(t, p, "nivel_requerido"),
		aplicabilidad.C("padron"), aplicabilidad.V("N")))
	if len(niveles) != 1 {
		t.Fatalf("se respondio una sola dimension y hay %d hechos de nivel: %v", len(niveles), niveles)
	}
	if niveles[0].Args[1] != "MEDIO" {
		t.Fatalf("el unico nivel tenia que ser MEDIO y es %v", niveles[0])
	}
	for _, d := range dimensiones {
		n := m.Consultar(aplicabilidad.A("nivel_"+d, aplicabilidad.C("padron"), aplicabilidad.V("N")))
		if d == "confidencialidad" {
			if len(n) != 1 {
				t.Errorf("la confidencialidad se respondio y hay %d hechos", len(n))
			}
			continue
		}
		if len(n) != 0 {
			t.Errorf("la dimension %s no se respondio y aun asi hay hecho de nivel: %v", d, n)
		}
	}
	if cat := categoriaDe(t, m, "sede"); len(cat) != 1 || cat[0] != "MEDIA" {
		t.Fatalf("con una sola dimension en MEDIO la categoria es MEDIA y salio %v", cat)
	}
}

// ---------------------------------------------------------------------------
// 4. El espacio de nombres. main lo implanto mientras esto se construia y
//    cambia la regla del juego: un paquete que DEFINE un predicado se queda con
//    el, y los hechos que el sujeto afirma con ese nombre dejan de alimentar sus
//    reglas. Las dos puertas de aqui abajo son las que impiden que el modelo del
//    anexo I vuelva a romperlo.
// ---------------------------------------------------------------------------

// local devuelve como se llama un predicado propio del paquete una vez aislado.
func local(t *testing.T, p *corpus.Paquete, pred string) string {
	t.Helper()
	prog, errs := p.Programa()
	if len(errs) != 0 {
		t.Fatalf("%s no produce un programa valido: %v", p.URN, errs)
	}
	return prog.Local(pred)
}

// hechosDelSujeto es el vocabulario de ENTRADA: lo que el sujeto afirma al
// describir su alcance. El paquete lo CONSUME y no puede definirlo.
var hechosDelSujeto = []string{
	"maneja", "nivel_dimension", "ambito", "trata_datos_personales",
	"servicios_externalizados", "usa_servicios_en_la_nube",
	"nivel_disponibilidad", "nivel_autenticidad", "nivel_integridad",
	"nivel_confidencialidad", "nivel_trazabilidad",
	"manejada_por_el_sistema", "prestado_por_el_sistema",
}

func TestElPaqueteNoDefineNingunHechoQueAporteElSujeto(t *testing.T) {
	p := paqueteENS(t)
	prog, errs := p.Programa()
	if len(errs) != 0 {
		t.Fatalf("%s no produce un programa valido: %v", p.URN, errs)
	}
	prohibido := map[string]bool{}
	for _, h := range hechosDelSujeto {
		prohibido[h] = true
	}
	for _, r := range prog.Reglas {
		if prohibido[r.Cabeza.Pred] {
			t.Errorf("la regla %s pone cabeza sobre %s, que es un hecho del sujeto. Al "+
				"definirlo el paquete se queda con el predicado, se le prefija el urn, y los "+
				"hechos que el sujeto afirma con ese nombre dejan de alimentar ninguna regla. "+
				"El arreglo es derivar un predicado PROPIO a partir del hecho, como hacen "+
				"alcance_del_sistema y nivel_requerido", r.ID, r.Cabeza.Pred)
		}
	}
}

// Un atributo que el paquete PIDE al sujeto y ademas DEFINE con una regla tiene
// que estar exportado, o el hecho respondido no alimenta nada. Es el caso de
// categoria, que se pregunta y se deriva a la vez.
func TestTodoAtributoQueElPaqueteTambienDefineEstaExportado(t *testing.T) {
	p := paqueteENS(t)
	prog, errs := p.Programa()
	if len(errs) != 0 {
		t.Fatalf("%s no produce un programa valido: %v", p.URN, errs)
	}
	exporta := map[string]bool{}
	for _, e := range prog.Exporta {
		exporta[e] = true
	}
	definidos := map[string]string{}
	for _, r := range prog.Reglas {
		definidos[r.Cabeza.Pred] = r.ID
	}
	for _, e := range p.Entidades {
		for _, a := range e.Atributos {
			regla, definido := definidos[a.Nombre]
			if !definido || exporta[a.Nombre] {
				continue
			}
			t.Errorf("%s.%s se le pregunta al sujeto y ademas lo define la regla %s sin "+
				"exportarlo. El predicado queda aislado con el urn del paquete, asi que la "+
				"respuesta del sujeto no alimenta ninguna regla y nadie se entera",
				e.Nombre, a.Nombre, regla)
		}
	}
}

// Y la compatibilidad hacia atras, que es lo que aplicabilidad_corpus_test.go de
// la raiz da por supuesto: quien ya cargaba el alcance afirmando maneja y
// nivel_dimension en crudo sigue derivando la misma categoria. Si esto se rompe,
// se rompe un expediente ya emitido.
func TestElAlcanceAfirmadoEnCrudoSigueDerivandoLaCategoria(t *testing.T) {
	p := paqueteENS(t)
	m := motorConENS(t, p)
	for _, h := range []aplicabilidad.Hecho{
		aplicabilidad.H("ambito", "sede", "sector_publico"),
		aplicabilidad.H("maneja", "sede", "padron"),
		aplicabilidad.H("nivel_dimension", "padron", "confidencialidad", "MEDIO"),
		aplicabilidad.H("nivel_dimension", "padron", "integridad", "BAJO"),
	} {
		m.Afirmar(h)
	}
	if _, err := m.Evaluar(); err != nil {
		t.Fatalf("evaluar: %v", err)
	}
	if cat := categoriaDe(t, m, "sede"); len(cat) != 1 || cat[0] != "MEDIA" {
		t.Fatalf("el maximo de MEDIO y BAJO es MEDIO, o sea categoria MEDIA, y salio %v", cat)
	}
	if !aplicables(m, "sede")["ens.art31.auditoria_ordinaria"] {
		t.Error("de la categoria MEDIA cuelga la auditoria bienal del art. 31.1")
	}
}
