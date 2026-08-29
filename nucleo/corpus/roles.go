package corpus

import (
	"errors"
	"fmt"
	"strings"
)

// LAS FIGURAS A LAS QUE SE ESCALA, declaradas por el paquete.
//
// POR QUE EXISTE ESTE FICHERO. El corpus tenia 53 escalones en diez paquetes
// apuntando a 14 nombres de rol, y no habia vocabulario en ningun sitio: ni en
// el formato, ni en el codigo, ni en SCIM, que tiene usuarios, grupos y
// jerarquia de mando y ningun concepto de "el responsable de seguridad". El
// linter comprobaba que el destinatario no estuviera vacio. El dia que el
// escalado envie, un correo a un nombre que la organizacion no asigno no llega
// a nadie, Y NO DA ERROR: da un escalon que no escala, que es la subfamilia
// "alcanzabilidad, no existencia" — la regla existe, se ve, y nadie puede
// satisfacerla.
//
// POR QUE NO ES UNA LISTA CERRADA EN GO, que era la tentacion. `clase_e2e` y
// `TipoRecurso` si lo son, pero aquellos son vocabularios DEL PRODUCTO. Un rol
// es una figura DE LA NORMA: el delegado de proteccion de datos es del art. 37
// del RGPD y el responsable de la seguridad del art. 13 del ENS. Cerrarlos en
// Go pondria normas en el codigo, que es el invariante 2 al reves. Van donde
// van las entidades y las preguntas: los declara el paquete.
//
// LAS DOS REGLAS DE FONDO, y las dos salieron de mirar el corpus:
//
//	(1) CADA FIGURA LLEVA SU ORIGEN, igual que un intervalo (D-12). O la
//	    define la norma, y entonces hay cita verificada; o la propone plazum
//	    porque la norma no nombra a nadie, y entonces hay justificacion y el
//	    cliente la cambia sin discutir con nadie. Medido: de las figuras del
//	    corpus, unas cuantas son de la primera clase (ENS art. 13, RGPD art.
//	    37, DORA arts. 5 y 6, NIS2 art. 20, RD 43/2021 art. 7, MDR art. 15) y
//	    el resto de la segunda, porque sus normas NO nombran a nadie: el CRA,
//	    eIDAS, el AI Act y el RDL 19/2018 no traen figura interna ninguna.
//
//	(2) NO SE UNIFICAN FIGURAS ENTRE NORMAS SIN CITA. El responsable de la
//	    seguridad del ENS y el organo de direccion de NIS2 no son la misma
//	    figura por parecerse el nombre; la equivalencia es una AFIRMACION y
//	    necesita fuente, igual que una cadencia gemela. Por eso los ids van
//	    prefijados por paquete, como los de obligacion: `ens.responsable_de_la
//	    _seguridad` y `nis2.responsable_de_seguridad` no se pueden confundir ni
//	    leyendo por encima.
//
//	    Que una PERSONA tenga varias figuras a la vez es otra cosa, es normal,
//	    y ese mapeo es del cliente (SCIM o a mano). El art. 7.4 del RD 43/2021
//	    lo dice con todas las letras: las funciones del responsable de la
//	    seguridad de la informacion "podran compatibilizarse" con las del
//	    Responsable de Seguridad del ENS. Compatibilizar dos figuras en una
//	    persona no es fundirlas en una figura.
//
// Y LOS TRES NOMBRES DEL MISMO SENOR SE MUEREN SOLOS. No hace falta un
// detector de alias: si el escalon tiene que nombrar una figura DECLARADA, un
// nombre que nadie declaro se cae al cargar.
type Rol struct {
	// ID es la identidad, prefijada por paquete como la de una obligacion.
	ID string `json:"id"`

	// Titulo es como se llama la figura para quien la lee. En una figura
	// definida por la norma se deriva de las palabras del propio articulo: es
	// la misma regla que la de los titulos de obligacion en un paquete
	// transcrito, y apartarse de ella exige decirlo.
	Titulo string `json:"titulo"`

	// Origen dice DE QUIEN es la figura. Vocabulario cerrado de dos.
	Origen string `json:"origen"`

	// Cita es obligatoria cuando la norma define la figura: el articulo, con
	// su verificacion. Opcional en una propuesta, donde puede apuntar al
	// gancho (el articulo que obliga a que exista alguien, sin nombrarlo).
	Cita string `json:"cita,omitempty"`

	// Justificacion es obligatoria en una propuesta: por que plazum sugiere
	// esta figura y no otra. Es lo que el cliente lee antes de cambiarla.
	Justificacion string `json:"justificacion,omitempty"`

	// Descripcion es que hace la figura, en una frase.
	Descripcion string `json:"descripcion,omitempty"`
}

const (
	// FiguraDeLaNorma: la norma la nombra. Exige cita.
	FiguraDeLaNorma = "definido_por_la_norma"
	// FiguraPropuesto: la norma no nombra a nadie y plazum sugiere quien suele
	// tenerla. Exige justificacion, y el cliente la cambia.
	FiguraPropuesto = "propuesto"
)

