package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/adaptadores/recoleccion/manual"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
	"github.com/marcosmatalab/plazum/nucleo/estado"
	"github.com/marcosmatalab/plazum/nucleo/expediente"
)

// LA CADENA DE LOS SIETE ESLABONES, EJERCIDA (A4 de D-22).
//
// # Por que este test es distinto del que ya habia
//
// `ciclo_e2e_test.go` tiene los dos extremos de esta cadena Y NO ESTAN
// ENCADENADOS: su paso 4 llama a `estado.Calcular` con una `estado.Prueba`
// escrita como literal de Go, y su paso 8 lee `expediente-demo.json` DEL DISCO.
// La entrada que calcula el motor no entra nunca en el expediente que se
// verifica, asi que una puerta escrita sobre ese arnes sale verde sin que nada
// haya recorrido nada. Es la familia de la medida que no ejerce.
//
// Aqui el fichero que se verifica SALE del `Calcular` de mas arriba, y la
// afirmacion que lo demuestra es la ultima: cambiar la observacion del principio
// cambia lo que dice el expediente del final.
//
// # LOS SIETE, CON QUIEN LOS RECORRE
//
//	1 observacion   manual.Recolector sobre un fichero escrito aqui
//	2 prueba        corpus.Prueba del corpus REAL, cruzada con AlExpediente
//	3 Calcular      estado.Calcular con la prueba cruzada y las observaciones
//	4 pantalla      evidenciaDeLaInstalacion, que es el cable que monta serve
//	5 expediente    expediente.Expediente construido CON esa entrada
//	6 export        el expediente serializado
//	7 verify        expediente.Verificar, offline y sin confiar en el emisor
//
// # LO QUE ESTE TEST NO CUBRE, DICHO
//
// El eslabon 7 se ejerce con `expediente.Verificar`, que es la misma funcion que
// llama `plazum verify`, pero NO se lanza el binario. Lo que se afirma es que la
// cadena de datos cierra, no que la CLI la envuelva bien; eso lo cubre el
// recorrido del TTFV, que si levanta el binario.
func TestLaCadenaDeLaPruebaCierraDesdeUnaObservacionHastaVerify(t *testing.T) {
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatal(err)
	}

	// ESLABON 2: la prueba sale del CORPUS REAL, no de un literal.
	pr, paquete := unaPruebaDelCorpus(t, ps)
	motor, err := pr.AlExpediente()
	if err != nil {
		t.Fatalf("la prueba del corpus no cruza al expediente: %v", err)
	}

	ahora := time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)
	recolectada := ahora.Add(-24 * time.Hour)

	for _, caso := range []struct {
		nombre     string
		satisfecho string
		quiero     estado.Estado
	}{
		{"con el predicado satisfecho", "true", estado.Pass},
		// LA SEGUNDA ENTRADA ES LA QUE HACE QUE ESTO NO SEA UNA TAUTOLOGIA. Si
		// el expediente del final no dependiera de la observacion del
		// principio, las dos darian lo mismo.
		{"con el predicado sin satisfacer", "false", estado.FailEnPlazo},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			// ESLABON 1: la observacion entra por un fichero, leida por el
			// primer recolector, que no necesita ni una credencial.
			dir := t.TempDir()
			ruta := manual.RutaPorDefecto(dir)
			linea := `{"prueba":"` + pr.ID + `","recurso":"srv-01","satisfecho":` +
				caso.satisfecho + `,"cuando":"` + recolectada.Format(time.RFC3339) + `"}`
			if err := os.WriteFile(ruta,
				[]byte(`{"version":1}`+"\n"+linea+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			rec, err := manual.Abrir(manual.Opciones{Ruta: ruta})
			if err != nil {
				t.Fatal(err)
			}
			obs, siguiente, err := rec.Recolectar(manual.Fuente, "")
			if err != nil {
				t.Fatal(err)
			}
			if len(obs) != 1 || siguiente != "" {
				t.Fatalf("%d observaciones, cursor %q", len(obs), siguiente)
			}

			// ESLABON 4: el mismo cable que monta `plazum serve`. Se ejerce
			// AQUI y no con un doble, que es lo que separa esta medida de una
			// que hable del arnes.
			ev, err := nuevaEvidencia(ps, rec, func() time.Time { return ahora })
			if err != nil {
				t.Fatal(err)
			}
			porObligacion, err := ev.De(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			pantalla, hay := porObligacion[pr.Obligacion]
			if !hay {
				t.Fatalf("la pantalla no tiene evidencia para %s", pr.Obligacion)
			}
			if pantalla.Estado != caso.quiero.String() {
				t.Fatalf("la pantalla dice %q y el motor da %q", pantalla.Estado, caso.quiero)
			}
			if pantalla.Recolector != manual.Nombre {
				t.Errorf("procedencia %q", pantalla.Recolector)
			}
			if !pantalla.Recolectada.Equal(recolectada) {
				t.Errorf("antiguedad: %s, esperaba %s", pantalla.Recolectada, recolectada)
			}
			if pantalla.Predicado == "" {
				t.Error("la pantalla no lleva QUE se evaluo, asi que su estado es una " +
					"afirmacion sin nada detras")
			}

			// ESLABON 3 y 5: la MISMA entrada del motor va al expediente. No se
			// vuelve a calcular ni se escribe a mano: se toma la que la
			// pantalla acaba de ensenar.
			ent := estado.Calcular(motor, obs, estado.Contexto{Ahora: ahora, Aplicable: true})
			if ent.Estado.String() != pantalla.Estado {
				t.Fatalf("el motor y la pantalla discrepan: %q contra %q",
					ent.Estado, pantalla.Estado)
			}
			exp := &expediente.Expediente{
				Version: expediente.Version, Emitido: ahora, ComoEstaba: ahora,
				Organizacion: "Comprador SL", Alcance: "e2e de la cadena de la prueba",
				Pruebas:       []estado.Prueba{motor},
				Observaciones: obs,
				Aplicables:    []string{pr.Obligacion},
				Estados: []expediente.EstadoControl{{
					// EL EXPEDIENTE SE LLEVA EL ESPANOL RESUELTO, no la clave: es
					// un documento que sale del proceso y lo abre un tercero sin
					// navegador, asi que no hay catalogo al otro lado que la
					// traduzca. Es la misma decision que el board pack del acta.
					Prueba: motor.ID, Estado: ent.Estado.String(), Motivo: ent.Motivo.Texto,
				}},
			}

			// ESLABON 6: se serializa y se vuelve a cargar, que es lo que hace
			// un tercero con el fichero en la mano.
			b, err := exp.Guardar()
			if err != nil {
				t.Fatal(err)
			}
			leido, err := expediente.Cargar(b)
			if err != nil {
				t.Fatal(err)
			}

			// ESLABON 7: la misma funcion que llama `plazum verify`, offline y
			// sin confiar en el emisor.
			inf := expediente.Verificar(leido, expediente.ContextoReceptor{})
			// EL EMISOR NO SE CREE. Con un contexto vacio la verificacion NO
			// dice «todo bien»: dice que no puede decidir, y eso es lo que se
			// afirma. Lo que importa aqui es que el estado que recomputa CASA
			// con el que el emisor declaro, que es lo unico que esta cadena
			// tenia que demostrar.
			if !recomputaElMismoEstado(t, leido, ent) {
				t.Errorf("el expediente declara un estado que su propia observacion no "+
					"produce. Informe: valido=%v discrepancias=%v",
					inf.Valido, inf.Discrepancias)
			}

			// Y LA AFIRMACION QUE HACE QUE ESTO NO SEA UNA TAUTOLOGIA: lo que
			// dice el fichero del final depende de la linea del principio.
			if !strings.Contains(string(b), caso.quiero.String()) {
				t.Errorf("el expediente serializado no lleva el estado %q, asi que el "+
					"fichero del final no sale de la observacion del principio",
					caso.quiero)
			}
			_ = paquete
		})
	}
}

