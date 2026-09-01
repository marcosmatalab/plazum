package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// El censo minimo del ciclo: una cuenta con dos permisos y otra con uno.
const csvCiclo = "usuario;nombre;permiso\n" +
	"u1;Ana Martinez;admin\n" +
	"u1;Ana Martinez;lector\n" +
	"u2;Luis Gil;lector\n"

type banco struct {
	t     *testing.T
	csv   string
	reg   string
	sobre []string
}

func nuevoBanco(t *testing.T, contenido string) *banco {
	t.Helper()
	dir := t.TempDir()
	csv := filepath.Join(dir, "u.csv")
	if err := os.WriteFile(csv, []byte(contenido), 0o600); err != nil {
		t.Fatal(err)
	}
	reg := filepath.Join(dir, "l.json")
	b := &banco{t: t, csv: csv, reg: reg}
	b.sobre = []string{"--fichero", csv, "--ledger", reg, "--campana", "uar-h2"}
	return b
}

func (b *banco) abrir(cuando string) {
	b.t.Helper()
	var salida, errores bytes.Buffer
	codigo := cmdAccesos([]string{"ver", "--fichero", b.csv, "--sistema", "erp",
		"--quien", "u-042", "--ledger", b.reg, "--campana", "uar-h2", "--ahora", cuando},
		&salida, &errores)
	if codigo != 0 {
		b.t.Fatalf("abriendo: %s", errores.String())
	}
}

func (b *banco) correr(args ...string) (int, string, string) {
	b.t.Helper()
	var salida, errores bytes.Buffer
	codigo := cmdAccesos(append(append([]string{}, args...), b.sobre...), &salida, &errores)
	return codigo, salida.String(), errores.String()
}

// EL CICLO ENTERO, QUE ES LO QUE CONVIERTE EL MOTOR EN PRODUCTO.
//
// Una campana de accesos vive semanas y la revisan varias personas. Sin este
// camino, `nucleo/accesos` es una biblioteca: se abre, se decide, se cierra el
// portatil y no queda nada.
func TestElCicloDeUnaCampanaSobreviveEntreEjecuciones(t *testing.T) {
	b := nuevoBanco(t, csvCiclo)
	b.abrir("2026-09-01T09:00:00Z")

	// Decidir uno. Y la decision VUELVE en la ejecucion siguiente.
	if codigo, _, e := b.correr("decidir", "--quien", "ciso", "--cuenta", "u1",
		"--permiso", "admin", "--veredicto", "revocar", "--motivo", "cambio de puesto",
		"--ahora", "2026-09-02T10:00:00Z"); codigo != 0 {
		t.Fatalf("decidiendo: %s", e)
	}
	codigo, s, e := b.correr("cerrar", "--quien", "ciso", "--ahora", "2026-09-05T10:00:00Z")
	if codigo == 0 {
		t.Fatalf("se ha cerrado con dos accesos sin revisar:\n%s", s)
	}
	if !strings.Contains(e, "2 sin revisar") {
		t.Errorf("el cierre no dice cuantos faltan: %s", e)
	}
	// La revocacion de la ejecucion anterior sigue ahi: es lo que se comprueba.
	if !strings.Contains(e, "revocada:                    1") {
		t.Errorf("la decision de la ejecucion anterior no ha vuelto:\n%s", e)
	}

	// Delegar no termina, y lo dice al decirlo.
	codigo, s, e = b.correr("decidir", "--quien", "jefa", "--cuenta", "u2", "--permiso", "lector",
		"--veredicto", "delegar", "--a", "responsable-erp", "--ahora", "2026-09-03T09:00:00Z")
	if codigo != 0 {
		t.Fatalf("delegando: %s", e)
	}
	if !strings.Contains(s, "NO la termina") {
		t.Errorf("delegar no avisa de que no cierra nada:\n%s", s)
	}

	// Con la delegacion pendiente y el otro decidido, el cierre sigue bloqueado
	// Y POR LA DELEGACION, que es la rama que hay que aislar.
	if codigo, _, e := b.correr("decidir", "--quien", "ciso", "--cuenta", "u1",
		"--permiso", "lector", "--veredicto", "aprobar", "--ahora", "2026-09-02T10:05:00Z"); codigo != 0 {
		t.Fatalf("aprobando: %s", e)
	}
	codigo, _, e = b.correr("cerrar", "--quien", "ciso", "--ahora", "2026-09-05T10:00:00Z")
	if codigo == 0 {
		t.Fatal("se ha cerrado con una delegacion pendiente")
	}
	if !strings.Contains(e, "1 delegados") {
		t.Errorf("el cierre no dice que lo que falta es una delegacion: %s", e)
	}

	// El delegado decide y ahora si cierra.
	if codigo, _, e := b.correr("decidir", "--quien", "responsable-erp", "--cuenta", "u2",
		"--permiso", "lector", "--veredicto", "aprobar", "--ahora", "2026-09-04T09:00:00Z"); codigo != 0 {
		t.Fatalf("el delegado decidiendo: %s", e)
	}
	codigo, s, e = b.correr("cerrar", "--quien", "ciso", "--ahora", "2026-09-05T10:00:00Z")
	if codigo != 0 {
		t.Fatalf("no cierra con todo decidido: %s", e)
	}
	for _, quiero := range []string{"cerrada por ciso", "3 accesos, 3 decididos", "sello "} {
		if !strings.Contains(s, quiero) {
			t.Errorf("el cierre no dice %q:\n%s", quiero, s)
		}
	}

	// Y despues del cierre no entra nada.
	codigo, _, e = b.correr("decidir", "--quien", "otra", "--cuenta", "u1", "--permiso", "admin",
		"--veredicto", "aprobar", "--ahora", "2026-09-06T09:00:00Z")
	if codigo == 0 {
		t.Fatal("se ha admitido una decision despues de firmar")
	}
	if !strings.Contains(e, "ya esta cerrada") {
		t.Errorf("el error no lo dice: %s", e)
	}
}

