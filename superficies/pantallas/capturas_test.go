package pantallas

// LAS CAPTURAS: la puerta que convierte "me parece que se ve bien" en un fichero.
//
// # Que problema resuelve
//
// Todo lo demas de esta superficie se puede afirmar leyendo el HTML: que la
// barra sale, que el rotulo esta escrito, que el contraste de un par de colores
// da 4.5. Lo que NO se puede afirmar asi es si la pagina SE VE: una barra
// lateral que se come el contenido, una tabla que se sale, una pastilla ilegible
// sobre su fondo o un tema oscuro que deja un bloque en blanco son fallos que
// pasan por delante de axe y de todos los tests de subcadena de este fichero.
// Hasta aqui la unica revision de eso era mirar, y "me parece que se ve bien" no
// es un resultado que nadie pueda comprobar despues.
//
// # Que hace, exactamente
//
// Levanta el producto entero (las seis pantallas, el camino, el acta y la
// revision de accesos) sobre un servidor de pruebas, y le pide a Chrome sin
// ventana una captura de cada pantalla EN LOS DOS TEMAS. Las deja en capturas/
// con un nombre estable, para que dos ejecuciones se puedan comparar y para que
// una revision diga "esta" en vez de "creo".
//
// # Por que no corre en la suite normal
//
// Porque necesita un navegador instalado, y una puerta que dependa de eso se
// pondria roja en cualquier maquina sin Chrome y ensenaria a ignorarla. Se pide
// a mano:
//
//	PLAZUM_CAPTURAS=1 go test ./superficies/pantallas -run TestCapturas -v
//
// Y CUANDO NO SE PUEDE EJECUTAR, LO DICE. "No se pudo ejecutar" y "no encontro
// nada" son cosas distintas, y confundirlas hace que una maquina sin navegador
// se lea como una pantalla revisada. Sin la variable, se salta diciendolo; con
// la variable y sin navegador, FALLA, porque entonces alguien pidio capturas y
// no las ha tenido.

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	catalogoreal "github.com/marcosmatalab/plazum/adaptadores/catalogo"
	"github.com/marcosmatalab/plazum/nucleo/accesos"
	nucleoacta "github.com/marcosmatalab/plazum/nucleo/acta"
	"github.com/marcosmatalab/plazum/nucleo/auditoria"
	"github.com/marcosmatalab/plazum/nucleo/censo"
	"github.com/marcosmatalab/plazum/nucleo/incidente"
	"github.com/marcosmatalab/plazum/nucleo/ledger"
	sacta "github.com/marcosmatalab/plazum/superficies/acta"
	"github.com/marcosmatalab/plazum/superficies/camino"
	suar "github.com/marcosmatalab/plazum/superficies/uar"
)

// dondeVanLasCapturas es el directorio, relativo a este paquete.
const dondeVanLasCapturas = "capturas"

// temas son los dos que hay que mirar, y se miran los dos SIEMPRE.
//
// El tema oscuro no es una preferencia menor: es el que usa media plantilla de
// seguridad, y es donde se rompen los colores que se eligieron mirando el claro.
// Chrome lo fuerza con un ajuste de Blink (0 = oscuro, 1 = claro) en vez de con
// --force-dark-mode, que es otra cosa: aquel INVENTA un tema oscuro a partir del
// claro, y lo que aqui hay que fotografiar es el nuestro.
var temas = []struct {
	nombre  string
	ajustes string
}{
	{"claro", "preferredColorScheme=1"},
	{"oscuro", "preferredColorScheme=0"},
}

