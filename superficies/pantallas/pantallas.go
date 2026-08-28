// Package pantallas es la superficie web de las seis pantallas de la etapa 2.
//
// # Que hace, y sobre todo que no hace
//
// Aqui se PINTA el modelo que deriva nucleo/pantalla, y nada mas. Que se ensena
// y en que orden ya esta decidido alli, con casos dorados comparados byte a
// byte, y no se vuelve a decidir aqui: si hubiera dos derivaciones del mismo
// corpus, la promesa de producto ("anadir la norma 31 es un fichero de datos y
// la interfaz cambia sola") dejaria de ser comprobable en una revision de
// codigo. Consta en docs/puertos-propuestas.md, donde UIGenerada se retiro como
// puerto justamente por esto.
//
// # Un http.Handler autonomo
//
// Esto entrega un http.Handler completo: no registra rutas en ningun router
// compartido, no depende de superficies/serve y no trae middleware. Quien
// construya el servidor lo monta donde quiera, con http.StripPrefix si va bajo
// prefijo, y le pone delante lo suyo: cabeceras, rate limit, sesion y CSRF.
//
// # Todas las rutas son GET, y es una decision
//
// Ninguna ruta de esta superficie muta nada. Las respuestas de la entrevista
// viajan en la direccion de la pagina. Eso compra tres cosas:
//
//	la pagina se comparte, se marca y se vuelve a abrir con el alcance puesto,
//	que es como de verdad se trabaja esto entre dos personas;
//	no hace falta CSRF para algo que no cambia estado, asi que esta superficie
//	no puede tener el fallo de olvidarlo;
//	y no se finge una persistencia que no existe. El expediente llega despues.
//	Decirle al operador que su alcance queda guardado, y que al volver no este,
//	es peor que no guardarlo: la pantalla lo dice con esas palabras.
//
// El dia que haya un POST aqui, tiene que pasar por el middleware de CSRF de
// quien construye el servidor. No hay ninguno hoy y el test lo vigila.
//
// # La derivacion a un clic
//
// Desde Alcance, cada respuesta mueve obligaciones entre tres cestas (aplica,
// pendiente, no aplica) y ensena de que respuesta y de que articulo sale cada
// movimiento, sin pasar por ninguna pantalla de configuracion. Las preguntas
// vienen ya ordenadas por cuantas obligaciones desbloquea cada una, asi que la
// primera sin responder es siempre la que mas avanza: nunca se ensena un
// catalogo de controles en frio.
//
// Lo de aqui NO es el motor de aplicabilidad. Ver el encabezado de derivacion.go.
package pantallas

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/marcosmatalab/plazum/adaptadores/plantilla"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/pantalla"
	"github.com/marcosmatalab/plazum/puertos"
)

// Nombres de los parametros que viajan en la direccion.
const (
	ParamSi     = "si"
	ParamNo     = "no"
	ParamFiltro = "f"
	ParamPagina = "p"
)

// Limites de la superficie. Son puertas, no adornos: una peticion adversaria no
// puede convertirse en una pagina de varios megabytes ni en un parseo caro.
const (
	// MaxConsulta acota la parte de consulta de la direccion. Con IDs de
	// pregunta reales caben cientos de respuestas, y la entrevista mas larga
	// que existe tiene decenas.
	MaxConsulta = 8192
	// PorPaginaPorDefecto acota cuantas filas se pintan de una vez.
	PorPaginaPorDefecto = 200
	// MaxProximas acota cuantas obligaciones pendientes se adelantan en
	// Alcance. Es una ayuda, no un listado: el listado esta en Controles.
	MaxProximas = 8
	// MaxAplican acota cuantas obligaciones aplicables se detallan en el
	// panel de Alcance. El contador dice el total y el listado esta en
	// Controles: el panel es para entender, no para inventariar.
	MaxAplican = 12
)

// Errores de construccion, como centinelas.
var (
	ErrSinCatalogo = errors.New("superficie de pantallas sin catalogo")
	ErrBase        = errors.New("prefijo de montaje invalido")
)

