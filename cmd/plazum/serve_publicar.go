package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/marcosmatalab/plazum/adaptadores/instalacion"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// EL ALCANCE DE LA INSTALACION, PUBLICADO DESDE EL NAVEGADOR.
//
// # Que apaga, con el numero delante
//
// Las dos ordenes de terminal que pedian el calendario y el plan de avisos en su
// estado vacio: `plazum alcance --cuenta ... --salida alcance.json` y
// `plazum serve --alcance alcance.json`. Medidas por el arnes del TTFV son 3m0s
// cobrados, la ultima partida de ordenes que quedaba del camino guiado.
//
// # HAY DOS ALCANCES Y ESTE ES EL DE LA INSTALACION (invariante 12)
//
// El calendario y el plan de avisos se sirven SIN SESION, asi que su alcance no
// puede salir de la cuenta que mira: tiene que ser uno publicado a proposito. Lo
// publica quien adopta la entrevista, con el rotulo de la pantalla diciendo que
// lo que adopta lo va a ver cualquiera que abra esta instalacion.
//
// # NO HAY UNA SEGUNDA CONVERSION, y eso es lo que importa aqui
//
// La traduccion de «respuestas de la entrevista» a «alcance con sus hechos» la
// hace `exportarAlcance`, que es exactamente la misma funcion que corre por
// `plazum alcance`. Escribir aqui una version propia habria dado dos motores
// para la misma cuenta -- que respuesta produce que hecho, cual no tiene puente,
// cual pide valor -- y el dia que discreparan, el calendario del navegador y el
// del terminal ensenarian obligaciones distintas sobre las mismas respuestas.
//
// # DE QUIEN ES: DE LA INSTALACION, NO DE QUIEN PUBLICA
//
// El sujeto y la organizacion salen de `adaptadores/instalacion`, o sea de lo
// que se contesto una vez en /primer-admin. Sacarlos de la cuenta que publica
// haria que el calendario cambiara de dueno segun quien pulsara el boton, y el
// sujeto es justamente lo que NO se puede mover: de el cuelgan las derivaciones,
// las citas y lo que ya este escrito en un expediente.
type alcanceDeLaInstalacion struct {
	ruta     string
	paquetes []*corpus.Paquete
	quienEs  func() instalacion.Identidad
}

// NombreDelAlcancePublicado es como se llama dentro del directorio de datos.
const nombreDelAlcancePublicado = "alcance.json"

func rutaDelAlcancePublicado(datos string) string {
	return filepath.Join(datos, nombreDelAlcancePublicado)
}

// Publicar compone el alcance y lo escribe.
func (a alcanceDeLaInstalacion) Publicar(_ context.Context, respuestas url.Values) error {
	id := a.quienEs()
	if !id.Hay() {
		// NO SE PUBLICA UN ALCANCE SIN SUJETO. El sujeto es el nombre con el que
		// las reglas de aplicabilidad hablan de la organizacion: sin el, el
		// motor derivaria las obligaciones de nadie, y el calendario saldria
		// vacio sin decir por que. Es la misma negativa que cargarAlcance.
		return errors.New("esta instalacion todavia no dice de quien es, asi que no se puede " +
			"publicar su calendario.\n" +
			"  El nombre se pregunta al crear el primer administrador, en /primer-admin.")
	}
	exp, _, err := exportarAlcance(a.paquetes, respuestas, id.Sujeto, id.Organizacion)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(exp, "", "  ")
	if err != nil {
		return err
	}
	// El mismo fsync que la orden de terminal, y por lo mismo: un fichero que se
	// pierde en el cache de pagina cuando se va la luz deja un calendario que
	// dice haberse publicado y no esta.
	if err := escribirConFsync(a.ruta, append(b, '\n')); err != nil {
		return fmt.Errorf("no se puede publicar el alcance en %s: %w", a.ruta, err)
	}
	return nil
}
