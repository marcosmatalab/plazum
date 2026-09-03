package pantallas

// LA PRIMERA RUTA QUE MUTA DE ESTA SUPERFICIE, y el porque de cada decision.
//
// # Lo que habia, y por que estaba bien mientras duro
//
// Hasta hoy TODA ruta de aqui era GET, y el encabezado del paquete lo explicaba:
// las respuestas viajaban en la direccion, la pagina se compartia, y sobre todo
// «no se finge una persistencia que no existe». `docs/instantanea.md` lo dejo
// escrito con esas palabras: *un boton que no guarda seria la peor mentira de
// esa pantalla*. Era cierto: no habia donde guardar.
//
// Ahora lo hay (`adaptadores/usuarios` puso las cuentas), asi que el alcance
// pasa a ser ESTADO DE LA CUENTA. Y como ahora si guarda, hay que hacerlo con
// las tres cosas que un GET no necesitaba nunca:
//
//	METODO POST      un enlace que escribe lo dispara cualquier cosa que
//	                 precargue enlaces: el antivirus del correo, el prefetch del
//	                 navegador, un rastreador. Con GET, abrir la bandeja de
//	                 entrada te podia borrar la entrevista.
//	CSRF             lo exige el middleware de superficies/serve POR METODO, asi
//	                 que esta ruta entra sola en su puerta. Ver mas abajo por que
//	                 esta superficie no lo comprueba ella.
//	REDIRECCION      303 despues de escribir. Sin ella, F5 reenvia el formulario
//	                 y el navegador pregunta si quieres reenviarlo, que es la
//	                 pantalla que enseña a la gente a desconfiar de un producto.
//
// # POR QUE ESTA SUPERFICIE NO COMPRUEBA EL TOKEN, y como se cierra el hueco
//
// No puede: el token va atado a la SESION, y esta superficie no conoce el
// almacen de sesiones ni el nombre de la cookie (que depende de si hay TLS).
// Es la misma frontera que ya tenia `superficies/uar`. Lo que hace aqui es lo
// unico que puede hacer: PEDIR el token a quien monta (Opciones.Tokens) y, si
// no se lo dan, NO PINTAR NINGUN BOTON. Un formulario sin token es un boton que
// no funciona, y un boton que no funciona es peor que ninguno.
//
// La otra mitad la cierran dos puertas de fuera: `cableado_test.go` en la raiz,
// que manda una peticion mutante a cada ruta del conjunto montado, y
// `cmd/plazum/serve_alcance_test.go`, que lo hace sobre el servidor de verdad.
//
// # Y LA RUTA SOLO EXISTE SI HAY DONDE GUARDAR
//
// Sin `Opciones.Alcances` la superficie no registra ningun POST y sigue siendo
// exactamente la de antes, GET de arriba abajo, con su descargo de «esto no se
// guarda». No es una comodidad de tests: es que una ruta que escribe en un
// almacen que no existe solo puede contestar un error, y una pantalla que
// ofrece guardar y no guarda es la mentira que este fichero viene a no contar.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/pantalla"
)

// CampoCSRF es el nombre del campo oculto que lleva el token.
//
// Se escribe aqui y NO se importa de superficies/serve a proposito: esta
// superficie es un http.Handler autonomo y no depende de quien la monte (lo
// dice el encabezado del paquete). El precio es que las dos cadenas tienen que
// coincidir, y ese precio se paga con una puerta en `cmd/plazum`, que es el
// unico sitio que importa las dos y por tanto el unico que puede compararlas.
const CampoCSRF = "csrf"

// Los nombres de los campos del formulario de guardado.
const (
	// CampoAccion dice QUE se pide hacer. Es obligatorio y su vocabulario es
	// cerrado: ver Accion.
	CampoAccion = "accion"
	// CampoPregunta dice SOBRE QUE. Obligatorio en las acciones que tocan una
	// pregunta, y se lee con otra funcion que el opcional a proposito.
	CampoPregunta = "pregunta"
)