func TestCapturasDeCadaPantallaEnLosDosTemas(t *testing.T) {
	if os.Getenv("PLAZUM_CAPTURAS") == "" {
		t.Skip("capturas apagadas: se piden con PLAZUM_CAPTURAS=1. Necesitan un navegador " +
			"instalado, y una puerta que dependa de eso se pondria roja en cualquier " +
			"maquina sin el, que es como se ensena a ignorar una puerta")
	}
	navegador := buscarNavegador()
	if navegador == "" {
		// PEDIDAS Y NO TOMADAS ES UN FALLO, no un salto. Si esto se saltara,
		// una maquina sin navegador daria el mismo verde que una pantalla
		// revisada, que es el invariante 8 aplicado al propio arnes.
		t.Fatal("se han pedido capturas y no hay navegador. Arreglo: instalar Chrome o " +
			"Edge, o apuntar PLAZUM_NAVEGADOR al ejecutable")
	}

	srv := httptest.NewServer(productoEntero(t))
	defer srv.Close()

	dir, err := filepath.Abs(dondeVanLasCapturas)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	perfiles := t.TempDir()

	// LAS PANTALLAS. Estan TODAS las que tienen armazon, y ademas los estados
	// que un comprador ve el primer dia: la portada, la entrevista, la tabla, el
	// camino, el acta y la revision de accesos.
	pantallas := []struct {
		nombre string
		ruta   string
	}{
		{"hoy", "/hoy"},
		{"alcance", "/alcance"},
		{"controles", "/controles"},
		{"certificados", "/certificados"},
		{"camino", camino.BasePorDefecto + "/"},
		{"acta", "/acta/"},
		{"acta-derivacion", "/acta/derivacion/1.1.1"},
		{"uar", "/uar/"},
	}

	hechas := 0
	for _, p := range pantallas {
		for _, tema := range temas {
			destino := filepath.Join(dir, p.nombre+"-"+tema.nombre+".png")
			args := []string{
				"--headless=new", "--disable-gpu", "--no-sandbox",
				"--hide-scrollbars", "--force-device-scale-factor=1",
				"--window-size=1280,900",
				"--user-data-dir=" + filepath.Join(perfiles, p.nombre+tema.nombre),
				"--blink-settings=" + tema.ajustes,
				"--screenshot=" + destino,
				srv.URL + p.ruta,
			}
			// #nosec G204 -- el ejecutable sale de una lista fija o de una
			// variable de entorno del operador que corre el test, y la URL es
			// la del servidor de pruebas de este mismo proceso. Aqui no hay
			// entrada de un cliente.
			cmd := exec.Command(navegador, args...)
			salida, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("no se ha podido capturar %s (%s): %v\n%s",
					p.nombre, tema.nombre, err, salida)
				continue
			}
			inf, err := os.Stat(destino)
			if err != nil {
				t.Errorf("el navegador dijo que si y no hay fichero para %s (%s): %v",
					p.nombre, tema.nombre, err)
				continue
			}
			// UN PNG DE CUATRO BYTES ES UNA CAPTURA QUE NO EXISTE. Sin este
			// minimo, un navegador que falle a medias deja el fichero y esta
			// puerta da verde sobre una imagen vacia.
			if inf.Size() < 2000 {
				t.Errorf("la captura de %s (%s) pesa %d bytes: eso no es una pantalla",
					p.nombre, tema.nombre, inf.Size())
				continue
			}
			hechas++
		}
	}
	esperadas := len(pantallas) * len(temas)
	if hechas != esperadas {
		t.Fatalf("se han tomado %d capturas de %d", hechas, esperadas)
	}
	t.Logf("%d capturas (%d pantallas x %d temas) en %s", hechas, len(pantallas), len(temas), dir)
}

