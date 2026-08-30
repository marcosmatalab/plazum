package diagnostico

// La comprobacion del DIA UNO del escalado.
//
// El linter del corpus ya garantiza que todo escalon nombra una figura que su
// paquete declara. Lo que no puede saber es si ESTA organizacion tiene a
// alguien detras de esa figura, porque eso es dato del cliente. Sin esta
// comprobacion, el aviso se descubre roto EL DIA DEL INCIDENTE, que es el unico
// dia en que ya no se puede arreglar.
//
// Va en doctor porque doctor es lo que se ejecuta al instalar y lo que se pone
// en un systemd de arranque: el sitio donde una cosa que falta se entera antes
// de hacer falta.

import (
	"context"
	"fmt"
	"strings"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/escalado"
	"github.com/marcosmatalab/plazum/puertos"
)

// cuantasFaltasSeEnumeran es el tope de figuras que salen con nombre. Las demas
// se cuentan. Enumerar veintisiete lineas en un doctor es no decir nada.
const cuantasFaltasSeEnumeran = 5

func (d *Doctor) figuras(context.Context) puertos.Comprobacion {
	c := puertos.Comprobacion{Nombre: "figuras"}
	ps, err := corpus.Cargar(d.o.Corpus)
	if err != nil || len(ps) == 0 {
		// El corpus tiene su propia comprobacion y ya lo ha dicho con detalle.
		// Repetirlo aqui seria dar dos veces el mismo problema con dos nombres.
		c.Estado = puertos.Aviso
		c.Detalle = "no se puede comprobar quien ocupa cada figura sin un corpus que las declare"
		c.Arreglo = "arregla antes la comprobacion `corpus`, que dice que falla exactamente"
		return c
	}

	total := 0
	for _, p := range ps {
		total += len(p.Roles)
	}
	if total == 0 {
		c.Estado = puertos.Correcto
		c.Detalle = "ningun paquete instalado declara figuras de escalado"
		return c
	}

	faltan := escalado.FigurasSinPersona(ps, d.o.Figuras)
	avisosPerdidos := 0
	deLaNorma := 0
	for _, f := range faltan {
		avisosPerdidos += f.Escalones
		if f.Origen == corpus.FiguraDeLaNorma {
			deLaNorma++
		}
	}
	if len(faltan) == 0 {
		c.Estado = puertos.Correcto
		c.Detalle = fmt.Sprintf("las %d figuras que declara el corpus tienen a alguien detras", total)
		return c
	}

	donde := "no has dado ningun alcance"
	if d.o.Alcance != "" {
		donde = "el alcance de " + d.o.Alcance
	}
	c.Estado = puertos.Aviso
	c.Detalle = fmt.Sprintf("%d de %d figuras no las ocupa nadie (%s), y eso deja %d aviso(s) "+
		"de escalado sin destinatario. %d de las que faltan las nombra la norma. %s",
		len(faltan), total, donde, avisosPerdidos, deLaNorma, listaDeFaltas(faltan))
	c.Arreglo = "abre el alcance y anade el mapa `figuras` con el id de cada una y la persona " +
		"que la ocupa. Las que propone plazum se pueden cambiar por otra figura tuya; las que " +
		"nombra la norma hay que ocuparlas. Esto NO dice que hayas incumplido nada: dice que " +
		"si hoy venciera uno de esos plazos, el aviso no llegaria a ningun sitio"
	return c
}

// listaDeFaltas enumera las mas caras y CUENTA el resto. Un tope silencioso
// diria "estas son las que faltan" enseñando cinco de veintisiete.
func listaDeFaltas(faltan []escalado.Falta) string {
	var b strings.Builder
	b.WriteString("Las que mas avisos se llevan: ")
	n := len(faltan)
	if n > cuantasFaltasSeEnumeran {
		n = cuantasFaltasSeEnumeran
	}
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(faltan[i].Frase())
	}
	if resto := len(faltan) - n; resto > 0 {
		fmt.Fprintf(&b, ". Y %d mas, que no se enumeran aqui para que esta linea se pueda leer",
			resto)
	}
	return b.String()
}
