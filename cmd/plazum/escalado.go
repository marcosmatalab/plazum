package main

// `plazum escalado`: que avisos saldrian hoy, y a quien.
//
// EN SECO POR DEFECTO, Y HAY QUE PEDIR MANDAR. Un comando que manda correos al
// ejecutarlo sin bandera es un comando que alguien ejecuta una vez para ver que
// hace y le llega a media empresa. El valor cero de esta orden es el
// restrictivo (invariante 8): ensena el plan, no lo ejecuta. Y en seco es util
// por si solo, porque contesta la pregunta que el operador tiene el dia uno:
// "si esto se pusiera a avisar ahora, ¿a quien escribiria y de que?".
//
// EL INSTANTE ENTRA COMO DATO desde aqui hacia dentro. Este es uno de los
// bordes donde vive un `time.Now()` (invariante 1).

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"sort"
	"time"

	"github.com/marcosmatalab/plazum/adaptadores/canal"
	"github.com/marcosmatalab/plazum/adaptadores/escalador"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/escalado"
	"github.com/marcosmatalab/plazum/nucleo/pantalla"
)

type opcionesEscalado struct {
	Corpus, Alcance, Diario string
	Ahora                   time.Time
	Mandar                  bool
	Base                    string
	// Los canales, si se manda de verdad.
	SMTP, De, Teams string
	Permitidos      []string
}

