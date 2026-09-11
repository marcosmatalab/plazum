package plazum

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// EL FRAGMENTO DE UNA OBLIGACION APUNTA AL ARTICULO QUE ESA OBLIGACION DICE.
//
// # El defecto, y por que el linter no podia verlo
//
// `Obligacion.Fragmento` es lo que va detras de la almohadilla en el enlace
// profundo. El linter lo valida por FORMA y eso esta bien y no basta: la forma
// no sabe nada del articulo que la obligacion dice.
//
// Medido sobre el corpus publicado el 11-09-2026, y el reparto es el
// diagnostico entero:
//
//	dialecto del BOE   5 fragmentos de articulo de UN digito, los 5 correctos
//	                  10 fragmentos de articulo de DOS digitos, los 10 MAL
//	dialecto de la UE   1 mas, `aiact.art111_4`, que declara `art_1` para el
//	                  articulo 111
//	                  ----
//	                  11 de 73 obligaciones con fragmento
//
// # LA CAUSA NO ERA UNA ERRATA REPETIDA: LA FORMA DECLARADA ERA IMPOSIBLE
//
// Diez de diez con el mismo patron pedia mirar el documento, y al mirarlo sale
// otra cosa. Verificado el 11-09-2026 contra las paginas ELI que este producto
// enlaza —`boe.es/eli/es/lo/2018/12/05/3/con` y
// `boe.es/eli/es/l/2023/02/20/2/con`— el BOE **no numera sus anclas `a` mas el
// numero del articulo**:
//
//	articulos 1 a 9      a1 ... a9
//	del 10 en adelante   a<primer digito>-<orden en su decena>
//	                     el art. 22 es `a2-4` y el art. 65 es `a6-7`
//	disposiciones        da, da-2, dt, dt-2, df, df-2, dd
//
// O sea que `a22` NO EXISTE en el documento, y la forma que el linter declaraba
// —`^(a|da|dt|df|dd)[0-9]+$`— **era incapaz de expresar el ancla correcta de
// cualquier articulo del 10 en adelante**. Los diez de dos digitos no estaban
// mal por descuido: estaban mal porque lo correcto no se podia escribir. Y las
// disposiciones iban mal en la otra direccion, con la forma pidiendo `da1`
// donde el BOE escribe `da` y `da-2`.
//
// Lo que quedaba escrito, `a6` para el articulo 65, es el PREFIJO del ancla
// buena: lleva al articulo 6, que existe, se abre y no salta. Un fragmento que
// apunta a un articulo que no existe se nota; uno que apunta a OTRO articulo que
// si existe no se nota nunca, porque la pagina carga.
//
// **Y el de la UE ensena la otra mitad**: su cita apunta a
// `eli/reg/2026/1744/oj#art_1`, que es el articulo 1 del omnibus que ANADIO ese
// apartado, y ahi `art_1` seria correcto. Pero el enlace no se deriva de la
// cita: `EnlaceDeLaObligacion` lo compone sobre el `Identificador` DEL PAQUETE,
// que es el AI Act, asi que `art_1` lleva al articulo 1 del AI Act, que es
// «Objeto». El fragmento viaja con el documento del paquete, no con la URL que
// alguien pego en la cita.
//
// # POR QUE ESTA PUERTA VA AQUI Y NO EN EL LINTER, que es la pregunta obvia
//
// Porque el linter valida paquetes de CUALQUIERA, y para contrastar hay que leer
// `Articulo`, que es texto libre a proposito: «anexo, punto 12.2.3», «ritual
// plazum sobre el art. 78.6, segunda frase», «Disposicion adicional segunda». Un
// lector de eso se equivoca en las formas raras, y un linter que rechaza un
// paquete correcto de un tercero es peor que este defecto.
//
// Aqui el terreno es otro: es NUESTRO corpus, lo que no se pueda contrastar se
// cuenta con igualdad exacta, y crecer el hueco cuesta venir a escribir por que.
// Es la misma forma que la puerta de los apartados de las versiones
// linguisticas, y por el mismo motivo.
func TestElFragmentoDeUnaObligacionApuntaAlArticuloQueDice(t *testing.T) {
	fs, err := filepath.Glob(filepath.Join("paquetes", "*", "paquete.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) < MinimoDeMarcos {
		t.Fatalf("solo %d paquetes: el arnes no ha encontrado el corpus", len(fs))
	}

	contrastados, conFragmento, sinFragmento := 0, 0, 0
	var mudos []string
	for _, f := range fs {
		b, err := os.ReadFile(f) // #nosec G304 -- ruta derivada de un glob del propio repositorio
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			Obligaciones []struct {
				ID        string `json:"id"`
				Articulo  string `json:"articulo"`
				Fragmento string `json:"fragmento"`
			} `json:"obligaciones"`
		}
		if err := json.Unmarshal(b, &doc); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for _, o := range doc.Obligaciones {
			fr := strings.TrimSpace(o.Fragmento)
			if fr == "" {
				sinFragmento++
				continue
			}
			conFragmento++
			apunta, ok := articuloDelFragmento(fr)
			dice, ok2 := articuloDelCampo(o.Articulo)
			if !ok || !ok2 {
				mudos = append(mudos, o.ID+" (fragmento "+fr+", articulo "+o.Articulo+")")
				continue
			}
			contrastados++
			if elFragmentoApuntaAl(apunta, dice) {
				continue
			}
			t.Errorf(`%s: el fragmento apunta a OTRO articulo.

  fragmento  %-10s -> articulo %s
  articulo   %-10s -> articulo %s

  Esto es lo que va detras de la almohadilla en el enlace que se le da a un
  cliente. Un fragmento que apunta a un articulo que NO existe se nota; uno que
  apunta a otro que SI existe no se nota nunca, porque la pagina carga y el
  lector cree que esta leyendo su articulo.

  El linter valida la FORMA del fragmento y no la correspondencia, asi que «a2»
  para el articulo 22 le parece impecable.

  Y el fragmento viaja con el documento DEL PAQUETE: EnlaceDeLaObligacion lo
  compone sobre el Identificador del paquete, no sobre la URL que lleve la cita.
  Si el punto lo anadio otra norma, eso se cuenta en la cita y el fragmento
  sigue siendo el del documento que se enlaza.`,
				o.ID, fr, apunta, strings.TrimSpace(o.Articulo), dice)
		}
	}

	if contrastados == 0 {
		t.Fatal("ni un fragmento contrastado contra su articulo: los lectores se han roto y " +
			"esta puerta estaria verde sin mirar nada")
	}

	// EL HUECO VA CON SU CARDINAL Y CON IGUALDAD EXACTA.
	//
	// Se llega aqui cuando uno de los dos lados no se deja leer: las
	// disposiciones del BOE (`da1`, `dt2`, `df3`), cuyo `articulo` se escribe en
	// letra («Disposicion adicional segunda»), y los `articulo` en prosa que no
	// empiezan por un numero. No es un fallo, y no puede crecer en silencio:
	// cada unidad es un enlace que se le da a un cliente y que no comprueba
	// nadie.
	const mudosDeclarados = 0
	if len(mudos) != mudosDeclarados {
		sort.Strings(mudos)
		t.Errorf("hay %d obligacion(es) con fragmento que no se pueden contrastar y se "+
			"declaran %d.\n  Las de ahora: %v\n"+
			"  Si ha SUBIDO, hay enlaces profundos que nadie comprueba. O el articulo de "+
			"verdad no se puede leer (una disposicion adicional) y se sube este numero A "+
			"PROPOSITO diciendo cual, o el campo `articulo` se puede escribir de forma que "+
			"si se lea.\n  Si ha BAJADO, sobra parte de esta cuenta.",
			len(mudos), mudosDeclarados, mudos)
	}
	if !t.Failed() {
		t.Logf("MEDIDO: %d obligacion(es) con fragmento, %d contrastadas contra su articulo, "+
			"%d sin contrastar.\n"+
			"  Y el otro lado, que se cuenta y no se acusa: %d obligacion(es) SIN fragmento. "+
			"El campo es opcional a proposito (hacerlo obligatorio pondria rojas cientas de "+
			"una vez y una puerta asi se afloja), asi que este numero es el trabajo que "+
			"queda, no un defecto.",
			conFragmento, contrastados, len(mudos), sinFragmento)
	}
}

