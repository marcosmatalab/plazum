package ventana

import (
	"testing"
	"time"
)

// LO QUE LA FAMILIA A NECESITA DEL MOTOR, medido antes de escribir corpus.
//
// La familia A del censo son once fuentes y treinta y tres relojes de
// notificacion escalonada de incidente. Antes de escribir ni uno, aqui se mide
// si el motor hace las dos cosas que esa familia necesita y que el censo nombra
// de pasada:
//
//	(a) HITOS ENCADENADOS: la notificacion intermedia cuenta desde la REMISION
//	    DE LA INICIAL, no desde el incidente. Si se contara desde el incidente,
//	    un obligado que remite la inicial tarde tendria menos tiempo para la
//	    intermedia del que la norma le da, y el motor le estaria dando una fecha
//	    peor que la real.
//
//	(b) EL LIMITE DECIDIDO POR UNA CATEGORIA QUE ASIGNA EL PROPIO OBLIGADO: el
//	    mismo articulo da unos plazos si el incidente es de un nivel y otros si
//	    es de otro, y quien pone el nivel es quien sufre el incidente.
//
// Escribir 33 relojes contra una primitiva que no hace lo que hace falta es
// escribirlos dos veces, asi que esto va antes.

// Instantes y nombres neutros: aqui no se cablea ninguna norma, se mide el
// motor. Quien nombre articulos sera un paquete de corpus.
const (
	conocimiento = "incidente.conocimiento"
	nivelAlto    = "incidente.nivel.alto"
	nivelBajo    = "incidente.nivel.bajo"
)

func regNat(t *testing.T) Regimen {
	return Regimen{Comp: Naturales, Cal: calES(t), Cierre: CierreExacto}
}

// ---------------------------------------------------------------------------
// (a) Hitos encadenados
// ---------------------------------------------------------------------------

// EL ARRANQUE DE LA FAMILIA A ENTRA COMO HECHO. Es la propiedad que decide la
// primitiva, y la medicion original (26-08-2026) la establecio comparando
// `Plazo` contra `Secuencia`: aquella cableaba el arranque en la estructura y
// esta lo toma de `Hechos[Disparador]`, que es lo unico que sirve cuando el
// arranque es un incidente del cliente y el paquete se escribe meses antes.
//
// `Secuencia` se borro el 02-09-2026 (ver el hueco 6 de primitivas.go), asi que
// el contraejemplo ya no se puede ejecutar. NO SE ECHA DE MENOS: lo que habia
// que fijar era la propiedad, y la propiedad se afirma sola. Un contraejemplo
// que obliga a conservar el codigo que refuta es un test que se cobra su propia
// conclusion.
func TestElArranqueDeLaFamiliaAEntraComoHecho(t *testing.T) {
	reg := regNat(t)
	d24, err := ParseDuracion("PT24H")
	if err != nil {
		t.Fatal(err)
	}
	p := Plazo{Disparador: conocimiento, Hitos: []Hito{{ID: "inicial", Limite: d24, Reg: reg}}}

	// SIN el hecho el reloj no ha arrancado, Y SE DICE: callarlo lo leeria
	// cualquiera como "nada que hacer".
	sinHecho := p.Vencimientos(Hechos{}, time.Time{})
	if len(sinHecho) != 1 || sinHecho[0].Estado != PendienteDeHecho {
		t.Fatalf("sin el hecho de arranque, el reloj no ha arrancado y hay que decirlo: %+v",
			sinHecho)
	}

	// CON el hecho, fecha. Y la fecha SE MUEVE con el hecho, que es lo que
	// distingue tomar el arranque del dato de tenerlo cableado: si no se
	// moviera, esta primitiva seria la que la medicion descarto.
	uno := p.Vencimientos(Hechos{conocimiento: ts(t, "2026-03-02T09:00:00+01:00")}, time.Time{})
	dos := p.Vencimientos(Hechos{conocimiento: ts(t, "2026-03-09T09:00:00+01:00")}, time.Time{})
	if uno[0].Estado != Determinado || dos[0].Estado != Determinado {
		t.Fatalf("con el hecho puesto tiene que dar fecha: %+v / %+v", uno, dos)
	}
	if !uno[0].Vence.Before(dos[0].Vence) {
		t.Fatalf("mover el hecho de arranque una semana no ha movido el vencimiento (%s vs %s): "+
			"entonces el arranque no viene del hecho y esta primitiva no sirve para la familia A",
			uno[0].Vence.Format(time.RFC3339), dos[0].Vence.Format(time.RFC3339))
	}
}

