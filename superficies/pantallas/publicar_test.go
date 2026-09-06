package pantallas

import (
	"strings"
	"testing"
)

// SU VALOR CERO ES NO PUBLICAR, Y HASTA HOY NADIE LO MIRABA.
//
// Lo encontro el barrido de `godoc_vigilado_test.go` el 06-09-2026: el godoc de
// `Publicaciones` decia, con estas palabras, que prometer una publicacion que no
// va a ocurrir es peor que no ofrecerla, porque el calendario se quedaria vacio
// y quien adopto creeria que ya esta hecho. El cable existia
// (`Publica: est.PuedeGuardar && s.publicar != nil`) y no habia ni un test que
// recorriera las dos ramas. Es el mismo hallazgo que M12 y por el mismo sitio:
// el peligro escrito, la guarda puesta, y nadie vigilando que siga puesta.
//
// Las dos direcciones, que es lo que hace que la puerta sirva: sin adaptador no
// se pinta el boton, y CON adaptador si. Sin la segunda, la puerta se aprobaria
// borrando el bloque entero de la plantilla.
func TestSinPublicadorLaPantallaNoOfreceLaPublicacion(t *testing.T) {
	const (
		boton  = `value="publicar"`
		clave  = "alcance.publicar.boton"
		nota   = "alcance.publicar.explica"
		quien  = "ana@ejemplo"
		alcanc = "/alcance?" + ParamVer + "=" + VerTodas
	)

	// SIN quien sepa publicar: hay sesion y hay almacen, o sea que ADOPTAR si
	// sale. Lo que no puede salir es la promesa de publicar.
	al := nuevoAlmacenFalso()
	s, cat := superficie(t, corpusDemo(), conGuardado(al, quien))
	_, cuerpo := pedir(t, s, alcanc)
	if strings.Contains(cuerpo, boton) {
		t.Errorf("sin adaptador de publicacion se pinta el boton igual:\n%s", cuerpo)
	}
	for _, k := range []string{clave, nota} {
		if cat.vistas()[k] > 0 {
			t.Errorf("sin adaptador de publicacion se pide %q, o sea que la promesa "+
				"esta en la pagina", k)
		}
	}
	// Control positivo de que este montaje SI llega hasta donde se decide el
	// boton. `.Publica` cuelga de `PuedeGuardar && publicar != nil`, asi que
	// hay que demostrar que la primera mitad esta en TRUE y que la unica razon
	// de que no salga es la segunda. Sin esto, un cambio que dejara de pintar
	// el bloque entero aprobaria la afirmacion de arriba sin recorrer nada.
	if cat.vistas()["alcance.derivacion.titulo"] == 0 {
		t.Fatal("este montaje no llega ni a la derivacion: la afirmacion de " +
			"arriba no ha recorrido nada")
	}
	if cat.vistas()["alcance.derivacion.no_guardado"] > 0 {
		t.Fatal("este montaje no puede guardar, asi que no dice nada sobre publicar: " +
			"la afirmacion de arriba se cumple por el motivo equivocado")
	}

	// CON quien sepa publicar: la promesa sale, y sale con su explicacion. Un
	// boton que publica el alcance de la INSTALACION sin decir que lo va a ver
	// todo el que entre es un dato cruzando de lo privado a lo publico en
	// silencio (invariante 12).
	al2 := nuevoAlmacenFalso()
	s2, cat2 := superficie(t, corpusDemo(), conPublicacion(al2, quien, &publicadorFalso{}))
	_, cuerpo2 := pedir(t, s2, alcanc)
	if !strings.Contains(cuerpo2, boton) {
		t.Errorf("con adaptador de publicacion NO se pinta el boton:\n%s", cuerpo2)
	}
	for _, k := range []string{clave, nota} {
		if cat2.vistas()[k] == 0 {
			t.Errorf("con adaptador de publicacion no se pide %q", k)
		}
	}
}
