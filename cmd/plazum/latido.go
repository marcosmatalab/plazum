package main

// `plazum latido`: el pulso de vida, y el aviso de que el planificador esta
// muerto.
//
// LA ORDEN QUE DE VERDAD IMPORTA ES LA QUE NO MANDA NADA. `plazum latido` a
// secas no sale a la red: lee las marcas de la instalacion, le pregunta el
// veredicto a nucleo/pantalla y TERMINA CON CODIGO 1 si el planificador lleva
// mas de 24 horas callado. Eso es lo que se engancha a un cron, a un timer de
// systemd o al monitor que el cliente ya tiene, y por eso el aviso no depende
// de que nosotros estemos vivos. Si dependiera, nuestra caida se leeria como su
// caida y en dos semanas nadie miraria el aviso.
//
// El pulso (opt-in, apagado de fabrica) es la capa de encima, para el caso en
// que la maquina entera muera y no quede nadie que corra el cron. Se enciende
// en dos pasos, como `plazum update`: primero se ensena lo que se manda, y solo
// con --acepto se enciende. Un consentimiento que se da antes de leer lo que se
// acepta no es un consentimiento.

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"plazum/adaptadores/catalogo"
	"plazum/adaptadores/latido"
	"plazum/adaptadores/secretos"
	"plazum/nucleo/pantalla"
	"plazum/puertos"
)

func cmdLatido(args []string, salida, errores io.Writer) int {
	// El subcomando va delante de las opciones, que es como lo teclea la
	// gente. flag.Parse para en el primer argumento que no es una opcion, asi
	// que se saca antes.
	orden := "estado"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		orden, args = args[0], args[1:]
	}

	fs := flag.NewFlagSet("plazum latido", flag.ContinueOnError)
	fs.SetOutput(errores)
	datos := fs.String("datos", ".", "directorio de datos de la instalacion")
	destino := fs.String("destino", "", "a donde va el pulso; vacio es "+latido.DestinoPorDefecto)
	acepto := fs.Bool("acepto", false, "acepta lo que se manda y enciende el pulso (sin esto, activar solo lo ensena)")
	idioma := fs.String("idioma", catalogo.PorDefecto, "idioma de los mensajes")
	ahoraTxt := fs.String("ahora", "", "instante desde el que se juzga (RFC3339); vacio es el reloj del sistema")
	fs.Usage = func() {
		fmt.Fprintln(errores, "uso: plazum latido                     como esta el planificador; codigo 1 si lleva 24 h callado")
		fmt.Fprintln(errores, "     plazum latido ciclo               apunta que el planificador ha corrido (esto es lo que va al cron)")
		fmt.Fprintln(errores, "     plazum latido activar             ensena que se mandaria, y no manda nada")
		fmt.Fprintln(errores, "     plazum latido activar --acepto    enciende el pulso y registra el consentimiento")
		fmt.Fprintln(errores, "     plazum latido desactivar          lo apaga y borra el identificador")
		fmt.Fprintln(errores, "     plazum latido probar              smoke test del canal, sin esperar al ciclo")
		fmt.Fprintln(errores, "")
		fmt.Fprintln(errores, "El aviso de que tu planificador lleva 24 h callado se calcula con TU reloj y")
		fmt.Fprintln(errores, "sin salir a la red, asi que no depende de que nosotros estemos vivos.")
		fmt.Fprintln(errores, "")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	ahora := time.Now().UTC()
	if *ahoraTxt != "" {
		t, err := time.Parse(time.RFC3339, *ahoraTxt)
		if err != nil {
			fmt.Fprintf(errores, "error: --ahora %q no es una fecha RFC3339: %v\n", *ahoraTxt, err)
			return 2
		}
		ahora = t.UTC()
	}

	cat, err := catalogo.Nuevo()
	if err != nil {
		fmt.Fprintln(errores, "error: no puedo cargar el catalogo de mensajes:", err)
		return 1
	}

	switch orden {
	case "estado":
		return latidoEstado(*datos, *idioma, ahora, cat, salida, errores)
	case "ciclo":
		return latidoCiclo(*datos, *idioma, ahora, cat, salida, errores)
	case "activar":
		return latidoActivar(*datos, *destino, *acepto, ahora, salida, errores)
	case "desactivar":
		return latidoDesactivar(*datos, salida, errores)
	case "probar":
		return latidoProbar(*datos, *idioma, ahora, cat, salida, errores)
	}
	fmt.Fprintf(errores, "error: `plazum latido %s` no existe.\n", orden)
	fs.Usage()
	return 2
}

