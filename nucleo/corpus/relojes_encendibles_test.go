package corpus

import (
	"errors"
	"strings"
	"testing"
)

// EL RELOJ QUE NADIE PUEDE ENCENDER.
//
// Un paquete podia declarar `temporalidad` y no declarar ninguna regla que
// derivara `aplica()` sobre esa obligacion. El paquete cargaba, el linter
// callaba, los dorados pasaban (un dorado ejecuta el reloj DIRECTAMENTE, sin
// pasar por la aplicabilidad) y el reloj no se encendia para nadie: `aplica(O,
// S)` es el unico predicado por el que una obligacion llega a un sujeto, y lo
// consultan expediente.go, explain.go, motor.go y demo.go. El resultado no era
// un error, era SILENCIO, y el silencio se lee como "no me toca".
//
// MEDIDO ANTES DE ESCRIBIR LA PUERTA, y por eso hay dos granos y no uno: de los
// trece paquetes con reloj del corpus publicado, LOS TRECE declaraban alguna
// regla (o sea, los trece pasaban el grano grueso) y CUATRO tenian relojes que
// ninguna regla alcanzaba: dora, nis1-es, psd2-es y ens, siete obligaciones en
// total. Una puerta que la deuda que ya tienes delante aprueba no es una
// puerta: es un adorno.

func conTemporalidad(p *Paquete) *Paquete {
	p.Obligaciones[0].Temporalidad = &Temporalidad{
		Primitiva: "periodica", Cadencia: "P24M",
		Regimen:            RegimenSpec{Computo: "naturales", Cierre: "fin_de_dia"},
		OrigenDelIntervalo: IntervaloSueloLegal,
		CitaDelIntervalo:   "RD 311/2022, art. 31.1: auditoria ordinaria AL MENOS CADA DOS ANOS (fixture)",
	}
	return p
}

func reglaQueEnciende(id string) ReglaSpec {
	return ReglaSpec{ID: "enciende", Cita: "art. 2.1", Regla: `aplica("` + id + `", S) :- en_ambito(S)`}
}

func hay(errs []error, centinela error) bool {
	for _, e := range errs {
		if errors.Is(e, centinela) {
			return true
		}
	}
	return false
}

// LAS DOS FORMAS DE LA NADA (invariante 8). `Reglas` a nil y `Reglas` a lista
// vacia presente son dos cosas distintas en Go y en JSON: la primera sale de
// omitir el campo, la segunda de escribir `"reglas": []`. Las dos tienen que
// caer del mismo lado, y el lado tiene que ser el restrictivo. Se recorren por
// separado a proposito: comprobar solo `nil` es comprobar la que no se olvida.
func TestUnPaqueteConRelojYSinReglasNoCarga(t *testing.T) {
	for _, c := range []struct {
		nombre string
		reglas []ReglaSpec
	}{
		{"campo ausente (nil)", nil},
		{"lista vacia presente", []ReglaSpec{}},
	} {
		t.Run(c.nombre, func(t *testing.T) {
			p := conTemporalidad(base())
			p.Aplicabilidad = Aplicabilidad{Reglas: c.reglas}
			errs := p.Validar()
			if !hay(errs, ErrRelojSinAplicabilidad) {
				t.Fatalf("un paquete con reloj y sin reglas tiene que caerse con "+
					"ErrRelojSinAplicabilidad y no lo hace: %v", errs)
			}
		})
	}
}

