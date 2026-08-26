package tsa

// La consulta RFC 3161, construida aqui.
//
// POR QUE ESTAS CUARENTA LINEAS EXISTEN. Con ellas sale del proyecto la ultima
// dependencia de `digitorus`, y con ella `github.com/digitorus/pkcs7` sale del
// BINARIO, no solo del camino de ejecucion.
//
// La diferencia no es cosmetica en un producto de seguridad. Hasta ahora la
// frase era "no lo llamamos", y eso el comprador no lo puede comprobar sin
// leerse el codigo. "No esta" se comprueba con un comando:
//
//	go list -deps ./cmd/plazum | grep digitorus
//
// y lo vigila TestElBinarioNoLlevaNadaDeDigitorus.
//
// Y construir la consulta nunca fue frontera de confianza: los bytes los
// ponemos nosotros y quien los lee es la TSA. Se traia de fuera porque el
// ASN.1 "a mano son semanas", que era cierto del CMS entero y falso de una
// estructura de seis campos.

import (
	"crypto"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"fmt"
	"math/big"
)

// oidSHA256 es el algoritmo con el que sella este adaptador. Se declara aqui y
// no se toma de una tabla: es una decision, no una configuracion.
var oidAlgoritmoSHA256 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}

// peticionSello es TimeStampReq (RFC 3161, apartado 2.4.1).
//
// El orden de los campos ES la norma. Y `CertReq` va SIN `optional` a
// proposito: su valor por defecto en ASN.1 es `false`, asi que marcarlo
// opcional haria que `encoding/asn1` lo omitiera al codificar cuando vale
// false... y tambien cuando vale true si algun dia alguien lo pone a false por
// error. Se codifica siempre, que es lo que hace que la TSA no tenga que
// adivinar.
type peticionSello struct {
	Version     int
	Impronta    improntaMensaje
	Politica    asn1.ObjectIdentifier `asn1:"optional"`
	Nonce       *big.Int              `asn1:"optional"`
	ConCert     bool                  `asn1:"optional,default:false"`
	Extensiones []pkix.Extension      `asn1:"optional,tag:0"`
}

// ErrHashDeLongitudRara lo devuelve construirPeticion cuando el resumen no mide
// lo que mide un SHA-256.
var ErrHashDeLongitudRara = errors.New("el resumen a sellar no mide 32 bytes")

// construirPeticion arma el TimeStampReq DER.
//
// SE PIDE SIEMPRE EL CERTIFICADO (`certReq` a true). Sin el, el token no trae
// con que comprobar la firma y no se puede encadenar a las anclas del receptor,
// asi que no serviria para nada de lo que este producto promete. No es una
// opcion: es la unica forma en que un sello vale.
func construirPeticion(hash []byte, nonce *big.Int) ([]byte, error) {
	if len(hash) != crypto.SHA256.Size() {
		return nil, fmt.Errorf("%w: son %d y tienen que ser %d. Aqui se sella el resumen "+
			"SHA-256 de la raiz Merkle, no otra cosa",
			ErrHashDeLongitudRara, len(hash), crypto.SHA256.Size())
	}
	if nonce == nil || nonce.Sign() == 0 {
		// El nonce ata la respuesta a ESTA peticion. Uno vacio, o cero, deja
		// que una respuesta guardada para el mismo hash se reproduzca, que es
		// exactamente lo que el nonce existe para cerrar.
		return nil, errors.New("la consulta va sin nonce, asi que una respuesta guardada de " +
			"antes para el mismo hash se podria reproducir. Arreglo: generar el nonce con " +
			"crypto/rand antes de llamar")
	}
	return asn1.Marshal(peticionSello{
		Version:  1,
		Impronta: improntaMensaje{Algoritmo: pkix.AlgorithmIdentifier{Algorithm: oidAlgoritmoSHA256}, Resumen: hash},
		Nonce:    nonce,
		ConCert:  true,
	})
}
