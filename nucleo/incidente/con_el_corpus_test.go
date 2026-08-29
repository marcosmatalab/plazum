package incidente_test

import (
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/incidente"
	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// LA PRUEBA DE QUE LA JUNTA CIERRA, contra el corpus entero.
//
// La medicion de nucleo/corpus/por_objeto_test.go dejo escrito lo que pasaba
// antes de que este paquete existiera: dos incidentes bajo el nombre de hecho
// que escribe el paquete daban UN vencimiento, y el que desaparecia vencia
// nueve dias antes. Aqui se comprueba el mismo escenario al derecho.
//
// LOS RELOJES SE ELIGEN POR PROPIEDAD, NUNCA POR NOMBRE, y por dos motivos que
// apuntan al mismo sitio. El invariante 2 prohibe nombrar una norma en el
// codigo, y lo hizo cumplir: la primera version de este fichero nombraba tres
// obligaciones y TestNingunaNormaCableada la puso roja. Y ademas una lista de
// ids escrita al lado del test solo prueba lo que su autor penso, mientras que
// la propiedad se aplica sola al reloj que alguien escriba manana.
//
// Lo que se afirma, entonces, no es "estas tres obligaciones funcionan": es que
// TODO plazo del corpus que arranca de un hecho se comporta bien cuando lo
// mueve un incidente.

// reloj es un plazo del corpus que arranca de un hecho, con sus hitos ya
// normalizados: un paquete puede declarar un hito suelto o una lista, y aqui
// las dos formas se miran igual.
type reloj struct {
	o          corpus.Obligacion
	disparador string
	hitos      []corpus.HitoSpec
}

func (r reloj) clases() []string {
	var out []string
	vistas := map[string]bool{}
	for _, h := range r.hitos {
		if h.Clase != "" && !vistas[h.Clase] {
			vistas[h.Clase] = true
			out = append(out, h.Clase)
		}
	}
	return out
}

func (r reloj) hitosDeClase(clase string) []corpus.HitoSpec {
	var out []corpus.HitoSpec
	for _, h := range r.hitos {
		if h.Clase == clase {
			out = append(out, h)
		}
	}
	return out
}

// conPlazoPropio dice si un hito puede dar fecha con SOLO el disparador. Los
// que no: los que la norma deja sin plazo ("inmediata", "sin dilacion
// indebida"), los que cuelgan de la remision de otro y los que traen un tope
// que corre de un segundo hecho. Los tres son formas legitimas del corpus y
// ninguno es un fallo; lo que seria un fallo es que un test los diera por
// determinados y luego relajara la afirmacion hasta que dejara de decir nada.
func conPlazoPropio(h corpus.HitoSpec) bool {
	return h.DesdeHito == "" && h.Tope == nil && h.Limite != "" && h.Limite != "indeterminado"
}

func (r reloj) encadenados() []corpus.HitoSpec {
	var out []corpus.HitoSpec
	for _, h := range r.hitos {
		if h.DesdeHito != "" {
			out = append(out, h)
		}
	}
	return out
}

// relojesDeIncidente devuelve todos los plazos del corpus que arrancan de un
// hecho. Cargar recorre los paquetes en orden, asi que un fallo sale siempre
// en el mismo sitio.
func relojesDeIncidente(t *testing.T) []reloj {
	t.Helper()
	paquetes, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatalf("cargando el corpus: %v", err)
	}
	var out []reloj
	for _, p := range paquetes {
		for _, o := range p.Obligaciones {
			tm := o.Temporalidad
			if tm == nil || tm.Primitiva != "plazo" || tm.Disparador["hecho"] == "" {
				continue
			}
			hs := tm.Hitos
			if len(hs) == 0 {
				id := tm.Hito
				if id == "" {
					id = "limite"
				}
				hs = []corpus.HitoSpec{{ID: id, Limite: tm.Limite}}
			}
			out = append(out, reloj{o: o, disparador: tm.Disparador["hecho"], hitos: hs})
		}
	}
	if len(out) == 0 {
		t.Fatal("ningun plazo con disparador en el corpus: el test no ha comprobado nada")
	}
	return out
}

