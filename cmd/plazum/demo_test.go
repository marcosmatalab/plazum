package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

// El instante fijo desde el que se calculan los relojes del demo en las
// pruebas. Sin fijarlo, un test del demo mediria el reloj de la maquina y
// fallaria un martes cualquiera.
func ahoraFijo() string { return "2026-08-25T09:00:00Z" }

func ejecutar(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var salida, errores bytes.Buffer
	codigo := cmdDemo(args, &salida, &errores)
	return salida.String(), errores.String(), codigo
}

// La puerta del tiempo hasta el valor: UN comando, sin configurar nada, y en la
// pantalla tiene que haber las cuatro cosas que hacen que esto sirva de algo.
//
// Se comprueba el CONTENIDO y no solo que no reviente, porque un demo que
// imprime cabeceras vacias termina con codigo 0 igual que uno que funciona, y
// entonces la puerta de TTFV del CI mediria el arranque de un binario en vez
// del tiempo hasta el valor.
func TestElDemoLlegaAValorConUnSoloComando(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	salida, errores, codigo := ejecutar(t, "--dir", dir, "--ahora", ahoraFijo())
	if codigo != 0 {
		t.Fatalf("plazum demo devolvio %d: %s", codigo, errores)
	}

	quiero := map[string]string{
		"el nombre de la empresa de ejemplo":  "Ferretera Meridional SL",
		"al menos una obligacion que aplica":  "APLICA  demo.",
		"al menos una que NO aplica":          "no aplica  demo.",
		"la regla que lo deriva":              ":- demo.en_ambito(",
		"un reloj con fecha":                  "vence el 2026-",
		"los casos dorados contra el motor":   "casos dorados recalculados contra el motor",
		"que hacer despues":                   "QUE HACER AHORA",
		"como deshacerlo":                     "--deshacer",
		"el descargo de asesoramiento":        "no es asesoramiento juridico",
		"que los datos de la empresa son fal": "inventados",
	}
	for que, texto := range quiero {
		if !strings.Contains(salida, texto) {
			t.Errorf("la primera pantalla no ensena %s (falta %q)", que, texto)
		}
	}
	// Los tres relojes del paquete tienen que estar corriendo, no solo uno.
	for _, ob := range []string{
		"demo.notificacion_de_incidente",
		"demo.auditoria_bienal",
		"demo.revision_trimestral_de_accesos",
	} {
		if !strings.Contains(salida, ob) {
			t.Errorf("el reloj de %s no sale en la pantalla", ob)
		}
	}
	// Y el mas urgente sale el primero: un plazo de 72 horas por debajo de una
	// auditoria bienal es una pantalla ordenada por casualidad. Se mira DENTRO
	// de la seccion de relojes: fuera de ella el orden es otro, y comparar
	// posiciones sobre el texto entero daria verde por el sitio equivocado.
	corte := strings.Index(salida, "3. LOS RELOJES")
	if corte < 0 {
		t.Fatal("no hay seccion de relojes en la pantalla")
	}
	relojes := salida[corte:]
	iUrgente := strings.Index(relojes, "demo.notificacion_de_incidente\n")
	iBienal := strings.Index(relojes, "demo.auditoria_bienal\n")
	if iUrgente < 0 || iBienal < 0 || iUrgente > iBienal {
		t.Errorf("los relojes no salen por urgencia: el de 72 horas esta en %d y el bienal en %d",
			iUrgente, iBienal)
	}
}

// Un demo que ensucia la instalacion de verdad no lo ejecuta nadie dos veces.
func TestElDemoNoEscribeNadaFueraDeSuDirectorio(t *testing.T) {
	base := t.TempDir()
	vecino := filepath.Join(base, "instalacion-de-verdad")
	if err := os.MkdirAll(vecino, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vecino, "keystore.json"), []byte("secreto"), 0o600); err != nil {
		t.Fatal(err)
	}
	antes := arbol(t, base)

	dir := filepath.Join(base, "demo")
	if _, errores, codigo := ejecutar(t, "--dir", dir, "--ahora", ahoraFijo()); codigo != 0 {
		t.Fatalf("plazum demo devolvio %d: %s", codigo, errores)
	}
	// Todo lo nuevo tiene que estar dentro de dir.
	for ruta := range diferencia(arbol(t, base), antes) {
		if ruta != "demo" && !strings.HasPrefix(ruta, "demo/") {
			t.Errorf("el demo escribio %q, que esta fuera de su directorio", ruta)
		}
	}
	// Y despues de deshacer, el arbol vuelve a estar exactamente como estaba.
	if _, errores, codigo := ejecutar(t, "--dir", dir, "--deshacer"); codigo != 0 {
		t.Fatalf("plazum demo --deshacer devolvio %d: %s", codigo, errores)
	}
	if despues := arbol(t, base); !mismoArbol(despues, antes) {
		t.Errorf("tras deshacer queda algo del demo:\nantes:   %v\ndespues: %v", antes, despues)
	}
}

