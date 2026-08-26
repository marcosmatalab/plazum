package tsa

import (
	"crypto/x509/pkix"
	"encoding/asn1"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"plazum/adaptadores/tsa/internal/pkcs7"
)

// Las guardas del parser RFC 3161 propio, una a una.
//
// Se prueban contra `selloDelContenido` y no montando un CMS entero a proposito:
// lo que aqui se vigila es la lectura del TSTInfo, y hacerla pasar por la firma
// mezclaria dos cosas. La firma tiene sus propios tests hostiles.

var oidPolitica = asn1.ObjectIdentifier{1, 2, 3, 4}
var oidSHA256 = asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}

// tstInfoDe arma un TSTInfo DER a partir de la estructura, para poder torcer un
// campo cada vez.
func tstInfoDe(t *testing.T, i infoSello) []byte {
	t.Helper()
	b, err := asn1.Marshal(i)
	if err != nil {
		t.Fatalf("no puedo armar el TSTInfo de prueba: %v", err)
	}
	return b
}

func tstInfoBueno() infoSello {
	return infoSello{
		Version:     1,
		Politica:    oidPolitica,
		Impronta:    improntaMensaje{Algoritmo: pkix.AlgorithmIdentifier{Algorithm: oidSHA256}, Resumen: make([]byte, 32)},
		NumeroSerie: big.NewInt(42),
		GenTime:     time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC),
	}
}

func cmsCon(tipo asn1.ObjectIdentifier, contenido []byte) *pkcs7.PKCS7 {
	return &pkcs7.PKCS7{ContentType: tipo, Content: contenido}
}

// CONTROL POSITIVO, y va primero: si esto no pasara, todos los rechazos de
// abajo estarian pasando porque el parser rechaza siempre.
func TestUnTSTInfoCorrectoSeLee(t *testing.T) {
	s, err := selloDelContenido(cmsCon(oidTSTInfo, tstInfoDe(t, tstInfoBueno())))
	if err != nil {
		t.Fatalf("un TSTInfo correcto tiene que leerse: %v", err)
	}
	if s.Hash != 5 { // crypto.SHA256
		t.Errorf("el algoritmo no se ha leido: %v", s.Hash)
	}
	if len(s.Sellado) != 32 {
		t.Errorf("el resumen no se ha leido: %d bytes", len(s.Sellado))
	}
	if !s.Instante.Equal(time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)) {
		t.Errorf("el instante no se ha leido: %v", s.Instante)
	}
	if s.Instante.Location() != time.UTC {
		t.Errorf("el instante no sale en UTC (%v). Un sello que se lee en la zona de quien "+
			"verifica hace que el mismo expediente cuente otra hora en cada maquina",
			s.Instante.Location())
	}
}

// LA CONFUSION DE TIPOS, que es la guarda que el tipo de contenido existe para
// cerrar. Unos bytes firmados por la misma TSA con OTRO tipo de contenido no se
// pueden leer como un sello: la firma es valida sobre ellos y el certificado
// lleva el uso de sellado, asi que nada mas abajo lo detendria.
func TestUnContenidoQueNoEsUnSelloNoSeLeeComoSello(t *testing.T) {
	// El mismo TSTInfo, byte a byte, pero declarado como id-data.
	oidData := asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}
	_, err := selloDelContenido(cmsCon(oidData, tstInfoDe(t, tstInfoBueno())))
	if err == nil {
		t.Fatal("HALLAZGO: un CMS que declara id-data se ha leido como un sello de tiempo. " +
			"RFC 3161 apartado 2.4.2 exige id-ct-TSTInfo, y sin esa comprobacion cualquier " +
			"cosa que la TSA haya firmado y que por casualidad parsee vale como sello")
	}
	if !strings.Contains(err.Error(), "id-ct-TSTInfo") {
		t.Errorf("el error no dice que se esperaba: %v", err)
	}
}