// EL CONTROL NEGATIVO DE LOS DOS LECTORES, EN LAS DOS DIRECCIONES.
//
// Sin esto, la puerta de arriba pasaria con un par de lectores que se rindieran
// siempre (todo al saco de los mudos) o que devolvieran lo mismo para los dos
// lados. Las dos cosas dan verde sobre un corpus sano.
func TestLosLectoresDeFragmentoYArticuloAcusanYSeCallan(t *testing.T) {
	for _, c := range []struct {
		fragmento, quiero string
		hay               bool
		porQue            string
	}{
		{"a6", "6", true, "el ancla del BOE para un articulo de un digito ES el digito, y " +
			"esos cinco eran los unicos que estaban bien"},
		{"a2-4", "2+", true, "el ancla del BOE por decenas: fija la decena y no el articulo, " +
			"y el sufijo + es lo que obliga a comparar con esa precision y no con otra"},
		{"a6-7", "6+", true, "el articulo 65, verificado contra la pagina del BOE el 11-09-2026"},
		{"art_87", "87", true, "el dialecto de la Union, que si pone el numero entero"},
		{"art_19a", "19a", true, "un articulo bis: el sufijo es parte del articulo, no ruido"},
		{"anx_VIII", "anexo VIII", true, "los anexos van en romano y eso lo dijo el corpus"},
		{"a22", "", false, "NO ES UNA FORMA DEL BOE: no existe en el documento, y darla por " +
			"buena es lo que dejo el defecto once veces. Un ancla que no existe deja al " +
			"lector arriba del todo"},
		{"da-2", "", false, "una disposicion adicional no es un articulo y no se contrasta"},
		{"dt", "", false, "ni una transitoria"},
		{"", "", false, "sin fragmento no hay nada que leer"},
		{"basura", "", false, "lo que no tenga forma conocida NO se da por bueno"},
	} {
		hay, ok := articuloDelFragmento(c.fragmento)
		if ok != c.hay || hay != c.quiero {
			t.Errorf("articuloDelFragmento(%q) = (%q, %v) y se esperaba (%q, %v). %s",
				c.fragmento, hay, ok, c.quiero, c.hay, c.porQue)
		}
	}

	for _, c := range []struct {
		articulo, quiero string
		hay              bool
	}{
		{"22.3", "22", true},
		{"111.4", "111", true},
		{"87.1, 87.2, 87.3, 87.4 y 87.5", "87", true},
		{"19 bis, apartado 1, letra b", "19a", true},
		{"24, apartado 3", "24", true},
		{"anexo VIII, punto 3.2", "anexo VIII", true},
		{"ritual plazum sobre el art. 32.1.d", "32", true},
		{"ritual plazum sobre el anexo I, parte II, punto 3", "anexo I", true},
		{"Disposicion adicional segunda", "", false},
		{"", "", false},
	} {
		hay, ok := articuloDelCampo(c.articulo)
		if ok != c.hay || hay != c.quiero {
			t.Errorf("articuloDelCampo(%q) = (%q, %v) y se esperaba (%q, %v)",
				c.articulo, hay, ok, c.quiero, c.hay)
		}
	}

	// Y LA COMPARACION, QUE ES DONDE VIVE EL DEFECTO. Sin estos casos, un
	// comparador que dijera «si» a todo pasaria todo lo de arriba y no acusaria
	// nunca, y uno que dijera «no» a todo tampoco se distinguiria.
	for _, c := range []struct {
		fragmento, articulo string
		apunta              bool
		porQue              string
	}{
		{"6", "6", true, "el caso trivial que tiene que seguir valiendo"},
		{"6+", "65", true, "el ancla por decenas vale para cualquier articulo de su decena"},
		{"6+", "60", true, "y para el primero de la decena"},
		{"6", "65", false, "ES EL DEFECTO: el ancla sin guion es el articulo 6, no el 65. " +
			"Once obligaciones del corpus estaban asi"},
		{"6+", "7", false, "un ancla de la decena del 6 no puede ser del articulo 7"},
		{"6+", "6", false, "ni del articulo 6, que tiene su propia ancla sin guion"},
		{"111", "111", true, "el dialecto de la Union compara exacto"},
		{"1", "111", false, "y por eso ahi el defecto se ve entero: art_1 no es el 111"},
		{"anexo VIII", "anexo VIII", true, "los anexos comparan exacto"},
	} {
		if hay := elFragmentoApuntaAl(c.fragmento, c.articulo); hay != c.apunta {
			t.Errorf("elFragmentoApuntaAl(%q, %q) = %v y se esperaba %v. %s",
				c.fragmento, c.articulo, hay, c.apunta, c.porQue)
		}
	}
}