// (a) propiamente dicha: la intermedia cuenta desde la REMISION de la inicial.
func TestLaIntermediaCuentaDesdeLaRemisionDeLaInicialYNoDesdeElIncidente(t *testing.T) {
	reg := regNat(t)
	d24, err := ParseDuracion("PT24H")
	if err != nil {
		t.Fatal(err)
	}
	d72, err := ParseDuracion("PT72H")
	if err != nil {
		t.Fatal(err)
	}

	p := Plazo{Disparador: conocimiento, Hitos: []Hito{
		{ID: "inicial", Limite: d24, Reg: reg},
		{ID: "intermedia", Limite: d72, Reg: reg, DesdeHito: "inicial"},
	}}

	incidente := ts(t, "2026-03-02T09:00:00+01:00")

	// 1. Sin la inicial remitida, la intermedia NO tiene fecha. Es correcto y es
	//    lo que hay que ensenar: su reloj no ha arrancado.
	sinRemitir := p.Vencimientos(Hechos{conocimiento: incidente}, time.Time{})
	var intermedia *Vencimiento
	for i := range sinRemitir {
		if sinRemitir[i].Hito == "intermedia" {
			intermedia = &sinRemitir[i]
		}
	}
	if intermedia == nil {
		t.Fatal("la intermedia no sale en la lista")
	}
	if intermedia.Estado != PendienteDeHecho {
		t.Fatalf("HALLAZGO: la intermedia tiene fecha sin que se haya remitido la inicial. "+
			"Entonces se esta contando desde el incidente, y el obligado que remite tarde "+
			"vera un plazo mas corto del que la norma le da: %+v", intermedia)
	}

	// 2. Con la inicial remitida TARDE (a las 20 horas, dentro de plazo), la
	//    intermedia cuenta desde ESA remision.
	remision := ts(t, "2026-03-03T05:00:00+01:00") // 20 h despues
	conRemision := p.Vencimientos(Hechos{
		conocimiento:       incidente,
		"inicial.cumplido": remision,
	}, time.Time{})
	for i := range conRemision {
		if conRemision[i].Hito == "intermedia" {
			intermedia = &conRemision[i]
		}
	}
	if intermedia.Estado != Determinado {
		t.Fatalf("con la inicial remitida, la intermedia tiene que tener fecha: %+v", intermedia)
	}
	// 72 h desde la remision, calculado aparte: 2026-03-06T05:00:00+01:00.
	quiero := ts(t, "2026-03-06T05:00:00+01:00")
	if !intermedia.Vence.Equal(quiero) {
		t.Fatalf("la intermedia no cuenta desde la remision.\n  esperado %s (remision + 72 h)\n"+
			"  obtenido %s\n  Si sale 2026-03-05T09:00, esta contando desde el incidente, que "+
			"le quita al obligado las 20 horas que tardo en remitir la inicial\n  derivacion: %s",
			quiero.Format(time.RFC3339), intermedia.Vence.Format(time.RFC3339), intermedia.Regla)
	}
	// Y la derivacion tiene que DECIRLO, o un auditor no puede saber de donde
	// sale la fecha.
	if !contiene(intermedia.Regla, "cumplimiento efectivo de inicial") {
		t.Errorf("la derivacion no dice que cuenta desde la remision de la inicial: %s",
			intermedia.Regla)
	}
}

// ---------------------------------------------------------------------------
// (b) El limite decidido por una categoria que asigna el obligado
// ---------------------------------------------------------------------------

