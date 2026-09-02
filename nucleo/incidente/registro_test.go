package incidente

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// EL REGISTRO EN DISCO: ida y vuelta, y lo que NO se admite.
//
// La propiedad que se persigue no es «el JSON parsea»: es que un incidente
// leido de disco haya pasado POR LAS MISMAS REGLAS que uno creado a mano. Si el
// lector rellenara campos privados, el fichero seria la autoridad y las reglas
// se quedarian fuera, que es exactamente como se cuela un incidente sin
// apertura en un expediente que despues se firma.

func ts(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return v.UTC()
}

func unoDeVerdad(t *testing.T) *Incidente {
	t.Helper()
	i, err := Abrir("inc-2026-014",
		ts(t, "2026-08-30T22:15:00Z"), ts(t, "2026-08-31T07:40:00Z"), "SIEM")
	if err != nil {
		t.Fatal(err)
	}
	if err := i.Registrar(Suceso{Tipo: Clasificacion, Clase: "incidente.nivel.alto",
		InstanteHecho: ts(t, "2026-08-31T09:00:00Z"), InstanteRegistro: ts(t, "2026-08-31T09:05:00Z")}); err != nil {
		t.Fatal(err)
	}
	if err := i.Registrar(Suceso{Tipo: Notificacion, Hito: "notificacion_inicial",
		InstanteHecho: ts(t, "2026-09-01T08:00:00Z"), InstanteRegistro: ts(t, "2026-09-01T08:02:00Z")}); err != nil {
		t.Fatal(err)
	}
	return i
}

// LA IDA Y VUELTA. Se compara lo que importa (id y sucesos), no los bytes: dos
// serializaciones distintas del mismo incidente son el mismo incidente, y atar
// el test a los bytes convertiria cualquier cambio de formato en un rojo que
// invita a regenerar el fichero esperado sin mirar.
func TestLoEscritoSeReleeIgual(t *testing.T) {
	orig := unoDeVerdad(t)
	b, err := Escribir([]*Incidente{orig})
	if err != nil {
		t.Fatalf("escribir: %v", err)
	}
	vuelta, err := Reconstruir(b)
	if err != nil {
		t.Fatalf("releer lo que acabamos de escribir: %v\n%s", err, b)
	}
	if len(vuelta) != 1 {
		t.Fatalf("han vuelto %d incidentes y se escribio 1", len(vuelta))
	}
	leido := vuelta[0]
	if leido.ID() != orig.ID() {
		t.Errorf("id: %q frente a %q", leido.ID(), orig.ID())
	}
	if !leido.Abierto() {
		t.Fatal("el incidente releido no esta abierto, asi que no paso por Abrir")
	}
	a, b2 := orig.Sucesos(), leido.Sucesos()
	if len(a) != len(b2) {
		t.Fatalf("sucesos: %d frente a %d", len(a), len(b2))
	}
	for i := range a {
		if a[i].Tipo != b2[i].Tipo || a[i].Clase != b2[i].Clase || a[i].Hito != b2[i].Hito ||
			!a[i].InstanteHecho.Equal(b2[i].InstanteHecho) ||
			!a[i].InstanteRegistro.Equal(b2[i].InstanteRegistro) {
			t.Errorf("el suceso %d no vuelve igual:\n  escrito %+v\n  leido   %+v", i, a[i], b2[i])
		}
	}
	// Y LO QUE DE VERDAD IMPORTA: los datos con consecuencia legal salen igual.
	oc1, ok1 := orig.PrimerConocimiento()
	oc2, ok2 := leido.PrimerConocimiento()
	if !ok1 || !ok2 || !oc1.Equal(oc2) {
		t.Errorf("el primer conocimiento no sobrevive a la ida y vuelta: %v/%t frente a %v/%t",
			oc1, ok1, oc2, ok2)
	}
}

// EL TIPO VIAJA POR SU NOMBRE, y esto es lo que lo demuestra.
//
// Si se serializara el uint8, insertar un tipo nuevo en medio del iota
// reinterpretaria en silencio todos los ficheros escritos: un `cierre` pasaria a
// leerse como `notificacion`. Nadie firma el orden de un iota.
func TestElTipoDeSucesoViajaPorSuNombreYNoPorSuNumero(t *testing.T) {
	b, err := Escribir([]*Incidente{unoDeVerdad(t)})
	if err != nil {
		t.Fatal(err)
	}
	texto := string(b)
	for _, n := range nombresDeTipo[:3] {
		if !strings.Contains(texto, `"`+n+`"`) {
			t.Errorf("el documento no trae el tipo %q escrito con su nombre:\n%s", n, texto)
		}
	}
	if strings.Contains(texto, `"tipo": 0`) || strings.Contains(texto, `"tipo": 1`) {
		t.Error("el tipo se esta escribiendo como numero: el fichero queda atado al orden " +
			"del iota, y anadir un tipo en medio reinterpretaria los ficheros ya escritos")
	}
}

