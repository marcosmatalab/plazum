package documentos

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// LO QUE SE ACEPTA TIENE QUE SER UNA PROPUESTA QUE EXISTA.
//
// # Por que este test existe, y lo dijo una mutacion
//
// La mutacion M6 quito la guarda y dejo que se guardara lo que llegara en el
// formulario. La suite se puso roja, si — pero por el CONTRATO DE CLAVES, porque
// `documentos.aceptar.no_casa` dejaba de pedirse. O sea que lo que la cazaba era
// «una clave del catalogo se ha quedado sin quien la pida», no «se ha escrito en
// el expediente algo que nadie propuso».
//
// Eso es un rojo por el motivo equivocado, y este repositorio ya sabe lo que
// cuesta: el dia que alguien anadiera otro estado que pidiera esa clave, la
// mutacion pasaria en verde. La propiedad necesita su propia puerta.
//
// # Que se afirma
//
// Un POST con un campo, un valor y una huella que NO casan con ninguna propuesta
// de esta cuenta se rechaza con su clave, y NO GUARDA NADA. Las dos mitades, y la
// segunda es la que importa: un rechazo que ademas escribe es peor que no
// rechazar, porque deja la pantalla diciendo que no y el expediente diciendo que
// si.
func TestNoSeAceptaUnValorQueNoCasaConNingunaPropuesta(t *testing.T) {
	a := nuevoAlmacenFalso()
	s := superficie(t, conAlmacen(a))
	if codigo, cuerpo := subir(t, s, "p.pdf", []byte("un parrafo")); codigo != http.StatusSeeOther {
		t.Fatalf("la subida contesta %d: %s", codigo, avisoDe(cuerpo))
	}
	fs, err := a.Ficha(context.Background(), "ciso@ejemplo")
	if err != nil || len(fs) != 1 {
		t.Fatalf("el doble propone %d campos y tenia que proponer uno (err %v)", len(fs), err)
	}
	buena := fs[0]

	// LAS TRES FORMAS DE NO CASAR, una por campo del emparejamiento. Cada una
	// sola, para que el rechazo no pueda venir de otra: si solo se probara
	// cambiando los tres a la vez, un `if` que mirara UNO bastaria para pasar.
	for _, c := range []struct {
		que             string
		campo, val, hue string
	}{
		{"otro campo", "documentos.ficha.campo.firmante", buena.Valor, buena.Huella},
		{"otro valor", buena.Campo, "1999-01-01", buena.Huella},
		{"otra huella", buena.Campo, buena.Valor, strings.Repeat("f", 64)},
	} {
		t.Run(c.que, func(t *testing.T) {
			codigo, cuerpo := aceptar(t, s, c.campo, c.val, c.hue)
			if codigo != http.StatusUnprocessableEntity {
				t.Errorf("contesta %d y tenia que rechazar con 422", codigo)
			}
			if got := avisoDe(cuerpo); got != "documentos.aceptar.no_casa" {
				t.Errorf("el rechazo es %q y tenia que ser documentos.aceptar.no_casa.\n"+
					"  Un dato que llega, se entiende y no casa con ninguna propuesta NO es "+
					"una ausencia: es la tercera forma de la nada, y tiene su propia "+
					"respuesta", got)
			}
			// Y LO QUE IMPORTA: NO SE HA GUARDADO NADA.
			ac, err := a.Aceptados(context.Background(), "ciso@ejemplo")
			if err != nil {
				t.Fatal(err)
			}
			if len(ac) != 0 {
				t.Errorf("tras rechazar hay %d campos aceptados: %+v\n"+
					"  Un rechazo que ademas escribe es peor que no rechazar, porque deja "+
					"la pantalla diciendo que no y el expediente diciendo que si", len(ac), ac)
			}
		})
	}

	// EL CONTROL POSITIVO: la propuesta BUENA si se acepta, y queda con el nombre
	// de quien la acepto. Sin esta mitad, un handler que rechazara todo pasaria
	// lo de arriba entero.
	if codigo, cuerpo := aceptar(t, s, buena.Campo, buena.Valor, buena.Huella); codigo != http.StatusSeeOther {
		t.Fatalf("la propuesta buena contesta %d y tenia que redirigir con 303: %s",
			codigo, avisoDe(cuerpo))
	}
	ac, err := a.Aceptados(context.Background(), "ciso@ejemplo")
	if err != nil {
		t.Fatal(err)
	}
	if len(ac) != 1 {
		t.Fatalf("tras aceptar la buena hay %d campos aceptados", len(ac))
	}
	if ac[0].Quien != "ciso@ejemplo" {
		t.Errorf("lo aceptado dice que lo acepto %q y lo acepto ciso@ejemplo.\n"+
			"  Sin el nombre, lo aceptado es indistinguible de lo propuesto", ac[0].Quien)
	}
}

