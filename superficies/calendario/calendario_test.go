package calendario

import (
	"bytes"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/marcosmatalab/plazum/nucleo/pantalla"
	"github.com/marcosmatalab/plazum/nucleo/ventana"
)

var instante = time.Date(2026, 8, 27, 9, 0, 0, 0, time.UTC)

func fechaDePrueba(titulo string) pantalla.Fecha {
	return pantalla.Fecha{
		Vence:      time.Date(2026, 10, 31, 23, 59, 59, 0, time.UTC),
		Marco:      "urn:demo:x",
		Obligacion: "demo.una",
		Titulo:     titulo,
		Articulo:   "9.2.2",
		Cita:       "cita de prueba",
		Hito:       "auditoria#1",
		Estado:     ventana.Determinado,
		Regla:      "base 2025-10-31 ; +12 mes(es) ; fin de dia",
	}
}

func calDe(fs ...pantalla.Fecha) pantalla.Calendario {
	return pantalla.Calendario{
		Desde: instante, Hasta: instante.AddDate(1, 0, 0),
		Meses: []pantalla.Mes{{Ano: 2026, Mes: time.October, Clave: "ui.mes.10", Fechas: fs}},
	}
}

func escribir(t *testing.T, cal pantalla.Calendario) string {
	t.Helper()
	var b bytes.Buffer
	if err := Escribir(&b, cal, Opciones{Ahora: instante, Organizacion: "ACME", Dominio: "prueba"}); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// --- un parser minimo, ESCRITO A MANO EN EL TEST -----------------------------
//
// No comparte una sola linea con el generador, y eso es la mitad del valor: un
// test que desplegara con la misma funcion que plego estaria comprobando que
// f(g(x)) = x para su propia f y su propia g, o sea nada. Este desplegado
// implementa la regla del RFC leyendola, no copiandola.

func desplegarSegunElRFC(t *testing.T, s string) []string {
	t.Helper()
	if !strings.HasSuffix(s, "\r\n") {
		t.Fatal("el fichero no termina en CRLF")
	}
	fisicas := strings.Split(strings.TrimSuffix(s, "\r\n"), "\r\n")
	var out []string
	for _, l := range fisicas {
		if strings.HasPrefix(l, " ") || strings.HasPrefix(l, "\t") {
			if len(out) == 0 {
				t.Fatal("el fichero empieza con una linea de continuacion")
			}
			out[len(out)-1] += l[1:]
			continue
		}
		out = append(out, l)
	}
	return out
}

func TestElFicheroCumpleLoQueExigeElRFC5545(t *testing.T) {
	s := escribir(t, calDe(fechaDePrueba("Auditoria interna")))

	t.Run("todas las lineas terminan en CRLF y ninguna lleva un LF suelto", func(t *testing.T) {
		if strings.Contains(strings.ReplaceAll(s, "\r\n", ""), "\n") {
			t.Error("hay un LF sin su CR delante. Outlook rechaza el fichero entero y no dice por que")
		}
	})

	t.Run("ninguna linea fisica pasa de 75 octetos", func(t *testing.T) {
		for i, l := range strings.Split(strings.TrimSuffix(s, "\r\n"), "\r\n") {
			if len(l) > LimiteDeOctetos {
				t.Errorf("linea %d de %d octetos (limite %d): %q", i+1, len(l), LimiteDeOctetos, l)
			}
		}
	})

	t.Run("el contenido es UTF-8 valido despues de plegar", func(t *testing.T) {
		if !utf8.ValidString(s) {
			t.Error("el fichero no es UTF-8 valido: el plegado ha partido un caracter por la mitad")
		}
	})

	t.Run("la estructura y las propiedades obligatorias estan", func(t *testing.T) {
		l := desplegarSegunElRFC(t, s)
		if l[0] != "BEGIN:VCALENDAR" || l[len(l)-1] != "END:VCALENDAR" {
			t.Fatalf("el fichero no abre y cierra el VCALENDAR: %q ... %q", l[0], l[len(l)-1])
		}
		for _, quiero := range []string{"VERSION:2.0", "PRODID:", "BEGIN:VEVENT", "UID:",
			"DTSTAMP:", "DTSTART:", "SUMMARY:", "END:VEVENT"} {
			hay := false
			for _, x := range l {
				if strings.HasPrefix(x, quiero) {
					hay = true
					break
				}
			}
			if !hay {
				t.Errorf("falta la propiedad obligatoria %q", quiero)
			}
		}
	})

	t.Run("el vencimiento va como instante UTC, no como dia completo", func(t *testing.T) {
		if !strings.Contains(s, "DTSTART:20261031T235959Z") {
			t.Error("DTSTART no es el instante exacto en UTC. Un plazo que vence a las 23:59:59 " +
				"no es \"ese dia\": pintarlo como evento de dia completo le regala al que lo lee " +
				"un dia entero que no tiene")
		}
		if strings.Contains(s, "VALUE=DATE") {
			t.Error("hay un DTSTART de tipo DATE")
		}
	})
}

// EL ESCAPADO, y el punto y coma importa aqui mas que en ningun otro sitio: la
// derivacion del motor usa " ; " como separador, asi que TODA descripcion lleva
// punto y coma dentro.
func TestElTextoVaEscapadoSegunLaSeccion3311(t *testing.T) {
	f := fechaDePrueba("Titulo con coma, punto y coma; barra \\ y salto")
	f.Cita = "cita ; con ; separadores"
	s := escribir(t, calDe(f))
	l := desplegarSegunElRFC(t, s)

	var resumen string
	for _, x := range l {
		if strings.HasPrefix(x, "SUMMARY:") {
			resumen = x
		}
	}
	if resumen == "" {
		t.Fatal("no hay SUMMARY")
	}
	// La comprobacion se hace CARACTER A CARACTER mirando si va precedido de
	// barra invertida. La version ingenua (buscar ", " con strings.Contains)
	// esta rota y lo estuvo: `\, ` contiene `, ` como subcadena, asi que daba
	// rojo sobre texto correctamente escapado. Es el mismo error que este
	// repositorio ya cometio buscando la palabra "puerta" dentro de un
	// comentario.
	cuerpo := strings.TrimPrefix(resumen, "SUMMARY:")
	for i := 0; i < len(cuerpo); i++ {
		c := cuerpo[i]
		if c != ',' && c != ';' {
			continue
		}
		barras := 0
		for j := i - 1; j >= 0 && cuerpo[j] == '\\'; j-- {
			barras++
		}
		if barras%2 == 0 {
			t.Errorf("el SUMMARY lleva %q sin escapar en la posicion %d: %q",
				string(c), i, resumen)
		}
	}
	if !strings.Contains(resumen, "\\,") || !strings.Contains(resumen, "\\;") {
		t.Errorf("el SUMMARY no lleva los escapes esperados: %q", resumen)
	}
	// La barra invertida se escapa PRIMERO, o se escaparian las que mete el
	// resto del escapado. Si el orden estuviera al reves, aqui habria \\\\,
	if strings.Contains(resumen, "\\\\\\") {
		t.Errorf("la barra invertida se ha escapado despues que el resto: %q", resumen)
	}
}

// EL CASO SE CONSTRUYE A MANO, y hay que decir por que: el corpus real de hoy no
// produce ninguna linea que caiga con un caracter multibyte justo en el octeto
// 75, asi que un test derivado del corpus estaria verde sin vigilar nada. Este
// caso pone una "n" con tilde exactamente ahi.
func TestElPlegadoNoParteUnCaracterPorLaMitad(t *testing.T) {
	for relleno := 60; relleno < 90; relleno++ {
		s := strings.Repeat("a", relleno) + "ñññññññññññññññññññññññññññññ"
		lineas := Plegar(s)
		junto := strings.Join(lineas, "")
		if !utf8.ValidString(junto) {
			t.Fatalf("relleno %d: el plegado ha partido una runa y el resultado ya no es UTF-8",
				relleno)
		}
		for i, l := range lineas {
			if len(l) > LimiteDeOctetos {
				t.Fatalf("relleno %d, linea %d: %d octetos", relleno, i, len(l))
			}
			if !utf8.ValidString(l) {
				t.Fatalf("relleno %d: la linea fisica %d no es UTF-8 valida", relleno, i)
			}
		}
		// Y no se pierde ni se duplica nada: quitando el espacio de
		// continuacion de cada linea menos la primera, se recupera la entrada.
		var rehecho strings.Builder
		for i, l := range lineas {
			if i > 0 {
				l = strings.TrimPrefix(l, " ")
			}
			rehecho.WriteString(l)
		}
		if rehecho.String() != s {
			t.Fatalf("relleno %d: el plegado ha perdido o duplicado bytes", relleno)
		}
	}
}

// EL UID TIENE QUE SER ESTABLE. Si cambia entre ejecuciones, cada importacion
// duplica los eventos en el calendario de quien lo usa, y a la tercera semana
// tiene la agenda llena de basura y desinstala.
func TestElUIDEsEstableEntreEjecucionesYDistingueVencimientos(t *testing.T) {
	f := fechaDePrueba("Auditoria")
	if UIDDe(f) != UIDDe(f) {
		t.Fatal("el UID no es estable ni consigo mismo")
	}
	// Dos ejecuciones enteras dan los mismos bytes.
	if a, b := escribir(t, calDe(f)), escribir(t, calDe(f)); a != b {
		t.Error("dos ejecuciones con el mismo instante dan ficheros distintos")
	}
	// Y distingue lo que hay que distinguir.
	base := UIDDe(f)
	casos := map[string]pantalla.Fecha{}
	otro := f
	otro.Hito = "auditoria#2"
	casos["otro hito"] = otro
	otro2 := f
	otro2.Obligacion = "demo.otra"
	casos["otra obligacion"] = otro2
	otro3 := f
	otro3.Vence = f.Vence.AddDate(0, 0, 1)
	casos["otra fecha"] = otro3
	otro4 := f
	otro4.Marco = "urn:demo:y"
	casos["otro marco"] = otro4
	for nombre, c := range casos {
		if UIDDe(c) == base {
			t.Errorf("%s da el mismo UID: dos eventos distintos se pisarian en el calendario",
				nombre)
		}
	}
	// El titulo NO entra en el UID: corregir una errata de redaccion no puede
	// dejar huerfano el evento que el CISO ya tiene aceptado en su agenda.
	otro5 := f
	otro5.Titulo = "Auditoria interna del sistema de gestion"
	if UIDDe(otro5) != base {
		t.Error("cambiar el titulo cambia el UID. Corregir una errata dejaria un evento " +
			"duplicado en el calendario de todos los que ya lo tenian")
	}
}

// Lo que NO tiene fecha no entra en el .ics, y no es un olvido: un evento de
// calendario sin fecha no existe, y ponerle una inventada seria fabricar un
// plazo. Se queda en la salida de texto, que si sabe decir "obliga y no tiene
// numero".
func TestLoQueNoTieneFechaNoEntraEnElCalendario(t *testing.T) {
	cal := calDe(fechaDePrueba("Con fecha"))
	cal.SinFecha = []pantalla.SinFecha{{
		Marco: "urn:demo:x", Obligacion: "demo.sin", Titulo: "Sin numero",
		Motivo: pantalla.MotivoSinPlazoLegal, Regla: "la norma no fija limite"}}
	s := escribir(t, cal)
	if n := strings.Count(s, "BEGIN:VEVENT"); n != 1 {
		t.Errorf("%d eventos, esperaba 1", n)
	}
	if strings.Contains(s, "Sin numero") {
		t.Error("una obligacion sin fecha ha entrado en el .ics con alguna fecha inventada")
	}
}

// Un calendario vacio sigue siendo un fichero valido, y eso importa: la puerta
// de CI compara bytes, y un fichero vacio que "pasa" es el verde falso mas
// barato que hay. Por eso la puerta cuenta EVENTOS ademas de comparar.
func TestUnCalendarioVacioProduceUnFicheroValidoYSinEventos(t *testing.T) {
	s := escribir(t, pantalla.Calendario{Desde: instante, Hasta: instante.AddDate(1, 0, 0)})
	l := desplegarSegunElRFC(t, s)
	if l[0] != "BEGIN:VCALENDAR" || l[len(l)-1] != "END:VCALENDAR" {
		t.Error("un calendario vacio no es un VCALENDAR valido")
	}
	if strings.Contains(s, "BEGIN:VEVENT") {
		t.Error("hay eventos en un calendario vacio")
	}
}
