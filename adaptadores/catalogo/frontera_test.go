package catalogo

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// La regla que este fichero hace cumplir: EL CATALOGO NUNCA TRANSPORTA TEXTO
// LEGAL. Y la hace cumplir de verdad, no la afirma.
//
// Se comprueba en dos planos, porque uno solo no basta:
//
//	el cargador  rechaza lo que tiene FORMA de norma, mire quien lo mire y
//	             haya corpus instalado o no. Es la frontera que vale tambien
//	             para un catalogo que llegue de fuera.
//	el contraste rechaza lo que ES texto del corpus instalado, comparando
//	             trozo a trozo contra el texto legal de todos los paquetes.
//	             Eso el cargador no lo puede saber sin atarse al corpus.
//
// Las dos con control negativo: se demuestra que saltan cuando deben Y que no
// saltan con una cadena de interfaz normal, porque un detector que dice que no
// a todo protege lo mismo que uno que dice que si a todo.

func TestElCargadorRechazaLoQueNoEsUnaCadenaDeInterfaz(t *testing.T) {
	casos := []struct {
		nombre   string
		clave    string
		valor    string
		mencione string
	}{
		{"clave con nombre de norma", "demo.art31.titulo", "Auditoría",
			"espacio de nombres"},
		{"clave sin espacio de nombres", "guardar", "Guardar", "forma de una clave"},
		{"clave con mayúsculas", "Ui.Guardar", "Guardar", "forma de una clave"},
		{"clave con tilde", "ui.más", "Más", "forma de una clave"},
		{"clave con espacio", "ui. guardar", "Guardar", "forma de una clave"},

		{"cita de un artículo", "aviso.plazo",
			"El responsable la notificará a la autoridad de control en el plazo de 72 horas, según el artículo 33.",
			"cita normativa"},
		{"cita de un real decreto", "aviso.marco",
			"Medidas del Real Decreto 311 de 2022, con su categoría y su perfil.",
			"cita normativa"},
		{"cita de un reglamento europeo", "aviso.rgpd",
			"Lo exige el Reglamento (UE) 2016 sobre protección de datos.",
			"cita normativa"},
		{"cita en inglés", "aviso.annex", "See Annex I for the list of essential entities.",
			"cita normativa"},
		{"cita con abreviatura", "aviso.abrev", "Recuerde el art. 31 antes de cerrar.",
			"cita normativa"},

		{"marcado HTML", "ui.guardar", "<b>Guardar</b>", "marcado HTML"},
		{"script incrustado", "ui.guardar", "Guardar<script>alert(1)</script>",
			"marcado HTML"},
		{"URL ejecutable", "ui.ir", "javascript:alert(1)", "marcado HTML"},
		{"entidad HTML numérica", "ui.ir", "Guardar &#60;", "marcado HTML"},

		{"identificador de paquete", "aviso.paquete", "Instale urn:demo:marco antes de seguir.",
			"identificador de paquete"},
		{"salto de línea", "aviso.dos_lineas", "Primera línea.\nSegunda línea.",
			"caracter de control"},
		{"marca de orden de escritura", "ui.guardar", "Guardar‮gnitset",
			"invisible de formato"},
		{"espacio de ancho cero", "ui.guardar", "Guar​dar",
			"invisible de formato"},
		{"párrafo largo", "aviso.largo", strings.Repeat("palabra ", 40), "limite son"},
	}
	for _, cs := range casos {
		t.Run(cs.nombre, func(t *testing.T) {
			_, err := Cargar(fstest.MapFS{
				"es.json": &fstest.MapFile{Data: []byte(objetoJSON(cs.clave, cs.valor))},
			}, "es")
			if err == nil {
				t.Fatalf("el cargador acepto %q con el valor %q, y eso no es una cadena "+
					"de interfaz", cs.clave, cs.valor)
			}
			if !strings.Contains(err.Error(), cs.mencione) {
				t.Fatalf("el error no menciona %q y dice: %v", cs.mencione, err)
			}
		})
	}
}

