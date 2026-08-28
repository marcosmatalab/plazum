package export

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/marcosmatalab/plazum/nucleo/estado"
	"github.com/marcosmatalab/plazum/nucleo/expediente"
	"github.com/marcosmatalab/plazum/nucleo/ledger"
)

// El escenario de prueba es sintetico a proposito: ninguna norma real se nombra
// aqui, ni siquiera en un caso de prueba. Las obligaciones y las bases legales
// llevan URN de demostracion.

func clave(n byte) []byte {
	c := make([]byte, 32)
	for i := range c {
		c[i] = n
	}
	return c
}

func nonce(n byte) []byte {
	c := make([]byte, 12)
	for i := range c {
		c[i] = n
	}
	return c
}

// operador da una clave ed25519 fija: el expediente tiene que salir igual en dos
// ejecuciones, y una clave aleatoria metería una diferencia por ejecucion en el
// campo de clave publica del checkpoint.
func operador() ed25519.PrivateKey { return ed25519.NewKeyFromSeed(clave(7)) }

func comoEstaba() time.Time {
	return time.Date(2026, 9, 18, 9, 0, 0, 0, time.UTC)
}

func carga(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("no se puede serializar la carga de prueba: %v", err)
	}
	return b
}

// escenario construye un expediente con cadena v2, claves divulgadas, un
// checkpoint sellado, dos controles y dos plazos.
type escenario struct {
	exp *expediente.Expediente
	ks  *ledger.Keystore
}

func nuevoEscenario(t *testing.T, cargas ...[]byte) *escenario {
	t.Helper()
	cadena := &ledger.CadenaV2{}
	ks := ledger.NuevoKeystore()
	claves := map[uint64]string{}
	for i, c := range cargas {
		k, nc := clave(byte(i+1)), nonce(byte(i+1))
		e, err := cadena.Anadir(ks, k, nc, c)
		if err != nil {
			t.Fatalf("anadir la entrada %d: %v", i, err)
		}
		claves[e.Indice] = hex.EncodeToString(k)
	}
	cadena.Cerrar(operador(), comoEstaba().Add(-time.Hour), "urn:demo:tsa", []byte("sello de prueba"))

	exp := &expediente.Expediente{
		Version:        "v2",
		Emitido:        comoEstaba(),
		ComoEstaba:     comoEstaba(),
		Organizacion:   "Organizacion de ejemplo",
		Cadena:         *cadena,
		ClavesEntradas: claves,
		Estados: []expediente.EstadoControl{
			{Prueba: "control.uno", Estado: estado.FailVencido.String(), Motivo: "sin remediar"},
			{Prueba: "control.dos", Estado: estado.Pass.String()},
		},
		Reclamaciones: []expediente.Reclamacion{
			{Obligacion: "urn:demo:obligacion:uno", Hito: "aviso", Estado: "determinado",
				Vence: comoEstaba().Add(-24 * time.Hour)},
			{Obligacion: "urn:demo:obligacion:dos", Hito: "informe", Estado: "pendiente de hecho"},
		},
	}
	return &escenario{exp: exp, ks: ks}
}

// suprime pone la lapida SIN retirar la clave divulgada. Es el estado de
// deriva que el export tiene que aguantar, no un descuido del test: la lapida
// viaja en la cadena firmada y la clave vive en un almacen que se replica
// aparte, asi que una copia de seguridad restaurada la devuelve.
func (e *escenario) suprime(t *testing.T, indice uint64, baseLegal, cuando, prueba string) {
	t.Helper()
	if _, err := e.exp.Cadena.Borrar(e.ks, operador(), indice, baseLegal, cuando); err != nil {
		t.Fatalf("borrar la entrada %d: %v", indice, err)
	}
	e.exp.SupresionesDeEvidencia = append(e.exp.SupresionesDeEvidencia,
		expediente.SupresionDeEvidencia{Entrada: indice, Prueba: prueba})
}

func exportar(t *testing.T, exp *expediente.Expediente) (string, Resumen) {
	t.Helper()
	var b bytes.Buffer
	res, err := Exportar(&b, exp)
	if err != nil {
		t.Fatalf("exportar: %v", err)
	}
	return b.String(), res
}

func lineas(t *testing.T, s string) []map[string]any {
	t.Helper()
	var out []map[string]any
	for i, l := range strings.Split(s, "\n") {
		if l == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("la linea %d no es un objeto JSON completo (%v): %s", i+1, err, l)
		}
		out = append(out, m)
	}
	return out
}

