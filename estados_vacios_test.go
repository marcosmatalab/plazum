package plazum

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/catalogo"
	scimadaptador "github.com/marcosmatalab/plazum/adaptadores/scim"
	"github.com/marcosmatalab/plazum/adaptadores/secretos"
	"github.com/marcosmatalab/plazum/internal/modulo"
	"github.com/marcosmatalab/plazum/nucleo/accesos"
	"github.com/marcosmatalab/plazum/nucleo/acta"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/ledger"
	actaWeb "github.com/marcosmatalab/plazum/superficies/acta"
	"github.com/marcosmatalab/plazum/superficies/calendario"
	"github.com/marcosmatalab/plazum/superficies/camino"
	escaladoWeb "github.com/marcosmatalab/plazum/superficies/escalado"
	"github.com/marcosmatalab/plazum/superficies/pantallas"
	"github.com/marcosmatalab/plazum/superficies/scim"
	"github.com/marcosmatalab/plazum/superficies/serve"
	uarWeb "github.com/marcosmatalab/plazum/superficies/uar"
)

// LA PUERTA D11-b: TODO ESTADO VACIO TRAE SU SIGUIENTE PASO, Y LAS SUPERFICIES
// SE ENUMERAN DESDE EL ARBOL.
//
// # El agujero que cierra, con su cardinal
//
// Cinco pantallas tenian ya su test de estado vacio (acta, calendario, escalado,
// uar y la tabla de pantallas) y NINGUNA PUERTA LAS ENUMERABA. O sea: cada
// mitad pasaba la suya, y una pantalla nueva podia salir con un estado vacio sin
// verbo sin que nada se pusiera rojo. Es la familia exacta del trinquete de
// alcanzabilidad (piezas verificadas, juntas no) dentro de la casilla que la
// nombra.
//
// # Que se afirma aqui, en una frase
//
// Por cada paquete de superficies/ que sirve HTTP, o hay un estado vacio
// declarado con la ruta por la que se llega y el VERBO que trae, o hay una
// linea escrita que dice por que ese paquete no puede tener uno, Y UNA
// COMPROBACION QUE LO RECORRE.
//
// # Por que el verbo se busca DENTRO de <main> y no en la pagina
//
// Aqui nacio floja la primera version de esta puerta y merece contarse, con la
// medida exacta de cuanto. La exigencia era «el estado vacio tiene que traer un
// enlace o una orden», y se buscaba en el cuerpo entero de la respuesta. Eso
// deja SIN VIGILAR TODOS LOS VERBOS QUE SON ENLACE, que hoy son dos de los
// seis: el armazon compartido pinta la barra lateral con los seis pasos del
// camino EN TODAS LAS PAGINAS, asi que href="/alcance" esta siempre, en el
// estado vacio y en el lleno, en la pantalla que tiene verbo y en la que no.
//
// Se demostro quitando de tabla.html el unico enlace que el estado vacio pinta
// dentro de <main>: con el recorte, la puerta se pone roja y nombra la
// pantalla; buscando en el cuerpo entero, VERDE. Una puerta que se satisface
// con el marco de la ventana no mira la habitacion.
//
// Los verbos que son ORDEN DE TERMINAL no tenian ese agujero (una orden no sale
// en el armazon), y por eso el recorte no basta como unica defensa: hacen falta
// los dos, el recorte y el suelo que para si el recorte deja de casar.
//
// El verbo se exige DENTRO de <main>, que es el unico trozo que escribe la
// pantalla. Y hay un suelo: si la extraccion de <main> deja de casar, la puerta
// para en vez de recorrer el vacio.
//
// # Por que el censo vive en este fichero y no al lado del cableado
//
// A diferencia del trinquete de superficies, que vive en cmd/plazum porque ese
// es el fichero que MONTA, este censo tiene que construir las superficies con
// sus fuentes vacias, y cmd/plazum es `package main`: no se puede importar. La
// raiz es el unico paquete que puede a la vez recorrer superficies/ y llamar a
// los constructores de todas. Se paga con una identidad de mas que comprobar, y
// se comprueba: cada entrada demuestra que lo que construye es del paquete que
// dice, por la RUTA DE IMPORT sacada por reflexion, que es un dato que el censo
// no puede escribir.

// EstadoVacioDeSuperficie es que se hace con el estado vacio de una superficie
// que sirve HTTP. Vocabulario cerrado.
type EstadoVacioDeSuperficie uint8

const (
	// VacioSinDeclarar es el VALOR CERO Y ES INVALIDO. Una superficie que
	// llega aqui con el cero es una que nadie declaro, y el silencio es
	// exactamente el estado que esta puerta existe para impedir: es el que
	// dejaria salir una pantalla nueva con un callejon dentro.
	VacioSinDeclarar EstadoVacioDeSuperficie = iota
	// VacioAlcanzable: esta superficie se puede quedar sin datos y entonces
	// pinta una pagina. Exige ruta y verbo, y los dos se comprueban contra la
	// respuesta HTTP de verdad, en las DOS formas de la nada.
	VacioAlcanzable
	// VacioImposible: esta superficie NO puede quedarse en una pantalla vacia,
	// y eso se demuestra, no se afirma. Exige motivo Y una comprobacion que
	// recorra la rama: un descargo que ninguna entrada alcanza es un descargo
	// que no existe (M47).
	VacioImposible
)