// LO QUE NO SE ADMITE. Cada caso es una forma real de que un fichero mienta, y
// ninguna puede producir un objeto a medias: o sale entero y valido, o error.
func TestUnRegistroQueNoCuadraNoProduceUnIncidenteAMedias(t *testing.T) {
	casos := []struct {
		nombre     string
		doc        string
		centinela  error
		yEnElTexto string
	}{
		{"sin version", `{"incidentes":[]}`, ErrRegistroSinVersion, "version"},
		{"version del futuro", `{"version":99,"incidentes":[]}`, ErrRegistroVersionDesconocida, "99"},
		{"no es json", `{`, ErrRegistroIlegible, ""},
		{"incidente sin sucesos",
			`{"version":1,"incidentes":[{"id":"a","sucesos":[]}]}`, ErrRegistroIlegible, "apertura"},
		{"el primero no es la apertura",
			`{"version":1,"incidentes":[{"id":"a","sucesos":[
			  {"tipo":"clasificacion","clase":"c","instante_hecho":"2026-01-01T00:00:00Z","instante_registro":"2026-01-01T00:00:00Z"}]}]}`,
			ErrRegistroIlegible, "apertura"},
		{"dos incidentes con el mismo id",
			`{"version":1,"incidentes":[
			  {"id":"a","sucesos":[{"tipo":"apertura","instante_hecho":"2026-01-01T00:00:00Z","instante_registro":"2026-01-01T00:00:00Z"}]},
			  {"id":"a","sucesos":[{"tipo":"apertura","instante_hecho":"2026-01-01T00:00:00Z","instante_registro":"2026-01-01T00:00:00Z"}]}]}`,
			ErrIncidenteRepetido, "a"},
		{"tipo inventado",
			`{"version":1,"incidentes":[{"id":"a","sucesos":[
			  {"tipo":"apertura","instante_hecho":"2026-01-01T00:00:00Z","instante_registro":"2026-01-01T00:00:00Z"},
			  {"tipo":"invento","instante_hecho":"2026-01-02T00:00:00Z","instante_registro":"2026-01-02T00:00:00Z"}]}]}`,
			ErrSucesoIlegible, "invento"},
		// LA TERCERA FORMA DE LA NADA, y es la que sale por descuido: un
		// instante presente y no interpretable. No puede convertirse en el cero
		// de time.Time, que es el ano 1 con cara de dato.
		{"instante que no se entiende",
			`{"version":1,"incidentes":[{"id":"a","sucesos":[
			  {"tipo":"apertura","instante_hecho":"el martes","instante_registro":"2026-01-01T00:00:00Z"}]}]}`,
			ErrSucesoIlegible, "el martes"},
		{"instante vacio",
			`{"version":1,"incidentes":[{"id":"a","sucesos":[
			  {"tipo":"apertura","instante_hecho":"","instante_registro":"2026-01-01T00:00:00Z"}]}]}`,
			ErrSucesoIlegible, "vacio"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			is, err := Reconstruir([]byte(c.doc))
			if err == nil {
				t.Fatalf("se ha aceptado un registro que no vale, y ha dado %d incidentes", len(is))
			}
			if !errors.Is(err, c.centinela) {
				t.Errorf("el error no es el centinela %v: %v", c.centinela, err)
			}
			if is != nil {
				t.Error("con error tiene que volver nil: si no, quien llame podria usar la " +
					"lista a medias sin mirar el error")
			}
			if c.yEnElTexto != "" && !strings.Contains(err.Error(), c.yEnElTexto) {
				t.Errorf("el error no dice %q, asi que no es accionable: %v", c.yEnElTexto, err)
			}
		})
	}
}

// LAS REGLAS DEL OBJETO SIGUEN VIGENTES AL LEER DE DISCO, y esto es la razon de
// ser de todo el fichero.
//
// Un fichero con un campo ajeno (una clase en un suceso que no es clasificacion)
// tiene que morir en Registrar, sin que este lector repita esa regla: una
// segunda copia de las reglas se separa de la primera, y la que se separa es la
// que se aplica a los ficheros que vienen de fuera.
func TestElLectorNoSaltaLasReglasDelObjeto(t *testing.T) {
	doc := `{"version":1,"incidentes":[{"id":"a","sucesos":[
	  {"tipo":"apertura","instante_hecho":"2026-01-01T00:00:00Z","instante_registro":"2026-01-01T00:00:00Z"},
	  {"tipo":"notificacion","clase":"esta_clase_no_pinta_nada","hito":"h",
	   "instante_hecho":"2026-01-02T00:00:00Z","instante_registro":"2026-01-02T00:00:00Z"}]}]}`
	if _, err := Reconstruir([]byte(doc)); err == nil {
		t.Fatal("se ha leido de disco un suceso con un campo ajeno que el objeto rechaza " +
			"cuando lo construye una persona. Entonces el fichero es la autoridad y las " +
			"reglas se han quedado fuera")
	} else if !errors.Is(err, ErrCampoAjeno) {
		t.Errorf("el error tenia que venir del propio objeto (ErrCampoAjeno) y vino de "+
			"otro sitio: %v", err)
	}

	// CONTROL POSITIVO: el mismo suceso SIN el campo ajeno si entra. Sin esta
	// rama, un lector que rechazara todas las notificaciones pasaria el caso de
	// arriba por el motivo equivocado.
	bueno := strings.Replace(doc, `"clase":"esta_clase_no_pinta_nada",`, "", 1)
	if _, err := Reconstruir([]byte(bueno)); err != nil {
		t.Errorf("el documento correcto tampoco carga, asi que el lector dice que no a "+
			"todo: %v", err)
	}
}

// UN REGISTRO VACIO ES LEGITIMO, Y NO ES LO MISMO QUE NO TENER REGISTRO.
//
// Es la distincion entera por la que el acta tiene el campo
// HayRegistroDeIncidentes: «cero incidentes en el periodo» es una noticia y
// «nadie ha conectado el registro» es un hueco, y se leen al reves. Aqui se
// comprueba que el lector sabe devolver la primera.
func TestUnRegistroConCeroIncidentesSeLeeSinError(t *testing.T) {
	is, err := Reconstruir([]byte(`{"version":1,"incidentes":[]}`))
	if err != nil {
		t.Fatalf("un registro conectado y sin incidentes tiene que leerse bien: %v", err)
	}
	if len(is) != 0 {
		t.Errorf("han salido %d incidentes de un registro vacio", len(is))
	}
}
