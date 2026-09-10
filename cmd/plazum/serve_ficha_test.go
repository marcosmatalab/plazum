package main

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/adaptadores/ia"
	"github.com/marcosmatalab/plazum/adaptadores/ingesta"
	"github.com/marcosmatalab/plazum/superficies/documentos"
)

// LA CADENA DE LA PIEZA 7, DE EXTREMO A EXTREMO.
//
// Un documento subido a mano llega a una ficha propuesta con su parrafo
// verificado por hash, una persona acepta un campo, y lo aceptado queda con SU
// NOMBRE. Si esto no cierra, no hay pieza 7.
//
// SE MONTA EL ALMACEN DE VERDAD (`indicesPorCuenta`), no un doble: lo que se
// quiere descartar es que las piezas funcionen por separado y no juntas, que es
// la familia que este repositorio lleva catorce hallazgos persiguiendo.

// laPoliticaConFicha es un documento con cabecera, como los que sube un cliente.
const laPoliticaConFicha = `Politica de seguridad de la informacion

Fecha: 2026-01-15
Alcance: los sistemas de la sede de Madrid
Firmado por: Marta Ruiz
Valido hasta: 2027-01-15

La organizacion revisa su plan de continuidad cada doce meses y deja constancia
de la revision en el expediente.
`

func TestUnDocumentoSubidoAManoProponeSuFichaYUnaPersonaLaAcepta(t *testing.T) {
	t.Setenv("PLAZUM_SIN_IA", "1")
	a := nuevosIndicesPorCuenta(corpusInstalado(t))
	a.ahora = func() time.Time { return time.Date(2026, 9, 11, 9, 0, 0, 0, time.UTC) }
	const quien = "ciso@acme"
	ctx := context.Background()

	if _, err := a.Subir(ctx, quien, "politica.txt", []byte(laPoliticaConFicha)); err != nil {
		t.Fatalf("subiendo: %v", err)
	}

	// 1. LOS CUATRO CAMPOS SE PROPONEN, y cada uno con su parrafo.
	fs, err := a.Ficha(ctx, quien)
	if err != nil {
		t.Fatalf("pidiendo la ficha: %v", err)
	}
	if len(fs) == 0 {
		t.Fatal("el documento se ha subido y no se propone NI UN CAMPO de ficha.\n" +
			"  La cadena esta rota: o la extraccion no ve la cabecera, o la verificacion " +
			"por hash esta descartando lo que deberia dejar pasar. Las dos hay que mirarlas.")
	}
	porCampo := map[string]documentos.PropuestaDeFicha{}
	for _, f := range fs {
		porCampo[f.Campo] = f
		// CADA PROPUESTA TRAE EL PARRAFO DEL QUE SALE, y ese parrafo esta
		// LITERALMENTE en el documento. Es la puerta antialucinacion: lo que se
		// ensena es lo que pone en el fichero, no lo que dijo quien propuso.
		if f.Parrafo == "" {
			t.Errorf("%s se propone sin parrafo", f.Campo)
			continue
		}
		if !strings.Contains(laPoliticaConFicha, f.Parrafo) {
			t.Errorf("%s: el parrafo NO esta literalmente en el documento subido.\n"+
				"  parrafo: %q\n"+
				"  Eso significa que lo que se ensena no es lo que pone en el documento, "+
				"que es la puerta antialucinacion entera", f.Campo, f.Parrafo)
		}
		if f.Documento != "politica.txt" {
			t.Errorf("%s dice salir de %q", f.Campo, f.Documento)
		}
		if f.Huella == "" {
			t.Errorf("%s se propone sin huella, asi que aceptarlo no podria casar con SU "+
				"documento", f.Campo)
		}
	}
	for _, campo := range []string{
		"documentos.ficha.campo.fecha", "documentos.ficha.campo.alcance",
		"documentos.ficha.campo.firmante", "documentos.ficha.campo.caducidad",
	} {
		if _, hay := porCampo[campo]; !hay {
			t.Errorf("no se propone %s, y la cabecera del documento lo trae", campo)
		}
	}
	if p := porCampo["documentos.ficha.campo.fecha"]; p.Valor != "2026-01-15" {
		t.Errorf("la fecha propuesta es %q y el documento dice 2026-01-15", p.Valor)
	}

	// 2. NADA ESTA ACEPTADO TODAVIA. Es la mitad que separa una propuesta de un
	//    dato, y sin comprobarla el paso 3 no demostraria nada.
	if ac, err := a.Aceptados(ctx, quien); err != nil || len(ac) != 0 {
		t.Fatalf("antes de aceptar hay %d campos aceptados (err %v): una propuesta que "+
			"nace aceptada no es una propuesta", len(ac), err)
	}

	// 3. UNA PERSONA ACEPTA, Y QUEDA SU NOMBRE.
	elegida := porCampo["documentos.ficha.campo.firmante"]
	if err := a.Aceptar(ctx, quien, elegida); err != nil {
		t.Fatalf("aceptando: %v", err)
	}
	ac, err := a.Aceptados(ctx, quien)
	if err != nil {
		t.Fatal(err)
	}
	if len(ac) != 1 {
		t.Fatalf("tras aceptar uno hay %d campos aceptados", len(ac))
	}
	if ac[0].Quien != quien {
		t.Errorf("lo aceptado dice que lo acepto %q y lo acepto %q.\n"+
			"  Sin el nombre, lo aceptado es indistinguible de lo propuesto, que es "+
			"exactamente lo que el invariante 13 no admite", ac[0].Quien, quien)
	}
	if ac[0].Cuando != "2026-09-11T09:00:00Z" {
		t.Errorf("lo aceptado se fecha %q y el reloj decia 2026-09-11T09:00:00Z",
			ac[0].Cuando)
	}
	if ac[0].Valor != elegida.Valor || ac[0].Parrafo != elegida.Parrafo {
		t.Errorf("lo aceptado no es lo que se propuso:\n  aceptado %+v\n  propuesto %+v",
			ac[0], elegida)
	}
	t.Logf("cadena de la pieza 7: %d campos propuestos, 1 aceptado por %s", len(fs), quien)
}

