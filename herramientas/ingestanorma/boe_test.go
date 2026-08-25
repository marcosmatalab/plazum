package main

// Los tests NO salen a la red. Todo lo que se prueba aqui son respuestas REALES
// del BOE recortadas a mano y guardadas en testdata/: el texto consolidado del
// Real Decreto 311/2022 (con la disposicion adicional segunda, que es el unico
// bloque de esa norma con dos versiones), la busqueda que resuelve su ELI, sus
// metadatos, y el articulo 12 de la Ley 34/2002, que esta derogado de verdad.
//
// Recortadas, no inventadas: un fixture escrito a mano prueba que el parser lee
// lo que yo creo que publica el BOE, no lo que el BOE publica.

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func leerFixture(t *testing.T, nombre string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", nombre))
	if err != nil {
		t.Fatalf("falta el fixture %s: %v", nombre, err)
	}
	return b
}

func metadatosDePrueba(t *testing.T) respuestaMetadatosBOE {
	t.Helper()
	var m respuestaMetadatosBOE
	if err := decodificarBOE(leerFixture(t, "boe-metadatos.xml"), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

// La fecha de referencia de los tests: un dia cualquiera posterior a todas las
// versiones del fixture. Fija a proposito, para que el test no cambie de
// resultado segun el dia en que se ejecute.
const hoyDePrueba = "20260826"

func articulosDePrueba(t *testing.T, fixture string, op opcionesBOE) []Articulo {
	t.Helper()
	meta := metadatosDePrueba(t)
	as, err := parsearTextoBOE(leerFixture(t, fixture), meta,
		"https://www.boe.es/eli/es/rd/2022/05/03/311/con", op)
	if err != nil {
		t.Fatalf("%s: %v", fixture, err)
	}
	return as
}

func porReferencia(as []Articulo, ref string) (Articulo, bool) {
	for _, a := range as {
		if a.Referencia == ref {
			return a, true
		}
	}
	return Articulo{}, false
}

func TestLosMetadatosDelBOESeLeenEnteros(t *testing.T) {
	m := metadatosDePrueba(t).Meta
	if m.Identificador == "" || m.Titulo == "" || m.URLELI == "" {
		t.Fatalf("faltan campos: %+v", m)
	}
	if m.FechaVigencia == "" || m.FechaActualizacion == "" {
		t.Fatal("sin fecha de vigencia y sin marca de actualizacion no hay vigilancia posible")
	}
	if m.EstatusDerogacion != "N" {
		t.Fatalf("la norma del fixture no esta derogada y el campo dice %q", m.EstatusDerogacion)
	}
	if got := citaCorta(m.Titulo); got != "Real Decreto 311/2022" {
		t.Fatalf("la cita corta salio %q", got)
	}
}

// El caso central: un articulo entero, con su titulo separado del encabezado, su
// cita compuesta y su enlace con ancla.
func TestUnArticuloSaleConSuCitaYSuAncla(t *testing.T) {
	as := articulosDePrueba(t, "boe-texto.xml", opcionesBOE{Referencia: hoyDePrueba})
	a, ok := porReferencia(as, "Artículo 31")
	if !ok {
		t.Fatalf("no salio el articulo 31; salieron %d: %v", len(as), referencias(as))
	}
	if a.Numero != "31" {
		t.Errorf("numero %q", a.Numero)
	}
	if a.Titulo != "Auditoría de la seguridad" {
		t.Errorf("titulo %q: el encabezado y el titulo tienen que separarse", a.Titulo)
	}
	if a.Cita != "Real Decreto 311/2022, art. 31" {
		t.Errorf("cita %q", a.Cita)
	}
	if a.Fuente != "https://www.boe.es/eli/es/rd/2022/05/03/311/con#a3-3" {
		t.Errorf("fuente %q: sin ancla al bloque, verificar una cita obliga a leer la norma "+
			"entera", a.Fuente)
	}
	if a.VigenciaDesde != "2022-05-05" {
		t.Errorf("vigencia_desde %q", a.VigenciaDesde)
	}
	if a.ModificadoPor != "" {
		t.Errorf("este articulo no lo ha tocado nadie y dice que si: %q", a.ModificadoPor)
	}
	if a.Derogado {
		t.Error("no esta derogado")
	}
	// El texto tiene que ser el texto, no el encabezado repetido.
	if strings.HasPrefix(a.Texto, "Artículo") {
		t.Errorf("el encabezado se ha colado en el texto: %.60s", a.Texto)
	}
	if !strings.Contains(a.Texto, "al menos cada dos años") {
		t.Errorf("falta la parte que da el reloj del articulo: %.200s", a.Texto)
	}
	if len(a.Parrafos) != 9 {
		t.Errorf("el articulo 31 tiene 9 parrafos en la fuente y salieron %d", len(a.Parrafos))
	}
	if a.Huella == "" || !strings.HasPrefix(a.Huella, "sha256:") {
		t.Errorf("huella %q", a.Huella)
	}
}

func referencias(as []Articulo) []string {
	out := make([]string, 0, len(as))
	for _, a := range as {
		out = append(out, a.Referencia)
	}
	return out
}

// Lo que NO sale por defecto: capitulos, firma y preambulo no son obligaciones y
// llenarian la vigilancia de ruido.
func TestPorDefectoNoSalenNiElCapituloNiLaFirma(t *testing.T) {
	as := articulosDePrueba(t, "boe-texto.xml", opcionesBOE{Referencia: hoyDePrueba})
	for _, ref := range []string{"CAPÍTULO I", "[firma]"} {
		if _, hay := porReferencia(as, ref); hay {
			t.Errorf("%q no es articulado y ha salido", ref)
		}
	}
	todos := articulosDePrueba(t, "boe-texto.xml", opcionesBOE{Referencia: hoyDePrueba, Todo: true})
	if len(todos) <= len(as) {
		t.Fatalf("-todo tiene que traer mas bloques: %d frente a %d", len(todos), len(as))
	}
	if _, hay := porReferencia(todos, "CAPÍTULO I"); !hay {
		t.Error("con -todo si sale el capitulo")
	}
}

// Un anexo llega como "encabezado", igual que un capitulo. Distinguirlos por el
// marcado (anexo_num) y no por el texto del titulo es lo que hace que el anexo
// no se pierda.
func TestUnAnexoSeDistingueDeUnCapitulo(t *testing.T) {
	as := articulosDePrueba(t, "boe-texto.xml", opcionesBOE{Referencia: hoyDePrueba})
	a, ok := porReferencia(as, "ANEXO III")
	if !ok {
		t.Fatalf("el anexo se ha perdido; salieron %v", referencias(as))
	}
	if a.Tipo != "anexo" {
		t.Errorf("tipo %q", a.Tipo)
	}
	if a.Titulo != "Auditoría de la seguridad" {
		t.Errorf("titulo del anexo %q", a.Titulo)
	}
	if a.Numero != "" {
		t.Errorf("un anexo no tiene numero de articulo y dice %q", a.Numero)
	}
	if a.Cita != "Real Decreto 311/2022, ANEXO III" {
		t.Errorf("cita %q", a.Cita)
	}
}

// La disposicion adicional segunda es el unico bloque de esta norma con dos
// versiones: la original y la de noviembre de 2024. Por defecto gana la vigente,
// y trae quien la modifico.
func TestGanaLaVersionVigenteYDiceQuienLaModifico(t *testing.T) {
	as := articulosDePrueba(t, "boe-texto.xml", opcionesBOE{Referencia: hoyDePrueba})
	a, ok := porReferencia(as, "Disposición adicional segunda")
	if !ok {
		t.Fatalf("salieron %v", referencias(as))
	}
	if a.VigenciaDesde != "2024-11-07" {
		t.Errorf("vigencia_desde %q: tenia que ganar la version de 2024", a.VigenciaDesde)
	}
	if a.ModificadoPor != "BOE-A-2024-22935" {
		t.Errorf("modificado_por %q", a.ModificadoPor)
	}
	if !strings.Contains(a.Nota, "Se modifica") {
		t.Errorf("la nota al pie de la fuente explica el cambio y salio %q", a.Nota)
	}
	if strings.Contains(a.Texto, "Se modifica por la disposición final") {
		t.Error("la nota al pie se ha colado en el texto normativo, que es lo que se transcribe")
	}
	if !strings.Contains(a.Texto, "Ministerio para la Transformación Digital") {
		t.Errorf("el texto no es el de la version de 2024: %.120s", a.Texto)
	}
}

// El viaje en el tiempo, que es lo que hace util al BOE frente a un scraping: a
// una fecha anterior a la modificacion sale el texto que estaba en vigor
// entonces. Es la misma propiedad bitemporal que el resto del producto.
func TestElTextoAUnaFechaPasadaEsElDeEntonces(t *testing.T) {
	as := articulosDePrueba(t, "boe-texto.xml", opcionesBOE{Referencia: "20230101"})
	a, ok := porReferencia(as, "Disposición adicional segunda")
	if !ok {
		t.Fatal("no salio la disposicion")
	}
	if a.VigenciaDesde != "2022-05-05" {
		t.Errorf("a 2023-01-01 la version en vigor era la de 2022 y salio %q", a.VigenciaDesde)
	}
	if a.ModificadoPor != "" {
		t.Errorf("en 2023 nadie la habia modificado todavia y dice %q", a.ModificadoPor)
	}
	if !strings.Contains(a.Texto, "Secretaría de Estado de Digitalización") {
		t.Errorf("no es el texto de 2022: %.120s", a.Texto)
	}
}

// Un articulo derogado de verdad: el art. 12 de la Ley 34/2002, que el BOE deja
// con el cuerpo reducido a "(Derogado)".
func TestUnArticuloDerogadoSeReconoce(t *testing.T) {
	meta := metadatosDePrueba(t)
	as, err := parsearTextoBOE(leerFixture(t, "boe-texto-derogado.xml"), meta,
		"https://www.boe.es/eli/es/l/2002/07/11/34/con", opcionesBOE{Referencia: hoyDePrueba})
	if err != nil {
		t.Fatal(err)
	}
	var derogados, vivos int
	for _, a := range as {
		if a.Derogado {
			derogados++
			if !strings.Contains(strings.ToLower(a.Nota), "se deroga") {
				t.Errorf("%s: la nota tiene que decir por que norma se derogo, y dice %q",
					a.Referencia, a.Nota)
			}
		} else {
			vivos++
		}
	}
	if derogados != 1 || vivos != 1 {
		t.Fatalf("el fixture tiene un derogado y uno vivo, y salieron %d y %d (%v)",
			derogados, vivos, referencias(as))
	}
}

// El control negativo del detector de derogados: un articulo que HABLA de normas
// derogadas no puede confundirse con uno derogado. Si esto se comprobara por
// contencion en vez de contra el cuerpo entero, medio corpus saldria derogado.
func TestHablarDeDerogacionNoEsEstarDerogado(t *testing.T) {
	caza := []string{"(Derogado)", "Derogado", " (derogada) ", "(Suprimido)"}
	calla := []string{
		"Queda derogado el Real Decreto 3/2010, de 8 de enero.",
		"1. Se deroga la disposicion adicional segunda del texto anterior.",
		"El presente reglamento no deroga ninguna norma anterior.",
	}
	for _, s := range caza {
		if !derogadoPorTexto(s) {
			t.Errorf("%q es un cuerpo derogado y no se caza", s)
		}
	}
	for _, s := range calla {
		if derogadoPorTexto(s) {
			t.Errorf("%q es texto normativo VIVO que habla de derogaciones, y se ha marcado "+
				"como derogado", s)
		}
	}
}

// --- el ELI ---

func TestElELISePartePorSusPiezas(t *testing.T) {
	p, err := partirELI("https://www.boe.es/eli/es/rd/2022/05/03/311/con")
	if err != nil {
		t.Fatal(err)
	}
	if p.Rango != "rd" || p.Ano != "2022" || p.Mes != "05" || p.Dia != "03" || p.Numero != "311" {
		t.Fatalf("piezas mal: %+v", p)
	}
	if p.Base != "https://www.boe.es/eli/es/rd/2022/05/03/311" {
		t.Fatalf("base %q: los sufijos de version se recortan", p.Base)
	}
	if got := urnSugeridoES(p.Rango, p.Ano, p.Numero); got != "urn:es:rd:2022:311" {
		t.Fatalf("urn sugerido %q", got)
	}
}

// LA FRONTERA LEGAL, hecha codigo. Solo fuente primaria: la licencia de un
// repositorio de terceros no alcanza al texto normativo que quien lo subio no
// poseia, diga MIT o diga Apache.
func TestSoloSeIngiereDeFuentePrimaria(t *testing.T) {
	fuera := []string{
		"https://raw.githubusercontent.com/alguien/leyes/main/eli/es/rd/2022/05/03/311",
		"https://boe.es.ejemplo.invalid/eli/es/rd/2022/05/03/311",
		"https://ejemplo.invalid/eli/es/rd/2022/05/03/311",
	}
	for _, u := range fuera {
		if _, err := partirELI(u); !errors.Is(err, ErrFuenteNoAutorizada) {
			t.Errorf("%s tenia que rechazarse por no ser fuente primaria y dio %v", u, err)
		}
	}
	for _, h := range []string{"www.boe.es", "boe.es", "publications.europa.eu", "eur-lex.europa.eu"} {
		if !anfitrionAutorizado(h) {
			t.Errorf("%s es fuente primaria y se rechaza", h)
		}
	}
	if anfitrionAutorizado("") {
		t.Error("un anfitrion vacio no es fuente primaria")
	}
}

func TestUnELIMalFormadoSeRechazaConSuArreglo(t *testing.T) {
	malos := []string{
		"https://www.boe.es/buscar/act.php?id=BOE-A-2022-7191", // no es un ELI
		"https://www.boe.es/eli/es/rd/2022/05",                 // le faltan piezas
		"https://www.boe.es/eli/es/rd/22/5/3/311",              // fecha mal formada
	}
	for _, u := range malos {
		_, err := partirELI(u)
		if !errors.Is(err, ErrIdentificadorInvalido) {
			t.Errorf("%s: se esperaba ErrIdentificadorInvalido y dio %v", u, err)
		}
		if err != nil && !strings.Contains(err.Error(), "Arreglo") {
			t.Errorf("%s: el error no dice como arreglarlo: %v", u, err)
		}
	}
}

// La resolucion del ELI a identificador se hace contra la API y casando el ELI
// EXACTO. Coger "el que mas se parece" es como se acaba ingiriendo otra norma.
func TestElELISeResuelvePorIgualdadYNoPorParecido(t *testing.T) {
	p, err := partirELI("https://www.boe.es/eli/es/rd/2022/05/03/311")
	if err != nil {
		t.Fatal(err)
	}
	id, err := resolverELIBOE(leerFixture(t, "boe-busqueda.xml"), p)
	if err != nil {
		t.Fatal(err)
	}
	if id != "BOE-A-2022-7191" {
		t.Fatalf("resolvio a %q", id)
	}
	// El mismo dia, otra norma: 309 esta en el fixture y no puede colarse.
	otro, err := partirELI("https://www.boe.es/eli/es/rd/2022/05/03/999")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolverELIBOE(leerFixture(t, "boe-busqueda.xml"), otro); !errors.Is(err, ErrNormaNoEncontrada) {
		t.Fatalf("un ELI que no esta en la respuesta tiene que dar ErrNormaNoEncontrada y dio %v", err)
	}
}

func TestElCodigoDeEstadoViajaDentroDelXML(t *testing.T) {
	// La API contesta HTTP 200 con un 404 dentro. Mirar solo el HTTP deja pasar
	// el vacio y lo guarda en el almacen como si fuera la norma.
	cuerpo := []byte(`<?xml version="1.0" encoding="utf-8"?><response><status>` +
		`<code>404</code><text>La informacion solicitada no existe</text></status><data/></response>`)
	var r respuestaTextoBOE
	if err := decodificarBOE(cuerpo, &r); !errors.Is(err, ErrNormaNoEncontrada) {
		t.Fatalf("se esperaba ErrNormaNoEncontrada y dio %v", err)
	}
	malo := []byte(`<html><body>no soy XML del BOE</body>`)
	if err := decodificarBOE(malo, &r); !errors.Is(err, ErrRespuestaIlegible) {
		t.Fatalf("se esperaba ErrRespuestaIlegible y dio %v", err)
	}
}

func TestUnIdentificadorDelBOEConBarrasNoSaleDelCamino(t *testing.T) {
	for _, id := range []string{
		"BOE-A-2022-7191/../../otra-cosa",
		"BOE-A-2022-7191?x=1",
		"OTRA-COSA-2022-1",
		"BOE",
	} {
		if err := validarIDBOE(id); err == nil {
			t.Errorf("%q tenia que rechazarse", id)
		}
	}
	if err := validarIDBOE("BOE-A-2022-7191"); err != nil {
		t.Errorf("un identificador normal se rechaza: %v", err)
	}
}

func TestUnTextoSinBloquesSeDenunciaYNoSeGuardaVacio(t *testing.T) {
	cuerpo := []byte(`<?xml version="1.0" encoding="utf-8"?><response><status><code>200</code>` +
		`<text>ok</text></status><data><texto></texto></data></response>`)
	_, err := parsearTextoBOE(cuerpo, metadatosDePrueba(t), "https://www.boe.es/x",
		opcionesBOE{Referencia: hoyDePrueba})
	if !errors.Is(err, ErrSinArticulos) {
		t.Fatalf("una norma sin articulos tiene que denunciarse: %v", err)
	}
}

// La tabla publica de vigilancia mezcla BOE y DOUE en la misma columna de
// "fecha de la fuente", asi que las dos marcas tienen que poder compararse.
func TestLaMarcaDeActualizacionDelBOESeNormalizaAISOExtendido(t *testing.T) {
	if got := instanteISO("20260803T080238Z"); got != "2026-08-03T08:02:38Z" {
		t.Errorf("%q", got)
	}
	for _, s := range []string{"", "2026-08-03T08:02:38Z", "20260803", "AAAAMMDDTHHmmSSZ"} {
		if got := instanteISO(s); got != s {
			t.Errorf("%q se convirtio en %q, y no habia de donde sacar un instante", s, got)
		}
	}
	if got := origenDeMetadatos(metadatosDePrueba(t), piezasELI{}, "u", "").ActualizadaEn; got != "2026-08-03T08:02:38Z" {
		t.Errorf("la ficha no normaliza la marca: %q", got)
	}
}

func TestLasFechasDelBOESePasanAGuiones(t *testing.T) {
	if got := guion("20220505"); got != "2022-05-05" {
		t.Errorf("%q", got)
	}
	// Lo que no parece una fecha se devuelve tal cual: no se inventa.
	for _, s := range []string{"", "2022", "AAAAMMDD", "2022-05-05"} {
		if got := guion(s); got != s {
			t.Errorf("%q se convirtio en %q, y no habia de donde sacar una fecha", s, got)
		}
	}
}