// Opciones construye la superficie.
type Opciones struct {
	// Paquetes es el corpus instalado. Puede venir vacio: entonces las
	// pantallas que salen del corpus se pintan vacias diciendo que falta
	// instalar corpus, que es un problema distinto de no tener estado.
	Paquetes []*corpus.Paquete
	// Catalogo pone el texto de la interfaz. Obligatorio: sin el, la pagina
	// saldria con las claves crudas.
	Catalogo puertos.Catalogo
	// Plantilla es opcional. Si es nil se construye la de referencia sobre
	// las plantillas embebidas de este paquete.
	Plantilla puertos.Plantilla
	// Base es el prefijo bajo el que se monta la superficie, solo para
	// componer enlaces: "" o "/ui", sin barra final. Quien monte bajo
	// prefijo tiene que usar ademas http.StripPrefix.
	Base string
	// PorPagina acota las filas por pagina. 0 usa PorPaginaPorDefecto.
	PorPagina int
	// AlFallar recibe los errores que no se pueden ensenar al usuario
	// (fallos de render). Opcional; sin el, se pierden.
	AlFallar func(error)

	// Ahora es el reloj de la superficie. Opcional; sin el, time.Now en UTC.
	//
	// Esta aqui y no en nucleo/pantalla porque el nucleo no lee el reloj: el
	// instante entra como dato, y este es el sitio donde ese dato se
	// obtiene. Que sea una funcion y no un instante fijo importa: el estado
	// del planificador se juzga en cada peticion, no en el arranque, y una
	// pagina que dijera "late" porque latia cuando arranco el servidor seria
	// la mentira exacta que esta pieza existe para no contar.
	Ahora func() time.Time

	// Marcas dice lo que la instalacion sabe de si misma (cuando corrio el
	// ultimo ciclo del planificador, si el latido esta encendido, cuando
	// llego el ultimo pulso). Opcional; sin ella, la pantalla Hoy dice que
	// el planificador no ha reportado ningun ciclo, que es la verdad.
	//
	// Se pasa una FUNCION y no un valor porque el estado cambia mientras el
	// servidor corre: lo escribe el ciclo del planificador en su fichero, y
	// la pantalla tiene que leer lo de ahora, no lo del arranque.
	Marcas func() pantalla.Marcas
}

// modelo es el corpus ya derivado. Se guarda derivado y no se deriva por
// peticion porque la derivacion es pura y determinista: el mismo corpus da el
// mismo modelo, asi que recalcularlo en cada peticion solo gasta.
type modelo struct {
	pantallas []pantalla.Pantalla
	porID     map[pantalla.ID]pantalla.Pantalla
	preguntas []pantalla.Pregunta
	idx       indicePreguntas
	// fuentes es la atribucion del corpus instalado. Solo la usa la pagina
	// de error, que no corresponde a ninguna pantalla; las demas la leen de
	// la pantalla que estan pintando.
	fuentes []pantalla.Fuente
}

func derivarModelo(ps []*corpus.Paquete) modelo {
	m := modelo{pantallas: sanearPantallas(pantalla.Derivar(ps)),
		porID: map[pantalla.ID]pantalla.Pantalla{}}
	for _, p := range m.pantallas {
		m.porID[p.ID] = p
	}
	m.preguntas = m.porID[pantalla.Alcance].Preguntas
	m.idx = indexar(m.preguntas)
	if len(m.pantallas) > 0 {
		m.fuentes = m.pantallas[0].Fuentes
	}
	return m
}

// Superficie es el http.Handler de las seis pantallas.
type Superficie struct {
	mux *http.ServeMux
	// patrones son las rutas registradas, anotadas por registrar. La puerta de
	// "ninguna ruta muta" pregunta aqui en vez de fiarse de una lista paralela
	// escrita a mano, que es lo que habia y no cazaba una ruta nueva.
	patrones  []string
	plt       puertos.Plantilla
	base      string
	porPagina int
	alFallar  func(error)
	// idiomas y idiomaDefecto son los que declara el catalogo. Se guardan
	// para poder comprobar que el idioma que acaba en <html lang> es uno de
	// ellos, pase lo que pase con el adaptador de plantillas.
	idiomas       map[string]bool
	idiomaDefecto string
	// ahora y marcas son de donde sale el estado del planificador que se
	// pinta en Hoy. Ver Opciones.
	ahora  func() time.Time
	marcas func() pantalla.Marcas

	mu     sync.RWMutex
	modelo modelo
}

var _ http.Handler = (*Superficie)(nil)

