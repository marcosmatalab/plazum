package main

// LA FUENTE DEL ACTA, LEIDA DE DISCO.
//
// # Que pasaba antes de esto
//
// La pantalla del acta se montaba SIN FUENTE. Estaba entera, con sus tests en
// verde y su estado vacio bien escrito, y no habia forma de que ensenara nada
// aunque la instalacion tuviera datos: `plazum serve` le pasaba `Fuente: nil` y
// punto. El acta es la demo de venta del producto (el acta 9.3 delante de un
// consejo), asi que la mejor pantalla de la v1 estaba en blanco permanente.
//
// # Lo que se cablea aqui, y lo que NO, dicho con su cardinal
//
// Un acta se compone de TRES entradas (nucleo/acta.Entradas): el programa de
// auditoria, la campana de revision de accesos y el registro de incidentes. LAS
// TRES SE PUEDEN LEER YA, y esto las cablea:
//
//	campana de accesos   SI. Ya tenia lector y esta probado: censo.Tomar lee el
//	                     CSV y accesos.Reconstruir replica el ledger. Es
//	                     exactamente lo que campanaEnFichero hace para la UAR, y
//	                     por eso se reutiliza esa misma pieza en vez de escribir
//	                     una segunda.
//	programa de auditoria  SI, desde el 02-09-2026: auditoria.Reconstruir lo
//	                     replica por Abrir, Auditar, Diferir, Anotar y Cerrar,
//	                     asi que un programa leido de disco pasa por las mismas
//	                     reglas que uno construido a mano.
//	registro de incidentes SI, desde el 02-09-2026: nucleo/incidente.Reconstruir
//	                     lee el fichero y lo REPLICA por Abrir y Registrar, asi
//	                     que un incidente leido de disco pasa por las mismas
//	                     reglas que uno creado a mano.
//
// # Por que se monta igual con una de tres
//
// Porque el acta YA SABE decirlo. nucleo/acta.Componer pinta SIEMPRE las cuatro
// secciones, en el orden del vocabulario, con la que no tiene datos diciendo
// que le falta.
//
// Y `HayRegistroDeIncidentes` ES EL CAMPO QUE HACE ESTO POSIBLE, porque separa
// las dos formas de la nada (invariante 8): «cero incidentes en el periodo» es
// una NOTICIA y «nadie ha conectado el registro» es un HUECO, y en un acta se
// leen al reves. Aqui se pone a true SOLO cuando el operador ha dado el fichero
// y se ha podido leer, que es la unica condicion bajo la cual plazum puede
// afirmar que no hubo incidentes.

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/acta"
	"github.com/marcosmatalab/plazum/nucleo/auditoria"
	"github.com/marcosmatalab/plazum/nucleo/incidente"
)

// actaDeLaInstalacion compone el acta con lo que esta instalacion tiene en
// disco. Implementa superficies/acta.Actas.
//
// SE RECOMPONE EN CADA PETICION, igual que campanaEnFichero y por la misma
// razon: la campana cambia con cada hecho que se anota, y un acta cacheada al
// arrancar contaria lo que habia el dia que se levanto el servidor. Un acta que
// dice lo que era verdad hace tres semanas es peor que ninguna.
type actaDeLaInstalacion struct {
	organizacion string
	desde, hasta time.Time
	campana      campanaEnFichero
	// incidentes es la ruta del registro, o cadena vacia si no lo hay. La
	// cadena vacia significa «no conectado», que NO es «cero incidentes».
	incidentes string
	// programa es la ruta del programa de auditoria, con el mismo criterio:
	// vacia es «no conectado», que NO es «no hubo hallazgos».
	programa string
}

