package ventana

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// Escenario real: empresa espanola que es a la vez responsable de tratamiento y
// fabricante de un producto con elementos digitales. Conoce un incidente el
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

	cra := Plazo{
		Disparador: "incidente.ts_conocimiento",
		Hitos: []Hito{
			{ID: "cra.alerta_24h", Limite: Duracion{Horas: 24}, Reg: horasExactas,
				Nota: "CSIRT coordinador y ENISA, via plataforma unica SRP"},
			{ID: "cra.notificacion_72h", Limite: Duracion{Horas: 72}, Reg: horasExactas,
				Nota: "CSIRT coordinador y ENISA, via SRP"},
			{ID: "cra.informe_final", Limite: Duracion{Meses: 1}, Reg: mesesUE, DesdeHito: "cra.notificacion_72h",
				Nota: "incidente grave: 1 mes desde la notificacion de 72 h. La variante de 14 dias es para vulnerabilidad explotada activamente, no para esto"},
		},
	}

	rgpd := Plazo{
		Disparador: "incidente.ts_conocimiento",
		Hitos: []Hito{
			{ID: "rgpd.art33_72h", Limite: Duracion{Horas: 72}, Reg: horasExactas,
				Nota: "AEPD, sede electronica, formulario de brechas",
				Alternativas: []Lectura{
					{ID: "lectura_1182_71", Limite: Duracion{Horas: 72}, Reg: lectura118271,
						Cita: "Rgto. 1182/71 art. 3.4 y 3.5, doctrina alemana. El EDPB sostiene lo contrario: 72 horas exactas"},
				}},
			{ID: "rgpd.art34_afectados", Limite: Duracion{Indeterminado: true}, Reg: horasExactas,
				Nota: "comunicacion a los interesados sin dilacion indebida: la norma no fija plazo"},
		},
	}

	// Estado 1: solo se conoce el incidente. El informe final aun no tiene fecha.
	h := Hechos{"incidente.ts_conocimiento": conocimiento}
	var b strings.Builder
	fmt.Fprintf(&b, "\n=== Estado 1: incidente conocido el %s ===\n", conocimiento.Format(time.RFC3339))
	imprimir(&b, "CRA (Rgto. UE 2024/2847, art. 14, vigente desde 11-09-2026)", cra.Vencimientos(h, time.Time{}))
	imprimir(&b, "RGPD (art. 33 y 34)", rgpd.Vencimientos(h, time.Time{}))
	fmt.Fprintf(&b, "\nNIS2 art. 23: NO se genera reloj.\n"+
		"  motivo: la Directiva (UE) 2022/2555 no esta transpuesta en Espana a esta fecha\n"+
		"  (la Comision llevo a Espana ante el TJUE en julio de 2026) y una directiva no\n"+
		"  transpuesta no crea obligaciones directas para un sujeto privado. Rige el marco\n"+
		"  del RDL 12/2018 y el RD 43/2021, que es otro paquete con otros plazos.\n"+
		"  Esto lo decide la vigencia declarada en el paquete de corpus, no un if en el codigo.\n")

	// Estado 2: la notificacion de 72 h se presenta de verdad el 20 a las 11:15.
	h["cra.notificacion_72h.cumplido"] = ts(t, "2026-09-20T11:15:00+02:00")
	fmt.Fprintf(&b, "\n=== Estado 2: notificacion de 72 h presentada el %s ===\n", h["cra.notificacion_72h.cumplido"].Format(time.RFC3339))
	imprimir(&b, "CRA", cra.Vencimientos(h, time.Time{}))
	t.Log(b.String())

	// Comprobaciones duras del escenario.
	v1 := indexar(cra.Vencimientos(Hechos{"incidente.ts_conocimiento": conocimiento}, time.Time{}))
	if v1["cra.informe_final"].Estado != PendienteDeHecho {
		t.Fatal("el informe final no puede tener fecha antes de que se presente la notificacion de 72 h")
	}
	v2 := indexar(cra.Vencimientos(h, time.Time{}))
	if got := v2["cra.informe_final"].Vence.Format(time.RFC3339); got != "2026-10-20T23:59:59+02:00" {
		t.Fatalf("informe final: esperado 2026-10-20T23:59:59+02:00, obtenido %s", got)
	}
	vr := indexar(rgpd.Vencimientos(h, time.Time{}))
	if len(vr["rgpd.art33_72h"].Divergencias) != 1 {
		t.Fatal("la divergencia de lectura del plazo de 72 h tiene que aflorar, no elegirse en silencio")
	}
	d := vr["rgpd.art33_72h"].Divergencias[0]
	if d.Delta != 27*time.Hour+59*time.Minute+59*time.Second {
		t.Fatalf("delta entre lecturas: %v", d.Delta)
	}
	if vr["rgpd.art34_afectados"].Estado != SinPlazoLegal {
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

// Las seis primitivas, cada una con una obligacion real de una norma distinta.
func TestLasSeisPrimitivasCubrenLasSieteTuberias(t *testing.T) {
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
		{"D metodologia", "ENS art. 28, analisis de riesgos", Secuencia{Inicio: inicio, Reg: reg, Fases: []Fase{
			{ID: "inventario", Duracion: Duracion{Dias: 15}},
			{ID: "valoracion", Duracion: Duracion{Dias: 20}, DependeDe: "inventario"},
			{ID: "tratamiento", Duracion: Duracion{Dias: 30}, DependeDe: "valoracion"}}}},
		{"F atestacion", "SOC 2 tipo II", Observacion{Hito: "muestreo", Ventana: Intervalo{Desde: inicio, Hasta: inicio.AddDate(1, 0, 0)}, Muestreo: Duracion{Meses: 3}, Reg: reg}},
		{"G producto", "CRA, desde la puesta en mercado", Continua{Hito: "soporte", I: Intervalo{Desde: inicio, Hasta: inicio.AddDate(5, 0, 0)}}},
		{"puntual", "AI Act, fecha de aplicacion", Puntual{Hito: "aplicacion", En: ts(t, "2027-12-02T00:00:00+01:00")}},
	}

	h := Hechos{"x": inicio}
	usadas := map[string]bool{}
	for _, c := range casos {
		vs := c.p.Vencimientos(h, hasta)
		if len(vs) == 0 {
			t.Fatalf("%s (%s) con primitiva %s no produjo ningun vencimiento", c.tuberia, c.norma, c.p.Nombre())
		}
		usadas[c.p.Nombre()] = true
	}
	if len(usadas) != 6 {
		t.Fatalf("se esperaban las 6 primitivas ejercitadas, se usaron %d: %v", len(usadas), usadas)
	}
	// Lo que este test NO demuestra, dicho por escrito: que no exista una
	// septima primitiva. Solo demuestra que las siete tuberias se expresan con
	// estas seis y que las seis son alcanzables. La falsacion de verdad es
	// ingerir una norma nueva y contar cuantas obligaciones no reducen.
}
