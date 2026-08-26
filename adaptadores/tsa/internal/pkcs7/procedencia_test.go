package pkcs7

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// rutaDeAguasArriba es de donde salio esta copia. Se escribe una vez y la usan
// las dos puertas de este fichero.
const rutaDeAguasArriba = "github.com/digitorus/pkcs7"

// La procedencia del codigo vendorizado, comprobada y no prometida.
//
// POR QUE EXISTE. Un directorio con codigo de otro y sin procedencia es codigo
// huerfano: nadie sabe de que commit salio, ni si alguien lo ha tocado, ni con
// que licencia se esta distribuyendo. Y una tabla de procedencia escrita a mano
// envejece a la primera edicion que nadie anota, que es exactamente la forma en
// la que este proyecto ya se quemo una vez con una pseudo-version de tres años.
//
// Asi que la tabla de LEEME.md no es documentacion, es un manifiesto que se
// ejecuta: este test recalcula el sha256 de cada fichero vendorizado y lo
// contrasta con el que la tabla declara.
//
// QUE PRUEBA Y QUE NO, dicho con precision porque es facil creerse de mas. Este
// test prueba que **nadie ha editado el codigo ajeno sin dejarlo escrito**. NO
// prueba que la columna de "sha256 aguas arriba" sea cierta: eso se comprueba
// contra el proxy de modulos con el comando que hay en LEEME.md, que ademas trae
// la firma de la suma de comprobacion.

// ficherosVendorizados son los que la tabla tiene que cubrir: el codigo ajeno y
// su licencia. Los _test.go son nuestros y no van en la tabla; testdata
// tampoco, que son semillas del fuzzer y no codigo de nadie.
func ficherosVendorizados(t *testing.T) map[string]string {
	t.Helper()
	entradas, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, e := range entradas {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		esCodigo := strings.HasSuffix(n, ".go") && !strings.HasSuffix(n, "_test.go")
		if !esCodigo && n != "LICENSE" {
			continue
		}
		b, err := os.ReadFile(n) // #nosec G304 -- nombre que sale de leer este mismo directorio
		if err != nil {
			t.Fatal(err)
		}
		s := sha256.Sum256(b)
		out[n] = hex.EncodeToString(s[:])
	}
	// Sin suelo, el dia que este recorrido deje de encontrar ficheros el test
	// compararia dos mapas vacios y saldria verde.
	if len(out) < 5 {
		t.Fatalf("solo se han encontrado %d ficheros vendorizados (%v). O el directorio ha "+
			"adelgazado, o este recorrido ha dejado de reconocerlos, y en los dos casos la "+
			"procedencia estaria comprobando el vacio", len(out), out)
	}
	return out
}

// filaDeTabla saca (fichero, sha256 local) de las filas de la tabla de
// procedencia de LEEME.md. La forma es:
//
//	| `ber.go` | `<sha arriba>` | `<sha aqui>` | estado |
var filaDeTabla = regexp.MustCompile("(?m)^\\| `([^`]+)` \\| `([0-9a-f]{64})` \\| `([0-9a-f]{64})` \\|")

func procedenciaDeclarada(t *testing.T, md string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, m := range filaDeTabla.FindAllStringSubmatch(md, -1) {
		out[m[1]] = m[3]
	}
	return out
}

// discrepancias contrasta lo declarado con lo que hay en disco, en las DOS
// direcciones. La direccion que suele faltar es la segunda: un fichero nuevo
// que nadie anoto pasa desapercibido si solo se recorre la tabla.
func discrepancias(declarado, enDisco map[string]string) []string {
	var out []string
	for fichero, sha := range enDisco {
		d, ok := declarado[fichero]
		if !ok {
			out = append(out, fichero+": esta en el directorio y NO esta en la tabla de procedencia")
			continue
		}
		if d != sha {
			out = append(out, fichero+": la tabla declara "+d[:16]+"... y el fichero es "+sha[:16]+"...")
		}
	}
	for fichero := range declarado {
		if _, ok := enDisco[fichero]; !ok {
			out = append(out, fichero+": la tabla lo declara y NO esta en el directorio")
		}
	}
	sort.Strings(out)
	return out
}

func TestElVendorizadoEsElQueDiceLaProcedencia(t *testing.T) {
	b, err := os.ReadFile("LEEME.md")
	if err != nil {
		t.Fatalf("no esta LEEME.md (%v). Un directorio con codigo de otro y sin procedencia "+
			"es codigo huerfano: no se sabe de que commit salio ni con que licencia se "+
			"distribuye", err)
	}
	md := string(b)
	declarado := procedenciaDeclarada(t, md)
	enDisco := ficherosVendorizados(t)

	if d := discrepancias(declarado, enDisco); len(d) > 0 {
		t.Errorf("la procedencia y el directorio no dicen lo mismo:\n  %s\n"+
			"  Alguien ha tocado codigo vendorizado sin anotarlo, o ha anadido un fichero\n"+
			"  sin decir de donde sale.\n"+
			"  Arreglo: portar el cambio a conciencia (LEEME.md dice como se sigue aguas\n"+
			"  arriba) y actualizar la tabla con `sha256sum ber.go pkcs7.go verify.go sign.go LICENSE`.",
			strings.Join(d, "\n  "))
	}
	t.Logf("%d ficheros vendorizados, todos con su sha256 declarado", len(enDisco))
}

