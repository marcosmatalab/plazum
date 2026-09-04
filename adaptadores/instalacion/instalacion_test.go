package instalacion

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func almacen(t *testing.T) (*Almacen, string) {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), "instalacion.json")
	a, err := Abrir(Opciones{Ruta: ruta, Ahora: func() time.Time {
		return time.Date(2026, 9, 4, 21, 0, 0, 0, time.UTC)
	}})
	if err != nil {
		t.Fatalf("abrir: %v", err)
	}
	return a, ruta
}

// UNA INSTALACION RECIEN DESCARGADA NO TIENE IDENTIDAD, Y ESO NO ES UN ERROR.
// Es la unica de las tres formas de la nada que es la nada de verdad.
func TestUnaInstalacionSinConfigurarNoEsUnError(t *testing.T) {
	a, _ := almacen(t)
	if q := a.Quien(); q.Hay() {
		t.Errorf("una instalacion recien descargada dice tener identidad: %#v", q)
	}
}

func TestLoFijadoSobreviveAlReinicio(t *testing.T) {
	a, ruta := almacen(t)
	if err := a.Fijar("Acme S.L."); err != nil {
		t.Fatalf("fijar: %v", err)
	}
	otro, err := Abrir(Opciones{Ruta: ruta})
	if err != nil {
		t.Fatalf("reabrir: %v", err)
	}
	q := otro.Quien()
	if !q.Hay() {
		t.Fatal("la identidad no ha sobrevivido al reinicio")
	}
	if q.Organizacion != "Acme S.L." {
		t.Errorf("el nombre ha vuelto como %q", q.Organizacion)
	}
	if q.Sujeto != "acme-s-l" {
		t.Errorf("el sujeto ha vuelto como %q y se acuno acme-s-l", q.Sujeto)
	}
	if q.Acunado.IsZero() {
		t.Error("el instante de acunacion ha vuelto a cero, que es el ano 1 con cara de dato")
	}
}

// EL SUJETO SE ACUNA UNA VEZ Y NO SE MUEVE. Es la decision del paquete y por eso
// tiene su propia puerta: cambiar el nombre es un hecho normal, y re-derivar el
// cumplimiento entero por haber renombrado la empresa no lo es.
func TestCambiarElNombreNoMueveElSujeto(t *testing.T) {
	a, ruta := almacen(t)
	if err := a.Fijar("Acme S.L."); err != nil {
		t.Fatal(err)
	}
	antes := a.Quien()

	if err := a.Fijar("Beta Sistemas SA"); err != nil {
		t.Fatalf("cambiar el nombre: %v", err)
	}
	despues := a.Quien()

	if despues.Organizacion != "Beta Sistemas SA" {
		t.Errorf("el nombre no ha cambiado: %q", despues.Organizacion)
	}
	if despues.Sujeto != antes.Sujeto {
		t.Errorf(`el sujeto se ha movido de %q a %q al renombrar la organizacion.

  Con el sujeto se mueve la instancia de la que cuelgan TODAS las respuestas, y
  con ella las derivaciones, las citas y lo que ya este escrito en un
  expediente. Renombrar una empresa es un hecho normal; re-derivar su
  cumplimiento entero por haberla renombrado, no.`, antes.Sujeto, despues.Sujeto)
	}
	if !despues.Acunado.Equal(antes.Acunado) {
		t.Errorf("el instante de acunacion se ha movido de %v a %v: la acunacion es una sola",
			antes.Acunado, despues.Acunado)
	}

	// Y en disco tambien, que es donde importa.
	otro, err := Abrir(Opciones{Ruta: ruta})
	if err != nil {
		t.Fatal(err)
	}
	if otro.Quien().Sujeto != antes.Sujeto {
		t.Errorf("en disco el sujeto es %q y tenia que ser %q", otro.Quien().Sujeto, antes.Sujeto)
	}
}

func TestFijarElSujetoAManoSoloValeAntesDeAcunar(t *testing.T) {
	a, _ := almacen(t)
	if err := a.FijarSujeto("sis"); err != nil {
		t.Fatalf("fijar el sujeto en una instalacion sin configurar: %v", err)
	}
	if q := a.Quien(); q.Sujeto != "sis" {
		t.Errorf("el sujeto puesto a mano no ha entrado: %#v", q)
	}
	// Y la segunda vez NO.
	if err := a.FijarSujeto("otro"); !errors.Is(err, ErrSujetoYaAcunado) {
		t.Fatalf("cambiar el sujeto acunado tenia que fallar con ErrSujetoYaAcunado y da %v", err)
	}
	// Fijar el nombre despues NO lo mueve tampoco.
	if err := a.Fijar("Acme S.L."); err != nil {
		t.Fatal(err)
	}
	if q := a.Quien(); q.Sujeto != "sis" {
		t.Errorf("fijar el nombre ha movido el sujeto puesto a mano: %#v", q)
	}
}