func (e EstadoVacioDeSuperficie) String() string {
	switch e {
	case VacioAlcanzable:
		return "tiene estado vacio alcanzable"
	case VacioImposible:
		return "no puede quedarse en una pantalla vacia"
	default:
		// El cero se nombra por lo que es. Llamarlo «pendiente» suavizaria
		// justo el caso que esto existe para cazar.
		return "SIN DECLARAR (valor cero)"
	}
}

// FormaDeLaNada son las dos formas en que a una superficie no le llegan datos.
//
// INVARIANTE 8, Y AQUI NO ES DECORATIVO: `nil` sale de olvidarse el campo y
// vacio-presente de construir la fuente y que no tenga nada dentro. Un
// constructor puede tratarlas distinto sin que nadie lo note, y la que se olvida
// es siempre la permisiva. Las dos se recorren en cada superficie.
type FormaDeLaNada uint8

const (
	// NadaNil es la fuente ausente: el campo se quedo sin poner.
	NadaNil FormaDeLaNada = iota
	// NadaPresente es la fuente construida que no tiene nada que dar.
	NadaPresente
)

func (f FormaDeLaNada) String() string {
	if f == NadaNil {
		return "fuente nil"
	}
	return "fuente presente y vacia"
}

// VerboDelEstadoVacio es EL SIGUIENTE PASO CONCRETO que la pantalla vacia
// ofrece. Una de las dos formas, al menos.
//
// No vale una explicacion de por que no hay datos: eso es contexto, no verbo. La
// pregunta que contesta un verbo es «¿y ahora que hago yo?», y solo hay dos
// respuestas que una persona pueda ejecutar sin preguntarle a nadie: pulsar algo
// o teclear algo.
type VerboDelEstadoVacio struct {
	// Comando es la orden de terminal EXACTA que saca de este estado. Se
	// comprueba que sale literalmente dentro de <main>.
	Comando string
	// Enlace es la ruta a la que se puede pulsar desde dentro de <main>.
	//
	// TIENE QUE SER UNA RUTA QUE DECLARE camino.Canonico() (o la pantalla del
	// camino misma): asi no es una cadena que el censo se invente, sino un
	// destino que el camino guiado ya declara y que el trinquete de
	// alcanzabilidad comprueba que contesta en el servidor montado.
	Enlace string
	// Formulario es la ruta a la que ENVIA un formulario pintado dentro de
	// <main>. Se comprueba que sale como action="...".
	//
	// ES LA TERCERA FORMA DEL VERBO Y HACIA FALTA, y su ausencia no era un
	// descuido: las dos que habia (una orden de terminal y un enlace a otra
	// pantalla) son las dos que MANDAN A OTRO SITIO. Un estado vacio que se
	// resuelve SIN SALIR DE LA PANTALLA no se podia declarar, asi que la
	// unica forma de cerrar la puerta era escribir una orden que ya no hacia
	// falta. Una puerta que solo admite el verbo caro empuja a dejar el verbo
	// caro puesto, que es exactamente lo que estaba pasando con el TTFV.
	//
	// NO SE COMPRUEBA CONTRA camino.Canonico() como el Enlace, y el motivo es
	// que aqui no vale: los destinos del camino son PANTALLAS y esto es una
	// ruta mutante, que por diseno no esta en el camino. Lo que sostiene que
	// esta ruta existe de verdad es otra puerta, la que enumera los patrones
	// del enrutador y le manda una peticion mutante a cada uno.
	Formulario string
}

// EstadoVacioConcreto es UN estado vacio de una superficie: por donde se llega y
// que verbo trae.
type EstadoVacioConcreto struct {
	// Que nombra el estado, para que el fallo diga cual de ellos es.
	Que string
	// Ruta es la direccion que se pide, ya con el prefijo de montaje.
	Ruta string
	// Verbo es el siguiente paso. Al menos una de sus dos formas.
	Verbo VerboDelEstadoVacio
}

// DeclaracionDeEstadoVacio es lo que se afirma de un paquete de superficies/
// que sirve HTTP.
type DeclaracionDeEstadoVacio struct {
	Estado EstadoVacioDeSuperficie
	// Motivo es obligatorio en VacioImposible. Sin motivo, dentro de tres
	// meses nadie sabe si la ausencia es decision o deuda.
	Motivo string
	// Construir levanta la superficie SIN DATOS, en la forma de la nada que se
	// le pide. Obligatoria en VacioAlcanzable.
	//
	// Devuelve un http.Handler y no una ruta: la identidad por la que se
	// empareja con el arbol es la RUTA DE IMPORT del tipo devuelto, sacada por
	// reflexion. Ese dato no se puede escribir en el censo, que es justo por lo
	// que se usa (invariante 7).
	Construir func(t *testing.T, forma FormaDeLaNada) http.Handler
	// Vacios son los estados vacios de esta superficie. Al menos uno en
	// VacioAlcanzable.
	Vacios []EstadoVacioConcreto
	// Demostrar RECORRE la rama de VacioImposible. Obligatoria ahi: es el
	// control positivo del descargo. Falla el test si la superficie resulta
	// tener una pantalla vacia despues de todo.
	Demostrar func(t *testing.T)
}