// Accion es lo que un envio del formulario pide hacer. Vocabulario CERRADO.
//
// EL VALOR CERO ES INVALIDO, y es toda la defensa de esta frontera: si el cero
// significara «responder», un `accion` ausente, uno vacio y uno con basura
// dentro darian los tres una escritura que nadie pidio. Son tres cosas
// distintas y las tres son un error aqui, porque este campo es obligatorio.
type Accion uint8

const (
	// AccionSinDeclarar es el cero y no se atiende nunca.
	AccionSinDeclarar Accion = iota
	// AccionSi y AccionNo guardan la respuesta de UNA pregunta.
	AccionSi
	AccionNo
	// AccionOlvidar quita la respuesta de UNA pregunta.
	AccionOlvidar
	// AccionLimpiar deja la cuenta sin ninguna respuesta.
	AccionLimpiar
	// AccionAdoptar guarda en la cuenta las respuestas que traiga el propio
	// envio (las de un enlace que alguien te ha pasado).
	AccionAdoptar
)

// leerAccion lee el campo obligatorio `accion`.
//
// Devuelve (accion, interpretable). Las tres formas de la nada colapsan aqui a
// la misma respuesta, y es correcto PORQUE EL CAMPO ES OBLIGATORIO: ausente,
// presente-vacio y presente-con-basura son los tres «no se que me estas
// pidiendo que haga», y en una escritura eso no se resuelve con un valor por
// defecto. La funcion que lee un campo OPCIONAL es otra, y esta abajo.
func leerAccion(v url.Values) (Accion, bool) {
	valores, presente := v[CampoAccion]
	if !presente || len(valores) != 1 {
		// Repetido tampoco: dos acciones en el mismo envio no se reducen a una
		// sin elegir por quien la manda.
		return AccionSinDeclarar, false
	}
	switch valores[0] {
	case "si":
		return AccionSi, true
	case "no":
		return AccionNo, true
	case "olvidar":
		return AccionOlvidar, true
	case "limpiar":
		return AccionLimpiar, true
	case "adoptar":
		return AccionAdoptar, true
	}
	return AccionSinDeclarar, false
}

// TocaUnaPregunta dice si esta accion necesita el campo `pregunta`.
func (a Accion) TocaUnaPregunta() bool {
	return a == AccionSi || a == AccionNo || a == AccionOlvidar
}

// leerPreguntaObligatoria lee el campo `pregunta` de una accion que lo exige, y
// ademas comprueba que esa pregunta EXISTE en el corpus instalado.
//
// LAS DOS COMPROBACIONES SON UNA SOLA COSA AQUI. Sin la segunda, un envio
// fabricado a mano meteria en el almacen un identificador que ninguna
// obligacion nombra: no cambiaria ningun veredicto (nadie lo consulta) y en
// cambio se quedaria en el fichero para siempre, engordandolo y saliendo despues
// como «respuesta guardada que no corresponde a ninguna pregunta». El corpus es
// quien decide que preguntas hay, igual que en `De`.
// Y DEVUELVE LA COPIA DEL CORPUS, NO LA CADENA QUE MANDO EL CLIENTE.
//
// Son iguales byte a byte (el mapa se indexa por ese mismo id), asi que no es
// una comprobacion mas: es una decision de PROCEDENCIA. Lo que sale de aqui va a
// parar, entre otros sitios, a la cabecera `Location` de la redireccion, y en
// ese sitio la regla de la casa es la misma que en la pagina de error: no se
// refleja lo que mando el cliente.
//
// Y ademas se nota desde fuera. Con la cadena del cliente, gosec marcaba el
// `http.Redirect` con G710 (open redirect por analisis de contaminacion) y tenia
// razon en lo que puede ver: un valor de la peticion llegaba a `Location`.
// Devolviendo la del corpus, el valor que viaja NACE del modelo derivado y no de
// la peticion, y eso lo puede comprobar una herramienta en vez de creerselo.
func leerPreguntaObligatoria(v url.Values, idx indicePreguntas) (string, bool) {
	valores, presente := v[CampoPregunta]
	if !presente || len(valores) != 1 {
		return "", false
	}
	id := strings.TrimSpace(valores[0])
	if id == "" {
		return "", false
	}
	q, existe := idx[id]
	if !existe {
		return "", false
	}
	return q.ID, true
}

