package expediente

import (
	"crypto/ed25519"
	"strings"
	"testing"

	"plazum/nucleo/estado"
	"plazum/nucleo/ledger"
)

// Ataques de la revision hostil de la etapa 1 sobre el expediente. Se
// escribieron en rojo, como hallazgos, y se quedan como regresion.

// sabotearObligacionInventada es el ataque 10 en su forma desnuda: el emisor se
// inventa una obligacion DENTRO de un paquete ya anclado, aporta la regla que la
// deriva y recalcula el digest del paquete para que salga de su propio
// contenido. Lo unico que queda por decidir es que escribe en AnclasDeclaradas,
// que es lo que separa las dos capas del apartado 2 de Verificar.
func sabotearObligacionInventada(t *testing.T, e *Expediente, urn string) {
	t.Helper()
	var tocado bool
	for i := range e.Programas {
		if e.Programas[i].Paquete != urn {
			continue
		}
		tocado = true
		e.Programas[i].Reglas = append(e.Programas[i].Reglas,
			regla(t, "inventada", "Anexo II op.exp.99",
				`aplica(demo.inventada, E) :- responsable(E)`))
	}
	if !tocado {
		t.Fatalf("el sabotaje tiene que caer sobre un paquete que exista; %s no esta en el expediente", urn)
	}
	e.Obligaciones = append(e.Obligaciones, Obligacion{
		ID: "demo.inventada", Paquete: urn, Articulo: "Anexo II op.exp.99", Primitiva: "continua",
		Control: "demo.mfa", Afirmacion: "obligacion que no existe en el BOE",
	})
	e.Aplicables = append(e.Aplicables, "demo.inventada")
	for i, p := range e.Paquetes {
		e.Paquetes[i].Digest = DigestPaquete(p.URN, e.Programas, e.Obligaciones)
	}
}

// ATAQUE 10, el bloqueante. El emisor se inventa una obligacion DENTRO de un
// paquete ya anclado, aporta la regla que la deriva, recalcula el digest y se
// escribe el ancla que cuadra. Antes verificaba limpio, porque las anclas
// viajaban en el propio fichero y la comprobacion anti-circular comparaba al
// emisor consigo mismo.
//
// HALLAZGO (P1 12). Este test afirmaba solo sobre inf.Valido, y en este
// escenario saltan DOS comprobaciones: la declarativa (2.a, "ancla declarada
// de X") y la sustantiva (2.b, "contenido de X"). Apagando la sustantiva, que
// es la que su comentario promete, el test seguia verde porque lo salvaba la
// declarativa. Un test en verde que no prueba lo que su nombre dice es
// exactamente el patron "tapado" del que este proyecto se defiende. Ahora
// afirma sobre la discrepancia SUSTANTIVA por su texto, asi que apagar 2.b lo
// pone rojo y apagar 2.a no, que es lo que hay que poder distinguir.
func TestHostilElEmisorYaNoSeFabricaSusPropiasAnclas(t *testing.T) {
	e := construirExpediente(t)
	// El contexto del receptor se toma ANTES del sabotaje: es lo que el auditor
	// tiene en su registro, y el emisor no puede tocarlo.
	ctx := contextoDePrueba(t, e)

	const urn = "urn:demo:agregada"
	sabotearObligacionInventada(t, e, urn)
	// Y se escribe a si mismo el ancla que cuadra con su contenido nuevo.
	for _, p := range e.Paquetes {
		e.AnclasDeclaradas[p.URN] = p.Digest
	}

	b, err := e.Guardar()
	if err != nil {
		t.Fatal(err)
	}
	otro, err := Cargar(b)
	if err != nil {
		t.Fatal(err)
	}
	inf := Verificar(otro, ctx)
	if inf.Valido {
		t.Fatal("el emisor se ha inventado una obligacion y se ha escrito el ancla que cuadra: " +
			"contra las anclas del RECEPTOR eso no puede verificar")
	}
	// Y lo que tiene que cazarlo es el RECALCULO DEL CONTENIDO contra el
	// registro del receptor, no el contraste de lo que el emisor declara.
	exigeDiscrepancia(t, inf, "contenido de "+urn)
	t.Logf("detectado: %v", inf.Discrepancias)
}

