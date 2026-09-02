package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/acta"
	"github.com/marcosmatalab/plazum/superficies/camino"
	"github.com/marcosmatalab/plazum/superficies/pantallas"
	"github.com/marcosmatalab/plazum/superficies/serve"
)

// LA PUERTA DEL ACTA CON DATOS: que deje de estar en blanco cuando los hay, y
// que NO se invente lo que no tiene.
//
// POR QUE HACIA FALTA. La pantalla del acta estaba entera, con sus tests en
// verde, y `plazum serve` le pasaba `Fuente: nil` sin excepcion: no habia forma
// de que ensenara nada aunque la instalacion tuviera datos. Es el mismo fallo
// de las juntas que este bloque persigue, con la diferencia de que aqui la mitad
// que faltaba no era una comprobacion sino un adaptador.
//
// LAS DOS MITADES QUE SE COMPRUEBAN, y ninguna sobra:
//
//	CON datos, el acta ensena la campana de verdad. Sin esto, el adaptador
//	podria no estar cableado y nadie se enteraria, que es de donde venimos.
//	CON datos, las DOS secciones que no se pueden leer de disco (el programa
//	de auditoria y el registro de incidentes) siguen diciendo que su fuente no
//	esta conectada. Esta es la que de verdad importa: un acta que rellenara
//	esas dos con silencio se leeria como «no hubo hallazgos» y «no hubo
//	incidentes», y eso es acusar al reves, que es el unico error que un
//	producto de cumplimiento no puede cometer ni una vez.
//
// EL CENSO SE SIEMBRA CON LA ORDEN DE VERDAD (`plazum accesos`), no escribiendo
// un ledger a mano. Un ledger escrito aqui seria una segunda implementacion del
// formato, y el dia que el formato cambiara este test seguiria verde midiendo un
// fichero que el producto ya no escribe.

const censoDelActa = "usuario;nombre;permiso\n" +
	"u1;Ana Perez;admin\n" +
	"u1;Ana Perez;lector\n" +
	"u2;Luis Gil;lector\n"

// sembrarCampana deja en disco un CSV y un ledger con la campana abierta, y
// devuelve las tres rutas que `plazum serve` pide por bandera.
func sembrarCampana(t *testing.T) (fichero, registro, campana string) {
	t.Helper()
	dir := t.TempDir()
	fichero = filepath.Join(dir, "cuentas.csv")
	registro = filepath.Join(dir, "ledger.json")
	campana = "uar-2026-h2"
	if err := os.WriteFile(fichero, []byte(censoDelActa), 0o600); err != nil {
		t.Fatal(err)
	}
	var salida, errores strings.Builder
	rc := cmdAccesos([]string{"ver",
		"--fichero", fichero, "--sistema", "erp", "--quien", "u-042",
		"--campana", campana, "--ledger", registro,
		"--revisor", "u-099", "--ahora", "2026-09-01T09:00:00Z",
	}, &salida, &errores)
	if rc != 0 {
		t.Fatalf("sembrar la campana con la orden de verdad ha salido %d:\n%s", rc, errores.String())
	}
	return fichero, registro, campana
}

// actaConDatos monta la pantalla igual que cmdServe, con la fuente puesta.
func actaConDatos(t *testing.T) *actaDeLaInstalacion {
	t.Helper()
	fichero, registro, campana := sembrarCampana(t)
	fuente, err := fuenteDelActa(opcionesActa{
		Organizacion: "Ferretera Meridional SL",
		Desde:        "2026-07-01",
		Hasta:        "2026-12-31",
		Campana:      campanaEnFichero{fichero: fichero, ledger: registro, id: campana},
		HayCampana:   true,
	})
	if err != nil {
		t.Fatalf("la fuente del acta no se construye: %v", err)
	}
	if fuente == nil {
		t.Fatal("la fuente del acta ha salido nil con los tres datos puestos y la campana " +
			"configurada: entonces el acta seguiria en blanco con datos delante")
	}
	return fuente
}

