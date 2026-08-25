package main

// `plazum doctor`: por que no funciona, y que hacer para que funcione.
//
// La regla del comando, y la unica que importa: NINGUNA linea que no este
// correcta sale sin decir como se arregla. Un diagnostico que solo dice "fallo"
// le pasa el trabajo al operador, que es justo lo que esta pieza existe para
// evitar. Lo vigila la suite de contrato del puerto, no la buena voluntad.
//
// Y la segunda mitad, que es la que convierte el comando en soporte gratis:
// `--issue` imprime el mismo diagnostico en un bloque copiable a un issue, con
// las rutas del usuario redactadas. El informe de fallo lo genera la
// herramienta, no la persona que esta enfadada porque algo no arranca.

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"plazum/adaptadores/diagnostico"
	"plazum/puertos"
)

func cmdDoctor(args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("plazum doctor", flag.ContinueOnError)
	fs.SetOutput(errores)
	datos := fs.String("datos", ".", "directorio de datos de la instalacion")
	corpusDir := fs.String("corpus", "", "directorio de paquetes; vacio es <datos>/paquetes")
	direccion := fs.String("direccion", diagnostico.DireccionPorDefecto, "direccion en la que va a escuchar el servidor")
	keystore := fs.String("keystore", "", "fichero de claves; vacio es <datos>/keystore.json")
	contexto := fs.String("contexto", "", "fichero de contexto del receptor, para comprobar las raices de TSA que declaras")
	issue := fs.Bool("issue", false, "imprime el diagnostico en un bloque copiable a un issue, con las rutas redactadas")
	ahoraTxt := fs.String("ahora", "", "instante desde el que se juzga (RFC3339); vacio es el reloj del sistema")
	fs.Usage = func() {
		fmt.Fprintln(errores, "uso: plazum doctor [--datos DIR] [--corpus DIR] [--direccion HOST:PUERTO] [--issue]")
		fmt.Fprintln(errores, "")
		fmt.Fprintln(errores, "Comprueba lo que puede fallar de verdad en esta maquina y dice como se")
		fmt.Fprintln(errores, "arregla cada cosa. Termina con codigo 1 si algo esta roto, 0 si solo hay")
		fmt.Fprintln(errores, "avisos: sirve como comprobacion de arranque en un systemd o en un CI.")
		fmt.Fprintln(errores, "")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	o := diagnostico.Opciones{
		Datos: *datos, Corpus: *corpusDir, Direccion: *direccion, Keystore: *keystore,
	}
	if *ahoraTxt != "" {
		t, err := time.Parse(time.RFC3339, *ahoraTxt)
		if err != nil {
			fmt.Fprintf(errores, "error: --ahora %q no es una fecha RFC3339: %v\n", *ahoraTxt, err)
			return 2
		}
		o.Ahora = t.UTC()
	}
	if *contexto != "" {
		raices, err := raicesDelFichero(*contexto)
		if err != nil {
			fmt.Fprintln(errores, "error:", err)
			return 2
		}
		o.RaicesTSA = raices
	}

	cs := diagnostico.Nuevo(o).Comprobar(context.Background())
	if *issue {
		imprimirParaIssue(salida, cs)
	} else {
		imprimirDiagnostico(salida, cs)
	}
	for _, c := range cs {
		if c.Estado == puertos.Roto {
			return 1
		}
	}
	return 0
}

// raicesDelFichero saca el bloque raices_tsa del contexto del receptor. Se
// reutiliza el tipo que ya lee `plazum verify` para que las dos ordenes entiendan
// exactamente el mismo fichero: dos lecturas distintas del mismo formato son
// dos formatos.
func raicesDelFichero(ruta string) ([]byte, error) {
	b, err := os.ReadFile(ruta) // #nosec G304 -- CLI: la ruta la teclea el operador en su propia maquina
	if err != nil {
		return nil, fmt.Errorf("no puedo leer el contexto %s: %w; quita --contexto si solo "+
			"quieres comprobar las raices que trae el binario", ruta, err)
	}
	var f ficheroContexto
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("el contexto %s no es JSON valido: %w", ruta, err)
	}
	if f.RaicesTSA == "" {
		return nil, nil
	}
	return []byte(f.RaicesTSA), nil
}

