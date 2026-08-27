package main

// `plazum demo`: la pantalla de valor a un comando de distancia.
//
// Contra que se juzga esto. Un responsable de seguridad de una empresa de 200
// personas descarga el binario un martes por la manana y no tiene a quien
// preguntar. Si en quince minutos no ha visto algo que le sirva, cierra la
// pestana. Todo lo de este fichero existe para que ese rato sea de un minuto y
// no de quince, y para que lo que vea sea el producto de verdad y no una
// captura de pantalla animada.
//
// Las tres decisiones que sostienen eso:
//
//	empotrado    el paquete del demo viaja DENTRO del binario. Nada que
//	             descargar, ninguna red, ningun directorio que preparar.
//	reversible   todo cae en un directorio propio, marcado, y `--deshacer` lo
//	             borra entero. Un demo que ensucia la instalacion de verdad no
//	             lo ejecuta nadie dos veces.
//	relojes vivos las fechas del alcance son desplazamientos desde ahora, no
//	             fechas fijas: el demo ensena plazos corriendo hoy, no un
//	             escenario caducado de cuando se escribio.

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"plazum/nucleo/aplicabilidad"
	"plazum/nucleo/corpus"
	"plazum/nucleo/ventana"
	demoempresa "plazum/paquetes/demo-empresa"
)

// DirDemoPorDefecto es donde cae el demo si no se dice otra cosa. Relativo al
// directorio actual a proposito: el operador lo ve en su `ls` y sabe que
// borrar sin buscarlo por la maquina.
const DirDemoPorDefecto = "plazum-demo"

// nombreMarcaDemo es el fichero que dice "este directorio lo hizo el demo".
// Sin el, `--deshacer` NO borra: un comando que hace RemoveAll sobre una ruta
// que teclea el operador es un accidente esperando a pasar.
const nombreMarcaDemo = ".plazum-demo"

// horizonteDeRelojes es hasta donde se buscan ocurrencias de un ritual
// periodico. Cinco anos cubren de sobra el proximo vencimiento de cualquier
// cadencia razonable sin ponerse a enumerar hasta el fin de los tiempos.
const horizonteDeRelojes = 5 * 365 * 24 * time.Hour

type marcaDemo struct {
	Paquete   string `json:"paquete"`
	Version   string `json:"version"`
	Instalado string `json:"instalado"`
}

// opcionesDemo es lo que el operador puede cambiar.
type opcionesDemo struct {
	Dir      string
	Corpus   string
	Deshacer bool
	Ahora    time.Time
}

