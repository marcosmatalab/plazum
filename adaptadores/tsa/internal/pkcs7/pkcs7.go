// Copia vendorizada de github.com/digitorus/pkcs7 (MIT). La procedencia, el
// commit exacto y el procedimiento para seguir los arreglos de aguas arriba
// estan en LEEME.md, en este mismo directorio.
//
// pkcs7.go va RECORTADO. Lo que se ha quitado, y por que, esta enumerado en
// LEEME.md con la salida de gosec que lo obligo. En resumen: fuera todo lo de
// firmar y todo lo de cifrar, y Parse solo acepta SignedData, que es lo unico
// que trae un token RFC 3161. Lo que queda es texto de aguas arriba palabra por
// palabra.

// Package pkcs7 implements parsing and generation of some PKCS#7 structures.
package pkcs7

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"fmt"

	_ "crypto/sha1" // #nosec G505 -- for crypto.SHA1: se registra la implementacion para que crypto.SHA1.New() no entre en panico si un token declara ese OID. Aqui no se elige el algoritmo, se lee el que trae el token, y quien decide es getSignatureAlgorithm
)

// PKCS7 Represents a PKCS7 structure
type PKCS7 struct {
	Content      []byte
	Certificates []*x509.Certificate
	CRLs         []pkix.CertificateList
	Signers      []signerInfo
	raw          interface{}
}

type contentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"explicit,optional,tag:0"`
}

// ErrUnsupportedContentType is returned when a PKCS7 content is not supported.
// Currently only Data (1.2.840.113549.1.7.1), Signed Data (1.2.840.113549.1.7.2),
// and Enveloped Data are supported (1.2.840.113549.1.7.3)
var ErrUnsupportedContentType = errors.New("pkcs7: cannot parse data: unimplemented content type")

// ErrUnsupportedAlgorithm tells you when our quick dev assumptions have failed
//
// Viene de decrypt.go de aguas arriba, que aqui no se ha vendorizado. Se copia
// tal cual porque getHashForOID lo devuelve.
var ErrUnsupportedAlgorithm = errors.New("pkcs7: cannot decrypt data: only RSA, DES, DES-EDE3, AES-256-CBC and AES-128-GCM supported")

type unsignedData []byte

var (
	// Signed Data OIDs
	OIDData                   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}
	OIDSignedData             = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}
	OIDEnvelopedData          = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 3}
	OIDEncryptedData          = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 6}
	OIDAttributeContentType   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 3}
	OIDAttributeMessageDigest = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 4}
	OIDAttributeSigningTime   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 5}

	// Digest Algorithms
	OIDDigestAlgorithmSHA1   = asn1.ObjectIdentifier{1, 3, 14, 3, 2, 26}
	OIDDigestAlgorithmSHA256 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}
	OIDDigestAlgorithmSHA384 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 2}
	OIDDigestAlgorithmSHA512 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 3}

	OIDDigestAlgorithmDSA     = asn1.ObjectIdentifier{1, 2, 840, 10040, 4, 1}
	OIDDigestAlgorithmDSASHA1 = asn1.ObjectIdentifier{1, 2, 840, 10040, 4, 3}

	OIDDigestAlgorithmECDSASHA1   = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 1}
	OIDDigestAlgorithmECDSASHA256 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 2}
	OIDDigestAlgorithmECDSASHA384 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 3}
	OIDDigestAlgorithmECDSASHA512 = asn1.ObjectIdentifier{1, 2, 840, 10045, 4, 3, 4}

	// Signature Algorithms
	OIDEncryptionAlgorithmRSA       = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 1}
	OIDEncryptionAlgorithmRSASHA1   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 5}
	OIDEncryptionAlgorithmRSASHA256 = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 11}
	OIDEncryptionAlgorithmRSASHA384 = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 12}
	OIDEncryptionAlgorithmRSASHA512 = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 13}

	OIDEncryptionAlgorithmECDSAP256 = asn1.ObjectIdentifier{1, 2, 840, 10045, 3, 1, 7}
	OIDEncryptionAlgorithmECDSAP384 = asn1.ObjectIdentifier{1, 3, 132, 0, 34}
	OIDEncryptionAlgorithmECDSAP521 = asn1.ObjectIdentifier{1, 3, 132, 0, 35}

	OIDEncryptionAlgorithmEDDSA25519 = asn1.ObjectIdentifier{1, 3, 101, 112}
)

func getHashForOID(oid asn1.ObjectIdentifier) (crypto.Hash, error) {
	switch {
	case oid.Equal(OIDDigestAlgorithmSHA1), oid.Equal(OIDDigestAlgorithmECDSASHA1),
		oid.Equal(OIDDigestAlgorithmDSA), oid.Equal(OIDDigestAlgorithmDSASHA1),
		oid.Equal(OIDEncryptionAlgorithmRSA):
		return crypto.SHA1, nil
	case oid.Equal(OIDDigestAlgorithmSHA256), oid.Equal(OIDDigestAlgorithmECDSASHA256):
		return crypto.SHA256, nil
	case oid.Equal(OIDDigestAlgorithmSHA384), oid.Equal(OIDDigestAlgorithmECDSASHA384):
		return crypto.SHA384, nil
	case oid.Equal(OIDDigestAlgorithmSHA512), oid.Equal(OIDDigestAlgorithmECDSASHA512):
		return crypto.SHA512, nil
	}
	return crypto.Hash(0), ErrUnsupportedAlgorithm
}

// Parse decodes a DER encoded PKCS7 package
//
// RECORTE RESPECTO DE AGUAS ARRIBA: las ramas de EnvelopedData y EncryptedData
// ya no llaman a un parser, devuelven un error que dice por que. El unico
// contenido que un token RFC 3161 puede traer es SignedData; los otros dos son
// contenido cifrado, y descifrar no es algo que este adaptador haga nunca. Con
// esto no entran en el proceso dos asn1.Unmarshal mas sobre bytes de un
// tercero, y encrypt.go y decrypt.go se pueden quedar fuera del repositorio.
func Parse(data []byte) (p7 *PKCS7, err error) {
	if len(data) == 0 {
		return nil, errors.New("pkcs7: input data is empty")
	}
	var info contentInfo
	der, err := ber2der(data)
	if err != nil {
		return nil, err
	}
	rest, err := asn1.Unmarshal(der, &info)
	if len(rest) > 0 {
		err = asn1.SyntaxError{Msg: "trailing data"}
		return
	}
	if err != nil {
		return
	}

	switch {
	case info.ContentType.Equal(OIDSignedData):
		return parseSignedData(info.Content.Bytes)
	case info.ContentType.Equal(OIDEnvelopedData), info.ContentType.Equal(OIDEncryptedData):
		return nil, fmt.Errorf("%w: el CMS es de tipo %v, que es contenido cifrado. "+
			"Esta copia vendorizada solo verifica SignedData, que es lo unico que trae un "+
			"token RFC 3161: el cifrado y el descifrado se quitaron a proposito "+
			"(adaptadores/tsa/internal/pkcs7/LEEME.md). Si esto sale sobre un sello de una "+
			"TSA, el sello esta mal formado, no le falta soporte al verificador",
			ErrUnsupportedContentType, info.ContentType)
	}
	return nil, ErrUnsupportedContentType
}

func (raw rawCertificates) Parse() ([]*x509.Certificate, error) {
	if len(raw.Raw) == 0 {
		return nil, nil
	}

	var val asn1.RawValue
	if _, err := asn1.Unmarshal(raw.Raw, &val); err != nil {
		return nil, err
	}

	return x509.ParseCertificates(val.Bytes)
}

func isCertMatchForIssuerAndSerial(cert *x509.Certificate, ias issuerAndSerial) bool {
	return cert.SerialNumber.Cmp(ias.SerialNumber) == 0 && bytes.Equal(cert.RawIssuer, ias.IssuerName.FullBytes)
}
