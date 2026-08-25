package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dutiq/adaptadores/actualizador"
)

func update(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var salida, errores bytes.Buffer
	codigo := cmdUpdate(args, &salida, &errores)
	return salida.String(), errores.String(), codigo
}

// canalConDosVersiones deja un canal de directorio listo y devuelve su ruta.
func canalConDosVersiones(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	var vs []actualizador.Version
	for _, v := range []string{"v0.2.0", "v0.1.0"} {
		contenido := []byte("binario de " + v)
		ruta := filepath.Join(dir, v)
		if err := os.MkdirAll(ruta, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(ruta, "dutiq"), contenido, 0o600); err != nil {
			t.Fatal(err)
		}
		suma := sha256.Sum256(contenido)
		vs = append(vs, actualizador.Version{
			Version:  v,
			Notas:    "lo que cambia en " + v,
			Ficheros: map[string]string{"dutiq": hex.EncodeToString(suma[:])},
		})
	}
	if err := actualizador.EscribirCatalogo(dir, vs); err != nil {
		t.Fatal(err)
	}
	return dir
}

// `dutiq update` a secas NO actualiza. En un producto que vigila plazos
// legales, leer que cambia antes de aplicarlo no es un paso de mas.
func TestUpdateSinAplicarNoTocaNadaYDiceComoAplicar(t *testing.T) {
	raiz := t.TempDir()
	canal := canalConDosVersiones(t)

	salida, errores, codigo := update(t, "--raiz", raiz, "--canal", canal)
	if codigo != 0 {
		t.Fatalf("consultar devolvio %d: %s", codigo, errores)
	}
	if _, err := os.Stat(filepath.Join(raiz, "dutiq")); !os.IsNotExist(err) {
		t.Fatal("consultar ha instalado algo. `dutiq update` sin --aplicar no puede tocar nada")
	}
	if !strings.Contains(salida, "lo que cambia en v0.2.0") {
		t.Error("no se ensenan las notas de la version, o sea que hay que actualizar a ciegas")
	}
	if !strings.Contains(salida, "--aplicar") {
		t.Error("no se dice como aplicarla")
	}
}

// El comando que se sugiere tiene que funcionar tal cual se copia. Uno que se
// deja por el camino las opciones que el operador acaba de teclear es una
// trampa disfrazada de ayuda.
func TestLosComandosSugeridosLlevanLasOpcionesQueHacenFalta(t *testing.T) {
	raiz := t.TempDir()
	canal := canalConDosVersiones(t)

	consulta, _, _ := update(t, "--raiz", raiz, "--canal", canal)
	if !strings.Contains(consulta, "--raiz "+raiz) {
		t.Errorf("el comando sugerido para aplicar no lleva --raiz, asi que copiado actualiza "+
			"otra instalacion:\n%s", consulta)
	}

	aplicada, errores, codigo := update(t, "--raiz", raiz, "--canal", canal, "--aplicar")
	if codigo != 0 {
		t.Fatalf("aplicar devolvio %d: %s", codigo, errores)
	}
	if !strings.Contains(aplicada, "--raiz "+raiz+" --deshacer ") {
		t.Errorf("el comando sugerido para deshacer no lleva --raiz:\n%s", aplicada)
	}

	// Y el punto que sugiere existe de verdad: se extrae de la salida y se usa.
	punto := puntoDe(t, aplicada)
	vuelta, errores, codigo := update(t, "--raiz", raiz, "--deshacer", punto)
	if codigo != 0 {
		t.Fatalf("el punto que sugirio la propia salida no sirve para deshacer: %d %s", codigo, errores)
	}
	if !strings.Contains(vuelta, punto) {
		t.Errorf("la vuelta atras no confirma a que punto se volvio:\n%s", vuelta)
	}
	if _, err := os.Stat(filepath.Join(raiz, "dutiq")); !os.IsNotExist(err) {
		t.Error("tras deshacer sigue el binario que instalo la actualizacion")
	}
}

// Deshacer sin canal tiene que funcionar: volver atras no puede depender de que
// el sitio de donde vino la version siga en pie.
func TestSePuedeDeshacerAunqueElCanalYaNoEste(t *testing.T) {
	raiz := t.TempDir()
	canal := canalConDosVersiones(t)
	aplicada, _, codigo := update(t, "--raiz", raiz, "--canal", canal, "--aplicar")
	if codigo != 0 {
		t.Fatal("aplicar fallo")
	}
	punto := puntoDe(t, aplicada)
	if err := os.RemoveAll(canal); err != nil {
		t.Fatal(err)
	}
	if _, errores, codigo := update(t, "--raiz", raiz, "--deshacer", punto); codigo != 0 {
		t.Fatalf("con el canal desaparecido ya no se puede volver atras: %d %s", codigo, errores)
	}
}

