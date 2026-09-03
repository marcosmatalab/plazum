package corpus

import "testing"

// LA TERCERA DIRECCION DEL EMPAREJAMIENTO DEL PUENTE: EL VALOR.
//
// # Como salio, y por que no salio de leer el diff
//
// Escribiendo el puente de los catorce paquetes que faltan. El bloque `hecho`
// se recorre en las dos direcciones por el NOMBRE del predicado y por la
// ARIDAD, y las dos comprobaciones son ciertas. La pregunta del invariante 7 es
// si hay alguna direccion que no se recorra en ninguna de las dos, y la habia:
// **el VALOR**.
//
// Un atributo enumerado puede declarar el predicado bueno, con la aridad buena,
// y unos valores que ninguna regla prueba en ese hueco. El hecho se afirma, no
// casa con ninguna regla, y no da error en ningun sitio. Es exactamente el
// estado del que `hecho` venia a sacarnos, un escalon mas abajo.
//
// # ESTA PUERTA NACIO VERDE SOBRE EL CORPUS ENTERO, y se dice
//
// Se estreno contra `paquetes/` y no rechazo nada, porque hoy solo un paquete
// declara el puente. Una puerta que nace verde sobre el corpus real o vigila
// poco o llega tarde, y aqui es lo segundo: llega ANTES de que se escriban los
// catorce puentes que faltan, que es cuando iba a hacer falta. Se deja dicho
// para que nadie lea el verde como una medida.
//
// Lo que si se vio fallar, sobre datos reales y no sobre una mutacion: una
// sonda que declaraba `estado_certificacion` (no_certificado, en_certificacion,
// certificado) afirmando el predicado `adopta`, cuyas reglas solo prueban el
// identificador del marco. Pasaba el linter entero antes y lo rechaza ahora.

// conPuente devuelve el paquete base con el puente declarado en su unico
// atributo y con las reglas que se le pasen.
func conPuente(h *HechoDeAtributo, reglas ...string) *Paquete {
	p := base()
	p.Entidades[0].Atributos[0].Hecho = h
	for i, r := range reglas {
		p.Aplicabilidad.Reglas = append(p.Aplicabilidad.Reglas, ReglaSpec{
			ID:    "demo.r" + string(rune('a'+i)),
			Cita:  "RD 311/2022 art. 40, para el fixture",
			Regla: r,
		})
	}
	return p
}

func TestUnPuenteConValoresQueNingunaReglaMiraNoCarga(t *testing.T) {
	// El predicado casa (`categoria`), la aridad casa (2), y los valores del
	// atributo son BASICA, MEDIA y ALTA mientras la regla prueba "ALTISIMA".
	p := conPuente(&HechoDeAtributo{Forma: PuenteConValor, Predicado: "categoria"},
		`aplica("demo.auditoria_bienal", S) :- categoria(S, "ALTISIMA")`)

	errs := p.Validar()
	if !tieneError(errs, ErrPuenteValorHuerfano) {
		t.Fatalf("un puente cuyos valores no prueba ninguna regla tiene que caerse, y no se "+
			"cayo. El nombre del predicado y la aridad casan, asi que las dos comprobaciones "+
			"viejas dan verde: si esta no salta, el hecho se afirma y no casa con nada.\n"+
			"  errores que dio: %v", errs)
	}
}

// CONTROL NEGATIVO 1: cuando los valores SI casan, no se dice nada.
//
// Sin este caso, un `return error` incondicional pasaria el test de arriba y la
// puerta seria un rechazador de todo.
func TestUnPuenteConValoresQueLasReglasSiMiranCarga(t *testing.T) {
	p := conPuente(&HechoDeAtributo{Forma: PuenteConValor, Predicado: "categoria"},
		`aplica("demo.auditoria_bienal", S) :- categoria(S, "ALTA")`)

	if errs := p.Validar(); tieneError(errs, ErrPuenteValorHuerfano) {
		t.Fatalf("los valores del atributo incluyen ALTA y la regla prueba ALTA: no hay nada "+
			"que decir, y la puerta ha hablado.\n  errores: %v", errs)
	}
}

