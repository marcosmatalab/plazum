package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/acta"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// LA PUERTA QUE CIERRA EL HUECO ENTERO: una instalacion NUEVA puede llegar a
// tener acta.
//
// # Que estaba roto y por que ningun test lo veia
//
// El acta se compone de TRES fuentes y plazum sabia LEER las tres de disco. De
// escribirlas sabia UNA: `plazum accesos` producia la campana, y las otras dos
// no tenian ninguna orden. O sea que la mejor pantalla del producto solo se
// podia llenar componiendo dos JSON a mano, adivinando los nombres de los campos
// y el valor de `version`.
//
// Y no habia un solo test rojo, porque el test que probaba las tres fuentes
// conectadas ESCRIBIA ESOS DOS FICHEROS A MANO. Un fichero escrito a mano en un
// test es una SEGUNDA IMPLEMENTACION del formato: mide que el lector lee lo que
// el test escribe, no que el producto escriba algo que el producto lee. El dia
// que el formato cambiara, ese test habria seguido verde midiendo un formato que
// ya nadie produce.
//
// # Lo que este test hace distinto
//
// Los tres ficheros los producen LAS TRES ORDENES DE VERDAD, en el orden en el
// que las teclearia alguien, y despues se compone el acta con el mismo camino
// que usa `plazum serve`. Es la unica forma de afirmar que la cadena existe.
func TestUnaInstalacionNuevaPuedeLlegarATenerActa(t *testing.T) {
	dir := t.TempDir()
	// FUENTE 1, la que ya tenia orden: la campana de revision de accesos.
	fichero, ledger, campana := sembrarCampana(t)

	// FUENTE 2: el registro de incidentes, con la orden nueva.
	incidentes := filepath.Join(dir, "incidentes.json")
	correrIncidentesOK(t, "abrir",
		"--registro", incidentes, "--id", "INC-1",
		"--ocurrio", "2026-08-30T22:15:00Z", "--se-supo", "2026-08-31T07:40:00Z",
		"--fuente", "guardia")
	correrIncidentesOK(t, "clasificar",
		"--registro", incidentes, "--id", "INC-1",
		"--clase", "incidente.nivel.alto", "--cuando", "2026-08-31T09:00:00Z",
		"--ahora", "2026-08-31T09:05:00Z")

	// FUENTE 3: el programa de auditoria, con la otra orden nueva. Su alcance
	// SALE DEL CORPUS Y DEL ALCANCE, no de una lista tecleada.
	alcance := alcanceDelPuente(t, dir, "alcance.json", false)
	programa := filepath.Join(dir, "programa.json")
	correrAuditoriaOK(t, "abrir",
		"--programa", programa, "--id", "P-2026", "--ciclo", "2026-2028",
		"--desde", "2026-01-01", "--hasta", "2028-12-31",
		"--alcance", alcance, "--corpus", "../../paquetes")

	// Y AHORA EL ACTA, por el mismo camino que `plazum serve`.
	fuente, err := fuenteDelActa(opcionesActa{
		Organizacion: "Organismo de prueba", Desde: "2026-01-01", Hasta: "2026-12-31",
		Campana:    campanaEnFichero{fichero: fichero, ledger: ledger, id: campana},
		HayCampana: true, Incidentes: incidentes, Programa: programa,
	})
	if err != nil || fuente == nil {
		t.Fatalf("la fuente del acta no se construye con los tres ficheros que acaban de "+
			"escribir las ordenes: %v", err)
	}
	a, hay, err := fuente.Ultima()
	if err != nil || !hay {
		t.Fatalf("el acta no se compone: hay=%t err=%v", hay, err)
	}
	for _, f := range []acta.Fuente{
		acta.DelProgramaDeAuditoria, acta.DeLaCampanaDeAccesos, acta.DeLosIncidentes,
	} {
		s := seccionPorFuente(t, a, f)
		if !s.Aportada {
			t.Errorf("la seccion %v sigue diciendo que le falta la fuente (%q), y el fichero "+
				"lo acaba de escribir una orden del producto.\n"+
				"  Entonces la orden escribe algo que el lector del acta no reconoce, que es "+
				"el fallo exacto que estas ordenes vienen a cerrar", f, s.PorQueFalta)
		}
	}
}

