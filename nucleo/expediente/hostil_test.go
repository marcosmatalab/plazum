package expediente

import (
	"testing"
)

// Ataques de la revision hostil de la etapa 1 sobre el expediente. Se
// escribieron en rojo, como hallazgos, y se quedan como regresion.

// ATAQUE 10, el bloqueante. El emisor se inventa una obligacion DENTRO de un
// paquete ya anclado, aporta la regla que la deriva, recalcula el digest y se
// escribe el ancla que cuadra. Antes verificaba limpio, porque las anclas
// viajaban en el propio fichero y la comprobacion anti-circular comparaba al
// emisor consigo mismo.
func TestHostilElEmisorYaNoSeFabricaSusPropiasAnclas(t *testing.T) {
	e := construirExpediente(t)
	// El contexto del receptor se toma ANTES del sabotaje: es lo que el auditor
	// tiene en su registro, y el emisor no puede tocarlo.
	ctx := contextoDePrueba(t, e)

	const urn = "urn:demo:agregada"
	for i := range e.Programas {
		if e.Programas[i].Paquete != urn {
			continue
		}
		e.Programas[i].Reglas = append(e.Programas[i].Reglas,
			regla(t, "inventada", "Anexo II op.exp.99",
				`aplica(demo.inventada, E) :- responsable(E)`))
	}
	e.Obligaciones = append(e.Obligaciones, Obligacion{
		ID: "demo.inventada", Paquete: urn, Articulo: "Anexo II op.exp.99", Primitiva: "continua",
		Control: "demo.mfa", Afirmacion: "obligacion que no existe en el BOE",
	})
	e.Aplicables = append(e.Aplicables, "demo.inventada")

	// Y se reescribe a si mismo el digest y el ancla declarada.
	for i, p := range e.Paquetes {
		calc := DigestPaquete(p.URN, e.Programas, e.Obligaciones)
		e.Paquetes[i].Digest = calc
		e.AnclasDeclaradas[p.URN] = calc
	}

	b, err := e.Guardar()
	if err != nil {
		t.Fatal(err)
	}
	otro, err := Cargar(b)
	if err != nil {
		t.Fatal(err)
	}
	inf := Verificar(otro, ctx)
	if inf.Valido {
		t.Fatal("el emisor se ha inventado una obligacion y se ha escrito el ancla que cuadra: " +
			"contra las anclas del RECEPTOR eso no puede verificar")
	}
	t.Logf("detectado: %v", inf.Discrepancias)
}

// Control negativo del anterior: sin sabotaje, el mismo expediente verifica
// contra el mismo contexto. Sin esto, el test de arriba pasaria aunque la
// verificacion rechazara todo por cualquier motivo.
func TestHostilSinSabotajeElExpedienteVerifica(t *testing.T) {
	e := construirExpediente(t)
	inf := Verificar(e, contextoDePrueba(t, e))
	if !inf.Valido {
		t.Fatalf("el expediente limpio tiene que verificar: %v", inf.Discrepancias)
	}
}

// Y la discrepancia entre lo que el emisor DECLARA haber usado y lo que el
// receptor tiene en su registro se informa, en vez de callarse.
func TestHostilElAnclaDeclaradaQueNoCuadraSeInforma(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	e.AnclasDeclaradas["urn:demo:agregada"] = "sha256:otra-cosa"

	inf := Verificar(e, ctx)
	var visto bool
	for _, d := range inf.Discrepancias {
		if d.Que == "ancla declarada de urn:demo:agregada" {
			visto = true
		}
	}
	if !visto {
		t.Fatalf("una diferencia entre lo declarado y el registro del receptor tiene que verse: %v",
			inf.Discrepancias)
	}
}

// ATAQUE 11. Un expediente sin paquetes y un receptor sin anclas: la
// verificacion tiene que declararse circular en vez de dar un visto bueno.
func TestHostilSinAnclasDelReceptorNoHayVistoBueno(t *testing.T) {
	e := construirExpediente(t)
	inf := Verificar(e, ContextoReceptor{})
	if inf.Valido {
		t.Fatal("sin nada que el receptor aporte no se puede verificar nada")
	}
}

// El emisor oculta una entrada de la cadena: ni divulga su clave ni pone
// lapida. Antes no habia forma de notarlo porque el ledger iba en claro.
func TestHostilEntradaOcultaSinLapidaSeDetecta(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	delete(e.ClavesEntradas, 1) // la observacion incomoda

	inf := Verificar(e, ctx)
	if inf.Valido {
		t.Fatal("una entrada sin clave y sin lapida es contenido oculto sin justificar")
	}
	var visto bool
	for _, d := range inf.Discrepancias {
		if d.Que == "entrada 1 de la cadena" {
			visto = true
		}
	}
	if !visto {
		t.Fatalf("la discrepancia tiene que senalar la entrada oculta: %v", inf.Discrepancias)
	}
}

// Y una clave divulgada que no abre su entrada tampoco cuela: es donde el
// compromiso de clave deja de ser teoria.
func TestHostilClaveDivulgadaQueNoAbreSeDetecta(t *testing.T) {
	e := construirExpediente(t)
	ctx := contextoDePrueba(t, e)
	e.ClavesEntradas[1] = e.ClavesEntradas[0] // la clave de otra entrada

	inf := Verificar(e, ctx)
	if inf.Valido {
		t.Fatal("una clave que no compromete esa entrada no puede darse por buena")
	}
}
