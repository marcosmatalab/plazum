package plazum

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
	// anchore/sbom-action sube por su cuenta, y no se veia. Sus dos entradas
	// `upload-artifact` y `upload-release-assets` valen TRUE por defecto (leido de
	// su action.yml el 04-09-2026): sube el SBOM como artefacto del workflow
	// siempre, y en un push de etiqueta busca la release de ese tag para
	// adjuntarselo. Hoy no la encuentra porque el paso corre ANTES de que
	// action-gh-release la cree, o sea que lo unico que lo salva es el ORDEN de
	// los pasos. Un valor por defecto permisivo que solo esta apagado por accidente
	// es el invariante 8 con otro traje.
	{"anchore/sbom-action", "generar el SBOM y, por defecto, subirlo como artefacto y a la release"},
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
// 1 bis. La forma de la etiqueta: quien mueve `latest` y quien no
// ---------------------------------------------------------------------------
//
// EL CANDADO CONTESTA UNA PREGUNTA Y HAY DOS. `publicar` dice SI sale algo de la
// maquina; no dice CON QUE FORMA sale. Hasta el 04-09-2026 la segunda no la
// contestaba nadie: `-t "${destino}:latest"` era incondicional y
// `softprops/action-gh-release` se invocaba sin `prerelease`, asi que empujar
// v0.1.0-rc1 habria movido ghcr.io/.../plazum:latest al candidato y lo habria
// marcado como la release actual del repositorio.
//
// POR QUE ESO ES CARO Y NO SOLO FEO. `latest` es lo que se descarga quien no
// elige version: un `docker pull`, un Dockerfile ajeno con `FROM`, un
// despliegue que no fija nada. Moverlo a un candidato no rompe el candidato,
// rompe a todo el que creia estar en la version estable, y no hay commit que lo
// deshaga porque ya se lo bajaron.
//
// Y NO SE PUEDE VER EJECUTANDO: este workflow lleva CERO ejecuciones (medido el
// 04-09-2026 con `gh run list --workflow=release.yml`, vacio) y CERO etiquetas
// en el remoto. La primera ejecucion de verdad iba a ser tambien la primera vez
// que alguien miraba lo que hacia.

// marcadorDeForma es la salida que separa una version DE VERDAD de un
// candidato.
//
// NO VALE `startsWith(github.ref, 'refs/tags/v')`, y en esa diferencia esta
// todo el fallo: esa condicion dice «esto es una etiqueta», y `v0.1.0-rc1`
// TAMBIEN es una etiqueta. Una guarda que no distingue las dos deja pasar
// exactamente el caso para el que se escribio.
const marcadorDeForma = "outputs.definitiva"

// marcadoresDeEtiquetaFlotante son las formas de mover un puntero que apunta a
// «lo ultimo». Una etiqueta flotante no es un nombre mas: es la que se lleva
// quien no eligio.
//
// Cada patron lleva los dos puntos delante a proposito. `latest` a secas caza
// `runs-on: ubuntu-latest` en los trece workflows y una puerta que grita en
// cada linea de `runs-on` es una puerta que alguien acaba borrando.
var marcadoresDeEtiquetaFlotante = []struct{ patron, que string }{
	{":latest", "mover 'latest' en un registro de imagenes"},
	{":stable", "mover 'stable' en un registro de imagenes"},
	{":edge", "mover 'edge' en un registro de imagenes"},
	{"--all-tags", "empujar de golpe todas las etiquetas locales, 'latest' incluida"},
	{"make_latest", "declarar una release como la actual del repositorio"},
}

// marcadoresDeRelease son las formas de crear una release de GitHub. Se separan
// de los de publicacion porque la pregunta es otra: aquellos preguntan si
// publica, estos si lo publicado se presenta como version o como candidato.
var marcadoresDeRelease = []string{"softprops/action-gh-release", "gh release create"}

// pasoDeWorkflow es un paso de CI con lo justo para juzgarlo: sus lineas y las
// dos condiciones que pueden guardarlo.
type pasoDeWorkflow struct {
	fichero     string
	condTrabajo string
	condPaso    string
	lineas      []string
}

// recorrerPasos parte los workflows en pasos, por indentacion. Es por
// indentacion y no con una biblioteca de YAML porque DEPENDENCIAS.md es lista
// cerrada, igual que el resto de este fichero.
//
// Los ficheros se recorren ORDENADOS: con el orden de un mapa, dos ejecuciones
// del mismo arbol dan la misma lista de fallos en distinto orden, y eso hace
// que un diff de salidas de CI parezca un cambio cuando no lo es.
func recorrerPasos(cuerpos map[string]string) []pasoDeWorkflow {
	nombres := make([]string, 0, len(cuerpos))
	for n := range cuerpos {
		nombres = append(nombres, n)
	}
	sort.Strings(nombres)

	var pasos []pasoDeWorkflow
	for _, nombre := range nombres {
		condTrabajo, condPaso := "", ""
		enPaso := false
		var lineas []string
		cerrar := func() {
			if enPaso && len(lineas) > 0 {
				pasos = append(pasos, pasoDeWorkflow{nombre, condTrabajo, condPaso, lineas})
			}
		}
		for _, linea := range strings.Split(cuerpos[nombre], "\n") {
			switch {
			case inicioDeTrabajo.MatchString(linea):
				cerrar()
				condTrabajo, condPaso, enPaso, lineas = "", "", false, nil
			case inicioDePaso.MatchString(linea):
				cerrar()
				condPaso, enPaso, lineas = "", true, nil
			case condicionJob.MatchString(linea) && !enPaso:
				condTrabajo = linea
			case condicionDePaso.MatchString(linea):
				condPaso = linea
			}
			if enPaso {
				lineas = append(lineas, linea)
			}
		}
		cerrar()
	}
	return pasos
}

