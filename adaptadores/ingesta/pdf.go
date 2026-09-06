package ingesta

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// LA EXTRACCION DE TEXTO DE UN PDF, CON CERO DEPENDENCIAS Y CON SUS LIMITES
// ESCRITOS.
//
// # Por que a mano
//
// `go.mod` de este repositorio no tiene una sola linea `require`, lo vigila
// TestElBinarioNoLlevaNingunaDependenciaExterna y una puerta de CI corre la
// suite entera con GOPROXY=off. Meter una libreria de PDF no es una linea: es
// una decision de producto con su fila en DEPENDENCIAS.md, y ademas es una
// libreria que parsea entrada adversaria, que es la peor clase de dependencia
// que se puede tener en un producto que recibe ficheros de desconocidos.
//
// Lo que hace falta aqui es MUCHO menos que un PDF completo: sacar el texto de
// los flujos de contenido. Todo lo demas (formas, imagenes, anotaciones,
// formularios) no se cita nunca.
//
// # LO QUE ESTE EXTRACTOR NO SABE HACER, dicho entero
//
//	PDF escaneado sin capa de texto      no sale texto. ERROR, no cero.
//	fuente incrustada sin ToUnicode      salen bytes que no son lenguaje.
//	                                     Los caza el contraste de Leer. ERROR.
//	filtros que no son FlateDecode       (LZW, RunLength, ASCII85) se saltan.
//	                                     Si TODO el documento va asi, ERROR.
//	flujos de objeto (ObjStm) comprimidos con el arbol de paginas dentro
//	                                     no se recorre el arbol; el contaje de
//	                                     paginas puede no salir, y entonces la
//	                                     pagina se declara DESCONOCIDA (0).
//	PDF cifrado                          ERROR explicito, nunca basura.
//
// Ninguna de esas limitaciones puede producir texto equivocado presentado como
// bueno: todas acaban en error o en «no lo se». Esa es la unica propiedad que
// aqui importa de verdad, porque el texto que salga de aqui va a acabar debajo
// de una cita en una pantalla de cumplimiento.
//
// # LA PAGINA SE DECLARA O SE DICE QUE NO SE SABE (invariante 8)
//
// La pagina es lo que se le ensena a una persona: «tu politica, pagina 4». Una
// pagina equivocada es peor que ninguna, porque manda a alguien a mirar donde no
// esta y le hace dudar del resto. Asi que solo se numera cuando el contaje de
// paginas del documento CUADRA con el numero de flujos de contenido con texto.
// Si no cuadra, todos los fragmentos salen con pagina 0, que significa «no lo
// se» y la pantalla lo dice.

var (
	reTipoPagina = regexp.MustCompile(`/Type\s*/Page[^s]`)
	reEncrypt    = regexp.MustCompile(`/Encrypt\s`)
)

// extraerPDF saca el texto de los flujos de contenido de un PDF.
func extraerPDF(datos []byte) (paginas int, frags []Fragmento, truncado bool, err error) {
	if reEncrypt.Match(datos) {
		return 0, nil, false, fmt.Errorf(
			"%w. Arreglo: quitale la contrasena y vuelve a subirlo, o exportalo a "+
				"texto. plazum no intenta descifrarlo: un producto que adivina "+
				"contrasenas de documentos ajenos no es lo que quieres tener instalado",
			ErrPDFCifrado)
	}

	paginas = len(reTipoPagina.FindAll(datos, -1))

	flujos, trunc := flujosDeContenido(datos)
	truncado = trunc

	// Los flujos que traen texto, en el orden en que aparecen.
	var textos []string
	for _, f := range flujos {
		t := textoDeFlujo(f)
		if strings.TrimSpace(t) != "" {
			textos = append(textos, t)
		}
	}
	if len(textos) == 0 {
		return paginas, nil, truncado, nil
	}

	// LA DECISION DE LA PAGINA. Solo se numera si cuadra.
	numerar := paginas > 0 && paginas == len(textos)

	orden := 0
	for i, t := range textos {
		pagina := 0
		if numerar {
			pagina = i + 1
		}
		trozos, cortado := fragmentarTexto(t)
		if cortado {
			truncado = true
		}
		for _, tr := range trozos {
			if len(frags) >= MaximoFragmentos {
				return paginas, frags, true, nil
			}
			orden++
			frags = append(frags, Fragmento{Pagina: pagina, Orden: orden, Texto: tr.Texto})
		}
	}
	return paginas, frags, truncado, nil
}

