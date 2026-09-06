package evidencia_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/busqueda"
	"github.com/marcosmatalab/plazum/adaptadores/evidencia"
	"github.com/marcosmatalab/plazum/adaptadores/ia"
	"github.com/marcosmatalab/plazum/adaptadores/ingesta"
)

// La politica del cliente, escrita como se escriben: parrafos sueltos, sin
// nombrar ninguna norma.
const politicaDelCliente = `La direccion aprueba esta politica de seguridad de la informacion y la revisa al menos una vez al ano, y siempre que se produzca un cambio significativo.

Existe un inventario de activos de informacion mantenido por el area de sistemas, con un responsable asignado a cada activo y su clasificacion.

Las copias de seguridad se realizan diariamente y se restauran de prueba una vez al trimestre para comprobar que sirven.

El personal recibe formacion en seguridad al incorporarse y con caracter anual.`

func arnes(t *testing.T, texto string) (*busqueda.Indice, *ia.Verificador) {
	t.Helper()
	datos := []byte(texto)
	doc, err := ingesta.Leer("politica.txt", datos)
	if err != nil {
		t.Fatal(err)
	}
	fs, err := ia.FuentesAportadas(ia.Huella(datos), doc)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := busqueda.Nuevo(ia.Documentos(fs))
	if err != nil {
		t.Fatal(err)
	}
	// Un verificador que admite lo aportado, DICIENDOLO: es lo que esta
	// superficie hace y no se puede hacer con el constructor Estricto.
	v, err := ia.Nuevo(ia.Opciones{
		Fuentes:    fs,
		Admite:     []ia.Procedencia{ia.Aportado},
		MinimoCita: ia.MinimoCitaPorDefecto,
	})
	if err != nil {
		t.Fatal(err)
	}
	return idx, v
}

