package plazum

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/evidencia"
	"github.com/marcosmatalab/plazum/adaptadores/ia"
	"github.com/marcosmatalab/plazum/adaptadores/ingesta"
	"github.com/marcosmatalab/plazum/superficies/documentos"
)

// EL INVARIANTE 13 EN LA FRONTERA DE ENTRADA: un documento del cliente no trae
// veredictos.
//
// # Que hay de verdad en esta frontera, dicho antes de vigilarla
//
// Lo que entra por aqui es un FICHERO DE TEXTO: un PDF o un .txt del que se
// extraen parrafos. Un texto no tiene campos, asi que no hay ningun «campo con
// forma de juicio» que un fichero subido pueda traer y que este arbol pueda leer
// por descuido. Decirlo importa, porque la puerta que se escribiera sobre eso
// naceria vigilando el vacio, que es el verde mas facil de conseguir.
//
// La frontera de verdad es la MISMA que resuelve el invariante 13 para los
// recolectores, y por la misma razon: EL TIPO. Todo lo que la cadena de la pieza
// 3 puede llegar a decir de un documento del cliente cabe en estas siete
// estructuras y en ninguna otra. Si ninguna de ellas sabe transportar un juicio,
// no hay forma de que un juicio llegue a la pantalla, la escriba quien la
// escriba.
//
// # Los dos regimenes, y por que no es el mismo para toda la cadena
//
// LO QUE CRUZA A LA PANTALLA no puede traer ni un campo con forma de juicio, sin
// excepciones y sin lista de excusas. Son `documentos.Hallazgo` y
// `documentos.ResumenDeDocumento`: lo que una persona lee. Una lista de
// obligaciones con un numero al lado se lee como una nota, aunque el numero sea
// de otra cosa.
//
// LO QUE VIVE DENTRO DE LA CADENA puede llevar una medida mecanica, y entonces
// la lleva CON SU LINEA ESCRITA. `evidencia.Hallazgo.Puntuacion` es el ejemplo y
// es el que hace util a esta puerta: es la relevancia BM25, sirve para ORDENAR
// dentro del adaptador, y NO existe en el tipo de la pantalla. Esa ausencia es
// una decision y hasta hoy no la sostenia nada: quitar `Puntuacion` de
// `documentos.Hallazgo` es un descuido de una linea, y lo que volveria seria un
// porcentaje al lado de una norma.
//
// # Lo que esta puerta NO mira, y quien mira ahi
//
// El TEXTO de la pantalla. Un hallazgo puede no tener ni un campo de veredicto y
// la plantilla escribir «cumples» al lado: eso lo vigila
// TestLaPantallaDeHallazgosNoDiceQueCumplasNada, sobre el HTML servido. Son las
// dos mitades y ninguna sirve sola.

// tiposQueCruzanALaPantalla son los que una persona acaba leyendo. Regimen
// estricto: ni un campo con forma de juicio, y NO hay lista de excusas para
// ellos a proposito.
var tiposQueCruzanALaPantalla = []any{
	documentos.Hallazgo{},
	documentos.ResumenDeDocumento{},
}

// tiposDeLaCadena son los eslabones internos, del fichero al hallazgo.
var tiposDeLaCadena = []any{
	ingesta.Documento{},
	ingesta.Fragmento{},
	ia.Fuente{},
	evidencia.Consulta{},
	evidencia.Hallazgo{},
}

// medidasJustificadas son los campos de la CADENA que suenan a juicio y no lo
// son, cada uno con su linea. Misma forma que `camposJustificados` de
// nucleo/estado, y por el mismo motivo: el gesto que se fuerza es escribir el
// porque, no aprobar el campo.
var medidasJustificadas = map[string]string{
	"evidencia.Hallazgo.Puntuacion": "es la relevancia BM25 del fragmento contra la " +
		"consulta, o sea cuanto se parecen dos textos: una medida mecanica del buscador " +
		"y no una afirmacion sobre el cliente. Sirve para ORDENAR y para cortar por el " +
		"minimo, y NO cruza a la pantalla, que es la mitad que de verdad la hace " +
		"inocua: un numero al lado de una obligacion se lee como una nota, diga lo que " +
		"diga su godoc.",
}

