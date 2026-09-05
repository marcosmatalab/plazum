// Package uar es la pantalla donde se revisan los accesos, y es la primera
// superficie de plazum que MUTA.
//
// POR QUE NO VIVE EN superficies/pantallas, que es donde estarian sus hermanas.
// Aquella superficie es GET-only POR DISENO, no por casualidad: su cabecera lo
// declara ("ninguna ruta de esta superficie muta nada"), compra con ello que las
// respuestas de la entrevista viajen en la direccion de la pagina y que no pueda
// tener el fallo de olvidarse el CSRF, y hay una puerta que enumera sus rutas y
// le manda una peticion mutante a cada una. Meter aqui un POST habria roto su
// invariante central para ahorrarse un paquete.
//
// Asi que la mutacion entra por su propia superficie, montada DETRAS del
// ProtectorCSRF de superficies/serve, que exige token por METODO y no por ruta:
// lo que no se ha pensado queda exigido igual.
//
// # Lo que esta pantalla no decide
//
// Nada. Los cubos, el bloqueo del cierre, la ley de conservacion y la frase de
// lo no revisado los pone nucleo/accesos. Aqui solo se decide que enlace lleva a
// donde y que clave de catalogo rotula cada cosa. Si esta superficie pudiera
// decidir si un acceso cuenta como revisado, habria dos motores.
//
// # Las tres puertas de D11 que esta pantalla estrena
//
//   - TODO ESTADO VACIO TRAE SU SIGUIENTE PASO. Sin campana configurada, esta
//     pantalla no dice "no hay datos": dice la orden exacta que hay que teclear,
//     con sus dos ficheros. Una pantalla vacia sin verbo es un callejon.
//   - CADA NUMERO ES CLICABLE HASTA SU DERIVACION. Los cubos no son adornos:
//     cada recuento filtra la lista y ensena QUE accesos lo componen. Una cifra
//     sin enlace obliga a fiarse.
//   - EL CAMINO ES DETERMINISTA. Aqui no hay IA ni la habra: lo que decide una
//     persona sobre un acceso no se propone solo.
//
// # Y la que no es de D11 pero es la que mas cuesta si falla
//
// Lo no revisado se presenta como DATO QUE FALTA y no como hallazgo, con la
// frase pegada al dato. La escribe nucleo/accesos y aqui se traduce por
// catalogo; un test exige que las dos digan lo mismo en espanol, porque una
// frase que vive en dos sitios se corrige en uno.
package uar

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/marcosmatalab/plazum/adaptadores/plantilla"
	"github.com/marcosmatalab/plazum/nucleo/accesos"
	"github.com/marcosmatalab/plazum/nucleo/ledger"
	"github.com/marcosmatalab/plazum/puertos"
	"github.com/marcosmatalab/plazum/superficies/camino"
)

// Campanas es de donde sale la campana y donde vuelven los hechos.
//
// La superficie no sabe de ficheros ni de ledgers: eso es del adaptador que la
// monta. Lo unico que necesita saber es que los hechos se ANOTAN, no se guardan
// encima de nada.
type Campanas interface {
	// Abierta devuelve la campana sobre la que se trabaja. Devolver (nil, nil)
	// significa "no hay ninguna configurada", que NO es un error: es el estado
	// vacio, y esta pantalla lo sabe pintar con su siguiente paso.
	Abierta() (*accesos.Campana, error)
	// Anotar escribe un hecho.
	Anotar(e ledger.Entrada) error
}