// EstadosVaciosDeLasSuperficies es el censo, por el nombre del paquete bajo
// superficies/.
//
// SE CRUZA CON EL ARBOL EN LOS DOS SENTIDOS. Un paquete que sirva HTTP y no
// salga aqui rompe la puerta; una entrada de aqui que ya no exista, o que haya
// dejado de servir HTTP, tambien.
var EstadosVaciosDeLasSuperficies = map[string]DeclaracionDeEstadoVacio{
	"pantallas": {
		Estado:    VacioAlcanzable,
		Construir: construirPantallasVacias,
		Vacios: []EstadoVacioConcreto{
			{
				Que:   "la tabla de obligaciones sin corpus instalado",
				Ruta:  "/controles",
				Verbo: VerboDelEstadoVacio{Enlace: "/alcance"},
			},
			{
				Que:  "el panel de inicio sin corpus instalado",
				Ruta: "/hoy",
				// El panel no manda a la terminal: manda a responder la
				// entrevista, que es el paso 1 del camino.
				Verbo: VerboDelEstadoVacio{Enlace: "/alcance"},
			},
		},
	},
	"calendario": {
		Estado:    VacioAlcanzable,
		Construir: construirCalendarioVacio,
		Vacios: []EstadoVacioConcreto{{
			Que:   "sin alcance del que derivar fechas",
			Ruta:  calendario.BasePorDefecto + "/",
			Verbo: VerboDelEstadoVacio{Comando: "plazum serve --alcance"},
		}},
	},
	"escalado": {
		Estado:    VacioAlcanzable,
		Construir: construirEscaladoVacio,
		Vacios: []EstadoVacioConcreto{{
			Que:   "sin alcance del que planificar avisos",
			Ruta:  escaladoWeb.BasePorDefecto + "/",
			Verbo: VerboDelEstadoVacio{Comando: "plazum serve --alcance"},
		}},
	},
	"uar": {
		Estado:    VacioAlcanzable,
		Construir: construirUARVacia,
		Vacios: []EstadoVacioConcreto{{
			Que:  "sin campana de revision de accesos configurada",
			Ruta: "/uar/",
			// EL VERBO ERA `plazum serve --accesos-fichero` Y ES UN FORMULARIO.
			// Las dos ordenes que pedia este estado vacio son 3m0s del TTFV del
			// camino guiado, y no eran evitables mientras el censo solo se
			// pudiera subir por el terminal. Ahora se sube aqui.
			Verbo: VerboDelEstadoVacio{Formulario: "/uar/abrir"},
		}},
	},
	"acta": {
		Estado:    VacioAlcanzable,
		Construir: construirActaVacia,
		Vacios: []EstadoVacioConcreto{{
			Que:  "sin ninguna acta que componer",
			Ruta: "/acta/",
			// EL VERBO QUE FALTABA. Ver el commit que trae esta puerta: este
			// estado vacio explicaba de que se compone un acta y no decia como
			// tener una, asi que era un callejon con buena prosa.
			Verbo: VerboDelEstadoVacio{Comando: "plazum serve --acta-organizacion"},
		}},
	},

	// LAS TRES QUE NO PUEDEN QUEDARSE EN UNA PANTALLA VACIA, cada una por una
	// razon distinta, y las tres con su rama recorrida.
	"camino": {
		Estado: VacioImposible,
		Motivo: "el camino guiado no tiene fuente de datos: sus pasos son la lista canonica, " +
			"que es codigo. Un camino vacio no es un estado, es una construccion invalida, y " +
			"el constructor se niega con ErrSinPasos en las DOS formas de la nada. Darle un " +
			"estado vacio seria darle una pantalla que dice que plazum no tiene camino, que " +
			"es la mentira mas cara que esta pieza puede contar.",
		Demostrar: demostrarQueElCaminoNoPuedeEstarVacio,
	},
	"serve": {
		Estado: VacioImposible,
		Motivo: "es el servidor, no una pantalla de datos: cuelga a las demas y su unica " +
			"pagina propia es la puerta de entrada. Sin sesion, la raiz no pinta un estado " +
			"vacio, redirige a /entrar, asi que no hay pagina sin datos a la que ponerle un " +
			"verbo. Su hueco propio esta anotado en docs/hallazgos-pantallas.md.",
		Demostrar: demostrarQueServeNoPintaPantallaVacia,
	},
	"scim": {
		Estado: VacioImposible,
		Motivo: "no es una pantalla: es el endpoint de aprovisionamiento del IdP y contesta " +
			"JSON a una maquina. Ademas no se puede construir sin token (el constructor se " +
			"niega), asi que no existe una instancia sin configurar a la que llegar con un " +
			"navegador y encontrarla vacia.",
		Demostrar: demostrarQueScimNoSeConstruyeSinCredencial,
	},
}

// ---------------------------------------------------------------------------
// La enumeracion desde el arbol. Es la mitad que nadie puede olvidarse de
// actualizar, y por eso es la que manda.
// ---------------------------------------------------------------------------

