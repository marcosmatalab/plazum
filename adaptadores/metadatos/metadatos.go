// Package metadatos propone la ficha de un documento del cliente a partir de su
// texto: fecha, alcance, firmante y caducidad.
//
// Es la pieza 7 de `docs/ia.md`.
//
// # LO QUE PROPONE SON CAMPOS, NUNCA JUICIOS (invariante 13)
//
// «Este documento lo firma Marta Ruiz» es un HECHO sobre el documento; «este
// documento acredita el control 4.2.3» es un juicio, y aqui no se emite ninguno.
// La frontera es la misma que ya tiene puesta `adaptadores/evidencia` y la misma
// que vigila la puerta del tipo: `Propuesta` no lleva ni un campo con forma de
// veredicto, y lo que sale de aqui no dice si el documento sirve para nada.
//
// # Y SE PROPONE, NO SE ESCRIBE. Lo confirma una persona
//
// Ninguna propuesta de este paquete es un dato del expediente hasta que alguien
// la acepta, y lo aceptado guarda QUIEN lo acepto y CUANDO. El motivo no es
// ceremonial: una fecha de caducidad sacada de un PDF por una expresion regular
// puede ser la del documento, la de un anexo o la del pie de pagina de una
// plantilla, y las tres se parecen. Quien responde por el expediente es una
// persona, asi que la que decide es una persona.
//
// # CADA PROPUESTA VIAJA CON EL TROZO LITERAL DEL QUE SALE
//
// No hay ninguna propuesta sin cita. `Propuesta.Cita` es el texto EXACTO del
// documento —con sus espacios y su tipografia— del que se ha sacado el valor, y
// `Fuente` es el identificador del fragmento. Eso es lo que permite verificarla
// por hash antes de ensenarla: si la cita no resuelve contra la fuente, la
// propuesta se descarta y no llega a la pantalla. La verificacion NO se hace
// aqui, se hace en el cable, con el mismo `ia.Verificador` que ya usa la pieza 3.
//
// # SIN MODELO, y eso es la decision
//
// La extraccion es determinista: patrones fijos sobre el texto ya extraido. Eso
// hace que funcione con `PLAZUM_SIN_IA=1`, que el mismo documento de siempre el
// mismo resultado, y sobre todo que se pueda AUDITAR: cada propuesta se puede
// contrastar contra el trozo que la produjo sin creerse a nadie.
//
// Lo que se paga a cambio esta contado y no disimulado: los patrones cazan las
// formas frecuentes en castellano y en ingles, y no cazan las demas. Una ficha
// vacia no dice que el documento no tenga fecha: dice que aqui no se ha
// encontrado ninguna, y la pantalla lo dice con esas palabras.
package metadatos

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/marcosmatalab/plazum/adaptadores/ingesta"
)

// Campo es que se propone. Vocabulario cerrado: son los cuatro que nombra la
// casilla de `ETAPAS.md`, y uno nuevo entra aqui a proposito y no por descuido.
type Campo string

const (
	// Fecha es la del documento: cuando se emitio o se aprobo.
	Fecha Campo = "fecha"
	// Alcance es a que se aplica: sistemas, sedes, unidades.
	Alcance Campo = "alcance"
	// Firmante es quien lo firma o lo aprueba.
	Firmante Campo = "firmante"
	// Caducidad es hasta cuando vale.
	Caducidad Campo = "caducidad"
)

// Campos son los cuatro, en el orden en que se pintan.
func Campos() []Campo { return []Campo{Fecha, Alcance, Firmante, Caducidad} }

// Propuesta es un valor propuesto para un campo, con el trozo del que sale.
//
// NO LLEVA NI UN CAMPO CON FORMA DE JUICIO, y no es casualidad: lo vigila la
// puerta del invariante 13 sobre los tipos que cruzan a la pantalla. No hay
// «confianza», no hay puntuacion y no hay «valido»: hay un valor, de donde sale,
// y el trozo literal que lo dice.
type Propuesta struct {
	Campo Campo
	// Valor es lo propuesto, ya normalizado cuando se puede (una fecha sale en
	// AAAA-MM-DD). Es lo unico que alguien va a aceptar.
	Valor string
	// Cita es el trozo LITERAL del documento del que sale, con sus espacios.
	// Sin ella no hay propuesta: es lo que se verifica por hash.
	Cita string
	// Fuente es el identificador del fragmento, y Pagina y Orden dicen donde
	// esta dentro del documento. Pagina 0 significa «no se sabe» y no «la
	// pagina cero»: en un formato sin paginas no hay ninguna.
	Fuente string
	Pagina int
	Orden  int
}

