// Package scim es el servidor SCIM 2.0: un http.Handler autonomo.
//
// # Autonomo a proposito
//
// No conoce ningun router compartido y no registra rutas en ningun sitio.
// Entrega un http.Handler y quien construya el servidor decide donde lo cuelga.
// La etapa 2 se esta construyendo en varios frentes a la vez, y un router
// compartido seria el punto exacto por el que se romperian entre ellos.
//
// # La autenticacion no es opcional, es estructural
//
// Este endpoint crea y borra usuarios. Si se pudiera llamar sin credencial,
// seria una puerta trasera con forma de estandar. Por eso el constructor FALLA
// si no se le da un token: no hay forma de construir un servidor SCIM abierto,
// ni siquiera por descuido en un fichero de configuracion.
//
// # Lo que no hace, y de quien es
//
// Ni rate limiting ni cabeceras de seguridad: eso es el middleware del frente
// que construye el servidor, se aplica a todo por igual y aqui seria una
// segunda implementacion divergente. Lo que si hace, porque es suyo, es acotar
// el tamano del cuerpo: un cuerpo sin limite es una fuga de memoria por una
// ruta que exige credencial pero que se llama antes de leerla.
package scim

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	adaptador "plazum/adaptadores/scim"
)

// MaxCuerpoPorDefecto acota el cuerpo de una peticion.
//
// Un recurso User ronda el kilobyte; un PATCH de grupo con quinientos miembros
// ronda los cuarenta. Un mebibyte es holgado y cierra la puerta a que alguien
// nos mande un gigabyte por una ruta que, aunque exige credencial, tiene que
// leer antes de poder decir que no.
const MaxCuerpoPorDefecto = 1 << 20

// LongitudMinimaToken es lo minimo que se acepta como credencial.
//
// Treinta y dos caracteres no es una cifra ritual: por debajo de eso, un token
// generado a mano por un administrador con prisa se adivina, y este endpoint da
// de alta y de baja a personas.
const LongitudMinimaToken = 32

// Opciones configura el servidor.
type Opciones struct {
	// Token es la credencial que el IdP manda como `Authorization: Bearer`.
	// Obligatoria. Se guarda solo su resumen: si alguien vuelca la memoria del
	// proceso o el servidor en un log, no sale el token.
	Token string
	// Base es el prefijo de las rutas. Vacio significa /scim/v2.
	Base string
	// MaxCuerpo acota la peticion. Cero significa MaxCuerpoPorDefecto.
	MaxCuerpo int64
	// Ahora es el reloj. Nil significa time.Now. Se inyecta para poder probar
	// el ciclo de vida sin esperar.
	Ahora func() time.Time
}

// Servidor es el http.Handler de SCIM.
type Servidor struct {
	dir       *adaptador.Directorio
	resumen   [32]byte
	base      string
	maxCuerpo int64
	ahora     func() time.Time
	mux       *http.ServeMux

	mu  sync.Mutex
	act Actividad
}

// Actividad es lo que hace falta para responder "¿esta funcionando el SCIM o
// solo lo espero?".
//
// Es una pregunta de comprador, no de desarrollador. Un aprovisionamiento que
// nunca se conecto y uno que funciona se parecen mucho desde fuera: en los dos
// casos no pasa nada raro. La diferencia solo se ve mirando si alguna vez llego
// una peticion.
type Actividad struct {
	// UltimaPeticion es cuando llego la ultima, sea correcta o no. Cero
	// significa que el IdP no ha llamado nunca.
	UltimaPeticion time.Time
	// UltimaCorrecta es cuando llego la ultima que se acepto.
	UltimaCorrecta time.Time
	// Peticiones y Rechazos cuentan desde el arranque.
	Peticiones int
	Rechazos   int
	// UltimoRechazo dice que fallo, sin secretos y sin cuerpos.
	UltimoRechazo string
	// RechazosDeCredencial se cuenta aparte: un goteo aqui es un token mal
	// pegado, y un chorro es alguien probando.
	RechazosDeCredencial int
}

