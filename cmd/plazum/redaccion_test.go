package main

import (
	"os"
	"runtime"
	"testing"
)

// La redaccion del informe de fallo, probada contra hogares QUE NO SON EL DE LA
// MAQUINA QUE EJECUTA EL TEST.
//
// POR QUE ESTE FICHERO EXISTE. La unica comprobacion que habia usaba
// `os.UserHomeDir()` directamente, o sea el hogar real. El hogar real de una
// maquina Windows y el de un runner de Actions son los dos largos y de segundo
// nivel (`C:\Users\algo`, `/home/runner`), asi que dos clases enteras de hogar
// no se probaban nunca: el de primer nivel (`/root`, que es el de un contenedor
// que corre como root) y el degenerado (`/`).
//
// El resultado fue una puerta verde en las dos maquinas donde nadie ejecuta el
// producto y roja dentro del propio Dockerfile de este repositorio.

// fijarHogar apunta os.UserHomeDir a una ruta de mentira, y COMPRUEBA QUE LO HA
// CONSEGUIDO. Sin esa comprobacion, una tabla de hogares sinteticos que no se
// llegan a aplicar estaria probando el hogar real una y otra vez, y saldria
// verde sin medir nada de lo que dice medir.
func fijarHogar(t *testing.T, ruta string) {
	t.Helper()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("USERPROFILE", ruta)
	case "plan9":
		t.Setenv("home", ruta)
	default:
		t.Setenv("HOME", ruta)
	}
	got, err := os.UserHomeDir()
	if err != nil || got != ruta {
		t.Fatalf("no he podido fijar el hogar a %q (os.UserHomeDir dice %q, err %v).\n"+
			"  Sin poder fijarlo, esta tabla probaria el hogar real en todos los casos y "+
			"seria verde sin medir nada", ruta, got, err)
	}
}

// fijarTemporal hace lo mismo con os.TempDir y DEVUELVE LO QUE EL SISTEMA HA
// ENTENDIDO, que no siempre es lo que se le pidio.
//
// Windows normaliza: con TMP=/tmp, os.TempDir() devuelve "C:\tmp", porque una
// ruta sin volumen no es absoluta ahi. La primera version de este ayudante
// exigia igualdad exacta y se ponia roja en Windows, que es el fallo simetrico
// del que trajo a este fichero: una comprobacion calibrada contra un solo
// sistema operativo. Se devuelve el valor efectivo y quien llama construye sus
// expectativas con el.
func fijarTemporal(t *testing.T, ruta string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Setenv("TMP", ruta)
		t.Setenv("TEMP", ruta)
	} else {
		t.Setenv("TMPDIR", ruta)
	}
	efectivo := os.TempDir()
	if efectivo == "" {
		t.Fatalf("tras fijar el temporal a %q, os.TempDir devuelve vacio", ruta)
	}
	return efectivo
}

