package usuarios_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/adaptadores/usuarios"
)

// El almacen de usuarios es la unica pieza del producto que guarda una
// credencial, asi que sus tests son de propiedades y no de casos: que el secreto
// no se pueda recuperar, que la nada no se confunda con lo roto, que dos
// peticiones no creen dos administradores, y que el error no diga si el usuario
// existe.

const (
	usuarioBueno = "ciso"
	secretoBueno = "contrasena-de-prueba-1"
)

func abrirEn(t *testing.T, ruta string) *usuarios.Almacen {
	t.Helper()
	a, err := usuarios.Abrir(usuarios.Opciones{Ruta: ruta})
	if err != nil {
		t.Fatalf("abriendo el almacen: %v", err)
	}
	return a
}

func rutaNueva(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), usuarios.NombreDelFichero)
}

// El camino entero: no hay nadie, se crea el administrador, entra, y sigue
// entrando despues de parar y volver a arrancar.
//
// La ULTIMA parte es la que importa: un almacen que solo vive en memoria deja al
// operador con la ventana de instalacion abierta en cada arranque.
func TestElAdministradorSobreviveAlReinicio(t *testing.T) {
	ruta := rutaNueva(t)
	ctx := context.Background()

	a := abrirEn(t, ruta)
	if hay, err := a.HayAdministrador(ctx); err != nil || hay {
		t.Fatalf("un almacen nuevo dice que ya hay administrador (hay=%v, err=%v)", hay, err)
	}
	if err := a.CrearPrimerAdministrador(ctx, usuarioBueno, secretoBueno); err != nil {
		t.Fatalf("creando el primer administrador: %v", err)
	}
	if hay, err := a.HayAdministrador(ctx); err != nil || !hay {
		t.Fatalf("creado el administrador, el almacen dice que no lo hay (hay=%v, err=%v)", hay, err)
	}

	// El proceso se para y vuelve: se lee del disco lo que se escribio.
	b := abrirEn(t, ruta)
	if hay, err := b.HayAdministrador(ctx); err != nil || !hay {
		t.Fatalf("tras reabrir, el almacen no ve al administrador (hay=%v, err=%v).\n"+
			"  Sin persistencia, cada arranque reabriria la ventana de instalacion", hay, err)
	}
	sujeto, err := b.Autenticar(ctx, usuarioBueno, secretoBueno)
	if err != nil || sujeto != usuarioBueno {
		t.Fatalf("tras reabrir, las credenciales buenas no entran: sujeto=%q err=%v", sujeto, err)
	}
}

// EL SECRETO NO ESTA EN EL FICHERO. Se busca la contrasena entera y tambien
// cachos suyos, porque una codificacion tonta (base64, hex) la dejaria dentro
// sin que la cadena literal apareciera.
func TestElSecretoNoSeGuardaEnClaroNiSePuedeRecuperar(t *testing.T) {
	ruta := rutaNueva(t)
	a := abrirEn(t, ruta)
	if err := a.CrearPrimerAdministrador(context.Background(), usuarioBueno, secretoBueno); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(ruta) // #nosec G304 -- ruta del propio test
	if err != nil {
		t.Fatal(err)
	}
	fichero := string(b)
	for _, aguja := range []string{
		secretoBueno,
		"contrasena-de-prueba",
		"Y29udHJhc2VuYS1kZS1wcnVlYmEtMQ==",         // base64
		"636f6e74726173656e612d64652d707275656261", // hex
	} {
		if strings.Contains(fichero, aguja) {
			t.Errorf("el fichero de cuentas contiene %q. La contrasena tiene que quedar "+
				"derivada y no recuperable:\n%s", aguja, fichero)
		}
	}
	// Y el tipo no tiene ningun metodo que la devuelva: lo unico que se puede
	// preguntar es cuantas cuentas hay.
	if a.Cuentas() != 1 {
		t.Errorf("Cuentas() dice %d y hay una", a.Cuentas())
	}
}

