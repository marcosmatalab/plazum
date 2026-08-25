package ledger

import (
	"crypto/ed25519"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

// Ataques de la revision hostil de la etapa 1. Se escribieron en rojo, como
// hallazgos, y se quedan como regresion: si alguno vuelve a pasar de largo, la
// propiedad se ha roto otra vez.

const instHostil = "2026-08-25T09:00:00Z"

// ATAQUE 1. El anclaje era c.Anclaje != "" y nada mas, asi que un texto libre
// inventado verificaba igual que un sello RFC 3161 real.
func TestHostilAnclajeInventadoYaNoSeCuela(t *testing.T) {
	l := &Ledger{}
	k := clave(t)
	if _, err := l.Anadir(Entrada{Tipo: "observacion", Sujeto: "x"}); err != nil {
		t.Fatal(err)
	}
	// Etiqueta bonita, sin token: lo que antes colaba.
	l.Cerrar(k, time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC), "me lo acabo de inventar", nil)
	err := l.Verificar(confianzaDe(t, k))
	if err == nil {
		t.Fatal("un anclaje sin sello que comprobar no puede verificar")
	}
	if !strings.Contains(err.Error(), "sello") {
		t.Fatalf("tiene que quejarse del sello, y dijo: %v", err)
	}
}

// Control negativo del anterior: con sello bueno, verifica.
func TestHostilConSelloBuenoElCheckpointVerifica(t *testing.T) {
	l, _, cf := ledgerDePrueba(t)
	if err := l.Verificar(cf); err != nil {
		t.Fatalf("el checkpoint con sello tiene que verificar: %v", err)
	}
}

// ATAQUE 2. La lapida se firmaba sobre indice, base legal e instante, sin hash
// de entrada ni identidad de cadena, asi que una supresion legitima se pegaba
// en otra cadena y suprimia alli lo que ocupara el mismo indice.
func TestHostilLapidaYaNoSeTransplantaAOtraCadena(t *testing.T) {
	k := clave(t)
	cf := confianzaDe(t, k)

	a, ksA := cadenaDePrueba(t)
	lap, err := a.Borrar(ksA, k, 1, "RGPD art. 17", instHostil)
	if err != nil {
		t.Fatal(err)
	}

	// Cadena B: otra cadena, con contenido distinto en el indice 1.
	b, ksB := &CadenaV2{}, NuevoKeystore()
	for i := byte(0); i < 4; i++ {
		if _, err := b.Anadir(ksB, claveFija(i+10), nonceFijo(i+10),
			[]byte{'o', 't', 'r', 'o', i}); err != nil {
			t.Fatal(err)
		}
	}
	b.Lapidas = append(b.Lapidas, lap)

	if _, err := b.Verificar(cf); err == nil {
		t.Fatal("una lapida firmada sobre otra entrada no puede darse por buena aqui")
	}
	_ = lap
}

// Control negativo: en su propia cadena, la lapida sigue valiendo.
func TestHostilLaLapidaSigueValiendoEnSuCadena(t *testing.T) {
	k := clave(t)
	c, ks := cadenaDePrueba(t)
	if _, err := c.Borrar(ks, k, 1, "RGPD art. 17", instHostil); err != nil {
		t.Fatal(err)
	}
	inf, err := c.Verificar(confianzaDe(t, k))
	if err != nil {
		t.Fatalf("la supresion legitima tiene que seguir verificando: %v", err)
	}
	if len(inf.Suprimidas) != 1 {
		t.Fatalf("tiene que informar exactamente una supresion, informo %d", len(inf.Suprimidas))
	}
}

// ATAQUE 3. Una lapida para un indice que no existe se daba por buena y salia
// en el informe como supresion real.
func TestHostilLapidaDeEntradaInexistenteSeRechaza(t *testing.T) {
	k := clave(t)
	c, _ := cadenaDePrueba(t) // 4 entradas: 0..3

	falsa := Lapida{EntradaBorrada: 999, BaseLegal: "RGPD art. 17", Instante: instHostil}
	falsa.Firma = ed25519.Sign(k, falsa.contenidoFirmado())
	c.Lapidas = append(c.Lapidas, falsa)

	if _, err := c.Verificar(confianzaDe(t, k)); err == nil {
		t.Fatal("no se puede suprimir una entrada que no existe")
	}
}

// ATAQUE 4. Una clave publica de tamano equivocado hacia panic en ed25519,
// y la clave la aporta el receptor: un fichero mal copiado tumbaba el verificador.
func TestHostilClavePublicaMalformadaDaErrorYNoPanico(t *testing.T) {
	k := clave(t)
	c, ks := cadenaDePrueba(t)
	if _, err := c.Borrar(ks, k, 0, "RGPD art. 17", instHostil); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("tiene que dar error, no reventar: %v", r)
		}
	}()
	cf := confianzaDe(t, k)
	cf.ClaveOperador = []byte("clave corta")
	if _, err := c.Verificar(cf); err == nil {
		t.Fatal("una clave de tamano equivocado no puede darse por buena")
	}
}