// TestElActaEnsenaLaCampanaCuandoLaInstalacionLaTiene es la mitad positiva.
func TestElActaEnsenaLaCampanaCuandoLaInstalacionLaTiene(t *testing.T) {
	fuente := actaConDatos(t)
	a, hay, err := fuente.Ultima()
	if err != nil {
		t.Fatalf("componer el acta: %v", err)
	}
	if !hay {
		t.Fatal("con campana en disco y periodo puesto, Ultima() dice que no hay acta. La " +
			"pantalla seguiria en blanco con los datos delante, que es de donde venimos")
	}
	if a.Organizacion == "" || a.ID == "" {
		t.Errorf("el acta sale sin organizacion (%q) o sin id (%q)", a.Organizacion, a.ID)
	}
	// LAS CUATRO SECCIONES SIEMPRE. Una seccion que desaparece cuando no tiene
	// datos deja un acta que PARECE completa, y esa es la propiedad que
	// nucleo/acta promete: aqui se comprueba sobre el acta que compone el
	// adaptador de verdad, no sobre una de laboratorio.
	if len(a.Secciones) != 4 {
		t.Fatalf("el acta trae %d secciones y son cuatro siempre: programa, accesos, "+
			"incidentes y direccion", len(a.Secciones))
	}
	// Y LA DE ACCESOS ESTA APORTADA Y CON CIFRAS DE VERDAD. El censo sembrado
	// tiene tres filas, asi que esa seccion no puede salir sin numeros.
	accesos := seccionPorFuente(t, a, acta.DeLaCampanaDeAccesos)
	if !accesos.Aportada {
		t.Fatalf("la seccion de accesos sale como NO aportada teniendo la campana en disco: "+
			"%q. El adaptador no esta llegando", accesos.PorQueFalta)
	}
	if cifras(accesos) == 0 {
		t.Error("la seccion de accesos esta aportada y no trae ni una cifra, y la campana " +
			"sembrada tiene tres filas")
	}
}

// seccionPorFuente saca una seccion por SU FUENTE, que es su identidad, y nunca
// por la posicion en la lista (invariante 7). El orden de las secciones lo fija
// FuentesPosibles y no hay ninguna razon para que un test dependa de el.
func seccionPorFuente(t *testing.T, a acta.Acta, f acta.Fuente) acta.Seccion {
	t.Helper()
	for _, s := range a.Secciones {
		if s.Fuente == f {
			return s
		}
	}
	t.Fatalf("el acta no trae la seccion %v y las cuatro salen siempre", f)
	return acta.Seccion{}
}

func cifras(s acta.Seccion) int {
	n := 0
	for _, r := range s.Repartos {
		n += len(r.Cifras)
	}
	return n
}

// TestElActaNoInventaLoQueNadieLeHaDado es la mitad que importa.
//
// ACTUALIZADO el 02-09-2026, y el cambio importa: cuando se escribio, ni el
// programa de auditoria ni el registro de incidentes se podian leer de disco.
// Ahora los incidentes SI (nucleo/incidente.Reconstruir), asi que lo que este
// test afirma ya no es «no hay forma de leerlo» sino algo mas fuerte y que
// sigue siendo verdad: SIN QUE EL OPERADOR LO CONECTE, el acta no afirma nada
// sobre esa fuente. «No lo hemos mirado» y «no hubo nada» se leen al reves, y
// solo la segunda necesita que alguien haya dado el dato.
//
// El programa de auditoria sigue sin poder leerse: nucleo/auditoria no tiene
// formato en disco ni reconstruccion.
func TestElActaNoInventaLoQueNadieLeHaDado(t *testing.T) {
	fuente := actaConDatos(t)
	a, hay, err := fuente.Ultima()
	if err != nil || !hay {
		t.Fatalf("el acta no se compone: hay=%t err=%v", hay, err)
	}
	// LAS DOS SECCIONES SIGUEN AHI Y DICEN QUE LES FALTA LA FUENTE. Si
	// desaparecieran, el acta pareceria completa con dos tercios de sus fuentes
	// sin conectar; si salieran como aportadas y vacias, se leerian como «no
	// hubo hallazgos» y «no hubo incidentes», que es acusar al reves.
	for _, f := range []acta.Fuente{acta.DelProgramaDeAuditoria, acta.DeLosIncidentes} {
		s := seccionPorFuente(t, a, f)
		if s.Aportada {
			t.Errorf("la seccion %v sale como APORTADA y este montaje NO le ha dado esa "+
				"fuente. El acta esta afirmando que no hubo nada cuando lo que pasa es que "+
				"nadie se lo ha dado, que es la afirmacion mas cara que este documento "+
				"puede hacer", f)
			continue
		}
		if strings.TrimSpace(s.PorQueFalta) == "" {
			t.Errorf("la seccion %v no esta aportada y no dice que falta. Un hueco sin verbo "+
				"es un callejon (D11-b), y en el documento que va a leer un consejo es peor: "+
				"un espacio en blanco se lee como un cero", f)
		}
	}
}

