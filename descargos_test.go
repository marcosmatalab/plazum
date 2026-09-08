package plazum

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// EL DESCARGO ESTÁ EN VEINTICUATRO SITIOS Y NADIE LOS ENUMERABA.
//
// # El hueco, con su nombre y su fecha
//
// `CLAUDE.md` lo llevaba escrito desde el 04-09-2026 como una de las casillas
// que el barrido no podía resolver: *«la frase está en cuatro sitios y NINGUNA
// PUERTA LAS ENUMERA, que es la forma que tenía D11-b antes de cerrarse»*. Eran
// cuatro entonces; medidos hoy sobre el catálogo, son **24**.
//
// Cada superficie tiene su propio test del descargo y todos pasan. Lo que no
// existía es la lista: una superficie nueva que enseñe pasado y se olvide de su
// descargo **no pone nada rojo**, porque no hay quien compare lo que hay con lo
// que debería haber. Es exactamente el modo de fallo que D11-b cerró para los
// estados vacíos, un año-luz más barato de cometer.
//
// # Por qué acusar en falso es el error que no se puede cometer ni una vez
//
// Está en `CLAUDE.md` y no se repite aquí entero: un vencimiento pasado sin
// registro de cumplimiento NO es un incumplimiento, es una ausencia de dato, y
// plazum no sabe distinguirlos. Quien lea una acusación falsa deja de creerse el
// resto de la pantalla, y con razón.
//
// # Qué es un descargo, mecánicamente
//
// La familia se reconoce por su FORMA y no por sus palabras: es la negación de
// una acusación seguida de lo que sí se sabe, *«esto NO dice que X: dice que Y»*.
// Se detecta por esa negación y no por la palabra «consta», y ese cambio
// encontró once descargos más:
//
//	con «consta»   13   (los que hablan de que algo no está registrado)
//	sin «consta»   11   (independencia, ilegible, fuera del periodo, salida del
//	                     alcance, empate de clasificación, sin avisos...)
//
// Buscar «consta» habría dejado fuera casi la mitad, y ninguna de esa mitad es
// menos descargo que las otras: `acta.descargo.independencia` dice *«esto NO dice
// que la auditoría esté mal hecha»*, que es la misma promesa exactamente.
//
// # Las dos direcciones
//
//	1. Todo descargo del catálogo está en el censo, y el censo nombra EL TEST
//	   QUE RECORRE SU RAMA. Sin esto, un descargo nuevo entra sin que nadie
//	   compruebe que llega a una pantalla.
//	2. Toda entrada del censo sigue existiendo y sigue siendo un descargo. Sin
//	   esto, el censo envejece igual que la prosa que vino a vigilar.
//
// # Y el control POSITIVO es el punto, no un extra (M47)
//
// «Una rama de descargo que ninguna entrada recorre es una rama que no existe»:
// la mutación M47 cambió *«en tus respuestas no aparece»* por *«lo has
// incumplido»* y no puso nada rojo, porque no había entrada que llegara ahí. Por
// eso el censo no pide «un test», pide **el test que RENDERIZA esa rama**.
//
// # SU LÍMITE, DICHO
//
// No puede comprobar que el test nombrado recorra de verdad la rama: eso
// exigiría entender el test. Hace lo mismo que
// `afirmaciones_del_producto_test.go`, que es poner la pregunta delante de
// alguien en el momento exacto en que se puede contestar, y contar los huecos
// que se declaran en vez de esconderlos.

const rutaDelCatalogoES = "adaptadores/catalogo/cadenas/es.json"

// LA FORMA DEL DESCARGO. Ver el godoc: se reconoce por la negación de la
// acusación, no por la palabra «consta».
var formaDelDescargo = regexp.MustCompile(`(?i)no dice que|no se sigue que`)

// quienRecorreElDescargo es el censo: por cada descargo, el test que renderiza
// su rama.
//
// El motivo solo se admite cuando NO hay test, y entonces cuenta como hueco y se
// imprime su cardinal. Un hueco declarado se cuenta; una atadura inventada, no.
type quienRecorre struct {
	test   string
	motivo string
}

