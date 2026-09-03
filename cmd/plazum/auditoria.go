package main

// `plazum auditoria`: LA ORDEN QUE ESCRIBE EL PROGRAMA DE AUDITORIA.
//
// # El hueco que cierra
//
// El mismo que el de `plazum incidentes` y la otra mitad de la misma pieza:
// `nucleo/auditoria` sabia LEER un programa de disco y `plazum serve
// --acta-programa` lo montaba en la pantalla del acta, y nada lo ESCRIBIA. De
// las tres fuentes del acta, la campana de accesos tenia su orden desde hacia
// semanas (`plazum accesos`) y las otras dos no tenian ninguna.
//
// # El alcance del programa SALE DEL CORPUS Y DEL ALCANCE, no de tu teclado
//
// Es la decision de esta orden. Un programa de auditoria 9.2 se abre sobre unas
// UNIDADES, y una unidad es una obligacion concreta de un paquete concreto. Si
// esa lista se teclea, dos cosas pasan siempre: se queda corta, y se queda vieja
// el dia que el corpus gana un paquete. Aqui se deriva de lo que ya sabe el
// producto: se cargan los paquetes instalados, se pasa el alcance por el motor
// de aplicabilidad y el alcance del programa son EXACTAMENTE las obligaciones
// que te alcanzan.
//
// Eso ademas cierra el circulo con el resto: la misma entrevista que produce el
// calendario produce el alcance del programa de auditoria, asi que no puede
// haber una obligacion que el calendario vigile y el programa no cubra.
//
// # El arrastre entre ciclos, con su orden
//
// `auditoria.Abrir` recibe lo que viene del ciclo anterior, y de donde salga eso
// es cosa del adaptador: aqui sale de `--anterior`, el fichero del programa del
// ciclo pasado, por su propio metodo `ParaElCicloSiguiente()`. Sin esa bandera el
// arrastre es el cero, que es lo correcto en un primer ciclo y es DISTINTO de
// «no lo hemos mirado»: por eso hay bandera y no adivinacion.
//
// # Hechos que se anaden y nunca se editan
//
// Igual que en accesos y en incidentes: no hay `modificar` ni `borrar`. Un
// hallazgo se cierra con OTRO hecho, que lleva su propio «como», que es lo que
// un auditor externo va a pedir.

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/auditoria"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// ordenesDeAuditoria son las subordenes. `ver` es la de por defecto porque es la
// unica que no escribe.
var ordenesDeAuditoria = []string{"ver", "abrir", "auditar", "diferir", "anotar", "cerrar"}

const ayudaAuditoria = `plazum auditoria: el programa de auditoria interna del que se compone el acta.

  plazum auditoria ver     --programa programa.json
  plazum auditoria abrir   --programa F --id P-2026 --ciclo 2026-2028 --desde D --hasta D
                           --alcance alcance.json [--corpus paquetes] [--anterior F]
  plazum auditoria auditar --programa F --id S1 --auditor quien --cuando T
                           --unidades "paquete|obligacion,paquete|obligacion" [--que ...]
  plazum auditoria diferir --programa F --unidad paquete|obligacion --quien Q --motivo M --cuando T
  plazum auditoria anotar  --programa F --id H1 --sesion S1 --unidad paquete|obligacion
                           --clase C --texto T --quien Q --cuando T
  plazum auditoria cerrar  --programa F --hallazgo H1 --quien Q --como C --cuando T

  Las fechas de --desde y --hasta se escriben AAAA-MM-DD; los instantes, en
  RFC3339 (2026-09-03T09:00:00Z).

  EL ALCANCE DEL PROGRAMA NO SE TECLEA. Sale de tus respuestas (--alcance) y del
  corpus instalado (--corpus): son exactamente las obligaciones que te alcanzan.
  Una lista escrita a mano se queda corta el primer dia y vieja el segundo.

  NO HAY modificar NI borrar. Un hallazgo se cierra con OTRO hecho, y ese hecho
  lleva su COMO, que es lo que un auditor externo va a pedir: "cerrado" a secas
  no es evidencia de nada.

  Este fichero es el que lee: plazum serve --acta-programa F
`

