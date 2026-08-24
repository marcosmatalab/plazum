package corpus

import (
	"os"
	"strings"
	"testing"
)

// paquete minimo valido, para partir de el en cada caso
func base() *Paquete {
	return &Paquete{
		URN: "ens@2022.311", Version: "1.0.0", Clase: Transcrito,
		Licencia: "art. 13 TRLPI", Fuente: "https://www.boe.es/eli/es/rd/2022/05/03/311",
		Consolidado: true,
		Entidades: []TipoEntidad{{
			Nombre: "sistema", Descripcion: "sistema de informacion en el ambito del ENS",
			Atributos: []Atributo{{
				Nombre: "categoria", Tipo: Enumerado,
				Valores: []string{"BASICA", "MEDIA", "ALTA"}, Escala: "ens.categoria",
				Obligado: true, Cita: "RD 311/2022 art. 40 y Anexo I",
			}},
		}},
		Preguntas: []Pregunta{{
			ID: "ens.q.categoria", Texto: "Que nivel alcanza cada dimension?",
			Cita: "RD 311/2022 Anexo I", Entidad: "sistema", Atributo: "categoria",
			Desbloquea: []string{"ens.art31.auditoria"},
		}},
		Obligaciones: []Obligacion{{
			ID: "ens.art31.auditoria", Articulo: "31",
			TextoLegal: "Los sistemas de informacion... seran objeto de una auditoria regular ordinaria, al menos cada dos anos.",
			Cita:       "RD 311/2022 art. 31", Vigencia: Vigencia{Desde: "2022-05-05"},
			Entregable: "ens.informe_auditoria", Preguntas: []string{"ens.q.categoria"},
			Recursos: []TipoRecurso{"Sistema"},
		}},
		Plantillas: []Plantilla{{
			ID: "ens.informe_auditoria", Titulo: "Informe de auditoria bienal",
			Cita: "RD 311/2022 art. 31 y guia CCN-STIC 802",
			Campos: []CampoPlantilla{
				{Nombre: "categoria", Origen: "entidad:sistema.categoria"},
				{Nombre: "fecha_anterior", Origen: "obligacion:ens.art31.auditoria.ultimo_hito"},
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
// mete el texto de una ISO o de PCI DSS en un paquete.
func TestReferencialRechazaTextoNormativo(t *testing.T) {
	p := base()
	p.URN, p.Clase = "iso27001@2022", Referencial
	p.Obligaciones[0].TextoLegal = strings.Repeat("x", LimiteTextoReferencial+1)
	errs := p.Validar()
	if len(errs) == 0 {
		t.Fatal("un paquete referencial con texto normativo largo debe ser rechazado")
	}
	if !strings.Contains(errs[0].Error(), "referencial") {
		t.Fatalf("el error debe explicar por que: %v", errs[0])
	}
	// y el identificador con titulo corto si pasa
	p.Obligaciones[0].TextoLegal = "A.8.13 Copia de seguridad de la informacion"
	if errs := p.Validar(); len(errs) != 0 {
		t.Fatalf("identificador y titulo corto deben permitirse: %v", errs)
	}
}

func TestDelegadoNoDistribuyeNadaYExigeHerramienta(t *testing.T) {
	p := base()
	p.URN, p.Clase = "cis@8.1", Delegado
	errs := p.Validar()
	if len(errs) != 2 { // lleva texto y no declara herramienta
		t.Fatalf("esperaba 2 errores, %d: %v", len(errs), errs)
	}
	p.Obligaciones[0].TextoLegal = ""
	p.Obligaciones[0].Delegado = "openscap:xccdf_org.cisecurity_benchmark"
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
	b.URN = "nis2@ue-2022.2555"
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
		ID: "ens.art38.conformidad", Cita: "RD 311/2022 art. 38",
		Vigencia: Vigencia{Desde: "2022-05-05"},
	})
	p.Preguntas = append(p.Preguntas, Pregunta{
		ID: "ens.q.ambito", Texto: "Es sector publico?", Cita: "art. 2",
		Entidad: "sistema", Atributo: "categoria",
		Desbloquea: []string{"ens.art31.auditoria", "ens.art38.conformidad"},
	})
	e := Entrevista([]*Paquete{p})
	if e[0].ID != "ens.q.ambito" {
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
	if tr[0].Obligacion != "ens.art31.auditoria" || tr[0].Origen == "" {
		t.Fatalf("traza incompleta: %+v", tr[0])
	}
}

func TestConectoresOrdenaPorObligacionesDesbloqueadas(t *testing.T) {
	p := base()
	p.Obligaciones = append(p.Obligaciones, Obligacion{
		ID: "ens.op.acc.5", Cita: "Anexo II", Vigencia: Vigencia{Desde: "2022-05-05"},
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
		ID: "ens.org.1", Cita: "Anexo II", Vigencia: Vigencia{Desde: "2022-05-05"},
	})
	c := Medir(p)
	if c.Total != 2 || len(c.SinAutomatizar) != 1 {
		t.Fatalf("la que no tiene recurso ni herramienta cuenta como no automatizada: %+v", c)
	}
	if !strings.Contains(c.String(), "ens.org.1") {
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
	// puede distribuir de ISO, PCI DSS, SOC 2 ni TISAX
	escribirPaquete(t, dir, "iso", `{
      "urn":"x@1","version":"1","clase":2,"fuente":"https://iso.org",
      "obligaciones":[{"id":"a","cita":"c","texto_legal":"`+strings.Repeat("y", 200)+`"}]}`)
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