// TestNingunDocumentoDelClienteLlegaConVeredicto es la puerta.
func TestNingunDocumentoDelClienteLlegaConVeredicto(t *testing.T) {
	vocabulario := vocabularioDeVeredictoDelNucleo(t)

	// REGIMEN 1: lo que cruza a la pantalla, sin excusas.
	mirados := 0
	for _, v := range tiposQueCruzanALaPantalla {
		tipo := reflect.TypeOf(v)
		if tipo.NumField() == 0 {
			t.Fatalf("%s no tiene campos: el detector esta mirando el vacio", tipo)
		}
		for i := 0; i < tipo.NumField(); i++ {
			nombre := tipo.Field(i).Name
			mirados++
			if raiz := casaAlguna(nombre, vocabulario); raiz != "" {
				t.Errorf("%s.%s cruza a la pantalla y suena a veredicto (%q).\n"+
					"  Esta pantalla enfrenta obligaciones de una norma con parrafos de la "+
					"politica de una persona: un campo con forma de juicio ahi convierte "+
					"«este parrafo habla de esto» en «esto lo cumples», que es el unico "+
					"error que un producto de cumplimiento no puede cometer ni una vez.\n"+
					"  Y aqui NO hay lista de excusas: si el campo hace falta, vive en la "+
					"cadena y no en el tipo que se pinta.", tipo, nombre, raiz)
			}
		}
	}

	// REGIMEN 2: la cadena, con su linea escrita por cada medida.
	vistos := map[string]bool{}
	for _, v := range tiposDeLaCadena {
		tipo := reflect.TypeOf(v)
		if tipo.NumField() == 0 {
			t.Fatalf("%s no tiene campos: el detector esta mirando el vacio", tipo)
		}
		for i := 0; i < tipo.NumField(); i++ {
			campo := tipo.Field(i)
			if !campo.IsExported() {
				continue // lo privado no sale del paquete y no puede llegar a nadie
			}
			mirados++
			clave := nombreCorto(tipo) + "." + campo.Name
			raiz := casaAlguna(campo.Name, vocabulario)
			motivo, justificado := medidasJustificadas[clave]
			vistos[clave] = true
			switch {
			case raiz != "" && !justificado:
				t.Errorf("%s suena a veredicto (%q) y no tiene linea escrita.\n"+
					"  Invariante 13: lo que sale de leer el documento de alguien son "+
					"HECHOS. El juicio lo hace una persona con el parrafo delante.\n"+
					"  Arreglo: o el campo sale, o entra en medidasJustificadas diciendo "+
					"que MIDE y por que no cruza a la pantalla.", clave, raiz)
			case raiz == "" && justificado:
				// LA DIRECCION CONTRARIA. Una excusa para un campo que ya no
				// suena a veredicto es una excusa huerfana, y una lista de
				// excusas huerfanas es como una lista negra deja de vigilar sin
				// que nadie se entere.
				t.Errorf("medidasJustificadas justifica %s y ese campo ya no suena a "+
					"veredicto. Arreglo: quitar la entrada.\n  decia: %s", clave, motivo)
			}
		}
	}
	// Y toda excusa apunta a un campo que existe.
	for clave := range medidasJustificadas {
		if !vistos[clave] {
			t.Errorf("medidasJustificadas justifica %q, que ya no es un campo exportado "+
				"de ningun tipo de la cadena. Arreglo: quitar la entrada", clave)
		}
	}

	// EL SUELO: si los tipos se vacian o el recorrido deja de verlos, esta
	// puerta seguiria verde recorriendo la nada.
	if mirados < 25 {
		t.Fatalf("solo se han mirado %d campos en las siete estructuras de la cadena, y hoy "+
			"son mas: este recorrido esta midiendo el vacio", mirados)
	}
	t.Logf("%d campos mirados en %d estructuras, %d medida(s) justificada(s)",
		mirados, len(tiposQueCruzanALaPantalla)+len(tiposDeLaCadena), len(medidasJustificadas))
}