func cmdAuditoria(args []string, salida, errores io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "abrir":
			return cmdAuditoriaAbrir(args[1:], salida, errores)
		case "auditar":
			return cmdAuditoriaAuditar(args[1:], salida, errores)
		case "diferir":
			return cmdAuditoriaDiferir(args[1:], salida, errores)
		case "anotar":
			return cmdAuditoriaAnotar(args[1:], salida, errores)
		case "cerrar":
			return cmdAuditoriaCerrar(args[1:], salida, errores)
		case "ver":
			args = args[1:]
		}
	}
	return cmdAuditoriaVer(args, salida, errores)
}

// leerPrograma lee el programa, con las tres respuestas del invariante 8: no
// existe, existe y se lee, existe y NO se entiende. La tercera es error, nunca
// un programa vacio: machacar un registro de hechos ilegible pierde lo que
// hubiera dentro y deja un fichero que parece bueno.
func leerPrograma(ruta string) (*auditoria.Programa, error) {
	b, err := os.ReadFile(ruta) // #nosec G304 -- CLI: la ruta la teclea el operador en su propia maquina
	switch {
	case err == nil:
		p, err := auditoria.Reconstruir(b)
		if err != nil {
			return nil, fmt.Errorf("%s existe y no es un programa de auditoria legible: %w.\n"+
				"  NO se ha tocado: sobrescribir un registro de hechos que no se entiende es "+
				"peor que no escribir", ruta, err)
		}
		return p, nil
	case os.IsNotExist(err):
		return nil, nil // programa nuevo
	default:
		return nil, err
	}
}

func escribirPrograma(ruta string, p *auditoria.Programa) error {
	b, err := auditoria.Escribir(p)
	if err != nil {
		return err
	}
	return escribirConFsync(ruta, append(b, '\n'))
}

// programaExistente lee el programa y se niega si no hay ninguno.
//
// Toda suborden que anade un hecho pasa por aqui: un hecho sobre un programa que
// no existe no es un hecho, es una linea perdida.
func programaExistente(ruta string, errores io.Writer) (*auditoria.Programa, int) {
	if strings.TrimSpace(ruta) == "" {
		fmt.Fprintln(errores, "falta --programa (los hechos van ahi o no van a ningun sitio)")
		return nil, 2
	}
	p, err := leerPrograma(strings.TrimSpace(ruta))
	if err != nil {
		fmt.Fprintln(errores, err)
		return nil, 1
	}
	if p == nil {
		fmt.Fprintf(errores, "en %s no hay ningun programa de auditoria.\n", ruta)
		fmt.Fprintln(errores, "  Arreglo: abrirlo primero con `plazum auditoria abrir`, que es lo")
		fmt.Fprintln(errores, "  que fija su ciclo y su alcance. Sin alcance, un programa no puede")
		fmt.Fprintln(errores, "  decir que se ha quedado sin auditar, que es lo unico que hace.")
		return nil, 1
	}
	return p, 0
}

// fechaDelCiclo lee una fecha AAAA-MM-DD obligatoria.
//
// UNA FECHA QUE NO SE ENTIENDE ES UN ERROR, NUNCA EL CERO (invariante 8, tercera
// forma): un time.Time cero daria un ciclo que empieza en el ano 1, y `Cubre`
// aceptaria entonces cualquier sesion de cualquier fecha como «de este ciclo».
func fechaDelCiclo(bandera, v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, fmt.Errorf("falta %s. Un ciclo sin principio o sin fin no puede "+
			"decir que sesion cae dentro de el, y de eso depende toda la cobertura", bandera)
	}
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s no se entiende: %q no es una fecha AAAA-MM-DD",
			bandera, v)
	}
	return t.UTC(), nil
}

