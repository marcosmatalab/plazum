package acta

import (
	"embed"
	"io/fs"

	"github.com/marcosmatalab/plazum/nucleo/acta"
	"github.com/marcosmatalab/plazum/puertos"
)

//go:embed plantillas
var plantillasFS embed.FS

// Plantillas expone el sistema de ficheros embebido, por si alguien quiere
// montar otro motor de render.
func Plantillas() fs.FS { return plantillasFS }

// LA REGLA DEL FICHERO, y aqui es distinta de la de las demas superficies.
//
// En las otras pantallas los ROTULOS son claves de catalogo y el CONTENIDO viaja
// tal cual. Aqui el contenido lo escribe plazum, no el cliente ni el corpus, asi
// que TAMBIEN se traduce: los rotulos de cubo, los de reparto y los catorce
// descargos llegan ya traducidos desde el catalogo, con la clave que declara
// nucleo/acta. Lo que sigue viajando tal cual es lo que no es de plazum: la
// identidad de un elemento, la nota que lleva dentro las palabras de una persona
// y la cita de un paquete.
//
// La traduccion se resuelve EN GO y no en la plantilla porque varias frases
// llevan huecos que rellena un dato, y html/template no sabe desplegar una lista
// de argumentos.

// Vista es lo unico que ve la plantilla.
type Vista struct {
	Idioma   string
	Base     string
	Estatico string
	// Titulo es la CLAVE de catalogo.
	Titulo string
	// Aviso es texto ya resuelto (viene de un error).
	Aviso string

	// Los tres estados que no son "aqui esta el acta".
	SinSesion bool
	SinActa   bool
	NoExiste  bool

	Subtitulo    string
	Organizacion string
	Periodo      string
	ID           string

	Cabecera    []ParrafoVista
	Calibra     []FuenteVista
	Descuadres  []string
	Secciones   []SeccionVista
	Prosa       []MontonVista
	NoHayCuarta string
	// Los titulos que declara el NUCLEO llegan ya traducidos, igual que los
	// cubos: son palabras del documento, no del marco. Asi la plantilla solo
	// nombra con `t` claves de ESTA superficie, y eso se puede exigir con un test.
	TituloQuePuedeDecir string
	TituloProcedencias  string

	// Derivacion, cuando la peticion es la de una cifra abierta.
	Derivacion *DerivacionVista
}

// ParrafoVista es prosa con su atribucion.
type ParrafoVista struct {
	Texto string
	// Quien y Cita van vacios menos cuando la prosa es de alguien. Que el
	// documento diga de quien es cada frase es la mitad de su valor.
	Quien string
	Cita  string
}

// FuenteVista es una linea del bloque "que puede y que no puede decir".
type FuenteVista struct {
	Nombre string
	Hay    bool
	PorQue string
}

// SeccionVista es un bloque del acta.
type SeccionVista struct {
	Nombre   string
	Aportada bool
	PorQue   string
	Parrafos []ParrafoVista
	Repartos []RepartoVista
}

// RepartoVista es una particion con sus cubos.
type RepartoVista struct {
	Rotulo   string
	Universo int
	Cuadra   bool
	Suma     int
	Cifras   []CifraVista
}

// CifraVista es un numero CON EL ENLACE A SU DERIVACION.
//
// El enlace no es un adorno: es la puerta D11-c. Una cifra sin enlace obliga a
// fiarse, y este documento existe justamente para no tener que fiarse.
type CifraVista struct {
	Ref      string
	Cubo     string
	Valor    int
	Descargo string
	URL      string
}

// MontonVista es una fila del bloque de procedencias.
type MontonVista struct {
	Procedencia string
	N           int
}

// DerivacionVista es una cifra abierta: la lista entera de lo que la compone.
type DerivacionVista struct {
	Ref      string
	Fuente   string
	Reparto  string
	Cubo     string
	Valor    int
	Descargo string
	// Elementos van TAL CUAL: la identidad de una fila y la nota que la
	// acompana no son de plazum, asi que no se traducen.
	Elementos []acta.Elemento
	Volver    string
}

