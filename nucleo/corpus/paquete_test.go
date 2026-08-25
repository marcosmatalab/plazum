package corpus

import (
	"os"
	"strings"
	"testing"
)

// paquete minimo valido, para partir de el en cada caso.
//
// Los identificadores son sinteticos, del espacio urn:demo: y demo.: el nucleo
// es autonomo y no conoce el directorio del corpus, asi que ningun test de
// nucleo/ puede depender de un URN real de paquetes/. Lo que si es real es la
// FORMA: un paquete de clase Transcrito, amparado en el art. 13 TRLPI, con su
// fuente enlazada, una obligacion procedimental de auditoria periodica y su
// plantilla de entregable. Esa forma es la que validan los tests de abajo.
func base() *Paquete {
	return &Paquete{
		URN: "urn:demo:transcrita", Version: "1.0.0", Clase: Transcrito,
		Licencia: "art. 13 TRLPI", Fuente: "https://www.boe.es/eli/es/rd/2022/05/03/311",
		Consolidado: true,
		Entidades: []TipoEntidad{{
			Nombre: "sistema", Descripcion: "sistema de informacion en el ambito de la norma",
			Atributos: []Atributo{{
				Nombre: "categoria", Tipo: Enumerado,
				Valores: []string{"BASICA", "MEDIA", "ALTA"}, Escala: "demo.categoria",
				Obligado: true, Cita: "RD 311/2022 art. 40 y Anexo I",
			}},
		}},
		Preguntas: []Pregunta{{
			ID: "demo.q.categoria", Texto: "Que nivel alcanza cada dimension?",
			Cita: "RD 311/2022 Anexo I", Entidad: "sistema", Atributo: "categoria",
			Desbloquea: []string{"demo.auditoria_bienal"},
		}},
		Obligaciones: []Obligacion{{
			ID: "demo.auditoria_bienal", Articulo: "31", ClaseE2E: "procedimental",
			TextoLegal: "Los sistemas de informacion... seran objeto de una auditoria regular ordinaria, al menos cada dos anos.",
			Cita:       "RD 311/2022 art. 31", Vigencia: Vigencia{Desde: "2022-05-05"},
			Entregable: "demo.informe_auditoria", Preguntas: []string{"demo.q.categoria"},
			Recursos: []TipoRecurso{"Sistema"},
		}},
		Plantillas: []Plantilla{{
			ID: "demo.informe_auditoria", Titulo: "Informe de auditoria bienal",
			Cita: "RD 311/2022 art. 31 y guia CCN-STIC 802",
			Campos: []CampoPlantilla{
				{Nombre: "categoria", Origen: "entidad:sistema.categoria"},
				{Nombre: "fecha_anterior", Origen: "obligacion:demo.auditoria_bienal.ultimo_hito"},
			},
		}},
	}
}

func TestPaqueteValidoNoDaErrores(t *testing.T) {
	if errs := base().Validar(); len(errs) != 0 {
		t.Fatalf("paquete base deberia validar, errores: %v", errs)
	}
}

// El test que hace cumplir la frontera legal. Es el que rompe el build si alguien
// mete el texto de un catalogo de pago (ISO, PCI DSS, SOC 2, TISAX, CIS) en un
// paquete: de esos solo se puede distribuir identificador y titulo corto.
func TestReferencialRechazaTextoNormativo(t *testing.T) {
	p := base()
	p.URN, p.Clase = "urn:demo:referencial", Referencial
	p.Obligaciones[0].TextoLegal = strings.Repeat("x", LimiteTextoReferencial+1)
	errs := p.Validar()
	if len(errs) == 0 {
		t.Fatal("un paquete referencial con texto normativo largo debe ser rechazado")
	}
	if !strings.Contains(errs[0].Error(), "referencial") {
		t.Fatalf("el error debe explicar por que: %v", errs[0])
	}
	// y el identificador con titulo corto si pasa
	p.Obligaciones[0].TextoLegal = "C.5.1 Titulo corto del control referenciado"
	if errs := p.Validar(); len(errs) != 0 {
		t.Fatalf("identificador y titulo corto deben permitirse: %v", errs)
	}
}

