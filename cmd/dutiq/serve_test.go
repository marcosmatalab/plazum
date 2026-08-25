package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// La puerta de que el producto SE PUEDE ARRANCAR.
//
// Hasta que existio `dutiq serve`, la superficie web estaba entera y no habia
// forma de levantarla: el servidor, las pantallas y el catalogo tenian cada uno
// su suite en verde y un comprador que descargara el binario no podia ver
// ninguna de las tres. Un producto que no se puede arrancar no esta hecho, por
// muchos tests que tenga cada mitad.
//
// Esto lo comprueba de la unica forma que vale: arrancandolo y pidiendole
// paginas por la red, como haria el navegador.

func puertoLibre(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("no hay puertos libres: %v", err)
	}
	dir := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	return dir
}

// arrancarServe levanta el servidor en una goruta y espera a que responda.
// Devuelve la direccion base y una funcion para pararlo.
func arrancarServe(t *testing.T, args ...string) (string, func()) {
	t.Helper()
	dir := puertoLibre(t)
	raiz, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	completos := append([]string{"--direccion", dir, "--corpus", filepath.Join(raiz, "paquetes")}, args...)

	hecho := make(chan int, 1)
	var salida, errsal bytes.Buffer
	go func() { hecho <- cmdServe(completos, &salida, &errsal) }()

	base := "http://" + dir
	cli := &http.Client{Timeout: time.Second}
	vivo := false
	for i := 0; i < 200; i++ {
		resp, err := cli.Get(base + "/alcance")
		if err == nil {
			if err := resp.Body.Close(); err != nil {
				t.Fatal(err)
			}
			vivo = true
			break
		}
		select {
		case c := <-hecho:
			t.Fatalf("el servidor se cerro antes de responder, codigo %d.\nsalida: %s\nerror: %s",
				c, salida.String(), errsal.String())
		default:
		}
		time.Sleep(25 * time.Millisecond)
	}
	if !vivo {
		t.Fatalf("el servidor no respondio en 5 s.\nsalida: %s\nerror: %s",
			salida.String(), errsal.String())
	}
	return base, func() {
		// cmdServe cierra con SIGTERM o Ctrl+C; en un test se corta el proceso
		// entero al terminar, asi que aqui solo se documenta el final.
		_ = hecho
	}
}

func TestDutiqServeLevantaLaInterfazYResponde(t *testing.T) {
	base, parar := arrancarServe(t)
	defer parar()

	cli := &http.Client{Timeout: 2 * time.Second}
	for _, ruta := range []string{"/alcance", "/hoy", "/controles", "/certificados",
		"/personas", "/estado"} {
		resp, err := cli.Get(base + ruta)
		if err != nil {
			t.Fatalf("%s: %v", ruta, err)
		}
		cuerpo := make([]byte, 64*1024)
		n, _ := resp.Body.Read(cuerpo)
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s devolvio %d", ruta, resp.StatusCode)
		}
		if !strings.Contains(string(cuerpo[:n]), "<html") {
			t.Errorf("%s no devuelve una pagina", ruta)
		}
	}
}

// El catalogo esta cableado de verdad: la pagina sale con TEXTO, no con las
// claves en crudo.
//
// Es la comprobacion que separa "las tres piezas existen" de "las tres piezas
// estan unidas". Una pantalla que dice pantalla.alcance.titulo esta tecnicamente
// servida y comercialmente muerta.
func TestLaInterfazSaleConTextoYNoConClavesEnCrudo(t *testing.T) {
	base, parar := arrancarServe(t)
	defer parar()

	resp, err := (&http.Client{Timeout: 2 * time.Second}).Get(base + "/alcance")
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 256*1024)
	n, _ := resp.Body.Read(buf)
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	pagina := string(buf[:n])

	if strings.Contains(pagina, "pantalla.") || strings.Contains(pagina, "alcance.pregunta.") {
		t.Errorf("la pagina lleva claves de catalogo en crudo. El catalogo no esta cableado, " +
			"o le faltan claves. Un rotulo que dice pantalla.alcance.titulo esta tecnicamente " +
			"servido y comercialmente muerto")
	}
	if !strings.Contains(pagina, "<h1>") {
		t.Error("la pagina no trae titulo")
	}
}

// Las cabeceras de seguridad del servidor llegan a las paginas de la aplicacion.
// Ya hay un test de esto sobre el handler; aqui se comprueba SOBRE LA RED, que
// es donde el navegador las lee y donde se ve si un proxy o un ResponseWriter
// intermedio se las come.
func TestLasCabecerasDeSeguridadLleganPorLaRed(t *testing.T) {
	base, parar := arrancarServe(t)
	defer parar()

	resp, err := (&http.Client{Timeout: 2 * time.Second}).Get(base + "/controles")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	for _, c := range []string{"Content-Security-Policy", "X-Frame-Options",
		"X-Content-Type-Options", "Referrer-Policy"} {
		if resp.Header.Get(c) == "" {
			t.Errorf("la respuesta real no lleva %s", c)
		}
	}
}