// EL CONTROL NEGATIVO, en las dos direcciones y sobre el detector.
//
// Sin esta mitad, un `casaAlguna` que devolviera siempre vacio dejaria la puerta
// de arriba verde sobre cualquier cosa, que es como se consigue el verde mas
// caro de este repositorio.
func TestElDetectorDeVeredictosDeEntradaAcusaYSeCalla(t *testing.T) {
	vocabulario := vocabularioDeVeredictoDelNucleo(t)

	// ACUSA: nombres con los que se escribe un juicio.
	for _, n := range []string{"Cumple", "Cumplimiento", "Conforme", "Veredicto",
		"Aprobado", "Puntuacion", "Nota", "Satisfecho", "Severidad", "Criticidad"} {
		if casaAlguna(n, vocabulario) == "" {
			t.Errorf("el detector NO acusa %q, que es un juicio escrito con todas sus "+
				"letras. Con este hueco, la puerta de arriba deja pasar justo lo que "+
				"persigue", n)
		}
	}
	// SE CALLA: nombres con los que se escribe un hecho.
	for _, n := range []string{"Pagina", "Orden", "Texto", "Fragmentos", "Huella",
		"Fichero", "Parrafo", "Documento", "Obligacion", "Marco", "Truncado"} {
		if raiz := casaAlguna(n, vocabulario); raiz != "" {
			t.Errorf("el detector acusa %q por la raiz %q, y eso es un HECHO.\n"+
				"  Una puerta que acusa en falso se acaba aflojando, y entonces no "+
				"vigila nada", n, raiz)
		}
	}
}

// vocabularioDeVeredictoDelNucleo lee la lista de raices DEL FUENTE de
// nucleo/estado, y no la copia.
//
// # Por que se lee y no se escribe otra vez aqui
//
// Porque dos listas de lo mismo son dos listas que se separan, y la que manda
// acaba siendo la otra. Es el mismo error que este repositorio ya se ha comido
// con los cardinales de sus documentos: un conjunto de copias de acuerdo entre
// si es indistinguible de un conjunto correcto, y solo el original rompe el
// empate. `vocabularioDeVeredicto` es privado de `nucleo/estado` y vive en un
// `_test.go`, asi que no se puede importar: se lee su literal del AST, que es lo
// mismo que hacen otras puertas de este arbol con los ficheros de CI.
func vocabularioDeVeredictoDelNucleo(t *testing.T) []string {
	t.Helper()
	const fuente = "nucleo/estado/veredicto_test.go"
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, fuente, nil, 0)
	if err != nil {
		t.Fatalf("no he podido leer %s (%v).\n"+
			"  Esto NO es «no hay veredictos»: es que la lista no se ha podido leer, y "+
			"las dos cosas se parecen solo si no se miran", fuente, err)
	}
	var raices []string
	ast.Inspect(f, func(n ast.Node) bool {
		vs, ok := n.(*ast.ValueSpec)
		if !ok || len(vs.Names) != 1 || vs.Names[0].Name != "vocabularioDeVeredicto" {
			return true
		}
		for _, v := range vs.Values {
			lit, ok := v.(*ast.CompositeLit)
			if !ok {
				continue
			}
			for _, e := range lit.Elts {
				b, ok := e.(*ast.BasicLit)
				if !ok || b.Kind != token.STRING {
					continue
				}
				raices = append(raices, strings.Trim(b.Value, `"`))
			}
		}
		return false
	})
	sort.Strings(raices)
	// EL SUELO DE LA LECTURA. Si el literal se renombra o cambia de forma, esto
	// devolveria una lista corta y la puerta de arriba saldria verde sin haber
	// comprobado nada. Hoy son 16 raices.
	if len(raices) < 14 {
		t.Fatalf("he leido %d raices de %s (%v) y son 16: el literal ha cambiado de "+
			"forma y esta lectura ha dejado de encontrarlo", len(raices), fuente, raices)
	}
	return raices
}

// casaAlguna devuelve la raiz que ha casado, o vacio.
func casaAlguna(nombre string, raices []string) string {
	n := strings.ToLower(nombre)
	for _, r := range raices {
		if strings.Contains(n, r) {
			return r
		}
	}
	return ""
}

// nombreCorto es «paquete.Tipo», que es como se escriben las claves de
// medidasJustificadas.
func nombreCorto(t reflect.Type) string {
	ruta := t.PkgPath()
	if i := strings.LastIndex(ruta, "/"); i >= 0 {
		ruta = ruta[i+1:]
	}
	return ruta + "." + t.Name()
}
