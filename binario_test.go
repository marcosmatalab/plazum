package plazum

import (
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// LA PUERTA DEL TAMANO PUBLICADO.
//
// # Por que existe, y por que nace el mismo dia que el numero
//
// El 04-09-2026 se pidio publicar en el README lo que ocupa el binario, porque
// es el tipo de dato que un comprador tecnico busca y casi nadie da. Y el mismo
// dia se escribio la regla que gobierna esto: **todo documento que publique un
// numero lo ata a quien lo computa DEL ARBOL, nunca a otro documento.**
//
// Un tamano publicado a mano es el caso mas facil de esa familia. No hay
// segunda copia que lo contradiga, no hay puerta que lo mire, y envejece en
// silencio mientras el binario engorda: exactamente lo que acababa de pasar, que
// el binario subio 0,9 MB al meter el corpus dentro y el numero publicado
// siguio siendo el de antes.
//
// # EL TAMANO NO ES UNA PROPIEDAD DEL REPOSITORIO, Y ESTA PUERTA LO APRENDIO EN
// ROJO
//
// Esta puerta nacio comparando bytes con igualdad exacta contra lo que
// construyera la maquina que la corriera, y **puso `main` en rojo en su primer
// viaje a CI**, en dos workflows a la vez. El motivo no es un descuido de la
// comparacion: es que la afirmacion era imposible.
//
//	construido en CI, linux/amd64 nativo     12.181.688 bytes  (11,6 MB)
//	cruzado desde Windows a linux/amd64      12.730.530 bytes  (12,1 MB)
//	                                         ------------------------------
//	                                         medio mega de diferencia
//
// El tamano depende de (repositorio + cadena de herramientas + anfitrion), y un
// test del repositorio solo puede afirmar cosas del repositorio. Comparar por
// igualdad contra lo que construya CUALQUIER maquina es exigir que todas
// construyan igual, que es falso y ademas no lo dice ninguna parte.
//
// Y trajo de la mano un error PUBLICADO: el README decia que el binario habia
// subido **0,9 MB** al meter el corpus, y ese numero salia de comparar la medida
// vieja de CI (Linux) con una cruzada de Windows. **La subida real son 0,4 MB**
// (11.788.580 -> 12.181.688). Una comparacion entre dos maquinas distintas
// disfrazada de serie temporal.
//
// # Lo que se compara ahora, y por que en dos regimenes
//
//	linux/amd64 NATIVO   igualdad exacta al decimo de MB. Es la maquina que
//	                     define el numero: la misma que construye la release, y
//	                     donde corre CI. Aqui la afirmacion si es comprobable.
//	cualquier otra       banda del 5 %. No para ser laxo, sino porque desde
//	                     fuera de esa maquina lo unico honesto que se puede
//	                     decir es «no se ha ido de madre».
//
// La banda del 5 % sobre 11,6 MB son ~0,6 MB, y la diferencia entre maquinas
// medida es 0,5, o sea que cabe justa. Lo que la banda SI caza es un salto de la
// clase que importa: meter el corpus dentro fueron 0,4 MB sobre 11,2, un 3,3 %,
// asi que un cambio del doble de ese tamano se ve desde cualquier sitio.
//
// # Se compara el MB CON UN DECIMAL
//
// Un decimal de MB son 0,1 x 1024^2, o sea ~105 KB de resolucion. La version
// anterior de este comentario decia que «el MB con un decimal se traga medio
// mega», y era falso: venia de comparar con un MB SIN decimales y no se rehizo
// al anadirlo.
//
// # LO QUE ESTA PUERTA NO ES
//
// No es el presupuesto. El presupuesto (25 MB) lo comprueba CI con
// `.github/presupuesto.sh`, que ademas pasa cada medida por un control negativo
// contra un limite imposible, porque un presupuesto con 13 MB de margen no se
// ve fallar nunca. Esto es otra cosa y mas pequena: que la CIFRA PUBLICADA sea
// la de hoy.
//
// # POR QUE NO SE SALTA SI NO PUEDE CONSTRUIR
//
// Un `t.Skip` aqui convertiria «no he podido medir» en verde, que es el
// invariante 8 aplicado a esta puerta: de las dos formas de la nada, la
// peligrosa es la que sale por descuido. Si la construccion cruzada falla, esto
// se pone rojo y dice que ha fallado la construccion, que es una cosa distinta
// de «el numero esta mal» y se lee distinta.

// reTamanoDelREADME lee la cifra de su bloque, y SOLO de su bloque. El README
// esta lleno de numeros con la misma forma.
var reTamanoDelREADME = regexp.MustCompile(
	`(?s)<!-- binario:inicio -->.*?mide \*\*(\d+,\d) MB\*\* contra un presupuesto ` +
		`declarado de \*\*(\d+) MB\*\*.*?<!-- binario:fin -->`)

// EL MB DE ESTA CASA ES 1024x1024, Y SE FIJA AQUI PORQUE ES UNA AMBIGUEDAD
// SILENCIOSA.
//
// La primera version de esta puerta dividio por 10^6 y salio ROJA en su primera
// ejecucion diciendo que 12.714.146 bytes son 12,7 MB cuando el README decia
// 12,1. Los dos numeros eran «correctos» con convenios distintos, que es la peor
// forma de estar en desacuerdo porque las dos partes se creen bien.
//
// Manda el convenio que ya usaba el resto del proyecto (11.788.580 bytes se
// venian publicando como 11,2 MB, que es 1024^2). Estrictamente son MiB; se
// escriben MB porque es lo que un comprador espera leer, y esta constante es el
// sitio donde esa decision queda escrita en vez de vivir en la cabeza de nadie.
const bytesPorMega = 1024 * 1024

func TestElTamanoPublicadoDelBinarioEsElDeHoy(t *testing.T) {
	readme := leerDoc(t, rutaDelREADME)
	m := reTamanoDelREADME.FindStringSubmatch(readme)
	if m == nil {
		t.Fatal("el README no trae el bloque binario:inicio/binario:fin con la frase " +
			"«mide **X,Y MB** contra un presupuesto declarado de **Z MB**».\n" +
			"  O la frase cambio de forma, o el bloque se movio, y en los dos casos esta " +
			"puerta estaria dando verde sin mirar nada")
	}

	// LA CONSTRUCCION, con las mismas banderas que el README manda teclear.
	destino := filepath.Join(t.TempDir(), "plazum-linux")
	cmd := exec.Command("go", "build", "-ldflags=-s -w", "-trimpath", "-o", destino, "./cmd/plazum")
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	if salida, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("no he podido construir el binario de Linux (%v):\n%s\n"+
			"  Esto NO es «el numero esta mal»: es que la medida no se ha podido tomar, y "+
			"las dos cosas se parecen solo si no se miran. Sin poder construir, esta puerta "+
			"no comprueba nada y no puede darse por verde", err, salida)
	}
	fi, err := os.Stat(destino)
	if err != nil {
		t.Fatalf("he construido el binario y no puedo medirlo: %v", err)
	}
	medidos := fi.Size()

	// Suelo: un binario de tamano absurdo significa que la construccion produjo
	// otra cosa. Sin esto, un fallo silencioso daria una comparacion contra
	// basura.
	if medidos < 1_000_000 {
		t.Fatalf("el binario construido mide %d bytes, que no puede ser. La construccion ha "+
			"producido otra cosa y comparar contra eso no significaria nada", medidos)
	}

	publicados, err := strconv.ParseFloat(strings.Replace(m[1], ",", ".", 1), 64)
	if err != nil {
		t.Fatalf("la cifra publicada %q no es un numero: %v", m[1], err)
	}
	megas := float64(medidos) / bytesPorMega
	medido := strings.Replace(strconv.FormatFloat(megas, 'f', 1, 64), ".", ",", 1)

	// EL PRESUPUESTO PUBLICADO, y va ANTES de los dos regimenes porque vale
	// para los dos: se comprueba que la frase no se contradice a si misma, o sea
	// que lo medido cabe en lo que la propia frase dice que es el techo. El
	// techo de verdad (25 MB, con su control negativo) lo vigila CI.
	techo, err := strconv.Atoi(m[2])
	if err != nil {
		t.Fatalf("el presupuesto publicado no es un numero: %v", err)
	}
	if megas >= float64(techo) {
		t.Errorf("el binario mide %s MB y la propia frase dice que el presupuesto son %d MB. "+
			"La frase se contradice a si misma", medido, techo)
	}

	// LA MAQUINA QUE DEFINE EL NUMERO es linux/amd64 nativo: la misma que
	// construye la release y donde corre CI. Solo ahi la igualdad es una
	// afirmacion comprobable; fuera, exigirla seria pedir que todas las cadenas
	// de herramientas construyan igual, que no es cierto y no lo promete nadie.
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		if m[1] != medido {
			t.Errorf(`el README publica %s MB y el binario de hoy mide %s MB (%d bytes).

  Esto se ha medido en linux/amd64 NATIVO, que es la maquina que define el
  numero: la misma que construye la release. Aqui la comparacion es exacta.

  Un decimal de MB son ~105 KB, asi que no salta por cualquier cosa: si ha
  saltado, el binario ha crecido lo bastante como para que un comprador lo note.

  Arreglo: poner %s en el bloque binario:inicio del README, Y DECIR POR QUE ha
  cambiado en el mismo commit. Un binario que engorda sin explicacion es lo que
  hace que nadie se crea el resto de la pagina.`, m[1], medido, medidos, medido)
		}
		if !t.Failed() {
			t.Logf("linux/amd64 nativo: %d bytes (%s MB), exacto contra el README",
				medidos, medido)
		}
		return
	}

	// FUERA DE ESA MAQUINA, la banda. No es laxitud: es lo unico que se puede
	// afirmar desde aqui, y se dice en el mensaje para que nadie lea este verde
	// como si fuera el otro.
	const banda = 0.05
	if desvio := math.Abs(megas-publicados) / publicados; desvio > banda {
		t.Errorf(`el README publica %s MB y esta maquina construye %s MB (%d bytes): %.1f %% de desvio.

  Esta NO es linux/amd64 nativo (%s/%s), asi que la comparacion es una BANDA del
  %.0f %% y no una igualdad: el tamano depende de la cadena de herramientas y del
  anfitrion, y entre dos maquinas hay medio mega de diferencia medida.

  Que se haya pasado de la banda significa que el binario ha cambiado de tamano
  de verdad, no que tu maquina sea distinta. Constrúyelo en linux/amd64 para
  tener el numero bueno y actualiza el README con ese.`,
			m[1], medido, medidos, desvio*100, runtime.GOOS, runtime.GOARCH, banda*100)
	}
	if !t.Failed() {
		t.Logf("%s/%s: %d bytes (%s MB), dentro de la banda del %.0f %% sobre los %s MB "+
			"publicados. La igualdad exacta solo se comprueba en linux/amd64",
			runtime.GOOS, runtime.GOARCH, medidos, medido, banda*100, m[1])
	}

}