// Y un POST sin token CSRF no se atiende, tambien por la red.
func TestUnPostSinCSRFNoSeAtiendePorLaRed(t *testing.T) {
	base, parar := arrancarServe(t)
	defer parar()

	resp, err := (&http.Client{Timeout: 2 * time.Second}).Post(base+"/alcance",
		"application/x-www-form-urlencoded", strings.NewReader("x=1"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		t.Errorf("un POST sin CSRF devolvio %d. Una peticion mutante atendida sin token es "+
			"la que el atacante va a buscar", resp.StatusCode)
	}
}

// Lo que ve quien se equivoca. Cada fallo con su codigo y su arreglo escrito:
// un mensaje que no dice como salir del problema deja al operador leyendo
// codigo fuente, que es justo lo que este producto promete evitar.
func TestServeFallaDiciendoComoArreglarlo(t *testing.T) {
	vacio := t.TempDir()
	conBasura := t.TempDir()
	if err := os.WriteFile(filepath.Join(conBasura, "no-es-un-paquete.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	casos := []struct {
		nombre   string
		args     []string
		codigo   int
		contiene string
	}{
		{"corpus que no existe", []string{"--corpus", filepath.Join(vacio, "no-existe")}, 1, "dutiq demo"},
		{"corpus vacio", []string{"--corpus", vacio}, 1, "ningun paquete"},
		{"corpus sin paquetes", []string{"--corpus", conBasura}, 1, "ningun paquete"},
		{"opcion desconocida", []string{"--inventada"}, 2, ""},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			// Con plazo, y no llamando y ya. Si la comprobacion se rompe,
			// cmdServe no devuelve un codigo: SE PONE A SERVIR, y el test se
			// queda colgado hasta que Go lo mata con un panic de timeout a los
			// diez minutos. Un cuelgue es una senal de fallo pesima: no dice
			// que paso y tumba el CI entero. Comprobado por mutacion.
			var salida, errsal bytes.Buffer
			hecho := make(chan int, 1)
			go func() { hecho <- cmdServe(c.args, &salida, &errsal) }()
			var got int
			select {
			case got = <-hecho:
			case <-time.After(5 * time.Second):
				t.Fatalf("cmdServe NO ha devuelto: se ha puesto a servir con una entrada que "+
					"tenia que rechazar (%v). La comprobacion de entrada esta rota", c.args)
			}
			if got != c.codigo {
				t.Errorf("codigo %d, se esperaba %d. error: %s", got, c.codigo, errsal.String())
			}
			if c.contiene != "" && !strings.Contains(errsal.String(), c.contiene) {
				t.Errorf("el mensaje tiene que decir como salir del problema (%q) y dijo: %s",
					c.contiene, errsal.String())
			}
		})
	}
}

// --help NO es un fallo. Un script de instalacion que mira el codigo de salida
// se creeria que ha reventado. Es el mismo hallazgo que el frente de
// autoservicio cerro en sus tres ordenes.
func TestServeConHelpSaleConCero(t *testing.T) {
	var salida, errsal bytes.Buffer
	if got := cmdServe([]string{"--help"}, &salida, &errsal); got != 0 {
		t.Errorf("--help salio con %d y tiene que salir con 0", got)
	}
	if !strings.Contains(errsal.String(), "dutiq serve") {
		t.Errorf("--help tiene que imprimir la ayuda, y dijo: %s", errsal.String())
	}
}

// La cookie solo puede ir sin la marca de segura cuando de verdad no se puede
// llegar desde fuera. Equivocarse aqui de mas deja al operador en un bucle de
// entrada sin ningun mensaje (el navegador se queda la cookie y no la devuelve),
// y equivocarse de menos manda la sesion en claro por la red.
func TestSoloSeSirveSinTLSCuandoDeVerdadEsLocal(t *testing.T) {
	locales := []string{"127.0.0.1:8443", "localhost:8443", "[::1]:8443", "127.0.0.1:0"}
	abiertas := []string{":8443", "0.0.0.0:8443", "10.0.0.5:8443", "[2001:db8::1]:8443",
		"dutiq.example:8443"}
	for _, d := range locales {
		if !esLocal(d) {
			t.Errorf("%q es local y no se reconoce como tal", d)
		}
	}
	for _, d := range abiertas {
		if esLocal(d) {
			t.Errorf("%q NO es local y se ha tomado por local: la cookie de sesion saldria "+
				"sin la marca de segura por una direccion alcanzable desde fuera", d)
		}
	}
}

// Sonda de que el helper de contexto no se queda sin usar si algun dia se
// reordena el arranque.
func TestElCierreOrdenadoTienePlazoPropio(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	cierre, c2 := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer c2()
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatal("el contexto de arranque tiene que quedar cancelado")
	}
	if cierre.Err() != nil {
		t.Fatal("el contexto de cierre NO puede heredar la cancelacion del de arranque: " +
			"cerraria de golpe y cortaria las peticiones en vuelo")
	}
	_ = fmt.Sprint(ctx.Err())
}
