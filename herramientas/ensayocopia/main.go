// Command ensayocopia es el ensayo de restauracion: crea una instalacion, la
// copia, DESTRUYE el original, la restaura y verifica lo restaurado con el
// verificador del nucleo.
//
// POR QUE NO BASTA CON COMPROBAR QUE EL FICHERO EXISTE. Una copia que restaura
// bytes y deja un ledger que no verifica es peor que no tener copia, porque da
// confianza sin darla: nadie mira un backup hasta el dia que hace falta, y ese
// dia ya no hay de donde sacar otro. Por eso el ensayo termina donde termina un
// tercero: cadena encadenada, lapidas firmadas con su base legal, y supresiones
// que siguen siendo supresiones.
//
// Uso:
//
//	ensayocopia ensayo [-dir D] [-romper MODO] [-expediente F]
//	ensayocopia verificar -dir D -confianza C
//	ensayocopia modos
//
// `verificar` es el que sirve fuera del ensayo: se apunta al directorio que
// acaba de dejar un `litestream restore` de verdad y dice si lo restaurado
// prueba algo. El procedimiento completo, con lo que se copia y lo que no, esta
// en docs/copias.md.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Codigos de salida. Separados a proposito: un control negativo tiene que poder
// exigir el rojo QUE ESPERA y no cualquier rojo. Un fallo de compilacion, un
// directorio sin permisos y "la copia esta rota" salen los tres con codigo
// distinto de cero, y confundirlos es como se cuelan las puertas que no vigilan.
const (
	salidaOK       = 0
	salidaFallo    = 1 // el ensayo no pudo ni ejecutarse (disco, permisos, escenario)
	salidaUso      = 2 // el operador se equivoco al invocarlo
	salidaNoPrueba = 3 // el ensayo se ejecuto y lo restaurado NO prueba nada
)

func main() {
	os.Exit(ejecutar(os.Args[1:], os.Stdout, os.Stderr))
}

func uso(w io.Writer) {
	fmt.Fprintln(w, "uso: ensayocopia ensayo    [-dir D] [-romper MODO] [-expediente F] [-semilla S]")
	fmt.Fprintln(w, "     ensayocopia verificar  -dir D -confianza C")
	fmt.Fprintln(w, "     ensayocopia modos")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "ensayo     siembra, copia, destruye el original, restaura y verifica.")
	fmt.Fprintln(w, "verificar  comprueba un directorio ya restaurado (el de un litestream restore).")
	fmt.Fprintln(w, "modos      lista las formas de romper la copia que el ensayo sabe provocar.")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Codigos de salida: 0 bien, 1 no pudo ejecutarse, 2 uso, 3 lo restaurado no prueba nada.")
}

func ejecutar(args []string, salida, errores io.Writer) int {
	if len(args) == 0 {
		uso(errores)
		return salidaUso
	}
	switch args[0] {
	case "ensayo":
		return cmdEnsayo(args[1:], salida, errores)
	case "verificar":
		return cmdVerificar(args[1:], salida, errores)
	case "modos":
		for _, m := range ModosRotos {
			fmt.Fprintf(salida, "%-22s rompe %s\n%-22s lo caza %s\n", m.Nombre, m.Que, "", m.Caza)
		}
		return salidaOK
	case "-h", "--help", "help":
		uso(salida)
		return salidaOK
	default:
		fmt.Fprintf(errores, "orden desconocida %q\n", args[0])
		uso(errores)
		return salidaUso
	}
}

func cmdVerificar(args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("ensayocopia verificar", flag.ContinueOnError)
	fs.SetOutput(errores)
	dir := fs.String("dir", "", "directorio restaurado que hay que verificar")
	confianza := fs.String("confianza", "", "fichero con la clave publica del operador; TIENE que estar fuera de -dir")
	if err := fs.Parse(args); err != nil {
		return salidaUso
	}
	if *dir == "" || *confianza == "" {
		fmt.Fprintln(errores, "faltan -dir y -confianza.")
		fmt.Fprintln(errores, "  -confianza es el fichero que aportas TU, con la clave publica del operador.")
		fmt.Fprintln(errores, "  Sirve el mismo contexto que le pasas a `plazum verify`.")
		return salidaUso
	}
	r, err := Verificar(*dir, *confianza)
	if err != nil {
		return informarFallo(errores, err)
	}
	informar(salida, r)
	return salidaOK
}

