package main

// Las copias rotas, que son la mitad que hace de esto una puerta.
//
// Una puerta que nunca se ha visto fallar no es una puerta. Un ensayo de
// restauracion que solo ha visto restaurar bien no dice si vigila o si
// acompana: la unica forma de saberlo es romper la copia A PROPOSITO de las
// maneras en que se rompe de verdad y exigir que salga rojo, con el mensaje que
// corresponde a ESE fallo y no a otro.
//
// Cada modo de aqui rompe UNA cosa y esta pensado para que lo cace UNA
// comprobacion distinta. Los que tocan la base o el keystore RECALCULAN el
// manifiesto a proposito: si no, el manifiesto cazaria el cambio primero y la
// comprobacion que se queria ejercitar no se ejecutaria nunca. Es la trampa de
// "la mutacion rompio menos de lo que esperabas porque rompio de mas", al
// reves: aqui el riesgo es que rompa ANTES de lo que se quiere medir.

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"plazum/nucleo/blobs"
)

// ModoRoto describe una forma de romper la copia y que se espera que la cace.
type ModoRoto struct {
	Nombre string
	Que    string // que se rompe, en una linea
	Caza   string // que comprobacion tiene que cazarlo
}

// ModosRotos es la lista completa, y es lo que recorre el control negativo de
// CI. Va aqui y no en el workflow para que anadir un modo sin ejercitarlo sea
// visible: el test comprueba que el workflow los nombra todos.
var ModosRotos = []ModoRoto{
	{"sin-keystore", "la replica no trae keystore.json",
		"la restauracion se niega y dice que se ha perdido el contenido"},
	{"entrada-manipulada", "un byte del cifrado de una entrada viva",
		"el encadenado de la cadena, con el verificador del nucleo"},
	{"lapida-sin-base-legal", "la base legal de la lapida, borrada",
		"la verificacion de lapidas del nucleo"},
	{"clave-resucitada", "la clave de la entrada suprimida, repuesta en el keystore",
		"la comprobacion de que un borrado legal sobrevive a la restauracion"},
	{"keystore-viejo", "la generacion del keystore, retrasada antes del borrado",
		"el contraste de generaciones entre base y keystore"},
	{"evidencia-sustituida", "el contenido de una evidencia, cambiado bajo su misma direccion",
		"el direccionamiento por contenido de nucleo/blobs"},
	{"ancla-dentro", "el fichero de confianza, metido dentro de lo restaurado",
		"la negativa a comprobar una firma con una clave que viaja en la copia"},
	{"manifiesto-fuera-de-sitio", "el manifiesto, con un artefacto que sale del directorio",
		"la negativa a restaurar un nombre con .. o con separadores"},
}

func modoConocido(nombre string) bool {
	for _, m := range ModosRotos {
		if m.Nombre == nombre {
			return true
		}
	}
	return false
}

// rehacerManifiesto vuelve a sellar la replica tras tocarla. Ver el comentario
// de cabecera: sin esto, el manifiesto cazaria el cambio antes que la
// comprobacion que se quiere ejercitar.
func rehacerManifiesto(replica string) error {
	var m Manifiesto
	if err := leerJSON(filepath.Join(replica, NombreManifiesto), &m); err != nil {
		return err
	}
	nombres := make([]string, 0, len(m.Artefactos))
	for n := range m.Artefactos {
		nombres = append(nombres, n)
	}
	sort.Strings(nombres)
	for _, n := range nombres {
		h, err := sha256Fichero(filepath.Join(replica, n))
		if err != nil {
			return err
		}
		m.Artefactos[n] = h
	}
	return escribirJSON(filepath.Join(replica, NombreManifiesto), m)
}

