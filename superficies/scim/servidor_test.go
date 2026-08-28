package scim

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	adaptador "github.com/marcosmatalab/plazum/adaptadores/scim"
	"github.com/marcosmatalab/plazum/puertos"
)

var ahoraFijo = time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)

// El token de los tests. Es sintetico y no vale en ningun sitio; se escribe
// entero para que se vea que pasa el minimo de longitud.
const tokenPrueba = "token-de-aprovisionamiento-de-prueba-0001"

type secretosDeterministas struct {
	mu sync.Mutex
	n  uint64
}

func (s *secretosDeterministas) Bytes(b []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := 0; i < len(b); {
		s.n++
		var sem [8]byte
		binary.BigEndian.PutUint64(sem[:], s.n)
		h := sha256.Sum256(sem[:])
		i += copy(b[i:], h[:])
	}
	return nil
}

func (s *secretosDeterministas) Token(n int) (string, error) {
	b := make([]byte, n)
	if err := s.Bytes(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type banco struct {
	t   *testing.T
	srv *Servidor
	dir *adaptador.Directorio
	// reloj es mutable para poder mover el tiempo dentro de un test.
	reloj time.Time
}

func nuevoBanco(t *testing.T) *banco {
	t.Helper()
	dir, err := adaptador.NuevoDirectorio(&secretosDeterministas{})
	if err != nil {
		t.Fatal(err)
	}
	b := &banco{t: t, dir: dir, reloj: ahoraFijo}
	srv, err := Nuevo(dir, Opciones{
		Token: tokenPrueba,
		Ahora: func() time.Time { return b.reloj },
	})
	if err != nil {
		t.Fatal(err)
	}
	b.srv = srv
	return b
}

// pedir manda una peticion autenticada.
func (b *banco) pedir(metodo, ruta, cuerpo string) *httptest.ResponseRecorder {
	b.t.Helper()
	return b.pedirCon(metodo, ruta, cuerpo, "Bearer "+tokenPrueba)
}

func (b *banco) pedirCon(metodo, ruta, cuerpo, autorizacion string) *httptest.ResponseRecorder {
	b.t.Helper()
	var cuerpoLector *strings.Reader
	if cuerpo != "" {
		cuerpoLector = strings.NewReader(cuerpo)
	} else {
		cuerpoLector = strings.NewReader("")
	}
	r := httptest.NewRequest(metodo, ruta, cuerpoLector)
	if autorizacion != "" {
		r.Header.Set("Authorization", autorizacion)
	}
	r.Header.Set("Content-Type", adaptador.TipoContenido)
	w := httptest.NewRecorder()
	b.srv.ServeHTTP(w, r)
	return w
}

func (b *banco) json(w *httptest.ResponseRecorder) map[string]any {
	b.t.Helper()
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		b.t.Fatalf("la respuesta no es JSON (%d): %s", w.Code, w.Body.String())
	}
	return m
}

// ---------------------------------------------------------------------------
// La credencial
// ---------------------------------------------------------------------------

// TestSinCredencialNoSeHaceNada es la propiedad mas importante de este fichero.
//
// Este endpoint crea y borra usuarios. Si se pudiera llamar sin credencial,
// seria una puerta trasera con forma de estandar. Se recorren TODAS las rutas,
// no una de muestra: la que se olvide es exactamente la que se usara.
func TestSinCredencialNoSeHaceNada(t *testing.T) {
	b := nuevoBanco(t)
	u := b.crearUsuario("ana@ejemplo.es")

	rutas := []struct{ metodo, ruta, cuerpo string }{
		{"GET", "/scim/v2/Users", ""},
		{"POST", "/scim/v2/Users", `{"userName":"colado@ejemplo.es"}`},
		{"GET", "/scim/v2/Users/" + u, ""},
		{"PUT", "/scim/v2/Users/" + u, `{"userName":"otro@ejemplo.es"}`},
		{"PATCH", "/scim/v2/Users/" + u, `{"Operations":[{"op":"replace","path":"active","value":false}]}`},
		{"DELETE", "/scim/v2/Users/" + u, ""},
		{"GET", "/scim/v2/Groups", ""},
		{"POST", "/scim/v2/Groups", `{"displayName":"colados"}`},
		{"GET", "/scim/v2/Groups/x", ""},
		{"PATCH", "/scim/v2/Groups/x", `{"Operations":[]}`},
		{"DELETE", "/scim/v2/Groups/x", ""},
		{"GET", "/scim/v2/ServiceProviderConfig", ""},
		{"GET", "/scim/v2/ResourceTypes", ""},
		{"GET", "/scim/v2/Schemas", ""},
	}
	credenciales := []string{
		"",
		"Bearer ",
		"Bearer token-equivocado",
		"Bearer " + tokenPrueba + "x",
		"Bearer " + tokenPrueba[:len(tokenPrueba)-1],
		"Basic " + tokenPrueba,
		tokenPrueba,
	}
	for _, r := range rutas {
		for _, cred := range credenciales {
			w := b.pedirCon(r.metodo, r.ruta, r.cuerpo, cred)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("%s %s con la credencial %q respondio %d y tenia que ser 401. "+
					"Una sola ruta abierta en este endpoint es una puerta trasera",
					r.metodo, r.ruta, cred, w.Code)
			}
			if w.Header().Get("WWW-Authenticate") == "" {
				t.Errorf("%s %s: un 401 sin WWW-Authenticate deja al IdP sin saber que mandar",
					r.metodo, r.ruta)
			}
		}
	}
	// Y nada de aquello tuvo efecto.
	if _, err := b.dir.Leer(u); err != nil {
		t.Fatal("el usuario se borro con una peticion sin credencial")
	}
	if _, total, _ := b.dir.Listar(adaptador.Filtro{}); total != 1 {
		t.Fatalf("hay %d usuarios: alguna peticion sin credencial creo algo", total)
	}

	// CONTROL NEGATIVO: con la credencial buena, esas mismas rutas responden.
	if w := b.pedir("GET", "/scim/v2/Users", ""); w.Code != http.StatusOK {
		t.Fatalf("CONTROL NEGATIVO EN ROJO: con la credencial correcta tambien se rechaza "+
			"(%d: %s). Entonces el test de arriba no prueba nada", w.Code, w.Body.String())
	}
}