// unidadesDelAlcance deriva el alcance del programa del corpus y del alcance.
//
// EMPAREJA POR EL ID DE LA OBLIGACION, que es el mismo identificador con el que
// el motor de aplicabilidad devuelve lo que te alcanza y el mismo que el propio
// paquete declara. No hay indice ni posicion por medio (invariante 7): recorrer
// dos listas en paralelo aqui bailaria el dia que el corpus gane un paquete.
func unidadesDelAlcance(ps []*corpus.Paquete, al alcance) ([]auditoria.Unidad, error) {
	aplica, _, err := aplicableDelAlcance(ps, al, false)
	if err != nil {
		return nil, err
	}
	var out []auditoria.Unidad
	for _, p := range ps {
		for _, ob := range p.Obligaciones {
			if ok, _ := aplica(ob.ID); !ok {
				continue
			}
			out = append(out, auditoria.Unidad{
				Paquete: p.URN, Version: p.Version, Obligacion: ob.ID,
				Titulo: ob.TituloLegible(),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Clave() < out[j].Clave() })
	return out, nil
}

func cmdAuditoriaAbrir(args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("auditoria abrir", flag.ContinueOnError)
	fs.SetOutput(errores)
	fs.Usage = func() { fmt.Fprint(errores, ayudaAuditoria) }
	ruta := fs.String("programa", "", "fichero JSON del programa que se crea")
	id := fs.String("id", "", "identificador del programa")
	ciclo := fs.String("ciclo", "", "como llama la organizacion a este ciclo (2026-2028)")
	desde := fs.String("desde", "", "primer dia del ciclo (AAAA-MM-DD)")
	hasta := fs.String("hasta", "", "ultimo dia del ciclo (AAAA-MM-DD)")
	rutaAlcance := fs.String("alcance", "", "fichero con las respuestas de tu organizacion")
	dirCorpus := fs.String("corpus", "paquetes", "directorio de paquetes de corpus instalados")
	anterior := fs.String("anterior", "",
		"programa del CICLO ANTERIOR, del que se arrastra lo no auditado y lo no cerrado")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	var faltan []string
	if strings.TrimSpace(*ruta) == "" {
		faltan = append(faltan, "--programa")
	}
	if strings.TrimSpace(*id) == "" {
		faltan = append(faltan, "--id")
	}
	if strings.TrimSpace(*rutaAlcance) == "" {
		faltan = append(faltan, "--alcance (el alcance del programa sale de tus respuestas, no "+
			"se teclea)")
	}
	if len(faltan) > 0 {
		fmt.Fprintf(errores, "faltan datos: %s\n", strings.Join(faltan, "; "))
		return 2
	}
	d, err := fechaDelCiclo("--desde", *desde)
	if err != nil {
		fmt.Fprintln(errores, err)
		return 2
	}
	h, err := fechaDelCiclo("--hasta", *hasta)
	if err != nil {
		fmt.Fprintln(errores, err)
		return 2
	}

	// UN PROGRAMA NO SE ABRE ENCIMA DE OTRO. Sobrescribir uno abierto perderia
	// sus sesiones, sus hallazgos y su arrastre, que son hechos.
	if ya, err := leerPrograma(strings.TrimSpace(*ruta)); err != nil {
		fmt.Fprintln(errores, err)
		return 1
	} else if ya != nil {
		fmt.Fprintf(errores, "en %s ya hay un programa abierto (%s, ciclo %q).\n",
			*ruta, ya.ID(), ya.Ciclo().Nombre)
		fmt.Fprintln(errores, "  No se abre encima: perderia sus sesiones, sus hallazgos y su")
		fmt.Fprintln(errores, "  arrastre, que son hechos. Para el ciclo siguiente, abre OTRO")
		fmt.Fprintln(errores, "  fichero con --anterior apuntando a este.")
		return 1
	}

	ps, err := corpus.Cargar(strings.TrimSpace(*dirCorpus))
	if err != nil {
		fmt.Fprintf(errores, "el corpus de %s no carga: %v\n", *dirCorpus, err)
		return 1
	}
	al, err := cargarAlcance(strings.TrimSpace(*rutaAlcance))
	if err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}
	unidades, err := unidadesDelAlcance(ps, al)
	if err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}
	if len(unidades) == 0 {
		// UN PROGRAMA VACIO NO SE ABRE, y el mensaje separa las dos causas: o el
		// alcance no enciende nada, o el corpus instalado no declara reglas que
		// puedan encender nada. Son problemas distintos y el arreglo tambien.
		fmt.Fprintf(errores, "con %s y el corpus de %s no te alcanza ninguna obligacion, asi "+
			"que el programa saldria vacio.\n", *rutaAlcance, *dirCorpus)
		fmt.Fprintln(errores, "  Un programa sin alcance no puede decir que se ha quedado sin")
		fmt.Fprintln(errores, "  auditar, que es lo unico que hace.")
		fmt.Fprintln(errores, "  Comprueba primero que `plazum calendario --alcance ...` te ensena")
		fmt.Fprintln(errores, "  obligaciones: si tampoco, el problema esta en el alcance o en el corpus.")
		return 1
	}

	// EL ARRASTRE, si hay ciclo anterior. Sin bandera es el CERO, que en un
	// primer ciclo es correcto y es distinto de «no lo hemos mirado»: por eso
	// hay bandera y no adivinacion.
	var arr auditoria.Arrastre
	if strings.TrimSpace(*anterior) != "" {
		prev, err := leerPrograma(strings.TrimSpace(*anterior))
		if err != nil {
			fmt.Fprintln(errores, err)
			return 1
		}
		if prev == nil {
			fmt.Fprintf(errores, "--anterior apunta a %s y ahi no hay ningun programa.\n"+
				"  Si este es el primer ciclo, no pases --anterior: el arrastre vacio de un\n"+
				"  primer ciclo es correcto y es distinto de un arrastre que no se ha mirado.\n",
				*anterior)
			return 1
		}
		arr = prev.ParaElCicloSiguiente()
	}

	p, err := auditoria.Abrir(strings.TrimSpace(*id),
		auditoria.Ciclo{Nombre: strings.TrimSpace(*ciclo), Desde: d, Hasta: h}, unidades, arr)
	if err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}
	if err := escribirPrograma(strings.TrimSpace(*ruta), p); err != nil {
		fmt.Fprintln(errores, "el programa NO se ha escrito:", err)
		return 1
	}
	fmt.Fprintf(salida, "Programa %s abierto en %s.\n", p.ID(), *ruta)
	fmt.Fprintf(salida, "  ciclo    %s, del %s al %s\n", p.Ciclo().Nombre,
		d.Format("2006-01-02"), h.Format("2006-01-02"))
	fmt.Fprintf(salida, "  alcance  %d unidades, derivadas de %s sobre el corpus de %s\n",
		p.Alcance(), *rutaAlcance, *dirCorpus)
	imprimirArrastre(salida, p)
	imprimirComoSeMonta(salida, "--acta-programa", *ruta)
	return 0
}

