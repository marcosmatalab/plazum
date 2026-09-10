package plazum

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// EL IDIOMA DEL TEXTO NO PUEDE TOCAR EL RELOJ (D-25).
//
// # Que afirma, y por que sobre el TIPO y no sobre los casos
//
// Un mismo articulo tiene un solo plazo. Si el modelo permitiera que la version
// inglesa llevara una temporalidad distinta de la castellana, dos clientes
// leyendo la misma norma en dos idiomas verian dos fechas, y el reloj legal es
// el producto.
//
// Se escribe ANTES de la primera carga y va sobre la FORMA, por el mismo motivo
// que la puerta del invariante 13: hoy hay una sola version linguistica en el
// arbol, y una puerta escrita sobre lo que hay nace vigilando casi nada. Sobre
// el tipo, vigila tambien las que escriba alguien dentro de un ano.
//
// # POR QUE NO BASTA CON QUE EL TIPO DE GO NO TENGA EL CAMPO
//
// Porque `nucleo/corpus` NO usa DisallowUnknownFields. Medido el 10-09-2026: en
// todo el arbol solo lo usan `adaptadores/recoleccion/manual` y `evals/arnes`.
// Asi que un "temporalidad" escrito dentro de una version linguistica de un
// paquete.json NO rompe nada: se ignora EN SILENCIO. El paquete carga, el linter
// calla, y quien lo escribio cree que hizo algo, que es peor que un error.
//
// Por eso esta puerta lee el JSON CRUDO y no la estructura ya parseada: lo que
// busca es justo lo que el parser tira.
//
// # EL VOCABULARIO SE DERIVA DEL TIPO, no se copia
//
// Los nombres de los campos que afectan al reloj salen por reflexion de
// `corpus.Obligacion` y de `corpus.Temporalidad`. Copiarlos a mano aqui seria
// una segunda lista, y el dia que el motor gane una primitiva con un campo nuevo
// esta puerta seguiria vigilando el vocabulario viejo sin decirlo.
func TestNingunaVersionLinguisticaPuedeLlevarUnReloj(t *testing.T) {
	prohibidos := camposQueTocanElReloj()
	if len(prohibidos) < 5 {
		t.Fatalf("el vocabulario de reloj derivado del tipo trae %d campos (%v) y son "+
			"pocos para que esta puerta signifique algo. Si `corpus.Obligacion` o "+
			"`corpus.Temporalidad` cambiaron de forma, este derivador se quedo viejo y hay "+
			"que arreglarlo, no bajar el minimo", len(prohibidos), prohibidos)
	}
	t.Logf("vocabulario de reloj derivado del tipo (%d): %v", len(prohibidos), prohibidos)

	ficheros, err := filepath.Glob(filepath.Join("paquetes", "*", "paquete.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(ficheros) < 15 {
		t.Fatalf("solo %d paquetes: el arnes no ha encontrado el corpus", len(ficheros))
	}

	revisadas, conVersion := 0, 0
	for _, f := range ficheros {
		crudo, err := os.ReadFile(f) // #nosec G304 -- ruta derivada de un glob del propio repositorio
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			Obligaciones []struct {
				ID       string                     `json:"id"`
				Versions map[string]json.RawMessage `json:"versiones_linguisticas"`
			} `json:"obligaciones"`
		}
		if err := json.Unmarshal(crudo, &doc); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for _, o := range doc.Obligaciones {
			for lengua, cruda := range o.Versions {
				conVersion++
				var campos map[string]json.RawMessage
				if err := json.Unmarshal(cruda, &campos); err != nil {
					t.Errorf("%s: la version %q de %s no es un objeto: %v",
						f, lengua, o.ID, err)
					continue
				}
				for campo := range campos {
					revisadas++
					if !prohibidos[campo] {
						continue
					}
					t.Errorf(`%s: la version %q de %s declara %q.

  El idioma del texto NO puede tocar el reloj (D-25). Un mismo articulo tiene un
  solo plazo, y dos clientes leyendo la misma norma en dos idiomas tienen que ver
  la misma fecha.

  Y esto NO se cae solo: nucleo/corpus no usa DisallowUnknownFields, asi que ese
  campo se ignora en silencio. El paquete cargaria, el linter callaria, y quien lo
  escribio creeria que puso un reloj.

  Arreglo: el reloj va en la obligacion, una vez, fuera de versiones_linguisticas.`,
						f, lengua, o.ID, campo)
				}
			}
		}
	}

	// EL CONTROL POSITIVO NO PUEDE SER «hay versiones»: hoy puede no haberlas.
	// Lo que si se afirma es que el detector FUNCIONA, y eso lo demuestra
	// TestElDetectorDeRelojesEnVersionesLinguisticasAcusaYSeCalla, que le pone
	// delante las dos formas. Aqui se cuenta lo que se recorrio, para que un dia
	// en que el glob deje de encontrar paquetes no se lea como verde.
	t.Logf("%d version(es) linguistica(s) en el corpus, %d campo(s) revisado(s)",
		conVersion, revisadas)
}

