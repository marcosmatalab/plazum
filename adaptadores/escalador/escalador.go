// Package escalador es el lazo que junta las dos mitades del escalado: lee el
// plan que decide el nucleo, manda lo que toque, y anota lo que paso.
//
// EL TIEMPO ENTRA AQUI Y NO EN EL NUCLEO. Este es el unico sitio de la cadena
// que sabe que hora es: `nucleo/escalado` recibe el instante como dato y no
// llama a `time.Now()` nunca. Por eso el lazo vive en un adaptador y no dentro
// del motor (invariante 1).
//
// ENTREGA AL MENOS UNA VEZ, Y SE DICE. Exactamente-una-vez NO EXISTE aqui y
// pretenderlo seria mentir: el lazo puede morir entre el envio y el registro, y
// desde fuera esos dos mundos — mande y no lo apunte, o no llegue a mandar — son
// indistinguibles. Asi que se elige al-menos-una-vez y se escribe la eleccion.
//
// LO QUE HACE QUE ESA ELECCION SEA SOPORTABLE ES LA CLAVE DE DEDUPLICACION.
// Sin ella, un reinicio re-manda los 53 avisos del corpus y el operador aprende
// en una tarde a filtrar el remitente, que es el fin de la pieza. Con ella, un
// reinicio re-manda COMO MUCHO UNO: el que estaba en vuelo. La clave es
// obligacion + hito + ciclo + nivel, y el ciclo es el instante del vencimiento,
// que es lo que distingue la revision de este ano de la del que viene.
//
// EL REGISTRO VA ANTES QUE EL ENVIO, y esa es toda la diferencia entre "no se
// si se mando" y "no hay ni rastro". Se anota la INTENCION, se envia, se anota
// el CURSO. Un lazo que muere en medio deja una intencion sin curso, que es un
// dato: dice que ese aviso estaba en vuelo. Al reves — enviar y luego anotar —
// dejaria un hueco, y un hueco no se distingue de no haber pasado nunca.
//
// LA VIDA DEL LAZO ALIMENTA EL LATIDO, no se duplica. Un planificador que deja
// de correr mientras todo lo demas sigue en pie es exactamente el fallo que el
// latido existe para vigilar, asi que al cerrar cada vuelta se marca el ciclo
// con `latido.Ciclo` y la regla de las 24 horas sigue viviendo en
// `nucleo/pantalla`, donde estaba. Copiar aqui un "si han pasado 24 horas"
// crearia dos reglas que un dia dirian cosas distintas.
package escalador

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/escalado"
	"github.com/marcosmatalab/plazum/nucleo/ventana"
	"github.com/marcosmatalab/plazum/puertos"
)

var (
	ErrSinDiario  = errors.New("el lazo no tiene diario")
	ErrSinCanal   = errors.New("el lazo no tiene canal")
	ErrDiarioRoto = errors.New("el diario no se puede leer")
)

// ---------------------------------------------------------------------------
// La clave de deduplicacion
// ---------------------------------------------------------------------------

// Clave identifica UN aviso concreto de UN ciclo concreto.
//
// EL CICLO ES EL INSTANTE DEL VENCIMIENTO, y no un contador nuestro. Un
// contador se reinicia con el proceso y dos ciclos distintos acabarian con la
// misma clave; el vencimiento sale del reloj legal y distingue la revision de
// este ano de la del que viene sin que nadie tenga que llevar la cuenta.
type Clave struct {
	Obligacion string
	Hito       string
	Vence      time.Time
	Nivel      int
}

func (c Clave) String() string {
	return fmt.Sprintf("%s|%s|%s|%d", c.Obligacion, c.Hito,
		c.Vence.UTC().Format(time.RFC3339), c.Nivel)
}

// ---------------------------------------------------------------------------
// El diario
// ---------------------------------------------------------------------------

// TipoApunte es el vocabulario cerrado del diario.
type TipoApunte string

