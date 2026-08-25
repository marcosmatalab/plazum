package main

// La vigilancia normativa, que es la mitad que hace valiosa dos veces a esta
// herramienta.
//
// Ingerir una norma una vez es util. Volver a ingerirla y que diga QUE HA
// CAMBIADO desde la vez anterior es el producto: es lo que alimenta la pagina
// publica de vigilancia (la tabla fecha de la fuente hacia fecha del paquete) y
// es lo unico que un competidor no puede fingir, porque es historico.
//
// Por eso el almacen se disena para eso desde el principio y no despues:
//
//	<almacen>/<clave>/instantanea.json   la ultima extraccion completa. Es contra
//	                                     lo que se compara la siguiente.
//	<almacen>/<clave>/historial.jsonl    una linea por observacion, solo se anade.
//	                                     Es el track record, y por eso no se
//	                                     reescribe nunca.
//
// El historial es append-only a proposito. Un track record que se puede reescribir
// no es un track record.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Cambios es el veredicto de una reejecucion.
type Cambios struct {
	PrimeraVez  bool     `json:"primera_vez"`
	Nuevos      []string `json:"nuevos"`
	Modificados []string `json:"modificados"`
	Derogados   []string `json:"derogados"`
	SinCambio   int      `json:"sin_cambio"`
	HuellaAntes string   `json:"huella_antes,omitempty"`
	HuellaAhora string   `json:"huella_ahora"`
	// FuenteAntes y FuenteAhora son la marca de actualizacion que declara la
	// fuente. Una norma puede reconsolidarse sin que cambie ni una coma del
	// articulado, y eso tambien se quiere ver.
	FuenteAntes string `json:"fecha_fuente_antes,omitempty"`
	FuenteAhora string `json:"fecha_fuente_ahora,omitempty"`
}

// Hay dice si esta observacion tiene algo que contar. La primera vez NO cuenta
// como cambio: no habia nada con lo que comparar, y decir "132 articulos nuevos"
// la primera vez es ruido que entrena a ignorar la herramienta.
func (c Cambios) Hay() bool {
	return !c.PrimeraVez && (len(c.Nuevos) > 0 || len(c.Modificados) > 0 || len(c.Derogados) > 0)
}

// Comparar es el corazon de la vigilancia. Casa articulos por su referencia (el
// rotulo que publica la fuente) y no por su id interno, porque los id del BOE se
// derivan de la posicion y se desplazan al insertar un articulo.
func Comparar(antes, ahora *Extraccion) Cambios {
	c := Cambios{HuellaAhora: ahora.Huella, FuenteAhora: ahora.Fuente.ActualizadaEn}
	if antes == nil {
		c.PrimeraVez = true
		return c
	}
	c.HuellaAntes = antes.Huella
	c.FuenteAntes = antes.Fuente.ActualizadaEn

	previos := map[string]Articulo{}
	for _, a := range antes.Articulos {
		previos[a.Referencia] = a
	}
	vistos := map[string]bool{}
	for _, a := range ahora.Articulos {
		vistos[a.Referencia] = true
		p, habia := previos[a.Referencia]
		switch {
		case !habia:
			c.Nuevos = append(c.Nuevos, a.Referencia)
		case a.Derogado && !p.Derogado:
			// Derogar es un cambio de texto ademas de un cambio de estado. Se
			// cuenta como derogacion y no como modificacion porque es lo que
			// tiene que leer primero quien mantiene el paquete.
			c.Derogados = append(c.Derogados, a.Referencia)
		case a.Huella != p.Huella:
			c.Modificados = append(c.Modificados, a.Referencia)
		default:
			c.SinCambio++
		}
	}
	// Lo que estaba y ya no aparece: el BOE deja de servir el bloque cuando el
	// precepto desaparece del texto consolidado.
	for _, a := range antes.Articulos {
		if !vistos[a.Referencia] {
			c.Derogados = append(c.Derogados, a.Referencia)
		}
	}
	sort.Strings(c.Nuevos)
	sort.Strings(c.Modificados)
	sort.Strings(c.Derogados)
	return c
}

// --- el almacen ---

// Almacen guarda instantaneas e historial en disco.
type Almacen struct{ Dir string }

// Entrada es una linea del historial: lo que se observo y cuando.
type Entrada struct {
	Observado     string  `json:"observado"` // RFC3339, cuando lo vio esta herramienta
	Jurisdiccion  string  `json:"jurisdiccion"`
	Identificador string  `json:"identificador"`
	Titulo        string  `json:"titulo"`
	ELI           string  `json:"eli,omitempty"`
	URLDocumento  string  `json:"url_documento"`
	Articulos     int     `json:"articulos"`
	Cambios       Cambios `json:"cambios"`
}

