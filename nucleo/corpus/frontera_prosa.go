package corpus

// La puerta de atras de la prosa.
//
// EL HALLAZGO, y no es teorico. Al justificar el intervalo del punto 6.7.3 del
// anexo de 2024/2690, el argumento que salio fue "el sector de medios de pago
// lleva anos exigiendo la revision del conjunto de reglas de cortafuegos cada
// seis meses". Eso es CONTENIDO DE PCI DSS parafraseado, y el linter no lo veia:
// el limite de la frontera legal mide LONGITUD, no PROCEDENCIA. Un campo de 200
// caracteres pasa igual lleve dentro un razonamiento propio o el criterio de un
// catalogo de pago.
//
// LAS DOS CAPAS, porque una sola no llega:
//
//	lintable   un paquete no NOMBRA un marco de estrato cerrado que no sea el
//	           suyo. Es un proxy: no caza la parafrasis anonima, pero cierra la
//	           via ancha y no tiene ni una decision de juicio dentro.
//	humana     la parafrasis SIN nombre (el caso real de arriba) la caza la
//	           pasada de coherencia, preguntando si el argumento se sostiene sin
//	           el apoyo fantasma. Va declarada en docs/pendientes.md como
//	           familia propia, porque va a reaparecer en ayudas, en plantillas y
//	           en la IA cuando llegue.
//
// LA LISTA NEGRA SE DERIVA DEL CORPUS, no se mantiene a mano. Un marco entra en
// ella por su CLASE (referencial o delegado), que ya esta declarada en su
// paquete, asi que instalar un catalogo nuevo lo protege solo. Una lista escrita
// a mano seria una segunda lista, y una segunda lista se queda vieja.
//
// LO QUE ESTA REGLA NO MIRA NUNCA, y es la frontera importante: `texto_legal`.
// Ahi va transcrito lo que dice el boletin, y el boletin nombra normas privadas
// continuamente (el ENS remite a ISO/IEC 27001 en varios sitios). Aplicar la
// regla al texto legal seria CENSURAR LA LEY para cumplir una regla nuestra, que
// es exactamente al reves de para que existe. La regla mira la prosa de plazum:
// citas, titulos, ayudas, notas y justificaciones.

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

// ErrProsaNombraMarcoCerrado: la prosa de un paquete nombra un marco de estrato
// referencial o delegado que no es el suyo.
var ErrProsaNombraMarcoCerrado = errors.New("la prosa del paquete nombra un marco de estrato cerrado ajeno")

// genericosDelURN son segmentos que no identifican a nadie por si solos. Sin
// esta lista, "dss" casaria con cualquier mencion y "iso" con todas.
var genericosDelURN = map[string]bool{
	"urn": true, "iso": true, "iec": true, "dss": true, "tsc": true, "benchmarks": true,
}

