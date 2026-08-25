package main

// La herramienta entera, de la linea de comandos a la tabla, SIN RED. El cliente
// se sustituye por uno que sirve los fixtures de testdata/, asi que estos tests
// prueban el recorrido completo (resolver el ELI, bajar metadatos, bajar texto,
// comparar con lo anterior, registrar y pintar) sin depender de que el BOE este
// levantado ni de que la norma no cambie manana.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// falsaRed sirve los fixtures. Guarda tambien que URL se pidio, que es como se
// comprueba que no se descarga lo que ya se tiene.
type falsaRed struct {
	t         *testing.T
	textoBOE  string // que fixture de texto devolver
	pedidas   []string
	sinTexto  bool
	respuesta map[string][]byte
}

func (f *falsaRed) obtener(u string, cab map[string]string) ([]byte, error) {
	f.pedidas = append(f.pedidas, u)
	if f.respuesta != nil {
		for marca, b := range f.respuesta {
			if strings.Contains(u, marca) {
				return b, nil
			}
		}
	}
	switch {
	case strings.Contains(u, "query="):
		return leerFixture(f.t, "boe-busqueda.xml"), nil
	case strings.HasSuffix(u, "/metadatos"):
		return leerFixture(f.t, "boe-metadatos.xml"), nil
	case strings.HasSuffix(u, "/texto"):
		if f.sinTexto {
			return nil, fmt.Errorf("%w: no tendria que haberse pedido el texto", ErrNormaNoEncontrada)
		}
		return leerFixture(f.t, f.textoBOE), nil
	case strings.Contains(u, "/resource/celex/"):
		if cab["Accept"] == "application/xhtml+xml" {
			return leerFixture(f.t, "eurlex-original.xhtml"), nil
		}
		return leerFixture(f.t, "eurlex-noticia.xml"), nil
	}
	return nil, fmt.Errorf("la falsa red no sabe servir %q", u)
}

func clienteDe(f *falsaRed) *cliente {
	return &cliente{Local: f.obtener}
}

// relojFijo evita que un test cambie de resultado segun el dia. La herramienta
// necesita el instante para saber a que fecha esta vigente un bloque y para
// anotar el historial; entra como dato, no como time.Now() escondido.
func relojFijo(t time.Time) func() time.Time { return func() time.Time { return t } }

var momento = time.Date(2026, 8, 26, 9, 30, 0, 0, time.UTC)

const eliDePrueba = "https://www.boe.es/eli/es/rd/2022/05/03/311"

func correr(t *testing.T, op opciones, red *falsaRed) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	op.ClienteParaPruebas = clienteDe(red)
	err := ejecutar(op, &buf, relojFijo(momento))
	return buf.String(), err
}

// El recorrido completo: de un ELI a la tabla, pasando por la resolucion del
// identificador contra la API.
func TestDeUnELIALaTablaSinTocarLaRed(t *testing.T) {
	alm := t.TempDir()
	red := &falsaRed{t: t, textoBOE: "boe-texto-antes.xml"}
	salida, err := correr(t, opciones{ELI: eliDePrueba, Almacen: alm}, red)
	if err != nil {
		t.Fatal(err)
	}
	for _, quiero := range []string{
		"BOE-A-2022-7191",              // el ELI se resolvio a su identificador
		"urn:es:rd:2022:311",           // y se propone un urn de paquete
		"Artículo 31",                  // el articulado sale
		"primera vez",                  // la vigilancia dice donde esta
		"13 TRLPI",                     // la licencia de la fuente viaja con el dato
		"meramente informativo",        // y la atribucion que el BOE exige
		"no es asesoramiento juridico", // y el descargo
	} {
		if !strings.Contains(salida, quiero) {
			t.Errorf("la salida no dice %q:\n%s", quiero, salida)
		}
	}
	// Y ha dejado el almacen listo para la proxima vez.
	inst := filepath.Join(alm, "es-boe-a-2022-7191", "instantanea.json")
	if _, err := os.Stat(inst); err != nil {
		t.Fatalf("no se ha guardado la instantanea: %v", err)
	}
	if _, err := os.Stat(filepath.Join(alm, "es-boe-a-2022-7191", "historial.jsonl")); err != nil {
		t.Fatalf("no se ha anotado el historial: %v", err)
	}
}

