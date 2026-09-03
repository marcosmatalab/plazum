package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/adaptadores/latido"
)

// La orden `plazum latido`, probada por donde de verdad se usa: el CODIGO DE
// SALIDA.
//
// Un mensaje en la terminal no despierta a nadie de madrugada. Lo que engancha
// esto al monitor que el cliente ya tiene es que salga con 1, y eso es lo que
// hay que vigilar. Todo lo de aqui corre sin salir de la maquina: el receptor
// de pruebas lo levanta httptest.

func correr(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var salida, errores bytes.Buffer
	rc := cmdLatido(args, &salida, &errores)
	return rc, salida.String(), errores.String()
}

// EL AVISO DE LAS 24 HORAS, que es la casilla entera. Sin red, con el reloj del
// operador, y con codigo 1 cuando el planificador lleva mas de un dia callado.
func TestLatidoSaleConCodigo1CuandoElPlanificadorLlevaUnDiaCallado(t *testing.T) {
	dir := t.TempDir()
	if err := latido.Guardar(dir, latido.Estado{
		UltimoCiclo: enRFC3339(t, "2026-08-24T09:00:00Z"),
	}); err != nil {
		t.Fatal(err)
	}

	// Un minuto antes de las 24 horas: verde.
	rc, salida, _ := correr(t, "--datos", dir, "--ahora", "2026-08-25T08:59:00Z")
	if rc != 0 {
		t.Errorf("a las 23h59m el comando sale con %d y tenia que salir con 0.\n%s", rc, salida)
	}

	// Pasadas las 24 horas: rojo, y el mensaje dice lo que pasa.
	rc, salida, _ = correr(t, "--datos", dir, "--ahora", "2026-08-25T09:00:00Z")
	if rc != 1 {
		t.Fatalf("con el planificador callado 24 horas el comando sale con %d.\n"+
			"  Sin codigo 1 no hay forma de enganchar esto a un cron, y entonces el aviso\n"+
			"  solo existe para quien entre a mirar la pantalla.\n%s", rc, salida)
	}
	if !strings.Contains(salida, "ROTO") {
		t.Errorf("la salida no dice que esta roto:\n%s", salida)
	}
	for _, quiero := range []string{"vencen sin que nadie los mire", "plazum latido ciclo"} {
		if !strings.Contains(salida, quiero) {
			t.Errorf("la salida no contiene %q, o sea que no dice que pasa o como se "+
				"arregla:\n%s", quiero, salida)
		}
	}
}

// Con el latido apagado, que es como viene de fabrica, el comando NO sale a la
// red y sigue dando el veredicto. Es el caso de casi todo el mundo.
func TestLatidoFuncionaConElPulsoApagado(t *testing.T) {
	dir := t.TempDir()
	rc, salida, errores := correr(t, "--datos", dir, "--ahora", "2026-08-26T09:00:00Z")
	if rc != 0 {
		t.Fatalf("sin fichero de estado el comando sale con %d.\n%s\n%s", rc, salida, errores)
	}
	// El pulso apagado se informa como CORRECTO, no como un problema: es el
	// valor de fabrica y la postura correcta. Un producto que pinta en
	// amarillo el hecho de no haber activado la telemetria esta empujando a
	// activarla, que es lo que hace todo el mundo y por lo que nadie se fia.
	if !strings.Contains(salida, "pulso         CORRECTO") ||
		!strings.Contains(salida, "apagado") {
		t.Errorf("no dice que el pulso esta apagado, o lo pinta como un problema:\n%s", salida)
	}
	if _, err := os.Stat(filepath.Join(dir, latido.NombreDelFichero)); err == nil {
		t.Error("consultar el estado ha creado el fichero de estado")
	}
}

