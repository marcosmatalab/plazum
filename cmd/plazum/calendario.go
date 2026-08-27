package main

// `plazum calendario`: los proximos doce meses, con su articulo.
//
// Es una orden de PRESENTACION. Todo lo que hay debajo existe desde hace
// semanas: el reloj legal (nucleo/ventana), la aplicabilidad (nucleo/aplicabilidad),
// el corpus (nucleo/corpus) y la derivacion (nucleo/pantalla). Lo que faltaba era
// que un CISO pudiera ver sus fechas sin abrir un JSON.
//
// Y es, a la vez, el test de integracion de que los relojes del corpus
// atraviesan el motor de punta a punta. Un corpus con 58 hitos y ningun sitio
// donde mirarlos es un corpus que nadie ha ejecutado nunca entero.
//
// LO QUE ESTA ORDEN NO HACE, y es deliberado:
//
//	no adivina    lo que no puede derivar NO lo ensena como si le aplicara. Un
//	              calendario que le ensena a un banco las obligaciones de un
//	              fabricante de productos sanitarios no es util, es ruido caro.
//	no esconde    y lo que no puede derivar lo CUENTA, con el motivo. La
//	              contabilidad del pie es tan importante como las fechas: sin
//	              ella, un calendario corto se lee como "no tengo nada" cuando
//	              lo que pasa es que el corpus no sabe si te alcanza.
//	no asesora    cada fila lleva su cita y su derivacion, y el descargo va al
//	              final como en `plazum explain`.

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"plazum/nucleo/corpus"
	"plazum/nucleo/pantalla"
	calendarioics "plazum/superficies/calendario"
)

// mesesEnEspanol es la unica tabla de texto de este fichero, y esta aqui y no en
// nucleo/pantalla a proposito: el nucleo emite CLAVES de catalogo (ui.mes.9) y
// quien pinta decide el idioma. La interfaz web usa adaptadores/catalogo; el CLI
// todavia no tiene catalogo propio, asi que lo resuelve aqui y se dice.
var mesesEnEspanol = map[string]string{
	"ui.mes.1": "enero", "ui.mes.2": "febrero", "ui.mes.3": "marzo",
	"ui.mes.4": "abril", "ui.mes.5": "mayo", "ui.mes.6": "junio",
	"ui.mes.7": "julio", "ui.mes.8": "agosto", "ui.mes.9": "septiembre",
	"ui.mes.10": "octubre", "ui.mes.11": "noviembre", "ui.mes.12": "diciembre",
}

var motivosEnEspanol = map[string]string{
	pantalla.MotivoPendienteDeHecho: "falta un dato que pones tu",
	pantalla.MotivoSinPlazoLegal:    "obliga y la norma no da numero",
	pantalla.MotivoSinEjecutor:      "el motor todavia no calcula esta forma de reloj",
}

type opcionesCalendario struct {
	Corpus  string
	Alcance string
	Perfil  perfilPedido
	Ahora   time.Time
	Todos   bool
	ICS     bool
}