// EL CONTROL NEGATIVO, EN LAS DOS DIRECCIONES.
//
// Sin esto, la puerta de arriba pasaria con un detector que no acusara nunca, que
// es exactamente lo que parece cuando el corpus todavia no tiene versiones. Aqui
// se le ponen delante las dos formas y se exige que distinga.
func TestElDetectorDeRelojesEnVersionesLinguisticasAcusaYSeCalla(t *testing.T) {
	prohibidos := camposQueTocanElReloj()

	// ACUSA: los campos con los que se escribe un reloj.
	for _, campo := range []string{"temporalidad", "escalado", "vigencia",
		"primitiva", "cadencia", "limite"} {
		if !prohibidos[campo] {
			t.Errorf("el detector NO acusa %q y es un campo de reloj. Una version "+
				"linguistica con ese campo pasaria, y con ella la divergencia de fechas "+
				"que D-25 impide", campo)
		}
	}

	// SE CALLA: los campos legitimos de una version linguistica. Si acusara a
	// uno de estos, la puerta seria imposible de satisfacer y la reaccion barata
	// seria aflojarla.
	for _, campo := range []string{"texto", "enlace", "celex", "consultado"} {
		if prohibidos[campo] {
			t.Errorf("el detector acusa %q, que es un campo legitimo de una version "+
				"linguistica. Una puerta que no se puede satisfacer se afloja", campo)
		}
	}

	// Y LOS CUATRO LEGITIMOS SON EXACTAMENTE LOS DEL TIPO, derivados y no
	// escritos: si `VersionLinguistica` gana un campo, esta lista se queda vieja
	// y hay que verlo.
	delTipo := camposJSONDe(reflect.TypeOf(corpus.VersionLinguistica{}))
	sort.Strings(delTipo)
	quiero := []string{"celex", "consultado", "enlace", "texto"}
	if !reflect.DeepEqual(delTipo, quiero) {
		t.Errorf("corpus.VersionLinguistica tiene los campos %v y se esperaban %v.\n"+
			"  Si ha ganado uno, di si toca el reloj: el tipo es la primera mitad de la\n"+
			"  guarda de D-25 y el que decide que NO haya donde escribir una fecha.",
			delTipo, quiero)
	}
}

// camposQueTocanElReloj deriva del TIPO los nombres JSON con los que se escribe
// un reloj: los de `Temporalidad` entera, mas los de `Obligacion` que fijan
// cuando algo vence o a quien escala.
func camposQueTocanElReloj() map[string]bool {
	out := map[string]bool{}
	for _, c := range camposJSONDe(reflect.TypeOf(corpus.Temporalidad{})) {
		out[c] = true
	}
	// De la obligacion, los tres que deciden el reloj. Se nombran porque de esa
	// estructura NO todo es reloj (el titulo o la cita no lo son), y el criterio
	// es el que dice D-25: lo que puede hacer que dos idiomas den dos fechas.
	for _, c := range []string{"temporalidad", "escalado", "vigencia"} {
		out[c] = true
	}
	return out
}

// camposJSONDe saca los nombres de etiqueta json de una estructura.
func camposJSONDe(t reflect.Type) []string {
	var out []string
	for i := 0; i < t.NumField(); i++ {
		etiqueta := t.Field(i).Tag.Get("json")
		if etiqueta == "" || etiqueta == "-" {
			continue
		}
		nombre, _, _ := strings.Cut(etiqueta, ",")
		if nombre != "" {
			out = append(out, nombre)
		}
	}
	return out
}
