package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/accesos"
	"github.com/marcosmatalab/plazum/nucleo/censo"
)

// `plazum accesos` sube un fichero de cuentas y permisos y abre la campana de
// revision sobre el.
//
// QUE ENSENA Y POR QUE EN ESE ORDEN. Lo primero de todo es LO QUE PLAZUM HA
// ENTENDIDO del fichero: con que codificacion lo leyo, con que separador, que
// columnas ha tirado. Va antes que ningun numero porque quien sube un CSV a las
// nueve de la manana no tiene forma de saber si lo que ve corresponde a su
// fichero, y una decision de lectura que no se cuenta es una suposicion que
// nadie puede revisar.
//
// Despues los cubos, que son la ley de conservacion impresa, y solo entonces el
// hash y el sello, que es lo que hace la revision reproducible por un tercero.
//
// LO QUE ESTA ORDEN NO HACE, dicho para que nadie lo suponga: no recoge
// decisiones. Aprobar y revocar 400 accesos no se hace en una terminal, se hace
// en la interfaz, y esa parte va con el portal. Aqui se sube, se sella, se ve
// que falta y se registra la ingesta.
func cmdAccesos(args []string, salida, errores io.Writer) int {
	// EL DESPACHO. `ver` es la de por defecto porque es la unica que no cambia
	// nada: teclear `plazum accesos` sin mas no puede escribir un hecho.
	if len(args) > 0 {
		switch args[0] {
		case "decidir":
			return cmdAccesosDecidir(args[1:], salida, errores)
		case "excusar":
			return cmdAccesosExcusar(args[1:], salida, errores)
		case "cerrar":
			return cmdAccesosCerrar(args[1:], salida, errores)
		case "ver":
			args = args[1:]
		}
	}

	fs := flag.NewFlagSet("accesos", flag.ContinueOnError)
	fs.SetOutput(errores)
	fichero := fs.String("fichero", "", "CSV de cuentas y permisos exportado del IdP")
	sistema := fs.String("sistema", "", "de que sistema son estas cuentas (el fichero no lo sabe)")
	quien := fs.String("quien", "", "identificador estable de quien sube el fichero")
	fuente := fs.String("fuente", "importacion manual", "de donde salio el fichero")
	retencion := fs.String("retencion", "12 meses desde el cierre de la campana",
		"cuanto se guarda esta lista y por que: es dato personal")
	revisores := fs.String("revisores", "", "CSV opcional 'cuenta;persona' con quien revisa cada cuenta")
	revisor := fs.String("revisor", "", "una sola persona revisa todo (alternativa a --revisores)")
	campana := fs.String("campana", "", "identificador de la campana (por defecto: accesos-<sistema>-<fecha>)")
	ahora := fs.String("ahora", "", "instante RFC3339; por defecto el reloj de esta maquina")
	registro := fs.String("ledger", "", "fichero JSON del ledger al que anadir la ingesta")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*fichero) == "" || strings.TrimSpace(*sistema) == "" || strings.TrimSpace(*quien) == "" {
		fmt.Fprintln(errores, "faltan datos. Uso:")
		fmt.Fprintln(errores, "  plazum accesos --fichero usuarios.csv --sistema erp --quien tu-id")
		fmt.Fprintf(errores, "  ordenes: %s (por defecto, ver)\n", strings.Join(ordenesDeAccesos, ", "))
		fmt.Fprintln(errores, "  decidir, excusar y cerrar necesitan --ledger y el MISMO fichero:")
		fmt.Fprintln(errores, "  las filas no se guardan en ningun sitio, se vuelven a leer.")
		fmt.Fprintln(errores, "")
		fmt.Fprintln(errores, "  --sistema y --quien no tienen valor por defecto a proposito: una lista")
		fmt.Fprintln(errores, "  de cuentas sin de que sistema es y sin quien la subio no se puede atar")
		fmt.Fprintln(errores, "  a nada y no certifica nada.")
		return 2
	}

	instante := time.Now().UTC()
	if *ahora != "" {
		t, err := time.Parse(time.RFC3339, *ahora)
		if err != nil {
			fmt.Fprintf(errores, "--ahora no es una fecha RFC3339 (%v)\n", err)
			return 2
		}
		instante = t
	}

	datos, err := os.ReadFile(*fichero) // #nosec G304 -- CLI: la ruta la teclea el operador en su propia maquina
	if err != nil {
		fmt.Fprintf(errores, "no se puede leer %s: %v\n", *fichero, err)
		return 1
	}

	ins, err := censo.Tomar(datos, censo.Opciones{
		Sistema:   *sistema,
		Fuente:    *fuente,
		Quien:     *quien,
		Tomada:    instante,
		Retencion: *retencion,
		Columnas:  censo.ColumnasHabituales(),
	})
	if err != nil {
		// El error de censo ya es accionable y trae el arreglo dentro: se
		// imprime tal cual, sin envolverlo en "error:" que no anade nada.
		fmt.Fprintln(errores, err)
		return 1
	}

	imprimirLoEntendido(salida, ins)
	imprimirCubosDelFichero(salida, ins)

	id := *campana
	if strings.TrimSpace(id) == "" {
		id = fmt.Sprintf("accesos-%s-%s", *sistema, instante.Format("2006-01-02"))
	}
	asignados, err := leerRevisores(*revisores, *revisor, ins)
	if err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}
	c, err := accesos.Abrir(id, ins, instante, asignados)
	if err != nil {
		fmt.Fprintln(errores, err)
		return 1
	}

	fmt.Fprintln(salida)
	fmt.Fprint(salida, c.Informar().Texto())

	if strings.TrimSpace(*registro) != "" {
		if err := anotarApertura(*registro, ins, id); err != nil {
			fmt.Fprintf(errores, "la instantanea se ha leido pero NO se ha registrado: %v\n", err)
			return 1
		}
		fmt.Fprintf(salida, "\nIngesta anotada en %s.\n", *registro)
	}

	fmt.Fprintln(salida)
	fmt.Fprintln(salida, "Las decisiones (aprobar, revocar, delegar) no se recogen en la terminal:")
	fmt.Fprintln(salida, "cuatrocientos accesos no se revisan tecleando. Van en la interfaz.")
	return 0
}