func cmdDemo(args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("plazum demo", flag.ContinueOnError)
	fs.SetOutput(errores)
	dir := fs.String("dir", DirDemoPorDefecto, "directorio donde se instala el demo")
	extra := fs.String("corpus", "", "directorio de corpus real a cargar ADEMAS del paquete del demo")
	deshacer := fs.Bool("deshacer", false, "borra el directorio del demo y no deja nada")
	ahoraTxt := fs.String("ahora", "", "instante desde el que se calculan los relojes (RFC3339); "+
		"vacio es el reloj del sistema")
	fs.Usage = func() {
		fmt.Fprintln(errores, "uso: plazum demo [--dir DIR] [--corpus DIR] [--ahora RFC3339]")
		fmt.Fprintln(errores, "     plazum demo --deshacer [--dir DIR]")
		fmt.Fprintln(errores, "")
		fmt.Fprintln(errores, "Instala una empresa de ejemplo con sus relojes corriendo y ensena")
		fmt.Fprintln(errores, "que obligaciones le aplican, por que, y cuando vencen. No toca nada")
		fmt.Fprintln(errores, "fuera de DIR y se deshace entero con --deshacer.")
		fmt.Fprintln(errores, "")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		// Pedir la ayuda no es un fallo. Devolver 2 aqui hace que un script
		// que llame a `plazum demo --help` se crea que algo ha ido mal.
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	o := opcionesDemo{Dir: *dir, Corpus: *extra, Deshacer: *deshacer}
	if *ahoraTxt != "" {
		t, err := time.Parse(time.RFC3339, *ahoraTxt)
		if err != nil {
			fmt.Fprintf(errores, "error: --ahora %q no es una fecha RFC3339 (2026-08-25T09:00:00Z): %v\n",
				*ahoraTxt, err)
			return 2
		}
		o.Ahora = t.UTC()
	} else {
		o.Ahora = time.Now().UTC()
	}

	if o.Deshacer {
		if err := deshacerDemo(o.Dir, salida); err != nil {
			fmt.Fprintln(errores, "error:", err)
			return 1
		}
		return 0
	}
	if err := ejecutarDemo(o, salida); err != nil {
		fmt.Fprintln(errores, "error:", err)
		return 1
	}
	return 0
}

// ---------------------------------------------------------------------------
// Instalar y deshacer
// ---------------------------------------------------------------------------

// instalarDemo vuelca el paquete empotrado en DIR/paquetes/demo-empresa.
//
// Se niega a escribir en un directorio que existe y no hizo el demo. La
// alternativa (escribir dentro de lo que haya) convierte un `--dir` mal
// tecleado en ficheros sueltos por la instalacion de alguien.
func instalarDemo(dir string, ahora time.Time) error {
	marca := filepath.Join(dir, nombreMarcaDemo)
	if fi, err := os.Stat(dir); err == nil {
		if !fi.IsDir() {
			return fmt.Errorf("%s existe y no es un directorio. Elige otro con --dir", dir)
		}
		if _, err := os.Stat(marca); err != nil {
			vacio, errV := directorioVacio(dir)
			if errV != nil {
				return errV
			}
			if !vacio {
				return fmt.Errorf("%s ya existe, tiene contenido y no lo creo el demo (no hay %s "+
					"dentro). No escribo ahi: si es tu instalacion, un demo encima te la mezcla. "+
					"Usa --dir con un directorio nuevo, o borra ese a mano si sabes lo que hay",
					dir, nombreMarcaDemo)
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("no puedo mirar %s: %w", dir, err)
	}

	destino := filepath.Join(dir, "paquetes", demoempresa.Directorio)
	if err := os.MkdirAll(destino, 0o750); err != nil {
		return fmt.Errorf("no puedo crear %s: %w. Comprueba permisos y espacio en disco", destino, err)
	}
	err := fs.WalkDir(demoempresa.Ficheros, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := demoempresa.Ficheros.ReadFile(p)
		if err != nil {
			return err
		}
		ruta := filepath.Join(destino, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(ruta), 0o750); err != nil {
			return err
		}
		return os.WriteFile(ruta, b, 0o600)
	})
	if err != nil {
		return fmt.Errorf("no puedo escribir el paquete del demo en %s: %w", destino, err)
	}

	b, err := json.MarshalIndent(marcaDemo{
		Paquete: demoempresa.Directorio, Version: "empotrada en el binario",
		Instalado: ahora.Format(time.RFC3339),
	}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(marca, b, 0o600); err != nil {
		return fmt.Errorf("no puedo marcar %s como directorio del demo: %w. Sin la marca, "+
			"`plazum demo --deshacer` se negaria a borrarlo", dir, err)
	}
	return nil
}

func deshacerDemo(dir string, salida io.Writer) error {
	marca := filepath.Join(dir, nombreMarcaDemo)
	if _, err := os.Stat(marca); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s no lo creo el demo (no tiene %s dentro), asi que no lo borro. "+
				"Si el demo esta en otro sitio, dilo con --dir; si ya lo borraste, no hay nada "+
				"que hacer", dir, nombreMarcaDemo)
		}
		return fmt.Errorf("no puedo mirar %s: %w", marca, err)
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("no puedo borrar %s: %w. Cierra lo que tenga abiertos esos ficheros "+
			"y vuelve a intentarlo", dir, err)
	}
	fmt.Fprintf(salida, "Borrado %s entero. No queda nada del demo en esta maquina.\n", dir)
	return nil
}

func hayFichero(ruta string) bool {
	fi, err := os.Stat(ruta)
	return err == nil && !fi.IsDir()
}

func directorioVacio(dir string) (bool, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return false, fmt.Errorf("no puedo leer %s: %w", dir, err)
	}
	return len(ents) == 0, nil
}

// ---------------------------------------------------------------------------
// La pantalla
// ---------------------------------------------------------------------------