// El mismo ataque 10 con la capa declarativa AMORDAZADA: el emisor recalcula el
// digest de su contenido falseado, pero en anclas_declaradas escribe justo lo
// que el receptor tiene en su registro. Asi el contraste de lo declarado no
// puede saltar (declarado == registro) y solo queda en pie el recalculo del
// contenido.
//
// Es el emisor mas listo, y por eso es el test que de verdad mide la cobertura:
// si algun dia el ataque 10 vuelve a descansar sobre la capa declarativa, aqui
// no hay capa declarativa que lo sostenga y el expediente verificaria limpio.
func TestHostilElAnclaDeclaradaImpecableNoTapaElContenidoFalseado(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)

	const urn = "urn:demo:agregada"
	sabotearObligacionInventada(t, e, urn)
	// Lo que el auditor espera leer, palabra por palabra.
	for u, a := range ctx.Anclas {
		e.AnclasDeclaradas[u] = a
	}

	b, err := e.Guardar()
	if err != nil {
		t.Fatal(err)
	}
	otro, err := Cargar(b)
	if err != nil {
		t.Fatal(err)
	}
	inf := Verificar(otro, ctx)
	if inf.Valido {
		t.Fatal("declarar el ancla que el receptor espera no puede tapar un contenido que no es el anclado")
	}
	// La capa declarativa esta muda por construccion. Se afirma, porque si
	// dejara de estarlo este test volveria a probar otra cosa que su nombre.
	for _, d := range inf.Discrepancias {
		if strings.HasPrefix(d.Que, "ancla declarada de ") {
			t.Fatalf("la capa declarativa tenia que estar muda en este escenario y ha saltado: %+v", d)
		}
	}
	exigeDiscrepancia(t, inf, "contenido de "+urn)
	// Y es la UNICA secuela: nada mas separa a este emisor de un expediente
	// limpio. Si algun dia lo caza otra comprobacion, hay que releer esto antes
	// de relajar la de aqui.
	if len(inf.Discrepancias) != 1 {
		t.Fatalf("el recalculo del contenido es lo unico que para este ataque, y hay %d "+
			"discrepancia(s): %+v", len(inf.Discrepancias), inf.Discrepancias)
	}
}

// Control negativo del anterior: sin sabotaje, el mismo expediente verifica
// contra el mismo contexto. Sin esto, el test de arriba pasaria aunque la
// verificacion rechazara todo por cualquier motivo.
func TestHostilSinSabotajeElExpedienteVerifica(t *testing.T) {
	e := construirExpediente(t)
	inf := Verificar(e, contextoDePrueba(t, e))
	if !inf.Valido {
		t.Fatalf("el expediente limpio tiene que verificar: %v", inf.Discrepancias)
	}
}

// La otra capa, aislada al reves: el CONTENIDO cuadra con el registro del
// receptor (la capa sustantiva esta muda por construccion) y lo unico que no
// cuadra es lo que el emisor declara haber usado. Eso se informa, en vez de
// callarse.
func TestHostilElAnclaDeclaradaQueNoCuadraSeInforma(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	e.AnclasDeclaradas["urn:demo:agregada"] = "sha256:otra-cosa"

	inf := Verificar(e, ctx)
	exigeDiscrepancia(t, inf, "ancla declarada de urn:demo:agregada")
	// El contenido no se ha tocado, asi que 2.b no puede estar sosteniendo este
	// test. Sin esta asercion volveriamos a no saber cual de las dos capas cazo.
	for _, d := range inf.Discrepancias {
		if strings.HasPrefix(d.Que, "contenido de ") || strings.HasPrefix(d.Que, "digest de ") {
			t.Fatalf("la capa sustantiva tenia que estar muda en este escenario y ha saltado: %+v", d)
		}
	}
}

// ATAQUE 11. Un expediente sin paquetes y un receptor sin anclas: la
// verificacion tiene que declararse circular en vez de dar un visto bueno.
func TestHostilSinAnclasDelReceptorNoHayVistoBueno(t *testing.T) {
	e := construirExpediente(t)
	inf := Verificar(e, ContextoReceptor{})
	if inf.Valido {
		t.Fatal("sin nada que el receptor aporte no se puede verificar nada")
	}
}

// El emisor oculta una entrada de la cadena: ni divulga su clave ni pone
// lapida. Antes no habia forma de notarlo porque el ledger iba en claro.
func TestHostilEntradaOcultaSinLapidaSeDetecta(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	delete(e.ClavesEntradas, 1) // la observacion incomoda

	inf := Verificar(e, ctx)
	if inf.Valido {
		t.Fatal("una entrada sin clave y sin lapida es contenido oculto sin justificar")
	}
	var visto bool
	for _, d := range inf.Discrepancias {
		if d.Que == "entrada 1 de la cadena" {
			visto = true
		}
	}
	if !visto {
		t.Fatalf("la discrepancia tiene que senalar la entrada oculta: %v", inf.Discrepancias)
	}
}