func obsDePrueba(prueba, recurso string, satisfecho bool) estado.Observacion {
	return estado.Observacion{
		Prueba:      prueba,
		Recurso:     recurso,
		Satisfecho:  satisfecho,
		Recolectada: comoEstaba().Add(-2 * time.Hour),
		Recolector:  "recolector-de-prueba",
		Version:     "1.0.0",
		HashCarga:   "sha256:deadbeef",
	}
}

// --- forma del fichero ------------------------------------------------------

func TestCadaLineaEsUnEventoJSONCompletoYSinEnvoltura(t *testing.T) {
	e := nuevoEscenario(t, carga(t, obsDePrueba("control.uno", "recurso-a", false)))
	salida, res := exportar(t, e.exp)

	if !strings.HasPrefix(salida, "{") {
		t.Fatalf("el fichero no empieza por un objeto: un array envuelto obliga al "+
			"receptor a leerlo entero antes de la primera linea. Empieza por %q",
			salida[:min(20, len(salida))])
	}
	if !strings.HasSuffix(salida, "\n") {
		t.Error("el fichero no acaba en salto de linea: la ultima linea se queda sin cerrar " +
			"y el receptor la descarta o la pega a la siguiente ingesta")
	}
	if strings.Contains(salida, "\r") {
		t.Error("hay retorno de carro en la salida. Con CRLF, un receptor que parte por " +
			"salto de linea deja un \\r pegado al final de cada objeto")
	}
	ms := lineas(t, salida)
	if len(ms) != res.Eventos {
		t.Fatalf("el resumen dice %d eventos y el fichero trae %d lineas", res.Eventos, len(ms))
	}
	if len(ms) < 5 {
		t.Fatalf("el escenario tiene 1 entrada, 1 checkpoint, 2 controles y 2 plazos, "+
			"o sea 6 eventos, y han salido %d. Si esto adelgaza, el resto de los tests "+
			"de este fichero comprueban el vacio", len(ms))
	}
	for i, m := range ms {
		for _, campo := range []string{"@timestamp", "event.action", "event.sequence",
			"event.id", "event.outcome", "message", "organization.name", "plazum.esquema"} {
			if _, hay := m[campo]; !hay {
				t.Errorf("la linea %d no trae %q, que es de los que todo evento lleva: %v",
					i+1, campo, m)
			}
		}
	}
}

func TestLaSecuenciaEsContiguaYEmpiezaEnUno(t *testing.T) {
	e := nuevoEscenario(t,
		carga(t, obsDePrueba("control.uno", "recurso-a", false)),
		carga(t, obsDePrueba("control.dos", "recurso-b", true)))
	salida, _ := exportar(t, e.exp)
	for i, m := range lineas(t, salida) {
		n, ok := m["event.sequence"].(float64)
		if !ok {
			t.Fatalf("la linea %d no trae numero de secuencia: sin el, el receptor no "+
				"puede detectar un hueco", i+1)
		}
		if int(n) != i+1 {
			t.Fatalf("la linea %d dice secuencia %v: la numeracion tiene que ser 1..N "+
				"sin huecos ni repeticiones", i+1, n)
		}
	}
}

func TestDosEjecucionesDanElMismoFicheroByteAByte(t *testing.T) {
	e := nuevoEscenario(t,
		carga(t, obsDePrueba("control.uno", "recurso-a", false)),
		carga(t, obsDePrueba("control.dos", "recurso-b", true)),
		carga(t, map[string]any{"tipo": "decision", "actor": "operador", "evidencia": "sha256:abc"}))
	e.suprime(t, 0, "urn:demo:ley art. 17", "2026-09-18T07:30:00Z", "control.uno")

	uno, _ := exportar(t, e.exp)
	dos, _ := exportar(t, e.exp)
	if uno != dos {
		t.Fatalf("dos exportaciones del mismo expediente difieren. Sin esto el receptor "+
			"no puede deduplicar ni detectar huecos.\n--- 1 ---\n%s\n--- 2 ---\n%s", uno, dos)
	}
}

