package pantallas

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/pantalla"
)

// NO SE ABSUELVE CUANDO EL MOTOR NO LEE LA RESPUESTA.
//
// # El defecto, en una frase
//
// La pantalla de alcance pintaba, debajo de una pregunta, «contestar que si aqui
// no activa ninguna obligacion nueva». Ese texto sale de una consecuencia
// calculada que vale cero. Y hay dos ceros distintos que viajan en el mismo int:
//
//	cero HONESTO     el hecho llega al motor y ninguna regla se enciende con el.
//	                 Eso es conocimiento y se dice.
//	cero ESTRUCTURAL el atributo declara corpus.PuenteNoLlegaAlMotor, asi que el
//	                 puente TIRA el hecho antes de que el motor lo vea. El cero no
//	                 puede ser otra cosa jamas, conteste lo que conteste nadie.
//
// El segundo no es «no activa nada»: es «no lo se», y decirlo del otro modo es
// una afirmacion sobre el cumplimiento de alguien hecha por un canal que ni
// siquiera se ha consultado. Es la misma familia que M12 y que el 8 de 72, y el
// godoc de Consecuencias ya la nombraba con estas palabras sin que nadie mirase.
//
// # SENTIDO UNICO, y esta escrito aqui porque importa
//
// Esta puerta PROHIBE ABSOLVER. No exige que la consecuencia coincida con nada.
// La razon es que la otra derivacion de la casa, la de /controles, es
// DELIBERADAMENTE SOBREINCLUSIVA y lo dice en su godoc
// (superficies/pantallas/derivacion.go: «entre dejarla PENDIENTE y darla por
// alcanzada, se elige lo que no absuelve»). Una puerta de igualdad entre las dos
// derivaciones contradiria una decision ya tomada y obligaria a estrechar la que
// esta bien.
//
// # Lo que esta puerta NO mira, dicho al escribirla
//
// Sigue habiendo un cero honesto debajo de un chip que dice «decide N
// obligaciones», y las dos frases conviven en el mismo <li>. No es una
// absolucion falsa (el motor de verdad no enciende nada), es una contradiccion
// de lectura, y su causa esta en el corpus y no aqui: el chip cuenta
// `desbloquea` y lo que se evalua es `Obligacion.Preguntas`, que son dos
// direcciones del mismo enlace y solo una tiene linter. Queda contado en
// docs/pendientes.md con su cardinal.
func TestNoSeAbsuelveCuandoElMotorNoLeeLaRespuesta(t *testing.T) {
	// EL PEOR CASO A PROPOSITO: una calculadora que contesta CERO a todo. Si la
	// pantalla fuera a absolver alguna vez, con esta absuelve siempre.
	//
	// Y esto no es un detalle del arnes: una superficie SIN adaptador de
	// consecuencias no pinta ninguna, o sea que mediria cero absoluciones y
	// daria verde sin haber ejercido nada. La trampa esta contada porque cuesta
	// caer en ella.
	s, _ := superficie(t, corpusReal(t), conConsecuencias(consecuenciasFalsas{n: 0}))
	_, cuerpo := pedir(t, s, "/alcance?"+ParamVer+"="+VerTodas)

	callejones, llegan := preguntasPorPuente(t, corpusReal(t))
	if len(callejones) == 0 {
		t.Fatal("el corpus publicado no trae ni una pregunta cuyo atributo declare el " +
			"callejon, asi que esta puerta estaria prohibiendo algo que no puede pasar.\n" +
			"  Si el corpus ha cerrado todos los callejones, es una buena noticia y hay " +
			"que decirlo aqui, no borrar la comprobacion")
	}
	t.Logf("preguntas de si/no del corpus publicado: %d con callejon declarado, %d que el "+
		"motor si lee", len(callejones), len(llegan))

	// DIRECCION 1, la prohibicion.
	var absueltas []string
	for _, id := range callejones {
		if bloqueDeLaPregunta(t, cuerpo, id) == "" {
			continue
		}
		if strings.Contains(bloqueDeLaPregunta(t, cuerpo, id), claveEnPagina("alcance.consecuencia.ninguna")) {
			absueltas = append(absueltas, id)
		}
	}
	if len(absueltas) > 0 {
		t.Errorf("la pantalla dice que contestar que si a estas preguntas no activa ninguna "+
			"obligacion, y su atributo declara que la respuesta NO llega al motor: %v\n"+
			"  Ese cero no lo ha calculado nadie, lo ha producido el puente al tirar el "+
			"hecho. Es «no lo se» disfrazado de «no pasa nada», en la pantalla donde alguien "+
			"esta decidiendo que declara sobre su organizacion.\n"+
			"  Arreglo: no pintar consecuencia cuando corpus.PreguntaEntrevista.LlegaAlMotor "+
			"es falso. Ni cero ni nada.", absueltas)
	}

	// DIRECCION 2, EL CONTROL POSITIVO. Sin esto, un producto que hubiera dejado
	// de pintar consecuencias del todo aprobaria la direccion 1 con nota, y una
	// rama de descargo que ninguna entrada recorre es una rama que no existe.
	var conAbsolucion int
	for _, id := range llegan {
		if strings.Contains(bloqueDeLaPregunta(t, cuerpo, id), claveEnPagina("alcance.consecuencia.ninguna")) {
			conAbsolucion++
		}
	}
	if conAbsolucion == 0 {
		t.Errorf("con una calculadora que contesta cero a todo, NINGUNA de las %d preguntas "+
			"que el motor si lee pinta la absolucion.\n"+
			"  Entonces la direccion 1 no demuestra nada: estaria comprobando que no aparece "+
			"un texto que no aparece nunca.", len(llegan))
	}
	t.Logf("absoluciones pintadas sobre preguntas que el motor si lee: %d de %d",
		conAbsolucion, len(llegan))
}

