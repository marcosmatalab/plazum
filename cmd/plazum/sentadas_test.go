package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// La seccion de sentadas sale, y sale ANTES que el listado por meses.
//
// El orden es la decision de producto y por eso se comprueba: lo primero que
// lee un CISO tiene que ser cuantas veces hay que sentarse, no cuantas casillas
// hay. Si algun dia alguien mueve la seccion detras del listado, esto se pone
// rojo y hay que decidirlo a proposito.
func TestLasSentadasSalenAntesQueElListadoPorMeses(t *testing.T) {
	out, _, codigo := correrCalendario(t,
		"--corpus=../../paquetes", "--pais=ES", "--sector=servicios-digitales", "--empleados=200")
	if codigo != 0 {
		t.Fatalf("codigo %d\n%s", codigo, out)
	}
	iSent := strings.Index(out, "LAS SENTADAS")
	if iSent < 0 {
		t.Fatalf("no sale la seccion de sentadas:\n%s", out)
	}
	iCuenta := strings.Index(out, "LA CUENTA, ENTERA")
	if iCuenta < 0 || iSent > iCuenta {
		t.Error("las sentadas tienen que salir antes de la contabilidad final")
	}
	// El titular dice las tres cosas: obligaciones, marcos y sentadas.
	for _, quiero := range []string{"obligaciones periodicas", "marcos", "sentadas al ano"} {
		if !strings.Contains(out, quiero) {
			t.Errorf("el titular no dice %q", quiero)
		}
	}
}

// LA PIEZA QUE JUSTIFICA TODO ESTO: una sentada que cubre DOS marcos.
//
// Es composicion entre marcos computada, no prometida: la revision
// independiente del SGSI y la del punto 2.3.4 del anexo del reglamento tecnico
// de NIS2 arrancan del MISMO hecho registrado y caen el mismo mes. Un catalogo
// de controles no sabe decir eso, porque no tiene reloj.
//
// El test no cablea los dos marcos: exige que EXISTA alguna sentada con mas de
// uno. Cablearlos ataria esta comprobacion al corpus de hoy, y lo que se vigila
// es la propiedad, no el par concreto.
func TestAlgunaSentadaCubreMasDeUnMarco(t *testing.T) {
	out, _, codigo := correrCalendario(t,
		"--corpus=../../paquetes", "--pais=ES", "--sector=servicios-digitales", "--empleados=200")
	if codigo != 0 {
		t.Fatalf("codigo %d\n%s", codigo, out)
	}
	if !strings.Contains(out, "de 2 marcos") && !strings.Contains(out, "de 3 marcos") {
		t.Errorf(`ninguna sentada cubre mas de un marco.

  Esta es la unica linea de esta pantalla que un catalogo de controles no puede
  escribir: dos obligaciones de marcos distintos que se despachan en la misma
  sesion porque arrancan del mismo hecho. Si desaparece, o el corpus ha perdido
  la obligacion que la producia o la agrupacion ha dejado de cruzar marcos.
%s`, out)
	}
}

// El consejo de agrupar NUNCA propone mover lo que no se puede mover.
//
// Con --sentadas, cada fecha dice si es adelantable. La palabra sale del
// `origen_del_intervalo` del corpus, que es un dato del paquete y no una
// opinion de la pantalla.
func TestElDetalleDiceQueSePuedeAdelantarYQueNo(t *testing.T) {
	out, _, codigo := correrCalendario(t,
		"--corpus=../../paquetes", "--pais=ES", "--sector=servicios-digitales",
		"--empleados=200", "--sentadas")
	if codigo != 0 {
		t.Fatalf("codigo %d\n%s", codigo, out)
	}
	if !strings.Contains(out, "(adelantable)") {
		t.Errorf("con --sentadas ninguna fecha dice si se puede adelantar:\n%s", out)
	}

	// CONTROL: sin la bandera, el detalle NO sale. Una bandera que no cambia
	// nada es una bandera que no existe.
	sinBandera, _, _ := correrCalendario(t,
		"--corpus=../../paquetes", "--pais=ES", "--sector=servicios-digitales", "--empleados=200")
	if strings.Contains(sinBandera, "(adelantable)") {
		t.Error("el detalle de --sentadas sale sin pedirlo: la bandera no hace nada")
	}
	if !strings.Contains(sinBandera, "LAS SENTADAS") {
		t.Error("el resumen de sentadas tiene que salir SIEMPRE; lo que la bandera anade " +
			"es el detalle")
	}
}

// Un ciclo cuenta OBLIGACIONES, no fechas, y eso se ve en la pantalla.
//
// Con el corpus de hoy, el ciclo anual agrupa muchas mas obligaciones que
// fechas hay en la ventana, porque casi todas esperan un dato del operador. Si
// alguna vez la seccion empezara a contar fechas, el numero se dispararia y
// esta comprobacion lo veria.
func TestElCicloAnualAgrupaObligacionesQueTodaviaNoTienenFecha(t *testing.T) {
	out, _, codigo := correrCalendario(t,
		"--corpus=../../paquetes", "--pais=ES", "--sector=servicios-digitales", "--empleados=200")
	if codigo != 0 {
		t.Fatalf("codigo %d\n%s", codigo, out)
	}
	if !strings.Contains(out, "esperando un dato tuyo") {
		t.Errorf(`la seccion no cuenta los relojes que esperan un dato del operador.

  Son el estado normal de TODA cadencia el dia uno de un cliente, y es justo el
  dia en que mas falta hace saber cuantas veces al ano habra que sentarse. Un
  ciclo existe aunque su primera fecha no se pueda calcular todavia.
%s`, out)
	}
}

