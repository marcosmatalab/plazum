package plazum

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Las puertas de la distribucion: lo que sale de esta maquina y con que forma.
//
// Dos cosas se vigilan aqui, y las dos por el mismo motivo: un error de
// distribucion no se puede deshacer con un commit.
//
//  1. QUE NADA SE PUBLIQUE MIENTRAS LA MARCA ESTE CONGELADA. La firma keyless de
//     cosign sube el certificado al log de Rekor, que es append-only: publica la
//     identidad del repositorio y no se retira. La comprobacion de TMview del
//     25-08-2026 encontro dos marcas de la Union Europea REGISTRADAS con solape
//     literal sobre el nombre (docs/marca.md). Hasta que un agente de la
//     propiedad industrial se pronuncie, nada sale.
//
//  2. QUE LA IMAGEN SIGA SIENDO LA QUE SE REVISO. Un Dockerfile es el fichero
//     mas facil de degradar del repositorio: una etiqueta en vez de un digest,
//     un `FROM ubuntu` "temporal" para depurar, un USER que se cae. Ninguna de
//     esas tres cosas rompe nada al construir, y las tres cambian lo que el
//     comprador ejecuta.

// ---------------------------------------------------------------------------
// 1. El candado de marca
// ---------------------------------------------------------------------------

// ficheroCandado es el que decide. Vive en el repositorio a proposito: un
// secreto o una variable de entorno de GitHub la cambia una persona en un panel
// que no deja rastro en el codigo, y esta decision tiene que verse en un diff.
const ficheroCandado = ".github/marca-congelada"

// marcadoresDePublicacion son las formas que tiene un paso de CI de sacar algo
// de la maquina. La lista es de SUBCADENAS y se busca sobre la linea sin
// comentario.
//
// Cada una esta escrita lo bastante larga para no cazar prosa: "cosign" a secas
// aparece en un nombre de paso, "cosign sign" no aparece mas que cuando se
// firma.
var marcadoresDePublicacion = []struct{ patron, que string }{
	{"docker/login-action", "entrar en un registro de imagenes"},
	{"docker push", "subir una imagen"},
	{"--push", "subir una imagen con buildx"},
	{"docker/build-push-action", "construir y subir una imagen"},
	{"sigstore/cosign-installer", "instalar cosign, que es el paso previo a firmar en Rekor"},
	{"cosign sign", "firmar y publicar el certificado en el log publico de Rekor"},
	{"softprops/action-gh-release", "crear una release de GitHub"},
	{"gh release create", "crear una release de GitHub"},
	{"actions/deploy-pages", "publicar la web"},
	{"npm publish", "publicar en npm"},
}

func leerWorkflows(t *testing.T) map[string]string {
	t.Helper()
	dir := filepath.Join(".github", "workflows")
	entradas, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("no puedo leer %s: %v", dir, err)
	}
	out := map[string]string{}
	for _, e := range entradas {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yml") {
			continue // .disabled no se ejecuta
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		out[e.Name()] = string(b)
	}
	if len(out) < 3 {
		t.Fatalf("solo hay %d workflows activos. Si el directorio se movio, este test "+
			"estaria auditando el vacio y dando verde", len(out))
	}
	return out
}

var (
	// Un trabajo empieza en la columna 2 y un paso en la 6. El recorrido es
	// por indentacion porque el repositorio no tiene (ni va a tener) una
	// biblioteca de YAML: DEPENDENCIAS.md es lista cerrada.
	inicioDeTrabajo = regexp.MustCompile(`^  [A-Za-z0-9_-]+:\s*$`)
	inicioDePaso    = regexp.MustCompile(`^      - `)
	condicionDePaso = regexp.MustCompile(`^        if:`)
	condicionJob    = regexp.MustCompile(`^    if:`)
)