// Ultima devuelve el acta. El (Acta{}, false, nil) NO es un error: es «todavia
// no hay ninguna», y la pantalla sabe pintar ese estado con su siguiente paso.
func (a actaDeLaInstalacion) Ultima() (acta.Acta, bool, error) {
	c, err := a.campana.Abierta()
	if err != nil {
		// UN FALLO AL LEER NO SE CONVIERTE EN «no hay acta». Son dos cosas
		// distintas y solo una es inocua: si el ledger no se puede leer, quien
		// mira la pantalla tiene que saber que hay algo roto, no creerse que
		// todavia no ha empezado. Es el invariante 8 en la pantalla que mas
		// circula del producto.
		return acta.Acta{}, false, err
	}
	// EL REGISTRO DE INCIDENTES, SI ESTA CONECTADO. Se lee en cada peticion,
	// igual que la campana: un acta cacheada al arrancar contaria los
	// incidentes que habia el dia que se levanto el servidor.
	var incidentes []*incidente.Incidente
	hayRegistro := false
	if strings.TrimSpace(a.incidentes) != "" {
		datos, err := os.ReadFile(a.incidentes) // #nosec G304 -- la ruta la da el operador al arrancar
		if err != nil {
			return acta.Acta{}, false, fmt.Errorf("no se puede leer el registro de "+
				"incidentes: %w", err)
		}
		incidentes, err = incidente.Reconstruir(datos)
		if err != nil {
			// UN REGISTRO ILEGIBLE NO SE CONVIERTE EN «no hubo incidentes».
			// Seria la afirmacion mas cara que este documento puede hacer, y
			// saldria de un fichero roto.
			return acta.Acta{}, false, fmt.Errorf("el registro de incidentes no se "+
				"entiende: %w", err)
		}
		hayRegistro = true
	}

	// EL PROGRAMA DE AUDITORIA, SI ESTA CONECTADO. El nil sigue significando
	// «esta fuente no esta conectada»: aqui solo deja de serlo cuando el
	// operador da el fichero y se puede leer.
	var programa *auditoria.Programa
	if strings.TrimSpace(a.programa) != "" {
		datos, err := os.ReadFile(a.programa) // #nosec G304 -- la ruta la da el operador al arrancar
		if err != nil {
			return acta.Acta{}, false, fmt.Errorf("no se puede leer el programa de "+
				"auditoria: %w", err)
		}
		programa, err = auditoria.Reconstruir(datos)
		if err != nil {
			// UN PROGRAMA ILEGIBLE NO SE CONVIERTE EN «no hubo hallazgos», por
			// lo mismo que el registro de incidentes.
			return acta.Acta{}, false, fmt.Errorf("el programa de auditoria no se "+
				"entiende: %w", err)
		}
	}

	compuesta, err := acta.Componer(acta.Entradas{
		ID:           a.id(),
		Organizacion: a.organizacion,
		Periodo:      acta.Periodo{Desde: a.desde, Hasta: a.hasta},
		Campana:      c,
		// LOS NOMBRES DEL CENSO NO VIAJAN. El valor cero es el restrictivo y
		// aqui se deja a proposito: el acta se imprime y se manda por correo, y
		// un consejo no necesita saber que cuenta tiene quien; necesita cuantos
		// accesos quedaron sin revisar. Quien quiera ver a las personas tiene la
		// pantalla de la UAR, que es de quien es ese dato.
		ConNombresDelCenso: false,
		// EL REGISTRO DE INCIDENTES, con las dos formas de la nada separadas:
		// hayRegistro es true solo si el operador lo conecto Y se leyo.
		HayRegistroDeIncidentes: hayRegistro,
		Incidentes:              incidentes,
		// EL PROGRAMA. nil sigue diciendo «esta fuente no esta conectada», que
		// es distinto de «no hubo hallazgos».
		Programa: programa,
	})
	if err != nil {
		return acta.Acta{}, false, err
	}
	return compuesta, true, nil
}

// id da el identificador del acta. Sale del periodo y no de un contador: dos
// arranques del servidor sobre el mismo periodo tienen que dar el mismo id, o
// el expediente tendria dos actas distintas del mismo trimestre.
func (a actaDeLaInstalacion) id() string {
	return fmt.Sprintf("acta-%s-%s", a.desde.Format("20060102"), a.hasta.Format("20060102"))
}

