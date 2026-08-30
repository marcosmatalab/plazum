package canal

import (
	"encoding/json"
	"errors"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/internal/modulo"
	"github.com/marcosmatalab/plazum/puertos"
)

// LA FRONTERA DE ESTE ADAPTADOR, y es de imports antes que de conducta.
//
// "No lo mandamos" y "no podemos mandarlo" son dos promesas distintas, y solo
// la segunda la puede comprobar un comprador sin leerse el codigo.

func TestEsteAdaptadorNoPuedeLeerNadaDeNucleo(t *testing.T) {
	mod, err := modulo.Ruta()
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	ficheros, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	mirados := 0
	for _, ruta := range ficheros {
		if strings.HasSuffix(ruta, "_test.go") {
			continue
		}
		mirados++
		a, err := parser.ParseFile(fset, ruta, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("no puedo parsear %s: %v", ruta, err)
		}
		for _, imp := range a.Imports {
			v := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(v, mod+"/nucleo/") {
				t.Errorf(`%s importa %s.

  Un canal que PUEDE leer el estado de cumplimiento acaba mandandolo, igual
  que un adaptador de telemetria que puede leerlo acaba mandandolo. Este
  paquete recibe cuatro cadenas y las lleva: si necesita saber de que
  obligacion se trata, el que decide es quien tiene que decirselo, ya
  redactado.`, ruta, v)
			}
		}
	}
	// Suelo: si el recorrido deja de encontrar ficheros, "no importa nada" se
	// leeria igual que "todo en orden".
	if mirados == 0 {
		t.Fatal("no se ha mirado ni un fichero del adaptador")
	}
}

// ---------------------------------------------------------------------------
// La lista blanca, y su valor cero
// ---------------------------------------------------------------------------

// transporteDePrueba es un canal que no habla con nadie: sirve para mirar la
// lista blanca sin abrir nada.
type transporteDePrueba struct {
	nombre, anfitrion string
	err               error
	entregas          int
}

func (t *transporteDePrueba) Nombre() string    { return t.nombre }
func (t *transporteDePrueba) Anfitrion() string { return t.anfitrion }
func (t *transporteDePrueba) Entregar(string, string, string) error {
	t.entregas++
	return t.err
}

// LAS DOS FORMAS DE LA NADA (invariante 8). Ni la lista a nil ni la lista
// vacia-presente pueden significar "todos": el valor cero de una frontera de
// confianza tiene que ser el restrictivo, y esta es una frontera que saca datos
// de la maquina del operador.
func TestElValorCeroDeLaListaBlancaNoEnviaANingunSitio(t *testing.T) {
	for nombre, cfg := range map[string]Config{
		"nil":            {Permitidos: nil},
		"vacia presente": {Permitidos: []string{}},
	} {
		t.Run(nombre, func(t *testing.T) {
			tr := &transporteDePrueba{nombre: "teams", anfitrion: "hooks.ejemplo"}
			m, err := Nuevo(cfg, tr)
			if !errors.Is(err, ErrSinPermitidos) {
				t.Fatalf("se construye un mensajero sin lista blanca: %v (%v)", m, err)
			}
			if tr.entregas != 0 {
				t.Fatal("ha entregado algo antes de que nadie lo autorizara")
			}
		})
	}
}

