package camino

import (
	"net/http"
	"net/url"
	"strings"
)

// EL CONMUTADOR DE IDIOMA, y por que vive aqui.
//
// Vive en `superficies/camino` por lo mismo que la plantilla del armazon: lo
// necesitan las OCHO plantillas de las SIETE superficies con pantalla, y una
// pieza copiada siete veces se separa el dia que una cambie. Este paquete ya es
// el sitio compartido de las pantallas (`ConArmazon`, `TiraDe`, `InicioDe`), asi
// que no se inventa un sitio nuevo.
//
// # QUE PROBLEMA RESUELVE, medido antes de escribirlo
//
// Hasta hoy el idioma lo decidia SOLO `Accept-Language`: las siete superficies
// llamaban a `motor.Resolver(r.Header.Get("Accept-Language"))` y no habia forma
// de pedir otro. Un ingles con el navegador en castellano no podia leer plazum
// en ingles, y el catalogo tiene las 586 claves traducidas desde hace semanas.
//
// # EL MECANISMO: parametro de consulta que se persiste en cookie
//
// Se eligio esto y no un formulario POST ni una ruta propia. Los tres motivos,
// que son restricciones de esta casa y no gusto:
//
//  1. CERO JAVASCRIPT. La CSP por defecto (`serve.CSPPorDefecto`) no admite nada
//     inline, y el armazon prohibe `on*` y `style=`. Un conmutador que dependa de
//     un script no se puede pintar aqui. Un enlace si.
//  2. LAS PANTALLAS SON GET-ONLY y tienen puerta que lo vigila
//     (`TestNingunaRutaDelConjuntoMutaSinPasarPorCSRF`). Un POST obligaria a
//     meter el token CSRF en las ocho plantillas, incluidas las que hoy se
//     sirven sin sesion, que es donde no hay token que poner.
//  3. UNA RUTA NUEVA tendria que entrar en la enumeracion del servidor y en la
//     comprobacion de CSRF, y ademas rompe el enlace compartible: con parametro,
//     `/calendario/?idioma=en` se puede pegar en un correo y llega en ingles.
//
// # POR QUE UN GET QUE ESCRIBE UNA COOKIE NO ES UNA MUTACION, dicho porque
// parece que lo es
//
// No toca expediente, ni ledger, ni alcance, ni nada del servidor: la cookie es
// una preferencia de PRESENTACION que vive en el navegador que la pidio. Nadie
// mas la ve y no cambia ni una respuesta del producto para otra cuenta. La
// puerta de rutas mutantes vigila que POST/PUT/DELETE/PATCH no sean atendidos, y
// eso sigue igual: aqui no se registra ninguna ruta nueva.
//
// # LAS TRES FORMAS DE LA NADA (invariante 8), contestadas una por una
//
//	parametro AUSENTE             se pasa a la cookie, y si no hay, a
//	                              Accept-Language. Es la nada de verdad.
//	parametro PRESENTE Y VACIO    `?idioma=` no es «ponme el de por defecto»:
//	                              es una preferencia que falta. Mismo trato que
//	                              ausente, y NO se escribe cookie.
//	parametro PRESENTE Y RARO     `?idioma=zz` es un dato que hay y no se
//	                              entiende. NO se acepta y sobre todo NO SE
//	                              PERSISTE: lo que el invariante prohibe es
//	                              convertir lo no interpretable en un valor, y
//	                              persistirlo seria justo eso, ademas para
//	                              siempre. Se sirve el idioma que tocara.
//
// Y no se contesta con un error a proposito: el conjunto de idiomas es CERRADO y
// sale del motor, los enlaces del conmutador solo generan valores validos, y un
// 400 romperia un enlace compartido por un cambio de version. Lo que importa del
// invariante se cumple donde importa, que es que el dato raro no entra a ningun
// sitio donde se pueda volver a leer.
//
// LO VIGILA: TestElIdiomaElegidoManda, en las dos direcciones y con las tres
// formas de la nada recorridas.

