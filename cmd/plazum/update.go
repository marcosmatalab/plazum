package main

// `plazum update`: actualizar sin poder quedarse tirado.
//
// La decision de interfaz que hay detras, y por que no es paternalismo. `plazum
// update` a secas NO actualiza: consulta, ensena las notas de la version y dice
// el comando exacto para aplicarla. Aplicar exige `--aplicar`. En un producto
// que vigila plazos legales, una actualizacion que sale mal deja a alguien sin
// ver un vencimiento, asi que leer que cambia antes de aplicarlo no es un paso
// de mas: es el paso.
//
// Y lo que hace que se pueda actualizar tranquilo: cada `--aplicar` deja un
// punto de retorno comprobado, y `--deshacer` vuelve de verdad. Si el punto no
// existe o su copia no cuadra, se dice y NO se restaura nada. Un operador que
// cree que ha vuelto atras y no ha vuelto esta peor que uno que sabe que no
// puede.

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/marcosmatalab/plazum/adaptadores/actualizador"
)

func cmdUpdate(args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("plazum update", flag.ContinueOnError)
	fs.SetOutput(errores)
	raiz := fs.String("raiz", ".", "directorio de la instalacion que se actualiza")
	canal := fs.String("canal", "", "directorio del canal de versiones")
	aplicar := fs.Bool("aplicar", false, "aplica de verdad la version (sin esto, solo se consulta)")
	version := fs.String("version", "", "version concreta a aplicar; vacio es la mas nueva del canal")
	deshacer := fs.String("deshacer", "", "vuelve al punto de retorno indicado")
	puntos := fs.Bool("puntos", false, "lista los puntos de retorno guardados")
	reparar := fs.Bool("reparar", false, "deshace una actualizacion que quedo a medias")
	fs.Usage = func() {
		fmt.Fprintln(errores, "uso: plazum update --canal DIR                    consulta si hay version nueva")
		fmt.Fprintln(errores, "     plazum update --canal DIR --aplicar          la aplica, dejando punto de retorno")
		fmt.Fprintln(errores, "     plazum update --puntos                       lista los puntos de retorno")
		fmt.Fprintln(errores, "     plazum update --deshacer PUNTO               vuelve a un punto de retorno")
		fmt.Fprintln(errores, "     plazum update --reparar                      deshace una actualizacion a medias")
		fmt.Fprintln(errores, "")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	a := actualizador.Nuevo(actualizador.Opciones{
		Raiz: *raiz, Canal: canalDe(*canal), Ahora: time.Now().UTC(),
	})
	ctx := context.Background()

	// El estado a medias se avisa SIEMPRE, haga lo que haga el operador: es la
	// unica situacion en la que la instalacion puede estar mintiendo sobre que
	// version tiene, y callarselo mientras se hace otra cosa seria esconderlo.
	if m, hay, err := a.Interrumpida(); err != nil {
		fmt.Fprintln(errores, "AVISO:", err)
	} else if hay && !*reparar {
		reparacion := "plazum update --reparar"
		if *raiz != "." {
			reparacion = "plazum update --raiz " + *raiz + " --reparar"
		}
		fmt.Fprintf(errores, "AVISO: hay una actualizacion a medias (de %s a %s, punto %s). "+
			"Ejecuta `%s` antes de nada.\n", oNinguna(m.Desde), m.Hacia, m.Punto, reparacion)
	}

	// El prefijo con el que se le vuelve a llamar. Un comando sugerido que se
	// deja por el camino las opciones que el operador acaba de teclear no
	// funciona cuando lo copia, y entonces el mensaje de ayuda es una trampa.
	inv := "plazum update"
	if *raiz != "." {
		inv += " --raiz " + *raiz
	}

	switch {
	case *puntos:
		return listarPuntos(a, inv, salida, errores)
	case *deshacer != "":
		if err := a.Deshacer(ctx, *deshacer); err != nil {
			fmt.Fprintln(errores, "error:", err)
			return 1
		}
		v, _ := a.VersionInstalada()
		fmt.Fprintf(salida, "Vuelto al punto %s. La instalacion esta en la version %s.\n", *deshacer, oNinguna(v))
		return 0
	case *reparar:
		punto, err := a.Reparar(ctx)
		if err != nil {
			fmt.Fprintln(errores, "error:", err)
			return 1
		}
		if punto == "" {
			fmt.Fprintln(salida, "No hay ninguna actualizacion a medias. Nada que reparar.")
			return 0
		}
		v, _ := a.VersionInstalada()
		fmt.Fprintf(salida, "Reparado: se deshizo la actualizacion a medias volviendo al punto %s. "+
			"La instalacion esta en la version %s.\n", punto, oNinguna(v))
		return 0
	case *aplicar:
		return aplicarVersion(ctx, a, *version, *canal, inv, salida, errores)
	default:
		return consultar(ctx, a, *canal, inv, salida, errores)
	}
}

// canalDe construye el canal, o nil si no se declaro. Nil es un estado
// legitimo: `--puntos` y `--deshacer` funcionan sin canal, porque volver atras
// no puede depender de que el sitio de donde vino la version siga en pie.
func canalDe(dir string) actualizador.Canal {
	if dir == "" {
		return nil
	}
	return actualizador.CanalDirectorio{Dir: dir}
}

func oNinguna(v string) string {
	if v == "" {
		return "(sin fichero VERSION)"
	}
	return v
}

func consultar(ctx context.Context, a *actualizador.Actualizador, canal, inv string,
	salida, errores io.Writer) int {

	actual, err := a.VersionInstalada()
	if err != nil {
		fmt.Fprintln(errores, "error:", err)
		return 1
	}
	fmt.Fprintf(salida, "\n  version instalada   %s\n", oNinguna(actual))
	if canal == "" {
		fmt.Fprintf(salida, "\n  No has dicho de donde salen las versiones. Anade --canal con el\n")
		fmt.Fprintf(salida, "  directorio del canal para consultar si hay una nueva.\n\n")
		return 2
	}
	nueva, notas, err := a.Disponible(ctx)
	if err != nil {
		fmt.Fprintln(errores, "error:", err)
		return 1
	}
	if nueva == "" {
		fmt.Fprintf(salida, "\n  No hay version nueva en el canal. Nada que hacer.\n\n")
		return 0
	}
	fmt.Fprintf(salida, "  version disponible  %s\n\n", nueva)
	fmt.Fprintf(salida, "  QUE CAMBIA\n\n")
	for _, l := range dividirLineas(notas) {
		fmt.Fprintf(salida, "    %s\n", l)
	}
	fmt.Fprintf(salida, "\n  Para aplicarla, dejando punto de retorno:\n\n")
	fmt.Fprintf(salida, "    %s --canal %s --aplicar\n\n", inv, canal)
	fmt.Fprintf(salida, "  Si algo va mal despues, `%s --puntos` lista los puntos de\n", inv)
	fmt.Fprintf(salida, "  retorno y `%s --deshacer PUNTO` vuelve al que digas.\n\n", inv)
	return 0
}

func aplicarVersion(ctx context.Context, a *actualizador.Actualizador, version, canal, inv string,
	salida, errores io.Writer) int {

	if canal == "" {
		fmt.Fprintln(errores, "error: falta --canal. Aplicar una version exige decir de donde sale; "+
			"un actualizador que se inventa el origen es un actualizador que instala cualquier cosa")
		return 2
	}
	if version == "" {
		nueva, _, err := a.Disponible(ctx)
		if err != nil {
			fmt.Fprintln(errores, "error:", err)
			return 1
		}
		if nueva == "" {
			fmt.Fprintln(salida, "Ya tienes la ultima version del canal. Nada que aplicar.")
			return 0
		}
		version = nueva
	}
	antes, _ := a.VersionInstalada()
	punto, err := a.Aplicar(ctx, version)
	if err != nil {
		fmt.Fprintln(errores, "error:", err)
		if errors.Is(err, actualizador.ErrAMedias) || errors.Is(err, actualizador.ErrOcupado) {
			return 1
		}
		return 1
	}
	if antes == "" {
		fmt.Fprintf(salida, "\n  Instalada la version %s. Antes no habia fichero VERSION.\n", version)
	} else {
		fmt.Fprintf(salida, "\n  Instalada la version %s, que antes era la %s.\n", version, antes)
	}
	fmt.Fprintf(salida, "  Punto de retorno %s, comprobado fichero a fichero.\n\n", punto)
	fmt.Fprintf(salida, "  Si algo no va como esperabas:\n\n")
	fmt.Fprintf(salida, "    %s --deshacer %s\n\n", inv, punto)
	fmt.Fprintf(salida, "  Y antes de arrancar nada, `plazum doctor` dice si esta maquina puede.\n\n")
	return 0
}

func listarPuntos(a *actualizador.Actualizador, inv string, salida, errores io.Writer) int {
	ps, err := a.Puntos()
	if err != nil {
		fmt.Fprintln(errores, "error:", err)
		return 1
	}
	if len(ps) == 0 {
		fmt.Fprintln(salida, "\n  No hay puntos de retorno: esta instalacion no se ha actualizado")
		fmt.Fprintln(salida, "  nunca con `plazum update`.")
		fmt.Fprintln(salida)
		return 0
	}
	fmt.Fprintf(salida, "\n  %-34s %-22s %s\n", "punto", "fecha", "cambio")
	for _, m := range ps {
		fmt.Fprintf(salida, "  %-34s %-22s de %s a %s\n", m.Punto, m.Fecha, oNinguna(m.Desde), m.Hacia)
	}
	fmt.Fprintf(salida, "\n  Para volver a uno: %s --deshacer %s\n\n", inv, ps[0].Punto)
	return 0
}

func dividirLineas(s string) []string {
	if s == "" {
		return []string{"(el canal no publica notas para esta version)"}
	}
	var out []string
	for _, l := range splitCRLF(s) {
		out = append(out, l)
	}
	return out
}

func splitCRLF(s string) []string {
	var out []string
	inicio := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			l := s[inicio:i]
			if len(l) > 0 && l[len(l)-1] == '\r' {
				l = l[:len(l)-1]
			}
			out = append(out, l)
			inicio = i + 1
		}
	}
	return append(out, s[inicio:])
}
