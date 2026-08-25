package expediente

import (
	"os"
	"testing"
)

// Valida en memoria el expediente del escenario de pruebas.
//
// ANTES escribia ../../expediente-demo.json con DUTIQ_ESCRIBIR_DEMO=1, y ESO YA
// NO SE PUEDE HACER DESDE AQUI. Conviene entender por que, porque es una tension
// de diseño de verdad y no una restriccion caprichosa.
//
// El expediente de demostracion es un artefacto de PRODUCTO: es lo que verifica
// `dutiq verify` recien instalado, y su valor esta justo en que enseña normas
// reales, el ENS y el RGPD y el CRA, con sus articulos. Un demo que enseña
// urn:demo:agregada no demuestra nada a quien lo abre.
//
// El escenario de ESTE fichero es un artefacto de PRUEBA, y vive en nucleo/, que
// no puede cablear normas: lo vigila TestNingunaNormaCableada, que desde el
// 25-08-2026 mira tambien los _test.go de nucleo/ y adaptadores/.
//
// Las dos cosas compartian un solo constructor, y con la purga de identificadores
// eso se volvio una mina: quien ejecutara la regeneracion convertiria el demo
// publicado en sintetico, las anclas de contexto-demo.json dejarian de cuadrar y
// TestLaDemoVerificaConElVerificadorDeVerdad se caeria. Falla ruidosamente, no en
// silencio, pero es una mina igual.
//
// La puerta se cierra aqui en vez de dejarla abierta. El arreglo de verdad es
// sacar el generador del demo a herramientas/, con el escenario como fichero de
// datos que si puede nombrar normas reales, igual que hace herramientas/sellardemo
// con el sello. Esta apuntado como P1 en docs/pendientes.md.
func TestElEscenarioDePruebaSigueSiendoUnExpedienteValido(t *testing.T) {
	e := construirExpediente(t)
	if _, err := e.Guardar(); err != nil {
		t.Fatalf("el escenario de prueba ya no serializa: %v", err)
	}
}

// Si alguien viene con el comando viejo en la memoria o en un runbook, que se
// encuentre una explicacion y no un demo roto.
func TestLaRegeneracionDelDemoYaNoViveAqui(t *testing.T) {
	if os.Getenv("DUTIQ_ESCRIBIR_DEMO") == "" {
		return
	}
	t.Fatal("DUTIQ_ESCRIBIR_DEMO ya no hace nada en este paquete, y no es un descuido.\n" +
		"  El escenario de aqui usa identificadores sinteticos (urn:demo:...), porque nucleo/\n" +
		"  no puede cablear normas. Regenerar el demo desde aqui lo convertiria en un demo\n" +
		"  sintetico, dejaria las anclas de contexto-demo.json sin cuadrar y romperia\n" +
		"  TestLaDemoVerificaConElVerificadorDeVerdad.\n" +
		"  Arreglo pendiente (P1 en docs/pendientes.md): sacar el generador a herramientas/,\n" +
		"  con el escenario como fichero de datos que si puede nombrar el ENS, el RGPD y el CRA.\n" +
		"  Mientras tanto, expediente-demo.json y contexto-demo.json se editan a mano o no se\n" +
		"  editan, y el sello lo repone herramientas/sellardemo.")
}