// LAS TRES FORMAS DE LA NADA, recorridas enteras. La primera es la unica que no
// es error.
func TestUnFicheroQueHayYNoSeEntiendeEsErrorYNuncaSinConfigurar(t *testing.T) {
	casos := []struct {
		nombre    string
		contenido string
		quiero    error
	}{
		{"presente y vacio", "", ErrAlmacenVacio},
		{"no es json", "{no", ErrAlmacenIlegible},
		{"sin version", `{"organizacion":"Acme","sujeto":"acme","acunado":"2026-09-04T21:00:00Z"}`,
			ErrAlmacenIlegible},
		{"version de otro plazum",
			`{"version":99,"organizacion":"Acme","sujeto":"acme","acunado":"2026-09-04T21:00:00Z"}`,
			ErrVersionDesconocida},
		{"nombre en blanco",
			`{"version":1,"organizacion":"  ","sujeto":"acme","acunado":"2026-09-04T21:00:00Z"}`,
			ErrAlmacenIlegible},
		{"sujeto en blanco",
			`{"version":1,"organizacion":"Acme","sujeto":"","acunado":"2026-09-04T21:00:00Z"}`,
			ErrAlmacenIlegible},
		{"sujeto que no tiene forma de identificador",
			`{"version":1,"organizacion":"Acme","sujeto":"Acme S.L.","acunado":"2026-09-04T21:00:00Z"}`,
			ErrAlmacenIlegible},
		{"instante que no se entiende",
			`{"version":1,"organizacion":"Acme","sujeto":"acme","acunado":"ayer"}`,
			ErrAlmacenIlegible},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			ruta := filepath.Join(t.TempDir(), "instalacion.json")
			if err := os.WriteFile(ruta, []byte(c.contenido), 0o600); err != nil {
				t.Fatal(err)
			}
			a, err := Abrir(Opciones{Ruta: ruta})
			if !errors.Is(err, c.quiero) {
				t.Fatalf("tenia que fallar con %v y da %v", c.quiero, err)
			}
			if a != nil {
				t.Error("ha devuelto un almacen ademas del error: quien no compruebe el error " +
					"trabajaria sobre una instalacion sin identidad creyendo que la tiene")
			}
		})
	}

	// EL CONTROL POSITIVO: un fichero bien escrito SI carga. Sin el, lo de
	// arriba solo demuestra que Abrir sabe fallar.
	t.Run("uno bien escrito carga", func(t *testing.T) {
		ruta := filepath.Join(t.TempDir(), "instalacion.json")
		doc := `{"version":1,"organizacion":"Acme S.L.","sujeto":"acme-s-l",` +
			`"acunado":"2026-09-04T21:00:00Z"}`
		if err := os.WriteFile(ruta, []byte(doc), 0o600); err != nil {
			t.Fatal(err)
		}
		a, err := Abrir(Opciones{Ruta: ruta})
		if err != nil {
			t.Fatalf("un fichero bien escrito no carga: %v", err)
		}
		if q := a.Quien(); q.Organizacion != "Acme S.L." || q.Sujeto != "acme-s-l" {
			t.Errorf("ha cargado %#v", q)
		}
	})
}

// EL NOMBRE: lo que no vale, y por que no se limpia en silencio.
func TestUnNombreQueNoValeNoSeLimpiaEnSilencio(t *testing.T) {
	a, _ := almacen(t)
	casos := []struct{ nombre, valor string }{
		{"en blanco", "   "},
		{"con salto de linea", "Acme\nS.L."},
		{"con un invisible dentro", "Acme​S.L."},
		{"mas largo que el tope", strings.Repeat("x", MaxLongitudDelNombre+1)},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if err := a.Fijar(c.valor); !errors.Is(err, ErrNombreNoValido) {
				t.Fatalf("tenia que fallar con ErrNombreNoValido y da %v", err)
			}
			if a.Quien().Hay() {
				t.Error("se ha rechazado el nombre y aun asi la instalacion dice tener identidad")
			}
		})
	}
}

// LA ACUNACION, con sus casos y con el que NO se inventa un identificador.
func TestAcunarSujetoNoSeInventaNadaCuandoNoSalePadre(t *testing.T) {
	buenos := []struct{ nombre, quiero string }{
		{"Acme S.L.", "acme-s-l"},
		{"Acme", "acme"},
		{"  Fundación Público  ", "fundacion-publico"},
		{"Beta 2026 SA", "beta-2026-sa"},
		{"---Acme---", "acme"},
	}
	for _, c := range buenos {
		t.Run(c.nombre, func(t *testing.T) {
			s, err := AcunarSujeto(c.nombre)
			if err != nil {
				t.Fatalf("acunar %q: %v", c.nombre, err)
			}
			if s != c.quiero {
				t.Errorf("ha acunado %q y esperaba %q", s, c.quiero)
			}
		})
	}

	// Y LA DIRECCION QUE IMPORTA: de un nombre del que no sale nada NO se
	// inventa un identificador.
	malos := []string{"...", "   ", "株式会社"}
	for _, n := range malos {
		t.Run("no sale nada de "+n, func(t *testing.T) {
			s, err := AcunarSujeto(n)
			if !errors.Is(err, ErrSujetoNoValido) {
				t.Fatalf("de %q ha salido %q sin error.\n"+
					"  Inventarse uno seria acunar en silencio la instancia de la que van a "+
					"colgar todas las respuestas para siempre", n, s)
			}
		})
	}
}

// El fichero no es del resto de la maquina: dice a que se dedica una
// organizacion y con que nombre la llaman sus reglas.
func TestElFicheroQuedaEn0600(t *testing.T) {
	// SE PREGUNTA A runtime Y NO AL ENTORNO. La primera version de esta linea
	// miraba os.Getenv("GOOS"), que en ejecucion no vale nada: GOOS es una
	// variable del CONSTRUCTOR, no del proceso. Asi que el salto no saltaba
	// nunca y el test fallaba en Windows por una guarda que no guardaba, que es
	// la familia de siempre en su version mas pequena.
	if runtime.GOOS == "windows" {
		t.Skip("los permisos POSIX no significan lo mismo en Windows: el bit de grupo y el " +
			"de otros no existen, y comprobarlos aqui daria rojo sobre algo que el sistema " +
			"no tiene. En CI corre en ubuntu, que es donde la comprobacion significa algo")
	}
	a, ruta := almacen(t)
	if err := a.Fijar("Acme S.L."); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if m := fi.Mode().Perm(); m&0o077 != 0 {
		t.Errorf("el fichero ha quedado en %o y tiene que estar en 0600", m)
	}
}
