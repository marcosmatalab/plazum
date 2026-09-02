package auditoria

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

// EL PROGRAMA EN DISCO: ida y vuelta, y lo que NO se admite.
//
// La propiedad es la misma que en el registro de incidentes: un programa leido
// de disco tiene que haber pasado POR LAS MISMAS reglas que uno construido a
// mano. Si el lector rellenara campos privados, el fichero seria la autoridad y
// las reglas se quedarian fuera, que es como entra un hallazgo que apunta a una
// unidad que no esta en el alcance.

// unidadesDePrueba da el alcance. Las claves de los hechos se sacan de
// Unidad.Clave() y NO se escriben a mano: el formato de la clave es del objeto,
// y escribirlo aqui seria una segunda copia que se queda vieja el dia que el
// separador cambie. Esta primera version las escribio a mano, con el formato
// equivocado, y lo caza el control positivo del lector.
func unidadesDePrueba() []Unidad {
	return []Unidad{
		{Paquete: "marco.a", Version: "1", Obligacion: "o1", Titulo: "La primera"},
		{Paquete: "marco.a", Version: "1", Obligacion: "o2", Titulo: "La segunda"},
	}
}

func inst(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return v.UTC()
}

func programaDeVerdad(t *testing.T) *Programa {
	t.Helper()
	p, err := Abrir("prog-2026", Ciclo{
		Nombre: "2026-2028",
		Desde:  inst(t, "2026-01-01T00:00:00Z"),
		Hasta:  inst(t, "2028-12-31T00:00:00Z"),
	}, unidadesDePrueba(), Arrastre{
		DeCiclo:    "2023-2025",
		SinAuditar: map[string]int{unidadesDePrueba()[1].Clave(): 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Auditar(Sesion{
		ID: "s1", Auditor: "aud-01", Cuando: inst(t, "2026-03-10T09:00:00Z"),
		Unidades: []string{unidadesDePrueba()[0].Clave()},
		Alcance:  "revision documental y muestreo",
	}); err != nil {
		t.Fatal(err)
	}
	if err := p.Anotar(Hallazgo{
		ID: "h1", Sesion: "s1", Unidad: unidadesDePrueba()[0].Clave(), Clase: NoConformidadMenor,
		Texto: "un caso sin registrar", Quien: "aud-01",
		Cuando: inst(t, "2026-03-10T12:00:00Z"),
	}); err != nil {
		t.Fatal(err)
	}
	if err := p.Cerrar(CierreDeHallazgo{
		Hallazgo: "h1", Quien: "resp-02", Cuando: inst(t, "2026-04-01T09:00:00Z"),
		Como: "se anadio el registro y se reviso el procedimiento",
	}); err != nil {
		t.Fatal(err)
	}
	if err := p.Diferir(Diferimiento{
		Unidad: unidadesDePrueba()[1].Clave(), Quien: "resp-01",
		Motivo: "la unidad se externaliza en junio",
		Cuando: inst(t, "2026-02-01T09:00:00Z"),
	}); err != nil {
		t.Fatal(err)
	}
	return p
}

// LA IDA Y VUELTA, comparada por lo que importa y no por los bytes.
func TestElProgramaEscritoSeReleeIgual(t *testing.T) {
	orig := programaDeVerdad(t)
	b, err := Escribir(orig)
	if err != nil {
		t.Fatalf("escribir: %v", err)
	}
	leido, err := Reconstruir(b)
	if err != nil {
		t.Fatalf("releer lo que acabamos de escribir: %v\n%s", err, b)
	}
	if leido.ID() != orig.ID() {
		t.Errorf("id: %q frente a %q", leido.ID(), orig.ID())
	}
	if leido.Alcance() != orig.Alcance() {
		t.Errorf("alcance: %d unidades frente a %d", leido.Alcance(), orig.Alcance())
	}
	if len(leido.Sesiones()) != len(orig.Sesiones()) ||
		len(leido.Hallazgos()) != len(orig.Hallazgos()) ||
		len(leido.Diferimientos()) != len(orig.Diferimientos()) ||
		len(leido.Cierres()) != len(orig.Cierres()) {
		t.Fatalf("los conjuntos no vuelven completos: sesiones %d/%d, hallazgos %d/%d, "+
			"diferimientos %d/%d, cierres %d/%d",
			len(leido.Sesiones()), len(orig.Sesiones()),
			len(leido.Hallazgos()), len(orig.Hallazgos()),
			len(leido.Diferimientos()), len(orig.Diferimientos()),
			len(leido.Cierres()), len(orig.Cierres()))
	}

	// LO QUE DE VERDAD IMPORTA: los datos con consecuencia en el acta salen
	// iguales. La cuenta por cobertura es lo que un consejo mira, y los
	// hallazgos abiertos son lo que le hace tomar una decision.
	if len(leido.Abiertos()) != len(orig.Abiertos()) {
		t.Errorf("hallazgos abiertos: %d frente a %d. Un cierre perdido en la ida y vuelta "+
			"hace que el acta diga que hay no conformidades sin cerrar que si se cerraron",
			len(leido.Abiertos()), len(orig.Abiertos()))
	}
	for _, c := range CoberturasPosibles() {
		if leido.Cuenta()[c] != orig.Cuenta()[c] {
			t.Errorf("la cobertura %q vuelve como %d y era %d",
				c, leido.Cuenta()[c], orig.Cuenta()[c])
		}
	}
}

// EL CIERRE SOBREVIVE, y esta comprobado aparte porque su accesor NO EXISTIA.
//
// `Cierres()` se anadio para poder serializar. Sin el, el escritor habria
// perdido los cierres EN SILENCIO y al releer el programa los hallazgos cerrados
// volverian a salir abiertos: el acta diria que hay no conformidades sin cerrar
// que alguien cerro con su fecha y su como. Es acusar en falso por un accesor
// que faltaba, asi que esta rama va con nombre propio.
func TestUnHallazgoCerradoNoVuelveAbiertoAlReleerlo(t *testing.T) {
	orig := programaDeVerdad(t)
	if len(orig.Abiertos()) != 0 {
		t.Fatalf("el programa de prueba tenia que traer su unico hallazgo cerrado, y trae %d "+
			"abiertos: este test no estaria midiendo lo que cree", len(orig.Abiertos()))
	}
	b, err := Escribir(orig)
	if err != nil {
		t.Fatal(err)
	}
	leido, err := Reconstruir(b)
	if err != nil {
		t.Fatal(err)
	}
	if abiertos := leido.Abiertos(); len(abiertos) != 0 {
		t.Errorf("tras la ida y vuelta hay %d hallazgos abiertos y no habia ninguno: %v",
			len(abiertos), abiertos)
	}
}

// LA CLASE VIAJA POR SU NOMBRE. Si viajara como numero, insertar una clase nueva
// en medio del iota convertiria una «no conformidad mayor» en una «menor» en
// todos los ficheros ya escritos, y eso es lo primero que mira un consejo.
func TestLaClaseDeHallazgoViajaPorSuNombre(t *testing.T) {
	b, err := Escribir(programaDeVerdad(t))
	if err != nil {
		t.Fatal(err)
	}
	texto := string(b)
	if !strings.Contains(texto, `"`+NoConformidadMenor.String()+`"`) {
		t.Errorf("la clase no sale con su nombre:\n%s", texto)
	}
	if strings.Contains(texto, `"clase": 1`) {
		t.Error("la clase se escribe como numero: el fichero queda atado al orden del iota")
	}
}

// LO QUE NO SE ADMITE.
func TestUnProgramaQueNoCuadraNoSeLee(t *testing.T) {
	base := `"ciclo":{"nombre":"c","desde":"2026-01-01T00:00:00Z","hasta":"2028-01-01T00:00:00Z"},
	         "alcance":[{"paquete":"m","version":"1","obligacion":"o1"}]`
	casos := []struct {
		nombre    string
		doc       string
		centinela error
	}{
		{"sin version", `{"id":"p",` + base + `}`, ErrProgramaSinVersion},
		{"version del futuro", `{"version":77,"id":"p",` + base + `}`, ErrProgramaVersionDesconocida},
		{"no es json", `{`, ErrProgramaIlegible},
		{"clase de hallazgo inventada",
			`{"version":1,"id":"p",` + base + `,
			 "sesiones":[{"id":"s1","auditor":"a","cuando":"2026-03-01T00:00:00Z","unidades":["m|o1"]}],
			 "hallazgos":[{"id":"h1","sesion":"s1","unidad":"m|o1","clase":"gravisima","texto":"t","quien":"q","cuando":"2026-03-01T00:00:00Z"}]}`,
			ErrClaseDesconocida},
		// LA TERCERA FORMA DE LA NADA: presente y no interpretable.
		{"fecha de ciclo que no se entiende",
			`{"version":1,"id":"p","ciclo":{"nombre":"c","desde":"el lunes","hasta":"2028-01-01T00:00:00Z"},
			  "alcance":[{"paquete":"m","version":"1","obligacion":"o1"}]}`,
			ErrInstanteIlegible},
		{"fecha de ciclo vacia",
			`{"version":1,"id":"p","ciclo":{"nombre":"c","desde":"","hasta":"2028-01-01T00:00:00Z"},
			  "alcance":[{"paquete":"m","version":"1","obligacion":"o1"}]}`,
			ErrInstanteIlegible},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			p, err := Reconstruir([]byte(c.doc))
			if err == nil {
				t.Fatal("se ha aceptado un programa que no vale")
			}
			if !errors.Is(err, c.centinela) {
				t.Errorf("el error no es el centinela %v: %v", c.centinela, err)
			}
			if p != nil {
				t.Error("con error tiene que volver nil")
			}
		})
	}
}

// LAS REGLAS DEL OBJETO SIGUEN VIGENTES AL LEER DE DISCO.
//
// Un hallazgo que apunta a una unidad que NO esta en el alcance tiene que morir
// en Anotar, sin que el lector repita esa regla: una segunda copia se separa de
// la primera, y la que se separa es la que se aplica a lo que viene de fuera.
func TestElLectorDelProgramaNoSaltaLasReglasDelObjeto(t *testing.T) {
	doc := `{"version":1,"id":"p",
	  "ciclo":{"nombre":"c","desde":"2026-01-01T00:00:00Z","hasta":"2028-01-01T00:00:00Z"},
	  "alcance":[{"paquete":"m","version":"1","obligacion":"o1"}],
	  "sesiones":[{"id":"s1","auditor":"a","cuando":"2026-03-01T00:00:00Z","unidades":["m|o1"]}],
	  "hallazgos":[{"id":"h1","sesion":"s1","unidad":"m|NO_EXISTE","clase":"no conformidad menor",
	                "texto":"t","quien":"q","cuando":"2026-03-01T00:00:00Z"}]}`
	if _, err := Reconstruir([]byte(doc)); err == nil {
		t.Fatal("se ha leido de disco un hallazgo que apunta a una unidad fuera del alcance, " +
			"y el objeto lo rechaza cuando lo construye una persona. Entonces el fichero es " +
			"la autoridad y las reglas se han quedado fuera")
	}
	// CONTROL POSITIVO: el mismo documento con la unidad buena SI carga. Sin
	// esta rama, un lector que rechazara todos los hallazgos pasaria el caso de
	// arriba por el motivo equivocado.
	bueno := strings.Replace(doc, "m|NO_EXISTE", "m|o1", 1)
	if _, err := Reconstruir([]byte(bueno)); err != nil {
		t.Errorf("el documento correcto tampoco carga, asi que el lector dice que no a "+
			"todo: %v", err)
	}
}

// EL ORDEN DEL FICHERO NO DECIDE. Un documento con los cierres escritos antes
// que sus hallazgos tiene que cargar igual: se reproduce por dependencias, no
// por el orden en que aparecen. Sin esto, un fichero valido escrito por otra
// herramienta fallaria con un error que habla de un hallazgo inexistente.
func TestElOrdenDelDocumentoNoDecideSiCarga(t *testing.T) {
	// Los cierres van ANTES que los hallazgos en el JSON.
	doc := `{"version":1,"id":"p",
	  "ciclo":{"nombre":"c","desde":"2026-01-01T00:00:00Z","hasta":"2028-01-01T00:00:00Z"},
	  "alcance":[{"paquete":"m","version":"1","obligacion":"o1"}],
	  "cierres":[{"hallazgo":"h1","quien":"q","cuando":"2026-04-01T00:00:00Z","como":"se arreglo"}],
	  "hallazgos":[{"id":"h1","sesion":"s1","unidad":"m|o1","clase":"no conformidad menor",
	                "texto":"t","quien":"q","cuando":"2026-03-01T00:00:00Z"}],
	  "sesiones":[{"id":"s1","auditor":"a","cuando":"2026-03-01T00:00:00Z","unidades":["m|o1"]}]}`
	p, err := Reconstruir([]byte(doc))
	if err != nil {
		t.Fatalf("un documento valido con los bloques en otro orden no carga: %v", err)
	}
	if len(p.Abiertos()) != 0 {
		t.Errorf("el cierre no se ha aplicado: quedan %d hallazgos abiertos", len(p.Abiertos()))
	}
}

// NINGUN CAMPO DEL DOMINIO SE PIERDE AL ESCRIBIR, Y LO VIGILA EL TIPO.
//
// POR QUE EXISTE, y salio mientras se escribia este fichero: la primera version
// de `hallazgoEnDisco` NO LLEVABA `Quien`. Un hallazgo escrito a disco perdia
// quien lo anoto, en silencio, y al releerlo salia sin autor. La ida y vuelta de
// arriba no lo veia porque compara recuentos y cobertura, no campo a campo, asi
// que el fallo habria viajado entero.
//
// Contar campos es una guarda tosca y es exactamente la que hace falta: no
// comprueba que el mapeo sea correcto (eso lo hace la ida y vuelta), comprueba
// que NADIE ANADA UN CAMPO AL DOMINIO SIN PASAR POR AQUI. El dia que alguien
// anada `Severidad` a Hallazgo, este test se pone rojo y le obliga a decidir si
// viaja o no; con la ida y vuelta sola, el campo nuevo se perderia y todo
// seguiria verde.
//
// Es la misma forma que TestCadaCampoDeTextoDelFormatoEstaClasificado en corpus.
func TestNingunCampoDelDominioSePierdeAlEscribirlo(t *testing.T) {
	casos := []struct {
		nombre        string
		dominio       any
		disco         any
		enDominioSolo []string // campos que a proposito NO viajan, con su motivo
	}{
		{"Unidad", Unidad{}, unidadEnDisco{}, nil},
		{"Sesion", Sesion{}, sesionEnDisco{}, nil},
		{"Hallazgo", Hallazgo{}, hallazgoEnDisco{}, nil},
		{"CierreDeHallazgo", CierreDeHallazgo{}, cierreEnDisco{}, nil},
		{"Diferimiento", Diferimiento{}, diferimientoEnDisco{}, nil},
		{"Arrastre", Arrastre{}, arrastreEnDisco{}, nil},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			dom := reflect.TypeOf(c.dominio)
			dis := reflect.TypeOf(c.disco)
			quiere := dom.NumField() - len(c.enDominioSolo)
			if dis.NumField() != quiere {
				var faltan, sobran []string
				enDisco := map[string]bool{}
				for i := 0; i < dis.NumField(); i++ {
					enDisco[strings.ToLower(dis.Field(i).Name)] = true
				}
				for i := 0; i < dom.NumField(); i++ {
					if !enDisco[strings.ToLower(dom.Field(i).Name)] {
						faltan = append(faltan, dom.Field(i).Name)
					}
				}
				enDominio := map[string]bool{}
				for i := 0; i < dom.NumField(); i++ {
					enDominio[strings.ToLower(dom.Field(i).Name)] = true
				}
				for i := 0; i < dis.NumField(); i++ {
					if !enDominio[strings.ToLower(dis.Field(i).Name)] {
						sobran = append(sobran, dis.Field(i).Name)
					}
				}
				t.Errorf("%s tiene %d campos y su forma en disco %d.\n"+
					"  Sin viajar: %v\n  De mas en disco: %v\n"+
					"  Un campo que no viaja se pierde EN SILENCIO: paso con Hallazgo.Quien, "+
					"y un hallazgo sin autor en el acta es evidencia de nada. Arreglo: o "+
					"viaja, o se declara en enDominioSolo con su motivo escrito.",
					c.nombre, dom.NumField(), dis.NumField(), faltan, sobran)
			}
		})
	}
}

// CONTROL NEGATIVO DEL CONTADOR: si se le da un par que NO cuadra, tiene que
// decirlo. Sin esto, un contador que devolviera siempre cero diferencias
// dejaria verdes los seis casos de arriba.
func TestElContadorDeCamposCazaUnParQueNoCuadra(t *testing.T) {
	type dominioFalso struct {
		A, B, C string
	}
	type discoFalso struct {
		A, B string
	}
	dom := reflect.TypeOf(dominioFalso{})
	dis := reflect.TypeOf(discoFalso{})
	if dom.NumField() == dis.NumField() {
		t.Fatal("el par de prueba cuadra, asi que este control no prueba nada")
	}
}