// Nuevo construye el servidor. Falla si falta el directorio o la credencial.
func Nuevo(dir *adaptador.Directorio, op Opciones) (*Servidor, error) {
	if dir == nil {
		return nil, errors.New("scim: falta el directorio")
	}
	if strings.TrimSpace(op.Token) == "" {
		// El mensaje NO nombra ningun comando para generarlo, y es a
		// proposito: `plazum scim token` todavia no existe (ver
		// docs/pendientes.md). Un error que manda a ejecutar algo que no esta
		// es peor que uno que no dice nada, porque quema la confianza en el
		// resto de los mensajes.
		return nil, fmt.Errorf("scim: falta el token de aprovisionamiento. Este endpoint crea "+
			"y borra usuarios: sin credencial seria una puerta trasera con forma de estandar. "+
			"Pon una cadena aleatoria de al menos %d caracteres en la configuracion de la "+
			"instancia y pega la misma en el aprovisionamiento del IdP", LongitudMinimaToken)
	}
	if len(op.Token) < LongitudMinimaToken {
		return nil, fmt.Errorf("scim: el token ocupa %d caracteres y el minimo es %d. Un token "+
			"corto se adivina, y con el se dan de alta y de baja personas", len(op.Token),
			LongitudMinimaToken)
	}
	base := strings.TrimSuffix(op.Base, "/")
	if base == "" {
		base = "/scim/v2"
	}
	maxCuerpo := op.MaxCuerpo
	if maxCuerpo <= 0 {
		maxCuerpo = MaxCuerpoPorDefecto
	}
	ahora := op.Ahora
	if ahora == nil {
		ahora = time.Now
	}
	s := &Servidor{
		dir:       dir,
		resumen:   sha256.Sum256([]byte(op.Token)),
		base:      base,
		maxCuerpo: maxCuerpo,
		ahora:     ahora,
	}
	s.mux = http.NewServeMux()
	s.rutas()
	return s, nil
}

// Base devuelve el prefijo bajo el que cuelga. Lo necesita quien lo monta.
func (s *Servidor) Base() string { return s.base }

func (s *Servidor) rutas() {
	r := func(patron string, h func(http.ResponseWriter, *http.Request) error) {
		s.mux.HandleFunc(patron, s.envolver(h))
	}
	r("GET "+s.base+"/Users", s.listarUsuarios)
	r("POST "+s.base+"/Users", s.crearUsuario)
	r("GET "+s.base+"/Users/{id}", s.leerUsuario)
	r("PUT "+s.base+"/Users/{id}", s.reemplazarUsuario)
	r("PATCH "+s.base+"/Users/{id}", s.parchearUsuario)
	r("DELETE "+s.base+"/Users/{id}", s.borrarUsuario)

	r("GET "+s.base+"/Groups", s.listarGrupos)
	r("POST "+s.base+"/Groups", s.crearGrupo)
	r("GET "+s.base+"/Groups/{id}", s.leerGrupo)
	r("PATCH "+s.base+"/Groups/{id}", s.parchearGrupo)
	r("DELETE "+s.base+"/Groups/{id}", s.borrarGrupo)

	r("GET "+s.base+"/ServiceProviderConfig", s.configuracion)
	r("GET "+s.base+"/ResourceTypes", s.tiposDeRecurso)
	r("GET "+s.base+"/Schemas", s.esquemas)
}

