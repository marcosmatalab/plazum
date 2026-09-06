package ingesta_test

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/ingesta"
)

// ---------------------------------------------------------------------------
// El arnes: PDFs construidos a mano, porque un fixture binario opaco en el
// repositorio no se puede revisar y este es justo el sitio donde hay que poder
// leer que dice la entrada.
// ---------------------------------------------------------------------------

// pdfCon monta un PDF minimo pero valido con un flujo de contenido por pagina.
// Si comprimir, los flujos van con FlateDecode, que es lo que hace todo
// generador real.
func pdfCon(comprimir bool, paginas ...string) []byte {
	var sb bytes.Buffer
	sb.WriteString("%PDF-1.4\n")
	for i, p := range paginas {
		contenido := "BT /F1 12 Tf 72 720 Td (" + escapar(p) + ") Tj ET\n"
		cuerpo := []byte(contenido)
		dic := ""
		if comprimir {
			var z bytes.Buffer
			w := zlib.NewWriter(&z)
			_, _ = w.Write(cuerpo)
			_ = w.Close()
			cuerpo = z.Bytes()
			dic = "/Filter /FlateDecode "
		}
		fmt.Fprintf(&sb, "%d 0 obj\n<< /Type /Page /Parent 1 0 R >>\nendobj\n", 10+i)
		fmt.Fprintf(&sb, "%d 0 obj\n<< %s/Length %d >>\nstream\n", 100+i, dic, len(cuerpo))
		sb.Write(cuerpo)
		sb.WriteString("\nendstream\nendobj\n")
	}
	sb.WriteString("trailer\n<< /Root 1 0 R >>\n%%EOF\n")
	return sb.Bytes()
}

func escapar(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "(", `\(`)
	return strings.ReplaceAll(s, ")", `\)`)
}

const parrafo = "La direccion aprueba esta politica de seguridad de la informacion " +
	"y la revisa al menos una vez al ano, y siempre que se produzca un cambio " +
	"significativo en la organizacion."

// ---------------------------------------------------------------------------
// LO QUE SI SE LEE
// ---------------------------------------------------------------------------

func TestUnTextoPlanoSePartePorParrafosYNoPorPuntos(t *testing.T) {
	// El punto de «art. 5.1.b» es la trampa: partir por punto rompe una cita
	// legal por la mitad, que en este dominio es la peor forma de cortar.
	entrada := "Primer parrafo, que menciona el art. 5.1.b del reglamento y sigue.\n\n" +
		"Segundo parrafo, con su propio contenido suficientemente largo."
	doc, err := ingesta.Leer("politica.txt", []byte(entrada))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Formato != ingesta.TextoPlano {
		t.Errorf("formato %v", doc.Formato)
	}
	if len(doc.Fragmentos) != 2 {
		t.Fatalf("%d fragmentos, esperaba 2: %#v", len(doc.Fragmentos), doc.Fragmentos)
	}
	if !strings.Contains(doc.Fragmentos[0].Texto, "art. 5.1.b del reglamento") {
		t.Errorf("la cita legal salio partida: %q", doc.Fragmentos[0].Texto)
	}
	for i, f := range doc.Fragmentos {
		if f.Orden != i+1 {
			t.Errorf("fragmento %d con orden %d", i, f.Orden)
		}
		if f.Pagina != 0 {
			t.Errorf("un texto plano no tiene paginas y este dice %d", f.Pagina)
		}
	}
}

func TestUnPDFConTextoSeLeeYSeNumeranSusPaginas(t *testing.T) {
	for _, comprimido := range []bool{false, true} {
		nombre := "sin comprimir"
		if comprimido {
			nombre = "con FlateDecode"
		}
		t.Run(nombre, func(t *testing.T) {
			doc, err := ingesta.Leer("politica.pdf",
				pdfCon(comprimido, parrafo, "Segunda pagina con su texto, tambien bastante largo."))
			if err != nil {
				t.Fatal(err)
			}
			if doc.Formato != ingesta.PDF {
				t.Fatalf("formato %v", doc.Formato)
			}
			if doc.Paginas != 2 {
				t.Errorf("%d paginas, esperaba 2", doc.Paginas)
			}
			if len(doc.Fragmentos) != 2 {
				t.Fatalf("%d fragmentos: %#v", len(doc.Fragmentos), doc.Fragmentos)
			}
			// LA PAGINA SE ENSENA A UNA PERSONA: o es la buena o es 0.
			for i, f := range doc.Fragmentos {
				if f.Pagina != i+1 {
					t.Errorf("fragmento %d dice pagina %d", i, f.Pagina)
				}
			}
			if !strings.Contains(doc.Fragmentos[0].Texto, "revisa al menos una vez al ano") {
				t.Errorf("texto extraido: %q", doc.Fragmentos[0].Texto)
			}
		})
	}
}