// AlcanceGuardado es lo que una cuenta tiene guardado.
//
// Actualizado en cero significa «esta cuenta no ha guardado nunca». Ese cero SI
// es la nada de verdad: no viene de un campo que no se entendiera, viene de no
// haber fila. El almacen se niega a leer un instante que no entiende, asi que
// aqui no puede llegar un cero disfrazado.
type AlcanceGuardado struct {
	Respuestas  map[string]Respuesta
	Actualizado time.Time
}

// Alcances es donde esta superficie guarda y de donde recupera las respuestas.
//
// # Por que el interfaz vive AQUI y no en puertos/
//
// Es el interfaz que consume esta superficie, no un puerto del producto: lo
// declara quien lo usa, que es lo idiomatico en Go y lo que permite que el
// adaptador no tenga que importar esta superficie. Quien lo implementa hoy es
// `adaptadores/usuarios/alcances`, y la traduccion entre los dos vocabularios
// de respuesta la escribe `cmd/plazum`, que es el sitio donde se cablea.
//
// # El vocabulario de la frontera es Respuesta, NO una cadena
//
// Con cadenas, un adaptador podria escribir "SI" o "sí" y aqui no se notaria
// hasta que alguien volviera a entrar y viera su entrevista en blanco. Con el
// tipo, el compilador lo dice. Y lo que NO admite esta frontera es
// SinResponder ni Contradictoria: no responder se escribe con Olvidar, y una
// contradiccion es de la direccion de la pagina, no de una cuenta.
type Alcances interface {
	De(ctx context.Context, usuario string) (AlcanceGuardado, error)
	Responder(ctx context.Context, usuario, pregunta string, r Respuesta) error
	Olvidar(ctx context.Context, usuario, pregunta string) error
	Reemplazar(ctx context.Context, usuario string, rs map[string]Respuesta) error
}

// Procedencia dice de donde salen las respuestas que se estan pintando.
//
// EL VALOR CERO ES «de la direccion», que es lo que esta superficie ha hecho
// siempre y lo que hace cuando no hay cuenta ni almacen. Es el restrictivo en el
// sentido que importa aqui: no afirma que nada este guardado.
type Procedencia uint8

const (
	// DeLaDireccion: las respuestas vienen de la propia URL. Es lo de siempre.
	DeLaDireccion Procedencia = iota
	// DeLaCuenta: vienen del almacen, de la cuenta que esta mirando.
	DeLaCuenta
)

// estadoDelAlcance es lo que la superficie sabe de las respuestas de ESTA
// peticion: cuales son, de donde salen y que se puede hacer con ellas.
type estadoDelAlcance struct {
	Respuestas  Respuestas
	Procedencia Procedencia
	// Quien es el sujeto de la sesion. Vacio si no ha entrado nadie.
	Quien string
	// PuedeGuardar: hay almacen Y hay sesion Y hay token. Las tres o ninguna.
	PuedeGuardar bool
	// Guardado es cuando se escribio por ultima vez lo de esta cuenta. Cero si
	// nunca.
	Guardado time.Time
	// Huerfanas son las respuestas GUARDADAS cuya pregunta el corpus instalado
	// ya no declara.
	//
	// ES LA DIRECCION CONTRARIA DEL EMPAREJAMIENTO, y por eso se cuenta en vez
	// de descartarse en silencio (invariante 7). El emparejamiento entre lo
	// guardado y el corpus casa por el ID DE PREGUNTA: la direccion
	// corpus -> almacen es la que decide que se pinta, y la direccion
	// almacen -> corpus es esta. Sin ella, desinstalar un paquete haria
	// desaparecer respuestas del recuento sin una linea en ningun sitio, que es
	// exactamente el silencio del que este producto se defiende.
	Huerfanas int
	// CSRF es el token de esta peticion, vacio si no hay.
	CSRF string
}