func (s *Servidor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Todo lo que no case con una ruta conocida responde en formato SCIM, no
	// con el 404 de texto plano de Go: el IdP parsea la respuesta y un cuerpo
	// que no es JSON le sale por una traza que nadie entiende.
	//
	// La consulta se hace con Handler y el despacho con ServeHTTP, y son dos
	// llamadas a proposito. Handler resuelve el patron pero NO rellena los
	// comodines de la ruta: si se despachara con el handler que devuelve,
	// r.PathValue("id") sale vacio y todas las rutas con {id} contestan 404.
	//
	// Merece la pena dejarlo escrito porque el fallo era invisible: los tests
	// que esperaban 404 (leer un usuario borrado, por ejemplo) seguian en
	// verde, con el codigo correcto y el motivo equivocado. Lo destapo un test
	// que esperaba un 200.
	if _, patron := s.mux.Handler(r); patron == "" {
		s.registrar(false, "ruta desconocida "+r.Method+" "+r.URL.Path)
		s.escribirError(w, &adaptador.Error{
			Estado:  http.StatusNotFound,
			Detalle: "no hay ninguna ruta " + r.Method + " " + r.URL.Path + " en este servidor SCIM",
		})
		return
	}
	s.mux.ServeHTTP(w, r)
}

// envolver aplica lo que es comun a todas las rutas: credencial, limite de
// cuerpo, registro de actividad y traduccion del error al formato SCIM.
//
// La credencial se comprueba ANTES que nada, incluido antes de mirar el cuerpo.
func (s *Servidor) envolver(h func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.autenticado(r) {
			s.registrarCredencial()
			w.Header().Set("WWW-Authenticate", `Bearer realm="plazum scim"`)
			s.escribirError(w, &adaptador.Error{
				Estado: http.StatusUnauthorized,
				Detalle: "credencial ausente o invalida. El IdP tiene que mandar " +
					"`Authorization: Bearer <token>` con el token de aprovisionamiento de " +
					"esta instancia",
			})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, s.maxCuerpo)
		if err := h(w, r); err != nil {
			var e *adaptador.Error
			if !errors.As(err, &e) {
				e = &adaptador.Error{Estado: http.StatusInternalServerError, Detalle: err.Error()}
			}
			s.registrar(false, e.Detalle)
			s.escribirError(w, e)
			return
		}
		s.registrar(true, "")
	}
}

// autenticado compara la credencial en tiempo constante.
//
// Se comparan RESUMENES y no los tokens: asi la comparacion siempre opera sobre
// 32 bytes y el tiempo no depende de la longitud del token que mandan, que es
// por donde se filtra la longitud del bueno.
func (s *Servidor) autenticado(r *http.Request) bool {
	cabecera := r.Header.Get("Authorization")
	const prefijo = "Bearer "
	if len(cabecera) <= len(prefijo) || !strings.EqualFold(cabecera[:len(prefijo)], prefijo) {
		return false
	}
	presentado := sha256.Sum256([]byte(cabecera[len(prefijo):]))
	return subtle.ConstantTimeCompare(presentado[:], s.resumen[:]) == 1
}

func (s *Servidor) registrar(correcta bool, detalle string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ahora := s.ahora()
	s.act.UltimaPeticion = ahora
	s.act.Peticiones++
	if correcta {
		s.act.UltimaCorrecta = ahora
		return
	}
	s.act.Rechazos++
	s.act.UltimoRechazo = detalle
}

func (s *Servidor) registrarCredencial() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.act.UltimaPeticion = s.ahora()
	s.act.Peticiones++
	s.act.Rechazos++
	s.act.RechazosDeCredencial++
	// El token presentado NO se guarda ni se registra. Un token en un log es
	// un token comprometido, tambien cuando es el equivocado: el equivocado de
	// hoy suele ser el bueno de otro sitio.
	s.act.UltimoRechazo = "credencial ausente o invalida"
}

// Actividad devuelve el estado del aprovisionamiento.
func (s *Servidor) Actividad() Actividad {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.act
}

