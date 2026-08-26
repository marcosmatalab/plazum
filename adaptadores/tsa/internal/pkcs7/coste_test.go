package pkcs7

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
	"testing"
)

// Que el parseo de una secuencia de longitud indefinida no vuelva a ser
// cuadratico.
//
// EL FALLO. Hasta el 26-08-2026 `isIndefiniteTermination` hacia
// `bytes.Index(ber[offset:], []byte{0x0, 0x0}) == 0`, o sea recorria TODO lo que
// quedaba del bufer para responder una pregunta sobre dos bytes. Como se llama
// una vez por objeto dentro de la secuencia, el conjunto era cuadratico sobre
// entrada que elige el atacante. Medido entonces:
//
//	 4 KiB ->   573 us
//	 8 KiB ->  1,03 ms
//	16 KiB ->  3,63 ms
//	32 KiB -> 15,76 ms
//
// Cuatro veces por cada duplicacion, que es lo que se espera de un cuadratico.
// Con el tope `maxToken` de 32 KiB queda acotado en 16 ms; sin el, 1 MB serian
// ~16 segundos de CPU por peticion. Portado el arreglo de aguas arriba
// (comparacion directa de dos bytes, semanticamente identica), la misma medida
// da 5,7 en vez de 13,8 al cuadruplicar la entrada.
//
// POR QUE ESTA PUERTA MIRA EL CODIGO Y NO EL RELOJ, y esto se aprendio aqui
// mismo: la primera version media tiempos y comparaba la razon entre dos
// tamanos. Paso sola (5,7) y se puso ROJA al correr `go test ./...`, donde los
// paquetes van en paralelo y la maquina esta cargada: la misma razon subio a
// 11,4 sin que el codigo cambiara. Un umbral de tiempo en un runner compartido
// es una bomba con la mecha encendida, y este repositorio tiene regla escrita
// contra eso.
//
// LO QUE ESTA PUERTA CUBRE Y LO QUE NO, dicho para que nadie la lea de mas:
// caza la regresion CONCRETA (que vuelva el recorrido del bufer restante), que
// es la que ocurrio y la que puede volver al portar de aguas arriba. NO caza
// "cualquier coste cuadratico que alguien introduzca de otra forma". Para eso
// esta el fuzzing con su tope y, sobre todo, `maxToken`, que es la defensa que
// acota el peor caso venga de donde venga.
func TestElFinDeContenidoNoRecorreElBuferQueQueda(t *testing.T) {
	cuerpo := cuerpoDeFuncion(t, "ber.go", "isIndefiniteTermination")

	for _, prohibido := range []string{"bytes.Index", "bytes.HasPrefix", "bytes.Contains"} {
		if strings.Contains(cuerpo, prohibido) {
			t.Errorf(`isIndefiniteTermination vuelve a llamar a %s.

  Esa funcion responde si el fin de contenido empieza EN EL OFFSET, o sea una
  pregunta sobre DOS BYTES. Recorrer con %s lo que queda del bufer para
  contestarla la hace O(n), y como se llama una vez por objeto, el parseo entero
  se vuelve cuadratico sobre entrada que elige el atacante.

  Medido cuando pasaba: 4 KiB -> 573 us y 32 KiB -> 15,76 ms, cuatro veces por
  cada duplicacion.

  Arreglo, que es lo que hace aguas arriba y es semanticamente identico:
      return ber[offset] == 0 && ber[offset+1] == 0, nil
  La guarda len(ber)-offset < 2 de dos lineas mas arriba hace segura la
  indexacion directa.`, prohibido, prohibido)
		}
	}

	// CONTROL DEL DETECTOR, no del hecho: sin esto no se sabe si el recorrido
	// esta leyendo la funcion o una cadena vacia. Se comprueba que ve el cuerpo
	// de verdad buscando algo que TIENE que estar.
	if !strings.Contains(cuerpo, "ber[offset]") {
		t.Fatalf("no parece que se este leyendo el cuerpo de isIndefiniteTermination:\n%s\n"+
			"  Mientras eso pase, esta puerta esta dando verde sin mirar nada", cuerpo)
	}

	// Y la propiedad SE COMPRUEBA TAMBIEN POR COMPORTAMIENTO, sin reloj: el
	// resultado tiene que ser el mismo que el del recorrido, en las cuatro
	// formas que importan. Si algun dia alguien "optimiza" esto mal, aqui se ve.
	casos := []struct {
		nombre string
		ber    []byte
		offset int
		quiero bool
	}{
		{"fin de contenido justo en el offset", []byte{0x00, 0x00}, 0, true},
		{"fin de contenido mas adelante, NO cuenta", []byte{0x04, 0x00, 0x00, 0x00}, 0, false},
		{"un solo cero", []byte{0x00, 0x04}, 0, false},
		{"con cola detras", []byte{0x00, 0x00, 0x30, 0x00}, 0, true},
	}
	for _, c := range casos {
		got, err := isIndefiniteTermination(c.ber, c.offset)
		if err != nil {
			t.Errorf("%s: %v", c.nombre, err)
			continue
		}
		if got != c.quiero {
			t.Errorf("%s: esperaba %v y salio %v", c.nombre, c.quiero, got)
		}
		// Y lo mismo que habria dicho el recorrido, para que las dos formas no
		// puedan discrepar en silencio.
		conRecorrido := bytes.Index(c.ber[c.offset:], []byte{0x0, 0x0}) == 0
		if got != conRecorrido {
			t.Errorf("%s: la comparacion directa dice %v y el recorrido decia %v. Las dos "+
				"formas tienen que ser semanticamente identicas, o el port cambio el "+
				"comportamiento y no solo el coste", c.nombre, got, conRecorrido)
		}
	}
	// Y el borde: sin dos bytes disponibles, error y no panico.
	if _, err := isIndefiniteTermination([]byte{0x00}, 0); err == nil {
		t.Error("con un solo byte disponible tenia que dar error: la indexacion directa " +
			"depende de esa guarda")
	}
}

// cuerpoDeFuncion devuelve el codigo de una funcion, sin comentarios.
func cuerpoDeFuncion(t *testing.T, fichero, nombre string) string {
	t.Helper()
	fset := token.NewFileSet()
	arbol, err := parser.ParseFile(fset, fichero, nil, 0)
	if err != nil {
		t.Fatalf("no puedo parsear %s: %v", fichero, err)
	}
	for _, d := range arbol.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Name.Name != nombre || fn.Recv != nil {
			continue
		}
		// SE REIMPRIME EL AST, no se lee el rango crudo del fichero.
		//
		// La primera version leia los bytes de la funcion tal cual, y la puerta
		// se disparo con su PROPIO COMENTARIO: el arreglo lleva escrito dentro
		// por que ya no se usa `bytes.Index`, y el detector encontro ahi la
		// cadena que buscaba. Un detector que caza la explicacion de un arreglo
		// como si fuera el fallo es un falso positivo que acaba desactivandolo.
		//
		// El arbol se parseo sin comentarios (parser.ParseFile con 0), asi que
		// reimprimirlo devuelve solo codigo.
		var buf bytes.Buffer
		if err := printer.Fprint(&buf, fset, fn); err != nil {
			t.Fatal(err)
		}
		return buf.String()
	}
	t.Fatalf("no encuentro la funcion %s en %s. Si se ha renombrado, esta puerta esta "+
		"vigilando un nombre que ya no existe", nombre, fichero)
	return ""
}
