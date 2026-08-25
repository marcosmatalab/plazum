package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// Este fichero es la pasada del atacante sobre el adaptador de OIDC. No prueba
// que el flujo funcione (eso es flujo_test.go): prueba que un IdP hostil, o
// alguien que hable con nuestro retorno, no consigue lo que busca.

// TestUnKidInventadoNoProvocaUnaPeticionPorIntento es el limite de recarga del
// JWKS.
//
// La recarga ante `kid` desconocido existe porque asi se sobrevive a una
// rotacion de claves sin reiniciar. Sin tope, es un amplificador: mil tokens con
// mil `kid` distintos son mil peticiones al IdP, con nuestra IP en sus logs y
// nuestro proceso esperando mil veces.
func TestUnKidInventadoNoProvocaUnaPeticionPorIntento(t *testing.T) {
	i := nuevoIdP(t)
	claves := NuevasClaves(descubrirJWKS(t, i), ClientePorDefecto()).ConIntervaloRecarga(time.Minute)
	v, err := NuevoVerificador(configPrueba(i), claves)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for n := 0; n < 200; n++ {
		tok := i.acunar(t, map[string]any{
			"alg": "RS256", "kid": fmt.Sprintf("inventado-%d", n), "typ": "JWT",
		}, i.cuerpoBueno(clientePrueba))
		if _, err := v.Verificar(ctx, tok, ahoraFijo, esperadoPrueba); err == nil {
			t.Fatalf("un kid inventado (%d) fue aceptado", n)
		}
	}
	if n := i.llamadasJWKS(); n > 1 {
		t.Fatalf("200 kid inventados provocaron %d lecturas del JWKS. Con el intervalo de "+
			"recarga puesto tenia que ser 1: si no, cada intento sale a la red y el IdP nos "+
			"acaba bloqueando (o peor, nos usan para atacarle)", n)
	}

	// Control negativo: pasado el intervalo, SI se recarga. Sin esto, el test
	// de arriba se pasaria igual con la recarga desactivada del todo, y
	// entonces una rotacion de claves dejaria a todo el mundo fuera hasta
	// reiniciar.
	despues := ahoraFijo.Add(2 * time.Minute)
	tok := i.acunar(t, map[string]any{"alg": "RS256", "kid": "otro-mas", "typ": "JWT"}, i.cuerpoBueno(clientePrueba))
	if _, err := v.Verificar(ctx, tok, despues, esperadoPrueba); err == nil {
		t.Fatal("kid inventado aceptado")
	}
	if n := i.llamadasJWKS(); n != 2 {
		t.Fatalf("pasado el intervalo tenia que haber una segunda lectura del JWKS y hubo %d "+
			"en total. Sin recarga, una rotacion de claves del IdP deja fuera a todo el "+
			"mundo hasta que alguien reinicie", n)
	}
}

// TestLaRotacionDeClavesSeResuelveSola es la otra cara: la recarga tiene que
// SERVIR. Un limite que impide recargar nunca protege muy bien y no funciona.
func TestLaRotacionDeClavesSeResuelveSola(t *testing.T) {
	i := nuevoIdP(t)
	claves := NuevasClaves(descubrirJWKS(t, i), ClientePorDefecto()).ConIntervaloRecarga(time.Minute)
	v, err := NuevoVerificador(configPrueba(i), claves)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := v.Verificar(ctx, i.bueno(t, clientePrueba), ahoraFijo, esperadoPrueba); err != nil {
		t.Fatalf("antes de rotar tiene que valer: %v", err)
	}
	// El IdP rota: la clave pasa a llamarse de otra forma.
	i.mu.Lock()
	i.kid = "clave-rsa-2"
	i.mu.Unlock()
	tok := i.acunar(t, map[string]any{"alg": "RS256", "kid": "clave-rsa-2", "typ": "JWT"}, i.cuerpoBueno(clientePrueba))
	// Justo despues no se puede recargar: el intervalo no ha pasado.
	if _, err := v.Verificar(ctx, tok, ahoraFijo, esperadoPrueba); err == nil {
		t.Fatal("la clave nueva no podia conocerse todavia")
	}
	// Pasado el intervalo, se resuelve sin reiniciar nada.
	if _, err := v.Verificar(ctx, tok, ahoraFijo.Add(2*time.Minute), esperadoPrueba); err != nil {
		t.Fatalf("tras el intervalo, la clave rotada tiene que verificar: %v", err)
	}
}

