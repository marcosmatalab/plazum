package corpus

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

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
	for _, n := range nombres {
		b, err := os.ReadFile(filepath.Join(raiz, n, "paquete.json"))
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
		if errs := p.Validar(); len(errs) > 0 {
			return nil, fmt.Errorf("%s: %d fallos de linter, el primero: %w", n, len(errs), errs[0])
		}
		ps = append(ps, &p)
	}
	return ps, nil
}
