package ledger

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"
)

func clave(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	// Semilla fija: el test tiene que ser reproducible byte a byte.
	semilla := make([]byte, ed25519.SeedSize)
	copy(semilla, []byte("obligo-test-semilla-determinista"))
	return ed25519.NewKeyFromSeed(semilla)
}

func ledgerDePrueba(t *testing.T) (*Ledger, Checkpoint) {
	t.Helper()
	l := &Ledger{}
	base := time.Date(2026, 8, 23, 9, 0, 0, 0, time.UTC)
	for i := 0; i < 7; i++ {
		carga, _ := json.Marshal(map[string]any{"prueba": "mfa.todos", "satisfecho": i%3 != 0})
		if _, err := l.Anadir(Entrada{
			Instante: base.Add(time.Duration(i) * time.Minute), Tipo: "observacion",
			Sujeto: "sede-electronica", Paquete: "ens@2022.311",
			PaqueteHash: "sha256:1f3a", Carga: carga, Actor: "conector:entra-id",
		}); err != nil {
			t.Fatal(err)
		}
	}
	k := clave(t)
	l.ClavesConfiables = []string{hex.EncodeToString(k.Public().(ed25519.PublicKey))}
	c := l.Cerrar(k, base.Add(time.Hour), "tsa:rfc3161://tsa.obligo.example")
	return l, c
}

func TestCadenaYFirmaVerifican(t *testing.T) {
	l, _ := ledgerDePrueba(t)
	if err := l.Verificar(); err != nil {
		t.Fatalf("un ledger recien creado tiene que verificar: %v", err)
	}
}

func TestAlterarUnaEntradaRompeLaCadena(t *testing.T) {
	l, _ := ledgerDePrueba(t)
	// La entrada 4 (i=3) es la primera con satisfecho=false tras la inicial.
	var carga map[string]any
	_ = json.Unmarshal(l.Entradas[3].Carga, &carga)
	if carga["satisfecho"] == true {
		t.Fatal("el caso de prueba tiene que partir de un fallo real, si no no altera nada")
	}
	carga["satisfecho"] = true // el clasico: convertir un fallo en un cumplimiento
	l.Entradas[3].Carga, _ = json.Marshal(carga)
	if err := l.Verificar(); err == nil {
		t.Fatal("editar una entrada intermedia tiene que detectarse")
	} else {
		t.Logf("detectado: %v", err)
	}
}

func TestBorrarUnaEntradaRompeLaCadena(t *testing.T) {
	l, _ := ledgerDePrueba(t)
	l.Entradas = append(l.Entradas[:3], l.Entradas[4:]...)
	if err := l.Verificar(); err == nil {
		t.Fatal("borrar una entrada tiene que detectarse")
	}
}

func TestRehacerLaCadenaEnteraNoEnganaAlCheckpoint(t *testing.T) {
	// Este es el ataque realista: el operador controla el binario y la base,
	// asi que puede recalcular TODA la cadena. Lo que no puede es reproducir
	// una raiz ya publicada y anclada fuera.
	_, c := ledgerDePrueba(t)
	raizPublicada := c.RaizMerkle

	l2 := &Ledger{}
	k := clave(t)
	l2.ClavesConfiables = []string{hex.EncodeToString(k.Public().(ed25519.PublicKey))}
	base := time.Date(2026, 8, 23, 9, 0, 0, 0, time.UTC)
	for i := 0; i < 7; i++ {
		carga, _ := json.Marshal(map[string]any{"prueba": "mfa.todos", "satisfecho": true})
		if _, err := l2.Anadir(Entrada{Instante: base.Add(time.Duration(i) * time.Minute), Tipo: "observacion",
			Sujeto: "sede-electronica", Paquete: "ens@2022.311", PaqueteHash: "sha256:1f3a",
			Carga: carga, Actor: "conector:entra-id"}); err != nil {
			t.Fatal(err)
		}
	}
	c2 := l2.Cerrar(k, base.Add(time.Hour), "tsa:rfc3161://tsa.obligo.example")
	if err := l2.Verificar(); err != nil {
		t.Fatalf("la cadena rehecha es internamente coherente, y eso es justo el problema: %v", err)
	}
	if c2.RaizMerkle == raizPublicada {
		t.Fatal("dos historias distintas no pueden dar la misma raiz")
	}
	t.Log("la cadena rehecha verifica sola; lo que la delata es la raiz anclada fuera. " +
		"Por eso el checkpoint sin anclaje se rechaza.")
}