// TestUnJWKSGiganteNoSeCarga: el IdP hostil que devuelve miles de claves.
func TestUnJWKSGiganteNoSeCarga(t *testing.T) {
	i := nuevoIdP(t)
	i.mu.Lock()
	i.clavesExtra = MaxClaves + 10
	i.mu.Unlock()
	claves := NuevasClaves(descubrirJWKS(t, i), ClientePorDefecto())
	v, err := NuevoVerificador(configPrueba(i), claves)
	if err != nil {
		t.Fatal(err)
	}
	_, err = v.Verificar(context.Background(), i.bueno(t, clientePrueba), ahoraFijo, esperadoPrueba)
	if err == nil {
		t.Fatal("un JWKS con mas de 50 claves se cargo entero")
	}
	if !strings.Contains(err.Error(), "y el maximo es") {
		t.Errorf("se rechazo por otro motivo: %v", err)
	}
}

// TestUnCuerpoInfinitoDelIdPNoSeComeLaMemoria. Un tercero no puede decidir
// cuanta memoria gastamos.
func TestUnCuerpoInfinitoDelIdPNoSeComeLaMemoria(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		bloque := strings.Repeat("a", 4096)
		for escrito := 0; escrito < 4*LimiteCuerpo; escrito += len(bloque) {
			if _, err := w.Write([]byte(bloque)); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	_, err := traer(context.Background(), ClientePorDefecto(), srv.URL)
	if err == nil {
		t.Fatal("se leyo entero un cuerpo de 2 MiB del IdP")
	}
	if !strings.Contains(err.Error(), "se corta sin parsear") {
		t.Errorf("se corto por otro motivo: %v", err)
	}
}

// TestUnIdPLentoNoBloqueaParaSiempre. Un IdP que no responde tiene que dar un
// error a los pocos segundos, no quedarse la goroutine hasta que se canse el
// sistema operativo.
func TestUnIdPLentoNoBloqueaParaSiempre(t *testing.T) {
	listo := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-listo:
		case <-r.Context().Done():
		}
	}))
	defer func() { close(listo); srv.Close() }()

	cliente := ClientePorDefecto()
	cliente.Timeout = 150 * time.Millisecond
	inicio := time.Now()
	_, err := traer(context.Background(), cliente, srv.URL)
	if err == nil {
		t.Fatal("un IdP que no contesta nunca devolvio exito")
	}
	if transcurrido := time.Since(inicio); transcurrido > 3*time.Second {
		t.Fatalf("se esperaron %s a un IdP que no contesta: el plazo del cliente no se aplica", transcurrido)
	}
}

// TestElDescubrimientoNoSigueAUnEmisorSuplantado. Es la comprobacion que impide
// que un documento manipulado mande el token endpoint a otro dominio.
func TestElDescubrimientoNoSigueAUnEmisorSuplantado(t *testing.T) {
	malo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer malo.Close()

	casos := []struct {
		nombre string
		doc    func(propio string) map[string]any
		espera string
	}{
		{
			nombre: "el issuer del documento no es el esperado",
			doc: func(propio string) map[string]any {
				return map[string]any{
					"issuer":                 "https://el-de-verdad.ejemplo",
					"authorization_endpoint": propio + "/a",
					"token_endpoint":         propio + "/t",
					"jwks_uri":               propio + "/j",
				}
			},
			espera: "y se esperaba",
		},
		{
			nombre: "el token endpoint apunta a otro dominio",
			doc: func(propio string) map[string]any {
				return map[string]any{
					"issuer":                 propio,
					"authorization_endpoint": propio + "/a",
					"token_endpoint":         malo.URL + "/t",
					"jwks_uri":               propio + "/j",
				}
			},
			espera: "no es el host del emisor",
		},
		{
			nombre: "falta el jwks_uri",
			doc: func(propio string) map[string]any {
				return map[string]any{
					"issuer":                 propio,
					"authorization_endpoint": propio + "/a",
					"token_endpoint":         propio + "/t",
				}
			},
			espera: "no trae jwks_uri",
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			var propio string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(c.doc(propio))
			}))
			defer srv.Close()
			propio = srv.URL

			_, err := Descubrir(context.Background(), ClientePorDefecto(), srv.URL)
			if err == nil {
				t.Fatal("el documento manipulado se acepto")
			}
			if !strings.Contains(err.Error(), c.espera) {
				t.Errorf("se rechazo por otro motivo: %v", err)
			}
		})
	}

	// Control negativo: un documento honrado del mismo servidor SI se acepta.
	i := nuevoIdP(t)
	if _, err := Descubrir(context.Background(), ClientePorDefecto(), i.emisor()); err != nil {
		t.Fatalf("CONTROL NEGATIVO EN ROJO: un descubrimiento honrado se rechaza: %v", err)
	}
}

