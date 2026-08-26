// Copia vendorizada de github.com/digitorus/pkcs7 (MIT). La procedencia, el
// commit exacto y el procedimiento para seguir los arreglos de aguas arriba
// estan en LEEME.md, en este mismo directorio.
//
// verify.go va RECORTADO, y este es el unico recorte que cambia una decision de
// seguridad, asi que va explicado entero:
//
//  1. Fuera `Verify()`. Aguas arriba es un envoltorio sobre VerifyWithChain(nil)
//     y su propio comentario dice "effectively disabling certificate
//     verification when validating a signature". Un metodo que se llama Verify
//     y no verifica la cadena no puede estar al alcance de la mano en el
//     paquete que decide si un sello de tiempo es de fiar.
//
//  2. Fuera `VerifyWithChain` y `VerifyWithChainAtTime`. Son azucar sobre
//     VerifyWithOpts y el primero acepta un almacen nil, o sea el caso 1 otra
//     vez por otra puerta.
//
//  3. `VerifyWithOpts` EXIGE opts.CurrentTime. Aguas arriba, si viene a cero,
//     comprueba la validez del certificado contra time.Now(). Esa rama rompe la
//     promesa entera de este adaptador, que es que dados el mismo hash, el mismo
//     token y las mismas anclas, dos maquinas cualesquiera dan el mismo
//     veredicto hoy y dentro de cinco anos. Un sello de 2026 verificado en 2031
//     con el reloj de 2031 sale invalido porque el certificado ya caduco, y eso
//     es exactamente lo que un expediente no puede hacer. Con el recorte, esa
//     rama no existe: falta el instante, error, y el error dice cual poner.
//
//  4. `VerifyWithOpts` EXIGE opts.Roots, y esto lo anadio la revision hostil
//     porque SIN ELLO LOS PUNTOS 1 Y 2 ERAN TEATRO. Quitar Verify() y
//     VerifyWithChain(nil) cerro dos puertas y dejo abierta la tercera, que
//     ademas es la unica que quedaba exportada: aguas arriba, y aqui hasta este
//     arreglo, verifySignatureAtTime encadena el certificado SOLO si
//     `opts.Roots != nil`. Con el almacen a nil se comprueba la firma y no se
//     comprueba la cadena, que es exactamente lo que decia el comentario de
//     aguas arriba que se cito para justificar el recorte 1: "effectively
//     disabling certificate verification".
//
//     Medido, no supuesto: un token sellado por una CA que nadie ha declarado
//     salia `<nil>` de VerifyWithOpts con Roots a nil, y "certificate signed by
//     unknown authority" con las anclas de verdad. O sea que el valor CERO de
//     x509.VerifyOptions, que es el que sale de escribir la estructura sin
//     pensar, era "acepto cualquier sello". Ahora falta el almacen y sale
//     ErrSinAnclas, simetrico con ErrSinInstante del punto 3: los dos campos
//     que deciden si un sello es de fiar son obligatorios los dos.
//
//     Lo fija por el lado del atacante TestHostilVerificarSinAnclasNoEsVerificar
//     en adaptadores/tsa, y la afirmacion 4 del fuzzer, que hasta ahora solo
//     recorria la direccion del almacen vacio y NO NIL.
//
// Con 1, 2 y 3 fuera desaparece tambien la unica llamada a time.Now() de todo
// el codigo vendorizado. El resto del fichero es texto de aguas arriba palabra
// por palabra.

package pkcs7

import (
	"crypto/subtle"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"fmt"
	"time"
)

// ErrSinInstante lo devuelve VerifyWithOpts cuando no se le dice CONTRA QUE
// INSTANTE hay que comprobar la validez del certificado del firmante.
// Comprobarlo con errors.Is, nunca comparando el texto.
var ErrSinInstante = errors.New("pkcs7: falta opts.CurrentTime")

