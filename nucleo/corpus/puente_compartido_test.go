package corpus

import (
	"encoding/json"
	"errors"
	"testing"
)

// EL PREDICADO QUE CRUZA DE PAQUETE.
//
// # Como salio, y es un rojo sobre dato real
//
// Escribiendo el puente de los veinte paquetes que faltaban. El atributo
// `sgsi.trata_datos_personales` de un paquete referencial afirma un predicado
// que las reglas de ESE paquete no usan y que leen, con aridad 1, las de tres
// paquetes distintos. El linter lo rechazaba y el arreglo que proponia su
// mensaje («declara que no llega al motor») era UNA MENTIRA MEDIDA: ese hecho
// enciende 8 obligaciones ajenas, contadas con el motor del producto.
//
// # Por que una bandera y no una excepcion del linter
//
// Porque un puente que solo funciona porque otro paquete esta instalado es un
// puente que se apaga el dia que ese paquete se desinstala, y eso hay que poder
// leerlo EN EL DATO. Aceptarlo en silencio (mirar el corpus entero siempre)
// habria borrado la unica diferencia que importa entre las dos situaciones.
//
// Las tres comprobaciones, y cada una tiene su caso aqui:
//
//	el valor cero (`compartido` ausente) manda la comprobacion dura de siempre;
//	`compartido` cierto exige que el PROPIO paquete no use el predicado;
//	`compartido` cierto exige que alguien del corpus si lo use, y eso no cabe
//	mirando un paquete: vive en ValidarPuenteEntrePaquetes.

// TestUnPuenteQueCruzaDePaqueteSeDeclaraYCarga es el caso que desbloqueo el
// trabajo: sin la bandera, esto no cargaba.
func TestUnPuenteQueCruzaDePaqueteSeDeclaraYCarga(t *testing.T) {
	// El paquete afirma `trata_datos_personales` con aridad 1 y sus propias
	// reglas no lo usan: solo hablan de `categoria`.
	p := conPuenteBooleano(&HechoDeAtributo{
		Forma: PuenteAfirmaSi, Predicado: "trata_datos_personales", Compartido: true,
	}, `aplica("demo.auditoria_bienal", S) :- categoria(S, "ALTA")`)

	if errs := p.Validar(); tieneError(errs, ErrPuentePredicadoHuerfano) {
		t.Fatalf("un puente que DECLARA que cruza de paquete no puede caerse por huerfano: "+
			"eso es lo que la bandera existe para decir.\n  errores: %v", errs)
	}
}

// CONTROL NEGATIVO 1, y es el que de verdad vigila: SIN la bandera, el mismo
// puente se sigue cayendo. Si este caso pasara, `compartido` no estaria
// relajando nada, estaria abierta la puerta para todos.
func TestSinLaBanderaElPuenteQueCruzaDePaqueteSigueCayendose(t *testing.T) {
	p := conPuenteBooleano(&HechoDeAtributo{
		Forma: PuenteAfirmaSi, Predicado: "trata_datos_personales",
	}, `aplica("demo.auditoria_bienal", S) :- categoria(S, "ALTA")`)

	if errs := p.Validar(); !tieneError(errs, ErrPuentePredicadoHuerfano) {
		t.Fatalf("sin `compartido`, un predicado que ninguna regla del paquete usa tiene que "+
			"caerse, y no se cayo. El valor cero de la bandera es el RESTRICTIVO (invariante "+
			"8): si el cero deja pasar, la comprobacion dura ha dejado de existir para "+
			"todo el corpus.\n  errores: %v", errs)
	}
}

// CONTROL NEGATIVO 2: LA BANDERA VIEJA. Si el propio paquete SI usa el
// predicado, `compartido` afirma algo que no es, y da igual si sobra porque
// alguien la puso de mas o porque alguien escribio la regla despues y no la
// quito: en los dos casos el dato miente.
//
// Es la direccion que se olvida, y sin ella la bandera seria un interruptor
// para saltarse la comprobacion.
func TestLaBanderaDeCompartidoSobraSiElPropioPaqueteUsaElPredicado(t *testing.T) {
	p := conPuenteBooleano(&HechoDeAtributo{
		Forma: PuenteAfirmaSi, Predicado: "trata_datos_personales", Compartido: true,
	}, `aplica("demo.auditoria_bienal", S) :- trata_datos_personales(S)`)

	if errs := p.Validar(); !tieneError(errs, ErrPuenteCompartidoSobrante) {
		t.Fatalf("el paquete dice que su predicado lo leen otros y lo lee el suyo: la bandera "+
			"es vieja o falsa y tiene que caerse.\n  errores: %v", errs)
	}
}