// Activar tiene DOS PASOS. Sin --acepto se ensena lo que se mandaria y no se
// activa nada, igual que `plazum update` no actualiza sin --aplicar.
//
// Un consentimiento que se da antes de poder leer lo que se acepta no es un
// consentimiento, y esta es la unica forma de garantizar que las dos lineas de
// QueSeManda han pasado por delante de los ojos del operador.
func TestActivarSinAceptarNoActivaNadaYEnsenaLoQueSeMandaria(t *testing.T) {
	dir := t.TempDir()
	rc, salida, _ := correr(t, "activar", "--datos", dir)
	if rc != 0 {
		t.Fatalf("activar sin --acepto sale con %d", rc)
	}
	// Las dos lineas de la declaracion, enteras.
	for _, l := range strings.Split(latido.QueSeManda, "\n") {
		if !strings.Contains(salida, l) {
			t.Errorf("la salida no ensena la linea %q de lo que se manda:\n%s", l, salida)
		}
	}
	if !strings.Contains(salida, latido.DestinoPorDefecto) {
		t.Errorf("no dice a donde iria el pulso:\n%s", salida)
	}
	if !strings.Contains(salida, "--acepto") {
		t.Errorf("no dice el comando exacto para encenderlo:\n%s", salida)
	}
	if _, err := os.Stat(filepath.Join(dir, latido.NombreDelFichero)); err == nil {
		t.Fatal("activar SIN --acepto ha escrito el estado: el opt-in se estaria dando por " +
			"dado antes de que nadie lo acepte")
	}

	e, err := latido.Cargar(dir)
	if err != nil || e.Activado {
		t.Fatalf("el latido ha quedado activado sin --acepto: %+v (%v)", e, err)
	}
}

