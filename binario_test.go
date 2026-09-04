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
// silencio mientras el binario engorda: exactamente lo que acaba de pasar, que
// el binario subio 0,9 MB al meter el corpus dentro y el numero publicado
// siguio siendo el de antes.
//
// # Que se compara, y por que en bytes y no en MB
//
// Se construye el binario de Linux con las MISMAS banderas que el README manda
// teclear —si se construyera de otra forma esto mediria otra cosa— y se compara
// el numero de bytes con igualdad exacta.
//
// Se compara el BYTE y no el «12,1 MB», porque los MB con un decimal se tragan
// medio mega: el binario podria crecer 400 KB sin que el numero publicado
// cambiara, y entonces la puerta estaria mirando y no viendo. Es la misma razon
// por la que el marcador publica su numerador al lado del ponderado.
//
// El MB tambien se comprueba, contra el byte, porque los dos estan en la misma
// frase y una frase que se contradice a si misma es peor que una frase vieja.
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

// reBytesDelREADME lee los bytes de su bloque, y SOLO de su bloque. El README
// esta lleno de cifras con puntos de millar.
var reBytesDelREADME = regexp.MustCompile(
	`(?s)<!-- binario:inicio -->.*?\*\*([\d.]+) bytes \((\d+,\d) MB\)\*\* contra un presupuesto ` +
		`declarado de \*\*(\d+) MB\*\*.*?<!-- binario:fin -->`)

// milesAEntero convierte «12.714.146» en 12714146. El separador de millar es el
// punto porque el documento esta en espanol.
func milesAEntero(t *testing.T, s string) int64 {
	t.Helper()
	n, err := strconv.ParseInt(strings.ReplaceAll(s, ".", ""), 10, 64)
	if err != nil {
		t.Fatalf("no puedo leer %q como numero de bytes: %v", s, err)
	}
	return n
}

func TestElTamanoPublicadoDelBinarioEsElDeHoy(t *testing.T) {
	readme := leerDoc(t, rutaDelREADME)
	m := reBytesDelREADME.FindStringSubmatch(readme)
	if m == nil {
		t.Fatal("el README no trae el bloque binario:inicio/binario:fin con la frase " +
			"«**N bytes (X,Y MB)** contra un presupuesto declarado de **Z MB**».\n" +
			"  O la frase cambio de forma, o el bloque se movio, y en los dos casos esta " +
			"puerta estaria dando verde sin mirar nada")
	}
	publicados := milesAEntero(t, m[1])

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

	// Suelo: un binario de cero bytes, o de tamano absurdo, significa que la
	// construccion produjo otra cosa. Sin esto, un fallo silencioso daria una
	// comparacion contra basura.
	if medidos < 1_000_000 {
		t.Fatalf("el binario construido mide %d bytes, que no puede ser. La construccion ha "+
			"producido otra cosa y comparar contra eso no significaria nada", medidos)
	}

	if medidos != publicados {
		delta := medidos - publicados
		t.Errorf(`el README publica %s bytes y el binario de hoy mide %d (%+d).

  Un tamano publicado a mano envejece en silencio mientras el binario engorda.
  Paso el 04-09-2026: la release empezo a llevar el corpus dentro, el binario
  subio 0,9 MB y el numero publicado siguio siendo el de antes.

  Arreglo: poner %d en el bloque binario:inicio del README, con su MB, Y DECIR
  POR QUE ha cambiado en el mismo commit. Un binario que engorda sin
  explicacion es lo que hace que nadie se crea el resto de la pagina.`,
			m[1], medidos, delta, medidos)
	}

	// LA SEGUNDA MITAD DE LA FRASE, contra la primera. Los dos numeros viven en
	// la misma oracion y nadie los compara.
	//
	// EL MB DE ESTA CASA ES 1024x1024, Y SE DICE AQUI PORQUE ES UNA AMBIGUEDAD
	// SILENCIOSA. La primera version de esta puerta dividio por 10^6 y salio
	// roja en su primera ejecucion diciendo que 12.714.146 bytes son 12,7 MB;
	// el README decia 12,1. Los dos numeros eran «correctos» con convenios
	// distintos, que es la peor forma de estar en desacuerdo, porque las dos
	// partes se creen bien.
	//
	// Manda el convenio que ya usaba el resto del proyecto (11.788.580 bytes se
	// venian publicando como 11,2 MB, que es 1024^2), y se fija aqui para que la
	// proxima cifra no vuelva a elegir. Estrictamente son MiB; se escriben MB
	// porque es lo que un comprador espera leer, y esta linea es el sitio donde
	// esa decision queda escrita en vez de vivir en la cabeza de nadie.
	const porMega = 1024 * 1024
	mb := float64(medidos) / porMega
	quiero := strconv.FormatFloat(mb, 'f', 1, 64)
	quiero = strings.Replace(quiero, ".", ",", 1)
	if m[2] != quiero {
		t.Errorf("el README dice %s MB y %s bytes son %s MB.\n"+
			"  Los dos numeros estan en la misma frase, asi que uno de los dos esta viejo.",
			m[2], m[1], quiero)
	}

	if !t.Failed() {
		t.Logf("el binario de Linux mide %d bytes (%s MB) y el README lo dice", medidos, m[2])
	}
}

// EL CONTROL NEGATIVO, sobre el lector.
//
// El fallo probable no es el veredicto: es que la expresion coja una cifra de
// otra parte del README, que esta lleno de numeros con puntos de millar. Se
// comprueba que exige sus dos marcadores y que no casa sin ellos.
func TestElLectorDelTamanoExigeSuBloque(t *testing.T) {
	casos := []struct {
		nombre string
		texto  string
		casa   bool
	}{
		{
			"con sus dos marcadores",
			"<!-- binario:inicio -->\nmide **12.714.146 bytes (12,1 MB)** contra un " +
				"presupuesto declarado de **25 MB**.\n<!-- binario:fin -->\n",
			true,
		},
		{
			"sin el marcador de cierre no casa",
			"<!-- binario:inicio -->\nmide **12.714.146 bytes (12,1 MB)** contra un " +
				"presupuesto declarado de **25 MB**.\n",
			false,
		},
		{
			"la misma frase fuera del bloque no casa",
			"mide **99.999.999 bytes (99,9 MB)** contra un presupuesto declarado de **25 MB**.\n",
			false,
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			casa := reBytesDelREADME.FindStringSubmatch(c.texto) != nil
			if casa != c.casa {
				t.Errorf("ha casado %v y esperaba %v", casa, c.casa)
			}
		})
	}
}