// LOS AYUDANTES, y por que ejecutan LA ORDEN Y NO SU CUERPO.
//
// Todo lo de este fichero pasa por cmdIncidentes y cmdAuditoria, o sea por el
// mismo despacho de subordenes, el mismo parseo de banderas y los mismos
// mensajes que ve quien teclea. Llamar a las funciones internas se saltaria
// justo la mitad que falla en la practica: una bandera que no se lee, una
// suborden que no se despacha, un mensaje que no dice el arreglo.

func correrIncidentes(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var salida, errores strings.Builder
	rc := cmdIncidentes(args, &salida, &errores)
	return rc, salida.String(), errores.String()
}

func correrIncidentesOK(t *testing.T, args ...string) string {
	t.Helper()
	rc, salida, errores := correrIncidentes(t, args...)
	if rc != 0 {
		t.Fatalf("`plazum incidentes %s` ha salido %d:\n%s",
			strings.Join(args, " "), rc, errores)
	}
	return salida
}

func correrAuditoria(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var salida, errores strings.Builder
	rc := cmdAuditoria(args, &salida, &errores)
	return rc, salida.String(), errores.String()
}

func correrAuditoriaOK(t *testing.T, args ...string) string {
	t.Helper()
	rc, salida, errores := correrAuditoria(t, args...)
	if rc != 0 {
		t.Fatalf("`plazum auditoria %s` ha salido %d:\n%s",
			strings.Join(args, " "), rc, errores)
	}
	return salida
}

// corpusDePrueba carga el corpus REAL del repositorio. Se usa donde hay que
// comparar una derivacion del producto con otra: con un corpus inventado, las
// dos saldrian de la misma nada.
func corpusDePrueba(t *testing.T) []*corpus.Paquete {
	t.Helper()
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatalf("cargar el corpus: %v", err)
	}
	return ps
}

// LOS DOS EJES DE UN SUCESO SE PIDEN LOS DOS.
//
// Es el invariante 8 en su tercera forma, y aqui cuesta un plazo legal: hay
// obligaciones de notificacion que cuentan desde que la organizacion tiene
// constancia (art. 33 del Reglamento (UE) 2016/679), no desde que el incidente
// ocurrio. Rellenar el segundo con el primero moveria ese plazo sin que nadie lo
// hubiera decidido, y no daria error en ningun sitio.
func TestLosDosEjesDeUnIncidenteSePidenLosDosYNingunoRellenaAlOtro(t *testing.T) {
	registro := filepath.Join(t.TempDir(), "incidentes.json")
	rc, _, errores := correrIncidentes(t, "abrir",
		"--registro", registro, "--id", "INC-1", "--ocurrio", "2026-08-30T22:15:00Z")
	if rc == 0 {
		t.Fatal("se ha abierto un incidente sin --se-supo. Entonces alguien lo esta " +
			"rellenando con --ocurrio, y eso mueve el plazo del art. 33 sin decirlo")
	}
	if !strings.Contains(errores, "--se-supo") {
		t.Errorf("el error no nombra la bandera que falta:\n%s", errores)
	}
	if _, err := os.Stat(registro); err == nil {
		t.Error("con la orden rechazada, el registro se ha creado igual: un fichero a medias " +
			"parece un registro bueno")
	}

	// CONTROL POSITIVO: con las dos, se abre. Sin esta rama, una orden que
	// rechazara siempre pasaria la comprobacion de arriba.
	rc, _, errores = correrIncidentes(t, "abrir",
		"--registro", registro, "--id", "INC-1",
		"--ocurrio", "2026-08-30T22:15:00Z", "--se-supo", "2026-08-31T07:40:00Z")
	if rc != 0 {
		t.Fatalf("con los dos ejes puestos tampoco se abre, asi que la comprobacion de arriba "+
			"se cumple por rechazarlo todo:\n%s", errores)
	}
}

