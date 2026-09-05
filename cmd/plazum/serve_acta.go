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
	"github.com/marcosmatalab/plazum/superficies/uar"
)

// actaDeLaInstalacion compone el acta con lo que esta instalacion tiene en
// disco. Implementa superficies/acta.Actas.
//
// SE RECOMPONE EN CADA PETICION, igual que campanaEnFichero y por la misma
// razon: la campana cambia con cada hecho que se anota, y un acta cacheada al
// arrancar contaria lo que habia el dia que se levanto el servidor. Un acta que
// dice lo que era verdad hace tres semanas es peor que ninguna.
type actaDeLaInstalacion struct {
	// organizacion es la de la BANDERA, y manda. Vacia si no se dio.
	organizacion string
	// identidad es la de la INSTALACION, y se lee EN CADA PETICION.
	//
	// Es una funcion y no una cadena, y esa es la correccion que costo una
	// medida entera. Leida al arrancar, el nombre que se contesta en
	// /primer-admin NO EXISTE TODAVIA: `plazum serve` se levanta antes de
	// que nadie cree el primer administrador, asi que la instalacion mas
	// comun de todas -- la nueva -- se quedaba con el acta en su estado
	// vacio para siempre, hasta reiniciar. Se vio porque el TTFV seguia
	// cobrando la orden de terminal de esta pantalla despues de haberla
	// quitado: la medida dijo que el cable no llegaba.
	//
	// Es la misma regla que el resto de este fichero (la campana, los
	// incidentes y el programa se releen siempre), aplicada al unico dato
	// que se habia colado como constante de arranque.
	identidad func() string
	// desde y hasta son el periodo de las BANDERAS. Cuando porDefecto es
	// true no valen nada y el periodo se deriva del reloj en cada peticion:
	// un servidor levantado en marzo y mirado en abril tiene que ensenar el
	// trimestre que acaba de cerrarse, no el de cuando arranco.
	desde, hasta time.Time
	porDefecto   bool
	ahora        func() time.Time
	campana      uar.Campanas
	// incidentes es la ruta del registro, o cadena vacia si no lo hay. La
	// cadena vacia significa «no conectado», que NO es «cero incidentes».
	incidentes string
	// programa es la ruta del programa de auditoria, con el mismo criterio:
	// vacia es «no conectado», que NO es «no hubo hallazgos».
	programa string
}

// Ultima devuelve el acta. El (Acta{}, false, nil) NO es un error: es «todavia
// no hay ninguna», y la pantalla sabe pintar ese estado con su siguiente paso.
// quienEs resuelve de quien es el acta: la bandera manda, y si no hay, la
// identidad de la instalacion, leida ahora.
func (a actaDeLaInstalacion) quienEs() string {
	if a.organizacion != "" {
		return a.organizacion
	}
	if a.identidad == nil {
		return ""
	}
	return strings.TrimSpace(a.identidad())
}

// periodo da el del acta: el de las banderas, o el ultimo trimestre natural
// cerrado en el momento de mirar.
func (a actaDeLaInstalacion) periodo() (time.Time, time.Time) {
	if !a.porDefecto {
		return a.desde, a.hasta
	}
	ahora := a.ahora
	if ahora == nil {
		ahora = time.Now
	}
	return ultimoTrimestreCerrado(ahora())
}