// codigoDe devuelve la linea sin su comentario.
//
// LA ALMOHADILLA SE BUSCA FUERA DE LAS COMILLAS, y no es un detalle: cortar en
// la primera ` #` que aparezca deja `docker push "$r:latest"  # ojo` bien
// juzgado pero convierte `echo "a # b"; docker push "$r:latest"` en una linea
// sin push. El fallo de ese atajo va SIEMPRE en la direccion mala: lo que se
// pierde al cortar de mas es la parte que publica, nunca la prosa.
func codigoDe(linea string) string {
	if strings.HasPrefix(strings.TrimSpace(linea), "#") {
		return ""
	}
	comillas := 0
	for i := 0; i+1 < len(linea); i++ {
		if linea[i] == '"' {
			comillas++
		}
		if linea[i] == ' ' && linea[i+1] == '#' && comillas%2 == 0 {
			return linea[:i]
		}
	}
	return linea
}

// hallazgoDeEtiqueta es una linea que mueve un puntero flotante sin preguntar
// por la forma de la etiqueta.
type hallazgoDeEtiqueta struct {
	fichero, linea, que, condPaso, condTrabajo string
}

// preguntaPorLaFormaDeLaEtiqueta dice si un paso CONSULTA de verdad el criterio,
// y no si lo menciona.
//
// LA DIFERENCIA LA ENCONTRO UNA MUTACION DE ESTA MISMA PUERTA, el 04-09-2026.
// La primera version buscaba el marcador en el bloque entero del paso, asi que
// bastaba nombrarlo. Se borro la unica linea que lo usaba de verdad
// (`DEFINITIVA: ${{ needs.candado.outputs.definitiva }}`) y la puerta SIGUIO
// VERDE, porque el mensaje de error del propio paso decia «Sale de
// needs.candado.outputs.definitiva». O sea que la guarda la satisfacia su
// propia prosa: exactamente la familia de guardas que no guardan, y esta vez
// dentro de la guarda escrita para cerrarla.
//
// La regla que lo arregla es la que hace el trabajo en un workflow: un valor
// solo LLEGA a un paso si viaja dentro de una expresion `${{ }}` o si esta en
// un `if:`, que ya es contexto de expresion. Un `echo` que lo nombra y un
// comentario que lo explica no mueven nada, y ahora no cuentan.
func preguntaPorLaFormaDeLaEtiqueta(p pasoDeWorkflow) bool {
	if strings.Contains(p.condPaso, marcadorDeForma) || strings.Contains(p.condTrabajo, marcadorDeForma) {
		return true
	}
	for _, linea := range p.lineas {
		codigo := codigoDe(linea)
		if strings.Contains(codigo, marcadorDeForma) && strings.Contains(codigo, "${{") {
			return true
		}
	}
	return false
}

// buscarEtiquetasFlotantes es el recorrido, apartado de su test para que el
// control negativo pueda darle de comer workflows sintéticos. Un guarda que
// solo se ha probado contra el arbol de hoy no se sabe si acusa a quien debe.
//
// Devuelve tambien cuantas lineas flotantes ha VISTO: cero vistas es una puerta
// sin nada que vigilar, y eso se dice, no se da por verde.
func buscarEtiquetasFlotantes(cuerpos map[string]string) (hallazgos []hallazgoDeEtiqueta, vistos int) {
	for _, p := range recorrerPasos(cuerpos) {
		preguntaPorLaForma := preguntaPorLaFormaDeLaEtiqueta(p)

		for _, linea := range p.lineas {
			codigo := codigoDe(linea)
			for _, m := range marcadoresDeEtiquetaFlotante {
				if !strings.Contains(codigo, m.patron) {
					continue
				}
				vistos++
				if preguntaPorLaForma {
					continue
				}
				hallazgos = append(hallazgos, hallazgoDeEtiqueta{
					fichero: p.fichero, linea: strings.TrimSpace(linea), que: m.que,
					condPaso: strings.TrimSpace(p.condPaso), condTrabajo: strings.TrimSpace(p.condTrabajo),
				})
			}
		}
	}
	return hallazgos, vistos
}

// TestNadieMueveLatestSinPreguntarPorLaFormaDeLaEtiqueta.
func TestNadieMueveLatestSinPreguntarPorLaFormaDeLaEtiqueta(t *testing.T) {
	hallazgos, vistos := buscarEtiquetasFlotantes(leerWorkflows(t))
	for _, h := range hallazgos {
		t.Errorf(`%s: un paso mueve una etiqueta flotante sin preguntar por la forma de la etiqueta.
    linea:  %s
    hace:   %s
    if del paso:     %q
    if del trabajo:  %q
  'latest' es lo que se lleva quien NO eligio version: un docker pull a secas, un
  FROM ajeno, un despliegue que no fija nada. Apuntarlo a un candidato (-rc, -alpha,
  -beta) no rompe el candidato: rompe a todo el que creia estar en la version
  estable, y no hay commit que lo deshaga porque ya se lo bajaron.
  Arreglo: que el paso lea la salida %q del trabajo candado, que vale 'si' solo
  para vX.Y.Z sin sufijo, y que arme las etiquetas segun ella.
  OJO: startsWith(github.ref, 'refs/tags/v') NO sirve aqui. v0.1.0-rc1 tambien
  empieza por refs/tags/v, y es justo el caso que hay que dejar fuera.`,
			h.fichero, h.linea, h.que, h.condPaso, h.condTrabajo, marcadorDeForma)
	}
	if vistos == 0 {
		t.Fatal("no se ha encontrado NI UNA linea que mueva una etiqueta flotante en los " +
			"workflows. O la release ha dejado de publicar imagenes, o los marcadores de este " +
			"test han dejado de reconocerlas, y las dos cosas dejan la puerta sin nada que vigilar")
	}
	if !t.Failed() {
		t.Logf("%d lineas mueven una etiqueta flotante, todas detras del criterio de forma", vistos)
	}
}

