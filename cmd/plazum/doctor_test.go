package main

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"plazum/adaptadores/diagnostico"
)

func doctor(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var salida, errores bytes.Buffer
	codigo := cmdDoctor(args, &salida, &errores)
	return salida.String(), errores.String(), codigo
}

func direccionLibre(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	d := l.Addr().String()
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	return d
}

// El codigo de salida es lo que hace que doctor sirva de comprobacion de
// arranque en un systemd o en un CI. Si siempre devolviera 0, seria un texto
// bonito que nadie puede automatizar.
func TestElCodigoDeSalidaDistingueLoRotoDeLoQueSoloAvisa(t *testing.T) {
	t.Run("solo avisos: 0", func(t *testing.T) {
		_, _, codigo := doctor(t, "--datos", t.TempDir(), "--direccion", direccionLibre(t))
		if codigo != 0 {
			t.Fatalf("una instalacion recien hecha solo tiene avisos y devolvio %d", codigo)
		}
	})
	t.Run("algo roto: 1", func(t *testing.T) {
		ocupado, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = ocupado.Close() }()
		salida, _, codigo := doctor(t, "--datos", t.TempDir(), "--direccion", ocupado.Addr().String())
		if codigo != 1 {
			t.Fatalf("con el puerto ocupado tenia que devolver 1 y devolvio %d", codigo)
		}
		if !strings.Contains(salida, "ROTO") {
			t.Error("el codigo dice que algo esta roto y la pantalla no lo senala")
		}
	})
}

// La regla del comando: nada que no este correcto sale sin decir como se
// arregla. Se comprueba sobre la SALIDA, no sobre la estructura, porque es lo
// que ve el operador.
func TestNingunaLineaConProblemaSaleSinSuArreglo(t *testing.T) {
	ocupado, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ocupado.Close() }()

	salida, _, _ := doctor(t,
		"--datos", t.TempDir(),
		"--direccion", ocupado.Addr().String(),
		"--ahora", "2020-01-01T00:00:00Z", // reloj atrasado
		"--corpus", filepath.Join(t.TempDir(), "no-existe"),
	)
	lineas := strings.Split(salida, "\n")
	problemas := 0
	for i, l := range lineas {
		if !strings.Contains(l, "AVISO") && !strings.Contains(l, "ROTO") {
			continue
		}
		problemas++
		if i+1 >= len(lineas) || !strings.Contains(lineas[i+1], "arreglo:") {
			t.Errorf("la linea %q no va seguida de su arreglo", strings.TrimSpace(l))
		}
	}
	if problemas < 3 {
		t.Fatalf("esta instalacion tenia tres cosas mal a proposito y solo se detectaron %d: "+
			"la comprobacion de arriba no esta mirando nada", problemas)
	}
}

// La salida para un issue lleva rutas absolutas, y una ruta absoluta lleva
// dentro el nombre de usuario, que es un dato personal. Publicarlo en un
// repositorio publico es una cesion que nadie ha consentido.
func TestElInformeParaUnIssueNoPublicaElNombreDelUsuario(t *testing.T) {
	casa, err := os.UserHomeDir()
	if err != nil || casa == "" {
		t.Skip("este sistema no declara directorio de usuario")
	}
	usuario := filepath.Base(casa)
	if len(usuario) <= 2 {
		t.Skip("el nombre de usuario es demasiado corto para redactarlo sin destrozar el texto")
	}

	got := redactar("no puedo escribir en " + filepath.Join(casa, "plazum", "datos"))
	if strings.Contains(got, usuario) {
		t.Errorf("la redaccion deja el nombre de usuario en el texto: %q", got)
	}
	// Y ademas la ruta del hogar se colapsa entera a ~.
	//
	// BARRIDO DE MUTACION: sin esta segunda comprobacion, borrar la sustitucion
	// de la ruta del hogar seguia dando verde, porque la sustitucion del nombre
	// de usuario tapaba el hueco por su cuenta. Dos comprobaciones que se
	// cubren la una a la otra son una comprobacion, y la que quedaba deja
	// escapar la estructura del sistema de ficheros del operador.
	if !strings.HasPrefix(got, "no puedo escribir en ~") {
		t.Errorf("la ruta del hogar no se ha colapsado a ~: %q", got)
	}
	if padre := filepath.Dir(casa); padre != "" && padre != "." && strings.Contains(got, padre) {
		t.Errorf("la redaccion deja el camino hasta el hogar (%q) en el texto: %q", padre, got)
	}
	if !strings.Contains(got, "plazum") {
		t.Errorf("la redaccion se ha llevado por delante la ruta entera y el mensaje ya no dice "+
			"donde estaba el problema: %q", got)
	}
}

func TestElInformeParaUnIssueSaleEnUnBloqueCopiable(t *testing.T) {
	salida, _, _ := doctor(t, "--datos", t.TempDir(), "--direccion", direccionLibre(t), "--issue")
	if strings.Count(salida, "```") != 2 {
		t.Errorf("la salida no viene en un bloque de codigo, asi que al pegarla en un issue se "+
			"desarma:\n%s", salida)
	}
	if !strings.Contains(salida, "sistema ") {
		t.Error("el informe no dice en que sistema corre, que es la primera pregunta de quien lo lee")
	}
	// Y sigue diciendo el arreglo: un informe para un issue que se come los
	// arreglos obliga a quien lo recibe a repetir el diagnostico.
	if !strings.Contains(salida, "arreglo:") {
		t.Error("el informe para el issue no lleva los arreglos")
	}
}

// Las raices que declara el operador se comprueban de verdad, y el fichero de
// contexto que se lee es EL MISMO que lee `plazum verify`: dos lecturas distintas
// del mismo formato son dos formatos.
func TestDoctorLeeLasRaicesDelMismoContextoQueVerify(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "contexto.json")
	if err := os.WriteFile(ruta, []byte(`{"raices_tsa": "no soy un PEM"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	salida, _, codigo := doctor(t, "--datos", t.TempDir(), "--direccion", direccionLibre(t),
		"--contexto", ruta)
	if codigo != 1 {
		t.Fatalf("con unas raices declaradas que no son certificados, doctor tenia que devolver 1 "+
			"y devolvio %d:\n%s", codigo, salida)
	}
	if !strings.Contains(salida, "raices_tsa") {
		t.Errorf("el mensaje no dice que campo del contexto esta mal: %s", salida)
	}
}

func TestUnContextoQueNoExisteSeDiceEnVezDeIgnorarse(t *testing.T) {
	_, errores, codigo := doctor(t, "--contexto", filepath.Join(t.TempDir(), "no-esta.json"))
	if codigo == 0 {
		t.Fatal("se pidio comprobar un contexto que no existe y doctor siguio como si nada")
	}
	if !strings.Contains(errores, "--contexto") {
		t.Errorf("el error no dice como seguir sin el: %s", errores)
	}
}

// La direccion que comprueba doctor tiene que ser la MISMA en la que va a
// escuchar el servidor. Comprobar un puerto parecido no comprueba nada.
func TestLaDireccionPorDefectoEsLaQueUsaElProducto(t *testing.T) {
	salida, _, _ := doctor(t, "--datos", t.TempDir())
	if !strings.Contains(salida, diagnostico.DireccionPorDefecto) {
		t.Errorf("doctor no dice que direccion ha comprobado, o comprueba otra distinta de %q:\n%s",
			diagnostico.DireccionPorDefecto, salida)
	}
}
