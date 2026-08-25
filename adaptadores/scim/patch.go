package scim

import (
	"encoding/json"
	"strings"
	"time"
)

// MaxOperacionesPatch acota cuantas operaciones lleva un PATCH.
//
// Un PATCH real de Entra ID trae entre una y seis. Mil operaciones en una
// peticion no son un aprovisionamiento, son una forma de hacernos trabajar.
const MaxOperacionesPatch = 100

// Operacion es una operacion de PATCH (RFC 7644 seccion 3.5.2).
type Operacion struct {
	Op    string          `json:"op"`
	Ruta  string          `json:"path"`
	Valor json.RawMessage `json:"value"`
}

// Parcheo es el cuerpo completo de un PATCH.
type Parcheo struct {
	Esquemas    []string    `json:"schemas"`
	Operaciones []Operacion `json:"Operations"`
}

// atributosProhibidos son los que NUNCA se aceptan de un PATCH, con el motivo
// escrito en el mensaje.
//
// Esta es la defensa contra el PATCH que escala privilegios, y es una lista
// blanca por la puerta de atras: lo que no esta en atributosEscribibles ya se
// rechaza, pero estos se nombran uno a uno para que el IdP reciba un mensaje
// que explique la decision de producto en vez de un "ruta no admitida" que
// parece un fallo nuestro.
var atributosProhibidos = map[string]string{
	"id": "el `id` lo asigna plazum y es inmutable. Si el IdP quiere su propio identificador, " +
		"eso es `externalId`",
	"meta":    "`meta` es de solo lectura: lo escribe el servidor",
	"schemas": "`schemas` describe el recurso, no se parchea",
	"groups": "`groups` es de solo lectura en el esquema User (RFC 7643 seccion 4.1.2). " +
		"La pertenencia se cambia parcheando el GRUPO, no el usuario",
	"roles": "plazum no toma los roles del IdP: el rol dentro de plazum se asigna dentro de " +
		"plazum. Si el aprovisionamiento pudiera mandarlos, quien controle el token de SCIM " +
		"controlaria los privilegios. Quita el mapeo de `roles` en el aprovisionamiento",
	"entitlements": "igual que `roles`: los privilegios no entran por el aprovisionamiento",
	"password": "plazum no tiene contrasenas propias, la autenticacion es OIDC. Aceptar una " +
		"contrasena crearia una segunda via de entrada que nadie vigila. Quita el mapeo de " +
		"`password`",
	"x509certificates": "no se guardan certificados de usuario",
}

// atributosEscribibles es la lista blanca de lo que un PATCH o un PUT pueden
// tocar, en minusculas y sin el prefijo de esquema.
var atributosEscribibles = map[string]bool{
	"username": true, "externalid": true, "active": true, "displayname": true,
	"title": true, "emails": true,
	"name.givenname": true, "name.familyname": true, "name.formatted": true,
	// La extension enterprise.
	"manager": true, "department": true, "employeenumber": true,
}

// atributosIgnorables son atributos legitimos del esquema que este modelo no
// guarda.
//
// Se aceptan y se descartan, en vez de rechazarse, porque los IdP mandan
// muchos por defecto y rechazarlos rompe el aprovisionamiento entero por un
// campo que no le importa a nadie. Que esten ENUMERADOS y no descartados a
// ciegas es lo que impide que este agujero crezca: un atributo nuevo que no
// este ni aqui ni en la lista blanca se rechaza y alguien tiene que decidir.
var atributosIgnorables = map[string]bool{
	"nickname": true, "profileurl": true, "preferredlanguage": true, "locale": true,
	"timezone": true, "addresses": true, "phonenumbers": true, "ims": true,
	"photos": true, "usertype": true,
	"name.middlename": true, "name.honorificprefix": true, "name.honorificsuffix": true,
	"organization": true, "division": true, "costcenter": true,
}

