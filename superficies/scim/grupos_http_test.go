package scim

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	adaptador "plazum/adaptadores/scim"
)

// La mitad de la superficie HTTP que no se ejercitaba: PUT, los grupos enteros
// y los tres endpoints de descubrimiento.
//
// POR QUE IMPORTA MAS QUE UN NUMERO DE COBERTURA. Esto es superficie de red
// AUTENTICADA que da de alta y de baja a personas: es la que se lleva un CVE, y
// es por donde un IdP mal configurado hace daño de verdad. Un endpoint sin test
// no es codigo sin probar, es una puerta que nadie ha intentado abrir.

// ---------------------------------------------------------------------------
// PUT
// ---------------------------------------------------------------------------

func TestElPutPorHTTPSustituyeYNoResucita(t *testing.T) {
	b := nuevoBanco(t)
	id := b.crearUsuario("ana")

	w := b.pedir("PUT", "/scim/v2/Users/"+id, `{
		"schemas":["`+adaptador.EsquemaUsuario+`"],
		"userName":"ana","displayName":"Ana Garcia","active":false}`)
	if w.Code != http.StatusOK {
		t.Fatalf("un PUT legitimo tenia que salir 200 y salio %d: %s", w.Code, w.Body.String())
	}
	m := b.json(w)
	if m["displayName"] != "Ana Garcia" || m["active"] != false {
		t.Fatalf("el PUT no ha sustituido: %+v", m)
	}
	// Lo que el PUT no manda desaparece: eso es un PUT y no un PATCH.
	if _, hay := m["emails"]; hay {
		t.Errorf("el PUT no traia emails y siguen ahi: %+v", m["emails"])
	}
	if m["id"] != id {
		t.Fatal("el PUT ha cambiado el id del recurso: toda referencia se rompe")
	}

	// Un cuerpo que no es JSON no se interpreta a medias.
	if w := b.pedir("PUT", "/scim/v2/Users/"+id, `{`); w.Code != http.StatusBadRequest {
		t.Errorf("un PUT con el cuerpo roto tenia que salir 400 y salio %d", w.Code)
	}
	// Y sobre un id que no existe, 404 en formato SCIM.
	w = b.pedir("PUT", "/scim/v2/Users/no-existe", `{
		"schemas":["`+adaptador.EsquemaUsuario+`"],"userName":"x"}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("un PUT sobre un id inventado tenia que salir 404 y salio %d", w.Code)
	}

	// EL QUE IMPORTA: sobre un usuario BORRADO no se resucita nada. Para SCIM
	// ese recurso ya no existe, y un PUT que lo devolviera a la vida seria un
	// alta que no ha pasado por el alta.
	if w := b.pedir("DELETE", "/scim/v2/Users/"+id, ""); w.Code != http.StatusNoContent {
		t.Fatalf("el borrado previo no ha funcionado: %d", w.Code)
	}
	w = b.pedir("PUT", "/scim/v2/Users/"+id, `{
		"schemas":["`+adaptador.EsquemaUsuario+`"],"userName":"ana","active":true}`)
	if w.Code != http.StatusNotFound {
		t.Fatalf("HALLAZGO: un PUT sobre un usuario borrado ha salido %d en vez de 404. "+
			"Eso resucita una cuenta sin pasar por el alta: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Los grupos, por HTTP
// ---------------------------------------------------------------------------

func (b *banco) crearGrupo(nombre string, miembros ...string) string {
	b.t.Helper()
	var refs []string
	for _, m := range miembros {
		refs = append(refs, `{"value":"`+m+`"}`)
	}
	w := b.pedir("POST", "/scim/v2/Groups", `{
		"schemas":["`+adaptador.EsquemaGrupo+`"],
		"displayName":"`+nombre+`",
		"externalId":"ext-`+nombre+`",
		"members":[`+strings.Join(refs, ",")+`]}`)
	if w.Code != http.StatusCreated {
		b.t.Fatalf("crear el grupo %s: %d %s", nombre, w.Code, w.Body.String())
	}
	m := b.json(w)
	id, _ := m["id"].(string)
	if id == "" {
		b.t.Fatalf("el grupo creado no trae id: %s", w.Body.String())
	}
	return id
}

func TestElCicloDeVidaDeUnGrupoPorHTTP(t *testing.T) {
	b := nuevoBanco(t)
	ana := b.crearUsuario("ana")
	luis := b.crearUsuario("luis")

	id := b.crearGrupo("Seguridad", ana, luis)

	// GET del grupo, con su Location y su ETag: es lo que usa el IdP para no
	// reescribir lo que no ha cambiado.
	w := b.pedir("GET", "/scim/v2/Groups/"+id, "")
	if w.Code != http.StatusOK {
		t.Fatalf("leer el grupo: %d %s", w.Code, w.Body.String())
	}
	m := b.json(w)
	if m["displayName"] != "Seguridad" {
		t.Fatalf("el grupo leido no es el creado: %+v", m)
	}
	miembros, _ := m["members"].([]any)
	if len(miembros) != 2 {
		t.Fatalf("dos miembros y salen %d: %+v", len(miembros), m["members"])
	}

	// LIST, con su envoltorio de ListResponse.
	w = b.pedir("GET", "/scim/v2/Groups", "")
	if w.Code != http.StatusOK {
		t.Fatalf("listar grupos: %d %s", w.Code, w.Body.String())
	}
	lista := b.json(w)
	if lista["totalResults"] != float64(1) {
		t.Fatalf("un grupo y totalResults dice %v", lista["totalResults"])
	}

	// LIST con filtro por displayName, que es como un IdP comprueba si ya
	// existe antes de crear.
	w = b.pedir("GET", `/scim/v2/Groups?filter=displayName+eq+"Seguridad"`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("filtrar grupos: %d %s", w.Code, w.Body.String())
	}
	if b.json(w)["totalResults"] != float64(1) {
		t.Fatalf("el filtro por displayName no encuentra el grupo: %s", w.Body.String())
	}

	// Y uno que no existe da CERO, no la lista entera. Si diera la lista
	// entera, el IdP concluiria que ya existe y no lo crearia nunca.
	w = b.pedir("GET", `/scim/v2/Groups?filter=displayName+eq+"NoExiste"`, "")
	if w.Code != http.StatusOK || b.json(w)["totalResults"] != float64(0) {
		t.Fatalf("HALLAZGO: filtrar por un grupo que no existe no da cero: %d %s",
			w.Code, w.Body.String())
	}

	// PATCH: quitar a uno.
	w = b.pedir("PATCH", "/scim/v2/Groups/"+id, `{
		"schemas":["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
		"Operations":[{"op":"remove","path":"members[value eq \"`+luis+`\"]"}]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("el PATCH del grupo: %d %s", w.Code, w.Body.String())
	}
	miembros, _ = b.json(w)["members"].([]any)
	if len(miembros) != 1 {
		t.Fatalf("tenia que quedar uno y quedan %d", len(miembros))
	}

	// DELETE, y el 404 despues: un grupo borrado no vuelve.
	if w := b.pedir("DELETE", "/scim/v2/Groups/"+id, ""); w.Code != http.StatusNoContent {
		t.Fatalf("borrar el grupo tenia que salir 204 y salio %d: %s", w.Code, w.Body.String())
	}
	if w := b.pedir("GET", "/scim/v2/Groups/"+id, ""); w.Code != http.StatusNotFound {
		t.Fatalf("un grupo borrado sigue devolviendose: %d", w.Code)
	}
	// CONTROL NEGATIVO: borrar dos veces da 404, no 204. Un DELETE que siempre
	// dice que si deja al IdP creyendo que borro algo que no estaba.
	if w := b.pedir("DELETE", "/scim/v2/Groups/"+id, ""); w.Code != http.StatusNotFound {
		t.Fatalf("borrar dos veces el mismo grupo ha salido %d", w.Code)
	}
	// Y los usuarios siguen vivos: borrar un grupo no da de baja a nadie.
	if w := b.pedir("GET", "/scim/v2/Users/"+ana, ""); w.Code != http.StatusOK {
		t.Fatalf("borrar un grupo ha dado de baja a un usuario: %d", w.Code)
	}
}

func TestElGrupoRechazaLoQueNoDebeAceptar(t *testing.T) {
	b := nuevoBanco(t)
	ana := b.crearUsuario("ana")
	id := b.crearGrupo("Seguridad", ana)

	casos := []struct {
		nombre, metodo, ruta, cuerpo string
		espera                       int
	}{
		{"un cuerpo que no es JSON", "POST", "/scim/v2/Groups", `{`, http.StatusBadRequest},
		{"un grupo sin displayName", "POST", "/scim/v2/Groups",
			`{"schemas":["` + adaptador.EsquemaGrupo + `"]}`, http.StatusBadRequest},
		{"un nombre repetido", "POST", "/scim/v2/Groups",
			`{"schemas":["` + adaptador.EsquemaGrupo + `"],"displayName":"Seguridad"}`, http.StatusConflict},
		{"un miembro que no existe", "POST", "/scim/v2/Groups",
			`{"schemas":["` + adaptador.EsquemaGrupo + `"],"displayName":"Otro","members":[{"value":"no-existe"}]}`,
			http.StatusBadRequest},
		{"leer un grupo inventado", "GET", "/scim/v2/Groups/no-existe", "", http.StatusNotFound},
		{"parchear un grupo inventado", "PATCH", "/scim/v2/Groups/no-existe",
			`{"schemas":["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
			  "Operations":[{"op":"replace","path":"displayName","value":"x"}]}`, http.StatusNotFound},
		{"un PATCH con el cuerpo roto", "PATCH", "/scim/v2/Groups/" + id, `{`, http.StatusBadRequest},
		{"un PATCH que toca el id", "PATCH", "/scim/v2/Groups/" + id,
			`{"schemas":["urn:ietf:params:scim:api:messages:2.0:PatchOp"],
			  "Operations":[{"op":"replace","path":"id","value":"otro"}]}`, http.StatusBadRequest},
		{"un filtro de grupo que no se entiende", "GET",
			`/scim/v2/Groups?filter=members+co+"ana"`, "", http.StatusBadRequest},
	}
	for _, c := range casos {
		w := b.pedirCon(c.metodo, c.ruta, c.cuerpo, "Bearer "+tokenPrueba)
		if w.Code != c.espera {
			t.Errorf("%s: esperaba %d y salio %d: %s", c.nombre, c.espera, w.Code, w.Body.String())
			continue
		}
		// Y el cuerpo del error tiene que ser SCIM, no el texto plano de Go: el
		// IdP lo parsea, y un cuerpo que no puede parsear lo deja sin saber
		// que ha pasado.
		var e map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &e); err != nil {
			t.Errorf("%s: el error no sale en JSON SCIM: %s", c.nombre, w.Body.String())
			continue
		}
		if e["detail"] == nil || e["status"] == nil {
			t.Errorf("%s: el error no dice ni que ha pasado ni con que estado: %+v", c.nombre, e)
		}
	}

	// CONTROL NEGATIVO: uno legitimo pasa. Sin esto, toda la tabla de arriba
	// podria estar pasando porque el endpoint de grupos falla siempre.
	if w := b.pedir("POST", "/scim/v2/Groups",
		`{"schemas":["`+adaptador.EsquemaGrupo+`"],"displayName":"Legitimo"}`); w.Code != http.StatusCreated {
		t.Fatalf("un grupo legitimo tenia que crearse y salio %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// El descubrimiento: lo primero que pide un IdP al conectarse
// ---------------------------------------------------------------------------

// Sin estos tres endpoints, un IdP conforme no llega ni a intentar el primer
// alta: los pide antes de nada para saber que sabe hacer el servidor. Un 404
// aqui se lee, desde el otro lado, como "este no es un servidor SCIM".
func TestLoQueUnIdPPideAntesDeEmpezar(t *testing.T) {
	b := nuevoBanco(t)

	// ResourceTypes: que recursos hay y bajo que URL.
	w := b.pedir("GET", "/scim/v2/ResourceTypes", "")
	if w.Code != http.StatusOK {
		t.Fatalf("ResourceTypes: %d %s", w.Code, w.Body.String())
	}
	lista := b.json(w)
	if lista["totalResults"] != float64(2) {
		t.Fatalf("hay User y Group, o sea dos tipos, y dice %v", lista["totalResults"])
	}
	recursos, _ := lista["Resources"].([]any)
	vistos := map[string]bool{}
	for _, r := range recursos {
		m, _ := r.(map[string]any)
		nombre, _ := m["name"].(string)
		vistos[nombre] = true
		if m["endpoint"] == nil || m["schema"] == nil {
			t.Errorf("el tipo %q no dice su endpoint o su esquema: %+v", nombre, m)
		}
	}
	if !vistos["User"] || !vistos["Group"] {
		t.Fatalf("faltan tipos: %+v", vistos)
	}

	// Schemas: los atributos que se admiten de verdad.
	w = b.pedir("GET", "/scim/v2/Schemas", "")
	if w.Code != http.StatusOK {
		t.Fatalf("Schemas: %d %s", w.Code, w.Body.String())
	}
	cuerpo := w.Body.String()
	for _, urn := range []string{adaptador.EsquemaUsuario, adaptador.EsquemaGrupo, adaptador.EsquemaEmpresa} {
		if !strings.Contains(cuerpo, urn) {
			t.Errorf("Schemas no declara %q, y un IdP conforme ignora lo que no esta "+
				"declarado: acabaria sin mandar esos atributos", urn)
		}
	}

	// Y los tres estan detras de la credencial, igual que todo lo demas: un
	// endpoint de descubrimiento abierto dice la forma de la instalacion a
	// cualquiera que pregunte.
	for _, ruta := range []string{"/scim/v2/ResourceTypes", "/scim/v2/Schemas", "/scim/v2/ServiceProviderConfig"} {
		if w := b.pedirCon("GET", ruta, "", ""); w.Code != http.StatusUnauthorized {
			t.Errorf("HALLAZGO: %s responde %d sin credencial. Un endpoint de "+
				"descubrimiento abierto le cuenta a cualquiera que hay aqui", ruta, w.Code)
		}
	}
}

// ---------------------------------------------------------------------------
// La credencial equivocada, que es distinta de la credencial ausente
// ---------------------------------------------------------------------------

// TestSinCredencialNoSeHaceNada ya recorre todas las rutas sin cabecera. Esto
// recorre las formas de traerla MAL, que es lo que pasa de verdad cuando un IdP
// se configura a medias, y comprueba que ninguna cuela.
func TestUnaCredencialEquivocadaNoEsMediaCredencial(t *testing.T) {
	b := nuevoBanco(t)
	b.crearUsuario("ana")

	casos := []struct{ nombre, autorizacion string }{
		{"otro token entero", "Bearer " + strings.Repeat("z", len(tokenPrueba))},
		{"el token bueno con un caracter cambiado", "Bearer " + tokenPrueba[:len(tokenPrueba)-1] + "z"},
		{"el token bueno con un caracter de mas", "Bearer " + tokenPrueba + "z"},
		{"el token bueno cortado", "Bearer " + tokenPrueba[:len(tokenPrueba)-1]},
		{"sin el esquema Bearer", tokenPrueba},
		{"con otro esquema", "Basic " + tokenPrueba},
		{"la cabecera vacia pero presente", " "},
		{"solo el esquema", "Bearer"},
		{"el esquema y nada", "Bearer "},
	}
	for _, c := range casos {
		w := b.pedirCon("GET", "/scim/v2/Users", "", c.autorizacion)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("HALLAZGO: con %s la peticion ha salido %d en vez de 401. Esta es la "+
				"superficie que da de alta y de baja a personas: %s",
				c.nombre, w.Code, w.Body.String())
			continue
		}
		// Y el 401 tiene que traer WWW-Authenticate, o el cliente no sabe que
		// se le esta pidiendo y reintenta igual de mal.
		if w.Header().Get("WWW-Authenticate") == "" {
			t.Errorf("%s: el 401 no dice como autenticarse", c.nombre)
		}
		// El token bueno NO puede aparecer en la respuesta de un fallo de
		// credencial: seria un oraculo.
		if strings.Contains(w.Body.String(), tokenPrueba) {
			t.Errorf("%s: el token bueno sale en el cuerpo de la respuesta", c.nombre)
		}
	}

	// CONTROL NEGATIVO: el bueno SI entra. Sin esto, toda la tabla podria estar
	// pasando porque el servidor responde 401 a todo.
	if w := b.pedirCon("GET", "/scim/v2/Users", "", "Bearer "+tokenPrueba); w.Code != http.StatusOK {
		t.Fatalf("la credencial buena tenia que entrar y salio %d: %s", w.Code, w.Body.String())
	}

	// Y LO QUE SI TIENE QUE COLAR, que tambien hay que fijarlo o alguien lo
	// "arregla" un dia y rompe a un IdP real.
	//
	// `bearer` en minusculas ENTRA, y es correcto: RFC 7235 apartado 2.1 dice
	// que el nombre del esquema de autenticacion es insensible a mayusculas, y
	// hay clientes que lo mandan asi. Este test se escribio primero afirmando
	// lo contrario y salio rojo: la equivocada era la afirmacion.
	//
	// Lo que NO es insensible es el token, que se compara por su sha256 sobre
	// los bytes crudos. Las dos cosas juntas son la propiedad entera.
	if w := b.pedirCon("GET", "/scim/v2/Users", "", "bearer "+tokenPrueba); w.Code != http.StatusOK {
		t.Errorf("`bearer` en minusculas tenia que entrar (RFC 7235 2.1: el esquema es "+
			"insensible a mayusculas) y salio %d", w.Code)
	}
	if w := b.pedirCon("GET", "/scim/v2/Users", "", "BEARER "+tokenPrueba); w.Code != http.StatusOK {
		t.Errorf("`BEARER` tenia que entrar por lo mismo y salio %d", w.Code)
	}
	// El TOKEN, en cambio, distingue mayusculas. Si no las distinguiera, un
	// token hexadecimal de 32 caracteres perdiria entropia a lo tonto.
	if w := b.pedirCon("GET", "/scim/v2/Users", "", "Bearer "+strings.ToUpper(tokenPrueba)); w.Code != http.StatusUnauthorized {
		t.Errorf("HALLAZGO: el token en mayusculas ha entrado (%d). El token no puede "+
			"ser insensible a mayusculas: regala entropia", w.Code)
	}
}

// El usuario que se desaprovisiona: el caso que un IdP hace todos los meses y
// el que decide si alguien que se fue sigue teniendo acceso.
//
// Y lo que se fija aqui no es solo el 404: es que el borrado SCIM y el permiso
// de entrada al producto son la MISMA decision. Si se pudieran separar, un
// usuario borrado del IdP podria seguir entrando por la puerta del producto.
func TestUnUsuarioDesaprovisionadoDejaDeExistirParaElIdPYParaElProducto(t *testing.T) {
	b := nuevoBanco(t)
	id := b.crearUsuario("ana")

	u, err := b.dir.Leer(id)
	if err != nil {
		t.Fatal(err)
	}
	correo := u.CorreoPrincipal()
	// CONTROL POSITIVO: antes del borrado entra. Sin esto, el de abajo pasaria
	// igual si PuedeEntrar dijera que no siempre.
	if err := b.dir.PuedeEntrar("ana", correo); err != nil {
		t.Fatalf("antes del borrado tenia que poder entrar: %v", err)
	}

	if w := b.pedir("DELETE", "/scim/v2/Users/"+id, ""); w.Code != http.StatusNoContent {
		t.Fatalf("el DELETE tenia que salir 204 y salio %d: %s", w.Code, w.Body.String())
	}

	// 1. Para SCIM ya no existe: ni por GET ni por LIST.
	if w := b.pedir("GET", "/scim/v2/Users/"+id, ""); w.Code != http.StatusNotFound {
		t.Errorf("un usuario borrado se sigue devolviendo por GET: %d", w.Code)
	}
	w := b.pedir("GET", "/scim/v2/Users", "")
	if b.json(w)["totalResults"] != float64(0) {
		t.Errorf("un usuario borrado sigue saliendo en la lista: %s", w.Body.String())
	}

	// 2. Y para el producto tampoco: no entra.
	if err := b.dir.PuedeEntrar("ana", correo); err == nil {
		t.Fatal("HALLAZGO: alguien borrado del IdP sigue pudiendo entrar en el producto. " +
			"Desaprovisionar es exactamente esto y si no ocurre no sirve de nada")
	}

	// 3. Pero el rastro queda, que es lo que distingue una lapida de un olvido:
	//    una obligacion suya se puede seguir ensenando con su nombre en vez de
	//    con un identificador que no resuelve.
	if _, ok := b.dir.Historico(id); !ok {
		t.Error("el borrado ha olvidado que ese usuario existio, y entonces una " +
			"obligacion huerfana se lee como un error del sistema")
	}

	// 4. Y su userName queda libre: si no lo quedara, readmitir a alguien
	//    obligaria a inventarle un nombre.
	if w := b.pedir("POST", "/scim/v2/Users", `{
		"schemas":["`+adaptador.EsquemaUsuario+`"],"userName":"ana","active":true}`); w.Code != http.StatusCreated {
		t.Errorf("el userName de un borrado no ha quedado libre: %d %s", w.Code, w.Body.String())
	}
}