func imprimirDiagnostico(w io.Writer, cs []puertos.Comprobacion) {
	rotos, avisos := 0, 0
	fmt.Fprintln(w)
	for _, c := range cs {
		fmt.Fprintf(w, "  %-9s %-12s %s\n", rotulo(c.Estado), c.Nombre, c.Detalle)
		if c.Estado == puertos.Correcto {
			continue
		}
		if c.Estado == puertos.Roto {
			rotos++
		} else {
			avisos++
		}
		// La sangria de la continuacion es EXACTAMENTE el ancho del prefijo de
		// arriba (2 + 9 + 1 + 12 + 1 + len("arreglo: ")), para que el arreglo
		// salga como un bloque y no como texto desalineado.
		fmt.Fprintf(w, "  %-9s %-12s arreglo: %s\n", "", "", envolver(c.Arreglo, 66, strings.Repeat(" ", 34)))
	}
	fmt.Fprintln(w)
	switch {
	case rotos > 0:
		fmt.Fprintf(w, "  %d roto(s) y %d aviso(s). Lo roto impide que plazum funcione; arreglalo\n",
			rotos, avisos)
		fmt.Fprintf(w, "  por orden, de arriba abajo, porque lo de arriba causa lo de abajo.\n")
	case avisos > 0:
		fmt.Fprintf(w, "  Nada roto y %d aviso(s). plazum funciona; los avisos dicen que te estas\n", avisos)
		fmt.Fprintf(w, "  perdiendo y cuando empezara a importar.\n")
	default:
		fmt.Fprintf(w, "  Todo correcto. Si algo no va, es un fallo nuestro: `plazum doctor --issue`\n")
		fmt.Fprintf(w, "  genera el informe para abrir la incidencia.\n")
	}
	fmt.Fprintln(w)
}

// imprimirParaIssue saca el mismo diagnostico en Markdown, listo para pegar.
//
// La redaccion NO es cosmetica: la salida lleva rutas absolutas, y una ruta
// absoluta en Linux o en Windows lleva dentro el nombre de usuario, que es un
// dato personal. Publicarlo en un issue de un repositorio publico es una cesion
// que nadie ha consentido, asi que se sustituye antes de imprimir y no se
// confia en que el operador se acuerde de borrarlo.
func imprimirParaIssue(w io.Writer, cs []puertos.Comprobacion) {
	fmt.Fprintln(w, "<!-- generado por `plazum doctor --issue`; rutas de usuario redactadas -->")
	fmt.Fprintln(w, "```")
	fmt.Fprintf(w, "sistema   %s/%s, go %s\n", runtime.GOOS, runtime.GOARCH, runtime.Version())
	for _, c := range cs {
		fmt.Fprintf(w, "%-9s %-12s %s\n", c.Estado, c.Nombre, redactar(c.Detalle))
		if c.Estado != puertos.Correcto {
			fmt.Fprintf(w, "%-9s %-12s arreglo: %s\n", "", "", redactar(c.Arreglo))
		}
	}
	fmt.Fprintln(w, "```")
}

// redactar quita de un texto lo que identifica a una persona o a su maquina.
//
// Se hace por sustitucion de las rutas conocidas y no por una expresion regular
// que adivine nombres: adivinar produce dos fallos, tapar lo que no toca y
// dejar pasar lo que si. Lo que no se sepa redactar se queda, y por eso el
// bloque lleva escrito que es una salida generada: quien la pegue puede mirarla.
func redactar(s string) string {
	casa, err := os.UserHomeDir()
	if err == nil && casa != "" {
		s = strings.ReplaceAll(s, casa, "~")
		s = strings.ReplaceAll(s, filepath.ToSlash(casa), "~")
		if usuario := filepath.Base(casa); len(usuario) > 2 {
			s = strings.ReplaceAll(s, usuario, "<usuario>")
		}
	}
	if tmp := os.TempDir(); tmp != "" && len(tmp) > 4 {
		s = strings.ReplaceAll(s, tmp, "<temporal>")
		s = strings.ReplaceAll(s, filepath.ToSlash(tmp), "<temporal>")
	}
	return s
}

func rotulo(e puertos.EstadoComprobacion) string {
	switch e {
	case puertos.Correcto:
		return "  ok"
	case puertos.Aviso:
		return "  AVISO"
	default:
		return "  ROTO"
	}
}

// envolver parte un arreglo largo para que quepa en un terminal estrecho. Un
// arreglo que se sale de la pantalla se lee a medias, y un arreglo leido a
// medias es peor que ninguno.
func envolver(s string, ancho int, sangria string) string {
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