func TestCheckpointSinAnclajeSeRechaza(t *testing.T) {
	l := &Ledger{}
	k := clave(t)
	l.ClavesConfiables = []string{hex.EncodeToString(k.Public().(ed25519.PublicKey))}
	if _, err := l.Anadir(Entrada{Tipo: "observacion", Sujeto: "x"}); err != nil {
		t.Fatal(err)
	}
	l.Cerrar(k, time.Date(2026, 8, 23, 9, 0, 0, 0, time.UTC), "")
	if err := l.Verificar(); err == nil {
		t.Fatal("un checkpoint sin anclaje externo no puede darse por bueno")
	}
}

func TestPruebaDeInclusionSinLaBase(t *testing.T) {
	l, c := ledgerDePrueba(t)
	ruta, err := l.PruebaInclusion(3, c)
	if err != nil {
		t.Fatal(err)
	}
	hoja := l.Entradas[2].HashCadena
	if !ComprobarInclusion(hoja, ruta, c.RaizMerkle) {
		t.Fatal("el auditor tiene que poder comprobar su entrada con la ruta y la raiz, sin la base")
	}
	if ComprobarInclusion("0000", ruta, c.RaizMerkle) {
		t.Fatal("una hoja que no esta no puede verificar")
	}
	t.Logf("prueba de inclusion de %d hashes para un ledger de %d entradas", len(ruta), len(l.Entradas))
}

// --- Tests anadidos tras la revision adversarial ---

// Una carga que no es JSON valido hacia que json.Marshal fallara, el error se
// descartaba y la entrada hasheaba a sha256(""): dos entradas distintas con el
// mismo hash. Falsificacion directa.
func TestCargaInvalidaSeRechazaEnVezDeHashearAVacio(t *testing.T) {
	l := &Ledger{}
	if _, err := l.Anadir(Entrada{Tipo: "observacion", Sujeto: "x",
		Carga: json.RawMessage("{esto no es json")}); err == nil {
		t.Fatal("una carga no serializable tiene que rechazarse al anadir, no hashear a vacio")
	}
	if len(l.Entradas) != 0 {
		t.Fatal("una entrada rechazada no puede quedar en el ledger")
	}
}

// Reindentar el fichero no puede romper la cadena: el hash es sobre JSON canonico.
func TestReindentarNoRompeLaCadena(t *testing.T) {
	l, _ := ledgerDePrueba(t)
	b, err := json.MarshalIndent(l, "", "    ")
	if err != nil {
		t.Fatal(err)
	}
	var otro Ledger
	if err := json.Unmarshal(b, &otro); err != nil {
		t.Fatal(err)
	}
	if err := otro.Verificar(); err != nil {
		t.Fatalf("el fallo de un proxy que normaliza JSON no puede ser indistinguible de un ataque: %v", err)
	}
}

// Rehacer la historia y firmarla con una clave nueva ya no cuela.
func TestFirmaConClaveNoConfiableSeRechaza(t *testing.T) {
	l, _ := ledgerDePrueba(t)
	otraSemilla := make([]byte, ed25519.SeedSize)
	copy(otraSemilla, []byte("clave-del-atacante--------------"))
	otra := ed25519.NewKeyFromSeed(otraSemilla)
	l.Checkpoints = nil
	l.Cerrar(otra, time.Date(2026, 8, 23, 10, 0, 0, 0, time.UTC), "tsa:rfc3161://cualquiera")
	if err := l.Verificar(); err == nil {
		t.Fatal("una firma solo vale contra una clave que el receptor ya conocia")
	} else {
		t.Logf("rechazado: %v", err)
	}
}

// Separacion de dominio: un nodo interno no puede presentarse como hoja.
func TestSeparacionDeDominioMerkle(t *testing.T) {
	if hashHoja("aa") == hashInterno("aa", "") {
		t.Fatal("hoja e interno tienen que hashear distinto (RFC 6962)")
	}
}
