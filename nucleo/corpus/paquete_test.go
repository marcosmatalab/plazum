package corpus

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
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
		Licencia:       "art. 13 TRLPI",
		LicenciaFuente: BOETRLPI13,
		Atribucion: "Texto de una disposicion legal, reproducido citando la fuente oficial " +
			"que enlaza este paquete.",
		Identificador: Identificador{Tipo: ELIBOE, Valor: "es/rd/2022/05/03/311/con"},
		Consolidado:   true, Vigencia: Vigencia{Desde: "2022-05-05"},
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
	p.URN, p.Clase, p.LicenciaFuente = "urn:demo:referencial", Referencial, SinLicenciaDeTexto
	p.Atribucion = "Sin texto normativo: identificador y titulo corto. La copia licenciada " +
		"la aporta quien usa el paquete."
	p.Obligaciones[0].TextoLegal = strings.Repeat("x", LimiteTextoReferencial+1)
	errs := p.Validar()
	if len(errs) == 0 {
		t.Fatal("un paquete referencial con texto normativo largo debe ser rechazado")
	}
	// Por identidad del error y no por subcadena: "referencial" aparece tambien
	// en el URN del paquete y en media docena de mensajes de este linter.
	if !errors.Is(errs[0], ErrTextoRedistribuido) {
		t.Fatalf("el error debe ser el de la frontera legal: %v", errs[0])
	}
	// y el identificador con titulo corto si pasa
	p.Obligaciones[0].TextoLegal = "C.5.1 Titulo corto del control referenciado"
	if errs := p.Validar(); len(errs) != 0 {
		t.Fatalf("identificador y titulo corto deben permitirse: %v", errs)
	}
}

// referencialConTexto devuelve un paquete referencial valido y una funcion que
// le mete un parrafo en el campo que se le diga. Es la base de los dos tests de
// la frontera ampliada.
func referencialConTexto() *Paquete {
	p := base()
	p.URN, p.Clase, p.LicenciaFuente = "urn:demo:referencial", Referencial, SinLicenciaDeTexto
	p.Licencia = "referencial: sin licencia de texto normativo"
	p.Atribucion = "Sin texto normativo: identificador y titulo corto. La copia licenciada " +
		"la aporta quien usa el paquete."
	// Un referencial no distribuye texto legal; lo que trae es el identificador.
	p.Obligaciones[0].TextoLegal = "C.5.1 Titulo corto del control referenciado"
	return p
}