func TestLasGuardasDelTSTInfo(t *testing.T) {
	casos := []struct {
		nombre  string
		torcer  func(i *infoSello)
		enError string
		porque  string
	}{
		{
			"version distinta de 1",
			func(i *infoSello) { i.Version = 2 },
			"version",
			"RFC 3161 fija la version 1. Un TSTInfo que declara otra no es un sello de esta " +
				"norma, y seguir leyendo sus campos es interpretar bytes que pueden significar otra cosa",
		},
		{
			"sin resumen en la impronta",
			func(i *infoSello) { i.Impronta.Resumen = nil },
			"messageImprint",
			"un sello que no dice QUE ha sellado no ata nada: la comparacion con el hash del " +
				"checkpoint compararia contra vacio",
		},
		{
			"algoritmo de impronta desconocido",
			func(i *infoSello) { i.Impronta.Algoritmo.Algorithm = asn1.ObjectIdentifier{1, 2, 3} },
			"algoritmo de resumen",
			"si el algoritmo no se conoce no se puede saber si el resumen corresponde al " +
				"contenido, y aceptarlo dejaria pasar cualquier cosa de 32 bytes",
		},
		{
			"impronta SHA-1",
			func(i *infoSello) {
				i.Impronta.Algoritmo.Algorithm = asn1.ObjectIdentifier{1, 3, 14, 3, 2, 26}
				i.Impronta.Resumen = make([]byte, 20)
			},
			"algoritmo de resumen",
			"de SHA-1 se saben construir colisiones desde 2017: un sello con impronta SHA-1 " +
				"no ata el contenido que dice sellar",
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			i := tstInfoBueno()
			c.torcer(&i)
			_, err := selloDelContenido(cmsCon(oidTSTInfo, tstInfoDe(t, i)))
			if err == nil {
				t.Fatalf("HALLAZGO: se ha aceptado un TSTInfo con %s.\n  Por que importa: %s",
					c.nombre, c.porque)
			}
			if !strings.Contains(err.Error(), c.enError) {
				t.Errorf("el error no menciona %q, asi que quien lo lee no sabe que mirar: %v",
					c.enError, err)
			}
		})
	}
}

// Basura detras de una estructura completa. Es la forma de meter dos cosas en
// unos mismos bytes y que cada lector se quede con una distinta.
func TestNoSeAceptaNadaDetrasDelTSTInfo(t *testing.T) {
	bueno := tstInfoDe(t, tstInfoBueno())
	conCola := append(append([]byte{}, bueno...), 0x05, 0x00)

	if _, err := selloDelContenido(cmsCon(oidTSTInfo, bueno)); err != nil {
		t.Fatalf("control positivo: sin cola tiene que leerse (%v)", err)
	}
	_, err := selloDelContenido(cmsCon(oidTSTInfo, conCola))
	if err == nil {
		t.Fatal("HALLAZGO: se han aceptado bytes de mas detras del TSTInfo. Con eso, dos " +
			"lectores del mismo token pueden quedarse cada uno con una cosa")
	}
	if !strings.Contains(err.Error(), "bytes de mas") {
		t.Errorf("el error no dice lo que pasa: %v", err)
	}
	// Y sin contenido no hay nada que leer.
	if _, err := selloDelContenido(cmsCon(oidTSTInfo, nil)); err == nil {
		t.Fatal("un token sin contenido se ha leido como sello")
	}
}

// La respuesta de la TSA: el estado se mira ANTES que el token.
func TestLaRespuestaDeLaTSASeLeeConSuEstado(t *testing.T) {
	armar := func(t *testing.T, estado int, token []byte) []byte {
		t.Helper()
		r := respuestaSello{Estado: pkiStatusInfo{Status: estado}}
		if len(token) > 0 {
			r.Token = asn1.RawValue{FullBytes: token}
		}
		b, err := asn1.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	// Un "token" cualquiera: leerRespuesta no lo parsea, solo lo extrae.
	tokenFalso, err := asn1.Marshal(asn1.RawValue{Tag: asn1.TagOctetString, Class: asn1.ClassUniversal, Bytes: []byte{1, 2, 3}})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("concedido con token", func(t *testing.T) {
		got, err := leerRespuesta(armar(t, 0, tokenFalso))
		if err != nil {
			t.Fatalf("un PKIStatus 0 con token tiene que salir bien: %v", err)
		}
		if len(got) == 0 {
			t.Fatal("no ha devuelto el token")
		}
	})
	t.Run("concedido con cambios tambien vale", func(t *testing.T) {
		if _, err := leerRespuesta(armar(t, 1, tokenFalso)); err != nil {
			t.Fatalf("grantedWithMods es un sello emitido: %v", err)
		}
	})
	t.Run("rechazado", func(t *testing.T) {
		_, err := leerRespuesta(armar(t, 2, nil))
		if err == nil {
			t.Fatal("HALLAZGO: un rechazo de la TSA se ha tratado como sello concedido")
		}
		if !strings.Contains(err.Error(), "PKIStatus 2") {
			t.Errorf("el error no dice el estado, que es lo unico util que trae un rechazo: %v", err)
		}
	})
	t.Run("rechazado PERO con token: manda el estado", func(t *testing.T) {
		// Es la trampa: una TSA (o un intermediario) que manda un token y a la
		// vez dice que no lo concede. Mirar el token primero lo daria por bueno.
		if _, err := leerRespuesta(armar(t, 2, tokenFalso)); err == nil {
			t.Fatal("HALLAZGO: se ha cogido el token de una respuesta que dice PKIStatus 2. " +
				"El estado se mira ANTES que el token, o el estado no sirve de nada")
		}
	})
	t.Run("concedido y sin token", func(t *testing.T) {
		_, err := leerRespuesta(armar(t, 0, nil))
		if err == nil {
			t.Fatal("HALLAZGO: una respuesta que concede y no manda sello ha salido bien")
		}
	})
	t.Run("basura detras", func(t *testing.T) {
		conCola := append(armar(t, 0, tokenFalso), 0x05, 0x00)
		if _, err := leerRespuesta(conCola); err == nil {
			t.Fatal("se han aceptado bytes de mas detras del TimeStampResp")
		}
	})
	t.Run("no es ASN.1", func(t *testing.T) {
		if _, err := leerRespuesta([]byte("hola")); err == nil {
			t.Fatal("se ha aceptado algo que no es un TimeStampResp")
		}
	})
}

// servidorConNonceFijo es una TSA que IGNORA el nonce de la peticion y sella
// siempre con el mismo. Es lo que hace un intermediario que devuelve un token
// guardado de antes: el hash coincide, la firma es de la TSA legitima, y lo
// unico que no cuadra es el nonce.
func servidorConNonceFijo(t *testing.T, p *pki, fijo *big.Int) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		pet := leerPeticion(t, b)
		token := armarToken(t, p, tstInfoPara(pet, instanteSello, fijo), opcionesTSAFalsa{})
		w.Header().Set("Content-Type", "application/timestamp-reply")
		_, _ = w.Write(armarRespuesta(t, 0, token))
	}))
	t.Cleanup(s.Close)
	return s
}

