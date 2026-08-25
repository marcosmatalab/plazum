package scim

import (
	"errors"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"dutiq/puertos"
)

// Directorio es el almacen de usuarios, grupos y jerarquia.
//
// Es seguro para uso concurrente. El IdP aprovisiona en paralelo: Entra ID
// manda hasta cinco peticiones a la vez en un ciclo de sincronizacion, y dos
// altas simultaneas del mismo `userName` tienen que dar una y un 409, no dos.
type Directorio struct {
	sec puertos.Secretos

	mu        sync.RWMutex
	usuarios  map[string]*Usuario
	grupos    map[string]*Grupo
	porNombre map[string]string // userName normalizado -> id
	// manualPorEmpleado son las relaciones que declaro el operador. Viven
	// aparte de las del IdP, no mezcladas, para que siempre se pueda decir de
	// donde vino cada una y para que una no pise a la otra sin que se vea.
	manualPorEmpleado map[string]relacionManual
}

type relacionManual struct {
	manager string
	desde   time.Time
	// Autor es quien la declaro. Una jerarquia escrita a mano es una decision
	// de una persona, y en un producto de cumplimiento las decisiones llevan
	// nombre.
	autor string
}

// NuevoDirectorio construye el almacen. La fuente de aleatoriedad es un puerto
// porque los identificadores de recurso tienen que ser impredecibles en
// produccion y reproducibles en los tests.
func NuevoDirectorio(sec puertos.Secretos) (*Directorio, error) {
	if sec == nil {
		return nil, errors.New("scim: falta el puerto de Secretos. Los identificadores de " +
			"recurso son aleatoriedad criptografica: si se generan con un contador, se " +
			"adivinan, y adivinar un id es la mitad de un acceso indebido")
	}
	return &Directorio{
		sec:               sec,
		usuarios:          map[string]*Usuario{},
		grupos:            map[string]*Grupo{},
		porNombre:         map[string]string{},
		manualPorEmpleado: map[string]relacionManual{},
	}, nil
}

func (d *Directorio) nuevoID() (string, error) { return d.sec.Token(16) }

// ---------------------------------------------------------------------------
// Ciclo de vida del usuario
// ---------------------------------------------------------------------------

