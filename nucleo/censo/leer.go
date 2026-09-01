package censo

import (
	"encoding/csv"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ErrCodificacion es el fichero que no se puede leer como texto de una sola
// pasada. Hoy solo lo dispara UTF-16, y a proposito: ver aTexto.
var ErrCodificacion = errors.New("censo: el fichero no esta en una codificacion que se pueda leer aqui")

// aTexto normaliza los bytes a texto UTF-8 valido, y DICE lo que ha hecho.
//
// Las dos correcciones que hace no son cosmeticas, las dos se midieron:
//
//   - EL BOM. Excel escribe EF BB BF delante, y csv se lo mete DENTRO del nombre
//     de la primera columna. La cabecera sale como "\ufeffusuario", la busqueda
//     de "usuario" falla, y el error que ve quien sube el fichero es "falta la
//     columna usuario" sobre un fichero que la tiene. Un error que miente es
//     peor que un error feo.
//
//   - cp1252. Si los bytes no son UTF-8 valido se decodifican como cp1252, que
//     es lo que sale de Excel en español. La alternativa medida es dejarlos
//     pasar, y csv NO DA NINGUN ERROR: la cadena invalida viaja hasta la
//     pantalla, y si el acento estaba en el identificador y no en el rotulo, lo
//     corrompido en silencio es la identidad por la que se empareja todo.
//
// La heuristica "si es UTF-8 valido, es UTF-8" es la estandar y falla solo con
// ficheros cp1252 cuyos bytes altos formen por casualidad secuencias UTF-8
// validas ("Ã©" en vez de "é"). Cuando pasa, se ve a simple vista en el rotulo,
// no se corrompe la identidad en silencio, y es el unico caso que este paquete
// no distingue.
func aTexto(datos []byte) (string, Notas) {
	var n Notas
	if len(datos) >= 3 && datos[0] == 0xEF && datos[1] == 0xBB && datos[2] == 0xBF {
		datos = datos[3:]
		n.QuitadoElBOM = true
	}
	if utf8.Valid(datos) {
		n.Codificacion = "utf-8"
		return string(datos), n
	}
	n.Codificacion = "cp1252 (los bytes no eran UTF-8 valido)"
	var b strings.Builder
	b.Grow(len(datos) * 2)
	for _, x := range datos {
		if r, hay := altosDeCP1252[x]; hay {
			b.WriteRune(r)
			continue
		}
		// 0x00-0x7F y 0xA0-0xFF: en cp1252 valen su propio punto de codigo, que
		// es lo mismo que en ISO-8859-1.
		b.WriteRune(rune(x))
	}
	return b.String(), n
}

// altosDeCP1252 es el unico tramo donde cp1252 y latin-1 se separan (0x80-0x9F).
// Es la tabla entera, escrita a mano: son 27 entradas y traer una dependencia
// para esto seria pagar un modulo por veintisiete constantes.
var altosDeCP1252 = map[byte]rune{
	0x80: '€', 0x82: '‚', 0x83: 'ƒ', 0x84: '„',
	0x85: '…', 0x86: '†', 0x87: '‡', 0x88: 'ˆ',
	0x89: '‰', 0x8A: 'Š', 0x8B: '‹', 0x8C: 'Œ',
	0x8E: 'Ž', 0x91: '‘', 0x92: '’', 0x93: '“',
	0x94: '”', 0x95: '•', 0x96: '–', 0x97: '—',
	0x98: '˜', 0x99: '™', 0x9A: 'š', 0x9B: '›',
	0x9C: 'œ', 0x9E: 'ž', 0x9F: 'Ÿ',
	// 0x81, 0x8D, 0x8F, 0x90 y 0x9D no existen en cp1252. Se dejan pasar como su
	// propio punto de codigo, que es lo que hacen los navegadores, en vez de
	// convertirlos en el caracter de reemplazo: asi el byte sigue siendo
	// distinguible si alguien va a mirar el fichero.
}

// esUTF16 detecta los dos BOM de UTF-16.
//
// SE RECHAZA EN VEZ DE DECODIFICARSE, y es una decision, no una omision: el
// "Texto Unicode" de Excel guarda UTF-16 **y ademas separa por tabuladores**, y
// decodificar una cosa mientras se adivina la otra es como se acumulan dos
// suposiciones y sale un censo plausible. Rechazar con instrucciones concretas
// cuesta a quien lo sube treinta segundos de "guardar como CSV UTF-8"; adivinar
// mal cuesta una revision de accesos hecha sobre otra cosa.
func esUTF16(datos []byte) bool {
	if len(datos) < 2 {
		return false
	}
	return (datos[0] == 0xFF && datos[1] == 0xFE) || (datos[0] == 0xFE && datos[1] == 0xFF)
}

// contarLineas cuenta las lineas fisicas del texto, SIN saber nada de comillas.
//
// ESTA CEGUERA ES EL DISENO, no un descuido, y es lo unico que hace que la ley
// de conservacion sirva de algo. Si este contador respetara las comillas como el
// parser, se tragaria exactamente las mismas lineas que el parser se traga y
// cuadraria siempre: seria una guarda que confirma al vigilado. Es la disciplina
// de los metodos ciegos entre si.
//
// La consecuencia, dicha en voz alta: un campo con un salto de linea DENTRO,
// que es CSV legitimo, hace que el recuento no cuadre y que la instantanea no
// cargue. Es un falso positivo conocido y se prefiere: cuesta mirar el fichero,
// mientras que el falso negativo cuesta certificar una revision completa sobre
// un censo al que le faltan personas.
// LAS LINEAS EN BLANCO NO SE CUENTAN, y no es una comodidad: es que el lector de
// CSV se las salta en silencio. Medido: "u;p\nana;admin\n\n" da dos registros, no
// tres, y con CRLF igual. Si este contador las contara, un fichero acabado en
// una linea en blanco (o sea, casi cualquiera que haya pasado por un editor)
// daria descuadre y no cargaria. Una guarda que salta con lo normal se acaba
// quitando, y entonces deja de guardar lo raro, que era su trabajo.
//
// Una linea con UN ESPACIO si es un registro para el lector (medido: sale como
// un campo con un espacio), asi que aqui tambien cuenta. Las dos formas del
// vacio otra vez, y las dos tienen que tratarse como las trata el lector o el
// recuento se separa del suyo.
func contarLineas(texto string) int {
	if texto == "" {
		return 0
	}
	n := 0
	for _, l := range strings.Split(texto, "\n") {
		if strings.TrimSuffix(l, "\r") != "" {
			n++
		}
	}
	return n
}

// separadoresCandidatos, en el orden en el que se prueban. El punto y coma va
// primero porque es lo que escribe Excel en español: en una configuracion
// regional donde la coma es el separador decimal, el CSV se separa por ';'.
var separadoresCandidatos = []rune{';', ',', '\t', '|'}

// deducirSeparador elige el que parte la cabecera en mas columnas.
//
// Se mide sobre la CABECERA y no sobre el fichero entero porque la cabecera es
// la unica linea de la que se sabe que no lleva datos con comas dentro. Y el
// resultado se DICE en las notas: una deduccion que no se cuenta es una
// suposicion que nadie puede revisar.
//
// El caso que esto evita se midio: un fichero de ';' leido con ',' no da ningun
// error, da UNA columna llamada "usuario;permiso".
// Devuelve tambien los candidatos EMPATADOS con el ganador, si los hay. Un
// empate no se resuelve en silencio por el orden de una lista: se dice, que es
// lo que hace el motor con dos clasificaciones del mismo instante. Quien lea la
// instantanea tiene que poder ver que aqui hubo una eleccion.
func deducirSeparador(texto string) (rune, []rune) {
	cabecera := texto
	if i := strings.IndexAny(texto, "\r\n"); i >= 0 {
		cabecera = texto[:i]
	}
	cuentas := map[rune]int{}
	mejor, mejorN := separadoresCandidatos[0], -1
	for _, s := range separadoresCandidatos {
		n := strings.Count(cabecera, string(s))
		cuentas[s] = n
		if n > mejorN {
			mejor, mejorN = s, n
		}
	}
	var empatados []rune
	if mejorN > 0 {
		for _, s := range separadoresCandidatos {
			if s != mejor && cuentas[s] == mejorN {
				empatados = append(empatados, s)
			}
		}
	}
	return mejor, empatados
}

func nombreDelSeparador(r rune) string {
	switch r {
	case ';':
		return "punto y coma"
	case ',':
		return "coma"
	case '\t':
		return "tabulador"
	case '|':
		return "barra vertical"
	}
	return string(r)
}

// papeles dice en que columna esta cada cosa. -1 es "no esta".
type papeles struct{ cuenta, permiso, rotulo int }

// mapearColumnas empareja los nombres de la cabecera con los papeles.
//
// DOS COLUMNAS PARA EL MISMO PAPEL ES UN ERROR, NO UNA ELECCION. Si el fichero
// trae "usuario" y "id" y los dos estan declarados como identificador, adivinar
// cual manda es adivinar A QUIEN se esta revisando. Se para y se pregunta.
func mapearColumnas(cabecera []string, c Columnas) (papeles, []string, error) {
	p := papeles{cuenta: -1, permiso: -1, rotulo: -1}
	var ignoradas []string
	papelesDeclarados := []struct {
		destino   *int
		sinonimos []string
		nombre    string
		campo     string
	}{
		{&p.cuenta, c.Cuenta, "cuenta", "Columnas.Cuenta"},
		{&p.permiso, c.Permiso, "permiso", "Columnas.Permiso"},
		{&p.rotulo, c.Rotulo, "rotulo", "Columnas.Rotulo"},
	}
	for i, nombre := range cabecera {
		usada := false
		for _, pd := range papelesDeclarados {
			if !contiene(pd.sinonimos, normalizar(nombre)) {
				continue
			}
			if *pd.destino >= 0 {
				return papeles{}, nil, fmt.Errorf("%w: las columnas %q y %q valen las dos como %s.\n"+
					"  No se elige por orden ni por parecido: elegir mal la columna de identidad es "+
					"revisar a quien no era.\n"+
					"  Arreglo: quitar una del fichero, o dejar en %s solo el nombre que manda en "+
					"esta organizacion",
					ErrColumnaAmbigua, cabecera[*pd.destino], nombre, pd.nombre, pd.campo)
			}
			*pd.destino = i
			usada = true
		}
		if !usada {
			// MINIMIZACION: la columna no se guarda. Y se dice cual, porque un
			// descarte mudo de columnas es el mismo fallo que un descarte mudo
			// de filas mirado desde el otro eje.
			ignoradas = append(ignoradas, nombre)
		}
	}
	if p.cuenta < 0 {
		return papeles{}, nil, fmt.Errorf("%w.\n"+
			"  Cabeceras del fichero: %s.\n"+
			"  Nombres que se aceptan como identificador: %s.\n"+
			"  Arreglo: renombrar la columna en el fichero, o anadir su nombre a Columnas.Cuenta. "+
			"Sin identificador estable no hay nada que revisar: el nombre de una persona no "+
			"identifica, dos se llaman igual y una cambia de apellido",
			ErrSinColumnaDeCuenta, strings.Join(entrecomillar(cabecera), ", "),
			strings.Join(entrecomillar(c.Cuenta), ", "))
	}
	return p, ignoradas, nil
}

// aFila saca la fila del registro, o dice por que no se puede.
func aFila(registro []string, p papeles, sistema string, linea int) (Fila, string) {
	falta := func(i int) bool { return i < 0 || i >= len(registro) }
	if falta(p.cuenta) {
		return Fila{}, fmt.Sprintf("la fila tiene %d columnas y el identificador estaba en la %d",
			len(registro), p.cuenta+1)
	}
	f := Fila{
		Sistema: sistema,
		Cuenta:  strings.TrimSpace(registro[p.cuenta]),
		Linea:   linea,
	}
	if !falta(p.permiso) {
		f.Permiso = strings.TrimSpace(registro[p.permiso])
	}
	if !falta(p.rotulo) {
		f.Rotulo = strings.TrimSpace(registro[p.rotulo])
	}
	if f.Cuenta == "" {
		return Fila{}, "el identificador de cuenta esta vacio"
	}
	// Un carcater de control dentro de un identificador no es un identificador:
	// se cuela en logs, en pantallas y en la clave por la que se empareja todo.
	// Medido: encoding/csv deja pasar un byte nulo sin decir nada.
	if i := strings.IndexFunc(f.Cuenta, esControl); i >= 0 {
		return Fila{}, fmt.Sprintf("el identificador lleva un caracter de control (%U) en la "+
			"posicion %d", f.Cuenta[i], i)
	}
	if i := strings.IndexFunc(f.Permiso, esControl); i >= 0 {
		return Fila{}, fmt.Sprintf("el permiso lleva un caracter de control (%U) en la posicion %d",
			f.Permiso[i], i)
	}
	return f, ""
}

func esControl(r rune) bool {
	return r != '\t' && unicode.IsControl(r)
}

// span dice que lineas se ha llevado un error.
//
// IMPORTA MAS DE LO QUE PARECE: una comilla sin cerrar en la linea 3 se traga la
// 4 y la 5, y el error lo dice ("record on line 3; parse error on line 5"). Sin
// leer ese rango, esas dos lineas no estarian en ningun cubo y la ley de
// conservacion las echaria de menos sin saber donde estan.
func span(err error, linea int) (int, int) {
	var pe *csv.ParseError
	if errors.As(err, &pe) {
		desde, hasta := pe.StartLine, pe.Line
		if desde <= 0 {
			desde = linea
		}
		if hasta < desde {
			hasta = desde
		}
		return desde, hasta
	}
	return linea, linea
}

func motivoLegible(err error) string {
	var pe *csv.ParseError
	if errors.As(err, &pe) {
		switch {
		case errors.Is(pe.Err, csv.ErrQuote):
			return "comilla sin cerrar o suelta: el lector se lleva por delante las lineas " +
				"siguientes hasta encontrar el cierre"
		case errors.Is(pe.Err, csv.ErrFieldCount):
			return "la fila tiene un numero de columnas distinto del de la cabecera"
		case errors.Is(pe.Err, csv.ErrBareQuote):
			return "una comilla suelta dentro de un campo sin entrecomillar"
		}
		return pe.Err.Error()
	}
	return err.Error()
}

// avisoDeListaEnUnCampo se fija en los permisos que parecen varios metidos en
// uno ("admin,lector"). NO LOS PARTE: partirlos seria adivinar, y un permiso
// llamado "lectura, escritura" existe. Se dice y decide una persona.
func avisoDeListaEnUnCampo(permiso string) string {
	for _, s := range []string{",", ";", "|"} {
		if strings.Contains(permiso, s) {
			return fmt.Sprintf("el permiso %q lleva un %q dentro: si el fichero mete varios permisos "+
				"en una celda, cada uno tiene que ir en su fila o no se puede revocar por separado. "+
				"No se parte aqui: partirlo seria adivinar", permiso, s)
		}
	}
	return ""
}

func normalizar(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch r {
		case 'á', 'à', 'ä', 'â':
			b.WriteRune('a')
		case 'é', 'è', 'ë', 'ê':
			b.WriteRune('e')
		case 'í', 'ì', 'ï', 'î':
			b.WriteRune('i')
		case 'ó', 'ò', 'ö', 'ô':
			b.WriteRune('o')
		case 'ú', 'ù', 'ü', 'û':
			b.WriteRune('u')
		case ' ', '_', '-', '.':
			// se comen: "user name", "user_name" y "username" son la misma
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// plural existe porque estos errores los lee una persona que no ha escrito este
// paquete, y "Quedan 1 lineas" hace dudar de todo lo demas que diga la pantalla.
func plural(n int, uno, varios string) string {
	if n == 1 {
		return uno
	}
	return fmt.Sprintf(varios, n)
}

func contiene(xs []string, x string) bool {
	for _, y := range xs {
		if normalizar(y) == x {
			return true
		}
	}
	return false
}

func entrecomillar(xs []string) []string {
	out := make([]string, len(xs))
	for i, x := range xs {
		out[i] = fmt.Sprintf("%q", x)
	}
	return out
}
