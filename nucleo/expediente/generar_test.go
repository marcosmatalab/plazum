package expediente

import (
	"os"
	"testing"
)

// Valida en memoria el expediente del escenario de pruebas.
//
// Aqui NO se genera el expediente de demostracion, y conviene entender por que,
// porque es una tension de diseño de verdad y no una restriccion caprichosa.
//
// El expediente de demostracion es un artefacto de PRODUCTO: es lo que verifica
// `dutiq verify` recien instalado, y su valor esta justo en que enseña normas
// reales, con sus articulos. Un demo que enseña identificadores sinteticos no
// demuestra nada a quien lo abre.
//
// El escenario de ESTE fichero es un artefacto de PRUEBA, y vive en nucleo/, que
// no puede cablear normas: lo vigila TestNingunaNormaCableada, que desde el
// 25-08-2026 mira tambien los _test.go de nucleo/ y adaptadores/.
//
// Las dos cosas compartian un solo constructor, y con la purga de identificadores
// eso se volvio una mina: quien ejecutara la regeneracion convertiria el demo
// publicado en sintetico, las anclas de contexto-demo.json dejarian de cuadrar y
// TestLaDemoVerificaConElVerificadorDeVerdad se caeria.
//
// El generador del demo vive desde ahora en herramientas/generardemo, con el
// escenario en un fichero de datos que si puede nombrar normas reales, igual que
// hace herramientas/sellardemo con el sello. Alli se genera y alli esta la puerta
// que comprueba que lo publicado es exactamente lo que sale del escenario.
func TestElEscenarioDePruebaSigueSiendoUnExpedienteValido(t *testing.T) {
	e := construirExpediente(t)
	if _, err := e.Guardar(); err != nil {
		t.Fatalf("el escenario de prueba ya no serializa: %v", err)
	}
}

// Si alguien viene con el comando viejo en la memoria o en un runbook, que se
// encuentre el comando nuevo y no un demo roto.
func TestLaRegeneracionDelDemoYaNoViveAqui(t *testing.T) {
	if os.Getenv("DUTIQ_ESCRIBIR_DEMO") == "" {
		return
	}
	t.Fatal("DUTIQ_ESCRIBIR_DEMO ya no hace nada en este paquete, y no es un descuido.\n" +
		"  El escenario de aqui usa identificadores sinteticos, porque nucleo/ no puede\n" +
		"  cablear normas. Regenerar el demo desde aqui lo convertiria en un demo sintetico,\n" +
		"  dejaria las anclas de contexto-demo.json sin cuadrar y romperia\n" +
		"  TestLaDemoVerificaConElVerificadorDeVerdad.\n" +
		"  El generador del demo vive en herramientas/generardemo, con el escenario como\n" +
		"  fichero de datos:\n" +
		"    go run ./herramientas/generardemo             comprueba y no escribe nada\n" +
		"    go run ./herramientas/generardemo -escribir   dice que cambia y lo escribe\n" +
		"  El sello lo sigue reponiendo herramientas/sellardemo, que sale a la red.")
}