// TestNingunPasoQuePublicaCorreSinPreguntarPorElCandado recorre los workflows y,
// por cada linea que saca algo de la maquina, exige que el paso o su trabajo
// esten condicionados a la salida del candado.
//
// El recorrido es por indentacion y FALLA CERRADO: si un workflow nuevo usa otra
// forma de sangrar, el marcador se queda sin guardia detectado y esto se pone
// rojo. Un falso rojo aqui cuesta cinco minutos; un falso verde cuesta una
// entrada en Rekor que no se borra.
func TestNingunPasoQuePublicaCorreSinPreguntarPorElCandado(t *testing.T) {
	vistos := 0
	for nombre, cuerpo := range leerWorkflows(t) {
		condTrabajo, condPaso := "", ""
		enPaso := false
		for _, linea := range strings.Split(cuerpo, "\n") {
			limpia := strings.TrimSpace(linea)
			if strings.HasPrefix(limpia, "#") {
				continue // un comentario que nombra cosign no firma nada
			}
			switch {
			case inicioDeTrabajo.MatchString(linea):
				condTrabajo, condPaso, enPaso = "", "", false
			case inicioDePaso.MatchString(linea):
				condPaso, enPaso = "", true
			case condicionJob.MatchString(linea) && !enPaso:
				condTrabajo = linea
			case condicionDePaso.MatchString(linea):
				condPaso = linea
			}
			// El comentario de linea no cuenta como codigo.
			codigo := linea
			if i := strings.Index(codigo, " #"); i >= 0 {
				codigo = codigo[:i]
			}
			for _, m := range marcadoresDePublicacion {
				if !strings.Contains(codigo, m.patron) {
					continue
				}
				vistos++
				if guardadoPorElCandado(condPaso) || guardadoPorElCandado(condTrabajo) {
					continue
				}
				t.Errorf(`%s: un paso publica sin preguntar por el candado de marca.
    linea:  %s
    hace:   %s
    if del paso:     %q
    if del trabajo:  %q
  La marca de plazum esta congelada (%s, motivo en docs/marca.md) y la firma keyless
  publica la identidad del repositorio en Rekor, que es un log append-only: no se
  borra, no se negocia y no se retrasa.
  Arreglo: anadir al paso, o a su trabajo,
      if: needs.candado.outputs.publicar == 'si'
  y declarar 'needs: candado'. El trabajo candado lee %s.`,
					nombre, strings.TrimSpace(linea), m.que, strings.TrimSpace(condPaso),
					strings.TrimSpace(condTrabajo), ficheroCandado, ficheroCandado)
			}
		}
	}
	if vistos == 0 {
		t.Fatal("no se ha encontrado NI UNA forma de publicar en los workflows. O la release " +
			"ha desaparecido, o los marcadores de este test han dejado de reconocerla, y las " +
			"dos cosas dejan la puerta sin nada que vigilar")
	}
	if !t.Failed() {
		t.Logf("%d pasos de publicacion, todos detras del candado", vistos)
	}
}

