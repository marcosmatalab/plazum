package tsa

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Espera base y tope del reintento. Con base 5 min y tope 24 h, un corte largo
// de las dos TSAs se reintenta a los 5, 10, 20, 40 minutos y asi hasta una vez
// al dia: ni martillea a una TSA caida ni se olvida del anclaje.
const (
	esperaBase = 5 * time.Minute
	esperaTope = 24 * time.Hour
	// Un hash pendiente no caduca, pero si lleva demasiados intentos deja de
	// ser un corte y pasa a ser un problema de configuracion que hay que mirar.
	intentosParaAvisar = 10
)

// Pendiente es un anclaje que no se pudo sellar y espera reintento.
type Pendiente struct {
	Hash     string    `json:"hash"` // hex del sha256
	Encolado time.Time `json:"encolado"`
	Intentos int       `json:"intentos"`
	Ultimo   time.Time `json:"ultimo_intento"`
}

// Resuelto es un pendiente que por fin consiguio sello. El llamante lo engancha
// al checkpoint que quedo sin anclaje.
type Resuelto struct {
	Hash  []byte
	Token []byte
	// Encolado dice cuanto llevaba esperando, que es el dato que interesa al
	// auditor: el sello no prueba la fecha del checkpoint sino la del sello.
	Encolado time.Time
}

// Cola es la cola local de anclajes pendientes, persistida en un fichero.
//
// Persistida y no en memoria porque el caso que tiene que cubrir es
// exactamente el de reiniciar: si las dos TSAs estan caidas y el proceso se
// para, los anclajes pendientes no pueden evaporarse.
type Cola struct {
	ruta string
	mu   sync.Mutex
}

// NuevaCola abre (o crea al primer uso) la cola en ruta.
func NuevaCola(ruta string) *Cola { return &Cola{ruta: ruta} }

// Encolar anota un hash pendiente. Si ya estaba, no lo duplica: el mismo
// checkpoint reintentado sigue siendo un solo anclaje pendiente.
func (q *Cola) Encolar(hash []byte, ahora time.Time) error {
	if len(hash) == 0 {
		return fmt.Errorf("no se encola un hash vacio")
	}
	q.mu.Lock()
	defer q.mu.Unlock()

	ps, err := q.leer()
	if err != nil {
		return err
	}
	h := hex.EncodeToString(hash)
	for _, p := range ps {
		if p.Hash == h {
			return nil
		}
	}
	ps = append(ps, Pendiente{Hash: h, Encolado: ahora.UTC()})
	return q.escribir(ps)
}

// Pendientes devuelve lo que queda por sellar, del mas antiguo al mas nuevo.
func (q *Cola) Pendientes() ([]Pendiente, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	ps, err := q.leer()
	if err != nil {
		return nil, err
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].Encolado.Before(ps[j].Encolado) })
	return ps, nil
}

// Atascados son los pendientes que llevan demasiados intentos: ya no es un
// corte pasajero, es algo que mirar (URL mal, TSA que cerro, red cortada).
func (q *Cola) Atascados() ([]Pendiente, error) {
	ps, err := q.Pendientes()
	if err != nil {
		return nil, err
	}
	var out []Pendiente
	for _, p := range ps {
		if p.Intentos >= intentosParaAvisar {
			out = append(out, p)
		}
	}
	return out, nil
}

// Resolver saca un hash de la cola: ya tiene sello.
func (q *Cola) Resolver(hash []byte) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	ps, err := q.leer()
	if err != nil {
		return err
	}
	h := hex.EncodeToString(hash)
	out := ps[:0]
	for _, p := range ps {
		if p.Hash != h {
			out = append(out, p)
		}
	}
	return q.escribir(out)
}

// anotarIntento sube el contador de un pendiente tras un reintento fallido.
func (q *Cola) anotarIntento(hash string, ahora time.Time) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	ps, err := q.leer()
	if err != nil {
		return err
	}
	for i := range ps {
		if ps[i].Hash == hash {
			ps[i].Intentos++
			ps[i].Ultimo = ahora.UTC()
		}
	}
	return q.escribir(ps)
}

// toca dice si a un pendiente le corresponde reintento ya, segun el backoff.
func (p Pendiente) toca(ahora time.Time) bool {
	if p.Intentos == 0 || p.Ultimo.IsZero() {
		return true
	}
	espera := esperaBase << p.Intentos
	if espera > esperaTope || espera <= 0 {
		espera = esperaTope
	}
	return !ahora.UTC().Before(p.Ultimo.Add(espera))
}

func (q *Cola) leer() ([]Pendiente, error) {
	b, err := os.ReadFile(q.ruta) // #nosec G304 -- la ruta la fija el operador al construir la Cola
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("no puedo leer la cola de anclajes %s: %w", q.ruta, err)
	}
	if len(b) == 0 {
		return nil, nil
	}
	var ps []Pendiente
	if err := json.Unmarshal(b, &ps); err != nil {
		return nil, fmt.Errorf("la cola de anclajes %s esta corrupta: %w; "+
			"son anclajes pendientes, no la borres sin mirarla", q.ruta, err)
	}
	return ps, nil
}

// escribir deja el fichero completo de una vez: temporal y rename, para que un
// corte a media escritura no deje la cola a medias.
func (q *Cola) escribir(ps []Pendiente) error {
	if ps == nil {
		ps = []Pendiente{}
	}
	b, err := json.MarshalIndent(ps, "", "  ")
	if err != nil {
		return fmt.Errorf("no puedo serializar la cola: %w", err)
	}
	if dir := filepath.Dir(q.ruta); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("no puedo crear %s: %w", dir, err)
		}
	}
	tmp := q.ruta + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("no puedo escribir la cola: %w", err)
	}
	if err := os.Rename(tmp, q.ruta); err != nil {
		return fmt.Errorf("no puedo cerrar la cola: %w", err)
	}
	return nil
}

// Reintentar pasa por los pendientes a los que les toca y los intenta sellar.
// Devuelve los que lo consiguieron; los que no, se quedan con un intento mas
// anotado y esperan al siguiente backoff.
//
// No falla si una TSA sigue caida: ese es el caso normal de esta funcion.
func (c *Cadena) Reintentar(ahora time.Time) ([]Resuelto, error) {
	if c.Cola == nil {
		return nil, nil
	}
	ps, err := c.Cola.Pendientes()
	if err != nil {
		return nil, err
	}
	var out []Resuelto
	for _, p := range ps {
		if !p.toca(ahora) {
			continue
		}
		hash, err := hex.DecodeString(p.Hash)
		if err != nil {
			return out, fmt.Errorf("hash ilegible en la cola (%q): %w", p.Hash, err)
		}
		token, err := c.Sellar(hash)
		if err != nil {
			// Sellar ya reencola si hace falta; aqui solo se anota el intento.
			if err := c.Cola.anotarIntento(p.Hash, ahora); err != nil {
				return out, err
			}
			continue
		}
		if err := c.Cola.Resolver(hash); err != nil {
			return out, err
		}
		out = append(out, Resuelto{Hash: hash, Token: token, Encolado: p.Encolado})
	}
	return out, nil
}
