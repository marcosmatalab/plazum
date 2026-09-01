package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/accesos"
	"github.com/marcosmatalab/plazum/nucleo/censo"
	"github.com/marcosmatalab/plazum/nucleo/ledger"
)

// LAS ORDENES QUE HACEN LA CAMPANA UTILIZABLE ENTRE SESIONES.
//
// `plazum accesos ver` sube el fichero y ensena donde esta la campana;
// `decidir`, `excusar` y `cerrar` anaden hechos. Todos escriben en el ledger y
// ninguno guarda el censo.
//
// POR QUE TODAS PIDEN EL FICHERO, que es la pregunta que se hara quien las use.
// Las filas del censo -- identificador de cuenta, permiso y rotulo de una
// persona -- no se guardan en ningun sitio. La campana se reconstruye desde el
// FICHERO, que lo tiene el operador, y el LEDGER, que tiene los hechos. Es
// incomodo y se acepta: lo que se compra es que plazum no tenga la plantilla de
// nadie en disco. Y no es una promesa, es comprobable: hay un test que recorre
// el ledger escrito y exige que no aparezca ni un nombre, ni un identificador de
// cuenta, ni un permiso.

// ordenesDeAccesos son las subordenes. `ver` es la de por defecto porque es la
// unica que no cambia nada: teclear `plazum accesos` sin mas no puede escribir
// un hecho.
var ordenesDeAccesos = []string{"ver", "decidir", "excusar", "cerrar"}

// contexto es lo que toda suborden necesita: el censo leido y el ledger.
type contexto struct {
	ins      censo.Instantanea
	l        ledger.Ledger
	id       string
	instante time.Time
	ruta     string
}

// EL SELLO SE RECONSTRUYE CON LOS DATOS DE LA APERTURA, NO CON LOS DE AHORA.
//
// SE ENCONTRO EJECUTANDOLO, no leyendo el diff, y el fallo era mio: el sello
// cubre el fichero MAS quien lo subio, cuando, de que sistema y con que
// retencion. La primera version reconstruia con las banderas de la orden actual,
// asi que `--quien ciso` al decidir daba un sello distinto del que dejo
// `--quien u-042` al subir, y la campana no se reabria NUNCA. Salio a la primera
// ejecucion de punta a punta y no lo habria visto ningun test de los que habia,
// porque todos construian el censo una sola vez.
//
// Lo correcto es evidente dicho asi: la identidad de la instantanea se fija al
// abrir. `--quien` en `decidir` es quien DECIDE hoy, que es otra persona y otro
// dato; el resto sale del ledger, que es quien lo sabe.
func reconstruirConLaApertura(ctx *contexto, fichero string, errores io.Writer) (*accesos.Campana, int) {
	ap, subioLo, cuando, ok := datosDeApertura(ctx.l, ctx.id)
	if !ok {
		fmt.Fprintf(errores, "en %s no consta la apertura de la campana %q.\n", ctx.ruta, ctx.id)
		fmt.Fprintln(errores, "  Arreglo: abrirla primero con `plazum accesos ver ... --ledger ...`,")
		fmt.Fprintln(errores, "  que es lo que anota la subida.")
		return nil, 1
	}
	datos, err := os.ReadFile(fichero) // #nosec G304 -- CLI: la ruta la teclea el operador en su propia maquina
	if err != nil {
		fmt.Fprintf(errores, "no se puede leer %s: %v\n", fichero, err)
		return nil, 1
	}
	ins, err := censo.Tomar(datos, censo.Opciones{
		Sistema: ap.Sistema, Fuente: ap.Fuente, Quien: subioLo,
		Tomada: cuando, Retencion: ap.Retencion, Columnas: censo.ColumnasHabituales(),
	})
	if err != nil {
		fmt.Fprintln(errores, err)
		return nil, 1
	}
	camp, err := accesos.Reconstruir(ctx.id, ins, ctx.l, nil)
	if err != nil {
		fmt.Fprintln(errores, err)
		return nil, 1
	}
	ctx.ins = ins
	return camp, 0
}