// RomperReplica aplica un modo sobre la replica, antes de restaurar. Devuelve
// falso si el modo no se aplica ahi (ancla-dentro se aplica despues).
func RomperReplica(replica, modo, semilla string) (bool, error) {
	switch modo {
	case "sin-keystore":
		ruta := filepath.Join(replica, NombreKeystore)
		if err := os.Remove(ruta); err != nil {
			return true, fmt.Errorf("no puedo quitar %s de la replica: %w", ruta, err)
		}
		// El manifiesto se deja como estaba a proposito: una replica cuyo
		// manifiesto declara un fichero que no esta es exactamente la forma en
		// que se pierde un keystore de verdad.
		return true, nil

	case "entrada-manipulada":
		base, err := CargarBase(replica)
		if err != nil {
			return true, err
		}
		viva := -1
		suprimidas := map[uint64]bool{}
		for _, l := range base.Cadena.Lapidas {
			suprimidas[l.EntradaBorrada] = true
		}
		for i, e := range base.Cadena.Entradas {
			if !suprimidas[e.Indice] {
				viva = i
				break
			}
		}
		if viva < 0 || len(base.Cadena.Entradas[viva].Cifrado) == 0 {
			return true, fmt.Errorf("no hay ninguna entrada viva con cifrado que manipular")
		}
		base.Cadena.Entradas[viva].Cifrado[0] ^= 0x01
		if err := escribirJSON(filepath.Join(replica, NombreBase), base); err != nil {
			return true, err
		}
		return true, rehacerManifiesto(replica)

	case "lapida-sin-base-legal":
		base, err := CargarBase(replica)
		if err != nil {
			return true, err
		}
		if len(base.Cadena.Lapidas) == 0 {
			return true, fmt.Errorf("la base copiada no tiene lapidas que romper")
		}
		base.Cadena.Lapidas[0].BaseLegal = ""
		if err := escribirJSON(filepath.Join(replica, NombreBase), base); err != nil {
			return true, err
		}
		return true, rehacerManifiesto(replica)

	case "clave-resucitada":
		base, err := CargarBase(replica)
		if err != nil {
			return true, err
		}
		ks, err := CargarKeystore(replica)
		if err != nil {
			return true, err
		}
		if len(base.Cadena.Lapidas) == 0 {
			return true, fmt.Errorf("la base copiada no tiene lapidas, no hay clave que resucitar")
		}
		i := base.Cadena.Lapidas[0].EntradaBorrada
		// La clave se vuelve a derivar igual que en la siembra: es lo que
		// habria en la generacion anterior de la replica del keystore.
		ks.Entradas[fmt.Sprint(i)] = hex.EncodeToString(derivar(semilla, "entrada", i, 32))
		if err := escribirJSON(filepath.Join(replica, NombreKeystore), ks); err != nil {
			return true, err
		}
		return true, rehacerManifiesto(replica)

	case "keystore-viejo":
		base, err := CargarBase(replica)
		if err != nil {
			return true, err
		}
		ks, err := CargarKeystore(replica)
		if err != nil {
			return true, err
		}
		if len(base.Cadena.Lapidas) == 0 {
			return true, fmt.Errorf("la base copiada no tiene lapidas")
		}
		// Solo se retrasa la fecha, sin reponer ninguna clave: asi lo que caza
		// esto es el contraste de generaciones y NO la comprobacion de la clave
		// resucitada, que se ejercita en su propio modo. Un keystore viejo de
		// verdad traeria ademas la clave, y entonces saltarian las dos.
		ks.Generacion = "2026-01-01T00:00:00Z"
		if err := escribirJSON(filepath.Join(replica, NombreKeystore), ks); err != nil {
			return true, err
		}
		return true, rehacerManifiesto(replica)

	case "evidencia-sustituida":
		base, err := CargarBase(replica)
		if err != nil {
			return true, err
		}
		ks, err := CargarKeystore(replica)
		if err != nil {
			return true, err
		}
		cambiada := false
		for i, ev := range base.Evidencias {
			clave, hay := ks.ClaveEvidencia(ev.Hash)
			if !hay {
				continue // la de la entrada suprimida, que ya no tiene clave
			}
			// Se vuelve a sellar OTRO contenido con la misma clave y el mismo
			// nonce, y se deja la direccion original. Asi el descifrado
			// funciona y lo unico que lo caza es que el claro no hashea a la
			// direccion, que es lo que comprueba blobs.Abrir.
			otro, err := blobs.Sellar(clave, ev.Nonce, []byte("evidencia sustituida por otra"))
			if err != nil {
				return true, err
			}
			base.Evidencias[i].Cifrado = otro.Cifrado
			base.Evidencias[i].Compromiso = otro.Compromiso
			cambiada = true
			break
		}
		if !cambiada {
			return true, fmt.Errorf("no hay ninguna evidencia viva que sustituir")
		}
		if err := escribirJSON(filepath.Join(replica, NombreBase), base); err != nil {
			return true, err
		}
		return true, rehacerManifiesto(replica)

	case "manifiesto-fuera-de-sitio":
		// El artefacto de mas lleva el hash del keystore, que SI existe, para
		// que la unica razon posible de rechazo sea el nombre. Si llevara un
		// hash cualquiera, el rechazo podria venir del manifiesto y no de la
		// comprobacion del nombre, y este control negativo estaria dando por
		// cazado algo que no se comprobo.
		var m Manifiesto
		if err := leerJSON(filepath.Join(replica, NombreManifiesto), &m); err != nil {
			return true, err
		}
		m.Artefactos["../fuera-de-la-replica.json"] = m.Artefactos[NombreKeystore]
		return true, escribirJSON(filepath.Join(replica, NombreManifiesto), m)
	}
	return false, nil
}

// RomperTrasRestaurar aplica los modos que solo tienen sentido sobre lo ya
// restaurado. Devuelve la ruta de confianza que hay que usar.
func RomperTrasRestaurar(restaurado, modo, confianza string) (string, error) {
	if modo != "ancla-dentro" {
		return confianza, nil
	}
	b, err := os.ReadFile(confianza) // #nosec G304 -- ruta del propio ensayo
	if err != nil {
		return confianza, err
	}
	dentro := filepath.Join(restaurado, "confianza.json")
	if err := os.WriteFile(dentro, b, 0o600); err != nil {
		return confianza, err
	}
	return dentro, nil
}
