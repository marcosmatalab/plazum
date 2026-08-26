// Package latido es el pulso de vida de una instalacion, y el vigilante que
// avisa cuando ese pulso se para.
//
// # El problema que resuelve
//
// El fallo que mata a un producto que vende "no pierdas nunca la conformidad"
// no es que el ordenador se apague, porque eso se nota. Es que el planificador
// deje de correr ciclos mientras todo lo demas sigue en pie: la pantalla abre,
// el corpus esta instalado, y los plazos vencen sin que nadie los mire. Desde
// fuera, una instalacion muerta y una instalacion sin nada que vencer se ven
// exactamente igual.
//
// # Las tres capas del aviso, y el limite de cada una
//
// Ninguna de las tres es suficiente sola, y por eso estan las tres. Lo que
// importa es que la que de verdad avisa NO PASA POR NOSOTROS:
//
//	la pantalla Hoy   dice si el planificador late, con los relojes del
//	                  operador. Limite: hay que entrar a mirarla.
//	`plazum latido`   el mismo veredicto en la terminal, con CODIGO DE SALIDA:
//	                  1 si lleva mas de 24 h callado. Se engancha a un cron, a
//	                  un timer de systemd o al monitor que el operador ya
//	                  tenga, y no depende de que nosotros estemos vivos.
//	                  Limite: si la maquina entera muere, no corre nadie.
//	el pulso          la instalacion manda un pulso al destino que el operador
//	                  configure. Cubre el caso de la maquina muerta. Limite,
//	                  y es grande: si el destino es el nuestro, NO PODEMOS
//	                  AVISARLE, porque no tenemos su correo a proposito (ver
//	                  QueSeManda). Por eso el destino se puede cambiar: quien
//	                  lo apunte a su propio monitor de "dead man's switch"
//	                  recibe el aviso de su propio sistema, sin nosotros en
//	                  medio.
//
// La direccion del aviso es lo unico que de verdad importa de esta pieza. Si el
// veredicto dependiera de que nuestro receptor conteste, UNA CAIDA NUESTRA SE
// LEERIA COMO UNA CAIDA SUYA, y en dos semanas el operador habria aprendido a
// ignorar el rojo. Por eso la regla vive en nucleo/pantalla.Vigilar, se calcula
// con los relojes del operador, y no mira el canal.
//
// # El opt-in
//
// Apagado por defecto, y una sola forma de encenderlo: Activar, que registra el
// consentimiento con el texto literal de lo que se acepto. Un producto
// autoalojado cuya tesis es que el receptor no se fia del emisor no puede
// mandar telemetria sin permiso explicito. Lo que sale cabe en dos lineas
// (QueSeManda) y esta comprobado contra una LISTA BLANCA, no contra una lista
// negra: la pregunta no es "¿va aqui algo prohibido?", es "¿es esto exactamente
// lo unico que va?".
//
// # Lo que este paquete no puede hacer, por construccion
//
// No importa ni el corpus, ni el expediente, ni el estado de cumplimiento, y no
// puede: hay un test que lee sus imports y se pone rojo si aparece cualquier
// paquete de nucleo/ que no sea el modelo de pantalla. Un adaptador de
// telemetria que puede leer el estado de cumplimiento acaba mandandolo.
package latido

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"plazum/nucleo/pantalla"
	"plazum/puertos"
)

// QueSeManda es la declaracion de lo que sale de la maquina. DOS LINEAS, y esa
// restriccion es la puerta: si la lista de campos no cabe en dos lineas de
// documentacion, es que se esta mandando demasiado.
//
// Se guarda LITERALMENTE en el fichero de estado al activar, con la fecha. Asi
// el consentimiento no es un booleano que alguien puso a true: es el texto que
// se acepto, en el disco del operador, para poder mirarlo un ano despues.
const QueSeManda = "Se manda un identificador aleatorio de esta instalacion, que generamos aqui y no se deriva de nada tuyo, y el instante del pulso.\n" +
	"Nada mas: ni tu nombre, ni el de tu organizacion, ni tu direccion, ni que paquetes normativos tienes, ni nada de tu estado de cumplimiento."

