package tsa

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509/pkix"
	"encoding/asn1"
	"math/big"
	"testing"
	"time"
)

// La TSA de mentira, construida entera aqui.
//
// POR QUE. Hasta el 26-08-2026 la armaba `github.com/digitorus/timestamp` con
// `Timestamp.CreateResponse`, y esa era la ultima razon por la que la
// dependencia seguia en `go.mod`. Con ella fuera, `github.com/digitorus/pkcs7`
// sale del binario y no solo del camino de ejecucion, que es la diferencia
// entre "no lo llamamos" (que el comprador no puede comprobar sin leerse el
// codigo) y "no esta" (que se comprueba con un comando).
//
// Y ADEMAS DA CONTROL QUE LA LIBRERIA NO DABA. Con la respuesta armada aqui se
// puede emitir un token con el tipo de contenido equivocado, con dos firmantes,
// o sin atributos firmados. Con la libreria eso no se podia pedir, asi que esos
// ataques solo se podian probar contra funciones internas y no de punta a punta.
//
// ESTO NO ES CODIGO DE PRODUCCION: plazum no emite sellos, los verifica. Vive
// en un `_test.go` a proposito.

// --- Las estructuras CMS que hacen falta para FIRMAR (RFC 5652) ---
//
// Se declaran aqui y no se reutilizan las del paquete vendorizado porque
// aquellas son `internal` y ademas estan recortadas para verificar. Un emisor
// necesita mas campos que un verificador.

// infoContenidoCMS es ContentInfo y tambien EncapsulatedContentInfo: los dos
// tienen la misma forma.
//
// EL CAMPO NO LLEVA ETIQUETA DE ESTRUCTURA, y esto costo un rato: un
// `asn1.RawValue` con `FullBytes` puesto se emite EN CRUDO, y `encoding/asn1`
// se salta la etiqueta `explicit,tag:0` del campo. El resultado era un CMS al
// que le faltaba el envoltorio [0] y que el verificador rechazaba con "sequence
// truncated". Aqui el envoltorio se construye a mano en el valor, que es
// predecible.
type infoContenidoCMS struct {
	Tipo      asn1.ObjectIdentifier
	Contenido asn1.RawValue
}

// explicito0 envuelve unos bytes DER en un [0] EXPLICIT.
func explicito0(der []byte) asn1.RawValue {
	return asn1.RawValue{Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true, Bytes: der}
}

type emisorYSerie struct {
	Emisor asn1.RawValue
	Serie  *big.Int
}

// atributoCMS es Attribute. El valor es un SET OF AttributeValue, y el campo va
// SIN etiqueta de estructura por el mismo motivo que infoContenidoCMS: un
// RawValue con FullBytes se emite en crudo y se salta el `set`. El envoltorio
// se construye en el valor con conjunto().
type atributoCMS struct {
	Tipo  asn1.ObjectIdentifier
	Valor asn1.RawValue
}

// conjunto envuelve unos bytes DER en un SET OF.
func conjunto(der []byte) asn1.RawValue {
	return asn1.RawValue{Class: asn1.ClassUniversal, Tag: asn1.TagSet, IsCompound: true, Bytes: der}
}

type infoFirmanteCMS struct {
	Version           int
	EmisorYSerie      emisorYSerie
	AlgoritmoResumen  pkix.AlgorithmIdentifier
	AtributosFirmados []atributoCMS `asn1:"optional,omitempty,tag:0"`
	AlgoritmoFirma    pkix.AlgorithmIdentifier
	Firma             []byte
}

// datosFirmadosCMS es SignedData con certificados. Los certificados van en un
// [0] IMPLICIT SET OF Certificate, que en la practica es la concatenacion de
// los DER con esa cabecera.
type datosFirmadosCMS struct {
	Version           int
	AlgoritmosResumen []pkix.AlgorithmIdentifier `asn1:"set"`
	Contenido         infoContenidoCMS
	Certificados      asn1.RawValue
	Firmantes         []infoFirmanteCMS `asn1:"set"`
}

