// Package plantilla implementa puertos.Plantilla con html/template.
//
// Las tres decisiones que sostienen este adaptador:
//
//	un juego de plantillas POR IDIOMA, construido al arrancar. La funcion `t`
//	de las plantillas tiene que traducir a UN idioma, y html/template ata las
//	funciones al parsear, no al ejecutar. Las alternativas eran clonar el arbol
//	en cada peticion (coste por peticion, y Clone falla despues de ejecutar) o
//	pasar el idioma en cada llamada dentro de la plantilla (que se olvida en
//	una linea y nadie lo nota). Con un arbol por idioma, olvidarlo no es
//	posible: no hay donde escribirlo.
//
//	el catalogo devuelve TEXTO, no HTML. Lo que sale de Traducir se escapa como
//	cualquier otro dato. Un catalogo es un fichero de datos y un fichero de
//	datos no inyecta marcado. Aqui no hay ni un template.HTML, y lo vigila un
//	test.
//
//	el contenido del corpus NO pasa por el catalogo. Este adaptador no traduce
//	nada por su cuenta: solo expone `t` para las CLAVES de interfaz. La
//	etiqueta de un atributo, la ayuda, la cita y el texto de una pregunta
//	viajan tal cual, en el idioma del paquete. Traducir texto transcrito del
//	BOE crea obra derivada y se sale de la estratificacion de licencias del
//	corpus (ver el godoc de puertos.Catalogo).
package plantilla

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"sort"
	"strings"

	"dutiq/puertos"
)

// Errores del adaptador, como centinelas. Un test que compruebe que un motor
// sin catalogo se rechaza tiene que poder hacerlo con errors.Is y no buscando
// una subcadena que manana cambia de redaccion.
var (
	ErrSinCatalogo   = errors.New("motor de plantillas sin catalogo utilizable")
	ErrSinPlantillas = errors.New("el motor de plantillas no ha cargado ninguna plantilla")
	ErrNoExiste      = errors.New("plantilla que no existe")
	ErrRender        = errors.New("la plantilla fallo al renderizar")
)

// Motor renderiza plantillas ya parseadas, una vez por idioma del catalogo.
//
// Es seguro usarlo desde varias goroutines: despues de Nuevo no se vuelve a
// parsear ni a registrar funciones, y ejecutar una plantilla ya parseada lo es.
type Motor struct {
	porIdioma map[string]*template.Template
	idiomas   []string
	defecto   string
	nombres   []string
}

var _ puertos.Plantilla = (*Motor)(nil)

// Nuevo carga las plantillas de sistema que casen con patrones, una vez por
// idioma declarado por el catalogo.
//
// sistema suele ser un embed.FS: las plantillas viajan dentro del binario
// porque un fichero de plantilla suelto en disco es una via de ejecucion de
// codigo en el servidor de quien las pueda escribir.
//
// Si patrones viene vacio se usa "*.html".
func Nuevo(sistema fs.FS, cat puertos.Catalogo, patrones ...string) (*Motor, error) {
	if sistema == nil {
		return nil, fmt.Errorf("%w: no se ha dado ningun sistema de ficheros de donde "+
			"leer las plantillas. Arreglo: pasa el embed.FS del paquete que las tiene",
			ErrSinPlantillas)
	}
	if cat == nil {
		return nil, fmt.Errorf("%w: el catalogo es nil. Arreglo: construye el motor con un "+
			"puertos.Catalogo. Sin el, los rotulos de la interfaz saldrian como claves "+
			"(\"pantalla.alcance.titulo\") y la pagina seria ilegible", ErrSinCatalogo)
	}
	idiomas := cat.Idiomas()
	if len(idiomas) == 0 {
		return nil, fmt.Errorf("%w: el catalogo no declara ningun idioma, asi que no hay "+
			"idioma por defecto al que caer. Arreglo: que Idiomas() devuelva al menos uno, "+
			"y el primero es el de por defecto", ErrSinCatalogo)
	}
	if len(patrones) == 0 {
		patrones = []string{"*.html"}
	}
	// Se comprueba ANTES de parsear para poder dar un error propio. ParseFS
	// falla con "pattern matches no files", que no dice si el problema es el
	// patron o la directiva go:embed, y es justo lo que le pasa a quien
	// mueve un fichero de sitio.
	casan := 0
	for _, p := range patrones {
		f, err := fs.Glob(sistema, p)
		if err != nil {
			return nil, fmt.Errorf("%w: el patron %q no es valido: %w", ErrSinPlantillas, p, err)
		}
		casan += len(f)
	}
	if casan == 0 {
		return nil, fmt.Errorf("%w: los patrones %v no casan con ningun fichero. Arreglo: "+
			"revisa la directiva go:embed del paquete que embebe las plantillas y que los "+
			"patrones lleven el mismo prefijo de directorio", ErrSinPlantillas, patrones)
	}

	m := &Motor{
		porIdioma: make(map[string]*template.Template, len(idiomas)),
		idiomas:   append([]string(nil), idiomas...),
		defecto:   idiomas[0],
	}
	for _, idioma := range idiomas {
		if _, repetido := m.porIdioma[idioma]; repetido {
			return nil, fmt.Errorf("%w: el catalogo declara el idioma %q dos veces. "+
				"Arreglo: que Idiomas() no repita", ErrSinCatalogo, idioma)
		}
		funcs := template.FuncMap{
			// t traduce una CLAVE de interfaz. Devuelve string, no
			// template.HTML: lo que ponga el catalogo se escapa.
			"t": traductor(cat, idioma),
		}
		t, err := template.New("dutiq").Funcs(funcs).ParseFS(sistema, patrones...)
		if err != nil {
			return nil, fmt.Errorf("no puedo parsear las plantillas para el idioma %q "+
				"con los patrones %v: %w. Arreglo: comprueba que los ficheros estan "+
				"embebidos y que la sintaxis de la plantilla es valida", idioma, patrones, err)
		}
		m.porIdioma[idioma] = t
	}

	// Los nombres cargados, para poder decirlos cuando alguien pida uno que no
	// existe. Un "plantilla no encontrada" sin la lista obliga a leer codigo.
	for _, t := range m.porIdioma[m.defecto].Templates() {
		if n := t.Name(); n != "" && n != "dutiq" {
			m.nombres = append(m.nombres, n)
		}
	}
	sort.Strings(m.nombres)
	if len(m.nombres) == 0 {
		return nil, fmt.Errorf("%w: los patrones %v no han casado con ninguna plantilla. "+
			"Arreglo: revisa la directiva go:embed y los patrones", ErrSinPlantillas, patrones)
	}
	return m, nil
}

