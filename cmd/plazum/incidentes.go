package main

// `plazum incidentes`: LA ORDEN QUE ESCRIBE EL REGISTRO DE INCIDENTES.
//
// # El hueco que cierra, y por que era el que mas molestaba
//
// `nucleo/incidente` sabia LEER un registro de disco desde el 02-09-2026
// (Reconstruir, con su version exigida) y `plazum serve --acta-incidentes` lo
// montaba en la pantalla del acta. Lo que NO existia era nada que lo ESCRIBIERA.
// O sea que una instalacion nueva no podia llegar a tener acta ni queriendo: la
// unica forma de conseguir uno de sus ficheros era abrir un editor de texto y
// componer el JSON a mano, adivinando el nombre de los campos y el valor del
// campo `version`.
//
// Un producto cuyo mejor documento se compone de tres fuentes y no sabe crear
// dos de ellas no tiene ese documento: tiene su plantilla.
//
// # El patron es el de `plazum accesos`, y no por parecido
//
// Subordenes; `ver` por defecto porque es la unica que no cambia nada; `--ahora`
// para que el instante entre como dato y la suite no dependa del reloj; y
// HECHOS QUE SE ANADEN Y NUNCA SE EDITAN. No hay `modificar` ni `borrar`, y esa
// ausencia es el diseno: una correccion es un suceso mas con su propio instante
// de registro, igual que en el ledger de la campana de accesos.
//
// # Los dos ejes, y por que se piden los dos
//
// Todo suceso de un incidente lleva CUANDO PASO EN EL MUNDO y CUANDO SE SUPO, y
// los dos son obligatorios en el nucleo. Aqui tambien, y no se rellena el
// segundo con el primero ni con el reloj en silencio: hay plazos que cuentan del
// segundo (art. 33 del RGPD cuenta desde que el responsable tiene constancia) y
// un valor por defecto ahi mueve un plazo legal sin que nadie lo haya decidido.
// Es la regla de la casa: un campo obligatorio y uno opcional son dos preguntas
// distintas y se leen con dos funciones distintas.
//
// # Un fichero que no se entiende NO se machaca
//
// Igual que `leerLedger`: si el registro existe y no se puede leer, la orden se
// para y no escribe. Sobrescribir un documento append-only ilegible pierde lo
// que hubiera dentro y ademas deja un fichero que parece bueno.

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/incidente"
)

// ordenesDeIncidentes son las subordenes. `ver` es la de por defecto porque es
// la unica que no escribe: teclear `plazum incidentes` a secas no puede anadir
// un hecho a un registro del que se compone un acta.
var ordenesDeIncidentes = []string{"ver", "abrir", "clasificar", "notificar", "cerrar"}

const ayudaIncidentes = `plazum incidentes: el registro de incidentes del que se compone el acta.

  plazum incidentes ver        --registro incidentes.json
  plazum incidentes abrir      --registro F --id INC-1 --ocurrio T --se-supo T --fuente quien
  plazum incidentes clasificar --registro F --id INC-1 --clase C --cuando T
  plazum incidentes notificar  --registro F --id INC-1 --hito H --cuando T
  plazum incidentes cerrar     --registro F --id INC-1 --cuando T

  Los instantes se escriben en RFC3339 (2026-09-03T09:00:00Z).

  LOS DOS INSTANTES DE UN SUCESO SON DISTINTOS Y SE PIDEN LOS DOS. --ocurrio y
  --cuando dicen cuando paso en el mundo; --ahora dice cuando se registra. Hay
  plazos que cuentan de lo segundo (art. 33 del Reglamento (UE) 2016/679 cuenta
  desde que el responsable tiene constancia), asi que rellenar uno con el otro
  moveria un plazo legal sin que nadie lo hubiera decidido.

  NO HAY modificar NI borrar, y no es un olvido: un incidente es un registro de
  hechos. Una correccion es un suceso mas, con su propio instante de registro.

  Este fichero es el que lee: plazum serve --acta-incidentes F
`

func cmdIncidentes(args []string, salida, errores io.Writer) int {
	if len(args) > 0 {
		switch args[0] {
		case "abrir":
			return cmdIncidenteAbrir(args[1:], salida, errores)
		case "clasificar":
			return cmdIncidenteSuceso(incidente.Clasificacion, args[1:], salida, errores)
		case "notificar":
			return cmdIncidenteSuceso(incidente.Notificacion, args[1:], salida, errores)
		case "cerrar":
			return cmdIncidenteSuceso(incidente.Cierre, args[1:], salida, errores)
		case "ver":
			args = args[1:]
		}
	}
	return cmdIncidentesVer(args, salida, errores)
}

