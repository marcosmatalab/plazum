package plazum

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// LOS ITEMS DE VIGILANCIA, atados al corpus en LAS DOS DIRECCIONES.
//
// POR QUE ESTE FICHERO EXISTE, y lo pago el proyecto en un dia. El 27-08-2026 el
// paquete ai-act afirmaba que dos fechas del AI Act "NO VINCULAN porque el
// omnibus digital no esta publicado en el DOUE". Llevaban publicadas treinta y
// cuatro dias (Reglamento (UE) 2026/1744, de 8 de julio de 2026). Lo encontro
// una revision, no una puerta.
//
// Una lectura divergente es una apuesta sobre algo que todavia no ha pasado. El
// campo `espera` la ata al hecho de fuera que la puede matar, y este test es lo
// que impide que esa atadura se quede colgando.
//
// LAS DOS DIRECCIONES NO SON LA MISMA, y la que se olvida es la segunda:
//
//	adelante  toda lectura con `espera` nombra un item QUE EXISTE, y ese item la
//	          reconoce entre las suyas.
//	atras     todo item nombra lecturas QUE EXISTEN. Un item huerfano (que
//	          apunta a una obligacion borrada, o a una alternativa renombrada)
//	          parece vigilancia y no vigila nada, que es exactamente la familia
//	          de guardas que este repositorio ya ha visto fallar catorce veces.
//
// Y NO SE COMPARA NINGUNA FECHA CON EL RELOJ DE PARED. Los items de clase
// `fecha` los mira el workflow con horario, no un test: un test que compara con
// el reloj es una bomba con la mecha ya encendida (docs/pendientes.md, decima y
// decimotercera de la familia).
//
// Este fichero SI nombra normas reales, y por eso vive en la raiz: es donde el
// codigo se encuentra con paquetes/, y donde TestNingunaNormaCableada no llega.

const dirItemsVigilancia = "vigilancia/items"

// MinimoDeItemsDeVigilancia es el suelo. Sin el, el dia que el directorio se
// mueva o se vacie este test recorreria la nada y saldria verde, que es la forma
// exacta en la que una puerta deja de guardar.
const MinimoDeItemsDeVigilancia = 2

type enlaceDeVigilancia struct {
	Paquete    string `json:"paquete"`
	Obligacion string `json:"obligacion"`
	Lectura    string `json:"lectura"`
}

type itemDeVigilancia struct {
	ID     string `json:"id"`
	Nombre string `json:"nombre"`
	Clase  string `json:"clase"`
	PorQue string `json:"porque"`
	Vigila []struct {
		Jurisdiccion  string `json:"jurisdiccion"`
		Identificador string `json:"identificador"`
		Que           string `json:"que"`
	} `json:"vigila"`
	Cuando       string               `json:"cuando,omitempty"`
	VentanaDias  int                  `json:"ventana_de_aviso_dias,omitempty"`
	CuelgaDe     []enlaceDeVigilancia `json:"cuelga_de"`
	AlDispararse string               `json:"al_dispararse"`
	Ocurrido     *struct {
		Cuando string `json:"cuando"`
		Que    string `json:"que"`
		Coste  string `json:"coste"`
	} `json:"ocurrido,omitempty"`
}