// rellenarCon traduce el acta a la vista.
func (v *Vista) rellenarCon(a acta.Acta, idioma string, c puertos.Catalogo, base string) {
	v.Subtitulo = c.Traducir(idioma, "acta.titulo.subtitulo")
	v.Organizacion = a.Organizacion
	v.Periodo = a.Periodo.String()
	v.ID = a.ID
	v.NoHayCuarta = c.Traducir(idioma, "acta.parrafo.no_hay_cuarta")
	v.TituloQuePuedeDecir = c.Traducir(idioma, "acta.titulo.que_puede_decir")
	v.TituloProcedencias = c.Traducir(idioma, "acta.titulo.procedencias")
	v.Descuadres = a.Descuadres()

	for _, p := range a.Cabecera {
		v.Cabecera = append(v.Cabecera, parrafoVista(p, idioma, c))
	}
	for _, s := range a.Secciones {
		v.Calibra = append(v.Calibra, FuenteVista{
			Nombre: c.Traducir(idioma, s.Fuente.Clave().Clave),
			Hay:    s.Aportada,
			PorQue: s.PorQueFalta,
		})
	}
	// LA REFERENCIA DE CADA CIFRA LA DA EL NUCLEO, no se recalcula aqui.
	//
	// Se indexa por identidad (fuente + clave del reparto + clave del cubo), que
	// son valores de vocabulario cerrado y unicos dentro de su sitio, NUNCA por
	// la posicion en la lista (invariante 7). Recalcular la formula aqui habria
	// sido el mismo error con otra cara: el dia que el nucleo cambiara como
	// numera, el papel y la pantalla numerarian distinto y un consejero que
	// leyera "[1.2.3]" impreso abriria otra cosa en el navegador.
	refs := map[string]string{}
	for _, cs := range a.Cifras() {
		refs[claveDeSitio(cs.Fuente, cs.Reparto.Clave, cs.Cifra.Cubo.Clave)] = cs.Ref
	}

	for _, s := range a.Secciones {
		sv := SeccionVista{
			Nombre:   c.Traducir(idioma, s.Fuente.Clave().Clave),
			Aportada: s.Aportada,
			PorQue:   s.PorQueFalta,
		}
		for _, p := range s.Parrafos {
			sv.Parrafos = append(sv.Parrafos, parrafoVista(p, idioma, c))
		}
		for _, r := range s.Repartos {
			rv := RepartoVista{
				Rotulo:   c.Traducir(idioma, r.Rotulo.Clave),
				Universo: r.Universo,
				Cuadra:   r.Cuadra(),
				Suma:     r.Suma(),
			}
			for _, cf := range r.Cifras {
				ref := refs[claveDeSitio(s.Fuente, r.Rotulo.Clave, cf.Cubo.Clave)]
				cv := CifraVista{
					Ref:   ref,
					Cubo:  c.Traducir(idioma, cf.Cubo.Clave),
					Valor: cf.Valor(),
					URL:   base + "/derivacion/" + ref,
				}
				// EL DESCARGO SOLO CUANDO EL CUBO TIENE ALGO DENTRO. Una frase
				// que sale siempre deja de leerse, y entonces no protege de nada.
				if cf.Valor() > 0 && !cf.Descargo.Vacia() {
					cv.Descargo = c.Traducir(idioma, cf.Descargo.Clave)
				}
				rv.Cifras = append(rv.Cifras, cv)
			}
			sv.Repartos = append(sv.Repartos, rv)
		}
		v.Secciones = append(v.Secciones, sv)
	}
	prosa := a.Prosa()
	for _, p := range acta.ProcedenciasPosibles() {
		v.Prosa = append(v.Prosa, MontonVista{
			Procedencia: c.Traducir(idioma, p.Clave().Clave),
			N:           len(prosa[p]),
		})
	}
}

// parrafoVista traduce un parrafo SEGUN SU PROCEDENCIA, que es la regla entera.
//
// Lo que escribe plazum lleva clave y se traduce, con sus datos rellenando los
// huecos. Lo que escribe una persona y lo que dice una norma NO llevan clave y
// viajan tal cual: traducir el texto de una norma crea obra derivada, y traducir
// la conclusion de un consejo es ponerle palabras al consejo.
func parrafoVista(p acta.Parrafo, idioma string, c puertos.Catalogo) ParrafoVista {
	if p.Clave == "" {
		return ParrafoVista{Texto: p.Texto, Quien: p.Quien, Cita: p.Cita}
	}
	args := make([]any, len(p.Args))
	for i, a := range p.Args {
		args[i] = a
	}
	return ParrafoVista{Texto: c.Traducir(idioma, p.Clave, args...)}
}

func derivacionVista(cs acta.CifraSituada, idioma string, c puertos.Catalogo, base string) *DerivacionVista {
	d := &DerivacionVista{
		Ref:       cs.Ref,
		Fuente:    c.Traducir(idioma, cs.Fuente.Clave().Clave),
		Reparto:   c.Traducir(idioma, cs.Reparto.Clave),
		Cubo:      c.Traducir(idioma, cs.Cifra.Cubo.Clave),
		Valor:     cs.Cifra.Valor(),
		Elementos: cs.Cifra.Elementos,
		Volver:    base + "/",
	}
	if cs.Cifra.Valor() > 0 && !cs.Cifra.Descargo.Vacia() {
		d.Descargo = c.Traducir(idioma, cs.Cifra.Descargo.Clave)
	}
	return d
}

// claveDeSitio identifica una cifra por lo que es y no por donde esta.
func claveDeSitio(f acta.Fuente, reparto, cubo string) string {
	return string(f) + "|" + reparto + "|" + cubo
}
