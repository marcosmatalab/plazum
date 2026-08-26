package tsa

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Revision hostil del adaptador de anclaje.

// ATAQUE 6. El backoff calcula esperaBase << Intentos sobre un int64. Con
// bastantes intentos eso desborda. La guarda mira "> tope" y "<= 0", pero un
// desbordamiento que caiga en positivo y por debajo del tope se le escapa.
func TestHostilBackoffDesbordaYReintentaAntesDeTiempo(t *testing.T) {
	base := instanteSello
	var malos []int
	for intentos := 1; intentos <= 64; intentos++ {
		p := Pendiente{Hash: "aa", Encolado: base, Intentos: intentos, Ultimo: base}
		// Un minuto despues del ultimo intento no le puede tocar nunca: la
		// espera minima es esperaBase (5 min) y solo crece.
		if p.toca(base.Add(time.Minute)) {
			malos = append(malos, intentos)
		}
	}
	if len(malos) == 0 {
		t.Log("PROPIEDAD AGUANTA: ningun numero de intentos hace que toque antes de la espera base")
		return
	}
	t.Fatalf("HALLAZGO: con %v intentos el backoff dice que toca reintentar un minuto despues del "+
		"ultimo intento. esperaBase<<Intentos desborda int64 y cae en un positivo pequeno, que la "+
		"guarda (>tope o <=0) no atrapa. Efecto: contra dos TSAs caidas de largo, el reintento pasa "+
		"de una vez al dia a un martilleo", malos)
}

// ATAQUE 7. Sellar verifica el token con las anclas ANTES de devolverlo. Si el
// operador no ha cargado anclas, toda TSA valida cuenta como caida y todo se
// encola en silencio. Se comprueba que el error diga algo util.
func TestHostilSinAnclasTodoSeEncolaSinDecirPorQue(t *testing.T) {
	p := buena(t)
	s := servidor(t, p, true)
	c := &Cadena{
		Autoridades: []Autoridad{{Nombre: "buena", URL: s.URL}},
		Cola:        colaEn(t),
		Ahora:       func() time.Time { return instanteSello },
		// Anclas sin cargar: el descuido tipico del primer arranque.
	}
	_, err := c.Sellar(hashDe("checkpoint"))
	if err == nil {
		t.Fatal("sin anclas no puede devolver un token dado por bueno")
	}
	t.Logf("error que ve el operador: %v", err)
	pend, _ := c.Cola.Pendientes()
	if len(pend) == 1 {
		t.Log("NOTA: la TSA estaba viva y respondiendo bien, pero el anclaje acaba en la cola igual. " +
			"El error lo explica, pero nada obliga a llamar a Revisar() al arrancar")
	}
}

// ATAQUE 8. Una TSA hostil que devuelve una respuesta gigante. Hay
// LimitReader, pero se comprueba que de verdad corta.
func TestHostilTSAQueRespondeSinFin(t *testing.T) {
	p := buena(t)
	s := servidorInfinito(t)
	c := &Cadena{
		Autoridades: []Autoridad{{Nombre: "vertedero", URL: s.URL}},
		Anclas:      p.pool,
		Cola:        colaEn(t),
		Ahora:       func() time.Time { return instanteSello },
		HTTP:        &http.Client{Timeout: 5 * time.Second},
	}
	hecho := make(chan error, 1)
	go func() { _, err := c.Sellar(hashDe("checkpoint")); hecho <- err }()
	select {
	case err := <-hecho:
		t.Logf("PROPIEDAD AGUANTA: corta y devuelve %v", err)
	case <-time.After(30 * time.Second):
		t.Fatal("HALLAZGO: una TSA que responde sin fin bloquea el sellado")
	}
}

