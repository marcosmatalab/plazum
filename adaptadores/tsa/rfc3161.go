package tsa

// RFC 3161 a mano: la respuesta de la TSA y el TSTInfo del token.
//
// POR QUE ESTO ESTA AQUI Y NO ES UNA DEPENDENCIA.
//
// Hasta el 26-08-2026 el veredicto de un sello salia de `timestamp.Parse`, que
// usa el `pkcs7` de AGUAS ARRIBA, mientras la firma se comprobaba sobre el
// `pkcs7` VENDORIZADO. Dos lecturas independientes de los mismos bytes, sin
// ninguna identidad dentro de lo firmado que las atara (invariante 7 de
// CLAUDE.md), y el binario llevaba las dos copias dentro.
//
// Peor: `timestamp.Parse` llama a `p7.Verify()` cuando el token trae
// certificados, y `Verify()` es exactamente la funcion que el recorte 1 quito
// de nuestra copia porque su propio comentario aguas arriba dice que inicializa
// un almacen de confianza vacio "effectively disabling certificate
// verification". O sea que el parser del que salia el veredicto ejecutaba una
// verificacion que nuestra copia se niega a exponer.
//
// LA SALIDA NO ERA VENDORIZAR MAS, ERA ENCOGER. Vendorizar `timestamp` habria
// duplicado el deber heredado en vez de quitarlo: son dos LEEME.md, dos tablas
// de procedencia y dos canarios en vez de uno. Lo que hacia falta era quedarse
// con un solo parser, y el TSTInfo es una estructura ASN.1 de once campos que
// cabe aqui.
//
// LA REGLA GENERAL, que vale para la proxima vez y esta en docs/pendientes.md:
// **vendorizar una libreria que otra dependencia tambien importa no quita el
// codigo de en medio, anade una copia.** Antes de vendorizar algo hay que mirar
// quien mas lo arrastra.
//
// QUE SE PARSEA AQUI Y CON QUE:
//
//   - `TimeStampResp` (RFC 3161 apartado 2.4.2), que es lo que contesta la TSA
//     por HTTP: un PKIStatusInfo mas el token. NO es frontera de confianza en
//     el sentido del expediente (es una respuesta viva a una peticion nuestra,
//     atada por el nonce), pero se parsea igual con `encoding/asn1`.
//   - `TSTInfo` (apartado 2.4.2), que SI decide: de ahi salen el hash sellado y
//     el instante. Se parsea sobre `p7.Content`, o sea sobre los bytes que
//     nuestro propio pkcs7 ya ha extraido y cuya firma se comprueba en el mismo
//     sitio. **Un solo parser, los mismos bytes.**

import (
	"crypto"
	"encoding/asn1"
	"errors"
	"fmt"
	"math/big"
	"time"

	"crypto/x509/pkix"

	"plazum/adaptadores/tsa/internal/pkcs7"
)

// Los estados de PKIStatus que valen como sello emitido (RFC 3161, 2.4.2).
const (
	estadoConcedido           = 0 // granted
	estadoConcedidoConCambios = 1 // grantedWithMods
)

// oidTSTInfo es id-ct-TSTInfo, el tipo de contenido que tiene que declarar un
// token para ser un sello de tiempo (RFC 3161, apartado 2.4.2).
var oidTSTInfo = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 16, 1, 4}

// --- Las estructuras ASN.1, tal como las define la norma ---

// pkiStatusInfo es RFC 3161 apartado 2.4.2.
//
// StatusString va como RawValue y no como []string a proposito: es un
// PKIFreeText (SEQUENCE OF UTF8String) que algunas TSAs mandan vacio o con
// codificaciones que `encoding/asn1` no acepta, y **un campo informativo no
// puede tumbar el parseo de la respuesta entera**. Se decodifica aparte y a la
// buena de Dios, que es lo que se puede hacer con un texto de cortesia.
type pkiStatusInfo struct {
	Status       int
	StatusString asn1.RawValue  `asn1:"optional"`
	FailInfo     asn1.BitString `asn1:"optional"`
}

// respuestaSello es TimeStampResp.
type respuestaSello struct {
	Estado pkiStatusInfo
	Token  asn1.RawValue `asn1:"optional"`
}

// improntaMensaje es MessageImprint: que se sello y con que algoritmo.
type improntaMensaje struct {
	Algoritmo pkix.AlgorithmIdentifier
	Resumen   []byte
}

