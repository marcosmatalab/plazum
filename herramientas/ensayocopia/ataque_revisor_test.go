package main

// REVISION HOSTIL. Ataque contra la propiedad 5 de verificar.go:
//
//	"la evidencia que colgaba de la entrada borrada tambien tiene que estar
//	 muerta"
//
// La propiedad se comprueba recorriendo base.Evidencias y preguntandole a CADA
// EVIDENCIA de que entrada cuelga (ev.Entrada). Ese numero lo declara el mismo
// fichero que el atacante acaba de reescribir, y no lo ata nada: la cadena
// hashea entradas, no evidencias; la lapida firma el hash de la ENVOLTURA de la
// entrada, no el de la evidencia; el keystore indexa por direccion del blob,
// que no cambia. O sea: quien pueda escribir en la replica decide de que entrada
// cuelga cada evidencia, y por tanto decide a cual de las dos ramas del if entra.

import (
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/blobs"
)

// ATAQUE: recolgar la evidencia de la entrada suprimida de una entrada VIVA.
//
// El atacante hace tres cosas, todas al alcance de quien escribe en un bucket o
// en un NAS (que es el escenario que el propio autor invoca para la travesia de
// rutas):
//
//  1. cambia ev.Entrada de la evidencia suprimida: 2 -> 0.
//  2. repone SOLO la clave de la evidencia en el keystore. NO repone la de la
//     entrada, que es lo unico que mira ErrClaveResucitada.
//  3. rehace el manifiesto, igual que hace romper.go para todos los demas modos.
//
// Resultado esperado por el ensayo: rojo. Resultado real: VERDE, y el PDF del
// documento de identidad de la persona que ejercio el art. 17 vuelve a abrirse.
func TestAtaqueLaEvidenciaSuprimidaSeRecuelgaDeUnaEntradaViva(t *testing.T) {
	trabajo, vivo, replica, confianza := montar(t)

	base, err := CargarBase(replica)
	if err != nil {
		t.Fatal(err)
	}
	ks, err := CargarKeystore(replica)
	if err != nil {
		t.Fatal(err)
	}
	borrada := base.Cadena.Lapidas[0].EntradaBorrada

	// Buscar la evidencia que cuelga de la entrada suprimida y una entrada viva
	// de la que recolgarla.
	i := -1
	for n, ev := range base.Evidencias {
		if ev.Entrada == borrada {
			i = n
			break
		}
	}
	if i < 0 {
		t.Fatal("el escenario ya no trae una evidencia colgada del borrado; sin eso este ataque no mide nada")
	}
	var viva uint64
	hayViva := false
	for _, e := range base.Cadena.Entradas {
		if e.Indice != borrada {
			viva, hayViva = e.Indice, true
			break
		}
	}
	if !hayViva {
		t.Fatal("no hay entrada viva de la que recolgar")
	}

	hash := base.Evidencias[i].Hash
	base.Evidencias[i].Entrada = viva // (1)
	// (2) La clave de la evidencia, la que estaba en la generacion anterior de
	// la replica del keystore. La de la ENTRADA no se repone a proposito.
	ks.Evidencias[hash] = hex.EncodeToString(derivar("prueba", "evidencia", 1, 32))

	if err := escribirJSON(filepath.Join(replica, NombreBase), base); err != nil {
		t.Fatal(err)
	}
	if err := escribirJSON(filepath.Join(replica, NombreKeystore), ks); err != nil {
		t.Fatal(err)
	}
	if err := rehacerManifiesto(replica); err != nil { // (3)
		t.Fatal(err)
	}

	r, err := restaurarY(t, trabajo, vivo, replica, confianza)

	// Primero, ensenar que el dato borrado esta de vuelta y legible. Si esto
	// falla, el ataque no ha resucitado nada y el hallazgo no existe.
	restaurada, errB := CargarBase(filepath.Join(trabajo, "restaurado"))
	if errB == nil {
		clave, _ := hex.DecodeString(ks.Evidencias[hash])
		for _, ev := range restaurada.Evidencias {
			if ev.Hash != hash {
				continue
			}
			claro, errA := blobs.Abrir(clave, ev.Blob())
			if errA != nil {
				t.Fatalf("el ataque no resucita el contenido (%v); sin eso no hay hallazgo", errA)
			}
			t.Logf("DATO SUPRIMIDO RECUPERADO TRAS RESTAURAR: %q", string(claro))
		}
	}

	if !errors.Is(err, ErrSupresionLegible) && !errors.Is(err, ErrClaveResucitada) {
		t.Fatalf("PROPIEDAD TUMBADA: la evidencia de la entrada suprimida %d se recolgo de la "+
			"entrada viva %d y el ensayo salio por err=%v, resultado=%+v.\n"+
			"  El ensayo declara 'supresiones que siguen siendo supresiones: %d' mientras el\n"+
			"  documento de identidad de la persona que ejercio el art. 17 vuelve a abrirse.",
			borrada, viva, err, r, len(r.Supresiones))
	}
}