// latidoEstado imprime el veredicto y devuelve el codigo de salida.
//
// EL CODIGO ES LA PIEZA. Un mensaje bonito en la terminal no despierta a nadie
// a las tres de la manana; un codigo 1 en un cron, si.
func latidoEstado(datos, idioma string, ahora time.Time, cat puertos.Catalogo,
	salida, errores io.Writer) int {

	e, err := latido.Cargar(datos)
	if err != nil {
		fmt.Fprintln(errores, "error:", err)
		return 2
	}
	v := pantalla.Vigilar(e.Marcas(), ahora)
	imprimirVigilancia(salida, cat, idioma, v)
	if v.Nivel == pantalla.NivelRoto {
		return 1
	}
	return 0
}

// imprimirVigilancia escribe el veredicto con las MISMAS cadenas que ensena la
// pantalla Hoy.
//
// Se tira del catalogo y no de un texto escrito aqui a proposito: dos copias de
// la misma frase acaban diciendo cosas distintas, y entonces el operador que
// lee la terminal y el que mira la pantalla discuten sobre cual de los dos
// tiene razon.
func imprimirVigilancia(w io.Writer, cat puertos.Catalogo, idioma string, v pantalla.Planificador) {
	fmt.Fprintf(w, "  planificador  %s\n", strings.ToUpper(v.Nivel.String()))
	fmt.Fprintf(w, "    %s\n", cat.Traducir(idioma, v.Clave, v.Horas))
	if v.Arreglo != "" {
		fmt.Fprintf(w, "    %s\n", cat.Traducir(idioma, v.Arreglo, v.UmbralHoras))
	}
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  pulso         %s\n", strings.ToUpper(v.Canal.Nivel.String()))
	fmt.Fprintf(w, "    %s\n", cat.Traducir(idioma, v.Canal.Clave, v.Canal.Horas))
	if v.Canal.Arreglo != "" {
		fmt.Fprintf(w, "    %s\n", cat.Traducir(idioma, v.Canal.Arreglo))
	}
	fmt.Fprintf(w, "    %s\n", cat.Traducir(idioma, v.Canal.Descargo))
}

// latidoCiclo apunta que el planificador ha corrido, y pulsa si esta encendido.
//
// SALE CON 0 AUNQUE EL PULSO FALLE, y eso es la decision de esta funcion. Si un
// fallo de nuestro canal pusiera este comando en rojo, el cron del cliente le
// mandaria un correo diciendo que plazum falla cada vez que a nosotros se nos
// cae el receptor. Nuestra caida no se puede convertir en su alarma.
func latidoCiclo(datos, idioma string, ahora time.Time, cat puertos.Catalogo,
	salida, errores io.Writer) int {

	e, err := latido.Ciclo(context.Background(), datos, latido.CanalHTTP{}, ahora)
	if err != nil && !errors.Is(err, latido.ErrCanal) {
		fmt.Fprintln(errores, "error:", err)
		return 1
	}
	if errors.Is(err, latido.ErrCanal) {
		// A la salida de errores, porque es un aviso, y con codigo 0.
		fmt.Fprintln(errores, "aviso: el ciclo ha quedado apuntado y el pulso no ha salido.")
		fmt.Fprintln(errores, "  ", err)
	}
	fmt.Fprintf(salida, "ciclo apuntado: %s\n\n", e.UltimoCiclo.Format(time.RFC3339))
	imprimirVigilancia(salida, cat, idioma, pantalla.Vigilar(e.Marcas(), ahora))
	return 0
}