const (
	// CookieIdioma es donde se recuerda la eleccion.
	//
	// # POR QUE NO LLEVA EL PREFIJO __Host-, QUE SI LLEVA LA DE SESION
	//
	// Porque __Host- EXIGE Secure, y acertar con Secure aqui no se puede sin la
	// configuracion del servidor: detras de un proxy que termina el TLS,
	// `r.TLS` es nil aunque el cliente venga por https, y `X-Forwarded-Proto`
	// solo vale si el operador ha declarado que hay un proxy suyo delante
	// (`serve.Config.ProxiesDeConfianza`). Esta cookie la escriben las
	// superficies, que no conocen esa configuracion, y hacer que la conozcan
	// obligaria a que `serve` importe las pantallas o al reves: hoy `serve` no
	// depende de ninguna superficie, y no se rompe eso por una preferencia de
	// presentacion.
	//
	// Y la asimetria con la de sesion esta justificada por lo que cuesta
	// equivocarse, que es lo que decide: un Secure mal puesto hace que el
	// navegador ACEPTE la cookie y no la devuelva NUNCA, en silencio, que es el
	// fallo que `serve.avisoDeCookieQueNoVolvera` existe para diagnosticar. En
	// la de sesion eso es un bucle de entrada; aqui seria un conmutador que no
	// conmuta y nadie sabria por que. A cambio, lo que se arriesga es que un
	// subdominio comprometido te plante el idioma de la interfaz, que no es una
	// escalada: no da acceso a nada y se deshace con un clic.
	//
	// NO ES UNA CREDENCIAL y por eso admite una regla mas laxa. La de sesion no.
	CookieIdioma = "plazum_idioma"

	// ParametroIdioma es el nombre del parametro de consulta del conmutador.
	ParametroIdioma = "idioma"

	// MaxEdadIdioma es cuanto dura la eleccion: un año.
	//
	// No es una sesion: quien elige leer en ingles quiere leer en ingles mañana
	// tambien, y una preferencia que se olvida al cerrar el navegador se lee
	// como que el conmutador no funciona.
	MaxEdadIdioma = 365 * 24 * 60 * 60
)

// MotorDeIdiomas es lo que este paquete necesita saber de quien resuelve
// idiomas, que es lo justo: cuales hay cargados y a cual cae uno pedido.
//
// Es un interfaz declarado por QUIEN LO USA y no un puerto del producto, que es
// lo idiomatico en Go y lo que evita que `adaptadores/plantilla` tenga que
// importar una superficie. Lo implementa `*plantilla.Motor` sin tocarlo.
type MotorDeIdiomas interface {
	// Resolver dice a que idioma cargado cae uno pedido. Nunca falla.
	Resolver(idioma string) string
	// Idiomas son los cargados; el primero es el de por defecto.
	Idiomas() []string
}

// Elegido dice en que idioma hay que servir esta peticion, y si hay que
// recordarlo.
//
// EL ORDEN DE PRECEDENCIA, que es la decision entera:
//
//  1. el parametro de consulta, si nombra un idioma cargado. Es un acto
//     deliberado de quien mira y gana a todo.
//  2. la cookie, si nombra un idioma cargado. Es el mismo acto, de antes.
//  3. Accept-Language, que es lo que habia y sigue siendo el buen defecto.
//  4. el idioma por defecto del motor, que es donde cae Resolver.
//
// El segundo valor es `true` solo cuando la eleccion viene del PARAMETRO: es el
// unico caso en el que hay algo nuevo que recordar. Volver a escribir la cookie
// en cada peticion que ya la traia solo serviria para renovar su caducidad, y
// eso convierte una preferencia en algo que no caduca nunca mientras navegues.
//
// EL VALOR CERO DE m ES EL RESTRICTIVO: con `m` nil no hay idiomas cargados que
// validar, asi que no se acepta ninguna eleccion y se devuelve cadena vacia, que
// es lo que las superficies ya tratan como «resuelvelo tu». No se inventa un
// idioma.
func Elegido(r *http.Request, m MotorDeIdiomas) (string, bool) {
	if r == nil || m == nil {
		return "", false
	}
	cargados := m.Idiomas()

	// 1. EL PARAMETRO. Se lee de la URL cruda y no de r.FormValue: FormValue
	// consume el cuerpo en un POST, y estas pantallas son GET pero el interfaz
	// no puede depender de eso.
	if pedido := r.URL.Query().Get(ParametroIdioma); pedido != "" {
		if elegido, vale := cargado(pedido, cargados); vale {
			return elegido, true
		}
		// Presente y no interpretable: se cae al resto SIN persistir nada.
	}

	// 2. LA COOKIE.
	if c, err := r.Cookie(CookieIdioma); err == nil && c.Value != "" {
		if elegido, vale := cargado(c.Value, cargados); vale {
			return elegido, false
		}
		// Cookie presente y no interpretable: se cae a Accept-Language. No se
		// borra ni se reescribe, porque esta funcion solo LEE: quien decide
		// escribir es RecordarIdioma, y lo hace con lo que el usuario pide.
	}

	// 3 y 4. Lo de siempre.
	return m.Resolver(r.Header.Get("Accept-Language")), false
}