// LA FRONTERA LEGAL, CAMPO A CAMPO. El limite solo miraba texto_legal, asi que
// el enunciado de un control de un catalogo de pago entraba por la ayuda, por la
// descripcion de la entidad, por el texto de la pregunta o por el titulo de la
// plantilla, y el linter no decia nada. Es el mismo agujero que la clase fuera
// de rango, por otra puerta.
//
// Cada caso lleva su CONTROL NEGATIVO en el mismo bucle: el mismo campo con un
// caracter MENOS del limite tiene que validar limpio. Sin eso, un test que
// rechaza todo tambien pasaria.
func TestLaFronteraLegalMiraTodosLosCamposDeTextoNoSoloElTextoLegal(t *testing.T) {
	// El parrafo simula el enunciado de un control de un catalogo de pago: lo
	// que importa no es que dice, sino que no cabe.
	parrafo := strings.Repeat("y", LimiteTextoReferencial+1)
	justo := strings.Repeat("y", LimiteTextoReferencial)

	casos := []struct {
		campo  string
		poner  func(p *Paquete, v string)
		porque string
	}{
		{"Paquete.Obligaciones[].TextoLegal", func(p *Paquete, v string) {
			p.Obligaciones[0].TextoLegal = v
		}, "es el unico que se vigilaba"},
		{"Paquete.Obligaciones[].Titulo", func(p *Paquete, v string) {
			p.Obligaciones[0].Titulo = v
		}, "un titulo es donde se pega el enunciado del control, de buena fe"},
		{"Paquete.Obligaciones[].Articulo", func(p *Paquete, v string) {
			p.Obligaciones[0].Articulo = v
		}, "el articulo trae etiqueta dentro, o sea que cabe el control entero"},
		{"Paquete.Entidades[].Descripcion", func(p *Paquete, v string) {
			p.Entidades[0].Descripcion = v
		}, "la descripcion de la entidad se ensena en el formulario"},
		{"Paquete.Entidades[].Atributos[].Ayuda", func(p *Paquete, v string) {
			p.Entidades[0].Atributos[0].Ayuda = v
		}, "explicar un control copiando el control es lo que sale solo"},
		{"Paquete.Preguntas[].Texto", func(p *Paquete, v string) {
			p.Preguntas[0].Texto = v
		}, "una pregunta de alcance puede ser el requisito transcrito"},
		{"Paquete.Preguntas[].Ayuda", func(p *Paquete, v string) {
			p.Preguntas[0].Ayuda = v
		}, "la ayuda de la pregunta es el mismo caso que la del atributo"},
		{"Paquete.Plantillas[].Titulo", func(p *Paquete, v string) {
			p.Plantillas[0].Titulo = v
		}, "el titulo del entregable lo ve el auditor"},
	}

	for _, c := range casos {
		p := referencialConTexto()
		c.poner(p, parrafo)
		errs := p.Validar()
		var cazado error
		for _, e := range errs {
			if errors.Is(e, ErrTextoRedistribuido) && strings.Contains(e.Error(), c.campo) {
				cazado = e
			}
		}
		if cazado == nil {
			t.Errorf("HALLAZGO: %s admite %d caracteres de texto de un tercero en un paquete "+
				"referencial y el linter no dice nada (%s). Errores: %v",
				c.campo, len(parrafo), c.porque, errs)
			continue
		}
		// El error tiene que ser accionable: que fila y que campo.
		if !strings.Contains(cazado.Error(), c.campo) {
			t.Errorf("el error de %s no nombra el campo: %v", c.campo, cazado)
		}

		// Control negativo: el mismo campo justo en el limite valida limpio.
		q := referencialConTexto()
		c.poner(q, justo)
		if errs := q.Validar(); len(errs) != 0 {
			t.Errorf("%s con exactamente %d caracteres tiene que valer, y el linter da %v",
				c.campo, len(justo), errs)
		}
	}
}

// El techo de los campos de REFERENCIA. Una cita es un localizador y tiene que
// seguir valiendo aunque sea larga, porque ahi es donde el paquete explica por
// que apunta a ese articulo. Lo que no puede es ser un vertedero sin fondo.
func TestElCampoDeReferenciaTieneTechoPropioYNoElDeLaProsa(t *testing.T) {
	// Una cita de verdad, con la nota del autor detras: 229 caracteres es lo que
	// gasta hoy el corpus publicado. Con el limite de la prosa no cargaria.
	citaLarga := "CAT/DEMO 9999:2026 A.5.1. " + strings.Repeat("z", 200)
	if len(citaLarga) <= LimiteTextoReferencial {
		t.Fatalf("la cita de prueba tiene que pasar del limite de la prosa: %d", len(citaLarga))
	}
	p := referencialConTexto()
	p.Obligaciones[0].Cita = citaLarga
	if errs := p.Validar(); len(errs) != 0 {
		t.Fatalf("una cita larga es un localizador con su nota, no texto normativo: %v", errs)
	}

	// Y el techo existe: pasado LimiteCitaReferencial ya no es una cita.
	p.Obligaciones[0].Cita = strings.Repeat("z", LimiteCitaReferencial+1)
	errs := p.Validar()
	if len(errs) == 0 {
		t.Fatal("HALLAZGO: la cita de un referencial no tiene techo, asi que es un canal " +
			"abierto para meter el enunciado de un control")
	}
	if !errors.Is(errs[0], ErrCitaDesbordada) {
		t.Fatalf("el error tiene que ser el del techo de la cita: %v", errs[0])
	}
	// Control negativo del techo: justo en el limite sigue valiendo.
	p.Obligaciones[0].Cita = strings.Repeat("z", LimiteCitaReferencial)
	if errs := p.Validar(); len(errs) != 0 {
		t.Fatalf("una cita de exactamente %d caracteres tiene que valer: %v",
			LimiteCitaReferencial, errs)
	}
}