const (
	// Intencion: se va a mandar. Se escribe ANTES de mandar.
	Intencion TipoApunte = "intencion"
	// Curso: que paso con el envio. Se escribe DESPUES.
	Curso TipoApunte = "curso"
	// Suprimido: no se manda, y por que (colapso, silencio, sin destinatario).
	// Se anota igual que un envio: un aviso que no sale por decision tiene que
	// dejar constancia de la decision.
	Suprimido TipoApunte = "suprimido"
)

// Apunte es una linea del diario. Append-only, una por linea, JSON.
type Apunte struct {
	Clave    string          `json:"clave"`
	Tipo     TipoApunte      `json:"tipo"`
	Instante time.Time       `json:"instante"`
	Estado   escalado.Estado `json:"estado,omitempty"`
	Canal    string          `json:"canal,omitempty"`
	Motivo   string          `json:"motivo,omitempty"`
}

// Diario es el registro persistente del lazo.
//
// Es JSON por lineas y se escribe con fsync antes de cada envio. El fsync no es
// celo: sin el, "el registro va antes que el envio" es una intencion del codigo
// y no un hecho del disco, y un corte de corriente deja el orden al azar del
// cache del sistema de ficheros.
type Diario struct {
	ruta    string
	vidas   map[string]escalado.Estado // clave -> ultimo estado conocido
	envuelo map[string]bool            // clave -> hay intencion sin curso
}

// AbrirDiario lee lo que haya y deja el fichero listo para anadir.
func AbrirDiario(ruta string) (*Diario, error) {
	d := &Diario{ruta: ruta, vidas: map[string]escalado.Estado{}, envuelo: map[string]bool{}}
	// #nosec G304 -- la ruta del diario la da el operador con --diario, en su
	// propia maquina. No llega de una peticion y no hay atacante remoto:
	// quien escriba una ruta rara se la escribe a si mismo. Inflarlo seria
	// tan malo como ignorarlo.
	f, err := os.Open(ruta)
	if errors.Is(err, os.ErrNotExist) {
		return d, nil // un diario que no existe todavia es un diario vacio
	}
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDiarioRoto, err)
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	linea := 0
	for sc.Scan() {
		linea++
		if strings.TrimSpace(sc.Text()) == "" {
			continue
		}
		var a Apunte
		if err := json.Unmarshal(sc.Bytes(), &a); err != nil {
			// UNA LINEA ROTA NO SE SALTA. Saltarla convertiria un diario
			// truncado por un corte en un diario "sin esa intencion", y
			// entonces el aviso en vuelo se re-mandaria creyendo que nunca
			// existio, o peor, se daria por hecho.
			return nil, fmt.Errorf("%w: la linea %d de %s no es un apunte legible",
				ErrDiarioRoto, linea, filepath.Base(ruta))
		}
		d.aplicar(a)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDiarioRoto, err)
	}
	return d, nil
}

func (d *Diario) aplicar(a Apunte) {
	switch a.Tipo {
	case Intencion:
		d.envuelo[a.Clave] = true
	case Curso, Suprimido:
		d.envuelo[a.Clave] = false
		d.vidas[a.Clave] = a.Estado
	}
}