// TestLaConfiguracionDelActaNoSeDegradaAPantallaVacia recorre la tercera
// respuesta de fuenteDelActa, que es la que se olvida.
//
// Con algo configurado y mal, NO se devuelve «no hay acta»: se devuelve error.
// Degradar aqui convertiria el error del operador en una pantalla plausible, y
// la plausible es la que nadie arregla. Es la tercera forma del invariante 8:
// presente y no interpretable no es la nada.
func TestLaConfiguracionDelActaNoSeDegradaAPantallaVacia(t *testing.T) {
	fichero, registro, campana := sembrarCampana(t)
	conCampana := campanaEnFichero{fichero: fichero, ledger: registro, id: campana}

	casos := []struct {
		nombre string
		o      opcionesActa
	}{
		{"periodo sin organizacion", opcionesActa{
			Desde: "2026-07-01", Hasta: "2026-12-31", Campana: conCampana, HayCampana: true}},
		{"organizacion sin periodo", opcionesActa{
			Organizacion: "X", Campana: conCampana, HayCampana: true}},
		{"fecha que no se entiende", opcionesActa{
			Organizacion: "X", Desde: "el lunes", Hasta: "2026-12-31",
			Campana: conCampana, HayCampana: true}},
		{"periodo que no avanza", opcionesActa{
			Organizacion: "X", Desde: "2026-12-31", Hasta: "2026-07-01",
			Campana: conCampana, HayCampana: true}},
		{"acta pedida sin campana", opcionesActa{
			Organizacion: "X", Desde: "2026-07-01", Hasta: "2026-12-31", HayCampana: false}},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			f, err := fuenteDelActa(c.o)
			if err == nil {
				t.Errorf("se ha aceptado una configuracion que no vale, y la pantalla saldria "+
					"como si no hubiera nada configurado (fuente=%v)", f)
			}
			if f != nil {
				t.Error("con error, la fuente tiene que ser nil: si no, quien monta podria " +
					"usarla igual")
			}
		})
	}

	// CONTROL POSITIVO EN LAS DOS RAMAS QUE SI SON VALIDAS. Sin esto, un
	// validador que dijera que no a todo pasaria los cinco casos de arriba.
	if f, err := fuenteDelActa(opcionesActa{}); err != nil || f != nil {
		t.Errorf("sin nada configurado tiene que salir (nil, nil), que es la pantalla vacia "+
			"de la puerta D11-b: f=%v err=%v", f, err)
	}
	if f, err := fuenteDelActa(opcionesActa{
		Organizacion: "X", Desde: "2026-07-01", Hasta: "2026-12-31",
		Campana: conCampana, HayCampana: true,
	}); err != nil || f == nil {
		t.Errorf("la configuracion buena tampoco se acepta, asi que el validador dice que no "+
			"a todo: f=%v err=%v", f, err)
	}
}

// EL ID DEL ACTA ES ESTABLE. Dos arranques del servidor sobre el mismo periodo
// tienen que dar el mismo id, o el expediente acabaria con dos actas distintas
// del mismo trimestre y nadie sabria cual es la buena.
func TestElIdDelActaNoCambiaEntreArranques(t *testing.T) {
	uno, dos := actaConDatos(t), actaConDatos(t)
	a1, _, err1 := uno.Ultima()
	a2, _, err2 := dos.Ultima()
	if err1 != nil || err2 != nil {
		t.Fatalf("componer: %v / %v", err1, err2)
	}
	if a1.ID != a2.ID {
		t.Errorf("dos arranques sobre el mismo periodo dan ids distintos (%q y %q)", a1.ID, a2.ID)
	}
	if a1.ID == "" {
		t.Error("el id del acta sale vacio")
	}
}