// EL SELLO SE RECONSTRUYE CON LOS DATOS DE LA APERTURA, NO CON LOS DE AHORA.
//
// ESTE TEST EXISTE POR UN FALLO QUE SOLO SALIO EJECUTANDOLO. El sello cubre el
// fichero MAS quien lo subio, cuando, de que sistema y con que retencion. La
// primera version reconstruia con las banderas de la orden actual, asi que
// `--quien ciso` al decidir daba un sello distinto del que dejo `--quien u-042`
// al subir, y la campana NO SE REABRIA NUNCA. Ningun test de los que habia lo
// veia, porque todos construian el censo una sola vez.
//
// El caso que lo aisla: abre una persona y decide OTRA, que es lo normal.
func TestDecideOtraPersonaDistintaDeLaQueSubioElFichero(t *testing.T) {
	b := nuevoBanco(t, csvCiclo)
	b.abrir("2026-09-01T09:00:00Z")
	codigo, s, e := b.correr("decidir", "--quien", "una-persona-distinta", "--cuenta", "u2",
		"--permiso", "lector", "--veredicto", "aprobar", "--ahora", "2026-09-02T10:00:00Z")
	if codigo != 0 {
		t.Fatalf("quien decide no es quien subio el fichero, que es el caso NORMAL, y no se "+
			"puede decidir:\n%s", e)
	}
	if !strings.Contains(s, "aprobada:                    1") {
		t.Errorf("la decision no ha entrado:\n%s", s)
	}
}

