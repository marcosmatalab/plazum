package evidencia_test

import (
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/busqueda"
	"github.com/marcosmatalab/plazum/adaptadores/evidencia"
	"github.com/marcosmatalab/plazum/adaptadores/ia"
	"github.com/marcosmatalab/plazum/adaptadores/ingesta"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// LA MEDIDA QUE DEJA ABIERTA LA PIEZA 1, y por que se mide aqui y no se
// construye la pieza.
//
// La casilla de la pieza 1 (`ETAPAS.md`) dice que el cliente suelta sus
// documentos y el sistema PROPONE CADA RESPUESTA de la entrevista con su cita.
// El mecanismo evidente es el que ya existe: `Mapear`, con el enunciado de la
// pregunta de consulta en vez del texto de la obligacion. El propio tipo
// `Consulta` esta escrito para eso («el texto es el de la obligacion o el de la
// pregunta de la entrevista») y `MinimoAbsoluto` se midio el 06-09-2026
// pensando en frases de una linea, que es lo que son las preguntas.
//
// # POR QUE NO SE CONSTRUYO, medido y no supuesto
//
// Porque una respuesta de la entrevista decide QUE OBLIGACIONES TE APLICAN, y
// para proponerla hay que afirmar QUE DICE el parrafo, no solo encontrarlo. Y
// eso es exactamente lo que un emparejamiento por bolsa de palabras no puede
// hacer: BM25 no ve la negacion.
//
// Un parrafo que niega es el MEJOR emparejamiento lexico posible para la
// pregunta afirmativa, porque comparte todos sus terminos. El mecanismo es mas
// confiado cuanto mas claramente el documento dice lo contrario, y eso invierte
// la unica proteccion que tenia el diseno: proponer solo el SI es conservador
// cuando el si se apoya en un parrafo que calla, y es lo contrario de
// conservador cuando se apoya en un parrafo que NIEGA. La pantalla enseñaria
// «propuesto: si» con la cita que lo refuta debajo, ya verificada por hash y
// literal. Quien lo lea deja de creerse el resto de la pantalla, y con razon.
//
// # LO QUE ESTE TEST AFIRMA, que es un hecho y no una propiedad deseada
//
// Que sobre dos documentos IDENTICOS salvo en el parrafo que contesta, las
// preguntas que emparejan con el parrafo que AFIRMA emparejan tambien con el que
// NIEGA, y este puntua IGUAL O MAS. No es que el mecanismo no los distinga: los
// ordena al reves, porque negar aporta terminos y BM25 los cuenta como afinidad.
// Un umbral no las separa, y ordenar por confianza pondria las negaciones
// primero. Por eso no se arregla afinando el minimo de aciertos.
//
// NO SE NOMBRA NINGUNA PREGUNTA NI NINGUNA NORMA: el par sale del corpus
// instalado en tiempo de ejecucion (invariante 2, y la puerta de
// `TestNingunaNormaCableada` se cobro la primera version de este fichero por
// escribir un identificador de pregunta en un literal). Eso ademas lo hace mas
// fuerte: la afirmacion es universal sobre lo que el corpus traiga, y no sobre
// un caso que eligio el propio test.
//
// # SU CONTROL POSITIVO, y por que sin el no valdria
//
// Se exige que la rama del SI encuentre al menos una pregunta. Una medida que
// solo recorriera la negacion no demostraria que el mecanismo la lee al reves:
// demostraria que se le puso delante lo que falla. Y si algun dia el
// emparejamiento dejara de encontrar NADA, este test se pondria rojo por el
// control positivo en vez de quedarse verde afirmando lo que ya no mide.
//
// SE PONDRA ROJO EL DIA QUE ALGUIEN LO ARREGLE, y eso es la señal correcta: no
// es una regresion, es el aviso de que la pieza 1 se puede volver a abrir. Ese
// dia se lee `docs/pendientes.md`, que lleva el hueco con su cardinal.
//
// El hueco que ya estaba escrito en el encabezado de este paquete era la
// PARAFRASIS («revision anual» frente a «cada doce meses»), y era el hueco
// benigno: NO ENCONTRAR. Este es el otro lado y es peor: encontrar y leerlo al
// reves.

// alcanceQueAfirma y alcanceQueNiega son el par minimo, y son IDENTICOS salvo el
// parrafo que contesta. Esa es la mitad que hace la medida: si cambiara algo mas,
// la diferencia de puntuacion podria venir de otro sitio.
//
// Son documentos de ALCANCE, que es el genero que un cliente subiria justo para
// contestar esta entrevista, y es tambien el genero donde la negacion es mas
// densa: un alcance existe para delimitar, y delimitar es decir que queda fuera.
const (
	alcanceQueAfirma = `ALCANCE DEL SISTEMA DE GESTION

El alcance cubre la sede central y los servicios prestados desde ella, con el personal propio y el subcontratado.

Se ha designado un delegado de proteccion de datos y sus datos de contacto estan publicados en la web corporativa y comunicados a la autoridad de control.

Se mantiene un registro de actividades de tratamiento con la finalidad, la base juridica y los plazos de conservacion de cada tratamiento.`

	alcanceQueNiega = `ALCANCE DEL SISTEMA DE GESTION

El alcance cubre la sede central y los servicios prestados desde ella, con el personal propio y el subcontratado.

La organizacion no ha designado delegado de proteccion de datos por no concurrir ninguno de los supuestos del articulo 37.

Se mantiene un registro de actividades de tratamiento con la finalidad, la base juridica y los plazos de conservacion de cada tratamiento.`
)

// loQueCambia es lo unico que distingue a los dos documentos. Se usa para
// comprobar que el emparejamiento cayo DE VERDAD en el parrafo que contesta y no
// en otro, que es lo que haria que la comparacion de puntuaciones no significara
// nada. No es contenido normativo: son las palabras del documento de un cliente.
const loQueCambia = "designado"

func TestElEmparejamientoLexicoNoDistingueElSiDelNo(t *testing.T) {
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatal(err)
	}

	// LAS PREGUNTAS SALEN DEL CORPUS, todas, sin elegir. Una consulta escrita al
	// lado del test seria una consulta que eligio el test.
	var cs []evidencia.Consulta
	for _, q := range corpus.Entrevista(ps) {
		cs = append(cs, evidencia.Consulta{ID: q.ID, Texto: q.Texto})
	}
	if len(cs) < 20 {
		t.Fatalf("solo %d preguntas en la entrevista: el corpus no se ha cargado entero "+
			"y esta medida no estaria midiendo nada", len(cs))
	}

	afirma := porConsulta(mapearContra(t, alcanceQueAfirma, cs))
	niega := porConsulta(mapearContra(t, alcanceQueNiega, cs))

	// EL EMPAREJAMIENTO ENTRE LOS DOS LADOS CASA POR ConsultaID, que es identidad
	// y no posicion (invariante 7). Casar por indice haria que una pregunta que
	// aparece en un lado y no en el otro desplazara todas las demas.
	comparadas := 0
	for id, a := range afirma {
		n, hay := niega[id]
		if !hay {
			continue
		}
		// Las dos citas tienen que ser el parrafo que cambia. Si el
		// emparejamiento cayo en otro sitio, comparar sus puntuaciones no dice
		// nada sobre la negacion.
		if !strings.Contains(a.Cita, loQueCambia) || !strings.Contains(n.Cita, loQueCambia) {
			continue
		}
		comparadas++

		// LA CITA, LITERAL Y DEL CLIENTE. La puerta antialucinacion sigue
		// haciendo su trabajo, y esa es justo la parte inquietante: lo que falla
		// no es la cita, es lo que se deduciria de ella.
		if !strings.Contains(alcanceQueNiega, n.Cita) {
			t.Errorf("%s: la cita no esta literal en el documento del cliente: %q", id, n.Cita)
		}

		// EL HECHO QUE SOSTIENE LA PARADA. Se afirma con >= y no con la igualdad
		// medida hoy: lo que bloquea la casilla es que la negacion no puntue POR
		// DEBAJO, que es lo unico que permitiria separarlas con un umbral. El
		// numero exacto se imprime y no se cablea, porque depende del par de
		// documentos y no es el hecho.
		if n.Puntuacion < a.Puntuacion {
			t.Errorf("%s: la puntuacion del parrafo que NIEGA (%.2f) ha bajado por debajo "+
				"de la del que AFIRMA (%.2f). Eso seria una señal utilizable y la pieza 1 "+
				"se puede volver a abrir: lee el hueco en docs/pendientes.md antes de "+
				"tocar este test", id, n.Puntuacion, a.Puntuacion)
		}
		t.Logf("%s (%q):\n  AFIRMA [%d aciertos, %.2f] %s\n  NIEGA  [%d aciertos, %.2f] %s",
			id, textoDe(cs, id),
			a.Aciertos, a.Puntuacion, a.Cita,
			n.Aciertos, n.Puntuacion, n.Cita)
	}

	// CONTROL POSITIVO. Sin al menos un par comparado, todo lo de arriba es un
	// bucle que no se ejecuta, y un descargo que ninguna entrada alcanza es un
	// descargo que no existe.
	if comparadas == 0 {
		t.Fatal("ni una sola pregunta del corpus emparejo con el parrafo que cambia en " +
			"LOS DOS documentos.\nEsta medida no ha comparado nada, asi que no dice nada: " +
			"o el emparejamiento dejo de encontrar (y entonces la pieza 1 cambia de motivo) " +
			"o el par de documentos ya no ejerce el corpus instalado")
	}
	t.Logf("%d pregunta(s) del corpus emparejan con el parrafo que contesta en los dos "+
		"documentos, y en todas la NEGACION puntua igual o mas que la AFIRMACION. "+
		"Proponer el SI desde aqui propondria lo contrario del documento, con la cita que "+
		"lo refuta debajo. Es el motivo de que la pieza 1 siga abierta", comparadas)
}