// El anfitrion casa EXACTO. Los comodines y los sufijos son la forma clasica de
// que una lista blanca deje de serlo sin que nadie lo note.
func TestUnAnfitrionQueSoloSEPARECEAlPermitidoNoRecibeNada(t *testing.T) {
	permitido := "hooks.ejemplo.com"
	for _, anfitrion := range []string{
		"malo.hooks.ejemplo.com", // subdominio
		"hooks.ejemplo.com.malo", // sufijo
		"hooks-ejemplo.com",      // guion por punto
		"hooks.ejemplo.co",       // recortado
		"",                       // sin decir a donde
	} {
		tr := &transporteDePrueba{nombre: "teams", anfitrion: anfitrion}
		if _, err := Nuevo(Config{Permitidos: []string{permitido}}, tr); err == nil {
			t.Errorf("el anfitrion %q pasa la lista blanca de %q", anfitrion, permitido)
		}
	}
	// CONTROL POSITIVO: el permitido de verdad SI pasa, y sin distinguir
	// mayusculas, que es como se escriben los nombres de anfitrion.
	for _, bueno := range []string{permitido, strings.ToUpper(permitido)} {
		tr := &transporteDePrueba{nombre: "teams", anfitrion: bueno}
		if _, err := Nuevo(Config{Permitidos: []string{permitido}}, tr); err != nil {
			t.Errorf("el anfitrion permitido %q se rechaza: %v", bueno, err)
		}
	}
}

// ---------------------------------------------------------------------------
// Los secretos no salen ni en los errores
// ---------------------------------------------------------------------------