// SI EL FICHERO CAMBIO, NO SE DECIDE SOBRE EL.
func TestConOtroFicheroLaCampanaNoSigue(t *testing.T) {
	b := nuevoBanco(t, csvCiclo)
	b.abrir("2026-09-01T09:00:00Z")
	if err := os.WriteFile(b.csv, []byte(csvCiclo+"u9;Nueva Persona;admin\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	codigo, _, e := b.correr("decidir", "--quien", "ciso", "--cuenta", "u2", "--permiso", "lector",
		"--veredicto", "aprobar", "--ahora", "2026-09-02T10:00:00Z")
	if codigo == 0 {
		t.Fatal("se ha decidido sobre un fichero distinto del que se subio")
	}
	if !strings.Contains(e, "no es el mismo sobre el que se abrio") {
		t.Errorf("el error no lo explica: %s", e)
	}
}

// LA CUENTA CON VARIOS PERMISOS SE PREGUNTA, NO SE ADIVINA.
func TestUnaCuentaConVariosAccesosNoSeDecideAlBulto(t *testing.T) {
	b := nuevoBanco(t, csvCiclo)
	b.abrir("2026-09-01T09:00:00Z")
	codigo, _, e := b.correr("decidir", "--quien", "ciso", "--cuenta", "u1",
		"--veredicto", "revocar", "--motivo", "se fue", "--ahora", "2026-09-02T10:00:00Z")
	if codigo == 0 {
		t.Fatal("se ha decidido sobre una cuenta con dos accesos sin decir cual")
	}
	for _, quiero := range []string{`--permiso "admin"`, `--permiso "lector"`, "Se revisa el ACCESO"} {
		if !strings.Contains(e, quiero) {
			t.Errorf("el error no dice %q: %s", quiero, e)
		}
	}
	// Y una cuenta que no existe tampoco se aproxima.
	codigo, _, e = b.correr("decidir", "--quien", "ciso", "--cuenta", "u17", "--permiso", "admin",
		"--veredicto", "revocar", "--motivo", "x", "--ahora", "2026-09-02T10:00:00Z")
	if codigo == 0 {
		t.Fatal("se ha decidido sobre una cuenta que no esta en el fichero")
	}
	if !strings.Contains(e, "No se elige por parecido") {
		t.Errorf("el error no lo dice: %s", e)
	}
}

// LA EXCUSA SOBREVIVE, con quien y por que, y desbloquea el cierre.
func TestLaExcusaSobreviveYDesbloqueaElCierre(t *testing.T) {
	con := "usuario;nombre;permiso\nu1;Ana Martinez;admin\n;Sin Cuenta;lector\nu2;Luis Gil;lector\n"
	b := nuevoBanco(t, con)
	b.abrir("2026-09-01T09:00:00Z")
	for _, c := range [][2]string{{"u1", "admin"}, {"u2", "lector"}} {
		if codigo, _, e := b.correr("decidir", "--quien", "ciso", "--cuenta", c[0],
			"--permiso", c[1], "--veredicto", "aprobar", "--ahora", "2026-09-02T10:00:00Z"); codigo != 0 {
			t.Fatalf("decidiendo %v: %s", c, e)
		}
	}
	codigo, _, e := b.correr("cerrar", "--quien", "ciso", "--ahora", "2026-09-05T10:00:00Z")
	if codigo == 0 {
		t.Fatal("se ha certificado completitud con una linea que nadie pudo leer")
	}
	if !strings.Contains(e, "no se pudieron leer") && !strings.Contains(e, "no se pudo leer") {
		t.Errorf("el error no habla de la linea ilegible: %s", e)
	}

	if codigo, _, e := b.correr("excusar", "--quien", "ciso", "--desde", "3",
		"--motivo", "fila de prueba del IdP", "--ahora", "2026-09-05T09:00:00Z"); codigo != 0 {
		t.Fatalf("excusando: %s", e)
	}
	codigo, s, e := b.correr("cerrar", "--quien", "ciso", "--ahora", "2026-09-05T10:00:00Z")
	if codigo != 0 {
		t.Fatalf("con la linea excusada tenia que cerrar: %s", e)
	}
	if !strings.Contains(s, "excusadas por ciso") || !strings.Contains(s, "fila de prueba del IdP") {
		t.Errorf("la excusa no sale en el informe del cierre:\n%s", s)
	}
}

// UNA DECISION MAL FORMADA NO LLEGA AL LEDGER.
//
// El orden importa y estaba escrito en un comentario sin nada que lo sostuviera:
// se registra en la campana ANTES de escribir. Al reves, una revocacion sin
// motivo quedaria anotada para siempre en un registro append-only y habria que
// convivir con ella, o peor, alguien la borraria a mano y rompería la cadena.
func TestUnaDecisionMalFormadaNoLlegaAlLedger(t *testing.T) {
	b := nuevoBanco(t, csvCiclo)
	b.abrir("2026-09-01T09:00:00Z")
	antes, err := os.ReadFile(b.reg)
	if err != nil {
		t.Fatal(err)
	}
	// Revocar sin motivo: la campana lo rechaza.
	codigo, _, e := b.correr("decidir", "--quien", "ciso", "--cuenta", "u2", "--permiso", "lector",
		"--veredicto", "revocar", "--ahora", "2026-09-02T10:00:00Z")
	if codigo == 0 {
		t.Fatal("se ha admitido una revocacion sin motivo")
	}
	if !strings.Contains(e, "motivo") {
		t.Errorf("el error no dice que falta el motivo: %s", e)
	}
	despues, err := os.ReadFile(b.reg)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(antes, despues) {
		t.Fatalf("el ledger ha cambiado con una decision que se rechazo:\n%s", despues)
	}
}

// Y EL LEDGER DE UNA CAMPANA ENTERA SIGUE SIN LLEVAR UN SOLO NOMBRE.
//
// Se comprueba sobre el ciclo COMPLETO y no sobre la apertura sola: la apertura
// ya lo cumplia, y lo que hay que saber es si lo sigue cumpliendo despues de
// cuatro decisiones, una excusa y un cierre. Una garantia que solo se mide en el
// primer paso es una garantia sobre el primer paso.
func TestElLedgerDeUnaCampanaEnteraNoLlevaNiUnNombre(t *testing.T) {
	b := nuevoBanco(t, csvCiclo)
	b.abrir("2026-09-01T09:00:00Z")
	for _, c := range [][2]string{{"u1", "admin"}, {"u1", "lector"}, {"u2", "lector"}} {
		if codigo, _, e := b.correr("decidir", "--quien", "ciso", "--cuenta", c[0],
			"--permiso", c[1], "--veredicto", "aprobar", "--ahora", "2026-09-02T10:00:00Z"); codigo != 0 {
			t.Fatalf("decidiendo %v: %s", c, e)
		}
	}
	if codigo, _, e := b.correr("cerrar", "--quien", "ciso", "--ahora", "2026-09-05T10:00:00Z"); codigo != 0 {
		t.Fatalf("cerrando: %s", e)
	}
	crudo, err := os.ReadFile(b.reg)
	if err != nil {
		t.Fatal(err)
	}
	anotado := string(crudo)
	for _, prohibido := range []string{"Ana Martinez", "Luis Gil", "u1", "u2", "admin", "lector"} {
		if strings.Contains(anotado, prohibido) {
			t.Errorf("%q ha llegado al ledger de una campana cerrada:\n%s", prohibido, anotado)
		}
	}
	// Y lo que si tiene que estar: quien actuo y con que se contrasta.
	for _, quiero := range []string{"u-042", "ciso", "erp"} {
		if !strings.Contains(anotado, quiero) {
			t.Errorf("el ledger no dice %q:\n%s", quiero, anotado)
		}
	}
}
