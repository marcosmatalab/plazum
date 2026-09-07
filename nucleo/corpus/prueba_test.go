package corpus

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// LAS TRES PASADAS DE A1, EN ORDEN, Y LA PRIMERA ES LA QUE MAS COSTABA.
//
// # Pasada 1: ¿puede un paquete declarar una prueba SIN TOCAR CODIGO?
//
// Es la pregunta que obliga a hacerse el invariante 2, y la que salio de
// `maximo`: una primitiva construida, con sus dorados en verde, y APAGADA para
// el corpus durante semanas porque solo se encendia escribiendo Go. Aqui se
// contesta con un paquete escrito a mano y cargado desde disco, no con un
// literal de Go, porque un literal demuestra que el TIPO existe y no que el
// FORMATO lo admita.
//
// Y ademas se contesta sobre el corpus real: `paquetes/ens` declara una prueba
// de verdad desde el 07-09-2026, asi que el bloque no nace apagado.

// paqueteConPruebas monta el JSON minimo que carga, con las pruebas que se le
// pasen. Se escribe como TEXTO y no como estructura a proposito: es la unica
// forma de comprobar que el formato admite el bloque.
func paqueteConPruebas(obligaciones, pruebas string) string {
	return fmt.Sprintf(`{
  "urn":"urn:demo:pruebas@1","version":"1","clase":4,
  "licencia_fuente":"del-proyecto",
  "atribucion":"Paquete de prueba del propio proyecto.",
  "identificador":{"tipo":"sin-identificador","valor":"demo",
    "motivo":"paquete sintetico de un test"},
  "vigencia":{"desde":"2020-01-01"},
  "obligaciones":[%s],
  "pruebas":[%s]}`, obligaciones, pruebas)
}

// unaObligacionObservable es la obligacion contra la que se declaran las
// pruebas de abajo. Observable por CLASE PRIMARIA y con dos recursos.
const unaObligacionObservable = `{
  "id":"demo.accesos","articulo":"1","cita":"art. 1",
  "vigencia":{"desde":"2020-01-01"},
  "clase_e2e":"observable","recursos":["Identidad","Privilegio"]}`

// unaObligacionDocumental NO es observable por ningun lado. Es el arnes de la
// puerta que pide D-22.
const unaObligacionDocumental = `{
  "id":"demo.papeles","articulo":"2","cita":"art. 2",
  "vigencia":{"desde":"2020-01-01"},
  "clase_e2e":"documental","recursos":["Documento"]}`

const pruebaBuena = `{
  "id":"demo.accesos.privilegios","obligacion":"demo.accesos","recurso":"Privilegio",
  "ttl":"P30D","sla":"P7D",
  "predicado":"cada privilegio consta asignado a una funcion declarada"}`

func cargarConPruebas(t *testing.T, obligaciones, pruebas string) error {
	t.Helper()
	dir := t.TempDir()
	escribirPaquete(t, dir, "conpruebas", paqueteConPruebas(obligaciones, pruebas))
	_, err := Cargar(dir)
	return err
}

// TestUnPaqueteDeclaraUnaPruebaSinTocarCodigo es la pasada 1, y es la unica de
// las tres que puede dejar la casilla sin marcar aunque el linter este verde.
func TestUnPaqueteDeclaraUnaPruebaSinTocarCodigo(t *testing.T) {
	if err := cargarConPruebas(t, unaObligacionObservable, pruebaBuena); err != nil {
		t.Fatalf("un paquete escrito a mano con una prueba dentro NO carga: %v\n"+
			"  Si esto falla, la primitiva esta a medias aunque sus tests pasen: seria "+
			"`maximo` otra vez, construido y apagado para el corpus.", err)
	}
}

