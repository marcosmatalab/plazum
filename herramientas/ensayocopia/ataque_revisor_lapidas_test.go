package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"plazum/nucleo/ledger"
)

// REVISION HOSTIL, segunda tanda. La propiedad atacada es la que da nombre a la
// casilla: "un borrado legal sigue borrado despues de restaurar".
//
// Primero se intento por lo obvio, quitar la lapida de la replica, y salio rojo.
// Pero salio rojo por ErrSinLapidas, que NO es una comprobacion de seguridad
// sino una comprobacion de que el escenario trae algo que mirar. Salta solo
// porque el escenario sembrado tiene EXACTAMENTE UNA lapida, asi que quitarla
// deja la lista vacia. Esa es la trampa de mutar dentro de la lista que el
// propio test eligio, en su version de fixture: la forma del escenario es la que
// hace que la comprobacion parezca que vigila.
//
// El control negativo `clave-resucitada` de CI pasa por lo mismo. Repone la
// clave y DEJA la lapida, asi que la entrada cae en la rama de suprimidas y se
// caza. Un atacante quita las dos cosas.

// Medido, con el escenario tal y como se publica (UNA lapida):
//
//	quitar la lapida                        -> rojo por ErrSinLapidas
//	quitar la lapida Y reponer la clave     -> rojo por ErrSinLapidas
//
// O sea que las dos veces lo que paro el ataque fue "esta instalacion no tiene
// ninguna supresion que comprobar", no una comprobacion del borrado.
//
// ESCALADA, y esta es la que tumba la propiedad: DOS lapidas, se quita UNA.
//
// Una instalacion real tiene muchas. Aqui se siembra una con DOS supresiones y se
// quita UNA sola, reponiendo su clave. La lista sigue sin estar vacia, asi que
// ErrSinLapidas no salta, y no queda nadie mas mirando: la lista de lapidas no la
// cubre ningun hash de la cadena (hashEntradaV2 solo hashea indice, previo,
// nonce, cifrado y compromiso) ni ningun checkpoint (hashesDeEntradas tampoco la
// toca).
func TestZZDosLapidasQuitarUnaYReponerSuClave(t *testing.T) {
	trabajo := t.TempDir()
	vivo := filepath.Join(trabajo, "vivo")
	replica := filepath.Join(trabajo, "replica")
	confianza := filepath.Join(trabajo, "confianza.json")

	esc, err := cargarEscenario()
	if err != nil {
		t.Fatal(err)
	}
	s, err := Sembrar(vivo, "prueba", esc)
	if err != nil {
		t.Fatal(err)
	}
	// El escenario ya trae DOS supresiones desde que se cerro este agujero, asi
	// que este test no tiene que fabricar la segunda a mano. Antes si: cuando se
	// escribio, escenario.json borraba una sola entrada y por eso quitar su
	// lapida hacia saltar ErrSinLapidas, que es un rojo por el motivo
	// equivocado. Esa es la mitad del hallazgo que se arreglo en el escenario.
	if len(esc.Borrados) < 2 {
		t.Fatalf("el escenario trae %d borrado(s); con menos de dos este ataque no mide "+
			"lo que dice medir", len(esc.Borrados))
	}

	// El acta se escribe DESPUES de la segunda supresion y la incluye: el
	// receptor estuvo delante de las dos, asi que su recuerdo tiene las dos. Si
	// se escribiera antes, el acta seria de una instalacion que ya no existe y
	// el ensayo compararia contra un recuerdo caduco, que es otra forma de no
	// comprobar nada.
	if err := EscribirConfianzaConActa(confianza, s.ClaveOperador, s.Acta); err != nil {
		t.Fatal(err)
	}

	if _, err := Copiar(vivo, replica, "2026-08-21T03:10:00Z"); err != nil {
		t.Fatal(err)
	}

	// Comprobacion previa: la instalacion SANA de dos lapidas sale verde. Sin
	// esto, un rojo del ataque podria venir de haber sembrado mal.
	sano := filepath.Join(trabajo, "sano")
	if err := Restaurar(replica, sano); err != nil {
		t.Fatalf("la instalacion de dos lapidas ni siquiera restaura: %v", err)
	}
	rs, err := Verificar(sano, confianza)
	if err != nil {
		t.Fatalf("la instalacion SANA de dos lapidas sale en rojo (%v); el ataque no mediria nada", err)
	}
	if len(rs.Supresiones) != 2 {
		t.Fatalf("la instalacion sembrada tiene %d supresiones y tenian que ser 2", len(rs.Supresiones))
	}

	// EL ATAQUE. Se quita la lapida de la entrada suprimida por el art. 17.1.b
	// y se repone su clave. Queda una lapida, asi que ErrSinLapidas no salta.
	bat, err := CargarBase(replica)
	if err != nil {
		t.Fatal(err)
	}
	kat, err := CargarKeystore(replica)
	if err != nil {
		t.Fatal(err)
	}
	victima := esc.Borrados[0].Entrada
	var quedan []ledger.Lapida
	for _, l := range bat.Cadena.Lapidas {
		if l.EntradaBorrada != victima {
			quedan = append(quedan, l)
		}
	}
	bat.Cadena.Lapidas = quedan
	kat.Entradas[fmt.Sprint(victima)] = hex.EncodeToString(derivar("prueba", "entrada", victima, 32))
	// Y de paso la evidencia que colgaba de ella.
	for _, ev := range bat.Evidencias {
		if ev.Entrada == victima {
			kat.Evidencias[ev.Hash] = hex.EncodeToString(derivar("prueba", "evidencia", 1, 32))
		}
	}
	if err := escribirJSON(filepath.Join(replica, NombreBase), bat); err != nil {
		t.Fatal(err)
	}
	if err := escribirJSON(filepath.Join(replica, NombreKeystore), kat); err != nil {
		t.Fatal(err)
	}
	if err := rehacerManifiesto(replica); err != nil {
		t.Fatal(err)
	}

	roto := filepath.Join(trabajo, "roto")
	if err := os.RemoveAll(vivo); err != nil {
		t.Fatal(err)
	}
	if err := Restaurar(replica, roto); err != nil {
		t.Logf("la restauracion se niega: %v", err)
		return
	}
	r, err := Verificar(roto, confianza)
	if err != nil {
		t.Logf("cazado: %v", err)
		return
	}

	// Verde. Y ahora se lee lo que estaba suprimido.
	br, _ := CargarBase(roto)
	clave, _ := hex.DecodeString(kat.Entradas[fmt.Sprint(victima)])
	claro, errL := ledger.AbrirComprometido(clave, br.Cadena.Entradas[victima].Nonce,
		br.Cadena.Entradas[victima].Cifrado, br.Cadena.Entradas[victima].Compromiso)
	t.Fatalf("PROPIEDAD TUMBADA: quitada UNA lapida de dos y repuesta su clave, el ensayo sale "+
		"en VERDE: %d entradas, %d vivas, %d supresiones (%v).\n"+
		"  Contenido de la entrada %d, suprimida con %q, leido tras restaurar: %q (err=%v)",
		r.Entradas, r.Vivas, len(r.Supresiones), r.Supresiones, victima,
		esc.Borrados[0].BaseLegal, string(claro), errL)
}