// TestNoSePuedeConstruirUnServidorAbierto. La defensa estructural: aunque
// alguien deje el token vacio en un fichero de configuracion, no arranca.
func TestNoSePuedeConstruirUnServidorAbierto(t *testing.T) {
	dir, err := adaptador.NuevoDirectorio(&secretosDeterministas{})
	if err != nil {
		t.Fatal(err)
	}
	// Cada caso exige ADEMAS el mensaje que le toca. HALLAZGO del barrido de
	// mutacion: la comprobacion de token vacio se podia borrar entera y el
	// caso seguia rechazandose por la de longitud, con un mensaje que le dice
	// al administrador que su token es corto cuando lo que pasa es que no ha
	// pegado ninguno. Rechazar por el motivo equivocado le manda a buscar
	// donde no es.
	casos := []struct{ token, pista string }{
		{"", "falta el token de aprovisionamiento"},
		{"   ", "falta el token de aprovisionamiento"},
		{"corto", "el minimo es"},
		{"1234567890123456789012345678901", "el minimo es"},
	}
	for _, c := range casos {
		_, err := Nuevo(dir, Opciones{Token: c.token})
		if err == nil {
			t.Fatalf("se construyo un servidor SCIM con el token %q. No puede haber forma de "+
				"construir uno abierto, ni por descuido", c.token)
		}
		if !strings.Contains(err.Error(), c.pista) {
			t.Errorf("con el token %q el error dice %q y se esperaba que mencionara %q",
				c.token, err, c.pista)
		}
	}
	if _, err := Nuevo(nil, Opciones{Token: tokenPrueba}); err == nil {
		t.Fatal("se construyo un servidor sin directorio")
	}
	// Control negativo: con un token de longitud suficiente si construye.
	if _, err := Nuevo(dir, Opciones{Token: tokenPrueba}); err != nil {
		t.Fatalf("CONTROL NEGATIVO EN ROJO: %v", err)
	}
}