// TestElCallejonYElCeroHonestoSeDistinguenEnLaMismaPagina es el control de las
// dos direcciones con dato sintetico, al lado y en la misma peticion.
//
// Existe porque la puerta de arriba depende de que el corpus publicado traiga
// las dos clases de pregunta, y eso puede dejar de ser cierto sin que nadie lo
// decida. Aqui las dos estan puestas a mano y no se pueden ir: si el producto
// dejara de distinguirlas, una de las dos afirmaciones se pone roja.
func TestElCallejonYElCeroHonestoSeDistinguenEnLaMismaPagina(t *testing.T) {
	s, _ := superficie(t, []*corpus.Paquete{paqueteDeLosDosCeros()},
		conConsecuencias(consecuenciasFalsas{n: 0}))
	_, cuerpo := pedir(t, s, "/alcance?"+ParamVer+"="+VerTodas)

	absolucion := claveEnPagina("alcance.consecuencia.ninguna")

	honesto := bloqueDeLaPregunta(t, cuerpo, "dos.q.honesta")
	if honesto == "" {
		t.Fatal("la pregunta con puente no se ha pintado: el caso no se esta ejerciendo")
	}
	if !strings.Contains(honesto, absolucion) {
		t.Errorf("la pregunta cuyo hecho SI llega al motor no pinta la absolucion con una "+
			"calculadora que dice cero.\n  Ese cero es conocimiento y se dice: el motor la "+
			"leyo y ninguna regla se encendio.\n%s", honesto)
	}

	callejon := bloqueDeLaPregunta(t, cuerpo, "dos.q.callejon")
	if callejon == "" {
		t.Fatal("la pregunta del callejon no se ha pintado: el caso no se esta ejerciendo")
	}
	if strings.Contains(callejon, absolucion) {
		t.Errorf("la pregunta cuyo atributo declara el callejon pinta la absolucion.\n"+
			"  Su cero no lo calculo el motor: el puente tiro el hecho antes.\n%s", callejon)
	}
	if strings.Contains(callejon, `class="consecuencia"`) {
		t.Errorf("la pregunta del callejon pinta el bloque de la consecuencia, aunque sea "+
			"vacio. La tercera rama es NO PINTAR NADA:\n%s", callejon)
	}
}

