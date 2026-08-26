package main

// `plazum export`: el log de auditoria hacia el SIEM del cliente, en JSON
// lineas.
//
// La decision de interfaz, y por que importa mas de lo que parece: los EVENTOS
// salen por la salida estandar y NADA MAS sale por ahi. El resumen, los avisos y
// los errores van por el canal de errores. Asi
//
//	plazum export expediente.json | nc -q0 siem.interno 5514
//
// manda eventos y solo eventos, y el operador sigue viendo por pantalla cuantos
// mando. Una sola linea de cortesia mezclada en la tuberia rompe la ingesta del
// receptor, y la rompe en silencio: el SIEM descarta la linea que no parsea y no
// avisa a nadie.

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"plazum/nucleo/expediente"
	"plazum/superficies/export"
)

func cmdExport(e *expediente.Expediente, args []string, salida, errores io.Writer) int {
	fs := flag.NewFlagSet("plazum export", flag.ContinueOnError)
	fs.SetOutput(errores)
	destino := fs.String("salida", "-", "fichero donde escribir; - es la salida estandar")
	fs.Usage = func() {
		fmt.Fprintln(errores, "uso: plazum export <expediente.json> [--salida FICHERO]")
		fmt.Fprintln(errores, "")
		fmt.Fprintln(errores, "Escribe el log de auditoria en JSON lineas: una linea, un evento,")
		fmt.Fprintln(errores, "sin envoltura. Lo traga cualquier SIEM sin transformador.")
		fmt.Fprintln(errores, "")
		fmt.Fprintln(errores, "Lo que un borrado legal borro NO sale: de una entrada con lapida")
		fmt.Fprintln(errores, "sale el hecho de la supresion y su base legal, nunca su contenido.")
		fmt.Fprintln(errores, "")
		// La guia va aqui y no solo en el repositorio: quien teclea esto esta a
		// punto de mandar el rastro de auditoria de su organizacion a un tercero
		// con retencion propia, y ese es el momento de leer que viaja y que no.
		fmt.Fprintln(errores, "Como se conecta Splunk, Elastic, Sentinel o Loki, que campos salen")
		fmt.Fprintln(errores, "y que NO viaja nunca: docs/siem.md")
		fmt.Fprintln(errores, "")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	w := salida
	if *destino != "-" {
		// 0600: este fichero lleva el rastro de auditoria de la organizacion en
		// texto plano. Que lo lea cualquier cuenta de la maquina no es lo que
		// espera quien lo esta mandando a un SIEM con control de acceso.
		f, err := os.OpenFile(*destino, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600) // #nosec G304 -- CLI: la ruta la teclea el operador en su propia maquina
		if err != nil {
			fmt.Fprintf(errores, "no puedo escribir en %s: %v\n", *destino, err)
			fmt.Fprintln(errores, "Arreglo: comprueba el directorio y los permisos, o quita --salida")
			fmt.Fprintln(errores, "para mandarlo por la salida estandar.")
			return 1
		}
		defer func() { _ = f.Close() }()
		w = f
	}

	buf := bufio.NewWriter(w)
	res, err := export.Exportar(buf, e)
	if err != nil {
		fmt.Fprintln(errores, "no se ha podido construir el log de auditoria:", err)
		return 1
	}
	if err := buf.Flush(); err != nil {
		fmt.Fprintln(errores, "el log de auditoria se ha quedado a medias:", err)
		fmt.Fprintln(errores, "Arreglo: no ingieras el fichero, esta incompleto. Vuelve a")
		fmt.Fprintln(errores, "exportarlo y comprueba el espacio en disco o el destino de la tuberia.")
		return 1
	}

	// El resumen por el canal de errores, siempre: un export que sale con 0 y
	// no mando nada es indistinguible de uno que mando todo, y esa es la forma
	// de romperse que este proyecto persigue en todas partes.
	fmt.Fprintln(errores, res.String())
	if res.Eventos == 0 {
		fmt.Fprintln(errores, "AVISO: el expediente no tiene nada que auditar todavia. "+
			"Si esperabas eventos, comprueba que es el expediente que crees.")
	}
	return 0
}