// traductor cierra sobre el catalogo y UN idioma. Esta extraida para que quede
// claro que el idioma se fija aqui y no se puede pasar desde la plantilla.
func traductor(cat puertos.Catalogo, idioma string) func(string, ...any) string {
	return func(clave string, args ...any) string {
		return cat.Traducir(idioma, clave, args...)
	}
}

// Render escribe la plantilla nombre con datos, en idioma.
//
// AVISO al llamante: html/template escribe segun ejecuta, asi que un fallo a
// mitad deja salida a medias en w. Quien sirva HTTP debe renderizar a un buffer
// y volcar solo si esto devuelve nil; si no, un error acaba siendo una pagina
// rota con un 200. La superficie de pantallas lo hace asi.
func (m *Motor) Render(w io.Writer, nombre string, datos any, idioma string) error {
	if w == nil {
		return fmt.Errorf("%w: no hay donde escribir (w es nil)", ErrRender)
	}
	t := m.porIdioma[m.Resolver(idioma)]
	if t.Lookup(nombre) == nil {
		return fmt.Errorf("%w: %q. Las cargadas son %v. Arreglo: usa uno de esos nombres "+
			"o anade el fichero al go:embed del paquete que las embebe",
			ErrNoExiste, nombre, m.nombres)
	}
	if err := t.ExecuteTemplate(w, nombre, datos); err != nil {
		return fmt.Errorf("%w %q (idioma %q): %w", ErrRender, nombre, idioma, err)
	}
	return nil
}

// Resolver dice que idioma cargado se va a usar para uno pedido.
//
// Exacto, si no la etiqueta primaria ("es-ES" cae en "es"), si no el de por
// defecto. NUNCA falla: un idioma desconocido tiene que dar una pagina en el
// idioma de por defecto, no un error. Es publico porque quien sirve HTTP
// necesita saber que idioma acabo saliendo para ponerlo en <html lang>: una
// pagina que declara un idioma que no es el que lleva dentro rompe los
// lectores de pantalla y es un fallo de accesibilidad, no un detalle.
func (m *Motor) Resolver(idioma string) string {
	if _, ok := m.porIdioma[idioma]; ok {
		return idioma
	}
	if i := strings.IndexAny(idioma, "-_"); i > 0 {
		if primaria := idioma[:i]; m.porIdioma[primaria] != nil {
			return primaria
		}
	}
	return m.defecto
}

// Idiomas devuelve los idiomas cargados; el primero es el de por defecto.
func (m *Motor) Idiomas() []string { return append([]string(nil), m.idiomas...) }

// Nombres devuelve las plantillas cargadas, ordenadas.
func (m *Motor) Nombres() []string { return append([]string(nil), m.nombres...) }