// datosDeApertura devuelve lo que el ledger sabe de la subida.
func datosDeApertura(l ledger.Ledger, campana string) (accesos.CargaDeApertura, string, time.Time, bool) {
	sujeto := accesos.Sujeto(campana)
	for _, e := range l.Entradas {
		if e.Sujeto != sujeto || e.Tipo != accesos.TipoApertura {
			continue
		}
		var a accesos.CargaDeApertura
		if err := json.Unmarshal(e.Carga, &a); err != nil {
			return accesos.CargaDeApertura{}, "", time.Time{}, false
		}
		return a, e.Actor, e.Instante, true
	}
	return accesos.CargaDeApertura{}, "", time.Time{}, false
}

// sobreCampana es lo que necesitan las ordenes que anaden hechos: el fichero, el
// ledger, la campana y quien actua AHORA.
type sobreCampana struct {
	fichero  *string
	campana  *string
	quien    *string
	ahora    *string
	registro *string
}

func banderasSobreCampana(fs *flag.FlagSet) *sobreCampana {
	return &sobreCampana{
		fichero:  fs.String("fichero", "", "el MISMO CSV sobre el que se abrio la campana"),
		campana:  fs.String("campana", "", "identificador de la campana"),
		quien:    fs.String("quien", "", "identificador estable de quien decide AHORA"),
		ahora:    fs.String("ahora", "", "instante RFC3339; por defecto el reloj de esta maquina"),
		registro: fs.String("ledger", "", "fichero JSON del ledger con los hechos de la campana"),
	}
}

func prepararSobreCampana(c *sobreCampana, errores io.Writer) (*contexto, int) {
	var faltan []string
	if strings.TrimSpace(*c.fichero) == "" {
		faltan = append(faltan, "--fichero (el mismo que se subio: las filas no se guardan)")
	}
	if strings.TrimSpace(*c.registro) == "" {
		faltan = append(faltan, "--ledger (los hechos van ahi o no van a ningun sitio)")
	}
	if strings.TrimSpace(*c.campana) == "" {
		faltan = append(faltan, "--campana (cual de ellas)")
	}
	if strings.TrimSpace(*c.quien) == "" {
		faltan = append(faltan, "--quien (quien decide ahora, no quien subio el fichero)")
	}
	if len(faltan) > 0 {
		fmt.Fprintf(errores, "faltan datos: %s\n", strings.Join(faltan, "; "))
		return nil, 2
	}
	instante := time.Now().UTC()
	if *c.ahora != "" {
		t, err := time.Parse(time.RFC3339, *c.ahora)
		if err != nil {
			fmt.Fprintf(errores, "--ahora no es una fecha RFC3339 (%v)\n", err)
			return nil, 2
		}
		instante = t
	}
	l, err := leerLedger(strings.TrimSpace(*c.registro))
	if err != nil {
		fmt.Fprintln(errores, err)
		return nil, 1
	}
	return &contexto{l: l, id: strings.TrimSpace(*c.campana), instante: instante,
		ruta: strings.TrimSpace(*c.registro)}, 0
}

// resolverFila traduce lo que teclea una persona (cuenta y permiso) a la clave.
//
// NO ADIVINA. Si la cuenta no esta, o si esta con varios permisos y no se dijo
// cual, se para y ensena las opciones. Elegir por parecido aqui es revocarle el
// acceso a quien no era.
func resolverFila(ins censo.Instantanea, cuenta, permiso string, errores io.Writer) (string, bool) {
	cuenta = strings.TrimSpace(cuenta)
	permiso = strings.TrimSpace(permiso)
	if cuenta == "" {
		fmt.Fprintln(errores, "falta --cuenta: hay que decir sobre que acceso se decide.")
		return "", false
	}
	var candidatas []censo.Fila
	for _, f := range ins.Filas {
		if f.Cuenta != cuenta {
			continue
		}
		if permiso != "" && f.Permiso != permiso {
			continue
		}
		candidatas = append(candidatas, f)
	}
	if len(candidatas) == 1 {
		return candidatas[0].Clave(), true
	}
	if len(candidatas) == 0 {
		fmt.Fprintf(errores, "no hay ningun acceso de la cuenta %q", cuenta)
		if permiso != "" {
			fmt.Fprintf(errores, " con el permiso %q", permiso)
		}
		fmt.Fprintln(errores, " en este fichero.")
		if cs := cuentasParecidas(ins, cuenta); len(cs) > 0 {
			fmt.Fprintf(errores, "  Cuentas que empiezan igual: %s\n", strings.Join(cs, ", "))
		}
		fmt.Fprintln(errores, "  No se elige por parecido: revocarle el acceso a quien no era no")
		fmt.Fprintln(errores, "  se deshace con un ctrl-z.")
		return "", false
	}
	fmt.Fprintf(errores, "la cuenta %q tiene %d accesos en este fichero y hay que decir cual:\n",
		cuenta, len(candidatas))
	for _, f := range candidatas {
		fmt.Fprintf(errores, "    --permiso %q\n", f.Permiso)
	}
	fmt.Fprintln(errores, "  Se revisa el ACCESO, no la persona: la mitad de las revocaciones de una")
	fmt.Fprintln(errores, "  revision real son de un permiso concreto de alguien que sigue en la empresa.")
	return "", false
}

