// Comando cotejapkcs7: coteja la copia vendorizada de pkcs7 con aguas arriba,
// funcion a funcion.
//
// POR QUE ES UNA HERRAMIENTA Y NO UN TEST, y el cambio tiene fecha: hasta el
// 26-08-2026 esto vivia dentro de `go test`, y localizaba el original con
// `go list -m github.com/digitorus/pkcs7`. Ese dia el modulo salio de `go.mod`
// entero (con el se fue `timestamp`, que era quien lo arrastraba), asi que ya
// no hay cache de modulos donde mirar. El test se quedo sin poder ejecutarse.
//
// La respuesta correcta no era devolver la dependencia para poder vigilarla:
// eso es exactamente la guarda que llega a lo que vigila por un camino que el
// producto ya no usa. La respuesta es que esto **no es una puerta de PR, es
// vigilancia**: pregunta por algo de FUERA que cambia solo. Su sitio es el
// canario mensual de `.github/workflows/vigilancia.yml`, junto a la comparacion
// de commits, y no el `go test` de cada cambio.
//
// QUE COMPRUEBA. Que la copia vendorizada sigue siendo un SUBCONJUNTO FIEL del
// original. Vendorizar significa heredar el deber de portar los arreglos ajenos
// a mano, y ese deber solo es manejable si la unica distancia con el original
// son recortes DECLARADOS. Sin esto, una edicion que nadie anote convierte la
// copia en un fork silencioso, y entonces el triaje de aguas arriba deja de
// significar nada porque ya no se sabe contra que se compara.
//
// Compara el CODIGO y no los bytes: la copia lleva cabecera de procedencia y
// comentarios `#nosec` que gosec exige, y esos no cambian lo que el parser
// hace. Cada funcion se reimprime con go/printer sin comentarios.
//
// Uso:
//
//	cotejapkcs7 -nuestro adaptadores/tsa/internal/pkcs7 -suyo /tmp/pkcs7
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// ficherosCotejados son los que se vendorizaron. Se nombran uno a uno y no se
// recorre el directorio: si manana se vendoriza un quinto fichero, esta lista
// tiene que cambiar EN EL MISMO COMMIT, y eso obliga a pensar en el.
var ficherosCotejados = []string{"ber.go", "pkcs7.go", "verify.go", "sign.go"}

// RecortesDeclarados son las funciones que a proposito NO son iguales a aguas
// arriba, con el motivo. Cada una esta explicada largo en la cabecera de su
// fichero; aqui va la etiqueta corta para poder distinguir un recorte pensado
// de una edicion que nadie anoto.
var RecortesDeclarados = map[string]string{
	"verify.go:(*PKCS7).VerifyWithOpts": "recortes 3, 4 y 5: exige CurrentTime, Roots y KeyUsages",
	"verify.go:getSignatureAlgorithm":   "recorte 6: DSA fuera, con error accionable (portado del 1390b412643f)",
	"verify.go:parseSignedData":         "recorte 7: expone el eContentType, que aguas arriba lee y tira",
	"pkcs7.go:Parse":                    "solo acepta SignedData: lo demas es contenido cifrado y aqui no se descifra nada",
}

// MinimoDeFunciones es el suelo del cotejo. Sin el, un recorrido que deje de
// encontrar ficheros diria "ninguna diferencia" y se leeria igual que "todo en
// orden": la misma trampa que el canario del cribador de marcas.
const MinimoDeFunciones = 20

func main() {
	nuestro := flag.String("nuestro", "adaptadores/tsa/internal/pkcs7", "directorio de la copia vendorizada")
	suyo := flag.String("suyo", "", "directorio del original de aguas arriba")
	flag.Parse()
	os.Exit(cotejar(*nuestro, *suyo, os.Stdout))
}

// Hallazgo es una diferencia que hay que mirar.
type Hallazgo struct {
	Clave  string
	Motivo string
}