// cabecerasDeSalida pone lo que toda respuesta SCIM lleva SIEMPRE.
//
// El nosniff no es adorno, y lo encontro gosec (G705) senalando el Write de
// abajo. El razonamiento largo, porque se va a querer quitar algun dia:
//
// Estas respuestas van con "application/scim+json", que ningun navegador
// interpreta como HTML. Pero el middleware de seguridad de superficies/serve NO
// cubre /scim/v2 todavia (P1 20 de docs/pendientes.md), asi que estas salidas
// eran las unicas del producto que salian sin nosniff. Y el cuerpo de una
// respuesta SCIM lleva texto que puso un tercero: el nombre de un usuario viene
// del proveedor de identidad, o sea de fuera. Con un tipo raro y sin nosniff,
// un navegador que husmee puede decidir que eso es HTML.
//
// No se silencia con #nosec porque no era un falso positivo: era un hueco
// pequeno y real. Cuando el middleware cubra /scim/v2 esto sera redundante, y
// redundante esta bien aqui: la defensa la pone quien escribe el cuerpo.
func cabecerasDeSalida(w http.ResponseWriter) {
	w.Header().Set("Content-Type", adaptador.TipoContenido)
	w.Header().Set("X-Content-Type-Options", "nosniff")
}

func (s *Servidor) escribirError(w http.ResponseWriter, e *adaptador.Error) {
	cabecerasDeSalida(w)
	w.WriteHeader(e.Estado)
	_, _ = w.Write(adaptador.ErrorJSON(e))
}

func (s *Servidor) escribir(w http.ResponseWriter, estado int, cuerpo []byte, localizacion string) {
	cabecerasDeSalida(w)
	if localizacion != "" {
		w.Header().Set("Location", localizacion)
	}
	w.WriteHeader(estado)
	// #nosec G705 -- el cuerpo es JSON de scim+json y sale con nosniff puesto en
	// cabecerasDeSalida, asi que no hay camino a interpretacion como HTML.
	_, _ = w.Write(cuerpo)
}

func (s *Servidor) leerCuerpo(r *http.Request) ([]byte, error) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, &adaptador.Error{
			Estado:  http.StatusRequestEntityTooLarge,
			Detalle: fmt.Sprintf("el cuerpo pasa del maximo de %d bytes", s.maxCuerpo),
		}
	}
	if len(b) == 0 {
		return nil, &adaptador.Error{
			Estado: http.StatusBadRequest, Tipo: "invalidSyntax",
			Detalle: "el cuerpo esta vacio",
		}
	}
	return b, nil
}

// ---------------------------------------------------------------------------
// Users
// ---------------------------------------------------------------------------

func (s *Servidor) crearUsuario(w http.ResponseWriter, r *http.Request) error {
	cuerpo, err := s.leerCuerpo(r)
	if err != nil {
		return err
	}
	u, err := adaptador.DeUsuarioJSON(cuerpo)
	if err != nil {
		return err
	}
	creado, err := s.dir.Crear(u, s.ahora())
	if err != nil {
		return err
	}
	salida, err := s.dir.AUsuarioJSON(creado, s.base)
	if err != nil {
		return err
	}
	s.escribir(w, http.StatusCreated, salida, s.base+"/Users/"+creado.ID)
	return nil
}

func (s *Servidor) leerUsuario(w http.ResponseWriter, r *http.Request) error {
	u, err := s.dir.Leer(r.PathValue("id"))
	if err != nil {
		return err
	}
	salida, err := s.dir.AUsuarioJSON(u, s.base)
	if err != nil {
		return err
	}
	s.escribir(w, http.StatusOK, salida, "")
	return nil
}

func (s *Servidor) reemplazarUsuario(w http.ResponseWriter, r *http.Request) error {
	cuerpo, err := s.leerCuerpo(r)
	if err != nil {
		return err
	}
	u, err := adaptador.DeUsuarioJSON(cuerpo)
	if err != nil {
		return err
	}
	nuevo, err := s.dir.Reemplazar(r.PathValue("id"), u, s.ahora())
	if err != nil {
		return err
	}
	salida, err := s.dir.AUsuarioJSON(nuevo, s.base)
	if err != nil {
		return err
	}
	s.escribir(w, http.StatusOK, salida, "")
	return nil
}