// Y una clave divulgada que no abre su entrada tampoco cuela: es donde el
// compromiso de clave deja de ser teoria.
func TestHostilClaveDivulgadaQueNoAbreSeDetecta(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	e.ClavesEntradas[1] = e.ClavesEntradas[0] // la clave de otra entrada

	inf := Verificar(e, ctx)
	if inf.Valido {
		t.Fatal("una clave que no compromete esa entrada no puede darse por buena")
	}
}

// --- Borrado legal y estados huerfanos (P1 10) ---

// suprimirEnPruebas hace un borrado legal completo sobre el expediente de
// prueba: lapida firmada por el operador, clave divulgada retirada y la
// observacion fuera de la lista visible, que es lo que hace un emisor honesto.
// Devuelve la observacion que se llevo por delante.
//
// El keystore es uno nuevo a proposito: aqui no se prueba la destruccion de la
// clave en el almacen del emisor (eso vive en nucleo/ledger), sino lo que el
// receptor puede deducir del fichero que le llega.
func suprimirEnPruebas(t *testing.T, e *Expediente, indice uint64) estado.Observacion {
	t.Helper()
	semilla := make([]byte, ed25519.SeedSize)
	copy(semilla, []byte("plazum-demo-semilla-determinista"))
	k := ed25519.NewKeyFromSeed(semilla)
	if _, err := e.Cadena.Borrar(ledger.NuevoKeystore(), k, indice,
		"RGPD art. 17", "2026-09-18T09:30:00Z"); err != nil {
		t.Fatal(err)
	}
	delete(e.ClavesEntradas, indice)
	if indice >= uint64(len(e.Observaciones)) {
		t.Fatalf("la entrada %d no tiene observacion visible que retirar", indice)
	}
	o := e.Observaciones[indice]
	var quedan []estado.Observacion
	for i := range e.Observaciones {
		if uint64(i) != indice {
			quedan = append(quedan, e.Observaciones[i])
		}
	}
	e.Observaciones = quedan
	return o
}

// EL ATAQUE QUE MOTIVA LA REGLA. La observacion de la entrada 1 es la que FALLA
// (u-interventor sin MFA). Borrarla con base legal deja a la prueba con solo la
// observacion que pasa, asi que el recalculo a secas daria "pass": un borrado
// legal que convierte un incumplimiento en un conforme.
//
// Contra eso, la prueba cuya evidencia se suprimio vale obsoleto pase lo que
// pase. Este test es el control de esa linea: apagarla lo pone rojo.
func TestHostilElBorradoLegalNoBlanqueaUnIncumplimiento(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	o := suprimirEnPruebas(t, e, 1)
	if o.Satisfecho {
		t.Fatalf("el ataque solo tiene sentido borrando la observacion que falla: %+v", o)
	}
	e.SupresionesDeEvidencia = []SupresionDeEvidencia{{Entrada: 1, Prueba: o.Prueba}}
	// Y el emisor declara lo que dicen las que sobreviven.
	for i := range e.Estados {
		if e.Estados[i].Prueba == o.Prueba {
			e.Estados[i].Estado = "pass"
		}
	}

	inf := Verificar(e, ctx)
	if inf.Valido {
		t.Fatal("borrar la evidencia que fallaba no puede sacar a nadie de un fallo a un pass")
	}
	exigeDiscrepanciaPor(t, inf, "estado de "+o.Prueba, "pass")
}

// El emisor jura haber suprimido una entrada y publica su clave en el mismo
// fichero. La supresion que declara no ha ocurrido, y ademas se estaria
// cobrando el obsoleto sin pagar el borrado.
func TestHostilLapidaConLaClaveTodaviaDivulgada(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	clave := e.ClavesEntradas[1]
	o := suprimirEnPruebas(t, e, 1)
	e.ClavesEntradas[1] = clave // la vuelve a publicar
	e.SupresionesDeEvidencia = []SupresionDeEvidencia{{Entrada: 1, Prueba: o.Prueba}}

	inf := Verificar(e, ctx)
	if inf.Valido {
		t.Fatal("publicar la clave de una entrada que se jura destruida es una supresion fingida")
	}
	exigeDiscrepancia(t, inf, "supresion de la entrada 1")
}