// TestElTokenNoSaleEnNingunaRespuestaNiEnLaActividad.
func TestElTokenNoSaleEnNingunaRespuestaNiEnLaActividad(t *testing.T) {
	b := nuevoBanco(t)
	const intentado = "TOKEN-QUE-ALGUIEN-PROBO-Y-NO-PUEDE-QUEDAR-REGISTRADO"
	w := b.pedirCon("GET", "/scim/v2/Users", "", "Bearer "+intentado)
	if strings.Contains(w.Body.String(), intentado) {
		t.Errorf("el token presentado vuelve en el cuerpo de la respuesta: %s", w.Body.String())
	}
	if strings.Contains(w.Body.String(), tokenPrueba) {
		t.Errorf("el token BUENO sale en la respuesta de error: %s", w.Body.String())
	}
	act := b.srv.Actividad()
	if strings.Contains(act.UltimoRechazo, intentado) || strings.Contains(act.UltimoRechazo, tokenPrueba) {
		t.Errorf("hay un token en la actividad, que es lo que se ensena en pantalla: %q",
			act.UltimoRechazo)
	}
	if act.RechazosDeCredencial != 1 {
		t.Errorf("el rechazo de credencial no se conto: %+v", act)
	}
}

// ---------------------------------------------------------------------------
// El ciclo de vida por HTTP
// ---------------------------------------------------------------------------

func (b *banco) crearUsuario(userName string) string {
	b.t.Helper()
	w := b.pedir("POST", "/scim/v2/Users", `{
		"schemas":["`+adaptador.EsquemaUsuario+`"],
		"userName":"`+userName+`",
		"externalId":"ext-`+userName+`",
		"name":{"givenName":"Nombre","familyName":"Apellido"},
		"displayName":"`+userName+`",
		"emails":[{"value":"`+userName+`","type":"work","primary":true}],
		"active":true
	}`)
	if w.Code != http.StatusCreated {
		b.t.Fatalf("crear %s: %d %s", userName, w.Code, w.Body.String())
	}
	m := b.json(w)
	id, _ := m["id"].(string)
	if id == "" {
		b.t.Fatalf("el recurso creado no trae id: %s", w.Body.String())
	}
	return id
}

