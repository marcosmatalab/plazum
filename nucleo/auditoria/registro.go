package auditoria

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// EL PROGRAMA DE AUDITORIA EN DISCO.
//
// # Que faltaba, y era la ultima de las tres
//
// El acta de revision por la direccion se compone de tres fuentes. La campana
// de accesos se podia leer desde el principio; el registro de incidentes se
// cerro el 02-09-2026; esta era la que quedaba, y con ella la pantalla que mas
// circula del producto deja de tener ninguna seccion diciendo «esta fuente no
// esta conectada» por falta de lector.
//
// # Se reconstruye REPLICANDO, igual que los incidentes
//
// `Reconstruir` no escribe en los campos privados: llama a `Abrir` y despues a
// `Auditar`, `Diferir`, `Anotar` y `Cerrar` con los mismos hechos, en un orden
// que respeta sus dependencias. Un programa leido de disco pasa POR LAS MISMAS
// reglas que uno construido a mano: que la unidad de un hallazgo este en el
// alcance, que el hallazgo que se cierra exista, que un diferimiento traiga
// motivo. Rellenar campos privados desde JSON convertiria el fichero en la
// autoridad y dejaria las reglas fuera.
//
// # El orden de reproduccion NO es el del fichero, y esto importa
//
// Los hechos dependen unos de otros: un hallazgo apunta a una sesion y a una
// unidad, y un cierre apunta a un hallazgo. Si se reprodujeran en el orden en
// que aparecen en el documento, un fichero perfectamente valido escrito en otro
// orden fallaria al leerse, y el error hablaria de un hallazgo inexistente en
// vez de decir que el orden no era el esperado.
//
// Se reproducen POR CLASE y en el orden de sus dependencias: sesiones,
// diferimientos, hallazgos, cierres. Dentro de cada clase se respeta el orden
// del fichero, que es el que da el escritor.

var (
	// ErrProgramaIlegible: el documento no es lo que dice ser.
	ErrProgramaIlegible = errors.New("programa de auditoria ilegible")
	// ErrProgramaSinVersion: falta la version del formato.
	ErrProgramaSinVersion = errors.New("programa de auditoria sin version")
	// ErrProgramaVersionDesconocida: version que este binario no lee.
	ErrProgramaVersionDesconocida = errors.New("version de programa desconocida")
	// ErrClaseDesconocida: una clase de hallazgo que no existe.
	ErrClaseDesconocida = errors.New("clase de hallazgo desconocida")
	// ErrInstanteIlegible: un instante presente y no interpretable.
	ErrInstanteIlegible = errors.New("instante ilegible")
)

// VersionDelPrograma es la unica que este binario lee y escribe. Se exige y no
// se supone, por lo mismo que en el registro de incidentes: un fichero sin
// version se leeria con las reglas de cada momento, en silencio.
const VersionDelPrograma = 1

type unidadEnDisco struct {
	Paquete    string `json:"paquete"`
	Version    string `json:"version"`
	Obligacion string `json:"obligacion"`
	Titulo     string `json:"titulo,omitempty"`
}

type sesionEnDisco struct {
	ID       string   `json:"id"`
	Auditor  string   `json:"auditor"`
	Cuando   string   `json:"cuando"`
	Unidades []string `json:"unidades"`
	Alcance  string   `json:"alcance,omitempty"`
}

type hallazgoEnDisco struct {
	ID     string `json:"id"`
	Sesion string `json:"sesion"`
	Unidad string `json:"unidad"`
	// LA CLASE VIAJA POR SU NOMBRE, no por el numero del iota. Insertar una
	// clase nueva en medio reinterpretaria en silencio los ficheros escritos, y
	// una «no conformidad mayor» pasaria a leerse como «menor». Nadie firma el
	// orden de un iota, y aqui la diferencia entre mayor y menor es lo que un
	// consejo mira primero.
	Clase  string `json:"clase"`
	Texto  string `json:"texto,omitempty"`
	Quien  string `json:"quien"`
	Cuando string `json:"cuando"`
}

type cierreEnDisco struct {
	Hallazgo string `json:"hallazgo"`
	Quien    string `json:"quien"`
	Cuando   string `json:"cuando"`
	Como     string `json:"como,omitempty"`
}

type diferimientoEnDisco struct {
	Unidad string `json:"unidad"`
	Quien  string `json:"quien"`
	Motivo string `json:"motivo"`
	Cuando string `json:"cuando"`
}

