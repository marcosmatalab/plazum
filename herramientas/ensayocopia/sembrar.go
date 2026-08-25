package main

// Sembrar: monta una instalacion con lo que el ensayo necesita mirar despues de
// restaurarla, que es exactamente lo que docs/guia.md pide del restore drill:
// la cadena entera, una entrada borrada que siga ilegible, y su lapida con la
// base legal.
//
// Se siembra con los tipos del NUCLEO, no con una imitacion: la cadena es
// ledger.CadenaV2, las lapidas las firma ledger.Borrar y las evidencias las
// sella blobs.Sellar. Si manana esos tipos cambian de forma, este ensayo se
// rompe al compilar, que es donde se quiere que se rompa.
//
// LAS CLAVES DE AQUI SON DETERMINISTAS Y ESO ES CORRECTO SOLO AQUI. Se derivan
// de una semilla con SHA-256 para que dos ejecuciones del ensayo produzcan la
// misma instalacion y un fallo se pueda reproducir. El producto NO genera claves
// asi: las pide a crypto/rand por el puerto Secretos. Una semilla fija en el
// producto seria una clave publicada.

import (
	"crypto/ed25519"
	"crypto/sha256"
	_ "embed"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"plazum/nucleo/blobs"
	"plazum/nucleo/ledger"
)

//go:embed escenario.json
var escenarioEmpotrado []byte

// Escenario es escenario.json: que se siembra y que se borra.
type Escenario struct {
	GeneracionBase     string `json:"generacion_base"`
	GeneracionKeystore string `json:"generacion_keystore"`
	Entradas           []struct {
		Prueba  string `json:"prueba"`
		Recurso string `json:"recurso"`
		Detalle string `json:"detalle"`
	} `json:"entradas"`
	Evidencias []struct {
		Entrada   uint64 `json:"entrada"`
		Contenido string `json:"contenido"`
	} `json:"evidencias"`
	Borrado struct {
		Entrada   uint64 `json:"entrada"`
		BaseLegal string `json:"base_legal"`
		Instante  string `json:"instante"`
	} `json:"borrado"`
}

func cargarEscenario() (Escenario, error) {
	var e Escenario
	if err := json.Unmarshal(escenarioEmpotrado, &e); err != nil {
		return e, fmt.Errorf("escenario.json no es JSON valido: %w", err)
	}
	if len(e.Entradas) == 0 {
		return e, fmt.Errorf("el escenario no tiene entradas, asi que el ensayo verificaria " +
			"una cadena vacia y saldria verde sin mirar nada")
	}
	if e.Borrado.BaseLegal == "" {
		return e, fmt.Errorf("el escenario no declara base legal del borrado, y un borrado sin " +
			"base legal no es un borrado legal: el nucleo lo rechaza al firmarlo")
	}
	if int(e.Borrado.Entrada) >= len(e.Entradas) {
		return e, fmt.Errorf("el escenario borra la entrada %d y solo hay %d",
			e.Borrado.Entrada, len(e.Entradas))
	}
	return e, nil
}

// derivar produce material deterministico a partir de la semilla y un proposito.
func derivar(semilla, proposito string, n uint64, tamano int) []byte {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], n)
	h := sha256.Sum256([]byte("plazum/ensayocopia/" + semilla + "/" + proposito + "/" + string(buf[:])))
	if tamano <= len(h) {
		return h[:tamano]
	}
	out := make([]byte, 0, tamano)
	for i := 0; len(out) < tamano; i++ {
		hh := sha256.Sum256(append(h[:], byte(i)))
		out = append(out, hh[:]...)
	}
	return out[:tamano]
}

// Sembrado es lo que queda escrito, para que quien llama pueda informar sin
// volver a leer el disco.
type Sembrado struct {
	Dir            string
	ClaveOperador  ed25519.PublicKey
	Entradas       int
	Evidencias     int
	EntradaBorrada uint64
	BaseLegal      string
}

