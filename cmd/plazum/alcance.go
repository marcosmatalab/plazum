package main

// El ALCANCE: las respuestas de una organizacion sobre si misma.
//
// Se extrajo de demo.go el 27-08-2026 porque `plazum calendario` necesita
// exactamente lo mismo que el demo, y tener dos lectores del mismo fichero es
// tener dos derivas. Es la misma leccion que costo la unificacion de los dos
// traductores del reloj declarado (nucleo/corpus.VencimientosDe): el dia que las
// dos se separan, alguien ensena una fecha que la otra no da.
//
// Va como datos y FUERA de paquete.json porque un paquete de corpus declara
// REGLAS, nunca hechos sobre un sujeto: un paquete que afirmara hechos estaria
// afirmando algo que no puede saber.

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/estricto"
	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// alcance son las respuestas de la empresa de ejemplo: lo que un operador
// habria tecleado en la pantalla de Alcance.
//
// Va como datos y no en el codigo porque es contenido del demo, no logica; y
// va FUERA de paquete.json porque un paquete de corpus declara REGLAS, nunca
// hechos sobre un sujeto: un paquete que afirmara hechos estaria afirmando algo
// que no puede saber.
type alcance struct {
	Organizacion string `json:"organizacion"`
	Sujeto       string `json:"sujeto"`
	Descripcion  string `json:"descripcion"`
	Respuestas   []struct {
		Campo    string `json:"campo"`
		Valor    string `json:"valor"`
		Pregunta string `json:"pregunta"`
	} `json:"respuestas"`
	Hechos []struct {
		Pred string   `json:"pred"`
		Args []string `json:"args"`
	} `json:"hechos"`
	Fechas map[string]string `json:"fechas"`
	// NotasDeLasFechas explica, dentro del propio fichero, como se leen las
	// fechas de arriba. JSON no tiene comentarios y este es su sustituto.
	//
	// TIENE CAMPO Y SE IMPRIME, que son las dos mitades. Sin campo, la
	// decodificacion estricta rechazaba el alcance del demo entero — y ese
	// fichero lo escribe `plazum demo` y lo nombra la ayuda de este comando, o
	// sea que el producto producia un fichero que despues no cargaba. Con campo
	// y sin imprimirlo seria un huerfano, que es la otra mitad de la misma
	// familia.
	NotasDeLasFechas string `json:"notas_de_las_fechas,omitempty"`
	// Figuras dice QUIEN ocupa cada figura declarada por un paquete. La
	// clave es el id de figura con su prefijo (ens.responsable_de_la_seguridad)
	// y el valor, la persona. Es dato del cliente: sale de SCIM o se pone a
	// mano. Sin el, los escalones existen y no llegan a nadie.
	Figuras map[string]string `json:"figuras,omitempty"`
}

// fechasDelAlcance resuelve las fechas del demo contra el instante de calculo.
//
// Admite dos formas y las dos existen por una razon: una fecha absoluta
// (2026-01-15) para congelar un escenario, y un desplazamiento (-45d, -30h)
// para que el demo ensene siempre plazos vivos. Un demo con fechas fijas
// envejece, y a los seis meses ensena tres relojes vencidos, que es justo lo
// contrario de lo que tiene que ensenar.
func fechasDelAlcance(al alcance, ahora time.Time) (ventana.Hechos, error) {
	h := ventana.Hechos{}
	for clave, valor := range al.Fechas {
		t, err := resolverFecha(valor, ahora)
		if err != nil {
			return nil, fmt.Errorf("la fecha %q del alcance del demo (%q) no se entiende: %w",
				clave, valor, err)
		}
		h[clave] = t
	}
	return h, nil
}

func resolverFecha(v string, ahora time.Time) (time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, errors.New("esta vacia")
	}
	if v[0] == '-' || v[0] == '+' {
		signo := 1
		if v[0] == '-' {
			signo = -1
		}
		cuerpo := v[1:]
		if len(cuerpo) < 2 {
			return time.Time{}, errors.New("un desplazamiento se escribe como -45d o -30h")
		}
		unidad := cuerpo[len(cuerpo)-1]
		n, err := strconv.Atoi(cuerpo[:len(cuerpo)-1])
		if err != nil {
			return time.Time{}, fmt.Errorf("%q no es un numero de %s", cuerpo[:len(cuerpo)-1],
				string(unidad))
		}
		switch unidad {
		case 'd':
			return ahora.AddDate(0, 0, signo*n), nil
		case 'h':
			return ahora.Add(time.Duration(signo*n) * time.Hour), nil
		default:
			return time.Time{}, fmt.Errorf("unidad %q desconocida; se admiten d (dias) y h (horas)",
				string(unidad))
		}
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t.UTC(), nil
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return time.Time{}, errors.New("ni es AAAA-MM-DD, ni RFC3339, ni un desplazamiento como -45d")
	}
	return t.UTC(), nil
}

// cargarAlcance lee un alcance del DISCO. Es lo que separa al calendario del
// demo: el demo lo trae empotrado y aqui lo pone quien lo usa.
//
// El error dice como se arregla, con el fichero delante, porque quien teclea
// esto por primera vez no ha visto nunca este formato.
func cargarAlcance(ruta string) (alcance, error) {
	var al alcance
	b, err := os.ReadFile(ruta) // #nosec G304 -- CLI: la ruta la teclea el operador en su propia maquina
	if err != nil {
		return al, fmt.Errorf("no puedo leer el alcance %s: %w.\n"+
			"  El alcance son las respuestas de tu organizacion sobre si misma: que eres, que \n"+
			"  haces y cuando ocurrio cada cosa. Saca uno de ejemplo con `plazum demo` y mira \n"+
			"  plazum-demo/paquetes/demo-empresa/alcance.json", ruta, err)
	}
	// Estricto: el alcance son las respuestas del operador, y una respuesta con
	// el identificador mal escrito descartada en silencio da un expediente al
	// que le faltan obligaciones sin que nadie lo diga. Ver nucleo/estricto.
	if err := estricto.Decodificar(b, &al, "el alcance "+ruta); err != nil {
		return al, err
	}
	if al.Sujeto == "" {
		return al, fmt.Errorf("el alcance %s no declara `sujeto`.\n"+
			"  El sujeto es el nombre con el que las reglas de aplicabilidad hablan de tu\n"+
			"  organizacion: sin el, el motor deriva las obligaciones de nadie", ruta)
	}
	return al, nil
}