// El tercer techo, el de la DERIVACION: la cita_del_esperado de un dorado
// justifica la fecha paso a paso y por eso es legitimamente larga, pero sigue
// siendo un campo de texto libre que viaja dentro del paquete.
func TestLaDerivacionDeUnDoradoTieneSuPropioTecho(t *testing.T) {
	dorado := func(n int) *Paquete {
		p := referencialConTexto()
		p.Dorados = []Dorado{{Caso: "el plazo cruza el cambio de mes",
			Obligacion: p.Obligaciones[0].ID, CitaDelEsperado: strings.Repeat("z", n),
			Esperado: []EsperadoDorado{{Hito: "limite", Vence: "2026-05-04T23:59:59Z"}}}}
		return p
	}
	if errs := dorado(LimiteDerivacionReferencial).Validar(); len(errs) != 0 {
		t.Fatalf("la cuenta dia a dia de un dorado tiene que caber en %d caracteres: %v",
			LimiteDerivacionReferencial, errs)
	}
	errs := dorado(LimiteDerivacionReferencial + 1).Validar()
	if len(errs) == 0 {
		t.Fatal("HALLAZGO: la cita_del_esperado de un dorado no tiene techo, y los dorados " +
			"viajan dentro del paquete como todo lo demas")
	}
	if !errors.Is(errs[0], ErrCitaDesbordada) {
		t.Fatalf("el error tiene que ser el del techo: %v", errs[0])
	}
}

// rutasDeTexto recorre el TIPO del formato y devuelve la ruta de cada campo de
// cadena. Es el control de exhaustividad de la frontera legal: la clasificacion
// se escribe a mano, campo a campo, porque un criterio legal hay que poder
// leerlo; pero que no falte ninguno lo comprueba la maquina.
func rutasDeTexto(t reflect.Type, ruta string, out *[]string) {
	switch t.Kind() {
	case reflect.String:
		*out = append(*out, ruta)
	case reflect.Pointer:
		rutasDeTexto(t.Elem(), ruta, out)
	case reflect.Slice, reflect.Array:
		rutasDeTexto(t.Elem(), ruta+"[]", out)
	case reflect.Map:
		rutasDeTexto(t.Elem(), ruta+"[]", out)
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.PkgPath != "" { // no exportado: no viaja en el JSON del paquete
				continue
			}
			// La aplicabilidad YA NO queda fuera de este barrido. Estuvo
			// fuera con esta excusa: "sus campos son reglas de un dialecto
			// Datalog, viven en otro fichero y tienen su propio linter". La
			// excusa era falsa por la mitad que importa: ese linter comprueba
			// que la regla se PARSEA, no cuanto texto lleva dentro, y una
			// regla es una cadena libre con literales.
			rutasDeTexto(f.Type, ruta+"."+f.Name, out)
		}
	}
}

