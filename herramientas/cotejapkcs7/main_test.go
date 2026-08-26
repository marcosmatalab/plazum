package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// El cotejo se prueba OFFLINE, contra directorios que se fabrican aqui.
//
// No se prueba contra el original de verdad a proposito: esa comparacion es
// vigilancia (la hace el canario mensual con un clon fresco) y depende de que
// aguas arriba no se mueva, que es justo lo que se quiere que cambie. Lo que se
// prueba aqui es que la herramienta DISTINGUE, que es lo que decide si el
// canario sirve para algo.

const cabeceraNuestra = `package pkcs7

// Copia vendorizada. Este comentario no puede contar como diferencia.
`

func escribir(t *testing.T, dir, nombre, contenido string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, nombre), []byte(contenido), 0o600); err != nil {
		t.Fatal(err)
	}
}

// paresDePrueba deja dos directorios con los cuatro ficheros cotejados. Los
// cuatro tienen que existir en los dos lados o la herramienta sale con 2.
func paresDePrueba(t *testing.T, nuestro, suyo map[string]string) (string, string) {
	t.Helper()
	dirN, dirS := t.TempDir(), t.TempDir()
	// Relleno para pasar el suelo de funciones: la herramienta se niega a dar
	// un veredicto con pocas funciones, y con razon.
	var relleno strings.Builder
	for i := 0; i < MinimoDeFunciones+5; i++ {
		relleno.WriteString("func relleno")
		relleno.WriteString(string(rune('a' + i%26)))
		relleno.WriteString(string(rune('a' + i/26)))
		relleno.WriteString("() {}\n")
	}
	for _, f := range ficherosCotejados {
		n, s := nuestro[f], suyo[f]
		if f == "ber.go" {
			n = relleno.String() + n
			s = relleno.String() + s
		}
		escribir(t, dirN, f, cabeceraNuestra+n)
		escribir(t, dirS, f, "package pkcs7\n"+s)
	}
	return dirN, dirS
}

func correr(t *testing.T, nuestro, suyo string) (int, string) {
	t.Helper()
	var b bytes.Buffer
	return cotejar(nuestro, suyo, &b), b.String()
}

// CONTROL POSITIVO, y va primero: dos copias con el mismo codigo y comentarios
// distintos NO son una diferencia. Si esto fallara, la herramienta gritaria por
// todo y se acabaria desactivando, que es peor que no tenerla.
func TestLosComentariosNoCuentanComoDiferencia(t *testing.T) {
	mismo := map[string]string{
		"ber.go":    "func leer() int { return 1 }\n",
		"pkcs7.go":  "func Parse() {}\n",
		"verify.go": "func parseSignedData() {}\nfunc getSignatureAlgorithm() {}\n",
		"sign.go":   "func marshalAttributes() {}\n",
	}
	dirN, dirS := paresDePrueba(t, mismo, mismo)
	codigo, salida := correr(t, dirN, dirS)
	// Los recortes declarados no existen en estos ficheros de mentira, asi que
	// se esperan sus avisos de "nombrado y no visto". Lo que NO puede haber es
	// una diferencia de codigo.
	if strings.Contains(salida, "YA NO ES EL MISMO CODIGO") {
		t.Errorf("un comentario distinto se ha contado como diferencia de codigo:\n%s", salida)
	}
	if codigo == 2 {
		t.Fatalf("la herramienta no ha podido cotejar:\n%s", salida)
	}
}

// LO QUE TIENE QUE CAZAR: una funcion que difiere y no esta declarada.
func TestUnaDiferenciaNoDeclaradaSeCaza(t *testing.T) {
	nuestro := map[string]string{
		"ber.go":    "func leer() int { return 1 }\n",
		"pkcs7.go":  "",
		"verify.go": "",
		"sign.go":   "",
	}
	suyo := map[string]string{
		"ber.go":    "func leer() int { return 2 }\n", // <- otra cosa
		"pkcs7.go":  "",
		"verify.go": "",
		"sign.go":   "",
	}
	dirN, dirS := paresDePrueba(t, nuestro, suyo)
	codigo, salida := correr(t, dirN, dirS)
	if codigo != 1 {
		t.Fatalf("una diferencia no declarada tenia que salir con 1 y salio %d:\n%s", codigo, salida)
	}
	if !strings.Contains(salida, "ber.go:leer") {
		t.Errorf("no se dice QUE funcion difiere:\n%s", salida)
	}
}

