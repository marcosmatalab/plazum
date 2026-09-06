package evidencia_test

import (
	"strings"
	"testing"

	"github.com/marcosmatalab/plazum/adaptadores/busqueda"
	"github.com/marcosmatalab/plazum/adaptadores/evidencia"
	"github.com/marcosmatalab/plazum/adaptadores/ia"
	"github.com/marcosmatalab/plazum/adaptadores/ingesta"
	"github.com/marcosmatalab/plazum/nucleo/corpus"
)

// LA MEDIDA CONTRA EL CORPUS REAL, que es donde esta pieza se estrena.
//
// Una medida sobre un corpus de mentira dice lo que el autor quiso que dijera.
// Esta se hace contra los paquetes instalados de verdad, y su numero es el que
// contesta la pregunta que importa: DE TODO LO QUE LAS NORMAS PIDEN, A CUANTO
// LLEGA HOY UN BUSCADOR LEXICO SOBRE UNA POLITICA CORRIENTE.
//
// No hay techo ni suelo afirmados sobre ese numero A PROPOSITO. Un umbral aqui
// seria una promesa sobre el contenido de un documento de mentira, y la
// tentacion evidente el dia que baje seria aflojar el minimo de aciertos hasta
// que suba, que es maquillar la medida en vez de mejorar la pieza. Lo que si se
// afirma son las DOS propiedades que no pueden fallar nunca:
//
//	ninguna cita sale que no este LITERAL en el documento del cliente;
//	ninguna cita sale de una fuente que no sea del cliente.
//
// La segunda es la que separa esta pantalla de una alucinacion: senalar un
// articulo del BOE como si fuera la politica del cliente seria decirle que ya
// tiene escrito lo que no tiene.

// politicaDeUnaPyme es lo que de verdad manda un cliente de 200 empleados: un
// documento corto, en prosa, sin nombrar ni una norma.
const politicaDeUnaPyme = `POLITICA DE SEGURIDAD DE LA INFORMACION

La direccion aprueba esta politica y la revisa al menos una vez al ano, y siempre que haya un cambio significativo en la organizacion o en su entorno de riesgo.

Existe un inventario de activos de informacion, mantenido por el area de sistemas, con un responsable asignado y su nivel de clasificacion.

Se realiza un analisis de riesgos anual sobre los activos del inventario, y las medidas que salen de el se recogen en un plan de tratamiento con responsable y fecha.

Las copias de seguridad se realizan diariamente, se guardan cifradas fuera de la sede y se restauran de prueba una vez al trimestre.

El control de acceso sigue el principio de minimo privilegio. Los permisos se revisan cada seis meses y se retiran el mismo dia de la baja de la persona.

Se registran los accesos a los sistemas que tratan datos personales y los registros se conservan durante dos anos.

El personal recibe formacion en seguridad de la informacion al incorporarse y con caracter anual, y firma un compromiso de confidencialidad.

Los incidentes de seguridad se notifican al responsable de seguridad en cuanto se detectan. El responsable decide si procede notificar a la autoridad competente y en que plazo.

Los proveedores que tratan informacion de la organizacion firman un contrato con clausulas de seguridad y se evaluan antes de contratarlos.

El cifrado de los soportes extraibles y de los portatiles es obligatorio.

Existe un plan de continuidad de negocio que se prueba una vez al ano.`