// buscarNavegador devuelve el ejecutable, o cadena vacia si no hay.
//
// Se mira PRIMERO la variable del operador: quien tenga el navegador en otro
// sitio no tiene que tocar este fichero.
func buscarNavegador() string {
	if v := strings.TrimSpace(os.Getenv("PLAZUM_NAVEGADOR")); v != "" {
		if _, err := os.Stat(v); err == nil {
			return v
		}
		return ""
	}
	candidatos := []string{"google-chrome", "chromium", "chromium-browser", "chrome"}
	if runtime.GOOS == "windows" {
		candidatos = append(candidatos,
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		)
	}
	for _, c := range candidatos {
		if strings.ContainsAny(c, `/\`) {
			if _, err := os.Stat(c); err == nil {
				return c
			}
			continue
		}
		if ruta, err := exec.LookPath(c); err == nil {
			return ruta
		}
	}
	return ""
}

// productoEntero monta las CUATRO superficies con pantalla igual que las monta
// `plazum serve`, para que la captura sea del producto y no de una maqueta.
func productoEntero(t *testing.T) http.Handler {
	t.Helper()
	// EL CATALOGO DE VERDAD, no el doble de los tests. El doble pinta
	// "[es:clave]" en cada rotulo, que es justo lo que hace falta para afirmar
	// cosas sobre claves y justo lo que NO sirve para mirar una pantalla: una
	// captura de eso no ensena el producto, ensena el arnes.
	cat, err := catalogoreal.Nuevo()
	if err != nil {
		t.Fatal(err)
	}

	app, _ := superficie(t, corpusDemo(), conCamino(), func(o *Opciones) { o.Catalogo = cat })

	cam, err := camino.Nuevo(camino.Opciones{
		Pasos: camino.Canonico(), Catalogo: cat,
		Base: camino.BasePorDefecto, Estatico: "/estatico",
	})
	if err != nil {
		t.Fatal(err)
	}

	act, err := sacta.Nuevo(sacta.Opciones{
		Fuente: actaFija{a: actaParaCapturas(t)}, Catalogo: cat,
		Base: "/acta", Estatico: "/estatico",
		CaminoRuta: camino.BasePorDefecto + "/", CaminoClave: camino.ClaveTitulo,
		Pasos: camino.Canonico(),
		Quien: func(*http.Request) string { return "ciso" },
	})
	if err != nil {
		t.Fatal(err)
	}

	rev, err := suar.Nuevo(suar.Opciones{
		Fuente: campanaFija{c: campanaParaCapturas(t)}, Catalogo: cat,
		Base: "/uar", Estatico: "/estatico",
		CaminoRuta: camino.BasePorDefecto + "/", CaminoClave: camino.ClaveTitulo,
		Tokens: func(*http.Request) (string, error) { return "tok-capturas", nil },
		Quien:  func(*http.Request) string { return "ciso" },
	})
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/acta/", act)
	mux.Handle("/uar/", rev)
	mux.Handle(camino.BasePorDefecto+"/", cam)
	mux.Handle("/", app)
	return mux
}

// actaFija y campanaFija son los dobles minimos de las dos fuentes.
type actaFija struct{ a nucleoacta.Acta }

func (f actaFija) Ultima() (nucleoacta.Acta, bool, error) { return f.a, true, nil }

type campanaFija struct{ c *accesos.Campana }

func (f campanaFija) Abierta() (*accesos.Campana, error) { return f.c, nil }
func (f campanaFija) Anotar(ledger.Entrada) error        { return nil }

func diaDeCaptura(a, m, d int) time.Time {
	return time.Date(a, time.Month(m), d, 12, 0, 0, 0, time.UTC)
}

// actaParaCapturas es un acta con las cuatro secciones vivas.
//
// Tiene datos a proposito: una captura del acta vacia no ensena lo que hay que
// revisar, que es como se ve el documento que alguien lleva a un consejo. El
// estado vacio ya lo cubren los tests de subcadena.
func actaParaCapturas(t *testing.T) nucleoacta.Acta {
	t.Helper()
	p, err := auditoria.Abrir("prog-1", auditoria.Ciclo{
		Nombre: "2026-2028", Desde: diaDeCaptura(2026, 1, 1), Hasta: diaDeCaptura(2028, 12, 31),
	}, []auditoria.Unidad{
		{Paquete: "p1", Version: "0.1", Obligacion: "o1", Titulo: "Copias"},
		{Paquete: "p1", Version: "0.1", Obligacion: "o2", Titulo: "Accesos"},
	}, auditoria.Arrastre{DeCiclo: "2023-2025", SinAuditar: map[string]int{"p1|o2": 2}})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Auditar(auditoria.Sesion{ID: "S1", Auditor: "aud-1",
		Cuando: diaDeCaptura(2026, 3, 1), Unidades: []string{"p1|o1"}}); err != nil {
		t.Fatal(err)
	}
	c := campanaParaCapturas(t)
	i1, err := incidente.Abrir("INC-1", diaDeCaptura(2026, 2, 1), diaDeCaptura(2026, 2, 2), "guardia")
	if err != nil {
		t.Fatal(err)
	}
	a, err := nucleoacta.Componer(nucleoacta.Entradas{
		ID: "acta-2026", Organizacion: "Molduras del Norte SL",
		Periodo:                 nucleoacta.Periodo{Desde: diaDeCaptura(2026, 1, 1), Hasta: diaDeCaptura(2026, 12, 31)},
		Programa:                p,
		Responsables:            map[string]string{"p1|o1": "aud-1"},
		Campana:                 c,
		HayRegistroDeIncidentes: true,
		Incidentes:              []*incidente.Incidente{i1},
		Esperadas: []nucleoacta.Notificacion{
			{Incidente: "INC-1", Hito: "n-inicial", Que: "notificacion inicial"},
		},
		QuienAsistio: []string{"Ana Perez"},
		Decisiones: []nucleoacta.Parrafo{{
			Frase: nucleoacta.Frase{Texto: "El consejo aprueba dos horas semanales para cerrar H1."},
			De:    nucleoacta.DeUnaPersona, Quien: "Ana Perez",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func campanaParaCapturas(t *testing.T) *accesos.Campana {
	t.Helper()
	ins, err := censo.Tomar([]byte(
		"usuario;nombre;permiso\n"+
			"u1;Ana Martinez;admin\n"+
			"u1;Ana Martinez;lector\n"+
			"u2;Luis Gil;lector\n"),
		censo.Opciones{Sistema: "erp", Fuente: "export", Quien: "u-042",
			Tomada: diaDeCaptura(2026, 6, 1), Retencion: "12 meses",
			Columnas: censo.ColumnasHabituales()})
	if err != nil {
		t.Fatal(err)
	}
	c, err := accesos.Abrir("uar-2026", ins, diaDeCaptura(2026, 6, 2),
		map[string]string{"erp|u1|admin": "jefa"})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Registrar(accesos.Decision{Fila: "erp|u1|admin", Veredicto: accesos.Aprobar,
		Quien: "jefa", Cuando: diaDeCaptura(2026, 6, 10)}); err != nil {
		t.Fatal(err)
	}
	return c
}