// Los patrones. SE DECLARAN AQUI Y NO SE COMPONEN EN EL BUCLE por dos razones:
// compilar una expresion regular por fragmento y por campo es caro sobre un
// documento largo, y sobre todo porque asi la lista de lo que se reconoce es
// LEGIBLE, que es lo que permite decir en voz alta que NO se reconoce.
//
// Los dos idiomas van juntos a proposito: el documento lo escribe el cliente y
// puede estar en cualquiera de los dos, y separar los patrones por idioma
// obligaria a adivinar el idioma del documento, que es un problema mas grande
// que el que se esta resolviendo.
var (
	// La etiqueta que introduce cada campo. `(?i)` porque un PDF escribe
	// «FECHA:», «Fecha:» y «fecha:» sin avisar.
	reEtiquetaFecha = regexp.MustCompile(
		`(?i)\b(fecha(?: de (?:emision|aprobacion|documento))?|date|issued(?: on)?|aprobado el)\s*[:\-]\s*(.{4,60})`)
	reEtiquetaCaducidad = regexp.MustCompile(
		`(?i)\b(caducidad|valido hasta|vigencia hasta|vigente hasta|expira(?: el)?|valid until|expiry(?: date)?|expires(?: on)?)\s*[:\-]\s*(.{4,60})`)
	reEtiquetaFirmante = regexp.MustCompile(
		`(?i)\b(firmado por|firma|firmante|aprobado por|autorizado por|signed by|approved by)\s*[:\-]\s*(.{3,80})`)
	reEtiquetaAlcance = regexp.MustCompile(
		`(?i)\b(alcance|ambito(?: de aplicacion)?|aplicable a|scope|applies to)\s*[:\-]\s*(.{3,120})`)

	// Las formas de fecha que se reconocen, en el orden en que se intentan.
	reFechaISO     = regexp.MustCompile(`\b(\d{4})-(\d{2})-(\d{2})\b`)
	reFechaBarras  = regexp.MustCompile(`\b(\d{1,2})[/.](\d{1,2})[/.](\d{4})\b`)
	reFechaEnLetra = regexp.MustCompile(
		`(?i)\b(\d{1,2})\s+de\s+([a-zñáéíóú]+)\s+de\s+(\d{4})\b`)
)

// mesesEnLetra son los nombres que reconoce reFechaEnLetra. Sin tildes: el texto
// se normaliza antes de buscar, por lo mismo que el resto del arbol.
var mesesEnLetra = map[string]time.Month{
	"enero": time.January, "febrero": time.February, "marzo": time.March,
	"abril": time.April, "mayo": time.May, "junio": time.June,
	"julio": time.July, "agosto": time.August, "septiembre": time.September,
	"setiembre": time.September, "octubre": time.October,
	"noviembre": time.November, "diciembre": time.December,
}

// MaxPropuestasPorCampo acota cuantas se guardan de cada campo.
//
// EL NUMERO SALE DE PARA QUE SIRVE LA PANTALLA. Quien mira una ficha propuesta
// elige entre unas pocas o se rinde: veinte candidatos a «fecha» no son mas
// informacion, son la misma pantalla que se estaba intentando evitar. Y hay un
// motivo mas duro: un documento adversario puede traer diez mil lineas que
// digan «Fecha:», y sin tope esto seria un amplificador de memoria por una
// subida de 4 MiB.
const MaxPropuestasPorCampo = 5

