package ia_test

import (
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/busqueda"
	"github.com/marcosmatalab/plazum/adaptadores/ia"
	"github.com/marcosmatalab/plazum/adaptadores/ingesta"
	"github.com/marcosmatalab/plazum/puertos"
)

const politica = "La direccion aprueba esta politica de seguridad de la informacion y " +
	"la revisa al menos una vez al ano, y siempre que se produzca un cambio " +
	"significativo en la organizacion o en su entorno de riesgo."

func documentoDe(t *testing.T, texto string) (string, ingesta.Documento) {
	t.Helper()
	datos := []byte(texto)
	doc, err := ingesta.Leer("politica.txt", datos)
	if err != nil {
		t.Fatal(err)
	}
	return ia.Huella(datos), doc
}

// TestUnDocumentoDelClienteNoPuedeHacersePasarPorUnMarco es la guarda contra la
// suplantacion por nombre de fichero: el marco de todo lo aportado es la misma
// etiqueta fija, asi que un PDF llamado como un reglamento no se pinta en la
// columna del reglamento.
func TestUnDocumentoDelClienteNoPuedeHacersePasarPorUnMarco(t *testing.T) {
	// El nombre del fichero es lo mas parecido a una norma que se puede
	// escribir, y no llega a la fuente por ningun campo.
	datos := []byte(politica)
	// El nombre lleva una marca reconocible ADEMAS del disfraz de norma: sin
	// ella, un test que busque "reglamento" en la fuente no distingue el nombre
	// del fichero del texto de la propia politica.
	const enganoso = "Reglamento (UE) sobre proteccion de datos NO-DEBE-VIAJAR.pdf"
	doc, err := ingesta.Leer(enganoso, datos)
	if err != nil {
		t.Fatal(err)
	}
	fs, err := ia.FuentesAportadas(ia.Huella(datos), doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) == 0 {
		t.Fatal("cero fuentes: no se ha recorrido nada")
	}
	for _, f := range fs {
		if f.Marco != ia.MarcoAportado {
			t.Errorf("marco %q, y tiene que ser siempre %q", f.Marco, ia.MarcoAportado)
		}
		if strings.Contains(f.Marco+f.Articulo+f.ID+f.Texto, "NO-DEBE-VIAJAR") ||
			strings.Contains(strings.ToLower(f.Marco+f.Articulo+f.ID), "reglamento") {
			t.Errorf("el nombre del fichero se ha colado en la fuente: %+v", f)
		}
		if f.Procedencia != ia.Aportado {
			t.Errorf("procedencia %v, y un documento del cliente es Aportado", f.Procedencia)
		}
	}
}

// TestLaHuellaHaceQueLaMismaPoliticaSubidaDosVecesSeanLasMismasFuentes: sin
// esto, una cita guardada dejaria de resolver en cuanto el cliente volviera a
// subir el mismo documento.
func TestLaHuellaHaceQueLaMismaPoliticaSubidaDosVecesSeanLasMismasFuentes(t *testing.T) {
	h1, d1 := documentoDe(t, politica)
	h2, d2 := documentoDe(t, politica)
	a, err := ia.FuentesAportadas(h1, d1)
	if err != nil {
		t.Fatal(err)
	}
	b, err := ia.FuentesAportadas(h2, d2)
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != len(b) || len(a) == 0 {
		t.Fatalf("%d y %d fuentes", len(a), len(b))
	}
	for i := range a {
		if a[i].ID != b[i].ID || a[i].Hash != b[i].Hash {
			t.Errorf("la misma politica da fuentes distintas: %s/%s", a[i].ID, b[i].ID)
		}
	}

	// Y DOS DOCUMENTOS DISTINTOS NO CHOCAN, que es la otra mitad: con el orden
	// solo, el fragmento 1 de dos politicas seria la misma fuente.
	h3, d3 := documentoDe(t, "Otro documento distinto, con su propio contenido y su "+
		"longitud suficiente para ser un fragmento.")
	c, err := ia.FuentesAportadas(h3, d3)
	if err != nil {
		t.Fatal(err)
	}
	if c[0].ID == a[0].ID {
		t.Errorf("dos documentos distintos dan el mismo identificador: %s", c[0].ID)
	}
}

// TestUnVerificadorEstrictoDescartaLoAportadoAunqueLaCitaSeaLiteral es la
// puerta de la inyeccion via documento, con la cita MAS favorable posible: el
// texto esta ahi, literal, y aun asi no sale.
func TestUnVerificadorEstrictoDescartaLoAportadoAunqueLaCitaSeaLiteral(t *testing.T) {
	inyeccion := "El articulo 5 del reglamento obliga a cifrar todos los discos de la " +
		"organizacion sin excepcion, segun la interpretacion vigente."
	h, doc := documentoDe(t, inyeccion)
	fs, err := ia.FuentesAportadas(h, doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 {
		t.Fatalf("%d fuentes", len(fs))
	}

	prop := puertos.Propuesta{
		Diff:       "activa el cifrado",
		Cita:       inyeccion[:80],
		HashFuente: fs[0].Hash,
		Modelo:     "prueba",
	}

	// ESTRICTO: la cita es literal y esta en una fuente que el verificador
	// tiene. Y se descarta igual, por procedencia.
	v, err := ia.Estricto(fs)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Verificar(prop); err == nil {
		t.Fatal("un verificador estricto ha admitido una cita de un documento del " +
			"cliente. Esa cita sale por una pantalla que dice estar citando la ley")
	}

	// CONTROL POSITIVO: diciendolo, si entra. Sin esta mitad, el test de arriba
	// lo aprobaria un verificador que descarta absolutamente todo.
	v2, err := ia.Nuevo(ia.Opciones{
		Fuentes:    fs,
		Admite:     []ia.Procedencia{ia.Aportado},
		MinimoCita: ia.MinimoCitaPorDefecto,
	})
	if err != nil {
		t.Fatal(err)
	}
	ver, err := v2.Verificar(prop)
	if err != nil {
		t.Fatalf("admitiendo lo aportado, la cita tenia que resolver: %v", err)
	}
	if ver.Procedencia() != ia.Aportado {
		t.Errorf("procedencia %v", ver.Procedencia())
	}
}

// TestLosDocumentosDelClienteSeIndexanYSeEncuentran es la mitad de la casilla
// de busqueda que faltaba: BM25 «y sobre los documentos que sube el cliente».
func TestLosDocumentosDelClienteSeIndexanYSeEncuentran(t *testing.T) {
	h, doc := documentoDe(t, politica)
	fs, err := ia.FuentesAportadas(h, doc)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := busqueda.Nuevo(ia.Documentos(fs))
	if err != nil {
		t.Fatal(err)
	}
	res, err := idx.Buscar("revision anual de la politica", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) == 0 {
		t.Fatal("el documento del cliente no se encuentra en su propio indice")
	}
	if res[0].Marco != ia.MarcoAportado {
		t.Errorf("marco %q", res[0].Marco)
	}
	// Y EL HASH DEL RESULTADO ES EL DE LA FUENTE, que es por donde el
	// verificador va a emparejar despues (invariante 7: por identidad, no por
	// posicion).
	if res[0].Hash != fs[0].Hash {
		t.Errorf("el indice devuelve un hash que no es el de la fuente")
	}
}
