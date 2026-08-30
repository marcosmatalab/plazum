package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// `plazum escalado` NO MANDA NADA SIN QUE SE LO PIDAN.
//
// Es el valor cero restrictivo aplicado a una orden que escribe a personas: un
// comando que manda correos al ejecutarlo para ver que hace es un comando que
// alguien ejecuta una vez y le llega a media empresa.
func TestElEscaladoEnSecoNiMandaNiTocaElDiario(t *testing.T) {
	dir := t.TempDir()
	diario := filepath.Join(dir, "diario.jsonl")

	var salida, errores bytes.Buffer
	codigo := cmdEscalado([]string{
		"--corpus", "../../paquetes",
		"--alcance", "../../paquetes/demo-empresa/alcance.json",
		"--diario", diario,
		"--ahora", "2026-09-01T09:00:00Z",
		// Con SMTP configurado, para que haya canal y el plan diga "saldria":
		// asi el caso demuestra que NO manda aun teniendo por donde.
		"--smtp", "correo.ejemplo:25", "--de", "plazum@ejemplo.com",
	}, &salida, &errores)
	if codigo != 0 {
		t.Fatalf("salio con codigo %d: %s", codigo, errores.String())
	}
	s := salida.String()
	if !strings.Contains(s, "EN SECO") || !strings.Contains(s, "NO se ha mandado nada") {
		t.Fatalf("la salida no dice que va en seco:\n%s", s)
	}
	if !strings.Contains(s, "[saldria]") {
		t.Fatalf("no ensena ningun aviso que saldria, asi que este caso no demuestra que se "+
			"contenga: demuestra que no hay nada que mandar\n%s", s)
	}
	// EL DIARIO NO SE TOCA. Anotar una intencion que no se va a cumplir dejaria
	// el diario diciendo que hubo un aviso en vuelo, y la vuelta siguiente lo
	// "reenviaria" sin que nunca hubiera salido.
	if _, err := os.Stat(diario); !os.IsNotExist(err) {
		b, _ := os.ReadFile(diario)
		t.Fatalf("en seco se ha escrito el diario:\n%s", b)
	}
}

// Y la salida dice a QUIEN y por que CANAL, que es la pregunta del dia uno:
// si esto se pusiera a avisar ahora, a quien escribiria.
func TestElEscaladoEnSecoDiceAQuienYPorDonde(t *testing.T) {
	var salida, errores bytes.Buffer
	if codigo := cmdEscalado([]string{
		"--corpus", "../../paquetes",
		"--alcance", "../../paquetes/demo-empresa/alcance.json",
		"--diario", filepath.Join(t.TempDir(), "d.jsonl"),
		"--ahora", "2026-09-01T09:00:00Z",
		"--smtp", "correo.ejemplo:25", "--de", "plazum@ejemplo.com",
	}, &salida, &errores); codigo != 0 {
		t.Fatalf("salio con codigo %d: %s", codigo, errores.String())
	}
	s := salida.String()
	for _, quiero := range []string{"demo.responsable_de_seguridad", "por email", "escalon 1"} {
		if !strings.Contains(s, quiero) {
			t.Errorf("la salida no dice %q:\n%s", quiero, s)
		}
	}
	// LA LEY DE CONSERVACION, IMPRESA: la cuenta de los cubos aparece y no hay
	// aviso de descuadre.
	if !strings.Contains(s, "avisos planificados:") {
		t.Errorf("no se imprime la cuenta:\n%s", s)
	}
	if strings.Contains(s, "AVISO: los cubos suman") {
		t.Errorf("la cuenta no cuadra:\n%s", s)
	}
}

// --mandar sin ningun canal no manda: lo dice y sale con error. Un comando que
// dijera "mandado" sin canal seria la peor manera de descubrir que no hay.
func TestMandarSinCanalNoSeQuedaCallado(t *testing.T) {
	var salida, errores bytes.Buffer
	codigo := cmdEscalado([]string{
		"--corpus", "../../paquetes",
		"--alcance", "../../paquetes/demo-empresa/alcance.json",
		"--diario", filepath.Join(t.TempDir(), "d.jsonl"),
		"--ahora", "2026-09-01T09:00:00Z",
		"--mandar",
	}, &salida, &errores)
	if codigo == 0 {
		t.Fatalf("--mandar sin canal sale con exito:\n%s", salida.String())
	}
	if !strings.Contains(errores.String(), "sin ningun canal configurado") {
		t.Fatalf("el error no dice que falta el canal: %s", errores.String())
	}
}

// Y con canal pero sin lista de permitidos tampoco: la lista blanca vacia no
// significa "todos" (invariante 8), y el mensaje lo dice.
func TestMandarSinListaDePermitidosNoEscribeANadie(t *testing.T) {
	var salida, errores bytes.Buffer
	codigo := cmdEscalado([]string{
		"--corpus", "../../paquetes",
		"--alcance", "../../paquetes/demo-empresa/alcance.json",
		"--diario", filepath.Join(t.TempDir(), "d.jsonl"),
		"--ahora", "2026-09-01T09:00:00Z",
		"--mandar", "--smtp", "correo.ejemplo:25", "--de", "plazum@ejemplo.com",
	}, &salida, &errores)
	if codigo == 0 {
		t.Fatalf("se manda sin lista de permitidos:\n%s", salida.String())
	}
	if !strings.Contains(errores.String(), "permitido") {
		t.Fatalf("el error no habla de la lista de permitidos: %s", errores.String())
	}
}

// EL FICHERO DE ALCANCE DEL DEMO CARGA. Parece una perogrullada y no lo era:
// `plazum demo` lo escribe, la ayuda de `alcance.go` lo nombra por su ruta, y
// la decodificacion estricta lo RECHAZABA por un campo que nadie habia
// declarado. O sea que el producto producia un fichero que despues no cargaba,
// justo en los primeros cinco minutos de quien lo prueba.
func TestElAlcanceQueEscribeElDemoCargaDeVerdad(t *testing.T) {
	al, err := cargarAlcance("../../paquetes/demo-empresa/alcance.json")
	if err != nil {
		t.Fatalf("el alcance del demo no carga: %v", err)
	}
	if al.NotasDeLasFechas == "" {
		t.Fatal("la nota de las fechas no llega al tipo")
	}
	// Y SE IMPRIME: un campo que se lee y no se pinta es la otra mitad de la
	// misma familia.
	var salida, errores bytes.Buffer
	if codigo := cmdCalendario([]string{
		"--corpus", "../../paquetes",
		"--alcance", "../../paquetes/demo-empresa/alcance.json",
		"--ahora", "2026-09-01T09:00:00Z",
	}, &salida, &errores); codigo != 0 {
		t.Fatalf("el calendario del demo sale con codigo %d: %s", codigo, errores.String())
	}
	if !strings.Contains(salida.String(), al.NotasDeLasFechas) {
		t.Fatalf("la nota del alcance no se imprime en ningun sitio:\n%s", salida.String())
	}
}