// Una funcion que solo existe en la copia: codigo propio infiltrado.
func TestUnaFuncionQueSoloEstaEnLaCopiaSeCaza(t *testing.T) {
	nuestro := map[string]string{
		"ber.go": "func leer() int { return 1 }\nfunc inventada() {}\n",
	}
	suyo := map[string]string{"ber.go": "func leer() int { return 1 }\n"}
	for _, f := range ficherosCotejados[1:] {
		nuestro[f], suyo[f] = "", ""
	}
	dirN, dirS := paresDePrueba(t, nuestro, suyo)
	codigo, salida := correr(t, dirN, dirS)
	if codigo != 1 || !strings.Contains(salida, "inventada") {
		t.Fatalf("una funcion que solo esta en la copia tenia que cazarse (%d):\n%s", codigo, salida)
	}
}

// EL SUELO: con pocas funciones no se da veredicto. Es la diferencia entre "no
// hay diferencias" y "no he mirado".
func TestConPocasFuncionesNoSeDaVeredicto(t *testing.T) {
	dirN, dirS := t.TempDir(), t.TempDir()
	for _, f := range ficherosCotejados {
		escribir(t, dirN, f, "package pkcs7\nfunc una() {}\n")
		escribir(t, dirS, f, "package pkcs7\nfunc una() {}\n")
	}
	codigo, salida := correr(t, dirN, dirS)
	if codigo != 2 {
		t.Fatalf("con %d funciones tenia que negarse a dar veredicto (2) y salio %d:\n%s",
			MinimoDeFunciones, codigo, salida)
	}
	if !strings.Contains(salida, "PUERTA ROTA") {
		t.Errorf("no se dice que no ha podido mirar:\n%s", salida)
	}
}

// Sin el directorio de aguas arriba, tampoco hay veredicto. Un cotejo contra
// nada no es un cotejo limpio.
func TestSinElOriginalNoSeDaVeredicto(t *testing.T) {
	codigo, salida := correr(t, ".", "")
	if codigo != 2 {
		t.Fatalf("sin -suyo tenia que salir 2 y salio %d:\n%s", codigo, salida)
	}
	if !strings.Contains(salida, "git clone") {
		t.Errorf("no se dice como conseguir el original:\n%s", salida)
	}
}

// EL RECEPTOR EN LA CLAVE, que es el fallo que la herramienta se comio a si
// misma cuando vivia dentro de go test: `Parse` la funcion y `(x) Parse` el
// metodo se llaman igual, y sin el receptor uno pisaba al otro en el mapa.
func TestElReceptorEntraEnLaClave(t *testing.T) {
	fuente := "package pkcs7\nfunc Parse() {}\nfunc (r raw) Parse() {}\ntype raw struct{}\n"
	dir := t.TempDir()
	escribir(t, dir, "x.go", fuente)
	fns, err := funcionesDe(filepath.Join(dir, "x.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(fns) != 2 {
		t.Fatalf("la funcion y el metodo del mismo nombre tienen que ser dos entradas y son "+
			"%d: %v. Con una sola, la comparacion se hace contra la que quede, o sea por "+
			"accidente y no por identidad", len(fns), fns)
	}
	if _, hay := fns["(raw).Parse"]; !hay {
		t.Errorf("el metodo no lleva su receptor en la clave: %v", claves(fns))
	}
}

// Y el cotejo real, contra el arbol de verdad, SOLO para comprobar que la
// herramienta sabe leer la copia vendorizada tal como esta hoy. No compara con
// aguas arriba: eso es del canario.
func TestSabeLeerLaCopiaVendorizadaDeVerdad(t *testing.T) {
	base := filepath.Join("..", "..", "adaptadores", "tsa", "internal", "pkcs7")
	total := 0
	for _, f := range ficherosCotejados {
		fns, err := funcionesDe(filepath.Join(base, f))
		if err != nil {
			t.Fatalf("no puedo leer %s de la copia vendorizada: %v.\n"+
				"  Si el directorio se movio, esta herramienta y el canario que la usa estarian "+
				"cotejando el vacio", f, err)
		}
		total += len(fns)
	}
	if total < MinimoDeFunciones {
		t.Fatalf("la copia vendorizada tiene %d funciones y el suelo del cotejo es %d: el "+
			"canario se negaria a dar veredicto todos los meses", total, MinimoDeFunciones)
	}
	// Y los recortes declarados nombran funciones que EXISTEN. Un nombre mal
	// escrito ahi es una excepcion que no protege nada.
	for clave := range RecortesDeclarados {
		fichero, nombre, ok := strings.Cut(clave, ":")
		if !ok {
			t.Errorf("la clave %q no tiene la forma fichero:funcion", clave)
			continue
		}
		fns, err := funcionesDe(filepath.Join(base, fichero))
		if err != nil {
			t.Errorf("el recorte %q nombra un fichero ilegible: %v", clave, err)
			continue
		}
		if _, hay := fns[nombre]; !hay {
			t.Errorf("el recorte declarado %q nombra una funcion que no existe en la copia. "+
				"Una excepcion mal escrita no protege nada y tapa lo que venga detras", clave)
		}
	}
}

func claves(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
