package main

import (
	"encoding/json"
	"testing"
)

// Los tests no salen a la red: prueban la logica, que es donde estuvo el fallo.
// La consulta a TMview es un POST con user-agent y ya esta; lo que se equivoco
// no fue la llamada, fue no mirar en la tercera direccion.

// La lente que nos mordio: DUTIQ contiene UTIQ. Si esta lista no genera "utiq",
// la herramienta no sirve para lo que se construyo.
func TestLasSubcadenasIncluyenLaQueNosMordio(t *testing.T) {
	subs := subcadenas("dutiq")
	var hayUtiq, hayDuti bool
	for _, s := range subs {
		switch s {
		case "utiq":
			hayUtiq = true
		case "duti":
			hayDuti = true
		}
		if s == "dutiq" {
			t.Fatal("el candidato entero no es subcadena propia: eso ya lo cubre la lente 1")
		}
		if len(s) < minSubcadena {
			t.Fatalf("subcadena demasiado corta, solo genera ruido: %q", s)
		}
	}
	if !hayUtiq {
		t.Fatal("HALLAZGO: 'utiq' no sale de 'dutiq'. Es literalmente el caso que motivo esta " +
			"herramienta: la marca ajena va DENTRO del candidato")
	}
	if !hayDuti {
		t.Fatal("faltan subcadenas: 'duti' tambien es una subcadena propia de 'dutiq'")
	}
}

// De larga a corta: una marca de cuatro letras dentro del candidato preocupa
// mas que una de tres.
func TestLasSubcadenasVanDeLargaACorta(t *testing.T) {
	subs := subcadenas("dutiq")
	for i := 1; i < len(subs); i++ {
		if len(subs[i]) > len(subs[i-1]) {
			t.Fatalf("orden roto en %d: %q (%d) despues de %q (%d)",
				i, subs[i], len(subs[i]), subs[i-1], len(subs[i-1]))
		}
	}
}

func TestUnaMarcaCaducadaNoCuenta(t *testing.T) {
	casos := []struct {
		estado string
		quiere bool
	}{
		{"Registered", true},
		{"Filed", true},
		{"Opposition Pending", true},
		{"Expired", false},
		{"Ended", false},
		{"Withdrawn", false},
	}
	for _, c := range casos {
		if got := (marca{Estado: c.estado}).vigente(); got != c.quiere {
			t.Errorf("estado %q: vigente=%v, se esperaba %v", c.estado, got, c.quiere)
		}
	}
}

func TestSoloCuentanLasClasesQueImportan(t *testing.T) {
	utiq := marca{Clases: []int{9, 25, 35, 38, 42}}
	if !utiq.tocaClases([]int{9, 42}) {
		t.Fatal("UTIQ toca las clases 9 y 42: tiene que contar")
	}
	if !utiq.tocaClases(nil) {
		t.Fatal("sin filtro de clases, cuentan todas")
	}
	ropa := marca{Clases: []int{25}}
	if ropa.tocaClases([]int{9, 42}) {
		t.Fatal("una marca solo de ropa no colisiona con software")
	}
}