// alcanceDeLaPeticion decide QUE respuestas se pintan y DE DONDE salen.
//
// # La regla, y es una sola linea
//
// Si la direccion trae respuestas, mandan las de la direccion; si no, las de la
// cuenta. Asi un enlace compartido sigue funcionando exactamente como hasta hoy
// (se abre y se ve lo que el enlace dice), y quien entra sin nada en la
// direccion ve LO SUYO. Las dos cosas se pueden distinguir en pantalla y se
// distinguen: pintar las de un enlace diciendo que son las tuyas seria contar
// que has guardado algo que no has guardado.
//
// # Un fallo del almacen NO se degrada a «no has contestado nada»
//
// Es la tentacion de este sitio, y seria la peor: quien abre la pagina veria su
// entrevista en blanco y creeria que su trabajo se ha perdido. Se devuelve el
// error y la pagina que sale es la de error, con su explicacion.
func (s *Superficie) alcanceDeLaPeticion(r *http.Request, m modelo) (estadoDelAlcance, error) {
	e := estadoDelAlcance{Respuestas: De(r.URL.Query(), m.preguntas)}
	if s.alcances == nil {
		return e, nil
	}
	if s.quien != nil {
		e.Quien = strings.TrimSpace(s.quien(r))
	}
	if e.Quien == "" {
		// SIN SESION NO HAY CUENTA, y sin cuenta no hay nada que guardar ni que
		// recuperar. No se cae a un cajon comun: un alcance sin dueno seria de
		// todos, y lo escribiria el primero que pasara.
		return e, nil
	}
	if s.tokens != nil {
		if tok, err := s.tokens(r); err == nil {
			e.CSRF = tok
		}
	}
	// SIN TOKEN NO SE PINTA NINGUN BOTON. Se sigue leyendo lo guardado (eso no
	// muta nada), pero no se ofrece guardar: un formulario sin token es un boton
	// que contesta 403.
	e.PuedeGuardar = e.CSRF != ""

	guardado, err := s.alcances.De(r.Context(), e.Quien)
	if err != nil {
		return e, err
	}
	e.Guardado = guardado.Actualizado
	for id := range guardado.Respuestas {
		if _, existe := m.idx[id]; !existe {
			e.Huerfanas++
		}
	}
	if len(r.URL.Query()[ParamSi]) > 0 || len(r.URL.Query()[ParamNo]) > 0 {
		// LA DIRECCION MANDA. Es un enlace compartido, y lo que hay que ensenar
		// es lo que el enlace dice.
		return e, nil
	}
	e.Respuestas = deLoGuardado(guardado.Respuestas, m.preguntas)
	e.Procedencia = DeLaCuenta
	return e, nil
}

// deLoGuardado construye las respuestas de la pantalla desde lo guardado.
//
// PASA POR LA MISMA PUERTA QUE LA DIRECCION, `De`, y no por un camino propio.
// Es deliberado: `De` es quien sabe que un id que el corpus no declara no entra
// en el estado de la pantalla, y escribir aqui una segunda construccion daria
// dos reglas para lo mismo, que un dia dirian cosas distintas.
func deLoGuardado(rs map[string]Respuesta, conocidas []pantalla.Pregunta) Respuestas {
	v := url.Values{}
	for id, r := range rs {
		switch r {
		case Si:
			v.Add(ParamSi, id)
		case No:
			v.Add(ParamNo, id)
		}
		// SinResponder y Contradictoria no se escriben: el almacen no las
		// guarda, y si alguna llegara, no responder es no tener fila.
	}
	return De(v, conocidas)
}

