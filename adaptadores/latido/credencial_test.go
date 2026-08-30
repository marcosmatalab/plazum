package latido

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// EL DESTINO DEL PULSO ES, POR DISENO, LA URL DE OTRO SERVICIO.
//
// La cabecera de este paquete invita a apuntarlo al monitor de "dead man's
// switch" del propio operador, y esas URL llevan el identificador secreto EN LA
// RUTA: healthchecks.io es `https://hc-ping.com/<uuid>`, y quien tenga ese uuid
// puede marcar el latido como vivo. O sea que el consejo del producto lleva
// derecho a configurar una credencial aqui.
//
// Y `url.Redacted()` NO SIRVE, que era lo que habia: solo sustituye la
// contrasena del userinfo y deja intactos ruta, consulta y fragmento. Usarlo
// creyendo que redacta es peor que no usar nada, porque parece hecho.
func TestElDestinoDelPulsoNoSaleEnteroEnNingunError(t *testing.T) {
	const centinela = "CENTINELA-UUID-DE-PING-QUE-NO-DEBE-SALIR"

	t.Run("cuando el destino redirige", func(t *testing.T) {
		final := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		defer final.Close()
		redirige := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, final.URL+"/"+centinela, http.StatusFound)
			}))
		defer redirige.Close()

		err := CanalHTTP{}.Entregar(context.Background(), redirige.URL+"/pulso", []byte(`{}`))
		if err == nil {
			t.Fatal("una redireccion del destino no dio error")
		}
		if strings.Contains(err.Error(), centinela) {
			t.Errorf("el error lleva el secreto del destino final dentro: %v", err)
		}
	})

	t.Run("cuando la peticion no se puede construir", func(t *testing.T) {
		// Un destino con un caracter de control: pasa la comprobacion de forma
		// y revienta al construir la peticion, que es el otro camino.
		err := CanalHTTP{}.Entregar(context.Background(),
			"https://ejemplo.invalido/\x7f"+centinela, []byte(`{}`))
		if err == nil {
			t.Fatal("un destino ilegible no dio error")
		}
		if strings.Contains(err.Error(), centinela) {
			t.Errorf("el error lleva el destino entero dentro: %v", err)
		}
	})
}
