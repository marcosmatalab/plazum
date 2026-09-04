// Comando plazum: superficie minima del producto.
//
//	plazum verify <expediente.json> <contexto-receptor.json>
//	                                 recalcula el expediente entero, sin red, contra
//	                                 las anclas y claves que aporta el RECEPTOR
//	plazum explain <expediente.json>  imprime la derivacion de cada conclusion
//	plazum estado  <expediente.json>  los cinco denominadores, nunca un porcentaje
//	plazum corpus                     que corpus hay, y si es el que publico este binario
//	plazum cobertura <dir_paquetes>   la cobertura honesta de cada paquete instalado
//	plazum calendario                 las fechas de los proximos doce meses, con su articulo
//	plazum escalado                   que avisos saldrian y a quien, en seco por defecto
//	plazum demo                       una empresa de ejemplo con sus relojes corriendo
//	plazum doctor                     por que no funciona, con el arreglo de cada cosa
//	plazum update                     actualizar con vuelta atras comprobada
//	plazum serve                      levanta la interfaz web sobre el corpus instalado
//	plazum latido                     si el planificador sigue vivo, y el pulso opt-in
package main

import (
	"fmt"
	"os"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/expediente"
)

func main() {
	// Las ordenes del autoservicio van ANTES de la comprobacion de arriba
	// porque no llevan fichero: `plazum demo` a secas tiene que funcionar, que
	// es literalmente su razon de ser. Cada una parsea sus propias opciones con
	// flag y devuelve su codigo de salida.
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "demo":
			os.Exit(cmdDemo(os.Args[2:], os.Stdout, os.Stderr))
		case "doctor":
			os.Exit(cmdDoctor(os.Args[2:], os.Stdout, os.Stderr))
		case "update":
			os.Exit(cmdUpdate(os.Args[2:], os.Stdout, os.Stderr))
		case "serve":
			os.Exit(cmdServe(os.Args[2:], os.Stdout, os.Stderr))
		case "latido":
			os.Exit(cmdLatido(os.Args[2:], os.Stdout, os.Stderr))
		case "calendario":
			os.Exit(cmdCalendario(os.Args[2:], os.Stdout, os.Stderr))
		case "escalado":
			os.Exit(cmdEscalado(os.Args[2:], os.Stdout, os.Stderr))
		case "accesos":
			os.Exit(cmdAccesos(os.Args[2:], os.Stdout, os.Stderr))
		case "incidentes":
			os.Exit(cmdIncidentes(os.Args[2:], os.Stdout, os.Stderr))
		case "auditoria":
			os.Exit(cmdAuditoria(os.Args[2:], os.Stdout, os.Stderr))
		case "alcance":
			os.Exit(cmdAlcance(os.Args[2:], os.Stdout, os.Stderr))
		case "corpus":
			os.Exit(cmdCorpus(os.Args[2:], os.Stdout, os.Stderr))
		}
	}
	if len(os.Args) < 3 {
		// La primera linea dice por donde empezar, y esta puesta ahi a
		// proposito: quien teclea `plazum` a secas casi siempre lo acaba de
		// descargar, y una lista de seis ordenes sin punto de entrada es lo
		// mismo que ninguna.
		fmt.Fprintln(os.Stderr, "empieza por aqui:")
		// `corpus --instalar` VA EL PRIMERO Y NO ES UN DETALLE DE ORDEN. El
		// binario a secas no trae los treinta marcos: viajan al lado, como
		// activo firmado de la release. Quien se baje esto y solo vea `demo`
		// probara un paquete con tres relojes y se ira pensando que plazum no
		// trae nada. La orden que convierte la descarga en el producto va
		// arriba del todo, y dice en la misma linea que es lo real.
		fmt.Fprintln(os.Stderr, "     plazum corpus --instalar plazum-corpus.tar.gz")
		fmt.Fprintln(os.Stderr, "                      los 30 marcos de verdad, comprobados contra la huella")
		fmt.Fprintln(os.Stderr, "                      que este binario lleva dentro. El .tar.gz viene en la")
		fmt.Fprintln(os.Stderr, "                      misma pagina de descarga que este programa")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "     plazum demo      una empresa de ejemplo con sus relojes corriendo,")
		fmt.Fprintln(os.Stderr, "                      sin configurar nada, sin red y sin corpus. Es el paseo")
		fmt.Fprintln(os.Stderr, "                      de dos minutos, no es tu cumplimiento")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "     plazum calendario  las fechas de los proximos doce meses, con su")
		fmt.Fprintln(os.Stderr, "                      articulo. Con --ics te las llevas al Outlook")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "     plazum escalado    que avisos saldrian y a quien. NO manda nada")
		fmt.Fprintln(os.Stderr, "                      salvo que se lo pidas con --mandar")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "el resto:")
		// serve va el primero del resto porque es la superficie del producto:
		// quien acaba de ver `plazum demo` en la terminal lo siguiente que
		// quiere es abrirlo en el navegador. Estuvo implementada y sin
		// aparecer aqui, asi que no habia forma de descubrirla salvo leyendo
		// el codigo fuente, y de paso la puerta de accesibilidad de CI, que
		// pregunta por esta lista para saber si hay pantallas que auditar, se
		// quedaba en rojo diciendo que el producto no sabia servirlas.
		fmt.Fprintln(os.Stderr, "     plazum alcance   convierte las respuestas de la entrevista en el")
		fmt.Fprintln(os.Stderr, "                      alcance.json que piden calendario, escalado y serve")
		fmt.Fprintln(os.Stderr, "     plazum accesos   sube el CSV de cuentas de tu IdP y abre la revision")
		fmt.Fprintln(os.Stderr, "                      de accesos; dice que ha entendido antes de dar un numero")
		fmt.Fprintln(os.Stderr, "     plazum incidentes  el registro de incidentes del que se compone el acta")
		fmt.Fprintln(os.Stderr, "     plazum auditoria   el programa de auditoria interna, con su arrastre")
		fmt.Fprintln(os.Stderr, "                      entre ciclos; la otra fuente del acta")
		fmt.Fprintln(os.Stderr, "     plazum serve     la interfaz web sobre el corpus instalado")
		fmt.Fprintln(os.Stderr, "     plazum corpus    que corpus tienes instalado y si cuadra con el que")
		fmt.Fprintln(os.Stderr, "                      publico este binario")
		fmt.Fprintln(os.Stderr, "     plazum doctor    por que no funciona, con el arreglo de cada cosa")
		fmt.Fprintln(os.Stderr, "     plazum latido    si tu planificador sigue vivo; codigo 1 si lleva 24 h callado")
		fmt.Fprintln(os.Stderr, "     plazum update    actualizar con vuelta atras comprobada")
		fmt.Fprintln(os.Stderr, "     plazum verify    <expediente.json> <contexto-receptor.json>")
		fmt.Fprintln(os.Stderr, "     plazum explain   <expediente.json>")
		fmt.Fprintln(os.Stderr, "     plazum estado    <expediente.json>")
		fmt.Fprintln(os.Stderr, "     plazum export    <expediente.json>   el log de auditoria para tu SIEM, en JSON lineas")
		fmt.Fprintln(os.Stderr, "     plazum cobertura <dir_paquetes>")
		os.Exit(2)
	}
	if os.Args[1] == "cobertura" {
		ps, err := corpus.Cargar(os.Args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		for _, p := range ps {
			fmt.Println(corpus.Medir(p))
		}
		return
	}
	b, err := os.ReadFile(os.Args[2]) // #nosec G304,G703 -- CLI: la ruta la teclea el operador en su propia maquina, no hay frontera de confianza que cruzar
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	e, err := expediente.Cargar(b)
	if err != nil {
		fmt.Fprintln(os.Stderr, "expediente ilegible:", err)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "verify":
		// El contexto del receptor es obligatorio: sin el, verificar no
		// significa nada. Se pide como tercer argumento y se dice por que.
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "uso: plazum verify <expediente.json> <contexto-receptor.json>")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "El contexto lo aportas TU, no el expediente: tus anclas de corpus,")
			fmt.Fprintln(os.Stderr, "las claves publicas que ya conocias y las raices de TSA que aceptas.")
			fmt.Fprintln(os.Stderr, "Verificar un expediente con los datos que trae el propio expediente")
			fmt.Fprintln(os.Stderr, "seria comparar al emisor consigo mismo.")
			os.Exit(2)
		}
		ctx, err := cargarContexto(os.Args[3])
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		for _, a := range avisosDelContexto(ctx) {
			fmt.Fprintln(os.Stderr, "AVISO:", a)
		}
		inf := expediente.Verificar(e, ctx)
		fmt.Printf("expediente de %s, como estaba el %s\n\n", e.Organizacion, e.ComoEstaba.Format("2006-01-02 15:04 -07:00"))
		for _, c := range inf.Comprobaciones {
			fmt.Println("  ok        ", c)
		}
		for _, d := range inf.Discrepancias {
			fmt.Printf("  DISCREPA   %s\n             declarado:   %s\n             recalculado: %s\n", d.Que, d.Esperado, d.Obtenido)
		}
		fmt.Println()
		if inf.Valido {
			fmt.Println("VERIFICADO. Recalculado desde cero sin red y sin confiar en el emisor.")
			return
		}
		fmt.Printf("NO VERIFICA: %d discrepancia(s).\n", len(inf.Discrepancias))
		os.Exit(1)

	case "explain":
		// El cuerpo vive en explain.go, con el descargo de asesoramiento
		// juridico que cierra la salida y su puerta al lado.
		os.Exit(cmdExplain(e, os.Stdout, os.Stderr))

	case "export":
		// El cuerpo vive en export.go. Los eventos salen por stdout y nada mas
		// sale por ahi: el resumen va por stderr para que la tuberia al SIEM
		// lleve solo JSON lineas.
		os.Exit(cmdExport(e, os.Args[3:], os.Stdout, os.Stderr))

	case "estado":
		fmt.Printf("estado de %s\n\n", e.Organizacion)
		for _, s := range e.Estados {
			fmt.Printf("  %-22s %s\n", s.Prueba, s.Estado)
		}
		d := e.Denominadores
		fmt.Printf("\ncinco denominadores separados, sin porcentaje unico:\n")
		fmt.Printf("  verificable por maquina   %d\n  declarado por humano      %d\n"+
			"  atestado por externo      %d\n  desconocido               %d\n"+
			"  caducado o contradicho    %d\n",
			d.Maquina, d.Humano, d.Externo, d.Desconocido, d.CaducadoOContradicho)

	default:
		fmt.Fprintln(os.Stderr, "orden desconocida:", os.Args[1])
		os.Exit(2)
	}
}