// flujosDeContenido devuelve los flujos ya descomprimidos que pueden llevar
// texto. Los de imagen se descartan por su diccionario, no por su contenido.
func flujosDeContenido(datos []byte) (out [][]byte, truncado bool) {
	total := 0
	resto := datos
	desplazamiento := 0
	for {
		i := bytes.Index(resto, []byte("stream"))
		if i < 0 {
			return out, truncado
		}
		// El diccionario del objeto es lo que hay justo antes. 2 KiB sobran
		// para un diccionario de flujo y evitan recorrer el fichero entero por
		// cada flujo, que seria cuadratico.
		ini := i - 2048
		if ini < 0 {
			ini = 0
		}
		dic := string(resto[ini:i])

		cuerpo := resto[i+len("stream"):]
		// Tras `stream` va CRLF o LF, y no otra cosa.
		switch {
		case bytes.HasPrefix(cuerpo, []byte("\r\n")):
			cuerpo = cuerpo[2:]
		case bytes.HasPrefix(cuerpo, []byte("\n")):
			cuerpo = cuerpo[1:]
		case bytes.HasPrefix(cuerpo, []byte("\r")):
			cuerpo = cuerpo[1:]
		default:
			// No era la palabra clave sino parte de otra (`endstream`). Se
			// avanza y se sigue: un PDF raro no puede pararnos el recorrido.
			resto = resto[i+len("stream"):]
			desplazamiento += i + len("stream")
			continue
		}
		fin := bytes.Index(cuerpo, []byte("endstream"))
		if fin < 0 {
			return out, truncado
		}
		bruto := cuerpo[:fin]
		avance := (i + len("stream")) + (len(cuerpo) - fin) // solo para mover el cursor
		_ = avance
		resto = cuerpo[fin+len("endstream"):]

		if esFlujoDeImagen(dic) {
			continue
		}
		var texto []byte
		switch {
		case strings.Contains(dic, "/FlateDecode"):
			d, cortado, err := inflar(bruto, MaximoExtraido-total)
			if err != nil {
				continue // un flujo que no infla no para el documento
			}
			truncado = truncado || cortado
			texto = d
		case !strings.Contains(dic, "/Filter"):
			texto = bruto
		default:
			// Filtro que no sabemos deshacer. No se adivina.
			continue
		}
		total += len(texto)
		out = append(out, texto)
		if total >= MaximoExtraido {
			return out, true
		}
	}
}

func esFlujoDeImagen(dic string) bool {
	for _, m := range []string{"/Image", "/DCTDecode", "/JPXDecode", "/CCITTFaxDecode"} {
		if strings.Contains(dic, m) {
			return true
		}
	}
	return false
}

// inflar descomprime con tope. El tope es la guarda contra la bomba: un flujo
// de 3 KB puede declarar gigabytes al descomprimirse.
func inflar(b []byte, tope int) (out []byte, truncado bool, err error) {
	if tope <= 0 {
		return nil, true, nil
	}
	z, err := zlib.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = z.Close() }()
	// tope+1 para poder DISTINGUIR «justo el tope» de «se paso», que si no se
	// leeria un documento cortado como uno completo.
	out, err = io.ReadAll(io.LimitReader(z, int64(tope)+1))
	if err != nil && len(out) == 0 {
		return nil, false, err
	}
	if len(out) > tope {
		return out[:tope], true, nil
	}
	return out, false, nil
}

