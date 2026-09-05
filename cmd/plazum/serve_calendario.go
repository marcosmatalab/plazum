package main

// LAS DOS PANTALLAS QUE FALTABAN, CABLEADAS: calendario y escalado.
//
// # Que pasaba antes de esto
//
// El camino guiado tiene seis pasos y DOS DE ELLOS NO ERAN PANTALLA. El camino
// lo decia en voz alta, con la orden de terminal delante de cada uno (esa es la
// puerta D11-b y evita el callejon), y aun asi el recorrido del comprador era:
// entrevista en el navegador, salir a una terminal, volver. El ultimo paso del
// producto terminaba en un bloque de texto para copiar.
//
// # De donde sale lo que pintan, y por que de aqui
//
// Las dos superficies NO derivan nada: reciben un `pantalla.Calendario` y un
// plan ya resueltos. Quien los resuelve es este fichero, con LAS MISMAS
// funciones que usa `plazum calendario` y `plazum escalado`
// (`aplicableDelAlcance`, `pantalla.Derivar12Meses`, `trabajosDelCalendario`,
// `escalado.Planificar`). Es la regla de la casa: dos derivaciones del mismo
// dato son dos derivas, y el sintoma de que se separan es que la pantalla y la
// terminal ensenan fechas distintas al mismo cliente.
//
// # El alcance se relee EN CADA PETICION
//
// Y no se cachea al arrancar, por lo mismo que el acta y la campana de accesos:
// un calendario calculado al levantar el servidor cuenta lo que vencia el dia
// que se levanto, y un servidor que lleva tres semanas en pie estaria dando las
// fechas de hace tres semanas. Ademas el fichero de alcance CAMBIA: lo reescribe
// `plazum alcance` cada vez que el operador rehace la entrevista.
//
// # Sin --alcance las dos pantallas existen igual
//
// Puerta D11-b otra vez. Sin el fichero, las dos pintan su estado vacio con la
// orden exacta que lo produce. Lo que no hacen es fingir que hay calendario ni
// que no hay avisos, que son dos afirmaciones muy distintas de «no lo se».

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	nescalado "github.com/marcosmatalab/plazum/nucleo/escalado"
	"github.com/marcosmatalab/plazum/nucleo/pantalla"
	"github.com/marcosmatalab/plazum/superficies/calendario"
	"github.com/marcosmatalab/plazum/superficies/camino"
	"github.com/marcosmatalab/plazum/superficies/escalado"
)

// aplicableDelAlcance monta el motor y devuelve el filtro de aplicabilidad.
//
// EXISTE PARA QUE HAYA UNA SOLA COPIA. Estaba escrito dos veces (en
// `ejecutarCalendario` y en `ejecutarEscalado`) y las dos copias ya se habian
// separado en un campo: la del escalado pasaba `supuesto` a false a pelo, asi
// que un plan derivado de un perfil de arranque habria salido sin marcar como
// conjetura. Hoy no es alcanzable (la orden de escalado no admite las banderas
// de perfil), pero la copia estaba puesta y esperando.
func aplicableDelAlcance(ps []*corpus.Paquete, al alcance, supuesto bool) (
	pantalla.Aplicable, derivacion, error) {

	d, err := montarMotor(ps, al)
	if err != nil {
		return nil, d, err
	}
	derivadas := map[string]bool{}
	for _, a := range aplicablesDe(d, al.Sujeto) {
		derivadas[a.Obligacion] = true
	}
	return func(id string) (bool, bool) { return derivadas[id], supuesto }, d, nil
}

// alcanceEnFichero es el alcance de la instalacion leido de disco.
//
// SE RELEE EN CADA LLAMADA. Ver el encabezado: el fichero cambia mientras el
// servidor corre, porque lo reescribe `plazum alcance`.
//
// La ruta VACIA significa «no configurado», que NO es «alcance vacio»: son las
// dos formas de la nada del invariante 8 y aqui se leen al reves. Sin ruta, las
// pantallas pintan su estado vacio con el siguiente paso; con ruta y fichero
// roto, sale el error, porque un alcance ilegible convertido en «no te vence
// nada» es la afirmacion mas cara que este producto puede hacer.
type alcanceEnFichero struct {
	ruta  string
	ahora func() time.Time
	// publicado dice que esta ruta es la del alcance que publica la
	// INSTALACION desde el navegador, y no una que haya tecleado el
	// operador con --alcance.
	//
	// LA DIFERENCIA ES QUE SIGNIFICA QUE NO ESTE. Una ruta que ha tecleado
	// una persona y no existe es un ERROR suyo, y callarlo dejaria el
	// calendario vacio con el operador convencido de que lo configuro. La
	// ruta por defecto, en cambio, no la ha tecleado nadie: que no exista
	// es «todavia nadie ha adoptado la entrevista», que es el estado vacio
	// y NO un error. Son las dos formas de la nada, y solo se distinguen
	// sabiendo quien puso la ruta.
	publicado bool
}