// Nuevo construye la superficie y registra sus rutas.
func Nuevo(o Opciones) (*Superficie, error) {
	if o.Catalogo == nil {
		return nil, fmt.Errorf("%w: el catalogo es nil, asi que la pagina saldria con las "+
			"claves crudas (\"pantalla.alcance.titulo\" en vez de un titulo). Arreglo: "+
			"construye la superficie con un puertos.Catalogo que cubra las claves de "+
			"pantallas.ClavesDeCatalogo()", ErrSinCatalogo)
	}
	if o.Base != "" {
		if err := validarBase(o.Base); err != nil {
			return nil, err
		}
	}
	plt := o.Plantilla
	if plt == nil {
		var err error
		plt, err = plantilla.Nuevo(Plantillas(), o.Catalogo, "plantillas/*.html")
		if err != nil {
			return nil, fmt.Errorf("no puedo construir el motor de plantillas de "+
				"referencia: %w", err)
		}
	}
	idiomas := o.Catalogo.Idiomas()
	if len(idiomas) == 0 {
		return nil, fmt.Errorf("%w: el catalogo no declara ningun idioma, asi que no hay "+
			"idioma por defecto al que caer. Arreglo: que Idiomas() devuelva al menos uno, "+
			"y el primero es el de por defecto", ErrSinCatalogo)
	}
	s := &Superficie{
		mux:           http.NewServeMux(),
		plt:           plt,
		base:          o.Base,
		porPagina:     o.PorPagina,
		alFallar:      o.AlFallar,
		idiomas:       map[string]bool{},
		idiomaDefecto: idiomas[0],
		ahora:         o.Ahora,
		marcas:        o.Marcas,
		modelo:        derivarModelo(o.Paquetes),
	}
	if s.ahora == nil {
		s.ahora = func() time.Time { return time.Now().UTC() }
	}
	if s.marcas == nil {
		// Sin nadie que diga lo contrario, no se sabe nada del
		// planificador. Y no saber nada NO es "correcto": Vigilar lo dice
		// con esas palabras.
		s.marcas = func() pantalla.Marcas { return pantalla.Marcas{} }
	}
	for _, i := range idiomas {
		s.idiomas[i] = true
	}
	if s.porPagina <= 0 {
		s.porPagina = PorPaginaPorDefecto
	}
	s.rutas()
	return s, nil
}

// Recargar cambia el corpus sin reconstruir la superficie ni reiniciar el
// servidor. Existe porque instalar un paquete de corpus no puede exigir un
// reinicio: el producto se vende diciendo que la norma nueva es un fichero.
func (s *Superficie) Recargar(ps []*corpus.Paquete) {
	m := derivarModelo(ps)
	s.mu.Lock()
	s.modelo = m
	s.mu.Unlock()
}

func (s *Superficie) instantanea() modelo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.modelo
}

// rutas registra una ruta por pantalla del modelo, la raiz y los estaticos.
//
// Se registran recorriendo el modelo derivado y no escribiendo seis rutas a
// mano: si nucleo/pantalla anade una septima pantalla, tiene ruta y entrada de
// menu sin tocar esto. Las rutas se registran a partir de las pantallas que
// existen al construir, que son siempre las mismas seis; Recargar cambia el
// contenido, no el conjunto.
func (s *Superficie) rutas() {
	m := s.instantanea()
	for _, p := range m.pantallas {
		id := p.ID
		s.registrar("GET "+rutaDe(id), func(w http.ResponseWriter, r *http.Request) {
			s.verPantalla(w, r, id)
		})
	}
	inicio := pantalla.Alcance
	if len(m.pantallas) > 0 {
		inicio = m.pantallas[0].ID
	}
	s.registrar("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, s.base+rutaDe(inicio), http.StatusSeeOther)
	})
	s.registrar("GET /estatico/{fichero}", s.verEstatico)
}

// registrar es el UNICO sitio por el que se registra una ruta, y anota el
// patron ademas de registrarlo.
//
// HALLAZGO SOBRE EL TEST QUE HABIA. TestNingunaRutaDeLaSuperficieMuta probaba
// metodos mutantes contra una LISTA DE RUTAS ESCRITA A MANO al lado del test.
// La mutacion que lo daba por bueno anadia un POST a una ruta que ya estaba en
// esa lista, asi que se cazaba sola. Anadiendo POST /guardar, una ruta que la
// lista no conocia, el test seguia en VERDE. O sea que no sostenia la propiedad
// que decia sostener, que es exactamente el fallo del que este proyecto se
// defiende: una lista a mano se desincroniza el dia que alguien anade un
// handler, que es el dia en que hace falta.
//
// Con esto, la lista de rutas SALE DEL REGISTRO. Y para que nadie se salte el
// registro llamando al mux directamente, hay un test de AST que lo prohibe.
func (s *Superficie) registrar(patron string, h http.HandlerFunc) {
	s.patrones = append(s.patrones, patron)
	s.mux.HandleFunc(patron, h)
}