// paqueteDeLosDosCeros trae las dos preguntas booleanas que se distinguen: una
// cuyo atributo declara el puente y otra cuyo atributo declara el callejon.
//
// Las dos desbloquean obligaciones, y eso importa: si la del callejon no
// desbloqueara nada, se dormiria y no se pintaria, y el caso no se ejerceria.
func paqueteDeLosDosCeros() *corpus.Paquete {
	return &corpus.Paquete{
		URN: "urn:demo:dosceros", Version: "2026.1", Clase: corpus.Propio,
		Licencia:       "Apache-2.0",
		Identificador:  corpus.Identificador{Tipo: corpus.ELIUE, Valor: "reg/9999/3/oj"},
		LicenciaFuente: corpus.DelProyecto,
		Atribucion:     "Paquete sintetico de demostracion. Sin tercero con derechos.",
		Vigencia:       corpus.Vigencia{Desde: "2026-01-01"},
		Entidades: []corpus.TipoEntidad{{
			Nombre: "organizacion", Descripcion: "el sujeto obligado",
			Atributos: []corpus.Atributo{
				{Nombre: "honesta", Tipo: corpus.Booleano, Cita: "demo dos art. 1",
					Hecho: &corpus.HechoDeAtributo{
						Forma: corpus.PuenteAfirmaSi, Predicado: "honesta"}},
				{Nombre: "callejon", Tipo: corpus.Booleano, Cita: "demo dos art. 2",
					Hecho: &corpus.HechoDeAtributo{
						Forma:  corpus.PuenteNoLlegaAlMotor,
						Porque: "es un dato del expediente, no del sujeto: no decide a quien alcanza nada"}},
			},
		}},
		Preguntas: []corpus.Pregunta{
			{ID: "dos.q.honesta", Texto: "Una pregunta que el motor si lee",
				Cita: "demo dos art. 1", Entidad: "organizacion", Atributo: "honesta",
				Desbloquea: []string{"dos.o.una"}},
			{ID: "dos.q.callejon", Texto: "Una pregunta que el motor no lee",
				Cita: "demo dos art. 2", Entidad: "organizacion", Atributo: "callejon",
				Desbloquea: []string{"dos.o.otra"}},
		},
		Obligaciones: []corpus.Obligacion{
			{ID: "dos.o.una", Articulo: "1", Cita: "demo dos art. 1",
				Vigencia: corpus.Vigencia{Desde: "2026-01-01"}, ClaseE2E: "documental",
				Preguntas: []string{"dos.q.honesta"}},
			{ID: "dos.o.otra", Articulo: "2", Cita: "demo dos art. 2",
				Vigencia: corpus.Vigencia{Desde: "2026-01-01"}, ClaseE2E: "documental",
				Preguntas: []string{"dos.q.callejon"}},
		},
	}
}

// preguntasPorPuente parte las preguntas de SI/NO del corpus en las dos clases
// que esta puerta distingue.
//
// Solo las de si/no: la consecuencia no se pinta en las que piden un valor, y
// meterlas aqui haria que la direccion 2 contara preguntas que nunca podrian
// pintar nada, o sea que el control positivo pasaria a mirar el vacio.
func preguntasPorPuente(t *testing.T, ps []*corpus.Paquete) (callejones, llegan []string) {
	t.Helper()
	m := pantallasDe(t, ps)
	voc := VocabularioDe(ps, m.porID[pantalla.Alcance].Preguntas)
	for _, q := range corpus.Entrevista(ps) {
		if voc.Tipo(q.ID).PideValor() {
			continue
		}
		if q.LlegaAlMotor {
			llegan = append(llegan, q.ID)
		} else {
			callejones = append(callejones, q.ID)
		}
	}
	sort.Strings(callejones)
	sort.Strings(llegan)
	return callejones, llegan
}

// claveEnPagina compone lo que el catalogo de pruebas imprime para una clave.
//
// Se compone en vez de escribir el texto: un literal del catalogo real se queda
// viejo el dia que alguien mejore la redaccion, y entonces esta puerta pasaria a
// buscar algo que no existe, o sea a dar verde sobre cualquier cosa.
func claveEnPagina(clave string) string { return fmt.Sprintf("[%s:%s]", "es", clave) }

// bloqueDeLaPregunta recorta el <li> de UNA pregunta.
//
// Se corta por el id que la plantilla pone en cada <li> (`id="p-<id>"`), que es
// una identidad y no una posicion: buscar el n-esimo <li> haria que insertar una
// pregunta cambiara de que pregunta habla esta puerta sin tocarla (invariante 7).
func bloqueDeLaPregunta(t *testing.T, cuerpo, id string) string {
	t.Helper()
	marca := `id="p-` + id + `"`
	i := strings.Index(cuerpo, marca)
	if i < 0 {
		return ""
	}
	resto := cuerpo[i:]
	if fin := strings.Index(resto, "</li>"); fin >= 0 {
		return resto[:fin]
	}
	return resto
}