// UN REGISTRO QUE NO SE ENTIENDE NO SE MACHACA.
//
// Sobrescribir un registro de hechos ilegible pierde lo que hubiera dentro y
// ademas deja un fichero que PARECE bueno, que es lo peor de las dos cosas.
// Misma disciplina que `leerLedger` en la campana de accesos.
func TestUnRegistroDeIncidentesIlegibleNoSeMachaca(t *testing.T) {
	registro := filepath.Join(t.TempDir(), "incidentes.json")
	// Sin `version`: la forma que sale por descuido al escribirlo a mano, y la
	// que el lector rechaza a proposito.
	const roto = `{"incidentes":[{"id":"inc-1","sucesos":[]}]}`
	if err := os.WriteFile(registro, []byte(roto), 0o600); err != nil {
		t.Fatal(err)
	}
	rc, _, errores := correrIncidentes(t, "abrir",
		"--registro", registro, "--id", "INC-2",
		"--ocurrio", "2026-08-30T22:15:00Z", "--se-supo", "2026-08-31T07:40:00Z")
	if rc == 0 {
		t.Fatal("se ha escrito encima de un registro ilegible")
	}
	if !strings.Contains(errores, "NO se ha tocado") {
		t.Errorf("el error no dice que el fichero se ha dejado como estaba:\n%s", errores)
	}
	b, err := os.ReadFile(registro)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != roto {
		t.Errorf("el registro ilegible ha cambiado.\n  antes: %s\n  ahora: %s", roto, string(b))
	}
}

// LOS HECHOS SE ANADEN Y NUNCA SE EDITAN.
//
// Dos clasificaciones del mismo incidente son DOS SUCESOS, no una correccion
// encima de la otra: reclasificar es un hecho mas con su propio instante de
// registro. Si la segunda machacara a la primera, la linea de tiempo que ve un
// auditor perderia que hubo un cambio de criterio, que es justo lo que se va a
// preguntar.
func TestReclasificarAnadeUnSucesoYNoEditaElAnterior(t *testing.T) {
	registro := filepath.Join(t.TempDir(), "incidentes.json")
	correrIncidentesOK(t, "abrir", "--registro", registro, "--id", "INC-1",
		"--ocurrio", "2026-08-30T22:15:00Z", "--se-supo", "2026-08-31T07:40:00Z")
	correrIncidentesOK(t, "clasificar", "--registro", registro, "--id", "INC-1",
		"--clase", "incidente.nivel.bajo", "--cuando", "2026-08-31T09:00:00Z",
		"--ahora", "2026-08-31T09:05:00Z")
	correrIncidentesOK(t, "clasificar", "--registro", registro, "--id", "INC-1",
		"--clase", "incidente.nivel.alto", "--cuando", "2026-09-01T10:00:00Z",
		"--ahora", "2026-09-01T10:05:00Z")

	is, err := leerRegistroDeIncidentes(registro)
	if err != nil {
		t.Fatal(err)
	}
	if len(is) != 1 {
		t.Fatalf("hay %d incidentes y tenia que haber 1", len(is))
	}
	clases := []string{}
	for _, s := range is[0].Sucesos() {
		if s.Clase != "" {
			clases = append(clases, s.Clase)
		}
	}
	if len(clases) != 2 {
		t.Fatalf("constan %d clasificaciones (%v) y tenian que constar las DOS: una "+
			"reclasificacion es un suceso mas, nunca una edicion del anterior", len(clases), clases)
	}
	if clases[0] != "incidente.nivel.bajo" || clases[1] != "incidente.nivel.alto" {
		t.Errorf("las clasificaciones no salen en el orden en que ocurrieron: %v", clases)
	}
}

// UN INCIDENTE NACE UNA SOLA VEZ, y la segunda apertura no se traga en silencio.
func TestUnIncidenteNoSeAbreDosVeces(t *testing.T) {
	registro := filepath.Join(t.TempDir(), "incidentes.json")
	correrIncidentesOK(t, "abrir", "--registro", registro, "--id", "INC-1",
		"--ocurrio", "2026-08-30T22:15:00Z", "--se-supo", "2026-08-31T07:40:00Z")
	rc, _, errores := correrIncidentes(t, "abrir", "--registro", registro, "--id", "INC-1",
		"--ocurrio", "2026-08-30T22:15:00Z", "--se-supo", "2026-08-31T07:40:00Z")
	if rc == 0 {
		t.Fatal("se ha abierto dos veces el mismo incidente")
	}
	if !strings.Contains(errores, "ya consta") {
		t.Errorf("el error no dice que ya estaba:\n%s", errores)
	}
}

