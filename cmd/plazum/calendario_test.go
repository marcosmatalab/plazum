package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// El instante va CABLEADO en todos los casos. Un test del calendario que use el
// reloj del sistema es una bomba con la mecha encendida: las fechas se acercan
// solas y el verde caduca un martes cualquiera sin que nadie toque una linea.
const ahoraDelTest = "2026-08-27T09:00:00Z"

func correrCalendario(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	var salida, errores bytes.Buffer
	codigo := cmdCalendario(append([]string{"--ahora=" + ahoraDelTest}, args...), &salida, &errores)
	return salida.String(), errores.String(), codigo
}

// EL CAMINO QUE SE COMPARTE: tres banderas y fechas en pantalla. Es la captura
// de pantalla del producto, asi que si deja de funcionar hay que enterarse.
func TestElArranqueConTresBanderasProduceFechasYLasMarcaTodasComoSupuestas(t *testing.T) {
	out, _, codigo := correrCalendario(t,
		"--corpus=../../paquetes", "--pais=ES", "--sector=servicios-digitales", "--empleados=200")
	if codigo != 0 {
		t.Fatalf("codigo %d\n%s", codigo, out)
	}

	// Un suelo, no un numero exacto: el corpus crece cada semana y clavar la
	// cifra convertiria este test en un test del corpus. Lo que se vigila es
	// que el camino de punta a punta PRODUZCA fechas, que es lo que se rompe.
	const minimo = 3
	n := strings.Count(out, "[supuesto]")
	if n < minimo {
		t.Errorf("solo %d fechas marcadas como supuestas y el suelo son %d.\n"+
			"  Este es el camino que se comparte en una captura de pantalla: si sale vacio, o el\n"+
			"  perfil ha dejado de derivar, o los paquetes han perdido sus reglas de aplicabilidad.\n%s",
			n, minimo, out)
	}

	// Y TODA fila lleva la marca. Una sola sin marcar convierte una conjetura
	// del perfil en una obligacion de la organizacion, que es el fallo caro.
	for _, l := range strings.Split(out, "\n") {
		esFila := strings.HasPrefix(l, "    ") && strings.Contains(l, " 23:59")
		if esFila && !strings.Contains(l, "[supuesto]") {
			t.Errorf("esta fila sale de un perfil y NO lleva [supuesto]: %q", l)
		}
	}

	for _, quiero := range []string{
		"LO QUE ESTE PERFIL SUPONE",              // lo que asume
		"LO QUE NO SUPONE, Y POR TANTO NO VERAS", // y lo que no, que es la mitad util
		"LA CUENTA, ENTERA",                      // la contabilidad honesta
		"no es asesoramiento juridico",           // el descargo
	} {
		if !strings.Contains(out, quiero) {
			t.Errorf("la salida no trae %q", quiero)
		}
	}
}

// EL DETERMINISMO, que es lo que hace comparable una salida. Con el mismo
// instante, dos ejecuciones dan los mismos bytes.
func TestDosEjecucionesConElMismoInstanteDanLoMismo(t *testing.T) {
	a, _, _ := correrCalendario(t, "--corpus=../../paquetes", "--pais=ES",
		"--sector=servicios-digitales", "--empleados=200")
	b, _, _ := correrCalendario(t, "--corpus=../../paquetes", "--pais=ES",
		"--sector=servicios-digitales", "--empleados=200")
	if a != b {
		t.Error("dos ejecuciones con el mismo --ahora dan salidas distintas. Sin orden total, " +
			"las fechas empatadas salen barajadas y cualquier comparacion se vuelve intermitente")
	}
}

