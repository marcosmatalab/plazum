package incidente

import (
	"errors"
	"testing"
	"time"
)

func ins(t *testing.T, s string) time.Time {
	t.Helper()
	x, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("instante %q: %v", s, err)
	}
	return x
}

// abierto: el incidente de referencia. Ocurrio de madrugada, se supo por la
// manana, que son nueve horas de distancia y todo el asunto del art. 33.
func abierto(t *testing.T) *Incidente {
	t.Helper()
	i, err := Abrir("incidente/2026-014",
		ins(t, "2026-03-01T04:00:00Z"), ins(t, "2026-03-02T09:00:00Z"), "soc")
	if err != nil {
		t.Fatal(err)
	}
	return i
}

// ---------------------------------------------------------------------------
// El valor cero, en sus DOS formas (invariante 8)
// ---------------------------------------------------------------------------

// Las dos formas de la nada no son la misma y la peligrosa es la que sale por
// olvidarse. Aqui las dos tienen que negarse a dar un instante, porque el
// instante que darian es el cero, y el cero con cara de dato es un plazo
// vencido en el ano 1.
func TestElValorCeroDeUnIncidenteNoDaNingunInstante(t *testing.T) {
	casos := map[string]*Incidente{
		"nil":            nil,
		"vacio presente": {},
		"con id y sin apertura": func() *Incidente {
			// La tercera forma, y es la que se cuela: parece construido.
			return &Incidente{id: "incidente/2026-099"}
		}(),
	}
	for nombre, i := range casos {
		t.Run(nombre, func(t *testing.T) {
			if i.Abierto() {
				t.Fatal("se declara abierto sin apertura")
			}
			if i == nil {
				return // los demas metodos son sobre puntero no nulo a proposito
			}
			if _, ok := i.PrimerConocimiento(); ok {
				t.Error("da primer conocimiento sin apertura: de ahi cuentan 72 horas")
			}
			if _, ok := i.Ocurrio(); ok {
				t.Error("da fecha de ocurrencia sin apertura")
			}
			if _, err := i.Hechos("constancia"); !errors.Is(err, ErrSinApertura) {
				t.Errorf("Hechos sin apertura devuelve %v, y tiene que decir que no hay "+
					"apertura en vez de un mapa con el instante cero", err)
			}
		})
	}
}

// El nombre del hecho vacio es el valor cero permisivo de esta frontera: el
// instante se escribiria bajo la clave "" y ningun reloj la busca, asi que el
// plazo no arrancaria y nadie sabria por que.
func TestSinNombreDelHechoNoSeEscribeNada(t *testing.T) {
	i := abierto(t)
	if _, err := i.Hechos(""); !errors.Is(err, ErrCampoQueFalta) {
		t.Fatalf("con el disparador vacio devuelve %v", err)
	}
}

func TestUnIncidenteSinIdentidadNoSeAbre(t *testing.T) {
	if _, err := Abrir("", ins(t, "2026-03-01T04:00:00Z"),
		ins(t, "2026-03-02T09:00:00Z"), "soc"); !errors.Is(err, ErrSinIdentidad) {
		t.Fatalf("un incidente sin id se abre: %v", err)
	}
}

// ---------------------------------------------------------------------------
// El reloj que miente
// ---------------------------------------------------------------------------

// Nadie tiene constancia de algo antes de que ocurra. Sin esta guarda, un
// sistema con la hora mal (o un import de CSV con las columnas cambiadas) mete
// un incidente cuyo plazo vence antes de arrancar.
func TestNadieTieneConstanciaAntesDeQueElHechoOcurra(t *testing.T) {
	_, err := Abrir("incidente/2026-014",
		ins(t, "2026-03-02T09:00:00Z"), // ocurrio
		ins(t, "2026-03-01T04:00:00Z"), // se supo ANTES
		"import-csv")
	if !errors.Is(err, ErrRegistroAntesDelHecho) {
		t.Fatalf("acepta constancia anterior al hecho: %v", err)
	}
	// Y el mismo instante SI vale: saberlo en el momento en que pasa es lo
	// normal, no un error.
	if _, err := Abrir("incidente/2026-015",
		ins(t, "2026-03-02T09:00:00Z"), ins(t, "2026-03-02T09:00:00Z"), "soc"); err != nil {
		t.Fatalf("rechaza el caso normal (se supo al ocurrir): %v", err)
	}
}

