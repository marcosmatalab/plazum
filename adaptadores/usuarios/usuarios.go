// Package usuarios es el almacen de cuentas locales de plazum: el que permite
// que alguien ENTRE.
//
// # Que faltaba, y por que costaba tanto
//
// `superficies/serve` tenia construido y probado el mecanismo entero de entrada
// (token de un solo uso para el primer administrador, caducable, CSRF, rotacion
// de sesion al subir de privilegio, rate limit del cubo estrecho) y
// `cmd/plazum/serve.go` montaba el servidor SIN pasarle las tres funciones que
// ese mecanismo necesita. El resultado, medido sobre el binario de verdad: tres
// de los seis pasos del camino guiado contestaban 401, `/primer-admin` contestaba
// 503 con un mensaje escrito para un programador y `/entrar` pedia credenciales
// que no podia tener nadie.
//
// Cada mitad pasaba SU puerta. Lo que no existia era la junta, y la junta es
// este paquete: el almacen de usuarios. Es la misma familia que la primitiva
// `maximo` encendida en el motor y apagada para el corpus.
//
// # Lo que este paquete promete, y lo que no
//
// Promete: que el secreto no se guarda en claro, que no se puede recuperar, que
// dos peticiones simultaneas no crean dos primeros administradores, y que un
// fichero que no se entiende NO se lee como «aqui no hay nadie».
//
// No promete identidad federada: eso es `adaptadores/oidc`, y es el camino
// recomendado para una organizacion con IdP. Esto es lo que tiene que funcionar
// ANTES de que exista IdP, igual que las dos pantallas de arranque de `serve`
// son lo que tiene que funcionar antes de que exista interfaz.
//
// # Por que PBKDF2 de la biblioteca estandar y NINGUNA dependencia nueva
//
// `DEPENDENCIAS.md` dice que a 26-08-2026 el binario se compila con CERO
// dependencias externas, y que el dia que entre la primera hay que cambiar
// `TestElBinarioNoLlevaNingunaDependenciaExterna` a proposito, en el mismo
// commit que su fila. `golang.org/x/crypto` esta en la tabla como PLANEADA, no
// como puesta, y traerla para esto costaria esa fila, ese test y un modulo mas
// que vigilar.
//
// No hace falta: Go 1.24 subio PBKDF2 a la biblioteca estandar (`crypto/pbkdf2`,
// RFC 8018). PBKDF2-HMAC-SHA256 con 600.000 iteraciones es el parametro que
// recomienda la hoja de OWASP para esa combinacion, y es lo que se escribe aqui.
// Lo que NO da PBKDF2 y si darian scrypt o argon2 es dureza frente a hardware
// dedicado (memoria); se dice en voz alta en vez de callarlo, y el dia que el
// producto quiera esa propiedad la decision es anadir la dependencia con su fila,
// no fingir que ya la tiene.
//
// # El coste va DENTRO del fichero, y con suelo
//
// Cada cuenta guarda su algoritmo y su numero de iteraciones, para poder subir
// el coste sin invalidar las cuentas viejas. Y se comprueba un SUELO al cargar:
// un fichero que declare un coste por debajo de `IteracionesMinimas` no carga.
// Sin suelo, bajar el coste a mano seria un downgrade silencioso, que es la
// forma en que este tipo de campo se rompe siempre.
package usuarios

import (
	"bytes"
	"context"
	"crypto/pbkdf2"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/marcosmatalab/plazum/adaptadores/secretos"
	"github.com/marcosmatalab/plazum/puertos"
)

// VersionDelAlmacen es la unica que este binario lee y escribe.
//
// SE EXIGE Y NO SE SUPONE, igual que `incidente.VersionDelRegistro`. Un fichero
// de cuentas sin version se leeria hoy con las reglas de hoy y dentro de un ano
// con las de entonces, en silencio y sobre el mismo contenido. Es la tercera
// forma del invariante 8 aplicada a un formato: el campo ausente no es «la
// version actual», es un dato que falta.
const VersionDelAlmacen = 1

// NombreDelFichero es como se llama el almacen dentro del directorio de datos.
const NombreDelFichero = "usuarios.json"