// LA REPRODUCCION DE UN SELLO VIEJO, que es lo unico que el nonce cierra.
//
// Sin esta comprobacion, un intermediario que guarde un sello de la TSA
// legitima para un hash cualquiera puede devolverlo mas tarde: la firma
// verifica, la cadena encadena, el hash es el que se pidio si el atacante
// espera al sello del mismo contenido, y el instante es el de entonces. El
// nonce es lo unico que ata la respuesta a ESTA peticion.
//
// SE ESCRIBIO PORQUE UNA MUTACION SALIO VERDE. Borrar la comprobacion del nonce
// no ponia rojo nada: la propiedad estaba en el codigo y no en la suite, que es
// exactamente el estado en el que una comprobacion desaparece en una limpieza
// y nadie se entera.
func TestUnSelloConOtroNonceEsUnaReproduccion(t *testing.T) {
	p := buena(t)
	h := hashDe("checkpoint")

	// CONTROL POSITIVO: con la TSA que respeta el nonce, pedir funciona. Sin
	// esto, lo de abajo pasaria igual si pedir fallara siempre.
	sana := servidor(t, p, true)
	c := &Cadena{Anclas: p.pool}
	if _, err := c.pedir(Autoridad{Nombre: "legitima", URL: sana.URL}, h); err != nil {
		t.Fatalf("la TSA que devuelve el nonce de la peticion tiene que valer: %v", err)
	}

	// Y ahora la que sella siempre con el mismo nonce.
	mentirosa := servidorConNonceFijo(t, p, big.NewInt(1))
	_, err := c.pedir(Autoridad{Nombre: "reproductora", URL: mentirosa.URL}, h)
	if err == nil {
		t.Fatal("HALLAZGO: se ha aceptado un sello cuyo nonce no es el de la peticion. " +
			"Con eso, un intermediario devuelve un sello guardado de antes y el verificador " +
			"lo da por nuevo: la firma es de la TSA legitima y el hash es el que se pidio")
	}
	if !strings.Contains(err.Error(), "nonce") {
		t.Errorf("el error no dice que el problema es el nonce: %v", err)
	}
}

// EL NONCE SE COMPRUEBA SOBRE EL TOKEN, NO SOBRE LA RESPUESTA, y esto cambio
// con el parser propio.
//
// La respuesta de la TSA no la firma nadie: es un sobre. El TSTInfo SI va
// firmado. Comprobar el nonce contra un campo del sobre deja que un
// intermediario devuelva un token guardado de antes con un sobre que lleve el
// nonce de hoy, que es exactamente el ataque que el nonce existe para cerrar.
func TestElNonceSeComprubaSobreElTokenFirmado(t *testing.T) {
	p := buena(t)
	s := servidor(t, p, true)
	c := &Cadena{Anclas: p.pool}
	h := hashDe("checkpoint")

	token, err := c.pedir(Autoridad{Nombre: "legitima", URL: s.URL}, h)
	if err != nil {
		t.Fatalf("la TSA de prueba tiene que sellar: %v", err)
	}
	sello, err := leerSello(token)
	if err != nil {
		t.Fatal(err)
	}
	if sello.Nonce == nil {
		t.Fatal("el sello no trae nonce, asi que la comprobacion de `pedir` no puede estar " +
			"midiendo nada: o la TSA de prueba dejo de devolverlo, o se esta comprobando " +
			"contra el sobre otra vez")
	}
}