// La procedencia tiene que traer lo que hace falta para AUDITARLA y para
// SEGUIRLA. Sin el commit no se sabe contra que diffear; sin el procedimiento,
// el deber heredado se queda en una intencion.
func TestLaProcedenciaTraeCommitLicenciaYProcedimiento(t *testing.T) {
	b, err := os.ReadFile("LEEME.md")
	if err != nil {
		t.Fatal(err)
	}
	md := string(b)

	if !regexp.MustCompile("`[0-9a-f]{40}`").MatchString(md) {
		t.Error("LEEME.md no trae el commit completo de aguas arriba (40 hexadecimales). " +
			"Sin el, el `git diff` que sigue los arreglos ajenos no tiene contra que diffear")
	}
	for _, quiero := range []struct{ texto, porque string }{
		{"github.com/digitorus/pkcs7", "de que repositorio sale el codigo"},
		{"MIT", "con que licencia se esta distribuyendo codigo ajeno"},
		{"v0.0.0-20250729175123-57bd227bfa2f", "la pseudo-version exacta, que es lo que va en go.mod"},
		{"git diff 57bd227bfa2f..origin/master", "el comando concreto que sigue los arreglos de aguas arriba"},
		{"dependabot", "el aviso de que esta actualizacion NO la vigila nadie automaticamente"},
	} {
		if !strings.Contains(md, quiero.texto) {
			t.Errorf("LEEME.md no dice %q, que es lo que responde a: %s", quiero.texto, quiero.porque)
		}
	}
	if !strings.Contains(md, "LICENSE") {
		t.Error("LEEME.md no menciona el fichero LICENSE, que es lo que hace legal distribuir esto")
	}
}

// CONTROL NEGATIVO del comparador, y se muta FUERA de lo que el test conoce: no
// se toca la tabla de verdad ni el directorio de verdad, se le dan al detector
// tres situaciones sinteticas y se exige que cace las tres. Sin esto, el verde
// de arriba solo dice que hoy coinciden dos mapas, no que el detector sepa
// distinguirlos.
func TestElDetectorDeProcedenciaCazaLasTresFormasDeMentir(t *testing.T) {
	const a = "1111111111111111111111111111111111111111111111111111111111111111"
	const b = "2222222222222222222222222222222222222222222222222222222222222222"

	casos := []struct {
		nombre             string
		declarado, enDisco map[string]string
		esperaSubcadena    string
	}{
		{
			nombre:          "fichero tocado sin anotarlo",
			declarado:       map[string]string{"ber.go": a},
			enDisco:         map[string]string{"ber.go": b},
			esperaSubcadena: "la tabla declara",
		},
		{
			nombre:          "fichero nuevo que nadie anoto",
			declarado:       map[string]string{"ber.go": a},
			enDisco:         map[string]string{"ber.go": a, "colado.go": b},
			esperaSubcadena: "NO esta en la tabla",
		},
		{
			nombre:          "fila que se quedo despues de borrar el fichero",
			declarado:       map[string]string{"ber.go": a, "fantasma.go": b},
			enDisco:         map[string]string{"ber.go": a},
			esperaSubcadena: "NO esta en el directorio",
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			d := discrepancias(c.declarado, c.enDisco)
			if len(d) == 0 {
				t.Fatalf("el detector no ha cazado %q. Mientras eso pase, el verde de "+
					"TestElVendorizadoEsElQueDiceLaProcedencia no significa nada", c.nombre)
			}
			if !strings.Contains(strings.Join(d, "\n"), c.esperaSubcadena) {
				t.Fatalf("el detector caza %q pero no lo explica: %v", c.nombre, d)
			}
		})
	}
	// Y que NO grita cuando todo cuadra: un detector que grita por todo se
	// desactiva la primera semana y entonces no vigila nada.
	if d := discrepancias(map[string]string{"ber.go": a}, map[string]string{"ber.go": a}); len(d) != 0 {
		t.Fatalf("falso positivo sobre una tabla que cuadra: %v", d)
	}
}

// EL COTEJO CON AGUAS ARRIBA SE FUE DE AQUI EL 26-08-2026, y conviene contar
// por que en vez de dejar el hueco.
//
// Vivia aqui un TestLosDosParsersSiguenSiendoElMismoCodigo que comparaba funcion
// a funcion la copia con el original, y localizaba el original con
// `go list -m github.com/digitorus/pkcs7`. Ese dia el modulo salio de `go.mod`
// entero, asi que ya no hay cache de modulos donde mirar y el test se quedo sin
// poder ejecutarse.
//
// La respuesta correcta NO era devolver la dependencia para poder vigilarla:
// eso es la guarda que llega a lo que vigila por un camino que el producto ya no
// usa, que es la numero 16 de la familia. La respuesta es que ese cotejo **no
// era una puerta de PR, era vigilancia**: pregunta por algo de FUERA que cambia
// solo, no por si este cambio esta bien.
//
// Vive ahora en `herramientas/cotejapkcs7`, con sus propios tests offline, y lo
// ejecuta el canario mensual de `.github/workflows/vigilancia.yml` contra un
// clon fresco.
//
// LO QUE SE QUEDA AQUI es lo que si se puede comprobar sin red y en cada
// cambio: que nadie ha tocado codigo ajeno sin anotarlo. Eso lo hace
// TestElVendorizadoEsElQueDiceLaProcedencia, arriba, recalculando los sha256.
