// Package perimetro es la multi-entidad: grupo, filiales, unidades. Scoping
// computado, no instancias aisladas: el perimetro es un atributo del sujeto,
// el corpus se hereda hacia abajo y el roll-up se calcula hacia arriba.
package perimetro

import "fmt"

type Perimetro struct {
	ID, Nombre string
	Padre      string // vacio = raiz
	Atributos  map[string]string
}

// Arbol valida y consulta la jerarquia.
type Arbol struct {
	nodos map[string]Perimetro
}

// NuevoArbol rechaza duplicados, padres inexistentes y ciclos: un arbol de
// entidades legales con un ciclo no es un error raro en produccion, es un
// error de carga.
func NuevoArbol(ps []Perimetro) (*Arbol, error) {
	a := &Arbol{nodos: map[string]Perimetro{}}
	for _, p := range ps {
		if _, dup := a.nodos[p.ID]; dup {
			return nil, fmt.Errorf("perimetro %s duplicado", p.ID)
		}
		a.nodos[p.ID] = p
	}
	for _, p := range a.nodos {
		if p.Padre == "" {
			continue
		}
		if _, ok := a.nodos[p.Padre]; !ok {
			return nil, fmt.Errorf("perimetro %s: padre %s no existe", p.ID, p.Padre)
		}
		// deteccion de ciclo subiendo con limite
		visto := map[string]bool{p.ID: true}
		for cur := p.Padre; cur != ""; cur = a.nodos[cur].Padre {
			if visto[cur] {
				return nil, fmt.Errorf("ciclo de perimetros en %s", cur)
			}
			visto[cur] = true
		}
	}
	return a, nil
}

// Cadena devuelve el perimetro y sus ancestros hasta la raiz (para heredar
// paquetes y politicas: lo instalado en el padre aplica a los hijos).
func (a *Arbol) Cadena(id string) ([]string, error) {
	p, ok := a.nodos[id]
	if !ok {
		return nil, fmt.Errorf("perimetro %s no existe", id)
	}
	out := []string{p.ID}
	for cur := p.Padre; cur != ""; cur = a.nodos[cur].Padre {
		out = append(out, cur)
	}
	return out, nil
}

// Hereda dice si lo instalado en `en` aplica al perimetro `para`: cierto si
// `en` esta en la cadena de ancestros de `para` (o es el mismo).
func (a *Arbol) Hereda(para, en string) (bool, error) {
	cadena, err := a.Cadena(para)
	if err != nil {
		return false, err
	}
	for _, c := range cadena {
		if c == en {
			return true, nil
		}
	}
	return false, nil
}

// RollUp agrega recuentos por perimetro hacia arriba: lo que cuenta una filial
// cuenta tambien en el grupo. La entrada es un recuento por perimetro hoja; la
// salida incluye cada ancestro con la suma de su subarbol.
func (a *Arbol) RollUp(recuentos map[string]int) (map[string]int, error) {
	out := map[string]int{}
	for id, n := range recuentos {
		cadena, err := a.Cadena(id)
		if err != nil {
			return nil, err
		}
		for _, c := range cadena {
			out[c] += n
		}
	}
	return out, nil
}