// Patrones devuelve los patrones registrados, en el orden en que se
// registraron. Existe para que la puerta de "ninguna ruta muta" pregunte al
// router en vez de fiarse de una lista paralela.
func (s *Superficie) Patrones() []string { return append([]string(nil), s.patrones...) }

// ServeHTTP sirve la superficie. Lo que no case con ninguna ruta sale como una
// pagina 404 de la propia superficie, con su menu: un 404 en blanco a mitad de
// una sesion deja al operador sin saber por donde volver.
func (s *Superficie) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if len(r.URL.RawQuery) > MaxConsulta {
		s.fallo(w, r, http.StatusRequestURITooLong, "error.consulta_larga")
		return
	}
	h, patron := s.mux.Handler(r)
	if patron == "" {
		s.fallo(w, r, http.StatusNotFound, "error.no_encontrado")
		return
	}
	h.ServeHTTP(w, r)
}

// verPantalla despacha una pantalla del modelo.
//
// El reparto se hace por origen y por forma, no por una lista de seis casos:
// una pantalla que sale del estado se pinta vacia con su explicacion, una que
// trae preguntas es la entrevista y una que trae filas es una tabla. Solo
// Certificados se nombra, y por una razon de fondo, no de estilo: alli
// Fila.Requiere son IDs de obligacion y no de pregunta, asi que se lee distinto
// (ver evaluarEntregable).
func (s *Superficie) verPantalla(w http.ResponseWriter, r *http.Request, id pantalla.ID) {
	m := s.instantanea()
	p, ok := m.porID[id]
	if !ok {
		s.fallo(w, r, http.StatusNotFound, "error.no_encontrado")
		return
	}
	resp := De(r.URL.Query(), m.preguntas)
	switch {
	case p.ID == pantalla.Hoy:
		s.verHoy(w, r, m, p, resp)
	case p.Origen == pantalla.DelEstado:
		s.verVacia(w, r, m, p, resp)
	case len(p.Preguntas) > 0 || len(p.Campos) > 0 || p.ID == pantalla.Alcance:
		s.verAlcance(w, r, m, p, resp)
	default:
		s.verTabla(w, r, m, p, resp, p.ID == pantalla.Certificados)
	}
}

// verAlcance pinta la entrevista con su derivacion al lado.
func (s *Superficie) verAlcance(w http.ResponseWriter, r *http.Request, m modelo,
	p pantalla.Pantalla, resp Respuestas) {

	controles := veredictosDeControles(m, resp)
	res := resumir(controles)

	v := VistaAlcance{
		Marco:           s.marco(m, p, resp, res.Aplica, "cuerpo-alcance"),
		Vacia:           p.Vacia,
		PorQue:          p.PorQue,
		Origen:          claveOrigen(p.Origen),
		Campos:          p.Campos,
		TotalPreguntas:  len(p.Preguntas),
		Respondidas:     resp.Respondidas(),
		Contradictorias: resp.Contradictorias(),
		Resumen:         res,
		HayRespuestas:   len(resp.Consulta()) > 0,
		URLControles:    s.enlace(rutaDe(pantalla.Controles), resp.Consulta()),
		URLLimpiar:      s.enlace(rutaDe(p.ID), nil),
	}

	// La siguiente sugerida es la primera sin responder. Las preguntas ya
	// llegan ordenadas por cuantas obligaciones desbloquean, o sea que la
	// primera sin responder es siempre la que mas avanza.
	for _, q := range p.Preguntas {
		if d := resp.Dice(q.ID); (d == SinResponder || d == Contradictoria) && v.Siguiente == "" {
			v.Siguiente = q.ID
		}
	}
	for _, q := range p.Preguntas {
		d := resp.Dice(q.ID)
		v.Preguntas = append(v.Preguntas, VistaPregunta{
			Pregunta:         q,
			EsSi:             d == Si,
			EsNo:             d == No,
			EsContradictoria: d == Contradictoria,
			SinResponder:     d == SinResponder,
			Sugerida:         q.ID == v.Siguiente,
			URLSi:            s.enlace(rutaDe(p.ID), resp.Con(q.ID, Si).Consulta()),
			URLNo:            s.enlace(rutaDe(p.ID), resp.Con(q.ID, No).Consulta()),
			URLLimpiar:       s.enlace(rutaDe(p.ID), resp.Con(q.ID, SinResponder).Consulta()),
		})
	}

	for _, c := range controles {
		switch {
		case c.Estado == Aplica:
			if len(v.Aplican) < MaxAplican {
				v.Aplican = append(v.Aplican, c)
			} else {
				v.AplicanMas++
			}
		case c.Estado == Pendiente && v.Siguiente != "" && len(v.Proximas) < MaxProximas &&
			requiere(c.Fila, v.Siguiente):
			// Las que desbloquea justo la pregunta sugerida: es lo que
			// convierte "responde esto" en "responde esto y sabras si
			// te alcanzan estas tres".
			v.Proximas = append(v.Proximas, c)
		}
	}
	s.responder(w, r, http.StatusOK, "pagina", &v)
}