// Algoritmo es el unico nombre de derivacion que este binario acepta. Va escrito
// DENTRO del fichero para que subir de algoritmo no obligue a adivinar cual se
// uso, y para que un fichero de otro producto no se lea como si fuera nuestro.
const Algoritmo = "pbkdf2-hmac-sha256"

const (
	// IteracionesPorDefecto es lo que se escribe hoy: la recomendacion de OWASP
	// para PBKDF2-HMAC-SHA256.
	IteracionesPorDefecto = 600_000
	// IteracionesMinimas es el suelo que se exige AL CARGAR. Un fichero que
	// declare menos no carga: bajar el coste a mano seria un downgrade que no
	// da error en ningun sitio y deja las contrasenas al alcance de una GPU.
	IteracionesMinimas = 210_000
	// LongitudDeSal en bytes. Sal POR USUARIO, no del almacen: con una sola sal
	// compartida, dos personas con la misma contrasena tienen la misma clave
	// derivada y eso se ve leyendo el fichero.
	LongitudDeSal = 16
	// LongitudDeClave en bytes: 256 bits, el tamano natural de SHA-256.
	LongitudDeClave = 32
	// LongitudMinimaDelSecreto es el suelo de la contrasena. Coincide a
	// proposito con el que exige la pantalla de primer administrador de
	// `superficies/serve`: si el almacen fuera mas permisivo, la pantalla seria
	// la unica guarda, y la pantalla no es la unica puerta de entrada.
	LongitudMinimaDelSecreto = 12
	// LongitudMaximaDelSecreto acota lo que se deriva. Sin tope, una contrasena
	// de diez megabytes convierte cada intento de entrada en trabajo gratis
	// para quien la manda: es denegacion de servicio con forma de credencial.
	LongitudMaximaDelSecreto = 1024
	// LongitudMaximaDelUsuario en runas.
	LongitudMaximaDelUsuario = 64
)

// Los centinelas. Cada uno se puede comprobar con errors.Is desde la orden que
// monta, que es quien decide si plazum arranca o no.
var (
	// ErrAlmacenIlegible: el fichero existe y no es lo que dice ser.
	ErrAlmacenIlegible = errors.New("el almacen de usuarios no se puede leer")
	// ErrAlmacenVacio: el fichero existe y no tiene ni un byte.
	//
	// ES UN ERROR Y NO «AQUI NO HAY NADIE», y es la decision mas importante de
	// este paquete. Un fichero de cero bytes donde deberia haber cuentas no es
	// una instalacion nueva: es una escritura que se corto, un disco lleno o un
	// `> usuarios.json` de alguien. Leerlo como «no hay administrador» REABRE
	// la ventana de instalacion en un sistema que ya estaba instalado, o sea
	// que convierte un fichero truncado en una toma de control. El valor cero
	// en esta frontera tiene que ser el restrictivo (invariante 8), y el
	// restrictivo aqui es negarse.
	ErrAlmacenVacio = errors.New("el almacen de usuarios existe y esta vacio")
	// ErrAlmacenSinVersion: falta la version del formato.
	ErrAlmacenSinVersion = errors.New("el almacen de usuarios no dice su version")
	// ErrVersionDesconocida: una version que este binario no sabe leer.
	ErrVersionDesconocida = errors.New("version del almacen de usuarios desconocida")
	// ErrCosteInsuficiente: una cuenta declara menos coste que el suelo.
	ErrCosteInsuficiente = errors.New("una cuenta declara menos coste de derivacion que el minimo")
	// ErrCredenciales: usuario o contrasena incorrectos. UNO SOLO para los dos
	// casos: dos centinelas distintos serian enumeracion de usuarios servida
	// por el propio tipo de error.
	ErrCredenciales = errors.New("usuario o contrasena incorrectos")
	// ErrYaHayCuentas: se pidio crear el PRIMER administrador y ya hay alguien.
	ErrYaHayCuentas = errors.New("este almacen ya tiene cuentas")
	// ErrUsuarioNoValido: el nombre de usuario no sirve.
	ErrUsuarioNoValido = errors.New("nombre de usuario no valido")
	// ErrSecretoNoValido: la contrasena no cumple el minimo.
	ErrSecretoNoValido = errors.New("contrasena no valida")
	// ErrSinRuta: se pidio abrir un almacen sin decir donde vive.
	ErrSinRuta = errors.New("almacen de usuarios sin ruta")
)

