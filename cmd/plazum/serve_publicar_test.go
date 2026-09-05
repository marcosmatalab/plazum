package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/adaptadores/instalacion"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// LAS DOS FORMAS DE LA NADA EN LA RUTA DEL ALCANCE, Y LA TERCERA QUE NO ES NADA.
//
// # De donde sale este test, y no fue de leer el codigo
//
// De una MUTACION QUE SOBREVIVIO (M7, 05-09-2026). Cambiado
// `todaviaNoPublicado` para que devolviera true ante CUALQUIER error, la suite
// entera de cmd/plazum siguio en verde:
//
//	.github/mutar.sh comprobar "go test ./cmd/plazum/ -count=1"
//	ok  github.com/marcosmatalab/plazum/cmd/plazum  11.754s
//	LA MUTACION HA SOBREVIVIDO
//
// O sea que un alcance publicado ROTO -- un JSON a medias, una escritura que se
// corto, un fichero de otra version -- se habria leido como «todavia nadie ha
// contestado la entrevista», y el calendario habria salido vacio, tranquilo y
// con su siguiente paso puesto. Es la peor forma de fallar de este producto: una
// pantalla plausible es la que nadie arregla, y quien la mire dara por hecho que
// no ha empezado cuando lo que tiene es un fichero corrupto.
//
// # Las tres respuestas, que son tres y no dos
//
//	fichero AUSENTE y ruta del producto    «todavia no»: estado vacio, sin error
//	fichero AUSENTE y ruta del operador    error: la tecleo una persona
//	fichero PRESENTE y no interpretable    error SIEMPRE, venga de donde venga
func TestUnAlcancePublicadoRotoNoSeLeeComoTodaviaNoPublicado(t *testing.T) {
	dir := t.TempDir()
	ruta := rutaDelAlcancePublicado(dir)

	// La ruta la pone el producto y el fichero no esta: es el estado vacio.
	falta := alcanceEnFichero{ruta: ruta, ahora: time.Now, publicado: true}
	_, _, err := falta.leer(nil)
	if err == nil {
		t.Fatal("leer un alcance que no existe tenia que dar error de lectura")
	}
	if !falta.todaviaNoPublicado(err) {
		t.Errorf("con la ruta del producto y sin fichero, esto tiene que ser «todavia no»: "+
			"si no, una instalacion recien hecha abre el calendario con un aviso rojo.\n  %v",
			err)
	}

	// La misma ausencia, pero con la ruta que tecleo una persona: eso es un
	// error suyo y callarlo la deja con el calendario vacio convencida de que lo
	// configuro.
	tecleada := alcanceEnFichero{ruta: ruta, ahora: time.Now, publicado: false}
	_, _, err = tecleada.leer(nil)
	if err == nil {
		t.Fatal("leer un alcance que no existe tenia que dar error de lectura")
	}
	if tecleada.todaviaNoPublicado(err) {
		t.Error("con la ruta que tecleo el operador, un fichero que no esta NO es «todavia no»")
	}

	// Y LO QUE MATA LA MUTACION: presente y no interpretable. Se prueban las
	// tres formas de romperlo, porque cada una falla en un sitio distinto del
	// lector y una sola podria estar cazandose por casualidad.
	for _, caso := range []struct{ nombre, dentro string }{
		{"json a medias", `{"sujeto": "acme",`},
		{"no es json", "esto no es un alcance"},
		{"json valido sin sujeto", `{"organizacion": "Acme"}`},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if err := os.WriteFile(ruta, []byte(caso.dentro), 0o600); err != nil {
				t.Fatal(err)
			}
			roto := alcanceEnFichero{ruta: ruta, ahora: time.Now, publicado: true}
			_, _, err := roto.leer(nil)
			if err == nil {
				t.Fatalf("un alcance %q se ha leido sin error", caso.nombre)
			}
			if roto.todaviaNoPublicado(err) {
				t.Errorf("un alcance %q se lee como «todavia no ha publicado nadie».\n"+
					"  El calendario saldria VACIO Y TRANQUILO sobre un fichero roto, y quien "+
					"lo mire dara por hecho que no ha empezado.\n  %v", caso.nombre, err)
			}
			// Y la ausencia SIGUE siendo «todavia no» despues de todo esto: sin
			// este control positivo, una guarda que dijera que no a todo pasaria
			// las tres filas de arriba.
			if err := os.Remove(ruta); err != nil {
				t.Fatal(err)
			}
			if _, _, err := roto.leer(nil); !roto.todaviaNoPublicado(err) {
				t.Error("la ausencia ha dejado de ser «todavia no»: la guarda dice que no a todo")
			}
		})
	}
}

// PUBLICAR SIN SABER DE QUIEN ES LA INSTALACION NO ESCRIBE NADA.
//
// El sujeto es el nombre con el que las reglas de aplicabilidad hablan de la
// organizacion. Sin el, el motor derivaria las obligaciones de nadie y el
// calendario saldria vacio SIN DECIR POR QUE, que es peor que no publicar: la
// pantalla diria que ya esta hecho.
func TestPublicarSinIdentidadDeLaInstalacionNoEscribeNada(t *testing.T) {
	dir := t.TempDir()
	ruta := rutaDelAlcancePublicado(dir)
	a := alcanceDeLaInstalacion{
		ruta:     ruta,
		paquetes: []*corpus.Paquete{},
		quienEs:  func() instalacion.Identidad { return instalacion.Identidad{} },
	}
	err := a.Publicar(t.Context(), nil)
	if err == nil {
		t.Fatal("se ha publicado un alcance sin sujeto")
	}
	if !strings.Contains(err.Error(), "/primer-admin") {
		t.Errorf("el error no dice donde se arregla:\n%v", err)
	}
	if _, err := os.Stat(ruta); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ha quedado un fichero en %s despues de una publicacion rechazada", ruta)
	}
	// CONTROL POSITIVO: con identidad SI publica. Sin esto, un Publicar que
	// fallara siempre pasaria la mitad de arriba.
	a.quienEs = func() instalacion.Identidad {
		return instalacion.Identidad{Organizacion: "Ejemplo SL", Sujeto: "ejemplo-sl"}
	}
	if err := a.Publicar(t.Context(), nil); err != nil {
		t.Fatalf("con identidad tenia que publicar: %v", err)
	}
	b, err := os.ReadFile(ruta) // #nosec G304 -- ruta de un directorio temporal del test
	if err != nil {
		t.Fatalf("no ha quedado el alcance publicado: %v", err)
	}
	if !strings.Contains(string(b), "ejemplo-sl") {
		t.Errorf("el alcance publicado no lleva el sujeto de la instalacion:\n%s", b)
	}
	if filepath.Base(ruta) != nombreDelAlcancePublicado {
		t.Errorf("el alcance se publica en %q y no en %q", filepath.Base(ruta),
			nombreDelAlcancePublicado)
	}
}