var censoDeDescargos = map[string]quienRecorre{
	// --- el acta (superficies/acta y nucleo/acta) ---
	"acta.descargo.asignacion_que_no_casa": {
		test: "TestLoQueConstaYNoEsCulpaTambienLlevaSuFrase"},
	"acta.descargo.empate_de_clasificacion": {
		test: "TestLoQueConstaYNoEsCulpaTambienLlevaSuFrase"},
	"acta.descargo.fuera_del_periodo": {
		test: "TestLoQueConstaYNoEsCulpaTambienLlevaSuFrase"},
	"acta.descargo.ilegible": {
		test: "TestCadaSeccionTraeSuNoConstaConLaFraseDeSuPaquete"},
	"acta.descargo.independencia": {
		test: "TestLoQueConstaYNoEsCulpaTambienLlevaSuFrase"},
	"acta.descargo.no_auditado": {
		test: "TestCadaSeccionTraeSuNoConstaConLaFraseDeSuPaquete"},
	"acta.descargo.no_clasificado": {
		test: "TestCadaSeccionTraeSuNoConstaConLaFraseDeSuPaquete"},
	"acta.descargo.no_notificado": {
		test: "TestCadaSeccionTraeSuNoConstaConLaFraseDeSuPaquete"},
	"acta.descargo.no_revisado": {
		test: "TestCadaSeccionTraeSuNoConstaConLaFraseDeSuPaquete"},
	"acta.descargo.notificacion_que_no_casa": {
		test: "TestLoQueConstaYNoEsCulpaTambienLlevaSuFrase"},
	"acta.descargo.remision_fuera_del_periodo": {
		test: "TestLoQueConstaYNoEsCulpaTambienLlevaSuFrase"},
	"acta.descargo.salida_del_alcance": {
		test: "TestLoQueConstaYNoEsCulpaTambienLlevaSuFrase"},
	"acta.descargo.sin_responsable": {
		test: "TestCadaSeccionTraeSuNoConstaConLaFraseDeSuPaquete"},
	"acta.parrafo.no_dice_cumplido": {
		test: "TestElActaNoDiceQueLoQueCubreEsteCumplido"},
	"acta.parrafo.sin_incidentes": {
		test: "TestSinRegistroDeIncidentesElActaLoDiceYNoConcluye"},

	// --- el calendario ---
	"calendario.cifra.no_es_tuyo": {
		test: "TestLaPaginaDeLoQueNoTeAlcanzaNoAcusaDeIncumplir"},
	"calendario.pantalla.vencido.frase": {
		test: "TestLoVencidoSaleConSuDescargoPegado"},

	// --- el camino guiado ---
	"camino.sin_progreso": {
		test: "TestElCaminoNoDiceQueNingunPasoEsteHecho"},

	// --- las pantallas ---
	"error.alcance_ilegible": {
		test: "TestUnAlmacenQueNoSeLeeNoSeConvierteEnUnaEntrevistaEnBlanco"},
	"escalado.pantalla.sin_avisos": {
		test: "TestElPlanVacioNoDiceQueNoTengasObligaciones"},
	"evidencia.descargo": {
		test: "TestConSesionLaEvidenciaSaleConSuEstadoSuFechaYQuienLoTrajo"},
	"evidencia.ilegible": {
		test: "TestLaEvidenciaIlegibleNoSeLeeComoQueNadieHaRecolectado"},
	"pantalla.hoy.sin_constancia.descargo": {
		test: "TestElDescargoDeLoNoConstatadoVaDentroDeLaMismaTarjetaQueElNumero"},

	// --- la revisión de accesos ---
	"uar.no_consta": {
		test: "TestLaFraseDelDescargoEsLaMismaEnElNucleoYEnLaPantalla"},
}