// CONTROL NEGATIVO 3: un callejon no comparte nada, porque no afirma nada.
func TestUnCallejonNoPuedeDeclararseCompartido(t *testing.T) {
	p := conPuenteBooleano(&HechoDeAtributo{
		Forma: PuenteNoLlegaAlMotor, Porque: "alimenta el reloj, no la regla", Compartido: true,
	}, `aplica("demo.auditoria_bienal", S) :- categoria(S, "ALTA")`)

	if errs := p.Validar(); !tieneError(errs, ErrPuentePredicadoSobrante) {
		t.Fatalf("un callejon con `compartido` tiene que caerse: no hay predicado que "+
			"compartir.\n  errores: %v", errs)
	}
}

// ---------------------------------------------------------------------------
// LA SEGUNDA PASADA: que alguien lo lea de verdad
// ---------------------------------------------------------------------------

// TestUnPuenteCompartidoQueNadieLeeNoPasaLaSegundaPasada es la otra direccion
// del emparejamiento, y sin ella la bandera seria una via para afirmar
// cualquier predicado inventado.
func TestUnPuenteCompartidoQueNadieLeeNoPasaLaSegundaPasada(t *testing.T) {
	emisor := conPuenteBooleano(&HechoDeAtributo{
		Forma: PuenteAfirmaSi, Predicado: "predicado_que_no_lee_nadie", Compartido: true,
	}, `aplica("demo.auditoria_bienal", S) :- categoria(S, "ALTA")`)
	otro := base()
	otro.URN = "urn:demo:otro"
	otro.Aplicabilidad.Reglas = []ReglaSpec{{
		ID: "otro.r", Cita: "RD 311/2022 art. 40, para el fixture",
		Regla: `aplica("demo.auditoria_bienal", S) :- trata_datos_personales(S)`,
	}}

	errs := ValidarPuenteEntrePaquetes([]*Paquete{emisor, otro})
	if !tieneError(errs, ErrPuenteCompartidoNadieLoLee) {
		t.Fatalf("el puente afirma un predicado que NINGUN paquete del corpus lee: la "+
			"respuesta del operador no derivaria nada y no daria error en ningun sitio, que "+
			"es el agujero entero que este bloque cierra.\n  errores: %v", errs)
	}
}

// CONTROL POSITIVO de la segunda pasada: cuando el otro paquete SI lo lee, con
// la aridad que le toca, no se dice nada.
//
// Sin este caso, un `return error` incondicional pasaria el test de arriba, y
// entonces `compartido` seria inutilizable: la puerta rechazaria justo el corpus
// que existe para permitir.
func TestUnPuenteCompartidoQueOtroPaqueteSiLeePasa(t *testing.T) {
	emisor := conPuenteBooleano(&HechoDeAtributo{
		Forma: PuenteAfirmaSi, Predicado: "trata_datos_personales", Compartido: true,
	}, `aplica("demo.auditoria_bienal", S) :- categoria(S, "ALTA")`)
	otro := base()
	otro.URN = "urn:demo:otro"
	otro.Aplicabilidad.Reglas = []ReglaSpec{{
		ID: "otro.r", Cita: "RD 311/2022 art. 40, para el fixture",
		Regla: `aplica("demo.auditoria_bienal", S) :- trata_datos_personales(S)`,
	}}

	if errs := ValidarPuenteEntrePaquetes([]*Paquete{emisor, otro}); len(errs) > 0 {
		t.Fatalf("otro paquete lee el predicado con aridad 1, que es la que el atributo "+
			"afirma: no hay nada que decir.\n  errores: %v", errs)
	}
}

// CONTROL NEGATIVO 4: LA ARIDAD SIGUE CONTANDO AL CRUZAR. Que alguien use el
// nombre del predicado no basta si lo usa con otro numero de argumentos: el
// hecho no casaria igual, y el fallo seria igual de silencioso.
func TestUnPuenteCompartidoConOtraAridadNoPasa(t *testing.T) {
	emisor := conPuenteBooleano(&HechoDeAtributo{
		Forma: PuenteAfirmaSi, Predicado: "trata_datos_personales", Compartido: true,
	}, `aplica("demo.auditoria_bienal", S) :- categoria(S, "ALTA")`)
	otro := base()
	otro.URN = "urn:demo:otro"
	otro.Aplicabilidad.Reglas = []ReglaSpec{{
		ID: "otro.r", Cita: "RD 311/2022 art. 40, para el fixture",
		// Aqui se usa con DOS argumentos, y el atributo afirma con uno.
		Regla: `aplica("demo.auditoria_bienal", S) :- trata_datos_personales(S, "si")`,
	}}

	if errs := ValidarPuenteEntrePaquetes([]*Paquete{emisor, otro}); !tieneError(
		errs, ErrPuenteCompartidoNadieLoLee) {
		t.Fatalf("nadie lo usa con la aridad que el atributo afirma: tiene que caerse.\n"+
			"  errores: %v", errs)
	}
}