// Con --acepto se enciende y el consentimiento queda escrito.
func TestActivarConAceptoEnciendeYRegistraElConsentimiento(t *testing.T) {
	dir := t.TempDir()
	rc, salida, errores := correr(t, "activar", "--datos", dir, "--acepto",
		"--ahora", "2026-08-26T09:00:00Z")
	if rc != 0 {
		t.Fatalf("activar --acepto sale con %d: %s", rc, errores)
	}
	e, err := latido.Cargar(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !e.Activado || e.Consentimiento == nil || e.Consentimiento.Texto != latido.QueSeManda {
		t.Fatalf("el consentimiento no ha quedado registrado: %+v", e)
	}
	if !strings.Contains(salida, "desactivar") {
		t.Errorf("no dice como se apaga:\n%s", salida)
	}

	// Y desactivar borra el identificador.
	if rc, _, errores = correr(t, "desactivar", "--datos", dir); rc != 0 {
		t.Fatalf("desactivar sale con %d: %s", rc, errores)
	}
	e, err = latido.Cargar(dir)
	if err != nil {
		t.Fatal(err)
	}
	if e.Activado || e.Instancia != "" {
		t.Errorf("tras desactivar queda %+v", e)
	}
}

// LA DIRECCION DEL AVISO, en el sitio donde mas caro sale equivocarse: el cron.
//
// `plazum latido ciclo` apunta el ciclo y sale con 0 AUNQUE EL PULSO FALLE. Si
// saliera con 1, el cron del cliente le mandaria un correo diciendo que plazum
// falla cada vez que a nosotros se nos caiga el receptor, y en dos semanas
// habria filtrado esos correos a la papelera. Junto con ellos, el dia que su
// planificador se pare de verdad.
func TestElCicloSaleVerdeAunqueNuestroReceptorEsteCaido(t *testing.T) {
	caido := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer caido.Close()

	dir := t.TempDir()
	if rc, _, errores := correr(t, "activar", "--datos", dir, "--destino", caido.URL+"/latido",
		"--acepto", "--ahora", "2026-08-26T09:00:00Z"); rc != 0 {
		t.Fatalf("activar contra el receptor de pruebas sale con %d: %s", rc, errores)
	}

	rc, salida, errores := correr(t, "ciclo", "--datos", dir, "--ahora", "2026-08-26T10:00:00Z")
	if rc != 0 {
		t.Fatalf("con NUESTRO receptor caido, el ciclo del cliente sale con %d.\n"+
			"  Nuestra caida se estaria convirtiendo en su alarma.\n%s\n%s",
			rc, salida, errores)
	}
	if !strings.Contains(errores, "aviso") {
		t.Errorf("el fallo del canal no se avisa siquiera:\n%s", errores)
	}
	e, err := latido.Cargar(dir)
	if err != nil {
		t.Fatal(err)
	}
	if e.UltimoCiclo.IsZero() {
		t.Fatal("el ciclo no ha quedado apuntado porque el pulso fallo: manana esto diria " +
			"que su planificador lleva un dia muerto")
	}
	if !e.FalloElUltimoIntento {
		t.Error("el fallo del canal no ha quedado apuntado")
	}

	// Y el veredicto que se imprime: planificador CORRECTO, pulso en aviso.
	if !strings.Contains(salida, "planificador  CORRECTO") {
		t.Errorf("con el ciclo recien corrido, el planificador no sale correcto:\n%s", salida)
	}
	if !strings.Contains(salida, "pulso         AVISO") {
		t.Errorf("con el receptor caido, el pulso no sale en aviso:\n%s", salida)
	}
}

// El smoke test del canal SI sale con 1 cuando el canal no entrega: el operador
// ha preguntado por el canal, asi que la respuesta es sobre el canal.
func TestProbarSaleConCodigo1CuandoElCanalNoEntrega(t *testing.T) {
	caido := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer caido.Close()
	bueno := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer bueno.Close()

	dir := t.TempDir()
	if rc, _, e := correr(t, "activar", "--datos", dir, "--destino", caido.URL+"/latido",
		"--acepto", "--ahora", "2026-08-26T09:00:00Z"); rc != 0 {
		t.Fatalf("activar sale con %d: %s", rc, e)
	}
	rc, _, errores := correr(t, "probar", "--datos", dir, "--ahora", "2026-08-26T09:01:00Z")
	if rc != 1 {
		t.Fatalf("probar contra un receptor que contesta 502 sale con %d", rc)
	}
	if !strings.Contains(errores, "no toca a tus plazos") {
		t.Errorf("el fallo del canal no dice que esto no afecta a los plazos, que es lo "+
			"unico que el operador necesita saber en ese momento:\n%s", errores)
	}

	// Y contra un receptor que si acepta, verde y con la marca escrita.
	if rc, _, e := correr(t, "activar", "--datos", dir, "--destino", bueno.URL+"/latido",
		"--acepto", "--ahora", "2026-08-26T09:02:00Z"); rc != 0 {
		t.Fatalf("reactivar contra el receptor bueno sale con %d: %s", rc, e)
	}
	rc, salida, errores := correr(t, "probar", "--datos", dir, "--ahora", "2026-08-26T09:03:00Z")
	if rc != 0 {
		t.Fatalf("probar contra un receptor que acepta sale con %d: %s", rc, errores)
	}
	if !strings.Contains(salida, "el canal entrega") {
		t.Errorf("no dice que el canal entrega:\n%s", salida)
	}
}

// Probar con el pulso apagado no es un fallo del canal: es que no hay canal.
// Se distingue, porque el arreglo es distinto.
func TestProbarConElPulsoApagadoLoDiceEnVezDeFallar(t *testing.T) {
	dir := t.TempDir()
	rc, _, errores := correr(t, "probar", "--datos", dir)
	if rc != 2 {
		t.Errorf("probar con el pulso apagado sale con %d y esperaba 2 (uso)", rc)
	}
	if !strings.Contains(errores, "activar") {
		t.Errorf("no dice como se enciende:\n%s", errores)
	}
}

// Una orden que no existe se dice, con la lista de las que hay.
func TestUnaOrdenDeLatidoQueNoExisteSeDice(t *testing.T) {
	rc, _, errores := correr(t, "apagalotodo")
	if rc != 2 {
		t.Errorf("una orden inexistente sale con %d", rc)
	}
	if !strings.Contains(errores, "no existe") || !strings.Contains(errores, "plazum latido ciclo") {
		t.Errorf("no dice que ordenes hay:\n%s", errores)
	}
}

// enRFC3339 parsea un instante fijo para las marcas de los tests. Los tests de
// aqui no usan el reloj de la maquina en ningun sitio: si lo usaran, el que
// mide las 24 horas pasaria de dia y fallaria de noche.
func enRFC3339(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return v.UTC()
}

// EL CIERRE DEL CIRCUITO: lo que escribe la orden `plazum latido ciclo` es lo
// que lee la pantalla Hoy del servidor de verdad.
//
// Sin esto, las dos mitades estan cada una en verde y no se hablan: el operador
// programa el ciclo en su cron, entra en Hoy, y la pantalla le sigue diciendo
// que su planificador no ha corrido nunca. Es el mismo fallo por el que `plazum
// serve` estuvo implementado y sin aparecer en la lista de ordenes.
//
// Se comprueba arrancando el binario y pidiendole la pagina por la red, que es
// como la pide el navegador, y contra el catalogo de verdad: el texto que se
// busca es el que leeria el comprador.
func TestLaPantallaHoyDelServidorLeeLasMarcasDelPlanificador(t *testing.T) {
	dir := t.TempDir()
	if err := latido.Guardar(dir, latido.Estado{
		UltimoCiclo: time.Now().UTC().Add(-48 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	// INSTALADO, y no solo arrancado. Sin administrador, /hoy redirige a la
	// instalacion y lo que se leeria abajo seria el formulario de primer
	// administrador: este test daria por buena una pantalla Hoy que nadie
	// estaria sirviendo.
	s := arrancarServeInstalado(t, "--datos", dir)
	base := s.base

	cuerpo := pedirPagina(t, base+"/hoy", s.cli)
	if !strings.Contains(cuerpo, "sin correr un ciclo") {
		t.Errorf("con el planificador parado dos dias, Hoy no lo dice.\n"+
			"  Lo que escribe el ciclo y lo que lee la pantalla no se estan hablando: el\n"+
			"  operador programa el cron y la pantalla le sigue diciendo que no ha corrido\n"+
			"  nunca.\n%s", recorte(cuerpo))
	}

	// CONTROL NEGATIVO: con el ciclo recien corrido, la pagina NO dice eso, y
	// dice que esta vivo. Sin esto, una pagina que dijera siempre lo mismo
	// pasaria la comprobacion de arriba.
	if err := latido.Guardar(dir, latido.Estado{
		UltimoCiclo: time.Now().UTC().Add(-time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	cuerpo = pedirPagina(t, base+"/hoy", s.cli)
	if strings.Contains(cuerpo, "sin correr un ciclo") {
		t.Errorf("con el ciclo recien corrido, Hoy sigue diciendo que esta parado.\n%s",
			recorte(cuerpo))
	}
	if !strings.Contains(cuerpo, "El planificador est") {
		t.Errorf("con el ciclo recien corrido, Hoy no dice que el planificador esta vivo.\n%s",
			recorte(cuerpo))
	}
}

// pedirPagina pide una pagina CON la sesion del que la pide. El cliente entra
// por parametro y no se construye aqui: desde que plazum tiene puerta, un
// cliente sin cookies no ve las pantallas, y uno construido a escondidas dentro
// del helper haria que el test midiera la pagina de instalacion sin enterarse.
func pedirPagina(t *testing.T, url string, cli *http.Client) string {
	t.Helper()
	resp, err := cli.Get(url) // #nosec G107 -- la direccion es la del servidor que levanta el propio test
	if err != nil {
		t.Fatalf("%s: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s ha contestado %d", url, resp.StatusCode)
	}
	return string(b)
}

// recorte deja el trozo de la pagina donde vive el veredicto, para que un fallo
// no vuelque cien kilobytes de HTML en la salida del test.
func recorte(cuerpo string) string {
	i := strings.Index(cuerpo, "vigilancia")
	if i < 0 {
		return cuerpo[:min(len(cuerpo), 400)]
	}
	return cuerpo[i:min(len(cuerpo), i+700)]
}
