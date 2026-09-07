package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/marcosmatalab/plazum/adaptadores/recoleccion/manual"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/estado"
	"github.com/marcosmatalab/plazum/superficies/pantallas"
)

// EL CABLE QUE UNE LOS TRES PRIMEROS ESLABONES (A2 y A3 de D-22).
//
// Aqui se juntan las tres piezas que hasta hoy no se habian visto nunca en la
// misma linea:
//
//	el recolector   adaptadores/recoleccion/manual, que lee observaciones de un
//	                fichero y no necesita ni una credencial
//	el puente       corpus.Prueba.AlExpediente(), que convierte la prueba que
//	                declara un paquete en la que entiende el motor
//	el motor        estado.Calcular, que hasta hoy solo llamaba el verificador
//	                del expediente
//
// # POR QUE EL CABLE VIVE EN cmd/plazum Y NO EN LA SUPERFICIE
//
// Porque es el unico sitio que conoce a los tres. `superficies/pantallas` no
// puede importar el corpus para convertir pruebas ni el adaptador para leer
// ficheros: su contrato es un interfaz de lectura y nada mas. Es la misma forma
// que `consecuenciasDeLaEntrevista` y que `alcanceDeLaInstalacion`.
//
// # SE RECALCULA EN CADA PETICION, Y ESO ES LA DECISION
//
// No se cachea el resultado del motor. Un estado de evidencia calculado al
// arrancar diria «consta» sobre un dato que caduco anoche, y llevaria
// diciendolo hasta el siguiente reinicio. Es exactamente la mentira que
// `verHoy` documenta y evita con el veredicto del planificador.
//
// Lo que si se hace una vez es INDEXAR las pruebas del corpus, que no cambian
// mientras el proceso vive porque los paquetes se cargan al arrancar.
type evidenciaDeLaInstalacion struct {
	// pruebas son las del corpus, ya cruzadas al vocabulario del motor, con lo
	// que cada una necesita para pintarse al lado de su obligacion.
	pruebas []pruebaCableada
	// recolector es de donde salen las observaciones.
	recolector *manual.Recolector
	// ahora es el reloj, inyectado: `nucleo/` no llama a time.Now() y este
	// cable tampoco deberia, para que su test pueda fijar el instante.
	ahora func() time.Time

	mu sync.Mutex
}

// pruebaCableada es una prueba del corpus con lo que hace falta para pintarla.
type pruebaCableada struct {
	// obligacion es el id contra el que se empareja en la pantalla. Es el mismo
	// campo que `estado.Prueba.Control` (invariante 7: identidad, no posicion).
	obligacion string
	// motor es la prueba en el vocabulario de estado.Calcular.
	motor estado.Prueba
	// cierra y predicado NO cruzan el puente y hacen falta en la pantalla, asi
	// que se llevan al lado. Ver el godoc de AlExpediente.
	cierra    bool
	predicado string
}

// nuevaEvidencia indexa las pruebas del corpus y prepara el cable.
//
// UNA PRUEBA QUE NO CRUZA EL PUENTE NO SE SALTA EN SILENCIO: se devuelve el
// error. El linter del corpus ya exige lo que el puente exige, asi que si esto
// falla es que hay una incoherencia entre los dos, y tragarsela dejaria una
// obligacion sin evidencia sin que nadie supiera por que.
func nuevaEvidencia(ps []*corpus.Paquete, r *manual.Recolector,
	ahora func() time.Time) (*evidenciaDeLaInstalacion, error) {

	if ahora == nil {
		return nil, errors.New("evidencia: sin reloj. Arreglo: pasa time.Now o el " +
			"reloj de la peticion. Un reloj nil no es «el de por defecto»")
	}
	var out []pruebaCableada
	for _, p := range ps {
		for _, pr := range p.Pruebas {
			m, err := pr.AlExpediente()
			if err != nil {
				return nil, fmt.Errorf("la prueba %s del paquete %s no cruza al "+
					"expediente: %w", pr.ID, p.URN, err)
			}
			out = append(out, pruebaCableada{
				obligacion: pr.Obligacion,
				motor:      m,
				cierra:     pr.Cierra != nil && *pr.Cierra,
				predicado:  pr.Predicado,
			})
		}
	}
	return &evidenciaDeLaInstalacion{pruebas: out, recolector: r, ahora: ahora}, nil
}

