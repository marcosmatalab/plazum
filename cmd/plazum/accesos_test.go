package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/accesos"
	"github.com/marcosmatalab/plazum/nucleo/ledger"
)

// El fichero hostil de verdad: BOM de Excel, punto y coma, columnas que no se
// guardan (DNI, departamento), una fila sin identificador, una repetida y un
// permiso con dos valores en una celda.
const csvHostil = "\uFEFFusuario;nombre;permiso;dni;departamento\n" +
	"u1;Ana Martinez;admin;12345678Z;IT\n" +
	"u1;Ana Martinez;lector;12345678Z;IT\n" +
	"u2;Luis Gil;lector;87654321X;Ventas\n" +
	"u2;Luis Gil;lector;87654321X;Ventas\n" +
	";Sin Cuenta;lector;;\n" +
	"u3;Eva Roca;admin,lector;11111111H;Finanzas\n"

func escribirCSV(t *testing.T, contenido string) string {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), "usuarios.csv")
	if err := os.WriteFile(ruta, []byte(contenido), 0o600); err != nil {
		t.Fatal(err)
	}
	return ruta
}

// LO QUE HA ENTENDIDO VA ANTES QUE NINGUN NUMERO.
//
// Quien sube un CSV a las nueve de la manana no tiene forma de saber si lo que
// ve corresponde a su fichero. Si el primer dato de la pantalla es "4 accesos",
// se lo cree; si lo primero es "lo he leido en utf-8, separado por punto y coma
// y he tirado el DNI", puede decir que no.
func TestElComandoDiceQueHaEntendidoAntesDeDarUnNumero(t *testing.T) {
	var salida, errores bytes.Buffer
	codigo := cmdAccesos([]string{
		"--fichero", escribirCSV(t, csvHostil),
		"--sistema", "erp", "--quien", "u-042",
		"--ahora", "2026-09-01T09:00:00Z",
	}, &salida, &errores)
	if codigo != 0 {
		t.Fatalf("codigo %d: %s", codigo, errores.String())
	}
	s := salida.String()

	entendido := strings.Index(s, "LO QUE PLAZUM HA ENTENDIDO")
	numero := strings.Index(s, "accesos que se pueden revisar")
	if entendido < 0 || numero < 0 || entendido > numero {
		t.Fatalf("los numeros salen antes que lo que se ha entendido del fichero:\n%s", s)
	}
	for _, quiero := range []string{
		"traia BOM al principio",
		"punto y coma",
		"descartadas:  dni, departamento",
	} {
		if !strings.Contains(s, quiero) {
			t.Errorf("la salida no dice %q:\n%s", quiero, s)
		}
	}
	// MINIMIZACION COMPROBADA EN LA SALIDA, no solo prometida: el DNI se nombra
	// como columna descartada y su VALOR no aparece en ningun sitio.
	if strings.Contains(s, "12345678Z") || strings.Contains(s, "87654321X") {
		t.Error("un DNI ha llegado a la pantalla")
	}
}

// LOS CUBOS Y LA FRASE. La ley de conservacion impresa, y lo no revisado
// presentado como dato que falta.
func TestLaSalidaLlevaLosCubosYNoAcusaDeLoQueSoloFalta(t *testing.T) {
	var salida, errores bytes.Buffer
	if codigo := cmdAccesos([]string{
		"--fichero", escribirCSV(t, csvHostil),
		"--sistema", "erp", "--quien", "u-042",
		"--ahora", "2026-09-01T09:00:00Z",
	}, &salida, &errores); codigo != 0 {
		t.Fatalf("codigo %d: %s", codigo, errores.String())
	}
	s := salida.String()
	for _, quiero := range []string{
		"accesos que se pueden revisar: 4",
		"lineas ilegibles:              1",
		"filas repetidas:               1",
		"no consta que nadie los haya revisado",
		"no tienen revisor asignado",
	} {
		if !strings.Contains(s, quiero) {
			t.Errorf("la salida no dice %q:\n%s", quiero, s)
		}
	}
	if strings.Contains(s, "AVISO: los cubos NO cubren") {
		t.Errorf("los cubos no cuadran:\n%s", s)
	}
}

// UN FICHERO AL QUE LE FALTAN LINEAS NO SE SUBE, y el comando sale con error en
// vez de ensenar un recuento que no cuadra. Es la garantia del censo llegando
// hasta la superficie.
func TestUnFicheroConLineasSinCuboNoSeSube(t *testing.T) {
	// Un campo con un salto de linea dentro: el lector lo ve como un registro y
	// el contador ciego como dos, y nadie puede decir cual es.
	roto := "usuario;permiso\n\"ana\nmaria\";admin\nluis;lector\n"
	var salida, errores bytes.Buffer
	codigo := cmdAccesos([]string{
		"--fichero", escribirCSV(t, roto),
		"--sistema", "erp", "--quien", "u-042",
		"--ahora", "2026-09-01T09:00:00Z",
	}, &salida, &errores)
	if codigo == 0 {
		t.Fatalf("se ha subido un fichero al que le faltan lineas:\n%s", salida.String())
	}
	if !strings.Contains(errores.String(), "sin cubo") {
		t.Fatalf("el error no explica el descuadre: %s", errores.String())
	}
	if strings.Contains(salida.String(), "accesos que se pueden revisar") {
		t.Error("se ha impreso un recuento de un fichero que no ha cargado")
	}
}