// cuenta es una cuenta ya validada, en memoria.
//
// EL SECRETO NO ESTA AQUI Y NO PUEDE ESTARLO: lo que se guarda es la clave
// derivada, y de ella no se vuelve. No hay ningun metodo en este paquete que
// devuelva ni la clave ni la sal, y no es un olvido.
type cuenta struct {
	usuario     string
	iteraciones int
	sal         []byte
	clave       []byte
	creado      time.Time
}

// cuentaEnDisco es la forma serializada.
//
// LOS PARAMETROS VIAJAN CON LA CUENTA y no en una cabecera del fichero: asi
// subir el coste de derivacion es escribir cuentas nuevas con el coste nuevo,
// sin invalidar las viejas ni tener que migrar el fichero entero de golpe.
type cuentaEnDisco struct {
	Usuario     string `json:"usuario"`
	Algoritmo   string `json:"algoritmo"`
	Iteraciones int    `json:"iteraciones"`
	Sal         string `json:"sal"`
	Clave       string `json:"clave"`
	Creado      string `json:"creado"`
}

type almacenEnDisco struct {
	Version int             `json:"version"`
	Cuentas []cuentaEnDisco `json:"cuentas"`
}

// Almacen es el conjunto de cuentas de una instalacion, respaldado por un
// fichero. Seguro para uso concurrente: es lo que atiende las peticiones HTTP.
type Almacen struct {
	mu          sync.Mutex
	ruta        string
	cuentas     []cuenta
	iteraciones int
	sec         puertos.Secretos
	ahora       func() time.Time

	// salDeRelleno es la sal con la que se deriva cuando el usuario NO EXISTE.
	// Existe para que un usuario inexistente cueste lo mismo que uno existente:
	// sin ella, la diferencia de tiempo entre «no esta» (respuesta inmediata) y
	// «esta y la contrasena falla» (600.000 iteraciones) es un listado de
	// usuarios servido por el reloj.
	salDeRelleno []byte
}

// Opciones para abrir un almacen.
type Opciones struct {
	// Ruta del fichero. Obligatoria: un almacen sin ruta es un almacen que se
	// pierde al parar el proceso, y un administrador que hay que volver a crear
	// en cada arranque no es un administrador.
	Ruta string
	// Secretos es la fuente de aleatoriedad de las sales. Nil usa
	// adaptadores/secretos, que es crypto/rand y nada mas.
	Secretos puertos.Secretos
	// Reloj para la fecha de creacion. Nil es time.Now.
	Reloj func() time.Time
	// Iteraciones de PBKDF2 para las cuentas que se creen a partir de ahora.
	// Cero usa IteracionesPorDefecto. Por debajo de IteracionesMinimas, Abrir
	// falla: aflojar el coste desde fuera es lo unico que este parametro podria
	// servir para hacer.
	Iteraciones int
}