func TestQueParteDeLoQuePidenLasNormasEncuentraHoyEnUnaPoliticaCorriente(t *testing.T) {
	ps, err := corpus.Cargar("../../paquetes")
	if err != nil {
		t.Fatal(err)
	}

	datos := []byte(politicaDeUnaPyme)
	doc, err := ingesta.Leer("politica.txt", datos)
	if err != nil {
		t.Fatal(err)
	}
	fs, err := ia.FuentesAportadas(ia.Huella(datos), doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) < 8 {
		t.Fatalf("la politica de prueba solo da %d fragmentos: el arnes no ejerce nada", len(fs))
	}
	idx, err := busqueda.Nuevo(ia.Documentos(fs))
	if err != nil {
		t.Fatal(err)
	}
	v, err := ia.Nuevo(ia.Opciones{
		Fuentes:    fs,
		Admite:     []ia.Procedencia{ia.Aportado},
		MinimoCita: ia.MinimoCitaPorDefecto,
	})
	if err != nil {
		t.Fatal(err)
	}

	// TODAS las obligaciones del corpus instalado, sin filtrar por marco: es lo
	// que un cliente tiene delante.
	var cs []evidencia.Consulta
	citables := 0
	for _, p := range ps {
		for _, o := range p.Obligaciones {
			texto := o.TextoLegal
			if texto == "" {
				continue
			}
			citables++
			cs = append(cs, evidencia.Consulta{ID: o.ID, Texto: texto})
		}
	}
	if citables < 100 {
		t.Fatalf("solo %d obligaciones con texto: el corpus no se ha cargado entero", citables)
	}

	hs, err := evidencia.Mapear(idx, v, cs)
	if err != nil {
		t.Fatal(err)
	}

	// LAS DOS PROPIEDADES QUE NO PUEDEN FALLAR.
	fuentesDelCliente := map[string]bool{}
	for _, f := range fs {
		fuentesDelCliente[f.ID] = true
	}
	for _, h := range hs {
		if !strings.Contains(politicaDeUnaPyme, h.Cita) {
			t.Errorf("%s: la cita no esta literal en la politica del cliente: %q",
				h.ConsultaID, h.Cita)
		}
		if !fuentesDelCliente[h.Fuente] {
			t.Errorf("%s: la cita sale de %q, que NO es un documento del cliente. "+
				"Senalar un articulo de la norma como si fuera su politica es decirle "+
				"que ya tiene escrito lo que no tiene", h.ConsultaID, h.Fuente)
		}
	}

	// Y EL CARDINAL, derivado y no escrito. Es la respuesta honesta a «cuanto
	// de esto ahorra la tarde» y es el numero que un modelo tendria que subir.
	t.Logf("evidencia sobre el corpus real: %d obligaciones con texto, %d con un parrafo "+
		"senalado en una politica de %d fragmentos (%.1f %%). El numero se lee CON su "+
		"muestra de abajo y no solo: la precision no la mide este test, se mira. Lo que "+
		"queda fuera es parafrasis y morfologia, que es el hueco declarado del paquete",
		citables, len(hs), len(fs), 100*float64(len(hs))/float64(citables))

	for i, h := range hs {
		if i%4 != 0 {
			continue
		}
		var pedido string
		for _, c := range cs {
			if c.ID == h.ConsultaID {
				pedido = c.Texto
				break
			}
		}
		if len(pedido) > 110 {
			pedido = pedido[:110]
		}
		cita := h.Cita
		if len(cita) > 110 {
			cita = cita[:110]
		}
		t.Logf("MUESTRA %s (aciertos %d, punt %.2f) | pide: %s | cita: %s",
			h.ConsultaID, h.Aciertos, h.Puntuacion, pedido, cita)
	}

	// LA BANDA, EN LAS DOS DIRECCIONES, y es lo unico que se afirma del numero.
	//
	// Una cifra cuyo fallo probable es FAVORECERTE necesita puerta en los dos
	// sentidos, y esta ya se equivoco una vez a favor: la primera version daba
	// 53 % y casi todo era falso. El camino barato para «mejorar» esta pieza es
	// aflojar el minimo de aciertos hasta que el numero suba, y ese camino tiene
	// que estar cerrado desde el dia uno.
	//
	// Se afirma la FRACCION y no el contaje porque el corpus crece: con un
	// contaje, esta puerta se pondria roja cada vez que entre un paquete, que es
	// la frecuencia que entrena a esquivar una puerta.
	//
	// El techo dice «alguien aflojo el umbral». El suelo dice «algo se rompio»,
	// o que un paquete nuevo trajo obligaciones que este emparejamiento no
	// alcanza, y las dos cosas hay que mirarlas. Ninguno de los dos bordes es
	// una promesa de calidad: la calidad se mira en la muestra de arriba.
	const (
		suelo = 0.05
		techo = 0.25
	)
	frac := float64(len(hs)) / float64(citables)
	if frac < suelo || frac > techo {
		t.Errorf("el mapeo cubre el %.1f %% de las obligaciones (%d de %d) y la banda "+
			"declarada es %.0f-%.0f %%. "+
			"Si ha SUBIDO: mira la muestra antes de celebrarlo. El camino barato para "+
			"subir este numero es aflojar evidencia.MinimoAciertos, y la primera version "+
			"de esta pieza daba 53 %% con casi todo falso. "+
			"Si ha BAJADO: o algo se rompio, o ha entrado corpus que este emparejamiento "+
			"lexico no alcanza, y eso ultimo es informacion y no un fallo.",
			100*frac, len(hs), citables, 100*suelo, 100*techo)
	}
}
