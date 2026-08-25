package expediente

import (
	"testing"
	"time"
)

// Controles negativos aislados, uno por comprobacion del verificador.
//
// De donde salen: la revision hostil muto las 23 comprobaciones de Verificar
// una a una y 14 no las rompia ningun test. No estaban sin probar, estaban
// TAPADAS: otro control cazaba el mismo escenario de refilon, asi que quitar
// una sola no ponia nada en rojo. El caso peor fue el anti-circular, cuyas dos
// comprobaciones se cubrian mutuamente y juntas no cubrian el ataque real.
//
// Cada test de aqui afirma sobre el TEXTO de la discrepancia concreta, no sobre
// inf.Valido. Esa es la diferencia: si se quita la comprobacion que le toca, el
// test se pone rojo aunque el expediente siga siendo invalido por otra via.

func exigeDiscrepancia(t *testing.T, inf Informe, que string) {
	t.Helper()
	for _, d := range inf.Discrepancias {
		if d.Que == que {
			return
		}
	}
	t.Fatalf("faltaba la discrepancia %q. Las que hubo: %v", que, inf.Discrepancias)
}

// exigeDiscrepanciaPor es lo mismo cuando VARIAS comprobaciones comparten el
// mismo Que, como las tres del apartado 5c. Lo que las separa es Esperado, que
// es un literal fijo del verificador; el mismo patron que ya usa
// TestDetectaProgramaInvalidoQueOcultaUnaObligacion. Sin esto, un test podria
// darse por bueno con la discrepancia de la comprobacion de al lado.
func exigeDiscrepanciaPor(t *testing.T, inf Informe, que, esperado string) {
	t.Helper()
	for _, d := range inf.Discrepancias {
		if d.Que == que && d.Esperado == esperado {
			return
		}
	}
	t.Fatalf("faltaba la discrepancia %q con esperado %q. Las que hubo: %+v",
		que, esperado, inf.Discrepancias)
}

func TestControlVersionDelFormato(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	e.Version = "plazum-expediente-v1"
	exigeDiscrepancia(t, Verificar(e, ctx), "version")
}

func TestControlCadenaRota(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	e.Cadena.Entradas[1].Previo = []byte("no encadena")
	exigeDiscrepancia(t, Verificar(e, ctx), "cadena")
}

func TestControlReceptorSinAnclas(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	ctx.Anclas = nil
	exigeDiscrepancia(t, Verificar(e, ctx), "anclas de confianza")
}

func TestControlDigestQueNoSaleDelContenido(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	e.Paquetes[0].Digest = "sha256:me-lo-invento"
	exigeDiscrepancia(t, Verificar(e, ctx), "digest de urn:demo:agregada")
}

func TestControlPaqueteQueElReceptorNoReconoce(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	delete(ctx.Anclas, "urn:demo:agregada")
	exigeDiscrepancia(t, Verificar(e, ctx), "ancla de urn:demo:agregada")
}

func TestControlContenidoQueNoCuadraConElRegistro(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	// El digest declarado sale del contenido, pero el registro del receptor
	// dice otra cosa: el corpus del emisor no es el publicado.
	ctx.Anclas["urn:demo:agregada"] = "sha256:lo-que-dice-el-registro"
	exigeDiscrepancia(t, Verificar(e, ctx), "contenido de urn:demo:agregada")
}

func TestControlObligacionDeUnPaqueteNoDeclarado(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	e.Obligaciones = append(e.Obligaciones, Obligacion{
		ID: "fantasma.1", Paquete: "urn:demo:referencial", Articulo: "A.5.1", Primitiva: "continua"})
	exigeDiscrepancia(t, Verificar(e, ctx), "paquete de fantasma.1")
}

func TestControlProgramaNoEstratificable(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	// Negacion en ciclo: p depende de no-r y r depende de no-p. El motor tiene
	// que rechazarlo en vez de no terminar.
	for i := range e.Programas {
		if e.Programas[i].Paquete != "urn:demo:agregada" {
			continue
		}
		e.Programas[i].Reglas = append(e.Programas[i].Reglas,
			regla(t, "ciclo_p", "sintetica", `p(X) :- responsable(X), not r(X)`),
			regla(t, "ciclo_r", "sintetica", `r(X) :- responsable(X), not p(X)`))
	}
	for i, p := range e.Paquetes {
		calc := DigestPaquete(p.URN, e.Programas, e.Obligaciones)
		e.Paquetes[i].Digest = calc
		ctx.Anclas[p.URN] = calc
	}
	exigeDiscrepancia(t, Verificar(e, ctx), "aplicabilidad")
}

func TestControlAplicableDerivadoQueNoSeDeclara(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	var quedan []string
	for _, a := range e.Aplicables {
		if a != "demo.auditoria_bienal" {
			quedan = append(quedan, a)
		}
	}
	e.Aplicables = quedan
	exigeDiscrepancia(t, Verificar(e, ctx), "aplicabilidad de demo.auditoria_bienal")
}

