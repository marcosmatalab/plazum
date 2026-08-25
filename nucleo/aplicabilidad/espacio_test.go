package aplicabilidad

import (
	"errors"
	"strings"
	"testing"
)

// El test de la propiedad de producto: anadir la norma 31 no puede romper la 12.
//
// Dos normas distintas llaman `en_ambito` a cosas distintas, que es lo normal:
// cada marco define su propio ambito de aplicacion y ninguna consulta a la otra
// como llamo al suyo. Antes del aislamiento, las reglas de una derivaban sobre
// los hechos de la otra en silencio, y el resultado era una obligacion que
// aplica a quien no le aplica. Eso no es un caso borde: es lo que pasa el dia
// que el corpus pasa de dos paquetes con reglas a treinta.

func programaAlfa() Programa {
	return Programa{Paquete: "urn:demo:alfa", Reglas: []Regla{
		// Para alfa, estar en ambito es ser del sector publico.
		{ID: "ambito", Cita: "demo alfa art. 2",
			Cabeza: A("en_ambito", V("E")),
			Cuerpo: []Atomo{A("sector", V("E"), C("publico"))}},
		{ID: "obliga", Cita: "demo alfa art. 31",
			Cabeza: A("aplica", C("demo.alfa.auditoria"), V("E")),
			Cuerpo: []Atomo{A("en_ambito", V("E"))}},
	}}
}

func programaBeta() Programa {
	return Programa{Paquete: "urn:demo:beta", Reglas: []Regla{
		// Para beta, estar en ambito es tratar datos personales. Nada que ver.
		{ID: "ambito", Cita: "demo beta art. 3",
			Cabeza: A("en_ambito", V("E")),
			Cuerpo: []Atomo{A("trata_datos", V("E"))}},
		{ID: "obliga", Cita: "demo beta art. 33",
			Cabeza: A("aplica", C("demo.beta.notificacion"), V("E")),
			Cuerpo: []Atomo{A("en_ambito", V("E"))}},
	}}
}

func aplicablesDe(m *Motor, sujeto string) map[string]bool {
	out := map[string]bool{}
	for _, h := range m.Consultar(A("aplica", V("O"), C(sujeto))) {
		out[h.Args[0]] = true
	}
	return out
}

func TestDosPaquetesQueLlamanIgualASusPredicadosNoSeContaminan(t *testing.T) {
	m := NuevoMotor()
	cargar(t, m, programaAlfa())
	cargar(t, m, programaBeta())

	// Un ayuntamiento: sector publico Y trata datos. Le aplican las dos.
	m.Afirmar(H("sector", "ayto", "publico"))
	m.Afirmar(H("trata_datos", "ayto"))
	// Una tienda: trata datos y NO es sector publico. Solo le aplica beta.
	m.Afirmar(H("trata_datos", "tienda"))
	// Un organismo sin datos personales: solo le aplica alfa.
	m.Afirmar(H("sector", "organismo", "publico"))

	if _, err := m.Evaluar(); err != nil {
		t.Fatalf("evaluar: %v", err)
	}

	casos := []struct {
		sujeto string
		alfa   bool
		beta   bool
		porQue string
	}{
		{"ayto", true, true, "es sector publico y trata datos"},
		{"tienda", false, true, "trata datos pero NO es sector publico"},
		{"organismo", true, false, "es sector publico pero NO trata datos"},
	}
	for _, c := range casos {
		ap := aplicablesDe(m, c.sujeto)
		if ap["demo.alfa.auditoria"] != c.alfa {
			t.Errorf("%s (%s): alfa deberia ser %v y es %v. Si sale de mas, las reglas de beta "+
				"han derivado sobre el en_ambito de alfa: eso es anadir la norma 31 y romper la 12",
				c.sujeto, c.porQue, c.alfa, ap["demo.alfa.auditoria"])
		}
		if ap["demo.beta.notificacion"] != c.beta {
			t.Errorf("%s (%s): beta deberia ser %v y es %v", c.sujeto, c.porQue, c.beta,
				ap["demo.beta.notificacion"])
		}
	}
}

// CONTROL NEGATIVO. Sin el aislamiento, el test de arriba falla: es lo unico que
// demuestra que el aislamiento esta haciendo algo y no que el escenario da lo
// mismo con o sin el.
//
// Se monta el mismo escenario cargando los programas SIN pasar por el espacio de
// nombres, que es como estaba el motor antes.
func TestSinAislamientoLosDosPaquetesSiSeContaminan(t *testing.T) {
	m := NuevoMotor()
	// A pelo, saltandose Cargar: asi eran los programas antes del aislamiento.
	m.programas = append(m.programas, programaAlfa(), programaBeta())

	m.Afirmar(H("trata_datos", "tienda")) // NO es sector publico
	if _, err := m.Evaluar(); err != nil {
		t.Fatalf("evaluar: %v", err)
	}
	if !aplicablesDe(m, "tienda")["demo.alfa.auditoria"] {
		t.Fatal("sin aislamiento, la tienda TENIA que recibir la obligacion de alfa por " +
			"contaminacion: si no la recibe, el escenario no reproduce el fallo y el test de " +
			"arriba no demuestra nada")
	}
}

