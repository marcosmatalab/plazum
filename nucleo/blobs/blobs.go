// Package blobs es el almacen de evidencia con fichero: content-addressed y
// cifrado por entrada con el mismo regimen comprometido del ledger.
//
// Por que dentro del nucleo: la logica (direccion por hash del contenido en
// claro, cifrado con compromiso de clave, borrado por destruccion de clave)
// es pura. La persistencia (tabla blobs de SQLite) es del adaptador; asi hay
// UN solo fichero que respaldar y Litestream lo replica todo junto.
package blobs

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"dutiq/nucleo/ledger"
)

// Blob es una evidencia cifrada, direccionada por el hash de su contenido en
// claro. El hash identifica; el compromiso ata la clave; borrar la clave del
// keystore la hace irrecuperable sin tocar la direccion.
type Blob struct {
	Hash       string // sha256 hex del contenido en claro
	Nonce      []byte
	Cifrado    []byte
	Compromiso []byte
}

// Sellar cifra un contenido y lo direcciona. Clave (32B) y nonce (12B) los
// aporta el llamador: el nucleo no tiene aleatoriedad.
func Sellar(clave, nonce, contenido []byte) (Blob, error) {
	cifrado, compromiso, err := ledger.SellarComprometido(clave, nonce, contenido)
	if err != nil {
		return Blob{}, err
	}
	h := sha256.Sum256(contenido)
	return Blob{Hash: hex.EncodeToString(h[:]), Nonce: nonce, Cifrado: cifrado, Compromiso: compromiso}, nil
}

// Abrir descifra y COMPRUEBA que el contenido corresponde a la direccion: un
// blob cuyo claro no hashea a su direccion es un blob sustituido.
func Abrir(clave []byte, b Blob) ([]byte, error) {
	claro, err := ledger.AbrirComprometido(clave, b.Nonce, b.Cifrado, b.Compromiso)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(claro)
	if hex.EncodeToString(h[:]) != b.Hash {
		return nil, fmt.Errorf("el contenido no corresponde a la direccion %s: blob sustituido", b.Hash[:12])
	}
	return claro, nil
}
