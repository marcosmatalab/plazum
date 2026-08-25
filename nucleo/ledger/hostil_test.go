package ledger

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
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
	// Tiene que quejarse de que el sello NO VERIFICA, que es lo que prueba el
	// ataque. Afirmar "sello" a secas valia igual para ErrSinVerificadorDeSello
	// ("no hay con que comprobar el sello"), que es un fallo de configuracion
	// del receptor y no la deteccion del anclaje inventado.
	if !errors.Is(err, ErrSelloNoVerifica) {
		t.Fatalf("tiene que quejarse de que el sello no verifica, y dijo: %v", err)
	}
	if errors.Is(err, ErrSinVerificadorDeSello) {
		t.Fatalf("el receptor SI aporto verificador de sello: esto no prueba el ataque: %v", err)
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

	// Tiene que pararlo la comprobacion EXPLICITA del hash de la entrada. Si se
	// aceptara cualquier error, quitar esa comprobacion dejaria el test en
	// verde: la firma tambien falla aqui (la lapida se firmo sobre otra cadena)
	// y el test no distinguiria una capa de la otra. El respaldo por firma
	// tiene su propio test mas abajo.
	_, err = b.Verificar(cf)
	if !errors.Is(err, ErrLapidaDeOtraEntrada) {
		t.Fatalf("una lapida firmada sobre otra entrada no puede darse por buena aqui, "+
			"y tiene que pararla la comprobacion del hash de entrada: %v", err)
	}
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

	// Por identidad: la lapida esta bien firmada y trae base legal, asi que el
	// unico rechazo legitimo es el de rango. Cualquier otro seria casualidad.
	if _, err := c.Verificar(confianzaDe(t, k)); !errors.Is(err, ErrLapidaFueraDeRango) {
		t.Fatalf("no se puede suprimir una entrada que no existe: %v", err)
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
	// La copia es identica a la buena: hash de entrada, cadena y firma cuadran.
	// El unico motivo por el que puede caer es el recuento duplicado.
	if _, err := c.Verificar(confianzaDe(t, k)); !errors.Is(err, ErrLapidaDuplicada) {
		t.Fatalf("informar dos veces la misma supresion falsea el recuento: %v", err)
	}
	// Y Borrar tampoco deja hacerlo dos veces.
	if _, err := c.Borrar(ks, k, 1, "RGPD art. 17", instHostil); !errors.Is(err, ErrYaSuprimida) {
		t.Fatalf("no se puede suprimir dos veces la misma entrada: %v", err)
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
	if !errors.Is(err, ErrClaveNoReconocida) {
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
	// Con Confianza{} falta todo, y lo primero que se echa en falta son las
	// claves. Pinarlo evita que el test se conforme con, por ejemplo, un fallo
	// de la cadena que no tiene nada que ver con la confianza del receptor.
	if err := l.Verificar(Confianza{}); !errors.Is(err, ErrSinClavesConfiables) {
		t.Fatalf("sin claves ni verificador de sello no hay nada que dar por bueno: %v", err)
	}
}

// La firma de la lapida cubre el hash de la entrada Y la raiz de la cadena, no
// solo el indice. Es lo que impide FABRICAR una lapida a medida del destino.
//
// CORRECCION (barrido de aserciones flojas): aqui ponia que la firma es "el
// respaldo" de la comprobacion explicita de HashEntrada y que si alguien
// quitara esa comprobacion la firma seguiria impidiendo el transplante por si
// sola. Es falso, y la mutacion lo demuestra: al retirar la comprobacion
// explicita, TestHostilLapidaYaNoSeTransplantaAOtraCadena pasa a devolver nil,
// porque la lapida transplantada tal cual conserva SUS HashEntrada y Cadena y
// la firma verifica sobre ellos. La firma no sabe en que cadena esta metida la
// lapida; solo la comprobacion explicita lo mira. Las dos capas son
// necesarias y no se suplen: por eso cada test pina la suya por identidad.
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
	// Por ErrFirmaLapida y no por la subcadena "firma invalida": ese mismo
	// texto lo dice ErrFirmaCheckpoint, y ademas ErrLapidaDeOtraEntrada es la
	// otra capa que este test necesita descartar expresamente.
	if !errors.Is(err, ErrFirmaLapida) {
		t.Fatalf("tiene que fallar por la FIRMA, no por otra comprobacion, y dijo: %v", err)
	}
	if errors.Is(err, ErrLapidaDeOtraEntrada) {
		t.Fatalf("lo paro la comprobacion explicita, no la firma: el parcheo del atacante "+
			"tendria que haberla esquivado: %v", err)
	}
}