// EL GRANO FINO, que es el que muerde. El grano grueso se cumple escribiendo
// una regla de ambito que no enciende nada, que es exactamente lo que hacian
// dora, nis1-es y psd2-es: `en_ambito(E) :- designado(E, "...")` y ni una sola
// `aplica`. Sin esta comprobacion, la puerta habria dado verde sobre la deuda.
func TestUnRelojQueNingunaReglaAlcanzaNoCarga(t *testing.T) {
	p := conTemporalidad(base())
	p.Aplicabilidad = Aplicabilidad{Reglas: []ReglaSpec{{
		ID: "solo_ambito", Cita: "art. 2.1",
		Regla: `en_ambito(S) :- designado(S, "lo_que_sea")`,
	}}}
	errs := p.Validar()
	if !hay(errs, ErrRelojQueNadieEnciende) {
		t.Fatalf("una regla de ambito no enciende ningun reloj, y el linter tiene que decirlo: %v", errs)
	}
	// Y no basta con que se caiga: tiene que caerse NOMBRANDO la obligacion,
	// porque el autor necesita saber cual de sus relojes esta muerto.
	var msg string
	for _, e := range errs {
		if errors.Is(e, ErrRelojQueNadieEnciende) {
			msg = e.Error()
		}
	}
	if !strings.Contains(msg, "demo.auditoria_bienal") {
		t.Fatalf("el error no dice que obligacion es: %q", msg)
	}
}

// CONTROL NEGATIVO 1: con la regla que le corresponde, calla.
func TestUnRelojConSuReglaCarga(t *testing.T) {
	p := conTemporalidad(base())
	p.Aplicabilidad = Aplicabilidad{Reglas: []ReglaSpec{reglaQueEnciende("demo.auditoria_bienal")}}
	for _, e := range p.Validar() {
		if errors.Is(e, ErrRelojSinAplicabilidad) || errors.Is(e, ErrRelojQueNadieEnciende) {
			t.Fatalf("un reloj con su regla no puede dar fallo de encendido: %v", e)
		}
	}
}

// CONTROL NEGATIVO 2: una regla que enciende OTRA obligacion no vale. Es el
// fallo que un emparejamiento por posicion (o por "hay alguna aplica") dejaria
// pasar: hay una `aplica`, pero no es la de este reloj.
func TestUnaAplicaDeOtraObligacionNoEnciendeEstReloj(t *testing.T) {
	p := conTemporalidad(base())
	p.Obligaciones = append(p.Obligaciones, Obligacion{
		ID: "demo.otra", Articulo: "32", ClaseE2E: "procedimental",
		TextoLegal: "Otra cosa.", Cita: "RD 311/2022 art. 32",
		Vigencia: Vigencia{Desde: "2022-05-05"}, Recursos: []TipoRecurso{"Sistema"},
	})
	p.Aplicabilidad = Aplicabilidad{Reglas: []ReglaSpec{reglaQueEnciende("demo.otra")}}
	if !hay(p.Validar(), ErrRelojQueNadieEnciende) {
		t.Fatal("una aplica() de otra obligacion no enciende este reloj, y el linter tiene que verlo")
	}
}

// CONTROL NEGATIVO 3: una obligacion SIN reloj no exige nada. La puerta vigila
// relojes, no obligaciones; si se pusiera roja aqui estaria pidiendo una regla
// por cada control de un catalogo de 129 y seria imposible de cumplir.
func TestUnaObligacionSinRelojNoExigeRegla(t *testing.T) {
	p := base() // sin temporalidad
	p.Aplicabilidad = Aplicabilidad{}
	for _, e := range p.Validar() {
		if errors.Is(e, ErrRelojSinAplicabilidad) || errors.Is(e, ErrRelojQueNadieEnciende) {
			t.Fatalf("sin reloj no hay nada que encender: %v", e)
		}
	}
}

// LA VIA DE ESCAPE, dicha en voz alta: una cabeza con VARIABLE puede derivar
// cualquier id, asi que apaga la comprobacion fina. Se prueba para que quede
// escrito que existe y no se descubra de casualidad.
func TestUnaAplicaConCabezaVariableApagaLaComprobacion(t *testing.T) {
	p := conTemporalidad(base())
	p.Aplicabilidad = Aplicabilidad{Reglas: []ReglaSpec{{
		ID: "comodin", Cita: "art. 2.1", Regla: `aplica(O, S) :- obligacion_del_paquete(O), en_ambito(S)`,
	}}}
	if hay(p.Validar(), ErrRelojQueNadieEnciende) {
		t.Fatal("una cabeza con variable cubre cualquier obligacion: no puede dar reloj muerto")
	}
}
