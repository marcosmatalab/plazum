package oidc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

const clientePrueba = "cliente-de-dutiq"

func configPrueba(i *idpFalso) Configuracion {
	return Configuracion{
		Emisor:      i.emisor(),
		ClienteID:   clientePrueba,
		RedirectURI: "https://dutiq.ejemplo/auth/retorno",
	}
}

func verificadorDePrueba(t *testing.T, i *idpFalso) *Verificador {
	t.Helper()
	prov, err := Descubrir(context.Background(), ClientePorDefecto(), i.emisor())
	if err != nil {
		t.Fatalf("descubrir el IdP de prueba: %v", err)
	}
	v, err := NuevoVerificador(configPrueba(i), NuevasClaves(prov.JWKS, ClientePorDefecto()))
	if err != nil {
		t.Fatalf("construir el verificador: %v", err)
	}
	return v
}

var esperadoPrueba = Esperado{Nonce: "nonce-de-la-peticion"}

// TestUnTokenHonradoPasa es el control negativo GLOBAL de todo este fichero.
//
// Sin el, una tabla entera de tokens rechazados no demuestra nada: un
// verificador que devuelve error siempre la pasa entera. Cada caso de
// TestFalsificacionesDelIDToken repite ademas su propio control con el mismo
// verificador, porque un control global no descarta que la falsificacion haya
// estropeado el montaje.
func TestUnTokenHonradoPasa(t *testing.T) {
	i := nuevoIdP(t)
	v := verificadorDePrueba(t, i)
	id, err := v.Verificar(context.Background(), i.bueno(t, clientePrueba), ahoraFijo, esperadoPrueba)
	if err != nil {
		t.Fatalf("el token honrado tiene que pasar y dio: %v", err)
	}
	if id.Sujeto != "usuario-0001" {
		t.Errorf("sujeto %q, se esperaba usuario-0001", id.Sujeto)
	}
	if id.Correo != "ana@ejemplo.es" || !id.CorreoVerificado {
		t.Errorf("el correo verificado no llego: %+v", id)
	}
	if !id.Caduca.Equal(ahoraFijo.Add(time.Hour)) {
		t.Errorf("caducidad %s, se esperaba %s", id.Caduca, ahoraFijo.Add(time.Hour))
	}
}