// ErrSinAnclas lo devuelve VerifyWithOpts cuando no se le dice CONTRA QUE
// RAICES hay que encadenar el certificado del firmante. Comprobarlo con
// errors.Is, nunca comparando el texto.
//
// Aguas arriba, y aqui hasta la revision hostil, un almacen nil no era un error
// sino un permiso: se comprobaba la firma y se saltaba la cadena entera. Ver el
// punto 4 de la cabecera de este fichero.
var ErrSinAnclas = errors.New("pkcs7: falta opts.Roots")

// VerifyWithOpts checks the signatures of a PKCS7 object.
//
// It accepts x509.VerifyOptions as a parameter.
// This struct contains a root certificate pool, an intermediate certificate pool,
// an optional list of EKUs, and an optional time that certificates should be
// checked as being valid during.
//
// If VerifyOpts.Roots is not nil it verifies the chain of trust of
// the end-entity signer cert to one of the roots in the truststore.
//
// RECORTE RESPECTO DE AGUAS ARRIBA: opts.CurrentTime es obligatorio. Ver la
// cabecera del fichero.
func (p7 *PKCS7) VerifyWithOpts(opts x509.VerifyOptions) (err error) {
	// if KeyUsage isn't set, default to ExtKeyUsageAny
	if opts.KeyUsages == nil {
		opts.KeyUsages = []x509.ExtKeyUsage{x509.ExtKeyUsageAny}
	}

	if len(p7.Signers) == 0 {
		return errors.New("pkcs7: Message has no signers")
	}

	if opts.CurrentTime.IsZero() {
		return fmt.Errorf("%w: sin instante, la validez del certificado del firmante se "+
			"comprobaria contra el reloj de la maquina que verifica, y entonces el mismo "+
			"expediente saldria valido hoy e invalido dentro de cinco anos. "+
			"Arreglo: opts.CurrentTime = la fecha que declara el propio sello", ErrSinInstante)
	}

	// RECORTE 4, ver la cabecera. Sin esto, un almacen nil no era un error sino
	// un permiso: verifySignatureAtTime encadena solo si opts.Roots != nil, asi
	// que se comprobaba la firma del token y no se comprobaba de quien era la
	// clave. Un sello firmado por una CA que nadie ha declarado salia valido.
	if opts.Roots == nil {
		return fmt.Errorf("%w: sin raices, se comprobaria que el token esta firmado pero no "+
			"DE QUIEN es la clave que lo firma, y entonces cualquiera que se fabrique un "+
			"certificado sella lo que quiera. El token trae su propio certificado y darlo "+
			"por bueno porque se firma a si mismo es circular. "+
			"Arreglo: opts.Roots = las raices de las TSAs que el receptor acepta", ErrSinAnclas)
	}

	for _, signer := range p7.Signers {
		if err := verifySignatureAtTime(p7, signer, opts); err != nil {
			return err
		}
	}
	return nil
}

func verifySignatureAtTime(p7 *PKCS7, signer signerInfo, opts x509.VerifyOptions) (err error) {
	signedData := p7.Content
	ee := getCertFromCertsByIssuerAndSerial(p7.Certificates, signer.IssuerAndSerialNumber)
	if ee == nil {
		return errors.New("pkcs7: No certificate for signer")
	}
	if len(signer.AuthenticatedAttributes) > 0 {
		// TODO(fullsailor): First check the content type match
		var (
			digest      []byte
			signingTime time.Time
		)
		err := unmarshalAttribute(signer.AuthenticatedAttributes, OIDAttributeMessageDigest, &digest)
		if err != nil {
			return err
		}
		hash, err := getHashForOID(signer.DigestAlgorithm.Algorithm)
		if err != nil {
			return err
		}
		h := hash.New()
		h.Write(p7.Content)
		computed := h.Sum(nil)
		if subtle.ConstantTimeCompare(digest, computed) != 1 {
			return &MessageDigestMismatchError{
				ExpectedDigest: digest,
				ActualDigest:   computed,
			}
		}
		signedData, err = marshalAttributes(signer.AuthenticatedAttributes)
		if err != nil {
			return err
		}
		err = unmarshalAttribute(signer.AuthenticatedAttributes, OIDAttributeSigningTime, &signingTime)
		if err == nil {
			// signing time found, performing validity check
			if signingTime.After(ee.NotAfter) || signingTime.Before(ee.NotBefore) {
				return fmt.Errorf("pkcs7: signing time %q is outside of certificate validity %q to %q",
					signingTime.Format(time.RFC3339),
					ee.NotBefore.Format(time.RFC3339),
					ee.NotAfter.Format(time.RFC3339))
			}
		}
	}
	if opts.Roots != nil {
		_, err = ee.Verify(opts)
		if err != nil {
			return fmt.Errorf("pkcs7: failed to verify certificate chain: %v", err)
		}
	}
	sigalg, err := getSignatureAlgorithm(signer.DigestEncryptionAlgorithm, signer.DigestAlgorithm)
	if err != nil {
		return err
	}
	return ee.CheckSignature(sigalg, signedData, signer.EncryptedDigest)
}