// TestUnaReleaseDeGitHubDiceSiEsUnCandidatoOUnaVersion.
//
// El valor por defecto de `prerelease` en action-gh-release es false, o sea el
// permisivo: una release creada sin decir nada sale como version buena. Es el
// invariante 8 en su forma de siempre, el valor cero que significa «sin
// restriccion», y aqui la restriccion es justo lo que hay que declarar.
func TestUnaReleaseDeGitHubDiceSiEsUnCandidatoOUnaVersion(t *testing.T) {
	vistos := 0
	for _, p := range recorrerPasos(leerWorkflows(t)) {
		bloque := strings.Join(p.lineas, "\n")
		crea := false
		for _, linea := range p.lineas {
			codigo := codigoDe(linea)
			for _, m := range marcadoresDeRelease {
				if strings.Contains(codigo, m) {
					crea = true
				}
			}
		}
		if !crea {
			continue
		}
		vistos++
		// `prerelease` se busca en codigo y el criterio TIENE que llegar dentro
		// de una expresion: el mismo agujero que en el guarda de `latest`, y se
		// cierra con la misma regla.
		declaraPrerelease := false
		for _, linea := range p.lineas {
			if strings.Contains(codigoDe(linea), "prerelease") {
				declaraPrerelease = true
			}
		}
		if declaraPrerelease && preguntaPorLaFormaDeLaEtiqueta(p) {
			continue
		}
		t.Errorf(`%s: un paso crea una release de GitHub sin decir si es un candidato o una version.

  Sin `+"`prerelease`"+`, un v0.1.0-rc1 aparece como LA release actual del repositorio:
  es la que la portada ofrece descargar y la que responde la API de "latest release".
  Y sin sacarlo de %q seria una segunda copia del criterio, escrita a mano al lado
  de la primera, que es como se separan las dos.

  Arreglo:
      prerelease:  ${{ needs.candado.outputs.definitiva != 'si' }}
      make_latest: ${{ needs.candado.outputs.definitiva == 'si' }}

  paso:
%s`, p.fichero, marcadorDeForma, bloque)
	}
	if vistos == 0 {
		t.Fatal("ningun paso de ningun workflow crea una release de GitHub. O la release ha " +
			"desaparecido, o este recorrido ha dejado de encontrarla: las dos cosas dejan la " +
			"puerta sin nada que vigilar")
	}
	if !t.Failed() {
		t.Logf("%d pasos crean releases, todos declarando si son candidato o version", vistos)
	}
}

// TestElCriterioDeVersionDefinitivaViveEnUnSoloSitioYFallaCerrado.
//
// Las dos salidas se exigen por lo mismo que se exigen las del candado: un
// criterio que solo sabe decir que si no es un criterio. Y la expresion regular
// se exige ANCLADA porque sin el `$` final `v0.1.0-rc1` casa con `^v[0-9]+\.`
// y el candidato pasa por version, que es exactamente el fallo que esto vigila.
func TestElCriterioDeVersionDefinitivaViveEnUnSoloSitioYFallaCerrado(t *testing.T) {
	release, hay := leerWorkflows(t)["release.yml"]
	if !hay {
		t.Fatal("no hay .github/workflows/release.yml. Si el workflow de release se ha " +
			"renombrado, este test esta vigilando un fichero que ya no existe")
	}
	for _, quiero := range []struct{ trozo, porque string }{
		{"definitiva=si", "el criterio tiene que saber decir que si"},
		{"definitiva=no", "un criterio que solo sabe decir que si no es un criterio"},
		{`=~ ^v[0-9]+\.[0-9]+\.[0-9]+$`, "sin la expresion ANCLADA por los dos extremos, " +
			"v0.1.0-rc1 casa y el candidato pasa por version definitiva"},
	} {
		if !strings.Contains(release, quiero.trozo) {
			t.Errorf("release.yml ya no contiene %q: %s", quiero.trozo, quiero.porque)
		}
	}
	// El estado de hoy, en el registro, junto al del candado: los dos juntos son
	// lo que decide que pasa si alguien empuja una etiqueta ahora mismo.
	t.Logf("criterio de version definitiva: solo vX.Y.Z sin sufijo mueve 'latest' y es " +
		"la release actual")
}

// ---------------------------------------------------------------------------
// 1 ter. Los controles negativos de los dos guardas de arriba
// ---------------------------------------------------------------------------

