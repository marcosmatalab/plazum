package plazum

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// obligacionBilingue es lo que estas puertas necesitan leer de un paquete.
type obligacionBilingue struct {
	Paquete    string
	ID         string
	Articulo   string
	TextoLegal string
	Versiones  map[string]string // lengua -> texto
}

// leerBilingues devuelve las obligaciones del corpus que traen alguna version
// linguistica, leyendo el JSON crudo por el mismo motivo que la puerta de D-25:
// nucleo/corpus no usa DisallowUnknownFields, asi que lo que se quiera vigilar
// del fichero hay que leerlo del fichero.
func leerBilingues(t *testing.T) []obligacionBilingue {
	t.Helper()
	fs, err := filepath.Glob(filepath.Join("paquetes", "*", "paquete.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) < MinimoDeMarcos {
		t.Fatalf("solo %d paquetes: el arnes no ha encontrado el corpus", len(fs))
	}
	var out []obligacionBilingue
	for _, f := range fs {
		b, err := os.ReadFile(f) // #nosec G304 -- ruta derivada de un glob del propio repositorio
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			Obligaciones []struct {
				ID         string `json:"id"`
				Articulo   string `json:"articulo"`
				TextoLegal string `json:"texto_legal"`
				Versiones  map[string]struct {
					Texto string `json:"texto"`
				} `json:"versiones_linguisticas"`
			} `json:"obligaciones"`
		}
		if err := json.Unmarshal(b, &doc); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for _, o := range doc.Obligaciones {
			if len(o.Versiones) == 0 {
				continue
			}
			vs := map[string]string{}
			for lengua, v := range o.Versiones {
				vs[lengua] = v.Texto
			}
			out = append(out, obligacionBilingue{
				Paquete: filepath.Base(filepath.Dir(f)), ID: o.ID,
				Articulo: o.Articulo, TextoLegal: o.TextoLegal, Versiones: vs,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// DOS VERSIONES DE UN ARTICULO SON LA MISMA LEY SI Y SOLO SI CUBREN LOS MISMOS
// APARTADOS.
//
// # Por que esta puerta y no una de formato
//
// Porque lo que hay que impedir no es que el JSON este mal escrito, es que la
// pagina inglesa y la castellana enseñen **trozos distintos de la norma** bajo
// la misma obligacion, con el mismo plazo y la misma fecha. Un cliente que lea
// en ingles un apartado que la version castellana no trae esta leyendo otra
// cosa, y nada se lo dice.
//
// Con dos obligaciones se alinea a ojo, que es como se alinearon las de `mdr`.
// Con las 38 del AI Act, no. Esta puerta es la condicion para seguir.
//
// # EL DETECTOR ES FIABLE EN LAS DOS LENGUAS, Y ESO SE MIDIO ANTES DE ESCRIBIRLO
//
// Medido el 10-09-2026 sobre CUATRO actos y los DOS dialectos del XHTML de
// Cellar, comparando lengua contra lengua articulo por articulo:
//
//	MDR 2017/745                          109 articulos con apartados, 109 iguales
//	RGPD 2016/679                          82                           81
//	NIS2 2022/2555                         38                           38
//	eIDAS consolidado 02014R0910-20241018  71                           70
//	                                      ---                          ---
//	                                      300                          298   99,3 %
//
// **Las dos que no coinciden no son fallos del detector: son el punto que le
// falta al texto oficial**, y hay una en cada lengua, que es lo que impide
// leerlo como que una lengua se transcribe peor. El RGPD publica en castellano
// «6 La Comision entablara consultas» sin el punto del apartado 6; el eIDAS
// consolidado publica en ingles «3a The validity of a certification» sin el
// punto del 3 bis. El detector EXIGE el punto a proposito: aflojarlo para
// recoger esos dos convertiria en apartado cualquier parrafo que empiece por una
// cifra, y hay muchos.
//
// # LO QUE LA MEDIDA ENCONTRO Y NO SE ESPERABA, que es lo que decide el diseño
//
// **Las dos lenguas NO nombran igual los apartados añadidos.** Donde el
// castellano escribe «1 bis», «1 ter», «1 quater», el ingles escribe «1a», «1b»,
// «1c». Sin normalizar, el acuerdo sobre el consolidado de eIDAS caia al 88,7 %
// (63 de 71) y las ocho diferencias eran todas de vocabulario y ninguna de
// contenido. La correspondencia es posicional y exacta (bis→a, ter→b, quater→c,
// quinquies→d, sexies→e) y va en una tabla, no en una expresion regular por
// lengua: una segunda expresion regular es una segunda lista que se queda vieja.
//
// # LO QUE ESTA PUERTA NO MIRA, dicho al escribirla
//
// Que el texto de cada apartado DIGA lo mismo. Eso no lo puede afirmar ninguna
// puerta de este repositorio sin traducir, que es justo lo que D-11 prohibe. Lo
// que se afirma es la COBERTURA, que es mecanica y es la que se rompe sola
// cuando alguien copia de menos.
func TestLasVersionesLinguisticasDeUnaObligacionCubrenElMismoArticulado(t *testing.T) {
	obs := leerBilingues(t)
	if len(obs) == 0 {
		t.Fatal("ni una obligacion con version linguistica: o el corpus perdio las versiones " +
			"o el lector se rompio, y en los dos casos esta puerta estaria verde sin mirar nada")
	}

	comparadas, mudas := 0, 0
	var sinApartados []string
	for _, o := range obs {
		base := apartadosDelTexto(o.TextoLegal)
		for _, lengua := range lenguasDe(o.Versiones) {
			otra := apartadosDelTexto(o.Versiones[lengua])
			if len(base) == 0 && len(otra) == 0 {
				mudas++
				sinApartados = append(sinApartados, o.ID+" ("+lengua+")")
				continue
			}
			comparadas++
			if reflect.DeepEqual(base, otra) {
				continue
			}
			t.Errorf(`%s (%s): las dos lenguas NO cubren los mismos apartados.

  %s  %v
  %s  %v
  falta en %s: %v
  falta en %s: %v

  Dos versiones de un articulo son la misma ley si y solo si cubren el mismo
  articulado. Si no, la pagina de un idioma enseña un trozo de norma que la del
  otro no enseña, bajo la misma obligacion, con el mismo plazo y la misma fecha,
  y nada se lo dice al que lee.

  Arreglo: recortar la version larga al mismo conjunto, o completar la corta. El
  conjunto que manda es el que se derivo de la version en %s, que es la lengua
  del texto legal.`,
				o.ID, o.Paquete,
				LenguaDelTextoLegalCorpus, base, lengua, otra,
				lengua, faltan(base, otra), LenguaDelTextoLegalCorpus, faltan(otra, base),
				LenguaDelTextoLegalCorpus)
		}
	}

	if comparadas == 0 {
		t.Fatal("ninguna pareja de versiones traia apartados numerados, asi que esta puerta " +
			"ha comparado el vacio contra el vacio y ha salido verde. Eso no es un verde")
	}

	// EL HUECO VA CON SU CARDINAL Y CON IGUALDAD EXACTA.
	//
	// Una obligacion cuyo texto no numera apartados no se puede contrastar, y eso
	// no es un fallo: hay articulos del DOUE que son un solo parrafo sin numerar
	// (el art. 19 del RGPD lo es). Lo que no puede pasar es que ese numero crezca
	// en silencio, porque cada unidad es una obligacion bilingue que NADIE
	// comprueba, y desde fuera un hueco y un acierto se ven igual.
	const mudasDeclaradas = 0
	if mudas != mudasDeclaradas {
		t.Errorf("hay %d pareja(s) de versiones sin apartados numerados y se declaran %d.\n"+
			"  Las de ahora: %v\n"+
			"  Si ha SUBIDO, hay obligaciones bilingues que esta puerta no comprueba: o el "+
			"articulo de verdad no numera (art. 19 del RGPD, parrafo unico) y se sube este "+
			"numero A PROPOSITO diciendo cual, o la transcripcion se dejo fuera el numero de "+
			"apartado y hay que ponerlo, que ademas es mas fiel.\n"+
			"  Si ha BAJADO, sobra parte de esta cuenta.",
			mudas, mudasDeclaradas, sinApartados)
	}
	if !t.Failed() {
		t.Logf("MEDIDO: %d pareja(s) de versiones comparadas apartado a apartado sobre %d "+
			"obligacion(es) bilingue(s), %d sin apartados numerados.",
			comparadas, len(obs), mudas)
	}
}

// EL CAMPO `articulo` DICE LOS APARTADOS QUE TRAE EL TEXTO.
//
// # De donde sale
//
// De que con dos lenguas la misma afirmacion tiene que ser cierta en TRES sitios
// y hasta hoy solo se comprobaba en uno. `mdr.art87` declaraba «87.1, 87.3, 87.4
// y 87.5» y su `texto_legal` traia los apartados 1, 2, 3, 4 y 5: **el 2 estaba en
// el texto y no en la declaracion**, y el 87.2 no es relleno, es el que dice que
// el plazo de la notificacion depende de la gravedad del incidente. La segunda,
// `mdr.art16_4`, declaraba solo «16.4» y transcribe tambien el 16.2, que es el
// que define que actividades disparan el preaviso.
//
// # POR QUE SOLO ALCANZA A LAS BILINGUES, y es una decision medida
//
// Porque `N.M` no significa lo mismo en todos los marcos. Medido el 10-09-2026
// sobre el corpus entero: la regla aplicada a todo acusaria **67** obligaciones,
// y la inmensa mayoria son falsos positivos de la propia regla —`iso27001.9.2.1`
// declara «9.2.1», que es una CLAUSULA y no «articulo 9, apartado 2»—. Una puerta
// que acusa a 67 filas legitimas se borra en una semana.
//
// En una obligacion con versiones linguisticas el terreno es otro y por eso ahi
// si se puede exigir: es un articulo del DOUE, con apartados numerados, y el
// conjunto lo controlamos nosotros al transcribir.
//
// # POR QUE EL CAMPO `articulo` Y NO LA `cita`
//
// Porque la cita es PROSA y nombra legitimamente articulos que no son el suyo:
// la de `mdr.art87` menciona el art. 92, apartados 5 y 7, y el art. 88, que son
// remisiones del propio texto legal. Un lector de N.M sobre la cita convertiria
// cada remision en un apartado declarado y acusaria a una cita correcta. El
// campo `articulo` es el unico sitio del formato donde el conjunto se declara y
// nada mas. La cita se corrige a mano cuando cambia el campo, y eso se dice
// aqui en vez de fingir que tambien esta vigilado.
func TestElCampoArticuloDeUnaObligacionBilingueDiceLosApartadosQueTraeElTexto(t *testing.T) {
	obs := leerBilingues(t)
	if len(obs) == 0 {
		t.Fatal("ni una obligacion con version linguistica: esta puerta no esta mirando nada")
	}

	contrastadas := 0
	var sinDeclarar []string
	for _, o := range obs {
		delTexto := apartadosDelTexto(o.TextoLegal)
		declarados := apartadosDeclarados(o.Articulo)
		if len(declarados) == 0 || len(delTexto) == 0 {
			sinDeclarar = append(sinDeclarar, o.ID+" (articulo: "+o.Articulo+")")
			continue
		}
		contrastadas++
		if reflect.DeepEqual(declarados, delTexto) {
			continue
		}
		t.Errorf(`%s: el campo «articulo» y el texto transcrito no dicen lo mismo.

  articulo declara  %v   (%q)
  el texto trae     %v
  en el texto y no en la declaracion: %v
  en la declaracion y no en el texto: %v

  Con dos lenguas, esta afirmacion tiene que ser cierta en TRES sitios: la
  castellana, la inglesa y el campo que las nombra. Un apartado que esta en el
  texto y no en la declaracion se transcribe, se enseña y no se cita; uno que
  esta en la declaracion y no en el texto se cita y no se enseña.

  Arreglo: que el campo diga lo que el texto trae. Si un apartado sobra en el
  texto, quitalo de los dos idiomas.`,
			o.ID, declarados, o.Articulo, delTexto,
			faltan(declarados, delTexto), faltan(delTexto, declarados))
	}

	if contrastadas == 0 {
		t.Fatal("ninguna obligacion bilingue tenia a la vez apartados declarados y apartados " +
			"en el texto: esta puerta ha salido verde sin contrastar nada")
	}

	// El hueco, con su cardinal y su igualdad exacta, por lo mismo que el otro.
	const sinContrastarDeclaradas = 0
	if len(sinDeclarar) != sinContrastarDeclaradas {
		t.Errorf("hay %d obligacion(es) bilingue(s) que no se pueden contrastar y se declaran "+
			"%d.\n  Las de ahora: %v\n"+
			"  Se llega aqui cuando el campo «articulo» no tiene forma N.M (el art. 19 del RGPD "+
			"es «19» a secas, porque no tiene apartados) o cuando el texto no los numera. Si "+
			"sube, hay obligaciones bilingues cuya declaracion no comprueba nadie.",
			len(sinDeclarar), sinContrastarDeclaradas, sinDeclarar)
	}
	if !t.Failed() {
		t.Logf("MEDIDO: %d declaracion(es) contrastadas contra su texto, %d sin contrastar.",
			contrastadas, len(sinDeclarar))
	}
}

// EL CONTROL NEGATIVO DEL DETECTOR, EN LAS DOS DIRECCIONES Y EN LAS DOS LENGUAS.
//
// Sin esto, las dos puertas de arriba pasarian con un detector que no encontrara
// nunca nada (todo serian conjuntos vacios iguales) o que encontrara de mas por
// igual en los dos lados. Las dos cosas dan verde y desde fuera no se distinguen
// de un corpus alineado.
func TestElDetectorDeApartadosAcusaYSeCalla(t *testing.T) {
	for _, c := range []struct {
		nombre, texto string
		quiero        []string
		porQue        string
	}{
		{
			nombre: "castellano corriente",
			texto:  "1. El fabricante informara.\n\n2. Como norma general.\n\n3. El fabricante informara de cualquier incidente.",
			quiero: []string{"1", "2", "3"},
			porQue: "es la forma normal del DOUE en castellano",
		},
		{
			nombre: "ingles corriente",
			texto:  "1. Manufacturers shall report.\n\n2. As a general rule.\n\n3. Manufacturers shall report any serious incident.",
			quiero: []string{"1", "2", "3"},
			porQue: "es la forma normal del DOUE en ingles, y tiene que dar lo mismo",
		},
		{
			nombre: "apartados anadidos en castellano",
			texto:  "1. Al expedir.\n\n1 bis. La verificacion.\n\n1 ter. La verificacion de los atributos.\n\n2. Los prestadores.",
			quiero: []string{"1", "1a", "1b", "2"},
			porQue: "un consolidado numera lo anadido con bis y ter, y se normaliza a la forma inglesa",
		},
		{
			nombre: "los mismos apartados anadidos en ingles",
			texto:  "1. When issuing.\n\n1a. The verification.\n\n1b. The verification of the attributes.\n\n2. A qualified trust service provider.",
			quiero: []string{"1", "1a", "1b", "2"},
			porQue: "ES DONDE ESTABA EL FALLO: sin normalizar, el ingles y el castellano del " +
				"MISMO articulo dan conjuntos distintos y la puerta acusaria a una " +
				"transcripcion correcta",
		},
		{
			nombre: "un apartado que la fuente abre con comillas",
			texto:  "En el articulo 2 se anade el siguiente apartado:\n\n«4. La Comision podra adoptar medidas.",
			quiero: []string{"4"},
			porQue: "un acto modificativo cita el apartado que anade entre comillas, y en ingles " +
				"la comilla es otra: si el detector no las quita, la misma norma da " +
				"conjuntos distintos por lengua",
		},
		{
			nombre: "la misma, con la comilla simple inglesa",
			texto:  "In Article 2, the following paragraph is added:\n\n‘4. The Commission may adopt measures.",
			quiero: []string{"4"},
			porQue: "medido: era la unica diferencia entre las dos lenguas del art. 119 del MDR",
		},
		{
			nombre: "una fecha al principio de un parrafo NO es un apartado",
			texto:  "1 de julio de 2016 es la fecha de aplicacion.\n\n21 de mayo de 2025, la Comision establecera.",
			quiero: nil,
			porQue: "sin el punto no hay apartado, y por eso el punto es obligatorio",
		},
		{
			nombre: "las letras de un apartado no son apartados",
			texto:  "a) | todo incidente grave.\n\nb) | toda accion correctiva.\n\n(a) | the rule of law.",
			quiero: nil,
			porQue: "si contaran, cada articulo con letras daria un conjunto distinto segun " +
				"cuantas letras se transcriban",
		},
		{
			nombre: "un rango de apartados citado dentro del texto",
			texto:  "Los apartados 2 a 9 se aplicaran.\n\n12 de mayo de 2020 quedo derogado.",
			quiero: nil,
			porQue: "una cifra seguida de texto no es una cabecera de apartado",
		},
		{
			nombre: "el apartado sin punto que publica la fuente NO se recoge",
			texto:  "5. Cuando la informacion disponible.\n\n6 La Comision entablara consultas.\n\n7. Toda decision.",
			quiero: []string{"5", "7"},
			porQue: "es literalmente lo que el RGPD publica en castellano en su art. 45, y se " +
				"deja fuera A PROPOSITO: recogerlo obligaria a aceptar cualquier parrafo " +
				"que empiece por una cifra",
		},
		{
			nombre: "un apartado repetido no se cuenta dos veces",
			texto:  "1. Primero.\n\n1. Otra vez el uno.\n\n2. Segundo.",
			quiero: []string{"1", "2"},
			porQue: "el conjunto es de apartados, no de parrafos",
		},
		{
			nombre: "texto vacio",
			texto:  "",
			quiero: nil,
			porQue: "un referencial no tiene texto legal y no se le inventa articulado",
		},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			hay := apartadosDelTexto(c.texto)
			if !reflect.DeepEqual(hay, c.quiero) {
				t.Errorf("apartadosDelTexto dio %v y se esperaba %v.\n  Por que importa: %s",
					hay, c.quiero, c.porQue)
			}
		})
	}

	// Y EL DECLARADOR, que es el otro lado de la puerta del campo `articulo`.
	for _, c := range []struct {
		articulo string
		quiero   []string
	}{
		{"87.1, 87.3, 87.4 y 87.5", []string{"1", "3", "4", "5"}},
		{"16.4", []string{"4"}},
		{"33.1", []string{"1"}},
		{"ritual plazum sobre el art. 32.1.d", []string{"1"}},
		{"19", nil},             // articulo de parrafo unico
		{"24, apartado 3", nil}, // la forma en prosa no se interpreta
		{"anexo VIII, punto 3.2", []string{"2"}},
	} {
		if hay := apartadosDeclarados(c.articulo); !reflect.DeepEqual(hay, c.quiero) {
			t.Errorf("apartadosDeclarados(%q) = %v y se esperaba %v", c.articulo, hay, c.quiero)
		}
	}
}

// LenguaDelTextoLegalCorpus es la lengua en la que se transcribe el texto legal
// del corpus. Se nombra aqui para que los mensajes no la escriban a mano.
const LenguaDelTextoLegalCorpus = "es"

var (
	// reApartado reconoce la cabecera de un apartado al principio de un bloque.
	//
	// EL PUNTO ES OBLIGATORIO, y es la decision del detector. Sin el, «1 de julio
	// de 2016» y «21 de mayo de 2025» serian apartados, y hay muchos parrafos que
	// empiezan por una cifra. El precio son los dos apartados que la fuente
	// publica sin punto (RGPD art. 45.6 en castellano, eIDAS art. 30.3 bis en
	// ingles), contados arriba.
	//
	// La clase de comillas de delante no es adorno: un acto modificativo cita
	// entre comillas el apartado que anade, y la comilla de apertura del
	// castellano no es la del ingles.
	reApartado = regexp.MustCompile(`^[«‘“"\s]*([0-9]{1,3})[ \t]*(bis|ter|quater|quinquies|sexies|[a-h])?[ \t]*\.`)

	// reDeclarado reconoce «87.1» dentro del campo `articulo`.
	reDeclarado = regexp.MustCompile(`\b([0-9]{1,3})\.([0-9]{1,3})([a-h])?\b`)

	// ordinalLatino pasa la forma castellana de un apartado anadido a la inglesa,
	// que es la que se usa como forma canonica por ser mas corta. La
	// correspondencia es posicional y la fija el propio legislador.
	ordinalLatino = map[string]string{
		"bis": "a", "ter": "b", "quater": "c", "quinquies": "d", "sexies": "e",
	}
)

// apartadosDelTexto devuelve, en orden de aparicion y sin repetir, los apartados
// que un texto legal transcrito declara.
func apartadosDelTexto(texto string) []string {
	var out []string
	visto := map[string]bool{}
	for _, bloque := range strings.Split(texto, "\n\n") {
		m := reApartado.FindStringSubmatch(strings.TrimSpace(bloque))
		if m == nil {
			continue
		}
		sufijo := m[2]
		if eq, hay := ordinalLatino[sufijo]; hay {
			sufijo = eq
		}
		n := m[1] + sufijo
		if visto[n] {
			continue
		}
		visto[n] = true
		out = append(out, n)
	}
	return out
}

// apartadosDeclarados saca del campo `articulo` los apartados que nombra, en
// orden y sin repetir. Devuelve nil si el campo no tiene ninguna forma N.M, que
// es el caso de un articulo de parrafo unico y el de una referencia en prosa.
func apartadosDeclarados(articulo string) []string {
	var out []string
	visto := map[string]bool{}
	for _, m := range reDeclarado.FindAllStringSubmatch(articulo, -1) {
		n := m[2] + m[3]
		if visto[n] {
			continue
		}
		visto[n] = true
		out = append(out, n)
	}
	return out
}

// faltan devuelve lo que esta en a y no en b.
func faltan(a, b []string) []string {
	hay := map[string]bool{}
	for _, x := range b {
		hay[x] = true
	}
	var out []string
	for _, x := range a {
		if !hay[x] {
			out = append(out, x)
		}
	}
	return out
}

// lenguasDe devuelve las lenguas de un mapa de versiones, ordenadas, para que el
// recorrido y los mensajes no dependan del orden del mapa.
func lenguasDe(vs map[string]string) []string {
	out := make([]string, 0, len(vs))
	for l := range vs {
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}
