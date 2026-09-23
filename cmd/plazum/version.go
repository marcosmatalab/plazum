package main

// `plazum version` y `plazum --version`: que binario es este.
//
// Hasta el 23-09-2026 `plazum --version` imprimia la ayuda, o sea que nadie
// podia decir que version tenia instalada sin comparar tamanos de fichero. En
// un producto cuya release firma los binarios y ancla su corpus, no saber que
// binario tienes delante deja la firma sin a quien aplicarse.
//
// DE DONDE SALE CADA DATO, por orden de preferencia:
//
//	version   la que inyecta la release con -ldflags (versionPublicada); si no,
//	          la que Go graba al instalar desde un tag (go install ...@v0.2.0);
//	          si no, "(devel)", que es lo que Go dice de un binario compilado
//	          desde el codigo y no desde una version publicada.
//	commit    vcs.revision de la informacion de construccion, cuando existe:
//	          un go install desde el proxy de modulos no la trae, y entonces
//	          la linea no sale en vez de salir con un valor inventado.
//
// LO VIGILA: TestLaVersionSaleDeLaReleaseDeGoOEsDevel

import (
	"flag"
	"fmt"
	"io"
	"runtime"
	"runtime/debug"
)

// versionPublicada la inyecta la release:
//
//	go build -ldflags "-X main.versionPublicada=v0.2.0" ./cmd/plazum
//
// Vacia es "no la inyecto nadie", y entonces manda lo que sepa Go.
var versionPublicada string

// datosDeVersion es lo que imprime `plazum version`, separado de como se
// imprime para poder probar el formato sin compilar un binario por caso.
type datosDeVersion struct {
	Version    string
	Commit     string
	Fecha      string
	Modificado bool
	Go         string
	Sistema    string
}

// leerVersion combina lo inyectado por la release con lo que Go grabo en el
// binario. bi puede ser nil: un binario sin informacion de construccion sigue
// teniendo que decir algo cierto.
func leerVersion(inyectada string, bi *debug.BuildInfo) datosDeVersion {
	d := datosDeVersion{
		Version: "(devel)",
		Go:      runtime.Version(),
		Sistema: runtime.GOOS + "/" + runtime.GOARCH,
	}
	if bi != nil {
		if v := bi.Main.Version; v != "" {
			d.Version = v
		}
		if bi.GoVersion != "" {
			d.Go = bi.GoVersion
		}
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				d.Commit = s.Value
			case "vcs.time":
				d.Fecha = s.Value
			case "vcs.modified":
				d.Modificado = s.Value == "true"
			}
		}
	}
	if inyectada != "" {
		d.Version = inyectada
	}
	return d
}

// escribirVersion imprime la version en la primera linea, sola, para que un
// script la lea con `plazum version | head -1`, y el detalle debajo.
func escribirVersion(w io.Writer, d datosDeVersion) {
	fmt.Fprintf(w, "plazum %s\n", d.Version)
	if d.Commit != "" {
		linea := d.Commit
		if d.Fecha != "" {
			linea += " (" + d.Fecha + ")"
		}
		if d.Modificado {
			linea += ", con cambios sin commitear"
		}
		fmt.Fprintf(w, "  commit   %s\n", linea)
	}
	fmt.Fprintf(w, "  go       %s\n", d.Go)
	fmt.Fprintf(w, "  sistema  %s\n", d.Sistema)
}

func cmdVersion(args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(errores)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(errores, "error: version no lleva argumentos y ha recibido %q\n", fs.Args())
		return 2
	}
	bi, _ := debug.ReadBuildInfo()
	escribirVersion(salida, leerVersion(versionPublicada, bi))
	return 0
}
