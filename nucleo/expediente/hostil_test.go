package expediente

import (
	"testing"

	"dutiq/nucleo/aplicabilidad"
)

// Revision hostil de la etapa 1 sobre el expediente.

// ATAQUE 10, el gordo. El campo AnclasDeConfianza lleva este comentario:
//
//	"son los digests de paquete que el RECEPTOR acepta, obtenidos del registro
//	 firmado, no del expediente"
//
// y encima una nota que dice que sin esto la verificacion era circular y que
// una revision anterior lo arreglo. Pero el campo esta en el struct con su
// etiqueta JSON, viaja DENTRO del fichero, y Verificar(e) no recibe nada mas
// que el expediente. O sea que las anclas las sigue poniendo el emisor.
//
// Si es asi, el emisor puede inventarse una obligacion, adjuntar la regla que
// la deriva, recalcular el digest del paquete y escribir el ancla que cuadra.
func TestHostilElEmisorSeFabricaSusPropiasAnclas(t *testing.T) {
	e := construirExpediente(t)

	// Es la misma jugada que TestDetectaObligacionInventadaConSuPropiaRegla,
	// con una diferencia: la obligacion inventada se mete DENTRO de un paquete
	// que ya esta declarado y anclado, en vez de traerse un paquete nuevo. Asi
	// la comprobacion de "programa de un paquete no reconocido" no salta.
	const urn = "ens@2022.311"
	for i := range e.Programas {
		if e.Programas[i].Paquete != urn {
			continue
		}
		e.Programas[i].Reglas = append(e.Programas[i].Reglas, aplicabilidad.Regla{
			ID: "inventada", Cita: "Anexo II op.exp.99",
			Cabeza: aplicabilidad.A("aplica", aplicabilidad.C("ens.inventada"), aplicabilidad.V("E")),
			Cuerpo: []aplicabilidad.Atomo{aplicabilidad.A("responsable", aplicabilidad.V("E"))},
		})
	}
	e.Obligaciones = append(e.Obligaciones, Obligacion{
		ID: "ens.inventada", Paquete: urn, Articulo: "Anexo II op.exp.99", Primitiva: "continua",
		Control: "ens.op.acc.5.mfa", Afirmacion: "obligacion que no existe en el BOE",
	})
	e.Aplicables = append(e.Aplicables, "ens.inventada")

	// Y ahora lo que cierra el circulo: recalcula el digest de su propio
	// paquete y se escribe a si mismo el ancla que cuadra.
	for i, p := range e.Paquetes {
		calc := DigestPaquete(p.URN, e.Programas, e.Obligaciones)
		e.Paquetes[i].Digest = calc
		e.AnclasDeConfianza[p.URN] = calc
	}

	b, err := e.Guardar()
	if err != nil {
		t.Fatal(err)
	}
	otro, err := Cargar(b)
	if err != nil {
		t.Fatal(err)
	}
	inf := Verificar(otro)
	if !inf.Valido {
		t.Logf("PROPIEDAD AGUANTA, lo caza por: %v", inf.Discrepancias)
		return
	}
	t.Fatal("HALLAZGO: el emisor se ha inventado la obligacion ens.inventada dentro de un paquete " +
		"anclado, ha aportado la regla que la deriva, ha recalculado el digest y se ha escrito el " +
		"ancla que cuadra, y el expediente verifica limpio. AnclasDeConfianza viaja dentro del " +
		"fichero y Verificar() no admite ninguna fuente externa, asi que la comprobacion " +
		"anti-circular esta comparando al emisor consigo mismo")
}

// ATAQUE 11. Las dos comprobaciones anti-circulares (la de anclas vacias y la
// de paquete no reconocido) se tapan la una a la otra: quitar cualquiera de
// las dos no rompe ningun test. Aqui se comprueba el caso en que la segunda no
// puede cubrir a la primera, un expediente sin paquetes.
func TestHostilExpedienteSinPaquetesNiAnclas(t *testing.T) {
	e := construirExpediente(t)
	e.Paquetes = nil
	e.AnclasDeConfianza = nil
	b, _ := e.Guardar()
	otro, _ := Cargar(b)
	inf := Verificar(otro)
	if !inf.Valido {
		t.Logf("PROPIEDAD AGUANTA: %d discrepancias", len(inf.Discrepancias))
		return
	}
	t.Fatal("HALLAZGO: un expediente sin paquetes y sin anclas verifica")
}