func TestNingunSucesoSeRegistraSinSusDosEjes(t *testing.T) {
	i := abierto(t)
	for _, s := range []Suceso{
		{Tipo: Clasificacion, Clase: "grave", InstanteRegistro: ins(t, "2026-03-02T10:00:00Z")},
		{Tipo: Clasificacion, Clase: "grave", InstanteHecho: ins(t, "2026-03-02T10:00:00Z")},
	} {
		if err := i.Registrar(s); !errors.Is(err, ErrInstanteCero) {
			t.Errorf("acepta un suceso con un eje a cero: %v", err)
		}
	}
}

// ---------------------------------------------------------------------------
// El flujo es inmutable: se anade, no se corrige encima
// ---------------------------------------------------------------------------

func TestUnIncidenteNaceUnaSolaVez(t *testing.T) {
	i := abierto(t)
	err := i.Registrar(Suceso{Tipo: Apertura,
		InstanteHecho: ins(t, "2026-03-05T04:00:00Z"), InstanteRegistro: ins(t, "2026-03-05T04:00:00Z")})
	if !errors.Is(err, ErrSegundaApertura) {
		t.Fatalf("admite una segunda apertura, que moveria el arranque de todos sus "+
			"plazos sin dejar rastro: %v", err)
	}
}

func TestNadaSeRegistraEnUnIncidenteQueNoSeHaAbierto(t *testing.T) {
	i := &Incidente{id: "incidente/2026-099"}
	err := i.Registrar(Suceso{Tipo: Clasificacion, Clase: "grave",
		InstanteHecho: ins(t, "2026-03-02T10:00:00Z"), InstanteRegistro: ins(t, "2026-03-02T10:00:00Z")})
	if !errors.Is(err, ErrSinApertura) {
		t.Fatalf("clasifica un incidente sin apertura: %v", err)
	}
}

// RECLASIFICAR ES UN HECHO NUEVO. La clasificacion anterior sigue ahi, con su
// instante, porque la linea de tiempo que ve un auditor tiene que poder decir
// que se creyo y desde cuando.
func TestReclasificarNoBorraLaClasificacionAnterior(t *testing.T) {
	i := abierto(t)
	primera := ins(t, "2026-03-02T10:00:00Z")
	segunda := ins(t, "2026-03-04T18:00:00Z")
	for _, c := range []struct {
		clase string
		en    time.Time
	}{{"incidente_general", primera}, {"incidente_fallecimiento", segunda}} {
		if err := i.Registrar(Suceso{Tipo: Clasificacion, Clase: c.clase,
			InstanteHecho: c.en, InstanteRegistro: c.en, Fuente: "responsable"}); err != nil {
			t.Fatal(err)
		}
	}

	// Manda la mas reciente...
	clase, empate, ok := i.ClaseEn(ins(t, "2026-03-10T00:00:00Z"))
	if !ok || empate || clase != "incidente_fallecimiento" {
		t.Fatalf("clase vigente: %q (empate %v, ok %v)", clase, empate, ok)
	}
	// ...y la anterior sigue siendo consultable en su momento, que es lo que
	// distingue un flujo de hechos de un campo que se sobreescribe.
	clase, _, ok = i.ClaseEn(ins(t, "2026-03-03T00:00:00Z"))
	if !ok || clase != "incidente_general" {
		t.Fatalf("el 3 de marzo la clase vigente era la primera y sale %q (ok %v)", clase, ok)
	}
	if n := len(i.Sucesos()); n != 3 {
		t.Fatalf("el flujo tiene %d sucesos y tenia que tener 3 (apertura y dos "+
			"clasificaciones): reclasificar ha borrado algo", n)
	}
}

