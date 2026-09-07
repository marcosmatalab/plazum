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

// unaObligacionPorFaceta es documental Y ADEMAS tiene un aspecto observable.
// Una prueba en verde sobre ella aporta a ese aspecto y NO la cierra: lo que la
// norma exige ahi es un documento, y ese documento puede no existir.
const unaObligacionPorFaceta = `{
  "id":"demo.papeles_con_aspecto","articulo":"3","cita":"art. 3",
  "vigencia":{"desde":"2020-01-01"},
  "clase_e2e":"documental","facetas":["observable"],"recursos":["Documento"]}`

// unaObligacionDocumental NO es observable por ningun lado. Es el arnes de la
// puerta que pide D-22.
const unaObligacionDocumental = `{
  "id":"demo.papeles","articulo":"2","cita":"art. 2",
  "vigencia":{"desde":"2020-01-01"},
  "clase_e2e":"documental","recursos":["Documento"]}`

const pruebaBuena = `{
  "id":"demo.accesos.privilegios","obligacion":"demo.accesos","recurso":"Privilegio",
  "ttl":"P30D","sla":"P7D","cierra":true,
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
	porClase, porFaceta, faltanClase, faltanFaceta := 0, 0, 0, 0
	for _, p := range ps {
		for _, o := range p.Obligaciones {
			if o.ObservablePorClase() {
				porClase++
			}
			if o.ObservablePorFaceta() {
				porFaceta++
			}
		}
		f := p.ObservablesSinPrueba()
		faltanClase += len(f.PorClase)
		faltanFaceta += len(f.PorFaceta)
	}
	// LAS DOS MITADES, NUNCA LA UNION. No cuestan lo mismo ni significan lo
	// mismo: una observable por clase sin prueba es una comprobacion que falta
	// entera; una por faceta es un aspecto sin cubrir de algo que se cierra por
	// otra via. Un solo numero las hace parecer homogeneas.
	t.Logf("observables por CLASE %d (faltan %d), por FACETA %d (faltan %d)",
		porClase, faltanClase, porFaceta, faltanFaceta)
	if porClase == 0 || porFaceta == 0 {
		t.Errorf("una de las dos mitades sale a cero (clase %d, faceta %d): el contador "+
			"no esta mirando los dos campos, que es el error que costo publicar un techo "+
			"cuatro veces mas bajo que el real", porClase, porFaceta)
	}
	if faltanClase+faltanFaceta == 0 {
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
				"ttl":"P30D","sla":"P7D","cierra":true,"predicado":"x"}`,
			centinela: ErrPruebaHuerfana,
			porque:    "es un enlace que nadie puede seguir, y en silencio no se comprueba nada",
		},
		{
			nombre:       "una prueba sobre una obligacion que NO es observable",
			obligaciones: unaObligacionDocumental,
			pruebas: `{"id":"p","obligacion":"demo.papeles","recurso":"Documento",
				"ttl":"P30D","sla":"P7D","cierra":true,"predicado":"x"}`,
			centinela: ErrPruebaNoObservable,
			porque: "promete una comprobacion automatica que ningun recolector va a poder " +
				"hacer: es LA puerta que pide D-22",
		},
		{
			nombre:       "un recurso que la obligacion no declara",
			obligaciones: unaObligacionObservable,
			pruebas: `{"id":"p","obligacion":"demo.accesos","recurso":"Sistema",
				"ttl":"P30D","sla":"P7D","cierra":true,"predicado":"x"}`,
			centinela: ErrPruebaRecursoAjeno,
			porque:    "inventarse el alcance de la comprobacion",
		},
		{
			nombre:       "sin recurso",
			obligaciones: unaObligacionObservable,
			pruebas: `{"id":"p","obligacion":"demo.accesos",
				"ttl":"P30D","sla":"P7D","cierra":true,"predicado":"x"}`,
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
				"sla":"P7D","cierra":true,"predicado":"x"}`,
			centinela: ErrPruebaTTLInvalido,
			porque: "el cero permisivo del invariante 8: haria que una observacion de hace " +
				"tres anos valiera igual que la de esta manana",
		},
		{
			nombre:       "ttl negativo",
			obligaciones: unaObligacionObservable,
			pruebas: `{"id":"p","obligacion":"demo.accesos","recurso":"Privilegio",
				"ttl":"-P1D","sla":"P7D","cierra":true,"predicado":"x"}`,
			centinela: ErrPruebaTTLInvalido,
			porque:    "una frescura negativa no es una frescura corta, es un dato ilegible",
		},
		{
			nombre:       "ttl de cero dias",
			obligaciones: unaObligacionObservable,
			pruebas: `{"id":"p","obligacion":"demo.accesos","recurso":"Privilegio",
				"ttl":"P0D","sla":"P7D","cierra":true,"predicado":"x"}`,
			centinela: ErrPruebaTTLInvalido,
			porque:    "una observacion que caduca al instante no la puede aportar nadie",
		},
		{
			nombre:       "sla mayor que el ttl",
			obligaciones: unaObligacionObservable,
			pruebas: `{"id":"p","obligacion":"demo.accesos","recurso":"Privilegio",
				"ttl":"P7D","sla":"P30D","cierra":true,"predicado":"x"}`,
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
				"ttl":"P30D","sla":"P7D","cierra":true}`,
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
				"ttl":"P30D","sla":"P7D","cierra":true,"predicado":"x"},
				{"id":"p","obligacion":"demo.accesos","recurso":"Identidad",
				"ttl":"P30D","sla":"P7D","cierra":true,"predicado":"y"}`,
			centinela: ErrPruebaRepetida,
			porque: "las observaciones de una acabarian contadas en la otra, y cual gana " +
				"lo decidiria el orden del fichero",
		},
		{
			nombre:       "una prueba sin id",
			obligaciones: unaObligacionObservable,
			pruebas: `{"obligacion":"demo.accesos","recurso":"Privilegio",
				"ttl":"P30D","sla":"P7D","cierra":true,"predicado":"x"}`,
			centinela: ErrPruebaSinID,
			porque:    "una observacion la nombra por su id",
		},
		{
			// C2: LA MITAD QUE ABSUELVE DE MAS. Si esto cargara, la pantalla
			// pintaria «consta» sobre una obligacion documental porque su
			// aspecto observable dio verde, y quien lo lea deja de buscar el
			// documento.
			nombre:       "una prueba que dice CERRAR una obligacion observable solo por faceta",
			obligaciones: unaObligacionPorFaceta,
			pruebas: `{"id":"p","obligacion":"demo.papeles_con_aspecto","recurso":"Documento",
				"ttl":"P30D","sla":"P7D","cierra":true,"predicado":"x"}`,
			centinela: ErrPruebaCierraUnaFaceta,
			porque: "absolver de mas con cara de dato, que es el error simetrico de acusar " +
				"en falso y el que nadie mira",
		},
		{
			// El invariante 8 en su forma de JSON: un bool AUSENTE y un bool a
			// FALSE son dos cosas distintas, y solo una es una decision.
			nombre:       "sin decir si cierra",
			obligaciones: unaObligacionObservable,
			pruebas: `{"id":"p","obligacion":"demo.accesos","recurso":"Privilegio",
				"ttl":"P30D","sla":"P7D","predicado":"x"}`,
			centinela: ErrPruebaSinCierra,
			porque:    "el campo ausente no se puede leer como false: eso es elegir por el autor",
		},
		{
			nombre:       "una prueba sin obligacion",
			obligaciones: unaObligacionObservable,
			pruebas: `{"id":"p","recurso":"Privilegio",
				"ttl":"P30D","sla":"P7D","cierra":true,"predicado":"x"}`,
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
				"ttl":"P6M","sla":"P6M","activa":"2026-01-01","pass_por_defecto":true,"cierra":true,
				"predicado":"el proveedor garantiza la unicidad de la identidad"}`,
		},
		{
			// sla IGUAL que ttl vale: lo que no vale es MAYOR. El borde se
			// recorre a proposito, que es donde viven los off-by-one.
			"sla exactamente igual que el ttl",
			`{"id":"p","obligacion":"demo.accesos","recurso":"Privilegio",
				"ttl":"P7D","sla":"P7D","cierra":true,"predicado":"x"}`,
		},
		{
			// LA DIRECCION CONTRARIA DE C2: sobre una faceta, `cierra:false` SI
			// vale. Sin este caso, la puerta de arriba la aprobaria un linter
			// que rechazara toda prueba sobre una faceta, que es otra cosa y
			// dejaria sin cubrir 34 obligaciones.
			"cierra false sobre una faceta",
			`{"id":"p","obligacion":"demo.accesos","recurso":"Privilegio",
				"ttl":"P30D","sla":"P7D","cierra":false,"predicado":"x"}`,
		},
		{
			"dos pruebas distintas sobre la misma obligacion",
			`{"id":"p1","obligacion":"demo.accesos","recurso":"Privilegio",
				"ttl":"P30D","sla":"P7D","cierra":true,"predicado":"x"},
				{"id":"p2","obligacion":"demo.accesos","recurso":"Identidad",
				"ttl":"P30D","sla":"P7D","cierra":true,"predicado":"y"}`,
		},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if err := cargarConPruebas(t, unaObligacionObservable, c.pruebas); err != nil {
				t.Fatalf("una prueba legitima no carga: %v", err)
			}
		})
	}

	// Y sobre una obligacion observable SOLO POR FACETA, con cierra a false.
	t.Run("una prueba que aporta a una faceta sin cerrarla", func(t *testing.T) {
		if err := cargarConPruebas(t, unaObligacionPorFaceta,
			`{"id":"p","obligacion":"demo.papeles_con_aspecto","recurso":"Documento",
				"ttl":"P30D","sla":"P7D","cierra":false,
				"predicado":"consta el documento firmado en el repositorio documental"}`); err != nil {
			t.Fatalf("una prueba que APORTA a una faceta tiene que valer: %v", err)
		}
	})
}