// TestLoQueSeSenalaEsUnParrafoDelClienteYSaleLiteral es la propiedad entera:
// lo que la pantalla ensenaria sale del documento del cliente, palabra por
// palabra, y no de lo que dijo el buscador.
func TestLoQueSeSenalaEsUnParrafoDelClienteYSaleLiteral(t *testing.T) {
	idx, v := arnes(t, politicaDelCliente)
	hs, err := evidencia.Mapear(idx, v, []evidencia.Consulta{
		{ID: "inventario", Texto: "Mantener un inventario de activos con responsable asignado"},
		{ID: "copias", Texto: "Las copias de seguridad se realizan y se restauran de prueba"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(hs) != 2 {
		t.Fatalf("%d hallazgos: %+v", len(hs), hs)
	}
	por := map[string]evidencia.Hallazgo{}
	for _, h := range hs {
		por[h.ConsultaID] = h
	}
	if !strings.Contains(por["inventario"].Cita, "inventario de activos de informacion") {
		t.Errorf("inventario senala %q", por["inventario"].Cita)
	}
	if !strings.Contains(por["copias"].Cita, "copias de seguridad se realizan diariamente") {
		t.Errorf("copias senala %q", por["copias"].Cita)
	}
	for _, h := range hs {
		// LITERAL: la cita tiene que estar, tal cual, en el documento original.
		if !strings.Contains(politicaDelCliente, h.Cita) {
			t.Errorf("la cita de %s no esta literal en el documento: %q", h.ConsultaID, h.Cita)
		}
		if h.Fuente == "" || h.Referencia == "" {
			t.Errorf("hallazgo sin de donde sale: %+v", h)
		}
	}
}

// TestUnaConsultaSinEvidenciaNoSeRellenaConLoPrimeroQueSuene: la direccion
// contraria, y es la que de verdad protege. Sin ella, un Mapear que devuelva
// siempre el primer fragmento aprobaria el test de arriba.
func TestUnaConsultaSinEvidenciaNoSeRellenaConLoPrimeroQueSuene(t *testing.T) {
	idx, v := arnes(t, politicaDelCliente)
	hs, err := evidencia.Mapear(idx, v, []evidencia.Consulta{
		{ID: "cifrado", Texto: "Cifrar los soportes extraibles con algoritmos homologados"},
		{ID: "biometria", Texto: "Tratamiento de datos biometricos de pacientes hospitalizados"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(hs) != 0 {
		t.Fatalf("se han inventado %d hallazgos sobre una politica que no habla de eso: %+v",
			len(hs), hs)
	}
}

// TestElHuecoDeLaParafrasisEstaMedidoYDicho. No es un test que pida un arreglo:
// es la medida del limite que este paquete declara, escrita como caso para que
// el dia que deje de ser cierta (porque entre un modelo) alguien se entere.
//
// Y EL HUECO ES MAS ANCHO DE LO QUE SU NOMBRE DICE, lo cual se descubrio al
// ajustar el umbral con la muestra delante: no es solo la parafrasis
// («cada doce meses» por «anual»), es tambien la MORFOLOGIA. «Revision» no casa
// con «revisa» y «restauracion» no casa con «restauran», porque el tokenizador
// no lematiza y no va a lematizar sin una dependencia. Los dos casos de abajo
// estan escritos con las formas que SI casan, a proposito y diciendolo: un test
// escrito con las que no casan estaria midiendo la lematizacion que falta y no
// la propiedad que se afirma.
func TestElHuecoDeLaParafrasisEstaMedidoYDicho(t *testing.T) {
	idx, v := arnes(t, politicaDelCliente)

	// Las mismas palabras: se encuentra.
	conLasMismas, err := evidencia.Mapear(idx, v, []evidencia.Consulta{
		{ID: "revision", Texto: "La direccion aprueba la politica y la revisa cada ano"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(conLasMismas) != 1 {
		t.Fatalf("con las mismas palabras tenia que encontrarlo: %+v", conLasMismas)
	}

	// La misma obligacion dicha de otra forma: NO se encuentra, y esto es el
	// hueco. Un buscador lexico no sabe que «cada doce meses» es «anual».
	parafraseada, err := evidencia.Mapear(idx, v, []evidencia.Consulta{
		{ID: "revision", Texto: "Someter el documento rector a examen cada doce meses"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(parafraseada) != 0 {
		t.Logf("la parafrasis SI se encuentra ahora (%+v). Si es porque ha entrado un "+
			"modelo, este caso hay que reescribirlo; si no, el umbral se ha aflojado y "+
			"hay que mirar por que", parafraseada)
		t.Fatal("el limite declarado del paquete ya no es cierto: hay que actualizarlo")
	}
}

// TestElValorCeroDeMapearEsUnErrorYNoUnaListaVacia recorre las tres nadas de la
// llamada: sin indice, sin verificador y sin consultas. Ninguna es «no se
// encontro nada».
func TestElValorCeroDeMapearEsUnErrorYNoUnaListaVacia(t *testing.T) {
	idx, v := arnes(t, politicaDelCliente)
	cs := []evidencia.Consulta{{ID: "x", Texto: "inventario de activos"}}

	if _, err := evidencia.Mapear(nil, v, cs); !errors.Is(err, evidencia.ErrSinIndice) {
		t.Errorf("sin indice: %v", err)
	}
	if _, err := evidencia.Mapear(idx, nil, cs); !errors.Is(err, evidencia.ErrSinVerificador) {
		t.Errorf("sin verificador: %v", err)
	}
	for _, c := range [][]evidencia.Consulta{nil, {}} {
		if _, err := evidencia.Mapear(idx, v, c); !errors.Is(err, evidencia.ErrSinConsultas) {
			t.Errorf("sin consultas (%v): %v", c, err)
		}
	}
	// CONTROL POSITIVO: con las tres cosas, funciona. Sin esto, un Mapear que
	// devolviera error siempre aprobaria las cuatro afirmaciones de arriba.
	if hs, err := evidencia.Mapear(idx, v, cs); err != nil || len(hs) != 1 {
		t.Errorf("con todo puesto tenia que haber un hallazgo: %d, %v", len(hs), err)
	}
}

// TestUnaConsultaQueNoTieneNiUnTerminoConContenidoNoDevuelveNada: la tercera
// forma de la nada dentro de una consulta suelta. No para la llamada entera,
// pero tampoco produce un hallazgo.
func TestUnaConsultaQueNoTieneNiUnTerminoConContenidoNoDevuelveNada(t *testing.T) {
	idx, v := arnes(t, politicaDelCliente)
	hs, err := evidencia.Mapear(idx, v, []evidencia.Consulta{
		{ID: "vacia", Texto: "de la que se ha ..."},
		{ID: "buena", Texto: "inventario de activos de informacion"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(hs) != 1 || hs[0].ConsultaID != "buena" {
		t.Fatalf("%+v", hs)
	}
}

// TestUnDocumentoAjenoNoSeCuelaPorElVerificador es la guarda que separa este
// paquete de un buscador cualquiera: si el indice y las fuentes no son los
// mismos, no se ensena nada en vez de ensenar texto sin comprobar.
func TestUnDocumentoAjenoNoSeCuelaPorElVerificador(t *testing.T) {
	idx, _ := arnes(t, politicaDelCliente)
	_, otro := arnes(t, "Otro documento completamente distinto, que habla de la gestion "+
		"de proveedores y de sus contratos marco.")

	hs, err := evidencia.Mapear(idx, otro, []evidencia.Consulta{
		{ID: "inventario", Texto: "Mantener un inventario de activos con responsable asignado"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(hs) != 0 {
		t.Fatalf("se ha ensenado una cita que el verificador no tenia: %+v", hs)
	}
}