func (s *Servidor) parchearUsuario(w http.ResponseWriter, r *http.Request) error {
	cuerpo, err := s.leerCuerpo(r)
	if err != nil {
		return err
	}
	var p adaptador.Parcheo
	if err := json.Unmarshal(cuerpo, &p); err != nil {
		return &adaptador.Error{
			Estado: http.StatusBadRequest, Tipo: "invalidSyntax",
			Detalle: "el cuerpo no es un PatchOp de SCIM: " + err.Error(),
		}
	}
	nuevo, err := s.dir.Parchear(r.PathValue("id"), p, s.ahora())
	if err != nil {
		return err
	}
	salida, err := s.dir.AUsuarioJSON(nuevo, s.base)
	if err != nil {
		return err
	}
	s.escribir(w, http.StatusOK, salida, "")
	return nil
}

func (s *Servidor) borrarUsuario(w http.ResponseWriter, r *http.Request) error {
	if err := s.dir.Borrar(r.PathValue("id"), s.ahora()); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Servidor) listarUsuarios(w http.ResponseWriter, r *http.Request) error {
	f, err := filtroDe(r)
	if err != nil {
		return err
	}
	us, total, err := s.dir.Listar(f)
	if err != nil {
		return err
	}
	recursos := make([]json.RawMessage, 0, len(us))
	for _, u := range us {
		b, err := s.dir.AUsuarioJSON(u, s.base)
		if err != nil {
			return err
		}
		recursos = append(recursos, b)
	}
	salida, err := adaptador.ListaJSON(recursos, total, f.Desde)
	if err != nil {
		return err
	}
	s.escribir(w, http.StatusOK, salida, "")
	return nil
}

// ---------------------------------------------------------------------------
// Groups
// ---------------------------------------------------------------------------

func (s *Servidor) crearGrupo(w http.ResponseWriter, r *http.Request) error {
	cuerpo, err := s.leerCuerpo(r)
	if err != nil {
		return err
	}
	g, err := adaptador.DeGrupoJSON(cuerpo)
	if err != nil {
		return err
	}
	creado, err := s.dir.CrearGrupo(g, s.ahora())
	if err != nil {
		return err
	}
	salida, err := s.dir.AGrupoJSON(creado, s.base)
	if err != nil {
		return err
	}
	s.escribir(w, http.StatusCreated, salida, s.base+"/Groups/"+creado.ID)
	return nil
}

func (s *Servidor) leerGrupo(w http.ResponseWriter, r *http.Request) error {
	g, err := s.dir.LeerGrupo(r.PathValue("id"))
	if err != nil {
		return err
	}
	salida, err := s.dir.AGrupoJSON(g, s.base)
	if err != nil {
		return err
	}
	s.escribir(w, http.StatusOK, salida, "")
	return nil
}

func (s *Servidor) parchearGrupo(w http.ResponseWriter, r *http.Request) error {
	cuerpo, err := s.leerCuerpo(r)
	if err != nil {
		return err
	}
	var p adaptador.Parcheo
	if err := json.Unmarshal(cuerpo, &p); err != nil {
		return &adaptador.Error{
			Estado: http.StatusBadRequest, Tipo: "invalidSyntax",
			Detalle: "el cuerpo no es un PatchOp de SCIM: " + err.Error(),
		}
	}
	g, err := s.dir.ParchearGrupo(r.PathValue("id"), p, s.ahora())
	if err != nil {
		return err
	}
	salida, err := s.dir.AGrupoJSON(g, s.base)
	if err != nil {
		return err
	}
	s.escribir(w, http.StatusOK, salida, "")
	return nil
}

