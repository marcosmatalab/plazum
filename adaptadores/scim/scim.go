// Package scim es el aprovisionamiento de usuarios y grupos desde el IdP, con
// la extension enterprise y su atributo `manager`.
//
// # Por que el manager no es un capricho
//
// Es lo que hace posible el escalado jerarquico: cuando una obligacion vence y
// su responsable no responde, se sube a su jefe. Sin jerarquia, el escalado es
// una lista de correos escrita a mano que se queda obsoleta el primer dia. Y
// por eso existe tambien el mapeo manual: la mitad de los clientes no publica
// `manager` en su IdP, y un producto que solo funciona con el IdP perfecto no
// funciona.
//
// # Lo que este paquete es y lo que no
//
// Es el MODELO y el ALMACEN: usuarios, grupos, ciclo de vida, jerarquia. No
// habla HTTP: el servidor vive en superficies/scim y es un http.Handler
// autonomo. La separacion no es estetica, es lo que permite probar el ciclo de
// vida y los ciclos de la jerarquia sin levantar nada.
//
// # El instante entra como dato
//
// Todas las operaciones que fechan algo reciben `ahora`. Es la misma disciplina
// del nucleo, y aqui hace falta por lo mismo: un usuario desactivado el martes
// tiene que poder probarse sin esperar al martes.
//
// # Donde nos desviamos de RFC 7643 y 7644, dicho en voz alta
//
//   - El filtro implementa `eq` sobre `userName`, `externalId` y `displayName`,
//     y nada mas. Es lo que mandan Entra ID y Okta para comprobar si un recurso
//     ya existe; el resto de la gramatica de filtros (`co`, `sw`, `and`, `or`,
//     parentesis) devuelve un error de filtro no admitido en vez de fingir que
//     filtro. Un filtro que se ignora en silencio devuelve la lista entera y el
//     IdP concluye lo contrario de lo que pregunto.
//   - `PUT` sustituye los atributos que este modelo conoce; los que no conoce
//     no se guardan. Estan enumerados en atributosIgnorables, no descartados a
//     ciegas.
//   - No hay `/Me`, ni `/Bulk`, ni ordenacion por `sortBy`. Ningun IdP de los
//     que importan los usa para aprovisionar.
//   - `password` se RECHAZA. dutiq no tiene contrasenas propias: la
//     autenticacion es OIDC. Aceptar una contrasena del IdP crearia una segunda
//     via de entrada que nadie vigila.
//   - `roles` y `entitlements` se RECHAZAN. El rol dentro de dutiq se asigna
//     dentro de dutiq. Si el IdP pudiera mandarlo, quien controle el
//     aprovisionamiento controla los privilegios, y el token de SCIM pasaria a
//     valer lo que la cuenta de administrador.
package scim

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Los identificadores de esquema de SCIM 2.0 (RFC 7643 y 7644).
const (
	EsquemaUsuario = "urn:ietf:params:scim:schemas:core:2.0:User"
	EsquemaGrupo   = "urn:ietf:params:scim:schemas:core:2.0:Group"
	EsquemaEmpresa = "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"
	EsquemaError   = "urn:ietf:params:scim:api:messages:2.0:Error"
	EsquemaLista   = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	EsquemaParcheo = "urn:ietf:params:scim:api:messages:2.0:PatchOp"
)

// TipoContenido es el que exige SCIM.
const TipoContenido = "application/scim+json"

// Error es un fallo de SCIM con su codigo HTTP y su `scimType`.
//
// Lleva los dos porque el IdP los usa para decidir: Okta reintenta un 409 de
// `uniqueness` y no reintenta un 400 de `invalidValue`. Devolver 400 para todo
// convierte un choque de nombres recuperable en un fallo permanente de
// aprovisionamiento que alguien tiene que ir a desatascar a mano.
type Error struct {
	// Estado es el codigo HTTP.
	Estado int
	// Tipo es el `scimType` de RFC 7644 seccion 3.12. Vacio si no aplica.
	Tipo string
	// Detalle dice que paso y, cuando se puede, como se arregla.
	Detalle string
}

func (e *Error) Error() string {
	if e.Tipo == "" {
		return fmt.Sprintf("scim %d: %s", e.Estado, e.Detalle)
	}
	return fmt.Sprintf("scim %d (%s): %s", e.Estado, e.Tipo, e.Detalle)
}

func errorf(estado int, tipo, formato string, args ...any) *Error {
	return &Error{Estado: estado, Tipo: tipo, Detalle: fmt.Sprintf(formato, args...)}
}