// DestinoPorDefecto es el receptor del pulso. Dominio provisional, atado a la
// casilla de marca de la semana 0.
//
// Es sustituible a proposito, y no por comodidad de pruebas: quien lo apunte a
// su propio monitor recibe el aviso de su propio sistema y nos saca a nosotros
// de en medio, que es la unica forma de que la capa 3 avise de verdad.
const DestinoPorDefecto = "https://plazum.dev/latido"

// NombreDelFichero es donde vive el estado, dentro del directorio de datos.
const NombreDelFichero = "latido.json"

// IntervaloDePulso es cada cuanto sale un pulso como mucho: uno al dia.
//
// El ciclo del planificador se programa cada hora (asi esta el ejemplo de
// systemd en docs/latido.md), y el pulso NO sale en cada ciclo. Son dos cosas
// distintas con dos ritmos distintos: el ciclo es del operador y cuanto mas a
// menudo mejor, el pulso es hacia fuera y cuanto menos, mejor. La casilla dice
// "pulso diario", asi que diario.
//
// Un pulso que falla no actualiza la marca, o sea que el siguiente ciclo
// vuelve a intentarlo: reintento por hora mientras el canal este roto, uno al
// dia cuando funciona. Eso sale gratis de comparar contra el ultimo pulso
// ACEPTADO y es justo lo que se quiere.
const IntervaloDePulso = 24 * time.Hour

// BytesDeInstancia es el tamano del identificador de instalacion: 16 bytes,
// 128 bits, en hexadecimal.
//
// Es aleatorio y NO se deriva del nombre de la maquina, ni de una MAC, ni de
// nada del operador. Un identificador derivado es un identificador de la
// organizacion con otro nombre, y ademas seria el mismo en dos instalaciones
// que restauren la misma copia.
const BytesDeInstancia = 16

// Errores, como centinelas. Se comparan con errors.Is y nunca por texto.
var (
	// ErrApagado: se ha pedido pulsar y el latido no esta activado.
	ErrApagado = errors.New("el latido esta apagado")
	// ErrEstadoIlegible: el fichero de estado no se puede leer o no cuadra.
	ErrEstadoIlegible = errors.New("el estado del latido no se puede leer")
	// ErrDestinoInseguro: el destino no vale como destino de un pulso.
	ErrDestinoInseguro = errors.New("destino de latido invalido")
	// ErrSinConsentimiento: hay estado activado sin consentimiento escrito.
	ErrSinConsentimiento = errors.New("latido activado sin consentimiento registrado")
)

// Pulso es TODO lo que sale de la maquina del operador. Dos campos.
//
// Es una estructura y no un mapa a proposito: un mapa admite una clave mas sin
// que nadie lo note, y una estructura obliga a que anadir un campo sea una
// linea en un diff. La lista blanca de frontera_test.go lo comprueba ademas
// sobre los bytes que de verdad viajan.
type Pulso struct {
	// Instancia es el identificador aleatorio de esta instalacion.
	Instancia string `json:"instancia"`
	// Instante es cuando se emite el pulso, en UTC.
	Instante time.Time `json:"instante"`
}

// Consentimiento es el permiso del operador, con el texto que acepto.
type Consentimiento struct {
	Otorgado time.Time `json:"otorgado"`
	Texto    string    `json:"texto"`
}

