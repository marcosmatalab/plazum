package plazum

import (
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// LA PUERTA DE LA FOTO QUE SE QUEDA VIEJA.
//
// # El fallo que la trae, y es del instrumento y no del producto
//
// El 04-09-2026 el marcador publicaba 8,32 de subindice y 6,41 de global. Los
// mismos numeros que el tramo anterior, con 22 relojes nuevos, la capa visual
// entera y los cimientos de la IA dentro del arbol. Un instrumento que no se
// mueve cuando el producto se mueve mucho esta roto, y este lo estaba.
//
// LA CAUSA, medida y no supuesta: el marcador se computo entre las 00:47 y las
// 02:22, y las cuatro rebanadas del tramo aterrizaron entre las 14:47 y las
// 15:35 DEL MISMO DIA. La foto era cierta por la manana y falsa por la tarde.
//
// # Por que su puerta no lo caza, que es lo unico que importa aqui
//
// `subindice_test.go` compara `docs/marcador.md` con `docs/instantanea.md` y
// con `docs/diseno.md`, celda a celda y en las dos direcciones. Es una buena
// puerta y no tenia nada que decir: LOS TRES DOCUMENTOS ERAN CONSISTENTES ENTRE
// SI. Lo que ninguno hacia era mirar el ARBOL.
//
// O sea: la puerta vigilaba la coherencia y no la frescura, y son cosas
// distintas. Tres documentos de acuerdo entre ellos y en desacuerdo con el
// repositorio dan verde para siempre.
//
// # Lo que esta puerta hace, y por que es barata
//
// La instantanea publica cardinales que OTRAS puertas ya computan del arbol.
// Esa segunda copia es exactamente «la lista que se queda vieja» que este
// repositorio persigue en todas partes, y aqui se quedo vieja de verdad. Asi
// que la copia se ata a su origen:
//
//	relojes escritos     <- relojesDelCorpus(), el mismo contador que vigila ETAPAS.md
//	cobertura de la v1   <- el bloque cobertura-v1 del README, que YA esta atado al arbol
//	                        por TestElPorcentajeDeLaV1LoComputaUnTestYNoUnaPersona
//	puertas de CI        <- las invocaciones puerta() de .github/workflows/*.yml
//
// La cobertura se compara contra el README y no se recomputa aqui a proposito:
// recomputarla seria una TERCERA implementacion del mismo numero, y tres
// implementaciones de una cifra es como se consigue que dos esten de acuerdo y
// la que manda sea la otra. El README ya tiene puerta contra el arbol; esto
// engancha la instantanea a esa cadena.
//
// # Lo que esta puerta NO puede hacer, dicho aqui y no descubierto luego
//
// No puede exigir que una NOTA se mueva. Una nota es un juicio, y ningun test
// juzga. Lo que si hace es quitarle la excusa al juicio: con los cardinales
// atados, una nota que siga diciendo «230 relojes» no compila la frase, y quien
// la escriba tiene que mirar el numero de hoy antes de decidir si la nota se
// mueve o no.

var (
	// | Obligaciones con reloj escritas | **252** |
	reRelojesDeLaInstantanea = regexp.MustCompile(
		`(?m)^\|\s*Obligaciones con reloj escritas\s*\|\s*\*\*(\d+)\*\*\s*\|`)
	// | Puertas de CI | **25**, en 12 workflows...
	rePuertasDeLaInstantanea = regexp.MustCompile(
		`(?m)^\|\s*Puertas de CI\s*\|\s*\*\*(\d+)\*\*`)
	// **56,7 %** de cobertura estricta de la v1
	reCoberturaDeLaInstantanea = regexp.MustCompile(
		`\*\*(\d{1,3},\d) %\*\* de cobertura estricta de la v1`)
	// Del bloque del README, y solo de ese bloque.
	reCoberturaDelREADME = regexp.MustCompile(
		`(?s)<!-- cobertura-v1:inicio -->.*?\*\*(\d{1,3},\d) %\*\* de cobertura estricta` +
			`.*?<!-- cobertura-v1:fin -->`)
	// Las invocaciones puerta() de los workflows, contadas igual que en comprobar.sh.
	rePuertaEnWorkflow = regexp.MustCompile(`(?m)^\s*puerta "`)
)

// unico saca el grupo 1 de la unica coincidencia que tiene que haber. El suelo
// va aqui y no en cada llamante: una expresion que deja de casar devolveria
// «no hay numero» y esta puerta se saltaria la comprobacion en silencio, que es
// el verde vacio de siempre.
func unico(t *testing.T, re *regexp.Regexp, texto, que string) string {
	t.Helper()
	todas := re.FindAllStringSubmatch(texto, -1)
	if len(todas) == 0 {
		t.Fatalf("no encuentro %s. O la frase cambio de forma, o el bloque se movio, y en "+
			"los dos casos esta puerta estaria dando verde sin mirar nada", que)
	}
	if len(todas) > 1 {
		t.Fatalf("he encontrado %d veces %s y esperaba una. Con varias copias no se cual "+
			"manda, y vigilar la equivocada es peor que no vigilar", len(todas), que)
	}
	return todas[0][1]
}

func TestLaInstantaneaNoPublicaCardinalesQueElArbolYaDesmiente(t *testing.T) {
	inst := leerDoc(t, rutaDeLaInstantanea)

	// ---- 1. Los relojes, contra el contador del producto -----------------
	quiero := relojesDelCorpus(t)
	dice, err := strconv.Atoi(unico(t, reRelojesDeLaInstantanea, inst,
		"la fila «Obligaciones con reloj escritas» de la instantanea"))
	if err != nil {
		t.Fatalf("la fila de relojes no trae un numero: %v", err)
	}
	if dice != quiero {
		t.Errorf(`la instantanea publica %d relojes escritos y el corpus tiene %d.

  Es el fallo que trajo esta puerta: la foto se midio por la manana, el corpus
  crecio por la tarde, y los tres documentos del marcador siguieron de acuerdo
  entre ellos porque ninguno miraba el arbol.

  Arreglo: volver a medir la foto ENTERA (su cabecera dice que se rehace o no se
  hace) y despues recomputar docs/marcador.md, en ese orden. Recomputar el
  marcador sobre notas viejas solo mueve el numero sin mover lo que dice.`,
			dice, quiero)
	}

	// ---- 2. La cobertura, contra el bloque del README, que ya tiene puerta -
	readme := leerDoc(t, rutaDelREADME)
	enREADME := unico(t, reCoberturaDelREADME, readme, "la cobertura del bloque cobertura-v1 del README")
	enInst := unico(t, reCoberturaDeLaInstantanea, inst, "la cobertura estricta de la instantanea")
	if enREADME != enInst {
		t.Errorf(`la instantanea dice %s %% de cobertura de la v1 y el README dice %s %%.

  El del README esta atado al arbol por
  TestElPorcentajeDeLaV1LoComputaUnTestYNoUnaPersona, asi que el que esta mal es
  el de la instantanea. Esta es la copia que se quedo vieja el 04-09-2026,
  literalmente: la rebanada de corpus actualizo el README y no la foto.`,
			enInst, enREADME)
	}

	// ---- 3. Las puertas de CI, contra los workflows -----------------------
	nPuertas := 0
	for _, f := range ficherosDeWorkflow(t) {
		nPuertas += len(rePuertaEnWorkflow.FindAllString(f, -1))
	}
	if nPuertas == 0 {
		t.Fatal("he contado CERO invocaciones de puerta() en los workflows. O la forma de " +
			"la invocacion cambio, o el directorio se movio, y en los dos casos esta " +
			"comprobacion estaria comparando contra el vacio")
	}
	dicePuertas, err := strconv.Atoi(unico(t, rePuertasDeLaInstantanea, inst,
		"la fila «Puertas de CI» de la instantanea"))
	if err != nil {
		t.Fatalf("la fila de puertas no trae un numero: %v", err)
	}
	if dicePuertas != nPuertas {
		t.Errorf(`la instantanea publica %d puertas de CI y en los workflows hay %d.

  En las DOS direcciones: que el conjunto ENCOJA rompe igual que si crece, que
  es la unica forma de enterarse de que una puerta desaparecio.`, dicePuertas, nPuertas)
	}

	if !t.Failed() {
		t.Logf("la foto cuadra con el arbol: %d relojes, %s %% de cobertura, %d puertas",
			quiero, enInst, nPuertas)
	}
}

// EL CONTROL NEGATIVO, sobre los lectores.
//
// El fallo probable de esta puerta no es el veredicto: es que una expresion
// deje de casar y `unico` se lleve el test a Fatal... o peor, que case de mas y
// vigile la cifra equivocada. Se comprueba con texto sintetico que cada lector
// coge SU numero y no el de al lado, porque los tres documentos estan llenos de
// cifras con la misma forma.
func TestLosLectoresDeLaFotoCogenSuPropioNumero(t *testing.T) {
	casos := []struct {
		nombre string
		re     *regexp.Regexp
		texto  string
		quiero string
	}{
		{
			"los relojes no se confunden con las casillas de al lado",
			reRelojesDeLaInstantanea,
			"| Casos de test ejecutados | **2.733** |\n| Obligaciones con reloj escritas | **252** |\n",
			"252",
		},
		{
			"las puertas cogen el primer numero de su fila y no el de los workflows",
			rePuertasDeLaInstantanea,
			"| Puertas de CI | **25**, en 12 workflows. La 25 es la suite |\n",
			"25",
		},
		{
			"la cobertura del README sale de SU bloque y no de otra seccion",
			reCoberturaDelREADME,
			"antes del bloque hay un **99,9 %** de cobertura estricta que no cuenta.\n" +
				"<!-- cobertura-v1:inicio -->\n- **56,7 %** de cobertura estricta: 89 relojes\n" +
				"<!-- cobertura-v1:fin -->\n",
			"56,7",
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			m := c.re.FindAllStringSubmatch(c.texto, -1)
			if len(m) != 1 {
				t.Fatalf("he casado %d veces y esperaba 1", len(m))
			}
			if m[0][1] != c.quiero {
				t.Errorf("he leido %q y esperaba %q", m[0][1], c.quiero)
			}
		})
	}

	// Y la otra direccion: sobre un texto que NO trae la cifra, el lector tiene
	// que devolver cero coincidencias y no inventarse una.
	vacios := []struct {
		nombre string
		re     *regexp.Regexp
		texto  string
	}{
		{"relojes, sin su fila", reRelojesDeLaInstantanea, "| Otra cosa | **252** |\n"},
		{"cobertura, con el bloque sin cerrar", reCoberturaDelREADME,
			"<!-- cobertura-v1:inicio -->\n- **56,7 %** de cobertura estricta\n"},
	}
	for _, c := range vacios {
		t.Run("no casa: "+c.nombre, func(t *testing.T) {
			if m := c.re.FindAllStringSubmatch(c.texto, -1); len(m) != 0 {
				t.Errorf("ha casado %d veces sobre un texto que no trae la cifra: %v", len(m), m)
			}
		})
	}
}

// ficherosDeWorkflow devuelve el contenido de cada .yml de .github/workflows.
func ficherosDeWorkflow(t *testing.T) []string {
	t.Helper()
	rutas, err := filepath.Glob(".github/workflows/*.yml")
	if err != nil {
		t.Fatalf("no puedo enumerar los workflows: %v", err)
	}
	if len(rutas) == 0 {
		t.Fatal("cero ficheros en .github/workflows. O el directorio se movio, o el patron " +
			"dejo de casar, y esta puerta estaria contando el vacio")
	}
	out := make([]string, 0, len(rutas))
	for _, r := range rutas {
		out = append(out, leerDoc(t, r))
	}
	return out
}