// Control negativo de la frontera: lo legitimo tiene que pasar.
//
// La lista lleva a proposito los casos que un detector ingenuo se cargaria: un
// boton que dice "Ver el artículo" sin numero detras, un porcentaje literal,
// una cifra con el signo menor, y una etiqueta que menciona una ley sin citarla.
func TestLaFronteraNoRechazaCadenasDeInterfazLegitimas(t *testing.T) {
	buenos := map[string]string{
		"ui.ver_articulo":     "Ver el artículo",
		"tabla.cobertura":     "Cubierto al 80%",
		"aviso.presupuesto":   "El binario ocupa <25 MB",
		"tabla.seccion":       "Sección 2 del formulario",
		"aviso.legal":         "Esto no es asesoramiento jurídico ni sustituye a su abogado.",
		"error.limite":        "Demasiadas peticiones. Espere %s y vuelva a intentarlo.",
		"pantalla.hoy.titulo": "Hoy",
		"estado.aplica":       "Te aplica",
	}
	for clave, valor := range buenos {
		if _, err := Cargar(fstest.MapFS{
			"es.json": &fstest.MapFile{Data: []byte(objetoJSON(clave, valor))},
		}, "es"); err != nil {
			t.Errorf("la frontera rechaza una cadena de interfaz legitima (%q = %q): %v.\n"+
				"Un detector que dice que no a todo se acaba desactivando, y entonces no "+
				"protege de nada", clave, valor, err)
		}
	}
}

// ---------------------------------------------------------------------------
// El contraste contra el corpus instalado
// ---------------------------------------------------------------------------

// ventanaDeTrozo son las palabras consecutivas que forman un trozo comparable.
//
// Seis. Con menos aparecen coincidencias inocentes ("en el plazo de", "de la
// autoridad de"), y un detector con falsos positivos se acaba desactivando. Con
// mas se escapa la cita corta. Lo que se le escapa a seis, dicho para que
// conste: una parafrasis, y un trozo literal de cinco palabras o menos. Contra
// eso queda la frontera del cargador y la revision humana.
const ventanaDeTrozo = 6

func TestElCatalogoNoTransportaTextoDelCorpus(t *testing.T) {
	c := nuevoParaTest(t)
	trozos := trozosDelCorpus(t)
	for _, idioma := range c.Idiomas() {
		for _, clave := range c.Claves() {
			valor := c.Traducir(idioma, clave)
			if de, hay := delataTextoDelCorpus(trozos, valor); hay {
				t.Errorf("la clave %q en %q lleva texto del paquete %s.\n"+
					"El catalogo NO transporta texto legal: traducirlo crea obra derivada y "+
					"se sale de la estratificacion de licencias del corpus. Arreglo: el "+
					"texto va en su paquete, en el idioma de su fuente", clave, idioma, de)
			}
		}
	}
}

// Control negativo del contraste. Sin esto, el test de arriba estaria verde
// aunque el detector no mirase nada.
func TestControlNegativoElContrasteConElCorpusMuerde(t *testing.T) {
	trozos := trozosDelCorpus(t)
	fragmento := fragmentoRealDelCorpus(t)

	if de, hay := delataTextoDelCorpus(trozos, fragmento); !hay {
		t.Fatalf("el contraste NO caza un trozo literal del corpus (%q), asi que el test "+
			"de arriba esta verde por no mirar nada", fragmento)
	} else {
		t.Logf("cazado, como debe: %q sale de %s", fragmento, de)
	}

	// Y camuflado dentro de una cadena con pinta de rotulo, que es como se
	// colaria de verdad: alguien "explica" la norma en la propia interfaz.
	camuflado := "Recuerde: " + fragmento
	if _, hay := delataTextoDelCorpus(trozos, camuflado); !hay {
		t.Fatalf("el contraste no caza el texto del corpus incrustado en una cadena de "+
			"interfaz: %q", camuflado)
	}

	// Y el otro lado del control: una cadena de interfaz normal no puede
	// dispararlo.
	for _, inocente := range []string{
		"Guardar los cambios del alcance antes de continuar con la entrevista",
		"No hay ningún paquete normativo instalado en esta instancia todavía",
		"Demasiadas peticiones. Espere un momento y vuelva a intentarlo, por favor",
	} {
		if de, hay := delataTextoDelCorpus(trozos, inocente); hay {
			t.Errorf("falso positivo del contraste sobre %q (dice que sale de %s)",
				inocente, de)
		}
	}
}

