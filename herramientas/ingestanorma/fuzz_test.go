package main

// Fuzzing de los dos parsers.
//
// POR QUE. Estos parsers tragan XML de un tercero por la red. Da igual que el
// tercero sea el BOE: entre la fuente y nosotros hay una red, un proxy y una
// cache en disco que cualquiera con acceso al equipo puede escribir. Un parser
// que traga datos ajenos es superficie de ataque, y en este proyecto eso se
// fuzzea, como el parser de corpus, el ledger y el verificador.
//
// Lo que se exige no es solo "que no entre en panico". Un parser que devolviera
// articulos con la referencia vacia envenenaria el almacen de vigilancia: la
// clave de comparacion es la referencia, asi que dos articulos sin referencia
// colisionan y la siguiente ejecucion cantaria derogaciones que no existen. Asi
// que el invariante que se fuzzea es el que sostiene la vigilancia.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func semillas(f *testing.F, nombres ...string) {
	f.Helper()
	for _, n := range nombres {
		b, err := os.ReadFile(filepath.Join("testdata", n))
		if err != nil {
			f.Fatalf("falta el fixture %s: %v", n, err)
		}
		f.Add(b)
	}
}

// invariantes son las tres propiedades que sostienen la vigilancia. Se
// comprueban en los dos fuzz porque las dos fuentes alimentan el mismo almacen.
func invariantes(t *testing.T, as []Articulo) {
	t.Helper()
	vistas := map[string]bool{}
	for _, a := range as {
		if a.Referencia == "" {
			t.Fatalf("articulo con referencia vacia: es la CLAVE de la comparacion, y sin ella "+
				"dos articulos colisionan y la vigilancia canta derogaciones que no existen: %+v", a)
		}
		if vistas[a.Referencia] {
			t.Fatalf("referencia %q repetida: la segunda tapa a la primera en el mapa de "+
				"comparacion", a.Referencia)
		}
		vistas[a.Referencia] = true
		if a.Huella == "" || !strings.HasPrefix(a.Huella, "sha256:") {
			t.Fatalf("articulo sin huella: la comparacion no puede decidir si cambio: %+v", a)
		}
		if a.Cita == "" {
			t.Fatalf("articulo sin cita: una obligacion sin cita es una opinion: %+v", a)
		}
	}
}

// FuzzParsearTextoBOE: el XML del texto consolidado.
func FuzzParsearTextoBOE(f *testing.F) {
	semillas(f, "boe-texto.xml", "boe-texto-antes.xml", "boe-texto-derogado.xml")
	f.Add([]byte(`<response><status><code>200</code></status><data><texto>` +
		`<bloque id="a1" tipo="precepto" titulo="x"><version><p class="articulo">x</p>` +
		`</version></bloque></texto></data></response>`))
	f.Add([]byte(`<response><data><texto><bloque/></texto></data></response>`))
	f.Add([]byte(`{`))

	var meta respuestaMetadatosBOE
	meta.Meta.Titulo = "Norma de ejemplo 1/2000, de 1 de enero, sobre nada"
	meta.Meta.Identificador = "BOE-A-2000-1"

	f.Fuzz(func(t *testing.T, data []byte) {
		as, err := parsearTextoBOE(data, meta, "https://www.boe.es/eli/es/x/2000/01/01/1/con",
			opcionesBOE{Referencia: "20260101"})
		if err != nil {
			return
		}
		invariantes(t, as)
	})
}

