package dutiq

// El ciclo de extremo a extremo, encadenado de verdad con las piezas reales:
//
//	paquete (JSON con linter legal)
//	  -> entrevista y esquema de UI derivados
//	  -> reloj declarado calculado por el motor (dorados contra el texto)
//	  -> observacion -> estado (funcion pura)
//	  -> historia bitemporal
//	  -> evidencia cifrada content-addressed -> ledger v2 comprometido
//	  -> borrado legal con lapida, cadena intacta
//	  -> expediente verificado offline por un tercero
//
// No es un dibujo del ciclo: cada flecha es una llamada a la API real y cada
// paso tiene su asercion. Lo que el ciclo aun no cubre queda dicho al final
// del test, no escondido.

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"os"
	"testing"
	"time"

	"dutiq/nucleo/blobs"
	"dutiq/nucleo/corpus"
	"dutiq/nucleo/estado"
	"dutiq/nucleo/expediente"
	"dutiq/nucleo/historia"
	"dutiq/nucleo/ledger"
)

func TestCicloE2E(t *testing.T) {
	// 1. El corpus publicado carga con su linter legal.
	ps, err := corpus.Cargar("paquetes")
	if err != nil {
		t.Fatal(err)
	}
	// Por URN completo y no por trozo: el corpus son 30 marcos y una subcadena
	// puede casar con mas de uno (una consolidacion, una version derivada), y
	// entonces el ciclo se estaria ejecutando sobre un paquete que no es el que
	// dice el test.
	const urnENS = "urn:es:rd:2022:311"
	var ens *corpus.Paquete
	for _, p := range ps {
		if p.URN == urnENS {
			if ens != nil {
				t.Fatalf("dos paquetes con el URN %s: el ciclo no sabria cual esta probando", urnENS)
			}
			ens = p
		}
	}
	if ens == nil {
		t.Fatalf("el paquete del ENS (%s) tiene que estar publicado", urnENS)
	}

	// 2. El alcance se deriva del paquete: entrevista y formularios, sin UI escrita.
	if n := len(corpus.Entrevista(ps)); n == 0 {
		t.Fatal("la entrevista se deriva de los paquetes instalados")
	}
	if n := len(corpus.EsquemaUI(ps)); n == 0 {
		t.Fatal("el esquema de UI se deriva de los paquetes instalados")
	}

	// 3. El reloj declarado se calcula con el motor y coincide con el texto:
	//    los dorados del paquete son la prueba, y ya cubren art. 31 e INES.
	if errs := corpus.EjecutarDorados(ens); len(errs) != 0 {
		t.Fatalf("los relojes del ENS no cuadran con su texto: %v", errs)
	}

	// 4. Una observacion se convierte en estado con la funcion pura.
	ahora := time.Date(2026, 9, 18, 9, 0, 0, 0, time.UTC)
	prueba := estado.Prueba{ID: "ens.art31.evidencia_auditoria", Control: "ens.art31",
		TTL: 90 * 24 * time.Hour, SLA: 30 * 24 * time.Hour}
	obs := []estado.Observacion{{Prueba: prueba.ID, Recurso: "sede-electronica",
		Satisfecho: true, Recolectada: ahora.Add(-24 * time.Hour), Recolector: "ingesta-manual"}}
	entrada := estado.Calcular(prueba, obs, estado.Contexto{Ahora: ahora, Aplicable: true})
	if entrada.Estado != estado.Pass {
		t.Fatalf("con observacion fresca y satisfecha, el estado es pass: %v", entrada.Estado)
	}

	// 5. El cambio queda en la historia bitemporal y el pasado es consultable.
	h := &historia.Historia{}
	h.Registrar(historia.CambioEstado{Prueba: prueba.ID, De: estado.Obsoleto, A: entrada.Estado,
		InstanteHecho: ahora, InstanteRegistro: ahora.Add(2 * time.Minute), Causa: "observacion"})
	if e, ok := h.EstadoEn(prueba.ID, ahora.Add(time.Hour)); !ok || e != estado.Pass {
		t.Fatalf("la historia reconstruye el estado: %v %v", e, ok)
	}

	// 6. La evidencia con fichero se cifra content-addressed y se encadena en
	//    el ledger v2 con compromiso de clave.
	acta := []byte("acta de la auditoria bienal del ENS, sistema sede-electronica")
	clave, nonce := claveE2E(1), nonceE2E(1)
	blob, err := blobs.Sellar(clave, nonce, acta)
	if err != nil {
		t.Fatal(err)
	}
	cadena, ks := &ledger.CadenaV2{}, ledger.NuevoKeystore()
	if _, err := cadena.Anadir(ks, claveE2E(2), nonceE2E(2), []byte(`{"evidencia":"`+blob.Hash+`"}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := blobs.Abrir(clave, blob); err != nil {
		t.Fatalf("la evidencia se recupera integra: %v", err)
	}

	// 7. El borrado legal: la clave se destruye, la lapida firma la base legal,
	//    la cadena sigue integra y lo informa.
	pub, priv, _ := ed25519.GenerateKey(lectorE2E{})
	cadena.Cerrar(priv, comoEstabaE2E(t), "tsa:rfc3161://tsa.example", []byte("sello e2e"))
	cfE2E := ledger.Confianza{
		ClavesConfiables: []string{hex.EncodeToString(pub)},
		ClaveOperador:    pub,
		VerificarSello: func(hash, token []byte) error {
			if len(token) == 0 {
				return errors.New("checkpoint sin sello")
			}
			return nil
		},
	}
	if _, err := cadena.Borrar(ks, priv, 0, "RGPD art. 17", "2026-09-18T09:30:00Z"); err != nil {
		t.Fatal(err)
	}
	inf2, err := cadena.Verificar(cfE2E)
	if err != nil || len(inf2.Suprimidas) != 1 {
		t.Fatalf("tras el borrado la cadena verifica y lo informa: %v %v", err, inf2.Suprimidas)
	}

	// 8. El expediente completo lo verifica un tercero, offline, sin confiar en
	//    el emisor: el mismo fichero que verifica la CLI.
	//
	//    HALLAZGO DE REVISION HOSTIL: antes este paso cargaba el fichero demo y
	//    ya esta, asi que la flecha "ledger v2 -> borrado legal -> expediente
	//    verificado" NO existia: la CadenaV2 de los pasos 6 y 7 era otro objeto
	//    y el expediente llevaba un ledger v1 en claro. Ahora el expediente
	//    lleva la cadena v2 y el borrado legal se hace SOBRE ELLA, aqui abajo.
	b, err := os.ReadFile("expediente-demo.json")
	if err != nil {
		t.Fatal(err)
	}
	exp, err := expediente.Cargar(b)
	if err != nil {
		t.Fatal(err)
	}
	ctx := contextoE2E(t, exp)
	informe := expediente.Verificar(exp, ctx)
	if !informe.Valido {
		t.Fatalf("el expediente demo debe verificar: %v", informe.Discrepancias)
	}
	if len(exp.Cadena.Entradas) == 0 || len(exp.Cadena.Checkpoints) == 0 {
		t.Fatal("el expediente tiene que llevar la cadena v2 con sus checkpoints, no un ledger en claro")
	}

	// 9. Y el borrado legal SOBRE EL EXPEDIENTE que se acaba de verificar: se
	//    retira la clave divulgada y se pone la lapida. La cadena no se toca,
	//    la raiz del checkpoint no cambia, y la verificacion informa la
	//    supresion con su base legal en vez de gritar manipulacion.
	//
	//    Lo que NO pasa, y aqui ponia que si: el expediente no queda valido.
	//    Retirar la observacion suprimida deja huerfano el estado que sostenia,
	//    y el recalculo lo saca como discrepancia. Esta afirmado abajo, en 9.c.
	kDemo := claveDelOperadorDemo(t)
	ksDemo := ledger.NuevoKeystore()
	if _, err := exp.Cadena.Borrar(ksDemo, kDemo, 1, "RGPD art. 17", "2026-09-18T09:30:00Z"); err != nil {
		t.Fatal(err)
	}
	delete(exp.ClavesEntradas, 1)
	// La observacion suprimida sale tambien de la lista visible: el emisor no
	// puede seguir afirmando lo que ya no puede probar.
	var quedan []estado.Observacion
	for i, o := range exp.Observaciones {
		if i != 1 {
			quedan = append(quedan, o)
		}
	}
	exp.Observaciones = quedan

	tras := expediente.Verificar(exp, ctx)

	// 9.a La supresion se informa, y con la linea exacta que lee el auditor:
	//     indice, base legal e instante. Con una subcadena, un informe que
	//     perdiera el indice o la fecha seguiria pasando por bueno.
	quiere := "supresion: entrada 1 suprimida con base legal RGPD art. 17 el 2026-09-18T09:30:00Z"
	var informaLaSupresion bool
	for _, c := range tras.Comprobaciones {
		if c == quiere {
			informaLaSupresion = true
		}
	}
	if !informaLaSupresion {
		t.Fatalf("tras el borrado legal el expediente tiene que informar la supresion con su base "+
			"legal:\n  quiero %q\n  tengo  %v / %v", quiere, tras.Comprobaciones, tras.Discrepancias)
	}

	// 9.b La cadena de custodia aguanta el borrado: ninguna discrepancia de
	//     "cadena". Es la propiedad que el paso 9 existe para demostrar.
	for _, d := range tras.Discrepancias {
		if d.Que == "cadena" {
			t.Fatalf("el borrado legal no puede romper la cadena de custodia: %+v", d)
		}
	}

	// 9.c HALLAZGO DEL BARRIDO DE ASERCIONES FLOJAS. Aqui no se afirmaba nada
	//     sobre tras.Valido, solo se buscaba la linea de la supresion, asi que
	//     lo que el paso 9 decia de si mismo ("el expediente sigue
	//     verificando") no lo comprobaba nadie. Y es que no se cumple: al
	//     retirar la observacion que sostenia un estado declarado, el recalculo
	//     ya no da ese estado y el verificador lo dice.
	//
	//     Se deja afirmado lo que de verdad pasa, que ademas es coherente: quien
	//     borra la prueba no puede seguir afirmando lo que la prueba sostenia.
	//     Lo que hay que decidir (y no decide este test) es si el expediente
	//     deberia poder declarar "sin evidencia por supresion legal" en vez de
	//     salir como discrepancia. Mientras tanto, la forma exacta queda pinada:
	//     UNA sola discrepancia y justo la del estado que se quedo huerfano.
	if len(tras.Discrepancias) != 1 {
		t.Fatalf("tras el borrado legal la unica secuela esperada es el estado que se quedo "+
			"sin evidencia, y hay %d discrepancia(s): %+v", len(tras.Discrepancias), tras.Discrepancias)
	}
	if d := tras.Discrepancias[0]; d.Que != "estado de mfa.usuarios" ||
		d.Esperado != "fail_en_plazo" || d.Obtenido != "pass" {
		t.Fatalf("la secuela del borrado tiene que ser el estado que sostenia la observacion "+
			"suprimida, y es: %+v", d)
	}

	// Lo que este ciclo AUN no encadena, dicho aqui y no escondido: la
	// aplicabilidad Datalog desde reglas del paquete (las reglas visten JSON en
	// una etapa posterior; el motor ya existe y tiene sus tests), el escalado
	// entregando por un canal real (etapa 4), y la historia dentro del
	// expediente (etapa 1, casilla pendiente). El sello RFC 3161 se verifica
	// aqui con un stub: el verificador real y sus controles negativos viven en
	// adaptadores/tsa, con TSA falsa, porque el CI no sale a la red.
}

// claveDelOperadorDemo reconstruye la clave con la que se firmo el expediente
// demo. Semilla fija: el demo tiene que ser reproducible byte a byte.
func claveDelOperadorDemo(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	semilla := make([]byte, ed25519.SeedSize)
	copy(semilla, []byte("dutiq-demo-semilla-determinista"))
	return ed25519.NewKeyFromSeed(semilla)
}

func comoEstabaE2E(t *testing.T) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, "2026-09-18T09:00:00+02:00")
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// contextoE2E hace de registro del receptor. Aqui toma los digests del propio
// expediente porque lo que se prueba en este test es que las piezas encajan;
// que un emisor NO puede fabricarse sus anclas tiene su propio test en
// nucleo/expediente (TestHostilElEmisorYaNoSeFabricaSusPropiasAnclas).
func contextoE2E(t *testing.T, e *expediente.Expediente) expediente.ContextoReceptor {
	t.Helper()
	pub := claveDelOperadorDemo(t).Public().(ed25519.PublicKey)
	anclas := map[string]string{}
	for _, p := range e.Paquetes {
		anclas[p.URN] = p.Digest
	}
	return expediente.ContextoReceptor{
		Anclas:           anclas,
		ClavesConfiables: []string{hex.EncodeToString(pub)},
		ClaveOperador:    pub,
		VerificarSello: func(hash, token []byte) error {
			if len(token) == 0 {
				return errors.New("checkpoint sin sello")
			}
			return nil
		},
	}
}

func claveE2E(b byte) []byte {
	c := make([]byte, 32)
	for i := range c {
		c[i] = b
	}
	return c
}
func nonceE2E(b byte) []byte {
	n := make([]byte, 12)
	for i := range n {
		n[i] = b
	}
	return n
}

type lectorE2E struct{}

func (lectorE2E) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = byte(i*13 + 5)
	}
	return len(p), nil
}