// El semaforo, que es donde estaba el fallo de verdad.
//
// La primera version pintaba ROJO en cuanto aparecia CUALQUIER marca dentro
// del candidato, de la longitud que fuera. Contra la base entera de la Union y
// con subcadenas de tres letras, eso ocurre siempre: no hay candidato que
// salga de otro color, porque VEN, ENC, NCI, CIA, REC, ECE, CEP y EPT estan
// todas registradas en la clase 9. Un semaforo que siempre dice ROJO no
// distingue "UTIQ dentro de DUTIQ" de "CIA dentro de VENCIA", que es
// exactamente la distincion para la que se construyo la herramienta.
//
// Lo que pesa es la COBERTURA: cuanto del signo corto ocupa la coincidencia.
// Los casos llevan nombres y numeros reales porque son los que hay que poder
// reproducir manana.
func TestElSemaforoSeparaLaMarcaAjenaDelAcronimo(t *testing.T) {
	casos := []struct {
		nombre string
		h      Hallazgo
		quiere string
	}{
		{
			// El caso que costo la marca: 4 letras de 5, el 80%.
			nombre: "UTIQ dentro de DUTIQ",
			h: Hallazgo{Candidato: "dutiq", Contenidas: []marca{
				{Nombre: "Utiq", Numero: "018838934", Coincidente: "utiq"}}},
			quiere: "ROJO",
		},
		{
			// Al reves: el candidato dentro de una marca ajena, 6 de 7.
			nombre: "VENCIA dentro de AVENCIA",
			h: Hallazgo{Candidato: "vencia", Contenedoras: []marca{
				{Nombre: "AVENCIA", Numero: "019216770", Coincidente: "vencia"}}},
			quiere: "ROJO",
		},
		{
			// 7 letras de 9, el 78%, y en las dos clases que importan.
			nombre: "PRECEPT dentro de PRECEPTUM",
			h: Hallazgo{Candidato: "preceptum", Contenidas: []marca{
				{Nombre: "PRECEPT", Numero: "018314665", Coincidente: "precept"}}},
			quiere: "ROJO",
		},
		{
			nombre: "colision exacta",
			h: Hallazgo{Candidato: "dutiq", Colisiones: []marca{
				{Nombre: "dutiq", Coincidente: "dutiq"}}},
			quiere: "ROJO",
		},
		{
			// EL CONTROL NEGATIVO. Sin esto el test no prueba nada: un semaforo
			// que solo sabe decir ROJO pasaria todos los casos de arriba.
			// Son marcas reales, vivas y en clase 9, y ninguna se parece a
			// VENCIA porque ninguna ocupa medio nombre.
			nombre: "solo acronimos de tres letras",
			h: Hallazgo{Candidato: "vencia", Contenidas: []marca{
				{Nombre: "VEN", Numero: "009158734", Coincidente: "ven"},
				{Nombre: "ENC", Numero: "018656168", Coincidente: "enc"},
				{Nombre: "NCI", Numero: "018756795", Coincidente: "nci"},
				{Nombre: "CIA", Numero: "006139208", Coincidente: "cia"}}},
			quiere: "sin hallazgos",
		},
		{
			// Cuatro letras, pero solo el 44% de un nombre de nueve.
			nombre: "CEPT dentro de PRECEPTUM",
			h: Hallazgo{Candidato: "preceptum", Contenidas: []marca{
				{Nombre: "CEPT", Numero: "016947228", Coincidente: "cept"}}},
			quiere: "sin hallazgos",
		},
		{
			// La franja de en medio existe y hay que verla: medio nombre.
			nombre: "cuatro letras que son la mitad del candidato",
			h: Hallazgo{Candidato: "vencido", Contenidas: []marca{
				{Nombre: "VENC", Coincidente: "venc"}}},
			quiere: "AMBAR",
		},
	}
	for _, c := range casos {
		h := c.h
		clasificar(&h)
		if got := h.Riesgo(); got != c.quiere {
			t.Errorf("%s: el semaforo dio %q y se esperaba %q", c.nombre, got, c.quiere)
			for _, ms := range [][]marca{h.Colisiones, h.Contenedoras, h.Contenidas} {
				for _, m := range ms {
					t.Logf("   %s (%s) cobertura %.2f nivel %s",
						m.Nombre, m.Coincidente, m.Cobertura, m.Nivel)
				}
			}
		}
	}
}

// Que el ruido se CUENTE y no se tire. Un umbral que descarta en silencio hace
// que "sin hallazgos" se lea como "se ha mirado todo", y no es lo mismo.
func TestLoQueQuedaBajoElUmbralSigueEnLosDatos(t *testing.T) {
	h := Hallazgo{Candidato: "vencia", Contenidas: []marca{
		{Nombre: "CIA", Coincidente: "cia"},
		{Nombre: "ENCI", Coincidente: "enci"}}}
	clasificar(&h)
	if len(h.Contenidas) != 2 {
		t.Fatalf("clasificar ha borrado hallazgos: quedan %d de 2", len(h.Contenidas))
	}
	var ruido, pintado int
	for _, m := range h.Contenidas {
		if m.Nivel == nivelRuido {
			ruido++
		} else {
			pintado++
		}
	}
	if ruido != 1 || pintado != 1 {
		t.Fatalf("se esperaba un hallazgo por encima del umbral y uno por debajo, y hay %d y %d",
			pintado, ruido)
	}
}

