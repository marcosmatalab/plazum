package pkcs7

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
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

// ---------------------------------------------------------------------------
// La copia vendorizada sigue siendo un subconjunto FIEL de aguas arriba
// ---------------------------------------------------------------------------
//
// ESTA PUERTA NACIO POR OTRO MOTIVO, Y ESE MOTIVO YA NO EXISTE. Se escribio
// porque `adaptadores/tsa` sacaba el veredicto de `timestamp.Parse` (que usa el
// pkcs7 de AGUAS ARRIBA) y comprobaba la firma sobre esta copia: dos lecturas
// independientes de los mismos bytes, sin ninguna identidad dentro de lo
// firmado que las atara. Lo unico que hacia que ese doble parseo no tuviera
// diferencial era que las dos copias fueran el mismo codigo, y esta puerta lo
// vigilaba.
//
// El 26-08-2026 se quito el segundo parser: el TSTInfo se lee con
// `encoding/asn1` sobre el `p7.Content` de ESTA copia (`adaptadores/tsa/rfc3161.go`).
// El pendiente 53 se murio en vez de quedarse vigilado, que es lo que se
// buscaba.
//
// LA PUERTA SE QUEDA, Y CON OTRA RAZON, que conviene decir para que nadie la
// borre creyendo que sobra: **la copia vendorizada tiene que seguir siendo un
// SUBCONJUNTO FIEL de aguas arriba**. Vendorizar significa heredar el deber de
// portar los arreglos ajenos a mano, y ese deber solo es manejable si la unica
// distancia con el original son recortes DECLARADOS. Sin esto, una edicion que
// nadie anote convierte la copia en un fork silencioso, y entonces el triaje
// de aguas arriba (LEEME.md) deja de significar nada porque ya no se sabe
// contra que se compara.
//
// Y sigue vigilando la otra mitad, la que no cambia: un recorte declarado que
// ya NO difiere se pone rojo, porque una excepcion caducada tapa el dia que
// vuelva a diferir.
func TestLosDosParsersSiguenSiendoElMismoCodigo(t *testing.T) {
	arriba := dirDeAguasArriba(t)
	comparados, declarados := 0, map[string]bool{}

	for fichero := range ficherosVendorizados(t) {
		if !strings.HasSuffix(fichero, ".go") {
			continue
		}
		nuestras := funcionesDe(t, filepath.Join(".", fichero))
		suyas := funcionesDe(t, filepath.Join(arriba, fichero))
		for nombre, nuestra := range nuestras {
			suya, existe := suyas[nombre]
			if !existe {
				// Una funcion que solo esta aqui es codigo NUESTRO dentro de la
				// copia, y eso tampoco puede pasar en silencio.
				t.Errorf("%s: %s existe en la copia y NO en aguas arriba. Una copia "+
					"vendorizada no inventa funciones: o viene de otro fichero de arriba "+
					"que no se vendorizo (dilo en LEEME.md) o es codigo propio infiltrado "+
					"en territorio ajeno", fichero, nombre)
				continue
			}
			comparados++
			clave := fichero + ":" + nombre
			motivo, esRecorte := recortesDeclarados[clave]
			if nuestra == suya {
				if esRecorte {
					// CONTROL NEGATIVO INCORPORADO: un recorte declarado que ya
					// no difiere es una declaracion caducada, y una lista de
					// excepciones que no se limpia acaba tapando cambios de
					// verdad.
					t.Errorf("%s esta declarado como recorte (%q) y ya es IDENTICO a "+
						"aguas arriba. Quitalo de recortesDeclarados: una excepcion que "+
						"no excepciona nada deja pasar el dia que vuelva a diferir", clave, motivo)
					declarados[clave] = true // visto: que no se avise dos veces del mismo
				}
				continue
			}
			if !esRecorte {
				t.Errorf(`%s YA NO ES EL MISMO CODIGO que aguas arriba, y no esta declarado como recorte.

  Por que importa, y no es una curiosidad de mantenimiento: adaptadores/tsa
  saca el veredicto de timestamp.Parse, que usa el pkcs7 de AGUAS ARRIBA, y
  comprueba la firma sobre pkcs7.Parse, que es ESTA copia. Son dos lecturas
  independientes de los mismos bytes y no las ata ninguna identidad dentro de
  lo firmado (invariante 7). Lo unico que las ataba era ser el mismo codigo.

  Con un diferencial entre los dos parsers, el verificador puede dar por bueno
  un TSTInfo cuya firma nunca comprobo.

  Arreglo, y hay que elegir:
    a) portar el cambio de aguas arriba a la copia (LEEME.md dice como),
    b) declararlo en recortesDeclarados con su motivo, si es deliberado, o
    c) dejar de derivar el veredicto de un parser distinto del que verifica la
       firma, que es lo que la etapa 8 hace al quitarse timestamp de encima.

  Lo que NO vale es actualizar el sha256 de la tabla de procedencia y seguir:
  eso hace callar a la otra puerta sin cerrar esta.`, clave)
				continue
			}
			declarados[clave] = true
		}
	}

	// Suelo, por el mismo motivo que el minimo de puerta.sh: si un dia el
	// recorrido deja de encontrar ficheros, "cero diferencias" se lee igual que
	// "todo en orden".
	if comparados < 20 {
		t.Fatalf("solo se han comparado %d funciones. Con tan pocas, esta puerta estaria "+
			"dando verde sin mirar el codigo que decide", comparados)
	}
	for clave := range recortesDeclarados {
		if !declarados[clave] {
			t.Errorf("recortesDeclarados nombra %q y el recorrido no lo ha visto. O el "+
				"fichero ya no se vendoriza, o el nombre esta mal escrito: en los dos casos "+
				"esa excepcion no protege nada y tapa lo que venga detras", clave)
		}
	}
	t.Logf("%d funciones comparadas con aguas arriba, %d recortes declarados y vivos",
		comparados, len(declarados))
}