// Un webhook de Teams ES una credencial: quien lo tenga puede escribir en ese
// canal. Y `http.Client` mete la URL entera en sus errores, asi que el camino
// natural (devolver el error tal cual) filtra la credencial al log del
// operador, al informe de `--issue` y a cualquier sitio donde ese error acabe.
func TestNiElWebhookNiLaContrasenaSalenEnNingunError(t *testing.T) {
	const centinela = "CENTINELA-CREDENCIAL-QUE-NO-DEBE-SALIR"
	permitidos := Config{Permitidos: []string{"127.0.0.1"}}

	t.Run("teams: la URL no llega al error de red", func(t *testing.T) {
		// Un servidor que se cierra al momento: el error de red es real.
		srv := httptest.NewTLSServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {}))
		url := srv.URL + "/" + centinela
		srv.Close() // ahora la peticion falla de verdad

		tm, err := NuevoTeams(url, srv.Client())
		if err != nil {
			t.Fatal(err)
		}
		m, err := Nuevo(permitidos, tm)
		if err != nil {
			t.Fatal(err)
		}
		err = m.Enviar("teams", "equipo", "asunto", "cuerpo")
		if err == nil {
			t.Fatal("la entrega a un servidor cerrado no fallo")
		}
		if strings.Contains(err.Error(), centinela) {
			t.Errorf("el error lleva la credencial dentro: %v", err)
		}
		// Y sigue diciendo lo que hace falta para diagnosticar.
		if !strings.Contains(err.Error(), "127.0.0.1") {
			t.Errorf("el error no dice con quien no pudo hablar: %v", err)
		}
	})

	t.Run("teams: una URL ilegible no se reproduce", func(t *testing.T) {
		_, err := NuevoTeams("://"+centinela, nil)
		if err == nil {
			t.Fatal("una URL sin esquema se acepta")
		}
		if strings.Contains(err.Error(), centinela) {
			t.Errorf("el error de construccion lleva la URL dentro: %v", err)
		}
	})

	t.Run("email: la credencial del envio no llega al error", func(t *testing.T) {
		// La funcion de envio falla con un error que lleva la credencial, que
		// es justo lo que hace net/smtp cuando la autenticacion se rechaza.
		envio := func(string, string, []string, []byte) error {
			return errors.New("535 auth failed for " + centinela)
		}
		e, err := NuevoEmail("127.0.0.1:25", "plazum@ejemplo.com", envio)
		if err != nil {
			t.Fatal(err)
		}
		m, err := Nuevo(permitidos, e)
		if err != nil {
			t.Fatal(err)
		}
		err = m.Enviar("email", "ana@ejemplo.com", "asunto", "cuerpo")
		if err == nil {
			t.Fatal("un envio que falla no da error")
		}
		if strings.Contains(err.Error(), centinela) {
			t.Errorf("el error lleva la credencial dentro: %v", err)
		}
	})

	// CONTROL NEGATIVO: el mismo centinela por un camino AUTORIZADO si sale.
	// Sin esto, un adaptador que devolviera siempre el mismo error generico
	// pasaria los tres de arriba sin filtrar nada porque no dice nada.
	t.Run("CONTROL: un dato no secreto SI aparece en el error", func(t *testing.T) {
		tr := &transporteDePrueba{nombre: "teams", anfitrion: "127.0.0.1",
			err: errors.New("el servidor contesto 503")}
		m, err := Nuevo(permitidos, tr)
		if err != nil {
			t.Fatal(err)
		}
		err = m.Enviar("teams", "equipo", "asunto", "cuerpo")
		if err == nil || !strings.Contains(err.Error(), "503") {
			t.Fatalf("el motivo del fallo tampoco llega, asi que los tests de arriba no "+
				"demuestran que se filtre lo secreto: demuestran que no se dice nada: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// Lo que viaja: lista blanca de campos
// ---------------------------------------------------------------------------

// El cuerpo que sale a Teams lleva exactamente tres campos. Se comprueba la
// forma del tipo Y los bytes que viajan: lo primero caza un campo anadido, lo
// segundo caza uno que hoy no se serializa y manana si.
func TestElMensajeDeTeamsLlevaExactamenteTresCampos(t *testing.T) {
	quiero := []string{"summary", "text", "title"}
	b, err := json.Marshal(mensajeTeams{Titulo: "a", Texto: "b", Para: "c"})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	var hay []string
	for k := range m {
		hay = append(hay, k)
	}
	sort.Strings(hay)
	if !reflect.DeepEqual(hay, quiero) {
		t.Fatalf("el mensaje de Teams manda %v y lo declarado es %v. Cada campo nuevo es una "+
			"via por la que sale dato del cliente a un servicio de terceros", hay, quiero)
	}
}

// ---------------------------------------------------------------------------
// La inyeccion clasica de SMTP
// ---------------------------------------------------------------------------

// Un asunto con un salto de linea dentro inyecta cabeceras SMTP: con un
// "\r\nBcc: otro@sitio" el aviso se copia a quien lo escriba. Y el asunto viene
// del titulo de una obligacion, que es texto de un paquete de corpus.
func TestUnAsuntoConSaltosNoInyectaCabecerasSMTP(t *testing.T) {
	var mandado []byte
	envio := func(_ string, _ string, _ []string, msg []byte) error {
		mandado = msg
		return nil
	}
	e, err := NuevoEmail("127.0.0.1:25", "plazum@ejemplo.com", envio)
	if err != nil {
		t.Fatal(err)
	}
	m, err := Nuevo(Config{Permitidos: []string{"127.0.0.1"}}, e)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Enviar("email", "ana@ejemplo.com",
		"vence manana\r\nBcc: espia@ejemplo.com", "cuerpo"); err != nil {
		t.Fatal(err)
	}
	cabeceras, _, _ := strings.Cut(string(mandado), "\r\n\r\n")
	// La inyeccion es una LINEA NUEVA que empieza por una cabecera. Que las
	// palabras "Bcc:" aparezcan DENTRO del valor del asunto es inofensivo, y
	// comprobarlo por subcadena daria rojo con el arreglo puesto: un test que no
	// distingue el ataque de su remedio no sirve para ninguno de los dos.
	quiero := []string{"From:", "To:", "Subject:", "MIME-Version:", "Content-Type:"}
	var vistas []string
	for _, l := range strings.Split(cabeceras, "\r\n") {
		nombre, _, hay := strings.Cut(l, ":")
		if !hay {
			t.Fatalf("una linea de cabecera sin nombre: %q", l)
		}
		vistas = append(vistas, nombre+":")
	}
	if !reflect.DeepEqual(vistas, quiero) {
		t.Fatalf("las cabeceras que salen son %v y las declaradas son %v: se ha inyectado "+
			"una desde el asunto o el destinatario\n%s", vistas, quiero, cabeceras)
	}
	// CONTROL POSITIVO: el asunto llega entero, solo que aplanado. Sin esto, un
	// Entregar que tirara el asunto pasaria la comprobacion de arriba.
	if !strings.Contains(cabeceras, "vence manana") {
		t.Fatalf("el asunto no llega:\n%s", cabeceras)
	}
}

// ---------------------------------------------------------------------------
// UltimoExito y el envio sin destinatario
// ---------------------------------------------------------------------------

// SOLO AVANZA CON UN EXITO DE VERDAD. Si avanzara al intentarlo, un canal roto
// se leeria como un canal vivo, que es el fallo que este dato existe para
// detectar.
func TestUltimoExitoSoloAvanzaConUnaEntregaQueSalioBien(t *testing.T) {
	reloj := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	tr := &transporteDePrueba{nombre: "teams", anfitrion: "127.0.0.1",
		err: errors.New("503")}
	m, err := Nuevo(Config{Permitidos: []string{"127.0.0.1"},
		Reloj: func() time.Time { return reloj }}, tr)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Enviar("teams", "equipo", "a", "b"); err == nil {
		t.Fatal("la entrega fallida no dio error")
	}
	if v, _ := m.UltimoExito("teams"); v != "" {
		t.Fatalf("un canal que solo ha fallado dice que entrego el %s", v)
	}
	tr.err = nil
	if err := m.Enviar("teams", "equipo", "a", "b"); err != nil {
		t.Fatal(err)
	}
	v, err := m.UltimoExito("teams")
	if err != nil || v != reloj.Format(time.RFC3339) {
		t.Fatalf("tras una entrega buena, el ultimo exito es %q (%v)", v, err)
	}
	// Y un canal que no existe se dice, en vez de contestar "nunca".
	if _, err := m.UltimoExito("humo"); !errors.Is(err, ErrCanalDesconocido) {
		t.Fatalf("un canal inexistente contesta %v en vez de decir que no existe", err)
	}
}

func TestUnEnvioSinDestinatarioNoSaleComoEnviado(t *testing.T) {
	tr := &transporteDePrueba{nombre: "teams", anfitrion: "127.0.0.1"}
	m, err := Nuevo(Config{Permitidos: []string{"127.0.0.1"}}, tr)
	if err != nil {
		t.Fatal(err)
	}
	for _, destinatario := range []string{"", "   "} {
		if err := m.Enviar("teams", destinatario, "a", "b"); !errors.Is(err, ErrSinDestinatario) {
			t.Errorf("un envio a %q devuelve %v", destinatario, err)
		}
	}
	if tr.entregas != 0 {
		t.Fatalf("ha entregado %d veces sin destinatario", tr.entregas)
	}
	if v, _ := m.UltimoExito("teams"); v != "" {
		t.Fatal("un envio sin destinatario ha contado como exito")
	}
}

// El webhook por http (sin TLS) manda la credencial en claro por la red.
func TestUnWebhookSinTLSNoSeAcepta(t *testing.T) {
	if _, err := NuevoTeams("http://hooks.ejemplo.com/abc", nil); err == nil {
		t.Fatal("se acepta un webhook por http")
	}
	if _, err := NuevoTeams("https://hooks.ejemplo.com/abc", nil); err != nil {
		t.Fatalf("se rechaza un webhook por https: %v", err)
	}
}

// El adaptador cumple el puerto DE VERDAD, contra el interfaz real y no contra
// una copia suya escrita al lado: copiar la forma del puerto aqui cumpliria la
// letra y dejaria de avisar el dia que el puerto cambie.
//
// Y lo declara el TEST, no el codigo: asi `puertos` — que si conoce el corpus —
// no entra en los imports del adaptador, y la frontera de arriba sigue siendo
// absoluta en lo que se compila y se distribuye.
var _ puertos.Notificacion = (*Mensajero)(nil)