// Proponer saca la ficha de un documento ya leido.
//
// # EL ORDEN IMPORTA Y ES EL DEL DOCUMENTO
//
// Las propuestas salen en el orden en que aparecen en el texto, y NO ordenadas
// por ninguna heuristica de calidad. Ordenarlas por «cual parece mejor» seria
// emitir un juicio por la puerta de atras: la primera de una lista se lee como
// la recomendada, y aqui no hay nada que recomiende nada. La primera es la
// primera que aparece, y eso se puede comprobar mirando el documento.
//
// # UN FICHERO SIN NADA RECONOCIBLE DEVUELVE CERO PROPUESTAS Y NINGUN ERROR
//
// No haber encontrado una fecha no es un fallo de lectura ni una carencia del
// documento: es que aqui no se ha reconocido ninguna. La diferencia la dice la
// pantalla; este paquete solo devuelve lo que ha encontrado.
func Proponer(doc ingesta.Documento, fuenteDe func(orden int) string) []Propuesta {
	if fuenteDe == nil {
		// Sin forma de identificar el fragmento no se puede citar, y una
		// propuesta sin cita no se puede verificar: no se devuelve ninguna. Es
		// el valor cero restrictivo (invariante 8).
		return nil
	}
	var out []Propuesta
	porCampo := map[Campo]int{}
	for _, f := range doc.Fragmentos {
		for _, c := range candidatosDelFragmento(f) {
			if porCampo[c.Campo] >= MaxPropuestasPorCampo {
				continue
			}
			c.Fuente = fuenteDe(f.Orden)
			c.Pagina = f.Pagina
			c.Orden = f.Orden
			if c.Fuente == "" || c.Cita == "" || c.Valor == "" {
				continue
			}
			porCampo[c.Campo]++
			out = append(out, c)
		}
	}
	return out
}

// candidatosDelFragmento saca lo que haya en un fragmento.
func candidatosDelFragmento(f ingesta.Fragmento) []Propuesta {
	var out []Propuesta
	// LAS DOS FECHAS PRIMERO, y en este orden: caducidad ANTES que fecha. Una
	// linea que dice «Valido hasta: 2027-01-01» casa las dos etiquetas si se
	// mira la de fecha primero, porque «hasta» no la excluye. Mirando la mas
	// especifica antes, y quitando del texto lo que ya caso, cada linea aporta
	// como mucho una fecha y a su campo correcto.
	resto := f.Texto
	if p, consumido, ok := unaEtiqueta(resto, reEtiquetaCaducidad, Caducidad, true); ok {
		out = append(out, p)
		resto = strings.Replace(resto, consumido, " ", 1)
	}
	if p, _, ok := unaEtiqueta(resto, reEtiquetaFecha, Fecha, true); ok {
		out = append(out, p)
	}
	if p, _, ok := unaEtiqueta(f.Texto, reEtiquetaFirmante, Firmante, false); ok {
		out = append(out, p)
	}
	if p, _, ok := unaEtiqueta(f.Texto, reEtiquetaAlcance, Alcance, false); ok {
		out = append(out, p)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return strings.Index(f.Texto, out[i].Cita) < strings.Index(f.Texto, out[j].Cita)
	})
	return out
}

// unaEtiqueta busca una etiqueta y saca su valor.
//
// Devuelve ademas el trozo consumido, para que quien llama pueda quitarlo del
// texto antes de buscar una etiqueta menos especifica.
//
// LA CITA ES LA ETIQUETA MAS SU VALOR, y ni una palabra mas.
//
// # Las dos mitades de esa frase, y las dos costaron
//
// LA ETIQUETA VA DENTRO porque «2027-01-01» suelto no dice nada y ademas puede
// aparecer veinte veces en un documento: verificarlo por hash no probaria de
// donde salio, y quien lo lea no puede decidir si acepta.
//
// Y NI UNA PALABRA MAS porque la primera version citaba lo que casara la
// expresion regular, que son hasta 120 caracteres. Un PDF junta las lineas de un
// parrafo, asi que la cita de «Alcance:» salia siendo «Alcance: los sistemas de
// la sede de Madrid Firmado por: Marta Ruiz Valido hasta: 2027-01-15»: tres
// campos dentro de la cita de uno. Lo dijo el test de extremo a extremo de la
// pieza 7 en su primera ejecucion.
//
// La cita se recorta SIEMPRE como subcadena literal del fragmento, nunca
// recomponiendo texto: si no fuera literal, la verificacion por hash la
// descartaria, y con razon.
func unaEtiqueta(texto string, re *regexp.Regexp, campo Campo, esFecha bool) (Propuesta, string, bool) {
	m := re.FindStringSubmatchIndex(texto)
	if m == nil {
		return Propuesta{}, "", false
	}
	crudo := texto[m[4]:m[5]]
	var valor, enBruto string
	if esFecha {
		f, b, ok := unaFecha(crudo)
		if !ok {
			return Propuesta{}, "", false
		}
		valor, enBruto = f, b
	} else {
		valor = hastaElFinalDeLaFrase(strings.TrimSpace(crudo))
		if valor == "" {
			return Propuesta{}, "", false
		}
		enBruto = valor
	}
	// DONDE ACABA EL VALOR DENTRO DEL TEXTO, para cortar la cita ahi.
	i := strings.Index(crudo, enBruto)
	if i < 0 {
		// No deberia poder pasar: `enBruto` sale de `crudo`. Si pasara, la cita
		// no seria literal y el verificador la tiraria, asi que se descarta aqui
		// en vez de producir una propuesta que nace muerta.
		return Propuesta{}, "", false
	}
	cita := strings.TrimSpace(texto[m[0] : m[4]+i+len(enBruto)])
	return Propuesta{Campo: campo, Valor: valor, Cita: cita}, cita, true
}