// Y con el .ics, lo mismo y ademas con eventos DE VERDAD. Un VCALENDAR vacio es
// deterministicamente vacio, y una puerta que solo compare bytes se queda verde
// para siempre sobre un fichero sin un solo evento.
func TestElICSSaleDeterministaYConEventos(t *testing.T) {
	a, _, codigo := correrCalendario(t, "--corpus=../../paquetes", "--pais=ES",
		"--sector=servicios-digitales", "--empleados=200", "--ics")
	if codigo != 0 {
		t.Fatalf("codigo %d", codigo)
	}
	b, _, _ := correrCalendario(t, "--corpus=../../paquetes", "--pais=ES",
		"--sector=servicios-digitales", "--empleados=200", "--ics")
	if a != b {
		t.Error("dos .ics del mismo instante difieren")
	}
	if n := strings.Count(a, "BEGIN:VEVENT"); n < 3 {
		t.Errorf("el .ics trae %d eventos. Comparar bytes de un fichero vacio es el verde falso "+
			"mas barato que hay: la puerta cuenta eventos ademas de comparar", n)
	}
	if !strings.HasPrefix(a, "BEGIN:VCALENDAR\r\n") {
		t.Error("el .ics no empieza como un VCALENDAR con CRLF")
	}
}

// LA PASADA CONTRA EL COMPRADOR: lo que pasa cuando se teclea mal. Cada error
// dice que hacer, porque quien lo lee acaba de descargar el binario y no tiene a
// quien preguntar.
func TestLosErroresDelCalendarioDicenQueHacer(t *testing.T) {
	casos := []struct {
		nombre string
		args   []string
		quiero string
	}{
		{"sin nada", []string{"--corpus=../../paquetes"},
			"no se de quien es este calendario"},
		{"sector que no existe",
			[]string{"--corpus=../../paquetes", "--pais=ES", "--sector=inventado"},
			"Los que hay:"},
		{"pais sin sector", []string{"--corpus=../../paquetes", "--pais=ES"},
			"necesita --pais Y --sector"},
		{"alcance y banderas a la vez",
			[]string{"--corpus=../../paquetes", "--alcance=x.json", "--pais=ES", "--sector=y"},
			"Elige uno"},
		{"corpus que no esta", []string{"--corpus=no-existe", "--pais=ES", "--sector=x"},
			"no carga"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			_, errores, codigo := correrCalendario(t, c.args...)
			if codigo == 0 {
				t.Fatalf("codigo 0 con %v: no ha fallado y tenia que fallar", c.args)
			}
			if !strings.Contains(errores, c.quiero) {
				t.Errorf("el error no trae %q y por tanto no dice que hacer:\n%s", c.quiero, errores)
			}
		})
	}
}

// --todos-los-relojes NO es lo que te aplica, y la salida tiene que decirlo. Es
// la diferencia entre inspeccionar el corpus y planificar el ano.
func TestTodosLosRelojesAvisaDeQueNoEsLoQueTeAplica(t *testing.T) {
	out, _, codigo := correrCalendario(t, "--corpus=../../paquetes", "--todos-los-relojes")
	if codigo != 0 {
		t.Fatalf("codigo %d\n%s", codigo, out)
	}
	if !strings.Contains(out, "NO es lo que te aplica") {
		t.Error("la salida de --todos-los-relojes no avisa de que no es lo que te aplica.\n" +
			"  Sin ese aviso, una captura de esa pantalla se lee como el calendario de alguien")
	}
	if strings.Contains(out, "[supuesto]") {
		t.Error("--todos-los-relojes marca filas como supuestas, y no lo son: no hay perfil " +
			"detras, hay ausencia de filtro")
	}
}

// El instante entra como dato hasta el fondo. Se comprueba moviendo --ahora un
// ano: el calendario tiene que cambiar.
func TestElInstanteMandaHastaElFondo(t *testing.T) {
	var a, b bytes.Buffer
	base := []string{"--corpus=../../paquetes", "--pais=ES",
		"--sector=servicios-digitales", "--empleados=200"}
	cmdCalendario(append([]string{"--ahora=" + ahoraDelTest}, base...), &a, &bytes.Buffer{})
	dentroDeUnAno := time.Date(2027, 8, 27, 9, 0, 0, 0, time.UTC).Format(time.RFC3339)
	cmdCalendario(append([]string{"--ahora=" + dentroDeUnAno}, base...), &b, &bytes.Buffer{})
	if a.String() == b.String() {
		t.Error("mover --ahora un ano no cambia nada. O el instante no llega hasta el motor, o " +
			"las fechas del perfil no son desplazamientos y el calendario esta congelado")
	}
	if !strings.Contains(b.String(), "2027") {
		t.Error("con --ahora en 2027 la ventana no arranca en 2027")
	}
}

