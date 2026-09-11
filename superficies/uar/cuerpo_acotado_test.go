package uar

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// NINGUNA RUTA QUE MUTA ACEPTA UN CUERPO SIN TOPE, Y SON CUATRO.
//
// # El agujero, y por que es P0 aqui y no en otra superficie
//
// `decidir` y `excusar` leian `motivo` y `a` con `PostFormValue`, que parsea el
// cuerpo ENTERO en memoria, y los metian en `ledger`. Un POST con un mega de
// `motivo` escribia 1.048.789 bytes en un fichero **append-only**, que por
// definicion no se puede podar sin romper la cadena de hashes. No es un
// desperdicio de disco: es un dato que no se puede quitar nunca.
//
// Y no hacia falta ninguna herramienta: `curl` con un fichero grande.
//
// # LAS CUATRO, y por que no valen dos
//
// `abrir` ya tenia `MaxBytesReader` (abrir.go:132) porque sube un CSV, asi que
// quien mirase el paquete veria un tope y daria el resto por hecho. Las otras
// tres no lo tenian. `cerrar` no lee ningun campo, pero recibe el cuerpo igual y
// es una ruta que muta: dejarla fuera obliga a razonar cada vez por que una de
// las cuatro es distinta, y ese razonamiento se pierde en cuanto alguien le
// anada un campo.
//
// # LA HERMANA YA LO TENIA RESUELTO, y eso es lo que hace esto peor
//
// `superficies/documentos/aceptar.go:46` acota el cuerpo a 8 KiB y lo explica:
// «EL CUERPO SE ACOTA. Es un formulario de tres campos cortos». La solucion
// estaba escrita en el arbol, en una superficie hermana, y no habia nada que
// obligara a aplicarla aqui. Eso es lo que esta puerta pasa a hacer.
func TestNingunaRutaQueMutaAceptaUnCuerpoSinTope(t *testing.T) {
	// LAS RUTAS SE DERIVAN DE LOS PATRONES, no se escriben aqui: una lista al
	// lado del test es la que se queda vieja el dia que alguien anada la quinta.
	s := superficie(t, &fuente{c: campana(t, censoBase, nil)}, false)
	var mutantes []string
	for _, p := range s.Patrones() {
		metodo, ruta, hay := strings.Cut(p, " ")
		if !hay || metodo != http.MethodPost {
			continue
		}
		mutantes = append(mutantes, ruta)
	}
	if len(mutantes) < 4 {
		t.Fatalf("solo %d ruta(s) POST en la superficie (%v) y son cuatro: el derivador se "+
			"ha roto y esta puerta estaria mirando menos de lo que dice", len(mutantes), mutantes)
	}

	// SE MIDE LO QUE EL MANEJADOR LLEGA A LEER, no el codigo de respuesta.
	//
	// La primera version de esta puerta miraba si la respuesta era un 303, y era
	// floja: `excusar` y `cerrar` contestan con un aviso por motivos que no
	// tienen nada que ver con el tamano (una linea que no es ilegible, una
	// campana sin terminar), asi que salian verdes sin tener tope. Un verde por
	// el motivo equivocado es el que no se distingue de un verde de verdad.
	//
	// Contando bytes, la afirmacion es directa y no depende de la logica de cada
	// ruta: **con o sin tope, un manejador con `MaxBytesReader` no puede llegar a
	// leer mas alla del tope**.
	const grande = 4 << 20
	relleno := strings.Repeat("m", grande)

	for _, ruta := range mutantes {
		t.Run(ruta, func(t *testing.T) {
			form := url.Values{
				"veredicto": {"revocar"},
				"fila":      {"erp|u1|admin"},
				"desde":     {"2"},
				"motivo":    {relleno},
				"a":         {relleno},
			}
			leidos := pedirContando(t, s, http.MethodPost, ruta, form)
			// El techo del arnes va holgado sobre el tope real: lo que se afirma
			// es que EXISTE un tope y que es de un orden razonable, no cual es.
			// `cerrar` no lee ningun campo, asi que hoy sale con cero, y eso
			// tambien esta bien: lo que esta puerta impide es que el dia que
			// alguien le anada un campo, el cuerpo entre sin acotar.
			const techoDelArnes = 1 << 20
			if leidos > techoDelArnes {
				t.Errorf(`%s ha leido %d bytes de un cuerpo de %d. No hay tope.

  Lo que hay detras de esta ruta es un fichero APPEND-ONLY: lo que entre no se
  puede podar sin romper la cadena de hashes. Un mega de «motivo» no es disco
  desperdiciado, es un dato que no se puede quitar nunca.

  La superficie hermana ya lo tiene resuelto y lo explica, en
  superficies/documentos/aceptar.go: «EL CUERPO SE ACOTA. Es un formulario de
  tres campos cortos». Y dentro de esta misma superficie, abrir.go ya pone
  http.MaxBytesReader porque sube un CSV.

  Arreglo: http.MaxBytesReader en las CUATRO rutas que mutan, no en la que se
  acordo su autor.`, ruta, leidos, grande)
			}
		})
	}
}