// TestElGuardaDeLatestAcusaAlQuePublicaYNoALaProsa recorre las DOS direcciones.
//
// La de acusar es la obvia. La de NO acusar hace igual de falta y es la que se
// olvida: este repositorio escribe mas comentario que codigo en los workflows, y
// `release.yml` explica en prosa el fallo que arreglo, con `:latest` escrito
// dentro. Un guarda que acuse a esa prosa se pone rojo el dia que alguien
// documenta bien, y entonces se borra el comentario o se borra el guarda.
//
// Y el caso que da nombre a todo esto es el tercero: `startsWith(github.ref,
// 'refs/tags/v')` NO basta. Es la condicion que ya estaba puesta el 04-09-2026 y
// la que dejaba pasar v0.1.0-rc1.
func TestElGuardaDeLatestAcusaAlQuePublicaYNoALaProsa(t *testing.T) {
	const cabecera = "name: prueba\njobs:\n"
	casos := []struct {
		nombre string
		yaml   string
		acusa  bool
	}{
		{
			nombre: "mueve latest a pelo",
			acusa:  true,
			yaml: cabecera + `  imagen:
    steps:
      - name: subir
        run: |
          docker buildx build -t "$d:$e" -t "$d:latest" --push .
`,
		},
		{
			nombre: "el criterio en el env del paso",
			acusa:  false,
			yaml: cabecera + `  imagen:
    steps:
      - name: subir
        env:
          DEFINITIVA: ${{ needs.candado.outputs.definitiva }}
        run: |
          docker buildx build -t "$d:$e" -t "$d:latest" --push .
`,
		},
		{
			nombre: "solo startsWith de etiqueta, que es el fallo del 04-09-2026",
			acusa:  true,
			yaml: cabecera + `  imagen:
    steps:
      - name: subir
        if: startsWith(github.ref, 'refs/tags/v')
        run: |
          docker buildx build -t "$d:$e" -t "$d:latest" --push .
`,
		},
		{
			nombre: "solo el candado, que contesta la otra pregunta",
			acusa:  true,
			yaml: cabecera + `  imagen:
    steps:
      - name: subir
        if: needs.candado.outputs.publicar == 'si'
        run: |
          docker buildx build -t "$d:$e" -t "$d:latest" --push .
`,
		},
		{
			nombre: "latest solo en un comentario",
			acusa:  false,
			yaml: cabecera + `  imagen:
    steps:
      - name: subir
        run: |
          # antes esto ponia -t "$d:latest" y era incondicional
          docker buildx build -t "$d:$e" --push .
`,
		},
		{
			nombre: "ubuntu-latest no es una etiqueta flotante",
			acusa:  false,
			yaml: cabecera + `  imagen:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - name: nada
        run: echo hola
`,
		},
		{
			nombre: "una almohadilla dentro de comillas no tapa el push que viene detras",
			acusa:  true,
			yaml: cabecera + `  imagen:
    steps:
      - name: subir
        run: |
          echo "clave # valor" && docker push "$d:latest"
`,
		},
		{
			// EL AGUJERO QUE ENCONTRO LA MUTACION DEL 04-09-2026, congelado en
			// un caso. El paso NOMBRA el criterio en un mensaje y no lo consulta
			// en ningun sitio: la primera version de este guarda lo daba por
			// bueno, y era la guarda satisfecha por su propia prosa.
			nombre: "nombrar el criterio en un echo no es consultarlo",
			acusa:  true,
			yaml: cabecera + `  imagen:
    steps:
      - name: subir
        run: |
          echo "esto sale de needs.candado.outputs.definitiva, que calcula el candado"
          docker buildx build -t "$d:$e" -t "$d:latest" --push .
`,
		},
		{
			// Y el mismo criterio en un COMENTARIO tampoco.
			nombre: "el criterio en un comentario tampoco es consultarlo",
			acusa:  true,
			yaml: cabecera + `  imagen:
    steps:
      - name: subir
        run: |
          # ojo: needs.candado.outputs.definitiva decide si esto mueve latest
          docker buildx build -t "$d:$e" -t "$d:latest" --push .
`,
		},
		{
			nombre: "el criterio en el if del trabajo si cuenta",
			acusa:  false,
			yaml: cabecera + `  imagen:
    if: needs.candado.outputs.definitiva == 'si'
    steps:
      - name: subir
        run: |
          docker buildx build -t "$d:$e" -t "$d:latest" --push .
`,
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			hallazgos, vistos := buscarEtiquetasFlotantes(map[string]string{"prueba.yml": c.yaml})
			switch {
			case c.acusa && len(hallazgos) == 0:
				t.Errorf("el guarda NO acusa y tenia que acusar (%d lineas flotantes vistas).\n"+
					"  Un guarda que deja pasar esto es el guarda que dejo pasar el caso real.\n%s",
					vistos, c.yaml)
			case !c.acusa && len(hallazgos) > 0:
				t.Errorf("el guarda acusa y NO tenia que acusar: %+v\n"+
					"  Un falso rojo aqui se paga borrando el comentario que lo provoca, o el\n"+
					"  guarda entero, y las dos salidas son peores que el problema.\n%s",
					hallazgos, c.yaml)
			}
		})
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

// EL ENSAYO TIENE QUE SEGUIR HABLANDO CUANDO ALGO FALLA, QUE ES CUANDO HACE
// FALTA.
//
// # El caso, sobre una ejecucion real y no sobre una mutacion
//
// `ensayo` es el trabajo cuyo unico oficio es contar que habria salido de esta
// ejecucion si hubiera publicado. Declara `needs: [candado, binarios, imagen]`,
// y en GitHub Actions un trabajo con `needs` SE SALTA cuando cualquiera de los
// que necesita falla, salvo que su condicion diga lo contrario.
//
// Asi que se callaba exactamente en la ejecucion que habia que entender. Run
// 33853740997 del 04-09-2026, el primero de la vida de release.yml:
//
//	imagen Docker        failure
//	firmar y publicar    skipped
//	ensayo sin publicar  skipped   <- el unico que explicaba algo
//
// Es la familia de «el diagnostico se apaga con lo que diagnostica», y es
// hermana de lo que ya vigila TestNingunPasoSeRompeCuandoElCandadoSeQuita: una
// salida que solo existe en el camino feliz no es una salida, es un adorno.
//
// # Que exige esta puerta, y por que no basta con mirar el `needs`
//
// No se le puede quitar el `needs`: el ensayo NECESITA los artefactos de
// `binarios` para listarlos. Lo que se exige es que su condicion sobreviva al
// fallo de un dependiente, o sea que nombre `always()` o `!cancelled()`.
//
// Y se exige tambien que el paso diga el ESTADO de sus dependientes. Un ensayo
// que corre tras un fallo y lista cero artefactos sin decir por que es la
// version tranquilizadora de no saber nada: cero ficheros se lee como «no habia
// nada que construir».
func TestElEnsayoSigueHablandoCuandoAlgoDeLoQueNecesitaFalla(t *testing.T) {
	cuerpos := leerWorkflows(t)
	rel, hay := cuerpos["release.yml"]
	if !hay {
		t.Fatal("no encuentro release.yml entre los workflows leidos: esta puerta estaria " +
			"comprobando el vacio")
	}

	i := strings.Index(rel, "\n  ensayo:")
	if i < 0 {
		t.Fatal("release.yml ya no declara el trabajo `ensayo`.\n" +
			"  Si se ha renombrado, esta puerta hay que reapuntarla, no borrarla: lo que " +
			"vigila es que la salida que explica una ejecucion que NO publica no desaparezca " +
			"cuando algo falla.")
	}
	bloque := rel[i:]
	// El siguiente trabajo empieza con dos espacios y un nombre; si no hay
	// ninguno, `ensayo` es el ultimo y el bloque llega hasta el final.
	if j := strings.Index(bloque[1:], "\n  ensayo"); j >= 0 {
		t.Fatal("release.yml declara `ensayo` dos veces")
	}

	cond := ""
	for _, l := range strings.Split(bloque, "\n") {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "if:") {
			cond = t
			break
		}
	}
	if cond == "" {
		t.Fatal("el trabajo `ensayo` no tiene condicion `if:`, asi que corre siempre que sus " +
			"dependientes salgan bien y NUNCA cuando alguno falle. Es justo al reves de lo " +
			"que hace falta")
	}
	if !strings.Contains(cond, "always()") && !strings.Contains(cond, "cancelled()") {
		t.Errorf("la condicion de `ensayo` es:\n    %s\n"+
			"  y no nombra `always()` ni `!cancelled()`. Con `needs: [candado, binarios, "+
			"imagen]`, GitHub SALTA este trabajo en cuanto uno de los tres falla, o sea que "+
			"el unico trabajo que explica una ejecucion que no publica se calla exactamente "+
			"en la ejecucion que hay que entender.\n"+
			"  Paso de verdad: run 33853740997, `imagen` failure y `ensayo` skipped.\n"+
			"  Arreglo: envolver la condicion en `${{ !cancelled() && (...) }}`.", cond)
	}

	// Y QUE DIGA EL ESTADO DE LOS QUE NECESITA. Sin esto, el arreglo de arriba
	// consigue que el trabajo corra y no consigue que sirva: listaria cero
	// artefactos sin decir que es porque `binarios` no llego a producirlos.
	for _, dep := range []string{"candado", "binarios", "imagen"} {
		if !strings.Contains(bloque, "needs."+dep+".result") {
			t.Errorf("el cuerpo de `ensayo` no imprime `needs.%s.result`.\n"+
				"  Un ensayo que corre despues de un fallo y lista cero ficheros sin decir "+
				"que dependiente fallo es la version tranquilizadora de no saber nada: cero "+
				"artefactos se lee como «no habia nada que construir».", dep)
		}
	}

	// Y QUE LA DESCARGA DE ARTEFACTOS NO PUEDA MATARLO, que es la misma puerta
	// por la que se colaria el silencio otra vez: sin artefactos,
	// download-artifact sale con error y tumba el paso.
	if !strings.Contains(bloque, "continue-on-error: true") {
		t.Error("`ensayo` descarga artefactos sin `continue-on-error: true`.\n" +
			"  Si `binarios` fallo no hay nada que descargar y la accion sale con error, " +
			"asi que el ensayo volveria a callarse en la ejecucion rota, por otra puerta.")
	}
}