var (
	// reFragmentoArticulo cubre los dos dialectos de articulo.
	reFragmentoArticuloUE = regexp.MustCompile(`^(?:art|rct)_([0-9]+)([a-z]{0,3})$`)
	reFragmentoAnexoUE    = regexp.MustCompile(`^anx_([IVXLC]+)$`)
	// EL BOE NUMERA POR DECENAS Y ESO CAMBIA LO QUE SE PUEDE AFIRMAR.
	//
	// `a2` es el articulo 2; `a2-4` es el articulo 22. El segundo numero es el
	// ORDEN dentro de la decena, no el articulo, asi que de un ancla del BOE no
	// se puede sacar el numero exacto sin tener el documento delante, y una
	// puerta no baja documentos.
	//
	// Lo que SI es exactamente cierto, y es lo que se afirma:
	//
	//	a<d>        el articulo tiene UN digito y es <d>
	//	a<d>-<k>    el articulo tiene DOS O MAS digitos y empieza por <d>
	//
	// Eso caza el defecto entero —`a6` para el articulo 65 es un ancla sin guion
	// para un articulo de dos digitos— y no puede acusar en falso.
	reFragmentoBOESimple = regexp.MustCompile(`^a([0-9])$`)
	reFragmentoBOEDecena = regexp.MustCompile(`^a([0-9])-[0-9]+$`)

	// reArticuloNumero lee el numero con el que empieza el campo `articulo`.
	reArticuloNumero = regexp.MustCompile(`^([0-9]{1,3})\s*(bis|ter|quater|quinquies|sexies)?`)
	// Los anexos, con la mayuscula indiferente: el campo los escribe «anexo
	// VIII» y el fragmento `anx_VIII`. La primera version bajaba la cadena a
	// minusculas ANTES de aplicar esta expresion, que pedia numeracion romana en
	// mayusculas, asi que no casaba nunca y mandaba cinco obligaciones del CRA al
	// saco de las que no se pueden contrastar. Lo cazo el control negativo, que
	// es exactamente para lo que esta.
	reArticuloAnexo = regexp.MustCompile(`(?i)^anexo\s+([ivxlc]+)`)
	// Y los dos ultimos recursos, para el `articulo` escrito en prosa:
	// «ritual plazum sobre el art. 32.1.d» y «ritual plazum sobre el anexo I,
	// parte II, punto 3».
	reArticuloEnProsa      = regexp.MustCompile(`\bart\.?\s*([0-9]{1,3})`)
	reArticuloAnexoEnProsa = regexp.MustCompile(`(?i)\banexo\s+([ivxlc]+)\b`)

	// ordinalDelArticulo es la misma tabla que usa la puerta de los apartados, y
	// por el mismo motivo: `19 bis` en castellano es `19a` en el identificador.
	ordinalDelArticulo = map[string]string{
		"bis": "a", "ter": "b", "quater": "c", "quinquies": "d", "sexies": "e",
	}
)