// precisionSello es Accuracy. No se usa para decidir nada, pero tiene que
// parsearse o el campo siguiente se lee corrido.
type precisionSello struct {
	Segundos int `asn1:"optional"`
	Milis    int `asn1:"optional,tag:0"`
	Micros   int `asn1:"optional,tag:1"`
}

// infoSello es TSTInfo. El orden de los campos ES la norma: cambiarlo hace que
// se lea otra cosa.
type infoSello struct {
	Version     int
	Politica    asn1.ObjectIdentifier
	Impronta    improntaMensaje
	NumeroSerie *big.Int
	GenTime     time.Time        `asn1:"generalized"`
	Precision   precisionSello   `asn1:"optional"`
	Ordenado    bool             `asn1:"optional,default:false"`
	Nonce       *big.Int         `asn1:"optional"`
	TSA         asn1.RawValue    `asn1:"optional,tag:0"`
	Extensiones []pkix.Extension `asn1:"optional,tag:1"`
}

// Sello es lo que este adaptador necesita de un token, y nada mas.
//
// No se expone la estructura ASN.1 entera: lo que no se usa no se ensena, y asi
// nadie construye una decision sobre un campo que no se ha comprobado.
type Sello struct {
	// Hash es el algoritmo de la impronta.
	Hash crypto.Hash
	// Sellado es el resumen que la TSA dice haber sellado.
	Sellado []byte
	// Instante es lo que declara el sello, en UTC.
	Instante time.Time
	// Nonce es el de la peticion, si el sello lo devuelve.
	Nonce *big.Int
}

// ErrSinToken lo devuelve leerRespuesta cuando la TSA contesta sin sello.
var ErrSinToken = errors.New("la respuesta no trae token")

// leerRespuesta saca el token de un TimeStampResp y comprueba el estado.
//
// El estado se mira ANTES que nada: una TSA que rechaza la peticion contesta
// con un PKIStatusInfo y sin token, y tratar eso como "token vacio" perderia el
// motivo del rechazo, que es lo unico util que trae esa respuesta.
func leerRespuesta(der []byte) ([]byte, error) {
	var r respuestaSello
	resto, err := asn1.Unmarshal(der, &r)
	if err != nil {
		return nil, fmt.Errorf("la respuesta de la TSA no es un TimeStampResp legible: %w", err)
	}
	if len(resto) > 0 {
		// Bytes de mas detras de una estructura ASN.1 completa. No se ignoran:
		// es la forma de meter dos respuestas en un mismo cuerpo y que cada
		// lector se quede con una distinta.
		return nil, fmt.Errorf("la respuesta de la TSA trae %d bytes de mas detras del "+
			"TimeStampResp; se rechaza en vez de quedarse con la primera mitad", len(resto))
	}
	if r.Estado.Status != estadoConcedido && r.Estado.Status != estadoConcedidoConCambios {
		return nil, fmt.Errorf("la TSA ha rechazado la peticion (PKIStatus %d%s). "+
			"Arreglo: si el estado es 2 (rejection) mira el motivo que devuelve la autoridad; "+
			"si es 3 (waiting) esta TSA pide recogida diferida y este adaptador no la hace",
			r.Estado.Status, textoDelEstado(r.Estado))
	}
	if len(r.Token.FullBytes) == 0 {
		return nil, fmt.Errorf("%w: la TSA dice que ha concedido el sello (PKIStatus %d) y no "+
			"lo manda", ErrSinToken, r.Estado.Status)
	}
	return r.Token.FullBytes, nil
}

// textoDelEstado saca el PKIFreeText si se deja, y si no se calla. Es texto de
// cortesia: no decide nada y no puede tumbar el mensaje de error.
func textoDelEstado(e pkiStatusInfo) string {
	if len(e.StatusString.FullBytes) == 0 {
		return ""
	}
	var textos []string
	if _, err := asn1.Unmarshal(e.StatusString.FullBytes, &textos); err != nil || len(textos) == 0 {
		return ""
	}
	return ": " + textos[0]
}

// leerSello parsea el TSTInfo de un token CMS ya legible.
//
// EL PARSER ES UNO SOLO, y esa es toda la razon de ser de esta funcion: el
// contenido sale de `pkcs7.Parse`, que es la copia vendorizada, y es el MISMO
// objeto sobre el que se comprueba la firma en VerificarOffline. Antes el
// contenido de la izquierda y el de la derecha venian de dos librerias
// distintas y no los ataba nada.
func leerSello(token []byte) (Sello, error) {
	p7, err := pkcs7.Parse(token)
	if err != nil {
		return Sello{}, fmt.Errorf("el token no es un CMS legible: %w", err)
	}
	return selloDelContenido(p7)
}