// latidoActivar enciende el pulso, en dos pasos.
//
// Sin --acepto se ensena lo que se mandaria y NO se toca nada. Es la misma
// forma que `plazum update`, y por la misma razon: lo que no se puede deshacer
// (aqui, empezar a mandar datos) no se hace con un comando que se teclea de
// carrerilla.
func latidoActivar(datos, destino string, acepto bool, ahora time.Time,
	salida, errores io.Writer) int {

	e, err := latido.Cargar(datos)
	if err != nil && !errors.Is(err, latido.ErrSinConsentimiento) {
		fmt.Fprintln(errores, "error:", err)
		return 2
	}
	aDonde := destino
	if aDonde == "" {
		aDonde = e.DestinoEfectivo()
	}
	if err := latido.ComprobarDestino(aDonde); err != nil {
		fmt.Fprintln(errores, "error:", err)
		return 2
	}

	fmt.Fprintln(salida, "El pulso de vida esta APAGADO de fabrica. Esto es lo que se mandaria,")
	fmt.Fprintln(salida, "una vez al dia, si lo enciendes:")
	fmt.Fprintln(salida, "")
	for _, l := range strings.Split(latido.QueSeManda, "\n") {
		fmt.Fprintln(salida, "   ", l)
	}
	fmt.Fprintln(salida, "")
	fmt.Fprintln(salida, "    destino:", aDonde)
	fmt.Fprintln(salida, "")
	fmt.Fprintln(salida, "Para que te sirva a TI: el aviso de que tu planificador lleva 24 horas")
	fmt.Fprintln(salida, "callado NO necesita esto. Se calcula con tu reloj, sin salir a la red, y")
	fmt.Fprintln(salida, "lo da `plazum latido` con codigo de salida 1. El pulso solo anade el caso")
	fmt.Fprintln(salida, "en que la maquina entera muera, y para que te avise a ti tienes que")
	fmt.Fprintln(salida, "apuntarlo con --destino a tu propio monitor: nosotros no tenemos tu")
	fmt.Fprintln(salida, "correo, a proposito, asi que no podemos avisarte.")
	fmt.Fprintln(salida, "")

	if !acepto {
		fmt.Fprintln(salida, "No se ha activado nada y no ha salido nada de esta maquina.")
		fmt.Fprintln(salida, "Si lo quieres encendido:")
		fmt.Fprintln(salida, "")
		if destino != "" {
			fmt.Fprintf(salida, "    plazum latido activar --destino %s --acepto\n", destino)
		} else {
			fmt.Fprintln(salida, "    plazum latido activar --acepto")
		}
		return 0
	}

	e, err = latido.Activar(datos, destino, ahora, secretos.Nuevo())
	if err != nil {
		fmt.Fprintln(errores, "error:", err)
		return 1
	}
	fmt.Fprintln(salida, "Pulso ENCENDIDO. Consentimiento registrado el",
		e.Consentimiento.Otorgado.Format(time.RFC3339))
	fmt.Fprintln(salida, "    identificador de esta instalacion:", e.Instancia)
	fmt.Fprintln(salida, "    se apaga con: plazum latido desactivar")
	return 0
}

// latidoDesactivar apaga el pulso y borra el identificador.
func latidoDesactivar(datos string, salida, errores io.Writer) int {
	if _, err := latido.Desactivar(datos); err != nil {
		fmt.Fprintln(errores, "error:", err)
		return 1
	}
	fmt.Fprintln(salida, "Pulso APAGADO. Se ha borrado el identificador de esta instalacion,")
	fmt.Fprintln(salida, "asi que si algun dia vuelves a encenderlo sera con uno nuevo.")
	fmt.Fprintln(salida, "")
	fmt.Fprintln(salida, "El aviso de que tu planificador lleva 24 horas callado sigue funcionando:")
	fmt.Fprintln(salida, "no dependia de esto.")
	return 0
}

// latidoProbar es el smoke test del canal.
//
// Aqui SI se sale con 1 cuando el canal falla, al reves que en `ciclo`: el
// operador ha preguntado por el canal, asi que la respuesta es sobre el canal.
func latidoProbar(datos, idioma string, ahora time.Time, cat puertos.Catalogo,
	salida, errores io.Writer) int {

	e, err := latido.Probar(context.Background(), datos, latido.CanalHTTP{}, ahora)
	if errors.Is(err, latido.ErrApagado) {
		fmt.Fprintln(errores, "el pulso esta apagado, asi que no hay canal que probar.")
		fmt.Fprintln(errores, "Se enciende con: plazum latido activar")
		return 2
	}
	if err != nil {
		fmt.Fprintln(errores, "el canal NO entrega:")
		fmt.Fprintln(errores, "  ", err)
		fmt.Fprintln(errores, "")
		fmt.Fprintln(errores, "Esto no toca a tus plazos: el aviso de las 24 horas se calcula con tu")
		fmt.Fprintln(errores, "reloj y sin salir a la red.")
		return 1
	}
	fmt.Fprintf(salida, "el canal entrega. Pulso aceptado por %s a las %s\n\n",
		e.DestinoEfectivo(), e.UltimoPulso.Format(time.RFC3339))
	imprimirVigilancia(salida, cat, idioma, pantalla.Vigilar(e.Marcas(), ahora))
	return 0
}