// leer devuelve el alcance y su calendario derivado.
func (a alcanceEnFichero) leer(ps []*corpus.Paquete) (alcance, pantalla.Calendario, error) {
	al, err := cargarAlcance(a.ruta)
	if err != nil {
		return alcance{}, pantalla.Calendario{}, err
	}
	ahora := a.ahora()
	hechos, err := fechasDelAlcance(al, ahora)
	if err != nil {
		return alcance{}, pantalla.Calendario{}, err
	}
	// SUPUESTO EN false Y ESO ES CIERTO AQUI: un alcance leido de un fichero
	// son las respuestas del operador. Los perfiles de arranque solo entran por
	// las banderas de `plazum calendario`, que este camino no tiene.
	aplica, _, err := aplicableDelAlcance(ps, al, false)
	if err != nil {
		return alcance{}, pantalla.Calendario{}, err
	}
	return al, pantalla.Derivar12Meses(ps, aplica, hechos, ahora), nil
}

// todaviaNoPublicado dice si este error es «nadie ha adoptado la entrevista
// todavia» y no un fallo.
//
// Solo lo es cuando la ruta la puso el producto (publicado) Y lo que falla es
// que el fichero no esta. Cualquier otro error de lectura -- permisos, un
// JSON que no se entiende, un alcance sin sujeto -- sigue siendo un error y se
// ensena: degradarlos todos a «todavia no» convertiria un fichero corrupto en
// una pantalla vacia plausible, que es la que nadie arregla.
func (a alcanceEnFichero) todaviaNoPublicado(err error) bool {
	return a.publicado && errors.Is(err, os.ErrNotExist)
}

// quienEsElAlcance da el rotulo con el que se presenta un alcance.
func quienEsElAlcance(al alcance) string {
	if strings.TrimSpace(al.Organizacion) != "" {
		return al.Organizacion
	}
	return al.Sujeto
}

// ---------------------------------------------------------------------------
// La fuente del calendario
// ---------------------------------------------------------------------------

type calendarioDeLaInstalacion struct {
	paquetes []*corpus.Paquete
	alcance  alcanceEnFichero
}

var _ calendario.Fuente = calendarioDeLaInstalacion{}

func (c calendarioDeLaInstalacion) Actual() (calendario.Derivado, bool, error) {
	al, cal, err := c.alcance.leer(c.paquetes)
	if err != nil {
		if c.alcance.todaviaNoPublicado(err) {
			return calendario.Derivado{}, false, nil
		}
		return calendario.Derivado{}, false, err
	}
	return calendario.Derivado{
		Calendario: cal, Organizacion: quienEsElAlcance(al), Supuesto: false,
	}, true, nil
}

// ---------------------------------------------------------------------------
// La fuente del escalado
// ---------------------------------------------------------------------------

type escaladoDeLaInstalacion struct {
	paquetes []*corpus.Paquete
	alcance  alcanceEnFichero
	// base es la direccion de ESTA instancia, para el enlace del aviso. No sale
	// de la peticion a proposito: un enlace compuesto con la cabecera Host de
	// quien pregunta es un enlace que un tercero decide, y estos enlaces acaban
	// dentro de correos.
	base string
}

var _ escalado.Fuente = escaladoDeLaInstalacion{}

