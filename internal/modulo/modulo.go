// Package modulo dice cual es la ruta de import de este modulo, LEYENDOLA de
// go.mod.
//
// POR QUE EXISTE, y no es higiene. Hasta el 28-08-2026 el modulo se llamaba
// `plazum` a secas, y cinco puertas llevaban esa ruta escrita a mano para
// decidir que es codigo de casa y que es de fuera:
//
//	arquitectura_test.go   el nucleo solo importa <modulo>/nucleo/...
//	dependencias_test.go   el binario no lleva nada que no sea <modulo>/... ni stdlib
//	ia_test.go             el nucleo no importa <modulo>/puertos
//	latido/frontera_test.go  el adaptador no mira al nucleo ni a las superficies
//
// Al pasar el modulo a `github.com/marcosmatalab/plazum` (hacia falta para que
// `go install` funcione, y con el repositorio ya publico hacia falta de
// verdad), esas cinco comparaciones dejaron de casar con nada.
//
// DOS DE ELLAS SE QUEDARON VERDES VIGILANDO EL VACIO, medido y no supuesto:
// con `esElPuertoDeIA` devuelto a la ruta cableada, TestElNucleoNoConoceLaIA
// pasa (no hay nada que detectar cuando se busca una cadena que ya no existe)
// y solo su control negativo se pone rojo. Igual la frontera del latido.
//
// O sea: la ruta del modulo estaba escrita en go.mod Y en cinco tests, y esa
// es exactamente la segunda lista que este repositorio lleva catorce hallazgos
// prohibiendo. Se lee de go.mod, que es donde el compilador la lee, o no se
// escribe.
//
// La puerta que impide que vuelva a cablearse es TestNadieCableaLaRutaDelModulo
// en la raiz: recorre el AST de todos los ficheros .go y prohibe que la ruta
// aparezca en un literal que NO sea un import.
package modulo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Ruta devuelve la ruta de import del modulo (la linea `module` de go.mod).
//
// Sube desde el directorio actual hasta encontrar go.mod, para que valga igual
// desde la raiz que desde el directorio de un paquete, que es donde `go test`
// deja el directorio de trabajo.
func Ruta() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		f := filepath.Join(dir, "go.mod")
		b, err := os.ReadFile(f) // #nosec G304 -- f se construye subiendo desde el cwd del proceso
		if err == nil {
			for _, l := range strings.Split(string(b), "\n") {
				l = strings.TrimSpace(l)
				if resto, ok := strings.CutPrefix(l, "module "); ok {
					m := strings.TrimSpace(resto)
					if m == "" {
						return "", fmt.Errorf("%s tiene una linea `module` vacia", f)
					}
					return m, nil
				}
			}
			return "", fmt.Errorf("%s no tiene linea `module`", f)
		}
		padre := filepath.Dir(dir)
		if padre == dir {
			return "", fmt.Errorf("no hay go.mod desde el directorio de trabajo hacia arriba; " +
				"sin el no se puede saber cual es la ruta de import del modulo, y una puerta " +
				"que la adivine estaria comparando contra una constante que se queda vieja")
		}
		dir = padre
	}
}

// EsDeCasa dice si un import es de este modulo.
func EsDeCasa(imp, ruta string) bool {
	return imp == ruta || strings.HasPrefix(imp, ruta+"/")
}

// Interno devuelve el camino de un import de casa SIN el prefijo del modulo
// ("nucleo/ventana"). Para los de fuera devuelve la cadena tal cual, porque un
// import externo no tiene parte interna que ensenar.
func Interno(imp, ruta string) string {
	if imp == ruta {
		return ""
	}
	if resto, ok := strings.CutPrefix(imp, ruta+"/"); ok {
		return resto
	}
	return imp
}
