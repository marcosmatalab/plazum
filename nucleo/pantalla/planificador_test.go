package pantalla

import (
	"sort"
	"strings"
	"testing"
	"time"
)

// El instante de referencia de estos tests. Fijo y con zona: si el veredicto
// dependiera del reloj de la maquina, estos tests pasarian de dia y fallarian
// de noche, que es la forma mas rapida de que una suite se ignore.
var t0 = time.Date(2026, 8, 26, 9, 0, 0, 0, time.UTC)

func hace(d time.Duration) time.Time { return t0.Add(-d) }

// El numero de la casilla, comprobado en el borde.
//
// Se prueba en 23h59m, en 24h clavadas y en 25h porque el fallo que de verdad
// se comete aqui es un `>` donde va un `>=`: con el, una instalacion parada
// exactamente 24 horas sale en verde, y ese caso existe todos los dias en
// cualquier cosa que corra a la misma hora.
func TestElPlanificadorSeDeclaraMuertoALas24Horas(t *testing.T) {
	casos := []struct {
		nombre   string
		silencio time.Duration
		quiero   Nivel
		clave    string
		horas    int
	}{
		{"un ciclo hace un minuto", time.Minute, NivelCorrecto, ClavePlanificadorLate, 0},
		{"un ciclo hace 23h59m", 23*time.Hour + 59*time.Minute, NivelCorrecto, ClavePlanificadorLate, 23},
		{"24 horas clavadas", 24 * time.Hour, NivelRoto, ClavePlanificadorCallado, 24},
		{"25 horas", 25 * time.Hour, NivelRoto, ClavePlanificadorCallado, 25},
		{"nueve dias", 9 * 24 * time.Hour, NivelRoto, ClavePlanificadorCallado, 216},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			p := Vigilar(Marcas{UltimoCiclo: hace(c.silencio)}, t0)
			if p.Nivel != c.quiero {
				t.Errorf("con %v de silencio el veredicto es %q y tenia que ser %q",
					c.silencio, p.Nivel, c.quiero)
			}
			if p.Clave != c.clave {
				t.Errorf("la clave es %q y esperaba %q", p.Clave, c.clave)
			}
			if p.Horas != c.horas {
				t.Errorf("dice %d horas de silencio y son %d", p.Horas, c.horas)
			}
			if p.Nivel != NivelCorrecto && p.Arreglo == "" {
				t.Error("un veredicto que no esta correcto y no dice como se arregla le " +
					"pasa el trabajo al operador, que es lo que esta pieza existe para evitar")
			}
			if p.UmbralHoras != 24 {
				t.Errorf("el umbral que se ensena es %d y la casilla dice 24", p.UmbralHoras)
			}
		})
	}
}

// LA PROPIEDAD DE ESTE FICHERO, y la razon de que la pieza exista.
//
// El veredicto del planificador se calcula con los relojes del operador y no
// mira el canal. Si lo mirara, una caida NUESTRA se leeria como una caida SUYA:
// el operador abriria Hoy, veria rojo, no encontraria nada roto en su maquina y
// en dos semanas habria aprendido a ignorar el rojo. Un aviso que se ignora es
// peor que no tener aviso, porque ademas da tranquilidad.
//
// Se prueba de la unica forma que no admite discusion: con el planificador
// latiendo hace un minuto y TODAS las formas de canal roto que existen, el
// veredicto tiene que ser correcto en las tres.
func TestUnCanalRotoNoPuedeEnsuciarElVeredictoDelPlanificador(t *testing.T) {
	vivo := hace(time.Minute)
	canales := []struct {
		nombre string
		m      Marcas
	}{
		{"el canal nunca llego", Marcas{UltimoCiclo: vivo, LatidoActivado: true}},
		{"el canal lleva nueve dias callado", Marcas{UltimoCiclo: vivo, LatidoActivado: true,
			UltimoPulso: hace(9 * 24 * time.Hour)}},
		{"el ultimo intento fallo", Marcas{UltimoCiclo: vivo, LatidoActivado: true,
			UltimoPulso: hace(time.Hour), FalloElUltimoIntento: true}},
	}
	for _, c := range canales {
		t.Run(c.nombre, func(t *testing.T) {
			p := Vigilar(c.m, t0)
			if p.Nivel != NivelCorrecto || p.Clave != ClavePlanificadorLate {
				t.Errorf("con el planificador latiendo hace un minuto y %s, el veredicto "+
					"del PLANIFICADOR sale %q (%s). Nuestra caida se estaria leyendo como "+
					"la suya", c.nombre, p.Nivel, p.Clave)
			}
			// Y el canal si tiene que enterarse: si saliera correcto, el
			// test de arriba pasaria por no mirar nada.
			if p.Canal.Nivel != NivelAviso {
				t.Errorf("%s y el canal sale %q: entonces la comprobacion de arriba no "+
					"esta distinguiendo nada", c.nombre, p.Canal.Nivel)
			}
			// Y jamas por encima de aviso: el canal no puede pintar rojo.
			if p.Canal.Nivel == NivelRoto {
				t.Error("el canal se ha pintado ROTO. El rojo es para lo que deja de mirar " +
					"los plazos del operador, y el canal hacia nosotros no es eso")
			}
		})
	}
}

