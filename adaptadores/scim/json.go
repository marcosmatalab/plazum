package scim

import (
	"encoding/json"
	"strings"
	"time"
)

// Este fichero es la representacion SCIM 2.0 de los recursos: lo que sale por
// el cable y lo que entra.
//
// Se escribe a mano en vez de con etiquetas de struct sobre el modelo interno
// por una razon concreta: el modelo interno tiene campos que NO deben salir
// (la lapida de borrado) y el formato de fuera tiene una extension que no es un
// campo sino un objeto anidado bajo un URN. Mezclar las dos cosas en un tipo
// termina publicando por accidente lo que no toca.

// Meta es el `meta` de RFC 7643.
type Meta struct {
	ResourceType string `json:"resourceType"`
	Created      string `json:"created,omitempty"`
	LastModified string `json:"lastModified,omitempty"`
	Location     string `json:"location,omitempty"`
	Version      string `json:"version,omitempty"`
}

// referencia es el objeto {value, $ref, display} que SCIM usa para apuntar a
// otro recurso.
type referencia struct {
	Valor   string `json:"value,omitempty"`
	Ref     string `json:"$ref,omitempty"`
	Mostrar string `json:"display,omitempty"`
}

// empresa es la extension enterprise: el objeto que cuelga del URN.
type empresa struct {
	NumeroEmpleado string      `json:"employeeNumber,omitempty"`
	Departamento   string      `json:"department,omitempty"`
	Manager        *referencia `json:"manager,omitempty"`
}

// usuarioJSON es el recurso User tal y como viaja.
type usuarioJSON struct {
	Esquemas   []string     `json:"schemas"`
	ID         string       `json:"id,omitempty"`
	ExternalID string       `json:"externalId,omitempty"`
	UserName   string       `json:"userName"`
	Nombre     *Nombre      `json:"name,omitempty"`
	Mostrar    string       `json:"displayName,omitempty"`
	Titulo     string       `json:"title,omitempty"`
	Correos    []Correo     `json:"emails,omitempty"`
	Activo     bool         `json:"active"`
	Grupos     []referencia `json:"groups,omitempty"`
	Empresa    *empresa     `json:"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User,omitempty"`
	Meta       *Meta        `json:"meta,omitempty"`
}

// AUsuarioJSON serializa un usuario en el formato de SCIM.
//
// base es el prefijo de las URL de `location` y de `$ref`; vacio las omite. Los
// grupos se calculan, no se guardan: `groups` es de solo lectura en el esquema.
func (d *Directorio) AUsuarioJSON(u Usuario, base string) ([]byte, error) {
	esquemas := []string{EsquemaUsuario}
	var ext *empresa
	if u.ManagerIdP != "" || u.Departamento != "" || u.NumeroEmpleado != "" {
		ext = &empresa{NumeroEmpleado: u.NumeroEmpleado, Departamento: u.Departamento}
		if u.ManagerIdP != "" {
			ref := &referencia{Valor: u.ManagerIdP}
			if jefe, ok := d.Historico(u.ManagerIdP); ok {
				ref.Mostrar = jefe.Mostrar
			}
			if base != "" {
				ref.Ref = base + "/Users/" + u.ManagerIdP
			}
			ext.Manager = ref
		}
		esquemas = append(esquemas, EsquemaEmpresa)
	}
	j := usuarioJSON{
		Esquemas:   esquemas,
		ID:         u.ID,
		ExternalID: u.ExternalID,
		UserName:   u.UserName,
		Mostrar:    u.Mostrar,
		Titulo:     u.Titulo,
		Correos:    u.Correos,
		Activo:     u.Activo,
		Empresa:    ext,
		Meta: &Meta{
			ResourceType: "User",
			Created:      marcaTiempo(u.Creado),
			LastModified: marcaTiempo(u.Modificado),
			Version:      version(u.Modificado),
		},
	}
	if u.Nombre != (Nombre{}) {
		n := u.Nombre
		j.Nombre = &n
	}
	if base != "" {
		j.Meta.Location = base + "/Users/" + u.ID
	}
	for _, g := range d.GruposDe(u.ID) {
		r := referencia{Valor: g.ID, Mostrar: g.Mostrar}
		if base != "" {
			r.Ref = base + "/Groups/" + g.ID
		}
		j.Grupos = append(j.Grupos, r)
	}
	return json.MarshalIndent(j, "", "  ")
}