// LAS TRES FORMAS DE LA NADA, y solo una de ellas es la nada (invariante 8).
//
// EL CONTROL POSITIVO ES EL PRIMER CASO: un fichero ausente TIENE que abrir sin
// error y sin administrador. Sin ese caso, un Abrir que fallara siempre pasaria
// todos los demas y dejaria el producto sin poder instalarse.
func TestLasTresFormasDeLaNadaNoSonLaMisma(t *testing.T) {
	t.Run("ausente es la nada de verdad", func(t *testing.T) {
		a, err := usuarios.Abrir(usuarios.Opciones{Ruta: rutaNueva(t)})
		if err != nil {
			t.Fatalf("un fichero que no existe es una instalacion nueva y tiene que abrir "+
				"sin error: %v", err)
		}
		if hay, _ := a.HayAdministrador(context.Background()); hay {
			t.Error("un almacen sin fichero no puede decir que hay administrador")
		}
	})

	t.Run("vacio presente es un error, no la nada", func(t *testing.T) {
		ruta := rutaNueva(t)
		if err := os.WriteFile(ruta, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := usuarios.Abrir(usuarios.Opciones{Ruta: ruta})
		if !errors.Is(err, usuarios.ErrAlmacenVacio) {
			t.Fatalf("un fichero de cuentas de cero bytes se ha leido como «aqui no hay "+
				"nadie» (err=%v).\n"+
				"  Eso REABRE la ventana del primer administrador en un sistema ya "+
				"instalado: plazum volveria a imprimir un token y quien llegue antes se "+
				"queda con la instalacion.", err)
		}
	})

	t.Run("presente y no interpretable es un error, nunca el valor por defecto", func(t *testing.T) {
		casos := []struct {
			nombre    string
			contenido string
			centinela error
		}{
			{"no es JSON", "esto no es json", usuarios.ErrAlmacenIlegible},
			{"solo espacios", "   \n\t\n  ", usuarios.ErrAlmacenVacio},
			{"sin version", `{"cuentas":[]}`, usuarios.ErrAlmacenSinVersion},
			{"version futura", `{"version":99,"cuentas":[]}`, usuarios.ErrVersionDesconocida},
			{"cuenta sin algoritmo", `{"version":1,"cuentas":[{"usuario":"a",` +
				`"iteraciones":600000,"sal":"` + strings.Repeat("00", 16) + `","clave":"` +
				strings.Repeat("00", 32) + `","creado":"2026-09-03T09:00:00Z"}]}`,
				usuarios.ErrAlmacenIlegible},
			{"cuenta sin iteraciones", `{"version":1,"cuentas":[{"usuario":"a",` +
				`"algoritmo":"` + usuarios.Algoritmo + `","sal":"` + strings.Repeat("00", 16) +
				`","clave":"` + strings.Repeat("00", 32) + `","creado":"2026-09-03T09:00:00Z"}]}`,
				usuarios.ErrAlmacenIlegible},
			{"coste por debajo del suelo", `{"version":1,"cuentas":[{"usuario":"a",` +
				`"algoritmo":"` + usuarios.Algoritmo + `","iteraciones":1000,"sal":"` +
				strings.Repeat("00", 16) + `","clave":"` + strings.Repeat("00", 32) +
				`","creado":"2026-09-03T09:00:00Z"}]}`, usuarios.ErrCosteInsuficiente},
			{"sal que no es hexadecimal", `{"version":1,"cuentas":[{"usuario":"a",` +
				`"algoritmo":"` + usuarios.Algoritmo + `","iteraciones":600000,"sal":"zz",` +
				`"clave":"` + strings.Repeat("00", 32) + `","creado":"2026-09-03T09:00:00Z"}]}`,
				usuarios.ErrAlmacenIlegible},
			{"instante de creacion que no se entiende", `{"version":1,"cuentas":[{"usuario":"a",` +
				`"algoritmo":"` + usuarios.Algoritmo + `","iteraciones":600000,"sal":"` +
				strings.Repeat("00", 16) + `","clave":"` + strings.Repeat("00", 32) +
				`","creado":"ayer"}]}`, usuarios.ErrAlmacenIlegible},
			{"dos cuentas con el mismo nombre", `{"version":1,"cuentas":[` +
				cuentaJSON("a") + `,` + cuentaJSON("a") + `]}`, usuarios.ErrAlmacenIlegible},
			{"nombre con dos puntos", `{"version":1,"cuentas":[` +
				cuentaJSON("anonimo:sin-autenticar") + `]}`, usuarios.ErrAlmacenIlegible},
		}
		for _, c := range casos {
			t.Run(c.nombre, func(t *testing.T) {
				ruta := rutaNueva(t)
				if err := os.WriteFile(ruta, []byte(c.contenido), 0o600); err != nil {
					t.Fatal(err)
				}
				a, err := usuarios.Abrir(usuarios.Opciones{Ruta: ruta})
				if err == nil {
					hay, _ := a.HayAdministrador(context.Background())
					t.Fatalf("un almacen que no se entiende ha cargado como si nada "+
						"(hay administrador: %v). Un dato que HAY y no se entiende es un "+
						"error, nunca el valor por defecto", hay)
				}
				if !errors.Is(err, c.centinela) {
					t.Errorf("el error no es %v sino %v: quien monta no puede distinguir "+
						"un fichero roto de un fichero que falta", c.centinela, err)
				}
			})
		}
	})
}

// cuentaJSON compone una cuenta valida en forma, con el nombre que se le pase.
func cuentaJSON(usuario string) string {
	b, err := json.Marshal(usuario)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf(`{"usuario":%s,"algoritmo":%q,"iteraciones":%d,"sal":%q,"clave":%q,`+
		`"creado":"2026-09-03T09:00:00Z"}`, b, usuarios.Algoritmo,
		usuarios.IteracionesPorDefecto, strings.Repeat("00", usuarios.LongitudDeSal),
		strings.Repeat("00", usuarios.LongitudDeClave))
}

// EL SUELO DEL COSTE NO SE PUEDE AFLOJAR DESDE FUERA. Sin esto, el parametro de
// iteraciones solo serviria para una cosa: bajarlo.
func TestNoSePuedeAbrirUnAlmacenConMenosCosteDelMinimo(t *testing.T) {
	_, err := usuarios.Abrir(usuarios.Opciones{Ruta: rutaNueva(t), Iteraciones: 1000})
	if !errors.Is(err, usuarios.ErrCosteInsuficiente) {
		t.Fatalf("se ha podido abrir un almacen con 1000 iteraciones (err=%v). El suelo son "+
			"%d", err, usuarios.IteracionesMinimas)
	}
	// CONTROL POSITIVO: con el minimo justo SI abre. Sin el, un Abrir que
	// rechazara siempre pasaria la comprobacion de arriba.
	if _, err := usuarios.Abrir(usuarios.Opciones{
		Ruta: rutaNueva(t), Iteraciones: usuarios.IteracionesMinimas,
	}); err != nil {
		t.Fatalf("con el minimo exacto tiene que abrir: %v", err)
	}
}

// NI EL MENSAJE NI EL CENTINELA DICEN SI EL USUARIO EXISTE. Si lo dijeran, la
// primera orden de terminal que imprima este error seria un listado de usuarios.
func TestElFalloDeCredencialesNoDiceSiElUsuarioExiste(t *testing.T) {
	ctx := context.Background()
	a := abrirEn(t, rutaNueva(t))
	if err := a.CrearPrimerAdministrador(ctx, usuarioBueno, secretoBueno); err != nil {
		t.Fatal(err)
	}

	_, errNoExiste := a.Autenticar(ctx, "nadie-de-esta-casa", secretoBueno)
	_, errMalaClave := a.Autenticar(ctx, usuarioBueno, "contrasena-equivocada-1")
	if errNoExiste == nil || errMalaClave == nil {
		t.Fatal("unas credenciales que no valen tienen que fallar")
	}
	if errNoExiste.Error() != errMalaClave.Error() {
		t.Errorf("el mensaje distingue usuario inexistente (%q) de contrasena mala (%q): "+
			"eso es enumeracion de usuarios servida por el propio error",
			errNoExiste, errMalaClave)
	}
	if !errors.Is(errNoExiste, usuarios.ErrCredenciales) ||
		!errors.Is(errMalaClave, usuarios.ErrCredenciales) {
		t.Errorf("los dos fallos tienen que llevar el mismo centinela: %v / %v",
			errNoExiste, errMalaClave)
	}
	// Y EL SECRETO NO VIAJA DENTRO DEL ERROR, que es donde acabaria en el log.
	for _, e := range []error{errNoExiste, errMalaClave} {
		if strings.Contains(e.Error(), secretoBueno) ||
			strings.Contains(e.Error(), "contrasena-equivocada-1") {
			t.Errorf("el error lleva la contrasena dentro: %v", e)
		}
	}
	// CONTROL POSITIVO: las credenciales buenas SI entran, y con el nombre
	// canonico. Sin esto, un Autenticar que fallara siempre pasaria todo lo de
	// arriba.
	if sujeto, err := a.Autenticar(ctx, "  CISO  ", secretoBueno); err != nil || sujeto != usuarioBueno {
		t.Errorf("las credenciales buenas no entran, o el sujeto no es el canonico: "+
			"sujeto=%q err=%v", sujeto, err)
	}
}

// EL TIEMPO TAMPOCO LO DICE. Un usuario inexistente que conteste al instante
// mientras uno existente tarda un cuarto de segundo es un listado de usuarios
// medible desde fuera sin ninguna herramienta especial.
//
// EL UMBRAL ES FLOJO A PROPOSITO (un tercio del coste de una derivacion): esta
// puerta no persigue una diferencia de microsegundos, que en una maquina
// compartida es ruido, sino la diferencia de ORDEN DE MAGNITUD que sale cuando
// alguien pone un `return` antes de derivar.
func TestUnUsuarioQueNoExisteCuestaLoMismoQueUnoQueSi(t *testing.T) {
	if testing.Short() {
		t.Skip("mide tiempos: no en -short")
	}
	ctx := context.Background()
	a := abrirEn(t, rutaNueva(t))
	if err := a.CrearPrimerAdministrador(ctx, usuarioBueno, secretoBueno); err != nil {
		t.Fatal(err)
	}
	medir := func(usuario string) time.Duration {
		mejor := time.Duration(1<<62 - 1)
		for i := 0; i < 3; i++ {
			inicio := time.Now()
			_, _ = a.Autenticar(ctx, usuario, "contrasena-equivocada-1")
			if d := time.Since(inicio); d < mejor {
				mejor = d
			}
		}
		return mejor
	}
	existe := medir(usuarioBueno)
	noExiste := medir("nadie-de-esta-casa")
	diferencia := existe - noExiste
	if diferencia < 0 {
		diferencia = -diferencia
	}
	if diferencia > existe/3 {
		t.Errorf("autenticar a un usuario que existe tarda %s y a uno que no existe %s.\n"+
			"  Esa diferencia es un listado de usuarios: se mide desde fuera con un reloj.\n"+
			"  Arreglo: derivar tambien cuando el usuario no esta, con la sal de relleno.",
			existe.Round(time.Millisecond), noExiste.Round(time.Millisecond))
	}
}

// LA CARRERA DEL PRIMER ADMINISTRADOR. Dos peticiones simultaneas a
// /primer-admin no pueden crear dos cuentas, y la exclusion tiene que vivir
// TAMBIEN aqui: apoyarse solo en la reserva del token de `superficies/serve`
// seria dar por hecho que esa pantalla es la unica puerta de entrada al almacen.
func TestDosCreacionesSimultaneasNoCreanDosAdministradores(t *testing.T) {
	ruta := rutaNueva(t)
	a := abrirEn(t, ruta)

	const intentos = 8
	var espera sync.WaitGroup
	salida := make(chan error, intentos)
	arranque := make(chan struct{})
	for i := 0; i < intentos; i++ {
		espera.Add(1)
		go func(n int) {
			defer espera.Done()
			<-arranque
			salida <- a.CrearPrimerAdministrador(context.Background(),
				fmt.Sprintf("admin%d", n), secretoBueno)
		}(i)
	}
	close(arranque)
	espera.Wait()
	close(salida)

	exitos, yaHay, otros := 0, 0, 0
	for err := range salida {
		switch {
		case err == nil:
			exitos++
		case errors.Is(err, usuarios.ErrYaHayCuentas):
			yaHay++
		default:
			otros++
			t.Errorf("una creacion simultanea fallo por algo que no es «ya hay cuentas»: %v", err)
		}
	}
	if exitos != 1 {
		t.Fatalf("%d de %d creaciones simultaneas han tenido exito. Tiene que haber "+
			"exactamente UNA: dos administradores creados a la vez con el mismo token de "+
			"instalacion es una toma de control", exitos, intentos)
	}
	if yaHay != intentos-1 {
		t.Errorf("%d rechazos por «ya hay cuentas» y tenian que ser %d (%d por otras causas)",
			yaHay, intentos-1, otros)
	}
	// Y EL DISCO TIENE UNA SOLA CUENTA. Sin esto, un almacen que rechazara en
	// memoria y escribiera igual pasaria lo de arriba.
	if b := abrirEn(t, ruta); b.Cuentas() != 1 {
		t.Errorf("en el disco han quedado %d cuentas y tenia que quedar una", b.Cuentas())
	}
}

// La sal es POR USUARIO. Dos instalaciones con el mismo usuario y la misma
// contrasena no pueden producir la misma clave derivada: si la produjeran, una
// tabla precalculada valdria para las dos.
func TestDosCuentasConLaMismaContrasenaNoTienenLaMismaClave(t *testing.T) {
	ctx := context.Background()
	claves := make([]string, 2)
	for i := range claves {
		ruta := rutaNueva(t)
		a := abrirEn(t, ruta)
		if err := a.CrearPrimerAdministrador(ctx, usuarioBueno, secretoBueno); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(ruta) // #nosec G304 -- ruta del propio test
		if err != nil {
			t.Fatal(err)
		}
		claves[i] = string(b)
	}
	if claves[0] == claves[1] {
		t.Error("dos almacenes con el mismo usuario y la misma contrasena han producido el " +
			"mismo fichero. La sal no es por usuario, o no es aleatoria")
	}
}

// Los nombres de usuario que no valen, y por que cada uno.
func TestNombresDeUsuarioQueNoValen(t *testing.T) {
	casos := []struct{ nombre, valor string }{
		{"vacio", ""},
		{"solo espacios", "   "},
		{"con dos puntos", "anonimo:sin-autenticar"},
		{"con un espacio dentro", "el ciso"},
		{"con un caracter de control", "ci" + string(rune(0)) + "so"},
		{"demasiado largo", strings.Repeat("a", usuarios.LongitudMaximaDelUsuario+1)},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if _, err := usuarios.NormalizarUsuario(c.valor); !errors.Is(err, usuarios.ErrUsuarioNoValido) {
				t.Errorf("%q se ha aceptado como nombre de usuario (err=%v)", c.valor, err)
			}
		})
	}
	// CONTROL POSITIVO, y ademas comprueba la canonizacion.
	if n, err := usuarios.NormalizarUsuario("  CISO  "); err != nil || n != "ciso" {
		t.Errorf("un nombre normal tiene que valer y salir canonico: %q, %v", n, err)
	}
}

