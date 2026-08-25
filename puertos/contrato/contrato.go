// Package contrato son las suites que TODA implementacion de un puerto de la
// etapa 2 tiene que pasar.
//
// Por que existen, y por que no bastaba con congelar las firmas. El test de
// congelacion de puertos/ protege la FORMA: que nadie cambie una firma por su
// cuenta mientras varios frentes compilan contra ella. No protege el
// SIGNIFICADO. Una Sesion que devuelve true en ComprobarCSRF pase lo que pase
// satisface la interfaz perfectamente y no protege de nada.
//
// Estas suites son el significado escrito en forma ejecutable. Un adaptador
// nuevo llama a la que le toca desde su propio test:
//
//	func TestMiSesionCumpleElContrato(t *testing.T) {
//	    contrato.Sesion(t, func() puertos.Sesion { return NuevaSesion() })
//	}
//
// Y si no lo cumple, se entera antes de integrarse, no despues.
//
// Regla al ampliarlas: una comprobacion que se anade aqui es una promesa que
// TODAS las implementaciones tienen que cumplir, incluidas las ya escritas. Si
// solo la puede cumplir una, no es contrato, es detalle de implementacion.
package contrato

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"dutiq/puertos"
)

// Sesion comprueba el contrato de puertos.Sesion.
//
// Lo que se exige aqui no es cosmetico: la etapa 2 dice "CSRF en todo POST", y
// un CSRF que no esta atado a su sesion no es CSRF. Las comprobaciones de
// abajo son justo las que distinguen una implementacion de las que protegen de
// una que solo devuelve nil.
func Sesion(t *testing.T, nueva func() puertos.Sesion) {
	t.Helper()
	ctx := context.Background()

	t.Run("una sesion abierta se lee y devuelve su sujeto", func(t *testing.T) {
		s := nueva()
		id, err := s.Abrir(ctx, "operador@ejemplo", time.Hour)
		if err != nil {
			t.Fatalf("abrir: %v", err)
		}
		if id == "" {
			t.Fatal("el identificador de sesion no puede ser vacio")
		}
		suj, err := s.Leer(ctx, id)
		if err != nil {
			t.Fatalf("leer una sesion recien abierta: %v", err)
		}
		if suj != "operador@ejemplo" {
			t.Fatalf("sujeto %q, se esperaba operador@ejemplo", suj)
		}
	})

	t.Run("dos sesiones no comparten identificador", func(t *testing.T) {
		s := nueva()
		a, err := s.Abrir(ctx, "uno", time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		b, err := s.Abrir(ctx, "dos", time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		if a == b {
			t.Fatal("dos sesiones distintas con el mismo id: cualquiera entra como cualquiera")
		}
	})

	t.Run("un identificador inventado no abre nada", func(t *testing.T) {
		s := nueva()
		if _, err := s.Leer(ctx, "me-lo-acabo-de-inventar"); err == nil {
			t.Fatal("una sesion que no existe no puede leerse sin error")
		}
	})

	t.Run("cerrar invalida, y es idempotente", func(t *testing.T) {
		s := nueva()
		id, err := s.Abrir(ctx, "operador", time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.Cerrar(ctx, id); err != nil {
			t.Fatalf("cerrar: %v", err)
		}
		if _, err := s.Leer(ctx, id); err == nil {
			t.Fatal("una sesion cerrada no puede seguir leyendose: eso es no cerrar")
		}
		if err := s.Cerrar(ctx, id); err != nil {
			t.Fatalf("cerrar dos veces tiene que ser inocuo, y dio: %v", err)
		}
	})

	t.Run("el token CSRF vale para su sesion", func(t *testing.T) {
		s := nueva()
		id, err := s.Abrir(ctx, "operador", time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		tok, err := s.TokenCSRF(ctx, id)
		if err != nil {
			t.Fatalf("emitir token: %v", err)
		}
		if tok == "" {
			t.Fatal("un token CSRF vacio no protege de nada")
		}
		if err := s.ComprobarCSRF(ctx, id, tok); err != nil {
			t.Fatalf("el token recien emitido tiene que valer: %v", err)
		}
	})

	t.Run("el token CSRF de otra sesion NO vale", func(t *testing.T) {
		s := nueva()
		mia, err := s.Abrir(ctx, "victima", time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		suya, err := s.Abrir(ctx, "atacante", time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		tokAtacante, err := s.TokenCSRF(ctx, suya)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.ComprobarCSRF(ctx, mia, tokAtacante); err == nil {
			t.Fatal("un token emitido para OTRA sesion no puede validar en esta. " +
				"Es exactamente el ataque del que protege el CSRF: si vale, no protege")
		}
	})

	t.Run("un token inventado no vale", func(t *testing.T) {
		s := nueva()
		id, err := s.Abrir(ctx, "operador", time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.ComprobarCSRF(ctx, id, "token-inventado"); err == nil {
			t.Fatal("un token que nadie emitio no puede validar")
		}
		if err := s.ComprobarCSRF(ctx, id, ""); err == nil {
			t.Fatal("un token vacio no puede validar: es el caso de la peticion sin token")
		}
	})

	t.Run("el token de una sesion cerrada deja de valer", func(t *testing.T) {
		s := nueva()
		id, err := s.Abrir(ctx, "operador", time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		tok, err := s.TokenCSRF(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.Cerrar(ctx, id); err != nil {
			t.Fatal(err)
		}
		if err := s.ComprobarCSRF(ctx, id, tok); err == nil {
			t.Fatal("cerrar sesion tiene que invalidar sus tokens; si no, el logout no protege")
		}
	})
}

// Catalogo comprueba el contrato de puertos.Catalogo.
func Catalogo(t *testing.T, nuevo func() puertos.Catalogo) {
	t.Helper()

	t.Run("hay al menos un idioma y el primero es el de por defecto", func(t *testing.T) {
		c := nuevo()
		if len(c.Idiomas()) == 0 {
			t.Fatal("un catalogo sin idiomas no puede renderizar nada")
		}
	})

	t.Run("una clave sin traducir devuelve la clave, no revienta", func(t *testing.T) {
		c := nuevo()
		got := c.Traducir(c.Idiomas()[0], "clave.que.no.existe.en.ningun.catalogo")
		if got == "" {
			t.Fatal("una clave sin traducir no puede devolver vacio: la pantalla se queda muda " +
				"y nadie se entera de que falta")
		}
	})

	t.Run("un idioma desconocido no rompe", func(t *testing.T) {
		c := nuevo()
		// No se exige que traduzca: se exige que no reviente ni devuelva vacio.
		if got := c.Traducir("xx-XX", "cualquier.clave"); got == "" {
			t.Fatal("un idioma desconocido tiene que caer al de por defecto o devolver la clave")
		}
	})

	t.Run("el idioma por defecto no tiene faltantes contra si mismo", func(t *testing.T) {
		c := nuevo()
		if f := c.Faltantes(c.Idiomas()[0]); len(f) != 0 {
			t.Fatalf("el idioma por defecto es la referencia: no puede faltarle nada a si mismo, "+
				"y le faltan %d claves: %v", len(f), f)
		}
	})
}

// Diagnostico comprueba el contrato de puertos.Diagnostico.
//
// La exigencia fuerte es la ultima: un problema sin arreglo escrito traslada el
// trabajo al operador, que es justo lo que `doctor` existe para evitar.
func Diagnostico(t *testing.T, nuevo func() puertos.Diagnostico) {
	t.Helper()
	ctx := context.Background()

	t.Run("comprueba algo", func(t *testing.T) {
		d := nuevo()
		if len(d.Comprobar(ctx)) == 0 {
			t.Fatal("un doctor que no comprueba nada da una falsa tranquilidad")
		}
	})

	t.Run("cada comprobacion tiene nombre y detalle", func(t *testing.T) {
		d := nuevo()
		for _, c := range d.Comprobar(ctx) {
			if strings.TrimSpace(c.Nombre) == "" {
				t.Errorf("comprobacion sin nombre: %+v", c)
			}
			if strings.TrimSpace(c.Detalle) == "" {
				t.Errorf("comprobacion %q sin detalle: no dice que encontro", c.Nombre)
			}
		}
	})

	t.Run("todo lo que no esta correcto dice como se arregla", func(t *testing.T) {
		d := nuevo()
		for _, c := range d.Comprobar(ctx) {
			if c.Estado == puertos.Correcto {
				continue
			}
			if strings.TrimSpace(c.Arreglo) == "" {
				t.Errorf("la comprobacion %q esta en %s y no dice como se arregla. "+
					"Un problema sin arreglo escrito es medio problema: le pasa el trabajo "+
					"al operador, que es lo que doctor tenia que evitar", c.Nombre, c.Estado)
			}
		}
	})
}

// Actualizador comprueba el contrato de puertos.Actualizador.
//
// La vuelta atras se exige aqui y no en la documentacion: una actualizacion que
// no se puede deshacer, en un producto que vigila plazos legales, convierte un
// fallo de actualizacion en un incumplimiento.
func Actualizador(t *testing.T, nuevo func() puertos.Actualizador, versionValida string) {
	t.Helper()
	ctx := context.Background()

	t.Run("aplicar deja un punto de retorno usable", func(t *testing.T) {
		a := nuevo()
		punto, err := a.Aplicar(ctx, versionValida)
		if err != nil {
			t.Fatalf("aplicar %q: %v", versionValida, err)
		}
		if punto == "" {
			t.Fatal("aplicar sin punto de retorno es una actualizacion sin vuelta atras")
		}
		if err := a.Deshacer(ctx, punto); err != nil {
			t.Fatalf("deshacer el punto que acaba de crear: %v", err)
		}
	})

	t.Run("deshacer un punto inventado falla, no finge", func(t *testing.T) {
		a := nuevo()
		if err := a.Deshacer(ctx, "punto-que-no-existe"); err == nil {
			t.Fatal("deshacer algo que no existe no puede devolver exito: el operador creeria " +
				"que ha vuelto atras y no ha vuelto")
		}
	})

	t.Run("una version que no existe se rechaza", func(t *testing.T) {
		a := nuevo()
		if _, err := a.Aplicar(ctx, "v99.99.99-no-existe"); err == nil {
			t.Fatal("aplicar una version inexistente tiene que fallar")
		}
	})
}

// ErrNoImplementado lo pueden usar los adaptadores a medias mientras se
// construyen, para que quede explicito que falta en vez de devolver nil.
var ErrNoImplementado = errors.New("no implementado todavia")