// El reloj que miente. Una marca en el futuro hace que la resta salga negativa,
// y un silencio negativo es siempre menor que el umbral: la alarma se APAGA.
// Lo consigue un salto de NTP, un contenedor con la hora mal o un fichero de
// estado tocado a mano, y las tres cosas pasan.
func TestUnaMarcaEnElFuturoNoApagaLaAlarma(t *testing.T) {
	p := Vigilar(Marcas{UltimoCiclo: t0.Add(72 * time.Hour)}, t0)
	if p.Nivel == NivelCorrecto {
		t.Fatalf("con la ultima marca 72 horas en el FUTURO el veredicto sale %q (%s). "+
			"Escribir una fecha futura en el fichero de estado apagaria el aviso",
			p.Nivel, p.Clave)
	}
	if p.Clave != ClavePlanificadorFuturo {
		t.Errorf("la clave es %q y tenia que decir que la marca esta en el futuro", p.Clave)
	}
	if p.Horas != -1 {
		t.Errorf("dice %d horas de silencio con una marca futura: inventar un numero es "+
			"peor que decir que no se sabe", p.Horas)
	}
	if p.Arreglo == "" {
		t.Error("no dice como se arregla un reloj que va adelantado")
	}
}

// Sin instante no se da nada por bueno. Es un fallo de integracion (alguien
// derivo sin pasar el reloj), no del operador, pero un vigilante que dice
// "correcto" sobre lo que no ha podido mirar es el fallo entero de la pieza.
func TestSinInstanteNoSeDaNadaPorBueno(t *testing.T) {
	for _, m := range []Marcas{
		{},
		{UltimoCiclo: hace(time.Minute)},
		{UltimoCiclo: hace(30 * 24 * time.Hour)},
	} {
		p := Vigilar(m, time.Time{})
		if p.Nivel == NivelCorrecto {
			t.Errorf("sin instante, y con la marca %v, el veredicto sale correcto",
				m.UltimoCiclo)
		}
		if p.Clave != ClavePlanificadorSinInstante {
			t.Errorf("sin instante la clave es %q y tenia que decir que falta el instante",
				p.Clave)
		}
	}
}

// El latido apagado NO es un aviso, y esto es producto, no estilo.
//
// El pulso es opt-in y su valor por defecto es apagado. Un producto que pinta
// en amarillo el hecho de no haber activado la telemetria esta empujando a
// activarla, que es lo que hace todo el mundo y es exactamente por lo que nadie
// se fia de nadie en este mercado.
func TestElLatidoApagadoNoEsUnAviso(t *testing.T) {
	p := Vigilar(Marcas{UltimoCiclo: hace(time.Minute)}, t0)
	if p.Canal.Activado {
		t.Fatal("el canal sale activado sin que nadie lo haya activado")
	}
	if p.Canal.Nivel != NivelCorrecto {
		t.Errorf("con el latido apagado, el canal sale %q. Apagado es el valor por defecto "+
			"y la postura correcta: marcarlo en amarillo es empujar a encenderlo",
			p.Canal.Nivel)
	}
	if p.Canal.Clave != ClaveLatidoApagado {
		t.Errorf("la clave del canal apagado es %q", p.Canal.Clave)
	}
	if p.Canal.Arreglo != "" {
		t.Errorf("el canal apagado trae un arreglo (%q), o sea que pide que lo enciendas",
			p.Canal.Arreglo)
	}
}