// CONTROL NEGATIVO de todo el apartado 5c, y la propiedad que pedia el P1 10:
// declarado como lo que es, el borrado legal deja el expediente VALIDO, sin
// discrepancias, y el control que se quedo sin apoyo se ve en el informe con su
// base legal. Sin este test, los de arriba pasarian igual con un verificador que
// rechazara cualquier expediente con lapidas, que seria hacer desaparecer el
// problema en vez de resolverlo.
func TestElBorradoLegalDeclaradoVerificaYElHuerfanoSeVe(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	o := suprimirEnPruebas(t, e, 1)
	e.SupresionesDeEvidencia = []SupresionDeEvidencia{{Entrada: 1, Prueba: o.Prueba}}
	for i := range e.Estados {
		if e.Estados[i].Prueba != o.Prueba {
			continue
		}
		if e.Estados[i].Estado != "fail_en_plazo" {
			t.Fatalf("el escenario cuenta con que %s venia de un fallo dentro de plazo, y viene de %s",
				o.Prueba, e.Estados[i].Estado)
		}
		e.Estados[i].Estado = "obsoleto"
	}
	// Sale del cajon de maquina y entra en el de caducado_o_contradicho.
	e.Denominadores.Maquina--
	e.Denominadores.CaducadoOContradicho++

	inf := Verificar(e, ctx)
	if !inf.Valido || len(inf.Discrepancias) != 0 {
		t.Fatalf("el borrado legal declarado como lo que es tiene que verificar limpio: %+v",
			inf.Discrepancias)
	}
	// Y no puede volverse invisible por haber verificado. Prefijo entero, que es
	// lo falsable: prueba, indice, base legal e instante.
	prefijo := "estado de " + o.Prueba +
		": sin evidencia por supresion legal (entrada 1, RGPD art. 17, el 2026-09-18T09:30:00Z)"
	var visto bool
	for _, c := range inf.Comprobaciones {
		if strings.HasPrefix(c, prefijo) && strings.Contains(c, "obsoleto") {
			visto = true
		}
	}
	if !visto {
		t.Fatalf("un control sin evidencia sigue siendo un riesgo y tiene que verse:\n"+
			"  quiero un ok que empiece por %q\n  tengo  %v", prefijo, inf.Comprobaciones)
	}
}

// Una supresion que el emisor declara que no dejo sin apoyo a ningun control.
// Es una afirmacion suya, no una exencion: la entrada sigue exigiendo lapida y
// la afirmacion queda impresa. Lo que no puede haber es borrados mudos.
func TestElBorradoQueNoDejaControlesCojosSeDeclaraIgual(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	// La entrada 2 sostiene backup.restauracion, que ya estaba obsoleto por
	// caducidad: retirarla no cambia su estado recalculado.
	o := suprimirEnPruebas(t, e, 2)
	if o.Prueba != "backup.restauracion" {
		t.Fatalf("el escenario cuenta con que la entrada 2 es la de backup: %+v", o)
	}
	e.SupresionesDeEvidencia = []SupresionDeEvidencia{{Entrada: 2, Prueba: ""}}

	inf := Verificar(e, ctx)
	if !inf.Valido {
		t.Fatalf("declarar que un borrado no dejo controles cojos es una afirmacion legitima: %+v",
			inf.Discrepancias)
	}
	quiere := "supresion sin efecto en controles: entrada 2, RGPD art. 17, el 2026-09-18T09:30:00Z" +
		"; el emisor afirma que esa evidencia no sostenia el estado de ninguna prueba"
	var visto bool
	for _, c := range inf.Comprobaciones {
		if c == quiere {
			visto = true
		}
	}
	if !visto {
		t.Fatalf("la afirmacion tiene que quedar impresa para que se le pueda pedir cuentas:\n"+
			"  quiero %q\n  tengo  %v", quiere, inf.Comprobaciones)
	}
}