// LA PUERTA DEL P0 DE ESTE TRAMO: EL CORPUS REAL TIENE QUE VIAJAR EN LA RELEASE.
//
// # El fallo que la trae, medido y no supuesto
//
// A 04-09-2026 la release publicaba SEIS BINARIOS Y NINGUN CORPUS. La prueba de
// la maquina limpia (docs/lanzamiento/maquina-limpia.sh) salia con codigo 0 y
// llegaba al calendario, y lo hacia con el corpus de DEMOSTRACION: 1 paquete y 3
// relojes. Los 33 directorios de paquetes/ y los 222 relojes que ensena
// `plazum calendario --todos-los-relojes` se quedaban en el repositorio.
//
// Nadie mentia. El guion decia la verdad («EL CALENDARIO: ok») y la decia con el
// corpus equivocado, que es la forma de enganar que no necesita una sola frase
// falsa: basta con no decir cual de los dos corpus contesto.
//
// Publicar asi significa que el primer plazum publico del mundo es una demo
// vacia. La primera impresion de todo el que lo pruebe es «esto no trae nada», y
// esa no se repite.
//
// # Por que este fichero y no una linea mas en distribucion_test.go
//
// Porque lo que vigila no es la forma de un workflow: es que DOS ARTEFACTOS
// DISTINTOS SIGAN CASANDO. El binario lleva dentro la huella del corpus y el
// corpus viaja al lado; si dejan de casar, no se rompe nada visible aqui, se
// rompe en la maquina de quien se lo baje.
//
// # El emparejamiento, dicho en voz alta (invariante 7)
//
// El workflow escribe `-X main.anclaCorpus=<huella>` y el codigo declara la
// variable `anclaCorpus` en el paquete `main` de cmd/plazum. CASAN POR EL NOMBRE
// DEL SIMBOLO, que en un lado es una cadena dentro de un YAML y en el otro es un
// identificador de Go. Nadie comprueba que sean el mismo.
//
// Y el enlazador tampoco: `-X` sobre un simbolo que no existe NO ES UN ERROR.
// `go build` sale con 0 y produce un binario con el ancla vacia. Ese binario
// pasa el resto del workflow entero (verifica el expediente, tiene su suma, se
// firma en Rekor) y solo se rompe en manos del comprador, cuando `--instalar` se
// niega porque no tiene contra que comprobar. O sea: el fallo llega DESPUES del
// unico paso irreversible del proyecto.
//
// Por eso la comprobacion es cruzada y en los dos sentidos: lo que el workflow
// nombra tiene que existir en el codigo, y el codigo tiene que seguir teniendo
// quien lo ponga.