// TestFalsificacionesDelIDToken es la tabla del encargo: cada fila falsifica el
// token de UNA forma concreta, deja el resto bien formado, y exige el rechazo.
//
// Cada fila lleva su control negativo dentro: con el MISMO verificador y el
// MISMO instante, el token honrado tiene que pasar. Sin eso, "se rechaza" y "se
// rechaza todo" son indistinguibles.
func TestFalsificacionesDelIDToken(t *testing.T) {
	casos := []struct {
		nombre string
		// cab y cuerpo modifican el token ANTES de firmarlo.
		cab    func(*idpFalso, map[string]any)
		cuerpo func(map[string]any)
		// romper manipula el token YA acunado, para las falsificaciones que
		// no se pueden expresar antes de firmar.
		romper func(*testing.T, *idpFalso, string) string
		espera string
	}{
		{
			nombre: "alg none, que es el fallo clasico",
			cab:    func(_ *idpFalso, c map[string]any) { c["alg"] = "none" },
			espera: "no trae firma",
		},
		{
			nombre: "alg HS256 firmado con la clave PUBLICA del IdP (confusion de algoritmo)",
			cab:    func(_ *idpFalso, c map[string]any) { c["alg"] = "HS256" },
			espera: "no esta en la lista blanca",
		},
		{
			nombre: "alg ES256 con la clave RSA que firmo de verdad",
			cab:    func(_ *idpFalso, c map[string]any) { c["alg"] = "ES256" },
			espera: "el JWKS declara",
		},
		{
			nombre: "kid inventado",
			cab:    func(_ *idpFalso, c map[string]any) { c["kid"] = "clave-que-no-existe" },
			espera: "kid no encontrado",
		},
		{
			nombre: "kid de OTRA clave del mismo IdP",
			cab:    func(i *idpFalso, c map[string]any) { c["kid"] = i.kidEC },
			espera: "el JWKS declara",
		},
		{
			nombre: "typ at+jwt, o sea un access token colado como ID token",
			cab:    func(_ *idpFalso, c map[string]any) { c["typ"] = "at+jwt" },
			espera: "se esperaba un ID token",
		},
		{
			nombre: "iss de otro emisor",
			cuerpo: func(c map[string]any) { c["iss"] = "https://idp-del-atacante.ejemplo" },
			espera: "el emisor del token es",
		},
		{
			nombre: "iss con barra final de mas",
			cuerpo: func(c map[string]any) { c["iss"] = c["iss"].(string) + "/" },
			espera: "el emisor del token es",
		},
		{
			nombre: "aud de otra aplicacion",
			cuerpo: func(c map[string]any) { c["aud"] = "otra-aplicacion-del-tenant" },
			espera: "no incluye el cliente_id",
		},
		{
			nombre: "aud multiple sin azp",
			cuerpo: func(c map[string]any) { c["aud"] = []string{clientePrueba, "otra"} },
			espera: "no trae `azp`",
		},
		{
			nombre: "aud multiple con azp de otra aplicacion",
			cuerpo: func(c map[string]any) {
				c["aud"] = []string{clientePrueba, "otra"}
				c["azp"] = "otra"
			},
			espera: "no abre sesion aqui",
		},
		{
			nombre: "sin sub",
			cuerpo: func(c map[string]any) { delete(c, "sub") },
			espera: "no trae `sub`",
		},
		{
			nombre: "exp ya pasado",
			cuerpo: func(c map[string]any) { c["exp"] = ahoraFijo.Add(-2 * time.Hour).Unix() },
			espera: "caduco a las",
		},
		{
			nombre: "exp justo fuera del margen de reloj",
			cuerpo: func(c map[string]any) {
				c["exp"] = ahoraFijo.Add(-MargenRelojPorDefecto - time.Second).Unix()
			},
			espera: "caduco a las",
		},
		{
			nombre: "sin exp",
			cuerpo: func(c map[string]any) { delete(c, "exp") },
			espera: "no trae `exp`",
		},
		{
			nombre: "nbf en el futuro",
			cuerpo: func(c map[string]any) { c["nbf"] = ahoraFijo.Add(time.Hour).Unix() },
			espera: "no vale hasta las",
		},
		{
			nombre: "iat absurdamente en el futuro",
			cuerpo: func(c map[string]any) { c["iat"] = ahoraFijo.Add(72 * time.Hour).Unix() },
			espera: "que es el futuro",
		},
		{
			nombre: "sin iat",
			cuerpo: func(c map[string]any) { delete(c, "iat") },
			espera: "no trae `iat`",
		},
		{
			nombre: "nonce de otra peticion",
			cuerpo: func(c map[string]any) { c["nonce"] = "nonce-de-otra-sesion" },
			espera: "no es el de esta peticion",
		},
		{
			nombre: "sin nonce habiendolo pedido",
			cuerpo: func(c map[string]any) { delete(c, "nonce") },
			espera: "se pidio `nonce` y el token no lo trae",
		},
		{
			nombre: "cuerpo cambiado despues de firmar",
			romper: func(t *testing.T, i *idpFalso, tok string) string {
				partes := strings.Split(tok, ".")
				var c map[string]any
				crudo, err := base64.RawURLEncoding.DecodeString(partes[1])
				if err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(crudo, &c); err != nil {
					t.Fatal(err)
				}
				c["sub"] = "el-jefe"
				nuevo, err := json.Marshal(c)
				if err != nil {
					t.Fatal(err)
				}
				partes[1] = base64.RawURLEncoding.EncodeToString(nuevo)
				return strings.Join(partes, ".")
			},
			espera: "la firma no valida",
		},
		{
			nombre: "firma de otro token del mismo IdP (trasplante)",
			romper: func(t *testing.T, i *idpFalso, tok string) string {
				otro := i.cuerpoBueno(clientePrueba)
				otro["sub"] = "otro-usuario"
				firmadoOtro := i.acunar(t, map[string]any{"alg": "RS256", "kid": i.kid, "typ": "JWT"}, otro)
				return strings.Join(strings.Split(tok, ".")[:2], ".") + "." + strings.Split(firmadoOtro, ".")[2]
			},
			espera: "la firma no valida",
		},
		{
			nombre: "firma vacia con alg RS256 en la cabecera",
			romper: func(_ *testing.T, _ *idpFalso, tok string) string {
				return strings.Join(strings.Split(tok, ".")[:2], ".") + "."
			},
			espera: "no trae firma",
		},
		{
			nombre: "cuatro partes",
			romper: func(_ *testing.T, _ *idpFalso, tok string) string { return tok + ".extra" },
			espera: "tres partes",
		},
		{
			nombre: "base64 con relleno, que RFC 7515 prohibe",
			romper: func(_ *testing.T, _ *idpFalso, tok string) string {
				partes := strings.Split(tok, ".")
				partes[0] += "=="
				return strings.Join(partes, ".")
			},
			espera: "no es base64url sin relleno",
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			i := nuevoIdP(t)
			v := verificadorDePrueba(t, i)
			ctx := context.Background()

			cab := map[string]any{"alg": "RS256", "kid": i.kid, "typ": "JWT"}
			cuerpo := i.cuerpoBueno(clientePrueba)
			if c.cab != nil {
				c.cab(i, cab)
			}
			if c.cuerpo != nil {
				c.cuerpo(cuerpo)
			}
			tok := i.acunar(t, cab, cuerpo)
			if c.romper != nil {
				tok = c.romper(t, i, tok)
			}

			_, err := v.Verificar(ctx, tok, ahoraFijo, esperadoPrueba)
			if err == nil {
				t.Fatalf("el token falsificado (%s) fue ACEPTADO. Esta falsificacion es "+
					"precisamente por donde entra cualquiera", c.nombre)
			}
			if !strings.Contains(err.Error(), c.espera) {
				t.Errorf("se rechazo, pero por otro motivo. Error: %v\nSe esperaba que "+
					"mencionara %q, porque si el motivo no es el que se cree, el test esta "+
					"vigilando otra cosa", err, c.espera)
			}

			// El control negativo, con el MISMO verificador y el MISMO
			// instante. Si esto falla, el caso de arriba no demuestra que se
			// rechace la falsificacion: demuestra que se rechaza todo.
			if _, err := v.Verificar(ctx, i.bueno(t, clientePrueba), ahoraFijo, esperadoPrueba); err != nil {
				t.Fatalf("CONTROL NEGATIVO EN ROJO: el token honrado tambien se rechaza (%v). "+
					"El caso de arriba no prueba nada", err)
			}
		})
	}
}