// unoDeCada es un paquete con UN valor en cada campo de texto del formato, para
// que camposDeTexto tenga que emitirlos todos. No pretende ser valido.
func unoDeCada() *Paquete {
	return &Paquete{
		URN: "urn:demo:completo", Version: "1", Licencia: "l",
		Identificador:  Identificador{Tipo: SinIdentificador, Valor: "f", Registro: "r", Motivo: "m"},
		FuenteHeredada: "vieja",
		LicenciaFuente: DelProyecto, Atribucion: "a",
		Vigencia: Vigencia{Desde: "2026-01-01", Hasta: "2027-01-01",
			Alternativas: []LecturaVigencia{{ID: "lv", Desde: "2028-01-01", Hasta: "2029-01-01", Cita: "c", Espera: "e"}}},
		Escalas: []string{"demo.escala"},
		Aplicabilidad: Aplicabilidad{
			Exporta: []string{"categoria"},
			Reglas: []ReglaSpec{{
				ID: "r", Cita: "c", Regla: "aplica(\"o\", S) :- en_ambito(S)",
				Agregado: "maximo", Sobre: "N",
				Escala: &EscalaSpec{Nombre: "demo.escala", Orden: []string{"BAJO", "ALTO"}},
			}},
		},
		Entidades: []TipoEntidad{{Nombre: "e", Descripcion: "d", Atributos: []Atributo{{
			Nombre: "a", Valores: []string{"v"}, Escala: "s", Ayuda: "y", Cita: "c",
		}}}},
		Preguntas: []Pregunta{{ID: "q", Texto: "t", Cita: "c", Entidad: "e",
			Atributo: "a", Desbloquea: []string{"o"}, Ayuda: "y"}},
		Obligaciones: []Obligacion{{
			ID: "o", Articulo: "a", Titulo: "ti", TextoLegal: "tl", Cita: "c",
			Vigencia: Vigencia{Desde: "2026-01-01", Hasta: "2027-01-01",
				Alternativas: []LecturaVigencia{{ID: "lv", Desde: "2028-01-01", Hasta: "2029-01-01", Cita: "c", Espera: "e"}}},
			Entregable: "pl", Recursos: []TipoRecurso{"R"}, Delegado: "d",
			Preguntas: []string{"q"}, ClaseE2E: "documental", Facetas: []string{"observable"},
			Temporalidad: &Temporalidad{Primitiva: "plazo", Hito: "h", Cadencia: "P1M",
				Limite: "P10D", En: "2026-12-02T23:59:59Z",
				Regimen: RegimenSpec{Computo: "naturales", Cierre: "exacto",
					Traslado: "ninguno"}, Disparador: map[string]string{"hecho": "x"},
				Hitos: []HitoSpec{{ID: "h2", Limite: "PT24H", DesdeHito: "h", Clase: "c",
					Nota:         "n",
					Alternativas: []LecturaSpec{{ID: "l", Limite: "PT48H", Cita: "c"}},
					Tope:         &TopeSpec{Desde: "otro_hecho", Limite: "PT12H", Caduca: true, Cita: "c"},
					Regimen: &RegimenSpec{Computo: "naturales", Cierre: "fin_de_dia",
						Traslado: "siguiente_habil"}}}},
			Escalado: []Escalon{{Tras: "P3D", A: "rol"}},
		}},
		Plantillas: []Plantilla{{ID: "pl", Titulo: "ti", Cita: "c",
			Campos: []CampoPlantilla{{Nombre: "n", Origen: "o"}}}},
		Dorados: []Dorado{{Caso: "caso", Obligacion: "o", Hasta: "2027-01-01",
			Hechos: map[string]string{"x": "2026-01-01"}, CitaDelEsperado: "c",
			Esperado:          []EsperadoDorado{{Hito: "h", Vence: "2026-01-11T23:59:59Z", Estado: "determinado"}},
			SubconjuntoPorque: "sp"}},
	}
}

// Ningun campo de texto del formato queda sin clasificar. Es lo que impide que
// el agujero vuelva: un campo nuevo que nadie clasifica es un canal nuevo por
// el que entra el texto que no se puede redistribuir.
func TestCadaCampoDeTextoDelFormatoEstaClasificado(t *testing.T) {
	var esperadas []string
	rutasDeTexto(reflect.TypeOf(Paquete{}), "Paquete", &esperadas)
	if len(esperadas) < 40 {
		t.Fatalf("el barrido del tipo encontro solo %d campos de texto: se ha roto el "+
			"recorrido y este test dejaria de probar nada", len(esperadas))
	}

	vistas := map[string]bool{}
	for _, c := range camposDeTexto(unoDeCada()) {
		vistas[c.Campo] = true
	}
	for _, r := range esperadas {
		if !vistas[r] {
			t.Errorf("el campo %s es texto libre del formato y la frontera legal no lo mira. "+
				"Clasificalo en camposDeTexto (prosa, referencia o derivacion): un campo sin "+
				"clasificar es por donde vuelve a entrar el texto de un catalogo de pago", r)
		}
	}
	// Y al reves: una ruta que ya no existe en el tipo es una clasificacion
	// muerta que da falsa sensacion de cobertura.
	enTipo := map[string]bool{}
	for _, r := range esperadas {
		enTipo[r] = true
	}
	for c := range vistas {
		if !enTipo[c] {
			t.Errorf("camposDeTexto clasifica %q, que ya no es un campo del formato", c)
		}
	}
}

