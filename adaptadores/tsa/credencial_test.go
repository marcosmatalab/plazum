package tsa

import (
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// LA URL DE UNA TSA LA CONFIGURA EL OPERADOR, Y PUEDE SER UNA CREDENCIAL.
//
// FreeTSA y Certum, que son las que trae el binario, son publicas y con ellas
// esto da igual. Pero `Cadena.Autoridades` es configuracion, y una TSA de pago
// pone el token en la consulta de la URL: en cuanto alguien configure una, el
// error de sellado la filtra al log, a la pantalla y sobre todo al bloque
// copiable de `plazum doctor --issue`, que esta hecho PARA PEGARLO EN UN ISSUE
// PUBLICO.
//
// Aqui se filtraba DOS VECES en la misma linea, y la segunda es la que no se ve
// leyendo: interpolada con %s, y otra vez dentro del error de `http.Client`, que
// lleva la URL entera y viajaba envuelto con %w. Por eso el arreglo no es solo
// redactar el mensaje: es NO ENVOLVER ese error.
func TestLaURLDeUnaTSANoSaleEnNingunError(t *testing.T) {
	const centinela = "CENTINELA-TOKEN-DE-TSA-QUE-NO-DEBE-SALIR"
	hash := sha256.Sum256([]byte("lo que sea"))

	t.Run("cuando la TSA no responde", func(t *testing.T) {
		// Un servidor cerrado: el error de red es de verdad, no simulado.
		srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		url := srv.URL + "/sellar?token=" + centinela
		srv.Close()

		c := &Cadena{Autoridades: []Autoridad{{Nombre: "la de pruebas", URL: url}}}
		_, err := c.Sellar(hash[:])
		if err == nil {
			t.Fatal("sellar contra un servidor cerrado no dio error")
		}
		if strings.Contains(err.Error(), centinela) {
			t.Errorf("el error lleva la credencial dentro: %v", err)
		}
	})

	t.Run("cuando la TSA contesta un codigo que no es 200", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer srv.Close()
		url := srv.URL + "/sellar?token=" + centinela

		c := &Cadena{Autoridades: []Autoridad{{Nombre: "la de pruebas", URL: url}}}
		_, err := c.Sellar(hash[:])
		if err == nil {
			t.Fatal("un 403 de la TSA no dio error")
		}
		if strings.Contains(err.Error(), centinela) {
			t.Errorf("el error lleva la credencial dentro: %v", err)
		}
		// Y sigue diciendo lo que hace falta para arreglarlo: quien fallo y con
		// que codigo. Sin esta mitad, "no filtra" y "no dice nada" se leerian
		// igual de verdes.
		if !strings.Contains(err.Error(), "403") ||
			!strings.Contains(err.Error(), "la de pruebas") {
			t.Errorf("el error no dice quien fallo ni por que: %v", err)
		}
	})
}
