package aplicabilidad

import (
	"errors"
	"fmt"
	"sort"
)

// El espacio de nombres de los predicados: lo que impide que anadir la norma 31
// rompa la 12.
//
// EL PROBLEMA. Hasta ahora todos los predicados de todos los paquetes vivian en
// un solo espacio plano. Dos paquetes que declararan `en_ambito` con
// significados distintos derivaban el uno sobre los hechos del otro, en
// silencio, y el resultado era una obligacion que aplica a quien no le aplica.
// El campo Exporta existia y el godoc afirmaba que lo evitaba: no era cierto, la
// comprobacion tenia las dos ramas haciendo lo mismo. Con un solo paquete con
// reglas eso era teorico. Con treinta, no.
//
// LA REGLA, y tiene tres casos porque los predicados no son todos iguales:
//
//	COMUNES      aplica, desplaza, equivale. Son el vocabulario de SALIDA del
//	             corpus entero, el que consume el resto del producto. Globales
//	             siempre.
//	DE ENTRADA   los que el paquete USA y no DEFINE: ambito, categoria, maneja,
//	             nivel_dimension. Vienen de los hechos que declara el sujeto al
//	             describir su alcance, y ese vocabulario es compartido POR
//	             DISENO: corpus.EsquemaUI ya funde los atributos que se llaman
//	             igual, porque "un dato pedido por tres normas se pregunta una
//	             vez". Globales.
//	LOCALES      los que el paquete DEFINE y no exporta: nivel_max, en_ambito y
//	             cualquier paso intermedio de su razonamiento. Se prefijan con
//	             el paquete. Son suyos y no los ve nadie mas.
//
// Y EXPORTAR es el acto deliberado de publicar una derivacion propia al espacio
// comun, para que otro paquete pueda encadenar sobre ella. Un paquete que
// exporta se hace responsable de ese nombre: por eso dos paquetes no pueden
// exportar el mismo, y por eso no se puede exportar lo que no se define.
//
// QUE CAMBIA respecto de antes. El encadenamiento entre paquetes pasa a exigir
// Exporta EXPLICITO. Un paquete que derivaba sobre lo que otro producia sin
// declararlo deja de hacerlo, y eso es lo que se quiere: el acoplamiento entre
// dos normas tiene que estar escrito en el fichero de datos, no ocurrir porque
// dos autores eligieron la misma palabra.

var (
	ErrHechoQueNadieUsa   = errors.New("hecho con un predicado que un paquete define como propio")
	ErrExportaSinDefinir  = errors.New("exporta un predicado que no define")
	ErrExportaComun       = errors.New("exporta un predicado que ya es comun")
	ErrExportaColisionado = errors.New("dos paquetes exportan el mismo predicado")
)

// definidos son los predicados que aparecen en la cabeza de alguna regla del
// programa. Es lo que separa "lo que este paquete produce" de "lo que consume".
func (p Programa) definidos() map[string]bool {
	d := map[string]bool{}
	for _, r := range p.Reglas {
		d[r.Cabeza.Pred] = true
	}
	return d
}

// Local dice como se llama un predicado de este paquete una vez aislado.
func (p Programa) Local(pred string) string { return p.Paquete + "." + pred }

// ComprobarExportaciones valida el bloque Exporta contra las reglas.
//
// Las dos comprobaciones son de la misma familia: una exportacion es una
// promesa, y una promesa que no se puede cumplir es peor que no hacerla.
func (p Programa) ComprobarExportaciones() error {
	def := p.definidos()
	for _, e := range p.Exporta {
		if comunes[e] {
			return fmt.Errorf("%s: %w (%s). Los comunes ya son globales para todos los "+
				"paquetes; quitalo de exporta", p.Paquete, ErrExportaComun, e)
		}
		if !def[e] {
			return fmt.Errorf("%s: %w (%s). Exportar es publicar lo que TU derivas, y ninguna "+
				"regla de este paquete tiene %s en la cabeza. O te falta la regla, o sobra la "+
				"exportacion", p.Paquete, ErrExportaSinDefinir, e, e)
		}
	}
	return nil
}

// ConEspacioDeNombres devuelve una copia del programa con sus predicados
// LOCALES prefijados por el paquete. No modifica el original.
func (p Programa) ConEspacioDeNombres() Programa {
	exp := map[string]bool{}
	for _, e := range p.Exporta {
		exp[e] = true
	}
	def := p.definidos()

	// esLocal: lo define este paquete, no es comun y no lo publica.
	esLocal := func(pred string) bool { return def[pred] && !comunes[pred] && !exp[pred] }

	renombrar := func(a Atomo) Atomo {
		if !esLocal(a.Pred) {
			return a
		}
		b := Atomo{Pred: p.Local(a.Pred), Args: append([]Termino(nil), a.Args...)}
		return b
	}

	fuera := Programa{Paquete: p.Paquete, Exporta: append([]string(nil), p.Exporta...)}
	for _, r := range p.Reglas {
		n := r
		n.Cabeza = renombrar(r.Cabeza)
		n.Cuerpo = nil
		for _, a := range r.Cuerpo {
			n.Cuerpo = append(n.Cuerpo, renombrar(a))
		}
		n.Negados = nil
		for _, a := range r.Negados {
			n.Negados = append(n.Negados, renombrar(a))
		}
		fuera.Reglas = append(fuera.Reglas, n)
	}
	return fuera
}