func TestDelegadoNoDistribuyeNadaYExigeHerramienta(t *testing.T) {
	p := base()
	p.URN, p.Clase, p.LicenciaFuente = "urn:demo:delegada", Delegado, LaTieneLaHerramienta
	p.Atribucion = "No se distribuye texto: la licencia del contenido la tiene la " +
		"herramienta externa que ejecuta la comprobacion."
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

// La fuente sigue siendo obligatoria; lo que cambia es COMO se declara. Las
// puertas del identificador, con sus formas hostiles, viven en
// identificador_test.go.

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
      "urn":"x@1","version":"1","clase":2,
      "identificador":{"tipo":"sin-identificador","valor":"https://example.invalid/catalogo",
        "motivo":"catalogo de pago sin identificador citable"},
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
		// La ventana va declarada: sin ella el linter tiene DOS cosas que
		// decir y este caso solo mide una. Un test que mira errs[0] se rompe
		// en cuanto aparece otro fallo antes, aunque el que mide siga estando.
		Hasta:           "2027-03-11T23:59:59Z",
		Esperado:        []EsperadoDorado{{Hito: "auditoria#1", Vence: "2027-03-10T23:59:59Z"}},
		CitaDelEsperado: "art. 31"}}
	errs := p.Validar()
	if len(errs) == 0 || !strings.Contains(errs[0].Error(), "minimo 3") {
		t.Fatalf("un reloj con menos de 3 dorados debe rechazarse: %v", errs)
	}
}

func TestDoradoHuerfanoOSinCitaSeRechaza(t *testing.T) {
	p := base()
	p.Dorados = []Dorado{{Caso: "huerfano", Obligacion: "no.existe",
		Esperado: []EsperadoDorado{{Hito: "h", Vence: "2027-01-01T00:00:00Z"}}, CitaDelEsperado: "x"}}
	if errs := p.Validar(); len(errs) == 0 {
		t.Fatal("un dorado que apunta a una obligacion inexistente debe rechazarse")
	}
	p.Dorados = []Dorado{{Caso: "sin cita", Obligacion: p.Obligaciones[0].ID,
		Esperado: []EsperadoDorado{{Hito: "h", Vence: "2027-01-01T00:00:00Z"}}}}
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
		Esperado:        []EsperadoDorado{{Hito: "auditoria#1", Vence: "2027-03-11T23:59:59Z"}},
		CitaDelEsperado: "a proposito mal"}
	if err := EjecutarDorado(o, d); err == nil {
		t.Fatal("un esperado que no coincide con el motor debe fallar")
	}
}

// --- el titulo legible de una obligacion ---

// El titulo es OPCIONAL, y por eso hace falta un respaldo: hoy la unica etiqueta
// de una obligacion es su articulo, que en el corpus real es cosas como "Anexo
// II 4.2.5 Mecanismo de autenticacion (usuarios externos) [op.acc.5]". Sirve
// para leerlo, pero no es un titulo, y la pantalla de controles lo ensena tal
// cual. Que el respaldo exista es lo que permite anadir el campo sin reescribir
// hoy los 30 paquetes del corpus.
func TestTituloLegibleCaeAlArticuloYLuegoAlID(t *testing.T) {
	o := Obligacion{ID: "demo.auditoria_bienal"}
	if o.TituloLegible() != "demo.auditoria_bienal" {
		t.Errorf("sin titulo ni articulo, el respaldo es el id: %q", o.TituloLegible())
	}
	o.Articulo = "Anexo II 4.2.5 Mecanismo de autenticacion (usuarios externos)"
	if o.TituloLegible() != o.Articulo {
		t.Errorf("con articulo, el respaldo es el articulo: %q", o.TituloLegible())
	}
	o.Titulo = "Autenticacion de usuarios externos"
	if o.TituloLegible() != "Autenticacion de usuarios externos" {
		t.Errorf("con titulo, manda el titulo: %q", o.TituloLegible())
	}
	// Y devuelve vacio solo si no hay nada de donde derivarlo, que es un caso
	// que el linter ya no deja cargar: quien pinta una tabla de controles no
	// tiene que inventarse un texto de relleno.
	if (Obligacion{}).TituloLegible() != "" {
		t.Error("sin nada de donde derivar, devuelve vacio y el linter lo rechaza antes")
	}
}