func descargosDelCatalogo(t *testing.T) map[string]string {
	t.Helper()
	b, err := os.ReadFile(rutaDelCatalogoES) // #nosec G304 -- ruta constante del repositorio
	if err != nil {
		t.Fatalf("no puedo leer %s: %v", rutaDelCatalogoES, err)
	}
	var todas map[string]any
	if err := json.Unmarshal(b, &todas); err != nil {
		t.Fatalf("el catalogo no es JSON: %v", err)
	}
	out := map[string]string{}
	for k, v := range todas {
		s, ok := v.(string)
		if ok && formaDelDescargo.MatchString(s) {
			out[k] = s
		}
	}
	if len(out) < 10 {
		t.Fatalf("el catalogo trae %d descargos y hoy son mas de veinte: o el detector ha "+
			"dejado de casar, o el producto ha dejado de descargar, y las dos cosas hay "+
			"que saberlas", len(out))
	}
	return out
}

func TestTodoDescargoDelCatalogoTieneQuienRecorraSuRama(t *testing.T) {
	descargos := descargosDelCatalogo(t)
	nombres := nombresDeTestDelArbol(t)

	// DIRECCION 1: un descargo que nadie recorre.
	var huecos []string
	for _, k := range clavesDelDescargo(descargos) {
		q, hay := censoDeDescargos[k]
		if !hay {
			t.Errorf("la clave %q del catalogo es un DESCARGO y no esta en el censo.\n"+
				"  «%s»\n"+
				"  Un descargo dice al cliente lo que plazum NO esta afirmando, y es la "+
				"unica frase que impide que una ausencia de dato se lea como una acusacion. "+
				"Si nadie recorre su rama, la rama puede no existir: es M47, que cambio "+
				"«en tus respuestas no aparece» por «lo has incumplido» sin poner nada rojo.\n"+
				"  Arreglo: anadirla al censo con EL TEST QUE RENDERIZA ESA RAMA, no uno que "+
				"compruebe que la cadena esta en el catalogo.",
				k, recortarTexto(descargos[k], 120))
			continue
		}
		switch {
		case q.test != "" && q.motivo != "":
			t.Errorf("%q declara test y motivo a la vez: o hay quien lo recorre o hay hueco, "+
				"y las dos cosas juntas dejan el hueco sin contar", k)
		case q.test == "" && q.motivo == "":
			t.Errorf("%q esta en el censo con las dos casillas vacias, que es peor que no "+
				"estar: cuenta como atado y no lo esta", k)
		case q.test != "" && !nombres[q.test]:
			t.Errorf("%q dice que lo recorre %s y ese test NO EXISTE en el arbol.\n"+
				"  Un nombre plausible cumple la letra y rompe el fondo, que es la trampa "+
				"que esta familia ya se ha comido una vez.", k, q.test)
		case q.motivo != "":
			huecos = append(huecos, fmt.Sprintf("%s (%s)", k, q.motivo))
		}
	}

	// DIRECCION 2, la que envejece: una entrada del censo que ya no es descargo.
	for _, k := range clavesDelDescargo(censoDeDescargos) {
		if _, sigue := descargos[k]; !sigue {
			t.Errorf("el censo trae %q y esa clave ya no existe en el catalogo o ya no "+
				"descarga nada.\n"+
				"  Es la mitad que se olvida: un censo que cita frases que ya no estan "+
				"envejece igual que la prosa que vino a vigilar.", k)
		}
	}

	t.Logf("descargos: %d en el catalogo, %d en el censo, %d huecos declarados",
		len(descargos), len(censoDeDescargos), len(huecos))
	for _, h := range huecos {
		t.Logf("  hueco: %s", h)
	}
}