// clave es el nombre de directorio de una norma dentro del almacen. Se limpia a
// conciencia: el identificador viene de una fuente externa y acaba en una ruta
// de fichero, que es exactamente donde un identificador con ../ hace dano.
func clave(o Origen) string {
	sanear := func(s string) string {
		var b strings.Builder
		for _, r := range strings.ToLower(s) {
			switch {
			case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-':
				b.WriteRune(r)
			default:
				b.WriteByte('_')
			}
		}
		out := b.String()
		if len(out) > 100 {
			out = out[:100]
		}
		return out
	}
	id := sanear(o.Identificador)
	if strings.Trim(id, "_-") == "" {
		// Sin identificador la extraccion esta rota, pero mejor un directorio
		// que se ve raro que todas las normas amontonadas en el mismo sitio.
		id = "sin-identificador"
	}
	j := sanear(o.Jurisdiccion)
	if strings.Trim(j, "_-") == "" {
		j = "sin-jurisdiccion"
	}
	return j + "-" + id
}

func (a Almacen) dirDe(k string) string { return filepath.Join(a.Dir, k) }

// Anterior devuelve la ultima instantanea guardada, o nil si es la primera vez.
//
// Una instantanea ilegible NO se trata como "primera vez": eso convertiria un
// fichero corrupto en un parte de "sin novedad", que es la peor mentira que
// puede contar una herramienta de vigilancia.
func (a Almacen) Anterior(k string) (*Extraccion, error) {
	ruta := filepath.Join(a.dirDe(k), "instantanea.json")
	b, err := os.ReadFile(ruta) // #nosec G304 -- ruta derivada de clave(), saneada
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("no puedo leer la instantanea anterior %s: %w. "+
			"Arreglo: si el fichero esta corrupto, borralo a mano y la proxima ejecucion "+
			"volvera a contar como primera vez", ruta, err)
	}
	var e Extraccion
	if err := json.Unmarshal(b, &e); err != nil {
		return nil, fmt.Errorf("la instantanea anterior %s no es JSON valido: %w. "+
			"Arreglo: borrala a mano; NO se trata como si no existiera, porque entonces un "+
			"fichero corrupto se leeria como sin novedad", ruta, err)
	}
	return &e, nil
}

// Registrar guarda la instantanea nueva y anade la linea al historial.
//
// Primero el historial y despues la instantanea: si el proceso muere entre las
// dos, la proxima ejecucion vuelve a comparar contra la instantanea vieja y
// vuelve a anotar el cambio. Al reves se perderia la anotacion y el cambio ya no
// se veria nunca, porque la instantanea nueva ya lo incluiria.
func (a Almacen) Registrar(k string, e *Extraccion, c Cambios, ahora time.Time) error {
	dir := a.dirDe(k)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("no puedo crear %s: %w", dir, err)
	}
	entrada := Entrada{
		Observado:     ahora.UTC().Format(time.RFC3339),
		Jurisdiccion:  e.Fuente.Jurisdiccion,
		Identificador: e.Fuente.Identificador,
		Titulo:        e.Fuente.Titulo,
		ELI:           e.Fuente.ELI,
		URLDocumento:  e.Fuente.URLDocumento,
		Articulos:     len(e.Articulos),
		Cambios:       c,
	}
	linea, err := json.Marshal(entrada)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dir, "historial.jsonl"), // #nosec G304 -- ruta saneada
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("no puedo abrir el historial de %s: %w", k, err)
	}
	if _, err := f.Write(append(linea, '\n')); err != nil {
		_ = f.Close()
		return fmt.Errorf("no puedo anadir al historial de %s: %w", k, err)
	}
	if err := f.Close(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "instantanea.json"), b, 0o600)
}

// Historial devuelve todo lo observado, de lo mas reciente a lo mas antiguo. Es
// la entrada de la pagina publica de vigilancia.
func (a Almacen) Historial() ([]Entrada, error) {
	entradas, err := os.ReadDir(a.Dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("no puedo leer el almacen %s: %w", a.Dir, err)
	}
	var out []Entrada
	for _, e := range entradas {
		if !e.IsDir() {
			continue
		}
		ruta := filepath.Join(a.Dir, e.Name(), "historial.jsonl")
		b, err := os.ReadFile(ruta) // #nosec G304 -- ruta compuesta del propio almacen
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("no puedo leer %s: %w", ruta, err)
		}
		for n, linea := range strings.Split(string(b), "\n") {
			if strings.TrimSpace(linea) == "" {
				continue
			}
			var ent Entrada
			if err := json.Unmarshal([]byte(linea), &ent); err != nil {
				return nil, fmt.Errorf("%s linea %d no es JSON: %w. Arreglo: el historial es "+
					"append-only, asi que una linea rota se corrige a mano y se deja anotado",
					ruta, n+1, err)
			}
			out = append(out, ent)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Observado > out[j].Observado })
	return out, nil
}