// leerRegistroDeIncidentes lee el registro, y NO se inventa uno vacio si el
// fichero existe y no se entiende.
//
// LAS TRES RESPUESTAS, y son tres y no dos (invariante 8): no existe todavia
// (registro nuevo, normal la primera vez), existe y se lee, y existe y NO se
// entiende. La tercera es un dato presente y no interpretable, y devolverla como
// «registro vacio» machacaria lo que hubiera dentro.
func leerRegistroDeIncidentes(ruta string) ([]*incidente.Incidente, error) {
	b, err := os.ReadFile(ruta) // #nosec G304 -- CLI: la ruta la teclea el operador en su propia maquina
	switch {
	case err == nil:
		is, err := incidente.Reconstruir(b)
		if err != nil {
			return nil, fmt.Errorf("%s existe y no es un registro de incidentes legible: %w.\n"+
				"  NO se ha tocado: sobrescribir un registro de hechos que no se entiende es "+
				"peor que no escribir", ruta, err)
		}
		return is, nil
	case os.IsNotExist(err):
		return nil, nil // registro nuevo: es normal la primera vez
	default:
		return nil, err
	}
}

func escribirRegistroDeIncidentes(ruta string, is []*incidente.Incidente) error {
	b, err := incidente.Escribir(is)
	if err != nil {
		return err
	}
	return escribirConFsync(ruta, append(b, '\n'))
}

// instanteObligatorio lee un instante que TIENE que venir.
//
// Y ES UNA FUNCION DISTINTA DE instanteOAhora a proposito, que es la regla de la
// casa: un campo obligatorio y uno opcional son dos preguntas distintas, y
// meterlas en una con valor por defecto es por donde se cuela el cero. Un
// instante en cero aqui es el 1 de enero del ano 1 con cara de dato, y de ahi
// salen plazos vencidos hace dos mil anos.
func instanteObligatorio(bandera, v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, fmt.Errorf("falta %s, y no tiene valor por defecto: es un instante "+
			"del que cuelgan plazos legales.\n"+
			"  Se escribe en RFC3339, por ejemplo 2026-09-03T09:00:00Z", bandera)
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s no se entiende: %q no es un instante RFC3339 "+
			"(2026-09-03T09:00:00Z)", bandera, v)
	}
	return t.UTC(), nil
}

// instanteOAhora lee un instante OPCIONAL que por defecto es el reloj.
//
// Vale solo para el eje de REGISTRO (cuando se anota), nunca para el eje del
// mundo: registrar ahora lo que se anota ahora es cierto por construccion, y
// suponer que algo PASO ahora no lo es.
func instanteOAhora(bandera, v string) (time.Time, error) {
	if strings.TrimSpace(v) == "" {
		// El unico time.Now() de esta orden, y esta en el borde (invariante 1).
		return time.Now().UTC(), nil
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(v))
	if err != nil {
		return time.Time{}, fmt.Errorf("%s no se entiende: %q no es un instante RFC3339 "+
			"(2026-09-03T09:00:00Z)", bandera, v)
	}
	return t.UTC(), nil
}

// banderasDeIncidente son las que comparten todas las subordenes que escriben.
type banderasDeIncidente struct {
	registro *string
	id       *string
	ahora    *string
}

func declararBanderasDeIncidente(fs *flag.FlagSet) *banderasDeIncidente {
	return &banderasDeIncidente{
		registro: fs.String("registro", "", "fichero JSON del registro de incidentes"),
		id:       fs.String("id", "", "identificador estable del incidente (tu numero de expediente)"),
		ahora: fs.String("ahora", "",
			"instante RFC3339 en que se REGISTRA; por defecto el reloj de esta maquina"),
	}
}

func (b *banderasDeIncidente) validar(errores io.Writer) bool {
	var faltan []string
	if strings.TrimSpace(*b.registro) == "" {
		faltan = append(faltan, "--registro (los hechos van ahi o no van a ningun sitio)")
	}
	if strings.TrimSpace(*b.id) == "" {
		faltan = append(faltan, "--id (la identidad es lo que separa un incidente de otro y lo "+
			"que hace que sus plazos no se pisen)")
	}
	if len(faltan) > 0 {
		fmt.Fprintf(errores, "faltan datos: %s\n", strings.Join(faltan, "; "))
		return false
	}
	return true
}