// TestNingunPasoSeRompeCuandoElCandadoSeQuita.
//
// EL FALLO QUE LA TRAE. El 26-08-2026 se borro .github/marca-congelada, que es
// como el propio fichero decia que habia que abrirlo. etapa2-distribucion.yml se
// puso rojo en el acto: tenia un paso que hacia `cat .github/marca-congelada` a
// secas para imprimir el motivo, y `cat` sobre un fichero que no existe sale con
// 1. Los pasos `bash` de GitHub corren con -e, asi que el paso murio.
//
// Lo de fondo no es el `cat`. Es que ese paso daba por hecho que el candado
// existiria SIEMPRE, cuando el candado esta puesto precisamente para quitarse
// algun dia. Un fichero cuya desaparicion esta prevista no se puede leer como si
// fuera permanente.
//
// Y lo caro es que el rojo NO lo caza ningun test: lo caza mirar CI despues de
// empujar, que es una costumbre y no una puerta. En este repositorio ya hubo un
// bloqueante rojo cinco commits seguidos sin que nadie lo leyera.
//
// La regla: un `run:` que nombre el candado, o esta bajo un `if:` que ya decidio
// que el candado esta, o comprueba el mismo que existe.
func TestNingunPasoSeRompeCuandoElCandadoSeQuita(t *testing.T) {
	vistos := 0
	for nombre, cuerpo := range leerWorkflows(t) {
		condTrabajo, condPaso := "", ""
		enPaso := false
		var lineasDelPaso []string
		nombrado := false

		// cerrar juzga el paso que acaba de terminar.
		cerrar := func() {
			if !nombrado {
				return
			}
			vistos++
			bloque := strings.Join(lineasDelPaso, "\n")
			// Se comprueba a si mismo: cualquier forma de preguntar si el
			// fichero esta antes de leerlo.
			seComprueba := strings.Contains(bloque, "[ -f "+ficheroCandado+" ]") ||
				strings.Contains(bloque, "[ ! -f "+ficheroCandado+" ]") ||
				strings.Contains(bloque, "test -f "+ficheroCandado)
			if seComprueba || guardadoPorElCandado(condPaso) || guardadoPorElCandado(condTrabajo) {
				return
			}
			t.Errorf(`%s: un paso lee %s sin comprobar que exista y sin estar bajo un if.

  El candado esta puesto PARA QUITARSE. El dia que se quite, este paso se pone
  rojo: los pasos `+"`bash`"+` de GitHub corren con -e y un `+"`cat`"+` de un fichero que no
  existe sale con 1. Ya paso el 26-08-2026 con etapa2-distribucion.yml.

  Y un rojo permanente es tan invisible como un verde falso: nadie mira un job
  que lleva semanas rojo, y un job que lleva semanas rojo no mide nada.

  Arreglo, uno de los dos:
    a) envolverlo en `+"`if [ -f "+ficheroCandado+" ]; then ... else ... fi`"+`,
       contando las DOS historias, o
    b) poner el paso o su trabajo bajo un if que ya haya decidido que el
       candado esta (if: needs.candado.outputs.publicar != 'si').

  paso:
%s`, nombre, ficheroCandado, bloque)
		}

		for _, linea := range strings.Split(cuerpo, "\n") {
			switch {
			case inicioDeTrabajo.MatchString(linea):
				cerrar()
				condTrabajo, condPaso, enPaso, nombrado = "", "", false, false
				lineasDelPaso = nil
			case inicioDePaso.MatchString(linea):
				cerrar()
				condPaso, enPaso, nombrado = "", true, false
				lineasDelPaso = nil
			case condicionJob.MatchString(linea) && !enPaso:
				condTrabajo = linea
			case condicionDePaso.MatchString(linea):
				condPaso = linea
			}
			if !enPaso {
				continue
			}
			lineasDelPaso = append(lineasDelPaso, linea)
			codigo := linea
			if i := strings.Index(codigo, " #"); i >= 0 {
				codigo = codigo[:i]
			}
			if strings.TrimSpace(codigo) != "" && !strings.HasPrefix(strings.TrimSpace(codigo), "#") &&
				strings.Contains(codigo, ficheroCandado) {
				nombrado = true
			}
		}
		cerrar()
	}
	// Suelo: si nadie nombra el candado, esta puerta no vigila nada y el
	// mecanismo entero se ha ido sin que se note.
	if vistos == 0 {
		t.Fatal("ningun paso de ningun workflow nombra el candado de marca. O el " +
			"mecanismo ha desaparecido, o este recorrido ha dejado de encontrarlo: las " +
			"dos cosas dejan la puerta sin nada que vigilar")
	}
	if !t.Failed() {
		t.Logf("%d pasos leen el candado, todos preparados para que no este", vistos)
	}
}

func guardadoPorElCandado(cond string) bool {
	return strings.Contains(cond, "candado.outputs.publicar")
}

// El candado se lee de un FICHERO del repositorio y no de un ajuste de la
// plataforma.
//
// Por que importa la diferencia: un secreto o una variable de repositorio los
// cambia una persona sola, en un panel, sin revision y sin dejar rastro en el
// historico. Borrar un fichero deja un commit de una linea con autor y fecha, y
// esa es exactamente la traza que hace falta el dia que alguien pregunte quien
// decidio publicar.
func TestElCandadoDeMarcaSeDecideEnUnFicheroDelRepositorio(t *testing.T) {
	release, hay := leerWorkflows(t)["release.yml"]
	if !hay {
		t.Fatal("no hay .github/workflows/release.yml. Si el workflow de release se ha " +
			"renombrado, este test esta vigilando un fichero que ya no existe")
	}
	if !strings.Contains(release, ficheroCandado) {
		t.Errorf("release.yml no nombra %s. El candado tiene que salir de un fichero del "+
			"repositorio: un ajuste de la plataforma lo cambia una persona en un panel y no "+
			"queda en el historico", ficheroCandado)
	}
	if !strings.Contains(release, "publicar=no") || !strings.Contains(release, "publicar=si") {
		t.Error("el trabajo candado de release.yml no publica las dos salidas (publicar=si y " +
			"publicar=no). Un candado que solo sabe decir que si no es un candado")
	}
	// Y el estado de hoy, dicho en el registro del test para que quede en la
	// salida de CI: si el fichero desaparece, la siguiente ejecucion de la
	// release publica de verdad.
	if _, err := os.Stat(ficheroCandado); err == nil {
		t.Logf("la marca sigue congelada: %s existe, hoy no se publica nada", ficheroCandado)
	} else {
		t.Logf("ATENCION: %s NO existe. El candado esta abierto y la proxima release "+
			"publicara en Rekor de forma irreversible", ficheroCandado)
	}
}

