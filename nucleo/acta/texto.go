package acta

import (
	"fmt"
	"strings"
)

// El board pack impreso.
//
// POR QUE EL TEXTO SE ESCRIBE AQUI Y NO EN LA SUPERFICIE. Por lo mismo que el
// informe de la campana de accesos: la frase de lo no constatado tiene que
// viajar CON EL DATO. Si la pusiera cada pantalla, la primera que se olvide
// acusa en falso. Y ademas este documento tiene que existir sin navegador: el
// board pack se imprime, se manda por correo y se adjunta a un expediente.
//
// EN PAPEL NO SE PUEDE CLICAR, asi que la derivacion se resuelve con una
// REFERENCIA ESTABLE. Cada cifra lleva su ref (1.2.3 = seccion, reparto, cubo) y
// el apendice trae la lista entera de lo que compone cada numero, sin recortar.
// Recortar el apendice seria justo lo que esta pieza existe para no hacer: un
// numero con una lista a medias vuelve a ser un numero que hay que creerse.

// Ref es la referencia estable de una cifra dentro del acta: seccion.reparto.cubo.
//
// Sale de la POSICION, y esa es la unica cosa de este paquete que se identifica
// asi. Se puede porque las tres listas son de vocabulario cerrado y orden fijo
// (FuentesPosibles, los repartos que arma cada seccion, los cubos de cada
// reparto), asi que la posicion no depende del dato: dos actas de la misma
// version de plazum ponen el mismo numero en el mismo sitio. Lo que NO se hace
// nunca con posiciones es emparejar dos conjuntos (invariante 7), y aqui no se
// empareja nada: es una etiqueta de impresion.
func ref(seccion, reparto, cubo int) string {
	return fmt.Sprintf("%d.%d.%d", seccion+1, reparto+1, cubo+1)
}

// Derivar devuelve la cifra de una referencia. Es lo que contesta a un clic.
func (a Acta) Derivar(r string) (CifraSituada, bool) {
	for _, c := range a.Cifras() {
		if c.Ref == r {
			return c, true
		}
	}
	return CifraSituada{}, false
}

// Prosa reparte los parrafos del acta por su procedencia, CON LOS CUBOS VACIOS.
//
// Es la derivacion de la regla 2: la afirmacion "en este documento no hay prosa
// generada" no se hace en una nota, se hace ensenando los tres montones y
// dejando ver que no hay un cuarto.
func (a Acta) Prosa() map[Procedencia][]Parrafo {
	out := map[Procedencia][]Parrafo{}
	for _, p := range ProcedenciasPosibles() {
		out[p] = nil
	}
	for _, p := range a.Parrafos() {
		out[p.De] = append(out[p.De], p)
	}
	return out
}

// Texto es el board pack.
func (a Acta) Texto() string {
	var b strings.Builder
	a.escribirCabecera(&b)
	for i, s := range a.Secciones {
		a.escribirSeccion(&b, i, s)
	}
	a.escribirProcedencias(&b)
	a.escribirDerivaciones(&b)
	return b.String()
}

func (a Acta) escribirCabecera(b *strings.Builder) {
	// LOS ROTULOS DEL DOCUMENTO SALEN DE frases.go, no de literales aqui: el
	// papel y la pantalla titulan igual el mismo acta porque leen la misma
	// frase, y esa frase tiene su clave de catalogo.
	fmt.Fprintf(b, "%s\n%s\n", strings.ToUpper(tiDocumento.Texto), tiSubtitulo.Texto)
	fmt.Fprintf(b, "%s\n", a.Organizacion)
	fmt.Fprintf(b, "periodo: %s\n", a.Periodo)
	fmt.Fprintf(b, "acta:    %s\n\n", a.ID)
	for _, p := range a.Cabecera {
		escribirParrafo(b, p)
	}
	a.escribirQuePuedeDecir(b)
	if !a.Cuadra() {
		fmt.Fprintf(b, "\nAVISO: %s\n", avDescuadre.Texto)
		for _, d := range a.Descuadres() {
			fmt.Fprintf(b, "  %s\n", d)
		}
	}
}

// escribirQuePuedeDecir es lo primero que necesita quien no ha visto plazum
// nunca: para que sirve esto y donde no llega.
//
// Sale de la pasada contra el comprador. Un consejo que abre el documento tiene
// que poder calibrarlo antes de leer un solo numero, y lo que lo calibra no es
// una cifra, es de cuantas de sus cuatro fuentes hay registro. Un acta con la
// mitad de las fuentes sin conectar se lee exactamente igual de completa que una
// entera si no se dice aqui, porque las secciones salen las cuatro.
//
// No lleva ningun recuento a proposito: los numeros de este acta viven en los
// repartos, donde llevan su referencia y se pueden abrir.
func (a Acta) escribirQuePuedeDecir(b *strings.Builder) {
	fmt.Fprintf(b, "%s\n%s\n", strings.ToUpper(tiQuePuedeDecir.Texto),
		strings.Repeat("-", len(tiQuePuedeDecir.Texto)))
	for _, s := range a.Secciones {
		if s.Aportada {
			fmt.Fprintf(b, "  SI  %s\n", s.Fuente.Clave().Texto)
			continue
		}
		fmt.Fprintf(b, "  NO  %s\n", s.Fuente.Clave().Texto)
		for _, l := range envolver(s.PorQueFalta, 68) {
			fmt.Fprintf(b, "      %s\n", l)
		}
	}
	fmt.Fprintln(b)
}