func cuentasParecidas(ins censo.Instantanea, cuenta string) []string {
	vistas := map[string]bool{}
	pre := strings.ToLower(cuenta)
	if len(pre) > 3 {
		pre = pre[:3]
	}
	for _, f := range ins.Filas {
		if strings.HasPrefix(strings.ToLower(f.Cuenta), pre) {
			vistas[f.Cuenta] = true
		}
	}
	out := make([]string, 0, len(vistas))
	for k := range vistas {
		out = append(out, k)
	}
	sort.Strings(out)
	if len(out) > 6 {
		out = out[:6]
	}
	return out
}

func cmdAccesosDecidir(args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("accesos decidir", flag.ContinueOnError)
	fs.SetOutput(errores)
	c := banderasSobreCampana(fs)
	cuenta := fs.String("cuenta", "", "identificador de la cuenta sobre la que se decide")
	permiso := fs.String("permiso", "", "permiso concreto; hace falta si la cuenta tiene varios")
	veredicto := fs.String("veredicto", "", "aprobar, revocar o delegar")
	motivo := fs.String("motivo", "", "por que; obligatorio para revocar")
	a := fs.String("a", "", "a quien se delega; obligatorio para delegar")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	ctx, codigo := prepararSobreCampana(c, errores)
	if ctx == nil {
		return codigo
	}
	camp, codigo := reconstruirConLaApertura(ctx, *c.fichero, errores)
	if camp == nil {
		return codigo
	}
	v, err := accesos.VeredictoDe(strings.TrimSpace(*veredicto))
	if err != nil {
		fmt.Fprintln(errores, err)
		return 2
	}
	clave, ok := resolverFila(ctx.ins, *cuenta, *permiso, errores)
	if !ok {
		return 2
	}
	d := accesos.Decision{Fila: clave, Veredicto: v, Quien: *c.quien, Cuando: ctx.instante,
		Motivo: *motivo, A: *a}
	// SE REGISTRA EN LA CAMPANA ANTES DE ESCRIBIR EN EL LEDGER. Al reves, una
	// decision mal formada quedaria anotada para siempre en un registro
	// append-only y habria que convivir con ella.
	if err := camp.Registrar(d); err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}
	e, err := accesos.DecisionComoEntrada(d, ctx.ins.Sello(), ctx.id)
	if err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}
	if err := anotarEnLedger(ctx.ruta, ctx.l, e); err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}
	fmt.Fprintf(salida, "%s: %s sobre %s|%s\n", ctx.id, v, ctx.ins.Sistema, *cuenta)
	if v == accesos.Delegar {
		fmt.Fprintf(salida, "  delegado en %s. Delegar traslada la revision, NO la termina: el\n", *a)
		fmt.Fprintln(salida, "  acceso sigue contando como sin revisar y bloquea el cierre.")
	}
	fmt.Fprintln(salida)
	fmt.Fprint(salida, camp.Informar().Texto())
	return 0
}