// GetOnlySigner returns an x509.Certificate for the first signer of the signed
// data payload. If there are more or less than one signer, nil is returned
func (p7 *PKCS7) GetOnlySigner() *x509.Certificate {
	if len(p7.Signers) != 1 {
		return nil
	}
	signer := p7.Signers[0]
	return getCertFromCertsByIssuerAndSerial(p7.Certificates, signer.IssuerAndSerialNumber)
}

// UnmarshalSignedAttribute decodes a single attribute from the signer info
func (p7 *PKCS7) UnmarshalSignedAttribute(attributeType asn1.ObjectIdentifier, out interface{}) error {
	sd, ok := p7.raw.(signedData)
	if !ok {
		return errors.New("pkcs7: payload is not signedData content")
	}
	if len(sd.SignerInfos) < 1 {
		return errors.New("pkcs7: payload has no signers")
	}
	attributes := sd.SignerInfos[0].AuthenticatedAttributes
	return unmarshalAttribute(attributes, attributeType, out)
}

func parseSignedData(data []byte) (*PKCS7, error) {
	var sd signedData
	if _, err := asn1.Unmarshal(data, &sd); err != nil {
		return nil, err
	}
	certs, err := sd.Certificates.Parse()
	if err != nil {
		return nil, err
	}
	// fmt.Printf("--> Signed Data Version %d\n", sd.Version)

	var compound asn1.RawValue
	var content unsignedData

	// The Content.Bytes maybe empty on PKI responses.
	if len(sd.ContentInfo.Content.Bytes) > 0 {
		if _, err := asn1.Unmarshal(sd.ContentInfo.Content.Bytes, &compound); err != nil {
			return nil, err
		}
	}
	// Compound octet string
	if compound.IsCompound {
		if compound.Tag == 4 {
			if _, err = asn1.Unmarshal(compound.Bytes, &content); err != nil {
				return nil, err
			}
		} else {
			content = compound.Bytes
		}
	} else {
		// assuming this is tag 04
		content = compound.Bytes
	}
	return &PKCS7{
		Content:      content,
		Certificates: certs,
		CRLs:         sd.CRLs,
		Signers:      sd.SignerInfos,
		raw:          sd,
	}, nil
}

// MessageDigestMismatchError is returned when the signer data digest does not
// match the computed digest for the contained content
type MessageDigestMismatchError struct {
	ExpectedDigest []byte
	ActualDigest   []byte
}

func (err *MessageDigestMismatchError) Error() string {
	return fmt.Sprintf("pkcs7: Message digest mismatch\n\tExpected: %X\n\tActual  : %X", err.ExpectedDigest, err.ActualDigest)
}

