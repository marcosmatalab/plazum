package main

import (
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