// TestElCicloDeVidaPorHTTP recorre lo que hace un IdP de verdad.
func TestElCicloDeVidaPorHTTP(t *testing.T) {
	b := nuevoBanco(t)

	w := b.pedir("POST", "/scim/v2/Users", `{
		"schemas":["`+adaptador.EsquemaUsuario+`"],
		"userName":"ana@ejemplo.es","active":true
	}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST: %d %s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); ct != adaptador.TipoContenido {
		t.Errorf("Content-Type %q, se esperaba %q", ct, adaptador.TipoContenido)
	}
	if w.Header().Get("Location") == "" {
		t.Error("un 201 sin Location deja al IdP sin saber donde quedo el recurso")
	}
	m := b.json(w)
	id := m["id"].(string)
	if esquemas, ok := m["schemas"].([]any); !ok || len(esquemas) == 0 ||
		esquemas[0] != adaptador.EsquemaUsuario {
		t.Errorf("el recurso no declara su esquema: %v", m["schemas"])
	}
	if meta, ok := m["meta"].(map[string]any); !ok || meta["resourceType"] != "User" {
		t.Errorf("falta meta.resourceType: %v", m["meta"])
	}

	// El alta duplicada da 409, que es lo que el IdP sabe interpretar.
	w = b.pedir("POST", "/scim/v2/Users", `{"userName":"ana@ejemplo.es","active":true}`)
	if w.Code != http.StatusConflict {
		t.Errorf("el alta duplicada dio %d y tenia que ser 409", w.Code)
	}
	if b.json(w)["scimType"] != "uniqueness" {
		t.Errorf("scimType %v, se esperaba uniqueness", b.json(w)["scimType"])
	}

	// El IdP comprueba si existe con un filtro. Esta es LA consulta del
	// aprovisionamiento.
	w = b.pedir("GET", `/scim/v2/Users?filter=userName%20eq%20%22ana@ejemplo.es%22`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("filtrar: %d %s", w.Code, w.Body.String())
	}
	lista := b.json(w)
	if lista["totalResults"].(float64) != 1 {
		t.Fatalf("el filtro devolvio %v resultados: %s", lista["totalResults"], w.Body.String())
	}

	// El GET por id devuelve a ESE usuario. Parece trivial y es lo que
	// destapo que el despacho no rellenaba los comodines de la ruta: los tests
	// que esperaban 404 seguian en verde con el motivo equivocado.
	w = b.pedir("GET", "/scim/v2/Users/"+id, "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET por id: %d %s", w.Code, w.Body.String())
	}
	if leido := b.json(w); leido["id"] != id || leido["userName"] != "ana@ejemplo.es" {
		t.Fatalf("el GET por id devolvio otra cosa: %s", w.Body.String())
	}

	// Desactivar por PATCH.
	w = b.pedir("PATCH", "/scim/v2/Users/"+id,
		`{"schemas":["`+adaptador.EsquemaParcheo+`"],"Operations":[{"op":"replace","value":{"active":false}}]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH: %d %s", w.Code, w.Body.String())
	}
	if b.json(w)["active"] != false {
		t.Error("el PATCH no desactivo")
	}

	// Borrar.
	w = b.pedir("DELETE", "/scim/v2/Users/"+id, "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("DELETE: %d %s", w.Code, w.Body.String())
	}
	w = b.pedir("GET", "/scim/v2/Users/"+id, "")
	if w.Code != http.StatusNotFound {
		t.Errorf("tras el DELETE, el GET dio %d y tenia que ser 404", w.Code)
	}
}

// TestElManagerViajaEnLaExtensionEnterprise. Es el atributo del que sale el
// escalado: si no viaja bien, la etapa 4 no tiene jerarquia.
func TestElManagerViajaEnLaExtensionEnterprise(t *testing.T) {
	b := nuevoBanco(t)
	jefa := b.crearUsuario("jefa@ejemplo.es")

	w := b.pedir("POST", "/scim/v2/Users", `{
		"schemas":["`+adaptador.EsquemaUsuario+`","`+adaptador.EsquemaEmpresa+`"],
		"userName":"ana@ejemplo.es","active":true,
		"`+adaptador.EsquemaEmpresa+`":{
			"employeeNumber":"0042","department":"Seguridad",
			"manager":{"value":"`+jefa+`"}
		}
	}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("crear con manager: %d %s", w.Code, w.Body.String())
	}
	m := b.json(w)
	ext, ok := m[adaptador.EsquemaEmpresa].(map[string]any)
	if !ok {
		t.Fatalf("la respuesta no trae la extension enterprise: %s", w.Body.String())
	}
	mgr, ok := ext["manager"].(map[string]any)
	if !ok || mgr["value"] != jefa {
		t.Fatalf("el manager no volvio bien: %v", ext["manager"])
	}
	if ext["department"] != "Seguridad" || ext["employeeNumber"] != "0042" {
		t.Errorf("la extension perdio campos: %v", ext)
	}
	// Y los esquemas declarados incluyen la extension.
	esquemas, _ := m["schemas"].([]any)
	var declara bool
	for _, e := range esquemas {
		if e == adaptador.EsquemaEmpresa {
			declara = true
		}
	}
	if !declara {
		t.Errorf("el recurso trae la extension y no la declara en `schemas`: %v", esquemas)
	}
}

// TestUnPatchQueCierraUnCicloDevuelve400. El IdP tiene que ENTERARSE.
func TestUnPatchQueCierraUnCicloDevuelve400(t *testing.T) {
	b := nuevoBanco(t)
	a := b.crearUsuario("a@ejemplo.es")
	c := b.crearUsuario("c@ejemplo.es")

	w := b.pedir("PATCH", "/scim/v2/Users/"+a,
		`{"Operations":[{"op":"replace","path":"manager","value":{"value":"`+c+`"}}]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("CONTROL NEGATIVO EN ROJO: el primer manager tiene que entrar: %d %s",
			w.Code, w.Body.String())
	}
	w = b.pedir("PATCH", "/scim/v2/Users/"+c,
		`{"Operations":[{"op":"replace","path":"manager","value":{"value":"`+a+`"}}]}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("cerrar un ciclo dio %d y tenia que ser 400: si el IdP recibe un 200 cree "+
			"que su dato entro y el escalado se queda colgado sin que nadie se entere", w.Code)
	}
	if !strings.Contains(w.Body.String(), "ciclo") {
		t.Errorf("el cuerpo no explica que es un ciclo: %s", w.Body.String())
	}
}

// TestUnCuerpoEnormeSeCorta. Un cuerpo sin limite es una fuga de memoria por
// una ruta que exige credencial pero que tiene que leer antes de decir que no.
func TestUnCuerpoEnormeSeCorta(t *testing.T) {
	dir, err := adaptador.NuevoDirectorio(&secretosDeterministas{})
	if err != nil {
		t.Fatal(err)
	}
	srv, err := Nuevo(dir, Opciones{Token: tokenPrueba, MaxCuerpo: 512,
		Ahora: func() time.Time { return ahoraFijo }})
	if err != nil {
		t.Fatal(err)
	}
	cuerpo := `{"userName":"ana@ejemplo.es","displayName":"` + strings.Repeat("A", 2000) + `"}`
	r := httptest.NewRequest("POST", "/scim/v2/Users", strings.NewReader(cuerpo))
	r.Header.Set("Authorization", "Bearer "+tokenPrueba)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Code == http.StatusCreated {
		t.Fatal("un cuerpo de 2 KiB con el limite en 512 bytes se acepto entero")
	}
	if _, total, _ := dir.Listar(adaptador.Filtro{}); total != 0 {
		t.Fatal("se creo el usuario pese a cortarse el cuerpo")
	}
}

// TestUnaRutaDesconocidaRespondeEnFormatoSCIM. El IdP parsea la respuesta: un
// 404 en texto plano le sale por una traza que nadie entiende.
func TestUnaRutaDesconocidaRespondeEnFormatoSCIM(t *testing.T) {
	b := nuevoBanco(t)
	w := b.pedir("GET", "/scim/v2/LoQueSea", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("codigo %d", w.Code)
	}
	m := b.json(w)
	esquemas, ok := m["schemas"].([]any)
	if !ok || len(esquemas) == 0 || esquemas[0] != adaptador.EsquemaError {
		t.Errorf("el 404 no viene en el formato de error de SCIM: %s", w.Body.String())
	}
}

// TestElFiltroQueNoSeEntiendeNoDevuelveLaListaEntera, por HTTP.
func TestElFiltroQueNoSeEntiendeNoDevuelveLaListaEntera(t *testing.T) {
	b := nuevoBanco(t)
	b.crearUsuario("ana@ejemplo.es")
	b.crearUsuario("luis@ejemplo.es")

	for _, filtro := range []string{
		`userName%20co%20%22ana%22`,
		`userName%20eq%20%22ana@ejemplo.es%22%20and%20active%20eq%20true`,
		`(userName%20eq%20%22ana@ejemplo.es%22)`,
		`activo`,
	} {
		w := b.pedir("GET", "/scim/v2/Users?filter="+filtro, "")
		if w.Code == http.StatusOK {
			t.Errorf("el filtro %q devolvio 200. Un filtro ignorado devuelve la lista entera "+
				"y hace que el IdP concluya lo contrario de lo que pregunto: entonces duplica "+
				"usuarios o deja de crearlos", filtro)
		}
	}
	// Control negativo: el filtro que si se entiende funciona.
	w := b.pedir("GET", `/scim/v2/Users?filter=userName%20eq%20%22ana@ejemplo.es%22`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("CONTROL NEGATIVO EN ROJO: %d %s", w.Code, w.Body.String())
	}
}

// TestElServiceProviderConfigDiceLaVerdad. Declarar capacidades que no se
// tienen es peor que no declararlas: el IdP se fia y falla a medio ciclo.
func TestElServiceProviderConfigDiceLaVerdad(t *testing.T) {
	b := nuevoBanco(t)
	w := b.pedir("GET", "/scim/v2/ServiceProviderConfig", "")
	if w.Code != http.StatusOK {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	m := b.json(w)
	for clave, esperado := range map[string]bool{
		"patch": true, "bulk": false, "sort": false, "changePassword": false,
	} {
		sub, ok := m[clave].(map[string]any)
		if !ok {
			t.Fatalf("falta %q en ServiceProviderConfig", clave)
		}
		if sub["supported"] != esperado {
			t.Errorf("%q dice supported=%v y la verdad es %v", clave, sub["supported"], esperado)
		}
	}
}

// ---------------------------------------------------------------------------
// El diagnostico: la pregunta del comprador
// ---------------------------------------------------------------------------

// TestElDiagnosticoDistingueNuncaConectadoDeFuncionando.
//
// Es la pregunta del comprador: "¿esta funcionando el SCIM o solo lo espero?".
// Un aprovisionamiento que nunca se conecto y uno que funciona se parecen mucho
// desde fuera: en los dos casos no pasa nada raro.
func TestElDiagnosticoDistingueNuncaConectadoDeFuncionando(t *testing.T) {
	b := nuevoBanco(t)

	c := buscar(t, b.srv.Comprobaciones(ahoraFijo), "scim-conexion")
	if c.Estado != puertos.Aviso || !strings.Contains(c.Detalle, "todavia no ha llegado") {
		t.Fatalf("con el IdP sin conectar nunca, el diagnostico dice: %+v", c)
	}
	if c.Arreglo == "" {
		t.Error("un problema sin arreglo escrito le pasa el trabajo al operador")
	}

	b.crearUsuario("ana@ejemplo.es")
	c = buscar(t, b.srv.Comprobaciones(ahoraFijo), "scim-conexion")
	if c.Estado != puertos.Correcto {
		t.Fatalf("tras una peticion correcta, el diagnostico dice: %+v", c)
	}

	// Y si el IdP calla mas de un dia, se pone en rojo: un aprovisionamiento
	// parado significa que quien salio conserva el acceso.
	c = buscar(t, b.srv.Comprobaciones(ahoraFijo.Add(48*time.Hour)), "scim-conexion")
	if c.Estado != puertos.Roto {
		t.Fatalf("tras 48 horas de silencio, el diagnostico dice: %+v", c)
	}
	if c.Arreglo == "" {
		t.Error("sin arreglo escrito")
	}
}

// TestElDiagnosticoDiceQueNadieTraeManager. Es lo que ve el comprador cuando su
// IdP no publica el atributo, que es la mitad de los casos.
func TestElDiagnosticoDiceQueNadieTraeManager(t *testing.T) {
	b := nuevoBanco(t)
	b.crearUsuario("ana@ejemplo.es")
	b.crearUsuario("luis@ejemplo.es")

	c := buscar(t, b.srv.Comprobaciones(ahoraFijo), "scim-jerarquia")
	if c.Estado != puertos.Aviso {
		t.Fatalf("con nadie trayendo manager, el diagnostico dice: %+v", c)
	}
	if !strings.Contains(c.Detalle, "escalado no tiene a donde subir") {
		t.Errorf("el detalle no explica la consecuencia: %q", c.Detalle)
	}
	if !strings.Contains(c.Arreglo, "a mano") {
		t.Errorf("el arreglo no menciona la alternativa manual, que es para lo que existe: %q",
			c.Arreglo)
	}

	// Control negativo: con la jerarquia completa, se pone en verde.
	us, _, err := b.dir.Listar(adaptador.Filtro{})
	if err != nil {
		t.Fatal(err)
	}
	if err := b.dir.FijarManagerManual(us[0].ID, us[1].ID, "operador", ahoraFijo); err != nil {
		t.Fatal(err)
	}
	c = buscar(t, b.srv.Comprobaciones(ahoraFijo), "scim-jerarquia")
	if c.Estado == puertos.Correcto {
		t.Fatal("con uno de dos sin jefe no puede estar en verde")
	}
	if !strings.Contains(c.Detalle, "1 de 2") {
		t.Errorf("no se dice cuantos faltan: %q", c.Detalle)
	}
}

// TestTodaComprobacionQueNoEstaCorrectaDiceComoSeArregla. Es la exigencia de la
// suite de contrato del puerto Diagnostico, aplicada a lo que aporta el SCIM.
func TestTodaComprobacionQueNoEstaCorrectaDiceComoSeArregla(t *testing.T) {
	b := nuevoBanco(t)
	b.crearUsuario("ana@ejemplo.es")
	b.pedirCon("GET", "/scim/v2/Users", "", "Bearer mal")

	for _, ahora := range []time.Time{ahoraFijo, ahoraFijo.Add(72 * time.Hour)} {
		for _, c := range b.srv.Comprobaciones(ahora) {
			if strings.TrimSpace(c.Nombre) == "" {
				t.Errorf("comprobacion sin nombre: %+v", c)
			}
			if strings.TrimSpace(c.Detalle) == "" {
				t.Errorf("%q sin detalle", c.Nombre)
			}
			if c.Estado != puertos.Correcto && strings.TrimSpace(c.Arreglo) == "" {
				t.Errorf("la comprobacion %q esta en %s y no dice como se arregla",
					c.Nombre, c.Estado)
			}
		}
	}
}

func buscar(t *testing.T, cs []puertos.Comprobacion, nombre string) puertos.Comprobacion {
	t.Helper()
	for _, c := range cs {
		if c.Nombre == nombre {
			return c
		}
	}
	t.Fatalf("no hay ninguna comprobacion llamada %q en %+v", nombre, cs)
	return puertos.Comprobacion{}
}

// ---------------------------------------------------------------------------
// Grupos
// ---------------------------------------------------------------------------

// TestElCicloDeVidaDeUnGrupo, con el PATCH de miembros que mandan los IdP.
func TestElCicloDeVidaDeUnGrupo(t *testing.T) {
	b := nuevoBanco(t)
	ana := b.crearUsuario("ana@ejemplo.es")
	luis := b.crearUsuario("luis@ejemplo.es")

	w := b.pedir("POST", "/scim/v2/Groups", `{
		"schemas":["`+adaptador.EsquemaGrupo+`"],"displayName":"Comite de seguridad"
	}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("crear grupo: %d %s", w.Code, w.Body.String())
	}
	id := b.json(w)["id"].(string)

	w = b.pedir("PATCH", "/scim/v2/Groups/"+id,
		`{"Operations":[{"op":"add","path":"members","value":[{"value":"`+ana+`"},{"value":"`+luis+`"}]}]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("anadir miembros: %d %s", w.Code, w.Body.String())
	}
	if ms, ok := b.json(w)["members"].([]any); !ok || len(ms) != 2 {
		t.Fatalf("el grupo tiene %v miembros", b.json(w)["members"])
	}

	// El remove con filtro, que es como lo manda Okta.
	w = b.pedir("PATCH", "/scim/v2/Groups/"+id,
		`{"Operations":[{"op":"remove","path":"members[value eq \"`+ana+`\"]"}]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("quitar un miembro: %d %s", w.Code, w.Body.String())
	}
	ms, _ := b.json(w)["members"].([]any)
	if len(ms) != 1 {
		t.Fatalf("tras quitar a uno quedan %d miembros: quitar a UNO y vaciar el grupo no "+
			"son lo mismo ni de lejos", len(ms))
	}
	if m := ms[0].(map[string]any); m["value"] != luis {
		t.Errorf("se quito al que no era: queda %v", m["value"])
	}

	// Un miembro que no existe se rechaza.
	w = b.pedir("PATCH", "/scim/v2/Groups/"+id,
		`{"Operations":[{"op":"add","path":"members","value":[{"value":"no-existe"}]}]}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("anadir un miembro inexistente dio %d", w.Code)
	}

	// Y al borrar un usuario, sale del grupo.
	if w := b.pedir("DELETE", "/scim/v2/Users/"+luis, ""); w.Code != http.StatusNoContent {
		t.Fatal(w.Code)
	}
	w = b.pedir("GET", "/scim/v2/Groups/"+id, "")
	if ms, _ := b.json(w)["members"].([]any); len(ms) != 0 {
		t.Errorf("un usuario borrado sigue en el grupo: %v", ms)
	}
}

// TestUnPOSTNoPuedeColarLoQueElPATCHRechaza. Si el alta admitiera `roles`,
// bastaria borrar y volver a crear para saltarse la lista blanca del PATCH.
func TestUnPOSTNoPuedeColarLoQueElPATCHRechaza(t *testing.T) {
	b := nuevoBanco(t)
	for _, cuerpo := range []string{
		`{"userName":"colado@ejemplo.es","roles":[{"value":"admin"}]}`,
		`{"userName":"colado@ejemplo.es","password":"hunter2"}`,
		`{"userName":"colado@ejemplo.es","entitlements":[{"value":"todo"}]}`,
	} {
		w := b.pedir("POST", "/scim/v2/Users", cuerpo)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("el alta con %s dio %d. Si el POST admite lo que el PATCH rechaza, "+
				"basta borrar y volver a crear para saltarse la lista blanca", cuerpo, w.Code)
		}
	}
	// Control negativo: el alta honrada entra.
	if w := b.pedir("POST", "/scim/v2/Users", `{"userName":"ana@ejemplo.es"}`); w.Code != http.StatusCreated {
		t.Fatalf("CONTROL NEGATIVO EN ROJO: %d %s", w.Code, w.Body.String())
	}
}

// Ninguna respuesta SCIM sale sin nosniff, y esto recorre los CUATRO caminos de
// salida, no uno.
//
// De donde viene. Lo encontro gosec (G705, XSS por analisis de contaminacion)
// senalando el `w.Write(cuerpo)` del servidor, y la tentacion inmediata fue
// ponerle un `#nosec` y seguir: al fin y al cabo esto sirve
// "application/scim+json", que ningun navegador trata como HTML.
//
// No era un falso positivo. El middleware de seguridad de superficies/serve NO
// cubre todavia /scim/v2 (P1 20 de docs/pendientes.md), asi que estas eran las
// unicas respuestas del producto que salian sin nosniff. Y el cuerpo lleva texto
// de un tercero: el nombre de un usuario lo pone el proveedor de identidad.
//
// El test va por los cuatro caminos porque anadir la cabecera en el sitio comun
// y dejarse uno es la forma natural de equivocarse aqui: exito con cuerpo, exito
// con Location, error de dominio y ruta desconocida salen por funciones
// distintas.
func TestNingunaRespuestaSCIMSaleSinNosniff(t *testing.T) {
	b := nuevoBanco(t)

	casos := []struct {
		nombre         string
		metodo, ruta   string
		cuerpo         string
		autorizacion   string
		estadoEsperado int
	}{
		{"creacion, que responde con Location", "POST", "/scim/v2/Users",
			`{"schemas":["urn:ietf:params:scim:schemas:core:2.0:User"],` +
				`"userName":"nosniff@ejemplo.invalid","active":true}`,
			"Bearer " + tokenPrueba, http.StatusCreated},
		{"lectura, que responde con cuerpo y sin Location", "GET",
			"/scim/v2/ServiceProviderConfig", "", "Bearer " + tokenPrueba, http.StatusOK},
		{"error de dominio", "GET", "/scim/v2/Users/no-existe", "",
			"Bearer " + tokenPrueba, http.StatusNotFound},
		{"ruta desconocida", "GET", "/scim/v2/Cualquiera", "",
			"Bearer " + tokenPrueba, http.StatusNotFound},
		{"sin credencial", "GET", "/scim/v2/Users", "", "", http.StatusUnauthorized},
	}

	for _, c := range casos {
		w := b.pedirCon(c.metodo, c.ruta, c.cuerpo, c.autorizacion)
		if w.Code != c.estadoEsperado {
			t.Errorf("%s: estado %d y se esperaba %d (el caso ya no prueba lo que decia)",
				c.nombre, w.Code, c.estadoEsperado)
			continue
		}
		if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("%s (%s %s): X-Content-Type-Options es %q y tiene que ser \"nosniff\". "+
				"El middleware de serve no cubre /scim/v2, asi que esta cabecera la pone "+
				"quien escribe el cuerpo o no la pone nadie",
				c.nombre, c.metodo, c.ruta, got)
		}
		if got := w.Header().Get("Content-Type"); got != adaptador.TipoContenido {
			t.Errorf("%s: Content-Type es %q y tiene que ser %q",
				c.nombre, got, adaptador.TipoContenido)
		}
	}
}

// CONTROL NEGATIVO del test de arriba. Si el arnes no supiera leer cabeceras, el
// test anterior pasaria con el servidor desnudo y no probaria nada.
func TestElArnesDeCabecerasDistingueUnaRespuestaSinNosniff(t *testing.T) {
	w := httptest.NewRecorder()
	w.WriteHeader(http.StatusOK)
	if got := w.Header().Get("X-Content-Type-Options"); got == "nosniff" {
		t.Fatal("una respuesta a la que nadie le ha puesto la cabecera no puede traerla: " +
			"el arnes esta mirando otra cosa")
	}
}