// imprimirLoEntendido va PRIMERO. Quien sube el fichero no tiene otra forma de
// saber si lo que ve corresponde a lo que subio.
func imprimirLoEntendido(w io.Writer, ins censo.Instantanea) {
	fmt.Fprintln(w, "LO QUE PLAZUM HA ENTENDIDO DEL FICHERO")
	fmt.Fprintf(w, "  codificacion: %s\n", ins.Notas.Codificacion)
	if ins.Notas.QuitadoElBOM {
		fmt.Fprintln(w, "  traia BOM al principio (cosa de Excel) y se ha quitado antes de leer")
	}
	fmt.Fprintf(w, "  separador:    %s\n", ins.Notas.Separador)
	fmt.Fprintf(w, "  columnas:     %s\n", strings.Join(ins.Notas.Cabeceras, ", "))
	if len(ins.Notas.ColumnasIgnoradas) > 0 {
		fmt.Fprintf(w, "  descartadas:  %s\n", strings.Join(ins.Notas.ColumnasIgnoradas, ", "))
		fmt.Fprintln(w, "                (no se guardan: la lista es dato personal y solo se")
		fmt.Fprintln(w, "                 conserva lo minimo para poder revisar)")
	}
	for _, a := range ins.Notas.Avisos {
		fmt.Fprintf(w, "  AVISO: %s\n", a)
	}
}

