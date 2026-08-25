package main

// Tambien sin red. Los fixtures son respuestas REALES de Cellar recortadas:
//
//	eurlex-original.xhtml     Reglamento (UE) 2016/679, arts. 33 y 34, dialecto
//	                          "original" (p.oj-ti-art, p.oj-normal)
//	eurlex-consolidado.xhtml  el mismo reglamento en su version consolidada,
//	                          dialecto "consolidado" (p.title-article-norm,
//	                          div.norm, span.no-parag) y con los articulos
//	                          ANIDADOS dentro del contenedor de capitulo
//	eurlex-anexo.xhtml        Reglamento (UE) 2024/2847, art. 14 y anexo I, que
//	                          es un acto nuevo sin Formex y con el articulado
//	                          dentro de tablas
//	eurlex-noticia.xml        la ficha de Cellar del primero
//
// Los tres documentos de texto estan a proposito: si el parser solo aguantara
// uno, el corpus se quedaria sin la mitad del DOUE y no se sabria hasta el dia
// de ingerirla.

import (
	"errors"
	"strings"
	"testing"
)

func TestLaFichaDeCellarDaTituloELIYFechaDeActualizacion(t *testing.T) {
	titulo, eli, actualizada, err := parsearNoticiaCellar(leerFixture(t, "eurlex-noticia.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(titulo, "Reglamento (UE) 2016/679") {
		t.Fatalf("titulo %q", titulo)
	}
	if eli != "https://eur-lex.europa.eu/eli/reg/2016/679/oj" {
		t.Fatalf("eli %q: el ELI interno de la Oficina de Publicaciones se pasa al que resuelve "+
			"en EUR-Lex, que es el que se pega en un paquete", eli)
	}
	if actualizada == "" {
		t.Fatal("sin fecha de ultima modificacion no hay mitad izquierda de la tabla de vigilancia")
	}
	if got := citaCorta(titulo); got != "Reglamento (UE) 2016/679" {
		t.Fatalf("cita corta %q", got)
	}
}

func TestUnaFichaVaciaNoSeConfundeConUnaNormaSinTitulo(t *testing.T) {
	_, _, _, err := parsearNoticiaCellar([]byte(`<NOTICE type="object"><EXPRESSION/></NOTICE>`))
	if !errors.Is(err, ErrNormaNoEncontrada) {
		t.Fatalf("se esperaba ErrNormaNoEncontrada y dio %v", err)
	}
	_, _, _, err = parsearNoticiaCellar([]byte(`no soy xml`))
	if !errors.Is(err, ErrRespuestaIlegible) {
		t.Fatalf("se esperaba ErrRespuestaIlegible y dio %v", err)
	}
}

func articulosUE(t *testing.T, fixture string) []Articulo {
	t.Helper()
	as, err := parsearXHTMLEurLex(leerFixture(t, fixture),
		"Reglamento (UE) 2016/679 del Parlamento Europeo y del Consejo, de 27 de abril de 2016",
		"https://eur-lex.europa.eu/eli/reg/2016/679/oj")
	if err != nil {
		t.Fatalf("%s: %v", fixture, err)
	}
	return as
}

func TestElDialectoOriginalDelDOUESeLee(t *testing.T) {
	as := articulosUE(t, "eurlex-original.xhtml")
	if len(as) != 2 {
		t.Fatalf("el fixture tiene los articulos 33 y 34 y salieron %d: %v", len(as), referencias(as))
	}
	a, ok := porReferencia(as, "Artículo 33")
	if !ok {
		t.Fatalf("salieron %v", referencias(as))
	}
	if a.Numero != "33" || a.ID != "art_33" {
		t.Errorf("numero %q id %q", a.Numero, a.ID)
	}
	if !strings.HasPrefix(a.Titulo, "Notificación de una violación") {
		t.Errorf("titulo %q", a.Titulo)
	}
	if a.Cita != "Reglamento (UE) 2016/679, art. 33" {
		t.Errorf("cita %q", a.Cita)
	}
	if a.Fuente != "https://eur-lex.europa.eu/eli/reg/2016/679/oj#art_33" {
		t.Errorf("fuente %q", a.Fuente)
	}
	// El plazo, que es lo que este proyecto viene a calcular.
	if !strings.Contains(a.Texto, "72 horas") {
		t.Errorf("falta el plazo del articulo: %.300s", a.Texto)
	}
	// El rotulo no se repite dentro del texto.
	if strings.HasPrefix(a.Texto, "Artículo") {
		t.Errorf("el rotulo se ha colado en el texto: %.60s", a.Texto)
	}
	// Las letras de la lista del apartado 3 van en tabla y no se pueden perder.
	if !strings.Contains(a.Texto, "describir la naturaleza de la violación") {
		t.Errorf("se ha perdido el contenido que la fuente pinta como tabla: %.600s", a.Texto)
	}
}

// El dialecto consolidado cambia las clases Y anida los articulos dentro del
// contenedor del capitulo. Un parser que abriera "la primera division ELI que
// vea" se tragaria el capitulo entero como un solo articulo.
func TestElDialectoConsolidadoConArticulosAnidadosSeLee(t *testing.T) {
	as := articulosUE(t, "eurlex-consolidado.xhtml")
	if len(as) < 4 {
		t.Fatalf("el capitulo I tiene cuatro articulos y salieron %d: %v", len(as), referencias(as))
	}
	a, ok := porReferencia(as, "Artículo 1")
	if !ok {
		t.Fatalf("salieron %v", referencias(as))
	}
	if a.Titulo != "Objeto" {
		t.Errorf("titulo %q", a.Titulo)
	}
	if !strings.HasPrefix(a.Texto, "1.") {
		t.Errorf("el numero de apartado va en un span aparte y tiene que quedar pegado al "+
			"texto: %.80s", a.Texto)
	}
	if !strings.Contains(a.Texto, "protección de las personas físicas") {
		t.Errorf("texto %.200s", a.Texto)
	}
	for _, r := range referencias(as) {
		if strings.HasPrefix(r, "CAPÍTULO") || strings.HasPrefix(r, "cpt_") {
			t.Errorf("%q es estructura, no articulado: el contenedor de capitulo se ha "+
				"tomado por una division", r)
		}
	}
}

// Un acto de 2024: ya no tiene Formex, y su articulado va dentro de tablas.
func TestUnActoNuevoConElArticuladoEnTablasSeLee(t *testing.T) {
	as, err := parsearXHTMLEurLex(leerFixture(t, "eurlex-anexo.xhtml"),
		"Reglamento (UE) 2024/2847 del Parlamento Europeo y del Consejo, de 23 de octubre de 2024",
		"https://eur-lex.europa.eu/eli/reg/2024/2847/oj")
	if err != nil {
		t.Fatal(err)
	}
	a, ok := porReferencia(as, "Artículo 14")
	if !ok {
		t.Fatalf("salieron %v", referencias(as))
	}
	if a.Cita != "Reglamento (UE) 2024/2847, art. 14" {
		t.Errorf("cita %q", a.Cita)
	}
	// Los dos relojes del articulo, y los dos viven dentro de celdas de tabla.
	for _, quiero := range []string{"veinticuatro horas", "setenta y dos horas"} {
		if !strings.Contains(a.Texto, quiero) {
			t.Errorf("falta %q, que va dentro de una tabla: %.400s", quiero, a.Texto)
		}
	}
	// El anexo, que es un eli-container y no un eli-subdivision.
	anexo, ok := porReferencia(as, "ANEXO I")
	if !ok {
		t.Fatalf("el anexo no ha salido; salieron %v", referencias(as))
	}
	if anexo.Tipo != "anexo" {
		t.Errorf("tipo %q", anexo.Tipo)
	}
	if anexo.Titulo != "REQUISITOS ESENCIALES DE CIBERSEGURIDAD" {
		t.Errorf("titulo del anexo %q: el rotulo y el titulo son dos parrafos seguidos de la "+
			"misma clase", anexo.Titulo)
	}
	if !strings.Contains(anexo.Texto, "nivel adecuado de ciberseguridad") {
		t.Errorf("el cuerpo del anexo se ha perdido: %.300s", anexo.Texto)
	}
}

func TestUnDocumentoSinDivisionesELISeDenuncia(t *testing.T) {
	_, err := parsearXHTMLEurLex(
		[]byte(`<html xmlns="http://www.w3.org/1999/xhtml"><body><p>portada</p></body></html>`),
		"Reglamento (UE) 2016/679", "https://eur-lex.europa.eu/x")
	if !errors.Is(err, ErrSinArticulos) {
		t.Fatalf("se esperaba ErrSinArticulos y dio %v", err)
	}
	// Es EL caso real: la entrada legal-content/ES/TXT/XML devuelve HTTP 200 con
	// la portada del Diario Oficial. Un parser que se callara guardaria eso como
	// "la norma no tiene articulos" y la vigilancia diria que se derogo entera.
	if !strings.Contains(err.Error(), "Arreglo") {
		t.Errorf("el error no dice como arreglarlo: %v", err)
	}
}

func TestElCELEXSeValidaAntesDeMeterloEnUnaURL(t *testing.T) {
	buenos := []string{"32016R0679", "32024R2847", "02016R0679-20160504", "32011D0833"}
	for _, c := range buenos {
		if _, err := validarCELEX(c); err != nil {
			t.Errorf("%s es un CELEX valido y se rechaza: %v", c, err)
		}
	}
	malos := []string{
		"32016R0679/../../otra",
		"32016R0679?x=1",
		"http://ejemplo.invalid/32016R0679",
		"R0679",
		"",
		"32016 R0679",
	}
	for _, c := range malos {
		if _, err := validarCELEX(c); !errors.Is(err, ErrIdentificadorInvalido) {
			t.Errorf("%q tenia que rechazarse y dio %v", c, err)
		}
	}
	if got, _ := validarCELEX(" 32016r0679 "); got != "32016R0679" {
		t.Errorf("el CELEX se normaliza a mayusculas: %q", got)
	}
}

func TestElURNSugeridoSaleDelCELEXYNoDeUnaTabla(t *testing.T) {
	casos := map[string]string{
		"32016R0679": "urn:eu:reg:2016:679",
		"32022L2555": "urn:eu:dir:2022:2555",
		"32011D0833": "urn:eu:dec:2011:833",
		// Sector distinto de 3 (legislacion): no hay paquete que proponer.
		"52021DC0236": "",
		"C2020/123":   "",
	}
	for c, quiero := range casos {
		if got := urnSugeridoUE(c); got != quiero {
			t.Errorf("%s dio %q y se esperaba %q", c, got, quiero)
		}
	}
}

// La atribucion del DOUE no es cortesia: la Decision 2011/833/UE autoriza la
// reutilizacion con indicacion de la fuente. Si un dia alguien la deja vacia
// "porque ya se sabe", esto se pone rojo.
func TestLaAtribucionYLaLicenciaNoPuedenQuedarseVacias(t *testing.T) {
	casos := []struct{ nombre, licencia, atribucion string }{
		{"BOE", LicenciaBOE, AtribucionBOE},
		{"DOUE", LicenciaDOUE, AtribucionDOUE},
	}
	for _, c := range casos {
		if len(c.licencia) < 40 {
			t.Errorf("%s: la licencia de la fuente esta vacia o es un placeholder: %q",
				c.nombre, c.licencia)
		}
		if len(c.atribucion) < 40 {
			t.Errorf("%s: la atribucion esta vacia o es un placeholder: %q", c.nombre, c.atribucion)
		}
	}
	if !strings.Contains(LicenciaBOE, "13 TRLPI") {
		t.Error("la licencia del BOE tiene que citar el art. 13 TRLPI, que es lo que la autoriza")
	}
	if !strings.Contains(AtribucionBOE, "meramente informativo") {
		t.Error("las condiciones del BOE exigen decir que el texto consolidado es de caracter " +
			"meramente informativo, y esa frase tiene que viajar con el dato")
	}
	if !strings.Contains(LicenciaDOUE, "2011/833") {
		t.Error("la licencia del DOUE tiene que citar la Decision 2011/833/UE")
	}
	if !strings.Contains(AtribucionDOUE, "Union Europea") {
		t.Error("la atribucion del DOUE tiene que nombrar a la Union Europea: es la que la " +
			"Decision exige indicar")
	}
}