// recortesDeclarados son las funciones que a proposito NO son iguales a aguas
// arriba, con el motivo. Cada una esta explicada largo en la cabecera de su
// fichero; aqui va la etiqueta corta para que la puerta pueda distinguir un
// recorte pensado de una edicion que nadie anoto.
var recortesDeclarados = map[string]string{
	"verify.go:(*PKCS7).VerifyWithOpts": "recortes 3, 4 y 5: exige CurrentTime, Roots y KeyUsages",
	"verify.go:getSignatureAlgorithm":   "recorte 6: DSA fuera, con error accionable (portado del 1390b412643f)",
	"verify.go:parseSignedData":         "recorte 7: expone el eContentType, que aguas arriba lee y tira",
	"pkcs7.go:Parse":                    "solo acepta SignedData: lo demas es contenido cifrado y aqui no se descifra nada",
}

// funcionesDe devuelve, por nombre, la forma canonica de cada funcion del
// fichero: reimpresa SIN COMENTARIOS, porque una cabecera de procedencia o un
// `#nosec` no cambian lo que el parser hace.
func funcionesDe(t *testing.T, ruta string) map[string]string {
	t.Helper()
	fset := token.NewFileSet()
	arbol, err := parser.ParseFile(fset, ruta, nil, 0) // 0 = sin comentarios
	if err != nil {
		t.Fatalf("no puedo parsear %s: %v", ruta, err)
	}
	out := map[string]string{}
	for _, d := range arbol.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		var buf bytes.Buffer
		if err := printer.Fprint(&buf, fset, fn); err != nil {
			t.Fatalf("no puedo reimprimir %s de %s: %v", fn.Name.Name, ruta, err)
		}
		// EL RECEPTOR VA EN LA CLAVE, y no es cosmetico: `Parse` la funcion y
		// `(rawCertificates) Parse` el metodo se llaman igual. Con la clave a
		// secas, uno pisaba al otro en el mapa y la comparacion se hacia contra
		// el que quedara, que es un emparejamiento por accidente y no por
		// identidad. Es el invariante 7 mordiendo dentro del test que existe
		// para vigilar el invariante 7.
		nombre := fn.Name.Name
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			var rb bytes.Buffer
			if err := printer.Fprint(&rb, fset, fn.Recv.List[0].Type); err != nil {
				t.Fatalf("no puedo reimprimir el receptor de %s: %v", nombre, err)
			}
			nombre = "(" + rb.String() + ")." + nombre
		}
		if _, repetida := out[nombre]; repetida {
			t.Fatalf("%s declara %s dos veces: la comparacion se haria contra una sola "+
				"de las dos y no se sabria cual", ruta, nombre)
		}
		out[nombre] = buf.String()
	}
	return out
}

// dirDeAguasArriba localiza el modulo original en la cache. Si no esta, esto
// FALLA, no se salta: un test que se salta en silencio es un verde que no ha
// mirado nada, y este vigila justo la clase de cosa que nadie mira.
func dirDeAguasArriba(t *testing.T) string {
	t.Helper()
	salida, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", rutaDeAguasArriba).Output()
	if err != nil {
		t.Fatalf(`no puedo localizar %s en la cache de modulos: %v

  Sin el original no se puede comprobar que la copia siga siendo el mismo
  codigo, y esa es la propiedad de la que depende que el doble parseo de
  adaptadores/tsa no tenga diferencial.
  Arreglo: go mod download %s`, rutaDeAguasArriba, err, rutaDeAguasArriba)
	}
	dir := strings.TrimSpace(string(salida))
	if dir == "" {
		t.Fatalf("go list no ha dicho donde vive %s. Arreglo: go mod download %s",
			rutaDeAguasArriba, rutaDeAguasArriba)
	}
	return dir
}