// TestElCorpusRealYaDeclaraUnaPrueba: el bloque no nace apagado.
//
// UNA, y se dice que es una. La campana de declararlas en los 12 marcos es la
// casilla de la etapa 3, no esta. Lo que esta comprueba es que el camino de
// datos esta abierto sobre el corpus de verdad y no solo sobre un temporal.
func TestElCorpusRealYaDeclaraUnaPrueba(t *testing.T) {
	ps, err := Cargar("../../paquetes")
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, p := range ps {
		total += len(p.Pruebas)
	}
	if total == 0 {
		t.Fatal("el corpus no declara ni una prueba: el bloque esta construido y apagado, " +
			"que es exactamente lo que le paso a `maximo`")
	}
	t.Logf("el corpus real declara %d prueba(s)", total)
}

// TestElTechoDeLaRecoleccionSeDerivaYNoSeEscribe imprime el cardinal que la
// casilla de la etapa 3 tiene que bajar.
//
// NO afirma un techo ni un suelo: afirmar un numero aqui seria una promesa sobre
// el contenido del corpus, y el corpus crece. Lo que se afirma es que el
// contador CUENTA, o sea que no se ha quedado mirando el vacio.
func TestElTechoDeLaRecoleccionSeDerivaYNoSeEscribe(t *testing.T) {
	ps, err := Cargar("../../paquetes")
	if err != nil {
		t.Fatal(err)
	}
	observables, sinPrueba := 0, 0
	for _, p := range ps {
		for _, o := range p.Obligaciones {
			if o.EsObservable() {
				observables++
			}
		}
		sinPrueba += len(p.ObservablesSinPrueba())
	}
	t.Logf("%d obligaciones observables, %d sin prueba declarada", observables, sinPrueba)
	if observables == 0 {
		t.Error("cero observables sobre el corpus entero: EsObservable no esta mirando " +
			"los dos campos, que es el error que costo publicar un techo cuatro veces " +
			"mas bajo que el real")
	}
	if sinPrueba == 0 {
		t.Error("cero observables sin prueba: o el corpus ya esta terminado (y entonces " +
			"la casilla de la etapa 3 se cierra) o el contador no esta contando")
	}
}