// articuloDelFragmento dice a que articulo apunta un fragmento, y con que
// PRECISION, que no es la misma en los dos dialectos.
//
// El ELI de la Union pone el numero entero en el ancla (`art_111`), asi que la
// correspondencia se comprueba exacta. El del BOE pone el primer digito y el
// orden dentro de la decena (`a6-7` es el articulo 65), asi que solo se puede
// comprobar el primer digito y cuantos digitos tiene. Devolver «65» a partir de
// `a6-7` seria una regla aritmetica (decena*10 + orden - 2) que se cumple 164
// veces de 165 en los dos documentos medidos y falla una —el art. 53 de la
// LOPDGDD lleva `a5-12`—, y una puerta que acusa en falso una vez de ciento
// sesenta y cinco se acaba borrando.
//
// Devuelve false para las disposiciones (da, dt, df, dd), que no son articulos y
// cuyo `articulo` se escribe en letra.
func articuloDelFragmento(fr string) (string, bool) {
	if m := reFragmentoArticuloUE.FindStringSubmatch(fr); m != nil {
		return m[1] + m[2], true
	}
	if m := reFragmentoAnexoUE.FindStringSubmatch(fr); m != nil {
		return "anexo " + m[1], true
	}
	if m := reFragmentoBOESimple.FindStringSubmatch(fr); m != nil {
		return m[1], true
	}
	if m := reFragmentoBOEDecena.FindStringSubmatch(fr); m != nil {
		// La decena: se compara contra el primer digito del articulo, y el
		// llamante sabe que ademas el articulo tiene que tener dos digitos o mas.
		return m[1] + "+", true
	}
	return "", false
}