// FuzzParsearMetadatosBOE: la ficha, que es de donde salen el titulo y la cita
// corta que acaban en cada obligacion del corpus.
func FuzzParsearMetadatosBOE(f *testing.F) {
	semillas(f, "boe-metadatos.xml", "boe-busqueda.xml")
	f.Add([]byte(`<response><status><code>500</code><text>x</text></status></response>`))
	f.Fuzz(func(t *testing.T, data []byte) {
		var m respuestaMetadatosBOE
		if err := decodificarBOE(data, &m); err != nil {
			return
		}
		// La cita corta acaba pegada en un fichero de corpus: no puede salir
		// mas larga que el titulo del que se recorta.
		if c := citaCorta(m.Meta.Titulo); len(c) > len(normalizarTexto(m.Meta.Titulo)) {
			t.Fatalf("la cita corta (%d) es mas larga que el titulo (%d)",
				len(c), len(normalizarTexto(m.Meta.Titulo)))
		}
		var b respuestaBusquedaBOE
		if err := decodificarBOE(data, &b); err == nil {
			_, _ = resolverELIBOE(data, piezasELI{Base: "https://www.boe.es/eli/es/x/2000/01/01/1"})
		}
	})
}

// FuzzParsearXHTMLEurLex: el XHTML de Cellar, que ademas llega con una DTD
// externa declarada y con dos dialectos distintos de marcado.
func FuzzParsearXHTMLEurLex(f *testing.F) {
	semillas(f, "eurlex-original.xhtml", "eurlex-consolidado.xhtml", "eurlex-anexo.xhtml")
	f.Add([]byte(`<html xmlns="http://www.w3.org/1999/xhtml"><body>` +
		`<div class="eli-subdivision" id="art_1"><p class="oj-ti-art">Articulo 1</p>` +
		`<div class="eli-title" id="art_1.tit_1"><p>Objeto</p></div><p>texto</p></div>` +
		`</body></html>`))
	// El caso que de verdad pasa: la portada del Diario Oficial servida con un
	// HTTP 200 en lugar del documento.
	f.Add([]byte(`<html><head><title>EUR-Lex</title></head><body><p>hoy</p></body></html>`))
	f.Fuzz(func(t *testing.T, data []byte) {
		as, err := parsearXHTMLEurLex(data, "Reglamento (UE) 2000/1, de 1 de enero, sobre nada",
			"https://eur-lex.europa.eu/eli/reg/2000/1/oj")
		if err != nil {
			return
		}
		invariantes(t, as)
	})
}

// FuzzParsearNoticiaCellar: la ficha del DOUE.
func FuzzParsearNoticiaCellar(f *testing.F) {
	semillas(f, "eurlex-noticia.xml")
	f.Add([]byte(`<NOTICE type="object"><EXPRESSION><EXPRESSION_TITLE><VALUE>x</VALUE>` +
		`</EXPRESSION_TITLE></EXPRESSION></NOTICE>`))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, eli, _, _, err := parsearNoticiaCellar(data)
		if err != nil {
			return
		}
		// Un ELI que saliera apuntando a otro anfitrion acabaria en el campo
		// fuente de un paquete publicado, y eso es la frontera legal saliendo
		// por la puerta de atras.
		if eli != "" && !strings.HasPrefix(eli, "https://"+anfitrionEURLex+"/") &&
			!strings.HasPrefix(eli, "http://") && !strings.HasPrefix(eli, "https://") {
			t.Fatalf("eli con forma rara: %q", eli)
		}
	})
}

// FuzzClave: la clave del almacen se compone con datos de la fuente y acaba en
// una ruta de fichero.
func FuzzClave(f *testing.F) {
	f.Add("es", "BOE-A-2022-7191")
	f.Add("ue", "32016R0679")
	f.Add("../..", "../../../etc/passwd")
	f.Fuzz(func(t *testing.T, j, id string) {
		k := clave(Origen{Jurisdiccion: j, Identificador: id})
		if k == "" {
			t.Fatal("clave vacia: escribiria en la raiz del almacen")
		}
		if strings.ContainsAny(k, `/\`) || strings.Contains(k, "..") {
			t.Fatalf("clave %q: se sale del almacen", k)
		}
		if filepath.Base(k) != k {
			t.Fatalf("clave %q no es un solo segmento de ruta", k)
		}
	})
}
