package main

// El borrador se prueba contra el formato de paquete DE VERDAD (nucleo/corpus),
// no contra una copia local del esquema. Si el formato cambia y el borrador se
// queda atras, esto se pone rojo el mismo dia.
//
// Y se prueba en las dos direcciones, que es lo que hace util a esta pieza:
//
//	tiene que CASAR con el formato   -> si no, no sirve de nada
//	tiene que NO validar tal cual    -> un paquete lo autoriza una persona

import (
	"encoding/json"
	"strings"
	"testing"

	"plazum/nucleo/corpus"
)

func borradorDePrueba(t *testing.T) (borradorPaquete, corpus.Paquete) {
	t.Helper()
	b := borradorDe(extraccionDePrueba(t, "boe-texto.xml"))
	crudo, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	var p corpus.Paquete
	if err := json.Unmarshal(crudo, &p); err != nil {
		t.Fatalf("el borrador no casa con el formato de paquete: %v\n%s", err, crudo)
	}
	return b, p
}

func TestElBorradorTieneLaFormaDeUnPaqueteDeVerdad(t *testing.T) {
	b, p := borradorDePrueba(t)
	if p.URN != "urn:es:rd:2022:311" {
		t.Errorf("urn %q", p.URN)
	}
	if p.Clase != corpus.Transcrito {
		t.Errorf("clase %v: BOE y DOUE son los dos estratos que se pueden transcribir", p.Clase)
	}
	if int(corpus.Transcrito) != claseTranscrito {
		t.Fatalf("HALLAZGO: la constante local vale %d y corpus.Transcrito vale %d. El borrador "+
			"estaria declarando una clase que no es la que cree", claseTranscrito, int(corpus.Transcrito))
	}
	// La procedencia sale como IDENTIFICADOR, no como URL: el borrador tiene
	// que escribir la ruta del ELI, y el enlace se deriva de ella. Un borrador
	// que siguiera escribiendo la URL entera meteria en el corpus el formato
	// que este cambio retira.
	if p.Identificador.Tipo != corpus.ELIBOE {
		t.Errorf("tipo de identificador %q: un borrador del BOE declara %q",
			p.Identificador.Tipo, corpus.ELIBOE)
	}
	if strings.Contains(p.Identificador.Valor, "://") {
		t.Errorf("el borrador escribio una URL (%q) donde va el identificador. Una "+
			"direccion como dato se rompe el dia que la pagina se mueve", p.Identificador.Valor)
	}
	if !strings.HasPrefix(p.Enlace(), "https://") {
		t.Errorf("del identificador %q no sale enlace (%q): las condiciones del BOE "+
			"exigen citar la fuente", p.Identificador.Valor, p.Enlace())
	}
	if esquemaELIBOE != string(corpus.ELIBOE) || esquemaELIUE != string(corpus.ELIUE) {
		t.Fatalf("HALLAZGO: las constantes locales valen %q y %q, y corpus dice %q y %q. El "+
			"borrador estaria declarando un esquema que el linter no conoce",
			esquemaELIBOE, esquemaELIUE, corpus.ELIBOE, corpus.ELIUE)
	}
	if identificadorDe(Origen{Jurisdiccion: "ue", ELI: "https://eur-lex.europa.eu/eli/reg/2016/679/oj"}) !=
		(identificadorBorrador{Tipo: esquemaELIUE, Valor: "reg/2016/679/oj"}) {
		t.Error("un borrador de la Union tiene que salir con el ELI de la Union en ruta")
	}
	if !p.Consolidado {
		t.Error("el texto del BOE es consolidado, y de eso depende el aviso de texto informativo")
	}
	if p.Vigencia.Desde == "" {
		t.Error("sin vigencia el paquete no carga")
	}
	if len(p.Obligaciones) == 0 {
		t.Fatal("el borrador sale sin obligaciones")
	}
	for _, o := range p.Obligaciones {
		if o.Articulo == "" || o.Cita == "" || o.TextoLegal == "" || o.Vigencia.Desde == "" {
			t.Errorf("obligacion incompleta en lo que SI se puede derivar: %+v", o)
		}
	}
	// Los dos campos de procedencia salen con su nombre definitivo.
	if b.LicenciaFuente == "" || b.Atribucion == "" {
		t.Fatal("el borrador tiene que llevar licencia_fuente y atribucion")
	}
	// Y licencia_fuente es un IDENTIFICADOR del vocabulario cerrado, no el
	// parrafo que lo explica. La constante local se cruza contra la de corpus,
	// igual que claseTranscrito: si el vocabulario se renombra alli, esto se
	// pone rojo el mismo dia en vez de escribir borradores que no cargan.
	if b.LicenciaFuente != string(corpus.BOETRLPI13) {
		t.Errorf("licencia_fuente %q: el borrador de una fuente espanola declara %q",
			b.LicenciaFuente, corpus.BOETRLPI13)
	}
	if regimenBOE != string(corpus.BOETRLPI13) || regimenDOUE != string(corpus.DOUEDecision2011833) {
		t.Fatalf("HALLAZGO: las constantes locales valen %q y %q, y corpus dice %q y %q. El "+
			"borrador estaria declarando un regimen que el linter no conoce",
			regimenBOE, regimenDOUE, corpus.BOETRLPI13, corpus.DOUEDecision2011833)
	}
	if regimenDe("ue") != string(corpus.DOUEDecision2011833) {
		t.Errorf("una fuente de la Union tiene que declarar %q", corpus.DOUEDecision2011833)
	}
	crudo, _ := json.Marshal(b)
	for _, campo := range []string{`"licencia_fuente"`, `"atribucion"`, `"identificador"`} {
		if !strings.Contains(string(crudo), campo) {
			t.Errorf("falta el campo %s en el borrador", campo)
		}
	}
}

