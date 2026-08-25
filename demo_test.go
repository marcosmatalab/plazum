package dutiq_test

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"dutiq/adaptadores/tsa"
	"dutiq/nucleo/expediente"
)

// La demo del producto tiene que verificar con el verificador del producto.
//
// Por que este test existe. Lo primero que hace cualquiera es `dutiq verify`
// sobre el expediente demo. Durante un tiempo eso fallaba, porque el demo
// llevaba un sello de relleno y el verificador, con razon, se negaba a dar por
// bueno un anclaje que no podia comprobar. La tentacion era meter un atajo en
// el verificador; la solucion correcta fue sellar el demo DE VERDAD una vez
// (herramientas/sellardemo) y empotrar en el binario las raices de las TSAs.
//
// Este test es lo que impide que eso se rompa en silencio. Y es hermetico: el
// token esta commiteado y las raices van embebidas, asi que no toca la red.
//
// Que hace falta volver a sellar y que no, dicho con precision porque la
// primera version de este comentario decia de mas: la raiz Merkle cubre LAS
// ENTRADAS DE LA CADENA, no el expediente entero. Cambiar la organizacion, un
// reloj o un paquete no obliga a resellar; cambiar las observaciones que se
// encadenan, si. Comprobado por mutacion: tocar un byte de una entrada pone
// este test en rojo, tocar el nombre de la organizacion no.
func TestLaDemoVerificaConElVerificadorDeVerdad(t *testing.T) {
	b, err := os.ReadFile("expediente-demo.json")
	if err != nil {
		t.Fatal(err)
	}
	e, err := expediente.Cargar(b)
	if err != nil {
		t.Fatal(err)
	}

	ctx, err := contextoDeLaDemo(t, e)
	if err != nil {
		t.Fatal(err)
	}
	inf := expediente.Verificar(e, ctx)
	if !inf.Valido {
		t.Fatalf("la demo tiene que verificar con el verificador real y las raices embebidas.\n"+
			"Si has cambiado las observaciones que se encadenan, su raiz Merkle ha cambiado y "+
			"hay que volver a sellarla:\n"+
			"  go run ./herramientas/sellardemo\n"+
			"  DUTIQ_ESCRIBIR_DEMO=1 go test ./nucleo/expediente -run TestGenerarDemo\n\n"+
			"discrepancias: %v", inf.Discrepancias)
	}

	// Y que el sello sea de verdad, no de relleno: que lo verifique el
	// adaptador contra las raices que trae el binario.
	if len(e.Cadena.Checkpoints) == 0 {
		t.Fatal("la demo tiene que traer al menos un checkpoint")
	}
	cp := e.Cadena.Checkpoints[0]
	if len(cp.Token) == 0 {
		t.Fatal("el checkpoint de la demo tiene que traer un sello RFC 3161 real")
	}
	inst, err := tsa.Instante(cp.Token)
	if err != nil {
		t.Fatalf("el sello de la demo no es un token RFC 3161 legible: %v", err)
	}
	t.Logf("la demo lleva un sello autentico del %s, verificado offline contra las raices embebidas",
		inst.Format("2006-01-02 15:04:05 MST"))
}

// contextoDeLaDemo monta lo que aportaria el receptor, usando el verificador
// real de sellos con las raices que trae el binario.
func contextoDeLaDemo(t *testing.T, e *expediente.Expediente) (expediente.ContextoReceptor, error) {
	t.Helper()
	var ctx expediente.ContextoReceptor

	b, err := os.ReadFile("contexto-demo.json")
	if err != nil {
		return ctx, err
	}
	var f struct {
		Anclas           map[string]string `json:"anclas"`
		ClavesConfiables []string          `json:"claves_confiables"`
		ClaveOperador    string            `json:"clave_operador"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return ctx, err
	}
	k, err := hex.DecodeString(f.ClaveOperador)
	if err != nil {
		return ctx, err
	}
	pool, err := tsa.RaicesPorDefecto()
	if err != nil {
		return ctx, err
	}
	cadena := &tsa.Cadena{Anclas: pool}
	return expediente.ContextoReceptor{
		Anclas:           f.Anclas,
		ClavesConfiables: f.ClavesConfiables,
		ClaveOperador:    k,
		VerificarSello:   cadena.VerificarOffline,
	}, nil
}