// CONTROL NEGATIVO DEL DETECTOR, que es donde esta el riesgo.
//
// Su fallo probable tiene dos caras y las dos importan. Casar de MENOS deja
// descargos sin censar, que es el agujero que esta puerta viene a tapar; casar de
// MAS mete en el censo rotulos que no prometen nada, y entonces el censo se llena
// de ruido y deja de leerse.
//
// El caso real que decidio la forma del detector: buscar «consta» daba 13 de los
// 24. Los otros 11 son descargos igual y no llevan esa palabra.
func TestElDetectorDeDescargosReconoceLaFormaYNoLaPalabra(t *testing.T) {
	casos := []struct {
		nombre string
		texto  string
		quiero bool
	}{
		{"la forma canonica, con constancia",
			"Esto NO dice que se haya incumplido: dice que en tus respuestas no consta que " +
				"se hiciera.", true},
		{"la misma forma SIN la palabra consta, que es la mitad que se escapaba",
			"Esto NO dice que la auditoria este mal hecha: dice que quien audito es la " +
				"misma persona que opera.", true},
		{"la variante del camino, que niega con otras palabras",
			"plazum guarda tus respuestas, pero de eso no se sigue que el trabajo se haya " +
				"hecho.", true},
		{"un rotulo que dice «no consta» y no promete nada",
			"no consta remitida", false},
		{"un estado vacio que pide un dato, que no es descargo",
			"No consta ningun programa de auditoria interna. Hace falta abrir uno.", false},
		{"una nota sobre el FUTURO, que no descarga de un pasado",
			"Hoy todavia no obligan, asi que no hay nada que entregar.", false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if got := formaDelDescargo.MatchString(c.texto); got != c.quiero {
				t.Errorf("el detector dice %v y esperaba %v sobre %q", got, c.quiero, c.texto)
			}
		})
	}
}

// nombresDeTestDelArbol recorre los _test.go del arbol de git y saca los nombres
// de las funciones de test.
//
// Del arbol de GIT y no del disco, por lo mismo que la puerta de los cuadernos:
// los worktrees de `.claude/` llevan copias y un nombre que solo exista alli
// contaria como existente.
func nombresDeTestDelArbol(t *testing.T) map[string]bool {
	t.Helper()
	salida, err := gitDice("ls-files", "*_test.go")
	if err != nil {
		t.Skipf("SALTADO, no comprobado: git no contesta aqui (%v). NO es verde", err)
	}
	re := regexp.MustCompile(`(?m)^func (Test[A-Za-z0-9_]+)\(`)
	out := map[string]bool{}
	n := 0
	for _, ruta := range strings.Split(salida, "\n") {
		ruta = strings.TrimSpace(ruta)
		if ruta == "" {
			continue
		}
		b, err := os.ReadFile(ruta) // #nosec G304 -- rutas del propio arbol de git
		if err != nil {
			continue
		}
		n++
		for _, m := range re.FindAllStringSubmatch(string(b), -1) {
			out[m[1]] = true
		}
	}
	if n < 50 || len(out) < 300 {
		t.Fatalf("he mirado %d ficheros de test y he encontrado %d nombres, y hoy son "+
			"cientos: este recorrido esta midiendo el vacio", n, len(out))
	}
	return out
}