func TestUnaObligacionSinIDNoCarga(t *testing.T) {
	p := base()
	p.Obligaciones[0].ID = ""
	errs := p.Validar()
	var cazado bool
	for _, e := range errs {
		if errors.Is(e, ErrObligacionSinID) {
			cazado = true
		}
	}
	if !cazado {
		t.Fatalf("una obligacion sin id no se puede citar ni seguir en el expediente: %v", errs)
	}
	// Control negativo: con id, ese error no sale.
	if errs := base().Validar(); len(errs) != 0 {
		t.Fatalf("el paquete base tiene que seguir validando: %v", errs)
	}
}

// --- la vigencia: el campo que se declaraba y no entraba en ningun calculo ---

func instante(t *testing.T, s string) time.Time {
	t.Helper()
	x, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("el instante del caso esta mal escrito: %v", err)
	}
	return x
}

// Los bordes, que es donde vive el fallo. Una fecha sola en "hasta" cubre el DIA
// ENTERO: "vigente hasta el 4 de mayo de 2024" es hasta el final de ese dia, no
// hasta su primer segundo. Cortar a las 00:00 deroga la norma un dia antes de lo
// que dice el BOE, y eso es una respuesta incorrecta, no un redondeo.
func TestLaVigenciaCubreElDiaEnteroYSusBordes(t *testing.T) {
	v := Vigencia{Desde: "2026-01-01", Hasta: "2026-05-04"}
	casos := []struct {
		cuando  string
		vigente bool
		porque  string
	}{
		{"2025-12-31T23:59:59Z", false, "un instante antes de entrar en vigor"},
		{"2026-01-01T00:00:00Z", true, "el primer instante del dia de entrada, inclusive"},
		{"2026-05-04T23:59:59Z", true, "el ultimo dia declarado cuenta entero"},
		{"2026-05-05T00:00:00Z", false, "el dia siguiente ya no"},
	}
	for _, c := range casos {
		got, err := v.VigenteEn(instante(t, c.cuando))
		if err != nil {
			t.Fatalf("%s: %v", c.cuando, err)
		}
		if got != c.vigente {
			t.Errorf("en %s la vigencia %+v da %v y tenia que dar %v (%s)",
				c.cuando, v, got, c.vigente, c.porque)
		}
	}

	// Abierta por arriba, que es el caso normal: una norma en vigor no declara
	// cuando la derogaran.
	abierta := Vigencia{Desde: "2026-01-01"}
	if ok, err := abierta.VigenteEn(instante(t, "2999-01-01T00:00:00Z")); err != nil || !ok {
		t.Errorf("una vigencia sin hasta sigue en vigor: %v %v", ok, err)
	}

	// Con hora, el corte es exacto al segundo declarado.
	conHora := Vigencia{Desde: "2026-01-01T00:00:00Z", Hasta: "2026-05-04T12:00:00Z"}
	if ok, _ := conHora.VigenteEn(instante(t, "2026-05-04T12:00:00Z")); !ok {
		t.Error("el instante declarado en hasta esta cubierto")
	}
	if ok, _ := conHora.VigenteEn(instante(t, "2026-05-04T12:00:01Z")); ok {
		t.Error("un segundo despues del instante declarado ya no esta cubierto")
	}
}

// La vigencia viene de un fichero de datos de un tercero. Las cuatro formas de
// que venga mal se rechazan EN LA CARGA, con centinela, y no al usarla.
func TestLaVigenciaMalEscritaNoCarga(t *testing.T) {
	casos := []struct {
		nombre    string
		romper    func(p *Paquete)
		centinela error
	}{
		{"fecha que no existe", func(p *Paquete) {
			p.Obligaciones[0].Vigencia.Desde = "2026-02-30"
		}, ErrVigenciaIlegible},
		{"fecha en palabras", func(p *Paquete) {
			p.Vigencia.Desde = "cuando entre en vigor"
		}, ErrVigenciaIlegible},
		{"desde posterior a hasta", func(p *Paquete) {
			p.Obligaciones[0].Vigencia = Vigencia{Desde: "2026-05-05", Hasta: "2022-05-05"}
		}, ErrVigenciaInvertida},
		{"paquete sin desde", func(p *Paquete) {
			p.Vigencia = Vigencia{}
		}, ErrVigenciaSinDesde},
	}
	for _, c := range casos {
		p := base()
		c.romper(p)
		errs := p.Validar()
		var cazado bool
		for _, e := range errs {
			if errors.Is(e, c.centinela) {
				cazado = true
			}
		}
		if !cazado {
			t.Errorf("%s: tenia que rechazarse con %v y los errores fueron %v",
				c.nombre, c.centinela, errs)
		}
	}
	// Control negativo: el paquete sin romper valida limpio, o sea que lo de
	// arriba no lo esta rechazando todo.
	if errs := base().Validar(); len(errs) != 0 {
		t.Fatalf("el paquete base tiene que validar: %v", errs)
	}
}