func ejecutarDemo(o opcionesDemo, w io.Writer) error {
	if err := instalarDemo(o.Dir, o.Ahora); err != nil {
		return err
	}
	al, err := cargarAlcanceDemo()
	if err != nil {
		return err
	}
	ps, err := corpus.Cargar(filepath.Join(o.Dir, "paquetes"))
	if err != nil {
		return fmt.Errorf("el paquete del demo recien instalado no carga: %w. Es un fallo del "+
			"binario, no de tu maquina: vuelve a descargarlo de la release oficial", err)
	}
	if o.Corpus != "" {
		otros, err := corpus.Cargar(o.Corpus)
		if err != nil {
			return fmt.Errorf("el corpus de %s no carga: %w. Quita --corpus para ver el demo "+
				"solo, o corrige el paquete que el linter rechaza", o.Corpus, err)
		}
		ps = append(ps, otros...)
	}

	d, err := montarMotor(ps, al)
	if err != nil {
		return err
	}
	m, reglas := d.Motor, d.Reglas

	hechos, err := fechasDelAlcance(al, o.Ahora)
	if err != nil {
		return err
	}

	obligaciones := 0
	for _, p := range ps {
		obligaciones += len(p.Obligaciones)
	}
	// El encabezado dice QUE es esto antes de ensenar nada. Quien acaba de
	// descargar el binario no tiene por que saber que va a ver, y una pantalla
	// que empieza por el nombre de una empresa inventada se lee como un
	// ejemplo suelto en vez de como el producto funcionando.
	fmt.Fprintf(w, "\n  plazum calcula que obligaciones aplican a una organizacion y cuando vencen,\n")
	fmt.Fprintf(w, "  a partir del texto de las normas. Esto es una empresa de ejemplo pasada por\n")
	fmt.Fprintf(w, "  el motor de verdad.\n")
	fmt.Fprintf(w, "\n  %s\n", al.Organizacion)
	fmt.Fprintf(w, "  %s\n", ajustar(al.Descripcion, 76, "  "))
	fmt.Fprintf(w, "  Instalado en %s. Nada fuera de ahi se ha tocado.\n", o.Dir)
	fmt.Fprintf(w, "  Corpus cargado: %d paquete(s), %d obligaciones.\n", len(ps), obligaciones)
	fmt.Fprintf(w, "  Instante de calculo: %s\n", o.Ahora.Format("2006-01-02 15:04 MST"))
	if o.Corpus != "" {
		fmt.Fprintf(w, "\n  Has cargado el corpus de %s ADEMAS del demo. Ojo con lo que significa:\n", o.Corpus)
		fmt.Fprintf(w, "  el alcance de abajo es el de la empresa de ejemplo y solo responde a las\n")
		fmt.Fprintf(w, "  preguntas del paquete del demo, asi que las obligaciones de los demas\n")
		fmt.Fprintf(w, "  paquetes saldran como no aplicables. Estan cargadas y vigiladas; lo que\n")
		fmt.Fprintf(w, "  les falta es que alguien responda SU alcance, que es la pantalla que\n")
		fmt.Fprintf(w, "  `plazum serve` ensena la primera.\n")
	}

	imprimirAlcance(w, al)
	aplicables := imprimirAplicabilidad(w, m, ps, reglas, al.Sujeto)
	imprimirRelojes(w, ps, aplicables, hechos, o.Ahora)
	if err := imprimirDorados(w, ps); err != nil {
		return err
	}
	imprimirSiguientesPasos(w, o)
	return nil
}

func cargarAlcanceDemo() (alcance, error) {
	var al alcance
	b, err := demoempresa.Ficheros.ReadFile("alcance.json")
	if err != nil {
		return al, fmt.Errorf("el binario no trae el alcance del demo: %w", err)
	}
	if err := json.Unmarshal(b, &al); err != nil {
		return al, fmt.Errorf("el alcance del demo no es JSON valido: %w", err)
	}
	if al.Sujeto == "" {
		return al, errors.New("el alcance del demo no dice sobre que sujeto se razona")
	}
	return al, nil
}

func imprimirAlcance(w io.Writer, al alcance) {
	fmt.Fprintf(w, "\n1. LO QUE RESPONDIO LA EMPRESA (el alcance)\n\n")
	for _, r := range al.Respuestas {
		fmt.Fprintf(w, "   %-42s %s\n", r.Campo, r.Valor)
	}
}