// TestUnTokenES256HonradoPasa existe para que el rechazo cruzado de curvas no
// se confunda con "no sabemos verificar curva eliptica".
func TestUnTokenES256HonradoPasa(t *testing.T) {
	i := nuevoIdP(t)
	v := verificadorDePrueba(t, i)
	tok := i.acunar(t, map[string]any{"alg": "ES256", "kid": i.kidEC, "typ": "JWT"}, i.cuerpoBueno(clientePrueba))
	if _, err := v.Verificar(context.Background(), tok, ahoraFijo, esperadoPrueba); err != nil {
		t.Fatalf("un ES256 bien firmado tiene que pasar y dio: %v", err)
	}
}

// TestElAlgoritmoLoDecideLaClaveNoElToken cierra el hueco que queda cuando el
// JWKS NO declara `alg` para su clave, que es legitimo y frecuente.
//
// Sin la comprobacion de compatibilidad entre algoritmo y tipo de clave, este
// caso se apoyaria en que la verificacion falle por casualidad. Aqui falla por
// decision y el mensaje lo dice.
func TestElAlgoritmoLoDecideLaClaveNoElToken(t *testing.T) {
	i := nuevoIdP(t)
	// Un JWKS con la clave RSA de verdad pero SIN declarar `alg`.
	sinAlg := jwkRSA(i.kid, &i.privRSA.PublicKey)
	delete(sinAlg, "alg")
	crudo, err := json.Marshal(map[string]any{"keys": []map[string]string{sinAlg}})
	if err != nil {
		t.Fatal(err)
	}
	i.mu.Lock()
	i.cuerpoJWKS = string(crudo)
	i.mu.Unlock()

	v := verificadorDePrueba(t, i)

	// Control negativo primero: con el JWKS sin `alg`, el token honrado pasa.
	if _, err := v.Verificar(context.Background(), i.bueno(t, clientePrueba), ahoraFijo, esperadoPrueba); err != nil {
		t.Fatalf("CONTROL NEGATIVO EN ROJO: con el JWKS sin `alg` el token honrado se "+
			"rechaza (%v)", err)
	}
	// Y ahora la falsificacion: el token dice ES256 y la clave es RSA.
	tok := i.acunar(t, map[string]any{"alg": "ES256", "kid": i.kid, "typ": "JWT"}, i.cuerpoBueno(clientePrueba))
	_, err = v.Verificar(context.Background(), tok, ahoraFijo, esperadoPrueba)
	if err == nil {
		t.Fatal("un token que dice ES256 verificado contra una clave RSA fue ACEPTADO")
	}
	if !strings.Contains(err.Error(), "casar con el tipo de la clave") {
		t.Errorf("se rechazo por otro motivo: %v", err)
	}
}