// UN SUCESO SOBRE UN INCIDENTE QUE NO EXISTE NO SE ESCRIBE.
//
// Y el error ENUMERA los que si hay: un mensaje que dice «no existe» y se calla
// la lista obliga a abrir el JSON, que es lo que estas ordenes existen para
// evitar.
func TestUnSucesoSobreUnIncidenteQueNoExisteSeRechazaYEnumera(t *testing.T) {
	registro := filepath.Join(t.TempDir(), "incidentes.json")
	correrIncidentesOK(t, "abrir", "--registro", registro, "--id", "INC-1",
		"--ocurrio", "2026-08-30T22:15:00Z", "--se-supo", "2026-08-31T07:40:00Z")
	rc, _, errores := correrIncidentes(t, "notificar", "--registro", registro, "--id", "INC-9",
		"--hito", "inicial", "--cuando", "2026-09-01T10:00:00Z")
	if rc == 0 {
		t.Fatal("se ha registrado una notificacion sobre un incidente que no existe")
	}
	if !strings.Contains(errores, "INC-1") {
		t.Errorf("el error no enumera los incidentes que si hay:\n%s", errores)
	}
}

// ---------------------------------------------------------------------------
// El programa de auditoria
// ---------------------------------------------------------------------------

// EL ALCANCE DEL PROGRAMA SALE DEL CORPUS Y DEL ALCANCE, NO DEL TECLADO.
//
// Es la decision de `plazum auditoria abrir`, y esta es su comprobacion: las
// unidades del programa tienen que ser EXACTAMENTE las obligaciones que el motor
// deriva con ese alcance. Una lista tecleada se queda corta el primer dia y
// vieja el segundo, y entonces habria obligaciones que el calendario vigila y el
// programa de auditoria no cubre, sin que nada lo dijera.
func TestElAlcanceDelProgramaSaleDelCorpusYNoDelTeclado(t *testing.T) {
	dir := t.TempDir()
	ruta := alcanceDelPuente(t, dir, "alcance.json", false)
	programa := filepath.Join(dir, "programa.json")
	correrAuditoriaOK(t, "abrir", "--programa", programa, "--id", "P-2026",
		"--ciclo", "2026-2028", "--desde", "2026-01-01", "--hasta", "2028-12-31",
		"--alcance", ruta, "--corpus", "../../paquetes")

	p, err := leerPrograma(programa)
	if err != nil || p == nil {
		t.Fatalf("el programa que acaba de escribir la orden no se relee: %v", err)
	}
	// LA MISMA DERIVACION, por el mismo camino del producto.
	ps := corpusDePrueba(t)
	al, err := cargarAlcance(ruta)
	if err != nil {
		t.Fatal(err)
	}
	esperadas, err := unidadesDelAlcance(ps, al)
	if err != nil {
		t.Fatal(err)
	}
	if len(esperadas) == 0 {
		t.Fatal("con el alcance del puente no se deriva ni una obligacion: este test estaria " +
			"comparando dos listas vacias")
	}
	if p.Alcance() != len(esperadas) {
		t.Errorf("el programa tiene %d unidades y la derivacion da %d",
			p.Alcance(), len(esperadas))
	}
	// SE CRUZA POR LA CLAVE (paquete|obligacion), que sale del corpus y no de
	// una cadena escrita aqui. Nunca por posicion: el alcance se reordena cada
	// vez que el corpus gana un paquete (invariante 7).
	tiene := map[string]bool{}
	for _, u := range p.Unidades() {
		tiene[u.Clave()] = true
	}
	for _, u := range esperadas {
		if !tiene[u.Clave()] {
			t.Errorf("la obligacion %s te alcanza y NO esta en el alcance del programa: seria "+
				"una obligacion que el calendario vigila y la auditoria no cubre", u.Clave())
		}
	}
}

// UN PROGRAMA NO SE ABRE ENCIMA DE OTRO.
//
// Sobrescribir uno abierto perderia sus sesiones, sus hallazgos y su arrastre,
// que son hechos. El error dice ademas como se hace lo que la persona queria
// hacer (otro fichero, con --anterior), que es lo que separa un error accionable
// de un "no".
func TestUnProgramaNoSeAbreEncimaDeOtro(t *testing.T) {
	dir := t.TempDir()
	ruta := alcanceDelPuente(t, dir, "alcance.json", false)
	programa := filepath.Join(dir, "programa.json")
	abrir := func() (int, string) {
		rc, _, errores := correrAuditoria(t, "abrir", "--programa", programa, "--id", "P-2026",
			"--ciclo", "2026-2028", "--desde", "2026-01-01", "--hasta", "2028-12-31",
			"--alcance", ruta, "--corpus", "../../paquetes")
		return rc, errores
	}
	if rc, errores := abrir(); rc != 0 {
		t.Fatalf("el primer abrir ha salido %d:\n%s", rc, errores)
	}
	rc, errores := abrir()
	if rc == 0 {
		t.Fatal("se ha abierto un programa encima de otro: se habrian perdido sus sesiones, " +
			"sus hallazgos y su arrastre")
	}
	if !strings.Contains(errores, "--anterior") {
		t.Errorf("el error no dice como se abre el ciclo siguiente:\n%s", errores)
	}
}