// LA PUERTA ANTIALUCINACION DE LA PIEZA 7: una propuesta cuya cita no resuelve
// NO se ensena.
//
// Con la extraccion determinista de hoy esto no deberia poder ocurrir, y por eso
// hay que provocarlo: se desincroniza el verificador de la cuenta, que es
// exactamente lo que pasaria si el indice se rehiciera y la propuesta viniera de
// un documento que ya no esta. Ensenarla entonces seria citar un documento que
// el cliente no tiene.
func TestUnaPropuestaDeFichaQueNoResuelveNoSeEnsena(t *testing.T) {
	t.Setenv("PLAZUM_SIN_IA", "1")
	a := nuevosIndicesPorCuenta(corpusInstalado(t))
	const quien = "ciso@acme"
	ctx := context.Background()
	if _, err := a.Subir(ctx, quien, "politica.txt", []byte(laPoliticaConFicha)); err != nil {
		t.Fatal(err)
	}
	// CONTROL POSITIVO PRIMERO: con el verificador bueno SI salen. Sin esta
	// mitad, un `Ficha` que devolviera siempre vacio pasaria lo de abajo.
	antes, err := a.Ficha(ctx, quien)
	if err != nil || len(antes) == 0 {
		t.Fatalf("con el verificador bueno salen %d propuestas (err %v): este test no "+
			"puede demostrar nada", len(antes), err)
	}

	// Y AHORA EL VERIFICADOR DE OTRO DOCUMENTO. Las citas de las propuestas
	// dejan de resolver contra sus fuentes.
	const otroTexto = "otro documento que no tiene nada que ver con el primero"
	doc, err := ingesta.Leer("otro.txt", []byte(otroTexto))
	if err != nil {
		t.Fatal(err)
	}
	otras, err := ia.FuentesAportadas(ia.Huella([]byte(otroTexto)), doc)
	if err != nil {
		t.Fatal(err)
	}
	ver, err := ia.Nuevo(ia.Opciones{
		Fuentes: otras, Admite: []ia.Procedencia{ia.Aportado},
		// EL MISMO MINIMO QUE USA LA FICHA, para que lo unico que cambie sea LA
		// FUENTE. Con otro minimo, un descarte podria venir del largo de la cita
		// y este test estaria demostrando algo distinto de lo que dice.
		MinimoCita: MinimoCitaDeFicha,
	})
	if err != nil {
		t.Fatal(err)
	}
	a.mu.Lock()
	a.por["ciso@acme"].verFicha = ver
	a.mu.Unlock()

	despues, err := a.Ficha(ctx, quien)
	if err != nil {
		t.Fatal(err)
	}
	if len(despues) != 0 {
		t.Errorf("con el verificador desincronizado siguen saliendo %d propuestas de "+
			"ficha.\n"+
			"  Una cita que no resuelve contra su fuente NO se ensena: es la puerta "+
			"antialucinacion, y da igual que hoy la escriba un extractor de patrones y "+
			"no un modelo.\n  primera: %+v", len(despues), despues[0])
	}
}