// Y LA PANTALLA MONTADA CON FUENTE SIGUE PIDIENDO SESION. El acta lleva nombres
// de actores, asi que conectar la fuente no puede haber abierto la pantalla:
// seria una fuga por la puerta de al lado del cableado nuevo.
func TestConectarLaFuenteNoAbreElActaSinSesion(t *testing.T) {
	fuente := actaConDatos(t)
	act, err := construirActa(catDePrueba(t), func(*http.Request) string { return "" }, fuente)
	if err != nil {
		t.Fatal(err)
	}
	ruta, _ := camino.RutaDe("acta")
	srv, err := serve.Nuevo(serve.Config{
		App:            montarSuperficies(pantallasVaciasDePrueba(t), montaje{prefijo: ruta, h: act}),
		CookieInsegura: true,
		Salida:         &strings.Builder{},
	})
	if err != nil {
		t.Fatal(err)
	}
	codigo, cuerpo := pedirCamino(t, srv.Handler(), ruta)
	if codigo != http.StatusUnauthorized {
		t.Errorf("GET %s sin sesion y con fuente conectada ha respondido %d, esperaba 401. "+
			"El acta lleva nombres de personas", ruta, codigo)
	}
	if strings.Contains(cuerpo, "Ana Perez") || strings.Contains(cuerpo, "Luis Gil") {
		t.Error("la pantalla sin sesion ha pintado nombres del censo: conectar la fuente ha " +
			"abierto una fuga")
	}
}