// De lee las observaciones y calcula el estado de cada prueba.
//
// EL ERROR NO SE DEGRADA. Si el fichero de observaciones no se puede leer, sale
// el error y la pantalla lo dice: leerlo como «todavia nadie ha recolectado» le
// diria a alguien que su instalacion esta al dia cuando lo que pasa es que no
// hemos podido leer su fichero.
func (e *evidenciaDeLaInstalacion) De(context.Context) (map[string]pantallas.Evidencia, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	obs, _, err := e.recolector.Recolectar(manual.Fuente, "")
	if err != nil {
		return nil, err
	}
	// LAS OBSERVACIONES SE AGRUPAN POR EL ID DE LA PRUEBA, que es el campo por
	// el que el recolector y el paquete se emparejan (invariante 7). Nunca por
	// orden: nadie firma el orden de un fichero de observaciones.
	porPrueba := map[string][]estado.Observacion{}
	for _, o := range obs {
		porPrueba[o.Prueba] = append(porPrueba[o.Prueba], o)
	}

	ahora := e.ahora()
	out := make(map[string]pantallas.Evidencia, len(e.pruebas))
	for _, pc := range e.pruebas {
		suyas := porPrueba[pc.motor.ID]
		ent := estado.Calcular(pc.motor, suyas, estado.Contexto{
			Ahora: ahora,
			// APLICABLE A TRUE, Y HAY QUE DECIR POR QUE. La aplicabilidad la
			// decide el motor de reglas del corpus contra el alcance publicado,
			// y esta pantalla YA la pinta en su propia columna: la de la
			// izquierda dice si te aplica. Pasar aqui un false haria que la
			// columna de evidencia dijera «fuera de tu alcance» por su cuenta,
			// o sea la misma pregunta contestada dos veces y por dos caminos
			// distintos, que es como acaban discrepando.
			//
			// LO QUE ESTO NO CUBRE, dicho: una obligacion que NO te aplica sale
			// aqui con su estado de evidencia igual. La pantalla las pone una
			// al lado de la otra y quien mire ve las dos, que es mas honesto
			// que esconder una detras de la otra.
			Aplicable: true,
		})
		// EL MOTIVO CRUZA EN CLAVE Y ARGUMENTOS, no en frase. El motor escribe
		// en castellano y esta superficie tiene una pagina inglesa: pasar
		// `ent.Motivo.Texto` era lo que hacia que la pagina en ingles imprimiera
		// espanol sin que ninguna puerta de i18n pudiera verlo.
		out[pc.obligacion] = pantallas.Evidencia{
			Estado:           ent.Estado.String(),
			MotivoClave:      ent.Motivo.Clave,
			MotivoArgs:       ent.Motivo.Args,
			Recolectada:      ent.Recolectada,
			SinObservaciones: len(suyas) == 0,
			Recolector:       recolectorDe(suyas),
			Cierra:           pc.cierra,
			Predicado:        pc.predicado,
		}
	}
	return out, nil
}

// recolectorDe dice QUIEN trajo el dato.
//
// Si hay varios recolectores para la misma prueba, se dice que son varios en
// vez de elegir uno: elegir el primero haria que la procedencia dependiera del
// orden del fichero, que es justo lo que el invariante 7 prohibe usar.
func recolectorDe(obs []estado.Observacion) string {
	visto := ""
	for _, o := range obs {
		switch {
		case o.Recolector == "":
			continue
		case visto == "":
			visto = o.Recolector
		case visto != o.Recolector:
			return "varios"
		}
	}
	return visto
}