type arrastreEnDisco struct {
	SinAuditar map[string]int `json:"sin_auditar,omitempty"`
	Abiertos   map[string]int `json:"abiertos,omitempty"`
	DeCiclo    string         `json:"de_ciclo,omitempty"`
	Salidas    []string       `json:"salidas,omitempty"`
}

type programaEnDisco struct {
	Version int    `json:"version"`
	ID      string `json:"id"`
	Ciclo   struct {
		Nombre string `json:"nombre"`
		Desde  string `json:"desde"`
		Hasta  string `json:"hasta"`
	} `json:"ciclo"`
	Alcance       []unidadEnDisco       `json:"alcance"`
	Arrastre      arrastreEnDisco       `json:"arrastre,omitempty"`
	Sesiones      []sesionEnDisco       `json:"sesiones,omitempty"`
	Diferimientos []diferimientoEnDisco `json:"diferimientos,omitempty"`
	Hallazgos     []hallazgoEnDisco     `json:"hallazgos,omitempty"`
	Cierres       []cierreEnDisco       `json:"cierres,omitempty"`
}

func clasePorNombre(n string) (Clase, bool) {
	for i, nombre := range nombresDeClase {
		if nombre == n {
			return Clase(i), true
		}
	}
	return 0, false
}

// instanteDeDisco lee un instante RFC3339. Un instante que no se entiende es un
// error, NUNCA el cero: el cero de time.Time es el ano 1, y en un programa de
// auditoria eso significa una sesion celebrada hace dos mil anos.
func instanteDeDisco(campo, v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, fmt.Errorf("%w: %s esta vacio", ErrInstanteIlegible, campo)
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: %s = %q no es un instante RFC3339 "+
			"(2026-09-02T09:00:00Z)", ErrInstanteIlegible, campo, v)
	}
	return t.UTC(), nil
}

// Reconstruir lee un programa de auditoria de un documento.
func Reconstruir(datos []byte) (*Programa, error) {
	var doc programaEnDisco
	if err := json.Unmarshal(datos, &doc); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProgramaIlegible, err)
	}
	if doc.Version == 0 {
		return nil, fmt.Errorf("%w: el documento no dice con que version del formato se "+
			"escribio. Arreglo: anade \"version\": %d",
			ErrProgramaSinVersion, VersionDelPrograma)
	}
	if doc.Version != VersionDelPrograma {
		return nil, fmt.Errorf("%w: el documento dice version %d y este binario lee la %d",
			ErrProgramaVersionDesconocida, doc.Version, VersionDelPrograma)
	}

	desde, err := instanteDeDisco("ciclo.desde", doc.Ciclo.Desde)
	if err != nil {
		return nil, err
	}
	hasta, err := instanteDeDisco("ciclo.hasta", doc.Ciclo.Hasta)
	if err != nil {
		return nil, err
	}
	alcance := make([]Unidad, 0, len(doc.Alcance))
	for _, u := range doc.Alcance {
		// CONVERSION Y NO LITERAL CAMPO A CAMPO, y no es solo lo que pide
		// staticcheck: la conversion NO COMPILA el dia que Unidad gane un campo
		// que unidadEnDisco no tenga. Es la misma guarda que
		// TestNingunCampoDelDominioSePierdeAlEscribirlo, pero en el compilador,
		// que es mejor sitio. Solo vale donde los dos tipos son identicos; en
		// los que llevan instantes (Sesion, Hallazgo) no se puede, porque en
		// disco son cadenas, y ahi la guarda la da el test.
		alcance = append(alcance, Unidad(u))
	}
	p, err := Abrir(doc.ID, Ciclo{Nombre: doc.Ciclo.Nombre, Desde: desde, Hasta: hasta},
		alcance, Arrastre(doc.Arrastre))
	if err != nil {
		return nil, fmt.Errorf("abriendo el programa del documento: %w", err)
	}

	// EL ORDEN DE REPRODUCCION ES EL DE LAS DEPENDENCIAS, no el del fichero.
	for i, s := range doc.Sesiones {
		cuando, err := instanteDeDisco(fmt.Sprintf("sesiones[%d].cuando", i), s.Cuando)
		if err != nil {
			return nil, err
		}
		if err := p.Auditar(Sesion{
			ID: s.ID, Auditor: s.Auditor, Cuando: cuando,
			Unidades: s.Unidades, Alcance: s.Alcance,
		}); err != nil {
			return nil, fmt.Errorf("reproduciendo la sesion %q: %w", s.ID, err)
		}
	}
	for i, d := range doc.Diferimientos {
		cuando, err := instanteDeDisco(fmt.Sprintf("diferimientos[%d].cuando", i), d.Cuando)
		if err != nil {
			return nil, err
		}
		if err := p.Diferir(Diferimiento{
			Unidad: d.Unidad, Quien: d.Quien, Motivo: d.Motivo, Cuando: cuando,
		}); err != nil {
			return nil, fmt.Errorf("reproduciendo el diferimiento de %q: %w", d.Unidad, err)
		}
	}
	for i, h := range doc.Hallazgos {
		clase, ok := clasePorNombre(h.Clase)
		if !ok {
			return nil, fmt.Errorf("%w: el hallazgo %q dice ser %q, y las que hay son %v",
				ErrClaseDesconocida, h.ID, h.Clase, nombresDeClase)
		}
		cuando, err := instanteDeDisco(fmt.Sprintf("hallazgos[%d].cuando", i), h.Cuando)
		if err != nil {
			return nil, err
		}
		if err := p.Anotar(Hallazgo{
			ID: h.ID, Sesion: h.Sesion, Unidad: h.Unidad,
			Clase: clase, Texto: h.Texto, Quien: h.Quien, Cuando: cuando,
		}); err != nil {
			return nil, fmt.Errorf("reproduciendo el hallazgo %q: %w", h.ID, err)
		}
	}
	for i, c := range doc.Cierres {
		cuando, err := instanteDeDisco(fmt.Sprintf("cierres[%d].cuando", i), c.Cuando)
		if err != nil {
			return nil, err
		}
		if err := p.Cerrar(CierreDeHallazgo{
			Hallazgo: c.Hallazgo, Quien: c.Quien, Cuando: cuando, Como: c.Como,
		}); err != nil {
			return nil, fmt.Errorf("reproduciendo el cierre del hallazgo %q: %w", c.Hallazgo, err)
		}
	}
	return p, nil
}