func TestDeshacerUnPuntoInventadoTerminaConError(t *testing.T) {
	raiz := t.TempDir()
	_, errores, codigo := update(t, "--raiz", raiz, "--deshacer", "me-lo-invento")
	if codigo == 0 {
		t.Fatal("deshacer un punto inventado termino con exito: el operador creeria que ha " +
			"vuelto atras y no ha vuelto")
	}
	if !strings.Contains(errores, "--puntos") {
		t.Errorf("el error no dice como ver los puntos que si existen: %s", errores)
	}
}

func TestAplicarSinCanalSeNiegaEnVezDeInventarseElOrigen(t *testing.T) {
	_, errores, codigo := update(t, "--raiz", t.TempDir(), "--aplicar")
	if codigo == 0 {
		t.Fatal("se aplico una version sin decir de donde sale")
	}
	if !strings.Contains(errores, "--canal") {
		t.Errorf("el error no dice que falta: %s", errores)
	}
}

func TestSinPuntosLoDiceEnVezDeSalirVacio(t *testing.T) {
	salida, _, codigo := update(t, "--raiz", t.TempDir(), "--puntos")
	if codigo != 0 {
		t.Fatalf("listar puntos en una instalacion nueva devolvio %d", codigo)
	}
	if !strings.Contains(salida, "no se ha actualizado") {
		t.Errorf("una lista vacia sin explicacion se lee como un fallo: %s", salida)
	}
}

// Una actualizacion a medias se avisa haga lo que haga el operador: es la unica
// situacion en la que la instalacion puede estar mintiendo sobre su version.
func TestUnaActualizacionAMediasSeAvisaSiempreYSeRepara(t *testing.T) {
	raiz := t.TempDir()
	canal := canalConDosVersiones(t)
	aplicada, _, codigo := update(t, "--raiz", raiz, "--canal", canal, "--aplicar")
	if codigo != 0 {
		t.Fatal("aplicar fallo")
	}
	punto := puntoDe(t, aplicada)

	// Se simula el proceso que muere antes de borrar su marcador.
	marcador := filepath.Join(raiz, actualizador.DirInterno, "actualizacion.json")
	contenido := `{"punto":"` + punto + `","version":"v0.2.0","inicio":"2026-08-25T10:00:00Z","pid":1}`
	if err := os.WriteFile(marcador, []byte(contenido), 0o600); err != nil {
		t.Fatal(err)
	}

	_, errores, _ := update(t, "--raiz", raiz, "--puntos")
	if !strings.Contains(errores, "a medias") {
		t.Errorf("listando puntos no se avisa de la actualizacion a medias: %s", errores)
	}
	if !strings.Contains(errores, "--reparar") {
		t.Errorf("el aviso no dice como salir del paso: %s", errores)
	}

	salida, errores, codigo := update(t, "--raiz", raiz, "--reparar")
	if codigo != 0 {
		t.Fatalf("reparar devolvio %d: %s", codigo, errores)
	}
	if !strings.Contains(salida, punto) {
		t.Errorf("reparar no dice a que punto ha vuelto: %s", salida)
	}
	if _, err := os.Stat(marcador); !os.IsNotExist(err) {
		t.Error("tras reparar sigue el marcador de la actualizacion a medias")
	}
	// Y ahora no queda nada a medias.
	if _, errores, _ := update(t, "--raiz", raiz, "--puntos"); strings.Contains(errores, "a medias") {
		t.Errorf("tras reparar se sigue avisando de una actualizacion a medias: %s", errores)
	}
}

func TestRepararSinNadaQueRepararNoInventaTrabajo(t *testing.T) {
	salida, _, codigo := update(t, "--raiz", t.TempDir(), "--reparar")
	if codigo != 0 {
		t.Fatalf("reparar sin nada roto devolvio %d", codigo)
	}
	if !strings.Contains(salida, "Nada que reparar") {
		t.Errorf("no se dice que no habia nada que hacer: %s", salida)
	}
}

// puntoDe saca el identificador del punto de retorno de la salida de --aplicar.
func puntoDe(t *testing.T, salida string) string {
	t.Helper()
	const marca = "--deshacer "
	i := strings.Index(salida, marca)
	if i < 0 {
		t.Fatalf("la salida no sugiere ningun comando para deshacer:\n%s", salida)
	}
	resto := salida[i+len(marca):]
	if j := strings.IndexAny(resto, " \n\r"); j >= 0 {
		resto = resto[:j]
	}
	if resto == "" {
		t.Fatalf("la salida sugiere deshacer sin decir que punto:\n%s", salida)
	}
	return resto
}