// El descargo de direccion viaja SIEMPRE, no solo cuando el canal falla.
//
// Un descargo que solo sale en el caso malo es un descargo que se lee cuando la
// persona ya ha sacado la conclusion equivocada.
func TestElDescargoDeDireccionViajaSiempre(t *testing.T) {
	for _, m := range []Marcas{
		{},
		{UltimoCiclo: hace(time.Minute)},
		{UltimoCiclo: hace(time.Minute), LatidoActivado: true, UltimoPulso: hace(time.Minute)},
		{UltimoCiclo: hace(48 * time.Hour), LatidoActivado: true},
	} {
		if c := Vigilar(m, t0).Canal; c.Descargo != ClaveLatidoNoEsTuPlanificador {
			t.Errorf("el canal sale sin el descargo de direccion (%q) con las marcas %+v",
				c.Descargo, m)
		}
	}
}

// El smoke test del canal tiene que verse en el modelo: probar el canal y que
// falle no puede esperar 24 horas para notarse.
func TestElFalloDelUltimoIntentoSeVeAunqueElPulsoAnteriorLlegara(t *testing.T) {
	m := Marcas{UltimoCiclo: hace(time.Minute), LatidoActivado: true,
		UltimoPulso: hace(2 * time.Hour)}
	if c := Vigilar(m, t0).Canal; c.Nivel != NivelCorrecto || c.Clave != ClaveLatidoLate {
		t.Fatalf("sin fallo del ultimo intento el canal sale %q (%s), y este caso es el "+
			"control de lo de abajo", c.Nivel, c.Clave)
	}
	m.FalloElUltimoIntento = true
	c := Vigilar(m, t0).Canal
	if c.Nivel != NivelAviso || c.Clave != ClaveLatidoFallo {
		t.Errorf("el ultimo intento fallo y el canal sale %q (%s): el smoke test no se "+
			"estaria viendo hasta pasadas 24 horas", c.Nivel, c.Clave)
	}
}

// ---------------------------------------------------------------------------
// Las claves publicadas
// ---------------------------------------------------------------------------

// estadosAlcanzables recorre las combinaciones de marcas e instantes que
// producen cada veredicto. Vive fuera del test para que el control negativo de
// abajo pase por este mismo codigo.
func estadosAlcanzables() []Planificador {
	vivo := hace(time.Minute)
	var out []Planificador
	for _, c := range []struct {
		m     Marcas
		ahora time.Time
	}{
		{Marcas{}, time.Time{}}, // sin instante
		{Marcas{}, t0},          // nunca latio
		{Marcas{UltimoCiclo: t0.Add(72 * time.Hour)}, t0}, // marca futura
		{Marcas{UltimoCiclo: vivo}, t0},                   // late, canal apagado
		{Marcas{UltimoCiclo: hace(48 * time.Hour)}, t0},   // callado
	} {
		out = append(out, Vigilar(c.m, c.ahora))
	}
	// Y las cuatro formas del canal, con el planificador vivo.
	for _, m := range []Marcas{
		{UltimoCiclo: vivo, LatidoActivado: true},
		{UltimoCiclo: vivo, LatidoActivado: true, UltimoPulso: hace(time.Hour)},
		{UltimoCiclo: vivo, LatidoActivado: true, UltimoPulso: hace(48 * time.Hour)},
		{UltimoCiclo: vivo, LatidoActivado: true, UltimoPulso: hace(time.Hour),
			FalloElUltimoIntento: true},
	} {
		out = append(out, Vigilar(m, t0))
	}
	return out
}

// clavesDe saca las claves de catalogo que un veredicto pone en pantalla.
func clavesDe(p Planificador) []string {
	var out []string
	for _, c := range []string{p.Clave, p.Arreglo, p.Canal.Clave, p.Canal.Arreglo,
		p.Canal.Descargo} {
		if c != "" {
			out = append(out, c)
		}
	}
	return out
}