// conPuenteBooleano es como conPuente pero deja el atributo en BOOLEANO, que es
// lo que exigen `afirma_si` y `afirma_si_valor`. El fixture base trae un
// enumerado, y con el las formas booleanas se caen por tipo antes de llegar a
// la comprobacion que estos casos miden.
func conPuenteBooleano(h *HechoDeAtributo, reglas ...string) *Paquete {
	p := conPuente(h, reglas...)
	p.Entidades[0].Atributos[0].Tipo = Booleano
	p.Entidades[0].Atributos[0].Valores = nil
	return p
}

// ---------------------------------------------------------------------------
// EL CABLE: que la segunda pasada este ENCHUFADA a Cargar
// ---------------------------------------------------------------------------

// TestCargarRechazaUnPuenteCompartidoQueNadieLee existe porque una mutacion lo
// pidio a gritos.
//
// Con los siete casos de arriba puestos y verdes, se corto el cable dentro de
// `Cargar` y LA SUITE ENTERA SIGUIO VERDE, incluida la del corpus real. O sea
// que la comprobacion existia, se probaba llamandola a mano, y el producto
// podia dejar de llamarla sin que nadie se enterara: la rama que nunca se
// ejecuta, en el sitio donde mas cara sale.
//
// Este caso llama a `Cargar` de verdad, sobre un corpus de dos paquetes en
// disco, que es como lo llama el producto.
func TestCargarRechazaUnPuenteCompartidoQueNadieLee(t *testing.T) {
	dir := t.TempDir()
	escribirJSON(t, dir, "emisor", conPuenteBooleano(&HechoDeAtributo{
		Forma: PuenteAfirmaSi, Predicado: "predicado_que_no_lee_nadie", Compartido: true,
	}, `aplica("demo.auditoria_bienal", S) :- categoria(S, "ALTA")`))
	escribirJSON(t, dir, "lector", lectorDe("trata_datos_personales"))

	_, err := Cargar(dir)
	if err == nil {
		t.Fatal("Cargar acepta un corpus con un puente compartido que nadie lee. La " +
			"comprobacion existe y NO ESTA ENCHUFADA, que es peor que no tenerla: quien " +
			"lea el codigo dara por hecho que se mira")
	}
	if !errors.Is(err, ErrPuenteCompartidoNadieLoLee) {
		t.Fatalf("Cargar se cae, pero por otra cosa: el corpus tiene que rechazarse por el "+
			"puente y no de rebote.\n  error: %v", err)
	}
}

// CONTROL POSITIVO DEL CABLE: el mismo corpus con un lector de verdad CARGA.
//
// Sin este caso, cortar el cable y ademas romper cualquier otra cosa del
// fixture daria el mismo rojo, y el de arriba no estaria midiendo el puente.
func TestCargarAceptaUnPuenteCompartidoQueOtroPaqueteLee(t *testing.T) {
	dir := t.TempDir()
	escribirJSON(t, dir, "emisor", conPuenteBooleano(&HechoDeAtributo{
		Forma: PuenteAfirmaSi, Predicado: "trata_datos_personales", Compartido: true,
	}, `aplica("demo.auditoria_bienal", S) :- categoria(S, "ALTA")`))
	escribirJSON(t, dir, "lector", lectorDe("trata_datos_personales"))

	if _, err := Cargar(dir); err != nil {
		t.Fatalf("el otro paquete lee el predicado con la aridad que le toca: este corpus "+
			"tiene que cargar.\n  error: %v", err)
	}
}

// lectorDe devuelve un paquete cuyo unico papel es LEER ese predicado con
// aridad 1. Su URN es distinto del emisor a proposito: dos paquetes con el
// mismo urn no cargan, y ese rojo se confundiria con el que se busca.
func lectorDe(pred string) *Paquete {
	p := base()
	p.URN = "urn:demo:lector"
	p.Aplicabilidad.Reglas = []ReglaSpec{{
		ID: "lector.r", Cita: "RD 311/2022 art. 40, para el fixture",
		Regla: `aplica("demo.auditoria_bienal", S) :- ` + pred + `(S)`,
	}}
	return p
}

// escribirJSON serializa un fixture al disco con el mismo formato que un
// paquete de verdad. Se serializa el TIPO en vez de escribir el JSON a mano
// porque un JSON escrito a mano es una segunda implementacion del formato: el
// dia que el formato cambie, el fixture seguiria midiendo el formato viejo.
func escribirJSON(t *testing.T, raiz, nombre string, p *Paquete) {
	t.Helper()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	escribirPaquete(t, raiz, nombre, string(b))
}
