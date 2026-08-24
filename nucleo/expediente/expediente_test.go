package expediente

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"dutiq/nucleo/aplicabilidad"
	"dutiq/nucleo/estado"
	"dutiq/nucleo/ledger"
)

func ts(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// Escenario: un ayuntamiento con sede electronica (ENS), que es responsable de
// tratamiento (RGPD) y que ademas distribuye una app propia (CRA). Tres normas,
// tres formas de obligacion distintas, un solo expediente.
func construirExpediente(t *testing.T) *Expediente {
	t.Helper()
	comoEstaba := ts(t, "2026-09-18T09:00:00+02:00")

	progENS := aplicabilidad.Programa{Paquete: "ens@2022.311", Reglas: []aplicabilidad.Regla{
		{ID: "nivel_max", Cita: "RD 311/2022 Anexo I",
			Cabeza:   aplicabilidad.A("nivel_max", aplicabilidad.V("S"), aplicabilidad.V("_AGG")),
			Cuerpo:   []aplicabilidad.Atomo{aplicabilidad.A("maneja", aplicabilidad.V("S"), aplicabilidad.V("I")), aplicabilidad.A("nivel_dimension", aplicabilidad.V("I"), aplicabilidad.V("D"), aplicabilidad.V("N"))},
			Agregado: aplicabilidad.Maximo, SobreVar: "N",
			Escala: aplicabilidad.Escala{Nombre: "ens.niveles", Orden: []string{"BAJO", "MEDIO", "ALTO"}}},
		{ID: "categoria_media", Cita: "RD 311/2022 Anexo I",
			Cabeza: aplicabilidad.A("categoria", aplicabilidad.V("S"), aplicabilidad.C("MEDIA")),
			Cuerpo: []aplicabilidad.Atomo{aplicabilidad.A("nivel_max", aplicabilidad.V("S"), aplicabilidad.C("MEDIO"))}},
		{ID: "auditoria", Cita: "RD 311/2022 art. 31",
			Cabeza: aplicabilidad.A("aplica", aplicabilidad.C("ens.art31.auditoria"), aplicabilidad.V("S")),
			Cuerpo: []aplicabilidad.Atomo{aplicabilidad.A("categoria", aplicabilidad.V("S"), aplicabilidad.C("MEDIA"))}},
		{ID: "mfa_media", Cita: "RD 311/2022 Anexo II op.acc.5",
			Cabeza: aplicabilidad.A("aplica", aplicabilidad.C("ens.op.acc.5.mfa"), aplicabilidad.V("S")),
			Cuerpo: []aplicabilidad.Atomo{aplicabilidad.A("categoria", aplicabilidad.V("S"), aplicabilidad.C("MEDIA"))}},
		{ID: "copias", Cita: "RD 311/2022 Anexo II mp.info.6",
			Cabeza: aplicabilidad.A("aplica", aplicabilidad.C("ens.mp.info.6"), aplicabilidad.V("S")),
			Cuerpo: []aplicabilidad.Atomo{aplicabilidad.A("categoria", aplicabilidad.V("S"), aplicabilidad.C("MEDIA"))}},
	}}
	progRGPD := aplicabilidad.Programa{Paquete: "rgpd@2016.679", Reglas: []aplicabilidad.Regla{
		{ID: "brechas", Cita: "RGPD art. 33",
			Cabeza: aplicabilidad.A("aplica", aplicabilidad.C("rgpd.art33.notificacion"), aplicabilidad.V("E")),
			Cuerpo: []aplicabilidad.Atomo{aplicabilidad.A("responsable", aplicabilidad.V("E"))}},
	}}
	progCRA := aplicabilidad.Programa{Paquete: "cra@2024.2847", Reglas: []aplicabilidad.Regla{
		{ID: "notificacion", Cita: "Rgto. 2024/2847 art. 14",
			Cabeza: aplicabilidad.A("aplica", aplicabilidad.C("cra.art14.alerta"), aplicabilidad.V("E")),
			Cuerpo: []aplicabilidad.Atomo{aplicabilidad.A("comercializa_producto_digital", aplicabilidad.V("E")), aplicabilidad.A("actividad_comercial", aplicabilidad.V("E"))}},
	}}

	hechos := []aplicabilidad.Hecho{
		aplicabilidad.H("maneja", "sede-electronica", "padron"),
		aplicabilidad.H("nivel_dimension", "padron", "confidencialidad", "MEDIO"),
		aplicabilidad.H("nivel_dimension", "padron", "integridad", "BAJO"),
		aplicabilidad.H("responsable", "ayto"),
		aplicabilidad.H("comercializa_producto_digital", "ayto"),
		aplicabilidad.H("actividad_comercial", "ayto"),
	}

	cal := CalendarioDeclarado{ID: "es-2026", Zona: "Europe/Madrid", Ambito: "nacional",
		Fuente: "BOE calendario laboral 2026",
		Festivos: []string{"2026-01-01", "2026-01-06", "2026-04-03", "2026-05-01",
			"2026-08-15", "2026-10-12", "2026-11-02", "2026-12-08", "2026-12-25"}}

	relojes := []RelojDeclarado{
		{Obligacion: "cra.art14.alerta", Disparador: "incidente.conocimiento",
			Hechos:     map[string]string{"incidente.conocimiento": "2026-09-17T20:00:00+02:00"},
			Calendario: cal,
			Hitos: []HitoDeclarado{
				{ID: "alerta_24h", Limite: "PT24H", Computo: "naturales", Cierre: "exacto",
					Traslado: "ninguno", Fuente: "Rgto. 1182/71 art. 3.1: los plazos en horas corren de hora a hora"},
				{ID: "notificacion_72h", Limite: "PT72H", Computo: "naturales", Cierre: "exacto",
					Traslado: "ninguno", Fuente: "Rgto. 1182/71 art. 3.1"},
				{ID: "informe_final", Limite: "P1M", DesdeHito: "notificacion_72h", Computo: "naturales",
					Cierre: "auto", Traslado: "siguiente_habil", Fuente: "Rgto. 1182/71 art. 3.2.c y 3.4"},
			}},
		{Obligacion: "rgpd.art33.notificacion", Disparador: "incidente.conocimiento",
			Hechos:     map[string]string{"incidente.conocimiento": "2026-09-17T20:00:00+02:00"},
			Calendario: cal,
			Hitos: []HitoDeclarado{
				{ID: "aepd_72h", Limite: "PT72H", Computo: "naturales", Cierre: "exacto",
					Traslado: "ninguno", Fuente: "RGPD art. 33.1, lectura EDPB: 72 horas exactas"},
			}},
	}

	pruebas := []estado.Prueba{
		{ID: "mfa.usuarios", Control: "ens.op.acc.5.mfa", TTL: 24 * time.Hour, SLA: 72 * time.Hour,
			Descripcion: "todos los usuarios con acceso administrativo tienen MFA"},
		{ID: "backup.restauracion", Control: "ens.mp.info.6", TTL: 90 * 24 * time.Hour, SLA: 30 * 24 * time.Hour,
			Descripcion: "prueba de restauracion realizada"},
	}
	observaciones := []estado.Observacion{
		{Prueba: "mfa.usuarios", Recurso: "u-alcalde", Satisfecho: true,
			Recolectada: comoEstaba.Add(-2 * time.Hour), Recolector: "entra-id", Version: "1.2.0", HashCarga: "sha256:aa"},
		{Prueba: "mfa.usuarios", Recurso: "u-interventor", Satisfecho: false,
			Recolectada: comoEstaba.Add(-2 * time.Hour), Recolector: "entra-id", Version: "1.2.0", HashCarga: "sha256:bb"},
		{Prueba: "backup.restauracion", Recurso: "sede-electronica", Satisfecho: true,
			Recolectada: comoEstaba.Add(-120 * 24 * time.Hour), Recolector: "manual", Version: "1.0.0", HashCarga: "sha256:cc"},
	}

	e := &Expediente{
		Version: Version, Emitido: comoEstaba, ComoEstaba: comoEstaba,
		Organizacion: "Ayuntamiento de ejemplo", Alcance: "sede electronica y app municipal",
		Paquetes: []Paquete{
			{URN: "ens@2022.311", Version: "2022.311", Digest: "sha256:1f3a", Clase: "normativo",
				Vigencia: Intervalo{Desde: ts(t, "2022-05-05T00:00:00+02:00")}},
			{URN: "rgpd@2016.679", Version: "2016.679", Digest: "sha256:2b7c", Clase: "normativo",
				Vigencia: Intervalo{Desde: ts(t, "2018-05-25T00:00:00+02:00")}},
			{URN: "cra@2024.2847", Version: "2024.2847", Digest: "sha256:9f3c", Clase: "normativo",
				Vigencia: Intervalo{Desde: ts(t, "2026-09-11T00:00:00+02:00")}},
		},
		Programas: []aplicabilidad.Programa{progENS, progRGPD, progCRA},
		Hechos:    hechos,
		Obligaciones: []Obligacion{
			{ID: "ens.art31.auditoria", Paquete: "ens@2022.311", Articulo: "31", Primitiva: "periodica",
				Afirmacion: "el sistema ha sido auditado en los ultimos 24 meses"},
			{ID: "ens.op.acc.5.mfa", Paquete: "ens@2022.311", Articulo: "Anexo II op.acc.5",
				Control: "ens.op.acc.5.mfa", Primitiva: "continua", Afirmacion: "la autenticacion usa mas de un factor"},
			{ID: "rgpd.art33.notificacion", Paquete: "rgpd@2016.679", Articulo: "33", Primitiva: "plazo",
				Afirmacion: "la brecha se notifico a la AEPD en 72 horas"},
			{ID: "cra.art14.alerta", Paquete: "cra@2024.2847", Articulo: "14", Primitiva: "plazo",
				Afirmacion: "se notifico al CSIRT coordinador y a ENISA"},
		},
		Pruebas: pruebas, Observaciones: observaciones,
		Relojes: relojes,
	}

	// El emisor declara lo que ha calculado. La verificacion lo recalcula.
	e.Aplicables = []string{"ens.art31.auditoria", "ens.op.acc.5.mfa", "ens.mp.info.6",
		"rgpd.art33.notificacion", "cra.art14.alerta"}
	e.Reclamaciones = []Reclamacion{
		{Obligacion: "cra.art14.alerta", Hito: "alerta_24h", Estado: "determinado", Vence: ts(t, "2026-09-18T20:00:00+02:00")},
		{Obligacion: "cra.art14.alerta", Hito: "notificacion_72h", Estado: "determinado", Vence: ts(t, "2026-09-20T20:00:00+02:00")},
		{Obligacion: "cra.art14.alerta", Hito: "informe_final", Estado: "pendiente de hecho"},
		{Obligacion: "rgpd.art33.notificacion", Hito: "aepd_72h", Estado: "determinado", Vence: ts(t, "2026-09-20T20:00:00+02:00")},
	}
	e.Estados = []EstadoControl{
		{Prueba: "mfa.usuarios", Estado: "fail_en_plazo"},
		{Prueba: "backup.restauracion", Estado: "obsoleto"},
	}
	e.Denominadores = estado.Denominadores{Maquina: 1, CaducadoOContradicho: 1}

	// Digests reales, calculados sobre el contenido, y anclas que en produccion
	// vendrian del registro OCI firmado y no del propio expediente.
	e.AnclasDeConfianza = map[string]string{}
	for i := range e.Paquetes {
		d := DigestPaquete(e.Paquetes[i].URN, e.Programas, e.Obligaciones)
		e.Paquetes[i].Digest = d
		e.AnclasDeConfianza[e.Paquetes[i].URN] = d
	}

	// Ledger con las observaciones y un checkpoint firmado y anclado.
	semilla := make([]byte, ed25519.SeedSize)
	copy(semilla, []byte("dutiq-demo-semilla-determinista"))
	k := ed25519.NewKeyFromSeed(semilla)
	e.Ledger.ClavesConfiables = []string{hex.EncodeToString(k.Public().(ed25519.PublicKey))}
	for _, o := range observaciones {
		carga, _ := json.Marshal(o)
		if _, err := e.Ledger.Anadir(ledger.Entrada{
			Instante: o.Recolectada, Tipo: "observacion", Sujeto: o.Recurso,
			Paquete: "ens@2022.311", PaqueteHash: e.Paquetes[0].Digest, Carga: carga,
			Actor: "conector:" + o.Recolector}); err != nil {
			t.Fatal(err)
		}
	}
	e.Ledger.Cerrar(k, comoEstaba, "tsa:rfc3161://tsa.example + testigo publico diario")
	return e
}

func TestExpedienteVerificaSinRed(t *testing.T) {
	e := construirExpediente(t)
	b, err := e.Guardar()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("expediente serializado: %d bytes", len(b))

	// Un tercero solo tiene el fichero.
	otro, err := Cargar(b)
	if err != nil {
		t.Fatal(err)
	}
	inf := Verificar(otro)
	for _, c := range inf.Comprobaciones {
		t.Log("  ok  " + c)
	}
	for _, d := range inf.Discrepancias {
		t.Errorf("  DISCREPANCIA %s: esperado %q, obtenido %q", d.Que, d.Esperado, d.Obtenido)
	}
	if !inf.Valido {
		t.Fatal("el expediente emitido tiene que verificar en otra maquina, sin red y sin confiar en el emisor")
	}
}

func TestDetectaVencimientoFalseado(t *testing.T) {
	e := construirExpediente(t)
	// El emisor se da tres dias mas en el reloj del CRA.
	for i := range e.Reclamaciones {
		if e.Reclamaciones[i].Hito == "notificacion_72h" {
			e.Reclamaciones[i].Vence = e.Reclamaciones[i].Vence.Add(72 * time.Hour)
		}
	}
	b, _ := e.Guardar()
	otro, _ := Cargar(b)
	inf := Verificar(otro)
	if inf.Valido {
		t.Fatal("un vencimiento falseado tiene que detectarse al recalcular")
	}
	t.Logf("detectado: %s -> esperado %s, recalculado %s",
		inf.Discrepancias[0].Que, inf.Discrepancias[0].Esperado, inf.Discrepancias[0].Obtenido)
}

func TestDetectaAplicabilidadInventada(t *testing.T) {
	e := construirExpediente(t)
	e.Aplicables = append(e.Aplicables, "iso27001.a.5.1")
	b, _ := e.Guardar()
	otro, _ := Cargar(b)
	inf := Verificar(otro)
	if inf.Valido {
		t.Fatal("declarar aplicable algo que las reglas no derivan tiene que detectarse")
	}
	t.Logf("detectado: %v", inf.Discrepancias[0])
}

func TestDetectaEvaluacionAntesDeLaVigencia(t *testing.T) {
	e := construirExpediente(t)
	// El CRA no existe antes del 11 de septiembre de 2026.
	e.ComoEstaba = ts(t, "2026-08-01T09:00:00+02:00")
	b, _ := e.Guardar()
	otro, _ := Cargar(b)
	inf := Verificar(otro)
	if inf.Valido {
		t.Fatal("evaluar una obligacion antes de la vigencia de su norma tiene que detectarse")
	}
	var visto bool
	for _, d := range inf.Discrepancias {
		if d.Que == "vigencia de cra.art14.alerta" {
			visto = true
			t.Logf("detectado: %v", d)
		}
	}
	if !visto {
		t.Fatalf("esperaba una discrepancia de vigencia, obtuve %v", inf.Discrepancias)
	}
}

func TestDetectaEstadoFalseado(t *testing.T) {
	e := construirExpediente(t)
	for i := range e.Estados {
		if e.Estados[i].Prueba == "backup.restauracion" {
			e.Estados[i].Estado = "pass" // "obsoleto" contado como conforme
		}
	}
	b, _ := e.Guardar()
	otro, _ := Cargar(b)
	inf := Verificar(otro)
	if inf.Valido {
		t.Fatal("una evidencia caducada presentada como conforme tiene que detectarse")
	}
	t.Logf("detectado: %v", inf.Discrepancias)
}

func TestDetectaLedgerManipulado(t *testing.T) {
	e := construirExpediente(t)
	var carga map[string]any
	_ = json.Unmarshal(e.Ledger.Entradas[1].Carga, &carga)
	carga["Satisfecho"] = true
	e.Ledger.Entradas[1].Carga, _ = json.Marshal(carga)
	b, _ := e.Guardar()
	otro, _ := Cargar(b)
	if inf := Verificar(otro); inf.Valido {
		t.Fatal("manipular el ledger tiene que detectarse")
	}
}

// --- Tests anadidos tras la revision adversarial ---
// Los cuatro ataques que ANTES pasaban la verificacion.

func TestDetectaObligacionInventadaConSuPropiaRegla(t *testing.T) {
	e := construirExpediente(t)
	// El emisor no solo declara la obligacion: tambien aporta la regla que la
	// deriva. Antes esto verificaba limpio, porque nadie contrastaba el corpus.
	e.Programas = append(e.Programas, aplicabilidad.Programa{Paquete: "iso27001@2022",
		Reglas: []aplicabilidad.Regla{{ID: "inventada", Cita: "A.5.1",
			Cabeza: aplicabilidad.A("aplica", aplicabilidad.C("iso27001.a.5.1"), aplicabilidad.V("E")),
			Cuerpo: []aplicabilidad.Atomo{aplicabilidad.A("responsable", aplicabilidad.V("E"))}}}})
	e.Aplicables = append(e.Aplicables, "iso27001.a.5.1")
	b, _ := e.Guardar()
	otro, _ := Cargar(b)
	inf := Verificar(otro)
	if inf.Valido {
		t.Fatal("inventar una obligacion junto con su regla tiene que detectarse contra el registro")
	}
	t.Logf("detectado: %v", inf.Discrepancias[0])
}

func TestDetectaObservacionCambiadaSinTocarElLedger(t *testing.T) {
	e := construirExpediente(t)
	for i := range e.Observaciones {
		if e.Observaciones[i].Recurso == "u-interventor" {
			e.Observaciones[i].Satisfecho = true // el fallo desaparece
		}
	}
	for i := range e.Estados {
		if e.Estados[i].Prueba == "mfa.usuarios" {
			e.Estados[i].Estado = "pass"
		}
	}
	b, _ := e.Guardar()
	otro, _ := Cargar(b)
	if inf := Verificar(otro); inf.Valido {
		t.Fatal("cambiar una observacion sin tocar el ledger tiene que detectarse por el anclaje")
	}
}

func TestDetectaReclamacionSinRelojQueLaProduzca(t *testing.T) {
	e := construirExpediente(t)
	e.Reclamaciones = append(e.Reclamaciones, Reclamacion{
		Obligacion: "ens.art31.auditoria", Hito: "inventado", Estado: "determinado",
		Vence: ts(t, "2099-01-01T00:00:00+01:00")})
	b, _ := e.Guardar()
	otro, _ := Cargar(b)
	if inf := Verificar(otro); inf.Valido {
		t.Fatal("una fecha declarada que ningun reloj calcula tiene que detectarse")
	}
}

func TestDetectaDenominadoresFalseados(t *testing.T) {
	e := construirExpediente(t)
	e.Denominadores.Humano = 47 // antes este campo no se comprobaba
	b, _ := e.Guardar()
	otro, _ := Cargar(b)
	if inf := Verificar(otro); inf.Valido {
		t.Fatal("los cinco denominadores tienen que compararse, no dos")
	}
}

func TestSinAnclasDeConfianzaNoVerifica(t *testing.T) {
	e := construirExpediente(t)
	e.AnclasDeConfianza = nil
	b, _ := e.Guardar()
	otro, _ := Cargar(b)
	inf := Verificar(otro)
	if inf.Valido {
		t.Fatal("sin anclas del receptor la verificacion es circular y no puede darse por buena")
	}
	t.Log("la verificacion se declara circular en vez de fingir que no lo es")
}
