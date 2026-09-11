package usuarios_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/usuarios"
)

// UN SECRETO QUE `ComprobarSecreto` RECHAZA NO ABRE SESION. NUNCA.
//
// # El agujero que cierra, y por que es P0
//
// `Autenticar` acotaba el secreto con un RECORTE antes de derivar:
//
//	if len(secreto) > LongitudMaximaDelSecreto {
//	    secreto = secreto[:LongitudMaximaDelSecreto]   // <- aqui
//	}
//
// y `ComprobarSecreto`, que es la que gobierna la creacion, RECHAZA lo que pasa
// de ese tope. Las dos mitades juntas dicen esto: con la contrasena mas larga
// que se puede llegar a crear —exactamente `LongitudMaximaDelSecreto` bytes—,
// **la contrasena mas CUALQUIER sufijo abre sesion**, porque el sufijo se cae
// por el recorte antes de derivar.
//
// # Por que el recorte esta mal aunque el tope este bien
//
// El tope esta bien y su motivo tambien: derivar diez megabytes es trabajo que
// paga el servidor y elige quien ataca. Para eso basta con **rechazar**.
// Recortar hace otra cosa: convierte «dato presente que no cumple» en «dato
// valido», que es la TERCERA forma de la nada del invariante 8 —presente y no
// interpretable— en el unico sitio del producto donde el valor por defecto
// **abre una sesion**.
//
// # POR QUE LA PUERTA VA SOBRE LAS DOS FUNCIONES Y NO SOBRE EL RECORTE
//
// Porque prohibir el recorte es prohibir una linea, y lo que hay que afirmar es
// la relacion: **lo que una mitad rechaza, la otra no lo acepta**. Escrita sobre
// la linea, la puerta se quedaria vieja el dia que el acotado cambie de forma;
// escrita sobre la relacion, sigue valiendo. Y ademas cubre el caso que nadie
// mira, que es el de al lado: que el secreto de longitud EXACTA siga entrando.
func TestUnSecretoQueElAltaRechazaNoAutenticaNunca(t *testing.T) {
	ruta := rutaNueva(t)
	a := abrirEn(t, ruta)

	// El secreto mas largo que se puede llegar a crear: justo el tope.
	enElTope := strings.Repeat("z", usuarios.LongitudMaximaDelSecreto)
	if err := usuarios.ComprobarSecreto(enElTope); err != nil {
		t.Fatalf("el secreto de longitud exacta tiene que poder crearse, y no: %v.\n"+
			"  Si esto falla, el caso de abajo no prueba nada porque no hay cuenta que atacar", err)
	}
	if err := a.CrearPrimerAdministrador(context.Background(), usuarioBueno, enElTope); err != nil {
		t.Fatal(err)
	}

	// CONTROL POSITIVO: con el secreto entero se entra. Sin esto, un `Autenticar`
	// que fallara siempre pasaria el caso de abajo, y eso no se distingue desde
	// fuera de una puerta que funciona.
	if _, err := a.Autenticar(context.Background(), usuarioBueno, enElTope); err != nil {
		t.Fatalf("con el secreto correcto no se ha podido entrar: %v", err)
	}

	// LA ACUSACION. Cada sufijo produce un secreto que `ComprobarSecreto`
	// RECHAZA, asi que ninguno puede ser la contrasena de nadie.
	for _, sufijo := range []string{
		"x",
		"-lo-que-sea",
		strings.Repeat("y", 4096),
	} {
		conSufijo := enElTope + sufijo
		if err := usuarios.ComprobarSecreto(conSufijo); !errors.Is(err, usuarios.ErrSecretoNoValido) {
			t.Fatalf("el arnes esta mal: %d bytes tendrian que ser rechazados por "+
				"ComprobarSecreto y no lo son", len(conSufijo))
		}
		if _, err := a.Autenticar(context.Background(), usuarioBueno, conSufijo); err == nil {
			t.Errorf(`un secreto de %d bytes ha ABIERTO SESION, y el alta lo rechaza.

  La contrasena de esta cuenta tiene %d bytes, que es el tope. Con un recorte
  antes de derivar, la contrasena mas cualquier sufijo entra: el sufijo se cae y
  lo que se deriva es la contrasena buena.

  El tope existe para no derivar diez megabytes, y para eso basta con RECHAZAR.
  Recortar convierte «dato presente que no cumple» en «dato valido», que es la
  tercera forma de la nada del invariante 8 en el unico sitio del producto donde
  el valor por defecto abre una sesion.

  Arreglo: en Autenticar, un secreto por encima del tope se rechaza sin derivar,
  y se devuelve ErrCredenciales como cualquier otro fallo de credenciales.`,
				len(conSufijo), len(enElTope))
		}
	}
}

// EL OTRO LADO DE LA MISMA FRONTERA: rechazar no puede costar menos que fallar.
//
// # Por que hace falta este segundo test
//
// Porque el arreglo obvio del de arriba —salir antes de derivar cuando el
// secreto es demasiado largo— reabre por la puerta de atras lo que el godoc de
// `Autenticar` promete con todas sus letras: que el fallo **no dice nada**, ni
// por el mensaje ni por el reloj. Un camino que contesta sin derivar contesta en
// microsegundos donde el normal tarda un cuarto de segundo, y eso se mide desde
// fuera sin ninguna herramienta especial.
//
// Aqui no se mide tiempo, que seria una puerta con rojos aleatorios en una
// maquina cargada. Se afirma lo unico que se puede afirmar mecanicamente y es lo
// que de verdad importa: que el error es **el mismo centinela**, asi que quien
// lo reciba no aprende nada. Lo del reloj lo sostiene el codigo derivando
// igualmente, y eso se dice en el godoc de `Autenticar`.
func TestElSecretoDemasiadoLargoFallaComoCualquierOtraCredencial(t *testing.T) {
	a := abrirEn(t, rutaNueva(t))
	if err := a.CrearPrimerAdministrador(context.Background(), usuarioBueno, secretoBueno); err != nil {
		t.Fatal(err)
	}

	demasiado := strings.Repeat("z", usuarios.LongitudMaximaDelSecreto+1)
	_, errLargo := a.Autenticar(context.Background(), usuarioBueno, demasiado)
	_, errMalo := a.Autenticar(context.Background(), usuarioBueno, "esto-no-es-la-contrasena")
	_, errNadie := a.Autenticar(context.Background(), "no-existe-esta-cuenta", secretoBueno)

	for _, c := range []struct {
		nombre string
		err    error
	}{
		{"secreto por encima del tope", errLargo},
		{"secreto que no casa", errMalo},
		{"usuario que no existe", errNadie},
	} {
		if !errors.Is(c.err, usuarios.ErrCredenciales) {
			t.Errorf("%s: el error es %v y tenia que ser ErrCredenciales.\n"+
				"  Los tres fallos contestan lo mismo a proposito: si el de la longitud "+
				"tuviera centinela propio, quien lo reciba sabria que su contrasena se "+
				"parece a una valida y solo le sobra tamano, que es informacion que no "+
				"tiene por que tener.", c.nombre, c.err)
		}
	}
}
