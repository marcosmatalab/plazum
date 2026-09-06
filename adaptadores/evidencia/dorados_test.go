package evidencia_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/busqueda"
	"github.com/marcosmatalab/plazum/adaptadores/evidencia"
	"github.com/marcosmatalab/plazum/adaptadores/ia"
	"github.com/marcosmatalab/plazum/adaptadores/ingesta"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// EL CONJUNTO DORADO DE LA PRECISION, Y POR QUE HIZO FALTA.
//
// # De donde sale: una mutacion que sobrevivio
//
// M23 (06-09-2026) quito el filtro de terminos discriminantes de `Mapear` y LA
// BANDA DE COBERTURA SE QUEDO VERDE. Medido: el numero no sube, BAJA, de 64 a 29
// hallazgos, y 29 sigue dentro de la banda declarada.
//
// Lo que eso demuestra es exacto y hay que decirlo asi: **el filtro no cambia
// CUANTO se empareja, cambia QUE se empareja.** Cambia la precision. Y la
// precision no la mide un contaje, asi que la banda no podia cazar esa mutacion
// ni podra nunca. La afirmacion que yo habia escrito al ponerla —que cerraba el
// camino barato de aflojar el umbral— era cierta para una mitad (M24, quitar el
// minimo, si la caza) y falsa para la otra.
//
// Es la leccion del bloque anterior en su propia casa: **una puerta que vigila
// una frontera no vigila lo que pasa al otro lado**, y aqui el otro lado era el
// unico que importaba.
//
// # Por que en datos y no en Go
//
// Porque el linter de normas cableadas no deja escribir un identificador de
// obligacion en codigo, y tiene razon: un conjunto dorado es DATO, igual que una
// obligacion. Se anaden casos sin tocar Go, que es el invariante 2 aplicado a
// los evals, igual que en `evals/citas/dorados.json`.
//
// # Los casos son de dato real, no inventados
//
// Los tres negativos son LOS TRES FALSOS POSITIVOS QUE DE VERDAD OCURRIERON con
// el umbral de dos aciertos: el marcado de contenido sintetico del AI Act, las
// instrucciones al usuario del CRA y el contenido de la notificacion del art.
// 33.3 del RGPD, los tres emparejados con el mismo parrafo de registros de
// acceso. No son mutaciones que yo le puse delante: son lo que hacia el codigo.

type conjuntoDorado struct {
	Nombre    string       `json:"nombre"`
	Porque    string       `json:"porque"`
	Documento string       `json:"documento"`
	Casos     []casoDorado `json:"casos"`
}

type casoDorado struct {
	Obligacion string `json:"obligacion"`
	Veredicto  string `json:"veredicto"`
	Contiene   string `json:"contiene"`
	Porque     string `json:"porque"`
}

func TestLaPrecisionDelMapeoContraSuConjuntoDorado(t *testing.T) {
	b, err := os.ReadFile("testdata/dorados.json")
	if err != nil {
		t.Fatal(err)
	}
	var c conjuntoDorado
	if err := json.Unmarshal(b, &c); err != nil {
		t.Fatal(err)
	}
	if len(c.Casos) == 0 {
		t.Fatal("conjunto dorado vacio: no vigila nada")
	}

	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatal(err)
	}
	textos := map[string]string{}
	for _, p := range ps {
		for _, o := range p.Obligaciones {
			textos[o.ID] = o.TextoLegal
		}
	}

	datos := []byte(c.Documento)
	doc, err := ingesta.Leer("politica.txt", datos)
	if err != nil {
		t.Fatal(err)
	}
	fs, err := ia.FuentesAportadas(ia.Huella(datos), doc)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := busqueda.Nuevo(ia.Documentos(fs))
	if err != nil {
		t.Fatal(err)
	}
	v, err := ia.Nuevo(ia.Opciones{
		Fuentes:    fs,
		Admite:     []ia.Procedencia{ia.Aportado},
		MinimoCita: ia.MinimoCitaPorDefecto,
	})
	if err != nil {
		t.Fatal(err)
	}

	positivos, negativos := 0, 0
	for _, caso := range c.Casos {
		texto, hay := textos[caso.Obligacion]
		if !hay {
			// UN CASO QUE APUNTA A UNA OBLIGACION QUE YA NO EXISTE NO SE SALTA.
			// Saltarlo dejaria el conjunto encogiendo en silencio hasta no
			// vigilar nada, que es como muere un conjunto dorado.
			t.Errorf("el caso %q apunta a una obligacion que el corpus ya no tiene. "+
				"Arreglo: o el identificador cambio y hay que actualizarlo, o la "+
				"obligacion se fue y el caso sobra. Lo que no vale es dejarlo.",
				caso.Obligacion)
			continue
		}

		hs, err := evidencia.Mapear(idx, v, []evidencia.Consulta{
			{ID: caso.Obligacion, Texto: texto},
		})
		if err != nil {
			t.Fatalf("%s: %v", caso.Obligacion, err)
		}

		switch caso.Veredicto {
		case "senala":
			positivos++
			if len(hs) != 1 {
				t.Errorf("%s: no se senala nada, y el conjunto dice que si.\n  porque: %s",
					caso.Obligacion, caso.Porque)
				continue
			}
			if !strings.Contains(hs[0].Cita, caso.Contiene) {
				t.Errorf("%s: se senala otro parrafo.\n  esperaba que contuviera: %q\n"+
					"  y trae: %q\n  porque: %s",
					caso.Obligacion, caso.Contiene, hs[0].Cita, caso.Porque)
			}
		case "no_senala":
			negativos++
			if len(hs) != 0 {
				t.Errorf("%s: se senala un parrafo y el conjunto dice que NO.\n"+
					"  trae: %q\n  porque: %s\n"+
					"  Si esto acaba de ponerse rojo, mira si alguien ha aflojado "+
					"evidencia.MinimoAciertos o el filtro de terminos discriminantes.",
					caso.Obligacion, hs[0].Cita, caso.Porque)
			}
		default:
			t.Errorf("%s: veredicto %q, que no es ni senala ni no_senala",
				caso.Obligacion, caso.Veredicto)
		}
	}

	// LAS DOS DIRECCIONES TIENEN QUE ESTAR POBLADAS. Un conjunto que solo tuviera
	// negativos lo aprobaria un Mapear que no devuelve nada nunca, y uno que solo
	// tuviera positivos lo aprobaria el de dos aciertos que empareja con todo.
	if positivos == 0 || negativos == 0 {
		t.Fatalf("el conjunto tiene %d positivos y %d negativos: hacen falta los dos, "+
			"porque cada mitad sola la aprueba un emparejador degenerado distinto",
			positivos, negativos)
	}
	t.Logf("conjunto dorado de precision: %d casos, %d positivos y %d negativos",
		len(c.Casos), positivos, negativos)
}
