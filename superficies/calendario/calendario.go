// Package calendario escribe un calendario iCalendar (RFC 5545) con los
// vencimientos del corpus, para llevarselos a Outlook, a Google Calendar o a
// Apple Calendar.
//
// POR QUE A MANO. Este proyecto tiene cero dependencias externas y las va a
// seguir teniendo (invariante 5). iCalendar es un formato de texto con cuatro
// reglas incomodas y ninguna dificil, asi que se escribe aqui, con sus tests.
//
// LAS CUATRO REGLAS INCOMODAS, cada una con lo que pasa si se ignora:
//
//	CRLF        toda linea termina en \r\n (RFC 5545, 3.1). Con \n suelto,
//	            Outlook rechaza el fichero entero sin decir por que.
//	plegado     ninguna linea pasa de 75 OCTETOS; lo que sobra continua en la
//	            siguiente empezando por un espacio (3.1). Se cuenta en octetos,
//	            no en caracteres, y hay que cortar en frontera de runa: partir un
//	            caracter multibyte produce un fichero que no es UTF-8 valido.
//	escapado    en un valor de texto, la barra invertida, el punto y coma y la
//	            coma van escapados, y el salto de linea se escribe \n (3.3.11).
//	            El punto y coma importa aqui mas que en ningun otro sitio: la
//	            derivacion del motor usa " ; " como separador, asi que TODA
//	            descripcion lleva punto y coma dentro.
//	UID estable dos exportaciones del mismo vencimiento tienen que dar el MISMO
//	            UID, o cada importacion duplica los eventos en el calendario de
//	            quien lo usa. Se deriva por hash de la identidad del vencimiento,
//	            nunca de un aleatorio ni del reloj.
package calendario

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"plazum/nucleo/pantalla"
)

// LimiteDeOctetos es el maximo de una linea fisica, contenido incluido pero sin
// el CRLF. Lo fija el RFC 5545, seccion 3.1.
const LimiteDeOctetos = 75

// Opciones es lo que el que llama tiene que decidir. No hay valores por defecto
// magicos: DTSTAMP exige un instante y este paquete NO llama a time.Now().
type Opciones struct {
	// Ahora es el instante del sello DTSTAMP. Entra como dato para que dos
	// ejecuciones con el mismo instante den bytes identicos, que es lo unico
	// que hace comprobable un fichero generado.
	Ahora time.Time
	// Organizacion sale en el nombre del calendario.
	Organizacion string
	// Dominio es el sufijo del UID. Solo tiene que ser estable, no resoluble.
	Dominio string
}

// Escribir vuelca el calendario entero.
//
// Lo que NO escribe, y es una decision: los vencimientos SIN FECHA. Un evento de
// calendario sin fecha no existe, y meterlos con una fecha inventada (hoy, o el
// 1 de enero) seria fabricar un plazo. Se quedan en la salida de texto de
// `plazum calendario`, que si sabe decir "obliga y no tiene numero".
func Escribir(w io.Writer, cal pantalla.Calendario, o Opciones) error {
	if o.Dominio == "" {
		o.Dominio = "plazum.invalid"
	}
	var b strings.Builder
	linea(&b, "BEGIN:VCALENDAR")
	linea(&b, "VERSION:2.0")
	linea(&b, "PRODID:-//plazum//calendario de obligaciones//ES")
	linea(&b, "CALSCALE:GREGORIAN")
	linea(&b, "METHOD:PUBLISH")
	linea(&b, "X-WR-CALNAME:"+escapar("plazum: "+o.Organizacion))

	sello := o.Ahora.UTC().Format("20060102T150405") + "Z"
	for _, m := range cal.Meses {
		for _, f := range m.Fechas {
			evento(&b, f, sello, o.Dominio)
		}
	}
	// LAS TRANSICIONES TAMBIEN VIAJAN, y no viajaban. El `.ics` solo recorria
	// cal.Meses, o sea solo los VENCIMIENTOS: el dia que una norma empieza a
	// obligarte no llegaba al Outlook de nadie. En el perfil del fabricante de
	// software eso significaba que la fila con mas actualidad de todo el
	// producto (el art. 14 del CRA arrancando el 11-09-2026) salia por pantalla
	// y se quedaba en la pantalla.
	for _, e := range cal.Estrenos {
		transicion(&b, transicionICS{
			Dia: e.Desde, Marco: e.Marco, Obligacion: e.Obligacion,
			Titulo: e.Titulo, Articulo: e.Articulo, Cita: e.Cita,
			Supuesta: e.Supuesta, Clase: "estreno",
			Prefijo: "Empieza a obligarte: ",
			Nota: "Desde esta fecha esta obligacion te alcanza. Hoy todavia no obliga: " +
				"no hay nada que entregar y tampoco nada que hayas incumplido.",
		}, sello, o.Dominio)
	}
	for _, c := range cal.Ceses {
		transicion(&b, transicionICS{
			Dia: c.Hasta, Marco: c.Marco, Obligacion: c.Obligacion,
			Titulo: c.Titulo, Articulo: c.Articulo, Cita: c.Cita,
			Supuesta: c.Supuesta, Clase: "cese",
			Prefijo: "Deja de obligarte: ",
			Nota: "Esta obligacion te alcanza HASTA esta fecha, incluida. Despues puedes " +
				"dejar de hacerla; conviene guardar la evidencia de lo hecho hasta entonces.",
		}, sello, o.Dominio)
	}
	linea(&b, "END:VCALENDAR")
	_, err := io.WriteString(w, b.String())
	return err
}

