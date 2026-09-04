package plazum

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
// # Se compara el MB CON UN DECIMAL, y esa eleccion tiene historia
//
// La primera version comparaba BYTES con igualdad exacta, con este argumento
// escrito al lado: «el MB con un decimal se traga medio mega». **Era falso, y lo
// dijo la aritmetica en la primera hora de vida de la puerta**: un decimal de MB
// son 0,1 x 1024^2, o sea ~105 KB de resolucion, no medio mega. El argumento
// venia de comparar con un MB SIN decimales y no se rehizo al anadirlo.
//
// Y el byte exacto tenia un precio que no compensaba: **obligaba a editar el
// README en cada commit que tocara codigo**. Se vio a la primera: anadir el
// almacen de valores movio el binario 16.384 bytes y puso roja una puerta que no
// tenia nada que decir sobre ese cambio. Una puerta que grita por todo se apaga,
// y una que obliga a un ritual en cada commit acaba con el ritual hecho sin
// mirar, que es peor que no tenerla.
//
// Con ~105 KB de resolucion lo que se caza es lo que importa: un crecimiento que
// un comprador notaria. El corpus dentro del binario fueron 900 KB, o sea nueve
// veces el umbral.
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

	quiero := strings.Replace(
		strconv.FormatFloat(float64(medidos)/bytesPorMega, 'f', 1, 64), ".", ",", 1)
	if m[1] != quiero {
		t.Errorf(`el README publica %s MB y el binario de hoy mide %s MB (%d bytes).

  Un tamano publicado a mano envejece en silencio mientras el binario engorda.
  Paso el 04-09-2026: la release empezo a llevar el corpus dentro, el binario
  subio 0,9 MB y el numero publicado siguio siendo el de antes.

  Un decimal de MB son ~105 KB, asi que esto NO salta por un commit normal: si
  ha saltado, el binario ha crecido lo bastante como para que un comprador lo
  note.

  Arreglo: poner %s en el bloque binario:inicio del README, Y DECIR POR QUE ha
  cambiado en el mismo commit. Un binario que engorda sin explicacion es lo que
  hace que nadie se crea el resto de la pagina.`, m[1], quiero, medidos, quiero)
	}

	// EL PRESUPUESTO PUBLICADO, contra el que vigila CI. No se recomputa aqui:
	// se comprueba que la frase no se contradice a si misma, o sea que lo
	// medido cabe en lo que la propia frase dice que es el techo.
	techo, err := strconv.Atoi(m[2])
	if err != nil {
		t.Fatalf("el presupuesto publicado no es un numero: %v", err)
	}
	if float64(medidos)/bytesPorMega >= float64(techo) {
		t.Errorf("el binario mide %s MB y la propia frase dice que el presupuesto son %d MB. "+
			"La frase se contradice a si misma", quiero, techo)
	}

	if !t.Failed() {
		t.Logf("el binario de Linux mide %d bytes (%s MB) y el README lo dice", medidos, quiero)
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