// informarFallo separa el rojo de la COPIA del rojo de la LINEA DE ORDENES.
//
// Hallazgo de la pasada del comprador: teclear mal la ruta del contexto salia
// con "LO RESTAURADO NO PRUEBA NADA" y codigo 3, o sea que quien lo leyera a las
// tres de la manana se pondria a restaurar otra generacion en vez de mirar la
// ruta. Un contexto a medias no invalida lo restaurado, invalida la
// verificacion, y son dos cosas distintas.
func informarFallo(errores io.Writer, err error) int {
	if errors.Is(err, ErrContextoIlegible) {
		fmt.Fprintf(errores, "NO SE HA PODIDO VERIFICAR, y no es culpa de la copia: %v\n", err)
		return salidaUso
	}
	fmt.Fprintf(errores, "LO RESTAURADO NO PRUEBA NADA: %v\n", err)
	return salidaNoPrueba
}

func informar(w io.Writer, r Resultado) {
	fmt.Fprintf(w, "cadena verificada: %d entradas, %d abiertas con las claves restauradas\n",
		r.Entradas, r.Vivas)
	fmt.Fprintf(w, "evidencias abiertas y comprobadas contra su direccion: %d\n", r.EvidenciasVivas)
	fmt.Fprintf(w, "supresiones que siguen siendo supresiones: %d\n", len(r.Supresiones))
	// Y con que fuerza se ha comprobado. Sin acta del receptor, quitar una lapida
	// y recolgar una evidencia suprimida de una entrada viva son INVISIBLES
	// (docs/modelo-de-amenaza.md, ataque 14). Un verde mas debil que se lee igual
	// que uno fuerte es lo que este proyecto no hace, asi que se dice cual es.
	if r.ConActa {
		fmt.Fprintf(w, "  con acta del receptor: tambien se ha comprobado que ninguna supresion\n"+
			"  ha desaparecido y que ninguna evidencia suprimida se abre, cuelgue de donde cuelgue\n")
	} else {
		fmt.Fprintf(w, "  SIN acta del receptor: NO se ha comprobado que las supresiones sigan\n"+
			"  estando ni que una evidencia suprimida no se haya recolgado de una entrada viva.\n"+
			"  Las dos cosas son indetectables mirando solo la replica. Ver el ataque 14 en\n"+
			"  docs/modelo-de-amenaza.md\n")
	}
	for _, s := range r.Supresiones {
		fmt.Fprintf(w, "  %s\n", s)
	}
	if r.MaestraAusente {
		fmt.Fprintf(w, "aviso: la copia no trae %s, y es CORRECTO que no la traiga.\n", NombreMaestra)
		fmt.Fprintln(w, "  El historico verifica sin ella. Lo que no se puede hasta reponerla desde")
		fmt.Fprintln(w, "  la custodia es firmar lapidas nuevas ni cerrar checkpoints nuevos.")
	}
}