func evento(b *strings.Builder, f pantalla.Fecha, sello, dominio string) {
	linea(b, "BEGIN:VEVENT")
	linea(b, "UID:"+UIDDe(f)+"@"+dominio)
	linea(b, "DTSTAMP:"+sello)

	// EL VENCIMIENTO ES UN INSTANTE, NO UN DIA, y por eso va como DATE-TIME en
	// UTC y no como evento de dia completo. Un plazo que vence a las 23:59:59
	// no es "ese dia": es ese segundo, y un evento de dia completo lo pinta a
	// las 00:00, dandole al que lo lee un dia entero que no tiene.
	inicio := f.Vence.UTC().Format("20060102T150405") + "Z"
	linea(b, "DTSTART:"+inicio)
	linea(b, "DTEND:"+inicio)
	linea(b, "TRANSP:TRANSPARENT")

	titulo := f.Titulo
	if f.Supuesta {
		titulo = "[supuesto] " + titulo
	}
	linea(b, "SUMMARY:"+escapar(titulo))
	linea(b, "DESCRIPTION:"+escapar(descripcion(f)))
	linea(b, "CATEGORIES:"+escapar(f.Marco))
	linea(b, "END:VEVENT")
}

// transicionICS es un estreno o un cese aplanado, para escribir los dos con el
// mismo codigo sin darle a este paquete una interfaz que solo usaria una vez.
type transicionICS struct {
	Dia        time.Time
	Marco      string
	Obligacion string
	Titulo     string
	Articulo   string
	Cita       string
	Supuesta   bool
	// Clase entra en el UID para que un estreno y un cese de la MISMA
	// obligacion en la MISMA fecha no colisionen. Puede pasar: una obligacion
	// con vigencia de un solo dia dentro de la ventana.
	Clase   string
	Prefijo string
	Nota    string
}

// transicion escribe un estreno o un cese como evento de DIA COMPLETO.
//
// Y AQUI LA REGLA ES LA CONTRARIA QUE EN UN VENCIMIENTO, que es lo que hay que
// mirar dos veces. Un vencimiento va como DATE-TIME porque vence a un segundo
// concreto y pintarlo de dia completo regala un dia entero. Una transicion va
// como DATE porque NO es un instante: la norma empieza a aplicarse un dia, no a
// una hora, y ponerle las 00:00:00Z en el calendario de alguien la convierte en
// una cita a medianoche y, con zona horaria por medio, la mueve al dia
// anterior. Es el mismo error por los dos lados opuestos.
func transicion(b *strings.Builder, t transicionICS, sello, dominio string) {
	linea(b, "BEGIN:VEVENT")
	linea(b, "UID:"+uidDeTransicion(t)+"@"+dominio)
	linea(b, "DTSTAMP:"+sello)

	dia := t.Dia.UTC().Format("20060102")
	linea(b, "DTSTART;VALUE=DATE:"+dia)
	// DTEND de un evento de dia completo es EXCLUSIVO (RFC 5545, 3.8.2.2): para
	// que ocupe un solo dia hay que poner el siguiente. Con el mismo dia,
	// algunos clientes lo pintan de duracion cero y otros lo esconden.
	linea(b, "DTEND;VALUE=DATE:"+t.Dia.UTC().AddDate(0, 0, 1).Format("20060102"))
	linea(b, "TRANSP:TRANSPARENT")

	titulo := t.Prefijo + t.Titulo
	if t.Supuesta {
		titulo = "[supuesto] " + titulo
	}
	linea(b, "SUMMARY:"+escapar(titulo))

	var p []string
	p = append(p, "Marco: "+t.Marco)
	if t.Articulo != "" {
		p = append(p, "Articulo: "+t.Articulo)
	}
	p = append(p, t.Nota)
	if t.Cita != "" {
		p = append(p, "Cita: "+t.Cita)
	}
	if t.Supuesta {
		p = append(p, "SUPUESTO: sale de un perfil de arranque, no de tus respuestas.")
	}
	linea(b, "DESCRIPTION:"+escapar(strings.Join(p, " ; ")))
	linea(b, "CATEGORIES:"+escapar(t.Marco))
	linea(b, "END:VEVENT")
}