// cotejar devuelve 0 si todo cuadra, 1 si hay diferencias que mirar y 2 si no
// ha podido cotejar. Los tres se distinguen a proposito: "no he podido mirar"
// no es "no hay nada".
func cotejar(nuestro, suyo string, w io.Writer) int {
	if suyo == "" {
		fmt.Fprintln(w, "falta -suyo: el directorio del original de aguas arriba.")
		fmt.Fprintln(w, "  Se clona con: git clone --depth 1 https://github.com/digitorus/pkcs7")
		return 2
	}
	var hallazgos []Hallazgo
	vistosRecortes := map[string]bool{}
	comparadas := 0

	for _, fichero := range ficherosCotejados {
		nuestras, err := funcionesDe(filepath.Join(nuestro, fichero))
		if err != nil {
			fmt.Fprintf(w, "no puedo leer la copia vendorizada de %s: %v\n", fichero, err)
			return 2
		}
		suyas, err := funcionesDe(filepath.Join(suyo, fichero))
		if err != nil {
			fmt.Fprintf(w, "no puedo leer el original de %s: %v\n", fichero, err)
			return 2
		}
		for _, nombre := range ordenadas(nuestras) {
			clave := fichero + ":" + nombre
			suya, existe := suyas[nombre]
			if !existe {
				hallazgos = append(hallazgos, Hallazgo{clave,
					"existe en la copia y NO en aguas arriba. Una copia vendorizada no inventa " +
						"funciones: o viene de otro fichero que no se vendorizo (dilo en LEEME.md) " +
						"o es codigo propio infiltrado en territorio ajeno"})
				continue
			}
			comparadas++
			motivo, esRecorte := RecortesDeclarados[clave]
			if nuestras[nombre] == suya {
				if esRecorte {
					vistosRecortes[clave] = true
					hallazgos = append(hallazgos, Hallazgo{clave,
						"esta declarado como recorte (" + motivo + ") y ya es IDENTICO a aguas " +
							"arriba. Quitalo de RecortesDeclarados: una excepcion que no excepciona " +
							"nada deja pasar el dia que vuelva a diferir"})
				}
				continue
			}
			if !esRecorte {
				hallazgos = append(hallazgos, Hallazgo{clave,
					"YA NO ES EL MISMO CODIGO que aguas arriba y no esta declarado como recorte. " +
						"Portalo a mano (LEEME.md dice como) o declaralo en RecortesDeclarados con " +
						"su motivo. Lo que NO vale es actualizar el sha256 de la tabla de " +
						"procedencia y seguir: eso hace callar a la otra puerta sin cerrar esta"})
				continue
			}
			vistosRecortes[clave] = true
		}
	}
	for clave := range RecortesDeclarados {
		if !vistosRecortes[clave] {
			hallazgos = append(hallazgos, Hallazgo{clave,
				"RecortesDeclarados lo nombra y el recorrido no lo ha visto. O el fichero ya no " +
					"se vendoriza, o el nombre esta mal escrito: en los dos casos esa excepcion " +
					"no protege nada y tapa lo que venga detras"})
		}
	}

	if comparadas < MinimoDeFunciones {
		fmt.Fprintf(w, "PUERTA ROTA: solo se han comparado %d funciones y el minimo es %d.\n",
			comparadas, MinimoDeFunciones)
		fmt.Fprintln(w, "  Con tan pocas, este cotejo estaria dando verde sin mirar el codigo")
		fmt.Fprintln(w, "  que decide. Comprueba que -nuestro y -suyo apuntan a donde crees.")
		return 2
	}
	sort.Slice(hallazgos, func(i, j int) bool { return hallazgos[i].Clave < hallazgos[j].Clave })
	for _, h := range hallazgos {
		fmt.Fprintf(w, "DIFERENCIA  %s\n   %s\n", h.Clave, h.Motivo)
	}
	fmt.Fprintf(w, "%d funciones cotejadas, %d recortes declarados, %d diferencias que mirar\n",
		comparadas, len(RecortesDeclarados), len(hallazgos))
	if len(hallazgos) > 0 {
		return 1
	}
	return 0
}

// funcionesDe devuelve, por nombre, la forma canonica de cada funcion: sin
// comentarios, porque un `#nosec` o una cabecera de procedencia no cambian lo
// que el codigo hace.
//
// EL RECEPTOR VA EN LA CLAVE, y no es cosmetico: `Parse` la funcion y
// `(rawCertificates) Parse` el metodo se llaman igual. Con la clave a secas uno
// pisaba al otro en el mapa y la comparacion se hacia contra el que quedara, o
// sea por accidente y no por identidad. Es el invariante 7 mordiendo dentro de
// la herramienta escrita para vigilar el invariante 7.
func funcionesDe(ruta string) (map[string]string, error) {
	fset := token.NewFileSet()
	arbol, err := parser.ParseFile(fset, ruta, nil, 0) // 0 = sin comentarios
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, d := range arbol.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		var buf bytes.Buffer
		if err := printer.Fprint(&buf, fset, fn); err != nil {
			return nil, err
		}
		nombre := fn.Name.Name
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			var rb bytes.Buffer
			if err := printer.Fprint(&rb, fset, fn.Recv.List[0].Type); err != nil {
				return nil, err
			}
			nombre = "(" + rb.String() + ")." + nombre
		}
		if _, repetida := out[nombre]; repetida {
			return nil, fmt.Errorf("%s declara %s dos veces: la comparacion se haria contra "+
				"una sola de las dos y no se sabria cual", ruta, nombre)
		}
		out[nombre] = buf.String()
	}
	return out, nil
}

func ordenadas(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