func getSignatureAlgorithm(digestEncryption, digest pkix.AlgorithmIdentifier) (x509.SignatureAlgorithm, error) {
	switch {
	case digestEncryption.Algorithm.Equal(OIDDigestAlgorithmECDSASHA1):
		return x509.ECDSAWithSHA1, nil
	case digestEncryption.Algorithm.Equal(OIDDigestAlgorithmECDSASHA256):
		return x509.ECDSAWithSHA256, nil
	case digestEncryption.Algorithm.Equal(OIDDigestAlgorithmECDSASHA384):
		return x509.ECDSAWithSHA384, nil
	case digestEncryption.Algorithm.Equal(OIDDigestAlgorithmECDSASHA512):
		return x509.ECDSAWithSHA512, nil
	case digestEncryption.Algorithm.Equal(OIDEncryptionAlgorithmRSA),
		digestEncryption.Algorithm.Equal(OIDEncryptionAlgorithmRSASHA1),
		digestEncryption.Algorithm.Equal(OIDEncryptionAlgorithmRSASHA256),
		digestEncryption.Algorithm.Equal(OIDEncryptionAlgorithmRSASHA384),
		digestEncryption.Algorithm.Equal(OIDEncryptionAlgorithmRSASHA512):
		switch {
		case digest.Algorithm.Equal(OIDDigestAlgorithmSHA1):
			return x509.SHA1WithRSA, nil
		case digest.Algorithm.Equal(OIDDigestAlgorithmSHA256):
			return x509.SHA256WithRSA, nil
		case digest.Algorithm.Equal(OIDDigestAlgorithmSHA384):
			return x509.SHA384WithRSA, nil
		case digest.Algorithm.Equal(OIDDigestAlgorithmSHA512):
			return x509.SHA512WithRSA, nil
		default:
			return -1, fmt.Errorf("pkcs7: unsupported digest %q for encryption algorithm %q",
				digest.Algorithm.String(), digestEncryption.Algorithm.String())
		}
	case digestEncryption.Algorithm.Equal(OIDDigestAlgorithmDSA),
		digestEncryption.Algorithm.Equal(OIDDigestAlgorithmDSASHA1):
		switch {
		case digest.Algorithm.Equal(OIDDigestAlgorithmSHA1):
			return x509.DSAWithSHA1, nil
		case digest.Algorithm.Equal(OIDDigestAlgorithmSHA256):
			return x509.DSAWithSHA256, nil
		default:
			return -1, fmt.Errorf("pkcs7: unsupported digest %q for encryption algorithm %q",
				digest.Algorithm.String(), digestEncryption.Algorithm.String())
		}
	case digestEncryption.Algorithm.Equal(OIDEncryptionAlgorithmECDSAP256),
		digestEncryption.Algorithm.Equal(OIDEncryptionAlgorithmECDSAP384),
		digestEncryption.Algorithm.Equal(OIDEncryptionAlgorithmECDSAP521):
		switch {
		case digest.Algorithm.Equal(OIDDigestAlgorithmSHA1):
			return x509.ECDSAWithSHA1, nil
		case digest.Algorithm.Equal(OIDDigestAlgorithmSHA256):
			return x509.ECDSAWithSHA256, nil
		case digest.Algorithm.Equal(OIDDigestAlgorithmSHA384):
			return x509.ECDSAWithSHA384, nil
		case digest.Algorithm.Equal(OIDDigestAlgorithmSHA512):
			return x509.ECDSAWithSHA512, nil
		default:
			return -1, fmt.Errorf("pkcs7: unsupported digest %q for encryption algorithm %q",
				digest.Algorithm.String(), digestEncryption.Algorithm.String())
		}
	case digestEncryption.Algorithm.Equal(OIDEncryptionAlgorithmEDDSA25519):
		return x509.PureEd25519, nil
	default:
		return -1, fmt.Errorf("pkcs7: unsupported algorithm %q",
			digestEncryption.Algorithm.String())
	}
}

func getCertFromCertsByIssuerAndSerial(certs []*x509.Certificate, ias issuerAndSerial) *x509.Certificate {
	for _, cert := range certs {
		if isCertMatchForIssuerAndSerial(cert, ias) {
			return cert
		}
	}
	return nil
}

func unmarshalAttribute(attrs []attribute, attributeType asn1.ObjectIdentifier, out interface{}) error {
	for _, attr := range attrs {
		if attr.Type.Equal(attributeType) {
			_, err := asn1.Unmarshal(attr.Value.Bytes, out)
			return err
		}
	}
	return errors.New("pkcs7: attribute type not in attributes")
}