// Abrir lee el almacen del disco, o devuelve uno vacio si el fichero no existe.
//
// LAS TRES FORMAS DE LA NADA, que no son la misma (invariante 8):
//
//	fichero AUSENTE            instalacion nueva. Almacen vacio, sin error. Es
//	                           el unico caso en que «no hay nadie» es cierto.
//	fichero PRESENTE Y VACIO   ErrAlmacenVacio. No es una instalacion nueva: es
//	                           un fichero roto, y tomarlo por nuevo reabre la
//	                           ventana del primer administrador.
//	fichero PRESENTE Y NO
//	INTERPRETABLE              error, siempre. Un JSON que no casa, una version
//	                           que no conocemos, una sal que no es hexadecimal
//	                           o un coste por debajo del suelo son datos que HAY
//	                           y no se entienden, nunca el valor por defecto.
func Abrir(o Opciones) (*Almacen, error) {
	ruta := strings.TrimSpace(o.Ruta)
	if ruta == "" {
		return nil, fmt.Errorf("%w. Arreglo para quien monta: pasa Opciones.Ruta con el "+
			"fichero donde viven las cuentas de esta instalacion", ErrSinRuta)
	}
	a := &Almacen{
		ruta:        ruta,
		iteraciones: o.Iteraciones,
		sec:         o.Secretos,
		ahora:       o.Reloj,
	}
	if a.iteraciones == 0 {
		a.iteraciones = IteracionesPorDefecto
	}
	if a.iteraciones < IteracionesMinimas {
		return nil, fmt.Errorf("%w: se han pedido %d iteraciones y el minimo son %d. "+
			"Arreglo: no bajes este parametro; si lo que quieres es que los tests corran "+
			"rapido, reduce el numero de contrasenas que derivan, no su coste",
			ErrCosteInsuficiente, a.iteraciones, IteracionesMinimas)
	}
	if a.sec == nil {
		a.sec = secretos.Nuevo()
	}
	if a.ahora == nil {
		a.ahora = time.Now
	}
	a.salDeRelleno = make([]byte, LongitudDeSal)
	if err := a.sec.Bytes(a.salDeRelleno); err != nil {
		return nil, fmt.Errorf("no se puede preparar el almacen de usuarios: %w", err)
	}

	// #nosec G304 -- la ruta la elige el operador con --usuarios en su propia
	// maquina, igual que --datos en adaptadores/latido.
	b, err := os.ReadFile(ruta)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		// LA NADA DE VERDAD: instalacion nueva. Sin error y sin aviso.
		return a, nil
	case err != nil:
		return nil, fmt.Errorf("%w: no puedo leer %s: %w.\n"+
			"Arreglo: comprueba que el fichero es legible por el usuario que ejecuta "+
			"plazum, o elige otro sitio con --usuarios", ErrAlmacenIlegible, ruta, err)
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return nil, fmt.Errorf("%w: %s tiene %d bytes.\n"+
			"NO se lee como «todavia no hay administrador»: si se leyera asi, plazum "+
			"volveria a imprimir un token de instalacion y cualquiera que llegue antes "+
			"que tu se quedaria con una instalacion que YA estaba instalada.\n"+
			"Arreglo: restaura el fichero de tu copia de seguridad. Si de verdad esta "+
			"instalacion es nueva y ese fichero sobra, borralo (no lo vacies) y vuelve "+
			"a arrancar", ErrAlmacenVacio, ruta, len(b))
	}

	var doc almacenEnDisco
	// DisallowUnknownFields no se usa a proposito: un campo de mas de una
	// version futura tiene que poder ignorarse. Lo que NO se ignora es un campo
	// de menos, que es lo que comprueba cuentaDeDisco.
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("%w: %s no es un almacen de usuarios en JSON: %w.\n"+
			"Arreglo: restauralo de tu copia de seguridad. No lo edites a mano: las "+
			"contrasenas de dentro estan derivadas y no se pueden volver a escribir",
			ErrAlmacenIlegible, ruta, err)
	}
	if doc.Version == 0 {
		return nil, fmt.Errorf("%w: %s no dice con que version del formato se escribio. "+
			"Sin ella se leeria hoy con las reglas de hoy y manana con las de manana, en "+
			"silencio. Arreglo: si el fichero es de plazum, restauralo de tu copia; si lo "+
			"escribio otra cosa, no es un almacen de plazum", ErrAlmacenSinVersion, ruta)
	}
	if doc.Version != VersionDelAlmacen {
		return nil, fmt.Errorf("%w: %s dice version %d y este plazum lee la %d.\n"+
			"Arreglo: si el fichero es mas nuevo, actualiza plazum; si es mas viejo, usa "+
			"la version de plazum que lo escribio para migrarlo",
			ErrVersionDesconocida, ruta, doc.Version, VersionDelAlmacen)
	}

	vistos := make(map[string]bool, len(doc.Cuentas))
	for i, c := range doc.Cuentas {
		leida, err := cuentaDeDisco(ruta, i, c)
		if err != nil {
			return nil, err
		}
		if vistos[leida.usuario] {
			return nil, fmt.Errorf("%w: %s trae dos cuentas con el nombre %q. Con dos "+
				"cuentas del mismo nombre, cual manda depende del orden del fichero, y el "+
				"orden no lo firma nadie", ErrAlmacenIlegible, ruta, leida.usuario)
		}
		vistos[leida.usuario] = true
		a.cuentas = append(a.cuentas, leida)
	}
	return a, nil
}