func clavesDelDescargo[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func recortarTexto(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}

// LA SEGUNDA FAMILIA DEL NUCLEO QUE ALCANZA A LAS PANTALLAS: EL VALOR CERO.
//
// # Qué exige, y por qué en las superficies y no solo en el núcleo
//
// El invariante 8 dice que en una frontera de confianza el valor cero de una
// estructura de opciones tiene que ser el RESTRICTIVO, o estar prohibido
// explícitamente. `Opciones` es exactamente eso: la frontera por la que quien
// monta el servidor le pasa a una superficie sus fuentes, su catálogo y su
// camino. Un campo que puede llegar nil y no dice qué significa nil es la forma
// permisiva de la nada esperando a que alguien la descubra en producción.
//
// # El estado el 08-09-2026, medido antes de escribir la puerta
//
// **13 campos** de `Opciones` en `superficies/` pueden llegar nil o vacíos. NUEVE
// declaraban su valor cero y **CUATRO no**: `acta.Fuente`, `escalado.Fuente`,
// `uar.Fuente` y `camino.Pasos`. Los cuatro se COMPORTAN bien —se comprobó uno a
// uno, y el de `uar` con cuidado porque es la única superficie que muta: su
// `preparar` corta con `Fuente == nil` antes de llegar a `Anotar`—, así que lo
// que faltaba no era la guarda, era la declaración. Que es justo la mitad que
// nadie echa de menos hasta que alguien añade el campo catorce.
//
// # Las dos formas que admite, y las dos valen
//
//	restrictivo   «EL VALOR CERO ES no pintar nada / no guardar nada»
//	prohibido     «EL VALOR CERO ESTÁ PROHIBIDO», y se rechaza al construir
//
// Lo que no vale es callarlo. `camino.Pasos` es de la segunda clase y ahora lo
// dice: rellenarlo con el camino canónico cuando llega vacío convertiría un
// olvido en una pantalla plausible.
func TestTodoCampoQuePuedeLlegarNiloDiceQueSignificaSuValorCero(t *testing.T) {
	salida, err := gitDice("ls-files", "superficies/*.go")
	if err != nil {
		t.Skipf("SALTADO, no comprobado: git no contesta aqui (%v). NO es verde", err)
	}
	// Campos exportados de un `type Opciones struct` cuyo tipo puede ser nil o
	// vacio: interfaz, puntero o rebanada.
	reOpciones := regexp.MustCompile(`(?ms)^type Opciones struct \{(.*?)^\}`)
	reCampo := regexp.MustCompile(
		`(?m)^\t([A-Z][A-Za-z0-9]*)\s+(\*[\w.\[\]]+|\[\][\w.\[\]]+|[A-Z][\w.]*)\s*$`)

	mirados, sinDeclarar := 0, 0
	for _, ruta := range strings.Split(salida, "\n") {
		ruta = strings.TrimSpace(ruta)
		if ruta == "" || strings.HasSuffix(ruta, "_test.go") {
			continue
		}
		b, err := os.ReadFile(ruta) // #nosec G304 -- rutas del propio arbol de git
		if err != nil {
			continue
		}
		for _, bloque := range reOpciones.FindAllStringSubmatch(string(b), -1) {
			cuerpo := bloque[1]
			for _, c := range reCampo.FindAllStringSubmatchIndex(cuerpo, -1) {
				nombre := cuerpo[c[2]:c[3]]
				mirados++
				if strings.Contains(strings.ToUpper(docAnterior(cuerpo[:c[0]])), "VALOR CERO") {
					continue
				}
				sinDeclarar++
				t.Errorf("%s: el campo Opciones.%s puede llegar nil o vacio y no dice que "+
					"significa eso.\n"+
					"  `Opciones` es la frontera por la que quien monta el servidor le pasa "+
					"a esta superficie sus fuentes. Un campo que puede ser nil y no declara "+
					"su cero es la forma PERMISIVA de la nada esperando a que alguien la "+
					"descubra en produccion (invariante 8).\n"+
					"  Las dos formas valen y hay que escribir una: «EL VALOR CERO ES <lo "+
					"restrictivo>», o «EL VALOR CERO ESTA PROHIBIDO» y se rechaza al "+
					"construir.", ruta, nombre)
			}
		}
	}
	if mirados < 10 {
		t.Fatalf("he mirado %d campos de Opciones en superficies/ y hoy son mas de diez: "+
			"el patron ha dejado de casar y esto esta midiendo el vacio", mirados)
	}
	t.Logf("%d campos de Opciones que pueden llegar nil, %d sin declarar su valor cero",
		mirados, sinDeclarar)
}

// docAnterior devuelve el bloque de comentario pegado al campo, o vacio.
//
// Se para en la primera linea en blanco: sin eso, el comentario de un campo se
// leeria como el del siguiente y un campo sin documentar heredaria la
// declaracion del de arriba, que es la forma de aprobar esta puerta sin escribir
// nada.
func docAnterior(antes string) string {
	var doc []string
	lineas := strings.Split(antes, "\n")
	for i := len(lineas) - 1; i >= 0; i-- {
		l := strings.TrimSpace(lineas[i])
		switch {
		case strings.HasPrefix(l, "//"):
			doc = append([]string{l}, doc...)
		case l == "" && len(doc) == 0:
			continue // la linea en blanco que separa el campo de su doc
		default:
			return strings.Join(doc, "\n")
		}
	}
	return strings.Join(doc, "\n")
}