// contador envuelve un cuerpo de peticion y apunta cuanto se llega a leer de el.
//
// Es lo que convierte esta puerta en una afirmacion sobre el TOPE y no sobre la
// logica de cada ruta: sin tope, ParseForm lee el cuerpo entero y el contador lo
// dice; con tope, la lectura se corta ahi, diga lo que diga la respuesta.
type contador struct {
	r      *strings.Reader
	leidos int
}

func (c *contador) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.leidos += n
	return n, err
}

// pedirContando manda un POST y devuelve cuantos bytes del cuerpo se leyeron.
func pedirContando(t *testing.T, s *Superficie, metodo, ruta string, form url.Values) int {
	t.Helper()
	cuerpo := &contador{r: strings.NewReader(form.Encode())}
	r := httptest.NewRequest(metodo, ruta, cuerpo)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// NewRequest con un io.Reader que no es *bytes.Reader deja ContentLength en
	// -1, que es lo que queremos: un cuerpo de longitud desconocida es el caso
	// que un tope tiene que cubrir, porque con `Content-Length` declarado hay
	// quien confia en el numero que manda el cliente.
	s.ServeHTTP(httptest.NewRecorder(), r)
	return cuerpo.leidos
}

// EL TEXTO QUE ENTRA AL REGISTRO TIENE TOPE, Y ESE TOPE NO ES EL DEL CUERPO.
//
// # Por que hacen falta las dos cosas
//
// Porque acotar el cuerpo a 8 KiB deja escribir 8 KiB de `motivo` en el fichero
// append-only, y eso sigue siendo prosa que nadie va a leer y que no se puede
// quitar. El tope del CUERPO defiende la memoria del proceso; el tope del CAMPO
// defiende el registro. Son dos fronteras y cada una es de su capa.
//
// Y el del campo vive en `nucleo/accesos` y no aqui a proposito: es una regla
// del dominio —cuanto puede medir el motivo de una decision— y tiene que valer
// para cualquiera que llame a `Registrar`, no solo para quien llegue por esta
// pantalla. La superficie no es la unica puerta del almacen.
func TestUnMotivoInterminableNoLlegaAlRegistro(t *testing.T) {
	f := &fuente{c: campana(t, censoBase, nil)}
	s := superficie(t, f, false)

	// CONTROL POSITIVO PRIMERO: un motivo normal entra y deja su entrada. Sin
	// esto, un manejador que rechazara todo pasaria el caso de abajo.
	w := pedir(t, s, http.MethodPost, "/uar/decidir", url.Values{
		"veredicto": {"revocar"}, "fila": {"erp|u1|admin"},
		"motivo": {"se va de la empresa el viernes"},
	})
	if w.Code != http.StatusSeeOther {
		t.Fatalf("una decision normal tiene que pasar y ha dado %d", w.Code)
	}
	if len(f.anotado) != 1 {
		t.Fatalf("una decision normal tiene que dejar UNA entrada y hay %d", len(f.anotado))
	}

	// LA ACUSACION: por debajo del tope del cuerpo y muy por encima de lo que
	// nadie escribe en un motivo.
	largo := strings.Repeat("x", 6<<10)
	w = pedir(t, s, http.MethodPost, "/uar/decidir", url.Values{
		"veredicto": {"revocar"}, "fila": {"erp|u2|lector"}, "motivo": {largo},
	})
	if w.Code == http.StatusSeeOther || len(f.anotado) != 1 {
		t.Errorf("un motivo de %d caracteres ha entrado al registro (codigo %d, %d entradas).\n"+
			"  Cabe de sobra en el tope del cuerpo, asi que acotar el cuerpo no basta: el "+
			"tope del CAMPO es una regla del dominio y vive en nucleo/accesos, para que "+
			"valga tambien para quien no venga por esta pantalla.",
			len(largo), w.Code, len(f.anotado))
	}
}