// TestElSecretoDelClienteNoSaleEnNingunaTraza.
//
// Un client_secret en un log es un incidente con notificacion, no una molestia.
// Se comprueba en los dos sitios por los que se escapa de verdad: el formateo
// del tipo de configuracion, y los errores del canje.
func TestElSecretoDelClienteNoSaleEnNingunaTraza(t *testing.T) {
	const secreto = "ESTO-ES-EL-SECRETO-DEL-CLIENTE-NO-PUEDE-APARECER"
	i := nuevoIdP(t)
	cfg := configPrueba(i)
	cfg.ClienteSecreto = secreto

	for _, formato := range []string{"%v", "%s", "%+v", "%#v"} {
		if s := fmt.Sprintf(formato, cfg); strings.Contains(s, secreto) {
			t.Errorf("el secreto sale con el verbo %s: %s", formato, s)
		}
	}

	// El IdP contesta con un error de OAuth, que es el camino de error mas
	// transitado del canje.
	i.mu.Lock()
	i.respuestaToken = func(w http.ResponseWriter, _ *http.Request) bool {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid_client", "error_description": "credenciales invalidas",
		})
		return true
	}
	i.mu.Unlock()

	a, err := NuevoAutenticador(context.Background(), cfg, nuevaSesionMemoria(), &secretosDeterministas{}, Opciones{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.canjear(context.Background(), "codigo", "verificador-secreto-tambien")
	if err == nil {
		t.Fatal("el canje contra un IdP que responde invalid_client tiene que fallar")
	}
	if strings.Contains(err.Error(), secreto) {
		t.Errorf("el secreto del cliente aparece en el error del canje: %v", err)
	}
	if strings.Contains(err.Error(), "verificador-secreto-tambien") {
		t.Errorf("el verificador PKCE aparece en el error del canje: %v", err)
	}
	// Control negativo: el error SI dice lo que hay que arreglar. Un error que
	// no filtra y no informa es un error inutil, y el arreglo trivial de
	// "devolver siempre 'error'" pasaria la parte de arriba.
	if !strings.Contains(err.Error(), "Certificados y secretos") {
		t.Errorf("el error no dice donde se arregla en el IdP: %v", err)
	}
}

// TestElVerificadorAguantaConcurrencia. 200 personas entrando el lunes a las
// nueve son 200 verificaciones a la vez, y una sola recarga del JWKS.
func TestElVerificadorAguantaConcurrencia(t *testing.T) {
	i := nuevoIdP(t)
	claves := NuevasClaves(descubrirJWKS(t, i), ClientePorDefecto())
	v, err := NuevoVerificador(configPrueba(i), claves)
	if err != nil {
		t.Fatal(err)
	}
	tok := i.bueno(t, clientePrueba)
	var wg sync.WaitGroup
	fallos := make(chan error, 64)
	for n := 0; n < 64; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := v.Verificar(context.Background(), tok, ahoraFijo, esperadoPrueba); err != nil {
				fallos <- err
			}
		}()
	}
	wg.Wait()
	close(fallos)
	for err := range fallos {
		t.Fatalf("verificacion concurrente fallida: %v", err)
	}
	if n := i.llamadasJWKS(); n != 1 {
		t.Errorf("64 verificaciones simultaneas provocaron %d lecturas del JWKS; tenia que "+
			"ser 1", n)
	}
}

// TestUnaClaveRSADebilNoSeAcepta. Si el JWKS elige el tamano de clave, el IdP
// elige cuanta seguridad tenemos.
func TestUnaClaveRSADebilNoSeAcepta(t *testing.T) {
	// Una clave RSA de 1024 bits, escrita a mano en el JWKS.
	debil := map[string]string{
		"kty": "RSA", "kid": "debil", "use": "sig",
		// 1024 bits = 128 bytes de modulo.
		"n": strings.Repeat("qq", 86), // ~128 bytes al decodificar base64url
		"e": "AQAB",
	}
	crudo, err := json.Marshal(map[string]any{"keys": []map[string]string{debil}})
	if err != nil {
		t.Fatal(err)
	}
	i := nuevoIdP(t)
	i.mu.Lock()
	i.cuerpoJWKS = string(crudo)
	i.mu.Unlock()

	claves := NuevasClaves(descubrirJWKS(t, i), ClientePorDefecto())
	if err := claves.recargar(context.Background(), ahoraFijo); err == nil {
		t.Fatal("un JWKS cuya unica clave es RSA de 1024 bits se cargo")
	} else if !strings.Contains(err.Error(), "ninguna es utilizable") {
		t.Errorf("se rechazo por otro motivo: %v", err)
	}
}