// UN DOCUMENTO SIN CABECERA NO PROPONE NADA, Y LA PANTALLA LO DICE.
//
// No haber reconocido ningun campo NO dice que el documento no tenga fecha ni
// firmante: dice que aqui no se han reconocido, porque la lectura son patrones
// fijos. Es la misma familia que «cero hallazgos no es cero cumplimiento», y su
// descargo tiene que salir.
func TestUnDocumentoSinCabeceraNoProponeFichaYSeDice(t *testing.T) {
	t.Setenv("PLAZUM_SIN_IA", "1")
	a := nuevosIndicesPorCuenta(corpusInstalado(t))
	const quien = "ciso@acme"
	ctx := context.Background()
	if _, err := a.Subir(ctx, quien, "suelto.txt",
		[]byte("Este documento describe el procedimiento de gestion de incidentes.")); err != nil {
		t.Fatal(err)
	}
	fs, err := a.Ficha(ctx, quien)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 0 {
		t.Fatalf("de un texto sin cabecera salen %d propuestas: %+v", len(fs), fs)
	}
	// Y LA PANTALLA LO DICE CON SUS PALABRAS, que es la mitad que el almacen no
	// puede dar: una lista vacia sin frase se lee como que el documento no tiene
	// ficha.
	s := superficieDeDocumentos(t, a, func(*http.Request) string { return quien })
	cuerpo := verDocumentos(t, s).Body.String()
	if !strings.Contains(cuerpo, "NO dice que tus documentos no tengan fecha") {
		t.Errorf("con un documento sin cabecera, la pantalla NO trae el descargo de la "+
			"ficha:\n%s", cuerpo)
	}
}

// ACEPTAR DOS VECES EL MISMO CAMPO SUSTITUYE Y NO DUPLICA, y lo sustituye entero.
//
// Cambiar de opinion sobre la fecha de un documento es un acto nuevo y lo firma
// quien lo hace ahora. Dejar las dos filas daria un expediente que afirma dos
// cosas distintas del mismo campo sin decir cual vale.
func TestAceptarDosVecesElMismoCampoSustituyeYVuelveAFirmar(t *testing.T) {
	t.Setenv("PLAZUM_SIN_IA", "1")
	a := nuevosIndicesPorCuenta(corpusInstalado(t))
	instante := time.Date(2026, 9, 11, 9, 0, 0, 0, time.UTC)
	a.ahora = func() time.Time { return instante }
	ctx := context.Background()
	if _, err := a.Subir(ctx, "ana@acme", "politica.txt", []byte(laPoliticaConFicha)); err != nil {
		t.Fatal(err)
	}
	fs, err := a.Ficha(ctx, "ana@acme")
	if err != nil || len(fs) == 0 {
		t.Fatalf("sin propuestas no se puede probar nada (err %v)", err)
	}
	if err := a.Aceptar(ctx, "ana@acme", fs[0]); err != nil {
		t.Fatal(err)
	}
	// La misma propuesta, aceptada por OTRA persona y en OTRO instante.
	instante = instante.Add(24 * time.Hour)
	if err := a.Aceptar(ctx, "ana@acme", fs[0]); err != nil {
		t.Fatal(err)
	}
	ac, err := a.Aceptados(ctx, "ana@acme")
	if err != nil {
		t.Fatal(err)
	}
	if len(ac) != 1 {
		t.Fatalf("tras aceptar dos veces el mismo campo hay %d filas: un expediente con "+
			"dos valores del mismo campo no dice cual vale", len(ac))
	}
	if ac[0].Cuando != instante.UTC().Format(time.RFC3339) {
		t.Errorf("la fila sustituida conserva la hora vieja (%q): aceptar otra vez es un "+
			"acto nuevo y lo firma quien lo hace ahora", ac[0].Cuando)
	}
}

// LA FICHA DE UNA CUENTA NO LA VE OTRA (invariante 12).
func TestLaFichaDeUnaCuentaNoLaVeOtra(t *testing.T) {
	t.Setenv("PLAZUM_SIN_IA", "1")
	a := nuevosIndicesPorCuenta(corpusInstalado(t))
	ctx := context.Background()
	if _, err := a.Subir(ctx, "ana@acme", "politica.txt", []byte(laPoliticaConFicha)); err != nil {
		t.Fatal(err)
	}
	// CONTROL POSITIVO: Ana SI ve la suya.
	if fs, _ := a.Ficha(ctx, "ana@acme"); len(fs) == 0 {
		t.Fatal("Ana no ve su propia ficha: este test no puede demostrar nada")
	}
	if fs, err := a.Ficha(ctx, "otro@acme"); err != nil || len(fs) != 0 {
		t.Errorf("otra cuenta ve %d propuestas de la ficha de Ana (err %v).\n"+
			"  Lo que hay ahi dentro es la cabecera de la politica de una organizacion, "+
			"con el nombre de quien la firma", len(fs), err)
	}
	fs, _ := a.Ficha(ctx, "ana@acme")
	if err := a.Aceptar(ctx, "ana@acme", fs[0]); err != nil {
		t.Fatal(err)
	}
	if ac, err := a.Aceptados(ctx, "otro@acme"); err != nil || len(ac) != 0 {
		t.Errorf("otra cuenta ve %d campos aceptados por Ana (err %v)", len(ac), err)
	}
}
