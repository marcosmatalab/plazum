// Comando dutiq: superficie minima del producto.
//
//	dutiq verify <expediente.json> <contexto-receptor.json>
//	                                 recalcula el expediente entero, sin red, contra
//	                                 las anclas y claves que aporta el RECEPTOR
//	dutiq explain <expediente.json>  imprime la derivacion de cada conclusion
//	dutiq estado  <expediente.json>  los cinco denominadores, nunca un porcentaje
//	dutiq cobertura <dir_paquetes>   la cobertura honesta de cada paquete instalado
//	dutiq demo                       una empresa de ejemplo con sus relojes corriendo
//	dutiq doctor                     por que no funciona, con el arreglo de cada cosa
//	dutiq update                     actualizar con vuelta atras comprobada
//	dutiq serve                      levanta la interfaz web sobre el corpus instalado
package main

import (
	"fmt"
	"os"

	"dutiq/nucleo/corpus"
	"dutiq/nucleo/expediente"
)

func main() {
	// Las ordenes del autoservicio van ANTES de la comprobacion de arriba
	// porque no llevan fichero: `dutiq demo` a secas tiene que funcionar, que
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
		}
	}
	if len(os.Args) < 3 {
		// La primera linea dice por donde empezar, y esta puesta ahi a
		// proposito: quien teclea `dutiq` a secas casi siempre lo acaba de
		// descargar, y una lista de seis ordenes sin punto de entrada es lo
		// mismo que ninguna.
		fmt.Fprintln(os.Stderr, "empieza por aqui:")
		fmt.Fprintln(os.Stderr, "     dutiq demo      una empresa de ejemplo con sus relojes corriendo,")
		fmt.Fprintln(os.Stderr, "                     sin configurar nada y sin red")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "el resto:")
		// serve va el primero del resto porque es la superficie del producto:
		// quien acaba de ver `dutiq demo` en la terminal lo siguiente que
		// quiere es abrirlo en el navegador. Estuvo implementada y sin
		// aparecer aqui, asi que no habia forma de descubrirla salvo leyendo
		// el codigo fuente, y de paso la puerta de accesibilidad de CI, que
		// pregunta por esta lista para saber si hay pantallas que auditar, se
		// quedaba en rojo diciendo que el producto no sabia servirlas.
		fmt.Fprintln(os.Stderr, "     dutiq serve     la interfaz web sobre el corpus instalado")
		fmt.Fprintln(os.Stderr, "     dutiq doctor    por que no funciona, con el arreglo de cada cosa")
		fmt.Fprintln(os.Stderr, "     dutiq update    actualizar con vuelta atras comprobada")
		fmt.Fprintln(os.Stderr, "     dutiq verify    <expediente.json> <contexto-receptor.json>")
		fmt.Fprintln(os.Stderr, "     dutiq explain   <expediente.json>")
		fmt.Fprintln(os.Stderr, "     dutiq estado    <expediente.json>")
		fmt.Fprintln(os.Stderr, "     dutiq cobertura <dir_paquetes>")
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
			fmt.Fprintln(os.Stderr, "uso: dutiq verify <expediente.json> <contexto-receptor.json>")
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