// imprimirArrastre cuenta lo que viene del ciclo anterior, incluidas LAS
// SALIDAS: una unidad que venia arrastrada y ya no esta en el alcance no se
// arrastra (seguiria echandose de menos para siempre) y NO se calla, porque
// entonces una obligacion desapareceria del programa sin que constara.
func imprimirArrastre(w io.Writer, p *auditoria.Programa) {
	a := p.DelCicloAnterior()
	if a.DeCiclo == "" && len(a.SinAuditar) == 0 && len(a.Abiertos) == 0 && len(a.Salidas) == 0 {
		fmt.Fprintln(w, "  arrastre no hay ciclo anterior. En un primer ciclo eso es correcto,")
		fmt.Fprintln(w, "           y es distinto de un arrastre que nadie ha mirado.")
		return
	}
	fmt.Fprintf(w, "  arrastre del ciclo %s: %d unidades sin auditar, %d hallazgos abiertos\n",
		a.DeCiclo, len(a.SinAuditar), len(a.Abiertos))
	if len(a.Salidas) > 0 {
		fmt.Fprintf(w, "           %d unidades venian arrastradas y YA NO ESTAN en el alcance "+
			"de este ciclo:\n", len(a.Salidas))
		for _, s := range a.Salidas {
			fmt.Fprintf(w, "             %s\n", s)
		}
		fmt.Fprintln(w, "           No se arrastran (se echarian de menos para siempre) y no se")
		fmt.Fprintln(w, "           callan: que una unidad salga del alcance es una decision.")
	}
}