// Opciones para construir la superficie.
type Opciones struct {
	// Fuente puede ser nil: entonces la pantalla existe y dice como configurarla.
	Fuente Campanas
	// Abrir es quien sabe crear una campana con un fichero que sube el
	// navegador. Puede ser nil: entonces la pantalla NO pinta el formulario
	// de subida y dice por que, en vez de ensenar un boton que no va a
	// funcionar. Es el mismo trato que Tokens y por el mismo motivo
	// (invariante 8): el valor cero de una capacidad es no tenerla.
	Abrir Aperturas
	// Catalogo traduce los rotulos de interfaz. Obligatorio.
	Catalogo puertos.Catalogo
	// Base es el prefijo bajo el que se monta, sin barra final.
	Base string
	// Estatico es de donde cuelgan el CSS y htmx, que sirve otra superficie.
	Estatico string
	// CaminoRuta y CaminoClave son la vuelta al CAMINO GUIADO: la direccion de
	// la pantalla que dice en que orden se recorre plazum, y la clave de
	// catalogo de su rotulo. Los pone quien monta, que es el unico que sabe
	// donde esta montado el camino.
	//
	// EL VALOR CERO ES NO PINTAR NADA, que es el restrictivo. Lo que no se
	// admite es MEDIO enlace: direccion sin rotulo o rotulo sin direccion se
	// rechazan al construir.
	CaminoRuta  string
	CaminoClave string
	// Pasos es EL CAMINO ENTERO, para la barra lateral. Lo pasa quien monta,
	// igual que en el acta y por lo mismo: esta superficie no decide cual es
	// el camino, lo pinta.
	//
	// EL VALOR CERO ES NO PINTAR BARRA, que es el restrictivo. Rellenarlo con
	// el canonico cuando llega vacio convertiria un olvido de quien monta en
	// una barra plausible que enlaza a donde el producto quizas no monta nada.
	Pasos []camino.Paso
	// Raiz es el prefijo del SITIO del que cuelgan las rutas de los pasos, no
	// el de esta pantalla. Vacio es la raiz.
	Raiz string
	// Tokens emite el token CSRF de esta peticion. Lo inyecta quien monta,
	// que es el unico que conoce el almacen de sesiones y el nombre de la
	// cookie.
	//
	// SI ES NIL, LA PANTALLA NO PINTA NINGUN FORMULARIO y dice por que. Es el
	// invariante 8 en una frontera de construccion: el valor cero de "no se
	// como emitir un token" tiene que ser "no ensenes botones que no van a
	// funcionar", no "ensenalos sin token".
	Tokens func(*http.Request) (string, error)
	// Ahora se inyecta para poder probar sin dormir. Nil es time.Now.
	Ahora func() time.Time
	// Quien devuelve quien esta operando. Nil devuelve cadena vacia, y
	// entonces no se admite ninguna decision: un hecho sin autor no es un
	// hecho.
	Quien func(*http.Request) string
}

// Superficie es el http.Handler.
type Superficie struct {
	o        Opciones
	mux      *http.ServeMux
	motor    *plantilla.Motor
	patrones []string
}

// Nuevo construye la superficie y registra sus rutas.
func Nuevo(o Opciones) (*Superficie, error) {
	if o.Catalogo == nil {
		return nil, errors.New("uar: falta el catalogo. Sin el, la pantalla saldria con las " +
			"claves en vez de las palabras")
	}
	if o.Ahora == nil {
		o.Ahora = time.Now
	}
	if err := validarCamino(o.CaminoRuta, o.CaminoClave); err != nil {
		return nil, err
	}
	// LOS PASOS LOS JUZGA EL MISMO VALIDADOR que la pantalla del camino. Dos
	// jueces de la misma propiedad acaban discrepando, y el dia que discrepen
	// esta barra y la pantalla del camino ensenaran caminos distintos.
	if len(o.Pasos) > 0 {
		if err := camino.Validar(o.Pasos); err != nil {
			return nil, fmt.Errorf("uar: el camino que se va a pintar en la barra "+
				"lateral no es recorrible: %w", err)
		}
	}
	o.Base = strings.TrimSuffix(o.Base, "/")
	// LAS PROPIAS MAS EL ARMAZON COMPARTIDO, igual que el acta: la barra
	// lateral de esta pantalla es la misma que la de las demas y sale del
	// mismo fichero. Copiar el marcado aqui seria la cuarta copia.
	m, err := plantilla.Nuevo(camino.ConArmazon(plantillasFS), o.Catalogo,
		"plantillas/*.html", camino.PatronDelArmazon)
	if err != nil {
		return nil, fmt.Errorf("uar: no se pueden cargar las plantillas: %w", err)
	}
	s := &Superficie{o: o, mux: http.NewServeMux(), motor: m}
	// Los patrones llevan la Base dentro, para que quien monte no tenga que
	// acordarse del StripPrefix. Un montaje que se olvida el prefijo no da un
	// error: da 404 en todas las rutas, que se lee como "la pantalla no existe".
	s.registrar("GET "+s.o.Base+"/{$}", s.ver)
	s.registrar("POST "+s.o.Base+"/abrir", s.abrir)
	s.registrar("POST "+s.o.Base+"/decidir", s.decidir)
	s.registrar("POST "+s.o.Base+"/excusar", s.excusar)
	s.registrar("POST "+s.o.Base+"/cerrar", s.cerrar)
	return s, nil
}