func TestDelegadoNoDistribuyeNadaYExigeHerramienta(t *testing.T) {
	p := base()
	p.URN, p.Clase = "urn:demo:delegada", Delegado
	errs := p.Validar()
	if len(errs) != 2 { // lleva texto y no declara herramienta
		t.Fatalf("esperaba 2 errores, %d: %v", len(errs), errs)
	}
	p.Obligaciones[0].TextoLegal = ""
	p.Obligaciones[0].Delegado = "openscap:xccdf_org.demo_benchmark"
	if errs := p.Validar(); len(errs) != 0 {
		t.Fatalf("delegado bien formado debe validar: %v", errs)
	}
}

func TestFuenteObligatoria(t *testing.T) {
	p := base()
	p.Fuente = ""
	errs := p.Validar()
	if len(errs) == 0 || !strings.Contains(errs[0].Error(), "fuente") {
		t.Fatalf("la fuente con enlace es condicion de reutilizacion del BOE: %v", errs)
	}
}

func TestEntregableDeclaradoDebeExistir(t *testing.T) {
	p := base()
	p.Obligaciones[0].Entregable = "no_existe"
	errs := p.Validar()
	if len(errs) == 0 {
		t.Fatal("una obligacion no puede apuntar a una plantilla que no esta")
	}
}

func TestCampoDePlantillaSinOrigenSeRechaza(t *testing.T) {
	p := base()
	p.Plantillas[0].Campos[0].Origen = ""
	errs := p.Validar()
	if len(errs) == 0 || !strings.Contains(errs[0].Error(), "origen") {
		t.Fatalf("un hueco sin trazabilidad no es un entregable: %v", errs)
	}
}

func TestPreguntaQueNoDesbloqueaNadaSeRechaza(t *testing.T) {
	p := base()
	p.Preguntas[0].Desbloquea = nil
	errs := p.Validar()
	if len(errs) == 0 {
		t.Fatal("no se le hacen preguntas al usuario que no sirvan para nada")
	}
}

func TestPreguntaApuntaAAtributoInexistente(t *testing.T) {
	p := base()
	p.Preguntas[0].Atributo = "inventado"
	if errs := p.Validar(); len(errs) == 0 {
		t.Fatal("una pregunta debe rellenar un atributo declarado")
	}
}

// --- lo derivado ---

func TestEsquemaUIUneAtributosYDiceQuienLosPide(t *testing.T) {
	a := base()
	b := base()
	b.URN = "urn:demo:segunda"
	campos := EsquemaUI([]*Paquete{a, b})
	if len(campos) != 1 {
		t.Fatalf("el mismo atributo pedido por dos normas se pregunta una vez, %d", len(campos))
	}
	if len(campos[0].Paquetes) != 2 {
		t.Fatalf("debe decir que dos paquetes lo piden: %v", campos[0].Paquetes)
	}
}

func TestEntrevistaOrdenaPorObligacionesDesbloqueadas(t *testing.T) {
	p := base()
	p.Obligaciones = append(p.Obligaciones, Obligacion{
		ID: "demo.declaracion_conformidad", Cita: "RD 311/2022 art. 38", ClaseE2E: "documental",
		Vigencia: Vigencia{Desde: "2022-05-05"},
	})
	p.Preguntas = append(p.Preguntas, Pregunta{
		ID: "demo.q.ambito", Texto: "Es sector publico?", Cita: "art. 2",
		Entidad: "sistema", Atributo: "categoria",
		Desbloquea: []string{"demo.auditoria_bienal", "demo.declaracion_conformidad"},
	})
	e := Entrevista([]*Paquete{p})
	if e[0].ID != "demo.q.ambito" {
		t.Fatalf("primero la que mas desbloquea, salio %s", e[0].ID)
	}
	if e[0].NDesbloquea != 2 {
		t.Fatalf("cuenta mal: %d", e[0].NDesbloquea)
	}
}

