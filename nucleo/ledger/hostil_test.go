package ledger

import (
	"crypto/ed25519"
	"encoding/hex"
	"testing"
	"time"
)

// Revision hostil de la etapa 1. Cada test de aqui INTENTA romper una
// propiedad declarada. Si pasa, la propiedad aguanta; si falla, es hallazgo.

const instHostil = "2026-08-25T09:00:00Z"

// ATAQUE 1. El checkpoint dice exigir "anclaje externo", y el error promete que
// sin el "la cadena solo prueba coherencia interna". Se comprueba si el anclaje
// es algo mas que una cadena no vacia.
func TestHostilAnclajeInventadoSeCuela(t *testing.T) {
	l := &Ledger{}
	k := clave(t)
	l.ClavesConfiables = []string{hex.EncodeToString(k.Public().(ed25519.PublicKey))}
	if _, err := l.Anadir(Entrada{Tipo: "observacion", Sujeto: "x"}); err != nil {
		t.Fatal(err)
	}
	l.Cerrar(k, time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC), "me lo acabo de inventar")
	if err := l.Verificar(); err != nil {
		t.Logf("PROPIEDAD AGUANTA: %v", err)
		return
	}
	t.Fatal("HALLAZGO: un anclaje que es texto libre inventado verifica igual que un sello RFC 3161 " +
		"real. La comprobacion es c.Anclaje != \"\" y nada mas: no hay token, no se parsea y no se " +
		"verifica contra ninguna TSA. El adaptador que sabe hacerlo (adaptadores/tsa) no lo usa nadie")
}

// ATAQUE 2. La lapida se firma sobre indice + base legal + instante. No entra
// ni el hash de la entrada ni identidad de cadena, asi que una lapida legitima
// deberia poder transplantarse a OTRA cadena y suprimir alli el mismo indice.
func TestHostilLapidaSeTransplantaAOtraCadena(t *testing.T) {
	k := clave(t)
	pub := k.Public().(ed25519.PublicKey)

	// Cadena A: supresion legitima de la entrada 1.
	a, ksA := cadenaDePrueba(t)
	lap, err := a.Borrar(ksA, k, 1, "RGPD art. 17", instHostil)
	if err != nil {
		t.Fatal(err)
	}

	// Cadena B: otra cadena distinta, con su entrada 1 viva e incomoda.
	b, _ := cadenaDePrueba(t)
	b.Lapidas = append(b.Lapidas, lap) // se pega tal cual, sin tocar nada mas

	inf, err := b.Verificar(pub)
	if err != nil {
		t.Logf("PROPIEDAD AGUANTA: %v", err)
		return
	}
	t.Fatalf("HALLAZGO: la lapida de otra cadena verifica aqui y el informe la da por buena: %q. "+
		"contenidoFirmado() es indice||base legal||instante, no ata la lapida ni al hash de la "+
		"entrada ni a la cadena, asi que una supresion legitima se recicla para tapar otra cosa",
		inf.Suprimidas)
}

// ATAQUE 3. Una lapida para un indice que no existe en la cadena.
func TestHostilLapidaDeEntradaInexistente(t *testing.T) {
	k := clave(t)
	pub := k.Public().(ed25519.PublicKey)
	c, _ := cadenaDePrueba(t) // 4 entradas: 0..3

	falsa := Lapida{EntradaBorrada: 999, BaseLegal: "RGPD art. 17", Instante: instHostil}
	falsa.Firma = ed25519.Sign(k, falsa.contenidoFirmado())
	c.Lapidas = append(c.Lapidas, falsa)

	inf, err := c.Verificar(pub)
	if err != nil {
		t.Logf("PROPIEDAD AGUANTA: %v", err)
		return
	}
	t.Fatalf("HALLAZGO: se acepta una lapida de la entrada 999 en una cadena de 4, y el informe la "+
		"lista como supresion real: %q", inf.Suprimidas)
}

// ATAQUE 4. La clave publica la aporta el RECEPTOR. Si carga una malformada,
// ed25519.Verify hace panic en vez de devolver error.
func TestHostilClavePublicaMalformadaRevientaElVerificador(t *testing.T) {
	k := clave(t)
	c, ks := cadenaDePrueba(t)
	if _, err := c.Borrar(ks, k, 0, "RGPD art. 17", instHostil); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("HALLAZGO: una clave publica de tamano equivocado hace panic en vez de dar error "+
				"(%v). La clave la aporta el receptor, asi que un fichero de anclas mal copiado tumba "+
				"el verificador en vez de decir que la clave no vale", r)
		}
	}()
	if _, err := c.Verificar([]byte("clave corta")); err != nil {
		t.Logf("PROPIEDAD AGUANTA: %v", err)
	}
}

// ATAQUE 5. Dos lapidas para el mismo indice: la supresion se cuenta dos veces.
func TestHostilLapidasDuplicadas(t *testing.T) {
	k := clave(t)
	pub := k.Public().(ed25519.PublicKey)
	c, ks := cadenaDePrueba(t)
	lap, err := c.Borrar(ks, k, 1, "RGPD art. 17", instHostil)
	if err != nil {
		t.Fatal(err)
	}
	c.Lapidas = append(c.Lapidas, lap) // la misma, repetida

	inf, err := c.Verificar(pub)
	if err != nil {
		t.Logf("PROPIEDAD AGUANTA: %v", err)
		return
	}
	if len(inf.Suprimidas) > 1 {
		t.Fatalf("HALLAZGO: la misma supresion se informa %d veces (%q). Un informe que dice "+
			"'2 entradas suprimidas' cuando fue una es un dato falso en un documento probatorio",
			len(inf.Suprimidas), inf.Suprimidas)
	}
}

// ATAQUE 7 (lo encontro Marcos, no yo). ClavesConfiables tiene la misma
// enfermedad que AnclasDeConfianza: su comentario dice que son "las claves
// publicas que el receptor acepta" y que "una firma solo vale contra una clave
// que el receptor ya conocia", pero lleva etiqueta json y viaja dentro del
// fichero del emisor. O sea que el emisor se escribe la clave contra la que se
// comprueba su propia firma, y la guarda de len(ClavesConfiables)==0 no sirve
// de nada: se pone una y ya esta.
func TestHostilElEmisorSeEscribeSusPropiasClavesConfiables(t *testing.T) {
	// Una clave que el receptor no ha visto en su vida.
	semilla := make([]byte, ed25519.SeedSize)
	copy(semilla, []byte("clave que el receptor no conoce"))
	suya := ed25519.NewKeyFromSeed(semilla)

	l := &Ledger{}
	if _, err := l.Anadir(Entrada{Tipo: "observacion", Sujeto: "sede"}); err != nil {
		t.Fatal(err)
	}
	// El emisor firma con SU clave y se declara a si mismo como confiable.
	l.ClavesConfiables = []string{hex.EncodeToString(suya.Public().(ed25519.PublicKey))}
	l.Cerrar(suya, time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC), "tsa:lo-que-sea")

	if err := l.Verificar(); err != nil {
		t.Logf("PROPIEDAD AGUANTA: %v", err)
		return
	}
	t.Fatal("HALLAZGO: el emisor ha firmado con una clave que nadie le ha reconocido, se ha " +
		"declarado a si mismo confiable en el propio fichero, y verifica limpio. " +
		"Ledger.Verificar() no recibe ningun parametro: toda la confianza sale del fichero " +
		"que aporta el emisor. Es la misma clase de bug que AnclasDeConfianza")
}
