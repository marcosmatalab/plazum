// Package demoempresa lleva el paquete de corpus del demo DENTRO del binario.
//
// Por que existe este fichero Go dentro de paquetes/, que es el arbol de datos.
// El demo tiene que funcionar sobre un binario recien descargado, en una
// maquina donde no hay repositorio, ni directorio paquetes/, ni red. Si
// `dutiq demo` necesitara encontrar el corpus en disco, la primera pantalla de
// valor dejaria de estar a un comando de distancia y el TTFV se perderia en un
// "descargate ademas esto". Un demo que pide preparativos no lo ejecuta nadie.
//
// Por que aqui y no una copia bajo cmd/. go:embed no puede salir del directorio
// del paquete, asi que la alternativa era commitear una copia del JSON al lado
// del CLI y vigilarla con un test de igualdad. Se descarto: dos ficheros con el
// mismo contenido son dos fuentes de verdad aunque haya un test, y el dia que
// el test se relaje el demo ensena un corpus distinto del publicado. Aqui la
// fuente es una: el mismo paquete.json que carga corpus.Cargar y que pasa el
// linter en CI es, byte a byte, el que se instala en el demo.
//
// Que NO va aqui: ni una obligacion, ni una regla, ni un identificador de
// norma. Solo la directiva de empotrado. Si algun dia hay que tocar el
// contenido del demo, se toca el JSON.
package demoempresa

import "embed"

// Ficheros son el paquete de corpus del demo, sus casos dorados y las
// respuestas de alcance de la empresa de ejemplo, tal como estan en el
// repositorio.
//
// Los dorados se empotran a proposito y no solo por completitud: el paquete
// declara tres relojes, y el linter de corpus RECHAZA un paquete con reloj y
// sin sus tres casos dorados. Un demo que instalara solo paquete.json no
// cargaria. Ademas permiten que el demo termine recalculando sus propios
// relojes contra el motor delante del operador, que es la diferencia entre
// ensenar unas fechas y ensenar que las fechas se comprueban.
//
//go:embed paquete.json pruebas/*.json alcance.json
var Ficheros embed.FS

// Directorio es el nombre que tiene el paquete dentro de paquetes/, y el que se
// le pone al instalarlo en el demo. Va como constante para que el instalador y
// el cargador no puedan discrepar.
const Directorio = "demo-empresa"