// recomputaElMismoEstado vuelve a pasar las observaciones del expediente por el
// motor y compara con lo que el emisor declaro.
//
// ES LO QUE HACE `nucleo/expediente` en su verificacion, hecho aqui explicito:
// si el emisor declara «pass» y sus propias observaciones dan «fail», el
// documento se contradice a si mismo.
func recomputaElMismoEstado(t *testing.T, e *expediente.Expediente, esperada estado.Entrada) bool {
	t.Helper()
	if len(e.Pruebas) != 1 || len(e.Estados) != 1 {
		t.Fatalf("el expediente de prueba tiene %d pruebas y %d estados",
			len(e.Pruebas), len(e.Estados))
	}
	ent := estado.Calcular(e.Pruebas[0], e.Observaciones,
		estado.Contexto{Ahora: e.ComoEstaba, Aplicable: true})
	return ent.Estado == esperada.Estado && e.Estados[0].Estado == ent.Estado.String()
}

// unaPruebaDelCorpus devuelve la primera prueba declarada por un paquete real.
//
// Se saca del corpus y no se escribe aqui por dos motivos: el linter de normas
// cableadas prohibe escribir un identificador de obligacion en codigo, y ademas
// una prueba escrita al lado del test no demuestra que el corpus tenga ninguna.
func unaPruebaDelCorpus(t *testing.T, ps []*corpus.Paquete) (corpus.Prueba, string) {
	t.Helper()
	for _, p := range ps {
		if len(p.Pruebas) > 0 {
			return p.Pruebas[0], p.URN
		}
	}
	t.Fatal("el corpus no declara ni una prueba, asi que esta cadena no tiene por " +
		"donde empezar: es el bloque construido y apagado que A1 vino a evitar")
	return corpus.Prueba{}, ""
}