func TestTrazabilidadObligacionEntregableCampo(t *testing.T) {
	tr := Trazabilidad([]*Paquete{base()})
	if len(tr) != 2 {
		t.Fatalf("dos campos, dos trazas, salieron %d", len(tr))
	}
	if tr[0].Obligacion != "demo.auditoria_bienal" || tr[0].Origen == "" {
		t.Fatalf("traza incompleta: %+v", tr[0])
	}
}

func TestConectoresOrdenaPorObligacionesDesbloqueadas(t *testing.T) {
	p := base()
	p.Obligaciones = append(p.Obligaciones, Obligacion{
		ID: "demo.control_observable", Cita: "Anexo II", ClaseE2E: "observable", Vigencia: Vigencia{Desde: "2022-05-05"},
		Recursos: []TipoRecurso{"Identidad", "Sistema"},
	})
	c := Conectores([]*Paquete{p})
	if c[0].Recurso != "Sistema" || c[0].Obligaciones != 2 {
		t.Fatalf("Sistema lo piden 2 obligaciones y debe ir primero: %+v", c)
	}
}

func TestMedirNoRedondeaAFavor(t *testing.T) {
	p := base()
	p.Obligaciones = append(p.Obligaciones, Obligacion{
		ID: "demo.control_manual", Cita: "Anexo II", ClaseE2E: "procedimental", Vigencia: Vigencia{Desde: "2022-05-05"},
	})
	c := Medir(p)
	if c.Total != 2 || len(c.SinAutomatizar) != 1 {
		t.Fatalf("la que no tiene recurso ni herramienta cuenta como no automatizada: %+v", c)
	}
	if !strings.Contains(c.String(), "demo.control_manual") {
		t.Fatal("COBERTURA.md debe nombrar lo que falta, no solo contarlo")
	}
}

// --- carga desde disco: el linter corre al cargar, no al usar ---