func cmdAuditoriaAuditar(args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("auditoria auditar", flag.ContinueOnError)
	fs.SetOutput(errores)
	fs.Usage = func() { fmt.Fprint(errores, ayudaAuditoria) }
	ruta := fs.String("programa", "", "fichero JSON del programa")
	id := fs.String("id", "", "identificador de la sesion")
	auditor := fs.String("auditor", "", "identificador estable del auditor, no su nombre")
	cuando := fs.String("cuando", "", "instante RFC3339 de la sesion")
	unidades := fs.String("unidades", "",
		"claves paquete|obligacion separadas por coma; tienen que estar en el alcance")
	que := fs.String("que", "", "que se miro, escrito por una persona")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	p, codigo := programaExistente(*ruta, errores)
	if p == nil {
		return codigo
	}
	t, err := instanteObligatorio("--cuando", *cuando)
	if err != nil {
		fmt.Fprintln(errores, err)
		return 2
	}
	us := separarPorComa(*unidades)
	if len(us) == 0 {
		fmt.Fprintln(errores, "falta --unidades: una sesion que no mira nada no es una sesion.")
		fmt.Fprintln(errores, "  Las claves son paquete|obligacion y salen de `plazum auditoria ver`.")
		return 2
	}
	if err := p.Auditar(auditoria.Sesion{
		ID: strings.TrimSpace(*id), Auditor: strings.TrimSpace(*auditor), Cuando: t,
		Unidades: us, Alcance: *que,
	}); err != nil {
		// El error del nucleo trae el arreglo dentro (dice que unidades no
		// estaban en el alcance, o que el ciclo no cubre esa fecha).
		fmt.Fprintln(errores, err)
		return 1
	}
	if err := escribirPrograma(strings.TrimSpace(*ruta), p); err != nil {
		fmt.Fprintln(errores, "la sesion NO se ha registrado:", err)
		return 1
	}
	fmt.Fprintf(salida, "Sesion %s registrada: %d unidades auditadas el %s.\n",
		*id, len(us), t.Format("2006-01-02"))
	imprimirCobertura(salida, p)
	return 0
}

func cmdAuditoriaDiferir(args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("auditoria diferir", flag.ContinueOnError)
	fs.SetOutput(errores)
	fs.Usage = func() { fmt.Fprint(errores, ayudaAuditoria) }
	ruta := fs.String("programa", "", "fichero JSON del programa")
	unidad := fs.String("unidad", "", "clave paquete|obligacion que se deja para el ciclo siguiente")
	quien := fs.String("quien", "", "quien lo decide")
	motivo := fs.String("motivo", "", "por que se deja fuera de la cobertura de este ciclo")
	cuando := fs.String("cuando", "", "instante RFC3339 de la decision")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	p, codigo := programaExistente(*ruta, errores)
	if p == nil {
		return codigo
	}
	t, err := instanteObligatorio("--cuando", *cuando)
	if err != nil {
		fmt.Fprintln(errores, err)
		return 2
	}
	if err := p.Diferir(auditoria.Diferimiento{
		Unidad: strings.TrimSpace(*unidad), Quien: strings.TrimSpace(*quien),
		Motivo: strings.TrimSpace(*motivo), Cuando: t,
	}); err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}
	if err := escribirPrograma(strings.TrimSpace(*ruta), p); err != nil {
		fmt.Fprintln(errores, "el diferimiento NO se ha registrado:", err)
		return 1
	}
	fmt.Fprintf(salida, "%s diferida por %s: %s\n", *unidad, *quien, *motivo)
	fmt.Fprintln(salida, "Queda escrito y sale en el informe de cobertura: un diferimiento que")
	fmt.Fprintln(salida, "no se ve es una unidad que desaparece sin que nadie responda de ella.")
	imprimirCobertura(salida, p)
	return 0
}