func instanteC(t *testing.T, s string) time.Time {
	t.Helper()
	x, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("instante %q: %v", s, err)
	}
	return x
}

// mezclar une los hechos del INCIDENTE con los que no son suyos.
//
// Los dos juegos existen y son de duenos distintos, y conviene verlo escrito.
// Una clase puede ser del incidente (que se clasificara como grave) o de la
// ENTIDAD (que sea esencial o importante), y la de la entidad no cambia porque
// haya un incidente: viene del alcance. La junta de D-18 esta aqui, y el
// mecanismo es el mismo para las dos: un nombre que pone el paquete y un
// instante que pone quien tiene el dato.
func mezclar(a, b ventana.Hechos) ventana.Hechos {
	h := ventana.Hechos{}
	for k, v := range a {
		h[k] = v
	}
	for k, v := range b {
		h[k] = v
	}
	return h
}

func determinados(vs []ventana.Vencimiento) map[string]time.Time {
	out := map[string]time.Time{}
	for _, v := range vs {
		if v.Estado == ventana.Determinado {
			out[v.Hito] = v.Vence
		}
	}
	return out
}

// plazosDe calcula los vencimientos de un reloj movido por un incidente, con
// los hechos que no son del incidente por separado.
func plazosDe(t *testing.T, r reloj, i *incidente.Incidente, ajenos ventana.Hechos) []ventana.Vencimiento {
	t.Helper()
	h, err := i.Hechos(r.disparador)
	if err != nil {
		t.Fatalf("%s: %v", r.o.ID, err)
	}
	vs, err := corpus.VencimientosDe(r.o, mezclar(ajenos, h), time.Time{})
	if err != nil {
		t.Fatalf("%s: %v", r.o.ID, err)
	}
	return vs
}

// ---------------------------------------------------------------------------
// Lo que no existia hace una hora: dos incidentes, dos plazos
// ---------------------------------------------------------------------------

// El escenario medido, al derecho. Dos incidentes que OCURRIERON el mismo dia y
// se SUPIERON con nueve dias de diferencia tienen dos juegos de plazos, y son
// distintos: la constancia es lo que arranca el reloj, no la ocurrencia.
func TestDosIncidentesDanDosPlazosYNoSePisan(t *testing.T) {
	const ocurrio = "2026-03-01T04:00:00Z"
	primero, segundo := "2026-03-02T09:00:00Z", "2026-03-11T17:00:00Z"

	comprobados := 0
	var sinFecha []string
	for _, r := range relojesDeIncidente(t) {
		// UNA sola clase por pasada: dos clasificaciones distintas en el mismo
		// instante son un empate, y un empate no da fecha a proposito.
		ajenos := ventana.Hechos{}
		if cs := r.clases(); len(cs) > 0 {
			ajenos[cs[0]] = instanteC(t, "2020-01-01T00:00:00Z")
		}

		fechas := make([]map[string]time.Time, 0, 2)
		for n, cuando := range []string{primero, segundo} {
			i, err := incidente.Abrir("incidente/2026-01"+string(rune('4'+n)),
				instanteC(t, ocurrio), instanteC(t, cuando), "soc")
			if err != nil {
				t.Fatal(err)
			}
			fechas = append(fechas, determinados(plazosDe(t, r, i, ajenos)))
		}
		// UN RELOJ PUEDE NO DAR NINGUNA FECHA Y ESTAR BIEN: si todos sus hitos
		// cuelgan de otro, traen tope o los deja la norma sin plazo. No se calla
		// (eso seria el descarte silencioso otra vez): se cuenta y se dice.
		if len(fechas[0]) == 0 {
			sinFecha = append(sinFecha, r.o.ID)
			continue
		}
		for hito, uno := range fechas[0] {
			dos, ok := fechas[1][hito]
			if !ok {
				t.Errorf("%s: el hito %q tiene fecha con una constancia y no con la otra",
					r.o.ID, hito)
				continue
			}
			// LA AFIRMACION QUE NO SE PODIA HACER ANTES: son dos fechas, no una.
			if uno.Equal(dos) {
				t.Errorf("%s: el hito %q da la MISMA fecha (%s) para dos incidentes conocidos "+
					"el %s y el %s: se estan pisando", r.o.ID, hito,
					uno.Format(time.RFC3339), primero, segundo)
			}
			// Y en el orden correcto: saberlo despues no puede adelantar el plazo.
			if dos.Before(uno) {
				t.Errorf("%s: el hito %q vence ANTES para el incidente conocido despues "+
					"(%s frente a %s)", r.o.ID, hito,
					dos.Format(time.RFC3339), uno.Format(time.RFC3339))
			}
			comprobados++
		}
	}
	if comprobados == 0 {
		t.Fatal("ningun hito comparado: el test no ha comprobado nada")
	}
	if !t.Failed() {
		t.Logf("%d hitos comparados en dos incidentes distintos; %d relojes sin fecha propia "+
			"(todos sus hitos cuelgan de otro, traen tope o la norma no les da plazo): %v",
			comprobados, len(sinFecha), sinFecha)
	}
}

