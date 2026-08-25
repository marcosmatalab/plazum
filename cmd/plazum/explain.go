package main

import (
	"fmt"
	"io"

	"plazum/adaptadores/catalogo"
	"plazum/nucleo/aplicabilidad"
	"plazum/nucleo/expediente"
)

// plazum explain: por que cada conclusion del expediente es la que es.
//
// Es la orden que un auditor abre para discutir una linea. Ensena los paquetes
// instalados con su digest, la aplicabilidad rederivada regla a regla, y los
// relojes con su vencimiento.
//
// Y termina con el descargo, que no es adorno legal ni relleno: esta pantalla
// dice "esta obligacion te aplica y vence el dia tal" en un tono que se parece
// mucho al de un dictamen, y no lo es. Lo que hay debajo es el texto de los
// paquetes que el operador tenga instalados, con su cita, derivado por un motor
// determinista. Si el paquete esta desactualizado o la interpretacion del
// articulo es discutible, la salida sale igual de segura de si misma. Decirlo
// aqui, donde se lee el resultado, y no en un fichero de licencia que nadie
// abre, es la unica forma de que llegue.

// claveDescargo es la MISMA clave que pinta el pie de las pantallas servidas
// (superficies/pantallas/plantillas/base.html). Reusarla en vez de escribir
// aqui otro texto es lo que hace que la consola y la web digan exactamente lo
// mismo: dos descargos distintos para el mismo producto es un descargo que
// alguien tendra que interpretar.
const claveDescargo = "ui.pie.no_asesoramiento"

// descargoDeReserva es el texto que se imprime si el catalogo de interfaz no
// carga.
//
// Por que existe una copia. El catalogo va embebido y su cargador es estricto
// (rechaza cadenas que parezcan citas normativas, ver adaptadores/catalogo/
// frontera.go), asi que puede negarse a cargar. Que `plazum explain` deje de
// explicar un expediente porque falla la traduccion de un rotulo seria un
// producto roto por el rotulo. Pero el descargo NO es opcional: si se cae el
// catalogo, se imprime esto.
//
// Que las dos versiones no se separen lo vigila
// TestElDescargoDeReservaDiceLoMismoQueElCatalogo, que compara esta constante
// con el valor de la clave: una copia sin puerta se desvia el primer dia que
// alguien mejora la redaccion de una sola de las dos.
const descargoDeReserva = "plazum no presta asesoramiento jurídico. Lo que ves aquí es lo que " +
	"dicen los paquetes normativos que tienes instalados, con su cita, para que puedas " +
	"comprobarlo tú."

// descargo devuelve el texto del descargo en el idioma por defecto del
// catalogo, o el de reserva si el catalogo no carga.
func descargo() string {
	cat, err := catalogo.Nuevo()
	if err != nil {
		return descargoDeReserva
	}
	idiomas := cat.Idiomas()
	if len(idiomas) == 0 {
		return descargoDeReserva
	}
	return cat.Traducir(idiomas[0], claveDescargo)
}

// cmdExplain imprime la derivacion de cada conclusion del expediente y devuelve
// el codigo de salida.
func cmdExplain(e *expediente.Expediente, salida, errores io.Writer) int {
	fmt.Fprintf(salida, "paquetes instalados (%d)\n", len(e.Paquetes))
	for _, p := range e.Paquetes {
		fmt.Fprintf(salida, "  %-20s %-12s vigente desde %s  %s\n",
			p.URN, p.Clase, p.Vigencia.Desde.Format("2006-01-02"), p.Digest)
	}
	m := aplicabilidad.NuevoMotor()
	for _, pr := range e.Programas {
		if err := m.Cargar(pr); err != nil {
			fmt.Fprintf(errores, "programa invalido en el paquete %s: %v\n", pr.Paquete, err)
			return 1
		}
	}
	for _, h := range e.Hechos {
		m.Afirmar(h)
	}
	if _, err := m.Evaluar(); err != nil {
		fmt.Fprintln(errores, "aplicabilidad:", err)
		return 1
	}
	fmt.Fprintf(salida, "\naplicabilidad derivada\n")
	for _, h := range m.Consultar(aplicabilidad.A("aplica", aplicabilidad.V("O"), aplicabilidad.V("S"))) {
		fmt.Fprintf(salida, "  %-28s sobre %-20s <- %s\n", h.Args[0], h.Args[1], m.Explicar(h))
	}
	fmt.Fprintf(salida, "\nrelojes\n")
	for _, r := range e.Relojes {
		for _, rec := range e.Reclamaciones {
			if rec.Obligacion != r.Obligacion {
				continue
			}
			if rec.Estado == "determinado" {
				fmt.Fprintf(salida, "  %-28s %-18s vence %s\n",
					r.Obligacion, rec.Hito, rec.Vence.Format("2006-01-02 15:04 -07:00"))
			} else {
				fmt.Fprintf(salida, "  %-28s %-18s %s\n", r.Obligacion, rec.Hito, rec.Estado)
			}
		}
	}
	// El descargo va al final y por SALIDA, no por la de errores: quien
	// redirige `plazum explain > informe.txt` para llevarlo a una reunion tiene
	// que llevarselo dentro.
	fmt.Fprintf(salida, "\n%s\n", ajustar(descargo(), 76, ""))
	return 0
}