func TestNoSePuedeExportarLoQueNoSeDefine(t *testing.T) {
	p := Programa{Paquete: "urn:demo:x", Exporta: []string{"inventado"}, Reglas: []Regla{
		{ID: "r", Cita: "art. 1", Cabeza: A("otro", V("E")), Cuerpo: []Atomo{A("hecho", V("E"))}},
	}}
	err := NuevoMotor().Cargar(p)
	if !errors.Is(err, ErrExportaSinDefinir) {
		t.Fatalf("exportar lo que no se define tiene que rechazarse con ErrExportaSinDefinir, y dio: %v", err)
	}
	// Control positivo: exportando lo que si define, entra.
	p.Exporta = []string{"otro"}
	if err := NuevoMotor().Cargar(p); err != nil {
		t.Fatalf("exportar lo que si define tiene que valer: %v", err)
	}
}

func TestNoSePuedeExportarUnPredicadoComun(t *testing.T) {
	p := Programa{Paquete: "urn:demo:x", Exporta: []string{"aplica"}, Reglas: []Regla{
		{ID: "r", Cita: "art. 1", Cabeza: A("aplica", C("demo.o"), V("E")),
			Cuerpo: []Atomo{A("hecho", V("E"))}},
	}}
	if err := NuevoMotor().Cargar(p); !errors.Is(err, ErrExportaComun) {
		t.Fatalf("exportar un comun tiene que rechazarse con ErrExportaComun, y dio: %v", err)
	}
}

func TestDosPaquetesNoPuedenExportarElMismoPredicado(t *testing.T) {
	a := programaAlfa()
	a.Exporta = []string{"en_ambito"}
	b := programaBeta()
	b.Exporta = []string{"en_ambito"}

	m := NuevoMotor()
	if err := m.Cargar(a); err != nil {
		t.Fatalf("el primero tiene que entrar: %v", err)
	}
	err := m.Cargar(b)
	if !errors.Is(err, ErrExportaColisionado) {
		t.Fatalf("dos duenos del mismo predicado comun tiene que rechazarse con "+
			"ErrExportaColisionado, y dio: %v", err)
	}
	// El mensaje tiene que decir los DOS paquetes y como salir del lio: si solo
	// dice "colision", el autor del segundo paquete no sabe con quien choca.
	for _, quiero := range []string{"urn:demo:alfa", "urn:demo:beta", "urn:demo:beta.en_ambito"} {
		if !strings.Contains(err.Error(), quiero) {
			t.Errorf("el error tiene que mencionar %q, y dijo: %v", quiero, err)
		}
	}
}

// Cargar el MISMO paquete dos veces no es una colision consigo mismo. Pasa al
// recargar el corpus, y si diera error el producto no arrancaria dos veces.
func TestElMismoPaqueteDosVecesNoColisionaConsigoMismo(t *testing.T) {
	a := programaAlfa()
	a.Exporta = []string{"en_ambito"}
	m := NuevoMotor()
	if err := m.Cargar(a); err != nil {
		t.Fatal(err)
	}
	if err := m.Cargar(a); err != nil {
		t.Fatalf("recargar el mismo paquete no puede ser una colision: %v", err)
	}
}

// Un hecho con el nombre de un predicado que un paquete define como propio no
// alimenta nada, y eso tiene que ser un error y no un silencio.
func TestUnHechoQueChocaConUnPredicadoPropioSeDenuncia(t *testing.T) {
	m := NuevoMotor()
	cargar(t, m, programaAlfa()) // define en_ambito como propio
	m.Afirmar(H("en_ambito", "ayto"))

	_, err := m.Evaluar()
	if !errors.Is(err, ErrHechoQueNadieUsa) {
		t.Fatalf("afirmar en_ambito cuando alfa lo define como propio tiene que denunciarse "+
			"con ErrHechoQueNadieUsa, y dio: %v", err)
	}
	if !strings.Contains(err.Error(), "urn:demo:alfa") {
		t.Errorf("el error tiene que decir QUE paquete lo define, y dijo: %v", err)
	}

	// Control positivo: los hechos que el paquete consume de verdad no se
	// denuncian. Sin esto, un guardia que gritara siempre pasaria el test de
	// arriba y bloquearia el producto entero.
	m2 := NuevoMotor()
	cargar(t, m2, programaAlfa())
	m2.Afirmar(H("sector", "ayto", "publico"))
	if _, err := m2.Evaluar(); err != nil {
		t.Fatalf("un hecho que el paquete consume no puede denunciarse: %v", err)
	}
}