// textoDeFlujo saca el texto de un flujo de contenido ya descomprimido.
//
// Se atienden los operadores de texto (Tj, TJ, ' y ") y los de salto de linea
// (Td, TD, T*), que es lo que separa un parrafo de otro. Todo lo demas (graficos,
// estados, matrices) se ignora, que es lo correcto: aqui no se pinta nada.
func textoDeFlujo(b []byte) string {
	var sb strings.Builder
	s := string(b)
	i := 0
	// pendiente son las cadenas del operador actual, que no se emiten hasta
	// saber cual es: `(a) (b) TJ` no es lo mismo que `(a) Tj`.
	var pendiente []string
	emitir := func() {
		for _, p := range pendiente {
			sb.WriteString(p)
		}
		pendiente = nil
	}
	for i < len(s) {
		switch s[i] {
		case '(':
			lit, n := cadenaLiteral(s[i:])
			pendiente = append(pendiente, lit)
			i += n
		case '<':
			if i+1 < len(s) && s[i+1] == '<' {
				i += 2
				continue
			}
			lit, n := cadenaHex(s[i:])
			pendiente = append(pendiente, lit)
			i += n
		case 'T':
			if i+1 < len(s) {
				switch s[i+1] {
				case 'j', 'J':
					emitir()
					sb.WriteString(" ")
				case 'd', 'D', '*':
					emitir()
					sb.WriteString("\n")
				}
			}
			i += 2
		case '\'', '"':
			emitir()
			sb.WriteString("\n")
			i++
		case 'E':
			if strings.HasPrefix(s[i:], "ET") {
				emitir()
				sb.WriteString("\n\n")
				i += 2
				continue
			}
			i++
		default:
			i++
		}
	}
	emitir()
	return sb.String()
}

// cadenaLiteral lee una cadena PDF `(...)` con sus escapes y sus parentesis
// anidados, y devuelve cuantos bytes ha consumido.
func cadenaLiteral(s string) (string, int) {
	var sb strings.Builder
	nivel := 0
	i := 0
	for i < len(s) {
		c := s[i]
		switch c {
		case '(':
			nivel++
			if nivel > 1 {
				sb.WriteByte(c)
			}
			i++
		case ')':
			nivel--
			if nivel == 0 {
				return aTexto(sb.String()), i + 1
			}
			sb.WriteByte(c)
			i++
		case '\\':
			if i+1 >= len(s) {
				return aTexto(sb.String()), i + 1
			}
			d := s[i+1]
			switch d {
			case 'n':
				sb.WriteByte('\n')
			case 'r':
				sb.WriteByte('\r')
			case 't':
				sb.WriteByte('\t')
			case 'b', 'f':
				sb.WriteByte(' ')
			case '(', ')', '\\':
				sb.WriteByte(d)
			case '\n':
				// continuacion de linea: no produce nada
			case '\r':
				if i+2 < len(s) && s[i+2] == '\n' {
					i++
				}
			default:
				if d >= '0' && d <= '7' {
					j, v := i+1, 0
					for j < len(s) && j < i+4 && s[j] >= '0' && s[j] <= '7' {
						v = v*8 + int(s[j]-'0')
						j++
					}
					sb.WriteByte(byte(v))
					i = j
					continue
				}
				sb.WriteByte(d)
			}
			i += 2
		default:
			sb.WriteByte(c)
			i++
		}
	}
	return aTexto(sb.String()), i
}

// cadenaHex lee una cadena PDF `<...>`.
func cadenaHex(s string) (string, int) {
	fin := strings.IndexByte(s, '>')
	if fin < 0 {
		return "", len(s)
	}
	h := strings.Map(func(r rune) rune {
		if strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return r
		}
		return -1
	}, s[1:fin])
	if len(h)%2 == 1 {
		h += "0" // lo dice la especificacion: el ultimo digito impar se rellena
	}
	var sb strings.Builder
	for i := 0; i+1 < len(h); i += 2 {
		v, err := strconv.ParseUint(h[i:i+2], 16, 8)
		if err != nil {
			continue
		}
		sb.WriteByte(byte(v))
	}
	return aTexto(sb.String()), fin + 1
}

// aTexto convierte los bytes de una cadena PDF a runas.
//
// Se interpretan como Latin-1, que es lo que produce WinAnsiEncoding para el
// texto occidental y por tanto acierta con los acentos de un documento en
// espanol. Si el PDF usa otra codificacion sale basura, Y ESO ES LO QUE SE
// QUIERE: la basura la caza el contraste de interpretabilidad de `Leer` y sale
// un error. Lo que no puede pasar es que salga texto plausible y equivocado.
func aTexto(b string) string {
	if esASCII(b) {
		return b
	}
	rs := make([]rune, 0, len(b))
	for i := 0; i < len(b); i++ {
		rs = append(rs, rune(b[i]))
	}
	return string(rs)
}

func esASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}