// LA TABLA. Los hogares que rompen, y el que rompio de verdad.
func TestLaRedaccionAguantaHogaresQueNoSonElDeQuienEscribeElTest(t *testing.T) {
	casos := []struct {
		nombre   string
		hogar    string
		mensaje  string
		esperado string
		porque   string
	}{
		{
			"hogar de primer nivel, que es el de root en un contenedor",
			"/root",
			"no puedo escribir en /root/plazum/datos",
			"no puedo escribir en ~/plazum/datos",
			"es el caso que puso rojo el test en origin/main: el padre de /root es la raiz, " +
				"y toda ruta absoluta contiene la raiz",
		},
		{
			"hogar normal de segundo nivel",
			"/home/marcos",
			"no puedo escribir en /home/marcos/plazum/datos",
			"no puedo escribir en ~/plazum/datos",
			"el caso comodo, el unico que se probaba antes",
		},
		{
			"EL HOGAR DEGENERADO: la raiz",
			"/",
			"no puedo escribir en /var/lib/plazum",
			"no puedo escribir en /var/lib/plazum",
			"HOME=/ es posible en un contenedor. La raiz esta DENTRO de toda ruta absoluta, " +
				"asi que sustituirla no redacta nada y destruye el mensaje entero. Nunca " +
				"vacio no es lo mismo que siempre util (invariante 8 de CLAUDE.md)",
		},
		{
			"un hogar de dos letras, tambien degenerado",
			"/a",
			"no puedo escribir en /var/lib/plazum",
			"no puedo escribir en /var/lib/plazum",
			"demasiado corto para sustituirlo sin destrozar el texto. Se prefiere no " +
				"redactar a redactar mal, y el bloque lleva escrito que es una salida generada",
		},
		{
			"el hogar de otra persona con el mismo prefijo",
			"/home/ana",
			"no puedo leer /home/anastasia/x ni /home/ana/y",
			"no puedo leer /home/<usuario>stasia/x ni ~/y",
			"DOS PROPIEDADES DISTINTAS, y este caso las separa. La RUTA /home/ana no se " +
				"sustituye dentro de /home/anastasia, porque sustituirRuta exige frontera: " +
				"si se sustituyera saldria ~stasia y el informe mentiria sobre donde estaba " +
				"el problema. El NOMBRE ana si se sustituye en cualquier posicion, a " +
				"proposito, y por eso queda <usuario>stasia: redactar de mas estropea la " +
				"lectura y redactar de menos publica un dato personal, y de los dos errores " +
				"el que puede permitirse un informe de fallo es el primero",
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			fijarHogar(t, c.hogar)
			// Un temporal largo y ajeno, para que no interfiera con lo que aqui
			// se mide.
			_ = fijarTemporal(t, "/un/temporal/que/no/sale/en/estos/mensajes")
			got := redactar(c.mensaje)
			if got != c.esperado {
				t.Fatalf("con hogar %q\n  mensaje  %q\n  esperado %q\n  obtenido %q\n  porque: %s",
					c.hogar, c.mensaje, c.esperado, got, c.porque)
			}
		})
	}
}

// El hermano del hogar, roto en la OTRA direccion: la guarda pedia len > 4 y
// "/tmp" son exactamente cuatro caracteres, asi que en Linux el directorio
// temporal no se redactaba nunca. Verde en Windows, donde el temporal es una
// ruta larga dentro del perfil del usuario, y por tanto silenciosamente inutil
// en el sistema donde de verdad corre el producto.
//
// Se prueba contra sustituirRuta con "/tmp" LITERAL y no a traves de
// os.TempDir, y el motivo es el mismo que hizo falta arriba: Windows no deja
// que os.TempDir devuelva "/tmp", asi que pasar por ahi convertiria este test
// en un test de otra cosa segun donde corra. La longitud de cuatro es la
// propiedad, y se mide directamente.
func TestElTemporalDeCuatroCaracteresTambienSeRedacta(t *testing.T) {
	if !redactable("/tmp") {
		t.Fatal("HALLAZGO: /tmp se da por no redactable. Son cuatro caracteres exactos y la " +
			"guarda pedia mas de cuatro, asi que en Linux el temporal no se redactaba nunca. " +
			"En muchos sistemas el temporal lleva dentro el nombre de usuario")
	}
	if got := sustituirRuta("el ensayo dejo basura en /tmp/plazum-123", "/tmp", "<temporal>"); got != "el ensayo dejo basura en <temporal>/plazum-123" {
		t.Errorf("/tmp no se sustituye: %q", got)
	}
	// Y la frontera, que es lo que permite bajar la guarda a cuatro sin
	// estropear el texto.
	if got := sustituirRuta("mira /tmpfiles/x", "/tmp", "<temporal>"); got != "mira /tmpfiles/x" {
		t.Errorf("HALLAZGO: /tmp dentro de /tmpfiles se ha sustituido: %q", got)
	}

	// Y el camino completo, con el temporal que el sistema si acepte: lo que
	// redactar hace de verdad, sea cual sea el sistema.
	fijarHogar(t, "/home/marcos")
	tmp := fijarTemporal(t, "/un/temporal/de/prueba")
	got := redactar("el ensayo dejo basura en " + tmp + string(os.PathSeparator) + "plazum-123")
	if got != "el ensayo dejo basura en <temporal>"+string(os.PathSeparator)+"plazum-123" {
		t.Errorf("el temporal efectivo (%q) no se redacta de punta a punta: %q", tmp, got)
	}
	// CONTROL NEGATIVO: un temporal degenerado no se sustituye.
	if got := sustituirRuta("no puedo escribir en /var/lib/plazum", "/", "<temporal>"); got != "no puedo escribir en /var/lib/plazum" {
		t.Errorf("un temporal degenerado ha destrozado el mensaje: %q", got)
	}
}