// Estado es lo que vive en <datos>/latido.json.
//
// Guarda el opt-in, el consentimiento y las marcas de vida. Las marcas van aqui
// y no en otro fichero porque el ciclo del planificador y el pulso se escriben
// juntos: separarlos abre la puerta a que uno se quede atras y el vigilante
// juzgue con una marca vieja.
type Estado struct {
	// Activado es el opt-in. El cero es APAGADO, y por eso es un bool y no
	// un puntero: el valor por defecto de este producto tiene que ser el que
	// sale de una estructura en cero.
	Activado bool `json:"activado"`
	// Instancia es el identificador aleatorio. Vacio mientras no se active.
	Instancia string `json:"instancia,omitempty"`
	// Destino es a donde va el pulso. Vacio significa DestinoPorDefecto.
	Destino string `json:"destino,omitempty"`
	// Consentimiento es el permiso, con su fecha y su texto.
	Consentimiento *Consentimiento `json:"consentimiento,omitempty"`

	// UltimoCiclo es cuando el planificador termino su ultimo ciclo. ESTA
	// marca es la que decide el veredicto, y se escribe aunque el pulso
	// falle: el canal hacia nosotros no puede llevarse por delante la
	// constancia de que su planificador corrio.
	UltimoCiclo time.Time `json:"ultimo_ciclo,omitempty"`
	// UltimoPulso es el ultimo pulso ACEPTADO por el destino.
	UltimoPulso time.Time `json:"ultimo_pulso,omitempty"`
	// FalloElUltimoIntento dice si el ultimo intento no llego. Es lo que
	// hace util el smoke test: probar y verlo, sin esperar 24 horas.
	FalloElUltimoIntento bool `json:"fallo_el_ultimo_intento,omitempty"`
}

// Marcas traduce el estado al modelo que juzga nucleo/pantalla.
//
// La traduccion es tonta a proposito: la regla de los umbrales vive en el
// nucleo, con el instante entrando como dato, y aqui no se decide nada. Si
// alguien copiara aqui un "if han pasado 24 horas", habria dos reglas y un dia
// dirian cosas distintas.
func (e Estado) Marcas() pantalla.Marcas {
	return pantalla.Marcas{
		UltimoCiclo:          e.UltimoCiclo,
		LatidoActivado:       e.Activado,
		UltimoPulso:          e.UltimoPulso,
		FalloElUltimoIntento: e.FalloElUltimoIntento,
	}
}

// DestinoEfectivo es el destino configurado, o el de por defecto.
func (e Estado) DestinoEfectivo() string {
	if strings.TrimSpace(e.Destino) == "" {
		return DestinoPorDefecto
	}
	return e.Destino
}

// ruta compone la ruta del fichero de estado dentro del directorio de datos.
func ruta(datos string) string { return filepath.Join(datos, NombreDelFichero) }

// Cargar lee el estado del latido.
//
// SIN FICHERO, APAGADO. No es un caso de error y no lleva aviso: una
// instalacion recien hecha no tiene telemetria activada, y ese es el estado
// correcto, no un problema que resolver.
func Cargar(datos string) (Estado, error) {
	b, err := os.ReadFile(ruta(datos)) // #nosec G304 -- el directorio de datos lo elige el operador en su maquina
	if errors.Is(err, os.ErrNotExist) {
		return Estado{}, nil
	}
	if err != nil {
		return Estado{}, fmt.Errorf("%w: no puedo leer %s: %w. Arreglo: comprueba los "+
			"permisos del directorio de datos, o borra el fichero para volver a empezar "+
			"con el latido apagado", ErrEstadoIlegible, ruta(datos), err)
	}
	var e Estado
	if err := json.Unmarshal(b, &e); err != nil {
		return Estado{}, fmt.Errorf("%w: %s no es un estado de latido en JSON: %w. "+
			"Arreglo: borra el fichero y vuelve a activar el latido si lo quieres",
			ErrEstadoIlegible, ruta(datos), err)
	}
	// Un estado activado sin consentimiento escrito no se acepta. Es la
	// unica forma de que el opt-in no se pueda encender editando un booleano
	// a mano y luego decir que el producto lo mando solo.
	//
	// Junto al centinela se devuelve el estado PARSEADO con el opt-in forzado
	// a apagado, y no Estado{}. Devolver el cero aqui era el agujero: Activar
	// y Desactivar toleran este centinela a proposito y guardan lo que Cargar
	// les devuelve, o sea que seguir el arreglo que este mismo error imprime
	// (`plazum latido desactivar`) BORRABA la marca del ultimo ciclo. Un
	// planificador muerto hace seis dias volvia a salir en verde con el rotulo
	// "no ha corrido ningun ciclo todavia, si acabas de instalar es lo
	// normal", y `plazum latido` pasaba de codigo 1 a codigo 0. Un problema
	// del LATIDO no puede llevarse por delante la constancia de que el
	// planificador corrio: es la propiedad entera de la pieza.
	//
	// Forzar Activado a false es lo que mantiene el opt-in cerrado: quien
	// ignore el error se encuentra un estado apagado, y ademas Pulsar
	// comprueba consentimiento e instancia por su cuenta.
	sinOptIn := e
	sinOptIn.Activado = false
	if e.Activado && (e.Consentimiento == nil || strings.TrimSpace(e.Consentimiento.Texto) == "") {
		return sinOptIn, fmt.Errorf("%w: %s dice activado y no trae el texto que se "+
			"acepto ni cuando. Arreglo: `plazum latido desactivar` y, si lo quieres "+
			"encendido, `plazum latido activar`, que registra el consentimiento",
			ErrSinConsentimiento, ruta(datos))
	}
	if e.Activado && e.Instancia == "" {
		return sinOptIn, fmt.Errorf("%w: %s dice activado y no trae identificador de "+
			"instalacion. Arreglo: `plazum latido desactivar` y volver a activar",
			ErrSinConsentimiento, ruta(datos))
	}
	return e, nil
}