var origenesDeFigura = map[string]bool{FiguraDeLaNorma: true, FiguraPropuesto: true}

var (
	ErrFiguraSinOrigen     = errors.New("figura sin origen declarado")
	ErrFiguraSinCita       = errors.New("figura definida por la norma sin cita")
	ErrEscalonHaciaNadie   = errors.New("escalon hacia una figura que el paquete no declara")
	ErrFiguraQueNadieUsa   = errors.New("figura declarada a la que nadie escala")
	ErrFiguraDuplicada     = errors.New("figura declarada dos veces")
	ErrFiguraSinJustificar = errors.New("figura propuesta sin justificacion")
	ErrFiguraSinIdOTitulo  = errors.New("figura sin id o sin titulo")
)

// validarRoles comprueba el vocabulario de figuras y, sobre todo, LAS DOS
// DIRECCIONES del emparejamiento entre escalones y figuras (invariante 7).
//
// La direccion que se olvida es la segunda: una figura declarada a la que nadie
// escala no rompe nada hoy, y manana es una pregunta mas que el cliente tiene
// que contestar ("quien es el responsable de X") para que el dato no lo lea
// nadie. Es el campo huerfano en su version de personas.
func (p *Paquete) validarRoles(e func(string, ...any)) {
	declaradas := map[string]bool{}
	for _, r := range p.Roles {
		if r.ID == "" || r.Titulo == "" {
			e("%w: %q / %q. Una figura sin id no se puede escalar y una sin titulo no se le "+
				"puede ensenar a nadie", ErrFiguraSinIdOTitulo, r.ID, r.Titulo)
			continue
		}
		if declaradas[r.ID] {
			e("%w: %s", ErrFiguraDuplicada, r.ID)
		}
		declaradas[r.ID] = true

		if !origenesDeFigura[r.Origen] {
			e("%w: la figura %s declara origen %q, y los que hay son %q y %q. Sin origen no se "+
				"sabe si el nombre lo pone la norma o lo pone plazum, que es justo lo que el "+
				"cliente necesita saber para cambiarlo", ErrFiguraSinOrigen, r.ID, r.Origen,
				FiguraDeLaNorma, FiguraPropuesto)
			continue
		}
		switch r.Origen {
		case FiguraDeLaNorma:
			if motivo := citaVagaPorque(r.Cita); motivo != "" {
				e("%w: la figura %s dice que la define la norma y su cita %s. Una figura "+
					"definida por la norma sin articulo que lo diga es una afirmacion sin "+
					"fuente, y de esas viven las equivalencias inventadas entre marcos",
					ErrFiguraSinCita, r.ID, motivo)
			}
		case FiguraPropuesto:
			if len(strings.TrimSpace(r.Justificacion)) < 40 {
				e("%w: la figura %s la propone plazum y no dice por que. El cliente la va a "+
					"cambiar, y para cambiarla con criterio necesita leer de donde salio",
					ErrFiguraSinJustificar, r.ID)
			}
		}
	}

	// DIRECCION 1: todo escalon nombra una figura declarada.
	usadas := map[string]bool{}
	for _, o := range p.Obligaciones {
		for _, esc := range o.Escalado {
			usadas[esc.A] = true
			if !declaradas[esc.A] {
				e("%w: la obligacion %s escala a %q tras %s, y el paquete no declara esa "+
					"figura. Un aviso a un nombre que la organizacion no ha asignado no llega "+
					"a nadie y no da error: es un escalon que no escala. Arreglo: declararla "+
					"en `roles` con su origen, o corregir el nombre",
					ErrEscalonHaciaNadie, o.ID, esc.A, esc.Tras)
			}
		}
	}
	// DIRECCION 2, la que se olvida: toda figura declarada la usa alguien.
	for _, r := range p.Roles {
		if r.ID != "" && !usadas[r.ID] {
			e("%w: %s. Nadie escala a ella, asi que preguntarle al cliente quien la ocupa es "+
				"pedirle un dato que no se va a leer. Arreglo: usarla en un escalon o quitarla",
				ErrFiguraQueNadieUsa, r.ID)
		}
	}
}

// citaVagaPorque devuelve por que una cita no identifica un documento, o "" si
// lo identifica. Misma regla que `fuentes_del_intervalo`: minimo de longitud y
// al menos un numero, porque "el ENS" es una norma y "RD 311/2022, art. 13.2.c"
// es un sitio.
func citaVagaPorque(c string) string {
	c = strings.TrimSpace(c)
	switch {
	case c == "":
		return "esta vacia"
	case len(c) < 12:
		return fmt.Sprintf("es demasiado corta (%q): no identifica un documento", c)
	case !strings.ContainsAny(c, "0123456789"):
		return fmt.Sprintf("no lleva ningun numero (%q): sin numero de norma ni de articulo "+
			"no se puede ir a mirarlo", c)
	}
	return ""
}