// EnSeco resuelve el plan SIN TOCAR NINGUN CANAL.
//
// No hay `Canal` ni `Diario` en el lazo: se llama a `escalado.Planificar`
// directamente, que es una funcion pura. Ni siquiera se abre el diario, por lo
// mismo que la orden en seco no lo abre: anotar una intencion que no se va a
// cumplir dejaria el diario diciendo que hubo un aviso en vuelo.
func (e escaladoDeLaInstalacion) EnSeco() (escalado.Plan, bool, error) {
	al, cal, err := e.alcance.leer(e.paquetes)
	if err != nil {
		// LA MISMA DISTINCION QUE EL CALENDARIO, y aqui tambien: las dos
		// pantallas leen el mismo fichero y las dos se sirven sin sesion, asi
		// que si una dijera «todavia no» y la otra un error, la misma
		// instalacion contaria dos cosas distintas de si misma.
		if e.alcance.todaviaNoPublicado(err) {
			return escalado.Plan{}, false, nil
		}
		return escalado.Plan{}, false, err
	}
	trabajos, err := trabajosDelCalendario(e.paquetes, cal)
	if err != nil {
		return escalado.Plan{}, false, err
	}
	figuras := nescalado.Asignacion(al.Figuras)
	enlace := func(obligacion, hito string) string {
		return e.base + "/app/obligacion/" + obligacion + "#" + hito
	}

	p := escalado.Plan{
		Organizacion: quienEsElAlcance(al),
		Cuenta:       map[nescalado.Estado]int{},
		Faltas:       nescalado.FigurasSinPersona(e.paquetes, figuras),
		ComoMandar:   ordenDeMandar(e.alcance.ruta),
	}
	for _, t := range trabajos {
		pasos, err := nescalado.Planificar(t.Obligacion, t.Hito, t.Vence, t.Regimen,
			figuras, nil, enlace)
		if err != nil {
			return escalado.Plan{}, false, err
		}
		for _, paso := range pasos {
			p.Planificados++
			p.Cuenta[paso.Estado]++
		}
		p.Trabajos = append(p.Trabajos, escalado.Trabajo{
			Obligacion: t.Obligacion.ID, Titulo: t.Obligacion.Titulo,
			Hito: t.Hito, Vence: t.Vence, Pasos: pasos,
		})
	}
	return p, true, nil
}

// ordenDeMandar compone la orden que SI manda, con el fichero de alcance que
// este servidor esta usando puesto dentro.
//
// SE COMPONE CON LA RUTA REAL Y NO CON UN HUECO tipo --alcance FICHERO: esto
// sale en un bloque que invita a copiar, y una orden que hay que editar antes de
// pegarla es un callejon con luz. Es la misma leccion que se pago en el camino
// guiado con `--empleados=N`, que ni siquiera parseaba.
func ordenDeMandar(rutaAlcance string) string {
	return fmt.Sprintf("plazum escalado --alcance %s --mandar --smtp SERVIDOR:587 "+
		"--de avisos@tu-dominio --permitidos SERVIDOR", rutaAlcance)
}

// ---------------------------------------------------------------------------
// El montaje
// ---------------------------------------------------------------------------

// construirCalendario arma la pantalla del calendario.
//
// LA FUENTE PUEDE SER NIL Y LA PANTALLA SE MONTA IGUAL (puerta D11-b), como el
// acta y la revision de accesos. El tipo del parametro es CONCRETO y no
// calendario.Fuente a proposito: un puntero nil metido en un interfaz deja de
// ser nil, y aqui el valor nil tiene que poder distinguirse.
func construirCalendario(cat catalogoDeInterfaz, fuente *calendarioDeLaInstalacion) (
	*calendario.Superficie, error) {

	if cat == nil {
		return nil, fmt.Errorf("calendario: falta el catalogo")
	}
	var f calendario.Fuente
	if fuente != nil {
		f = *fuente
	}
	return calendario.NuevaPantalla(calendario.OpcionesPantalla{
		Fuente:      f,
		Catalogo:    cat,
		Base:        prefijoDeLaPantalla(camino.IDDelCalendario, calendario.BasePorDefecto),
		Estatico:    "/estatico",
		CaminoRuta:  camino.BasePorDefecto + "/",
		CaminoClave: camino.ClaveTitulo,
		// EL CAMINO ENTERO PARA LA BARRA LATERAL, y se pasa explicitamente por
		// lo mismo que a las demas: una superficie que se rellenara sola el
		// camino cuando llega vacio convertiria un olvido de aqui en una barra
		// plausible que enlaza a donde nadie ha montado nada.
		Pasos: camino.Canonico(),
		// La raiz del SITIO, no la de esta pantalla: el enlace de la marca va a
		// la portada y no a /calendario/.
		Raiz: "",
	})
}