// nombresDelMarco deriva las formas con las que se puede nombrar a un marco, del
// URN que el propio paquete declara y del directorio en el que vive.
//
// El directorio hace falta: `urn:pcissc:dss:4` no contiene "pci dss" por ningun
// lado, y el directorio se llama `pci-dss`. Entre los dos sale el vocabulario
// sin escribir ni un nombre a mano.
func nombresDelMarco(urn, dir string) []string {
	set := map[string]bool{}
	anadir := func(s string) {
		s = strings.TrimSpace(strings.ToLower(s))
		if len(s) >= 3 {
			set[s] = true
		}
	}
	anadir(dir)
	anadir(strings.ReplaceAll(dir, "-", " "))
	anadir(strings.ReplaceAll(dir, "-", ""))
	for _, seg := range strings.Split(urn, ":") {
		seg = strings.ToLower(strings.TrimSpace(seg))
		if len(seg) < 3 || genericosDelURN[seg] || esNumero(seg) {
			continue
		}
		anadir(seg)
	}
	// Las formas humanas de una norma ISO: el directorio `iso27001` se escribe
	// "ISO 27001" y "ISO/IEC 27001" mucho mas a menudo que "iso27001".
	if m := regexp.MustCompile(`^iso(\d{4,5})$`).FindStringSubmatch(dir); m != nil {
		anadir("iso " + m[1])
		anadir("iso/iec " + m[1])
		anadir("iso-iec " + m[1])
		anadir(m[1])
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func esNumero(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return len(s) > 0
}

// prosaDelPaquete devuelve los campos de texto libre que ESCRIBE PLAZUM, con su
// ruta. Deliberadamente NO incluye texto_legal (ver la cabecera del fichero).
func prosaDelPaquete(p *Paquete) map[string]string {
	out := map[string]string{}
	pon := func(ruta, txt string) {
		if strings.TrimSpace(txt) != "" {
			out[ruta] = txt
		}
	}
	pon("atribucion", p.Atribucion)
	for _, te := range p.Entidades {
		pon("entidad "+te.Nombre+": descripcion", te.Descripcion)
		for _, a := range te.Atributos {
			pon("entidad "+te.Nombre+", atributo "+a.Nombre+": ayuda", a.Ayuda)
			pon("entidad "+te.Nombre+", atributo "+a.Nombre+": cita", a.Cita)
		}
	}
	for _, q := range p.Preguntas {
		pon("pregunta "+q.ID+": texto", q.Texto)
		pon("pregunta "+q.ID+": cita", q.Cita)
	}
	for _, o := range p.Obligaciones {
		pon("obligacion "+o.ID+": titulo", o.Titulo)
		pon("obligacion "+o.ID+": cita", o.Cita)
		if t := o.Temporalidad; t != nil {
			pon("obligacion "+o.ID+": justificacion_del_intervalo", t.JustificacionDelIntervalo)
			pon("obligacion "+o.ID+": cita_del_intervalo", t.CitaDelIntervalo)
			pon("obligacion "+o.ID+": cadencia_distinta_porque", t.CadenciaDistintaPorque)
			for _, h := range t.Hitos {
				pon("obligacion "+o.ID+", hito "+h.ID+": nota", h.Nota)

			}
		}
	}
	for _, t := range p.Plantillas {
		pon("plantilla "+t.ID+": titulo", t.Titulo)
		pon("plantilla "+t.ID+": cita", t.Cita)
	}
	for _, r := range p.Aplicabilidad.Reglas {
		pon("regla "+r.ID+": cita", r.Cita)
	}
	for _, d := range p.Dorados {
		pon("dorado "+d.Caso+": cita_del_esperado", d.CitaDelEsperado)
	}
	return out
}

// ValidarProsaEntrePaquetes comprueba, sobre el corpus YA CARGADO, que ningun
// paquete nombre en su prosa un marco de estrato cerrado ajeno.
//
// Es cross-paquete por necesidad: la lista negra se deriva de las CLASES de los
// demas paquetes, asi que no se puede decidir mirando uno solo. Por eso vive
// aqui y no en Paquete.Validar.
//
// NOMBRARSE A SI MISMO SE PUEDE. El paquete de la 27001 tiene que poder decir
// "ISO/IEC 27001:2022": es su identidad, y el estrato referencial prohibe
// redistribuir su TEXTO, no mencionar que la norma existe.
func ValidarProsaEntrePaquetes(ps []*Paquete, dirs []string) []error {
	if len(ps) != len(dirs) {
		return []error{fmt.Errorf("ValidarProsaEntrePaquetes: %d paquetes y %d directorios",
			len(ps), len(dirs))}
	}
	type marco struct {
		urn     string
		nombres []string
	}
	var cerrados []marco
	for i, p := range ps {
		if p.Clase == Referencial || p.Clase == Delegado {
			cerrados = append(cerrados, marco{p.URN, nombresDelMarco(p.URN, dirs[i])})
		}
	}

	var errs []error
	for i, p := range ps {
		prosa := prosaDelPaquete(p)
		rutas := make([]string, 0, len(prosa))
		for r := range prosa {
			rutas = append(rutas, r)
		}
		sort.Strings(rutas) // orden estable del informe

		for _, ruta := range rutas {
			bajo := sinTildesMinusculas(prosa[ruta])
			for _, m := range cerrados {
				if m.urn == p.URN {
					continue // nombrarse a si mismo
				}
				for _, n := range m.nombres {
					if !nombraA(bajo, n) {
						continue
					}
					errs = append(errs, fmt.Errorf("%w: %s, en %s, nombra %q (%s). El estrato "+
						"referencial y el delegado no se pueden redistribuir, y un argumento "+
						"apoyado en lo que exige un catalogo de pago redistribuye su CRITERIO "+
						"aunque no copie su texto: el limite de caracteres mide longitud, no "+
						"procedencia. Apoya el argumento en fuente primaria (NIST, ENISA, BOE, "+
						"DOUE) o en el reloj propio del punto. Nombrar la norma de la que trata "+
						"el propio paquete si se puede; esta es ajena",
						ErrProsaNombraMarcoCerrado, p.URN, ruta, prosa[ruta][:min(len(prosa[ruta]), 90)], m.urn))
					break
				}
			}
		}
		_ = i
	}
	return errs
}

// nombraA busca el nombre con frontera de palabra por los dos lados. Sin la
// frontera, "cis" casaria dentro de "precision" y "decision", que en castellano
// es un falso positivo por linea.
//
// SIN REGEXP, Y NO ES MICRO-OPTIMIZACION. La primera version compilaba una
// expresion regular DENTRO de esta funcion, que se llama una vez por cada
// (campo de prosa x marco cerrado x forma de nombrarlo): con el corpus de hoy
// son decenas de miles de compilaciones por cada Cargar. Costaba 214 ms, y bajo
// `-race` eso se multiplica lo bastante como para que `plazum serve` no llegue
// a responder en los 5 s que su test le da. Lo caz la puerta de carreras de CI,
// que es una de las tres que la maquina local se salta por no tener cgo: o sea
// que el unico sitio donde se veia era el sitio donde no se miro.
//
// El recorrido a mano no compila nada, no reserva memoria y hace exactamente lo
// mismo: buscar la subcadena y mirar que a los lados no haya alfanumerico.
func nombraA(texto, nombre string) bool {
	desde := 0
	for {
		i := strings.Index(texto[desde:], nombre)
		if i < 0 {
			return false
		}
		i += desde
		fin := i + len(nombre)
		if !alfanumericoEn(texto, i-1) && !alfanumericoEn(texto, fin) {
			return true
		}
		desde = i + 1
	}
}

// alfanumericoEn dice si en esa posicion hay [a-z0-9]. Fuera de rango es "no
// hay nada", que cuenta como frontera: el principio y el final del texto lo son.
func alfanumericoEn(s string, i int) bool {
	if i < 0 || i >= len(s) {
		return false
	}
	c := s[i]
	return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// sinTildesMinusculas baja a minusculas y quita los diacriticos.
//
// Sin quitar tildes, "revision periodica segun ISO" y la misma frase acentuada
// serian textos distintos para la busqueda, y bastaria escribir el nombre con
// una tilde de mas para saltarse la regla. Se hace a mano, con la tabla del
// castellano, porque nucleo/ no importa nada externo (invariante 5) y
// golang.org/x/text no entra por esto.
func sinTildesMinusculas(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
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
		case 'ñ':
			b.WriteRune('n')
		case 'ç':
			b.WriteRune('c')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