// Sembrar escribe una instalacion completa en dir.
func Sembrar(dir, semilla string, esc Escenario) (Sembrado, error) {
	var s Sembrado
	priv := ed25519.NewKeyFromSeed(derivar(semilla, "operador", 0, ed25519.SeedSize))
	pub, ok := priv.Public().(ed25519.PublicKey)
	if !ok {
		return s, fmt.Errorf("la clave del operador no es ed25519, que es imposible salvo " +
			"que alguien haya cambiado el tipo")
	}

	cadena := &ledger.CadenaV2{}
	ks := ledger.NuevoKeystore()
	claves := map[string]string{}

	for i, ent := range esc.Entradas {
		contenido, err := json.Marshal(EntradaClara{Prueba: ent.Prueba, Recurso: ent.Recurso, Detalle: ent.Detalle})
		if err != nil {
			return s, fmt.Errorf("no puedo serializar la entrada %d del escenario: %w", i, err)
		}
		clave := derivar(semilla, "entrada", uint64(i), 32)
		nonce := derivar(semilla, "nonce-entrada", uint64(i), 12)
		if _, err := cadena.Anadir(ks, clave, nonce, contenido); err != nil {
			return s, fmt.Errorf("no puedo anadir la entrada %d a la cadena: %w", i, err)
		}
		claves[fmt.Sprint(i)] = hex.EncodeToString(clave)
	}

	evidencias := make([]Evidencia, 0, len(esc.Evidencias))
	clavesEv := map[string]string{}
	for i, ev := range esc.Evidencias {
		if ev.Entrada >= uint64(len(esc.Entradas)) {
			return s, fmt.Errorf("la evidencia %d se ancla a la entrada %d y solo hay %d",
				i, ev.Entrada, len(esc.Entradas))
		}
		clave := derivar(semilla, "evidencia", uint64(i), 32)
		nonce := derivar(semilla, "nonce-evidencia", uint64(i), 12)
		b, err := blobs.Sellar(clave, nonce, []byte(ev.Contenido))
		if err != nil {
			return s, fmt.Errorf("no puedo sellar la evidencia %d: %w", i, err)
		}
		evidencias = append(evidencias, Evidencia{
			Hash: b.Hash, Nonce: b.Nonce, Cifrado: b.Cifrado, Compromiso: b.Compromiso,
			Entrada: ev.Entrada,
		})
		clavesEv[b.Hash] = hex.EncodeToString(clave)
	}

	// El borrado legal. Es lo unico que hace que este ensayo se distinga de
	// copiar un fichero y volver a leerlo: destruye la clave de la entrada Y la
	// de su evidencia, y deja una lapida firmada con la base legal.
	if _, err := cadena.Borrar(ks, priv, esc.Borrado.Entrada, esc.Borrado.BaseLegal, esc.Borrado.Instante); err != nil {
		return s, fmt.Errorf("no puedo borrar la entrada %d con base legal %q: %w",
			esc.Borrado.Entrada, esc.Borrado.BaseLegal, err)
	}
	delete(claves, fmt.Sprint(esc.Borrado.Entrada))
	for _, ev := range evidencias {
		if ev.Entrada == esc.Borrado.Entrada {
			delete(clavesEv, ev.Hash)
		}
	}

	base := Base{Generacion: esc.GeneracionBase, Cadena: *cadena, Evidencias: evidencias}
	if err := escribirJSON(filepath.Join(dir, NombreBase), base); err != nil {
		return s, err
	}
	if err := escribirJSON(filepath.Join(dir, NombreKeystore), Keystore{
		Generacion: esc.GeneracionKeystore, Entradas: claves, Evidencias: clavesEv,
	}); err != nil {
		return s, err
	}
	// La maestra se escribe en la instalacion y NO entra en copiables. Que este
	// aqui es lo que permite ensenar, al restaurar, que no ha viajado.
	if err := os.WriteFile(filepath.Join(dir, NombreMaestra),
		[]byte(hex.EncodeToString(priv.Seed())+"\n"), 0o600); err != nil {
		return s, fmt.Errorf("no puedo escribir la clave maestra: %w", err)
	}

	return Sembrado{
		Dir: dir, ClaveOperador: pub, Entradas: len(esc.Entradas), Evidencias: len(evidencias),
		EntradaBorrada: esc.Borrado.Entrada, BaseLegal: esc.Borrado.BaseLegal,
	}, nil
}

// EscribirConfianza deja el fichero que aporta EL RECEPTOR: la clave publica del
// operador. Tiene el mismo formato que contexto-demo.json a proposito, para que
// un operador pueda apuntar el ensayo a su contexto de siempre.
func EscribirConfianza(ruta string, pub ed25519.PublicKey) error {
	return escribirJSON(ruta, Confianza{ClaveOperador: hex.EncodeToString(pub)})
}