func cmdAuditoriaAnotar(args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("auditoria anotar", flag.ContinueOnError)
	fs.SetOutput(errores)
	fs.Usage = func() { fmt.Fprint(errores, ayudaAuditoria) }
	ruta := fs.String("programa", "", "fichero JSON del programa")
	id := fs.String("id", "", "identificador del hallazgo")
	sesion := fs.String("sesion", "", "id de la sesion de la que sale")
	unidad := fs.String("unidad", "", "clave paquete|obligacion sobre la que se anota")
	clase := fs.String("clase", "", "clase del hallazgo")
	texto := fs.String("texto", "", "que se encontro, escrito por el auditor")
	quien := fs.String("quien", "", "quien lo anota")
	cuando := fs.String("cuando", "", "instante RFC3339")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	p, codigo := programaExistente(*ruta, errores)
	if p == nil {
		return codigo
	}
	t, err := instanteObligatorio("--cuando", *cuando)
	if err != nil {
		fmt.Fprintln(errores, err)
		return 2
	}
	// LA CLASE SE TRADUCE CON EL VOCABULARIO DEL NUCLEO Y PUEDE FALLAR. Una
	// clase que no se reconoce NO cae al cero: el cero es «no conformidad
	// mayor», la mas grave, y caer ahi acusa de mas igual que caer a la ultima
	// esconderia un incumplimiento.
	c, err := auditoria.ClaseDe(strings.TrimSpace(*clase))
	if err != nil {
		fmt.Fprintln(errores, err)
		return 2
	}
	if err := p.Anotar(auditoria.Hallazgo{
		ID: strings.TrimSpace(*id), Sesion: strings.TrimSpace(*sesion),
		Unidad: strings.TrimSpace(*unidad), Clase: c, Texto: *texto,
		Quien: strings.TrimSpace(*quien), Cuando: t,
	}); err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}
	if err := escribirPrograma(strings.TrimSpace(*ruta), p); err != nil {
		fmt.Fprintln(errores, "el hallazgo NO se ha registrado:", err)
		return 1
	}
	fmt.Fprintf(salida, "Hallazgo %s anotado sobre %s: %s.\n", *id, *unidad, c)
	if c.Exige() {
		fmt.Fprintln(salida, "  Esta clase EXIGE plan de accion: cuenta como abierta hasta que se")
		fmt.Fprintln(salida, "  cierre con `plazum auditoria cerrar`, y el cierre lleva su COMO.")
	}
	fmt.Fprintf(salida, "  hallazgos abiertos ahora: %d\n", len(p.Abiertos()))
	return 0
}

func cmdAuditoriaCerrar(args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("auditoria cerrar", flag.ContinueOnError)
	fs.SetOutput(errores)
	fs.Usage = func() { fmt.Fprint(errores, ayudaAuditoria) }
	ruta := fs.String("programa", "", "fichero JSON del programa")
	hallazgo := fs.String("hallazgo", "", "id del hallazgo que se cierra")
	quien := fs.String("quien", "", "quien lo cierra")
	como := fs.String("como", "", "que se hizo para cerrarlo: es lo que pedira un auditor externo")
	cuando := fs.String("cuando", "", "instante RFC3339")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	p, codigo := programaExistente(*ruta, errores)
	if p == nil {
		return codigo
	}
	t, err := instanteObligatorio("--cuando", *cuando)
	if err != nil {
		fmt.Fprintln(errores, err)
		return 2
	}
	if err := p.Cerrar(auditoria.CierreDeHallazgo{
		Hallazgo: strings.TrimSpace(*hallazgo), Quien: strings.TrimSpace(*quien),
		Cuando: t, Como: *como,
	}); err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}
	if err := escribirPrograma(strings.TrimSpace(*ruta), p); err != nil {
		fmt.Fprintln(errores, "el cierre NO se ha registrado:", err)
		return 1
	}
	fmt.Fprintf(salida, "Hallazgo %s cerrado por %s el %s.\n", *hallazgo, *quien,
		t.Format("2006-01-02"))
	fmt.Fprintf(salida, "  hallazgos abiertos ahora: %d\n", len(p.Abiertos()))
	return 0
}