// LA RESPUESTA INCORRECTA QUE ESTO ARREGLA: una obligacion derogada seguia
// saliendo, porque el campo vigencia no entraba en ningun calculo.
func TestUnaObligacionDerogadaDejaDeEstarEnVigor(t *testing.T) {
	p := base()
	p.Vigencia = Vigencia{Desde: "2022-05-05"}
	p.Obligaciones[0].Vigencia = Vigencia{Desde: "2022-05-05", Hasta: "2024-05-05"}

	if ok, err := p.EnVigor(p.Obligaciones[0], instante(t, "2024-05-05T23:59:59Z")); err != nil || !ok {
		t.Errorf("el ultimo dia de la disposicion transitoria todavia se exige: %v %v", ok, err)
	}
	if ok, err := p.EnVigor(p.Obligaciones[0], instante(t, "2024-05-06T00:00:00Z")); err != nil || ok {
		t.Errorf("una obligacion derogada no se exige: %v %v", ok, err)
	}
	vs, err := p.VigentesEn(instante(t, "2026-01-01T00:00:00Z"))
	if err != nil {
		t.Fatal(err)
	}
	if len(vs) != 0 {
		t.Errorf("HALLAZGO: la obligacion derogada sigue en la lista: %v", vs)
	}
	// Control negativo: antes de la derogacion si sale.
	vs, err = p.VigentesEn(instante(t, "2023-01-01T00:00:00Z"))
	if err != nil || len(vs) != 1 {
		t.Errorf("antes de derogarse tiene que salir: %d %v", len(vs), err)
	}
}

// La vigencia de la obligacion se INTERSECA con la del paquete, y hereda de el
// lo que no declara. Sin interseccion, un paquete mal escrito (o escrito de mala
// fe) exige una obligacion antes de que su norma exista o despues de derogarla.
func TestLaObligacionNoSeExigeFueraDeLaVigenciaDeSuNorma(t *testing.T) {
	p := base()
	p.Vigencia = Vigencia{Desde: "2026-01-01", Hasta: "2026-12-31"}

	// Herencia: la obligacion no declara nada y sigue a su norma.
	p.Obligaciones[0].Vigencia = Vigencia{}
	if ok, err := p.EnVigor(p.Obligaciones[0], instante(t, "2026-06-01T00:00:00Z")); err != nil || !ok {
		t.Errorf("sin vigencia propia, hereda la del paquete: %v %v", ok, err)
	}
	if ok, err := p.EnVigor(p.Obligaciones[0], instante(t, "2027-01-01T00:00:00Z")); err != nil || ok {
		t.Errorf("derogada la norma, no queda obligacion: %v %v", ok, err)
	}

	// Una obligacion que se adelanta a su norma.
	p.Obligaciones[0].Vigencia = Vigencia{Desde: "2020-01-01"}
	if ok, err := p.EnVigor(p.Obligaciones[0], instante(t, "2021-01-01T00:00:00Z")); err != nil || ok {
		t.Errorf("HALLAZGO: se exige una obligacion seis anos antes de que exista su "+
			"norma: %v %v", ok, err)
	}

	// Y una que se queda viva despues de derogada la norma.
	p.Obligaciones[0].Vigencia = Vigencia{Desde: "2026-01-01"}
	if ok, err := p.EnVigor(p.Obligaciones[0], instante(t, "2030-01-01T00:00:00Z")); err != nil || ok {
		t.Errorf("HALLAZGO: la obligacion sobrevive a la norma que la sostiene: %v %v", ok, err)
	}

	// Control negativo del bloque: dentro de las dos vigencias, se exige.
	if ok, err := p.EnVigor(p.Obligaciones[0], instante(t, "2026-03-01T00:00:00Z")); err != nil || !ok {
		t.Errorf("dentro de las dos vigencias tiene que exigirse: %v %v", ok, err)
	}
}