// ---------------------------------------------------------------------------
// Reclasificar mueve el hito que rige, y no borra nada
// ---------------------------------------------------------------------------

// Reclasificar NO edita la clasificacion anterior: anade una nueva, y el plazo
// se recalcula solo porque el motor elige por la mas reciente.
//
// Lo que se comprueba es que CAMBIA EL HITO QUE RIGE, no solo la fecha:
// ensenar los dos a la vez le daria al operador dos fechas para la misma
// obligacion sin forma de saber cual es la suya.
func TestReclasificarCambiaElHitoQueRigeYNoBorraLaClasificacionAnterior(t *testing.T) {
	primera := instanteC(t, "2026-03-02T12:00:00Z")
	segunda := instanteC(t, "2026-03-04T18:00:00Z")

	relojes := 0
	for _, r := range relojesDeIncidente(t) {
		cs := r.clases()
		if len(cs) < 2 {
			continue // aqui no hay nada que reclasificar
		}
		relojes++
		i, err := incidente.Abrir("incidente/2026-021",
			instanteC(t, "2026-03-01T04:00:00Z"), instanteC(t, "2026-03-02T09:00:00Z"), "soc")
		if err != nil {
			t.Fatal(err)
		}
		clasifica := func(clase string, en time.Time) {
			t.Helper()
			if err := i.Registrar(incidente.Suceso{Tipo: incidente.Clasificacion, Clase: clase,
				InstanteHecho: en, InstanteRegistro: en, Fuente: "responsable"}); err != nil {
				t.Fatal(err)
			}
		}
		// Solo se exige fecha a los hitos que pueden darla con el disparador: los
		// encadenados esperan una remision que aqui no consta, y eso es correcto.
		rigen := func(clase string, fechas map[string]time.Time, quiere bool) {
			t.Helper()
			for _, h := range r.hitosDeClase(clase) {
				if !conPlazoPropio(h) {
					continue
				}
				if _, ok := fechas[h.ID]; ok != quiere {
					if quiere {
						t.Errorf("%s: clasificado como %q, el hito %q no rige", r.o.ID, clase, h.ID)
					} else {
						t.Errorf("%s: sigue rigiendo el hito %q de la clasificacion vieja (%q), "+
							"con fecha %s. Dos fechas para la misma obligacion es peor que "+
							"ninguna", r.o.ID, h.ID, clase, fechas[h.ID].Format(time.RFC3339))
					}
				}
			}
		}

		clasifica(cs[0], primera)
		rigen(cs[0], determinados(plazosDe(t, r, i, ventana.Hechos{})), true)

		clasifica(cs[1], segunda)
		conLaSegunda := determinados(plazosDe(t, r, i, ventana.Hechos{}))
		rigen(cs[1], conLaSegunda, true)
		rigen(cs[0], conLaSegunda, false) // el viejo ya no

		// Y LA CLASIFICACION ANTERIOR NO SE HA BORRADO: sigue en el flujo, con
		// su instante. Es lo que un auditor pregunta: que se creyo y desde cuando.
		if clase, empate, ok := i.ClaseEn(instanteC(t, "2026-03-03T00:00:00Z")); !ok ||
			empate || clase != cs[0] {
			t.Errorf("%s: el 3 de marzo la clasificacion vigente era %q y sale %q",
				r.o.ID, cs[0], clase)
		}
		if n := len(i.Sucesos()); n != 3 {
			t.Errorf("%s: el flujo tiene %d sucesos y tenia que tener 3: reclasificar ha "+
				"borrado algo", r.o.ID, n)
		}
	}
	if relojes == 0 {
		t.Fatal("ningun reloj con dos clases: el test no ha comprobado nada")
	}
}