// datosFirmadosSinCert es el mismo SignedData sin el campo de certificados.
//
// Van DOS ESTRUCTURAS y no una con el campo opcional porque un `asn1.RawValue`
// no tiene forma de decir "no me emitas": el valor cero se codifica igual. Dos
// tipos son feos y son correctos; un opcional que a veces emite un tag vacio
// produce un CMS que unos parsers aceptan y otros no, que es peor.
type datosFirmadosSinCert struct {
	Version           int
	AlgoritmosResumen []pkix.AlgorithmIdentifier `asn1:"set"`
	Contenido         infoContenidoCMS
	Firmantes         []infoFirmanteCMS `asn1:"set"`
}

var (
	oidDatosFirmados     = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2}
	oidAtributoTipo      = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 3}
	oidAtributoResumen   = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 4}
	oidAlgoritmoSHA256AI = pkix.AlgorithmIdentifier{Algorithm: asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}}
	oidAlgoritmoRSA      = pkix.AlgorithmIdentifier{Algorithm: asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 1}}
)

// opcionesTSAFalsa es lo que se le puede torcer a la TSA de prueba. Cada campo
// existe porque hay un ataque que lo necesita.
type opcionesTSAFalsa struct {
	// SinCertificado deja el token sin con que comprobar la firma.
	SinCertificado bool
	// TipoDeContenido sustituye id-ct-TSTInfo. Vacio: el correcto.
	TipoDeContenido asn1.ObjectIdentifier
	// NonceFijo ignora el de la peticion y sella siempre con este.
	NonceFijo *big.Int
	// SinAtributosFirmados firma el contenido directamente, sin la SET de
	// atributos. Es legal en CMS y cambia lo que se firma.
	SinAtributosFirmados bool
}

// armarToken construye un token de sello RFC 3161 firmado.
func armarToken(t *testing.T, p *pki, info infoSello, op opcionesTSAFalsa) []byte {
	t.Helper()

	contenido, err := asn1.Marshal(info)
	if err != nil {
		t.Fatalf("no puedo armar el TSTInfo: %v", err)
	}
	tipo := oidTSTInfo
	if len(op.TipoDeContenido) > 0 {
		tipo = op.TipoDeContenido
	}
	// El eContent va envuelto en un OCTET STRING dentro del [0] explicito.
	octetos, err := asn1.Marshal(contenido)
	if err != nil {
		t.Fatal(err)
	}
	eContent := infoContenidoCMS{Tipo: tipo, Contenido: explicito0(octetos)}

	firmante := infoFirmanteCMS{
		Version:          1,
		EmisorYSerie:     emisorYSerie{Emisor: asn1.RawValue{FullBytes: p.hoja.RawIssuer}, Serie: p.hoja.SerialNumber},
		AlgoritmoResumen: oidAlgoritmoSHA256AI,
		AlgoritmoFirma:   oidAlgoritmoRSA,
	}

	// QUE SE FIRMA. Con atributos firmados, la firma va sobre la SET DE
	// ATRIBUTOS (y uno de ellos, messageDigest, lleva el resumen del
	// contenido); sin ellos, sobre el contenido directamente. Las dos formas
	// son legales en CMS y el verificador tiene que hacer lo mismo que el
	// emisor, o no verifica nada.
	aFirmar := contenido
	if !op.SinAtributosFirmados {
		resumen := sha256.Sum256(contenido)
		valorTipo, err := asn1.Marshal(tipo)
		if err != nil {
			t.Fatal(err)
		}
		valorResumen, err := asn1.Marshal(resumen[:])
		if err != nil {
			t.Fatal(err)
		}
		firmante.AtributosFirmados = []atributoCMS{
			{Tipo: oidAtributoTipo, Valor: conjunto(valorTipo)},
			{Tipo: oidAtributoResumen, Valor: conjunto(valorResumen)},
		}
		aFirmar = atributosParaFirmar(t, firmante.AtributosFirmados)
	}

	h := sha256.Sum256(aFirmar)
	firma, err := rsa.SignPKCS1v15(rand.Reader, p.privHoja, crypto.SHA256, h[:])
	if err != nil {
		t.Fatalf("no puedo firmar: %v", err)
	}
	firmante.Firma = firma

	var sdDER []byte
	if op.SinCertificado {
		sdDER, err = asn1.Marshal(datosFirmadosSinCert{
			Version:           3,
			AlgoritmosResumen: []pkix.AlgorithmIdentifier{oidAlgoritmoSHA256AI},
			Contenido:         eContent,
			Firmantes:         []infoFirmanteCMS{firmante},
		})
	} else {
		sdDER, err = asn1.Marshal(datosFirmadosCMS{
			Version:           3,
			AlgoritmosResumen: []pkix.AlgorithmIdentifier{oidAlgoritmoSHA256AI},
			Contenido:         eContent,
			Certificados: asn1.RawValue{
				Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true, Bytes: p.hoja.Raw,
			},
			Firmantes: []infoFirmanteCMS{firmante},
		})
	}
	if err != nil {
		t.Fatalf("no puedo armar el SignedData: %v", err)
	}
	token, err := asn1.Marshal(infoContenidoCMS{
		Tipo:      oidDatosFirmados,
		Contenido: explicito0(sdDER),
	})
	if err != nil {
		t.Fatalf("no puedo armar el ContentInfo: %v", err)
	}
	return token
}

