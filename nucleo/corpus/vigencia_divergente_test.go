package corpus

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// Las lecturas divergentes de VIGENCIA.
//
// El mecanismo de divergencias existia solo para el COMPUTO (dos cifras para el
// mismo plazo y la norma no dice cual rige: el art. 14.2.a del CRA, la tabla 3
// del RD 43/2021). El 27-08-2026 aparecio la otra mitad, que es la discrepancia
// sobre DESDE CUANDO obliga algo, y es peor: un plazo mal leido da una fecha
// equivocada, una vigencia mal leida hace que la obligacion entera aparezca o
// desaparezca de la lista.
//
// LO QUE ESTE FICHERO DEFIENDE, y es una sola cosa: **una lectura divergente no
// cambia NUNCA lo que vincula**. Manda lo publicado. Si un dia una alternativa
// moviera VigenteEn, un acuerdo politico sin publicar en el DOUE se habria
// convertido en derecho aplicado dentro del producto, y ese es exactamente el
// fallo que un GRC no se puede permitir: enseñar como exigible algo que no lo es.

func fecha(t *testing.T, s string) time.Time {
	t.Helper()
	x, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatal(err)
	}
	return x
}

// LAS DOS FORMAS DE LA NADA, que es el invariante 8. Un test de "sin
// alternativas" que solo recorra `nil` deja sin mirar el vacio-presente, y al
// reves. Aqui las dos tienen que dar el MISMO resultado que la tercera fila, que
// es la que si trae una alternativa capaz de cambiar la respuesta.
func TestUnaLecturaDivergenteDeVigenciaNoCambiaLoQueVincula(t *testing.T) {
	// El caso real: el art. 73 del AI Act aplica desde el 02-08-2026 segun el
	// DOUE, y un acuerdo politico sin publicar lo llevaria a diciembre de 2027.
	// El 27-08-2026 la respuesta correcta es SI, esta vigente.
	const publicada = "2026-08-02"
	const aplazada = "2027-12-02"
	momento := fecha(t, "2026-08-27")

	casos := []struct {
		nombre string
		v      Vigencia
	}{
		{"sin alternativas: nil", Vigencia{Desde: publicada}},
		{"sin alternativas: vacio-presente", Vigencia{Desde: publicada,
			Alternativas: []LecturaVigencia{}}},
		{"con una alternativa que aplazaria la obligacion", Vigencia{Desde: publicada,
			Alternativas: []LecturaVigencia{{
				ID: "digital-omnibus", Desde: aplazada,
				Cita: "acuerdo politico de 05-2026, sin publicar en el DOUE",
			}}}},
		{"con una alternativa que la adelantaria", Vigencia{Desde: publicada,
			Alternativas: []LecturaVigencia{{
				ID: "lectura-adelantada", Desde: "2025-01-01", Cita: "c",
			}}}},
		{"con una alternativa que la habria derogado", Vigencia{Desde: publicada,
			Alternativas: []LecturaVigencia{{
				ID: "lectura-derogada", Desde: publicada, Hasta: "2026-08-10", Cita: "c",
			}}}},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			ok, err := c.v.VigenteEn(momento)
			if err != nil {
				t.Fatalf("%v", err)
			}
			if !ok {
				t.Errorf("la vigencia declarada dice desde %s y se pregunta por %s: tenia que "+
					"estar vigente. Si esto falla, una lectura divergente ha entrado en el "+
					"calculo, y entonces lo que el producto ensena como exigible sale de algo "+
					"que no esta publicado", publicada, momento.Format("2006-01-02"))
			}
		})
	}
}

// Y la direccion contraria, que es la que de verdad se olvida: la lectura
// divergente TAMPOCO puede rescatar una vigencia que aun no ha empezado.
func TestUnaLecturaDivergenteNoAdelantaLoQueTodaviaNoObliga(t *testing.T) {
	v := Vigencia{
		Desde: "2027-08-02",
		Alternativas: []LecturaVigencia{{
			ID: "lectura-adelantada", Desde: "2025-01-01", Cita: "c",
		}},
	}
	ok, err := v.VigenteEn(fecha(t, "2026-08-27"))
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("la vigencia declarada empieza en 2027 y el motor la da por vigente en 2026: " +
			"la alternativa ha entrado en el calculo por el otro lado")
	}
}