func itemsDeVigilancia(t *testing.T) map[string]itemDeVigilancia {
	t.Helper()
	entradas, err := os.ReadDir(dirItemsVigilancia)
	if err != nil {
		t.Fatalf("no puedo leer %s: %v.\n"+
			"  Si el directorio se ha movido, este test estaria auditando el vacio y dando verde",
			dirItemsVigilancia, err)
	}
	out := map[string]itemDeVigilancia{}
	for _, e := range entradas {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		ruta := filepath.Join(dirItemsVigilancia, e.Name())
		b, err := os.ReadFile(ruta) // #nosec G304 -- ruta del propio repositorio
		if err != nil {
			t.Fatal(err)
		}
		var it itemDeVigilancia
		if err := json.Unmarshal(b, &it); err != nil {
			t.Fatalf("%s no es un item legible: %v", ruta, err)
		}
		// El nombre del fichero ES el identificador. Dos sitios con el mismo
		// dato es un sitio que se queda viejo en silencio.
		quiero := strings.TrimSuffix(e.Name(), ".json")
		if it.ID != quiero {
			t.Errorf("%s declara id %q y el fichero se llama %q. El nombre del fichero es el "+
				"identificador: si no casan, el issue que abra la vigilancia apuntara a un item "+
				"que nadie encuentra", ruta, it.ID, quiero)
		}
		out[it.ID] = it
	}
	if len(out) < MinimoDeItemsDeVigilancia {
		t.Fatalf("solo hay %d items de vigilancia y el suelo son %d. Si el recorte es "+
			"intencionado, baja MinimoDeItemsDeVigilancia EN ESTE MISMO COMMIT y di por que",
			len(out), MinimoDeItemsDeVigilancia)
	}
	return out
}

// lecturasDelCorpus indexa toda alternativa de vigencia por (paquete,
// obligacion, lectura), que es la identidad con la que la nombra un item.
func lecturasDelCorpus(t *testing.T) map[enlaceDeVigilancia]corpus.LecturaVigencia {
	t.Helper()
	ps, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatalf("cargar el corpus: %v", err)
	}
	out := map[enlaceDeVigilancia]corpus.LecturaVigencia{}
	for _, p := range ps {
		for _, l := range p.Vigencia.Alternativas {
			out[enlaceDeVigilancia{p.URN, "", l.ID}] = l
		}
		for _, o := range p.Obligaciones {
			for _, l := range o.Vigencia.Alternativas {
				out[enlaceDeVigilancia{p.URN, o.ID, l.ID}] = l
			}
		}
	}
	return out
}

