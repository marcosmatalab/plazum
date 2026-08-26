package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"plazum/adaptadores/catalogo"
	"plazum/adaptadores/latido"
	"plazum/nucleo/corpus"
	"plazum/nucleo/pantalla"
	"plazum/superficies/pantallas"
	"plazum/superficies/serve"
)

// plazum serve: el cableado de las dos superficies.
//
// Se construyeron en frentes distintos y contra los mismos puertos congelados, a
// proposito: superficies/serve no conoce las pantallas y superficies/pantallas
// no conoce el servidor. Esa independencia es lo que permite sustituir cualquiera
// de las dos sin tocar la otra, y el precio es que alguien tiene que juntarlas.
// Ese alguien es este fichero, y es el unico sitio del producto donde las dos se
// nombran a la vez.
//
// Por que existe como orden y no solo como codigo. Hasta ahora la superficie web
// estaba entera y NO SE PODIA ARRANCAR: un comprador que descargaba el binario
// tenia el servidor, tenia las pantallas, tenia el catalogo, y ninguna forma de
// levantarlos. Un producto que no se puede arrancar no esta hecho, por muchos
// tests que tenga cada mitad.
//
// Lo que NO hace todavia, dicho aqui para que no se busque: no hay almacen de
// usuarios, asi que la autenticacion por usuario y contrasena no existe y el
// unico camino de entrada es el token de primer administrador. Y no hay
// expediente, asi que las pantallas que salen del estado (Hoy, Personas, Estado)
// se pintan vacias diciendo por que.

const ayudaServe = `plazum serve: levanta la interfaz web sobre el corpus instalado.

  plazum serve [--direccion 127.0.0.1:8443] [--corpus paquetes] [--tls-cert F --tls-clave F]

  --direccion   donde escuchar. Por defecto 127.0.0.1:8443, o sea SOLO esta
                maquina. Para abrirlo a la red hace falta decirlo (:8443), y
                entonces lee docs/tls.md antes.
  --corpus      directorio de paquetes de corpus. Por defecto "paquetes".
  --datos       directorio de datos de la instalacion. Por defecto ".". De ahi
                sale el estado del planificador que ensena la pantalla Hoy, que
                es lo que escribe la orden plazum latido ciclo.
  --idioma      idioma de la interfaz. Por defecto el primero del catalogo.
  --tls-cert
  --tls-clave   certificado y clave en PEM. Sin ellos sirve por http, que solo
                vale en local: por http la cookie de sesion no puede ir marcada
                como segura, y un navegador se la queda sin devolverla.
`

// corpusDelDemo es donde `plazum demo` deja el corpus de ejemplo. Sale de
// DirDemoPorDefecto y no de una cadena escrita aqui: dos copias del mismo
// directorio se separan el dia que una cambie.
var corpusDelDemo = filepath.Join(DirDemoPorDefecto, "paquetes")