// ErrCamino: el enlace de vuelta al camino guiado llego a medias.
var ErrCamino = errors.New("uar: enlace al camino guiado invalido")

// validarCamino comprueba el enlace de vuelta al camino guiado.
//
// SE COMPRUEBA AQUI Y TAMBIEN EN LAS OTRAS SUPERFICIES, a proposito: cada una
// recibe el dato por su cuenta y cada una lo pinta, asi que cada una tiene su
// frontera. Es la misma razon por la que la familia de las URL de configuracion
// lleva dos guardas (invariante 11): una sola no llega.
//
// LAS DOS MITADES O NINGUNA, y la direccion tiene que ser de este sitio: con
// dos barras al principio el navegador la lee como otro anfitrion, asi que el
// enlace que existe para no perder a nadie sacaria al revisor de plazum.
func validarCamino(ruta, clave string) error {
	if ruta == "" && clave == "" {
		return nil // el valor cero: no se pinta nada
	}
	if ruta == "" || clave == "" {
		return fmt.Errorf("%w: llega la direccion %q y el rotulo %q, y hacen falta los dos. "+
			"Arreglo: pasar CaminoRuta y CaminoClave juntos, o ninguno", ErrCamino, ruta, clave)
	}
	if !strings.HasPrefix(ruta, "/") || strings.HasPrefix(ruta, "//") {
		return fmt.Errorf("%w: la direccion del camino es %q y tiene que empezar por una sola "+
			"barra. Con dos, el navegador la lee como otro anfitrion", ErrCamino, ruta)
	}
	return nil
}

// registrar es el UNICO sitio por el que se registra una ruta, y anota el
// patron. Es la misma disciplina que en superficies/pantallas y por el mismo
// motivo: la puerta que enumera rutas mutantes tiene que preguntarle al
// enrutador, no a una lista escrita a mano al lado del test.
func (s *Superficie) registrar(patron string, h http.HandlerFunc) {
	s.patrones = append(s.patrones, patron)
	s.mux.HandleFunc(patron, h)
}

// Patrones son las rutas registradas, para que la puerta de CSRF de
// superficies/serve las enumere sin conocer este paquete.
func (s *Superficie) Patrones() []string { return append([]string(nil), s.patrones...) }

func (s *Superficie) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

// idioma resuelve el de la peticion contra los del catalogo.
func (s *Superficie) idioma(r *http.Request) string {
	return s.motor.Resolver(r.Header.Get("Accept-Language"))
}

// ver pinta la pantalla.
func (s *Superficie) ver(w http.ResponseWriter, r *http.Request) {
	v, codigo := s.vista(r)
	s.pintar(w, r, v, codigo)
}