func TestTodoItemDeVigilanciaCasaConSuCorpusEnLasDosDirecciones(t *testing.T) {
	items := itemsDeVigilancia(t)
	lecturas := lecturasDelCorpus(t)
	if len(lecturas) == 0 {
		t.Fatal("el corpus publicado no tiene ni una lectura divergente de vigencia. O se han " +
			"borrado, o este test ha dejado de encontrarlas: las dos cosas lo dejan sin nada " +
			"que comprobar")
	}

	// ADELANTE: toda lectura con `espera` nombra un item que existe, y ese item
	// la reconoce.
	conEspera := 0
	for enlace, l := range lecturas {
		if l.Espera == "" {
			continue // legal: hay lecturas que no dependen de ningun evento
		}
		conEspera++
		it, ok := items[l.Espera]
		if !ok {
			t.Errorf("la lectura %q de %s espera al item %q, que no existe en %s.\n"+
				"  Una divergencia que dice colgar de un hecho inexistente envejece sin que "+
				"nadie se entere, que es justo lo que este mecanismo existe para evitar.",
				enlace.Lectura, enlace.Obligacion, l.Espera, dirItemsVigilancia)
			continue
		}
		reconocida := false
		for _, c := range it.CuelgaDe {
			if c == enlace {
				reconocida = true
				break
			}
		}
		if !reconocida {
			t.Errorf("la lectura %q de %s espera al item %q, y ese item NO la lista en "+
				"`cuelga_de`.\n"+
				"  La atadura tiene que estar en los dos sitios: si solo esta en uno, el dia que "+
				"el item dispare nadie sabra a que lectura tocar.",
				enlace.Lectura, enlace.Obligacion, l.Espera)
		}
	}
	if conEspera == 0 {
		t.Error("ninguna lectura del corpus declara `espera`. El campo existe y no lo usa nadie, " +
			"asi que esta direccion del test no esta comprobando nada")
	}

	// ATRAS, que es la que se olvida: todo `cuelga_de` de todo item apunta a una
	// lectura que existe DE VERDAD.
	//
	// No se exige que esa lectura nombre a ESTE item: `espera` dice a que evento
	// espera lo PROXIMO que le puede pasar, y un item puede afectar a lecturas
	// que ya esperan a otro. Lo que no se admite es apuntar al vacio.
	for id, it := range items {
		if len(it.CuelgaDe) == 0 {
			t.Errorf("el item %q no cuelga de ninguna lectura. Un item que no ata nada al corpus "+
				"parece vigilancia y no vigila nada", id)
		}
		for _, c := range it.CuelgaDe {
			// Y LA TERCERA DIRECCION, que la pasada de mutacion destapo: una
			// lectura reclamada por un item TIENE que declarar `espera`.
			//
			// Sin esto se puede DESATAR una lectura en silencio. Quitarle el
			// `espera` no rompia nada: la direccion de adelante solo mira las
			// que lo tienen, y la de atras solo mira que la lectura exista. La
			// mutacion "quitar espera" salio verde, y por eso esta esto aqui.
			if l, ok := lecturas[c]; ok && l.Espera == "" {
				t.Errorf("el item %q reclama la lectura %q de %s, y esa lectura no declara "+
					"`espera`.\n"+
					"  Desatar una lectura es tan silencioso como borrar el item: la divergencia "+
					"se queda en el corpus, envejece, y ya nadie la esta esperando.",
					id, c.Lectura, c.Obligacion)
			}
			if _, ok := lecturas[c]; !ok {
				t.Errorf("el item %q dice colgar de la lectura %q de la obligacion %s del "+
					"paquete %s, y esa lectura no existe.\n"+
					"  Un item huerfano es la forma mas silenciosa de que la vigilancia deje de "+
					"vigilar: sigue en el directorio, sigue contando para el suelo, y no ata nada.",
					id, c.Lectura, c.Obligacion, c.Paquete)
			}
		}
	}
	t.Logf("%d items, %d lecturas divergentes, %d con `espera`", len(items), len(lecturas), conEspera)
}

// Lo que un item tiene que traer para que el issue que abra sirva de algo. Un
// item sin `porque` ni `al_dispararse` es una alarma sin instrucciones, y una
// alarma sin instrucciones se silencia a la tercera.
func TestTodoItemDeVigilanciaSeExplicaYDiceQueHacer(t *testing.T) {
	clases := map[string]bool{"publicacion": true, "fecha": true}
	for id, it := range itemsDeVigilancia(t) {
		if !clases[it.Clase] {
			t.Errorf("el item %q declara clase %q, que no es ni `publicacion` ni `fecha`. La "+
				"clase decide COMO se detecta, asi que sin ella el workflow no sabe que hacer "+
				"con el", id, it.Clase)
		}
		for campo, v := range map[string]string{
			"nombre":        it.Nombre,
			"porque":        it.PorQue,
			"al_dispararse": it.AlDispararse,
		} {
			if len(strings.TrimSpace(v)) < 20 {
				t.Errorf("el item %q tiene %s de %d caracteres. Eso no es una instruccion, es un "+
					"hueco relleno, y quien lea el issue dentro de un ano no sabra que hacer",
					id, campo, len(v))
			}
		}
		switch it.Clase {
		case "publicacion":
			if len(it.Vigila) == 0 {
				t.Errorf("el item %q es de clase `publicacion` y no dice que instrumento vigila, "+
					"asi que no hay nada que reejecutar", id)
			}
		case "fecha":
			if it.Cuando == "" {
				t.Errorf("el item %q es de clase `fecha` y no trae `cuando`", id)
			}
			if it.VentanaDias <= 0 {
				t.Errorf("el item %q es de clase `fecha` y no declara ventana de aviso. Sin "+
					"ventana, el aviso llega el mismo dia, que es cuando ya no sirve", id)
			}
		}
	}
}