// Guardar escribe el estado. 0600: no tiene secretos dentro, pero tampoco tiene
// por que leerlo el resto de la maquina.
func Guardar(datos string, e Estado) error {
	b, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return fmt.Errorf("no puedo serializar el estado del latido: %w", err)
	}
	if err := os.MkdirAll(datos, 0o750); err != nil {
		return fmt.Errorf("no puedo crear el directorio de datos %s: %w. Arreglo: "+
			"comprueba los permisos o elige otro con --datos", datos, err)
	}
	if err := os.WriteFile(ruta(datos), append(b, '\n'), 0o600); err != nil {
		return fmt.Errorf("no puedo escribir %s: %w. Arreglo: comprueba los permisos "+
			"del directorio de datos", ruta(datos), err)
	}
	return nil
}

// Activar enciende el latido. Es la UNICA forma de encenderlo.
//
// Registra el consentimiento con el texto literal de QueSeManda y la fecha, y
// genera el identificador de instalacion con la fuente de secretos que se le
// pase. El identificador no se deriva de nada del operador: es aleatorio.
//
// Activar sobre un latido ya activado NO regenera el identificador: seria
// perder el hilo del pulso por teclear dos veces el mismo comando.
func Activar(datos string, destino string, ahora time.Time, s puertos.Secretos) (Estado, error) {
	if s == nil {
		return Estado{}, errors.New("no hay fuente de secretos para generar el " +
			"identificador de instalacion. Arreglo: construir el comando con " +
			"adaptadores/secretos.Nuevo()")
	}
	e, err := Cargar(datos)
	if err != nil && !errors.Is(err, ErrSinConsentimiento) {
		return Estado{}, err
	}
	if destino = strings.TrimSpace(destino); destino != "" {
		if err := ComprobarDestino(destino); err != nil {
			return Estado{}, err
		}
		e.Destino = destino
	}
	if err := ComprobarDestino(e.DestinoEfectivo()); err != nil {
		return Estado{}, err
	}
	if e.Instancia == "" {
		id, err := s.Token(BytesDeInstancia)
		if err != nil {
			return Estado{}, fmt.Errorf("no puedo generar el identificador de "+
				"instalacion: %w", err)
		}
		e.Instancia = id
	}
	e.Activado = true
	e.Consentimiento = &Consentimiento{Otorgado: ahora.UTC(), Texto: QueSeManda}
	if err := Guardar(datos, e); err != nil {
		return Estado{}, err
	}
	return e, nil
}