// El empate no se resuelve a cara o cruz. Dos clasificaciones distintas en el
// mismo instante son una contradiccion del dato, y elegir una daria un plazo
// distinto segun el orden del recorrido. Misma decision que ventana.claseVigente.
func TestDosClasificacionesEnElMismoInstanteSeDicenEnVozAlta(t *testing.T) {
	i := abierto(t)
	en := ins(t, "2026-03-02T10:00:00Z")
	for _, c := range []string{"incidente_general", "incidente_fallecimiento"} {
		if err := i.Registrar(Suceso{Tipo: Clasificacion, Clase: c,
			InstanteHecho: en, InstanteRegistro: en}); err != nil {
			t.Fatal(err)
		}
	}
	if _, empate, ok := i.ClaseEn(ins(t, "2026-03-10T00:00:00Z")); !ok || !empate {
		t.Fatalf("el empate no se declara (empate %v, ok %v)", empate, ok)
	}
}

// Sucesos devuelve una COPIA. Devolver el slice interno seria dar una via de
// edicion a un objeto cuya propiedad entera es que no se edita.
func TestSucesosNoDaAccesoAlFlujoInterno(t *testing.T) {
	i := abierto(t)
	s := i.Sucesos()
	s[0].InstanteRegistro = ins(t, "2026-07-01T00:00:00Z")
	pc, _ := i.PrimerConocimiento()
	if !pc.Equal(ins(t, "2026-03-02T09:00:00Z")) {
		t.Fatalf("tocando el resultado de Sucesos se movio el primer conocimiento a %s: "+
			"eso mueve el plazo de 72 horas cuatro meses", pc.Format(time.RFC3339))
	}
}

// ---------------------------------------------------------------------------
// Cada tipo trae lo suyo, y no lo ajeno: LAS DOS DIRECCIONES
// ---------------------------------------------------------------------------

// La direccion que falta siempre es la que muerde. Un campo que falta deja el
// suceso inservible; un campo ajeno (una clase en una notificacion) es un dato
// escrito con cuidado que no lee nadie, que es la clase del campo huerfano.
func TestCadaTipoTraeLoSuyoYNoLoAjeno(t *testing.T) {
	en := ins(t, "2026-03-02T10:00:00Z")
	base := func(tp Tipo) Suceso {
		return Suceso{Tipo: tp, InstanteHecho: en, InstanteRegistro: en}
	}
	falta := []Suceso{
		base(Clasificacion), // sin clase
		base(Notificacion),  // sin hito
	}
	ajeno := []Suceso{
		{Tipo: Apertura, Clase: "grave", InstanteHecho: en, InstanteRegistro: en},
		{Tipo: Notificacion, Hito: "alerta", Clase: "grave", InstanteHecho: en, InstanteRegistro: en},
		{Tipo: Clasificacion, Clase: "grave", Hito: "alerta", InstanteHecho: en, InstanteRegistro: en},
		{Tipo: Cierre, Hito: "alerta", InstanteHecho: en, InstanteRegistro: en},
	}
	for _, s := range falta {
		if err := camposDe(s); !errors.Is(err, ErrCampoQueFalta) {
			t.Errorf("un %s sin su campo obligatorio pasa: %v", s.Tipo, err)
		}
	}
	for _, s := range ajeno {
		if err := camposDe(s); !errors.Is(err, ErrCampoAjeno) {
			t.Errorf("un %s con un campo ajeno pasa: %v", s.Tipo, err)
		}
	}
}

func TestUnTipoFueraDelVocabularioNoEntraYSuMensajeNoRevienta(t *testing.T) {
	i := abierto(t)
	en := ins(t, "2026-03-02T10:00:00Z")
	err := i.Registrar(Suceso{Tipo: Tipo(99), InstanteHecho: en, InstanteRegistro: en})
	if !errors.Is(err, ErrTipoDesconocido) {
		t.Fatalf("acepta un tipo fuera del vocabulario: %v", err)
	}
	// Y el String de un tipo invalido no entra en panico, que es lo que pasaria
	// justo al formatear el error que explicaba el problema.
	if s := Tipo(99).String(); s == "" {
		t.Fatal("el String de un tipo invalido no dice nada")
	}
}

// ---------------------------------------------------------------------------
// Lo que consta y lo que no
// ---------------------------------------------------------------------------

