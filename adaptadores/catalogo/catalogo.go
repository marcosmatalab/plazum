// Package catalogo traduce las cadenas de la INTERFAZ. Es la implementacion de
// puertos.Catalogo, con los ficheros de cadenas embebidos en el binario.
//
// # Que entra aqui y que no entra jamas
//
// Entran los rotulos de la herramienta: titulos de pantalla, acciones, ayudas
// de la propia interfaz, mensajes de error. Y nada mas.
//
// NO entra el texto de una norma. La regla esta escrita en el godoc de
// puertos.Catalogo y en el de nucleo/pantalla, y no es de estilo, es legal: el
// texto del BOE se puede reproducir citando la fuente (art. 13 TRLPI, Decision
// 2011/833/UE), pero una traduccion nuestra ya no es el BOE, es obra nuestra
// presentada como si fuera la norma. El idioma del corpus va POR PAQUETE,
// dentro del paquete: un paquete en aleman es un paquete distinto con su propia
// fuente, no una traduccion en tiempo de render.
//
// Como se hace cumplir, que es lo que separa una regla de un comentario:
//
//	frontera.go       el cargador RECHAZA una entrada con pinta de norma
//	                  (cita normativa, marcado HTML, parrafo largo, clave en un
//	                  espacio de nombres que no existe). Vale para cualquier
//	                  catalogo, incluido uno que llegue de fuera manana.
//	frontera_test.go  contrasta cada valor del catalogo contra el texto legal
//	                  de TODOS los paquetes instalados, por trozos de seis
//	                  palabras. Con su control negativo: se coge un trozo de
//	                  verdad del corpus y se exige que el detector lo cace.
//
// # Los dos idiomas, y solo dos
//
// Se cargan es y en. El aleman llega cuando exista un partner DACH que lo
// revise, y hasta entonces no se promete en ningun sitio. La promesa recortada
// es ejecutable: el go:embed de abajo nombra los dos ficheros uno a uno, asi
// que dejar un de.json en cadenas/ no hace nada, y un test exige que el
// directorio tenga exactamente esos dos. Anadir un idioma es una decision
// consciente en tres sitios, no un fichero que aparece.
//
// # Y el ingles es britanico
//
// programme, organisation, authorisation, analyse, centre. Nunca program,
// organization, authorization, analyze, center.
//
// Es una decision de producto y no una costumbre: el comprador de plazum es
// europeo y el ingles que ya esta leyendo, el de las normas de la UE, esta
// escrito asi. Se tomo de hecho al escribir las cadenas del acta, que salieron
// todas en britanico sin que nadie lo hubiera escrito en ningun sitio, y una
// eleccion que no esta escrita no es una eleccion: es lo que salio.
//
// La vigila TestElInglesDelCatalogoEsBritanico con una lista cerrada de
// grafias, leyendo los VALORES y nunca las claves (una clave se llama
// acta.pantalla.organizacion porque los identificadores de este repositorio van
// en castellano, y su valor Organisation es correcto). El porque entero, con el
// caso de "program", en docs/traducir.md seccion 8.
//
// # Por que un hueco degrada y una ilegalidad se niega
//
// Son dos clases de problema distintas. Que a un idioma le falte una clave es
// incompletitud: la pagina tiene que seguir sirviendose con la clave en crudo
// (lo manda el puerto: Traducir NUNCA falla) y el hueco se caza en CI con
// Faltantes. Que una cadena traiga texto de una norma es un defecto legal: eso
// no degrada, eso no carga.
package catalogo

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"sort"
	"strings"
	"unicode/utf8"
)

// Los ficheros van nombrados uno a uno y no con cadenas/*.json a proposito:
// esta linea ES el recorte de la promesa de aleman. Un de.json en el directorio
// no entra en el binario mientras no se escriba aqui.
//
//go:embed cadenas/es.json cadenas/en.json
var embebidas embed.FS

// PorDefecto es el idioma de referencia. Es el que fija el inventario de claves
// y contra el que Faltantes mide a los demas.
const PorDefecto = "es"