func TestUnaCadenaHexadecimalTambienEsTexto(t *testing.T) {
	// Los generadores de PDF emiten <...> tan a menudo como (...). Si solo se
	// leyera una de las dos formas, medio documento saldria vacio y el
	// contraste de interpretabilidad lo llamaria ilegible: un falso error.
	hex := "BT <4C6120706F6C69746963612073652072657669736120616E75616C6D656E74652E> Tj ET"
	var sb bytes.Buffer
	sb.WriteString("%PDF-1.4\n1 0 obj\n<< /Type /Page >>\nendobj\n")
	fmt.Fprintf(&sb, "2 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", len(hex), hex)
	doc, err := ingesta.Leer("x.pdf", sb.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Fragmentos) != 1 || !strings.Contains(doc.Fragmentos[0].Texto, "revisa anualmente") {
		t.Fatalf("no se leyo la cadena hexadecimal: %#v", doc.Fragmentos)
	}
}

// ---------------------------------------------------------------------------
// LAS TRES FORMAS DE LA NADA (invariante 8), que es para lo que existe este
// paquete
// ---------------------------------------------------------------------------

func TestUnFicheroVacioNoEsUnError(t *testing.T) {
	// PRESENTE Y VACIO: es un dato. Cero fragmentos, sin error.
	for _, c := range []struct {
		nombre string
		datos  []byte
	}{
		{"cero bytes", []byte{}},
		{"solo espacios", []byte("   \n\n\t  \n")},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			doc, err := ingesta.Leer("vacio.txt", c.datos)
			if err != nil {
				t.Fatalf("un fichero vacio no es un error, y dio: %v", err)
			}
			if len(doc.Fragmentos) != 0 {
				t.Errorf("%d fragmentos de un fichero vacio", len(doc.Fragmentos))
			}
		})
	}
}

// TestUnPDFDelQueNoSaleTextoEsUnErrorYNoUnDocumentoVacio es LA casilla: un PDF
// del que no se extrae nada interpretable no da un valor por defecto.
//
// Las cuatro entradas son las cuatro formas reales de que esto pase, y las
// cuatro tienen que acabar en el mismo sitio. La que mas duele es la primera:
// un PDF escaneado es un PDF perfectamente valido, con sus paginas y su
// estructura, del que no sale una sola letra.
func TestUnPDFDelQueNoSaleTextoEsUnErrorYNoUnDocumentoVacio(t *testing.T) {
	escaneado := []byte("%PDF-1.4\n1 0 obj\n<< /Type /Page >>\nendobj\n" +
		"2 0 obj\n<< /Subtype /Image /Filter /DCTDecode /Length 8 >>\nstream\n" +
		"\xff\xd8\xff\xe0\x00\x10JF\nendstream\nendobj\ntrailer\n<< >>\n%%EOF\n")

	// Una fuente incrustada sin mapa a Unicode NO da cero bytes: da bytes, y
	// son basura. Si esto no fuera error, la basura se indexaria y una cita
	// podria resolver contra ella.
	basura := "BT ("
	for i := 0; i < 200; i++ {
		basura += string(rune(0x01 + i%8))
	}
	basura += ") Tj ET"
	sinMapa := []byte("%PDF-1.4\n1 0 obj\n<< /Type /Page >>\nendobj\n" +
		fmt.Sprintf("2 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n",
			len(basura), basura))

	casos := map[string][]byte{
		"un PDF escaneado, sin capa de texto":      escaneado,
		"una fuente incrustada sin mapa a Unicode": sinMapa,
		"un PDF cortado a la mitad":                pdfCon(true, parrafo)[:60],
		"un flujo con un filtro que no sabemos deshacer": []byte(
			"%PDF-1.4\n1 0 obj\n<< /Type /Page >>\nendobj\n" +
				"2 0 obj\n<< /Filter /LZWDecode /Length 4 >>\nstream\nzzzz\nendstream\nendobj\n"),
	}
	for nombre, datos := range casos {
		t.Run(nombre, func(t *testing.T) {
			doc, err := ingesta.Leer("politica.pdf", datos)
			if err == nil {
				t.Fatalf("NO dio error, y devolvio %d fragmentos. Un documento que no se "+
					"ha podido leer presentado como un documento vacio hace que el cliente "+
					"crea que su politica no dice nada", len(doc.Fragmentos))
			}
			if !errors.Is(err, ingesta.ErrSinTextoInterpretable) &&
				!errors.Is(err, ingesta.ErrFormatoNoReconocido) {
				t.Fatalf("error de otra clase: %v", err)
			}
			// EL ERROR NO ACUSA AL DOCUMENTO. Es la misma regla que en las
			// pantallas: decir «no dice nada» cuando lo que pasa es «no he
			// sabido leerlo» es acusar en falso al cliente.
			if errors.Is(err, ingesta.ErrSinTextoInterpretable) &&
				!strings.Contains(err.Error(), ingesta.FraseDelNoLeido) {
				t.Errorf("el error no descarga al documento: %v", err)
			}
		})
	}
}