// TestLasDosMitadesDeObservableNoSeSolapan es la puerta del cardinal partido:
// una obligacion cae en UNA de las dos y nunca en las dos, que es lo que hace
// que sumarlas de el total sin contar nada dos veces.
func TestLasDosMitadesDeObservableNoSeSolapan(t *testing.T) {
	casos := []struct {
		nombre        string
		o             Obligacion
		clase, faceta bool
	}{
		{"clase primaria", Obligacion{ClaseE2E: "observable"}, true, false},
		{"clase primaria Y faceta repetida", Obligacion{ClaseE2E: "observable",
			Facetas: []string{"observable"}}, true, false},
		{"solo faceta", Obligacion{ClaseE2E: "documental",
			Facetas: []string{"observable"}}, false, true},
		{"ninguna", Obligacion{ClaseE2E: "documental"}, false, false},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			if got := c.o.ObservablePorClase(); got != c.clase {
				t.Errorf("por clase %v, esperaba %v", got, c.clase)
			}
			if got := c.o.ObservablePorFaceta(); got != c.faceta {
				t.Errorf("por faceta %v, esperaba %v", got, c.faceta)
			}
			if c.o.ObservablePorClase() && c.o.ObservablePorFaceta() {
				t.Error("cae en las DOS mitades: sumarlas contaria esta obligacion dos veces")
			}
			if quiero := c.clase || c.faceta; c.o.EsObservable() != quiero {
				t.Errorf("EsObservable %v, esperaba %v", c.o.EsObservable(), quiero)
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