// idiomasEmbebidos son los que trae el binario, en orden. El primero manda.
var idiomasEmbebidos = []string{PorDefecto, "en"}

// Catalogo es una tabla de cadenas por idioma, inmutable una vez cargada.
//
// Se lee desde cada peticion HTTP y desde varias a la vez, asi que despues de
// Cargar nadie escribe: no hay mutex porque no hay escritura.
type Catalogo struct {
	orden  []string // idiomas cargados; orden[0] es el de por defecto
	tablas map[string]map[string]string
}

// Nuevo devuelve el catalogo embebido en el binario, con es y en.
//
// Devuelve error, aunque los ficheros vengan compilados dentro, porque la
// alternativa es un panic en el arranque de un producto que vigila plazos
// legales. En la practica solo falla si alguien edito las cadenas saltandose
// los tests, y entonces el mensaje dice exactamente que entrada las rompio.
func Nuevo() (*Catalogo, error) {
	sub, err := fs.Sub(embebidas, "cadenas")
	if err != nil {
		return nil, fmt.Errorf("no se puede abrir el directorio de cadenas embebido: %w", err)
	}
	return Cargar(sub, idiomasEmbebidos...)
}

// Cargar construye un catalogo leyendo <idioma>.json de fsys. El primer idioma
// de la lista es el de por defecto.
//
// Se expone con fs.FS y no con una ruta para que los tests puedan darle un
// catalogo hostil sin tocar el disco, y para que un futuro paquete de idioma
// entre por el mismo sitio y pase la misma frontera.
func Cargar(fsys fs.FS, idiomas ...string) (*Catalogo, error) {
	if fsys == nil {
		return nil, errors.New("no hay de donde leer las cadenas. Arreglo: pasar el fs.FS " +
			"con los ficheros de idioma, o usar Nuevo() para los embebidos")
	}
	if len(idiomas) == 0 {
		return nil, errors.New("catalogo sin idiomas. Arreglo: pasar al menos uno, " +
			"y recordar que el primero de la lista es el de por defecto")
	}
	c := &Catalogo{
		orden:  make([]string, 0, len(idiomas)),
		tablas: make(map[string]map[string]string, len(idiomas)),
	}
	for _, id := range idiomas {
		if id != normalizarIdioma(id) || id == "" {
			return nil, fmt.Errorf("el idioma %q no es un codigo limpio. Arreglo: usar el "+
				"codigo corto en minusculas (es, en), sin region y sin espacios", id)
		}
		if _, repetido := c.tablas[id]; repetido {
			return nil, fmt.Errorf("el idioma %q se pide dos veces: el segundo taparia al "+
				"primero en silencio", id)
		}
		tabla, err := leerTabla(fsys, id)
		if err != nil {
			return nil, err
		}
		c.orden = append(c.orden, id)
		c.tablas[id] = tabla
	}
	return c, nil
}

// Traducir devuelve la cadena de clave en idioma. NUNCA falla: si la clave no
// esta, devuelve la clave, para que en pantalla falte una etiqueta en vez de
// romperse la pagina.
//
// Tres decisiones que no son obvias:
//
//	clave ausente     se devuelve la CLAVE, no la cadena del idioma por
//	                  defecto. Lo manda el puerto, y la razon es que caer al
//	                  castellano ESCONDE el hueco: nadie lo ve, nadie lo
//	                  arregla, y el usuario ingles se queda con media pagina en
//	                  otro idioma sin saber por que. La clave en crudo es fea y
//	                  se arregla; el castellano silencioso, no.
//	idioma ausente    aqui SI se cae al de por defecto, porque un locale
//	                  desconocido (en-GB, es-419, xx) no es un hueco del
//	                  catalogo, es un navegador. Antes se normaliza: en-GB es en.
//	formateo roto     si el numero de verbos no casa con los argumentos, se
//	                  devuelve la cadena sin formatear en vez de escupir el
//	                  %!s(MISSING) de fmt. El descuadre se caza en CI con
//	                  Descuadres; en pantalla no aparece basura.
//
// Y el plural. Una cadena puede traer sus formas separadas por barra vertical,
// "%d aplica|%d aplican", y se elige segun el primer argumento entero. La
// decision es del catalogo y no de quien llama, a proposito: la forma plural
// depende del idioma (el ruso tiene tres, el arabe seis) y quien pinta la
// pantalla no puede saberlo. Ver elegirForma.
func (c *Catalogo) Traducir(idioma, clave string, args ...any) string {
	valor, ok := c.valor(idioma, clave)
	if !ok {
		return clave
	}
	valor = elegirForma(valor, args)
	if len(args) == 0 {
		// Sin argumentos no se formatea: asi un % suelto en la cadena es un
		// porcentaje y no un verbo a medias.
		return valor
	}
	s := fmt.Sprintf(valor, args...)
	if strings.Contains(s, "%!") {
		return valor
	}
	return s
}