// ATAQUE 5. Dos lapidas del mismo indice inflaban el recuento de supresiones.
func TestHostilLapidasDuplicadasSeRechazan(t *testing.T) {
	k := clave(t)
	c, ks := cadenaDePrueba(t)
	lap, err := c.Borrar(ks, k, 1, "RGPD art. 17", instHostil)
	if err != nil {
		t.Fatal(err)
	}
	c.Lapidas = append(c.Lapidas, lap) // la misma, repetida
	if _, err := c.Verificar(confianzaDe(t, k)); err == nil {
		t.Fatal("informar dos veces la misma supresion falsea el recuento")
	}
	// Y Borrar tampoco deja hacerlo dos veces.
	if _, err := c.Borrar(ks, k, 1, "RGPD art. 17", instHostil); err == nil {
		t.Fatal("no se puede suprimir dos veces la misma entrada")
	}
}

// ATAQUE 7 (lo encontro Marcos). ClavesConfiables viajaba dentro del fichero
// del emisor, o sea que el emisor se escribia la clave contra la que se
// comprueba su propia firma. Ahora la confianza entra por parametro y lo que
// el fichero declara no decide nada.
func TestHostilLoQueElEmisorDeclaraNoDecideNada(t *testing.T) {
	semilla := make([]byte, ed25519.SeedSize)
	copy(semilla, []byte("clave que el receptor no conoce"))
	suya := ed25519.NewKeyFromSeed(semilla)

	l := &Ledger{}
	if _, err := l.Anadir(Entrada{Tipo: "observacion", Sujeto: "sede"}); err != nil {
		t.Fatal(err)
	}
	// El emisor firma con SU clave y se declara a si mismo confiable.
	l.ClavesDeclaradas = []string{hex.EncodeToString(suya.Public().(ed25519.PublicKey))}
	l.Cerrar(suya, time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC), "tsa:lo-que-sea", tokenDePrueba)

	// El receptor solo conoce OTRA clave.
	honrada := clave(t)
	err := l.Verificar(confianzaDe(t, honrada))
	if err == nil {
		t.Fatal("una firma con una clave que el receptor no reconoce no puede verificar, " +
			"por mucho que el fichero se declare a si mismo confiable")
	}
	if !strings.Contains(err.Error(), "no reconoce") {
		t.Fatalf("el error tiene que decir que el receptor no reconoce la clave, y dijo: %v", err)
	}
}

// Control negativo del anterior: si el receptor SI conoce la clave, verifica.
func TestHostilConLaClaveDelReceptorSiVerifica(t *testing.T) {
	k := clave(t)
	l := &Ledger{}
	if _, err := l.Anadir(Entrada{Tipo: "observacion", Sujeto: "sede"}); err != nil {
		t.Fatal(err)
	}
	l.Cerrar(k, time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC), "tsa:buena", tokenDePrueba)
	if err := l.Verificar(confianzaDe(t, k)); err != nil {
		t.Fatalf("con la clave que el receptor conoce tiene que verificar: %v", err)
	}
}

// Y el receptor que no aporta nada no obtiene un "verificado" gratis.
func TestHostilSinConfianzaDelReceptorNoVerifica(t *testing.T) {
	l, _, _ := ledgerDePrueba(t)
	if err := l.Verificar(Confianza{}); err == nil {
		t.Fatal("sin claves ni verificador de sello no hay nada que dar por bueno")
	}
}

// La firma de la lapida cubre el hash de la entrada Y la raiz de la cadena, no
// solo el indice. Es el respaldo de la comprobacion explicita de HashEntrada:
// si alguien quita esa comprobacion en un refactor, la firma tiene que seguir
// impidiendo el transplante por si sola.
//
// Aqui se falsifica lo que haria un atacante que YA hubiera esquivado la
// comprobacion explicita: coge una lapida legitima de otra cadena y le parchea
// HashEntrada y Cadena para que cuadren con el destino. La firma no cuadra,
// porque se hizo sobre los valores de origen.
//
// El test cubre las DOS piezas a la vez, y no hay forma limpia de aislarlas por
// separado: cada una tapa a la otra (parchear solo una deja la otra
// desajustada). Comprobado por mutacion: quitar cualquiera de las dos no rompe
// nada, quitar las dos hace que este test pase la firma y falle.
func TestHostilLaFirmaDeLaLapidaAtaEntradaYCadena(t *testing.T) {
	k := clave(t)
	cf := confianzaDe(t, k)

	a, ksA := cadenaDePrueba(t)
	lap, err := a.Borrar(ksA, k, 1, "RGPD art. 17", instHostil)
	if err != nil {
		t.Fatal(err)
	}

	b, ksB := &CadenaV2{}, NuevoKeystore()
	for i := byte(0); i < 4; i++ {
		if _, err := b.Anadir(ksB, claveFija(i+10), nonceFijo(i+10),
			[]byte{'o', 't', 'r', 'o', i}); err != nil {
			t.Fatal(err)
		}
	}

	// El atacante parchea justo lo que mira la comprobacion explicita.
	falsa := lap
	falsa.HashEntrada = append([]byte(nil), b.Entradas[1].Hash...)
	falsa.Cadena = raizMerkle(b.hashesDeEntradas())
	b.Lapidas = append(b.Lapidas, falsa)

	_, err = b.Verificar(cf)
	if err == nil {
		t.Fatal("la firma de la lapida tiene que atarla a su entrada y a su cadena: " +
			"parchear los campos que mira la comprobacion explicita no puede bastar")
	}
	if !strings.Contains(err.Error(), "firma invalida") {
		t.Fatalf("tiene que fallar por la FIRMA, no por otra comprobacion, y dijo: %v", err)
	}
}