func requiere(f pantalla.Fila, pregunta string) bool {
	for _, q := range f.Requiere {
		if q == pregunta {
			return true
		}
	}
	return false
}

// verTabla pinta Controles y Certificados.
func (s *Superficie) verTabla(w http.ResponseWriter, r *http.Request, m modelo,
	p pantalla.Pantalla, resp Respuestas, entregables bool) {

	controles := veredictosDeControles(m, resp)
	filas := controles
	if entregables {
		porObligacion := make(map[string]Veredicto, len(controles))
		for _, c := range controles {
			porObligacion[c.Fila.ID] = c
		}
		filas = nil
		for _, f := range p.Filas {
			filas = append(filas, evaluarEntregable(f, porObligacion))
		}
	}

	res := resumir(filas)
	filtro, hayFiltro := estadoDeFiltro(r.URL.Query().Get(ParamFiltro))
	visibles := filas
	if hayFiltro {
		visibles = nil
		for _, f := range filas {
			if f.Estado == filtro {
				visibles = append(visibles, f)
			}
		}
	}

	q := resp.Consulta()
	conFiltro := func(e Estado, activo bool) string {
		c := url.Values{}
		for k, vs := range q {
			c[k] = vs
		}
		if !activo {
			c.Set(ParamFiltro, e.String())
		}
		return s.enlace(rutaDe(p.ID), c)
	}

	v := VistaTabla{
		Marco:         s.marco(m, p, resp, resumir(controles).Aplica, "cuerpo-tabla"),
		Vacia:         p.Vacia,
		PorQue:        p.PorQue,
		Origen:        claveOrigen(p.Origen),
		Resumen:       res,
		Total:         len(visibles),
		EsEntregables: entregables,
		URLAlcance:    s.enlace(rutaDe(pantalla.Alcance), q),
		Filtros: []VistaFiltro{
			{Clave: "filtro.todos", URL: s.enlace(rutaDe(p.ID), q), Activo: !hayFiltro,
				N: res.Total},
			{Clave: Aplica.Clave(), URL: conFiltro(Aplica, hayFiltro && filtro == Aplica),
				Activo: hayFiltro && filtro == Aplica, N: res.Aplica},
			{Clave: Pendiente.Clave(), URL: conFiltro(Pendiente, hayFiltro && filtro == Pendiente),
				Activo: hayFiltro && filtro == Pendiente, N: res.Pendiente},
			{Clave: NoAplica.Clave(), URL: conFiltro(NoAplica, hayFiltro && filtro == NoAplica),
				Activo: hayFiltro && filtro == NoAplica, N: res.NoAplica},
		},
	}

	// Paginacion. Una pagina fuera de rango se lleva a la ultima en vez de
	// dar error: el operador ha llegado ahi navegando, no atacando.
	paginas := (len(visibles) + s.porPagina - 1) / s.porPagina
	pagina := 1
	if n, err := strconv.Atoi(r.URL.Query().Get(ParamPagina)); err == nil && n > 1 {
		pagina = n
	}
	if paginas > 0 && pagina > paginas {
		pagina = paginas
	}
	desde := (pagina - 1) * s.porPagina
	hasta := min(desde+s.porPagina, len(visibles))
	if desde < len(visibles) {
		v.Filas = visibles[desde:hasta]
		v.Desde, v.Hasta = desde+1, hasta
	}
	v.SinResultados = len(visibles) == 0 && !p.Vacia
	conPagina := func(n int) string {
		c := url.Values{}
		for k, vs := range q {
			c[k] = vs
		}
		if hayFiltro {
			c.Set(ParamFiltro, filtro.String())
		}
		if n > 1 {
			c.Set(ParamPagina, strconv.Itoa(n))
		}
		return s.enlace(rutaDe(p.ID), c)
	}
	if pagina > 1 {
		v.URLAnterior = conPagina(pagina - 1)
	}
	if pagina < paginas {
		v.URLSiguiente = conPagina(pagina + 1)
	}
	v.Columnas, v.ColumnasDesconocidas = columnasPresentes(v.Filas)
	s.responder(w, r, http.StatusOK, "pagina", &v)
}