func TestElDemoSeNiegaAEscribirEnUnDirectorioQueNoHizoEl(t *testing.T) {
	ajeno := filepath.Join(t.TempDir(), "instalacion")
	if err := os.MkdirAll(ajeno, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ajeno, "keystore.json"), []byte("secreto"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, errores, codigo := ejecutar(t, "--dir", ajeno, "--ahora", ahoraFijo())
	if codigo == 0 {
		t.Fatal("el demo escribio dentro de un directorio con contenido que no creo el")
	}
	if !strings.Contains(errores, "--dir") {
		t.Errorf("el error no dice como salir del paso: %s", errores)
	}
	if _, err := os.Stat(filepath.Join(ajeno, "keystore.json")); err != nil {
		t.Errorf("el demo toco el contenido ajeno: %v", err)
	}
}

func TestDeshacerSeNiegaSiElDirectorioNoLoHizoElDemo(t *testing.T) {
	ajeno := filepath.Join(t.TempDir(), "importante")
	if err := os.MkdirAll(ajeno, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ajeno, "datos.db"), []byte("la base de datos"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, errores, codigo := ejecutar(t, "--dir", ajeno, "--deshacer")
	if codigo == 0 {
		t.Fatal("--deshacer borro un directorio que no creo el demo. Es un RemoveAll sobre una " +
			"ruta que teclea el operador: un accidente esperando a pasar")
	}
	if _, err := os.Stat(filepath.Join(ajeno, "datos.db")); err != nil {
		t.Fatalf("y ademas borro el contenido: %v", err)
	}
	if !strings.Contains(errores, nombreMarcaDemo) {
		t.Errorf("el error no explica por que se niega: %s", errores)
	}
}

func TestElDemoDosVecesSeguidasDaLoMismo(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	primera, _, codigo := ejecutar(t, "--dir", dir, "--ahora", ahoraFijo())
	if codigo != 0 {
		t.Fatal("la primera ejecucion fallo")
	}
	segunda, errores, codigo := ejecutar(t, "--dir", dir, "--ahora", ahoraFijo())
	if codigo != 0 {
		t.Fatalf("la segunda ejecucion fallo con %d: %s. Un demo que solo se puede ejecutar una "+
			"vez obliga a limpiar a mano antes de volver a ensenarlo", codigo, errores)
	}
	if primera != segunda {
		t.Error("dos ejecuciones con el mismo instante dan pantallas distintas: hay algo no " +
			"determinista, y un demo que cambia solo no se puede ensenar a un cliente")
	}
}

// Pedir la ayuda no es un fallo. Si `--help` termina con codigo distinto de
// cero, cualquier script que lo llame se cree que algo ha ido mal, y el primer
// script que llama a un binario nuevo suele ser el de instalacion de alguien.
func TestPedirLaAyudaNoEsUnFallo(t *testing.T) {
	casos := map[string]func([]string, io.Writer, io.Writer) int{
		"demo":   cmdDemo,
		"doctor": cmdDoctor,
		"update": cmdUpdate,
	}
	for nombre, cmd := range casos {
		t.Run(nombre, func(t *testing.T) {
			var salida, errores bytes.Buffer
			if codigo := cmd([]string{"--help"}, &salida, &errores); codigo != 0 {
				t.Errorf("`plazum %s --help` termino con %d", nombre, codigo)
			}
			if errores.Len() == 0 && salida.Len() == 0 {
				t.Errorf("`plazum %s --help` no imprimio nada", nombre)
			}
		})
	}
}

func TestUnInstanteInvalidoSeRechazaConUnEjemplo(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	_, errores, codigo := ejecutar(t, "--dir", dir, "--ahora", "el martes")
	if codigo == 0 {
		t.Fatal("se acepto una fecha que no es una fecha")
	}
	if !strings.Contains(errores, "2026-") {
		t.Errorf("el error no ensena el formato que si vale: %s", errores)
	}
}

// Cargar el corpus real ADEMAS del demo tiene que funcionar y, sobre todo, no
// puede convertir la pantalla de valor en un volcado de trescientas lineas.
func TestElDemoConElCorpusRealNoVuelcaElCatalogoEntero(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "demo")
	salida, errores, codigo := ejecutar(t, "--dir", dir, "--ahora", ahoraFijo(),
		"--corpus", filepath.Join("..", "..", "paquetes"))
	if codigo != 0 {
		t.Fatalf("con el corpus real el demo devolvio %d: %s", codigo, errores)
	}
	if n := strings.Count(salida, "   no aplica  "); n > 9 {
		t.Errorf("la pantalla lista %d obligaciones no aplicables: con el corpus real son "+
			"cientos y la primera pantalla deja de serlo", n)
	}
	if !strings.Contains(salida, "y ") || !strings.Contains(salida, "mas") {
		t.Error("la lista se acota y no dice cuantas se ha dejado fuera")
	}
	// Y se dice por que salen como no aplicables, que si no parece que el
	// corpus real no funcione.
	if !strings.Contains(salida, "SU alcance") {
		t.Error("no se explica que las obligaciones del corpus real no aplican porque nadie ha " +
			"respondido su alcance, y eso se lee como que el producto no las ve")
	}
	if !strings.Contains(salida, "casos dorados recalculados contra el motor") {
		t.Error("con el corpus real cargado, sus casos dorados tambien tienen que recalcularse")
	}
}

// ---------------------------------------------------------------------------
// La puerta de la traduccion del reloj
// ---------------------------------------------------------------------------

// Este test es la razon por la que se puede permitir que VencimientosDe viva
// aqui en vez de en nucleo/corpus.
//
// La traduccion de Temporalidad a primitiva de ventana existe ya dentro de
// corpus/dorados.go, sin exportar. Escribirla otra vez aqui es una duplicacion,
// y una duplicacion que se puede desviar en silencio seria peor que el problema
// que resuelve. Asi que ESTA funcion se ejecuta contra TODOS los casos dorados
// del corpus publicado, que son los derivados del texto legal: si las dos
// derivas se separan aunque sea en un segundo, esto se pone rojo.
func TestLaTraduccionDelRelojReproduceTodosLosDoradosDelCorpus(t *testing.T) {
	ps, err := corpus.Cargar(filepath.Join("..", "..", "paquetes"))
	if err != nil {
		t.Fatalf("cargar el corpus publicado: %v", err)
	}
	comprobados := 0
	for _, p := range ps {
		idx := map[string]corpus.Obligacion{}
		for _, o := range p.Obligaciones {
			idx[o.ID] = o
		}
		for _, d := range p.Dorados {
			o, ok := idx[d.Obligacion]
			if !ok {
				t.Errorf("%s: el dorado %q apunta a una obligacion que no existe", p.URN, d.Caso)
				continue
			}
			if err := casaElDorado(o, d, 0); err != nil {
				t.Errorf("%s: %v", p.URN, err)
			}
			comprobados++
		}
	}
	if comprobados == 0 {
		t.Fatal("no se ha comprobado ningun dorado: el corpus se ha quedado sin relojes o la " +
			"carga se ha roto, y esta puerta estaria en verde sin comprobar nada")
	}
	t.Logf("%d casos dorados reproducidos por la traduccion del CLI", comprobados)
}

// El control negativo: se demuestra que la comparacion de arriba TIENE dientes.
// Se desplaza el esperado una hora y se exige que salte. Sin esto, un verde no
// prueba que la traduccion coincida, solo que el bucle no revienta.
func TestLaComprobacionDeLaTraduccionSaltaCuandoDebe(t *testing.T) {
	ps, err := corpus.Cargar(filepath.Join("..", "..", "paquetes"))
	if err != nil {
		t.Fatal(err)
	}
	saltos, casos := 0, 0
	for _, p := range ps {
		idx := map[string]corpus.Obligacion{}
		for _, o := range p.Obligaciones {
			idx[o.ID] = o
		}
		for _, d := range p.Dorados {
			o, ok := idx[d.Obligacion]
			if !ok {
				continue
			}
			casos++
			if err := casaElDorado(o, d, time.Hour); err != nil {
				saltos++
			}
		}
	}
	if casos == 0 {
		t.Fatal("no hay dorados con los que hacer el control negativo")
	}
	if saltos != casos {
		t.Fatalf("desplazando el esperado una hora, la comprobacion tenia que fallar en los %d "+
			"casos y solo fallo en %d: no distingue una fecha correcta de una incorrecta",
			casos, saltos)
	}
}

// casaElDorado calcula el reloj con la traduccion del CLI y lo compara con el
// esperado del dorado, mas un desvio opcional que sirve de control negativo.
func casaElDorado(o corpus.Obligacion, d corpus.Dorado, desvio time.Duration) error {
	hechos := ventana.Hechos{}
	for clave, valor := range d.Hechos {
		t, err := resolverFecha(valor, time.Time{})
		if err != nil {
			return fmt.Errorf("dorado %q: el hecho %q (%q) no es una fecha: %v", d.Caso, clave, valor, err)
		}
		hechos[clave] = t
	}
	// El esperado de un dorado es la LISTA exhaustiva de vencimientos. Aqui se
	// comprueba solo que la traduccion del CLI reproduce cada FECHA declarada,
	// que es lo que esta puerta vigila: que las dos derivaciones de la misma
	// Temporalidad no se separen. La exhaustividad (que no sobre ni falte
	// ningun hito) la comprueba corpus.EjecutarDorado, que es el ejecutor de
	// verdad; duplicarla aqui seria la segunda lectura del mismo dato que este
	// fichero existe para cazar.
	var ultima time.Time
	type fila struct {
		hito  string
		vence time.Time
	}
	var filas []fila
	for _, esp := range d.Esperado {
		if esp.Vence == "" {
			continue // pendiente de hecho o sin plazo legal: no hay fecha que cotejar
		}
		t, err := resolverFecha(esp.Vence, time.Time{})
		if err != nil {
			return fmt.Errorf("dorado %q: esperado.vence %q ilegible: %v", d.Caso, esp.Vence, err)
		}
		t = t.Add(desvio)
		filas = append(filas, fila{hito: esp.Hito, vence: t})
		if t.After(ultima) {
			ultima = t
		}
	}
	if len(filas) == 0 {
		// UN ESPERADO SIN NI UNA FECHA ES LEGITIMO, y hasta el 29-08-2026 no lo
		// era: una reapertura por evento sale SIN PLAZO LEGAL, o sea sin fecha,
		// y con la regla anterior esos 22 dorados daban error aqui. Devolver un
		// error los habria dejado fuera; saltarselos los habria dejado sin
		// comprobar por el CLI, que es peor.
		//
		// Lo que se compara entonces es el ESTADO por hito, que es exactamente
		// lo que hay que reproducir cuando no hay fecha que cotejar.
		return casanLosEstados(o, d, hechos, desvio)
	}

	vs, err := VencimientosDe(o, hechos, ultima.Add(24*time.Hour))
	if err != nil {
		return fmt.Errorf("dorado %q: %v", d.Caso, err)
	}
	// El emparejamiento es POR HITO, que es una identidad dentro del dato, no
	// por indice ni por orden (invariante 7).
	for _, f := range filas {
		casa := false
		for _, v := range vs {
			if v.Hito == f.hito && v.Vence.Equal(f.vence) {
				casa = true
				break
			}
		}
		if !casa {
			return fmt.Errorf("dorado %q (obligacion %s): la traduccion del CLI no da %s para "+
				"el hito %q. Las dos derivaciones de la misma Temporalidad se han separado, "+
				"que es exactamente lo que esta puerta existe para cazar",
				d.Caso, o.ID, f.vence.Format(time.RFC3339), f.hito)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Fechas relativas del alcance
// ---------------------------------------------------------------------------

func TestLasFechasDelAlcanceAdmitenDesplazamientoYFechaFija(t *testing.T) {
	ahora := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	casos := []struct {
		texto  string
		quiero time.Time
	}{
		{"-45d", ahora.AddDate(0, 0, -45)},
		{"+10d", ahora.AddDate(0, 0, 10)},
		{"-30h", ahora.Add(-30 * time.Hour)},
		{"2026-01-15", time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)},
		{"2026-01-15T08:30:00Z", time.Date(2026, 1, 15, 8, 30, 0, 0, time.UTC)},
	}
	for _, c := range casos {
		got, err := resolverFecha(c.texto, ahora)
		if err != nil {
			t.Errorf("%q: %v", c.texto, err)
			continue
		}
		if !got.Equal(c.quiero) {
			t.Errorf("%q dio %s y tenia que dar %s", c.texto, got, c.quiero)
		}
	}
	for _, malo := range []string{"", "-", "-45", "-45x", "-x5d", "el martes", "45d"} {
		if _, err := resolverFecha(malo, ahora); err == nil {
			t.Errorf("%q se acepto como fecha", malo)
		}
	}
}

// Las fechas del alcance del demo son RELATIVAS a proposito. Si alguien las
// congela, el demo envejece y a los seis meses ensena tres relojes vencidos,
// que es lo contrario de lo que tiene que ensenar.
func TestElAlcanceDelDemoNoLlevaFechasCongeladas(t *testing.T) {
	al, err := cargarAlcanceDemo()
	if err != nil {
		t.Fatal(err)
	}
	if len(al.Fechas) == 0 {
		t.Fatal("el alcance del demo no trae ninguna fecha: sin fechas no hay relojes corriendo")
	}
	for clave, valor := range al.Fechas {
		if valor == "" || (valor[0] != '-' && valor[0] != '+') {
			t.Errorf("la fecha %q del demo es fija (%q). Un demo con fechas fijas envejece: "+
				"escribela como desplazamiento (-45d, -30h) para que los relojes corran siempre",
				clave, valor)
		}
	}
	// Y con un instante cualquiera del futuro, los relojes siguen sin estar
	// todos vencidos: es la propiedad que las fechas relativas compran.
	dir := filepath.Join(t.TempDir(), "demo")
	salida, _, codigo := ejecutar(t, "--dir", dir, "--ahora", "2031-04-17T11:00:00Z")
	if codigo != 0 {
		t.Fatalf("el demo no funciona en 2031 (codigo %d)", codigo)
	}
	if strings.Contains(salida, "VENCIDO") {
		t.Error("cinco anos despues el demo ensena relojes vencidos: las fechas no son relativas " +
			"de verdad")
	}
}

// ---------------------------------------------------------------------------
// Utilidades del test
// ---------------------------------------------------------------------------

func arbol(t *testing.T, raiz string) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	err := filepath.WalkDir(raiz, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(raiz, p)
		if err != nil {
			return err
		}
		if rel != "." {
			out[filepath.ToSlash(rel)] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func diferencia(a, b map[string]bool) map[string]bool {
	out := map[string]bool{}
	for k := range a {
		if !b[k] {
			out[k] = true
		}
	}
	return out
}

func mismoArbol(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// casanLosEstados compara la traduccion del CLI con un esperado que no trae ni
// una fecha: se emparejan los hitos y se exige el mismo estado.
//
// El emparejamiento es POR HITO, que es identidad dentro del dato, no por
// indice ni por orden (invariante 7). Y se recorre en las DOS direcciones: ni
// un hito de menos ni uno de mas.
// `desvio` es el control negativo. Un esperado sin fechas no se puede desplazar
// una hora, asi que la mutacion tiene que ser otra: se estropea el ESTADO
// esperado y se exige que la comparacion lo note.
//
// Lo saco el propio control negativo, que se puso rojo diciendo «tenia que
// fallar en los 314 casos y solo fallo en 292». Los 22 que faltaban eran justo
// los nuevos, los que no tienen fecha. Un control negativo que no sabe mutar
// una clase de caso deja esa clase sin demostrar, y lo dijo el solo.
func casanLosEstados(o corpus.Obligacion, d corpus.Dorado, hechos ventana.Hechos,
	desvio time.Duration) error {
	hasta, err := resolverFecha(d.Hasta, time.Time{})
	if err != nil {
		return fmt.Errorf("dorado %q: `hasta` ilegible (%q): %v", d.Caso, d.Hasta, err)
	}
	vs, err := VencimientosDe(o, hechos, hasta)
	if err != nil {
		return fmt.Errorf("dorado %q: %v", d.Caso, err)
	}
	delMotor := map[string]string{}
	for _, v := range vs {
		delMotor[v.Hito] = v.Estado.String()
	}
	delTexto := map[string]string{}
	for _, e := range d.Esperado {
		est := e.Estado
		if est == "" {
			est = "determinado"
		}
		if desvio != 0 {
			est = "estado-mutado-" + est
		}
		delTexto[e.Hito] = est
	}
	for hito, quiero := range delTexto {
		tengo, ok := delMotor[hito]
		if !ok {
			return fmt.Errorf("dorado %q: el texto espera el hito %q y la traduccion del CLI "+
				"no lo devuelve", d.Caso, hito)
		}
		if tengo != quiero {
			return fmt.Errorf("dorado %q, hito %q: el texto dice %q y la traduccion del CLI "+
				"dice %q", d.Caso, hito, quiero, tengo)
		}
	}
	for hito := range delMotor {
		if _, ok := delTexto[hito]; !ok {
			return fmt.Errorf("dorado %q: la traduccion del CLI devuelve el hito %q, que el "+
				"texto no espera", d.Caso, hito)
		}
	}
	return nil
}