// CONTROL NEGATIVO 2, Y ES EL QUE HABRIA ROTO EL CORPUS REAL.
//
// Cuando alguna regla usa ese hueco con una VARIABLE, la regla acepta cualquier
// valor y no hay solapamiento que exigir. Es el caso de los niveles del ENS:
// `nivel_disponibilidad(X, N)` no nombra BAJO ni ALTO en ninguna parte, los
// compara despues. Sin esta rama, esta puerta habria rechazado once atributos
// del unico paquete que hoy declara el puente, y el arreglo barato habria sido
// apagarla.
func TestUnPuenteCuyoHuecoAlgunaReglaUsaConVariableNoExigeSolapamiento(t *testing.T) {
	p := conPuente(&HechoDeAtributo{Forma: PuenteConValor, Predicado: "categoria"},
		`nivel_max(S, N) :- categoria(S, N)`,
		`aplica("demo.auditoria_bienal", S) :- nivel_max(S, "CUALQUIERA")`)

	if errs := p.Validar(); tieneError(errs, ErrPuenteValorHuerfano) {
		t.Fatalf("alguna regla usa el hueco con una variable, asi que acepta cualquier valor "+
			"y no hay nada que exigir. La puerta ha rechazado corpus correcto, que es como "+
			"acaba apagada.\n  errores: %v", errs)
	}
}

// CONTROL NEGATIVO 3: el rodeo legitimo sigue cabiendo.
//
// Un enumerado de UN SOLO valor que es el que la regla prueba es hoy la unica
// forma de expresar «un si que afirma pred(instancia, CONSTANTE)», y aunque es
// un rodeo feo (ver docs/hallazgos-puente.md) es correcto. Esta puerta no puede
// cerrarlo de paso.
func TestElRodeoDelEnumeradoDeUnSoloValorSigueCabiendo(t *testing.T) {
	p := base()
	p.Entidades[0].Atributos[0].Valores = []string{"BASICA"}
	p.Entidades[0].Atributos[0].Hecho = &HechoDeAtributo{
		Forma: PuenteConValor, Predicado: "categoria",
	}
	p.Aplicabilidad.Reglas = append(p.Aplicabilidad.Reglas, ReglaSpec{
		ID: "demo.ra", Cita: "RD 311/2022 art. 40, para el fixture",
		Regla: `aplica("demo.auditoria_bienal", S) :- categoria(S, "BASICA")`,
	})
	if errs := p.Validar(); tieneError(errs, ErrPuenteValorHuerfano) {
		t.Fatalf("el enumerado de un solo valor que la regla prueba es correcto y la puerta "+
			"lo ha rechazado.\n  errores: %v", errs)
	}
}

// CONTROL NEGATIVO 4: un valor que APAGA no tiene que aparecer en ninguna regla.
//
// Basta con que UNO de los valores case, y no se exigen todos a proposito: el
// `ambito` del ENS tiene un valor que enciende obligaciones del sector publico
// y otro que enciende las del privado, y un enumerado puede perfectamente tener
// un valor que solo sirve para NO afirmar lo que afirman los otros.
func TestUnValorQueNingunaReglaPruebaNoRompeSiOtroSiCasa(t *testing.T) {
	p := conPuente(&HechoDeAtributo{Forma: PuenteConValor, Predicado: "categoria"},
		`aplica("demo.auditoria_bienal", S) :- categoria(S, "ALTA")`)
	// BASICA y MEDIA no las prueba ninguna regla, y eso es legitimo.
	if errs := p.Validar(); tieneError(errs, ErrPuenteValorHuerfano) {
		t.Fatalf("basta con que uno de los valores case y ALTA casa. La puerta esta exigiendo "+
			"que casen todos, y eso rechaza corpus correcto.\n  errores: %v", errs)
	}
}
