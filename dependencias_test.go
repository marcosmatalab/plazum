package plazum

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/internal/modulo"
)

// Las dos puertas del grafo de dependencias.
//
// POR QUE ESTAN EN LA RAIZ Y NO EN adaptadores/tsa. Lo que vigilan no es de un
// paquete: es del BINARIO y del modulo. Un test dentro del adaptador podria
// decir "yo no importo eso" mientras otro paquete lo arrastra.
//
// EL PORQUE DE FONDO, y es lo que convierte esto en producto y no en higiene.
// Hasta el 26-08-2026 la frase de este repositorio sobre `digitorus/pkcs7` era
// "ya no lo llamamos": cierta, y **imposible de comprobar para el comprador sin
// leerse el codigo**. En un producto de seguridad esa diferencia es la que
// separa una promesa de un hecho. "No esta" se comprueba con un comando, y ese
// comando es este test.

// TestElBinarioNoLlevaNadaDeDigitorus.
//
// `go list -deps ./cmd/plazum` enumera TODO lo que entra en el binario,
// transitivo incluido. Si `digitorus` aparece ahi, esta dentro, se llame o no
// se llame.
func TestElBinarioNoLlevaNadaDeDigitorus(t *testing.T) {
	deps := dependenciasDelBinario(t)

	for _, d := range deps {
		if strings.Contains(d, "digitorus") {
			t.Errorf(`el binario lleva %q dentro.

  El 26-08-2026 se quito github.com/digitorus/timestamp entero: la consulta
  RFC 3161 se construye en adaptadores/tsa/rfc3161_peticion.go y el TSTInfo se
  lee en rfc3161.go. Con timestamp fuera, github.com/digitorus/pkcs7 salio del
  grafo, porque era timestamp quien lo arrastraba.

  Que alguien lo haya devuelto significa una de dos: o hay otra vez dos parsers
  del mismo ASN.1 en el binario (ver docs/pendientes.md 53), o se ha importado
  el pkcs7 de aguas arriba en vez de la copia vendorizada de
  plazum/adaptadores/tsa/internal/pkcs7.

  Arreglo: quitar el import y, si hacia falta un campo del TSTInfo, anadirlo a
  infoSello en adaptadores/tsa/rfc3161.go.`, d)
		}
	}

	// Suelo, por el mismo motivo que el minimo de puerta.sh: si `go list` no
	// devuelve nada, "no hay digitorus" se leeria igual que "todo en orden".
	if len(deps) < 50 {
		t.Fatalf("go list -deps ./cmd/plazum ha devuelto solo %d paquetes. El binario tiene "+
			"muchos mas, asi que esta puerta estaria dando verde sin mirar nada", len(deps))
	}
	if !t.Failed() {
		t.Logf("%d paquetes en el binario, ninguno de digitorus", len(deps))
	}
}

// Y la otra mitad: tampoco puede volver como dependencia SOLO DE TESTS.
//
// `go list -deps ./cmd/plazum` no ve las dependencias de test, asi que sin esto
// alguien puede reintroducir el modulo en un `_test.go` y el binario seguiria
// limpio mientras `go.mod` vuelve a tener la fila. Ya pasó una vez con el
// pkcs7 transitivo: se importó en un test para vigilarlo, y eso solo lo revela
// go.mod.
func TestGoModNoVuelveATraerADigitorus(t *testing.T) {
	b, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("no puedo leer go.mod (%v). Si el fichero se movio, esta puerta estaria "+
			"comprobando el vacio", err)
	}
	if strings.Contains(string(b), "digitorus") {
		t.Error("go.mod ha vuelto a traer github.com/digitorus. Aunque sea solo para tests, " +
			"la fila reaparece en el modulo y con ella el deber de vigilarla. Si de verdad " +
			"hace falta, va con su fila en DEPENDENCIAS.md y su porque, que es lista cerrada")
	}
}

// TestGoModEstaOrdenado: `go mod tidy` no produce diff.
//
// POR QUE ES UNA PUERTA Y NO UNA MANIA. En este repositorio DEPENDENCIAS.md es
// LISTA CERRADA: anadir una dependencia exige una fila con su porque y su
// licencia. Un `go.mod` desordenado es esa lista **mintiendo por omision**: una
// fila que sobra, un `// indirect` que se quedo viejo o un modulo que ya nadie
// importa hacen que lo que dice el documento y lo que dice el modulo dejen de
// coincidir, y el documento es lo unico que lee una persona.
//
// SALIO DE UN CASO REAL, y por eso esta escrito: al importar el pkcs7
// transitivo dentro de dependencias_test.go para poder vigilarlo, su marcador
// `// indirect` se quedo viejo y `go mod tidy` empezo a producir diff. Nadie lo
// habria notado, porque nada lo miraba.
//
// Se usa `-diff`, que NO toca el fichero: un test que reescribe el repositorio
// para comprobarlo deja de ser un test y pasa a ser una edicion.
func TestGoModEstaOrdenado(t *testing.T) {
	cmd := exec.Command("go", "mod", "tidy", "-diff")
	salida, err := cmd.CombinedOutput()
	if err == nil {
		return // sin diferencias
	}
	// `go mod tidy -diff` sale distinto de cero cuando HAY diferencias, y
	// tambien cuando no puede ejecutarse. Se distinguen por la salida: un diff
	// de verdad trae el fichero.
	texto := string(salida)
	if !strings.Contains(texto, "go.mod") && !strings.Contains(texto, "go.sum") {
		t.Fatalf("no he podido ejecutar `go mod tidy -diff` (%v):\n%s\n"+
			"  Sin poder ejecutarlo esta puerta no comprueba nada. Si `go` no esta en el "+
			"PATH del entorno de pruebas, esto tiene que salir rojo y no verde", err, texto)
	}
	t.Errorf(`go.mod o go.sum no estan ordenados: %v

  En este repositorio DEPENDENCIAS.md es LISTA CERRADA, asi que un go.mod
  desordenado es esa lista mintiendo por omision: una fila que sobra o un
  marcador // indirect viejo hacen que el documento y el modulo dejen de decir
  lo mismo, y el documento es lo unico que lee una persona.

  Arreglo: go mod tidy, y si aparece o desaparece una dependencia, tocar
  DEPENDENCIAS.md EN EL MISMO COMMIT.

%s`, err, texto)
}