func imprimirAplicabilidad(w io.Writer, m *aplicabilidad.Motor, ps []*corpus.Paquete,
	reglas map[string]corpus.ReglaSpec, sujeto string) map[string]bool {

	fmt.Fprintf(w, "\n2. LO QUE SE DERIVA DE ESO, Y POR QUE\n\n")
	aplicables := map[string]bool{}
	var lista []aplicable
	for _, h := range m.Consultar(aplicabilidad.A("aplica",
		aplicabilidad.V("O"), aplicabilidad.C(sujeto))) {
		id := h.Args[0]
		aplicables[id] = true
		idRegla := strings.TrimPrefix(m.Explicar(h), "regla ")
		idRegla = strings.TrimSuffix(idRegla, " (agregado)")
		lista = append(lista, aplicable{Obligacion: id, Regla: reglas[idRegla], IDRegla: idRegla})
	}
	if len(lista) == 0 {
		fmt.Fprintf(w, "   Ninguna obligacion aplica con estas respuestas.\n")
	}
	for _, a := range lista {
		fmt.Fprintf(w, "   APLICA  %s\n", a.Obligacion)
		if a.Regla.Regla != "" {
			fmt.Fprintf(w, "           regla   %s\n", a.Regla.Regla)
			fmt.Fprintf(w, "           segun   %s\n", a.Regla.Cita)
		} else {
			fmt.Fprintf(w, "           derivada por la regla %s\n", a.IDRegla)
		}
	}

	// Lo que NO aplica es informacion, no ausencia de informacion: es la
	// diferencia entre un catalogo de controles en frio y un alcance derivado.
	var noAplican []string
	for _, p := range ps {
		for _, ob := range p.Obligaciones {
			if !aplicables[ob.ID] {
				noAplican = append(noAplican, ob.ID)
			}
		}
	}
	sort.Strings(noAplican)
	if len(noAplican) == 0 {
		fmt.Fprintf(w, "\n   Con estas respuestas aplican todas las obligaciones instaladas.\n")
		return aplicables
	}
	// La lista de las que no aplican se acota: con el corpus real cargado son
	// cientos, y una pantalla de valor que empieza volcando trescientas lineas
	// deja de ser una pantalla de valor.
	fmt.Fprintln(w)
	const tope = 8
	for i, id := range noAplican {
		if i == tope {
			fmt.Fprintf(w, "   no aplica  ... y %d mas\n", len(noAplican)-tope)
			break
		}
		fmt.Fprintf(w, "   no aplica  %s\n", id)
	}
	fmt.Fprintf(w, "\n   Eso de arriba es la mitad que importa: %d obligacion(es) instaladas NO\n",
		len(noAplican))
	fmt.Fprintf(w, "   aplican a esta empresa, asi que no salen en ninguna pantalla, ni en el\n")
	fmt.Fprintf(w, "   informe, ni en la lista de huecos. Un catalogo de controles en frio te las\n")
	fmt.Fprintf(w, "   ensenaria todas y te dejaria a ti decidir cuales te tocan.\n")
	return aplicables
}

type filaReloj struct {
	Obligacion string
	Hito       string
	Vence      time.Time
	Falta      time.Duration
	Regla      string
}

func imprimirRelojes(w io.Writer, ps []*corpus.Paquete, aplicables map[string]bool,
	hechos ventana.Hechos, ahora time.Time) {

	fmt.Fprintf(w, "\n3. LOS RELOJES QUE CORREN AHORA MISMO\n\n")
	var filas []filaReloj
	var sinArrancar []string
	for _, p := range ps {
		for _, ob := range p.Obligaciones {
			if !aplicables[ob.ID] || ob.Temporalidad == nil {
				continue
			}
			vs, err := VencimientosDe(ob, hechos, ahora.Add(horizonteDeRelojes))
			if err != nil {
				sinArrancar = append(sinArrancar, fmt.Sprintf("%s (%v)", ob.ID, err))
				continue
			}
			v, ok := proximo(vs, ahora)
			if !ok {
				sinArrancar = append(sinArrancar, fmt.Sprintf("%s (el reloj no ha arrancado: "+
					"falta el hecho %s)", ob.ID, ob.Temporalidad.Disparador["hecho"]))
				continue
			}
			filas = append(filas, filaReloj{Obligacion: ob.ID, Hito: v.Hito,
				Vence: v.Vence, Falta: v.Vence.Sub(ahora), Regla: v.Regla})
		}
	}
	sort.Slice(filas, func(i, j int) bool { return filas[i].Vence.Before(filas[j].Vence) })
	if len(filas) == 0 {
		fmt.Fprintf(w, "   Ningun reloj ha arrancado todavia.\n")
	}
	for _, f := range filas {
		fmt.Fprintf(w, "   %-16s %s\n", cuentaAtras(f.Falta), f.Obligacion)
		fmt.Fprintf(w, "   %-16s vence el %s, hito %s\n", "", f.Vence.Format("2006-01-02 15:04 MST"), f.Hito)
	}
	for _, s := range sinArrancar {
		fmt.Fprintf(w, "   %-16s %s\n", "sin arrancar", s)
	}
	fmt.Fprintf(w, "\n   Cada fecha se puede desmontar entera con `plazum explain`: de que hecho\n")
	fmt.Fprintf(w, "   sale, que duracion se sumo y que regla de computo se aplico.\n")
}