// Idiomas lista los cargados; el primero es el de por defecto.
//
// Devuelve una copia: quien recibe la lista no puede reordenarla y cambiarle el
// idioma por defecto al catalogo de todo el proceso.
func (c *Catalogo) Idiomas() []string {
	return append([]string(nil), c.orden...)
}

// Faltantes son las claves que idioma no cubre respecto al de por defecto.
// Vacio significa catalogo completo. Es lo que se mira en CI.
//
// Una clave presente con el valor vacio cuenta como faltante: un hueco que el
// traductor dejo en blanco es un hueco.
func (c *Catalogo) Faltantes(idioma string) []string {
	ref, ok := c.referencia()
	if !ok {
		return nil
	}
	tabla := c.tablas[normalizarIdioma(idioma)] // nil si no esta cargado
	var out []string
	for k := range ref {
		if v, hay := tabla[k]; !hay || strings.TrimSpace(v) == "" {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// Sobrantes son las claves que idioma tiene y el de por defecto no.
//
// No esta en el puerto y aqui si, porque el hueco al reves tambien es un
// defecto: una clave que solo existe en ingles es una clave muerta o un
// renombrado a medias, y en los dos casos alguien va a leer una etiqueta que
// nadie ensena. Un idioma que no esta cargado no tiene sobrantes.
func (c *Catalogo) Sobrantes(idioma string) []string {
	ref, ok := c.referencia()
	if !ok {
		return nil
	}
	tabla, cargado := c.tablas[normalizarIdioma(idioma)]
	if !cargado {
		return nil
	}
	var out []string
	for k := range tabla {
		if _, hay := ref[k]; !hay {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// Descuadres son las claves cuyos verbos de formateo no casan con los del
// idioma por defecto, o cuyas formas plurales no casan entre si.
//
// Es el defecto de traduccion mas facil de cometer y el mas dificil de ver: el
// traductor se come el %s o lo escribe dos veces, y esa etiqueta concreta sale
// mal solo cuando el usuario llega a ella. Traducir se protege devolviendo la
// cadena cruda, pero eso tapa el sintoma; esto lo pone en rojo en CI.
func (c *Catalogo) Descuadres(idioma string) []string {
	ref, ok := c.referencia()
	if !ok {
		return nil
	}
	tabla, cargado := c.tablas[normalizarIdioma(idioma)]
	if !cargado {
		return nil
	}
	var out []string
	for k, v := range tabla {
		patron, hay := ref[k]
		if !hay {
			continue // eso es un sobrante, no un descuadre
		}
		vsRef, coherenteRef := verbosDeValor(patron)
		vs, coherente := verbosDeValor(v)
		if !coherenteRef || !coherente ||
			strings.Join(vsRef, ",") != strings.Join(vs, ",") {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// Claves es el inventario del idioma por defecto, ordenado.
func (c *Catalogo) Claves() []string {
	ref, ok := c.referencia()
	if !ok {
		return nil
	}
	out := make([]string, 0, len(ref))
	for k := range ref {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (c *Catalogo) referencia() (map[string]string, bool) {
	if c == nil || len(c.orden) == 0 {
		return nil, false
	}
	return c.tablas[c.orden[0]], true
}

func (c *Catalogo) valor(idioma, clave string) (string, bool) {
	if c == nil || len(c.orden) == 0 {
		return "", false
	}
	tabla, ok := c.tablas[normalizarIdioma(idioma)]
	if !ok {
		tabla = c.tablas[c.orden[0]]
	}
	v, ok := tabla[clave]
	if !ok || strings.TrimSpace(v) == "" {
		return "", false
	}
	return v, true
}

// normalizarIdioma reduce un locale a su codigo corto: "en-GB" y "EN_gb" son
// "en". Un navegador manda lo que quiere en Accept-Language y eso no puede
// decidir si el usuario ve la interfaz o ve claves.
func normalizarIdioma(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if i := strings.IndexAny(s, "-_"); i >= 0 {
		s = s[:i]
	}
	return s
}

// separadorDeFormas parte las formas plurales de una cadena. Se elige la barra
// vertical porque no aparece en prosa de interfaz en ningun idioma latino.
const separadorDeFormas = "|"

// elegirForma escoge la forma plural de una cadena segun el primer argumento
// entero que reciba.
//
// La regla de hoy es la de es y en, y de casi todas las lenguas germanicas y
// romances: uno es singular y el resto es plural. Va aqui, en el catalogo, y no
// en quien pinta la pantalla, porque la pantalla no sabe en que idioma esta
// escribiendo. Cuando entre un idioma con mas formas (el ruso tiene tres, el
// arabe seis), ESTE es el sitio donde se amplia, y las formas de mas ya viajan
// en el fichero de cadenas sin que haga falta tocar a quien llama.
//
// Sin contador entre los argumentos se devuelve la ULTIMA forma, que es el
// plural: es la que suena bien con una cantidad desconocida.
func elegirForma(valor string, args []any) string {
	if !strings.Contains(valor, separadorDeFormas) {
		return valor
	}
	formas := strings.Split(valor, separadorDeFormas)
	n, hay := primerContador(args)
	if hay && n == 1 {
		return formas[0]
	}
	return formas[len(formas)-1]
}

// primerContador busca el primer argumento entero. Los enteros con signo y sin
// signo van todos porque quien llama pasa lo que le devuelve len() o un campo
// del modelo, y no tiene por que convertir.
func primerContador(args []any) (int64, bool) {
	for _, a := range args {
		switch v := a.(type) {
		case int:
			return int64(v), true
		case int8:
			return int64(v), true
		case int16:
			return int64(v), true
		case int32:
			return int64(v), true
		case int64:
			return v, true
		case uint:
			return int64(v), true // #nosec G115 -- un contador de pantalla no llega a 2^63
		case uint8:
			return int64(v), true
		case uint16:
			return int64(v), true
		case uint32:
			return int64(v), true
		case uint64:
			return int64(v), true // #nosec G115 -- idem
		}
	}
	return 0, false
}

// verbosDeValor devuelve los verbos de una cadena con formas plurales, y si
// todas las formas usan los mismos.
//
// Que sean los mismos importa: dos formas de la misma clave tienen que ser
// intercambiables. Si el singular lleva %d y el plural no, la etiqueta sale sin
// el numero justo en el caso que menos se prueba, que es el de uno.
func verbosDeValor(valor string) (vs []string, coherente bool) {
	formas := strings.Split(valor, separadorDeFormas)
	vs = verbos(formas[0])
	patron := strings.Join(vs, ",")
	for _, f := range formas[1:] {
		if strings.Join(verbos(f), ",") != patron {
			return vs, false
		}
	}
	return vs, true
}

// verbos devuelve, en orden, los verbos de formateo de una cadena: "%s de %d"
// da [s d]. El %% es un porcentaje literal y no cuenta.
func verbos(s string) []string {
	var out []string
	for i := 0; i < len(s); i++ {
		if s[i] != '%' {
			continue
		}
		j := i + 1
		if j < len(s) && s[j] == '%' {
			i = j // porcentaje literal
			continue
		}
		// banderas, ancho, precision e indice explicito (%[1]s) hasta la letra
		for j < len(s) && !esLetra(s[j]) {
			j++
		}
		if j >= len(s) {
			out = append(out, "?") // un % colgando al final: descuadre seguro
			break
		}
		out = append(out, string(s[j]))
		i = j
	}
	return out
}

func esLetra(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }

func leerTabla(fsys fs.FS, idioma string) (map[string]string, error) {
	nombre := idioma + ".json"
	b, err := fs.ReadFile(fsys, nombre)
	if err != nil {
		return nil, fmt.Errorf("no se puede leer el catalogo %s: %w. Arreglo: cada idioma "+
			"cargado necesita su fichero de cadenas", nombre, err)
	}
	if !utf8.Valid(b) {
		return nil, fmt.Errorf("%s no es UTF-8 valido. Arreglo: guardar el fichero en UTF-8, "+
			"sin BOM y sin bytes sueltos de otra codificacion", nombre)
	}
	m, err := decodificarPlano(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", nombre, err)
	}
	if len(m) == 0 {
		return nil, fmt.Errorf("%s no tiene ninguna cadena. Arreglo: un catalogo vacio "+
			"deja la interfaz entera en claves crudas, asi que no se carga", nombre)
	}
	for clave, valor := range m {
		if motivo := motivoRechazo(clave, valor); motivo != "" {
			return nil, fmt.Errorf("%s: la clave %q no puede vivir en el catalogo. %s",
				nombre, clave, motivo)
		}
	}
	return m, nil
}

// decodificarPlano exige un objeto JSON de cadena a cadena y NADA mas.
//
// Va a mano con el Decoder en vez de json.Unmarshal por dos cosas que el
// Unmarshal se traga en silencio: una clave repetida (gana la ultima, y el
// traductor que escribio la primera nunca sabe por que no sale) y un valor que
// no es cadena (un objeto anidado se convertiria en un catalogo a medias).
func decodificarPlano(b []byte) (map[string]string, error) {
	d := json.NewDecoder(bytes.NewReader(b))
	t, err := d.Token()
	if err != nil {
		return nil, fmt.Errorf("JSON ilegible: %w. Arreglo: el catalogo es un objeto JSON "+
			"de clave a cadena, sin comas de mas ni comentarios", err)
	}
	if delim, ok := t.(json.Delim); !ok || delim != '{' {
		return nil, errors.New("el catalogo tiene que ser un objeto JSON de clave a cadena, " +
			"y empieza por otra cosa")
	}
	m := map[string]string{}
	for d.More() {
		kt, err := d.Token()
		if err != nil {
			return nil, fmt.Errorf("JSON ilegible leyendo una clave: %w", err)
		}
		clave, ok := kt.(string)
		if !ok {
			return nil, fmt.Errorf("una clave del catalogo no es una cadena: %v", kt)
		}
		if _, repetida := m[clave]; repetida {
			return nil, fmt.Errorf("la clave %q aparece dos veces. Arreglo: dejar una sola. "+
				"La segunda tapa a la primera y el traductor que escribio la primera no "+
				"llega a ver nunca su cadena", clave)
		}
		vt, err := d.Token()
		if err != nil {
			return nil, fmt.Errorf("JSON ilegible leyendo el valor de %q: %w", clave, err)
		}
		valor, ok := vt.(string)
		if !ok {
			return nil, fmt.Errorf("el valor de %q no es una cadena. Arreglo: el catalogo es "+
				"plano, sin objetos ni listas anidadas: una clave, una cadena", clave)
		}
		m[clave] = valor
	}
	if _, err := d.Token(); err != nil { // el cierre del objeto
		return nil, fmt.Errorf("JSON ilegible en el cierre: %w", err)
	}
	if _, err := d.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("sobra contenido detras del objeto del catalogo. Arreglo: " +
			"un fichero de cadenas es un solo objeto JSON y nada mas")
	}
	return m, nil
}