// LA PRUEBA DE LA VIGILANCIA VISTA POR QUIEN LA USA: se ejecuta, se vuelve a
// ejecutar sobre la norma ya modificada, y la herramienta lo dice. Y a la
// tercera, sin cambios, se calla.
func TestReejecutarDiceQueHaCambiadoYLuegoSeCalla(t *testing.T) {
	alm := t.TempDir()
	if _, err := correr(t, opciones{ELI: eliDePrueba, Almacen: alm},
		&falsaRed{t: t, textoBOE: "boe-texto-antes.xml"}); err != nil {
		t.Fatal(err)
	}
	salida, err := correr(t, opciones{ELI: eliDePrueba, Almacen: alm},
		&falsaRed{t: t, textoBOE: "boe-texto.xml"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(salida, "MODIFICADO  Disposición adicional segunda") {
		t.Fatalf("HALLAZGO: la norma cambio y la segunda ejecucion no lo dice:\n%s", salida)
	}
	if !strings.Contains(salida, "BOE-A-2024-22935") {
		t.Errorf("y tiene que decir QUE norma lo modifico:\n%s", salida)
	}
	tercera, err := correr(t, opciones{ELI: eliDePrueba, Almacen: alm},
		&falsaRed{t: t, textoBOE: "boe-texto.xml"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tercera, "sin cambios") {
		t.Fatalf("HALLAZGO: nada ha cambiado y la tercera ejecucion sigue avisando:\n%s", tercera)
	}
	if strings.Contains(tercera, "MODIFICADO") {
		t.Errorf("y no puede repetir el aviso de la vez anterior:\n%s", tercera)
	}
}

// Registrar una extraccion PARCIAL envenenaria el almacen: la siguiente
// ejecucion completa veria el resto de la norma como articulos nuevos.
func TestUnaExtraccionParcialNoSeRegistra(t *testing.T) {
	alm := t.TempDir()
	salida, err := correr(t, opciones{ELI: eliDePrueba, Almacen: alm, Articulos: "31"},
		&falsaRed{t: t, textoBOE: "boe-texto.xml"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(salida, "no se compara ni se registra") {
		t.Errorf("tiene que decir que no registra y por que:\n%s", salida)
	}
	if _, err := os.Stat(filepath.Join(alm, "es-boe-a-2022-7191")); err == nil {
		t.Fatal("HALLAZGO: una extraccion parcial ha entrado en el almacen. La proxima " +
			"ejecucion completa veria el resto de la norma como articulos nuevos")
	}
	// Y ha filtrado de verdad.
	if strings.Contains(salida, "ANEXO III") {
		t.Errorf("-articulos 31 no filtro:\n%s", salida)
	}
}

// Igual con el texto a una fecha pasada: registrarlo haria retroceder la
// vigilancia y la siguiente ejecucion cantaria un cambio que no existio.
func TestElTextoAUnaFechaPasadaNoSeRegistra(t *testing.T) {
	alm := t.TempDir()
	salida, err := correr(t, opciones{ELI: eliDePrueba, Almacen: alm, AFecha: "2023-01-01"},
		&falsaRed{t: t, textoBOE: "boe-texto.xml"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(salida, "fecha pasada") {
		t.Errorf("tiene que decir por que no registra:\n%s", salida)
	}
	if _, err := os.Stat(filepath.Join(alm, "es-boe-a-2022-7191")); err == nil {
		t.Fatal("una extraccion a una fecha pasada ha entrado en el almacen")
	}
	if !strings.Contains(salida, "Secretaría de Estado de Digitalización") &&
		!strings.Contains(salida, "texto vigente a 2023-01-01") {
		t.Errorf("y tiene que avisar de que no es la ultima version:\n%s", salida)
	}
}

func TestUnArticuloQueNoExisteEsUnErrorYNoUnaListaVacia(t *testing.T) {
	_, err := correr(t, opciones{ELI: eliDePrueba, Almacen: t.TempDir(), Articulos: "999"},
		&falsaRed{t: t, textoBOE: "boe-texto.xml"})
	if err == nil {
		t.Fatal("HALLAZGO: pedir un articulo que no existe devuelve una lista vacia sin decir " +
			"nada, que es como uno se cree que el articulo no existe")
	}
	if !strings.Contains(err.Error(), "Arreglo") {
		t.Errorf("el error no dice como arreglarlo: %v", err)
	}
}

func TestHayQueDarExactamenteUnaNorma(t *testing.T) {
	red := &falsaRed{t: t, textoBOE: "boe-texto.xml", sinTexto: true}
	for _, op := range []opciones{
		{Almacen: t.TempDir()},
		{ELI: eliDePrueba, CELEX: "32016R0679", Almacen: t.TempDir()},
		{ID: "BOE-A-2022-7191", CELEX: "32016R0679", Almacen: t.TempDir()},
	} {
		if _, err := correr(t, op, red); err == nil {
			t.Errorf("%+v tenia que rechazarse", op)
		}
	}
	// -fecha con el DOUE: el DOUE no sirve todas las versiones, y callarselo
	// daria la ultima version haciendo creer que es la de esa fecha.
	if _, err := correr(t, opciones{CELEX: "32016R0679", AFecha: "2020-01-01", Almacen: t.TempDir()}, red); err == nil {
		t.Error("-fecha con -celex tiene que rechazarse, no ignorarse en silencio")
	}
}

func TestLaSalidaJSONEsLaExtraccionCompleta(t *testing.T) {
	salida, err := correr(t, opciones{ELI: eliDePrueba, Almacen: t.TempDir(), JSON: true},
		&falsaRed{t: t, textoBOE: "boe-texto.xml"})
	if err != nil {
		t.Fatal(err)
	}
	var e Extraccion
	if err := json.Unmarshal([]byte(salida), &e); err != nil {
		t.Fatalf("la salida JSON no es JSON: %v\n%s", err, salida)
	}
	if e.Esquema != EsquemaIngesta {
		t.Errorf("esquema %q", e.Esquema)
	}
	if e.LicenciaFuente == "" || e.Atribucion == "" {
		t.Fatal("HALLAZGO: licencia_fuente y atribucion son obligatorios y han salido vacios")
	}
	if e.Fuente.URLDocumento == "" || e.Fuente.URLDatos == "" {
		t.Errorf("faltan las dos urls: %+v", e.Fuente)
	}
	if len(e.Articulos) == 0 || e.Articulos[0].Cita == "" || e.Articulos[0].Fuente == "" {
		t.Errorf("los articulos salen sin cita o sin enlace: %+v", e.Articulos)
	}
	// Los dos campos tienen que estar en el JOSN aunque estuvieran vacios: son
	// obligatorios del formato, no opcionales que aparecen si hay suerte.
	if !strings.Contains(salida, `"licencia_fuente"`) || !strings.Contains(salida, `"atribucion"`) {
		t.Error("los campos obligatorios no pueden llevar omitempty")
	}
}

func TestElCELEXRecorreLaFichaYElTexto(t *testing.T) {
	red := &falsaRed{t: t}
	salida, err := correr(t, opciones{CELEX: "32016R0679", Almacen: t.TempDir()}, red)
	if err != nil {
		t.Fatal(err)
	}
	for _, quiero := range []string{
		"Reglamento (UE) 2016/679",
		"https://eur-lex.europa.eu/eli/reg/2016/679/oj",
		"urn:eu:reg:2016:679",
		"Artículo 33",
		"2011/833", // la licencia que autoriza transcribir el DOUE
		"Union Europea",
	} {
		if !strings.Contains(salida, quiero) {
			t.Errorf("la salida no dice %q:\n%s", quiero, salida)
		}
	}
	if len(red.pedidas) != 2 {
		t.Errorf("son dos peticiones (ficha y texto) y se hicieron %d: %v", len(red.pedidas), red.pedidas)
	}
}

func TestElHistorialSeImprimeYSeSirveEnJSON(t *testing.T) {
	alm := t.TempDir()
	if _, err := correr(t, opciones{ELI: eliDePrueba, Almacen: alm},
		&falsaRed{t: t, textoBOE: "boe-texto-antes.xml"}); err != nil {
		t.Fatal(err)
	}
	if _, err := correr(t, opciones{ELI: eliDePrueba, Almacen: alm},
		&falsaRed{t: t, textoBOE: "boe-texto.xml"}); err != nil {
		t.Fatal(err)
	}
	tabla, err := correr(t, opciones{Historial: true, Almacen: alm}, &falsaRed{t: t})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tabla, "BOE-A-2022-7191") || !strings.Contains(tabla, "2 observaciones") {
		t.Errorf("la tabla de vigilancia no cuadra:\n%s", tabla)
	}
	crudo, err := correr(t, opciones{Historial: true, JSON: true, Almacen: alm}, &falsaRed{t: t})
	if err != nil {
		t.Fatal(err)
	}
	var entradas []Entrada
	if err := json.Unmarshal([]byte(crudo), &entradas); err != nil {
		t.Fatalf("el historial en JSON no es JSON: %v", err)
	}
	if len(entradas) != 2 {
		t.Fatalf("dos observaciones y salieron %d", len(entradas))
	}
	if entradas[0].Cambios.FuenteAhora == "" {
		t.Error("sin la fecha que declara la fuente no se puede pintar la tabla publica")
	}
	// Un almacen vacio no revienta: dice que esta vacio.
	vacio, err := correr(t, opciones{Historial: true, Almacen: filepath.Join(t.TempDir(), "nada")},
		&falsaRed{t: t})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(vacio, "vacio") {
		t.Errorf("un almacen vacio tiene que decirlo:\n%s", vacio)
	}
}

// La frontera legal tambien en el cliente: una URL de un anfitrion que no es
// fuente primaria no se descarga, aunque se cuele por otro camino.
func TestElClienteNoDescargaDeFueraDeLaFuentePrimaria(t *testing.T) {
	c := nuevoCliente(t.TempDir(), true)
	_, err := c.obtener("https://raw.githubusercontent.com/alguien/leyes/main/boe.xml", nil)
	if !errors.Is(err, ErrFuenteNoAutorizada) {
		t.Fatalf("se esperaba ErrFuenteNoAutorizada y dio %v", err)
	}
	if c.Peticiones != 0 {
		t.Error("y no ha llegado a pedirlo")
	}
}

func TestLaFechaDeReferenciaSeValida(t *testing.T) {
	if _, err := fechaReferencia("2023-13-45", momento); err == nil {
		t.Error("una fecha imposible tiene que rechazarse")
	}
	if _, err := fechaReferencia("hace dos anos", momento); err == nil {
		t.Error("lo que no es una fecha tiene que rechazarse")
	}
	got, err := fechaReferencia("", momento)
	if err != nil || got != "20260826" {
		t.Errorf("sin -fecha se usa el instante que entra como dato: %q %v", got, err)
	}
	if got, _ := fechaReferencia("2023-01-01", momento); got != "20230101" {
		t.Errorf("%q", got)
	}
}
