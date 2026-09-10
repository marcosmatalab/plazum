package corpus

import "sort"

// LAS VERSIONES LINGUISTICAS DE UN ARTICULO (D-25).
//
// Vive en su propio fichero y no en `paquete.go` por el techo de lineas que
// vigila `TestLosPaquetesGrandesTienenTechoYNoSorpresa`: meterlo alli dejaba el
// fichero en 2704 sobre un techo de 2700. Subir el techo habria sido la salida
// barata; el techo existe justo para que un fichero no crezca sin que nadie lo
// decida, y aqui la decision correcta es que este formato tiene sitio propio.

// VersionLinguistica es el mismo articulo publicado en otra lengua.
//
// # SE LLAMA ASI Y NO «TRADUCCION», Y LA PALABRA ES LA DECISION
//
// En la UE todas las versiones linguisticas de un acto son AUTENTICAS: la
// inglesa no es una traduccion de la castellana, es el mismo acto publicado en
// otra lengua. Llamarlo traduccion invitaria a lo que D-11 prohibe, que es que
// la escribamos nosotros.
//
// # NO TIENE NI UN CAMPO QUE AFECTE AL RELOJ, y eso es el tipo haciendo su
// trabajo
//
// Cuatro campos y los cuatro son de texto o de procedencia. Anadir aqui un
// `Temporalidad`, un `Escalado` o una `Vigencia` es lo que la puerta de arriba
// impide, y es tambien lo que haria falta para que dos idiomas dieran dos
// fechas del mismo articulo.
type VersionLinguistica struct {
	// Texto es el literal de la version oficial. Nunca lo escribimos nosotros.
	Texto string `json:"texto"`
	// Enlace apunta a ESA version concreta, no al acto en general: un enlace al
	// acto no permite comprobar de que lengua salio el texto.
	//
	// SE LLAMA `enlace` Y NO `fuente` A PROPOSITO. `fuente` es un nombre RETIRADO
	// del formato de paquete, y hay una puerta que recorre los BYTES de cada
	// paquete.json para que no vuelva (TestNingunPaquetePublicadoGuardaUnaDireccion
	// ComoIdentificador). La primera version de este campo se llamaba asi y la
	// puso roja: reusar el nombre habria revivido una confusion que el proyecto
	// ya cerro, y ademas habria dejado a esa puerta acusando a un campo legitimo,
	// que es como se acaba aflojando una guarda buena.
	Enlace string `json:"enlace"`
	// Celex identifica el acto en EUR-Lex. Va en cada version porque el
	// invariante 10 se comprueba por obligacion y no por paquete.
	Celex string `json:"celex"`
	// Consultado es el dia en que se miro la fuente, en AAAA-MM-DD.
	//
	// Es la mitad del invariante 10 que se olvida: un dato correcto que nadie
	// puede reauditar es indistinguible de un acierto por suerte.
	Consultado string `json:"consultado"`
}

// lenguasOrdenadas devuelve las etiquetas de lengua en orden estable.
//
// No reusa `clavesOrdenadas` (cadencia_gemela.go) porque aquella es sobre
// map[string]bool: son dos tipos y en Go eso son dos funciones.
func lenguasOrdenadas(m map[string]VersionLinguistica) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TextoEnLengua devuelve el texto legal de esta obligacion en la lengua pedida,
// y dice SI ES LA QUE SE PIDIO.
//
// # LOS DOS VALORES, y el segundo es el que importa
//
// Devolver solo el texto obligaria a quien llama a comparar cadenas para saber
// si le dieron lo que pidio, y nadie lo hace. Con el bool, la superficie puede
// pintar `aviso.idioma_del_corpus` EXACTAMENTE donde hace falta: al lado de un
// texto que esta en otra lengua que la pagina.
//
// Caer al castellano es lo correcto —un texto en la lengua equivocada es mejor
// que ningun texto, porque el corpus es lo que el cliente ha venido a leer— pero
// caer EN SILENCIO no lo es: quien lee la pagina inglesa tiene que saber que ese
// parrafo no lo esta.
//
// LA LENGUA VACIA DEVUELVE EL CASTELLANO CON hay=false, y no es un detalle: el
// valor cero de una peticion sin idioma no puede leerse como «me diste lo que
// pedi», porque no se pidio nada.
//
// LO VIGILA: TestElTextoEnOtraLenguaDiceSiEsLaQueSePidio, con las dos ramas y
// las tres formas de la nada.
func (o Obligacion) TextoEnLengua(lengua string) (texto string, hay bool) {
	if lengua == "" || lengua == LenguaDelTextoLegal {
		return o.TextoLegal, lengua == LenguaDelTextoLegal
	}
	if v, ok := o.VersionesLinguisticas[lengua]; ok && v.Texto != "" {
		return v.Texto, true
	}
	// Presente y sin version: se cae al castellano DICIENDOLO.
	return o.TextoLegal, false
}

// LenguaDelTextoLegal es la etiqueta de la lengua en la que esta `TextoLegal`.
//
// Es castellano en todo el corpus de hoy, incluidos los marcos de origen
// anglosajon, y por eso es una constante y no un campo: un campo por paquete
// invitaria a declarar una lengua distinta sin traer el texto en esa lengua, y
// entonces `TextoEnLengua` mentiria en su segundo valor.
//
// El dia que un paquete nazca con su texto legal en otra lengua, esto pasa a ser
// un campo del paquete Y `TextoEnLengua` deja de poder compararlo con una
// constante. Es un cambio de formato, no un ajuste, y por eso esta escrito.
const LenguaDelTextoLegal = "es"