func cmdIncidenteAbrir(args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("incidentes abrir", flag.ContinueOnError)
	fs.SetOutput(errores)
	fs.Usage = func() { fmt.Fprint(errores, ayudaIncidentes) }
	b := declararBanderasDeIncidente(fs)
	ocurrio := fs.String("ocurrio", "", "instante RFC3339 en que ocurrio EN EL MUNDO")
	seSupo := fs.String("se-supo", "",
		"instante RFC3339 en que la organizacion tuvo constancia; de aqui cuenta el art. 33")
	fuente := fs.String("fuente", "", "quien lo registra")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if !b.validar(errores) {
		return 2
	}
	// LOS DOS EJES SE PIDEN LOS DOS Y NINGUNO RELLENA AL OTRO. Ver el encabezado.
	cuandoPaso, err := instanteObligatorio("--ocurrio", *ocurrio)
	if err != nil {
		fmt.Fprintln(errores, err)
		return 2
	}
	cuandoSeSupo, err := instanteObligatorio("--se-supo", *seSupo)
	if err != nil {
		fmt.Fprintln(errores, err)
		fmt.Fprintln(errores, "  No se rellena con --ocurrio: enterarse y que pase no son lo")
		fmt.Fprintln(errores, "  mismo, y hay plazos de notificacion que cuentan de lo segundo.")
		return 2
	}

	is, err := leerRegistroDeIncidentes(strings.TrimSpace(*b.registro))
	if err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}
	// UN SEGUNDO INCIDENTE CON EL MISMO ID SE RECHAZA AQUI, con un error que
	// dice que hacer. El escritor tambien lo rechazaria (Reconstruir se niega a
	// leer dos con el mismo id), pero mucho mas tarde y diciendo otra cosa.
	for _, ya := range is {
		if ya.ID() == strings.TrimSpace(*b.id) {
			fmt.Fprintf(errores, "en %s ya consta el incidente %q.\n", *b.registro, *b.id)
			fmt.Fprintln(errores, "  No se abre otra vez: un incidente nace una sola vez. Si lo")
			fmt.Fprintln(errores, "  que quieres es anadirle algo, usa clasificar, notificar o cerrar.")
			return 1
		}
	}

	i, err := incidente.Abrir(strings.TrimSpace(*b.id), cuandoPaso, cuandoSeSupo,
		strings.TrimSpace(*fuente))
	if err != nil {
		// El error del nucleo ya es accionable y trae el arreglo dentro.
		fmt.Fprintln(errores, err)
		return 1
	}
	is = append(is, i)
	if err := escribirRegistroDeIncidentes(strings.TrimSpace(*b.registro), is); err != nil {
		fmt.Fprintln(errores, "el incidente NO se ha registrado:", err)
		return 1
	}
	fmt.Fprintf(salida, "Incidente %s abierto en %s.\n", i.ID(), *b.registro)
	fmt.Fprintf(salida, "  ocurrio  %s\n", cuandoPaso.Format(time.RFC3339))
	fmt.Fprintf(salida, "  se supo  %s\n", cuandoSeSupo.Format(time.RFC3339))
	imprimirComoSeMonta(salida, "--acta-incidentes", *b.registro)
	return 0
}

// cmdIncidenteSuceso anade un suceso a un incidente que ya existe.
//
// UNA SOLA FUNCION PARA LOS TRES TIPOS, y no tres copias, porque lo unico que
// cambia entre clasificar, notificar y cerrar es que campo trae cada uno. El
// nucleo ya comprueba que cada tipo traiga LO SUYO y no lo ajeno (clase solo en
// clasificacion, hito solo en notificacion), asi que repetir esas reglas aqui
// seria una segunda copia que se separa de la primera.
func cmdIncidenteSuceso(tipo incidente.Tipo, args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("incidentes "+tipo.String(), flag.ContinueOnError)
	fs.SetOutput(errores)
	fs.Usage = func() { fmt.Fprint(errores, ayudaIncidentes) }
	b := declararBanderasDeIncidente(fs)
	cuando := fs.String("cuando", "", "instante RFC3339 en que ocurrio EN EL MUNDO")
	clase := fs.String("clase", "",
		"clasificacion, con el nombre que espera el paquete de la norma (solo en clasificar)")
	hito := fs.String("hito", "", "id del hito que se remitio (solo en notificar)")
	fuente := fs.String("fuente", "", "quien lo registra")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if !b.validar(errores) {
		return 2
	}
	cuandoPaso, err := instanteObligatorio("--cuando", *cuando)
	if err != nil {
		fmt.Fprintln(errores, err)
		return 2
	}
	cuandoSeRegistra, err := instanteOAhora("--ahora", *b.ahora)
	if err != nil {
		fmt.Fprintln(errores, err)
		return 2
	}

	ruta := strings.TrimSpace(*b.registro)
	is, err := leerRegistroDeIncidentes(ruta)
	if err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}
	elID := strings.TrimSpace(*b.id)
	var elIncidente *incidente.Incidente
	for _, i := range is {
		if i.ID() == elID {
			elIncidente = i
			break
		}
	}
	if elIncidente == nil {
		fmt.Fprintf(errores, "en %s no consta ningun incidente con id %q.\n", ruta, elID)
		fmt.Fprintln(errores, "  Arreglo: abrirlo primero con `plazum incidentes abrir`, que es lo")
		fmt.Fprintln(errores, "  que le da su primer conocimiento. Sin apertura no hay de donde")
		fmt.Fprintln(errores, "  contar ningun plazo.")
		if len(is) > 0 {
			fmt.Fprintf(errores, "  Los que hay: %s\n", strings.Join(idsDeIncidentes(is), ", "))
		}
		return 1
	}

	// SE REGISTRA EN EL OBJETO ANTES DE ESCRIBIR EN DISCO. Al reves, un suceso
	// mal formado quedaria anotado para siempre en un registro de hechos y
	// habria que convivir con el. Misma disciplina que `plazum accesos decidir`.
	if err := elIncidente.Registrar(incidente.Suceso{
		Tipo: tipo, Clase: strings.TrimSpace(*clase), Hito: strings.TrimSpace(*hito),
		InstanteHecho: cuandoPaso, InstanteRegistro: cuandoSeRegistra,
		Fuente: strings.TrimSpace(*fuente),
	}); err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}
	if err := escribirRegistroDeIncidentes(ruta, is); err != nil {
		fmt.Fprintln(errores, "el suceso NO se ha registrado:", err)
		return 1
	}
	fmt.Fprintf(salida, "%s: %s el %s (registrado el %s).\n", elID, tipo,
		cuandoPaso.Format(time.RFC3339), cuandoSeRegistra.Format(time.RFC3339))
	imprimirComoSeMonta(salida, "--acta-incidentes", ruta)
	return 0
}

