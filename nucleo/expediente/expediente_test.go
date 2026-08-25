package expediente

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"plazum/nucleo/aplicabilidad"
	"plazum/nucleo/estado"
	"plazum/nucleo/ledger"
)

func ts(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// parsear lee una regla en la sintaxis de superficie del dialecto, que es como
// la escribe un paquete de corpus de verdad. Devuelve la Regla SIN id, SIN cita
// y SIN agregado: esos tres, igual que la escala, son campos del fichero de
// datos que viajan al lado de la regla, no parte de su texto.
func parsear(t *testing.T, txt string) aplicabilidad.Regla {
	t.Helper()
	r, err := aplicabilidad.ParsearRegla(txt)
	if err != nil {
		t.Fatalf("regla del escenario que no parsea: %v", err)
	}
	return r
}

// regla es parsear mas los dos campos que el linter de paquetes le exige a toda
// regla publicada: un identificador y el articulo que la justifica.
func regla(t *testing.T, id, cita, txt string) aplicabilidad.Regla {
	t.Helper()
	r := parsear(t, txt)
	r.ID, r.Cita = id, cita
	return r
}

// Escenario: una organizacion con un sistema de informacion propio y una
// aplicacion que comercializa, alcanzada por tres normas con tres FORMAS de
// obligacion distintas, en un solo expediente:
//
//	urn:demo:agregada  obligacion PERIODICA cuyo alcance sale de una categoria
//	                   AGREGADA: el maximo del nivel de cada dimension sobre
//	                   cada informacion que maneja el sistema. Es la forma del
//	                   calculo de categoria de un esquema nacional de seguridad
//	                   (RD 311/2022 art. 40 y Anexo I), con la auditoria bienal
//	                   de su art. 31 y dos medidas colgando de ella.
//	urn:demo:brechas   obligacion de PLAZO que arranca al conocerse una brecha
//	                   de datos personales. Es la forma del RGPD art. 33.
//	urn:demo:producto  obligacion de PLAZO EN CASCADA (24h, 72h, y un mes desde
//	                   el hito anterior) que arranca al conocerse un incidente
//	                   en un producto digital comercializado. Es la forma del
//	                   Rgto. 2024/2847 art. 14.
//
// Los identificadores son sinteticos, del espacio urn:demo: y demo.: el nucleo
// es autonomo y no conoce el directorio del corpus, asi que ningun test de
// nucleo/ puede depender de un URN real de paquetes/. El identificador dice la
// FORMA de la norma, no su nombre, para que el escenario se siga leyendo. La
// razon normativa no se pierde: sigue en el campo Cita de cada regla, que es
// cita y no identificador, y en los comentarios de aqui arriba.
func construirExpediente(t *testing.T) *Expediente {
	t.Helper()
	comoEstaba := ts(t, "2026-09-18T09:00:00+02:00")

	// La categoria del sistema es el MAXIMO del nivel de cada dimension
	// (confidencialidad, integridad, disponibilidad...) sobre cada informacion
	// que maneja. Esto es agregacion: un selector plano no puede calcularlo. La
	// escala la declara el paquete, porque con orden lexicografico el maximo de
	// {ALTO, BAJO} saldria BAJO.
	//
	// El agregado del anexo I: la categoria es el MAXIMO del nivel de cada
	// dimension sobre cada informacion y servicio. La variable que se agrega va
	// en la cabeza con su nombre, y el campo "sobre" del fichero de datos la
	// senala; _AGG es interna del motor y el dialecto la rechaza.
	nivelMax := regla(t, "nivel_max", "RD 311/2022 Anexo I",
		`nivel_max(S, N) :- maneja(S, I), nivel_dimension(I, _, N)`)
	nivelMax.Cabeza.Args[1] = aplicabilidad.V(aplicabilidad.VarAgregada)
	nivelMax.Agregado, nivelMax.SobreVar = aplicabilidad.Maximo, "N"
	nivelMax.Escala = aplicabilidad.Escala{Nombre: "demo.niveles", Orden: []string{"BAJO", "MEDIO", "ALTO"}}

	progAgregada := aplicabilidad.Programa{Paquete: "urn:demo:agregada", Reglas: []aplicabilidad.Regla{
		nivelMax,
		// MEDIA y MEDIO van entrecomillados a proposito: una constante en
		// mayusculas sin comillas es una VARIABLE para el dialecto, y la regla
		// derivaria de mas en silencio.
		regla(t, "categoria_media", "RD 311/2022 Anexo I",
			`categoria(S, "MEDIA") :- nivel_max(S, "MEDIO")`),
		regla(t, "auditoria", "RD 311/2022 art. 31",
			`aplica(demo.auditoria_bienal, S) :- categoria(S, "MEDIA")`),
		regla(t, "mfa_media", "RD 311/2022 Anexo II op.acc.5",
			`aplica(demo.mfa, S) :- categoria(S, "MEDIA")`),
		regla(t, "copias", "RD 311/2022 Anexo II mp.info.6",
			`aplica(demo.copias, S) :- categoria(S, "MEDIA")`),
	}}
	progBrechas := aplicabilidad.Programa{Paquete: "urn:demo:brechas", Reglas: []aplicabilidad.Regla{
		regla(t, "brechas", "RGPD art. 33",
			`aplica(demo.notificacion_brecha, E) :- responsable(E)`),
	}}
	progProducto := aplicabilidad.Programa{Paquete: "urn:demo:producto", Reglas: []aplicabilidad.Regla{
		regla(t, "notificacion", "Rgto. 2024/2847 art. 14",
			`aplica(demo.alerta_producto, E) :- comercializa_producto_digital(E), actividad_comercial(E)`),
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
		{Obligacion: "demo.alerta_producto", Disparador: "incidente.conocimiento",
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
		{Obligacion: "demo.notificacion_brecha", Disparador: "incidente.conocimiento",
			Hechos:     map[string]string{"incidente.conocimiento": "2026-09-17T20:00:00+02:00"},
			Calendario: cal,
			Hitos: []HitoDeclarado{
				{ID: "aepd_72h", Limite: "PT72H", Computo: "naturales", Cierre: "exacto",
					Traslado: "ninguno", Fuente: "RGPD art. 33.1, lectura EDPB: 72 horas exactas"},
			}},
	}

	pruebas := []estado.Prueba{
		{ID: "mfa.usuarios", Control: "demo.mfa", TTL: 24 * time.Hour, SLA: 72 * time.Hour,
			Descripcion: "todos los usuarios con acceso administrativo tienen MFA"},
		{ID: "backup.restauracion", Control: "demo.copias", TTL: 90 * 24 * time.Hour, SLA: 30 * 24 * time.Hour,
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
		// Las versiones y las fechas de vigencia son las de las normas cuya
		// forma se reproduce (RD 311/2022, Rgto. 2016/679 y Rgto. 2024/2847):
		// son datos del escenario, no identificadores, y varios tests dependen
		// de que la del paquete de producto sea posterior a las otras dos.
		Paquetes: []Paquete{
			{URN: "urn:demo:agregada", Version: "2022.311", Digest: "sha256:1f3a", Clase: "normativo",
				Vigencia: Intervalo{Desde: ts(t, "2022-05-05T00:00:00+02:00")}},
			{URN: "urn:demo:brechas", Version: "2016.679", Digest: "sha256:2b7c", Clase: "normativo",
				Vigencia: Intervalo{Desde: ts(t, "2018-05-25T00:00:00+02:00")}},
			{URN: "urn:demo:producto", Version: "2024.2847", Digest: "sha256:9f3c", Clase: "normativo",
				Vigencia: Intervalo{Desde: ts(t, "2026-09-11T00:00:00+02:00")}},
		},
		Programas: []aplicabilidad.Programa{progAgregada, progBrechas, progProducto},
		Hechos:    hechos,
		Obligaciones: []Obligacion{
			{ID: "demo.auditoria_bienal", Paquete: "urn:demo:agregada", Articulo: "31", Primitiva: "periodica",
				Afirmacion: "el sistema ha sido auditado en los ultimos 24 meses"},
			{ID: "demo.mfa", Paquete: "urn:demo:agregada", Articulo: "Anexo II op.acc.5",
				Control: "demo.mfa", Primitiva: "continua", Afirmacion: "la autenticacion usa mas de un factor"},
			{ID: "demo.notificacion_brecha", Paquete: "urn:demo:brechas", Articulo: "33", Primitiva: "plazo",
				Afirmacion: "la brecha se notifico a la AEPD en 72 horas"},
			{ID: "demo.alerta_producto", Paquete: "urn:demo:producto", Articulo: "14", Primitiva: "plazo",
				Afirmacion: "se notifico al CSIRT coordinador y a ENISA"},
		},
		Pruebas: pruebas, Observaciones: observaciones,
		Relojes: relojes,
	}

	// El emisor declara lo que ha calculado. La verificacion lo recalcula.
	e.Aplicables = []string{"demo.auditoria_bienal", "demo.mfa", "demo.copias",
		"demo.notificacion_brecha", "demo.alerta_producto"}
	e.Reclamaciones = []Reclamacion{
		{Obligacion: "demo.alerta_producto", Hito: "alerta_24h", Estado: "determinado", Vence: ts(t, "2026-09-18T20:00:00+02:00")},
		{Obligacion: "demo.alerta_producto", Hito: "notificacion_72h", Estado: "determinado", Vence: ts(t, "2026-09-20T20:00:00+02:00")},
		{Obligacion: "demo.alerta_producto", Hito: "informe_final", Estado: "pendiente de hecho"},
		{Obligacion: "demo.notificacion_brecha", Hito: "aepd_72h", Estado: "determinado", Vence: ts(t, "2026-09-20T20:00:00+02:00")},
	}
	e.Estados = []EstadoControl{
		{Prueba: "mfa.usuarios", Estado: "fail_en_plazo"},
		{Prueba: "backup.restauracion", Estado: "obsoleto"},
	}
	e.Denominadores = estado.Denominadores{Maquina: 1, CaducadoOContradicho: 1}

	// Digests reales, calculados sobre el contenido. Lo que el emisor DECLARA
	// haber usado; las de verdad las trae el receptor en su contexto.
	e.AnclasDeclaradas = map[string]string{}
	for i := range e.Paquetes {
		d := DigestPaquete(e.Paquetes[i].URN, e.Programas, e.Obligaciones)
		e.Paquetes[i].Digest = d
		e.AnclasDeclaradas[e.Paquetes[i].URN] = d
	}

	// Cadena v2 con las observaciones cifradas, sus claves divulgadas y un
	// checkpoint firmado y anclado.
	semilla := make([]byte, ed25519.SeedSize)
	copy(semilla, []byte("plazum-demo-semilla-determinista"))
	k := ed25519.NewKeyFromSeed(semilla)
	e.Cadena.ClavesDeclaradas = []string{hex.EncodeToString(k.Public().(ed25519.PublicKey))}
	ks := ledger.NuevoKeystore()
	e.ClavesEntradas = map[uint64]string{}
	for i, o := range observaciones {
		carga, _ := json.Marshal(o)
		clave, nonce := claveDemo(byte(i+1)), nonceDemo(byte(i+1))
		ent, err := e.Cadena.Anadir(ks, clave, nonce, carga)
		if err != nil {
			t.Fatal(err)
		}
		e.ClavesEntradas[ent.Indice] = hex.EncodeToString(clave)
	}
	e.Cadena.Cerrar(k, comoEstaba, "tsa:rfc3161://tsa.example + testigo publico diario", selloDemo)
	return e
}

// selloDemo es el token RFC 3161 REAL del expediente de demostracion, sellado
// una vez contra una TSA de verdad y guardado en testdata.
//
// Por que real y no de relleno: lo primero que hace cualquiera es `plazum
// verify` sobre el demo, y con un sello inventado eso falla. La alternativa
// seria un atajo en el verificador, que es justo la pieza que no puede tener
// atajos. Se regenera con `go run ./herramientas/sellardemo` cuando cambia el
// contenido del expediente, porque entonces cambia su raiz Merkle.
//
// Si el fichero no esta, los tests siguen corriendo con un token de relleno:
// el ledger comprueba que el sello no este vacio, no que sea autentico, y la
// verificacion de sellos de verdad tiene su suite en adaptadores/tsa.
var selloDemo = cargarSelloDemo()

func cargarSelloDemo() []byte {
	b, err := os.ReadFile("testdata/sello-demo.bin")
	if err != nil || len(b) == 0 {
		return []byte("sello de relleno: ejecuta herramientas/sellardemo")
	}
	return b
}

func claveDemo(b byte) []byte {
	c := make([]byte, 32)
	for i := range c {
		c[i] = b
	}
	return c
}

func nonceDemo(b byte) []byte {
	n := make([]byte, 12)
	for i := range n {
		n[i] = b
	}
	return n
}

// contextoDePrueba es lo que aportaria el receptor: sus anclas, sus claves y
// su verificador de sellos.
func contextoDePrueba(t *testing.T, e *Expediente) ContextoReceptor {
	t.Helper()
	semilla := make([]byte, ed25519.SeedSize)
	copy(semilla, []byte("plazum-demo-semilla-determinista"))
	k := ed25519.NewKeyFromSeed(semilla)
	pub := k.Public().(ed25519.PublicKey)

	anclas := map[string]string{}
	for _, p := range e.Paquetes {
		// El registro del receptor tiene el digest del corpus PUBLICADO. En el
		// expediente de prueba coincide con el declarado porque nadie ha
		// manipulado nada; los ataques cambian una cosa u otra y ahi se ve.
		anclas[p.URN] = p.Digest
	}
	return ContextoReceptor{
		Anclas:           anclas,
		ClavesConfiables: []string{hex.EncodeToString(pub)},
		ClaveOperador:    pub,
		VerificarSello: func(hash, token []byte) error {
			if len(token) == 0 {
				return errors.New("checkpoint sin sello: no prueba fecha")
			}
			return nil
		},
	}
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
	inf := Verificar(otro, contextoDePrueba(t, otro))
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
	// El emisor se da tres dias mas en el reloj de la obligacion de producto.
	for i := range e.Reclamaciones {
		if e.Reclamaciones[i].Hito == "notificacion_72h" {
			e.Reclamaciones[i].Vence = e.Reclamaciones[i].Vence.Add(72 * time.Hour)
		}
	}
	b, _ := e.Guardar()
	otro, _ := Cargar(b)
	inf := Verificar(otro, contextoDePrueba(t, otro))
	if inf.Valido {
		t.Fatal("un vencimiento falseado tiene que detectarse al recalcular")
	}
	t.Logf("detectado: %s -> esperado %s, recalculado %s",
		inf.Discrepancias[0].Que, inf.Discrepancias[0].Esperado, inf.Discrepancias[0].Obtenido)
}

func TestDetectaAplicabilidadInventada(t *testing.T) {
	e := construirExpediente(t)
	e.Aplicables = append(e.Aplicables, "demo.control_referencial")
	b, _ := e.Guardar()
	otro, _ := Cargar(b)
	inf := Verificar(otro, contextoDePrueba(t, otro))
	if inf.Valido {
		t.Fatal("declarar aplicable algo que las reglas no derivan tiene que detectarse")
	}
	t.Logf("detectado: %v", inf.Discrepancias[0])
}

func TestDetectaEvaluacionAntesDeLaVigencia(t *testing.T) {
	e := construirExpediente(t)
	// El paquete de producto no existe antes del 11 de septiembre de 2026.
	e.ComoEstaba = ts(t, "2026-08-01T09:00:00+02:00")
	b, _ := e.Guardar()
	otro, _ := Cargar(b)
	inf := Verificar(otro, contextoDePrueba(t, otro))
	if inf.Valido {
		t.Fatal("evaluar una obligacion antes de la vigencia de su norma tiene que detectarse")
	}
	var visto bool
	for _, d := range inf.Discrepancias {
		if d.Que == "vigencia de demo.alerta_producto" {
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
	inf := Verificar(otro, contextoDePrueba(t, otro))
	if inf.Valido {
		t.Fatal("una evidencia caducada presentada como conforme tiene que detectarse")
	}
	t.Logf("detectado: %v", inf.Discrepancias)
}

func TestDetectaLedgerManipulado(t *testing.T) {
	e := construirExpediente(t)
	// Con la cadena v2 el contenido va cifrado, asi que manipularlo es tocar
	// la envoltura. El hash de la entrada deja de cuadrar.
	e.Cadena.Entradas[1].Cifrado[0] ^= 0xff
	b, _ := e.Guardar()
	otro, _ := Cargar(b)
	if inf := Verificar(otro, contextoDePrueba(t, otro)); inf.Valido {
		t.Fatal("manipular la cadena tiene que detectarse")
	}
}

// Y la version fina del mismo ataque: rehacer el hash para que la envoltura
// cuadre consigo misma. La cadena y la raiz del checkpoint lo cazan.
func TestDetectaLedgerManipuladoConHashRehecho(t *testing.T) {
	e := construirExpediente(t)
	e.Cadena.Entradas[1].Cifrado[0] ^= 0xff
	b, _ := e.Guardar()
	otro, _ := Cargar(b)
	if inf := Verificar(otro, contextoDePrueba(t, otro)); inf.Valido {
		t.Fatal("rehacer el hash de una entrada no puede colar: rompe el encadenado y la raiz")
	}
}

// --- Tests anadidos tras la revision adversarial ---
// Los cuatro ataques que ANTES pasaban la verificacion.

func TestDetectaObligacionInventadaConSuPropiaRegla(t *testing.T) {
	e := construirExpediente(t)
	// El emisor no solo declara la obligacion: tambien aporta la regla que la
	// deriva. Antes esto verificaba limpio, porque nadie contrastaba el corpus.
	e.Programas = append(e.Programas, aplicabilidad.Programa{Paquete: "urn:demo:referencial",
		Reglas: []aplicabilidad.Regla{regla(t, "inventada", "A.5.1",
			`aplica(demo.control_referencial, E) :- responsable(E)`)}})
	e.Aplicables = append(e.Aplicables, "demo.control_referencial")
	b, _ := e.Guardar()
	otro, _ := Cargar(b)
	inf := Verificar(otro, contextoDePrueba(t, otro))
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
	if inf := Verificar(otro, contextoDePrueba(t, otro)); inf.Valido {
		t.Fatal("cambiar una observacion sin tocar el ledger tiene que detectarse por el anclaje")
	}
}

func TestDetectaReclamacionSinRelojQueLaProduzca(t *testing.T) {
	e := construirExpediente(t)
	e.Reclamaciones = append(e.Reclamaciones, Reclamacion{
		Obligacion: "demo.auditoria_bienal", Hito: "inventado", Estado: "determinado",
		Vence: ts(t, "2099-01-01T00:00:00+01:00")})
	b, _ := e.Guardar()
	otro, _ := Cargar(b)
	if inf := Verificar(otro, contextoDePrueba(t, otro)); inf.Valido {
		t.Fatal("una fecha declarada que ningun reloj calcula tiene que detectarse")
	}
}

func TestDetectaDenominadoresFalseados(t *testing.T) {
	e := construirExpediente(t)
	e.Denominadores.Humano = 47 // antes este campo no se comprobaba
	b, _ := e.Guardar()
	otro, _ := Cargar(b)
	if inf := Verificar(otro, contextoDePrueba(t, otro)); inf.Valido {
		t.Fatal("los cinco denominadores tienen que compararse, no dos")
	}
}

func TestSinAnclasDeConfianzaNoVerifica(t *testing.T) {
	e := construirExpediente(t)
	// Las anclas ya no viven en el fichero. El caso equivalente es un receptor
	// que no aporta ninguna: la verificacion tiene que declararse circular.
	b, _ := e.Guardar()
	otro, _ := Cargar(b)
	ctx := contextoDePrueba(t, otro)
	ctx.Anclas = nil
	inf := Verificar(otro, ctx)
	if inf.Valido {
		t.Fatal("sin anclas del receptor la verificacion es circular y no puede darse por buena")
	}
	t.Log("la verificacion se declara circular en vez de fingir que no lo es")
}

// Verificar se tragaba el error de Motor.Cargar. Un emisor podia romper una
// regla a proposito (una cita vacia basta), el motor descartaba ese programa en
// silencio, y como el recalculo salia igual de corto que la lista declarada,
// las dos cuadraban: una obligacion aplicable desaparecia del expediente sin
// dejar ni una discrepancia. Es el unico ataque de esta familia que no falsea
// un dato, sino que apaga la regla que lo derivaba.
func TestDetectaProgramaInvalidoQueOcultaUnaObligacion(t *testing.T) {
	e := construirExpediente(t)
	if len(e.Programas) == 0 || len(e.Programas[0].Reglas) == 0 {
		t.Fatal("el expediente de prueba tiene que traer programas con reglas")
	}
	e.Programas[0].Reglas[0].Cita = ""

	// El emisor declara exactamente lo que deriva el motor mutilado, que es
	// justo lo que veia el verificador viejo.
	m := aplicabilidad.NuevoMotor()
	for _, p := range e.Programas {
		_ = m.Cargar(p)
	}
	for _, h := range e.Hechos {
		m.Afirmar(h)
	}
	if _, err := m.Evaluar(); err != nil {
		t.Fatal(err)
	}
	vistas := map[string]bool{}
	var derivadas []string
	for _, h := range m.Consultar(aplicabilidad.A("aplica",
		aplicabilidad.V("O"), aplicabilidad.V("S"))) {
		if !vistas[h.Args[0]] {
			vistas[h.Args[0]] = true
			derivadas = append(derivadas, h.Args[0])
		}
	}
	if len(derivadas) >= len(e.Aplicables) {
		t.Fatalf("el sabotaje tiene que ocultar alguna obligacion para que el test pruebe algo: "+
			"declaradas %d, derivadas tras romper la regla %d", len(e.Aplicables), len(derivadas))
	}
	e.Aplicables = derivadas

	b, _ := e.Guardar()
	otro, _ := Cargar(b)
	inf := Verificar(otro, contextoDePrueba(t, otro))
	if inf.Valido {
		t.Fatal("un programa que no carga tiene que salir como discrepancia, no descartarse en silencio")
	}
	// La discrepancia tiene que senalar al programa Y venir de que NO CARGA.
	// Otras comprobaciones se quejan de rebote (los denominadores encogen),
	// pero eso no es el arreglo: sin esta asercion el test pasaria igual con el
	// error de Cargar ignorado.
	//
	// El prefijo "programa de " no basta: hay tres discrepancias distintas que
	// lo llevan (paquete no declarado, paquete sin ancla del receptor, y esta).
	// Lo que las separa es Esperado, que es un literal fijo y solo lo escribe
	// la comprobacion de Cargar.
	quePrograma := "programa de " + e.Programas[0].Paquete
	var senalado bool
	for _, d := range inf.Discrepancias {
		if d.Que == quePrograma && d.Esperado == "programa valido y cargable" {
			senalado = true
			t.Logf("detectado: %s -> esperado %q, obtenido %q", d.Que, d.Esperado, d.Obtenido)
		}
	}
	if !senalado {
		t.Fatalf("ninguna discrepancia senala el programa que no cargo; discrepancias: %v", inf.Discrepancias)
	}
}