// Sin clasificar, el motor NO calla y NO acusa: dice que falta un dato que pone
// el obligado. La alternativa (que el hito desaparezca de la lista) se leeria
// como "nada que hacer", que es el peor error posible aqui.
func TestSinClasificarNingunHitoDeClaseDesaparece(t *testing.T) {
	relojes := 0
	for _, r := range relojesDeIncidente(t) {
		if len(r.clases()) == 0 {
			continue
		}
		relojes++
		i, err := incidente.Abrir("incidente/2026-022",
			instanteC(t, "2026-03-01T04:00:00Z"), instanteC(t, "2026-03-02T09:00:00Z"), "soc")
		if err != nil {
			t.Fatal(err)
		}
		visto := map[string]ventana.EstadoVenc{}
		for _, v := range plazosDe(t, r, i, ventana.Hechos{}) {
			visto[v.Hito] = v.Estado
		}
		for _, h := range r.hitos {
			if h.Clase == "" {
				continue
			}
			e, ok := visto[h.ID]
			if !ok {
				t.Errorf("%s: sin clasificar, el hito %q no sale en ningun sitio", r.o.ID, h.ID)
				continue
			}
			if e != ventana.PendienteDeHecho {
				t.Errorf("%s: sin que nadie clasifique, el hito %q sale como %q", r.o.ID, h.ID, e)
			}
		}
	}
	if relojes == 0 {
		t.Fatal("ningun reloj con clases: el test no ha comprobado nada")
	}
}

// ---------------------------------------------------------------------------
// El hito encadenado: cuenta desde la remision EFECTIVA
// ---------------------------------------------------------------------------