// ---------------------------------------------------------------------------
// 2. La imagen
// ---------------------------------------------------------------------------

func leerDockerfile(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("Dockerfile")
	if err != nil {
		t.Fatalf("no hay Dockerfile: %v.\nLa imagen es la unica forma de arrancar plazum sin "+
			"instalar Go, o sea la unica que un CISO va a probar el primer dia", err)
	}
	return string(b)
}

// lineasDeCodigo devuelve las instrucciones del Dockerfile, sin comentarios: el
// fichero lleva mas comentario que codigo a proposito, y un test que busque
// "FROM scratch" en la prosa da verde sobre un Dockerfile que no lo hace.
func lineasDeCodigo(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		out = append(out, t)
	}
	return out
}

// La imagen final no lleva sistema operativo dentro.
//
// No es por el tamano. Es que sin interprete de ordenes y sin gestor de
// paquetes, una ejecucion remota de codigo se queda sin nada que ejecutar, y un
// escaner de vulnerabilidades sobre la imagen habla de plazum y no de la lista de
// CVEs de una distribucion entera que nadie de aqui parchea.
func TestLaImagenFinalNoLlevaUnSistemaOperativoDentro(t *testing.T) {
	lineas := lineasDeCodigo(leerDockerfile(t))
	var ultimoFrom string
	etapas := 0
	for _, l := range lineas {
		if strings.HasPrefix(l, "FROM ") {
			ultimoFrom = l
			etapas++
		}
	}
	if etapas < 2 {
		t.Errorf("el Dockerfile tiene %d etapa(s). Sin etapa de construccion separada, el "+
			"compilador de Go, el codigo fuente y la cache de modulos viajan dentro de la "+
			"imagen que se publica", etapas)
	}
	campos := strings.Fields(ultimoFrom)
	if len(campos) < 2 {
		t.Fatalf("no entiendo el ultimo FROM: %q", ultimoFrom)
	}
	base := campos[1]
	if base != "scratch" && !strings.Contains(base, "distroless") {
		t.Errorf(`la imagen final se construye sobre %q y tiene que ser scratch o distroless.
  Con una distribucion dentro, la imagen hereda su shell, su gestor de paquetes y su
  lista de CVEs, y ninguna de las tres cosas la parchea nadie de este proyecto.
  Si hacia falta un shell para depurar, se depura con una etiqueta aparte y no aqui.`, base)
	}
}

// El proceso no corre como root.
func TestLaImagenNoCorreComoRoot(t *testing.T) {
	lineas := lineasDeCodigo(leerDockerfile(t))
	usuario := ""
	for _, l := range lineas {
		if strings.HasPrefix(l, "USER ") {
			usuario = strings.TrimSpace(strings.TrimPrefix(l, "USER "))
		}
	}
	switch {
	case usuario == "":
		t.Error("el Dockerfile no lleva USER, asi que el contenedor corre como root. Si algo " +
			"se escapa del proceso, se escapa con uid 0. Arreglo: USER 65532:65532")
	case strings.HasPrefix(usuario, "0") || usuario == "root":
		t.Errorf("el contenedor corre como %q, o sea como root", usuario)
	case !regexp.MustCompile(`^\d+(:\d+)?$`).MatchString(usuario):
		t.Errorf(`USER es %q y tiene que ser NUMERICO (65532:65532).
  Con un nombre, quien despliegue con una politica de "runAsNonRoot" no puede
  comprobar que no es root sin resolver el nombre dentro de la imagen, y en
  Kubernetes eso hace que el pod no arranque.`, usuario)
	}
}

