package ventana

import (
	"errors"
	"strings"
	"testing"
)

// EL DEFAULT PERMISIVO DE String, que es el invariante 8 en tres lineas.
//
// HALLAZGO, medido: el switch de EstadoVenc.String no tenia rama para
// Determinado, asi que su default devolvia "determinado" para CUALQUIER valor.
// Determinado es el estado MAS FUERTE de los tres (es el unico que autoriza a
// fiarse de Vence), o sea que un valor fuera de rango se disfrazaba del bueno.
// Y no es teoria: el expediente compara estados POR TEXTO
// (nucleo/expediente/expediente.go, `d.Estado != v.Estado.String()`), y ahora
// tambien el ejecutor de dorados.
//
// La mutacion que lo demuestra: devolver "determinado" en el ultimo return
// pone este test rojo.
func TestUnEstadoFueraDeRangoNoSeDisfrazaDeDeterminado(t *testing.T) {
	raro := EstadoVenc(200)
	got := raro.String()
	if got == Determinado.String() {
		t.Fatalf("HALLAZGO: EstadoVenc(200) se imprime como %q, que es el estado que dice "+
			"\"hay fecha y hora exactas\". Quien compare estados por texto (el expediente y "+
			"el ejecutor de dorados lo hacen) leeria que hay fecha donde el motor no sabe "+
			"que hay", got)
	}
	if !strings.Contains(got, "200") {
		t.Errorf("un estado desconocido tiene que decir CUAL es, para poder rastrearlo: %q", got)
	}
	// Control negativo: los tres que existen siguen imprimiendose por su nombre.
	for e, quiero := range map[EstadoVenc]string{
		Determinado:      "determinado",
		PendienteDeHecho: "pendiente de hecho",
		SinPlazoLegal:    "sin plazo legal",
	} {
		if e.String() != quiero {
			t.Errorf("%d se imprime %q y tenia que ser %q", e, e.String(), quiero)
		}
	}
}

// ParseEstadoVenc es el inverso EXACTO de String, y las dos leen la misma tabla.
// Si un dia se anade un estado y solo se toca una direccion, esto se pone rojo.
func TestElNombreDeUnEstadoSeLeeYSeEscribeConLaMismaTabla(t *testing.T) {
	for i := range nombresDeEstado {
		e := EstadoVenc(i) // #nosec G115 -- i indexa la propia tabla
		leido, err := ParseEstadoVenc(e.String())
		if err != nil {
			t.Fatalf("%q lo imprime String y ParseEstadoVenc no lo lee: %v", e.String(), err)
		}
		if leido != e {
			t.Fatalf("%q se lee como %d y se imprimio desde %d", e.String(), leido, e)
		}
	}
	// LA CADENA VACIA NO ES UN ESTADO. Sin esto caeria en el valor cero
	// (Determinado) por la puerta de atras, que es la permisividad que String
	// acaba de perder: quien parsee un campo ausente se llevaria el estado mas
	// fuerte sin haberlo pedido.
	if _, err := ParseEstadoVenc(""); !errors.Is(err, ErrEstadoDesconocido) {
		t.Errorf("HALLAZGO: la cadena vacia se lee como un estado (%v)", err)
	}
	for _, malo := range []string{"determinao", "pendiente", "sin plazo", "DETERMINADO",
		"estado desconocido (200)"} {
		if _, err := ParseEstadoVenc(malo); !errors.Is(err, ErrEstadoDesconocido) {
			t.Errorf("HALLAZGO: %q se acepta como estado del motor", malo)
		}
	}
	// Y el error dice cuales son, que es lo que necesita quien escribe un caso.
	_, err := ParseEstadoVenc("determinao")
	for _, n := range nombresDeEstado {
		if !strings.Contains(err.Error(), n) {
			t.Errorf("el error no enumera %q: %v", n, err)
		}
	}
}
