package pantallas

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/marcosmatalab/plazum/nucleo/pantalla"
)

// La pantalla Hoy, que es donde el operador se entera de que su planificador
// esta muerto.
//
// Lo que se prueba aqui es de PRESENTACION: que el veredicto llega a la pagina,
// que llega el de la peticion y no el del arranque, y que un canal callado no
// pinta el planificador en rojo. La regla de las 24 horas no se prueba aqui
// porque no se decide aqui: vive en nucleo/pantalla, con su tabla de bordes y
// su caso dorado.

var instante = time.Date(2026, 8, 26, 9, 0, 0, 0, time.UTC)

// relojFijo devuelve una opcion que fija el reloj de la superficie.
func relojFijo(t time.Time) func(*Opciones) {
	return func(o *Opciones) { o.Ahora = func() time.Time { return t } }
}

// conMarcas devuelve una opcion que fija lo que la instalacion sabe de si misma.
func conMarcas(m pantalla.Marcas) func(*Opciones) {
	return func(o *Opciones) { o.Marcas = func() pantalla.Marcas { return m } }
}

// vigilancias son los estados que la pantalla Hoy puede pintar, uno por
// combinacion de opciones.
//
// Existe para que el barrido de claves de catalogo pase por TODOS los
// veredictos. Sin esto, el barrido solo alcanzaria el estado por defecto y las
// claves de los demas se quedarian sin traducir hasta que un cliente se las
// encontrara en crudo.
func vigilancias() []func(*Opciones) {
	vivo := instante.Add(-time.Hour)
	con := func(m pantalla.Marcas, ahora time.Time) func(*Opciones) {
		return func(o *Opciones) {
			relojFijo(ahora)(o)
			conMarcas(m)(o)
		}
	}
	return []func(*Opciones){
		// El planificador, en sus cinco estados.
		con(pantalla.Marcas{}, instante),                                           // nunca latio
		con(pantalla.Marcas{UltimoCiclo: vivo}, instante),                          // late
		con(pantalla.Marcas{UltimoCiclo: instante.Add(-48 * time.Hour)}, instante), // callado
		con(pantalla.Marcas{UltimoCiclo: instante.Add(72 * time.Hour)}, instante),  // futuro
		con(pantalla.Marcas{UltimoCiclo: vivo}, time.Time{}),                       // sin instante
		// El canal, en sus cinco.
		con(pantalla.Marcas{UltimoCiclo: vivo, LatidoActivado: true}, instante),
		con(pantalla.Marcas{UltimoCiclo: vivo, LatidoActivado: true,
			UltimoPulso: instante.Add(-time.Hour)}, instante),
		con(pantalla.Marcas{UltimoCiclo: vivo, LatidoActivado: true,
			UltimoPulso: instante.Add(-48 * time.Hour)}, instante),
		con(pantalla.Marcas{UltimoCiclo: vivo, LatidoActivado: true,
			UltimoPulso: instante.Add(-time.Hour), FalloElUltimoIntento: true}, instante),
	}
}

// Hoy pinta el veredicto del planificador, con su nivel y su arreglo.
func TestHoyEnsenaElEstadoDelPlanificador(t *testing.T) {
	casos := []struct {
		nombre string
		marcas pantalla.Marcas
		ahora  time.Time
		nivel  string
		clave  string
		// arreglo vacio significa que la pagina NO tiene que traer arreglo.
		arreglo string
	}{
		{"nunca ha corrido un ciclo", pantalla.Marcas{}, instante, "aviso",
			pantalla.ClavePlanificadorNunca, pantalla.ClaveArreglaNunca},
		{"late", pantalla.Marcas{UltimoCiclo: instante.Add(-time.Hour)}, instante,
			"correcto", pantalla.ClavePlanificadorLate, ""},
		{"callado dos dias", pantalla.Marcas{UltimoCiclo: instante.Add(-48 * time.Hour)},
			instante, "roto", pantalla.ClavePlanificadorCallado, pantalla.ClaveArreglaCallado},
		{"con la marca en el futuro", pantalla.Marcas{UltimoCiclo: instante.Add(72 * time.Hour)},
			instante, "aviso", pantalla.ClavePlanificadorFuturo, pantalla.ClaveArreglaFuturo},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			s, _ := superficie(t, corpusDemo(), relojFijo(c.ahora), conMarcas(c.marcas))
			w, cuerpo := pedir(t, s, "/hoy")
			if w.Code != 200 {
				t.Fatalf("GET /hoy dio %d", w.Code)
			}
			exige(t, cuerpo, rotulo("es", c.clave))
			exige(t, cuerpo, rotulo("es", "pantalla.hoy.planificador"))
			if !strings.Contains(cuerpo, `class="vigilancia n-`+c.nivel+`"`) {
				t.Errorf("la pagina no marca el nivel %q del veredicto, asi que la hoja de "+
					"estilo no puede distinguir un planificador muerto de uno vivo", c.nivel)
			}
			if c.arreglo != "" {
				exige(t, cuerpo, rotulo("es", c.arreglo))
			}
		})
	}
}