// El binario se construye sin cgo, sin rutas de la maquina dentro y sin el
// identificador que cambia en cada construccion.
//
// Las tres cosas juntas son lo que hace que dos construcciones del mismo commit
// den el MISMO binario. Sin eso, la firma de una release dice quien la subio y
// no dice que salga de este codigo, que es justo lo que un receptor querria
// comprobar.
func TestElBinarioDeLaImagenSeConstruyeReproducible(t *testing.T) {
	fuente := leerDockerfile(t)
	codigo := strings.Join(lineasDeCodigo(fuente), "\n")
	for _, exigido := range []struct{ trozo, porque string }{
		{"CGO_ENABLED=0", "con cgo el binario enlaza contra la libc del sistema y deja de " +
			"arrancar en scratch"},
		{"-trimpath", "sin el, las rutas absolutas de la maquina que construyo se quedan " +
			"dentro del binario y dos maquinas dan binarios distintos"},
		{"-buildid=", "sin vaciarlo, el enlazador mete un identificador que cambia en cada " +
			"construccion"},
	} {
		if !strings.Contains(codigo, exigido.trozo) {
			t.Errorf("el Dockerfile ya no construye con %s: %s.\n"+
				"  La puerta que lo mide esta en .github/workflows/etapa2-distribucion.yml "+
				"(dos construcciones, mismo sha256).", exigido.trozo, exigido.porque)
		}
	}
}

// La imagen base va fijada por digest.
//
// Una etiqueta la reescribe quien la publica. Con `FROM golang:1.24-alpine` a
// secas, la misma linea construye otra cosa cada mes: la reproducibilidad se
// pierde sin que el diff ensene nada, y un compromiso aguas arriba entra solo.
func TestLaImagenBaseVaFijadaPorDigest(t *testing.T) {
	for _, l := range lineasDeCodigo(leerDockerfile(t)) {
		if !strings.HasPrefix(l, "FROM ") {
			continue
		}
		campos := strings.Fields(l)
		base := campos[1]
		if base == "scratch" || strings.HasPrefix(base, "$") {
			continue
		}
		// FROM --platform=... imagen: la imagen es el primer campo que no
		// empieza por guion.
		for _, c := range campos[1:] {
			if strings.HasPrefix(c, "-") {
				continue
			}
			base = c
			break
		}
		if base == "scratch" {
			continue
		}
		if !strings.Contains(base, "@sha256:") {
			t.Errorf(`la imagen base %q no va fijada por digest.
  Una etiqueta la reescribe quien la publica, asi que esta misma linea puede
  construir otra cosa el mes que viene sin que el diff ensene nada.
  Arreglo: docker buildx imagetools inspect %s   y pegar el digest detras de la
  etiqueta, con la fecha en el comentario de al lado.`, base, base)
		}
	}
}

// El contexto de construccion excluye lo que cambia solo.
func TestElContextoDeLaImagenExcluyeLoQueCambiaSolo(t *testing.T) {
	b, err := os.ReadFile(".dockerignore")
	if err != nil {
		t.Fatalf("no hay .dockerignore: %v.\nSin el, la carpeta .git entera entra en el "+
			"contexto de construccion: la imagen deja de ser reproducible y ademas se lleva "+
			"dentro el historico del repositorio", err)
	}
	s := string(b)
	for _, quiero := range []string{".git", ".claude"} {
		if !strings.Contains(s, quiero) {
			t.Errorf(".dockerignore no excluye %q", quiero)
		}
	}
}

// El corpus y el expediente de ejemplo viajan dentro de la imagen.
//
// Sin ellos la imagen arranca y no ensena nada: `serve` se niega a levantarse
// sin corpus y `verify` no tiene que recalcular. Un comprador que baja la imagen
// y se encuentra con que tiene que clonar el repositorio para probarla ha
// perdido ya la primera mañana, que es toda la que da.
func TestLaImagenTraeCorpusYExpedienteParaQueArranqueSola(t *testing.T) {
	codigo := strings.Join(lineasDeCodigo(leerDockerfile(t)), "\n")
	for _, quiero := range []string{"paquetes", "expediente-demo.json", "contexto-demo.json"} {
		if !strings.Contains(codigo, quiero) {
			t.Errorf("la imagen no copia %q dentro. Sin eso, `docker run plazum` arranca y no "+
				"tiene nada que ensenar", quiero)
		}
	}
	if !strings.Contains(codigo, "ENTRYPOINT") {
		t.Error("la imagen no declara ENTRYPOINT, asi que `docker run plazum verify ...` no " +
			"funciona como una orden y hay que escribir la ruta del binario a mano")
	}
	if !strings.Contains(codigo, "CMD") {
		t.Error("la imagen no declara CMD, asi que `docker run plazum` a secas no hace nada. " +
			"Es justo lo primero que se teclea")
	}
}