// cargado dice si un idioma pedido es uno de los que hay, y devuelve el cargado.
//
// Compara por la etiqueta primaria en minusculas ("en-GB" y "EN" son "en"),
// igual que hace el catalogo, para que el conmutador no dependa de como escriba
// el codigo quien componga el enlace.
//
// NO SE USA m.Resolver PARA VALIDAR, y esa es la trampa que hay que ver:
// `Resolver` NUNCA FALLA, cae al idioma por defecto ante cualquier basura. Si se
// usara para validar, `?idioma=zz` resolveria a "es" y se persistiria "es" como
// si alguien lo hubiera elegido. Un dato no interpretable convertido en un valor
// y guardado: exactamente lo que el invariante 8 prohibe. Por eso se compara
// contra la lista.
func cargado(pedido string, cargados []string) (string, bool) {
	p := primaria(pedido)
	if p == "" {
		return "", false
	}
	for _, c := range cargados {
		if primaria(c) == p {
			return c, true
		}
	}
	return "", false
}

// primaria reduce un locale a su etiqueta primaria en minusculas.
func primaria(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if i := strings.IndexAny(s, "-_"); i >= 0 {
		s = s[:i]
	}
	return s
}

// RecordarIdioma resuelve el idioma de esta peticion y lo persiste si viene
// elegido a proposito.
//
// ES EL UNICO PUNTO DE ENTRADA que necesitan las superficies, y va al principio
// de su ServeHTTP: ahi hay `w`, hay `r`, y se ejecuta una vez por peticion.
// Repartir la llamada por cada punto de render habria dado varias escrituras de
// la misma cookie en una respuesta.
//
// Devuelve el idioma para quien lo quiera, pero las superficies siguen leyendolo
// con `Elegido` donde lo necesitan: esta funcion existe por el efecto, no por el
// valor.
func RecordarIdioma(w http.ResponseWriter, r *http.Request, m MotorDeIdiomas) string {
	idioma, hayQueRecordar := Elegido(r, m)
	if hayQueRecordar {
		Persistir(w, r, idioma)
	}
	return idioma
}

// Persistir escribe la cookie con el idioma elegido.
//
// EL ATRIBUTO Secure SALE DE LA PETICION y no de una configuracion, por lo que
// dice el comentario de CookieIdioma: las superficies no conocen la del
// servidor. `r.TLS != nil` se queda CORTO detras de un proxy que termina el TLS
// (la cookie viajaria sin Secure entre proxy y cliente), y eso es lo que se
// acepta a cambio de que nunca se pierda en silencio. Pasarse en la otra
// direccion romperia el conmutador sin decir nada.
//
// HttpOnly va puesto aunque no haya JavaScript que la lea: no hay ningun caso en
// el que el navegador tenga que leerla, y un atributo restrictivo que no cuesta
// nada se pone.
func Persistir(w http.ResponseWriter, r *http.Request, idioma string) {
	if w == nil || r == nil || idioma == "" {
		return
	}
	// #nosec G124 -- HttpOnly y SameSite van fijos; Secure sale de si la
	// peticion llego por TLS y por eso gosec no puede verlo constante. Su valor
	// real lo comprueba TestLaCookieDelIdiomaLlevaSusAtributos.
	http.SetCookie(w, &http.Cookie{
		Name:     CookieIdioma,
		Value:    idioma,
		Path:     "/",
		MaxAge:   MaxEdadIdioma,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		// Lax y no Strict por lo mismo que la de sesion: llegar desde el enlace
		// de un correo tiene que respetar el idioma que elegiste.
		SameSite: http.SameSiteLaxMode,
	})
}

// OpcionDeIdioma es un idioma en el conmutador, ya con su enlace compuesto.
type OpcionDeIdioma struct {
	// Codigo es la etiqueta ("es", "en"). Va al atributo lang y hreflang, y NO
	// se traduce: es un codigo, no prosa.
	Codigo string
	// Clave es la clave de catalogo con el nombre del idioma.
	//
	// EL NOMBRE DE CADA IDIOMA SE ESCRIBE EN SU PROPIO IDIOMA y no en el de la
	// pagina: quien tiene la interfaz en castellano y busca el ingles busca
	// «English», no «Inglés». Es la convencion de todo conmutador de idioma que
	// funciona, y la razon es que quien lo necesita es justo quien no lee el
	// idioma actual. Lo consigue el catalogo teniendo la MISMA cadena en los dos
	// ficheros, y hay una puerta que lo exige.
	Clave string
	// URL es esta misma pagina con el idioma cambiado.
	URL string
	// Actual dice si es el que se esta sirviendo.
	Actual bool
}

