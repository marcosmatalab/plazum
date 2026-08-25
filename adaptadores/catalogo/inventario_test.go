package catalogo

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dutiq/nucleo/pantalla"
	"dutiq/superficies/pantallas"
)

// El inventario del catalogo, comprobado contra quien pide las cadenas.
//
// Por que hace falta un test y no basta con revisarlo. Faltantes(idioma), que
// es lo que da el puerto, compara un idioma contra el de por defecto: responde
// "¿le falta al ingles lo que el espanol tiene?" y no responde la pregunta que
// de verdad rompe la pantalla, que es "¿tiene el catalogo lo que la interfaz
// pide?". Un catalogo completo en los dos idiomas y sin la clave
// "alcance.pregunta.si" da una pagina con botones que ponen
// "alcance.pregunta.si", y Faltantes da verde.
//
// Los dos sentidos importan:
//
//	del codigo al catalogo   una clave que se pide y no esta sale en crudo en
//	                         la pantalla de un cliente
//	del catalogo al codigo   una clave traducida que ya nadie pide es peso
//	                         muerto que hay que traducir a cada idioma nuevo
//
// El inventario NO se escribe aqui a mano. Sale de dos sitios que lo calculan:
// superficies/pantallas.ClavesDeCatalogo(), que es el contrato que publica la
// superficie, y las claves que emite nucleo/pantalla, leidas de su AST.

// clavesPropias son las claves que este catalogo trae y que hoy no pide nadie.
// Cada una tiene que estar justificada aqui, con nombre y apellidos, porque la
// regla general es que el catalogo no lleva nada que no se ensene.
var clavesPropias = map[string]string{
	"aviso.idioma_del_corpus": "Explica por que el texto de las normas sigue en el idioma " +
		"de su fuente cuando la interfaz esta en otro. Es una decision LEGAL (traducir el " +
		"BOE crea obra derivada) que al usuario le parece un fallo de traduccion si nadie " +
		"se la cuenta, asi que la cadena se escribe aqui y queda apuntado en " +
		"docs/pendientes.md que la pantalla tiene que pintarla al lado del texto del corpus.",
}

func TestElCatalogoCubreExactamenteLoQuePideLaInterfaz(t *testing.T) {
	c := nuevoParaTest(t)

	pedidas := map[string]string{}
	for _, k := range pantallas.ClavesDeCatalogo() {
		pedidas[k] = "superficies/pantallas.ClavesDeCatalogo()"
	}
	if len(pedidas) < 50 {
		t.Fatalf("la interfaz declara %d claves y son muchas menos de las que tiene: "+
			"o el contrato se ha vaciado, o este test esta mirando otra cosa", len(pedidas))
	}
	for k, donde := range clavesEnElCodigo(t) {
		if _, ya := pedidas[k]; !ya {
			pedidas[k] = donde
		}
	}

	tiene := map[string]bool{}
	for _, k := range c.Claves() {
		tiene[k] = true
	}

	for clave, donde := range pedidas {
		if !tiene[clave] {
			t.Errorf("%s pide la clave %q y el catalogo no la tiene.\n"+
				"Arreglo: redactarla en adaptadores/catalogo/cadenas/es.json y traducirla "+
				"en en.json. Sin eso sale en crudo en la pantalla", donde, clave)
			continue
		}
		for _, idioma := range c.Idiomas() {
			if c.Traducir(idioma, clave) == clave {
				t.Errorf("la clave %q no tiene cadena en %q", clave, idioma)
			}
		}
	}

	for clave := range tiene {
		if _, pedida := pedidas[clave]; pedida {
			continue
		}
		if porque, propia := clavesPropias[clave]; propia {
			t.Logf("clave propia justificada: %s\n  %s", clave, porque)
			continue
		}
		t.Errorf("el catalogo traduce %q y no la pide nadie.\n"+
			"Arreglo: borrarla de es.json y de en.json, o arreglar el renombrado a medias "+
			"que la dejo huerfana. Si de verdad tiene que estar sin que nadie la pinte "+
			"todavia, va en clavesPropias con el motivo escrito", clave)
	}
}

// Las seis pantallas derivadas, comprobadas contra el derivador de verdad y no
// contra una lista escrita a mano.
func TestLasSeisPantallasSalenRotuladasSinCorpusInstalado(t *testing.T) {
	c := nuevoParaTest(t)
	ps := pantalla.Derivar(nil)
	if len(ps) != 6 {
		t.Fatalf("el derivador da %d pantallas y la etapa 2 tiene 6", len(ps))
	}
	for _, p := range ps {
		for _, clave := range []string{p.Titulo, p.PorQue} {
			if clave == "" {
				continue
			}
			for _, idioma := range c.Idiomas() {
				if got := c.Traducir(idioma, clave); got == clave {
					t.Errorf("la pantalla %q sale con la clave %q en crudo en %q. "+
						"Un comprador que instala esto y no tiene corpus ve seis pantallas "+
						"vacias, y lo unico que le explica que hacer es ese rotulo",
						p.ID, clave, idioma)
				}
			}
		}
	}
}

// arbolesQueEmitenClaves son los directorios cuyo codigo de produccion emite
// claves de catalogo como literal.
//
// superficies/ NO esta aqui, y es a proposito: esa superficie publica su
// inventario calculado en ClavesDeCatalogo(), que es mejor que raspar su codigo
// fuente (buena parte de sus claves se componen, "columna."+nombre, y un
// literal partido no se puede traducir). Una superficie nueva o publica su
// inventario igual, o se anade aqui.
var arbolesQueEmitenClaves = []string{"../../nucleo/pantalla"}

// clavesEnElCodigo devuelve las claves de catalogo que aparecen como literal en
// el codigo de produccion de los arboles vigilados, con el fichero donde salen.
//
// Mira el AST y no el texto porque un literal partido en dos ("pantalla." + x)
// no es una clave que se pueda traducir, y un comentario que mencione una clave
// no es una clave emitida.
func clavesEnElCodigo(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	fset := token.NewFileSet()
	for _, raiz := range arbolesQueEmitenClaves {
		if _, err := os.Stat(raiz); err != nil {
			t.Fatalf("el arbol %s no existe: o se ha movido, o este escaneo lleva tiempo "+
				"sin mirar nada", raiz)
		}
		err := filepath.WalkDir(raiz, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			a, err := parser.ParseFile(fset, p, nil, 0)
			if err != nil {
				return err
			}
			ast.Inspect(a, func(n ast.Node) bool {
				lit, ok := n.(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				v := strings.Trim(lit.Value, "`\"")
				if pareceClave(v) {
					out[v] = filepath.ToSlash(p)
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatalf("recorriendo %s: %v", raiz, err)
		}
	}
	if len(out) == 0 {
		t.Fatal("el escaneo no ha encontrado ni una clave de catalogo en el codigo. " +
			"O los arboles se han movido, o el escaner ha dejado de escanear: en los dos " +
			"casos estaria verde por vacio")
	}
	return out
}

// pareceClave reconoce una clave de catalogo por su forma y por su espacio de
// nombres. Exigir el espacio evita que cualquier cadena con un punto dentro
// ("no.se.que") se convierta en una traduccion pendiente.
func pareceClave(s string) bool { return motivoRechazoClave(s) == "" }