func cmdEscalado(args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("plazum escalado", flag.ContinueOnError)
	fs.SetOutput(errores)
	dirCorpus := fs.String("corpus", "paquetes", "directorio de paquetes instalados")
	rutaAlcance := fs.String("alcance", "", "fichero con las respuestas de tu organizacion")
	diario := fs.String("diario", "escalado.jsonl", "diario de avisos (intenciones y cursos)")
	base := fs.String("base", "http://localhost:8080", "direccion de TU instancia, para el enlace")
	mandar := fs.Bool("mandar", false, "manda los avisos de verdad; sin esto solo se ensenan")
	smtp := fs.String("smtp", "", "servidor SMTP como anfitrion:puerto")
	de := fs.String("de", "", "remitente de los correos")
	teams := fs.String("teams", "", "webhook entrante de Teams (https)")
	permitidos := fs.String("permitidos", "", "anfitriones a los que se puede escribir, separados por coma")
	ahoraTxt := fs.String("ahora", "", "instante desde el que se juzga (RFC3339); vacio es el reloj")
	fs.Usage = func() {
		fmt.Fprintln(errores, "uso: plazum escalado --alcance FICHERO [--corpus DIR] [--mandar]")
		fmt.Fprintln(errores, "")
		fmt.Fprintln(errores, "Ensena que avisos de escalado saldrian hoy y a quien. NO MANDA NADA")
		fmt.Fprintln(errores, "salvo que se lo pidas con --mandar: un comando que escribe a")
		fmt.Fprintln(errores, "personas al ejecutarlo para ver que hace es un comando peligroso.")
		fmt.Fprintln(errores, "")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	o := opcionesEscalado{
		Corpus: *dirCorpus, Alcance: *rutaAlcance, Diario: *diario, Base: *base,
		Mandar: *mandar, SMTP: *smtp, De: *de, Teams: *teams,
	}
	if *permitidos != "" {
		o.Permitidos = separarPorComa(*permitidos)
	}
	if *ahoraTxt != "" {
		t, err := time.Parse(time.RFC3339, *ahoraTxt)
		if err != nil {
			fmt.Fprintf(errores, "--ahora no es un instante RFC3339: %v\n", err)
			return 2
		}
		o.Ahora = t
	} else {
		o.Ahora = time.Now().UTC()
	}

	if err := ejecutarEscalado(o, salida, errores); err != nil {
		fmt.Fprintln(errores, "error:", err)
		return 1
	}
	return 0
}

func separarPorComa(s string) []string {
	var out []string
	for _, x := range splitYRecortar(s, ',') {
		if x != "" {
			out = append(out, x)
		}
	}
	return out
}

func splitYRecortar(s string, sep rune) []string {
	var out []string
	actual := ""
	for _, r := range s {
		if r == sep {
			out = append(out, recortar(actual))
			actual = ""
			continue
		}
		actual += string(r)
	}
	return append(out, recortar(actual))
}

func recortar(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

func ejecutarEscalado(o opcionesEscalado, w, errores io.Writer) error {
	ps, err := corpus.Cargar(o.Corpus)
	if err != nil {
		return fmt.Errorf("el corpus de %s no carga: %w", o.Corpus, err)
	}
	al, _, err := alcanceParaElCalendario(opcionesCalendario{
		Corpus: o.Corpus, Alcance: o.Alcance, Ahora: o.Ahora,
	})
	if err != nil {
		return err
	}
	hechos, err := fechasDelAlcance(al, o.Ahora)
	if err != nil {
		return err
	}
	d, err := montarMotor(ps, al)
	if err != nil {
		return err
	}
	derivadas := map[string]bool{}
	for _, a := range aplicablesDe(d, al.Sujeto) {
		derivadas[a.Obligacion] = true
	}
	cal := pantalla.Derivar12Meses(ps, func(id string) (bool, bool) {
		return derivadas[id], false
	}, hechos, o.Ahora)

	trabajos, err := trabajosDelCalendario(ps, cal)
	if err != nil {
		return err
	}

	lazo := &escalador.Lazo{
		Figuras: escalado.Asignacion(al.Figuras),
		Enlace: func(obligacion, hito string) string {
			return o.Base + "/app/obligacion/" + obligacion + "#" + hito
		},
		Destino: func(persona string) (string, string) { return destinoDe(persona, o) },
	}
	if !o.Mandar {
		return enSeco(w, lazo, trabajos, o)
	}

	men, err := canalDeVerdad(o)
	if err != nil {
		return err
	}
	dia, err := escalador.AbrirDiario(o.Diario)
	if err != nil {
		return err
	}
	lazo.Canal, lazo.Diario = men, dia
	r, err := lazo.Vuelta(o.Ahora, trabajos)
	imprimirResumen(w, r)
	return err
}

// trabajosDelCalendario convierte las fechas derivadas en trabajos del lazo.
//
// SOLO LAS OBLIGACIONES QUE DECLARAN ESCALADO. Una sin escalones no es un
// trabajo del lazo: es una fila del calendario, y confundirlas haria que el
// resumen contara como "sin destinatario" obligaciones que nunca quisieron
// avisar a nadie.
func trabajosDelCalendario(ps []*corpus.Paquete, cal pantalla.Calendario) ([]escalador.Trabajo, error) {
	porID := map[string]corpus.Obligacion{}
	for _, p := range ps {
		for _, ob := range p.Obligaciones {
			porID[ob.ID] = ob
		}
	}
	var out []escalador.Trabajo
	// Las fechas viven repartidas por meses. Se recorren todos: coger solo el
	// mes en curso dejaria fuera los avisos de cortesia, que son justo los que
	// se mandan con sesenta dias de antelacion.
	for _, m := range cal.Meses {
		for _, f := range m.Fechas {
			ob, hay := porID[f.Obligacion]
			if !hay || len(ob.Escalado) == 0 {
				continue
			}
			reg, err := corpus.RegimenDe(ob)
			if err != nil {
				return nil, fmt.Errorf("obligacion %s: %w", ob.ID, err)
			}
			out = append(out, escalador.Trabajo{Obligacion: ob, Hito: f.Hito, Vence: f.Vence,
				Regimen: reg})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Vence.Before(out[j].Vence) })
	return out, nil
}

// destinoDe traduce una persona a canal y direccion.
//
// De momento es tonto a proposito: si hay SMTP configurado, correo; si hay
// webhook, Teams. El mapeo persona -> direccion por canal es dato del alcance y
// llega con la ingesta manual firmada; hasta entonces se dice lo que se hace en
// vez de fingir que se sabe.
func destinoDe(persona string, o opcionesEscalado) (string, string) {
	if persona == "" {
		return "", ""
	}
	if o.SMTP != "" {
		return "email", persona
	}
	if o.Teams != "" {
		return "teams", persona
	}
	return "", ""
}

func canalDeVerdad(o opcionesEscalado) (*canal.Mensajero, error) {
	var trs []canal.Transporte
	if o.SMTP != "" {
		if o.De == "" {
			return nil, errors.New("con --smtp hace falta --de: un correo sin remitente lo " +
				"rechaza cualquier servidor")
		}
		e, err := canal.NuevoEmail(o.SMTP, o.De, enviarPorSMTP)
		if err != nil {
			return nil, err
		}
		trs = append(trs, e)
	}
	if o.Teams != "" {
		t, err := canal.NuevoTeams(o.Teams, &http.Client{Timeout: 15 * time.Second})
		if err != nil {
			return nil, err
		}
		trs = append(trs, t)
	}
	if len(trs) == 0 {
		return nil, errors.New("--mandar sin ningun canal configurado. Anade --smtp (con --de) " +
			"o --teams, y --permitidos con sus anfitriones")
	}
	return canal.Nuevo(canal.Config{Permitidos: o.Permitidos}, trs...)
}

func imprimirResumen(w io.Writer, r escalador.Resumen) {
	fmt.Fprintf(w, "\nVUELTA DEL %s\n", r.Vuelta.Format(time.RFC3339))
	fmt.Fprintf(w, "  avisos planificados: %d\n", r.Planificados)
	for _, e := range escalado.EstadosPosibles() {
		if n := r.Cuenta[e]; n > 0 {
			fmt.Fprintf(w, "  %-32s %d\n", e, n)
		}
	}
	if r.YaCerrados > 0 {
		fmt.Fprintf(w, "  %-32s %d\n", "ya avisados en otra vuelta", r.YaCerrados)
	}
	if r.Reintentos > 0 {
		fmt.Fprintf(w, "  %-32s %d  (quedaron en vuelo tras un corte)\n", "reenviados", r.Reintentos)
	}
	// LA LEY DE CONSERVACION, IMPRESA. Si la suma no cuadra, hay avisos que no
	// estan en ningun sitio y quien lea esto tiene que verlo.
	if r.Suma() != r.Planificados {
		fmt.Fprintf(w, "\n  AVISO: los cubos suman %d y se planificaron %d. Faltan %d avisos "+
			"por explicar, y eso es un fallo del producto, no tuyo\n",
			r.Suma(), r.Planificados, r.Planificados-r.Suma())
	}
}

// enSeco ensena el plan sin tocar ningun canal.
//
// Es el modo por defecto, y contesta la pregunta del dia uno: si esto se
// pusiera a avisar ahora, a quien escribiria y de que. No abre el diario
// siquiera: en seco no se anota nada, porque anotar una intencion que no se va
// a cumplir dejaria el diario diciendo que hubo un aviso en vuelo.
func enSeco(w io.Writer, l *escalador.Lazo, trabajos []escalador.Trabajo, o opcionesEscalado) error {
	fmt.Fprintf(w, "EN SECO: esto es lo que saldria, y NO se ha mandado nada.\n")
	fmt.Fprintf(w, "Para mandarlo de verdad: --mandar, con --smtp/--de o --teams y --permitidos.\n\n")
	cuenta := map[escalado.Estado]int{}
	total := 0
	for _, t := range trabajos {
		pasos, err := escalado.Planificar(t.Obligacion, t.Hito, t.Vence, t.Regimen,
			l.Figuras, l.Silencios, l.Enlace)
		if err != nil {
			return err
		}
		for _, p := range pasos {
			total++
			cuenta[p.Estado]++
			if p.Aviso == nil {
				fmt.Fprintf(w, "  [%s] %s / %s escalon %d: %s\n", p.Estado, t.Obligacion.ID,
					t.Hito, p.Nivel, p.Motivo)
				continue
			}
			canal, direccion := "", ""
			if l.Destino != nil {
				canal, direccion = l.Destino(p.Persona)
			}
			donde := fmt.Sprintf("%s por %s", direccion, canal)
			if canal == "" {
				donde = "SIN CANAL configurado todavia"
			}
			fmt.Fprintf(w, "  [saldria] %s el %s -> %s (%s), escalon %d\n",
				t.Obligacion.ID, p.Cuando.Format("2006-01-02"), p.Figura, donde, p.Nivel)
		}
	}
	fmt.Fprintf(w, "\n  avisos planificados: %d\n", total)
	for _, e := range escalado.EstadosPosibles() {
		if n := cuenta[e]; n > 0 {
			fmt.Fprintf(w, "  %-32s %d\n", e, n)
		}
	}
	suma := 0
	for _, n := range cuenta {
		suma += n
	}
	if suma != total {
		fmt.Fprintf(w, "\n  AVISO: los cubos suman %d y se planificaron %d\n", suma, total)
	}
	if total == 0 {
		fmt.Fprintf(w, "\n  No hay ningun aviso que dar en los proximos doce meses con tus "+
			"respuestas. Esto NO dice que no tengas obligaciones: dice que ninguna de las "+
			"que te alcanzan declara escalones, o que ninguna vence en la ventana.\n")
	}
	_ = o
	return nil
}

// enviarPorSMTP es el envio de verdad. Vive aqui, en el borde, y se le pasa al
// adaptador: asi el adaptador se puede probar sin servidor y la credencial no
// tiene que atravesar ninguna capa.
func enviarPorSMTP(direccion, de string, a []string, mensaje []byte) error {
	return smtp.SendMail(direccion, nil, de, a, mensaje)
}