// pantallasVaciasDePrueba es la aplicacion de la raiz. Hace falta porque
// montarSuperficies exige una: sin ella el acta se montaria sola y el test
// mediria un servidor que no es el que se despliega.
func pantallasVaciasDePrueba(t *testing.T) *pantallas.Superficie {
	t.Helper()
	p, err := pantallas.Nuevo(pantallas.Opciones{Catalogo: catDePrueba(t)})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// LAS DOS FORMAS DE LA NADA EN EL ACTA, RECORRIDAS LAS DOS.
//
// Es la distincion mas cara del documento y la razon de ser del campo
// HayRegistroDeIncidentes:
//
//	SIN registro conectado   la seccion dice que su fuente no esta conectada.
//	                         plazum NO sabe si hubo incidentes.
//	CON registro y vacio     la seccion dice que no hubo incidentes en el
//	                         periodo. Es una AFIRMACION, y plazum solo la puede
//	                         hacer porque alguien le ha dado el registro.
//
// Las dos ramas se recorren con dato real. Sin la segunda, la rama de «cero
// incidentes» no la alcanza ninguna entrada y seria una rama que no existe
// (M47): la mutacion la dejaria verde porque no hay nada que romper.
func TestElActaDistingueSinRegistroDeCeroIncidentes(t *testing.T) {
	fichero, registro, campana := sembrarCampana(t)
	base := opcionesActa{
		Organizacion: "Organismo de prueba",
		Desde:        "2026-07-01", Hasta: "2026-12-31",
		Campana:    campanaEnFichero{fichero: fichero, ledger: registro, id: campana},
		HayCampana: true,
	}

	seccionDeIncidentes := func(o opcionesActa) acta.Seccion {
		t.Helper()
		f, err := fuenteDelActa(o)
		if err != nil || f == nil {
			t.Fatalf("la fuente del acta no se construye: %v", err)
		}
		a, hay, err := f.Ultima()
		if err != nil || !hay {
			t.Fatalf("el acta no se compone: hay=%t err=%v", hay, err)
		}
		return seccionPorFuente(t, a, acta.DeLosIncidentes)
	}

	// RAMA 1: sin conectar.
	sinRegistro := seccionDeIncidentes(base)
	if sinRegistro.Aportada {
		t.Error("sin registro conectado, la seccion de incidentes sale como APORTADA. " +
			"Entonces el acta esta afirmando algo sobre los incidentes del periodo sin que " +
			"nadie le haya dado el registro")
	}
	if strings.TrimSpace(sinRegistro.PorQueFalta) == "" {
		t.Error("sin registro, la seccion no dice que falta: en el documento que lee un " +
			"consejo, un espacio en blanco se lee como un cero")
	}

	// RAMA 2: conectado y vacio. Es una AFIRMACION distinta, y hay que poder
	// hacerla, porque un periodo sin incidentes es una noticia que el acta tiene
	// que poder dar.
	vacio := filepath.Join(t.TempDir(), "incidentes.json")
	if err := os.WriteFile(vacio, []byte(`{"version":1,"incidentes":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	conRegistro := base
	conRegistro.Incidentes = vacio
	ceroIncidentes := seccionDeIncidentes(conRegistro)
	if !ceroIncidentes.Aportada {
		t.Error("con el registro conectado y sin incidentes, la seccion sigue diciendo que " +
			"su fuente no esta conectada. Entonces conectar el registro no sirve de nada y " +
			"un periodo limpio no se puede contar")
	}

	// Y LAS DOS TIENEN QUE SER DISTINTAS. Si salieran iguales, el campo
	// HayRegistroDeIncidentes no estaria haciendo nada y las dos frases del acta
	// dirian lo mismo sobre dos situaciones opuestas.
	if sinRegistro.Aportada == ceroIncidentes.Aportada {
		t.Error("el acta pinta igual «no hay registro conectado» y «el registro dice que no " +
			"hubo incidentes». Son afirmaciones opuestas y solo una la puede hacer plazum")
	}
}

// UN REGISTRO CON INCIDENTES DE VERDAD LLEGA AL ACTA.
//
// Sin esta rama, todo lo de arriba se cumpliria con un lector que devolviera
// siempre la lista vacia: el acta diria «no hubo incidentes» sobre un fichero
// que trae tres.
func TestLosIncidentesDelRegistroLleganAlActa(t *testing.T) {
	fichero, registro, campana := sembrarCampana(t)
	ruta := filepath.Join(t.TempDir(), "incidentes.json")
	doc := `{"version":1,"incidentes":[
	  {"id":"inc-2026-014","sucesos":[
	    {"tipo":"apertura","instante_hecho":"2026-08-30T22:15:00Z","instante_registro":"2026-08-31T07:40:00Z","fuente":"SIEM"},
	    {"tipo":"clasificacion","clase":"incidente.nivel.alto","instante_hecho":"2026-08-31T09:00:00Z","instante_registro":"2026-08-31T09:05:00Z"}]}]}`
	if err := os.WriteFile(ruta, []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := fuenteDelActa(opcionesActa{
		Organizacion: "Organismo de prueba", Desde: "2026-07-01", Hasta: "2026-12-31",
		Campana:    campanaEnFichero{fichero: fichero, ledger: registro, id: campana},
		HayCampana: true, Incidentes: ruta,
	})
	if err != nil || f == nil {
		t.Fatalf("la fuente no se construye: %v", err)
	}
	a, hay, err := f.Ultima()
	if err != nil || !hay {
		t.Fatalf("el acta no se compone: hay=%t err=%v", hay, err)
	}
	s := seccionPorFuente(t, a, acta.DeLosIncidentes)
	if !s.Aportada {
		t.Fatal("con un registro que trae un incidente, la seccion sale como no aportada")
	}
	if cifras(s) == 0 {
		t.Error("la seccion de incidentes esta aportada y no trae ni una cifra, y el " +
			"registro trae un incidente clasificado como alto")
	}
}

// UN REGISTRO ROTO NO SE CONVIERTE EN «no hubo incidentes».
//
// Es la afirmacion mas cara que este documento puede hacer, y saldria de un
// fichero corrupto. Tiene que ser error, y el acta no se compone.
func TestUnRegistroDeIncidentesRotoNoSeLeeComoPeriodoLimpio(t *testing.T) {
	fichero, registro, campana := sembrarCampana(t)
	roto := filepath.Join(t.TempDir(), "roto.json")
	// Sin version: es la forma que sale por descuido al escribirlo a mano.
	if err := os.WriteFile(roto, []byte(`{"incidentes":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := fuenteDelActa(opcionesActa{
		Organizacion: "Organismo de prueba", Desde: "2026-07-01", Hasta: "2026-12-31",
		Campana:    campanaEnFichero{fichero: fichero, ledger: registro, id: campana},
		HayCampana: true, Incidentes: roto,
	})
	if err != nil || f == nil {
		t.Fatalf("la fuente no se construye: %v", err)
	}
	a, hay, err := f.Ultima()
	if err == nil {
		t.Fatalf("un registro ilegible ha producido un acta (hay=%t, secciones=%d). Si el "+
			"acta se compone igual, dira que no hubo incidentes basandose en un fichero que "+
			"no se entiende", hay, len(a.Secciones))
	}
	if hay {
		t.Error("con error, `hay` tiene que ser false")
	}
}

// EL ACTA CON SUS TRES FUENTES CONECTADAS.
//
// Es el estado que cierra el punto 2 de la orden: hasta hoy la mejor pantalla
// del producto salia con dos tercios diciendo que su fuente no estaba
// conectada, y no era un olvido de cableado sino que faltaban los dos lectores.
//
// Y SE COMPRUEBA LA SIMETRIA, no solo que las tres salgan llenas: la misma
// distincion entre «no conectado» y «no hubo» tiene que valer para el programa
// igual que para los incidentes, porque es la misma clase de afirmacion.
func TestElActaConLasTresFuentesConectadas(t *testing.T) {
	fichero, registro, campana := sembrarCampana(t)
	dir := t.TempDir()

	incidentes := filepath.Join(dir, "incidentes.json")
	if err := os.WriteFile(incidentes, []byte(`{"version":1,"incidentes":[
	  {"id":"inc-1","sucesos":[
	    {"tipo":"apertura","instante_hecho":"2026-08-30T22:15:00Z","instante_registro":"2026-08-31T07:40:00Z"}]}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	programa := filepath.Join(dir, "programa.json")
	if err := os.WriteFile(programa, []byte(`{"version":1,"id":"prog-2026",
	  "ciclo":{"nombre":"2026-2028","desde":"2026-01-01T00:00:00Z","hasta":"2028-12-31T00:00:00Z"},
	  "alcance":[{"paquete":"m","version":"1","obligacion":"o1","titulo":"La primera"}],
	  "sesiones":[{"id":"s1","auditor":"aud-01","cuando":"2026-03-10T09:00:00Z",
	               "unidades":["m|o1"],"alcance":"revision documental"}],
	  "hallazgos":[{"id":"h1","sesion":"s1","unidad":"m|o1","clase":"no conformidad menor",
	                "texto":"un caso sin registrar","quien":"aud-01","cuando":"2026-03-10T12:00:00Z"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	base := opcionesActa{
		Organizacion: "Organismo de prueba", Desde: "2026-01-01", Hasta: "2026-12-31",
		Campana:    campanaEnFichero{fichero: fichero, ledger: registro, id: campana},
		HayCampana: true,
	}
	componer := func(o opcionesActa) acta.Acta {
		t.Helper()
		f, err := fuenteDelActa(o)
		if err != nil || f == nil {
			t.Fatalf("la fuente no se construye: %v", err)
		}
		a, hay, err := f.Ultima()
		if err != nil || !hay {
			t.Fatalf("el acta no se compone: hay=%t err=%v", hay, err)
		}
		return a
	}

	// LAS TRES CONECTADAS: ninguna seccion de fuente puede decir que le falta.
	conTodo := base
	conTodo.Incidentes, conTodo.Programa = incidentes, programa
	a := componer(conTodo)
	for _, f := range []acta.Fuente{
		acta.DelProgramaDeAuditoria, acta.DeLaCampanaDeAccesos, acta.DeLosIncidentes,
	} {
		s := seccionPorFuente(t, a, f)
		if !s.Aportada {
			t.Errorf("con las tres fuentes conectadas, la seccion %v sigue diciendo que le "+
				"falta: %q", f, s.PorQueFalta)
		}
	}

	// LA SIMETRIA DEL PROGRAMA: sin la bandera, no aportada. Es la misma
	// distincion que la de incidentes, y si valiera solo para una de las dos el
	// acta estaria tratando distinto dos afirmaciones de la misma clase.
	soloIncidentes := base
	soloIncidentes.Incidentes = incidentes
	sinPrograma := seccionPorFuente(t, componer(soloIncidentes), acta.DelProgramaDeAuditoria)
	if sinPrograma.Aportada {
		t.Error("sin --acta-programa, la seccion del programa sale como APORTADA. El acta " +
			"estaria afirmando que no hubo hallazgos sin que nadie le haya dado el programa")
	}
	if strings.TrimSpace(sinPrograma.PorQueFalta) == "" {
		t.Error("sin programa, la seccion no dice que falta")
	}
}

// UN PROGRAMA ROTO NO SE LEE COMO «no hubo hallazgos».
func TestUnProgramaDeAuditoriaRotoNoSeLeeComoCicloLimpio(t *testing.T) {
	fichero, registro, campana := sembrarCampana(t)
	roto := filepath.Join(t.TempDir(), "roto.json")
	// Sin version: la forma que sale por descuido al escribirlo a mano.
	if err := os.WriteFile(roto, []byte(`{"id":"p","alcance":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := fuenteDelActa(opcionesActa{
		Organizacion: "Organismo de prueba", Desde: "2026-01-01", Hasta: "2026-12-31",
		Campana:    campanaEnFichero{fichero: fichero, ledger: registro, id: campana},
		HayCampana: true, Programa: roto,
	})
	if err != nil || f == nil {
		t.Fatalf("la fuente no se construye: %v", err)
	}
	if _, hay, err := f.Ultima(); err == nil {
		t.Fatalf("un programa ilegible ha producido un acta (hay=%t). Si el acta se compone "+
			"igual, dira que no hubo hallazgos basandose en un fichero que no se entiende", hay)
	}
}

// LA JUNTA QUE FALTABA: que `plazum serve` PASE de verdad cada bandera del acta.
//
// POR QUE FALTABA, y es el fallo de este bloque otra vez. Todos los tests de
// arriba llaman a `fuenteDelActa` DIRECTAMENTE con un opcionesActa construido a
// mano. Ninguno pasa por cmdServe. O sea que si una bandera estuviera mal
// escrita en el flag.String, o se declarara y no se metiera en el opcionesActa,
// TODOS SEGUIRIAN VERDES y el operador teclearia una bandera que el producto
// acepta y no lee. Cada mitad pasando su prueba, la junta sin mirar.
//
// COMO SE COMPRUEBA SIN LEVANTAR UN SERVIDOR: se le dan valores que hacen que
// `fuenteDelActa` falle, y se exige que cmdServe salga con 2. Eso solo puede
// pasar si la bandera se parseo Y llego hasta ahi, que es justo lo que se
// quiere demostrar. Un valor bueno arrancaria el servidor, asi que la rama
// negativa es la unica que se puede recorrer barata, y es la que caza el fallo.
func TestPlazumServePasaCadaBanderaDelActaHastaLaFuente(t *testing.T) {
	fichero, registro, campana := sembrarCampana(t)
	corpusReal := "../../paquetes"
	accesos := []string{
		"--accesos-fichero", fichero, "--accesos-ledger", registro, "--accesos-campana", campana,
	}
	// Una ruta que no existe, para las dos banderas de fichero.
	noExiste := filepath.Join(t.TempDir(), "no-esta.json")

	casos := []struct {
		bandera string
		args    []string
		enError string
	}{
		{"--acta-desde", append([]string{
			"--acta-organizacion", "X", "--acta-desde", "el lunes", "--acta-hasta", "2026-12-31",
		}, accesos...), "--acta-desde"},
		{"--acta-hasta", append([]string{
			"--acta-organizacion", "X", "--acta-desde", "2026-01-01", "--acta-hasta", "el jueves",
		}, accesos...), "--acta-hasta"},
		{"--acta-organizacion", append([]string{
			"--acta-desde", "2026-01-01", "--acta-hasta", "2026-12-31",
		}, accesos...), "--acta-organizacion"},
		{"--acta-incidentes", append([]string{
			"--acta-organizacion", "X", "--acta-desde", "2026-01-01", "--acta-hasta", "2026-12-31",
			"--acta-incidentes", noExiste,
		}, accesos...), "--acta-incidentes"},
		{"--acta-programa", append([]string{
			"--acta-organizacion", "X", "--acta-desde", "2026-01-01", "--acta-hasta", "2026-12-31",
			"--acta-programa", noExiste,
		}, accesos...), "--acta-programa"},
	}
	for _, c := range casos {
		t.Run(c.bandera, func(t *testing.T) {
			var salida, errores strings.Builder
			args := append([]string{"--corpus", corpusReal}, c.args...)
			rc := cmdServe(args, &salida, &errores)
			if rc != 2 {
				t.Fatalf("`plazum serve` con %s mal puesta ha salido %d y esperaba 2.\n"+
					"  Si ha salido 0 o 1, esa bandera se declara y NO llega a la fuente del "+
					"acta: el operador la teclea y el producto la acepta sin leerla.\n"+
					"  salida: %s\n  errores: %s", c.bandera, rc, salida.String(), errores.String())
			}
			if !strings.Contains(errores.String(), c.enError) {
				t.Errorf("el error no nombra %s, asi que quien lo lea no sabe que bandera "+
					"arreglar:\n%s", c.enError, errores.String())
			}
		})
	}
}