// "Consta" y "se hizo" no son lo mismo, y el metodo se llama como se llama por
// eso: que no conste una notificacion NO dice que no se hiciera, dice que en
// las respuestas del cliente no aparece. Acusar en falso es el unico error que
// un producto de cumplimiento no puede cometer ni una vez.
func TestNotificadoDiceLoQueCONSTAYNoLoQueSeHizo(t *testing.T) {
	i := abierto(t)
	if _, ok := i.Notificado("alerta_temprana"); ok {
		t.Fatal("dice que consta una notificacion que nadie registro")
	}
	en := ins(t, "2026-03-03T08:00:00Z")
	if err := i.Registrar(Suceso{Tipo: Notificacion, Hito: "alerta_temprana",
		InstanteHecho: en, InstanteRegistro: en, Fuente: "csirt"}); err != nil {
		t.Fatal(err)
	}
	cuando, ok := i.Notificado("alerta_temprana")
	if !ok || !cuando.Equal(en) {
		t.Fatalf("la notificacion registrada no consta: %v %v", cuando, ok)
	}
	if _, ok := i.Notificado("informe_final"); ok {
		t.Fatal("consta un hito distinto del registrado: el emparejamiento es por hito")
	}
}

// ---------------------------------------------------------------------------
// La junta de D-18: el paquete pone el nombre, el incidente la instancia
// ---------------------------------------------------------------------------

func TestLosHechosLlevanElInstanteDeCadaCosaBajoElNombreQuePideElPaquete(t *testing.T) {
	i := abierto(t)
	clasificado := ins(t, "2026-03-02T10:00:00Z")
	remitido := ins(t, "2026-03-03T08:00:00Z")
	if err := i.Registrar(Suceso{Tipo: Clasificacion, Clase: "incidente_general",
		InstanteHecho: clasificado, InstanteRegistro: clasificado}); err != nil {
		t.Fatal(err)
	}
	if err := i.Registrar(Suceso{Tipo: Notificacion, Hito: "alerta_temprana",
		InstanteHecho: remitido, InstanteRegistro: remitido}); err != nil {
		t.Fatal(err)
	}

	h, err := i.Hechos("constancia_incidente_significativo")
	if err != nil {
		t.Fatal(err)
	}
	quiere := map[string]time.Time{
		// EL ARRANQUE ES EL PRIMER CONOCIMIENTO, no cuando ocurrio: el art. 33
		// cuenta "desde que haya tenido constancia".
		"constancia_incidente_significativo": ins(t, "2026-03-02T09:00:00Z"),
		"incidente_general":                  clasificado,
		"alerta_temprana.cumplido":           remitido,
	}
	for k, v := range quiere {
		got, ok := h[k]
		if !ok {
			t.Errorf("falta el hecho %q, que es el que el paquete busca", k)
			continue
		}
		if !got.Equal(v) {
			t.Errorf("el hecho %q vale %s y tenia que valer %s", k,
				got.Format(time.RFC3339), v.Format(time.RFC3339))
		}
	}
	if len(h) != len(quiere) {
		t.Errorf("sobran hechos: %v", h)
	}
}

// DOS INCIDENTES, DOS JUEGOS DE HECHOS. Es la medicion de por_objeto_test.go al
// derecho: con un solo mapa por organizacion, el segundo incidente pisaba al
// primero y el que desaparecia vencia nueve dias antes.
func TestDosIncidentesNoCompartenNiUnHecho(t *testing.T) {
	uno := abierto(t)
	dos, err := Abrir("incidente/2026-015",
		ins(t, "2026-03-01T04:00:00Z"), ins(t, "2026-03-11T17:00:00Z"), "soc")
	if err != nil {
		t.Fatal(err)
	}
	const disp = "constancia"
	hUno, err := uno.Hechos(disp)
	if err != nil {
		t.Fatal(err)
	}
	hDos, err := dos.Hechos(disp)
	if err != nil {
		t.Fatal(err)
	}
	if hUno[disp].Equal(hDos[disp]) {
		t.Fatal("los dos incidentes dan el mismo arranque: se estan pisando")
	}
	// Y ninguno puede tocar al otro: son mapas distintos, no vistas del mismo.
	hUno[disp] = ins(t, "2027-01-01T00:00:00Z")
	if hDos[disp].Equal(hUno[disp]) {
		t.Fatal("tocar los hechos de un incidente mueve los del otro")
	}
	pc, _ := uno.PrimerConocimiento()
	if pc.Equal(hUno[disp]) {
		t.Fatal("tocar el mapa devuelto mueve el incidente")
	}
}