func cmdServe(args []string, salida, errsal io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(errsal)
	fs.Usage = func() { fmt.Fprint(errsal, ayudaServe) }
	direccion := fs.String("direccion", "127.0.0.1:8443", "donde escuchar")
	dirCorpus := fs.String("corpus", "paquetes", "directorio de paquetes")
	datos := fs.String("datos", ".", "directorio de datos de la instalacion")
	idioma := fs.String("idioma", "", "idioma de la interfaz")
	cert := fs.String("tls-cert", "", "certificado PEM")
	clave := fs.String("tls-clave", "", "clave PEM")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			// --help NO es un fallo. Un script de instalacion que mira el
			// codigo de salida se creeria que ha reventado.
			return 0
		}
		return 2
	}

	ps, err := corpus.Cargar(*dirCorpus)
	if err != nil {
		fmt.Fprintf(errsal, "no se puede cargar el corpus de %q: %v\n", *dirCorpus, err)
		// El caso que de verdad pasa: alguien acaba de ejecutar `plazum demo`,
		// lee la lista de ordenes, teclea `plazum serve` y se estrella, porque
		// el demo deja su corpus en plazum-demo/paquetes y serve mira en
		// paquetes. Decirle ahi "ejecuta plazum demo" es mandarle a repetir lo
		// que acaba de hacer. Si el corpus del demo esta delante, se dice el
		// comando exacto; no se coge solo, que adivinar directorios en silencio
		// es peor que fallar.
		if _, e := os.Stat(corpusDelDemo); e == nil {
			fmt.Fprintf(errsal, "Aqui hay un corpus del demo. Arreglo:\n"+
				"      plazum serve --corpus %s\n", corpusDelDemo)
			return 1
		}
		fmt.Fprintf(errsal, "Arreglo: pasa --corpus con el directorio donde estan los paquetes, o ejecuta\n"+
			"`plazum demo` para ver el producto con un corpus de ejemplo.\n")
		return 1
	}
	if len(ps) == 0 {
		// Arrancar igual seria peor: la interfaz sale entera y vacia, y el
		// operador no sabe si le falta corpus o le falta estado.
		fmt.Fprintf(errsal, "el directorio %q no tiene ningun paquete de corpus.\n"+
			"Arreglo: instala al menos uno, o ejecuta `plazum demo` para ver el producto\n"+
			"con un corpus de ejemplo.\n", *dirCorpus)
		return 1
	}

	cat, err := catalogo.Nuevo()
	if err != nil {
		fmt.Fprintln(errsal, "el catalogo de la interfaz no carga:", err)
		return 1
	}
	if *idioma != "" && !tiene(cat.Idiomas(), *idioma) {
		fmt.Fprintf(errsal, "el idioma %q no esta cargado. Los que hay son %v.\n",
			*idioma, cat.Idiomas())
		return 2
	}

	// El estado del planificador que ensena Hoy se LEE EN CADA PETICION, del
	// fichero que escribe `plazum latido ciclo`. Leerlo aqui una vez seria
	// contar lo que pasaba cuando arranco el servidor: un servidor que lleva
	// tres semanas levantado diria "late" para siempre, que es exactamente la
	// mentira que el vigilante existe para no contar.
	//
	// Si el fichero no esta, las marcas salen en cero y Hoy dice que el
	// planificador no ha corrido ningun ciclo, que es la verdad.
	marcas := func() pantalla.Marcas {
		e, err := latido.Cargar(*datos)
		if err != nil {
			// Un estado ilegible NO se convierte en "todo va bien": las
			// marcas en cero hacen que Hoy diga que no ha corrido nada.
			fmt.Fprintln(errsal, "aviso: no puedo leer el estado del latido:", err)
			return pantalla.Marcas{}
		}
		return e.Marcas()
	}

	app, err := pantallas.Nuevo(pantallas.Opciones{
		Paquetes: ps, Catalogo: cat, Marcas: marcas,
	})
	if err != nil {
		fmt.Fprintln(errsal, "no se pueden construir las pantallas:", err)
		return 1
	}

	// La cookie solo puede ir sin la marca de segura cuando de verdad no hay
	// TLS Y se escucha en local. Decidirlo aqui y no dentro del servidor es a
	// proposito: es una decision de despliegue, y el servidor tiene que poder
	// negarse si alguien la toma mal.
	insegura := *cert == "" && esLocal(*direccion)

	srv, err := serve.Nuevo(serve.Config{
		App:            app,
		Estaticos:      nil, // las pantallas sirven los suyos bajo /estatico/
		CertificadoTLS: *cert,
		ClaveTLS:       *clave,
		CookieInsegura: insegura,
		Salida:         salida,
	})
	if err != nil {
		fmt.Fprintln(errsal, "no se puede construir el servidor:", err)
		return 1
	}

	if *cert == "" && !esLocal(*direccion) {
		fmt.Fprintf(errsal, "AVISO: vas a servir en %s SIN TLS.\n"+
			"  Por http la cookie de sesion no puede ir marcada como segura, y ademas todo\n"+
			"  viaja en claro. Si tienes un proxy con TLS delante, esto es correcto.\n"+
			"  Si no lo tienes, lee docs/tls.md antes de dejarlo asi.\n", *direccion)
	}

	// Ctrl+C y SIGTERM cierran en orden. Sin esto, parar el servicio corta las
	// peticiones en vuelo, y una de ellas puede ser la que estaba escribiendo
	// en el ledger.
	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parar()

	if err := srv.Arrancar(ctx, *direccion); err != nil {
		fmt.Fprintln(errsal, "el servidor no ha podido arrancar:", err)
		return 1
	}
	// Cierre ordenado con su propio plazo: el contexto de arriba ya esta
	// cancelado cuando llegamos aqui, asi que reusarlo cerraria de golpe.
	cierre, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	if err := srv.Parar(cierre); err != nil {
		fmt.Fprintln(errsal, "el cierre no ha terminado del todo:", err)
		return 1
	}
	return 0
}

// esLocal dice si la direccion solo es alcanzable desde esta maquina. Se usa
// para decidir si se puede servir por http sin mentirle a nadie.
func esLocal(dir string) bool {
	host := dir
	if i := strings.LastIndex(dir, ":"); i > 0 {
		host = dir[:i]
	}
	host = strings.Trim(host, "[]")
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}

func tiene(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// compilacion: que el handler de pantallas siga siendo lo que serve espera.
var _ http.Handler = (*pantallas.Superficie)(nil)