// Los constructores de los errores que se repiten, para que el `scimType` sea
// el mismo desde todos los sitios.
func errNoEncontrado(que, id string) *Error {
	return errorf(http.StatusNotFound, "", "no hay ningun %s con id %q", que, id)
}

func errValor(formato string, args ...any) *Error {
	return errorf(http.StatusBadRequest, "invalidValue", formato, args...)
}

func errSintaxis(formato string, args ...any) *Error {
	return errorf(http.StatusBadRequest, "invalidSyntax", formato, args...)
}

func errRuta(formato string, args ...any) *Error {
	return errorf(http.StatusBadRequest, "invalidPath", formato, args...)
}

func errMutabilidad(formato string, args ...any) *Error {
	return errorf(http.StatusBadRequest, "mutability", formato, args...)
}

func errUnicidad(formato string, args ...any) *Error {
	return errorf(http.StatusConflict, "uniqueness", formato, args...)
}

// Origen dice de donde salio una relacion de jerarquia.
//
// Existe porque el mapeo manual NO puede ser un segundo sistema paralelo:
// misma estructura, otro origen, y que se vea de donde vino cada relacion. Un
// operador que mira el organigrama tiene que poder distinguir lo que dice su
// IdP de lo que escribio el mismo hace ocho meses.
type Origen int

const (
	// OrigenNinguno: no hay relacion.
	OrigenNinguno Origen = iota
	// OrigenIdP: vino en el atributo `manager` de la extension enterprise.
	OrigenIdP
	// OrigenManual: lo declaro el operador dentro de dutiq.
	OrigenManual
)

func (o Origen) String() string {
	switch o {
	case OrigenIdP:
		return "idp"
	case OrigenManual:
		return "manual"
	case OrigenNinguno:
		return "ninguno"
	}
	return "desconocido"
}

// Nombre es el `name` de RFC 7643.
type Nombre struct {
	Formateado string `json:"formatted,omitempty"`
	Familia    string `json:"familyName,omitempty"`
	Pila       string `json:"givenName,omitempty"`
}

// Correo es un elemento de `emails`.
type Correo struct {
	Valor     string `json:"value"`
	Tipo      string `json:"type,omitempty"`
	Principal bool   `json:"primary,omitempty"`
}

// Usuario es el recurso User con lo que el producto usa de verdad.
//
// No estan todos los atributos del esquema a proposito: guardar lo que no se usa
// es guardar datos personales sin base para hacerlo, y este producto vende
// cumplimiento.
type Usuario struct {
	ID         string
	ExternalID string
	UserName   string
	Nombre     Nombre
	Mostrar    string
	Correos    []Correo
	Titulo     string
	// Activo es el `active` del esquema. Un usuario inactivo no entra, y sus
	// obligaciones quedan VISIBLES como huerfanas.
	Activo bool
	// ManagerIdP es el `value` del atributo `manager` de la extension
	// enterprise: el id del jefe SEGUN EL IdP.
	ManagerIdP     string
	Departamento   string
	NumeroEmpleado string
	Creado         time.Time
	Modificado     time.Time
	// Borrado es la lapida. Cero significa vivo.
	//
	// Un DELETE de SCIM no borra el rastro, lo marca. RFC 7644 exige que el
	// recurso deje de devolverse por GET y por LIST, y eso se cumple; lo que
	// no exige es olvidar que existio, y olvidarlo tiene un coste concreto:
	// una obligacion cuyo responsable se borro pasaria a apuntar a un id que
	// no resuelve, y en pantalla se leeria como un error del sistema en vez de
	// como lo que es, una obligacion sin dueno.
	Borrado time.Time
}

// Vivo dice si el usuario sigue existiendo para SCIM.
func (u Usuario) Vivo() bool { return u.Borrado.IsZero() }

// CorreoPrincipal devuelve el correo marcado como principal, o el primero.
func (u Usuario) CorreoPrincipal() string {
	for _, c := range u.Correos {
		if c.Principal {
			return c.Valor
		}
	}
	if len(u.Correos) > 0 {
		return u.Correos[0].Valor
	}
	return ""
}

// Grupo es el recurso Group.
type Grupo struct {
	ID         string
	ExternalID string
	Mostrar    string
	// Miembros son ids de Usuario. Se guardan como referencias y se validan
	// contra el directorio: un grupo con miembros que no existen es una lista
	// de correos rota que nadie mira hasta que falla un escalado.
	Miembros   []string
	Creado     time.Time
	Modificado time.Time
}

// normalizar deja un atributo listo para comparar sin importar mayusculas ni
// espacios. SCIM declara `userName` como caseExact false.
func normalizar(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