func cmdAccesosExcusar(args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("accesos excusar", flag.ContinueOnError)
	fs.SetOutput(errores)
	c := banderasSobreCampana(fs)
	desde := fs.Int("desde", 0, "primera linea del fichero que se excusa")
	hasta := fs.Int("hasta", 0, "ultima linea; por defecto la misma que --desde")
	motivo := fs.String("motivo", "", "por que se deja fuera del cierre")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	ctx, codigo := prepararSobreCampana(c, errores)
	if ctx == nil {
		return codigo
	}
	camp, codigo := reconstruirConLaApertura(ctx, *c.fichero, errores)
	if camp == nil {
		return codigo
	}
	e := accesos.Excusa{Desde: *desde, Hasta: *hasta, Quien: *c.quien, Motivo: *motivo,
		Cuando: ctx.instante}
	if e.Hasta < e.Desde {
		e.Hasta = e.Desde
	}
	if err := camp.Excusar(e); err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}
	ent, err := accesos.ExcusaComoEntrada(e, ctx.ins.Sello(), ctx.id)
	if err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}
	if err := anotarEnLedger(ctx.ruta, ctx.l, ent); err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}
	fmt.Fprintf(salida, "lineas %d-%d excusadas por %s: %s\n", e.Desde, e.Hasta, e.Quien, e.Motivo)
	fmt.Fprintln(salida, "Queda escrito en el ledger y sale en el informe: una excusa que no se ve")
	fmt.Fprintln(salida, "es exactamente lo que esto existe para no permitir.")
	fmt.Fprintln(salida)
	fmt.Fprint(salida, camp.Informar().Texto())
	return 0
}

func cmdAccesosCerrar(args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("accesos cerrar", flag.ContinueOnError)
	fs.SetOutput(errores)
	c := banderasSobreCampana(fs)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	ctx, codigo := prepararSobreCampana(c, errores)
	if ctx == nil {
		return codigo
	}
	camp, codigo := reconstruirConLaApertura(ctx, *c.fichero, errores)
	if camp == nil {
		return codigo
	}
	cierre, err := camp.Cerrar(*c.quien, ctx.instante)
	if err != nil {
		fmt.Fprintln(errores, err)
		fmt.Fprintln(errores)
		fmt.Fprint(errores, camp.Informar().Texto())
		return 1
	}
	ent, err := accesos.CierreComoEntrada(cierre, ctx.id)
	if err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}
	if err := anotarEnLedger(ctx.ruta, ctx.l, ent); err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}
	fmt.Fprintf(salida, "Campana %s cerrada por %s el %s.\n", ctx.id, cierre.Quien,
		cierre.Cuando.Format("2006-01-02 15:04 MST"))
	fmt.Fprintf(salida, "  %d accesos, %d decididos, sello %s\n", cierre.Accesos, cierre.Decididos,
		cierre.Sello)
	fmt.Fprintln(salida)
	fmt.Fprint(salida, camp.Informar().Texto())
	return 0
}

// leerLedger lee el ledger, y NO se inventa uno vacio si el fichero existe y no
// se entiende: machacar un registro append-only ilegible pierde lo que hubiera
// dentro y ademas deja un fichero que parece bueno.
func leerLedger(ruta string) (ledger.Ledger, error) {
	var l ledger.Ledger
	b, err := os.ReadFile(ruta) // #nosec G304 -- CLI: la ruta la teclea el operador en su propia maquina
	switch {
	case err == nil:
		if err := json.Unmarshal(b, &l); err != nil {
			return ledger.Ledger{}, fmt.Errorf("%s existe pero no es un ledger legible: %v.\n"+
				"  NO se ha tocado: sobrescribir un registro append-only que no se entiende es "+
				"peor que no escribir", ruta, err)
		}
	case os.IsNotExist(err):
		// Ledger nuevo. Es normal la primera vez.
	default:
		return ledger.Ledger{}, err
	}
	return l, nil
}

func anotarEnLedger(ruta string, l ledger.Ledger, e ledger.Entrada) error {
	if _, err := l.Anadir(e); err != nil {
		return err
	}
	nuevo, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	return escribirConFsync(ruta, append(nuevo, '\n'))
}