// superficiesQueSirvenHTTP recorre superficies/ y dice cuales declaran
// ServeHTTP(http.ResponseWriter, *http.Request) fuera de un fichero de test.
//
// Es el interfaz http.Handler, o sea la unica definicion que no depende de que
// alguien se acuerde de anotarse. Se lee el AST y no se usa reflexion porque un
// paquete que nadie importa no existe para la reflexion, y ese es exactamente el
// paquete que hay que cazar.
func superficiesQueSirvenHTTP(t *testing.T) (todas []string, sirven map[string]string) {
	t.Helper()
	raiz := "superficies"
	entradas, err := os.ReadDir(raiz)
	if err != nil {
		t.Fatalf("no puedo leer %s: %v", raiz, err)
	}
	sirven = map[string]string{}
	for _, e := range entradas {
		if !e.IsDir() {
			continue // superficies/doc.go
		}
		todas = append(todas, e.Name())
		if f := ficheroConServeHTTP(t, filepath.Join(raiz, e.Name())); f != "" {
			sirven[e.Name()] = f
		}
	}
	sort.Strings(todas)
	return todas, sirven
}

func ficheroConServeHTTP(t *testing.T, dir string) string {
	t.Helper()
	entradas, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("no puedo leer %s: %v", dir, err)
	}
	fset := token.NewFileSet()
	for _, e := range entradas {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		ruta := filepath.Join(dir, n)
		f, err := parser.ParseFile(fset, ruta, nil, 0)
		if err != nil {
			t.Fatalf("%s: %v", ruta, err)
		}
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Name.Name != "ServeHTTP" || fd.Recv == nil {
				continue
			}
			if firmaDeHandler(fd) {
				return ruta
			}
		}
	}
	return ""
}

// firmaDeHandler mira los DOS parametros. El nombre del metodo no basta: un
// `ServeHTTP()` cualquiera no es un handler, y el interfaz es la firma.
func firmaDeHandler(fd *ast.FuncDecl) bool {
	if fd.Type.Params == nil || len(fd.Type.Params.List) != 2 {
		return false
	}
	sel, ok := fd.Type.Params.List[0].Type.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "ResponseWriter" {
		return false
	}
	estrella, ok := fd.Type.Params.List[1].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel2, ok := estrella.X.(*ast.SelectorExpr)
	return ok && sel2.Sel.Name == "Request"
}

// ---------------------------------------------------------------------------
// LA PUERTA, primera mitad: el censo cuadra con el arbol en los dos sentidos.
// ---------------------------------------------------------------------------