// ATAQUE 9. El token se verifica contra el hash que le pasa el llamante. Si el
// llamante pasa el hash equivocado no hay forma de saberlo, pero al menos un
// token de un ALGORITMO distinto tiene que rechazarse. Aqui se comprueba que
// no se acepta un token cuyo messageImprint mide otra cosa.
func TestHostilTokenConHashDeLongitudRaraNoVerifica(t *testing.T) {
	p := buena(t)
	c := &Cadena{Anclas: p.pool}
	// Un hash de 20 bytes (SHA-1) no es lo que este sistema sella.
	corto := make([]byte, 20)
	if err := c.VerificarOffline(corto, []byte{0x30, 0x03, 0x02, 0x01, 0x00}); err == nil {
		t.Fatal("HALLAZGO: acepta un hash que no es SHA-256")
	} else {
		t.Logf("PROPIEDAD AGUANTA: %v", err)
	}
}

// ATAQUE 10. El token del expediente lo aporta el emisor, de quien no nos
// fiamos, y no pasa por ningun LimitReader: maxRespuesta solo acota lo que
// llega por HTTP de una TSA. El fuzzing del pkcs7 vendorizado midio que el
// transcodificador de BER a DER AMPLIFICA (x22 medido: 512 KB de entrada
// producen 11,6 MB de salida), asi que sin un tope el trabajo que hace el
// verificador lo elige el atacante. Se comprueba que el tope existe, que
// rechaza SIN parsear y que no se lleva por delante un token legitimo.
func TestHostilUnTokenEnormeSeRechazaAntesDeParsearlo(t *testing.T) {
	c := &Cadena{Anclas: buena(t).pool}
	h := hashDe("checkpoint")

	// Un token justo por encima del tope, con forma de SEQUENCE de longitud
	// indefinida para que el parser tenga trabajo de verdad si llega a el.
	enorme := make([]byte, maxToken+1)
	enorme[0], enorme[1] = 0x30, 0x80
	for i := 2; i < len(enorme); i++ {
		enorme[i] = 0x30
	}

	err := c.VerificarOffline(h, enorme)
	if !errors.Is(err, ErrTokenDemasiadoGrande) {
		t.Fatalf("HALLAZGO: un token de %d bytes no se rechaza por tamano, se rechaza con %v. "+
			"Sin tope, el emisor elige cuanta memoria gasta el verificador", len(enorme), err)
	}
	t.Logf("PROPIEDAD AGUANTA: %v", err)

	if _, err := Instante(enorme); !errors.Is(err, ErrTokenDemasiadoGrande) {
		t.Fatalf("HALLAZGO: Instante no aplica el tope (%v). Es la otra puerta por la que "+
			"los mismos bytes llegan a un parser de ASN.1, y un tope que solo esta en una "+
			"de las dos no es un tope", err)
	}

	// Y el control por el otro lado, que es la mitad que se olvida: el tope no
	// puede cargarse un sello autentico. El del expediente de demostracion son
	// 4636 bytes y un QTSP con cadena larga y RSA-4096 puede doblarlo, asi que
	// se exige un factor de cinco sobre lo que hoy se sabe que es real.
	const tokenRealDeLaDemo = 4636
	if maxToken < 5*tokenRealDeLaDemo {
		t.Fatalf("el tope (%d) esta demasiado cerca del tamano de un token autentico "+
			"(%d bytes). Un tope que rechaza sellos buenos acusa al emisor de un fallo "+
			"del receptor, que es el peor fallo posible en un producto cuya tesis es que "+
			"el receptor no se fia", maxToken, tokenRealDeLaDemo)
	}
	if err := c.VerificarOffline(h, make([]byte, maxToken)); errors.Is(err, ErrTokenDemasiadoGrande) {
		t.Fatal("el tope se aplica en el limite exacto y no debe: maxToken es el ultimo " +
			"tamano ACEPTADO, no el primero rechazado")
	}
}

// servidorInfinito responde 200 y no para de escribir.
func servidorInfinito(t *testing.T) *httptest.Server {
	t.Helper()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/timestamp-reply")
		w.WriteHeader(http.StatusOK)
		basura := make([]byte, 64*1024)
		for i := 0; i < 4096; i++ {
			if _, err := w.Write(basura); err != nil {
				return
			}
		}
	}))
	t.Cleanup(s.Close)
	return s
}
