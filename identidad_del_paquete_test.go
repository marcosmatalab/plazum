package plazum

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// UN PAQUETE ES UN ACTO, Y SUS DOS IDENTIFICADORES NOMBRAN EL MISMO.
//
// # De donde sale, y por que no basta con haberlo escrito
//
// `paquetes/eidas2` declaraba `urn: urn:eu:reg:2024:1183` (el reglamento
// MODIFICATIVO) y `identificador: reg/2014/910/oj` (el acto BASE). Su propio
// LEEME.md lo decia con todas las letras —«lo que sigue mal: el urn todavia
// nombra al modificativo»— y lo dejaba «para la autoria». Estuvo escrito
// semanas y no impidio nada, porque una nota en un LEEME no es una puerta.
//
// # Que rompe tener dos identidades, que no es cosmetico
//
// El URN es la identidad del paquete: es por donde se empareja con su
// instantanea de `corpus-vigilancia`, es lo que apunta el expediente y es lo que
// resuelve las equivalencias entre marcos. Si nombra al acto equivocado, TODO lo
// que cuelgue de el sale del acto equivocado, y lo peor es que sale en verde:
// `eidas2.art24_3.publicacion_de_la_revocacion` heredaba el 20-05-2024 (vigor del
// 2024/1183) siendo texto base del 910/2014 (marca ▼B en la consolidada
// 02014R0910-20241018, verificada en EUR-Lex el 10-09-2026), y la puerta de
// vigencias decia que CASABA, porque emparejaba por URN y la ficha del
// modificativo declara justamente esa fecha. Acerto la fecha del acto
// equivocado.
//
// La fecha correcta de ese apartado es el 01-07-2016, que es cuando el 910/2014
// se APLICA (art. 52.2, y el 24.3 no esta entre las excepciones que se aplican
// desde el 17-09-2014). Siete anos y diez meses de diferencia en una fila que
// el cliente ve en su calendario.
//
// # Por que la comprobacion es esta y no «que el urn este bien»
//
// Porque «bien» no es comprobable y «iguales entre si» si lo es. Un paquete
// tiene ya dos escrituras del mismo acto por motivos distintos: el URN, que es
// interno y estable, y el ELI, que es la direccion publica de la norma. Que las
// dos digan el mismo ano y el mismo numero no se puede satisfacer por descuido,
// y es exactamente lo que le faltaba a eidas2.
//
// # LO QUE QUEDA JUSTO FUERA DE ESTA PUERTA, dicho al escribirla
//
// Los marcos sin ELI (ISO, PCI DSS, SOC 2, TISAX), cuyo identificador es un
// numero de norma o una direccion y no tiene ano y numero que contrastar. No se
// dan por buenos: se cuentan, y el recuento sale por pantalla. Ahi no puede
// aparecer el defecto que esta puerta persigue, porque un catalogo privativo no
// tiene actos modificativos con ficha propia en el almacen de vigilancia.
//
// Y lo que tampoco mira: que el ELI exista o resuelva. Eso es red, y esto es una
// puerta de arbol.
func TestElURNDeUnPaqueteNombraElMismoActoQueSuIdentificador(t *testing.T) {
	fs, err := filepath.Glob(filepath.Join("paquetes", "*", "paquete.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) < MinimoDeMarcos {
		t.Fatalf("solo %d paquetes: el arnes no ha encontrado el corpus", len(fs))
	}

	contrastados := 0
	var sinELI []string
	for _, f := range fs {
		b, err := os.ReadFile(f) // #nosec G304 -- ruta derivada de un glob del propio repositorio
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			URN           string `json:"urn"`
			Identificador struct {
				Tipo  string `json:"tipo"`
				Valor string `json:"valor"`
			} `json:"identificador"`
		}
		if err := json.Unmarshal(b, &doc); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		delELI, ok := actoDelELI(doc.Identificador.Tipo, doc.Identificador.Valor)
		if !ok {
			sinELI = append(sinELI, fmt.Sprintf("%s (%s)", doc.URN, doc.Identificador.Tipo))
			continue
		}
		delURN, ok := actoDelURN(doc.URN)
		if !ok {
			t.Errorf("%s: el urn %q no termina en «ano:numero» y no se puede contrastar con su "+
				"identificador. Un urn que no nombra un acto no identifica nada.", f, doc.URN)
			continue
		}
		contrastados++
		if delURN == delELI {
			continue
		}
		t.Errorf(`%s: el urn y el identificador nombran DOS ACTOS DISTINTOS.

  urn            %s   ->  %s
  identificador  %s   ->  %s

  Un paquete es UN acto. El urn es la identidad: por el se empareja con su
  instantanea de corpus-vigilancia, por el lo apunta el expediente y por el se
  resuelven las equivalencias. Si nombra a otro acto, todo lo que herede sale del
  acto equivocado Y SALE EN VERDE, porque la puerta de vigencias contrasta contra
  la ficha del acto que el urn nombra.

  Es lo que le paso a eidas2: el urn decia 2024:1183 (el reglamento que MODIFICA)
  y el identificador reg/2014/910 (el acto MODIFICADO). Una obligacion de texto
  base heredo la fecha de vigor del modificativo y la puerta la confirmo.

  Arreglo: el urn nombra el acto del que sale el texto. Una obligacion cuyo texto
  venga de un acto modificativo lo dice en su cita y lleva vigencia propia, que es
  para lo que existe origen: "propia".`,
			f, doc.URN, delURN, doc.Identificador.Valor, delELI)
	}

	if contrastados == 0 {
		t.Fatal("ni un paquete con identificador ELI contrastado: el extractor se ha roto y " +
			"esta puerta estaria verde sin mirar nada")
	}
	sort.Strings(sinELI)
	if !t.Failed() {
		t.Logf("MEDIDO: %d paquetes con identificador ELI, urn y ELI de acuerdo en los %d.\n"+
			"  Fuera del alcance por no tener ELI (%d): %v",
			contrastados, contrastados, len(sinELI), sinELI)
	}
}