func TestTodaSuperficieDeclaraQueHaceConSuEstadoVacio(t *testing.T) {
	todas, sirven := superficiesQueSirvenHTTP(t)
	// Los suelos protegen del fallo silencioso de la propia puerta: si el
	// recorrido o el detector dejan de casar, los mapas salen vacios y todo lo
	// de abajo se pone verde recorriendo la nada.
	if len(todas) < 8 {
		t.Fatalf("bajo superficies/ salen %d paquetes (%v) y hoy hay al menos 8: o han "+
			"desaparecido, o este recorrido esta midiendo el vacio", len(todas), todas)
	}
	if len(sirven) < 6 {
		t.Fatalf("solo %d de %d paquetes de superficies/ declaran ServeHTTP (%v) y hoy son "+
			"al menos 6. El detector de AST ha dejado de casar", len(sirven), len(todas),
			ordenadasDelArbol(sirven))
	}

	// SENTIDO 1: todo paquete que sirva HTTP esta declarado, y no con el cero.
	for _, nombre := range ordenadasDelArbol(sirven) {
		d, hay := EstadosVaciosDeLasSuperficies[nombre]
		if !hay {
			t.Errorf("superficies/%s sirve HTTP (%s) y NO sale en "+
				"EstadosVaciosDeLasSuperficies.\n"+
				"  Una pantalla nueva puede salir con un estado vacio sin verbo y nada se "+
				"pone rojo: eso es el callejon que la puerta D11-b existe para impedir.\n"+
				"  Arreglo: declararla en estados_vacios_test.go, con su ruta y su verbo, o "+
				"como VacioImposible con su motivo y su demostracion.",
				nombre, sirven[nombre])
			continue
		}
		if d.Estado == VacioSinDeclarar {
			t.Errorf("superficies/%s esta en el censo con el VALOR CERO (%s). El cero no es "+
				"un estado, es el olvido: escribe cual de los dos es", nombre, d.Estado)
		}
	}

	// SENTIDO 2: toda entrada del censo existe y sigue sirviendo HTTP. Sin
	// esta mitad, el censo envejece hasta ser una lista de lo que hubo.
	for _, nombre := range ordenadasDelCenso() {
		if _, hay := sirven[nombre]; hay {
			continue
		}
		if hayDirectorio(t, filepath.Join("superficies", nombre)) {
			t.Errorf("el censo declara superficies/%s y ese paquete ya NO sirve HTTP. O dejo "+
				"de ser una superficie web y sale del censo, o alguien se llevo su ServeHTTP "+
				"por delante", nombre)
			continue
		}
		t.Errorf("el censo declara superficies/%s y ese paquete no existe en el arbol", nombre)
	}

	// SENTIDO 3: cada estado exige lo suyo, y lo que exige no es decorativo.
	for _, nombre := range ordenadasDelCenso() {
		d := EstadosVaciosDeLasSuperficies[nombre]
		switch d.Estado {
		case VacioAlcanzable:
			if d.Construir == nil {
				t.Errorf("superficies/%s dice tener estado vacio y no dice como levantarla "+
					"sin datos: la declaracion no se puede comprobar", nombre)
			}
			if len(d.Vacios) == 0 {
				t.Errorf("superficies/%s dice tener estado vacio y no declara ni uno. Una "+
					"declaracion sin ruta afirma algo que nadie puede pedirle al servidor",
					nombre)
			}
			if d.Motivo != "" {
				t.Errorf("superficies/%s tiene estado vacio y ademas trae Motivo (%q). Ese "+
					"campo es el de las que NO pueden tenerlo", nombre, d.Motivo)
			}
			if d.Demostrar != nil {
				t.Errorf("superficies/%s tiene estado vacio y ademas trae Demostrar. Esa "+
					"funcion es el control positivo de la rama contraria", nombre)
			}
			for _, v := range d.Vacios {
				if strings.TrimSpace(v.Que) == "" || strings.TrimSpace(v.Ruta) == "" {
					t.Errorf("superficies/%s declara un estado vacio sin nombre o sin ruta: "+
						"%+v", nombre, v)
				}
				// EL VERBO ES LO QUE SE EXIGE, y su ausencia es el callejon.
				if v.Verbo.Comando == "" && v.Verbo.Enlace == "" && v.Verbo.Formulario == "" {
					t.Errorf("superficies/%s: el estado vacio %q no declara VERBO.\n"+
						"  Una pantalla vacia sin verbo es un callejon: explica por que no "+
						"hay datos y no dice que puede hacer quien la esta mirando.\n"+
						"  Arreglo: darle la orden exacta que saca de este estado, el "+
						"enlace al paso del camino que lo produce, o el formulario que lo "+
						"resuelve sin salir de la pantalla.", nombre, v.Que)
					continue
				}
				// Y UN ENLACE NO SE INVENTA: tiene que ser un destino que el
				// camino guiado ya declare. Con una cadena libre, el censo
				// podria escribir cualquier ruta y el test la buscaria en la
				// pagina sin que nadie comprobara que lleva a alguna parte.
				if v.Verbo.Enlace != "" && !esDestinoDelCamino(v.Verbo.Enlace) {
					t.Errorf("superficies/%s: el estado vacio %q enlaza a %q y esa ruta no la "+
						"declara camino.Canonico() ni es la pantalla del camino. Un verbo "+
						"que apunta a una direccion que el producto no monta es peor que "+
						"ninguno", nombre, v.Que, v.Verbo.Enlace)
				}
			}
		case VacioImposible:
			if strings.TrimSpace(d.Motivo) == "" {
				t.Errorf("superficies/%s se declara %q y no dice por que. Un hueco sin motivo "+
					"se lee como deuda y como decision a la vez", nombre, d.Estado)
			}
			if d.Demostrar == nil {
				t.Errorf("superficies/%s se declara %q y no trae quien lo demuestre.\n"+
					"  Una rama de descargo que ninguna comprobacion recorre es una rama que "+
					"no existe, y la mutacion la deja verde porque no hay nada que romper "+
					"(M47).", nombre, d.Estado)
			}
			if len(d.Vacios) != 0 || d.Construir != nil {
				t.Errorf("superficies/%s se declara %q y ademas trae estados vacios o "+
					"constructor. O puede quedarse vacia o no puede", nombre, d.Estado)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// LA PUERTA, segunda mitad: el verbo sale EN LA RESPUESTA, no en el fichero.
// ---------------------------------------------------------------------------

// TestTodoEstadoVacioTraeSuVerboEnLaRespuesta es la puerta de verdad.
//
// POR QUE HACE FALTA APARTE de la de arriba: aquella lee texto y comprueba que
// lo escrito es coherente consigo mismo. Nada de eso impide declarar un verbo
// que la pantalla no pinta, que es el error mas comodo de cometer y el que deja
// el callejon en pie con una declaracion tranquilizadora encima.
//
// LAS DOS FORMAS DE LA NADA se recorren siempre (invariante 8): un constructor
// puede tratar `nil` y vacio-presente distinto sin que nadie lo note.
func TestTodoEstadoVacioTraeSuVerboEnLaRespuesta(t *testing.T) {
	mod, err := modulo.Ruta()
	if err != nil {
		t.Fatalf("leyendo la ruta del modulo de go.mod: %v", err)
	}
	prefijo := mod + "/superficies/"

	comprobados := 0
	for _, nombre := range ordenadasDelCenso() {
		d := EstadosVaciosDeLasSuperficies[nombre]
		if d.Estado != VacioAlcanzable || d.Construir == nil {
			continue
		}
		for _, forma := range []FormaDeLaNada{NadaNil, NadaPresente} {
			h := d.Construir(t, forma)
			if h == nil {
				t.Errorf("superficies/%s no se ha podido levantar con %s", nombre, forma)
				continue
			}
			// LA IDENTIDAD POR LA QUE CASA, y no es el nombre del paquete
			// escrito en la clave del mapa: es la RUTA DE IMPORT del tipo que
			// devuelve el constructor, sacada por reflexion. Una entrada del
			// censo no puede fabricar ese dato, asi que no puede heredar el
			// estado vacio de otra superficie escribiendo su nombre. Es el
			// fallo exacto que tumbo la primera version del trinquete de
			// alcanzabilidad (invariante 7).
			if p := paquetePorReflexion(h); p != prefijo+nombre {
				t.Errorf("la entrada %q del censo construye un handler del paquete %q. O la "+
					"clave miente, o esta heredando la comprobacion de otra superficie",
					nombre, p)
				continue
			}
			for _, v := range d.Vacios {
				comprobados++
				comprobarUnEstadoVacio(t, nombre, forma, h, v)
			}
		}
	}
	// El suelo: si el censo se queda sin ninguna superficie con estado vacio,
	// esta puerta seguiria verde recorriendo la nada.
	if comprobados < 10 {
		t.Fatalf("solo se han comprobado %d estados vacios (5 pantallas por 2 formas de la "+
			"nada son 12 hoy). Esta puerta esta midiendo el vacio", comprobados)
	}
}

func comprobarUnEstadoVacio(t *testing.T, nombre string, forma FormaDeLaNada,
	h http.Handler, v EstadoVacioConcreto) {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, v.Ruta, nil))
	// UN ESTADO VACIO NO ES UN ERROR. Una pantalla que desaparece cuando no
	// hay datos deja al operador sin saber que existia.
	if w.Code != http.StatusOK {
		t.Errorf("superficies/%s, %s (%s): GET %s contesta %d.\n"+
			"  Un estado vacio no es un error ni un 404: es un estado, y tiene que pintarse",
			nombre, v.Que, forma, v.Ruta, w.Code)
		return
	}
	cuerpo := w.Body.String()
	principal := contenidoPrincipal(t, nombre, v, cuerpo)

	if c := v.Verbo.Comando; c != "" && !strings.Contains(principal, c) {
		t.Errorf("superficies/%s, %s (%s): el estado vacio no trae su VERBO.\n"+
			"  Se esperaba la orden %q dentro de <main> y no esta.\n"+
			"  Una pantalla vacia que explica por que no hay datos y no dice que hacer es un "+
			"callejon (D11-b).\n--- <main> ---\n%s", nombre, v.Que, forma, c, recortar(principal))
	}
	if f := v.Verbo.Formulario; f != "" && !strings.Contains(principal, `action="`+f+`"`) {
		t.Errorf("superficies/%s, %s (%s): el estado vacio no trae su VERBO.\n"+
			"  Se esperaba un formulario que envie a %q dentro de <main> y no esta.\n"+
			"  Este es el verbo que NO manda al terminal ni a otra pantalla, asi que si "+
			"desaparece lo que vuelve no es un hueco: es una orden de terminal, y cada "+
			"una son 1m30s del TTFV.\n--- <main> ---\n%s",
			nombre, v.Que, forma, f, recortar(principal))
	}
	if e := v.Verbo.Enlace; e != "" && !strings.Contains(principal, `href="`+e) {
		t.Errorf("superficies/%s, %s (%s): el estado vacio no trae su VERBO.\n"+
			"  Se esperaba un enlace a %q dentro de <main> y no esta. Ojo: la barra lateral "+
			"del armazon enlaza a los seis pasos en TODAS las paginas, asi que un enlace "+
			"fuera de <main> no es el verbo de esta pantalla, es el marco de la ventana.\n"+
			"--- <main> ---\n%s", nombre, v.Que, forma, e, recortar(principal))
	}
}

// contenidoPrincipal recorta lo que escribe la pantalla, o sea <main>...</main>.
//
// Sin este recorte la puerta nace floja y da verde sobre todo: el armazon
// compartido pinta la barra lateral con los seis pasos del camino en todas las
// paginas. Si la extraccion deja de casar, se PARA en vez de dar por bueno el
// cuerpo entero, que seria volver a la version floja sin enterarse.
func contenidoPrincipal(t *testing.T, nombre string, v EstadoVacioConcreto, cuerpo string) string {
	t.Helper()
	i := strings.Index(cuerpo, "<main ")
	j := strings.Index(cuerpo, "</main>")
	if i < 0 || j < i {
		t.Fatalf("superficies/%s, %s: la respuesta de %s no trae un <main>...</main> del que "+
			"recortar el contenido.\n"+
			"  Esta puerta busca el verbo DENTRO de <main> a proposito: en el cuerpo entero "+
			"lo encuentra siempre, porque la barra lateral enlaza a los seis pasos en todas "+
			"las paginas. Sin el recorte, la puerta da verde sobre cualquier cosa.\n"+
			"--- respuesta ---\n%s", nombre, v.Que, v.Ruta, recortar(cuerpo))
	}
	return cuerpo[i:j]
}

// esDestinoDelCamino dice si una ruta la declara el camino guiado.
func esDestinoDelCamino(ruta string) bool {
	if ruta == camino.BasePorDefecto || ruta == camino.BasePorDefecto+"/" {
		return true
	}
	for _, p := range camino.Canonico() {
		if p.EsPantalla() && p.Ruta == ruta {
			return true
		}
	}
	return false
}

// paquetePorReflexion da la ruta de import del paquete que declara el tipo del
// handler. Es el campo por el que se empareja, y no se puede escribir.
func paquetePorReflexion(h http.Handler) string {
	tipo := reflect.TypeOf(h)
	for tipo != nil && tipo.Kind() == reflect.Ptr {
		tipo = tipo.Elem()
	}
	if tipo == nil {
		return ""
	}
	return tipo.PkgPath()
}

// ---------------------------------------------------------------------------
// CONTROL NEGATIVO DEL DETECTOR. Todo cuelga de el, y tiene dos formas de
// mentir en silencio: decir que si a todo y decir que no a todo. La segunda es
// la que se lee como verde, y es la que deja pasar superficies nuevas.
// ---------------------------------------------------------------------------

func TestElDetectorDeSuperficiesDeEstadosVaciosDistingueLasDosRespuestas(t *testing.T) {
	todas, sirven := superficiesQueSirvenHTTP(t)
	for _, n := range []string{"pantallas", "acta", "uar", "calendario", "escalado"} {
		if _, hay := sirven[n]; !hay {
			t.Errorf("el detector dice que superficies/%s no sirve HTTP, y es una pantalla "+
				"del producto: esta roto en la direccion que deja pasar superficies nuevas "+
				"sin declarar", n)
		}
	}
	sinHTTP := 0
	for _, n := range todas {
		if _, hay := sirven[n]; !hay {
			sinHTTP++
		}
	}
	if sinHTTP == 0 {
		t.Error("el detector dice que TODOS los paquetes de superficies/ sirven HTTP. Hoy hay " +
			"al menos uno que no (export escribe ficheros para un SIEM), asi que esta " +
			"diciendo que si a todo")
	}
}

// ---------------------------------------------------------------------------
// LOS TRES CONTROLES POSITIVOS DE LA RAMA «no puede quedarse vacia».
//
// Se ejecutan desde una puerta que los enumera del censo, no uno a uno: asi, el
// dia que alguien anada una cuarta superficie imposible sin demostracion, la
// puerta de arriba se pone roja y esta la recorre en cuanto la escriba.
// ---------------------------------------------------------------------------

func TestLasSuperficiesSinEstadoVacioLoDemuestran(t *testing.T) {
	recorridas := 0
	for _, nombre := range ordenadasDelCenso() {
		d := EstadosVaciosDeLasSuperficies[nombre]
		if d.Estado != VacioImposible || d.Demostrar == nil {
			continue
		}
		recorridas++
		t.Run(nombre, d.Demostrar)
	}
	if recorridas < 3 {
		t.Fatalf("solo se han recorrido %d ramas de «no puede quedarse vacia» y hoy hay 3 "+
			"(camino, serve, scim). Un descargo que nadie recorre no existe", recorridas)
	}
}

// El camino guiado no puede estar vacio: el constructor se niega, en las DOS
// formas de la nada.
func demostrarQueElCaminoNoPuedeEstarVacio(t *testing.T) {
	cat := catalogoReal(t)
	casos := []struct {
		forma FormaDeLaNada
		pasos []camino.Paso
	}{
		{NadaNil, nil},
		{NadaPresente, []camino.Paso{}},
	}
	for _, c := range casos {
		_, err := camino.Nuevo(camino.Opciones{Pasos: c.pasos, Catalogo: cat,
			Base: camino.BasePorDefecto})
		if err == nil {
			t.Errorf("con %s el camino se ha construido igual, asi que SI puede pintar una "+
				"pantalla vacia y el censo dice que no. Esa pantalla diria que plazum no "+
				"tiene camino", c.forma)
		}
	}
}

// serve no pinta una pantalla vacia: sin sesion, la raiz manda a la puerta.
func demostrarQueServeNoPintaPantallaVacia(t *testing.T) {
	srv, err := serve.Nuevo(serve.Config{})
	if err != nil {
		t.Fatalf("construyendo el servidor pelado: %v", err)
	}
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code == http.StatusOK {
		t.Errorf("la raiz de un servidor sin pantallas montadas contesta 200: eso ES una "+
			"pagina sin datos, y entonces le toca traer verbo como a las demas.\n%s",
			recortar(w.Body.String()))
	}
	if destino := w.Header().Get("Location"); !strings.Contains(destino, "/entrar") {
		t.Errorf("la raiz sin sesion contesta %d hacia %q, y se esperaba la puerta de "+
			"entrada. Si ha dejado de redirigir, esta superficie tiene una pagina propia y "+
			"el censo se ha quedado viejo", w.Code, destino)
	}
}

// scim no se puede construir sin credencial, asi que no hay instancia vacia.
func demostrarQueScimNoSeConstruyeSinCredencial(t *testing.T) {
	dir, err := scimadaptador.NuevoDirectorio(secretos.Nuevo())
	if err != nil {
		t.Fatalf("construyendo el directorio del aprovisionamiento: %v", err)
	}
	for _, forma := range []FormaDeLaNada{NadaNil, NadaPresente} {
		op := scim.Opciones{}
		if forma == NadaPresente {
			op.Token = "" // presente y vacio: la otra forma de la nada
		}
		if _, err := scim.Nuevo(dir, op); err == nil {
			t.Errorf("scim se ha construido con %s, o sea sin token: existiria un endpoint de "+
				"altas y bajas abierto, y ademas una instancia vacia a la que llegar", forma)
		}
	}
}

// ---------------------------------------------------------------------------
// Los constructores de las superficies vacias. Uno por paquete, y cada uno
// recorre las dos formas de la nada.
// ---------------------------------------------------------------------------

func catalogoReal(t *testing.T) *catalogo.Catalogo {
	t.Helper()
	c, err := catalogo.Nuevo()
	if err != nil {
		t.Fatalf("cargando el catalogo: %v", err)
	}
	return c
}

func construirPantallasVacias(t *testing.T, forma FormaDeLaNada) http.Handler {
	t.Helper()
	var ps []*corpus.Paquete
	if forma == NadaPresente {
		ps = []*corpus.Paquete{} // vacio-presente
	}
	s, err := pantallas.Nuevo(pantallas.Opciones{
		Paquetes: ps, Catalogo: catalogoReal(t),
		CaminoRuta: camino.BasePorDefecto + "/", CaminoClave: camino.ClaveTitulo,
		Pasos: camino.Canonico(),
	})
	if err != nil {
		t.Fatalf("construyendo las pantallas sin corpus (%s): %v", forma, err)
	}
	return s
}

type calendarioVacio struct{}

func (calendarioVacio) Actual() (calendario.Derivado, bool, error) {
	return calendario.Derivado{}, false, nil
}

func construirCalendarioVacio(t *testing.T, forma FormaDeLaNada) http.Handler {
	t.Helper()
	var f calendario.Fuente
	if forma == NadaPresente {
		f = calendarioVacio{}
	}
	s, err := calendario.NuevaPantalla(calendario.OpcionesPantalla{
		Fuente: f, Catalogo: catalogoReal(t), Base: calendario.BasePorDefecto,
		CaminoRuta: camino.BasePorDefecto + "/", CaminoClave: camino.ClaveTitulo,
		Pasos: camino.Canonico(),
	})
	if err != nil {
		t.Fatalf("construyendo el calendario sin alcance (%s): %v", forma, err)
	}
	return s
}

type escaladoVacio struct{}

func (escaladoVacio) EnSeco() (escaladoWeb.Plan, bool, error) {
	return escaladoWeb.Plan{}, false, nil
}

func construirEscaladoVacio(t *testing.T, forma FormaDeLaNada) http.Handler {
	t.Helper()
	var f escaladoWeb.Fuente
	if forma == NadaPresente {
		f = escaladoVacio{}
	}
	s, err := escaladoWeb.Nuevo(escaladoWeb.Opciones{
		Fuente: f, Catalogo: catalogoReal(t), Base: escaladoWeb.BasePorDefecto,
		CaminoRuta: camino.BasePorDefecto + "/", CaminoClave: camino.ClaveTitulo,
		Pasos: camino.Canonico(),
		Quien: func(*http.Request) string { return "ciso" },
	})
	if err != nil {
		t.Fatalf("construyendo el escalado sin alcance (%s): %v", forma, err)
	}
	return s
}

type uarVacia struct{}

func (uarVacia) Abierta() (*accesos.Campana, error) { return nil, nil }
func (uarVacia) Anotar(ledger.Entrada) error        { return nil }

// aperturaDePrueba dice que SI hay quien sepa abrir una campana, que es lo que
// monta `plazum serve` sobre su directorio de datos. Sin esto, este censo
// construiria la pantalla en una forma que el producto no sirve, y comprobaria
// el estado vacio de una instalacion que no existe.
type aperturaDePrueba struct{}

func (aperturaDePrueba) Abrir([]byte, string, string) error { return nil }

func construirUARVacia(t *testing.T, forma FormaDeLaNada) http.Handler {
	t.Helper()
	var f uarWeb.Campanas
	if forma == NadaPresente {
		f = uarVacia{}
	}
	s, err := uarWeb.Nuevo(uarWeb.Opciones{
		Fuente: f, Abrir: aperturaDePrueba{}, Catalogo: catalogoReal(t), Base: "/uar",
		CaminoRuta: camino.BasePorDefecto + "/", CaminoClave: camino.ClaveTitulo,
		Pasos: camino.Canonico(),
		Quien: func(*http.Request) string { return "ciso" },
		// EL TOKEN, porque sin el la pantalla no pinta formularios y el verbo
		// de este estado vacio ES un formulario. Es la misma pareja que exige
		// PuedeSubir.
		Tokens: func(*http.Request) (string, error) { return "tok-censo", nil },
	})
	if err != nil {
		t.Fatalf("construyendo la revision de accesos sin campana (%s): %v", forma, err)
	}
	return s
}

type actaVacia struct{}

func (actaVacia) Ultima() (acta.Acta, bool, error) { return acta.Acta{}, false, nil }

func construirActaVacia(t *testing.T, forma FormaDeLaNada) http.Handler {
	t.Helper()
	var f actaWeb.Actas
	if forma == NadaPresente {
		f = actaVacia{}
	}
	s, err := actaWeb.Nuevo(actaWeb.Opciones{
		Fuente: f, Catalogo: catalogoReal(t), Base: "/acta",
		CaminoRuta: camino.BasePorDefecto + "/", CaminoClave: camino.ClaveTitulo,
		Pasos: camino.Canonico(),
		Quien: func(*http.Request) string { return "ciso" },
	})
	if err != nil {
		t.Fatalf("construyendo el acta sin ninguna (%s): %v", forma, err)
	}
	return s
}

// ---------------------------------------------------------------------------

func hayDirectorio(t *testing.T, ruta string) bool {
	t.Helper()
	info, err := os.Stat(ruta)
	if err != nil {
		if !os.IsNotExist(err) {
			t.Fatalf("mirando %s: %v", ruta, err)
		}
		return false
	}
	return info.IsDir()
}

// Los dos ordenadores existen para que el informe de un fallo salga siempre
// igual: un mapa se recorre en orden distinto en cada ejecucion.
func ordenadasDelArbol(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func ordenadasDelCenso() []string {
	out := make([]string, 0, len(EstadosVaciosDeLasSuperficies))
	for k := range EstadosVaciosDeLasSuperficies {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func recortar(s string) string {
	const max = 1200
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n[...]"
}