func (s *Servidor) borrarGrupo(w http.ResponseWriter, r *http.Request) error {
	if err := s.dir.BorrarGrupo(r.PathValue("id")); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Servidor) listarGrupos(w http.ResponseWriter, r *http.Request) error {
	f, err := filtroDe(r)
	if err != nil {
		return err
	}
	gs, total, err := s.dir.ListarGrupos(f)
	if err != nil {
		return err
	}
	recursos := make([]json.RawMessage, 0, len(gs))
	for _, g := range gs {
		b, err := s.dir.AGrupoJSON(g, s.base)
		if err != nil {
			return err
		}
		recursos = append(recursos, b)
	}
	salida, err := adaptador.ListaJSON(recursos, total, f.Desde)
	if err != nil {
		return err
	}
	s.escribir(w, http.StatusOK, salida, "")
	return nil
}

// ---------------------------------------------------------------------------
// El filtro
// ---------------------------------------------------------------------------

// filtroDe parsea el subconjunto de filtro que se admite: `atributo eq "valor"`.
//
// Cualquier otra cosa da error de filtro no admitido. La alternativa (ignorar
// el filtro y devolver la lista entera) es peor de lo que parece: el IdP
// pregunta "¿existe ya alguien con este userName?", recibe la plantilla entera,
// concluye que si, y deja de aprovisionar a nadie.
func filtroDe(r *http.Request) (adaptador.Filtro, error) {
	q := r.URL.Query()
	f := adaptador.Filtro{}
	if v := q.Get("startIndex"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return f, &adaptador.Error{
				Estado: http.StatusBadRequest, Tipo: "invalidValue",
				Detalle: "`startIndex` tiene que ser un entero de 1 en adelante",
			}
		}
		f.Desde = n
	}
	if v := q.Get("count"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return f, &adaptador.Error{
				Estado: http.StatusBadRequest, Tipo: "invalidValue",
				Detalle: "`count` tiene que ser un entero de 0 en adelante",
			}
		}
		f.Cuantos = n
	}
	expr := strings.TrimSpace(q.Get("filter"))
	if expr == "" {
		return f, nil
	}
	atributo, valor, err := parsearEq(expr)
	if err != nil {
		return f, err
	}
	f.Atributo = atributo
	f.Valor = valor
	return f, nil
}

func parsearEq(expr string) (string, string, error) {
	noAdmitido := func() (string, string, error) {
		return "", "", &adaptador.Error{
			Estado: http.StatusBadRequest, Tipo: "invalidFilter",
			Detalle: "solo se admite un filtro de la forma `atributo eq \"valor\"`. El filtro " +
				"recibido no lo es. Se prefiere decirlo a ignorarlo: un filtro ignorado " +
				"devuelve la lista entera y hace que el IdP concluya lo contrario de lo que " +
				"pregunto, y entonces duplica usuarios",
		}
	}
	i := strings.Index(strings.ToLower(expr), " eq ")
	if i < 0 {
		return noAdmitido()
	}
	atributo := strings.TrimSpace(expr[:i])
	valor := strings.TrimSpace(expr[i+4:])
	if atributo == "" || valor == "" {
		return noAdmitido()
	}
	// Nada de operadores compuestos: si detras hay otro termino, no se admite.
	bajo := strings.ToLower(valor)
	if strings.Contains(bajo, " and ") || strings.Contains(bajo, " or ") ||
		strings.ContainsAny(atributo, "()[] ") {
		return noAdmitido()
	}
	if len(valor) >= 2 && valor[0] == '"' && valor[len(valor)-1] == '"' {
		var s string
		if err := json.Unmarshal([]byte(valor), &s); err != nil {
			return noAdmitido()
		}
		valor = s
	}
	// El atributo llega con o sin prefijo de esquema; se normaliza igual que
	// las rutas de PATCH para que los dos caminos entiendan lo mismo.
	atributo = strings.ToLower(atributo)
	if i := strings.LastIndex(atributo, ":"); i >= 0 {
		atributo = atributo[i+1:]
	}
	return atributo, valor, nil
}

// ---------------------------------------------------------------------------
// Los tres endpoints de descubrimiento de SCIM
// ---------------------------------------------------------------------------