// LA PROPIEDAD QUE IMPORTA: el borrador NO carga tal cual. Le faltan a proposito
// dos cosas independientes, y las dos son criterio humano, no transformacion de
// texto: el identificador de cada obligacion y su clase e2e.
//
// Dos y no una porque una sola comprobacion se quita de un tiron sin darse
// cuenta y entonces se podria commitear derecho sin leerlo.
func TestElBorradorNoCargaHastaQueUnaPersonaLoCompleta(t *testing.T) {
	_, p := borradorDePrueba(t)
	errs := p.Validar()
	if len(errs) == 0 {
		t.Fatal("HALLAZGO: el borrador valida tal cual sale. Un paquete de corpus lo autoriza " +
			"una persona: si el borrador carga, se puede commitear derecho sin haberlo leido")
	}
	var faltaID, faltaClase bool
	for _, err := range errs {
		if strings.Contains(err.Error(), "clase_e2e") {
			faltaClase = true
		}
		if strings.Contains(err.Error(), "no se puede citar") {
			faltaID = true
		}
	}
	if !faltaID {
		t.Error("tiene que faltar el id de la obligacion")
	}
	if !faltaClase {
		t.Error("tiene que faltar la clase e2e de la obligacion")
	}
}

// La otra mitad: lo que falta es SOLO eso. Si el borrador dejara ademas huecos
// que quien autora no puede saber (la cita, la vigencia, la fuente), no seria un
// borrador util sino un fichero vacio con formato.
func TestRellenandoLoQueEsCriterioHumanoElBorradorYaCarga(t *testing.T) {
	_, p := borradorDePrueba(t)
	for i := range p.Obligaciones {
		p.Obligaciones[i].ID = "ejemplo.borrador." + strings.ReplaceAll(
			strings.ToLower(p.Obligaciones[i].Articulo), " ", "_")
		p.Obligaciones[i].ClaseE2E = "procedimental"
	}
	if errs := p.Validar(); len(errs) != 0 {
		t.Fatalf("con el id y la clase puestos tenia que cargar, y quedan %d fallos: %v",
			len(errs), errs)
	}
}

// Un articulo derogado no genera obligacion: sigue en la extraccion (para que se
// vea que se derogo) y no en el borrador.
func TestElBorradorNoProponeObligacionesDerogadas(t *testing.T) {
	meta := metadatosDePrueba(t)
	as, err := parsearTextoBOE(leerFixture(t, "boe-texto-derogado.xml"), meta,
		"https://www.boe.es/eli/es/l/2002/07/11/34/con", opcionesBOE{Referencia: hoyDePrueba})
	if err != nil {
		t.Fatal(err)
	}
	e := armar(origenDeMetadatos(meta, piezasELI{Rango: "l", Ano: "2002", Numero: "34",
		Base: "https://www.boe.es/eli/es/l/2002/07/11/34"},
		"https://www.boe.es/eli/es/l/2002/07/11/34/con", ""), LicenciaBOE, AtribucionBOE, as, momento)

	b := borradorDe(e)
	if len(e.Articulos) != 2 {
		t.Fatalf("el fixture trae dos articulos y salieron %d", len(e.Articulos))
	}
	if len(b.Obligaciones) != 1 {
		t.Fatalf("uno de los dos esta derogado: el borrador tiene que proponer una obligacion "+
			"y propone %d", len(b.Obligaciones))
	}
	if strings.Contains(b.Obligaciones[0].TextoLegal, "Derogado") {
		t.Error("y no puede ser el derogado")
	}
}