func cmdEnsayo(args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("ensayocopia ensayo", flag.ContinueOnError)
	fs.SetOutput(errores)
	dir := fs.String("dir", "", "directorio de trabajo del ensayo; vacio es uno temporal que se borra al salir")
	romper := fs.String("romper", "", "rompe la copia de esta manera y espera que el ensayo salga en rojo (ver `modos`)")
	expediente := fs.String("expediente", "", "expediente emitido que se mete en la instalacion para que viaje en la copia")
	semilla := fs.String("semilla", "ensayo", "semilla del material deterministico de la siembra")
	if err := fs.Parse(args); err != nil {
		return salidaUso
	}
	if *romper != "" && !modoConocido(*romper) {
		fmt.Fprintf(errores, "modo de rotura desconocido %q. Los que hay:\n", *romper)
		for _, m := range ModosRotos {
			fmt.Fprintf(errores, "  %s\n", m.Nombre)
		}
		return salidaUso
	}

	trabajo := *dir
	if trabajo == "" {
		t, err := os.MkdirTemp("", "plazum-ensayocopia-")
		if err != nil {
			fmt.Fprintf(errores, "no puedo crear el directorio de trabajo: %v\n", err)
			return salidaFallo
		}
		trabajo = t
		defer func() { _ = os.RemoveAll(trabajo) }()
	}
	if err := os.MkdirAll(trabajo, 0o750); err != nil {
		fmt.Fprintf(errores, "no puedo crear %s: %v\n", trabajo, err)
		return salidaFallo
	}

	vivo := filepath.Join(trabajo, "vivo")
	replica := filepath.Join(trabajo, "replica")
	restaurado := filepath.Join(trabajo, "restaurado")
	confianza := filepath.Join(trabajo, "confianza.json")

	paso := func(n int, texto string) { fmt.Fprintf(salida, "%d. %s\n", n, texto) }

	esc, err := cargarEscenario()
	if err != nil {
		fmt.Fprintf(errores, "%v\n", err)
		return salidaFallo
	}

	// 1. Sembrar.
	s, err := Sembrar(vivo, *semilla, esc)
	if err != nil {
		fmt.Fprintf(errores, "no puedo sembrar la instalacion: %v\n", err)
		return salidaFallo
	}
	if err := EscribirConfianzaConActa(confianza, s.ClaveOperador, s.Acta); err != nil {
		fmt.Fprintf(errores, "%v\n", err)
		return salidaFallo
	}
	paso(1, fmt.Sprintf("instalacion sembrada en %s: %d entradas, %d evidencias, "+
		"entrada %d borrada con base legal %q",
		vivo, s.Entradas, s.Evidencias, s.EntradaBorrada, s.BaseLegal))

	if *expediente != "" {
		b, err := os.ReadFile(*expediente) // #nosec G304 -- ruta que teclea el operador
		if err != nil {
			fmt.Fprintf(errores, "no puedo leer el expediente %s: %v\n", *expediente, err)
			return salidaFallo
		}
		// #nosec G703 -- vivo cuelga de -dir, que teclea el operador en su propia
		// maquina, y el nombre del fichero es una constante. No hay entrada de
		// terceros en esta ruta.
		if err := os.WriteFile(filepath.Join(vivo, NombreExpediente), b, 0o600); err != nil {
			fmt.Fprintf(errores, "no puedo meter el expediente en la instalacion: %v\n", err)
			return salidaFallo
		}
		paso(1, fmt.Sprintf("   y el expediente emitido %s, que viajara en la copia", *expediente))
	}

	// 2. Copiar.
	m, err := Copiar(vivo, replica, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		fmt.Fprintf(errores, "no puedo hacer la copia: %v\n", err)
		return salidaFallo
	}
	nombres := make([]string, 0, len(m.Artefactos))
	for n := range m.Artefactos {
		nombres = append(nombres, n)
	}
	paso(2, fmt.Sprintf("copia hecha en %s: %s", replica, strings.Join(ordenar(nombres), ", ")))
	for _, n := range noSeCopia {
		paso(2, fmt.Sprintf("   NO se copia %s: %s", n.Fichero, n.Motivo))
	}

	if *romper != "" {
		aplicado, err := RomperReplica(replica, *romper, *semilla)
		if err != nil {
			fmt.Fprintf(errores, "no puedo aplicar la rotura %q: %v\n", *romper, err)
			return salidaFallo
		}
		if aplicado {
			paso(2, fmt.Sprintf("   ROTURA APLICADA sobre la copia: %s", *romper))
		}
	}

	// 3. Destruir el original. Es el paso que hace que esto sea un ensayo: sin
	//    destruirlo, nada garantiza que lo que se verifica despues no sea el
	//    original de siempre.
	if err := os.RemoveAll(vivo); err != nil {
		fmt.Fprintf(errores, "no puedo destruir la instalacion original: %v\n", err)
		return salidaFallo
	}
	if _, err := os.Stat(vivo); !os.IsNotExist(err) {
		fmt.Fprintf(errores, "la instalacion original sigue en %s despues de borrarla. "+
			"Sin destruirla de verdad, el ensayo verificaria el original y saldria verde "+
			"aunque la copia estuviera vacia\n", vivo)
		return salidaFallo
	}
	paso(3, fmt.Sprintf("original destruido: %s ya no existe", vivo))

	// 4. Restaurar.
	if err := Restaurar(replica, restaurado); err != nil {
		fmt.Fprintf(errores, "LA RESTAURACION SE NIEGA: %v\n", err)
		return salidaNoPrueba
	}
	if !mismoContenido(filepath.Join(replica, NombreBase), filepath.Join(restaurado, NombreBase)) {
		fmt.Fprintf(errores, "la base restaurada no es byte a byte la de la replica. "+
			"Restaurar tiene que devolver los mismos bytes, no unos parecidos\n")
		return salidaNoPrueba
	}
	paso(4, fmt.Sprintf("restaurado en %s", restaurado))

	confianzaUsada, err := RomperTrasRestaurar(restaurado, *romper, confianza)
	if err != nil {
		fmt.Fprintf(errores, "no puedo aplicar la rotura %q: %v\n", *romper, err)
		return salidaFallo
	}

	// 5. Verificar, que es donde termina un tercero.
	r, err := Verificar(restaurado, confianzaUsada)
	if err != nil {
		return informarFallo(errores, err)
	}
	paso(5, "verificado con el verificador del nucleo:")
	informar(salida, r)
	if *romper != "" {
		fmt.Fprintf(errores, "PUERTA ROTA: se rompio la copia con %q y el ensayo ha salido "+
			"en VERDE.\n  Esa comprobacion no vigila nada. Arreglo: mira que la rotura se "+
			"aplicara de verdad (el modo la aplica sobre %s) y que la comprobacion que "+
			"tenia que cazarla siga en verificar.go\n", *romper, replica)
		return salidaFallo
	}
	return salidaOK
}

func ordenar(s []string) []string {
	out := append([]string(nil), s...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