// porConsulta indexa los hallazgos por el ID de su consulta, que es por donde se
// emparejan los dos lados.
func porConsulta(hs []evidencia.Hallazgo) map[string]evidencia.Hallazgo {
	out := make(map[string]evidencia.Hallazgo, len(hs))
	for _, h := range hs {
		out[h.ConsultaID] = h
	}
	return out
}

// textoDe recupera el enunciado de una consulta para poder imprimirlo.
func textoDe(cs []evidencia.Consulta, id string) string {
	for _, c := range cs {
		if c.ID == id {
			return c.Texto
		}
	}
	return ""
}

// mapearContra monta la cadena entera (ingesta, fuentes, indice, verificador) y
// mapea las consultas contra UN documento del cliente.
func mapearContra(t *testing.T, texto string, cs []evidencia.Consulta) []evidencia.Hallazgo {
	t.Helper()
	datos := []byte(texto)
	doc, err := ingesta.Leer("alcance.txt", datos)
	if err != nil {
		t.Fatal(err)
	}
	fs, err := ia.FuentesAportadas(ia.Huella(datos), doc)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := busqueda.Nuevo(ia.Documentos(fs))
	if err != nil {
		t.Fatal(err)
	}
	v, err := ia.Nuevo(ia.Opciones{
		Fuentes:    fs,
		Admite:     []ia.Procedencia{ia.Aportado},
		MinimoCita: ia.MinimoCitaPorDefecto,
	})
	if err != nil {
		t.Fatal(err)
	}
	hs, err := evidencia.Mapear(idx, v, cs)
	if err != nil {
		t.Fatal(err)
	}
	return hs
}