func imprimirCubosDelFichero(w io.Writer, ins censo.Instantanea) {
	c := ins.Cuenta()
	fmt.Fprintln(w)
	fmt.Fprintf(w, "EL FICHERO, LINEA A LINEA (%d lineas de datos)\n", c["lineas de datos"])
	fmt.Fprintf(w, "  accesos que se pueden revisar: %d\n", c["legibles"])
	fmt.Fprintf(w, "  lineas ilegibles:              %d\n", c["ilegibles"])
	fmt.Fprintf(w, "  filas repetidas:               %d\n", c["duplicadas"])
	for _, il := range ins.Ilegibles {
		if il.Desde == il.Hasta {
			fmt.Fprintf(w, "    linea %d: %s\n", il.Desde, il.Motivo)
			continue
		}
		fmt.Fprintf(w, "    lineas %d-%d: %s\n", il.Desde, il.Hasta, il.Motivo)
	}
	if !ins.Cuadra() {
		// No deberia poder pasar: censo.Tomar no devuelve instantaneas
		// descuadradas. Se imprime igual porque una guarda que solo existe rio
		// arriba es una guarda que se cae el dia que alguien construya la
		// instantanea por otro camino.
		fmt.Fprintln(w, "  AVISO: los cubos NO cubren todas las lineas. Este recuento no vale.")
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "IDENTIDAD DE ESTA INSTANTANEA")
	fmt.Fprintf(w, "  sha256 del fichero: %s\n", ins.Hash)
	fmt.Fprintf(w, "  sello de la lectura: %s\n", ins.Sello())
	fmt.Fprintf(w, "  retencion declarada: %s\n", ins.Retencion)
}

// leerRevisores admite las dos formas y no las mezcla.
func leerRevisores(ruta, unico string, ins censo.Instantanea) (map[string]string, error) {
	if strings.TrimSpace(ruta) != "" && strings.TrimSpace(unico) != "" {
		return nil, fmt.Errorf("--revisores y --revisor dicen dos cosas a la vez. Elige una: " +
			"un fichero con quien revisa cada cuenta, o una sola persona para todo")
	}
	out := map[string]string{}
	if strings.TrimSpace(unico) != "" {
		for _, f := range ins.Filas {
			out[f.Clave()] = unico
		}
		return out, nil
	}
	if strings.TrimSpace(ruta) == "" {
		return out, nil
	}
	b, err := os.ReadFile(ruta) // #nosec G304 -- CLI: la ruta la teclea el operador en su propia maquina
	if err != nil {
		return nil, fmt.Errorf("no se puede leer el fichero de revisores %s: %v", ruta, err)
	}
	// Se empareja por CUENTA y no por la clave entera: nadie teclea
	// "erp|u1|admin" en un fichero a mano, y la persona que revisa lo hace por
	// cuenta, no por permiso.
	porCuenta := map[string]string{}
	for n, linea := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
		linea = strings.TrimSpace(linea)
		if linea == "" || strings.HasPrefix(linea, "#") {
			continue
		}
		partes := strings.SplitN(linea, ";", 2)
		if len(partes) != 2 || strings.TrimSpace(partes[1]) == "" {
			return nil, fmt.Errorf("%s linea %d: se esperaba 'cuenta;persona' y hay %q",
				ruta, n+1, linea)
		}
		porCuenta[strings.TrimSpace(partes[0])] = strings.TrimSpace(partes[1])
	}
	for _, f := range ins.Filas {
		if p, hay := porCuenta[f.Cuenta]; hay {
			out[f.Clave()] = p
		}
	}
	return out, nil
}

// anotarApertura anade la subida al ledger, encadenada.
//
// LA CARGA LA ARMA nucleo/accesos, no esto. La primera version tenia aqui su
// propia estructura `cargaDeIngesta` con los mismos campos, o sea una SEGUNDA
// definicion del formato que viaja: el dia que una gane un campo, la otra lo
// pierde y nadie se entera hasta que un ledger antiguo no se deja leer. Es la
// misma familia que la conversion de regimen duplicada que encontro staticcheck.
func anotarApertura(ruta string, ins censo.Instantanea, campana string) error {
	l, err := leerLedger(ruta)
	if err != nil {
		return err
	}
	e, err := accesos.AperturaComoEntrada(ins, campana)
	if err != nil {
		return err
	}
	return anotarEnLedger(ruta, l, e)
}

// escribirConFsync escribe y sincroniza. Un ledger que se pierde en el cache de
// pagina cuando se va la luz es un ledger que dice menos de lo que promete.
func escribirConFsync(ruta string, datos []byte) error {
	f, err := os.OpenFile(ruta, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) // #nosec G304 -- CLI: la ruta la teclea el operador en su propia maquina
	if err != nil {
		return err
	}
	if _, err := f.Write(datos); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
