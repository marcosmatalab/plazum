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

// TestElActaNoInventaLasDosFuentesQueNoSabeLeer es la mitad que importa.
//
// El programa de auditoria y el registro de incidentes NO se pueden leer de
// disco hoy (nucleo/auditoria y nucleo/incidente no tienen formato ni orden que
// lo escriba). El acta tiene que decirlo, no callarlo: «no lo hemos mirado» y
// «no hubo nada» se leen al reves y solo uno es verdad.
func TestElActaNoInventaLasDosFuentesQueNoSabeLeer(t *testing.T) {
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
			t.Errorf("la seccion %v sale como APORTADA, y hoy no hay forma de leer esa fuente "+
				"de disco (nucleo/auditoria y nucleo/incidente no tienen formato ni orden que "+
				"lo escriba). O se ha conectado, y entonces hay que actualizar esta puerta y "+
				"el comentario de serve_acta.go, o el acta esta afirmando que no hubo nada "+
				"cuando lo que pasa es que no se ha mirado", f)
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
