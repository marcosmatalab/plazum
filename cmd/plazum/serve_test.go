package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// La puerta de que el producto SE PUEDE ARRANCAR.
//
// Hasta que existio `plazum serve`, la superficie web estaba entera y no habia
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

// Las credenciales del administrador que crean los tests. Van aqui, con nombre,
// porque son la unica forma de entrar y varios ficheros las necesitan. La
// contrasena llega al minimo que exige el almacen a proposito: si un dia sube,
// esto tiene que ponerse rojo y no ir subiendo detras en silencio.
const (
	UsuarioDePrueba = "ciso"
	SecretoDePrueba = "contrasena-de-prueba-1"
	// OrganizacionDePrueba es de quien es la instalacion de los tests.
	// /primer-admin la pregunta desde que el acta sale de la identidad de la
	// instalacion en vez de una bandera de terminal.
	OrganizacionDePrueba = "Ejemplo SL"
)

// bufferSeguro recoge lo que escribe el servidor mientras sirve.
//
// EL MUTEX NO ES ADORNO: cmdServe escribe desde su goruta y el test lee para
// sacar el token de instalacion. Un bytes.Buffer compartido entre las dos es una
// carrera de datos, y la puerta `suite completa con detector de carreras` de
// ci.yml la encuentra.
type bufferSeguro struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *bufferSeguro) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *bufferSeguro) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// servidorServe es `plazum serve` levantado de verdad, con lo que hace falta
// para hablar con el como habla un navegador.
type servidorServe struct {
	base string
	// cli lleva el tarro de galletas con la sesion del administrador.
	cli *http.Client
	// crudo no lleva cookies: es el visitante que no ha entrado.
	//
	// NINGUNO DE LOS DOS SIGUE REDIRECCIONES, y esa es la leccion mas cara de
	// este fichero. Con un cliente que las seguia, `TestPlazumServeLevantaLa
	// InterfazYResponde` se puso VERDE sobre un producto en el que las seis
	// pantallas contestaban 303 a la instalacion: el cliente iba detras del 303,
	// encontraba el 200 de /primer-admin, veia su <html> y daba por servida una
	// pantalla que nadie estaba sirviendo. Un test que no ve el codigo que de
	// verdad contesta el servidor no esta mirando el servidor.
	crudo  *http.Client
	salida func() string
	// dirEstado es el temporal donde este servidor guarda usuarios.json y
	// respuestas.json. Se expone para poder mirar el disco: que una respuesta
	// «se ha guardado» solo es cierto si esta en un fichero.
	dirEstado string
	instalado bool
}