// EL CONTROL NEGATIVO, sobre el lector.
//
// El fallo probable no es el veredicto: es que la expresion coja una cifra de
// otra parte del README, que esta lleno de numeros con la misma forma. Se
// comprueba que exige sus dos marcadores y que no casa sin ellos.
func TestElLectorDelTamanoExigeSuBloque(t *testing.T) {
	const frase = "mide **12,1 MB** contra un presupuesto declarado de **25 MB**.\n"
	casos := []struct {
		nombre string
		texto  string
		casa   bool
	}{
		{"con sus dos marcadores",
			"<!-- binario:inicio -->\n" + frase + "<!-- binario:fin -->\n", true},
		{"sin el marcador de cierre no casa",
			"<!-- binario:inicio -->\n" + frase, false},
		{"la misma frase fuera del bloque no casa",
			"mide **99,9 MB** contra un presupuesto declarado de **25 MB**.\n", false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if casa := reTamanoDelREADME.FindStringSubmatch(c.texto) != nil; casa != c.casa {
				t.Errorf("ha casado %v y esperaba %v", casa, c.casa)
			}
		})
	}

	// Y que coge SU cifra y no la del presupuesto, que esta en la misma frase.
	m := reTamanoDelREADME.FindStringSubmatch(
		"<!-- binario:inicio -->\n" + frase + "<!-- binario:fin -->\n")
	if m == nil {
		t.Fatal("no ha casado el caso bueno")
	}
	if m[1] != "12,1" || m[2] != "25" {
		t.Errorf("ha leido tamano %q y techo %q, y esperaba 12,1 y 25", m[1], m[2])
	}
}