// delataTextoDelCorpus dice si un valor contiene un trozo de texto del corpus, y
// de que paquete sale.
func delataTextoDelCorpus(trozos map[string]string, valor string) (string, bool) {
	palabras := strings.Fields(normalizar(valor))
	for i := 0; i+ventanaDeTrozo <= len(palabras); i++ {
		t := strings.Join(palabras[i:i+ventanaDeTrozo], " ")
		if de, hay := trozos[t]; hay {
			return de + " (trozo: " + t + ")", true
		}
	}
	return "", false
}

// trozosDelCorpus indexa el texto que viene de los paquetes instalados, por
// trozos de ventanaDeTrozo palabras normalizadas.
//
// Indexa el texto legal Y el resto de contenido que el paquete aporta a la
// pantalla (preguntas, ayudas, titulos de plantilla), porque la regla de
// nucleo/pantalla es la misma para todo: el contenido del corpus viaja tal
// cual, en el idioma del paquete, y no pasa por el catalogo.
func trozosDelCorpus(t *testing.T) map[string]string {
	t.Helper()
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatalf("no se puede cargar el corpus para contrastar contra el: %v", err)
	}
	trozos := map[string]string{}
	indexar := func(texto, urn string) {
		palabras := strings.Fields(normalizar(texto))
		for i := 0; i+ventanaDeTrozo <= len(palabras); i++ {
			trozos[strings.Join(palabras[i:i+ventanaDeTrozo], " ")] = urn
		}
	}
	for _, p := range ps {
		for _, o := range p.Obligaciones {
			indexar(o.TextoLegal, p.URN)
		}
		for _, q := range p.Preguntas {
			indexar(q.Texto, p.URN)
			indexar(q.Ayuda, p.URN)
		}
		for _, e := range p.Entidades {
			indexar(e.Descripcion, p.URN)
			for _, a := range e.Atributos {
				indexar(a.Ayuda, p.URN)
			}
		}
		for _, pl := range p.Plantillas {
			indexar(pl.Titulo, p.URN)
		}
	}
	if len(trozos) < 100 {
		t.Fatalf("el corpus instalado solo da %d trozos comparables. Con tan poco texto "+
			"este contraste no prueba nada: o los paquetes se han vaciado, o el indexado "+
			"ha dejado de indexar", len(trozos))
	}
	return trozos
}

// fragmentoRealDelCorpus devuelve un trozo literal de texto legal de verdad,
// leido del corpus en tiempo de ejecucion.
//
// Se saca del corpus y no se escribe aqui a mano por dos razones: porque un
// fragmento cableado envejece en cuanto el paquete cambia, y porque escribir el
// identificador de una norma en un test de adaptadores/ es justo lo que
// prohibe el invariante 2.
func fragmentoRealDelCorpus(t *testing.T) string {
	t.Helper()
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range ps {
		for _, o := range p.Obligaciones {
			palabras := strings.Fields(o.TextoLegal)
			if len(palabras) >= ventanaDeTrozo+4 {
				return strings.Join(palabras[:ventanaDeTrozo+4], " ")
			}
		}
	}
	t.Fatal("no hay ni un texto legal en el corpus instalado del que sacar un fragmento: " +
		"el control negativo del contraste no se puede construir")
	return ""
}

// objetoJSON compone un catalogo de una sola entrada sin depender del propio
// codificador que se esta probando.
func objetoJSON(clave, valor string) string {
	esc := func(s string) string {
		s = strings.ReplaceAll(s, `\`, `\\`)
		s = strings.ReplaceAll(s, `"`, `\"`)
		s = strings.ReplaceAll(s, "\n", `\n`)
		s = strings.ReplaceAll(s, "\t", `\t`)
		return s
	}
	return `{"` + esc(clave) + `": "` + esc(valor) + `"}`
}
