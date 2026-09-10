// Package demoempresa lleva el paquete de corpus del demo DENTRO del binario.
//
// Por que este paquete vive en demo/ y NO en paquetes/, que es de donde salio
// el 10-09-2026. Todo lo que hay bajo paquetes/ viaja entero al cliente: el
// Dockerfile lo copia sin filtro a /datos/paquetes y la release lo empaqueta
// con `corpus --empaquetar paquetes`. Mientras esto estuvo ahi, la empresa
// sintetica del demo —con sus siete obligaciones y sus dos figuras inventadas—
// se instalaba dentro del corpus REAL de quien se bajara el producto, y el
// escalado manda avisos a las figuras que el alcance declare. No es ruido en
// una pantalla: es un aviso de cumplimiento con un destinatario que no existe.
// La linea que lo impide vive en su propia puerta, no en el cuidado de nadie:
// TestElCorpusQueSeInstalaNoLlevaElPaqueteDelDemo.
//
// El demo tiene que funcionar sobre un binario recien descargado, en una
// maquina donde no hay repositorio, ni directorio paquetes/, ni red. Si
// `plazum demo` necesitara encontrar el corpus en disco, la primera pantalla de
// valor dejaria de estar a un comando de distancia y el TTFV se perderia en un
// "descargate ademas esto". Un demo que pide preparativos no lo ejecuta nadie.
//
// Por que aqui y no una copia bajo cmd/. go:embed no puede salir del directorio
// del paquete, asi que la alternativa era commitear una copia del JSON al lado
// del CLI y vigilarla con un test de igualdad. Se descarto: dos ficheros con el
// mismo contenido son dos fuentes de verdad aunque haya un test, y el dia que
// el test se relaje el demo ensena un corpus distinto del publicado. Aqui la
// fuente es una: el mismo paquete.json que se empotra es el que carga
// corpus.Cargar("demo") y el que pasa el linter y ejecuta sus dorados en cada
// ./comprobar.sh.
//
// Y POR ESO EL DIRECTORIO ES demo/demo-empresa/ Y NO demo/ A SECAS: corpus.Cargar
// enumera los SUBDIRECTORIOS de su raiz y abre <raiz>/<sub>/paquete.json, o sea
// que un demo/paquete.json suelto no lo puede cargar nadie. Sacarlo de paquetes/
// dejandolo donde el cargador no llega habria quitado del lazo el linter y los
// nueve dorados del unico paquete que ve TODO primer usuario, y el fallo se
// habria descubierto en su maquina, con el mensaje de demo.go diciendole que es
// un fallo del binario.
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

// Directorio es el nombre que se le pone al paquete al INSTALARLO en el demo,
// bajo <dir>/paquetes/. Va como constante para que el instalador y el cargador
// no puedan discrepar.
//
// NO CAMBIA CON LA MUDANZA A demo/, y es a proposito: es el nombre en la maquina
// del usuario, y de el cuelgan el guion de maquina limpia y la ayuda que
// cmd/plazum/alcance.go imprime con la ruta escrita. Mover el hogar en el
// repositorio no tiene por que mover nada en la maquina de quien ejecuta el demo.
const Directorio = "demo-empresa"