// El linter rechaza toda lectura que no se pueda defender sola, porque una
// lectura divergente se le ENSENA al cliente al lado de la fecha que vincula.
func TestElLinterRechazaLecturasDeVigenciaQueNoSeSostienen(t *testing.T) {
	casos := []struct {
		nombre  string
		lectura LecturaVigencia
		quiero  error
		texto   string // cuando no hay centinela, un trozo del mensaje
	}{
		{"sin cita no es una lectura, es una opinion",
			LecturaVigencia{ID: "l", Desde: "2027-12-02"},
			ErrLecturaVigenciaSinCita, ""},
		{"sin desde ni hasta no discrepa de nada",
			LecturaVigencia{ID: "l", Cita: "c"},
			ErrLecturaVigenciaVacia, ""},
		{"identica a la declarada se lee como un desacuerdo que no existe",
			LecturaVigencia{ID: "l", Desde: "2022-05-05", Cita: "c"},
			ErrLecturaVigenciaQueNoDiverge, ""},
		{"fecha ilegible",
			LecturaVigencia{ID: "l", Desde: "diciembre de 2027", Cita: "c"},
			ErrVigenciaIlegible, ""},
		{"invertida",
			LecturaVigencia{ID: "l", Desde: "2028-01-01", Hasta: "2027-01-01", Cita: "c"},
			ErrVigenciaInvertida, ""},
		{"sin id no se puede nombrar donde se ensene",
			LecturaVigencia{Desde: "2027-12-02", Cita: "c"},
			nil, "no tiene id"},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			p := base()
			p.Obligaciones[0].Vigencia.Alternativas = []LecturaVigencia{c.lectura}
			errs := p.Validar()
			if len(errs) == 0 {
				t.Fatalf("el paquete ha cargado con la lectura %+v dentro", c.lectura)
			}
			for _, e := range errs {
				if c.quiero != nil && errors.Is(e, c.quiero) {
					return
				}
				if c.texto != "" && strings.Contains(e.Error(), c.texto) {
					return
				}
			}
			t.Errorf("ninguno de los %d errores es el que tocaba: %v", len(errs), errs)
		})
	}
}

// CONTROL NEGATIVO. Sin esto, los seis casos de arriba estarian igual de verdes
// si el linter rechazara cualquier alternativa, incluida la buena, y el
// mecanismo no serviria para nada.
func TestUnaLecturaDivergenteBienEscritaCarga(t *testing.T) {
	p := base()
	p.Obligaciones[0].Vigencia.Alternativas = []LecturaVigencia{{
		ID:    "digital-omnibus",
		Desde: "2027-12-02",
		Cita:  "acuerdo politico de 05-2026 sobre el paquete omnibus digital, sin publicar en el DOUE",
	}}
	if errs := p.Validar(); len(errs) != 0 {
		t.Fatalf("una lectura divergente bien escrita no carga: %v", errs)
	}
	// Y la misma en la cabecera del paquete, que es el otro sitio del formato
	// donde vive una Vigencia: si el linter solo mirara las obligaciones, el
	// paquete entero podria declarar lo que quisiera.
	q := base()
	q.Vigencia.Alternativas = []LecturaVigencia{{ID: "l", Cita: "c"}} // sin fecha: tiene que doler
	errs := q.Validar()
	encontrado := false
	for _, e := range errs {
		if errors.Is(e, ErrLecturaVigenciaVacia) {
			encontrado = true
		}
	}
	if !encontrado {
		t.Errorf("la vigencia de CABECERA no pasa por el mismo linter que la de las "+
			"obligaciones: %v", errs)
	}
}
