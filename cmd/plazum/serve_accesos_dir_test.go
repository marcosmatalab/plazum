package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/superficies/uar"
)

// EL CENSO DE PRUEBA. Tres cuentas y cuatro accesos, con las columnas que
// censo.ColumnasHabituales() sabe reconocer.
const censoDePrueba = "usuario,permiso,nombre\n" +
	"ana,admin,Ana Ruiz\n" +
	"ana,lectura,Ana Ruiz\n" +
	"luis,lectura,Luis Paz\n" +
	"eva,admin,Eva Sanz\n"

// uarConSubida monta la pantalla sobre un directorio de datos vacio, que es la
// instalacion recien descargada.
func uarConSubida(t *testing.T) *uar.Superficie {
	t.Helper()
	u, _ := uarConSubidaEn(t, t.TempDir())
	return u
}

func uarConSubidaEn(t *testing.T, dir string) (*uar.Superficie, campanaEnDirectorio) {
	t.Helper()
	d := campanaEnDirectorio{dir: dir, ahora: time.Now}
	u, err := construirUAR(opcionesUAR{
		Fuente:   d,
		Abrir:    d,
		Catalogo: catDePrueba(t),
		Quien:    func(*http.Request) string { return "ciso" },
		// UN TOKEN CUALQUIERA. Aqui se prueba la superficie sola, sin el
		// ProtectorCSRF de serve delante; que el token se EXIJA lo prueba
		// TestLasRutasMutantesDeLaUARExigenTokenCSRF, que si monta el servidor.
		Tokens: func(*http.Request) (string, error) { return "t0ken", nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	return u, d
}

// subir compone la peticion multipart tal y como la manda el navegador.
//
// hayFichero es un parametro aparte del contenido A PROPOSITO: «no viene
// ninguna parte de fichero» y «viene una parte de fichero con cero bytes» son
// dos cosas distintas y son las dos formas de la nada, y con un solo argumento
// (contenido vacio = no mandar nada) el segundo caso no se puede ni escribir. Es
// exactamente la trampa que el invariante 8 persigue, aqui en el arnes de
// pruebas en vez de en el producto.
func subir(t *testing.T, contenido string, hayFichero bool, sistema string) *http.Request {
	t.Helper()
	var cuerpo bytes.Buffer
	w := multipart.NewWriter(&cuerpo)
	if sistema != "" {
		if err := w.WriteField("sistema", sistema); err != nil {
			t.Fatal(err)
		}
	}
	if hayFichero {
		f, err := w.CreateFormFile("fichero", "usuarios.csv")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(contenido)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/uar/abrir", &cuerpo)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

// EL CAMINO ENTERO SIN TOCAR EL TERMINAL, que es lo que esta pieza viene a
// comprar: se sube el CSV por el navegador y la siguiente peticion ya ensena la
// campana con sus filas.
//
// SE MIDE SOBRE EL RESULTADO Y NO SOBRE LA ESCRITURA. Comprobar que el fichero
// aparece en disco diria que se escribio algo; lo que hace falta saber es que la
// PANTALLA lo ensena, porque entre las dos cosas estan el sello, el ledger y la
// reconstruccion, que es justo donde puede romperse.
func TestSubirElCensoPorElNavegadorAbreLaCampanaYLaPantallaLaEnsena(t *testing.T) {
	dir := t.TempDir()
	u, _ := uarConSubidaEn(t, dir)

	rec := httptest.NewRecorder()
	u.ServeHTTP(rec, subir(t, censoDePrueba, true, "erp"))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("codigo %d al subir, se esperaba 303:\n%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	u.ServeHTTP(rec, httptest.NewRequest("GET", "/uar/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("codigo %d al mirar la campana recien abierta", rec.Code)
	}
	cuerpo := rec.Body.String()
	for _, quien := range []string{"ana", "luis", "eva"} {
		if !strings.Contains(cuerpo, quien) {
			t.Errorf("la campana no ensena a %q:\n%s", quien, cuerpo)
		}
	}
	if strings.Contains(cuerpo, catDePrueba(t).Traducir("es", "uar.sin_campana.titulo")) {
		t.Errorf("sigue diciendo que no hay campana despues de subirla:\n%s", cuerpo)
	}
	// Y EN DISCO QUEDAN LOS DOS FICHEROS, con el CSV nombrado por su sello.
	ledgers, err := filepath.Glob(filepath.Join(dir, "ledger.json"))
	if err != nil || len(ledgers) != 1 {
		t.Fatalf("no ha quedado el ledger en %s (%v)", dir, err)
	}
	csvs, err := filepath.Glob(filepath.Join(dir, "*.csv"))
	if err != nil || len(csvs) != 1 {
		t.Fatalf("se esperaba UN csv en %s y hay %d (%v)", dir, len(csvs), err)
	}
	nombre := strings.TrimSuffix(filepath.Base(csvs[0]), ".csv")
	if !reSello.MatchString(nombre) {
		t.Errorf("el csv se llama %q y tenia que llamarse como su sello, 64 hexadecimales. "+
			"Un nombre que venga del usuario es por donde se escribe fuera del directorio",
			nombre)
	}
}

// DOS SUBIDAS DEL MISMO SISTEMA EL MISMO DIA SON DOS CAMPANAS, Y MANDA LA
// SEGUNDA.
//
// Es la manana de un martes cualquiera: alguien exporta mal del IdP, lo arregla
// y lo vuelve a mandar. Con el identificador compuesto solo de sistema y fecha,
// las dos campanas se llamarian igual, accesos.Reconstruir se quedaria con la
// PRIMERA apertura del sujeto y la pantalla seguiria ensenando el fichero viejo
// SIN DECIR NADA. Un fallo que no da error y ademas ensena datos caducados es el
// que nadie encuentra.
func TestDosSubidasElMismoDiaNoSePisanYMandaLaSegunda(t *testing.T) {
	dir := t.TempDir()
	u, _ := uarConSubidaEn(t, dir)

	rec := httptest.NewRecorder()
	u.ServeHTTP(rec, subir(t, censoDePrueba, true, "erp"))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("la primera subida ha contestado %d", rec.Code)
	}
	segundo := censoDePrueba + "mario,admin,Mario Gil\n"
	rec = httptest.NewRecorder()
	u.ServeHTTP(rec, subir(t, segundo, true, "erp"))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("la segunda subida ha contestado %d:\n%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	u.ServeHTTP(rec, httptest.NewRequest("GET", "/uar/", nil))
	cuerpo := rec.Body.String()
	if !strings.Contains(cuerpo, "mario") {
		t.Errorf("la pantalla sigue ensenando la campana vieja: la fila nueva no esta.\n%s",
			cuerpo)
	}
}

// LO QUE LLEGA POR LA FRONTERA Y NO SE ENTIENDE ES UN ERROR, NUNCA UN DEFECTO.
//
// Las cuatro filas son las tres formas de la nada del invariante 8 mas la
// travesia: campo ausente, campo presente y vacio, fichero presente y no
// interpretable. Ninguna de las cuatro puede acabar en una campana abierta,
// porque una campana abierta es una entrada en un registro que no se reescribe.
func TestLaSubidaRechazaLoQueNoSePuedeInterpretar(t *testing.T) {
	casos := []struct {
		nombre     string
		contenido  string
		hayFichero bool
		sistema    string
	}{
		{"sin sistema", censoDePrueba, true, ""},
		{"sistema en blanco", censoDePrueba, true, "   "},
		{"sin parte de fichero", "", false, "erp"},
		{"parte de fichero con cero bytes", "", true, "erp"},
		{"fichero que no es un censo", "esto no es un csv\n", true, "erp"},
		{"sistema con travesia", censoDePrueba, true, "../../etc"},
		{"sistema con salto de linea", censoDePrueba, true, "erp\nadmin"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			dir := t.TempDir()
			u, _ := uarConSubidaEn(t, dir)
			rec := httptest.NewRecorder()
			u.ServeHTTP(rec, subir(t, c.contenido, c.hayFichero, c.sistema))
			if rec.Code == http.StatusSeeOther {
				t.Fatalf("la subida se ha aceptado y tenia que rechazarse")
			}
			// Y NO HA QUEDADO NADA ESCRITO. Es la mitad que importa: un rechazo
			// que ya haya tocado el disco deja una campana a medias en un
			// registro append-only.
			entradas, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entradas) != 0 {
				t.Errorf("el rechazo ha dejado %d ficheros en %s: %v", len(entradas), dir,
					entradas)
			}
		})
	}
}

// SIN SESION NO SE SUBE NADA, Y ADEMAS NO SE CUENTA QUE HAY ALGO QUE SUBIR.
//
// Una instantanea del censo sin quien la subio no se puede atar a nadie y no
// certifica nada, igual que una decision sin autor. El 401 es el mismo que
// devuelve la pantalla al mirarla, y por lo mismo: quien no ha entrado no tiene
// que enterarse de que detras hay una lista de personas.
func TestSinSesionLaSubidaDelCensoNoEscribeNada(t *testing.T) {
	dir := t.TempDir()
	d := campanaEnDirectorio{dir: dir, ahora: time.Now}
	u, err := construirUAR(opcionesUAR{
		Fuente:   d,
		Abrir:    d,
		Catalogo: catDePrueba(t),
		Tokens:   func(*http.Request) (string, error) { return "t0ken", nil },
		// sin Quien
	})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	u.ServeHTTP(rec, subir(t, censoDePrueba, true, "erp"))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("codigo %d, se esperaba 401", rec.Code)
	}
	entradas, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entradas) != 0 {
		t.Fatalf("una subida sin sesion ha escrito %d ficheros en %s", len(entradas), dir)
	}
}

// SIN ADAPTADOR NO SE PINTA EL FORMULARIO, y el estado vacio dice quien tiene
// que arreglarlo.
//
// Es el invariante 8 en una frontera de construccion: el valor cero de "no se
// abrir campanas" es no ensenar el boton, no ensenarlo y contestar 422. Y el
// siguiente paso sigue existiendo, solo que lo da otra persona: quien monto la
// instalacion, no quien mira la pantalla.
func TestSinAdaptadorDeAperturaElEstadoVacioSigueTeniendoSiguientePaso(t *testing.T) {
	u, err := construirUAR(opcionesUAR{
		Catalogo: catDePrueba(t),
		Quien:    func(*http.Request) string { return "ciso" },
		Tokens:   func(*http.Request) (string, error) { return "t0ken", nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	u.ServeHTTP(rec, httptest.NewRequest("GET", "/uar/", nil))
	cuerpo := rec.Body.String()
	if strings.Contains(cuerpo, `action="/uar/abrir"`) {
		t.Errorf("pinta un formulario de subida sin nadie que sepa abrir la campana:\n%s",
			cuerpo)
	}
	if !strings.Contains(cuerpo, catDePrueba(t).Traducir("es", "uar.subir.sin_almacen")) {
		t.Errorf("no dice por que no se puede subir ni quien lo arregla:\n%s", cuerpo)
	}
}

// UN DIRECTORIO SIN CAMPANA NO ES UN ERROR: ES EL ESTADO VACIO.
//
// Es la diferencia entre las dos formas de la nada aplicada al arranque. Si esto
// devolviera error, una instalacion recien descargada abriria la pantalla con un
// aviso rojo la primera vez, y un producto que empieza acusando no lo abre nadie
// dos veces.
func TestUnDirectorioDeAccesosVacioNoEsUnError(t *testing.T) {
	d := campanaEnDirectorio{dir: t.TempDir(), ahora: time.Now}
	c, err := d.Abierta()
	if err != nil {
		t.Fatalf("un directorio sin campana ha dado error: %v", err)
	}
	if c != nil {
		t.Fatalf("un directorio sin campana ha devuelto una campana")
	}
}

// UN SELLO QUE NO SE ENTIENDE NO COMPONE UNA RUTA.
//
// El sello sale del ledger, que es un fichero de disco que alguien pudo editar.
// Si el nombre del CSV se compusiera con lo que ponga ahi, un sello con `../`
// dentro leeria lo que ese alguien quiera. Se prueban las dos formas de la nada
// y la travesia, que son las tres que salen.
func TestUnSelloQueNoEsUnHashNoComponeUnaRuta(t *testing.T) {
	d := campanaEnDirectorio{dir: t.TempDir(), ahora: time.Now}
	for _, malo := range []string{"", "   ", "../../etc/passwd", "ZZZZ", strings.Repeat("a", 63)} {
		if _, err := d.rutaDelCenso(malo); err == nil {
			t.Errorf("el sello %q ha compuesto una ruta y no tenia que hacerlo", malo)
		}
	}
	bueno := strings.Repeat("ab", 32)
	ruta, err := d.rutaDelCenso(bueno)
	if err != nil {
		t.Fatalf("un sello valido ha sido rechazado: %v", err)
	}
	if filepath.Dir(ruta) != d.dir {
		t.Errorf("la ruta %q se sale del directorio %q", ruta, d.dir)
	}
}
