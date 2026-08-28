package main

// El arranque en diez segundos: tres banderas, un perfil de SUPUESTOS.
//
// LO QUE SE MIDIO ANTES DE ESCRIBIR ESTO, y decide todo el diseno. Con el corpus
// de hoy, `--pais`, `--sector` y `--empleados` no derivan casi nada por si
// solos, y no por un fallo del motor: 27 de los 33 paquetes instalados NO
// declaran reglas de aplicabilidad, asi que sobre 21 de los relojes del corpus
// no hay nada que preguntar. Y los que si declaran piden hechos que un sector no
// implica: `designado(operador_servicios_esenciales)` es una designacion de la
// autoridad, no una consecuencia de dedicarse a algo.
//
// Habia dos salidas y una es mentira:
//
//	la mentira   cablear "sector salud implica tal norma" y ensenar una lista
//	             larga que impresiona. Rompe el invariante 2, rompe el build, y
//	             sobre todo convierte una conjetura en una obligacion, que es lo
//	             que este producto existe para no hacer.
//	la buena     que las banderas monten un PERFIL de hechos supuestos, cada uno
//	             con su porque escrito, y que todo lo que salga de ahi vaya
//	             marcado como supuesto hasta la ultima fila. Lo que el perfil NO
//	             supone se dice tambien, que es la mitad util.
//
// Los perfiles son DATOS empotrados (paquete perfiles), no codigo.

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/marcosmatalab/plazum/perfiles"
)

// perfilPedido es lo que teclea quien arranca.
type perfilPedido struct {
	Pais      string
	Sector    string
	Empleados int
}

// hechoSupuesto es un hecho que el perfil afirma en nombre de la organizacion,
// con el porque delante. El `porque` no es documentacion: es lo que se le ensena
// a quien lea el calendario para que sepa que esta mirando.
type hechoSupuesto struct {
	Pred      string   `json:"pred"`
	Args      []string `json:"args"`
	PorQue    string   `json:"porque"`
	Confianza string   `json:"confianza"`
}

type bandaDeEmpleados struct {
	Desde  int             `json:"desde"`
	Hechos []hechoSupuesto `json:"hechos"`
}

type perfil struct {
	ID       string             `json:"id"`
	Pais     string             `json:"pais"`
	Sector   string             `json:"sector"`
	Nombre   string             `json:"nombre"`
	Aviso    string             `json:"aviso"`
	Hechos   []hechoSupuesto    `json:"hechos"`
	Fechas   map[string]string  `json:"fechas"`
	Bandas   []bandaDeEmpleados `json:"bandas"`
	NoSupone []string           `json:"no_supone"`
}

// sujetoDelPerfil es el nombre con el que las reglas hablan de la organizacion
// en el modo de arranque. Se escribe asi, feo y evidente, para que quien vea la
// salida sepa que no esta mirando su empresa.
const sujetoDelPerfil = "perfil"

// marcaDeSujeto es lo que los ficheros de perfil escriben donde va el sujeto.
const marcaDeSujeto = "$sujeto"

func cargarPerfiles() ([]perfil, error) {
	entradas, err := fs.ReadDir(perfiles.Ficheros, ".")
	if err != nil {
		return nil, err
	}
	var out []perfil
	for _, e := range entradas {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := perfiles.Ficheros.ReadFile(e.Name())
		if err != nil {
			return nil, err
		}
		var p perfil
		if err := json.Unmarshal(b, &p); err != nil {
			return nil, fmt.Errorf("el perfil %s no es legible: %w", e.Name(), err)
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// alcanceDePerfil monta un alcance a partir de las tres banderas.
//
// El error de "no hay perfil" enumera los que SI hay, porque un mensaje que dice
// "sector desconocido" y se calla la lista obliga a leer el codigo fuente, que
// es exactamente lo que la tercera pasada de este proyecto persigue.
func alcanceDePerfil(pedido perfilPedido) (alcance, error) {
	ps, err := cargarPerfiles()
	if err != nil {
		return alcance{}, err
	}
	if pedido.Pais == "" || pedido.Sector == "" {
		return alcance{}, fmt.Errorf("el arranque necesita --pais Y --sector.%s", listaDePerfiles(ps))
	}
	var elegido *perfil
	for i := range ps {
		if strings.EqualFold(ps[i].Pais, pedido.Pais) && strings.EqualFold(ps[i].Sector, pedido.Sector) {
			elegido = &ps[i]
			break
		}
	}
	if elegido == nil {
		return alcance{}, fmt.Errorf("no hay perfil para --pais=%s --sector=%s.%s",
			pedido.Pais, pedido.Sector, listaDePerfiles(ps))
	}

	al := alcance{
		Organizacion: elegido.Nombre + " (perfil de arranque)",
		Sujeto:       sujetoDelPerfil,
		Descripcion:  elegido.Aviso,
		Fechas:       elegido.Fechas,
	}
	anadir := func(hs []hechoSupuesto) {
		for _, h := range hs {
			args := make([]string, len(h.Args))
			for i, a := range h.Args {
				if a == marcaDeSujeto {
					a = sujetoDelPerfil
				}
				args[i] = a
			}
			al.Hechos = append(al.Hechos, struct {
				Pred string   `json:"pred"`
				Args []string `json:"args"`
			}{Pred: h.Pred, Args: args})
		}
	}
	anadir(elegido.Hechos)
	for _, b := range elegido.Bandas {
		if pedido.Empleados >= b.Desde {
			anadir(b.Hechos)
		}
	}
	return al, nil
}

func listaDePerfiles(ps []perfil) string {
	if len(ps) == 0 {
		return "\n  Y no hay ningun perfil empotrado, que es un fallo del binario."
	}
	var b strings.Builder
	b.WriteString("\n  Los que hay:\n")
	for _, p := range ps {
		fmt.Fprintf(&b, "    --pais=%s --sector=%-22s %s\n", p.Pais, p.Sector, p.Nombre)
	}
	b.WriteString("  Y si lo que quieres es lo tuyo de verdad, --alcance con tus respuestas.")
	return b.String()
}

// explicarPerfil escribe lo que el perfil supone y lo que NO supone. La segunda
// mitad es la que evita la llamada de soporte: quien lee "no te sale DORA" sin
// saber por que asume que el producto no lo tiene.
func explicarPerfil(b *strings.Builder, pedido perfilPedido) {
	ps, err := cargarPerfiles()
	if err != nil {
		return
	}
	for _, p := range ps {
		if !strings.EqualFold(p.Pais, pedido.Pais) || !strings.EqualFold(p.Sector, pedido.Sector) {
			continue
		}
		b.WriteString("LO QUE ESTE PERFIL SUPONE\n\n")
		escribir := func(hs []hechoSupuesto) {
			for _, h := range hs {
				fmt.Fprintf(b, "    %s(%s)   confianza %s\n", h.Pred,
					strings.Join(h.Args, ", "), h.Confianza)
				fmt.Fprintf(b, "        %s\n", h.PorQue)
			}
		}
		escribir(p.Hechos)
		for _, banda := range p.Bandas {
			if pedido.Empleados >= banda.Desde {
				fmt.Fprintf(b, "    (por tener %d empleados o mas)\n", banda.Desde)
				escribir(banda.Hechos)
			}
		}
		if len(p.NoSupone) > 0 {
			b.WriteString("\nLO QUE NO SUPONE, Y POR TANTO NO VERAS AQUI\n\n")
			for _, n := range p.NoSupone {
				fmt.Fprintf(b, "    %s\n", n)
			}
		}
		b.WriteString("\n")
		return
	}
}