// Desactivar apaga el latido y BORRA el identificador de instalacion y el
// consentimiento.
//
// Borrarlo no es limpieza: es que volver a activar mas tarde tiene que generar
// un identificador nuevo. Si se conservara, quien recibe los pulsos podria
// enlazar el "antes" y el "despues" de una baja, que es exactamente lo que
// alguien que se da de baja no quiere.
//
// Las marcas del planificador SI se conservan: son suyas, no nuestras, y son lo
// que hace que el aviso de las 24 horas siga funcionando con el latido apagado.
func Desactivar(datos string) (Estado, error) {
	e, err := Cargar(datos)
	if err != nil && !errors.Is(err, ErrSinConsentimiento) {
		return Estado{}, err
	}
	e.Activado = false
	e.Instancia = ""
	e.Consentimiento = nil
	e.UltimoPulso = time.Time{}
	e.FalloElUltimoIntento = false
	if err := Guardar(datos, e); err != nil {
		return Estado{}, err
	}
	return e, nil
}

// ComprobarDestino rechaza lo que no vale como destino de un pulso.
//
// Las cuatro razones, y ninguna es de estilo:
//
//	solo https      un pulso por http lo ve cualquiera del camino. Se admite
//	                http SOLO contra localhost, que es como se prueba un
//	                receptor propio antes de publicarlo.
//	sin consulta    la parte de consulta de una direccion es justo donde se
//	                cuela un identificador ("?org=acme"), y ademas acaba en
//	                los logs de todos los intermediarios.
//	sin usuario     unas credenciales en la direccion son un secreto en el
//	                fichero de configuracion y en los logs.
//	sin fragmento   no significa nada en una peticion y solo sirve para
//	                esconder algo a la vista.
func ComprobarDestino(destino string) error {
	u, err := url.Parse(destino)
	if err != nil {
		return fmt.Errorf("%w: %q no es una direccion: %w. Arreglo: usa algo como %s",
			ErrDestinoInseguro, destino, err, DestinoPorDefecto)
	}
	local := esLocal(u.Hostname())
	if u.Scheme != "https" && !(u.Scheme == "http" && local) {
		return fmt.Errorf("%w: %q va por %q. El pulso sale de la red del operador, asi "+
			"que va por https; http solo se admite contra localhost, para probar un "+
			"receptor propio. Arreglo: usa https://",
			ErrDestinoInseguro, destino, u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("%w: %q no dice a que maquina. Arreglo: usa algo como %s",
			ErrDestinoInseguro, destino, DestinoPorDefecto)
	}
	if u.RawQuery != "" {
		return fmt.Errorf("%w: %q lleva parte de consulta (?...). Ahi es donde se cuela un "+
			"identificador sin que nadie lo note, y ademas acaba en los logs de cada "+
			"intermediario. Arreglo: quita el ?", ErrDestinoInseguro, destino)
	}
	if u.User != nil {
		return fmt.Errorf("%w: %q lleva usuario o contrasena en la direccion, o sea un "+
			"secreto en un fichero de configuracion y en los logs. Arreglo: quitalo",
			ErrDestinoInseguro, destino)
	}
	if u.Fragment != "" {
		return fmt.Errorf("%w: %q lleva fragmento (#...), que no significa nada en una "+
			"peticion. Arreglo: quitalo", ErrDestinoInseguro, destino)
	}
	return nil
}

// esLocal dice si un nombre de maquina es la propia maquina. Se compara contra
// nombres, no contra rangos: aqui no se resuelve nada, porque resolver seria
// salir a la red para validar una configuracion.
func esLocal(host string) bool {
	switch strings.ToLower(host) {
	case "localhost", "127.0.0.1", "::1", "[::1]":
		return true
	}
	return false
}

// Pulsar manda UN pulso y devuelve el estado con las marcas actualizadas.
//
// Con el latido apagado devuelve ErrApagado y NO TOCA EL CANAL. Es la puerta
// del opt-in en el unico sitio donde puede fallar de verdad: da igual lo que
// diga la interfaz si la funcion que manda paquetes no pregunta.
func Pulsar(ctx context.Context, e Estado, c Canal, ahora time.Time) (Estado, error) {
	if !e.Activado {
		return e, fmt.Errorf("%w. Arreglo: `plazum latido activar` si quieres encenderlo; "+
			"si no lo quieres, no hace falta hacer nada, el aviso de las 24 horas "+
			"funciona igual sin el", ErrApagado)
	}
	if e.Consentimiento == nil {
		return e, ErrSinConsentimiento
	}
	if e.Instancia == "" {
		return e, ErrSinConsentimiento
	}
	if c == nil {
		return e, errors.New("no hay canal por el que mandar el pulso. Arreglo: " +
			"construir el cliente con latido.CanalHTTP{}")
	}
	destino := e.DestinoEfectivo()
	if err := ComprobarDestino(destino); err != nil {
		return e, err
	}
	cuerpo, err := json.Marshal(Pulso{Instancia: e.Instancia, Instante: ahora.UTC()})
	if err != nil {
		return e, fmt.Errorf("no puedo serializar el pulso: %w", err)
	}
	if err := c.Entregar(ctx, destino, cuerpo); err != nil {
		e.FalloElUltimoIntento = true
		return e, err
	}
	e.UltimoPulso = ahora.UTC()
	e.FalloElUltimoIntento = false
	return e, nil
}

// Ciclo apunta que el planificador ha corrido y, si el latido esta encendido,
// manda el pulso. Es lo que llama el cron o el timer de systemd.
//
// EL ORDEN IMPORTA Y ES LA PROPIEDAD DE ESTA FUNCION: la marca del ciclo se
// escribe SIEMPRE, tambien cuando el pulso falla. Si un fallo de red hacia
// nosotros dejara sin escribir la marca del ciclo, nuestra caida se convertiria
// en el aviso "tu planificador lleva 24 horas muerto", que es exactamente la
// mentira que esta pieza existe para no contar.
//
// Devuelve el estado guardado y el error del pulso, si lo hubo. El error del
// pulso NO es el error de la funcion para quien mira el planificador: se
// informa, y el estado ya guardado dice que el canal fallo.
func Ciclo(ctx context.Context, datos string, c Canal, ahora time.Time) (Estado, error) {
	e, err := Cargar(datos)
	if err != nil {
		return Estado{}, err
	}
	e.UltimoCiclo = ahora.UTC()
	var fallo error
	if tocaPulsar(e, ahora) {
		e, fallo = Pulsar(ctx, e, c, ahora)
	}
	if err := Guardar(datos, e); err != nil {
		return e, err
	}
	return e, fallo
}

// tocaPulsar dice si a este ciclo le toca mandar pulso. Ver IntervaloDePulso.
func tocaPulsar(e Estado, ahora time.Time) bool {
	if !e.Activado {
		return false
	}
	if e.UltimoPulso.IsZero() {
		return true
	}
	if e.UltimoPulso.After(ahora) {
		// El reloj se ha movido hacia atras, o la marca esta en el futuro.
		// Si se comparara a secas, la resta saldria negativa y el pulso no
		// volveria a salir NUNCA. Se pulsa, que es el lado seguro para el
		// operador: lo que no puede pasar es que deje de pulsar en silencio.
		return true
	}
	return ahora.Sub(e.UltimoPulso) >= IntervaloDePulso
}

// Probar es el smoke test del canal: manda un pulso de verdad, por el canal de
// verdad, y deja escrito el resultado.
//
// Manda un pulso normal y no uno "de prueba", y eso es deliberado: un smoke
// test que usa un camino distinto del real prueba el camino distinto. Y un
// campo "es_prueba" en el pulso seria un campo mas de los dos que caben en la
// declaracion de QueSeManda.
//
// No toca la marca del ciclo: probar el canal no es haber corrido un ciclo, y
// confundirlos dejaria al operador tranquilo con un planificador parado.
func Probar(ctx context.Context, datos string, c Canal, ahora time.Time) (Estado, error) {
	e, err := Cargar(datos)
	if err != nil {
		return Estado{}, err
	}
	e, fallo := Pulsar(ctx, e, c, ahora)
	if errors.Is(fallo, ErrApagado) {
		// Con el latido apagado no hay nada que probar y no se escribe
		// nada: el estado en disco no puede cambiar porque alguien haya
		// preguntado.
		return e, fallo
	}
	if err := Guardar(datos, e); err != nil {
		return e, err
	}
	return e, fallo
}