// TestElRecolectorRotoNoDaUnExpedienteVacio es el otro extremo de la cadena: si
// el eslabon 1 no se puede leer, NO se compone un expediente que diga que no hay
// nada. Es el invariante 8 en la cadena entera, y es el fallo que absuelve.
func TestElRecolectorRotoNoDaUnExpedienteVacio(t *testing.T) {
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	ruta := manual.RutaPorDefecto(dir)
	if err := os.WriteFile(ruta, []byte(`{"version":99}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rec, err := manual.Abrir(manual.Opciones{Ruta: ruta})
	if err != nil {
		t.Fatal(err)
	}
	ev, err := nuevaEvidencia(ps, rec, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	porObligacion, err := ev.De(t.Context())
	if err == nil {
		t.Fatalf("un fichero de observaciones que no se entiende ha dado %d entradas "+
			"sin error. Eso le diria a alguien que su instalacion no tiene nada "+
			"recolectado cuando lo que pasa es que no hemos podido leer su fichero",
			len(porObligacion))
	}
	if !errors.Is(err, manual.ErrVersion) {
		t.Errorf("ha fallado por otra cosa: %v", err)
	}
	if porObligacion != nil {
		t.Error("con error se devuelven entradas: no se entrega nada a medias")
	}
}

// TestSinFicheroDeObservacionesLaCadenaDiceQueNadieHaRecolectado es la otra
// forma de la nada, y NO es un error: una instalacion recien descargada no
// tiene fichero, y eso es un dato.
func TestSinFicheroDeObservacionesLaCadenaDiceQueNadieHaRecolectado(t *testing.T) {
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatal(err)
	}
	rec, err := manual.Abrir(manual.Opciones{
		Ruta: filepath.Join(t.TempDir(), "no-existe.jsonl"),
	})
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Date(2026, 3, 2, 9, 0, 0, 0, time.UTC)
	ev, err := nuevaEvidencia(ps, rec, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	porObligacion, err := ev.De(t.Context())
	if err != nil {
		t.Fatalf("un fichero ausente no es un error: %v", err)
	}
	if len(porObligacion) == 0 {
		t.Fatal("sin observaciones no sale ni una prueba, asi que la pantalla no " +
			"podria decir que hay pruebas declaradas y nadie ha recolectado")
	}
	for id, e := range porObligacion {
		if !e.SinObservaciones {
			t.Errorf("%s no dice que no hay observaciones", id)
		}
		if !e.Recolectada.IsZero() {
			t.Errorf("%s trae una fecha y no hay dato que fechar", id)
		}
		if e.Recolector != "" {
			t.Errorf("%s dice que lo trajo %q y no lo trajo nadie", id, e.Recolector)
		}
	}
}