// El parseo, contra una respuesta real de TMview recortada. Si TMview cambia
// los nombres de campo, esto se pone rojo y se sabe por que.
func TestParsearUnaRespuestaRealDeTMview(t *testing.T) {
	cruda := []byte(`{"tradeMarks":[{
		"applicationNumber":"018838934","registrationNumber":"018838934",
		"tradeMarkStatus":"Registered","niceClass":[9,25,35,38,42],
		"applicantName":["Utiq"],"tmName":"Utiq","tmOffice":"EM",
		"tmOfficeURL":"https://euipo.europa.eu/trademark/data/EM500000018838934",
		"applicationDate":"2023-02-21T12:00:00.000Z","tradeMarkType":"Word",
		"registrationDate":"2023-06-13T12:00:00.000Z",
		"expirationDate":"2033-02-21T12:00:00.000Z"}]}`)
	ms, err := parsear(cruda)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 1 {
		t.Fatalf("una marca, y salieron %d", len(ms))
	}
	m := ms[0]
	if m.Numero != "018838934" || m.Nombre != "Utiq" || m.Tipo != "Word" {
		t.Fatalf("mal parseada: %+v", m)
	}
	if m.Estado != "Registered" || !m.vigente() {
		t.Fatalf("una registrada tiene que contar: %+v", m)
	}
	if m.Registrada != "2023-06-13" || m.Expira != "2033-02-21" {
		t.Fatalf("las fechas se recortan a la fecha sola: %+v", m)
	}
	if !m.tocaClases([]int{9, 42}) {
		t.Fatal("018838934 esta en 9 y 42")
	}
}

func TestParsearAguantaBasura(t *testing.T) {
	for _, b := range [][]byte{
		[]byte(``), []byte(`{`), []byte(`{"tradeMarks":null}`),
		[]byte(`{"tradeMarks":[{"niceClass":"no es una lista"}]}`),
	} {
		// No puede reventar: lo unico que se exige es que no haga panic.
		_, _ = parsear(b)
	}
}

func TestParsearClasesRechazaLoQueNoExiste(t *testing.T) {
	if _, err := parsearClases("9,42"); err != nil {
		t.Fatalf("9 y 42 son validas: %v", err)
	}
	if _, err := parsearClases("46"); err == nil {
		t.Fatal("la clase 46 no existe: Niza va de 1 a 45")
	}
	if _, err := parsearClases("nueve"); err == nil {
		t.Fatal("una clase que no es un numero tiene que rechazarse")
	}
	cs, err := parsearClases("")
	if err != nil || cs != nil {
		t.Fatal("sin clases significa todas, no error")
	}
}

// El JSON de salida tiene que ser consumible por otra herramienta.
func TestLaSalidaJSONEsUsable(t *testing.T) {
	h := Hallazgo{Candidato: "dutiq", Contenidas: []marca{{
		Numero: "018838934", Nombre: "Utiq", Estado: "Registered",
		Clases: []int{9, 42}, Coincidente: "utiq"}}}
	b, err := json.Marshal([]Hallazgo{h})
	if err != nil {
		t.Fatal(err)
	}
	var vuelta []Hallazgo
	if err := json.Unmarshal(b, &vuelta); err != nil {
		t.Fatal(err)
	}
	if len(vuelta) != 1 || len(vuelta[0].Contenidas) != 1 ||
		vuelta[0].Contenidas[0].Coincidente != "utiq" {
		t.Fatalf("el JSON no sobrevive la ida y vuelta: %s", b)
	}
}