// cuentaDeDisco valida una cuenta leida. TODO campo que falte, no se entienda o
// se salga de rango es un error, nunca un valor por defecto.
func cuentaDeDisco(ruta string, i int, c cuentaEnDisco) (cuenta, error) {
	donde := fmt.Sprintf("%s, cuenta %d", ruta, i+1)
	nombre, err := NormalizarUsuario(c.Usuario)
	if err != nil {
		return cuenta{}, fmt.Errorf("%w: %s: %w", ErrAlmacenIlegible, donde, err)
	}
	if c.Algoritmo != Algoritmo {
		// Vacio y desconocido se tratan igual A PROPOSITO: los dos significan
		// «no se con que se derivo esto», y suponer el nuestro seria comparar
		// una clave contra otra funcion.
		return cuenta{}, fmt.Errorf("%w: %s declara el algoritmo %q y este plazum solo "+
			"sabe %q. Un algoritmo que no se conoce no se puede suponer: la comparacion "+
			"daria siempre falso y nadie podria entrar",
			ErrAlmacenIlegible, donde, c.Algoritmo, Algoritmo)
	}
	if c.Iteraciones <= 0 {
		return cuenta{}, fmt.Errorf("%w: %s no dice cuantas iteraciones se usaron. Sin ese "+
			"numero la clave no se puede recalcular", ErrAlmacenIlegible, donde)
	}
	if c.Iteraciones < IteracionesMinimas {
		return cuenta{}, fmt.Errorf("%w: %s declara %d iteraciones y el suelo son %d. "+
			"Arreglo: esa cuenta se creo con un coste que ya no se acepta; borrala del "+
			"fichero y vuelve a crearla",
			ErrCosteInsuficiente, donde, c.Iteraciones, IteracionesMinimas)
	}
	sal, err := hex.DecodeString(strings.TrimSpace(c.Sal))
	if err != nil || len(sal) != LongitudDeSal {
		return cuenta{}, fmt.Errorf("%w: %s no trae una sal de %d bytes en hexadecimal",
			ErrAlmacenIlegible, donde, LongitudDeSal)
	}
	clave, err := hex.DecodeString(strings.TrimSpace(c.Clave))
	if err != nil || len(clave) != LongitudDeClave {
		return cuenta{}, fmt.Errorf("%w: %s no trae una clave derivada de %d bytes en "+
			"hexadecimal", ErrAlmacenIlegible, donde, LongitudDeClave)
	}
	// EL INSTANTE QUE NO SE ENTIENDE ES UN ERROR, NUNCA EL CERO. El cero de
	// time.Time es el ano 1, y de ahi salen fechas de creacion con cara de dato.
	creado, err := time.Parse(time.RFC3339, strings.TrimSpace(c.Creado))
	if err != nil {
		return cuenta{}, fmt.Errorf("%w: %s no trae un instante de creacion RFC3339 "+
			"(2026-09-03T09:00:00Z)", ErrAlmacenIlegible, donde)
	}
	return cuenta{
		usuario:     nombre,
		iteraciones: c.Iteraciones,
		sal:         sal,
		clave:       clave,
		creado:      creado.UTC(),
	}, nil
}

// Ruta dice donde vive este almacen. Se usa para contarselo al operador en el
// arranque: un fichero de credenciales cuyo sitio no se dice es un fichero que
// nadie incluye en su copia de seguridad.
func (a *Almacen) Ruta() string { return a.ruta }

// Cuentas dice cuantas hay. No devuelve los nombres: este tipo no tiene ningun
// metodo que enumere quien esta dentro, porque no hace falta para nada de lo
// que hoy monta el producto y porque una superficie que lo tuviera acabaria
// ensenandolo.
func (a *Almacen) Cuentas() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.cuentas)
}

// HayAdministrador dice si esta instalacion ya tiene alguna cuenta.
//
// La firma es la de serve.Config.HayAdmin. El contexto no se usa hoy porque el
// almacen es local; esta en la firma porque el puerto lo pide y porque el
// almacen que venga despues (SQLite, LDAP) si lo necesitara.
func (a *Almacen) HayAdministrador(_ context.Context) (bool, error) {
	return a.Cuentas() > 0, nil
}