// registrarExportaciones comprueba que ningun otro programa ya cargado exporta
// el mismo predicado.
//
// Es un error y no un aviso a proposito. Un predicado del espacio comun tiene un
// dueno: el paquete que lo define y lo publica. Dos duenos significa que las
// reglas de uno derivan sobre los hechos del otro sin que nadie lo haya
// decidido, que es exactamente el fallo que este fichero existe para impedir. Si
// dos normas necesitan de verdad compartir un concepto, una lo exporta y la otra
// lo consume sin definirlo.
func (m *Motor) registrarExportaciones(p Programa) error {
	if m.exportados == nil {
		m.exportados = map[string]string{}
	}
	nuevos := append([]string(nil), p.Exporta...)
	sort.Strings(nuevos)
	for _, e := range nuevos {
		if duenyo, ok := m.exportados[e]; ok && duenyo != p.Paquete {
			return fmt.Errorf("%w: %q lo exportan %s y %s. Un predicado del espacio comun tiene "+
				"un dueno, el que lo define y lo publica. Si las dos normas necesitan compartir "+
				"el concepto, una lo exporta y la otra lo consume sin definirlo; si no lo "+
				"necesitan, la segunda lo quita de exporta y pasa a ser suyo (%s)",
				ErrExportaColisionado, e, duenyo, p.Paquete, p.Local(e))
		}
		m.exportados[e] = p.Paquete
	}
	return nil
}

// registrarLocales anota que predicados quedaron aislados y de quien son, con su
// nombre ORIGINAL. Sirve para un guardia que solo se puede montar aqui.
func (m *Motor) registrarLocales(p Programa) {
	if m.locales == nil {
		m.locales = map[string]string{}
	}
	exp := map[string]bool{}
	for _, e := range p.Exporta {
		exp[e] = true
	}
	for pred := range p.definidos() {
		if !comunes[pred] && !exp[pred] {
			m.locales[pred] = p.Paquete
		}
	}
}

// comprobarHechosContraLocales caza la trampa que el aislamiento trae puesta.
//
// EL CASO. Un paquete escribe la regla `provee_a(A, C) :- provee_a(A, B),
// provee_a(B, C)` para cerrar transitivamente la cadena de suministro. Como
// DEFINE provee_a, provee_a pasa a ser suyo. Pero los hechos que declara el
// sujeto se afirman con el nombre global provee_a, asi que YA NO ALIMENTAN esa
// regla: el paquete deriva sobre un predicado vacio y no aplica nada. En
// silencio, que es lo peor.
//
// Y el fallo de fondo no es de nombres, es de modelado: un paquete NO debe
// redefinir un predicado que el sujeto aporta. Si quiere cerrar transitivamente
// lo que el sujeto declara, deriva un predicado PROPIO a partir de el
// (proveedor_de(A, C) :- provee_a(A, C), y la transitiva sobre proveedor_de).
// Es la misma regla que ya rige en el dialecto textual, donde un paquete no
// puede declarar hechos: los hechos son del sujeto y el paquete razona sobre
// ellos, no los reescribe.
//
// Se comprueba al evaluar y no al cargar porque hasta que no hay hechos no se
// sabe cuales se van a afirmar.
func (m *Motor) comprobarHechosContraLocales() error {
	if len(m.locales) == 0 {
		return nil
	}
	var choques []string
	vistos := map[string]bool{}
	for _, k := range m.orden {
		h := m.hechos[k]
		duenyo, ok := m.locales[h.Pred]
		if !ok || vistos[h.Pred] {
			continue
		}
		vistos[h.Pred] = true
		choques = append(choques, fmt.Sprintf("%s (lo define %s)", h.Pred, duenyo))
	}
	if len(choques) == 0 {
		return nil
	}
	sort.Strings(choques)
	return fmt.Errorf("%w: %v. Esos hechos NO alimentan las reglas del paquete que los "+
		"redefine, asi que no derivarian nada y nadie se enteraria. Arreglo: que el paquete "+
		"derive un predicado PROPIO a partir del hecho en vez de redefinirlo, o que lo exporte "+
		"si de verdad quiere ser su dueno", ErrHechoQueNadieUsa, choques)
}