// El fichero no puede depender del orden en que el emisor escribio sus listas.
// Si dependiera, dos expedientes con el mismo contenido y distinto orden darian
// ficheros distintos, y el receptor veria eventos nuevos donde no los hay.
func TestElOrdenNoDependeDelOrdenEnQueVieneElExpediente(t *testing.T) {
	e := nuevoEscenario(t,
		carga(t, obsDePrueba("control.uno", "recurso-a", false)),
		carga(t, obsDePrueba("control.dos", "recurso-b", true)))
	e.suprime(t, 0, "urn:demo:ley art. 17", "2026-09-18T07:30:00Z", "control.uno")
	esperado, _ := exportar(t, e.exp)

	invertir := func(exp *expediente.Expediente) {
		for i, j := 0, len(exp.Cadena.Entradas)-1; i < j; i, j = i+1, j-1 {
			exp.Cadena.Entradas[i], exp.Cadena.Entradas[j] = exp.Cadena.Entradas[j], exp.Cadena.Entradas[i]
		}
		for i, j := 0, len(exp.Estados)-1; i < j; i, j = i+1, j-1 {
			exp.Estados[i], exp.Estados[j] = exp.Estados[j], exp.Estados[i]
		}
		for i, j := 0, len(exp.Reclamaciones)-1; i < j; i, j = i+1, j-1 {
			exp.Reclamaciones[i], exp.Reclamaciones[j] = exp.Reclamaciones[j], exp.Reclamaciones[i]
		}
	}
	invertir(e.exp)
	obtenido, _ := exportar(t, e.exp)
	if obtenido != esperado {
		t.Fatalf("invertir las listas del expediente cambia el fichero.\n--- antes ---\n%s\n"+
			"--- despues ---\n%s", esperado, obtenido)
	}
}

// --- el instante, que aqui es dato y no reloj -------------------------------

func TestElEventoDiceDeDondeSaleSuInstante(t *testing.T) {
	e := nuevoEscenario(t,
		carga(t, obsDePrueba("control.uno", "recurso-a", false)),
		carga(t, map[string]any{"tipo": "decision"}))
	salida, _ := exportar(t, e.exp)

	visto := map[string]string{}
	for _, m := range lineas(t, salida) {
		if m["event.action"] != AccionEntrada {
			continue
		}
		visto[m["plazum.instante_es"].(string)] = m["@timestamp"].(string)
	}
	if _, hay := visto[InstanteObservacion]; !hay {
		t.Errorf("una entrada con observacion fechada tiene que llevar el instante de la "+
			"observacion, y no la cota del checkpoint: %v", visto)
	}
	if _, hay := visto[InstanteCota]; !hay {
		t.Errorf("una entrada sin fecha dentro tiene que llevar la COTA del checkpoint, "+
			"dicha como cota. La cadena no fecha sus entradas y escribir la cota como si "+
			"fuera el momento del hecho seria inventar precision: %v", visto)
	}
}

func TestUnPlazoYaVencidoSaleComoFalloYUnoFuturoNo(t *testing.T) {
	e := nuevoEscenario(t, carga(t, obsDePrueba("control.uno", "recurso-a", true)))
	e.exp.Reclamaciones = []expediente.Reclamacion{
		{Obligacion: "urn:demo:obligacion:pasada", Hito: "aviso", Estado: "determinado",
			Vence: comoEstaba().Add(-time.Hour)},
		{Obligacion: "urn:demo:obligacion:futura", Hito: "aviso", Estado: "determinado",
			Vence: comoEstaba().Add(time.Hour)},
	}
	salida, res := exportar(t, e.exp)
	if res.Vencidos != 1 {
		t.Fatalf("el resumen cuenta %d plazos vencidos y tenia que contar 1", res.Vencidos)
	}
	for _, m := range lineas(t, salida) {
		if m["event.action"] != AccionVencimiento {
			continue
		}
		pasada := strings.Contains(m["plazum.obligacion"].(string), "pasada")
		if pasada && m["event.outcome"] != Fallo {
			t.Errorf("un plazo anterior a la fecha del expediente tiene que salir como %q "+
				"para que el SIEM pueda alertar: %v", Fallo, m)
		}
		// CONTROL NEGATIVO: si el futuro tambien saliera como fallo, la alerta
		// del receptor se dispararia con todo y dejaria de significar nada.
		if !pasada && m["event.outcome"] == Fallo {
			t.Errorf("un plazo que aun no ha vencido sale como fallo: %v", m)
		}
	}
}

// --- el vocabulario de estados ----------------------------------------------