// EL CONTROL NEGATIVO, EN LAS DOS DIRECCIONES.
//
// Sin esto, la puerta de arriba pasaria con un extractor que devolviera siempre
// lo mismo para los dos lados, o que se rindiera («ok=false») en todo lo que no
// entiende: las dos cosas dan verde sobre un corpus sano y no se distinguen
// desde fuera.
//
// Los dos dialectos de ELI que hay en el corpus tienen la forma distinta y por
// eso van los dos: el europeo lleva ano y numero seguidos (`reg/2014/910/oj`) y
// el espanol mete la fecha completa por medio (`es/rd/2022/05/03/311/con`), asi
// que un extractor escrito para uno se equivoca de campo en el otro sin fallar.
func TestElExtractorDeActoAcusaYSeCalla(t *testing.T) {
	for _, c := range []struct {
		nombre, tipo, valor, quiero string
		hay                         bool
	}{
		{"eli europeo", "eli-ue", "reg/2014/910/oj", "2014:910", true},
		{"eli europeo de directiva", "eli-ue", "dir/2022/2555/oj", "2022:2555", true},
		{"eli europeo de reglamento de ejecucion", "eli-ue", "reg_impl/2024/2690/oj", "2024:2690", true},
		{"eli espanol con la fecha por medio", "eli-es", "es/rd/2022/05/03/311/con", "2022:311", true},
		{"eli espanol de ley organica", "eli-es", "es/lo/2018/12/05/3/con", "2018:3", true},
		{"identificador de una norma privativa", "iso", "ISO/IEC 27001:2022", "", false},
		{"una direccion, que no es un eli", "url", "https://enx.com/en-US/TISAX/", "", false},
		{"un eli europeo cortado", "eli-ue", "reg/2014", "", false},
		{"un eli espanol cortado", "eli-es", "es/rd/2022/05", "", false},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			acto, hay := actoDelELI(c.tipo, c.valor)
			if hay != c.hay {
				t.Fatalf("actoDelELI(%q, %q) dijo hay=%v y se esperaba %v. Un extractor que se "+
					"rinde de mas deja paquetes sin contrastar; uno que se rinde de menos "+
					"inventa un acto", c.tipo, c.valor, hay, c.hay)
			}
			if acto != c.quiero {
				t.Errorf("actoDelELI(%q, %q) = %q y se esperaba %q", c.tipo, c.valor, acto, c.quiero)
			}
		})
	}

	for _, c := range []struct {
		urn, quiero string
		hay         bool
	}{
		{"urn:eu:reg:2014:910", "2014:910", true},
		{"urn:es:rdl:2018:19", "2018:19", true},
		{"urn:eu:reg-ejec:2024:2690", "2024:2690", true},
		{"urn:iso-iec:27001:2022", "27001:2022", true}, // forma valida, aunque no se use
		{"urn:corto", "", false},
		{"", "", false},
	} {
		acto, hay := actoDelURN(c.urn)
		if hay != c.hay || acto != c.quiero {
			t.Errorf("actoDelURN(%q) = (%q, %v) y se esperaba (%q, %v)",
				c.urn, acto, hay, c.quiero, c.hay)
		}
	}

	// Y LA DIRECCION QUE IMPORTA: el par que la puerta tiene que poner rojo.
	// Sin este caso, un extractor que devolviera el mismo valor para los dos
	// lados pasaria los de arriba y no acusaria nunca.
	delURN, _ := actoDelURN("urn:eu:reg:2024:1183")
	delELI, _ := actoDelELI("eli-ue", "reg/2014/910/oj")
	if delURN == delELI {
		t.Errorf("el extractor da %q para el urn del acto modificativo y %q para el ELI del acto "+
			"base, y son iguales: entonces la puerta no puede distinguir los dos actos, que es "+
			"lo unico que hace", delURN, delELI)
	}
}

// actoDelURN devuelve el «ano:numero» final de un URN, que es lo que identifica
// al acto dentro de su jurisdiccion y su rango.
func actoDelURN(urn string) (string, bool) {
	t := strings.Split(urn, ":")
	if len(t) < 3 {
		return "", false
	}
	ano, num := t[len(t)-2], t[len(t)-1]
	if ano == "" || num == "" {
		return "", false
	}
	return ano + ":" + num, true
}

// actoDelELI devuelve el mismo «ano:numero» sacado de un identificador ELI.
//
// Los dos dialectos colocan los campos en sitios distintos y por eso hay dos
// ramas y no una expresion regular comun:
//
//	eli-ue  reg/2014/910/oj            ano y numero seguidos, tras el rango
//	eli-es  es/rd/2022/05/03/311/con   la fecha completa por medio, y el numero
//	                                   despues del dia
//
// Cualquier otro tipo devuelve false, y quien llama lo cuenta en vez de darlo
// por bueno.
func actoDelELI(tipo, valor string) (string, bool) {
	partes := strings.Split(strings.Trim(valor, "/"), "/")
	switch tipo {
	case "eli-ue":
		// rango / ano / numero / ...
		if len(partes) < 3 {
			return "", false
		}
		return partes[1] + ":" + partes[2], true
	case "eli-es":
		// es / rango / ano / mes / dia / numero / ...
		if len(partes) < 6 {
			return "", false
		}
		return partes[2] + ":" + partes[5], true
	}
	return "", false
}