// guardar es el UNICO manejador que escribe de esta superficie.
func (s *Superficie) guardar(w http.ResponseWriter, r *http.Request) {
	if s.alcances == nil {
		// No deberia poder llegarse: la ruta no se registra sin almacen. Se
		// contesta igual, porque una ruta que solo esta protegida por no
		// existir deja de estarlo el dia que alguien la registre siempre.
		s.fallo(w, r, http.StatusNotFound, "error.no_encontrado")
		return
	}
	quien := ""
	if s.quien != nil {
		quien = strings.TrimSpace(s.quien(r))
	}
	if quien == "" {
		// UNA ESCRITURA SIN AUTOR NO SE ATIENDE. No se rellena con «anonimo»:
		// eso guardaria el alcance de una organizacion en un cajon que leeria
		// el siguiente que pasara por aqui sin entrar.
		s.fallo(w, r, http.StatusForbidden, "error.sin_autor")
		return
	}
	if err := r.ParseForm(); err != nil {
		// El error de ParseForm NO se refleja: puede llevar dentro trozos de lo
		// que mando el cliente.
		s.fallo(w, r, http.StatusBadRequest, "error.formulario_ilegible")
		return
	}
	accion, interpretable := leerAccion(r.PostForm)
	if !interpretable {
		s.fallo(w, r, http.StatusBadRequest, "error.accion_desconocida")
		return
	}

	m := s.instantanea()
	ctx := r.Context()
	pregunta := ""
	if accion.TocaUnaPregunta() {
		var ok bool
		pregunta, ok = leerPreguntaObligatoria(r.PostForm, m.idx)
		if !ok {
			s.fallo(w, r, http.StatusBadRequest, "error.pregunta_desconocida")
			return
		}
	}

	var err error
	switch accion {
	case AccionSi:
		err = s.alcances.Responder(ctx, quien, pregunta, Si)
	case AccionNo:
		err = s.alcances.Responder(ctx, quien, pregunta, No)
	case AccionOlvidar:
		err = s.alcances.Olvidar(ctx, quien, pregunta)
	case AccionLimpiar:
		err = s.alcances.Reemplazar(ctx, quien, map[string]Respuesta{})
	case AccionAdoptar:
		// LAS RESPUESTAS DEL ENVIO PASAN POR `De` ANTES DE GUARDARSE, o sea
		// contra el corpus instalado: lo que un enlace fabricado meta y el
		// corpus no declare no llega al almacen. Y una contradictoria (la misma
		// pregunta que si y que no) NO se guarda de ninguna de las dos formas:
		// elegir una en silencio seria afirmar un alcance que nadie afirmo.
		err = s.alcances.Reemplazar(ctx, quien, aGuardar(De(r.PostForm, m.preguntas)))
	default:
		s.fallo(w, r, http.StatusBadRequest, "error.accion_desconocida")
		return
	}
	if err != nil {
		if s.alFallar != nil {
			s.alFallar(fmt.Errorf("guardando el alcance: %w", err))
		}
		// NO SE DICE «GUARDADO» CUANDO NO SE HA GUARDADO. Redirigir aqui
		// llevaria a la pantalla de siempre y quien la mirara daria por hecho
		// que su respuesta esta dentro.
		s.fallo(w, r, http.StatusInternalServerError, "error.no_se_ha_guardado")
		return
	}

	// EL DESTINO SE COMPONE CON LOS DOS VALORES YA VALIDADOS, no con la
	// peticion: el modo (un booleano) y el id de pregunta, que
	// `leerPreguntaObligatoria` ya ha contrastado contra el corpus instalado. La
	// peticion NO entra en la funcion que compone la direccion.
	//
	// Es lo que hace que el destino no pueda salir de este sitio, y ademas lo
	// que lo hace COMPROBABLE desde fuera: gosec marco este `http.Redirect` con
	// G710 (open redirect por analisis de contaminacion) cuando la peticion
	// viajaba hasta aqui, y tenia razon en lo que puede ver. Ver
	// destinoTrasGuardar.
	largo, interpretable := modoPedido(r.PostForm)
	http.Redirect(w, r, s.destinoTrasGuardar(interpretable && largo == ModoTodas, pregunta),
		http.StatusSeeOther)
}

// aGuardar traduce las respuestas de la pantalla a lo que el almacen admite.
//
// Se quedan fuera SinResponder (que es no tener fila) y Contradictoria (que es
// una entrada que se contradice, y no se resuelve eligiendo una).
func aGuardar(r Respuestas) map[string]Respuesta {
	out := map[string]Respuesta{}
	for _, id := range r.orden {
		switch r.porID[id] {
		case Si:
			out[id] = Si
		case No:
			out[id] = No
		}
	}
	return out
}