// simboloDelAncla es el emparejamiento, escrito una sola vez. Cambiarlo en el
// codigo sin cambiarlo aqui pone rojo este fichero, que es exactamente lo que
// tiene que pasar.
const simboloDelAncla = "main.anclaCorpus"

// nombreDeLaVariableDelAncla es la mitad de Go del mismo emparejamiento.
const nombreDeLaVariableDelAncla = "anclaCorpus"

// TestLaReleasePublicaElCorpusYNoSoloLosBinarios.
//
// Las cuatro cosas que tienen que estar, y ninguna se deduce de las otras:
// empaquetarlo, meter su huella en los binarios, que el trabajo que publica
// dependa de quien lo produce, y que el activo llegue a firmarse.
func TestLaReleasePublicaElCorpusYNoSoloLosBinarios(t *testing.T) {
	rel := leerRelease(t)

	exigencias := []struct{ trozo, que, porque string }{
		{
			trozo: "corpus --empaquetar",
			que:   "empaquetar el corpus como activo de la release",
			porque: "sin esto la release son seis binarios y ningun marco: quien se los " +
				"baje llega al calendario con el corpus de demostracion, un paquete y " +
				"tres relojes, y se va pensando que plazum no trae nada",
		},
		{
			trozo: "-X " + simboloDelAncla + "=",
			que:   "meter la huella del corpus dentro de los binarios",
			porque: "sin el ancla, el binario no puede decir si el corpus que le dan es el " +
				"que se publico con el. Un corpus que entra sin comprobar es peor que " +
				"ninguno: son fechas legales en las que alguien va a confiar",
		},
		{
			trozo:  "plazum-corpus.tar.gz",
			que:    "nombrar el activo del corpus",
			porque: "es el fichero que se baja el comprador y el que nombra la documentacion",
		},
		{
			trozo: "plazum-corpus.huella",
			que:   "publicar la huella en texto, al lado del activo",
			porque: "es lo que permite instalar un corpus MAS NUEVO que el binario sin " +
				"recompilar nada: el operador la lee de la pagina de la release y la pasa " +
				"con --huella-esperada. Sin ella, el ancla del binario seria un muro",
		},
	}
	for _, e := range exigencias {
		if !strings.Contains(rel, e.trozo) {
			t.Errorf("release.yml ya no hace esto: %s.\n  Falta %q.\n  Por que importa: %s",
				e.que, e.trozo, e.porque)
		}
	}
}

// TestElSimboloQueElWorkflowInyectaExisteEnElCodigo es la puerta que de verdad
// paga su sitio.
//
// `-X main.loQueSea=x` sobre un simbolo inexistente sale con 0. No hay ningun
// otro sitio del proyecto donde este emparejamiento se compruebe, y su fallo es
// silencioso hasta despues de publicar.
func TestElSimboloQueElWorkflowInyectaExisteEnElCodigo(t *testing.T) {
	rel := leerRelease(t)
	if !strings.Contains(rel, "-X "+simboloDelAncla+"=") {
		t.Fatalf("release.yml no inyecta %s. Si el ancla se ha movido de sitio, "+
			"actualiza simboloDelAncla en este fichero y di por que en el commit",
			simboloDelAncla)
	}

	// La mitad de Go. El simbolo es `main.anclaCorpus`, o sea la variable
	// `anclaCorpus` del paquete main de cmd/plazum.
	paquete, variable, ok := strings.Cut(simboloDelAncla, ".")
	if !ok || paquete != "main" {
		t.Fatalf("simboloDelAncla es %q y tiene que ser main.<variable>: -X solo alcanza a "+
			"una variable de cadena de un paquete", simboloDelAncla)
	}
	if variable != nombreDeLaVariableDelAncla {
		t.Fatalf("las dos mitades del emparejamiento no dicen lo mismo: el workflow inyecta "+
			"%q y este fichero espera la variable %q", variable, nombreDeLaVariableDelAncla)
	}

	fuentes, err := filepath.Glob(filepath.Join("cmd", "plazum", "*.go"))
	if err != nil || len(fuentes) == 0 {
		t.Fatalf("no encuentro las fuentes de cmd/plazum (%v). Si el comando se ha movido, "+
			"este test estaria auditando el vacio y dando verde", err)
	}
	declaracion := "var " + variable + " string"
	encontrada := ""
	for _, f := range fuentes {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f) // #nosec G304 -- rutas del propio repositorio, salidas de Glob
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), declaracion) {
			encontrada = f
			break
		}
	}
	if encontrada == "" {
		t.Fatalf("release.yml inyecta `-X %s=<huella>` y NO hay ningun `%s` en cmd/plazum.\n"+
			"  El enlazador NO se queja de esto: -X sobre un simbolo que no existe sale con 0\n"+
			"  y deja el binario con el ancla vacia. Ese binario pasa el resto del workflow\n"+
			"  entero, se firma en Rekor (append-only) y solo se rompe en la maquina del\n"+
			"  comprador, cuando `plazum corpus --instalar` se niega por no tener contra que\n"+
			"  comprobar.\n"+
			"  Arreglo: o vuelve a declarar la variable, o cambia simboloDelAncla y el -X\n"+
			"  del workflow a la vez, en el mismo commit.",
			simboloDelAncla, declaracion)
	}

	// Y QUE SIGA SIENDO UNA VARIABLE Y NO UNA CONSTANTE. -X no puede escribir
	// en una constante: `const anclaCorpus = "..."` compila, el -X se ignora en
	// silencio, y otra vez binario sin ancla firmado en Rekor.
	b, err := os.ReadFile(encontrada) // #nosec G304 -- ruta salida del bucle de arriba
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "const "+variable+" ") {
		t.Errorf("%s declara %s como constante. -X no puede escribir en una constante y "+
			"se ignora sin decir nada", encontrada, variable)
	}
}

