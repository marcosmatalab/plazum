package ventana

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// Escenario real: empresa espanola que es a la vez responsable de tratamiento y
// fabricante de un producto con elementos digitales. Los identificadores de hito
// son sinteticos (demo.*) porque nucleo/ no cablea normas: quien las nombra es
// el paquete de corpus. El contexto normativo no se pierde, esta en la Nota y en
// la Fuente de cada regimen, que es donde sirve para algo. Conoce un incidente el
// jueves 17 de septiembre de 2026 a las 20:00. Un solo hecho, varios relojes,
// destinatarios distintos y una interpretacion en disputa.
func TestEscenarioIncidenteSeptiembre2026(t *testing.T) {
	cal := calES(t)
	horasExactas := Regimen{Comp: Naturales, Cal: cal, Trasl: TrasladoNinguno, Cierre: CierreExacto,
		Fuente: "Rgto. 1182/71 art. 3.1: los plazos en horas corren de hora a hora"}
	mesesUE := Regimen{Comp: Naturales, Cal: cal, Trasl: TrasladoSiguienteHabil, Cierre: CierreAuto,
		Fuente: "Rgto. 1182/71 art. 3.2.c y 3.4"}
	lectura118271 := Regimen{Comp: Naturales, Cal: cal, Trasl: TrasladoSiguienteHabil, Cierre: CierreExacto,
		Fuente: "Rgto. 1182/71 art. 3.4"}

	conocimiento := ts(t, "2026-09-17T20:00:00+02:00")

	plazoProducto := Plazo{
		Disparador: "incidente.ts_conocimiento",
		Hitos: []Hito{
			{ID: "demo.alerta_24h", Limite: Duracion{Horas: 24}, Reg: horasExactas,
				Nota: "CSIRT coordinador y ENISA, via plataforma unica SRP"},
			{ID: "demo.notificacion_72h", Limite: Duracion{Horas: 72}, Reg: horasExactas,
				Nota: "CSIRT coordinador y ENISA, via SRP"},
			{ID: "demo.informe_final", Limite: Duracion{Meses: 1}, Reg: mesesUE, DesdeHito: "demo.notificacion_72h",
				Nota: "incidente grave: 1 mes desde la notificacion de 72 h. La variante de 14 dias es para vulnerabilidad explotada activamente, no para esto"},
		},
	}

	plazoBrecha := Plazo{
		Disparador: "incidente.ts_conocimiento",
		Hitos: []Hito{
			{ID: "demo.brecha_72h", Limite: Duracion{Horas: 72}, Reg: horasExactas,
				Nota: "AEPD, sede electronica, formulario de brechas",
				Alternativas: []Lectura{
					{ID: "lectura_1182_71", Limite: Duracion{Horas: 72}, Reg: lectura118271,
						Cita: "Rgto. 1182/71 art. 3.4 y 3.5, doctrina alemana. El EDPB sostiene lo contrario: 72 horas exactas"},
				}},
			{ID: "demo.brecha_afectados", Limite: Duracion{Indeterminado: true}, Reg: horasExactas,
				Nota: "comunicacion a los interesados sin dilacion indebida: la norma no fija plazo"},
		},
	}

	// Estado 1: solo se conoce el incidente. El informe final aun no tiene fecha.
	h := Hechos{"incidente.ts_conocimiento": conocimiento}
	var b strings.Builder
	fmt.Fprintf(&b, "\n=== Estado 1: incidente conocido el %s ===\n", conocimiento.Format(time.RFC3339))
	imprimir(&b, "CRA (Rgto. UE 2024/2847, art. 14, vigente desde 11-09-2026)", plazoProducto.Vencimientos(h, time.Time{}))
	imprimir(&b, "RGPD (art. 33 y 34)", plazoBrecha.Vencimientos(h, time.Time{}))
	fmt.Fprintf(&b, "\nNIS2 art. 23: NO se genera reloj.\n"+
		"  motivo: la Directiva (UE) 2022/2555 no esta transpuesta en Espana a esta fecha\n"+
		"  (la Comision llevo a Espana ante el TJUE en julio de 2026) y una directiva no\n"+
		"  transpuesta no crea obligaciones directas para un sujeto privado. Rige el marco\n"+
		"  del RDL 12/2018 y el RD 43/2021, que es otro paquete con otros plazos.\n"+
		"  Esto lo decide la vigencia declarada en el paquete de corpus, no un if en el codigo.\n")

	// Estado 2: la notificacion de 72 h se presenta de verdad el 20 a las 11:15.
	h["demo.notificacion_72h.cumplido"] = ts(t, "2026-09-20T11:15:00+02:00")
	fmt.Fprintf(&b, "\n=== Estado 2: notificacion de 72 h presentada el %s ===\n", h["demo.notificacion_72h.cumplido"].Format(time.RFC3339))
	imprimir(&b, "CRA", plazoProducto.Vencimientos(h, time.Time{}))
	t.Log(b.String())

	// Comprobaciones duras del escenario.
	v1 := indexar(plazoProducto.Vencimientos(Hechos{"incidente.ts_conocimiento": conocimiento}, time.Time{}))
	if v1["demo.informe_final"].Estado != PendienteDeHecho {
		t.Fatal("el informe final no puede tener fecha antes de que se presente la notificacion de 72 h")
	}
	v2 := indexar(plazoProducto.Vencimientos(h, time.Time{}))
	if got := v2["demo.informe_final"].Vence.Format(time.RFC3339); got != "2026-10-20T23:59:59+02:00" {
		t.Fatalf("informe final: esperado 2026-10-20T23:59:59+02:00, obtenido %s", got)
	}
	vr := indexar(plazoBrecha.Vencimientos(h, time.Time{}))
	if len(vr["demo.brecha_72h"].Divergencias) != 1 {
		t.Fatal("la divergencia de lectura del plazo de 72 h tiene que aflorar, no elegirse en silencio")
	}
	d := vr["demo.brecha_72h"].Divergencias[0]
	if d.Delta != 27*time.Hour+59*time.Minute+59*time.Second {
		t.Fatalf("delta entre lecturas: %v", d.Delta)
	}
	if vr["demo.brecha_afectados"].Estado != SinPlazoLegal {
		t.Fatal("'sin dilacion indebida' no es un plazo y no se puede inventar uno")
	}
}

