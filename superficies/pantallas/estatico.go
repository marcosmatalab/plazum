package pantallas

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
)

// Los ficheros que sirve la superficie, dentro del binario.
//
// htmx va VENDORIZADO, no por CDN, y no es una preferencia de estilo: una
// etiqueta script apuntando a un tercero convierte a ese tercero en autor de la
// pagina donde el operador decide si cumple la ley, y ademas hace que el
// producto necesite salida a internet en una red que puede no tenerla. Su
// version y su licencia constan en DEPENDENCIAS.md; la licencia viaja al lado
// del fichero, como exige distribuir codigo ajeno.
//
// No hay npm, ni Makefile, ni paso de construccion: el fichero esta en el repo
// y entra en el binario con go:embed.

//go:embed plantillas
var plantillasFS embed.FS

//go:embed estatico
var estaticoFS embed.FS

// Plantillas expone el sistema de ficheros de las plantillas embebidas, para
// quien quiera construir el motor de render por su cuenta (otro adaptador de
// puertos.Plantilla, por ejemplo) en vez de dejar que lo haga Nuevo.
func Plantillas() fs.FS { return plantillasFS }

// recurso es un fichero estatico ya leido, con lo que hace falta para servirlo
// sin volver a tocar el sistema de ficheros.
type recurso struct {
	cuerpo []byte
	tipo   string
}

// tiposPorExtension es la lista CERRADA de lo que esta superficie sabe servir.
//
// Es cerrada a proposito y no se delega en mime.TypeByExtension: ese consulta
// el registro del sistema operativo, o sea que el Content-Type de un fichero
// nuestro dependeria de la maquina donde corre. En Windows es un caso conocido
// de servir JavaScript como text/plain, y con nosniff puesto eso deja la pagina
// sin htmx sin decir por que.
var tiposPorExtension = map[string]string{
	".js":  "text/javascript; charset=utf-8",
	".css": "text/css; charset=utf-8",
	".txt": "text/plain; charset=utf-8",
	".svg": "image/svg+xml",
}

// estaticos lee el arbol embebido una vez. Devuelve tambien los nombres para
// que un test pueda exigir que todo lo embebido se pueda servir: un fichero
// embebido con una extension que no sabemos servir es peso muerto en el binario
// y una etiqueta rota en la pagina.
func estaticos() (map[string]recurso, []string, error) {
	sub, err := fs.Sub(estaticoFS, "estatico")
	if err != nil {
		return nil, nil, err
	}
	out := map[string]recurso{}
	var nombres []string
	err = fs.WalkDir(sub, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(sub, p)
		if err != nil {
			return err
		}
		nombres = append(nombres, p)
		if tipo, ok := tiposPorExtension[path.Ext(p)]; ok {
			out[p] = recurso{cuerpo: b, tipo: tipo}
		}
		return nil
	})
	sort.Strings(nombres)
	return out, nombres, err
}

// verEstatico sirve un fichero embebido por nombre exacto.
//
// Por nombre exacto y contra un mapa construido al arrancar: no hay ruta que
// recorrer, asi que no hay travesia de directorios que impedir. Lo que no este
// en el mapa es un 404 y punto.
//
// El nombre se saca de r.URL.Path y NO de r.PathValue. Es un detalle que costo
// un fallo: la superficie despacha con ServeMux.Handler(r), que devuelve el
// manejador pero NO rellena los comodines de la ruta, asi que PathValue habria
// devuelto siempre la cadena vacia y ni el CSS ni htmx se habrian servido.
func (s *Superficie) verEstatico(w http.ResponseWriter, r *http.Request) {
	fichero, ok := strings.CutPrefix(r.URL.Path, "/estatico/")
	if ok {
		// r.URL.Path llega ya descodificado, asi que un %2e%2e%2f se ve como
		// lo que es. Un nombre con barra dentro no es un fichero nuestro.
		ok = fichero != "" && !strings.Contains(fichero, "/")
	}
	var res recurso
	if ok {
		res, ok = ficherosEstaticos[fichero]
	}
	if !ok {
		s.fallo(w, r, http.StatusNotFound, "error.no_encontrado")
		return
	}
	w.Header().Set("Content-Type", res.tipo)
	w.Header().Set("Content-Length", strconv.Itoa(len(res.cuerpo)))
	// El nombre del fichero lleva la version dentro (htmx-2.0.10.min.js), asi
	// que su contenido no cambia nunca bajo la misma direccion y se puede
	// cachear a lo bruto. Al subir de version cambia el nombre y el navegador
	// pide el nuevo solo.
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(res.cuerpo)
}

// ficherosEstaticos se resuelve al cargar el paquete. Si el arbol embebido no se
// puede leer, es un fallo de construccion del binario y no de una peticion: se
// prefiere reventar aqui, donde lo ve quien compila, a servir paginas sin CSS.
var ficherosEstaticos, nombresEstaticos = func() (map[string]recurso, []string) {
	m, n, err := estaticos()
	if err != nil {
		panic("superficies/pantallas: no puedo leer los ficheros estaticos embebidos: " +
			err.Error() + ". Arreglo: revisa la directiva go:embed de estatico.go")
	}
	return m, n
}()