// Parchear aplica un PATCH a un usuario. Es el PATCH /Users/{id}.
func (d *Directorio) Parchear(id string, p Parcheo, ahora time.Time) (Usuario, error) {
	if len(p.Operaciones) == 0 {
		return Usuario{}, errSintaxis("un PATCH sin operaciones no cambia nada. Si el IdP " +
			"queria borrar el recurso, eso es un DELETE")
	}
	if len(p.Operaciones) > MaxOperacionesPatch {
		return Usuario{}, errSintaxis("el PATCH trae %d operaciones y el maximo es %d",
			len(p.Operaciones), MaxOperacionesPatch)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	actual, ok := d.usuarios[id]
	if !ok || !actual.Vivo() {
		return Usuario{}, errNoEncontrado("usuario", id)
	}
	// Se trabaja sobre una COPIA y solo se confirma si todas las operaciones
	// pasan. Un PATCH es atomico por especificacion (RFC 7644 seccion 3.5.2):
	// aplicar la mitad deja al IdP creyendo que fallo entero y al directorio
	// con la mitad aplicada, que es la peor combinacion posible.
	copia := *actual
	for n, op := range p.Operaciones {
		if err := d.aplicarOperacion(&copia, op, ahora); err != nil {
			if e, ok := err.(*Error); ok {
				e.Detalle = "operacion " + itoa(n+1) + ": " + e.Detalle
				return Usuario{}, e
			}
			return Usuario{}, err
		}
	}
	// Las comprobaciones que dependen del resultado final, no de cada paso.
	if strings.TrimSpace(copia.UserName) == "" {
		return Usuario{}, errValor("el PATCH deja el `userName` vacio")
	}
	if normalizar(copia.UserName) != normalizar(actual.UserName) {
		if otro, ya := d.porNombre[normalizar(copia.UserName)]; ya && otro != id {
			return Usuario{}, errUnicidad("el `userName` %q ya es de otro usuario (id %s)",
				copia.UserName, otro)
		}
	}
	if copia.ManagerIdP != "" && copia.ManagerIdP != actual.ManagerIdP {
		if err := d.validarManagerBloqueado(id, copia.ManagerIdP); err != nil {
			return Usuario{}, err
		}
	}
	delete(d.porNombre, normalizar(actual.UserName))
	copia.Modificado = ahora
	*actual = copia
	d.porNombre[normalizar(copia.UserName)] = id
	return copia, nil
}

func (d *Directorio) aplicarOperacion(u *Usuario, op Operacion, ahora time.Time) error {
	_ = ahora
	acc := normalizar(op.Op)
	switch acc {
	case "add", "replace", "remove":
	default:
		return errSintaxis("`op` vale %q y solo se admiten add, replace y remove", op.Op)
	}
	ruta := normalizarRuta(op.Ruta)
	if acc == "remove" && ruta == "" {
		return errRuta("un `remove` sin `path` borraria el recurso entero, y RFC 7644 lo " +
			"prohibe. Si el IdP queria dar de baja al usuario, eso es `replace` de " +
			"`active: false`, o un DELETE")
	}
	// PATCH sin `path`: el valor es un objeto de atributos. Es como Entra ID
	// manda casi todo.
	if ruta == "" {
		var campos map[string]json.RawMessage
		if err := json.Unmarshal(op.Valor, &campos); err != nil {
			return errSintaxis("un `%s` sin `path` necesita que `value` sea un objeto de "+
				"atributos, y no lo es", acc)
		}
		claves := make([]string, 0, len(campos))
		for k := range campos {
			claves = append(claves, k)
		}
		ordenar(claves)
		for _, k := range claves {
			if err := d.asignar(u, normalizarRuta(k), campos[k], acc); err != nil {
				return err
			}
		}
		return nil
	}
	return d.asignar(u, ruta, op.Valor, acc)
}

// normalizarRuta quita el prefijo de esquema y el filtro de un atributo
// multivaluado, y baja a minusculas.
//
// Los IdP mandan la ruta de tres formas para lo mismo: `manager`,
// `urn:...:enterprise:2.0:User:manager` y
// `urn:...:enterprise:2.0:User.manager`. Las tres tienen que llegar al mismo
// sitio o el aprovisionamiento falla segun el IdP que toque.
func normalizarRuta(r string) string {
	r = strings.TrimSpace(r)
	if r == "" {
		return ""
	}
	bajo := strings.ToLower(r)
	for _, esquema := range []string{strings.ToLower(EsquemaEmpresa), strings.ToLower(EsquemaUsuario)} {
		for _, sep := range []string{":", "."} {
			if strings.HasPrefix(bajo, esquema+sep) {
				bajo = bajo[len(esquema)+len(sep):]
			}
		}
	}
	// El filtro de un multivaluado: emails[type eq "work"].value -> emails.
	// No se implementa el filtro: se aplica sobre la coleccion entera, y eso
	// se dice en el godoc de este paquete. Para `emails`, que es lo unico
	// multivaluado que se guarda, la diferencia no cambia el resultado que le
	// importa al producto (que haya un correo con el que avisar).
	if i := strings.Index(bajo, "["); i >= 0 {
		if j := strings.Index(bajo[i:], "]"); j >= 0 {
			bajo = bajo[:i] + bajo[i+j+1:]
		} else {
			bajo = bajo[:i]
		}
	}
	bajo = strings.TrimPrefix(bajo, ".")
	return bajo
}

// asignar escribe un atributo, pasando por la lista blanca.
func (d *Directorio) asignar(u *Usuario, ruta string, valor json.RawMessage, acc string) error {
	if ruta == "" {
		return errRuta("ruta vacia")
	}
	raiz := ruta
	if i := strings.Index(ruta, "."); i > 0 {
		raiz = ruta[:i]
	}
	if motivo, prohibido := atributosProhibidos[raiz]; prohibido {
		return errMutabilidad("no se admite tocar %q: %s", raiz, motivo)
	}
	if atributosIgnorables[ruta] || atributosIgnorables[raiz] {
		return nil
	}
	if !atributosEscribibles[ruta] {
		return errRuta("no se sabe escribir %q. Los atributos que plazum guarda son: "+
			"userName, externalId, active, displayName, title, emails, name.givenName, "+
			"name.familyName, name.formatted, y de la extension enterprise manager, "+
			"department y employeeNumber. Se prefiere decirlo a ignorarlo en silencio: un "+
			"PATCH que se ignora hace creer al IdP que se aplico", ruta)
	}

	quitar := acc == "remove"
	switch ruta {
	case "username":
		return asignarCadena(&u.UserName, valor, quitar)
	case "externalid":
		return asignarCadena(&u.ExternalID, valor, quitar)
	case "displayname":
		return asignarCadena(&u.Mostrar, valor, quitar)
	case "title":
		return asignarCadena(&u.Titulo, valor, quitar)
	case "department":
		return asignarCadena(&u.Departamento, valor, quitar)
	case "employeenumber":
		return asignarCadena(&u.NumeroEmpleado, valor, quitar)
	case "name.givenname":
		return asignarCadena(&u.Nombre.Pila, valor, quitar)
	case "name.familyname":
		return asignarCadena(&u.Nombre.Familia, valor, quitar)
	case "name.formatted":
		return asignarCadena(&u.Nombre.Formateado, valor, quitar)
	case "active":
		if quitar {
			u.Activo = false
			return nil
		}
		var b bool
		if err := json.Unmarshal(valor, &b); err != nil {
			// Entra ID manda "True" y "False" como cadena. Rechazarlo seria
			// romper el offboarding por una comilla.
			var s string
			if err2 := json.Unmarshal(valor, &s); err2 != nil {
				return errValor("`active` tiene que ser true o false, y llego %s", recorte(valor))
			}
			switch normalizar(s) {
			case "true":
				b = true
			case "false":
				b = false
			default:
				return errValor("`active` tiene que ser true o false, y llego %q", s)
			}
		}
		u.Activo = b
		return nil
	case "emails":
		if quitar {
			u.Correos = nil
			return nil
		}
		return asignarCorreos(u, valor, acc)
	case "manager":
		if quitar {
			u.ManagerIdP = ""
			return nil
		}
		id, err := leerReferenciaManager(valor)
		if err != nil {
			return err
		}
		u.ManagerIdP = id
		return nil
	}
	return errRuta("no se sabe escribir %q", ruta)
}

func asignarCadena(destino *string, valor json.RawMessage, quitar bool) error {
	if quitar {
		*destino = ""
		return nil
	}
	var s string
	if err := json.Unmarshal(valor, &s); err != nil {
		return errValor("se esperaba una cadena y llego %s", recorte(valor))
	}
	*destino = strings.TrimSpace(s)
	return nil
}

func asignarCorreos(u *Usuario, valor json.RawMessage, acc string) error {
	var varios []Correo
	if err := json.Unmarshal(valor, &varios); err != nil {
		// Un solo correo, que es lo que llega cuando la ruta traia filtro.
		var uno Correo
		if err2 := json.Unmarshal(valor, &uno); err2 != nil {
			var s string
			if err3 := json.Unmarshal(valor, &s); err3 != nil {
				return errValor("`emails` tiene que ser una lista de correos y llego %s",
					recorte(valor))
			}
			uno = Correo{Valor: s}
		}
		varios = []Correo{uno}
	}
	for _, c := range varios {
		if strings.TrimSpace(c.Valor) == "" {
			return errValor("un correo sin `value` no sirve para avisar a nadie")
		}
	}
	if acc == "add" {
		u.Correos = append(u.Correos, varios...)
		return nil
	}
	u.Correos = varios
	return nil
}

// leerReferenciaManager acepta las dos formas en que llega el `manager`: el
// objeto de referencia {"value": "..."} y la cadena suelta.
//
// Okta manda el objeto, y hay conectores que mandan la cadena. Aceptar solo una
// significa que el atributo que sostiene el escalado depende del IdP que toque.
func leerReferenciaManager(valor json.RawMessage) (string, error) {
	var ref struct {
		Valor   string `json:"value"`
		Mostrar string `json:"displayName"`
		Ref     string `json:"$ref"`
	}
	if err := json.Unmarshal(valor, &ref); err == nil && ref.Valor != "" {
		return strings.TrimSpace(ref.Valor), nil
	}
	var s string
	if err := json.Unmarshal(valor, &s); err == nil && strings.TrimSpace(s) != "" {
		return strings.TrimSpace(s), nil
	}
	return "", errValor("`manager` tiene que ser el id del jefe, como cadena o como " +
		"{\"value\": \"<id>\"}. Es el atributo del que sale el escalado jerarquico: si no " +
		"llega, la obligacion vencida no tiene a quien subir")
}

// recorte acota lo que se devuelve del cuerpo que mando el IdP. Nunca se
// devuelve entero: si el cuerpo llevara algo sensible, iria de vuelta en el
// error y de ahi al log del IdP.
func recorte(b []byte) string {
	const max = 60
	s := strings.TrimSpace(string(b))
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

func ordenar(xs []string) {
	for i := 1; i < len(xs); i++ {
		for j := i; j > 0 && xs[j] < xs[j-1]; j-- {
			xs[j], xs[j-1] = xs[j-1], xs[j]
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// ParchearGrupo aplica un PATCH a un grupo. Solo `members` y `displayName`: es
// lo que los IdP cambian de un grupo.
func (d *Directorio) ParchearGrupo(id string, p Parcheo, ahora time.Time) (Grupo, error) {
	if len(p.Operaciones) == 0 {
		return Grupo{}, errSintaxis("un PATCH sin operaciones no cambia nada")
	}
	if len(p.Operaciones) > MaxOperacionesPatch {
		return Grupo{}, errSintaxis("el PATCH trae %d operaciones y el maximo es %d",
			len(p.Operaciones), MaxOperacionesPatch)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	actual, ok := d.grupos[id]
	if !ok {
		return Grupo{}, errNoEncontrado("grupo", id)
	}
	copia := *actual
	copia.Miembros = append([]string(nil), actual.Miembros...)

	for n, op := range p.Operaciones {
		acc := normalizar(op.Op)
		ruta := normalizarRuta(op.Ruta)
		// El filtro de un remove de miembro: members[value eq "id"]. Aqui SI
		// hace falta leerlo, porque quitar a uno y quitarlos a todos no es lo
		// mismo ni de lejos.
		objetivo := valorDelFiltro(op.Ruta)
		prefijo := "operacion " + itoa(n+1) + ": "
		switch {
		case ruta == "displayname" && (acc == "replace" || acc == "add"):
			if err := asignarCadena(&copia.Mostrar, op.Valor, false); err != nil {
				return Grupo{}, errValor("%s%v", prefijo, err)
			}
		case ruta == "externalid" && (acc == "replace" || acc == "add"):
			if err := asignarCadena(&copia.ExternalID, op.Valor, false); err != nil {
				return Grupo{}, errValor("%s%v", prefijo, err)
			}
		case ruta == "members" && acc == "remove":
			if objetivo != "" {
				copia.Miembros = sinElemento(copia.Miembros, objetivo)
				break
			}
			ids, err := leerMiembros(op.Valor)
			if err != nil {
				// remove de `members` sin valor y sin filtro: vaciar el grupo.
				copia.Miembros = nil
				break
			}
			for _, x := range ids {
				copia.Miembros = sinElemento(copia.Miembros, x)
			}
		case ruta == "members" && (acc == "add" || acc == "replace"):
			ids, err := leerMiembros(op.Valor)
			if err != nil {
				return Grupo{}, errValor("%s%v", prefijo, err)
			}
			if acc == "replace" {
				copia.Miembros = nil
			}
			copia.Miembros = append(copia.Miembros, ids...)
		case ruta == "id" || ruta == "meta" || ruta == "schemas":
			return Grupo{}, errMutabilidad("%sno se admite tocar %q en un grupo", prefijo, ruta)
		default:
			return Grupo{}, errRuta("%sno se sabe aplicar `%s` sobre %q en un grupo. Se "+
				"admiten displayName, externalId y members", prefijo, op.Op, op.Ruta)
		}
	}
	miembros, err := d.validarMiembrosBloqueado(copia.Miembros)
	if err != nil {
		return Grupo{}, err
	}
	copia.Miembros = miembros
	copia.Modificado = ahora
	*actual = copia
	return copia, nil
}

// valorDelFiltro saca el "id" de members[value eq "id"].
func valorDelFiltro(ruta string) string {
	i := strings.Index(ruta, "[")
	j := strings.LastIndex(ruta, "]")
	if i < 0 || j < i {
		return ""
	}
	dentro := ruta[i+1 : j]
	partes := strings.Fields(dentro)
	if len(partes) != 3 || normalizar(partes[0]) != "value" || normalizar(partes[1]) != "eq" {
		return ""
	}
	return strings.Trim(partes[2], `"`)
}

func leerMiembros(valor json.RawMessage) ([]string, error) {
	var refs []struct {
		Valor string `json:"value"`
	}
	if err := json.Unmarshal(valor, &refs); err == nil && len(refs) > 0 {
		out := make([]string, 0, len(refs))
		for _, r := range refs {
			if strings.TrimSpace(r.Valor) == "" {
				return nil, errValor("un miembro sin `value` no identifica a nadie")
			}
			out = append(out, r.Valor)
		}
		return out, nil
	}
	var una struct {
		Valor string `json:"value"`
	}
	if err := json.Unmarshal(valor, &una); err == nil && una.Valor != "" {
		return []string{una.Valor}, nil
	}
	return nil, errValor("`members` tiene que ser una lista de referencias {\"value\": \"<id>\"}")
}
