package corpus

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// ErrURNDuplicado: dos directorios distintos declaran el mismo URN de paquete.
//
// No es una mania de higiene. El URN es la IDENTIDAD del paquete: es lo que
// apunta el expediente junto al digest, lo que dice "quien me pide este dato" y
// lo que resuelve una equivalencia entre marcos. Con dos paquetes compartiendo
// URN, un directorio de mas en el corpus (que es un arbol de ficheros que se
// copia y se sincroniza) se hace pasar por la norma de verdad, y quien resuelva
// por URN se lleva el que salga. Se para en la carga.
var ErrURNDuplicado = errors.New("dos paquetes distintos con el mismo urn")

// Cargar lee todos los paquetes de un directorio y los valida. Un paquete es un
// directorio con paquete.json dentro; nada mas hace falta para que el sistema
// entero lo tenga en cuenta.
func Cargar(raiz string) ([]*Paquete, error) {
	ents, err := os.ReadDir(raiz)
	if err != nil {
		return nil, err
	}
	var nombres []string
	for _, e := range ents {
		if e.IsDir() {
			nombres = append(nombres, e.Name())
		}
	}
	sort.Strings(nombres)

	var ps []*Paquete
	deQuien := map[string]string{} // urn -> directorio que ya lo declaro
	for _, n := range nombres {
		b, err := os.ReadFile(filepath.Join(raiz, n, "paquete.json")) // #nosec G304 -- raiz la fija el operador; n viene de ReadDir
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		var p Paquete
		if err := json.Unmarshal(b, &p); err != nil {
			return nil, fmt.Errorf("%s: %w", n, err)
		}
		if err := cargarDorados(filepath.Join(raiz, n, "pruebas"), &p); err != nil {
			return nil, fmt.Errorf("%s: %w", n, err)
		}
		if errs := p.Validar(); len(errs) > 0 {
			return nil, fmt.Errorf("%s: %d fallos de linter, el primero: %w", n, len(errs), errs[0])
		}
		if otro, ya := deQuien[p.URN]; ya {
			return nil, fmt.Errorf("%w: %s y %s declaran los dos %s. El urn identifica la "+
				"norma en el expediente y en las equivalencias, asi que dos no pueden "+
				"compartirlo: renombra el urn del paquete nuevo o quita el directorio "+
				"repetido", ErrURNDuplicado, otro, n, p.URN)
		}
		deQuien[p.URN] = n
		ps = append(ps, &p)
	}
	return ps, nil
}

// ErrEsperadoDeLaFormaVieja: un dorado escrito con el formato anterior, en el
// que `esperado` era UN objeto en vez de la lista exhaustiva.
//
// No carga, y no se traduce en silencio. Un formato con dos formas vivas es un
// formato en el que gana la floja: la vieja afirma un solo vencimiento y no
// dice nada de los demas, que es exactamente el agujero que la lista cierra.
var ErrEsperadoDeLaFormaVieja = errors.New("el esperado de un dorado ya no es un objeto, es una lista")

// cargarDorados lee los casos dorados de pruebas/*.json. Cada fichero es un
// dorado o una lista de dorados.
//
// LA FORMA DEL FICHERO SE DECIDE UNA VEZ, mirando el primer byte util. Antes
// habia dos unmarshal encadenados (probar lista, y si falla probar objeto), y
// eso tenia dos problemas: el error que se reportaba era SIEMPRE el del primer
// intento, asi que un fichero con un dorado suelto y un fallo dentro se
// explicaba con "no es una lista", que no es lo que pasaba; y la forma vieja
// del `esperado` moria con un mensaje de tipos de encoding/json que no dice
// como se migra. Una sola lectura del dato, y el diagnostico en su sitio.
func cargarDorados(dir string, p *Paquete) error {
	ents, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range ents {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name())) // #nosec G304 -- dir viene del paquete del operador
		if err != nil {
			return err
		}
		lista, err := leerFicheroDeDorados(b)
		if err != nil {
			return fmt.Errorf("pruebas/%s: %w", e.Name(), err)
		}
		p.Dorados = append(p.Dorados, lista...)
	}
	return nil
}

func leerFicheroDeDorados(b []byte) ([]Dorado, error) {
	var crudos []json.RawMessage
	switch primerByteUtil(b) {
	case '[':
		if err := json.Unmarshal(b, &crudos); err != nil {
			return nil, fmt.Errorf("empieza por '[' y no es una lista de dorados: %w", err)
		}
	case '{':
		crudos = []json.RawMessage{b}
	default:
		return nil, fmt.Errorf("un fichero de pruebas es un dorado ({...}) o una lista de " +
			"dorados ([...]), y este no empieza por ninguno de los dos")
	}
	out := make([]Dorado, 0, len(crudos))
	for i, crudo := range crudos {
		if err := rechazarFormaVieja(crudo); err != nil {
			return nil, fmt.Errorf("dorado %d: %w", i+1, err)
		}
		var d Dorado
		if err := json.Unmarshal(crudo, &d); err != nil {
			return nil, fmt.Errorf("dorado %d: %w", i+1, err)
		}
		out = append(out, d)
	}
	return out, nil
}

func primerByteUtil(b []byte) byte {
	for _, c := range b {
		switch c {
		case ' ', '\t', '\r', '\n', 0xEF, 0xBB, 0xBF: // espacios y BOM
			continue
		}
		return c
	}
	return 0
}

// rechazarFormaVieja mira el `esperado` SIN interpretarlo: si es un objeto, el
// dorado esta escrito con el formato anterior y se dice como se migra.
func rechazarFormaVieja(crudo json.RawMessage) error {
	var sonda struct {
		Caso     string          `json:"caso"`
		Esperado json.RawMessage `json:"esperado"`
	}
	if err := json.Unmarshal(crudo, &sonda); err != nil {
		return err
	}
	if primerByteUtil(sonda.Esperado) != '{' {
		return nil
	}
	return fmt.Errorf("%w (caso %q). El esperado de un dorado es ahora el conjunto COMPLETO "+
		"de vencimientos que el motor devuelve con esos hechos, y se comprueba en las dos "+
		"direcciones: ni uno de menos ni uno de mas. Migracion: envuelve lo que tenias en "+
		"una lista y anade las filas que faltan, una por hito, con su `hito` obligatorio.\n"+
		"  antes: \"esperado\": {\"hito\": \"h\", \"vence\": \"2026-01-01T00:00:00Z\"}\n"+
		"  ahora: \"esperado\": [{\"hito\": \"h\", \"vence\": \"2026-01-01T00:00:00Z\"}, "+
		"{\"hito\": \"otro\", \"estado\": \"pendiente de hecho\"}]\n"+
		"  Para ver que devuelve hoy el motor: plazum explain, o el error de este mismo "+
		"ejecutor, que lista los hitos que sobran",
		ErrEsperadoDeLaFormaVieja, sonda.Caso)
}