// arrancarServe levanta el servidor en una goruta y espera a que responda.
//
// EL ALMACEN DE USUARIOS VA A UN DIRECTORIO TEMPORAL SIEMPRE, y se pasa
// explicitamente aunque el llamante ya de --datos: sin eso, el fichero de
// cuentas se escribiria en el directorio del paquete, o sea dentro del
// repositorio, y un test dejaria credenciales derivadas en el arbol.
func arrancarServe(t *testing.T, args ...string) *servidorServe {
	t.Helper()
	dir := puertoLibre(t)
	raiz, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	estado := t.TempDir()
	completos := append([]string{
		"--direccion", dir,
		"--corpus", filepath.Join(raiz, "paquetes"),
		// Y EL DIRECTORIO DE DATOS ENTERO, que es de donde cuelga todo lo demas
		// que esta orden escribe: la identidad de la instalacion y la campana de
		// revision de accesos con su ledger y su censo.
		//
		// FALTABA, y lo encontro un test que se colgo: al preguntar el nombre de
		// la organizacion en /primer-admin, un test escribio cmd/plazum/instalacion.json
		// DENTRO DEL REPOSITORIO, y el siguiente test que arrancaba serve sin
		// banderas del acta se lo encontro puesto: componia el acta en vez de
		// fallar, el servidor arrancaba de verdad y el test se quedo esperando
		// hasta el timeout. Las dos lineas de abajo tapaban los dos ficheros que
		// ya existian cuando se escribieron; se quedan por si el defecto de
		// --datos cambia, pero la que cubre lo que venga es esta.
		"--datos", estado,
		"--usuarios", filepath.Join(estado, "usuarios.json"),
		// EL ALMACEN DE RESPUESTAS TAMBIEN A UN TEMPORAL, y por lo mismo que el
		// de cuentas: sin esto se escribiria en el directorio del paquete, o sea
		// DENTRO del repositorio, y un test dejaria en el arbol lo que una
		// cuenta contesto sobre su organizacion.
		"--respuestas", filepath.Join(estado, "respuestas.json"),
	}, args...)

	hecho := make(chan int, 1)
	var salida bufferSeguro
	var errsal bufferSeguro
	go func() { hecho <- cmdServe(completos, &salida, &errsal) }()

	base := "http://" + dir
	tarro, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	sinSeguir := func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	s := &servidorServe{
		base:      base,
		salida:    salida.String,
		dirEstado: estado,
		cli: &http.Client{
			Timeout: 20 * time.Second, Jar: tarro, CheckRedirect: sinSeguir,
		},
		crudo: &http.Client{
			Timeout: 20 * time.Second, CheckRedirect: sinSeguir,
		},
	}

	vivo := false
	arranque := &http.Client{Timeout: time.Second}
	for i := 0; i < 200; i++ {
		resp, err := arranque.Get(base + "/salud")
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
	// cmdServe cierra con SIGTERM o Ctrl+C; en un test se corta el proceso
	// entero al terminar, asi que no hay nada que parar aqui.
	return s
}

// arrancarServeInstalado es lo que quiere casi todo test: el producto arrancado
// Y con administrador, que es como lo tiene el operador dos minutos despues de
// descargarlo.
func arrancarServeInstalado(t *testing.T, args ...string) *servidorServe {
	t.Helper()
	s := arrancarServe(t, args...)
	s.instalar(t)
	return s
}

func TestPlazumServeLevantaLaInterfazYResponde(t *testing.T) {
	s := arrancarServeInstalado(t)

	for _, ruta := range []string{"/alcance", "/hoy", "/controles", "/certificados",
		"/personas", "/estado"} {
		// CON EL CLIENTE CRUDO, que no sigue redirecciones. Con el que las
		// seguia, este test se puso VERDE sobre un producto en el que las seis
		// pantallas contestaban 303 a la instalacion: el cliente iba detras del
		// 303, encontraba el 200 de /primer-admin, veia su <html> y daba por
		// servida una pantalla que nadie estaba sirviendo.
		codigo, _, cuerpo := s.pedirCrudo(t, ruta)
		if codigo != http.StatusOK {
			t.Errorf("%s devolvio %d", ruta, codigo)
			continue
		}
		if !strings.Contains(cuerpo, "<html") {
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
	s := arrancarServeInstalado(t)

	resp, err := s.crudo.Get(s.base + "/alcance")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/alcance contesta %d con la aplicacion instalada: lo que se leeria abajo "+
			"seria otra pagina, y este test daria verde sobre ella", resp.StatusCode)
	}
	// SE LEE LA PAGINA ENTERA, no el primer trozo que quepa en un Read.
	//
	// Antes esto era un unico `resp.Body.Read(buf)`, y un Read devuelve lo que
	// hay disponible, no lo que cabe: la comprobacion miraba los primeros
	// kilobytes y daba por revisado el resto. Se vio el dia que la barra lateral
	// crecio y empujo el <h1> mas abajo: el test dijo "la pagina no trae
	// titulo" sobre una pagina que lo traia. Y la mitad cara es la otra, la de
	// las claves en crudo, que llevaba tiempo mirando solo la cabecera.
	//
	// El tope se conserva con LimitReader: una respuesta enorme no puede
	// convertir este test en un consumo de memoria.
	pagina := leerHasta(t, resp.Body, 256*1024)
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}

	if strings.Contains(pagina, "pantalla.") || strings.Contains(pagina, "alcance.pregunta.") {
		t.Errorf("la pagina lleva claves de catalogo en crudo. El catalogo no esta cableado, " +
			"o le faltan claves. Un rotulo que dice pantalla.alcance.titulo esta tecnicamente " +
			"servido y comercialmente muerto")
	}
	if !strings.Contains(pagina, "<h1>") {
		t.Error("la pagina no trae titulo")
	}
}

// leerHasta lee un cuerpo entero con tope. Existe para que ningun test de este
// fichero vuelva a mirar solo el primer trozo de una pagina.
func leerHasta(t *testing.T, r io.Reader, tope int64) string {
	t.Helper()
	b, err := io.ReadAll(io.LimitReader(r, tope))
	if err != nil {
		t.Fatalf("leyendo la respuesta: %v", err)
	}
	if int64(len(b)) == tope {
		t.Fatalf("la respuesta llega al tope de %d bytes: el test estaria mirando media "+
			"pagina otra vez", tope)
	}
	return string(b)
}

// Las cabeceras de seguridad del servidor llegan a las paginas de la aplicacion.
// Ya hay un test de esto sobre el handler; aqui se comprueba SOBRE LA RED, que
// es donde el navegador las lee y donde se ve si un proxy o un ResponseWriter
// intermedio se las come.
func TestLasCabecerasDeSeguridadLleganPorLaRed(t *testing.T) {
	s := arrancarServeInstalado(t)

	resp, err := s.crudo.Get(s.base + "/controles")
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
	s := arrancarServeInstalado(t)

	resp, err := s.crudo.Post(s.base+"/alcance",
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
		{"corpus que no existe", []string{"--corpus", filepath.Join(vacio, "no-existe")}, 1, "plazum demo"},
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
	if !strings.Contains(errsal.String(), "plazum serve") {
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
		"plazum.example:8443"}
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