// ATAQUE 13, y el mas barato de todos: NO HACE FALTA BORRAR NADA.
//
// Lo encontro la pasada adversaria sobre el arreglo del borrado legal, y lo peor
// es que era PREEXISTENTE: reproducido sobre la base, o sea que llevaba ahi
// desde que existe el verificador y ninguna de las trece rondas lo vio.
//
// El emisor tiene un control en fail_en_plazo. No toca la cadena, no destruye
// ninguna clave, no pone ninguna lapida y no declara ninguna supresion. Se
// limita a QUITAR de e.Observaciones la observacion que falla, dejando su
// entrada y su clave publicadas e intactas. El verificador recalculaba el estado
// con las que quedaban, salia pass, y devolvia Valido=true con cero
// discrepancias.
//
// El apartado 5b solo miraba observaciones -> cadena. La direccion contraria,
// cadena -> observaciones, no la miraba nadie, y por ahi se salta ENTERA la
// maquinaria de borrado legal: lapidas, keystore, declaracion y obsoleto
// forzado. Toda esa maquinaria estaba defendiendo una puerta con la pared
// abierta al lado.
func TestHostilRetirarUnaObservacionSinBorrarNadaNoBlanquea(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)

	// La que falla, sin tocar la cadena ni las claves.
	var fallona estado.Observacion
	var quedan []estado.Observacion
	for _, o := range e.Observaciones {
		if !o.Satisfecho && fallona.Prueba == "" {
			fallona = o
			continue
		}
		quedan = append(quedan, o)
	}
	if fallona.Prueba == "" {
		t.Fatal("el escenario tiene que traer una observacion que falla, o el ataque no existe")
	}
	e.Observaciones = quedan
	// Y el emisor declara el estado que sale de las que sobreviven.
	for i := range e.Estados {
		if e.Estados[i].Prueba == fallona.Prueba {
			e.Estados[i].Estado = "pass"
		}
	}

	inf := Verificar(e, ctx)
	if inf.Valido {
		t.Fatal("HALLAZGO: retirar de la lista la observacion que falla, con su entrada y su " +
			"clave intactas en la cadena, saca a un control de fail a pass y el expediente " +
			"verifica limpio. Toda la maquinaria de borrado legal se salta por aqui")
	}
	// Por el "Esperado", que es un literal fijo del verificador y no depende de
	// que entrada ni que prueba haya salido. Comparar el "Que" ataria el test al
	// numero de entrada del escenario, y entonces reordenar el escenario lo
	// pondria rojo sin que nada estuviera mal.
	exigeDiscrepanciaPor(t, inf, "entrada 1 de la cadena (mfa.usuarios/u-interventor)",
		"declarada en Observaciones, o suprimida con lapida y clave destruida")
}

// CONTROL NEGATIVO del de arriba: sin retirar nada, el mismo escenario verifica.
// Sin esto, un verificador que rechazara cualquier expediente pasaria el ataque
// 13 sin comprobar nada.
func TestHostilSinRetirarNadaLaCadenaYLasObservacionesCuadran(t *testing.T) {
	e := construirExpediente(t)
	if inf := Verificar(e, contextoDePrueba(t, e)); !inf.Valido {
		t.Fatalf("CONTROL NEGATIVO EN ROJO: el escenario intacto tiene que verificar, y dio %v",
			inf.Discrepancias)
	}
}

// La mutacion DEBILITADORA del ataque 13, que el escenario base no cazaba.
//
// El test de arriba usa un expediente SIN lapidas, asi que una excusa colectiva
// del tipo "si hay alguna lapida, no mires ninguna entrada" pasaba con la suite
// entera en verde. Es exactamente el patron que la pasada adversaria encontro en
// el apartado hermano: el unico escenario probado tenia UNA lapida y CERO
// declaraciones, asi que una excusa que valiera para todas se colaba.
//
// Aqui el emisor hace un borrado legal HONESTO de una entrada, declarado con su
// base legal y su supresion, y ADEMAS retira por lo bajo la observacion de otra
// entrada distinta, con su clave publicada. La parte honesta no puede comprar la
// deshonesta.
func TestHostilUnBorradoLegalHonradoNoExcusaRetirarOtraObservacion(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)

	// 1. El borrado legal de verdad, declarado como toca.
	suprimida := suprimirEnPruebas(t, e, 1)
	e.SupresionesDeEvidencia = []SupresionDeEvidencia{{Entrada: 1, Prueba: suprimida.Prueba}}
	for i := range e.Estados {
		if e.Estados[i].Prueba == suprimida.Prueba {
			e.Estados[i].Estado = "obsoleto"
		}
	}

	// 2. Y por lo bajo, se retira OTRA observacion, sin lapida y con su clave
	//    publicada. Es el ataque 13 escondido detras de un borrado honrado.
	var retirada estado.Observacion
	var quedan []estado.Observacion
	for _, o := range e.Observaciones {
		if o.Prueba != suprimida.Prueba && retirada.Prueba == "" {
			retirada = o
			continue
		}
		quedan = append(quedan, o)
	}
	if retirada.Prueba == "" {
		t.Fatal("hace falta una segunda observacion de otra prueba, o este caso no anade nada")
	}
	e.Observaciones = quedan

	inf := Verificar(e, ctx)
	if inf.Valido {
		t.Fatalf("HALLAZGO: un borrado legal honrado en una entrada excusa retirar la "+
			"observacion de OTRA entrada distinta. La observacion retirada era %s/%s y su "+
			"clave sigue publicada", retirada.Prueba, retirada.Recurso)
	}
	exigeDiscrepanciaPor(t, inf, "entrada 2 de la cadena (backup.restauracion/sede-electronica)",
		"declarada en Observaciones, o suprimida con lapida y clave destruida")
}
