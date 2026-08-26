// Copia vendorizada de github.com/digitorus/pkcs7 (MIT). La procedencia, el
// commit exacto y el procedimiento para seguir los arreglos de aguas arriba
// estan en LEEME.md, en este mismo directorio.
//
// sign.go va RECORTADO A LAS ESTRUCTURAS. De las 500 lineas de aguas arriba se
// quedan las seis declaraciones ASN.1 que el camino de verificacion necesita
// para deserializar un SignedData, y la funcion que vuelve a serializar los
// atributos autenticados para comprobar la firma sobre ellos.
//
// Fuera: SignedData, NewSignedData, AddSigner, AddSignerChain, SignWithoutAttr,
// Finish, DegenerateCertificate y todo lo demas de firmar. Este adaptador
// verifica sellos de una TSA, no los emite, y ese codigo arrastraba crypto/dsa
// y crypto/rand a la frontera de confianza sin que nadie los llame.
//
// El nombre del fichero se conserva a proposito, aunque ya no firme nada: es lo
// que hace que el `git diff` contra aguas arriba del procedimiento de LEEME.md
// se pueda leer fichero por fichero.

package pkcs7

import (
	"crypto/x509/pkix"
	"encoding/asn1"
	"math/big"
)

type signedData struct {
	Version                    int                        `asn1:"default:1"`
	DigestAlgorithmIdentifiers []pkix.AlgorithmIdentifier `asn1:"set"`
	ContentInfo                contentInfo
	Certificates               rawCertificates        `asn1:"optional,tag:0"`
	CRLs                       []pkix.CertificateList `asn1:"optional,tag:1"`
	SignerInfos                []signerInfo           `asn1:"set"`
}

type signerInfo struct {
	Version                   int `asn1:"default:1"`
	IssuerAndSerialNumber     issuerAndSerial
	DigestAlgorithm           pkix.AlgorithmIdentifier
	AuthenticatedAttributes   []attribute `asn1:"optional,omitempty,tag:0"`
	DigestEncryptionAlgorithm pkix.AlgorithmIdentifier
	EncryptedDigest           []byte
	UnauthenticatedAttributes []attribute `asn1:"optional,omitempty,tag:1"`
}

type attribute struct {
	Type  asn1.ObjectIdentifier
	Value asn1.RawValue `asn1:"set"`
}

func marshalAttributes(attrs []attribute) ([]byte, error) {
	encodedAttributes, err := asn1.Marshal(struct {
		A []attribute `asn1:"set"`
	}{A: attrs})
	if err != nil {
		return nil, err
	}

	// Remove the leading sequence octets
	var raw asn1.RawValue
	if _, err := asn1.Unmarshal(encodedAttributes, &raw); err != nil {
		return nil, err
	}
	return raw.Bytes, nil
}

type rawCertificates struct {
	Raw asn1.RawContent
}

type issuerAndSerial struct {
	IssuerName   asn1.RawValue
	SerialNumber *big.Int
}