// --sistema y --quien NO TIENEN VALOR POR DEFECTO. Una lista de cuentas sin de
// que sistema es y sin quien la subio no se puede atar a nada.
func TestSinSistemaNiQuienNoSeSube(t *testing.T) {
	for _, falta := range []string{"--sistema", "--quien"} {
		t.Run(falta, func(t *testing.T) {
			args := []string{"--fichero", escribirCSV(t, csvHostil),
				"--sistema", "erp", "--quien", "u-042"}
			for i, a := range args {
				if a == falta {
					args[i+1] = ""
				}
			}
			var salida, errores bytes.Buffer
			if codigo := cmdAccesos(args, &salida, &errores); codigo == 0 {
				t.Fatalf("se ha subido sin %s:\n%s", falta, salida.String())
			}
			if !strings.Contains(errores.String(), "no certifica nada") {
				t.Errorf("el error no dice por que importa: %s", errores.String())
			}
		})
	}
}

// LOS REVISORES: por fichero o uno para todo, pero no las dos cosas. Dos
// fuentes de la misma decision es como se acaba sin saber cual mando.
func TestLosDosModosDeAsignarRevisorNoSeMezclan(t *testing.T) {
	rev := filepath.Join(t.TempDir(), "revisores.csv")
	if err := os.WriteFile(rev, []byte("u1;jefa\nu2;responsable-ventas\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	base := []string{"--fichero", escribirCSV(t, csvHostil), "--sistema", "erp",
		"--quien", "u-042", "--ahora", "2026-09-01T09:00:00Z"}

	var salida, errores bytes.Buffer
	if codigo := cmdAccesos(append(append([]string{}, base...), "--revisores", rev, "--revisor", "jefa"),
		&salida, &errores); codigo == 0 {
		t.Fatal("se han admitido las dos formas a la vez")
	}
	if !strings.Contains(errores.String(), "dos cosas a la vez") {
		t.Errorf("el error no lo explica: %s", errores.String())
	}

	// Con el fichero: u3 se queda sin revisor y se dice.
	salida.Reset()
	errores.Reset()
	if codigo := cmdAccesos(append(append([]string{}, base...), "--revisores", rev),
		&salida, &errores); codigo != 0 {
		t.Fatalf("codigo %d: %s", codigo, errores.String())
	}
	if !strings.Contains(salida.String(), "1 accesos no tienen revisor asignado") &&
		!strings.Contains(salida.String(), "accesos no tienen revisor asignado") {
		t.Errorf("no se dice que u3 se queda sin revisor:\n%s", salida.String())
	}
	if !strings.Contains(salida.String(), "erp|u3|") {
		t.Errorf("no se dice CUAL se queda sin revisor:\n%s", salida.String())
	}
}

// LA INGESTA SE ANOTA EN EL LEDGER, encadenada, y con lo que hace falta para
// contrastarla sin volver a tener el fichero.
func TestLaIngestaSeAnotaEnElLedgerYSeEncadena(t *testing.T) {
	dir := t.TempDir()
	reg := filepath.Join(dir, "ledger.json")
	csv := escribirCSV(t, csvHostil)
	correr := func() {
		t.Helper()
		var salida, errores bytes.Buffer
		if codigo := cmdAccesos([]string{
			"--fichero", csv, "--sistema", "erp", "--quien", "u-042",
			"--ahora", "2026-09-01T09:00:00Z", "--ledger", reg,
		}, &salida, &errores); codigo != 0 {
			t.Fatalf("codigo %d: %s", codigo, errores.String())
		}
		if !strings.Contains(salida.String(), "Ingesta anotada en") {
			t.Fatalf("no dice que lo ha anotado:\n%s", salida.String())
		}
	}
	correr()
	correr()

	b, err := os.ReadFile(reg)
	if err != nil {
		t.Fatal(err)
	}
	var l ledger.Ledger
	if err := json.Unmarshal(b, &l); err != nil {
		t.Fatal(err)
	}
	if len(l.Entradas) != 2 {
		t.Fatalf("se esperaban dos entradas encadenadas y hay %d", len(l.Entradas))
	}
	// ENCADENADAS: la segunda apunta a la primera. Si no, no es un ledger, es un
	// fichero al que se le anaden lineas.
	if l.Entradas[1].HashPrevio != l.Entradas[0].HashCadena || l.Entradas[0].HashCadena == "" {
		t.Fatalf("la cadena esta rota:\n%+v", l.Entradas)
	}
	var carga accesos.CargaDeApertura
	if err := json.Unmarshal(l.Entradas[0].Carga, &carga); err != nil {
		t.Fatal(err)
	}
	if carga.Hash == "" || carga.Sello == "" {
		t.Error("la anotacion no lleva ni el hash del fichero ni el sello de la lectura")
	}
	if carga.Codificacion == "" || carga.Separador == "" {
		t.Error("la anotacion no dice como se leyo el fichero: son las dos decisiones que " +
			"cambian lo que se leyo, y un auditor que repita la lectura las necesita")
	}
	if carga.Accesos != 4 || carga.LineasIlegibles != 1 || carga.FilasRepetidas != 1 {
		t.Errorf("los recuentos no llegan al ledger: %+v", carga)
	}
	if l.Entradas[0].Actor != "u-042" {
		t.Errorf("no consta quien subio el fichero: %q", l.Entradas[0].Actor)
	}
}

// LA ANOTACION DEL LEDGER NO LLEVA NI UN DATO PERSONAL.
//
// POR QUE SE COMPRUEBA AQUI Y NO SOLO EN EL PAQUETE: el ledger es lo que VIAJA.
// Se copia, se ancla, se ensena a un auditor y acaba en un expediente que sale
// de la organizacion. La instantanea se queda en memoria y el fichero se queda
// donde lo dejo quien lo subio; la anotacion es la unica pieza de esta cadena
// que se distribuye, asi que es donde una fuga cuesta.
//
// Lo que se anota son recuentos, hashes y quien lo subio. Ni un nombre, ni un
// identificador de cuenta ajeno, ni un permiso. El hash es suficiente para
// contrastarlo: quien tenga el fichero comprueba que la revision fue sobre el,
// y quien no lo tenga no aprende nada de nadie.
func TestLoQueSeAnotaEnElLedgerNoLlevaNiUnNombre(t *testing.T) {
	dir := t.TempDir()
	reg := filepath.Join(dir, "ledger.json")
	var salida, errores bytes.Buffer
	if codigo := cmdAccesos([]string{
		"--fichero", escribirCSV(t, csvHostil), "--sistema", "erp", "--quien", "u-042",
		"--ahora", "2026-09-01T09:00:00Z", "--ledger", reg,
	}, &salida, &errores); codigo != 0 {
		t.Fatalf("codigo %d: %s", codigo, errores.String())
	}
	b, err := os.ReadFile(reg)
	if err != nil {
		t.Fatal(err)
	}
	anotado := string(b)
	// Todo lo que sale del fichero de personas: nombres, DNI, cuentas y
	// permisos. Nada de esto puede estar en lo que viaja.
	for _, prohibido := range []string{
		"Ana Martinez", "Luis Gil", "Eva Roca", // rotulos
		"12345678Z", "87654321X", "11111111H", // los que ni siquiera se guardan
		"u1", "u2", "u3", // identificadores de las cuentas revisadas
		"admin", "lector", // los permisos
		"Ventas", "Finanzas", // el departamento
	} {
		if strings.Contains(anotado, prohibido) {
			t.Errorf("%q ha llegado al ledger, que es la pieza de esta cadena que VIAJA:\n%s",
				prohibido, anotado)
		}
	}
	// Y lo que SI tiene que estar, porque sin ello la anotacion no sirve de
	// nada: quien lo subio y con que se contrasta.
	for _, quiero := range []string{"u-042", "erp"} {
		if !strings.Contains(anotado, quiero) {
			t.Errorf("el ledger no dice %q, y sin eso la anotacion no se puede atar a nadie:\n%s",
				quiero, anotado)
		}
	}
}

// UN LEDGER QUE NO SE ENTIENDE NO SE SOBRESCRIBE. Machacar un registro
// append-only ilegible es peor que no escribir: se pierde lo que hubiera dentro
// y ademas queda un fichero que parece bueno.
func TestUnLedgerIlegibleNoSeMachaca(t *testing.T) {
	dir := t.TempDir()
	reg := filepath.Join(dir, "ledger.json")
	basura := []byte("esto no es json{{{")
	if err := os.WriteFile(reg, basura, 0o600); err != nil {
		t.Fatal(err)
	}
	var salida, errores bytes.Buffer
	codigo := cmdAccesos([]string{
		"--fichero", escribirCSV(t, csvHostil), "--sistema", "erp", "--quien", "u-042",
		"--ahora", "2026-09-01T09:00:00Z", "--ledger", reg,
	}, &salida, &errores)
	if codigo == 0 {
		t.Fatal("ha salido con exito habiendo fallado el registro")
	}
	if !strings.Contains(errores.String(), "NO se ha tocado") {
		t.Errorf("el error no dice que el fichero se ha respetado: %s", errores.String())
	}
	tras, err := os.ReadFile(reg)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(tras, basura) {
		t.Fatalf("el ledger ilegible se ha sobrescrito:\n%s", tras)
	}
}