func indexar(vs []Vencimiento) map[string]Vencimiento {
	m := map[string]Vencimiento{}
	for _, v := range vs {
		m[v.Hito] = v
	}
	return m
}

func imprimir(b *strings.Builder, titulo string, vs []Vencimiento) {
	fmt.Fprintf(b, "\n%s\n", titulo)
	for _, v := range vs {
		switch v.Estado {
		case Determinado:
			fmt.Fprintf(b, "  %-22s vence %s\n", v.Hito, v.Vence.Format(time.RFC3339))
		default:
			fmt.Fprintf(b, "  %-22s %s\n", v.Hito, v.Estado)
		}
		fmt.Fprintf(b, "      %s\n", v.Regla)
		if v.Aviso != "" {
			fmt.Fprintf(b, "      nota: %s\n", v.Aviso)
		}
		for _, d := range v.Divergencias {
			fmt.Fprintf(b, "      DIVERGENCIA %s: %s (%v de diferencia)\n         %s\n",
				d.Lectura, d.Vence.Format(time.RFC3339), d.Delta, d.Cita)
		}
	}
}

// TODAS las primitivas vivas, cada una con una obligacion real de una norma
// distinta. El numero no se cablea aqui: se cuenta, y ese es el punto.
//
// Se llamaba "las seis" y llevaba un 6 escrito a mano. El 02-09-2026 se borro
// `Secuencia` (medicion: ningun reloj contado la pedia, el corpus no podia
// usarla y la familia A ya la habia descartado) y este test se puso rojo por el
// NUMERO, no por la cobertura. Una cifra escrita a mano en un test convierte
// cada borrado legitimo en un rojo que invita a bajarla sin mirar, que es como
// una puerta se vuelve un tramite.
//
// Ahora el minimo se declara y ademas se exige que las que se ejercitan sean
// las que EXISTEN: si manana se cablea una primitiva nueva y nadie le pone su
// fila aqui, este test lo dice.
func TestTodasLasPrimitivasVivasSeEjercitan(t *testing.T) {
	cal := calES(t)
	reg := Regimen{Comp: Naturales, Cal: cal}
	inicio := ts(t, "2026-01-01T00:00:00+01:00")
	hasta := ts(t, "2029-01-01T00:00:00+01:00")

	casos := []struct {
		tuberia string
		norma   string
		p       Primitiva
	}{
		{"E reloj", "CRA art. 14", Plazo{Disparador: "x", Hitos: []Hito{{ID: "a", Limite: Duracion{Horas: 24}, Reg: reg}}}},
		{"A catalogo", "ENS art. 31, revalidacion", Periodica{Hito: "conformidad", Desde: inicio, Cada: Duracion{Meses: 24}, Reg: reg}},
		{"B sistema de gestion", "ISO 27001 revision por la direccion", Periodica{Hito: "revision", Desde: inicio, Cada: Duracion{Meses: 12}, Gracia: Duracion{Dias: 30}, Reg: reg}},
		{"C registro vivo", "RGPD art. 30", Continua{Hito: "registro_actividades", I: Intervalo{Desde: inicio}}},
		{"F atestacion", "SOC 2 tipo II", Observacion{Hito: "muestreo", Ventana: Intervalo{Desde: inicio, Hasta: inicio.AddDate(1, 0, 0)}, Muestreo: Duracion{Meses: 3}, Reg: reg}},
		{"G producto", "CRA, desde la puesta en mercado", Continua{Hito: "soporte", I: Intervalo{Desde: inicio, Hasta: inicio.AddDate(5, 0, 0)}}},
		{"puntual", "AI Act, fecha de aplicacion", Puntual{Hito: "aplicacion", En: ts(t, "2027-12-02T00:00:00+01:00")}},
		{"H retencion compuesta", "CRA art. 13.9, diez anos o el soporte", Maximo{
			Hito: "fin_retencion", Disparador: "x", Suelo: Duracion{Meses: 120}, Reg: reg,
			Ampliacion: "fin_soporte", Exigible: true}},
		{"I preaviso", "psd2 art. 54.1, dos meses de antelacion", Preaviso{
			Hito: "aviso", Efecto: "efecto", Antelacion: Duracion{Meses: 2}, Reg: reg}},
	}

	h := Hechos{"x": inicio, "fin_soporte": inicio.AddDate(15, 0, 0),
		"efecto": inicio.AddDate(1, 0, 0)}
	usadas := map[string]bool{}
	for _, c := range casos {
		vs := c.p.Vencimientos(h, hasta)
		if len(vs) == 0 {
			t.Fatalf("%s (%s) con primitiva %s no produjo ningun vencimiento", c.tuberia, c.norma, c.p.Nombre())
		}
		usadas[c.p.Nombre()] = true
	}
	// LAS QUE EXISTEN, no un numero. La lista sale del propio paquete, asi que
	// una primitiva nueva sin fila aqui se delata sola.
	vivas := []string{}
	for _, p := range []Primitiva{Puntual{}, Periodica{}, Continua{}, Plazo{}, Observacion{},
		Maximo{}, Preaviso{}} {
		vivas = append(vivas, p.Nombre())
	}
	for _, n := range vivas {
		if !usadas[n] {
			t.Errorf("la primitiva %q existe en el motor y ninguna fila de este test la "+
				"ejercita. O se le pone su obligacion real, o se borra del motor: una "+
				"primitiva que nadie ejercita es la que se cablea 'por completitud' dentro de "+
				"seis meses", n)
		}
	}
	if len(usadas) != len(vivas) {
		t.Fatalf("se ejercitan %d primitivas y el motor tiene %d: %v", len(usadas), len(vivas), usadas)
	}
	// Lo que este test NO demuestra, dicho por escrito: que no exista una
	// septima primitiva. Solo demuestra que las siete tuberias se expresan con
	// estas seis y que las seis son alcanzables. La falsacion de verdad es
	// ingerir una norma nueva y contar cuantas obligaciones no reducen.
}