// Todo estado del dominio tiene desenlace. Si alguien anade el noveno estado en
// nucleo/estado, este test se pone rojo en vez de dejar que ese estado salga al
// SIEM como "unknown" sin que nadie lo decida.
func TestTodoEstadoDelDominioTieneDesenlace(t *testing.T) {
	n := cuantosEstadosHay(t)
	if n != len(resultadoPorEstado) {
		t.Fatalf("nucleo/estado tiene %d estados y la tabla de desenlaces cubre %d. "+
			"El que falta saldria al SIEM como desconocido sin que nadie lo haya decidido. "+
			"Arreglo: anadirlo a resultadoPorEstado con su desenlace", n, len(resultadoPorEstado))
	}
	for i := 0; i < n; i++ {
		e := estado.Estado(i) // #nosec G115 -- i < n, acotado por cuantosEstadosHay
		if _, hay := resultadoPorEstado[e]; !hay {
			t.Errorf("el estado %q no tiene desenlace en la tabla", e.String())
		}
		if estadoPorNombre[e.String()] != e {
			t.Errorf("el estado %q no se resuelve de vuelta desde su nombre", e.String())
		}
	}
}

// cuantosEstadosHay cuenta los estados de nucleo/estado sin escribir aqui la
// lista: String() indexa un array, asi que el primero que no existe entra en
// panico y ese es el borde. Contarlo asi es lo que hace que el test de arriba
// vea aparecer un estado nuevo; con la lista escrita al lado, la mutacion la
// elegiria el propio test y no demostraria nada.
func cuantosEstadosHay(t *testing.T) int {
	t.Helper()
	n := 0
	for i := 0; i < 64; i++ {
		if !existeEstado(estado.Estado(i)) { // #nosec G115 -- i < 64
			return n
		}
		n++
	}
	t.Fatal("nucleo/estado dice tener mas de 64 estados: el contador ha dejado de contar")
	return 0
}

func existeEstado(e estado.Estado) (existe bool) {
	defer func() {
		if recover() != nil {
			existe = false
		}
	}()
	_ = e.String()
	return true
}

func TestElContadorDeEstadosSabeDecirQueNo(t *testing.T) {
	// CONTROL NEGATIVO del contador: un valor que no existe tiene que contestar
	// que no. Sin esto, un contador que devolviera siempre true haria que el
	// test de arriba comparase 64 contra 8 y fallara por el motivo equivocado, o
	// peor, que devolviera siempre false y contase 0 contra 0.
	if existeEstado(estado.Estado(200)) {
		t.Fatal("el detector dice que existe el estado 200. Mientras conteste eso, el " +
			"recuento de estados no vigila nada")
	}
	if !existeEstado(estado.Pass) {
		t.Fatal("el detector dice que no existe un estado que si existe: con falsos " +
			"negativos contaria cero estados y la tabla de desenlaces pareceria completa")
	}
}

// --- higiene de los valores -------------------------------------------------

func TestUnValorLargoSeRecortaSinPartirElUTF8(t *testing.T) {
	largo := strings.Repeat("ñ", MaxValor) // 2 bytes por runa: el corte cae dentro de una
	e := nuevoEscenario(t, carga(t, map[string]any{"sujeto": largo}))
	salida, _ := exportar(t, e.exp)
	for _, m := range lineas(t, salida) {
		s, hay := m["plazum.sujeto"].(string)
		if !hay {
			continue
		}
		if len(s) > MaxValor+len(" [recortado]") {
			t.Errorf("el valor recortado ocupa %d bytes y el tope es %d", len(s), MaxValor)
		}
		if !strings.HasSuffix(s, "[recortado]") {
			t.Error("un valor recortado tiene que decir que lo esta: si no, el analista " +
				"lee media cadena creyendo que es entera")
		}
		if !utf8.ValidString(s) {
			t.Error("el recorte ha partido el UTF-8 por la mitad")
		}
	}
}

func TestLosCaracteresDeControlNoLleganAlFichero(t *testing.T) {
	e := nuevoEscenario(t, carga(t, map[string]any{
		"sujeto": "antes\x1b[31mrojo\x1b[0m y \x00nulo",
	}))
	salida, _ := exportar(t, e.exp)
	if strings.Contains(salida, "\\u001b") || strings.Contains(salida, "\\u0000") {
		t.Errorf("hay caracteres de control en la salida. Una secuencia de escape dentro "+
			"de un campo pinta lo que quiera en la terminal del que lee los logs:\n%s", salida)
	}
	if !strings.Contains(salida, "antes[31mrojo") {
		t.Errorf("se ha limpiado de mas o de menos: esperaba el texto sin los escapes.\n%s", salida)
	}
}

func TestSinExpedienteElErrorDiceComoSeArregla(t *testing.T) {
	_, _, err := Eventos(nil)
	if err == nil {
		t.Fatal("exportar el vacio tiene que fallar, no dar un fichero vacio que parezca " +
			"un expediente sin nada que auditar")
	}
	if !strings.Contains(err.Error(), "expediente.Cargar") {
		t.Errorf("el error no dice como se arregla: %v", err)
	}
}
