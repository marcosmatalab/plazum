package corpus

// Mismo texto legal, misma cadencia.
//
// EL CASO QUE LA TRAJO, y se caza antes de escribirlo. Al proponer los
// intervalos de las 34 cadencias sin numero del anexo del Reglamento de
// Ejecucion (UE) 2024/2690, los puntos 12.2.3 y 12.3.3 salieron a veinticuatro y
// a doce meses. Tienen el MISMO TEXTO LEGAL, palabra por palabra, y son puntos
// adyacentes de la misma seccion: lo unico que cambia es de que politica hablan.
// Nada en el paquete lo habria dicho, y un lector que abra las dos fichas
// seguidas encuentra la contradiccion en un minuto.
//
// POR QUE ES LINTABLE Y NO UNA REVISION. Porque no hay que entender el texto
// para verlo: basta comparar. Es de la familia mas barata que hay, la que se
// decide con una igualdad, y esas son justo las que no deberian estar esperando
// a que alguien las lea.
//
// LO QUE NO DICE ESTA REGLA. No dice que dos obligaciones con el mismo texto
// sean la misma obligacion, ni que no puedan tener cadencias distintas: dice que
// si las tienen, hay que ESCRIBIR POR QUE. La diferencia entre una decision y un
// descuido es exactamente esa frase.

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	// ErrCadenciaGemelaDiscrepante: dos obligaciones con el mismo texto legal y
	// distinta cadencia u origen, sin decir por que.
	ErrCadenciaGemelaDiscrepante = errors.New("mismo texto legal con cadencias distintas")
)

// minimoCadenciaDistintaPorque: un argumento, no una etiqueta. Mismo suelo que
// la justificacion del intervalo, porque es la misma clase de afirmacion.
const minimoCadenciaDistintaPorque = 60

// huellaDeTexto normaliza y resume el texto legal.
//
// Se normalizan los espacios ANTES de comparar. Sin eso, un salto de linea de
// mas convierte dos textos identicos en distintos y la regla se apaga sola sin
// que nadie se entere, que es el peor modo de fallo de una comprobacion: da
// verde y parece que ha mirado.
func huellaDeTexto(s string) string {
	n := strings.Join(strings.Fields(strings.ToLower(s)), " ")
	if n == "" {
		return ""
	}
	h := sha256.Sum256([]byte(n))
	return hex.EncodeToString(h[:8])
}

// validarCadenciasGemelas exige que dos obligaciones con el mismo texto legal
// lleven la misma cadencia y el mismo origen, o que TODAS digan por que no.
func (p *Paquete) validarCadenciasGemelas(anotar func(error)) {
	type fila struct {
		id       string
		cadencia string
		origen   string
		porque   string
	}
	grupos := map[string][]fila{}
	for _, o := range p.Obligaciones {
		t := o.Temporalidad
		if t == nil || t.Primitiva != "periodica" {
			continue
		}
		h := huellaDeTexto(o.TextoLegal)
		if h == "" {
			// Sin texto legal no hay nada que comparar. Un paquete referencial
			// puede no traerlo, y ahi la regla no aplica en vez de aplicar
			// sobre el vacio, que agrupara todo con todo.
			continue
		}
		grupos[h] = append(grupos[h], fila{
			id: o.ID, cadencia: t.Cadencia, origen: strings.TrimSpace(t.OrigenDelIntervalo),
			porque: strings.TrimSpace(t.CadenciaDistintaPorque),
		})
	}

	huellas := make([]string, 0, len(grupos))
	for h := range grupos {
		huellas = append(huellas, h)
	}
	sort.Strings(huellas) // orden estable: recorrer un mapa no tiene orden

	for _, h := range huellas {
		g := grupos[h]
		if len(g) < 2 {
			continue
		}
		cad, org := map[string]bool{}, map[string]bool{}
		for _, f := range g {
			cad[f.cadencia] = true
			org[f.origen] = true
		}
		if len(cad) == 1 && len(org) == 1 {
			continue
		}
		ids := make([]string, 0, len(g))
		for _, f := range g {
			ids = append(ids, f.id)
		}
		sort.Strings(ids)
		for _, f := range g {
			if len(f.porque) >= minimoCadenciaDistintaPorque {
				continue
			}
			anotar(fmt.Errorf("%w: %s/%s comparte texto legal palabra por palabra con %s y no "+
				"comparte %s. Dos obligaciones con el mismo texto y distinto reloj se leen como "+
				"una contradiccion en cuanto alguien abre las dos fichas seguidas. Si la "+
				"diferencia es deliberada, escribela en `cadencia_distinta_porque` (minimo %d "+
				"caracteres) EN TODAS las del grupo: con dos obligaciones no hay una canonica, "+
				"asi que decidir cual es la normal seria elegir por el autor",
				ErrCadenciaGemelaDiscrepante, p.URN, f.id, otrosDelGrupo(ids, f.id),
				loQueNoComparten(cad, org), minimoCadenciaDistintaPorque))
		}
	}
}

func otrosDelGrupo(ids []string, yo string) string {
	var out []string
	for _, i := range ids {
		if i != yo {
			out = append(out, i)
		}
	}
	return strings.Join(out, ", ")
}

func loQueNoComparten(cad, org map[string]bool) string {
	var q []string
	if len(cad) > 1 {
		q = append(q, "la cadencia ("+strings.Join(clavesOrdenadas(cad), " contra ")+")")
	}
	if len(org) > 1 {
		q = append(q, "el origen del intervalo ("+strings.Join(clavesOrdenadas(org), " contra ")+")")
	}
	return strings.Join(q, " ni ")
}

func clavesOrdenadas(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		if k == "" {
			k = "(sin declarar)"
		}
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