// atributosParaFirmar codifica los atributos como SET OF, que es lo que se
// firma. Es la misma operacion que hace `marshalAttributes` del verificador: si
// las dos no coinciden, ninguna firma cuadra nunca.
func atributosParaFirmar(t *testing.T, attrs []atributoCMS) []byte {
	t.Helper()
	b, err := asn1.Marshal(struct {
		A []atributoCMS `asn1:"set"`
	}{A: attrs})
	if err != nil {
		t.Fatal(err)
	}
	var raw asn1.RawValue
	if _, err := asn1.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	return raw.Bytes // sin la cabecera de SEQUENCE
}

// armarRespuesta envuelve un token en un TimeStampResp con el estado dado.
func armarRespuesta(t *testing.T, estado int, token []byte) []byte {
	t.Helper()
	r := respuestaSello{Estado: pkiStatusInfo{Status: estado}}
	if len(token) > 0 {
		r.Token = asn1.RawValue{FullBytes: token}
	}
	b, err := asn1.Marshal(r)
	if err != nil {
		t.Fatalf("no puedo armar el TimeStampResp: %v", err)
	}
	return b
}

// leerPeticion parsea el TimeStampReq que manda el adaptador, para que la TSA
// de mentira pueda contestar a lo que se le ha pedido de verdad.
//
// Existe tambien como comprobacion cruzada: si `construirPeticion` emitiera
// algo que no es un TimeStampReq, esto lo caza en cuanto la TSA falsa intente
// leerlo, sin necesitar una libreria ajena que haga de arbitro.
func leerPeticion(t *testing.T, der []byte) peticionSello {
	t.Helper()
	var p peticionSello
	resto, err := asn1.Unmarshal(der, &p)
	if err != nil {
		t.Fatalf("la consulta que manda el adaptador no es un TimeStampReq legible: %v", err)
	}
	if len(resto) > 0 {
		t.Fatalf("la consulta trae %d bytes de mas", len(resto))
	}
	return p
}

// tstInfoPara arma el TSTInfo que corresponde a una peticion.
func tstInfoPara(p peticionSello, cuando time.Time, nonce *big.Int) infoSello {
	if nonce == nil {
		nonce = p.Nonce
	}
	return infoSello{
		Version:     1,
		Politica:    asn1.ObjectIdentifier{1, 2, 3, 4, 1},
		Impronta:    p.Impronta,
		NumeroSerie: big.NewInt(42),
		GenTime:     cuando,
		Nonce:       nonce,
	}
}