func cmdIncidentesVer(args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("incidentes ver", flag.ContinueOnError)
	fs.SetOutput(errores)
	fs.Usage = func() { fmt.Fprint(errores, ayudaIncidentes) }
	registro := fs.String("registro", "", "fichero JSON del registro de incidentes")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if strings.TrimSpace(*registro) == "" {
		fmt.Fprint(errores, ayudaIncidentes)
		fmt.Fprintf(errores, "\n  ordenes: %s (por defecto, ver)\n",
			strings.Join(ordenesDeIncidentes, ", "))
		return 2
	}
	is, err := leerRegistroDeIncidentes(strings.TrimSpace(*registro))
	if err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}
	if len(is) == 0 {
		// «Todavia no hay ninguno» NO es «no hubo incidentes». La segunda es una
		// afirmacion que plazum no puede hacer y que en un acta se lee al reves.
		fmt.Fprintf(salida, "En %s no consta ningun incidente todavia.\n", *registro)
		fmt.Fprintln(salida, "Esto NO dice que no haya habido incidentes: dice que en este")
		fmt.Fprintln(salida, "registro no consta ninguno. Se anaden con `plazum incidentes abrir`.")
		return 0
	}
	fmt.Fprintf(salida, "REGISTRO DE INCIDENTES DE %s (%d)\n\n", *registro, len(is))
	for _, i := range is {
		fmt.Fprintf(salida, "  %s\n", i.ID())
		if t, ok := i.Ocurrio(); ok {
			fmt.Fprintf(salida, "    ocurrio             %s\n", t.Format(time.RFC3339))
		}
		if t, ok := i.PrimerConocimiento(); ok {
			fmt.Fprintf(salida, "    primer conocimiento %s\n", t.Format(time.RFC3339))
		}
		for _, s := range i.Sucesos() {
			extra := ""
			switch {
			case s.Clase != "":
				extra = "  clase " + s.Clase
			case s.Hito != "":
				extra = "  hito " + s.Hito
			}
			fmt.Fprintf(salida, "    %-14s %s%s\n", s.Tipo,
				s.InstanteHecho.Format(time.RFC3339), extra)
		}
		fmt.Fprintln(salida)
	}
	imprimirComoSeMonta(salida, "--acta-incidentes", *registro)
	return 0
}

func idsDeIncidentes(is []*incidente.Incidente) []string {
	out := make([]string, 0, len(is))
	for _, i := range is {
		out = append(out, i.ID())
	}
	return out
}

// imprimirComoSeMonta cierra toda suborden que toca uno de los ficheros del
// acta diciendo QUE SE HACE CON EL.
//
// POR QUE ESTA AQUI Y NO EN LA AYUDA. El fallo que estas ordenes vienen a
// cerrar no era que faltara un escritor: era que la cadena entera (escribir el
// fichero, montarlo, verlo en el acta) no constaba en ningun sitio. Un fichero
// escrito que nadie sabe donde enchufar es la mitad del hueco de antes.
func imprimirComoSeMonta(w io.Writer, bandera, ruta string) {
	fmt.Fprintf(w, "\nEsto entra en el acta con:\n  plazum serve %s %s\n", bandera, ruta)
}