// TestLasFormasDeRomperUnaPrueba es la pasada 2: el atacante escribe el bloque.
//
// Cada caso trae su centinela, porque quien llama tiene que poder distinguir
// «apunta a una obligacion que no existe» de «el ttl no se entiende» sin leer
// una cadena.
func TestLasFormasDeRomperUnaPrueba(t *testing.T) {
	casos := []struct {
		nombre       string
		obligaciones string
		pruebas      string
		centinela    error
		porque       string
	}{
		{
			nombre:       "una prueba sobre una obligacion que no existe",
			obligaciones: unaObligacionObservable,
			pruebas: `{"id":"p","obligacion":"demo.no_existe","recurso":"Privilegio",
				"ttl":"P30D","sla":"P7D","predicado":"x"}`,
			centinela: ErrPruebaHuerfana,
			porque:    "es un enlace que nadie puede seguir, y en silencio no se comprueba nada",
		},
		{
			nombre:       "una prueba sobre una obligacion que NO es observable",
			obligaciones: unaObligacionDocumental,
			pruebas: `{"id":"p","obligacion":"demo.papeles","recurso":"Documento",
				"ttl":"P30D","sla":"P7D","predicado":"x"}`,
			centinela: ErrPruebaNoObservable,
			porque: "promete una comprobacion automatica que ningun recolector va a poder " +
				"hacer: es LA puerta que pide D-22",
		},
		{
			nombre:       "un recurso que la obligacion no declara",
			obligaciones: unaObligacionObservable,
			pruebas: `{"id":"p","obligacion":"demo.accesos","recurso":"Sistema",
				"ttl":"P30D","sla":"P7D","predicado":"x"}`,
			centinela: ErrPruebaRecursoAjeno,
			porque:    "inventarse el alcance de la comprobacion",
		},
		{
			nombre:       "sin recurso",
			obligaciones: unaObligacionObservable,
			pruebas: `{"id":"p","obligacion":"demo.accesos",
				"ttl":"P30D","sla":"P7D","predicado":"x"}`,
			centinela: ErrPruebaSinRecurso,
			porque:    "no se sabe sobre que se observa",
		},
		{
			// LA DECISION DEL VALOR CERO, ESCRITA (invariante 8). Un ttl vacio
			// NO significa «no caduca»: significa que falta. Si de verdad no
			// caduca, eso se dice con pass_por_defecto.
			nombre:       "ttl ausente, que NO significa sin caducidad",
			obligaciones: unaObligacionObservable,
			pruebas: `{"id":"p","obligacion":"demo.accesos","recurso":"Privilegio",
				"sla":"P7D","predicado":"x"}`,
			centinela: ErrPruebaTTLInvalido,
			porque: "el cero permisivo del invariante 8: haria que una observacion de hace " +
				"tres anos valiera igual que la de esta manana",
		},
		{
			nombre:       "ttl negativo",
			obligaciones: unaObligacionObservable,
			pruebas: `{"id":"p","obligacion":"demo.accesos","recurso":"Privilegio",
				"ttl":"-P1D","sla":"P7D","predicado":"x"}`,
			centinela: ErrPruebaTTLInvalido,
			porque:    "una frescura negativa no es una frescura corta, es un dato ilegible",
		},
		{
			nombre:       "ttl de cero dias",
			obligaciones: unaObligacionObservable,
			pruebas: `{"id":"p","obligacion":"demo.accesos","recurso":"Privilegio",
				"ttl":"P0D","sla":"P7D","predicado":"x"}`,
			centinela: ErrPruebaTTLInvalido,
			porque:    "una observacion que caduca al instante no la puede aportar nadie",
		},
		{
			nombre:       "sla mayor que el ttl",
			obligaciones: unaObligacionObservable,
			pruebas: `{"id":"p","obligacion":"demo.accesos","recurso":"Privilegio",
				"ttl":"P7D","sla":"P30D","predicado":"x"}`,
			centinela: ErrPruebaSLAMayorQueTTL,
			porque: "es una escalada INALCANZABLE escrita como si fuera un plazo generoso: " +
				"la observacion se queda obsoleta antes de que el plazo venza",
		},
		{
			nombre:       "sla ausente",
			obligaciones: unaObligacionObservable,
			pruebas: `{"id":"p","obligacion":"demo.accesos","recurso":"Privilegio",
				"ttl":"P30D","predicado":"x"}`,
			centinela: ErrPruebaSLAInvalido,
			porque:    "sin plazo de remediacion no hay nada que escalar",
		},
		{
			nombre:       "sin predicado",
			obligaciones: unaObligacionObservable,
			pruebas: `{"id":"p","obligacion":"demo.accesos","recurso":"Privilegio",
				"ttl":"P30D","sla":"P7D"}`,
			centinela: ErrPruebaSinPredicado,
			porque: "sin predicado declarado, el Satisfecho de una observacion es un " +
				"veredicto disfrazado de dato (invariante 13)",
		},
		{
			nombre:       "fecha de rollout ilegible",
			obligaciones: unaObligacionObservable,
			pruebas: `{"id":"p","obligacion":"demo.accesos","recurso":"Privilegio",
				"ttl":"P30D","sla":"P7D","activa":"manana","predicado":"x"}`,
			centinela: ErrPruebaActivaIlegible,
			porque:    "la tercera forma de la nada: presente y no interpretable",
		},
		{
			nombre:       "dos pruebas con el mismo id",
			obligaciones: unaObligacionObservable,
			pruebas: `{"id":"p","obligacion":"demo.accesos","recurso":"Privilegio",
				"ttl":"P30D","sla":"P7D","predicado":"x"},
				{"id":"p","obligacion":"demo.accesos","recurso":"Identidad",
				"ttl":"P30D","sla":"P7D","predicado":"y"}`,
			centinela: ErrPruebaRepetida,
			porque: "las observaciones de una acabarian contadas en la otra, y cual gana " +
				"lo decidiria el orden del fichero",
		},
		{
			nombre:       "una prueba sin id",
			obligaciones: unaObligacionObservable,
			pruebas: `{"obligacion":"demo.accesos","recurso":"Privilegio",
				"ttl":"P30D","sla":"P7D","predicado":"x"}`,
			centinela: ErrPruebaSinID,
			porque:    "una observacion la nombra por su id",
		},
		{
			nombre:       "una prueba sin obligacion",
			obligaciones: unaObligacionObservable,
			pruebas: `{"id":"p","recurso":"Privilegio",
				"ttl":"P30D","sla":"P7D","predicado":"x"}`,
			centinela: ErrPruebaSinObligacion,
			porque:    "no se puede colgar de nada",
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			err := cargarConPruebas(t, c.obligaciones, c.pruebas)
			if err == nil {
				t.Fatalf("ha cargado. Y no puede: %s", c.porque)
			}
			if !errors.Is(err, c.centinela) {
				t.Fatalf("ha fallado por otra cosa:\n  esperaba: %v\n  y dio:    %v",
					c.centinela, err)
			}
			// UN ERROR DE LINTER SIN ARREGLO TRASLADA EL TRABAJO A QUIEN ESCRIBE
			// EL PAQUETE, que es justo a quien queremos que le resulte facil.
			if !strings.Contains(err.Error(), "Arreglo:") {
				t.Errorf("el error no dice como salir de ahi: %v", err)
			}
		})
	}
}