// CrearPrimerAdministrador crea la cuenta inicial. La firma es la de
// serve.Config.CrearAdmin.
//
// # LA CARRERA DEL PRIMER ADMINISTRADOR
//
// Dos peticiones simultaneas a /primer-admin con el mismo token no pueden crear
// dos cuentas. `superficies/serve` ya aparta el token mientras una creacion esta
// en curso, pero apoyarse SOLO en eso seria confiar en que la unica puerta de
// entrada a este almacen es esa pantalla, y no lo es: la orden de terminal que
// venga manana tambien llamara aqui.
//
// Asi que la exclusion vive TAMBIEN aqui, y es la del propio almacen: el mutex
// cubre la comprobacion y la escritura JUNTAS. Comprobar fuera del candado y
// escribir dentro es la version clasica de este fallo, y deja pasar las dos.
func (a *Almacen) CrearPrimerAdministrador(ctx context.Context, usuario, secreto string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.cuentas) > 0 {
		return fmt.Errorf("%w, asi que la instalacion ya esta hecha. Si has perdido la "+
			"contrasena, no hay forma de recuperarla y es a proposito: se recupera "+
			"borrando el almacen de usuarios y volviendo a instalar", ErrYaHayCuentas)
	}
	return a.crearBloqueado(ctx, usuario, secreto)
}

// crearBloqueado escribe la cuenta. Se llama con el candado tomado.
func (a *Almacen) crearBloqueado(_ context.Context, usuario, secreto string) error {
	nombre, err := NormalizarUsuario(usuario)
	if err != nil {
		return err
	}
	if err := ComprobarSecreto(secreto); err != nil {
		return err
	}
	for _, c := range a.cuentas {
		if c.usuario == nombre {
			return fmt.Errorf("%w: ya hay una cuenta que se llama %q", ErrUsuarioNoValido, nombre)
		}
	}
	sal := make([]byte, LongitudDeSal)
	if err := a.sec.Bytes(sal); err != nil {
		return fmt.Errorf("no se puede crear la cuenta sin aleatoriedad: %w", err)
	}
	clave, err := derivar(secreto, sal, a.iteraciones)
	if err != nil {
		return err
	}
	nueva := cuenta{
		usuario:     nombre,
		iteraciones: a.iteraciones,
		sal:         sal,
		clave:       clave,
		creado:      a.ahora().UTC(),
	}
	// SE ESCRIBE ANTES DE ACEPTARLA EN MEMORIA. Al reves, un fallo de disco
	// dejaria un administrador que existe en este proceso y desaparece al
	// reiniciar, y el operador se encontraria la ventana de instalacion abierta
	// otra vez sin entender por que.
	if err := a.guardar(append(append([]cuenta(nil), a.cuentas...), nueva)); err != nil {
		return err
	}
	a.cuentas = append(a.cuentas, nueva)
	return nil
}