// verHoy pinta Hoy: el estado del planificador arriba y el resto debajo.
//
// EL VEREDICTO SE CALCULA EN CADA PETICION, con el reloj de la peticion y con
// las marcas leidas en ese momento. No se lee del modelo derivado, que se
// calcula una vez al arrancar: una pagina que dijera "el planificador late"
// porque latia cuando arranco el servidor es exactamente la mentira que esta
// pieza existe para no contar. Lo vigila
// TestElEstadoDelPlanificadorSeJuzgaEnCadaPeticion.
//
// Y la regla de las 24 horas NO se decide aqui: se le pregunta a
// nucleo/pantalla.Vigilar, que es donde vive, con casos dorados al lado. Si se
// copiara aqui un "if han pasado 24 horas" habria dos reglas y un dia dirian
// cosas distintas.
func (s *Superficie) verHoy(w http.ResponseWriter, r *http.Request, m modelo,
	p pantalla.Pantalla, resp Respuestas) {

	res := resumir(veredictosDeControles(m, resp))
	porque := p.PorQue
	if porque == "" {
		porque = "vacia.sin_explicacion"
	}
	s.responder(w, r, http.StatusOK, "pagina", &VistaHoy{
		Marco:        s.marco(m, p, resp, res.Aplica, "cuerpo-hoy"),
		PorQue:       porque,
		Origen:       claveOrigen(p.Origen),
		URLAlcance:   s.enlace(rutaDe(pantalla.Alcance), resp.Consulta()),
		Planificador: pantalla.Vigilar(s.marcas(), s.ahora()),
	})
}

// verVacia pinta una pantalla sin contenido CON su explicacion.
func (s *Superficie) verVacia(w http.ResponseWriter, r *http.Request, m modelo,
	p pantalla.Pantalla, resp Respuestas) {

	res := resumir(veredictosDeControles(m, resp))
	porque := p.PorQue
	if porque == "" {
		// Defensa en profundidad: el modelo garantiza PorQue en las vacias
		// y hay un test que lo exige, pero una pantalla en blanco sin
		// explicacion no puede salir de aqui ni por un fallo aguas arriba.
		porque = "vacia.sin_explicacion"
	}
	s.responder(w, r, http.StatusOK, "pagina", &VistaVacia{
		Marco:      s.marco(m, p, resp, res.Aplica, "cuerpo-vacia"),
		PorQue:     porque,
		Origen:     claveOrigen(p.Origen),
		URLAlcance: s.enlace(rutaDe(pantalla.Alcance), resp.Consulta()),
	})
}

// veredictosDeControles evalua la pantalla de Controles entera. Se recalcula en
// cada peticion a proposito: depende de las respuestas, que son de la peticion.
func veredictosDeControles(m modelo, resp Respuestas) []Veredicto {
	filas := m.porID[pantalla.Controles].Filas
	out := make([]Veredicto, 0, len(filas))
	for _, f := range filas {
		out = append(out, evaluarControl(f, resp, m.idx))
	}
	return out
}

func (s *Superficie) marco(m modelo, p pantalla.Pantalla, resp Respuestas,
	aplican int, cuerpo string) Marco {
	return Marco{
		Base:     s.base,
		Estatico: s.base + "/estatico",
		Cuerpo:   cuerpo,
		Titulo:   p.Titulo,
		Menu:     s.menu(m, p.ID, resp, aplican),
		// Las fuentes salen de LA PANTALLA que se esta pintando, no de una
		// copia guardada en la superficie. Asi, si una pantalla llegara sin
		// su atribucion, la pagina de esa pantalla se queda sin ella y la
		// puerta lo ve; leerlas de un sitio comun taparia justo ese fallo.
		Fuentes: p.Fuentes,
	}
}