// OpcionesDeIdioma compone el conmutador para esta pagina.
//
// EL ENLACE APUNTA A LA MISMA PAGINA con solo el idioma cambiado. Es la mitad
// que hace util un conmutador: uno que te manda a la portada te obliga a volver
// a navegar hasta donde estabas, y entonces se deja de usar a la segunda vez.
//
// # LA CONSULTA LA PONE QUIEN LLAMA, Y NO SE LEE DE LA PETICION
//
// Es el mismo contrato que `TiraDe`, y por el mismo motivo, que costo un rojo:
// la primera version copiaba `r.URL.Query()` entera al enlace de cada idioma, y
// `TestUnaRespuestaInventadaNoEntraEnLaPagina` la puso roja. Ese test protege una
// propiedad escrita con estas palabras: una respuesta a una pregunta que no
// existe «ni se cuenta, ni se ensena, NI SE COPIA A LOS ENLACES de la pagina».
// Copiar la consulta cruda reflejaba entrada arbitraria del cliente en el HTML.
//
// No era una inyeccion (html/template escapa), y da igual: la propiedad que
// rompia es la que impide que una peticion adversaria se convierta en parte de
// la pagina, y esa no se negocia por comodidad.
//
// Asi que la consulta entra YA SANEADA por quien sabe que significa: las
// pantallas pasan `Respuestas.Consulta().Encode()`, que se reconstruye desde las
// respuestas VALIDAS; el camino pasa la suya reescrita desde `url.Values`; y las
// superficies que no llevan estado en la direccion pasan cadena vacia. Este
// paquete no puede saber que parametro es legitimo en una pantalla que no
// conoce, y adivinarlo es como se cuela lo que no se entiende.
//
// SI HAY UN SOLO IDIOMA CARGADO NO SE DEVUELVE NADA, y el armazon entonces no
// pinta el conmutador. Es el valor cero restrictivo: un conmutador con una sola
// opcion no conmuta, ocupa sitio y sugiere que hay algo que no hay.
func OpcionesDeIdioma(ruta string, m MotorDeIdiomas, actual, consulta string) []OpcionDeIdioma {
	if m == nil || ruta == "" {
		return nil
	}
	cargados := m.Idiomas()
	if len(cargados) < 2 {
		return nil
	}
	// La consulta que llega ya esta codificada por quien la compuso. Se parsea
	// para poder sustituir el parametro del idioma sin pegar cadenas, y lo que
	// no se entienda se descarta entero: es la tercera forma de la nada, y aqui
	// la salida segura es un enlace sin consulta y no un enlace con basura.
	q, err := url.ParseQuery(consulta)
	if err != nil {
		q = url.Values{}
	}
	q.Del(ParametroIdioma)

	out := make([]OpcionDeIdioma, 0, len(cargados))
	for _, c := range cargados {
		propia := url.Values{}
		for k, vs := range q {
			propia[k] = vs
		}
		propia.Set(ParametroIdioma, c)
		destino := ruta
		if e := propia.Encode(); e != "" {
			destino += "?" + e
		}
		out = append(out, OpcionDeIdioma{
			Codigo: c,
			Clave:  ClaveDeIdioma(c),
			URL:    destino,
			Actual: primaria(c) == primaria(actual),
		})
	}
	return out
}

// ClaveDeIdioma es la clave de catalogo con el nombre de un idioma.
//
// Se compone y no se escribe una por una para que añadir el tercer idioma sea
// una linea en los ficheros de cadenas y no un cambio de codigo, que es la misma
// regla que el invariante 2 aplica al corpus: lo que es dato vive en datos.
func ClaveDeIdioma(codigo string) string {
	return "ui.idioma." + primaria(codigo)
}

// ClavesDeIdioma son las claves que pide el conmutador para unos idiomas dados.
//
// Se suma a ClavesDelArmazon en el test de cobertura de catalogo de cada
// superficie, igual que las demas del armazon.
func ClavesDeIdioma(idiomas []string) []string {
	out := []string{"ui.idioma.rotulo"}
	for _, i := range idiomas {
		out = append(out, ClaveDeIdioma(i))
	}
	return out
}