// Autenticar comprueba unas credenciales y devuelve el sujeto de la sesion. La
// firma es la de serve.Config.Autenticar.
//
// # No dice si el usuario existe, ni por el mensaje ni por el reloj
//
// Por el MENSAJE: hay un solo centinela, ErrCredenciales, para los dos casos.
// `serve` ya pinta el mismo texto para todo fallo, pero si el error distinguiera,
// la primera orden de terminal que lo imprima seria un listado de usuarios.
//
// Por el RELOJ: cuando el usuario no existe se deriva IGUALMENTE, con una sal de
// relleno y el coste configurado, y se compara contra un valor que no puede
// casar. Sin eso, «no existe» contesta en microsegundos y «existe y la
// contrasena falla» tarda un cuarto de segundo, y esa diferencia se mide desde
// fuera sin ninguna herramienta especial.
func (a *Almacen) Autenticar(_ context.Context, usuario, secreto string) (string, error) {
	a.mu.Lock()
	cuentas := make([]cuenta, len(a.cuentas))
	copy(cuentas, a.cuentas)
	relleno := a.salDeRelleno
	iteraciones := a.iteraciones
	a.mu.Unlock()

	nombre, err := NormalizarUsuario(usuario)
	if err != nil {
		// Un nombre imposible no se puede normalizar, asi que no hay a quien
		// buscar. Se deriva igual, por lo mismo que abajo.
		nombre = ""
	}
	// El secreto se acota ANTES de derivar: derivar diez megabytes es trabajo
	// que paga el servidor y elige quien ataca.
	if len(secreto) > LongitudMaximaDelSecreto {
		secreto = secreto[:LongitudMaximaDelSecreto]
	}

	var hallada *cuenta
	for i := range cuentas {
		// Se recorren TODAS y no se sale al encontrar: salir antes hace que el
		// tiempo dependa de la POSICION de la cuenta en el fichero. Son unas
		// pocas cuentas, asi que el recorrido entero no cuesta nada.
		if subtle.ConstantTimeCompare([]byte(cuentas[i].usuario), []byte(nombre)) == 1 {
			hallada = &cuentas[i]
		}
	}

	sal, coste, esperada := relleno, iteraciones, make([]byte, LongitudDeClave)
	if hallada != nil {
		sal, coste, esperada = hallada.sal, hallada.iteraciones, hallada.clave
	}
	obtenida, err := derivar(secreto, sal, coste)
	if err != nil {
		return "", err
	}
	// La comparacion es en tiempo constante SIEMPRE, tambien en el camino del
	// usuario inexistente: si solo se hiciera en el otro, la rama barata volveria
	// a distinguirse.
	casa := subtle.ConstantTimeCompare(obtenida, esperada) == 1
	if hallada == nil || !casa {
		return "", ErrCredenciales
	}
	return hallada.usuario, nil
}

// derivar es la unica funcion de este paquete que toca la contrasena.
func derivar(secreto string, sal []byte, iteraciones int) ([]byte, error) {
	clave, err := pbkdf2.Key(sha256.New, secreto, sal, iteraciones, LongitudDeClave)
	if err != nil {
		// El error NO lleva el secreto ni la sal dentro: es lo unico que hay que
		// cuidar aqui, porque este mensaje acaba en el log del operador.
		return nil, fmt.Errorf("no se puede derivar la clave (%d iteraciones): %w",
			iteraciones, err)
	}
	return clave, nil
}

// NormalizarUsuario devuelve el nombre canonico de un usuario, o un error.
//
// Se normaliza a minusculas para que «CISO» y «ciso» no sean dos cuentas: dos
// cuentas que se ven iguales al leerlas son un fallo de auditoria antes que uno
// de seguridad, porque el acta diria que dos personas distintas hicieron lo que
// hizo una.
//
// LOS DOS PUNTOS ESTAN PROHIBIDOS, y no es cosmetica: `superficies/serve` reserva
// sujetos con espacio de nombres (el de la sesion anonima es
// «anonimo:sin-autenticar») y ademas se niega, con un 500, a abrir sesion para
// ese sujeto. Un usuario que pudiera llamarse asi obligaria a este paquete a
// conocer esa cadena, o sea a que un adaptador importara una superficie. Se
// prohibe la FORMA en vez del valor concreto, que es lo que no envejece.
func NormalizarUsuario(usuario string) (string, error) {
	n := strings.ToLower(strings.TrimSpace(usuario))
	if n == "" {
		return "", fmt.Errorf("%w: el nombre de usuario no puede ir vacio", ErrUsuarioNoValido)
	}
	if len([]rune(n)) > LongitudMaximaDelUsuario {
		return "", fmt.Errorf("%w: el nombre de usuario pasa de %d caracteres",
			ErrUsuarioNoValido, LongitudMaximaDelUsuario)
	}
	if strings.ContainsRune(n, ':') {
		return "", fmt.Errorf("%w: el nombre de usuario no puede llevar dos puntos. plazum "+
			"reserva esa forma para los sujetos internos de una sesion, asi que un usuario "+
			"que la use no podria entrar nunca. Arreglo: quita los dos puntos",
			ErrUsuarioNoValido)
	}
	for _, r := range n {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return "", fmt.Errorf("%w: el nombre de usuario no puede llevar espacios ni "+
				"caracteres de control. Un nombre con un espacio invisible dentro se ve "+
				"igual que otro y no es el mismo", ErrUsuarioNoValido)
		}
	}
	return n, nil
}