// uidDeTransicion deriva el UID igual que UIDDe: de la identidad, nunca de un
// aleatorio ni del reloj, para que reimportar no duplique. La CLASE entra en el
// hash porque un estreno y un cese son eventos distintos aunque compartan
// obligacion y dia.
func uidDeTransicion(t transicionICS) string {
	h := sha256.Sum256([]byte(strings.Join(
		[]string{t.Clase, t.Marco, t.Obligacion, t.Dia.UTC().Format(time.RFC3339)}, "\x1f")))
	return hex.EncodeToString(h[:16])
}

func descripcion(f pantalla.Fecha) string {
	var p []string
	p = append(p, "Marco: "+f.Marco)
	if f.Articulo != "" {
		p = append(p, "Articulo: "+f.Articulo)
	}
	p = append(p, "Hito: "+f.Hito)
	if f.Cita != "" {
		p = append(p, "Cita: "+f.Cita)
	}
	if f.Regla != "" {
		p = append(p, "De donde sale la fecha: "+f.Regla)
	}
	for _, d := range f.Divergencias {
		p = append(p, fmt.Sprintf("Otra lectura (%s): %s. %s",
			d.Lectura, d.Vence.UTC().Format("2006-01-02 15:04:05Z"), d.Cita))
	}
	if f.Aviso != "" {
		p = append(p, "Aviso: "+f.Aviso)
	}
	if f.Supuesta {
		p = append(p, "SUPUESTO: esta obligacion sale de un perfil de arranque, no de las "+
			"respuestas de la organizacion.")
	}
	p = append(p, "Esto no es asesoramiento juridico.")
	return strings.Join(p, "\n")
}

// UIDDe deriva un identificador ESTABLE del vencimiento.
//
// Estable entre ejecuciones (si no, cada importacion duplica los eventos) y
// unico entre vencimientos distintos. Se calcula sobre la identidad del
// vencimiento y NADA MAS: ni el reloj, ni un aleatorio, ni la version del
// paquete. Lo ultimo es deliberado: con la version dentro, actualizar un paquete
// de corpus dejaria huerfano todo lo que el CISO ya tiene en su Outlook.
//
// Los tres campos van separados por un byte que no puede aparecer dentro de
// ninguno (\x1f), porque concatenar sin separador hace que ("ab","c") y
// ("a","bc") den el mismo identificador.
func UIDDe(f pantalla.Fecha) string {
	h := sha256.Sum256([]byte(strings.Join(
		[]string{f.Marco, f.Obligacion, f.Hito, f.Vence.UTC().Format(time.RFC3339)}, "\x1f")))
	return hex.EncodeToString(h[:16])
}

// escapar aplica la seccion 3.3.11. El orden importa: la barra invertida
// PRIMERO, o se escaparian las barras que acaba de meter el resto.
func escapar(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\r\n", "\\n")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\n")
	return s
}

// linea escribe una linea de contenido ya plegada, con su CRLF.
func linea(b *strings.Builder, s string) {
	for _, l := range Plegar(s) {
		b.WriteString(l)
		b.WriteString("\r\n")
	}
}

// Plegar parte una linea de contenido en lineas fisicas de como mucho
// LimiteDeOctetos octetos, con un espacio al principio de cada continuacion.
//
// EL CORTE VA EN FRONTERA DE RUNA. Cortar a ciegas en el octeto 75 parte por la
// mitad cualquier caracter acentuado que caiga ahi, y el resultado ya no es
// UTF-8 valido: el fichero deja de importar y el mensaje del cliente de
// calendario no dice donde. El corpus real de hoy no llega a producir esa linea
// (se midio), asi que el caso del test se CONSTRUYE a mano: un test derivado del
// corpus estaria verde sin vigilar nada.
func Plegar(s string) []string {
	if len(s) <= LimiteDeOctetos {
		return []string{s}
	}
	var out []string
	// La primera linea admite LimiteDeOctetos; las siguientes, uno menos,
	// porque el espacio de continuacion tambien cuenta.
	limite := LimiteDeOctetos
	for len(s) > limite {
		corte := limite
		for corte > 0 && !utf8.RuneStart(s[corte]) {
			corte--
		}
		if corte == 0 {
			// Una runa mas larga que el limite no existe en UTF-8 (cuatro
			// octetos como mucho), pero si la entrada no es UTF-8 valido el
			// bucle de arriba puede llegar aqui. Se corta en el limite en vez
			// de girar para siempre.
			corte = limite
		}
		out = append(out, s[:corte])
		s = s[corte:]
		limite = LimiteDeOctetos - 1
		s = " " + s
	}
	out = append(out, s)
	return out
}