func (a actaDeLaInstalacion) Ultima() (acta.Acta, bool, error) {
	// DE QUIEN ES, Y SI NO CONSTA NO HAY ACTA.
	//
	// El (Acta{}, false, nil) es «todavia no», no un error: una instalacion
	// que aun no ha dicho de quien es no tiene un problema, tiene un paso
	// pendiente, y la pantalla lo pinta con su siguiente paso dentro.
	org := a.quienEs()
	if org == "" {
		return acta.Acta{}, false, nil
	}
	desde, hasta := a.periodo()
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
		ID:           idDelActa(desde, hasta),
		Organizacion: org,
		Periodo:      acta.Periodo{Desde: desde, Hasta: hasta},
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
func idDelActa(desde, hasta time.Time) string {
	return fmt.Sprintf("acta-%s-%s", desde.Format("20060102"), hasta.Format("20060102"))
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
	Campana uar.Campanas
	// HayCampana dice si esos ficheros estan configurados. Se pasa explicito y
	// no se deduce de que los campos esten vacios, porque quien monta ya lo
	// sabe y deducirlo aqui seria una segunda copia de esa decision.
	HayCampana bool
	// Identidad es el nombre de la organizacion que guarda la INSTALACION,
	// preguntado una vez al crear el primer administrador.
	//
	// Es la que apaga la orden de terminal de esta pantalla, y va aparte de
	// Organizacion en vez de fundirse con ella porque son dos cosas con dos
	// precedencias: la bandera es de quien esta arrancando este proceso
	// AHORA y manda, la identidad es de la instalacion y es el defecto. Con un
	// solo campo no se puede decir cual gana sin adivinarlo.
	// SE LEE EN CADA PETICION, y por eso es una funcion. El nombre se
	// contesta en /primer-admin, o sea DESPUES de que este proceso arranque:
	// leerlo aqui una vez dejaria el acta vacia hasta el siguiente reinicio
	// en la instalacion mas comun de todas, la nueva.
	Identidad func() string
	// Ahora es el reloj del que sale el periodo por defecto. Nil es time.Now.
	Ahora func() time.Time
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
	// DE QUIEN ES EL ACTA NO SE DECIDE AQUI, y ese es el cambio.
	//
	// La bandera se conoce ahora; la identidad de la instalacion, no: se
	// contesta en /primer-admin, que ocurre despues de este arranque. Asi que
	// esta funcion decide si la CONFIGURACION vale, y quien es la
	// organizacion lo resuelve la fuente en cada peticion.
	org := strings.TrimSpace(o.Organizacion)
	pedidoAlgo := org != "" || o.Identidad != nil ||
		strings.TrimSpace(o.Desde) != "" || strings.TrimSpace(o.Hasta) != ""
	if !pedidoAlgo {
		// NI BANDERA NI FORMA DE SABER QUIEN ES. Nada que componer nunca, asi
		// que la pantalla se monta sin fuente y sale en su estado vacio.
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
	// SIN BANDERA Y SIN IDENTIDAD, con un periodo pedido, es un error de
	// arranque: quien teclea --acta-desde esta pidiendo un acta, y un acta sin
	// organizacion no es evidencia de nadie. Con identidad NO se falla aqui,
	// porque el nombre puede llegar despues, al crear el primer administrador.
	if org == "" && o.Identidad == nil {
		return nil, errors.New("se ha pedido un periodo para el acta y no consta de quien " +
			"es esta instalacion.\n" +
			"  Un acta sin organizacion no es evidencia de nadie.\n" +
			"  Arreglo: el nombre se pregunta al crear el primer administrador, en " +
			"/primer-admin. Para una instalacion que ya lo paso, --acta-organizacion.")
	}
	// EL PERIODO, Y LAS TRES RESPUESTAS SON TRES.
	//
	// LAS DOS BANDERAS O NINGUNA. Media es un error y no se completa con un
	// defecto: quien escribe --acta-desde y se olvida de --acta-hasta esta
	// pidiendo un periodo concreto, y darle otro distinto en silencio es
	// componer un acta sobre un trimestre que nadie pidio.
	//
	// NINGUNA es la nada de verdad, y ahi si hay defecto: el ULTIMO TRIMESTRE
	// NATURAL CERRADO. Se elige asi por tres cosas y las tres importan.
	// Primero, esta ENTERO EN EL PASADO, asi que el acta no dice cubrir dias
	// que todavia no han ocurrido, que es lo que pasaria con «el ano en
	// curso». Segundo, es ESTABLE: cambia cuatro veces al ano, asi que el
	// identificador del acta no se mueve cada dia, que es lo que pasaria con
	// «los ultimos doce meses hasta hoy» y llenaria el expediente de un acta
	// por jornada. Y tercero, es la cadencia con la que se revisa esto de
	// verdad. Se dice EN LA PANTALLA con sus fechas, no se supone.
	hayDesde := strings.TrimSpace(o.Desde) != ""
	hayHasta := strings.TrimSpace(o.Hasta) != ""
	if hayDesde != hayHasta {
		falta := "--acta-hasta"
		if !hayDesde {
			falta = "--acta-desde"
		}
		return nil, fmt.Errorf("para componer el acta falta %s.\n"+
			"  Van las dos o ninguna: con las dos se compone el periodo que pides, sin "+
			"ninguna el ultimo trimestre natural cerrado.\n"+
			"  Con media, plazum tendria que inventarse la otra mitad, y un acta sobre un "+
			"periodo que nadie pidio no la puede firmar nadie", falta)
	}
	var desde, hasta time.Time
	if hayDesde {
		var err error
		desde, err = fechaDelActa("--acta-desde", o.Desde)
		if err != nil {
			return nil, err
		}
		hasta, err = fechaDelActa("--acta-hasta", o.Hasta)
		if err != nil {
			return nil, err
		}
		if !hasta.After(desde) {
			return nil, fmt.Errorf("--acta-hasta (%s) no es posterior a --acta-desde (%s). Un "+
				"periodo que no avanza no contiene nada", o.Hasta, o.Desde)
		}
	}
	// LAS RUTAS SE COMPRUEBAN AL ARRANCAR, aunque el CONTENIDO se lea en cada
	// peticion. Son dos cosas distintas y las dos importan:
	//
	//	el contenido se relee siempre, porque un acta cacheada al arrancar
	//	cuenta lo que habia el dia que se levanto el servidor;
	//	la RUTA se mira ahora, porque una ruta mal escrita es un fallo del
	//	operador que esta delante del teclado EN ESTE MOMENTO. Dejarlo para la
	//	primera visita significa que el servidor arranca diciendo que todo va
	//	bien y el fallo aparece dias despues, delante de otra persona y sin
	//	nadie que recuerde que se tecleo.
	for _, r := range []struct{ bandera, ruta string }{
		{"--acta-incidentes", strings.TrimSpace(o.Incidentes)},
		{"--acta-programa", strings.TrimSpace(o.Programa)},
	} {
		if r.ruta == "" {
			continue
		}
		if _, err := os.Stat(r.ruta); err != nil {
			return nil, fmt.Errorf("%s apunta a %q y no se puede abrir: %w. "+
				"Se comprueba al arrancar a proposito: sin esto el servidor se levantaria "+
				"diciendo que todo va bien y el fallo saldria en la primera visita al acta, "+
				"dias despues y delante de otra persona", r.bandera, r.ruta, err)
		}
	}

	return &actaDeLaInstalacion{
		organizacion: org,
		identidad:    o.Identidad,
		desde:        desde, hasta: hasta,
		porDefecto: !hayDesde,
		ahora:      o.Ahora,
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

// ultimoTrimestreCerrado da el periodo por defecto del acta: el trimestre
// natural ANTERIOR al que corre, entero, con las dos fechas inclusive.
//
// ESTA ENTERO EN EL PASADO Y ESO ES LA DECISION. Un acta es un documento que
// dice lo que consta, y lo que consta de un dia que todavia no ha ocurrido no
// consta: con «el ano en curso» o «los ultimos doce meses» el periodo llegaria
// a diciembre en septiembre, y el documento afirmaria cubrir un trimestre que
// no ha pasado. Es la misma familia que el descargo de lo no constatado, en la
// otra direccion.
//
// Y ES ESTABLE: cambia cuatro veces al ano. Con una ventana movil («los doce
// meses hasta hoy»), el identificador del acta -- que sale del periodo -- se
// moveria cada dia, y el expediente acumularia un acta por jornada. El id
// existe justamente para que dos arranques sobre el mismo periodo den la
// misma acta.
func ultimoTrimestreCerrado(ahora time.Time) (desde, hasta time.Time) {
	a := ahora.UTC()
	// El primer dia del trimestre EN CURSO. La division entera es la que
	// mapea enero-marzo a 0, abril-junio a 1, y asi.
	trimestre := (int(a.Month()) - 1) / 3
	inicioDelActual := time.Date(a.Year(), time.Month(trimestre*3+1), 1, 0, 0, 0, 0, time.UTC)
	// AddDate con -3 meses cruza el ano solo, y sin el salto que tiene con
	// los dias: el dia 1 existe en los doce meses, asi que aqui no hay
	// normalizacion posible (un 31 de marzo menos un mes si la tendria).
	desde = inicioDelActual.AddDate(0, -3, 0)
	hasta = inicioDelActual.AddDate(0, 0, -1)
	return desde, hasta
}