func (a Acta) escribirSeccion(b *strings.Builder, i int, s Seccion) {
	rotulo := s.Fuente.Clave().Texto
	fmt.Fprintf(b, "\n%s\n%s\n", strings.ToUpper(rotulo), strings.Repeat("-", len(rotulo)))
	for _, p := range s.Parrafos {
		escribirParrafo(b, p)
	}
	for j, r := range s.Repartos {
		fmt.Fprintf(b, "\n  %s: %d\n", r.Rotulo.Texto, r.Universo)
		for k, c := range r.Cifras {
			fmt.Fprintf(b, "    %-52s %5d   [%s]\n", c.Cubo.Texto, c.Valor(), ref(i, j, k))
			// LA FRASE VA AQUI, pegada al numero que la necesita, y solo cuando
			// el cubo tiene algo dentro: sin datos no hay a quien descargar de
			// nada, y una frase que sale siempre deja de leerse.
			if c.Valor() > 0 && !c.Descargo.Vacia() {
				for _, l := range envolver(c.Descargo.Texto, 68) {
					fmt.Fprintf(b, "      %s\n", l)
				}
			}
		}
		if !r.Cuadra() {
			fmt.Fprintf(b, "    AVISO: los cubos suman %d y el universo es %d. Este reparto NO "+
				"vale.\n", r.Suma(), r.Universo)
		}
	}
}

func (a Acta) escribirProcedencias(b *strings.Builder) {
	prosa := a.Prosa()
	fmt.Fprintf(b, "\n%s\n%s\n", strings.ToUpper(tiProcedencias.Texto),
		strings.Repeat("-", len(tiProcedencias.Texto)))
	total := 0
	for _, p := range ProcedenciasPosibles() {
		fmt.Fprintf(b, "  %-30s %d\n", p.Clave().Texto, len(prosa[p]))
		total += len(prosa[p])
	}
	fmt.Fprintf(b, "  %-30s %d\n", "total", total)
	fmt.Fprintln(b)
	for _, l := range envolver(pfNoHayCuarta.Texto, 72) {
		fmt.Fprintf(b, "  %s\n", l)
	}
}

func (a Acta) escribirDerivaciones(b *strings.Builder) {
	fmt.Fprintf(b, "\n%s\n%s\n", strings.ToUpper(tiDerivaciones.Texto),
		strings.Repeat("-", len(tiDerivaciones.Texto)))
	for _, l := range envolver(pfDerivacionesEnteras.Texto, 76) {
		fmt.Fprintf(b, "%s\n", l)
	}
	for _, c := range a.Cifras() {
		// En tres lineas y no en una: la de una sola se sale del ancho de un
		// folio, y este documento se imprime.
		fmt.Fprintf(b, "\n[%s] %s\n      %s\n      %s: %d\n", c.Ref, c.Fuente.Clave().Texto,
			c.Reparto.Texto, c.Cifra.Cubo.Texto, c.Cifra.Valor())
		if c.Cifra.Valor() == 0 {
			fmt.Fprintf(b, "  (vacio)\n")
			continue
		}
		for _, el := range c.Cifra.Elementos {
			fmt.Fprintf(b, "  %s", el.Clave)
			if el.Que != "" {
				fmt.Fprintf(b, "  %s", el.Que)
			}
			if el.Nota != "" {
				fmt.Fprintf(b, "  -- %s", el.Nota)
			}
			fmt.Fprintln(b)
		}
	}
}

// escribirParrafo imprime la prosa CON SU ATRIBUCION al lado. La de plazum no
// lleva marca porque es la voz del documento; las otras dos si, porque son de
// alguien y quien las lee tiene derecho a saber de quien.
func escribirParrafo(b *strings.Builder, p Parrafo) {
	for _, l := range envolver(p.Texto, 76) {
		fmt.Fprintf(b, "%s\n", l)
	}
	switch p.De {
	case DeUnaPersona:
		fmt.Fprintf(b, "    -- %s\n", p.Quien)
	case DeLaNorma:
		fmt.Fprintf(b, "    fuente: %s\n", p.Cita)
	}
	fmt.Fprintln(b)
}

// envolver parte un parrafo en lineas de ancho maximo. Sin cortar palabras: un
// documento que un consejo va a leer impreso no puede tener guiones inventados
// en mitad de una palabra.
func envolver(s string, ancho int) []string {
	campos := strings.Fields(s)
	if len(campos) == 0 {
		return nil
	}
	var out []string
	linea := campos[0]
	for _, w := range campos[1:] {
		if len(linea)+1+len(w) > ancho {
			out = append(out, linea)
			linea = w
			continue
		}
		linea += " " + w
	}
	return append(out, linea)
}