// TestElContrasteDeInterpretabilidadNoSeLlevaPorDelanteUnDocumentoCorto es el
// control positivo del umbral: sin el, subirlo hasta que todo sea ilegible
// aprobaria el test de arriba entero.
func TestElContrasteDeInterpretabilidadNoSeLlevaPorDelanteUnDocumentoCorto(t *testing.T) {
	corto := "El responsable de seguridad es la directora de sistemas."
	doc, err := ingesta.Leer("nota.txt", []byte(corto))
	if err != nil {
		t.Fatalf("un documento corto pero legible tiene que leerse, y dio: %v", err)
	}
	if len(doc.Fragmentos) != 1 {
		t.Fatalf("%d fragmentos", len(doc.Fragmentos))
	}
	// Y con acentos, que son runas de mas de un byte y la trampa obvia del
	// contraste.
	if _, err := ingesta.Leer("nota.txt",
		[]byte("La política de seguridad se revisa anualmente por la dirección.")); err != nil {
		t.Errorf("un texto con acentos tiene que leerse, y dio: %v", err)
	}
}

// ---------------------------------------------------------------------------
// EL FORMATO LO DECIDE EL CONTENIDO
// ---------------------------------------------------------------------------

func TestLaExtensionNoDecideNada(t *testing.T) {
	// Un zip (que es lo que hay dentro de un .docx) renombrado a .pdf.
	zipFalso := append([]byte("PK\x03\x04"), bytes.Repeat([]byte{0x00, 0xff, 0x13}, 100)...)
	if _, err := ingesta.Leer("politica.pdf", zipFalso); !errors.Is(err, ingesta.ErrFormatoNoReconocido) {
		t.Fatalf("un zip renombrado a .pdf tiene que rechazarse, y dio: %v", err)
	}

	// Y al reves: un PDF de verdad con el nombre equivocado SI se lee, y se
	// dice que el nombre engana. Rechazarlo seria fiarse de la extension, que
	// es el dato que elige quien sube el fichero.
	doc, err := ingesta.Leer("politica.txt", pdfCon(true, parrafo))
	if err != nil {
		t.Fatalf("un PDF con el nombre equivocado tiene que leerse igual: %v", err)
	}
	if doc.Formato != ingesta.PDF {
		t.Errorf("formato %v", doc.Formato)
	}
	if !doc.NombreEnganoso {
		t.Error("no se avisa de que la extension no casa con el contenido")
	}
	// Y con la extension correcta, no se avisa: un aviso que sale siempre no
	// avisa de nada.
	if doc, _ := ingesta.Leer("politica.pdf", pdfCon(true, parrafo)); doc.NombreEnganoso {
		t.Error("avisa de nombre enganoso con la extension correcta")
	}
}

// ---------------------------------------------------------------------------
// ENTRADA ADVERSARIA
// ---------------------------------------------------------------------------

func TestUnaBombaDeDescompresionNiSeLeeEnteraNiSeCalla(t *testing.T) {
	// 64 MiB de ceros comprimidos caben en unos pocos KB. Sin tope, esto pide
	// la memoria entera del servidor de cumplimiento de alguien.
	var z bytes.Buffer
	w := zlib.NewWriter(&z)
	_, _ = w.Write(bytes.Repeat([]byte("BT (a) Tj ET\n"), 5<<20/13))
	_ = w.Close()
	if z.Len() > 1<<20 {
		t.Fatalf("la bomba de prueba pesa %d bytes comprimida y no comprime bastante", z.Len())
	}

	var sb bytes.Buffer
	sb.WriteString("%PDF-1.4\n1 0 obj\n<< /Type /Page >>\nendobj\n")
	fmt.Fprintf(&sb, "2 0 obj\n<< /Filter /FlateDecode /Length %d >>\nstream\n", z.Len())
	sb.Write(z.Bytes())
	sb.WriteString("\nendstream\nendobj\n")

	doc, err := ingesta.Leer("bomba.pdf", sb.Bytes())
	if err != nil {
		// Un error tambien vale: lo que no vale es quedarse sin memoria.
		return
	}
	if len(doc.Fragmentos) > ingesta.MaximoFragmentos {
		t.Errorf("%d fragmentos, por encima del maximo declarado", len(doc.Fragmentos))
	}
	// Y SE DICE. Leer media politica y callarlo es mentir sobre lo que se ha
	// mirado, que en cumplimiento no es un detalle de comodidad.
	if !doc.Truncado {
		t.Error("se llego a un limite y el documento no dice que esta truncado")
	}
}