// Escribir serializa el programa al formato que lee Reconstruir.
//
// Existe por lo mismo que su hermano en incidentes: un lector sin escritor se
// prueba contra ficheros escritos a mano, que son una segunda implementacion
// del formato y siguen verdes cuando el formato cambia.
func Escribir(p *Programa) ([]byte, error) {
	if p == nil {
		return nil, fmt.Errorf("%w: no hay programa que escribir", ErrProgramaIlegible)
	}
	doc := programaEnDisco{Version: VersionDelPrograma, ID: p.ID()}
	c := p.Ciclo()
	doc.Ciclo.Nombre = c.Nombre
	doc.Ciclo.Desde = c.Desde.UTC().Format(time.RFC3339)
	doc.Ciclo.Hasta = c.Hasta.UTC().Format(time.RFC3339)
	for _, u := range p.Unidades() {
		doc.Alcance = append(doc.Alcance, unidadEnDisco(u))
	}
	doc.Arrastre = arrastreEnDisco(p.DelCicloAnterior())
	for _, s := range p.Sesiones() {
		doc.Sesiones = append(doc.Sesiones, sesionEnDisco{
			ID: s.ID, Auditor: s.Auditor, Cuando: s.Cuando.UTC().Format(time.RFC3339),
			Unidades: s.Unidades, Alcance: s.Alcance,
		})
	}
	for _, d := range p.Diferimientos() {
		doc.Diferimientos = append(doc.Diferimientos, diferimientoEnDisco{
			Unidad: d.Unidad, Quien: d.Quien, Motivo: d.Motivo,
			Cuando: d.Cuando.UTC().Format(time.RFC3339),
		})
	}
	for _, h := range p.Hallazgos() {
		doc.Hallazgos = append(doc.Hallazgos, hallazgoEnDisco{
			ID: h.ID, Sesion: h.Sesion, Unidad: h.Unidad, Clase: h.Clase.String(),
			Texto: h.Texto, Quien: h.Quien, Cuando: h.Cuando.UTC().Format(time.RFC3339),
		})
	}
	for _, cl := range p.Cierres() {
		doc.Cierres = append(doc.Cierres, cierreEnDisco{
			Hallazgo: cl.Hallazgo, Quien: cl.Quien,
			Cuando: cl.Cuando.UTC().Format(time.RFC3339), Como: cl.Como,
		})
	}
	return json.MarshalIndent(doc, "", "  ")
}