// Anotar anade un apunte y lo BAJA A DISCO antes de volver.
func (d *Diario) Anotar(a Apunte) error {
	// #nosec G304 -- la misma ruta del operador, ya validada al abrir.
	f, err := os.OpenFile(d.ruta, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	b, err := json.Marshal(a)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	d.aplicar(a)
	return nil
}

// Cerrado dice si esa clave ya tiene un desenlace que no hay que repetir.
//
// Un FALLIDO no cierra: el lazo esta para reintentarlo en la vuelta siguiente,
// que es media razon de que exista un lazo.
func (d *Diario) Cerrado(clave string) bool {
	e, hay := d.vidas[clave]
	if !hay {
		return false
	}
	return e != escalado.Fallido
}

// EnVuelo dice si hay una intencion sin curso: el lazo murio entre el envio y
// el registro, o esta ahora mismo en medio.
func (d *Diario) EnVuelo(clave string) bool { return d.envuelo[clave] }

// ---------------------------------------------------------------------------
// El lazo
// ---------------------------------------------------------------------------

// Trabajo es un vencimiento concreto que hay que escalar. Lo compone quien
// llama, a partir del corpus y del alcance.
type Trabajo struct {
	Obligacion corpus.Obligacion
	Hito       string
	Vence      time.Time
	Regimen    ventana.Regimen
}

// Lazo tiene todo lo que hace falta para una vuelta.
type Lazo struct {
	Diario    *Diario
	Canal     puertos.Notificacion
	Figuras   escalado.Asignacion
	Silencios []escalado.Silencio
	// Enlace construye la direccion de la instancia LOCAL. Lo da quien llama:
	// el nucleo no sabe donde vive la instancia del cliente.
	Enlace func(obligacion, hito string) string
	// Destino traduce una persona al canal y direccion por los que se le
	// escribe. Vacio el canal, el aviso sale como sin destinatario.
	Destino func(persona string) (canal, direccion string)
	// MarcarCiclo apunta que el planificador ha corrido. Lo cablea quien llama
	// a `latido.Ciclo`, y entra como funcion para que este paquete no dependa
	// del latido: la vida del lazo ALIMENTA al latido, no lo reimplementa.
	//
	// La regla de las 24 horas sigue viviendo en `nucleo/pantalla`. Copiar
	// aqui un "si han pasado 24 horas" crearia dos reglas que un dia dirian
	// cosas distintas, que es el fallo que esa pieza documenta.
	MarcarCiclo func(ahora time.Time) error
}

// Resumen es lo que sale de una vuelta.
type Resumen struct {
	Vuelta       time.Time
	Planificados int
	Cuenta       map[escalado.Estado]int
	// Reintentos son los avisos que se re-mandaron por haber quedado en vuelo
	// tras un corte. Se cuentan aparte porque es EL precio declarado de la
	// entrega al-menos-una-vez, y un precio que no se mide no se puede juzgar.
	Reintentos int
	// YaCerrados son los que no se tocaron porque su ciclo ya tenia desenlace.
	// Es lo que impide la tormenta de un reinicio, asi que se cuenta.
	YaCerrados int
}

// Suma comprueba la ley de conservacion de la vuelta: todo aviso planificado
// cae en exactamente un cubo.
func (r Resumen) Suma() int {
	n := r.YaCerrados
	for _, v := range r.Cuenta {
		n += v
	}
	return n
}

// Vuelta hace una pasada: planifica, manda lo que toca y anota.
//
// El orden de cada aviso es SIEMPRE el mismo y no es negociable:
//
//  1. anotar la intencion, con fsync
//  2. enviar
//  3. anotar el curso
//
// Si el proceso muere entre 1 y 3, la vuelta siguiente ve una intencion sin
// curso y REENVIA. Eso es al-menos-una-vez, cuesta como mucho un duplicado por
// corte, y se cuenta en Resumen.Reintentos.
func (l *Lazo) Vuelta(ahora time.Time, trabajos []Trabajo) (Resumen, error) {
	r := Resumen{Vuelta: ahora, Cuenta: map[escalado.Estado]int{}}
	if l.Diario == nil {
		return r, ErrSinDiario
	}
	if l.Canal == nil {
		return r, ErrSinCanal
	}
	for _, t := range trabajos {
		pasos, err := escalado.Planificar(t.Obligacion, t.Hito, t.Vence, t.Regimen,
			l.Figuras, l.Silencios, l.Enlace)
		if err != nil {
			return r, fmt.Errorf("planificando %s: %w", t.Obligacion.ID, err)
		}
		for _, p := range pasos {
			r.Planificados++
			c := Clave{Obligacion: t.Obligacion.ID, Hito: t.Hito, Vence: t.Vence, Nivel: p.Nivel}
			if err := l.unPaso(ahora, c, p, &r); err != nil {
				return r, err
			}
		}
	}
	// LA MARCA DEL CICLO SE ESCRIBE AUNQUE ALGUN ENVIO HAYA FALLADO, y es la
	// misma decision que toma el latido con su pulso: el canal hacia fuera no
	// puede llevarse por delante la constancia de que el planificador corrio.
	// Si un canal roto borrara esa marca, el operador veria "el planificador
	// esta muerto" cuando lo que pasa es que el correo no sale, y son dos
	// problemas con dos arreglos distintos.
	if l.MarcarCiclo != nil {
		if err := l.MarcarCiclo(ahora); err != nil {
			return r, fmt.Errorf("la vuelta termino y no se pudo apuntar el ciclo, asi que "+
				"el latido dira que el planificador esta parado sin estarlo: %w", err)
		}
	}
	return r, nil
}

func (l *Lazo) unPaso(ahora time.Time, c Clave, p escalado.Paso, r *Resumen) error {
	clave := c.String()
	if l.Diario.Cerrado(clave) {
		r.YaCerrados++
		return nil
	}
	// Lo que el plan decidio NO mandar se anota igual: un aviso que no sale por
	// decision tiene que dejar constancia de la decision, o "no se aviso" y "se
	// decidio no avisar" se leen igual.
	if p.Aviso == nil {
		r.Cuenta[p.Estado]++
		return l.Diario.Anotar(Apunte{Clave: clave, Tipo: Suprimido, Instante: ahora,
			Estado: p.Estado, Motivo: p.Motivo})
	}
	canal, direccion := "", ""
	if l.Destino != nil {
		canal, direccion = l.Destino(p.Persona)
	}
	if strings.TrimSpace(canal) == "" || strings.TrimSpace(direccion) == "" {
		r.Cuenta[escalado.SinDestinatario]++
		return l.Diario.Anotar(Apunte{Clave: clave, Tipo: Suprimido, Instante: ahora,
			Estado: escalado.SinDestinatario,
			Motivo: fmt.Sprintf("la figura %s la ocupa %q y no hay canal por el que "+
				"escribirle. Esto NO dice que el aviso fallara: dice que falta el dato "+
				"de por donde mandarlo", p.Figura, p.Persona)})
	}

	if l.Diario.EnVuelo(clave) {
		// Habia una intencion sin curso: el lazo murio despues de anotar y no
		// se sabe si llego. Se reenvia, que es la eleccion declarada, y se
		// cuenta el precio.
		r.Reintentos++
	}
	// 1. LA INTENCION, ANTES DE TOCAR EL CANAL.
	if err := l.Diario.Anotar(Apunte{Clave: clave, Tipo: Intencion, Instante: ahora,
		Canal: canal}); err != nil {
		return fmt.Errorf("no puedo anotar la intencion de %s: %w", clave, err)
	}
	// 2. EL ENVIO.
	errEnvio := l.Canal.Enviar(canal, direccion, p.Aviso.Asunto(), p.Aviso.Cuerpo())
	// 3. EL CURSO, pase lo que pase.
	estado, motivo := escalado.Enviado, ""
	if errEnvio != nil {
		estado, motivo = escalado.Fallido, errEnvio.Error()
	}
	r.Cuenta[estado]++
	return l.Diario.Anotar(Apunte{Clave: clave, Tipo: Curso, Instante: ahora,
		Estado: estado, Canal: canal, Motivo: motivo})
}

// Fallidos lista las claves que quedaron fallidas, ordenadas, para que la
// pantalla las ensene. UN FALLO QUE SOLO SE ANOTA ES UN ESCALADO QUE MUERE EN
// SILENCIO: el diario existe para poder ensenarlo, no para sustituirlo.
func (d *Diario) Fallidos() []string {
	var out []string
	for k, e := range d.vidas {
		if e == escalado.Fallido {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