// LA JUNTA ENTRE EL PERFIL Y LOS PAQUETES ESPANOLES, que no la vigilaba nadie.
//
// POR QUE HACE FALTA ADEMAS DEL SUELO DE ARRIBA. El test anterior exige tres
// fechas marcadas como supuestas, y ese suelo lo cubre el ENS el solo: los
// quince relojes de `lopdgdd` y `ley2-2023` pueden desaparecer del camino
// guiado entero sin que se ponga rojo nada. Y desaparecer es facil, porque no
// hace falta tocar el corpus: basta con quitarle un hecho a un fichero de
// perfil. Eso ya paso una vez con este mismo par (los paquetes cargaban, sus 44
// dorados estaban en verde y `plazum calendario` no ensenaba ni uno), asi que
// esto es la puerta de ese fallo, no una precaucion.
//
// LAS DOS DIRECCIONES, y la segunda es la que prueba que el perfil se MIRA:
//
//	llegan   con el perfil del sector publico salen los dos marcos espanoles,
//	         porque el art. 13.1 de la Ley 2/2023 obliga a toda entidad publica
//	         a tener Sistema interno y el art. 37.1.a del Reglamento (UE)
//	         2016/679 le obliga a tener delegado.
//	no llega el art. 65.4 de la Ley Organica 3/2018, que es el plazo de un mes
//	         de quien NO tiene delegado. Si apareciera, el perfil estaria
//	         ensenando a la vez el deber de quien tiene delegado y el de quien
//	         no lo tiene, que son excluyentes.
func TestElPerfilDelSectorPublicoLlegaALosDosMarcosEspanoles(t *testing.T) {
	out, _, codigo := correrCalendario(t,
		"--corpus=../../paquetes", "--pais=ES", "--sector=sector-publico", "--empleados=400")
	if codigo != 0 {
		t.Fatalf("codigo %d\n%s", codigo, out)
	}

	for _, c := range []struct{ urn, porQue string }{
		{"urn:es:l:2023:2", "el art. 13.1 de la Ley 2/2023 obliga a TODAS las entidades del " +
			"sector publico a disponer de un Sistema interno de informacion, sin umbral de " +
			"empleados, asi que el perfil afirma canal_de_denuncias_obligatorio"},
		{"urn:es:lo:2018:3", "el art. 37.1.a del Reglamento (UE) 2016/679 obliga a designar " +
			"delegado cuando el tratamiento lo lleva a cabo una autoridad u organismo publico, " +
			"asi que el perfil afirma designado(delegado_de_proteccion_de_datos_designado)"},
	} {
		if !strings.Contains(out, c.urn) {
			t.Errorf("el camino guiado del sector publico no llega a %s.\n"+
				"  Deberia: %s.\n"+
				"  O el perfil ha perdido su hecho, o el paquete ha perdido su regla de "+
				"aplicabilidad. Las dos formas dejan quince relojes invisibles con el corpus "+
				"cargando y sus dorados en verde.\n%s", c.urn, c.porQue, out)
		}
	}

	// La direccion que descarta, y la lecion de como escribirla: la primera
	// version buscaba "art. 65.4" a secas y NACIO ROJA contra la prosa del
	// propio perfil, que menciona ese articulo para explicar que tener delegado
	// lo APAGA. Acusar a la prosa que habla de lo que vigila es el fallo tipico
	// de esta clase de guarda, asi que se busca la FILA del calendario entera,
	// con el urn del paquete delante, que es lo unico que significa "el motor ha
	// emitido este vencimiento".
	const filaDel65_4 = "urn:es:lo:2018:3  art. 65.4"
	if strings.Contains(out, filaDel65_4) {
		t.Error("sale el art. 65.4 de la Ley Organica 3/2018 en un perfil que afirma tener " +
			"delegado de proteccion de datos designado. Ese plazo de un mes es del responsable " +
			"o encargado que NO ha designado delegado NI esta adherido a mecanismos de " +
			"resolucion extrajudicial de conflictos; con delegado, la Agencia remite al " +
			"delegado y el plazo aplicable es el mes del art. 37.2. Ensenar los dos a la vez " +
			"le pone al obligado dos fechas excluyentes para la misma reclamacion")
	}
}