func imprimirDorados(w io.Writer, ps []*corpus.Paquete) error {
	total := 0
	var discrepancias []error
	for _, p := range ps {
		total += len(p.Dorados)
		discrepancias = append(discrepancias, corpus.EjecutarDorados(p)...)
	}
	fmt.Fprintf(w, "\n4. LA COMPROBACION QUE HACE QUE LO DE ARRIBA VALGA ALGO\n\n")
	if fallos := len(discrepancias); fallos > 0 {
		for _, e := range discrepancias {
			fmt.Fprintf(w, "   DISCREPA  %v\n", e)
		}
		fmt.Fprintf(w, "\n   %d de %d casos dorados NO coinciden con el motor. Las fechas de arriba\n",
			fallos, total)
		fmt.Fprintf(w, "   no son de fiar. Esto es un fallo del binario: abre un issue con la\n")
		fmt.Fprintf(w, "   salida de `plazum doctor --issue`.\n")
		return fmt.Errorf("%d de %d casos dorados discrepan del motor", fallos, total)
	}
	fmt.Fprintf(w, "   %d de %d casos dorados recalculados contra el motor: todos coinciden.\n", total, total)
	fmt.Fprintf(w, "   Un caso dorado se deriva del TEXTO, no de la implementacion. Si el motor y\n")
	fmt.Fprintf(w, "   el texto discrepan, gana el texto y se arregla el motor. Eso es lo que\n")
	fmt.Fprintf(w, "   separa una fecha calculada de una fecha inventada.\n")
	return nil
}

func imprimirSiguientesPasos(w io.Writer, o opcionesDemo) {
	corpusDelDemo := filepath.ToSlash(filepath.Join(o.Dir, "paquetes"))
	pasos := []struct{ orden, porque string }{
		{"plazum doctor --corpus " + corpusDelDemo,
			"comprueba si esta maquina puede ejecutar plazum en serio, y da el arreglo de cada fallo"},
		{"plazum demo --corpus ./paquetes",
			"lo mismo con el corpus real de 30 marcos, si lo tienes al lado del binario"},
		{"plazum demo --deshacer",
			"borra " + o.Dir + " entero y no deja nada en esta maquina"},
	}
	// El expediente demo solo se ofrece SI ESTA. Sugerir un comando que va a
	// fallar por un fichero que el operador no tiene convierte la pantalla de
	// siguientes pasos en una pequena mentira, y esa es la pantalla que decide
	// si sigue mirando.
	if hayFichero("expediente-demo.json") && hayFichero("contexto-demo.json") {
		pasos = append(pasos, struct{ orden, porque string }{
			"plazum verify expediente-demo.json contexto-demo.json",
			"recalcula un expediente entero sin red y sin fiarse de quien lo emitio, que es " +
				"lo que hara tu auditor con el tuyo",
		})
	}
	fmt.Fprintf(w, "\nQUE HACER AHORA\n\n")
	for _, p := range pasos {
		fmt.Fprintf(w, "   %s\n", p.orden)
		fmt.Fprintf(w, "       %s\n\n", ajustar(p.porque, 68, "       "))
	}
	fmt.Fprintf(w, "\n   Los datos de esta empresa son inventados. El motor que los procesa es el\n")
	fmt.Fprintf(w, "   mismo que veria tu auditor. Esto no es asesoramiento juridico.\n\n")
}

// ---------------------------------------------------------------------------
// Del reloj declarado al motor de ventana
// ---------------------------------------------------------------------------