// Crear da de alta un usuario. Es el POST /Users.
func (d *Directorio) Crear(u Usuario, ahora time.Time) (Usuario, error) {
	if strings.TrimSpace(u.UserName) == "" {
		return Usuario{}, errValor("un usuario sin `userName` no se puede crear: es el " +
			"identificador con el que el IdP lo vuelve a encontrar")
	}
	id, err := d.nuevoID()
	if err != nil {
		return Usuario{}, errorf(http.StatusInternalServerError, "",
			"no hay aleatoriedad para el identificador del recurso: %v", err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if otro, ya := d.porNombre[normalizar(u.UserName)]; ya {
		return Usuario{}, errUnicidad("ya hay un usuario con `userName` %q (id %s). SCIM exige "+
			"que sea unico; si es un realta despues de un borrado, reactiva el existente con "+
			"PATCH `active: true` en vez de crear otro", u.UserName, otro)
	}
	if u.ManagerIdP != "" {
		if err := d.validarManagerBloqueado(id, u.ManagerIdP); err != nil {
			return Usuario{}, err
		}
	}
	u.ID = id
	u.Creado = ahora
	u.Modificado = ahora
	u.Borrado = time.Time{}
	copia := u
	d.usuarios[id] = &copia
	d.porNombre[normalizar(u.UserName)] = id
	return copia, nil
}

// Leer devuelve un usuario vivo. Un borrado no se devuelve: RFC 7644 dice que
// tras un DELETE el recurso no vuelve por GET.
func (d *Directorio) Leer(id string) (Usuario, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	u, ok := d.usuarios[id]
	if !ok || !u.Vivo() {
		return Usuario{}, errNoEncontrado("usuario", id)
	}
	return *u, nil
}

// Reemplazar es el PUT /Users/{id}: sustituye los atributos conocidos.
func (d *Directorio) Reemplazar(id string, nuevo Usuario, ahora time.Time) (Usuario, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	viejo, ok := d.usuarios[id]
	if !ok || !viejo.Vivo() {
		return Usuario{}, errNoEncontrado("usuario", id)
	}
	if strings.TrimSpace(nuevo.UserName) == "" {
		return Usuario{}, errValor("un PUT no puede dejar el `userName` vacio")
	}
	if otro, ya := d.porNombre[normalizar(nuevo.UserName)]; ya && otro != id {
		return Usuario{}, errUnicidad("el `userName` %q ya es de otro usuario (id %s)",
			nuevo.UserName, otro)
	}
	if nuevo.ManagerIdP != "" {
		if err := d.validarManagerBloqueado(id, nuevo.ManagerIdP); err != nil {
			return Usuario{}, err
		}
	}
	delete(d.porNombre, normalizar(viejo.UserName))
	nuevo.ID = id
	nuevo.Creado = viejo.Creado
	nuevo.Modificado = ahora
	nuevo.Borrado = time.Time{}
	*viejo = nuevo
	d.porNombre[normalizar(nuevo.UserName)] = id
	return *viejo, nil
}

// Borrar es el DELETE /Users/{id}. Pone lapida, no olvida.
//
// El recurso deja de existir para SCIM (no vuelve por GET ni por LIST, y su
// `userName` queda libre) y sigue existiendo para el producto, que necesita
// poder decir "esta obligacion la tenia alguien a quien borraron del IdP el
// dia X". Una obligacion sin responsable es un riesgo, no un problema resuelto.
func (d *Directorio) Borrar(id string, ahora time.Time) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	u, ok := d.usuarios[id]
	if !ok || !u.Vivo() {
		return errNoEncontrado("usuario", id)
	}
	u.Borrado = ahora
	u.Modificado = ahora
	u.Activo = false
	delete(d.porNombre, normalizar(u.UserName))
	// Se quitan sus pertenencias a grupos: un grupo que lista a un borrado es
	// una lista de correos rota.
	for _, g := range d.grupos {
		g.Miembros = sinElemento(g.Miembros, id)
	}
	return nil
}

// Historico devuelve un usuario aunque este borrado. Es lo que hace posible
// ensenar una obligacion huerfana con el nombre de quien la tenia en vez de con
// un identificador que no resuelve.
func (d *Directorio) Historico(id string) (Usuario, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	u, ok := d.usuarios[id]
	if !ok {
		return Usuario{}, false
	}
	return *u, true
}

// Filtro es lo que admite Listar.
type Filtro struct {
	// Atributo es el nombre del atributo, ya normalizado. Vacio significa
	// listar todo.
	Atributo string
	// Valor es lo que tiene que valer.
	Valor string
	// Desde y Cuantos son la paginacion de SCIM (startIndex, base 1, y count).
	Desde   int
	Cuantos int
}

// MaxPorPagina acota lo que se devuelve de una vez. Sin tope, un `count=100000`
// convierte una consulta en una fotocopia de la plantilla entera.
const MaxPorPagina = 200

// atributosFiltrables son los unicos sobre los que se sabe filtrar. Cualquier
// otro da error en vez de devolver la lista entera: un filtro que se ignora en
// silencio hace que el IdP concluya lo contrario de lo que pregunto, y entonces
// crea duplicados.
var atributosFiltrables = map[string]bool{
	"username": true, "externalid": true, "displayname": true,
}

// Listar devuelve los usuarios vivos que casan con el filtro, ordenados por id
// para que la paginacion sea estable.
func (d *Directorio) Listar(f Filtro) ([]Usuario, int, error) {
	if f.Atributo != "" && !atributosFiltrables[f.Atributo] {
		return nil, 0, errorf(http.StatusBadRequest, "invalidFilter",
			"no se sabe filtrar por %q. Se admite `eq` sobre userName, externalId y "+
				"displayName, que es lo que usan los IdP para comprobar si un recurso ya "+
				"existe. Se prefiere decirlo a devolver la lista entera, porque un filtro "+
				"ignorado hace que el IdP concluya lo contrario de lo que pregunto", f.Atributo)
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	var out []Usuario
	for _, u := range d.usuarios {
		if !u.Vivo() {
			continue
		}
		if f.Atributo != "" && !casa(*u, f.Atributo, f.Valor) {
			continue
		}
		out = append(out, *u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	total := len(out)
	return paginar(out, f), total, nil
}

func casa(u Usuario, atributo, valor string) bool {
	switch atributo {
	case "username":
		return normalizar(u.UserName) == normalizar(valor)
	case "externalid":
		return u.ExternalID == valor
	case "displayname":
		return normalizar(u.Mostrar) == normalizar(valor)
	}
	return false
}

func paginar[T any](xs []T, f Filtro) []T {
	desde := f.Desde
	if desde < 1 {
		desde = 1
	}
	if desde > len(xs) {
		return nil
	}
	xs = xs[desde-1:]
	cuantos := f.Cuantos
	if cuantos <= 0 || cuantos > MaxPorPagina {
		cuantos = MaxPorPagina
	}
	if cuantos < len(xs) {
		xs = xs[:cuantos]
	}
	return xs
}

func sinElemento(xs []string, x string) []string {
	out := xs[:0]
	for _, v := range xs {
		if v != x {
			out = append(out, v)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Admision: quien puede entrar
// ---------------------------------------------------------------------------

// ErrNoAdmitido es el rechazo de alguien que el IdP autentica y aqui no entra.
var ErrNoAdmitido = errors.New("cuenta no admitida")

// PuedeEntrar dice si un sujeto autenticado por el IdP puede abrir sesion.
//
// Se enchufa en el campo Admision del autenticador de OIDC. Es lo que convierte
// "desactivado en el IdP" en "no entra": sin esto, desactivar a alguien en
// Entra ID lo deja fuera de Entra ID y dentro de dutiq, que es la mitad del
// offboarding y la peor.
//
// El casamiento es por `externalId` primero (que es donde Entra ID y Okta ponen
// el identificador estable del usuario, el mismo que va en el `sub` del ID
// token) y por `userName` despues.
func (d *Directorio) PuedeEntrar(sujeto, correo string) error {
	u, ok := d.PorSujeto(sujeto, correo)
	if !ok {
		return errors.New("no hay ninguna cuenta aprovisionada para ese usuario. Si el " +
			"aprovisionamiento SCIM esta recien configurado, espera al primer ciclo del IdP " +
			"o lanza una sincronizacion a mano")
	}
	if !u.Vivo() {
		return errors.New("la cuenta se borro del IdP el " + u.Borrado.Format("2006-01-02") +
			". Si tiene que volver a entrar, vuelve a asignarle la aplicacion en el IdP")
	}
	if !u.Activo {
		return errors.New("la cuenta esta desactivada en el IdP (`active: false`). Se " +
			"reactiva desde el IdP, no desde dutiq: si se pudiera reactivar aqui, el " +
			"offboarding de la empresa dejaria de ser el offboarding")
	}
	return nil
}

// PorSujeto busca al usuario por el identificador del IdP, y si no, por correo.
func (d *Directorio) PorSujeto(sujeto, correo string) (Usuario, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	for _, u := range d.usuarios {
		if u.ExternalID != "" && u.ExternalID == sujeto {
			return *u, true
		}
	}
	if id, ok := d.porNombre[normalizar(sujeto)]; ok {
		return *d.usuarios[id], true
	}
	if correo != "" {
		if id, ok := d.porNombre[normalizar(correo)]; ok {
			return *d.usuarios[id], true
		}
		for _, u := range d.usuarios {
			if u.Vivo() && normalizar(u.CorreoPrincipal()) == normalizar(correo) {
				return *u, true
			}
		}
	}
	return Usuario{}, false
}

// ---------------------------------------------------------------------------
// Grupos
// ---------------------------------------------------------------------------

// CrearGrupo da de alta un grupo.
func (d *Directorio) CrearGrupo(g Grupo, ahora time.Time) (Grupo, error) {
	if strings.TrimSpace(g.Mostrar) == "" {
		return Grupo{}, errValor("un grupo sin `displayName` no se puede crear")
	}
	id, err := d.nuevoID()
	if err != nil {
		return Grupo{}, errorf(http.StatusInternalServerError, "", "no hay aleatoriedad: %v", err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, otro := range d.grupos {
		if normalizar(otro.Mostrar) == normalizar(g.Mostrar) {
			return Grupo{}, errUnicidad("ya hay un grupo llamado %q (id %s)", g.Mostrar, otro.ID)
		}
	}
	miembros, err := d.validarMiembrosBloqueado(g.Miembros)
	if err != nil {
		return Grupo{}, err
	}
	g.ID = id
	g.Miembros = miembros
	g.Creado = ahora
	g.Modificado = ahora
	copia := g
	d.grupos[id] = &copia
	return copia, nil
}

// LeerGrupo devuelve un grupo.
func (d *Directorio) LeerGrupo(id string) (Grupo, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	g, ok := d.grupos[id]
	if !ok {
		return Grupo{}, errNoEncontrado("grupo", id)
	}
	return *g, nil
}

// BorrarGrupo lo quita. Un grupo no deja lapida: no es titular de nada.
func (d *Directorio) BorrarGrupo(id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.grupos[id]; !ok {
		return errNoEncontrado("grupo", id)
	}
	delete(d.grupos, id)
	return nil
}

// ListarGrupos devuelve los grupos que casan con el filtro.
func (d *Directorio) ListarGrupos(f Filtro) ([]Grupo, int, error) {
	if f.Atributo != "" && f.Atributo != "displayname" && f.Atributo != "externalid" {
		return nil, 0, errorf(http.StatusBadRequest, "invalidFilter",
			"no se sabe filtrar grupos por %q; se admite `eq` sobre displayName y externalId",
			f.Atributo)
	}
	d.mu.RLock()
	defer d.mu.RUnlock()
	var out []Grupo
	for _, g := range d.grupos {
		switch f.Atributo {
		case "":
		case "displayname":
			if normalizar(g.Mostrar) != normalizar(f.Valor) {
				continue
			}
		case "externalid":
			if g.ExternalID != f.Valor {
				continue
			}
		}
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return paginar(out, f), len(out), nil
}

// GruposDe devuelve los grupos a los que pertenece un usuario. `groups` es de
// solo lectura en el esquema User: se calcula, no se escribe.
func (d *Directorio) GruposDe(idUsuario string) []Grupo {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var out []Grupo
	for _, g := range d.grupos {
		for _, m := range g.Miembros {
			if m == idUsuario {
				out = append(out, *g)
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// validarMiembrosBloqueado comprueba que todos los miembros existen y estan
// vivos, y quita duplicados. Se llama con el candado tomado.
func (d *Directorio) validarMiembrosBloqueado(ids []string) ([]string, error) {
	visto := map[string]bool{}
	var out []string
	for _, id := range ids {
		if visto[id] {
			continue
		}
		u, ok := d.usuarios[id]
		if !ok || !u.Vivo() {
			return nil, errValor("el miembro %q no existe. Un grupo con miembros que no "+
				"existen es una lista de correos rota que nadie mira hasta que falla un "+
				"escalado", id)
		}
		visto[id] = true
		out = append(out, id)
	}
	return out, nil
}