// LA MEDICION QUE DECIDE SI HAY QUE CONSTRUIR. Un mismo articulo da unos plazos
// para un nivel de peligrosidad y otros para otro, y el nivel lo pone quien
// sufre el incidente.
//
// La forma en que la categoria entra es como HECHO, y eso no es un apano: la
// clasificacion de un incidente OCURRE en un instante, igual que el incidente.
// Un hecho `incidente.nivel.alto` con su fecha dice a la vez que se clasifico
// asi y cuando, que es lo que hace falta para que una reclasificacion posterior
// sea otro hecho y no una edicion del anterior.
func TestElLimiteLoDecideLaCategoriaQueDeclaraElObligado(t *testing.T) {
	reg := regNat(t)
	d24, _ := ParseDuracion("PT24H")
	d72, _ := ParseDuracion("PT72H")
	incidente := ts(t, "2026-03-02T09:00:00+01:00")

	p := Plazo{Disparador: conocimiento, Hitos: []Hito{
		{ID: "inicial.alto", Limite: d24, Reg: reg, Clase: nivelAlto},
		{ID: "inicial.bajo", Limite: d72, Reg: reg, Clase: nivelBajo},
	}}

	t.Run("clasificado alto: rige el plazo del alto y el otro no sale", func(t *testing.T) {
		v := p.Vencimientos(Hechos{
			conocimiento: incidente,
			nivelAlto:    incidente,
		}, time.Time{})
		if len(v) != 1 {
			t.Fatalf("con una sola clase declarada tiene que salir un solo hito y salen %d: %+v",
				len(v), v)
		}
		if v[0].Hito != "inicial.alto" {
			t.Fatalf("ha regido el hito de la clase equivocada: %+v", v[0])
		}
		quiero := ts(t, "2026-03-03T09:00:00+01:00")
		if !v[0].Vence.Equal(quiero) {
			t.Fatalf("esperado %s y salio %s", quiero.Format(time.RFC3339), v[0].Vence.Format(time.RFC3339))
		}
	})

	t.Run("clasificado bajo: el mismo articulo da otro plazo", func(t *testing.T) {
		v := p.Vencimientos(Hechos{conocimiento: incidente, nivelBajo: incidente}, time.Time{})
		if len(v) != 1 || v[0].Hito != "inicial.bajo" {
			t.Fatalf("tenia que regir el hito de la clase baja: %+v", v)
		}
	})

	// EL CASO QUE NO SE PUEDE CALLAR: sin clasificar, no hay plazo que dar. Y
	// eso NO es "no hay obligacion": es que falta un dato que el obligado tiene
	// que poner. Devolver una lista vacia lo leeria como "nada que hacer".
	t.Run("sin clasificar: se dice, no se calla", func(t *testing.T) {
		v := p.Vencimientos(Hechos{conocimiento: incidente}, time.Time{})
		if len(v) == 0 {
			t.Fatal("HALLAZGO: sin clasificar el incidente no sale NADA. Una lista vacia se " +
				"lee como 'nada que hacer', y lo que pasa es que falta la clasificacion, que " +
				"la tiene que poner el obligado")
		}
		for _, x := range v {
			if x.Estado != PendienteDeHecho {
				t.Fatalf("sin clasificar no puede haber fecha cerrada: %+v", x)
			}
		}
	})

	// LA RECLASIFICACION, que es lo que pasa de verdad: un incidente que
	// empieza siendo de un nivel y se escala. Manda la clasificacion MAS
	// RECIENTE, y la derivacion tiene que decir cual se uso.
	t.Run("reclasificado: manda la mas reciente y se dice cual", func(t *testing.T) {
		v := p.Vencimientos(Hechos{
			conocimiento: incidente,
			nivelBajo:    incidente,
			nivelAlto:    incidente.Add(3 * time.Hour), // se escala tres horas despues
		}, time.Time{})
		if len(v) != 1 {
			t.Fatalf("tras reclasificar tiene que quedar un solo hito y quedan %d: %+v", len(v), v)
		}
		if v[0].Hito != "inicial.alto" {
			t.Fatalf("HALLAZGO: tras escalar a nivel alto sigue rigiendo el plazo del bajo. "+
				"El obligado veria un plazo mas largo del que le corresponde: %+v", v[0])
		}
		if !contiene(v[0].Regla, nivelAlto) {
			t.Errorf("la derivacion no dice que clasificacion se uso, asi que un auditor no "+
				"puede saber por que esa fecha y no la otra: %s", v[0].Regla)
		}
	})
}

// Y la otra mitad, que es lo que hace que esto se pueda mezclar: un hito SIN
// clase aplica siempre, conviva con los que si la tienen. Es la forma de un
// articulo que da una notificacion final a todos los niveles y una intermedia
// solo a los graves.
func TestUnHitoSinClaseAplicaSiempre(t *testing.T) {
	reg := regNat(t)
	d24, _ := ParseDuracion("PT24H")
	d20d, _ := ParseDuracion("P20D")
	incidente := ts(t, "2026-03-02T09:00:00+01:00")

	p := Plazo{Disparador: conocimiento, Hitos: []Hito{
		{ID: "inicial.alto", Limite: d24, Reg: reg, Clase: nivelAlto},
		{ID: "final", Limite: d20d, Reg: reg}, // sin clase: siempre
	}}
	v := p.Vencimientos(Hechos{conocimiento: incidente, nivelAlto: incidente}, time.Time{})
	if len(v) != 2 {
		t.Fatalf("el hito sin clase tiene que salir junto al de la clase declarada: %+v", v)
	}
	// Y sin clasificar, el sin clase SIGUE teniendo fecha: no depende de la
	// clasificacion.
	sinClasificar := p.Vencimientos(Hechos{conocimiento: incidente}, time.Time{})
	var final *Vencimiento
	for i := range sinClasificar {
		if sinClasificar[i].Hito == "final" {
			final = &sinClasificar[i]
		}
	}
	if final == nil || final.Estado != Determinado {
		t.Fatalf("HALLAZGO: un hito sin clase ha dejado de tener fecha porque el incidente no "+
			"esta clasificado. La notificacion final se debe a todos los niveles: %+v", sinClasificar)
	}
}