// TestElTrabajoQuePublicaDependeDeQuienProduceElCorpus.
//
// `publicar` junta artefactos con `find recogido -type f -exec cp {} dist/`. Si
// el corpus no llega (el trabajo no corrio, el artefacto cambio de nombre, la
// descarga fallo), ese comando NO da error: deja un `dist` con seis binarios y
// sin corpus, y la release sale verde, firmada y vacia de marcos.
//
// O sea que el P0 puede volver por una puerta que no es la que se cerro.
func TestElTrabajoQuePublicaDependeDeQuienProduceElCorpus(t *testing.T) {
	rel := leerRelease(t)

	bloque := trabajoDelWorkflow(t, rel, "publicar")
	if !strings.Contains(bloque, "corpus") {
		t.Error("el trabajo `publicar` no nombra `corpus` en sus `needs`.\n" +
			"  Sin esa dependencia, publicar puede correr antes de que el corpus exista y\n" +
			"  `find recogido -exec cp` no protesta: junta lo que haya. La release saldria\n" +
			"  verde y firmada con seis binarios y ningun marco dentro.")
	}
	// Y que ademas lo COMPRUEBE, porque una dependencia dice que el trabajo
	// termino, no que su artefacto llegara hasta aqui.
	if !strings.Contains(bloque, "dist/plazum-corpus.tar.gz") &&
		!strings.Contains(bloque, "plazum-corpus.tar.gz") {
		t.Error("el trabajo `publicar` no comprueba que el corpus este en `dist` antes de firmar.\n" +
			"  `needs` garantiza que el trabajo anterior acabo, no que su artefacto se haya\n" +
			"  descargado y copiado. Es el ultimo sitio donde se puede parar sin haber\n" +
			"  escrito en Rekor.")
	}
}

// TestLaPruebaDeLaMaquinaLimpiaDiceConQueCorpusLlegoAlCalendario.
//
// EL GUION YA MEDIA ESTO Y SE LO CALLABA. Antes de este tramo, maquina-limpia.sh
// calculaba `_marcos` con un grep sobre la salida del calendario y NO LO IMPRIMIA
// NUNCA: la variable se asignaba y se tiraba. La mitad contable de la afirmacion
// estaba escrita y muerta, y la transcripcion se leia como «el producto funciona
// entero» sin que ninguna linea lo dijera.
//
// Un hueco sin cardinal se olvida; uno con cardinal molesta hasta que se cierra.
// Asi que el guion tiene que IMPRIMIR los dos numeros, y esta puerta exige que
// siga imprimiendolos.
func TestLaPruebaDeLaMaquinaLimpiaDiceConQueCorpusLlegoAlCalendario(t *testing.T) {
	ruta := filepath.Join("docs", "lanzamiento", "maquina-limpia.sh")
	b, err := os.ReadFile(ruta) // #nosec G304 -- ruta fija del repositorio
	if err != nil {
		t.Fatalf("no puedo leer %s: %v", ruta, err)
	}
	guion := string(b)

	for _, e := range []struct{ trozo, porque string }{
		{"_paquetes", "sin el numero de paquetes, la transcripcion no distingue el corpus " +
			"real del de demostracion, que es exactamente el P0 de este tramo"},
		{"_relojes", "el numero de relojes es la otra mitad: un corpus con 33 paquetes y " +
			"tres relojes seria igual de vacio y contaria 33"},
	} {
		if !strings.Contains(guion, e.trozo) {
			t.Errorf("maquina-limpia.sh ya no cuenta %s. %s", e.trozo, e.porque)
		}
		// Y QUE LO IMPRIMA, no solo que lo calcule. Es el fallo exacto que trae
		// esta puerta: la variable existia y nadie la sacaba por pantalla.
		if !strings.Contains(guion, "${"+e.trozo+"}") && !strings.Contains(guion, "$"+e.trozo+" ") {
			t.Errorf("maquina-limpia.sh calcula %s y no lo usa en ninguna salida.\n"+
				"  Una medida que se toma y no se imprime es peor que no tomarla: el guion\n"+
				"  parece que lo comprueba y la transcripcion no lo dice.", e.trozo)
		}
	}

	// LA GUARDA QUE CADUCA. El guion traia un aviso («se ha llegado con el
	// corpus de DEMOSTRACION») que se rompia solo si aparecian mas de 5
	// paquetes, para que el aviso no mintiera al reves el dia que el corpus si
	// viajara. Ese dia es hoy. La guarda tiene que seguir existiendo y tiene
	// que apuntar al mundo nuevo: ahora lo sospechoso es tener POCOS paquetes.
	if !strings.Contains(guion, "PASO ROTO") {
		t.Error("maquina-limpia.sh ya no tiene ninguna guarda que se rompa sola cuando su " +
			"propio aviso caduque. Ese mecanismo es lo que hizo visible este P0.")
	}
}