// selloDelContenido lee el TSTInfo de un CMS ya parseado, para que quien ya
// tiene el p7 no lo parsee dos veces.
func selloDelContenido(p7 *pkcs7.PKCS7) (Sello, error) {
	// EL TIPO DE CONTENIDO, ANTES DE INTERPRETAR NADA. RFC 3161 apartado 2.4.2
	// exige que un token de sello declare id-ct-TSTInfo. Sin esta comprobacion,
	// unos bytes firmados por la misma TSA con OTRO tipo de contenido se
	// leerian como si fueran un sello si por casualidad parsean: la firma es
	// valida sobre esos bytes y el certificado lleva el uso de sellado, asi que
	// nada mas abajo lo detendria. Es confusion de tipos, y se corta aqui.
	if !p7.ContentType.Equal(oidTSTInfo) {
		return Sello{}, fmt.Errorf("el token declara el tipo de contenido %v y un sello de "+
			"tiempo tiene que declarar id-ct-TSTInfo (%v). Lo que hay dentro puede estar "+
			"firmado y no ser un sello", p7.ContentType, oidTSTInfo)
	}
	if len(p7.Content) == 0 {
		return Sello{}, errors.New("el token no trae contenido, asi que no hay TSTInfo que leer")
	}
	var info infoSello
	resto, err := asn1.Unmarshal(p7.Content, &info)
	if err != nil {
		return Sello{}, fmt.Errorf("el contenido del token no es un TSTInfo legible: %w", err)
	}
	if len(resto) > 0 {
		return Sello{}, fmt.Errorf("el TSTInfo trae %d bytes de mas detras; se rechaza en vez "+
			"de quedarse con la primera mitad", len(resto))
	}
	// La version la fija la norma en 1. Un TSTInfo que declara otra cosa no es
	// un sello de esta norma, y seguir leyendolo seria interpretar campos que
	// pueden significar otra cosa.
	if info.Version != 1 {
		return Sello{}, fmt.Errorf("el TSTInfo declara version %d y RFC 3161 fija la 1", info.Version)
	}
	if len(info.Impronta.Resumen) == 0 {
		return Sello{}, errors.New("el sello no dice que ha sellado: messageImprint sin resumen")
	}
	h, ok := hashDeOID(info.Impronta.Algoritmo.Algorithm)
	if !ok {
		return Sello{}, fmt.Errorf("el sello usa el algoritmo de resumen %v, que este "+
			"verificador no conoce. Arreglo: la autoridad tiene que sellar con SHA-256, "+
			"SHA-384 o SHA-512", info.Impronta.Algoritmo.Algorithm)
	}
	if info.GenTime.IsZero() {
		return Sello{}, errors.New("el sello no declara instante (genTime vacio)")
	}
	return Sello{
		Hash:     h,
		Sellado:  info.Impronta.Resumen,
		Instante: info.GenTime.UTC(),
		Nonce:    info.Nonce,
	}, nil
}

// hashDeOID traduce el OID del algoritmo de resumen.
//
// SHA-1 NO ESTA, y no es un olvido: un sello de tiempo cuya impronta es SHA-1
// no ata nada, porque de SHA-1 se saben construir colisiones desde 2017. Aqui
// se sella con SHA-256 y se rechaza lo demas ANTES de comparar, para que el
// rechazo diga que el algoritmo no vale en vez de que el hash no coincide.
func hashDeOID(oid asn1.ObjectIdentifier) (crypto.Hash, bool) {
	switch {
	case oid.Equal(asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}):
		return crypto.SHA256, true
	case oid.Equal(asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 2}):
		return crypto.SHA384, true
	case oid.Equal(asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 3}):
		return crypto.SHA512, true
	}
	return 0, false
}

// --- La peticion ---

// Nada de esto construye la peticion: eso lo sigue haciendo
// github.com/digitorus/timestamp, y a proposito. Construir un TimeStampReq no
// es frontera de confianza (los bytes los ponemos nosotros y el que los lee es
// la TSA), asi que ahi una libreria no decide nada que nos importe.
//
// El objetivo declarado en DEPENDENCIAS.md es que tambien salga, porque
// mientras `timestamp` este importada `github.com/digitorus/pkcs7` sigue en el
// grafo de modulos. Son unas treinta lineas de ASN.1 y no se hacen hoy para no
// mezclar dos cambios en el mismo commit.