// coincide dice si el articulo al que apunta un fragmento es el que declara el
// campo, teniendo en cuenta que el ancla del BOE por decenas solo fija el primer
// digito.
func elFragmentoApuntaAl(delFragmento, delCampo string) bool {
	if !strings.HasSuffix(delFragmento, "+") {
		return delFragmento == delCampo
	}
	// Ancla por decenas: el articulo tiene que tener DOS o mas digitos y empezar
	// por el de la decena. Las dos mitades hacen falta: sin la primera, `a2-4`
	// valdria para el articulo 2, y sin la segunda valdria para cualquiera.
	decena := strings.TrimSuffix(delFragmento, "+")
	return len(delCampo) >= 2 && strings.HasPrefix(delCampo, decena) && delCampo[0] != 'a'
}

// articuloDelCampo lee el articulo que declara el campo `articulo`, que es texto
// libre. Devuelve false cuando no se deja leer, y quien llama lo CUENTA en vez
// de darlo por bueno: rendirse en silencio es como el hueco se hace grande.
func articuloDelCampo(articulo string) (string, bool) {
	a := strings.TrimSpace(articulo)
	if a == "" {
		return "", false
	}
	if m := reArticuloAnexo.FindStringSubmatch(a); m != nil {
		return "anexo " + strings.ToUpper(m[1]), true
	}
	if m := reArticuloNumero.FindStringSubmatch(a); m != nil {
		return m[1] + ordinalDelArticulo[m[2]], true
	}
	if m := reArticuloEnProsa.FindStringSubmatch(a); m != nil {
		return m[1], true
	}
	if m := reArticuloAnexoEnProsa.FindStringSubmatch(a); m != nil {
		return "anexo " + strings.ToUpper(m[1]), true
	}
	return "", false
}