// EL ARRASTRE ENTRE CICLOS PASA POR LA ORDEN, y con el la mitad que se calla
// sola: las SALIDAS.
//
// Una unidad que el ciclo anterior no audito y que ya no esta en el alcance de
// este no se arrastra (se echaria de menos para siempre) y no se calla: que una
// unidad salga del alcance es una decision, y si nadie la ve, una obligacion
// desaparece del programa sin que conste que desaparecio.
func TestElArrastreEntreCiclosLlegaPorLaOrden(t *testing.T) {
	dir := t.TempDir()
	ruta := alcanceDelPuente(t, dir, "alcance.json", false)
	anterior := filepath.Join(dir, "2023.json")
	correrAuditoriaOK(t, "abrir", "--programa", anterior, "--id", "P-2023",
		"--ciclo", "2023-2025", "--desde", "2023-01-01", "--hasta", "2025-12-31",
		"--alcance", ruta, "--corpus", "../../paquetes")

	// El ciclo anterior no audito NADA, asi que todo su alcance se arrastra.
	nuevo := filepath.Join(dir, "2026.json")
	rc, salida, errores := correrAuditoria(t, "abrir", "--programa", nuevo, "--id", "P-2026",
		"--ciclo", "2026-2028", "--desde", "2026-01-01", "--hasta", "2028-12-31",
		"--alcance", ruta, "--corpus", "../../paquetes", "--anterior", anterior)
	if rc != 0 {
		t.Fatalf("abrir con arrastre ha salido %d:\n%s", rc, errores)
	}
	if !strings.Contains(salida, "arrastre del ciclo 2023-2025") {
		t.Errorf("la salida no dice de que ciclo viene el arrastre:\n%s", salida)
	}
	p, err := leerPrograma(nuevo)
	if err != nil || p == nil {
		t.Fatalf("el programa nuevo no se relee: %v", err)
	}
	a := p.DelCicloAnterior()
	if a.DeCiclo != "2023-2025" {
		t.Errorf("el arrastre dice venir de %q y venia de 2023-2025", a.DeCiclo)
	}
	if len(a.SinAuditar) == 0 {
		t.Error("el ciclo anterior no audito nada y el arrastre llega vacio: entonces la " +
			"bandera --anterior no esta haciendo nada")
	}

	// CONTROL NEGATIVO: sin --anterior el arrastre es el CERO, y eso es correcto
	// en un primer ciclo. Sin esta rama, un arrastre que llegara siempre lleno
	// pasaria la comprobacion de arriba.
	solo := filepath.Join(dir, "solo.json")
	rc, salida, _ = correrAuditoria(t, "abrir", "--programa", solo, "--id", "P-solo",
		"--ciclo", "2026-2028", "--desde", "2026-01-01", "--hasta", "2028-12-31",
		"--alcance", ruta, "--corpus", "../../paquetes")
	if rc != 0 {
		t.Fatalf("abrir sin arrastre ha salido %d", rc)
	}
	if !strings.Contains(salida, "no hay ciclo anterior") {
		t.Errorf("sin --anterior la salida no dice que no hay arrastre, y eso es distinto de "+
			"un arrastre que nadie ha mirado:\n%s", salida)
	}
	p2, err := leerPrograma(solo)
	if err != nil || p2 == nil {
		t.Fatal(err)
	}
	if len(p2.DelCicloAnterior().SinAuditar) != 0 {
		t.Error("sin --anterior el programa llega con arrastre: alguien lo esta adivinando")
	}
}

