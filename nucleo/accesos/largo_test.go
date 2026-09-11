package accesos

import (
	"errors"
	"strings"
	"testing"
)

// EL BORDE EXACTO DEL TOPE DE UN TEXTO, Y LAS DOS DIRECCIONES.
//
// # Por que hace falta aparte de la puerta de la superficie
//
// Porque la de la superficie demuestra que un motivo enorme no llega al registro
// POR ESA PANTALLA, y lo que se afirma aqui es mas fuerte: que no llega POR
// NINGUN SITIO. `Registrar` y `Excusar` son la frontera del dominio, y el proximo
// conector o la proxima orden de terminal entran por aqui y no por un formulario.
//
// # Y el borde va porque un `>=` donde toca un `>` deja fuera lo valido
//
// Un tope comprobado con el operador equivocado rechaza el texto que mide
// exactamente el maximo, y ese fallo es invisible mientras nadie escriba uno de
// esa longitud. Se comprueban los dos lados del borde, no «uno grande».
//
// Se cuenta en RUNAS y no en bytes a proposito: un motivo en castellano con
// enes y acentos mide mas en bytes que en caracteres, y un tope en bytes le
// daria menos sitio a quien escribe en castellano que a quien escribe en ingles.
func TestUnTextoEnElLimiteEntraYUnoMasNo(t *testing.T) {
	for _, c := range []struct {
		nombre  string
		largo   int
		entra   bool
		acentos bool
		porQue  string
	}{
		{
			nombre: "un motivo normal", largo: 40, entra: true,
			porQue: "es lo que escribe una persona de verdad, y si esto no entrara la " +
				"puerta seria imposible de satisfacer y la reaccion barata seria aflojarla",
		},
		{
			nombre: "justo el tope", largo: MaxLargoDeUnTexto, entra: true,
			porQue: "el maximo es el maximo: si se rechaza, el operador de comparacion esta " +
				"mal y nadie se entera hasta que alguien escriba uno asi",
		},
		{
			nombre: "uno mas que el tope", largo: MaxLargoDeUnTexto + 1, entra: false,
			porQue: "es el primer valor que no cabe, y es donde se ve si la guarda existe",
		},
		{
			nombre: "el tope, en caracteres con ene", largo: MaxLargoDeUnTexto, entra: true,
			acentos: true,
			porQue: "el tope se cuenta en RUNAS: en bytes, mil enes ocupan dos mil, y un " +
				"tope en bytes recortaria a quien escribe en castellano",
		},
		{
			nombre: "un motivo de un megabyte", largo: 1 << 20, entra: false,
			porQue: "es el caso que trajo todo esto: 1.048.789 bytes en un fichero " +
				"append-only que no se puede podar sin romper la cadena de hashes",
		},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			letra := "x"
			if c.acentos {
				letra = "ñ"
			}
			texto := strings.Repeat(letra, c.largo)

			cam := campana(t, censoBase, nil)
			err := cam.Registrar(Decision{
				Fila: "erp|u1|admin", Veredicto: Revocar, Quien: "ciso",
				Cuando: t1, Motivo: texto,
			})
			comprobar(t, err, c.entra, "Registrar", c.largo, c.porQue)

			// LA HERMANA, QUE ES LA QUE SE OLVIDA. `Excusar` escribe en el mismo
			// registro append-only y tiene su propio camino de validacion: una
			// guarda puesta solo en `Registrar` deja la puerta de al lado abierta,
			// que es justo la forma del hallazgo que trajo esto.
			cam2 := campana(t, censoBase, nil)
			err = cam2.Excusar(Excusa{Desde: 2, Hasta: 2, Quien: "ciso", Cuando: t1, Motivo: texto})
			// Excusar puede rechazar por OTRO motivo (la linea 2 si es legible),
			// asi que aqui solo se afirma la direccion que importa: si el texto
			// pasa del tope, tiene que fallar.
			if !c.entra && err == nil {
				t.Errorf("Excusar ha aceptado un motivo de %d caracteres.\n"+
					"  Por que importa: %s", c.largo, c.porQue)
			}
		})
	}
}

func comprobar(t *testing.T, err error, entra bool, quien string, largo int, porQue string) {
	t.Helper()
	if entra && err != nil {
		t.Errorf("%s: un motivo de %d caracteres tenia que entrar y ha dado: %v\n"+
			"  Por que importa: %s", quien, largo, err, porQue)
		return
	}
	if !entra {
		if err == nil {
			t.Errorf("%s: un motivo de %d caracteres ha entrado al registro.\n"+
				"  Por que importa: %s", quien, largo, porQue)
			return
		}
		if !errors.Is(err, ErrDecision) {
			t.Errorf("%s: el rechazo no lleva ErrDecision: %v.\n"+
				"  Sin centinela, quien llame no puede distinguir «esto no vale» de «ha "+
				"fallado el almacen», y las dos cosas piden reacciones distintas", quien, err)
		}
	}
}