// fallo pinta una pagina de error de la propia superficie.
//
// El mensaje sale del catalogo por CLAVE y NUNCA repite lo que mando el
// cliente. Reflejar la entrada en un error es la mitad de un XSS reflejado, y
// esa mitad no hace falta para nada.
func (s *Superficie) fallo(w http.ResponseWriter, r *http.Request, codigo int, clave string) {
	m := s.instantanea()
	s.responder(w, r, codigo, "pagina", &VistaError{
		Marco: Marco{
			Base: s.base, Estatico: s.base + "/estatico", Cuerpo: "cuerpo-error",
			Titulo: "pantalla.error.titulo", Menu: s.menu(m, "", Respuestas{}, 0),
			// La pagina de error no corresponde a ninguna pantalla, asi que
			// las fuentes se leen del modelo. El corpus esta instalado igual
			// y la atribucion se debe igual.
			Fuentes: m.fuentes,
		},
		Clave:      clave,
		Codigo:     codigo,
		URLAlcance: s.enlace(rutaDe(pantalla.Alcance), nil),
	})
}

// responder renderiza a un buffer y solo entonces escribe.
//
// Es lo unico que evita que un fallo a mitad de plantilla salga como media
// pagina con un 200: html/template escribe segun ejecuta. El coste es tener la
// pagina en memoria, y por eso las tablas van paginadas.
func (s *Superficie) responder(w http.ResponseWriter, r *http.Request, codigo int,
	nombre string, datos any) {

	idioma := idiomaPedido(r)
	if res, ok := s.plt.(interface{ Resolver(string) string }); ok {
		idioma = res.Resolver(idioma)
	}
	// Y se comprueba contra los idiomas que declara el catalogo, aunque la
	// plantilla ya haya resuelto.
	//
	// HALLAZGO DEL BARRIDO DE MUTACION: sin esto, todo dependia de que el
	// adaptador de plantillas implementara Resolver, que es una interfaz
	// OPCIONAL. Con otro adaptador de puertos.Plantilla (que es justo para lo
	// que existe el puerto), lo que el cliente escribiera en Accept-Language
	// acababa tal cual en <html lang>. No es una inyeccion (html/template lo
	// escapa) pero si una pagina que declara un idioma que no lleva dentro, y
	// eso lo sufre quien la lee con un lector de pantalla.
	if !s.idiomas[idioma] {
		idioma = s.idiomaDefecto
	}
	// El idioma que de verdad se va a renderizar es el que va a <html lang>.
	if p, ok := datos.(interface{ fijarIdioma(string) }); ok {
		p.fijarIdioma(idioma)
	}

	var buf bytes.Buffer
	if err := s.plt.Render(&buf, nombre, datos, idioma); err != nil {
		if s.alFallar != nil {
			s.alFallar(fmt.Errorf("render de %q: %w", nombre, err))
		}
		// Sin plantilla no se puede pintar la pagina de error, asi que
		// aqui se sale en texto plano y con codigo 500 de verdad.
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("500\n"))
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(codigo)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(buf.Bytes())
}

func (v *VistaAlcance) fijarIdioma(i string) { v.Idioma = i }
func (v *VistaTabla) fijarIdioma(i string)   { v.Idioma = i }
func (v *VistaVacia) fijarIdioma(i string)   { v.Idioma = i }
func (v *VistaHoy) fijarIdioma(i string)     { v.Idioma = i }
func (v *VistaError) fijarIdioma(i string)   { v.Idioma = i }

// idiomaPedido lee la primera etiqueta de Accept-Language y la sanea.
//
// Se sanea porque acaba en <html lang> y en la eleccion de catalogo, y llega de
// una cabecera que escribe el cliente. Lo que no sea una etiqueta de idioma
// plausible se descarta entero y se cae al idioma por defecto.
func idiomaPedido(r *http.Request) string {
	if r == nil {
		return ""
	}
	cabecera := r.Header.Get("Accept-Language")
	primera, _, _ := strings.Cut(cabecera, ",")
	primera, _, _ = strings.Cut(primera, ";")
	primera = strings.TrimSpace(primera)
	if primera == "" || primera == "*" || len(primera) > 35 {
		return ""
	}
	for _, c := range primera {
		ok := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-'
		if !ok {
			return ""
		}
	}
	return primera
}