// Un hito encadenado cuenta desde la REMISION de verdad del anterior, no desde
// cuando deberia haberse remitido: quien notifica tarde no tiene menos tiempo
// para el informe final del que la norma le da.
//
// Es lo que pidio la medicion de la familia A, y hasta hoy no habia de donde
// sacar el hecho ".cumplido" fuera de un caso dorado.
//
// LA PROPIEDAD, escrita para que no dependa de ninguna fecha concreta: mover la
// remision mueve el hito encadenado y NO mueve a los demas. La segunda mitad es
// la que distingue "cuenta desde la remision" de "cuenta desde cualquier cosa
// que hayamos tocado".
func TestElHitoEncadenadoSigueALaRemisionYLosDemasNo(t *testing.T) {
	comprobados := 0
	for _, r := range relojesDeIncidente(t) {
		if len(r.encadenados()) == 0 {
			continue
		}
		// UNA CLASE POR PASADA. Poner todas a la vez es un empate, y un empate
		// no da fecha a proposito: el test se quedaria verde sin comprobar nada
		// si ademas se le hubiera relajado la afirmacion.
		clases := r.clases()
		if len(clases) == 0 {
			clases = []string{""}
		}
		for _, clase := range clases {
			ajenos := ventana.Hechos{}
			if clase != "" {
				ajenos[clase] = instanteC(t, "2020-01-01T00:00:00Z")
			}
			var cadena []corpus.HitoSpec
			for _, h := range r.encadenados() {
				if h.Clase == clase {
					cadena = append(cadena, h)
				}
			}
			if len(cadena) == 0 {
				continue
			}

			conRemision := func(id, cuando string) map[string]time.Time {
				t.Helper()
				i, err := incidente.Abrir(id, instanteC(t, "2026-03-01T04:00:00Z"),
					instanteC(t, "2026-03-02T09:00:00Z"), "soc")
				if err != nil {
					t.Fatal(err)
				}
				if cuando != "" {
					x := instanteC(t, cuando)
					puestos := map[string]bool{}
					for _, h := range cadena {
						if puestos[h.DesdeHito] {
							continue
						}
						puestos[h.DesdeHito] = true
						if err := i.Registrar(incidente.Suceso{Tipo: incidente.Notificacion,
							Hito: h.DesdeHito, InstanteHecho: x, InstanteRegistro: x,
							Fuente: "csirt"}); err != nil {
							t.Fatal(err)
						}
					}
				}
				return determinados(plazosDe(t, r, i, ajenos))
			}

			// Sin la remision registrada, el encadenado NO tiene fecha.
			for _, h := range cadena {
				if v, ok := conRemision("incidente/2026-030", "")[h.ID]; ok {
					t.Errorf("%s: sin constar la remision de %q, el hito %q ya tiene fecha "+
						"(%s): esta contando desde un hecho que no ha ocurrido", r.o.ID,
						h.DesdeHito, h.ID, v.Format(time.RFC3339))
				}
			}

			// Y la remision, tarde y mas tarde: el encadenado se mueve con ella.
			pronto := conRemision("incidente/2026-031", "2026-03-06T11:00:00Z")
			tarde := conRemision("incidente/2026-032", "2026-03-09T11:00:00Z")
			esCadena := map[string]bool{}
			for _, h := range cadena {
				esCadena[h.ID] = true
				a, oka := pronto[h.ID]
				b, okb := tarde[h.ID]
				if !oka || !okb {
					// Un encadenado puede colgar de OTRO encadenado (el informe
					// final que cuelga del intermedio): entonces hace falta la
					// remision de los dos, y aqui solo consta la del padre
					// directo de la cadena declarada. No es un fallo y no se
					// calla: se salta con su motivo.
					continue
				}
				if !b.After(a) {
					t.Errorf("%s: remitir tres dias mas tarde deja el hito %q en %s (antes "+
						"%s): no esta contando desde la remision", r.o.ID, h.ID,
						b.Format(time.RFC3339), a.Format(time.RFC3339))
				}
				comprobados++
			}
			// LOS DEMAS NO SE MUEVEN. Es la mitad que distingue "cuenta desde la
			// remision" de "cuenta desde cualquier cosa que hayamos tocado".
			for hito, a := range pronto {
				b, ok := tarde[hito]
				if !ok || esCadena[hito] {
					continue
				}
				if !a.Equal(b) {
					t.Errorf("%s: el hito %q, que no cuelga de ningun otro, se movio de %s a "+
						"%s al cambiar la fecha de remision", r.o.ID, hito,
						a.Format(time.RFC3339), b.Format(time.RFC3339))
				}
			}
		}
	}
	if comprobados == 0 {
		t.Fatal("ningun hito encadenado comprobado: el test no ha comprobado nada")
	}
	if !t.Failed() {
		t.Logf("%d hitos encadenados comprobados", comprobados)
	}
}