// VencimientosDe delega en nucleo/corpus, que es donde vive la UNICA
// traduccion del reloj declarado.
//
// Aqui habia una copia entera, con su propia tabla de regimenes, y su propio
// comentario decia como iba a terminar: "el dia que las dos derivas se separen,
// el test se pone rojo. Anotado como P1: el sitio correcto para esto es una
// funcion exportada de nucleo/corpus". Ese dia fue el 26-08-2026, al escribir
// el primer plazo escalonado: se enseno a una de las dos a leer los hitos
// encadenados y la otra se quedo atras.
//
// Se conserva el nombre aqui porque lo usan la demo y sus tests, y porque
// borrarlo obligaria a tocar cosas que no tienen que ver con esto.
func VencimientosDe(o corpus.Obligacion, hechos ventana.Hechos, hasta time.Time) ([]ventana.Vencimiento, error) {
	return corpus.VencimientosDe(o, hechos, hasta)
}

// regimenDeclarado convierte el regimen del fichero de datos al del motor.
// Calendario UTC sin festivos, igual que el ejecutor de dorados: los
// calendarios por pais llegan con su propia seccion del corpus.
func regimenDeclarado(r corpus.RegimenSpec) (ventana.Regimen, error) {
	reg := ventana.Regimen{Cal: ventana.NuevoCalendario("utc-v1", "demo", "corpus", time.UTC)}
	switch r.Computo {
	case "naturales", "":
		reg.Comp = ventana.Naturales
	case "habiles":
		reg.Comp = ventana.Habiles
	default:
		return reg, fmt.Errorf("computo %q no reconocido (naturales, habiles)", r.Computo)
	}
	switch r.Cierre {
	case "":
		reg.Cierre = ventana.CierreAuto
	case "exacto":
		reg.Cierre = ventana.CierreExacto
	case "fin_de_dia":
		reg.Cierre = ventana.CierreFinDia
	default:
		return reg, fmt.Errorf("cierre %q no reconocido (exacto, fin_de_dia)", r.Cierre)
	}
	switch r.Traslado {
	case "", "ninguno":
		reg.Trasl = ventana.TrasladoNinguno
	case "siguiente_habil":
		reg.Trasl = ventana.TrasladoSiguienteHabil
	default:
		return reg, fmt.Errorf("traslado %q no reconocido (ninguno, siguiente_habil)", r.Traslado)
	}
	return reg, nil
}

// proximo devuelve el primer vencimiento determinado que todavia no ha pasado.
// Si todos han pasado, devuelve el ultimo: un plazo vencido es la informacion
// mas importante de la pantalla, no algo que ocultar.
func proximo(vs []ventana.Vencimiento, ahora time.Time) (ventana.Vencimiento, bool) {
	var ultimo ventana.Vencimiento
	hay := false
	for _, v := range vs {
		if v.Estado != ventana.Determinado {
			continue
		}
		ultimo, hay = v, true
		if v.Vence.After(ahora) {
			return v, true
		}
	}
	return ultimo, hay
}

// ---------------------------------------------------------------------------
// Fechas del alcance
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Texto
// ---------------------------------------------------------------------------

// cuentaAtras dice cuanto falta en la unidad que le importa a quien lo lee. A
// nadie le sirve "quedan 3628800 segundos", y a quien tiene 42 horas no le
// sirve "quedan 1 dia".
func cuentaAtras(d time.Duration) string {
	if d < 0 {
		return "VENCIDO " + magnitud(-d)
	}
	if d < 48*time.Hour {
		return "URGENTE " + magnitud(d)
	}
	return "en " + magnitud(d)
}

func magnitud(d time.Duration) string {
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%d min", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%d h", int(d.Hours()))
	case d < 90*24*time.Hour:
		return fmt.Sprintf("%d dias", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%d meses", int(d.Hours()/24/30))
	}
}

// ajustar parte un texto largo en lineas, para que la primera pantalla no salga
// con una linea de 300 caracteres en un terminal de 80.
func ajustar(s string, ancho int, sangria string) string {
	palabras := strings.Fields(s)
	var b strings.Builder
	col := 0
	for i, p := range palabras {
		if col > 0 && col+1+len(p) > ancho {
			b.WriteString("\n" + sangria)
			col = 0
		} else if i > 0 {
			b.WriteString(" ")
			col++
		}
		b.WriteString(p)
		col += len(p)
	}
	return b.String()
}