// EL EMPATE, que es lo que hace determinista a la seleccion de clase.
//
// Recorrer un mapa de Go da un orden distinto en cada ejecucion. Sin resolver
// el empate a proposito, dos clasificaciones con el mismo instante darian un
// ganador distinto cada vez, y el motor dejaria de ser determinista, que es la
// propiedad que sostiene el producto entero: dados los mismos hechos, dos
// maquinas cualesquiera dan la misma fecha.
//
// Se ejecuta muchas veces a proposito: un no-determinismo que solo se
// manifiesta a veces no se caza mirando una vez.
func TestDosClasificacionesEnElMismoInstanteNoSeResuelvenACaraOCruz(t *testing.T) {
	reg := regNat(t)
	d24, _ := ParseDuracion("PT24H")
	d72, _ := ParseDuracion("PT72H")
	incidente := ts(t, "2026-03-02T09:00:00+01:00")

	p := Plazo{Disparador: conocimiento, Hitos: []Hito{
		{ID: "inicial.alto", Limite: d24, Reg: reg, Clase: nivelAlto},
		{ID: "inicial.bajo", Limite: d72, Reg: reg, Clase: nivelBajo},
	}}
	hechos := Hechos{
		conocimiento: incidente,
		nivelAlto:    incidente, // mismo instante
		nivelBajo:    incidente, // mismo instante
	}

	for i := 0; i < 200; i++ {
		v := p.Vencimientos(hechos, time.Time{})
		for _, x := range v {
			if x.Estado != PendienteDeHecho {
				t.Fatalf("HALLAZGO: con dos clasificaciones en el MISMO instante se ha elegido "+
					"una (%s) y se ha dado una fecha. Recorrer un mapa de Go no tiene orden, "+
					"asi que esa eleccion es distinta en cada ejecucion y el motor deja de ser "+
					"determinista: %+v", x.Hito, x)
			}
			if !contiene(x.Regla, "mismo instante") {
				t.Fatalf("no se dice que el problema es el empate, asi que el operador no sabe "+
					"que corregir: %s", x.Regla)
			}
		}
	}

	// CONTROL NEGATIVO: separando las dos por un segundo, se resuelve y manda la
	// posterior. Sin esto, lo de arriba pasaria igual si la seleccion devolviera
	// siempre empate.
	hechos[nivelAlto] = incidente.Add(time.Second)
	v := p.Vencimientos(hechos, time.Time{})
	if len(v) != 1 || v[0].Hito != "inicial.alto" || v[0].Estado != Determinado {
		t.Fatalf("un segundo de diferencia tiene que resolver el empate a favor de la mas "+
			"reciente: %+v", v)
	}
}

// Y el determinismo del conjunto, sin empates: los mismos hechos dan el mismo
// resultado, incluido el ORDEN de la lista.
func TestLaSeleccionPorClaseEsDeterminista(t *testing.T) {
	reg := regNat(t)
	d24, _ := ParseDuracion("PT24H")
	d72, _ := ParseDuracion("PT72H")
	d20d, _ := ParseDuracion("P20D")
	incidente := ts(t, "2026-03-02T09:00:00+01:00")

	p := Plazo{Disparador: conocimiento, Hitos: []Hito{
		{ID: "inicial.alto", Limite: d24, Reg: reg, Clase: nivelAlto},
		{ID: "inicial.bajo", Limite: d72, Reg: reg, Clase: nivelBajo},
		{ID: "final", Limite: d20d, Reg: reg},
	}}
	hechos := Hechos{
		conocimiento: incidente,
		nivelBajo:    incidente,
		nivelAlto:    incidente.Add(3 * time.Hour),
	}

	primera := p.Vencimientos(hechos, time.Time{})
	for i := 0; i < 200; i++ {
		otra := p.Vencimientos(hechos, time.Time{})
		if len(otra) != len(primera) {
			t.Fatalf("dos evaluaciones de los mismos hechos dan %d y %d vencimientos",
				len(primera), len(otra))
		}
		for j := range otra {
			if otra[j].Hito != primera[j].Hito || !otra[j].Vence.Equal(primera[j].Vence) {
				t.Fatalf("dos evaluaciones de los mismos hechos dan resultados distintos en la "+
					"posicion %d:\n  %+v\n  %+v", j, primera[j], otra[j])
			}
		}
	}
}