// Toda clave que la vigilancia emite esta publicada, y toda clave publicada la
// emite algun estado alcanzable. Las dos direcciones.
//
// Una sola no sirve. Si solo se comprobara que lo emitido esta publicado, la
// lista podria tener veinte claves inventadas que alguien tendria que traducir
// a cada idioma nuevo; si solo se comprobara lo contrario, la lista podria
// estar vacia y las claves saldrian en crudo en la pantalla de un cliente.
//
// Lo que esto NO demuestra, dicho para que conste: que estadosAlcanzables
// llegue a todos los estados. Un estado nuevo en Vigilar al que no llegue
// ninguna fila de arriba no lo caza esta comprobacion. Lo caza la de la
// direccion contraria en cuanto su clave se publique, y el inventario del
// catalogo, que lee del AST de este paquete todas las claves literales.
func TestTodaClaveQueLaVigilanciaEmiteEstaPublicada(t *testing.T) {
	publicadas := map[string]bool{}
	for _, c := range ClavesDelPlanificador() {
		publicadas[c] = true
	}
	if len(publicadas) != len(ClavesDelPlanificador()) {
		t.Error("ClavesDelPlanificador() trae repetidos")
	}

	emitidas := map[string]bool{}
	for _, p := range estadosAlcanzables() {
		for _, c := range clavesDe(p) {
			emitidas[c] = true
			if !publicadas[c] {
				t.Errorf("la vigilancia emite %q y ClavesDelPlanificador() no la publica: "+
					"saldria en crudo en la pantalla de un cliente", c)
			}
		}
	}
	var muertas []string
	for c := range publicadas {
		if !emitidas[c] {
			muertas = append(muertas, c)
		}
	}
	sort.Strings(muertas)
	if len(muertas) > 0 {
		t.Errorf("ClavesDelPlanificador() publica %v y no las emite ningun estado "+
			"alcanzable. O sobran, o falta el caso que las alcanza", muertas)
	}
}

// CONTROL NEGATIVO de lo de arriba. Sin esto, el verde no demuestra que se
// mire nada: dos listas vacias darian exactamente el mismo verde.
func TestLaComprobacionDeClavesCazaUnaClaveSinPublicar(t *testing.T) {
	publicadas := map[string]bool{}
	for _, c := range ClavesDelPlanificador() {
		publicadas[c] = true
	}
	// Un veredicto con una clave que nadie publico, que es exactamente lo
	// que produce anadir un estado y olvidar la lista.
	roto := Planificador{Clave: "aviso.planificador.inventada", Arreglo: ClaveArreglaNunca}
	var sin []string
	for _, c := range clavesDe(roto) {
		if !publicadas[c] {
			sin = append(sin, c)
		}
	}
	if len(sin) != 1 || sin[0] != "aviso.planificador.inventada" {
		t.Fatalf("la comprobacion no caza una clave sin publicar: %v. Mientras esto pase, "+
			"su verde sobre los estados de verdad no significa nada", sin)
	}
}

// Las claves de la vigilancia son claves, no texto. Misma regla que los titulos
// de pantalla: los rotulos de la interfaz los pone el catalogo.
func TestLasClavesDeLaVigilanciaSonClavesDeCatalogo(t *testing.T) {
	for _, c := range ClavesDelPlanificador() {
		if !strings.HasPrefix(c, "aviso.") && !strings.HasPrefix(c, "pantalla.") {
			t.Errorf("%q no esta en un espacio de claves de la interfaz", c)
		}
		if hayTexto(c) {
			t.Errorf("%q es texto, no una clave", c)
		}
	}
}

func hayTexto(s string) bool {
	for _, r := range s {
		if r == ' ' || r > 127 {
			return true
		}
	}
	return false
}

// La pantalla Hoy trae el veredicto y las otras cinco NO.
//
// Que las otras no lo lleven no es ahorro: un veredicto de vigilancia en cero
// tiene exactamente el mismo aspecto que un veredicto "correcto", asi que una
// pantalla que lo llevara sin querer estaria diciendo que todo va bien.
func TestSoloHoyTraeElEstadoDelPlanificador(t *testing.T) {
	ps := DerivarCon(nil, Entorno{Ahora: t0, Marcas: Marcas{UltimoCiclo: hace(time.Minute)}})
	for _, p := range ps {
		if p.ID == Hoy {
			if p.Planificador == nil {
				t.Fatal("Hoy sale sin el estado del planificador, que es la casilla entera")
			}
			if p.Planificador.Clave != ClavePlanificadorLate {
				t.Errorf("Hoy dice %q con un ciclo de hace un minuto", p.Planificador.Clave)
			}
			continue
		}
		if p.Planificador != nil {
			t.Errorf("la pantalla %q trae un veredicto de vigilancia y no es Hoy", p.ID)
		}
	}
}

// Derivar sin entorno no finge estar bien. Es la firma contra la que compila el
// codigo que todavia no pasa el instante, y lo que tiene que salir de ahi es
// "no se me ha pasado el instante", nunca "correcto".
func TestDerivarSinEntornoDiceQueLeFaltaElInstante(t *testing.T) {
	for _, p := range Derivar(nil) {
		if p.ID != Hoy {
			continue
		}
		if p.Planificador == nil {
			t.Fatal("Hoy sale sin veredicto")
		}
		if p.Planificador.Nivel == NivelCorrecto {
			t.Error("derivando sin entorno, Hoy dice que el planificador esta correcto")
		}
	}
}