// ComprobarSecreto dice si una contrasena sirve, sin decir nada de su contenido.
func ComprobarSecreto(secreto string) error {
	n := len([]rune(secreto))
	if n < LongitudMinimaDelSecreto {
		return fmt.Errorf("%w: la contrasena necesita al menos %d caracteres y tiene %d. "+
			"Es la unica credencial de esta instalacion, asi que el suelo es alto a "+
			"proposito", ErrSecretoNoValido, LongitudMinimaDelSecreto, n)
	}
	if len(secreto) > LongitudMaximaDelSecreto {
		return fmt.Errorf("%w: la contrasena pasa de %d bytes. El tope existe para que "+
			"nadie pueda hacer trabajar al servidor mandando una contrasena enorme",
			ErrSecretoNoValido, LongitudMaximaDelSecreto)
	}
	return nil
}

// guardar escribe el almacen entero de forma atomica: fichero temporal en el
// MISMO directorio, sincronizado, y rename encima.
//
// El rename dentro del mismo sistema de ficheros es atomico, asi que no existe
// el instante en el que el fichero de cuentas esta a medias. Sin esto, un corte
// de corriente durante la escritura deja exactamente el fichero de cero bytes
// que Abrir se niega a leer, y el operador tendria que restaurar una copia por
// un fallo que se puede no tener.
func (a *Almacen) guardar(cuentas []cuenta) error {
	doc := almacenEnDisco{Version: VersionDelAlmacen}
	for _, c := range cuentas {
		doc.Cuentas = append(doc.Cuentas, cuentaEnDisco{
			Usuario:     c.usuario,
			Algoritmo:   Algoritmo,
			Iteraciones: c.iteraciones,
			Sal:         hex.EncodeToString(c.sal),
			Clave:       hex.EncodeToString(c.clave),
			Creado:      c.creado.UTC().Format(time.RFC3339),
		})
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("no puedo serializar el almacen de usuarios: %w", err)
	}
	b = append(b, '\n')

	dir := filepath.Dir(a.ruta)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("no puedo crear el directorio %s del almacen de usuarios: %w. "+
			"Arreglo: comprueba los permisos, o elige otro sitio con --usuarios", dir, err)
	}
	tmp, err := os.CreateTemp(dir, "usuarios-*.json.tmp")
	if err != nil {
		return fmt.Errorf("no puedo escribir en %s: %w. Arreglo: comprueba los permisos "+
			"del directorio", dir, err)
	}
	nombreTmp := tmp.Name()
	limpiar := func() { _ = tmp.Close(); _ = os.Remove(nombreTmp) }
	// CreateTemp ya crea con 0600, pero se fija explicitamente: este fichero
	// lleva claves derivadas y el resto de la maquina no tiene nada que hacer
	// con ellas.
	if err := os.Chmod(nombreTmp, 0o600); err != nil {
		limpiar()
		return fmt.Errorf("no puedo dejar el almacen de usuarios en 0600: %w", err)
	}
	if _, err := tmp.Write(b); err != nil {
		limpiar()
		return fmt.Errorf("no puedo escribir el almacen de usuarios: %w", err)
	}
	// Sin Sync, el rename puede quedar en el diario y el CONTENIDO no: el
	// resultado es el fichero de cero bytes que Abrir rechaza.
	if err := tmp.Sync(); err != nil {
		limpiar()
		return fmt.Errorf("no puedo sincronizar el almacen de usuarios a disco: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(nombreTmp)
		return fmt.Errorf("no puedo cerrar el almacen de usuarios: %w", err)
	}
	if err := os.Rename(nombreTmp, a.ruta); err != nil {
		_ = os.Remove(nombreTmp)
		return fmt.Errorf("no puedo dejar el almacen de usuarios en %s: %w. Arreglo: "+
			"comprueba los permisos del directorio", a.ruta, err)
	}
	return nil
}

// RutaPorDefecto compone donde vive el almacen dentro del directorio de datos.
func RutaPorDefecto(datos string) string {
	d := strings.TrimSpace(datos)
	if d == "" {
		d = "."
	}
	return filepath.Join(d, NombreDelFichero)
}