// configuracion es /ServiceProviderConfig. Los IdP lo leen para saber que
// pueden usar, y aqui se dice la verdad: sin bulk, sin ordenacion, y con el
// filtro limitado.
//
// Declarar capacidades que no se tienen es peor que no declararlas: el IdP se
// fia, las usa, y falla el aprovisionamiento a medio ciclo.
func (s *Servidor) configuracion(w http.ResponseWriter, _ *http.Request) error {
	cuerpo, err := json.MarshalIndent(map[string]any{
		// Sin `documentationUri`, que es opcional. Apuntaria a un dominio
		// todavia provisional (la decision de marca esta congelada), y
		// publicar en un protocolo una URL que puede no existir es peor que no
		// publicar ninguna.
		"schemas":        []string{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"},
		"patch":          map[string]any{"supported": true},
		"bulk":           map[string]any{"supported": false, "maxOperations": 0, "maxPayloadSize": 0},
		"filter":         map[string]any{"supported": true, "maxResults": adaptador.MaxPorPagina},
		"changePassword": map[string]any{"supported": false},
		"sort":           map[string]any{"supported": false},
		"etag":           map[string]any{"supported": false},
		"authenticationSchemes": []map[string]any{{
			"type":        "oauthbearertoken",
			"name":        "OAuth Bearer Token",
			"description": "token de aprovisionamiento de esta instancia, en la cabecera Authorization",
			"primary":     true,
		}},
		"meta": map[string]any{"resourceType": "ServiceProviderConfig", "location": s.base + "/ServiceProviderConfig"},
	}, "", "  ")
	if err != nil {
		return err
	}
	s.escribir(w, http.StatusOK, cuerpo, "")
	return nil
}

func (s *Servidor) tiposDeRecurso(w http.ResponseWriter, _ *http.Request) error {
	tipos := []map[string]any{
		{
			"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"},
			"id":       "User",
			"name":     "User",
			"endpoint": "/Users",
			"schema":   adaptador.EsquemaUsuario,
			"schemaExtensions": []map[string]any{{
				"schema": adaptador.EsquemaEmpresa, "required": false,
			}},
			"meta": map[string]any{"resourceType": "ResourceType", "location": s.base + "/ResourceTypes/User"},
		},
		{
			"schemas":  []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"},
			"id":       "Group",
			"name":     "Group",
			"endpoint": "/Groups",
			"schema":   adaptador.EsquemaGrupo,
			"meta":     map[string]any{"resourceType": "ResourceType", "location": s.base + "/ResourceTypes/Group"},
		},
	}
	recursos := make([]json.RawMessage, 0, len(tipos))
	for _, t := range tipos {
		b, err := json.Marshal(t)
		if err != nil {
			return err
		}
		recursos = append(recursos, b)
	}
	cuerpo, err := adaptador.ListaJSON(recursos, len(recursos), 1)
	if err != nil {
		return err
	}
	s.escribir(w, http.StatusOK, cuerpo, "")
	return nil
}

func (s *Servidor) esquemas(w http.ResponseWriter, _ *http.Request) error {
	ids := []string{adaptador.EsquemaUsuario, adaptador.EsquemaEmpresa, adaptador.EsquemaGrupo}
	recursos := make([]json.RawMessage, 0, len(ids))
	for _, id := range ids {
		b, err := json.Marshal(map[string]any{
			"id":   id,
			"name": nombreCorto(id),
			"meta": map[string]any{"resourceType": "Schema", "location": s.base + "/Schemas/" + id},
		})
		if err != nil {
			return err
		}
		recursos = append(recursos, b)
	}
	cuerpo, err := adaptador.ListaJSON(recursos, len(recursos), 1)
	if err != nil {
		return err
	}
	s.escribir(w, http.StatusOK, cuerpo, "")
	return nil
}

func nombreCorto(urn string) string {
	if i := strings.LastIndex(urn, ":"); i >= 0 {
		return urn[i+1:]
	}
	return urn
}