// ---------------------------------------------------------------------------
// utilidades
// ---------------------------------------------------------------------------

func leerRelease(t *testing.T) string {
	t.Helper()
	ruta := filepath.Join(".github", "workflows", "release.yml")
	b, err := os.ReadFile(ruta) // #nosec G304 -- ruta fija del repositorio
	if err != nil {
		t.Fatalf("no puedo leer %s: %v", ruta, err)
	}
	if len(b) < 500 {
		t.Fatalf("%s tiene %d bytes. Si el workflow se vacio o se movio, este fichero "+
			"estaria auditando la nada y dando verde", ruta, len(b))
	}
	return string(b)
}

// trabajoDelWorkflow devuelve el bloque de un trabajo, de su cabecera hasta la
// del siguiente. Por indentacion, como el resto de distribucion_test.go y por lo
// mismo: DEPENDENCIAS.md es lista cerrada y aqui no entra una biblioteca de YAML.
func trabajoDelWorkflow(t *testing.T, cuerpo, nombre string) string {
	t.Helper()
	lineas := strings.Split(cuerpo, "\n")
	inicio := -1
	for i, l := range lineas {
		if l == "  "+nombre+":" {
			inicio = i
			break
		}
	}
	if inicio < 0 {
		t.Fatalf("release.yml no tiene un trabajo llamado %q. Si se renombro, este test "+
			"estaria auditando el vacio", nombre)
	}
	for i := inicio + 1; i < len(lineas); i++ {
		l := lineas[i]
		// Otra cabecera de trabajo: dos espacios, nombre, dos puntos.
		if len(l) > 2 && l[0] == ' ' && l[1] == ' ' && l[2] != ' ' && strings.HasSuffix(strings.TrimSpace(l), ":") {
			return strings.Join(lineas[inicio:i], "\n")
		}
	}
	return strings.Join(lineas[inicio:], "\n")
}

// TestLaImagenApuntaAlRepositorioQueDeVerdadTieneSuFuente.
//
// # El fallo, encontrado leyendo y no por una puerta
//
// El 04-09-2026 la imagen declaraba:
//
//	org.opencontainers.image.source="https://github.com/plazum/plazum"
//
// y el modulo es github.com/marcosmatalab/plazum, que es tambien el remoto. La
// etiqueta apuntaba a un repositorio que no es este.
//
// # Por que no es cosmetica, y son dos motivos
//
//  1. LA LICENCIA. plazum es AGPL-3.0, o sea que distribuir el binario obliga a
//     ofrecer la fuente correspondiente. En una imagen de contenedor, esa oferta
//     ES esta etiqueta: es lo unico que lleva dentro un `scratch` para decir de
//     donde salio. Apuntando a otro sitio, la oferta manda a quien la siga a un
//     repositorio que no contiene el codigo que esta ejecutando.
//  2. GHCR LA USA PARA ENLAZAR EL PAQUETE CON SU REPOSITORIO. Con la etiqueta
//     equivocada, el paquete publicado puede quedarse sin enlazar, y entonces la
//     pagina que ve quien se lo descarga no lleva a ninguna fuente.
//
// # Por que se DERIVA de go.mod y no se escribe aqui
//
// Escribir la URL buena en este test crea una segunda copia del dato, y la regla
// del proyecto es que un motivo que repite un dato a mano es sospechoso entero.
// La ruta del modulo ya es la identidad del proyecto, esta en un solo sitio y la
// vigila el compilador. Si el proyecto se mueve de sitio, se cambia go.mod y
// esto sigue cuadrando solo.
func TestLaImagenApuntaAlRepositorioQueDeVerdadTieneSuFuente(t *testing.T) {
	b, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatalf("no puedo leer go.mod: %v", err)
	}
	var modulo string
	for _, l := range strings.Split(string(b), "\n") {
		if resto, ok := strings.CutPrefix(strings.TrimSpace(l), "module "); ok {
			modulo = strings.TrimSpace(resto)
			break
		}
	}
	if modulo == "" {
		t.Fatal("go.mod no declara `module`. Sin eso este test no tiene de donde derivar " +
			"la fuente y estaria dando verde sin comprobar nada")
	}
	// github.com/marcosmatalab/plazum -> https://github.com/marcosmatalab/plazum
	esperada := "https://" + modulo

	codigo := strings.Join(lineasDeCodigo(leerDockerfile(t)), "\n")
	const clave = "org.opencontainers.image.source="
	i := strings.Index(codigo, clave)
	if i < 0 {
		t.Fatalf("el Dockerfile no declara %s.\n"+
			"  plazum es AGPL-3.0: distribuir la imagen obliga a ofrecer la fuente, y en\n"+
			"  una imagen `scratch` esta etiqueta es la unica oferta que viaja dentro.", clave)
	}
	resto := codigo[i+len(clave):]
	valor := strings.Trim(strings.Fields(resto)[0], `"\`)

	if valor != esperada {
		t.Errorf("la imagen dice que su fuente esta en %q y el modulo es %q, o sea %q.\n"+
			"  Son dos sitios distintos. Quien siga la etiqueta para conseguir la fuente\n"+
			"  correspondiente (que es lo que la AGPL le da derecho a pedir) acaba en un\n"+
			"  repositorio que no tiene el codigo que esta ejecutando. Y ghcr.io usa esta\n"+
			"  etiqueta para enlazar el paquete con su repositorio: con la equivocada, el\n"+
			"  paquete publicado puede quedarse sin enlazar a ninguna fuente.\n"+
			"  Arreglo: pon %q en el Dockerfile, o cambia el modulo si el proyecto se ha\n"+
			"  movido de verdad.", valor, modulo, esperada, esperada)
	}
}
