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
		ps = append(ps, &p)
	}
	return ps, nil
}

// cargarDorados lee los casos dorados de pruebas/*.json. Cada fichero es un
// dorado o una lista de dorados.
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
		var lista []Dorado
		if err := json.Unmarshal(b, &lista); err != nil {
			var uno Dorado
			if err2 := json.Unmarshal(b, &uno); err2 != nil {
				return fmt.Errorf("pruebas/%s: ni dorado ni lista: %w", e.Name(), err)
			}
			lista = []Dorado{uno}
		}
		p.Dorados = append(p.Dorados, lista...)
	}
	return nil
}
