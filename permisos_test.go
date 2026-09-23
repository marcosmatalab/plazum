package plazum

import (
	"os/exec"
	"strings"
	"testing"
)

// EL PRIMER COMANDO DEL REPOSITORIO TIENE QUE ARRANCAR EN EL CLON DE OTRO.
//
// # El fallo que la trae
//
// El 22-09-2026 los NUEVE ficheros `.sh` versionados estaban en modo `100644`.
// En un clon de Unix eso es `Permission denied` y codigo de salida 126: el
// README manda teclear `./comprobar.sh` y `./comprobar.sh` no existe como orden
// ejecutable. En un repositorio cuya tesis entera es «no te fies de mi,
// ejecutalo», la orden de ejecutarlo estaba rota.
//
// No se vio en tres meses porque **aqui nunca se ejecuta asi**: en Windows el
// bit no existe, `bash comprobar.sh` funciona igual, y CI invoca `bash` con la
// ruta delante. Es un fallo que solo se manifiesta en la maquina de quien
// evalua, que es la peor clase.
//
// # Por que hace falta la puerta y no basta el arreglo
//
// Porque el arreglo **se deshace solo**. El modo lo lleva el INDICE de git, no
// el sistema de ficheros, y en un checkout de Windows con `core.filemode=true`
// git ve 100644 en disco y ofrece revertirlo en cada `git add`. O sea que la
// regresion no necesita que nadie se equivoque: llega por trabajar donde se
// trabaja. Una correccion sin puerta en esta clase dura hasta el proximo
// `git commit -a`.
//
// # Lo que NO mira, dicho
//
// Mira el modo del indice, que es lo que viaja en el clon. No comprueba que el
// script haga nada util ni que tenga `#!` en la primera linea: eso lo comprueban
// `comprobar_test.go` y `puertas_test.go`, cada uno de lo suyo.
func TestTodoScriptVersionadoEsEjecutable(t *testing.T) {
	entradas := modosDeLosScripts(t)
	if len(entradas) < 5 {
		t.Fatalf("he encontrado %d ficheros .sh versionados y hoy son nueve: o el patron "+
			"dejo de casar, o los scripts se movieron, y en los dos casos esta puerta "+
			"estaria midiendo el vacio", len(entradas))
	}
	rotos := 0
	for ruta, modo := range entradas {
		if modo != "100755" {
			rotos++
			t.Errorf(`%s esta en modo %s y tiene que estar en 100755.

  En un clon de Unix eso es «Permission denied» y codigo 126. El README manda
  teclear ./comprobar.sh y ./comprobar.sh no arranca.

  Arreglo: git update-index --chmod=+x %s

  Y si esto sale rojo sin que nadie lo haya tocado, la causa es Windows: con
  core.filemode=true git ve 100644 en disco y revierte el modo en cada adicion al
  indice. Se apaga con «git config core.filemode false» en el checkout, que es
  config local y no viaja.`, ruta, modo, ruta)
		}
	}
	t.Logf("%d scripts versionados, %d sin bit de ejecucion", len(entradas), rotos)
}

// EL CONTROL NEGATIVO: el detector tiene que acusar un modo que no es 100755 y
// callarse con el que lo es.
//
// Sin esto, un `modosDeLosScripts` que devolviera un mapa vacio, o un parser que
// se equivocara de columna y devolviera siempre la cadena vacia comparada con
// una cadena vacia, dejaria la puerta verde para siempre. Es el verde vacio de
// siempre, aplicado a un parser de tres columnas.
func TestElDetectorDePermisosAcusaYSeCalla(t *testing.T) {
	for _, c := range []struct {
		nombre string
		linea  string
		quiero string
	}{
		{"ejecutable", "100755 0b12780 0\tcomprobar.sh", "100755"},
		{"sin bit", "100644 0b12780 0\tcomprobar.sh", "100644"},
		{"enlace simbolico", "120000 0b12780 0\tcomprobar.sh", "120000"},
	} {
		modo, ruta, ok := modoYRuta(c.linea)
		if !ok {
			t.Errorf("%s: el parser no entiende %q, que es la forma que imprime "+
				"`git ls-files -s`", c.nombre, c.linea)
			continue
		}
		if modo != c.quiero {
			t.Errorf("%s: el parser lee el modo %q y la linea dice %q", c.nombre, modo, c.quiero)
		}
		if ruta != "comprobar.sh" {
			t.Errorf("%s: el parser lee la ruta %q y la linea dice comprobar.sh", c.nombre, ruta)
		}
	}
	// Y una linea que no tiene la forma no se cuela como modo vacio: un modo
	// vacio comparado con 100755 acusaria, pero una RUTA vacia haria que la
	// puerta acusara a un fichero que no existe y el mensaje seria inservible.
	if _, _, ok := modoYRuta("basura sin tabulador"); ok {
		t.Error("el parser da por buena una linea que no tiene la forma de git ls-files -s")
	}
}

// modosDeLosScripts devuelve {ruta: modo} de los .sh que git tiene indexados.
//
// Se pregunta a git y no al disco a proposito: el disco de Windows no sabe
// representar el bit, asi que mirar ahi daria «todos rotos» aqui y «todos bien»
// en Linux, o sea una puerta que depende de la maquina. Lo que viaja en el clon
// es el modo del indice.
func modosDeLosScripts(t *testing.T) map[string]string {
	t.Helper()
	salida, err := exec.Command("git", "ls-files", "-s", "*.sh").Output()
	if err != nil {
		t.Skipf("SALTADO, no comprobado: git no contesta aqui (%v). NO es verde", err)
	}
	out := map[string]string{}
	for _, l := range strings.Split(strings.ReplaceAll(string(salida), "\r\n", "\n"), "\n") {
		if modo, ruta, ok := modoYRuta(l); ok {
			out[ruta] = modo
		}
	}
	return out
}

// modoYRuta parte una linea de `git ls-files -s`: "<modo> <sha> <fase>\t<ruta>".
func modoYRuta(linea string) (modo, ruta string, ok bool) {
	izq, der, hayTab := strings.Cut(linea, "\t")
	if !hayTab {
		return "", "", false
	}
	campos := strings.Fields(izq)
	if len(campos) != 3 || strings.TrimSpace(der) == "" {
		return "", "", false
	}
	return campos[0], strings.TrimSpace(der), true
}