// Sin fecha de inicio en ninguno de los dos, la respuesta no es "vigente": es
// "no lo se", y sale por el error. Dar por vigente lo que no se sabe es como se
// cuela una obligacion inventada en un expediente.
func TestSinFechaDeInicioNoSeSuponeVigente(t *testing.T) {
	p := &Paquete{URN: "urn:demo:sin-vigencia"}
	ok, err := p.EnVigor(Obligacion{ID: "demo.x"}, instante(t, "2026-01-01T00:00:00Z"))
	if ok {
		t.Fatal("HALLAZGO: sin vigencia declarada la obligacion se da por vigente")
	}
	if !errors.Is(err, ErrVigenciaSinDesde) {
		t.Fatalf("el error tiene que decir que falta el desde: %v", err)
	}
}

// --- por que me piden este dato: una cita por norma ---

// Cuando tres normas piden el mismo dato, Paquetes decia quienes eran pero solo
// sobrevivia la cita de UNA. Al comprador que pregunta "por que me piden este
// dato" se le respondia con el articulo de una de tres, elegido por orden
// alfabetico de URN.
func TestElCampoDiceQueCitaPoneCadaNormaQueLoPide(t *testing.T) {
	mismoDato := func(urn, cita, ayuda string) *Paquete {
		p := base()
		p.URN = urn
		p.Entidades[0].Atributos[0].Cita = cita
		p.Entidades[0].Atributos[0].Ayuda = ayuda
		return p
	}
	// A proposito en orden inverso al de URN, para que se vea que ordena.
	ps := []*Paquete{
		mismoDato("urn:demo:tercera", "tercera art. 7", "lo que dice la tercera"),
		mismoDato("urn:demo:primera", "primera art. 3", "lo que dice la primera"),
		mismoDato("urn:demo:segunda", "segunda art. 5", ""),
	}
	campos := EsquemaUI(ps)
	if len(campos) != 1 {
		t.Fatalf("el mismo atributo se pregunta una vez: %d", len(campos))
	}
	c := campos[0]
	if len(c.Peticiones) != 3 {
		t.Fatalf("HALLAZGO: el dato lo piden 3 normas y solo se sabe por que lo piden "+
			"%d: %+v", len(c.Peticiones), c.Peticiones)
	}
	quiero := []Peticion{
		{Paquete: "urn:demo:primera", Cita: "primera art. 3", Ayuda: "lo que dice la primera"},
		{Paquete: "urn:demo:segunda", Cita: "segunda art. 5"},
		{Paquete: "urn:demo:tercera", Cita: "tercera art. 7", Ayuda: "lo que dice la tercera"},
	}
	for i, q := range quiero {
		if c.Peticiones[i] != q {
			t.Errorf("peticion %d: %+v, esperaba %+v", i, c.Peticiones[i], q)
		}
	}
	// Lo de antes sigue igual: el arreglo es aditivo. Ayuda y Cita son las de
	// la primera por URN, y Paquetes sigue diciendo quienes son.
	if c.Cita != "primera art. 3" || len(c.Paquetes) != 3 {
		t.Errorf("Cita y Paquetes tienen que seguir significando lo mismo: %q %v",
			c.Cita, c.Paquetes)
	}
}

// Un paquete que declara dos veces la misma entidad no puede aparecer dos veces
// diciendo dos cosas distintas sobre el mismo dato.
func TestUnPaqueteQueSeRepiteNoPideElDatoDosVeces(t *testing.T) {
	p := base()
	repetida := p.Entidades[0]
	repetida.Atributos = []Atributo{{Nombre: "categoria", Tipo: Enumerado,
		Valores: []string{"BASICA"}, Cita: "cita distinta para el mismo dato"}}
	p.Entidades = append(p.Entidades, repetida)
	c := EsquemaUI([]*Paquete{p})[0]
	if len(c.Peticiones) != 1 {
		t.Errorf("un paquete pide el dato una vez, aunque se repita: %+v", c.Peticiones)
	}
}
