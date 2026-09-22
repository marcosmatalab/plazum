package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// EL DEMO OFRECE LAS PANTALLAS, Y NO PROMETE QUE SEAN GRATIS.
//
// # El hallazgo que la trae
//
// El 22-09-2026 se cableo `plazum demo --serve` y el texto que lo anunciaba
// decia que levantaba «las seis pantallas sobre este mismo estado, y no hay nada
// que configurar». Al ejercerlo de verdad las siete rutas contestaron 303: van
// detras de sesion y antes hay que dar de alta al primer administrador.
//
// La frase era falsa el dia que se escribio, que es la forma barata de esta
// familia: quien la escribe tiene la pantalla delante y da por hecho lo que no
// ha comprobado. Lo que la cierra por el otro lado es
// `TestNingunaPantallaContestaSinSesion`, que arranca el binario de verdad y
// demuestra el 303; esto comprueba que la pantalla del demo lo DICE.
func TestLaPantallaDelDemoOfreceLasPantallasYDiceQueFaltaElAlta(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	salida, errores, codigo := ejecutar(t, "--dir", dir, "--ahora", ahoraFijo())
	if codigo != 0 {
		t.Fatalf("plazum demo devolvio %d: %s", codigo, errores)
	}
	if !strings.Contains(salida, "plazum demo --serve") {
		t.Error("la pantalla de siguientes pasos no ofrece `plazum demo --serve`.\n" +
			"  El producto tiene seis pantallas accesibles y la unica forma de llegar a " +
			"ellas desde aqui era componer un `plazum serve` con tres banderas a mano.")
	}
	if !strings.Contains(salida, "administrador") {
		t.Error(`la pantalla ofrece --serve y no dice que queda dar de alta al primer
  administrador.

  Es la afirmacion de interfaz sin atar al comportamiento: las siete rutas
  contestan 303 sin sesion, y lo demuestra TestNingunaPantallaContestaSinSesion.
  Prometer «no hay nada que configurar» deja al lector delante de un formulario
  que no esperaba, que es justo donde cierra la pestana.`)
	}
}

// Y SI EL ALCANCE NO ESTA, --serve NO ARRANCA: LO DICE.
//
// # Por que esta rama existe, y por que hace falta recorrerla
//
// `plazum serve` arranca igual sin `--alcance`, y lo que sale entonces son las
// seis pantallas en su estado VACIO: el calendario sin filas, el acta sin
// periodo y el camino con su primer paso puesto. Eso se lee como «el producto no
// hace nada», que es la impresion contraria a la que este comando existe para
// dar, y es la misma familia que la medida del TTFV que no ejercia el sistema.
//
// Un descargo que ninguna entrada alcanza es un descargo que no existe, asi que
// esta rama va con su control positivo: se borra el alcance a proposito.
func TestServirElDemoSinAlcanceParaYDiceElArreglo(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	if _, errores, codigo := ejecutar(t, "--dir", dir, "--ahora", ahoraFijo()); codigo != 0 {
		t.Fatalf("plazum demo devolvio %d: %s", codigo, errores)
	}
	alcance := filepath.Join(dir, "paquetes", "demo-empresa", "alcance.json")
	if _, err := os.Stat(alcance); err != nil {
		t.Fatalf("el demo no ha dejado el alcance en %s: %v.\n"+
			"  Sin ese fichero este test no recorre la rama que dice recorrer, y un "+
			"descargo que ninguna entrada alcanza es un descargo que no existe", alcance, err)
	}
	if err := os.Remove(alcance); err != nil {
		t.Fatal(err)
	}

	var salida, errores bytes.Buffer
	codigo := servirElDemo(opcionesDemo{Dir: dir}, &salida, &errores)
	if codigo == 0 {
		t.Fatalf("servirElDemo ha devuelto 0 sin el alcance del demo.\n"+
			"  Arrancar igual saca las seis pantallas vacias, y eso se lee como que el "+
			"producto no hace nada.\n  Salida: %s", salida.String())
	}
	for _, quiero := range []string{"alcance", "Arreglo:", "plazum demo"} {
		if !strings.Contains(errores.String(), quiero) {
			t.Errorf("el error no dice %q. Un error de este producto lleva causa y arreglo, "+
				"no «error inesperado».\n  Dijo: %s", quiero, errores.String())
		}
	}
}