func TestUnFicheroDemasiadoGrandeNiSeIntenta(t *testing.T) {
	grande := make([]byte, ingesta.MaximoEntrada+1)
	copy(grande, "%PDF-1.4\n")
	err := func() error { _, err := ingesta.Leer("x.pdf", grande); return err }()
	if !errors.Is(err, ingesta.ErrDemasiadoGrande) {
		t.Fatalf("error %v", err)
	}
	// El error dice cuanto se paso y que hacer: un error que solo dice «no» le
	// traslada el trabajo a quien lo lee.
	if !strings.Contains(err.Error(), "Arreglo:") {
		t.Errorf("el error no es accionable: %v", err)
	}
}

func TestUnPDFCifradoDaSuPropioErrorYNoBasura(t *testing.T) {
	cifrado := []byte("%PDF-1.4\n1 0 obj\n<< /Type /Page >>\nendobj\n" +
		"trailer\n<< /Encrypt 9 0 R /Root 1 0 R >>\n%%EOF\n")
	_, err := ingesta.Leer("secreto.pdf", cifrado)
	if !errors.Is(err, ingesta.ErrPDFCifrado) {
		t.Fatalf("error %v", err)
	}
	if strings.Contains(err.Error(), "contrasena") &&
		!strings.Contains(err.Error(), "quitale la contrasena") {
		t.Errorf("el error no dice como salir de ahi: %v", err)
	}
}

// TestSinPoderCuadrarLasPaginasNoSeInventaNinguna es la tercera forma de la nada
// aplicada al numero de pagina: una pagina equivocada manda a alguien a mirar
// donde no esta, asi que ante la duda se dice 0.
func TestSinPoderCuadrarLasPaginasNoSeInventaNinguna(t *testing.T) {
	// Dos flujos con texto y UNA sola pagina declarada: no cuadra.
	var sb bytes.Buffer
	sb.WriteString("%PDF-1.4\n1 0 obj\n<< /Type /Page >>\nendobj\n")
	for i, txt := range []string{parrafo, "Otro parrafo distinto, con longitud suficiente."} {
		c := "BT (" + txt + ") Tj ET"
		fmt.Fprintf(&sb, "%d 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n",
			20+i, len(c), c)
	}
	doc, err := ingesta.Leer("x.pdf", sb.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Fragmentos) != 2 {
		t.Fatalf("%d fragmentos", len(doc.Fragmentos))
	}
	for _, f := range doc.Fragmentos {
		if f.Pagina != 0 {
			t.Errorf("se ha inventado la pagina %d cuando el contaje no cuadra", f.Pagina)
		}
	}
	// Y el orden SI se sabe siempre, que es lo que hace que dos fragmentos del
	// mismo documento no colisionen aunque no se sepa su pagina.
	if doc.Fragmentos[0].Orden == doc.Fragmentos[1].Orden {
		t.Error("dos fragmentos con el mismo orden")
	}
}

func TestUnFragmentoLargoSeParteSinRomperPalabras(t *testing.T) {
	largo := strings.Repeat("palabra ", ingesta.LimiteFragmento/4)
	doc, err := ingesta.Leer("x.txt", []byte(largo))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Fragmentos) < 2 {
		t.Fatalf("%d fragmentos: no se ha partido", len(doc.Fragmentos))
	}
	for _, f := range doc.Fragmentos {
		if len([]rune(f.Texto)) > ingesta.LimiteFragmento {
			t.Errorf("fragmento de %d runas, por encima del limite", len([]rune(f.Texto)))
		}
		for _, p := range strings.Fields(f.Texto) {
			if p != "palabra" {
				t.Fatalf("palabra partida por la mitad: %q", p)
			}
		}
	}
}