// La frontera de ruta, aislada de la sustitucion del nombre.
//
// Va contra sustituirRuta y no contra redactar a proposito: dentro de redactar
// las dos sustituciones se pisan, porque el nombre de usuario ES el ultimo
// tramo de la ruta del hogar, y ahi no se puede ver una sin la otra. Es
// exactamente la trampa que este proyecto ya se ha comido una vez: dos
// comprobaciones que se cubren la una a la otra son una sola comprobacion.
func TestLaSustitucionDeRutaExigeFrontera(t *testing.T) {
	casos := []struct{ nombre, texto, ruta, esperado string }{
		{"al final del texto", "esta en /home/ana", "/home/ana", "esta en ~"},
		{"seguida de barra", "esta en /home/ana/x", "/home/ana", "esta en ~/x"},
		{"seguida de contrabarra", `esta en /home/ana\x`, "/home/ana", `esta en ~\x`},
		{"pegada a mas letras: NO se sustituye", "esta en /home/anastasia/x", "/home/ana",
			"esta en /home/anastasia/x"},
		{"dos veces, una valida y otra no", "/home/anabel y /home/ana/x", "/home/ana",
			"/home/anabel y ~/x"},
		{"no aparece", "esta en /var/lib/plazum", "/home/ana", "esta en /var/lib/plazum"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if got := sustituirRuta(c.texto, c.ruta, "~"); got != c.esperado {
				t.Fatalf("sustituirRuta(%q, %q)\n  esperado %q\n  obtenido %q",
					c.texto, c.ruta, c.esperado, got)
			}
		})
	}
}

// Y la otra mitad, dicha en voz alta para que nadie la "arregle": el NOMBRE de
// usuario se sustituye sin frontera, y es deliberado.
func TestElNombreDeUsuarioSeSustituyeSinFronteraYEsDeliberado(t *testing.T) {
	fijarHogar(t, "/home/ana")
	_ = fijarTemporal(t, "/un/temporal/largo/y/ajeno")
	got := redactar("el fichero de anastasia no es de ana")
	if got != "el fichero de <usuario>stasia no es de <usuario>" {
		t.Fatalf("la sustitucion del nombre ha dejado de ser sin frontera: %q\n"+
			"  Si esto se ha cambiado a proposito, mira antes que un nombre de usuario NO es "+
			"una ruta: no tiene separadores, asi que no hay frontera que exigir. Redactar de "+
			"mas estropea la lectura; redactar de menos publica un dato personal", got)
	}
}

// redactable por si misma, porque es la guarda y conviene verla fallar sola.
func TestRedactableRechazaLoDegenerado(t *testing.T) {
	rechazadas := []string{"", "   ", "/", "/a", "/ab", "."}
	for _, r := range rechazadas {
		if redactable(r) {
			t.Errorf("HALLAZGO: %q se da por redactable. Sustituirla no redacta nada y "+
				"destroza el texto", r)
		}
	}
	// CONTROL NEGATIVO: las que SI hay que redactar. Sin esto, una guarda que
	// dijera que no a todo pasaria el test de arriba y dejaria de redactar.
	aceptadas := []string{"/tmp", "/root", "/home/marcos", "/var/folders/xy/z"}
	for _, r := range aceptadas {
		if !redactable(r) {
			t.Errorf("HALLAZGO: %q se rechaza y hay que redactarla: es una ruta real de "+
				"hogar o de temporal, y lleva dentro datos de la persona", r)
		}
	}
}