// DeUsuarioJSON lee un recurso User del cuerpo de una peticion.
//
// Los atributos que este modelo no guarda se descartan, y los prohibidos se
// rechazan: si un POST pudiera traer `roles` y colarse, la lista blanca del
// PATCH no serviria de nada, porque bastaria borrar y volver a crear.
func DeUsuarioJSON(cuerpo []byte) (Usuario, error) {
	var crudo map[string]json.RawMessage
	if err := json.Unmarshal(cuerpo, &crudo); err != nil {
		return Usuario{}, errSintaxis("el cuerpo no es un objeto JSON: %v", err)
	}
	for clave := range crudo {
		raiz := normalizarRuta(clave)
		if i := strings.Index(raiz, "."); i > 0 {
			raiz = raiz[:i]
		}
		if motivo, prohibido := atributosProhibidos[raiz]; prohibido {
			// `id`, `meta` y `schemas` los manda todo el mundo en un POST y no
			// son un intento de escalar nada: se ignoran. Los que SI son un
			// intento de tocar privilegios se rechazan.
			if raiz == "id" || raiz == "meta" || raiz == "schemas" || raiz == "groups" {
				continue
			}
			return Usuario{}, errMutabilidad("el cuerpo trae %q y no se admite: %s", clave, motivo)
		}
	}
	var j usuarioJSON
	if err := json.Unmarshal(cuerpo, &j); err != nil {
		return Usuario{}, errSintaxis("el cuerpo no es un recurso User valido: %v", err)
	}
	u := Usuario{
		ExternalID: strings.TrimSpace(j.ExternalID),
		UserName:   strings.TrimSpace(j.UserName),
		Mostrar:    strings.TrimSpace(j.Mostrar),
		Titulo:     strings.TrimSpace(j.Titulo),
		Correos:    j.Correos,
		Activo:     j.Activo,
	}
	if j.Nombre != nil {
		u.Nombre = *j.Nombre
	}
	// `active` no viene en muchos POST de alta. La ausencia se toma como
	// ACTIVO, que es lo que significa dar de alta a alguien; tomarla como
	// inactivo dejaria a todo el mundo sin poder entrar el primer dia.
	if _, hay := crudo["active"]; !hay {
		u.Activo = true
	}
	if j.Empresa != nil {
		u.Departamento = strings.TrimSpace(j.Empresa.Departamento)
		u.NumeroEmpleado = strings.TrimSpace(j.Empresa.NumeroEmpleado)
		if j.Empresa.Manager != nil {
			u.ManagerIdP = strings.TrimSpace(j.Empresa.Manager.Valor)
		}
	}
	return u, nil
}

// grupoJSON es el recurso Group tal y como viaja.
type grupoJSON struct {
	Esquemas   []string     `json:"schemas"`
	ID         string       `json:"id,omitempty"`
	ExternalID string       `json:"externalId,omitempty"`
	Mostrar    string       `json:"displayName"`
	Miembros   []referencia `json:"members,omitempty"`
	Meta       *Meta        `json:"meta,omitempty"`
}

// AGrupoJSON serializa un grupo.
func (d *Directorio) AGrupoJSON(g Grupo, base string) ([]byte, error) {
	j := grupoJSON{
		Esquemas:   []string{EsquemaGrupo},
		ID:         g.ID,
		ExternalID: g.ExternalID,
		Mostrar:    g.Mostrar,
		Meta: &Meta{
			ResourceType: "Group",
			Created:      marcaTiempo(g.Creado),
			LastModified: marcaTiempo(g.Modificado),
			Version:      version(g.Modificado),
		},
	}
	if base != "" {
		j.Meta.Location = base + "/Groups/" + g.ID
	}
	for _, m := range g.Miembros {
		r := referencia{Valor: m}
		if u, ok := d.Historico(m); ok {
			r.Mostrar = u.Mostrar
		}
		if base != "" {
			r.Ref = base + "/Users/" + m
		}
		j.Miembros = append(j.Miembros, r)
	}
	return json.MarshalIndent(j, "", "  ")
}

// DeGrupoJSON lee un recurso Group.
func DeGrupoJSON(cuerpo []byte) (Grupo, error) {
	var j grupoJSON
	if err := json.Unmarshal(cuerpo, &j); err != nil {
		return Grupo{}, errSintaxis("el cuerpo no es un recurso Group valido: %v", err)
	}
	g := Grupo{
		ExternalID: strings.TrimSpace(j.ExternalID),
		Mostrar:    strings.TrimSpace(j.Mostrar),
	}
	for _, m := range j.Miembros {
		if strings.TrimSpace(m.Valor) == "" {
			return Grupo{}, errValor("un miembro sin `value` no identifica a nadie")
		}
		g.Miembros = append(g.Miembros, m.Valor)
	}
	return g, nil
}

// ListaJSON envuelve una lista en el ListResponse de SCIM.
func ListaJSON(recursos []json.RawMessage, total, desde int) ([]byte, error) {
	if desde < 1 {
		desde = 1
	}
	if recursos == nil {
		recursos = []json.RawMessage{}
	}
	return json.MarshalIndent(map[string]any{
		"schemas":      []string{EsquemaLista},
		"totalResults": total,
		"startIndex":   desde,
		"itemsPerPage": len(recursos),
		"Resources":    recursos,
	}, "", "  ")
}

// ErrorJSON escribe un error en el formato de SCIM.
func ErrorJSON(e *Error) []byte {
	m := map[string]any{
		"schemas": []string{EsquemaError},
		"status":  itoa(e.Estado),
		"detail":  e.Detalle,
	}
	if e.Tipo != "" {
		m["scimType"] = e.Tipo
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		// Un error al serializar un error no puede dejar al cliente sin
		// cuerpo: se responde algo valido a mano.
		return []byte(`{"schemas":["` + EsquemaError + `"],"status":"500",` +
			`"detail":"no se pudo serializar el error"}`)
	}
	return b
}

func marcaTiempo(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// version es el ETag del recurso. SCIM lo usa para el control de concurrencia
// optimista; aqui se deriva de la ultima modificacion, que es suficiente para
// que un IdP detecte que algo cambio.
func version(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return `W/"` + marcaTiempo(t) + `"`
}