func (s *Superficie) pintar(w http.ResponseWriter, r *http.Request, v Vista, codigo int) {
	// El cuerpo se arma entero antes de tocar el ResponseWriter: html/template
	// escribe segun ejecuta, y un fallo a mitad dejaria media pagina con un 200.
	var b strings.Builder
	if err := s.motor.Render(&b, "pagina", v, s.idioma(r)); err != nil {
		http.Error(w, s.o.Catalogo.Traducir(s.idioma(r), "uar.error.render"),
			http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(codigo)
	_, _ = w.Write([]byte(b.String()))
}

// decidir registra una decision y vuelve a la pantalla.
func (s *Superficie) decidir(w http.ResponseWriter, r *http.Request) {
	c, quien, ok := s.preparar(w, r)
	if !ok {
		return
	}
	v, err := accesos.VeredictoDe(r.PostFormValue("veredicto"))
	if err != nil {
		s.conAviso(w, r, err.Error())
		return
	}
	d := accesos.Decision{
		Fila: r.PostFormValue("fila"), Veredicto: v, Quien: quien, Cuando: s.o.Ahora(),
		Motivo: strings.TrimSpace(r.PostFormValue("motivo")),
		A:      strings.TrimSpace(r.PostFormValue("a")),
	}
	// SE REGISTRA EN LA CAMPANA ANTES DE ANOTAR. Al reves, una decision mal
	// formada quedaria en un registro append-only para siempre.
	if err := c.Registrar(d); err != nil {
		s.conAviso(w, r, err.Error())
		return
	}
	e, err := accesos.DecisionComoEntrada(d, c.Sello(), c.ID())
	if err != nil {
		s.conAviso(w, r, err.Error())
		return
	}
	if err := s.o.Fuente.Anotar(e); err != nil {
		s.conAviso(w, r, err.Error())
		return
	}
	s.volver(w, r)
}

func (s *Superficie) excusar(w http.ResponseWriter, r *http.Request) {
	c, quien, ok := s.preparar(w, r)
	if !ok {
		return
	}
	// EL ERROR DE Atoi NO SE TRAGA, y este es el arreglo de un P1.
	//
	// La primera version hacia `desde, _ := strconv.Atoi(...)`, asi que un campo
	// vacio, ausente o con letras se convertia en CERO en silencio y una excusa
	// {0,0} -- que no excusa nada -- entraba al ledger append-only para siempre.
	// Es el cero degenerado del invariante 8 en su forma exacta. Y la unica
	// guarda que lo impedia estaba en el `min="1" required` de la plantilla, o
	// sea en el navegador: un `curl` la ignora. Una validacion que solo vive en
	// el cliente no es una validacion, es una sugerencia.
	desde, err := numeroObligatorio(r, "desde")
	if err != nil {
		s.conAviso(w, r, s.o.Catalogo.Traducir(s.idioma(r), "uar.excusar.no_es_numero", "desde"))
		return
	}
	// `hasta` vacio significa "la misma linea", que es el caso normal. Aqui las
	// DOS formas de la nada (ausente y presente-vacio) valen lo mismo, y eso es
	// una decision escrita, no un descuido; lo que no vale es un valor que hay y
	// no se entiende. En `desde` no hay valor por defecto: la nada es un error.
	hasta, err := numeroOpcional(r, "hasta", desde)
	if err != nil {
		s.conAviso(w, r, s.o.Catalogo.Traducir(s.idioma(r), "uar.excusar.no_es_numero", "hasta"))
		return
	}
	e := accesos.Excusa{Desde: desde, Hasta: hasta, Quien: quien,
		Motivo: strings.TrimSpace(r.PostFormValue("motivo")), Cuando: s.o.Ahora()}
	if e.Hasta < e.Desde {
		e.Hasta = e.Desde
	}
	if err := c.Excusar(e); err != nil {
		s.conAviso(w, r, err.Error())
		return
	}
	ent, err := accesos.ExcusaComoEntrada(e, c.Sello(), c.ID())
	if err != nil {
		s.conAviso(w, r, err.Error())
		return
	}
	if err := s.o.Fuente.Anotar(ent); err != nil {
		s.conAviso(w, r, err.Error())
		return
	}
	s.volver(w, r)
}

func (s *Superficie) cerrar(w http.ResponseWriter, r *http.Request) {
	c, quien, ok := s.preparar(w, r)
	if !ok {
		return
	}
	cierre, err := c.Cerrar(quien, s.o.Ahora())
	if err != nil {
		// El error del cierre es LA INFORMACION, no un fallo que esconder:
		// dice exactamente que queda pendiente. Se ensena en la pantalla.
		s.conAviso(w, r, err.Error())
		return
	}
	ent, err := accesos.CierreComoEntrada(cierre, c.ID())
	if err != nil {
		s.conAviso(w, r, err.Error())
		return
	}
	if err := s.o.Fuente.Anotar(ent); err != nil {
		s.conAviso(w, r, err.Error())
		return
	}
	s.volver(w, r)
}

// preparar hace las comprobaciones que comparten las tres mutaciones.
func (s *Superficie) preparar(w http.ResponseWriter, r *http.Request) (*accesos.Campana, string, bool) {
	if s.o.Fuente == nil {
		s.conAviso(w, r, s.o.Catalogo.Traducir(s.idioma(r), "uar.sin_campana.titulo"))
		return nil, "", false
	}
	quien := ""
	if s.o.Quien != nil {
		quien = strings.TrimSpace(s.o.Quien(r))
	}
	if quien == "" {
		// UN HECHO SIN AUTOR NO ES UN HECHO. No se rellena con "anonimo": la
		// campana la firma una persona y el ledger dice quien decidio cada cosa.
		s.conAviso(w, r, s.o.Catalogo.Traducir(s.idioma(r), "uar.aviso.sin_autor"))
		return nil, "", false
	}
	c, err := s.o.Fuente.Abierta()
	if err != nil {
		s.conAviso(w, r, err.Error())
		return nil, "", false
	}
	if c == nil {
		s.conAviso(w, r, s.o.Catalogo.Traducir(s.idioma(r), "uar.sin_campana.titulo"))
		return nil, "", false
	}
	return c, quien, true
}

// volver redirige a la pantalla despues de una mutacion.
//
// Redirigir y no pintar directamente: asi recargar no repite la decision, que
// con un formulario de revocacion es la diferencia entre revocar una vez y
// revocar cada vez que alguien pulsa F5.
func (s *Superficie) volver(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, s.o.Base+"/", http.StatusSeeOther)
}

// conAviso vuelve a pintar la pantalla con el aviso arriba, en vez de con una
// pagina de error suelta. Sacar a alguien de la pantalla para decirle que le
// falta el motivo le hace perder lo que estaba mirando.
func (s *Superficie) conAviso(w http.ResponseWriter, r *http.Request, aviso string) {
	s.conAvisoDe(w, r, "", aviso)
}

// conAvisoDe es lo mismo diciendo QUE GUARDA hablo.
//
// La clave viaja hasta el HTML como `data-aviso` y no se pinta: existe para
// que una puerta pueda afirmar cual de los rechazos ha ocurrido sin comparar
// prosa. Con clave vacia se comporta exactamente como antes, que es el caso
// de los errores del dominio: los escribe el nucleo y no tienen clave.
func (s *Superficie) conAvisoDe(w http.ResponseWriter, r *http.Request, clave, aviso string) {
	v, _ := s.vista(r)
	v.Aviso = aviso
	v.AvisoClave = clave
	s.pintar(w, r, v, http.StatusUnprocessableEntity)
}

// vista arma el modelo. Aqui NO se decide nada del dominio: se pregunta.
func (s *Superficie) vista(r *http.Request) (Vista, int) {
	idi := s.idioma(r)
	v := Vista{
		Idioma: idi, Base: s.o.Base, Estatico: s.o.Estatico,
		Titulo: "uar.titulo",
		Cubo:   r.URL.Query().Get("cubo"),
		// EL CAMINO SE PINTA EN TODOS LOS ESTADOS, incluidos el de sin sesion
		// y el de sin campana. Son justo los dos en los que quien llega se
		// queda mirando una pagina que no le dice nada.
		Camino: EnlaceCamino{URL: s.o.CaminoRuta, Clave: s.o.CaminoClave},
		Inicio: camino.InicioDe(s.o.Raiz),
		// EL PASO SE MARCA POR SU IDENTIFICADOR, no por su posicion ni por su
		// ruta (invariante 7): la ruta puede cambiar y el orden tambien.
		Tira: camino.TiraDe(s.o.Pasos, s.o.Raiz, s.o.CaminoRuta, camino.IDDeLaUAR, ""),
	}
	if s.o.Tokens != nil {
		if tok, err := s.o.Tokens(r); err == nil {
			v.CSRF = tok
			v.CampoCSRF = "csrf"
		}
	}
	// SIN TOKEN NO SE PINTA NINGUN FORMULARIO. Un boton que no puede funcionar
	// es peor que ninguno: quien lo pulse creera que ha decidido.
	v.PuedeMutar = v.CSRF != ""
	// LAS DOS CONDICIONES JUNTAS. Un token sin adaptador pinta un
	// formulario que contesta 422, y un adaptador sin token uno que
	// contesta 403. Las dos formas de no poder subir se ven igual desde
	// fuera y ninguna de las dos se ensena.
	v.PuedeSubir = v.PuedeMutar && s.o.Abrir != nil

	// SIN SESION NO SE ENSENA EL CENSO, y esto no es una comodidad de interfaz.
	//
	// Las pantallas derivadas de plazum ensenan el corpus y el alcance de la
	// organizacion, que no identifica a nadie. Esta ensena NOMBRES DE PERSONAS Y
	// SUS PERMISOS, y servirla a quien no ha entrado la convierte en un
	// directorio de empleados publicado sin querer.
	//
	// YA NO ES LA UNICA CON DATO PERSONAL (01-09-2026): superficies/acta exige
	// sesion por lo mismo, y no lleva lo mismo. Esta ensena a los SUJETOS
	// revisados; aquella, a los ACTORES, o sea quien hizo que dentro de la
	// organizacion. La frase que ve el usuario decia "la unica pantalla" y se
	// corrigio: una afirmacion asi deja de ser verdad sin que nadie la toque.
	//
	// Se decide por lo mismo que decide si se puede mutar (que haya operador) y
	// no por una lista de rutas protegidas: una lista se desincroniza el dia que
	// alguien anade una pantalla.
	if s.o.Quien == nil || strings.TrimSpace(s.o.Quien(r)) == "" {
		v.SinSesion = true
		return v, http.StatusUnauthorized
	}

	if s.o.Fuente == nil {
		v.SinCampana = true
		return v, http.StatusOK
	}
	c, err := s.o.Fuente.Abierta()
	if err != nil {
		v.SinCampana = true
		v.Aviso = err.Error()
		return v, http.StatusInternalServerError
	}
	if c == nil {
		v.SinCampana = true
		return v, http.StatusOK
	}
	v.rellenarCon(c, idi, s.o.Catalogo)
	return v, http.StatusOK
}

// numeroObligatorio y numeroOpcional leen un entero SIN tragarse el error.
//
// Son dos funciones y no una con valor por defecto porque son dos preguntas
// distintas, y meterlas en una es como se cuela el cero degenerado: en un campo
// obligatorio, la nada es un ERROR; en uno opcional, la nada es el valor por
// defecto. `strconv.Atoi` con el error descartado convierte las tres cosas
// -- ausente, presente-vacio y presente-ilegible -- en el mismo cero.
//
// Las dos formas de la nada (invariante 8) se tratan igual dentro de cada
// funcion, y eso esta escrito porque es la decision: lo que NO puede pasar es
// que un valor que hay y no se entiende se confunda con no haberlo puesto.
func numeroObligatorio(r *http.Request, campo string) (int, error) {
	crudo := strings.TrimSpace(r.PostFormValue(campo))
	if crudo == "" {
		return 0, fmt.Errorf("falta %s", campo)
	}
	return strconv.Atoi(crudo)
}

func numeroOpcional(r *http.Request, campo string, porDefecto int) (int, error) {
	crudo := strings.TrimSpace(r.PostFormValue(campo))
	if crudo == "" {
		return porDefecto, nil
	}
	return strconv.Atoi(crudo)
}

// ordenar deja la lista estable entre recargas. Una tabla que se reordena sola
// hace que quien revisa pierda el sitio en cada decision.
func ordenarFilas(fs []FilaVista) {
	sort.Slice(fs, func(i, j int) bool {
		if fs[i].Cuenta != fs[j].Cuenta {
			return fs[i].Cuenta < fs[j].Cuenta
		}
		return fs[i].Permiso < fs[j].Permiso
	})
}
