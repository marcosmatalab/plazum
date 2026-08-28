// Package estricto decodifica JSON rechazando lo que el tipo no declara.
//
// EL HALLAZGO QUE LO TRAE, y es de los caros. El 28-08-2026, escribiendo las 34
// cadencias del anexo de 2024/2690, las fichas traian un campo
// `cuando_cambiarlo` que NO EXISTIA en la estructura `Temporalidad`. Con el
// decodificador por defecto, `encoding/json` tira en silencio todo campo que el
// tipo no conozca: las 34 habrian cargado, el linter habria dado verde, las
// puertas habrian dado verde, y el unico dato que un CISO usa para adaptar el
// numero a su casa se habria perdido sin una linea de aviso en ningun sitio.
// Se vio por una pasada de cierre que leyo el esquema, o sea por suerte.
//
// LA CLASE ENTERA, que es lo que cierra este paquete. Un dato que el operador
// escribe a mano y que el programa descarta sin decirlo no es un fallo de ese
// campo: es una propiedad del decodificador, y la tiene en TODOS los ficheros
// que este producto lee del disco. Un `paquete.json` con `justificacion` donde
// tocaba `justificacion_del_intervalo`, un `contexto.json` con `raices_tsa`
// escrito `raices` (y entonces la verificacion cae a las raices por defecto sin
// que nadie lo pida), un `alcance.json` con una respuesta cuyo identificador
// tiene una letra de mas: los tres cargan hoy, los tres se comportan como si el
// campo no se hubiera escrito, y los tres son indistinguibles de que funcione.
//
// Es hermano del invariante 8 (en una frontera de confianza el valor cero tiene
// que ser el restrictivo) visto desde el otro lado: alli el peligro es que la
// AUSENCIA signifique "sin restriccion"; aqui es que la PRESENCIA de algo que
// no se entiende signifique lo mismo que la ausencia.
//
// DOS COSAS SE COMPRUEBAN, no una:
//
//	campo desconocido   el error dice QUE campo y EN QUE LINEA. `encoding/json`
//	                    da el nombre pero no el sitio, y en un paquete.json de
//	                    2.000 lineas con 45 obligaciones el nombre solo no
//	                    basta para encontrarlo.
//	nada detras         `json.Unmarshal` rechaza lo que venga despues del valor;
//	                    `Decoder.Decode` NO, se para en cuanto tiene uno
//	                    completo. Pasar de uno a otro para ganar
//	                    DisallowUnknownFields ACEPTARIA en silencio un fichero
//	                    con dos objetos pegados, o con basura detras, o el
//	                    resultado de un merge mal resuelto. Se cambiaba una
//	                    forma de tragar en silencio por otra.
package estricto

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// ErrCampoDesconocido: el JSON trae un campo que el tipo no declara.
var ErrCampoDesconocido = errors.New("campo desconocido")

// ErrSobraContenido: despues del valor JSON hay algo mas.
var ErrSobraContenido = errors.New("sobra contenido detras del valor JSON")

// Decodificar decodifica b sobre v rechazando campos desconocidos y contenido
// sobrante. `donde` es lo que se ensena delante del error: la ruta del fichero,
// o lo que sepa el que llama.
func Decodificar(b []byte, v any, donde string) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		if campo, ok := campoDesconocido(err); ok {
			return fmt.Errorf("%w: %s%s trae %q, que no existe en el formato.\n"+
				"  Un campo que el programa no conoce se DESCARTA EN SILENCIO con el "+
				"decodificador por defecto, asi que el fichero cargaria y todo pareceria "+
				"funcionar mientras ese dato no llega a ninguna parte.\n"+
				"  Arreglo: corrige el nombre (suele ser una errata o el nombre de un "+
				"campo que se penso y no llego a existir), o anade el campo al tipo si "+
				"de verdad hace falta",
				ErrCampoDesconocido, donde, situar(b, campo), campo)
		}
		return fmt.Errorf("%s: %w", donde, err)
	}
	// Y NADA DETRAS. Decode se para en cuanto tiene un valor completo, asi que
	// sin esto un fichero con dos objetos pegados cargaria el primero y
	// olvidaria el segundo sin decir nada.
	if t, err := d.Token(); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: %s tiene mas cosas despues del primer valor (%v, %v).\n"+
			"  Suele ser un merge mal resuelto, dos ficheros concatenados o una coma "+
			"donde iba un cierre. Se para aqui porque cargar solo el primero y callarse "+
			"el resto es indistinguible de que el fichero estuviera bien",
			ErrSobraContenido, donde, t, err)
	}
	return nil
}

// campoDesconocido saca el nombre del campo del error de encoding/json.
//
// SE HACE POR TEXTO PORQUE NO HAY OTRA. `encoding/json` devuelve un
// `*errors.errorString` para este caso: no hay tipo al que hacer errors.As ni
// campo del que leer el nombre. El formato es estable desde Go 1.10
// ("json: unknown field \"x\""), y si algun dia cambia, esta funcion deja de
// reconocerlo y el error sale con el mensaje crudo de la biblioteca, que sigue
// nombrando el campo. O sea: se degrada a menos util, nunca a incorrecto.
func campoDesconocido(err error) (string, bool) {
	const marca = `json: unknown field "`
	m := err.Error()
	i := strings.Index(m, marca)
	if i < 0 {
		return "", false
	}
	resto := m[i+len(marca):]
	j := strings.IndexByte(resto, '"')
	if j < 0 {
		return "", false
	}
	return resto[:j], true
}

// situar busca el campo en el texto crudo y devuelve ", linea N" o ", lineas
// N, M y P".
//
// Hace falta porque encoding/json da el NOMBRE y no el SITIO, y un paquete.json
// con 45 obligaciones tiene dos mil lineas.
//
// SE DAN TODAS LAS LINEAS DONDE APARECE EL NOMBRE, no una. La tentacion era
// senalar la mala, y NO SE PUEDE: `Decoder.InputOffset()` despues del error no
// apunta al campo (el decodificador lee por delante; medido sobre un objeto
// anidado de 50 bytes, devolvia el final del fichero), y el error de la
// biblioteca no lleva posicion. Dar la primera y callar seria afirmar cual es
// la mala sin saberlo, que es la familia de fallos que este repositorio lleva
// catorce hallazgos persiguiendo. Se dan todas y se dice que son todas: es una
// pista para abrir el fichero por donde toca, y no miente.
func situar(b []byte, campo string) string {
	aguja := []byte(`"` + campo + `"`)
	var lineas []string
	desde, linea := 0, 1
	for {
		i := bytes.Index(b[desde:], aguja)
		if i < 0 {
			break
		}
		i += desde
		linea += bytes.Count(b[desde:i], []byte("\n"))
		lineas = append(lineas, strconv.Itoa(linea))
		desde = i + len(aguja)
		if len(lineas) == maxLineasQueSeSenalan {
			if bytes.Contains(b[desde:], aguja) {
				lineas = append(lineas, "y mas")
			}
			break
		}
	}
	switch len(lineas) {
	case 0:
		return ""
	case 1:
		return ", linea " + lineas[0]
	default:
		return ", lineas " + strings.Join(lineas[:len(lineas)-1], ", ") + " y " + lineas[len(lineas)-1]
	}
}

// maxLineasQueSeSenalan acota el mensaje: un campo repetido en cuarenta
// obligaciones convertiria el error en un listado ilegible.
const maxLineasQueSeSenalan = 6