func cmdAuditoriaVer(args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("auditoria ver", flag.ContinueOnError)
	fs.SetOutput(errores)
	fs.Usage = func() { fmt.Fprint(errores, ayudaAuditoria) }
	ruta := fs.String("programa", "", "fichero JSON del programa")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if strings.TrimSpace(*ruta) == "" {
		fmt.Fprint(errores, ayudaAuditoria)
		fmt.Fprintf(errores, "\n  ordenes: %s (por defecto, ver)\n",
			strings.Join(ordenesDeAuditoria, ", "))
		return 2
	}
	p, err := leerPrograma(strings.TrimSpace(*ruta))
	if err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}
	if p == nil {
		fmt.Fprintf(salida, "En %s no hay ningun programa de auditoria todavia.\n", *ruta)
		fmt.Fprintln(salida, "Esto NO dice que no se haya auditado nada: dice que en este fichero")
		fmt.Fprintln(salida, "no consta ningun programa. Se abre con `plazum auditoria abrir`.")
		return 0
	}
	fmt.Fprintf(salida, "PROGRAMA %s, ciclo %s (%s a %s)\n\n", p.ID(), p.Ciclo().Nombre,
		p.Ciclo().Desde.Format("2006-01-02"), p.Ciclo().Hasta.Format("2006-01-02"))
	imprimirCobertura(salida, p)
	if hs := p.Abiertos(); len(hs) > 0 {
		fmt.Fprintf(salida, "\nHALLAZGOS ABIERTOS (%d)\n", len(hs))
		for _, h := range hs {
			fmt.Fprintf(salida, "  %-10s %-22s %s\n", h.ID, h.Clase, h.Unidad)
			fmt.Fprintf(salida, "             %s\n", h.Texto)
		}
	}
	fmt.Fprintf(salida, "\nEL ALCANCE, CLAVE A CLAVE (%d)\n", p.Alcance())
	for _, u := range p.Unidades() {
		fmt.Fprintf(salida, "  %-14s %s  %s\n", p.CoberturaDe(u.Clave()), u.Clave(), u.Titulo)
	}
	imprimirComoSeMonta(salida, "--acta-programa", *ruta)
	return 0
}

// imprimirCobertura ensena la ley de conservacion del programa.
//
// LAS TRES COBERTURAS SE RECORREN ENTERAS Y EN SU ORDEN, no las claves del mapa:
// un mapa se recorre distinto en cada ejecucion, y un informe que baila no se
// puede comparar entre dos. Y el descuadre se ENSENA: si los cubos no cubren el
// alcance, hay unidades que no estan en ningun sitio.
func imprimirCobertura(w io.Writer, p *auditoria.Programa) {
	c := p.Cuenta()
	fmt.Fprintf(w, "COBERTURA DEL CICLO (%d unidades en el alcance)\n", p.Alcance())
	for _, cob := range auditoria.CoberturasPosibles() {
		fmt.Fprintf(w, "  %-14s %d\n", cob, c[cob])
	}
	if !p.Cuadra() {
		fmt.Fprintln(w, "  AVISO: los cubos NO cubren el alcance entero. Este recuento no vale, "+
			"y es un fallo del producto, no tuyo.")
	}
	// LA FRASE VA CON EL DATO. Lo no auditado NO es lo incumplido, y en un
	// informe de auditoria confundirlos es acusar en falso.
	if c[auditoria.SinAuditar] > 0 {
		fmt.Fprintf(w, "  %s\n", auditoria.LaFraseDeLoNoAuditado)
	}
}