// ATAQUE 2, y este no necesita adversario. Una evidencia que desaparece del
// listado de la base restaurada no la echa nadie de menos: el ensayo recorre
// base.Evidencias y contrasta cada una contra el keystore, pero NUNCA recorre el
// keystore para preguntar si cada clave de evidencia tiene su evidencia. La
// direccion que falta es la que se usa.
func TestAtaqueUnaEvidenciaQueDesapareceDeLaBaseNoLaEchaNadieDeMenos(t *testing.T) {
	trabajo, vivo, replica, confianza := montar(t)

	base, err := CargarBase(replica)
	if err != nil {
		t.Fatal(err)
	}
	ks, err := CargarKeystore(replica)
	if err != nil {
		t.Fatal(err)
	}

	antes := len(base.Evidencias)
	// Quitar TODAS las evidencias de la base. El keystore sigue trayendo sus
	// claves, o sea que la copia se contradice a si misma.
	base.Evidencias = nil
	if err := escribirJSON(filepath.Join(replica, NombreBase), base); err != nil {
		t.Fatal(err)
	}
	if err := rehacerManifiesto(replica); err != nil {
		t.Fatal(err)
	}

	r, err := restaurarY(t, trabajo, vivo, replica, confianza)
	if err == nil {
		t.Fatalf("PROPIEDAD TUMBADA: la copia perdio las %d evidencias (el keystore restaurado "+
			"sigue teniendo %d claves de evidencia) y el ensayo sale en VERDE con "+
			"'evidencias abiertas y comprobadas contra su direccion: %d'.\n"+
			"  Un ensayo de restauracion que da por buena una copia sin evidencias no mide lo\n"+
			"  que dice medir: el expediente restaurado no tiene con que probar nada.",
			antes, len(ks.Evidencias), r.EvidenciasVivas)
	}
}

// ATAQUE 3, y este tampoco necesita adversario: la copia vuelve CORTA.
//
// Es el fallo mas comun de un respaldo de verdad (replica interrumpida,
// generacion de la base mas vieja que la del keystore) y el ensayo existe para
// cazarlo. No lo caza: recorre las entradas de la cadena restaurada y les pide
// su clave al keystore, pero nunca recorre el keystore para preguntar de que
// entrada es cada clave que sobra. La cadena verifica igual, porque una cadena
// truncada por la cola sigue encadenando.
//
// Y aqui hay testigo independiente, que es lo que lo separa del "truncado de
// cola" que docs/modelo-de-amenaza.md declara indetectable: el keystore se
// replica APARTE, con su propia retencion, y trae una clave por entrada. Un
// keystore con clave para la entrada 3 y una base con tres entradas se
// contradicen solos.
func TestAtaqueUnaBaseQueVuelveCortaPasaElEnsayoConElKeystoreDelatandola(t *testing.T) {
	trabajo, vivo, replica, confianza := montar(t)

	base, err := CargarBase(replica)
	if err != nil {
		t.Fatal(err)
	}
	ks, err := CargarKeystore(replica)
	if err != nil {
		t.Fatal(err)
	}
	antes := len(base.Cadena.Entradas)
	if antes < 2 {
		t.Fatal("el escenario necesita mas de una entrada para poder truncar")
	}
	// Se quita la ULTIMA entrada, que es lo que pasa cuando una replica se
	// corta. La lapida es de la entrada 2, que sigue existiendo.
	base.Cadena.Entradas = base.Cadena.Entradas[:antes-1]
	if err := escribirJSON(filepath.Join(replica, NombreBase), base); err != nil {
		t.Fatal(err)
	}
	if err := rehacerManifiesto(replica); err != nil {
		t.Fatal(err)
	}

	// El testigo: claves del keystore para entradas que la base restaurada ya
	// no tiene. No es un descuadre de recuento (aqui salen 3 y 3, porque la
	// entrada suprimida tampoco tiene clave), es un descuadre de INDICES.
	var huerfanas []string
	hay := map[string]bool{}
	for _, e := range base.Cadena.Entradas {
		hay[fmt.Sprint(e.Indice)] = true
	}
	for i := range ks.Entradas {
		if !hay[i] {
			huerfanas = append(huerfanas, i)
		}
	}
	sort.Strings(huerfanas)

	r, err := restaurarY(t, trabajo, vivo, replica, confianza)
	if err == nil {
		t.Fatalf("PROPIEDAD TUMBADA: la base restaurada trae %d entradas de %d y el ensayo "+
			"sale en VERDE diciendo 'cadena verificada: %d entradas'.\n"+
			"  El keystore restaurado, que es una replica APARTE, trae clave para la(s)\n"+
			"  entrada(s) %v, que la base restaurada NO tiene. Se contradicen solos y nadie\n"+
			"  los contrasta: el ensayo recorre la cadena para pedirle claves al keystore y\n"+
			"  nunca recorre el keystore para preguntar de que entrada es cada clave que sobra.\n"+
			"  Ojo al recuento: %d claves y %d entradas, o sea que contar no basta, hay que\n"+
			"  comparar los indices.",
			len(base.Cadena.Entradas), antes, r.Entradas, huerfanas,
			len(ks.Entradas), len(base.Cadena.Entradas))
	}
}