// TestElCruceContrarioTambienSeRechaza: clave de curva eliptica y token que
// dice RS256.
//
// HALLAZGO del barrido de mutacion. El test de arriba solo cubria un sentido
// del cruce (clave RSA, token que dice ES), asi que la rama simetrica de
// comprobarFirma se podia borrar entera sin que nada se pusiera rojo. Media
// comprobacion vigilada es media comprobacion.
func TestElCruceContrarioTambienSeRechaza(t *testing.T) {
	i := nuevoIdP(t)
	// Un JWKS con la clave EC de verdad, sin declarar `alg`, para que la
	// decision recaiga en la compatibilidad con el tipo de clave.
	sinAlg := jwkEC(i.kidEC, &i.privEC.PublicKey)
	delete(sinAlg, "alg")
	crudo, err := json.Marshal(map[string]any{"keys": []map[string]string{sinAlg}})
	if err != nil {
		t.Fatal(err)
	}
	i.mu.Lock()
	i.cuerpoJWKS = string(crudo)
	i.mu.Unlock()

	v := verificadorDePrueba(t, i)

	// Control negativo: con ese JWKS, un ES256 honrado pasa.
	bueno := i.acunar(t, map[string]any{"alg": "ES256", "kid": i.kidEC, "typ": "JWT"},
		i.cuerpoBueno(clientePrueba))
	if _, err := v.Verificar(context.Background(), bueno, ahoraFijo, esperadoPrueba); err != nil {
		t.Fatalf("CONTROL NEGATIVO EN ROJO: un ES256 honrado se rechaza: %v", err)
	}

	malo := i.acunar(t, map[string]any{"alg": "RS256", "kid": i.kidEC, "typ": "JWT"},
		i.cuerpoBueno(clientePrueba))
	_, err = v.Verificar(context.Background(), malo, ahoraFijo, esperadoPrueba)
	if err == nil {
		t.Fatal("un token que dice RS256 verificado contra una clave de curva eliptica fue " +
			"ACEPTADO")
	}
	if !strings.Contains(err.Error(), "casar con el tipo de la clave") {
		t.Errorf("se rechazo por otro motivo: %v", err)
	}
}