func escribirPaquete(t *testing.T, raiz, nombre, contenido string) {
	t.Helper()
	d := raiz + "/" + nombre
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(d+"/paquete.json", []byte(contenido), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCargarRechazaPaqueteQueNoPasaElLinter(t *testing.T) {
	dir := t.TempDir()
	// paquete referencial con el texto de un control: exactamente lo que no se
	// puede distribuir de un catalogo de pago (ISO, PCI DSS, SOC 2, TISAX).
	// El nombre del directorio no puede contener "referencial", o la asercion
	// de abajo pasaria por el nombre en vez de por el mensaje del linter.
	escribirPaquete(t, dir, "catalogo-de-pago", `{
      "urn":"x@1","version":"1","clase":2,"fuente":"https://example.org/catalogo",
      "obligaciones":[{"id":"a","cita":"c","clase_e2e":"documental","texto_legal":"`+strings.Repeat("y", 200)+`"}]}`)
	if _, err := Cargar(dir); err == nil {
		t.Fatal("cargar debe rechazar un paquete que no pasa el linter")
	} else if !strings.Contains(err.Error(), "referencial") {
		t.Fatalf("el error debe decir por que: %v", err)
	}
}

func TestCargarIgnoraDirectoriosSinPaqueteJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(dir+"/vacio", 0o755); err != nil {
		t.Fatal(err)
	}
	ps, err := Cargar(dir)
	if err != nil || len(ps) != 0 {
		t.Fatalf("un directorio sin paquete.json no es un paquete: %v %d", err, len(ps))
	}
}

func TestCargarJSONInvalidoDaErrorConNombre(t *testing.T) {
	dir := t.TempDir()
	escribirPaquete(t, dir, "roto", `{no es json`)
	_, err := Cargar(dir)
	if err == nil || !strings.Contains(err.Error(), "roto") {
		t.Fatalf("el error debe nombrar el paquete: %v", err)
	}
}

func TestClaseSeImprimeLegible(t *testing.T) {
	for i, q := range []string{"importado", "transcrito", "referencial", "delegado", "propio"} {
		if Clase(i).String() != q {
			t.Errorf("clase %d: %s != %s", i, Clase(i), q)
		}
	}
	if Enumerado.String() != "enumerado" || Booleano.String() != "booleano" {
		t.Error("los tipos de atributo deben imprimirse legibles: van a la interfaz")
	}
}

// --- la extension e2e: controles negativos de las reglas nuevas ---

func TestClaseE2EInvalidaSeRechaza(t *testing.T) {
	p := base()
	p.Obligaciones[0].ClaseE2E = "magica"
	if errs := p.Validar(); len(errs) == 0 {
		t.Fatal("una clase e2e inventada debe rechazarse")
	}
}

func TestFacetaInvalidaSeRechaza(t *testing.T) {
	p := base()
	p.Obligaciones[0].Facetas = []string{"decorativa"}
	if errs := p.Validar(); len(errs) == 0 {
		t.Fatal("una faceta inventada debe rechazarse")
	}
}

func TestEscalonIncompletoSeRechaza(t *testing.T) {
	p := base()
	p.Obligaciones[0].Escalado = []Escalon{{Tras: "P3D"}}
	if errs := p.Validar(); len(errs) == 0 {
		t.Fatal("un escalon sin destinatario debe rechazarse")
	}
}

func TestTemporalidadSinTresDoradosSeRechaza(t *testing.T) {
	p := base()
	p.Obligaciones[0].Temporalidad = &Temporalidad{Primitiva: "periodica", Cadencia: "P24M",
		Regimen: RegimenSpec{Computo: "naturales", Cierre: "fin_de_dia"}}
	p.Dorados = []Dorado{{Caso: "solo uno", Obligacion: p.Obligaciones[0].ID,
		Esperado: EsperadoDorado{Vence: "2027-03-10T23:59:59Z"}, CitaDelEsperado: "art. 31"}}
	errs := p.Validar()
	if len(errs) == 0 || !strings.Contains(errs[0].Error(), "minimo 3") {
		t.Fatalf("un reloj con menos de 3 dorados debe rechazarse: %v", errs)
	}
}

func TestDoradoHuerfanoOSinCitaSeRechaza(t *testing.T) {
	p := base()
	p.Dorados = []Dorado{{Caso: "huerfano", Obligacion: "no.existe",
		Esperado: EsperadoDorado{Vence: "2027-01-01T00:00:00Z"}, CitaDelEsperado: "x"}}
	if errs := p.Validar(); len(errs) == 0 {
		t.Fatal("un dorado que apunta a una obligacion inexistente debe rechazarse")
	}
	p.Dorados = []Dorado{{Caso: "sin cita", Obligacion: p.Obligaciones[0].ID,
		Esperado: EsperadoDorado{Vence: "2027-01-01T00:00:00Z"}}}
	if errs := p.Validar(); len(errs) == 0 {
		t.Fatal("un dorado sin cita_del_esperado debe rechazarse")
	}
}

// Y el control negativo del ejecutor: un esperado equivocado DEBE fallar
// contra el motor. Sin esto, un verde no demuestra que se compara nada.
func TestElEjecutorDeDoradosDetectaUnEsperadoFalso(t *testing.T) {
	o := Obligacion{ID: "x", Cita: "c", ClaseE2E: "procedimental",
		Temporalidad: &Temporalidad{Primitiva: "periodica", Cadencia: "P24M",
			Regimen:    RegimenSpec{Computo: "naturales", Cierre: "fin_de_dia"},
			Disparador: map[string]string{"hecho": "ultima"}}}
	d := Dorado{Caso: "esperado falso", Obligacion: "x",
		Hechos:          map[string]string{"ultima": "2025-03-10"},
		Esperado:        EsperadoDorado{Vence: "2027-03-11T23:59:59Z", Hito: "auditoria#1"},
		CitaDelEsperado: "a proposito mal"}
	if err := EjecutarDorado(o, d); err == nil {
		t.Fatal("un esperado que no coincide con el motor debe fallar")
	}
}