// La contrasena tiene suelo y techo, y el techo NO es cosmetico: sin el, cada
// intento de entrada con una contrasena de diez megabytes es trabajo que paga el
// servidor y elige quien ataca.
func TestLaContrasenaTieneSueloYTecho(t *testing.T) {
	corta := strings.Repeat("a", usuarios.LongitudMinimaDelSecreto-1)
	if err := usuarios.ComprobarSecreto(corta); !errors.Is(err, usuarios.ErrSecretoNoValido) {
		t.Errorf("una contrasena de %d caracteres se ha aceptado", len(corta))
	}
	larga := strings.Repeat("a", usuarios.LongitudMaximaDelSecreto+1)
	if err := usuarios.ComprobarSecreto(larga); !errors.Is(err, usuarios.ErrSecretoNoValido) {
		t.Errorf("una contrasena de %d bytes se ha aceptado", len(larga))
	}
	if err := usuarios.ComprobarSecreto(secretoBueno); err != nil {
		t.Errorf("una contrasena normal tiene que valer: %v", err)
	}
}

// Un almacen sin ruta no se abre: seria un administrador que hay que volver a
// crear en cada arranque, o sea ninguno.
func TestUnAlmacenSinRutaNoSeAbre(t *testing.T) {
	for _, r := range []string{"", "   "} {
		if _, err := usuarios.Abrir(usuarios.Opciones{Ruta: r}); !errors.Is(err, usuarios.ErrSinRuta) {
			t.Errorf("se ha abierto un almacen con la ruta %q (err=%v)", r, err)
		}
	}
}

