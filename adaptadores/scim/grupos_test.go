package scim

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// Grupos, PUT y la capa JSON: la mitad del paquete que no tenia ni un test.
//
// POR QUE ESTA ES LA PARTE QUE MAS IMPORTA, y no es cobertura por cobertura.
// SCIM es la superficie por la que un IdP ajeno da de alta y de baja a
// personas, y los grupos son por donde se reparte quien ve que. Un grupo que se
// queda con un miembro borrado no es un dato feo: es alguien que sigue
// recibiendo un escalado, o una obligacion que sigue teniendo dueno cuando ya
// no lo tiene.
//
// Todo test de este fichero que fije una propiedad lleva su control negativo AL
// LADO, en el mismo test: se demuestra que la comprobacion alcanza, no solo que
// el camino feliz pasa.

func grupoDePrueba(t *testing.T, d *Directorio, nombre string, miembros ...string) Grupo {
	t.Helper()
	g, err := d.CrearGrupo(Grupo{Mostrar: nombre, Miembros: miembros}, ahoraFijo)
	if err != nil {
		t.Fatalf("crear el grupo %q: %v", nombre, err)
	}
	return g
}

// ---------------------------------------------------------------------------
// Ciclo de vida de un grupo
// ---------------------------------------------------------------------------

func TestElCicloDeVidaDeUnGrupo(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	ana := crear(t, d, "ana")
	luis := crear(t, d, "luis")

	g := grupoDePrueba(t, d, "Seguridad", ana.ID, luis.ID)
	if g.ID == "" {
		t.Fatal("un grupo sin id no se puede referenciar desde ningun sitio")
	}
	if !g.Creado.Equal(ahoraFijo) || !g.Modificado.Equal(ahoraFijo) {
		t.Fatalf("las marcas de tiempo tienen que ser el instante que entra como dato: %+v", g)
	}

	leido, err := d.LeerGrupo(g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(leido.Miembros) != 2 {
		t.Fatalf("dos miembros, y quedaron %d", len(leido.Miembros))
	}

	// La vista del otro lado: a que grupos pertenece alguien. Es la que usa la
	// serializacion del usuario, y la que un IdP mira para decidir permisos.
	if gs := d.GruposDe(ana.ID); len(gs) != 1 || gs[0].ID != g.ID {
		t.Fatalf("ana esta en Seguridad y GruposDe devolvio %+v", gs)
	}
	if gs := d.GruposDe("id-que-no-existe"); len(gs) != 0 {
		t.Fatalf("un id que no existe no pertenece a nada, y devolvio %+v", gs)
	}

	if err := d.BorrarGrupo(g.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := d.LeerGrupo(g.ID); err == nil {
		t.Fatal("un grupo borrado sigue leyendose")
	}
	// CONTROL NEGATIVO: borrar dos veces tiene que doler, no ser idempotente en
	// silencio. Un DELETE que siempre dice que si deja al IdP creyendo que
	// borro algo que no existia, y no vuelve a mirar.
	if err := d.BorrarGrupo(g.ID); err == nil {
		t.Fatal("borrar un grupo que ya no existe ha salido bien")
	}
	// Y los usuarios siguen ahi: borrar el grupo no borra a nadie.
	if _, err := d.Leer(ana.ID); err != nil {
		t.Fatalf("borrar un grupo ha borrado a un usuario: %v", err)
	}
}

// La comprobacion que separa un directorio de una lista de cadenas: un grupo NO
// puede tener miembros que no existen.
func TestUnGrupoNoAdmiteMiembrosQueNoExisten(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	ana := crear(t, d, "ana")

	// CONTROL POSITIVO: con miembros de verdad, se crea. Sin esto, lo de abajo
	// podria estar pasando porque CrearGrupo falla siempre.
	if _, err := d.CrearGrupo(Grupo{Mostrar: "de verdad", Miembros: []string{ana.ID}}, ahoraFijo); err != nil {
		t.Fatalf("un grupo con un miembro real tiene que crearse: %v", err)
	}

	_, err := d.CrearGrupo(Grupo{Mostrar: "roto", Miembros: []string{ana.ID, "no-existe"}}, ahoraFijo)
	if err == nil {
		t.Fatal("HALLAZGO: un grupo con un miembro inventado se ha creado. Es una lista " +
			"de correos rota que nadie mira hasta que falla un escalado")
	}
	if !strings.Contains(err.Error(), "no-existe") {
		t.Errorf("el error no dice CUAL es el miembro que falta, asi que el operador "+
			"tiene que adivinarlo: %v", err)
	}
}

// Y la version que de verdad muerde: el miembro existia y lo borraron DESPUES.
// Es el ciclo de vida real, no el camino feliz.
//
// ESTE TEST SE ESCRIBIO PRIMERO AL REVES, y conviene que quede dicho porque es
// la forma en que una suposicion se cuela en un test: se dio por hecho que
// borrar a una persona NO reescribia los grupos a su espalda, y que el proximo
// PATCH tendria que negarse a consolidar el grupo. Salio rojo. El codigo hace
// algo mejor: `Borrar` la saca de todos los grupos EN EL ACTO. La propiedad de
// verdad es mas fuerte que la que se iba a fijar, asi que se fija la de verdad.
func TestBorrarAAlguienLoSacaDeTodosSusGruposEnElActo(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	ana := crear(t, d, "ana")
	luis := crear(t, d, "luis")
	guardia := grupoDePrueba(t, d, "Guardia", ana.ID, luis.ID)
	comite := grupoDePrueba(t, d, "Comite", luis.ID)

	// CONTROL POSITIVO: antes del borrado esta en los dos. Sin esto, el test
	// pasaria igual si la pertenencia no se hubiera grabado nunca.
	if gs := d.GruposDe(luis.ID); len(gs) != 2 {
		t.Fatalf("luis tenia que estar en dos grupos y esta en %d", len(gs))
	}

	if err := d.Borrar(luis.ID, ahoraFijo.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	if gs := d.GruposDe(luis.ID); len(gs) != 0 {
		t.Fatalf("HALLAZGO: luis esta borrado del IdP y sigue en %d grupo(s). Un grupo "+
			"que lista a un borrado es una lista de correos rota: ese alguien sigue "+
			"recibiendo lo que reciba el grupo", len(gs))
	}
	g1, err := d.LeerGrupo(guardia.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(g1.Miembros) != 1 || g1.Miembros[0] != ana.ID {
		t.Fatalf("en Guardia tenia que quedar solo ana: %+v", g1.Miembros)
	}
	g2, err := d.LeerGrupo(comite.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(g2.Miembros) != 0 {
		t.Fatalf("Comite tenia que quedarse vacio: %+v", g2.Miembros)
	}

	// Y el grupo se puede seguir parcheando: si al borrado se le hubiera dejado
	// dentro, el siguiente PATCH moriria validando miembros y el operador no
	// podria ni renombrar el grupo.
	if _, err := d.ParchearGrupo(guardia.ID, Parcheo{Operaciones: []Operacion{{
		Op: "replace", Ruta: "displayName", Valor: json.RawMessage(`"Guardia nueva"`),
	}}}, ahoraFijo.Add(2*time.Hour)); err != nil {
		t.Fatalf("el grupo se ha quedado imposible de parchear tras el borrado: %v", err)
	}

	// CONTROL NEGATIVO de la validacion de miembros, que sigue existiendo y
	// tiene que morder: volver a meter al borrado a mano NO se admite.
	if _, err := d.ParchearGrupo(guardia.ID, Parcheo{Operaciones: []Operacion{{
		Op: "add", Ruta: "members", Valor: json.RawMessage(`{"value":"` + luis.ID + `"}`),
	}}}, ahoraFijo.Add(3*time.Hour)); err == nil {
		t.Fatal("HALLAZGO: se ha vuelto a meter en el grupo a alguien borrado del IdP")
	}
}

func TestElNombreDeGrupoEsUnico(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	grupoDePrueba(t, d, "Seguridad")
	// Con otra caja y con espacios: SCIM compara sin distinguir mayusculas, y
	// un duplicado que solo se diferencia en eso es como el IdP acaba con dos
	// grupos que creia que eran uno.
	if _, err := d.CrearGrupo(Grupo{Mostrar: "  seguridad "}, ahoraFijo); err == nil {
		t.Fatal("HALLAZGO: 'seguridad' se ha creado existiendo 'Seguridad'")
	}
	// CONTROL NEGATIVO: uno que de verdad es otro, se crea.
	if _, err := d.CrearGrupo(Grupo{Mostrar: "Seguridad fisica"}, ahoraFijo); err != nil {
		t.Fatalf("un nombre distinto tiene que poder crearse: %v", err)
	}
	// Y sin nombre no hay grupo: a un grupo sin displayName no lo referencia
	// nadie con sentido.
	if _, err := d.CrearGrupo(Grupo{Mostrar: "   "}, ahoraFijo); err == nil {
		t.Fatal("un grupo con el displayName en blanco se ha creado")
	}
}

func TestListarGruposFiltraYPagina(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	grupoDePrueba(t, d, "Alfa")
	grupoDePrueba(t, d, "Beta")
	grupoDePrueba(t, d, "Gamma")

	todos, total, err := d.ListarGrupos(Filtro{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || len(todos) != 3 {
		t.Fatalf("tres grupos, y salieron total=%d len=%d", total, len(todos))
	}

	uno, total, err := d.ListarGrupos(Filtro{Atributo: "displayname", Valor: "beta"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(uno) != 1 || uno[0].Mostrar != "Beta" {
		t.Fatalf("el filtro por displayName no casa sin distinguir mayusculas: %+v (total %d)", uno, total)
	}

	// EL QUE IMPORTA: un filtro que no se sabe aplicar NO devuelve la lista
	// entera. Si la devolviera, el IdP preguntaria "dame los que se llamen X",
	// recibiria todos, concluiria que X existe y no crearia nada. O al reves.
	if _, _, err := d.ListarGrupos(Filtro{Atributo: "miembros", Valor: "ana"}); err == nil {
		t.Fatal("HALLAZGO: un filtro no admitido se ha ignorado en silencio y ha " +
			"devuelto la lista entera. El IdP concluye lo contrario de lo que pregunto")
	}
}

// ---------------------------------------------------------------------------
// PUT: la operacion que sustituye entera y por eso puede perder cosas
// ---------------------------------------------------------------------------

func TestElPutSustituyeEnteroYConservaLoQueNoEsDelIdP(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	ana, err := d.Crear(Usuario{
		UserName: "ana", Mostrar: "Ana", Activo: true, Titulo: "CISO",
		Correos: []Correo{{Valor: "ana@ejemplo.es", Principal: true}},
	}, ahoraFijo)
	if err != nil {
		t.Fatal(err)
	}

	despues := ahoraFijo.Add(24 * time.Hour)
	puesto, err := d.Reemplazar(ana.ID, Usuario{
		UserName: "ana", Mostrar: "Ana Garcia", Activo: false,
	}, despues)
	if err != nil {
		t.Fatal(err)
	}
	// Lo que el PUT trae, sustituye.
	if puesto.Mostrar != "Ana Garcia" || puesto.Activo {
		t.Fatalf("el PUT no ha sustituido: %+v", puesto)
	}
	// Lo que el PUT NO trae, desaparece: eso es un PUT y no un PATCH.
	if puesto.Titulo != "" || len(puesto.Correos) != 0 {
		t.Fatalf("un PUT tiene que dejar vacio lo que no manda, y quedo %+v", puesto)
	}
	// Y lo que NO es del IdP se conserva, que es la parte que se olvida: el id
	// y la fecha de creacion no los pone quien manda el PUT.
	if puesto.ID != ana.ID {
		t.Fatal("el PUT ha cambiado el id: cualquier referencia al usuario se rompe")
	}
	if !puesto.Creado.Equal(ahoraFijo) {
		t.Fatalf("el PUT ha reescrito la fecha de creacion: %v", puesto.Creado)
	}
	if !puesto.Modificado.Equal(despues) {
		t.Fatalf("el PUT no ha actualizado la fecha de modificacion: %v", puesto.Modificado)
	}
}

func TestElPutNoRobaElUserNameDeOtroNiLoDejaVacio(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	ana := crear(t, d, "ana")
	luis := crear(t, d, "luis")

	if _, err := d.Reemplazar(ana.ID, Usuario{UserName: "luis"}, ahoraFijo); err == nil {
		t.Fatal("HALLAZGO: un PUT ha puesto a ana el userName de luis. Dos cuentas con " +
			"el mismo nombre es el IdP y el producto discrepando sobre quien es quien")
	}
	if _, err := d.Reemplazar(ana.ID, Usuario{UserName: "  "}, ahoraFijo); err == nil {
		t.Fatal("HALLAZGO: un PUT ha dejado el userName vacio")
	}
	// CONTROL NEGATIVO: el suyo propio SI, que es un PUT normal que no cambia el
	// nombre. Si esto fallara, la comprobacion de unicidad estaria comparando al
	// usuario consigo mismo.
	if _, err := d.Reemplazar(ana.ID, Usuario{UserName: "ana", Mostrar: "Ana"}, ahoraFijo); err != nil {
		t.Fatalf("un PUT que conserva su propio userName tiene que pasar: %v", err)
	}
	// Y sobre un borrado, no: para SCIM ese recurso ya no existe.
	if err := d.Borrar(luis.ID, ahoraFijo); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Reemplazar(luis.ID, Usuario{UserName: "luis2"}, ahoraFijo); err == nil {
		t.Fatal("HALLAZGO: un PUT ha resucitado a un usuario borrado")
	}
}

// ---------------------------------------------------------------------------
// La capa JSON: lo que sale por el cable
// ---------------------------------------------------------------------------

func TestElUsuarioQueSalePorElCableEsElQueEsperaUnIdP(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	jefa := crear(t, d, "jefa")
	ana, err := d.Crear(Usuario{
		UserName: "ana", Mostrar: "Ana", Activo: true, Titulo: "CISO",
		Nombre:         Nombre{Familia: "Garcia", Pila: "Ana"},
		Correos:        []Correo{{Valor: "ana@ejemplo.es", Principal: true}},
		ManagerIdP:     jefa.ID,
		Departamento:   "Seguridad",
		NumeroEmpleado: "0042",
	}, ahoraFijo)
	if err != nil {
		t.Fatal(err)
	}
	g := grupoDePrueba(t, d, "Seguridad", ana.ID)

	const base = "https://plazum.ejemplo.es/scim/v2"
	b, err := d.AUsuarioJSON(ana, base)
	if err != nil {
		t.Fatal(err)
	}
	var salida map[string]any
	if err := json.Unmarshal(b, &salida); err != nil {
		t.Fatalf("lo que sale por el cable no es JSON valido: %v", err)
	}

	// Los esquemas: sin ellos un IdP conforme rechaza el recurso entero.
	esquemas, _ := salida["schemas"].([]any)
	tieneEmpresa := false
	for _, e := range esquemas {
		if e == EsquemaEmpresa {
			tieneEmpresa = true
		}
	}
	if len(esquemas) == 0 || esquemas[0] != EsquemaUsuario {
		t.Fatalf("el primer esquema tiene que ser el de User y es %v", esquemas)
	}
	if !tieneEmpresa {
		t.Error("hay atributos de la extension enterprise y su esquema no se declara: " +
			"un IdP conforme ignora lo que no esta declarado")
	}

	// La pertenencia a grupos viaja EN el usuario, que es de donde la lee el IdP
	// para decidir permisos.
	grupos, _ := salida["groups"].([]any)
	if len(grupos) != 1 {
		t.Fatalf("ana esta en un grupo y en el JSON salen %d", len(grupos))
	}
	primero, _ := grupos[0].(map[string]any)
	if primero["value"] != g.ID || primero["display"] != "Seguridad" {
		t.Fatalf("la referencia al grupo esta incompleta: %+v", primero)
	}
	if primero["$ref"] != base+"/Groups/"+g.ID {
		t.Fatalf("el $ref del grupo no es una URL resoluble: %v", primero["$ref"])
	}

	// meta, que es lo que usa el IdP para no reescribir lo que no cambio.
	meta, _ := salida["meta"].(map[string]any)
	if meta["resourceType"] != "User" || meta["version"] == "" {
		t.Fatalf("meta incompleto: %+v", meta)
	}
	if meta["location"] != base+"/Users/"+ana.ID {
		t.Fatalf("meta.location no es la URL del recurso: %v", meta["location"])
	}

	// CONTROL NEGATIVO de la base: sin base no se inventan URL relativas que el
	// IdP no sabria resolver.
	b2, err := d.AUsuarioJSON(ana, "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b2), `"$ref"`) || strings.Contains(string(b2), `"location"`) {
		t.Errorf("sin base se han escrito enlaces igualmente:\n%s", b2)
	}
}

func TestElGrupoQueSalePorElCableLlevaSusMiembros(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	ana := crear(t, d, "ana")
	g := grupoDePrueba(t, d, "Seguridad", ana.ID)

	b, err := d.AGrupoJSON(g, "https://plazum.ejemplo.es/scim/v2")
	if err != nil {
		t.Fatal(err)
	}
	var salida map[string]any
	if err := json.Unmarshal(b, &salida); err != nil {
		t.Fatal(err)
	}
	if salida["displayName"] != "Seguridad" {
		t.Fatalf("el displayName no viaja: %+v", salida)
	}
	miembros, _ := salida["members"].([]any)
	if len(miembros) != 1 {
		t.Fatalf("un miembro, y en el JSON salen %d", len(miembros))
	}
	m, _ := miembros[0].(map[string]any)
	if m["value"] != ana.ID {
		t.Fatalf("el miembro no lleva su id: %+v", m)
	}
}

func TestLaListaYElErrorSonLosQueDiceElRFC(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	ana := crear(t, d, "ana")
	uno, err := d.AUsuarioJSON(ana, "")
	if err != nil {
		t.Fatal(err)
	}

	b, err := ListaJSON([]json.RawMessage{uno}, 7, 3)
	if err != nil {
		t.Fatal(err)
	}
	var lista map[string]any
	if err := json.Unmarshal(b, &lista); err != nil {
		t.Fatal(err)
	}
	// totalResults es el total REAL, no lo que cabe en la pagina: si fueran lo
	// mismo, el IdP dejaria de paginar al recibir la primera pagina y se
	// quedaria con media plantilla creyendola entera.
	if lista["totalResults"] != float64(7) {
		t.Fatalf("totalResults tiene que ser el total y es %v", lista["totalResults"])
	}
	if lista["itemsPerPage"] != float64(1) || lista["startIndex"] != float64(3) {
		t.Fatalf("la paginacion no se declara bien: %+v", lista)
	}

	// Una lista VACIA sigue siendo una lista: `Resources` no puede faltar ni
	// venir a null, que es lo que hace reventar a un cliente estricto.
	vacia, err := ListaJSON(nil, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	var listaVacia map[string]json.RawMessage
	if err := json.Unmarshal(vacia, &listaVacia); err != nil {
		t.Fatal(err)
	}
	crudo, hay := listaVacia["Resources"]
	if !hay || string(crudo) == "null" {
		t.Errorf("una lista vacia tiene que traer Resources como lista vacia y trae %s:\n%s",
			crudo, vacia)
	}

	// Y el error: con su esquema y su status, que es lo que un IdP sabe leer.
	je := ErrorJSON(errValor("esto no vale"))
	var e map[string]any
	if err := json.Unmarshal(je, &e); err != nil {
		t.Fatalf("el error que sale por el cable no es JSON: %v", err)
	}
	if e["status"] != "400" || e["scimType"] != "invalidValue" {
		t.Fatalf("el error no lleva status ni scimType utiles: %+v", e)
	}
	detalle, _ := e["detail"].(string)
	if !strings.Contains(detalle, "esto no vale") {
		t.Fatalf("el error no dice que ha pasado: %+v", e)
	}
}

// El POST no puede colar lo que el PATCH prohibe: si pudiera, la lista blanca
// del PATCH no serviria de nada, porque bastaria borrar y volver a crear.
func TestLoQueElPatchProhibeElPostTampocoLoCuela(t *testing.T) {
	_, err := DeUsuarioJSON([]byte(`{
		"schemas":["urn:ietf:params:scim:schemas:core:2.0:User"],
		"userName":"ana","roles":[{"value":"admin"}]}`))
	if err == nil {
		t.Fatal("HALLAZGO: un POST con `roles` se ha aceptado. La lista blanca del PATCH " +
			"no sirve de nada si se puede borrar y volver a crear con lo prohibido dentro")
	}
	// CONTROL NEGATIVO: sin el atributo prohibido, el mismo cuerpo entra.
	u, err := DeUsuarioJSON([]byte(`{
		"schemas":["urn:ietf:params:scim:schemas:core:2.0:User"],
		"userName":"ana","displayName":"Ana","active":true}`))
	if err != nil {
		t.Fatalf("un cuerpo legitimo tiene que entrar: %v", err)
	}
	if u.UserName != "ana" || !u.Activo {
		t.Fatalf("mal leido: %+v", u)
	}
}

func TestUnCuerpoMalFormadoNoSeInterpretaAMedias(t *testing.T) {
	for _, caso := range []struct{ nombre, cuerpo string }{
		{"no es JSON", `{`},
		{"vacio", ``},
		{"una lista donde va un objeto", `[]`},
	} {
		if _, err := DeUsuarioJSON([]byte(caso.cuerpo)); err == nil {
			t.Errorf("HALLAZGO: el cuerpo de usuario %q (%s) se ha aceptado. Un cuerpo que "+
				"se interpreta a medias da de alta a alguien distinto del que el IdP creia",
				caso.cuerpo, caso.nombre)
		}
	}
	for _, caso := range []struct{ nombre, cuerpo string }{
		{"no es JSON", `{`},
		{"vacio", ``},
	} {
		if _, err := DeGrupoJSON([]byte(caso.cuerpo)); err == nil {
			t.Errorf("HALLAZGO: el grupo %q (%s) se ha aceptado", caso.cuerpo, caso.nombre)
		}
	}
	// CONTROL NEGATIVO de los dos: uno bueno pasa, o lo de arriba no mediria nada.
	if _, err := DeGrupoJSON([]byte(`{"schemas":["urn:ietf:params:scim:schemas:core:2.0:Group"],
		"displayName":"Seguridad","members":[]}`)); err != nil {
		t.Fatalf("un grupo legitimo tiene que entrar: %v", err)
	}
}

// Lo que falta obligatorio NO lo rechaza el lector de JSON, lo rechaza el
// directorio. Y esto tambien se escribio primero al reves: se afirmo que
// DeUsuarioJSON tenia que exigir `userName` y salio rojo.
//
// La validacion esta POR CAPAS a proposito, y esta bien que lo este: el lector
// dice si el cuerpo es un recurso SCIM legible, y el directorio dice si ese
// recurso puede existir. Mezclarlas obligaria a que el lector supiera cosas del
// estado (que un userName ya esta tomado, por ejemplo), que es justo lo que no
// tiene delante.
//
// Lo que hay que fijar entonces no es "el lector rechaza", es que EL PAR NO
// DEJA PASAR NADA: lo que el lector admite, el directorio lo para.
func TestLoObligatorioLoParaElDirectorio(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)

	sinNombre := []byte(`{"schemas":["urn:ietf:params:scim:schemas:core:2.0:User"],"displayName":"Ana"}`)
	u, err := DeUsuarioJSON(sinNombre)
	if err != nil {
		t.Fatalf("el lector rechaza un cuerpo sin userName, asi que este test ya no "+
			"describe el reparto de capas y hay que reescribirlo: %v", err)
	}
	if _, err := d.Crear(u, ahoraFijo); err == nil {
		t.Fatal("HALLAZGO: un usuario sin `userName` ha entrado. Ni el lector ni el " +
			"directorio lo paran, o sea que NADIE lo para, y ese usuario no lo vuelve " +
			"a encontrar el IdP nunca")
	}

	sinMostrar := []byte(`{"schemas":["urn:ietf:params:scim:schemas:core:2.0:Group"]}`)
	g, err := DeGrupoJSON(sinMostrar)
	if err != nil {
		t.Fatalf("el lector rechaza un grupo sin displayName, hay que reescribir este test: %v", err)
	}
	if _, err := d.CrearGrupo(g, ahoraFijo); err == nil {
		t.Fatal("HALLAZGO: un grupo sin `displayName` ha entrado")
	}
}

// ---------------------------------------------------------------------------
// PATCH de grupo: la operacion con la que un IdP mueve a la gente
// ---------------------------------------------------------------------------

func TestElPatchDeGrupoAnadeQuitaYVacia(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	ana := crear(t, d, "ana")
	luis := crear(t, d, "luis")
	eva := crear(t, d, "eva")
	g := grupoDePrueba(t, d, "Guardia", ana.ID)

	parchear := func(ops ...Operacion) Grupo {
		t.Helper()
		out, err := d.ParchearGrupo(g.ID, Parcheo{Operaciones: ops}, ahoraFijo)
		if err != nil {
			t.Fatalf("PATCH: %v", err)
		}
		return out
	}

	// add con una lista de referencias.
	out := parchear(Operacion{Op: "add", Ruta: "members",
		Valor: json.RawMessage(`[{"value":"` + luis.ID + `"},{"value":"` + eva.ID + `"}]`)})
	if len(out.Miembros) != 3 {
		t.Fatalf("tres miembros y hay %d: %+v", len(out.Miembros), out.Miembros)
	}

	// remove con filtro: quita a UNO, no a todos. Es la diferencia entre mover a
	// una persona de equipo y vaciar el equipo.
	out = parchear(Operacion{Op: "remove", Ruta: `members[value eq "` + luis.ID + `"]`})
	if len(out.Miembros) != 2 {
		t.Fatalf("un remove con filtro tenia que quitar a uno y dejo %+v", out.Miembros)
	}
	for _, m := range out.Miembros {
		if m == luis.ID {
			t.Fatal("el remove con filtro no ha quitado a quien nombraba")
		}
	}

	// replace de members sustituye entero.
	out = parchear(Operacion{Op: "replace", Ruta: "members",
		Valor: json.RawMessage(`[{"value":"` + ana.ID + `"}]`)})
	if len(out.Miembros) != 1 || out.Miembros[0] != ana.ID {
		t.Fatalf("replace tenia que dejar solo a ana: %+v", out.Miembros)
	}

	// remove sin valor y sin filtro vacia el grupo.
	out = parchear(Operacion{Op: "remove", Ruta: "members"})
	if len(out.Miembros) != 0 {
		t.Fatalf("un remove de members sin filtro vacia el grupo y quedo %+v", out.Miembros)
	}

	// Los duplicados no se acumulan: dos add del mismo no dejan dos entradas.
	out = parchear(
		Operacion{Op: "add", Ruta: "members", Valor: json.RawMessage(`{"value":"` + ana.ID + `"}`)},
		Operacion{Op: "add", Ruta: "members", Valor: json.RawMessage(`{"value":"` + ana.ID + `"}`)},
	)
	if len(out.Miembros) != 1 {
		t.Fatalf("el mismo miembro dos veces tiene que quedar en uno: %+v", out.Miembros)
	}

	// Y el displayName y el externalId, que es lo otro que un IdP toca.
	out = parchear(
		Operacion{Op: "replace", Ruta: "displayName", Valor: json.RawMessage(`"Guardia B"`)},
		Operacion{Op: "add", Ruta: "externalId", Valor: json.RawMessage(`"grp-77"`)},
	)
	if out.Mostrar != "Guardia B" || out.ExternalID != "grp-77" {
		t.Fatalf("displayName o externalId no se aplicaron: %+v", out)
	}
}

func TestElPatchDeGrupoRechazaLoQueNoDebeTocar(t *testing.T) {
	d := nuevoDirectorioDePrueba(t)
	g := grupoDePrueba(t, d, "Guardia")

	for _, caso := range []struct {
		nombre string
		op     Operacion
	}{
		{"el id", Operacion{Op: "replace", Ruta: "id", Valor: json.RawMessage(`"otro"`)}},
		{"meta", Operacion{Op: "replace", Ruta: "meta", Valor: json.RawMessage(`{}`)}},
		{"los esquemas", Operacion{Op: "replace", Ruta: "schemas", Valor: json.RawMessage(`[]`)}},
		{"una ruta inventada", Operacion{Op: "replace", Ruta: "loquesea", Valor: json.RawMessage(`"x"`)}},
		{"members con basura", Operacion{Op: "add", Ruta: "members", Valor: json.RawMessage(`"no es una referencia"`)}},
		{"un miembro sin value", Operacion{Op: "add", Ruta: "members", Valor: json.RawMessage(`[{"value":"  "}]`)}},
	} {
		if _, err := d.ParchearGrupo(g.ID, Parcheo{Operaciones: []Operacion{caso.op}}, ahoraFijo); err == nil {
			t.Errorf("HALLAZGO: el PATCH ha aceptado tocar %s", caso.nombre)
		}
	}
	// Un PATCH sin operaciones no es un PATCH.
	if _, err := d.ParchearGrupo(g.ID, Parcheo{}, ahoraFijo); err == nil {
		t.Error("un PATCH sin operaciones se ha aceptado")
	}
	// Y sobre un grupo que no existe, tampoco.
	if _, err := d.ParchearGrupo("no-existe", Parcheo{Operaciones: []Operacion{
		{Op: "replace", Ruta: "displayName", Valor: json.RawMessage(`"x"`)},
	}}, ahoraFijo); err == nil {
		t.Error("se ha parcheado un grupo que no existe")
	}
	// CONTROL NEGATIVO: uno legitimo pasa. Sin esto, todo lo de arriba podria
	// estar pasando porque ParchearGrupo falla siempre.
	if _, err := d.ParchearGrupo(g.ID, Parcheo{Operaciones: []Operacion{
		{Op: "replace", Ruta: "displayName", Valor: json.RawMessage(`"Guardia B"`)},
	}}, ahoraFijo); err != nil {
		t.Fatalf("un PATCH legitimo tiene que pasar: %v", err)
	}
	// Y el grupo NO se ha quedado a medias por los rechazos de arriba: un PATCH
	// que falla no aplica media lista de operaciones.
	final, err := d.LeerGrupo(g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if final.ID != g.ID {
		t.Fatalf("el id del grupo ha cambiado pese a los rechazos: %+v", final)
	}
}