// opcionesActa es lo que el operador configura para el acta.
type opcionesActa struct {
	Organizacion string
	Desde        string
	Hasta        string
	// Incidentes es la ruta del registro. Vacia significa no conectado.
	Incidentes string
	// Programa es la ruta del programa de auditoria. Vacia, no conectado.
	Programa string
	// Campana son los mismos ficheros que ya lee la pantalla de la UAR. NO SE
	// PIDEN DOS VECES: el acta se compone de la campana, asi que configurarla
	// aparte permitiria que las dos pantallas ensenaran campanas distintas y
	// nadie sabria cual manda.
	Campana campanaEnFichero
	// HayCampana dice si esos ficheros estan configurados. Se pasa explicito y
	// no se deduce de que los campos esten vacios, porque quien monta ya lo
	// sabe y deducirlo aqui seria una segunda copia de esa decision.
	HayCampana bool
}

// fuenteDelActa decide si hay algo que componer.
//
// LAS TRES RESPUESTAS, y son tres y no dos:
//
//	nil, nil        no hay nada configurado. La pantalla sale en su estado
//	                vacio contando de que se compone un acta, que es la puerta
//	                D11-b y es lo que hay hoy.
//	fuente, nil     hay campana y periodo: se compone.
//	nil, err        hay algo configurado y esta MAL. No se degrada a la
//	                pantalla vacia: eso convertiria un error del operador en
//	                una pantalla plausible, y la plausible es la que nadie
//	                arregla. Es la tercera forma del invariante 8, presente y
//	                no interpretable, que aqui aparece en dos fechas.
func fuenteDelActa(o opcionesActa) (*actaDeLaInstalacion, error) {
	pedidoAlgo := strings.TrimSpace(o.Organizacion) != "" ||
		strings.TrimSpace(o.Desde) != "" || strings.TrimSpace(o.Hasta) != ""
	if !pedidoAlgo {
		return nil, nil
	}
	if !o.HayCampana {
		return nil, errors.New("has configurado el acta y no has dado la campana de revision " +
			"de accesos.\n" +
			"  El acta se compone de lo que ya consta, y hoy la unica de sus tres fuentes que\n" +
			"  plazum sabe leer de disco es la campana.\n" +
			"  Arreglo: anade --accesos-fichero, --accesos-ledger y --accesos-campana, que son\n" +
			"  las mismas que usa la pantalla de revision de accesos.")
	}
	faltan := []string{}
	if strings.TrimSpace(o.Organizacion) == "" {
		faltan = append(faltan, "--acta-organizacion (un acta sin organizacion no es evidencia "+
			"de nadie)")
	}
	if strings.TrimSpace(o.Desde) == "" {
		faltan = append(faltan, "--acta-desde")
	}
	if strings.TrimSpace(o.Hasta) == "" {
		faltan = append(faltan, "--acta-hasta")
	}
	if len(faltan) > 0 {
		return nil, fmt.Errorf("para componer el acta falta %s.\n"+
			"  Sin periodo no se puede decir que incidente entra y cual no, asi que no se\n"+
			"  compone a medias: se dice que falta", strings.Join(faltan, "; "))
	}
	desde, err := fechaDelActa("--acta-desde", o.Desde)
	if err != nil {
		return nil, err
	}
	hasta, err := fechaDelActa("--acta-hasta", o.Hasta)
	if err != nil {
		return nil, err
	}
	if !hasta.After(desde) {
		return nil, fmt.Errorf("--acta-hasta (%s) no es posterior a --acta-desde (%s). Un "+
			"periodo que no avanza no contiene nada", o.Hasta, o.Desde)
	}
	return &actaDeLaInstalacion{
		organizacion: strings.TrimSpace(o.Organizacion),
		desde:        desde, hasta: hasta,
		campana:    o.Campana,
		incidentes: strings.TrimSpace(o.Incidentes),
		programa:   strings.TrimSpace(o.Programa),
	}, nil
}

// fechaDelActa lee una fecha del operador.
//
// UNA FECHA QUE NO SE ENTIENDE ES UN ERROR, NUNCA EL CERO. Es la tercera
// hermana del invariante 8 en la frontera de entrada: campo ausente, campo
// vacio y campo presente que no se entiende son tres cosas, y solo las dos
// primeras son la nada. Un time.Time cero aqui daria un periodo que empieza en
// el ano 1 y un acta que dice cubrir dos milenios.
func fechaDelActa(bandera, v string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(v))
	if err != nil {
		return time.Time{}, fmt.Errorf("%s no se entiende: %q no es una fecha AAAA-MM-DD",
			bandera, v)
	}
	return t.UTC(), nil
}