// hastaElFinalDeLaFrase recorta el valor de un campo de texto libre.
//
// Un PDF junta lineas, asi que «Firmado por: Marta Ruiz Directora de Seguridad
// Fecha: 2026-01-02» llega en una sola cadena. Se corta en el primer separador
// fuerte para no arrastrar media pagina dentro de un nombre.
func hastaElFinalDeLaFrase(s string) string {
	for _, corte := range []string{".", ";", "  ", "\t"} {
		if i := strings.Index(s, corte); i > 0 {
			s = s[:i]
		}
	}
	// Y si detras viene otra etiqueta, se corta ahi.
	for _, re := range []*regexp.Regexp{
		reEtiquetaFecha, reEtiquetaCaducidad, reEtiquetaFirmante, reEtiquetaAlcance,
	} {
		if m := re.FindStringIndex(s); m != nil && m[0] > 0 {
			s = s[:m[0]]
		}
	}
	s = strings.TrimSpace(strings.Trim(strings.TrimSpace(s), ",-:"))
	if len([]rune(s)) < 2 {
		return ""
	}
	return s
}

// unaFecha normaliza a AAAA-MM-DD lo que reconozca, y dice que no cuando no
// reconoce nada.
//
// NO SE INVENTA NINGUNA. Un texto del que no sale una fecha valida NO produce
// propuesta: es la tercera forma de la nada del invariante 8, la del dato que
// esta y no se entiende, y su respuesta es no proponer, nunca un valor por
// defecto. Una fecha inventada en una ficha de caducidad se acepta de un clic y
// se queda en el expediente.
// Devuelve ademas el trozo EN BRUTO que ha casado, para que quien llama pueda
// cortar la cita justo donde acaba la fecha.
func unaFecha(s string) (string, string, bool) {
	if m := reFechaISO.FindStringSubmatch(s); m != nil {
		v, ok := validar(m[1], m[2], m[3])
		return v, m[0], ok
	}
	if m := reFechaBarras.FindStringSubmatch(s); m != nil {
		// DIA/MES/ANO y no MES/DIA/ANO: es la forma de los dos idiomas que
		// este producto sirve en Europa. Se dice porque la ambiguedad es real y
		// elegir en silencio seria acertar por casualidad la mitad de las veces.
		v, ok := validar(m[3], m[2], m[1])
		return v, m[0], ok
	}
	if m := reFechaEnLetra.FindStringSubmatch(s); m != nil {
		mes, ok := mesesEnLetra[strings.ToLower(sinTildes(m[2]))]
		if !ok {
			return "", "", false
		}
		v, ok := validar(m[3], strconv.Itoa(int(mes)), m[1])
		return v, m[0], ok
	}
	return "", "", false
}

// validar comprueba que la fecha existe de verdad y la normaliza.
//
// UN 31 DE FEBRERO NO ES UNA FECHA, y `time.Parse` con un formato fijo lo
// rechaza, que es justo lo que hace falta: aceptar «2026-02-31» y guardarlo
// produciria una caducidad que ningun reloj puede calcular.
func validar(ano, mes, dia string) (string, bool) {
	a, err1 := strconv.Atoi(ano)
	m, err2 := strconv.Atoi(mes)
	d, err3 := strconv.Atoi(dia)
	if err1 != nil || err2 != nil || err3 != nil {
		return "", false
	}
	if a < 1900 || a > 2200 {
		// Un ano fuera de rango es casi siempre otra cosa: un numero de
		// telefono, un codigo, un importe. Se descarta en vez de proponerlo.
		return "", false
	}
	s := time.Date(a, time.Month(m), d, 0, 0, 0, 0, time.UTC)
	if s.Year() != a || int(s.Month()) != m || s.Day() != d {
		return "", false
	}
	return s.Format("2006-01-02"), true
}

// sinTildes quita los acentos de los nombres de mes.
func sinTildes(s string) string {
	r := strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u")
	return r.Replace(s)
}