// Lo vencido sale ARRIBA DEL TODO, antes incluso que las sentadas.
//
// De todo lo que este calendario puede decir, un incumplimiento en curso es lo
// unico que no admite planificacion: ya ha pasado. Y va con su descargo, porque
// el producto NO sabe si se hizo: sabe que no consta.
func TestLoVencidoSaleArribaDelTodoYConSuDescargo(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "alcance.json")
	if err := os.WriteFile(ruta, []byte(`{
  "organizacion": "Prueba de vencidos",
  "sujeto": "acme",
  "hechos": [{"pred": "papel_nis2_tecnica", "args": ["acme", "entidad_pertinente"]}],
  "fechas": {"ultima_revision_de_roles_y_responsabilidades": "2025-01-15"}
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	out, _, codigo := correrCalendario(t, "--corpus=../../paquetes", "--alcance="+ruta)
	if codigo != 0 {
		t.Fatalf("codigo %d\n%s", codigo, out)
	}
	iVenc := strings.Index(out, "YA VENCIDO")
	if iVenc < 0 {
		t.Fatalf("una obligacion anual hecha por ultima vez en enero de 2025, mirada en "+
			"agosto de 2026, tiene que salir como vencida:\n%s", out)
	}
	if iSent := strings.Index(out, "LAS SENTADAS"); iSent >= 0 && iVenc > iSent {
		t.Error("lo vencido tiene que salir antes que las sentadas: un incumplimiento en " +
			"curso no admite planificacion")
	}
	// EL DESCARGO, que es lo que separa un dato de una acusacion.
	if !strings.Contains(out, "Esto NO dice que se haya incumplido") {
		t.Errorf(`la seccion no lleva su descargo.

  plazum NO sabe si la obligacion se cumplio: sabe que en las respuestas del
  operador no consta. Imprimir "vencido" sin esa frase convierte una ausencia de
  dato en una acusacion, y la primera reunion en la que eso pase se lleva por
  delante la confianza en la pantalla entera.
%s`, out)
	}
}

// TRES MARCOS EN UNA SESION, y los tres piden lo mismo con otras palabras.
//
// Es el caso que justifica la composicion entre marcos, y el corpus lo tiene
// entero desde el 29-08-2026:
//
//	RGPD art. 32.1.d   verificar la eficacia de las medidas tecnicas y organizativas
//	AI Act art. 9.2    revisar el sistema de gestion de riesgos del sistema de alto riesgo
//	DORA art. 6.5      revisar el marco de gestion del riesgo relacionado con las TIC
//
// Tres reglamentos, tres autoridades, y una sola tarde de trabajo si las fechas
// caen juntas. Ninguna herramienta que trate cada marco por separado puede
// decir esa frase, porque para decirla hay que tener los tres relojes a la vez.
func TestUnaSentadaPuedeCubrirTresReglamentos(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "alcance.json")
	if err := os.WriteFile(ruta, []byte(`{
  "organizacion": "Proveedor de IA de alto riesgo, S.L.",
  "sujeto": "acme",
  "hechos": [
    {"pred": "papel_ia", "args": ["acme", "proveedor"]},
    {"pred": "riesgo_ia", "args": ["acme", "alto_anexo_iii"]},
    {"pred": "trata_datos_personales", "args": ["acme"]},
    {"pred": "designado", "args": ["acme", "entidad_financiera"]}
  ],
  "fechas": {
    "ultima_verificacion_de_la_eficacia_de_las_medidas": "2027-03-10",
    "ultima_revision_del_sistema_de_gestion_de_riesgos": "2027-03-12",
    "ultima_revision_del_marco_de_riesgo_tic": "2027-03-20"
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	// EL INSTANTE DE ESTE CASO ES PROPIO Y POSTERIOR AL DE LOS DEMAS, y el
	// motivo es una correccion de corpus, no una comodidad del test: el
	// Reglamento (UE) 2026/1744 movio el capitulo III del AI Act al 02-12-2027,
	// asi que el art. 9.2 NO obliga a nadie en 2026 y la sentada de los tres
	// reglamentos no puede existir antes de esa fecha. Bajar la exigencia a dos
	// marcos habria sido la salida barata: la afirmacion que vende este producto
	// es que cruza marcos, y sigue siendo cierta, solo que un ano mas tarde.
	out, _, codigo := correrCalendario(t, "--ahora=2027-12-15T09:00:00Z",
		"--corpus=../../paquetes", "--alcance="+ruta, "--sentadas")
	if codigo != 0 {
		t.Fatalf("codigo %d\n%s", codigo, out)
	}
	if !strings.Contains(out, "3 fechas de 3 marcos") {
		t.Errorf(`ninguna sentada cubre los tres reglamentos.

  Las tres obligaciones (RGPD 32.1.d, AI Act 9.2 y DORA 6.5) piden verificar la
  eficacia de lo implantado, vencen el mismo mes y son una sola sesion. Si esta
  linea desaparece, o el corpus ha perdido una de las tres o la agrupacion ha
  dejado de cruzar marcos.
%s`, out)
	}
	// Y los tres marcos, nombrados: sin esto la linea de arriba podria salir de
	// tres obligaciones del mismo reglamento contadas mal.
	for _, urn := range []string{"urn:eu:reg:2016:679", "urn:eu:reg:2024:1689", "urn:eu:reg:2022:2554"} {
		if !strings.Contains(out, urn) {
			t.Errorf("la sentada no nombra %s", urn)
		}
	}
}