// SIN SESION NO SE ACEPTA NADA, y tampoco se cuenta que haya algo que aceptar.
//
// Pesa mas aqui que en la subida: lo que se guardaria lleva el nombre de quien
// lo acepta dentro, asi que un POST sin sesion escribiria una afirmacion firmada
// por nadie.
func TestSinSesionNoSeAceptaNingunCampoDeLaFicha(t *testing.T) {
	a := nuevoAlmacenFalso()
	s := superficie(t, conAlmacen(a))
	if codigo, _ := subir(t, s, "p.pdf", []byte("un parrafo")); codigo != http.StatusSeeOther {
		t.Fatal("la subida de partida no ha ido")
	}
	fs, _ := a.Ficha(context.Background(), "ciso@ejemplo")
	if len(fs) != 1 {
		t.Fatal("sin propuesta de partida este test no demuestra nada")
	}

	sinSes := superficie(t, conAlmacen(a), sinSesion)
	codigo, cuerpo := aceptar(t, sinSes, fs[0].Campo, fs[0].Valor, fs[0].Huella)
	if codigo != http.StatusUnauthorized {
		t.Errorf("aceptar sin sesion contesta %d y tenia que contestar 401", codigo)
	}
	if strings.Contains(cuerpo, fs[0].Valor) {
		t.Errorf("el 401 devuelve el valor propuesto (%q), o sea que cuenta que hay algo "+
			"detras", fs[0].Valor)
	}
	// Y NO SE HA ESCRITO NADA CON EL NOMBRE DE NADIE.
	for _, quien := range []string{"", "ciso@ejemplo"} {
		if ac, _ := a.Aceptados(context.Background(), quien); len(ac) != 0 {
			t.Errorf("tras un POST sin sesion hay %d campos aceptados para %q", len(ac), quien)
		}
	}
}

// SIN TOKEN NO SE PINTA EL BOTON DE ACEPTAR.
//
// Es la misma pareja que el formulario de subir: un boton sin token contesta 403
// y nadie sabe por que. Aqui ademas la propuesta se sigue viendo, con su parrafo:
// lo que se pierde es poder confirmarla, no poder leerla.
func TestSinTokenSeVeLaPropuestaYNoElBotonDeAceptar(t *testing.T) {
	a := nuevoAlmacenFalso()
	con := superficie(t, conAlmacen(a))
	if codigo, _ := subir(t, con, "p.pdf", []byte("un parrafo")); codigo != http.StatusSeeOther {
		t.Fatal("la subida de partida no ha ido")
	}
	// CONTROL POSITIVO: con token, el formulario esta.
	if _, cuerpo := ver(t, con); !strings.Contains(cuerpo, `action="`+BasePorDefecto+RutaDeAceptar+`"`) {
		t.Fatalf("con token no sale el formulario de aceptar:\n%s", cuerpo)
	}
	sin := superficie(t, conAlmacen(a), sinToken)
	_, cuerpo := ver(t, sin)
	if strings.Contains(cuerpo, `action="`+BasePorDefecto+RutaDeAceptar+`"`) {
		t.Error("sin token se pinta el formulario de aceptar, o sea un boton que contesta 403")
	}
	// Y LA PROPUESTA SIGUE VIENDOSE: lo que se pierde es confirmarla.
	if !strings.Contains(cuerpo, "2026-01-15") {
		t.Errorf("sin token tampoco se ve la propuesta, y eso es esconder de mas:\n%s", cuerpo)
	}
}