// EL VEREDICTO ES EL DE LA PETICION, no el del arranque.
//
// Es el fallo natural de esta pieza: el modelo derivado se calcula una vez y se
// guarda, asi que lo comodo es meter ahi el estado del planificador. Con eso,
// un servidor que lleve tres semanas levantado seguiria diciendo "late" para
// siempre, que es exactamente la mentira que el vigilante existe para no
// contar.
func TestElEstadoDelPlanificadorSeJuzgaEnCadaPeticion(t *testing.T) {
	var mu sync.Mutex
	ahora := instante
	reloj := func(o *Opciones) {
		o.Ahora = func() time.Time {
			mu.Lock()
			defer mu.Unlock()
			return ahora
		}
	}
	marcas := conMarcas(pantalla.Marcas{UltimoCiclo: instante.Add(-time.Hour)})

	s, _ := superficie(t, corpusDemo(), reloj, marcas)
	_, cuerpo := pedir(t, s, "/hoy")
	exige(t, cuerpo, rotulo("es", pantalla.ClavePlanificadorLate))

	// Pasan dos dias sin que nadie reinicie el servidor ni recargue el corpus.
	mu.Lock()
	ahora = instante.Add(48 * time.Hour)
	mu.Unlock()

	_, cuerpo = pedir(t, s, "/hoy")
	exige(t, cuerpo, rotulo("es", pantalla.ClavePlanificadorCallado))
	prohibe(t, cuerpo, rotulo("es", pantalla.ClavePlanificadorLate))
}

// LA DIRECCION DEL AVISO, comprobada sobre la pagina.
//
// Con el planificador latiendo y el canal hacia nosotros callado nueve dias, la
// pagina tiene que pintar el planificador CORRECTO, el canal en aviso, y decir
// con esas palabras que lo que calla es el canal. Si no, nuestra caida se lee
// como su planificador muerto y el operador aprende a ignorar el rojo.
func TestUnCanalCalladoNoPintaElPlanificadorEnRojo(t *testing.T) {
	s, _ := superficie(t, corpusDemo(),
		relojFijo(instante),
		conMarcas(pantalla.Marcas{
			UltimoCiclo:    instante.Add(-time.Minute),
			LatidoActivado: true,
			UltimoPulso:    instante.Add(-9 * 24 * time.Hour),
		}))
	_, cuerpo := pedir(t, s, "/hoy")

	if !strings.Contains(cuerpo, `class="vigilancia n-correcto"`) {
		t.Error("con el planificador latiendo hace un minuto, la pagina no lo pinta correcto")
	}
	if !strings.Contains(cuerpo, `class="canal n-aviso"`) {
		t.Error("con el canal callado nueve dias, la pagina no lo pinta en aviso")
	}
	if strings.Contains(cuerpo, "n-roto") {
		t.Error("hay algo pintado en rojo y lo unico que pasa es que NUESTRO canal calla")
	}
	exige(t, cuerpo,
		rotulo("es", pantalla.ClavePlanificadorLate),
		rotulo("es", pantalla.ClaveLatidoCallado),
		rotulo("es", pantalla.ClaveLatidoNoEsTuPlanificador),
	)
}

// El descargo de direccion sale SIEMPRE, tambien con el latido apagado, que es
// el estado por defecto y por tanto el que ve casi todo el mundo.
func TestElDescargoDeDireccionSaleTambienConElLatidoApagado(t *testing.T) {
	s, _ := superficie(t, corpusDemo(), relojFijo(instante),
		conMarcas(pantalla.Marcas{UltimoCiclo: instante.Add(-time.Minute)}))
	_, cuerpo := pedir(t, s, "/hoy")
	exige(t, cuerpo,
		rotulo("es", pantalla.ClaveLatidoApagado),
		rotulo("es", pantalla.ClaveLatidoNoEsTuPlanificador),
	)
	// Y el latido apagado NO se pinta como un problema: es el valor por
	// defecto y la postura correcta.
	if strings.Contains(cuerpo, `class="canal n-aviso"`) ||
		strings.Contains(cuerpo, `class="canal n-roto"`) {
		t.Error("el latido apagado se pinta como un problema, o sea que la pantalla esta " +
			"empujando a encender la telemetria")
	}
}

// Hoy sigue diciendo que aparecera aqui cuando haya expediente. El vigilante se
// anade encima, no en lugar de.
func TestHoySigueExplicandoQueApareceraCuandoHayaEstado(t *testing.T) {
	s, _ := superficie(t, corpusDemo(), relojFijo(instante))
	_, cuerpo := pedir(t, s, "/hoy")
	exige(t, cuerpo,
		rotulo("es", "pantalla.hoy.plazos"),
		rotulo("es", "pantalla.hoy.vacia"),
		rotulo("es", "origen.estado"),
		rotulo("es", "vacia.que_hacer"),
	)
	prohibe(t, cuerpo, rotulo("es", "vacia.sin_explicacion"))
}

// Sin nadie que le diga la hora ni las marcas, la superficie no da por bueno el
// planificador. Es el camino que recorre hoy `plazum serve`, que todavia no
// tiene planificador que enganchar.
func TestSinMarcasHoyNoDiceQueTodoVaBien(t *testing.T) {
	s, _ := superficie(t, corpusDemo())
	_, cuerpo := pedir(t, s, "/hoy")
	if strings.Contains(cuerpo, `class="vigilancia n-correcto"`) {
		t.Error("sin marcas del planificador, Hoy dice que el planificador esta correcto")
	}
	exige(t, cuerpo, rotulo("es", pantalla.ClavePlanificadorNunca))
}

// El veredicto no lleva ni un dato del corpus ni del expediente: son claves
// nuestras y numeros. Importa porque esta pieza es la vecina del canal que sale
// a internet, y lo que se pinta aqui es lo que alguien acabaria mandando.
func TestElVeredictoNoLlevaContenidoDelCorpus(t *testing.T) {
	v := pantalla.Vigilar(pantalla.Marcas{UltimoCiclo: instante.Add(-time.Hour)}, instante)
	for _, c := range []string{v.Clave, v.Arreglo, v.Canal.Clave, v.Canal.Arreglo,
		v.Canal.Descargo} {
		if c == "" {
			continue
		}
		if !strings.HasPrefix(c, "aviso.") && !strings.HasPrefix(c, "pantalla.") {
			t.Errorf("%q no es una clave de interfaz", c)
		}
	}
}