func TestControlRelojConDeclaracionInvalida(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	e.Relojes[0].Hitos[0].Limite = "esto no es una duracion ISO 8601"
	exigeDiscrepancia(t, Verificar(e, ctx), "reloj de demo.alerta_producto")
}

func TestControlHitoCalculadoQueNoSeDeclara(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	var quedan []Reclamacion
	for _, r := range e.Reclamaciones {
		if !(r.Obligacion == "demo.alerta_producto" && r.Hito == "alerta_24h") {
			quedan = append(quedan, r)
		}
	}
	e.Reclamaciones = quedan
	exigeDiscrepancia(t, Verificar(e, ctx), "demo.alerta_producto/alerta_24h")
}

func TestControlEstadoDelHitoFalseado(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	for i := range e.Reclamaciones {
		if e.Reclamaciones[i].Hito == "alerta_24h" {
			e.Reclamaciones[i].Estado = "pendiente de hecho"
		}
	}
	exigeDiscrepancia(t, Verificar(e, ctx), "demo.alerta_producto/alerta_24h (estado)")
}

func TestControlClaveDeEntradaQueNoEsHex(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	e.ClavesEntradas[0] = "zzz-no-es-hexadecimal"
	exigeDiscrepancia(t, Verificar(e, ctx), "clave de la entrada 0")
}

func TestControlClaveDeEntradaQueNoAbre(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	// Hex valido, clave equivocada: la caza el compromiso, antes de GCM.
	e.ClavesEntradas[0] = e.ClavesEntradas[1]
	exigeDiscrepancia(t, Verificar(e, ctx), "entrada 0 de la cadena")
}

func TestControlEstadoDeControlAusente(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	var quedan []EstadoControl
	for _, s := range e.Estados {
		if s.Prueba != "mfa.usuarios" {
			quedan = append(quedan, s)
		}
	}
	e.Estados = quedan
	exigeDiscrepancia(t, Verificar(e, ctx), "estado de mfa.usuarios")
}

// La evaluacion antes de la vigencia ya tenia test, pero afirmaba sobre
// inf.Valido. Aqui queda aislada por su texto.
func TestControlEvaluacionAntesDeLaVigencia(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	e.ComoEstaba = e.ComoEstaba.Add(-10 * 365 * 24 * time.Hour)
	exigeDiscrepancia(t, Verificar(e, ctx), "vigencia de demo.auditoria_bienal")
}

// --- Controles del apartado 5c: supresiones de evidencia (P1 10) ---
//
// Cada uno monta el escenario de forma que SOLO pueda saltar la comprobacion
// que le toca. Las tres del bucle de declaraciones comparten Que, asi que se
// afirman por Esperado.

// Un borrado legal mudo: la cadena lleva lapida y el expediente no dice que
// control se quedo sin evidencia. El receptor no puede razonar sobre eso.
func TestControlLapidaSinDecirQueControlSeQuedaSinEvidencia(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	suprimirEnPruebas(t, e, 1)
	// Sin declaracion ninguna: el bucle de declaraciones no se ejecuta.
	exigeDiscrepancia(t, Verificar(e, ctx), "supresion de evidencia de la entrada 1")
}

// Y al reves: declarar una supresion que la cadena no respalda. Sin esto,
// cualquiera rebaja un control a obsoleto sin haber borrado nada.
func TestControlSupresionDeclaradaSinLapidaQueLaRespalde(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	// Nada suprimido: la cadena no tiene lapidas, asi que el bucle de lapidas
	// sin atribuir tampoco se ejecuta.
	e.SupresionesDeEvidencia = []SupresionDeEvidencia{{Entrada: 1, Prueba: "mfa.usuarios"}}
	exigeDiscrepanciaPor(t, Verificar(e, ctx), "supresion de evidencia declarada de la entrada 1",
		"lapida con base legal, firmada por el operador, en una cadena que verifica")
}

// La misma supresion declarada dos veces: dos declaraciones de un borrado
// dejarian dos controles en obsoleto pagando un solo borrado.
func TestControlSupresionDeclaradaDosVeces(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	suprimirEnPruebas(t, e, 1)
	e.SupresionesDeEvidencia = []SupresionDeEvidencia{
		{Entrada: 1, Prueba: "mfa.usuarios"},
		{Entrada: 1, Prueba: "backup.restauracion"},
	}
	exigeDiscrepanciaPor(t, Verificar(e, ctx), "supresion de evidencia declarada de la entrada 1",
		"una sola declaracion por entrada suprimida")
}

// Una supresion que deja sin evidencia a un control que no existe.
func TestControlSupresionQueSenalaUnaPruebaQueNoExiste(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	suprimirEnPruebas(t, e, 1)
	e.SupresionesDeEvidencia = []SupresionDeEvidencia{{Entrada: 1, Prueba: "prueba.que.no.existe"}}
	exigeDiscrepanciaPor(t, Verificar(e, ctx), "supresion de evidencia declarada de la entrada 1",
		"una prueba declarada en el expediente")
}