// LO NO AUDITADO SE PRESENTA COMO DATO QUE FALTA, NUNCA COMO CULPA.
//
// Es la regla de la casa en un informe de auditoria, donde acusar en falso es
// mas caro que en ningun otro sitio: quien lea «estas obligaciones se incumplen»
// en un ciclo recien abierto deja de creerse el resto del informe, y con razon.
//
// CONTROL POSITIVO Y NEGATIVO: con unidades sin auditar la frase SALE; con el
// alcance entero cubierto NO sale, o estaria descargando de algo que no se ha
// dicho.
func TestLoNoAuditadoSaleComoDatoQueFaltaYNoComoIncumplimiento(t *testing.T) {
	dir := t.TempDir()
	ruta := alcanceDelPuente(t, dir, "alcance.json", false)
	programa := filepath.Join(dir, "programa.json")
	correrAuditoriaOK(t, "abrir", "--programa", programa, "--id", "P-2026",
		"--ciclo", "2026-2028", "--desde", "2026-01-01", "--hasta", "2028-12-31",
		"--alcance", ruta, "--corpus", "../../paquetes")

	_, salida, _ := correrAuditoria(t, "ver", "--programa", programa)
	if !strings.Contains(salida, "Esto NO dice") {
		t.Errorf("un ciclo recien abierto tiene todo su alcance sin auditar y el informe no "+
			"lleva el descargo. Sin el, la cifra se lee como una acusacion de "+
			"incumplimiento:\n%s", recorta(salida, 900))
	}

	// CONTROL NEGATIVO: se audita el alcance ENTERO y la frase desaparece.
	p, err := leerPrograma(programa)
	if err != nil || p == nil {
		t.Fatal(err)
	}
	var claves []string
	for _, u := range p.Unidades() {
		claves = append(claves, u.Clave())
	}
	correrAuditoriaOK(t, "auditar", "--programa", programa, "--id", "S1",
		"--auditor", "aud-01", "--cuando", "2026-03-10T09:00:00Z",
		"--unidades", strings.Join(claves, ","), "--que", "revision documental")
	_, salida, _ = correrAuditoria(t, "ver", "--programa", programa)
	if strings.Contains(salida, "Esto NO dice") {
		t.Errorf("con el alcance entero auditado el informe sigue pintando el descargo de lo "+
			"no auditado. Entonces sale siempre y la comprobacion de arriba no mide nada:\n%s",
			recorta(salida, 900))
	}
}

// UNA CLASE DE HALLAZGO QUE NO SE RECONOCE NO CAE AL CERO.
//
// El cero es «no conformidad mayor», la mas grave. Caer ahi acusa de mas, igual
// que caer a la ultima esconderia un incumplimiento: es el invariante 8 en una
// frontera de entrada, y aqui el error va en las dos direcciones.
func TestUnaClaseDeHallazgoQueNoSeEntiendeEsUnErrorYNoLaMasGrave(t *testing.T) {
	dir := t.TempDir()
	ruta := alcanceDelPuente(t, dir, "alcance.json", false)
	programa := filepath.Join(dir, "programa.json")
	correrAuditoriaOK(t, "abrir", "--programa", programa, "--id", "P-2026",
		"--ciclo", "2026-2028", "--desde", "2026-01-01", "--hasta", "2028-12-31",
		"--alcance", ruta, "--corpus", "../../paquetes")
	p, err := leerPrograma(programa)
	if err != nil || p == nil {
		t.Fatal(err)
	}
	unidad := p.Unidades()[0].Clave()
	correrAuditoriaOK(t, "auditar", "--programa", programa, "--id", "S1",
		"--auditor", "aud-01", "--cuando", "2026-03-10T09:00:00Z", "--unidades", unidad)

	rc, _, errores := correrAuditoria(t, "anotar", "--programa", programa, "--id", "H1",
		"--sesion", "S1", "--unidad", unidad, "--clase", "grave de la muerte",
		"--texto", "algo", "--quien", "aud-01", "--cuando", "2026-03-10T12:00:00Z")
	if rc == 0 {
		t.Fatal("se ha anotado un hallazgo con una clase que no existe. Si ha caido al cero, " +
			"el hallazgo consta como NO CONFORMIDAD MAYOR sin que nadie lo haya dicho")
	}
	if !strings.Contains(errores, "no conformidad mayor") {
		t.Errorf("el error no enumera las clases que hay:\n%s", errores)
	}
	// Y NO SE HA ESCRITO NADA.
	p2, err := leerPrograma(programa)
	if err != nil || p2 == nil {
		t.Fatal(err)
	}
	if len(p2.Hallazgos()) != 0 {
		t.Errorf("con la clase rechazada consta %d hallazgos: se ha escrito igual",
			len(p2.Hallazgos()))
	}
}