// destinoTrasGuardar compone a donde vuelve el navegador despues de escribir.
//
// SIN LAS RESPUESTAS EN LA DIRECCION, que es la mitad que hace util todo esto:
// se vuelve a `/alcance` a secas y la pagina las lee de la cuenta. Lo unico que
// sobrevive es el modo de lista (corta o larga), porque es lo que la persona
// estaba mirando, y el ancla de la pregunta que acaba de contestar, para no
// devolverla al principio de una entrevista de veinte.
//
// # NO RECIBE LA PETICION, y eso es una decision y no un detalle de firma
//
// La primera version tomaba `*http.Request` y leia de el el modo. Funcionaba, y
// **gosec la marco con G710** (open redirect por analisis de contaminacion):
// desde fuera, un valor que sale de la peticion llegaba a la cabecera
// `Location`, y eso es exactamente la forma de un redirect abierto. Que el valor
// estuviera validado por el camino no lo puede ver ni gosec ni quien lea el
// diff.
//
// El arreglo NO es una supresion. Es que aqui entren SOLO los dos valores ya
// juzgados: un booleano, y un id de pregunta que `leerPreguntaObligatoria` ha
// contrastado contra el corpus instalado. Con eso, el destino se compone de
// cosas que esta funcion puede demostrar:
//
//	s.base       validado en la construccion por validarBase (ruta local
//	             absoluta, sin «//» ni contrabarras: ver base.go)
//	rutaDe(...)  constante del modelo
//	q            como mucho `ver=todas`, una constante
//	el ancla     un id del corpus, ademas escapado
//
// Y la otra mitad esta en `leerPreguntaObligatoria`, que devuelve LA COPIA DEL
// CORPUS del identificador y no la cadena que mando el cliente: con eso el valor
// que acaba en `Location` nace del modelo derivado. Las dos mitades juntas
// apagan el G710 sin una sola supresion, y eso importa: suprimir habria dejado
// el mismo codigo con una nota diciendo que alguien lo miro; esto deja un codigo
// en el que no hay nada que mirar.
func (s *Superficie) destinoTrasGuardar(listaLarga bool, pregunta string) string {
	q := url.Values{}
	if listaLarga {
		q.Set(ParamVer, VerTodas)
	}
	destino := s.enlace(rutaDe(pantalla.Alcance), q)
	if pregunta != "" {
		// El identificador viene del corpus (leerPreguntaObligatoria lo ha
		// contrastado contra el indice), asi que no puede traer nada raro. Se
		// escapa igualmente: una cabecera Location es un sitio donde no se
		// confia ni en lo propio.
		destino += "#p-" + url.PathEscape(pregunta)
	}
	return destino
}

// validarPersistencia comprueba que las tres piezas del guardado van juntas.
//
// MEDIO GUARDADO ES PEOR QUE NINGUNO, igual que medio enlace al camino:
//
//	almacen sin Quien   escribiria el alcance de todo el mundo en el mismo sitio
//	almacen sin Tokens  pintaria formularios sin token, o sea botones que
//	                    contestan 403 y nadie sabe por que
//	Quien o Tokens sin almacen  no es un error: son utiles por si solos el dia
//	                    que otra cosa los use, y sin almacen no se registra
//	                    ninguna ruta que mute
func validarPersistencia(o Opciones) error {
	if o.Alcances == nil {
		return nil
	}
	if o.Quien == nil {
		return fmt.Errorf("%w: se ha pasado un almacen de alcances y no se ha pasado Quien, "+
			"asi que no habria forma de saber de quien es lo que se guarda. El alcance de una "+
			"organizacion escrito en un cajon sin dueno lo lee el siguiente que entre. "+
			"Arreglo: pasa las dos, o ninguna", ErrPersistencia)
	}
	if o.Tokens == nil {
		return fmt.Errorf("%w: se ha pasado un almacen de alcances y no se ha pasado Tokens, "+
			"asi que los formularios saldrian sin token CSRF y el middleware de quien monta "+
			"los rechazaria con un 403. Serian botones que no funcionan, que es peor que no "+
			"tener botones. Arreglo: pasa las dos, o ninguna", ErrPersistencia)
	}
	return nil
}

// ErrPersistencia: la configuracion del guardado esta a medias.
var ErrPersistencia = errors.New("configuracion de guardado incompleta")