// TestElMargenDeRelojEsExplicitoYAcotado. Un margen grande es una caducidad
// grande, asi que no puede ser un numero que alguien suba sin darse cuenta.
func TestElMargenDeRelojEsExplicitoYAcotado(t *testing.T) {
	i := nuevoIdP(t)
	cfg := configPrueba(i)
	cfg.MargenReloj = 30 * time.Minute
	if _, err := NuevoVerificador(cfg, NuevasClaves("https://x.ejemplo/jwks", nil)); err == nil {
		t.Fatal("un margen de reloj de 30 minutos se acepto. Con el, un token robado sigue " +
			"valiendo media hora despues de caducar")
	}

	// Control negativo: dentro del tope, se acepta, y el margen se USA.
	cfg.MargenReloj = 2 * time.Minute
	v, err := NuevoVerificador(cfg, NuevasClaves(descubrirJWKS(t, i), ClientePorDefecto()))
	if err != nil {
		t.Fatalf("un margen de 2 minutos tiene que valer: %v", err)
	}
	cuerpo := i.cuerpoBueno(clientePrueba)
	// Caducado hace 90 segundos: fuera del margen por defecto (60 s) y dentro
	// del configurado (120 s).
	cuerpo["exp"] = ahoraFijo.Add(-90 * time.Second).Unix()
	cuerpo["iat"] = ahoraFijo.Add(-time.Hour).Unix()
	tok := i.acunar(t, map[string]any{"alg": "RS256", "kid": i.kid, "typ": "JWT"}, cuerpo)
	if _, err := v.Verificar(context.Background(), tok, ahoraFijo, esperadoPrueba); err != nil {
		t.Fatalf("con margen de 2 minutos, un token caducado hace 90 s tiene que pasar: %v", err)
	}
	// Y el mismo token con el margen por defecto NO pasa: prueba que el margen
	// es el que decide, no otra cosa.
	cfg.MargenReloj = 0
	vd, err := NuevoVerificador(cfg, NuevasClaves(descubrirJWKS(t, i), ClientePorDefecto()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := vd.Verificar(context.Background(), tok, ahoraFijo, esperadoPrueba); err == nil {
		t.Fatal("con el margen por defecto (60 s) un token caducado hace 90 s fue ACEPTADO")
	}
}

func descubrirJWKS(t *testing.T, i *idpFalso) string {
	t.Helper()
	prov, err := Descubrir(context.Background(), ClientePorDefecto(), i.emisor())
	if err != nil {
		t.Fatal(err)
	}
	return prov.JWKS
}

// TestElTokenNoSeLeeAntesDeVerificarLaFirma. Las reclamaciones de un token sin
// firma comprobada son datos del atacante. El orden importa y se comprueba
// mirando por QUE motivo se rechaza: si el motivo fuera de reclamacion, es que
// se leyeron antes.
func TestElTokenNoSeLeeAntesDeVerificarLaFirma(t *testing.T) {
	i := nuevoIdP(t)
	v := verificadorDePrueba(t, i)
	cuerpo := i.cuerpoBueno(clientePrueba)
	cuerpo["iss"] = "https://otro.ejemplo" // reclamacion mala
	cuerpo["exp"] = ahoraFijo.Add(-time.Hour).Unix()
	tok := i.acunar(t, map[string]any{"alg": "RS256", "kid": i.kid, "typ": "JWT"}, cuerpo)
	// Se estropea la firma ademas.
	partes := strings.Split(tok, ".")
	partes[2] = base64.RawURLEncoding.EncodeToString([]byte("firma-que-no-es"))
	_, err := v.Verificar(context.Background(), strings.Join(partes, "."), ahoraFijo, esperadoPrueba)
	if err == nil {
		t.Fatal("token con firma rota aceptado")
	}
	if strings.Contains(err.Error(), "el emisor del token es") || strings.Contains(err.Error(), "caduco a las") {
		t.Errorf("se rechazo por una RECLAMACION (%v), o sea que se leyo el cuerpo antes de "+
			"comprobar la firma. Leer `iss` o `exp` de un token sin firma comprobada es leer "+
			"datos del atacante", err)
	}
}

// TestUnTokenEnormeNoSeParsea: el orden barato antes que el caro.
func TestUnTokenEnormeNoSeParsea(t *testing.T) {
	i := nuevoIdP(t)
	v := verificadorDePrueba(t, i)
	enorme := strings.Repeat("A", LimiteTokenBytes+1)
	if _, err := v.Verificar(context.Background(), enorme, ahoraFijo, esperadoPrueba); err == nil {
		t.Fatal("un token de mas de 32 KiB se acepto para parsear")
	} else if !strings.Contains(err.Error(), "el maximo es") {
		t.Errorf("se rechazo por otro motivo: %v", err)
	}
	if _, err := v.Verificar(context.Background(), i.bueno(t, clientePrueba), ahoraFijo, esperadoPrueba); err != nil {
		t.Fatalf("CONTROL NEGATIVO EN ROJO: %v", err)
	}
}