// El fichero queda en 0600 y no deja restos temporales. Los permisos solo se
// comprueban donde significan algo: en Windows, os.Chmod solo mueve el bit de
// solo lectura, asi que exigir 0600 alli seria una puerta que no vigila nada y
// ademas se pondria roja sin motivo.
func TestElFicheroDeCuentasQuedaCerradoYSinRestos(t *testing.T) {
	ruta := rutaNueva(t)
	a := abrirEn(t, ruta)
	if err := a.CrearPrimerAdministrador(context.Background(), usuarioBueno, secretoBueno); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		fi, err := os.Stat(ruta)
		if err != nil {
			t.Fatal(err)
		}
		if modo := fi.Mode().Perm(); modo != 0o600 {
			t.Errorf("el fichero de cuentas quedo en %#o y tiene que quedar en 0600: lleva "+
				"claves derivadas dentro", modo)
		}
	}
	entradas, err := os.ReadDir(filepath.Dir(ruta))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entradas {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("ha quedado el temporal %q. La escritura es temporal mas rename: si el "+
				"temporal sobrevive, o el rename no paso o no se limpio", e.Name())
		}
	}
	if len(entradas) != 1 {
		t.Errorf("en el directorio han quedado %d ficheros y tenia que quedar uno: %v",
			len(entradas), entradas)
	}
}