// TestElControlPositivoDelLinterDePruebas es la otra direccion, y sin el los
// catorce casos de arriba los aprobaria un linter que rechaza absolutamente
// toda prueba.
func TestElControlPositivoDelLinterDePruebas(t *testing.T) {
	casos := []struct {
		nombre  string
		pruebas string
	}{
		{"la prueba minima", pruebaBuena},
		{
			"con rollout y pass por defecto",
			`{"id":"p","obligacion":"demo.accesos","recurso":"Identidad",
				"ttl":"P6M","sla":"P6M","activa":"2026-01-01","pass_por_defecto":true,
				"predicado":"el proveedor garantiza la unicidad de la identidad"}`,
		},
		{
			// sla IGUAL que ttl vale: lo que no vale es MAYOR. El borde se
			// recorre a proposito, que es donde viven los off-by-one.
			"sla exactamente igual que el ttl",
			`{"id":"p","obligacion":"demo.accesos","recurso":"Privilegio",
				"ttl":"P7D","sla":"P7D","predicado":"x"}`,
		},
		{
			"dos pruebas distintas sobre la misma obligacion",
			`{"id":"p1","obligacion":"demo.accesos","recurso":"Privilegio",
				"ttl":"P30D","sla":"P7D","predicado":"x"},
				{"id":"p2","obligacion":"demo.accesos","recurso":"Identidad",
				"ttl":"P30D","sla":"P7D","predicado":"y"}`,
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if err := cargarConPruebas(t, unaObligacionObservable, c.pruebas); err != nil {
				t.Fatalf("una prueba legitima no carga: %v", err)
			}
		})
	}
}

// TestUnaObligacionEsObservablePorLosDosCampos es la puerta del cardinal que se
// publico mal: `observable` vive en `clase_e2e` y en `facetas`, y contar uno
// solo dio un techo cuatro veces mas bajo que el real.
func TestUnaObligacionEsObservablePorLosDosCampos(t *testing.T) {
	casos := []struct {
		nombre string
		o      Obligacion
		quiero bool
	}{
		{"clase primaria observable", Obligacion{ClaseE2E: "observable"}, true},
		{"faceta observable", Obligacion{ClaseE2E: "documental",
			Facetas: []string{"observable"}}, true},
		{"faceta observable entre otras", Obligacion{ClaseE2E: "procedimental",
			Facetas: []string{"documental", "observable"}}, true},
		{"ni una cosa ni la otra", Obligacion{ClaseE2E: "documental",
			Facetas: []string{"procedimental"}}, false},
		{"sin nada", Obligacion{}, false},
	}
	for _, c := range casos {
		if got := c.o.EsObservable(); got != c.quiero {
			t.Errorf("%s: %v, esperaba %v", c.nombre, got, c.quiero)
		}
	}
}