// construirEscalado arma la pantalla del plan de avisos. Mismo criterio.
func construirEscalado(cat catalogoDeInterfaz, quien func(*http.Request) string,
	fuente *escaladoDeLaInstalacion) (*escalado.Superficie, error) {

	if cat == nil {
		return nil, fmt.Errorf("escalado: falta el catalogo")
	}
	var f escalado.Fuente
	if fuente != nil {
		f = *fuente
	}
	return escalado.Nuevo(escalado.Opciones{
		Fuente:      f,
		Catalogo:    cat,
		Base:        prefijoDeLaPantalla(camino.IDDelEscalado, escalado.BasePorDefecto),
		Estatico:    "/estatico",
		Quien:       quien,
		CaminoRuta:  camino.BasePorDefecto + "/",
		CaminoClave: camino.ClaveTitulo,
		Pasos:       camino.Canonico(),
		Raiz:        "",
	})
}

// prefijoDeLaPantalla da el prefijo bajo el que se monta una superficie.
//
// LEE PRIMERO LA DECLARACION DEL CAMINO, que es la regla de la casa: el prefijo
// sale de ahi y no de una cadena escrita aqui, porque dos copias del mismo dato
// se separan y el sintoma es un 404 en un paso del camino.
//
// Y CAE A LA BASE QUE DECLARA LA PROPIA SUPERFICIE MIENTRAS EL CAMINO NO LA
// DECLARE, que es lo que pasa hoy con estas dos: `camino.Canonico()` las tiene
// como pasos SIN `Ruta`, o sea todavia no son pantalla para el camino. El
// fichero del camino es de otro frente y no se toca desde aqui; la peticion con
// el diff exacto va en el informe. Mientras tanto la superficie se monta y se
// llega a ella tecleando la direccion, que es peor que estar en el camino y
// mucho mejor que no existir.
//
// LA CAIDA TIENE SU GUARDA: hay un test que exige que, en cuanto el camino
// declare la ruta de uno de estos pasos, esa ruta sea EXACTAMENTE la base que
// declara su superficie. Sin esa guarda, el dia que el camino declarara
// "/calendario/" o "/agenda" la pantalla seguiria montada en otro sitio y el
// enlace del camino daria 404 sin que nada se pusiera rojo.
func prefijoDeLaPantalla(id, base string) string {
	if p := prefijoDelPaso(id); p != "" {
		return strings.TrimSuffix(p, "/")
	}
	return base
}

// enlaceDeLaInstancia compone la direccion de ESTA instalacion.
//
// SALE DE LA DIRECCION DE ESCUCHA Y NO DE LA CABECERA Host DE LA PETICION, y
// esa es la unica decision de esta funcion. Un enlace compuesto con el Host que
// manda quien pregunta es un enlace que decide un tercero, y estos enlaces
// acaban dentro de correos que salen de la organizacion: quien pudiera hacerle
// una peticion al servidor podria decidir a que sitio manda plazum a su
// responsable de cumplimiento.
//
// LA DIRECCION DE ESCUCHA NO ES UNA URL Y AQUI NO SE FINGE QUE LO SEA: ":8443"
// escucha en todas las interfaces y no dice por cual se llega. Cuando no empieza
// por un anfitrion se cae a localhost, que es cierto siempre (el servidor
// tambien escucha ahi) y no inventa un nombre de maquina.
func enlaceDeLaInstancia(direccion string, conTLS bool) string {
	esquema := "http"
	if conTLS {
		esquema = "https"
	}
	d := strings.TrimSpace(direccion)
	if d == "" || strings.HasPrefix(d, ":") {
		d = "localhost" + d
	}
	return esquema + "://" + d
}

// existeFichero dice si una ruta se puede abrir. Se usa al arrancar para no
// dejar que el servidor se levante con un --alcance mal escrito y que el fallo
// aparezca dias despues, delante de otra persona: es el mismo criterio que las
// rutas del acta.
func existeFichero(ruta string) error {
	_, err := os.Stat(ruta)
	return err
}