// dependenciasDelBinario devuelve lo que entra en cmd/plazum.
func dependenciasDelBinario(t *testing.T) []string {
	t.Helper()
	salida, err := exec.Command("go", "list", "-deps", "./cmd/plazum").Output()
	if err != nil {
		t.Fatalf("no puedo enumerar las dependencias del binario (%v). Sin eso, esta puerta "+
			"no comprueba nada y no puede darse por verde", err)
	}
	var out []string
	for _, l := range strings.Split(string(salida), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

// Y la propiedad que hoy es cierta y conviene fijar, porque es un argumento de
// venta y no un detalle: el binario no lleva NINGUNA dependencia externa.
//
// No es una aspiracion cumplida por casualidad. DEPENDENCIAS.md tiene planeadas
// cuatro para etapas futuras (sqlite, cel-go, extism, x/crypto), asi que el dia
// que entre la primera este test hay que cambiarlo A PROPOSITO y con su fila en
// el documento. Eso es exactamente lo que se quiere: que anadir la primera
// dependencia externa sea un acto consciente y no un `go get` de un martes.
func TestElBinarioNoLlevaNingunaDependenciaExterna(t *testing.T) {
	// SE PREGUNTA A `go list` QUIEN ES DE LA BIBLIOTECA ESTANDAR, no se adivina.
	//
	// La primera version usaba una heuristica: "un paquete estandar no tiene
	// punto en su primer tramo". Pasaba, y pasaba por casualidad: `go list
	// -deps` devuelve tambien el vendoring INTERNO de Go
	// (`vendor/golang.org/x/crypto/chacha20`, `crypto/internal/entropy/v1.0.0`),
	// que no son dependencias nuestras y que la heuristica dejaba fuera solo
	// porque su primer tramo es `vendor` o `crypto`.
	//
	// El campo `.Standard` de `go list` es la respuesta autoritativa, y ademas
	// hizo falta para escribir en el README un comando que de verdad no imprima
	// nada. Un argumento de venta que se comprueba con un comando tiene que
	// poder ejecutarse tal cual.
	salida, err := exec.Command("go", "list", "-deps",
		"-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", "./cmd/plazum").Output()
	if err != nil {
		t.Fatalf("no puedo enumerar las dependencias no estandar del binario: %v", err)
	}
	// El prefijo de casa se LEE de go.mod. Cableado aqui, el dia que el modulo
	// cambie de nombre este filtro deja de casar y el test empieza a contar como
	// "externo" el codigo propio. Paso el 28-08-2026 y salio rojo; en otras
	// cuatro puertas del repositorio el mismo cableado podia haber salido verde.
	// El porque entero, en internal/modulo.
	mod, err := modulo.Ruta()
	if err != nil {
		t.Fatalf("no se cual es la ruta del modulo (%v), asi que no puedo separar el "+
			"codigo de casa del de fuera", err)
	}
	var externas []string
	nuestras := 0
	for _, d := range strings.Split(string(salida), "\n") {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if modulo.EsDeCasa(d, mod) {
			nuestras++
			continue // codigo nuestro
		}
		externas = append(externas, d)
	}
	// Suelo: `go list -deps` sobre el binario devuelve decenas de paquetes
	// nuestros. Si devolviera cero, el filtro estaria mal y la lista de externas
	// saldria vacia por la razon equivocada.
	if nuestras < 10 {
		t.Fatalf("go list -deps solo ha devuelto %d paquetes de %s. O el binario ha "+
			"adelgazado a la nada, o el prefijo del modulo ha dejado de casar y esta "+
			"puerta estaria mirando el vacio", nuestras, mod)
	}
	if len(externas) > 0 {
		t.Errorf(`el binario ha dejado de ser solo biblioteca estandar: %v

  Hoy plazum se compila con CERO dependencias externas, y eso es un argumento
  de venta, no una casualidad: un producto de cumplimiento que se instala en la
  red del cliente y no arrastra codigo de nadie es algo que el comprador puede
  comprobar con un comando.

  DEPENDENCIAS.md tiene cuatro planeadas para etapas futuras (sqlite, cel-go,
  extism, x/crypto). El dia que entre la primera, este test se cambia A
  PROPOSITO y en el mismo commit que su fila del documento. Que ese dia haya que
  tocar un test es justo lo que se busca.`, externas)
	}
}