func cmdCalendario(args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("plazum calendario", flag.ContinueOnError)
	fs.SetOutput(errores)
	dirCorpus := fs.String("corpus", "paquetes", "directorio de paquetes de corpus instalados")
	rutaAlcance := fs.String("alcance", "", "fichero con las respuestas de tu organizacion")
	pais := fs.String("pais", "", "arranque sin alcance: pais (ES)")
	sector := fs.String("sector", "", "arranque sin alcance: sector")
	empleados := fs.Int("empleados", 0, "arranque sin alcance: numero de empleados")
	todos := fs.Bool("todos-los-relojes", false,
		"no filtrar por aplicabilidad: ensena TODO reloj del corpus en vigor. Para inspeccionar "+
			"el corpus, no para saber que te aplica")
	ics := fs.Bool("ics", false, "escribe un calendario iCalendar (RFC 5545) por la salida estandar")
	ahoraTxt := fs.String("ahora", "", "instante desde el que se calculan los relojes (RFC3339); "+
		"vacio es el reloj del sistema")
	fs.Usage = func() {
		fmt.Fprintln(errores, "uso: plazum calendario --alcance FICHERO [--corpus DIR] [--ics]")
		fmt.Fprintln(errores, "     plazum calendario --pais=ES --sector=SECTOR --empleados=N [--ics]")
		fmt.Fprintln(errores, "")
		fmt.Fprintln(errores, "Las fechas de los proximos doce meses, con su articulo, agrupadas por mes.")
		fmt.Fprintln(errores, "")
		fmt.Fprintln(errores, "Con --alcance sale lo que te aplica de verdad, derivado de tus respuestas.")
		fmt.Fprintln(errores, "Con las tres banderas sale un arranque en diez segundos, y CADA fila va")
		fmt.Fprintln(errores, "marcada como supuesta: son conjeturas de un perfil, no respuestas tuyas.")
		fmt.Fprintln(errores, "")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	o := opcionesCalendario{
		Corpus: *dirCorpus, Alcance: *rutaAlcance, Todos: *todos, ICS: *ics,
		Perfil: perfilPedido{Pais: *pais, Sector: *sector, Empleados: *empleados},
	}
	if *ahoraTxt != "" {
		t, err := time.Parse(time.RFC3339, *ahoraTxt)
		if err != nil {
			fmt.Fprintf(errores, "--ahora no es un instante RFC3339: %v\n", err)
			return 2
		}
		o.Ahora = t
	} else {
		// El unico time.Now() de esta orden, y esta en el borde: de aqui hacia
		// dentro el instante viaja como dato (invariante 1).
		o.Ahora = time.Now().UTC()
	}

	if err := ejecutarCalendario(o, salida, errores); err != nil {
		fmt.Fprintln(errores, "error:", err)
		return 1
	}
	return 0
}

func ejecutarCalendario(o opcionesCalendario, w, errores io.Writer) error {
	ps, err := corpus.Cargar(o.Corpus)
	if err != nil {
		return fmt.Errorf("el corpus de %s no carga: %w.\n"+
			"  Instala uno con `plazum demo`, o apunta --corpus al directorio que tenga tus "+
			"paquetes", o.Corpus, err)
	}
	if len(ps) == 0 {
		return fmt.Errorf("no hay ni un paquete en %s.\n"+
			"  `plazum calendario` ensena las fechas del corpus INSTALADO: sin corpus no hay "+
			"nada que ensenar. Prueba `plazum demo` primero", o.Corpus)
	}

	al, supuesto, err := alcanceParaElCalendario(o)
	if err != nil {
		return err
	}

	hechos, err := fechasDelAlcance(al, o.Ahora)
	if err != nil {
		return err
	}

	var aplica pantalla.Aplicable
	var d derivacion
	switch {
	case o.Todos:
		// Se pide en voz alta, y la salida lo dice en su cabecera: esto es para
		// inspeccionar el corpus, no para saber que te aplica.
		aplica = pantalla.TodoAplica
		d, _ = montarMotor(ps, al)
	default:
		d, err = montarMotor(ps, al)
		if err != nil {
			return err
		}
		derivadas := map[string]bool{}
		for _, a := range aplicablesDe(d, al.Sujeto) {
			derivadas[a.Obligacion] = true
		}
		aplica = func(id string) (bool, bool) { return derivadas[id], supuesto }
	}

	cal := pantalla.Derivar12Meses(ps, aplica, hechos, o.Ahora)

	if o.ICS {
		return escribirICS(w, cal, al, o.Ahora)
	}
	imprimirCalendario(w, cal, al, o, d, supuesto)
	return nil
}

// alcanceParaElCalendario decide de donde salen las respuestas. Devuelve ademas
// si lo que se ha montado es un SUPUESTO, que es lo que marca cada fila.
func alcanceParaElCalendario(o opcionesCalendario) (alcance, bool, error) {
	tieneBanderas := o.Perfil.Pais != "" || o.Perfil.Sector != "" || o.Perfil.Empleados != 0
	switch {
	case o.Alcance != "" && tieneBanderas:
		return alcance{}, false, errors.New(
			"has dado --alcance Y las banderas de arranque a la vez, y no son lo mismo.\n" +
				"  --alcance son TUS respuestas; --pais/--sector/--empleados son un perfil de\n" +
				"  supuestos para ver algo en diez segundos. Elige uno: mezclarlos haria que no\n" +
				"  se supiera cual de tus fechas es una respuesta y cual una conjetura")
	case o.Alcance != "":
		al, err := cargarAlcance(o.Alcance)
		return al, false, err
	case tieneBanderas:
		al, err := alcanceDePerfil(o.Perfil)
		return al, true, err
	case o.Todos:
		// Inspeccionar el corpus no necesita sujeto.
		return alcance{Organizacion: "(sin alcance)", Sujeto: "inspeccion"}, false, nil
	default:
		return alcance{}, false, errors.New(
			"no se de quien es este calendario.\n" +
				"  Dame --alcance con tus respuestas, o las tres banderas de arranque\n" +
				"  (--pais, --sector, --empleados) para ver un perfil de ejemplo en diez segundos.\n" +
				"  Y si lo que quieres es mirar el corpus entero sin filtrar, --todos-los-relojes")
	}
}

func imprimirCalendario(w io.Writer, cal pantalla.Calendario, al alcance,
	o opcionesCalendario, d derivacion, supuesto bool) {

	quien := al.Organizacion
	if quien == "" {
		quien = al.Sujeto
	}
	fmt.Fprintf(w, "PROXIMOS DOCE MESES de %s\n", quien)
	fmt.Fprintf(w, "desde el %s hasta el %s\n\n",
		cal.Desde.Format("2006-01-02"), cal.Hasta.Format("2006-01-02"))

	if o.Todos {
		fmt.Fprintln(w, "AVISO: --todos-los-relojes. Esto NO es lo que te aplica: es todo reloj")
		fmt.Fprintln(w, "del corpus en vigor, sin filtrar por aplicabilidad. Sirve para inspeccionar")
		fmt.Fprintln(w, "el corpus, no para planificar.")
		fmt.Fprintln(w)
	} else if supuesto {
		fmt.Fprintln(w, "AVISO: esto sale de un PERFIL, no de tus respuestas. Cada fecha lleva [supuesto]")
		fmt.Fprintln(w, "y ninguna es una conclusion sobre tu organizacion: es lo que le pasaria a una")
		fmt.Fprintln(w, "empresa del perfil que has pedido. Para lo tuyo de verdad, --alcance.")
		fmt.Fprintln(w)
		var b strings.Builder
		explicarPerfil(&b, o.Perfil)
		fmt.Fprint(w, b.String())
	}

	if cal.Total() == 0 {
		fmt.Fprintln(w, "Ninguna fecha en los proximos doce meses.")
	}
	for _, m := range cal.Meses {
		fmt.Fprintf(w, "  %s de %d\n", mesesEnEspanol[m.Clave], m.Ano)
		for _, f := range m.Fechas {
			marca := ""
			if f.Supuesta {
				marca = "  [supuesto]"
			}
			fmt.Fprintf(w, "    %s  %s%s\n", f.Vence.Format("02 15:04"), f.Titulo, marca)
			fmt.Fprintf(w, "              %s  art. %s  hito %s\n", f.Marco, f.Articulo, f.Hito)
			for _, dv := range f.Divergencias {
				fmt.Fprintf(w, "              otra lectura (%s): %s\n",
					dv.Lectura, dv.Vence.Format("2006-01-02 15:04"))
			}
			if f.Aviso != "" {
				fmt.Fprintf(w, "              %s\n", f.Aviso)
			}
		}
		fmt.Fprintln(w)
	}

	// LO QUE EMPIEZA A OBLIGAR, antes que lo que ya obliga sin fecha: de todo
	// lo que este calendario puede decir, una norma que arranca dentro de la
	// ventana es lo unico que tiene fecha de caducidad como noticia. Y es la
	// unica seccion que se puede quedar vacia siendo eso una buena noticia.
	if len(cal.Estrenos) > 0 {
		fmt.Fprintf(w, "EMPIEZA A OBLIGARTE DENTRO DE ESTA VENTANA (%d)\n\n", len(cal.Estrenos))
		for _, e := range cal.Estrenos {
			marca := ""
			if e.Supuesta {
				marca = "  [supuesto]"
			}
			fmt.Fprintf(w, "    desde el %s  %s%s\n", e.Desde.Format("2006-01-02"), e.Titulo, marca)
			fmt.Fprintf(w, "              %s  art. %s\n", e.Marco, e.Articulo)
			fmt.Fprintln(w, "              hoy todavia no obliga: no hay nada que entregar "+
				"y tampoco nada que hayas incumplido")
		}
		fmt.Fprintln(w)
	}

	if len(cal.SinFecha) > 0 {
		fmt.Fprintf(w, "LO QUE OBLIGA Y NO TIENE FECHA (%d)\n\n", len(cal.SinFecha))
		for _, s := range cal.SinFecha {
			fmt.Fprintf(w, "    %s\n", s.Titulo)
			fmt.Fprintf(w, "              %s  art. %s  %s\n", s.Marco, s.Articulo,
				motivosEnEspanol[s.Motivo])
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, "LA CUENTA, ENTERA")
	fmt.Fprintf(w, "    %3d relojes instalados en %s\n", cal.RelojesDelCorpus, o.Corpus)
	fmt.Fprintf(w, "    %3d en vigor el %s\n", cal.RelojesEnVigor, cal.Desde.Format("2006-01-02"))
	fmt.Fprintf(w, "    %3d alcanzados por la aplicabilidad\n", cal.RelojesAplicables)
	fmt.Fprintf(w, "    %3d con fecha en los proximos doce meses\n", cal.Total())
	fmt.Fprintf(w, "    %3d con fecha mas alla de los doce meses\n", cal.FueraDeLaVentana)
	fmt.Fprintf(w, "    %3d sin fecha, con su motivo arriba\n", len(cal.SinFecha))
	if cal.RelojesQueEstrenan > 0 {
		// Va fuera de "en vigor" a proposito: en el instante del calculo estos
		// no lo estaban. Sumarlos alli haria que esa linea dejara de significar
		// lo que su propio nombre dice.
		fmt.Fprintf(w, "    %3d que todavia no obligan y empiezan dentro de la ventana\n",
			cal.RelojesQueEstrenan)
	}

	// EL NUMERO QUE NADIE MAS ENSENA, y es el que hace honesto a todo lo de
	// arriba: cuantos relojes viven en paquetes que NO declaran reglas, o sea
	// sobre los que este calendario no puede opinar.
	if !o.Todos {
		sinReglas := relojesEnPaquetesSinReglas(o.Corpus)
		if sinReglas > 0 {
			fmt.Fprintln(w)
			fmt.Fprintf(w, "    Y %d relojes viven en paquetes que no declaran reglas de\n", sinReglas)
			fmt.Fprintf(w, "    aplicabilidad (%d de los %d instalados). Este calendario NO puede\n",
				d.PaquetesSinReglas, d.PaquetesSinReglas+d.PaquetesConReglas)
			fmt.Fprintln(w, "    saber si te alcanzan, asi que no los ensena. Que no salgan no")
			fmt.Fprintln(w, "    significa que no te obliguen: significa que el corpus todavia no")
			fmt.Fprintln(w, "    sabe decirlo. Verlos todos: --todos-los-relojes.")
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Esto no es asesoramiento juridico. Cada fila trae su articulo y su cita para")
	fmt.Fprintln(w, "que la puedas comprobar contra el texto oficial, que es lo unico autentico.")
}

// relojesEnPaquetesSinReglas cuenta los relojes que ninguna entrada puede
// encender. Se recalcula en vez de arrastrarse para que no pueda quedarse viejo.
func relojesEnPaquetesSinReglas(dir string) int {
	ps, err := corpus.Cargar(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, p := range ps {
		if len(p.Aplicabilidad.Reglas) > 0 {
			continue
		}
		for _, o := range p.Obligaciones {
			if o.Temporalidad != nil {
				n++
			}
		}
	}
	return n
}

// escribirICS vuelca el calendario en iCalendar. El serializador vive en
// superficies/calendario y no aqui: es una SUPERFICIE (un formato de salida),
// como el export a SIEM, y tiene sus propias pruebas de formato.
func escribirICS(w io.Writer, cal pantalla.Calendario, al alcance, ahora time.Time) error {
	quien := al.Organizacion
	if quien == "" {
		quien = al.Sujeto
	}
	return calendarioics.Escribir(w, cal, calendarioics.Opciones{
		Ahora: ahora, Organizacion: quien,
	})
}
